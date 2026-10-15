package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/identity"
	"campaigntrackerpro/services/identity/internal/service"

	"github.com/go-chi/chi/v5"
)

// ssoStateCookie carries the state, nonce and PKCE verifier across the
// round trip to the provider.
//
// A cookie rather than a server-side row because the data is useless to
// anyone else, is needed for about a minute, and a table would need
// expiring. httpOnly so a script cannot read the verifier, SameSite=Lax
// so it survives the provider's top-level redirect back — Strict would
// drop it on exactly that navigation and break every sign-in.
const ssoStateCookie = "ctp_sso"

// ssoStateTTL bounds how long a half-finished sign-in stays valid. Ten
// minutes is long enough to type a password and clear an MFA prompt,
// and short enough that an abandoned attempt is not left usable.
const ssoStateTTL = 10 * time.Minute

type ssoState struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
}

func (h *Handlers) ssoRoutes(r chi.Router) {
	if h.sso == nil {
		// Not configured: the endpoints do not exist rather than
		// existing and refusing. A route that 400s looks like a bug;
		// a route that is absent is a feature that is switched off.
		return
	}
	r.Get("/auth/sso/start", h.SSOStart)
	r.Get("/auth/sso/callback", h.SSOCallback)
}

// SSOStart sends the browser to the identity provider.
func (h *Handlers) SSOStart(w http.ResponseWriter, r *http.Request) {
	req, err := h.sso.Start()
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	blob, err := json.Marshal(ssoState{State: req.State, Nonce: req.Nonce, Verifier: req.Verifier})
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     ssoStateCookie,
		Value:    base64.RawURLEncoding.EncodeToString(blob),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ssoStateTTL.Seconds()),
	})
	http.Redirect(w, r, req.URL, http.StatusFound)
}

// SSOCallback completes the flow and issues a session.
func (h *Handlers) SSOCallback(w http.ResponseWriter, r *http.Request) {
	// The state cookie is single-use: cleared before anything else can
	// fail, so a rejected callback cannot be retried against the same
	// nonce.
	clearSSOCookie(w, h.secure)

	if e := r.URL.Query().Get("error"); e != "" {
		// The provider refused — usually the user cancelled. Back to the
		// sign-in screen rather than a JSON error page they cannot read.
		h.logger.Info("sso: provider returned an error", "error", e)
		http.Redirect(w, r, "/login?sso=denied", http.StatusFound)
		return
	}

	c, err := r.Cookie(ssoStateCookie)
	if err != nil {
		http.Redirect(w, r, "/login?sso=expired", http.StatusFound)
		return
	}
	blob, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		http.Redirect(w, r, "/login?sso=expired", http.StatusFound)
		return
	}
	var st ssoState
	if err := json.Unmarshal(blob, &st); err != nil {
		http.Redirect(w, r, "/login?sso=expired", http.StatusFound)
		return
	}

	// CSRF. Without this an attacker can hand someone a callback URL
	// carrying their own code and sign the victim into the attacker's
	// account.
	if q := r.URL.Query().Get("state"); q == "" || q != st.State {
		h.logger.Error("sso: state mismatch on callback")
		http.Redirect(w, r, "/login?sso=failed", http.StatusFound)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/login?sso=failed", http.StatusFound)
		return
	}

	id, err := h.sso.Exchange(r.Context(), code, st.Verifier, st.Nonce)
	if err != nil {
		h.logger.Error("sso: token exchange or validation failed", "err", err)
		http.Redirect(w, r, "/login?sso=failed", http.StatusFound)
		return
	}

	_, token, err := h.auth.SSOLogin(r.Context(), id, h.ssoOrgID, r.UserAgent(), clientIP(r))
	if err != nil {
		if errors.Is(err, service.ErrSSOUnverifiedEmail) {
			h.logger.Error("sso: provider asserted an unverified email", "email", id.Email)
			http.Redirect(w, r, "/login?sso=unverified", http.StatusFound)
			return
		}
		h.logger.Error("sso: sign-in failed", "err", err)
		http.Redirect(w, r, "/login?sso=failed", http.StatusFound)
		return
	}

	httpx.SetSessionCookie(w, token, h.secure, int(identity.SessionTTL.Seconds()))
	// A redirect rather than JSON: this is a top-level browser
	// navigation the provider initiated, not an API call.
	http.Redirect(w, r, "/overview", http.StatusFound)
}

func clearSSOCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: ssoStateCookie, Value: "", Path: "/",
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}
