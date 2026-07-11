package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/cache"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const SessionCookie = "doj_session"
const CSRFCookie = "doj_csrf"
const CSRFHeader = "X-DOJ-CSRF"

const sessionDays = 30

type Session struct {
	UserID    uint
	AuthHash  string
	ExpiresAt time.Time
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

func SetSessionCookie(c echo.Context, token string, expiresAt time.Time) {
	c.SetCookie(&http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Scheme() == "https",
	})
	SetCSRFCookie(c, token, expiresAt)
}

func SetCSRFCookie(c echo.Context, token string, expiresAt time.Time) {
	c.SetCookie(&http.Cookie{
		Name:     CSRFCookie,
		Value:    TokenHash(token),
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Scheme() == "https",
	})
}

func ClearSessionCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Scheme() == "https",
	})
	c.SetCookie(&http.Cookie{
		Name:     CSRFCookie,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Scheme() == "https",
	})
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
	if session.AuthHash != TokenHash(user.Auth) {
		_ = DeleteSession(c)
		ClearSessionCookie(c)
		return models.User{}, gorm.ErrRecordNotFound
	}
	return user, nil
}

func CreateUserSession(c echo.Context, user models.User, now time.Time) error {
	token, err := NewToken()
	if err != nil {
		return err
	}
	expiresAt := SessionExpiresAt(now)
	if err := saveSession(c.Request().Context(), token, Session{UserID: user.ID, AuthHash: TokenHash(user.Auth), ExpiresAt: expiresAt}); err != nil {
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
	found, err := cache.Get(ctx, sessionKey(token), &session)
	if err != nil {
		return Session{}, err
	}
	if !found || !session.ExpiresAt.After(now) {
		_ = cache.Delete(ctx, sessionKey(token))
		return Session{}, gorm.ErrRecordNotFound
	}
	return session, nil
}

func saveSession(ctx context.Context, token string, session Session) error {
	return cache.Set(ctx, sessionKey(token), session, time.Until(session.ExpiresAt))
}

func deleteSession(ctx context.Context, token string) {
	_ = cache.Delete(ctx, sessionKey(token))
}

func sessionKey(token string) string {
	return "doj:session:" + TokenHash(token)
}
