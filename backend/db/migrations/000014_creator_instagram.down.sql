DROP INDEX IF EXISTS creators_instagram_handle_idx;
ALTER TABLE creators
    DROP COLUMN IF EXISTS followers_synced_at,
    DROP COLUMN IF EXISTS instagram_handle;
