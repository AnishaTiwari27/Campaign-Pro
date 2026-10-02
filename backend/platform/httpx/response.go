// Package httpx is the shared HTTP layer every service transports over:
// the JSON envelope, the error-to-status mapping, and request middleware.
// It knows nothing about campaigns, creators or reports — services depend
// on it, never the reverse. Named httpx so files here can still import
// net/http without aliasing.
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Sentinel errors every service maps onto, so the HTTP layer can translate
// them to statuses without importing any service.
var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrValidation   = errors.New("validation")
	ErrUnauthorized = errors.New("unauthorized")
	// ErrConflict is for a request that is well-formed and permitted but
	// that the target's current state refuses. Distinct from ErrForbidden,
	// which is about the caller, and ErrValidation, about the request.
	ErrConflict = errors.New("conflict")
)

// WriteUnauthorized is distinct from forbidden on purpose: 401 tells the
// browser to show the login screen, 403 means signed in but not allowed.
func WriteUnauthorized(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusUnauthorized, Error{Error: message})
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

type Error struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

func WriteValidationError(w http.ResponseWriter, message, field string) {
	WriteJSON(w, http.StatusBadRequest, Error{Error: message, Field: field})
}

func WriteForbidden(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusForbidden, Error{Error: message})
}

func WriteNotFound(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusNotFound, Error{Error: message})
}

// WriteConflict carries its message through, because the reason a state
// refused the request is the only useful part of the response.
func WriteConflict(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusConflict, Error{Error: message})
}

// WriteServiceError maps the small set of sentinel errors every service
// method can return onto the right HTTP status; anything else is a 500,
// logged server-side but not leaked to the client.
func WriteServiceError(w http.ResponseWriter, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		WriteNotFound(w, "not found")
	case errors.Is(err, ErrUnauthorized):
		WriteUnauthorized(w, "sign in to continue")
	case errors.Is(err, ErrForbidden):
		WriteForbidden(w, "you don't have permission to do that")
	case errors.Is(err, ErrValidation):
		WriteValidationError(w, "invalid request", "")
	case errors.Is(err, ErrConflict):
		// Generic fallback. A handler that can say which state blocked the
		// request should write the specific reason itself.
		WriteConflict(w, "that isn't possible in this campaign's current state")
	default:
		logger.Error("internal error", "err", err)
		WriteJSON(w, http.StatusInternalServerError, Error{Error: "internal error"})
	}
}
