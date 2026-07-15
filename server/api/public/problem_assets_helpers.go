package public

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/doveccl/doj/server/cache"
	problemdata "github.com/doveccl/doj/server/problem"
	"github.com/doveccl/doj/server/storage"
	"io"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/contract/cases"
	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

func (api *API) syncProblemPackage(c echo.Context, id uint) (contract.ProblemPackage, error) {
	pkg, err := api.problemPackageFromDB(c.Request().Context(), id)
	if err != nil {
		return contract.ProblemPackage{}, err
	}
	api.cacheProblemPackage(c.Request().Context(), id, pkg)
	return pkg, nil
}

func (api *API) problemPackageFromDB(ctx context.Context, id uint) (contract.ProblemPackage, error) {
	var row models.Problem
	if err := api.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return contract.ProblemPackage{}, err
	}
	item, err := problemdata.Parse(row.Package)
	if err != nil {
		return contract.ProblemPackage{}, err
	}
	result := contract.ProblemPackage{Version: problemdata.ETag(packageJSON(row.Package))}
	for _, file := range item.Files {
		view := contract.PackageFile{Key: file.Path, Name: strings.TrimPrefix(strings.TrimPrefix(file.Path, "data/"), "judge/"), Size: file.Size}
		switch {
		case strings.HasPrefix(file.Path, "data/"):
			result.Data = append(result.Data, view)
			result.DataBytes += file.Size
		case strings.HasPrefix(file.Path, "judge/"):
			result.Judge = append(result.Judge, view)
		}
	}
	for _, item := range item.Cases {
		result.CaseList = append(result.CaseList, contract.PackageCase{ID: item.ID, Input: item.Input, Answer: item.Answer, Score: item.Score})
	}
	sort.Slice(result.Data, func(i, j int) bool { return cases.DataCaseFileLess(result.Data[i].Name, result.Data[j].Name) })
	result.Cases = len(result.CaseList)
	return result, nil
}

func packageJSON(raw []byte) []byte {
	if len(raw) == 0 || string(raw) == "null" {
		return []byte("{}")
	}
	return raw
}

func (api *API) problemPackageCached(ctx context.Context, id uint) (contract.ProblemPackage, error) {
	var cached contract.ProblemPackage
	found, err := cache.Get(ctx, problemPackageCacheKey(id), &cached)
	if err == nil && found {
		return cached, nil
	}
	pkg, err := api.problemPackageFromDB(ctx, id)
	if err != nil {
		return contract.ProblemPackage{}, err
	}
	api.cacheProblemPackage(ctx, id, pkg)
	return pkg, nil
}

func (api *API) cacheProblemPackage(ctx context.Context, id uint, pkg contract.ProblemPackage) {
	_ = cache.Set(ctx, problemPackageCacheKey(id), pkg, time.Minute)
}

func problemPackageCacheKey(id uint) string {
	return "doj:problem:" + strconv.FormatUint(uint64(id), 10) + ":package"
}

func writeProblemStatementZipFile(writer *zip.Writer, statement string) error {
	file, err := writer.CreateHeader(&zip.FileHeader{Name: "statement.md", Method: zip.Deflate})
	if err != nil {
		return err
	}
	_, err = io.WriteString(file, statement)
	return err
}

func writeAssetZipFiles(ctx context.Context, writer *zip.Writer, store storage.Store, section string, files []contract.PackageFile) error {
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
	clean, err := storage.CleanKey(normalized)
	if err != nil || clean != normalized {
		return "", false
	}
	return path.Join(section, clean), true
}

func assetFiles(ctx context.Context, store storage.Store, prefix string) ([]contract.PackageFile, error) {
	objects, err := store.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	fullPrefix := strings.TrimSuffix(prefix, "/") + "/"
	items := make([]contract.PackageFile, 0, len(objects))
	for _, object := range objects {
		if !strings.HasPrefix(object.Key, fullPrefix) {
			continue
		}
		name := strings.TrimPrefix(object.Key, fullPrefix)
		if name == "" {
			continue
		}
		items = append(items, contract.PackageFile{Key: object.Key, Name: name, Size: object.Size})
	}
	sort.Slice(items, func(i, j int) bool { return cases.DataCaseFileLess(items[i].Name, items[j].Name) })
	return items, nil
}

func problemAssetPrefix(id uint) string {
	return fmt.Sprintf("problems/%d/assets", id)
}

func packageSection(raw string) (string, error) {
	switch strings.TrimSpace(raw) {
	case "data", "judge":
		return strings.TrimSpace(raw), nil
	default:
		return "", echo.NewHTTPError(http.StatusBadRequest, "package section must be data or judge")
	}
}

func cleanPackageFileName(raw string) (string, error) {
	name := path.Base(strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/"))
	if name == "" || name == "." || name == ".." {
		return "", echo.NewHTTPError(http.StatusBadRequest, "package file name is required")
	}
	if _, err := storage.CleanKey(name); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid package file name")
	}
	return name, nil
}
