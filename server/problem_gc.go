package server

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	problemdata "github.com/doveccl/doj/server/problem"
	"github.com/doveccl/doj/server/storage"
	"gorm.io/gorm"
)

const (
	problemPackageGCInterval = 24 * time.Hour
	problemPackageGCGrace    = 24 * time.Hour
)

func startProblemPackageGC(ctx context.Context, db *gorm.DB) {
	store, err := storage.NewFromEnv()
	if err != nil {
		slog.Warn("problem package GC disabled", "err", err)
		return
	}
	go func() {
		run := func() {
			deleted, err := collectProblemPackages(ctx, db, store, time.Now().Add(-problemPackageGCGrace))
			if err != nil {
				slog.Warn("problem package GC failed", "err", err)
			} else if deleted > 0 {
				slog.Info("problem package GC complete", "deleted", deleted)
			}
		}
		run()
		ticker := time.NewTicker(problemPackageGCInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

func collectProblemPackages(ctx context.Context, db *gorm.DB, store storage.Store, before time.Time) (int, error) {
	keep := map[string]bool{}
	protected := map[uint]bool{}
	var problems []models.Problem
	if err := db.WithContext(ctx).Select("id", "package").Find(&problems).Error; err != nil {
		return 0, err
	}
	for _, row := range problems {
		item, err := problemdata.Parse(row.Package)
		if err != nil {
			protected[row.ID] = true
			slog.Warn("problem package GC skipped invalid row", "problem", row.ID, "err", err)
			continue
		}
		if item.Hash != "" {
			keep[problemdata.ObjectKey(row.ID, item.Hash)] = true
		}
	}
	var active []uint
	if err := db.WithContext(ctx).Model(&models.Submission{}).Distinct().Where("status = ?", "judging").Pluck("problem_id", &active).Error; err != nil {
		return 0, err
	}
	for _, id := range active {
		protected[id] = true
	}
	objects, err := store.List(ctx, "problems")
	if err != nil {
		return 0, err
	}
	deleted := 0
	var deleteErr error
	for _, object := range objects {
		id, ok := packageObjectProblemID(object.Key)
		if !ok || keep[object.Key] || protected[id] || !object.UpdatedAt.Before(before) {
			continue
		}
		if err := store.Delete(ctx, object.Key); err != nil {
			deleteErr = errors.Join(deleteErr, err)
			continue
		}
		deleted++
	}
	return deleted, deleteErr
}

func packageObjectProblemID(key string) (uint, bool) {
	parts := strings.Split(key, "/")
	if len(parts) != 4 || parts[0] != "problems" || parts[2] != "packages" || !strings.HasSuffix(parts[3], ".zip") {
		return 0, false
	}
	id, err := strconv.ParseUint(parts[1], 10, 64)
	hash := strings.TrimSuffix(parts[3], ".zip")
	decoded, hashErr := hex.DecodeString(hash)
	return uint(id), err == nil && id > 0 && hashErr == nil && len(decoded) == 32
}
