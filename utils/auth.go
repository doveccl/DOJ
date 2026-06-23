package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

type DevSession struct {
	UserID    uint
	Role      string
	ExpiresAt time.Time
}

var (
	devSessionMu sync.Mutex
	devSessions  = map[string]DevSession{}

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

func ConfigureCookiesFromEnv() error {
	secure, err := cookieSecureFromEnv(os.Getenv("DOJ_COOKIE_SECURE"))
	if err != nil {
		return err
	}
	sameSite, err := sameSiteFromEnv(os.Getenv("DOJ_COOKIE_SAMESITE"))
	if err != nil {
		return err
	}
	if sameSite == http.SameSiteNoneMode && (secure == nil || !*secure) {
		return fmt.Errorf("DOJ_COOKIE_SAMESITE=none requires DOJ_COOKIE_SECURE=true")
	}
	ConfigureCookies(CookieConfig{
		Domain:   strings.TrimSpace(os.Getenv("DOJ_COOKIE_DOMAIN")),
		Secure:   secure,
		SameSite: sameSite,
	})
	return nil
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
	return c.IsTLS()
}

func cookieSecureFromEnv(value string) (*bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return nil, nil
	case "1", "true", "yes", "on":
		secure := true
		return &secure, nil
	case "0", "false", "no", "off":
		secure := false
		return &secure, nil
	default:
		return nil, fmt.Errorf("invalid DOJ_COOKIE_SECURE %q", value)
	}
}

func sameSiteFromEnv(value string) (http.SameSite, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "lax":
		return http.SameSiteLaxMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	default:
		return 0, fmt.Errorf("invalid DOJ_COOKIE_SAMESITE %q", value)
	}
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
	if err := saveSession(c.Request().Context(), token, DevSession{UserID: userID, ExpiresAt: expiresAt}); err != nil {
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

func CreateDevSession(c echo.Context, role string, now time.Time) error {
	token, err := NewToken()
	if err != nil {
		return err
	}
	expiresAt := SessionExpiresAt(now)
	if err := saveSession(c.Request().Context(), token, DevSession{Role: role, ExpiresAt: expiresAt}); err != nil {
		return err
	}
	SetSessionCookie(c, token, expiresAt)
	return nil
}

func DeleteDevSession(c echo.Context) {
	token, ok := SessionToken(c)
	if ok {
		deleteSession(c.Request().Context(), token)
	}
	ClearSessionCookie(c)
}

func DevRole(c echo.Context, now time.Time) string {
	session, err := readSession(c.Request().Context(), c, now)
	if err == nil && session.Role != "" {
		return session.Role
	}
	return "guest"
}

func readSession(ctx context.Context, c echo.Context, now time.Time) (DevSession, error) {
	token, ok := SessionToken(c)
	if !ok {
		return DevSession{}, gorm.ErrRecordNotFound
	}
	if session, ok := readMemorySession(token, now); ok {
		return session, nil
	}
	session, err := readRedisSession(ctx, token, now)
	if err != nil {
		return DevSession{}, err
	}
	return session, nil
}

func saveSession(ctx context.Context, token string, session DevSession) error {
	devSessionMu.Lock()
	devSessions[token] = session
	devSessionMu.Unlock()

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
	devSessionMu.Lock()
	delete(devSessions, token)
	devSessionMu.Unlock()
	if client := sessionRedisClient(ctx); client != nil {
		_ = client.Del(ctx, sessionKey(token)).Err()
	}
}

func readMemorySession(token string, now time.Time) (DevSession, bool) {
	devSessionMu.Lock()
	defer devSessionMu.Unlock()
	session, ok := devSessions[token]
	if !ok || !session.ExpiresAt.After(now) {
		delete(devSessions, token)
		return DevSession{}, false
	}
	return session, true
}

func readRedisSession(ctx context.Context, token string, now time.Time) (DevSession, error) {
	client := sessionRedisClient(ctx)
	if client == nil {
		return DevSession{}, gorm.ErrRecordNotFound
	}
	raw, err := client.Get(ctx, sessionKey(token)).Bytes()
	if err == redis.Nil {
		return DevSession{}, gorm.ErrRecordNotFound
	}
	if err != nil {
		return DevSession{}, err
	}
	var session DevSession
	if err := json.Unmarshal(raw, &session); err != nil {
		return DevSession{}, err
	}
	if !session.ExpiresAt.After(now) {
		_ = client.Del(ctx, sessionKey(token)).Err()
		return DevSession{}, gorm.ErrRecordNotFound
	}
	devSessionMu.Lock()
	devSessions[token] = session
	devSessionMu.Unlock()
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
	options := &redis.Options{Addr: valkeyAddr()}
	if raw := strings.TrimSpace(os.Getenv("DOJ_VALKEY_URL")); raw != "" {
		parsed, err := redis.ParseURL(raw)
		if err != nil {
			sessionNoRedis = true
			return nil
		}
		options = parsed
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

func valkeyAddr() string {
	if value := strings.TrimSpace(os.Getenv("DOJ_VALKEY_ADDR")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("REDIS_ADDR")); value != "" {
		return value
	}
	return "localhost:6379"
}

func sessionKey(token string) string {
	return "doj:session:" + TokenHash(token)
}
