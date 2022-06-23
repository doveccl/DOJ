package router

import (
	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo) {
	api := e.Group("/api")
	api.GET("/config", getConfig)
}
