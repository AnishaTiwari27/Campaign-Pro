package platform

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when a request is rejected without even
// attempting the network call, because too many recent calls to this
// upstream have failed. The caller sees this the same way it'd see any
// other RoundTrip error (catalogclient/campaignsclient already treat "err
// != nil" as "upstream unreachable") — no special-casing needed at the
// call site for the breaker to take effect.
var ErrCircuitOpen = errors.New("circuit breaker open: upstream has been failing, not retrying yet")

type breakerState int

const (
	breakerClosed   breakerState = iota // normal — requests pass through
	breakerOpen                         // tripped — requests fail immediately, no network call
	breakerHalfOpen                     // cooldown elapsed — exactly one probe request allowed through
)

// circuitBreaker wraps a RoundTripper and stops calling a consistently
// failing upstream instead of letting every caller queue up behind the
// same timeout — the gap this closes: platform/client.go used to be a
// plain 5s-timeout client, so a degraded catalog-service stalled every
// POST /campaigns (which calls it synchronously) up to that timeout, on
// every single request, with no fast-fail. See docs/ROADMAP.md's Phase A.
//
// Closed -> N consecutive failures -> Open (fails fast) -> cooldown
// elapses -> HalfOpen (one probe allowed) -> success closes it again,
// failure reopens it for another cooldown.
//
// A 4xx response is never a "failure" here — that's the upstream
// correctly answering a bad request (e.g. catalog-service's 404 for an
// unknown subject name, which callers already treat as a normal, distinct
// outcome — see catalogclient.Lookup), not the upstream being unhealthy.
// Only a transport error or a 5xx counts.
type circuitBreaker struct {
	next             http.RoundTripper
	failureThreshold int
	cooldown         time.Duration

	mu               sync.Mutex
	state            breakerState
	consecutiveFails int
	openedAt         time.Time
}

func newCircuitBreaker(next http.RoundTripper, failureThreshold int, cooldown time.Duration) *circuitBreaker {
	return &circuitBreaker{next: next, failureThreshold: failureThreshold, cooldown: cooldown}
}

func (b *circuitBreaker) RoundTrip(req *http.Request) (*http.Response, error) {
	if !b.allow() {
		return nil, ErrCircuitOpen
	}

	resp, err := b.next.RoundTrip(req)
	b.record(err == nil && resp.StatusCode < 500) // short-circuits before touching resp when err != nil
	return resp, err
}

// allow reports whether this request should even attempt the network
// call, transitioning Open -> HalfOpen once the cooldown has elapsed.
func (b *circuitBreaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != breakerOpen {
		return true
	}
	if time.Since(b.openedAt) < b.cooldown {
		return false
	}
	b.state = breakerHalfOpen
	return true
}

func (b *circuitBreaker) record(ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ok {
		b.consecutiveFails = 0
		b.state = breakerClosed
		return
	}

	b.consecutiveFails++
	if b.state == breakerHalfOpen || b.consecutiveFails >= b.failureThreshold {
		b.state = breakerOpen
		b.openedAt = time.Now()
	}
}
