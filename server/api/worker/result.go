package worker

import (
	"net/http"
	"time"

	"github.com/doveccl/doj/contract/judger"
	"github.com/doveccl/doj/contract/limits"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/events"
	"github.com/doveccl/doj/server/judge"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (api *API) heartbeat(c echo.Context) error {
	var req judger.HeartbeatRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := validateHeartbeat(req); err != nil {
		return err
	}

	taskID, err := parseTaskID(c)
	if err != nil {
		return err
	}
	if taskID != req.SubmissionID {
		return c.NoContent(http.StatusAccepted)
	}

	applied := false
	ownerID := uint(0)
	now := time.Now()
	err = api.db.Transaction(func(tx *gorm.DB) error {
		var submission models.Submission
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&submission, req.SubmissionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		if submission.Attempt != req.Attempt {
			return nil
		}
		if err := api.requireTaskOwner(c, submission); err != nil {
			return err
		}
		if submission.JudgerID != nil {
			ownerID = *submission.JudgerID
		}
		until := now.Add(defaultLeaseSeconds * time.Second)
		if err := tx.Model(&models.Submission{}).Where("id = ? AND attempt = ?", submission.ID, req.Attempt).Update("lease_until", until).Error; err != nil {
			return err
		}
		applied = true
		return nil
	})
	if err != nil {
		return err
	}
	if applied {
		judge.TouchStatus(c.Request().Context(), ownerID, now)
		if req.Stage != "" {
			_ = judge.SaveProgress(c.Request().Context(), req.SubmissionID, judge.Progress{
				Attempt:   req.Attempt,
				Stage:     req.Stage,
				Done:      req.Done,
				Total:     req.Total,
				UpdatedAt: now,
			})
			events.SubmissionProgressChanged()
		}
	}
	return c.NoContent(http.StatusAccepted)
}

func (api *API) result(c echo.Context) error {
	var req judger.ResultRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := validateResult(req); err != nil {
		return err
	}

	taskID, err := parseTaskID(c)
	if err != nil {
		return err
	}
	if taskID != req.SubmissionID {
		return c.NoContent(http.StatusAccepted)
	}

	applied := false
	ownerID := uint(0)
	err = api.db.Transaction(func(tx *gorm.DB) error {
		var submission models.Submission
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&submission, req.SubmissionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		if submission.Attempt != req.Attempt {
			return nil
		}
		if err := api.requireTaskOwner(c, submission); err != nil {
			return err
		}
		if submission.JudgerID != nil {
			ownerID = *submission.JudgerID
		}
		update := map[string]any{
			"status":      req.Status,
			"score":       req.Score,
			"message":     req.Message,
			"time_ms":     req.TimeMS,
			"memory_kb":   req.MemoryKB,
			"judger_id":   nil,
			"lease_until": nil,
		}
		if err := tx.Model(&models.Submission{}).Where("id = ?", req.SubmissionID).Updates(update).Error; err != nil {
			return err
		}
		if len(req.Cases) > 0 {
			if err := tx.Where("submission_id = ?", req.SubmissionID).Delete(&models.Case{}).Error; err != nil {
				return err
			}
			for _, item := range req.Cases {
				row := models.Case{
					SubmissionID: req.SubmissionID,
					No:           item.No,
					CaseID:       item.ID,
					Status:       item.Status,
					Score:        item.Score,
					TimeMS:       item.TimeMS,
					MemoryKB:     item.MemoryKB,
					Message:      item.Message,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			}
		}
		applied = true
		return nil
	})
	if err != nil {
		return err
	}
	if applied {
		judge.TouchStatus(c.Request().Context(), ownerID, time.Now())
		judge.DeleteProgress(c.Request().Context(), req.SubmissionID)
		events.SubmissionChanged()
	}
	return c.NoContent(http.StatusAccepted)
}

func validateResult(req judger.ResultRequest) error {
	if req.SubmissionID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid result target")
	}
	if req.Attempt <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid result attempt")
	}
	if !validVerdict(req.Status) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid result status")
	}
	if req.Score < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid result score")
	}
	if len([]byte(req.Message)) > limits.MaxJudgerMessageBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "result message is too large")
	}
	if invalidUsage(req.TimeMS, limits.MaxProblemTimeMS*2) || invalidUsage(req.MemoryKB, limits.MaxProblemMemoryMB*1024) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid result resource usage")
	}
	if len(req.Cases) > limits.MaxJudgerCases {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "too many case results")
	}
	caseNumbers := make(map[int]struct{}, len(req.Cases))
	for _, item := range req.Cases {
		if item.No <= 0 || item.No > len(req.Cases) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid case number")
		}
		if _, exists := caseNumbers[item.No]; exists {
			return echo.NewHTTPError(http.StatusBadRequest, "duplicate case number")
		}
		caseNumbers[item.No] = struct{}{}
		if !validVerdict(item.Status) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid case status")
		}
		if item.ID == "" || len(item.ID) > 128 || item.Score < 0 || invalidUsage(item.TimeMS, limits.MaxProblemTimeMS*2) || invalidUsage(item.MemoryKB, limits.MaxProblemMemoryMB*1024) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid case result")
		}
		if len([]rune(item.Message)) > models.CaseMessageMax {
			return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "case message is too large")
		}
	}
	return nil
}

func invalidUsage(value *int, max int) bool {
	return value != nil && (*value < 0 || *value > max)
}

func validateHeartbeat(req judger.HeartbeatRequest) error {
	if req.SubmissionID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid heartbeat target")
	}
	if req.Attempt <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid heartbeat attempt")
	}
	if req.Stage != "" && !judge.ValidProgressStage(req.Stage) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid heartbeat stage")
	}
	if req.Done < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid heartbeat progress")
	}
	if req.Total != nil && *req.Total < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid heartbeat progress total")
	}
	return nil
}

func validVerdict(status string) bool {
	switch status {
	case "AC", "CE", "WA", "PE", "TLE", "MLE", "OLE", "RE", "SE":
		return true
	default:
		return false
	}
}
