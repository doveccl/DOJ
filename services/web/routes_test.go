package web

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/doveccl/doj/models"
	adminsvc "github.com/doveccl/doj/services/admin"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	e := echo.New()
	Register(e, db)

	guestProblem := decodeJSON[ProblemDTO](t, requestOK(t, e, http.MethodGet, "/api/problems/1000", ""))
	if guestProblem.Discussions != 2 {
		t.Fatalf("guest discussion count should use soft tags regardless of other problem tags: %+v", guestProblem)
	}
	adminProblem := decodeJSON[ProblemDTO](t, requestWithCookies(e, http.MethodGet, "/api/problems/1000", databaseSession(t, db, admin.ID), nil))
	if adminProblem.Discussions != 2 {
		t.Fatalf("admin discussion count should match soft tag count: %+v", adminProblem)
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
	if res := requestWithCookies(e, http.MethodGet, privateTarget, databaseSession(t, db, other.ID), nil); res.Code != http.StatusForbidden {
		t.Fatalf("other user should not read private DB submission, got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, privateTarget, databaseSession(t, db, owner.ID), nil); res.Code != http.StatusOK {
		t.Fatalf("owner should read private DB submission, got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, privateTarget, databaseSession(t, db, admin.ID), nil); res.Code != http.StatusOK {
		t.Fatalf("admin should read private DB submission, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestHiddenProblemReferencesDoNotLeakFromDatabaseProfilesAndContexts(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	outsider := models.User{Name: "outsider", Mail: "outsider@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	if err := db.Create(&outsider).Error; err != nil {
		t.Fatalf("create outsider: %v", err)
	}
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

	assignment := models.Assignment{Title: "HW", EndAt: time.Now().Add(time.Hour)}
	contest := models.Contest{Title: "Round", Kind: "OI", StartAt: time.Now().Add(-2 * time.Hour), EndAt: time.Now().Add(-time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	links := []any{
		&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: visible.ID, Sort: "A"},
		&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: hidden.ID, Sort: "B"},
		&models.ContestProblem{ContestID: contest.ID, ProblemID: visible.ID, Sort: "A"},
		&models.ContestProblem{ContestID: contest.ID, ProblemID: hidden.ID, Sort: "B"},
	}
	for _, link := range links {
		if err := db.Create(link).Error; err != nil {
			t.Fatalf("create link: %v", err)
		}
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: assignment.ID, UserID: student.ID}).Error; err != nil {
		t.Fatalf("assign student: %v", err)
	}

	assignmentID := assignment.ID
	contestID := contest.ID
	submissions := []models.Submission{
		{UserID: student.ID, ProblemID: visible.ID, Language: "cpp", Code: "visible", Status: "AC", Score: 100, Public: true},
		{UserID: student.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden", Status: "AC", Score: 100, Public: true},
		{UserID: student.ID, ProblemID: visible.ID, Language: "cpp", Code: "visible hw", AssignmentID: &assignmentID, Status: "AC", Score: 100, Public: true},
		{UserID: student.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden hw", AssignmentID: &assignmentID, Status: "AC", Score: 100, Public: true},
		{UserID: student.ID, ProblemID: visible.ID, Language: "cpp", Code: "visible round", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
		{UserID: student.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden round", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)

	profile := decodeJSON[UserProfile](t, requestOK(t, e, http.MethodGet, "/api/users/student", ""))
	if hasProblem(profile.Solved, hidden.ID) || hasSubmissionProblem(profile.Submissions, hidden.ID) {
		t.Fatalf("guest profile leaked hidden problem: %+v", profile)
	}
	if profile.User.AC != 1 || profile.User.Submit != 3 {
		t.Fatalf("guest profile stats should count visible activity only: %+v", profile.User)
	}
	today := time.Now().Format("2006-01-02")
	if got := countForDate(profile.Heatmap, today); got != 3 {
		t.Fatalf("guest profile heatmap should count visible submissions only, got %d", got)
	}

	if res := requestOK(t, e, http.MethodGet, "/api/assignments", ""); res.Body.String() != "[]\n" {
		t.Fatalf("guest assignment list should be empty, body=%s", res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), nil, nil); res.Code != http.StatusNotFound {
		t.Fatalf("guest assignment detail should be hidden, got %d body=%s", res.Code, res.Body.String())
	}
	outsiderCookies := databaseSession(t, db, outsider.ID)
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments", outsiderCookies, nil); res.Body.String() != "[]\n" {
		t.Fatalf("unassigned assignment list should be empty, body=%s", res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), outsiderCookies, nil); res.Code != http.StatusNotFound {
		t.Fatalf("unassigned assignment detail should be hidden, got %d body=%s", res.Code, res.Body.String())
	}
	studentAssignment := decodeJSON[AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), databaseSession(t, db, student.ID), nil))
	if hasProblem(studentAssignment.Problems, hidden.ID) || hasSubmissionProblem(studentAssignment.Submissions, hidden.ID) {
		t.Fatalf("student assignment leaked hidden problem: %+v", studentAssignment)
	}
	if studentAssignment.Assignment.Done != 1 || studentAssignment.Assignment.Total != 1 {
		t.Fatalf("student assignment progress should count visible AC only: %+v", studentAssignment.Assignment)
	}

	contestDetail := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
	if hasProblem(contestDetail.Problems, hidden.ID) || hasSubmissionProblem(contestDetail.Submissions, hidden.ID) {
		t.Fatalf("guest contest leaked hidden problem: %+v", contestDetail)
	}

	adminProfileRes := requestWithCookies(e, http.MethodGet, "/api/users/student", databaseSession(t, db, admin.ID), nil)
	if adminProfileRes.Code != http.StatusOK {
		t.Fatalf("admin profile got %d body=%s", adminProfileRes.Code, adminProfileRes.Body.String())
	}
	adminProfile := decodeJSON[UserProfile](t, adminProfileRes)
	if !hasProblem(adminProfile.Solved, hidden.ID) || !hasSubmissionProblem(adminProfile.Submissions, hidden.ID) {
		t.Fatalf("admin profile should include hidden problem: %+v", adminProfile)
	}
	if adminProfile.User.AC != 2 || adminProfile.User.Submit != 6 {
		t.Fatalf("admin profile stats should include hidden activity: %+v", adminProfile.User)
	}
	if got := countForDate(adminProfile.Heatmap, today); got != 6 {
		t.Fatalf("admin profile heatmap should include hidden submissions, got %d", got)
	}
}

func TestDatabaseRankUsesVisibleSubmissionStatsAndActiveUsers(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	disabled := models.User{Name: "disabled", Mail: "disabled@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&alice, &bob, &disabled, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	if err := db.Delete(&disabled).Error; err != nil {
		t.Fatalf("delete disabled user: %v", err)
	}
	visibleA := models.Problem{ID: 1000, Title: "Visible A", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	visibleB := models.Problem{ID: 1001, Title: "Visible B", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	hidden := models.Problem{ID: 1002, Title: "Hidden", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	for _, problem := range []*models.Problem{&visibleA, &visibleB, &hidden} {
		if err := db.Create(problem).Error; err != nil {
			t.Fatalf("create problem %s: %v", problem.Title, err)
		}
	}
	submissions := []models.Submission{
		{UserID: alice.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "a1", Status: "AC", Score: 100, Public: true},
		{UserID: alice.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "a2", Status: "WA", Score: 0, Public: true},
		{UserID: bob.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "b1", Status: "AC", Score: 100, Public: true},
		{UserID: bob.ID, ProblemID: visibleB.ID, Language: "cpp", Code: "b2", Status: "AC", Score: 100, Public: true},
		{UserID: alice.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden", Status: "AC", Score: 100, Public: true},
		{UserID: disabled.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "disabled", Status: "AC", Score: 100, Public: true},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)

	guestRank := decodeJSON[[]RankUserDTO](t, requestOK(t, e, http.MethodGet, "/api/rank", ""))
	if len(guestRank) != 3 {
		t.Fatalf("rank should include three active users, got %+v", guestRank)
	}
	if guestRank[0].User != "bob" || guestRank[0].AC != 2 || guestRank[0].Submit != 2 {
		t.Fatalf("bob should rank first by visible AC: %+v", guestRank)
	}
	if userInRank(guestRank, "disabled") {
		t.Fatalf("rank should not include disabled users: %+v", guestRank)
	}
	if res := request(e, http.MethodGet, "/api/users/disabled", "", nil); res.Code != http.StatusNotFound {
		t.Fatalf("disabled user profile should be hidden, got %d body=%s", res.Code, res.Body.String())
	}
	aliceGuest, ok := rankByUser(guestRank, "alice")
	if !ok || aliceGuest.AC != 1 || aliceGuest.Submit != 2 {
		t.Fatalf("alice guest stats should ignore hidden problem: %+v", guestRank)
	}

	adminRank := decodeJSON[[]RankUserDTO](t, requestWithCookies(e, http.MethodGet, "/api/rank", databaseSession(t, db, admin.ID), nil))
	aliceAdmin, ok := rankByUser(adminRank, "alice")
	if !ok || aliceAdmin.AC != 2 || aliceAdmin.Submit != 3 {
		t.Fatalf("alice admin stats should include hidden problem: %+v", adminRank)
	}
}

func TestDatabaseContestRankUsesContextSubmissions(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	disabled := models.User{Name: "disabled", Mail: "disabled@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&alice, &bob, &disabled, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	if err := db.Delete(&disabled).Error; err != nil {
		t.Fatalf("delete disabled user: %v", err)
	}

	visibleA := models.Problem{ID: 1000, Title: "Visible A", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	visibleB := models.Problem{ID: 1001, Title: "Visible B", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	hidden := models.Problem{ID: 1002, Title: "Hidden", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	for _, problem := range []*models.Problem{&visibleA, &visibleB, &hidden} {
		if err := db.Create(problem).Error; err != nil {
			t.Fatalf("create problem %s: %v", problem.Title, err)
		}
	}
	contest := models.Contest{Title: "Round", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	for index, problem := range []models.Problem{visibleA, visibleB, hidden} {
		link := models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: problemSort(index)}
		if err := db.Create(&link).Error; err != nil {
			t.Fatalf("create contest link: %v", err)
		}
	}

	contestID := contest.ID
	submissions := []models.Submission{
		{UserID: alice.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "a1", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
		{UserID: alice.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "a2", ContestID: &contestID, Status: "WA", Score: 0, Public: true},
		{UserID: alice.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
		{UserID: alice.ID, ProblemID: visibleB.ID, Language: "cpp", Code: "normal", Status: "AC", Score: 100, Public: true},
		{UserID: bob.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "b1", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
		{UserID: bob.ID, ProblemID: visibleB.ID, Language: "cpp", Code: "b2", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
		{UserID: disabled.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "disabled", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)

	guest := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
	if len(guest.Rank) != 2 || guest.Rank[0].User != "bob" {
		t.Fatalf("guest contest rank should include visible active competitors only: %+v", guest.Rank)
	}
	aliceGuest, ok := rankByUser(guest.Rank, "alice")
	if !ok || aliceGuest.AC != 2 || aliceGuest.Submit != 3 {
		t.Fatalf("alice guest running contest rank should include contest-hidden problems and ignore normal submissions: %+v", guest.Rank)
	}
	if userInRank(guest.Rank, "disabled") {
		t.Fatalf("contest rank should not include disabled users: %+v", guest.Rank)
	}

	adminDetail := decodeJSON[ContestDetail](t, requestWithCookies(e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), databaseSession(t, db, admin.ID), nil))
	aliceAdmin, ok := rankByUser(adminDetail.Rank, "alice")
	if !ok || aliceAdmin.AC != 2 || aliceAdmin.Submit != 3 {
		t.Fatalf("admin contest rank should include hidden contest submissions: %+v", adminDetail.Rank)
	}
}

func TestDatabaseSubmitStoresAndValidatesContext(t *testing.T) {
	db := testWebDB(t)
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	included := models.Problem{ID: 1000, Title: "Included", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
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
	okBody := `{"problemId":1000,"language":"cpp","code":"int main(){}","public":true}`
	res := requestJSONWithCookies(e, http.MethodPost, "/api/submissions", cookies, okBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("assignment submission got %d body=%s", res.Code, res.Body.String())
	}
	var row models.Submission
	if err := db.First(&row, "problem_id = ?", included.ID).Error; err != nil {
		t.Fatalf("read created submission: %v", err)
	}
	if row.AssignmentID == nil || *row.AssignmentID != assignment.ID {
		t.Fatalf("assignment id not inferred: %+v", row)
	}

	normalBody := `{"problemId":1001,"language":"cpp","code":"int main(){}","public":true}`
	res = requestJSONWithCookies(e, http.MethodPost, "/api/submissions", cookies, normalBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("normal submission got %d body=%s", res.Code, res.Body.String())
	}
}

func TestContestProblemVisibilityIsDerivedFromContestTime(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Contest Only", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	now := time.Now()
	contest := models.Contest{Title: "Future", Kind: "OI", StartAt: now.Add(time.Hour), EndAt: now.Add(2 * time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	e := echo.New()
	Register(e, db)

	guestList := decodeJSON[[]ProblemDTO](t, requestOK(t, e, http.MethodGet, "/api/problems", ""))
	if hasProblem(guestList, problem.ID) {
		t.Fatalf("upcoming contest problem leaked in problem list: %+v", guestList)
	}
	if res := request(e, http.MethodGet, "/api/problems/1000", "", nil); res.Code != http.StatusNotFound {
		t.Fatalf("upcoming contest problem detail got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/problems/1000", databaseSession(t, db, admin.ID), nil); res.Code != http.StatusOK {
		t.Fatalf("admin should see upcoming contest problem, got %d body=%s", res.Code, res.Body.String())
	}

	contest.StartAt = now.Add(-time.Hour)
	contest.EndAt = now.Add(time.Hour)
	if err := db.Save(&contest).Error; err != nil {
		t.Fatalf("start contest: %v", err)
	}
	if err := db.Model(&problem).Update("visible", false).Error; err != nil {
		t.Fatalf("hide problem: %v", err)
	}
	guestList = decodeJSON[[]ProblemDTO](t, requestOK(t, e, http.MethodGet, "/api/problems", ""))
	if hasProblem(guestList, problem.ID) {
		t.Fatalf("running contest problem leaked in problem list: %+v", guestList)
	}
	if res := request(e, http.MethodGet, "/api/problems/1000", "", nil); res.Code != http.StatusOK {
		t.Fatalf("running contest problem detail should be visible, got %d body=%s", res.Code, res.Body.String())
	}
	contestDetail := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
	if !hasProblem(contestDetail.Problems, problem.ID) {
		t.Fatalf("running contest detail should include linked problem: %+v", contestDetail.Problems)
	}

	contest.EndAt = now.Add(-time.Minute)
	if err := db.Save(&contest).Error; err != nil {
		t.Fatalf("end contest: %v", err)
	}
	if res := request(e, http.MethodGet, "/api/problems/1000", "", nil); res.Code != http.StatusNotFound {
		t.Fatalf("ended hidden contest problem detail got %d body=%s", res.Code, res.Body.String())
	}
	if err := db.Model(&problem).Update("visible", true).Error; err != nil {
		t.Fatalf("show problem: %v", err)
	}
	if res := request(e, http.MethodGet, "/api/problems/1000", "", nil); res.Code != http.StatusOK {
		t.Fatalf("ended visible contest problem detail got %d body=%s", res.Code, res.Body.String())
	}
}

func TestContestFreezeHidesLateResultsFromNonAdmin(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&alice, &bob, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "Frozen", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	now := time.Now()
	freezeAt := now.Add(-time.Hour)
	contest := models.Contest{Title: "Frozen Round", Kind: "OI", StartAt: now.Add(-2 * time.Hour), FreezeAt: &freezeAt, EndAt: now.Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	contestID := contest.ID
	before := models.Submission{UserID: alice.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "before", Status: "AC", Score: 100, Public: true, CreatedAt: now.Add(-90 * time.Minute)}
	after := models.Submission{UserID: bob.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "after", Status: "AC", Score: 100, Public: true, CreatedAt: now.Add(-30 * time.Minute)}
	if err := db.Create(&before).Error; err != nil {
		t.Fatalf("create before submission: %v", err)
	}
	if err := db.Create(&after).Error; err != nil {
		t.Fatalf("create after submission: %v", err)
	}

	e := echo.New()
	Register(e, db)
	target := "/api/contests/" + strconv.FormatUint(uint64(contest.ID), 10)
	guest := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if guest.Contest.Status != "frozen" {
		t.Fatalf("contest status should be frozen: %+v", guest.Contest)
	}
	if len(guest.Submissions) != 1 || guest.Submissions[0].User != "alice" {
		t.Fatalf("guest should only see pre-freeze submissions: %+v", guest.Submissions)
	}
	if len(guest.Rank) != 1 || guest.Rank[0].User != "alice" {
		t.Fatalf("guest rank should only use pre-freeze submissions: %+v", guest.Rank)
	}

	adminDetail := decodeJSON[ContestDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, admin.ID), nil))
	if len(adminDetail.Submissions) != 2 {
		t.Fatalf("admin should see all submissions: %+v", adminDetail.Submissions)
	}
	if _, ok := rankByUser(adminDetail.Rank, "bob"); !ok {
		t.Fatalf("admin rank should include post-freeze submitter: %+v", adminDetail.Rank)
	}
}

func TestContestICPCRankUsesPenalty(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	for _, user := range []*models.User{&alice, &bob} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "ICPC", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	startAt := time.Now().Add(-time.Hour)
	contest := models.Contest{Title: "ICPC Round", Kind: "ICPC", StartAt: startAt, EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	contestID := contest.ID
	submissions := []models.Submission{
		{UserID: alice.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "wa", Status: "WA", Score: 0, Public: true, CreatedAt: startAt.Add(5 * time.Minute)},
		{UserID: alice.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "ac", Status: "AC", Score: 100, Public: true, CreatedAt: startAt.Add(10 * time.Minute)},
		{UserID: bob.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "ac", Status: "AC", Score: 100, Public: true, CreatedAt: startAt.Add(20 * time.Minute)},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)
	detail := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
	if len(detail.Rank) != 2 {
		t.Fatalf("rank size = %+v", detail.Rank)
	}
	if detail.Rank[0].User != "bob" || detail.Rank[0].AC != 1 || detail.Rank[0].Penalty != 20 {
		t.Fatalf("bob should win by lower penalty: %+v", detail.Rank)
	}
	if detail.Rank[1].User != "alice" || detail.Rank[1].AC != 1 || detail.Rank[1].Penalty != 30 {
		t.Fatalf("alice penalty should include wrong submission: %+v", detail.Rank)
	}
}

func TestAssignmentMembershipCreateUpdateAndVisibility(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	for _, user := range []*models.User{&admin, &alice, &bob} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	group := models.Group{Name: "team"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := db.Create(&models.GroupUser{GroupID: group.ID, UserID: alice.ID}).Error; err != nil {
		t.Fatalf("add alice to group: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Included", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}

	e := echo.New()
	Register(e, db)
	adminCookies := databaseSession(t, db, admin.ID)
	deadline := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	createBody := `{"title":"HW","desc":"body","endAt":"` + deadline + `","problems":[1000],"users":[],"groups":[` + strconv.FormatUint(uint64(group.ID), 10) + `]}`
	createRes := requestJSONWithCookies(e, http.MethodPost, "/api/assignments", adminCookies, createBody)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create assignment got %d body=%s", createRes.Code, createRes.Body.String())
	}
	created := decodeJSON[AssignmentDTO](t, createRes)
	if len(created.Groups) != 1 || created.Groups[0] != group.ID || len(created.Users) != 0 {
		t.Fatalf("created assignment members not returned: %+v", created)
	}

	aliceDetail := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, alice.ID), nil)
	if aliceDetail.Code != http.StatusOK {
		t.Fatalf("group member should see assignment, got %d body=%s", aliceDetail.Code, aliceDetail.Body.String())
	}
	bobDetail := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, bob.ID), nil)
	if bobDetail.Code != http.StatusNotFound {
		t.Fatalf("unassigned user should not see assignment, got %d body=%s", bobDetail.Code, bobDetail.Body.String())
	}

	updateBody := `{"title":"HW","desc":"body","endAt":"` + deadline + `","problems":[1000],"users":[` + strconv.FormatUint(uint64(bob.ID), 10) + `],"groups":[]}`
	updateRes := requestJSONWithCookies(e, http.MethodPatch, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), adminCookies, updateBody)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("update assignment got %d body=%s", updateRes.Code, updateRes.Body.String())
	}
	updated := decodeJSON[AssignmentDTO](t, updateRes)
	if len(updated.Users) != 1 || updated.Users[0] != bob.ID || len(updated.Groups) != 0 {
		t.Fatalf("updated assignment members not returned: %+v", updated)
	}
	aliceDetail = requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, alice.ID), nil)
	if aliceDetail.Code != http.StatusNotFound {
		t.Fatalf("removed group member should lose assignment access, got %d body=%s", aliceDetail.Code, aliceDetail.Body.String())
	}
	bobDetail = requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, bob.ID), nil)
	if bobDetail.Code != http.StatusOK {
		t.Fatalf("directly assigned user should see assignment, got %d body=%s", bobDetail.Code, bobDetail.Body.String())
	}
}

func TestDatabaseAdminInputValidation(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	now := time.Now().UTC()
	startAt := now.Add(time.Hour).Format(time.RFC3339)
	endAt := now.Add(2 * time.Hour).Format(time.RFC3339)
	deadline := now.Add(24 * time.Hour).Format(time.RFC3339)
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "create problem invalid mode",
			method: http.MethodPost,
			path:   "/api/problems",
			body:   `{"title":"Bad Mode","tags":[],"mode":"bad","timeMs":1000,"memoryMb":256}`,
		},
		{
			name:   "update problem invalid mode",
			method: http.MethodPatch,
			path:   "/api/problems/1000",
			body:   `{"title":"Visible","statement":"# Visible","tags":[],"visible":true,"mode":"bad","timeMs":1000,"memoryMb":256}`,
		},
		{
			name:   "create assignment duplicate problem",
			method: http.MethodPost,
			path:   "/api/assignments",
			body:   `{"title":"HW","desc":"","endAt":"` + deadline + `","problems":[1000,1000]}`,
		},
		{
			name:   "create assignment missing problem",
			method: http.MethodPost,
			path:   "/api/assignments",
			body:   `{"title":"HW","desc":"","endAt":"` + deadline + `","problems":[9999]}`,
		},
		{
			name:   "create contest invalid kind",
			method: http.MethodPost,
			path:   "/api/contests",
			body:   `{"title":"Round","desc":"","kind":"abc","startAt":"` + startAt + `","endAt":"` + endAt + `","problems":[1000]}`,
		},
		{
			name:   "create contest missing problem",
			method: http.MethodPost,
			path:   "/api/contests",
			body:   `{"title":"Round","desc":"","kind":"OI","startAt":"` + startAt + `","endAt":"` + endAt + `","problems":[9999]}`,
		},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			res := requestJSONWithCookies(e, item.method, item.path, cookies, item.body)
			if res.Code != http.StatusBadRequest {
				t.Fatalf("%s got %d body=%s", item.name, res.Code, res.Body.String())
			}
		})
	}
}

func TestDiscussionProblemTagsAreSoftAssociations(t *testing.T) {
	db := testWebDB(t)
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{
		ID:       1000,
		Title:    "Hidden",
		Tags:     datatypes.JSON([]byte(`["hidden"]`)),
		Visible:  false,
		Mode:     "default",
		TimeMS:   1000,
		MemoryMB: 256,
	}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	discussion := models.Discussion{
		Title:   "Hidden discussion",
		Content: "secret",
		UserID:  admin.ID,
		Tags:    datatypes.JSON([]byte(`["P1000"]`)),
		Locked:  false,
	}
	if err := db.Create(&discussion).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}

	e := echo.New()
	Register(e, db)

	studentCookies := databaseSession(t, db, student.ID)
	adminCookies := databaseSession(t, db, admin.ID)
	discussionBody := `{"title":"Hidden tagged discussion","content":"secret","tags":["P1000"]}`
	res := requestJSONWithCookies(e, http.MethodPost, "/api/discussion", studentCookies, discussionBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("student create hidden discussion got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPost, "/api/discussion", adminCookies, discussionBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("admin create hidden discussion got %d body=%s", res.Code, res.Body.String())
	}

	body := `{"content":"I should not see this"}`
	target := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10) + "/comments"
	res = requestJSONWithCookies(e, http.MethodPost, target, studentCookies, body)
	if res.Code != http.StatusCreated {
		t.Fatalf("student comment on hidden discussion got %d body=%s", res.Code, res.Body.String())
	}

	res = requestJSONWithCookies(e, http.MethodPost, target, adminCookies, body)
	if res.Code != http.StatusCreated {
		t.Fatalf("admin comment on hidden discussion got %d body=%s", res.Code, res.Body.String())
	}
}

func TestDatabaseDiscussionAuthorsUseNames(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	discussion := models.Discussion{
		Title:   "Named discussion",
		Content: "body",
		UserID:  admin.ID,
		Tags:    datatypes.JSON([]byte(`["general"]`)),
	}
	if err := db.Create(&discussion).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}
	comment := models.Comment{DiscussionID: discussion.ID, UserID: student.ID, Content: "reply"}
	if err := db.Create(&comment).Error; err != nil {
		t.Fatalf("create comment: %v", err)
	}

	e := echo.New()
	Register(e, db)

	list := decodeJSON[[]DiscussionDTO](t, requestOK(t, e, http.MethodGet, "/api/discussion", ""))
	if len(list) != 1 || list[0].Author != "admin" {
		t.Fatalf("discussion list author should be username: %+v", list)
	}

	target := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10)
	detail := decodeJSON[DiscussionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if detail.Discussion.Author != "admin" || len(detail.Comments) != 1 || detail.Comments[0].Author != "student" {
		t.Fatalf("discussion detail authors should be usernames: %+v", detail)
	}

	updated := requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, admin.ID), `{"title":"Named discussion","content":"body","tags":["general"],"pinned":true,"locked":false}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("update discussion got %d body=%s", updated.Code, updated.Body.String())
	}
	dto := decodeJSON[DiscussionDTO](t, updated)
	if dto.Author != "admin" {
		t.Fatalf("updated discussion author should be username: %+v", dto)
	}
}

func TestImageUploadUsesRelativeMediaPathsAndHeaders(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}

	e := echo.New()
	Register(e, db)
	studentCookies := databaseSession(t, db, student.ID)
	adminCookies := databaseSession(t, db, admin.ID)

	userImage := uploadImageForTest(t, e, "/api/uploads/images", studentCookies, "avatar.png", tinyPNG())
	if !strings.HasPrefix(userImage.URL, "/api/media/users/") || strings.Contains(userImage.URL, "://") {
		t.Fatalf("user image url should be a relative media path, got %q", userImage.URL)
	}
	res := requestWithCookies(e, http.MethodGet, userImage.URL, studentCookies, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("read user image got %d body=%s", res.Code, res.Body.String())
	}
	if cache := res.Header().Get(echo.HeaderCacheControl); cache != "public, max-age=31536000, immutable" {
		t.Fatalf("user image cache header = %q", cache)
	}
	res = requestWithCookiesAndReferer(e, http.MethodGet, userImage.URL, studentCookies, "https://evil.example/post")
	if res.Code != http.StatusForbidden {
		t.Fatalf("cross-site media request got %d body=%s", res.Code, res.Body.String())
	}

	problemImage := uploadImageForTest(t, e, "/api/problems/1000/assets/images", adminCookies, "statement.png", tinyPNG())
	if !strings.HasPrefix(problemImage.URL, "/api/media/problems/1000/") || strings.Contains(problemImage.URL, "://") || strings.Contains(problemImage.URL, "/assets/") {
		t.Fatalf("problem image url should be a relative media path, got %q", problemImage.URL)
	}
	rel := strings.TrimPrefix(problemImage.URL, "/api/media/problems/1000/")
	if _, err := os.Stat(filepath.Join(utils.UploadRoot(), "problems", "1000", "assets", filepath.FromSlash(rel))); err != nil {
		t.Fatalf("problem image should keep the existing object key convention: %v", err)
	}
	res = requestWithCookies(e, http.MethodGet, problemImage.URL, adminCookies, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("read problem image got %d body=%s", res.Code, res.Body.String())
	}
}

func TestSafeAssetZipNameRejectsUnsafeNames(t *testing.T) {
	name, ok := safeAssetZipName("data", "cases/1.in")
	if !ok || name != "data/cases/1.in" {
		t.Fatalf("safe nested asset name = %q, %v", name, ok)
	}
	for _, unsafe := range []string{"../evil", "cases/../../evil", "/absolute", `cases\..\evil`, "cases//1.in"} {
		if name, ok := safeAssetZipName("data", unsafe); ok {
			t.Fatalf("unsafe asset name %q accepted as %q", unsafe, name)
		}
	}
}

func TestLargeTextAssetIsNotEditableOnline(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	e := echo.New()
	Register(e, db)
	adminCookies := databaseSession(t, db, admin.ID)

	assets := uploadAssetForTest(t, e, "/api/problems/1000/assets/files", adminCookies, "data", "big.txt", strings.Repeat("x", maxEditableAssetBytes+1))
	if len(assets.Data) != 1 {
		t.Fatalf("expected uploaded asset, got %+v", assets)
	}
	if assets.Data[0].Editable {
		t.Fatalf("large text asset should not be editable: %+v", assets.Data[0])
	}

	target := "/api/problems/1000/assets/files/content?key=" + url.QueryEscape(assets.Data[0].Key)
	res := requestWithCookies(e, http.MethodGet, target, adminCookies, nil)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large asset content got %d body=%s", res.Code, res.Body.String())
	}
}

func TestAssetContentUpdateRejectsLargeBody(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	e := echo.New()
	Register(e, db)
	adminCookies := databaseSession(t, db, admin.ID)
	assets := uploadAssetForTest(t, e, "/api/problems/1000/assets/files", adminCookies, "judge", "main.cc", "int main(){}")
	if len(assets.Judge) != 1 || !assets.Judge[0].Editable {
		t.Fatalf("small judge asset should be editable: %+v", assets.Judge)
	}

	body := `{"key":"` + assets.Judge[0].Key + `","content":"` + strings.Repeat("x", maxEditableAssetBytes+1) + `"}`
	res := requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000/assets/files/content", adminCookies, body)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large asset update got %d body=%s", res.Code, res.Body.String())
	}
}

func uploadAssetForTest(t *testing.T, e *echo.Echo, target string, cookies []*http.Cookie, section string, name string, content string) ProblemAssets {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("section", section); err != nil {
		t.Fatalf("write section failed: %v", err)
	}
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write asset failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("upload asset got %d body=%s", res.Code, res.Body.String())
	}
	return decodeJSON[ProblemAssets](t, res)
}

func uploadImageForTest(t *testing.T, e *echo.Echo, target string, cookies []*http.Cookie, name string, content []byte) UploadResult {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write image failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("upload image got %d body=%s", res.Code, res.Body.String())
	}
	return decodeJSON[UploadResult](t, res)
}

func requestOK(t *testing.T, e *echo.Echo, method string, target string, role string) *httptest.ResponseRecorder {
	t.Helper()
	res := request(e, method, target, role, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("%s %s as %s got %d body=%s", method, target, role, res.Code, res.Body.String())
	}
	return res
}

func request(e *echo.Echo, method string, target string, role string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, body)
	if role != "" {
		req.Header.Set("X-DOJ-Role", role)
	}
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func requestJSON(e *echo.Echo, method string, target string, role string, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if role != "" {
		req.Header.Set("X-DOJ-Role", role)
	}
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func requestJSONWithCookies(e *echo.Echo, method string, target string, cookies []*http.Cookie, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func requestWithCookies(e *echo.Echo, method string, target string, cookies []*http.Cookie, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, body)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func requestWithCookiesAndReferer(e *echo.Echo, method string, target string, cookies []*http.Cookie, referer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Referer", referer)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func addCSRFHeader(req *http.Request, cookies []*http.Cookie) {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return
	}
	for _, cookie := range cookies {
		if cookie.Name == utils.CSRFCookie {
			req.Header.Set(utils.CSRFHeader, cookie.Value)
			return
		}
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func testWebDB(t *testing.T) *gorm.DB {
	t.Helper()
	utils.ResetCacheForTest()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "web.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}
	if err := models.EnsureDefaultLanguage(db); err != nil {
		t.Fatalf("seed language: %v", err)
	}
	return db
}

func databaseSession(t *testing.T, db *gorm.DB, userID uint) []*http.Cookie {
	t.Helper()
	_ = db
	e := echo.New()
	res := httptest.NewRecorder()
	ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), res)
	if err := utils.CreateUserSession(ctx, userID, time.Now()); err != nil {
		t.Fatalf("create session: %v", err)
	}
	return res.Result().Cookies()
}

func allowGuest(t *testing.T, db *gorm.DB) {
	t.Helper()
	settings := adminsvc.AdminSettings{
		SiteName:            "DOJ",
		Registration:        false,
		Guest:               true,
		DefaultPublicSource: false,
		Notice:              "",
	}
	if err := adminsvc.SaveSettings(db, settings); err != nil {
		t.Fatalf("enable guest access: %v", err)
	}
}

func decodeJSON[T any](t *testing.T, res *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(res.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response failed: %v body=%s", err, res.Body.String())
	}
	return value
}

func hasProblem(items []ProblemDTO, id uint) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func problemByID(items []ProblemDTO, id uint) (ProblemDTO, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return ProblemDTO{}, false
}

func hasSubmissionProblem(items []SubmissionDTO, id uint) bool {
	for _, item := range items {
		if item.ProblemID == id {
			return true
		}
	}
	return false
}

func userInRank(items []RankUserDTO, user string) bool {
	_, ok := rankByUser(items, user)
	return ok
}

func rankByUser(items []RankUserDTO, user string) (RankUserDTO, bool) {
	for _, item := range items {
		if item.User == user {
			return item, true
		}
	}
	return RankUserDTO{}, false
}

func countForDate(items []HeatCell, date string) int {
	for _, item := range items {
		if item.Date == date {
			return item.Count
		}
	}
	return 0
}

func nonzeroHeatmapDays(items []HeatCell) int {
	count := 0
	for _, item := range items {
		if item.Count > 0 {
			count++
		}
	}
	return count
}

func zipHasFile(reader *zip.Reader, name string) bool {
	for _, file := range reader.File {
		if file.Name == name {
			return true
		}
	}
	return false
}

func tinyPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0xf8, 0xff, 0xff, 0x3f,
		0x00, 0x05, 0xfe, 0x02, 0xfe, 0xa7, 0x35, 0x81,
		0x84, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
