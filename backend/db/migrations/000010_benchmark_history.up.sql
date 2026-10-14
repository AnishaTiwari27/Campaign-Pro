-- Benchmarks over time.
--
-- The benchmark page answers "how does this category compare" from the
-- campaigns that are running right now. That is a snapshot, and a
-- snapshot cannot answer the question people actually ask next: is this
-- category getting cheaper or dearer, and is our position in it moving.
--
-- Reach and spend live on the campaign row and are overwritten as it
-- runs, so yesterday's median is not recoverable after the fact. The
-- only way to have the series is to have been writing it down, which is
-- what this table is for.
--
-- One row per organisation, category and day. The day is the grain
-- because the worker ticks every thirty seconds and nobody wants two
-- thousand points a day on a line chart; the primary key makes a repeat
-- capture an update rather than a duplicate.
CREATE TABLE benchmark_snapshots (
    org_id        TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    category      TEXT NOT NULL,
    captured_on   DATE NOT NULL,
    median_reach  DOUBLE PRECISION NOT NULL,
    top_reach     DOUBLE PRECISION NOT NULL,
    campaign_count INTEGER NOT NULL,
    -- Median spend per lakh of reach: the cost side of the same
    -- question, and the one that tells you whether a category is
    -- getting more expensive rather than just busier.
    median_cost_per_lakh DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (org_id, category, captured_on)
);

-- Every read is "this organisation's series, most recent first".
CREATE INDEX benchmark_snapshots_org_day_idx
    ON benchmark_snapshots (org_id, captured_on DESC);
