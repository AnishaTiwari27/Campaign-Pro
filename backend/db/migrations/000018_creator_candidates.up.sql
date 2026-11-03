-- Channels discovery turned up, waiting on a human.
--
-- Separate from creators on purpose. A search result is a guess that
-- someone matches a brief; the roster is people we work with, and
-- every figure the product computes — tier medians, benchmarks, cost
-- indices — is drawn from it. Tipping unvetted search results straight
-- in would move those baselines using channels nobody has looked at.
--
-- So discovery writes here, a person promotes, and the roster only
-- ever grows by a decision.
CREATE TABLE creator_candidates (
    id       TEXT PRIMARY KEY,
    org_id   TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    -- The platform's own id for them: a YouTube channel id. Stable
    -- where a handle or a display name is not.
    external_id TEXT NOT NULL,
    handle      TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- Null where the channel hides its subscriber count, which YouTube
    -- allows. Null is "not allowed to know"; a zero here would file a
    -- large channel as a nano creator.
    followers BIGINT,
    videos    BIGINT NOT NULL DEFAULT 0,
    views     BIGINT NOT NULL DEFAULT 0,

    -- The search that turned them up, kept so a reviewer can see what
    -- they were supposed to match. A candidate with no brief attached
    -- is a name with no reason beside it.
    found_for TEXT NOT NULL DEFAULT '',

    -- new | accepted | rejected. Rejected survives re-discovery: the
    -- same search will find them again next week, and without this the
    -- reviewer would be asked the same question forever.
    status              TEXT NOT NULL DEFAULT 'new',
    promoted_creator_id TEXT REFERENCES creators(id) ON DELETE SET NULL,

    discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at   TIMESTAMPTZ,

    UNIQUE (org_id, platform, external_id)
);
CREATE INDEX creator_candidates_review_idx
    ON creator_candidates (org_id, discovered_at DESC) WHERE status = 'new';
