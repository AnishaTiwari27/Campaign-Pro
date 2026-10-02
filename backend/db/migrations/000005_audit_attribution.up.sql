-- audit_events recorded `actor` as a display string: thirty rows said
-- "You", which cannot answer "who approved this?". And campaign_id was
-- NOT NULL, so nothing outside a campaign could be audited — not a login,
-- not a report run.
--
-- actor stays as the denormalised label: names change, and a trail that
-- rewrites history is not a trail.

ALTER TABLE audit_events ADD COLUMN user_id UUID REFERENCES users(id);
ALTER TABLE audit_events ADD COLUMN entity_type TEXT NOT NULL DEFAULT 'campaign';
ALTER TABLE audit_events ADD COLUMN entity_id TEXT;
ALTER TABLE audit_events ALTER COLUMN campaign_id DROP NOT NULL;

UPDATE audit_events SET entity_id = campaign_id WHERE entity_id IS NULL;
CREATE INDEX idx_audit_events_user_id ON audit_events (user_id, created_at DESC);
CREATE INDEX idx_audit_events_entity ON audit_events (entity_type, entity_id);
