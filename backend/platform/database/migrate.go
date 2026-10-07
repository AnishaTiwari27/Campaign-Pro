package database

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"campaigntrackerpro/db"

	"github.com/golang-migrate/migrate/v4"
	// The pgx/v5 driver, so migrations run over the same driver as the rest
	// of the app rather than pulling in a second Postgres stack.
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrate applies every pending migration and returns once the database is
// at the latest version. It is safe to call on every boot: already-applied
// migrations are a no-op, and golang-migrate takes a Postgres advisory lock,
// so two instances starting at once cannot both run the same migration.
//
// Running this at startup is what makes a deploy to a blank database work
// without shipping the migrate CLI in the image. It is a deliberate
// trade-off: it suits a single-instance deployment, where boot order is
// simple. A multi-instance rollout should prefer a release-phase command so
// schema changes land before any new code serves traffic.
func Migrate(databaseURL string, logger *slog.Logger) error {
	src, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}
	defer func() { _ = src.Close() }()

	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL(databaseURL))
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	// Close returns a source error and a database error; the source is ours
	// and already deferred, so only the database one is worth reporting.
	defer func() {
		if _, dberr := m.Close(); dberr != nil {
			logger.Warn("closing migrator", "err", dberr)
		}
	}()

	before, _, _ := m.Version()
	switch err := m.Up(); {
	case errors.Is(err, migrate.ErrNoChange):
		logger.Info("schema up to date", "version", before)
		return nil
	case err != nil:
		return fmt.Errorf("apply migrations: %w", err)
	}
	after, _, _ := m.Version()
	logger.Info("schema migrated", "from", before, "to", after)
	return nil
}

// migrateURL adapts the app's DATABASE_URL for the migrator. Operators set
// one variable; getting a second one subtly wrong is a trap, so the two
// adjustments needed are made here instead.
//
// First the scheme, to the one golang-migrate's pgx/v5 driver registers.
//
// Then the host, if it is one of Neon's pooled endpoints. golang-migrate
// serialises migrations with a session-scoped pg_advisory_lock, and a pooled
// endpoint is PgBouncer in transaction-pooling mode, which does not keep a
// session on one backend connection: the lock and its unlock can land on
// different connections, leaving the lock held by an idle one. Migrations
// then hang or fail. Connection pooling is right for ordinary traffic, so the
// app's own pool still uses the URL exactly as given — only migrations are
// routed around the pooler.
func migrateURL(databaseURL string) string {
	url := databaseURL
	for _, scheme := range []string{"postgresql://", "postgres://"} {
		if strings.HasPrefix(url, scheme) {
			url = "pgx5://" + strings.TrimPrefix(url, scheme)
			break
		}
	}
	return directHost(url)
}

// directHost turns a Neon pooled host into its direct equivalent:
// ep-x-123-pooler.region.aws.neon.tech -> ep-x-123.region.aws.neon.tech.
// Scoped to neon.tech so no other provider's hostname is rewritten on the
// strength of a substring.
func directHost(url string) string {
	if !strings.Contains(url, "neon.tech") || !strings.Contains(url, "-pooler.") {
		return url
	}
	return strings.Replace(url, "-pooler.", ".", 1)
}
