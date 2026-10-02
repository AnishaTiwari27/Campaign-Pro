// The published JSON shapes for a campaign. These are part of the public
// contract because other services embed campaign data in their own
// responses — the overview's ribbons, a region's campaign list, a
// creator's roster — and none of them can import internal/.
package campaigns

import "time"

const rfc3339 = time.RFC3339

type CampaignDTO struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	SubjectType   string  `json:"subjectType"`
	Role          string  `json:"role,omitempty"`
	Initials      string  `json:"initials"`
	Category      string  `json:"category"`
	Region        string  `json:"region"`
	AdType        string  `json:"adType"`
	Platform      string  `json:"platform"`
	Status        string  `json:"status"`
	DaysRunning   int     `json:"daysRunning"`
	Reach         float64 `json:"reach"`
	Spend         int64   `json:"spend"`
	Budget        int64   `json:"budget"`
	Frequency     float64 `json:"frequency"`
	Approval      string  `json:"approval"`
	CurveShape    string  `json:"curveShape"`
	FlagReason    string  `json:"flagReason,omitempty"`
	BrandDomain   string  `json:"brandDomain,omitempty"`
	CreatorID     string  `json:"creatorId,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
	Pace          float64 `json:"pace"`
	PaceClass     string  `json:"paceClass"`
	CPM           float64 `json:"cpm"`
	Index         float64 `json:"index"`
	CategoryIndex float64 `json:"categoryIndex"`
}

func NewCampaignDTO(r CampaignRow) CampaignDTO {
	return CampaignDTO{
		ID: r.ID, Name: r.Name, SubjectType: string(r.SubjectType), Role: r.Role,
		Initials: r.Initials, Category: r.Category, Region: r.Region, AdType: string(r.AdType),
		Platform: r.Platform, Status: string(r.Status), DaysRunning: r.DaysRunning,
		Reach: r.Reach, Spend: r.Spend, Budget: r.Budget, Frequency: r.Frequency,
		Approval: string(r.Approval), CurveShape: string(r.CurveShape), FlagReason: r.FlagReason,
		BrandDomain: r.BrandDomain, CreatorID: r.CreatorID,
		CreatedAt: r.CreatedAt.Format(rfc3339), UpdatedAt: r.UpdatedAt.Format(rfc3339),
		Pace: r.Pace, PaceClass: string(r.PaceClass), CPM: r.CPM, Index: r.Index,
		CategoryIndex: r.CategoryIndex,
	}
}

func NewCampaignDTOs(rows []CampaignRow) []CampaignDTO {
	out := make([]CampaignDTO, len(rows))
	for i, r := range rows {
		out[i] = NewCampaignDTO(r)
	}
	return out
}

type CreativeDTO struct {
	ID            string  `json:"id"`
	Headline      string  `json:"headline"`
	Kind          string  `json:"kind"`
	DurationLabel string  `json:"durationLabel"`
	Reach         float64 `json:"reach"`
	CTR           float64 `json:"ctr"`
}

func NewCreativeDTO(c Creative) CreativeDTO {
	return CreativeDTO{ID: c.ID, Headline: c.Headline, Kind: c.Kind, DurationLabel: c.DurationLabel, Reach: c.Reach, CTR: c.CTR}
}

func NewCreativeDTOs(cs []Creative) []CreativeDTO {
	out := make([]CreativeDTO, len(cs))
	for i, c := range cs {
		out[i] = NewCreativeDTO(c)
	}
	return out
}

type AuditEventDTO struct {
	ID        int64  `json:"id"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"createdAt"`
}

func NewAuditDTO(e AuditEvent) AuditEventDTO {
	return AuditEventDTO{ID: e.ID, Actor: e.Actor, Action: e.Action, Kind: e.Kind, CreatedAt: e.CreatedAt.Format(rfc3339)}
}

func NewAuditDTOs(es []AuditEvent) []AuditEventDTO {
	out := make([]AuditEventDTO, len(es))
	for i, e := range es {
		out[i] = NewAuditDTO(e)
	}
	return out
}

type ListResponse struct {
	Items []CampaignDTO `json:"items"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Pages int           `json:"pages"`
}

func NewListResponse(r ListResult) ListResponse {
	return ListResponse{Items: NewCampaignDTOs(r.Items), Total: r.Total, Page: r.Page, Pages: r.Pages}
}
