package admin

import (
	"net/http"
	"strings"
	"testing"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

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

	res := requestJSONWithCookies(e, http.MethodPost, "/api/admin/backups", databaseSession(t, admin.ID), `{}`)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing pg_dump got %d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "pg_dump is required") {
		t.Fatalf("missing pg_dump response should explain dependency: %s", res.Body.String())
	}
}
