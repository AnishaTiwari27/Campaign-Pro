package ingest

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Field is one column this product needs from an export.
type Field struct {
	Name     string
	Required bool
	Note     string
}

// Fields is the target schema a mapping has to satisfy.
var Fields = []Field{
	{"external_id", true, "the platform's own campaign id — the key everything is matched on"},
	{"name", true, "the campaign name as the platform shows it"},
	{"category", false, "business category, e.g. Beauty, Fintech"},
	{"region", false, "where it ran"},
	{"platform", false, "the surface it ran on, e.g. YouTube, Instagram"},
	{"status", false, "live, paused, ended or scheduled, in the platform's own words"},
	{"ad_type", false, "the format, in the platform's own words"},
	{"reach", false, "reach or impressions — say which unit it is in alongside the mapping"},
	{"spend", false, "money spent, in rupees"},
	{"budget", false, "the budget, in rupees"},
	{"days_running", false, "days live so far"},
	{"frequency", false, "average impressions per person"},
}

// ColumnMapping says which column of an export feeds which field.
// Fields with no column are absent from the map rather than blank, so
// "not present" and "present but empty" stay different things.
type ColumnMapping map[string]string

// Inspection is what the importer reports back before anything is
// written: the headers it found, the mapping it proposes, and a few
// rows as they would be read under that mapping.
//
// The proposal is never applied on its own. A model reading column
// headers is guessing, usually well, and a confident wrong guess here
// writes the wrong number into a spend column — so the mapping comes
// back for confirmation and the import runs against what the human
// agreed to, not what was suggested.
// ReachUnit says how to read the reach column.
//
// It is part of the mapping and not inferred from the header, because
// the two readings differ by a factor of a hundred thousand and the
// header does not reliably say which it is. A Google export column
// called "Impr." holds raw impressions; this product stores lakhs.
// Guessing wrong either overflows the column — which is how this was
// found — or, worse, stores a number that is wrong but plausible.
type ReachUnit string

const (
	ReachRaw  ReachUnit = "raw"  // people or impressions, as counted
	ReachLakh ReachUnit = "lakh" // already divided by 100,000
)

type Inspection struct {
	Headers   []string            `json:"headers"`
	Mapping   ColumnMapping       `json:"mapping"`
	Unmapped  []string            `json:"unmapped"`
	Missing   []string            `json:"missing"`
	Preview   []map[string]string `json:"preview"`
	Source    string              `json:"source"`
	Confident bool                `json:"confident"`
	// ReachUnit the import will use unless the confirmation overrides
	// it. Raw is the default because that is what every platform
	// export gives; this product's own export is the exception.
	ReachUnit ReachUnit `json:"reachUnit"`
}

const (
	MappingSourceExact = "exact"
	MappingSourceModel = "model"
)

// previewRows is how many rows come back with a proposal. Enough to
// see that a column holds what the mapping claims, few enough that the
// response stays readable.
const previewRows = 3

// ReadTable pulls the headers and rows out of an export once, so a
// mapping can be proposed and then applied without re-reading.
func ReadTable(r io.Reader) (headers []string, rows [][]string, err error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1 // ragged exports are common; pad instead of failing
	records, err := cr.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("read csv: file is empty")
	}
	headers = make([]string, 0, len(records[0]))
	for _, h := range records[0] {
		headers = append(headers, strings.TrimSpace(h))
	}
	return headers, records[1:], nil
}

// ExactMapping matches columns whose names already are the field
// names, which is what this product's own export produces. Tried
// before any model: an exact match needs no inference, costs nothing
// and cannot be wrong.
func ExactMapping(headers []string) ColumnMapping {
	m := ColumnMapping{}
	for _, f := range Fields {
		for _, h := range headers {
			name := strings.TrimSpace(h)
			// reach_lakh is accepted as a name for the reach field so
			// this product's own export still matches exactly, and
			// carries its unit in the name while doing so.
			if strings.EqualFold(name, f.Name) || (f.Name == "reach" && strings.EqualFold(name, "reach_lakh")) {
				m[f.Name] = h
				break
			}
		}
	}
	return m
}

// Describe reports what a mapping covers and what it leaves out.
func Describe(headers []string, rows [][]string, m ColumnMapping, source string) Inspection {
	used := map[string]bool{}
	for _, col := range m {
		used[col] = true
	}
	insp := Inspection{Headers: headers, Mapping: m, Source: source, ReachUnit: ReachRaw}
	// Our own export names the column for its unit, so that one case
	// can be read off the name rather than asked about.
	if col, ok := m["reach"]; ok && strings.Contains(strings.ToLower(col), "lakh") {
		insp.ReachUnit = ReachLakh
	}

	for _, h := range headers {
		if !used[h] {
			insp.Unmapped = append(insp.Unmapped, h)
		}
	}
	for _, f := range Fields {
		if _, ok := m[f.Name]; !ok {
			insp.Missing = append(insp.Missing, f.Name)
		}
	}
	sort.Strings(insp.Missing)

	// Confident means every required field is mapped. It is not a claim
	// that the mapping is right — only a human looking at the preview
	// can say that — just that the import would not fail outright.
	insp.Confident = true
	for _, f := range Fields {
		if f.Required {
			if _, ok := m[f.Name]; !ok {
				insp.Confident = false
			}
		}
	}

	index := map[string]int{}
	for i, h := range headers {
		index[h] = i
	}
	for i, row := range rows {
		if i == previewRows {
			break
		}
		out := map[string]string{}
		for field, col := range m {
			if at, ok := index[col]; ok && at < len(row) {
				out[field] = strings.TrimSpace(row[at])
			}
		}
		insp.Preview = append(insp.Preview, out)
	}
	return insp
}

// RowsToCampaigns applies a confirmed mapping.
func RowsToCampaigns(headers []string, rows [][]string, m ColumnMapping, unit ReachUnit) ([]ExternalCampaign, error) {
	if _, ok := m["external_id"]; !ok {
		return nil, fmt.Errorf("the mapping has no external_id column — there is nothing to match rows on")
	}
	index := map[string]int{}
	for i, h := range headers {
		index[h] = i
	}
	get := func(row []string, field string) string {
		col, ok := m[field]
		if !ok {
			return ""
		}
		at, ok := index[col]
		if !ok || at >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[at])
	}

	out := make([]ExternalCampaign, 0, len(rows))
	for n, row := range rows {
		id := get(row, "external_id")
		if id == "" {
			// A row with no id cannot be matched on a later sync and
			// would insert a duplicate every time, so it is refused
			// rather than silently dropped.
			return nil, fmt.Errorf("row %d has no external_id — there is nothing to match it on", n+2)
		}
		out = append(out, ExternalCampaign{
			ExternalID:  id,
			Name:        get(row, "name"),
			Category:    get(row, "category"),
			Region:      get(row, "region"),
			Platform:    get(row, "platform"),
			Status:      normaliseStatus(get(row, "status")),
			AdType:      normaliseAdType(get(row, "ad_type")),
			DaysRunning: int32(atoi(get(row, "days_running"))),
			ReachLakh:   toLakh(atof(get(row, "reach")), unit),
			Spend:       int64(atoi(get(row, "spend"))),
			Budget:      int64(atoi(get(row, "budget"))),
			Frequency:   atof(get(row, "frequency")),
		})
	}
	return out, nil
}

// toLakh converts a reach figure into the unit this product stores.
func toLakh(v float64, unit ReachUnit) float64 {
	if unit == ReachLakh {
		return v
	}
	return v / 100000
}
