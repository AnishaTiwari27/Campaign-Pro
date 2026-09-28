// Package store is auth-service's only door into Postgres — every query is
// qualified against "auth.*", the only schema this service (and only this
// service) can see. See db/init/00_roles.sql.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"campaigntrackerpro/services/auth/internal/models"
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

func (s *Store) GetUserByEmail(ctx context.Context, email string) (models.User, bool, error) {
	return s.scanUser(ctx, `SELECT id, email, password_hash, role FROM auth.users WHERE email = $1`, email)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (models.User, bool, error) {
	return s.scanUser(ctx, `SELECT id, email, password_hash, role FROM auth.users WHERE id = $1`, id)
}

// CreateUser inserts a new viewer account — role is hardcoded here, not
// taken as a parameter, so nothing above this layer can accidentally wire
// a caller-supplied role through to an INSERT. Returns models.ErrEmailTaken
// (mapped off Postgres' unique-violation error, auth.users.email is
// UNIQUE — db/init/03_auth.sql) if the email is already registered; letting
// the database's own constraint be the source of truth avoids a
// check-then-insert race against a concurrent registration of the same
// email that a separate existence check would have.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (models.User, error) {
	u := models.User{Email: email, PasswordHash: passwordHash, Role: "viewer"}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO auth.users (email, password_hash, role) VALUES ($1, $2, 'viewer') RETURNING id`,
		email, passwordHash,
	).Scan(&u.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return models.User{}, models.ErrEmailTaken
		}
		return models.User{}, err
	}
	return u, nil
}

func (s *Store) scanUser(ctx context.Context, sql string, arg any) (models.User, bool, error) {
	var u models.User
	err := s.pool.QueryRow(ctx, sql, arg).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
	if err == pgx.ErrNoRows {
		return models.User{}, false, nil
	}
	if err != nil {
		return models.User{}, false, err
	}
	return u, true, nil
}

// SaveRefreshToken persists tokenHash (never the raw token — see
// db/init/03_auth.sql) for userID, expiring at expiresAt.
func (s *Store) SaveRefreshToken(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO auth.refresh_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt)
	return err
}

// ConsumeRefreshToken atomically deletes the row for tokenHash (if it
// exists and hasn't expired) and returns the user it belonged to — a
// delete-and-return in one round trip so two concurrent /auth/refresh
// calls for the same stolen-and-legitimate token pair can't both succeed.
func (s *Store) ConsumeRefreshToken(ctx context.Context, tokenHash string) (userID string, ok bool, err error) {
	err = s.pool.QueryRow(ctx,
		`DELETE FROM auth.refresh_tokens WHERE token_hash = $1 AND expires_at > now() RETURNING user_id`,
		tokenHash).Scan(&userID)
	if err == pgx.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return userID, true, nil
}

// DeleteRefreshToken revokes tokenHash outright — backs POST /auth/logout.
// Deleting a token that's already gone (expired, already used, or unknown)
// is not an error: logout is idempotent from the caller's point of view.
func (s *Store) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM auth.refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

// ListUsers backs GET /api/v1/users (admin-only) — every account, ordered
// by email for a stable listing.
func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, email, password_hash, role FROM auth.users ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UpdateUserRole sets id's role — backs PATCH /api/v1/users/{id}/role
// (admin-only). Same not-found-is-not-an-error shape
// campaigns-service's UpdateBudget/UpdateApprovalStatus already use.
func (s *Store) UpdateUserRole(ctx context.Context, id, role string) (models.User, bool, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`UPDATE auth.users SET role = $1 WHERE id = $2 RETURNING id, email, password_hash, role`,
		role, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
	if err == pgx.ErrNoRows {
		return models.User{}, false, nil
	}
	if err != nil {
		return models.User{}, false, err
	}
	return u, true, nil
}
