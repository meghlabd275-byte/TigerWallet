-- TigerWallet database initialization script
-- Run by docker-entrypoint-initdb.d on first PostgreSQL boot.
-- Creates the tigerwallet database and the wallet-api schema.

CREATE DATABASE tigerwallet;
\connect tigerwallet;

-- Wallet API schema (canonical, used by go/wallet_api)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    kyc_status TEXT DEFAULT 'unverified',
    kyc_level INT DEFAULT 0,
    two_factor_enabled BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    chain_id BIGINT NOT NULL,
    address TEXT NOT NULL,
    encrypted_seed TEXT NOT NULL,
    derivation_path TEXT NOT NULL,
    account_index INT DEFAULT 0,
    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS address_book (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    chain_id BIGINT NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transaction_log (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    tx_hash TEXT NOT NULL,
    chain_id BIGINT NOT NULL,
    from_addr TEXT NOT NULL,
    to_addr TEXT NOT NULL,
    value TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wallets_user ON wallets(user_id);
CREATE INDEX IF NOT EXISTS idx_wallets_address ON wallets(address);
CREATE INDEX IF NOT EXISTS idx_txlog_user ON transaction_log(user_id);
CREATE INDEX IF NOT EXISTS idx_txlog_hash ON transaction_log(tx_hash);

-- Also create the admin database for the admin platform services
CREATE DATABASE tigerwallet_admin;
\connect tigerwallet_admin;
-- Admin schema is managed by the admin service migrations.

-- ProjectParty database (token listing + launchpad platform)
CREATE DATABASE tigerwallet_project_party;
\connect tigerwallet_project_party;
-- ProjectParty schema is managed by the project_party service migrations on boot.

-- Bridge database (cross-chain bridge transactions)
CREATE DATABASE tigerwallet_bridge;
\connect tigerwallet_bridge;
-- Bridge schema is managed by the bridge service migrations on boot.

-- Standalone WL-UserWallet database (independent of TigerWallet cloud).
CREATE DATABASE wl_userwallet;
\connect wl_userwallet;
-- WL-UserWallet schema is managed by the service's own migrations on boot
-- (users, wallets with encrypted_seed, transactions, address_book).

-- Standalone WL-MasterWallet database (independent of TigerWallet cloud).
CREATE DATABASE wl_masterwallet;
\connect wl_masterwallet;
-- WL-MasterWallet schema is managed by the service's own migrations on boot
-- (master_wallets with encrypted_seed, sub_wallets, transactions, policies,
--  fee_configs, auto_sign_rules, audit_log).

-- Standalone WL-Bots database (independent of TigerWallet cloud).
CREATE DATABASE wl_bots;
\connect wl_bots;
-- WL-Bots schema is managed by the service's own migrations on boot
-- (users, bots, bot_executions, subscriptions, fee_configs, api_keys, bot_logs).

-- Standalone WL-ProjectParty database (independent of TigerWallet cloud).
CREATE DATABASE wl_projectparty;
\connect wl_projectparty;
-- WL-ProjectParty schema is managed by the service's own migrations on boot
-- (users, tokens, listings, launchpad_projects, participations,
--  market_making_configs, fee_configs, favorites).

-- Standalone WL-Liquidity database (independent of TigerWallet cloud).
CREATE DATABASE wl_liquidity;
\connect wl_liquidity;
-- WL-Liquidity schema is managed by the service's own migrations on boot
-- (users, liquidity_sources, liquidity_routes).

-- Standalone WL-Card (WL-branded CryptoCard) database.
CREATE DATABASE wl_card;
\connect wl_card;
-- WL-Card schema is managed by the service's own migrations on boot
-- (users, cards with encrypted PAN/CVV, card_transactions).
