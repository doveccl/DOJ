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
	"regexp"
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
	plagiarismPrefix          = "plagiarism"
)

var (
	plagiarismLanguageIDs      = []string{"c", "cc", "cpp"}
	plagiarismLanguageSuffixes = []string{".c", ".cc", ".cpp", ".cxx", ".c++"}
)

type PlagiarismJobDTO struct {
	ID         uint       `json:"id"`
	Scope      string     `json:"scope"`
	ScopeID    uint       `json:"scopeId"`
	Status     string     `json:"status"`
	Message    string     `json:"message"`
	ReportURL  string     `json:"reportUrl"`
	CreatedAt  time.Time  `json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}

type PlagiarismJobs struct {
	Items []PlagiarismJobDTO `json:"items"`
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
	return api.streamPlagiarismReport(c, job)
}

func (api *API) plagiarismViewer(c echo.Context) error {
	if _, err := parseUintParam(c, "id", "invalid plagiarism job id"); err != nil {
		return err
	}
	c.Response().Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' https: data: blob:; font-src 'self' data:; connect-src 'self' https://api.github.com")
	return api.proxyPlagiarism(c, "viewer/")
}

func (api *API) plagiarismViewerReport(c echo.Context) error {
	id, ok := plagiarismJobIDFromReferer(c)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "report not found")
	}
	var job models.PlagiarismJob
	if err := api.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "report not found")
		}
		return err
	}
	return api.streamPlagiarismReport(c, job)
}

func (api *API) streamPlagiarismReport(c echo.Context, job models.PlagiarismJob) error {
	if job.Status != plagiarismStatusDone || job.ReportKey == "" {
		return echo.NewHTTPError(http.StatusNotFound, "report not found")
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	reader, _, err := store.Open(c.Request().Context(), job.ReportKey)
	if err != nil {
		return err
	}
	defer reader.Close()
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`inline; filename="plagiarism-%d.jplag"`, job.ID))
	return c.Stream(http.StatusOK, "application/zip", reader)
}

func plagiarismJobIDFromReferer(c echo.Context) (uint, bool) {
	ref := c.Request().Referer()
	if strings.TrimSpace(ref) == "" {
		return 0, false
	}
	u, err := url.Parse(ref)
	if err != nil {
		return 0, false
	}
	if u.Host != "" && !strings.EqualFold(u.Host, c.Request().Host) {
		return 0, false
	}
	if job := u.Query().Get("job"); job != "" {
		id, err := strconv.ParseUint(job, 10, 64)
		return uint(id), err == nil && id > 0
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "jplag" {
		return 0, false
	}
	id, err := strconv.ParseUint(parts[1], 10, 64)
	return uint(id), err == nil && id > 0
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
	}, ModifyResponse: func(resp *http.Response) error {
		if rel != "viewer/" {
			return nil
		}
		return injectPlagiarismViewerEntry(resp, c.Param("id"), c.Request().URL.RawQuery)
	}}
	proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}

func injectPlagiarismViewerEntry(resp *http.Response, jobID string, rawQuery string) error {
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get(echo.HeaderContentType), "text/html") {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	rootQuery, err := url.ParseQuery(rawQuery)
	if err != nil {
		rootQuery = url.Values{}
	}
	rootQuery.Set("job", jobID)
	root := "/?" + rootQuery.Encode()
	original := "/jplag/" + url.PathEscape(jobID)
	if rawQuery != "" {
		original += "?" + rawQuery
	}
	script := fmt.Sprintf(`<script>(()=>{const original=%s,root=%s;history.replaceState(history.state,"",root);addEventListener("DOMContentLoaded",()=>history.replaceState(history.state,"",original),{once:true});})();</script>`, jsonString(original), jsonString(root))
	text := strings.Replace(string(body), `<script type="module"`, script+"\n    <script type=\"module\"", 1)
	if text == string(body) {
		text = script + string(body)
	}
	data := []byte(text)
	resp.Body = io.NopCloser(bytes.NewReader(data))
	resp.ContentLength = int64(len(data))
	resp.Header.Set(echo.HeaderContentLength, strconv.Itoa(len(data)))
	return nil
}

func jsonString(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
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
	report, err = filterJPlagReport(report)
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

var plagiarismReportName = regexp.MustCompile(`P(\d+)#\d+U(\d+)\(`)

func filterJPlagReport(data []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	files := make(map[string][]byte, len(reader.File))
	keepComparison := map[string]bool{}
	for _, file := range reader.File {
		body, err := readZipEntry(file)
		if err != nil {
			return nil, err
		}
		files[file.Name] = body
		if strings.HasPrefix(file.Name, "comparisons/") {
			var item map[string]any
			if err := json.Unmarshal(body, &item); err != nil {
				return nil, err
			}
			keepComparison[file.Name] = differentUsers(asString(item["firstSubmissionId"]), asString(item["secondSubmissionId"]))
		}
	}
	if body, ok := files["topComparisons.json"]; ok {
		var items []map[string]any
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, err
		}
		filtered := items[:0]
		for _, item := range items {
			if differentUsers(asString(item["firstSubmission"]), asString(item["secondSubmission"])) {
				filtered = append(filtered, item)
			}
		}
		if files["topComparisons.json"], err = json.Marshal(filtered); err != nil {
			return nil, err
		}
		if body, ok := files["runInformation.json"]; ok {
			var info map[string]any
			if err := json.Unmarshal(body, &info); err == nil {
				info["totalComparisons"] = len(filtered)
				if body, err := json.Marshal(info); err == nil {
					files["runInformation.json"] = body
				}
			}
		}
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "comparisons/") && !keepComparison[file.Name] {
			continue
		}
		if err := zipBytes(zw, file.Name, files[file.Name]); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readZipEntry(file *zip.File) ([]byte, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	return io.ReadAll(src)
}

func differentUsers(a string, b string) bool {
	_, au, okA := plagiarismProblemUser(a)
	_, bu, okB := plagiarismProblemUser(b)
	return okA && okB && au != bu
}

func plagiarismProblemUser(name string) (string, string, bool) {
	match := plagiarismReportName.FindStringSubmatch(name)
	if len(match) != 3 {
		return "", "", false
	}
	return match[1], match[2], true
}

func asString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func (api *API) plagiarismPackage(scope string, scopeID uint) ([]byte, int, error) {
	problemIDs, err := api.plagiarismProblemIDs(scope, scopeID)
	if err != nil {
		return nil, 0, err
	}
	languageIDs, err := api.plagiarismLanguageIDs()
	if err != nil {
		return nil, 0, err
	}
	queryRows, err := api.plagiarismScopeSubmissions(scope, scopeID, problemIDs, languageIDs)
	if err != nil {
		return nil, 0, err
	}
	queryRows, err = api.plagiarismNewSubmissions(scope, scopeID, queryRows)
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
	if err := api.db.Where("problem_id IN ? AND language IN ? AND status = ?", problemIDs, languageIDs, "AC").Order("created_at asc, id asc").Find(&history).Error; err != nil {
		return nil, 0, err
	}
	users, err := api.plagiarismUserNames(append(queryRows, history...))
	if err != nil {
		return nil, 0, err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// Plagiarism scope: new uses the score-bearing representative submission,
	// old uses all AC history for the same problems; report filtering only drops same-user pairs.
	count := 0
	for _, row := range queryRows {
		name := plagiarismFileName("new", row, users)
		if err := zipText(zw, name, row.Code); err != nil {
			_ = zw.Close()
			return nil, 0, err
		}
		count++
	}
	for _, row := range history {
		if queryIDs[row.ID] {
			continue
		}
		name := plagiarismFileName("old", row, users)
		if err := zipText(zw, name, row.Code); err != nil {
			_ = zw.Close()
			return nil, 0, err
		}
		count++
	}
	if err := zw.Close(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), count, nil
}

func (api *API) plagiarismNewSubmissions(scope string, scopeID uint, rows []models.Submission) ([]models.Submission, error) {
	switch scope {
	case plagiarismScopeAssignment:
		return firstHighestSubmissionPerUserProblem(rows), nil
	case plagiarismScopeContest:
		var contest models.Contest
		if err := api.db.First(&contest, scopeID).Error; err != nil {
			return nil, err
		}
		if contest.Kind == "ICPC" {
			return firstACSubmissionPerUserProblem(rows), nil
		}
		return latestSubmissionPerUserProblem(rows), nil
	default:
		return nil, fmt.Errorf("invalid plagiarism scope")
	}
}

func firstHighestSubmissionPerUserProblem(rows []models.Submission) []models.Submission {
	index := make(map[[2]uint]int, len(rows))
	out := make([]models.Submission, 0, len(rows))
	for _, row := range rows {
		key := [2]uint{row.ProblemID, row.UserID}
		if i, ok := index[key]; ok {
			if row.Score > out[i].Score {
				out[i] = row
			}
			continue
		}
		index[key] = len(out)
		out = append(out, row)
	}
	return out
}

func firstACSubmissionPerUserProblem(rows []models.Submission) []models.Submission {
	index := make(map[[2]uint]bool, len(rows))
	out := make([]models.Submission, 0, len(rows))
	for _, row := range rows {
		if row.Status != "AC" {
			continue
		}
		key := [2]uint{row.ProblemID, row.UserID}
		if index[key] {
			continue
		}
		index[key] = true
		out = append(out, row)
	}
	return out
}

func latestSubmissionPerUserProblem(rows []models.Submission) []models.Submission {
	index := make(map[[2]uint]int, len(rows))
	out := make([]models.Submission, 0, len(rows))
	for _, row := range rows {
		key := [2]uint{row.ProblemID, row.UserID}
		if i, ok := index[key]; ok {
			out[i] = row
			continue
		}
		index[key] = len(out)
		out = append(out, row)
	}
	return out
}

func (api *API) plagiarismUserNames(rows []models.Submission) (map[uint]string, error) {
	ids := make([]uint, 0, len(rows))
	seen := map[uint]bool{}
	for _, row := range rows {
		if !seen[row.UserID] {
			seen[row.UserID] = true
			ids = append(ids, row.UserID)
		}
	}
	var users []models.User
	if err := api.db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	names := make(map[uint]string, len(users))
	for _, user := range users {
		names[user.ID] = user.Name
	}
	return names, nil
}

func (api *API) plagiarismLanguageIDs() ([]string, error) {
	ids := map[string]bool{}
	for _, id := range plagiarismLanguageIDs {
		ids[id] = true
	}
	var rows []models.Language
	if err := api.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		id := strings.ToLower(strings.TrimSpace(row.ID))
		if slices.Contains(plagiarismLanguageIDs, id) || slices.Contains(plagiarismLanguageSuffixes, strings.ToLower(path.Ext(row.Source))) {
			ids[row.ID] = true
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	slices.Sort(out)
	return out, nil
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

func (api *API) plagiarismScopeSubmissions(scope string, scopeID uint, problemIDs []uint, languageIDs []string) ([]models.Submission, error) {
	if len(problemIDs) == 0 {
		return nil, nil
	}
	var rows []models.Submission
	query := api.db.Where("problem_id IN ? AND language IN ? AND status NOT IN ?", problemIDs, languageIDs, []string{"queued", "judging"})
	switch scope {
	case plagiarismScopeAssignment:
		query = query.Where("assignment_id = ?", scopeID)
	case plagiarismScopeContest:
		query = query.Where("contest_id = ?", scopeID)
	default:
		return nil, fmt.Errorf("invalid plagiarism scope")
	}
	err := query.Order("created_at asc, id asc").Find(&rows).Error
	return rows, err
}

func plagiarismFileName(group string, row models.Submission, users map[uint]string) string {
	name := safePlagiarismName(users[row.UserID])
	if name == "" {
		name = fmt.Sprintf("user%d", row.UserID)
	}
	return path.Join(group, fmt.Sprintf("P%d#%dU%d(%s).cc", row.ProblemID, row.ID, row.UserID, name))
}

func safePlagiarismName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && b.String()[b.Len()-1] != '_':
			b.WriteByte('_')
		}
		if b.Len() >= 32 {
			break
		}
	}
	return strings.Trim(b.String(), "_")
}

func zipText(zw *zip.Writer, name string, text string) error {
	return zipBytes(zw, name, []byte(text))
}

func zipBytes(zw *zip.Writer, name string, data []byte) error {
	file, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
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
		dto.ReportURL = fmt.Sprintf("/api/admin/plagiarism/jobs/%d/report.jplag", row.ID)
	}
	return dto
}
