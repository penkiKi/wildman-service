package httpserver

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
	"sync"
	"time"

	"wildman-service/internal/config"
)

const (
	csrfCookieName   = "wildman_csrf"
	csrfHeaderName   = "X-CSRF-Token"
	loginLimit       = 10
	loginLimitWindow = time.Minute
)

type loginLimitEntry struct {
	attempts int
	resetAt  time.Time
}

type loginLimiter struct {
	mutex       sync.Mutex
	entries     map[string]loginLimitEntry
	lastCleanup time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{
		entries:     make(map[string]loginLimitEntry),
		lastCleanup: time.Now(),
	}
}

func (l *loginLimiter) Allow(key string) bool {
	now := time.Now()
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if now.Sub(l.lastCleanup) >= loginLimitWindow {
		for entryKey, entry := range l.entries {
			if !now.Before(entry.resetAt) {
				delete(l.entries, entryKey)
			}
		}
		l.lastCleanup = now
	}

	entry, exists := l.entries[key]
	if !exists || !now.Before(entry.resetAt) {
		l.entries[key] = loginLimitEntry{attempts: 1, resetAt: now.Add(loginLimitWindow)}
		return true
	}
	if entry.attempts >= loginLimit {
		return false
	}
	entry.attempts++
	l.entries[key] = entry
	return true
}

func loginLimitKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func requestCSRFValid(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	header := r.Header.Get(csrfHeaderName)
	if header == "" || len(header) != len(cookie.Value) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) == 1
}

func newCSRFToken() string {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(value)
}

func setCSRFCookie(w http.ResponseWriter, cfg config.Config, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: false,
		Secure:   cfg.Environment == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCSRFCookie(w http.ResponseWriter, cfg config.Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   cfg.Environment == "production",
		SameSite: http.SameSiteLaxMode,
	})
}
