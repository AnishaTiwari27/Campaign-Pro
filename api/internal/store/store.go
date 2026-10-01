// Package store adapts the sqlc-generated queries (internal/db/gen) onto
// domain types, so nothing outside this package imports pgtype or gen.
package store

import (
	"context"

	"campaigntrackerpro/internal/db/gen"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool    *pgxpool.Pool
	Queries *gen.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool, Queries: gen.New(pool)}
}

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
