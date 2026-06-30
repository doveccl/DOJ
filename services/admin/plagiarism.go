package admin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const (
	plagiarismScopeAssignment = "assignment"
	plagiarismScopeContest    = "contest"
	plagiarismStatusQueued    = "queued"
	plagiarismStatusRunning   = "running"
	plagiarismStatusDone      = "done"
	plagiarismStatusFailed    = "failed"
	plagiarismLanguage        = "cpp"
	plagiarismPrefix          = "plagiarism"
)

type PlagiarismJobDTO struct {
	ID         uint       `json:"id"`
	Scope      string     `json:"scope"`
	ScopeID    uint       `json:"scopeId"`
	Status     string     `json:"status"`
	Message    string     `json:"message"`
	ReportURL  string     `json:"reportUrl"`
	ViewerURL  string     `json:"viewerUrl"`
	CreatedAt  time.Time  `json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}

type PlagiarismJobs struct {
	Items []PlagiarismJobDTO `json:"items"`
}

type plagiarismManifest struct {
	Scope       string                 `json:"scope"`
	ScopeID     uint                   `json:"scopeId"`
	Language    string                 `json:"language"`
	NewDir      string                 `json:"newDir"`
	OldDir      string                 `json:"oldDir"`
	Submissions []plagiarismSubmission `json:"submissions"`
}

type plagiarismSubmission struct {
	ID        uint   `json:"id"`
	UserID    uint   `json:"userId"`
	ProblemID uint   `json:"problemId"`
	Group     string `json:"group"`
	File      string `json:"file"`
}

func (api *API) plagiarismJobs(c echo.Context) error {
	scope, id, err := parsePlagiarismScope(c.QueryParam("scope"), c.QueryParam("id"))
	if err != nil {
		return err
	}
	var rows []models.PlagiarismJob
	if err := api.db.Where("scope = ? AND scope_id = ?", scope, id).Order("id desc").Limit(20).Find(&rows).Error; err != nil {
		return err
	}
	items := make([]PlagiarismJobDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, plagiarismJobDTO(row))
	}
	return c.JSON(http.StatusOK, PlagiarismJobs{Items: items})
}

func (api *API) createAssignmentPlagiarismJob(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid assignment id")
	if err != nil {
		return err
	}
	if err := api.db.First(&models.Assignment{}, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "assignment not found")
		}
		return err
	}
	return api.createPlagiarismJob(c, plagiarismScopeAssignment, id)
}

func (api *API) createContestPlagiarismJob(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid contest id")
	if err != nil {
		return err
	}
	if err := api.db.First(&models.Contest{}, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "contest not found")
		}
		return err
	}
	return api.createPlagiarismJob(c, plagiarismScopeContest, id)
}

func (api *API) createPlagiarismJob(c echo.Context, scope string, scopeID uint) error {
	if strings.TrimSpace(os.Getenv("JPLAG")) == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "JPlag service is not configured")
	}
	job := models.PlagiarismJob{Scope: scope, ScopeID: scopeID, Status: plagiarismStatusQueued}
	if err := api.db.Create(&job).Error; err != nil {
		return err
	}
	go api.runPlagiarismJob(job.ID)
	return c.JSON(http.StatusAccepted, plagiarismJobDTO(job))
}

func (api *API) plagiarismReport(c echo.Context) error {
	job, err := api.plagiarismJobFromParam(c)
	if err != nil {
		return err
	}
	if job.Status != plagiarismStatusDone || job.ReportKey == "" {
		return echo.NewHTTPError(http.StatusNotFound, "report not found")
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	reader, contentType, err := store.Open(c.Request().Context(), job.ReportKey)
	if err != nil {
		return err
	}
	defer reader.Close()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`inline; filename="plagiarism-%d.jplag"`, job.ID))
	return c.Stream(http.StatusOK, contentType, reader)
}

func (api *API) plagiarismViewer(c echo.Context) error {
	return api.proxyPlagiarism(c, path.Join("viewer", strings.TrimPrefix(c.Param("*"), "/")))
}

func (api *API) plagiarismViewerAsset(c echo.Context) error {
	return api.proxyPlagiarism(c, path.Join("JPlag", strings.TrimPrefix(c.Param("*"), "/")))
}

func (api *API) proxyPlagiarism(c echo.Context, rel string) error {
	target := strings.TrimRight(strings.TrimSpace(os.Getenv("JPLAG")), "/")
	if target == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "JPlag service is not configured")
	}
	base, err := url.Parse(target)
	if err != nil {
		return err
	}
	proxy := &httputil.ReverseProxy{Rewrite: func(req *httputil.ProxyRequest) {
		req.SetURL(base)
		req.Out.URL.Path = "/" + strings.TrimLeft(path.Join(base.Path, rel), "/")
		if strings.HasSuffix(c.Request().URL.Path, "/") && !strings.HasSuffix(req.Out.URL.Path, "/") {
			req.Out.URL.Path += "/"
		}
		req.Out.URL.RawQuery = c.Request().URL.RawQuery
		req.Out.Header.Del(echo.HeaderCookie)
	}}
	proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}

func (api *API) runPlagiarismJob(id uint) {
	ctx := context.Background()
	finish := func(status string, message string, update map[string]any) {
		now := time.Now()
		update["status"] = status
		update["message"] = message
		update["finished_at"] = &now
		_ = api.db.Model(&models.PlagiarismJob{}).Where("id = ?", id).Updates(update).Error
	}
	_ = api.db.Model(&models.PlagiarismJob{}).Where("id = ?", id).Updates(map[string]any{"status": plagiarismStatusRunning}).Error
	var job models.PlagiarismJob
	if err := api.db.First(&job, id).Error; err != nil {
		return
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		finish(plagiarismStatusFailed, err.Error(), map[string]any{})
		return
	}
	payload, count, err := api.plagiarismPackage(job.Scope, job.ScopeID)
	if err != nil {
		finish(plagiarismStatusFailed, err.Error(), map[string]any{})
		return
	}
	if count == 0 {
		finish(plagiarismStatusFailed, "no C++ submissions to analyze", map[string]any{})
		return
	}
	inputKey := path.Join(plagiarismPrefix, fmt.Sprintf("%s-%d", job.Scope, job.ScopeID), fmt.Sprintf("job-%d-input.zip", job.ID))
	if err := store.Put(ctx, inputKey, bytes.NewReader(payload), int64(len(payload)), "application/zip"); err != nil {
		finish(plagiarismStatusFailed, err.Error(), map[string]any{})
		return
	}
	report, err := runJPlag(ctx, payload)
	if err != nil {
		finish(plagiarismStatusFailed, err.Error(), map[string]any{"input_key": inputKey})
		return
	}
	reportKey := path.Join(plagiarismPrefix, fmt.Sprintf("%s-%d", job.Scope, job.ScopeID), fmt.Sprintf("job-%d-report.jplag", job.ID))
	if err := store.Put(ctx, reportKey, bytes.NewReader(report), int64(len(report)), "application/octet-stream"); err != nil {
		finish(plagiarismStatusFailed, err.Error(), map[string]any{"input_key": inputKey})
		return
	}
	finish(plagiarismStatusDone, fmt.Sprintf("analyzed %d C++ submissions", count), map[string]any{"input_key": inputKey, "report_key": reportKey})
}

func runJPlag(ctx context.Context, payload []byte) ([]byte, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("JPLAG")), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("JPLAG is not configured")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "input.zip")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(payload); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/run", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	resp, err := (&http.Client{Timeout: 15 * time.Minute}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("JPlag failed: %s", jplagFailure(data))
	}
	return data, nil
}

func jplagFailure(data []byte) string {
	message := strings.TrimSpace(string(data))
	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &body); err == nil && strings.TrimSpace(body.Message) != "" {
		message = strings.TrimSpace(body.Message)
	}
	lines := strings.Split(message, "\n")
	items := lines[:0]
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "JPlagVersionChecker") {
			continue
		}
		items = append(items, line)
	}
	return strings.TrimPrefix(strings.Join(items, "\n"), "exit status 1: ")
}

func (api *API) plagiarismPackage(scope string, scopeID uint) ([]byte, int, error) {
	problemIDs, err := api.plagiarismProblemIDs(scope, scopeID)
	if err != nil {
		return nil, 0, err
	}
	queryRows, err := api.plagiarismScopeSubmissions(scope, scopeID, problemIDs)
	if err != nil {
		return nil, 0, err
	}
	if len(queryRows) == 0 {
		return nil, 0, nil
	}
	queryIDs := make(map[uint]bool, len(queryRows))
	for _, row := range queryRows {
		queryIDs[row.ID] = true
	}
	var history []models.Submission
	if err := api.db.Where("problem_id IN ? AND language = ? AND status = ?", problemIDs, plagiarismLanguage, "AC").Order("id asc").Find(&history).Error; err != nil {
		return nil, 0, err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	manifest := plagiarismManifest{Scope: scope, ScopeID: scopeID, Language: plagiarismLanguage, NewDir: "new", OldDir: "old"}
	for _, row := range queryRows {
		name := plagiarismFileName("new", row)
		if err := zipText(zw, name, row.Code); err != nil {
			_ = zw.Close()
			return nil, 0, err
		}
		manifest.Submissions = append(manifest.Submissions, plagiarismSubmission{ID: row.ID, UserID: row.UserID, ProblemID: row.ProblemID, Group: "new", File: name})
	}
	queryUsers := make(map[uint]bool, len(queryRows))
	for _, row := range queryRows {
		queryUsers[row.UserID] = true
	}
	for _, row := range history {
		if queryIDs[row.ID] || queryUsers[row.UserID] {
			continue
		}
		name := plagiarismFileName("old", row)
		if err := zipText(zw, name, row.Code); err != nil {
			_ = zw.Close()
			return nil, 0, err
		}
		manifest.Submissions = append(manifest.Submissions, plagiarismSubmission{ID: row.ID, UserID: row.UserID, ProblemID: row.ProblemID, Group: "old", File: name})
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = zw.Close()
		return nil, 0, err
	}
	if err := zipText(zw, "manifest.json", string(data)); err != nil {
		_ = zw.Close()
		return nil, 0, err
	}
	if err := zw.Close(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), len(manifest.Submissions), nil
}

func (api *API) plagiarismProblemIDs(scope string, scopeID uint) ([]uint, error) {
	var ids []uint
	switch scope {
	case plagiarismScopeAssignment:
		err := api.db.Model(&models.AssignmentProblem{}).Where("assignment_id = ?", scopeID).Pluck("problem_id", &ids).Error
		return ids, err
	case plagiarismScopeContest:
		err := api.db.Model(&models.ContestProblem{}).Where("contest_id = ?", scopeID).Pluck("problem_id", &ids).Error
		return ids, err
	default:
		return nil, fmt.Errorf("invalid plagiarism scope")
	}
}

func (api *API) plagiarismScopeSubmissions(scope string, scopeID uint, problemIDs []uint) ([]models.Submission, error) {
	if len(problemIDs) == 0 {
		return nil, nil
	}
	var rows []models.Submission
	query := api.db.Where("problem_id IN ? AND language = ? AND status = ?", problemIDs, plagiarismLanguage, "AC")
	switch scope {
	case plagiarismScopeAssignment:
		query = query.Where("assignment_id = ?", scopeID)
	case plagiarismScopeContest:
		query = query.Where("contest_id = ?", scopeID)
	default:
		return nil, fmt.Errorf("invalid plagiarism scope")
	}
	err := query.Order("id asc").Find(&rows).Error
	return rows, err
}

func plagiarismFileName(group string, row models.Submission) string {
	return path.Join(group, fmt.Sprintf("p%d_sub%d_user%d.cpp", row.ProblemID, row.ID, row.UserID))
}

func zipText(zw *zip.Writer, name string, text string) error {
	file, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(text))
	return err
}

func (api *API) plagiarismJobFromParam(c echo.Context) (models.PlagiarismJob, error) {
	id, err := parseUintParam(c, "id", "invalid plagiarism job id")
	if err != nil {
		return models.PlagiarismJob{}, err
	}
	var job models.PlagiarismJob
	if err := api.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return models.PlagiarismJob{}, echo.NewHTTPError(http.StatusNotFound, "plagiarism job not found")
		}
		return models.PlagiarismJob{}, err
	}
	return job, nil
}

func parsePlagiarismScope(scope string, rawID string) (string, uint, error) {
	scope = strings.TrimSpace(scope)
	if !slices.Contains([]string{plagiarismScopeAssignment, plagiarismScopeContest}, scope) {
		return "", 0, echo.NewHTTPError(http.StatusBadRequest, "invalid plagiarism scope")
	}
	parsed, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || parsed == 0 {
		return "", 0, echo.NewHTTPError(http.StatusBadRequest, "invalid plagiarism scope id")
	}
	return scope, uint(parsed), nil
}

func plagiarismJobDTO(row models.PlagiarismJob) PlagiarismJobDTO {
	dto := PlagiarismJobDTO{
		ID:         row.ID,
		Scope:      row.Scope,
		ScopeID:    row.ScopeID,
		Status:     row.Status,
		Message:    row.Message,
		CreatedAt:  row.CreatedAt,
		FinishedAt: row.FinishedAt,
	}
	if row.Status == plagiarismStatusDone && row.ReportKey != "" {
		report := fmt.Sprintf("/api/admin/plagiarism/jobs/%d/report.jplag", row.ID)
		dto.ReportURL = report
		dto.ViewerURL = "/api/admin/plagiarism/viewer/?file=" + url.QueryEscape(report)
	}
	return dto
}
