package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/auth"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestDatabaseAdminCrud(t *testing.T) {
	db := testAdminDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	other := models.User{Name: "other", Mail: "other@example.com", Auth: "hash", Admin: true}
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	for _, user := range []*models.User{&admin, &other, &student} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	e := echo.New()
	Register(e, db, nil)
	adminCookies := databaseSession(t, db, admin.ID)

	res := requestWithCookies(e, http.MethodGet, "/api/admin/settings", adminCookies)
	if res.Code != http.StatusOK {
		t.Fatalf("settings got %d body=%s", res.Code, res.Body.String())
	}
	var gotSettings AdminSettings
	if err := json.Unmarshal(res.Body.Bytes(), &gotSettings); err != nil {
		t.Fatalf("decode settings: %v body=%s", err, res.Body.String())
	}
	if gotSettings.AllowRegistration || gotSettings.AllowGuestAccess || gotSettings.DefaultSubmissionPublic {
		t.Fatalf("site switches should default off: %+v", gotSettings)
	}
	res = requestWithCookies(e, http.MethodGet, "/api/admin/members", adminCookies)
	if res.Code != http.StatusOK {
		t.Fatalf("members got %d body=%s", res.Code, res.Body.String())
	}
	var members Members
	if err := json.Unmarshal(res.Body.Bytes(), &members); err != nil {
		t.Fatalf("decode members: %v body=%s", err, res.Body.String())
	}
	if user, ok := findUser(members, "student"); !ok || user.Groups == nil {
		t.Fatalf("user groups should be an empty array, got %+v", user)
	}
	if len(members.Users) != 3 || len(members.Groups) != 0 {
		t.Fatalf("members should return only user/group option data: %+v", members)
	}
	userPage := decodePage[User](t, requestWithCookies(e, http.MethodGet, "/api/admin/users?page=1&pageSize=2", adminCookies))
	if userPage.Total != 3 || len(userPage.Items) != 2 || userPage.Page != 1 || userPage.PageSize != 2 {
		t.Fatalf("admin users should be remotely paged: %+v", userPage)
	}
	var membersRaw map[string]json.RawMessage
	if err := json.Unmarshal(res.Body.Bytes(), &membersRaw); err != nil {
		t.Fatalf("decode raw members: %v", err)
	}
	for _, key := range []string{"settings", "languages", "judgers", "queue"} {
		if _, ok := membersRaw[key]; ok {
			t.Fatalf("members endpoint should not include %s: %s", key, res.Body.String())
		}
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/settings", adminCookies, `{"siteName":"DOJ","allowRegistration":true,"allowGuestAccess":true,"defaultSubmissionPublic":true,"notice":""}`)
	if res.Code != http.StatusOK {
		t.Fatalf("update settings got %d body=%s", res.Code, res.Body.String())
	}
	var saved AdminSettings
	if err := json.Unmarshal(res.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode settings: %v body=%s", err, res.Body.String())
	}
	if !saved.AllowRegistration || !saved.AllowGuestAccess || !saved.DefaultSubmissionPublic {
		t.Fatalf("site switches should roundtrip: %+v", saved)
	}
	var settingRows []models.Setting
	if err := db.Order("key asc").Find(&settingRows).Error; err != nil {
		t.Fatalf("read settings rows: %v", err)
	}
	keys := map[string]bool{}
	for _, row := range settingRows {
		keys[row.Key] = true
	}
	for _, key := range []string{"allow_guest_access", "allow_registration", "default_submission_public", "home_notice", "site_name"} {
		if !keys[key] {
			t.Fatalf("stored settings missing row %q: %+v", key, settingRows)
		}
	}
	if keys["site"] {
		t.Fatalf("stored settings should not be packed into one site row: %+v", settingRows)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/settings", adminCookies, `{"allowGuestAccess":false}`)
	if res.Code != http.StatusOK {
		t.Fatalf("patch one setting got %d body=%s", res.Code, res.Body.String())
	}
	if err := json.Unmarshal(res.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode patched settings: %v body=%s", err, res.Body.String())
	}
	if saved.SiteName != "DOJ" || !saved.AllowRegistration || saved.AllowGuestAccess || !saved.DefaultSubmissionPublic {
		t.Fatalf("partial settings patch should update one field and preserve the rest: %+v", saved)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/settings", adminCookies, `{"allowRegistration":false,"defaultSubmissionPublic":false}`)
	if res.Code != http.StatusOK {
		t.Fatalf("patch false settings got %d body=%s", res.Code, res.Body.String())
	}
	if err := json.Unmarshal(res.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode false patched settings: %v body=%s", err, res.Body.String())
	}
	if saved.SiteName != "DOJ" || saved.AllowRegistration || saved.AllowGuestAccess || saved.DefaultSubmissionPublic {
		t.Fatalf("false settings patch should apply false values and preserve unrelated fields: %+v", saved)
	}
	res = requestWithCookies(e, http.MethodGet, "/api/admin/settings", adminCookies)
	if res.Code != http.StatusOK {
		t.Fatalf("get settings got %d body=%s", res.Code, res.Body.String())
	}
	if err := json.Unmarshal(res.Body.Bytes(), &gotSettings); err != nil {
		t.Fatalf("decode get settings: %v body=%s", err, res.Body.String())
	}
	if gotSettings != saved {
		t.Fatalf("get settings should return current settings: got %+v want %+v", gotSettings, saved)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/settings", adminCookies, `{}`)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("empty settings patch got %d body=%s", res.Code, res.Body.String())
	}

	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/users/other", adminCookies, `{"role":"user","groups":[]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("demote admin got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/users/admin", adminCookies, `{"role":"user","groups":[]}`)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("demote last admin got %d body=%s", res.Code, res.Body.String())
	}
	studentCookies := databaseSession(t, db, student.ID)
	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/users/student/password", adminCookies, `{}`)
	if res.Code != http.StatusOK {
		t.Fatalf("reset password got %d body=%s", res.Code, res.Body.String())
	}
	var reset PasswordReset
	if err := json.Unmarshal(res.Body.Bytes(), &reset); err != nil || reset.Password == "" {
		t.Fatalf("bad reset response: %+v err=%v", reset, err)
	}
	studentRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range studentCookies {
		studentRequest.AddCookie(cookie)
	}
	studentContext := echo.New().NewContext(studentRequest, httptest.NewRecorder())
	if _, err := auth.UserFromCookie(db, studentContext, time.Now()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("student session survived admin password reset: %v", err)
	}

	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/groups", adminCookies, `{"name":"team-a"}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("create group got %d body=%s", res.Code, res.Body.String())
	}
	members = decodeMembers(t, res)
	group, ok := findGroup(members, "team-a")
	if !ok {
		t.Fatalf("created group missing: %+v", members.Groups)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/users/student", adminCookies, `{"role":"user","groups":[`+itoa(group.ID)+`]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("assign user group got %d body=%s", res.Code, res.Body.String())
	}
	members = decodeMembers(t, res)
	if user, ok := findUser(members, "student"); !ok || len(user.Groups) != 1 || user.Groups[0] != group.ID {
		t.Fatalf("updated user groups missing: %+v", members.Users)
	}
	groupPage := decodePage[Group](t, requestWithCookies(e, http.MethodGet, "/api/admin/groups?q=student&page=1&pageSize=10", adminCookies))
	if groupPage.Total != 1 || len(groupPage.Items) != 1 || groupPage.Items[0].ID != group.ID {
		t.Fatalf("admin group search should be remote and include user names: %+v", groupPage)
	}
	res = requestWithCookies(e, http.MethodGet, "/api/admin/members?q=stud&groups="+itoa(group.ID), adminCookies)
	if res.Code != http.StatusOK {
		t.Fatalf("search members got %d body=%s", res.Code, res.Body.String())
	}
	if err := json.Unmarshal(res.Body.Bytes(), &members); err != nil {
		t.Fatalf("decode searched members: %v body=%s", err, res.Body.String())
	}
	if len(members.Users) != 1 || members.Users[0].Name != "student" {
		t.Fatalf("member search should find student: %+v", members.Users)
	}
	if len(members.Groups) != 1 || members.Groups[0].ID != group.ID {
		t.Fatalf("member search should include selected group ids: %+v", members.Groups)
	}
	res = requestWithCookies(e, http.MethodGet, "/api/admin/members?users="+itoa(student.ID)+"&groups="+itoa(group.ID), adminCookies)
	if res.Code != http.StatusOK {
		t.Fatalf("default members with selected ids got %d body=%s", res.Code, res.Body.String())
	}
	if err := json.Unmarshal(res.Body.Bytes(), &members); err != nil {
		t.Fatalf("decode default members: %v body=%s", err, res.Body.String())
	}
	if len(members.Users) <= 1 || len(members.Groups) != 1 {
		t.Fatalf("member defaults should include suggestions plus selected ids: %+v", members)
	}
	if user, ok := findUser(members, "student"); !ok || user.ID != student.ID {
		t.Fatalf("member defaults should keep selected user: %+v", members.Users)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/groups/"+itoa(group.ID), adminCookies, `{"name":"team-b"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("update group got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/users", adminCookies, `{"name":"New_User","mail":"New_User@example.com","password":"password123","role":"user","groups":[`+itoa(group.ID)+`]}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("create user got %d body=%s", res.Code, res.Body.String())
	}
	members = decodeMembers(t, res)
	created, ok := findUser(members, "New_User")
	if !ok || created.Mail != "new_user@example.com" || len(created.Groups) != 1 || created.Groups[0] != group.ID {
		t.Fatalf("created user missing or malformed: %+v", members.Users)
	}
	var createdRow models.User
	if err := db.First(&createdRow, "name = ?", "New_User").Error; err != nil {
		t.Fatalf("read created user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(createdRow.Auth), []byte("password123")); err != nil {
		t.Fatalf("created user password hash mismatch: %v", err)
	}
	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/users", adminCookies, `{"name":"new_user","mail":"other@example.com","password":"password123","role":"user","groups":[]}`)
	if res.Code != http.StatusConflict {
		t.Fatalf("case folded duplicate user got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/users/new_user", adminCookies, `{"role":"user","groups":[`+itoa(group.ID)+`]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("case folded update user got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/users/new_user/password", adminCookies, `{}`)
	if res.Code != http.StatusOK {
		t.Fatalf("case folded reset user password got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/groups/"+itoa(group.ID), adminCookies, `{"name":"team-c","users":[`+itoa(student.ID)+`,`+itoa(created.ID)+`]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("update group users got %d body=%s", res.Code, res.Body.String())
	}
	members = decodeMembers(t, res)
	group, ok = findGroup(members, "team-c")
	if !ok || len(group.Users) != 2 || group.Users[0] != student.ID || group.Users[1] != created.ID {
		t.Fatalf("group side users not saved: %+v", members.Groups)
	}
	if user, ok := findUser(members, "student"); !ok || len(user.Groups) != 1 || user.Groups[0] != group.ID {
		t.Fatalf("student group not reflected after group edit: %+v", members.Users)
	}
	if user, ok := findUser(members, "New_User"); !ok || len(user.Groups) != 1 || user.Groups[0] != group.ID {
		t.Fatalf("new user group not reflected after group edit: %+v", members.Users)
	}
	assignment := models.Assignment{Title: "Homework", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if err := db.Create(&models.AssignmentGroup{AssignmentID: assignment.ID, GroupID: group.ID}).Error; err != nil {
		t.Fatalf("create assignment group: %v", err)
	}
	res = requestJSONWithCookies(e, http.MethodDelete, "/api/admin/users/new_user", adminCookies, `{}`)
	if res.Code != http.StatusOK {
		t.Fatalf("delete user got %d body=%s", res.Code, res.Body.String())
	}
	members = decodeMembers(t, res)
	if _, ok := findUser(members, "New_User"); ok {
		t.Fatalf("deleted user should be hidden from members: %+v", members.Users)
	}
	group, ok = findGroup(members, "team-c")
	if !ok || len(group.Users) != 1 || group.Users[0] != student.ID {
		t.Fatalf("deleted user should be hidden from group users: %+v", members.Groups)
	}
	res = requestJSONWithCookies(e, http.MethodDelete, "/api/admin/groups/"+itoa(group.ID), adminCookies, `{}`)
	if res.Code != http.StatusOK {
		t.Fatalf("delete group got %d body=%s", res.Code, res.Body.String())
	}
	members = decodeMembers(t, res)
	if _, ok := findGroup(members, "team-c"); ok {
		t.Fatalf("deleted group should be hidden from members: %+v", members.Groups)
	}
	if user, ok := findUser(members, "student"); !ok || len(user.Groups) != 0 {
		t.Fatalf("deleted group should not stay in user groups: %+v", members.Users)
	}
	var groupRows int64
	if err := db.Model(&models.Group{}).Where("id = ?", group.ID).Count(&groupRows).Error; err != nil {
		t.Fatalf("count groups: %v", err)
	}
	if groupRows != 0 {
		t.Fatalf("deleted group should be hard deleted")
	}
	var groupLinks int64
	if err := db.Model(&models.GroupUser{}).Where("group_id = ?", group.ID).Count(&groupLinks).Error; err != nil {
		t.Fatalf("count group users: %v", err)
	}
	if groupLinks != 0 {
		t.Fatalf("deleted group should remove group_users, got %d", groupLinks)
	}
	var assignmentGroupLinks int64
	if err := db.Model(&models.AssignmentGroup{}).Where("group_id = ?", group.ID).Count(&assignmentGroupLinks).Error; err != nil {
		t.Fatalf("count assignment groups: %v", err)
	}
	if assignmentGroupLinks != 0 {
		t.Fatalf("deleted group should remove assignment_groups, got %d", assignmentGroupLinks)
	}
	langBody := `{"id":"py","name":"Python","source":"main.py","image":"python:3.13","compile":"","compileMs":10000,"run":"python3 main.py"}`
	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/languages", adminCookies, langBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("create language got %d body=%s", res.Code, res.Body.String())
	}
	updateLang := `{"id":"python","name":"Python","source":"main.py","image":"python:3.13","compile":"","compileMs":10000,"run":"python3 main.py"}`
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/languages/py", adminCookies, updateLang)
	if res.Code != http.StatusOK {
		t.Fatalf("update language got %d body=%s", res.Code, res.Body.String())
	}
	languages := decodeLanguages(t, res)
	if _, ok := findLanguage(languages, "python"); !ok {
		t.Fatalf("updated language missing: %+v", languages)
	}

	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/judgers", adminCookies, `{"name":"linux-a","auth":"token-a"}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("create judger got %d body=%s", res.Code, res.Body.String())
	}
	judgers := decodeJudgers(t, res)
	judger, ok := findJudger(judgers, "linux-a")
	if !ok {
		t.Fatalf("created judger missing: %+v", judgers.Judgers)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/judgers/"+itoa(judger.ID), adminCookies, `{"name":"linux-b"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("update judger got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodDelete, "/api/admin/judgers/"+itoa(judger.ID), adminCookies, `{}`)
	if res.Code != http.StatusOK {
		t.Fatalf("delete judger got %d body=%s", res.Code, res.Body.String())
	}
}
