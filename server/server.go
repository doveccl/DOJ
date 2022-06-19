package server

import (
	"flag"
	"os"

	"github.com/doveccl/DOJ/server/db"
	"github.com/doveccl/DOJ/server/ws"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var addr = flag.String("l", "127.0.0.1:7974", "listen address")
var dsn = flag.String("d", "user=postgres password=postgres", "postgresql dsn")

func Start() {
	flag.Parse()

	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		CustomTimeFormat: "2006/01/02 15:04:05",
		Format:           "[${time_custom}] ${status} ${method} ${path} ${latency_human}\n",
	}))
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll: true,
	}))
	e.Use(middleware.Gzip())

	e.GET("/ws", ws.Upgrade)
	api := e.Group("/api")
	api.GET("/test", nil)

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:  "dist",
		HTML5: true,
	}))

	if db.Connect(*dsn) != nil || e.Start(*addr) != nil {
		os.Exit(1)
	}
}
