package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/tigerwallet/billing/internal/config"
	"github.com/tigerwallet/billing/internal/models"
)

var Pool *pgxpool.Pool
var RedisClient *redis.Client
var ctx = context.Background()

// Initialize sets up database connections
func Initialize(cfg *config.Config) error {
	// Connect to PostgreSQL
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = cfg.Database.MaxConns
	poolConfig.MinConns = cfg.Database.MinConns
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	Pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := Pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Connect to Redis
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})

	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Seed default plans
	if err := seedDefaultPlans(); err != nil {
		return fmt.Errorf("failed to seed default plans: %w", err)
	}

	return nil
}

// Close closes all database connections
func Close() {
	if Pool != nil {
		Pool.Close()
	}
	if RedisClient != nil {
		RedisClient.Close()
	}
}

// runMigrations creates all necessary tables
func runMigrations() error {
	migrations := []string{
		// Plans table
		`CREATE TABLE IF NOT EXISTS plans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL UNIQUE,
			tier VARCHAR(50) NOT NULL,
			description TEXT,
			price_monthly BIGINT NOT NULL DEFAULT 0,
			price_yearly BIGINT NOT NULL DEFAULT 0,
			feature_flags TEXT[] DEFAULT '{}',
			api_quota_monthly BIGINT NOT NULL DEFAULT 0,
			storage_quota_gb BIGINT NOT NULL DEFAULT 0,
			max_users INT NOT NULL DEFAULT 1,
			max_wallets INT NOT NULL DEFAULT 0,
			max_bots INT NOT NULL DEFAULT 0,
			support_level VARCHAR(100) NOT NULL DEFAULT 'email',
			is_active BOOLEAN NOT NULL DEFAULT true,
			stripe_price_id VARCHAR(255),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Subscriptions table
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			plan_id UUID NOT NULL REFERENCES plans(id),
			stripe_subscription_id VARCHAR(255) UNIQUE,
			status VARCHAR(50) NOT NULL DEFAULT 'incomplete',
			current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
			current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
			trial_start TIMESTAMP WITH TIME ZONE,
			trial_end TIMESTAMP WITH TIME ZONE,
			cancel_at_period_end BOOLEAN NOT NULL DEFAULT false,
			canceled_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(tenant_id)
		)`,

		// Tenants table (white label clients)
		`CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			stripe_customer_id VARCHAR(255),
			subscription_id UUID REFERENCES subscriptions(id),
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Usage records table
		`CREATE TABLE IF NOT EXISTS usage_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			api_method VARCHAR(100) NOT NULL,
			count BIGINT NOT NULL DEFAULT 1,
			period_start TIMESTAMP WITH TIME ZONE NOT NULL,
			period_end TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Invoices table
		`CREATE TABLE IF NOT EXISTS invoices (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			invoice_number VARCHAR(100) NOT NULL UNIQUE,
			stripe_invoice_id VARCHAR(255) UNIQUE,
			amount BIGINT NOT NULL DEFAULT 0,
			amount_due BIGINT NOT NULL DEFAULT 0,
			amount_paid BIGINT NOT NULL DEFAULT 0,
			currency VARCHAR(10) NOT NULL DEFAULT 'usd',
			status VARCHAR(50) NOT NULL DEFAULT 'draft',
			due_date TIMESTAMP WITH TIME ZONE NOT NULL,
			paid_at TIMESTAMP WITH TIME ZONE,
			invoice_url TEXT,
			invoice_pdf TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Line items table
		`CREATE TABLE IF NOT EXISTS line_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
			description TEXT NOT NULL,
			quantity BIGINT NOT NULL DEFAULT 1,
			unit_price BIGINT NOT NULL DEFAULT 0,
			amount BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Payments table
		`CREATE TABLE IF NOT EXISTS payments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
			stripe_payment_id VARCHAR(255) UNIQUE,
			amount BIGINT NOT NULL DEFAULT 0,
			currency VARCHAR(10) NOT NULL DEFAULT 'usd',
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			payment_method VARCHAR(50),
			payment_method_id VARCHAR(255),
			receipt_url TEXT,
			failure_reason TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Payment methods table
		`CREATE TABLE IF NOT EXISTS payment_methods (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			stripe_payment_id VARCHAR(255) NOT NULL UNIQUE,
			type VARCHAR(50) NOT NULL,
			card_brand VARCHAR(50),
			last4 VARCHAR(4),
			exp_month INT,
			exp_year INT,
			is_default BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// API keys table
		`CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			key VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			permissions TEXT[] DEFAULT '{}',
			rate_limit INT NOT NULL DEFAULT 60,
			is_active BOOLEAN NOT NULL DEFAULT true,
			last_used_at TIMESTAMP WITH TIME ZONE,
			expires_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Webhooks table
		`CREATE TABLE IF NOT EXISTS webhooks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			url TEXT NOT NULL,
			events TEXT[] NOT NULL,
			secret VARCHAR(255) NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT true,
			failure_count INT NOT NULL DEFAULT 0,
			last_failure_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// API requests logging table
		`CREATE TABLE IF NOT EXISTS api_requests (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			api_key_id UUID REFERENCES api_keys(id) ON DELETE SET NULL,
			method VARCHAR(10) NOT NULL,
			path TEXT NOT NULL,
			status_code INT NOT NULL,
			latency_ms BIGINT NOT NULL DEFAULT 0,
			ip_address VARCHAR(45),
			user_agent TEXT,
			request_size BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Feature flags table
		`CREATE TABLE IF NOT EXISTS feature_flags (
			name VARCHAR(100) PRIMARY KEY,
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Create indexes
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant ON subscriptions(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_records_tenant_period ON usage_records(tenant_id, period_start, period_end)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_tenant ON invoices(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_tenant ON payments(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key)`,
		`CREATE INDEX IF NOT EXISTS idx_api_requests_tenant_created ON api_requests(tenant_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug)`,
	}

	for _, migration := range migrations {
		if _, err := Pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// seedDefaultPlans creates default subscription plans
func seedDefaultPlans() error {
	plans := []models.Plan{
		{
			Name:             "Free",
			Tier:             models.TierFree,
			Description:      "Free tier for testing and small projects",
			PriceMonthly:     0,
			PriceYearly:      0,
			FeatureFlags:     []string{"basic_wallet", "1_product"},
			APIQuotaMonthly:  1000,
			StorageQuotaGB:   1,
			MaxUsers:         1,
			MaxWallets:       5,
			MaxBots:          1,
			SupportLevel:     "community",
			IsActive:         true,
		},
		{
			Name:             "Basic",
			Tier:             models.TierBasic,
			Description:      "Basic tier for small teams",
			PriceMonthly:     9900, // $99/month
			PriceYearly:      99000, // $990/year
			FeatureFlags:     []string{"basic_wallet", "all_products", "email_support"},
			APIQuotaMonthly:  50000,
			StorageQuotaGB:   10,
			MaxUsers:         5,
			MaxWallets:       50,
			MaxBots:          10,
			SupportLevel:     "email",
			IsActive:         true,
		},
		{
			Name:             "Pro",
			Tier:             models.TierPro,
			Description:      "Professional tier for growing businesses",
			PriceMonthly:     29900, // $299/month
			PriceYearly:      299000, // $2990/year
			FeatureFlags:     []string{"all_features", "priority_support", "custom_branding"},
			APIQuotaMonthly:  500000,
			StorageQuotaGB:   100,
			MaxUsers:         25,
			MaxWallets:       500,
			MaxBots:          100,
			SupportLevel:     "priority",
			IsActive:         true,
		},
		{
			Name:             "Enterprise",
			Tier:             models.TierEnterprise,
			Description:      "Enterprise tier for large organizations",
			PriceMonthly:     99900, // $999/month
			PriceYearly:      999000, // $9990/year
			FeatureFlags:     []string{"all_features", "dedicated_support", "sla", "custom_contracts"},
			APIQuotaMonthly:  -1, // unlimited
			StorageQuotaGB:   -1, // unlimited
			MaxUsers:         -1, // unlimited
			MaxWallets:       -1,
			MaxBots:          -1,
			SupportLevel:     "dedicated",
			IsActive:         true,
		},
	}

	for _, plan := range plans {
		_, err := Pool.Exec(ctx, `
			INSERT INTO plans (name, tier, description, price_monthly, price_yearly, feature_flags, 
				api_quota_monthly, storage_quota_gb, max_users, max_wallets, max_bots, 
				support_level, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (name) DO NOTHING
		`, plan.Name, plan.Tier, plan.Description, plan.PriceMonthly, plan.PriceYearly,
			plan.FeatureFlags, plan.APIQuotaMonthly, plan.StorageQuotaGB, plan.MaxUsers,
			plan.MaxWallets, plan.MaxBots, plan.SupportLevel, plan.IsActive)

		if err != nil {
			return err
		}
	}

	return nil
}
