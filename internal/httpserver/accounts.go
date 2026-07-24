package httpserver

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	appaccount "wildman-service/internal/app/account"
	appauth "wildman-service/internal/app/auth"
)

type accountCredentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type accountAuthenticationResponse struct {
	AccountID string `json:"accountId"`
	Email     string `json:"email"`
	Token     string `json:"token"`
}
type deviceStartRequest struct {
	ClientName string `json:"clientName"`
}
type deviceApproveRequest struct {
	UserCode string `json:"userCode"`
}
type devicePollRequest struct {
	DeviceCode string `json:"deviceCode"`
}
type subscriptionRequest struct {
	Plan         string `json:"plan"`
	Status       string `json:"status"`
	MonthlyQuota int64  `json:"monthlyQuota"`
}

func registerAccountRoutes(router chi.Router, service *appaccount.Service, authService *appauth.Service) {
	limiter := newLoginLimiter()
	authenticate := func(w http.ResponseWriter, r *http.Request) (string, bool) {
		token, ok := requestBearerToken(r)
		if !ok {
			writeAPIError(w, r, http.StatusUnauthorized, "ACCOUNT_AUTH_REQUIRED", "需要账户登录")
			return "", false
		}
		account, err := service.Authenticate(r.Context(), token)
		if err != nil {
			writeAPIError(w, r, http.StatusUnauthorized, "ACCOUNT_AUTH_REQUIRED", "需要账户登录")
			return "", false
		}
		return account.ID, true
	}
	handleAuthentication := func(w http.ResponseWriter, r *http.Request, register bool) {
		if !limiter.Allow(loginLimitKey(r)) {
			w.Header().Set("Retry-After", "60")
			writeAPIError(w, r, http.StatusTooManyRequests, "ACCOUNT_RATE_LIMITED", "请求过于频繁")
			return
		}
		var input accountCredentialsRequest
		if err := decodeJSON(w, r, &input); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
			return
		}
		var result appaccount.Authentication
		var err error
		if register {
			result, err = service.Register(r.Context(), input.Email, input.Password)
		} else {
			result, err = service.Login(r.Context(), input.Email, input.Password)
		}
		switch {
		case err == nil:
			status := http.StatusOK
			if register {
				status = http.StatusCreated
			}
			writeJSON(w, r, status, accountAuthenticationResponse{AccountID: result.Account.ID, Email: result.Account.Email, Token: result.Token})
		case errors.Is(err, appaccount.ErrInvalidEmail), errors.Is(err, appaccount.ErrInvalidPassword):
			writeAPIError(w, r, http.StatusUnprocessableEntity, "ACCOUNT_INPUT_INVALID", "邮箱或密码不符合限制")
		case errors.Is(err, appaccount.ErrEmailExists):
			writeAPIError(w, r, http.StatusConflict, "ACCOUNT_EMAIL_EXISTS", "邮箱已注册")
		case errors.Is(err, appaccount.ErrInvalidCredentials):
			writeAPIError(w, r, http.StatusUnauthorized, "ACCOUNT_CREDENTIALS_INVALID", "邮箱或密码错误")
		default:
			writeInternalAPIError(w, r, err)
		}
	}
	router.Post("/accounts/register", func(w http.ResponseWriter, r *http.Request) { handleAuthentication(w, r, true) })
	router.Post("/accounts/login", func(w http.ResponseWriter, r *http.Request) { handleAuthentication(w, r, false) })
	router.Post("/device/authorizations", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(loginLimitKey(r)) {
			writeAPIError(w, r, http.StatusTooManyRequests, "ACCOUNT_RATE_LIMITED", "请求过于频繁")
			return
		}
		var input deviceStartRequest
		if err := decodeJSON(w, r, &input); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
			return
		}
		result, err := service.StartDeviceAuthorization(r.Context(), input.ClientName)
		if err != nil {
			writeAPIError(w, r, http.StatusUnprocessableEntity, "DEVICE_AUTH_INVALID", "设备授权请求无效")
			return
		}
		writeJSON(w, r, http.StatusCreated, struct {
			DeviceCode      string `json:"deviceCode"`
			UserCode        string `json:"userCode"`
			VerificationURI string `json:"verificationUri"`
			ExpiresIn       int    `json:"expiresIn"`
			Interval        int    `json:"interval"`
		}{result.DeviceCode, result.UserCode, "/account/device", result.ExpiresIn, result.Interval})
	})
	router.Post("/account/device/approve", func(w http.ResponseWriter, r *http.Request) {
		accountID, ok := authenticate(w, r)
		if !ok {
			return
		}
		var input deviceApproveRequest
		if err := decodeJSON(w, r, &input); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
			return
		}
		if err := service.ApproveDevice(r.Context(), accountID, input.UserCode); err != nil {
			writeAPIError(w, r, http.StatusUnprocessableEntity, "DEVICE_AUTH_INVALID", "设备码无效或已过期")
			return
		}
		writeJSON(w, r, http.StatusOK, nil)
	})
	router.Post("/device/token", func(w http.ResponseWriter, r *http.Request) {
		var input devicePollRequest
		if err := decodeJSON(w, r, &input); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
			return
		}
		token, err := service.PollDevice(r.Context(), input.DeviceCode)
		switch {
		case err == nil:
			writeJSON(w, r, http.StatusOK, struct {
				Token string `json:"token"`
			}{token})
		case errors.Is(err, appaccount.ErrDeviceAuthorizationPending):
			writeAPIError(w, r, http.StatusPreconditionRequired, "DEVICE_AUTH_PENDING", "设备授权尚未完成")
		case errors.Is(err, appaccount.ErrDeviceAuthorizationConsumed):
			writeAPIError(w, r, http.StatusGone, "DEVICE_AUTH_CONSUMED", "设备凭证已经领取")
		default:
			writeAPIError(w, r, http.StatusBadRequest, "DEVICE_AUTH_INVALID", "设备码无效或已过期")
		}
	})
	if authService != nil {
		router.Get("/accounts", func(w http.ResponseWriter, r *http.Request) {
			if _, authenticated := operatorUserID(w, r, authService); !authenticated {
				return
			}
			accounts, err := service.Accounts(r.Context())
			if err != nil {
				writeInternalAPIError(w, r, err)
				return
			}
			writeJSON(w, r, http.StatusOK, struct {
				Accounts []appaccount.AccountSummary `json:"accounts"`
			}{accounts})
		})
		router.Post("/accounts/{accountId}/subscription", func(w http.ResponseWriter, r *http.Request) {
			userID, authenticated := operatorUserID(w, r, authService)
			if !authenticated {
				return
			}
			if !requestCSRFValid(r) {
				writeAPIError(w, r, http.StatusForbidden, "CSRF_TOKEN_INVALID", "CSRF Token 无效或缺失")
				return
			}
			var input subscriptionRequest
			if err := decodeJSON(w, r, &input); err != nil {
				writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
				return
			}
			if err := service.UpdateSubscription(r.Context(), chi.URLParam(r, "accountId"), strings.TrimSpace(input.Plan), strings.TrimSpace(input.Status), input.MonthlyQuota, userID); err != nil {
				writeAPIError(w, r, http.StatusUnprocessableEntity, "SUBSCRIPTION_INVALID", "订阅参数或账户无效")
				return
			}
			writeJSON(w, r, http.StatusOK, nil)
		})
	}
}
