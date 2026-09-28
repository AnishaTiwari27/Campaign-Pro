// Command server runs the API gateway — the single entry point the
// frontend talks to. Reverse-proxies to catalog/analytics-service directly,
// and round-robins + health-checks across the campaigns-service replicas
// (see internal/lb) so it's a real, demonstrable load balancer.
//
//	go run ./cmd/server
//
// Env vars: PORT (8080), CATALOG_URLS, ANALYTICS_URLS, AUTH_URLS,
// CAMPAIGNS_URLS, AUDIT_URLS, NOTIFICATIONS_URLS (all comma-separated; NOTIFICATIONS_URLS
// empty/unset = that route isn't registered), ALLOWED_ORIGIN, JWT_SECRET
// (must match auth-service's — see docs/ARCHITECTURE.md), API_RATE_RPS,
// API_RATE_BURST, AUTH_RATE_RPS, AUTH_RATE_BURST (see platform/ratelimit.go —
// AUTH_RATE_* is deliberately stricter by default, /auth/* being
// brute-force-prone).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/gateway/internal/api"
	"campaigntrackerpro/services/gateway/internal/lb"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	shutdownTracer, err := platform.InitTracer("gateway")
	if err != nil {
		slog.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer shutdownTracer(context.Background())

	campaignsPool := lb.NewPool("campaigns-service", urlList("CAMPAIGNS_URLS", "http://localhost:8082,http://localhost:8083"), 5*time.Second)
	catalogPool := lb.NewPool("catalog-service", urlList("CATALOG_URLS", "http://localhost:8081"), 5*time.Second)
	analyticsPool := lb.NewPool("analytics-service", urlList("ANALYTICS_URLS", "http://localhost:8084"), 5*time.Second)
	authPool := lb.NewPool("auth-service", urlList("AUTH_URLS", "http://localhost:8086"), 5*time.Second)
	auditPool := lb.NewPool("audit-service", urlList("AUDIT_URLS", "http://localhost:8090"), 5*time.Second)

	// Unlike the other four, notifications has no default — an empty/unset
	// NOTIFICATIONS_URLS means the WS route simply isn't registered (same
	// behavior as before this was pool-based).
	var notificationsPool *lb.Pool
	if raw := os.Getenv("NOTIFICATIONS_URLS"); raw != "" {
		notificationsPool = lb.NewPool("notifications-service", strings.Split(raw, ","), 5*time.Second)
	}

	secret := envOr("JWT_SECRET", "dev-insecure-secret-change-me")
	if secret == "dev-insecure-secret-change-me" {
		slog.Warn("JWT_SECRET not set — using the insecure local-dev default; do not use this outside a laptop")
	}

	handler := api.NewRouter(api.Config{
		CatalogPool:       catalogPool,
		AnalyticsPool:     analyticsPool,
		AuthPool:          authPool,
		AuditPool:         auditPool,
		CampaignsPool:     campaignsPool,
		NotificationsPool: notificationsPool,
		AllowedOrigin:     envOr("ALLOWED_ORIGIN", "http://localhost:5173"),
		JWTSecret:         []byte(secret),
		APILimit:          platform.NewRateLimit(envFloat("API_RATE_RPS", 5), envInt("API_RATE_BURST", 20)),
		AuthLimit:         platform.NewRateLimit(envFloat("AUTH_RATE_RPS", 1), envInt("AUTH_RATE_BURST", 5)),
	})

	addr := ":" + envOr("PORT", "8080")
	slog.Info("gateway starting", "addr", addr)
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

// urlList reads a comma-separated env var, falling back to fallback
// (itself comma-separated) when unset — the same "comma-separated list of
// backends" convention CAMPAIGNS_URLS already used, now shared by every
// pool.
func urlList(key, fallback string) []string {
	return strings.Split(envOr(key, fallback), ",")
}

func envFloat(key string, fallback float64) float64 {
	v, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil {
		return fallback
	}
	return v
}

func envInt(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}
