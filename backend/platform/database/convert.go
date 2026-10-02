package database

import (
	"errors"
	"time"

	"campaigntrackerpro/db/gen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrNotFound is what every store returns for a missing row, so httpx can
// map it to a 404 without knowing which service asked.
var ErrNotFound = errors.New("not found")

// pgtype <-> Go conversions. Shared because every service's store crosses
// the same boundary; none of them should own these for the others.

func TextOrEmpty(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func TextParam(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func PgxBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}

func TimeOf(t pgtype.Timestamptz) time.Time {
	return t.Time
}

func UuidToString(id pgtype.UUID) string {
	u := uuid.UUID(id.Bytes)
	return u.String()
}

func UuidParam(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func TimestampParam(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func HookTypeParam(s string) gen.NullHookTypeT {
	if s == "" {
		return gen.NullHookTypeT{}
	}
	return gen.NullHookTypeT{HookTypeT: gen.HookTypeT(s), Valid: true}
}
