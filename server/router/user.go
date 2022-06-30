package router

import (
	"net/http"
	"time"

	"github.com/doveccl/DOJ/server/database"
	"github.com/doveccl/DOJ/server/middleware"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func sign(u *database.User) (string, error) {
	m := jwt.SigningMethodHS256
	k := []byte(database.PrivateConfigs["secret"])
	o := middleware.UserJWT{ID: u.ID, Group: u.Group}
	o.ExpiresAt = time.Now().Add(720 * time.Hour).Unix()
	return jwt.NewWithClaims(m, o).SignedString(k)
}

func compare(u *database.User, p string) bool {
	return bcrypt.CompareHashAndPassword(u.Auth, []byte(p)) == nil
}

func login(c echo.Context) error {
	user := c.FormValue("user")
	pass := c.FormValue("pass")

	u := database.GetUser(user)
	if u == nil || !compare(u, pass) {
		return echo.ErrUnauthorized
	} else if t, e := sign(u); e != nil {
		return echo.ErrInternalServerError
	} else {
		return c.JSON(http.StatusOK, echo.Map{"token": t})
	}
}

func self(c echo.Context) error {
	u := c.Get("user").(*middleware.UserJWT)
	return c.JSON(http.StatusOK, database.GetUser(u.ID))
}
