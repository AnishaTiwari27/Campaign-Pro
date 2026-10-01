package httpapi

import "net/http"

func (h *Handlers) GetOverview(w http.ResponseWriter, r *http.Request) {
	o, err := h.Overview.Get(r.Context())
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, overviewDTO(o))
}

func (h *Handlers) GetBenchmark(w http.ResponseWriter, r *http.Request) {
	b, err := h.Campaigns.Benchmark(r.Context())
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, benchmarkDTO(b))
}
