package api

import (
	"net/http"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/ingest"
	"campaigntrackerpro/services/ingest/internal/store"
)

type resultJSON struct {
	Source  string `json:"source"`
	Seen    int    `json:"seen"`
	Created int    `json:"created"`
	Updated int    `json:"updated"`
	Failed  int    `json:"failed"`
}

func resultDTO(r ingest.Result) resultJSON {
	return resultJSON{Source: r.Source, Seen: r.Seen, Created: r.Created, Updated: r.Updated, Failed: r.Failed}
}

type runJSON struct {
	ID         string  `json:"id"`
	Source     string  `json:"source"`
	StartedAt  string  `json:"startedAt"`
	FinishedAt *string `json:"finishedAt"`
	Seen       int32   `json:"seen"`
	Created    int32   `json:"created"`
	Updated    int32   `json:"updated"`
	Failed     int32   `json:"failed"`
	Error      string  `json:"error,omitempty"`
}

func runDTOs(runs []store.Run) []runJSON {
	out := make([]runJSON, 0, len(runs))
	for _, r := range runs {
		j := runJSON{
			ID: r.ID, Source: r.Source,
			StartedAt: r.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			Seen:      r.Seen, Created: r.Created, Updated: r.Updated, Failed: r.Failed,
			Error: r.Error,
		}
		if r.FinishedAt != nil {
			s := r.FinishedAt.Format("2006-01-02T15:04:05Z07:00")
			j.FinishedAt = &s
		}
		out = append(out, j)
	}
	return out
}

// isClient mirrors the check analytics makes: the principal says so.
func isClient(r *http.Request) bool {
	v, _ := httpx.UserFromContext(r.Context())
	c, ok := v.(interface{ IsClient() bool })
	return ok && c.IsClient()
}
