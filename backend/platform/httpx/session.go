package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

// SessionCookie is the cookie the browser holds. httpOnly so script can't
// read it, SameSite=Lax so it survives top-level navigation but not
// cross-site form posts.
const SessionCookie = "ctp_session"

// Principal is whatever the authenticator resolved. platform keeps this
// opaque so it never depends on the service that owns users.
type Principal any

// SessionAuthenticator turns an opaque token into a principal.
type SessionAuthenticator interface {
	UserForSession(ctx context.Context, token string) (Principal, error)
}

// SetSessionCookie issues the cookie after a successful login. Secure is
// driven by config rather than hardcoded, because local development is
// http and production must not be.
func SetSessionCookie(w http.ResponseWriter, token string, secure bool, maxAgeSeconds int) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAgeSeconds,
	})
}

// ClearSessionCookie expires the cookie. The server-side session must be
// deleted too — this only removes the browser's copy.
func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// TokenFromRequest reads the session cookie, falling back to a bearer
// header so API clients and tests don't need a cookie jar.
func TokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// RequireSession rejects anything without a valid session. This replaces
// the stub that loaded the seeded admin onto every request — the reason
// anyone reaching the API used to be an approver.
func RequireSession(auth SessionAuthenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := TokenFromRequest(r)
			if token == "" {
				WriteUnauthorized(w, "sign in to continue")
				return
			}
			user, err := auth.UserForSession(r.Context(), token)
			if err != nil {
				WriteUnauthorized(w, "your session has expired")
				return
			}
			ctx := context.WithValue(r.Context(), userCtxKey, user)
			ctx = context.WithValue(ctx, tokenCtxKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TokenFromContext returns the session token of the current request, which
// logout needs in order to revoke the right session.
func TokenFromContext(ctx context.Context) string {
	t, _ := ctx.Value(tokenCtxKey).(string)
	return t
}
