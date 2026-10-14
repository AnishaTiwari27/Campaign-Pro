-- name: UpsertExternalCampaign :one
-- The platform's own id is the match key. Names get edited and budgets
-- change, so matching on either would create a duplicate the first time
-- somebody renamed a campaign in the ad account.
INSERT INTO campaigns (
    id, name, subject_type, initials, category, region, ad_type, platform,
    status, days_running, reach, spend, budget, frequency, approval, curve_shape,
    org_id, external_source, external_id
) VALUES (
    @id, @name, @subject_type, @initials, @category, @region, @ad_type, @platform,
    @status, @days_running, @reach, @spend, @budget, @frequency, @approval, @curve_shape,
    @org_id, @external_source, @external_id
)
ON CONFLICT (org_id, external_source, external_id)
    WHERE external_source IS NOT NULL AND external_id IS NOT NULL
DO UPDATE SET
    name         = EXCLUDED.name,
    category     = EXCLUDED.category,
    region       = EXCLUDED.region,
    ad_type      = EXCLUDED.ad_type,
    platform     = EXCLUDED.platform,
    status       = EXCLUDED.status,
    days_running = EXCLUDED.days_running,
    reach        = EXCLUDED.reach,
    spend        = EXCLUDED.spend,
    budget       = EXCLUDED.budget,
    frequency    = EXCLUDED.frequency,
    updated_at   = now()
-- approval is deliberately absent from the update list. It is the
-- agency's decision, not the platform's, and letting a sync reset an
-- approved campaign to pending would undo somebody's sign-off every
-- fifteen minutes.
RETURNING id, (xmax = 0) AS created;

-- name: StartIngestRun :one
INSERT INTO ingest_runs (org_id, source) VALUES (@org_id, @source) RETURNING id;

-- name: FinishIngestRun :exec
UPDATE ingest_runs SET
    finished_at  = now(),
    rows_seen    = @rows_seen,
    rows_created = @rows_created,
    rows_updated = @rows_updated,
    rows_failed  = @rows_failed,
    error        = @error
WHERE id = @id;

-- name: ListIngestRuns :many
SELECT id, source, started_at, finished_at, rows_seen, rows_created, rows_updated, rows_failed, error
FROM ingest_runs
WHERE (@org_system::bool OR org_id = @org_id::text)
ORDER BY started_at DESC
LIMIT 50;
