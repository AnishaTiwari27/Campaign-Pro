package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/creators"

	"github.com/go-chi/chi/v5"
)

// CandidateDTO is a discovered channel awaiting a decision.
type CandidateDTO struct {
	ID          string `json:"id"`
	Platform    string `json:"platform"`
	ExternalID  string `json:"externalId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Followers   int64  `json:"followers"`
	// HasFollowers is false where the channel hides its count. The
	// client has to show "hidden" rather than "0", and has to ask for
	// a figure before this one can be promoted.
	HasFollowers bool   `json:"hasFollowers"`
	Videos       int64  `json:"videos"`
	Views        int64  `json:"views"`
	FoundFor     string `json:"foundFor"`
	// URL is built here rather than in the client: the id-to-url shape
	// belongs with the platform that issued the id.
	URL string `json:"url"`
}

func candidateDTO(c creators.Candidate) CandidateDTO {
	d := CandidateDTO{
		ID: c.ID, Platform: c.Platform, ExternalID: c.ExternalID,
		Name: c.Name, Description: c.Description,
		Followers: c.Followers, HasFollowers: c.HasFollowers,
		Videos: c.Videos, Views: c.Views, FoundFor: c.FoundFor,
	}
	if c.Platform == "youtube" && c.ExternalID != "" {
		d.URL = "https://www.youtube.com/channel/" + c.ExternalID
	}
	return d
}

func (h *Handlers) ListCandidates(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	cands, err := h.svc.NewCandidates(r.Context(), limit)
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	items := make([]CandidateDTO, len(cands))
	for i, c := range cands {
		items[i] = candidateDTO(c)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

type promoteRequest struct {
	Role      string   `json:"role"`
	Category  string   `json:"category"`
	Region    string   `json:"region"`
	Languages []string `json:"languages"`
	Followers int64    `json:"followers"`
}

// PromoteCandidate is the only door into the roster, and a person is
// always on the other side of it.
func (h *Handlers) PromoteCandidate(w http.ResponseWriter, r *http.Request) {
	var req promoteRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "malformed request", "")
		return
	}
	created, err := h.svc.Promote(r.Context(), chi.URLParam(r, "id"), creators.Promotion{
		Role: req.Role, Category: req.Category, Region: req.Region,
		Languages: req.Languages, Followers: req.Followers,
	})
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, creatorDTO(created))
}

func (h *Handlers) RejectCandidate(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Reject(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
