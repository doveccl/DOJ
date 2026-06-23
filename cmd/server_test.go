//go:build server

package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestIsDevOrigin(t *testing.T) {
	t.Setenv("DOJ_CORS_ORIGINS", "")
	config, err := corsConfig()
	if err != nil {
		t.Fatalf("corsConfig failed: %v", err)
	}
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

func TestWildcardCORSOriginIsRejected(t *testing.T) {
	t.Setenv("DOJ_CORS_ORIGINS", "*")
	_, err := corsConfig()
	if err == nil {
		t.Fatalf("wildcard CORS should fail configuration")
	}
}

func TestConfiguredCORSOriginsAreExact(t *testing.T) {
	t.Setenv("DOJ_CORS_ORIGINS", "https://doj.example.com, https://admin.example.com ")
	config, err := corsConfig()
	if err != nil {
		t.Fatalf("corsConfig failed: %v", err)
	}
	if len(config.AllowOrigins) != 2 {
		t.Fatalf("AllowOrigins = %+v, want two exact origins", config.AllowOrigins)
	}
	if config.AllowOrigins[0] != "https://doj.example.com" || config.AllowOrigins[1] != "https://admin.example.com" {
		t.Fatalf("AllowOrigins = %+v", config.AllowOrigins)
	}
	if config.AllowOriginFunc != nil {
		t.Fatalf("exact configured origins should not install a wildcard AllowOriginFunc")
	}
}

func TestBootstrapAdminConfigDefaultsAndOverrides(t *testing.T) {
	t.Setenv("DOJ_BOOTSTRAP_ADMIN", "")
	t.Setenv("DOJ_BOOTSTRAP_MAIL", "")
	t.Setenv("DOJ_BOOTSTRAP_PASSWORD", "")

	name, mail, password := bootstrapAdminConfig()
	if name != "admin" || mail != "admin@localhost" || password != "admin" {
		t.Fatalf("bootstrapAdminConfig defaults = %q, %q, %q", name, mail, password)
	}

	t.Setenv("DOJ_BOOTSTRAP_ADMIN", "root")
	t.Setenv("DOJ_BOOTSTRAP_MAIL", "root@example.com")
	t.Setenv("DOJ_BOOTSTRAP_PASSWORD", "secret")
	name, mail, password = bootstrapAdminConfig()
	if name != "root" || mail != "root@example.com" || password != "secret" {
		t.Fatalf("bootstrapAdminConfig overrides = %q, %q, %q", name, mail, password)
	}
}

func TestDatabaseDSNDefaultAndOverride(t *testing.T) {
	t.Setenv("DOJ_DATABASE_URL", "")
	t.Setenv("DATABASE_URL", "")
	if got := databaseDSN(); got != defaultDatabaseURL {
		t.Fatalf("databaseDSN default = %q, want %q", got, defaultDatabaseURL)
	}
	t.Setenv("DATABASE_URL", "postgres://database-url")
	if got := databaseDSN(); got != "postgres://database-url" {
		t.Fatalf("databaseDSN DATABASE_URL = %q", got)
	}
	t.Setenv("DOJ_DATABASE_URL", "postgres://doj-database-url")
	if got := databaseDSN(); got != "postgres://doj-database-url" {
		t.Fatalf("databaseDSN DOJ_DATABASE_URL = %q", got)
	}
}

func TestBodyLimitDefaultAndOverride(t *testing.T) {
	t.Setenv("DOJ_BODY_LIMIT", "")
	if got := bodyLimit(); got != "160M" {
		t.Fatalf("bodyLimit default = %q, want 160M", got)
	}
	t.Setenv("DOJ_BODY_LIMIT", "8M")
	if got := bodyLimit(); got != "8M" {
		t.Fatalf("bodyLimit override = %q, want 8M", got)
	}
}

func TestHTTPTimeoutDefaultsAndOverrides(t *testing.T) {
	t.Setenv("DOJ_READ_HEADER_TIMEOUT", "")
	t.Setenv("DOJ_READ_TIMEOUT", "")
	t.Setenv("DOJ_WRITE_TIMEOUT", "")
	t.Setenv("DOJ_IDLE_TIMEOUT", "")
	timeouts, err := httpTimeoutsFromEnv()
	if err != nil {
		t.Fatalf("httpTimeoutsFromEnv default failed: %v", err)
	}
	if timeouts.ReadHeader != 5*time.Second {
		t.Fatalf("ReadHeader default = %v, want 5s", timeouts.ReadHeader)
	}
	if timeouts.Read != 0 {
		t.Fatalf("Read default = %v, want disabled", timeouts.Read)
	}
	if timeouts.Write != 0 {
		t.Fatalf("Write default = %v, want disabled", timeouts.Write)
	}
	if timeouts.Idle != 60*time.Second {
		t.Fatalf("Idle default = %v, want 60s", timeouts.Idle)
	}

	t.Setenv("DOJ_READ_HEADER_TIMEOUT", "2s")
	t.Setenv("DOJ_READ_TIMEOUT", "30s")
	t.Setenv("DOJ_WRITE_TIMEOUT", "45s")
	t.Setenv("DOJ_IDLE_TIMEOUT", "90s")
	timeouts, err = httpTimeoutsFromEnv()
	if err != nil {
		t.Fatalf("httpTimeoutsFromEnv override failed: %v", err)
	}
	if timeouts.ReadHeader != 2*time.Second || timeouts.Read != 30*time.Second || timeouts.Write != 45*time.Second || timeouts.Idle != 90*time.Second {
		t.Fatalf("httpTimeoutsFromEnv override = %+v", timeouts)
	}
}

func TestHTTPTimeoutsRejectInvalidValues(t *testing.T) {
	cases := []struct {
		key   string
		value string
	}{
		{key: "DOJ_READ_HEADER_TIMEOUT", value: "0s"},
		{key: "DOJ_READ_HEADER_TIMEOUT", value: "-1s"},
		{key: "DOJ_READ_TIMEOUT", value: "-1s"},
		{key: "DOJ_WRITE_TIMEOUT", value: "soon"},
		{key: "DOJ_IDLE_TIMEOUT", value: "0s"},
	}
	for _, tc := range cases {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			t.Setenv("DOJ_READ_HEADER_TIMEOUT", "")
			t.Setenv("DOJ_READ_TIMEOUT", "")
			t.Setenv("DOJ_WRITE_TIMEOUT", "")
			t.Setenv("DOJ_IDLE_TIMEOUT", "")
			t.Setenv(tc.key, tc.value)
			if _, err := httpTimeoutsFromEnv(); err == nil {
				t.Fatalf("httpTimeoutsFromEnv should reject %s=%q", tc.key, tc.value)
			}
		})
	}
}

func TestConfigureHTTPServerTimeouts(t *testing.T) {
	e := echo.New()
	timeouts := httpTimeouts{
		ReadHeader: 2 * time.Second,
		Read:       30 * time.Second,
		Write:      45 * time.Second,
		Idle:       90 * time.Second,
	}
	configureHTTPServer(e, timeouts)
	if e.Server.ReadHeaderTimeout != timeouts.ReadHeader {
		t.Fatalf("ReadHeaderTimeout = %v, want %v", e.Server.ReadHeaderTimeout, timeouts.ReadHeader)
	}
	if e.Server.ReadTimeout != timeouts.Read {
		t.Fatalf("ReadTimeout = %v, want %v", e.Server.ReadTimeout, timeouts.Read)
	}
	if e.Server.WriteTimeout != timeouts.Write {
		t.Fatalf("WriteTimeout = %v, want %v", e.Server.WriteTimeout, timeouts.Write)
	}
	if e.Server.IdleTimeout != timeouts.Idle {
		t.Fatalf("IdleTimeout = %v, want %v", e.Server.IdleTimeout, timeouts.Idle)
	}
}

func TestShutdownTimeoutConfig(t *testing.T) {
	t.Setenv("DOJ_SHUTDOWN_TIMEOUT", "")
	timeout, err := shutdownTimeout()
	if err != nil {
		t.Fatalf("shutdownTimeout default failed: %v", err)
	}
	if timeout != 15*time.Second {
		t.Fatalf("shutdownTimeout default = %v, want 15s", timeout)
	}

	t.Setenv("DOJ_SHUTDOWN_TIMEOUT", "2500ms")
	timeout, err = shutdownTimeout()
	if err != nil {
		t.Fatalf("shutdownTimeout override failed: %v", err)
	}
	if timeout != 2500*time.Millisecond {
		t.Fatalf("shutdownTimeout override = %v, want 2500ms", timeout)
	}
}

func TestShutdownTimeoutRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"0s", "-1s", "soon"} {
		t.Setenv("DOJ_SHUTDOWN_TIMEOUT", value)
		if _, err := shutdownTimeout(); err == nil {
			t.Fatalf("shutdownTimeout(%q) should fail", value)
		}
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

func TestRegisterWebAppRejectsMissingIndex(t *testing.T) {
	err := registerWebApp(echo.New(), t.TempDir())
	if err == nil {
		t.Fatalf("registerWebApp should reject a directory without index.html")
	}
}

func mustWriteFile(t *testing.T, file string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", file, err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", file, err)
	}
}
