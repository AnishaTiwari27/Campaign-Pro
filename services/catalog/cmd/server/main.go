// Command server runs catalog-service: the only service allowed to read or
// write the "catalog" Postgres schema (categories, regions, ad types,
// brands, people).
//
//	go run ./cmd/server
//
// Env vars: PORT (default 8081), DATABASE_URL, REDIS_ADDR, ALLOWED_ORIGIN.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/catalog/internal/api"
	"campaigntrackerpro/services/catalog/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	shutdownTracer, err := platform.InitTracer("catalog-service")
	if err != nil {
		slog.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer shutdownTracer(context.Background())

	dbURL := envOr("DATABASE_URL", "postgres://catalog_service:catalog_dev_pw@localhost:5432/campaign_tracker?sslmode=disable")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	var cache *platform.Cache
	if addr := envOr("REDIS_ADDR", "localhost:6379"); addr != "" {
		c := platform.ConnectCache(addr)
		if err := c.Ping(context.Background()); err != nil {
			slog.Warn("redis unreachable, /meta will not be cached", "err", err)
		} else {
			cache = c
		}
	}

	server := &api.Server{Store: store.New(pool), Cache: cache}
	handler := api.NewRouter(server, envOr("ALLOWED_ORIGIN", "http://localhost:8080"))

	addr := ":" + envOr("PORT", "8081")
	slog.Info("catalog-service starting", "addr", addr)
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
