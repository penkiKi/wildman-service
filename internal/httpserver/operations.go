package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	appauth "wildman-service/internal/app/auth"
	"wildman-service/internal/app/operations"
)

func registerOperationsRoutes(router chi.Router, service *operations.Service, authService *appauth.Service) {
	router.Get("/operations/resolutions", func(w http.ResponseWriter, r *http.Request) {
		if _, authenticated := operatorUserID(w, r, authService); !authenticated {
			return
		}
		items, err := service.Resolutions(r.Context())
		if err != nil {
			writeInternalAPIError(w, r, err)
			return
		}
		writeJSON(w, r, http.StatusOK, struct {
			Resolutions []operations.ResolutionSummary `json:"resolutions"`
		}{items})
	})
	router.Get("/operations/provider", func(w http.ResponseWriter, r *http.Request) {
		if _, authenticated := operatorUserID(w, r, authService); !authenticated {
			return
		}
		summary, err := service.Provider(r.Context())
		if err != nil {
			writeInternalAPIError(w, r, err)
			return
		}
		writeJSON(w, r, http.StatusOK, summary)
	})
	router.Post("/operations/retention", func(w http.ResponseWriter, r *http.Request) {
		userID, authenticated := operatorUserID(w, r, authService)
		if !authenticated {
			return
		}
		if !requestCSRFValid(r) {
			writeAPIError(w, r, http.StatusForbidden, "CSRF_TOKEN_INVALID", "CSRF Token 无效或缺失")
			return
		}
		result, err := service.Purge(r.Context(), userID)
		if err != nil {
			writeInternalAPIError(w, r, err)
			return
		}
		writeJSON(w, r, http.StatusOK, result)
	})
}
