package admin

import (
	"errors"
	"net/http"

	backupsvc "github.com/doveccl/doj/server/backup"
	"github.com/labstack/echo/v4"
)

func (api *API) backupSettings(c echo.Context) error {
	settings, err := backupsvc.ReadSettings(api.db)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
}

func (api *API) updateBackupSettings(c echo.Context) error {
	var req backupsvc.Settings
	if err := c.Bind(&req); err != nil {
		return err
	}
	settings, err := backupsvc.CleanSettings(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := backupsvc.SaveSettings(api.db, settings); err != nil {
		return err
	}
	if api.backupScheduler != nil {
		if err := api.backupScheduler.Reload(settings); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, settings)
}

func (api *API) backups(c echo.Context) error {
	manager := backupsvc.Manager{DB: api.db}
	list, err := manager.List(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, list)
}

func (api *API) createBackup(c echo.Context) error {
	manager := backupsvc.Manager{DB: api.db}
	item, err := manager.BackupNow(c.Request().Context())
	if err != nil {
		if errors.Is(err, backupsvc.ErrRunning) {
			return echo.NewHTTPError(http.StatusConflict, "backup already running")
		}
		if errors.Is(err, backupsvc.ErrUnavailable) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, err.Error())
		}
		return err
	}
	return c.JSON(http.StatusCreated, item)
}

func (api *API) downloadBackup(c echo.Context) error {
	name := c.Param("name")
	manager := backupsvc.Manager{DB: api.db}
	reader, contentType, err := manager.Open(c.Request().Context(), name)
	if err != nil {
		return err
	}
	defer reader.Close()
	if contentType == "" {
		contentType = "application/gzip"
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+name+`"`)
	return c.Stream(http.StatusOK, contentType, reader)
}

func (api *API) deleteBackup(c echo.Context) error {
	manager := backupsvc.Manager{DB: api.db}
	if err := manager.Delete(c.Request().Context(), c.Param("name")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
