package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const SessionCookie = "doj_session"
const CSRFCookie = "doj_csrf"
const CSRFHeader = "X-DOJ-CSRF"

const sessionDays = 30

type Session struct {
	UserID    uint
	ExpiresAt time.Time
}

var (
	cookieConfigMu sync.RWMutex
	cookieConfig   = CookieConfig{SameSite: http.SameSiteLaxMode}
)

type CookieConfig struct {
	Domain   string
	Secure   *bool
	SameSite http.SameSite
}

func NewToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func SessionExpiresAt(now time.Time) time.Time {
	return now.Add(sessionDays * 24 * time.Hour)
}

func ConfigureCookies(config CookieConfig) {
	if config.SameSite == 0 {
		config.SameSite = http.SameSiteLaxMode
	}
	cookieConfigMu.Lock()
	cookieConfig = config
	cookieConfigMu.Unlock()
}

func SetSessionCookie(c echo.Context, token string, expiresAt time.Time) {
	config := currentCookieConfig()
	c.SetCookie(&http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		Domain:   config.Domain,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		SameSite: config.SameSite,
		Secure:   cookieSecure(c, config),
	})
	SetCSRFCookie(c, token, expiresAt)
}

func SetCSRFCookie(c echo.Context, token string, expiresAt time.Time) {
	config := currentCookieConfig()
	c.SetCookie(&http.Cookie{
		Name:     CSRFCookie,
		Value:    TokenHash(token),
		Path:     "/",
		Domain:   config.Domain,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: false,
		SameSite: config.SameSite,
		Secure:   cookieSecure(c, config),
	})
}

func ClearSessionCookie(c echo.Context) {
	config := currentCookieConfig()
	c.SetCookie(&http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		Domain:   config.Domain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: config.SameSite,
		Secure:   cookieSecure(c, config),
	})
	c.SetCookie(&http.Cookie{
		Name:     CSRFCookie,
		Value:    "",
		Path:     "/",
		Domain:   config.Domain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: false,
		SameSite: config.SameSite,
		Secure:   cookieSecure(c, config),
	})
}

func currentCookieConfig() CookieConfig {
	cookieConfigMu.RLock()
	defer cookieConfigMu.RUnlock()
	return cookieConfig
}

func cookieSecure(c echo.Context, config CookieConfig) bool {
	if config.Secure != nil {
		return *config.Secure
	}
	return c.Scheme() == "https"
}

func SessionToken(c echo.Context) (string, bool) {
	cookie, err := c.Cookie(SessionCookie)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

func UserFromCookie(db *gorm.DB, c echo.Context, now time.Time) (models.User, error) {
	var user models.User
	session, err := readSession(c.Request().Context(), c, now)
	if err != nil {
		return user, err
	}
	if session.UserID == 0 {
		return user, gorm.ErrRecordNotFound
	}
	if err := db.First(&user, "id = ?", session.UserID).Error; err != nil {
		return user, err
	}
	return user, nil
}

func CreateUserSession(c echo.Context, userID uint, now time.Time) error {
	token, err := NewToken()
	if err != nil {
		return err
	}
	expiresAt := SessionExpiresAt(now)
	if err := saveSession(c.Request().Context(), token, Session{UserID: userID, ExpiresAt: expiresAt}); err != nil {
		return err
	}
	SetSessionCookie(c, token, expiresAt)
	return nil
}

func DeleteSession(c echo.Context) error {
	token, ok := SessionToken(c)
	if !ok {
		return nil
	}
	deleteSession(c.Request().Context(), token)
	return nil
}

func readSession(ctx context.Context, c echo.Context, now time.Time) (Session, error) {
	token, ok := SessionToken(c)
	if !ok {
		return Session{}, gorm.ErrRecordNotFound
	}
	var session Session
	found, err := CacheGet(ctx, sessionKey(token), &session)
	if err != nil {
		return Session{}, err
	}
	if !found || !session.ExpiresAt.After(now) {
		_ = CacheDelete(ctx, sessionKey(token))
		return Session{}, gorm.ErrRecordNotFound
	}
	return session, nil
}

func saveSession(ctx context.Context, token string, session Session) error {
	return CacheSet(ctx, sessionKey(token), session, time.Until(session.ExpiresAt))
}

func deleteSession(ctx context.Context, token string) {
	_ = CacheDelete(ctx, sessionKey(token))
}

func sessionKey(token string) string {
	return "doj:session:" + TokenHash(token)
}
