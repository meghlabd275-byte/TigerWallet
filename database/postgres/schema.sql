-- ============================================================================
-- TigerWallet Complete PostgreSQL Database Schema
-- Ultra-low latency, high-performance design
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "hstore";

-- ============================================================================
-- ENUMS
-- ============================================================================

CREATE TYPE admin_role AS ENUM ('super_admin', 'master_admin', 'white_label_admin', 'support');
CREATE TYPE admin_status AS ENUM ('active', 'suspended', 'banned');
CREATE TYPE wl_status AS ENUM ('pending', 'active', 'suspended', 'revoked');
CREATE TYPE kyc_status AS ENUM ('none', 'pending', 'submitted', 'approved', 'rejected');
CREATE TYPE tx_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE tx_type AS ENUM ('deposit', 'withdrawal', 'transfer', 'swap', 'staking', 'bridge');
CREATE TYPE product_type AS ENUM ('trading', 'wallet', 'staking', 'nft', 'bridge', 'defi');
CREATE TYPE product_status AS ENUM ('enabled', 'disabled', 'maintenance');

-- ============================================================================
-- ADMIN USERS TABLE
-- ============================================================================

CREATE TABLE admin_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role admin_role NOT NULL DEFAULT 'support',
    security_level INTEGER NOT NULL DEFAULT 1,
    permissions JSONB DEFAULT '[]',
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login TIMESTAMP WITH TIME ZONE,
    status admin_status DEFAULT 'active',
    failed_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    ip_whitelist TEXT[],
    password_changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    must_change_password BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_admin_username ON admin_users(username);
CREATE INDEX idx_admin_email ON admin_users(email);
CREATE INDEX idx_admin_role ON admin_users(role);
CREATE INDEX idx_admin_status ON admin_users(status);

-- ============================================================================
-- ADMIN SESSIONS TABLE
-- ============================================================================

CREATE TABLE admin_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID REFERENCES admin_users(id) ON DELETE CASCADE,
    token VARCHAR(500) UNIQUE NOT NULL,
    refresh_token VARCHAR(500),
    ip_address INET NOT NULL,
    user_agent TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_valid BOOLEAN DEFAULT TRUE,
    device_info JSONB,
    location JSONB
);

CREATE INDEX idx_session_token ON admin_sessions(token);
CREATE INDEX idx_session_admin ON admin_sessions(admin_id);
CREATE INDEX idx_session_expires ON admin_sessions(expires_at);

-- ============================================================================
-- WHITE LABELS TABLE
-- ============================================================================

CREATE TABLE white_labels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE NOT NULL,
    subdomain VARCHAR(100),
    api_key_hash VARCHAR(255) NOT NULL,
    api_secret_hash VARCHAR(255),
    fee_percent DECIMAL(5,2) NOT NULL DEFAULT 20.00 CHECK (fee_percent BETWEEN 0 AND 20),
    profit_share_percent DECIMAL(5,2) NOT NULL DEFAULT 20 CHECK (profit_share_percent BETWEEN 0 AND 50),
    profit_share_schedule VARCHAR(50) DEFAULT 'monthly',
    status wl_status NOT NULL DEFAULT 'pending',
    custom_branding BOOLEAN DEFAULT TRUE,
    branding_config JSONB DEFAULT '{}',
    features JSONB DEFAULT '[]',
    master_wallet_address VARCHAR(100),
    master_wallet_private_key_encrypted TEXT,
    approved_by UUID REFERENCES admin_users(id),
    approved_at TIMESTAMP WITH TIME ZONE,
    created_by UUID REFERENCES admin_users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    plan_tier VARCHAR(50) DEFAULT 'basic',
    max_users INTEGER DEFAULT 1000,
    max_api_calls INTEGER DEFAULT 100000,
    monthly_fee DECIMAL(10,2) DEFAULT 0,
    custom_css TEXT,
    custom_js TEXT,
    support_email VARCHAR(255),
    terms_url VARCHAR(500),
    privacy_url VARCHAR(500),
    ssl_enabled BOOLEAN DEFAULT FALSE,
    ssl_cert TEXT,
    ssl_key TEXT
);

CREATE INDEX idx_wl_domain ON white_labels(domain);
CREATE INDEX idx_wl_status ON white_labels(status);
CREATE INDEX idx_wl_plan ON white_labels(plan_tier);

-- ============================================================================
-- WHITE LABEL API KEYS
-- ============================================================================

CREATE TABLE wl_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID NOT NULL REFERENCES white_labels(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    secret_hash VARCHAR(255),
    permissions JSONB DEFAULT '[]',
    rate_limit_minute INTEGER DEFAULT 60,
    rate_limit_day INTEGER DEFAULT 10000,
    is_active BOOLEAN DEFAULT TRUE,
    last_used TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES admin_users(id),
    ip_whitelist TEXT[],
    scopes TEXT[]
);

CREATE INDEX idx_wl_api_keys_wl ON wl_api_keys(white_label_id);
CREATE INDEX idx_wl_api_keys_hash ON wl_api_keys(key_hash);

-- ============================================================================
-- WHITE LABEL ADMINS
-- ============================================================================

CREATE TABLE white_label_admins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID NOT NULL REFERENCES white_labels(id) ON DELETE CASCADE,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'admin',
    permissions JSONB DEFAULT '[]',
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret VARCHAR(255),
    status admin_status DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login TIMESTAMP WITH TIME ZONE,
    failed_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    UNIQUE(white_label_id, username)
);

CREATE INDEX idx_wl_admin_wl ON white_label_admins(white_label_id);
CREATE INDEX idx_wl_admin_email ON white_label_admins(email);

-- ============================================================================
-- PRODUCTS TABLE
-- ============================================================================

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type product_type NOT NULL,
    description TEXT,
    status product_status DEFAULT 'enabled',
    fee_percent DECIMAL(6,4) DEFAULT 0,
    min_deposit DECIMAL(30,8) DEFAULT 0,
    max_deposit DECIMAL(30,8),
    min_withdrawal DECIMAL(30,8) DEFAULT 0,
    max_withdrawal DECIMAL(30,8),
    features JSONB DEFAULT '[]',
    supported_chains TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_global BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_product_wl ON products(white_label_id);
CREATE INDEX idx_product_type ON products(type);
CREATE INDEX idx_product_status ON products(status);

-- ============================================================================
-- TRADING PAIRS
-- ============================================================================

CREATE TABLE trading_pairs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id) ON DELETE CASCADE,
    base_token VARCHAR(100) NOT NULL,
    quote_token VARCHAR(100) NOT NULL,
    pair_symbol VARCHAR(50) NOT NULL,
    min_trade_amount DECIMAL(30,8) DEFAULT 0,
    max_trade_amount DECIMAL(30,8),
    maker_fee DECIMAL(6,4) DEFAULT 0.001,
    taker_fee DECIMAL(6,4) DEFAULT 0.001,
    status VARCHAR(50) DEFAULT 'active',
    chain_id VARCHAR(20),
    dex_router_address VARCHAR(100),
    liquidity_source VARCHAR(100),
    price_precision INTEGER DEFAULT 8,
    quantity_precision INTEGER DEFAULT 8,
    min_slippage DECIMAL(5,2) DEFAULT 0.5,
    max_slippage DECIMAL(5,2) DEFAULT 10,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(white_label_id, pair_symbol)
);

CREATE INDEX idx_pair_wl ON trading_pairs(white_label_id);
CREATE INDEX idx_pair_symbol ON trading_pairs(pair_symbol);

-- ============================================================================
-- USERS TABLE
-- ============================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id) ON DELETE SET NULL,
    email VARCHAR(255) UNIQUE,
    username VARCHAR(100),
    wallet_address VARCHAR(100) UNIQUE,
    public_key VARCHAR(200),
    kyc_status kyc_status DEFAULT 'none',
    kyc_level INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active',
    risk_score INTEGER DEFAULT 0,
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_active TIMESTAMP WITH TIME ZONE,
    referral_code VARCHAR(50) UNIQUE,
    referred_by UUID REFERENCES users(id),
    account_source VARCHAR(50) DEFAULT 'direct',
    platform VARCHAR(50),
    country VARCHAR(50),
    kyc_data JSONB
);

CREATE INDEX idx_user_wl ON users(white_label_id);
CREATE INDEX idx_user_wallet ON users(wallet_address);
CREATE INDEX idx_user_email ON users(email);
CREATE INDEX idx_user_kyc ON users(kyc_status);

-- ============================================================================
-- USER BALANCES
-- ============================================================================

CREATE TABLE user_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_symbol VARCHAR(50) NOT NULL,
    token_address VARCHAR(100),
    chain_id VARCHAR(20) NOT NULL,
    balance DECIMAL(30,8) DEFAULT 0,
    locked_balance DECIMAL(30,8) DEFAULT 0,
    available_balance DECIMAL(30,8) GENERATED ALWAYS AS (balance - locked_balance) STORED,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, token_symbol, chain_id)
);

CREATE INDEX idx_balance_user ON user_balances(user_id);
CREATE INDEX idx_balance_token ON user_balances(token_symbol);

-- ============================================================================
-- TRANSACTIONS
-- ============================================================================

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    white_label_id UUID REFERENCES white_labels(id) ON DELETE SET NULL,
    tx_hash VARCHAR(200) UNIQUE,
    type tx_type NOT NULL,
    status tx_status DEFAULT 'pending',
    from_address VARCHAR(100),
    to_address VARCHAR(100),
    token_symbol VARCHAR(50),
    token_address VARCHAR(100),
    chain_id VARCHAR(20),
    amount DECIMAL(30,8) NOT NULL,
    fee DECIMAL(30,8) DEFAULT 0,
    fee_token VARCHAR(50),
    usd_value DECIMAL(20,2),
    gas_used BIGINT,
    gas_price VARCHAR(50),
    block_number BIGINT,
    confirmations INTEGER DEFAULT 0,
    required_confirmations INTEGER DEFAULT 12,
    metadata JSONB DEFAULT '{}',
    error_message TEXT,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_tx_user ON transactions(user_id);
CREATE INDEX idx_tx_wl ON transactions(white_label_id);
CREATE INDEX idx_tx_hash ON transactions(tx_hash);
CREATE INDEX idx_tx_status ON transactions(status);
CREATE INDEX idx_tx_type ON transactions(type);
CREATE INDEX idx_tx_created ON transactions(created_at DESC);

-- ============================================================================
-- KYC RECORDS
-- ============================================================================

CREATE TABLE kyc_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kyc_type VARCHAR(50) NOT NULL,
    status kyc_status DEFAULT 'pending',
    document_type VARCHAR(50),
    document_id VARCHAR(100),
    document_url TEXT,
    selfie_url TEXT,
    verified_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,
    reviewed_by UUID REFERENCES admin_users(id),
    data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_kyc_user ON kyc_records(user_id);
CREATE INDEX idx_kyc_status ON kyc_records(status);

-- ============================================================================
-- BLOCKCHAINS
-- ============================================================================

CREATE TABLE blockchains (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    chain_id VARCHAR(20) UNIQUE NOT NULL,
    chain_type VARCHAR(50),
    rpc_urls JSONB DEFAULT '[]',
    explorer_urls JSONB DEFAULT '[]',
    native_token VARCHAR(50),
    decimals INTEGER DEFAULT 18,
    is_active BOOLEAN DEFAULT TRUE,
    is_testnet BOOLEAN DEFAULT FALSE,
    min_confirmations INTEGER DEFAULT 12,
    gas_limit INTEGER DEFAULT 21000,
    added_by UUID REFERENCES admin_users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_chain_id ON blockchains(chain_id);
CREATE INDEX idx_chain_active ON blockchains(is_active);

-- ============================================================================
-- FEE STRUCTURES
-- ============================================================================

CREATE TABLE fee_structures (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    token_symbol VARCHAR(50),
    chain_id VARCHAR(20),
    fee_percent DECIMAL(6,4) DEFAULT 0,
    fee_fixed DECIMAL(30,8) DEFAULT 0,
    min_amount DECIMAL(30,8),
    max_amount DECIMAL(30,8),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_fee_wl ON fee_structures(white_label_id);
CREATE INDEX idx_fee_type ON fee_structures(type);

-- ============================================================================
-- AUDIT LOGS
-- ============================================================================

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID REFERENCES admin_users(id) ON DELETE SET NULL,
    white_label_id UUID REFERENCES white_labels(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(50) DEFAULT 'success',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_audit_admin ON audit_logs(admin_id);
CREATE INDEX idx_audit_wl ON audit_logs(white_label_id);
CREATE INDEX idx_audit_action ON audit_logs(action);
CREATE INDEX idx_audit_created ON audit_logs(created_at DESC);

-- ============================================================================
-- WEBHOOKS
-- ============================================================================

CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    url TEXT NOT NULL,
    events TEXT[] NOT NULL,
    secret_hash VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    retry_count INTEGER DEFAULT 3,
    timeout_seconds INTEGER DEFAULT 30,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_triggered TIMESTAMP WITH TIME ZONE,
    failure_count INTEGER DEFAULT 0
);

CREATE INDEX idx_webhook_wl ON webhooks(white_label_id);
CREATE INDEX idx_webhook_active ON webhooks(is_active);

-- ============================================================================
-- NOTIFICATIONS
-- ============================================================================

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    white_label_id UUID REFERENCES white_labels(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    data JSONB DEFAULT '{}',
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_notification_user ON notifications(user_id);
CREATE INDEX idx_notification_read ON notifications(is_read);
CREATE INDEX idx_notification_created ON notifications(created_at DESC);

-- ============================================================================
-- RATE LIMITS
-- ============================================================================

CREATE TABLE rate_limits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    identifier VARCHAR(255) NOT NULL,
    endpoint VARCHAR(100) NOT NULL,
    request_count INTEGER DEFAULT 0,
    window_start TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    window_duration INTEGER DEFAULT 60,
    blocked_until TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(identifier, endpoint)
);

CREATE INDEX idx_rate_limit_identifier ON rate_limits(identifier);
CREATE INDEX idx_rate_limit_window ON rate_limits(window_start);

-- ============================================================================
-- API KEYS (USER)
-- ============================================================================

CREATE TABLE user_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    secret_hash VARCHAR(255),
    permissions JSONB DEFAULT '[]',
    rate_limit_minute INTEGER DEFAULT 60,
    rate_limit_day INTEGER DEFAULT 10000,
    is_active BOOLEAN DEFAULT TRUE,
    last_used TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    ip_whitelist TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES admin_users(id)
);

CREATE INDEX idx_user_api_key_user ON user_api_keys(user_id);
CREATE INDEX idx_user_api_key_hash ON user_api_keys(key_hash);

-- ============================================================================
-- TRADING BOTS
-- ============================================================================

CREATE TABLE trading_bots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    white_label_id UUID REFERENCES white_labels(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    bot_type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'inactive',
    running BOOLEAN DEFAULT FALSE,
    initial_capital DECIMAL(30,8),
    current_capital DECIMAL(30,8),
    profit_loss DECIMAL(30,8) DEFAULT 0,
    profit_loss_percent DECIMAL(10,4) DEFAULT 0,
    start_date TIMESTAMP WITH TIME ZONE,
    last_run TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_bot_user ON trading_bots(user_id);
CREATE INDEX idx_bot_wl ON trading_bots(white_label_id);
CREATE INDEX idx_bot_status ON trading_bots(status);

-- ============================================================================
-- LIQUIDITY POOLS
-- ============================================================================

CREATE TABLE liquidity_pools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id) ON DELETE CASCADE,
    pair_address VARCHAR(100) NOT NULL,
    token_a VARCHAR(50) NOT NULL,
    token_b VARCHAR(50) NOT NULL,
    reserve_a DECIMAL(30,8) DEFAULT 0,
    reserve_b DECIMAL(30,8) DEFAULT 0,
    liquidity_token_address VARCHAR(100),
    total_liquidity DECIMAL(30,8) DEFAULT 0,
    apr DECIMAL(10,4) DEFAULT 0,
    volume_24h DECIMAL(30,8) DEFAULT 0,
    fees_24h DECIMAL(30,8) DEFAULT 0,
    chain_id VARCHAR(20),
    dex VARCHAR(50),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_pool_wl ON liquidity_pools(white_label_id);
CREATE INDEX idx_pool_pair ON liquidity_pools(pair_address);

-- ============================================================================
-- TOKEN LISTING REQUESTS
-- ============================================================================

CREATE TABLE token_listing_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id) ON DELETE SET NULL,
    token_name VARCHAR(255) NOT NULL,
    token_symbol VARCHAR(50) NOT NULL,
    token_address VARCHAR(100) NOT NULL,
    chain_id VARCHAR(20) NOT NULL,
    decimals INTEGER,
    total_supply DECIMAL(30,8),
    logo_url TEXT,
    website_url TEXT,
    description TEXT,
    tier VARCHAR(50) DEFAULT 'basic',
    status VARCHAR(50) DEFAULT 'pending',
    one_time_fee DECIMAL(10,2) DEFAULT 0,
    monthly_fee DECIMAL(10,2) DEFAULT 0,
    requested_by UUID REFERENCES users(id),
    reviewed_by UUID REFERENCES admin_users(id),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_token_request_wl ON token_listing_requests(white_label_id);
CREATE INDEX idx_token_request_status ON token_listing_requests(status);

-- ============================================================================
-- CEX CONNECTIONS
-- ============================================================================

CREATE TABLE cex_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id) ON DELETE CASCADE,
    exchange_name VARCHAR(100) NOT NULL,
    api_key_encrypted TEXT,
    api_secret_encrypted TEXT,
    passphrase_encrypted TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    permissions JSONB DEFAULT '[]',
    rate_limit INTEGER DEFAULT 1200,
    last_sync TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_cex_wl ON cex_connections(white_label_id);

-- ============================================================================
-- DEX CONNECTIONS
-- ============================================================================

CREATE TABLE dex_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id) ON DELETE CASCADE,
    dex_name VARCHAR(100) NOT NULL,
    router_address VARCHAR(100),
    factory_address VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    supported_chains TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_dex_wl ON dex_connections(white_label_id);

-- ============================================================================
-- PROFIT SHARING
-- ============================================================================

CREATE TABLE profit_sharing (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID NOT NULL REFERENCES white_labels(id) ON DELETE CASCADE,
    super_admin_wallet VARCHAR(100) NOT NULL,
    master_wallet_address VARCHAR(100),
    profit_percentage DECIMAL(5,2) NOT NULL,
    min_percentage DECIMAL(5,2) DEFAULT 0,
    max_percentage DECIMAL(5,2) DEFAULT 50,
    is_active BOOLEAN DEFAULT TRUE,
    auto_transfer BOOLEAN DEFAULT TRUE,
    transfer_frequency VARCHAR(50) DEFAULT 'daily',
    last_transfer TIMESTAMP WITH TIME ZONE,
    total_transferred DECIMAL(30,8) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_profit_wl ON profit_sharing(white_label_id);

-- ============================================================================
-- PROFIT TRANSACTIONS
-- ============================================================================

CREATE TABLE profit_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID NOT NULL REFERENCES white_labels(id) ON DELETE CASCADE,
    super_admin_wallet VARCHAR(100) NOT NULL,
    amount DECIMAL(30,8) NOT NULL,
    percentage DECIMAL(5,2),
    gross_revenue DECIMAL(30,8),
    net_revenue DECIMAL(30,8),
    token VARCHAR(50) NOT NULL,
    tx_hash VARCHAR(200),
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_profit_tx_wl ON profit_transactions(white_label_id);
CREATE INDEX idx_profit_tx_status ON profit_transactions(status);

-- ============================================================================
-- FEATURE FLAGS
-- ============================================================================

CREATE TABLE feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    global_enabled BOOLEAN DEFAULT FALSE,
    enabled BOOLEAN DEFAULT FALSE,
    master_admin_id UUID REFERENCES admin_users(id) ON DELETE SET NULL,
    white_label_id UUID REFERENCES white_labels(id) ON DELETE CASCADE,
    updated_by UUID REFERENCES admin_users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_feature_name ON feature_flags(name);
CREATE INDEX idx_feature_wl ON feature_flags(white_label_id);

-- ============================================================================
-- FULL FETCHER DATA CACHE
-- ============================================================================

CREATE TABLE fetcher_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fetcher_name VARCHAR(100) NOT NULL,
    key VARCHAR(255) NOT NULL,
    value JSONB NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(fetcher_name, key)
);

CREATE INDEX idx_fetcher_cache_name ON fetcher_cache(fetcher_name);
CREATE INDEX idx_fetcher_cache_expires ON fetcher_cache(expires_at);

-- ============================================================================
-- PRICE FEEDS
-- ============================================================================

CREATE TABLE price_feeds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_symbol VARCHAR(50) NOT NULL,
    token_address VARCHAR(100),
    chain_id VARCHAR(20) NOT NULL,
    price_usd DECIMAL(30,8) NOT NULL,
    change_24h DECIMAL(10,4),
    volume_24h DECIMAL(30,8),
    market_cap DECIMAL(30,8),
    confidence INTEGER,
    source VARCHAR(100),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(token_symbol, chain_id)
);

CREATE INDEX idx_price_token ON price_feeds(token_symbol);
CREATE INDEX idx_price_chain ON price_feeds(chain_id);
CREATE INDEX idx_price_updated ON price_feeds(updated_at DESC);

-- ============================================================================
-- GAS ORACLE DATA
-- ============================================================================

CREATE TABLE gas_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_id VARCHAR(20) NOT NULL,
    gas_price_gwei DECIMAL(20,8),
    gas_limit BIGINT,
    estimated_gas BIGINT,
    max_fee_per_gas DECIMAL(20,8),
    max_priority_fee_per_gas DECIMAL(20,8),
    network_congestion VARCHAR(50),
    source VARCHAR(100),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(chain_id)
);

CREATE INDEX idx_gas_chain ON gas_data(chain_id);

-- ============================================================================
-- TOKEN METADATA
-- ============================================================================

CREATE TABLE token_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_address VARCHAR(100) NOT NULL,
    chain_id VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    decimals INTEGER NOT NULL,
    logo_url TEXT,
    total_supply DECIMAL(30,8),
    is_verified BOOLEAN DEFAULT FALSE,
    is_honeypot BOOLEAN DEFAULT FALSE,
    risk_score INTEGER DEFAULT 0,
    holder_count DECIMAL(20,2),
    transfer_count_24h DECIMAL(20,2),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(token_address, chain_id)
);

CREATE INDEX idx_token_meta_address ON token_metadata(token_address);
CREATE INDEX idx_token_meta_chain ON token_metadata(chain_id);
CREATE INDEX idx_token_meta_verified ON token_metadata(is_verified);

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Update timestamp trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create update triggers for all tables with updated_at
CREATE TRIGGER update_admin_users_updated_at BEFORE UPDATE ON admin_users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_white_labels_updated_at BEFORE UPDATE ON white_labels FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_white_label_admins_updated_at BEFORE UPDATE ON white_label_admins FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_trading_pairs_updated_at BEFORE UPDATE ON trading_pairs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_blockchains_updated_at BEFORE UPDATE ON blockchains FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_fee_structures_updated_at BEFORE UPDATE ON fee_structures FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_audit_logs_updated_at BEFORE UPDATE ON audit_logs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_webhooks_updated_at BEFORE UPDATE ON webhooks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_trading_bots_updated_at BEFORE UPDATE ON trading_bots FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_liquidity_pools_updated_at BEFORE UPDATE ON liquidity_pools FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_token_listing_requests_updated_at BEFORE UPDATE ON token_listing_requests FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_cex_connections_updated_at BEFORE UPDATE ON cex_connections FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_dex_connections_updated_at BEFORE UPDATE ON dex_connections FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_profit_sharing_updated_at BEFORE UPDATE ON profit_sharing FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_fetcher_cache_updated_at BEFORE UPDATE ON fetcher_cache FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_price_feeds_updated_at BEFORE UPDATE ON price_feeds FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_gas_data_updated_at BEFORE UPDATE ON gas_data FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_token_metadata_updated_at BEFORE UPDATE ON token_metadata FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to cleanup expired sessions
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS void AS $$
BEGIN
    DELETE FROM admin_sessions WHERE expires_at < NOW() AND is_valid = TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup expired rate limits
CREATE OR REPLACE FUNCTION cleanup_expired_rate_limits()
RETURNS void AS $$
BEGIN
    DELETE FROM rate_limits WHERE window_start < NOW() - (INTERVAL '1 minute' * window_duration);
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Insert default super admin (password: TigerAdmin2024!Secure)
INSERT INTO admin_users (username, email, password_hash, role, security_level, permissions, status) 
VALUES (
    'superadmin',
    'superadmin@tigerwallet.com',
    '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYNq.vzY/pS',
    'super_admin',
    4,
    '["*"]',
    'active'
);

-- Insert default blockchains
INSERT INTO blockchains (name, symbol, chain_id, chain_type, native_token, decimals, is_active) VALUES
    ('Ethereum', 'ETH', '1', 'evm', 'ETH', 18, TRUE),
    ('BNB Smart Chain', 'BNB', '56', 'evm', 'BNB', 18, TRUE),
    ('Polygon', 'MATIC', '137', 'evm', 'MATIC', 18, TRUE),
    ('Avalanche', 'AVAX', '43114', 'evm', 'AVAX', 18, TRUE),
    ('Arbitrum', 'ETH', '42161', 'evm', 'ETH', 18, TRUE),
    ('Optimism', 'ETH', '10', 'evm', 'ETH', 18, TRUE),
    ('Solana', 'SOL', 'solana', 'solana', 'SOL', 9, TRUE),
    ('Base', 'ETH', '8453', 'evm', 'ETH', 18, TRUE);

-- Insert default products
INSERT INTO products (name, type, description, fee_percent, is_global) VALUES
    ('Spot Trading', 'trading', 'Spot trading with limit and market orders', 0.001, TRUE),
    ('Perpetual Trading', 'trading', 'Perpetual futures trading', 0.0005, TRUE),
    ('Staking', 'staking', 'Stake tokens and earn rewards', 0, TRUE),
    ('NFT Marketplace', 'nft', 'Buy and sell NFTs', 0.025, TRUE),
    ('Bridge', 'bridge', 'Cross-chain token transfers', 0.001, TRUE),
    ('DeFi', 'defi', 'Decentralized finance products', 0.001, TRUE);

-- Insert default feature flags
INSERT INTO feature_flags (name, description, global_enabled) VALUES
    ('trading', 'Enable trading functionality', TRUE),
    ('staking', 'Enable staking functionality', TRUE),
    ('nft', 'Enable NFT features', TRUE),
    ('bridge', 'Enable cross-chain bridges', TRUE),
    ('defi', 'Enable DeFi features', TRUE),
    ('kyc_required', 'Require KYC for withdrawals', FALSE),
    ('whitelist_enabled', 'Enable whitelist only mode', FALSE);

-- Analyze tables for query optimization
ANALYZE admin_users;
ANALYZE white_labels;
ANALYZE users;
ANALYZE transactions;
ANALYZE blockchains;
