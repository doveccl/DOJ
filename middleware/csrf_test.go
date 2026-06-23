package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
)

func TestCSRFRequiresHeaderForSessionMutation(t *testing.T) {
	e := echo.New()
	e.Use(CSRF())
	e.POST("/write", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	utils.SetSessionCookie(c, "session-token", time.Now().Add(time.Hour))
	cookies := rec.Result().Cookies()

	req := httptest.NewRequest(http.MethodPost, "/write", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("write without csrf got %d, want 403", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/write", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
		if cookie.Name == utils.CSRFCookie {
			req.Header.Set(utils.CSRFHeader, cookie.Value)
		}
	}
	res = httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("write with csrf got %d, want 204", res.Code)
	}
}
