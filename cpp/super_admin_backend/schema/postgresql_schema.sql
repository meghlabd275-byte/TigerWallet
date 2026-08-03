-- ============================================================================
-- TigerWallet Super Admin - PostgreSQL Schema
-- Production-ready schema with proper indexing and constraints
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- Admins Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS admins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    role INTEGER NOT NULL DEFAULT 2,
    security_level INTEGER NOT NULL DEFAULT 3,
    permissions JSONB DEFAULT '[]',
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login TIMESTAMP WITH TIME ZONE,
    status INTEGER DEFAULT 1,
    failed_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    ip_whitelist TEXT DEFAULT '',
    password_changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity_at TIMESTAMP WITH TIME ZONE,
    created_by UUID REFERENCES admins(id),
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_admins_username ON admins(username);
CREATE INDEX idx_admins_email ON admins(email);
CREATE INDEX idx_admins_role ON admins(role);
CREATE INDEX idx_admins_status ON admins(status);
CREATE INDEX idx_admins_created_at ON admins(created_at DESC);

-- ============================================================================
-- Admin Sessions Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS admin_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    session_token TEXT UNIQUE NOT NULL,
    refresh_token TEXT,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE,
    device_info JSONB DEFAULT '{}'
);

CREATE INDEX idx_sessions_admin_id ON admin_sessions(admin_id);
CREATE INDEX idx_sessions_token ON admin_sessions(session_token);
CREATE INDEX idx_sessions_expires ON admin_sessions(expires_at);
CREATE INDEX idx_sessions_active ON admin_sessions(is_active);

-- ============================================================================
-- IP Whitelist Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS ip_whitelist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID REFERENCES admins(id) ON DELETE CASCADE,
    ip_address INET NOT NULL,
    cidr INTEGER DEFAULT 32,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES admins(id),
    last_used TIMESTAMP WITH TIME ZONE,
    use_count INTEGER DEFAULT 0,
    UNIQUE(admin_id, ip_address, cidr)
);

CREATE INDEX idx_ip_whitelist_admin ON ip_whitelist(admin_id);
CREATE INDEX idx_ip_whitelist_address ON ip_whitelist(ip_address);

-- ============================================================================
-- White Labels Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS white_labels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    admin_id UUID REFERENCES admins(id),
    brand_name VARCHAR(255),
    brand_logo TEXT,
    brand_color VARCHAR(20),
    brand_tagline TEXT,
    support_email VARCHAR(255),
    website_url VARCHAR(500),
    terms_of_service TEXT,
    privacy_policy TEXT,
    custom_domain VARCHAR(255),
    ssl_enabled BOOLEAN DEFAULT FALSE,
    status INTEGER DEFAULT 1,
    fee_percentage DECIMAL(5,2) DEFAULT 20.00,
    features JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    approved_at TIMESTAMP WITH TIME ZONE,
    approved_by UUID REFERENCES admins(id),
    suspended_at TIMESTAMP WITH TIME ZONE,
    suspended_reason TEXT,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_white_labels_slug ON white_labels(slug);
CREATE INDEX idx_white_labels_admin ON white_labels(admin_id);
CREATE INDEX idx_white_labels_status ON white_labels(status);

-- ============================================================================
-- White Label API Keys Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS wl_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID NOT NULL REFERENCES white_labels(id) ON DELETE CASCADE,
    key_name VARCHAR(100) NOT NULL,
    api_key TEXT UNIQUE NOT NULL,
    api_secret TEXT NOT NULL,
    permissions JSONB DEFAULT '{"read": true, "write": false, "withdraw": false}',
    rate_limit_min INTEGER DEFAULT 60,
    rate_limit_day INTEGER DEFAULT 10000,
    is_active BOOLEAN DEFAULT TRUE,
    last_used TIMESTAMP WITH TIME ZONE,
    use_count INTEGER DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by UUID REFERENCES admins(id)
);

CREATE INDEX idx_wl_api_keys_white_label ON wl_api_keys(white_label_id);
CREATE INDEX idx_wl_api_keys_key ON wl_api_keys(api_key);

-- ============================================================================
-- Audit Logs Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID REFERENCES admins(id),
    white_label_id UUID REFERENCES white_labels(id),
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    status INTEGER DEFAULT 1,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_admin ON audit_logs(admin_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_target ON audit_logs(target_type, target_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_white_label ON audit_logs(white_label_id);

-- ============================================================================
-- Users Table (Platform-wide)
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100),
    wallet_address VARCHAR(100),
    public_key TEXT,
    kyc_level INTEGER DEFAULT 0,
    kyc_status INTEGER DEFAULT 0,
    kyc_submitted_at TIMESTAMP WITH TIME ZONE,
    kyc_approved_at TIMESTAMP WITH TIME ZONE,
    status INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity TIMESTAMP WITH TIME ZONE,
    referrer_id UUID REFERENCES users(id),
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_users_white_label ON users(white_label_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_wallet ON users(wallet_address);
CREATE INDEX idx_users_kyc ON users(kyc_status);
CREATE INDEX idx_users_status ON users(status);

-- ============================================================================
-- User KYC Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_kyc (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kyc_type INTEGER NOT NULL,
    document_type VARCHAR(50),
    document_id VARCHAR(100),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    dob DATE,
    country VARCHAR(2),
    state VARCHAR(100),
    city VARCHAR(100),
    address TEXT,
    zip_code VARCHAR(20),
    status INTEGER DEFAULT 1,
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by UUID REFERENCES admins(id),
    rejection_reason TEXT,
    verification_data JSONB DEFAULT '{}'
);

CREATE INDEX idx_user_kyc_user ON user_kyc(user_id);
CREATE INDEX idx_user_kyc_status ON user_kyc(status);

-- ============================================================================
-- User Balances Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chain_id INTEGER NOT NULL,
    token_address VARCHAR(100),
    balance VARCHAR(100) DEFAULT '0',
    reserved_balance VARCHAR(100) DEFAULT '0',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, chain_id, token_address)
);

CREATE INDEX idx_balances_user ON user_balances(user_id);
CREATE INDEX idx_balances_chain ON user_balances(chain_id);

-- ============================================================================
-- Transactions Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    white_label_id UUID REFERENCES white_labels(id),
    tx_hash VARCHAR(200) UNIQUE NOT NULL,
    from_address VARCHAR(100) NOT NULL,
    to_address VARCHAR(100) NOT NULL,
    chain_id INTEGER NOT NULL,
    token_address VARCHAR(100),
    amount VARCHAR(100) NOT NULL,
    fee VARCHAR(100),
    status INTEGER DEFAULT 1,
    type INTEGER NOT NULL,
    block_number BIGINT,
    confirmations INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    confirmed_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_transactions_user ON transactions(user_id);
CREATE INDEX idx_transactions_hash ON transactions(tx_hash);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_transactions_chain ON transactions(chain_id);
CREATE INDEX idx_transactions_created ON transactions(created_at DESC);
CREATE INDEX idx_transactions_white_label ON transactions(white_label_id);

-- ============================================================================
-- Trading Pairs Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS trading_pairs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id),
    base_token VARCHAR(50) NOT NULL,
    quote_token VARCHAR(50) NOT NULL,
    pair_name VARCHAR(50) UNIQUE NOT NULL,
    pair_address VARCHAR(100),
    status INTEGER DEFAULT 1,
    min_trade_amount VARCHAR(50) DEFAULT '0',
    max_trade_amount VARCHAR(50),
    fee_percentage DECIMAL(10,4) DEFAULT 0.3,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,
    created_by UUID REFERENCES admins(id)
);

CREATE INDEX idx_pairs_white_label ON trading_pairs(white_label_id);
CREATE INDEX idx_pairs_status ON trading_pairs(status);

-- ============================================================================
-- Liquidity Pools Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS liquidity_pools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pair_id UUID NOT NULL REFERENCES trading_pairs(id),
    provider_address VARCHAR(100) NOT NULL,
    liquidity_token_amount VARCHAR(100) NOT NULL,
    base_token_amount VARCHAR(100) NOT NULL,
    quote_token_amount VARCHAR(100) NOT NULL,
    apr DECIMAL(10,4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_pools_pair ON liquidity_pools(pair_id);
CREATE INDEX idx_pools_provider ON liquidity_pools(provider_address);

-- ============================================================================
-- Blockchains Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS blockchains (
    id INTEGER PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    symbol VARCHAR(10) NOT NULL,
    type VARCHAR(20) NOT NULL,
    chain_id INTEGER,
    rpc_url TEXT,
    rpc_urls JSONB DEFAULT '[]',
    explorer_url TEXT,
    explorer_urls JSONB DEFAULT '[]',
    wss_url TEXT,
    is_testnet BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    confirmations INTEGER DEFAULT 12,
    decimals INTEGER DEFAULT 18,
    native_currency VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_blockchains_type ON blockchains(type);
CREATE INDEX idx_blockchains_active ON blockchains(is_active);

-- ============================================================================
-- Fee Structures Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS fee_structures (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id),
    name VARCHAR(100) NOT NULL,
    fee_type VARCHAR(20) NOT NULL,
    token_address VARCHAR(100),
    chain_id INTEGER,
    percentage DECIMAL(10,4) DEFAULT 0,
    flat_fee VARCHAR(50) DEFAULT '0',
    min_fee VARCHAR(50) DEFAULT '0',
    max_fee VARCHAR(50),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,
    created_by UUID REFERENCES admins(id)
);

CREATE INDEX idx_fees_white_label ON fee_structures(white_label_id);
CREATE INDEX idx_fees_type ON fee_structures(fee_type);

-- ============================================================================
-- Trading Bots Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS trading_bots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    white_label_id UUID REFERENCES white_labels(id),
    bot_type VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    tier VARCHAR(20) NOT NULL,
    status INTEGER DEFAULT 1,
    config JSONB NOT NULL DEFAULT '{}',
    pnl DECIMAL(20,8) DEFAULT 0,
    volume_24h DECIMAL(20,8) DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    paused_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_bots_user ON trading_bots(user_id);
CREATE INDEX idx_bots_status ON trading_bots(status);
CREATE INDEX idx_bots_type ON trading_bots(bot_type);
CREATE INDEX idx_bots_white_label ON trading_bots(white_label_id);

-- ============================================================================
-- CEX Connections Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS cex_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id),
    exchange_name VARCHAR(50) NOT NULL,
    api_key TEXT NOT NULL,
    api_secret TEXT NOT NULL,
    passphrase TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    last_sync TIMESTAMP WITH TIME ZONE,
    sync_status INTEGER DEFAULT 1,
    permissions JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_cex_white_label ON cex_connections(white_label_id);

-- ============================================================================
-- DEX Connections Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS dex_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id),
    dex_name VARCHAR(50) NOT NULL,
    router_address VARCHAR(100) NOT NULL,
    factory_address VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    supported_pairs JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_dex_white_label ON dex_connections(white_label_id);

-- ============================================================================
-- Token Listing Requests Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS token_listing_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    requester_id UUID NOT NULL REFERENCES users(id),
    white_label_id UUID REFERENCES white_labels(id),
    token_name VARCHAR(100) NOT NULL,
    token_symbol VARCHAR(20) NOT NULL,
    token_address VARCHAR(100) NOT NULL,
    chain_id INTEGER NOT NULL,
    decimals INTEGER,
    total_supply VARCHAR(100),
    tier VARCHAR(20) NOT NULL,
    one_time_fee VARCHAR(50) DEFAULT '0',
    monthly_fee VARCHAR(50) DEFAULT '0',
    status INTEGER DEFAULT 1,
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by UUID REFERENCES admins(id),
    rejection_reason TEXT,
    listing_data JSONB DEFAULT '{}'
);

CREATE INDEX idx_token_requests_requester ON token_listing_requests(requester_id);
CREATE INDEX idx_token_requests_status ON token_listing_requests(status);
CREATE INDEX idx_token_requests_white_label ON token_listing_requests(white_label_id);

-- ============================================================================
-- User API Keys Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_name VARCHAR(100) NOT NULL,
    api_key TEXT UNIQUE NOT NULL,
    api_secret TEXT NOT NULL,
    permissions JSONB DEFAULT '{"read": true, "trade": false, "withdraw": false}',
    rate_limit_min INTEGER DEFAULT 60,
    rate_limit_day INTEGER DEFAULT 10000,
    is_active BOOLEAN DEFAULT TRUE,
    last_used TIMESTAMP WITH TIME ZONE,
    use_count INTEGER DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_user_api_keys_user ON user_api_keys(user_id);
CREATE INDEX idx_user_api_keys_key ON user_api_keys(api_key);

-- ============================================================================
-- Webhooks Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id),
    url TEXT NOT NULL,
    events JSONB NOT NULL DEFAULT '[]',
    secret TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    last_triggered TIMESTAMP WITH TIME ZONE,
    failure_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES admins(id)
);

CREATE INDEX idx_webhooks_white_label ON webhooks(white_label_id);

-- ============================================================================
-- Notifications Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    admin_id UUID REFERENCES admins(id),
    type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    data JSONB DEFAULT '{}',
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_admin ON notifications(admin_id);
CREATE INDEX idx_notifications_read ON notifications(is_read);
CREATE INDEX idx_notifications_created ON notifications(created_at DESC);

-- ============================================================================
-- Rate Limits Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS rate_limits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    identifier VARCHAR(100) NOT NULL,
    identifier_type VARCHAR(20) NOT NULL,
    requests_count INTEGER DEFAULT 0,
    reset_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(identifier, identifier_type, reset_at)
);

CREATE INDEX idx_rate_limits_identifier ON rate_limits(identifier);
CREATE INDEX idx_rate_limits_reset ON rate_limits(reset_at);

-- ============================================================================
-- Functions and Triggers
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_admins_updated_at BEFORE UPDATE ON admins FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_white_labels_updated_at BEFORE UPDATE ON white_labels FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_trading_bots_updated_at BEFORE UPDATE ON trading_bots FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS void AS $$
BEGIN
    DELETE FROM admin_sessions WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Default Super Admin
-- ============================================================================

INSERT INTO admins (
    username, 
    password_hash, 
    email, 
    role, 
    security_level,
    permissions,
    status,
    created_at
) VALUES (
    'superadmin',
    crypt('SuperAdmin@2024!', gen_salt('bf', 10)),
    'superadmin@tigerwallet.com',
    1,
    3,
    '[
        {"permission": "master_wallet_creation", "access": true},
        {"permission": "multi_blockchain", "access": true},
        {"permission": "token_management", "access": true},
        {"permission": "user_wallet_ownership", "access": true},
        {"permission": "hd_wallet", "access": true},
        {"permission": "biometric_auth", "access": true},
        {"permission": "pin_code_auth", "access": true},
        {"permission": "nft_support", "access": true},
        {"permission": "defi_integration", "access": true},
        {"permission": "staking", "access": true},
        {"permission": "bridge_support", "access": true},
        {"permission": "mev_protection", "access": true},
        {"permission": "swap_trading", "access": true},
        {"permission": "hardware_wallet", "access": true},
        {"permission": "admin_controls", "access": true},
        {"permission": "network_management", "access": true},
        {"permission": "gas_optimization", "access": true},
        {"permission": "multi_sig", "access": true},
        {"permission": "transaction_history", "access": true},
        {"permission": "price_alerts", "access": true},
        {"permission": "privacy_zk", "access": true},
        {"permission": "coinjoin", "access": true},
        {"permission": "account_abstraction", "access": true},
        {"permission": "session_keys", "access": true},
        {"permission": "paymaster", "access": true},
        {"permission": "passkeys", "access": true},
        {"permission": "tax_integration", "access": true},
        {"permission": "analytics", "access": true},
        {"permission": "cross_chain_intent", "access": true},
        {"permission": "dapp_browser", "access": true}
    ]'::jsonb,
    1,
    NOW()
) ON CONFLICT (username) DO NOTHING;

-- ============================================================================
-- Default Blockchains
-- ============================================================================

INSERT INTO blockchains (id, name, symbol, type, chain_id, is_testnet, is_active, decimals, native_currency) VALUES
(1, 'Ethereum', 'ETH', 'evm', 1, false, true, 18, 'ETH'),
(56, 'BNB Chain', 'BNB', 'evm', 56, false, true, 18, 'BNB'),
(137, 'Polygon', 'MATIC', 'evm', 137, false, true, 18, 'MATIC'),
(42161, 'Arbitrum One', 'ETH', 'evm', 42161, false, true, 18, 'ETH'),
(10, 'Optimism', 'ETH', 'evm', 10, false, true, 18, 'ETH'),
(43114, 'Avalanche', 'AVAX', 'evm', 43114, false, true, 18, 'AVAX'),
(5, 'Goerli Testnet', 'ETH', 'evm', 5, true, true, 18, 'ETH'),
(97, 'BSC Testnet', 'BNB', 'evm', 97, true, true, 18, 'BNB')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- Default Fee Structures
-- ============================================================================

INSERT INTO fee_structures (name, fee_type, percentage, flat_fee, is_active) VALUES
('Standard Withdrawal', 'withdraw', 0.001, '5', true),
('Standard Swap', 'swap', 0.003, '0', true),
('Standard Deposit', 'deposit', 0, '0', true),
('Standard Trade', 'trade', 0.003, '0', true)
ON CONFLICT DO NOTHING;
