package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type ctxKey int

const userCtxKey ctxKey = iota

// AuthMiddleware loads the seeded admin user onto every request's context.
// It's a stub standing in for real SSO/session auth later; every handler
// reads the caller via UserFromContext, so swapping this out later won't
// touch handler code.
// UserLoader resolves the acting account for a request. Defined as a
// function so platform never depends on whichever service owns users.
type UserLoader func(ctx context.Context, email string) (any, error)

func AuthMiddleware(load UserLoader, adminEmail string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, err := load(r.Context(), adminEmail)
			if err != nil {
				logger.Error("auth: seeded admin user not found", "email", adminEmail, "err", err)
				WriteJSON(w, http.StatusInternalServerError, Error{Error: "server not configured"})
				return
			}
			ctx := context.WithValue(r.Context(), userCtxKey, u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext returns whatever the loader put there; callers assert
// their own user type.
func UserFromContext(ctx context.Context) (any, bool) {
	u := ctx.Value(userCtxKey)
	return u, u != nil
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
