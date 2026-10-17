package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

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
	// SSO is the discovered identity provider, or nil when none is
	// configured. Nil means the endpoints are not mounted at all.
	SSO identity.SSOProvider
	// SSOOrgID is the organisation an SSO user joins.
	SSOOrgID string
	// SignupOrgID is the organisation a self-registered account joins.
	// Signup runs outside the tenant middleware, so there is no session
	// to read one from and it has to be supplied.
	SignupOrgID string
	// SSOLabel names the provider on the button — "Google", "Microsoft",
	// or the generic wording for one this app does not recognise. The
	// composition root derives it from the issuer it discovered, so the
	// label cannot name a provider other than the one the button
	// actually reaches.
	SSOLabel string
}

type Handlers struct {
	auth     *service.Auth
	secure   bool
	signup   bool
	sso      identity.SSOProvider
	ssoOrgID string
	// signupOrgID is the organisation a self-registered account joins.
	signupOrgID string
	ssoLabel    string
	logger      *slog.Logger
}

func New(auth *service.Auth, opts Options, logger *slog.Logger) *Handlers {
	return &Handlers{
		auth: auth, secure: opts.SecureCookies, signup: opts.AllowSignup,
		sso: opts.SSO, ssoOrgID: opts.SSOOrgID, signupOrgID: opts.SignupOrgID,
		ssoLabel: opts.SSOLabel, logger: logger,
	}
}

// PublicRoutes mount outside the session guard — you cannot require a
// session on the endpoint that creates one.
func (h *Handlers) PublicRoutes(r chi.Router) {
	r.Post("/auth/login", h.Login)
	h.ssoRoutes(r)
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
	// SSOEnabled tells the sign-in screen whether to offer the button.
	// Read from whether a provider was discovered at boot, not from
	// config, so a provider that failed discovery does not get a button
	// that leads nowhere.
	SSOEnabled bool `json:"ssoEnabled"`
	// SSOLabel is the provider's name, for the button. Sent rather than
	// hardcoded in the client so one deployment can say "Google" and
	// another "Okta" without a rebuild.
	SSOLabel string `json:"ssoLabel"`
	// Echoed so the form can enforce the same rule the server will, and
	// the two cannot drift.
	MinPasswordLength int `json:"minPasswordLength"`
}

// AuthOptions describes what the sign-in screen may offer. Public by
// necessity: it is read before anyone has a session.
func (h *Handlers) AuthOptions(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, authOptionsDTO{
		SignupEnabled:     h.signup,
		SSOEnabled:        h.sso != nil,
		SSOLabel:          h.providerLabel(),
		MinPasswordLength: service.MinPasswordLength,
	})
}

// providerLabel falls back to the generic wording rather than an empty
// string, so a deployment that configured SSO without a recognised
// issuer still gets a button that reads as a sentence.
func (h *Handlers) providerLabel() string {
	if h.ssoLabel == "" {
		return "single sign-on"
	}
	return h.ssoLabel
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
	}, h.signupOrgID)
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
	r.Get("/me/sign-ins", h.SignIns)
	h.UserRoutes(r)
}

type signInDTO struct {
	At        string `json:"at"`
	UserAgent string `json:"userAgent"`
	IP        string `json:"ip"`
	Current   bool   `json:"current"`
}

// SignIns is the caller's own sign-in history.
//
// Scoped to the caller by construction, not by a filter someone has to
// remember: the user id comes from the resolved session, so there is no
// parameter that could be changed to read another account's.
func (h *Handlers) SignIns(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		httpx.WriteUnauthorized(w, "sign in to continue")
		return
	}
	list, err := h.auth.SignIns(r.Context(), user.ID, httpx.TokenFromContext(r.Context()))
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	out := make([]signInDTO, 0, len(list))
	for _, s := range list {
		out = append(out, signInDTO{
			At:        s.At.Format(time.RFC3339),
			UserAgent: s.UserAgent,
			IP:        s.IP,
			Current:   s.Current,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
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
