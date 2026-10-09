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

// Options are the deployment-level switches this service's HTTP surface
// needs. A struct rather than two bool parameters, which at a call site
// are trivially transposed and compile either way.
type Options struct {
	// SecureCookies must be true anywhere served over HTTPS.
	SecureCookies bool
	// AllowSignup decides whether self-registration exists at all.
	AllowSignup bool
}

type Handlers struct {
	auth   *service.Auth
	secure bool
	signup bool
	logger *slog.Logger
}

func New(auth *service.Auth, opts Options, logger *slog.Logger) *Handlers {
	return &Handlers{auth: auth, secure: opts.SecureCookies, signup: opts.AllowSignup, logger: logger}
}

// PublicRoutes mount outside the session guard — you cannot require a
// session on the endpoint that creates one.
func (h *Handlers) PublicRoutes(r chi.Router) {
	r.Post("/auth/login", h.Login)
	// Always present, so the sign-in screen can ask whether to offer a
	// "create an account" link rather than guessing and showing one that
	// leads nowhere.
	r.Get("/auth/options", h.AuthOptions)
	// Registered only when enabled: a disabled signup should not be an
	// endpoint that exists and refuses, it should not be an endpoint.
	if h.signup {
		r.Post("/auth/signup", h.Signup)
	}
}

type authOptionsDTO struct {
	SignupEnabled bool `json:"signupEnabled"`
	// Echoed so the form can enforce the same rule the server will, and
	// the two cannot drift.
	MinPasswordLength int `json:"minPasswordLength"`
}

// AuthOptions describes what the sign-in screen may offer. Public by
// necessity: it is read before anyone has a session.
func (h *Handlers) AuthOptions(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, authOptionsDTO{
		SignupEnabled:     h.signup,
		MinPasswordLength: service.MinPasswordLength,
	})
}

type signupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Signup registers an account and returns it without a session. The new
// user signs in next, which both proves the password works and keeps this
// endpoint from being a way to mint sessions.
func (h *Handlers) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}

	user, err := h.auth.Signup(r.Context(), service.SignupInput{
		Name: req.Name, Email: req.Email, Password: req.Password,
	})
	if err != nil {
		if msg, field, ok := service.AsValidationError(err); ok {
			httpx.WriteValidationError(w, msg, field)
			return
		}
		if errors.Is(err, service.ErrEmailTaken) {
			httpx.WriteConflict(w, err.Error())
			return
		}
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toDTO(user))
}

// Routes mount behind the guard.
func (h *Handlers) Routes(r chi.Router) {
	r.Post("/auth/logout", h.Logout)
	r.Get("/me", h.Me)
	h.UserRoutes(r)
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
	// CanManageUsers is derived, not stored. Sent so the client does not
	// re-derive a permission rule from role and agency and get it subtly
	// different from the server that enforces it.
	CanManageUsers bool `json:"canManageUsers"`
}

func toDTO(u identity.User) userDTO {
	return userDTO{
		ID: u.ID, Name: u.Name, Email: u.Email, Role: string(u.Role),
		IsAgency: u.IsAgency, CanApprove: u.CanApprove(), IsClient: u.IsClient(),
		CanManageUsers: u.CanManageUsers(),
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
