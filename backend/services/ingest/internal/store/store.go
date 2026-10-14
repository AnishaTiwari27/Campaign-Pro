// Package store is ingestion's persistence: the upsert that matches on a
// platform's own id, and the run history.
package store

import (
	"context"
	"time"

	"campaigntrackerpro/db/gen"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/ingest"

	"github.com/jackc/pgx/v5/pgtype"
)

type Store struct{ db *database.DB }

func New(db *database.DB) *Store { return &Store{db: db} }

// Upsert writes one external campaign and reports whether it was new.
func (s *Store) Upsert(ctx context.Context, source string, c ingest.ExternalCampaign) (created bool, err error) {
	o := scope.OrgFrom(ctx)
	row, err := s.db.Queries.UpsertExternalCampaign(ctx, gen.UpsertExternalCampaignParams{
		// A deterministic local id, so a row ingested twice keeps the
		// same primary key and every link to it survives a re-sync.
		ID:             source + ":" + c.ExternalID,
		Name:           c.Name,
		SubjectType:    gen.SubjectTypeT("brand"),
		Initials:       initialsOf(c.Name),
		Category:       c.Category,
		Region:         c.Region,
		AdType:         gen.AdTypeT(c.AdType),
		Platform:       c.Platform,
		Status:         gen.CampaignStatusT(c.Status),
		DaysRunning:    c.DaysRunning,
		Reach:          c.ReachLakh,
		Spend:          c.Spend,
		Budget:         c.Budget,
		Frequency:      c.Frequency,
		Approval:       gen.ApprovalStatusT("pending"),
		CurveShape:     gen.CurveShapeT("steady"),
		OrgID:          o.ID,
		ExternalSource: database.TextParam(source),
		ExternalID:     database.TextParam(c.ExternalID),
	})
	if err != nil {
		return false, err
	}
	return row.Created, nil
}

// Run is one sync, as the API reports it.
type Run struct {
	ID         string
	Source     string
	StartedAt  time.Time
	FinishedAt *time.Time
	Seen       int32
	Created    int32
	Updated    int32
	Failed     int32
	Error      string
}

func (s *Store) StartRun(ctx context.Context, source string) (string, error) {
	o := scope.OrgFrom(ctx)
	id, err := s.db.Queries.StartIngestRun(ctx, gen.StartIngestRunParams{OrgID: o.ID, Source: source})
	if err != nil {
		return "", err
	}
	return database.UuidToString(id), nil
}

func (s *Store) FinishRun(ctx context.Context, id string, r ingest.Result) error {
	uid, err := database.UuidParam(id)
	if err != nil {
		return err
	}
	var errText pgtype.Text
	if r.Err != nil {
		errText = database.TextParam(r.Err.Error())
	}
	return s.db.Queries.FinishIngestRun(ctx, gen.FinishIngestRunParams{
		ID:          uid,
		RowsSeen:    int32(r.Seen),
		RowsCreated: int32(r.Created),
		RowsUpdated: int32(r.Updated),
		RowsFailed:  int32(r.Failed),
		Error:       errText,
	})
}

func (s *Store) ListRuns(ctx context.Context) ([]Run, error) {
	o := scope.OrgFrom(ctx)
	rows, err := s.db.Queries.ListIngestRuns(ctx, gen.ListIngestRunsParams{
		OrgSystem: o.System, OrgID: o.ID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Run, 0, len(rows))
	for _, r := range rows {
		run := Run{
			ID: database.UuidToString(r.ID), Source: r.Source,
			StartedAt: r.StartedAt.Time,
			Seen:      r.RowsSeen, Created: r.RowsCreated,
			Updated: r.RowsUpdated, Failed: r.RowsFailed,
			Error: database.TextOrEmpty(r.Error),
		}
		if r.FinishedAt.Valid {
			t := r.FinishedAt.Time
			run.FinishedAt = &t
		}
		out = append(out, run)
	}
	return out, nil
}

// initialsOf is the two-letter chip the UI falls back to when a brand
// has no logo. Derived rather than required from the platform, which has
// no concept of it.
func initialsOf(name string) string {
	var out []rune
	for _, word := range splitWords(name) {
		if len(out) == 2 {
			break
		}
		out = append(out, []rune(word)[0])
	}
	if len(out) == 0 {
		return "??"
	}
	return string(out)
}

func splitWords(s string) []string {
	var words []string
	cur := ""
	for _, r := range s {
		if r == ' ' || r == '-' || r == '_' {
			if cur != "" {
				words = append(words, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		words = append(words, cur)
	}
	return words
}
