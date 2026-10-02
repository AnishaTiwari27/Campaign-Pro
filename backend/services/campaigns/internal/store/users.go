package store

import (
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/campaigns"
	"context"
	"errors"

	"campaigntrackerpro/db/gen"

	"github.com/jackc/pgx/v5"
)

func toDomainUser(u gen.User) campaigns.User {
	return campaigns.User{
		ID:         database.UuidToString(u.ID),
		Email:      u.Email,
		Name:       u.Name,
		Role:       string(u.Role),
		CanApprove: u.CanApprove,
		CreatedAt:  database.TimeOf(u.CreatedAt),
	}
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (campaigns.User, error) {
	u, err := s.db.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return campaigns.User{}, database.ErrNotFound
		}
		return campaigns.User{}, err
	}
	return toDomainUser(u), nil
}

func (s *Store) ListUsers(ctx context.Context) ([]campaigns.User, error) {
	us, err := s.db.Queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]campaigns.User, len(us))
	for i, u := range us {
		out[i] = toDomainUser(u)
	}
	return out, nil
}

func (s *Store) CreateUser(ctx context.Context, email, name, role string, canApprove bool) (campaigns.User, error) {
	u, err := s.db.Queries.CreateUser(ctx, gen.CreateUserParams{
		Email: email, Name: name, Role: gen.UserRoleT(role), CanApprove: canApprove,
	})
	if err != nil {
		return campaigns.User{}, err
	}
	return toDomainUser(u), nil
}

// SetUserAgency marks a user as agency staff, who see every account.
// Client users are scoped to the accounts granted to them instead.
func (s *Store) SetUserAgency(ctx context.Context, userID string, isAgency bool) error {
	uid, err := database.UuidParam(userID)
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `UPDATE users SET is_agency = $2 WHERE id = $1`, uid, isAgency)
	return err
}
