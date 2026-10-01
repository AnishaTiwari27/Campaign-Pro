// Package httpapi is internal/http from the spec's layout: handlers,
// middleware and the router. Named httpapi (not http) only so files here
// can still import net/http without aliasing.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"campaigntrackerpro/internal/service"
	"campaigntrackerpro/internal/store"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

type apiError struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

func writeValidationError(w http.ResponseWriter, message, field string) {
	writeJSON(w, http.StatusBadRequest, apiError{Error: message, Field: field})
}

func writeForbidden(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusForbidden, apiError{Error: message})
}

func writeNotFound(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusNotFound, apiError{Error: message})
}

// writeServiceError maps the small set of sentinel errors every service
// method can return onto the right HTTP status; anything else is a 500,
// logged server-side but not leaked to the client.
func writeServiceError(w http.ResponseWriter, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeNotFound(w, "not found")
	case errors.Is(err, service.ErrForbidden):
		writeForbidden(w, "you don't have permission to do that")
	case errors.Is(err, service.ErrValidation):
		writeValidationError(w, "invalid request", "")
	default:
		logger.Error("internal error", "err", err)
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "internal error"})
	}
}
