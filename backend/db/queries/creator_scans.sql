-- name: UpsertBrandPost :exec
-- Evidence of one disclosed brand post. A re-scan re-detects the same
-- caption, so this corrects the row rather than duplicating it — and
-- never touches dismissed, or every human correction would be undone
-- on the next run.
INSERT INTO creator_brand_posts (
    id, org_id, creator_id, brand, permalink, posted_at, matched_on, method
) VALUES (
    @id::text, @org_id::text, @creator_id::text, @brand::text,
    @permalink::text, sqlc.narg('posted_at')::timestamptz,
    @matched_on::text, @method::text
)
ON CONFLICT (creator_id, permalink, matched_on) DO UPDATE
SET brand = EXCLUDED.brand, posted_at = EXCLUDED.posted_at
WHERE creator_brand_posts.org_id = @org_id::text;

-- name: ListBrandPosts :many
SELECT * FROM creator_brand_posts
WHERE creator_id = @creator_id::text AND NOT dismissed
  AND (@org_system::bool OR org_id = @org_id::text)
ORDER BY posted_at DESC NULLS LAST;

-- name: BrandPostCountsByCreator :many
SELECT creator_id, COUNT(*) AS posts FROM creator_brand_posts
WHERE NOT dismissed AND (@org_system::bool OR org_id = @org_id::text)
GROUP BY creator_id;

-- name: DismissBrandPost :exec
UPDATE creator_brand_posts SET dismissed = true
WHERE id = @id::text AND (@org_system::bool OR org_id = @org_id::text);

-- name: MarkMediaScanned :exec
UPDATE creators SET media_scanned_at = @scanned_at::timestamptz, updated_at = now()
WHERE id = @id::text AND (@org_system::bool OR org_id = @org_id::text);

-- name: RecordFollowerSnapshot :exec
-- One row per creator per day. A second scan the same day corrects it
-- rather than adding a point, which would read as growth that never
-- happened.
INSERT INTO creator_follower_snapshots (
    org_id, creator_id, captured_on, followers, media_count, engagement_rate, posts_scanned
) VALUES (
    @org_id::text, @creator_id::text, @captured_on::date, @followers::bigint,
    @media_count::bigint, sqlc.narg('engagement_rate')::double precision, @posts_scanned::int
)
ON CONFLICT (org_id, creator_id, captured_on) DO UPDATE
SET followers = EXCLUDED.followers,
    media_count = EXCLUDED.media_count,
    engagement_rate = EXCLUDED.engagement_rate,
    posts_scanned = EXCLUDED.posts_scanned;

-- name: ListFollowerSnapshots :many
SELECT * FROM creator_follower_snapshots
WHERE creator_id = @creator_id::text
  AND (@org_system::bool OR org_id = @org_id::text)
ORDER BY captured_on;
