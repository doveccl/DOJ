package middleware

import (
	"github.com/doveccl/DOJ/server/database"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func Auth() echo.MiddlewareFunc {
	auth := middleware.JWTWithConfig(middleware.JWTConfig{
		TokenLookup: "header:token",
		Claims:      &database.UserJWT{},
		SigningKey:  []byte(database.ConfMap["secret"]),
	})
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return auth(func(c echo.Context) error {
			c.Set("user", c.Get("user").(*jwt.Token).Claims)
			return next(c)
		})
	}
}
