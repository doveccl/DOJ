package public

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	problemdata "github.com/doveccl/doj/server/problem"
	"github.com/doveccl/doj/server/storage"
	"net/http"
	"path"
	"strconv"
	"strings"

	contract "github.com/doveccl/doj/contract/web"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) problemPackage(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	pkg, err := api.syncProblemPackage(c, id)
	if err != nil {
		return err
	}
	c.Response().Header().Set("ETag", `"`+pkg.Version+`"`)
	return c.JSON(http.StatusOK, pkg)
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
	return api.problemPackageFile(c, "data")
}

func (api *API) problemPrivateJudge(c echo.Context) error {
	return api.problemPackageFile(c, "judge")
}

func (api *API) problemPackageFile(c echo.Context, section string) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	rel, err := storage.CleanKey(c.Param("*"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "package file not found")
	}
	return api.streamProblemPackageFile(c, id, path.Join(section, rel))
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
	return streamMedia(c, key, "media not found", true)
}

func (api *API) uploadProblemPackage(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	section, err := packageSection(c.FormValue("section"))
	if err != nil {
		return err
	}
	return api.uploadProblemPackageFiles(c, id, section)
}

func (api *API) deleteProblemPackage(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	key, err := storage.CleanKey(c.QueryParam("key"))
	if err != nil || key != "data" && key != "judge" && !strings.HasPrefix(key, "data/") && !strings.HasPrefix(key, "judge/") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid package key")
	}
	return api.deleteProblemPackageFiles(c, id, key)
}

func (api *API) downloadProblemArchive(c echo.Context) error {
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
	statement := problemStatement(problem)
	assets, err := assetFiles(c.Request().Context(), store, problemAssetPrefix(id))
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("P%d.zip", id)
	c.Response().Header().Set(echo.HeaderCacheControl, "private, no-store")
	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	c.Response().WriteHeader(http.StatusOK)
	writer := zip.NewWriter(c.Response().Writer)
	defer writer.Close()
	if err := writeProblemStatementZipFile(writer, statement); err != nil {
		return err
	}
	item, err := problemdata.Parse(problem.Package)
	if err != nil {
		return err
	}
	if err := writeProblemPackageArchive(c.Request().Context(), writer, store, id, item); err != nil {
		return err
	}
	if err := writeAssetZipFiles(c.Request().Context(), writer, store, "assets", assets); err != nil {
		return err
	}
	return nil
}

func (api *API) decorateProblemPackageStats(ctx context.Context, items []contract.Problem) error {
	if len(items) == 0 {
		return nil
	}
	for index := range items {
		id := items[index].ID
		pkg, err := api.problemPackageCached(ctx, id)
		if err != nil {
			return err
		}
		cases := pkg.Cases
		dataBytes := pkg.DataBytes
		items[index].Cases = &cases
		items[index].DataBytes = &dataBytes
	}
	return nil
}
