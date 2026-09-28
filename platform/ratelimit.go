package platform

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimit is gateway middleware enforcing a per-caller request ceiling —
// docs/API_CONTRACT.md has documented a 429 since early in this project;
// this is what actually returns it now. See docs/ROADMAP.md's Phase A.
//
// One bucket per key (see ByUser/ByIP for what the key is), each its own
// token bucket via golang.org/x/time/rate — a burst is allowed, but
// sustained traffic above the configured rate gets 429s.
//
// In-memory, per gateway instance — correct today (the gateway itself is
// single-instance, same as every other pool in this system before this
// pass); would need a shared store (Redis, already in the stack) if the
// gateway is ever load-balanced, so two instances don't each grant a
// caller the full rate independently. Not built now. Also unbounded: a
// caller (or key) is never evicted, so a very large number of distinct
// keys (e.g. an IP-scanning bot cycling addresses against /auth/login)
// grows this map indefinitely. Acceptable for a single-gateway deployment
// fronted by generous limits; a production-hardened version would add
// idle-eviction or cap the map size.
type RateLimit struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func NewRateLimit(requestsPerSecond float64, burst int) *RateLimit {
	return &RateLimit{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

func (rl *RateLimit) allow(key string) bool {
	rl.mu.Lock()
	limiter, ok := rl.limiters[key]
	if !ok {
		limiter = rate.NewLimiter(rl.rps, rl.burst)
		rl.limiters[key] = limiter
	}
	rl.mu.Unlock()
	return limiter.Allow()
}

// ByUser rate-limits keyed on the trusted X-User-Id header. Only meaningful
// wrapped *inside* RequireAuth (RequireAuth(secret, rl.ByUser(next))) —
// RequireAuth sets that header after verifying the token and strips any
// caller-supplied version first, so by the time ByUser reads it here it's
// trustworthy; called the other way around, there'd be no header yet.
func (rl *RateLimit) ByUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-User-Id")
		if key == "" {
			key = "unauthenticated" // shouldn't happen behind RequireAuth, but never key on an empty string shared by everyone
		}
		if !rl.allow(key) {
			WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, slow down")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ByIP rate-limits keyed on the caller's remote address — for routes
// reached before any identity is verified (/auth/*), where per-user
// keying isn't available yet and brute-forcing login is exactly the
// threat this is closing.
func (rl *RateLimit) ByIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr // no port present — use it as-is rather than dropping the caller's identity entirely
		}
		if !rl.allow(host) {
			WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, slow down")
			return
		}
		next.ServeHTTP(w, r)
	})
}
