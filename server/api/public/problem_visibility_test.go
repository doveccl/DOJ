package public

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestDeletedRunningContestDoesNotPublishHiddenResults(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	user := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, row := range []*models.User{&user, &admin} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	publicProblem := models.Problem{ID: 1000, Title: "Public", Tags: datatypes.JSON([]byte(`["public-tag"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	hiddenProblem := models.Problem{ID: 1001, Title: "Hidden", Tags: datatypes.JSON([]byte(`["hidden-tag"]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	contestProblem := models.Problem{ID: 1002, Title: "Contest", Tags: datatypes.JSON([]byte(`["contest-tag"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	for _, row := range []*models.Problem{&publicProblem, &hiddenProblem, &contestProblem} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create problem: %v", err)
		}
	}
	contest := models.Contest{Title: "Running", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: contestProblem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	contestID := contest.ID
	submission := models.Submission{UserID: user.ID, ProblemID: contestProblem.ID, ContestID: &contestID, Language: "cpp", Code: "answer", Status: "AC", Score: 100, Public: true}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	e := echo.New()
	Register(e, db)
	userCookies := databaseSession(t, db, user.ID)
	adminCookies := databaseSession(t, db, admin.ID)
	viewerCookies := [][]*http.Cookie{nil, userCookies}
	for index, cookies := range viewerCookies {
		tags := decodeJSON[[]string](t, requestWithCookies(e, http.MethodGet, "/api/tags?kind=problem", cookies, nil))
		if !hasTag(tags, "public-tag") || hasTag(tags, "hidden-tag") || hasTag(tags, "contest-tag") {
			t.Fatalf("viewer %d problem tags leaked hidden context: %+v", index, tags)
		}
		problems := decodePageItems[contract.Problem](t, requestWithCookies(e, http.MethodGet, "/api/problems", cookies, nil))
		if !hasProblem(problems, publicProblem.ID) || hasProblem(problems, hiddenProblem.ID) || hasProblem(problems, contestProblem.ID) {
			t.Fatalf("viewer %d problem list leaked hidden context: %+v", index, problems)
		}
		detail := decodeJSON[contract.SubmissionDetail](t, requestWithCookies(e, http.MethodGet, "/api/submissions/"+strconv.FormatUint(uint64(submission.ID), 10), cookies, nil))
		if detail.Submission.Status != "pending" || detail.Submission.Score != 0 {
			t.Fatalf("viewer %d saw running OI result: %+v", index, detail.Submission)
		}
		profile := decodeJSON[contract.UserProfile](t, requestWithCookies(e, http.MethodGet, "/api/users/student", cookies, nil))
		if profile.User.AC != 0 {
			t.Fatalf("viewer %d inferred running OI result from profile: %+v", index, profile.User)
		}
	}
	adminTags := decodeJSON[[]string](t, requestWithCookies(e, http.MethodGet, "/api/tags?kind=problem", adminCookies, nil))
	if !hasTag(adminTags, "public-tag") || !hasTag(adminTags, "hidden-tag") || !hasTag(adminTags, "contest-tag") {
		t.Fatalf("admin should see all problem tags: %+v", adminTags)
	}
	adminSubmission := decodeJSON[contract.SubmissionDetail](t, requestWithCookies(e, http.MethodGet, "/api/submissions/"+strconv.FormatUint(uint64(submission.ID), 10), adminCookies, nil))
	if adminSubmission.Submission.Status != "AC" || adminSubmission.Submission.Score != 100 {
		t.Fatalf("admin result was hidden: %+v", adminSubmission.Submission)
	}
	adminProfile := decodeJSON[contract.UserProfile](t, requestWithCookies(e, http.MethodGet, "/api/users/student", adminCookies, nil))
	if adminProfile.User.AC != 1 {
		t.Fatalf("admin profile result was hidden: %+v", adminProfile.User)
	}

	if err := db.Delete(&contest).Error; err != nil {
		t.Fatalf("delete contest: %v", err)
	}
	for index, cookies := range viewerCookies {
		tags := decodeJSON[[]string](t, requestWithCookies(e, http.MethodGet, "/api/tags?kind=problem", cookies, nil))
		if !hasTag(tags, "public-tag") || !hasTag(tags, "contest-tag") || hasTag(tags, "hidden-tag") {
			t.Fatalf("viewer %d tags still affected by deleted contest: %+v", index, tags)
		}
		problems := decodePageItems[contract.Problem](t, requestWithCookies(e, http.MethodGet, "/api/problems", cookies, nil))
		if !hasProblem(problems, contestProblem.ID) {
			t.Fatalf("viewer %d list still hid deleted-contest problem: %+v", index, problems)
		}
		problemDetail := decodeJSON[contract.Problem](t, requestWithCookies(e, http.MethodGet, "/api/problems/"+strconv.FormatUint(uint64(contestProblem.ID), 10), cookies, nil))
		if !hasTag(problemDetail.Tags, "contest-tag") {
			t.Fatalf("viewer %d detail still treated deleted contest as unfinished: %+v", index, problemDetail)
		}
		detail := decodeJSON[contract.SubmissionDetail](t, requestWithCookies(e, http.MethodGet, "/api/submissions/"+strconv.FormatUint(uint64(submission.ID), 10), cookies, nil))
		if detail.Submission.Status != "pending" || detail.Submission.Score != 0 {
			t.Fatalf("viewer %d saw a result because its running contest was deleted: %+v", index, detail.Submission)
		}
		profile := decodeJSON[contract.UserProfile](t, requestWithCookies(e, http.MethodGet, "/api/users/student", cookies, nil))
		if profile.User.AC != 0 {
			t.Fatalf("viewer %d inferred a result because its running contest was deleted: %+v", index, profile.User)
		}
	}
	if err := db.Model(&contestProblem).Update("visible", false).Error; err != nil {
		t.Fatalf("hide deleted-contest problem: %v", err)
	}
	for index, cookies := range viewerCookies {
		res := requestWithCookies(e, http.MethodGet, "/api/problems/"+strconv.FormatUint(uint64(contestProblem.ID), 10), cookies, nil)
		if res.Code != http.StatusNotFound {
			t.Fatalf("viewer %d saw hidden problem through deleted running contest: %d %s", index, res.Code, res.Body.String())
		}
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
	profile := decodeJSON[contract.UserProfile](t, profileRes)
	if hasSolvedProblem(profile.Solved.Items, hidden.ID) || hasActivityProblem(profile.Activities, hidden.ID) {
		t.Fatalf("guest profile leaked hidden problem: %+v", profile)
	}
	if !hasActivity(profile.Activities, "discussion", "Student note") {
		t.Fatalf("soft-linked discussion should remain visible on profile: %+v", profile.Activities)
	}
	if profile.User.AC != 2 || profile.User.Submit != 2 {
		t.Fatalf("guest profile stats should exclude private assignment and hidden contest rows: %+v", profile.User)
	}
	today := time.Now().Format("2006-01-02")
	if got := countForDate(profile.Heatmap, today); got != 2 {
		t.Fatalf("guest profile heatmap should exclude private assignment and hidden contest rows, got %d", got)
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

	if res := requestOK(t, e, http.MethodGet, "/api/assignments", ""); len(decodeJSON[contract.Page[contract.Assignment]](t, res).Items) != 0 {
		t.Fatalf("guest assignment list should be empty, body=%s", res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), nil, nil); res.Code != http.StatusNotFound {
		t.Fatalf("guest assignment detail should be hidden, got %d body=%s", res.Code, res.Body.String())
	}
	outsiderCookies := databaseSession(t, db, outsider.ID)
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments", outsiderCookies, nil); len(decodeJSON[contract.Page[contract.Assignment]](t, res).Items) != 0 {
		t.Fatalf("unassigned assignment list should be empty, body=%s", res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), outsiderCookies, nil); res.Code != http.StatusNotFound {
		t.Fatalf("unassigned assignment detail should be hidden, got %d body=%s", res.Code, res.Body.String())
	}
	studentAssignment := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), databaseSession(t, db, student.ID), nil))
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

	contestDetail := decodeJSON[contract.ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
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
	adminProfile := decodeJSON[contract.UserProfile](t, adminProfileRes)
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
