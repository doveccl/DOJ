package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/sqlite"
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
	Register(e, db)
	adminCookies := databaseSession(t, admin.ID)

	res := requestJSONWithCookies(e, http.MethodPatch, "/api/admin/users/other", adminCookies, `{"role":"user","groups":[]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("demote admin got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/users/admin", adminCookies, `{"role":"user","groups":[]}`)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("demote last admin got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/users/student/password", adminCookies, `{}`)
	if res.Code != http.StatusOK {
		t.Fatalf("reset password got %d body=%s", res.Code, res.Body.String())
	}
	var reset PasswordReset
	if err := json.Unmarshal(res.Body.Bytes(), &reset); err != nil || reset.Password == "" {
		t.Fatalf("bad reset response: %+v err=%v", reset, err)
	}

	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/groups", adminCookies, `{"name":"team-a"}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("create group got %d body=%s", res.Code, res.Body.String())
	}
	overview := decodeOverview(t, res)
	group, ok := findGroup(overview, "team-a")
	if !ok {
		t.Fatalf("created group missing: %+v", overview.Groups)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/users/student", adminCookies, `{"role":"user","groups":[`+itoa(group.ID)+`]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("assign user group got %d body=%s", res.Code, res.Body.String())
	}
	overview = decodeOverview(t, res)
	if user, ok := findUser(overview, "student"); !ok || len(user.Groups) != 1 || user.Groups[0] != group.ID {
		t.Fatalf("updated user groups missing: %+v", overview.Users)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/groups/"+itoa(group.ID), adminCookies, `{"name":"team-b"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("update group got %d body=%s", res.Code, res.Body.String())
	}

	langBody := `{"id":"py","name":"Python","source":"main.py","dockerfile":"FROM python:3.13\nCMD [\"python3\", \"/src/main.py\"]"}`
	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/languages", adminCookies, langBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("create language got %d body=%s", res.Code, res.Body.String())
	}
	updateLang := `{"id":"python","name":"Python","source":"main.py","dockerfile":"FROM python:3.13\nCMD [\"python3\", \"/src/main.py\"]"}`
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/admin/languages/py", adminCookies, updateLang)
	if res.Code != http.StatusOK {
		t.Fatalf("update language got %d body=%s", res.Code, res.Body.String())
	}
	overview = decodeOverview(t, res)
	if _, ok := findLanguage(overview, "python"); !ok {
		t.Fatalf("updated language missing: %+v", overview.Languages)
	}

	res = requestJSONWithCookies(e, http.MethodPost, "/api/admin/judgers", adminCookies, `{"name":"linux-a","auth":"token-a"}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("create judger got %d body=%s", res.Code, res.Body.String())
	}
	overview = decodeOverview(t, res)
	judger, ok := findJudger(overview, "linux-a")
	if !ok {
		t.Fatalf("created judger missing: %+v", overview.Judgers)
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

func requestJSON(e *echo.Echo, method string, target string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DOJ-Role", "admin")
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func requestJSONWithCookies(e *echo.Echo, method string, target string, cookies []*http.Cookie, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
		if cookie.Name == utils.CSRFCookie {
			req.Header.Set(utils.CSRFHeader, cookie.Value)
		}
	}
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func decodeOverview(t *testing.T, res *httptest.ResponseRecorder) Overview {
	t.Helper()
	var overview Overview
	if err := json.Unmarshal(res.Body.Bytes(), &overview); err != nil {
		t.Fatalf("decode overview failed: %v body=%s", err, res.Body.String())
	}
	return overview
}

func findGroup(overview Overview, name string) (Group, bool) {
	for _, item := range overview.Groups {
		if item.Name == name {
			return item, true
		}
	}
	return Group{}, false
}

func findUser(overview Overview, name string) (User, bool) {
	for _, item := range overview.Users {
		if item.Name == name {
			return item, true
		}
	}
	return User{}, false
}

func findLanguage(overview Overview, id string) (Language, bool) {
	for _, item := range overview.Languages {
		if item.ID == id {
			return item, true
		}
	}
	return Language{}, false
}

func findJudger(overview Overview, name string) (Judger, bool) {
	for _, item := range overview.Judgers {
		if item.Name == name {
			return item, true
		}
	}
	return Judger{}, false
}

func testAdminDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "admin.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func databaseSession(t *testing.T, userID uint) []*http.Cookie {
	t.Helper()
	e := echo.New()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := e.NewContext(req, res)
	if err := utils.CreateUserSession(ctx, userID, time.Now()); err != nil {
		t.Fatalf("create session: %v", err)
	}
	return res.Result().Cookies()
}

func itoa(id uint) string {
	return strconvFormatUint(uint64(id))
}

func strconvFormatUint(value uint64) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}
