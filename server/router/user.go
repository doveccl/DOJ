package router

import (
	"net/http"

	"github.com/doveccl/DOJ/server/database"
	"github.com/labstack/echo/v4"
)

type LoginForm struct {
	User string `form:"user" json:"user"`
	Pass string `form:"pass" json:"pass"`
}

func login(c echo.Context) error {
	if f := (LoginForm{}); c.Bind(&f) != nil {
		return echo.ErrBadRequest
	} else if u := database.GetUser(f.User); !u.Compare(f.Pass) {
		return echo.ErrUnauthorized
	} else if t, e := u.GetToken(); e != nil {
		return echo.ErrInternalServerError
	} else {
		return c.JSON(http.StatusOK, echo.Map{"token": t})
	}
}

func self(c echo.Context) error {
	u := c.Get("user").(*database.UserJWT)
	return c.JSON(http.StatusOK, database.GetUser(u.ID))
}
