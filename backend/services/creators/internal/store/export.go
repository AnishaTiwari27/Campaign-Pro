package store

import (
	"context"

	"campaigntrackerpro/db/gen"
	"campaigntrackerpro/platform/database"

	"github.com/jackc/pgx/v5/pgtype"
)

// RecordExport writes an audit row for a roster export.
//
// An export is the one action that takes data out of the app, past its
// access control and past every screen that qualified a figure. Once a
// file is on someone's laptop there is no further record of it, so the
// row written here is the only evidence that it ever happened — who
// took it, which columns, and how many creators.
//
// campaign_id is null: an export belongs to no campaign. The tenant is
// reachable through user_id, which is org-scoped, so the row stays
// attributable without audit_events needing its own org column.
func (s *Store) RecordExport(ctx context.Context, userID, actor, action string) error {
	var uid pgtype.UUID
	if userID != "" {
		if parsed, err := database.UuidParam(userID); err == nil {
			uid = parsed
		}
	}
	_, err := s.db.Queries.CreateAuditEvent(ctx, gen.CreateAuditEventParams{
		Actor:      actor,
		Action:     action,
		Kind:       gen.AuditKindT("user"),
		UserID:     uid,
		EntityType: "creator_export",
	})
	return err
}
