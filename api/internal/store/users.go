package store

import (
	"context"
	"errors"
	"time"

	"campaigntrackerpro/internal/db/gen"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID         string
	Email      string
	Name       string
	Role       string
	CanApprove bool
	CreatedAt  time.Time
}

func toDomainUser(u gen.User) User {
	return User{
		ID:         uuidToString(u.ID),
		Email:      u.Email,
		Name:       u.Name,
		Role:       string(u.Role),
		CanApprove: u.CanApprove,
		CreatedAt:  timeOf(u.CreatedAt),
	}
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	u, err := s.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return toDomainUser(u), nil
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	us, err := s.Queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]User, len(us))
	for i, u := range us {
		out[i] = toDomainUser(u)
	}
	return out, nil
}

func (s *Store) CreateUser(ctx context.Context, email, name, role string, canApprove bool) (User, error) {
	u, err := s.Queries.CreateUser(ctx, gen.CreateUserParams{
		Email: email, Name: name, Role: gen.UserRoleT(role), CanApprove: canApprove,
	})
	if err != nil {
		return User{}, err
	}
	return toDomainUser(u), nil
}
