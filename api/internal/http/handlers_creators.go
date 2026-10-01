package httpapi

import (
	"net/http"

	"campaigntrackerpro/internal/domain"

	"github.com/go-chi/chi/v5"
)

type CreatorDTO struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Role            string   `json:"role"`
	Initials        string   `json:"initials"`
	Category        string   `json:"category"`
	Region          string   `json:"region"`
	Tier            string   `json:"tier"`
	TierLabel       string   `json:"tierLabel"`
	Followers       int64    `json:"followers"`
	PrimaryPlatform string   `json:"primaryPlatform"`
	Languages       []string `json:"languages"`
}

type CreatorPerformanceDTO struct {
	CreatorDTO
	Campaigns        int     `json:"campaigns"`
	LiveCampaigns    int     `json:"liveCampaigns"`
	TotalReach       float64 `json:"totalReach"`
	TotalSpend       int64   `json:"totalSpend"`
	AvgReach         float64 `json:"avgReach"`
	TierMedian       float64 `json:"tierMedian"`
	TierIndex        float64 `json:"tierIndex"`
	CostPerLakh      float64 `json:"costPerLakh"`
	Consistency      float64 `json:"consistency"`
	AudienceReachPct float64 `json:"audienceReachPct"`
	Flagged          int     `json:"flagged"`
}

func creatorDTO(c domain.Creator) CreatorDTO {
	langs := c.Languages
	if langs == nil {
		langs = []string{}
	}
	return CreatorDTO{
		ID: c.ID, Name: c.Name, Role: c.Role, Initials: c.Initials,
		Category: c.Category, Region: c.Region, Tier: string(c.Tier),
		TierLabel: domain.TierLabel(c.Tier), Followers: c.Followers,
		PrimaryPlatform: c.PrimaryPlatform, Languages: langs,
	}
}

func creatorPerformanceDTO(p domain.CreatorPerformance) CreatorPerformanceDTO {
	return CreatorPerformanceDTO{
		CreatorDTO: creatorDTO(p.Creator), Campaigns: p.Campaigns, LiveCampaigns: p.LiveCampaigns,
		TotalReach: p.TotalReach, TotalSpend: p.TotalSpend, AvgReach: p.AvgReach,
		TierMedian: p.TierMedian, TierIndex: p.TierIndex, CostPerLakh: p.CostPerLakh,
		Consistency: p.Consistency, AudienceReachPct: p.AudienceReachPct, Flagged: p.Flagged,
	}
}

type LanguageReachDTO struct {
	Language string  `json:"language"`
	Reach    float64 `json:"reach"`
	Count    int     `json:"count"`
}

type CreatorDetailDTO struct {
	CreatorPerformanceDTO
	Campaigns         []CampaignDTO      `json:"campaignRows"`
	LanguageBreakdown []LanguageReachDTO `json:"languageBreakdown"`
	TierPeers         int                `json:"tierPeers"`
}

func (h *Handlers) ListCreators(w http.ResponseWriter, r *http.Request) {
	perf, err := h.Creators.List(r.Context())
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	items := make([]CreatorPerformanceDTO, len(perf))
	for i, p := range perf {
		items[i] = creatorPerformanceDTO(p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) GetCreator(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	d, err := h.Creators.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	langs := make([]LanguageReachDTO, len(d.LanguageBreakdown))
	for i, l := range d.LanguageBreakdown {
		langs[i] = LanguageReachDTO{Language: l.Language, Reach: l.Reach, Count: l.Count}
	}
	writeJSON(w, http.StatusOK, CreatorDetailDTO{
		CreatorPerformanceDTO: creatorPerformanceDTO(d.CreatorPerformance),
		Campaigns:             campaignDTOs(d.CampaignRows),
		LanguageBreakdown:     langs,
		TierPeers:             d.TierPeers,
	})
}
