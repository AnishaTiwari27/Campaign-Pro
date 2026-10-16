package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"campaigntrackerpro/platform/httpx"

	"github.com/go-chi/chi/v5"
)

type stubAuth struct{}

func (stubAuth) PublicRoutes(r chi.Router) {
	r.Post("/auth/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized) // every attempt here is a miss
	})
}
func (stubAuth) Authenticator() httpx.SessionAuthenticator { return stubSessions{} }

type stubSessions struct{}

func (stubSessions) UserForSession(context.Context, string) (httpx.Principal, error) {
	return nil, httpx.ErrNotFound
}

type stubGrants struct{}

func (stubGrants) For(context.Context, string) ([]string, error) { return nil, nil }

func post(t *testing.T, h http.Handler, forwardedFor string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Header.Set("X-Forwarded-For", forwardedFor)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// The limiter has to key on the forwarded address rather than on
// RemoteAddr. Behind a proxy every request carries the proxy's
// RemoteAddr, so keying on that would put the whole internet in one
// bucket and let a single noisy client lock everybody out. This is the
// test that catches it: two clients, each inside the budget, must both
// still be served.
func TestAuthLimitIsPerClientNotPerProxy(t *testing.T) {
	h := newRouter(
		func(http.ResponseWriter, *http.Request) {},
		stubAuth{}, stubGrants{},
		// No gate: these exercise the rate limiter, and a paywall in
		// front of it would be testing two things at once.
		nil, nil, nil,
		rateLimits{APIPerMinute: 1000, AuthPerMinute: 3},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	for i := 1; i <= 3; i++ {
		if got := post(t, h, "203.0.113.5"); got != http.StatusUnauthorized {
			t.Fatalf("attempt %d from the first client: got %d, want 401", i, got)
		}
	}
	if got := post(t, h, "203.0.113.5"); got != http.StatusTooManyRequests {
		t.Errorf("fourth attempt from the first client: got %d, want 429", got)
	}
	if got := post(t, h, "198.51.100.9"); got != http.StatusUnauthorized {
		t.Errorf("first attempt from a second client: got %d, want 401 — the buckets are shared", got)
	}
}

// A refusal must still be JSON: it would otherwise be the only non-JSON
// body the API can return, and would reach the client's parser as a
// surprise at exactly the moment things are going wrong.
func TestRateLimitRefusalIsJSON(t *testing.T) {
	h := newRouter(
		func(http.ResponseWriter, *http.Request) {},
		stubAuth{}, stubGrants{},
		// No gate: these exercise the rate limiter, and a paywall in
		// front of it would be testing two things at once.
		nil, nil, nil,
		rateLimits{APIPerMinute: 1000, AuthPerMinute: 1},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	post(t, h, "203.0.113.7")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type %q, want application/json", ct)
	}
	if body, _ := io.ReadAll(rec.Body); len(body) == 0 || body[0] != '{' {
		t.Errorf("body %q is not a JSON object", body)
	}
}
