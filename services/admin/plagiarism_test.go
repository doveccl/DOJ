package admin

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

func TestPlagiarismPackageUsesScopeAsNewAndProblemHistoryAsOld(t *testing.T) {
	db := testAdminDB(t)
	api := &API{db: db}
	assignment := models.Assignment{Title: "hw", EndAt: time.Now().Add(time.Hour)}
	rows := []any{
		&models.User{ID: 1, Name: "alice", Mail: "a@example.com", Auth: "x"},
		&models.User{ID: 2, Name: "bob", Mail: "b@example.com", Auth: "x"},
		&models.Problem{ID: 1000, Title: "A"},
		&assignment,
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: 1000}).Error; err != nil {
		t.Fatal(err)
	}
	submissions := []models.Submission{
		{ID: 1, UserID: 1, ProblemID: 1000, AssignmentID: &assignment.ID, Language: "cpp", Status: "AC", Code: "new"},
		{ID: 2, UserID: 2, ProblemID: 1000, Language: "cpp", Status: "AC", Code: "old"},
		{ID: 3, UserID: 1, ProblemID: 1000, Language: "cpp", Status: "AC", Code: "same user"},
		{ID: 4, UserID: 2, ProblemID: 1000, Language: "py", Status: "AC", Code: "python"},
	}
	for _, row := range submissions {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	payload, count, err := api.plagiarismPackage(plagiarismScopeAssignment, assignment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("submission count = %d, want 2", count)
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, file := range reader.File {
		names[file.Name] = true
	}
	for _, name := range []string{"new/p1000_sub1_user1.cpp", "old/p1000_sub2_user2.cpp", "manifest.json"} {
		if !names[name] {
			t.Fatalf("zip missing %s in %v", name, names)
		}
	}
	for name := range names {
		if strings.Contains(name, "sub3") || strings.Contains(name, "sub4") {
			t.Fatalf("unexpected file %s in zip", name)
		}
	}
}

func TestJPlagFailureUnwrapsEchoJSONAndDropsVersionWarning(t *testing.T) {
	got := jplagFailure([]byte(`{"message":"exit status 1: 2026-06-30 [WARN] JPlagVersionChecker - newer\n2026-06-30 [ERROR] CLI - Not enough valid submissions"}`))
	if strings.Contains(got, "JPlagVersionChecker") || strings.Contains(got, `{"message"`) {
		t.Fatalf("failure message was not cleaned: %q", got)
	}
	if !strings.Contains(got, "Not enough valid submissions") {
		t.Fatalf("failure message lost useful error: %q", got)
	}
}

func TestProxyPlagiarismDropsCookie(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie := r.Header.Get(echo.HeaderCookie); cookie != "" {
			t.Fatalf("proxied cookie = %q", cookie)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	t.Setenv("JPLAG", upstream.URL)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/JPlag/?file=x", nil)
	req.Header.Set(echo.HeaderCookie, "doj_session=secret")
	rec := httptest.NewRecorder()
	if err := (&API{}).proxyPlagiarism(e.NewContext(req, rec), "JPlag/"); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
