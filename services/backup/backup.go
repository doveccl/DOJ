package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/utils"
	"github.com/robfig/cron/v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	Prefix         = "backups"
	settingEnabled = "backup_enabled"
	settingKeep    = "backup_keep"
	settingCron    = "backup_cron"
	lockKey        = "backup:running"
	staleAfter     = 6 * time.Hour
	lockTTL        = 24 * time.Hour
	contentType    = "application/gzip"
)

type Settings struct {
	Enabled bool   `json:"enabled"`
	Cron    string `json:"cron"`
	Keep    int    `json:"keep"`
}

type Item struct {
	Name      string    `json:"name"`
	Database  string    `json:"database"`
	CreatedAt time.Time `json:"createdAt"`
	Size      int64     `json:"size"`
}

type Running struct {
	Name      string    `json:"name"`
	StartedAt time.Time `json:"startedAt"`
	Stale     bool      `json:"stale"`
}

type ListResult struct {
	Running *Running `json:"running,omitempty"`
	Items   []Item   `json:"items"`
}

type Runner interface {
	Dump(ctx context.Context, dsn string) (string, error)
}

type Manager struct {
	DB     *gorm.DB
	Store  utils.ObjectStore
	Runner Runner
	Now    func() time.Time
	DSN    string
}

type lockValue struct {
	Name      string    `json:"name"`
	StartedAt time.Time `json:"startedAt"`
}

var namePattern = regexp.MustCompile(`^(.+)_([0-9]{4}-[0-9]{2}-[0-9]{2})_([0-9]{2}-[0-9]{2}-[0-9]{2})\.sql\.gz$`)

func DefaultSettings() Settings {
	return Settings{Enabled: false, Cron: "0 3 * * *", Keep: 7}
}

func ReadSettings(db *gorm.DB) (Settings, error) {
	settings := DefaultSettings()
	var rows []models.Setting
	if err := db.Find(&rows, "key IN ?", settingKeys()).Error; err != nil {
		return settings, err
	}
	for _, row := range rows {
		switch row.Key {
		case settingEnabled:
			if err := unmarshalSetting(row.Value, &settings.Enabled); err != nil {
				return settings, err
			}
		case settingKeep:
			if err := unmarshalSetting(row.Value, &settings.Keep); err != nil {
				return settings, err
			}
		case settingCron:
			if err := unmarshalSetting(row.Value, &settings.Cron); err != nil {
				return settings, err
			}
		}
	}
	return CleanSettings(settings)
}

func SaveSettings(db *gorm.DB, settings Settings) error {
	clean, err := CleanSettings(settings)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		values := map[string]any{
			settingEnabled: clean.Enabled,
			settingKeep:    clean.Keep,
			settingCron:    clean.Cron,
		}
		for _, key := range settingKeys() {
			if err := saveSetting(tx, key, values[key]); err != nil {
				return err
			}
		}
		return nil
	})
}

func CleanSettings(settings Settings) (Settings, error) {
	if settings.Keep < 1 || settings.Keep > 100 {
		return settings, fmt.Errorf("backup keep must be between 1 and 100")
	}
	settings.Cron = strings.Join(strings.Fields(settings.Cron), " ")
	if settings.Cron == "" {
		settings.Cron = "0 3 * * *"
	}
	if _, err := cron.ParseStandard(settings.Cron); err != nil {
		return settings, err
	}
	return settings, nil
}

func (manager Manager) List(ctx context.Context) (ListResult, error) {
	store, err := manager.store()
	if err != nil {
		return ListResult{}, err
	}
	objects, err := store.List(ctx, Prefix)
	if err != nil {
		return ListResult{}, err
	}
	items := make([]Item, 0, len(objects))
	for _, object := range objects {
		item, ok := itemFromObject(object)
		if ok {
			items = append(items, item)
		}
	}
	sortItems(items)
	running, err := runningLock(ctx, manager.now())
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Running: running, Items: items}, nil
}

func (manager Manager) BackupNow(ctx context.Context) (Item, error) {
	store, err := manager.store()
	if err != nil {
		return Item{}, err
	}
	now := manager.now()
	name := BuildName(DatabaseName(manager.dsn()), now)
	lock := lockValue{Name: name, StartedAt: now}
	ok, err := utils.CacheSetNX(ctx, lockKey, lock, lockTTL)
	if err != nil {
		return Item{}, err
	}
	if !ok {
		return Item{}, ErrRunning
	}
	defer func() {
		_ = utils.CacheDelete(context.Background(), lockKey)
	}()

	filePath, err := manager.runner().Dump(ctx, manager.dsn())
	if err != nil {
		return Item{}, err
	}
	defer func() {
		_ = os.Remove(filePath)
	}()
	file, err := os.Open(filePath)
	if err != nil {
		return Item{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Item{}, err
	}
	key := path.Join(Prefix, name)
	if err := store.Put(ctx, key, file, info.Size(), contentType); err != nil {
		return Item{}, err
	}
	item := Item{Name: name, Database: DatabaseName(manager.dsn()), CreatedAt: now, Size: info.Size()}
	settings, err := ReadSettings(manager.DB)
	if err == nil {
		_ = manager.Prune(ctx, settings.Keep)
	}
	return item, nil
}

func (manager Manager) Open(ctx context.Context, name string) (io.ReadCloser, string, error) {
	clean, err := CleanName(name)
	if err != nil {
		return nil, "", err
	}
	store, err := manager.store()
	if err != nil {
		return nil, "", err
	}
	return store.Open(ctx, path.Join(Prefix, clean))
}

func (manager Manager) Delete(ctx context.Context, name string) error {
	clean, err := CleanName(name)
	if err != nil {
		return err
	}
	store, err := manager.store()
	if err != nil {
		return err
	}
	return store.Delete(ctx, path.Join(Prefix, clean))
}

func (manager Manager) Prune(ctx context.Context, keep int) error {
	if keep < 1 {
		keep = 1
	}
	list, err := manager.List(ctx)
	if err != nil {
		return err
	}
	if keep >= len(list.Items) {
		return nil
	}
	for _, item := range list.Items[keep:] {
		if err := manager.Delete(ctx, item.Name); err != nil {
			return err
		}
	}
	return nil
}

func (manager Manager) Due(ctx context.Context, settings Settings, now time.Time) (bool, error) {
	settings, err := CleanSettings(settings)
	if err != nil {
		return false, err
	}
	if !settings.Enabled {
		return false, nil
	}
	list, err := manager.List(ctx)
	if err != nil {
		return false, err
	}
	if list.Running != nil && !list.Running.Stale {
		return false, nil
	}
	schedule, err := cron.ParseStandard(settings.Cron)
	if err != nil {
		return false, err
	}
	if !schedule.Next(now.Add(-time.Minute)).Equal(now.Truncate(time.Minute)) {
		return false, nil
	}
	if len(list.Items) == 0 {
		return true, nil
	}
	latest := list.Items[0].CreatedAt
	return now.Sub(latest.In(now.Location())) >= time.Minute, nil
}

func (manager Manager) store() (utils.ObjectStore, error) {
	if manager.Store != nil {
		return manager.Store, nil
	}
	return utils.NewObjectStoreFromEnv()
}

func (manager Manager) runner() Runner {
	if manager.Runner != nil {
		return manager.Runner
	}
	return PgDumpRunner{}
}

func (manager Manager) now() time.Time {
	if manager.Now != nil {
		return manager.Now()
	}
	return time.Now()
}

func (manager Manager) dsn() string {
	if strings.TrimSpace(manager.DSN) != "" {
		return manager.DSN
	}
	return os.Getenv("DATABASE")
}

func itemFromObject(object utils.ObjectInfo) (Item, bool) {
	base := path.Base(object.Key)
	database, createdAt, ok := ParseName(base)
	if !ok {
		return Item{}, false
	}
	return Item{Name: base, Database: database, CreatedAt: createdAt, Size: object.Size}, true
}

func BuildName(database string, at time.Time) string {
	db := cleanDatabaseName(database)
	if db == "" {
		db = "doj"
	}
	return fmt.Sprintf("%s_%s.sql.gz", db, at.Format("2006-01-02_15-04-05"))
}

func ParseName(name string) (string, time.Time, bool) {
	matches := namePattern.FindStringSubmatch(name)
	if len(matches) != 4 {
		return "", time.Time{}, false
	}
	at, err := time.ParseInLocation("2006-01-02 15-04-05", matches[2]+" "+matches[3], time.Local)
	if err != nil {
		return "", time.Time{}, false
	}
	return matches[1], at, true
}

func CleanName(name string) (string, error) {
	if path.Base(name) != name {
		return "", fmt.Errorf("invalid backup name")
	}
	if _, _, ok := ParseName(name); !ok {
		return "", fmt.Errorf("invalid backup name")
	}
	return name, nil
}

func DatabaseName(dsn string) string {
	parsed, err := url.Parse(strings.TrimSpace(dsn))
	if err == nil && parsed.Path != "" {
		name := strings.Trim(strings.TrimSpace(parsed.Path), "/")
		if name != "" {
			return cleanDatabaseName(path.Base(name))
		}
	}
	return "doj"
}

func cleanDatabaseName(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	return strings.Trim(builder.String(), "_")
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].Name > items[j].Name
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func runningLock(ctx context.Context, now time.Time) (*Running, error) {
	var lock lockValue
	found, err := utils.CacheGet(ctx, lockKey, &lock)
	if err != nil || !found {
		return nil, err
	}
	return &Running{Name: lock.Name, StartedAt: lock.StartedAt, Stale: now.Sub(lock.StartedAt) > staleAfter}, nil
}

func settingKeys() []string {
	return []string{settingEnabled, settingKeep, settingCron}
}

func saveSetting(db *gorm.DB, key string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return db.Save(&models.Setting{Key: key, Value: datatypes.JSON(encoded)}).Error
}

func unmarshalSetting(data datatypes.JSON, value any) error {
	return json.Unmarshal(data, value)
}

var (
	ErrRunning     = errors.New("backup already running")
	ErrUnavailable = errors.New("backup unavailable")
)

type PgDumpRunner struct{}

func (PgDumpRunner) Dump(ctx context.Context, dsn string) (string, error) {
	if strings.TrimSpace(dsn) == "" {
		return "", fmt.Errorf("%w: DATABASE is required", ErrUnavailable)
	}
	if _, err := exec.LookPath("pg_dump"); err != nil {
		return "", fmt.Errorf("%w: pg_dump is required for database backups; install PostgreSQL client tools or use the DOJ container image: %v", ErrUnavailable, err)
	}
	file, err := os.CreateTemp("", "doj-backup-*.sql.gz")
	if err != nil {
		return "", err
	}
	filePath := file.Name()
	cleanup := true
	defer func() {
		_ = file.Close()
		if cleanup {
			_ = os.Remove(filePath)
		}
	}()

	cmd := exec.CommandContext(ctx, "pg_dump", "--no-owner", "--no-privileges", dsn)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return "", err
	}
	gzipWriter := gzip.NewWriter(file)
	_, copyErr := io.Copy(gzipWriter, stdout)
	closeErr := gzipWriter.Close()
	waitErr := cmd.Wait()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if waitErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return "", fmt.Errorf("%w: %s", waitErr, message)
		}
		return "", waitErr
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	cleanup = false
	return filePath, nil
}
