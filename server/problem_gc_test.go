package server

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/doveccl/doj/models"
	problemdata "github.com/doveccl/doj/server/problem"
	"github.com/doveccl/doj/server/storage"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCollectProblemPackagesKeepsCurrentRecentAndJudging(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "gc.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Problem{}, &models.Submission{}); err != nil {
		t.Fatal(err)
	}
	hash := func(char string) string { return strings.Repeat(char, 64) }
	for _, row := range []models.Problem{
		{ID: 1000, Title: "idle", Package: packageRaw(t, hash("a"))},
		{ID: 1001, Title: "judging", Package: packageRaw(t, hash("b"))},
		{ID: 1003, Title: "invalid", Package: datatypes.JSON(`{"broken"`)},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.Submission{UserID: 1, ProblemID: 1001, Language: "cpp", Code: "", Status: "judging"}).Error; err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Setenv("STORAGE", root)
	store, err := storage.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	current := problemdata.ObjectKey(1000, hash("a"))
	idleOld := problemdata.ObjectKey(1000, hash("c"))
	idleRecent := problemdata.ObjectKey(1000, hash("d"))
	judgingOld := problemdata.ObjectKey(1001, hash("e"))
	deletedOld := problemdata.ObjectKey(1002, hash("f"))
	invalidOld := problemdata.ObjectKey(1003, hash("9"))
	for _, key := range []string{current, idleOld, idleRecent, judgingOld, deletedOld, invalidOld} {
		if err := store.Put(t.Context(), key, bytes.NewReader([]byte("zip")), 3, "application/zip"); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-48 * time.Hour)
	for _, key := range []string{current, idleOld, judgingOld, deletedOld, invalidOld} {
		file := filepath.Join(root, filepath.FromSlash(key))
		if err := os.Chtimes(file, old, old); err != nil {
			t.Fatal(err)
		}
	}
	deleted, err := collectProblemPackages(t.Context(), db, store, time.Now().Add(-24*time.Hour))
	if err != nil || deleted != 2 {
		t.Fatalf("deleted = %d, err = %v", deleted, err)
	}
	for key, exists := range map[string]bool{current: true, idleOld: false, idleRecent: true, judgingOld: true, deletedOld: false, invalidOld: true} {
		reader, _, err := store.Open(t.Context(), key)
		if reader != nil {
			_ = reader.Close()
		}
		if (err == nil) != exists {
			t.Fatalf("%s exists = %t, want %t (err=%v)", key, err == nil, exists, err)
		}
	}
}

func packageRaw(t *testing.T, hash string) datatypes.JSON {
	t.Helper()
	raw, err := (problemdata.Package{Hash: hash}).JSON()
	if err != nil {
		t.Fatal(err)
	}
	return datatypes.JSON(raw)
}
