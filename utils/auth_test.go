package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestSessionCookiesUseAutoTLSByDefault(t *testing.T) {
	httpCookies := issueSessionCookies("http://example.test/")
	httpSession := cookieByName(httpCookies, SessionCookie)
	if httpSession == nil {
		t.Fatalf("missing %s", SessionCookie)
	}
	if httpSession.Secure {
		t.Fatalf("http request should not set secure cookie in auto mode")
	}
	if httpSession.SameSite != http.SameSiteLaxMode {
		t.Fatalf("default SameSite = %v, want lax", httpSession.SameSite)
	}

	httpsCookies := issueSessionCookies("https://example.test/")
	httpsSession := cookieByName(httpsCookies, SessionCookie)
	if httpsSession == nil {
		t.Fatalf("missing %s", SessionCookie)
	}
	if !httpsSession.Secure {
		t.Fatalf("https request should set secure cookie in auto mode")
	}
	httpsCSRF := cookieByName(httpsCookies, CSRFCookie)
	if httpsCSRF == nil {
		t.Fatalf("missing %s", CSRFCookie)
	}
	if httpsCSRF.HttpOnly {
		t.Fatalf("%s must be readable by the browser for the csrf header", CSRFCookie)
	}
}

func issueSessionCookies(target string) []*http.Cookie {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, target, nil), rec)
	SetSessionCookie(c, "session-token", time.Now().Add(time.Hour))
	return rec.Result().Cookies()
}

func cookieByName(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
