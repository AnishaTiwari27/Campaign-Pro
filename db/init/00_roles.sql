-- Run once as the Postgres superuser against the campaign_tracker database:
--   psql -d campaign_tracker -f db/init/00_roles.sql
--
-- Four service roles, each scoped to exactly one schema. Ownership + explicit
-- REVOKE makes this an enforced boundary, not just a documented convention —
-- see docs/ARCHITECTURE.md and the isolation check in the services' README.

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'catalog_service') THEN
    CREATE ROLE catalog_service LOGIN PASSWORD 'catalog_dev_pw';
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'campaigns_service') THEN
    CREATE ROLE campaigns_service LOGIN PASSWORD 'campaigns_dev_pw';
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'auth_service') THEN
    CREATE ROLE auth_service LOGIN PASSWORD 'auth_dev_pw';
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'audit_service') THEN
    CREATE ROLE audit_service LOGIN PASSWORD 'audit_dev_pw';
  END IF;
END $$;

GRANT CONNECT ON DATABASE campaign_tracker TO catalog_service, campaigns_service, auth_service, audit_service;

CREATE SCHEMA IF NOT EXISTS catalog   AUTHORIZATION catalog_service;
CREATE SCHEMA IF NOT EXISTS campaigns AUTHORIZATION campaigns_service;
CREATE SCHEMA IF NOT EXISTS auth      AUTHORIZATION auth_service;
CREATE SCHEMA IF NOT EXISTS audit     AUTHORIZATION audit_service;

-- Nobody gets anything by default...
REVOKE ALL ON SCHEMA catalog   FROM PUBLIC;
REVOKE ALL ON SCHEMA campaigns FROM PUBLIC;
REVOKE ALL ON SCHEMA auth      FROM PUBLIC;
REVOKE ALL ON SCHEMA audit     FROM PUBLIC;

-- ...and each service role only gets its own schema. These REVOKEs are
-- belt-and-suspenders (ownership alone doesn't imply cross-schema access) —
-- they make the isolation explicit and trivially verifiable. auth is the
-- most sensitive schema in the database (password hashes, refresh tokens),
-- so it gets the same treatment even though only one service will ever ask.
-- audit-service reads NATS events, not other schemas directly, so it gets
-- exactly as little cross-schema access as everyone else.
REVOKE ALL ON SCHEMA catalog   FROM campaigns_service, auth_service, audit_service;
REVOKE ALL ON SCHEMA campaigns FROM catalog_service,   auth_service, audit_service;
REVOKE ALL ON SCHEMA auth      FROM catalog_service,   campaigns_service, audit_service;
REVOKE ALL ON SCHEMA audit     FROM catalog_service,   campaigns_service, auth_service;

GRANT USAGE, CREATE ON SCHEMA catalog   TO catalog_service;
GRANT USAGE, CREATE ON SCHEMA campaigns TO campaigns_service;
GRANT USAGE, CREATE ON SCHEMA auth      TO auth_service;
GRANT USAGE, CREATE ON SCHEMA audit     TO audit_service;

-- So tables each role creates later are usable by that same role without a
-- manual per-table GRANT.
ALTER DEFAULT PRIVILEGES FOR ROLE catalog_service IN SCHEMA catalog
  GRANT ALL ON TABLES TO catalog_service;
ALTER DEFAULT PRIVILEGES FOR ROLE campaigns_service IN SCHEMA campaigns
  GRANT ALL ON TABLES TO campaigns_service;
ALTER DEFAULT PRIVILEGES FOR ROLE auth_service IN SCHEMA auth
  GRANT ALL ON TABLES TO auth_service;
ALTER DEFAULT PRIVILEGES FOR ROLE audit_service IN SCHEMA audit
  GRANT ALL ON TABLES TO audit_service;
