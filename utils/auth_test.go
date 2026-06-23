package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestSessionCookiesUseAutoTLSByDefault(t *testing.T) {
	resetCookieConfig(t)

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
}

func TestSessionCookiesCanBeForcedForProductionProxy(t *testing.T) {
	resetCookieConfig(t)
	secure := true
	ConfigureCookies(CookieConfig{
		Domain:   "example.com",
		Secure:   &secure,
		SameSite: http.SameSiteNoneMode,
	})

	cookies := issueSessionCookies("http://api.example.com/")
	for _, name := range []string{SessionCookie, CSRFCookie} {
		cookie := cookieByName(cookies, name)
		if cookie == nil {
			t.Fatalf("missing %s", name)
		}
		if cookie.Domain != "example.com" {
			t.Fatalf("%s domain = %q, want example.com", name, cookie.Domain)
		}
		if !cookie.Secure {
			t.Fatalf("%s should be secure", name)
		}
		if cookie.SameSite != http.SameSiteNoneMode {
			t.Fatalf("%s SameSite = %v, want none", name, cookie.SameSite)
		}
	}
}

func resetCookieConfig(t *testing.T) {
	t.Helper()
	ConfigureCookies(CookieConfig{SameSite: http.SameSiteLaxMode})
	t.Cleanup(func() {
		ConfigureCookies(CookieConfig{SameSite: http.SameSiteLaxMode})
	})
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
