package public

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	casecontract "github.com/doveccl/doj/contract/cases"
	"github.com/doveccl/doj/models"
	problemdata "github.com/doveccl/doj/server/problem"
	"github.com/doveccl/doj/server/storage"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (api *API) streamProblemPackageFile(c echo.Context, id uint, name string) error {
	var row models.Problem
	if err := api.db.First(&row, id).Error; err != nil {
		return err
	}
	item, err := problemdata.Parse(row.Package)
	if err != nil {
		return err
	}
	file, ok := problemdata.FindFile(item, name)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "asset not found")
	}
	store, err := storage.NewFromEnv()
	if err != nil {
		return err
	}
	reader, err := problemdata.OpenFile(c.Request().Context(), store, problemdata.ObjectKey(id, item.Hash), file)
	if err != nil {
		if storage.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "asset not found")
		}
		return err
	}
	defer reader.Close()
	prefix := make([]byte, min(int64(512), file.Size))
	if _, err := io.ReadFull(reader, prefix); err != nil {
		return err
	}
	c.Response().Header().Set(echo.HeaderCacheControl, "private, no-store")
	c.Response().Header().Set(echo.HeaderContentLength, fmt.Sprint(file.Size))
	return c.Stream(http.StatusOK, http.DetectContentType(prefix), io.MultiReader(bytes.NewReader(prefix), reader))
}

func (api *API) uploadProblemPackageFiles(c echo.Context, id uint, section string) error {
	var row models.Problem
	if err := api.db.First(&row, id).Error; err != nil {
		return err
	}
	baseJSON := packageJSON(row.Package)
	if err := requirePackageMatch(c, baseJSON); err != nil {
		return err
	}
	current, err := problemdata.Parse(baseJSON)
	if err != nil {
		return err
	}
	form, err := c.MultipartForm()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "asset files are required")
	}
	files := append([]*multipart.FileHeader(nil), form.File["files"]...)
	files = append(files, form.File["file"]...)
	if len(files) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "asset files are required")
	}
	work, err := os.MkdirTemp("", "doj-package-work-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	store, err := storage.NewFromEnv()
	if err != nil {
		return err
	}
	if current.Hash != "" {
		if err := problemdata.Extract(c.Request().Context(), store, problemdata.ObjectKey(id, current.Hash), current, work); err != nil {
			return err
		}
	}
	for _, zipFirst := range []bool{true, false} {
		for _, file := range files {
			isZip := strings.EqualFold(filepath.Ext(file.Filename), ".zip")
			if isZip != zipFirst {
				continue
			}
			limit := int64(problemdata.MaxFileBytes)
			if isZip {
				limit = problemdata.MaxPackageBytes
			}
			if file.Size > limit {
				return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "asset file is too large")
			}
			if isZip {
				if err := importProblemZip(file, section, work); err != nil {
					return err
				}
				continue
			}
			name, err := cleanPackageFileName(file.Filename)
			if err != nil {
				return err
			}
			if err := copyUpload(file, filepath.Join(work, section, filepath.FromSlash(name))); err != nil {
				return err
			}
		}
	}
	packagePath := filepath.Join(work, ".package.zip")
	next, err := problemdata.Build(work, packagePath, current.Cases)
	if err != nil {
		return err
	}
	if next.Hash != current.Hash {
		packageFile, err := os.Open(packagePath)
		if err != nil {
			return err
		}
		putErr := store.Put(c.Request().Context(), problemdata.ObjectKey(id, next.Hash), packageFile, next.Size, "application/zip")
		_ = packageFile.Close()
		if putErr != nil {
			return putErr
		}
	}
	nextJSON, err := next.JSON()
	if err != nil {
		return err
	}
	if err := api.compareAndSwapPackage(c, id, baseJSON, nextJSON); err != nil {
		return err
	}
	assets, err := api.syncProblemPackage(c, id)
	if err != nil {
		return err
	}
	c.Response().Header().Set("ETag", `"`+assets.Version+`"`)
	return c.JSON(http.StatusCreated, assets)
}

func (api *API) deleteProblemPackageFiles(c echo.Context, id uint, name string) error {
	var row models.Problem
	if err := api.db.First(&row, id).Error; err != nil {
		return err
	}
	baseJSON := packageJSON(row.Package)
	if err := requirePackageMatch(c, baseJSON); err != nil {
		return err
	}
	item, err := problemdata.Parse(baseJSON)
	if err != nil {
		return err
	}
	files := item.Files[:0]
	found := false
	section := name == "data" || name == "judge"
	for _, file := range item.Files {
		if file.Path == name || section && strings.HasPrefix(file.Path, name+"/") {
			found = true
			continue
		}
		files = append(files, file)
	}
	if !found && !section {
		return echo.NewHTTPError(http.StatusNotFound, "asset not found")
	}
	item.Files = files
	item.Cases = packageCases(item.Files, item.Cases)
	nextJSON, err := item.JSON()
	if err != nil {
		return err
	}
	if err := api.compareAndSwapPackage(c, id, baseJSON, nextJSON); err != nil {
		return err
	}
	assets, err := api.syncProblemPackage(c, id)
	if err != nil {
		return err
	}
	c.Response().Header().Set("ETag", `"`+assets.Version+`"`)
	return c.JSON(http.StatusOK, assets)
}

func (api *API) updateProblemCaseScore(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	caseID := strings.TrimSpace(c.QueryParam("case"))
	if caseID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "case is required")
	}
	var req struct {
		Score *int `json:"score"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Score != nil && *req.Score < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "case score is invalid")
	}
	var row models.Problem
	if err := api.db.First(&row, id).Error; err != nil {
		return err
	}
	baseJSON := packageJSON(row.Package)
	if err := requirePackageMatch(c, baseJSON); err != nil {
		return err
	}
	item, err := problemdata.Parse(baseJSON)
	if err != nil {
		return err
	}
	found := false
	for index := range item.Cases {
		if item.Cases[index].ID == caseID {
			item.Cases[index].Score = req.Score
			found = true
			break
		}
	}
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "case not found")
	}
	nextJSON, err := item.JSON()
	if err != nil {
		return err
	}
	if err := api.compareAndSwapPackage(c, id, baseJSON, nextJSON); err != nil {
		return err
	}
	assets, err := api.syncProblemPackage(c, id)
	if err != nil {
		return err
	}
	c.Response().Header().Set("ETag", `"`+assets.Version+`"`)
	return c.JSON(http.StatusOK, assets)
}

func packageCases(files []problemdata.File, old []problemdata.Case) []problemdata.Case {
	inputs := map[string]bool{}
	answers := map[string]bool{}
	for _, file := range files {
		name := strings.TrimPrefix(file.Path, "data/")
		stem, kind := casecontract.DataCaseStem(name)
		switch kind {
		case "in":
			inputs[stem] = true
		case "out":
			answers[stem] = true
		}
	}
	result := old[:0]
	for _, item := range old {
		if inputs[item.ID] && answers[item.ID] {
			result = append(result, item)
		}
	}
	return result
}

func requirePackageMatch(c echo.Context, raw []byte) error {
	want := `"` + problemdata.ETag(raw) + `"`
	if c.Request().Header.Get("If-Match") != want {
		return echo.NewHTTPError(http.StatusPreconditionFailed, "problem package changed; refresh and retry")
	}
	return nil
}

func (api *API) compareAndSwapPackage(c echo.Context, id uint, oldJSON []byte, nextJSON []byte) error {
	return api.db.WithContext(c.Request().Context()).Transaction(func(tx *gorm.DB) error {
		var current models.Problem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, id).Error; err != nil {
			return err
		}
		if !bytes.Equal(packageJSON(current.Package), oldJSON) {
			return echo.NewHTTPError(http.StatusPreconditionFailed, "problem package changed; refresh and retry")
		}
		return tx.Model(&models.Problem{}).Where("id = ?", id).Update("package", datatypes.JSON(nextJSON)).Error
	})
}

func copyUpload(file *multipart.FileHeader, destination string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	dst, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(dst, io.LimitReader(src, problemdata.MaxFileBytes+1))
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil || written > problemdata.MaxFileBytes {
		return fmt.Errorf("asset file is too large")
	}
	return nil
}

func importProblemZip(file *multipart.FileHeader, section string, work string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	tmp, err := os.CreateTemp(work, ".upload-*.zip")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	written, copyErr := io.Copy(tmp, io.LimitReader(src, problemdata.MaxPackageBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil || written > problemdata.MaxPackageBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "ZIP is too large")
	}
	reader, err := zip.OpenReader(tmpName)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ZIP")
	}
	defer reader.Close()
	if len(reader.File) > problemdata.MaxPackageEntries {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "ZIP has too many files")
	}
	rooted := false
	for _, entry := range reader.File {
		name := strings.TrimPrefix(strings.ReplaceAll(entry.Name, "\\", "/"), "./")
		if strings.HasPrefix(name, "data/") || strings.HasPrefix(name, "judge/") {
			rooted = true
			break
		}
	}
	var expanded int64
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		name := strings.TrimPrefix(strings.ReplaceAll(entry.Name, "\\", "/"), "./")
		clean, cleanErr := storage.CleanKey(name)
		if cleanErr != nil || clean != name {
			return echo.NewHTTPError(http.StatusBadRequest, "ZIP contains an invalid path")
		}
		if rooted && !strings.HasPrefix(clean, "data/") && !strings.HasPrefix(clean, "judge/") {
			continue
		}
		if !rooted {
			clean = path.Join(section, clean)
		}
		expanded += int64(entry.UncompressedSize64)
		if entry.UncompressedSize64 > problemdata.MaxFileBytes || expanded > problemdata.MaxExpandedBytes {
			return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "ZIP expands beyond limit")
		}
		target := filepath.Join(work, filepath.FromSlash(clean))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		in, err := entry.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		_ = in.Close()
		closeErr := out.Close()
		if copyErr != nil || closeErr != nil {
			return fmt.Errorf("extract %s", clean)
		}
	}
	return nil
}

func writeProblemPackageArchive(ctx context.Context, writer *zip.Writer, store storage.Store, id uint, item problemdata.Package) error {
	if item.Hash == "" {
		return nil
	}
	if item.Size <= 0 || item.Size > problemdata.MaxPackageBytes {
		return fmt.Errorf("invalid problem package size")
	}
	reader, _, err := store.Open(ctx, problemdata.ObjectKey(id, item.Hash))
	if err != nil {
		return err
	}
	defer reader.Close()
	files := append([]problemdata.File(nil), item.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Offset < files[j].Offset })
	position := int64(0)
	for _, file := range files {
		clean, cleanErr := storage.CleanKey(file.Path)
		if cleanErr != nil || clean != file.Path || file.Size < 0 || file.CompressedSize < 0 || file.Offset < position || file.Offset > item.Size || file.CompressedSize > item.Size-file.Offset {
			return fmt.Errorf("invalid package file %q", file.Path)
		}
		if _, err := io.CopyN(io.Discard, reader, file.Offset-position); err != nil {
			return fmt.Errorf("seek package file %q: %w", file.Path, err)
		}
		header := &zip.FileHeader{
			Name:               file.Path,
			Method:             zip.Deflate,
			Flags:              0x800,
			CreatorVersion:     20,
			ReaderVersion:      20,
			CRC32:              file.CRC32,
			CompressedSize64:   uint64(file.CompressedSize),
			UncompressedSize64: uint64(file.Size),
		}
		header.SetMode(0o600)
		destination, err := writer.CreateRaw(header)
		if err != nil {
			return err
		}
		if _, err := io.CopyN(destination, reader, file.CompressedSize); err != nil {
			return fmt.Errorf("copy package file %q: %w", file.Path, err)
		}
		position = file.Offset + file.CompressedSize
	}
	return nil
}
