package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
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
	adminsvc "github.com/doveccl/doj/server/admin"
	backupsvc "github.com/doveccl/doj/server/backup"
	judgersvc "github.com/doveccl/doj/server/judger"
	websvc "github.com/doveccl/doj/server/web"
	"github.com/doveccl/doj/utils"
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

	if err := utils.CachePing(context.Background()); err != nil {
		e.Logger.Fatal(err)
	}

	db, err := openDB()
	if err != nil {
		e.Logger.Fatal(err)
	}
	backupScheduler := backupsvc.StartScheduler(ctx, db)
	websvc.Register(e, db)
	adminsvc.Register(e, db, backupScheduler)
	judgersvc.Register(e, db)
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
		if c.QueryParam("jplag") != "" {
			return proxyJPlagViewer(c, "")
		}
		file, cache, found := webFile(root, index, requestPath)
		if !found {
			if strings.HasPrefix(strings.TrimPrefix(path.Clean("/"+requestPath), "/"), "assets/") {
				return proxyJPlagViewer(c, strings.TrimPrefix(path.Clean("/"+requestPath), "/"))
			}
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

func proxyJPlagViewer(c echo.Context, rel string) error {
	target := strings.TrimRight(strings.TrimSpace(os.Getenv("JPLAG")), "/")
	if target == "" {
		return echo.ErrNotFound
	}
	base, err := url.Parse(target)
	if err != nil {
		return err
	}
	proxy := &httputil.ReverseProxy{Rewrite: func(req *httputil.ProxyRequest) {
		req.SetURL(base)
		req.Out.URL.Path = "/" + strings.TrimLeft(path.Join(base.Path, "viewer", rel), "/")
		req.Out.URL.RawQuery = c.Request().URL.RawQuery
		req.Out.Header.Del(echo.HeaderCookie)
	}, ModifyResponse: func(resp *http.Response) error {
		resp.Header.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' https: data: blob:; font-src 'self' data:; connect-src 'self' https://api.github.com")
		return nil
	}}
	proxy.ServeHTTP(c.Response(), c.Request())
	return nil
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
