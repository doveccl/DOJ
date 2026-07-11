package admin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/auth"
	"github.com/doveccl/doj/server/judge"
	"github.com/labstack/echo/v4"
)

func (api *API) getJudgers(c echo.Context) error {
	judgers, err := api.judgers(c.Request().Context())
	if err != nil {
		return err
	}
	queue, err := api.queue()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Judgers{Judgers: judgers, Queue: queue})
}

func (api *API) updateJudger(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid judger id")
	if err != nil {
		return err
	}
	var req JudgerUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	cleanJudgerUpdate(&req)
	if err := validateJudger(req); err != nil {
		return err
	}

	updates := map[string]any{"name": req.Name}
	if req.Auth != "" {
		updates["auth"] = auth.TokenHash(req.Auth)
	}
	updated := api.db.Model(&models.Judger{}).Where("id = ?", id).Updates(updates)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "judger not found")
	}
	return api.getJudgers(c)
}

func (api *API) createJudger(c echo.Context) error {
	var req JudgerCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validateJudgerName(req.Name); err != nil {
		return err
	}

	token, err := auth.NewToken()
	if err != nil {
		return err
	}
	row := models.Judger{Name: req.Name, Auth: auth.TokenHash(token)}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	judgers, err := api.judgers(c.Request().Context())
	if err != nil {
		return err
	}
	for index := range judgers {
		if judgers[index].ID == row.ID {
			judgers[index].Token = token
			break
		}
	}
	queue, err := api.queue()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, Judgers{Judgers: judgers, Queue: queue})
}

func (api *API) deleteJudger(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid judger id")
	if err != nil {
		return err
	}

	deleted := api.db.Delete(&models.Judger{}, id)
	if deleted.Error != nil {
		return deleted.Error
	}
	if deleted.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "judger not found")
	}
	return api.getJudgers(c)
}

func (api *API) judgers(ctx context.Context) ([]Judger, error) {
	var rows []models.Judger
	if err := api.db.Order("id asc").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	items := make([]Judger, 0, len(rows))
	for _, row := range rows {
		status := judge.ReadStatus(ctx, row.ID, now)
		items = append(items, Judger{ID: row.ID, Name: row.Name, Online: status.Online, ConnectedAt: status.ConnectedAt, ActiveAt: status.ActiveAt, UptimeSeconds: status.UptimeSeconds})
	}
	return items, nil
}

func (api *API) queue() (JudgeQueue, error) {
	var queued int64
	if err := api.db.Model(&models.Submission{}).Where("status = ?", "queued").Count(&queued).Error; err != nil {
		return JudgeQueue{}, err
	}
	var running int64
	if err := api.db.Model(&models.Submission{}).Where("status = ?", "judging").Count(&running).Error; err != nil {
		return JudgeQueue{}, err
	}
	var done int64
	if err := api.db.Model(&models.Submission{}).Where("status NOT IN ?", []string{"queued", "judging"}).Count(&done).Error; err != nil {
		return JudgeQueue{}, err
	}
	return JudgeQueue{Queued: int(queued), Running: int(running), Done: int(done)}, nil
}

func cleanJudgerUpdate(req *JudgerUpdate) {
	req.Name = strings.TrimSpace(req.Name)
	req.Auth = strings.TrimSpace(req.Auth)
}

func validateJudger(req JudgerUpdate) error {
	return validateJudgerName(req.Name)
}

func validateJudgerName(name string) error {
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "judger name is required")
	}
	if len([]rune(name)) > maxNameRunes {
		return echo.NewHTTPError(http.StatusBadRequest, "judger name is too long")
	}
	return nil
}
