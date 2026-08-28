-- TigerWallet UserWallet address-scale schema.
--
-- PURPOSE: the wallet-address/balance index supporting "billions of UserWallet
-- addresses" (Phase 21). Existing platform tables in main_schema.sql are
-- untouched; this file is ADDITIVE and runs after main_schema.sql.
--
-- DESIGN (see docs/USER_WALLET_SHARDING.md):
--  * 256 hash buckets partitioned by the first byte of the address chain
--    (address bytea(20) or hex text) -> uniform spread -> per-bucket INDEX.
--  * Addresses are written to exactly ONE parent partitioned table from any
--    ingest worker -> no cross-region write hotspots.
--  * Balances/fees/history read from the local partition (pgBouncer route by
--    chain_id) and updated idempotently (last-write-wins by block height).
--  * The chain registry itself lives in master_wallet/backend:data.
--
-- Runtime compat: PostgreSQL 15+. Partition key uses BTREE-hash of the raw
-- hex address computed in SQL (no extension needed).

CREATE TABLE IF NOT EXISTS uw_addresses (
    chain_id      BIGINT      NOT NULL,
    address       TEXT        NOT NULL,           -- lowercase hex or base58
    account_index INTEGER     NOT NULL DEFAULT 0, -- HD account index (one seed -> many accounts)
    label         VARCHAR(64),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, address)
) PARTITION BY HASH (chain_id);

-- 16 hash buckets balance throughput vs. partition-management overhead.
-- Tune upward (32/64) via `database/schemas/user_wallet_sharding_recreate.sql`
-- only after measuring write/QPS distribution in production.
DO $$
DECLARE
    i INT;
BEGIN
    FOR i IN 0..15 LOOP
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS uw_addresses_%s PARTITION OF uw_addresses FOR VALUES WITH (MODULUS 16, REMAINDER %s)',
            to_hex(i), i);
    END LOOP;
END $$;

-- Per-chain recent balance cache (hot path for get_balance UX).
CREATE TABLE IF NOT EXISTS uw_balances (
    chain_id    BIGINT    NOT NULL,
    address     TEXT      NOT NULL,
    token       TEXT      NOT NULL DEFAULT 'native',     -- 'native' or contract addr
    amount      NUMERIC(78, 0) NOT NULL DEFAULT 0,       -- base units (wei/sat)
    block_number BIGINT   NOT NULL,                       -- idempotent ordering
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, address, token)
) PARTITION BY HASH (chain_id);

DO $$
DECLARE
    i INT;
BEGIN
    FOR i IN 0..15 LOOP
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS uw_balances_%s PARTITION OF uw_balances FOR VALUES WITH (MODULUS 16, REMAINDER %s)',
            to_hex(i), i);
    END LOOP;
END $$;

-- Per-address transaction index (append-only; hash partition -> no hotspot).
CREATE TABLE IF NOT EXISTS uw_transactions (
    chain_id     BIGINT      NOT NULL,
    address      TEXT        NOT NULL,                    -- from OR to (both rows for same tx okay)
    direction    CHAR(1)     NOT NULL CHECK (direction IN ('i','o')),
    tx_hash      TEXT        NOT NULL,
    block_number BIGINT      NOT NULL,
    value        NUMERIC(78,0) NOT NULL DEFAULT 0,
    token        TEXT        NOT NULL DEFAULT 'native',
    status       VARCHAR(16) NOT NULL DEFAULT 'confirmed', -- pending|confirmed|final|failed|reorged
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, address, tx_hash, direction)
) PARTITION BY HASH (chain_id);

DO $$
DECLARE
    i INT;
BEGIN
    FOR i IN 0..15 LOOP
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS uw_transactions_%s PARTITION OF uw_transactions FOR VALUES WITH (MODULUS 16, REMAINDER %s)',
            to_hex(i), i);
    END LOOP;
END $$;

-- Indexer high-water mark per chain (tells us which partition is synced to
-- which block height; used for optimistic catches + reorg handling).
CREATE TABLE IF NOT EXISTS uw_indexer_checkpoint (
    chain_id       BIGINT       PRIMARY KEY,
    block_number   BIGINT       NOT NULL,
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Hot-path HOT standby: last heartbeat came in; last block for this chain.
CREATE TABLE IF NOT EXISTS uw_chain_meta (
    chain_id    BIGINT       PRIMARY KEY,
    name        VARCHAR(64)  NOT NULL,
    symbol      VARCHAR(16)  NOT NULL,
    enabled     BOOLEAN      NOT NULL DEFAULT TRUE,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Functional index that actually gets used by the user client:
-- GET /balance(address@chain) and GET /history(address@chain).
CREATE INDEX IF NOT EXISTS idx_uw_balances_lookup
    ON uw_balances (chain_id, address) WHERE token = 'native';
CREATE INDEX IF NOT EXISTS idx_uw_transactions_history
    ON uw_transactions (chain_id, address, block_number DESC);

-- =============================================================================
-- BILLING/SCALE runbook (not a migration): the above tables are the start of
-- the per-chain pgBouncer route. In production:
--   1. One (or a few) `uw_*` parents per chain -> route by chain_id
--   2. Partition count raised only after measuring skew (per Phase 21)
--   3. uw_transactions is append-only and archived by CREATE TABLE ...
--      PARTITION OF ... RANGE (block_number) for cold storage on demand
-- =============================================================================
