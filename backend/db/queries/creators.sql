-- name: ListCreators :many
SELECT * FROM creators ORDER BY followers DESC;

-- name: GetCreator :one
SELECT * FROM creators WHERE id = $1;

-- name: CreateCreator :one
INSERT INTO creators (id, name, role, initials, category, region, tier, followers, primary_platform, languages)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListCampaignsByCreator :many
SELECT * FROM campaigns WHERE creator_id = $1 ORDER BY created_at DESC;

-- name: SetCampaignCreator :exec
UPDATE campaigns SET creator_id = $2, updated_at = now() WHERE id = $1;

-- SetCreatorPhoto attaches a creator's own photograph, or clears it when
-- the url is empty. The avatar takes over again when it is cleared.
-- name: SetCreatorPhoto :exec
UPDATE creators SET image_url = NULLIF($2, '') WHERE id = $1;
