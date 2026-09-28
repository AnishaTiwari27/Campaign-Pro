package platform

import (
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewInternalClient returns an http.Client for service-to-service calls.
// Its Transport is layered, outermost first:
//
//	circuitBreaker(retryTransport(otelhttp(http.DefaultTransport)))
//
// otelhttp propagates the current trace context (traceparent header)
// automatically — a request that enters at the gateway shows up as one
// continuous trace across gateway → campaigns-service → catalog-service,
// not three disconnected ones. retryTransport retries a transient GET
// failure a couple of times before giving up. circuitBreaker sits outside
// that — it sees the *final* outcome after retries are exhausted, so a
// genuinely unhealthy upstream trips it after a few requests' worth of
// real attempts, not one retry attempt each; once tripped, further calls
// fail immediately instead of each paying the retry+timeout cost. Every
// existing caller (catalogclient, campaignsclient) gets both for free —
// see docs/ROADMAP.md's Phase A for why this was missing before.
func NewInternalClient() *http.Client {
	transport := newRetryTransport(otelhttp.NewTransport(http.DefaultTransport), 2, 100*time.Millisecond)
	breaker := newCircuitBreaker(transport, 5, 10*time.Second)
	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: breaker,
	}
}
