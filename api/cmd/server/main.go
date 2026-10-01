// Command server runs the Campaign Tracker Pro API: HTTP router plus the
// in-process report/anomaly worker.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"campaigntrackerpro/internal/config"
	httpapi "campaigntrackerpro/internal/http"
	"campaigntrackerpro/internal/service"
	"campaigntrackerpro/internal/store"
	"campaigntrackerpro/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	st := store.New(pool)
	campaigns := service.NewCampaigns(st)
	overview := service.NewOverview(campaigns)
	mailer := service.LogMailer{Logger: logger}
	reports := service.NewReports(st, campaigns, mailer)
	creators := service.NewCreators(st, campaigns)
	anomaly := service.NewAnomalyDetector(campaigns, logger)

	bg := worker.New(anomaly, reports, logger, 30*time.Second)
	go bg.Run(ctx)

	router := httpapi.NewRouter(st, campaigns, overview, reports, creators, anomaly, cfg.AdminEmail, logger)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		logger.Info("server listening", "port", cfg.Port)
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
