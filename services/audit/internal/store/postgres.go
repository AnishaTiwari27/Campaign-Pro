// Package store is audit-service's only door into Postgres — every query
// is qualified against "audit.*", the only schema this service can see.
package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"campaigntrackerpro/services/audit/internal/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Insert appends one audit entry — this table is never updated or
// deleted from afterward, by convention (not by grant).
func (s *Store) Insert(ctx context.Context, e models.AuditEvent) error {
	var campaignID *int64
	if e.CampaignID != 0 {
		id := e.CampaignID
		campaignID = &id
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit.audit_log (action, actor_id, actor_email, actor_role, campaign_id, details)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		e.Action, e.ActorID, e.ActorEmail, e.ActorRole, campaignID, e.Details)
	return err
}

// List returns page/limit entries newest-first, optionally filtered to one
// campaign — same parameterized-clause-building shape
// campaigns-service's whereClause uses, sized down to the one filter this
// needs.
func (s *Store) List(ctx context.Context, page, limit int, campaignID *int64) ([]models.Entry, int, error) {
	clauses := []string{"1=1"}
	var args []any
	if campaignID != nil {
		args = append(args, *campaignID)
		clauses = append(clauses, fmt.Sprintf("campaign_id = $%d", len(args)))
	}
	where := strings.Join(clauses, " AND ")

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM audit.audit_log WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	listArgs := append(append([]any{}, args...), limit, offset)
	listSQL := fmt.Sprintf(`SELECT id, action, actor_id, actor_email, actor_role, campaign_id, details, created_at
		FROM audit.audit_log WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, len(listArgs)-1, len(listArgs))

	rows, err := s.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Entry
	for rows.Next() {
		var e models.Entry
		if err := rows.Scan(&e.ID, &e.Action, &e.ActorID, &e.ActorEmail, &e.ActorRole, &e.CampaignID, &e.Details, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}
