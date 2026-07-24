package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	appaccount "wildman-service/internal/app/account"
	appauth "wildman-service/internal/app/auth"
	appclient "wildman-service/internal/app/client"
	"wildman-service/internal/app/operations"
	appresolution "wildman-service/internal/app/resolution"
	appsystem "wildman-service/internal/app/system"
	"wildman-service/internal/config"
	webassets "wildman-service/web"
)

type healthResponse struct {
	Status      string `json:"status"`
	Service     string `json:"service"`
	Environment string `json:"environment"`
	Time        string `json:"time"`
}

func NewRouter(
	cfg config.Config,
	logger *slog.Logger,
	systemService *appsystem.Service,
	authService *appauth.Service,
	clientService *appclient.Service,
	resolutionService *appresolution.Service,
	operationsService *operations.Service,
	accountService *appaccount.Service,
) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(accessLog(logger))

	router.Route("/api/v1", func(api chi.Router) {
		api.Use(cors(cfg))
		api.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, r, http.StatusOK, healthResponse{
				Status:      "ok",
				Service:     "wildman-service",
				Environment: cfg.Environment,
				Time:        time.Now().UTC().Format(time.RFC3339),
			})
		})
		api.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
			readiness, ready := systemService.Readiness(r.Context())
			status := http.StatusOK
			if !ready {
				status = http.StatusServiceUnavailable
			}
			writeJSON(w, r, status, readiness)
		})
		api.Get("/system/info", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, r, http.StatusOK, systemService.Info())
		})
		if authService != nil {
			registerAuthRoutes(api, authService, cfg)
		}
		if accountService != nil {
			registerAccountRoutes(api, accountService, authService)
		}
		if authService != nil && clientService != nil {
			registerClientRoutes(api, clientService, authService)
		}
		if clientService != nil && resolutionService != nil {
			clientMiddleware := newClientMiddleware(clientService)
			api.Group(func(clientAPI chi.Router) {
				clientAPI.Use(clientMiddleware.Authenticate)
				registerResolutionRoutes(clientAPI, resolutionService, accountService)
			})
		}
		if authService != nil && operationsService != nil {
			registerOperationsRoutes(api, operationsService, authService)
		}
		api.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, r, http.StatusNotFound, "API_NOT_FOUND", "接口不存在")
		})
		api.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "请求方法不受支持")
		})
	})

	router.Handle("/*", webassets.SPAHandler())
	return router
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, value any) {
	writeAPIEnvelope(w, status, apiEnvelope{
		Data:      value,
		RequestID: middleware.GetReqID(r.Context()),
	})
}

func writeAPIEnvelope(w http.ResponseWriter, status int, response apiEnvelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func accessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(wrapped, r)
			logger.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.Status(),
				"bytes", wrapped.BytesWritten(),
				"duration_ms", time.Since(started).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
