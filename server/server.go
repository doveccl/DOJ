package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/api/admin"
	"github.com/doveccl/doj/server/api/public"
	"github.com/doveccl/doj/server/api/worker"
	"github.com/doveccl/doj/server/backup"
	"github.com/doveccl/doj/server/cache"
	dojmw "github.com/doveccl/doj/server/middleware"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

const (
	defaultListen      = ":7974"
	defaultDatabaseURL = "postgres://postgres@localhost"
	defaultWebDir      = "dist"
	shutdownGrace      = 15 * time.Second
)

func Main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(securityHeaders())
	e.Use(dojmw.CSRF())

	if err := cache.Ping(context.Background()); err != nil {
		e.Logger.Fatal(err)
	}

	db, err := openDB()
	if err != nil {
		e.Logger.Fatal(err)
	}
	backupScheduler := backup.StartScheduler(ctx, db)
	public.Register(e, db)
	admin.Register(e, db, backupScheduler)
	worker.Register(e, db)
	if err := registerWebApp(e, defaultWebDir); err != nil {
		e.Logger.Fatal(err)
	}

	if err := startServer(e); err != nil {
		e.Logger.Fatal(err)
	}
}

func startServer(e *echo.Echo) error {
	serverErr := make(chan error, 1)
	go func() {
		err := e.Start(listenAddr())
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
		slog.Info("shutting down server", "signal", sig.String(), "timeout", shutdownGrace.String())
		ctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
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
	if err := models.EnsureAdmin(db, "admin", "admin@localhost", "admin"); err != nil {
		return nil, err
	}
	if err := models.EnsureDefaultLanguage(db); err != nil {
		return nil, err
	}
	return db, nil
}

func databaseDSN() string {
	return getenv("DATABASE", defaultDatabaseURL)
}

func listenAddr() string {
	return getenv("LISTEN", defaultListen)
}

func getenv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func registerWebApp(e *echo.Echo, root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	index := filepath.Join(root, "index.html")
	info, err := os.Stat(index)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return nil
	}
	handler := webAppHandler(root, index)
	e.GET("/*", handler)
	e.HEAD("/*", handler)
	return nil
}

func securityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set(echo.HeaderXContentTypeOptions, "nosniff")
			h.Set(echo.HeaderXFrameOptions, "SAMEORIGIN")
			h.Set("Referrer-Policy", "same-origin")
			h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' https: data: blob:; font-src 'self' data:; connect-src 'self'")
			return next(c)
		}
	}
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
		} else if file == index {
			c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
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
