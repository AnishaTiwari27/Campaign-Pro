package store

import (
	"context"

	"campaigntrackerpro/internal/db/gen"
	"campaigntrackerpro/internal/domain"
)

func (s *Store) ListAuditEventsByCampaign(ctx context.Context, campaignID string) ([]domain.AuditEvent, error) {
	es, err := s.Queries.ListAuditEventsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	return toDomainAuditEvents(es), nil
}

// CreateAuditEvent writes one audit row. Every approve, reject, reopen,
// pause, resume, note and anomaly flag goes through this so the Activity
// tab and the audit trail are always the same source of truth.
func (s *Store) CreateAuditEvent(ctx context.Context, campaignID, actor, action, kind string) (domain.AuditEvent, error) {
	e, err := s.Queries.CreateAuditEvent(ctx, gen.CreateAuditEventParams{
		CampaignID: campaignID, Actor: actor, Action: action, Kind: gen.AuditKindT(kind),
	})
	if err != nil {
		return domain.AuditEvent{}, err
	}
	return toDomainAuditEvent(e), nil
}
