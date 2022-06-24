package router

import (
	"net/http"

	"github.com/doveccl/DOJ/server/database"
	"github.com/labstack/echo/v4"
)

func getConfig(c echo.Context) error {
	if k := c.Param("key"); k != "" {
		return c.JSON(http.StatusOK, echo.Map{k: database.PublicConfigs[k]})
	}
	return c.JSON(http.StatusOK, database.PublicConfigs)
}
