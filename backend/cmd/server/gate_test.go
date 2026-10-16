package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/billing"

	"github.com/go-chi/chi/v5"
)

// subscriber is a signed-in principal in an organisation — enough for
// RequireSession and orgScope to let the request reach the gate, which
// is the thing under test.
type subscriber struct{}

func (subscriber) Organisation() string { return "default" }
func (subscriber) IsClient() bool       { return false }
func (subscriber) PrincipalID() string  { return "user-1" }

type sessionAuth struct{}

func (sessionAuth) PublicRoutes(chi.Router) {}
func (sessionAuth) Authenticator() httpx.SessionAuthenticator {
	return resolvingSessions{}
}

type resolvingSessions struct{}

func (resolvingSessions) UserForSession(context.Context, string) (httpx.Principal, error) {
	return subscriber{}, nil
}

// gated is a stand-in for every service the gate sits in front of.
type gated struct{ hits *int }

func (g gated) Routes(r chi.Router) {
	r.Get("/campaigns", func(w http.ResponseWriter, _ *http.Request) {
		*g.hits++
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"ok": "data"})
	})
}

func gateRouter(t *testing.T, gate subscriptionGate, hits *int) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return newRouter(
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
		sessionAuth{}, stubGrants{},
		gate,
		// Billing's own routes, standing in for the real ones.
		func(r chi.Router) {
			r.Get("/billing", func(w http.ResponseWriter, _ *http.Request) {
				httpx.WriteJSON(w, http.StatusOK, map[string]string{"plan": "none"})
			})
		},
		nil,
		rateLimits{APIPerMinute: 600, AuthPerMinute: 600}, logger,
		gated{hits: hits},
	)
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(&http.Cookie{Name: "ctp_session", Value: "token"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGateLocksTheAppWithoutASubscription(t *testing.T) {
	cases := []struct {
		name string
		sub  billing.Subscription
		want int
	}{
		{"no subscription", billing.Subscription{Status: billing.StatusNone}, http.StatusPaymentRequired},
		{"trial expired", billing.Subscription{
			Status: billing.StatusTrialing, TrialEndsAt: time.Now().Add(-time.Hour),
		}, http.StatusPaymentRequired},
		{"cancelled", billing.Subscription{Status: billing.StatusCancelled}, http.StatusPaymentRequired},
		{"active", billing.Subscription{
			Status: billing.StatusActive, CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
		}, http.StatusOK},
		{"grandfathered, no expiry", billing.Subscription{Status: billing.StatusActive}, http.StatusOK},
		{"trialing", billing.Subscription{
			Status: billing.StatusTrialing, TrialEndsAt: time.Now().Add(time.Hour),
		}, http.StatusOK},
	}

	for _, c := range cases {
		hits := 0
		sub := c.sub
		h := gateRouter(t, func(context.Context, string) (billing.Subscription, bool, error) {
			return sub, sub.Allows(time.Now()), nil
		}, &hits)

		rec := get(t, h, "/api/campaigns")
		if rec.Code != c.want {
			t.Errorf("%s: GET /api/campaigns = %d, want %d (%s)", c.name, rec.Code, c.want, rec.Body.String())
		}
		// The handler must not have run at all on a refusal — a gate
		// that lets the work happen and discards the response has
		// already leaked whatever the work touched.
		if c.want == http.StatusPaymentRequired && hits != 0 {
			t.Errorf("%s: the handler ran %d times behind a closed gate", c.name, hits)
		}
	}
}

// A locked organisation has to be able to reach the thing that unlocks
// it, or the paywall is a door that locks from the inside.
func TestBillingStaysReachableWhileLocked(t *testing.T) {
	hits := 0
	h := gateRouter(t, func(context.Context, string) (billing.Subscription, bool, error) {
		return billing.Subscription{Status: billing.StatusNone}, false, nil
	}, &hits)

	if rec := get(t, h, "/api/billing"); rec.Code != http.StatusOK {
		t.Errorf("GET /api/billing while locked = %d, want 200 — there would be no way to pay", rec.Code)
	}
}

// A database that cannot say whether the tenant paid is not grounds for
// assuming it did.
func TestGateFailsClosed(t *testing.T) {
	hits := 0
	h := gateRouter(t, func(context.Context, string) (billing.Subscription, bool, error) {
		return billing.Subscription{}, false, errors.New("database is down")
	}, &hits)

	rec := get(t, h, "/api/campaigns")
	if rec.Code == http.StatusOK {
		t.Error("the gate let a request through while it could not check the subscription")
	}
	if hits != 0 {
		t.Error("the handler ran despite the check failing")
	}
}
