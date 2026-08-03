// Database Models - PostgreSQL Schema Definitions
// Production-ready with proper relationships and constraints

export const schema = `
-- Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    wallet_address VARCHAR(42) UNIQUE,
    phone VARCHAR(20),
    kyc_level VARCHAR(20) DEFAULT 'NONE',
    kyc_status VARCHAR(20) DEFAULT 'PENDING',
    status VARCHAR(20) DEFAULT 'ACTIVE',
    risk_score INTEGER DEFAULT 100,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP,
    
    CONSTRAINT valid_kyc CHECK (kyc_level IN ('NONE', 'BASIC', 'INTERMEDIATE', 'FULL')),
    CONSTRAINT valid_status CHECK (status IN ('ACTIVE', 'SUSPENDED', 'BANNED', 'KYC_PENDING'))
);

-- User Security Table
CREATE TABLE user_security (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret VARCHAR(255),
    login_otp VARCHAR(10),
    otp_expires_at TIMESTAMP,
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    last_password_change TIMESTAMP DEFAULT NOW(),
    password_history TEXT[], -- JSON array of old password hashes
    
    UNIQUE(user_id)
);

-- Wallets Table
CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    chain VARCHAR(20) NOT NULL,
    address VARCHAR(42) NOT NULL,
    private_key_encrypted VARCHAR(255) NOT NULL,
    balance DECIMAL(50, 18) DEFAULT 0,
    reserved_balance DECIMAL(50, 18) DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(chain, address)
);

-- Tokens Table
CREATE TABLE tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    decimals INTEGER NOT NULL,
    contract_address VARCHAR(42),
    chain VARCHAR(20) NOT NULL,
    is_native BOOLEAN DEFAULT FALSE,
    is_tradable BOOLEAN DEFAULT TRUE,
    min_swap_amount DECIMAL(50, 18) DEFAULT 0,
    max_swap_amount DECIMAL(50, 18),
    
    CONSTRAINT valid_chain CHECK (chain IN ('ETHEREUM', 'BSC', 'POLYGON', 'ARBITRUM', 'OPTIMISM', 'AVAX', 'SOLANA'))
);

-- P2P Merchants Table
CREATE TABLE p2p_merchants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    status VARCHAR(20) DEFAULT 'PENDING',
    collateral_token VARCHAR(20) NOT NULL,
    collateral_amount DECIMAL(50, 18) NOT NULL,
    collateral_tx_hash VARCHAR(66),
    collateral_locked_at TIMESTAMP,
    trader_level VARCHAR(20) DEFAULT 'NEWBIE',
    total_trades INTEGER DEFAULT 0,
    total_volume DECIMAL(50, 2) DEFAULT 0,
    completed_trades INTEGER DEFAULT 0,
    cancelled_trades INTEGER DEFAULT 0,
    dispute_count INTEGER DEFAULT 0,
    rating DECIMAL(3, 2) DEFAULT 0,
    total_reviews INTEGER DEFAULT 0,
    avg_response_time DECIMAL(10, 2) DEFAULT 0,
    avg_release_time DECIMAL(10, 2) DEFAULT 0,
    security_score INTEGER DEFAULT 100,
    is_verified BOOLEAN DEFAULT FALSE,
    kyc_level VARCHAR(20) DEFAULT 'NONE',
    joined_at TIMESTAMP DEFAULT NOW(),
    last_active_at TIMESTAMP,
    
    CONSTRAINT valid_merchant_status CHECK (status IN ('PENDING', 'ACTIVE', 'SUSPENDED', 'BANNED')),
    CONSTRAINT valid_trader_level CHECK (trader_level IN ('NEWBIE', 'BRONZE', 'SILVER', 'GOLD', 'PLATINUM', 'DIAMOND'))
);

-- Merchant Collateral History
CREATE TABLE merchant_collateral_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID REFERENCES p2p_merchants(id) ON DELETE CASCADE,
    transaction_type VARCHAR(20) NOT NULL,
    token VARCHAR(20) NOT NULL,
    amount DECIMAL(50, 18) NOT NULL,
    usd_value DECIMAL(50, 2) NOT NULL,
    tx_hash VARCHAR(66),
    reason VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT valid_tx_type CHECK (transaction_type IN ('LOCK', 'UNLOCK', 'SLASH', 'ADD'))
);

-- P2P Advertisements
CREATE TABLE p2p_adverts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID REFERENCES p2p_merchants(id) ON DELETE CASCADE,
    side VARCHAR(10) NOT NULL,
    token_id UUID REFERENCES tokens(id),
    fiat_currency VARCHAR(10) NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    price DECIMAL(50, 8) NOT NULL,
    min_amount DECIMAL(50, 2) NOT NULL,
    max_amount DECIMAL(50, 2) NOT NULL,
    available_amount DECIMAL(50, 18) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    auto_reply_message TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    
    CONSTRAINT valid_side CHECK (side IN ('BUY', 'SELL'))
);

-- P2P Orders
CREATE TABLE p2p_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    advert_id UUID REFERENCES p2p_adverts(id),
    buyer_id UUID REFERENCES users(id),
    seller_id UUID REFERENCES users(id),
    side VARCHAR(10) NOT NULL,
    token_id UUID REFERENCES tokens(id),
    fiat_currency VARCHAR(10) NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    price DECIMAL(50, 8) NOT NULL,
    amount DECIMAL(50, 18) NOT NULL,
    fiat_amount DECIMAL(50, 2) NOT NULL,
    buyer_deposit DECIMAL(50, 18),
    seller_deposit DECIMAL(50, 18),
    buyer_deposit_tx VARCHAR(66),
    seller_deposit_tx VARCHAR(66),
    status VARCHAR(20) DEFAULT 'PENDING',
    buyer_confirm_time TIMESTAMP,
    seller_confirm_time TIMESTAMP,
    release_time TIMESTAMP,
    cancel_time TIMESTAMP,
    cancel_reason VARCHAR(255),
    dispute_opened BOOLEAN DEFAULT FALSE,
    dispute_reason TEXT,
    dispute_resolution VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT valid_order_status CHECK (status IN ('PENDING', 'PAID', 'CONFIRMED', 'COMPLETED', 'CANCELLED', 'DISPUTED', 'REFUNDED'))
);

-- Security Deposits
CREATE TABLE security_deposits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES p2p_orders(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    deposit_type VARCHAR(20) NOT NULL,
    token_id UUID REFERENCES tokens(id),
    amount DECIMAL(50, 18) NOT NULL,
    usd_value DECIMAL(50, 2) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    status VARCHAR(20) DEFAULT 'LOCKED',
    locked_at TIMESTAMP DEFAULT NOW(),
    released_at TIMESTAMP,
    release_tx VARCHAR(66),
    
    CONSTRAINT valid_deposit_type CHECK (deposit_type IN ('BUYER_PROTECTION', 'SELLER_BOND')),
    CONSTRAINT valid_deposit_status CHECK (status IN ('LOCKED', 'RELEASED', 'FORFEITED', 'REFUNDED'))
);

-- Margin Accounts
CREATE TABLE margin_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    total_assets DECIMAL(50, 18) DEFAULT 0,
    total_liabilities DECIMAL(50, 18) DEFAULT 0,
    net_assets DECIMAL(50, 18) DEFAULT 0,
    available_balance DECIMAL(50, 18) DEFAULT 0,
    margin_ratio DECIMAL(10, 2) DEFAULT 0,
    risk_level VARCHAR(20) DEFAULT 'SAFE',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT valid_risk CHECK (risk_level IN ('SAFE', 'WARNING', 'LIQUIDATION'))
);

-- Margin Positions
CREATE TABLE margin_positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID REFERENCES margin_accounts(id) ON DELETE CASCADE,
    token_id UUID REFERENCES tokens(id),
    side VARCHAR(10) NOT NULL,
    size DECIMAL(50, 18) NOT NULL,
    entry_price DECIMAL(50, 8) NOT NULL,
    mark_price DECIMAL(50, 8),
    leverage INTEGER NOT NULL,
    margin DECIMAL(50, 18) NOT NULL,
    margin_mode VARCHAR(20) DEFAULT 'CROSS',
    pnl DECIMAL(50, 18) DEFAULT 0,
    liquidation_price DECIMAL(50, 8),
    status VARCHAR(20) DEFAULT 'OPEN',
    opened_at TIMESTAMP DEFAULT NOW(),
    closed_at TIMESTAMP,
    
    CONSTRAINT valid_margin_side CHECK (side IN ('LONG', 'SHORT')),
    CONSTRAINT valid_margin_mode CHECK (margin_mode IN ('CROSS', 'ISOLATED')),
    CONSTRAINT valid_position_status CHECK (status IN ('OPEN', 'CLOSED', 'LIQUIDATED'))
);

-- Margin Borrow Records
CREATE TABLE margin_borrows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID REFERENCES margin_accounts(id) ON DELETE CASCADE,
    token_id UUID REFERENCES tokens(id),
    amount DECIMAL(50, 18) NOT NULL,
    interest_rate DECIMAL(10, 6) NOT NULL,
    interest_accrued DECIMAL(50, 18) DEFAULT 0,
    status VARCHAR(20) DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT NOW(),
    repaid_at TIMESTAMP
);

-- Crypto Cards
CREATE TABLE crypto_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    card_number_encrypted VARCHAR(255) NOT NULL,
    cvv_encrypted VARCHAR(255) NOT NULL,
    expiry_date VARCHAR(5) NOT NULL,
    card_holder VARCHAR(100) NOT NULL,
    card_type VARCHAR(20) NOT NULL,
    network VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'ACTIVE',
    daily_limit DECIMAL(50, 2) DEFAULT 10000,
    monthly_limit DECIMAL(50, 2) DEFAULT 100000,
    daily_spent DECIMAL(50, 2) DEFAULT 0,
    monthly_spent DECIMAL(50, 2) DEFAULT 0,
    apple_pay_enabled BOOLEAN DEFAULT FALSE,
    google_pay_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    activated_at TIMESTAMP,
    terminated_at TIMESTAMP,
    
    CONSTRAINT valid_card_type CHECK (card_type IN ('VIRTUAL', 'PHYSICAL')),
    CONSTRAINT valid_card_network CHECK (network IN ('VISA', 'MASTERCARD')),
    CONSTRAINT valid_card_status CHECK (status IN ('PENDING', 'ACTIVE', 'FROZEN', 'TERMINATED'))
);

-- Card Transactions
CREATE TABLE card_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id UUID REFERENCES crypto_cards(id) ON DELETE CASCADE,
    merchant_name VARCHAR(100) NOT NULL,
    merchant_category VARCHAR(50),
    amount DECIMAL(50, 2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'USD',
    amount_usd DECIMAL(50, 2) NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING',
    tx_hash VARCHAR(66),
    created_at TIMESTAMP DEFAULT NOW(),
    settled_at TIMESTAMP
);

-- Fiat Orders (On-Ramp)
CREATE TABLE fiat_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    provider VARCHAR(50) NOT NULL,
    side VARCHAR(10) NOT NULL,
    fiat_currency VARCHAR(10) NOT NULL,
    crypto_currency VARCHAR(20) NOT NULL,
    fiat_amount DECIMAL(50, 2) NOT NULL,
    crypto_amount DECIMAL(50, 18) NOT NULL,
    exchange_rate DECIMAL(50, 8) NOT NULL,
    fee DECIMAL(50, 2) NOT NULL,
    payment_method VARCHAR(50),
    wallet_address VARCHAR(42),
    status VARCHAR(20) DEFAULT 'PENDING',
    provider_order_id VARCHAR(100),
    tx_hash VARCHAR(66),
    created_at TIMESTAMP DEFAULT NOW(),
    confirmed_at TIMESTAMP,
    completed_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    
    CONSTRAINT valid_fiat_side CHECK (side IN ('BUY', 'SELL'))
);

-- API Keys for programmatic access
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    permissions TEXT[], -- JSON array of permissions
    rate_limit INTEGER DEFAULT 100,
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Audit Log
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id VARCHAR(100),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for Performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_wallet ON users(wallet_address);
CREATE INDEX idx_p2p_merchants_user ON p2p_merchants(user_id);
CREATE INDEX idx_p2p_adverts_merchant ON p2p_adverts(merchant_id);
CREATE INDEX idx_p2p_orders_buyer ON p2p_orders(buyer_id);
CREATE INDEX idx_p2p_orders_seller ON p2p_orders(seller_id);
CREATE INDEX idx_p2p_orders_status ON p2p_orders(status);
CREATE INDEX idx_margin_positions_account ON margin_positions(account_id);
CREATE INDEX idx_crypto_cards_user ON crypto_cards(user_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);
`;

export default schema;
