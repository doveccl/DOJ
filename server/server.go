package server

import (
	"os"

	"github.com/doveccl/DOJ/server/database"
	"github.com/doveccl/DOJ/server/middleware"
	"github.com/doveccl/DOJ/server/router"
	"github.com/doveccl/DOJ/server/websocket"
	"github.com/labstack/echo/v4"
)

var env = map[string]string{
	"ADDR": ":7974",
	"DSN":  "host=localhost user=postgres",
}

func Start() {
	for k := range env {
		if e := os.Getenv("DOJ_" + k); e != "" {
			env[k] = e
		}
	}
	if database.Connect(env["DSN"]) == nil {
		e := echo.New()
		e.HideBanner = true
		middleware.RegisterMiddlewares(e)
		router.RegisterRoutes(e)
		websocket.Attach(e)
		e.Logger.Fatal(e.Start(env["ADDR"]))
	} else {
		os.Exit(1)
	}
}
