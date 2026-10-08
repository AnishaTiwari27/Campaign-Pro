// Package config loads runtime configuration from the environment.
package config

import (
	"errors"
	"os"
	"strings"
)

// devDatabaseURL matches what `make db-create` sets up, so a checkout runs
// with no configuration at all. It is only ever right on a developer's own
// machine.
const devDatabaseURL = "postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable"

// ErrDatabaseURLMissing is returned when the local development fallback is in
// use somewhere it cannot possibly work. The message names the fix rather than
// the symptom: the symptom is a connection refused to 127.0.0.1 as a user that
// exists only on a developer's laptop, which reads as a broken database.
var ErrDatabaseURLMissing = errors.New(
	"DATABASE_URL is not set, and this process is running on a managed host " +
		"where the local development fallback cannot work. Set DATABASE_URL on " +
		"the service to your Postgres URL, including ?sslmode=require. On Render " +
		"that is the service's Environment tab: render.yaml marks the key " +
		"sync: false, so a Blueprint sync deliberately never supplies the value")

// managedHostMarkers are environment variables that container platforms set on
// every instance they run. Their presence means the process is not on a
// developer's machine, which turns the database fallback from "probably wrong"
// into "certainly wrong".
//
// A heuristic, and deliberately a short one: it is a safety net that converts a
// confusing failure into a clear one, not a boundary anything relies on. A host
// that is not listed simply gets the old behaviour — the warning, then the
// connection error — so adding one is a one-line improvement and never a fix
// for something that would otherwise be broken.
var managedHostMarkers = []string{
	"RENDER",                     // Render
	"FLY_APP_NAME",               // Fly.io
	"DYNO",                       // Heroku
	"K_SERVICE",                  // Google Cloud Run
	"WEBSITE_INSTANCE_ID",        // Azure App Service
	"ECS_CONTAINER_METADATA_URI", // AWS ECS/Fargate
	"ECS_CONTAINER_METADATA_URI_V4",
}

type Config struct {
	DatabaseURL string
	Port        string
	TZ          string
	MailMode    string
	AdminEmail  string
	// SecureCookies must be true anywhere served over HTTPS; local dev
	// is http, so it is off by default and switched on in production.
	SecureCookies bool
	// DatabaseURLDefaulted reports that DATABASE_URL was unset and the local
	// development fallback is in use. Worth saying out loud: in a deployment
	// that fallback makes the process dial localhost as a user that does not
	// exist there, and "connection refused to 127.0.0.1" is a confusing way
	// to learn that a variable is missing.
	DatabaseURLDefaulted bool
	// ManagedHost reports that a container platform's marker is present, so
	// the defaults meant for a laptop are known to be wrong here.
	ManagedHost bool
	// AllowSignup decides whether the public signup page and endpoint
	// exist. On by default, so a fresh deployment is usable without
	// reaching for the database; set ALLOW_SIGNUP=false to close it once
	// the accounts that should exist do.
	AllowSignup bool
}

func Load() Config {
	databaseURL := cleanDSN(os.Getenv("DATABASE_URL"))
	return Config{
		DatabaseURL:          or(databaseURL, devDatabaseURL),
		DatabaseURLDefaulted: databaseURL == "",
		ManagedHost:          managedHost(),
		Port:                 getenv("PORT", "8090"),
		TZ:                   getenv("TZ", "Asia/Kolkata"),
		MailMode:             getenv("MAIL_MODE", "log"),
		AdminEmail:           getenv("ADMIN_EMAIL", "anishatiwari695@gmail.com"),
		SecureCookies:        getenv("SECURE_COOKIES", "false") == "true",
		AllowSignup:          getenv("ALLOW_SIGNUP", "true") == "true",
	}
}

// Validate reports configuration that cannot work, as distinct from
// configuration that is merely unusual. It is checked before the first
// connection is attempted, so a missing variable is reported as itself
// instead of as whatever the resulting dial happens to fail with.
func (c Config) Validate() error {
	if c.DatabaseURLDefaulted && c.ManagedHost {
		return ErrDatabaseURLMissing
	}
	return nil
}

// cleanDSN removes what a copy-paste into a dashboard field tends to carry:
// surrounding whitespace, and a matching pair of quotes taken along from a
// shell command like DATABASE_URL='postgres://...'. A platform stores the
// literal characters it was given, and pgx then fails on a DSN it cannot
// parse — an error that never mentions quoting. A valid DSN never begins and
// ends with a quote, so removing one pair cannot discard a working value.
func cleanDSN(v string) string {
	v = strings.TrimSpace(v)
	for _, quote := range []string{`"`, "'"} {
		if len(v) >= 2 && strings.HasPrefix(v, quote) && strings.HasSuffix(v, quote) {
			return strings.TrimSpace(v[1 : len(v)-1])
		}
	}
	return v
}

func managedHost() bool {
	for _, key := range managedHostMarkers {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}

func getenv(key, def string) string {
	return or(os.Getenv(key), def)
}

func or(v, def string) string {
	if v != "" {
		return v
	}
	return def
}
