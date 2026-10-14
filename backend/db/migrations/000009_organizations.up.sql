-- Multi-tenancy: the organisation owns the data.
--
-- Until now the tenant boundary was implicit and singular — one agency,
-- its book of business, and client accounts narrowed to named campaigns
-- within it. client_campaign_grants narrows *within* an agency; it was
-- never a boundary between agencies, and using it as one would mean
-- enumerating every campaign of every other tenant as "not granted".
--
-- So the boundary goes here, as a column every org-owned table carries
-- and every read filters on. Two levels, and they do different jobs:
--
--   org_id   — a hard wall. Always applied, to every account including
--              admins. Nothing crosses it, ever.
--   grants   — a narrowing inside one org, for client accounts only.
--
-- The agency is the tenant rather than the brand, because this is an
-- agency's tool: campaigns, creators, reports and staff all belong to
-- the agency, and a brand is a column on a campaign, not an owner of one.
CREATE TABLE organizations (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Everything that exists today belongs to the one organisation there has
-- always been. Named rather than generated so the backfill below is
-- readable and so a fresh seed lands in the same place.
INSERT INTO organizations (id, name) VALUES ('default', 'Campaign Tracker Pro');

-- DEFAULT 'default' is deliberate and temporary-feeling, but it is what
-- makes this migration safe to run against live data: every existing row
-- gets the one organisation, and any insert written before this migration
-- still works rather than failing a NOT NULL. The cost is that a future
-- bug which forgets to set org_id writes into 'default' silently instead
-- of erroring. The query-level test is what catches that class of thing,
-- not the column default.
ALTER TABLE users     ADD COLUMN org_id TEXT NOT NULL DEFAULT 'default' REFERENCES organizations(id);
ALTER TABLE campaigns ADD COLUMN org_id TEXT NOT NULL DEFAULT 'default' REFERENCES organizations(id);
ALTER TABLE creators  ADD COLUMN org_id TEXT NOT NULL DEFAULT 'default' REFERENCES organizations(id);
ALTER TABLE reports   ADD COLUMN org_id TEXT NOT NULL DEFAULT 'default' REFERENCES organizations(id);

-- Every read filters on org_id, so every one of these is a hot path.
CREATE INDEX users_org_idx     ON users (org_id);
CREATE INDEX campaigns_org_idx ON campaigns (org_id);
CREATE INDEX creators_org_idx  ON creators (org_id);
CREATE INDEX reports_org_idx   ON reports (org_id);

-- An email is unique per organisation, not globally: two agencies may
-- each have a person whose address is the same, and more practically the
-- same consultant may hold an account in both.
-- The constraint owns the index, so it has to go first; dropping the
-- index directly is refused while the constraint depends on it.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
DROP INDEX IF EXISTS users_email_key;
CREATE UNIQUE INDEX users_org_email_key ON users (org_id, lower(email));
