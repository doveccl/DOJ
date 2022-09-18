package router

import (
	"net/http"

	"github.com/doveccl/DOJ/server/database"
	"github.com/labstack/echo/v4"
)

func getConfig(c echo.Context) error {
	res := map[string]string{}
	for _, c := range database.ConfList {
		if c.Public {
			res[c.Key] = c.Value
		}
	}
	return c.JSON(http.StatusOK, res)
}
