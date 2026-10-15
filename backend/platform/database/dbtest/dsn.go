// Package dbtest resolves which database the Go tests run against.
//
// It exists because they used to run against the same one as
// everything else, including the Playwright suite — which truncates
// and reseeds on startup. Run the two together and the reseed wipes
// the Go fixtures mid-test, or the Go cleanup deletes rows Playwright
// is asserting on. The failures that produced looked like real
// regressions and had to be investigated as such before being
// recognised as contention, which is the expensive kind of flake: it
// costs a detour every time, and it teaches you to distrust the suite.
//
// So the DB-backed Go tests get their own database. Nothing else
// writes to it, which also means a test is free to leave a mess if it
// crashes without poisoning anyone's dev data.
package dbtest

import (
	"os"
	"strings"
)

// DefaultDSN points at a database used by nothing but these tests.
const DefaultDSN = "postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro_test?sslmode=disable"

// DSN returns the database the tests should use.
//
// TEST_DATABASE_URL wins, so CI can point somewhere of its own. Then
// the default. DATABASE_URL is deliberately NOT consulted: that is the
// dev database, the thing being kept away from, and silently falling
// back to it would reintroduce the collision the moment the variable
// happened to be exported in a shell.
func DSN() string {
	if v := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")); v != "" {
		return v
	}
	return DefaultDSN
}
