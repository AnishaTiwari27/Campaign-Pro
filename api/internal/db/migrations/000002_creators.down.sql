DROP INDEX IF EXISTS idx_creatives_hook_type;
DROP INDEX IF EXISTS idx_creatives_language;
ALTER TABLE creatives DROP COLUMN IF EXISTS analyzed_at;
ALTER TABLE creatives DROP COLUMN IF EXISTS festival;
ALTER TABLE creatives DROP COLUMN IF EXISTS claim;
ALTER TABLE creatives DROP COLUMN IF EXISTS hook_type;
ALTER TABLE creatives DROP COLUMN IF EXISTS language;
DROP TYPE IF EXISTS hook_type_t;

DROP INDEX IF EXISTS idx_campaigns_creator_id;
ALTER TABLE campaigns DROP COLUMN IF EXISTS creator_id;

DROP TABLE IF EXISTS creators;
DROP TYPE IF EXISTS creator_tier_t;
