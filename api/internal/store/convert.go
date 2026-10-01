package store

import (
	"encoding/json"
	"time"

	"campaigntrackerpro/internal/db/gen"
	"campaigntrackerpro/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func textOrEmpty(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func textParam(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func pgxBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}

func timeOf(t pgtype.Timestamptz) time.Time {
	return t.Time
}

func uuidToString(id pgtype.UUID) string {
	u := uuid.UUID(id.Bytes)
	return u.String()
}

func uuidParam(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func toDomainCampaign(c gen.Campaign) domain.Campaign {
	return domain.Campaign{
		ID:          c.ID,
		Name:        c.Name,
		SubjectType: domain.SubjectType(c.SubjectType),
		Role:        textOrEmpty(c.Role),
		Initials:    c.Initials,
		Category:    c.Category,
		Region:      c.Region,
		AdType:      domain.AdType(c.AdType),
		Platform:    c.Platform,
		Status:      domain.Status(c.Status),
		DaysRunning: int(c.DaysRunning),
		Reach:       c.Reach,
		Spend:       c.Spend,
		Budget:      c.Budget,
		Frequency:   c.Frequency,
		Approval:    domain.Approval(c.Approval),
		CurveShape:  domain.CurveShape(c.CurveShape),
		FlagReason:  textOrEmpty(c.FlagReason),
		CreatorID:   textOrEmpty(c.CreatorID),
		BrandDomain: textOrEmpty(c.BrandDomain),
		CreatedAt:   timeOf(c.CreatedAt),
		UpdatedAt:   timeOf(c.UpdatedAt),
	}
}

func toDomainCampaigns(cs []gen.Campaign) []domain.Campaign {
	out := make([]domain.Campaign, len(cs))
	for i, c := range cs {
		out[i] = toDomainCampaign(c)
	}
	return out
}

func toDomainCreative(c gen.Creative) domain.Creative {
	return domain.Creative{
		ID:            uuidToString(c.ID),
		CampaignID:    c.CampaignID,
		Headline:      c.Headline,
		Kind:          string(c.Kind),
		DurationLabel: c.DurationLabel,
		Reach:         c.Reach,
		CTR:           c.Ctr,
		Language:      textOrEmpty(c.Language),
		HookType:      string(c.HookType.HookTypeT),
		Claim:         textOrEmpty(c.Claim),
		Festival:      textOrEmpty(c.Festival),
		AnalyzedAt:    timeOf(c.AnalyzedAt),
		CreatedAt:     timeOf(c.CreatedAt),
	}
}

func toDomainCreatives(cs []gen.Creative) []domain.Creative {
	out := make([]domain.Creative, len(cs))
	for i, c := range cs {
		out[i] = toDomainCreative(c)
	}
	return out
}

func toDomainAuditEvent(e gen.AuditEvent) domain.AuditEvent {
	return domain.AuditEvent{
		ID:         e.ID,
		CampaignID: e.CampaignID,
		Actor:      e.Actor,
		Action:     e.Action,
		Kind:       string(e.Kind),
		CreatedAt:  timeOf(e.CreatedAt),
	}
}

func toDomainAuditEvents(es []gen.AuditEvent) []domain.AuditEvent {
	out := make([]domain.AuditEvent, len(es))
	for i, e := range es {
		out[i] = toDomainAuditEvent(e)
	}
	return out
}

func toDomainReport(r gen.Report) domain.Report {
	var scope map[string]any
	_ = json.Unmarshal(r.ScopeFilters, &scope)
	return domain.Report{
		ID:           uuidToString(r.ID),
		Name:         r.Name,
		Enabled:      r.Enabled,
		Cadence:      string(r.Cadence),
		Recipients:   r.Recipients,
		ScopeFilters: scope,
		ScopeLabel:   r.ScopeLabel,
		Columns:      r.Columns,
		CreatedAt:    timeOf(r.CreatedAt),
		UpdatedAt:    timeOf(r.UpdatedAt),
	}
}

func toDomainReports(rs []gen.Report) []domain.Report {
	out := make([]domain.Report, len(rs))
	for i, r := range rs {
		out[i] = toDomainReport(r)
	}
	return out
}

func toDomainReportRun(r gen.ReportRun) domain.ReportRun {
	return domain.ReportRun{
		ID:       r.ID,
		ReportID: uuidToString(r.ReportID),
		RanAt:    timeOf(r.RanAt),
		Result:   string(r.Result),
		RowCount: int(r.RowCount),
	}
}

func toDomainReportRuns(rs []gen.ReportRun) []domain.ReportRun {
	out := make([]domain.ReportRun, len(rs))
	for i, r := range rs {
		out[i] = toDomainReportRun(r)
	}
	return out
}

func hookTypeParam(s string) gen.NullHookTypeT {
	if s == "" {
		return gen.NullHookTypeT{}
	}
	return gen.NullHookTypeT{HookTypeT: gen.HookTypeT(s), Valid: true}
}

func timestampParam(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}
