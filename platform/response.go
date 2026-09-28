// Package platform is the shared horizontal layer every service imports —
// response envelopes, observability setup, an inter-service HTTP client,
// event-bus and cache helpers. Cross-cutting code lives here exactly once
// instead of being copy-pasted into each service.
package platform

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// APIError is the JSON body every service returns on a non-2xx response —
// carried over verbatim from the original monolith's response.go so every
// service (and the frontend) sees one consistent error shape.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{Error: APIError{Code: code, Message: message}})
}

func BadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, "invalid_request", message)
}

func NotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, "not_found", message)
}

func Internal(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, "internal_error", message)
}

func Upstream(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadGateway, "upstream_error", message)
}
