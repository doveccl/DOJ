package web

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	contract "github.com/doveccl/doj/common/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestProblemDiscussionCountsUseTagsWithoutProblemVisibilityCoupling(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	visible := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	hidden := models.Problem{ID: 1001, Title: "Hidden", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&visible).Error; err != nil {
		t.Fatalf("create visible problem: %v", err)
	}
	if err := db.Create(&hidden).Error; err != nil {
		t.Fatalf("create hidden problem: %v", err)
	}
	discussions := []models.Discussion{
		{Title: "Visible only", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1000"]`))},
		{Title: "Mixed hidden", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1000","P1001"]`))},
	}
	if err := db.Create(&discussions).Error; err != nil {
		t.Fatalf("create discussions: %v", err)
	}
	if err := db.Create(&[]models.Submission{
		{UserID: admin.ID, ProblemID: visible.ID, Language: "cpp", Code: "visible", Status: "AC", Score: 100},
		{UserID: admin.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden", Status: "AC", Score: 100},
	}).Error; err != nil {
		t.Fatalf("create submissions: %v", err)
	}
	e := echo.New()
	Register(e, db)

	guestState := decodeJSON[[]contract.ProblemState](t, requestOK(t, e, http.MethodGet, "/api/problem-state?ids=1000,1001", ""))
	if len(guestState) != 2 || guestState[0].AC != 1 || guestState[0].Submit != 1 || guestState[1].AC != 0 || guestState[1].Submit != 0 {
		t.Fatalf("guest stats should hide invisible problems: %+v", guestState)
	}
	if guestState[0].Discussions == nil || *guestState[0].Discussions != 2 || guestState[1].Discussions != nil {
		t.Fatalf("guest discussion count should use soft tags and hide invisible problems: %+v", guestState)
	}
	adminState := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?ids=1000,1001", databaseSession(t, db, admin.ID), nil))
	if len(adminState) != 2 || adminState[0].AC != 1 || adminState[0].Submit != 1 || adminState[1].AC != 1 || adminState[1].Submit != 1 {
		t.Fatalf("admin stats should include hidden problems: %+v", adminState)
	}
	if adminState[0].Discussions == nil || *adminState[0].Discussions != 2 || adminState[1].Discussions == nil || *adminState[1].Discussions != 1 {
		t.Fatalf("admin discussion count should match soft tag count: %+v", adminState)
	}
}

func TestWriteSSE(t *testing.T) {
	var out bytes.Buffer
	if err := writeSSE(&out, "submission", []byte("{\"changed\":\"submission\"}")); err != nil {
		t.Fatalf("write sse: %v", err)
	}
	want := "event: submission\ndata: {\"changed\":\"submission\"}\n\n"
	if out.String() != want {
		t.Fatalf("sse = %q, want %q", out.String(), want)
	}
}

func TestDynamicSelectSuggestionEndpoints(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	users := []models.User{
		{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true},
		{Name: "alice", Mail: "alice@example.com", Auth: "hash"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	admin := users[0]
	problem := models.Problem{ID: 1000, Title: "A+B", Tags: datatypes.JSON([]byte(`["math","beginner"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	otherProblem := models.Problem{ID: 1001, Title: "DP", Tags: datatypes.JSON([]byte(`["dp"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&[]models.Problem{problem, otherProblem}).Error; err != nil {
		t.Fatalf("create problems: %v", err)
	}
	if err := db.Create(&models.Discussion{Title: "General", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["general","P1000"]`))}).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}
	assignment := models.Assignment{Title: "Summer Homework", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if err := db.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create assignment problem: %v", err)
	}
	contest := models.Contest{Title: "Winter Cup", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)

	problemTags := decodeJSON[[]string](t, requestOK(t, e, http.MethodGet, "/api/tags?kind=problem&q=ma", ""))
	if len(problemTags) != 1 || problemTags[0] != "math" {
		t.Fatalf("problem tag suggestions = %+v", problemTags)
	}
	discussionTags := decodeJSON[[]string](t, requestOK(t, e, http.MethodGet, "/api/tags?kind=discussion&q=gen", ""))
	if len(discussionTags) != 1 || discussionTags[0] != "general" {
		t.Fatalf("discussion tag suggestions = %+v", discussionTags)
	}
	userOptions := decodeJSON[[]contract.UserOption](t, requestOK(t, e, http.MethodGet, "/api/users?q=ali", ""))
	if len(userOptions) != 1 || userOptions[0].Name != "alice" {
		t.Fatalf("user suggestions = %+v", userOptions)
	}
	assignments := decodePageItems[contract.Assignment](t, requestWithCookies(e, http.MethodGet, "/api/assignments?q=Summer", cookies, nil))
	if len(assignments) != 1 || assignments[0].Title != "Summer Homework" {
		t.Fatalf("assignment suggestions = %+v", assignments)
	}
	contests := decodePageItems[contract.Contest](t, requestWithCookies(e, http.MethodGet, "/api/contests?q=Winter", cookies, nil))
	if len(contests) != 1 || contests[0].Title != "Winter Cup" {
		t.Fatalf("contest suggestions = %+v", contests)
	}
}
