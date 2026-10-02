package main

import (
	"context"

	campaignsmod "campaigntrackerpro/services/campaigns/module"
)

// auditBridge lets identity record sign-in and sign-out without importing
// the service that owns the audit table. Identity declares the capability;
// this satisfies it.
type auditBridge struct {
	campaigns *campaignsmod.Module
}

func (a auditBridge) Record(ctx context.Context, userID, actor, action, entityType, entityID string) error {
	_, err := a.campaigns.Store().RecordEvent(ctx, "", userID, actor, action, "user", entityType, entityID)
	return err
}
