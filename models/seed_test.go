package models

import (
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureAdminRequiresBootstrapWhenNoAdminExists(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := EnsureAdmin(db, "admin", "admin@example.com", ""); err == nil {
		t.Fatalf("expected missing bootstrap password to fail")
	}
}

func TestEnsureAdminCreatesFirstAdmin(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := EnsureAdmin(db, " admin ", " admin@example.com ", "password123"); err != nil {
		t.Fatalf("ensure admin: %v", err)
	}
	var user User
	if err := db.First(&user, "name = ?", "admin").Error; err != nil {
		t.Fatalf("read admin: %v", err)
	}
	if !user.Admin || user.Mail != "admin@example.com" {
		t.Fatalf("unexpected admin user: %+v", user)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Auth), []byte("password123")); err != nil {
		t.Fatalf("admin password hash does not match")
	}
}

func TestEnsureAdminDoesNotRequireBootstrapAfterAdminExists(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := EnsureAdmin(db, "admin", "admin@example.com", "password123"); err != nil {
		t.Fatalf("initial ensure admin: %v", err)
	}
	if err := EnsureAdmin(db, "", "", ""); err != nil {
		t.Fatalf("existing admin should allow empty bootstrap settings: %v", err)
	}
	var count int64
	if err := db.Model(&User{}).Where("admin = ?", true).Count(&count).Error; err != nil {
		t.Fatalf("count admins: %v", err)
	}
	if count != 1 {
		t.Fatalf("admin count = %d, want 1", count)
	}
}

func TestEnsureDefaultLanguageCreatesCppWhenEmpty(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := EnsureDefaultLanguage(db); err != nil {
		t.Fatalf("ensure default lang: %v", err)
	}
	var lang Language
	if err := db.First(&lang, "id = ?", "cpp").Error; err != nil {
		t.Fatalf("read default lang: %v", err)
	}
	if lang.Name != "C/C++" || lang.Source != "main.cc" || lang.Image != "gcc:14" || lang.Compile == "" || lang.Run != "./main" {
		t.Fatalf("unexpected default lang: %+v", lang)
	}
}

func TestEnsureDefaultLanguageKeepsExistingLangs(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	custom := Language{ID: "py", Name: "Python", Source: "main.py", Image: "python:3", Run: "python3 main.py"}
	if err := db.Create(&custom).Error; err != nil {
		t.Fatalf("create custom lang: %v", err)
	}
	if err := EnsureDefaultLanguage(db); err != nil {
		t.Fatalf("ensure default lang: %v", err)
	}
	var count int64
	if err := db.Model(&Language{}).Count(&count).Error; err != nil {
		t.Fatalf("count langs: %v", err)
	}
	if count != 1 {
		t.Fatalf("lang count = %d, want 1", count)
	}
}

func TestEnsureProblemIDBaseNoopsOnSQLite(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := EnsureProblemIDBase(db); err != nil {
		t.Fatalf("ensure problem id base: %v", err)
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return db
}
