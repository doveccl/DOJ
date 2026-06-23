package judger

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/services/events"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type API struct {
	db        *gorm.DB
	leaseWait time.Duration
}

const (
	contextJudgerID     = "judgerID"
	contextJudgerLocal  = "judgerLocal"
	defaultLeaseSeconds = 60
	defaultLeaseWait    = 25 * time.Second
)

type LeaseRequest struct {
	Version string `json:"version"`
	Host    string `json:"host"`
	Arch    string `json:"arch"`
}

type LeaseResponse struct {
	Task *TaskPayload `json:"task"`
}

type TaskPayload struct {
	ID           uint           `json:"id"`
	SubmissionID uint           `json:"submissionId"`
	Attempt      int            `json:"attempt"`
	Source       string         `json:"source"`
	Lang         LangPayload    `json:"lang"`
	Mode         string         `json:"mode"`
	Limits       LimitsPayload  `json:"limits"`
	Cases        []CasePayload  `json:"cases"`
	Problem      ProblemPayload `json:"problem"`
}

type LangPayload struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	Dockerfile string `json:"dockerfile"`
}

type ProblemPayload struct {
	ID       uint     `json:"id"`
	Mode     string   `json:"mode"`
	TimeMS   int      `json:"timeMs"`
	MemoryMB int      `json:"memoryMb"`
	Tags     []string `json:"tags"`
}

type LimitsPayload struct {
	TimeMS   int `json:"timeMs"`
	MemoryKB int `json:"memoryKb"`
	OutputKB int `json:"outputKb"`
	Pids     int `json:"pids"`
	FileKB   int `json:"fileKb"`
}

type CasePayload struct {
	ID     string `json:"id"`
	Input  string `json:"input"`
	Answer string `json:"answer"`
	Score  int    `json:"score"`
}

type ResultRequest struct {
	SubmissionID uint         `json:"submissionId"`
	Attempt      int          `json:"attempt"`
	Status       string       `json:"status"`
	Score        int          `json:"score"`
	Message      string       `json:"message"`
	TimeMS       *int         `json:"timeMs"`
	MemoryKB     *int         `json:"memoryKb"`
	Cases        []CaseResult `json:"cases"`
}

type HeartbeatRequest struct {
	SubmissionID uint `json:"submissionId"`
	Attempt      int  `json:"attempt"`
}

type CaseResult struct {
	No       int    `json:"no"`
	Status   string `json:"status"`
	Score    int    `json:"score"`
	TimeMS   *int   `json:"timeMs"`
	MemoryKB *int   `json:"memoryKb"`
	Message  string `json:"message"`
}

func Register(e *echo.Echo, db *gorm.DB) {
	if db == nil {
		panic("judger API requires a database")
	}
	api := &API{db: db, leaseWait: defaultLeaseWait}
	group := e.Group("/api/judger", api.auth)
	group.POST("/lease", api.lease)
	group.GET("/tasks/:id/assets.zip", api.taskAssets)
	group.POST("/tasks/:id/heartbeat", api.heartbeat)
	group.POST("/tasks/:id/result", api.result)
}

func (api *API) auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if isDirectLoopbackRequest(c) {
			c.Set(contextJudgerLocal, true)
			return next(c)
		}
		token, ok := bearerToken(c)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
		}
		var row models.Judger
		if err := api.db.First(&row, "auth = ?", utils.TokenHash(token)).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
			}
			return err
		}
		TouchStatus(c.Request().Context(), row.ID, time.Now())
		c.Set(contextJudgerID, row.ID)
		return next(c)
	}
}

func (api *API) lease(c echo.Context) error {
	var req LeaseRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	judgerID, err := api.ensureJudger(c, req)
	if err != nil {
		return err
	}

	ch, unsubscribe := events.Default.Subscribe()
	defer unsubscribe()
	timer := time.NewTimer(api.longPollWait())
	defer timer.Stop()
	active := time.NewTicker(5 * time.Second)
	defer active.Stop()

	for {
		payload, err := api.tryLease(c.Request().Context(), judgerID)
		if err != nil {
			return err
		}
		if payload != nil {
			events.SubmissionChanged()
			return c.JSON(http.StatusOK, LeaseResponse{Task: payload})
		}

		select {
		case <-c.Request().Context().Done():
			return nil
		case <-timer.C:
			return c.JSON(http.StatusOK, LeaseResponse{})
		case now := <-active.C:
			TouchStatus(c.Request().Context(), judgerID, now)
		case <-ch:
		}
	}
}

func (api *API) longPollWait() time.Duration {
	if api.leaseWait > 0 {
		return api.leaseWait
	}
	return defaultLeaseWait
}

func (api *API) tryLease(ctx context.Context, judgerID uint) (*TaskPayload, error) {
	now := time.Now()
	var payload *TaskPayload
	err := api.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []models.Submission
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? OR (status = ? AND (lease_until IS NULL OR lease_until < ?))", "queued", "judging", now).
			Order("created_at asc").
			Limit(1).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		submission := rows[0]
		nextAttempt := submission.Attempt + 1
		until := now.Add(defaultLeaseSeconds * time.Second)
		updates := map[string]any{
			"status":      "judging",
			"attempt":     nextAttempt,
			"judger_id":   judgerID,
			"lease_until": until,
		}
		if err := tx.Model(&models.Submission{}).Where("id = ?", submission.ID).Updates(updates).Error; err != nil {
			return err
		}
		submission.Status = "judging"
		submission.Attempt = nextAttempt
		submission.JudgerID = &judgerID
		submission.LeaseUntil = &until
		got, err := buildPayload(ctx, tx, submission)
		if err != nil {
			return err
		}
		payload = got
		return nil
	})
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (api *API) taskAssets(c echo.Context) error {

	submissionID, err := parseTaskID(c)
	if err != nil {
		return err
	}
	if err := api.authorizeSubmission(c, submissionID); err != nil {
		return err
	}
	var submission models.Submission
	if err := api.db.First(&submission, submissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "task not found")
		}
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	files, err := listProblemObjects(c.Request().Context(), store, submission.ProblemID)
	if err != nil {
		return err
	}
	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="task-%d-assets.zip"`, submission.ID))
	c.Response().WriteHeader(http.StatusOK)
	writer := zip.NewWriter(c.Response().Writer)
	defer writer.Close()
	return writeTaskAssetZip(c.Request().Context(), writer, store, submission.ProblemID, files)
}

func (api *API) heartbeat(c echo.Context) error {
	var req HeartbeatRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := validateHeartbeat(req); err != nil {
		return err
	}

	taskID, err := parseTaskID(c)
	if err != nil {
		return err
	}
	if taskID != req.SubmissionID {
		return c.NoContent(http.StatusAccepted)
	}

	applied := false
	ownerID := uint(0)
	now := time.Now()
	err = api.db.Transaction(func(tx *gorm.DB) error {
		var submission models.Submission
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&submission, req.SubmissionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		if submission.Attempt != req.Attempt {
			return nil
		}
		if err := api.requireTaskOwner(c, submission); err != nil {
			return err
		}
		if submission.JudgerID != nil {
			ownerID = *submission.JudgerID
		}
		until := now.Add(defaultLeaseSeconds * time.Second)
		if err := tx.Model(&models.Submission{}).Where("id = ? AND attempt = ?", submission.ID, req.Attempt).Update("lease_until", until).Error; err != nil {
			return err
		}
		applied = true
		return nil
	})
	if err != nil {
		return err
	}
	if applied {
		TouchStatus(c.Request().Context(), ownerID, now)
	}
	return c.NoContent(http.StatusAccepted)
}

func (api *API) result(c echo.Context) error {
	var req ResultRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := validateResult(req); err != nil {
		return err
	}

	taskID, err := parseTaskID(c)
	if err != nil {
		return err
	}
	if taskID != req.SubmissionID {
		return c.NoContent(http.StatusAccepted)
	}

	applied := false
	ownerID := uint(0)
	err = api.db.Transaction(func(tx *gorm.DB) error {
		var submission models.Submission
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&submission, req.SubmissionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		if submission.Attempt != req.Attempt {
			return nil
		}
		if err := api.requireTaskOwner(c, submission); err != nil {
			return err
		}
		if submission.JudgerID != nil {
			ownerID = *submission.JudgerID
		}
		update := map[string]any{
			"status":      req.Status,
			"score":       req.Score,
			"message":     req.Message,
			"time_ms":     req.TimeMS,
			"memory_kb":   req.MemoryKB,
			"judger_id":   nil,
			"lease_until": nil,
		}
		if err := tx.Model(&models.Submission{}).Where("id = ?", req.SubmissionID).Updates(update).Error; err != nil {
			return err
		}
		if len(req.Cases) > 0 {
			if err := tx.Where("submission_id = ?", req.SubmissionID).Delete(&models.Case{}).Error; err != nil {
				return err
			}
			for _, item := range req.Cases {
				row := models.Case{
					SubmissionID: req.SubmissionID,
					No:           item.No,
					Status:       item.Status,
					TimeMS:       item.TimeMS,
					MemoryKB:     item.MemoryKB,
					Message:      item.Message,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			}
		}
		applied = true
		return nil
	})
	if err != nil {
		return err
	}
	if applied {
		TouchStatus(c.Request().Context(), ownerID, time.Now())
		events.SubmissionChanged()
	}
	return c.NoContent(http.StatusAccepted)
}

func (api *API) ensureJudger(c echo.Context, req LeaseRequest) (uint, error) {
	if id, ok := c.Get(contextJudgerID).(uint); ok && id > 0 {
		if host := strings.TrimSpace(req.Host); host != "" {
			if err := api.db.Model(&models.Judger{}).Where("id = ? AND name = ?", id, "").Update("name", host).Error; err != nil {
				return 0, err
			}
		}
		return id, nil
	}
	if !isLocalJudger(c) {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
	}
	name := judgerRequestName(req)
	var row models.Judger
	err := api.db.First(&row, "name = ?", name).Error
	if err == gorm.ErrRecordNotFound {
		row = models.Judger{Name: name, Auth: ""}
		if err := api.db.Create(&row).Error; err != nil {
			return 0, err
		}
		c.Set(contextJudgerID, row.ID)
		TouchStatus(c.Request().Context(), row.ID, time.Now())
		return row.ID, nil
	}
	if err != nil {
		return 0, err
	}
	c.Set(contextJudgerID, row.ID)
	TouchStatus(c.Request().Context(), row.ID, time.Now())
	return row.ID, nil
}

func (api *API) authorizeSubmission(c echo.Context, submissionID uint) error {
	if isLocalJudger(c) {
		return nil
	}
	judgerID, ok := c.Get(contextJudgerID).(uint)
	if !ok || judgerID == 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
	}
	var submission models.Submission
	if err := api.db.First(&submission, submissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "task not found")
		}
		return err
	}
	if submission.JudgerID == nil || *submission.JudgerID != judgerID {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
	}
	return nil
}

func (api *API) requireTaskOwner(c echo.Context, submission models.Submission) error {
	if isLocalJudger(c) {
		return nil
	}
	id := taskJudgerID(c)
	if id == 0 || submission.JudgerID == nil || *submission.JudgerID != id {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
	}
	return nil
}

func taskJudgerID(c echo.Context) uint {
	id, _ := c.Get(contextJudgerID).(uint)
	return id
}

func bearerToken(c echo.Context) (string, bool) {
	header := strings.TrimSpace(c.Request().Header.Get("Authorization"))
	value, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return value, true
}

func isDirectLoopbackRequest(c echo.Context) bool {
	if hasForwardedHeaders(c) {
		return false
	}
	host, _, err := net.SplitHostPort(c.Request().RemoteAddr)
	if err != nil {
		host = c.Request().RemoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && ip.IsLoopback()
}

func hasForwardedHeaders(c echo.Context) bool {
	headers := c.Request().Header
	for _, name := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-IP"} {
		if strings.TrimSpace(headers.Get(name)) != "" {
			return true
		}
	}
	return false
}

func isLocalJudger(c echo.Context) bool {
	value, _ := c.Get(contextJudgerLocal).(bool)
	return value
}

func validateResult(req ResultRequest) error {
	if req.SubmissionID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid result target")
	}
	if req.Attempt <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid result attempt")
	}
	if !validVerdict(req.Status) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid result status")
	}
	if req.Score < 0 || req.Score > 100 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid result score")
	}
	for _, item := range req.Cases {
		if item.No <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid case number")
		}
		if !validVerdict(item.Status) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid case status")
		}
	}
	return nil
}

func validateHeartbeat(req HeartbeatRequest) error {
	if req.SubmissionID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid heartbeat target")
	}
	if req.Attempt <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid heartbeat attempt")
	}
	return nil
}

func validVerdict(status string) bool {
	switch status {
	case "AC", "CE", "WA", "PE", "TLE", "MLE", "OLE", "RE", "SE":
		return true
	default:
		return false
	}
}

func judgerRequestName(req LeaseRequest) string {
	name := strings.TrimSpace(req.Host)
	if name != "" {
		return name
	}
	return "local-judger"
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
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return nil, err
	}
	dataFiles, err := store.List(ctx, problemAssetPrefix(problem.ID, "data"))
	if err != nil {
		return nil, err
	}
	cases := casePayloadsFromObjects(problem.ID, dataFiles)
	return &TaskPayload{
		ID:           submission.ID,
		SubmissionID: submission.ID,
		Attempt:      submission.Attempt,
		Source:       submission.Code,
		Lang: LangPayload{
			ID:         lang.ID,
			Source:     lang.Source,
			Dockerfile: lang.Dockerfile,
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
			ID:       problem.ID,
			Mode:     problem.Mode,
			TimeMS:   problem.TimeMS,
			MemoryMB: problem.MemoryMB,
			Tags:     readTags(problem.Tags),
		},
	}, nil
}

func listProblemObjects(ctx context.Context, store utils.ObjectStore, problemID uint) ([]utils.ObjectInfo, error) {
	var files []utils.ObjectInfo
	for _, section := range []string{"data", "judge"} {
		got, err := store.List(ctx, problemAssetPrefix(problemID, section))
		if err != nil {
			return nil, err
		}
		files = append(files, got...)
	}
	return files, nil
}

func writeTaskAssetZip(ctx context.Context, writer *zip.Writer, store utils.ObjectStore, problemID uint, files []utils.ObjectInfo) error {
	for _, object := range files {
		section, name, ok := taskAssetZipName(problemID, object.Key)
		if !ok {
			continue
		}
		zipName, ok := safeTaskAssetZipName(section, name)
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

func safeTaskAssetZipName(section string, name string) (string, bool) {
	normalized := strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	clean, err := utils.CleanObjectKey(normalized)
	if err != nil || clean != normalized {
		return "", false
	}
	return path.Join(section, clean), true
}

func taskAssetZipName(problemID uint, key string) (string, string, bool) {
	for _, section := range []string{"data", "judge"} {
		prefix := problemAssetPrefix(problemID, section) + "/"
		if strings.HasPrefix(key, prefix) {
			name := strings.TrimPrefix(key, prefix)
			return section, name, name != ""
		}
	}
	return "", "", false
}

func casePayloadsFromObjects(problemID uint, objects []utils.ObjectInfo) []CasePayload {
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
		stem, kind := utils.DataCaseStem(name)
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
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].id < pairs[j].id })
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

func parseTaskID(c echo.Context) (uint, error) {
	raw := strings.TrimSpace(c.Param("id"))
	var value uint64
	for _, char := range raw {
		if char < '0' || char > '9' {
			return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid task id")
		}
		value = value*10 + uint64(char-'0')
	}
	if value == 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid task id")
	}
	return uint(value), nil
}
