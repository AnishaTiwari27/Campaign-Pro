# Campaign Tracker Pro

An internal console for monitoring ad campaigns in the Indian market. It shows
what's live, what's waiting on a decision, what anomaly detection has flagged,
how campaigns compare against category benchmarks, and it emails scheduled CSV
reports.

Go + chi + Postgres on the back, React + TypeScript + Vite on the front.

## Running it

**Prerequisites:** Go 1.25+, Node 20+, PostgreSQL 16, and
[`golang-migrate`](https://github.com/golang-migrate/migrate). `sqlc` is only
needed if you change the SQL.

```bash
make db-create   # creates the database + role
make migrate     # applies backend/db/migrations
make seed        # 35 campaigns, 7 creators, 3 reports, 3 users
make dev         # api on :8090, web on :5173
```

`make seed` writes three accounts sharing one password,
`demo-password-change-me` — a deliberately loud placeholder that must never
reach a real deployment.

| Email | Role | Can approve? |
|---|---|---|
| `$ADMIN_EMAIL` | admin | yes |
| `approver@campaigntracker.test` | approver | yes |
| `analyst@campaigntracker.test` | analyst | no — controls hidden, API returns `403` |

The verified path is native Postgres. `docker-compose.yml` and
`backend/Dockerfile` are written to spec but have never been executed on this
machine; treat them as a starting point rather than a tested artifact.

## Make targets

| Target | What it does |
|---|---|
| `make dev` | API and web together |
| `make migrate` · `make migrate-down` | Apply migrations · roll back one |
| `make seed` | Truncate and reseed demo data |
| `make test` | Go, Vitest and Playwright — **reseeds on the way through** |
| `make lint` · `make fmt` | `go vet` + `gofmt` + ESLint · rewrite with `gofmt` |
| `make generate` | Regenerate sqlc code from `backend/db/queries` |
| `make verify-ingest SOURCE=…` | One real read-only call to an ad platform |
| `make logos` | Re-fetch brand logos (committed, so rarely needed) |

## Configuration

Everything is environment variables with working defaults for local use. The
only one with no sensible default is `DATABASE_URL` in a deployment.

| Variable | Default | Notes |
|---|---|---|
| `DATABASE_URL` | local dev DSN | Fatal on a managed host if unset |
| `TEST_DATABASE_URL` | `…_test` DSN | Where the DB-backed Go tests run |
| `PORT` | `8090` | |
| `ADMIN_EMAIL` | maintainer address | The first admin the seeder writes |
| `ALLOW_SIGNUP` | `true` | Set `false` once the right accounts exist |
| `SECURE_COOKIES` | `false` | Must be `true` behind HTTPS |
| `RATE_API_PER_MINUTE` · `RATE_AUTH_PER_MINUTE` | `600` · `120` | Per IP |
| `MAIL_MODE` + `SMTP_*` | `log` | `smtp` to actually send digests |
| `OIDC_*` | — | Single sign-on; absent means the button isn't shown |
| `ANTHROPIC_API_KEY` | — | Signal explanations; absent means they're assembled from the figures |
| `GOOGLE_ADS_*` · `META_*` | — | Ad platform ingestion |

Optional integrations are inert without their keys rather than broken: SSO
hides its button, explanations fall back to assembled text, ingestion lists
the platform as not-configured.

## Testing

```bash
make test-go     # Go, including DB-backed service tests
make test-web    # Vitest
make test-e2e    # Playwright — seeds itself first
```

The DB-backed Go tests use their own database (`campaign_tracker_pro_test`,
created by `make db-create`) so they can run alongside the e2e suite, which
truncates and reseeds the dev database on startup. Override with
`TEST_DATABASE_URL`.

## Repository layout

T-shaped: one shared platform, deep services, and a composition root that is
the only thing aware of all of them.

```
backend/
  cmd/server/      composition root — builds and mounts the modules
  platform/        shared by every service, aware of none of them
  platform/arch/   tests that enforce the dependency rules
  services/        identity, campaigns, creators, analytics, reports,
                   endorsements, ingest, insights, radar
  db/              migrations, queries, generated sqlc code
frontend/
  src/api/         typed client and TanStack Query hooks
  src/features/    one folder per page
  src/components/  shared UI
  tests/           Playwright specs
```

Each service exposes a public contract at its root, keeps everything else
under `internal/`, and is built through its `module/` factory. The two
dependency rules are enforced by tests, not convention.

## Documentation

| Document | What's in it |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Service boundaries, the two rules, layering, why one process |
| [docs/api.md](docs/api.md) | Endpoints, roles and permissions, what each metric means, the audit trail |
| [docs/deployment.md](docs/deployment.md) | Boot sequence, required config, the Render + Neon path |
| [docs/decisions.md](docs/decisions.md) | Choices that look arbitrary and aren't |
