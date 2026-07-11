package worker

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/contract/cases"
	"github.com/doveccl/doj/contract/judger"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/storage"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) taskPackage(c echo.Context) error {
	taskID, err := parseTaskID(c)
	if err != nil {
		return err
	}
	attempt, err := strconv.Atoi(strings.TrimSpace(c.QueryParam("attempt")))
	if err != nil || attempt <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid task attempt")
	}
	expectedHash := strings.TrimSpace(c.QueryParam("hash"))
	if len(expectedHash) != sha256.Size*2 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid package hash")
	}
	if _, err := hex.DecodeString(expectedHash); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid package hash")
	}

	var submission models.Submission
	if err := api.db.First(&submission, taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "task not found")
		}
		return err
	}
	if err := api.requireTaskOwner(c, submission); err != nil {
		return err
	}
	if submission.Status != "judging" || submission.Attempt != attempt || submission.LeaseUntil == nil || !submission.LeaseUntil.After(time.Now()) {
		return echo.NewHTTPError(http.StatusConflict, "stale task lease")
	}
	store, err := storage.NewFromEnv()
	if err != nil {
		return err
	}
	files, err := listProblemObjects(c.Request().Context(), store, submission.ProblemID)
	if err != nil {
		return err
	}
	currentHash := problemPackageHash(submission.ProblemID, files)
	if currentHash != expectedHash {
		return echo.NewHTTPError(http.StatusConflict, "problem package changed")
	}
	etag := `"` + currentHash + `"`
	c.Response().Header().Set(echo.HeaderCacheControl, "private, no-store")
	c.Response().Header().Set("ETag", etag)
	if c.Request().Header.Get("If-None-Match") == etag {
		return c.NoContent(http.StatusNotModified)
	}
	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="P%d.zip"`, submission.ProblemID))
	c.Response().WriteHeader(http.StatusOK)
	writer := zip.NewWriter(c.Response().Writer)
	defer writer.Close()
	return writeProblemPackageZip(c.Request().Context(), writer, store, submission.ProblemID, files)
}

func buildPayload(ctx context.Context, tx *gorm.DB, submission models.Submission) (*judger.TaskPayload, error) {
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
	files, err := listProblemObjects(ctx, store, problem.ID)
	if err != nil {
		return nil, err
	}
	cases := casePayloadsFromObjects(problem.ID, files)
	return &judger.TaskPayload{
		ID:           submission.ID,
		SubmissionID: submission.ID,
		Attempt:      submission.Attempt,
		Source:       submission.Code,
		Lang: judger.LangPayload{
			ID:      lang.ID,
			Source:  lang.Source,
			Image:   lang.Image,
			Compile: lang.Compile,
			Run:     lang.Run,
		},
		Mode: problem.Mode,
		Limits: judger.LimitsPayload{
			TimeMS:   problem.TimeMS,
			MemoryKB: problem.MemoryMB * 1024,
			OutputKB: 65536,
			Pids:     64,
			FileKB:   65536,
		},
		Cases: cases,
		Problem: judger.ProblemPayload{
			ID:          problem.ID,
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

func casePayloadsFromObjects(problemID uint, objects []storage.Info) []judger.CasePayload {
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
	cases := make([]judger.CasePayload, 0, len(pairs))
	for index, item := range pairs {
		score := base
		if index == len(pairs)-1 {
			score += remainder
		}
		cases = append(cases, judger.CasePayload{ID: item.id, Input: item.input, Answer: item.answer, Score: score})
	}
	return cases
}

func problemAssetPrefix(id uint, section string) string {
	return fmt.Sprintf("problems/%d/%s", id, section)
}
