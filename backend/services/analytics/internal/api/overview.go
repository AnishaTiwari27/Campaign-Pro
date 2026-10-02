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
	b, err := h.svc.Benchmark(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, benchmarkDTO(b))
}
