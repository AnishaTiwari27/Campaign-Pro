package arch_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// orgOwned are the tables that carry org_id. Anything reading or writing
// one of them has to say which organisation it means.
var orgOwned = []string{"campaigns", "creators", "reports", "users", "benchmark_snapshots", "ingest_runs", "launches", "subscriptions"}

// exempt lists the queries that legitimately run without an organisation,
// each with the reason. Keeping them here rather than as a comment in the
// SQL means adding one is a visible change to this list, which is the
// point: an exemption is a hole in the tenant boundary and should be
// argued for in review.
var exempt = map[string]string{
	// Login is how the organisation is discovered, so it cannot already
	// know it. The service refuses an ambiguous match rather than
	// picking one — see identity.
	"GetUserByEmail": "resolves the tenant; cannot presuppose it",
	// The rest of the authentication bootstrap. These run before an
	// organisation is known, because resolving the principal is how it
	// becomes known. All four are keyed by a primary key or a session
	// token, so none of them can be made to range over a tenant.
	"GetUserForAuth": "resolves the tenant at login; cannot presuppose it",
	"SessionUser":    "resolves the principal the org is then read from",
	"GetUserByID":    "keyed by a primary key resolved during auth",
	"TouchUserLogin": "keyed by a primary key resolved during auth",
	// Changes the caller's own name, keyed by the id resolved from
	// their session. There is no parameter that could name another
	// tenant's row.
	"SetUserName": "keyed by the caller's own id, resolved from their session",
	// Operate on a row already fetched under an org-scoped read, keyed
	// by a primary key the caller could only have learned from one.
	"SetCampaignCreator":  "writes a column on an already-scoped row",
	"SetCreatorPhoto":     "writes a column on an already-scoped row",
	"DeleteGrantsForUser": "keyed by a user id already resolved in-org",
	"GrantCampaign":       "keyed by ids already resolved in-org",
	"ListGrantsForUser":   "keyed by a user id already resolved in-org",
	// The id comes from StartIngestRun earlier in the same call and is
	// never taken from user input, so it cannot name another tenant's run.
	"FinishIngestRun": "closes a run this call opened in-org",
	// A provider webhook arrives with no session and so no tenant. The
	// provider's own subscription id is the only thing tying the event
	// to a row, and it was written onto that row by an org-scoped call
	// when the subscription was created. The statement also refuses an
	// empty id, so it cannot degenerate into updating every row that
	// has never been linked to a provider.
	"SetSubscriptionStatus": "keyed by a provider id stored during an in-org write; webhooks carry no tenant",
}

// splitQueries breaks a .sql file into (name, body) pairs.
func splitQueries(src string) map[string]string {
	out := map[string]string{}
	var name string
	var body []string
	flush := func() {
		if name != "" {
			out[name] = strings.Join(body, "\n")
		}
		body = nil
	}
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(line, "-- name:") {
			flush()
			fields := strings.Fields(strings.TrimPrefix(line, "-- name:"))
			if len(fields) > 0 {
				name = fields[0]
			}
			continue
		}
		body = append(body, line)
	}
	flush()
	return out
}

// mentionsOrgOwnedTable is deliberately crude — a word match against the
// table names. A false positive costs an org_id nobody needed; a false
// negative is a tenant leak, so it errs towards flagging.
func mentionsOrgOwnedTable(body string) bool {
	lower := strings.ToLower(body)
	for _, t := range orgOwned {
		for _, kw := range []string{"from " + t, "join " + t, "into " + t, "update " + t} {
			if strings.Contains(lower, kw) {
				return true
			}
		}
	}
	return false
}

// The tenant boundary is a column, which means it is only as good as the
// WHERE clauses that mention it. Discipline does not scale across thirty
// queries and everything added later, so this fails the build instead:
// touch an org-owned table without naming an organisation and the test
// names the query.
func TestEveryOrgOwnedQueryNamesItsOrganisation(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(root, "db", "queries")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// The layout moved; better to say so than to silently pass.
		t.Fatalf("cannot read %s: %v", dir, err)
	}

	var checked int
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for name, body := range splitQueries(string(src)) {
			if !mentionsOrgOwnedTable(body) {
				continue
			}
			if _, ok := exempt[name]; ok {
				continue
			}
			checked++
			if !strings.Contains(strings.ToLower(body), "org_id") {
				t.Errorf("%s/%s touches an org-owned table without naming an organisation —\n"+
					"add the org predicate, or add it to the exempt list with a reason", e.Name(), name)
			}
		}
	}
	if checked == 0 {
		t.Error("checked no queries — the splitter or the table list has gone stale")
	}
}
