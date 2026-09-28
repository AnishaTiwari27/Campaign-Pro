package platform

import (
	"net/http"
	"time"
)

// retryTransport retries a failed GET request a bounded number of times
// with a short linear backoff — never POST/PUT/DELETE, so a request that
// already had a side effect (like creating a campaign) is never silently
// repeated. A transient blip in an otherwise-healthy upstream (a dropped
// connection, one slow GC pause) shouldn't surface as a user-visible
// error just because it happened to land on the first attempt.
//
// Only safe for requests with no body to replay (every GET this codebase
// makes is built with a nil body — catalogclient.Lookup/ListAll,
// campaignsclient's aggregate calls) — a GET with a body would need its
// Body reset between attempts, which this doesn't do.
type retryTransport struct {
	next    http.RoundTripper
	retries int
	backoff time.Duration
}

func newRetryTransport(next http.RoundTripper, retries int, backoff time.Duration) *retryTransport {
	return &retryTransport{next: next, retries: retries, backoff: backoff}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return t.next.RoundTrip(req)
	}

	var resp *http.Response
	var err error
	for attempt := 0; attempt <= t.retries; attempt++ {
		if attempt > 0 {
			time.Sleep(t.backoff * time.Duration(attempt))
		}
		resp, err = t.next.RoundTrip(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if resp != nil {
			resp.Body.Close() // discarding this attempt's response before retrying — otherwise it leaks the connection
		}
	}
	return resp, err
}
