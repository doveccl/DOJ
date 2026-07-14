package public

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestDatabaseSubmitStoresAndValidatesContext(t *testing.T) {
	db := testWebDB(t)
	root := t.TempDir()
	t.Setenv("STORAGE", root)
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
		writeReadyProblemFiles(t, root, problem.ID, problem.Title)
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
	withoutContext := `{"problemId":1000,"language":"cpp","code":"int main(){}","public":true}`
	if res := requestJSONWithCookies(e, http.MethodPost, "/api/submissions", cookies, withoutContext); res.Code != http.StatusNotFound {
		t.Fatalf("hidden assignment problem must require explicit context: got %d body=%s", res.Code, res.Body.String())
	}
	okBody := `{"problemId":1000,"assignmentId":` + strconv.FormatUint(uint64(assignment.ID), 10) + `,"language":"cpp","code":"int main(){}","public":true}`
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
	if row.AssignmentID == nil || *row.AssignmentID != assignment.ID || row.ContestID != nil {
		t.Fatalf("assignment context not stored exclusively: %+v", row)
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
	row = models.Submission{}
	if err := db.First(&row, created.ID).Error; err != nil {
		t.Fatalf("read practice submission: %v", err)
	}
	if row.AssignmentID != nil || row.ContestID != nil {
		t.Fatalf("practice submission unexpectedly gained context: %+v", row)
	}

	contest := models.Contest{Title: "Round", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: included.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest link: %v", err)
	}
	contestBody := `{"problemId":1000,"contestId":` + strconv.FormatUint(uint64(contest.ID), 10) + `,"language":"cpp","code":"int main(){}","public":true}`
	res = requestJSONWithCookies(e, http.MethodPost, "/api/submissions", cookies, contestBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("contest submission got %d body=%s", res.Code, res.Body.String())
	}
	created = decodeJSON[contract.CreatedID](t, res)
	row = models.Submission{}
	if err := db.First(&row, created.ID).Error; err != nil {
		t.Fatalf("read contest submission: %v", err)
	}
	if row.ContestID == nil || *row.ContestID != contest.ID || row.AssignmentID != nil {
		t.Fatalf("contest context not stored exclusively: %+v", row)
	}

	bothBody := `{"problemId":1000,"assignmentId":` + strconv.FormatUint(uint64(assignment.ID), 10) + `,"contestId":` + strconv.FormatUint(uint64(contest.ID), 10) + `,"language":"cpp","code":"int main(){}","public":true}`
	if res := requestJSONWithCookies(e, http.MethodPost, "/api/submissions", cookies, bothBody); res.Code != http.StatusBadRequest {
		t.Fatalf("mixed assignment and contest context got %d body=%s", res.Code, res.Body.String())
	}

	unassigned := models.Assignment{Title: "Other HW", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&unassigned).Error; err != nil {
		t.Fatalf("create unassigned assignment: %v", err)
	}
	if err := db.Create(&models.AssignmentProblem{AssignmentID: unassigned.ID, ProblemID: included.ID}).Error; err != nil {
		t.Fatalf("create unassigned link: %v", err)
	}
	unassignedBody := `{"problemId":1000,"assignmentId":` + strconv.FormatUint(uint64(unassigned.ID), 10) + `,"language":"cpp","code":"int main(){}","public":true}`
	if res := requestJSONWithCookies(e, http.MethodPost, "/api/submissions", cookies, unassignedBody); res.Code != http.StatusNotFound {
		t.Fatalf("unassigned context got %d body=%s", res.Code, res.Body.String())
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
	submission := models.Submission{UserID: owner.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "secret", Status: "AC", Score: 100, Message: "accepted", TimeMS: &timeMS, MemoryKB: &memoryKB, Public: true}
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
	if otherView.Submission.Status != "pending" || otherView.Code != "" {
		t.Fatalf("running OI result and source leaked to another user: %+v", otherView)
	}
	adminView := decodeJSON[contract.SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, admin.ID), nil))
	if adminView.Submission.Status != "AC" || adminView.Submission.Score != 100 || len(adminView.Cases) != 1 || adminView.Code != "secret" {
		t.Fatalf("admin should see running OI result and source: %+v", adminView)
	}
	records := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?contest="+strconv.FormatUint(uint64(contest.ID), 10)+"&ids=1000", databaseSession(t, db, owner.ID), nil))
	if len(records) != 1 || records[0].Submit != 1 || records[0].AC != 0 || records[0].Status != "pending" {
		t.Fatalf("running OI contest problem should not expose record result: %+v", records)
	}
	contestStatePath := "/api/problem-state?contest=" + strconv.FormatUint(uint64(contest.ID), 10) + "&ids=1000"
	guestRecord := decodeJSON[[]contract.ProblemState](t, requestOK(t, e, http.MethodGet, contestStatePath, ""))[0]
	if guestRecord.Submit != 1 || guestRecord.AC != 0 || guestRecord.Status != "none" {
		t.Fatalf("guest contest aggregate exposed hidden OI result: %+v", guestRecord)
	}
	otherRecord := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, contestStatePath, databaseSession(t, db, other.ID), nil))[0]
	if otherRecord.Submit != 1 || otherRecord.AC != 0 || otherRecord.Status != "none" {
		t.Fatalf("outsider contest aggregate exposed hidden OI result: %+v", otherRecord)
	}
	adminRecord := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, contestStatePath, databaseSession(t, db, admin.ID), nil))[0]
	if adminRecord.Submit != 1 || adminRecord.AC != 1 {
		t.Fatalf("admin should see complete contest aggregate: %+v", adminRecord)
	}
	allRecords := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?ids=1000", databaseSession(t, db, owner.ID), nil))
	if len(allRecords) != 1 || allRecords[0].Submit != 2 || allRecords[0].AC != 0 || allRecords[0].Status != "pending" {
		t.Fatalf("global problem state should count all contexts without exposing OI results: %+v", allRecords)
	}
	assignmentDetail := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), databaseSession(t, db, owner.ID), nil))
	progress := assignmentDetail.Progress[0].Problems[0]
	if assignmentDetail.Assignment.Done != 0 || progress.Status != "tried" || progress.Score == nil || *progress.Score != tried.Score {
		t.Fatalf("assignment completion must ignore the separate contest submission: detail=%+v progress=%+v", assignmentDetail.Assignment, progress)
	}
	if err := db.Model(&tried).Updates(map[string]any{"status": "AC", "score": 100}).Error; err != nil {
		t.Fatalf("mark assignment submission accepted: %v", err)
	}
	assignmentState := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?assignment="+strconv.FormatUint(uint64(assignment.ID), 10)+"&ids=1000", databaseSession(t, db, owner.ID), nil))
	if len(assignmentState) != 1 || assignmentState[0].Submit != 1 || assignmentState[0].AC != 1 || assignmentState[0].Status != "ac" {
		t.Fatalf("assignment result should not be hidden by an unfinished contest on the problem: %+v", assignmentState)
	}
	assignmentList := decodeJSON[contract.Page[contract.Assignment]](t, requestWithCookies(e, http.MethodGet, "/api/assignments", databaseSession(t, db, owner.ID), nil))
	if len(assignmentList.Items) != 1 || assignmentList.Items[0].Done != 1 {
		t.Fatalf("assignment list done should not count hidden contest result: %+v", assignmentList)
	}
	ownerProfile := decodeJSON[contract.UserProfile](t, requestWithCookies(e, http.MethodGet, "/api/users/owner", databaseSession(t, db, owner.ID), nil))
	if activity, ok := activityBySubmission(ownerProfile.Activities, submission.ID); !ok || activity.Status != "pending" {
		t.Fatalf("profile activity should retain the hidden OI submission as pending: %+v", ownerProfile.Activities)
	}
	submissionList := decodeJSON[contract.Page[contract.SubmissionListItem]](t, requestWithCookies(e, http.MethodGet, "/api/submissions", databaseSession(t, db, owner.ID), nil))
	if len(submissionList.Items) != 2 ||
		submissionList.Items[0].ID != submission.ID || submissionList.Items[0].Status != "pending" || submissionList.Items[0].TimeMS != nil || submissionList.Items[0].MemoryKB != nil ||
		submissionList.Items[1].ID != tried.ID || submissionList.Items[1].Status != "AC" {
		t.Fatalf("submission list should hide only the direct unfinished OI result: %+v", submissionList)
	}
	otherAssignment := decodeJSON[contract.SubmissionDetail](t, requestWithCookies(e, http.MethodGet, "/api/submissions/"+strconv.FormatUint(uint64(tried.ID), 10), databaseSession(t, db, other.ID), nil))
	if otherAssignment.Submission.Status != "AC" || otherAssignment.Code != "" {
		t.Fatalf("unfinished contest membership should hide historical public source, not its result: %+v", otherAssignment)
	}
	hiddenAC := decodeJSON[contract.Page[contract.SubmissionListItem]](t, requestWithCookies(e, http.MethodGet, "/api/submissions?contest="+strconv.FormatUint(uint64(contest.ID), 10)+"&status=AC", databaseSession(t, db, owner.ID), nil))
	if hiddenAC.Total != 0 || len(hiddenAC.Items) != 0 {
		t.Fatalf("status filter must not reveal hidden OI verdict: %+v", hiddenAC)
	}
	adminAC := decodeJSON[contract.Page[contract.SubmissionListItem]](t, requestWithCookies(e, http.MethodGet, "/api/submissions?contest="+strconv.FormatUint(uint64(contest.ID), 10)+"&status=AC", databaseSession(t, db, admin.ID), nil))
	if adminAC.Total != 1 || len(adminAC.Items) != 1 || adminAC.Items[0].ID != submission.ID {
		t.Fatalf("admin status filter should see OI verdict: %+v", adminAC)
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
