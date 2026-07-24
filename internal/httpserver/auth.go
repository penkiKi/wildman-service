package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	appauth "wildman-service/internal/app/auth"
	"wildman-service/internal/config"
)

const sessionCookieName = "wildman_session"

type setupAdminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Status   string `json:"status"`
}

type setupAdminResponse struct {
	User userResponse `json:"user"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	User      userResponse `json:"user"`
	CSRFToken string       `json:"csrfToken,omitempty"`
}

func registerAuthRoutes(router chi.Router, service *appauth.Service, cfg config.Config) {
	limiter := newLoginLimiter()

	router.Get("/setup/status", func(w http.ResponseWriter, r *http.Request) {
		status, err := service.SetupStatus(r.Context())
		if err != nil {
			writeInternalAPIError(w, r, err)
			return
		}
		writeJSON(w, r, http.StatusOK, status)
	})

	router.Post("/setup/admin", func(w http.ResponseWriter, r *http.Request) {
		var request setupAdminRequest
		if err := decodeJSON(w, r, &request); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
			return
		}

		user, err := service.CreateInitialAdmin(r.Context(), request.Username, request.Password)
		switch {
		case err == nil:
			writeJSON(w, r, http.StatusCreated, setupAdminResponse{User: userResponse{
				ID:       user.ID,
				Username: user.Username,
				Status:   user.Status,
			}})
		case errors.Is(err, appauth.ErrSetupComplete):
			writeAPIError(w, r, http.StatusConflict, "SETUP_ALREADY_COMPLETE", "管理员初始化已经完成")
		case errors.Is(err, appauth.ErrInvalidUsername):
			writeAPIError(w, r, http.StatusUnprocessableEntity, "INVALID_USERNAME", "用户名需要包含 3 至 32 个字母、数字、点、下划线或连字符")
		case errors.Is(err, appauth.ErrInvalidPassword):
			writeAPIError(w, r, http.StatusUnprocessableEntity, "INVALID_PASSWORD", "密码至少需要 6 个字符，且不能超过 1,024 字节")
		default:
			writeInternalAPIError(w, r, err)
		}
	})

	router.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(loginLimitKey(r)) {
			w.Header().Set("Retry-After", "60")
			writeAPIError(w, r, http.StatusTooManyRequests, "LOGIN_RATE_LIMITED", "登录尝试过于频繁，请稍后重试")
			return
		}
		var request loginRequest
		if err := decodeJSON(w, r, &request); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
			return
		}

		result, err := service.Login(r.Context(), request.Username, request.Password)
		switch {
		case err == nil:
			csrfToken := newCSRFToken()
			setSessionCookie(w, cfg, result.Token, result.ExpiresAt)
			setCSRFCookie(w, cfg, csrfToken, result.ExpiresAt)
			writeJSON(w, r, http.StatusOK, loginResponse{
				User:      toUserResponse(result.User.ID, result.User.Username, result.User.Status),
				CSRFToken: csrfToken,
			})
		case errors.Is(err, appauth.ErrInvalidCredentials):
			writeAPIError(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "用户名或密码错误")
		default:
			writeInternalAPIError(w, r, err)
		}
	})

	router.Post("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if !requestCSRFValid(r) {
			writeAPIError(w, r, http.StatusForbidden, "CSRF_TOKEN_INVALID", "CSRF Token 无效或缺失")
			return
		}
		if err := service.Logout(r.Context(), sessionToken(r)); err != nil {
			writeInternalAPIError(w, r, err)
			return
		}
		clearSessionCookie(w, cfg)
		clearCSRFCookie(w, cfg)
		writeJSON(w, r, http.StatusOK, nil)
	})

	router.Get("/auth/me", func(w http.ResponseWriter, r *http.Request) {
		user, err := service.Authenticate(r.Context(), sessionToken(r))
		switch {
		case err == nil:
			writeJSON(w, r, http.StatusOK, loginResponse{User: toUserResponse(user.ID, user.Username, user.Status)})
		case errors.Is(err, appauth.ErrUnauthenticated):
			writeAPIError(w, r, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "请先登录")
		default:
			writeInternalAPIError(w, r, err)
		}
	})
}

func toUserResponse(id, username, status string) userResponse {
	return userResponse{ID: id, Username: username, Status: status}
}

func sessionToken(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setSessionCookie(w http.ResponseWriter, cfg config.Config, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   cfg.Environment == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, cfg config.Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Environment == "production",
		SameSite: http.SameSiteLaxMode,
	})
}
