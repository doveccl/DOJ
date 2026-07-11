package public

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/doveccl/doj/contract/limits"
	contract "github.com/doveccl/doj/contract/web"

	"github.com/doveccl/doj/models"
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
	return c.JSON(http.StatusOK, contract.Page[contract.Problem]{Items: problems, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) tags(c echo.Context) error {
	items, err := api.searchTags(c, c.QueryParam("kind"), c.QueryParam("q"), 50)
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
	item, err := api.problemViewWithStatement(c.Request().Context(), problem)
	if err != nil {
		return err
	}
	if !api.isAdmin(c) && api.problemInUnfinishedContest(problem.ID) {
		item.Tags = []string{}
	}
	items := []contract.Problem{item}
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
	var req contract.ProblemCreate
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
	timeMS, err := problemTimeLimit(req.TimeMS)
	if err != nil {
		return err
	}
	memoryMB, err := problemMemoryLimit(req.MemoryMB)
	if err != nil {
		return err
	}
	req.TimeMS = timeMS
	req.MemoryMB = memoryMB

	tags, _ := json.Marshal(req.Tags)
	row := models.Problem{
		Title:    req.Title,
		Tags:     tags,
		Visible:  false,
		Mode:     req.Mode,
		TimeMS:   req.TimeMS,
		MemoryMB: req.MemoryMB,
	}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, contract.CreatedID{ID: row.ID})
}

func (api *API) updateProblem(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}
	var req contract.ProblemUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Title == nil && req.Statement == nil && req.Tags == nil && req.Mode == nil && req.TimeMS == nil && req.MemoryMB == nil {
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
		timeMS, err := problemTimeLimit(*req.TimeMS)
		if err != nil {
			return err
		}
		row.TimeMS = timeMS
	}
	if req.MemoryMB != nil {
		memoryMB, err := problemMemoryLimit(*req.MemoryMB)
		if err != nil {
			return err
		}
		row.MemoryMB = memoryMB
	}
	var statement *string
	if req.Statement != nil {
		value := strings.TrimSpace(*req.Statement)
		if value == "" && row.Title != "" {
			value = "# " + row.Title
		}
		if err := validateTextBytes(value, limits.MaxMarkdownBytes, "statement is too large"); err != nil {
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
	return c.JSON(http.StatusOK, contract.CreatedID{ID: row.ID})
}

func problemTimeLimit(value int) (int, error) {
	if value <= 0 {
		return 1000, nil
	}
	if value > limits.MaxProblemTimeMS {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "time limit is too large")
	}
	return value, nil
}

func problemMemoryLimit(value int) (int, error) {
	if value <= 0 {
		return 256, nil
	}
	if value > limits.MaxProblemMemoryMB {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "memory limit is too large")
	}
	return value, nil
}

func (api *API) updateProblemVisibility(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}
	var req contract.ProblemVisibilityUpdate
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
	return c.JSON(http.StatusOK, problemView(row))
}

func (api *API) deleteProblem(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}

	var row models.Problem
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	if err := api.db.Delete(&row).Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
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

func validProblemMode(mode string) bool {
	return mode == "default" || mode == "strict" || mode == "custom"
}

func problemSort(index int) string {
	if index >= 0 && index < 26 {
		return string(rune('A' + index))
	}
	return strconv.Itoa(index + 1)
}

func normalizeProblemRefs(items []contract.ProblemRef) []contract.ProblemRef {
	for index := range items {
		items[index].Sort = strings.TrimSpace(items[index].Sort)
		if items[index].Sort == "" {
			items[index].Sort = problemSort(index)
		}
	}
	return items
}

func (api *API) validateProblemRefs(items []contract.ProblemRef) error {
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
