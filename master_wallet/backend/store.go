package main

// store.go — PostgreSQL persistence + Redis cache for the MasterWallet backend.
// Wires the canonical schema (master_wallets, sub_wallets, transactions,
// signers, approval_requests, wallet_users, whitelist, policies, audit_logs,
// fee_config, token_balances, notifications, api_keys, webhooks, sessions).
// No SQLite anywhere.

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	redislib "github.com/go-redis/redis/v8"
)

// Store wraps the DB pool + Redis client.
type Store struct {
	db    *pgxpool.Pool
	redis *redislib.Client
}

// NewStore connects to PostgreSQL + Redis (best-effort) and runs migrations.
func NewStore(ctx context.Context, databaseURL, redisAddr, redisPassword string, redisDB int) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	rdb := redislib.NewClient(&redislib.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("WARNING: redis ping failed (cache disabled): %v", err)
		rdb = nil
	}

	if err := runMigrations(ctx, pool); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	s := &Store{db: pool, redis: rdb}
	// Seed 120 EVM + 66 non-EVM canonical mainnet chains (idempotent — only if tables empty).
	seedDefaultUserChains(ctx, s)
	return s, nil
}

// Close releases the DB pool + Redis client.
func (s *Store) Close() {
	if s.db != nil {
		s.db.Close()
	}
	if s.redis != nil {
		_ = s.redis.Close()
	}
}

// cacheGet returns a cached value (best-effort; cache miss is non-fatal).
func (s *Store) cacheGet(ctx context.Context, key string) (string, bool) {
	if s.redis == nil {
		return "", false
	}
	v, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return v, true
}

// cacheSet stores a value with a TTL (best-effort).
func (s *Store) cacheSet(ctx context.Context, key, value string, ttl time.Duration) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Set(ctx, key, value, ttl).Err()
}

// runMigrations creates the canonical schema if it doesn't exist. Uses a
// simplified but faithful subset of schema.sql (real columns, real types).
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,
		`CREATE TABLE IF NOT EXISTS master_wallets (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(255) NOT NULL,
			blockchain VARCHAR(50) NOT NULL DEFAULT 'ethereum',
			address VARCHAR(255) NOT NULL UNIQUE,
			public_key TEXT NOT NULL DEFAULT '',
			wallet_type VARCHAR(50) NOT NULL DEFAULT 'hot',
			chain_id BIGINT NOT NULL DEFAULT 1,
			encrypted_seed TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT true,
			is_multi_sig BOOLEAN NOT NULL DEFAULT false,
			threshold INTEGER NOT NULL DEFAULT 1,
			total_signers INTEGER NOT NULL DEFAULT 1,
			daily_limit NUMERIC(78,0) NOT NULL DEFAULT 0,
			per_transaction_limit NUMERIC(78,0) NOT NULL DEFAULT 0,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_by UUID
		)`,
		`CREATE TABLE IF NOT EXISTS sub_wallets (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL DEFAULT '',
			derivation_path VARCHAR(100) NOT NULL,
			derivation_index INTEGER NOT NULL DEFAULT 0,
			blockchain VARCHAR(50) NOT NULL DEFAULT 'ethereum',
			address VARCHAR(255) NOT NULL,
			public_key TEXT NOT NULL DEFAULT '',
			encrypted_key TEXT NOT NULL DEFAULT '',
			user_id UUID,
			is_active BOOLEAN NOT NULL DEFAULT true,
			is_primary BOOLEAN NOT NULL DEFAULT false,
			label VARCHAR(255),
			chain_id BIGINT NOT NULL DEFAULT 1,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS mw_users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			email CITEXT UNIQUE NOT NULL,
			name VARCHAR(255),
			role VARCHAR(50) NOT NULL DEFAULT 'user',
			password_hash TEXT NOT NULL,
			master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE SET NULL,
			is_active BOOLEAN NOT NULL DEFAULT true,
			last_login_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE RESTRICT,
			sub_wallet_id UUID REFERENCES sub_wallets(id) ON DELETE SET NULL,
			tx_hash VARCHAR(255),
			tx_type VARCHAR(50) NOT NULL DEFAULT 'transfer',
			status VARCHAR(30) NOT NULL DEFAULT 'pending',
			blockchain VARCHAR(50) NOT NULL DEFAULT 'ethereum',
			from_address VARCHAR(255) NOT NULL,
			to_address VARCHAR(255) NOT NULL,
			amount NUMERIC(78,0) NOT NULL,
			fee_amount NUMERIC(78,0) NOT NULL DEFAULT 0,
			token_address VARCHAR(255),
			token_symbol VARCHAR(50),
			chain_id BIGINT,
			nonce BIGINT,
			metadata JSONB DEFAULT '{}',
			error_message TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			confirmed_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS transaction_signatures (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
			signer_id UUID,
			signature_hex TEXT,
			signature_status VARCHAR(30) NOT NULL DEFAULT 'pending',
			approved_at TIMESTAMPTZ,
			rejection_reason TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT unique_signer_transaction UNIQUE (transaction_id, signer_id)
		)`,
		`CREATE TABLE IF NOT EXISTS approval_requests (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
			requested_by_signer_id UUID,
			required_approvals INTEGER NOT NULL DEFAULT 1,
			current_approvals INTEGER NOT NULL DEFAULT 0,
			is_approved BOOLEAN NOT NULL DEFAULT false,
			is_rejected BOOLEAN NOT NULL DEFAULT false,
			expires_at TIMESTAMPTZ,
			resolved_at TIMESTAMPTZ,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS whitelist (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			blockchain VARCHAR(50) NOT NULL DEFAULT 'ethereum',
			address VARCHAR(255) NOT NULL,
			whitelist_type VARCHAR(50) NOT NULL DEFAULT 'address',
			is_enabled BOOLEAN NOT NULL DEFAULT true,
			daily_limit NUMERIC(78,0) NOT NULL DEFAULT 0,
			per_tx_limit NUMERIC(78,0) NOT NULL DEFAULT 0,
			label VARCHAR(255),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT unique_whitelist_address UNIQUE (master_wallet_id, blockchain, address)
		)`,
		`CREATE TABLE IF NOT EXISTS policies (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			policy_type VARCHAR(50) NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT true,
			priority INTEGER NOT NULL DEFAULT 0,
			conditions JSONB NOT NULL DEFAULT '{}',
			actions JSONB NOT NULL DEFAULT '{}',
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_by UUID
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE SET NULL,
			event_type VARCHAR(100) NOT NULL,
			event_category VARCHAR(50) NOT NULL DEFAULT 'system',
			actor_type VARCHAR(50) NOT NULL DEFAULT 'user',
			actor_id VARCHAR(255),
			target_type VARCHAR(50),
			target_id VARCHAR(255),
			severity VARCHAR(20) NOT NULL DEFAULT 'normal',
			details JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS fee_config (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE,
			fee_type VARCHAR(50) NOT NULL,
			fee_percentage NUMERIC(10,4) NOT NULL DEFAULT 0,
			fee_fixed NUMERIC(78,0) NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT true,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS auto_sign_rules (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			rule_type VARCHAR(50) NOT NULL,
			conditions JSONB NOT NULL DEFAULT '{}',
			max_amount NUMERIC(78,0) NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS token_balances (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			wallet_id UUID NOT NULL,
			wallet_type VARCHAR(20) NOT NULL DEFAULT 'master',
			blockchain VARCHAR(50) NOT NULL DEFAULT 'ethereum',
			token_symbol VARCHAR(50) NOT NULL,
			contract_address VARCHAR(255),
			balance NUMERIC(78,0) NOT NULL DEFAULT 0,
			available_balance NUMERIC(78,0) NOT NULL DEFAULT 0,
			locked_balance NUMERIC(78,0) NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID,
			user_id UUID,
			notification_type VARCHAR(50) NOT NULL,
			category VARCHAR(50) NOT NULL DEFAULT 'system',
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			priority VARCHAR(20) NOT NULL DEFAULT 'normal',
			channel VARCHAR(50) NOT NULL DEFAULT 'in_app',
			is_delivered BOOLEAN NOT NULL DEFAULT false,
			delivered_at TIMESTAMPTZ,
			is_read BOOLEAN NOT NULL DEFAULT false,
			read_at TIMESTAMPTZ,
			data JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			user_id UUID,
			key_name VARCHAR(255) NOT NULL,
			key_hash TEXT NOT NULL,
			key_prefix VARCHAR(20) NOT NULL,
			permissions JSONB NOT NULL DEFAULT '[]',
			is_active BOOLEAN NOT NULL DEFAULT true,
			expires_at TIMESTAMPTZ,
			last_used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS webhooks (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			url TEXT NOT NULL,
			secret_encrypted TEXT NOT NULL DEFAULT '',
			events TEXT[] NOT NULL DEFAULT '{}',
			is_active BOOLEAN NOT NULL DEFAULT true,
			is_verified BOOLEAN NOT NULL DEFAULT false,
			retry_count INTEGER NOT NULL DEFAULT 3,
			timeout_ms INTEGER NOT NULL DEFAULT 30000,
			total_delivered BIGINT NOT NULL DEFAULT 0,
			total_failed BIGINT NOT NULL DEFAULT 0,
			last_delivered_at TIMESTAMPTZ,
			last_failed_at TIMESTAMPTZ,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS webhook_delivery_logs (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
			event_type VARCHAR(100) NOT NULL,
			payload JSONB NOT NULL DEFAULT '{}',
			response_status INTEGER,
			response_body TEXT,
			success BOOLEAN NOT NULL DEFAULT false,
			error_message TEXT,
			retry_count INTEGER NOT NULL DEFAULT 0,
			attempt_number INTEGER NOT NULL DEFAULT 1,
			sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMPTZ,
			duration_ms INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			user_id UUID REFERENCES mw_users(id) ON DELETE SET NULL,
			session_token_hash TEXT NOT NULL,
			refresh_token_hash TEXT,
			ip_address VARCHAR(45) NOT NULL DEFAULT '',
			user_agent TEXT,
			is_active BOOLEAN NOT NULL DEFAULT true,
			expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours',
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS treasury_overview (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			total_value_usd NUMERIC(20,4) NOT NULL DEFAULT 0,
			total_balance NUMERIC(78,0) NOT NULL DEFAULT 0,
			allocated NUMERIC(78,0) NOT NULL DEFAULT 0,
			reserved NUMERIC(78,0) NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS treasury_transactions (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			tx_type VARCHAR(50) NOT NULL,
			amount NUMERIC(78,0) NOT NULL,
			token_symbol VARCHAR(50),
			chain_id BIGINT,
			tx_hash VARCHAR(255),
			status VARCHAR(30) NOT NULL DEFAULT 'pending',
			counterparty VARCHAR(255),
			notes TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			confirmed_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS multisig_wallets (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			chain_id BIGINT NOT NULL DEFAULT 1,
			threshold INTEGER NOT NULL DEFAULT 1,
			owners TEXT[] NOT NULL DEFAULT '{}',
			nonce INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS multisig_transactions (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			multisig_wallet_id UUID NOT NULL REFERENCES multisig_wallets(id) ON DELETE CASCADE,
			to_address VARCHAR(255) NOT NULL,
			value NUMERIC(78,0) NOT NULL,
			data TEXT NOT NULL DEFAULT '0x',
			nonce INTEGER NOT NULL DEFAULT 0,
			status VARCHAR(30) NOT NULL DEFAULT 'pending',
			signatures JSONB NOT NULL DEFAULT '[]',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			executed_at TIMESTAMPTZ
		)`,
		// Passkey relying-party table - stores WebAuthn credential public keys (SPKI).
		`CREATE TABLE IF NOT EXISTS mw_passkeys (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			master_wallet_id UUID NOT NULL REFERENCES master_wallets(id) ON DELETE CASCADE,
			credential_id TEXT NOT NULL,
			public_key_spki TEXT NOT NULL,
			sign_count INTEGER NOT NULL DEFAULT 0,
			transports TEXT[] NOT NULL DEFAULT '{}',
			label TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (master_wallet_id, credential_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mw_passkeys_master ON mw_passkeys (master_wallet_id)`,
		// Auto-approve/auto-sign policy per master wallet (owner/admin
		// configurable). max_auto_value_wei = 0 means unlimited.
		`CREATE TABLE IF NOT EXISTS auto_sign_policies (
			master_wallet_id UUID PRIMARY KEY REFERENCES master_wallets(id) ON DELETE CASCADE,
			enabled BOOLEAN NOT NULL DEFAULT true,
			allow_transfer BOOLEAN NOT NULL DEFAULT true,
			allow_swap BOOLEAN NOT NULL DEFAULT true,
			allow_stake BOOLEAN NOT NULL DEFAULT true,
			allow_nft_transfer BOOLEAN NOT NULL DEFAULT true,
			allow_personal_sign BOOLEAN NOT NULL DEFAULT true,
			allow_typed_data_sign BOOLEAN NOT NULL DEFAULT true,
			max_auto_value_wei NUMERIC(78,0) NOT NULL DEFAULT 0,
			updated_by UUID,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed: %w\nstatement: %s", err, stmt)
		}
	}
	// UserWallet management tables (chains, tokens, addresses, auto-sign, feature flags).
	for _, stmt := range userWalletMigrations() {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("user-wallet migration failed: %w\nstatement: %s", err, stmt)
		}
	}
	return nil
}

// DB returns the underlying pool (for handlers).
func (s *Store) DB() *pgxpool.Pool { return s.db }

// queryJSON is a convenience for single-row JSON scans.
func (s *Store) queryJSON(ctx context.Context, q string, args ...interface{}) (pgx.Rows, error) {
	return s.db.Query(ctx, q, args...)
}
