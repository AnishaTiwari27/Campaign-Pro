-- One row per creator per day: what the platform reported about them
-- that day.
--
-- This is the table that becomes worth money by existing early. A
-- discovery vendor's "growth trend" is nothing more cunning than
-- snapshots they started taking before we did — the figures are public
-- and first-hand, the history is the only scarce part. Six months of
-- these cannot be bought later at any price, which makes the cost of
-- delaying this table higher than the cost of any other table here.
--
-- It also carries the clearest bought-follower signal there is. A real
-- audience grows in a curve; a purchased one arrives as a step, and a
-- step is only visible to someone holding both sides of it.
CREATE TABLE creator_follower_snapshots (
    org_id      TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    creator_id  TEXT NOT NULL REFERENCES creators(id) ON DELETE CASCADE,
    captured_on DATE NOT NULL,

    followers   BIGINT NOT NULL CHECK (followers >= 0),
    media_count BIGINT NOT NULL DEFAULT 0,

    -- Null where the scan could not compute one: fewer than three
    -- organic posts, or no follower base to divide by. Null means
    -- "not computable", never zero engagement — a zero here would be
    -- read as a dead account and it is nothing of the kind.
    engagement_rate DOUBLE PRECISION,
    posts_scanned   INTEGER NOT NULL DEFAULT 0,

    -- Same day, same creator, one row. Re-running a scan corrects the
    -- day rather than stacking duplicate points into the series, which
    -- would show as growth that never happened.
    PRIMARY KEY (org_id, creator_id, captured_on)
);
CREATE INDEX creator_follower_snapshots_creator_idx
    ON creator_follower_snapshots (creator_id, captured_on DESC);
