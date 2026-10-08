package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/identity"
	"campaigntrackerpro/services/identity/internal/store"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidCredentials is deliberately the only failure a caller sees for
// a bad login. Distinguishing "no such user" from "wrong password" tells
// an attacker which emails are registered.
var ErrInvalidCredentials = errors.New("invalid email or password")

// Auditor records security-relevant events. Identity depends on the
// capability, not on whichever service owns the audit table.
type Auditor interface {
	Record(ctx context.Context, userID, actor, action, entityType, entityID string) error
}

type Auth struct {
	store   *store.Store
	auditor Auditor
	// directory writes the users table, which campaigns owns. Nil where
	// signup is not wired — createuser, for one, never registers anyone.
	directory Directory
}

func New(s *store.Store, auditor Auditor, directory Directory) *Auth {
	return &Auth{store: s, auditor: auditor, directory: directory}
}

// newToken returns 32 bytes of crypto/rand as URL-safe base64. Only the
// hash is stored; this value exists solely in the user's cookie.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Login verifies the password and issues a session. It returns the raw
// token for the caller to set as a cookie.
func (a *Auth) Login(ctx context.Context, email, password, userAgent, ip string) (identity.User, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	user, hash, err := a.store.UserForAuth(ctx, email)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			// Still hash something, so a missing account and a wrong
			// password take comparable time and the response can't be
			// used to enumerate registered emails.
			_ = bcrypt.CompareHashAndPassword([]byte("$2a$12$invalidinvalidinvalidinvalidinvalidinvalidinvalidinvalidinv"), []byte(password))
			return identity.User{}, "", ErrInvalidCredentials
		}
		return identity.User{}, "", err
	}
	if hash == "" {
		return identity.User{}, "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		_ = a.auditor.Record(ctx, user.ID, user.Name, "Failed sign-in", "session", user.ID)
		return identity.User{}, "", ErrInvalidCredentials
	}

	token, err := newToken()
	if err != nil {
		return identity.User{}, "", err
	}
	if err := a.store.CreateSession(ctx, token, user.ID, userAgent, ip); err != nil {
		return identity.User{}, "", err
	}
	_ = a.store.TouchLogin(ctx, user.ID)
	_ = a.auditor.Record(ctx, user.ID, user.Name, "Signed in", "session", user.ID)

	return user, token, nil
}

// Logout revokes the session server-side. Clearing the cookie alone would
// leave a working token in anyone's hands who had copied it.
func (a *Auth) Logout(ctx context.Context, token string, user identity.User) error {
	if err := a.store.DeleteSession(ctx, token); err != nil {
		return err
	}
	if user.ID != "" {
		_ = a.auditor.Record(ctx, user.ID, user.Name, "Signed out", "session", user.ID)
	}
	return nil
}

// UserForSession satisfies identity.Authenticator for the middleware.
func (a *Auth) UserForSession(ctx context.Context, token string) (identity.User, error) {
	u, err := a.store.UserForSession(ctx, token)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return identity.User{}, httpx.ErrUnauthorized
		}
		return identity.User{}, err
	}
	return u, nil
}

// HashPassword is used by the seeder and by admin user creation.
func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(h), err
}

func (a *Auth) SetPassword(ctx context.Context, userID, plain string) error {
	h, err := HashPassword(plain)
	if err != nil {
		return err
	}
	return a.store.SetPassword(ctx, userID, h)
}

func (a *Auth) PurgeExpiredSessions(ctx context.Context) error { return a.store.PurgeExpired(ctx) }
