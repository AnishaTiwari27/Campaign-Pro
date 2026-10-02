package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/identity"
	"campaigntrackerpro/services/reports/internal/service"

	"github.com/go-chi/chi/v5"
)

// Handlers is the reports transport; it owns the /reports URL space.
type Handlers struct {
	svc    *service.Reports
	logger *slog.Logger
}

func New(svc *service.Reports, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

func (h *Handlers) Routes(r chi.Router) {
	r.Route("/reports", func(r chi.Router) {
		r.Get("/", h.ListReports)
		r.Post("/", h.CreateReport)
		r.Get("/{id}", h.GetReport)
		r.Patch("/{id}", h.UpdateReport)
		r.Post("/{id}/run", h.RunReport)
		r.Post("/{id}/test", h.TestReport)
		r.Get("/{id}/runs", h.ListReportRuns)
	})
}

func (h *Handlers) ListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.svc.List(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	items := reportDTOs(reports)
	for i, rep := range reports {
		if run, err := h.svc.Store.GetLastReportRun(r.Context(), rep.ID); err == nil {
			d := reportRunDTO(run)
			items[i].LastRun = &d
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) GetReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rep, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reportDTO(rep))
}

type createReportRequest struct {
	Name         string         `json:"name"`
	Cadence      string         `json:"cadence"`
	Recipients   []string       `json:"recipients"`
	ScopeFilters map[string]any `json:"scopeFilters"`
	ScopeLabel   string         `json:"scopeLabel"`
	Columns      []string       `json:"columns"`
}

func (h *Handlers) CreateReport(w http.ResponseWriter, r *http.Request) {
	var req createReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}
	rep, err := h.svc.Create(r.Context(), service.CreateReportInput{
		Name: req.Name, Cadence: req.Cadence, Recipients: req.Recipients,
		ScopeFilters: req.ScopeFilters, ScopeLabel: req.ScopeLabel, Columns: req.Columns,
	})
	if err != nil {
		if err == service.ErrValidation {
			httpx.WriteValidationError(w, "check name, cadence, recipients and columns", "")
			return
		}
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, reportDTO(rep))
}

type updateReportRequest struct {
	Name       *string  `json:"name"`
	Enabled    *bool    `json:"enabled"`
	Cadence    *string  `json:"cadence"`
	Recipients []string `json:"recipients"`
	Columns    []string `json:"columns"`
}

func (h *Handlers) UpdateReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}
	rep, err := h.svc.Update(r.Context(), id, service.UpdateReportInput{
		Name: req.Name, Enabled: req.Enabled, Cadence: req.Cadence,
		Recipients: req.Recipients, Columns: req.Columns,
	})
	if err != nil {
		if err == service.ErrValidation {
			httpx.WriteValidationError(w, "check cadence, recipients and columns", "")
			return
		}
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reportDTO(rep))
}

func (h *Handlers) RunReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	run, err := h.svc.Run(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reportRunDTO(run))
}

func (h *Handlers) TestReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	v, _ := httpx.UserFromContext(r.Context())
	user, _ := v.(identity.User)
	if err := h.svc.Test(r.Context(), id, user.Email); err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sent": true, "to": user.Email})
}

func (h *Handlers) ListReportRuns(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit := int32(20)
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = int32(v)
	}
	runs, err := h.svc.ListRuns(r.Context(), id, limit)
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": reportRunDTOs(runs)})
}
