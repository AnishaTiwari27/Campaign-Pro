package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/database/dbtest"
	"campaigntrackerpro/platform/scope"
	campaignsmod "campaigntrackerpro/services/campaigns/module"
	identitymod "campaigntrackerpro/services/identity/module"
)

// Signing up and then signing in is tested here, in the composition
// root, rather than in identity — because the bug this guards against
// was not in identity's logic. It was in the wiring: signup is a public
// route, mounted outside the tenant middleware, so nothing had resolved
// an organisation by the time the handler ran. The insert went in with
// org_id = "" and Postgres refused it on the foreign key, which reached
// the user as a 500 on a form they had filled in correctly.
//
// Identity's own tests all passed, because every one of them stops at
// validateSignup and never writes a row. Only the assembled thing can
// show this, so the test lives where the thing is assembled.
func TestSignupThenSignIn(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pool, err := database.Connect(context.Background(), dbtest.DSN())
	if err != nil {
		t.Skipf("skipping: no local Postgres available (%v)", err)
	}
	t.Cleanup(pool.Close)
	db := database.New(pool)

	campaignsMod := campaignsmod.New(db, logger)
	identityMod := identitymod.New(db, auditBridge{campaignsMod}, campaignsMod.UserDirectory(),
		identitymod.Options{
			AllowSignup: true,
			// Exactly what the server passes in production.
			SignupOrgID: scope.DefaultOrg,
		}, logger)

	h := newRouter(
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
		identityMod, campaignsMod.Grants(),
		rateLimits{APIPerMinute: 600, AuthPerMinute: 600}, logger,
	)

	email := fmt.Sprintf("signup-%d@example.test", time.Now().UnixNano())
	const password = "correct horse battery staple"
	body := fmt.Sprintf(`{"name":"Asha Rao","email":%q,"password":%q}`, email, password)

	t.Cleanup(func() {
		ctx := scope.SystemOrg(context.Background())
		_, _ = pool.Exec(ctx, `DELETE FROM audit_events WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, email)
		_, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, email)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email)
	})

	rec := call(t, h, "/api/auth/signup", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup with valid details returned %d: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("signup response is not the user DTO: %v", err)
	}
	if created.ID == "" || created.Email != email {
		t.Fatalf("signup returned %+v, want the new account", created)
	}

	// The account has to land in a real organisation, or it signs in to
	// an empty app: every read below the tenant middleware is confined
	// to the org on the principal.
	var orgID string
	if err := pool.QueryRow(context.Background(),
		`SELECT org_id FROM users WHERE email = $1`, email).Scan(&orgID); err != nil {
		t.Fatalf("reading back the new account: %v", err)
	}
	if orgID != scope.DefaultOrg {
		t.Fatalf("new account is in organisation %q, want %q", orgID, scope.DefaultOrg)
	}

	// The password is written by a second statement, so a signup that
	// reports success but leaves no credential is a real failure mode.
	// Signing in is the only thing that proves it was stored.
	if rec := call(t, h, "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)); rec.Code != http.StatusOK {
		t.Fatalf("signing in with the password just chosen returned %d: %s", rec.Code, rec.Body.String())
	}

	// The duplicate check runs inside the organisation too; with no org
	// it found nothing and the clear message was replaced by whatever
	// the unique index said.
	if rec := call(t, h, "/api/auth/signup", body); rec.Code != http.StatusConflict {
		t.Fatalf("re-registering the same email returned %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

func call(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
