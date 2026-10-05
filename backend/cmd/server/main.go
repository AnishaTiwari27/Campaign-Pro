// Command server is the composition root. It is the only place that knows
// every service exists: it builds the shared platform, hands each service
// factory what it depends on, and mounts the result. It cannot reach into
// any service's internals — the compiler forbids it.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"campaigntrackerpro/platform/config"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/mail"
	analytics "campaigntrackerpro/services/analytics/module"
	campaigns "campaigntrackerpro/services/campaigns/module"
	creators "campaigntrackerpro/services/creators/module"
	identity "campaigntrackerpro/services/identity/module"
	reports "campaigntrackerpro/services/reports/module"
	"campaigntrackerpro/web"
)

const workerTick = 30 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Schema first, before anything queries it. A deploy to a blank database
	// has to come up working, and the alternative — remembering to run the
	// migrate CLI by hand against production — is the kind of step that gets
	// skipped exactly once.
	if err := database.Migrate(cfg.DatabaseURL, logger); err != nil {
		logger.Error("failed to migrate database", "err", err)
		os.Exit(1)
	}

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	db := database.New(pool)

	// Dependency order is the service graph: campaigns owns the core data,
	// everything else reads from it through a narrow interface.
	campaignsMod := campaigns.New(db, logger)
	identityMod := identity.New(db, auditBridge{campaignsMod}, cfg.SecureCookies, logger)
	creatorsMod := creators.New(db, campaignsMod.Service(), logger)
	analyticsMod := analytics.New(db, campaignsMod.Service(), creatorsMod.Store(), logger)
	reportsMod := reports.New(db, campaignsMod.Service(), campaignsMod.Detector(),
		mail.LogMailer{Logger: logger}, logger, workerTick)

	go reportsMod.Run(ctx)

	router := newRouter(analyticsMod.Health(), identityMod, logger,
		identityMod, campaignsMod, creatorsMod, analyticsMod, reportsMod)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: router}

	// Said once, loudly: an image built without the frontend serves a notice
	// page on every route, and that is far easier to diagnose from a log line
	// than from a browser.
	if !web.Built() {
		logger.Warn("no frontend build embedded — serving the placeholder notice; " +
			"use the root Dockerfile or `make build-web` to embed the real app")
	}

	go func() {
		logger.Info("server listening", "port", cfg.Port, "frontend", web.Built())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
