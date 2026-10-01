-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at ASC;

-- name: CreateUser :one
INSERT INTO users (email, name, role, can_approve)
VALUES ($1, $2, $3, $4) RETURNING *;
