package public

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestAdminCanRewriteOrDeleteUsedAssignment(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	other := models.User{Name: "other", Mail: "other@example.com", Auth: "hash"}
	for _, user := range []*models.User{&admin, &student, &other} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	problems := []models.Problem{
		{ID: 1000, Title: "A", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1001, Title: "B", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256},
	}
	if err := db.Create(&problems).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	assignment := models.Assignment{Title: "Homework", EndAt: now.Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: problems[0].ID, Sort: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: assignment.ID, UserID: student.ID}).Error; err != nil {
		t.Fatal(err)
	}
	assignmentID := assignment.ID
	if err := db.Create(&models.Submission{UserID: student.ID, ProblemID: problems[0].ID, AssignmentID: &assignmentID, Language: "cpp", Code: "code", Status: "AC", Score: 100}).Error; err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	path := "/api/assignments/" + strconv.FormatUint(uint64(assignment.ID), 10)
	body := func(end time.Time, problemID uint, userID uint) string {
		return `{"title":"Homework","endAt":"` + end.Format(time.RFC3339) + `","problems":[{"id":` + strconv.FormatUint(uint64(problemID), 10) + `,"sort":"A"}],"users":[` + strconv.FormatUint(uint64(userID), 10) + `],"groups":[]}`
	}
	extended := assignment.EndAt.Add(time.Hour)
	if res := requestJSONWithCookies(e, http.MethodPatch, path, cookies, body(extended, problems[0].ID, student.ID)); res.Code != http.StatusOK {
		t.Fatalf("deadline extension got %d body=%s", res.Code, res.Body.String())
	}
	for name, payload := range map[string]string{
		"problem":  body(extended, problems[1].ID, student.ID),
		"member":   body(extended, problems[0].ID, other.ID),
		"deadline": body(assignment.EndAt, problems[0].ID, student.ID),
	} {
		if res := requestJSONWithCookies(e, http.MethodPatch, path, cookies, payload); res.Code != http.StatusOK {
			t.Fatalf("%s rewrite got %d body=%s", name, res.Code, res.Body.String())
		}
	}
	if res := requestWithCookies(e, http.MethodDelete, path, cookies, nil); res.Code != http.StatusNoContent {
		t.Fatalf("used assignment delete got %d body=%s", res.Code, res.Body.String())
	}
}

func TestAssignmentAcceptsDraftProblems(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	problem := models.Problem{ID: 1000, Title: "Private draft", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Setenv("STORAGE", root)
	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	pastBody := `{"title":"Past","endAt":"` + time.Now().Add(-time.Hour).Format(time.RFC3339) + `","problems":[],"users":[],"groups":[]}`
	if res := requestJSONWithCookies(e, http.MethodPost, "/api/assignments", cookies, pastBody); res.Code != http.StatusBadRequest {
		t.Fatalf("create past assignment got %d body=%s", res.Code, res.Body.String())
	}
	body := `{"title":"Private","endAt":"` + time.Now().Add(time.Hour).Format(time.RFC3339) + `","problems":[{"id":1000,"sort":"A"}],"users":[],"groups":[]}`
	if res := requestJSONWithCookies(e, http.MethodPost, "/api/assignments", cookies, body); res.Code != http.StatusCreated {
		t.Fatalf("create assignment with draft got %d body=%s", res.Code, res.Body.String())
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
	createBody := `{"title":"HW","endAt":"` + deadline + `","problems":[{"id":1000,"sort":"A"}],"users":[],"groups":[` + strconv.FormatUint(uint64(group.ID), 10) + `]}`
	createRes := requestJSONWithCookies(e, http.MethodPost, "/api/assignments", adminCookies, createBody)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create assignment got %d body=%s", createRes.Code, createRes.Body.String())
	}
	created := decodeJSON[contract.CreatedID](t, createRes)
	createdDetail := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), adminCookies, nil))
	if len(createdDetail.Assignment.Groups) != 1 || createdDetail.Assignment.Groups[0] != group.ID || len(createdDetail.Assignment.Users) != 0 {
		t.Fatalf("created assignment members not persisted: %+v", createdDetail.Assignment)
	}
	adminList := decodeJSON[contract.Page[map[string]any]](t, requestWithCookies(e, http.MethodGet, "/api/assignments", adminCookies, nil)).Items
	if len(adminList) != 1 {
		t.Fatalf("admin assignment list got %+v", adminList)
	}
	if _, ok := adminList[0]["users"]; ok {
		t.Fatalf("assignment list should not include user members: %+v", adminList[0])
	}
	if _, ok := adminList[0]["groups"]; ok {
		t.Fatalf("assignment list should not include group members: %+v", adminList[0])
	}

	aliceDetail := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, alice.ID), nil)
	if aliceDetail.Code != http.StatusOK {
		t.Fatalf("group member should see assignment, got %d body=%s", aliceDetail.Code, aliceDetail.Body.String())
	}
	bobDetail := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, bob.ID), nil)
	if bobDetail.Code != http.StatusNotFound {
		t.Fatalf("unassigned user should not see assignment, got %d body=%s", bobDetail.Code, bobDetail.Body.String())
	}

	updateBody := `{"title":"HW","endAt":"` + deadline + `","problems":[{"id":1000,"sort":"A"}],"users":[` + strconv.FormatUint(uint64(bob.ID), 10) + `],"groups":[]}`
	updateRes := requestJSONWithCookies(e, http.MethodPatch, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), adminCookies, updateBody)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("update assignment got %d body=%s", updateRes.Code, updateRes.Body.String())
	}
	updated := decodeJSON[contract.CreatedID](t, updateRes)
	if updated.ID != created.ID {
		t.Fatalf("updated assignment should return id: %+v", updated)
	}
	updatedDetail := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), adminCookies, nil))
	if len(updatedDetail.Assignment.Users) != 1 || updatedDetail.Assignment.Users[0] != bob.ID || len(updatedDetail.Assignment.Groups) != 0 {
		t.Fatalf("updated assignment members not persisted: %+v", updatedDetail.Assignment)
	}
	aliceDetail = requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, alice.ID), nil)
	if aliceDetail.Code != http.StatusNotFound {
		t.Fatalf("removed group member should lose assignment access, got %d body=%s", aliceDetail.Code, aliceDetail.Body.String())
	}
	bobDetail = requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, bob.ID), nil)
	if bobDetail.Code != http.StatusOK {
		t.Fatalf("directly assigned user should see assignment, got %d body=%s", bobDetail.Code, bobDetail.Body.String())
	}
	if res := requestWithCookies(e, http.MethodDelete, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), adminCookies, nil); res.Code != http.StatusNoContent {
		t.Fatalf("delete unused assignment got %d body=%s", res.Code, res.Body.String())
	}
	for name, model := range map[string]any{
		"problems": &models.AssignmentProblem{},
		"users":    &models.AssignmentUser{},
		"groups":   &models.AssignmentGroup{},
	} {
		var count int64
		if err := db.Model(model).Where("assignment_id = ?", created.ID).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("deleted assignment kept %s: count=%d err=%v", name, count, err)
		}
	}
}

func TestAssignmentProgressVisibleToAssignedUsers(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	outsider := models.User{Name: "outsider", Mail: "outsider@example.com", Auth: "hash"}
	for _, user := range []*models.User{&admin, &alice, &bob, &outsider} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "Included", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	assignment := models.Assignment{Title: "HW", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	for _, row := range []any{
		&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: problem.ID, Sort: "A"},
		&models.AssignmentUser{AssignmentID: assignment.ID, UserID: alice.ID},
		&models.AssignmentUser{AssignmentID: assignment.ID, UserID: bob.ID},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create assignment fixture: %v", err)
		}
	}
	assignmentID := assignment.ID
	for _, row := range []*models.Submission{
		{UserID: alice.ID, ProblemID: problem.ID, AssignmentID: &assignmentID, Language: "cpp", Code: "alice", Status: "AC", Score: 100},
		{UserID: bob.ID, ProblemID: problem.ID, AssignmentID: &assignmentID, Language: "cpp", Code: "bob", Status: "WA", Score: 30},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)
	target := "/api/assignments/" + strconv.FormatUint(uint64(assignment.ID), 10)
	if res := requestWithCookies(e, http.MethodGet, target, nil, nil); res.Code != http.StatusNotFound {
		t.Fatalf("guest detail got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, outsider.ID), nil); res.Code != http.StatusNotFound {
		t.Fatalf("unassigned detail got %d body=%s", res.Code, res.Body.String())
	}
	aliceDetail := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, alice.ID), nil))
	if len(aliceDetail.Progress) != 2 || aliceDetail.Progress[0].User != alice.Name || aliceDetail.Progress[1].User != bob.Name {
		t.Fatalf("assigned user should see assignment progress: %+v", aliceDetail.Progress)
	}
	adminDetail := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, admin.ID), nil))
	if len(adminDetail.Progress) != 2 || adminDetail.Progress[0].User != alice.Name || adminDetail.Progress[1].User != bob.Name {
		t.Fatalf("admin should see all assigned users' progress: %+v", adminDetail.Progress)
	}
}

func TestEndedAssignmentKeepsMemberReviewAndPracticeAccess(t *testing.T) {
	db := testWebDB(t)
	root := t.TempDir()
	t.Setenv("STORAGE", root)
	member := models.User{Name: "member", Mail: "member@example.com", Auth: "hash"}
	outsider := models.User{Name: "outsider", Mail: "outsider@example.com", Auth: "hash"}
	for _, user := range []*models.User{&member, &outsider} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "Private homework", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	assignment := models.Assignment{Title: "Ended HW", EndAt: time.Now().Add(-time.Hour)}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatal(err)
	}
	writeReadyProblemFiles(t, db, root, problem.ID, problem.Title)
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{
		&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: problem.ID, Sort: "A"},
		&models.AssignmentUser{AssignmentID: assignment.ID, UserID: member.ID},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	e := echo.New()
	Register(e, db)
	memberCookies := databaseSession(t, db, member.ID)
	detail := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), memberCookies, nil))
	if len(detail.Problems) != 1 || detail.Problems[0].ID != problem.ID {
		t.Fatalf("ended assignment hid its problem from a member: %+v", detail)
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/problems/1000", memberCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("member review got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/problems/1000", databaseSession(t, db, outsider.ID), nil); res.Code != http.StatusNotFound {
		t.Fatalf("outsider review got %d body=%s", res.Code, res.Body.String())
	}
	res := requestJSONWithCookies(e, http.MethodPost, "/api/submissions", memberCookies, `{"problemId":1000,"language":"cpp","code":"int main(){}","public":false}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("post-assignment practice got %d body=%s", res.Code, res.Body.String())
	}
	created := decodeJSON[contract.CreatedID](t, res)
	var submission models.Submission
	if err := db.First(&submission, created.ID).Error; err != nil || submission.AssignmentID != nil || submission.ContestID != nil {
		t.Fatalf("practice submission kept an ended context: %+v err=%v", submission, err)
	}
}
