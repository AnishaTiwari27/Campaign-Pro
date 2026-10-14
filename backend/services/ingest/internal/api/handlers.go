// Package api is ingestion's transport. It owns /ingest.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/ingest"
	"campaigntrackerpro/services/ingest/internal/service"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	svc       *service.Sync
	providers map[string]ingest.Provider
	logger    *slog.Logger
}

func New(svc *service.Sync, providers map[string]ingest.Provider, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, providers: providers, logger: logger}
}

func (h *Handlers) Routes(r chi.Router) {
	r.Route("/ingest", func(r chi.Router) {
		r.Get("/sources", h.ListSources)
		r.Get("/runs", h.ListRuns)
		r.Post("/{source}/sync", h.Sync)
		r.Post("/csv", h.UploadCSV)
	})
}

// maxUpload bounds a CSV upload. Ten megabytes is far more than an
// export of a real ad account and small enough that a bad request
// cannot exhaust the instance's memory.
const maxUpload = 10 << 20

type sourceDTO struct {
	Name string `json:"name"`
	// Configured is false for a provider whose credentials are absent,
	// so the UI can show it as available-but-not-set-up rather than
	// offering a button that will only ever fail.
	Configured bool `json:"configured"`
}

func (h *Handlers) ListSources(w http.ResponseWriter, r *http.Request) {
	out := make([]sourceDTO, 0, len(h.providers))
	for name, p := range h.providers {
		configured := true
		if c, ok := p.(interface{ Configured() bool }); ok {
			configured = c.Configured()
		}
		out = append(out, sourceDTO{Name: name, Configured: configured})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handlers) ListRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := h.svc.Runs(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": runDTOs(runs)})
}

// Sync runs a configured provider. Writing to the campaign table is an
// agency operation; a client account has no business triggering it.
func (h *Handlers) Sync(w http.ResponseWriter, r *http.Request) {
	if isClient(r) {
		httpx.WriteForbidden(w, "ingestion writes to the agency's campaigns and isn't available to client accounts")
		return
	}
	name := chi.URLParam(r, "source")
	p, ok := h.providers[name]
	if !ok {
		httpx.WriteNotFound(w, "no such ingestion source")
		return
	}
	// A day's overlap. Providers filter on this, and overlapping is free
	// because the upsert is idempotent — whereas a gap loses rows.
	since := time.Now().Add(-24 * time.Hour)
	res, err := h.svc.Run(r.Context(), p, since)
	if err != nil {
		// The run is recorded either way, so this reports rather than
		// hides the failure.
		httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{
			"error": err.Error(), "source": res.Source,
		})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resultDTO(res))
}

// UploadCSV is the path that works without any platform credentials at
// all, and runs the same matching and upsert as every other source.
func (h *Handlers) UploadCSV(w http.ResponseWriter, r *http.Request) {
	if isClient(r) {
		httpx.WriteForbidden(w, "ingestion writes to the agency's campaigns and isn't available to client accounts")
		return
	}
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		httpx.WriteValidationError(w, "could not read the upload", "file")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteValidationError(w, "attach the export as 'file'", "file")
		return
	}
	defer file.Close()

	source := r.FormValue("source")
	if source == "" {
		source = "csv"
	}
	res, err := h.svc.Run(r.Context(), ingest.CSVProvider{SourceName: source, Reader: file}, time.Time{})
	if err != nil {
		// A malformed CSV is the user's to fix, so the message says what
		// is wrong with it rather than being swallowed as a 500.
		httpx.WriteValidationError(w, err.Error(), "file")
		return
	}
	h.logger.Info("csv ingested", "file", header.Filename, "rows", res.Seen)
	httpx.WriteJSON(w, http.StatusOK, resultDTO(res))
}
