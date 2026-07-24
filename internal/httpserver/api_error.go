package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

const maximumJSONBodySize = 1 << 20

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type apiEnvelope struct {
	Data      any       `json:"data"`
	Error     *apiError `json:"error"`
	RequestID string    `json:"requestId"`
}

func writeAPIError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeAPIEnvelope(w, status, apiEnvelope{
		Data: nil,
		Error: &apiError{
			Code:    code,
			Message: message,
		},
		RequestID: middleware.GetReqID(r.Context()),
	})
}

func writeInternalAPIError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed",
		"error", err,
		"request_id", middleware.GetReqID(r.Context()),
	)
	writeAPIError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时无法完成请求")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maximumJSONBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain one JSON object")
	}
	return nil
}
