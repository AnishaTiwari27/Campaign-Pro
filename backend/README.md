> **Superseded.** This single-process MVP has been replaced by the
> microservices under [`../services/`](../services/) — see
> [`../docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md). Kept around for
> reference; not run in the current deployment.

# Campaign Tracker Pro — API

Go backend for the Campaign Tracker Pro dashboard. Stdlib-only (`net/http`'s
Go 1.22+ pattern router) — zero third-party dependencies, so `go build` works
with no network access.

## Run

```bash
go run ./cmd/server
```

Serves on `:8080` by default. Env vars:

| Var              | Default                      |
|-------------------|-------------------------------|
| `PORT`            | `8080`                        |
| `ALLOWED_ORIGIN`  | `http://localhost:5173`       |

## Try it

```bash
curl localhost:8080/api/v1/meta
curl "localhost:8080/api/v1/campaigns?category=Fintech&limit=5"
curl localhost:8080/api/v1/kpis
curl localhost:8080/api/v1/campaigns/export -o report.csv
```

## Layout

```
cmd/server/main.go        entrypoint, wiring, env config
internal/models/          JSON-facing types
internal/data/            seed dataset — ported 1:1 from the React mock's
                           seedCampaigns(), so responses match the old UI
internal/store/           Store interface + in-memory implementation
                           (swap for a Postgres-backed one once db/schema.sql
                           is applied — the interface doesn't change)
internal/api/             routes, handlers, CORS + logging middleware
```

Full route list and JSON shapes: [`../docs/API_CONTRACT.md`](../docs/API_CONTRACT.md).
DB schema this will graduate to: [`../db/schema.sql`](../db/schema.sql), [`../docs/ERD.md`](../docs/ERD.md).

## Status

MVP (Phase 3, first cut): in-memory store, no auth, matches the roadmap's
"in-memory now → PostgreSQL via sqlc" stack decision. Not yet wired:

- [ ] Postgres-backed `store.Store` implementation + `sqlc` queries
- [ ] JWT auth (`/auth/login`, `/auth/refresh`)
- [ ] Real ad-platform ingestion (Google Ads / Meta / LinkedIn / YouTube)
- [ ] Redis TTL cache in front of the aggregate endpoints
