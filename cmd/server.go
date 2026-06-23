//go:build server

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	dojmw "github.com/doveccl/doj/middleware"
	"github.com/doveccl/doj/models"
	adminsvc "github.com/doveccl/doj/services/admin"
	judgersvc "github.com/doveccl/doj/services/judger"
	websvc "github.com/doveccl/doj/services/web"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

func main() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.BodyLimit(bodyLimit()))
	corsConfig, err := corsConfig()
	if err != nil {
		e.Logger.Fatal(err)
	}
	e.Use(middleware.CORSWithConfig(corsConfig))
	e.Use(dojmw.CSRF())

	if err := utils.ConfigureCookiesFromEnv(); err != nil {
		e.Logger.Fatal(err)
	}

	db, err := openDB()
	if err != nil {
		e.Logger.Fatal(err)
	}
	websvc.Register(e, db)
	adminsvc.Register(e, db)
	judgersvc.Register(e, db)
	if err := registerWebApp(e, webDir()); err != nil {
		e.Logger.Fatal(err)
	}

	addr := getenv("DOJ_ADDR", ":7974")
	serverTimeouts, err := httpTimeoutsFromEnv()
	if err != nil {
		e.Logger.Fatal(err)
	}
	configureHTTPServer(e, serverTimeouts)
	timeout, err := shutdownTimeout()
	if err != nil {
		e.Logger.Fatal(err)
	}
	if err := startServer(e, addr, timeout); err != nil {
		e.Logger.Fatal(err)
	}
}

func startServer(e *echo.Echo, addr string, timeout time.Duration) error {
	serverErr := make(chan error, 1)
	go func() {
		err := e.Start(addr)
		if err != nil && err != http.ErrServerClosed {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case err := <-serverErr:
		return err
	case sig := <-stop:
		slog.Info("shutting down server", "signal", sig.String(), "timeout", timeout.String())
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := e.Shutdown(ctx); err != nil {
			return err
		}
		select {
		case err := <-serverErr:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func openDB() (*gorm.DB, error) {
	dsn := databaseDSN()
	db, err := models.Open(dsn)
	if err != nil {
		return nil, err
	}
	if err := models.AutoMigrate(db); err != nil {
		return nil, err
	}
	if err := models.EnsureProblemIDBase(db); err != nil {
		return nil, err
	}
	admin, mail, password := bootstrapAdminConfig()
	if err := models.EnsureAdmin(db, admin, mail, password); err != nil {
		return nil, err
	}
	if err := models.EnsureDefaultLanguage(db); err != nil {
		return nil, err
	}
	return db, nil
}

func bootstrapAdminConfig() (string, string, string) {
	return getenv("DOJ_BOOTSTRAP_ADMIN", "admin"),
		getenv("DOJ_BOOTSTRAP_MAIL", "admin@localhost"),
		getenv("DOJ_BOOTSTRAP_PASSWORD", "admin")
}

func databaseDSN() string {
	return getenv("DOJ_DATABASE_URL", getenv("DATABASE_URL", defaultDatabaseURL))
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func bodyLimit() string {
	return getenv("DOJ_BODY_LIMIT", "160M")
}

func webDir() string {
	return getenv("DOJ_WEB_DIR", "")
}

type httpTimeouts struct {
	ReadHeader time.Duration
	Read       time.Duration
	Write      time.Duration
	Idle       time.Duration
}

func configureHTTPServer(e *echo.Echo, timeouts httpTimeouts) {
	e.Server.ReadHeaderTimeout = timeouts.ReadHeader
	e.Server.ReadTimeout = timeouts.Read
	e.Server.WriteTimeout = timeouts.Write
	e.Server.IdleTimeout = timeouts.Idle
}

func httpTimeoutsFromEnv() (httpTimeouts, error) {
	readHeader, err := durationFromEnv("DOJ_READ_HEADER_TIMEOUT", "5s", false)
	if err != nil {
		return httpTimeouts{}, err
	}
	read, err := durationFromEnv("DOJ_READ_TIMEOUT", "0s", true)
	if err != nil {
		return httpTimeouts{}, err
	}
	write, err := durationFromEnv("DOJ_WRITE_TIMEOUT", "0s", true)
	if err != nil {
		return httpTimeouts{}, err
	}
	idle, err := durationFromEnv("DOJ_IDLE_TIMEOUT", "60s", false)
	if err != nil {
		return httpTimeouts{}, err
	}
	return httpTimeouts{
		ReadHeader: readHeader,
		Read:       read,
		Write:      write,
		Idle:       idle,
	}, nil
}

func shutdownTimeout() (time.Duration, error) {
	return durationFromEnv("DOJ_SHUTDOWN_TIMEOUT", "15s", false)
}

func durationFromEnv(key string, fallback string, allowZero bool) (time.Duration, error) {
	raw := getenv(key, fallback)
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", key, raw, err)
	}
	if duration < 0 {
		return 0, fmt.Errorf("%s must not be negative", key)
	}
	if duration == 0 && !allowZero {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return duration, nil
}

const defaultDatabaseURL = "postgres://postgres@localhost/postgres?sslmode=disable"

func corsConfig() (middleware.CORSConfig, error) {
	origins := getenv("DOJ_CORS_ORIGINS", "")
	config := middleware.CORSConfig{
		AllowCredentials: true,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "X-DOJ-CSRF"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
	}
	if origins == "" {
		config.AllowOriginFunc = func(origin string) (bool, error) { return isDevOrigin(origin), nil }
		return config, nil
	}
	if origins == "*" {
		return config, fmt.Errorf("DOJ_CORS_ORIGINS=* is not allowed with credentialed requests; configure exact origins")
	}
	for _, origin := range strings.Split(origins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			config.AllowOrigins = append(config.AllowOrigins, origin)
		}
	}
	return config, nil
}

func isDevOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	host := parsed.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

func registerWebApp(e *echo.Echo, root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	index := filepath.Join(root, "index.html")
	info, err := os.Stat(index)
	if err != nil {
		return fmt.Errorf("invalid DOJ_WEB_DIR %q: %w", root, err)
	}
	if info.IsDir() {
		return fmt.Errorf("invalid DOJ_WEB_DIR %q: index.html is a directory", root)
	}
	handler := webAppHandler(root, index)
	e.GET("/*", handler)
	e.HEAD("/*", handler)
	return nil
}

func webAppHandler(root string, index string) echo.HandlerFunc {
	return func(c echo.Context) error {
		requestPath := c.Request().URL.Path
		if requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") {
			return echo.ErrNotFound
		}
		file, cache, found := webFile(root, index, requestPath)
		if !found {
			return echo.ErrNotFound
		}
		if cache {
			c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=31536000, immutable")
		}
		return c.File(file)
	}
}

func webFile(root string, index string, requestPath string) (string, bool, bool) {
	clean := path.Clean("/" + requestPath)
	rel := strings.TrimPrefix(clean, "/")
	if rel == "" {
		return index, false, true
	}
	file := filepath.Join(root, filepath.FromSlash(rel))
	if isRegularFile(file) {
		return file, strings.HasPrefix(rel, "assets/"), true
	}
	if strings.HasPrefix(rel, "assets/") {
		return "", false, false
	}
	return index, false, true
}

func isRegularFile(file string) bool {
	info, err := os.Stat(file)
	return err == nil && info.Mode().IsRegular()
}
