package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	appauth "wildman-service/internal/app/auth"
	appclient "wildman-service/internal/app/client"
	domainclient "wildman-service/internal/domain/client"
)

type createClientRequest struct {
	Name string `json:"name"`
}

type clientResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	TokenPrefix string  `json:"tokenPrefix"`
	Status      string  `json:"status"`
	LastSeenAt  *string `json:"lastSeenAt"`
	RevokedAt   *string `json:"revokedAt"`
	CreatedAt   string  `json:"createdAt"`
}

type createClientResponse struct {
	Client clientResponse `json:"client"`
	Token  string         `json:"token"`
}

type listClientsResponse struct {
	Clients []clientResponse `json:"clients"`
}

type revokeClientResponse struct {
	Client clientResponse `json:"client"`
}

type deleteClientRequest struct {
	Name string `json:"name"`
}

func registerClientRoutes(router chi.Router, service *appclient.Service, authService *appauth.Service) {
	router.Post("/clients", func(w http.ResponseWriter, r *http.Request) {
		userID, authenticated := operatorUserID(w, r, authService)
		if !authenticated {
			return
		}
		if !requestCSRFValid(r) {
			writeAPIError(w, r, http.StatusForbidden, "CSRF_TOKEN_INVALID", "CSRF Token 无效或缺失")
			return
		}

		var request createClientRequest
		if err := decodeJSON(w, r, &request); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
			return
		}
		result, err := service.Create(r.Context(), request.Name, userID)
		switch {
		case err == nil:
			writeJSON(w, r, http.StatusCreated, createClientResponse{
				Client: toClientResponse(result.Installation),
				Token:  result.Token,
			})
		case errors.Is(err, appclient.ErrInvalidInstallationName):
			writeAPIError(w, r, http.StatusUnprocessableEntity, "CLIENT_NAME_INVALID", "客户端名称需要包含 1 至 100 个字符")
		default:
			writeInternalAPIError(w, r, err)
		}
	})

	router.Get("/clients", func(w http.ResponseWriter, r *http.Request) {
		if _, authenticated := operatorUserID(w, r, authService); !authenticated {
			return
		}
		installations, err := service.List(r.Context())
		if err != nil {
			writeInternalAPIError(w, r, err)
			return
		}
		clients := make([]clientResponse, 0, len(installations))
		for _, installation := range installations {
			clients = append(clients, toClientResponse(installation))
		}
		writeJSON(w, r, http.StatusOK, listClientsResponse{Clients: clients})
	})

	router.Post("/clients/{clientId}/revoke", func(w http.ResponseWriter, r *http.Request) {
		userID, authenticated := operatorUserID(w, r, authService)
		if !authenticated {
			return
		}
		if !requestCSRFValid(r) {
			writeAPIError(w, r, http.StatusForbidden, "CSRF_TOKEN_INVALID", "CSRF Token 无效或缺失")
			return
		}

		installation, err := service.Revoke(r.Context(), chi.URLParam(r, "clientId"), userID)
		switch {
		case err == nil:
			writeJSON(w, r, http.StatusOK, revokeClientResponse{Client: toClientResponse(installation)})
		case errors.Is(err, appclient.ErrInstallationNotFound):
			writeAPIError(w, r, http.StatusNotFound, "CLIENT_NOT_FOUND", "客户端不存在")
		default:
			writeInternalAPIError(w, r, err)
		}
	})

	router.Post("/clients/{clientId}/delete", func(w http.ResponseWriter, r *http.Request) {
		userID, authenticated := operatorUserID(w, r, authService)
		if !authenticated {
			return
		}
		if !requestCSRFValid(r) {
			writeAPIError(w, r, http.StatusForbidden, "CSRF_TOKEN_INVALID", "CSRF Token 无效或缺失")
			return
		}
		var request deleteClientRequest
		if err := decodeJSON(w, r, &request); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
			return
		}
		err := service.Delete(r.Context(), chi.URLParam(r, "clientId"), request.Name, userID)
		switch {
		case err == nil:
			writeJSON(w, r, http.StatusOK, nil)
		case errors.Is(err, appclient.ErrInstallationNotFound):
			writeAPIError(w, r, http.StatusNotFound, "CLIENT_NOT_FOUND", "客户端不存在")
		case errors.Is(err, appclient.ErrClientDeletionNotAllowed):
			writeAPIError(w, r, http.StatusConflict, "CLIENT_DELETE_NOT_ALLOWED", "客户端必须已撤销且确认名称完全一致")
		default:
			writeInternalAPIError(w, r, err)
		}
	})
}

func operatorUserID(w http.ResponseWriter, r *http.Request, service *appauth.Service) (string, bool) {
	user, err := service.Authenticate(r.Context(), sessionToken(r))
	switch {
	case err == nil:
		return user.ID, true
	case errors.Is(err, appauth.ErrUnauthenticated):
		writeAPIError(w, r, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "请先登录")
	default:
		writeInternalAPIError(w, r, err)
	}
	return "", false
}

func toClientResponse(installation domainclient.ClientInstallation) clientResponse {
	return clientResponse{
		ID:          installation.ID,
		Name:        installation.Name,
		TokenPrefix: installation.TokenPrefix,
		Status:      string(installation.Status),
		LastSeenAt:  formatOptionalTime(installation.LastSeenAt),
		RevokedAt:   formatOptionalTime(installation.RevokedAt),
		CreatedAt:   installation.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}
