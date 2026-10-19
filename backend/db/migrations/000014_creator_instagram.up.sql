-- Where a creator's follower count comes from, and when.
--
-- The number has always been typed in from a press article, which
-- goes stale the day it is written and has no way of saying so. These
-- two columns are what let it be refreshed from Instagram and, just
-- as importantly, what let the app tell the difference between a
-- figure it fetched and one somebody keyed in.
ALTER TABLE creators
    -- The professional account to query, without the @. Empty where
    -- nobody has established it: a guessed handle is worse than none,
    -- because it attaches another person's audience to this one.
    ADD COLUMN instagram_handle TEXT NOT NULL DEFAULT '',
    -- Null means the follower count has never been fetched — it is
    -- seeded, and should be read as "about this, at some point".
    ADD COLUMN followers_synced_at TIMESTAMPTZ;

-- The sync walks creators that have a handle.
CREATE INDEX creators_instagram_handle_idx ON creators (instagram_handle)
    WHERE instagram_handle <> '';
