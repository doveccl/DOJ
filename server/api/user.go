package api

import "github.com/labstack/echo/v4"

func Login(c echo.Context) error {
	return c.String(200, "")
}
