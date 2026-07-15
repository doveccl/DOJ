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

func (api *API) syncProblemAssets(c echo.Context, id uint) (contract.ProblemAssets, error) {
	store, err := storage.NewFromEnv()
	if err != nil {
		return contract.ProblemAssets{}, err
	}
	assets, err := api.problemAssetsFromPackage(c.Request().Context(), id, store)
	if err != nil {
		return contract.ProblemAssets{}, err
	}
	api.cacheProblemAssets(c.Request().Context(), id, assets)
	return assets, nil
}

func (api *API) problemAssetsFromPackage(ctx context.Context, id uint, store storage.Store) (contract.ProblemAssets, error) {
	var row models.Problem
	if err := api.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return contract.ProblemAssets{}, err
	}
	item, err := problemdata.Parse(row.Package)
	if err != nil {
		return contract.ProblemAssets{}, err
	}
	result := contract.ProblemAssets{Version: problemdata.ETag(packageJSON(row.Package))}
	for _, file := range item.Files {
		asset := contract.AssetFile{Key: file.Path, Name: strings.TrimPrefix(strings.TrimPrefix(file.Path, "data/"), "judge/"), Size: file.Size}
		switch {
		case strings.HasPrefix(file.Path, "data/"):
			result.Data = append(result.Data, asset)
			result.DataBytes += file.Size
		case strings.HasPrefix(file.Path, "judge/"):
			result.Judge = append(result.Judge, asset)
		}
	}
	for _, item := range item.Cases {
		result.CaseList = append(result.CaseList, contract.AssetCase{ID: item.ID, Input: item.Input, Answer: item.Answer, Score: item.Score})
	}
	sort.Slice(result.Data, func(i, j int) bool { return cases.DataCaseFileLess(result.Data[i].Name, result.Data[j].Name) })
	result.Cases = len(result.CaseList)
	result.Assets, err = assetFiles(ctx, store, problemAssetPrefix(id, "assets"))
	return result, err
}

func packageJSON(raw []byte) []byte {
	if len(raw) == 0 || string(raw) == "null" {
		return []byte("{}")
	}
	return raw
}

func (api *API) problemAssetsCached(ctx context.Context, id uint, store storage.Store) (contract.ProblemAssets, error) {
	var cached contract.ProblemAssets
	found, err := cache.Get(ctx, problemAssetsCacheKey(id), &cached)
	if err == nil && found {
		return cached, nil
	}
	assets, err := api.problemAssetsFromPackage(ctx, id, store)
	if err != nil {
		return contract.ProblemAssets{}, err
	}
	api.cacheProblemAssets(ctx, id, assets)
	return assets, nil
}

func (api *API) cacheProblemAssets(ctx context.Context, id uint, assets contract.ProblemAssets) {
	_ = cache.Set(ctx, problemAssetsCacheKey(id), assets, time.Minute)
}

func problemAssetsCacheKey(id uint) string {
	return "doj:problem:" + strconv.FormatUint(uint64(id), 10) + ":assets"
}

func writeProblemStatementZipFile(writer *zip.Writer, statement string) error {
	file, err := writer.CreateHeader(&zip.FileHeader{Name: "statement.md", Method: zip.Deflate})
	if err != nil {
		return err
	}
	_, err = io.WriteString(file, statement)
	return err
}

func writeAssetZipFiles(ctx context.Context, writer *zip.Writer, store storage.Store, section string, files []contract.AssetFile) error {
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

func assetFiles(ctx context.Context, store storage.Store, prefix string) ([]contract.AssetFile, error) {
	objects, err := store.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	fullPrefix := strings.TrimSuffix(prefix, "/") + "/"
	items := make([]contract.AssetFile, 0, len(objects))
	for _, object := range objects {
		if !strings.HasPrefix(object.Key, fullPrefix) {
			continue
		}
		name := strings.TrimPrefix(object.Key, fullPrefix)
		if name == "" {
			continue
		}
		items = append(items, contract.AssetFile{Key: object.Key, Name: name, Size: object.Size})
	}
	sort.Slice(items, func(i, j int) bool { return cases.DataCaseFileLess(items[i].Name, items[j].Name) })
	return items, nil
}

func problemAssetPrefix(id uint, section string) string {
	return fmt.Sprintf("problems/%d/%s", id, section)
}

func problemAssetKeyAllowed(id uint, key string) bool {
	assets := problemAssetPrefix(id, "assets") + "/"
	return strings.HasPrefix(key, assets)
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
	if _, err := storage.CleanKey(name); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid asset file name")
	}
	return name, nil
}
