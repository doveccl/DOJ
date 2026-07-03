package backup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/doveccl/doj/utils"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const scheduleLockTTL = 2 * time.Minute

type Scheduler struct {
	ctx     context.Context
	manager Manager
	cron    *cron.Cron
	mu      sync.Mutex
	entry   cron.EntryID
}

func StartScheduler(ctx context.Context, db *gorm.DB) *Scheduler {
	scheduler := NewScheduler(ctx, Manager{DB: db})
	settings, err := ReadSettings(db)
	if err != nil {
		slog.Warn("read backup settings failed", "err", err)
	} else if err := scheduler.Reload(settings); err != nil {
		slog.Warn("load backup schedule failed", "err", err)
	}
	scheduler.Start()
	return scheduler
}

func NewScheduler(ctx context.Context, manager Manager) *Scheduler {
	return &Scheduler{
		ctx:     ctx,
		manager: manager,
		cron:    cron.New(cron.WithLocation(time.Local)),
	}
}

func (scheduler *Scheduler) Start() {
	scheduler.cron.Start()
	go func() {
		<-scheduler.ctx.Done()
		scheduler.cron.Stop()
	}()
}

func (scheduler *Scheduler) Reload(settings Settings) error {
	settings, err := CleanSettings(settings)
	if err != nil {
		return err
	}
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	scheduler.removeLocked()
	if !settings.Enabled {
		return nil
	}
	entry, err := scheduler.cron.AddFunc(settings.Cron, scheduler.backup)
	if err != nil {
		return err
	}
	scheduler.entry = entry
	return nil
}

func (scheduler *Scheduler) removeLocked() {
	if scheduler.entry != 0 {
		scheduler.cron.Remove(scheduler.entry)
		scheduler.entry = 0
	}
}

func (scheduler *Scheduler) backup() {
	ok, err := scheduler.acquireScheduleLock(time.Now())
	if err != nil {
		slog.Warn("scheduled backup lock failed", "err", err)
		return
	}
	if !ok {
		slog.Info("scheduled backup skipped", "err", ErrRunning)
		return
	}
	if _, err := scheduler.manager.BackupNow(scheduler.ctx); err != nil {
		if errors.Is(err, ErrRunning) {
			slog.Info("scheduled backup skipped", "err", err)
			return
		}
		slog.Warn("scheduled backup failed", "err", err)
	}
}

func (scheduler *Scheduler) acquireScheduleLock(now time.Time) (bool, error) {
	key := fmt.Sprintf("%s:schedule:%s", lockKey, now.In(time.Local).Format("200601021504"))
	return utils.CacheSetNX(scheduler.ctx, key, true, scheduleLockTTL)
}
