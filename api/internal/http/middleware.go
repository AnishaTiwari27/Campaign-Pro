package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"campaigntrackerpro/internal/store"

	"github.com/go-chi/chi/v5/middleware"
)

type ctxKey int

const userCtxKey ctxKey = iota

// AuthMiddleware loads the seeded admin user onto every request's context.
// It's a stub standing in for real SSO/session auth later; every handler
// reads the caller via UserFromContext, so swapping this out later won't
// touch handler code.
func AuthMiddleware(s *store.Store, adminEmail string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, err := s.GetUserByEmail(r.Context(), adminEmail)
			if err != nil {
				logger.Error("auth: seeded admin user not found", "email", adminEmail, "err", err)
				writeJSON(w, http.StatusInternalServerError, apiError{Error: "server not configured"})
				return
			}
			ctx := context.WithValue(r.Context(), userCtxKey, u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) (store.User, bool) {
	u, ok := ctx.Value(userCtxKey).(store.User)
	return u, ok
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("request",
				"method", r.Method, "path", r.URL.Path, "status", ww.Status(),
				"duration", time.Since(start).String(), "requestId", middleware.GetReqID(r.Context()))
		})
	}
}
