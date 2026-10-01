-- name: ListCreativesByCampaign :many
SELECT * FROM creatives WHERE campaign_id = $1 ORDER BY created_at ASC;

-- name: ListAllCreatives :many
SELECT * FROM creatives;

-- name: CreateCreative :one
INSERT INTO creatives (campaign_id, headline, kind, duration_label, reach, ctr, language, hook_type, claim, festival, analyzed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING *;

-- name: CountCreativesByCampaign :one
SELECT count(*) FROM creatives WHERE campaign_id = $1;

-- UpdateCreativeAnalysis is what the CreativeAnalyzer writes back after a
-- multimodal pass over the asset.
-- name: UpdateCreativeAnalysis :one
UPDATE creatives SET
    language   = coalesce(sqlc.narg('language')::text, language),
    hook_type  = coalesce(sqlc.narg('hook_type')::hook_type_t, hook_type),
    claim      = coalesce(sqlc.narg('claim')::text, claim),
    festival   = coalesce(sqlc.narg('festival')::text, festival),
    analyzed_at = now()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: ListUnanalyzedCreatives :many
SELECT * FROM creatives WHERE analyzed_at IS NULL LIMIT $1;
