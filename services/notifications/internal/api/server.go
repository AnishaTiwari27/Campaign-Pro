// Package api is notifications-service's HTTP surface: one WebSocket
// upgrade endpoint and a health check.
package api

import (
	"net/http"

	"github.com/coder/websocket"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/notifications/internal/hub"
)

type Server struct {
	Hub           *hub.Hub
	AllowedOrigin string
}

func NewRouter(s *Server) http.Handler {
	mux := http.NewServeMux()
	// /healthz gets OTel tracing like every other service's routes.
	// /ws/notifications deliberately does NOT go through platform.Traced:
	// otelhttp's ResponseWriter wrapper doesn't implement http.Hijacker, so
	// wrapping a WebSocket upgrade route with it turns every connection
	// attempt into a 501 Not Implemented. A long-lived WS connection isn't
	// a great fit for span-per-request tracing anyway.
	mux.Handle("GET /healthz", platform.Traced("notifications-service", http.HandlerFunc(s.handleHealth)))
	mux.HandleFunc("GET /ws/notifications", s.handleWebSocket)

	// No platform.CORS here — WebSocket upgrades don't use CORS preflight,
	// they're gated by OriginPatterns below instead.
	return platform.RequestLogger(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	platform.WriteJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"connections": s.Hub.Connections(),
	})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{s.AllowedOrigin, "localhost:*"},
	})
	if err != nil {
		return // Accept already wrote the appropriate HTTP error response
	}
	defer conn.CloseNow()

	s.Hub.Serve(conn) // blocks until the client disconnects
}
