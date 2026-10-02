package store

import (
	"context"

	"campaigntrackerpro/db/gen"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/campaigns"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) ListAuditEventsByCampaign(ctx context.Context, campaignID string) ([]campaigns.AuditEvent, error) {
	es, err := s.db.Queries.ListAuditEventsByCampaign(ctx, database.TextParam(campaignID))
	if err != nil {
		return nil, err
	}
	return toDomainAuditEvents(es), nil
}

// CreateAuditEvent writes one audit row. Every approve, reject, reopen,
// pause, resume, note and anomaly flag goes through this so the Activity
// tab and the audit trail are always the same source of truth.
//
// userID is what makes "who approved this?" answerable, since actor is only
// a display label — it is a parameter rather than a default so a caller
// cannot write an unattributed decision by omission. Pass "" only for the
// system actor (anomaly detection), which is no user.
func (s *Store) CreateAuditEvent(ctx context.Context, campaignID, userID, actor, action, kind string) (campaigns.AuditEvent, error) {
	return s.RecordEvent(ctx, campaignID, userID, actor, action, kind, "campaign", campaignID)
}

// RecordEvent is the general form: any entity, attributed to a user id so
// "who approved this?" is answerable. actor stays the display label
// because names change and a trail should not rewrite history.
func (s *Store) RecordEvent(ctx context.Context, campaignID, userID, actor, action, kind, entityType, entityID string) (campaigns.AuditEvent, error) {
	var uid pgtype.UUID
	if userID != "" {
		parsed, err := database.UuidParam(userID)
		if err == nil {
			uid = parsed
		}
	}
	e, err := s.db.Queries.CreateAuditEvent(ctx, gen.CreateAuditEventParams{
		CampaignID: database.TextParam(campaignID),
		Actor:      actor,
		Action:     action,
		Kind:       gen.AuditKindT(kind),
		UserID:     uid,
		EntityType: entityType,
		EntityID:   database.TextParam(entityID),
	})
	if err != nil {
		return campaigns.AuditEvent{}, err
	}
	return toDomainAuditEvent(e), nil
}
