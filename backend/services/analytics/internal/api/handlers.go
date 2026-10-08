package api

import (
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/analytics/internal/service"

	"github.com/go-chi/chi/v5"
)

// Handlers is the analytics transport. It owns the cross-cutting read
// endpoints: the overview, benchmarks, regions, search, health and me.
type Handlers struct {
	svc    *service.Analytics
	db     *database.DB
	logger *slog.Logger
}

func New(svc *service.Analytics, db *database.DB, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, db: db, logger: logger}
}

func (h *Handlers) Routes(r chi.Router) {
	r.Get("/search", h.Search)
	r.Get("/overview", h.GetOverview)
	r.Get("/benchmark", h.GetBenchmark)
	r.Route("/regions", func(r chi.Router) {
		r.Get("/", h.ListRegions)
		r.Get("/{id}", h.GetRegion)
	})
}

// clientPrincipal is the only thing this service needs to know about the
// caller: whether they are external. Declared here as an interface rather
// than imported, so analytics stays free of identity's concrete type — the
// same shape every other cross-service dependency in this codebase takes.
type clientPrincipal interface{ IsClient() bool }

// isClient reports whether the request is from an external client account.
// A principal that cannot answer is treated as not-a-client, because every
// route here already sits behind the session guard — this decides what an
// authenticated caller may see, not whether they are authenticated.
func isClient(r *http.Request) bool {
	v, _ := httpx.UserFromContext(r.Context())
	p, ok := v.(clientPrincipal)
	return ok && p.IsClient()
}

// Health is mounted outside /api too, so it stays a plain handler.
func (h *Handlers) HealthHandler() http.HandlerFunc { return h.Health }
