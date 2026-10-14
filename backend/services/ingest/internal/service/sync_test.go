package service

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/ingest"
	"campaigntrackerpro/services/ingest/internal/store"

	"github.com/stretchr/testify/require"
)

func testSync(t *testing.T) (*Sync, *database.DB) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable"
	}
	pool, err := database.Connect(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping: no local Postgres available (%v)", err)
	}
	t.Cleanup(pool.Close)
	db := database.New(pool)
	return New(store.New(db), slog.New(slog.NewTextHandler(io.Discard, nil))), db
}

func testCtx() context.Context {
	return scope.WithOrg(context.Background(), scope.Org{ID: scope.DefaultOrg})
}

const hdr = "external_id,name,category,region,platform,status,ad_type,reach_lakh,spend,budget,days_running,frequency\n"

func csvOf(rows string) ingest.Provider {
	return ingest.CSVProvider{SourceName: "svctest-src", Reader: strings.NewReader(hdr + rows)}
}

func cleanupIngested(t *testing.T, db *database.DB) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = db.Pool.Exec(ctx, `DELETE FROM campaigns WHERE external_source = 'svctest-src'`)
		_, _ = db.Pool.Exec(ctx, `DELETE FROM ingest_runs WHERE source = 'svctest-src'`)
	})
}

// The point of the whole design. A platform's own id is the match key,
// so running the same sync twice must update the rows rather than
// inserting a second copy of every campaign. "47 seen, 47 created" on a
// second run is what broken matching looks like, which is why created
// and updated are counted apart.
func TestASecondSyncUpdatesRatherThanDuplicating(t *testing.T) {
	svc, db := testSync(t)
	cleanupIngested(t, db)
	ctx := testCtx()

	rows := "ext-1,First,Beauty,Mumbai,Instagram,live,Social,10,1000,5000,5,1.2\n" +
		"ext-2,Second,FMCG,Delhi NCR,TV,live,Video,20,2000,6000,6,1.4\n"

	first, err := svc.Run(ctx, csvOf(rows), time.Time{})
	require.NoError(t, err)
	require.Equal(t, 2, first.Seen)
	require.Equal(t, 2, first.Created)
	require.Equal(t, 0, first.Updated)

	second, err := svc.Run(ctx, csvOf(rows), time.Time{})
	require.NoError(t, err)
	require.Equal(t, 2, second.Seen)
	require.Equal(t, 0, second.Created, "the same rows were inserted again — matching is broken")
	require.Equal(t, 2, second.Updated)

	var n int
	require.NoError(t, db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM campaigns WHERE external_source = 'svctest-src'`).Scan(&n))
	require.Equal(t, 2, n, "duplicate campaigns in the table")
}

// Figures change on a re-sync; the agency's own decision must not.
// Approval is the agency signing off on spend, and a sync that reset it
// would undo somebody's sign-off on every run.
func TestASyncUpdatesFiguresButNeverTheApproval(t *testing.T) {
	svc, db := testSync(t)
	cleanupIngested(t, db)
	ctx := testCtx()

	_, err := svc.Run(ctx, csvOf("ext-3,Third,Beauty,Mumbai,Instagram,live,Social,10,1000,5000,5,1.2\n"), time.Time{})
	require.NoError(t, err)

	// Somebody approves it.
	_, err = db.Pool.Exec(ctx,
		`UPDATE campaigns SET approval = 'approved' WHERE external_source = 'svctest-src' AND external_id = 'ext-3'`)
	require.NoError(t, err)

	// The platform reports new numbers.
	_, err = svc.Run(ctx, csvOf("ext-3,Third Renamed,Beauty,Mumbai,Instagram,live,Social,33,7777,5000,9,1.9\n"), time.Time{})
	require.NoError(t, err)

	var approval, name string
	var spend int64
	require.NoError(t, db.Pool.QueryRow(ctx,
		`SELECT approval, name, spend FROM campaigns WHERE external_source = 'svctest-src' AND external_id = 'ext-3'`).
		Scan(&approval, &name, &spend))

	require.Equal(t, "approved", approval, "the sync reset an approval the agency had given")
	require.Equal(t, "Third Renamed", name, "the sync did not pick up the new name")
	require.Equal(t, int64(7777), spend, "the sync did not pick up the new spend")
}

// Every sync is recorded, including a failed one: an ingestion that
// silently does nothing is indistinguishable from one that works until
// somebody asks why the numbers are stale.
func TestEveryRunIsRecordedIncludingAFailure(t *testing.T) {
	svc, db := testSync(t)
	cleanupIngested(t, db)
	ctx := testCtx()

	_, err := svc.Run(ctx, csvOf("ext-4,Fourth,Beauty,Mumbai,Instagram,live,Social,10,1000,5000,5,1.2\n"), time.Time{})
	require.NoError(t, err)

	// A provider that cannot run at all still leaves a run behind.
	_, err = svc.Run(ctx, ingest.GoogleAdsProvider{}, time.Time{})
	require.Error(t, err)

	runs, err := svc.Runs(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(runs), 2)

	var sawFailure bool
	for _, r := range runs {
		if r.Source == "google-ads" && r.Error != "" {
			sawFailure = true
			require.NotNil(t, r.FinishedAt, "a failed run was left open, so it reads as still running")
		}
	}
	require.True(t, sawFailure, "the failed run was not recorded")

	_, _ = db.Pool.Exec(ctx, `DELETE FROM ingest_runs WHERE source = 'google-ads'`)
}
