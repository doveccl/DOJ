package admin

import (
	"net/http"
	"time"

	"github.com/doveccl/doj/common/authn"
	"github.com/doveccl/doj/common/limits"
	"github.com/doveccl/doj/models"
	backupsvc "github.com/doveccl/doj/server/backup"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

type API struct {
	db              *gorm.DB
	backupScheduler *backupsvc.Scheduler
}

const maxNameRunes = models.NameMax

func Register(e *echo.Echo, db *gorm.DB, backupScheduler *backupsvc.Scheduler) {
	if db == nil {
		panic("admin API requires a database")
	}
	api := &API{db: db, backupScheduler: backupScheduler}
	group := e.Group("/api/admin", api.requireAdmin)
	group.GET("/settings", api.getSettings)
	group.PATCH("/settings", api.updateSettings, echomw.BodyLimit(limits.BodySettings))
	group.GET("/members", api.members)
	group.GET("/users", api.usersPage)
	group.POST("/users", api.createUser)
	group.PATCH("/users/:name", api.updateUser)
	group.DELETE("/users/:name", api.deleteUser)
	group.POST("/users/:name/password", api.resetUserPassword)
	group.GET("/groups", api.groupsPage)
	group.POST("/groups", api.createGroup)
	group.PATCH("/groups/:id", api.updateGroup)
	group.DELETE("/groups/:id", api.deleteGroup)
	group.GET("/languages", api.getLanguages)
	group.POST("/languages", api.createLanguage, echomw.BodyLimit(limits.BodyLanguage))
	group.PATCH("/languages/:id", api.updateLanguage, echomw.BodyLimit(limits.BodyLanguage))
	group.DELETE("/languages/:id", api.deleteLanguage)
	group.GET("/judgers", api.getJudgers)
	group.POST("/judgers", api.createJudger)
	group.PATCH("/judgers/:id", api.updateJudger)
	group.DELETE("/judgers/:id", api.deleteJudger)
	group.GET("/backups/settings", api.backupSettings)
	group.PATCH("/backups/settings", api.updateBackupSettings, echomw.BodyLimit(limits.BodySettings))
	group.GET("/backups", api.backups)
	group.POST("/backups", api.createBackup)
	group.GET("/backups/:name/download", api.downloadBackup)
	group.DELETE("/backups/:name", api.deleteBackup)
}

func (api *API) requireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !api.isAdmin(c) {
			return echo.NewHTTPError(http.StatusForbidden, "admin required")
		}
		return next(c)
	}
}

func (api *API) isAdmin(c echo.Context) bool {

	user, err := authn.UserFromCookie(api.db, c, time.Now())
	return err == nil && user.Admin
}
