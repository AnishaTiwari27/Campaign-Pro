-- Ingestion from real ad platforms.
--
-- Everything in campaigns so far was written by the seeder, which could
-- choose its own ids. A platform chooses its own too, and they are the
-- only stable handle on "the same campaign as last time" — names get
-- edited, budgets change, and matching on either would create a
-- duplicate every time somebody renamed something.
--
-- So a campaign records where it came from and what it is called there.
-- The unique index is the whole idempotency story: a sync that runs
-- twice, or overlaps itself, updates the same row rather than inserting
-- a second one.
ALTER TABLE campaigns ADD COLUMN external_source TEXT;
ALTER TABLE campaigns ADD COLUMN external_id TEXT;

-- Partial, so the thirty-five seeded campaigns with no external id do
-- not all collide on NULL.
CREATE UNIQUE INDEX campaigns_external_key
    ON campaigns (org_id, external_source, external_id)
    WHERE external_source IS NOT NULL AND external_id IS NOT NULL;

-- What happened on each sync, kept because an ingestion that silently
-- does nothing is indistinguishable from one that is working until
-- somebody asks why the numbers are stale.
CREATE TABLE ingest_runs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    source      TEXT NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    -- Counted separately: "47 rows, 2 new, 45 updated, 0 failed" is a
    -- different story from "47 rows, 47 new", and the second one on a
    -- second run means the matching is broken.
    rows_seen    INTEGER NOT NULL DEFAULT 0,
    rows_created INTEGER NOT NULL DEFAULT 0,
    rows_updated INTEGER NOT NULL DEFAULT 0,
    rows_failed  INTEGER NOT NULL DEFAULT 0,
    error        TEXT
);

CREATE INDEX ingest_runs_org_started_idx ON ingest_runs (org_id, started_at DESC);
