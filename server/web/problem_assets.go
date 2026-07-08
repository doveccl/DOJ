package web

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/doveccl/doj/common/storage"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	contract "github.com/doveccl/doj/common/web"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

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
	store, err := storage.NewFromEnv()
	if err != nil {
		return err
	}
	if err := store.Put(c.Request().Context(), key, bytes.NewReader(data), int64(len(data)), mime); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, contract.UploadResult{URL: fmt.Sprintf("/api/problems/%d/assets/%s", id, rel)})
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
	rel, err := storage.CleanKey(c.Param("*"))
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
	rel, err := storage.CleanKey(c.Param("*"))
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
	store, err := storage.NewFromEnv()
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
	key, err := storage.CleanKey(c.QueryParam("key"))
	if err != nil || !problemAssetKeyAllowed(id, key) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid asset key")
	}
	store, err := storage.NewFromEnv()
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
	store, err := storage.NewFromEnv()
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
	return c.JSON(http.StatusOK, contract.AssetContent{Key: key, Name: path.Base(key), Content: string(data)})
}

func (api *API) updateProblemAssetContent(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	var req contract.AssetContentUpdate
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
	store, err := storage.NewFromEnv()
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
	var req contract.AssetCaseCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	store, err := storage.NewFromEnv()
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
	store, err := storage.NewFromEnv()
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
	store, err := storage.NewFromEnv()
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

func (api *API) decorateProblemAssetStats(ctx context.Context, items []contract.Problem) error {
	if len(items) == 0 {
		return nil
	}
	store, err := storage.NewFromEnv()
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
