// Command server runs auth-service: the only service allowed to read or
// write the "auth" Postgres schema (users, refresh tokens), and the only
// one that signs a JWT — see docs/ARCHITECTURE.md's trust-boundary note.
//
//	go run ./cmd/server
//
// Env vars: PORT (default 8086), DATABASE_URL, JWT_SECRET, ACCESS_TOKEN_TTL,
// REFRESH_TOKEN_TTL (Go duration strings, e.g. "15m", "720h"), ALLOWED_ORIGIN.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/auth/internal/api"
	"campaigntrackerpro/services/auth/internal/models"
	"campaigntrackerpro/services/auth/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	shutdownTracer, err := platform.InitTracer("auth-service")
	if err != nil {
		slog.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer shutdownTracer(context.Background())

	dbURL := envOr("DATABASE_URL", "postgres://auth_service:auth_dev_pw@localhost:5432/campaign_tracker?sslmode=disable")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	secret := envOr("JWT_SECRET", "dev-insecure-secret-change-me")
	if secret == "dev-insecure-secret-change-me" {
		slog.Warn("JWT_SECRET not set — using the insecure local-dev default; do not use this outside a laptop")
	}

	server := &api.Server{
		Store:      store.New(pool),
		JWTSecret:  []byte(secret),
		AccessTTL:  envDuration("ACCESS_TOKEN_TTL", models.DefaultAccessTokenTTL),
		RefreshTTL: envDuration("REFRESH_TOKEN_TTL", models.DefaultRefreshTokenTTL),
	}
	handler := api.NewRouter(server, envOr("ALLOWED_ORIGIN", "http://localhost:8080"))

	addr := ":" + envOr("PORT", "8086")
	slog.Info("auth-service starting", "addr", addr)
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

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid duration env var, using default", "key", key, "value", v, "default", fallback)
		return fallback
	}
	return d
}
