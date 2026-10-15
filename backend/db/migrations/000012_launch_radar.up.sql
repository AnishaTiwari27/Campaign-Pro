-- Recently launched products and brands, and whether they look ready
-- to start marketing.
--
-- The point of the table, beyond holding the list, is the last two
-- columns. This system has no history of launches and when they
-- actually began advertising, so any estimate it makes today is a
-- heuristic over observable signals and not a forecast derived from
-- outcomes. Recording what was estimated, and then recording what
-- actually happened when a matching campaign turns up, is what turns
-- the first into the second. Nothing can measure the heuristic until
-- that data exists, so it is collected from the first day rather than
-- added once someone asks how accurate it is.
CREATE TABLE launches (
    id           TEXT PRIMARY KEY,
    org_id       TEXT NOT NULL DEFAULT 'default' REFERENCES organizations(id) ON DELETE CASCADE,
    source       TEXT NOT NULL,
    -- The source's own id, so a re-poll updates rather than duplicates.
    external_id  TEXT NOT NULL,
    name         TEXT NOT NULL,
    tagline      TEXT NOT NULL DEFAULT '',
    url          TEXT NOT NULL DEFAULT '',
    category     TEXT NOT NULL DEFAULT '',
    launched_at  TIMESTAMPTZ NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Observable signals the banding is computed from, kept so the
    -- estimate can be re-derived and audited rather than just trusted.
    traction     INTEGER NOT NULL DEFAULT 0,
    has_site     BOOLEAN NOT NULL DEFAULT false,

    -- What was estimated, and why.
    readiness    TEXT NOT NULL DEFAULT 'unclear',
    basis        TEXT NOT NULL DEFAULT '',

    -- What actually happened. Null until a campaign for this brand
    -- appears; days_to_market is then the gap that was being guessed.
    first_campaign_at TIMESTAMPTZ,
    days_to_market    INTEGER,

    UNIQUE (org_id, source, external_id)
);

CREATE INDEX launches_org_launched_idx ON launches (org_id, launched_at DESC);
-- The outcome sweep looks for launches still waiting on a campaign.
CREATE INDEX launches_pending_outcome_idx ON launches (org_id) WHERE first_campaign_at IS NULL;
