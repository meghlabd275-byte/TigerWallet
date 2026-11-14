-- Initial database bootstrap for the infrastructure/docker stack.
-- The official postgres image runs every *.sql in /docker-entrypoint-initdb.d
-- on first start of an empty data directory. This script only creates databases
-- (schema is owned and migrated by each service on startup) and is idempotent,
-- so a re-run against an existing cluster is a no-op.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tigerwallet_billing') THEN
        CREATE DATABASE tigerwallet_billing;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tigerwallet_notifications') THEN
        CREATE DATABASE tigerwallet_notifications;
    END IF;
END
$$;

-- POSTGRES_DB itself holds the tenants control-plane tables, which the
-- multi-tenancy service creates on first boot.
