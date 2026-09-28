// Package api is campaigns-service's HTTP surface.
package api

import (
	"context"
	"net/http"
	"time"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/campaigns/internal/catalogclient"
	"campaigntrackerpro/services/campaigns/internal/models"
	"campaigntrackerpro/services/campaigns/internal/store"
)

// CampaignStore is exactly what the handlers below call on *store.Store —
// not its full surface. Sized this way (Interface Segregation) so handler
// logic (validation, status codes, event publishing) can be unit-tested
// against a fake, no real Postgres; see handlers_test.go's fakeStore.
// *store.Store satisfies this automatically — nothing in store/ changes.
type CampaignStore interface {
	Ping(ctx context.Context) error
	List(ctx context.Context, f store.Filter, page, limit int, today time.Time) ([]models.Campaign, int, error)
	Export(ctx context.Context, f store.Filter, today time.Time) ([]models.Campaign, error)
	ByID(ctx context.Context, id int64, today time.Time) (models.Campaign, bool, error)
	Create(ctx context.Context, c models.Campaign, today time.Time) (models.Campaign, error)
	UpdateBudget(ctx context.Context, id int64, budgetRupees *int64, today time.Time) (models.Campaign, bool, error)
	UpdateApprovalStatus(ctx context.Context, id int64, status string, today time.Time) (models.Campaign, bool, error)

	// Aggregation — SQL-side (services/campaigns/internal/store/aggregates.go),
	// backing /internal/aggregates/*. See docs/ROADMAP.md's Phase A for why
	// these exist instead of analytics-service pulling raw rows.
	KPITotals(ctx context.Context, f store.Filter) (models.KPITotals, error)
	WeeklyBuckets(ctx context.Context, f store.Filter, today time.Time) ([]models.WeeklyBucket, error)
	TrendBuckets(ctx context.Context, f store.Filter) ([]models.TrendBucket, error)
	RegionCounts(ctx context.Context, f store.Filter) ([]models.RegionCount, error)
	BenchmarkRows(ctx context.Context, f store.Filter, sortKey string, asc bool) ([]models.BenchmarkAgg, error)

	// AnomalyReport backs GET /api/v1/campaigns/anomalies — deliberately
	// unfiltered, see models/anomalies.go and store/anomalies.go.
	AnomalyReport(ctx context.Context, today time.Time) (models.AnomalyReport, error)

	// SnapshotBenchmark records one benchmark_snapshots row per subject,
	// called by the BENCHMARK_SNAPSHOT_ENABLED ticker (cmd/server/main.go),
	// not by any HTTP handler — see store/aggregates.go.
	SnapshotBenchmark(ctx context.Context, today time.Time) error

	// Creative-level tracking — see models/creatives.go.
	ListCreatives(ctx context.Context, campaignID int64) ([]models.Creative, error)
	CreateCreative(ctx context.Context, campaignID int64, headline, creativeType string, reach, spend int64) (models.Creative, error)
}

// CatalogLookup is what campaigns-service actually calls on
// catalogclient.Client: Lookup from handleCreateCampaign, ListAll from
// cmd/server/main.go's demo ticker. *catalogclient.Client satisfies this
// automatically.
type CatalogLookup interface {
	Lookup(ctx context.Context, subjectType, name string) (catalogclient.Subject, bool, error)
	ListAll(ctx context.Context) ([]catalogclient.Subject, error)
}

// ReportMailer is exactly what handleEmailReport/sendReport call on
// *platform.Mailer — same Interface Segregation reasoning as
// CampaignStore/CatalogLookup, so the "sent successfully" path is
// unit-testable via a fake (handlers_test.go's fakeMailer), not just the
// "not configured" path platform.Mailer's real zero-config behavior
// already covers for free. *platform.Mailer satisfies this automatically.
type ReportMailer interface {
	Send(to, subject, textBody, attachmentName string, attachment []byte) error
}

type Server struct {
	Store      CampaignStore
	Catalog    CatalogLookup
	Events     *platform.EventBus // nil-safe: publish is skipped if unset
	InstanceID string             // which of the N campaigns-service replicas this is — surfaced on /healthz so the gateway's LB is visibly alternating
	// Mailer is always non-nil (construct with platform.NewMailer even
	// when SMTP isn't configured) — it degrades internally to
	// platform.ErrMailerNotConfigured, same "always non-nil, degrades
	// internally" shape as platform.Cache, so call sites never nil-check it.
	Mailer ReportMailer
}

func NewRouter(s *Server, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/campaigns", s.handleListCampaigns)
	mux.HandleFunc("GET /api/v1/campaigns/export", s.handleExportCampaigns)
	mux.HandleFunc("GET /api/v1/campaigns/anomalies", s.handleListAnomalies)
	mux.HandleFunc("POST /api/v1/campaigns/email-report", s.handleEmailReport)
	mux.HandleFunc("GET /api/v1/campaigns/{id}", s.handleGetCampaign)
	mux.HandleFunc("POST /api/v1/campaigns", s.handleCreateCampaign)
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}", s.handleUpdateCampaignBudget)
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}/approval", s.handleUpdateApprovalStatus)
	mux.HandleFunc("GET /api/v1/campaigns/{id}/creatives", s.handleListCreatives)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/creatives", s.handleCreateCreative)
	mux.HandleFunc("GET /internal/aggregates/kpis", s.handleAggregateKPIs)
	mux.HandleFunc("GET /internal/aggregates/trend", s.handleAggregateTrend)
	mux.HandleFunc("GET /internal/aggregates/regions", s.handleAggregateRegions)
	mux.HandleFunc("GET /internal/aggregates/benchmark", s.handleAggregateBenchmark)

	handler := platform.RequestLogger(platform.CORS(allowedOrigin)(mux))
	return platform.Traced("campaigns-service", handler)
}
