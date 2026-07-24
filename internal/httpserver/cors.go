package httpserver

import (
	"net/http"
	"net/url"
	"strings"

	"wildman-service/internal/config"
)

const (
	allowedMethods = "GET, POST, PATCH, DELETE, OPTIONS"
	allowedHeaders = "Accept, Authorization, Content-Type, Idempotency-Key, X-CSRF-Token"
)

func cors(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			if !originAllowed(origin, r.Host, cfg.AllowedOrigins) {
				writeAPIError(w, r, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "请求来源不受允许")
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			w.Header().Add("Vary", "Origin")

			if r.Method == http.MethodOptions {
				writeJSON(w, r, http.StatusOK, nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func originAllowed(origin, requestHost string, allowedOrigins []string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	if strings.EqualFold(parsed.Host, requestHost) {
		return true
	}

	normalizedOrigin := parsed.Scheme + "://" + parsed.Host
	for _, allowedOrigin := range allowedOrigins {
		if strings.EqualFold(normalizedOrigin, allowedOrigin) {
			return true
		}
	}
	return false
}
