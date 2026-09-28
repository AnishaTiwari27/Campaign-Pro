// Command server runs one campaigns-service instance — the only service
// allowed to read/write the "campaigns" Postgres schema.
//
//	PORT=8082 INSTANCE_ID=a go run ./cmd/server
//	PORT=8083 INSTANCE_ID=b go run ./cmd/server   # a second replica, for the gateway's LB to round-robin across
//
// Env vars: PORT (8082), INSTANCE_ID, DATABASE_URL, CATALOG_URL, NATS_URL,
// ALLOWED_ORIGIN, DEMO_TICKER ("true" to periodically synthesize a new
// campaign so the real-time notification pipeline has something to show).
//
// SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, SMTP_FROM configure
// outbound email (see platform/mail.go) — empty SMTP_HOST (the default)
// means email-report/scheduled-digest requests still work, they just
// report "not sent" instead of delivering anything; there's no SMTP relay
// in this environment by default. REPORTS_ENABLED ("true" to run the
// scheduled digest ticker), REPORTS_RECIPIENT (required if enabled —
// nothing guessed), REPORTS_INTERVAL (a Go duration, default "168h" =
// weekly; shorten it for local verification, same as DEMO_TICKER's
// compressed-for-dev 25s interval).
//
// BENCHMARK_SNAPSHOT_ENABLED ("true" to periodically record one
// campaigns.benchmark_snapshots row per subject, backing benchmark
// trending), BENCHMARK_SNAPSHOT_INTERVAL (a Go duration, default "24h";
// shorten it for local verification).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/campaigns/internal/api"
	"campaigntrackerpro/services/campaigns/internal/catalogclient"
	"campaigntrackerpro/services/campaigns/internal/models"
	"campaigntrackerpro/services/campaigns/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	instanceID := envOr("INSTANCE_ID", "a")

	shutdownTracer, err := platform.InitTracer("campaigns-service-" + instanceID)
	if err != nil {
		slog.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer shutdownTracer(context.Background())

	dbURL := envOr("DATABASE_URL", "postgres://campaigns_service:campaigns_dev_pw@localhost:5432/campaign_tracker?sslmode=disable")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	var bus *platform.EventBus
	if b, err := platform.ConnectEventBus(envOr("NATS_URL", "nats://127.0.0.1:4222")); err != nil {
		slog.Warn("NATS unreachable, campaign.created will not be published", "err", err)
	} else {
		bus = b
		defer bus.Close()
	}

	mailer := platform.NewMailer(
		envOr("SMTP_HOST", ""), envOr("SMTP_PORT", ""),
		envOr("SMTP_USER", ""), envOr("SMTP_PASSWORD", ""),
		envOr("SMTP_FROM", "reports@campaigntracker.dev"),
	)

	server := &api.Server{
		Store:      store.New(pool),
		Catalog:    catalogclient.New(envOr("CATALOG_URL", "http://localhost:8081")),
		Events:     bus,
		InstanceID: instanceID,
		Mailer:     mailer,
	}
	handler := api.NewRouter(server, envOr("ALLOWED_ORIGIN", "http://localhost:8080"))

	if envOr("DEMO_TICKER", "false") == "true" {
		go runDemoTicker(server)
	}

	if envOr("REPORTS_ENABLED", "false") == "true" {
		recipient := envOr("REPORTS_RECIPIENT", "")
		if recipient == "" {
			slog.Warn("REPORTS_ENABLED=true but REPORTS_RECIPIENT is empty — scheduled digest not started (explicit config required, nothing guessed)")
		} else {
			interval, err := time.ParseDuration(envOr("REPORTS_INTERVAL", "168h"))
			if err != nil {
				slog.Warn("invalid REPORTS_INTERVAL, scheduled digest not started", "err", err)
			} else {
				go runScheduledReports(server, recipient, interval)
			}
		}
	}

	if envOr("BENCHMARK_SNAPSHOT_ENABLED", "false") == "true" {
		interval, err := time.ParseDuration(envOr("BENCHMARK_SNAPSHOT_INTERVAL", "24h"))
		if err != nil {
			slog.Warn("invalid BENCHMARK_SNAPSHOT_INTERVAL, snapshot ticker not started", "err", err)
		} else {
			go runBenchmarkSnapshots(server, interval)
		}
	}

	addr := ":" + envOr("PORT", "8082")
	slog.Info("campaigns-service starting", "addr", addr, "instance", instanceID)
	if err := http.ListenAndServe(addr, handler); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// runDemoTicker periodically synthesizes one new campaign for a random
// existing subject, so the WebSocket/notifications pipeline has something
// real to push without needing an actual ad-platform feed wired up.
func runDemoTicker(s *api.Server) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := synthesizeOneCampaign(s); err != nil {
			slog.Warn("demo ticker: failed to synthesize campaign", "err", err)
		}
	}
}

// runScheduledReports periodically emails the unfiltered campaign report
// to recipient — same shape as runDemoTicker (time.NewTicker/defer
// Stop()/for range ticker.C, each tick bounded by its own
// context.WithTimeout, logs-and-continues on error, never crashes the
// service). Shares handleEmailReport's own generation/send logic via
// Server.GenerateAndSendReport, so a manual "email me this report" click
// and this ticker can never format a report differently.
func runScheduledReports(s *api.Server, recipient string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		sent, reason, err := s.GenerateAndSendReport(ctx, store.Filter{}, recipient)
		cancel()
		if err != nil {
			slog.Warn("scheduled report: failed to generate", "err", err)
			continue
		}
		if !sent {
			slog.Warn("scheduled report: not delivered", "reason", reason)
			continue
		}
		slog.Info("scheduled report: sent", "to", recipient)
	}
}

// runBenchmarkSnapshots periodically records one benchmark_snapshots row
// per subject — same shape as runDemoTicker/runScheduledReports
// (time.NewTicker, bounded context.WithTimeout per tick, logs-and-
// continues on error, never crashes the service).
func runBenchmarkSnapshots(s *api.Server, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := s.Store.SnapshotBenchmark(ctx, time.Now())
		cancel()
		if err != nil {
			slog.Warn("benchmark snapshot: failed", "err", err)
			continue
		}
		slog.Info("benchmark snapshot: recorded")
	}
}

var demoRegions = []string{"Mumbai", "Delhi NCR", "Bengaluru", "South Zone", "West Zone", "Pan-India"}
var demoAdTypes = map[string]string{
	"Social Media": "Instagram", "Influencer": "YouTube Creators", "Google Ads": "Google Search",
	"Display": "Google Display", "Video": "YouTube", "Performance": "Meta Ads",
}

func synthesizeOneCampaign(s *api.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Pull a live subject list from catalog-service rather than hardcoding
	// one here — stays correct if the catalog changes.
	subjects, err := s.Catalog.ListAll(ctx)
	if err != nil || len(subjects) == 0 {
		return err
	}
	subj := subjects[rand.Intn(len(subjects))]

	adTypes := make([]string, 0, len(demoAdTypes))
	for k := range demoAdTypes {
		adTypes = append(adTypes, k)
	}
	adType := adTypes[rand.Intn(len(adTypes))]
	region := demoRegions[rand.Intn(len(demoRegions))]

	today := time.Now()
	reach := int64(40000 + rand.Intn(900000))
	spend := reach / 22

	created, err := s.Store.Create(ctx, models.Campaign{
		Subject: subj.Name, SubjectType: subj.Type, Category: subj.Category,
		Region: region, AdType: adType, Platform: demoAdTypes[adType],
		Start: today, End: today.AddDate(0, 0, 14+rand.Intn(21)),
		Reach: reach, Spend: spend,
	}, today)
	if err != nil {
		return err
	}
	if s.Events != nil {
		s.Events.Publish("campaign.created", created.AsJSON())
	}
	// "system"/"system" in place of real X-User-* headers — a demo-ticker
	// tick isn't an HTTP request, so there's no caller to attribute this
	// to. Without this, the audit trail would look mysteriously
	// incomplete for every ticker-created row.
	s.AuditPublish("campaign.created", "system", "", "system", created.ID, fmt.Sprintf("created %q (demo ticker)", created.Subject))
	slog.Info("demo ticker: synthesized campaign", "id", created.ID, "subject", created.Subject)
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
