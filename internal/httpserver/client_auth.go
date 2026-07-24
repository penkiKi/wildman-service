package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	appclient "wildman-service/internal/app/client"
)

type clientIDContextKey struct{}

type clientMiddleware struct {
	service *appclient.Service
	limiter *clientRateLimiter
}

func newClientMiddleware(service *appclient.Service) *clientMiddleware {
	return &clientMiddleware{
		service: service,
		limiter: newClientRateLimiter(),
	}
}

func (middleware *clientMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := requestBearerToken(r)
		if !ok {
			writeClientAuthenticationError(w, r, "CLIENT_AUTH_REQUIRED", "缺少或无效客户端 Token")
			return
		}

		installation, err := middleware.service.Authenticate(r.Context(), token)
		switch {
		case err == nil:
		case errors.Is(err, appclient.ErrAuthenticationRequired):
			writeClientAuthenticationError(w, r, "CLIENT_AUTH_REQUIRED", "缺少或无效客户端 Token")
			return
		case errors.Is(err, appclient.ErrInstallationRevoked):
			writeClientAuthenticationError(w, r, "CLIENT_REVOKED", "客户端凭证已撤销")
			return
		default:
			writeInternalAPIError(w, r, err)
			return
		}

		allowed, retryAfter := middleware.limiter.Allow(installation.ID)
		if !allowed {
			retryAfterSeconds := max(1, int((retryAfter+time.Second-1)/time.Second))
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
			writeAPIError(w, r, http.StatusTooManyRequests, "CLIENT_RATE_LIMITED", "客户端请求过于频繁，请稍后重试")
			return
		}

		ctx := context.WithValue(r.Context(), clientIDContextKey{}, installation.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authenticatedClientID(ctx context.Context) (string, bool) {
	clientID, ok := ctx.Value(clientIDContextKey{}).(string)
	return clientID, ok && clientID != ""
}

func requestBearerToken(r *http.Request) (string, bool) {
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}
	scheme, token, found := strings.Cut(values[0], " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", false
	}
	return token, true
}

func writeClientAuthenticationError(w http.ResponseWriter, r *http.Request, code, message string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeAPIError(w, r, http.StatusUnauthorized, code, message)
}
