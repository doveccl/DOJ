package router

import (
	"github.com/doveccl/DOJ/server/middleware"
	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo) {
	api := e.Group("/api")
	api.POST("/login", login)
	api.GET("/config", getConfig)
	api.GET("/config/:key", getConfig)

	api.Use(middleware.Auth())
	api.GET("/self", self)
}
