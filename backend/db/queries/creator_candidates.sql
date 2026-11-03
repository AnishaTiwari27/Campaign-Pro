-- name: UpsertCandidate :exec
-- Re-running a search finds the same channels. This refreshes their
-- figures without resetting status, or a rejected candidate would come
-- back for review every week.
INSERT INTO creator_candidates (
    id, org_id, platform, external_id, handle, name, description,
    followers, videos, views, found_for
) VALUES (
    @id::text, @org_id::text, @platform::text, @external_id::text,
    @handle::text, @name::text, @description::text,
    sqlc.narg('followers')::bigint, @videos::bigint, @views::bigint, @found_for::text
)
ON CONFLICT (org_id, platform, external_id) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    followers = EXCLUDED.followers,
    videos = EXCLUDED.videos,
    views = EXCLUDED.views;

-- name: ListNewCandidates :many
SELECT * FROM creator_candidates
WHERE status = 'new' AND (@org_system::bool OR org_id = @org_id::text)
ORDER BY followers DESC NULLS LAST, discovered_at DESC
LIMIT @row_limit::int;

-- name: SetCandidateStatus :exec
UPDATE creator_candidates
SET status = @status::text, reviewed_at = now()
WHERE id = @id::text AND (@org_system::bool OR org_id = @org_id::text);

-- name: CountCandidatesByStatus :many
SELECT status, COUNT(*) AS total FROM creator_candidates
WHERE (@org_system::bool OR org_id = @org_id::text)
GROUP BY status;
