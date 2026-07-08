package judgeapi

import (
	"archive/zip"
	"bytes"
	"github.com/doveccl/doj/common/storage"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
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

func TestProblemPackageRouteAcceptsZipSuffix(t *testing.T) {
	db := newJudgerTestDB(t)
	storage := t.TempDir()
	t.Setenv("STORAGE", storage)
	if err := db.Create(&models.Problem{ID: 1000, Title: "A+B", Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 64}).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	mustWriteFile(t, filepath.Join(storage, "problems", "1000", "data", "1.in"), "1 2\n")
	mustWriteFile(t, filepath.Join(storage, "problems", "1000", "data", "1.out"), "3\n")

	e := echo.New()
	Register(e, db)
	req := httptest.NewRequest(http.MethodGet, "/api/judger/P1000.zip", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("package status = %d body=%q", res.Code, res.Body.String())
	}
	reader, err := zip.NewReader(bytes.NewReader(res.Body.Bytes()), int64(res.Body.Len()))
	if err != nil {
		t.Fatalf("read package zip: %v", err)
	}
	if !zipHasFile(reader, "data/1.in") || !zipHasFile(reader, "data/1.out") {
		t.Fatalf("package files = %+v", reader.File)
	}
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
