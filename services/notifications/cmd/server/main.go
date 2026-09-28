// Command server runs notifications-service — subscribes to NATS'
// campaign.created subject and fans each event out to every connected
// WebSocket client in real time.
//
//	go run ./cmd/server
//
// Env vars: PORT (8085), NATS_URL, ALLOWED_ORIGIN.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/notifications/internal/api"
	"campaigntrackerpro/services/notifications/internal/hub"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	shutdownTracer, err := platform.InitTracer("notifications-service")
	if err != nil {
		slog.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer shutdownTracer(context.Background())

	h := hub.New()

	bus, err := platform.ConnectEventBus(envOr("NATS_URL", "nats://127.0.0.1:4222"))
	if err != nil {
		slog.Error("NATS connect failed — notifications would be a no-op without it", "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	if err := bus.Subscribe("campaign.created", func(payload []byte) {
		h.Broadcast(payload)
	}); err != nil {
		slog.Error("NATS subscribe failed", "err", err)
		os.Exit(1)
	}

	server := &api.Server{Hub: h, AllowedOrigin: envOr("ALLOWED_ORIGIN", "http://localhost:8080")}
	handler := api.NewRouter(server)

	addr := ":" + envOr("PORT", "8085")
	slog.Info("notifications-service starting", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
