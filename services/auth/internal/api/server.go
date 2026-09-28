// Package api is auth-service's HTTP surface: /auth/login, /auth/refresh,
// /auth/logout, and /healthz. This is the only service in the system that
// ever sees a password or a raw refresh token.
package api

import (
	"context"
	"net/http"
	"time"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/auth/internal/models"
)

// UserStore is exactly what the handlers below need from persistence — not
// the full *store.Store surface. Sized this way (Interface Segregation) so
// tests can substitute a fake without a real Postgres; see
// server_test.go's fakeStore.
type UserStore interface {
	GetUserByEmail(ctx context.Context, email string) (models.User, bool, error)
	GetUserByID(ctx context.Context, id string) (models.User, bool, error)
	CreateUser(ctx context.Context, email, passwordHash string) (models.User, error)
	SaveRefreshToken(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error
	ConsumeRefreshToken(ctx context.Context, tokenHash string) (userID string, ok bool, err error)
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
	Ping(ctx context.Context) error

	// User management — admin-only, backing GET/PATCH /api/v1/users(/{id}/role).
	ListUsers(ctx context.Context) ([]models.User, error)
	UpdateUserRole(ctx context.Context, id, role string) (models.User, bool, error)
}

type Server struct {
	Store      UserStore
	JWTSecret  []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func NewRouter(s *Server, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /auth/register", s.handleRegister)
	mux.HandleFunc("POST /auth/login", s.handleLogin)
	mux.HandleFunc("POST /auth/refresh", s.handleRefresh)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)

	// Under /api/v1/, not /auth/, deliberately — these need an
	// already-verified admin session (the gateway wraps them in
	// protected(), same as every other /api/v1/* route), unlike every
	// other /auth/* route here which is pre-identity/public.
	mux.HandleFunc("GET /api/v1/users", s.handleListUsers)
	mux.HandleFunc("PATCH /api/v1/users/{id}/role", s.handleUpdateUserRole)

	handler := platform.RequestLogger(platform.CORS(allowedOrigin)(mux))
	return platform.Traced("auth-service", handler)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.Ping(r.Context()); err != nil {
		platform.Upstream(w, "database unreachable")
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
