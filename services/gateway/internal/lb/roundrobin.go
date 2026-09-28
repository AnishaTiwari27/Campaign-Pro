// Package lb is a small round-robin load balancer with active health
// checking — enough to make "kill one instance, traffic keeps flowing" a
// real, demonstrable property of the gateway rather than a diagram.
package lb

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type backend struct {
	url     *url.URL
	healthy atomic.Bool
}

// Pool round-robins across a fixed set of backend URLs, skipping any that
// its background health check has marked unhealthy.
type Pool struct {
	name     string
	backends []*backend
	next     atomic.Uint64
	client   *http.Client
}

// NewPool starts a Pool over rawURLs, health-checking each one's /healthz
// every checkEvery. All backends start assumed healthy so the pool is
// usable immediately, before the first check completes.
func NewPool(name string, rawURLs []string, checkEvery time.Duration) *Pool {
	p := &Pool{name: name, client: &http.Client{Timeout: 2 * time.Second}}
	for _, raw := range rawURLs {
		u, err := url.Parse(raw)
		if err != nil {
			slog.Error("lb: bad backend URL", "pool", name, "url", raw, "err", err)
			continue
		}
		b := &backend{url: u}
		b.healthy.Store(true)
		p.backends = append(p.backends, b)
	}
	go p.healthCheckLoop(checkEvery)
	return p
}

// Next returns the next healthy backend in rotation, or nil if every
// backend is currently marked unhealthy.
func (p *Pool) Next() *url.URL {
	n := len(p.backends)
	if n == 0 {
		return nil
	}
	// Try every backend at most once, starting from the next rotation slot,
	// so an unhealthy backend is skipped without breaking round-robin order
	// for the healthy ones.
	start := p.next.Add(1)
	for i := uint64(0); i < uint64(n); i++ {
		b := p.backends[(start+i)%uint64(n)]
		if b.healthy.Load() {
			return b.url
		}
	}
	return nil
}

func (p *Pool) healthCheckLoop(every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for range ticker.C {
		for _, b := range p.backends {
			healthy := p.probe(b.url)
			was := b.healthy.Swap(healthy)
			if was != healthy {
				slog.Warn("lb: backend health changed", "pool", p.name, "backend", b.url.String(), "healthy", healthy)
			}
		}
	}
}

func (p *Pool) probe(u *url.URL) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String()+"/healthz", nil)
	if err != nil {
		return false
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Status is a point-in-time snapshot, exposed on the gateway's own
// /healthz so you can see which backends are in rotation.
func (p *Pool) Status() map[string]bool {
	out := make(map[string]bool, len(p.backends))
	for _, b := range p.backends {
		out[b.url.String()] = b.healthy.Load()
	}
	return out
}
