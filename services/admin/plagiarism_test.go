package admin

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
)

func TestPlagiarismPackageUsesScopeRepresentativeAsNewAndACHistoryAsOld(t *testing.T) {
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
		{ID: 1, UserID: 1, ProblemID: 1000, AssignmentID: &assignment.ID, Language: "cpp", Status: "WA", Score: 80, Code: "lower score"},
		{ID: 2, UserID: 2, ProblemID: 1000, Language: "cc", Status: "AC", Score: 100, Code: "old"},
		{ID: 3, UserID: 1, ProblemID: 1000, AssignmentID: &assignment.ID, Language: "cpp", Status: "AC", Score: 100, Code: "first highest"},
		{ID: 4, UserID: 2, ProblemID: 1000, Language: "py", Status: "AC", Score: 100, Code: "python"},
		{ID: 5, UserID: 1, ProblemID: 1000, AssignmentID: &assignment.ID, Language: "cpp", Status: "AC", Score: 100, Code: "same user later tie"},
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
	if count != 3 {
		t.Fatalf("submission count = %d, want 3", count)
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, file := range reader.File {
		names[file.Name] = true
	}
	for _, name := range []string{"new/P1000#3U1(alice).cc", "old/P1000#2U2(bob).cc", "old/P1000#5U1(alice).cc"} {
		if !names[name] {
			t.Fatalf("zip missing %s in %v", name, names)
		}
	}
	for name := range names {
		if strings.Contains(name, "#1U") || strings.Contains(name, "#4U") {
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

func TestPlagiarismNewSubmissionsFollowScopeRules(t *testing.T) {
	db := testAdminDB(t)
	api := &API{db: db}
	oi := models.Contest{Title: "oi", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	icpc := models.Contest{Title: "icpc", Kind: "ICPC", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&oi).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&icpc).Error; err != nil {
		t.Fatal(err)
	}
	rows := []models.Submission{
		{ID: 1, UserID: 1, ProblemID: 1000, Status: "WA", Score: 0},
		{ID: 2, UserID: 1, ProblemID: 1000, Status: "AC", Score: 100},
		{ID: 3, UserID: 1, ProblemID: 1000, Status: "WA", Score: 0},
	}
	got, err := api.plagiarismNewSubmissions(plagiarismScopeContest, oi.ID, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != 3 {
		t.Fatalf("OI representative = %+v, want last submission 3", got)
	}
	got, err = api.plagiarismNewSubmissions(plagiarismScopeContest, icpc.ID, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("ICPC representative = %+v, want first AC 2", got)
	}
}

func TestFilterJPlagReportDropsOnlySameUser(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	mustZipText(t, zw, "topComparisons.json", `[
{"firstSubmission":"new_P1000#1U1(alice).cc","secondSubmission":"new_P1000#2U2(bob).cc"},
{"firstSubmission":"new_P1000#1U1(alice).cc","secondSubmission":"new_P1000#3U1(alice).cc"},
{"firstSubmission":"new_P1000#1U1(alice).cc","secondSubmission":"new_P1001#4U2(bob).cc"}]`)
	mustZipText(t, zw, "runInformation.json", `{"totalComparisons":3}`)
	mustZipText(t, zw, "comparisons/new_P1000#1U1(alice).cc-new_P1000#2U2(bob).cc.json", `{"firstSubmissionId":"new_P1000#1U1(alice).cc","secondSubmissionId":"new_P1000#2U2(bob).cc"}`)
	mustZipText(t, zw, "comparisons/new_P1000#1U1(alice).cc-new_P1000#3U1(alice).cc.json", `{"firstSubmissionId":"new_P1000#1U1(alice).cc","secondSubmissionId":"new_P1000#3U1(alice).cc"}`)
	mustZipText(t, zw, "comparisons/new_P1000#1U1(alice).cc-new_P1001#4U2(bob).cc.json", `{"firstSubmissionId":"new_P1000#1U1(alice).cc","secondSubmissionId":"new_P1001#4U2(bob).cc"}`)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := filterJPlagReport(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(got), int64(len(got)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, file := range reader.File {
		names[file.Name] = true
	}
	if !names["comparisons/new_P1000#1U1(alice).cc-new_P1000#2U2(bob).cc.json"] || !names["comparisons/new_P1000#1U1(alice).cc-new_P1001#4U2(bob).cc.json"] {
		t.Fatalf("kept comparison missing: %v", names)
	}
	if names["comparisons/new_P1000#1U1(alice).cc-new_P1000#3U1(alice).cc.json"] {
		t.Fatalf("invalid comparison kept: %v", names)
	}
}

func mustZipText(t *testing.T, zw *zip.Writer, name string, text string) {
	t.Helper()
	if err := zipText(zw, name, text); err != nil {
		t.Fatal(err)
	}
}

func TestPlagiarismRefFromReferer(t *testing.T) {
	cases := []struct {
		name    string
		host    string
		referer string
		ref     string
		ok      bool
	}{
		{name: "entry", host: "test.local", referer: "https://test.local/?jplag=A9&file=x", ref: "A9", ok: true},
		{name: "viewer route", host: "test.local", referer: "https://test.local/overview?jplag=C10&file=x", ref: "C10", ok: true},
		{name: "foreign host", host: "test.local", referer: "https://evil.local/?jplag=A42", ok: false},
		{name: "numeric job id", host: "test.local", referer: "https://test.local/?jplag=42", ok: false},
		{name: "missing id", host: "test.local", referer: "https://test.local/results.jplag", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/results.jplag", nil)
			req.Host = tc.host
			req.Header.Set("Referer", tc.referer)
			ref, ok := plagiarismRefFromReferer(e.NewContext(req, httptest.NewRecorder()))
			if ref != tc.ref || ok != tc.ok {
				t.Fatalf("ref, ok = %q, %v, want %q, %v", ref, ok, tc.ref, tc.ok)
			}
		})
	}
}

func TestPlagiarismReportURLUsesStorageState(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	ctx := context.Background()
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	state := plagiarismState{ID: "A12", Scope: plagiarismScopeAssignment, ScopeID: 12, Status: plagiarismStatusDone}
	if got := plagiarismReportKey(state.ID); got != "jplag/A12/result.jplag" {
		t.Fatalf("report key = %q", got)
	}
	if got := plagiarismStateKey(state.ID); got != "jplag/A12/meta.json" {
		t.Fatalf("state key = %q", got)
	}
	if dto := plagiarismJobDTO(state, plagiarismReportExists(ctx, store, state.ID)); dto.ReportURL != "" {
		t.Fatalf("reportUrl without storage object = %q", dto.ReportURL)
	}
	if err := store.Put(ctx, plagiarismReportKey(state.ID), strings.NewReader("report"), int64(len("report")), "application/octet-stream"); err != nil {
		t.Fatal(err)
	}
	if dto := plagiarismJobDTO(state, plagiarismReportExists(ctx, store, state.ID)); dto.ReportURL != "/api/admin/A12.jplag" {
		t.Fatalf("reportUrl = %q", dto.ReportURL)
	}
}
