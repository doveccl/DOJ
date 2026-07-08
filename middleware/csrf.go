package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/doveccl/doj/common/authn"
	"github.com/labstack/echo/v4"
)

func CSRF() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if safeMethod(c.Request().Method) {
				return next(c)
			}
			if _, ok := authn.SessionToken(c); !ok {
				return next(c)
			}
			cookie, err := c.Cookie(authn.CSRFCookie)
			if err != nil || cookie.Value == "" {
				return echo.NewHTTPError(http.StatusForbidden, "missing csrf token")
			}
			header := c.Request().Header.Get(authn.CSRFHeader)
			if header == "" || subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) != 1 {
				return echo.NewHTTPError(http.StatusForbidden, "invalid csrf token")
			}
			return next(c)
		}
	}
}

func safeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
