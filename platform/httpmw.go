package platform

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Traced wraps a handler with automatic OpenTelemetry span creation per
// request (one otelhttp.NewHandler call per service, applied at the mux
// level, rather than every handler remembering to start its own span).
func Traced(serviceName string, h http.Handler) http.Handler {
	return otelhttp.NewHandler(h, serviceName)
}

// RequestLogger logs method, path, status, and latency for every request —
// same shape as the original monolith's middleware.go, shared now instead
// of copy-pasted per service.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("request",
			"method", r.Method, "path", r.URL.Path,
			"status", sw.status, "duration_ms", time.Since(start).Milliseconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Hijack makes statusWriter itself an http.Hijacker when the underlying
// ResponseWriter is one — without this, embedding the http.ResponseWriter
// *interface* only promotes that interface's own methods, not Hijack()
// (which belongs to the separate http.Hijacker interface). Without it, any
// WebSocket upgrade routed through RequestLogger fails with a 501, since
// the WS library can't get at the raw connection to complete the handshake.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
	}
	return hj.Hijack()
}

// CORS is a hand-rolled CORS middleware — the MVP only needs to admit the
// dashboard's origin (and, behind the gateway, the gateway's own origin).
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowedOrigin == "*" || origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
