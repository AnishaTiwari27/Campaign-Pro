package store

import (
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/campaigns"

	"campaigntrackerpro/db/gen"
)

func toDomainCampaign(c gen.Campaign) campaigns.Campaign {
	return campaigns.Campaign{
		ID:          c.ID,
		Name:        c.Name,
		SubjectType: campaigns.SubjectType(c.SubjectType),
		Role:        database.TextOrEmpty(c.Role),
		Initials:    c.Initials,
		Category:    c.Category,
		Region:      c.Region,
		AdType:      campaigns.AdType(c.AdType),
		Platform:    c.Platform,
		Status:      campaigns.Status(c.Status),
		DaysRunning: int(c.DaysRunning),
		Reach:       c.Reach,
		Spend:       c.Spend,
		Budget:      c.Budget,
		Frequency:   c.Frequency,
		Approval:    campaigns.Approval(c.Approval),
		CurveShape:  campaigns.CurveShape(c.CurveShape),
		FlagReason:  database.TextOrEmpty(c.FlagReason),
		CreatorID:   database.TextOrEmpty(c.CreatorID),
		BrandDomain: database.TextOrEmpty(c.BrandDomain),
		CreatedAt:   database.TimeOf(c.CreatedAt),
		UpdatedAt:   database.TimeOf(c.UpdatedAt),
	}
}

func toDomainCampaigns(cs []gen.Campaign) []campaigns.Campaign {
	out := make([]campaigns.Campaign, len(cs))
	for i, c := range cs {
		out[i] = toDomainCampaign(c)
	}
	return out
}

func toDomainCreative(c gen.Creative) campaigns.Creative {
	return campaigns.Creative{
		ID:            database.UuidToString(c.ID),
		CampaignID:    c.CampaignID,
		Headline:      c.Headline,
		Kind:          string(c.Kind),
		DurationLabel: c.DurationLabel,
		Reach:         c.Reach,
		CTR:           c.Ctr,
		Language:      database.TextOrEmpty(c.Language),
		HookType:      string(c.HookType.HookTypeT),
		Claim:         database.TextOrEmpty(c.Claim),
		Festival:      database.TextOrEmpty(c.Festival),
		AnalyzedAt:    database.TimeOf(c.AnalyzedAt),
		CreatedAt:     database.TimeOf(c.CreatedAt),
	}
}

func toDomainCreatives(cs []gen.Creative) []campaigns.Creative {
	out := make([]campaigns.Creative, len(cs))
	for i, c := range cs {
		out[i] = toDomainCreative(c)
	}
	return out
}

func toDomainAuditEvent(e gen.AuditEvent) campaigns.AuditEvent {
	return campaigns.AuditEvent{
		ID:         e.ID,
		CampaignID: database.TextOrEmpty(e.CampaignID),
		Actor:      e.Actor,
		Action:     e.Action,
		Kind:       string(e.Kind),
		CreatedAt:  database.TimeOf(e.CreatedAt),
	}
}

func toDomainAuditEvents(es []gen.AuditEvent) []campaigns.AuditEvent {
	out := make([]campaigns.AuditEvent, len(es))
	for i, e := range es {
		out[i] = toDomainAuditEvent(e)
	}
	return out
}
