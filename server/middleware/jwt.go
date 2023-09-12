package middleware

import (
	"github.com/doveccl/DOJ/server/database"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

func Auth() echo.MiddlewareFunc {
	auth := echojwt.JWT([]byte(database.ConfMap["secret"]))
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return auth(func(c echo.Context) error {
			m := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
			c.Set("uid", m["UID"])
			c.Set("group", m["Group"])
			return next(c)
		})
	}
}
