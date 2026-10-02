package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/identity"
	"campaigntrackerpro/services/identity/internal/service"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	auth   *service.Auth
	secure bool
	logger *slog.Logger
}

func New(auth *service.Auth, secureCookies bool, logger *slog.Logger) *Handlers {
	return &Handlers{auth: auth, secure: secureCookies, logger: logger}
}

// PublicRoutes mount outside the session guard — you cannot require a
// session on the endpoint that creates one.
func (h *Handlers) PublicRoutes(r chi.Router) {
	r.Post("/auth/login", h.Login)
}

// Routes mount behind the guard.
func (h *Handlers) Routes(r chi.Router) {
	r.Post("/auth/logout", h.Logout)
	r.Get("/me", h.Me)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	IsAgency   bool   `json:"isAgency"`
	CanApprove bool   `json:"canApprove"`
	IsClient   bool   `json:"isClient"`
}

func toDTO(u identity.User) userDTO {
	return userDTO{
		ID: u.ID, Name: u.Name, Email: u.Email, Role: string(u.Role),
		IsAgency: u.IsAgency, CanApprove: u.CanApprove(), IsClient: u.IsClient(),
	}
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}
	if req.Email == "" || req.Password == "" {
		httpx.WriteValidationError(w, "email and password are required", "")
		return
	}

	user, token, err := h.auth.Login(r.Context(), req.Email, req.Password, r.UserAgent(), clientIP(r))
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			// One message for both causes, so the response can't be used
			// to find out which emails exist.
			httpx.WriteUnauthorized(w, "invalid email or password")
			return
		}
		httpx.WriteServiceError(w, h.logger, err)
		return
	}

	httpx.SetSessionCookie(w, token, h.secure, int(identity.SessionTTL.Seconds()))
	httpx.WriteJSON(w, http.StatusOK, toDTO(user))
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	token := httpx.TokenFromContext(r.Context())
	user, _ := currentUser(r)

	// Revoke server-side first: clearing the cookie alone would leave a
	// working token with anyone who had copied it.
	if err := h.auth.Logout(r.Context(), token, user); err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.ClearSessionCookie(w, h.secure)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		httpx.WriteUnauthorized(w, "sign in to continue")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toDTO(user))
}

func currentUser(r *http.Request) (identity.User, bool) {
	v, _ := httpx.UserFromContext(r.Context())
	u, ok := v.(identity.User)
	return u, ok
}

// clientIP prefers the proxy header, since in production the app sits
// behind a load balancer and RemoteAddr would be the proxy every time.
func clientIP(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		return f
	}
	return r.RemoteAddr
}
