package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Schema is the authoritative DDL for the multi-tenancy control plane. Every
// statement is additive and idempotent so a restart is always safe.
const Schema = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS plans (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tier        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    price_usd   NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO plans (tier, name, price_usd) VALUES
    ('free',       'Free',         0),
    ('starter',    'Starter',      99),
    ('pro',        'Professional', 499),
    ('enterprise', 'Enterprise',   1999)
ON CONFLICT (tier) DO NOTHING;

CREATE TABLE IF NOT EXISTS tenants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    email           TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'trial',
    plan_id         UUID NOT NULL REFERENCES plans(id),
    custom_domain   TEXT,
    logo_url        TEXT,
    primary_color   TEXT,
    secondary_color TEXT,
    timezone        TEXT NOT NULL DEFAULT 'UTC',
    language        TEXT NOT NULL DEFAULT 'en',
    features        JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    trial_ends_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

CREATE TABLE IF NOT EXISTS quotas (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    resource     TEXT NOT NULL,
    "limit"      BIGINT NOT NULL DEFAULT 0,
    used         BIGINT NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period_end   TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    reset_at     TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, resource)
);

CREATE TABLE IF NOT EXISTS tenant_configs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    wallet_settings       JSONB NOT NULL DEFAULT '{}'::jsonb,
    bot_settings          JSONB NOT NULL DEFAULT '{}'::jsonb,
    token_settings        JSONB NOT NULL DEFAULT '{}'::jsonb,
    security_settings     JSONB NOT NULL DEFAULT '{}'::jsonb,
    notification_settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    role        TEXT NOT NULL DEFAULT 'member',
    permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    status      TEXT NOT NULL DEFAULT 'active',
    invited_at  TIMESTAMPTZ,
    joined_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id)
);

CREATE TABLE IF NOT EXISTS tenant_invitations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'member',
    token      TEXT NOT NULL UNIQUE,
    status     TEXT NOT NULL DEFAULT 'pending',
    invited_by UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    user_id     UUID NOT NULL,
    action      TEXT NOT NULL,
    resource    TEXT NOT NULL,
    resource_id UUID,
    details     JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address  TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_tenant ON tenant_audit_logs(tenant_id, created_at DESC);
`

// Migrate applies the schema. It fails loudly rather than starting a service
// that would answer every request with a SQL error.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, Schema)
	return err
}
