-- name: GetUser :one
SELECT * FROM users WHERE id = @id AND (@org_system::bool OR org_id = @org_id::text);

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users WHERE (@org_system::bool OR org_id = @org_id::text) ORDER BY created_at ASC;

-- name: CreateUser :one
INSERT INTO users (email, name, role, can_approve, org_id)
VALUES ($1, $2, $3, $4, @org_id) RETURNING *;

-- name: ListGrantsForUser :many
SELECT campaign_id FROM client_campaign_grants WHERE user_id = $1 ORDER BY campaign_id;

-- name: DeleteGrantsForUser :exec
DELETE FROM client_campaign_grants WHERE user_id = $1;

-- name: GrantCampaign :exec
INSERT INTO client_campaign_grants (user_id, campaign_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;
