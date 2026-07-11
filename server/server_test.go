package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestDatabaseDSNDefaultAndOverride(t *testing.T) {
	t.Setenv("DATABASE", "")
	if got := databaseDSN(); got != defaultDatabaseURL {
		t.Fatalf("databaseDSN default = %q, want %q", got, defaultDatabaseURL)
	}
	t.Setenv("DATABASE", "postgres://doj@localhost/doj")
	if got := databaseDSN(); got != "postgres://doj@localhost/doj" {
		t.Fatalf("databaseDSN override = %q", got)
	}
}

func TestListenAddrDefaultAndOverride(t *testing.T) {
	t.Setenv("LISTEN", "")
	if got := listenAddr(); got != defaultListen {
		t.Fatalf("listenAddr default = %q, want %q", got, defaultListen)
	}
	t.Setenv("LISTEN", "127.0.0.1:7975")
	if got := listenAddr(); got != "127.0.0.1:7975" {
		t.Fatalf("listenAddr override = %q", got)
	}
}

func TestHTTPServerBoundsRequestReadsWithoutTimingOutSSEWrites(t *testing.T) {
	e := echo.New()
	configureHTTPServer(e)
	if e.Server.ReadHeaderTimeout != readHeaderTimeout || e.Server.ReadTimeout != readTimeout || e.Server.IdleTimeout != idleTimeout {
		t.Fatalf("unexpected HTTP timeouts: %+v", e.Server)
	}
	if e.Server.MaxHeaderBytes != maxHeaderBytes {
		t.Fatalf("max header bytes = %d, want %d", e.Server.MaxHeaderBytes, maxHeaderBytes)
	}
	if e.Server.WriteTimeout != 0 {
		t.Fatalf("write timeout = %v, SSE requires no global write deadline", e.Server.WriteTimeout)
	}
}

func TestTrustedProxyIPExtractorRejectsClientXFF(t *testing.T) {
	extract := trustedProxyIPExtractor()
	direct := httptest.NewRequest(http.MethodGet, "/", nil)
	direct.RemoteAddr = "198.51.100.10:1234"
	direct.Header.Set(echo.HeaderXForwardedFor, "203.0.113.99")
	if got := extract(direct); got != "198.51.100.10" {
		t.Fatalf("public client spoofed XFF: got %q", got)
	}
	private := httptest.NewRequest(http.MethodGet, "/", nil)
	private.RemoteAddr = "172.20.0.9:1234"
	private.Header.Set(echo.HeaderXForwardedFor, "203.0.113.99")
	if got := extract(private); got != "172.20.0.9" {
		t.Fatalf("private client spoofed XFF: got %q", got)
	}

	proxied := httptest.NewRequest(http.MethodGet, "/", nil)
	proxied.RemoteAddr = "127.0.0.1:1234"
	proxied.Header.Set(echo.HeaderXForwardedFor, "198.51.100.10")
	if got := extract(proxied); got != "198.51.100.10" {
		t.Fatalf("trusted proxy client IP = %q", got)
	}
}

func TestRegisterWebAppSkipsMissingDist(t *testing.T) {
	e := echo.New()
	if err := registerWebApp(e, filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("missing web dir should be ignored: %v", err)
	}
}

func TestSecurityHeadersAllowHTTPSImages(t *testing.T) {
	e := echo.New()
	e.Use(securityHeaders())
	e.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	res := httptest.NewRecorder()
	e.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))

	csp := res.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "img-src 'self' https: data: blob:") {
		t.Fatalf("content security policy = %q, want https images allowed", csp)
	}
	if strings.Contains(csp, "script-src 'self' https:") {
		t.Fatalf("content security policy = %q, should not allow external scripts", csp)
	}
}

func TestRegisterWebAppFallback(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "index.html"), "<html>app</html>")
	mustWriteFile(t, filepath.Join(dir, "assets", "app.js"), "console.log('app')")

	e := echo.New()
	e.GET("/api/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "api")
	})
	if err := registerWebApp(e, dir); err != nil {
		t.Fatalf("registerWebApp failed: %v", err)
	}

	cases := []struct {
		path   string
		status int
		body   string
		cache  string
	}{
		{path: "/", status: http.StatusOK, body: "<html>app</html>", cache: "no-store"},
		{path: "/problems/1000", status: http.StatusOK, body: "<html>app</html>", cache: "no-store"},
		{path: "/assets/app.js", status: http.StatusOK, body: "console.log('app')", cache: "public, max-age=31536000, immutable"},
		{path: "/assets/missing.js", status: http.StatusNotFound},
		{path: "/api/health", status: http.StatusOK, body: "api"},
		{path: "/api/missing", status: http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			res := httptest.NewRecorder()
			e.ServeHTTP(res, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if res.Code != tc.status {
				t.Fatalf("%s status = %d body=%q, want %d", tc.path, res.Code, res.Body.String(), tc.status)
			}
			if tc.body != "" && res.Body.String() != tc.body {
				t.Fatalf("%s body = %q, want %q", tc.path, res.Body.String(), tc.body)
			}
			if got := res.Header().Get(echo.HeaderCacheControl); got != tc.cache {
				t.Fatalf("%s cache control = %q, want %q", tc.path, got, tc.cache)
			}
		})
	}
}

func mustWriteFile(t *testing.T, file string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", file, err)
	}
}
