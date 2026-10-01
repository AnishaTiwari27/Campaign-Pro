-- name: GetCampaign :one
SELECT * FROM campaigns WHERE id = $1;

-- name: ListCampaigns :many
SELECT * FROM campaigns ORDER BY created_at DESC;

-- ListCampaignsFiltered applies every /campaigns filter in one query;
-- sort, pagination and derived fields (pace, cpm, index, trend) are done
-- in Go since sqlc can't parameterize ORDER BY columns.
-- name: ListCampaignsFiltered :many
SELECT * FROM campaigns
WHERE (sqlc.narg('subject_type')::subject_type_t IS NULL OR subject_type = sqlc.narg('subject_type')::subject_type_t)
  AND (sqlc.narg('category')::text IS NULL OR category = sqlc.narg('category')::text)
  AND (sqlc.narg('region')::text IS NULL OR region = sqlc.narg('region')::text)
  AND (sqlc.narg('ad_type')::ad_type_t IS NULL OR ad_type = sqlc.narg('ad_type')::ad_type_t)
  AND (sqlc.narg('status')::campaign_status_t IS NULL OR status = sqlc.narg('status')::campaign_status_t)
  AND (sqlc.narg('approval')::approval_status_t IS NULL OR approval = sqlc.narg('approval')::approval_status_t)
  AND (
    sqlc.narg('search')::text IS NULL
    OR name ILIKE '%' || sqlc.narg('search')::text || '%'
    OR category ILIKE '%' || sqlc.narg('search')::text || '%'
    OR region ILIKE '%' || sqlc.narg('search')::text || '%'
    OR ad_type::text ILIKE '%' || sqlc.narg('search')::text || '%'
    OR coalesce(role, '') ILIKE '%' || sqlc.narg('search')::text || '%'
  );

-- name: ListRunningCampaigns :many
SELECT * FROM campaigns WHERE status != 'scheduled' ORDER BY created_at DESC;

-- name: ListPendingCampaigns :many
SELECT * FROM campaigns WHERE approval = 'pending' ORDER BY spend DESC;

-- name: ListFlaggedCampaigns :many
SELECT * FROM campaigns WHERE flag_reason IS NOT NULL ORDER BY updated_at DESC;

-- name: CreateCampaign :one
INSERT INTO campaigns (
    id, name, subject_type, role, initials, category, region, ad_type, platform,
    status, days_running, reach, spend, budget, frequency, approval, curve_shape, flag_reason,
    brand_domain
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
) RETURNING *;

-- name: UpdateCampaignDecision :one
UPDATE campaigns SET approval = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: BulkUpdateApproval :many
UPDATE campaigns SET approval = sqlc.arg('approval')::approval_status_t, updated_at = now()
WHERE id = ANY(sqlc.arg('ids')::text[]) RETURNING *;

-- name: UpdateCampaignStatus :one
UPDATE campaigns SET status = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: UpdateCampaignFlag :one
UPDATE campaigns SET flag_reason = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: CountPendingApprovals :one
SELECT count(*) FROM campaigns WHERE approval = 'pending';

-- name: TotalPendingSpend :one
SELECT coalesce(sum(spend), 0)::bigint FROM campaigns WHERE approval = 'pending';
