# Architecture

A T-shaped layout: one shared horizontal platform, five deep vertical
services, and a composition root that is the only thing aware of all of
them.

```
                         ┌──────────────────────────────────┐
   browser ────────────▶ │  frontend/   React 18 · TS · Vite │
                         └────────────────┬─────────────────┘
                                          │  /api proxied to :8090
                                          ▼
   ╔══ backend/ ═══════════════════════════════════════════════════════════╗
   │  cmd/server      composition root — builds and mounts modules         │
   └──────┬─────────────┬─────────────┬─────────────┬─────────────┬────────┘
          ▼             ▼             ▼             ▼             ▼
   ┌────────────┬────────────┬────────────┬────────────┬───────────────────┐
   │  identity  │  campaigns │  creators  │  analytics │  reports          │
   │            │            │            │            │                   │
   │  contract  │  contract  │  contract  │ (no tables)│  contract         │
   │  ├internal/│  ├internal/│  ├internal/│  ├internal/│  ├ internal/      │
   │  │  api    │  │  api    │  │  api    │  │  api    │  │  api           │
   │  │  service│  │  service│  │  service│  │  service│  │  service       │
   │  │  store  │  │  store  │  │  store  │  │         │  │  store         │
   │  └module/  │  └module/  │  └module/  │  └module/  │  └ module/        │
   └────────────┴────────────┴────────────┴────────────┴───────────────────┘
          │             │             │             │             │
          └─────────────┴──────┬──────┴─────────────┴─────────────┘
                               ▼
   ┌───────────────────────────────────────────────────────────────────────┐
   │  platform/   config · database · httpx · mail · units · arch          │
   │              shared by every service, aware of none of them           │
   ╚═══════════════════════════════╤═══════════════════════════════════════╝
                                   ▼
                        ┌──────────────────┐
                        │   Postgres 16    │   db/ migrations · queries · gen
                        └──────────────────┘
```

## The two rules

**1. platform never depends on a service.** It is the horizontal bar — the
JSON envelope, error-to-status mapping, request and session middleware, the
connection pool, pgtype conversions, the mailer interface and Indian unit
formatting. If platform grew a dependency on `campaigns`, it would stop being
shareable. The session guard is the interesting case: it has to know who the
caller is without knowing which service owns users, so it takes a
`SessionAuthenticator` and stores the result as an opaque principal that
handlers type-assert back themselves.

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
| **identity** | `sessions`, and the credential columns of `users` (`password_hash`, `last_login_at`). Who the caller is, and the `Role` / `CanApprove` / `CanManageUsers` / `IsClient` permission model. | platform, campaigns (audit, write-only via a bridge) |
| **campaigns** | `campaigns`, `creatives`, `audit_events`, and the `users` row lifecycle. The `Campaign` type, every metric formula, anomaly detection, the approval guard, CSV. | platform |
| **creators** | `creators`. Tier bucketing and tier-normalised performance. | platform, campaigns (read) |
| **analytics** | *no tables.* Overview, benchmarks, regions, search — all derived. | platform, campaigns (read), creators (read) |
| **reports** | `reports`, `report_runs`. Cadence calendar maths and the scheduler. | platform, campaigns (read + detector) |

Analytics owning no tables is deliberate: every number it reports is derived
from what the other services publish, so there is no second copy of the
truth to drift.

`users` is the one table two services touch, and they touch disjoint parts of
it: campaigns creates and reads the row, identity owns the credentials on it.
Identity needs to record sign-in and sign-out in `audit_events`, which
campaigns owns, so it declares a one-method `Auditor` capability and the
composition root satisfies it with an `auditBridge` — identity never imports
the campaigns store, and the dependency stays one-directional.

## Composition root

`backend/cmd/server/main.go` is the only file that knows all five services exist.
It builds the platform, then hands each factory its dependencies in graph
order, then mounts whatever each module exposes:

```go
campaignsMod := campaigns.New(db, logger)
identityMod  := identity.New(db, auditBridge{campaignsMod}, cfg.SecureCookies, logger)
creatorsMod  := creators.New(db, campaignsMod.Service(), logger)
analyticsMod := analytics.New(db, campaignsMod.Service(), creatorsMod.Store(), logger)
reportsMod   := reports.New(db, campaignsMod.Service(), campaignsMod.Detector(), mailer, logger, tick)
```

The router takes modules as a `mountable` interface and never names a path —
each service owns its own URL space. Identity additionally satisfies
`publicMountable`, which contributes the one route that must work without a
session (`POST /api/auth/login`) and supplies the `httpx.SessionAuthenticator`
that guards everything else. Platform therefore knows only how to turn an
opaque token into a principal, never which service owns users.

## Data

`backend/db/` holds migrations, the sqlc query sources, and the generated code —
one schema, shared. Each service's store runs only its own queries, which
is enforced by the package being internal rather than by separate
credentials.
