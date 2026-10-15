package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"campaigntrackerpro/services/ingest"
)

const mapperSystem = `You map the columns of an advertising export onto a fixed schema.

You are given the export's column headers and two sample rows. Reply with a JSON object and nothing else: keys are schema field names, values are the exact column header that feeds them.

Rules:
1. Use ONLY column headers from the list given. Never invent one, never alter the spelling or case.
2. Omit a field entirely if no column fits. A wrong mapping is far worse than a missing one — it silently writes the wrong number into a figure someone will spend against.
3. Judge by the sample values as well as the header. A column called "cost" holding dates is not spend.
4. Map each column at most once.
5. No explanation, no markdown fence, no commentary. The JSON object only.`

// ProposeMapping suggests how an unfamiliar export lines up with the
// schema.
//
// An exact-name match is tried first and short-circuits: this
// product's own export already uses the field names, so the common
// case costs nothing and cannot be wrong. The model is only asked
// about headers it could not place.
//
// Whatever comes back is filtered against the real header list before
// it is returned. A model naming a column that is not in the file is
// the failure that matters here — it would map a field to nothing and
// read every row as blank — and it is cheap to make impossible.
func (s *Sync) ProposeMapping(ctx context.Context, headers []string, rows [][]string) ingest.Inspection {
	exact := ingest.ExactMapping(headers)
	insp := ingest.Describe(headers, rows, exact, ingest.MappingSourceExact)
	if insp.Confident && len(insp.Missing) == 0 {
		return insp
	}
	if s.model == nil {
		return insp
	}

	proposed, err := s.askMapper(ctx, headers, rows, exact)
	if err != nil {
		s.logger.Info("ingest: could not propose a mapping, falling back to exact matches",
			"err", err, "mapped", len(exact))
		return insp
	}
	// An exact match always wins over a suggestion.
	for field, col := range exact {
		proposed[field] = col
	}
	return ingest.Describe(headers, rows, proposed, ingest.MappingSourceModel)
}

func (s *Sync) askMapper(ctx context.Context, headers []string, rows [][]string, exact ingest.ColumnMapping) (ingest.ColumnMapping, error) {
	var b strings.Builder
	b.WriteString("SCHEMA FIELDS:\n")
	for _, f := range ingest.Fields {
		req := ""
		if f.Required {
			req = " (required)"
		}
		fmt.Fprintf(&b, "- %s%s: %s\n", f.Name, req, f.Note)
	}
	fmt.Fprintf(&b, "\nCOLUMN HEADERS:\n")
	for _, h := range headers {
		fmt.Fprintf(&b, "- %s\n", h)
	}
	b.WriteString("\nSAMPLE ROWS:\n")
	for i, row := range rows {
		if i == 2 {
			break
		}
		fmt.Fprintf(&b, "- %s\n", strings.Join(row, " | "))
	}
	if len(exact) > 0 {
		b.WriteString("\nAlready matched by name, do not change these:\n")
		for f, c := range exact {
			fmt.Fprintf(&b, "- %s -> %s\n", f, c)
		}
	}

	reply, err := s.model.Complete(ctx, mapperSystem, b.String())
	if err != nil {
		return nil, err
	}
	return parseMapping(reply, headers)
}

// parseMapping reads the model's JSON and discards anything that does
// not name a real column or a real field.
func parseMapping(reply string, headers []string) (ingest.ColumnMapping, error) {
	reply = strings.TrimSpace(reply)
	// Models fence JSON even when told not to.
	if i := strings.Index(reply, "{"); i > 0 {
		reply = reply[i:]
	}
	if j := strings.LastIndex(reply, "}"); j >= 0 && j < len(reply)-1 {
		reply = reply[:j+1]
	}

	var raw map[string]string
	if err := json.Unmarshal([]byte(reply), &raw); err != nil {
		return nil, fmt.Errorf("mapping reply was not JSON: %w", err)
	}

	valid := map[string]bool{}
	for _, f := range ingest.Fields {
		valid[f.Name] = true
	}
	known := map[string]string{}
	for _, h := range headers {
		known[strings.ToLower(strings.TrimSpace(h))] = h
	}

	out := ingest.ColumnMapping{}
	taken := map[string]bool{}
	for field, col := range raw {
		if !valid[field] {
			continue
		}
		actual, ok := known[strings.ToLower(strings.TrimSpace(col))]
		if !ok {
			// The column does not exist in the file. Dropped rather
			// than kept: a field pointing at nothing reads every row
			// as blank, which looks like missing data rather than a
			// bad mapping.
			continue
		}
		if taken[actual] {
			continue
		}
		taken[actual] = true
		out[field] = actual
	}
	return out, nil
}
