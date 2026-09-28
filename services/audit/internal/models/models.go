// Package models holds audit-service's event-decode and JSON-facing types.
package models

import "time"

// AuditEvent is the payload shape published to NATS' "campaign.audit"
// subject by campaigns-service (services/campaigns/internal/api's
// auditPublish). Defined independently here, not shared via a Go import —
// same "duplicate the wire shape per service" convention this codebase
// already uses for campaign.created (analytics-service's own models
// mirror campaigns-service's aggregate shapes rather than importing
// them): services are meant to be independently deployable, not coupled
// through shared internal packages across a module boundary.
type AuditEvent struct {
	Action     string `json:"action"`
	ActorID    string `json:"actorId"`
	ActorEmail string `json:"actorEmail"`
	ActorRole  string `json:"actorRole"`
	CampaignID int64  `json:"campaignId"`
	Details    string `json:"details"`
}

// Entry is one stored, immutable row of audit.audit_log.
type Entry struct {
	ID         int64
	Action     string
	ActorID    string
	ActorEmail string
	ActorRole  string
	CampaignID *int64 // nullable — not every future audit action need be campaign-scoped
	Details    string
	CreatedAt  time.Time
}

// EntryJSON mirrors Entry with the date formatted — same
// struct/AsJSON split every other service's JSON view types use.
type EntryJSON struct {
	ID         int64  `json:"id"`
	Action     string `json:"action"`
	ActorID    string `json:"actorId"`
	ActorEmail string `json:"actorEmail"`
	ActorRole  string `json:"actorRole"`
	CampaignID *int64 `json:"campaignId,omitempty"`
	Details    string `json:"details"`
	CreatedAt  string `json:"createdAt"`
}

func (e Entry) AsJSON() EntryJSON {
	return EntryJSON{
		ID: e.ID, Action: e.Action, ActorID: e.ActorID, ActorEmail: e.ActorEmail, ActorRole: e.ActorRole,
		CampaignID: e.CampaignID, Details: e.Details, CreatedAt: e.CreatedAt.Format(time.RFC3339),
	}
}
