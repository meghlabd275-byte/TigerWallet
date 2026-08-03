// Database Migrations for TigerWallet
// Run migrations using: go run migrate.go

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	host     = flag.String("host", "localhost", "Database host")
	port     = flag.String("port", "5432", "Database port")
	user     = flag.String("user", "tigerwallet", "Database user")
	password = flag.String("password", "tigerwallet", "Database password")
	dbname   = flag.String("dbname", "tigerwallet", "Database name")
	direction = flag.String("direction", "up", "Migration direction: up or down")
	step     = flag.Int("step", 0, "Number of migrations to run (0 = all)")
)

type Migration struct {
	Version string
	Name    string
	Up      string
	Down    string
}

var migrations = []Migration{
	{
		Version: "001",
		Name:    "create_extensions",
		Up: `
			CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
			CREATE EXTENSION IF NOT EXISTS "pgcrypto";
			CREATE EXTENSION IF NOT EXISTS "citext";
		`,
		Down: `
			DROP EXTENSION IF EXISTS "uuid-ossp";
			DROP EXTENSION IF EXISTS "pgcrypto";
			DROP EXTENSION IF EXISTS "citext";
		`,
	},
	{
		Version: "002",
		Name:    "create_enums",
		Up: `
			CREATE TYPE admin_role AS ENUM ('super_admin', 'admin', 'manager', 'support');
			CREATE TYPE admin_status AS ENUM ('active', 'suspended', 'blocked', 'pending');
			CREATE TYPE wl_status AS ENUM ('pending', 'active', 'suspended', 'revoked', 'destroyed');
			CREATE TYPE user_kyc_status AS ENUM ('none', 'pending', 'under_review', 'approved', 'rejected');
			CREATE TYPE transaction_type AS ENUM ('deposit', 'withdrawal', 'transfer', 'swap', 'trade', 'fee', 'reward', 'other');
			CREATE TYPE transaction_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');
			CREATE TYPE pair_status AS ENUM ('active', 'suspended', 'halted', 'delisted');
			CREATE TYPE fee_type AS ENUM ('deposit', 'withdrawal', 'trading', 'api', 'network');
		`,
		Down: `
			DROP TYPE IF EXISTS admin_role;
			DROP TYPE IF EXISTS admin_status;
			DROP TYPE IF EXISTS wl_status;
			DROP TYPE IF EXISTS user_kyc_status;
			DROP TYPE IF EXISTS transaction_type;
			DROP TYPE IF EXISTS transaction_status;
			DROP TYPE IF EXISTS pair_status;
			DROP TYPE IF EXISTS fee_type;
		`,
	},
	{
		Version: "003",
		Name:    "create_admin_users",
		Up: `
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
		`,
		Down: `DROP TABLE IF EXISTS admin_users;`,
	},
	{
		Version: "004",
		Name:    "create_white_labels",
		Up: `
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
				branding JSONB DEFAULT '{}'::jsonb,
				feature_flags JSONB DEFAULT '{}'::jsonb,
				max_users INTEGER DEFAULT 1000,
				current_users INTEGER DEFAULT 0,
				total_revenue DECIMAL(20,8) DEFAULT 0,
				created_by UUID REFERENCES admin_users(id),
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW()
			);
			CREATE INDEX idx_wl_domain ON white_labels(domain);
			CREATE INDEX idx_wl_status ON white_labels(status);
		`,
		Down: `DROP TABLE IF EXISTS white_labels;`,
	},
	{
		Version: "005",
		Name:    "create_users",
		Up: `
			CREATE TABLE users (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				email CITEXT UNIQUE NOT NULL,
				username VARCHAR(100) UNIQUE,
				wallet_address VARCHAR(100),
				kyc_status user_kyc_status DEFAULT 'none',
				kyc_data JSONB DEFAULT '{}'::jsonb,
				risk_score INTEGER DEFAULT 0,
				status VARCHAR(50) DEFAULT 'active',
				referral_code VARCHAR(50) UNIQUE,
				referrer_id UUID REFERENCES users(id),
				whitelabel_id UUID REFERENCES white_labels(id),
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW()
			);
			CREATE INDEX idx_users_email ON users(email);
			CREATE INDEX idx_users_wallet ON users(wallet_address);
			CREATE INDEX idx_users_kyc ON users(kyc_status);
			CREATE INDEX idx_users_wl ON users(whitelabel_id);
		`,
		Down: `DROP TABLE IF EXISTS users;`,
	},
	{
		Version: "006",
		Name:    "create_blockchains",
		Up: `
			CREATE TABLE blockchains (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				symbol VARCHAR(20) NOT NULL,
				chain_type VARCHAR(50) NOT NULL,
				chain_id BIGINT UNIQUE,
				decimals INTEGER DEFAULT 18,
				rpc_urls JSONB DEFAULT '[]'::jsonb,
				explorer_urls JSONB DEFAULT '[]'::jsonb,
				icon_url VARCHAR(500),
				is_active BOOLEAN DEFAULT TRUE,
				is_testnet BOOLEAN DEFAULT FALSE,
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW()
			);
			CREATE INDEX idx_blockchains_chain_id ON blockchains(chain_id);
			CREATE INDEX idx_blockchains_type ON blockchains(chain_type);
		`,
		Down: `DROP TABLE IF EXISTS blockchains;`,
	},
	{
		Version: "007",
		Name:    "create_tokens",
		Up: `
			CREATE TABLE tokens (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				blockchain_id INTEGER REFERENCES blockchains(id),
				contract_address VARCHAR(100),
				name VARCHAR(255) NOT NULL,
				symbol VARCHAR(50) NOT NULL,
				decimals INTEGER NOT NULL,
				total_supply DECIMAL(30,0),
				is_native BOOLEAN DEFAULT FALSE,
				is_verified BOOLEAN DEFAULT FALSE,
				logo_url VARCHAR(500),
				price_feed_url VARCHAR(500),
				is_active BOOLEAN DEFAULT TRUE,
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
				UNIQUE(blockchain_id, contract_address)
			);
			CREATE INDEX idx_tokens_symbol ON tokens(symbol);
			CREATE INDEX idx_tokens_blockchain ON tokens(blockchain_id);
		`,
		Down: `DROP TABLE IF EXISTS tokens;`,
	},
	{
		Version: "008",
		Name:    "create_trading_pairs",
		Up: `
			CREATE TABLE trading_pairs (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				base_token_id UUID REFERENCES tokens(id),
				quote_token_id UUID REFERENCES tokens(id),
				blockchain_id INTEGER REFERENCES blockchains(id),
				pair_address VARCHAR(100),
				factory_address VARCHAR(100),
				status pair_status DEFAULT 'active',
				maker_fee DECIMAL(10,8) DEFAULT 0.001,
				taker_fee DECIMAL(10,8) DEFAULT 0.001,
				min_trade_amount DECIMAL(30,0) DEFAULT 0,
				max_trade_amount DECIMAL(30,0),
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
				UNIQUE(base_token_id, quote_token_id, blockchain_id)
			);
			CREATE INDEX idx_pairs_status ON trading_pairs(status);
			CREATE INDEX idx_pairs_blockchain ON trading_pairs(blockchain_id);
		`,
		Down: `DROP TABLE IF EXISTS trading_pairs;`,
	},
	{
		Version: "009",
		Name:    "create_transactions",
		Up: `
			CREATE TABLE transactions (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				user_id UUID REFERENCES users(id),
				type transaction_type NOT NULL,
				status transaction_status DEFAULT 'pending',
				token_id UUID REFERENCES tokens(id),
				amount DECIMAL(40,0) NOT NULL,
				fee DECIMAL(40,0) DEFAULT 0,
				tx_hash VARCHAR(200),
				from_address VARCHAR(100),
				to_address VARCHAR(100),
				blockchain_id INTEGER REFERENCES blockchains(id),
				metadata JSONB DEFAULT '{}'::jsonb,
				whitelabel_id UUID REFERENCES white_labels(id),
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW()
			);
			CREATE INDEX idx_tx_user ON transactions(user_id);
			CREATE INDEX idx_tx_type ON transactions(type);
			CREATE INDEX idx_tx_status ON transactions(status);
			CREATE INDEX idx_tx_hash ON transactions(tx_hash);
			CREATE INDEX idx_tx_wl ON transactions(whitelabel_id);
		`,
		Down: `DROP TABLE IF EXISTS transactions;`,
	},
	{
		Version: "010",
		Name:    "create_audit_logs",
		Up: `
			CREATE TABLE audit_logs (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				user_id UUID,
				admin_id UUID REFERENCES admin_users(id),
				action VARCHAR(100) NOT NULL,
				entity_type VARCHAR(100),
				entity_id UUID,
				details JSONB DEFAULT '{}'::jsonb,
				ip_address INET,
				user_agent TEXT,
				created_at TIMESTAMP NOT NULL DEFAULT NOW()
			);
			CREATE INDEX idx_audit_admin ON audit_logs(admin_id);
			CREATE INDEX idx_audit_action ON audit_logs(action);
			CREATE INDEX idx_audit_created ON audit_logs(created_at);
		`,
		Down: `DROP TABLE IF EXISTS audit_logs;`,
	},
	{
		Version: "011",
		Name:    "create_api_keys",
		Up: `
			CREATE TABLE api_keys (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				user_id UUID REFERENCES users(id),
				name VARCHAR(100) NOT NULL,
				key_hash VARCHAR(255) NOT NULL,
				permissions JSONB DEFAULT '[]'::jsonb,
				rate_limit_minute INTEGER DEFAULT 60,
				rate_limit_day INTEGER DEFAULT 10000,
				is_active BOOLEAN DEFAULT TRUE,
				last_used TIMESTAMP,
				expires_at TIMESTAMP,
				created_at TIMESTAMP NOT NULL DEFAULT NOW()
			);
			CREATE INDEX idx_api_keys_user ON api_keys(user_id);
			CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
		`,
		Down: `DROP TABLE IF EXISTS api_keys;`,
	},
	{
		Version: "012",
		Name:    "create_products",
		Up: `
			CREATE TABLE products (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				name VARCHAR(255) NOT NULL,
				type VARCHAR(50) NOT NULL,
				description TEXT,
				status VARCHAR(50) DEFAULT 'enabled',
				fee_percent DECIMAL(10,8) DEFAULT 0,
				min_deposit DECIMAL(40,0) DEFAULT 0,
				max_deposit DECIMAL(40,0),
				features JSONB DEFAULT '[]'::jsonb,
				whitelabel_id UUID REFERENCES white_labels(id),
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW()
			);
			CREATE INDEX idx_products_type ON products(type);
			CREATE INDEX idx_products_wl ON products(whitelabel_id);
		`,
		Down: `DROP TABLE IF EXISTS products;`,
	},
}

func main() {
	flag.Parse()

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		*user, *password, *host, *port, *dbname)

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Create migrations table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(10) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	if *direction == "up" {
		runMigrationsUp(ctx, pool)
	} else {
		runMigrationsDown(ctx, pool)
	}
}

func runMigrationsUp(ctx context.Context, pool *pgxpool.Pool) {
	for i, m := range migrations {
		if *step > 0 && i >= *step {
			break
		}

		// Check if already applied
		var exists bool
		err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", m.Version).Scan(&exists)
		if err != nil {
			log.Printf("Error checking migration %s: %v", m.Version, err)
			continue
		}

		if exists {
			log.Printf("Migration %s already applied, skipping", m.Version)
			continue
		}

		// Run migration
		log.Printf("Applying migration %s: %s", m.Version, m.Name)
		
		// Split and execute statements
		statements := strings.Split(m.Up, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			_, err := pool.Exec(ctx, stmt)
			if err != nil {
				log.Printf("Error in migration %s: %v", m.Version, err)
				log.Printf("Statement: %s", stmt)
			}
		}

		// Record migration
		_, err = pool.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", m.Version, m.Name)
		if err != nil {
			log.Printf("Error recording migration %s: %v", m.Version, err)
		}

		log.Printf("Migration %s applied successfully", m.Version)
	}
}

func runMigrationsDown(ctx context.Context, pool *pgxpool.Pool) {
	for i := len(migrations) - 1; i >= 0; i-- {
		if *step > 0 && len(migrations)-1-i >= *step {
			break
		}

		m := migrations[i]

		// Check if applied
		var exists bool
		err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", m.Version).Scan(&exists)
		if err != nil || !exists {
			log.Printf("Migration %s not applied, skipping", m.Version)
			continue
		}

		// Run rollback
		log.Printf("Rolling back migration %s: %s", m.Version, m.Name)
		
		statements := strings.Split(m.Down, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			_, err := pool.Exec(ctx, stmt)
			if err != nil {
				log.Printf("Error in rollback %s: %v", m.Version, err)
			}
		}

		// Remove migration record
		_, err = pool.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", m.Version)
		if err != nil {
			log.Printf("Error removing migration record %s: %v", m.Version, err)
		}

		log.Printf("Migration %s rolled back successfully", m.Version)
	}
}
