-- PostgreSQL Database Schema for White Label System
-- High-performance, fully normalized schema

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- ENUM TYPES
-- ============================================================================

CREATE TYPE wl_status AS ENUM ('pending', 'active', 'suspended', 'halted', 'expired', 'revoked');
CREATE TYPE wl_plan AS ENUM ('starter', 'professional', 'enterprise', 'custom');
CREATE TYPE admin_role AS ENUM ('super_admin', 'admin', 'manager', 'support');
CREATE TYPE product_type AS ENUM ('trading', 'perpetual', 'staking', 'nft', 'wallet', 'bridge', 'launchpad');
CREATE TYPE product_status AS ENUM ('enabled', 'disabled', 'maintenance');
CREATE TYPE pair_status AS ENUM ('active', 'suspended', 'halted');
CREATE TYPE bot_strategy AS ENUM ('arbitrage', 'market_making', 'liquidity', 'grid', 'dca');
CREATE TYPE bot_status AS ENUM ('running', 'stopped', 'error', 'paused');
CREATE TYPE chain_category AS ENUM ('evm', 'solana', 'aptos', 'sui', 'ton', 'bitcoin', 'cosmos', 'polkadot');
CREATE TYPE token_type AS ENUM ('erc20', 'bep20', 'spl', 'native', 'trc20');

-- ============================================================================
-- CLIENTS TABLE
-- ============================================================================

CREATE TABLE wl_clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(512) NOT NULL UNIQUE,
    subdomain VARCHAR(255),
    custom_branding BOOLEAN DEFAULT true,
    logo_url TEXT,
    primary_color VARCHAR(7) DEFAULT '#1976d2',
    secondary_color VARCHAR(7) DEFAULT '#1a1a2e',
    status wl_status DEFAULT 'pending',
    plan wl_plan DEFAULT 'starter',
    max_users INTEGER DEFAULT 1000,
    current_users INTEGER DEFAULT 0,
    fee_percent DECIMAL(5,2) DEFAULT 20.00,
    features JSONB DEFAULT '{}',
    blockchain_access JSONB DEFAULT '[]',
    api_keys JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    approved_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    
    CONSTRAINT valid_fee CHECK (fee_percent >= 0 AND fee_percent <= 50),
    CONSTRAINT valid_users CHECK (max_users > 0)
);

CREATE INDEX idx_clients_status ON wl_clients(status);
CREATE INDEX idx_clients_domain ON wl_clients(domain);
CREATE INDEX idx_clients_plan ON wl_clients(plan);
CREATE INDEX idx_clients_created ON wl_clients(created_at DESC);

-- ============================================================================
-- ADMINS TABLE
-- ============================================================================

CREATE TABLE wl_admins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES wl_clients(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role admin_role DEFAULT 'support',
    permissions JSONB DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'active',
    two_factor_enabled BOOLEAN DEFAULT false,
    two_factor_secret VARCHAR(255),
    last_login TIMESTAMPTZ,
    login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT valid_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

CREATE INDEX idx_admins_client ON wl_admins(client_id);
CREATE INDEX idx_admins_email ON wl_admins(email);
CREATE INDEX idx_admins_role ON wl_admins(role);
CREATE INDEX idx_admins_status ON wl_admins(status);

-- ============================================================================
-- PRODUCTS TABLE
-- ============================================================================

CREATE TABLE wl_products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES wl_clients(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type product_type NOT NULL,
    description TEXT,
    status product_status DEFAULT 'enabled',
    fee DECIMAL(10,4) DEFAULT 0,
    min_deposit DECIMAL(32,8) DEFAULT 0,
    max_deposit DECIMAL(32,8) DEFAULT 1000000,
    min_withdrawal DECIMAL(32,8) DEFAULT 0,
    max_withdrawal DECIMAL(32,8) DEFAULT 1000000,
    features JSONB DEFAULT '[]',
    settings JSONB DEFAULT '{}',
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_products_client ON wl_products(client_id);
CREATE INDEX idx_products_type ON wl_products(type);
CREATE INDEX idx_products_status ON wl_products(status);

-- ============================================================================
-- TRADING PAIRS TABLE
-- ============================================================================

CREATE TABLE wl_trading_pairs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES wl_clients(id) ON DELETE CASCADE,
    base_token VARCHAR(50) NOT NULL,
    quote_token VARCHAR(50) NOT NULL,
    chain_id BIGINT NOT NULL,
    pair_address VARCHAR(100),
    status pair_status DEFAULT 'active',
    fee DECIMAL(10,4) DEFAULT 0.1,
    min_trade DECIMAL(32,8) DEFAULT 0.001,
    max_trade DECIMAL(32,8) DEFAULT 1000000,
    liquidity DECIMAL(32,8) DEFAULT 0,
    price_precision INTEGER DEFAULT 8,
    quantity_precision INTEGER DEFAULT 8,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_client_pair UNIQUE (client_id, base_token, quote_token, chain_id)
);

CREATE INDEX idx_pairs_client ON wl_trading_pairs(client_id);
CREATE INDEX idx_pairs_status ON wl_trading_pairs(status);
CREATE INDEX idx_pairs_chain ON wl_trading_pairs(chain_id);

-- ============================================================================
-- LIQUIDITY POOLS TABLE
-- ============================================================================

CREATE TABLE wl_liquidity_pools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES wl_clients(id) ON DELETE CASCADE,
    pair_id UUID REFERENCES wl_trading_pairs(id) ON DELETE CASCADE,
    provider VARCHAR(100) DEFAULT 'internal',
    token_a VARCHAR(50) NOT NULL,
    token_b VARCHAR(50) NOT NULL,
    amount_a DECIMAL(32,8) DEFAULT 0,
    amount_b DECIMAL(32,8) DEFAULT 0,
    value_usd DECIMAL(32,8) DEFAULT 0,
    apr DECIMAL(10,4) DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_pools_client ON wl_liquidity_pools(client_id);
CREATE INDEX idx_pools_pair ON wl_liquidity_pools(pair_id);

-- ============================================================================
-- TOKEN CONFIGURATIONS TABLE
-- ============================================================================

CREATE TABLE wl_token_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES wl_clients(id) ON DELETE CASCADE,
    address VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    decimals INTEGER DEFAULT 18,
    chain_id BIGINT NOT NULL,
    type token_type DEFAULT 'erc20',
    status VARCHAR(50) DEFAULT 'active',
    max_supply VARCHAR(100),
    features JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_token_chain UNIQUE (client_id, address, chain_id)
);

CREATE INDEX idx_tokens_client ON wl_token_configs(client_id);
CREATE INDEX idx_tokens_chain ON wl_token_configs(chain_id);
CREATE INDEX idx_tokens_address ON wl_token_configs(address);

-- ============================================================================
-- MARKET MAKER BOTS TABLE
-- ============================================================================

CREATE TABLE wl_market_maker_bots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES wl_clients(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    pair_ids JSONB DEFAULT '[]',
    strategy bot_strategy NOT NULL,
    status bot_status DEFAULT 'stopped',
    params JSONB DEFAULT '{}',
    profit DECIMAL(32,8) DEFAULT 0,
    volume_24h DECIMAL(32,8) DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    stopped_at TIMESTAMPTZ
);

CREATE INDEX idx_bots_client ON wl_market_maker_bots(client_id);
CREATE INDEX idx_bots_status ON wl_market_maker_bots(status);

-- ============================================================================
-- BLOCKCHAINS TABLE
-- ============================================================================

CREATE TABLE wl_blockchains (
    id BIGINT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    category chain_category NOT NULL,
    rpc_urls JSONB DEFAULT '[]',
    explorer_urls JSONB DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'enabled',
    is_default BOOLEAN DEFAULT false,
    icon_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- API KEYS TABLE
-- ============================================================================

CREATE TABLE wl_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES wl_clients(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    secret_hash VARCHAR(255),
    permissions JSONB DEFAULT '[]',
    rate_limit INTEGER DEFAULT 1000,
    status VARCHAR(50) DEFAULT 'active',
    last_used TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_apikeys_client ON wl_api_keys(client_id);
CREATE INDEX idx_apikeys_hash ON wl_api_keys(key_hash);

-- ============================================================================
-- AUDIT LOGS TABLE
-- ============================================================================

CREATE TABLE wl_audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES wl_clients(id) ON DELETE CASCADE,
    admin_id UUID REFERENCES wl_admins(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(50) DEFAULT 'success',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_client ON wl_audit_logs(client_id);
CREATE INDEX idx_audit_admin ON wl_audit_logs(admin_id);
CREATE INDEX idx_audit_action ON wl_audit_logs(action);
CREATE INDEX idx_audit_created ON wl_audit_logs(created_at DESC);

-- ============================================================================
-- NOTIFICATIONS TABLE
-- ============================================================================

CREATE TABLE wl_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES wl_clients(id) ON DELETE CASCADE,
    admin_id UUID REFERENCES wl_admins(id) ON DELETE SET NULL,
    type VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    data JSONB DEFAULT '{}',
    read BOOLEAN DEFAULT false,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notif_client ON wl_notifications(client_id);
CREATE INDEX idx_notif_admin ON wl_notifications(admin_id);
CREATE INDEX idx_notif_read ON wl_notifications(read);
CREATE INDEX idx_notif_created ON wl_notifications(created_at DESC);

-- ============================================================================
-- SESSIONS TABLE
-- ============================================================================

CREATE TABLE wl_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID REFERENCES wl_admins(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_activity TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sessions_admin ON wl_sessions(admin_id);
CREATE INDEX idx_sessions_token ON wl_sessions(token_hash);
CREATE INDEX idx_sessions_expires ON wl_sessions(expires_at);

-- ============================================================================
-- ANALYTICS TABLES
-- ============================================================================

CREATE TABLE wl_analytics_daily (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES wl_clients(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    active_users INTEGER DEFAULT 0,
    new_users INTEGER DEFAULT 0,
    total_volume DECIMAL(32,8) DEFAULT 0,
    trading_volume DECIMAL(32,8) DEFAULT 0,
    fees_collected DECIMAL(32,8) DEFAULT 0,
    transactions_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_client_date UNIQUE (client_id, date)
);

CREATE INDEX idx_analytics_client ON wl_analytics_daily(client_id);
CREATE INDEX idx_analytics_date ON wl_analytics_daily(date);

-- ============================================================================
-- FUNCTIONS AND TRIGGERS
-- ============================================================================

-- Update updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create update triggers
CREATE TRIGGER update_clients_updated_at BEFORE UPDATE ON wl_clients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_admins_updated_at BEFORE UPDATE ON wl_admins
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON wl_products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pairs_updated_at BEFORE UPDATE ON wl_trading_pairs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pools_updated_at BEFORE UPDATE ON wl_liquidity_pools
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tokens_updated_at BEFORE UPDATE ON wl_token_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bots_updated_at BEFORE UPDATE ON wl_market_maker_bots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Auto-delete expired sessions
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS void AS $$
BEGIN
    DELETE FROM wl_sessions WHERE expires_at < NOW();
END;
$$ language 'plpgsql';

-- Run cleanup daily
SELECT cron.schedule('cleanup-sessions', '0 0 * * *', 'SELECT cleanup_expired_sessions()');

-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Insert default blockchains
INSERT INTO wl_blockchains (id, name, symbol, category, status, is_default) VALUES
(1, 'Ethereum', 'ETH', 'evm', 'enabled', true),
(2, 'BNB Smart Chain', 'BNB', 'evm', 'enabled', false),
(3, 'Polygon', 'MATIC', 'evm', 'enabled', false),
(4, 'Arbitrum', 'ETH', 'evm', 'enabled', false),
(5, 'Optimism', 'ETH', 'evm', 'enabled', false),
(6, 'Base', 'ETH', 'evm', 'enabled', false),
(7, 'Avalanche', 'AVAX', 'evm', 'enabled', false),
(101, 'Bitcoin', 'BTC', 'bitcoin', 'enabled', false),
(102, 'Solana', 'SOL', 'solana', 'enabled', false),
(103, 'Tron', 'TRX', 'evm', 'enabled', false)
ON CONFLICT (id) DO NOTHING;

-- Insert default products
INSERT INTO wl_products (name, type, description, status, fee, min_deposit, max_deposit) VALUES
('Spot Trading', 'trading', 'Spot trading with limit and market orders', 'enabled', 0.1, 10, 1000000),
('Perpetual Trading', 'perpetual', 'Perpetual futures trading', 'enabled', 0.05, 100, 500000),
('Staking', 'staking', 'Stake tokens and earn rewards', 'enabled', 0, 0, 10000000),
('NFT Marketplace', 'nft', 'Buy and sell NFTs', 'enabled', 2.5, 0, 100000),
('Wallet', 'wallet', 'Multi-chain wallet', 'enabled', 0, 0, 10000000),
('Bridge', 'bridge', 'Cross-chain bridge', 'enabled', 0.3, 50, 500000),
('Launchpad', 'launchpad', 'Token launch and IDO platform', 'enabled', 5, 1000, 100000)
ON CONFLICT DO NOTHING;
