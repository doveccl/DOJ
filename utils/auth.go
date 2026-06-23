package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
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
	memorySessionMu sync.Mutex
	memorySessions  = map[string]Session{}

	cookieConfigMu sync.RWMutex
	cookieConfig   = CookieConfig{SameSite: http.SameSiteLaxMode}

	sessionClientMu sync.Mutex
	sessionClient   *redis.Client
	sessionNoRedis  bool
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
	if session, ok := readMemorySession(token, now); ok {
		return session, nil
	}
	session, err := readRedisSession(ctx, token, now)
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func saveSession(ctx context.Context, token string, session Session) error {
	memorySessionMu.Lock()
	memorySessions[token] = session
	memorySessionMu.Unlock()

	client := sessionRedisClient(ctx)
	if client == nil {
		return nil
	}
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	if err := client.Set(ctx, sessionKey(token), raw, time.Until(session.ExpiresAt)).Err(); err != nil {
		return err
	}
	return nil
}

func deleteSession(ctx context.Context, token string) {
	memorySessionMu.Lock()
	delete(memorySessions, token)
	memorySessionMu.Unlock()
	if client := sessionRedisClient(ctx); client != nil {
		_ = client.Del(ctx, sessionKey(token)).Err()
	}
}

func readMemorySession(token string, now time.Time) (Session, bool) {
	memorySessionMu.Lock()
	defer memorySessionMu.Unlock()
	session, ok := memorySessions[token]
	if !ok || !session.ExpiresAt.After(now) {
		delete(memorySessions, token)
		return Session{}, false
	}
	return session, true
}

func readRedisSession(ctx context.Context, token string, now time.Time) (Session, error) {
	client := sessionRedisClient(ctx)
	if client == nil {
		return Session{}, gorm.ErrRecordNotFound
	}
	raw, err := client.Get(ctx, sessionKey(token)).Bytes()
	if err == redis.Nil {
		return Session{}, gorm.ErrRecordNotFound
	}
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := json.Unmarshal(raw, &session); err != nil {
		return Session{}, err
	}
	if !session.ExpiresAt.After(now) {
		_ = client.Del(ctx, sessionKey(token)).Err()
		return Session{}, gorm.ErrRecordNotFound
	}
	memorySessionMu.Lock()
	memorySessions[token] = session
	memorySessionMu.Unlock()
	return session, nil
}

func sessionRedisClient(ctx context.Context) *redis.Client {
	sessionClientMu.Lock()
	defer sessionClientMu.Unlock()
	if sessionNoRedis {
		return nil
	}
	if sessionClient != nil {
		return sessionClient
	}
	raw := strings.TrimSpace(os.Getenv("REDIS"))
	if raw == "" {
		sessionNoRedis = true
		return nil
	}
	options, err := redis.ParseURL(raw)
	if err != nil {
		sessionNoRedis = true
		return nil
	}
	client := redis.NewClient(options)
	pingCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		sessionNoRedis = true
		return nil
	}
	sessionClient = client
	return sessionClient
}

func sessionKey(token string) string {
	return "doj:session:" + TokenHash(token)
}
