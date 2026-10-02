-- name: ListReportRuns :many
SELECT * FROM report_runs WHERE report_id = $1 ORDER BY ran_at DESC LIMIT $2;

-- name: GetLastReportRun :one
SELECT * FROM report_runs WHERE report_id = $1 ORDER BY ran_at DESC LIMIT 1;

-- name: CreateReportRun :one
INSERT INTO report_runs (report_id, result, row_count)
VALUES ($1, $2, $3) RETURNING *;
