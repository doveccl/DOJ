package web

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/doveccl/doj/common/cache"
	"github.com/doveccl/doj/common/storage"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/common/cases"
	contract "github.com/doveccl/doj/common/web"
	"github.com/labstack/echo/v4"
)

func (api *API) syncProblemAssets(c echo.Context, id uint) (contract.ProblemAssets, error) {
	store, err := storage.NewFromEnv()
	if err != nil {
		return contract.ProblemAssets{}, err
	}
	assets, err := problemAssetsFromStore(c.Request().Context(), id, store)
	if err != nil {
		return contract.ProblemAssets{}, err
	}
	api.cacheProblemAssets(c.Request().Context(), id, assets)
	return assets, nil
}

func (api *API) problemAssetsCached(ctx context.Context, id uint, store storage.Store) (contract.ProblemAssets, error) {
	var cached contract.ProblemAssets
	found, err := cache.Get(ctx, problemAssetsCacheKey(id), &cached)
	if err == nil && found {
		return cached, nil
	}
	assets, err := problemAssetsFromStore(ctx, id, store)
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

func clearProblemPackageCacheIfNeeded(ctx context.Context, id uint, key string) {
	data := problemAssetPrefix(id, "data") + "/"
	judge := problemAssetPrefix(id, "judge") + "/"
	if strings.HasPrefix(key, data) || strings.HasPrefix(key, judge) {
		clearProblemPackageCache(ctx, id)
	}
}

func clearProblemPackageCache(ctx context.Context, id uint) {
	_ = cache.Delete(ctx, cache.ProblemPackageKey(id))
}

func cleanEditableAssetKey(id uint, raw string) (string, error) {
	key, err := storage.CleanKey(raw)
	if err != nil || !problemAssetKeyAllowed(id, key) {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid asset key")
	}
	if !editableAssetName(key) {
		return "", echo.NewHTTPError(http.StatusBadRequest, "asset is not editable")
	}
	return key, nil
}

func problemAssetsFromStore(ctx context.Context, id uint, store storage.Store) (contract.ProblemAssets, error) {
	data, err := assetFiles(ctx, store, problemAssetPrefix(id, "data"))
	if err != nil {
		return contract.ProblemAssets{}, err
	}
	judge, err := assetFiles(ctx, store, problemAssetPrefix(id, "judge"))
	if err != nil {
		return contract.ProblemAssets{}, err
	}
	assets, err := assetFiles(ctx, store, problemAssetPrefix(id, "assets"))
	if err != nil {
		return contract.ProblemAssets{}, err
	}
	cases, dataBytes := dataStats(data)
	return contract.ProblemAssets{Data: data, Judge: judge, Assets: assets, Cases: cases, DataBytes: dataBytes}, nil
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
		items = append(items, contract.AssetFile{
			Key:      object.Key,
			Name:     name,
			Size:     object.Size,
			Editable: editableAsset(name, object.Size),
		})
	}
	sort.Slice(items, func(i, j int) bool { return cases.DataCaseFileLess(items[i].Name, items[j].Name) })
	return items, nil
}

func dataStats(files []contract.AssetFile) (int, int64) {
	inputs := map[string]bool{}
	outputs := map[string]bool{}
	var bytes int64
	for _, file := range files {
		bytes += file.Size
		stem, kind := cases.DataCaseStem(file.Name)
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
	if _, err := storage.CleanKey(name); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid asset file name")
	}
	return name, nil
}

func caseName(raw string, assets contract.ProblemAssets) (string, error) {
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

func nextCaseName(assets contract.ProblemAssets) string {
	used := map[string]bool{}
	for _, file := range assets.Data {
		stem, kind := cases.DataCaseStem(file.Name)
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
