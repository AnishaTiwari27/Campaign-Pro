package platform

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestRetryTransport_RetriesTransientGETFailureAndSucceeds(t *testing.T) {
	fake := &countingTripper{results: []roundTripResult{
		{err: errors.New("connection reset")},
		{status: http.StatusOK},
	}}
	rt := newRetryTransport(fake, 2, time.Millisecond)

	resp, err := rt.RoundTrip(req(t))
	if err != nil {
		t.Fatalf("expected the second attempt to succeed, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if fake.calls != 2 {
		t.Fatalf("expected exactly 2 calls (1 failure + 1 success), got %d", fake.calls)
	}
}

func TestRetryTransport_GivesUpAfterCap(t *testing.T) {
	fake := &countingTripper{results: []roundTripResult{{err: errors.New("always fails")}}}
	rt := newRetryTransport(fake, 2, time.Millisecond) // 1 initial + 2 retries = 3 attempts

	if _, err := rt.RoundTrip(req(t)); err == nil {
		t.Fatal("expected the final error to surface once retries are exhausted")
	}
	if fake.calls != 3 {
		t.Fatalf("expected 3 total attempts (initial + 2 retries), got %d", fake.calls)
	}
}

func TestRetryTransport_NeverRetriesNonGET(t *testing.T) {
	fake := &countingTripper{results: []roundTripResult{{err: errors.New("boom")}}}
	rt := newRetryTransport(fake, 2, time.Millisecond)

	postReq, err := http.NewRequest(http.MethodPost, "http://upstream.example/x", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if _, err := rt.RoundTrip(postReq); err == nil {
		t.Fatal("expected the failure to surface immediately")
	}
	if fake.calls != 1 {
		t.Fatalf("expected exactly 1 attempt for a POST — retrying it could double-create a resource, got %d calls", fake.calls)
	}
}

func TestRetryTransport_5xxIsRetriedTooNot4xx(t *testing.T) {
	fake := &countingTripper{results: []roundTripResult{
		{status: http.StatusServiceUnavailable},
		{status: http.StatusOK},
	}}
	rt := newRetryTransport(fake, 2, time.Millisecond)

	resp, err := rt.RoundTrip(req(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK || fake.calls != 2 {
		t.Fatalf("expected a 503 to be retried into a 200, got status=%d calls=%d", resp.StatusCode, fake.calls)
	}

	// A 4xx, by contrast, is a normal answer — never retried.
	fake2 := &countingTripper{results: []roundTripResult{{status: http.StatusNotFound}}}
	rt2 := newRetryTransport(fake2, 2, time.Millisecond)
	resp2, err := rt2.RoundTrip(req(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp2.StatusCode != http.StatusNotFound || fake2.calls != 1 {
		t.Fatalf("expected a 404 to be returned on the first attempt with no retry, got status=%d calls=%d", resp2.StatusCode, fake2.calls)
	}
}
