DROP INDEX IF EXISTS idx_audit_events_entity;
DROP INDEX IF EXISTS idx_audit_events_user_id;
DELETE FROM audit_events WHERE campaign_id IS NULL;
ALTER TABLE audit_events ALTER COLUMN campaign_id SET NOT NULL;
ALTER TABLE audit_events DROP COLUMN IF EXISTS entity_id;
ALTER TABLE audit_events DROP COLUMN IF EXISTS entity_type;
ALTER TABLE audit_events DROP COLUMN IF EXISTS user_id;
