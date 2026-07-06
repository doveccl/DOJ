package web

import (
	"github.com/doveccl/doj/models"
	judgersvc "github.com/doveccl/doj/server/judger"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPrivateSubmissionSourceVisibilityWithDatabase(t *testing.T) {
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
	problem := models.Problem{
		ID:       1000,
		Title:    "Visible",
		Tags:     datatypes.JSON([]byte(`[]`)),
		Visible:  true,
		Mode:     "default",
		TimeMS:   1000,
		MemoryMB: 256,
	}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	private := models.Submission{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "secret", Status: "AC", Score: 100, Public: false}
	public := models.Submission{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "visible", Status: "AC", Score: 100, Public: true}
	if err := db.Create(&private).Error; err != nil {
		t.Fatalf("create private submission: %v", err)
	}
	if err := db.Create(&public).Error; err != nil {
		t.Fatalf("create public submission: %v", err)
	}

	e := echo.New()
	Register(e, db)

	privateTarget := "/api/submissions/" + strconv.FormatUint(uint64(private.ID), 10)
	publicTarget := "/api/submissions/" + strconv.FormatUint(uint64(public.ID), 10)
	if res := requestWithCookies(e, http.MethodGet, publicTarget, nil, nil); res.Code != http.StatusOK {
		t.Fatalf("guest should read public DB submission, got %d body=%s", res.Code, res.Body.String())
	}
	otherDetail := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, privateTarget, databaseSession(t, db, other.ID), nil))
	if otherDetail.Code != "" || otherDetail.Submission.ID != private.ID {
		t.Fatalf("other user should read private submission detail without source: %+v", otherDetail)
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, privateTarget, databaseSession(t, db, owner.ID), nil)); got.Code != "secret" {
		t.Fatalf("owner should read private DB submission source: %+v", got)
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, privateTarget, databaseSession(t, db, admin.ID), nil)); got.Code != "secret" {
		t.Fatalf("admin should read private DB submission source: %+v", got)
	}
}

func TestSubmissionDetailIncludesTopLevelMessage(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	submission := models.Submission{
		UserID:    owner.ID,
		ProblemID: problem.ID,
		Language:  "cpp",
		Code:      "int main(){",
		Status:    "CE",
		Message:   "compile failed\nmain.cpp:1: error: expected '}'",
		Public:    true,
	}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	e := echo.New()
	Register(e, db)

	target := "/api/submissions/" + strconv.FormatUint(uint64(submission.ID), 10)
	got := decodeJSON[SubmissionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if got.Submission.Message != submission.Message {
		t.Fatalf("submission detail should expose top-level judge message: %+v", got.Submission)
	}
}

func TestSubmissionDetailIncludesProgressButListDoesNot(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	total := int64(12)
	submission := models.Submission{
		UserID:    owner.ID,
		ProblemID: problem.ID,
		Language:  "cpp",
		Code:      "int main(){}",
		Status:    "judging",
		Attempt:   4,
		Public:    true,
	}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	if err := judgersvc.SaveProgress(t.Context(), submission.ID, judgersvc.Progress{Attempt: 4, Stage: "judge", Done: 3, Total: &total, UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("save progress: %v", err)
	}

	e := echo.New()
	Register(e, db)

	target := "/api/submissions/" + strconv.FormatUint(uint64(submission.ID), 10)
	got := decodeJSON[SubmissionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if got.Progress == nil || got.Progress.Stage != "judge" || got.Progress.Done != 3 || got.Progress.Total == nil || *got.Progress.Total != total {
		t.Fatalf("submission detail progress = %+v", got.Progress)
	}

	list := requestOK(t, e, http.MethodGet, "/api/submissions", "")
	if strings.Contains(list.Body.String(), "progress") {
		t.Fatalf("submission list should not expose progress: %s", list.Body.String())
	}
}

func TestSubmissionPublicCanBeUpdatedByOwnerOrAdmin(t *testing.T) {
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
	problem := models.Problem{
		ID:       1000,
		Title:    "Visible",
		Tags:     datatypes.JSON([]byte(`[]`)),
		Visible:  true,
		Mode:     "default",
		TimeMS:   1000,
		MemoryMB: 256,
	}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	submission := models.Submission{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "code", Status: "AC", Score: 100, Public: false}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	e := echo.New()
	Register(e, db)
	target := "/api/submissions/" + strconv.FormatUint(uint64(submission.ID), 10)
	if res := requestJSONWithCookies(e, http.MethodPatch, target, nil, `{"public":true}`); res.Code != http.StatusUnauthorized {
		t.Fatalf("guest should not update submission public flag, got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, other.ID), `{"public":true}`); res.Code != http.StatusForbidden {
		t.Fatalf("other user should not update submission public flag, got %d body=%s", res.Code, res.Body.String())
	}
	ownerRes := requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, owner.ID), `{"public":true}`)
	if ownerRes.Code != http.StatusOK {
		t.Fatalf("owner should update submission public flag, got %d body=%s", ownerRes.Code, ownerRes.Body.String())
	}
	if got := decodeJSON[CreatedID](t, ownerRes); got.ID != submission.ID {
		t.Fatalf("owner update should return submission id: %+v", got)
	}
	var got models.Submission
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("reload submission: %v", err)
	}
	if !got.Public {
		t.Fatalf("owner update should persist public=true")
	}
	adminRes := requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, admin.ID), `{"public":false}`)
	if adminRes.Code != http.StatusOK {
		t.Fatalf("admin should update submission public flag, got %d body=%s", adminRes.Code, adminRes.Body.String())
	}
	if got := decodeJSON[CreatedID](t, adminRes); got.ID != submission.ID {
		t.Fatalf("admin update should return submission id: %+v", got)
	}
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("reload submission: %v", err)
	}
	if got.Public {
		t.Fatalf("admin update should persist public=false")
	}
}

func TestContextSubmissionSourceLockedUntilContextEnds(t *testing.T) {
	db := testWebDB(t)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	other := models.User{Name: "other", Mail: "other@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&owner, &other, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	now := time.Now()
	liveContest := models.Contest{Title: "Live Contest", Kind: "ICPC", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour)}
	endedContest := models.Contest{Title: "Ended Contest", Kind: "ICPC", StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Hour)}
	liveAssignment := models.Assignment{Title: "Live Assignment", EndAt: now.Add(time.Hour)}
	endedAssignment := models.Assignment{Title: "Ended Assignment", EndAt: now.Add(-time.Hour)}
	for _, row := range []any{&liveContest, &endedContest, &liveAssignment, &endedAssignment} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create context: %v", err)
		}
	}
	liveContestID := liveContest.ID
	endedContestID := endedContest.ID
	liveAssignmentID := liveAssignment.ID
	endedAssignmentID := endedAssignment.ID
	submissions := []models.Submission{
		{UserID: owner.ID, ProblemID: problem.ID, ContestID: &liveContestID, Language: "cpp", Code: "live contest", Status: "AC", Score: 100, Public: true},
		{UserID: owner.ID, ProblemID: problem.ID, ContestID: &endedContestID, Language: "cpp", Code: "ended contest", Status: "AC", Score: 100, Public: true},
		{UserID: owner.ID, ProblemID: problem.ID, AssignmentID: &liveAssignmentID, Language: "cpp", Code: "live assignment", Status: "AC", Score: 100, Public: true},
		{UserID: owner.ID, ProblemID: problem.ID, AssignmentID: &endedAssignmentID, Language: "cpp", Code: "ended assignment", Status: "AC", Score: 100, Public: true},
		{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "private", Status: "AC", Score: 100, Public: false},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)
	ownerCookies := databaseSession(t, db, owner.ID)
	otherCookies := databaseSession(t, db, other.ID)
	adminCookies := databaseSession(t, db, admin.ID)
	target := func(row models.Submission) string {
		return "/api/submissions/" + strconv.FormatUint(uint64(row.ID), 10)
	}

	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target(submissions[0]), otherCookies, nil)); got.Code != "" || got.Submission.Status != "AC" {
		t.Fatalf("other user should read live contest detail without source: %+v", got)
	}
	if res := requestWithCookies(e, http.MethodGet, target(submissions[0]), ownerCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("owner should read live contest source, got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, target(submissions[0]), adminCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("admin should read live contest source, got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target(submissions[1]), otherCookies, nil)); got.Code != "ended contest" {
		t.Fatalf("other user should read ended contest public source: %+v", got)
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target(submissions[2]), otherCookies, nil)); got.Code != "" || got.Submission.Status != "AC" {
		t.Fatalf("other user should read live assignment detail without source: %+v", got)
	}
	if res := requestWithCookies(e, http.MethodGet, target(submissions[2]), ownerCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("owner should read live assignment source, got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, target(submissions[2]), adminCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("admin should read live assignment source, got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target(submissions[3]), otherCookies, nil)); got.Code != "ended assignment" {
		t.Fatalf("other user should read ended assignment public source: %+v", got)
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target(submissions[4]), otherCookies, nil)); got.Code != "" {
		t.Fatalf("other user should read private detail without source: %+v", got)
	}
}

func TestSubmissionEndpointsReadHiddenProblemSubmissionsButHideFields(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	other := models.User{Name: "other", Mail: "other@example.com", Auth: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Hidden", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	assignment := models.Assignment{Title: "HW", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if err := db.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create assignment problem: %v", err)
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: assignment.ID, UserID: owner.ID}).Error; err != nil {
		t.Fatalf("assign owner: %v", err)
	}
	assignmentID := assignment.ID
	submission := models.Submission{UserID: owner.ID, ProblemID: problem.ID, AssignmentID: &assignmentID, Language: "cpp", Code: "hidden source", Status: "AC", Score: 100, Public: true}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	e := echo.New()
	Register(e, db)
	otherCookies := databaseSession(t, db, other.ID)
	list := decodeJSON[PageResult[SubmissionListItem]](t, requestWithCookies(e, http.MethodGet, "/api/submissions", otherCookies, nil))
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != submission.ID || list.Items[0].ProblemID != problem.ID || list.Items[0].Status != "AC" {
		t.Fatalf("hidden problem submission should remain visible in list with public result: %+v", list)
	}
	detail := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, "/api/submissions/"+strconv.FormatUint(uint64(submission.ID), 10), otherCookies, nil))
	if detail.Submission.ID != submission.ID || detail.Submission.ProblemTitle != "Hidden" || detail.Submission.Status != "AC" || detail.Code != "" {
		t.Fatalf("hidden problem submission detail should read but hide source from non-owner live assignment view: %+v", detail)
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/problems/1000", otherCookies, nil); res.Code != http.StatusNotFound {
		t.Fatalf("problem detail should still be hidden, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestRunningAssignmentShowsResultsButHidesOtherSource(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	other := models.User{Name: "other", Mail: "other@example.com", Auth: "hash"}
	for _, user := range []*models.User{&owner, &other} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "HW", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	assignment := models.Assignment{Title: "HW", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	assignmentID := assignment.ID
	submission := models.Submission{UserID: owner.ID, ProblemID: problem.ID, AssignmentID: &assignmentID, Language: "cpp", Code: "homework", Status: "WA", Score: 20, Public: true}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	e := echo.New()
	Register(e, db)
	got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, "/api/submissions/"+strconv.FormatUint(uint64(submission.ID), 10), databaseSession(t, db, other.ID), nil))
	if got.Submission.Status != "WA" || got.Submission.Score != 20 || got.Code != "" {
		t.Fatalf("running assignment should expose result but hide source from other users: %+v", got)
	}
}
