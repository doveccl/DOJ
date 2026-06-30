//go:build jplag

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	defaultJPlagListen  = ":7979"
	defaultJPlagJar     = "/app/jplag.jar"
	defaultJava         = "/opt/java/openjdk/bin/java"
	defaultViewerPort   = "1996"
	defaultViewerReport = "/app/viewer.jplag"
)

func main() {
	if strings.TrimSpace(os.Getenv("VIEWER")) == "" {
		stop, err := startBundledViewer()
		if err != nil {
			slog.Error("failed to start bundled jplag viewer", "error", err)
		} else {
			defer stop()
		}
	}
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.GET("/ready", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })
	e.POST("/run", runJPlagHandler)
	e.GET("/viewer/*", viewerHandler)
	e.GET("/JPlag/*", viewerHandler)
	if err := startJPlagServer(e); err != nil {
		e.Logger.Fatal(err)
	}
}

func startJPlagServer(e *echo.Echo) error {
	serverErr := make(chan error, 1)
	go func() {
		err := e.Start(getenv("LISTEN", defaultJPlagListen))
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
		slog.Info("shutting down jplag service", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := e.Shutdown(ctx); err != nil {
			return err
		}
		return <-serverErr
	}
}

func runJPlagHandler(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	work, err := os.MkdirTemp("", "doj-jplag-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	input := filepath.Join(work, "input.zip")
	if err := writeFile(input, src); err != nil {
		return err
	}
	if err := unzip(input, filepath.Join(work, "src")); err != nil {
		return err
	}
	for _, dir := range []string{"new", "old"} {
		if err := os.MkdirAll(filepath.Join(work, "src", dir), 0o755); err != nil {
			return err
		}
	}
	report := filepath.Join(work, "report.jplag")
	if err := runJPlag(c.Request().Context(), filepath.Join(work, "src", "new"), filepath.Join(work, "src", "old"), report); err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}
	return c.File(report)
}

func viewerHandler(c echo.Context) error {
	target := strings.TrimSpace(os.Getenv("VIEWER"))
	if target == "" {
		target = "http://127.0.0.1:" + getenv("VIEWER_PORT", defaultViewerPort) + "/"
	}
	base, err := url.Parse(target)
	if err != nil {
		return err
	}
	proxy := &httputil.ReverseProxy{Rewrite: func(req *httputil.ProxyRequest) {
		req.SetURL(base)
		rel := strings.TrimPrefix(c.Param("*"), "/")
		req.Out.URL.Path = joinURLPath(base.Path, rel)
		if strings.HasSuffix(c.Request().URL.Path, "/") && !strings.HasSuffix(req.Out.URL.Path, "/") {
			req.Out.URL.Path += "/"
		}
		req.Out.URL.RawQuery = c.Request().URL.RawQuery
		req.Out.Header.Del(echo.HeaderCookie)
	}, ModifyResponse: rewriteViewerHTML}
	proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}

func startBundledViewer() (func(), error) {
	port := getenv("VIEWER_PORT", defaultViewerPort)
	cmd := exec.Command(
		getenv("JAVA", defaultJava),
		"-jar", getenv("JAR", defaultJPlagJar),
		"--mode", "view",
		"--port", port,
		getenv("VIEWER_REPORT", defaultViewerReport),
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	stop := func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://127.0.0.1:" + port + "/")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return stop, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	stop()
	return nil, errors.New("bundled viewer did not become ready")
}

func rewriteViewerHTML(resp *http.Response) error {
	contentType := resp.Header.Get(echo.HeaderContentType)
	if !strings.Contains(contentType, "text/html") {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	text := strings.ReplaceAll(string(body), `src="/assets/`, `src="/JPlag/assets/`)
	text = strings.ReplaceAll(text, `href="/assets/`, `href="/JPlag/assets/`)
	text = strings.ReplaceAll(text, `href="/favicon.ico"`, `href="/JPlag/favicon.ico"`)
	data := []byte(text)
	resp.Body = io.NopCloser(bytes.NewReader(data))
	resp.ContentLength = int64(len(data))
	resp.Header.Set(echo.HeaderContentLength, strconv.Itoa(len(data)))
	return nil
}

func runJPlag(ctx context.Context, newDir string, oldDir string, report string) error {
	jar := getenv("JAR", defaultJPlagJar)
	args := []string{
		"-jar", jar,
		"--mode", "run",
		"--language", "cpp",
		"--new", newDir,
		"--old", oldDir,
		"--normalize",
		"--csv-export",
		"--result-file", report,
	}
	cmd := exec.CommandContext(ctx, getenv("JAVA", defaultJava), args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, jplagError(output))
	}
	if _, err := os.Stat(report); err != nil {
		return fmt.Errorf("JPlag did not create report: %w", err)
	}
	return nil
}

func jplagError(output []byte) string {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	items := lines[:0]
	for _, line := range lines {
		if strings.Contains(line, "JPlagVersionChecker") {
			continue
		}
		items = append(items, strings.TrimSpace(line))
	}
	return strings.Join(items, "\n")
}

func writeFile(name string, src io.Reader) error {
	dst, err := os.Create(name)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func unzip(zipFile string, root string) error {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		name := filepath.Clean(file.Name)
		if strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return errors.New("invalid zip path")
		}
		target := filepath.Join(root, name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		err = writeFile(target, src)
		_ = src.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func joinURLPath(base string, rel string) string {
	base = strings.TrimRight(base, "/")
	rel = strings.TrimLeft(rel, "/")
	if rel == "" {
		return base + "/"
	}
	return base + "/" + rel
}

func getenv(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
