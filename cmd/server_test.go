//go:build server

package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestIsDevOrigin(t *testing.T) {
	config := corsConfig()
	cases := []struct {
		origin string
		want   bool
	}{
		{origin: "http://localhost:28080", want: true},
		{origin: "http://127.0.0.1:28080", want: true},
		{origin: "http://" + net.IPv4(172, 16, 0, 10).String() + ":28080", want: true},
		{origin: "https://example.com", want: false},
		{origin: "not-a-url", want: false},
	}
	for _, tc := range cases {
		got, err := config.AllowOriginFunc(tc.origin)
		if err != nil {
			t.Fatalf("AllowOriginFunc(%q) returned error: %v", tc.origin, err)
		}
		if got != tc.want {
			t.Fatalf("AllowOriginFunc(%q) = %v, want %v", tc.origin, got, tc.want)
		}
	}
}

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

func TestRegisterWebAppSkipsMissingDist(t *testing.T) {
	e := echo.New()
	if err := registerWebApp(e, filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("missing web dir should be ignored: %v", err)
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
	}{
		{path: "/", status: http.StatusOK, body: "<html>app</html>"},
		{path: "/problems/1000", status: http.StatusOK, body: "<html>app</html>"},
		{path: "/assets/app.js", status: http.StatusOK, body: "console.log('app')"},
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
