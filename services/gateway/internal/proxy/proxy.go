// Package proxy turns a fixed target or a load-balanced pool into an
// http.Handler the gateway's mux can route to directly.
package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/gateway/internal/lb"
)

// Fixed proxies every request straight to target — for catalog-service and
// analytics-service, which the gateway doesn't load-balance across (a
// single local instance each is enough to demonstrate the pattern; the LB
// story is told by campaigns-service instead, see Balanced below).
func Fixed(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		panic(err) // misconfigured target — fail fast at startup, not on first request
	}
	return httputil.NewSingleHostReverseProxy(u)
}

// Balanced proxies each request to whichever backend pool.Next() returns,
// re-selecting per request so a backend that flips unhealthy mid-session is
// dropped out of rotation on the very next request.
func Balanced(pool *lb.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := pool.Next()
		if target == nil {
			platform.Upstream(w, "no healthy campaigns-service backend available")
			return
		}
		httputil.NewSingleHostReverseProxy(target).ServeHTTP(w, r)
	})
}
