package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
