-- The organisation predicate on every statement below is the tenant
-- boundary. @org_system is true only for callers that are not requests —
-- the report worker, the seeder, the CLI — which legitimately span
-- organisations. For a request it is false, and org_id is the signed-in
-- user's; an unresolved organisation is the empty string, which matches
-- nothing, so the failure mode of forgetting it is a blank page rather
-- than another tenant's data.

-- name: GetCampaign :one
SELECT * FROM campaigns WHERE id = @id AND (@org_system::bool OR org_id = @org_id::text);

-- name: ListCampaigns :many
SELECT * FROM campaigns WHERE (@org_system::bool OR org_id = @org_id::text) ORDER BY created_at DESC;

-- ListCampaignsFiltered applies every /campaigns filter in one query;
-- sort, pagination and derived fields (pace, cpm, index, trend) are done
-- in Go since sqlc can't parameterize ORDER BY columns.
-- name: ListCampaignsFiltered :many
SELECT * FROM campaigns
WHERE (@org_system::bool OR org_id = @org_id::text)
  AND (sqlc.narg('subject_type')::subject_type_t IS NULL OR subject_type = sqlc.narg('subject_type')::subject_type_t)
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
SELECT * FROM campaigns WHERE status != 'scheduled' AND (@org_system::bool OR org_id = @org_id::text) ORDER BY created_at DESC;

-- name: ListPendingCampaigns :many
SELECT * FROM campaigns WHERE approval = 'pending' AND (@org_system::bool OR org_id = @org_id::text) ORDER BY spend DESC;

-- name: ListFlaggedCampaigns :many
SELECT * FROM campaigns WHERE flag_reason IS NOT NULL AND (@org_system::bool OR org_id = @org_id::text) ORDER BY updated_at DESC;

-- name: CreateCampaign :one
INSERT INTO campaigns (
    id, name, subject_type, role, initials, category, region, ad_type, platform,
    status, days_running, reach, spend, budget, frequency, approval, curve_shape, flag_reason,
    brand_domain, flight_days, org_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
    @org_id
) RETURNING *;

-- name: UpdateCampaignDecision :one
UPDATE campaigns SET approval = @approval, updated_at = now()
WHERE id = @id AND (@org_system::bool OR org_id = @org_id::text) RETURNING *;

-- name: BulkUpdateApproval :many
UPDATE campaigns SET approval = sqlc.arg('approval')::approval_status_t, updated_at = now()
WHERE id = ANY(sqlc.arg('ids')::text[]) AND (@org_system::bool OR org_id = @org_id::text) RETURNING *;

-- name: UpdateCampaignStatus :one
UPDATE campaigns SET status = @status, updated_at = now()
WHERE id = @id AND (@org_system::bool OR org_id = @org_id::text) RETURNING *;

-- name: UpdateCampaignFlag :one
UPDATE campaigns SET flag_reason = @flag_reason, updated_at = now()
WHERE id = @id AND (@org_system::bool OR org_id = @org_id::text) RETURNING *;

-- The scope predicate below is the one every aggregate has to carry.
-- Row reads get their visibility applied in Go after loading, but an
-- aggregate has no rows left to filter by the time it returns a number,
-- so the restriction has to be in the WHERE clause or the figure is the
-- whole agency's. @restricted false means agency staff, who see
-- everything; true with an empty list means a client whose grants are
-- not set yet, who must see nothing rather than everything.

-- name: CountPendingApprovals :one
SELECT count(*) FROM campaigns
WHERE approval = 'pending'
  AND (@org_system::bool OR org_id = @org_id::text)
  AND (NOT @restricted::bool OR id = ANY(@visible_ids::text[]));

-- name: TotalPendingSpend :one
SELECT coalesce(sum(spend), 0)::bigint FROM campaigns
WHERE approval = 'pending'
  AND (@org_system::bool OR org_id = @org_id::text)
  AND (NOT @restricted::bool OR id = ANY(@visible_ids::text[]));

-- OverviewTotals replaces loading every campaign and summing it in Go.
-- One pass in the database instead of one row per campaign over the
-- wire, decoded into a slice, walked once. At the current fixture size
-- that is a wash; it is the shape that stops the overview getting
-- slower in proportion to the book of business.
-- name: OverviewTotals :one
SELECT
    count(*) FILTER (WHERE status = 'live')::bigint AS live_count,
    coalesce(sum(reach) FILTER (WHERE status = 'live'), 0)::float8 AS reach_live,
    coalesce(sum(spend), 0)::bigint AS spend_window,
    coalesce(sum(budget) FILTER (WHERE approval = 'approved'), 0)::bigint AS approved_budget
FROM campaigns
WHERE (@org_system::bool OR org_id = @org_id::text)
  AND (NOT @restricted::bool OR id = ANY(@visible_ids::text[]));
