-- name: ListAuditEventsByCampaign :many
SELECT * FROM audit_events WHERE campaign_id = $1 ORDER BY created_at DESC;

-- name: CreateAuditEvent :one
INSERT INTO audit_events (campaign_id, actor, action, kind)
VALUES ($1, $2, $3, $4) RETURNING *;
