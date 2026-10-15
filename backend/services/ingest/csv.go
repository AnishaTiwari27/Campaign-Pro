package ingest

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// CSVProvider reads an export. Not a stand-in for the real thing: most
// agencies can produce a CSV from any platform's UI long before anyone
// gets API credentials issued, and it runs the same matching and upsert
// path as every other source.
type CSVProvider struct {
	SourceName string
	Reader     io.Reader
}

func (p CSVProvider) Source() string {
	if p.SourceName == "" {
		return "csv"
	}
	return p.SourceName
}

// Required columns. Named rather than positional because a spreadsheet
// that has been through a human has had its columns reordered.
var csvRequired = []string{"external_id", "name", "category", "region", "platform", "status", "ad_type", "reach_lakh", "spend", "budget"}

// This path reads reach_lakh, which names its own unit. The mapped
// path in mapping.go asks for the unit instead, because a platform
// export's column name does not reliably say.

func (p CSVProvider) Fetch(_ context.Context, _ time.Time) ([]ExternalCampaign, error) {
	r := csv.NewReader(p.Reader)
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("read csv: file is empty")
	}

	idx := map[string]int{}
	for i, h := range records[0] {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	var missing []string
	for _, want := range csvRequired {
		if _, ok := idx[want]; !ok {
			missing = append(missing, want)
		}
	}
	if len(missing) > 0 {
		// Named in full, because the alternative is an import that
		// silently produces zero rows and a user who cannot tell why.
		return nil, fmt.Errorf("read csv: missing column(s): %s", strings.Join(missing, ", "))
	}

	out := make([]ExternalCampaign, 0, len(records)-1)
	for n, rec := range records[1:] {
		get := func(col string) string {
			i := idx[col]
			if i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}
		if get("external_id") == "" {
			return nil, fmt.Errorf("read csv: row %d has no external_id — there is nothing to match it on", n+2)
		}
		out = append(out, ExternalCampaign{
			ExternalID:  get("external_id"),
			Name:        get("name"),
			Category:    get("category"),
			Region:      get("region"),
			Platform:    get("platform"),
			Status:      normaliseStatus(get("status")),
			AdType:      normaliseAdType(get("ad_type")),
			DaysRunning: int32(atoi(get("days_running"))),
			ReachLakh:   atof(get("reach_lakh")),
			Spend:       int64(atoi(get("spend"))),
			Budget:      int64(atoi(get("budget"))),
			Frequency:   atof(get("frequency")),
		})
	}
	return out, nil
}

// normaliseStatus maps the words platforms actually use onto the four
// this product has. Anything unrecognised becomes paused rather than
// live: a campaign wrongly shown as paused is noticed and corrected, one
// wrongly shown as live quietly accrues spend nobody is watching.
func normaliseStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "live", "enabled", "active", "serving", "running":
		return "live"
	case "ended", "completed", "finished", "removed", "archived":
		return "ended"
	case "scheduled", "pending", "upcoming", "draft":
		return "scheduled"
	default:
		return "paused"
	}
}

// normaliseAdType maps a platform's own format names onto the six this
// product stores. Needed for the same reason as normaliseStatus and
// missed on the first pass: a Google Ads export says "Search", which is
// not one of the six, and the row was rejected by the enum rather than
// landing with a sensible label.
//
// Unknown becomes Display, which is the neutral one. Unlike status this
// carries no operational risk — ad type only affects how a campaign is
// grouped and labelled, and a wrong label is visible on the page where
// someone can correct it.
func normaliseAdType(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "social", "meta", "facebook", "instagram", "paid social":
		return "Social"
	case "influencer", "creator", "partnership":
		return "Influencer"
	case "search", "shopping", "google ads", "google", "sem":
		return "Google Ads"
	case "video", "youtube", "in-stream", "demand gen":
		return "Video"
	case "performance", "performance max", "pmax", "app":
		return "Performance"
	default:
		return "Display"
	}
}

// atoi and atof treat an unparseable cell as zero. A malformed number in
// one column should not reject the row — the figure is visibly wrong in
// the UI, where someone can see and fix it, which is more useful than a
// failed import with no data at all.
func atoi(s string) int {
	n, err := strconv.Atoi(strings.ReplaceAll(s, ",", ""))
	if err != nil {
		return 0
	}
	return n
}

func atof(s string) float64 {
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		return 0
	}
	return f
}
