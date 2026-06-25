package judger

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCasePayloadsFromObjects(t *testing.T) {
	cases := casePayloadsFromObjects(1000, []utils.ObjectInfo{
		{Key: "problems/1000/data/2.out", Size: 2},
		{Key: "problems/1000/data/1.in", Size: 4},
		{Key: "problems/1000/data/1.out", Size: 2},
		{Key: "problems/1000/data/2.in", Size: 4},
		{Key: "problems/1000/data/readme.txt", Size: 10},
	})
	if len(cases) != 2 {
		t.Fatalf("cases = %+v", cases)
	}
	if cases[0].ID != "1" || cases[0].Input != "data/1.in" || cases[0].Answer != "data/1.out" || cases[0].Score != 50 {
		t.Fatalf("case 1 = %+v", cases[0])
	}
	if cases[1].ID != "2" || cases[1].Input != "data/2.in" || cases[1].Answer != "data/2.out" || cases[1].Score != 50 {
		t.Fatalf("case 2 = %+v", cases[1])
	}
}

func TestCasePayloadsFromCommonDataNames(t *testing.T) {
	cases := casePayloadsFromObjects(1000, []utils.ObjectInfo{
		{Key: "problems/1000/data/output2.txt", Size: 2},
		{Key: "problems/1000/data/input1.txt", Size: 4},
		{Key: "problems/1000/data/answer1.txt", Size: 2},
		{Key: "problems/1000/data/input2.txt", Size: 4},
		{Key: "problems/1000/data/readme.txt", Size: 10},
	})
	if len(cases) != 2 {
		t.Fatalf("cases = %+v", cases)
	}
	if cases[0].ID != "1" || cases[0].Input != "data/input1.txt" || cases[0].Answer != "data/answer1.txt" || cases[0].Score != 50 {
		t.Fatalf("case 1 = %+v", cases[0])
	}
	if cases[1].ID != "2" || cases[1].Input != "data/input2.txt" || cases[1].Answer != "data/output2.txt" || cases[1].Score != 50 {
		t.Fatalf("case 2 = %+v", cases[1])
	}
}

func TestValidateResult(t *testing.T) {
	valid := ResultRequest{
		SubmissionID: 1,
		Attempt:      1,
		Status:       "AC",
		Score:        100,
		Cases:        []CaseResult{{No: 1, Status: "AC", Score: 100}},
	}
	if err := validateResult(valid); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}

	tests := []struct {
		name string
		req  ResultRequest
		want int
	}{
		{name: "missing target", req: ResultRequest{Status: "AC", Score: 100}, want: http.StatusBadRequest},
		{name: "missing attempt", req: ResultRequest{SubmissionID: 1, Status: "AC", Score: 100}, want: http.StatusBadRequest},
		{name: "bad status", req: ResultRequest{SubmissionID: 1, Attempt: 1, Status: "queued", Score: 0}, want: http.StatusBadRequest},
		{name: "negative score", req: ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: -1}, want: http.StatusBadRequest},
		{name: "large score", req: ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 101}, want: http.StatusBadRequest},
		{name: "large message", req: ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 0, Message: strings.Repeat("x", utils.MaxJudgerMessageBytes+1)}, want: http.StatusRequestEntityTooLarge},
		{name: "bad case no", req: ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 0, Cases: []CaseResult{{No: 0, Status: "WA"}}}, want: http.StatusBadRequest},
		{name: "bad case status", req: ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 0, Cases: []CaseResult{{No: 1, Status: "queued"}}}, want: http.StatusBadRequest},
		{name: "large case message", req: ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 0, Cases: []CaseResult{{No: 1, Status: "WA", Message: strings.Repeat("x", models.CaseMessageMax+1)}}}, want: http.StatusRequestEntityTooLarge},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			err := validateResult(item.req)
			if err == nil {
				t.Fatal("expected validation error")
			}
			expectHTTPStatus(t, err, item.want)
		})
	}
}

func TestAuthAndLeaseAuthorization(t *testing.T) {
	db := newJudgerTestDB(t)
	remote := models.Judger{Name: "linux-a", Auth: utils.TokenHash("token-a")}
	if err := db.Create(&remote).Error; err != nil {
		t.Fatalf("create judger: %v", err)
	}
	api := &API{db: db}

	c := remoteJudgerContext(t, api, "token-a")
	if id, ok := c.Get(contextJudgerID).(uint); !ok || id != remote.ID {
		t.Fatalf("remote judger id = %#v", c.Get(contextJudgerID))
	}
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "judging", Attempt: 1}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	if err := api.authorizeSubmission(c, 1); err == nil {
		t.Fatal("remote judger without lease should not authorize submission")
	}
	until := time.Now().Add(time.Minute)
	if err := db.Model(&models.Submission{}).Where("id = ?", submission.ID).Updates(map[string]any{"judger_id": remote.ID, "lease_until": until}).Error; err != nil {
		t.Fatalf("set lease: %v", err)
	}
	if err := api.authorizeSubmission(c, submission.ID); err != nil {
		t.Fatalf("remote judger lease rejected: %v", err)
	}

	wrong := remoteJudgerContextAllowError(t, api, "wrong-token")
	if wrong == nil {
		t.Fatal("wrong token unexpectedly authenticated")
	}

	local := localJudgerContext(t, api)
	if !isLocalJudger(local) {
		t.Fatal("loopback request was not marked local")
	}
	id, err := api.ensureJudger(local, LeaseRequest{})
	if err != nil || id == 0 {
		t.Fatalf("local judger ensure failed: id=%d err=%v", id, err)
	}
	if err := api.authorizeSubmission(local, 999); err != nil {
		t.Fatalf("local judger should authorize any leased submission: %v", err)
	}
}

func TestProxyLoopbackDoesNotBypassJudgerAuth(t *testing.T) {
	db := newJudgerTestDB(t)
	api := &API{db: db}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/judger/lease", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	req.Header.Set("X-Forwarded-Host", "example.test")
	req.Header.Set("X-Forwarded-Proto", "https")
	c := e.NewContext(req, httptest.NewRecorder())

	err := api.auth(func(echo.Context) error { return nil })(c)
	if err == nil {
		t.Fatal("proxied loopback request bypassed judger auth")
	}
	expectHTTPStatus(t, err, http.StatusUnauthorized)
	if isLocalJudger(c) {
		t.Fatal("proxied loopback request was marked local")
	}
}

func TestLeaseLongPollsWhenNoTask(t *testing.T) {
	db := newJudgerTestDB(t)
	api := &API{db: db, leaseWait: 30 * time.Millisecond}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/judger/lease", strings.NewReader("{}"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	started := time.Now()
	if err := api.auth(api.lease)(c); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if elapsed := time.Since(started); elapsed < 20*time.Millisecond {
		t.Fatalf("lease returned too early after %s", elapsed)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"task":null}` {
		t.Fatalf("response = %s", got)
	}
}

func TestResultIgnoresStaleAttempt(t *testing.T) {
	db := newJudgerTestDB(t)
	remote := models.Judger{Name: "linux-a", Auth: utils.TokenHash("token-a")}
	if err := db.Create(&remote).Error; err != nil {
		t.Fatalf("create judger: %v", err)
	}
	until := time.Now().Add(time.Minute)
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "judging", Attempt: 2, JudgerID: &remote.ID, LeaseUntil: &until}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	e := echo.New()
	Register(e, db)
	target := "/api/judger/tasks/" + strconv.FormatUint(uint64(submission.ID), 10) + "/result"

	stale := `{"submissionId":` + strconv.FormatUint(uint64(submission.ID), 10) + `,"attempt":1,"status":"AC","score":100,"message":"old","cases":[]}`
	res := judgerJSON(e, target, "token-a", stale)
	if res.Code != http.StatusAccepted {
		t.Fatalf("stale result got %d body=%s", res.Code, res.Body.String())
	}
	var got models.Submission
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("read submission: %v", err)
	}
	if got.Status != "judging" || got.Score != 0 {
		t.Fatalf("stale attempt changed submission: %+v", got)
	}

	current := `{"submissionId":` + strconv.FormatUint(uint64(submission.ID), 10) + `,"attempt":2,"status":"AC","score":100,"message":"ok","cases":[{"no":1,"status":"AC","score":100}]}`
	res = judgerJSON(e, target, "token-a", current)
	if res.Code != http.StatusAccepted {
		t.Fatalf("current result got %d body=%s", res.Code, res.Body.String())
	}
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("read updated submission: %v", err)
	}
	if got.Status != "AC" || got.Score != 100 || got.Message != "ok" {
		t.Fatalf("current attempt did not update submission: %+v", got)
	}
	var cases int64
	if err := db.Model(&models.Case{}).Where("submission_id = ?", submission.ID).Count(&cases).Error; err != nil {
		t.Fatalf("count cases: %v", err)
	}
	if cases != 1 {
		t.Fatalf("case results = %d, want 1", cases)
	}
}

func TestTryLeaseStoresDatabaseLease(t *testing.T) {
	db := newJudgerTestDB(t)
	seedTaskData(t, db)
	remote := models.Judger{Name: "linux-a", Auth: utils.TokenHash("token-a")}
	if err := db.Create(&remote).Error; err != nil {
		t.Fatalf("create judger: %v", err)
	}
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "queued"}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	api := &API{db: db}
	payload, err := api.tryLease(t.Context(), remote.ID)
	if err != nil {
		t.Fatalf("try lease: %v", err)
	}
	if payload == nil || payload.SubmissionID != submission.ID || payload.Attempt != 1 {
		t.Fatalf("payload = %+v", payload)
	}
	var got models.Submission
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("read submission: %v", err)
	}
	if got.Status != "judging" || got.Attempt != 1 || got.JudgerID == nil || *got.JudgerID != remote.ID || got.LeaseUntil == nil {
		t.Fatalf("submission lease not stored in db: %+v", got)
	}
}

func TestHeartbeatExtendsDatabaseLease(t *testing.T) {
	db := newJudgerTestDB(t)
	remote := models.Judger{Name: "linux-a", Auth: utils.TokenHash("token-a")}
	if err := db.Create(&remote).Error; err != nil {
		t.Fatalf("create judger: %v", err)
	}
	oldLease := time.Now().Add(5 * time.Second)
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "judging", Attempt: 3, JudgerID: &remote.ID, LeaseUntil: &oldLease}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	e := echo.New()
	Register(e, db)
	target := "/api/judger/tasks/" + strconv.FormatUint(uint64(submission.ID), 10) + "/heartbeat"
	body := `{"submissionId":` + strconv.FormatUint(uint64(submission.ID), 10) + `,"attempt":3}`
	res := judgerJSON(e, target, "token-a", body)
	if res.Code != http.StatusAccepted {
		t.Fatalf("heartbeat got %d body=%s", res.Code, res.Body.String())
	}
	var got models.Submission
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("read submission: %v", err)
	}
	if got.LeaseUntil == nil || !got.LeaseUntil.After(oldLease) {
		t.Fatalf("lease not extended: old=%s got=%v", oldLease, got.LeaseUntil)
	}
	if got.JudgerID == nil || *got.JudgerID != remote.ID || got.Attempt != 3 {
		t.Fatalf("heartbeat changed owner/attempt: %+v", got)
	}
}

func TestJudgerRequestNameDefault(t *testing.T) {
	if got := judgerRequestName(LeaseRequest{Host: " linux-a "}); got != "linux-a" {
		t.Fatalf("name = %q", got)
	}
	if got := judgerRequestName(LeaseRequest{}); got != "local-judger" {
		t.Fatalf("anonymous name = %q", got)
	}
}

func seedTaskData(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Setenv("STORAGE", t.TempDir())
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	for key, content := range map[string]string{
		"problems/1000/data/1.in":  "1 2\n",
		"problems/1000/data/1.out": "3\n",
	} {
		if err := store.Put(t.Context(), key, strings.NewReader(content), int64(len(content)), "text/plain"); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}
	if err := db.Create(&models.Language{ID: "cpp", Name: "C++", Source: "main.cc", Dockerfile: "FROM gcc"}).Error; err != nil {
		t.Fatalf("create language: %v", err)
	}
	if err := db.Create(&models.Problem{ID: 1000, Title: "A+B", Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
}

func judgerJSON(e *echo.Echo, target string, token string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func TestSafeTaskAssetZipNameRejectsUnsafeNames(t *testing.T) {
	name, ok := safeTaskAssetZipName("judge", "custom/main.cc")
	if !ok || name != "judge/custom/main.cc" {
		t.Fatalf("safe task asset name = %q, %v", name, ok)
	}
	for _, unsafe := range []string{"../evil", "custom/../../evil", "/absolute", `custom\..\evil`, "custom//main.cc"} {
		if name, ok := safeTaskAssetZipName("judge", unsafe); ok {
			t.Fatalf("unsafe task asset name %q accepted as %q", unsafe, name)
		}
	}
}

func newJudgerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	startRedis(t)
	utils.ResetCacheForTest()
	t.Cleanup(utils.ResetCacheForTest)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "judger.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}
	return db
}

func startRedis(t *testing.T) {
	t.Helper()
	server := miniredis.RunT(t)
	t.Setenv("REDIS", "redis://"+server.Addr()+"/0")
}

func remoteJudgerContext(t *testing.T, api *API, token string) echo.Context {
	t.Helper()
	c, err := remoteJudgerContextMaybe(api, token)
	if err != nil {
		t.Fatalf("auth token %q: %v", token, err)
	}
	return c
}

func remoteJudgerContextAllowError(t *testing.T, api *API, token string) error {
	t.Helper()
	_, err := remoteJudgerContextMaybe(api, token)
	return err
}

func remoteJudgerContextMaybe(api *API, token string) (echo.Context, error) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/judger/lease", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("Authorization", "Bearer "+token)
	c := e.NewContext(req, httptest.NewRecorder())
	return c, api.auth(func(echo.Context) error { return nil })(c)
}

func localJudgerContext(t *testing.T, api *API) echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/judger/lease", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	c := e.NewContext(req, httptest.NewRecorder())
	if err := api.auth(func(echo.Context) error { return nil })(c); err != nil {
		t.Fatalf("auth local judger: %v", err)
	}
	return c
}

func expectHTTPStatus(t *testing.T, err error, status int) {
	t.Helper()
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != status {
		t.Fatalf("error = %#v, want HTTP %d", err, status)
	}
}
