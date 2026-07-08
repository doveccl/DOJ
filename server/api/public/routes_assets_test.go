package public

import (
	"archive/zip"
	"bytes"
	"context"
	"github.com/doveccl/doj/server/storage"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

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
}

func TestProblemAssetDownloadsSupportNestedPathsAndExistingProblems(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
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
	adminCookies := databaseSession(t, db, admin.ID)
	nestedPath := filepath.Join(storage.Root(), "problems", "1000", "data", "cases", "1.in")
	if err := os.MkdirAll(filepath.Dir(nestedPath), 0o755); err != nil {
		t.Fatalf("create nested data dir: %v", err)
	}
	if err := os.WriteFile(nestedPath, []byte("nested input"), 0o644); err != nil {
		t.Fatalf("write nested data file: %v", err)
	}
	for key, body := range map[string]string{
		filepath.Join(storage.Root(), "problems", "1000", "statement.md"):           "# Visible\n\n![img](./assets/note.png)",
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

	res := requestWithCookies(e, http.MethodGet, "/api/problems/1000/data/cases/1.in", adminCookies, nil)
	if res.Code != http.StatusOK || res.Body.String() != "nested input" {
		t.Fatalf("nested asset download got %d body=%q", res.Code, res.Body.String())
	}
	res = requestWithCookies(e, http.MethodGet, "/api/problems/1000.zip", adminCookies, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("problem zip got %d body=%s", res.Code, res.Body.String())
	}
	if disposition := res.Header().Get(echo.HeaderContentDisposition); !strings.Contains(disposition, `filename="P1000.zip"`) {
		t.Fatalf("problem zip content disposition = %q", disposition)
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
	res = requestWithCookies(e, http.MethodGet, "/api/problems/404.zip", adminCookies, nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("missing problem zip got %d body=%s", res.Code, res.Body.String())
	}
}

func TestProblemAssetsUseCaseFileOrder(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	store, err := storage.NewFromEnv()
	if err != nil {
		t.Fatalf("object store: %v", err)
	}
	for _, name := range []string{"10.out", "2.out", "1.out", "10.in", "readme.txt", "3.ans", "2.in", "input4.txt", "answer4.txt", "3.in", "1.in"} {
		key := path.Join("problems", "1000", "data", name)
		if err := store.Put(context.Background(), key, strings.NewReader(name), int64(len(name)), "text/plain"); err != nil {
			t.Fatalf("put %s: %v", name, err)
		}
	}
	assets, err := problemAssetsFromStore(context.Background(), 1000, store)
	if err != nil {
		t.Fatalf("problem assets: %v", err)
	}
	var got []string
	for _, item := range assets.Data {
		got = append(got, item.Name)
	}
	want := []string{"1.in", "1.out", "2.in", "2.out", "3.in", "3.ans", "input4.txt", "answer4.txt", "10.in", "10.out", "readme.txt"}
	if !slices.Equal(got, want) {
		t.Fatalf("data files = %+v, want %+v", got, want)
	}
	if assets.Cases != 5 {
		t.Fatalf("cases = %d, want 5", assets.Cases)
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

func TestJudgeTemplateUsesDockerfileCMDAndInteractorArgs(t *testing.T) {
	files := judgeTemplateFiles()
	dockerfile := files["Dockerfile"]
	main := files["main.cc"]
	if !strings.Contains(dockerfile, `FROM gcc`) || strings.Contains(dockerfile, `gcc:`) || !strings.Contains(dockerfile, `g++ main.cc -o main`) || !strings.Contains(dockerfile, `CMD ["/src/main"]`) {
		t.Fatalf("Dockerfile template must build and expose the same CMD path:\n%s", dockerfile)
	}
	for _, want := range []string{"argv[1]", "argv[3]", "argv[4]", "thread feeder", "fclose(stdout)", "0 = AC", "1 = WA", "3 = checker/interactor error"} {
		if !strings.Contains(main, want) {
			t.Fatalf("main.cc template missing %s:\n%s", want, main)
		}
	}
}
