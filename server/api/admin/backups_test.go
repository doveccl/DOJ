package admin

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

func TestBackupDownloadIsNotCacheable(t *testing.T) {
	db := testAdminDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	root := t.TempDir()
	t.Setenv("STORAGE", root)
	name := "doj_2026-07-10_12-00-00.sql.gz"
	if err := os.MkdirAll(filepath.Join(root, "backups"), 0o755); err != nil {
		t.Fatalf("create backup dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "backups", name), []byte("backup"), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	e := echo.New()
	Register(e, db, nil)

	res := requestWithCookies(e, http.MethodGet, "/api/admin/backups/"+name+"/download", databaseSession(t, db, admin.ID))
	if res.Code != http.StatusOK || res.Body.String() != "backup" {
		t.Fatalf("download backup got %d body=%q", res.Code, res.Body.String())
	}
	if cache := res.Header().Get(echo.HeaderCacheControl); cache != "private, no-store" {
		t.Fatalf("backup cache header = %q", cache)
	}
}

func TestBackupCreateReportsUnavailableWhenPgDumpMissing(t *testing.T) {
	db := testAdminDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	t.Setenv("DATABASE", "postgres://postgres@localhost/doj_test")
	t.Setenv("STORAGE", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	e := echo.New()
	Register(e, db, nil)

	res := requestJSONWithCookies(e, http.MethodPost, "/api/admin/backups", databaseSession(t, db, admin.ID), `{}`)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing pg_dump got %d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "pg_dump is required") {
		t.Fatalf("missing pg_dump response should explain dependency: %s", res.Body.String())
	}
}
