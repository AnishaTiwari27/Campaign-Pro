// Package module is the identity service's factory. It owns users and
// sessions; no other service can read either.
package module

import (
	"context"
	"log/slog"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/identity"
	"campaigntrackerpro/services/identity/internal/api"
	"campaigntrackerpro/services/identity/internal/service"
	"campaigntrackerpro/services/identity/internal/store"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	auth    *service.Auth
	handler *api.Handlers
	store   *store.Store
}

// Options are this service's deployment switches. A struct, because two
// adjacent bool parameters are transposable without the compiler noticing.
type Options struct {
	// SecureCookies must be true anywhere served over HTTPS.
	SecureCookies bool
	// AllowSignup decides whether self-registration exists.
	AllowSignup bool
}

// New wires the service. directory is how signup writes an account:
// campaigns owns the users table, so it supplies that capability and may
// be nil wherever signup is not needed.
func New(db *database.DB, auditor service.Auditor, directory service.Directory,
	opts Options, logger *slog.Logger) *Module {
	st := store.New(db)
	auth := service.New(st, auditor, directory)
	handler := api.New(auth, api.Options{
		SecureCookies: opts.SecureCookies,
		AllowSignup:   opts.AllowSignup,
	}, logger)
	return &Module{auth: auth, handler: handler, store: st}
}

// PublicRoutes mount outside the session guard: login cannot require one.
func (m *Module) PublicRoutes(r chi.Router) { m.handler.PublicRoutes(r) }

// Routes mount behind the guard.
func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }

// Authenticator is what platform's RequireSession middleware needs.
func (m *Module) Authenticator() httpx.SessionAuthenticator { return sessionAdapter{m.auth} }

// sessionAdapter widens the return to httpx.Principal so platform stays
// free of any identity type.
type sessionAdapter struct{ auth *service.Auth }

func (s sessionAdapter) UserForSession(ctx context.Context, token string) (httpx.Principal, error) {
	return s.auth.UserForSession(ctx, token)
}

// SetPassword is used by the seeder to give fixture users a login.
func (m *Module) SetPassword(ctx context.Context, userID, plain string) error {
	return m.auth.SetPassword(ctx, userID, plain)
}

// PurgeExpiredSessions is run periodically by the worker.
func (m *Module) PurgeExpiredSessions(ctx context.Context) error {
	return m.auth.PurgeExpiredSessions(ctx)
}

var _ identity.Authenticator = (*service.Auth)(nil)
