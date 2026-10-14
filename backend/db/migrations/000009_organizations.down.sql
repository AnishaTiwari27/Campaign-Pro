DROP INDEX IF EXISTS users_org_email_key;
CREATE UNIQUE INDEX users_email_key ON users (email);

DROP INDEX IF EXISTS reports_org_idx;
DROP INDEX IF EXISTS creators_org_idx;
DROP INDEX IF EXISTS campaigns_org_idx;
DROP INDEX IF EXISTS users_org_idx;

ALTER TABLE reports   DROP COLUMN org_id;
ALTER TABLE creators  DROP COLUMN org_id;
ALTER TABLE campaigns DROP COLUMN org_id;
ALTER TABLE users     DROP COLUMN org_id;

DROP TABLE organizations;
