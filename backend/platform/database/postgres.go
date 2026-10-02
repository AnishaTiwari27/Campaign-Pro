// Package database owns the connection pool and the generated query set.
// It is the one place that knows how to reach Postgres; every service
// wraps this rather than opening its own pool.
package database

import (
	"context"

	"campaigntrackerpro/db/gen"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is the shared data-access primitive handed to each service's store.
type DB struct {
	Pool    *pgxpool.Pool
	Queries *gen.Queries
}

func New(pool *pgxpool.Pool) *DB {
	return &DB{Pool: pool, Queries: gen.New(pool)}
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
