package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimit_ByUser_AllowsBurstThenRejects(t *testing.T) {
	rl := NewRateLimit(1, 3) // 1 req/s sustained, burst of 3
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := rl.ByUser(next)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", nil)
		req.Header.Set("X-User-Id", "u1")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d within burst: status = %d, want 200", i, rr.Code)
		}
	}

	// The burst is exhausted — the next immediate request must be limited.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", nil)
	req.Header.Set("X-User-Id", "u1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429 once the burst is spent", rr.Code)
	}
}

func TestRateLimit_ByUser_SeparateBucketsPerUser(t *testing.T) {
	rl := NewRateLimit(1, 1)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := rl.ByUser(next)

	// Exhaust user "a"'s single-request burst.
	reqA := httptest.NewRequest(http.MethodGet, "/x", nil)
	reqA.Header.Set("X-User-Id", "a")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, reqA)
	if rr.Code != http.StatusOK {
		t.Fatalf("user a's first request: status = %d, want 200", rr.Code)
	}

	// User "b" must be unaffected — a noisy user can't exhaust another's budget.
	reqB := httptest.NewRequest(http.MethodGet, "/x", nil)
	reqB.Header.Set("X-User-Id", "b")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, reqB)
	if rr2.Code != http.StatusOK {
		t.Fatalf("user b's first request: status = %d, want 200 (separate bucket from user a)", rr2.Code)
	}
}

func TestRateLimit_ByIP_KeysOnRemoteAddr(t *testing.T) {
	rl := NewRateLimit(1, 1)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := rl.ByIP(next)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "203.0.113.5:54321"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first attempt: status = %d, want 200", rr.Code)
	}

	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req) // same IP, burst of 1 already spent
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second immediate attempt from the same IP: status = %d, want 429", rr2.Code)
	}
}
