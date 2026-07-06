package judger

import (
	"archive/zip"
	"github.com/alicebob/miniredis/v2"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustWriteFile(t *testing.T, file string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", file, err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", file, err)
	}
}

func zipHasFile(reader *zip.Reader, name string) bool {
	for _, file := range reader.File {
		if file.Name == name {
			return true
		}
	}
	return false
}

func seedTaskData(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Setenv("STORAGE", t.TempDir())
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	for key, content := range map[string]string{
		"problems/1000/data/1.in":  "1 2\n",
		"problems/1000/data/1.out": "3\n",
	} {
		if err := store.Put(t.Context(), key, strings.NewReader(content), int64(len(content)), "text/plain"); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}
	if err := db.Create(&models.Language{ID: "cpp", Name: "C++", Source: "main.cc", Image: "gcc:14", Compile: "g++ main.cc -o main", Run: "./main"}).Error; err != nil {
		t.Fatalf("create language: %v", err)
	}
	if err := db.Create(&models.Problem{ID: 1000, Title: "A+B", Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
}

func judgerJSON(e *echo.Echo, target string, token string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func newJudgerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	startRedis(t)
	utils.ResetCacheForTest()
	t.Cleanup(utils.ResetCacheForTest)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "judger.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}
	return db
}

func startRedis(t *testing.T) {
	t.Helper()
	server := miniredis.RunT(t)
	t.Setenv("REDIS", "redis://"+server.Addr()+"/0")
}

func remoteJudgerContext(t *testing.T, api *API, token string) echo.Context {
	t.Helper()
	c, err := remoteJudgerContextMaybe(api, token)
	if err != nil {
		t.Fatalf("auth token %q: %v", token, err)
	}
	return c
}

func remoteJudgerContextAllowError(t *testing.T, api *API, token string) error {
	t.Helper()
	_, err := remoteJudgerContextMaybe(api, token)
	return err
}

func remoteJudgerContextMaybe(api *API, token string) (echo.Context, error) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/judger/lease", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("Authorization", "Bearer "+token)
	c := e.NewContext(req, httptest.NewRecorder())
	return c, api.auth(func(echo.Context) error { return nil })(c)
}

func localJudgerContext(t *testing.T, api *API) echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/judger/lease", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	c := e.NewContext(req, httptest.NewRecorder())
	if err := api.auth(func(echo.Context) error { return nil })(c); err != nil {
		t.Fatalf("auth local judger: %v", err)
	}
	return c
}

func expectHTTPStatus(t *testing.T, err error, status int) {
	t.Helper()
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != status {
		t.Fatalf("error = %#v, want HTTP %d", err, status)
	}
}
