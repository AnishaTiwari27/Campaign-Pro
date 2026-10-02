-- name: ListAuditEventsByCampaign :many
SELECT * FROM audit_events WHERE campaign_id = sqlc.narg('campaign_id')::text ORDER BY created_at DESC;

-- name: CreateAuditEvent :one
INSERT INTO audit_events (campaign_id, actor, action, kind, user_id, entity_type, entity_id)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;
