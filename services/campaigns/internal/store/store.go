package store

import (
	"context"

	"campaigntrackerpro/platform/database"
)

// Store is this service's data access. It holds the shared pool rather
// than opening its own, but only ever runs this service's own queries —
// the package is internal, so no sibling service can borrow it.
type Store struct {
	db *database.DB
}

func New(db *database.DB) *Store { return &Store{db: db} }

// Exec runs a statement directly against the pool. Used by tests to set up
// and tear down fixture rows without going through the service layer.
func (s *Store) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := s.db.Pool.Exec(ctx, sql, args...)
	return err
}
