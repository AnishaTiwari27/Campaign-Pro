-- Run as audit_service against campaign_tracker:
--   psql -U audit_service -d campaign_tracker -f db/init/04_audit.sql
-- (requires 00_roles.sql to have run first)
--
-- audit-service is the only thing in the system that reads or writes this
-- schema — see 00_roles.sql's REVOKEs. Rows arrive from the campaign.audit
-- NATS subject (see platform/events.go, published by campaigns-service),
-- never written directly by a client — this table is a record of what
-- already happened, not something an API call constructs from scratch.

CREATE TABLE audit.audit_log (
    id          BIGSERIAL PRIMARY KEY,
    action      TEXT NOT NULL,          -- "campaign.created" | "campaign.budget_updated" | "campaign.approval_updated"
    actor_id    TEXT,                   -- X-User-Id at the time of the action; "system" for the demo ticker
    actor_email TEXT,
    actor_role  TEXT,
    campaign_id BIGINT,
    details     TEXT,                   -- short human-readable summary, e.g. "budget set to ₹200,000"
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
    -- Append-only by convention, not by grant — nothing in this service
    -- ever issues an UPDATE or DELETE against this table.
);

CREATE INDEX idx_audit_log_created_at ON audit.audit_log (created_at DESC);
CREATE INDEX idx_audit_log_campaign_id ON audit.audit_log (campaign_id);
