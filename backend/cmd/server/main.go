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
	"campaigntrackerpro/platform/llm"
	"campaigntrackerpro/platform/mail"
	"campaigntrackerpro/platform/oidc"
	"campaigntrackerpro/platform/scope"
	analytics "campaigntrackerpro/services/analytics/module"
	campaigns "campaigntrackerpro/services/campaigns/module"
	creators "campaigntrackerpro/services/creators/module"
	endorsements "campaigntrackerpro/services/endorsements/module"
	identitysvc "campaigntrackerpro/services/identity"
	identity "campaigntrackerpro/services/identity/module"
	ingestsvc "campaigntrackerpro/services/ingest"
	ingest "campaigntrackerpro/services/ingest/module"
	insights "campaigntrackerpro/services/insights/module"
	radar "campaigntrackerpro/services/radar/module"
	reports "campaigntrackerpro/services/reports/module"
	"campaigntrackerpro/web"
)

const workerTick = 30 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Both said before the connection is attempted, because the failure it
	// causes names localhost and a user that exists only on a developer's
	// machine, which reads as a database problem rather than a missing
	// variable.
	//
	// On a managed host the fallback cannot work at all, so that is fatal
	// here rather than a warning followed by a confusing dial: a deploy
	// should fail on the sentence that names the fix.
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	if cfg.DatabaseURLDefaulted {
		logger.Warn("DATABASE_URL is not set — falling back to the local development database; " +
			"set it to your managed Postgres URL if this is a deployment")
	}

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

	// Positive confirmation of where the data actually lives. A deploy quietly
	// pointing at the wrong database is expensive to notice later, and this is
	// the cheapest possible way to rule it out. Taken from the parsed config,
	// field by field, so the password cannot reach the log with it.
	conn := pool.Config().ConnConfig
	logger.Info("database connected",
		"host", conn.Host, "port", conn.Port, "database", conn.Database, "user", conn.User)

	db := database.New(pool)

	// Dependency order is the service graph: campaigns owns the core data,
	// everything else reads from it through a narrow interface.
	campaignsMod := campaigns.New(db, logger)
	// campaigns owns the users table, so it supplies the capability signup
	// needs to write one. Neither service imports the other.
	identityMod := identity.New(db, auditBridge{campaignsMod}, campaignsMod.UserDirectory(),
		identity.Options{
			SecureCookies: cfg.SecureCookies,
			AllowSignup:   cfg.AllowSignup,
			SSO:           discoverSSO(ctx, cfg, logger),
			SSOOrgID:      cfg.OIDCOrgID,
		}, logger)
	creatorsMod := creators.New(db, campaignsMod.Service(), logger)
	insightsMod := insights.New(campaignsMod.Service(), explainer(cfg, logger), logger)
	radarMod := radar.New(db, campaignsMod.Service(), logger)
	// No database: the public record is reference data, not the agency's.
	endorsementsMod := endorsements.New(logger)
	ingestMod := ingest.New(db, ingestsvc.GoogleAdsConfig{
		DeveloperToken:  cfg.GoogleAdsDeveloperToken,
		ClientID:        cfg.GoogleAdsClientID,
		ClientSecret:    cfg.GoogleAdsClientSecret,
		RefreshToken:    cfg.GoogleAdsRefreshToken,
		CustomerID:      cfg.GoogleAdsCustomerID,
		LoginCustomerID: cfg.GoogleAdsLoginCustomerID,
	}, ingestsvc.MetaConfig{
		AccessToken: cfg.MetaAccessToken,
		AdAccountID: cfg.MetaAdAccountID,
	}, explainer(cfg, logger), logger)
	analyticsMod := analytics.New(db, campaignsMod.Service(), creatorsMod.Store(), logger)
	reportsMod := reports.New(db, campaignsMod.Service(), campaignsMod.Detector(),
		mailerFor(cfg, logger), logger, workerTick)

	// The worker is not a request and is not confined to one tenant: it
	// runs every organisation's due reports. Without this it would
	// inherit the zero org, match nothing, and silently stop firing.
	go reportsMod.Run(scope.SystemOrg(ctx))

	// Benchmark history, on its own ticker rather than inside the report
	// worker: reports has no business knowing about analytics, and the
	// composition root is the one place allowed to know about both.
	go captureBenchmarks(scope.SystemOrg(ctx), analyticsMod, logger)

	// Launch radar: polls its sources and settles outcomes on its own
	// slow loop, for the same reason — the composition root is the one
	// place allowed to know about both.
	go radarMod.Run(ctx)

	router := newRouter(analyticsMod.Health(), identityMod, campaignsMod.Grants(),
		rateLimits{APIPerMinute: cfg.RateAPIPerMinute, AuthPerMinute: cfg.RateAuthPerMinute}, logger,
		identityMod, campaignsMod, creatorsMod, analyticsMod, reportsMod, endorsementsMod, ingestMod,
		insightsMod, radarMod)

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

// mailerFor picks the delivery implementation. Log mode is the default
// and stays the default: a deployment with no SMTP configured should
// say so in its logs rather than appear to be sending digests that
// nobody receives.
func mailerFor(cfg config.Config, logger *slog.Logger) mail.Mailer {
	if cfg.MailMode != "smtp" {
		return mail.LogMailer{Logger: logger}
	}
	if cfg.SMTPHost == "" {
		logger.Error("MAIL_MODE=smtp but SMTP_HOST is unset — falling back to log mode")
		return mail.LogMailer{Logger: logger}
	}
	logger.Info("sending report digests over smtp", "host", cfg.SMTPHost, "from", cfg.MailFrom)
	return mail.SMTPMailer{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort,
		Username: cfg.SMTPUser, Password: cfg.SMTPPass,
		From: cfg.MailFrom, Timeout: 30 * time.Second,
	}
}

// benchmarkTick is how often the benchmark history is rewritten. The
// write is keyed by day, so this only has to be often enough that a
// restart or a sleeping instance does not lose the day entirely —
// not often enough to catch every change.
const benchmarkTick = 30 * time.Minute

func captureBenchmarks(ctx context.Context, a *analytics.Module, logger *slog.Logger) {
	ticker := time.NewTicker(benchmarkTick)
	defer ticker.Stop()
	a.CaptureBenchmarks(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.CaptureBenchmarks(ctx)
		}
	}
}

// discoverSSO reads the identity provider's configuration at boot, or
// returns nil if none is configured.
//
// Discovery is the slowest part of the flow and the part most likely to
// be misconfigured, so it happens once here rather than on somebody's
// first sign-in. A provider that is configured but unreachable logs and
// disables SSO rather than killing the process: the password form still
// works, and taking the whole deployment down because an external
// service is having a bad morning is the wrong trade.
func discoverSSO(ctx context.Context, cfg config.Config, logger *slog.Logger) identitysvc.SSOProvider {
	if !cfg.SSOEnabled() {
		return nil
	}
	p, err := oidc.Discover(ctx, oidc.Config{
		Issuer:       cfg.OIDCIssuer,
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURL,
	})
	if err != nil {
		logger.Error("sso: discovery failed, continuing with password sign-in only",
			"issuer", cfg.OIDCIssuer, "err", err)
		return nil
	}
	logger.Info("sso enabled", "issuer", cfg.OIDCIssuer, "org", cfg.OIDCOrgID)
	return p
}

// explainer returns the model client, or nil when no key is set.
//
// Nil rather than an error: signal explanations are a nicety, and a
// deployment without a key should still show the figures assembled
// into a sentence rather than lose the panel or fail to boot.
func explainer(cfg config.Config, logger *slog.Logger) llm.Client {
	if cfg.LLMAPIKey == "" {
		logger.Info("no model key set — signal explanations will be assembled from the figures")
		return nil
	}
	logger.Info("signal explanations enabled", "model", cfg.LLMModel)
	return llm.Anthropic{APIKey: cfg.LLMAPIKey, Model: cfg.LLMModel, MaxTokens: 400, Timeout: 20 * time.Second}
}
