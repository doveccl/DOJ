package public

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestHomeProblemsUseCompactPayload(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	user := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	solved := models.Problem{ID: 1000, Title: "Solved", Tags: datatypes.JSON([]byte(`["math","beginner"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	unsolved := models.Problem{ID: 1001, Title: "Unsolved", Tags: datatypes.JSON([]byte(`["dp"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&[]models.Problem{solved, unsolved}).Error; err != nil {
		t.Fatalf("create problems: %v", err)
	}
	if err := db.Create(&models.Submission{UserID: user.ID, ProblemID: solved.ID, Language: "cpp", Code: "code", Status: "AC", Score: 100, Public: true}).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	e := echo.New()
	Register(e, db)

	res := requestWithCookies(e, http.MethodGet, "/api/home", databaseSession(t, db, user.ID), nil)
	home := decodeJSON[contract.Home](t, res)
	if len(home.Problems) != 1 || home.Problems[0].ID != unsolved.ID || home.Problems[0].Title != unsolved.Title {
		t.Fatalf("home problems should include unsolved compact identity fields only: %+v", home.Problems)
	}
	var raw struct {
		Problems []map[string]any `json:"problems"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw home: %v", err)
	}
	for _, key := range []string{"tags", "visible", "mode", "timeMs", "memoryMb", "discussions", "mine", "latest", "submission", "ac", "submit"} {
		if _, ok := raw.Problems[0][key]; ok {
			t.Fatalf("home problem should not include list-only field %q: %+v", key, raw.Problems[0])
		}
	}
}

func TestHomeFiltersAssignmentsAndContestsForCurrentUser(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	user := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	other := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}
	problems := []models.Problem{
		{ID: 1000, Title: "A", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1001, Title: "B", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
	}
	if err := db.Create(&problems).Error; err != nil {
		t.Fatalf("create problems: %v", err)
	}
	now := time.Now()
	assigned := models.Assignment{Title: "Assigned", EndAt: now.Add(24 * time.Hour)}
	unassigned := models.Assignment{Title: "Unassigned", EndAt: now.Add(48 * time.Hour)}
	if err := db.Create(&assigned).Error; err != nil {
		t.Fatalf("create assigned assignment: %v", err)
	}
	if err := db.Create(&unassigned).Error; err != nil {
		t.Fatalf("create unassigned assignment: %v", err)
	}
	assignmentProblems := []models.AssignmentProblem{
		{AssignmentID: assigned.ID, ProblemID: problems[0].ID, Sort: "A"},
		{AssignmentID: assigned.ID, ProblemID: problems[1].ID, Sort: "B"},
		{AssignmentID: unassigned.ID, ProblemID: problems[0].ID, Sort: "A"},
	}
	if err := db.Create(&assignmentProblems).Error; err != nil {
		t.Fatalf("create assignment problems: %v", err)
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: assigned.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("assign user: %v", err)
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: unassigned.ID, UserID: other.ID}).Error; err != nil {
		t.Fatalf("assign other: %v", err)
	}
	if err := db.Create(&models.Submission{UserID: user.ID, ProblemID: problems[0].ID, AssignmentID: &assigned.ID, Language: "cpp", Code: "code", Status: "AC", Score: 100, Public: true}).Error; err != nil {
		t.Fatalf("create assignment submission: %v", err)
	}
	running := models.Contest{Title: "Running", Kind: "OI", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour)}
	pending := models.Contest{Title: "Pending", Kind: "OI", StartAt: now.Add(time.Hour), EndAt: now.Add(2 * time.Hour)}
	ended := models.Contest{Title: "Ended", Kind: "OI", StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Hour)}
	if err := db.Create(&running).Error; err != nil {
		t.Fatalf("create running contest: %v", err)
	}
	if err := db.Create(&pending).Error; err != nil {
		t.Fatalf("create pending contest: %v", err)
	}
	if err := db.Create(&ended).Error; err != nil {
		t.Fatalf("create ended contest: %v", err)
	}
	contestProblems := []models.ContestProblem{
		{ContestID: running.ID, ProblemID: problems[0].ID, Sort: "A"},
		{ContestID: pending.ID, ProblemID: problems[0].ID, Sort: "A"},
		{ContestID: ended.ID, ProblemID: problems[0].ID, Sort: "A"},
	}
	if err := db.Create(&contestProblems).Error; err != nil {
		t.Fatalf("create contest problems: %v", err)
	}

	e := echo.New()
	Register(e, db)
	home := decodeJSON[contract.Home](t, requestWithCookies(e, http.MethodGet, "/api/home", databaseSession(t, db, user.ID), nil))
	if len(home.Assignments) != 1 || home.Assignments[0].ID != assigned.ID || home.Assignments[0].Done != 1 || home.Assignments[0].Total != 2 {
		t.Fatalf("home assignments should include assigned progress only: %+v", home.Assignments)
	}
	contestStatuses := map[string]string{}
	for _, contest := range home.Contests {
		contestStatuses[contest.Title] = contest.Status
	}
	if len(contestStatuses) != 2 || contestStatuses["Running"] != "running" || contestStatuses["Pending"] != "pending" {
		t.Fatalf("home contests should include active contests with status only: %+v", home.Contests)
	}
	if home.Contests[0].Title != "Running" || home.Contests[1].Title != "Pending" {
		t.Fatalf("home contests should prioritize running then nearest pending: %+v", home.Contests)
	}
}
