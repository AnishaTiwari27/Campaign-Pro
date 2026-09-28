package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"campaigntrackerpro/platform"
)

var testSecret = []byte("gateway-test-secret")

func tokenFor(t *testing.T, userID, email, role string, ttl time.Duration) string {
	t.Helper()
	tok, err := platform.SignAccessToken(testSecret, userID, email, role, ttl)
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}
	return tok
}

func TestRequireAuth_ValidToken_SetsTrustedHeaders(t *testing.T) {
	var gotHeaders http.Header
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	})
	handler := RequireAuth(testSecret, next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "u1", "a@b.com", "admin", time.Minute))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if gotHeaders.Get("X-User-Id") != "u1" || gotHeaders.Get("X-User-Email") != "a@b.com" || gotHeaders.Get("X-User-Role") != "admin" {
		t.Fatalf("trusted headers = %+v, want u1/a@b.com/admin", gotHeaders)
	}
}

func TestRequireAuth_StripsSpoofedHeaders(t *testing.T) {
	var gotHeaders http.Header
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	})
	handler := RequireAuth(testSecret, next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "u1", "a@b.com", "viewer", time.Minute))
	req.Header.Set("X-User-Role", "admin") // a caller trying to forge admin — must be overwritten, not trusted
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if gotHeaders.Get("X-User-Role") != "viewer" {
		t.Fatalf("X-User-Role = %q, want the verified claim (viewer) to win over the spoofed header", gotHeaders.Get("X-User-Role"))
	}
}

func TestRequireAuth_Rejections(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called when auth fails")
	})
	handler := RequireAuth(testSecret, next)

	cases := []struct {
		name string
		auth string
	}{
		{"missing header", ""},
		{"not bearer", "Basic dXNlcjpwYXNz"},
		{"garbage token", "Bearer not-a-jwt"},
		{"expired token", "Bearer " + tokenFor(t, "u1", "a@b.com", "admin", -time.Minute)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", nil)
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rr.Code)
			}
		})
	}
}

func TestRequireAuthQuery_ValidAndMissingToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := RequireAuthQuery(testSecret, next)

	ok := httptest.NewRequest(http.MethodGet, "/ws/notifications?token="+tokenFor(t, "u1", "a@b.com", "viewer", time.Minute), nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, ok)
	if rr.Code != http.StatusOK {
		t.Fatalf("with a valid ?token=, status = %d, want 200", rr.Code)
	}

	missing := httptest.NewRequest(http.MethodGet, "/ws/notifications", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, missing)
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("with no token query param, status = %d, want 401", rr2.Code)
	}
}
