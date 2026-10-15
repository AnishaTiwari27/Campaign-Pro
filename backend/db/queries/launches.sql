-- name: UpsertLaunch :exec
INSERT INTO launches (
    id, org_id, source, external_id, name, tagline, url, category,
    launched_at, traction, has_site, readiness, basis
) VALUES (
    @id, @org_id, @source, @external_id, @name, @tagline, @url, @category,
    @launched_at, @traction, @has_site, @readiness, @basis
)
ON CONFLICT (org_id, source, external_id) DO UPDATE SET
    name      = EXCLUDED.name,
    tagline   = EXCLUDED.tagline,
    url       = EXCLUDED.url,
    category  = EXCLUDED.category,
    traction  = EXCLUDED.traction,
    has_site  = EXCLUDED.has_site,
    readiness = EXCLUDED.readiness,
    basis     = EXCLUDED.basis;
-- first_seen_at and the outcome columns are deliberately absent from
-- the update: re-polling a launch must not reset when we first saw it,
-- nor erase an outcome already recorded against it.

-- name: ListLaunches :many
SELECT id, source, external_id, name, tagline, url, category, launched_at,
       first_seen_at, traction, has_site, readiness, basis,
       first_campaign_at, days_to_market
FROM launches
WHERE (@org_system::bool OR org_id = @org_id::text)
ORDER BY launched_at DESC
LIMIT 200;

-- name: ListLaunchesAwaitingOutcome :many
SELECT id, name, launched_at FROM launches
WHERE (@org_system::bool OR org_id = @org_id::text)
  AND first_campaign_at IS NULL;

-- name: RecordLaunchOutcome :exec
UPDATE launches
SET first_campaign_at = @first_campaign_at,
    days_to_market = @days_to_market
WHERE id = @id AND (@org_system::bool OR org_id = @org_id::text);
