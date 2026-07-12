package public

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/doveccl/doj/contract/limits"
	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestProblemLimitsHaveUpperBounds(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	tooLarge := `{"title":"Huge","timeMs":` + strconv.Itoa(limits.MaxProblemTimeMS+1) + `,"memoryMb":` + strconv.Itoa(limits.MaxProblemMemoryMB+1) + `}`
	if res := requestJSONWithCookies(e, http.MethodPost, "/api/problems", cookies, tooLarge); res.Code != http.StatusBadRequest {
		t.Fatalf("oversized create got %d body=%s", res.Code, res.Body.String())
	}
	valid := `{"title":"Bounded","timeMs":` + strconv.Itoa(limits.MaxProblemTimeMS) + `,"memoryMb":` + strconv.Itoa(limits.MaxProblemMemoryMB) + `}`
	res := requestJSONWithCookies(e, http.MethodPost, "/api/problems", cookies, valid)
	if res.Code != http.StatusCreated {
		t.Fatalf("maximum limits got %d body=%s", res.Code, res.Body.String())
	}
	id := decodeJSON[contract.CreatedID](t, res).ID
	patch := `{"memoryMb":` + strconv.Itoa(limits.MaxProblemMemoryMB+1) + `}`
	if res := requestJSONWithCookies(e, http.MethodPatch, "/api/problems/"+strconv.FormatUint(uint64(id), 10), cookies, patch); res.Code != http.StatusBadRequest {
		t.Fatalf("oversized patch got %d body=%s", res.Code, res.Body.String())
	}
}

func TestPublishedProblemCanBeHiddenWithoutTouchingStatementStorage(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	if err := db.Create(&models.Submission{ProblemID: problem.ID, UserID: admin.ID, Language: "cpp", Code: "int main(){}", Status: "AC", Score: 100, Public: true}).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	discussionTags, _ := json.Marshal([]string{"P1000"})
	if err := db.Create(&models.Discussion{Title: "Visible discussion", Content: "body", UserID: admin.ID, Tags: discussionTags}).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}
	blockedStorage := filepath.Join(t.TempDir(), "storage-file")
	if err := os.WriteFile(blockedStorage, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("create blocked storage marker: %v", err)
	}
	t.Setenv("STORAGE", blockedStorage)

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	res := requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000/visibility", cookies, `{"visible":false}`)
	if res.Code != http.StatusOK {
		t.Fatalf("visibility update got %d body=%s", res.Code, res.Body.String())
	}
	var updated models.Problem
	if err := db.First(&updated, problem.ID).Error; err != nil || updated.Visible {
		t.Fatalf("published problem visibility changed: %+v err=%v", updated, err)
	}
}

func TestProblemPatchOnlyUpdatesProvidedFields(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Original", Tags: datatypes.JSON([]byte(`["old"]`)), Visible: true, Mode: "default", TimeMS: 2000, MemoryMB: 512}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	t.Setenv("STORAGE", t.TempDir())

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	statement := "# Original\n\nBody"
	statementBody := `{"statement":` + strconv.Quote(statement) + `}`
	if res := requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000", cookies, statementBody); res.Code != http.StatusOK {
		t.Fatalf("statement patch got %d body=%s", res.Code, res.Body.String())
	} else if got := decodeJSON[contract.CreatedID](t, res); got.ID != problem.ID {
		t.Fatalf("statement patch should return problem id: %+v", got)
	}
	res := requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000", cookies, `{"mode":"strict"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("mode patch got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[contract.CreatedID](t, res); got.ID != problem.ID {
		t.Fatalf("mode patch should return problem id: %+v", got)
	}
	updated := decodeJSON[contract.Problem](t, requestWithCookies(e, http.MethodGet, "/api/problems/1000", cookies, nil))
	if updated.Mode != "strict" || updated.Title != "Original" || updated.Statement != statement || !updated.Visible || updated.TimeMS != 2000 || updated.MemoryMB != 512 {
		t.Fatalf("mode patch should preserve unrelated problem fields: %+v", updated)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000/visibility", cookies, `{"visible":false}`)
	if res.Code != http.StatusOK {
		t.Fatalf("visible false patch got %d body=%s", res.Code, res.Body.String())
	}
	updated = decodeJSON[contract.Problem](t, requestWithCookies(e, http.MethodGet, "/api/problems/1000", cookies, nil))
	if updated.Visible || updated.Mode != "strict" || updated.Title != "Original" || updated.Statement != statement {
		t.Fatalf("unpublish should preserve unrelated fields: %+v", updated)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000", cookies, `{"tags":[]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("empty tags patch got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[contract.CreatedID](t, res); got.ID != problem.ID {
		t.Fatalf("tags patch should return problem id: %+v", got)
	}
	updated = decodeJSON[contract.Problem](t, requestWithCookies(e, http.MethodGet, "/api/problems/1000", cookies, nil))
	if len(updated.Tags) != 0 || updated.Visible || updated.Mode != "strict" || updated.Statement != statement {
		t.Fatalf("empty tags patch should clear tags and preserve unrelated fields: %+v", updated)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000", cookies, `{"timeMs":0,"memoryMb":0}`)
	if res.Code != http.StatusOK {
		t.Fatalf("zero limit patch got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[contract.CreatedID](t, res); got.ID != problem.ID {
		t.Fatalf("limit patch should return problem id: %+v", got)
	}
	updated = decodeJSON[contract.Problem](t, requestWithCookies(e, http.MethodGet, "/api/problems/1000", cookies, nil))
	if updated.TimeMS != 1000 || updated.MemoryMB != 256 || updated.Mode != "strict" || updated.Statement != statement {
		t.Fatalf("zero limit patch should use defaults and preserve unrelated fields: %+v", updated)
	}
}

func TestProblemCreateDefaultsHiddenAndListSortsByID(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	rows := []models.Problem{
		{ID: 1002, Title: "B", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1000, Title: "A", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create problem %d: %v", row.ID, err)
		}
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	created := decodeJSON[contract.CreatedID](t, requestJSONWithCookies(e, http.MethodPost, "/api/problems", cookies, `{"title":"Created","tags":[],"mode":"default","timeMs":1000,"memoryMb":256}`))
	createdDetail := decodeJSON[contract.Problem](t, requestWithCookies(e, http.MethodGet, "/api/problems/"+strconv.FormatUint(uint64(created.ID), 10), cookies, nil))
	if createdDetail.Visible {
		t.Fatalf("unfinished problem should default to hidden: %+v", createdDetail)
	}
	items := decodePageItems[contract.Problem](t, requestWithCookies(e, http.MethodGet, "/api/problems", cookies, nil))
	if len(items) < 3 {
		t.Fatalf("problem list too short: %+v", items)
	}
	ids := []uint{items[0].ID, items[1].ID, items[2].ID}
	if ids[0] != 1000 || ids[1] != 1002 || ids[2] != created.ID {
		t.Fatalf("problem list should sort by id asc, got %+v", ids)
	}
}

func TestAdminCanPublishDraftProblem(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Setenv("STORAGE", root)
	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	created := decodeJSON[contract.CreatedID](t, requestJSONWithCookies(e, http.MethodPost, "/api/problems", cookies, `{"title":"Draft","tags":[],"mode":"default","timeMs":1000,"memoryMb":256}`))
	visibilityPath := "/api/problems/" + strconv.FormatUint(uint64(created.ID), 10) + "/visibility"
	if res := requestJSONWithCookies(e, http.MethodPatch, visibilityPath, cookies, `{"visible":true}`); res.Code != http.StatusOK {
		t.Fatalf("empty draft publish got %d body=%s", res.Code, res.Body.String())
	}
	files := map[string]string{
		"statement.md": "# Draft\n\nAdd two integers.",
		"data/1.in":    "1 2\n",
		"data/1.out":   "3\n",
	}
	for name, content := range files {
		file := filepath.Join(root, "problems", strconv.FormatUint(uint64(created.ID), 10), filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if res := requestJSONWithCookies(e, http.MethodPatch, visibilityPath, cookies, `{"visible":true}`); res.Code != http.StatusOK {
		t.Fatalf("ready problem publish got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestJSONWithCookies(e, http.MethodPatch, visibilityPath, cookies, `{"visible":false}`); res.Code != http.StatusOK {
		if res.Code != http.StatusConflict {
			t.Fatalf("published problem hide got %d body=%s", res.Code, res.Body.String())
		}
	}
	custom := decodeJSON[contract.CreatedID](t, requestJSONWithCookies(e, http.MethodPost, "/api/problems", cookies, `{"title":"Custom Draft","tags":[],"mode":"custom","timeMs":1000,"memoryMb":256}`))
	customFiles := map[string]string{
		"statement.md": "# Custom Draft\n\nJudge interactively.",
		"data/1.in":    "1\n",
		"data/1.out":   "1\n",
	}
	for name, content := range customFiles {
		file := filepath.Join(root, "problems", strconv.FormatUint(uint64(custom.ID), 10), filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	customVisibilityPath := "/api/problems/" + strconv.FormatUint(uint64(custom.ID), 10) + "/visibility"
	if res := requestJSONWithCookies(e, http.MethodPatch, customVisibilityPath, cookies, `{"visible":true}`); res.Code != http.StatusOK {
		t.Fatalf("custom problem without Dockerfile got %d body=%s", res.Code, res.Body.String())
	}
}

func TestAdminCanDeleteProblemsRegardlessOfUse(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	problems := []models.Problem{
		{ID: 1000, Title: "Published", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1001, Title: "Contest", Tags: datatypes.JSON([]byte(`[]`)), Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1002, Title: "Assignment", Tags: datatypes.JSON([]byte(`[]`)), Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1003, Title: "Submission", Tags: datatypes.JSON([]byte(`[]`)), Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1004, Title: "Discussion", Tags: datatypes.JSON([]byte(`[]`)), Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1005, Title: "Unused draft", Tags: datatypes.JSON([]byte(`[]`)), Mode: "default", TimeMS: 1000, MemoryMB: 256},
	}
	if err := db.Create(&problems).Error; err != nil {
		t.Fatal(err)
	}
	contest := models.Contest{Title: "Round", Kind: "OI", StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)}
	assignment := models.Assignment{Title: "Homework", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: 1001, Sort: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: 1002, Sort: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Submission{UserID: admin.ID, ProblemID: 1003, Language: "cpp", Code: "int main(){}", Status: "AC", Score: 100}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Discussion{Title: "Discussion", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1004"]`))}).Error; err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	for id := uint(1000); id <= 1004; id++ {
		res := requestWithCookies(e, http.MethodDelete, "/api/problems/"+strconv.FormatUint(uint64(id), 10), cookies, nil)
		if res.Code != http.StatusNoContent {
			t.Fatalf("delete P%d got %d body=%s", id, res.Code, res.Body.String())
		}
	}
	res := requestWithCookies(e, http.MethodDelete, "/api/problems/1005", cookies, nil)
	if res.Code != http.StatusNoContent {
		t.Fatalf("delete unused draft got %d body=%s", res.Code, res.Body.String())
	}
	var count int64
	if err := db.Model(&models.Problem{}).Where("id = ?", 1005).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("unused draft still exists: count=%d err=%v", count, err)
	}
}

func TestProblemListDoesNotTouchAssetStorage(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	blockedStorage := filepath.Join(t.TempDir(), "storage-file")
	if err := os.WriteFile(blockedStorage, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("create blocked storage marker: %v", err)
	}
	t.Setenv("STORAGE", blockedStorage)

	e := echo.New()
	Register(e, db)
	res := requestOK(t, e, http.MethodGet, "/api/problems", "")
	items := decodePageItems[contract.Problem](t, res)
	if len(items) != 1 || items[0].ID != problem.ID {
		t.Fatalf("problem list got %+v, want P%d", items, problem.ID)
	}
	if items[0].Cases != nil || items[0].DataBytes != nil {
		t.Fatalf("problem list should not compute storage-derived stats: %+v", items[0])
	}
}

func TestProblemListSearchesByCode(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	rows := []models.Problem{
		{ID: 1288, Title: "Window Median", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1289, Title: "Deer Tower", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create problem %d: %v", row.ID, err)
		}
	}

	e := echo.New()
	Register(e, db)

	for _, q := range []string{"1289", "P1289"} {
		got := decodePageItems[contract.Problem](t, requestOK(t, e, http.MethodGet, "/api/problems?q="+url.QueryEscape(q), ""))
		if len(got) != 1 || got[0].ID != 1289 {
			t.Fatalf("problem search %q got %+v, want P1289", q, got)
		}
	}
}

func TestProblemListFiltersByViewerStatus(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	user := models.User{Name: "solver", Mail: "solver@example.com", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	problems := []models.Problem{
		{ID: 1000, Title: "Solved", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1001, Title: "Tried", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1002, Title: "Untouched", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
	}
	if err := db.Create(&problems).Error; err != nil {
		t.Fatalf("create problems: %v", err)
	}
	if err := db.Create(&models.Submission{ProblemID: 1000, UserID: user.ID, Language: "cpp", Code: "int main(){}", Status: "AC", Score: 100}).Error; err != nil {
		t.Fatalf("create AC submission: %v", err)
	}
	if err := db.Create(&models.Submission{ProblemID: 1001, UserID: user.ID, Language: "cpp", Code: "int main(){}", Status: "WA", Score: 0}).Error; err != nil {
		t.Fatalf("create WA submission: %v", err)
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, user.ID)

	statusIDs := func(status string) []uint {
		res := requestWithCookies(e, http.MethodGet, "/api/problems?status="+status, cookies, nil)
		if res.Code != http.StatusOK {
			t.Fatalf("status=%s got %d body=%s", status, res.Code, res.Body.String())
		}
		items := decodePageItems[contract.Problem](t, res)
		ids := make([]uint, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		return ids
	}
	if got := statusIDs("ac"); len(got) != 1 || got[0] != 1000 {
		t.Fatalf("status=ac got %+v, want [1000]", got)
	}
	if got := statusIDs("tried"); len(got) != 1 || got[0] != 1001 {
		t.Fatalf("status=tried got %+v, want [1001]", got)
	}
	if got := statusIDs("none"); len(got) != 1 || got[0] != 1002 {
		t.Fatalf("status=none got %+v, want [1002]", got)
	}
	if got := statusIDs("all"); len(got) != 3 {
		t.Fatalf("status=all got %+v, want 3 problems", got)
	}

	// A guest must not be able to filter by another viewer's solve state.
	guest := decodePageItems[contract.Problem](t, requestOK(t, e, http.MethodGet, "/api/problems?status=ac", ""))
	if len(guest) != 3 {
		t.Fatalf("guest status filter should be ignored, got %+v", guest)
	}
}

func TestProblemStatusFilterHidesRunningOIContestAC(t *testing.T) {
	db := testWebDB(t)
	user := models.User{Name: "player", Mail: "player@example.com", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Shared Problem", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	contest := models.Contest{Title: "Running OI", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	contestID := contest.ID
	// A normal-context non-AC attempt, plus an AC inside the running OI contest
	// whose result is hidden. Only the normal attempt may drive list status, so
	// the problem must read as "tried", never "ac".
	if err := db.Create(&models.Submission{ProblemID: problem.ID, UserID: user.ID, Language: "cpp", Code: "int main(){}", Status: "WA", Score: 0}).Error; err != nil {
		t.Fatalf("create normal WA: %v", err)
	}
	if err := db.Create(&models.Submission{ProblemID: problem.ID, UserID: user.ID, ContestID: &contestID, Language: "cpp", Code: "int main(){}", Status: "AC", Score: 100}).Error; err != nil {
		t.Fatalf("create contest AC: %v", err)
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, user.ID)

	acRes := requestWithCookies(e, http.MethodGet, "/api/problems?status=ac", cookies, nil)
	if acRes.Code != http.StatusOK {
		t.Fatalf("status=ac got %d body=%s", acRes.Code, acRes.Body.String())
	}
	if items := decodePageItems[contract.Problem](t, acRes); len(items) != 0 {
		t.Fatalf("hidden contest AC must not count as solved, got %+v", items)
	}
	triedRes := requestWithCookies(e, http.MethodGet, "/api/problems?status=tried", cookies, nil)
	if triedRes.Code != http.StatusOK {
		t.Fatalf("status=tried got %d body=%s", triedRes.Code, triedRes.Body.String())
	}
	if items := decodePageItems[contract.Problem](t, triedRes); len(items) != 1 || items[0].ID != problem.ID {
		t.Fatalf("normal-context attempt should read as tried, got %+v", items)
	}
}
