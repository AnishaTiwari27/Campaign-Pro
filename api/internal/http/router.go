package httpapi

import (
	"log/slog"
	"net/http"

	"campaigntrackerpro/internal/service"
	"campaigntrackerpro/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(s *store.Store, campaigns *service.Campaigns, overview *service.OverviewService, reports *service.Reports, anomaly *service.AnomalyDetector, adminEmail string, logger *slog.Logger) http.Handler {
	h := &Handlers{Campaigns: campaigns, Overview: overview, Reports: reports, Anomaly: anomaly, Logger: logger}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(RequestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}))

	r.Get("/health", h.Health)

	r.Route("/api", func(r chi.Router) {
		r.Use(AuthMiddleware(s, adminEmail, logger))

		r.Get("/health", h.Health)
		r.Get("/me", h.Me)
		r.Get("/search", h.Search)

		r.Get("/overview", h.GetOverview)
		r.Get("/benchmark", h.GetBenchmark)

		r.Route("/campaigns", func(r chi.Router) {
			r.Get("/", h.ListCampaigns)
			r.Get("/anomalies", h.Anomalies)
			r.Get("/export.csv", h.ExportCSV)
			r.Post("/bulk-decision", h.BulkDecision)
			r.Get("/{id}", h.GetCampaign)
			r.Post("/{id}/decision", h.Decision)
			r.Post("/{id}/pause", h.Pause)
			r.Post("/{id}/notes", h.AddNote)
		})

		r.Route("/regions", func(r chi.Router) {
			r.Get("/", h.ListRegions)
			r.Get("/{id}", h.GetRegion)
		})

		r.Route("/reports", func(r chi.Router) {
			r.Get("/", h.ListReports)
			r.Post("/", h.CreateReport)
			r.Get("/{id}", h.GetReport)
			r.Patch("/{id}", h.UpdateReport)
			r.Post("/{id}/run", h.RunReport)
			r.Post("/{id}/test", h.TestReport)
			r.Get("/{id}/runs", h.ListReportRuns)
		})
	})

	return r
}
