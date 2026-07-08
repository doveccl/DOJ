package judgeapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/doveccl/doj/common/authn"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

func TestAuthAndLeaseAuthorization(t *testing.T) {
	db := newJudgerTestDB(t)
	remote := models.Judger{Name: "linux-a", Auth: authn.TokenHash("token-a")}
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

func TestLocalLoopbackLeaseReusesJudgerWithoutToken(t *testing.T) {
	db := newJudgerTestDB(t)
	api := &API{db: db, leaseWait: time.Millisecond}
	e := echo.New()
	group := e.Group("/api/judger", api.auth)
	group.POST("/lease", api.lease)

	for index := 0; index < 3; index++ {
		req := httptest.NewRequest(http.MethodPost, "/api/judger/lease", strings.NewReader(`{"host":"local-judger"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.RemoteAddr = "127.0.0.1:1234"
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("local lease %d got %d body=%s", index, res.Code, res.Body.String())
		}
	}

	var rows []models.Judger
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("list judgers: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "local-judger" || rows[0].Auth != "" {
		t.Fatalf("local leases should reuse one tokenless judger, rows=%+v", rows)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/judger/lease", strings.NewReader(`{"host":"remote"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.RemoteAddr = "203.0.113.10:1234"
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("remote lease without token got %d body=%s", res.Code, res.Body.String())
	}
}

func TestLocalLoopbackConcurrentEnsureReusesJudgerWithoutToken(t *testing.T) {
	db := newJudgerTestDB(t)
	api := &API{db: db}
	const workers = 4
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := api.ensureJudger(localJudgerContext(t, api), LeaseRequest{Host: "local-judger"})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("local concurrent ensure failed: %v", err)
		}
	}
	var rows []models.Judger
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("list judgers: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "local-judger" || rows[0].Auth != "" {
		t.Fatalf("local concurrent leases should reuse one tokenless judger, rows=%+v", rows)
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

func TestJudgerRequestNameDefault(t *testing.T) {
	if got := judgerRequestName(LeaseRequest{Host: " linux-a "}); got != "linux-a" {
		t.Fatalf("name = %q", got)
	}
	if got := judgerRequestName(LeaseRequest{}); got != "local-judger" {
		t.Fatalf("anonymous name = %q", got)
	}
}
