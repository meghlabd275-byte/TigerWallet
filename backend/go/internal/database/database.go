package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/tigerwallet/backend/internal/config"
)

func Connect(cfg *config.Config) (*sql.DB, error) {
	sslMode := "disable"
	if cfg.Database.SSLMode {
		sslMode = "require"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		sslMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * 60)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	migrations := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(50) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			wallet_address VARCHAR(42),
			phone VARCHAR(20),
			kyc_level VARCHAR(20) DEFAULT 'NONE',
			kyc_status VARCHAR(20) DEFAULT 'PENDING',
			status VARCHAR(20) DEFAULT 'ACTIVE',
			risk_score INTEGER DEFAULT 100,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			last_login_at TIMESTAMP
		)`,

		// Sessions table
		`CREATE TABLE IF NOT EXISTS sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			is_revoked BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			last_activity_at TIMESTAMP DEFAULT NOW()
		)`,

		// Tokens table
		`CREATE TABLE IF NOT EXISTS tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			symbol VARCHAR(20) UNIQUE NOT NULL,
			name VARCHAR(100) NOT NULL,
			decimals INTEGER NOT NULL,
			contract_address VARCHAR(42),
			chain VARCHAR(20) NOT NULL,
			is_native BOOLEAN DEFAULT FALSE,
			is_tradable BOOLEAN DEFAULT TRUE,
			min_swap_amount DECIMAL(50, 18) DEFAULT 0,
			max_swap_amount DECIMAL(50, 18)
		)`,

		// P2P Merchants table
		`CREATE TABLE IF NOT EXISTS p2p_merchants (
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
			last_active_at TIMESTAMP
		)`,

		// P2P Advertisements
		`CREATE TABLE IF NOT EXISTS p2p_adverts (
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
			expires_at TIMESTAMP
		)`,

		// P2P Orders
		`CREATE TABLE IF NOT EXISTS p2p_orders (
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
			updated_at TIMESTAMP DEFAULT NOW()
		)`,

		// Security Deposits
		`CREATE TABLE IF NOT EXISTS security_deposits (
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
			release_tx VARCHAR(66)
		)`,

		// Margin Accounts
		`CREATE TABLE IF NOT EXISTS margin_accounts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE UNIQUE,
			total_assets DECIMAL(50, 18) DEFAULT 0,
			total_liabilities DECIMAL(50, 18) DEFAULT 0,
			net_assets DECIMAL(50, 18) DEFAULT 0,
			available_balance DECIMAL(50, 18) DEFAULT 0,
			margin_ratio DECIMAL(10, 2) DEFAULT 0,
			risk_level VARCHAR(20) DEFAULT 'SAFE',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,

		// Margin Positions
		`CREATE TABLE IF NOT EXISTS margin_positions (
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
			closed_at TIMESTAMP
		)`,

		// Wallets
		`CREATE TABLE IF NOT EXISTS wallets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			chain VARCHAR(20) NOT NULL,
			address VARCHAR(42) NOT NULL,
			private_key_encrypted VARCHAR(255) NOT NULL,
			balance DECIMAL(50, 18) DEFAULT 0,
			reserved_balance DECIMAL(50, 18) DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(chain, address)
		)`,

		// Crypto Cards
		`CREATE TABLE IF NOT EXISTS crypto_cards (
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
			created_at TIMESTAMP DEFAULT NOW(),
			activated_at TIMESTAMP
		)`,

		// API Keys
		`CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			key_hash VARCHAR(255) NOT NULL,
			key_prefix VARCHAR(10) NOT NULL,
			name VARCHAR(100) NOT NULL,
			permissions TEXT[],
			rate_limit INTEGER DEFAULT 100,
			is_active BOOLEAN DEFAULT TRUE,
			last_used_at TIMESTAMP,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		// Audit Logs
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id),
			action VARCHAR(100) NOT NULL,
			entity_type VARCHAR(50),
			entity_id VARCHAR(100),
			details JSONB,
			ip_address INET,
			user_agent TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		// Lending Tables
		`CREATE TABLE IF NOT EXISTS lending_pools (
			token VARCHAR(20) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			symbol VARCHAR(20) NOT NULL,
			total_supplied DECIMAL(50, 18) DEFAULT 0,
			total_borrowed DECIMAL(50, 18) DEFAULT 0,
			supply_apy DECIMAL(10, 4) DEFAULT 0.05,
			borrow_apy DECIMAL(10, 4) DEFAULT 0.08,
			liquidity DECIMAL(50, 18) DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS lending_positions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			token VARCHAR(20) NOT NULL,
			supplied DECIMAL(50, 18) DEFAULT 0,
			borrowed DECIMAL(50, 18) DEFAULT 0,
			apy DECIMAL(10, 4) DEFAULT 0,
			accumulated DECIMAL(50, 18) DEFAULT 0,
			status VARCHAR(20) DEFAULT 'ACTIVE',
			supplied_at TIMESTAMP DEFAULT NOW()
		)`,

		// Bridge Tables
		`CREATE TABLE IF NOT EXISTS bridge_transactions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			from_chain VARCHAR(20) NOT NULL,
			to_chain VARCHAR(20) NOT NULL,
			token VARCHAR(20) NOT NULL,
			amount DECIMAL(50, 18) NOT NULL,
			fee DECIMAL(50, 18) DEFAULT 0,
			received_amount DECIMAL(50, 18) NOT NULL,
			status VARCHAR(20) DEFAULT 'PENDING',
			source_tx_hash VARCHAR(66),
			destination_tx_hash VARCHAR(66),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS bridge_tokens (
			token VARCHAR(20) NOT NULL,
			chain VARCHAR(20) NOT NULL,
			min_amount DECIMAL(50, 18) DEFAULT 0,
			max_amount DECIMAL(50, 18),
			is_active BOOLEAN DEFAULT TRUE,
			PRIMARY KEY(token, chain)
		)`,

		// Gift Card Tables
		`CREATE TABLE IF NOT EXISTS gift_cards (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			code VARCHAR(50) UNIQUE NOT NULL,
			token VARCHAR(20) NOT NULL,
			amount DECIMAL(50, 18) NOT NULL,
			template_id VARCHAR(50),
			status VARCHAR(20) DEFAULT 'ACTIVE',
			created_by UUID REFERENCES users(id),
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS gift_card_templates (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			image_url TEXT,
			is_active BOOLEAN DEFAULT TRUE
		)`,

		// Hardware Wallet Tables
		`CREATE TABLE IF NOT EXISTS hardware_wallets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			device_type VARCHAR(50) NOT NULL,
			serial_number VARCHAR(100) NOT NULL,
			firmware_version VARCHAR(50),
			status VARCHAR(20) DEFAULT 'ACTIVE',
			registered_at TIMESTAMP DEFAULT NOW(),
			last_used_at TIMESTAMP DEFAULT NOW()
		)`,

		// MPC Wallet Tables
		`CREATE TABLE IF NOT EXISTS mpc_wallets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			wallet_address VARCHAR(42) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS mpc_wallet_shares (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			device_id VARCHAR(100) NOT NULL,
			public_key VARCHAR(130) NOT NULL,
			encrypted_share TEXT NOT NULL,
			status VARCHAR(20) DEFAULT 'ACTIVE',
			created_at TIMESTAMP DEFAULT NOW(),
			last_used_at TIMESTAMP DEFAULT NOW()
		)`,

		// Social Recovery Tables
		`CREATE TABLE IF NOT EXISTS recovery_setups (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE UNIQUE,
			recovery_key VARCHAR(255) NOT NULL,
			threshold INTEGER NOT NULL,
			status VARCHAR(20) DEFAULT 'ACTIVE',
			guardian_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS guardians (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			address VARCHAR(42) NOT NULL,
			name VARCHAR(100) NOT NULL,
			relationship VARCHAR(50),
			status VARCHAR(20) DEFAULT 'PENDING',
			added_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS recovery_requests (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			guardian_id UUID REFERENCES guardians(id),
			status VARCHAR(20) DEFAULT 'PENDING',
			initiated_at TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS recovery_confirmations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			request_id UUID REFERENCES recovery_requests(id),
			guardian_id UUID REFERENCES guardians(id),
			confirmed_at TIMESTAMP DEFAULT NOW()
		)`,

		// Account Abstraction Tables
		`CREATE TABLE IF NOT EXISTS smart_accounts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			account_address VARCHAR(42) UNIQUE NOT NULL,
			owner_address VARCHAR(42) NOT NULL,
			nonce INTEGER DEFAULT 0,
			threshold INTEGER DEFAULT 1,
			status VARCHAR(20) DEFAULT 'ACTIVE',
			deployed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS account_signers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			account_address VARCHAR(42) NOT NULL,
			signer_address VARCHAR(42) NOT NULL,
			weight INTEGER DEFAULT 1,
			status VARCHAR(20) DEFAULT 'ACTIVE',
			added_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS session_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			account_address VARCHAR(42) NOT NULL,
			session_key VARCHAR(130) NOT NULL,
			permissions JSONB,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS user_operations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_op_hash VARCHAR(66) UNIQUE NOT NULL,
			sender VARCHAR(42) NOT NULL,
			nonce INTEGER NOT NULL,
			init_code TEXT,
			call_data TEXT,
			call_gas_limit INTEGER,
			verification_gas_limit INTEGER,
			pre_verification_gas INTEGER,
			max_fee_per_gas VARCHAR(50),
			max_priority_fee_per_gas VARCHAR(50),
			signature TEXT,
			status VARCHAR(20) DEFAULT 'PENDING',
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS paymaster_sponsors (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			account_address VARCHAR(42) NOT NULL,
			paymaster_address VARCHAR(42) NOT NULL,
			enabled_at TIMESTAMP DEFAULT NOW()
		)`,

		// DApp Browser Tables
		`CREATE TABLE IF NOT EXISTS dapps (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			url TEXT NOT NULL,
			description TEXT,
			logo_url TEXT,
			category VARCHAR(50),
			rating DECIMAL(3, 2) DEFAULT 0,
			users INTEGER DEFAULT 0,
			volume_24h DECIMAL(50, 2) DEFAULT 0,
			is_verified BOOLEAN DEFAULT FALSE,
			chains TEXT[],
			status VARCHAR(20) DEFAULT 'PENDING',
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS dapp_favorites (
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			dapp_id UUID REFERENCES dapps(id) ON DELETE CASCADE,
			added_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY(user_id, dapp_id)
		)`,

		`CREATE TABLE IF NOT EXISTS dapp_history (
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			dapp_id UUID REFERENCES dapps(id) ON DELETE CASCADE,
			url TEXT NOT NULL,
			visited_at TIMESTAMP DEFAULT NOW()
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_wallet ON users(wallet_address)`,
		`CREATE INDEX IF NOT EXISTS idx_p2p_merchants_user ON p2p_merchants(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_p2p_adverts_merchant ON p2p_adverts(merchant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_p2p_orders_buyer ON p2p_orders(buyer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_p2p_orders_seller ON p2p_orders(seller_id)`,
		`CREATE INDEX IF NOT EXISTS idx_p2p_orders_status ON p2p_orders(status)`,
		`CREATE INDEX IF NOT EXISTS idx_margin_positions_account ON margin_positions(account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_crypto_cards_user ON crypto_cards(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_lending_positions_user ON lending_positions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bridge_transactions_user ON bridge_transactions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_gift_cards_code ON gift_cards(code)`,
		`CREATE INDEX IF NOT EXISTS idx_hardware_wallets_user ON hardware_wallets(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mpc_shares_user ON mpc_wallet_shares(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_recovery_guardians_user ON guardians(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_smart_accounts_user ON smart_accounts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_ops_sender ON user_operations(sender)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w\nQuery: %s", err, migration)
		}
	}

	return nil
}
