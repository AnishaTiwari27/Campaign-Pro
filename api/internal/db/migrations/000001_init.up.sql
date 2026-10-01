CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE subject_type_t AS ENUM ('brand', 'person');
CREATE TYPE ad_type_t AS ENUM ('Social', 'Influencer', 'Google Ads', 'Display', 'Video', 'Performance');
CREATE TYPE campaign_status_t AS ENUM ('live', 'ended', 'scheduled', 'paused');
CREATE TYPE approval_status_t AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE curve_shape_t AS ENUM ('fast', 'steady', 'slow');
CREATE TYPE creative_kind_t AS ENUM ('video', 'banner', 'text');
CREATE TYPE audit_kind_t AS ENUM ('sys', 'alert', 'user');
CREATE TYPE report_cadence_t AS ENUM ('weekly_mon_9', 'weekday_830', 'monthly_1_9', 'on_flag');
CREATE TYPE report_result_t AS ENUM ('Delivered', 'Failed', 'Not sending');
CREATE TYPE user_role_t AS ENUM ('admin', 'viewer');

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    role        user_role_t NOT NULL DEFAULT 'viewer',
    can_approve BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Reach is stored in lakh (1L = 100,000 people); money in whole rupees.
CREATE TABLE campaigns (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    subject_type subject_type_t NOT NULL,
    role         TEXT,
    initials     TEXT NOT NULL,
    category     TEXT NOT NULL,
    region       TEXT NOT NULL,
    ad_type      ad_type_t NOT NULL,
    platform     TEXT NOT NULL,
    status       campaign_status_t NOT NULL DEFAULT 'live',
    days_running INT NOT NULL DEFAULT 0,
    reach        NUMERIC(8,2) NOT NULL DEFAULT 0 CHECK (reach >= 0),
    spend        BIGINT NOT NULL DEFAULT 0 CHECK (spend >= 0),
    budget       BIGINT NOT NULL DEFAULT 0 CHECK (budget >= 0),
    frequency    NUMERIC(4,2) NOT NULL DEFAULT 0,
    approval     approval_status_t NOT NULL DEFAULT 'pending',
    curve_shape  curve_shape_t NOT NULL DEFAULT 'steady',
    flag_reason  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_campaigns_category ON campaigns (category);
CREATE INDEX idx_campaigns_region ON campaigns (region);
CREATE INDEX idx_campaigns_status ON campaigns (status);
CREATE INDEX idx_campaigns_approval ON campaigns (approval);
CREATE INDEX idx_campaigns_ad_type ON campaigns (ad_type);

CREATE TABLE creatives (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id    TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    headline       TEXT NOT NULL,
    kind           creative_kind_t NOT NULL,
    duration_label TEXT NOT NULL,
    reach          NUMERIC(8,2) NOT NULL DEFAULT 0,
    ctr            NUMERIC(5,2) NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_creatives_campaign_id ON creatives (campaign_id);

-- Every approve, reject, reopen, pause, resume and note writes a row here.
CREATE TABLE audit_events (
    id          BIGSERIAL PRIMARY KEY,
    campaign_id TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    actor       TEXT NOT NULL,
    action      TEXT NOT NULL,
    kind        audit_kind_t NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_events_campaign_id ON audit_events (campaign_id, created_at DESC);

-- cadence is one of 4 fixed presets, not a cron expression — see
-- internal/worker's doc comment for why a plain preset enum plus real
-- calendar-aware "next occurrence" math is enough here.
CREATE TABLE reports (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    enabled       BOOLEAN NOT NULL DEFAULT true,
    cadence       report_cadence_t NOT NULL,
    recipients    TEXT[] NOT NULL DEFAULT '{}',
    scope_filters JSONB NOT NULL DEFAULT '{}',
    scope_label   TEXT NOT NULL DEFAULT '',
    columns       TEXT[] NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE report_runs (
    id         BIGSERIAL PRIMARY KEY,
    report_id  UUID NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    ran_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    result     report_result_t NOT NULL,
    row_count  INT NOT NULL DEFAULT 0
);
CREATE INDEX idx_report_runs_report_id ON report_runs (report_id, ran_at DESC);
