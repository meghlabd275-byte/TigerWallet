-- ============================================================================
-- TigerWallet MasterWallet Production Database Schema
-- PostgreSQL 14+ Optimized for Ultra-Low Latency
-- ============================================================================

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";
CREATE EXTENSION IF NOT EXISTS "hstore";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- ============================================================================
-- ENUM TYPES
-- ============================================================================

CREATE TYPE blockchain_type AS ENUM (
    'bitcoin', 'ethereum', 'polygon', 'bsc', 'avalanche', 'solana',
    'aptos', 'sui', 'near', 'algorand', 'cardano', 'injective',
    'sei', 'starknet', 'substrate', 'zksync'
);

CREATE TYPE wallet_type AS ENUM (
    'hot', 'warm', 'cold', 'treasury', 'operational'
);

CREATE TYPE transaction_status AS ENUM (
    'pending', 'confirmed', 'failed', 'cancelled', 'on_hold'
);

CREATE TYPE signature_status AS ENUM (
    'pending', 'signed', 'rejected', 'expired'
);

CREATE TYPE approval_level AS ENUM (
    'none', 'view', 'initiate', 'approve', 'admin'
);

-- ============================================================================
-- MASTER WALLETS TABLE
-- ============================================================================

CREATE TABLE master_wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    blockchain blockchain_type NOT NULL,
    address VARCHAR(255) NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    wallet_type wallet_type NOT NULL DEFAULT 'hot',
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_multi_sig BOOLEAN NOT NULL DEFAULT false,
    threshold INTEGER NOT NULL DEFAULT 1,
    total_signers INTEGER NOT NULL DEFAULT 1,
    
    -- Security
    encryption_key_hash TEXT NOT NULL,
    hsm_enabled BOOLEAN NOT NULL DEFAULT false,
    hsm_config JSONB,
    
    -- Limits
    daily_limit NUMERIC(78, 0) NOT NULL DEFAULT 0,
    per_transaction_limit NUMERIC(78, 0) NOT NULL DEFAULT 0,
    monthly_volume NUMERIC(78, 0) NOT NULL DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT positive_daily_limit CHECK (daily_limit >= 0),
    CONSTRAINT positive_per_tx_limit CHECK (per_transaction_limit >= 0),
    CONSTRAINT valid_threshold CHECK (threshold >= 1 AND threshold <= total_signers)
);

CREATE INDEX idx_master_wallets_address ON master_wallets(blockchain, address);
CREATE INDEX idx_master_wallets_blockchain ON master_wallets(blockchain);
CREATE INDEX idx_master_wallets_type ON master_wallets(wallet_type);
CREATE INDEX idx_master_wallets_active ON master_wallets(is_active);
CREATE INDEX idx_master_wallets_created ON master_wallets(created_at DESC);

-- ============================================================================
-- SUB WALLETS TABLE (User Wallets Owned by Master)
-- ============================================================================

CREATE TABLE sub_wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
    
    -- Address derivation
    derivation_path VARCHAR(100) NOT NULL,
    derivation_index INTEGER NOT NULL,
    blockchain blockchain_type NOT NULL,
    address VARCHAR(255) NOT NULL,
    public_key TEXT NOT NULL,
    private_key_encrypted TEXT NOT NULL,
    
    -- User association
    user_id UUID,
    user_wallet_id UUID,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    
    -- Balances
    balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
    pending_balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
    locked_balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
    
    -- Metadata
    label VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_address_per_master UNIQUE (master_wallet_id, blockchain, address),
    CONSTRAINT valid_derivation_index CHECK (derivation_index >= 0)
);

CREATE INDEX idx_sub_wallets_master ON sub_wallets(master_wallet_id);
CREATE INDEX idx_sub_wallets_address ON sub_wallets(blockchain, address);
CREATE INDEX idx_sub_wallets_user ON sub_wallets(user_id, user_wallet_id);
CREATE INDEX idx_sub_wallets_derivation ON sub_wallets(master_wallet_id, blockchain, derivation_index);
CREATE INDEX idx_sub_wallets_active ON sub_wallets(is_active);

-- ============================================================================
-- SIGNERS TABLE (Multi-Sig)
-- ============================================================================

CREATE TABLE signers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
    
    -- Signer identity
    signer_type VARCHAR(50) NOT NULL, -- 'internal', 'external', 'hardware', 'api'
    signer_id VARCHAR(255) NOT NULL,
    signer_name VARCHAR(255) NOT NULL,
    public_key TEXT NOT NULL,
    
    -- Authentication
    auth_method VARCHAR(50) NOT NULL DEFAULT 'password', -- password, biometric, hardware, api_key
    credentials_encrypted TEXT,
    
    -- Permissions
    approval_level approval_level NOT NULL DEFAULT 'view',
    can_initiate BOOLEAN NOT NULL DEFAULT false,
    can_approve BOOLEAN NOT NULL DEFAULT false,
    can_reject BOOLEAN NOT NULL DEFAULT true,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_required BOOLEAN NOT NULL DEFAULT false,
    order_index INTEGER NOT NULL DEFAULT 0,
    
    -- Limits
    daily_approval_limit NUMERIC(78, 0) NOT NULL DEFAULT 0,
    per_tx_limit NUMERIC(78, 0) NOT NULL DEFAULT 0,
    
    -- 2FA
    two_factor_enabled BOOLEAN NOT NULL DEFAULT false,
    two_factor_secret TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signers_master ON signers(master_wallet_id);
CREATE INDEX idx_signers_signer_id ON signers(signer_id);
CREATE INDEX idx_signers_active ON signers(is_active);
CREATE INDEX idx_signers_order ON signers(master_wallet_id, order_index);

-- ============================================================================
-- TRANSACTIONS TABLE
-- ============================================================================

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE RESTRICT,
    sub_wallet_id UUID REFERENCES sub_wallets(id) ON DELETE SET NULL,
    
    -- Transaction details
    tx_hash VARCHAR(255),
    tx_type VARCHAR(50) NOT NULL, -- transfer, swap, stake, bridge, batch
    status transaction_status NOT NULL DEFAULT 'pending',
    
    -- Blockchain
    blockchain blockchain_type NOT NULL,
    from_address VARCHAR(255) NOT NULL,
    to_address VARCHAR(255) NOT NULL,
    
    -- Amounts
    amount NUMERIC(78, 0) NOT NULL,
    fee_amount NUMERIC(78, 0) NOT NULL DEFAULT 0,
    fee_token VARCHAR(50) NOT NULL DEFAULT 'native',
    gas_limit BIGINT,
    gas_price BIGINT,
    gas_used BIGINT,
    
    -- Token
    token_address VARCHAR(255),
    token_symbol VARCHAR(50),
    token_decimals INTEGER,
    
    -- Data
    raw_data BYTEA,
    call_data JSONB,
    
    -- Timing
    nonce BIGINT,
    chain_id BIGINT,
    expiration_time TIMESTAMPTZ,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    
    -- Constraints
    CONSTRAINT positive_amount CHECK (amount > 0),
    CONSTRAINT positive_fee CHECK (fee_amount >= 0)
);

CREATE INDEX idx_transactions_master ON transactions(master_wallet_id);
CREATE INDEX idx_transactions_sub ON transactions(sub_wallet_id);
CREATE INDEX idx_transactions_hash ON transactions(tx_hash) WHERE tx_hash IS NOT NULL;
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_blockchain ON transactions(blockchain);
CREATE INDEX idx_transactions_from ON transactions(from_address);
CREATE INDEX idx_transactions_to ON transactions(to_address);
CREATE INDEX idx_transactions_created ON transactions(created_at DESC);
CREATE INDEX idx_transactions_type ON transactions(tx_type);

-- ============================================================================
-- TRANSACTION SIGNATURES TABLE (Multi-Sig)
-- ============================================================================

CREATE TABLE transaction_signatures (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    signer_id UUID NOT NULL REFERENCES signers(id) ON DELETE RESTRICT,
    
    -- Signature
    signature BYTEA,
    signature_hex TEXT,
    signature_status signature_status NOT NULL DEFAULT 'pending',
    
    -- Approval
    approved_at TIMESTAMPTZ,
    approved_ip VARCHAR(45),
    approved_user_agent TEXT,
    
    -- Rejection
    rejected_at TIMESTAMPTZ,
    rejection_reason TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_signer_transaction UNIQUE (transaction_id, signer_id)
);

CREATE INDEX idx_tx_signatures_tx ON transaction_signatures(transaction_id);
CREATE INDEX idx_tx_signatures_signer ON transaction_signatures(signer_id);
CREATE INDEX idx_tx_signatures_status ON transaction_signatures(signature_status);

-- ============================================================================
-- APPROVAL REQUESTS TABLE
-- ============================================================================

CREATE TABLE approval_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    requested_by_signer_id UUID NOT NULL REFERENCES signers(id) ON DELETE RESTRICT,
    
    -- Request details
    required_approvals INTEGER NOT NULL,
    current_approvals INTEGER NOT NULL DEFAULT 0,
    required_level approval_level NOT NULL DEFAULT 'approve',
    
    -- Status
    is_approved BOOLEAN NOT NULL DEFAULT false,
    is_rejected BOOLEAN NOT NULL DEFAULT false,
    
    -- Expiration
    expires_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_approval_requests_tx ON approval_requests(transaction_id);
CREATE INDEX idx_approval_requests_status ON approval_requests(is_approved, is_rejected);
CREATE INDEX idx_approval_requests_expires ON approval_requests(expires_at) WHERE expires_at > NOW();

-- ============================================================================
-- WALLET USERS TABLE
-- ============================================================================

CREATE TABLE wallet_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
    
    -- User identity
    external_user_id VARCHAR(255) NOT NULL,
    email CITEXT,
    phone VARCHAR(50),
    
    -- KYC
    kyc_status VARCHAR(50) NOT NULL DEFAULT 'none', -- none, pending, verified, rejected
    kyc_level INTEGER NOT NULL DEFAULT 0,
    kyc_verified_at TIMESTAMPTZ,
    
    -- Access
    access_level approval_level NOT NULL DEFAULT 'view',
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Limits
    daily_limit NUMERIC(78, 0) NOT NULL DEFAULT 0,
    monthly_limit NUMERIC(78, 0) NOT NULL DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_external_user UNIQUE (master_wallet_id, external_user_id)
);

CREATE INDEX idx_wallet_users_master ON wallet_users(master_wallet_id);
CREATE INDEX idx_wallet_users_external ON wallet_users(external_user_id);
CREATE INDEX idx_wallet_users_kyc ON wallet_users(kyc_status);
CREATE INDEX idx_wallet_users_active ON wallet_users(is_active);

-- ============================================================================
-- WHITELIST TABLE
-- ============================================================================

CREATE TABLE whitelist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
    
    -- Address
    blockchain blockchain_type NOT NULL,
    address VARCHAR(255) NOT NULL,
    
    -- Type
    whitelist_type VARCHAR(50) NOT NULL DEFAULT 'address', -- address, contract, domain
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    
    -- Limits
    daily_limit NUMERIC(78, 0) NOT NULL DEFAULT 0,
    per_tx_limit NUMERIC(78, 0) NOT NULL DEFAULT 0,
    
    -- Metadata
    label VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_whitelist_address UNIQUE (master_wallet_id, blockchain, address)
);

CREATE INDEX idx_whitelist_master ON whitelist(master_wallet_id);
CREATE INDEX idx_whitelist_address ON whitelist(blockchain, address);
CREATE INDEX idx_whitelist_enabled ON whitelist(is_enabled);

-- ============================================================================
-- POLICIES TABLE
-- ============================================================================

CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
    
    -- Policy
    name VARCHAR(255) NOT NULL,
    policy_type VARCHAR(50) NOT NULL, -- spending, time_lock, geo, kyc
    is_active BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL DEFAULT 0,
    
    -- Rules (JSONB for flexibility)
    conditions JSONB NOT NULL DEFAULT '{}',
    actions JSONB NOT NULL DEFAULT '{}',
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX idx_policies_master ON policies(master_wallet_id);
CREATE INDEX idx_policies_type ON policies(policy_type);
CREATE INDEX idx_policies_active ON policies(is_active);

-- ============================================================================
-- AUDIT LOGS TABLE
-- ============================================================================

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE SET NULL,
    
    -- Event
    event_type VARCHAR(100) NOT NULL,
    event_category VARCHAR(50) NOT NULL, -- wallet, transaction, signer, policy, user
    severity VARCHAR(20) NOT NULL DEFAULT 'info', -- debug, info, warning, error, critical
    
    -- Actor
    actor_type VARCHAR(50), -- signer, user, system, api
    actor_id VARCHAR(255),
    actor_ip VARCHAR(45),
    actor_user_agent TEXT,
    
    -- Target
    target_type VARCHAR(50),
    target_id VARCHAR(255),
    
    -- Details
    action VARCHAR(100) NOT NULL,
    details JSONB DEFAULT '{}',
    changes JSONB DEFAULT '{}',
    
    -- Result
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_master ON audit_logs(master_wallet_id);
CREATE INDEX idx_audit_event ON audit_logs(event_type, event_category);
CREATE INDEX idx_audit_actor ON audit_logs(actor_type, actor_id);
CREATE INDEX idx_audit_target ON audit_logs(target_type, target_id);
CREATE INDEX idx_audit_created ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_severity ON audit_logs(severity);

-- ============================================================================
-- FEE CONFIGURATION TABLE
-- ============================================================================

CREATE TABLE fee_config (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
    
    -- Fee details
    fee_type VARCHAR(50) NOT NULL, -- deposit, withdrawal, transfer, swap, network
    fee_name VARCHAR(255) NOT NULL,
    
    -- Pricing
    fee_model VARCHAR(50) NOT NULL, -- fixed, percentage, tiered, volume
    percentage DECIMAL(10, 4) NOT NULL DEFAULT 0,
    flat_fee NUMERIC(78, 0) NOT NULL DEFAULT 0,
    
    -- Limits
    min_amount NUMERIC(78, 0),
    max_amount NUMERIC(78, 0),
    
    -- Token
    token_address VARCHAR(255),
    token_symbol VARCHAR(50),
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMPTZ,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fee_config_master ON fee_config(master_wallet_id);
CREATE INDEX idx_fee_config_type ON fee_config(fee_type);
CREATE INDEX idx_fee_config_active ON fee_config(is_active);

-- ============================================================================
-- TOKEN BALANCES TABLE
-- ============================================================================

CREATE TABLE token_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id UUID NOT NULL, -- Could be master_wallet or sub_wallet
    wallet_type VARCHAR(20) NOT NULL, -- 'master' or 'sub'
    
    -- Token
    blockchain blockchain_type NOT NULL,
    token_address VARCHAR(255),
    token_symbol VARCHAR(50) NOT NULL,
    token_name VARCHAR(255),
    token_decimals INTEGER NOT NULL DEFAULT 18,
    
    -- Balance
    balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
    available_balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
    locked_balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
    pending_balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_token_balance UNIQUE (wallet_id, wallet_type, blockchain, token_address)
);

CREATE INDEX idx_token_balances_wallet ON token_balances(wallet_id, wallet_type);
CREATE INDEX idx_token_balances_blockchain ON token_balances(blockchain);
CREATE INDEX idx_token_balances_symbol ON token_balances(token_symbol);

-- ============================================================================
-- NOTIFICATIONS TABLE
-- ============================================================================

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
    user_id UUID REFERENCES wallet_users(id) ON DELETE CASCADE,
    
    -- Notification
    notification_type VARCHAR(50) NOT NULL, -- transaction, approval, security, system
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'normal', -- low, normal, high, urgent
    
    -- Delivery
    channel VARCHAR(50) NOT NULL DEFAULT 'in_app', -- email, sms, push, in_app, webhook
    is_delivered BOOLEAN NOT NULL DEFAULT false,
    delivered_at TIMESTAMPTZ,
    is_read BOOLEAN NOT NULL DEFAULT false,
    read_at TIMESTAMPTZ,
    
    -- Data
    data JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_notifications_master ON notifications(master_wallet_id);
CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_type ON notifications(notification_type);
CREATE INDEX idx_notifications_read ON notifications(is_read);
CREATE INDEX idx_notifications_created ON notifications(created_at DESC);

-- ============================================================================
-- API KEYS TABLE
-- ============================================================================

CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
    user_id UUID REFERENCES wallet_users(id) ON DELETE SET NULL,
    
    -- Key details
    key_name VARCHAR(255) NOT NULL,
    key_hash TEXT NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    
    -- Permissions
    permissions JSONB NOT NULL DEFAULT '[]',
    allowed_ips CIDR[], -- Array of CIDR blocks
    allowed_origins TEXT[],
    
    -- Limits
    rate_limit INTEGER NOT NULL DEFAULT 1000, -- requests per minute
    daily_limit BIGINT NOT NULL DEFAULT 1000000,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_master ON api_keys(master_wallet_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_active ON api_keys(is_active);

-- ============================================================================
-- WEBHOOKS TABLE
-- ============================================================================

CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
    
    -- Webhook details
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    secret_encrypted TEXT NOT NULL,
    
    -- Events
    events TEXT[] NOT NULL, -- Array of event types
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    
    -- Delivery
    retry_count INTEGER NOT NULL DEFAULT 3,
    timeout_ms INTEGER NOT NULL DEFAULT 30000,
    
    -- Stats
    total_delivered BIGINT NOT NULL DEFAULT 0,
    total_failed BIGINT NOT NULL DEFAULT 0,
    last_delivered_at TIMESTAMPTZ,
    last_failed_at TIMESTAMPTZ,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_master ON webhooks(master_wallet_id);
CREATE INDEX idx_webhooks_active ON webhooks(is_active);

-- ============================================================================
-- WEBHOOK DELIVERY LOGS TABLE
-- ============================================================================

CREATE TABLE webhook_delivery_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    
    -- Event
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    response_status INTEGER,
    response_body TEXT,
    
    -- Status
    success BOOLEAN NOT NULL DEFAULT false,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    
    -- Timing
    attempt_number INTEGER NOT NULL DEFAULT 1,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER
);

CREATE INDEX idx_webhook_logs_webhook ON webhook_delivery_logs(webhook_id);
CREATE INDEX idx_webhook_logs_status ON webhook_delivery_logs(success);
CREATE INDEX idx_webhook_logs_sent ON webhook_delivery_logs(sent_at DESC);

-- ============================================================================
-- SESSIONS TABLE
-- ============================================================================

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
    user_id UUID REFERENCES wallet_users(id) ON DELETE SET NULL,
    signer_id UUID REFERENCES signers(id) ON DELETE SET NULL,
    
    -- Session details
    session_token_hash TEXT NOT NULL,
    refresh_token_hash TEXT,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMPTZ NOT NULL,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_token ON sessions(session_token_hash);
CREATE INDEX idx_sessions_master ON sessions(master_wallet_id);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_active ON sessions(is_active);
CREATE INDEX idx_sessions_expires ON sessions(expires_at) WHERE expires_at > NOW();

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Partition by master_wallet_id for faster queries
CREATE INDEX idx_transactions_master_hash ON transactions(master_wallet_id, tx_hash);
CREATE INDEX idx_transactions_master_status ON transactions(master_wallet_id, status);
CREATE INDEX idx_transactions_master_created ON transactions(master_wallet_id, created_at DESC);

-- Composite for balance queries
CREATE INDEX idx_balances_wallet_token ON token_balances(wallet_id, wallet_type, token_symbol);

-- ============================================================================
-- FUNCTIONS & TRIGGERS
-- ============================================================================

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add triggers for all tables with updated_at
CREATE TRIGGER update_master_wallets_updated_at BEFORE UPDATE ON master_wallets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sub_wallets_updated_at BEFORE UPDATE ON sub_wallets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_signers_updated_at BEFORE UPDATE ON signers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_approval_requests_updated_at BEFORE UPDATE ON approval_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_wallet_users_updated_at BEFORE UPDATE ON wallet_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_whitelist_updated_at BEFORE UPDATE ON whitelist
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_fee_config_updated_at BEFORE UPDATE ON fee_config
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notifications_updated_at BEFORE UPDATE ON notifications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_webhooks_updated_at BEFORE UPDATE ON webhooks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sessions_updated_at BEFORE UPDATE ON sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to generate address from public key
CREATE OR REPLACE FUNCTION generate_address_from_pubkey(
    p_pubkey BYTEA,
    p_blockchain blockchain_type
) RETURNS VARCHAR(255) AS $$
DECLARE
    v_hash BYTEA;
    v_address VARCHAR(255);
BEGIN
    -- SHA256 of public key
    v_hash := digest(p_pubkey, 'sha256');
    
    -- RIPEMD160 of SHA256
    v_hash := digest(v_hash, 'ripemd160');
    
    -- Add version byte based on blockchain
    CASE p_blockchain
        WHEN 'bitcoin' THEN
            v_hash := decode('00', 'hex') || v_hash;
        ELSE
            -- Ethereum-style: no version byte
            v_hash := decode('00', 'hex') || v_hash;
    END CASE;
    
    -- Double SHA256 for checksum
    v_hash := digest(v_hash, 'sha256');
    v_hash := digest(v_hash, 'sha256');
    
    -- Append first 4 bytes as checksum
    v_hash := substr(v_hash, 1, 4);
    
    -- Base58 encode
    v_address := encode(v_hash, 'base58');
    
    RETURN v_address;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to derive sub-wallet address
CREATE OR REPLACE FUNCTION derive_subwallet_address(
    p_master_address VARCHAR(255),
    p_derivation_index INTEGER,
    p_blockchain blockchain_type
) RETURNS VARCHAR(255) AS $$
DECLARE
    v_seed VARCHAR(512);
    v_hash BYTEA;
    v_address VARCHAR(255);
BEGIN
    -- Create seed from master address and index
    v_seed := p_master_address || '-' || p_derivation_index;
    
    -- SHA256 hash
    v_hash := digest(v_seed, 'sha256');
    
    -- RIPEMD160
    v_hash := digest(v_hash, 'ripemd160');
    
    -- Add blockchain prefix
    CASE p_blockchain
        WHEN 'bitcoin' THEN
            v_hash := decode('00', 'hex') || v_hash;
        ELSE
            v_hash := decode('00', 'hex') || v_hash;
    END CASE;
    
    -- Checksum
    v_hash := digest(v_hash, 'sha256');
    v_hash := digest(v_hash, 'sha256');
    v_hash := v_hash || substr(v_hash, 1, 4);
    
    -- Base58
    v_address := encode(v_hash, 'base58');
    
    RETURN v_address;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- ============================================================================
-- ROW LEVEL SECURITY POLICIES
-- ============================================================================

-- Enable RLS on sensitive tables
ALTER TABLE master_wallets ENABLE ROW LEVEL SECURITY;
ALTER TABLE sub_wallets ENABLE ROW LEVEL SECURITY;
ALTER TABLE signers ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE wallet_users ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- ============================================================================
-- ANALYTICS VIEWS
-- ============================================================================

-- Daily transaction volume view
CREATE OR REPLACE VIEW daily_transaction_volume AS
SELECT 
    master_wallet_id,
    blockchain,
    DATE(created_at) as date,
    COUNT(*) as transaction_count,
    SUM(amount) as total_volume,
    SUM(fee_amount) as total_fees
FROM transactions
WHERE status = 'confirmed'
GROUP BY master_wallet_id, blockchain, DATE(created_at);

-- Signer activity view
CREATE OR REPLACE VIEW signer_activity AS
SELECT 
    s.id,
    s.signer_name,
    s.master_wallet_id,
    COUNT(ts.id) as total_signatures,
    SUM(CASE WHEN ts.signature_status = 'signed' THEN 1 ELSE 0 END) as approved_count,
    SUM(CASE WHEN ts.signature_status = 'rejected' THEN 1 ELSE 0 END) as rejected_count,
    MAX(ts.updated_at) as last_activity
FROM signers s
LEFT JOIN transaction_signatures ts ON s.id = ts.signer_id
GROUP BY s.id, s.signer_name, s.master_wallet_id;

-- Wallet balance summary
CREATE OR REPLACE VIEW wallet_balance_summary AS
SELECT 
    wb.wallet_id,
    wb.wallet_type,
    wb.blockchain,
    wb.token_symbol,
    wb.balance,
    wb.available_balance,
    wb.locked_balance,
    mw.name as master_wallet_name
FROM token_balances wb
LEFT JOIN master_wallets mw ON wb.wallet_id = mw.id AND wb.wallet_type = 'master';

-- ============================================================================
-- PERFORMANCE OPTIMIZATIONS
-- ============================================================================

-- Analyze all tables
ANALYZE master_wallets;
ANALYZE sub_wallets;
ANALYZE signers;
ANALYZE transactions;
ANALYZE transaction_signatures;
ANALYZE approval_requests;
ANALYZE wallet_users;
ANALYZE whitelist;
ANALYZE policies;
ANALYZE audit_logs;
ANALYZE fee_config;
ANALYZE token_balances;
ANALYZE notifications;
ANALYZE api_keys;
ANALYZE webhooks;
ANALYZE webhook_delivery_logs;
ANALYZE sessions;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE master_wallets IS 'Core master wallet table - owner of all sub-wallets';
COMMENT ON TABLE sub_wallets IS 'User wallets owned by master wallet - derived from master seed';
COMMENT ON TABLE signers IS 'Multi-sig signers with approval permissions';
COMMENT ON TABLE transactions IS 'All transactions initiated from master wallet';
COMMENT ON TABLE transaction_signatures IS 'Individual signatures for multi-sig transactions';
COMMENT ON TABLE approval_requests IS 'Approval workflow for multi-sig transactions';
COMMENT ON TABLE wallet_users IS 'Users associated with the master wallet';
COMMENT ON TABLE whitelist IS 'Approved addresses for transactions';
COMMENT ON TABLE policies IS 'Spending limits, time locks, and other rules';
COMMENT ON TABLE audit_logs IS 'Complete audit trail of all operations';
COMMENT ON TABLE fee_config IS 'Fee structure for different transaction types';
COMMENT ON TABLE token_balances IS 'Token balances for all wallets';
COMMENT ON TABLE notifications IS 'User notifications and alerts';
COMMENT ON TABLE api_keys IS 'API keys for programmatic access';
COMMENT ON TABLE webhooks IS 'Webhook endpoints for event notifications';
COMMENT ON TABLE sessions IS 'Active user sessions';

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================
