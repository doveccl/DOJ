package middleware

import (
	"github.com/doveccl/DOJ/server/database"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type UserJWT struct {
	ID    uint
	Group uint
	jwt.StandardClaims
}

func Auth() echo.MiddlewareFunc {
	auth := middleware.JWTWithConfig(middleware.JWTConfig{
		Claims:      &UserJWT{},
		TokenLookup: "header:token",
		SigningKey:  []byte(database.PrivateConfigs["secret"]),
	})
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return auth(func(c echo.Context) error {
			c.Set("user", c.Get("user").(*jwt.Token).Claims)
			return next(c)
		})
	}
}
