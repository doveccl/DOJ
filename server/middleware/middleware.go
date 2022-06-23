package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var logger = middleware.LoggerWithConfig(middleware.LoggerConfig{
	CustomTimeFormat: "2006/01/02 15:04:05",
	Format:           "${time_custom} ${status} ${method} ${path} ${latency_human}\n",
})

var static = middleware.StaticWithConfig(middleware.StaticConfig{
	Root:  "dist",
	HTML5: true,
})

func RegisterMiddlewares(e *echo.Echo) {
	e.Use(logger)
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())
	e.Use(static)
}
