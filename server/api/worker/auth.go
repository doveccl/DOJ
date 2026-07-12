package worker

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/doveccl/doj/contract/judger"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/auth"
	"github.com/doveccl/doj/server/judge"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (api *API) auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if isDirectLoopbackRequest(c) {
			c.Set(contextJudgerLocal, true)
			return next(c)
		}
		token, ok := bearerToken(c)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
		}
		var row models.Judger
		if err := api.db.First(&row, "auth = ?", auth.TokenHash(token)).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
			}
			return err
		}
		judge.TouchStatus(c.Request().Context(), row.ID, time.Now())
		c.Set(contextJudgerID, row.ID)
		return next(c)
	}
}

func (api *API) ensureJudger(c echo.Context, req judger.LeaseRequest) (uint, error) {
	if id, ok := c.Get(contextJudgerID).(uint); ok && id > 0 {
		if host := strings.TrimSpace(req.Host); host != "" {
			if err := api.db.Model(&models.Judger{}).Where("id = ? AND name = ?", id, "").Update("name", host).Error; err != nil {
				return 0, err
			}
		}
		return id, nil
	}
	if !isLocalJudger(c) {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
	}
	name := judgerRequestName(req)
	var row models.Judger
	if err := api.db.First(&row, "name = ?", name).Error; err == nil {
		c.Set(contextJudgerID, row.ID)
		judge.TouchStatus(c.Request().Context(), row.ID, time.Now())
		return row.ID, nil
	} else if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	row = models.Judger{Name: name, Auth: ""}
	if err := api.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoNothing: true}).Create(&row).Error; err != nil {
		return 0, err
	}
	if err := api.db.First(&row, "name = ?", name).Error; err != nil {
		return 0, err
	}
	c.Set(contextJudgerID, row.ID)
	judge.TouchStatus(c.Request().Context(), row.ID, time.Now())
	return row.ID, nil
}

func (api *API) authorizeSubmission(c echo.Context, submissionID uint) error {
	if isLocalJudger(c) {
		return nil
	}
	judgerID, ok := c.Get(contextJudgerID).(uint)
	if !ok || judgerID == 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
	}
	var submission models.Submission
	if err := api.db.First(&submission, submissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "task not found")
		}
		return err
	}
	if submission.JudgerID == nil || *submission.JudgerID != judgerID {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
	}
	return nil
}

func (api *API) requireTaskOwner(c echo.Context, submission models.Submission) error {
	if isLocalJudger(c) {
		return nil
	}
	id := taskJudgerID(c)
	if id == 0 || submission.JudgerID == nil || *submission.JudgerID != id {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid judger auth")
	}
	return nil
}

func taskJudgerID(c echo.Context) uint {
	id, _ := c.Get(contextJudgerID).(uint)
	return id
}

func bearerToken(c echo.Context) (string, bool) {
	header := strings.TrimSpace(c.Request().Header.Get("Authorization"))
	value, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return value, true
}

func isDirectLoopbackRequest(c echo.Context) bool {
	if hasForwardedHeaders(c) {
		return false
	}
	host, _, err := net.SplitHostPort(c.Request().RemoteAddr)
	if err != nil {
		host = c.Request().RemoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && ip.IsLoopback()
}

func hasForwardedHeaders(c echo.Context) bool {
	headers := c.Request().Header
	for _, name := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-IP"} {
		if strings.TrimSpace(headers.Get(name)) != "" {
			return true
		}
	}
	return false
}

func isLocalJudger(c echo.Context) bool {
	value, _ := c.Get(contextJudgerLocal).(bool)
	return value
}

func judgerRequestName(req judger.LeaseRequest) string {
	name := strings.TrimSpace(req.Host)
	if name != "" {
		return name
	}
	return "local-judger"
}
