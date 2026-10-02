# Campaign Tracker Pro

An internal console for monitoring ad campaigns in the Indian market. It shows
what's live, what's waiting on a decision, what anomaly detection has flagged,
how campaigns compare against category benchmarks, and it emails scheduled
CSV reports.

Go + chi + Postgres on the back, React + TypeScript + Vite on the front.

---

## Running it

The verified path is native Postgres — the machine this was built on has no
Docker, so `docker-compose.yml` and `api/Dockerfile` are written to spec but
have never been executed. Use them as a starting point, not a tested artifact.

**Prerequisites:** Go 1.25+, Node 20+, PostgreSQL 16, and
[`golang-migrate`](https://github.com/golang-migrate/migrate) (`brew install golang-migrate`).
`sqlc` is only needed if you change the SQL (`brew install sqlc`).

```bash
make db-create   # creates the campaign_tracker_pro database + role
make migrate     # applies api/internal/db/migrations
make seed        # 24 campaigns, 3 reports, the admin user
make dev         # api on :8090, web on :5173
```

Then open the web dev server (Vite prints the port — 5173 unless taken).

| Target | What it does |
|---|---|
| `make dev` | API and web together |
| `make seed` | Truncate and reseed demo data |
| `make test` | Go tests, Vitest, Playwright |
| `make lint` | `go vet`, `gofmt` check, ESLint |
| `make generate` | Regenerate sqlc code from `internal/db/queries` |

### Configuration

| Env var | Default |
|---|---|
| `DATABASE_URL` | `postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable` |
| `PORT` | `8090` |
| `TZ` | `Asia/Kolkata` |
| `MAIL_MODE` | `log` (the only implementation; writes the send to the log) |
| `ADMIN_EMAIL` | `anishatiwari695@gmail.com` — the seeded user auth loads onto every request |

---

## Architecture

```
                      ┌──────────────────────────────┐
  browser  ─────────▶ │  web (Vite dev server :5173) │
                      │  React 18 · TS · Router v6   │
                      │  TanStack Query · Zustand    │
                      └───────────────┬──────────────┘
                                      │ /api proxied to :8090
                                      ▼
                      ┌──────────────────────────────┐
                      │  api (:8090)                 │
                      │                              │
                      │  http/     handlers, router, │
                      │            auth, CORS, DTOs  │
                      │     ▼                        │
                      │  service/  use cases:        │
                      │            campaigns,        │
                      │            overview, reports,│
                      │            anomalies, search │
                      │     ▼           ▼            │
                      │  domain/     store/          │
                      │  formulas,   sqlc queries    │
                      │  no deps     → domain types  │
                      │                 │            │
                      │  worker/  ticker: anomaly    │
                      │           detection + report │
                      │           scheduling         │
                      └─────────────────┬────────────┘
                                        ▼
                               ┌──────────────────┐
                               │  Postgres 16     │
                               └──────────────────┘
```

**The layering rule:** `domain/` holds every metric formula and imports
nothing but the standard library, so the formulas are unit-testable in
isolation. `store/` maps sqlc's generated structs onto domain types at the
boundary, so no `pgtype` escapes it. `service/` composes the two. `http/` only
parses requests and writes DTOs.

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
| GET | `/regions` · `/regions/:id` | Rollups; detail adds campaigns by reach, category breakdown, pending count, budget share |
| GET/POST | `/reports` | List (with each report's last run) · create from a filter snapshot |
| GET/PATCH | `/reports/:id` | Read · edit name, enabled, cadence, recipients, columns |
| GET | `/reports/:id/runs` | Run history |
| POST | `/reports/:id/run` | Run now — emails real recipients, writes a run row |
| POST | `/reports/:id/test` | Send a copy to the caller only; no run row |
| GET | `/search?q=` | Command palette: sections, campaigns, regions, actions |
| GET | `/me` · `/health` | Current user · data-source health |

**Auth** is a stub: middleware loads the seeded admin onto every request's
context. Handlers read the caller via `UserFromContext` and `can_approve` is
enforced server-side, so swapping in real SSO later doesn't touch handlers.

---

## Metrics

Units: **reach is stored in lakh** (1L = 100,000 people); **money is whole
rupees**. Defined once in `services/campaigns/`, mirrored in
`web/src/lib/`.

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
  then runs the real detector, so the ~6 flagged campaigns are whatever the
  formulas actually produce from the seeded numbers.

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

- **Go** — `backend/services/campaigns` covers every formula and the IST cadence maths
  with no database. `internal/service` runs against a real Postgres (there's
  no Docker here, so `DATABASE_URL` stands in for testcontainers); each test
  uses `svctest-`prefixed rows and cleans up after itself, so it's safe to run
  alongside seeded data.
- **Vitest** — `frontend/src/lib/*.test.ts` checks the TypeScript mirrors of the
  formulas and the Indian-unit formatting.
- **Playwright** — one smoke test (`frontend/tests/approve.spec.ts`) approves a
  campaign starting from Overview and asserts the decision reaches the
  approvals queue, the rail badge and the overview counts. It drives the
  Chrome for Testing build in the local Playwright cache; override with
  `PLAYWRIGHT_CHROMIUM_PATH`, and point at a different origin with
  `E2E_BASE_URL`.

---

## Repository layout

T-shaped: one shared platform, four deep services, a composition root that
is the only thing aware of all of them. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

```
campaign-tracker-pro/
  backend/               everything Go; its own module
    cmd/
      server/            composition root: builds and mounts the modules
      seed/              the campaign + creator fixture
    platform/            shared by every service, aware of none of them
      config/            env loading
      database/          pool, generated query set, pgtype conversions
      httpx/             JSON envelope, error→status, middleware
      mail/              Mailer interface + log implementation
      units/             Indian unit formatting (lakh / crore)
      arch/              tests that enforce the dependency rules
    services/
      campaigns/         Campaign type, every metric formula, anomalies, CSV
      creators/          tier bucketing and tier-normalised performance
      analytics/         overview, benchmarks, regions, search (owns no tables)
      reports/           cadence maths, scheduler, run history
        <service>/
          *.go           public contract — the only thing siblings may import
          module/        factory: the only way to construct the service
          internal/
            api/         parse request → call service → write DTO
            service/     use cases
            store/       sqlc rows → contract types
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
