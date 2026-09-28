# Product roadmap — from demo to complete product

This doc answers three questions asked together: is the current DB/API/UI
architecture the right one to build on, what's missing for
scalable/reliable/low-latency/high-throughput, and what features turn this
from a working demo into a complete product. Written against the actual
codebase as of 2026-09-03, not generic advice — every claim below points at
a real file or was verified by reading/running the code.

## Where this stands today

A real (if small-scale) distributed system already: gateway + 5 backend
services + Postgres (schema-isolated per service) + Redis + NATS + OTel
tracing, JWT auth with role-based authorization, a React SPA. See
[`ARCHITECTURE.md`](ARCHITECTURE.md) for the full service map. The gap this
doc is about isn't "is it distributed" — it already is — it's "is it
*production*-shaped."

---

## Part 1 — Is the architecture right?

Short answer: **the shape is right, the depth in places is demo-grade.**
Nothing here needs a rewrite; several specific things need building out.
Graded against your four asks (scalable / reliable / low-latency /
high-throughput):

### Database — right shape, one real bottleneck

**What's good:** per-service Postgres schemas with role-level
`GRANT`/`REVOKE` isolation (`db/init/00_roles.sql`) is a genuinely solid
pattern — it's "logical microservices, physical monolith DB," which is the
correct starting point before physical database-per-service is worth its
complexity tax. Indexes exist on the columns that matter
(`subject_name`, `subject_type`, `region`, `start_date`).

**The real bottleneck:** `analytics-service` doesn't aggregate in SQL. It
calls campaigns-service's unpaginated `/internal/campaigns` over HTTP,
pulls every matching row as JSON, and sums/groups in Go memory
(`services/analytics/internal/aggregate/aggregate.go`). At today's ~65 rows
this is invisible. At real scale (hundreds of thousands of campaigns) this
is the first thing that falls over — network payload size, JSON decode
cost, and O(n) in-memory aggregation on every KPI/trend/region/benchmark
request (Redis caches the *result*, but every cache miss redoes all of
this). **Fix, in order of effort:** push aggregation into SQL
(`GROUP BY` queries campaigns-service exposes as new internal endpoints) →
materialized views refreshed on `campaign.created` → if it outgrows
Postgres, a columnar store (ClickHouse/DuckDB) fed by the same NATS event
stream, queried by analytics-service instead of the HTTP+in-memory path.

**Other gaps:** no migration tool (raw sequential `.sql` files, run by
hand — fine solo, risky with more than one person changing schema); no
`pgxpool` tuning (`MaxConns` etc. — every service uses driver defaults,
unverified against how many replicas × connections Postgres can actually
hold); no connection pooler (PgBouncer) in front of Postgres, which matters
once you're running N replicas of 3 different services all holding their
own pool.

### APIs — solid contract, no failure isolation between services

**What's good:** one consistent error envelope, a real gateway as the
single ingress + LB + auth boundary, load-balancing + health-checked
failover already proven for campaigns-service
(`services/gateway/internal/lb`).

**The reliability gap:** `platform/client.go`'s inter-service HTTP client
is a plain 5-second-timeout client — no retry, no circuit breaker. If
catalog-service gets slow, every `POST /campaigns` (which calls it
synchronously to validate the subject) queues up behind that 5s timeout
instead of failing fast or retrying with backoff. Under load this is
exactly how one slow service takes down another. Worth adding: a small
retry-with-backoff wrapper for idempotent (GET) inter-service calls, and a
circuit breaker (open after N consecutive failures, half-open probe) so a
degraded catalog-service fails campaign-creation fast instead of piling up
latency.

**The single-point-of-failure gap:** only campaigns-service is
load-balanced. catalog-service, analytics-service, auth-service, and
notifications-service are each a single instance (`proxy.Fixed` in
`services/gateway/internal/api/server.go`) — no redundancy. Extending the
same `lb.Pool` pattern already built and proven for campaigns-service to
the other four is mechanical, not a new pattern to invent.
notifications-service is the one service that's *already* safe to run
N-wide with no extra work — it subscribes to NATS independently per
instance and only needs to reach the WebSocket clients connected to itself
(`services/notifications/cmd/server/main.go`), so horizontal scaling there
is free once it's behind an LB too.

**Also missing:** rate limiting (`docs/API_CONTRACT.md` documents a `429`
that nothing returns — flagged and deliberately deferred earlier in this
project, still open); no OpenAPI/Swagger spec (the contract is a
hand-maintained markdown doc — fine today, a real liability once more than
one person/client integrates against it).

### Frontend — fine at 5 tabs, won't survive 15

**What's good:** the tab split fetches only what's on screen
(`frontend/src/tabs/*.jsx`), the color/data-viz system is now validated
rather than eyeballed, auth is wired end-to-end.

**What breaks as the product grows:** no router (tabs are React state, not
URLs — you can't bookmark or share a link to "Region, filtered to
Mumbai"; browser back/forward doesn't work); no data-fetching library
(no request de-duplication or caching across components — each tab
refetches on every filter change, already flagged as a known gap in
`frontend/README.md`); one JS bundle, no code-splitting (every build warns
about this — `dist/assets/index-*.js` is ~570kB); **zero frontend tests**
(the backend has real coverage now — services/*/internal/*/​*_test.go — the
frontend has none). None of these are urgent at 5 tabs and one screen of
filters; all of them are the first things to hurt once the product doubles
in surface area.

### The honest summary

Nothing here is architecturally wrong. It's a real distributed system with
the right boundaries. What's missing is exactly what you'd expect from a
project that went from monolith → microservices → auth → polish in one
continuous build: failure isolation between services, an aggregation path
that survives real data volume, and the frontend infrastructure (router,
data-fetching, tests, code-splitting) that a 5-tab demo doesn't need but a
real product does.

---

## Part 2 — Feature breakdown, in build order

Grouped so each phase is shippable on its own, not a checklist to do all
at once.

### Phase A — Harden what exists (no new features, makes everything above true)

**Done (2026-09-04).** All five items shipped and verified against the
live stack, not just built:

- **SQL-side aggregation** — `services/campaigns/internal/store/aggregates.go`
  (`KPITotals`, `WeeklyBuckets`, `TrendBuckets`, `RegionCounts`,
  `BenchmarkRows`), served over new `/internal/aggregates/*` routes,
  replacing the old `/internal/campaigns` full-row-dump (removed).
  analytics-service's public response shapes are unchanged — verified by
  diffing `/api/v1/kpis|trend|regions|benchmark` against the same live
  data before/after.
- **Retry + circuit breaker** — `platform/retry.go` + `platform/breaker.go`,
  wrapping `NewInternalClient`'s transport for every inter-service call.
  4xx is never treated as a failure (only transport errors / 5xx trip the
  breaker) — a bad subject name doesn't look like an unhealthy
  catalog-service.
- **Load-balanced the rest** — catalog/analytics/auth now run behind the
  same `lb.Pool` campaigns-service already used; verified live by killing
  one replica of each and confirming traffic kept flowing.
- **Rate limiting** — `platform/ratelimit.go`, per-user for `/api/v1/*`
  (post-auth), per-IP for `/auth/*` (brute-force-prone, pre-identity).
  Verified live: burst allowed through, then real `429`s.
- **Frontend** — React Router (filters now live in the URL, bookmarkable/
  shareable/back-button-safe), TanStack Query (replaced hand-rolled
  fetch+cancelled-flag state in every tab), route-based code-splitting
  (main bundle dropped from ~570kB to ~236kB, chunk-size build warning
  gone), first test suite (Vitest + Testing Library, 16 tests). Caught and
  fixed a real bug in the process: the search-box debounce fired an
  unconditional `setQuery("")` on every mount, which could race a
  concurrent filter click and silently drop it — found via a live
  Playwright reproduction, not by inspection.

### Phase B — Depth on what the product already does
- **Real ad-platform ingestion** — the schema already anticipates this
  (`campaigns.source`/`external_id`, discussed earlier this session).
  Start with one connector (Google Ads is the most standardized API),
  land it as a new `services/ingestion` worker that writes through
  campaigns-service, not a bypass.
- **Budget pacing & alerts** — **done (2026-09-06).** `campaigns.campaigns`
  gained a nullable `budget_paise`; pacing (`under`/`on`/`over`) is derived
  at read time from budget/spend/elapsed flight, never stored — see
  `docs/ARCHITECTURE.md`'s Budget pacing section for the exact formula.
  Set via a new admin-only `PATCH /campaigns/{id}` (there's still no
  campaign-creation UI, so this is the only way to assign one today —
  an inline control in the campaign drawer). Surfaced in the Campaigns
  table (Budget column + pacing badge) and a compact "N over/under pace"
  highlight on Overview, computed client-side from a bounded 50-row
  sample rather than a new aggregate endpoint (this app's total campaign
  count is in the dozens — noted as a real limitation if data volume
  grows). Found and fixed a real bug along the way: the shared CORS
  middleware (`platform/httpmw.go`) only allowed `GET, POST, OPTIONS`,
  so the browser's PATCH preflight silently failed until fixed — every
  service picked up the fix since they all share that one middleware.
- **Anomaly detection** — **done (2026-09-12).** Redefined from the
  original wording once it met the actual schema: a campaign row is an
  immutable snapshot (`reach`/`spend_paise` set once, never incremented),
  so there's no per-campaign time series to spike or dip against. Built
  instead: peer-outlier detection (a campaign's reach/spend far from its
  own category's median, `services/campaigns/internal/store/anomalies.go`)
  and stale-category detection (a category with real history that hasn't
  started a new campaign in 14+ days — campaign *creation* is a genuine
  time series at the aggregate level). `GET /campaigns/anomalies`
  deliberately ignores the shared filter params — see
  `docs/ARCHITECTURE.md`'s Anomaly detection section for the exact
  thresholds and why. Verified live by cross-checking the SQL's flagged
  rows against direct `psql` queries (same discipline Phase A's
  aggregates got), and in the browser via the Campaigns table's warning
  badge + tooltip and Overview's "N flagged" chip.
- **Scheduled/exported reports** — **done (2026-09-12).** The CSV export
  path (`campaigns-service`'s `/export`) is the report; emailing it is
  delivery, not new data plumbing, as scoped above. A new
  `platform.Mailer` (hand-built MIME over `net/smtp`, zero new
  dependencies) backs both an on-demand `POST /campaigns/email-report`
  (any authenticated role, emails the caller's own filtered view to their
  own account email) and an opt-in scheduled ticker
  (`REPORTS_ENABLED`/`REPORTS_RECIPIENT`/`REPORTS_INTERVAL`, same shape as
  the existing `DEMO_TICKER`). **No SMTP credentials exist in this
  environment** — the same shape of gap as the still-blocked ingestion
  item below — so `Mailer` degrades exactly like `platform.Cache`/
  `platform.EventBus` already do for Redis/NATS being down: it never
  fails construction, and reports `{"sent": false, "reason": "..."}`
  instead of a `5xx` or a false success. Verified live twice: once against
  a real (if throwaway) local SMTP listener with delivery actually
  succeeding — correct headers, MIME multipart, CSV attachment — and once
  with SMTP unconfigured, confirming the honest degraded response and
  that the UI shows it plainly instead of claiming success. See
  `docs/ARCHITECTURE.md`'s Scheduled/exported reports section.
- **Campaign approval workflow** — **done (2026-09-12).** A stored
  `approval_status` (`"pending"` \| `"approved"` \| `"rejected"`,
  `db/init/02_campaigns.sql`) — unlike `Status`/`Pacing` this is a
  decision a person makes, not derivable from a row's own dates. **A
  review checkpoint, not a visibility gate** — a `"pending"` campaign is
  still fully visible everywhere, which matters because the demo ticker
  and seed script create most of this app's campaigns by calling
  `Store.Create` directly, bypassing HTTP; hiding pending campaigns would
  have visibly broken the live demo pipeline. Defaults to `'approved'` so
  every existing/demo-ticker/seed row needed zero code changes; only a
  real `POST /campaigns` starts a campaign `"pending"`, hardcoded server-
  side (not a request field an admin could set to skip review). Set via a
  new, deliberately **separate** `PATCH /campaigns/{id}/approval` — not a
  second optional field folded into the existing budget PATCH, which
  would risk one request silently clearing the other. See
  `docs/ARCHITECTURE.md`'s Campaign approval section.

### Phase C — Multi-user / multi-org depth
- **Roles beyond viewer/admin** — **done (2026-09-13).** `editor` and
  `approver` (not the originally-sketched `analyst`, which would have
  been functionally identical to `viewer` — see
  `docs/ARCHITECTURE.md`'s RBAC section for why the real split is
  create-vs-approve, now that the approval workflow exists to approve
  *of*). `platform.RequireRole` is the first shared role-gating helper in
  the codebase, replacing three duplicated inline checks once a fourth
  role and a second service needed different allowed-role sets per route.
  Admin-only user management (`GET /users`, `PATCH /users/{id}/role`)
  shipped alongside it — otherwise the new roles would have had no path
  to actually get assigned beyond the seed data. Surfaced in a newly
  built Settings tab (previously a placeholder).
- **Multi-tenancy** — explicitly deferred by the user this pass (single-
  org is enough for now); still the biggest structural change on this
  list if it's ever needed — an `org_id` on every table, every query
  scoped by it, worth a dedicated design pass before starting.
- **Audit log** — **done (2026-09-13).** A new `audit-service`
  (append-only `audit.audit_log`, fed by a new `campaign.audit` NATS
  subject — deliberately separate from `campaign.created`, so existing
  subscribers see zero change), admin-only `GET /audit-log`. See
  `docs/ARCHITECTURE.md`'s Audit log section.
- **SSO** (SAML/OIDC) — explicitly deferred by the user this pass, same
  reasoning as the still-blocked ad-platform ingestion: needs a real
  identity provider's credentials to integrate against for real, not
  guessed at.

### Phase D — Exploratory / differentiating
- The AI-for-business-logic idea from earlier in this project (data
  extraction, filtering, sorting, response-building) — explicitly
  deferred again this pass; still needs its own scoping conversation
  before it's buildable.
- **Competitor benchmarking over time** — **done (2026-09-13).** An
  opt-in ticker snapshots the whole benchmark table
  (`campaigns.benchmark_snapshots`, append-only, no per-day dedup); the
  top-10 benchmark endpoint's rows each gain up to 8 recent trend points
  via one follow-up query, not a live per-request aggregation. See
  `docs/ARCHITECTURE.md`'s Benchmark trending section.
- **Creative-level tracking** — **done (2026-09-13).** `campaigns.creatives`,
  deliberately additive (a campaign's own reach/spend are never
  recalculated from its creatives' sum — see
  `docs/ARCHITECTURE.md`'s Creative-level tracking section for why).
  Surfaced as a "Creatives" section in the campaign drawer.

---

## Part 3 — Priority call

Original order, kept for context — Phase A, every buildable Phase B item,
and every buildable Phase C/D item are now done (2026-09-04 through
2026-09-13):

1. ~~**Phase A's aggregation + circuit-breaker + LB items**~~ — done.
2. ~~**Frontend router + data-fetching library**~~ — done, part of Phase A.
3. ~~**One real Phase B feature**~~ — all four are now accounted for:
   budget pacing, anomaly detection, scheduled/exported reports, and
   campaign approval are shipped; real ad-platform ingestion remains
   blocked on unanswered questions about live account access (the first
   thing to pick up if that access becomes available).

What's left: real ad-platform ingestion (Phase B, still blocked), and the
three explicitly-deferred-by-the-user Phase C/D items — multi-tenancy,
SSO, and the AI-for-business-logic idea, each blocked on either a real
decision to take on a large structural change or on external credentials
that don't exist in this environment yet. Everything else across all four
phases is shipped and live-verified. Tell me which of these three to pick
up (or if ingestion access has become available) and I'll scope it the
way every pass so far has gone — concrete plan, then build it, then
verify it against the running stack.
