package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/auth"
	"github.com/doveccl/doj/server/cache"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func requestJSONWithCookies(e *echo.Echo, method string, target string, cookies []*http.Cookie, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
		if cookie.Name == auth.CSRFCookie {
			req.Header.Set(auth.CSRFHeader, cookie.Value)
		}
	}
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func requestWithCookies(e *echo.Echo, method string, target string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func decodeMembers(t *testing.T, res *httptest.ResponseRecorder) Members {
	t.Helper()
	var members Members
	if err := json.Unmarshal(res.Body.Bytes(), &members); err != nil {
		t.Fatalf("decode members failed: %v body=%s", err, res.Body.String())
	}
	return members
}

func decodePage[T any](t *testing.T, res *httptest.ResponseRecorder) PageResult[T] {
	t.Helper()
	var page PageResult[T]
	if err := json.Unmarshal(res.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page failed: %v body=%s", err, res.Body.String())
	}
	return page
}

func decodeLanguages(t *testing.T, res *httptest.ResponseRecorder) []Language {
	t.Helper()
	var languages []Language
	if err := json.Unmarshal(res.Body.Bytes(), &languages); err != nil {
		t.Fatalf("decode languages failed: %v body=%s", err, res.Body.String())
	}
	return languages
}

func decodeJudgers(t *testing.T, res *httptest.ResponseRecorder) Judgers {
	t.Helper()
	var judgers Judgers
	if err := json.Unmarshal(res.Body.Bytes(), &judgers); err != nil {
		t.Fatalf("decode judgers failed: %v body=%s", err, res.Body.String())
	}
	return judgers
}

func findGroup(members Members, name string) (Group, bool) {
	for _, item := range members.Groups {
		if item.Name == name {
			return item, true
		}
	}
	return Group{}, false
}

func findUser(members Members, name string) (User, bool) {
	for _, item := range members.Users {
		if item.Name == name {
			return item, true
		}
	}
	return User{}, false
}

func findLanguage(languages []Language, id string) (Language, bool) {
	for _, item := range languages {
		if item.ID == id {
			return item, true
		}
	}
	return Language{}, false
}

func findJudger(judgers Judgers, name string) (Judger, bool) {
	for _, item := range judgers.Judgers {
		if item.Name == name {
			return item, true
		}
	}
	return Judger{}, false
}

func testAdminDB(t *testing.T) *gorm.DB {
	t.Helper()
	startRedis(t)
	cache.ResetForTest()
	t.Cleanup(cache.ResetForTest)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "admin.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func startRedis(t *testing.T) {
	t.Helper()
	server := miniredis.RunT(t)
	t.Setenv("REDIS", "redis://"+server.Addr()+"/0")
}

func databaseSession(t *testing.T, userID uint) []*http.Cookie {
	t.Helper()
	e := echo.New()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := e.NewContext(req, res)
	if err := auth.CreateUserSession(ctx, userID, time.Now()); err != nil {
		t.Fatalf("create session: %v", err)
	}
	return res.Result().Cookies()
}

func itoa(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
