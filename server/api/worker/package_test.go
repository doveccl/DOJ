package worker

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/auth"
	problemdata "github.com/doveccl/doj/server/problem"
	"github.com/labstack/echo/v4"
)

func TestTaskPackageRequiresCurrentOwnedLeaseAndHash(t *testing.T) {
	db := newJudgerTestDB(t)
	seedTaskData(t, db)
	judgerA := models.Judger{Name: "linux-a", Auth: auth.TokenHash("token-a")}
	judgerB := models.Judger{Name: "linux-b", Auth: auth.TokenHash("token-b")}
	for _, row := range []*models.Judger{&judgerA, &judgerB} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create judger: %v", err)
		}
	}
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "queued"}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	payload, err := (&API{db: db}).tryLease(t.Context(), judgerA.ID)
	if err != nil || payload == nil {
		t.Fatalf("lease task: payload=%+v err=%v", payload, err)
	}
	if payload.Limits.FileKB != 256<<10 {
		t.Fatalf("file limit = %d KB, want %d KB", payload.Limits.FileKB, 256<<10)
	}

	e := echo.New()
	Register(e, db)
	target := "/api/judger/tasks/" + strconv.FormatUint(uint64(payload.ID), 10) + "/package?attempt=" + strconv.Itoa(payload.Attempt) + "&hash=" + payload.Problem.Hash
	res := taskPackageRequest(e, target, "token-a", "")
	if res.Code != http.StatusOK {
		t.Fatalf("package status = %d body=%q", res.Code, res.Body.String())
	}
	if cache := res.Header().Get(echo.HeaderCacheControl); cache != "private, no-store" {
		t.Fatalf("package cache header = %q", cache)
	}
	reader, err := zip.NewReader(bytes.NewReader(res.Body.Bytes()), int64(res.Body.Len()))
	if err != nil {
		t.Fatalf("read package zip: %v", err)
	}
	if !zipHasFile(reader, "data/1.in") || !zipHasFile(reader, "data/1.out") {
		t.Fatalf("package files = %+v", reader.File)
	}
	etag := `"` + payload.Problem.Hash + `"`
	if res.Header().Get("ETag") != etag {
		t.Fatalf("package ETag = %q", res.Header().Get("ETag"))
	}
	if got := taskPackageRequest(e, target, "token-a", etag); got.Code != http.StatusNotModified {
		t.Fatalf("conditional package got %d body=%q", got.Code, got.Body.String())
	}
	if got := localTaskPackageRequest(e, target, etag); got.Code != http.StatusNotModified {
		t.Fatalf("local conditional package got %d body=%q", got.Code, got.Body.String())
	}
	if got := taskPackageRequest(e, target, "token-b", ""); got.Code != http.StatusUnauthorized {
		t.Fatalf("other worker package got %d body=%q", got.Code, got.Body.String())
	}
	wrongAttempt := "/api/judger/tasks/" + strconv.FormatUint(uint64(payload.ID), 10) + "/package?attempt=" + strconv.Itoa(payload.Attempt+1) + "&hash=" + payload.Problem.Hash
	if got := taskPackageRequest(e, wrongAttempt, "token-a", ""); got.Code != http.StatusConflict {
		t.Fatalf("wrong attempt package got %d body=%q", got.Code, got.Body.String())
	}
	if got := localTaskPackageRequest(e, wrongAttempt, ""); got.Code != http.StatusConflict {
		t.Fatalf("local wrong attempt package got %d body=%q", got.Code, got.Body.String())
	}
	if got := taskPackageRequest(e, "/api/judger/P1000.zip", "token-a", ""); got.Code != http.StatusNotFound {
		t.Fatalf("problem enumeration route got %d body=%q", got.Code, got.Body.String())
	}

	var changed models.Problem
	if err := db.First(&changed, 1000).Error; err != nil {
		t.Fatal(err)
	}
	item, err := problemdata.Parse(changed.Package)
	if err != nil {
		t.Fatal(err)
	}
	score := 20
	item.Cases[0].Score = &score
	raw, _ := item.JSON()
	if err := db.Model(&models.Problem{}).Where("id = ?", 1000).Update("package", raw).Error; err != nil {
		t.Fatal(err)
	}
	if got := taskPackageRequest(e, target, "token-a", ""); got.Code != http.StatusOK {
		t.Fatalf("leased package snapshot got %d body=%q", got.Code, got.Body.String())
	}
	if err := db.Model(&models.Submission{}).Where("id = ?", submission.ID).Update("attempt", payload.Attempt+1).Error; err != nil {
		t.Fatalf("advance attempt: %v", err)
	}
	if got := taskPackageRequest(e, target, "token-a", ""); got.Code != http.StatusConflict {
		t.Fatalf("old attempt package got %d body=%q", got.Code, got.Body.String())
	}
	if err := db.Model(&models.Submission{}).Where("id = ?", submission.ID).Updates(map[string]any{"attempt": payload.Attempt, "lease_until": time.Now().Add(-time.Second)}).Error; err != nil {
		t.Fatalf("expire lease: %v", err)
	}
	if got := taskPackageRequest(e, target, "token-a", ""); got.Code != http.StatusConflict {
		t.Fatalf("expired package lease got %d body=%q", got.Code, got.Body.String())
	}
}

func taskPackageRequest(e *echo.Echo, target string, token string, etag string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("Authorization", "Bearer "+token)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func localTaskPackageRequest(e *echo.Echo, target string, etag string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.RemoteAddr = "127.0.0.1:1234"
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}
