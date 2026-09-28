package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"campaigntrackerpro/services/auth/internal/models"
)

// fakeStore is an in-memory UserStore — one fixed user, refresh tokens kept
// in a map keyed by hash. Enough to exercise login/refresh/logout without a
// real Postgres.
type fakeStore struct {
	usersByEmail map[string]models.User
	usersByID    map[string]models.User
	refreshToks  map[string]struct {
		userID    string
		expiresAt time.Time
	}
}

func (f *fakeStore) Ping(context.Context) error { return nil }

func (f *fakeStore) GetUserByEmail(_ context.Context, email string) (models.User, bool, error) {
	u, ok := f.usersByEmail[email]
	return u, ok, nil
}

func (f *fakeStore) GetUserByID(_ context.Context, id string) (models.User, bool, error) {
	u, ok := f.usersByID[id]
	return u, ok, nil
}

func (f *fakeStore) SaveRefreshToken(_ context.Context, tokenHash, userID string, expiresAt time.Time) error {
	f.refreshToks[tokenHash] = struct {
		userID    string
		expiresAt time.Time
	}{userID, expiresAt}
	return nil
}

func (f *fakeStore) ConsumeRefreshToken(_ context.Context, tokenHash string) (string, bool, error) {
	rec, ok := f.refreshToks[tokenHash]
	if !ok || time.Now().After(rec.expiresAt) {
		return "", false, nil
	}
	delete(f.refreshToks, tokenHash) // single-use — see docs/ARCHITECTURE.md
	return rec.userID, true, nil
}

func (f *fakeStore) DeleteRefreshToken(_ context.Context, tokenHash string) error {
	delete(f.refreshToks, tokenHash)
	return nil
}

func (f *fakeStore) CreateUser(_ context.Context, email, passwordHash string) (models.User, error) {
	if _, taken := f.usersByEmail[email]; taken {
		return models.User{}, models.ErrEmailTaken
	}
	u := models.User{ID: fmt.Sprintf("u%d", len(f.usersByEmail)+1), Email: email, PasswordHash: passwordHash, Role: "viewer"}
	f.usersByEmail[email] = u
	f.usersByID[u.ID] = u
	return u, nil
}

func (f *fakeStore) ListUsers(context.Context) ([]models.User, error) {
	out := make([]models.User, 0, len(f.usersByID))
	for _, u := range f.usersByID {
		out = append(out, u)
	}
	return out, nil
}

func (f *fakeStore) UpdateUserRole(_ context.Context, id, role string) (models.User, bool, error) {
	u, ok := f.usersByID[id]
	if !ok {
		return models.User{}, false, nil
	}
	u.Role = role
	f.usersByID[id] = u
	f.usersByEmail[u.Email] = u
	return u, true, nil
}

const testPassword = "correct horse battery staple"

func newTestServer(t *testing.T) (*Server, *fakeStore) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	user := models.User{ID: "u1", Email: "a@b.com", PasswordHash: string(hash), Role: "admin"}
	fs := &fakeStore{
		usersByEmail: map[string]models.User{user.Email: user},
		usersByID:    map[string]models.User{user.ID: user},
		refreshToks: map[string]struct {
			userID    string
			expiresAt time.Time
		}{},
	}
	return &Server{Store: fs, JWTSecret: []byte("test-secret"), AccessTTL: time.Minute, RefreshTTL: time.Hour}, fs
}

func postJSON(t *testing.T, path string, body any) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
}

func TestHandleLogin_WrongPasswordAndUnknownEmailBothReject(t *testing.T) {
	s, _ := newTestServer(t)

	for _, tc := range []models.LoginRequest{
		{Email: "a@b.com", Password: "wrong"},
		{Email: "nobody@b.com", Password: testPassword},
	} {
		rr := httptest.NewRecorder()
		s.handleLogin(rr, postJSON(t, "/auth/login", tc))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("login(%+v): status = %d, want 401", tc, rr.Code)
		}
	}
}

func TestHandleLogin_Success(t *testing.T) {
	s, fs := newTestServer(t)
	rr := httptest.NewRecorder()
	s.handleLogin(rr, postJSON(t, "/auth/login", models.LoginRequest{Email: "a@b.com", Password: testPassword}))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
	var pair models.TokenPair
	if err := json.Unmarshal(rr.Body.Bytes(), &pair); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected both tokens set, got %+v", pair)
	}
	if len(fs.refreshToks) != 1 {
		t.Fatalf("expected exactly one refresh token persisted, got %d", len(fs.refreshToks))
	}
}

func TestHandleRefresh_RotatesAndInvalidatesOldToken(t *testing.T) {
	s, _ := newTestServer(t)

	rr := httptest.NewRecorder()
	s.handleLogin(rr, postJSON(t, "/auth/login", models.LoginRequest{Email: "a@b.com", Password: testPassword}))
	var first models.TokenPair
	_ = json.Unmarshal(rr.Body.Bytes(), &first)

	rr2 := httptest.NewRecorder()
	s.handleRefresh(rr2, postJSON(t, "/auth/refresh", models.RefreshRequest{RefreshToken: first.RefreshToken}))
	if rr2.Code != http.StatusOK {
		t.Fatalf("first refresh: status = %d, want 200", rr2.Code)
	}
	var second models.TokenPair
	_ = json.Unmarshal(rr2.Body.Bytes(), &second)
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("expected refresh to rotate to a new refresh token, got the same one back")
	}

	// Replaying the original (now-consumed) refresh token must fail.
	rr3 := httptest.NewRecorder()
	s.handleRefresh(rr3, postJSON(t, "/auth/refresh", models.RefreshRequest{RefreshToken: first.RefreshToken}))
	if rr3.Code != http.StatusUnauthorized {
		t.Fatalf("replayed refresh token: status = %d, want 401", rr3.Code)
	}
}

func TestHandleLogout_RevokesToken(t *testing.T) {
	s, fs := newTestServer(t)

	rr := httptest.NewRecorder()
	s.handleLogin(rr, postJSON(t, "/auth/login", models.LoginRequest{Email: "a@b.com", Password: testPassword}))
	var pair models.TokenPair
	_ = json.Unmarshal(rr.Body.Bytes(), &pair)

	rr2 := httptest.NewRecorder()
	s.handleLogout(rr2, postJSON(t, "/auth/logout", models.LogoutRequest{RefreshToken: pair.RefreshToken}))
	if rr2.Code != http.StatusNoContent {
		t.Fatalf("logout: status = %d, want 204", rr2.Code)
	}
	if len(fs.refreshToks) != 0 {
		t.Fatalf("expected the refresh token to be revoked, %d remain", len(fs.refreshToks))
	}

	// The now-revoked token can no longer refresh a session.
	rr3 := httptest.NewRecorder()
	s.handleRefresh(rr3, postJSON(t, "/auth/refresh", models.RefreshRequest{RefreshToken: pair.RefreshToken}))
	if rr3.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: status = %d, want 401", rr3.Code)
	}
}

func TestHandleRegister_Success(t *testing.T) {
	s, fs := newTestServer(t)
	rr := httptest.NewRecorder()
	s.handleRegister(rr, postJSON(t, "/auth/register", models.RegisterRequest{Email: "new@b.com", Password: "a-fine-password"}))

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s, want 201", rr.Code, rr.Body.String())
	}
	var pair models.TokenPair
	if err := json.Unmarshal(rr.Body.Bytes(), &pair); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected register to log the new account straight in, got %+v", pair)
	}
	if got := fs.usersByEmail["new@b.com"].Role; got != "viewer" {
		t.Fatalf("stored role = %q, want every self-registered account to be viewer", got)
	}
}

// A JSON body can carry a "role" key even though models.RegisterRequest has
// no such field — this locks in that the extra key is simply dropped by
// the decoder, not silently honored, so a client can't smuggle admin in.
func TestHandleRegister_IgnoresExtraRoleField(t *testing.T) {
	s, fs := newTestServer(t)
	body := bytes.NewReader([]byte(`{"email":"sneaky@b.com","password":"a-fine-password","role":"admin"}`))
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	rr := httptest.NewRecorder()
	s.handleRegister(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rr.Code)
	}
	if got := fs.usersByEmail["sneaky@b.com"].Role; got != "viewer" {
		t.Fatalf("stored role = %q, want viewer regardless of a role field in the request body", got)
	}
}

func TestHandleRegister_DuplicateEmailConflicts(t *testing.T) {
	s, _ := newTestServer(t)
	rr := httptest.NewRecorder()
	s.handleRegister(rr, postJSON(t, "/auth/register", models.RegisterRequest{Email: "a@b.com", Password: "a-fine-password"}))

	if rr.Code != http.StatusConflict {
		t.Fatalf("registering an already-taken email: status = %d, want 409", rr.Code)
	}
}

func TestHandleRegister_ValidatesInput(t *testing.T) {
	s, _ := newTestServer(t)

	cases := []models.RegisterRequest{
		{Email: "", Password: "a-fine-password"},
		{Email: "not-an-email", Password: "a-fine-password"},
		{Email: "short@b.com", Password: "short"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		s.handleRegister(rr, postJSON(t, "/auth/register", tc))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("register(%+v): status = %d, want 400", tc, rr.Code)
		}
	}
}

func withRole(req *http.Request, role, userID string) *http.Request {
	req.Header.Set("X-User-Role", role)
	req.Header.Set("X-User-Id", userID)
	return req
}

func TestHandleListUsers_RequiresAdminRole(t *testing.T) {
	s, _ := newTestServer(t)
	req := withRole(httptest.NewRequest(http.MethodGet, "/api/v1/users", nil), "viewer", "u1")
	rr := httptest.NewRecorder()
	s.handleListUsers(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer list users: status = %d, want 403", rr.Code)
	}
}

func TestHandleListUsers_AdminSucceeds(t *testing.T) {
	s, _ := newTestServer(t)
	req := withRole(httptest.NewRequest(http.MethodGet, "/api/v1/users", nil), "admin", "u1")
	rr := httptest.NewRecorder()
	s.handleListUsers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("admin list users: status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
	var got struct {
		Data []models.UserJSON `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Data) != 1 || got.Data[0].Email != "a@b.com" {
		t.Fatalf("data = %+v, want the one seeded user", got.Data)
	}
	if bytes.Contains(rr.Body.Bytes(), []byte("PasswordHash")) || bytes.Contains(rr.Body.Bytes(), []byte(testPassword)) {
		t.Fatalf("response leaked password hash material: %s", rr.Body.String())
	}
}

func patchRoleReq(t *testing.T, targetID, callerRole, callerID, newRole string) *http.Request {
	t.Helper()
	req := postJSON(t, "/api/v1/users/"+targetID+"/role", models.UpdateUserRoleRequest{Role: newRole})
	req.Method = http.MethodPatch
	req.SetPathValue("id", targetID)
	return withRole(req, callerRole, callerID)
}

func TestHandleUpdateUserRole_RequiresAdminRole(t *testing.T) {
	s, _ := newTestServer(t)
	rr := httptest.NewRecorder()
	s.handleUpdateUserRole(rr, patchRoleReq(t, "u1", "editor", "u2", "admin"))

	if rr.Code != http.StatusForbidden {
		t.Fatalf("non-admin update role: status = %d, want 403", rr.Code)
	}
}

func TestHandleUpdateUserRole_RejectsSelfTarget(t *testing.T) {
	s, _ := newTestServer(t)
	rr := httptest.NewRecorder()
	s.handleUpdateUserRole(rr, patchRoleReq(t, "u1", "admin", "u1", "viewer"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("admin targeting themselves: status = %d, want 400", rr.Code)
	}
}

func TestHandleUpdateUserRole_RejectsInvalidRole(t *testing.T) {
	s, fs := newTestServer(t)
	fs.usersByID["u2"] = models.User{ID: "u2", Email: "b@b.com", Role: "viewer"}
	rr := httptest.NewRecorder()
	s.handleUpdateUserRole(rr, patchRoleReq(t, "u2", "admin", "u1", "superuser"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid role: status = %d, want 400", rr.Code)
	}
}

func TestHandleUpdateUserRole_NotFound(t *testing.T) {
	s, _ := newTestServer(t)
	rr := httptest.NewRecorder()
	s.handleUpdateUserRole(rr, patchRoleReq(t, "no-such-id", "admin", "u1", "editor"))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("unknown id: status = %d, want 404", rr.Code)
	}
}

func TestHandleUpdateUserRole_AdminSucceeds(t *testing.T) {
	s, fs := newTestServer(t)
	fs.usersByID["u2"] = models.User{ID: "u2", Email: "b@b.com", Role: "viewer"}
	fs.usersByEmail["b@b.com"] = fs.usersByID["u2"]
	rr := httptest.NewRecorder()
	s.handleUpdateUserRole(rr, patchRoleReq(t, "u2", "admin", "u1", "editor"))

	if rr.Code != http.StatusOK {
		t.Fatalf("admin update role: status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
	var got models.UserJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Role != "editor" {
		t.Fatalf("role = %q, want editor", got.Role)
	}
}
