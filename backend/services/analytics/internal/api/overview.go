package api

import (
	"net/http"

	"campaigntrackerpro/platform/httpx"
)

func (h *Handlers) GetOverview(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.Get(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, overviewDTO(o))
}

func (h *Handlers) GetBenchmark(w http.ResponseWriter, r *http.Request) {
	// Benchmarks are the competitive set: every category's median and its
	// top performer across the whole book of business. An external client
	// seeing that would be seeing how the agency's other accounts are
	// doing. Enforced here rather than by hiding the nav item, because a
	// hidden link is not a permission.
	if isClient(r) {
		httpx.WriteForbidden(w, "benchmarks compare accounts across the agency and aren't shown to client accounts")
		return
	}
	b, err := h.svc.Benchmark(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, benchmarkDTO(b))
}
