-- Disclosed brand work, detected from a creator's own captions.
--
-- Evidence, not a verdict. The table stores which post, which marker
-- fired and which brand it named, because the alternative — a boolean
-- on creators saying "has done brand work" — is an unsubstantiated
-- commercial claim about a real named person, with nothing behind it a
-- human could check or overturn.
CREATE TABLE creator_brand_posts (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    creator_id TEXT NOT NULL REFERENCES creators(id) ON DELETE CASCADE,

    -- Empty where a disclosure fired but tagged nobody, which is common
    -- and honest. Guessing a brand out of the caption text would be the
    -- invented relationship this table exists to avoid.
    brand     TEXT NOT NULL DEFAULT '',
    permalink TEXT NOT NULL,
    posted_at TIMESTAMPTZ,

    -- The literal token that fired ('#ad', '@nykaa', 'paid partnership')
    -- and how it was found (hashtag | mention | phrase). Shown next to
    -- the detection so the reader sees the reason rather than a verdict.
    matched_on TEXT NOT NULL,
    method     TEXT NOT NULL,

    -- A human's say-so, which always outranks the detector. Dismissed
    -- survives re-detection: the same caption will match again on the
    -- next sync, and without this the correction would be undone every
    -- time the sync ran.
    dismissed    BOOLEAN NOT NULL DEFAULT false,
    confirmed_by UUID REFERENCES users(id),
    confirmed_at TIMESTAMPTZ,

    detected_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- One row per marker per post. Re-running the sync updates rather
    -- than piling up duplicates of the same evidence.
    UNIQUE (creator_id, permalink, matched_on)
);
CREATE INDEX creator_brand_posts_creator_idx ON creator_brand_posts (creator_id) WHERE NOT dismissed;
CREATE INDEX creator_brand_posts_org_idx     ON creator_brand_posts (org_id);

-- When this creator's media was last read.
--
-- Null is the load-bearing state: it means nobody has looked, which is
-- not the same as having looked and found nothing. Without this column
-- every unscanned creator would read as "no brand work", turning our
-- own backlog into a claim about them.
ALTER TABLE creators ADD COLUMN media_scanned_at TIMESTAMPTZ;
