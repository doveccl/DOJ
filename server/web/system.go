package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/server/web/contract"

	"github.com/doveccl/doj/models"
	adminsvc "github.com/doveccl/doj/server/admin"
	"github.com/doveccl/doj/server/events"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
)

func (api *API) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (api *API) ready(c echo.Context) error {

	sqlDB, err := api.db.DB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
	}
	if err := sqlDB.PingContext(c.Request().Context()); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}

func (api *API) events(c echo.Context) error {
	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "streaming is not supported")
	}

	response := c.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set(echo.HeaderConnection, "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)

	if err := writeSSE(response.Writer, "ready", []byte("{}")); err != nil {
		return nil
	}
	flusher.Flush()

	ch, unsubscribe := events.Default.Subscribe()
	defer unsubscribe()
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			if err := writeSSE(response.Writer, event.Type, event.Data); err != nil {
				return nil
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := io.WriteString(response.Writer, ": ping\n\n"); err != nil {
				return nil
			}
			flusher.Flush()
		}
	}
}

func writeSSE(writer io.Writer, event string, data []byte) error {
	if _, err := fmt.Fprintf(writer, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := writer.Write([]byte("data: ")); err != nil {
		return err
	}
	lines := bytes.Split(data, []byte("\n"))
	for index, line := range lines {
		if index > 0 {
			if _, err := writer.Write([]byte("\ndata: ")); err != nil {
				return err
			}
		}
		if _, err := writer.Write(line); err != nil {
			return err
		}
	}
	_, err := writer.Write([]byte("\n\n"))
	return err
}

func (api *API) site(c echo.Context) error {
	settings, err := adminsvc.Settings(api.db)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
}

func (api *API) languages(c echo.Context) error {

	var rows []models.Language
	if err := api.db.Order("id asc").Limit(100).Find(&rows).Error; err != nil {
		return err
	}
	items := make([]contract.Language, 0, len(rows))
	for _, row := range rows {
		items = append(items, langView(row))
	}
	return c.JSON(http.StatusOK, items)
}

func (api *API) uploadImage(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "image file is required")
	}
	data, mime, sha, ext, err := readUploadedImage(file)
	if err != nil {
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	year, month, day := uploadDateParts(time.Now())
	key := path.Join("users", strconv.Itoa(int(user.ID)), year, month, day, sha+ext)
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	if err := store.Put(c.Request().Context(), key, bytes.NewReader(data), int64(len(data)), mime); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, contract.UploadResult{URL: "/" + path.Join("api", key)})
}

func readUploadedImage(file *multipart.FileHeader) ([]byte, string, string, string, error) {
	src, err := file.Open()
	if err != nil {
		return nil, "", "", "", err
	}
	defer src.Close()

	const maxImageBytes = 5 << 20
	data, err := io.ReadAll(io.LimitReader(src, maxImageBytes+1))
	if err != nil {
		return nil, "", "", "", err
	}
	if len(data) > maxImageBytes {
		return nil, "", "", "", echo.NewHTTPError(http.StatusRequestEntityTooLarge, "image is too large")
	}
	mime := http.DetectContentType(data)
	if !strings.HasPrefix(mime, "image/") {
		return nil, "", "", "", echo.NewHTTPError(http.StatusBadRequest, "image file is required")
	}
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	ext := uploadExt(file.Filename, mime)
	return data, mime, sha, ext, nil
}

func uploadDateParts(now time.Time) (string, string, string) {
	return now.Format("2006"), now.Format("01"), now.Format("02")
}

func (api *API) userMedia(c echo.Context) error {
	userID, err := parseID(c, "id", "invalid user id")
	if err != nil {
		return err
	}
	rel, err := utils.CleanObjectKey(path.Join(
		strconv.Itoa(int(userID)),
		c.Param("year"),
		c.Param("month"),
		c.Param("day"),
		c.Param("*"),
	))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "media not found")
	}
	key := path.Join("users", rel)
	if !userUploadKeyAllowed(key) {
		return echo.NewHTTPError(http.StatusNotFound, "media not found")
	}
	return streamMedia(c, key, "media not found")
}

func streamMedia(c echo.Context, key string, notFound string) error {
	if !sameSiteMediaRequest(c) {
		return echo.NewHTTPError(http.StatusForbidden, "media hotlink is not allowed")
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	reader, contentType, err := store.Open(c.Request().Context(), key)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, notFound)
	}
	defer reader.Close()
	c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=31536000, immutable")
	c.Response().Header().Set(echo.HeaderXContentTypeOptions, "nosniff")
	return c.Stream(http.StatusOK, contentType, reader)
}

func userUploadKeyAllowed(key string) bool {
	parts := strings.Split(key, "/")
	return len(parts) >= 5 && parts[0] == "users" && parts[1] != ""
}

func sameSiteMediaRequest(c echo.Context) bool {
	raw := c.Request().Referer()
	if raw == "" {
		return true
	}
	ref, err := url.Parse(raw)
	if err != nil || ref.Hostname() == "" {
		return false
	}
	return strings.EqualFold(ref.Hostname(), requestHostname(c.Request().Host))
}

func requestHostname(host string) string {
	if value, _, err := net.SplitHostPort(host); err == nil {
		return strings.Trim(value, "[]")
	}
	return strings.Trim(host, "[]")
}

func langView(row models.Language) contract.Language {
	return contract.Language{ID: row.ID, Name: row.Name, Source: row.Source}
}

func uploadExt(_ string, mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".img"
	}
}
