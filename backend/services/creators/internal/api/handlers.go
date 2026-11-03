package api

import (
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/creators"
	"campaigntrackerpro/services/creators/internal/service"

	"github.com/go-chi/chi/v5"
)

// Handlers is the creators transport; it owns the /creators URL space.
type Handlers struct {
	svc    *service.Creators
	logger *slog.Logger
}

func New(svc *service.Creators, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

func (h *Handlers) Routes(r chi.Router) {
	r.Route("/creators", func(r chi.Router) {
		r.Get("/", h.ListCreators)
		// Before /{id}, or chi matches "export.csv" as a creator id.
		r.Get("/export.csv", h.ExportCreators)
		r.Get("/{id}", h.GetCreator)
	})
}

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
	// Empty when the agency has no photograph it may publish, which is the
	// normal case; the client then draws its generated avatar.
	ImageURL string `json:"imageUrl,omitempty"`
}

type CreatorPerformanceDTO struct {
	CreatorDTO
	Campaigns        int        `json:"campaigns"`
	LiveCampaigns    int        `json:"liveCampaigns"`
	TotalReach       float64    `json:"totalReach"`
	TotalSpend       int64      `json:"totalSpend"`
	AvgReach         float64    `json:"avgReach"`
	TierMedian       float64    `json:"tierMedian"`
	TierIndex        float64    `json:"tierIndex"`
	CostPerLakh      float64    `json:"costPerLakh"`
	Consistency      float64    `json:"consistency"`
	ConsistencyN     int        `json:"consistencyN"`
	AudienceReachPct float64    `json:"audienceReachPct"`
	TierCostMedian   float64    `json:"tierCostMedian"`
	CostIndex        float64    `json:"costIndex"`
	TierCostPeers    int        `json:"tierCostPeers"`
	Signals          SignalsDTO `json:"signals"`
	Flagged          int        `json:"flagged"`
}

func creatorDTO(c creators.Creator) CreatorDTO {
	langs := c.Languages
	if langs == nil {
		langs = []string{}
	}
	return CreatorDTO{
		ID: c.ID, Name: c.Name, Role: c.Role, Initials: c.Initials,
		Category: c.Category, Region: c.Region, Tier: string(c.Tier),
		TierLabel: creators.TierLabel(c.Tier), Followers: c.Followers,
		PrimaryPlatform: c.PrimaryPlatform, Languages: langs,
		ImageURL: c.ImageURL,
	}
}

func creatorPerformanceDTO(p creators.CreatorPerformance) CreatorPerformanceDTO {
	return CreatorPerformanceDTO{
		CreatorDTO: creatorDTO(p.Creator), Campaigns: p.Campaigns, LiveCampaigns: p.LiveCampaigns,
		TotalReach: p.TotalReach, TotalSpend: p.TotalSpend, AvgReach: p.AvgReach,
		TierMedian: p.TierMedian, TierIndex: p.TierIndex, CostPerLakh: p.CostPerLakh,
		Consistency: p.Consistency, ConsistencyN: p.ConsistencyN, AudienceReachPct: p.AudienceReachPct, Flagged: p.Flagged,
		TierCostMedian: p.TierCostMedian, CostIndex: p.CostIndex, TierCostPeers: p.TierCostPeers,
		Signals: signalsDTO(p.Signals),
	}
}

// SignalsDTO is the platform-derived half of a roster row.
//
// scanned travels to the client because every other field here is
// meaningless without it: a brandPosts of 0 on an unscanned creator is
// our own backlog, and the UI has to be able to tell the two apart.
type SignalsDTO struct {
	Scanned        bool    `json:"scanned"`
	Stage          string  `json:"stage"`
	StageLabel     string  `json:"stageLabel"`
	BrandPosts     int     `json:"brandPosts"`
	EngagementRate float64 `json:"engagementRate"`
	HasRate        bool    `json:"hasRate"`
	PostsScanned   int     `json:"postsScanned"`
	// Empty where never scanned, rather than a zero date the client
	// would have to know to special-case.
	ScannedAt string `json:"scannedAt,omitempty"`
}

func signalsDTO(s creators.Signals) SignalsDTO {
	d := SignalsDTO{
		Scanned: s.Scanned, Stage: string(s.Stage()), StageLabel: creators.StageLabel(s.Stage()),
		BrandPosts: s.BrandPosts, EngagementRate: s.EngagementRate,
		HasRate: s.HasRate, PostsScanned: s.PostsScanned,
	}
	if !s.ScanAt.IsZero() {
		d.ScannedAt = s.ScanAt.Format("2006-01-02")
	}
	return d
}

type LanguageReachDTO struct {
	Language string  `json:"language"`
	Reach    float64 `json:"reach"`
	Count    int     `json:"count"`
}

type CreatorDetailDTO struct {
	CreatorPerformanceDTO
	Campaigns         []campaigns.CampaignDTO `json:"campaignRows"`
	LanguageBreakdown []LanguageReachDTO      `json:"languageBreakdown"`
	TierPeers         int                     `json:"tierPeers"`
}

func (h *Handlers) ListCreators(w http.ResponseWriter, r *http.Request) {
	perf, err := h.svc.List(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	items := make([]CreatorPerformanceDTO, len(perf))
	for i, p := range perf {
		items[i] = creatorPerformanceDTO(p)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) GetCreator(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	d, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	langs := make([]LanguageReachDTO, len(d.LanguageBreakdown))
	for i, l := range d.LanguageBreakdown {
		langs[i] = LanguageReachDTO{Language: l.Language, Reach: l.Reach, Count: l.Count}
	}
	httpx.WriteJSON(w, http.StatusOK, CreatorDetailDTO{
		CreatorPerformanceDTO: creatorPerformanceDTO(d.CreatorPerformance),
		Campaigns:             campaigns.NewCampaignDTOs(d.CampaignRows),
		LanguageBreakdown:     langs,
		TierPeers:             d.TierPeers,
	})
}
