package worker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/doveccl/doj/contract/judger"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/events"
	"github.com/doveccl/doj/server/judge"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (api *API) lease(c echo.Context) error {
	var req judger.LeaseRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Version != judger.Version {
		return echo.NewHTTPError(http.StatusUpgradeRequired, "judger version mismatch")
	}

	judgerID, err := api.ensureJudger(c, req)
	if err != nil {
		return err
	}

	ch, unsubscribe := events.Default.Subscribe()
	defer unsubscribe()
	timer := time.NewTimer(api.longPollWait())
	defer timer.Stop()
	active := time.NewTicker(5 * time.Second)
	defer active.Stop()

	for {
		payload, err := api.tryLease(c.Request().Context(), judgerID)
		if err != nil {
			return err
		}
		if payload != nil {
			events.SubmissionChanged()
			return c.JSON(http.StatusOK, judger.LeaseResponse{Task: payload})
		}

		select {
		case <-c.Request().Context().Done():
			return nil
		case <-timer.C:
			return c.JSON(http.StatusOK, judger.LeaseResponse{})
		case now := <-active.C:
			judge.TouchStatus(c.Request().Context(), judgerID, now)
		case <-ch:
		}
	}
}

func (api *API) longPollWait() time.Duration {
	if api.leaseWait > 0 {
		return api.leaseWait
	}
	return defaultLeaseWait
}

func (api *API) tryLease(ctx context.Context, judgerID uint) (*judger.TaskPayload, error) {
	now := time.Now()
	var leased *models.Submission
	err := api.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []models.Submission
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? OR (status = ? AND (lease_until IS NULL OR lease_until < ?))", "queued", "judging", now).
			Order("created_at asc").
			Limit(1).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		submission := rows[0]
		nextAttempt := submission.Attempt + 1
		until := now.Add(defaultLeaseSeconds * time.Second)
		updates := map[string]any{
			"status":      "judging",
			"attempt":     nextAttempt,
			"judger_id":   judgerID,
			"lease_until": until,
		}
		if err := tx.Model(&models.Submission{}).Where("id = ?", submission.ID).Updates(updates).Error; err != nil {
			return err
		}
		submission.Status = "judging"
		submission.Attempt = nextAttempt
		submission.JudgerID = &judgerID
		submission.LeaseUntil = &until
		leased = &submission
		return nil
	})
	if err != nil {
		return nil, err
	}
	if leased == nil {
		return nil, nil
	}
	payload, err := buildPayload(ctx, api.db.WithContext(ctx), *leased)
	if err != nil {
		resetErr := api.db.WithContext(ctx).Model(&models.Submission{}).
			Where("id = ? AND status = ? AND attempt = ? AND judger_id = ?", leased.ID, "judging", leased.Attempt, judgerID).
			Updates(map[string]any{"status": "queued", "judger_id": nil, "lease_until": nil}).Error
		if resetErr != nil {
			return nil, fmt.Errorf("build task payload: %v; reset lease: %w", err, resetErr)
		}
		return nil, err
	}
	_ = judge.SaveProgress(ctx, payload.SubmissionID, judge.Progress{Attempt: payload.Attempt, Stage: "leased", UpdatedAt: now})
	return payload, nil
}
