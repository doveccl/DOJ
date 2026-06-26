package backup

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

const schedulerTick = time.Minute

func StartScheduler(ctx context.Context, db *gorm.DB) {
	manager := Manager{DB: db}
	go func() {
		ticker := time.NewTicker(schedulerTick)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				settings, err := ReadSettings(db)
				if err != nil {
					slog.Warn("read backup settings failed", "err", err)
					continue
				}
				due, err := manager.Due(ctx, settings, now)
				if err != nil {
					slog.Warn("check backup schedule failed", "err", err)
					continue
				}
				if !due {
					continue
				}
				if _, err := manager.BackupNow(ctx); err != nil {
					slog.Warn("scheduled backup failed", "err", err)
				}
			}
		}
	}()
}
