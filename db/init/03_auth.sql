-- Run as auth_service against campaign_tracker:
--   psql -U auth_service -d campaign_tracker -f db/init/03_auth.sql
-- (requires 00_roles.sql to have run first)
--
-- auth-service is the only thing in the system that ever reads or writes
-- this schema (see 00_roles.sql's REVOKEs) — password hashes and refresh
-- tokens never leave it. Every other service learns "who is this request
-- from" via the trusted X-User-* headers the gateway sets after verifying
-- the JWT auth-service issued; see docs/ARCHITECTURE.md.

CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

CREATE TABLE auth.users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,                   -- bcrypt
    -- 'editor' can create campaigns and set budgets; 'approver' can
    -- approve/reject them; neither can do the other's job or manage users
    -- — real separation of duties now that the approval workflow exists
    -- (see docs/ARCHITECTURE.md's RBAC section). 'admin' can do everything.
    role          TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('viewer', 'editor', 'approver', 'admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- token_hash is sha256(opaque refresh token) — the raw token is returned to
-- the client once and never stored, so a leaked row of this table alone
-- can't be replayed. /auth/refresh rotates: the row matching the presented
-- token is deleted and a new one inserted in the same request, so a stolen
-- refresh token stops working the moment its legitimate owner uses it again.
CREATE TABLE auth.refresh_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON auth.refresh_tokens (user_id);

-- ---------------------------------------------------------------------------
-- Dev/demo seed users — local credentials only, never use these values
-- outside a laptop/dev environment. Passwords: "ChangeMe123!admin",
-- "ChangeMe123!viewer", "ChangeMe123!editor", "ChangeMe123!approver" (see
-- docs/ARCHITECTURE.md's Auth section).
-- ---------------------------------------------------------------------------

INSERT INTO auth.users (email, password_hash, role) VALUES
    ('admin@campaigntracker.dev',    '$2a$10$1EQxNUa/SxVad/HwEOCpcu0766raNTeN5CsNid9S5T0mwEiW8qWr.', 'admin'),
    ('viewer@campaigntracker.dev',   '$2a$10$Fxk672SR6EtF9LaBFNYhfudeiB/jasz6NTxE9rUsQRurRYp3azV4K', 'viewer'),
    ('editor@campaigntracker.dev',   '$2a$10$SOTh1ZM.R3WU4kD5S1.ahO9CjRvLqWjPDgWXLstMcTN1GCmuXaQdS', 'editor'),
    ('approver@campaigntracker.dev', '$2a$10$JjVhLC5P/CYbBKwpFZzvtevSNGfuO8nAsbeH5vm.bgT0bmBlUxff6', 'approver');
