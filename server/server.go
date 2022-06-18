package server

import (
	"github.com/doveccl/DOJ/server/ws"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var logger = middleware.LoggerWithConfig(middleware.LoggerConfig{
	CustomTimeFormat: "2006/01/02 15:04:05",
	Format:           "[${time_custom}] ${status} ${method} ${path} ${latency_human}\n",
})

var recover = middleware.RecoverWithConfig(middleware.RecoverConfig{
	DisableStackAll: true,
})

var static = middleware.StaticWithConfig(middleware.StaticConfig{
	Root:  "dist",
	HTML5: true,
})

func Start() {
	e := echo.New()
	e.HideBanner = true

	e.Use(logger)
	e.Use(recover)
	e.Use(middleware.Gzip())
	e.Use(static)

	e.GET("/ws", ws.Upgrade)
	// api := e.Group("/api")

	e.Logger.Fatal(e.Start(":7974"))
}
