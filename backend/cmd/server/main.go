// Command server runs the Campaign Tracker Pro API.
//
//	go run ./cmd/server
//
// Env vars:
//
//	PORT             default "8080"
//	ALLOWED_ORIGIN   default "http://localhost:5173" (Vite dev server)
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"campaigntrackerpro/internal/api"
	"campaigntrackerpro/internal/data"
	"campaigntrackerpro/internal/store"
)

// seedAnchor matches the frontend mock's `new Date("2026-08-31")" — see the
// comment on api.Server.Today for why this can't just be time.Now() yet.
var seedAnchor = time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := envOr("PORT", "8080")
	allowedOrigin := envOr("ALLOWED_ORIGIN", "http://localhost:5173")

	campaigns := data.GenerateCampaigns(seedAnchor)
	memStore := store.NewMemoryStore(campaigns)

	server := &api.Server{Store: memStore, Today: seedAnchor}
	handler := api.NewRouter(server, allowedOrigin)

	addr := ":" + port
	slog.Info("campaign-tracker-pro api starting",
		"addr", addr, "allowed_origin", allowedOrigin, "campaigns_seeded", len(campaigns))

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
