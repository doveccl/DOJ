package judger

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

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

func TestCasePayloadsFromCompatNames(t *testing.T) {
	cases := casePayloadsFromObjects(1000, []utils.ObjectInfo{
		{Key: "problems/1000/data/in1.txt", Size: 4},
		{Key: "problems/1000/data/ans1.txt", Size: 2},
		{Key: "problems/1000/data/in2.txt", Size: 4},
	})
	if len(cases) != 1 {
		t.Fatalf("cases = %+v", cases)
	}
	if cases[0].Input != "data/in1.txt" || cases[0].Answer != "data/ans1.txt" || cases[0].Score != 100 {
		t.Fatalf("compat case = %+v", cases[0])
	}
}

func TestValidateResult(t *testing.T) {
	valid := ResultRequest{
		SubmissionID: 1,
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
	}{
		{name: "missing target", req: ResultRequest{Status: "AC", Score: 100}},
		{name: "bad status", req: ResultRequest{SubmissionID: 1, Status: "queued", Score: 0}},
		{name: "negative score", req: ResultRequest{SubmissionID: 1, Status: "WA", Score: -1}},
		{name: "large score", req: ResultRequest{SubmissionID: 1, Status: "WA", Score: 101}},
		{name: "bad case no", req: ResultRequest{SubmissionID: 1, Status: "WA", Score: 0, Cases: []CaseResult{{No: 0, Status: "WA"}}}},
		{name: "bad case status", req: ResultRequest{SubmissionID: 1, Status: "WA", Score: 0, Cases: []CaseResult{{No: 1, Status: "queued"}}}},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			err := validateResult(item.req)
			if err == nil {
				t.Fatal("expected validation error")
			}
			expectHTTPStatus(t, err, http.StatusBadRequest)
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
	if err := api.authorizeSubmission(c, 1); err == nil {
		t.Fatal("remote judger without lease should not authorize submission")
	}
	reserveLease(1, remote.ID, time.Now().Add(time.Minute), time.Now())
	if err := api.authorizeSubmission(c, 1); err != nil {
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

func TestJudgerRequestNameFallback(t *testing.T) {
	if got := judgerRequestName(LeaseRequest{Host: " linux-a "}); got != "linux-a" {
		t.Fatalf("name fallback = %q", got)
	}
	if got := judgerRequestName(LeaseRequest{}); got != "local-judger" {
		t.Fatalf("anonymous fallback = %q", got)
	}
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
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "judger.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}
	return db
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
