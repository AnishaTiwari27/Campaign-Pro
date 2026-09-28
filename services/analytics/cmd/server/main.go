// Command server runs analytics-service — stateless; every response is
// computed from campaigns-service's data and cached briefly in Redis.
//
//	go run ./cmd/server
//
// Env vars: PORT (8084), CAMPAIGNS_URL, REDIS_ADDR, NATS_URL, ALLOWED_ORIGIN.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/analytics/internal/api"
	"campaigntrackerpro/services/analytics/internal/campaignsclient"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	shutdownTracer, err := platform.InitTracer("analytics-service")
	if err != nil {
		slog.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer shutdownTracer(context.Background())

	var cache *platform.Cache
	if addr := envOr("REDIS_ADDR", "localhost:6379"); addr != "" {
		c := platform.ConnectCache(addr)
		if err := c.Ping(context.Background()); err != nil {
			slog.Warn("redis unreachable, aggregates will not be cached", "err", err)
		} else {
			cache = c
		}
	}

	server := &api.Server{
		Campaigns: campaignsclient.New(envOr("CAMPAIGNS_URL", "http://localhost:8082")),
		Cache:     cache,
	}

	// Any new campaign can change any filtered aggregate view, so on
	// campaign.created we flush every cached analytics response rather than
	// try to work out which specific cache keys it could have affected.
	if cache != nil {
		if bus, err := platform.ConnectEventBus(envOr("NATS_URL", "nats://127.0.0.1:4222")); err != nil {
			slog.Warn("NATS unreachable, cache will only expire via TTL", "err", err)
		} else {
			defer bus.Close()
			if err := bus.Subscribe("campaign.created", func(payload []byte) {
				cache.FlushPrefix(context.Background(), "analytics:")
				slog.Info("cache flushed on campaign.created")
			}); err != nil {
				slog.Warn("NATS subscribe failed", "err", err)
			}
		}
	}

	handler := api.NewRouter(server, envOr("ALLOWED_ORIGIN", "http://localhost:8080"))

	addr := ":" + envOr("PORT", "8084")
	slog.Info("analytics-service starting", "addr", addr)
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
