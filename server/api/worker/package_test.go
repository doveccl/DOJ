package worker

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/auth"
	"github.com/doveccl/doj/server/storage"
	"github.com/labstack/echo/v4"
)

func TestCasePayloadsFromObjects(t *testing.T) {
	cases := casePayloadsFromObjects(1000, []storage.Info{
		{Key: "problems/1000/data/10.in", Size: 4},
		{Key: "problems/1000/data/10.out", Size: 2},
		{Key: "problems/1000/data/2.out", Size: 2},
		{Key: "problems/1000/data/3.in", Size: 4},
		{Key: "problems/1000/data/3.ans", Size: 2},
		{Key: "problems/1000/data/1.in", Size: 4},
		{Key: "problems/1000/data/1.out", Size: 2},
		{Key: "problems/1000/data/2.in", Size: 4},
		{Key: "problems/1000/data/readme.txt", Size: 10},
	})
	if len(cases) != 4 {
		t.Fatalf("cases = %+v", cases)
	}
	if cases[0].ID != "1" || cases[0].Input != "data/1.in" || cases[0].Answer != "data/1.out" || cases[0].Score != 25 {
		t.Fatalf("case 1 = %+v", cases[0])
	}
	if cases[1].ID != "2" || cases[1].Input != "data/2.in" || cases[1].Answer != "data/2.out" || cases[1].Score != 25 {
		t.Fatalf("case 2 = %+v", cases[1])
	}
	if cases[2].ID != "3" || cases[2].Input != "data/3.in" || cases[2].Answer != "data/3.ans" || cases[2].Score != 25 {
		t.Fatalf("case 3 = %+v", cases[2])
	}
	if cases[3].ID != "10" || cases[3].Input != "data/10.in" || cases[3].Answer != "data/10.out" || cases[3].Score != 25 {
		t.Fatalf("case 10 = %+v", cases[3])
	}
}

func TestCasePayloadsFromCommonDataNames(t *testing.T) {
	cases := casePayloadsFromObjects(1000, []storage.Info{
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
	target := "/api/judger/tasks/" + strconv.FormatUint(uint64(payload.ID), 10) + "/package?attempt=" + strconv.Itoa(payload.Attempt) + "&hash=" + payload.Problem.PackageHash
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
	etag := `"` + payload.Problem.PackageHash + `"`
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
	wrongAttempt := "/api/judger/tasks/" + strconv.FormatUint(uint64(payload.ID), 10) + "/package?attempt=" + strconv.Itoa(payload.Attempt+1) + "&hash=" + payload.Problem.PackageHash
	if got := taskPackageRequest(e, wrongAttempt, "token-a", ""); got.Code != http.StatusConflict {
		t.Fatalf("wrong attempt package got %d body=%q", got.Code, got.Body.String())
	}
	if got := localTaskPackageRequest(e, wrongAttempt, ""); got.Code != http.StatusConflict {
		t.Fatalf("local wrong attempt package got %d body=%q", got.Code, got.Body.String())
	}
	if got := taskPackageRequest(e, "/api/judger/P1000.zip", "token-a", ""); got.Code != http.StatusNotFound {
		t.Fatalf("problem enumeration route got %d body=%q", got.Code, got.Body.String())
	}

	mustWriteFile(t, filepath.Join(storage.Root(), "problems", "1000", "data", "1.out"), "changed answer\n")
	if got := taskPackageRequest(e, target, "token-a", ""); got.Code != http.StatusConflict {
		t.Fatalf("changed package got %d body=%q", got.Code, got.Body.String())
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

func TestSafeProblemPackageZipNameRejectsUnsafeNames(t *testing.T) {
	name, ok := safeProblemPackageZipName("judge", "custom/main.cc")
	if !ok || name != "judge/custom/main.cc" {
		t.Fatalf("safe problem package name = %q, %v", name, ok)
	}
	for _, unsafe := range []string{"../evil", "custom/../../evil", "/absolute", `custom\..\evil`, "custom//main.cc"} {
		if name, ok := safeProblemPackageZipName("judge", unsafe); ok {
			t.Fatalf("unsafe problem package name %q accepted as %q", unsafe, name)
		}
	}
}
