package admin

import (
	"errors"
	"net/http"

	"github.com/doveccl/doj/server/settings"
	"github.com/labstack/echo/v4"
)

func (api *API) getSettings(c echo.Context) error {
	settings, err := settings.Get(api.db)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
}

func (api *API) updateSettings(c echo.Context) error {
	var req AdminSettingsPatch
	if err := c.Bind(&req); err != nil {
		return err
	}
	current, err := settings.Update(api.db, req)
	if errors.Is(err, settings.ErrEmptyPatch) {
		return echo.NewHTTPError(http.StatusBadRequest, "settings patch is empty")
	}
	if errors.Is(err, settings.ErrSiteNameEmpty) {
		return echo.NewHTTPError(http.StatusBadRequest, "site name is required")
	}
	if errors.Is(err, settings.ErrSiteNameTooLong) {
		return echo.NewHTTPError(http.StatusBadRequest, "site name is too long")
	}
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, current)
}
