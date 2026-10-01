package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"campaigntrackerpro/internal/service"

	"github.com/go-chi/chi/v5"
)

func parseListParams(r *http.Request) service.ListParams {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	per, _ := strconv.Atoi(q.Get("per"))
	if per <= 0 {
		per = 10
	}
	return service.ListParams{
		Search: q.Get("q"), SubjectType: q.Get("type"), Category: q.Get("category"),
		Region: q.Get("region"), AdType: q.Get("adType"), Status: q.Get("status"),
		Approval: q.Get("approval"), Range: q.Get("range"),
		Sort: q.Get("sort"), Dir: q.Get("dir"), Page: page, Per: per,
	}
}

func (h *Handlers) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	result, err := h.Campaigns.List(r.Context(), parseListParams(r))
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse(result))
}

func (h *Handlers) GetCampaign(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.Campaigns.GetDetail(r.Context(), id, parseListParams(r))
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignDetailDTO(detail))
}

type decisionRequest struct {
	Action string `json:"action"`
}

func (h *Handlers) Decision(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req decisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeValidationError(w, "invalid JSON body", "")
		return
	}
	user, _ := UserFromContext(r.Context())
	camp, err := h.Campaigns.Decision(r.Context(), id, req.Action, user.Name, user.CanApprove)
	if err != nil {
		if err == service.ErrValidation {
			writeValidationError(w, "action must be approve, reject or reopen", "action")
			return
		}
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignDTO(h.Campaigns.RowOf(r.Context(), camp)))
}

type bulkDecisionRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

func (h *Handlers) BulkDecision(w http.ResponseWriter, r *http.Request) {
	var req bulkDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeValidationError(w, "invalid JSON body", "")
		return
	}
	if len(req.IDs) == 0 {
		writeValidationError(w, "ids must not be empty", "ids")
		return
	}
	user, _ := UserFromContext(r.Context())
	camps, err := h.Campaigns.BulkDecision(r.Context(), req.IDs, req.Action, user.Name, user.CanApprove)
	if err != nil {
		if err == service.ErrValidation {
			writeValidationError(w, "action must be approve or reject", "action")
			return
		}
		writeServiceError(w, h.Logger, err)
		return
	}
	rows := make([]CampaignDTO, len(camps))
	for i, c := range camps {
		rows[i] = campaignDTO(h.Campaigns.RowOf(r.Context(), c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (h *Handlers) Pause(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, _ := UserFromContext(r.Context())
	camp, err := h.Campaigns.Pause(r.Context(), id, user.Name)
	if err != nil {
		if err == service.ErrValidation {
			writeValidationError(w, "campaign must be live or paused to toggle", "")
			return
		}
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignDTO(h.Campaigns.RowOf(r.Context(), camp)))
}

type noteRequest struct {
	Text string `json:"text"`
}

func (h *Handlers) AddNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeValidationError(w, "invalid JSON body", "")
		return
	}
	user, _ := UserFromContext(r.Context())
	event, err := h.Campaigns.AddNote(r.Context(), id, req.Text, user.Name)
	if err != nil {
		if err == service.ErrValidation {
			writeValidationError(w, "text must not be empty", "text")
			return
		}
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, auditDTO(event))
}

func (h *Handlers) Anomalies(w http.ResponseWriter, r *http.Request) {
	camps, err := h.Campaigns.Anomalies(r.Context())
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	rows := make([]CampaignDTO, len(camps))
	for i, c := range camps {
		rows[i] = campaignDTO(h.Campaigns.RowOf(r.Context(), c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (h *Handlers) ExportCSV(w http.ResponseWriter, r *http.Request) {
	params := parseListParams(r)
	params.Page = 1
	params.Per = 100000
	result, err := h.Campaigns.List(r.Context(), params)
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	body := service.ExportCSV(result.Items)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="campaigns.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
