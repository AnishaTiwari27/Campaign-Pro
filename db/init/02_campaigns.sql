-- Run as campaigns_service against campaign_tracker:
--   psql -U campaigns_service -d campaign_tracker -f db/init/02_campaigns.sql
-- (requires 00_roles.sql to have run first)
--
-- No FK into the catalog schema on purpose — campaigns_service cannot see
-- catalog's tables (see 00_roles.sql's REVOKEs). subject/region/ad_type
-- are write-time snapshots, validated against catalog-service over HTTP at
-- create time and then stored as plain columns, the same "order line item
-- snapshots product name" trade-off used everywhere denormalization meets
-- service boundaries: renaming a brand later doesn't rewrite old campaign
-- rows, and reads never need a cross-service call.

CREATE TABLE campaigns.campaigns (
    id                BIGSERIAL PRIMARY KEY,
    subject_name      TEXT NOT NULL,
    subject_type      TEXT NOT NULL CHECK (subject_type IN ('brand', 'person')),
    subject_category  TEXT NOT NULL,
    region            TEXT NOT NULL,
    ad_type           TEXT NOT NULL,
    platform          TEXT NOT NULL,
    start_date        DATE NOT NULL,
    end_date          DATE NOT NULL,
    reach             BIGINT NOT NULL CHECK (reach >= 0),
    spend_paise       BIGINT NOT NULL CHECK (spend_paise >= 0),
    -- Nullable: most campaigns (everything seeded/demo-ticker-created so
    -- far) have no planned budget. Pacing (spend vs. planned, given how
    -- much of the flight has elapsed) is only computed when this is set —
    -- see services/campaigns/internal/models's PacingOf — and is derived
    -- at read time, never stored, same as status ("Live"/"Completed").
    budget_paise      BIGINT CHECK (budget_paise >= 0),
    -- Review checkpoint, not an access gate — a 'pending' campaign is
    -- still fully visible everywhere. Defaults to 'approved' so every
    -- existing row and every demo-ticker/seed-created row (both call
    -- Store.Create directly, bypassing the API) keep behaving exactly as
    -- before; only a real POST /campaigns through handleCreateCampaign
    -- starts a campaign as 'pending'. See services/campaigns/internal/api
    -- handleUpdateApprovalStatus for the admin-only transition.
    approval_status   TEXT NOT NULL DEFAULT 'approved'
                          CHECK (approval_status IN ('pending', 'approved', 'rejected')),
    source            TEXT NOT NULL DEFAULT 'manual',
    external_id       TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date)
);

CREATE UNIQUE INDEX idx_campaigns_source_external
    ON campaigns.campaigns (source, external_id)
    WHERE external_id IS NOT NULL;

CREATE INDEX idx_campaigns_subject_name ON campaigns.campaigns (subject_name);
CREATE INDEX idx_campaigns_subject_type ON campaigns.campaigns (subject_type);
CREATE INDEX idx_campaigns_region       ON campaigns.campaigns (region);
CREATE INDEX idx_campaigns_start_date   ON campaigns.campaigns (start_date DESC);

-- Periodic (see BENCHMARK_SNAPSHOT_INTERVAL, cmd/server/main.go) full-table
-- benchmark totals, one row per subject per snapshot — backs benchmark
-- trending (GET /benchmark's "trend" field). Plain append-only inserts,
-- deliberately no unique constraint / upsert-per-day: an upsert would
-- collapse every snapshot taken on the same calendar day into one row,
-- which would make this untestable within a single dev session. A real
-- deployment just runs the ticker daily and naturally gets one row per
-- subject per day.
CREATE TABLE campaigns.benchmark_snapshots (
    id               BIGSERIAL PRIMARY KEY,
    subject_name     TEXT NOT NULL,
    subject_type     TEXT NOT NULL,
    subject_category TEXT NOT NULL,
    campaign_count   INT NOT NULL,
    total_reach      BIGINT NOT NULL,
    snapshotted_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_benchmark_snapshots_subject ON campaigns.benchmark_snapshots (subject_name, subject_type, snapshotted_at);

-- A campaign can run multiple creatives (real ad platforms do) — this is
-- deliberately additive, not decomposed: a campaign's own reach/spend
-- above are never recalculated from its creatives' sum. Reconciling the
-- two would need either a DB trigger or application-level consistency
-- logic, real added risk (double-counting, drift) for a feature whose
-- actual ask was "track creatives exist," not "replace campaign-level
-- totals."
CREATE TABLE campaigns.creatives (
    id            BIGSERIAL PRIMARY KEY,
    campaign_id   BIGINT NOT NULL REFERENCES campaigns.campaigns(id) ON DELETE CASCADE,
    headline      TEXT NOT NULL,
    creative_type TEXT NOT NULL CHECK (creative_type IN ('image', 'video', 'carousel', 'text')),
    reach         BIGINT NOT NULL DEFAULT 0 CHECK (reach >= 0),
    spend_paise   BIGINT NOT NULL DEFAULT 0 CHECK (spend_paise >= 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_creatives_campaign_id ON campaigns.creatives (campaign_id);
