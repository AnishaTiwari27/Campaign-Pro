// Package api wires the gateway's routing table: which upstream (or
// load-balanced pool) each path goes to.
package api

import (
	"net/http"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/gateway/internal/lb"
	"campaigntrackerpro/services/gateway/internal/proxy"
)

// Config's five pools all use the same lb.Pool type/pattern —
// campaigns-service was the first to be load-balanced (it's still the
// only one running true multi-replica in the demo-ticker sense), but
// catalog/analytics/auth/notifications are exactly as stateless and safe
// to run N-wide; see docs/ROADMAP.md's Phase A. NotificationsPool may be
// nil (no backend configured yet) — everything else is required.
type Config struct {
	CatalogPool       *lb.Pool
	AnalyticsPool     *lb.Pool
	AuthPool          *lb.Pool
	CampaignsPool     *lb.Pool
	AuditPool         *lb.Pool
	NotificationsPool *lb.Pool // nil = route not registered
	AllowedOrigin     string
	JWTSecret         []byte

	// Rate limits — see platform/ratelimit.go. APILimit is keyed per
	// authenticated user (so it must sit *inside* RequireAuth, applied to
	// the handler RequireAuth wraps, not around RequireAuth itself — see
	// RateLimit.ByUser's comment); AuthLimit is keyed per IP, ahead of any
	// verified identity, for the brute-force-prone /auth/* routes.
	APILimit  *platform.RateLimit
	AuthLimit *platform.RateLimit
}

func NewRouter(cfg Config) http.Handler {
	mux := http.NewServeMux()

	catalog := proxy.Balanced(cfg.CatalogPool)
	analytics := proxy.Balanced(cfg.AnalyticsPool)
	campaigns := proxy.Balanced(cfg.CampaignsPool)
	auth := proxy.Balanced(cfg.AuthPool)
	audit := proxy.Balanced(cfg.AuditPool)

	protected := func(h http.Handler) http.Handler { return RequireAuth(cfg.JWTSecret, cfg.APILimit.ByUser(h)) }
	rateLimitedAuth := cfg.AuthLimit.ByIP(auth)

	// Public — no session required. Everything else below needs one; see
	// authmw.go.
	mux.Handle("/auth/register", rateLimitedAuth)
	mux.Handle("/auth/login", rateLimitedAuth)
	mux.Handle("/auth/refresh", rateLimitedAuth)
	mux.Handle("/auth/logout", rateLimitedAuth)

	mux.Handle("/api/v1/meta", protected(catalog))
	mux.Handle("/api/v1/subjects", protected(catalog))

	// User management (list/change role) reuses the same auth proxy handler
	// as the public /auth/* routes above — no new pool — but wrapped in
	// protected() like every other /api/v1/* route: these need an
	// already-verified admin session, unlike /auth/* which is pre-identity.
	mux.Handle("/api/v1/users", protected(auth))
	mux.Handle("/api/v1/users/", protected(auth))

	mux.Handle("/api/v1/campaigns", protected(campaigns))  // list (GET) + create (POST, admin only — enforced in campaigns-service)
	mux.Handle("/api/v1/campaigns/", protected(campaigns)) // /{id} and /export

	mux.Handle("/api/v1/kpis", protected(analytics))
	mux.Handle("/api/v1/trend", protected(analytics))
	mux.Handle("/api/v1/regions", protected(analytics))
	mux.Handle("/api/v1/benchmark", protected(analytics))

	mux.Handle("/api/v1/audit-log", protected(audit))

	mux.HandleFunc("GET /healthz", handleGatewayHealth(cfg))

	traced := platform.Traced("gateway", platform.RequestLogger(platform.CORS(cfg.AllowedOrigin)(mux)))

	// /ws/notifications is deliberately routed outside the traced mux:
	// otelhttp's ResponseWriter wrapper doesn't implement http.Hijacker, so
	// proxying a WebSocket upgrade through it turns every connection
	// attempt into a 501 (see the identical note in
	// services/notifications/internal/api/server.go, where the upgrade is
	// actually handled). CORS is also irrelevant to a WS upgrade — the
	// browser doesn't preflight it — so RequestLogger alone is enough here.
	// It still requires a valid session: RequireAuthQuery, not RequireAuth,
	// since a browser's native WebSocket client can't set an Authorization
	// header on the upgrade request — the token travels as ?token= instead.
	outer := http.NewServeMux()
	if cfg.NotificationsPool != nil {
		outer.Handle("/ws/notifications", platform.RequestLogger(RequireAuthQuery(cfg.JWTSecret, proxy.Balanced(cfg.NotificationsPool))))
	}
	outer.Handle("/", traced)
	return outer
}

func handleGatewayHealth(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := map[string]any{
			"status":        "ok",
			"campaignsPool": cfg.CampaignsPool.Status(),
			"catalogPool":   cfg.CatalogPool.Status(),
			"analyticsPool": cfg.AnalyticsPool.Status(),
			"authPool":      cfg.AuthPool.Status(),
			"auditPool":     cfg.AuditPool.Status(),
		}
		if cfg.NotificationsPool != nil {
			status["notificationsPool"] = cfg.NotificationsPool.Status()
		}
		platform.WriteJSON(w, http.StatusOK, status)
	}
}
