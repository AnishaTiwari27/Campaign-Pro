package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/auth/internal/models"
)

const minPasswordLength = 8

// handleRegister creates a new "viewer" account and, on success, logs it
// straight in (same response shape as handleLogin) — one less step for a
// new user, and consistent with there being no email-verification step in
// this system. Every self-registered account is a viewer; there is no
// request field that could make it an admin (see models.RegisterRequest).
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.BadRequest(w, "invalid JSON body")
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		platform.BadRequest(w, "a valid email is required")
		return
	}
	if len(req.Password) < minPasswordLength {
		platform.BadRequest(w, "password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("register: hash password", "err", err)
		platform.Internal(w, "failed to register")
		return
	}

	user, err := s.Store.CreateUser(r.Context(), req.Email, string(hash))
	if err != nil {
		if errors.Is(err, models.ErrEmailTaken) {
			platform.WriteError(w, http.StatusConflict, "email_taken", "an account with this email already exists")
			return
		}
		slog.Error("register: create user", "err", err)
		platform.Internal(w, "failed to register")
		return
	}

	pair, err := s.issueTokenPair(r.Context(), user)
	if err != nil {
		slog.Error("register: issue tokens", "err", err)
		platform.Internal(w, "failed to register")
		return
	}
	platform.WriteJSON(w, http.StatusCreated, pair)
}

// handleLogin verifies email+password and issues a fresh access/refresh
// pair. The same 401 ("invalid email or password") is returned whether the
// email doesn't exist or the password is wrong — distinguishing the two
// would let a caller enumerate registered emails.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.BadRequest(w, "invalid JSON body")
		return
	}
	if req.Email == "" || req.Password == "" {
		platform.BadRequest(w, "email and password are required")
		return
	}

	user, found, err := s.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		slog.Error("login: lookup user", "err", err)
		platform.Internal(w, "failed to log in")
		return
	}
	if !found || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		platform.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}

	pair, err := s.issueTokenPair(r.Context(), user)
	if err != nil {
		slog.Error("login: issue tokens", "err", err)
		platform.Internal(w, "failed to log in")
		return
	}
	platform.WriteJSON(w, http.StatusOK, pair)
}

// handleRefresh consumes req.RefreshToken (single-use — see
// Store.ConsumeRefreshToken) and, if it's valid and unexpired, issues a
// brand new access/refresh pair for the same user.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		platform.BadRequest(w, "refreshToken is required")
		return
	}

	userID, ok, err := s.Store.ConsumeRefreshToken(r.Context(), hashToken(req.RefreshToken))
	if err != nil {
		slog.Error("refresh: consume token", "err", err)
		platform.Internal(w, "failed to refresh")
		return
	}
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token is invalid, expired, or already used")
		return
	}

	user, found, err := s.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		slog.Error("refresh: lookup user", "err", err)
		platform.Internal(w, "failed to refresh")
		return
	}
	if !found {
		// The user was deleted after this refresh token was issued.
		platform.WriteError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token is invalid, expired, or already used")
		return
	}

	pair, err := s.issueTokenPair(r.Context(), user)
	if err != nil {
		slog.Error("refresh: issue tokens", "err", err)
		platform.Internal(w, "failed to refresh")
		return
	}
	platform.WriteJSON(w, http.StatusOK, pair)
}

// handleLogout revokes req.RefreshToken. Always 204 — logging out a token
// that's already gone (or was never valid) isn't an error from the
// caller's side; they end this request with no usable session either way.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req models.LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		platform.BadRequest(w, "refreshToken is required")
		return
	}
	if err := s.Store.DeleteRefreshToken(r.Context(), hashToken(req.RefreshToken)); err != nil {
		slog.Error("logout: delete token", "err", err)
		platform.Internal(w, "failed to log out")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleListUsers backs GET /api/v1/users — admin-only, no pagination
// (this app's user count is nowhere near needing it). Never returns
// PasswordHash — every row goes through User.AsJSON() first.
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if !platform.RequireRole(w, r, "only admins can list users", platform.RoleAdmin) {
		return
	}
	users, err := s.Store.ListUsers(r.Context())
	if err != nil {
		slog.Error("list users", "err", err)
		platform.Internal(w, "failed to list users")
		return
	}
	out := make([]models.UserJSON, len(users))
	for i, u := range users {
		out[i] = u.AsJSON()
	}
	platform.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// handleUpdateUserRole backs PATCH /api/v1/users/{id}/role — admin-only.
// Rejects an admin targeting their own account: a cheap, real guard
// against accidentally locking yourself out, not full last-admin
// protection (which would need a count query this app's scale doesn't
// need yet).
func (s *Server) handleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	if !platform.RequireRole(w, r, "only admins can change a user's role", platform.RoleAdmin) {
		return
	}
	id := r.PathValue("id")
	if id == r.Header.Get("X-User-Id") {
		platform.BadRequest(w, "cannot change your own role — ask another admin")
		return
	}

	var req models.UpdateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.BadRequest(w, "invalid JSON body")
		return
	}
	if !platform.RoleValid(req.Role) {
		platform.BadRequest(w, `role must be one of "viewer", "editor", "approver", "admin"`)
		return
	}

	updated, ok, err := s.Store.UpdateUserRole(r.Context(), id, req.Role)
	if err != nil {
		slog.Error("update user role", "id", id, "err", err)
		platform.Internal(w, "failed to update role")
		return
	}
	if !ok {
		platform.NotFound(w, "no user with that id")
		return
	}
	platform.WriteJSON(w, http.StatusOK, updated.AsJSON())
}

// issueTokenPair signs a fresh access token and mints+stores a fresh
// refresh token for user — the one place both /auth/login and
// /auth/refresh build a response, so the two can't drift.
func (s *Server) issueTokenPair(ctx context.Context, user models.User) (models.TokenPair, error) {
	access, err := platform.SignAccessToken(s.JWTSecret, user.ID, user.Email, user.Role, s.AccessTTL)
	if err != nil {
		return models.TokenPair{}, err
	}

	refresh, err := newOpaqueToken()
	if err != nil {
		return models.TokenPair{}, err
	}
	if err := s.Store.SaveRefreshToken(ctx, hashToken(refresh), user.ID, time.Now().Add(s.RefreshTTL)); err != nil {
		return models.TokenPair{}, err
	}

	return models.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}
