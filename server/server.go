package server

import (
	"flag"
	"os"

	"github.com/doveccl/DOJ/server/database"
	"github.com/doveccl/DOJ/server/middleware"
	"github.com/doveccl/DOJ/server/router"
	"github.com/doveccl/DOJ/server/websocket"
	"github.com/labstack/echo/v4"
)

var listen = ":7974"
var dsn = "user=postgres"

func init() {
	flag.StringVar(&dsn, "d", dsn, "postgres dsn")
	flag.StringVar(&listen, "l", listen, "listen address")
}

func parse() {
	flag.Parse()
	if d := os.Getenv("DOJ_DSN"); d != "" {
		dsn = d
	}
	if l := os.Getenv("DOJ_LISTEN"); l != "" {
		listen = l
	}
}

func Start() {
	e := echo.New()
	e.HideBanner = true
	middleware.RegisterMiddlewares(e)
	router.RegisterRoutes(e)
	websocket.Attach(e)

	if parse(); database.Connect(dsn) == nil {
		e.Logger.Fatal(e.Start(listen))
	} else {
		os.Exit(1)
	}
}
