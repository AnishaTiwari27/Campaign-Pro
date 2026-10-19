-- name: ListCreators :many
SELECT * FROM creators WHERE (@org_system::bool OR org_id = @org_id::text) ORDER BY followers DESC;

-- name: GetCreator :one
SELECT * FROM creators WHERE id = @id AND (@org_system::bool OR org_id = @org_id::text);

-- name: CreateCreator :one
INSERT INTO creators (id, name, role, initials, category, region, tier, followers, primary_platform, languages, org_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, @org_id)
RETURNING *;

-- name: ListCampaignsByCreator :many
SELECT * FROM campaigns WHERE creator_id = @creator_id AND (@org_system::bool OR org_id = @org_id::text) ORDER BY created_at DESC;

-- name: SetCampaignCreator :exec
UPDATE campaigns SET creator_id = $2, updated_at = now() WHERE id = $1;

-- SetCreatorPhoto attaches a creator's own photograph, or clears it when
-- the url is empty. The avatar takes over again when it is cleared.
-- name: SetCreatorPhoto :exec
UPDATE creators SET image_url = NULLIF($2, '') WHERE id = $1;

-- ListCreatorsForFollowerSync is the roster the Instagram sync walks:
-- those with a handle established. Ordered so a run that stops early
-- resumes somewhere sensible rather than at the same few.
-- name: ListCreatorsForFollowerSync :many
SELECT id, name, instagram_handle
FROM creators
WHERE instagram_handle <> ''
  AND (@org_system::bool OR org_id = @org_id::text)
ORDER BY followers_synced_at ASC NULLS FIRST, id;

-- SetCreatorFollowers records a fetched follower count. Writing the
-- timestamp is what distinguishes it from the seeded figure it
-- replaces.
-- name: SetCreatorFollowers :exec
UPDATE creators
SET followers = $2, followers_synced_at = now(), updated_at = now()
WHERE id = $1 AND (@org_system::bool OR org_id = @org_id::text);

-- SetCreatorInstagramHandle records which account to look up. Kept
-- separate from creation because a handle is established later and by
-- hand, and is wrong often enough to be worth changing on its own.
-- name: SetCreatorInstagramHandle :exec
UPDATE creators
SET instagram_handle = $2, updated_at = now()
WHERE id = $1 AND (@org_system::bool OR org_id = @org_id::text);
