-- Real authentication. Until now a stub loaded the seeded admin onto every
-- request, which meant anyone reaching the API was an approver.

-- The four roles the product actually has. 'viewer' kept so existing rows
-- stay valid; it is superseded by 'analyst'.
ALTER TYPE user_role_t ADD VALUE IF NOT EXISTS 'approver';
ALTER TYPE user_role_t ADD VALUE IF NOT EXISTS 'analyst';
ALTER TYPE user_role_t ADD VALUE IF NOT EXISTS 'client';

ALTER TABLE users ADD COLUMN password_hash TEXT;
ALTER TABLE users ADD COLUMN is_agency BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN last_login_at TIMESTAMPTZ;

-- Sessions live server-side so logout genuinely revokes access. A JWT
-- would stay valid until expiry no matter what the user clicked.
--
-- token_hash, never the token: a dump of this table must not hand anyone
-- a working session.
CREATE TABLE sessions (
    token_hash BYTEA PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    user_agent TEXT,
    ip         TEXT
);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);
