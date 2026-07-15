package public

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"github.com/doveccl/doj/server/auth"
	"github.com/doveccl/doj/server/cache"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	problemdata "github.com/doveccl/doj/server/problem"
	"github.com/doveccl/doj/server/settings"
	"github.com/doveccl/doj/server/storage"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func writeReadyProblemFiles(t *testing.T, db *gorm.DB, _ string, id uint, _ string) {
	t.Helper()
	base := t.TempDir()
	for name, content := range map[string]string{
		"data/1.in":  "1 2\n",
		"data/1.out": "3\n",
	} {
		file := filepath.Join(base, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	packagePath := filepath.Join(t.TempDir(), "package.zip")
	item, err := problemdata.Build(base, packagePath, nil)
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(t.Context(), problemdata.ObjectKey(id, item.Hash), file, item.Size, "application/zip"); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	_ = file.Close()
	raw, err := item.JSON()
	if err != nil || db.Model(&models.Problem{}).Where("id = ?", id).Update("package", datatypes.JSON(raw)).Error != nil {
		t.Fatalf("update problem package: %v", err)
	}
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func uploadAssetForTest(t *testing.T, e *echo.Echo, target string, cookies []*http.Cookie, section string, name string, content string) contract.ProblemAssets {
	t.Helper()
	assetURL := strings.TrimSuffix(target, "/files")
	current := requestWithCookies(e, http.MethodGet, assetURL, cookies, nil)
	if current.Code != http.StatusOK {
		t.Fatalf("get assets got %d body=%s", current.Code, current.Body.String())
	}
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
	req.Header.Set("If-Match", current.Header().Get("ETag"))
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("upload asset got %d body=%s", res.Code, res.Body.String())
	}
	return decodeJSON[contract.ProblemAssets](t, res)
}

func uploadImageForTest(t *testing.T, e *echo.Echo, target string, cookies []*http.Cookie, name string, content []byte) contract.UploadResult {
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
	return decodeJSON[contract.UploadResult](t, res)
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
		if cookie.Name == auth.CSRFCookie {
			req.Header.Set(auth.CSRFHeader, cookie.Value)
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
	startRedis(t)
	cache.ResetForTest()
	t.Cleanup(cache.ResetForTest)
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

func startRedis(t *testing.T) {
	t.Helper()
	server := miniredis.RunT(t)
	t.Setenv("REDIS", "redis://"+server.Addr()+"/0")
}

func databaseSession(t *testing.T, db *gorm.DB, userID uint) []*http.Cookie {
	t.Helper()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		t.Fatalf("read session user: %v", err)
	}
	e := echo.New()
	res := httptest.NewRecorder()
	ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), res)
	if err := auth.CreateUserSession(ctx, user, time.Now()); err != nil {
		t.Fatalf("create session: %v", err)
	}
	return res.Result().Cookies()
}

func allowGuest(t *testing.T, db *gorm.DB) {
	t.Helper()
	site := settings.Settings{
		SiteName:                "DOJ",
		AllowRegistration:       false,
		AllowGuestAccess:        true,
		DefaultSubmissionPublic: false,
		Notice:                  "",
	}
	if err := settings.Save(db, site); err != nil {
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

func decodePageItems[T any](t *testing.T, res *httptest.ResponseRecorder) []T {
	t.Helper()
	return decodeJSON[contract.Page[T]](t, res).Items
}

func hasProblem(items []contract.Problem, id uint) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func hasSolvedProblem(items []contract.SolvedProblem, id uint) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func problemByID(items []contract.Problem, id uint) (contract.Problem, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return contract.Problem{}, false
}

func hasSubmissionProblem(items []contract.Submission, id uint) bool {
	for _, item := range items {
		if item.ProblemID == id {
			return true
		}
	}
	return false
}

func hasActivityProblem(items []contract.UserActivity, id uint) bool {
	for _, item := range items {
		if item.ProblemID == id {
			return true
		}
	}
	return false
}

func hasActivity(items []contract.UserActivity, kind string, title string) bool {
	for _, item := range items {
		if item.Type == kind && item.Title == title {
			return true
		}
	}
	return false
}

func activityBySubmission(items []contract.UserActivity, id uint) (contract.UserActivity, bool) {
	for _, item := range items {
		if item.Type == "submission" && item.ID == id {
			return item, true
		}
	}
	return contract.UserActivity{}, false
}

func hasSubmission(items []contract.Submission, id uint) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func userInRank(items []contract.RankUser, user string) bool {
	_, ok := rankByUser(items, user)
	return ok
}

func rankByUser(items []contract.RankUser, user string) (contract.RankUser, bool) {
	for _, item := range items {
		if item.User == user {
			return item, true
		}
	}
	return contract.RankUser{}, false
}

func rankProblemByID(items []contract.RankProblem, id uint) (contract.RankProblem, bool) {
	for _, item := range items {
		if item.ProblemID == id {
			return item, true
		}
	}
	return contract.RankProblem{}, false
}

func countForDate(items []contract.HeatCell, date string) int {
	for _, item := range items {
		if item.Date == date {
			return item.Count
		}
	}
	return 0
}

func nonzeroHeatmapDays(items []contract.HeatCell) int {
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
