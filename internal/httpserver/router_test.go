package httpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	appsystem "wildman-service/internal/app/system"
	"wildman-service/internal/config"
)

func TestHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{Environment: "test"}
	router := NewRouter(cfg, logger, appsystem.NewService(nil, cfg), nil, nil, nil, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body struct {
		Data healthResponse `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Data.Status != "ok" || body.Data.Service != "wildman-service" || body.Data.Environment != "test" {
		t.Fatalf("unexpected health response: %+v", body.Data)
	}
}

func TestSPAIsServed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{Environment: "test"}
	router := NewRouter(cfg, logger, appsystem.NewService(nil, cfg), nil, nil, nil, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType == "" {
		t.Fatal("expected a content type for the SPA response")
	}
}
