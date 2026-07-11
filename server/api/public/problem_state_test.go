package public

import (
	"bytes"
	"net/http"
	"strconv"
	"testing"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestProblemDiscussionCountsUseExplicitProblemReference(t *testing.T) {
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
		{Title: "Visible again", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1000","general"]`))},
		{Title: "Hidden", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1001"]`))},
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
		t.Fatalf("guest discussion count should hide invisible problems: %+v", guestState)
	}
	adminState := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?ids=1000,1001", databaseSession(t, db, admin.ID), nil))
	if len(adminState) != 2 || adminState[0].AC != 1 || adminState[0].Submit != 1 || adminState[1].AC != 1 || adminState[1].Submit != 1 {
		t.Fatalf("admin stats should include hidden problems: %+v", adminState)
	}
	if adminState[0].Discussions == nil || *adminState[0].Discussions != 2 || adminState[1].Discussions == nil || *adminState[1].Discussions != 1 {
		t.Fatalf("admin discussion count should match explicit problem references: %+v", adminState)
	}
}

func TestProblemPassRateCountsAcceptedSubmissions(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	user := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	problem := models.Problem{ID: 1000, Title: "A+B", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatal(err)
	}
	rows := []models.Submission{
		{UserID: user.ID, ProblemID: problem.ID, Language: "cpp", Code: "a", Status: "AC", Score: 100},
		{UserID: user.ID, ProblemID: problem.ID, Language: "cpp", Code: "b", Status: "WA"},
		{UserID: user.ID, ProblemID: problem.ID, Language: "cpp", Code: "c", Status: "AC", Score: 100},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	Register(e, db)
	state := decodeJSON[[]contract.ProblemState](t, requestOK(t, e, http.MethodGet, "/api/problem-state?ids=1000", ""))
	if len(state) != 1 || state[0].AC != 2 || state[0].Submit != 3 {
		t.Fatalf("pass rate must compare accepted submissions with all submissions: %+v", state)
	}
}

func TestProblemStateStatsAreScopedAndAccessControlled(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	peer := models.User{Name: "peer", Mail: "peer@example.com", Auth: "hash"}
	outsider := models.User{Name: "outsider", Mail: "outsider@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&[]*models.User{&owner, &peer, &outsider, &admin}).Error; err != nil {
		t.Fatal(err)
	}
	problem := models.Problem{ID: 1000, Title: "Scoped", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatal(err)
	}
	assignment := models.Assignment{Title: "Private", EndAt: time.Now().Add(time.Hour)}
	otherAssignment := models.Assignment{Title: "Other", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&[]*models.Assignment{&assignment, &otherAssignment}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]models.AssignmentProblem{
		{AssignmentID: assignment.ID, ProblemID: problem.ID, Sort: "A"},
		{AssignmentID: otherAssignment.ID, ProblemID: problem.ID, Sort: "A"},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]models.AssignmentUser{
		{AssignmentID: assignment.ID, UserID: owner.ID},
		{AssignmentID: assignment.ID, UserID: peer.ID},
		{AssignmentID: otherAssignment.ID, UserID: outsider.ID},
	}).Error; err != nil {
		t.Fatal(err)
	}
	assignmentID := assignment.ID
	otherAssignmentID := otherAssignment.ID
	practiceOwner := models.Submission{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "practice owner", Status: "WA"}
	practiceOutsider := models.Submission{UserID: outsider.ID, ProblemID: problem.ID, Language: "cpp", Code: "practice outsider", Status: "AC", Score: 100}
	assignmentOwner := models.Submission{UserID: owner.ID, ProblemID: problem.ID, AssignmentID: &assignmentID, Language: "cpp", Code: "assignment owner", Status: "AC", Score: 100}
	assignmentPeer := models.Submission{UserID: peer.ID, ProblemID: problem.ID, AssignmentID: &assignmentID, Language: "cpp", Code: "assignment peer", Status: "AC", Score: 100}
	otherContext := models.Submission{UserID: outsider.ID, ProblemID: problem.ID, AssignmentID: &otherAssignmentID, Language: "cpp", Code: "other assignment", Status: "AC", Score: 100}
	if err := db.Create(&[]*models.Submission{&practiceOwner, &practiceOutsider, &assignmentOwner, &assignmentPeer, &otherContext}).Error; err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	Register(e, db)
	practicePath := "/api/problem-state?ids=1000"
	guest := decodeJSON[[]contract.ProblemState](t, requestOK(t, e, http.MethodGet, practicePath, ""))[0]
	if guest.Submit != 2 || guest.AC != 1 || guest.Status != "none" {
		t.Fatalf("guest practice stats mixed private contexts: %+v", guest)
	}
	outsiderPractice := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, practicePath, databaseSession(t, db, outsider.ID), nil))[0]
	if outsiderPractice.Submit != 2 || outsiderPractice.AC != 1 || outsiderPractice.Status != "ac" || outsiderPractice.Submission == nil || outsiderPractice.Submission.ID != practiceOutsider.ID {
		t.Fatalf("outsider practice state mixed assignment context: %+v", outsiderPractice)
	}
	ownerPractice := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, practicePath, databaseSession(t, db, owner.ID), nil))[0]
	if ownerPractice.Submit != 2 || ownerPractice.AC != 1 || ownerPractice.Status != "tried" || ownerPractice.Submission == nil || ownerPractice.Submission.ID != practiceOwner.ID {
		t.Fatalf("owner practice state mixed assignment context: %+v", ownerPractice)
	}
	adminPractice := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, practicePath, databaseSession(t, db, admin.ID), nil))[0]
	if adminPractice.Submit != 2 || adminPractice.AC != 1 {
		t.Fatalf("admin practice stats mixed contextual submissions: %+v", adminPractice)
	}

	assignmentPath := "/api/problem-state?assignment=" + strconv.FormatUint(uint64(assignment.ID), 10) + "&ids=1000"
	if res := request(e, http.MethodGet, assignmentPath, "", nil); res.Code != http.StatusNotFound {
		t.Fatalf("guest queried private assignment stats: %d %s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, assignmentPath, databaseSession(t, db, outsider.ID), nil); res.Code != http.StatusNotFound {
		t.Fatalf("outsider queried private assignment stats: %d %s", res.Code, res.Body.String())
	}
	ownerAssignment := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, assignmentPath, databaseSession(t, db, owner.ID), nil))[0]
	if ownerAssignment.Submit != 1 || ownerAssignment.AC != 1 || ownerAssignment.Status != "ac" || ownerAssignment.Submission == nil || ownerAssignment.Submission.ID != assignmentOwner.ID {
		t.Fatalf("active assignment owner saw another user's aggregate: %+v", ownerAssignment)
	}
	adminAssignment := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, assignmentPath, databaseSession(t, db, admin.ID), nil))[0]
	if adminAssignment.Submit != 2 || adminAssignment.AC != 2 {
		t.Fatalf("admin assignment stats should include that assignment only: %+v", adminAssignment)
	}
}

func TestProblemStateTreatsLiveSubmissionAsPending(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	user := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	problem := models.Problem{ID: 1000, Title: "A+B", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Submission{UserID: user.ID, ProblemID: problem.ID, Language: "cpp", Code: "a", Status: "queued"}).Error; err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	Register(e, db)
	state := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?ids=1000", databaseSession(t, db, user.ID), nil))
	if len(state) != 1 || state[0].Status != "pending" || state[0].Submission == nil || state[0].Submission.Status != "queued" {
		t.Fatalf("live submission state = %+v", state)
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
