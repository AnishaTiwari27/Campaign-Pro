package platform

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

// countingTripper is a fake RoundTripper that returns whatever's next in
// results (looping on the last entry once exhausted) and counts calls.
type countingTripper struct {
	calls   int
	results []roundTripResult
}

type roundTripResult struct {
	status int
	err    error
}

func (c *countingTripper) RoundTrip(*http.Request) (*http.Response, error) {
	i := c.calls
	if i >= len(c.results) {
		i = len(c.results) - 1
	}
	c.calls++
	r := c.results[i]
	if r.err != nil {
		return nil, r.err
	}
	return &http.Response{StatusCode: r.status, Body: http.NoBody}, nil
}

func req(t *testing.T) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodGet, "http://upstream.example/x", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	return r
}

func TestCircuitBreaker_OpensAfterThresholdAndShortCircuits(t *testing.T) {
	fake := &countingTripper{results: []roundTripResult{{err: errors.New("connection refused")}}}
	b := newCircuitBreaker(fake, 3, time.Hour) // long cooldown — this test never waits it out

	for i := 0; i < 3; i++ {
		if _, err := b.RoundTrip(req(t)); err == nil {
			t.Fatalf("attempt %d: expected the underlying failure to surface, got nil error", i)
		}
	}
	if fake.calls != 3 {
		t.Fatalf("expected 3 real network calls before the breaker opens, got %d", fake.calls)
	}

	// The breaker should now be open: the next call must fail without
	// touching the network at all.
	_, err := b.RoundTrip(req(t))
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen once the threshold is hit, got %v", err)
	}
	if fake.calls != 3 {
		t.Fatalf("expected no additional network call while open, calls = %d", fake.calls)
	}
}

func TestCircuitBreaker_HalfOpenProbeRecoversOnSuccess(t *testing.T) {
	fake := &countingTripper{results: []roundTripResult{{err: errors.New("boom")}}}
	b := newCircuitBreaker(fake, 1, 10*time.Millisecond)

	if _, err := b.RoundTrip(req(t)); err == nil {
		t.Fatal("expected the first failure to open the breaker (threshold 1)")
	}
	if _, err := b.RoundTrip(req(t)); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected open, got %v", err)
	}

	time.Sleep(15 * time.Millisecond) // let the cooldown elapse
	fake.results = []roundTripResult{{status: http.StatusOK}}

	resp, err := b.RoundTrip(req(t)) // the half-open probe
	if err != nil {
		t.Fatalf("expected the probe to reach the network and succeed, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// Closed again — a subsequent call should reach the network normally.
	if _, err := b.RoundTrip(req(t)); err != nil {
		t.Fatalf("expected the breaker to stay closed after a successful probe, got %v", err)
	}
}

func TestCircuitBreaker_4xxIsNotAFailure(t *testing.T) {
	fake := &countingTripper{results: []roundTripResult{{status: http.StatusNotFound}}}
	b := newCircuitBreaker(fake, 1, time.Hour)

	for i := 0; i < 5; i++ {
		resp, err := b.RoundTrip(req(t))
		if err != nil {
			t.Fatalf("attempt %d: a 404 must not open the breaker, got err %v", i, err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}
	}
	if fake.calls != 5 {
		t.Fatalf("expected every call to reach the network (breaker never opened), got %d calls", fake.calls)
	}
}
