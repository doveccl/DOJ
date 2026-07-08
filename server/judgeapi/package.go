package judgeapi

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/doveccl/doj/common/cache"
	"github.com/doveccl/doj/common/storage"
	"io"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/common/cases"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (api *API) problemPackage(c echo.Context) error {
	problemID, err := parseProblemCode(c.Param("problem"))
	if err != nil {
		return err
	}
	var problem models.Problem
	if err := api.db.First(&problem, problemID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	store, err := storage.NewFromEnv()
	if err != nil {
		return err
	}
	files, err := listProblemObjectsCached(c.Request().Context(), store, problem.ID)
	if err != nil {
		return err
	}
	c.Response().Header().Set(echo.HeaderCacheControl, "private, max-age=0")
	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="P%d.zip"`, problem.ID))
	c.Response().WriteHeader(http.StatusOK)
	writer := zip.NewWriter(c.Response().Writer)
	defer writer.Close()
	return writeProblemPackageZip(c.Request().Context(), writer, store, problem.ID, files)
}

func buildPayload(ctx context.Context, tx *gorm.DB, submission models.Submission) (*TaskPayload, error) {
	var lang models.Language
	if err := tx.First(&lang, "id = ?", submission.Language).Error; err != nil {
		return nil, err
	}
	var problem models.Problem
	if err := tx.First(&problem, submission.ProblemID).Error; err != nil {
		return nil, err
	}
	store, err := storage.NewFromEnv()
	if err != nil {
		return nil, err
	}
	files, err := listProblemObjectsCached(ctx, store, problem.ID)
	if err != nil {
		return nil, err
	}
	cases := casePayloadsFromObjects(problem.ID, files)
	return &TaskPayload{
		ID:           submission.ID,
		SubmissionID: submission.ID,
		Attempt:      submission.Attempt,
		Source:       submission.Code,
		Lang: LangPayload{
			ID:      lang.ID,
			Source:  lang.Source,
			Image:   lang.Image,
			Compile: lang.Compile,
			Run:     lang.Run,
		},
		Mode: problem.Mode,
		Limits: LimitsPayload{
			TimeMS:   problem.TimeMS,
			MemoryKB: problem.MemoryMB * 1024,
			OutputKB: 65536,
			Pids:     64,
			FileKB:   65536,
		},
		Cases: cases,
		Problem: ProblemPayload{
			ID:          problem.ID,
			Mode:        problem.Mode,
			TimeMS:      problem.TimeMS,
			MemoryMB:    problem.MemoryMB,
			Tags:        readTags(problem.Tags),
			PackageHash: problemPackageHash(problem.ID, files),
		},
	}, nil
}

func listProblemObjects(ctx context.Context, store storage.Store, problemID uint) ([]storage.Info, error) {
	var files []storage.Info
	for _, section := range []string{"data", "judge"} {
		got, err := store.List(ctx, problemAssetPrefix(problemID, section))
		if err != nil {
			return nil, err
		}
		files = append(files, got...)
	}
	return files, nil
}

func listProblemObjectsCached(ctx context.Context, store storage.Store, problemID uint) ([]storage.Info, error) {
	var cached []storage.Info
	found, err := cache.Get(ctx, cache.ProblemPackageKey(problemID), &cached)
	if err == nil && found {
		return cached, nil
	}
	files, err := listProblemObjects(ctx, store, problemID)
	if err != nil {
		return nil, err
	}
	_ = cache.Set(ctx, cache.ProblemPackageKey(problemID), files, time.Minute)
	return files, nil
}

func problemPackageHash(problemID uint, files []storage.Info) string {
	ordered := append([]storage.Info(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Key < ordered[j].Key })
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "P%d\n", problemID)
	for _, object := range ordered {
		if _, _, ok := problemPackageZipName(problemID, object.Key); !ok {
			continue
		}
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00%s\x00%d\n", object.Key, object.Size, object.ETag, object.UpdatedAt.UnixNano())
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func writeProblemPackageZip(ctx context.Context, writer *zip.Writer, store storage.Store, problemID uint, files []storage.Info) error {
	for _, object := range files {
		section, name, ok := problemPackageZipName(problemID, object.Key)
		if !ok {
			continue
		}
		zipName, ok := safeProblemPackageZipName(section, name)
		if !ok {
			continue
		}
		reader, _, err := store.Open(ctx, object.Key)
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

func parseProblemCode(raw string) (uint, error) {
	code := strings.TrimPrefix(strings.TrimSpace(raw), "P")
	if code == raw {
		code = strings.TrimPrefix(strings.TrimSpace(raw), "p")
	}
	value, err := strconv.ParseUint(strings.TrimSuffix(code, ".zip"), 10, 64)
	if err != nil || value == 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid problem code")
	}
	return uint(value), nil
}

func safeProblemPackageZipName(section string, name string) (string, bool) {
	normalized := strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	clean, err := storage.CleanKey(normalized)
	if err != nil || clean != normalized {
		return "", false
	}
	return path.Join(section, clean), true
}

func problemPackageZipName(problemID uint, key string) (string, string, bool) {
	for _, section := range []string{"data", "judge"} {
		prefix := problemAssetPrefix(problemID, section) + "/"
		if strings.HasPrefix(key, prefix) {
			name := strings.TrimPrefix(key, prefix)
			return section, name, name != ""
		}
	}
	return "", "", false
}

func casePayloadsFromObjects(problemID uint, objects []storage.Info) []CasePayload {
	type pair struct {
		id     string
		input  string
		answer string
	}
	inputs := map[string]string{}
	answers := map[string]string{}
	prefix := problemAssetPrefix(problemID, "data") + "/"
	for _, object := range objects {
		if !strings.HasPrefix(object.Key, prefix) {
			continue
		}
		name := strings.TrimPrefix(object.Key, prefix)
		stem, kind := cases.DataCaseStem(name)
		if stem == "" || kind == "" {
			continue
		}
		relative := path.Join("data", name)
		if kind == "in" {
			inputs[stem] = relative
		} else {
			answers[stem] = relative
		}
	}
	var pairs []pair
	for stem, input := range inputs {
		if answer, ok := answers[stem]; ok {
			pairs = append(pairs, pair{id: stem, input: input, answer: answer})
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return cases.CaseStemLess(pairs[i].id, pairs[j].id) })
	if len(pairs) == 0 {
		return nil
	}
	base := 100 / len(pairs)
	remainder := 100 % len(pairs)
	cases := make([]CasePayload, 0, len(pairs))
	for index, item := range pairs {
		score := base
		if index == len(pairs)-1 {
			score += remainder
		}
		cases = append(cases, CasePayload{ID: item.id, Input: item.input, Answer: item.answer, Score: score})
	}
	return cases
}

func problemAssetPrefix(id uint, section string) string {
	return fmt.Sprintf("problems/%d/%s", id, section)
}

func readTags(raw datatypes.JSON) []string {
	var tags []string
	_ = json.Unmarshal(raw, &tags)
	return tags
}
