package store

import "campaigntrackerpro/platform/database"

// Store is this service's data access. It holds the shared pool rather
// than opening its own, but only ever runs this service's own queries —
// the package is internal, so no sibling service can borrow it.
type Store struct {
	db *database.DB
}

func New(db *database.DB) *Store { return &Store{db: db} }
