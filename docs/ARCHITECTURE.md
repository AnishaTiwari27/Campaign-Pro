# Architecture

A T-shaped layout: one shared horizontal platform, four deep vertical
services, and a composition root that is the only thing aware of all of
them.

```
                         ┌──────────────────────────────────┐
   browser ────────────▶ │  frontend/   React 18 · TS · Vite │
                         └────────────────┬─────────────────┘
                                          │  /api proxied to :8090
                                          ▼
   ╔══ backend/ ══════════════════════════════════════════════════════╗
   │  cmd/server      composition root — builds and mounts modules    │
   └───────┬──────────────┬───────────────┬───────────────┬───────────┘
           ▼              ▼               ▼               ▼
   ┌───────────────┬──────────────┬───────────────┬──────────────────┐
   │  campaigns    │  creators    │  analytics    │  reports         │
   │               │              │               │                  │
   │  contract     │  contract    │  (no tables)  │  contract        │
   │  ├ internal/  │  ├ internal/ │  ├ internal/  │  ├ internal/     │
   │  │  api       │  │  api      │  │  api       │  │  api          │
   │  │  service   │  │  service  │  │  service   │  │  service      │
   │  │  store     │  │  store    │  │            │  │  store        │
   │  └ module/    │  └ module/   │  └ module/    │  └ module/       │
   └───────────────┴──────────────┴───────────────┴──────────────────┘
           │              │               │               │
           └──────────────┴───────┬───────┴───────────────┘
                                  ▼
   ┌──────────────────────────────────────────────────────────────────┐
   │  platform/   config · database · httpx · mail · units · arch     │
   │              shared by every service, aware of none of them      │
   ╚══════════════════════════════╤═══════════════════════════════════╝
                                  ▼
                        ┌──────────────────┐
                        │   Postgres 16    │   db/ migrations · queries · gen
                        └──────────────────┘
```

## The two rules

**1. platform never depends on a service.** It is the horizontal bar — the
JSON envelope, error-to-status mapping, request middleware, the connection
pool, pgtype conversions, the mailer interface and Indian unit formatting.
If platform grew a dependency on `campaigns`, it would stop being shareable.

**2. A service uses a sibling's public contract, never its implementation.**
Each service's `store`, `service` and `api` packages sit under `internal/`,
which the Go toolchain refuses to let any other service — or `cmd/server` —
import. The only way in is the contract at the service root and the factory
in `module/`.

Both rules are tested in `backend/platform/arch/boundaries_test.go`, and those tests
have been verified to fail when violated rather than merely to pass.

## Why one process

The previous iteration of this repo ran eight services behind a gateway.
This one keeps the same module boundaries but a single binary, because the
boundaries are what was carrying the value — and here the compiler enforces
them for free, where a network hop only enforces them by convention. Each
`module/` package is a seam: if a service ever needs its own process, it
already has a factory, its own store, and no reach into anyone else's code.

## Layering inside a service

```
backend/services/campaigns/
  campaign.go  metrics.go  contract.go  dto.go  csv.go   ← public contract
  module/                                                 ← factory
  internal/
    api/        parse request → call service → write DTO
    service/    use cases; owns the sentinel errors
    store/      sqlc rows → contract types; no pgtype escapes
```

The contract holds what crosses a boundary: the `Campaign` type, every
metric formula, the published JSON shapes, and the CSV renderer. Analytics
embeds campaign DTOs in the overview; creators lists a creator's campaigns;
reports renders campaign rows into a CSV. None of them can see how any of
that is fetched.

## Dependency inversion in practice

Services do not import each other's concrete types. Each **consumer**
declares the narrow interface it needs, and the composition root supplies
something that satisfies it:

```go
// backend/services/creators/internal/service — creators says what it needs
type CampaignReader interface {
    All(ctx context.Context) ([]campaigns.Campaign, error)
    Enrich(ctx context.Context, all []campaigns.Campaign) map[string]campaigns.CampaignRow
    CreativesFor(ctx context.Context, campaignID string) ([]campaigns.Creative, error)
}
```

Analytics declares a wider one (it needs pending counts and the flagged
set), reports a different one again. The campaigns service satisfies all
three without knowing any of them exist. That is the I and D of SOLID doing
actual work: no consumer can reach past what it asked for, and campaigns
can be replaced by anything satisfying the interface.

## Service responsibilities

| Service | Owns | Depends on |
|---|---|---|
| **campaigns** | `campaigns`, `creatives`, `audit_events`, `users`. The `Campaign` type, every metric formula, anomaly detection, CSV. | platform |
| **creators** | `creators`. Tier bucketing and tier-normalised performance. | platform, campaigns (read) |
| **analytics** | *no tables.* Overview, benchmarks, regions, search — all derived. | platform, campaigns (read), creators (read) |
| **reports** | `reports`, `report_runs`. Cadence calendar maths and the scheduler. | platform, campaigns (read + detector) |

Analytics owning no tables is deliberate: every number it reports is derived
from what the other services publish, so there is no second copy of the
truth to drift.

## Composition root

`backend/cmd/server/main.go` is the only file that knows all four services exist.
It builds the platform, then hands each factory its dependencies in graph
order, then mounts whatever each module exposes:

```go
campaignsMod := campaigns.New(db, logger)
creatorsMod  := creators.New(db, campaignsMod.Service(), logger)
analyticsMod := analytics.New(db, campaignsMod.Service(), creatorsMod.Store(), logger)
reportsMod   := reports.New(db, campaignsMod.Service(), campaignsMod.Detector(), mailer, logger, tick)
```

The router takes modules as a `mountable` interface and never names a path
— each service owns its own URL space. Auth is a `UserLoader` function so
platform need not know which service owns users.

## Data

`backend/db/` holds migrations, the sqlc query sources, and the generated code —
one schema, shared. Each service's store runs only its own queries, which
is enforced by the package being internal rather than by separate
credentials.
