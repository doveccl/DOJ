package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/contract/judger"
	contractlimits "github.com/doveccl/doj/contract/limits"
	"github.com/doveccl/doj/models"
	problemdata "github.com/doveccl/doj/server/problem"
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
	etag := `"` + expectedHash + `"`
	c.Response().Header().Set(echo.HeaderCacheControl, "private, no-store")
	c.Response().Header().Set("ETag", etag)
	if c.Request().Header.Get("If-None-Match") == etag {
		return c.NoContent(http.StatusNotModified)
	}
	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="P%d.zip"`, submission.ProblemID))
	store, err := storage.NewFromEnv()
	if err != nil {
		return err
	}
	reader, _, err := store.Open(c.Request().Context(), problemdata.ObjectKey(submission.ProblemID, expectedHash))
	if err != nil {
		return err
	}
	defer reader.Close()
	return c.Stream(http.StatusOK, "application/zip", reader)
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
	item, err := problemdata.Parse(problem.Package)
	if err != nil {
		return nil, err
	}
	if item.Hash == "" || len(item.Cases) == 0 {
		return nil, fmt.Errorf("problem %d has no package or cases", problem.ID)
	}
	cases := make([]judger.CasePayload, 0, len(item.Cases))
	for _, got := range item.Cases {
		cases = append(cases, judger.CasePayload{ID: got.ID, Input: got.Input, Answer: got.Answer, Score: got.Points()})
	}
	files := make([]string, 0, len(item.Files))
	for _, got := range item.Files {
		files = append(files, got.Path)
	}
	return &judger.TaskPayload{
		ID:           submission.ID,
		SubmissionID: submission.ID,
		Attempt:      submission.Attempt,
		Source:       submission.Code,
		Lang: judger.LangPayload{
			ID:        lang.ID,
			Source:    lang.Source,
			Image:     lang.Image,
			Compile:   lang.Compile,
			CompileMS: lang.CompileMS,
			Run:       lang.Run,
		},
		Mode: problem.Mode,
		Limits: judger.LimitsPayload{
			TimeMS:   problem.TimeMS,
			MemoryKB: problem.MemoryMB * 1024,
			OutputKB: 65536,
			Pids:     64,
			FileKB:   contractlimits.DefaultJudgerFileKB,
		},
		Cases: cases,
		Problem: judger.ProblemPayload{
			ID:    problem.ID,
			Hash:  item.Hash,
			Files: files,
		},
	}, nil
}

func packageJSON(raw []byte) []byte {
	if len(raw) == 0 || string(raw) == "null" {
		return []byte("{}")
	}
	return raw
}
