package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"campaigntrackerpro/db/gen"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/identity"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Store struct {
	db *database.DB
}

func New(db *database.DB) *Store { return &Store{db: db} }

// HashToken is what gets stored. The raw token only ever exists in the
// user's cookie, so a dump of the sessions table yields nothing usable.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

type authUser struct {
	identity.User
	PasswordHash string
}

func toUser(id pgtype.UUID, email, name string, role gen.UserRoleT, isAgency bool,
	hash pgtype.Text, created pgtype.Timestamptz, orgID string) authUser {
	return authUser{
		User: identity.User{
			ID:        database.UuidToString(id),
			Email:     email,
			Name:      name,
			Role:      identity.Role(role),
			IsAgency:  isAgency,
			OrgID:     orgID,
			CreatedAt: database.TimeOf(created),
		},
		PasswordHash: database.TextOrEmpty(hash),
	}
}

func (s *Store) UserForAuth(ctx context.Context, email string) (identity.User, string, error) {
	r, err := s.db.Queries.GetUserForAuth(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.User{}, "", database.ErrNotFound
		}
		return identity.User{}, "", err
	}
	u := toUser(r.ID, r.Email, r.Name, r.Role, r.IsAgency, r.PasswordHash, r.CreatedAt, r.OrgID)
	return u.User, u.PasswordHash, nil
}

// UserForSession resolves a session token in one round trip, and only if
// the session has not expired.
func (s *Store) UserForSession(ctx context.Context, token string) (identity.User, error) {
	r, err := s.db.Queries.SessionUser(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.User{}, database.ErrNotFound
		}
		return identity.User{}, err
	}
	return toUser(r.ID, r.Email, r.Name, r.Role, r.IsAgency, r.PasswordHash, r.CreatedAt, r.OrgID).User, nil
}

func (s *Store) CreateSession(ctx context.Context, token, userID, userAgent, ip string) error {
	uid, err := database.UuidParam(userID)
	if err != nil {
		return err
	}
	_, err = s.db.Queries.CreateSession(ctx, gen.CreateSessionParams{
		TokenHash: HashToken(token),
		UserID:    uid,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(identity.SessionTTL), Valid: true},
		UserAgent: database.TextParam(userAgent),
		Ip:        database.TextParam(ip),
	})
	return err
}

// DeleteSession is what makes logout mean something.
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	return s.db.Queries.DeleteSession(ctx, HashToken(token))
}

func (s *Store) TouchLogin(ctx context.Context, userID string) error {
	uid, err := database.UuidParam(userID)
	if err != nil {
		return err
	}
	return s.db.Queries.TouchUserLogin(ctx, uid)
}

func (s *Store) SetPassword(ctx context.Context, userID, hash string) error {
	sys, org := orgParams(ctx)
	uid, err := database.UuidParam(userID)
	if err != nil {
		return err
	}
	return s.db.Queries.SetUserPassword(ctx, gen.SetUserPasswordParams{
		OrgSystem: sys, OrgID: org, ID: uid, PasswordHash: database.TextParam(hash)})
}

// PurgeExpired clears sessions past their expiry. Expired sessions are
// already rejected on read; this just stops the table growing forever.
func (s *Store) PurgeExpired(ctx context.Context) error {
	return s.db.Queries.DeleteExpiredSessions(ctx)
}

// orgParams turns the request's tenant into query parameters. Every
// statement touching an org-owned table takes these; platform/arch's
// tenancy test fails the build if one is added that does not.
//
// A system context spans organisations, which is right for the worker,
// the seeder and the CLI. Anything else is confined to one, and an
// unresolved organisation is the empty string — which matches no row, so
// forgetting to set it yields an empty result rather than a leak.
func orgParams(ctx context.Context) (bool, string) {
	o := scope.OrgFrom(ctx)
	return o.System, o.ID
}
