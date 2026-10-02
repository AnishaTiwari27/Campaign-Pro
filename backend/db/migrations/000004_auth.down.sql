DROP TABLE IF EXISTS sessions;
ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE users DROP COLUMN IF EXISTS is_agency;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
-- enum values cannot be removed in Postgres; harmless to leave.
