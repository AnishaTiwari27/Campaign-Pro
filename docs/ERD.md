# Entity Relationship Diagram

Source of truth: [`db/schema.sql`](../db/schema.sql). Rendered with Mermaid.
The actual per-service migrations that create these tables (each qualified
to its own schema, per [`ARCHITECTURE.md`](ARCHITECTURE.md)) live in
`db/init/`: `01_catalog.sql`, `02_campaigns.sql`, `03_auth.sql` (`USERS` /
`REFRESH_TOKENS`), and `04_audit.sql` (`AUDIT_LOG`, its own schema, no FK
into any other — see `ARCHITECTURE.md`'s Audit log section for why it's
fed by NATS events, not a direct relationship).

```mermaid
erDiagram
    CATEGORIES ||--o{ BRANDS : classifies
    BRANDS ||--o{ CAMPAIGNS : runs
    REGIONS ||--o{ CAMPAIGNS : targets
    AD_TYPES ||--o{ CAMPAIGNS : formats
    USERS ||--o{ REFRESH_TOKENS : owns
    CAMPAIGNS ||--o{ CREATIVES : contains

    CATEGORIES {
        smallint id PK
        text name
    }
    REGIONS {
        smallint id PK
        text name
    }
    AD_TYPES {
        smallint id PK
        text name
        text default_platform
        text color_hex
    }
    BRANDS {
        int id PK
        text name
        smallint category_id FK
        timestamptz created_at
    }
    CAMPAIGNS {
        bigint id PK
        int brand_id FK
        smallint region_id FK
        smallint ad_type_id FK
        text platform
        date start_date
        date end_date
        bigint reach
        bigint spend_paise
        bigint budget_paise "nullable"
        text approval_status "pending, approved, or rejected"
        text source
        text external_id
        timestamptz created_at
        timestamptz updated_at
    }
    CREATIVES {
        bigint id PK
        bigint campaign_id FK
        text headline
        text creative_type "image, video, carousel, or text"
        bigint reach
        bigint spend_paise
        timestamptz created_at
    }
    BENCHMARK_SNAPSHOTS {
        bigint id PK
        text subject_name
        text subject_type
        text subject_category
        int campaign_count
        bigint total_reach
        timestamptz snapshotted_at
    }
    USERS {
        uuid id PK
        text email
        text password_hash
        text role "viewer, editor, approver, or admin"
    }
    REFRESH_TOKENS {
        text token_hash PK
        uuid user_id FK
        timestamptz expires_at
    }
    AUDIT_LOG {
        bigint id PK
        text action
        text actor_id "no FK — a different service's schema"
        text actor_email
        text actor_role
        bigint campaign_id "soft reference, no FK"
        text details
        timestamptz created_at
    }
```

## Notes

- **`status` ("Live" / "Completed") is not a column.** It's derived from `end_date`
  vs. the current date at read time — one less thing to keep in sync.
- **Money is stored as `spend_paise` (integer)**, never a float, to avoid rounding
  drift. The API divides by 100 before it reaches JSON.
- **`campaigns.platform` is denormalized** from `ad_types.default_platform` — most
  rows just inherit it, but a campaign can override it (e.g. a video ad
  cross-posted to a platform outside its format's default).
- **`(source, external_id)` is a partial unique index**, not a plain unique
  constraint — it only applies to ingested rows, so manually entered campaigns
  (`external_id IS NULL`) aren't forced to be unique against each other.
- **`approval_status` is a real stored column**, unlike `status` above — it's
  a decision a person makes (see `ARCHITECTURE.md`'s Campaign approval
  section), not derivable from the row's own dates. Defaults to `'approved'`
  so existing/seeded/demo-ticker rows need no backfill; only a real
  `POST /campaigns` starts a row `'pending'`. Not a visibility gate — a
  `'pending'` campaign is still fully visible everywhere.
- **`CREATIVES` is additive, not decomposed** — a campaign's own
  `reach`/`spend` are never recalculated from its creatives' sum (see
  `ARCHITECTURE.md`'s Creative-level tracking section).
- **`BENCHMARK_SNAPSHOTS` has no foreign key into `CAMPAIGNS`** — it's
  keyed by `(subject_name, subject_type)`, a subject, not one campaign,
  and it's a plain append-only fact table (no unique constraint — see
  `ARCHITECTURE.md`'s Benchmark trending section for why).
- **`AUDIT_LOG` lives in its own schema (`audit`, `db/init/04_audit.sql`),
  owned by a separate service (`audit-service`)** — `actor_id`/
  `campaign_id` are plain columns, not foreign keys, because
  `audit_service`'s Postgres role can't see `auth`'s or `campaigns`'
  schemas at all (same per-service isolation every other schema already
  has). Rows arrive from NATS events, not a direct write path.
