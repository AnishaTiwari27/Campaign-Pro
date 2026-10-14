-- name: ListReports :many
SELECT * FROM reports WHERE (@org_system::bool OR org_id = @org_id::text) ORDER BY created_at DESC;

-- name: GetReport :one
SELECT * FROM reports WHERE id = @id AND (@org_system::bool OR org_id = @org_id::text);

-- name: ListEnabledReports :many
SELECT * FROM reports WHERE enabled = true AND (@org_system::bool OR org_id = @org_id::text);

-- name: ListEnabledReportsByCadence :many
SELECT * FROM reports WHERE enabled = true AND cadence = @cadence AND (@org_system::bool OR org_id = @org_id::text);

-- name: CreateReport :one
INSERT INTO reports (name, enabled, cadence, recipients, scope_filters, scope_label, columns, org_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, @org_id) RETURNING *;

-- UpdateReport applies whichever fields are non-null; every field is
-- independently optional so a PATCH can change just one of them.
-- name: UpdateReport :one
UPDATE reports SET
    name          = coalesce(sqlc.narg('name')::text, name),
    enabled       = coalesce(sqlc.narg('enabled')::bool, enabled),
    cadence       = coalesce(sqlc.narg('cadence')::report_cadence_t, cadence),
    recipients    = coalesce(sqlc.narg('recipients')::text[], recipients),
    scope_filters = coalesce(sqlc.narg('scope_filters')::jsonb, scope_filters),
    scope_label   = coalesce(sqlc.narg('scope_label')::text, scope_label),
    columns       = coalesce(sqlc.narg('columns')::text[], columns),
    updated_at    = now()
WHERE id = sqlc.arg('id') AND (@org_system::bool OR org_id = @org_id::text)
RETURNING *;

-- name: DeleteReport :exec
DELETE FROM reports WHERE id = @id AND (@org_system::bool OR org_id = @org_id::text);
