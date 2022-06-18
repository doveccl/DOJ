package ws

import (
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{}

func Upgrade(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()
	for {
		_, m, e := ws.ReadMessage()
		if e != nil {
			return nil
		}
		fmt.Println(string(m))
	}
}
