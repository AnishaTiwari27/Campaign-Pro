// Package models holds auth-service's domain and JSON-facing types.
package models

import (
	"errors"
	"time"
)

// ErrEmailTaken is returned by UserStore.CreateUser when the email already
// has an account — lives here (not in store) so the api package can
// errors.Is against it without importing the concrete store package,
// consistent with api depending only on the UserStore interface.
var ErrEmailTaken = errors.New("email already registered")

// User is one row of auth.users. PasswordHash never appears in a JSON tag —
// it must never leave this service, not even by accident via an encoder
// that reflects over exported fields with no explicit tag.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string // "viewer" | "editor" | "approver" | "admin" — see platform.RoleValid
}

// UserJSON is User's public view — never PasswordHash. Mirrors
// campaigns-service's Campaign/campaignJSON/AsJSON split exactly.
type UserJSON struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (u User) AsJSON() UserJSON {
	return UserJSON{ID: u.ID, Email: u.Email, Role: u.Role}
}

// UpdateUserRoleRequest is the PATCH /api/v1/users/{id}/role body —
// admin-only (see handleUpdateUserRole). Under /api/v1/, not /auth/,
// deliberately: every other /auth/* route is pre-identity/public at the
// gateway, but this needs an already-verified admin session.
type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

// LoginRequest is the POST /auth/login body.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest is the POST /auth/register body. Deliberately has no Role
// field — every self-registered account is "viewer"; there is no way for a
// caller to request "admin" through this route, not just a validation rule
// against one (see handleRegister).
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest is the POST /auth/refresh body.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// LogoutRequest is the POST /auth/logout body.
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// TokenPair is what /auth/login and /auth/refresh both return. refresh is
// rotated on every /auth/refresh call (the old one is deleted server-side in
// the same request) — a stolen refresh token stops working the moment its
// legitimate owner refreshes again, not just at its 30-day expiry. This is
// one documented deviation from docs/API_CONTRACT.md's original sketch
// (which only had /auth/refresh return accessToken) — same kind of
// deliberate, contract-updated change as campaigns-service's brand->subject
// rename.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// RefreshTokenTTL and AccessTokenTTL are the defaults main.go falls back to
// when the corresponding env var isn't set.
const (
	DefaultAccessTokenTTL  = 15 * time.Minute
	DefaultRefreshTokenTTL = 30 * 24 * time.Hour
)
