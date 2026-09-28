package api

import (
	"net/http"
	"strings"

	"campaigntrackerpro/platform"
)

// trustedHeaders are the identity headers RequireAuth/RequireAuthQuery set
// after verifying a caller's JWT. Every downstream service trusts these at
// face value instead of re-verifying the JWT itself — the gateway is the
// system's one JWT-verification boundary; see docs/ARCHITECTURE.md. They're
// stripped from every inbound request first so a caller can't just set
// X-User-Role: admin and skip the gateway's check.
var trustedHeaders = []string{"X-User-Id", "X-User-Email", "X-User-Role"}

func stripTrustedHeaders(r *http.Request) {
	for _, h := range trustedHeaders {
		r.Header.Del(h)
	}
}

func setTrustedHeaders(r *http.Request, claims *platform.Claims) {
	r.Header.Set("X-User-Id", claims.UserID)
	r.Header.Set("X-User-Email", claims.Email)
	r.Header.Set("X-User-Role", claims.Role)
}

// RequireAuth verifies the Authorization: Bearer <token> header, 401ing on
// anything missing/expired/malformed, and otherwise sets the trusted
// X-User-* headers before calling next. Wraps every /api/v1/* route —
// /auth/* is the only public surface (see NewRouter).
func RequireAuth(secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stripTrustedHeaders(r)

		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			platform.WriteError(w, http.StatusUnauthorized, "missing_token", "missing or malformed Authorization header")
			return
		}
		claims, err := platform.ParseAccessToken(secret, token)
		if err != nil {
			platform.WriteError(w, http.StatusUnauthorized, "invalid_token", "access token is invalid or expired")
			return
		}
		setTrustedHeaders(r, claims)
		next.ServeHTTP(w, r)
	})
}

// RequireAuthQuery is RequireAuth's counterpart for the one route a browser
// can't attach an Authorization header to: the WebSocket upgrade at
// /ws/notifications, which already bypasses the traced mux for an
// unrelated reason (otelhttp's ResponseWriter isn't a Hijacker — see the
// note in NewRouter). The token travels as ?token=<access token> instead.
func RequireAuthQuery(secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stripTrustedHeaders(r)

		token := r.URL.Query().Get("token")
		if token == "" {
			platform.WriteError(w, http.StatusUnauthorized, "missing_token", "missing token query parameter")
			return
		}
		claims, err := platform.ParseAccessToken(secret, token)
		if err != nil {
			platform.WriteError(w, http.StatusUnauthorized, "invalid_token", "access token is invalid or expired")
			return
		}
		setTrustedHeaders(r, claims)
		next.ServeHTTP(w, r)
	})
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
