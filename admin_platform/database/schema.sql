-- TigerWallet Admin Platform Database Schema
-- PostgreSQL Database Schema for Super Admin & RBAC Admin Panel

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

-- ============================================================================
-- ENUMS
-- ============================================================================

CREATE TYPE admin_role AS ENUM ('super_admin', 'admin', 'manager', 'support');
CREATE TYPE admin_status AS ENUM ('active', 'suspended', 'blocked', 'pending');
CREATE TYPE wl_status AS ENUM ('pending', 'active', 'suspended', 'revoked', 'destroyed');
CREATE TYPE user_kyc_status AS ENUM ('none', 'pending', 'under_review', 'approved', 'rejected');
CREATE TYPE transaction_type AS ENUM ('deposit', 'withdrawal', 'transfer', 'swap', 'trade', 'fee', 'reward', 'other');
CREATE TYPE transaction_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');
CREATE TYPE pair_status AS ENUM ('active', 'suspended', 'halted', 'delisted');
CREATE TYPE fee_type AS ENUM ('deposit', 'withdrawal', 'trading', 'api', 'network');

-- ============================================================================
-- ADMIN USERS TABLE
-- ============================================================================

CREATE TABLE admin_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email CITEXT UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role admin_role NOT NULL DEFAULT 'admin',
    security_level INTEGER NOT NULL DEFAULT 1 CHECK (security_level BETWEEN 1 AND 4),
    permissions JSONB DEFAULT '[]'::jsonb,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret VARCHAR(255),
    backup_codes JSONB DEFAULT '[]'::jsonb,
    status admin_status NOT NULL DEFAULT 'active',
    failed_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    last_login TIMESTAMP,
    last_ip INET,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_username ON admin_users(username);
CREATE INDEX idx_admin_email ON admin_users(email);
CREATE INDEX idx_admin_role ON admin_users(role);
CREATE INDEX idx_admin_status ON admin_users(status);

-- ============================================================================
-- SESSIONS TABLE
-- ============================================================================

CREATE TABLE admin_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_activity TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_token ON admin_sessions(token);
CREATE INDEX idx_session_admin ON admin_sessions(admin_id);
CREATE INDEX idx_session_expires ON admin_sessions(expires_at);

-- ============================================================================
-- IP WHITELIST TABLE
-- ============================================================================

CREATE TABLE ip_whitelist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    ip_address CIDR NOT NULL,
    description VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ip_whitelist_admin ON ip_whitelist(admin_id);

-- ============================================================================
-- WHITE LABELS TABLE
-- ============================================================================

CREATE TABLE white_labels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE NOT NULL,
    api_key_hash VARCHAR(255) NOT NULL,
    api_secret_hash VARCHAR(255),
    fee_percent DECIMAL(5,2) NOT NULL DEFAULT 20.00 CHECK (fee_percent BETWEEN 0 AND 20),
    profit_share_percent DECIMAL(5,2) NOT NULL DEFAULT 0 CHECK (profit_share_percent BETWEEN 0 AND 50),
    profit_share_schedule VARCHAR(50) DEFAULT 'monthly',
    status wl_status NOT NULL DEFAULT 'pending',
    custom_branding BOOLEAN DEFAULT TRUE,
    branding_config JSONB DEFAULT '{}'::jsonb,
    features JSONB DEFAULT '[]'::jsonb,
    approved_by UUID REFERENCES admin_users(id),
    approved_at TIMESTAMP,
    created_by UUID REFERENCES admin_users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wl_domain ON white_labels(domain);
CREATE INDEX idx_wl_status ON white_labels(status);

-- ============================================================================
-- WHITE LABEL API KEYS
-- ============================================================================

CREATE TABLE wl_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID NOT NULL REFERENCES white_labels(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    secret_hash VARCHAR(255),
    permissions JSONB DEFAULT '[]'::jsonb,
    rate_limit_minute INTEGER DEFAULT 60,
    rate_limit_day INTEGER DEFAULT 10000,
    is_active BOOLEAN DEFAULT TRUE,
    last_used TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wl_api_keys_wl ON wl_api_keys(white_label_id);

-- ============================================================================
-- AUDIT LOGS TABLE
-- ============================================================================

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID REFERENCES admin_users(id) ON DELETE SET NULL,
    white_label_id UUID REFERENCES white_labels(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    details JSONB DEFAULT '{}'::jsonb,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_admin ON audit_logs(admin_id);
CREATE INDEX idx_audit_action ON audit_logs(action);
CREATE INDEX idx_audit_created ON audit_logs(created_at DESC);

-- ============================================================================
-- USERS TABLE (for RBAC Admin)
-- ============================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email CITEXT UNIQUE NOT NULL,
    wallet_address VARCHAR(100),
    username VARCHAR(100),
    kyc_status user_kyc_status DEFAULT 'none',
    kyc_level INTEGER DEFAULT 0 CHECK (kyc_level BETWEEN 0 AND 3),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    risk_score INTEGER DEFAULT 0 CHECK (risk_score BETWEEN 0 AND 100),
    tags JSONB DEFAULT '[]'::jsonb,
    referrer_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_kyc ON users(kyc_status);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_wallet ON users(wallet_address);

-- ============================================================================
-- USER KYC TABLE
-- ============================================================================

CREATE TABLE user_kyc (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kyc_type VARCHAR(50) NOT NULL,
    document_type VARCHAR(50),
    document_id VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    date_of_birth DATE,
    nationality VARCHAR(3),
    address JSONB DEFAULT '{}'::jsonb,
    documents JSONB DEFAULT '[]'::jsonb,
    status user_kyc_status DEFAULT 'pending',
    rejection_reason TEXT,
    reviewed_by UUID REFERENCES admin_users(id),
    reviewed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kyc_user ON user_kyc(user_id);
CREATE INDEX idx_kyc_status ON user_kyc(status);

-- ============================================================================
-- USER BALANCES TABLE
-- ============================================================================

CREATE TABLE user_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chain_id UUID NOT NULL,
    token_address VARCHAR(100),
    balance NUMERIC(78, 0) DEFAULT 0,
    locked_balance NUMERIC(78, 0) DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_balances_user_chain ON user_balances(user_id, chain_id, token_address);

-- ============================================================================
-- TRANSACTIONS TABLE
-- ============================================================================

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    type transaction_type NOT NULL,
    status transaction_status NOT NULL DEFAULT 'pending',
    chain_id UUID NOT NULL,
    token_address VARCHAR(100),
    amount NUMERIC(78, 0) NOT NULL,
    fee NUMERIC(78, 0) DEFAULT 0,
    from_address VARCHAR(100),
    to_address VARCHAR(100),
    tx_hash VARCHAR(200),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tx_user ON transactions(user_id);
CREATE INDEX idx_tx_type ON transactions(type);
CREATE INDEX idx_tx_status ON transactions(status);
CREATE INDEX idx_tx_chain ON transactions(chain_id);
CREATE INDEX idx_tx_created ON transactions(created_at DESC);
CREATE INDEX idx_tx_hash ON transactions(tx_hash) WHERE tx_hash IS NOT NULL;

-- ============================================================================
-- TRADING PAIRS TABLE
-- ============================================================================

CREATE TABLE trading_pairs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL,
    base_token VARCHAR(50) NOT NULL,
    quote_token VARCHAR(50) NOT NULL,
    chain_id UUID NOT NULL,
    dex_id UUID,
    pair_address VARCHAR(100),
    status pair_status DEFAULT 'active',
    maker_fee DECIMAL(10, 6) DEFAULT 0,
    taker_fee DECIMAL(10, 6) DEFAULT 0,
    min_trade_amount NUMERIC(78, 0) DEFAULT 0,
    max_trade_amount NUMERIC(78, 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_pair_name_chain ON trading_pairs(name, chain_id);

-- ============================================================================
-- LIQUIDITY POOLS TABLE
-- ============================================================================

CREATE TABLE liquidity_pools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pair_id UUID NOT NULL REFERENCES trading_pairs(id),
    provider_address VARCHAR(100) NOT NULL,
    liquidity_tokens NUMERIC(78, 0) DEFAULT 0,
    reserve_a NUMERIC(78, 0) DEFAULT 0,
    reserve_b NUMERIC(78, 0) DEFAULT 0,
    apr DECIMAL(10, 4) DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lp_pair ON liquidity_pools(pair_id);
CREATE INDEX idx_lp_provider ON liquidity_pools(provider_address);

-- ============================================================================
-- BLOCKCHAINS TABLE
-- ============================================================================

CREATE TABLE blockchains (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    chain_type VARCHAR(20) NOT NULL,
    chain_id INTEGER,
    rpc_urls JSONB DEFAULT '[]'::jsonb,
    explorer_urls JSONB DEFAULT '[]'::jsonb,
    is_active BOOLEAN DEFAULT TRUE,
    is_maintenance BOOLEAN DEFAULT FALSE,
    native_token JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_blockchain_type ON blockchains(chain_type);
CREATE INDEX idx_blockchain_active ON blockchains(is_active);

-- ============================================================================
-- FEE STRUCTURES TABLE
-- ============================================================================

CREATE TABLE fee_structures (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    fee_type fee_type NOT NULL,
    chain_id UUID REFERENCES blockchains(id),
    token_address VARCHAR(100),
    maker_fee DECIMAL(10, 6) DEFAULT 0,
    taker_fee DECIMAL(10, 6) DEFAULT 0,
    fixed_fee NUMERIC(78, 0) DEFAULT 0,
    min_fee NUMERIC(78, 0) DEFAULT 0,
    max_fee NUMERIC(78, 0),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fee_type ON fee_structures(fee_type);
CREATE INDEX idx_fee_chain ON fee_structures(chain_id);

-- ============================================================================
-- TRADING BOTS TABLE
-- ============================================================================

CREATE TABLE trading_bots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    bot_type VARCHAR(50) NOT NULL,
    tier VARCHAR(20) DEFAULT 'basic',
    status VARCHAR(20) DEFAULT 'inactive',
    config JSONB DEFAULT '{}'::jsonb,
    pnl DECIMAL(78, 0) DEFAULT 0,
    volume_24h DECIMAL(78, 0) DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bot_user ON trading_bots(user_id);
CREATE INDEX idx_bot_status ON trading_bots(status);

-- ============================================================================
-- CEX CONNECTIONS TABLE
-- ============================================================================

CREATE TABLE cex_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    exchange VARCHAR(50) NOT NULL,
    api_key_hash VARCHAR(255),
    secret_hash VARCHAR(255),
    status VARCHAR(20) DEFAULT 'disconnected',
    can_trade BOOLEAN DEFAULT FALSE,
    can_withdraw BOOLEAN DEFAULT FALSE,
    sync_status JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- DEX CONNECTIONS TABLE
-- ============================================================================

CREATE TABLE dex_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    dex_name VARCHAR(50) NOT NULL,
    chain_id UUID NOT NULL,
    router_address VARCHAR(100),
    factory_address VARCHAR(100),
    status VARCHAR(20) DEFAULT 'active',
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dex_chain ON dex_connections(chain_id);

-- ============================================================================
-- TOKEN LISTING REQUESTS TABLE
-- ============================================================================

CREATE TABLE token_listing_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_name VARCHAR(100) NOT NULL,
    token_symbol VARCHAR(20) NOT NULL,
    token_address VARCHAR(100) NOT NULL,
    chain_id UUID NOT NULL,
    requester_id UUID NOT NULL REFERENCES users(id),
    tier VARCHAR(20) DEFAULT 'basic',
    status VARCHAR(20) DEFAULT 'pending',
    one_time_fee NUMERIC(78, 0) DEFAULT 0,
    monthly_fee NUMERIC(78, 0) DEFAULT 0,
    rejection_reason TEXT,
    reviewed_by UUID REFERENCES admin_users(id),
    reviewed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_token_req_status ON token_listing_requests(status);
CREATE INDEX idx_token_req_requester ON token_listing_requests(requester_id);

-- ============================================================================
-- USER API KEYS TABLE
-- ============================================================================

CREATE TABLE user_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    secret_hash VARCHAR(255),
    permissions JSONB DEFAULT '[]'::jsonb,
    rate_limit_minute INTEGER DEFAULT 10,
    rate_limit_day INTEGER DEFAULT 1000,
    tier VARCHAR(20) DEFAULT 'free',
    is_active BOOLEAN DEFAULT TRUE,
    last_used TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_api_keys_user ON user_api_keys(user_id);

-- ============================================================================
-- WEBHOOKS TABLE
-- ============================================================================

CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    white_label_id UUID REFERENCES white_labels(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    events JSONB DEFAULT '[]'::jsonb,
    secret_hash VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- NOTIFICATIONS TABLE
-- ============================================================================

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    admin_id UUID REFERENCES admin_users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_admin ON notifications(admin_id);
CREATE INDEX idx_notifications_created ON notifications(created_at DESC);

-- ============================================================================
-- RATE LIMITS TABLE
-- ============================================================================

CREATE TABLE rate_limits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    identifier VARCHAR(255) NOT NULL,
    identifier_type VARCHAR(20) NOT NULL,
    request_count INTEGER DEFAULT 0,
    window_start TIMESTAMP NOT NULL,
    window_duration INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rate_limits_identifier ON rate_limits(identifier, identifier_type, window_start);

-- ============================================================================
-- FUNCTION: UPDATE TIMESTAMP
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ============================================================================
-- TRIGGERS: UPDATE TIMESTAMP
-- ============================================================================

CREATE TRIGGER update_admin_users_updated_at BEFORE UPDATE ON admin_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_white_labels_updated_at BEFORE UPDATE ON white_labels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_kyc_updated_at BEFORE UPDATE ON user_kyc
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_trading_pairs_updated_at BEFORE UPDATE ON trading_pairs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_liquidity_pools_updated_at BEFORE UPDATE ON liquidity_pools
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_blockchains_updated_at BEFORE UPDATE ON blockchains
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_fee_structures_updated_at BEFORE UPDATE ON fee_structures
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_trading_bots_updated_at BEFORE UPDATE ON trading_bots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SEED DATA: SUPER ADMIN
-- ============================================================================

INSERT INTO admin_users (username, email, password_hash, role, security_level, permissions, status)
VALUES (
    'tigerwallet_admin',
    'admin@tigerwallet.com',
    crypt('TigerWallet2024!Admin', gen_salt('bf')),
    'super_admin',
    4,
    '["*"]'::jsonb,
    'active'
) ON CONFLICT (username) DO NOTHING;

-- ============================================================================
-- SEED DATA: BLOCKCHAINS
-- ============================================================================

INSERT INTO blockchains (name, symbol, chain_type, chain_id, rpc_urls, explorer_urls, is_active) VALUES
('Ethereum', 'ETH', 'evm', 1, '["https://eth.llamarpc.com","https://ethereum.publicnode.com"]', '["https://etherscan.io"]', TRUE),
('BNB Smart Chain', 'BNB', 'evm', 56, '["https://bsc-dataseed.binance.org","https://bsc.publicnode.com"]', '["https://bscscan.com"]', TRUE),
('Polygon', 'MATIC', 'evm', 137, '["https://polygon-rpc.com","https://polygon.publicnode.com"]', '["https://polygonscan.com"]', TRUE),
('Avalanche', 'AVAX', 'evm', 43114, '["https://api.avax.network","https://avalanche.publicnode.com"]', '["https://snowtrace.io"]', TRUE),
('Arbitrum', 'ARB', 'evm', 42161, '["https://arb1.arbitrum.io","https://arbitrum.publicnode.com"]', '["https://arbiscan.io"]', TRUE),
('Optimism', 'OP', 'evm', 10, '["https://mainnet.optimism.io","https://optimism.publicnode.com"]', '["https://optimistic.etherscan.io"]', TRUE),
('Solana', 'SOL', 'solana', NULL, '["https://api.mainnet-beta.solana.com","https://solana.publicnode.com"]', '["https://explorer.solana.com"]', TRUE),
('Aptos', 'APT', 'aptos', NULL, '["https://fullnode.mainnet.aptoslabs.com","https://aptos-mainnet.public.blastnode.io"]', '["https://aptoscan.com"]', TRUE),
('Sui', 'SUI', 'sui', NULL, '["https://rpc.mainnet.sui.io","https://sui-mainnet.public.blastnode.io"]', '["https://suiscan.xyz"]', TRUE),
('TON', 'TON', 'ton', NULL, '["https://toncenter.com/api/v2"]', '["https://tonscan.org"]', TRUE),
('Base', 'BASE', 'evm', 8453, '["https://mainnet.base.org","https://base.publicnode.com"]', '["https://basescan.org"]', TRUE),
('Linea', 'LINEA', 'evm', 59144, '["https://rpc.linea.build","https://linea.publicnode.com"]', '["https://lineascan.build"]', TRUE),
('Scroll', 'SCROLL', 'evm', 534352, '["https://rpc.scroll.io","https://scroll.publicnode.com"]', '["https://scrollscan.com"]', TRUE),
('zkSync Era', 'ZK', 'evm', 324, '["https://zksync-era.blockchain.io","https://zksync-era.publicnode.com"]', '["https://explorer.zksync.io"]', TRUE),
('Polkadot', 'DOT', 'polkadot', NULL, '["https://rpc.polkadot.io"]', '["https://polkadot.subscan.io"]', TRUE);

-- ============================================================================
-- SEED DATA: FEE STRUCTURES
-- ============================================================================

INSERT INTO fee_structures (name, fee_type, maker_fee, taker_fee, is_active) VALUES
('Default Trading Fee', 'trading', 0.001, 0.003, TRUE),
('Default Deposit Fee', 'deposit', 0, 0, TRUE),
('Default Withdrawal Fee', 'withdrawal', 0, 0, TRUE),
('API Trading Fee', 'api', 0.0005, 0.001, TRUE);
