# Architecture

Six independent Go services, one shared `platform` module, one Postgres
database split into per-service schemas. This doc covers service
boundaries, data ownership, and the auth trust model — the things that
don't live naturally in any one service's own comments.

## Services

| Service              | Owns                                  | Port (dev) |
|-----------------------|-----------------------------------------|------------|
| `gateway`              | Routing, load balancing, auth verification | 8080       |
| `catalog-service`      | Brands, people, categories, regions, ad types | 8081 |
| `campaigns-service`    | Campaigns, creatives, benchmark snapshots (2 replicas behind the gateway's LB) | 8082/8083 |
| `analytics-service`    | KPI/trend/region/benchmark aggregation (no data of its own) | 8084 |
| `notifications-service`| WebSocket fan-out of `campaign.created` | 8085 |
| `auth-service`         | Users, roles, sessions (JWT + refresh tokens) | 8086 |
| `audit-service`        | Append-only audit log, fed by `campaign.audit` NATS events | 8090 |

Every service owns exactly one Postgres schema and nothing else — enforced
by role-level `GRANT`/`REVOKE`, not just convention (`db/init/00_roles.sql`).
`campaigns_service` cannot query `catalog.*` even though both live in the
same database; a real network partition (a service in a different data
center, a different vendor) would look identical from the code's point of
view. Cross-service reads go over HTTP (`catalogclient`, `campaignsclient`,
`auth`'s login/refresh), never SQL.

`analytics-service` calls `campaigns-service` only, not `catalog-service` —
campaigns-service's rows already carry subject/category/region/ad-type
as denormalized snapshots (see `db/init/02_campaigns.sql`), so there's
nothing catalog-service could add for aggregation. One fewer service in the
critical path for every KPI/trend/region/benchmark request.

## Caching and invalidation

`catalog-service` and `analytics-service` cache read responses in Redis
(`platform/cache.go`) with a short TTL as a safety net. The real
invalidation is event-driven: `campaigns-service` publishes
`campaign.created` on NATS (`platform/events.go`) after every insert, and
analytics-service's subscriber flushes every cached aggregate on that event
rather than tracking which cached response a given write could have
affected. A down Redis or NATS degrades to "always recompute" / "cache
expires on its own schedule" — never a hard failure; see the comments on
`platform.Cache` and `platform.EventBus`.

## Auth

Sessions are JWT access tokens (15 min) plus opaque, rotating refresh
tokens (30 days) — see `db/init/03_auth.sql` and
`services/auth/internal/api`. Self-serve registration (`POST
/auth/register`) always creates a `viewer` — there's no request field that
could make it an `admin` (or any other role), not just a validation rule
against one (`RegisterRequest` simply has no `role` field). Four users are
seeded for local dev, one per role: `admin@campaigntracker.dev` /
`ChangeMe123!admin`, `editor@campaigntracker.dev` / `ChangeMe123!editor`,
`approver@campaigntracker.dev` / `ChangeMe123!approver`, and
`viewer@campaigntracker.dev` / `ChangeMe123!viewer` — see
`db/init/03_auth.sql`. Elevating a `viewer` beyond that is admin-only,
via `PATCH /api/v1/users/{id}/role` (see the RBAC section below) — there's
no other path to a non-`viewer` role.

**The gateway is the system's one JWT-verification boundary.** It validates
the `Authorization: Bearer` header on every `/api/v1/*` request
(`services/gateway/internal/api/authmw.go`) and, on success, forwards the
caller's identity to whichever service it's proxying to as trusted
`X-User-Id` / `X-User-Email` / `X-User-Role` headers — stripping any
client-supplied versions of those headers first. Downstream services don't
re-parse or re-verify the JWT; they trust the header contract.

This is a deliberate choice, not an oversight: it keeps JWT crypto in one
place instead of five, and it's consistent with a trust model this codebase
already had before auth existed — `catalog-service`'s
`/internal/subjects/lookup` and `campaigns-service`'s `/internal/campaigns`
are unauthenticated today and reachable only because they're not routed
through the gateway's public path. Auth adds one more thing the internal
network is trusted for.

**What this doesn't protect against:** a compromised or malicious service
on the same internal network can forge `X-User-Role: admin` directly
against `campaigns-service`, bypassing the gateway entirely. Closing that
gap — mTLS between services, or having each service verify the JWT itself
instead of trusting a header — is the natural next step if this network
stops being a single trusted boundary (e.g. services spread across
clusters/vendors, or a compliance requirement for defense in depth). Not
built now because nothing in this deployment crosses that boundary yet.

**Why the gateway forwards a token via query param for `/ws/notifications`:**
a browser's native WebSocket client can't set an `Authorization` header on
the upgrade request, so that one route reads `?token=` instead
(`RequireAuthQuery` in `authmw.go`) — same reasoning as why that route
already bypasses the traced mux (`otelhttp`'s `ResponseWriter` isn't a
`http.Hijacker`).

**Why `/auth/refresh` returns a rotated refresh token, not just a new
access token:** the original API sketch in `docs/API_CONTRACT.md` had
`/auth/refresh` return only `{ accessToken }`. Built instead as
single-use/rotating (`{ accessToken, refreshToken }`, old token deleted
server-side in the same request) — a stolen refresh token stops working the
moment its legitimate owner refreshes again, not just at its 30-day expiry.
Same kind of deliberate, contract-updated change as `campaigns-service`'s
`brand`→`subject` rename.

## Budget pacing

A campaign's `budget` (nullable — most don't have one) and derived
`pacing` (`"under"` \| `"on"` \| `"over"`, empty when there's no budget)
live in `services/campaigns/internal/models`' `PacingOf`. Computed at read
time from `budget`, `spend`, and how much of the campaign's start→end
flight has elapsed — never stored, same precedent `Status`
("Live"/"Completed" — `store/postgres.go`'s `statusOf`) already set, so
neither needs a background job to stay in sync as time passes.

`expectedSpend = budget × elapsedFraction` (elapsed clamped to `[0,1]` —
a campaign that hasn't started yet or has already ended still gets a
sensible answer instead of a nonsense fraction). `spend / expectedSpend`
over `1.15` is `"over"`, under `0.85` is `"under"`, otherwise `"on"`.
Thresholds are constants in `models.go`, not config — tightening them is a
one-line change.

There's no campaign-creation UI yet (see `docs/ROADMAP.md`'s Phase B), so
`PATCH /campaigns/{id}` — not a create-time-only field — is how a budget
actually gets set in practice; see `docs/API_CONTRACT.md`.

## Campaign approval

`approval_status` (`"pending"` \| `"approved"` \| `"rejected"`) is a real
stored column (`db/init/02_campaigns.sql`) — unlike `Status`/`Pacing`,
it's a decision a person makes, not derivable from a row's own dates, so
it can't be computed at read time. **It's a review checkpoint, not a
visibility gate**: a `"pending"` campaign is still fully visible in every
list, export, and aggregate — nothing filters it out. That's deliberate,
not an oversight — the demo ticker (`services/campaigns/cmd/server/main.go`)
and the seed script (`services/campaigns/cmd/seed/main.go`) both create
the bulk of this app's campaigns by calling `Store.Create` directly,
bypassing HTTP entirely; if "pending" hid a campaign until approved, the
live demo/notification pipeline would visibly stop working the moment
this shipped.

The column defaults to `'approved'` at the database level
(`Store.Create` resolves an empty `ApprovalStatus` to `"approved"` before
inserting), so every existing row and every future demo-ticker/seed row
needs zero code changes to keep behaving exactly as before. Only
`handleCreateCampaign` — the real `POST /campaigns` path — explicitly
sets `"pending"`, and it's hardcoded there, not a request field: an admin
can't self-approve at creation time through the API.

Setting it is a **separate** route from the budget PATCH
(`PATCH /campaigns/{id}/approval`, see `docs/API_CONTRACT.md`), not a
second optional field on the same one. Folding both into one PATCH body
would mean distinguishing "the caller didn't mention this field" from
"the caller explicitly cleared it" for two independent fields at once — a
real correctness risk (a request that only sets approval status could
silently reset the budget) for no benefit over one small new route.

## Anomaly detection

The roadmap's original wording ("spend spike, reach cliff") doesn't fit
this schema honestly: a campaign row is an immutable snapshot — `reach`/
`spend_paise` are set once at `Create` and never incremented — so there's
no per-campaign time series to spike or dip against. What's actually
computable, and built instead (`services/campaigns/internal/store/anomalies.go`):

- **Peer outliers** — a campaign whose `reach` or `spend_paise` is far
  (`> 2.5×` or `< 0.4×`, see `models.AnomalyRatioHigh`/`Low`) from its own
  `subject_category`'s median, computed via a `percentile_cont` CTE
  against the *whole* category (not the caller's current filters — see
  below), only when that category has `≥4` peers
  (`models.AnomalyMinPeers`) to make "median" meaningful.
- **Stale categories** — a category with real history
  (`≥3` campaigns ever, `models.StaleCategoryMinHistory`) that hasn't
  started a new one in 14+ days (`models.StaleCategoryDays`). Campaign
  *creation* is a genuine time series at the aggregate level even though
  any single campaign's own numbers aren't.

`GET /campaigns/anomalies` deliberately **ignores every filter param**
the rest of the API shares — this is a global "what needs attention"
signal, not a view of whatever's currently on screen, the same reasoning
a notifications inbox doesn't empty out when you scroll or filter. The
frontend fetches it once (`["anomalies"]`, React Query dedupes this
across the Campaigns table, the drawer, and Overview) and cross-references
by campaign id, the same "one bounded query, cross-referenced
client-side" shape Overview's budget-alert sample already established.

## Scheduled/exported reports

`GET /campaigns/export`'s CSV path is the actual "report" — emailing it
is delivery, not new data plumbing, per the roadmap's own scoping. Two
ways to trigger it, sharing one `Server.GenerateAndSendReport` method so
neither can format a report differently:

- **On demand** — `POST /campaigns/email-report`, any authenticated role.
  Filtered by the query string (identical to export), sent to the
  *caller's own* account email (trusted `X-User-Email`, never a
  client-supplied recipient).
- **Scheduled** — an opt-in ticker in `services/campaigns/cmd/server/main.go`
  (`REPORTS_ENABLED`/`REPORTS_RECIPIENT`/`REPORTS_INTERVAL`, same
  `time.NewTicker` shape as the existing `DEMO_TICKER`), unfiltered, to a
  fixed configured recipient. Refuses to start (logs a warning, doesn't
  guess) if enabled with no recipient set.

Delivery goes through `platform.Mailer` (`platform/mail.go`) — a hand-built
`multipart/mixed` MIME message over `net/smtp.SendMail`, zero new
dependencies. **No SMTP credentials exist in this environment**, the same
shape of gap as the still-blocked ad-platform ingestion item. `Mailer` is
built to degrade exactly like `platform.Cache`/`platform.EventBus` already
do for Redis/NATS being down: construction never fails
(`platform.NewMailer` always returns a usable, non-nil value even with an
empty host), and `Send` returns a typed `ErrMailerNotConfigured` instead
of attempting a network call when unconfigured. Both report routes turn
that into an honest `{"sent": false, "reason": "..."}` rather than a
`5xx` or a false "sent" — this is an expected environment gap, not a
server error.

## RBAC — roles beyond viewer/admin

Four roles now (`platform/roles.go`'s `RoleViewer`/`RoleEditor`/
`RoleApprover`/`RoleAdmin`), mapped onto the system's actual write
actions:

| Action                        | viewer | editor | approver | admin |
|--------------------------------|:---:|:---:|:---:|:---:|
| Read / export / email-report   | ✓ | ✓ | ✓ | ✓ |
| Create a campaign, set its budget | | ✓ | | ✓ |
| Approve/reject a campaign      | | | ✓ | ✓ |
| Manage user roles              | | | | ✓ |

`editor` and `approver` are a deliberate separation of duties, not the
roadmap's originally-sketched "analyst (read + export, no create)" — that
role would have been functionally identical to `viewer`. Now that the
approval workflow exists, splitting "can propose a campaign" from "can
sign off on one" is the genuinely useful split.

`platform.RequireRole(w, r, message, allowed...)` is the first shared
role-gating helper in the codebase — it replaced three identical inline
`if role != "admin"` checks in campaigns-service once a fourth role and a
second service (auth-service's new user-management routes) needed
different allowed-role sets per route; the codebase's own earlier comments
had explicitly deferred a shared helper "given there are currently only
two admin-gated routes" — this is past that threshold. `platform.RoleValid`
is the single source of truth for which role strings exist, checked
wherever a role is *assigned* (`handleUpdateUserRole`) so a bad value is a
clean `400`, not a raw Postgres `CHECK`-constraint violation.

**User management** (`GET /api/v1/users`, `PATCH /api/v1/users/{id}/role`,
both admin-only) lives under `/api/v1/`, not `/auth/`, on purpose: every
other `/auth/*` route is pre-identity/public at the gateway, but these need
an already-verified admin session, so they belong with everything else
`protected()` wraps. `handleUpdateUserRole` refuses to let an admin change
their *own* role — a cheap, real guard against accidentally locking
yourself out; not full last-admin protection (that would need a count
query this app's scale doesn't need yet).

## Audit log

A new `audit-service` — one small dedicated service per cross-cutting
concern, the same shape `notifications-service` already established
(subscribes to NATS, does its one job, isn't owned by the service whose
data triggered the event). `campaigns-service` publishes a **new, separate**
NATS subject, `campaign.audit` (never touching the existing
`campaign.created` payload/subject that notifications-service and
analytics-service's cache invalidation already depend on), for all three
write actions plus the demo ticker's synthetic creates — the ticker has no
real caller to attribute a row to, so it publishes with
`actorId: "system"`, `actorRole: "system"` rather than leaving the audit
trail looking mysteriously incomplete for its rows.

`audit-service` owns its own schema (`audit`, `db/init/04_audit.sql`),
subscribes to `campaign.audit`, and serves one admin-only read route,
`GET /api/v1/audit-log` (paginated, optional `campaignId` filter).
**Unlike notifications-service, a NATS-connect failure here is a warning,
not fatal** — a WebSocket hub with no event feed is pointless, but
audit-service's read API over already-stored rows stays useful even if
new events stop arriving temporarily. Kept single-instance for now
(stateless HTTP + one subscriber, trivial to scale later) — but NATS core
pub/sub delivers to *every* subscriber, not a load-balanced queue, so
running N replicas today would double-process every event; that would
need a queue-group subscribe method added to `platform.EventBus` first,
not built since nothing needs that scale yet.

## Benchmark trending

`campaigns.benchmark_snapshots` (`db/init/02_campaigns.sql`) is a plain,
append-only fact table — one row per subject per snapshot, recorded by an
opt-in ticker (`BENCHMARK_SNAPSHOT_ENABLED`/`BENCHMARK_SNAPSHOT_INTERVAL`,
same shape as `DEMO_TICKER`/the scheduled-reports ticker) that snapshots
the *whole* table (not the top-10 cap the benchmark endpoint itself uses —
trending needs full history to re-rank later). Deliberately **no unique
constraint / upsert-per-day**: that would collapse every snapshot taken on
the same calendar day into one row, useful in production (the ticker just
runs daily) but impossible to verify live within a single dev session.

`GET /benchmark`'s existing top-10 rows each gain a `trend` field — up to
8 recent snapshot points, oldest → newest — via **one follow-up query**
for exactly the subjects the main query already selected (a window
function ranks each subject's most recent snapshots), not one query per
subject and not a live per-request aggregation. Omitted (not a flat line)
until at least one snapshot exists.

## Creative-level tracking

`campaigns.creatives` (`db/init/02_campaigns.sql`) lets a campaign carry
multiple creatives — real ad platforms do. **Deliberately additive, not
decomposed**: a campaign's own `reach`/`spend` are never recalculated from
its creatives' sum. Reconciling the two would need either a DB trigger or
application-level consistency logic — real added risk (double-counting,
drift) for a feature whose actual ask was "track creatives exist," not
"replace campaign-level totals."

`GET /campaigns/{id}/creatives` is open to any authenticated role (same as
reading the campaign itself); `POST /campaigns/{id}/creatives` is
`editor`-or-`admin` — the same content-edit tier as creating the campaign.
Both check the parent campaign exists via `Store.ByID` first, so a bad id
is a clean `404` rather than a raw foreign-key-violation `500`.
