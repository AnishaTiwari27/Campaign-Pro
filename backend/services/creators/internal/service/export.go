package service

import (
	"context"
	"fmt"
	"strings"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/creators"
)

// Actor is who is taking the export. Carried explicitly rather than
// pulled from the context so an unattributed export cannot happen by
// omission — the audit row is the only record that the file ever left.
type Actor struct {
	ID   string
	Name string
}

// ExportCSV renders the roster and records that it left the building.
//
// Columns are validated against the allow-list rather than passed
// through: they arrive from a query string, and an unchecked column
// name is how a caller would ask for a field the export deliberately
// does not carry.
func (c *Creators) ExportCSV(ctx context.Context, columns []string, actor Actor) ([]byte, string, error) {
	if len(columns) == 0 {
		columns = creators.RosterColumns
	}
	for _, col := range columns {
		if !creators.IsValidRosterColumn(col) {
			return nil, "", fmt.Errorf("%w: no such column %q", httpx.ErrValidation, col)
		}
	}

	rows, err := c.List(ctx)
	if err != nil {
		return nil, "", err
	}
	out := creators.RosterCSV(rows, columns)

	// Written before the file is handed over, and a failure here fails
	// the export. An untracked export is the thing this is for, so
	// logging the error and serving the file anyway would defeat it.
	action := fmt.Sprintf("Exported %d creators (%s)", len(rows), strings.Join(columns, ", "))
	if err := c.Store.RecordExport(ctx, actor.ID, actor.Name, action); err != nil {
		return nil, "", fmt.Errorf("record export: %w", err)
	}
	return out, filename(len(rows)), nil
}

func filename(n int) string {
	return fmt.Sprintf("creators-%d.csv", n)
}
