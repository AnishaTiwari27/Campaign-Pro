-- Snapshots are written by the worker and are per-organisation, so they
-- carry org_id like every other tenant-owned row. They are not in the
-- arch test's org-owned list because the table did not exist when that
-- list was written; it is added there in the same change.

-- name: UpsertBenchmarkSnapshot :exec
INSERT INTO benchmark_snapshots (
    org_id, category, captured_on, median_reach, top_reach, campaign_count, median_cost_per_lakh
) VALUES (@org_id, @category, @captured_on, @median_reach, @top_reach, @campaign_count, @median_cost_per_lakh)
ON CONFLICT (org_id, category, captured_on) DO UPDATE SET
    median_reach         = EXCLUDED.median_reach,
    top_reach            = EXCLUDED.top_reach,
    campaign_count       = EXCLUDED.campaign_count,
    median_cost_per_lakh = EXCLUDED.median_cost_per_lakh;

-- name: ListBenchmarkTrend :many
SELECT category, captured_on, median_reach, top_reach, campaign_count, median_cost_per_lakh
FROM benchmark_snapshots
WHERE (@org_system::bool OR org_id = @org_id::text)
  AND captured_on >= @since::date
ORDER BY captured_on ASC, category ASC;

-- name: ListOrganizationIDs :many
SELECT id FROM organizations ORDER BY id;
