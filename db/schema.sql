-- Campaign Tracker Pro — PostgreSQL schema
-- Target: PostgreSQL 15+
-- Naming: snake_case, plural table names, singular FK columns.

CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- ---------------------------------------------------------------------------
-- Reference / lookup tables
-- Kept as real tables (not enums) so new regions, categories, and ad
-- formats can be added without a migration.
-- ---------------------------------------------------------------------------

CREATE TABLE categories (
    id   SMALLSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE          -- 'E-commerce', 'Fashion', 'Fintech', ...
);

CREATE TABLE regions (
    id   SMALLSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE          -- 'Mumbai', 'Delhi NCR', 'Pan-India', ...
);

CREATE TABLE ad_types (
    id               SMALLSERIAL PRIMARY KEY,
    name             TEXT NOT NULL UNIQUE,   -- 'Social Media', 'Performance', ...
    default_platform TEXT NOT NULL,          -- 'Instagram', 'Meta Ads', ...
    color_hex        TEXT NOT NULL           -- chart color, matches AD_TYPE_COLOR in the UI
);

-- ---------------------------------------------------------------------------
-- Core entities
-- ---------------------------------------------------------------------------

CREATE TABLE brands (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    category_id SMALLINT NOT NULL REFERENCES categories(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE campaigns (
    id            BIGSERIAL PRIMARY KEY,
    brand_id      INT NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    region_id     SMALLINT NOT NULL REFERENCES regions(id),
    ad_type_id    SMALLINT NOT NULL REFERENCES ad_types(id),
    platform      TEXT NOT NULL,             -- usually ad_types.default_platform, but
                                              -- stored per-row since a campaign can
                                              -- override it (e.g. cross-posted video)
    start_date    DATE NOT NULL,
    end_date      DATE NOT NULL,
    reach         BIGINT NOT NULL CHECK (reach >= 0),
    spend_paise   BIGINT NOT NULL CHECK (spend_paise >= 0), -- store money as integer paise
    budget_paise  BIGINT CHECK (budget_paise >= 0),  -- nullable — planned budget, for pacing; not every campaign has one
    approval_status TEXT NOT NULL DEFAULT 'approved'
                        CHECK (approval_status IN ('pending', 'approved', 'rejected')), -- review checkpoint, not a visibility gate
    source        TEXT NOT NULL DEFAULT 'manual',  -- 'google_ads' | 'meta' | 'linkedin' | 'youtube' | 'manual'
    external_id   TEXT,                       -- id on the source platform, for dedupe on ingest
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date)
);

-- Status ("Live" / "Completed") is derived from end_date vs. today, not stored —
-- avoids a background job to flip it. Computed in the query / application layer.

CREATE UNIQUE INDEX idx_campaigns_source_external
    ON campaigns (source, external_id)
    WHERE external_id IS NOT NULL;           -- ingestion dedupe key

CREATE INDEX idx_campaigns_brand      ON campaigns (brand_id);
CREATE INDEX idx_campaigns_region     ON campaigns (region_id);
CREATE INDEX idx_campaigns_ad_type    ON campaigns (ad_type_id);
CREATE INDEX idx_campaigns_start_date ON campaigns (start_date DESC);

-- Additive, not decomposed: a campaign's own reach/spend above are never
-- recalculated from its creatives' sum — see ARCHITECTURE.md's
-- Creative-level tracking section.
CREATE TABLE creatives (
    id            BIGSERIAL PRIMARY KEY,
    campaign_id   BIGINT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    headline      TEXT NOT NULL,
    creative_type TEXT NOT NULL CHECK (creative_type IN ('image', 'video', 'carousel', 'text')),
    reach         BIGINT NOT NULL DEFAULT 0 CHECK (reach >= 0),
    spend_paise   BIGINT NOT NULL DEFAULT 0 CHECK (spend_paise >= 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_creatives_campaign_id ON creatives (campaign_id);

-- Periodic full-table snapshot, one row per subject — backs benchmark
-- trending. Plain append-only inserts, no unique constraint (see
-- ARCHITECTURE.md's Benchmark trending section for why).
CREATE TABLE benchmark_snapshots (
    id               BIGSERIAL PRIMARY KEY,
    subject_name     TEXT NOT NULL,
    subject_type     TEXT NOT NULL,
    subject_category TEXT NOT NULL,
    campaign_count   INT NOT NULL,
    total_reach      BIGINT NOT NULL,
    snapshotted_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_benchmark_snapshots_subject ON benchmark_snapshots (subject_name, subject_type, snapshotted_at);

-- ---------------------------------------------------------------------------
-- Auth (Phase 3 — JWT-backed sessions)
-- ---------------------------------------------------------------------------

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,             -- bcrypt
    -- 'editor' can create campaigns and set budgets; 'approver' can
    -- approve/reject them; neither can do the other's job or manage
    -- users — real separation of duties (see ARCHITECTURE.md's RBAC
    -- section). 'admin' can do everything.
    role          TEXT NOT NULL DEFAULT 'viewer'
                      CHECK (role IN ('viewer', 'editor', 'approver', 'admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    token_hash TEXT PRIMARY KEY,             -- sha256 of the opaque refresh token
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Audit log — a separate service/schema in the real per-service-schema
-- layout (db/init/04_audit.sql, owned by audit_service); shown here in
-- this legacy single-schema reference file for completeness only.
-- actor_id/campaign_id are plain columns, not foreign keys, on purpose —
-- audit-service can't see auth's or campaigns' schemas at all. Rows
-- arrive from NATS events (campaign.audit), never a direct write path —
-- see ARCHITECTURE.md's Audit log section.
CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    action      TEXT NOT NULL,
    actor_id    TEXT,
    actor_email TEXT,
    actor_role  TEXT,
    campaign_id BIGINT,
    details     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_created_at ON audit_log (created_at DESC);
CREATE INDEX idx_audit_log_campaign_id ON audit_log (campaign_id);

-- ---------------------------------------------------------------------------
-- Seed data — the fixed vocabularies the UI's filter dropdowns expect.
-- Campaign/brand rows are seeded separately by the Go backend's dev seeder.
-- ---------------------------------------------------------------------------

INSERT INTO categories (name) VALUES
    ('E-commerce'), ('Fashion'), ('Food Delivery'), ('Technology'), ('Fintech'), ('FMCG');

INSERT INTO regions (name) VALUES
    ('Mumbai'), ('Delhi NCR'), ('Bengaluru'), ('South Zone'), ('West Zone'), ('Pan-India');

INSERT INTO ad_types (name, default_platform, color_hex) VALUES
    ('Social Media', 'Instagram',          '#C97A1A'),
    ('Influencer',   'YouTube Creators',   '#3B7A57'),
    ('Google Ads',   'Google Search',      '#B23A2A'),
    ('Display',      'Google Display',     '#8A6D3B'),
    ('Video',        'YouTube',            '#5A6ACF'),
    ('Performance',  'Meta Ads',           '#1F6F6F');
