-- name: GetUserForAuth :one
SELECT id, email, name, role, can_approve, is_agency, password_hash, created_at, org_id
FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, name, role, can_approve, is_agency, password_hash, created_at, org_id
FROM users WHERE id = $1;

-- name: SetUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1 AND (@org_system::bool OR org_id = @org_id::text);

-- name: TouchUserLogin :exec
UPDATE users SET last_login_at = now() WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (token_hash, user_id, expires_at, user_agent, ip)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- SessionUser resolves a session to its user in one round trip, and only
-- when the session has not expired.
-- name: SessionUser :one
SELECT u.id, u.email, u.name, u.role, u.can_approve, u.is_agency, u.password_hash, u.created_at, u.org_id
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= now();

-- ListSessionsForUser is the "recent sign-ins" list. Keyed by a user id
-- that was resolved from the caller's own session, so it can only ever
-- return the caller's.
-- name: ListSessionsForUser :many
SELECT token_hash, created_at, expires_at, user_agent, ip
FROM sessions
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 10;
