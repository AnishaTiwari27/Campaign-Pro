# Campaign Tracker Pro

An internal console for monitoring ad campaigns in the Indian market. It shows
what's live, what's waiting on a decision, what anomaly detection has flagged,
how campaigns compare against category benchmarks, and it emails scheduled
CSV reports.

Go + chi + Postgres on the back, React + TypeScript + Vite on the front.

---

## Running it

The verified path is native Postgres — the machine this was built on has no
Docker, so `docker-compose.yml` and `backend/Dockerfile` (reachable via
`make dev-docker`) are written to spec but have never been executed. Use them
as a starting point, not a tested artifact.

**Prerequisites:** Go 1.25+, Node 20+, PostgreSQL 16, and
[`golang-migrate`](https://github.com/golang-migrate/migrate) (`brew install golang-migrate`).
`sqlc` is only needed if you change the SQL (`brew install sqlc`).

```bash
make db-create   # creates the campaign_tracker_pro database + role
make migrate     # applies backend/db/migrations
make seed        # 35 campaigns, 7 creators, 3 reports, 3 users
make dev         # api on :8090, web on :5173
```

Then open the web dev server (Vite prints the port — 5173 unless taken) and
sign in. `make seed` writes three accounts, all sharing one password that is a
deliberately loud placeholder and should never reach a real deployment:

| Email | Role | Can approve? |
|---|---|---|
| `$ADMIN_EMAIL` (default `anishatiwari695@gmail.com`) | admin | yes |
| `approver@campaigntracker.test` | approver | yes |
| `analyst@campaigntracker.test` | analyst | no — every decision control is hidden, and the API returns `403` |

Password for all three: `demo-password-change-me`.

| Target | What it does |
|---|---|
| `make dev` | API and web together (`dev-api` / `dev-web` run one at a time) |
| `make migrate` · `make migrate-down` | Apply migrations · roll back one |
| `make seed` | Truncate and reseed demo data |
| `make logos` | Re-fetch brand logos into `frontend/public/logos`. They are committed, so this is only needed when adding a brand — a build must not depend on fetching them |
| `make test` | Go tests, Vitest, Playwright — **note this reseeds**, since `test-e2e` depends on `seed` |
| `make lint` · `make fmt` | `go vet` + `gofmt` check + ESLint · rewrite with `gofmt` |
| `make generate` | Regenerate sqlc code from `backend/db/queries` |
| `make dev-docker` | The whole stack via docker-compose (untested — see above) |

### Configuration

| Env var | Default |
|---|---|
| `DATABASE_URL` | `postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable` |
| `PORT` | `8090` |
| `TZ` | `Asia/Kolkata` |
| `MAIL_MODE` | `log` (the only implementation; writes the send to the log) |
| `SECURE_COOKIES` | `false` — must be `true` anywhere served over HTTPS, so the session cookie is not sent in clear |
| `ADMIN_EMAIL` | `anishatiwari695@gmail.com` — the email `make seed` gives the admin account. Read by the seeder only; the running server does not treat it as privileged |

---

## Architecture

```
                ┌────────────────────────────────────┐
  browser ────▶ │  frontend (Vite dev server :5173)  │
                │  React 18 · TS · Router v6         │
                │  TanStack Query · Zustand          │
                └─────────────────┬──────────────────┘
                                  │ /api proxied to :8090
                                  ▼
  ╔══ backend (:8090) ═══════════════════════════════════════════╗
  ║  cmd/server   composition root — the only file that knows     ║
  ║               all five services exist                         ║
  ╟───────┬──────────┬──────────┬───────────┬────────────────────╢
  ║   identity   campaigns   creators   analytics   reports       ║
  ║   sessions,  the core    tier       overview,   cadence,      ║
  ║   roles,     data +      metrics    benchmarks, scheduler,    ║
  ║   login      formulas               regions,    run history   ║
  ║                                     search                    ║
  ║      each: contract · module/ (factory) · internal/{api,      ║
  ║            service, store}                                    ║
  ╟───────────────────────────────────────────────────────────────╢
  ║  platform   config · database · httpx · mail · units · arch    ║
  ║             shared by every service, aware of none of them     ║
  ║  reports' worker ticks every 30s: anomaly detection, then      ║
  ║  any report whose next occurrence has passed                   ║
  ╚═══════════════════════════┬═══════════════════════════════════╝
                              ▼
                     ┌──────────────────┐
                     │   Postgres 16    │
                     └──────────────────┘
```

**The layering rule, per service:** the contract at the service root holds
what crosses a boundary — the types siblings receive and every metric formula,
importing nothing but the standard library so the formulas are unit-testable
with no database. `internal/store` maps sqlc's generated structs onto contract
types at the boundary, so no `pgtype` escapes it. `internal/service` composes
the two and owns the sentinel errors. `internal/api` only parses requests and
writes DTOs. Everything under `internal/` is unreachable from any other
service, enforced by the Go toolchain rather than by convention.

Full detail, including why this is one process rather than eight, is in
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

`frontend/src/lib/metrics.ts` mirrors `backend/services/campaigns/metrics.go` so the
frontend can rebuild a campaign's reach curve for its chart without the API
precomputing chart-shaped data. Both sides are tested against the same
fixtures.

---

## API

Everything is under `/api` and returns JSON. Validation errors are
`400 {error, field}`; a non-approver hitting a decision endpoint gets `403`;
an approver trying to approve a campaign whose budget state forbids it gets
`409` with the reason to show them (see **Approval guard**).

| Method | Path | Purpose |
|---|---|---|
| POST | `/auth/login` | `{email, password}` → sets the session cookie. **The only route outside the session guard** — requiring a session to sign in would be circular |
| POST | `/auth/logout` | Revokes the session server-side, then clears the cookie |
| GET | `/campaigns` | List. Params: `q, type, category, region, adType, status, approval, range(7\|30), sort, dir, page, per`. Returns `{items, total, page, pages}` with pace, cpm, index and categoryIndex derived per row. |
| GET | `/campaigns/:id` | Full record + creatives + audit + 5 similar + `position`/`total`/`prevId`/`nextId` within the caller's current filter+sort |
| POST | `/campaigns/:id/decision` | `{action: approve\|reject\|reopen}` |
| POST | `/campaigns/bulk-decision` | `{ids: [], action: approve\|reject}` |
| POST | `/campaigns/:id/pause` | Toggles live ↔ paused |
| POST | `/campaigns/:id/notes` | `{text}` → writes a user audit event |
| GET | `/campaigns/anomalies` | Currently flagged campaigns |
| GET | `/campaigns/export.csv` | Same filters as the list, as a CSV download |
| GET | `/overview` | KPIs, sparkline series, spotlight, and the four ribbons |
| GET | `/benchmark` | Category benchmarks + MED_ALL |
| GET | `/creators` · `/creators/:id` | The roster with tier-normalised performance; detail adds that creator's campaigns |
| GET | `/regions` · `/regions/:id` | Rollups; detail adds campaigns by reach, category breakdown, pending count, budget share |
| GET/POST | `/reports` | List (with each report's last run) · create from a filter snapshot |
| GET/PATCH | `/reports/:id` | Read · edit name, enabled, cadence, recipients, columns |
| GET | `/reports/:id/runs` | Run history |
| POST | `/reports/:id/run` | Run now — emails real recipients, writes a run row |
| POST | `/reports/:id/test` | Send a copy to the caller only; no run row |
| GET | `/search?q=` | Command palette: sections, campaigns, regions, actions |
| GET | `/me` | The signed-in account: role, `isAgency`, `canApprove`, `isClient` |
| GET | `/health` | Data-source health. Mounted twice: at `/health` **outside** the guard, so a readiness probe needs no credentials, and at `/api/health` behind it |

### Auth and roles

Sessions are real and server-side. `POST /auth/login` verifies the password,
stores a session row and sets an HTTP-only cookie; `POST /auth/logout` deletes
the row before clearing the cookie, so logout genuinely revokes access where a
JWT would stay valid until expiry no matter what the user clicked. Only the
token's hash is stored, so a dump of the table hands nobody a working session.
Login failures return one message for both a wrong password and an unknown
email, so the response cannot be used to enumerate accounts. Sessions last
seven days (`identity.SessionTTL`).

`httpx.RequireSession` resolves the cookie to a principal and puts it on the
request context; handlers read it with `UserFromContext`. Platform takes the
lookup as a `SessionAuthenticator` interface, so it never learns which service
owns users.

The role model lives in `backend/services/identity/identity.go`, and every
permission is a method on `User` rather than a check scattered through
handlers:

| Role | Reads | Decides | Manages users |
|---|---|---|---|
| `admin` | everything | yes | intended, **not implemented** |
| `approver` | everything | yes | no |
| `analyst` | everything | no | no |
| `client` | own account only — **not implemented** | never, by design | no |
| `viewer` | legacy, behaves as `analyst`; kept so pre-migration rows stay meaningful | no | no |

- `CanApprove()` is `isAgency && (admin || approver)`, enforced in the service
  layer, not just hidden in the UI. A client never approves whatever else is
  granted: sign-off is the agency's control over spend, and a client approving
  their own campaign defeats it.
- Two honest gaps. `CanManageUsers()` is defined and never called: there is no
  user-management endpoint or screen, so accounts come from `make seed` only.
  And `IsClient()` is called once, to populate `/me`, but nothing acts on what
  it returns. Client scoping is designed and not built: restricting an external
  user to their own accounts, suppressing benchmarks and stripping competitor
  names all remain to do, and there is no `client` user in the seed. Today this
  is an internal agency console with two effective permission levels — decide,
  or read-only.
- There is also no `created_by` on a campaign, so nothing can tell who
  submitted one. Four-eyes approval — stopping someone signing off their own
  submission — needs that column before it can be enforced.

---

## Metrics

Units: **reach is stored in lakh** (1L = 100,000 people); **money is whole
rupees**. Defined once in `backend/services/campaigns/metrics.go`, mirrored in
`frontend/src/lib/metrics.ts`.

| Metric | Definition |
|---|---|
| **Reach curve** | 8 cumulative points = `reach × weights[shape]`. fast `.10 .34 .55 .70 .81 .89 .95 1`; steady `.08 .20 .33 .46 .59 .72 .86 1`; slow `.04 .09 .17 .28 .42 .60 .79 1` |
| **Budget pace** | `round(spend / budget × 100)`. ≥100 over (red), ≥85 warn (amber), else good (green) |
| **CPM** | `spend / (reach × 100000 × frequency / 1000)`; renders `—` at zero |
| **Running set** | Every campaign whose status is not `scheduled` (so `ended` still counts) |
| **MED_ALL** | Median reach of the running set — the baseline behind every index in the app |
| **Index** | `reach / MED_ALL`, shown as `1.4x` |
| **Category benchmark** | Per category over the running set: `n`, median reach, top campaign, total spend. Sorted by median desc |
| **Category index** | `reach / category median` |
| **Movers** | Running campaigns with index ≥ 1.3, sorted by index desc |
| **Spotlight** | The flagged campaign with the highest index; falls back to the first campaign |
| **Region rollup** | Per region: campaign count, live count, total reach, total spend, ad-type split sorted desc |
| **Sparklines** | Sampled at days-ago offsets `[21,18,15,12,9,6,3,0]`; a campaign's value at an age is interpolated along its own curve. Growth = % change first→last |
| **Anomalies** | pace ≥ 100 → "Over budget pace"; ≥ 95 → "Budget nearly exhausted"; category index ≥ 1.8 → "Reach outlier"; index ≤ 0.3 after ≥ 2 days → "Under-delivering". Checked in that priority order |
| **Approval guard** | `ApprovalBlock` refuses an approval at pace ≥ 95 — the same two thresholds as the budget anomalies, from shared constants so the flag and the refusal cannot drift. The reach flags stay advisory |

Two deliberate modelling notes:

- There's no stored time series, so sparklines reconstruct history by walking
  each campaign's own reach curve backwards, and **spend is assumed to accrue
  on the same curve shape as reach** — the only timing model a campaign has.
- Anomaly flags are never seeded by hand. `make seed` writes campaigns and
  then runs the real detector, so the 6 flagged campaigns are whatever the
  formulas actually produce from the seeded numbers — change a seeded budget
  and that number changes with it.

### The audit trail

Every approve, reject, reopen, pause, resume and note writes an `audit_events`
row through one store method, so the Activity tab and the trail are the same
source of truth. Each row carries both `user_id` — the account, which is what
answers "who approved this?" — and `actor`, the display label it was recorded
under. Both, because a name can change and a trail that rewrites history is not
a trail, while a label alone cannot be joined back to an account. The HTTP
layer builds that pair once, in `actorOf`, and the store takes the user id as a
required parameter so a caller cannot write an unattributed decision by
omission. Rows the system wrote (anomaly flags) have no `user_id`.

Sign-ins, failed sign-ins and sign-outs land in the same table, written by
identity through a one-method `Auditor` capability that the composition root
satisfies with a bridge — so identity records security events without importing
the service that owns the table. Those rows hang off no campaign, which is why
`campaign_id` is nullable and `entity_type` / `entity_id` exist.

### Approval guard

`can_approve` says whether the caller may decide; `campaigns.ApprovalBlock`
says whether this campaign may be approved. Both are enforced server-side in
`Decision` and `BulkDecision`.

Only the budget conditions block. Approving a campaign that has already spent
its budget ratifies an overspend, and approving one at 95% authorises a flight
that stalls within days — in both cases the budget has to move before a
sign-off means anything. The reach flags ("Reach outlier", "Under-delivering")
are judgement calls an approver is entitled to make, so they stay advisory.
**Reject**, **reopen** and **pause** are never blocked: they are how an
approver responds to an over-budget campaign. A scheduled campaign has spent
nothing, so its pre-flight sign-off — the case the queue exists for — is never
blocked.

A blocked single decision is `409` carrying the reason. A blocked bulk approve
is all-or-nothing: nothing is approved and the response names the offenders, so
the caller is never left guessing which of their selection went through. The
UI disables Approve and says why on all four surfaces that offer it, from the
same rule mirrored in `frontend/src/lib/metrics.ts`.

### Reports and scheduling

Cadence is one of four presets, not a cron expression:
`weekly_mon_9`, `weekday_830`, `monthly_1_9`, `on_flag`. The worker computes
each report's next occurrence in IST from its last run (or creation), and runs
it when that moment passes — `weekday_830` skips weekends, `monthly_1_9` wraps
across years. `on_flag` has no calendar schedule; the worker fires those
reports immediately after anomaly detection newly flags something, well inside
the "within 15 min" bound. An `on_flag` report's rows come from the flagged
set rather than its saved filter snapshot.

---

## Deployment

One container serves both halves. `npm run build` output is embedded into the
Go binary (`backend/web`), which serves the API on `/api`, the health probe on
`/health`, and the React app on everything else.

This is not just tidy packaging. The session cookie is `SameSite=Lax` and the
fetch client sends no cross-origin credentials, so hosting the frontend on a
separate origin would break sign-in. Making that work would mean
`SameSite=None` (weaker — that is what CSRF protection is), `AllowCredentials:
true`, an origin allowlist and `credentials: "include"`. Same origin needs none
of it, and because `client.ts` calls `/api` as a relative path, the frontend
needs no build-time configuration at all.

```
make build          # frontend -> backend/web/dist -> ./bin/server
./bin/server        # the whole app on one port
```

The same shape runs on any container host, so moving from a free tier to AWS
(App Runner, ECS/Fargate, Elastic Beanstalk) is a hosting change, not a
rewrite.

### What the image does on boot

1. **Applies migrations** from the embedded SQL, then serves. A deploy to a
   blank database comes up with a schema and no separate tooling in the image.
   It uses golang-migrate, the same tool as `make migrate`, so both share its
   `schema_migrations` bookkeeping and cannot disagree. It takes a Postgres
   advisory lock, so two instances starting together cannot both migrate.
   This suits single-instance deployment; a multi-instance rollout should move
   migration to a release-phase command so schema changes land before new code
   serves traffic.
2. **Logs whether a frontend is embedded** (`"frontend":true`). An image built
   without one serves a notice page on every route, which is far easier to
   diagnose from one log line than from a blank tab.

### Required configuration

| Env var | Production value |
|---|---|
| `DATABASE_URL` | Your managed Postgres URL, including `sslmode=require`. Either of Neon's strings works: migrations are routed around a pooled host automatically (see below), while the app's pool uses the URL as given |
| `SECURE_COOKIES` | **`true`** — without it the session cookie ships without `Secure` |
| `TZ` | `Asia/Kolkata` — every report cadence is computed in IST |
| `PORT` | Whatever the platform injects; defaults to `8090` |

### Creating the first account

**Never run `make seed` against a deployment.** It begins with
`TRUNCATE audit_events, creatives, report_runs, reports, campaigns, creators,
users` — it is a fixture loader, and it would erase production.

There is no user-management endpoint yet, so accounts are made with
`cmd/createuser`, which creates exactly one and touches nothing else. Run it
from your machine against the deployment's database:

```bash
DATABASE_URL='postgres://...' go run ./cmd/createuser \
  -email you@example.com -name 'Your Name' -role admin
```

It prints a generated password once. Re-running it for an existing email
re-passwords that account, which is also the only way back in if a password is
lost, since there is no reset flow. The binary is in the image as
`createuser` too, for hosts that give you a shell.

### Render + Neon (the free path)

[`render.yaml`](render.yaml) declares the web service. The database is
deliberately **not** on Render: its free Postgres expires 30 days after
creation and is then deleted. Neon's free plan is permanent, so
`DATABASE_URL` points there.

**On connection pooling.** golang-migrate serialises migrations with a
session-scoped `pg_advisory_lock`, and Neon's pooled endpoint is PgBouncer in
transaction-pooling mode, which does not keep a session pinned to one backend
connection — the lock and its unlock can land on different ones, leaving it
held by an idle connection while migrations hang. Pooling is still right for
ordinary traffic, so `migrateURL` rewrites only the migration connection onto
Neon's direct host and leaves the app's pool using `DATABASE_URL` as given.
That means either Neon string can be pasted in without it being a trap.

Known trade-off: Render's free tier sleeps after about 15 minutes idle and
takes roughly a minute to wake. While it sleeps the in-process worker is not
running, so scheduled reports do not fire on time and anomaly detection only
runs while someone is using the app. Fine for demos; it is the first thing to
fix when this becomes real.

### Before scaling past one instance

The anomaly detector and report scheduler run in-process on a 30-second
ticker in `cmd/server/main.go`. Two instances means every scheduled report
sends twice. Keep this at one instance until that work moves behind a lock or
into a separate scheduler.

---

## Keyboard shortcuts

| Key | Action |
|---|---|
| `⌘K` / `Ctrl+K` | Toggle the command palette |
| `/` | Focus the page's search, or open the palette if there isn't one |
| `←` / `→` | Previous / next campaign on a detail page, following the current filter+sort |
| `Backspace` | Go back |
| `↑` `↓` `Enter` `Esc` | Move, open, and dismiss within the palette |

All of these are ignored while you're typing in a field (except `⌘K`).

---

## Testing

```bash
make test
```

`make test` runs all three suites, and reseeds on the way through — `test-e2e`
depends on `seed`, so don't run it against anything you want to keep.

- **Go** — `backend/services/campaigns` and `backend/services/reports` cover
  every formula and the IST cadence maths with no database.
  `backend/services/campaigns/internal/service` runs against a real Postgres
  (there's no Docker here, so `DATABASE_URL` stands in for testcontainers);
  each test uses `svctest-`prefixed rows and its own user, and cleans both up
  after itself, so it's safe to run alongside seeded data.
  `backend/platform/arch` enforces the two dependency rules, and those tests
  have been verified to fail when violated rather than merely to pass.
- **Vitest** — `frontend/src/lib/*.test.ts` checks the TypeScript mirrors of the
  formulas and the Indian-unit formatting, against the same boundary cases as
  their Go counterparts.
- **Playwright** — three specs in `frontend/tests`, run serially against a
  freshly seeded database because they share one pending queue and would
  otherwise race each other's counts. `auth.spec.ts` covers a signed-out
  visitor getting the sign-in screen, a wrong password rejected without
  revealing whether the account exists, logout revoking the session so Back
  cannot restore it, and an analyst reading everything while shown no approval
  controls. `approve.spec.ts` approves a campaign from Overview and from the
  "Your call" ribbon, asserting the decision reaches the approvals queue, the
  rail badge and the overview counts. `tabswitch.spec.ts` is the regression
  test for focus-triggered refetches discarding what you had typed, on both the
  sign-in form and the campaigns filters. They drive
  the Chrome for Testing build in the local Playwright cache; override with
  `PLAYWRIGHT_CHROMIUM_PATH`, and point at a different origin with
  `E2E_BASE_URL`.

---

## Repository layout

T-shaped: one shared platform, five deep services, a composition root that
is the only thing aware of all of them. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

```
campaign-tracker-pro/
  backend/               everything Go; its own module
    Dockerfile           written to spec, never executed (no Docker here)
    cmd/
      server/            composition root: builds and mounts the modules
      seed/              the campaign, creator and account fixtures
    platform/            shared by every service, aware of none of them
      config/            env loading
      database/          pool, generated query set, pgtype conversions
      httpx/             JSON envelope, error→status, session middleware
      mail/              Mailer interface + log implementation
      units/             Indian unit formatting (lakh / crore)
      arch/              tests that enforce the dependency rules
    services/
      identity/          sessions, password auth, roles and permissions
      campaigns/         Campaign type, every metric formula, anomalies, CSV
      creators/          tier bucketing and tier-normalised performance
      analytics/         overview, benchmarks, regions, search (owns no tables)
      reports/           cadence maths, scheduler, run history

      ...and inside each one:
        *.go             public contract — the only thing siblings may import
        module/          factory: the only way to construct the service
        internal/        unreachable from any sibling, per the Go toolchain
          api/           parse request → call service → write DTO
          service/       use cases; owns the sentinel errors
          store/         sqlc rows → contract types
    db/
      migrations/        golang-migrate SQL
      queries/           sqlc sources
      gen/               sqlc output (do not edit)
  frontend/              everything React; its own package
    src/
      app/               router, layout shell, providers, Zustand stores
      api/               typed client + TanStack Query hooks
      components/        Rail, CommandBar, Palette, DataTable, charts, …
      features/          one folder per screen
      lib/               format, metrics, keyboard, URL params
      styles/            tokens.css, base.css
    tests/               Playwright
  docs/                  ARCHITECTURE.md
```

Two rules hold the structure up, both tested in `backend/platform/arch`:
**platform never imports a service**, and **a service uses a sibling's
public contract, never its internals** — the latter enforced by the Go
toolchain via `internal/`.
