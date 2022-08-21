package router

import (
	"net/http"

	"github.com/doveccl/DOJ/server/database"
	"github.com/doveccl/DOJ/server/util"
	"github.com/labstack/echo/v4"
)

type LoginForm struct {
	User string `form:"user" json:"user"`
	Pass string `form:"pass" json:"pass"`
}

type RegisterForm struct {
	Name string `form:"name" json:"name"`
	Mail string `form:"mail" json:"mail"`
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

func register(c echo.Context) error {
	if database.PublicConfigs["registration"] != "1" {
		return echo.ErrForbidden
	}

	f := RegisterForm{}
	if c.Bind(&f) != nil ||
		!util.ValidateName(f.Name) ||
		!util.ValidateMail(f.Mail) ||
		len(f.Pass) < 8 || len(f.Pass) > 20 {
		return echo.ErrBadRequest
	}

	if database.GetUser(f.Name).ID > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "duplicate_name")
	}
	if database.GetUser(f.Mail).ID > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "duplicate_mail")
	}

	u := database.User{Name: f.Name, Mail: f.Mail, Group: 1}
	if u.SetPass(f.Pass) == nil && u.Create() {
		return c.JSON(http.StatusOK, &u)
	}
	return echo.ErrInternalServerError
}

func self(c echo.Context) error {
	u := c.Get("user").(*database.UserJWT)
	return c.JSON(http.StatusOK, database.GetUser(u.ID))
}
