// Command server runs audit-service — subscribes to NATS' campaign.audit
// subject (published by campaigns-service; see
// services/campaigns/internal/api's auditPublish) and persists each event
// as an append-only row, plus serves the admin-only read API
// (GET /api/v1/audit-log). See docs/ARCHITECTURE.md's Audit log section.
//
//	go run ./cmd/server
//
// Env vars: PORT (8090), DATABASE_URL, NATS_URL, ALLOWED_ORIGIN.
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/audit/internal/api"
	"campaigntrackerpro/services/audit/internal/models"
	"campaigntrackerpro/services/audit/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	shutdownTracer, err := platform.InitTracer("audit-service")
	if err != nil {
		slog.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer shutdownTracer(context.Background())

	dbURL := envOr("DATABASE_URL", "postgres://audit_service:audit_dev_pw@localhost:5432/campaign_tracker?sslmode=disable")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	st := store.New(pool)

	// Unlike notifications-service (where a NATS-less hub has nothing to
	// relay, so connect failure is fatal), this service's read API over
	// already-stored rows stays useful even if new events stop arriving —
	// warn and keep serving, don't exit.
	if bus, err := platform.ConnectEventBus(envOr("NATS_URL", "nats://127.0.0.1:4222")); err != nil {
		slog.Warn("NATS unreachable, audit entries will not be recorded until it's back", "err", err)
	} else {
		defer bus.Close()
		if err := bus.Subscribe("campaign.audit", func(payload []byte) {
			var event models.AuditEvent
			if err := json.Unmarshal(payload, &event); err != nil {
				slog.Error("audit event: bad payload", "err", err)
				return
			}
			if err := st.Insert(context.Background(), event); err != nil {
				slog.Error("audit event: insert failed", "action", event.Action, "err", err)
			}
		}); err != nil {
			slog.Warn("NATS subscribe failed, audit entries will not be recorded", "err", err)
		}
	}

	server := &api.Server{Store: st}
	handler := api.NewRouter(server, envOr("ALLOWED_ORIGIN", "http://localhost:8080"))

	addr := ":" + envOr("PORT", "8090")
	slog.Info("audit-service starting", "addr", addr)
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
