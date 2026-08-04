//! Database migrations for MasterWallet
//! Handles schema creation and updates

/// Run all migrations
pub fn run_migrations() -> &'static str {
    r#"
-- Master Wallets table
CREATE TABLE IF NOT EXISTS master_wallets (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    wallet_type VARCHAR(32) NOT NULL DEFAULT 'HOT',
    address VARCHAR(128) NOT NULL UNIQUE,
    public_key VARCHAR(256),
    chain_id BIGINT NOT NULL DEFAULT 1,
    encrypted_private_key TEXT,
    is_active BOOLEAN DEFAULT true,
    settings JSONB DEFAULT '{}',
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_master_wallets_address ON master_wallets(address);
CREATE INDEX IF NOT EXISTS idx_master_wallets_active ON master_wallets(is_active);

-- Sub Wallets table
CREATE TABLE IF NOT EXISTS sub_wallets (
    id VARCHAR(64) PRIMARY KEY,
    master_wallet_id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(128) NOT NULL,
    address_type VARCHAR(32) NOT NULL DEFAULT 'EVM',
    public_key VARCHAR(256),
    encrypted_private_key TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_subwallets_master ON sub_wallets(master_wallet_id);
CREATE INDEX IF NOT EXISTS idx_subwallets_address ON sub_wallets(address);
CREATE INDEX IF NOT EXISTS idx_subwallets_active ON sub_wallets(is_active);

-- Wallet Users table
CREATE TABLE IF NOT EXISTS wallet_users (
    id VARCHAR(64) PRIMARY KEY,
    wallet_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(32) NOT NULL DEFAULT 'USER',
    permissions TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    FOREIGN KEY (wallet_id) REFERENCES sub_wallets(id) ON DELETE CASCADE,
    UNIQUE(wallet_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_walletusers_wallet ON wallet_users(wallet_id);
CREATE INDEX IF NOT EXISTS idx_walletusers_email ON wallet_users(email);
CREATE INDEX IF NOT EXISTS idx_walletusers_userid ON wallet_users(user_id);

-- Auto Sign Rules table
CREATE TABLE IF NOT EXISTS auto_sign_rules (
    id VARCHAR(64) PRIMARY KEY,
    master_wallet_id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    max_amount VARCHAR(64) NOT NULL,
    chain_ids TEXT[] DEFAULT '{}',
    token_ids TEXT[] DEFAULT '{}',
    enabled BOOLEAN DEFAULT true,
    conditions TEXT[] DEFAULT '{}',
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_autosign_master ON auto_sign_rules(master_wallet_id);
CREATE INDEX IF NOT EXISTS idx_autosign_enabled ON auto_sign_rules(enabled);

-- Transaction Approvals table
CREATE TABLE IF NOT EXISTS transaction_approvals (
    id VARCHAR(64) PRIMARY KEY,
    master_wallet_id VARCHAR(64) NOT NULL,
    sub_wallet_id VARCHAR(64) NOT NULL,
    tx_hash VARCHAR(128) NOT NULL,
    from_address VARCHAR(64) NOT NULL,
    to_address VARCHAR(64) NOT NULL,
    amount VARCHAR(64) NOT NULL,
    token_id VARCHAR(64),
    chain_id BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    approved_by VARCHAR(64),
    rejected_by VARCHAR(64),
    reject_reason TEXT,
    gas_used VARCHAR(32),
    block_number BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id) ON DELETE CASCADE,
    FOREIGN KEY (sub_wallet_id) REFERENCES sub_wallets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_txapproval_master ON transaction_approvals(master_wallet_id);
CREATE INDEX IF NOT EXISTS idx_txapproval_subwallet ON transaction_approvals(sub_wallet_id);
CREATE INDEX IF NOT EXISTS idx_txapproval_status ON transaction_approvals(status);
CREATE INDEX IF NOT EXISTS idx_txapproval_hash ON transaction_approvals(tx_hash);
CREATE INDEX IF NOT EXISTS idx_txapproval_created ON transaction_approvals(created_at);

-- Role Permissions table
CREATE TABLE IF NOT EXISTS role_permissions (
    id SERIAL PRIMARY KEY,
    master_wallet_id VARCHAR(64),
    role_name VARCHAR(32) NOT NULL,
    permissions TEXT[] NOT NULL,
    description TEXT,
    created_at BIGINT NOT NULL,
    UNIQUE(master_wallet_id, role_name)
);

-- Insert default roles
INSERT INTO role_permissions (master_wallet_id, role_name, permissions, description, created_at)
VALUES 
    (NULL, 'ADMIN', ARRAY['all'], 'Full admin access', EXTRACT(EPOCH FROM NOW())::bigint),
    (NULL, 'USER', ARRAY['view', 'transact'], 'Regular user', EXTRACT(EPOCH FROM NOW())::bigint),
    (NULL, 'VIEWER', ARRAY['view'], 'Read-only access', EXTRACT(EPOCH FROM NOW())::bigint),
    (NULL, 'MANAGER', ARRAY['view', 'transact', 'approve'], 'Manager with approval rights', EXTRACT(EPOCH FROM NOW())::bigint)
ON CONFLICT (master_wallet_id, role_name) DO NOTHING;

-- Address Whitelist table
CREATE TABLE IF NOT EXISTS address_whitelist (
    id VARCHAR(64) PRIMARY KEY,
    master_wallet_id VARCHAR(64) NOT NULL,
    address VARCHAR(128) NOT NULL,
    address_type VARCHAR(32) NOT NULL DEFAULT 'EVM',
    label TEXT,
    is_verified BOOLEAN DEFAULT false,
    added_by VARCHAR(64) NOT NULL,
    created_at BIGINT NOT NULL,
    FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id) ON DELETE CASCADE,
    UNIQUE(master_wallet_id, address)
);

CREATE INDEX IF NOT EXISTS idx_whitelist_master ON address_whitelist(master_wallet_id);
CREATE INDEX IF NOT EXISTS idx_whitelist_address ON address_whitelist(address);

-- Wallet Tokens table
CREATE TABLE IF NOT EXISTS wallet_tokens (
    id VARCHAR(64) PRIMARY KEY,
    wallet_id VARCHAR(64) NOT NULL,
    token_id VARCHAR(64) NOT NULL,
    chain_id BIGINT NOT NULL,
    balance VARCHAR(64) NOT NULL DEFAULT '0',
    reserved_balance VARCHAR(64) NOT NULL DEFAULT '0',
    updated_at BIGINT NOT NULL,
    FOREIGN KEY (wallet_id) REFERENCES sub_wallets(id) ON DELETE CASCADE,
    UNIQUE(wallet_id, token_id, chain_id)
);

CREATE INDEX IF NOT EXISTS idx_wallettokens_wallet ON wallet_tokens(wallet_id);
CREATE INDEX IF NOT EXISTS idx_wallettokens_token ON wallet_tokens(token_id);

-- Fee Configuration table
CREATE TABLE IF NOT EXISTS fee_config (
    id VARCHAR(64) PRIMARY KEY,
    master_wallet_id VARCHAR(64) NOT NULL,
    fee_type VARCHAR(32) NOT NULL,
    percentage DECIMAL(10, 4) NOT NULL DEFAULT 0,
    flat_fee VARCHAR(32) NOT NULL DEFAULT '0',
    min_amount VARCHAR(32),
    max_amount VARCHAR(32),
    is_active BOOLEAN DEFAULT true,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id) ON DELETE CASCADE
);

-- Audit Log table
CREATE TABLE IF NOT EXISTS audit_logs (
    id VARCHAR(64) PRIMARY KEY,
    master_wallet_id VARCHAR(64),
    user_id VARCHAR(64),
    action VARCHAR(64) NOT NULL,
    entity_type VARCHAR(32),
    entity_id VARCHAR(64),
    details JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auditlogs_master ON audit_logs(master_wallet_id);
CREATE INDEX IF NOT EXISTS idx_auditlogs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_auditlogs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_auditlogs_created ON audit_logs(created_at);
"#
}

/// Rollback migrations (if needed)
pub fn rollback_migrations() -> &'static str {
    r#"
    DROP TABLE IF EXISTS audit_logs CASCADE;
    DROP TABLE IF EXISTS fee_config CASCADE;
    DROP TABLE IF EXISTS wallet_tokens CASCADE;
    DROP TABLE IF EXISTS address_whitelist CASCADE;
    DROP TABLE IF EXISTS role_permissions CASCADE;
    DROP TABLE IF EXISTS transaction_approvals CASCADE;
    DROP TABLE IF EXISTS auto_sign_rules CASCADE;
    DROP TABLE IF EXISTS wallet_users CASCADE;
    DROP TABLE IF EXISTS sub_wallets CASCADE;
    DROP TABLE IF EXISTS master_wallets CASCADE;
"#
}
