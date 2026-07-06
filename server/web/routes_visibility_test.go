package web

import (
	"encoding/json"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"net/http"
	"strconv"
	"testing"
	"time"
)

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
	contest := models.Contest{Title: "Round", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
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
	if err := db.Create(&models.Discussion{Title: "Student note", Content: "body", UserID: student.ID, Tags: datatypes.JSON([]byte(`["P1000"]`))}).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}

	e := echo.New()
	Register(e, db)

	profileRes := requestOK(t, e, http.MethodGet, "/api/users/student", "")
	profile := decodeJSON[UserProfile](t, profileRes)
	if hasSolvedProblem(profile.Solved.Items, hidden.ID) || hasActivityProblem(profile.Activities, hidden.ID) {
		t.Fatalf("guest profile leaked hidden problem: %+v", profile)
	}
	if !hasActivity(profile.Activities, "discussion", "Student note") {
		t.Fatalf("guest profile should include discussion activity: %+v", profile.Activities)
	}
	if profile.User.AC != 2 || profile.User.Submit != 6 {
		t.Fatalf("guest profile stats should include all activity: %+v", profile.User)
	}
	today := time.Now().Format("2006-01-02")
	if got := countForDate(profile.Heatmap, today); got != 6 {
		t.Fatalf("guest profile heatmap should include all submissions, got %d", got)
	}
	var rawProfile struct {
		Solved struct {
			Items []map[string]any `json:"items"`
		} `json:"solved"`
	}
	if err := json.Unmarshal(profileRes.Body.Bytes(), &rawProfile); err != nil {
		t.Fatalf("decode raw profile: %v", err)
	}
	if len(rawProfile.Solved.Items) > 0 {
		for _, key := range []string{"visible", "mode", "timeMs", "memoryMb", "discussions", "mine", "latest", "submission"} {
			if _, ok := rawProfile.Solved.Items[0][key]; ok {
				t.Fatalf("profile solved problem should not include list-only field %q: %+v", key, rawProfile.Solved.Items[0])
			}
		}
	}

	if res := requestOK(t, e, http.MethodGet, "/api/assignments", ""); len(decodeJSON[PageResult[AssignmentDTO]](t, res).Items) != 0 {
		t.Fatalf("guest assignment list should be empty, body=%s", res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), nil, nil); res.Code != http.StatusNotFound {
		t.Fatalf("guest assignment detail should be hidden, got %d body=%s", res.Code, res.Body.String())
	}
	outsiderCookies := databaseSession(t, db, outsider.ID)
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments", outsiderCookies, nil); len(decodeJSON[PageResult[AssignmentDTO]](t, res).Items) != 0 {
		t.Fatalf("unassigned assignment list should be empty, body=%s", res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), outsiderCookies, nil); res.Code != http.StatusNotFound {
		t.Fatalf("unassigned assignment detail should be hidden, got %d body=%s", res.Code, res.Body.String())
	}
	studentAssignment := decodeJSON[AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), databaseSession(t, db, student.ID), nil))
	if !hasProblem(studentAssignment.Problems, hidden.ID) {
		t.Fatalf("student assignment should include assigned hidden problem: %+v", studentAssignment)
	}
	if len(studentAssignment.Problems) != 2 || studentAssignment.Problems[0].Sort != "A" || studentAssignment.Problems[1].Sort != "B" {
		t.Fatalf("student assignment should expose collection problem order: %+v", studentAssignment.Problems)
	}
	if studentAssignment.Assignment.Done != 2 || studentAssignment.Assignment.Total != 2 {
		t.Fatalf("student assignment progress should include hidden problems in aggregate stats: %+v", studentAssignment.Assignment)
	}
	if len(studentAssignment.Progress) != 1 || studentAssignment.Progress[0].User != "student" {
		t.Fatalf("student assignment completion should include assigned student only: %+v", studentAssignment.Progress)
	}
	if len(studentAssignment.Progress[0].Problems) != 2 || studentAssignment.Progress[0].Problems[0].ProblemID != visible.ID || studentAssignment.Progress[0].Problems[1].ProblemID != hidden.ID {
		t.Fatalf("student assignment completion should expose assigned problem statuses: %+v", studentAssignment.Progress[0].Problems)
	}

	contestDetail := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
	if !hasProblem(contestDetail.Problems, hidden.ID) {
		t.Fatalf("running contest should include linked hidden problem: %+v", contestDetail)
	}
	if len(contestDetail.Problems) != 2 || contestDetail.Problems[0].Sort != "A" || contestDetail.Problems[1].Sort != "B" {
		t.Fatalf("guest contest should expose collection problem order: %+v", contestDetail.Problems)
	}

	adminProfileRes := requestWithCookies(e, http.MethodGet, "/api/users/student", databaseSession(t, db, admin.ID), nil)
	if adminProfileRes.Code != http.StatusOK {
		t.Fatalf("admin profile got %d body=%s", adminProfileRes.Code, adminProfileRes.Body.String())
	}
	adminProfile := decodeJSON[UserProfile](t, adminProfileRes)
	if !hasSolvedProblem(adminProfile.Solved.Items, hidden.ID) || !hasActivityProblem(adminProfile.Activities, hidden.ID) {
		t.Fatalf("admin profile should include hidden problem: %+v", adminProfile)
	}
	if adminProfile.User.AC != 2 || adminProfile.User.Submit != 6 {
		t.Fatalf("admin profile stats should include hidden activity: %+v", adminProfile.User)
	}
	if got := countForDate(adminProfile.Heatmap, today); got != 6 {
		t.Fatalf("admin profile heatmap should include hidden submissions, got %d", got)
	}
}
