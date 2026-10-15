package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func server(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	return s
}

func TestATokenIsMintedAndThenReused(t *testing.T) {
	var calls int
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-1", "expires_in": 3600})
	})
	src := &RefreshTokenSource{TokenURL: s.URL, RefreshToken: "r", HTTP: s.Client()}

	for i := 0; i < 3; i++ {
		got, err := src.Token(context.Background())
		if err != nil || got != "tok-1" {
			t.Fatalf("got %q, %v", got, err)
		}
	}
	if calls != 1 {
		t.Errorf("hit the token endpoint %d times for one valid token, want 1", calls)
	}
}

// A token is renewed before it expires, not after. Expiring mid-flight
// turns an otherwise fine sync into a 401.
func TestATokenNearingExpiryIsRenewedEarly(t *testing.T) {
	var calls int
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 30})
	})
	src := &RefreshTokenSource{TokenURL: s.URL, RefreshToken: "r", HTTP: s.Client()}

	if _, err := src.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	// 30s of life is inside the 60s slack, so the next ask refreshes.
	if _, err := src.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("calls %d, want 2 — a token inside the refresh window was reused", calls)
	}
}

// A revoked refresh token needs a human to re-authorise, so the
// provider's own error has to survive rather than becoming "failed".
func TestARevokedRefreshTokenSaysSo(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_grant", "error_description": "Token has been expired or revoked.",
		})
	})
	src := &RefreshTokenSource{TokenURL: s.URL, RefreshToken: "r", HTTP: s.Client()}

	_, err := src.Token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("got %v, want the provider's own error", err)
	}
}

// Google omits expires_in in some responses; a zero must not mean a
// token that is already expired and refreshes on every call.
func TestAMissingExpiryFallsBackToAnHour(t *testing.T) {
	var calls int
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
	})
	src := &RefreshTokenSource{TokenURL: s.URL, RefreshToken: "r", HTTP: s.Client()}
	src.Token(context.Background())
	src.Token(context.Background())
	if calls != 1 {
		t.Errorf("calls %d, want 1 — a missing expires_in was treated as expired", calls)
	}
}

// Two syncs starting together must not both refresh, and must not hand
// each other a half-written token.
func TestConcurrentCallersShareOneRefresh(t *testing.T) {
	var mu sync.Mutex
	var calls int
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 3600})
	})
	src := &RefreshTokenSource{TokenURL: s.URL, RefreshToken: "r", HTTP: s.Client()}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); src.Token(context.Background()) }()
	}
	wg.Wait()
	if calls != 1 {
		t.Errorf("refreshed %d times concurrently, want 1", calls)
	}
}
