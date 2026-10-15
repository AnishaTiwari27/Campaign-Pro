# API, roles and metrics

What the endpoints are, who may call them, and what the numbers on the
screen actually mean. The structural side — how the services are
separated and why — is in [ARCHITECTURE.md](ARCHITECTURE.md).

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
| GET | `/creators` · `/creators/:id` | The roster with tier-normalised performance; detail adds that person's campaigns. Shown in the UI as **Influencers** — the route, package and table keep the `creators` name |
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
