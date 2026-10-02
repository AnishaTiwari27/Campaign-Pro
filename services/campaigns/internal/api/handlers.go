package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/campaigns/internal/service"

	"github.com/go-chi/chi/v5"
)

// Handlers is this service's transport. It owns its own URL space: the
// composition root mounts Routes and never names an individual path.
type Handlers struct {
	svc    *service.Campaigns
	logger *slog.Logger
}

func New(svc *service.Campaigns, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

func (h *Handlers) Routes(r chi.Router) {
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
}

func parseListParams(r *http.Request) campaigns.ListParams {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	per, _ := strconv.Atoi(q.Get("per"))
	if per <= 0 {
		per = 10
	}
	return campaigns.ListParams{
		Search: q.Get("q"), SubjectType: q.Get("type"), Category: q.Get("category"),
		Region: q.Get("region"), AdType: q.Get("adType"), Status: q.Get("status"),
		Approval: q.Get("approval"), Range: q.Get("range"),
		Sort: q.Get("sort"), Dir: q.Get("dir"), Page: page, Per: per,
	}
}

func (h *Handlers) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.List(r.Context(), parseListParams(r))
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, campaigns.NewListResponse(result))
}

func (h *Handlers) GetCampaign(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.svc.GetDetail(r.Context(), id, parseListParams(r))
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewCampaignDetailDTO(detail))
}

type decisionRequest struct {
	Action string `json:"action"`
}

func (h *Handlers) Decision(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req decisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}
	user := currentUser(r)
	camp, err := h.svc.Decision(r.Context(), id, req.Action, user.Name, user.CanApprove)
	if err != nil {
		if err == service.ErrValidation {
			httpx.WriteValidationError(w, "action must be approve, reject or reopen", "action")
			return
		}
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, campaigns.NewCampaignDTO(h.svc.RowOf(r.Context(), camp)))
}

type bulkDecisionRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

func (h *Handlers) BulkDecision(w http.ResponseWriter, r *http.Request) {
	var req bulkDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}
	if len(req.IDs) == 0 {
		httpx.WriteValidationError(w, "ids must not be empty", "ids")
		return
	}
	user := currentUser(r)
	camps, err := h.svc.BulkDecision(r.Context(), req.IDs, req.Action, user.Name, user.CanApprove)
	if err != nil {
		if err == service.ErrValidation {
			httpx.WriteValidationError(w, "action must be approve or reject", "action")
			return
		}
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	rows := make([]campaigns.CampaignDTO, len(camps))
	for i, c := range camps {
		rows[i] = campaigns.NewCampaignDTO(h.svc.RowOf(r.Context(), c))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (h *Handlers) Pause(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := currentUser(r)
	camp, err := h.svc.Pause(r.Context(), id, user.Name)
	if err != nil {
		if err == service.ErrValidation {
			httpx.WriteValidationError(w, "campaign must be live or paused to toggle", "")
			return
		}
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, campaigns.NewCampaignDTO(h.svc.RowOf(r.Context(), camp)))
}

type noteRequest struct {
	Text string `json:"text"`
}

func (h *Handlers) AddNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}
	user := currentUser(r)
	event, err := h.svc.AddNote(r.Context(), id, req.Text, user.Name)
	if err != nil {
		if err == service.ErrValidation {
			httpx.WriteValidationError(w, "text must not be empty", "text")
			return
		}
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, campaigns.NewAuditDTO(event))
}

func (h *Handlers) Anomalies(w http.ResponseWriter, r *http.Request) {
	camps, err := h.svc.Anomalies(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	rows := make([]campaigns.CampaignDTO, len(camps))
	for i, c := range camps {
		rows[i] = campaigns.NewCampaignDTO(h.svc.RowOf(r.Context(), c))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (h *Handlers) ExportCSV(w http.ResponseWriter, r *http.Request) {
	params := parseListParams(r)
	params.Page = 1
	params.Per = 100000
	result, err := h.svc.List(r.Context(), params)
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	body := service.ExportCSV(result.Items)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="campaigns.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// currentUser asserts the account the auth middleware loaded. platform
// stores it as `any` so it needn't know this service's user type.
func currentUser(r *http.Request) campaigns.User {
	v, _ := httpx.UserFromContext(r.Context())
	u, _ := v.(campaigns.User)
	return u
}
