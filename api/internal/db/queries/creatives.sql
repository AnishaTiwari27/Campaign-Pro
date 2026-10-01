-- name: ListCreativesByCampaign :many
SELECT * FROM creatives WHERE campaign_id = $1 ORDER BY created_at ASC;

-- name: CreateCreative :one
INSERT INTO creatives (campaign_id, headline, kind, duration_label, reach, ctr)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: CountCreativesByCampaign :one
SELECT count(*) FROM creatives WHERE campaign_id = $1;
