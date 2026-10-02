package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type ctxKey int

const (
	userCtxKey ctxKey = iota
	tokenCtxKey
)

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
