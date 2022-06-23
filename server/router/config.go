package router

import (
	"net/http"

	"github.com/doveccl/DOJ/server/database"
	"github.com/labstack/echo/v4"
)

func getConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, database.PublicConfigs)
}
