package web

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) problems(c echo.Context) error {
	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	problems, total, err := api.searchProblemPage(c, c.QueryParam("q"), c.QueryParam("tag"), pageSize, offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, PageResult[ProblemDTO]{Items: problems, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) tags(c echo.Context) error {
	items, err := api.searchTags(c, c.QueryParam("kind"), c.QueryParam("q"), 50)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, items)
}

func (api *API) problemState(c echo.Context) error {
	ids, err := parseIDs(c.QueryParam("ids"), "invalid problem ids")
	if err != nil {
		return err
	}
	var assignmentID *uint
	if raw := c.QueryParam("assignment"); raw != "" {
		id, err := parseQueryID(raw, "invalid assignment id")
		if err != nil {
			return err
		}
		assignmentID = &id
	}
	var contestID *uint
	if raw := c.QueryParam("contest"); raw != "" {
		id, err := parseQueryID(raw, "invalid contest id")
		if err != nil {
			return err
		}
		contestID = &id
	}
	if assignmentID != nil && contestID != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "assignment and contest cannot both be set")
	}
	items, err := api.problemStateItems(c, ids, assignmentID, contestID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, items)
}

func (api *API) problem(c echo.Context) error {
	if strings.HasSuffix(c.Param("id"), ".zip") {
		return api.downloadProblemAssets(c)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid problem id")
	}

	var problem models.Problem
	if err := api.db.First(&problem, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	if !api.isAdmin(c) {
		visible, err := api.problemVisibleInDetail(c, problem)
		if err != nil {
			return err
		}
		if !visible {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
	}
	item, err := api.problemDTOWithStatement(c.Request().Context(), problem)
	if err != nil {
		return err
	}
	if !api.isAdmin(c) && api.problemInUnfinishedContest(problem.ID) {
		item.Tags = []string{}
	}
	items := []ProblemDTO{item}
	if err := api.decorateProblemAssetStats(c.Request().Context(), items); err != nil {
		return err
	}
	item = items[0]
	return c.JSON(http.StatusOK, item)
}

func (api *API) createProblem(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	var req ProblemCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Mode = strings.TrimSpace(req.Mode)
	req.Tags = normalizeTags(req.Tags)
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	if req.Mode == "" {
		req.Mode = "default"
	}
	if !validProblemMode(req.Mode) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid judge mode")
	}
	if req.TimeMS <= 0 {
		req.TimeMS = 1000
	}
	if req.MemoryMB <= 0 {
		req.MemoryMB = 256
	}

	tags, _ := json.Marshal(req.Tags)
	visible := true
	if req.Visible != nil {
		visible = *req.Visible
	}

	row := models.Problem{
		Title:    req.Title,
		Tags:     tags,
		Visible:  visible,
		Mode:     req.Mode,
		TimeMS:   req.TimeMS,
		MemoryMB: req.MemoryMB,
	}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, CreatedID{ID: row.ID})
}

func (api *API) updateProblem(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}
	var req ProblemUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Title == nil && req.Statement == nil && req.Tags == nil && req.Visible == nil && req.Mode == nil && req.TimeMS == nil && req.MemoryMB == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "no fields to update")
	}

	var row models.Problem
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "title is required")
		}
		if err := validateTitle(title); err != nil {
			return err
		}
		row.Title = title
	}
	if req.Tags != nil {
		tags, _ := json.Marshal(normalizeTags(*req.Tags))
		row.Tags = tags
	}
	if req.Visible != nil {
		row.Visible = *req.Visible
	}
	if req.Mode != nil {
		mode := strings.TrimSpace(*req.Mode)
		if mode == "" {
			mode = "default"
		}
		if !validProblemMode(mode) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid judge mode")
		}
		row.Mode = mode
	}
	if req.TimeMS != nil {
		timeMS := *req.TimeMS
		if timeMS <= 0 {
			timeMS = 1000
		}
		row.TimeMS = timeMS
	}
	if req.MemoryMB != nil {
		memoryMB := *req.MemoryMB
		if memoryMB <= 0 {
			memoryMB = 256
		}
		row.MemoryMB = memoryMB
	}
	var statement *string
	if req.Statement != nil {
		value := strings.TrimSpace(*req.Statement)
		if value == "" && row.Title != "" {
			value = "# " + row.Title
		}
		if err := validateTextBytes(value, utils.MaxMarkdownBytes, "statement is too large"); err != nil {
			return err
		}
		statement = &value
	}
	if err := api.db.Save(&row).Error; err != nil {
		return err
	}
	if statement != nil {
		if err := api.writeProblemStatement(c.Request().Context(), row.ID, *statement); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, CreatedID{ID: row.ID})
}

func (api *API) updateProblemVisibility(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}
	var req ProblemVisibilityUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}

	var row models.Problem
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	row.Visible = req.Visible
	if err := api.db.Save(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, problemDTO(row))
}

func (api *API) deleteProblem(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}

	if err := api.db.Delete(&models.Problem{}, id).Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) problemAssets(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	assets, err := api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, assets)
}

func (api *API) uploadProblemImage(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "image file is required")
	}
	data, mime, sha, ext, err := readUploadedImage(file)
	if err != nil {
		return err
	}
	rel := sha + ext
	key := path.Join("problems", strconv.Itoa(int(id)), "assets", rel)
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	if err := store.Put(c.Request().Context(), key, bytes.NewReader(data), int64(len(data)), mime); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, UploadResult{URL: fmt.Sprintf("/api/problems/%d/assets/%s", id, rel)})
}

func (api *API) problemPrivateData(c echo.Context) error {
	return api.problemPrivateAsset(c, "data")
}

func (api *API) problemPrivateJudge(c echo.Context) error {
	return api.problemPrivateAsset(c, "judge")
}

func (api *API) problemPrivateAsset(c echo.Context, section string) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	rel, err := utils.CleanObjectKey(c.Param("*"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "asset not found")
	}
	key := path.Join("problems", strconv.Itoa(int(id)), section, rel)
	return streamMedia(c, key, "asset not found")
}

func (api *API) problemPublicAsset(c echo.Context) error {
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}
	if err := api.requireProblemVisible(c, id); err != nil {
		return err
	}
	rel, err := utils.CleanObjectKey(c.Param("*"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "media not found")
	}
	key := path.Join("problems", strconv.Itoa(int(id)), "assets", rel)
	return streamMedia(c, key, "media not found")
}

func (api *API) uploadProblemAsset(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	section, err := assetSection(c.FormValue("section"))
	if err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "asset file is required")
	}
	name, err := cleanAssetName(file.Filename)
	if err != nil {
		return err
	}
	if file.Size > maxAssetBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "asset file is too large")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	buffer := make([]byte, 512)
	n, readErr := src.Read(buffer)
	if readErr != nil && readErr != io.EOF {
		return readErr
	}
	contentType := http.DetectContentType(buffer[:n])
	reader := io.MultiReader(bytes.NewReader(buffer[:n]), src)
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	key := path.Join(problemAssetPrefix(id, section), name)
	if err := store.Put(c.Request().Context(), key, reader, file.Size, contentType); err != nil {
		return err
	}
	clearProblemPackageCacheIfNeeded(c.Request().Context(), id, key)
	assets, err := api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, assets)
}

func (api *API) deleteProblemAsset(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	key, err := utils.CleanObjectKey(c.QueryParam("key"))
	if err != nil || !problemAssetKeyAllowed(id, key) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid asset key")
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	if err := store.Delete(c.Request().Context(), key); err != nil {
		return err
	}
	clearProblemPackageCacheIfNeeded(c.Request().Context(), id, key)
	assets, err := api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, assets)
}

func (api *API) problemAssetContent(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	key, err := cleanEditableAssetKey(id, c.QueryParam("key"))
	if err != nil {
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	reader, _, err := store.Open(c.Request().Context(), key)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "asset not found")
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxEditableAssetBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxEditableAssetBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "asset is too large to edit")
	}
	return c.JSON(http.StatusOK, AssetContent{Key: key, Name: path.Base(key), Content: string(data)})
}

func (api *API) updateProblemAssetContent(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	var req AssetContentUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	key, err := cleanEditableAssetKey(id, req.Key)
	if err != nil {
		return err
	}
	if len(req.Content) > maxEditableAssetBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "asset is too large to edit")
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	if err := store.Put(c.Request().Context(), key, strings.NewReader(req.Content), int64(len(req.Content)), "text/plain; charset=utf-8"); err != nil {
		return err
	}
	clearProblemPackageCacheIfNeeded(c.Request().Context(), id, key)
	assets, err := api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, assets)
}

func (api *API) createProblemCase(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	var req AssetCaseCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	assets, err := problemAssetsFromStore(c.Request().Context(), id, store)
	if err != nil {
		return err
	}
	name, err := caseName(req.Name, assets)
	if err != nil {
		return err
	}
	inputKey := path.Join(problemAssetPrefix(id, "data"), name+".in")
	outputKey := path.Join(problemAssetPrefix(id, "data"), name+".out")
	if err := store.Put(c.Request().Context(), inputKey, strings.NewReader(req.Input), int64(len(req.Input)), "text/plain; charset=utf-8"); err != nil {
		return err
	}
	if err := store.Put(c.Request().Context(), outputKey, strings.NewReader(req.Output), int64(len(req.Output)), "text/plain; charset=utf-8"); err != nil {
		return err
	}
	clearProblemPackageCache(c.Request().Context(), id)
	assets, err = api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, assets)
}

func (api *API) fillJudgeTemplate(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	for name, body := range judgeTemplateFiles() {
		key := path.Join(problemAssetPrefix(id, "judge"), name)
		if err := store.Put(c.Request().Context(), key, strings.NewReader(body), int64(len(body)), "text/plain; charset=utf-8"); err != nil {
			return err
		}
	}
	clearProblemPackageCache(c.Request().Context(), id)
	assets, err := api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, assets)
}

func (api *API) downloadProblemAssets(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	rawID := strings.TrimSuffix(c.Param("id"), ".zip")
	parsed, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid problem id")
	}
	id := uint(parsed)
	var problem models.Problem
	if err := api.db.First(&problem, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	statement, err := api.problemStatement(c.Request().Context(), id, problem.Title)
	if err != nil {
		return err
	}
	assets, err := problemAssetsFromStore(c.Request().Context(), id, store)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("P%d.zip", id)
	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	c.Response().WriteHeader(http.StatusOK)
	writer := zip.NewWriter(c.Response().Writer)
	defer writer.Close()
	if err := writeProblemStatementZipFile(writer, statement); err != nil {
		return err
	}
	if err := writeAssetZipFiles(c.Request().Context(), writer, store, "data", assets.Data); err != nil {
		return err
	}
	if err := writeAssetZipFiles(c.Request().Context(), writer, store, "judge", assets.Judge); err != nil {
		return err
	}
	if err := writeAssetZipFiles(c.Request().Context(), writer, store, "assets", assets.Assets); err != nil {
		return err
	}
	return nil
}

func (api *API) listProblems(c echo.Context, limit int) ([]ProblemDTO, error) {
	return api.findProblems(c, "", "", limit, "id desc")
}

func (api *API) searchProblems(c echo.Context, q string, tag string, limit int) ([]ProblemDTO, error) {
	return api.findProblems(c, q, tag, limit, "id asc")
}

func (api *API) searchProblemPage(c echo.Context, q string, tag string, limit int, offset int) ([]ProblemDTO, int64, error) {
	var rows []models.Problem
	query := api.db.Model(&models.Problem{})
	if !api.isAdmin(c) {
		query = api.applyProblemListVisibility(query)
	}
	query = applyProblemSearch(query, q)
	if tag != "" {
		rawTag, _ := json.Marshal([]string{tag})
		query = query.Where("tags @> ?::jsonb", string(rawTag))
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Session(&gorm.Session{}).Order("id asc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]ProblemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, problemDTO(row))
	}
	return items, total, nil
}

func (api *API) findProblems(c echo.Context, q string, tag string, limit int, order string) ([]ProblemDTO, error) {
	var rows []models.Problem
	query := api.db.Order(order).Limit(limit)
	if !api.isAdmin(c) {
		query = api.applyProblemListVisibility(query)
	}
	query = applyProblemSearch(query, q)
	if tag != "" {
		rawTag, _ := json.Marshal([]string{tag})
		query = query.Where("tags @> ?::jsonb", string(rawTag))
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ProblemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, problemDTO(row))
	}
	return items, nil
}

func applyProblemSearch(query *gorm.DB, q string) *gorm.DB {
	q = strings.TrimSpace(q)
	if q == "" {
		return query
	}
	like := "%" + q + "%"
	id := strings.TrimPrefix(strings.ToUpper(q), "P")
	if _, err := strconv.ParseUint(id, 10, 64); err == nil {
		return query.Where("CAST(id AS TEXT) LIKE ? OR LOWER(title) LIKE LOWER(?)", "%"+id+"%", like)
	}
	return query.Where("LOWER(title) LIKE LOWER(?)", like)
}

func problemDTO(row models.Problem) ProblemDTO {
	return ProblemDTO{
		ID:       row.ID,
		Title:    row.Title,
		Tags:     readTags([]byte(row.Tags)),
		Visible:  row.Visible,
		Mode:     row.Mode,
		TimeMS:   row.TimeMS,
		MemoryMB: row.MemoryMB,
	}
}

func (api *API) problemDTOWithStatement(ctx context.Context, row models.Problem) (ProblemDTO, error) {
	item := problemDTO(row)
	statement, err := api.problemStatement(ctx, row.ID, row.Title)
	if err != nil {
		return item, err
	}
	item.Statement = statement
	return item, nil
}

func (api *API) problemStatement(ctx context.Context, id uint, title string) (string, error) {
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return "", err
	}
	reader, _, err := store.Open(ctx, problemStatementKey(id))
	if err != nil {
		return "# " + title, nil
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, utils.MaxMarkdownBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > utils.MaxMarkdownBytes {
		return "", echo.NewHTTPError(http.StatusRequestEntityTooLarge, "statement is too large")
	}
	if strings.TrimSpace(string(data)) == "" {
		return "# " + title, nil
	}
	return string(data), nil
}

func (api *API) writeProblemStatement(ctx context.Context, id uint, statement string) error {
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	if err := validateTextBytes(statement, utils.MaxMarkdownBytes, "statement is too large"); err != nil {
		return err
	}
	body := strings.TrimSpace(statement)
	if body == "" {
		body = "# P" + strconv.Itoa(int(id))
	}
	return store.Put(ctx, problemStatementKey(id), strings.NewReader(body), int64(len(body)), "text/markdown; charset=utf-8")
}

func problemStatementKey(id uint) string {
	return fmt.Sprintf("problems/%d/statement.md", id)
}

func (api *API) problemDiscussionCounts(ctx context.Context) (map[uint]int, error) {
	counts := map[uint]int{}
	found, err := utils.CacheGet(ctx, problemDiscussionsCacheKey(), &counts)
	if err == nil && found {
		return counts, nil
	}
	var rows []models.Discussion
	if err := api.db.Select("id", "tags", "pinned", "locked").Order("updated_at desc").Limit(1000).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		item := DiscussionDTO{ID: row.ID, Tags: readTags([]byte(row.Tags)), Pinned: row.Pinned, Locked: row.Locked}
		for _, problemID := range discussionProblemIDs(item) {
			counts[problemID]++
		}
	}
	_ = utils.CacheSet(ctx, problemDiscussionsCacheKey(), counts, 10*time.Second)
	return counts, nil
}

func problemDiscussionsCacheKey() string {
	return "doj:problem:discussions"
}

func (api *API) problemStateItems(c echo.Context, ids []uint, assignmentID *uint, contestID *uint) ([]ProblemStateDTO, error) {
	ids = uniqueUint(ids)
	if len(ids) == 0 {
		return []ProblemStateDTO{}, nil
	}
	items := defaultProblemStateItems(ids)
	if err := api.fillProblemSummaryState(c, items); err != nil {
		return nil, err
	}
	return api.fillProblemUserState(c, items, assignmentID, contestID)
}

func (api *API) fillProblemSummaryState(c echo.Context, items []ProblemStateDTO) error {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProblemID)
	}
	submitByProblem, acByProblem, err := api.problemSubmissionStats(ids)
	if err != nil {
		return err
	}
	discussions, err := api.problemDiscussionCounts(c.Request().Context())
	if err != nil {
		return err
	}
	for index := range items {
		id := items[index].ProblemID
		if !api.isAdmin(c) {
			visible, err := api.problemVisibleForStats(c, id)
			if err != nil {
				return err
			}
			if !visible || api.problemInUnfinishedContest(id) {
				continue
			}
		}
		items[index].Submit = submitByProblem[id]
		items[index].AC = acByProblem[id]
		count := discussions[id]
		items[index].Discussions = &count
	}
	return nil
}

func (api *API) problemSubmissionStats(ids []uint) (map[uint]int, map[uint]int, error) {
	var submits []struct {
		ProblemID uint
		Count     int64
	}
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, count(*) AS count").
		Where("problem_id IN ?", ids).
		Group("problem_id").
		Find(&submits).Error; err != nil {
		return nil, nil, err
	}
	submitByProblem := map[uint]int{}
	for _, item := range submits {
		submitByProblem[item.ProblemID] = int(item.Count)
	}
	var acs []struct {
		ProblemID uint
		Count     int64
	}
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, count(DISTINCT user_id) AS count").
		Where("problem_id IN ? AND status = ?", ids, "AC").
		Group("problem_id").
		Find(&acs).Error; err != nil {
		return nil, nil, err
	}
	acByProblem := map[uint]int{}
	for _, item := range acs {
		acByProblem[item.ProblemID] = int(item.Count)
	}
	return submitByProblem, acByProblem, nil
}

func (api *API) decorateProblemAssetStats(ctx context.Context, items []ProblemDTO) error {
	if len(items) == 0 {
		return nil
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	for index := range items {
		id := items[index].ID
		assets, err := api.problemAssetsCached(ctx, id, store)
		if err != nil {
			return err
		}
		cases := assets.Cases
		dataBytes := assets.DataBytes
		items[index].Cases = &cases
		items[index].DataBytes = &dataBytes
	}
	return nil
}

func (api *API) fillProblemUserState(c echo.Context, items []ProblemStateDTO, assignmentID *uint, contestID *uint) ([]ProblemStateDTO, error) {
	if contestID != nil {
		var contest models.Contest
		if err := api.db.First(&contest, *contestID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, echo.NewHTTPError(http.StatusNotFound, "contest not found")
			}
			return nil, err
		}
		return api.fillProblemUserStateInContest(c, items, contest)
	}
	if api.role(c) == "guest" {
		return items, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ProblemID    uint
		ID           uint
		AssignmentID *uint
		ContestID    *uint
		Status       string
		Score        int
		CreatedAt    time.Time
	}
	ids := problemStateIDs(items)
	query := api.db.Model(&models.Submission{}).
		Select("problem_id, id, assignment_id, contest_id, status, score, created_at").
		Where("user_id = ? AND problem_id IN ?", user.ID, ids).
		Order("created_at desc")
	if assignmentID != nil {
		query = query.Where("assignment_id = ?", *assignmentID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	status := map[uint]string{}
	submission := map[uint]RecordDTO{}
	for _, row := range rows {
		resultVisible := true
		if row.ID != 0 {
			view, err := api.submissionView(c, models.Submission{ID: row.ID, UserID: user.ID, ProblemID: row.ProblemID, AssignmentID: row.AssignmentID, ContestID: row.ContestID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt})
			if err != nil {
				return nil, err
			}
			resultVisible = view.Result
		}
		submissionSet := false
		if _, ok := submission[row.ProblemID]; !ok {
			submission[row.ProblemID] = RecordDTO{ID: row.ID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
			submissionSet = true
		}
		if !resultVisible {
			if submissionSet {
				submission[row.ProblemID] = pendingRecord(submission[row.ProblemID])
			}
			if status[row.ProblemID] == "" {
				status[row.ProblemID] = "pending"
			}
			continue
		}
		if row.Status == "AC" {
			status[row.ProblemID] = "ac"
			continue
		}
		if status[row.ProblemID] == "" {
			status[row.ProblemID] = "tried"
		}
	}
	for index := range items {
		items[index].Status = status[items[index].ProblemID]
		if items[index].Status == "" {
			items[index].Status = "none"
		}
		if item, ok := submission[items[index].ProblemID]; ok && assignmentID == nil {
			items[index].Submission = &item
		}
	}
	return items, nil
}

func (api *API) fillProblemUserStateInContest(c echo.Context, items []ProblemStateDTO, contest models.Contest) ([]ProblemStateDTO, error) {
	if api.role(c) == "guest" {
		return items, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ProblemID uint
		ID        uint
		ContestID uint
		Status    string
		Score     int
		CreatedAt time.Time
	}
	ids := problemStateIDs(items)
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, id, contest_id, status, score, created_at").
		Where("user_id = ? AND contest_id = ? AND problem_id IN ?", user.ID, contest.ID, ids).
		Order("created_at desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	status := map[uint]string{}
	submission := map[uint]RecordDTO{}
	for _, row := range rows {
		resultVisible := true
		if row.ID != 0 {
			view, err := api.submissionView(c, models.Submission{ID: row.ID, UserID: user.ID, ProblemID: row.ProblemID, ContestID: &row.ContestID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt})
			if err != nil {
				return nil, err
			}
			resultVisible = view.Result
		}
		submissionSet := false
		if _, ok := submission[row.ProblemID]; !ok {
			submission[row.ProblemID] = RecordDTO{ID: row.ID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
			submissionSet = true
		}
		if !resultVisible {
			if submissionSet {
				submission[row.ProblemID] = pendingRecord(submission[row.ProblemID])
			}
			if status[row.ProblemID] == "" {
				status[row.ProblemID] = "pending"
			}
			continue
		}
		if contest.Kind == "OI" {
			if status[row.ProblemID] == "" {
				if row.Score >= 100 {
					status[row.ProblemID] = "ac"
				} else {
					status[row.ProblemID] = "tried"
				}
			}
			continue
		}
		if status[row.ProblemID] == "ac" {
			continue
		}
		if row.Status == "AC" {
			status[row.ProblemID] = "ac"
			submission[row.ProblemID] = RecordDTO{ID: row.ID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
			continue
		}
		if status[row.ProblemID] == "" {
			status[row.ProblemID] = "tried"
		}
	}
	for index := range items {
		items[index].Status = status[items[index].ProblemID]
		if items[index].Status == "" {
			items[index].Status = "none"
		}
		if item, ok := submission[items[index].ProblemID]; ok {
			items[index].Submission = &item
		}
	}
	return items, nil
}

func defaultProblemStateItems(ids []uint) []ProblemStateDTO {
	items := make([]ProblemStateDTO, 0, len(ids))
	for _, id := range ids {
		items = append(items, ProblemStateDTO{ProblemID: id, Status: "none"})
	}
	return items
}

func problemStateIDs(items []ProblemStateDTO) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProblemID)
	}
	return ids
}

func (api *API) requireProblemAdmin(c echo.Context) (uint, error) {
	if err := api.requireAdmin(c); err != nil {
		return 0, err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return 0, err
	}

	var count int64
	if err := api.db.Model(&models.Problem{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, echo.NewHTTPError(http.StatusNotFound, "problem not found")
	}
	return id, nil
}

func (api *API) requireProblemVisible(c echo.Context, id uint) error {
	if api.isAdmin(c) {
		return nil
	}

	var problem models.Problem
	if err := api.db.First(&problem, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	visible, err := api.problemVisibleInDetail(c, problem)
	if err != nil {
		return err
	}
	if !visible {
		return echo.NewHTTPError(http.StatusNotFound, "problem not found")
	}
	return nil
}

func (api *API) problemVisibleInDetail(c echo.Context, problem models.Problem) (bool, error) {
	if api.problemVisibleInList(problem) {
		return true, nil
	}
	if api.problemInRunningContest(problem.ID) {
		return true, nil
	}
	if api.role(c) == "guest" {
		return false, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return false, err
	}
	return api.problemInActiveAssignmentForUser(problem.ID, user.ID)
}

func (api *API) problemInActiveAssignmentForUser(problemID uint, userID uint) (bool, error) {
	var rows []models.Assignment
	if err := api.db.
		Joins("JOIN assignment_problems ON assignment_problems.assignment_id = assignments.id").
		Where("assignment_problems.problem_id = ? AND assignments.end_at >= ?", problemID, time.Now()).
		Find(&rows).Error; err != nil {
		return false, err
	}
	for _, row := range rows {
		ok, err := api.userAssignedTo(row.ID, userID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (api *API) problemVisibleInList(problem models.Problem) bool {
	if !problem.Visible {
		return false
	}
	return !api.problemInUnfinishedContest(problem.ID)
}

func (api *API) applyProblemListVisibility(query *gorm.DB) *gorm.DB {
	now := time.Now()
	return query.Where(
		`problems.visible = ? AND NOT EXISTS (
			SELECT 1 FROM contest_problems
			JOIN contests ON contests.id = contest_problems.contest_id
			WHERE contest_problems.problem_id = problems.id AND contests.end_at > ?
		)`,
		true,
		now,
	)
}

func (api *API) problemInUnfinishedContest(problemID uint) bool {
	var count int64
	err := api.db.Model(&models.ContestProblem{}).
		Joins("JOIN contests ON contests.id = contest_problems.contest_id").
		Where("contest_problems.problem_id = ? AND contests.end_at > ?", problemID, time.Now()).
		Count(&count).Error
	return err == nil && count > 0
}

func (api *API) problemVisibleForStats(c echo.Context, id uint) (bool, error) {
	var problem models.Problem
	if err := api.db.First(&problem, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return api.problemVisibleInDetail(c, problem)
}

func (api *API) problemInRunningContest(problemID uint) bool {
	var count int64
	now := time.Now()
	err := api.db.Model(&models.ContestProblem{}).
		Joins("JOIN contests ON contests.id = contest_problems.contest_id").
		Where("contest_problems.problem_id = ? AND contests.start_at <= ? AND contests.end_at > ?", problemID, now, now).
		Count(&count).Error
	return err == nil && count > 0
}

func (api *API) syncProblemAssets(c echo.Context, id uint) (ProblemAssets, error) {
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return ProblemAssets{}, err
	}
	assets, err := problemAssetsFromStore(c.Request().Context(), id, store)
	if err != nil {
		return ProblemAssets{}, err
	}
	api.cacheProblemAssets(c.Request().Context(), id, assets)
	return assets, nil
}

func (api *API) problemAssetsCached(ctx context.Context, id uint, store utils.ObjectStore) (ProblemAssets, error) {
	var cached ProblemAssets
	found, err := utils.CacheGet(ctx, problemAssetsCacheKey(id), &cached)
	if err == nil && found {
		return cached, nil
	}
	assets, err := problemAssetsFromStore(ctx, id, store)
	if err != nil {
		return ProblemAssets{}, err
	}
	api.cacheProblemAssets(ctx, id, assets)
	return assets, nil
}

func (api *API) cacheProblemAssets(ctx context.Context, id uint, assets ProblemAssets) {
	_ = utils.CacheSet(ctx, problemAssetsCacheKey(id), assets, time.Minute)
}

func problemAssetsCacheKey(id uint) string {
	return "doj:problem:" + strconv.FormatUint(uint64(id), 10) + ":assets"
}

func clearProblemPackageCacheIfNeeded(ctx context.Context, id uint, key string) {
	data := problemAssetPrefix(id, "data") + "/"
	judge := problemAssetPrefix(id, "judge") + "/"
	if strings.HasPrefix(key, data) || strings.HasPrefix(key, judge) {
		clearProblemPackageCache(ctx, id)
	}
}

func clearProblemPackageCache(ctx context.Context, id uint) {
	_ = utils.CacheDelete(ctx, utils.ProblemPackageCacheKey(id))
}

func cleanEditableAssetKey(id uint, raw string) (string, error) {
	key, err := utils.CleanObjectKey(raw)
	if err != nil || !problemAssetKeyAllowed(id, key) {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid asset key")
	}
	if !editableAssetName(key) {
		return "", echo.NewHTTPError(http.StatusBadRequest, "asset is not editable")
	}
	return key, nil
}

func problemAssetsFromStore(ctx context.Context, id uint, store utils.ObjectStore) (ProblemAssets, error) {
	data, err := assetFiles(ctx, store, problemAssetPrefix(id, "data"))
	if err != nil {
		return ProblemAssets{}, err
	}
	judge, err := assetFiles(ctx, store, problemAssetPrefix(id, "judge"))
	if err != nil {
		return ProblemAssets{}, err
	}
	assets, err := assetFiles(ctx, store, problemAssetPrefix(id, "assets"))
	if err != nil {
		return ProblemAssets{}, err
	}
	cases, dataBytes := dataStats(data)
	return ProblemAssets{Data: data, Judge: judge, Assets: assets, Cases: cases, DataBytes: dataBytes}, nil
}

func writeProblemStatementZipFile(writer *zip.Writer, statement string) error {
	file, err := writer.CreateHeader(&zip.FileHeader{Name: "statement.md", Method: zip.Deflate})
	if err != nil {
		return err
	}
	_, err = io.WriteString(file, statement)
	return err
}

func writeAssetZipFiles(ctx context.Context, writer *zip.Writer, store utils.ObjectStore, section string, files []AssetFile) error {
	for _, item := range files {
		zipName, ok := safeAssetZipName(section, item.Name)
		if !ok {
			continue
		}
		reader, _, err := store.Open(ctx, item.Key)
		if err != nil {
			return err
		}
		header := &zip.FileHeader{Name: zipName, Method: zip.Deflate}
		file, err := writer.CreateHeader(header)
		if err != nil {
			_ = reader.Close()
			return err
		}
		if _, err := io.Copy(file, reader); err != nil {
			_ = reader.Close()
			return err
		}
		if err := reader.Close(); err != nil {
			return err
		}
	}
	return nil
}

func safeAssetZipName(section string, name string) (string, bool) {
	normalized := strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	clean, err := utils.CleanObjectKey(normalized)
	if err != nil || clean != normalized {
		return "", false
	}
	return path.Join(section, clean), true
}

func assetFiles(ctx context.Context, store utils.ObjectStore, prefix string) ([]AssetFile, error) {
	objects, err := store.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	fullPrefix := strings.TrimSuffix(prefix, "/") + "/"
	items := make([]AssetFile, 0, len(objects))
	for _, object := range objects {
		if !strings.HasPrefix(object.Key, fullPrefix) {
			continue
		}
		name := strings.TrimPrefix(object.Key, fullPrefix)
		if name == "" {
			continue
		}
		items = append(items, AssetFile{
			Key:      object.Key,
			Name:     name,
			Size:     object.Size,
			Editable: editableAsset(name, object.Size),
		})
	}
	sort.Slice(items, func(i, j int) bool { return utils.DataCaseFileLess(items[i].Name, items[j].Name) })
	return items, nil
}

func dataStats(files []AssetFile) (int, int64) {
	inputs := map[string]bool{}
	outputs := map[string]bool{}
	var bytes int64
	for _, file := range files {
		bytes += file.Size
		stem, kind := utils.DataCaseStem(file.Name)
		switch kind {
		case "in":
			inputs[stem] = true
		case "out":
			outputs[stem] = true
		}
	}
	cases := 0
	for stem := range inputs {
		if outputs[stem] {
			cases++
		}
	}
	return cases, bytes
}

func editableAsset(name string, size int64) bool {
	if size > maxEditableAssetBytes {
		return false
	}
	return editableAssetName(name)
}

func editableAssetName(name string) bool {
	switch strings.ToLower(path.Base(name)) {
	case "dockerfile", "makefile":
		return true
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".c", ".cc", ".cpp", ".cxx", ".go", ".rs", ".py", ".java", ".js", ".ts", ".txt", ".md", ".json", ".yaml", ".yml", ".toml", ".in", ".out":
		return true
	default:
		return false
	}
}

func problemAssetPrefix(id uint, section string) string {
	return fmt.Sprintf("problems/%d/%s", id, section)
}

func problemAssetKeyAllowed(id uint, key string) bool {
	data := problemAssetPrefix(id, "data") + "/"
	judge := problemAssetPrefix(id, "judge") + "/"
	assets := problemAssetPrefix(id, "assets") + "/"
	return strings.HasPrefix(key, data) || strings.HasPrefix(key, judge) || strings.HasPrefix(key, assets)
}

func assetSection(raw string) (string, error) {
	switch strings.TrimSpace(raw) {
	case "data", "judge", "assets":
		return strings.TrimSpace(raw), nil
	default:
		return "", echo.NewHTTPError(http.StatusBadRequest, "asset section must be data, judge or assets")
	}
}

func cleanAssetName(raw string) (string, error) {
	name := path.Base(strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/"))
	if name == "" || name == "." || name == ".." {
		return "", echo.NewHTTPError(http.StatusBadRequest, "asset file name is required")
	}
	if _, err := utils.CleanObjectKey(name); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid asset file name")
	}
	return name, nil
}

func caseName(raw string, assets ProblemAssets) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		name = nextCaseName(assets)
	}
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".in"), ".out")
	var out []rune
	for _, char := range name {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_' {
			out = append(out, char)
		}
	}
	if len(out) == 0 {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid case name")
	}
	return string(out), nil
}

func nextCaseName(assets ProblemAssets) string {
	used := map[string]bool{}
	for _, file := range assets.Data {
		stem, kind := utils.DataCaseStem(file.Name)
		if kind != "" {
			used[stem] = true
		}
	}
	for i := 1; ; i++ {
		name := strconv.Itoa(i)
		if !used[name] {
			return name
		}
	}
}

func judgeTemplateFiles() map[string]string {
	return map[string]string{
		"Dockerfile": `FROM gcc
WORKDIR /src
COPY main.cc .
RUN g++ main.cc -o main
CMD ["/src/main"]
`,
		"main.cc": `#include <bits/stdc++.h>
using namespace std;

string read_all(istream& in) { return string(istreambuf_iterator<char>(in), {}); }
string read_file(const char* p) { ifstream f(p, ios::binary); return read_all(f); }
void trim_right(string& s) { while (!s.empty() && isspace((unsigned char)s.back())) s.pop_back(); }

int main(int argc, char** argv) {
  // argv: input, transcript, answer, result
  // return: 0 = AC, 1 = WA, 2 = PE, 3 = checker/interactor error
  if (argc != 5) return 3;

  // Feed input while reading output; doing one whole side first can deadlock on full pipes.
  thread feeder([&] {
    cout << ifstream(argv[1], ios::binary).rdbuf() << flush;
    fclose(stdout);
  });
  string got = read_all(cin);
  feeder.join();

  string ans = read_file(argv[3]);
  trim_right(got);
  trim_right(ans);

  if (got != ans) {
    ofstream(argv[4]) << "expected output differs";
    return 1; // WA
  }

  return 0; // AC
}
`,
	}
}

func validProblemMode(mode string) bool {
	return mode == "default" || mode == "strict" || mode == "custom"
}

func problemSort(index int) string {
	if index >= 0 && index < 26 {
		return string(rune('A' + index))
	}
	return strconv.Itoa(index + 1)
}

func normalizeProblemRefs(items []ProblemRef) []ProblemRef {
	for index := range items {
		items[index].Sort = strings.TrimSpace(items[index].Sort)
		if items[index].Sort == "" {
			items[index].Sort = problemSort(index)
		}
	}
	return items
}

func (api *API) validateProblemRefs(items []ProblemRef) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(items))
	seen := make(map[uint]bool, len(items))
	for _, item := range items {
		if item.ID == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid problem id")
		}
		if seen[item.ID] {
			return echo.NewHTTPError(http.StatusBadRequest, "duplicate problem id")
		}
		if len([]rune(item.Sort)) > models.SortMax {
			return echo.NewHTTPError(http.StatusBadRequest, "problem sort is too long")
		}
		seen[item.ID] = true
		ids = append(ids, item.ID)
	}
	var count int64
	if err := api.db.Model(&models.Problem{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return echo.NewHTTPError(http.StatusBadRequest, "problem not found")
	}
	return nil
}
