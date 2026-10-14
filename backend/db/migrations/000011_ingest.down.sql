DROP INDEX IF EXISTS ingest_runs_org_started_idx;
DROP TABLE IF EXISTS ingest_runs;
DROP INDEX IF EXISTS campaigns_external_key;
ALTER TABLE campaigns DROP COLUMN IF EXISTS external_id;
ALTER TABLE campaigns DROP COLUMN IF EXISTS external_source;
