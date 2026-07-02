package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubRunner struct {
	dir  string
	data []byte
	err  error
}

func (runner stubRunner) Dump(_ context.Context, _ string) (string, error) {
	if runner.err != nil {
		return "", runner.err
	}
	file, err := os.CreateTemp(runner.dir, "stub-*.sql.gz")
	if err != nil {
		return "", err
	}
	if _, err := file.Write(runner.data); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func TestBackupNowWritesSQLGZToStorage(t *testing.T) {
	db := testDB(t)
	root := t.TempDir()
	t.Setenv("STORAGE", root)
	t.Setenv("DATABASE", "postgres://postgres@localhost/doj_test")
	now := time.Date(2026, 6, 26, 3, 0, 0, 0, time.Local)
	data := gzipData(t, "select 1;\n")
	manager := Manager{DB: db, Runner: stubRunner{dir: t.TempDir(), data: data}, Now: func() time.Time { return now }}

	item, err := manager.BackupNow(t.Context())
	if err != nil {
		t.Fatalf("backup now: %v", err)
	}
	if item.Name != "doj_test_2026-06-26_03-00-00.sql.gz" || item.Size != int64(len(data)) {
		t.Fatalf("unexpected item: %+v", item)
	}

	list, err := manager.List(t.Context())
	if err != nil {
		t.Fatalf("list backups: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Name != item.Name {
		t.Fatalf("backup list should come from storage: %+v", list)
	}
	reader, _, err := manager.Open(t.Context(), item.Name)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer reader.Close()
	got := gunzip(t, reader)
	if got != "select 1;\n" {
		t.Fatalf("backup content = %q", got)
	}
}

func TestBackupRunningLockBlocksConcurrentBackup(t *testing.T) {
	db := testDB(t)
	t.Setenv("STORAGE", t.TempDir())
	t.Setenv("DATABASE", "postgres://postgres@localhost/doj")
	now := time.Date(2026, 6, 26, 3, 0, 0, 0, time.Local)
	ok, err := utils.CacheSetNX(t.Context(), lockKey, lockValue{Name: "doj_2026-06-26_03-00-00.sql.gz", StartedAt: now}, time.Hour)
	if err != nil || !ok {
		t.Fatalf("set lock ok=%v err=%v", ok, err)
	}
	manager := Manager{DB: db, Runner: stubRunner{dir: t.TempDir(), data: gzipData(t, "select 1;\n")}, Now: func() time.Time { return now }}
	if _, err := manager.BackupNow(t.Context()); !errors.Is(err, ErrRunning) {
		t.Fatalf("backup should be blocked by running lock, err=%v", err)
	}
}

func TestPruneKeepsNewestBackups(t *testing.T) {
	db := testDB(t)
	root := t.TempDir()
	t.Setenv("STORAGE", root)
	manager := Manager{DB: db}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	for _, name := range []string{
		"doj_2026-06-24_03-00-00.sql.gz",
		"doj_2026-06-25_03-00-00.sql.gz",
		"doj_2026-06-26_03-00-00.sql.gz",
	} {
		data := gzipData(t, name)
		if err := store.Put(t.Context(), filepath.ToSlash(filepath.Join(Prefix, name)), bytes.NewReader(data), int64(len(data)), contentType); err != nil {
			t.Fatalf("put %s: %v", name, err)
		}
	}
	if err := manager.Prune(t.Context(), 2); err != nil {
		t.Fatalf("prune: %v", err)
	}
	list, err := manager.List(t.Context())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 2 || list.Items[0].Name != "doj_2026-06-26_03-00-00.sql.gz" || list.Items[1].Name != "doj_2026-06-25_03-00-00.sql.gz" {
		t.Fatalf("unexpected remaining backups: %+v", list.Items)
	}
}

func TestPgDumpRunnerReportsMissingTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := PgDumpRunner{}.Dump(t.Context(), "postgres://postgres@localhost/doj")
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "pg_dump is required") {
		t.Fatalf("missing pg_dump should produce clear error, got %v", err)
	}
}

func TestBackupCronSettings(t *testing.T) {
	settings, err := CleanSettings(Settings{Enabled: true, Cron: "*/15 1-5 * * mon-fri", Keep: 7})
	if err != nil || settings.Cron != "*/15 1-5 * * mon-fri" {
		t.Fatalf("valid cron rejected settings=%+v err=%v", settings, err)
	}
	if _, err := CleanSettings(Settings{Enabled: true, Cron: "not cron", Keep: 7}); err == nil {
		t.Fatalf("invalid cron accepted")
	}
	db := testDB(t)
	t.Setenv("STORAGE", t.TempDir())
	manager := Manager{DB: db}
	due, err := manager.Due(t.Context(), Settings{Enabled: true, Cron: "30 3 * * *", Keep: 7}, time.Date(2026, 6, 26, 3, 30, 0, 0, time.Local))
	if err != nil || !due {
		t.Fatalf("cron should be due, due=%v err=%v", due, err)
	}
}

func gzipData(t *testing.T, body string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buffer.Bytes()
}

func gunzip(t *testing.T, reader io.Reader) string {
	t.Helper()
	gz, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gz.Close()
	data, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	return string(data)
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	server := miniredis.RunT(t)
	t.Setenv("REDIS", "redis://"+server.Addr()+"/0")
	utils.ResetCacheForTest()
	t.Cleanup(utils.ResetCacheForTest)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "backup.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
