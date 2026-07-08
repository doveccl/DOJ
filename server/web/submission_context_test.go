package web

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	contract "github.com/doveccl/doj/common/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestDatabaseSubmitStoresAndValidatesContext(t *testing.T) {
	db := testWebDB(t)
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	included := models.Problem{ID: 1000, Title: "Included", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	outside := models.Problem{ID: 1001, Title: "Outside", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	for _, problem := range []*models.Problem{&included, &outside} {
		if err := db.Create(problem).Error; err != nil {
			t.Fatalf("create problem %s: %v", problem.Title, err)
		}
	}
	assignment := models.Assignment{Title: "HW", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if err := db.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: included.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create assignment link: %v", err)
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: assignment.ID, UserID: student.ID}).Error; err != nil {
		t.Fatalf("assign student: %v", err)
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, student.ID)
	list := decodePageItems[contract.Problem](t, requestWithCookies(e, http.MethodGet, "/api/problems", cookies, nil))
	if hasProblem(list, included.ID) {
		t.Fatalf("assignment-only hidden problem should not appear in problem list: %+v", list)
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/problems/1000", cookies, nil); res.Code != http.StatusOK {
		t.Fatalf("assigned active assignment problem detail got %d body=%s", res.Code, res.Body.String())
	}
	okBody := `{"problemId":1000,"language":"cpp","code":"int main(){}","public":true}`
	res := requestJSONWithCookies(e, http.MethodPost, "/api/submissions", cookies, okBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("assignment submission got %d body=%s", res.Code, res.Body.String())
	}
	created := decodeJSON[contract.CreatedID](t, res)
	var row models.Submission
	if err := db.First(&row, "problem_id = ?", included.ID).Error; err != nil {
		t.Fatalf("read created submission: %v", err)
	}
	if created.ID != row.ID {
		t.Fatalf("assignment submission should return created id: got %d row %d", created.ID, row.ID)
	}
	if row.AssignmentID == nil || *row.AssignmentID != assignment.ID {
		t.Fatalf("assignment id not inferred: %+v", row)
	}

	normalBody := `{"problemId":1001,"language":"cpp","code":"int main(){}","public":true}`
	res = requestJSONWithCookies(e, http.MethodPost, "/api/submissions", cookies, normalBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("normal submission got %d body=%s", res.Code, res.Body.String())
	}
	created = decodeJSON[contract.CreatedID](t, res)
	if created.ID == 0 {
		t.Fatalf("normal submission should return created id: %+v", created)
	}
}

func TestRunningOIContestHidesSubmissionResults(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	other := models.User{Name: "other", Mail: "other@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&owner, &other, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "OI", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	contest := models.Contest{Title: "OI Round", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	assignment := models.Assignment{Title: "Overlapping HW", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if err := db.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create assignment problem: %v", err)
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: assignment.ID, UserID: owner.ID}).Error; err != nil {
		t.Fatalf("assign owner: %v", err)
	}
	contestID := contest.ID
	assignmentID := assignment.ID
	timeMS := 12
	memoryKB := 345
	tried := models.Submission{UserID: owner.ID, ProblemID: problem.ID, AssignmentID: &assignmentID, Language: "cpp", Code: "visible try", Status: "WA", Score: 20, Public: true, CreatedAt: time.Now().Add(-10 * time.Minute)}
	if err := db.Create(&tried).Error; err != nil {
		t.Fatalf("create tried submission: %v", err)
	}
	submission := models.Submission{UserID: owner.ID, ProblemID: problem.ID, AssignmentID: &assignmentID, ContestID: &contestID, Language: "cpp", Code: "secret", Status: "AC", Score: 100, Message: "accepted", TimeMS: &timeMS, MemoryKB: &memoryKB, Public: true}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	if err := db.Create(&models.Case{SubmissionID: submission.ID, No: 1, Status: "AC", TimeMS: &timeMS, MemoryKB: &memoryKB, Message: "ok"}).Error; err != nil {
		t.Fatalf("create case: %v", err)
	}

	e := echo.New()
	Register(e, db)
	target := "/api/submissions/" + strconv.FormatUint(uint64(submission.ID), 10)
	ownerView := decodeJSON[contract.SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, owner.ID), nil))
	if ownerView.Submission.Status != "pending" || ownerView.Submission.Score != 0 || ownerView.Submission.Message != "" || ownerView.Submission.TimeMS != nil || ownerView.Submission.MemoryKB != nil || len(ownerView.Cases) != 0 || ownerView.Code != "secret" {
		t.Fatalf("running OI owner should see source but pending result: %+v", ownerView)
	}
	otherView := decodeJSON[contract.SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, other.ID), nil))
	if otherView.Submission.Status != "pending" || otherView.Code != "" || len(otherView.Cases) != 0 {
		t.Fatalf("running OI other user should see pending detail without source: %+v", otherView)
	}
	adminView := decodeJSON[contract.SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, admin.ID), nil))
	if adminView.Submission.Status != "AC" || adminView.Submission.Score != 100 || len(adminView.Cases) != 1 || adminView.Code != "secret" {
		t.Fatalf("admin should see running OI result and source: %+v", adminView)
	}
	records := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?contest="+strconv.FormatUint(uint64(contest.ID), 10)+"&ids=1000", databaseSession(t, db, owner.ID), nil))
	if len(records) != 1 || records[0].Status != "pending" || records[0].Submission == nil || records[0].Submission.Status != "pending" || records[0].Submission.Score != 0 {
		t.Fatalf("running OI contest problem should not expose record result: %+v", records)
	}
	allRecords := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?ids=1000", databaseSession(t, db, owner.ID), nil))
	if len(allRecords) != 1 || allRecords[0].Status != "pending" || allRecords[0].Submission == nil || allRecords[0].Submission.ID != submission.ID || allRecords[0].Submission.Status != "pending" {
		t.Fatalf("global problem state should use the current user's submission with visibility applied: %+v", allRecords)
	}
	assignmentDetail := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), databaseSession(t, db, owner.ID), nil))
	progress := assignmentDetail.Progress[0].Problems[0]
	if assignmentDetail.Assignment.Done != 0 || progress.Status != "pending" || progress.Score != nil {
		t.Fatalf("assignment completion should not expose hidden contest result: detail=%+v progress=%+v", assignmentDetail.Assignment, progress)
	}
	assignmentList := decodeJSON[contract.Page[contract.Assignment]](t, requestWithCookies(e, http.MethodGet, "/api/assignments", databaseSession(t, db, owner.ID), nil))
	if len(assignmentList.Items) != 1 || assignmentList.Items[0].Done != 0 {
		t.Fatalf("assignment list done should not count hidden contest result: %+v", assignmentList)
	}
	ownerProfile := decodeJSON[contract.UserProfile](t, requestWithCookies(e, http.MethodGet, "/api/users/owner", databaseSession(t, db, owner.ID), nil))
	if _, ok := activityBySubmission(ownerProfile.Activities, submission.ID); ok {
		t.Fatalf("profile activity should not expose unfinished contest submission: %+v", ownerProfile.Activities)
	}
	submissionList := decodeJSON[contract.Page[contract.SubmissionListItem]](t, requestWithCookies(e, http.MethodGet, "/api/submissions", databaseSession(t, db, owner.ID), nil))
	if len(submissionList.Items) != 2 || submissionList.Items[0].ID != submission.ID || submissionList.Items[0].Status != "pending" || submissionList.Items[0].TimeMS != nil || submissionList.Items[0].MemoryKB != nil || submissionList.Items[1].ID != tried.ID || submissionList.Items[1].Status != "WA" {
		t.Fatalf("submission endpoint list should include running OI submission as pending: %+v", submissionList)
	}
	api := &API{db: db}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range databaseSession(t, db, owner.ID) {
		req.AddCookie(cookie)
	}
	ctx := echo.New().NewContext(req, httptest.NewRecorder())
	if got, err := api.submissionListItems(ctx, []models.Submission{submission}); err != nil || len(got) != 1 || got[0].Status != "pending" || got[0].TimeMS != nil || got[0].MemoryKB != nil {
		t.Fatalf("submission list item should hide running OI result: items=%+v err=%v", got, err)
	}
}
