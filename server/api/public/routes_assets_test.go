package public

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/doveccl/doj/server/storage"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	problemdata "github.com/doveccl/doj/server/problem"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestUserUploadQuotaCountsUniqueObjects(t *testing.T) {
	objects := []storage.Info{{Key: "users/1/old.png", Size: maxUserUploadBytes - 10}}
	if !userUploadWithinQuota(objects, "users/1/new.png", 10) {
		t.Fatal("upload at quota boundary was rejected")
	}
	if userUploadWithinQuota(objects, "users/1/new.png", 11) {
		t.Fatal("upload beyond quota was accepted")
	}
	objects = []storage.Info{{Key: "users/1/same.png", Size: maxUserUploadBytes}}
	if !userUploadWithinQuota(objects, "users/1/same.png", 10) {
		t.Fatal("content-addressed replacement counted twice")
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
	problem := models.Problem{ID: 1000, Title: "Visible", Content: "# Visible\n\n![img](./assets/note.png)", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}

	e := echo.New()
	Register(e, db)
	studentCookies := databaseSession(t, db, student.ID)
	adminCookies := databaseSession(t, db, admin.ID)

	userImage := uploadImageForTest(t, e, "/api/uploads/images", studentCookies, "avatar.png", tinyPNG())
	if !strings.HasPrefix(userImage.URL, "/api/users/") || strings.Contains(userImage.URL, "://") {
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
	if !strings.HasPrefix(problemImage.URL, "/api/problems/1000/assets/") || strings.Contains(problemImage.URL, "://") {
		t.Fatalf("problem image url should be a relative media path, got %q", problemImage.URL)
	}
	rel := strings.TrimPrefix(problemImage.URL, "/api/problems/1000/assets/")
	if strings.Contains(rel, "/") {
		t.Fatalf("problem image should not include date folders, got %q", problemImage.URL)
	}
	if _, err := os.Stat(filepath.Join(storage.Root(), "problems", "1000", "assets", filepath.FromSlash(rel))); err != nil {
		t.Fatalf("problem image should keep the existing object key convention: %v", err)
	}
	res = requestWithCookies(e, http.MethodGet, problemImage.URL, adminCookies, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("read problem image got %d body=%s", res.Code, res.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, problemImage.URL, nil)
	req.Host = "backend:7974"
	req.Header.Set("Referer", "https://frontend.example/problems/1000")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	for _, cookie := range adminCookies {
		req.AddCookie(cookie)
	}
	res = httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("read problem image through reverse proxy got %d body=%s", res.Code, res.Body.String())
	}
}

func TestProblemAssetDownloadsSupportNestedPathsAndExistingProblems(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Content: "# Visible\n\n![img](./assets/note.png)", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}

	e := echo.New()
	Register(e, db)
	adminCookies := databaseSession(t, db, admin.ID)
	nestedPath := filepath.Join(storage.Root(), "problems", "1000", "data", "cases", "1.in")
	if err := os.MkdirAll(filepath.Dir(nestedPath), 0o755); err != nil {
		t.Fatalf("create nested data dir: %v", err)
	}
	if err := os.WriteFile(nestedPath, []byte("nested input"), 0o644); err != nil {
		t.Fatalf("write nested data file: %v", err)
	}
	for key, body := range map[string]string{
		filepath.Join(storage.Root(), "problems", "1000", "data", "cases", "1.out"): "nested output",
		filepath.Join(storage.Root(), "problems", "1000", "judge", "main.cc"):       "int main(){}",
		filepath.Join(storage.Root(), "problems", "1000", "assets", "note.txt"):     "asset note",
	} {
		if err := os.MkdirAll(filepath.Dir(key), 0o755); err != nil {
			t.Fatalf("create asset dir: %v", err)
		}
		if err := os.WriteFile(key, []byte(body), 0o644); err != nil {
			t.Fatalf("write asset file: %v", err)
		}
	}
	packagePath := filepath.Join(t.TempDir(), "package.zip")
	item, err := problemdata.Build(filepath.Join(storage.Root(), "problems", "1000"), packagePath, nil)
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	packageFile, err := os.Open(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(t.Context(), problemdata.ObjectKey(1000, item.Hash), packageFile, item.Size, "application/zip"); err != nil {
		t.Fatal(err)
	}
	_ = packageFile.Close()
	raw, _ := item.JSON()
	if err := db.Model(&models.Problem{}).Where("id = ?", 1000).Update("package", datatypes.JSON(raw)).Error; err != nil {
		t.Fatal(err)
	}

	res := requestWithCookies(e, http.MethodGet, "/api/problems/1000/data/cases/1.in", adminCookies, nil)
	if res.Code != http.StatusOK || res.Body.String() != "nested input" {
		t.Fatalf("nested asset download got %d body=%q", res.Code, res.Body.String())
	}
	if cache := res.Header().Get(echo.HeaderCacheControl); cache != "private, no-store" {
		t.Fatalf("private problem asset cache header = %q", cache)
	}
	res = requestWithCookies(e, http.MethodGet, "/api/problems/1000.zip", adminCookies, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("problem zip got %d body=%s", res.Code, res.Body.String())
	}
	if disposition := res.Header().Get(echo.HeaderContentDisposition); !strings.Contains(disposition, `filename="P1000.zip"`) {
		t.Fatalf("problem zip content disposition = %q", disposition)
	}
	if cache := res.Header().Get(echo.HeaderCacheControl); cache != "private, no-store" {
		t.Fatalf("problem zip cache header = %q", cache)
	}
	reader, err := zip.NewReader(bytes.NewReader(res.Body.Bytes()), int64(res.Body.Len()))
	if err != nil {
		t.Fatalf("read problem zip: %v", err)
	}
	names := map[string]bool{}
	content := map[string]string{}
	for _, file := range reader.File {
		names[file.Name] = true
		body, err := readZipFile(file)
		if err != nil {
			t.Fatalf("read zip file %s: %v", file.Name, err)
		}
		content[file.Name] = string(body)
	}
	for _, name := range []string{"statement.md", "data/cases/1.in", "data/cases/1.out", "judge/main.cc", "assets/note.txt"} {
		if !names[name] {
			t.Fatalf("problem zip missing %s, got %+v", name, names)
		}
	}
	if content["statement.md"] != "# Visible\n\n![img](./assets/note.png)" {
		t.Fatalf("problem zip statement = %q", content["statement.md"])
	}
	item.Files = slices.DeleteFunc(item.Files, func(file problemdata.File) bool { return file.Path == "data/cases/1.out" })
	raw, _ = item.JSON()
	if err := db.Model(&models.Problem{}).Where("id = ?", 1000).Update("package", datatypes.JSON(raw)).Error; err != nil {
		t.Fatal(err)
	}
	res = requestWithCookies(e, http.MethodGet, "/api/problems/1000.zip", adminCookies, nil)
	filtered, err := zip.NewReader(bytes.NewReader(res.Body.Bytes()), int64(res.Body.Len()))
	if err != nil {
		t.Fatalf("read filtered problem zip: %v", err)
	}
	filteredNames := map[string]bool{}
	for _, file := range filtered.File {
		filteredNames[file.Name] = true
	}
	if filteredNames["data/cases/1.out"] || !filteredNames["judge/main.cc"] {
		t.Fatalf("filtered problem zip files = %+v", filteredNames)
	}
	res = requestWithCookies(e, http.MethodGet, "/api/problems/404.zip", adminCookies, nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("missing problem zip got %d body=%s", res.Code, res.Body.String())
	}
}

func TestProblemPackageBatchUploadRangeDeleteAndCAS(t *testing.T) {
	if raw := os.Getenv("DOJ_TEST_S3"); raw != "" {
		t.Setenv("STORAGE", raw)
	} else {
		t.Setenv("STORAGE", t.TempDir())
	}
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	problem := models.Problem{ID: 1000, Title: "Package", Tags: datatypes.JSON([]byte(`[]`)), Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	get := requestWithCookies(e, http.MethodGet, "/api/problems/1000/assets", cookies, nil)
	version := get.Header().Get("ETag")
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("section", "data")
	part, err := writer.CreateFormFile("files", "1.in")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("input"))
	var archive bytes.Buffer
	zipWriter := zip.NewWriter(&archive)
	for name, content := range map[string]string{"1.in": "zip input", "1.out": "answer"} {
		file, err := zipWriter.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = file.Write([]byte(content))
	}
	_ = zipWriter.Close()
	part, err = writer.CreateFormFile("files", "cases.zip")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(archive.Bytes())
	_ = writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/problems/1000/assets/files", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("If-Match", version)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("batch upload got %d body=%s", res.Code, res.Body.String())
	}
	assets := decodeJSON[contract.ProblemAssets](t, res)
	if len(assets.Data) != 2 || len(assets.CaseList) != 1 || assets.CaseList[0].Score != nil {
		t.Fatalf("assets = %+v", assets)
	}
	download := requestWithCookies(e, http.MethodGet, "/api/problems/1000/data/1.out", cookies, nil)
	if download.Code != http.StatusOK || download.Body.String() != "answer" {
		t.Fatalf("range download got %d body=%q", download.Code, download.Body.String())
	}
	input := requestWithCookies(e, http.MethodGet, "/api/problems/1000/data/1.in", cookies, nil)
	if input.Code != http.StatusOK || input.Body.String() != "input" {
		t.Fatalf("loose file should override same-request ZIP, got %d body=%q", input.Code, input.Body.String())
	}
	scoreRequest := httptest.NewRequest(http.MethodPatch, "/api/problems/1000/assets/cases/score?case=1", strings.NewReader(`{"score":1000001}`))
	scoreRequest.Header.Set("Content-Type", "application/json")
	scoreRequest.Header.Set("If-Match", `"`+assets.Version+`"`)
	for _, cookie := range cookies {
		scoreRequest.AddCookie(cookie)
	}
	addCSRFHeader(scoreRequest, cookies)
	scoreResponse := httptest.NewRecorder()
	e.ServeHTTP(scoreResponse, scoreRequest)
	if scoreResponse.Code != http.StatusOK {
		t.Fatalf("score update got %d body=%s", scoreResponse.Code, scoreResponse.Body.String())
	}
	scored := decodeJSON[contract.ProblemAssets](t, scoreResponse)
	if scored.CaseList[0].Score == nil || *scored.CaseList[0].Score != 1_000_001 {
		t.Fatalf("case score = %+v", scored.CaseList)
	}
	resetRequest := httptest.NewRequest(http.MethodPatch, "/api/problems/1000/assets/cases/score?case=1", strings.NewReader(`{"score":null}`))
	resetRequest.Header.Set("Content-Type", "application/json")
	resetRequest.Header.Set("If-Match", `"`+scored.Version+`"`)
	for _, cookie := range cookies {
		resetRequest.AddCookie(cookie)
	}
	addCSRFHeader(resetRequest, cookies)
	resetResponse := httptest.NewRecorder()
	e.ServeHTTP(resetResponse, resetRequest)
	if resetResponse.Code != http.StatusOK {
		t.Fatalf("score reset got %d body=%s", resetResponse.Code, resetResponse.Body.String())
	}
	reset := decodeJSON[contract.ProblemAssets](t, resetResponse)
	if reset.CaseList[0].Score != nil {
		t.Fatalf("default case score should be omitted: %+v", reset.CaseList)
	}
	store, err := storage.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	objects, err := store.List(t.Context(), "problems/1000/packages")
	if err != nil || len(objects) != 1 {
		t.Fatalf("DB-only score update package objects = %d, %v", len(objects), err)
	}
	deleteRequest := func(match string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodDelete, "/api/problems/1000/assets/files?key=data%2F1.out", nil)
		req.Header.Set("If-Match", match)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		addCSRFHeader(req, cookies)
		got := httptest.NewRecorder()
		e.ServeHTTP(got, req)
		return got
	}
	stale := deleteRequest(`"` + scored.Version + `"`)
	if stale.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale delete got %d body=%s", stale.Code, stale.Body.String())
	}
	deleted := deleteRequest(`"` + reset.Version + `"`)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete got %d body=%s", deleted.Code, deleted.Body.String())
	}
	missing := requestWithCookies(e, http.MethodGet, "/api/problems/1000/data/1.out", cookies, nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("deleted download got %d body=%s", missing.Code, missing.Body.String())
	}
	var afterDelete contract.ProblemAssets
	if err := json.Unmarshal(deleted.Body.Bytes(), &afterDelete); err != nil {
		t.Fatal(err)
	}
	clearRequest := httptest.NewRequest(http.MethodDelete, "/api/problems/1000/assets/files?key=data", nil)
	clearRequest.Header.Set("If-Match", `"`+afterDelete.Version+`"`)
	for _, cookie := range cookies {
		clearRequest.AddCookie(cookie)
	}
	addCSRFHeader(clearRequest, cookies)
	cleared := httptest.NewRecorder()
	e.ServeHTTP(cleared, clearRequest)
	if cleared.Code != http.StatusOK {
		t.Fatalf("clear got %d body=%s", cleared.Code, cleared.Body.String())
	}
	clearAssets := decodeJSON[contract.ProblemAssets](t, cleared)
	if len(clearAssets.Data) != 0 || len(clearAssets.CaseList) != 0 {
		t.Fatalf("cleared assets = %+v", clearAssets)
	}
	objects, err = store.List(t.Context(), "problems/1000/packages")
	if err != nil || len(objects) != 1 {
		t.Fatalf("DB-only clear package objects = %d, %v", len(objects), err)
	}
}

func TestImportProblemZipKeepsPackageRoots(t *testing.T) {
	var archive bytes.Buffer
	zipWriter := zip.NewWriter(&archive)
	for name, content := range map[string]string{"data/1.in": "input", "data/1.out": "answer", "judge/Dockerfile": "FROM gcc", "statement.md": "ignored"} {
		file, err := zipWriter.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = file.Write([]byte(content))
	}
	_ = zipWriter.Close()
	var body bytes.Buffer
	formWriter := multipart.NewWriter(&body)
	part, err := formWriter.CreateFormFile("files", "P1000.zip")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(archive.Bytes())
	_ = formWriter.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", formWriter.FormDataContentType())
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	if err := importProblemZip(req.MultipartForm.File["files"][0], "data", work); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"data/1.in", "data/1.out", "judge/Dockerfile"} {
		if _, err := os.Stat(filepath.Join(work, filepath.FromSlash(name))); err != nil {
			t.Fatalf("missing imported %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(work, "data", "statement.md")); !os.IsNotExist(err) {
		t.Fatalf("full package metadata should be ignored, err=%v", err)
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
