package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"campaigntrackerpro/internal/service"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) ListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.Reports.List(r.Context())
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	items := reportDTOs(reports)
	for i, rep := range reports {
		if run, err := h.Reports.Store.GetLastReportRun(r.Context(), rep.ID); err == nil {
			d := reportRunDTO(run)
			items[i].LastRun = &d
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) GetReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rep, err := h.Reports.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, reportDTO(rep))
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
		writeValidationError(w, "invalid JSON body", "")
		return
	}
	rep, err := h.Reports.Create(r.Context(), service.CreateReportInput{
		Name: req.Name, Cadence: req.Cadence, Recipients: req.Recipients,
		ScopeFilters: req.ScopeFilters, ScopeLabel: req.ScopeLabel, Columns: req.Columns,
	})
	if err != nil {
		if err == service.ErrValidation {
			writeValidationError(w, "check name, cadence, recipients and columns", "")
			return
		}
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, reportDTO(rep))
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
		writeValidationError(w, "invalid JSON body", "")
		return
	}
	rep, err := h.Reports.Update(r.Context(), id, service.UpdateReportInput{
		Name: req.Name, Enabled: req.Enabled, Cadence: req.Cadence,
		Recipients: req.Recipients, Columns: req.Columns,
	})
	if err != nil {
		if err == service.ErrValidation {
			writeValidationError(w, "check cadence, recipients and columns", "")
			return
		}
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, reportDTO(rep))
}

func (h *Handlers) RunReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	run, err := h.Reports.Run(r.Context(), id)
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, reportRunDTO(run))
}

func (h *Handlers) TestReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, _ := UserFromContext(r.Context())
	if err := h.Reports.Test(r.Context(), id, user.Email); err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sent": true, "to": user.Email})
}

func (h *Handlers) ListReportRuns(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit := int32(20)
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = int32(v)
	}
	runs, err := h.Reports.ListRuns(r.Context(), id, limit)
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": reportRunDTOs(runs)})
}
