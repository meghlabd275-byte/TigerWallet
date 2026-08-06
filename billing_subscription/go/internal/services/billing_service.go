package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/billing/internal/database"
	"github.com/tigerwallet/billing/internal/models"
)

type BillingService struct{}

func NewBillingService() *BillingService {
	return &BillingService{}
}

// Plan operations

func (s *BillingService) GetAllPlans(ctx context.Context) ([]models.Plan, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT id, name, tier, description, price_monthly, price_yearly, feature_flags,
			api_quota_monthly, storage_quota_gb, max_users, max_wallets, max_bots,
			support_level, is_active, stripe_price_id, created_at, updated_at
		FROM plans WHERE is_active = true ORDER BY price_monthly ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []models.Plan
	for rows.Next() {
		var plan models.Plan
		err := rows.Scan(
			&plan.ID, &plan.Name, &plan.Tier, &plan.Description, &plan.PriceMonthly,
			&plan.PriceYearly, &plan.FeatureFlags, &plan.APIQuotaMonthly, &plan.StorageQuotaGB,
			&plan.MaxUsers, &plan.MaxWallets, &plan.MaxBots, &plan.SupportLevel,
			&plan.IsActive, &plan.StripePriceID, &plan.CreatedAt, &plan.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func (s *BillingService) GetPlanByID(ctx context.Context, planID uuid.UUID) (*models.Plan, error) {
	var plan models.Plan
	err := database.Pool.QueryRow(ctx, `
		SELECT id, name, tier, description, price_monthly, price_yearly, feature_flags,
			api_quota_monthly, storage_quota_gb, max_users, max_wallets, max_bots,
			support_level, is_active, stripe_price_id, created_at, updated_at
		FROM plans WHERE id = $1
	`).Scan(
		&plan.ID, &plan.Name, &plan.Tier, &plan.Description, &plan.PriceMonthly,
		&plan.PriceYearly, &plan.FeatureFlags, &plan.APIQuotaMonthly, &plan.StorageQuotaGB,
		&plan.MaxUsers, &plan.MaxWallets, &plan.MaxBots, &plan.SupportLevel,
		&plan.IsActive, &plan.StripePriceID, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (s *BillingService) GetPlanByTier(ctx context.Context, tier models.SubscriptionTier) (*models.Plan, error) {
	var plan models.Plan
	err := database.Pool.QueryRow(ctx, `
		SELECT id, name, tier, description, price_monthly, price_yearly, feature_flags,
			api_quota_monthly, storage_quota_gb, max_users, max_wallets, max_bots,
			support_level, is_active, stripe_price_id, created_at, updated_at
		FROM plans WHERE tier = $1 AND is_active = true
	`).Scan(
		&plan.ID, &plan.Name, &plan.Tier, &plan.Description, &plan.PriceMonthly,
		&plan.PriceYearly, &plan.FeatureFlags, &plan.APIQuotaMonthly, &plan.StorageQuotaGB,
		&plan.MaxUsers, &plan.MaxWallets, &plan.MaxBots, &plan.SupportLevel,
		&plan.IsActive, &plan.StripePriceID, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// Tenant operations

func (s *BillingService) CreateTenant(ctx context.Context, name, email, slug string) (*models.Tenant, error) {
	tenant := &models.Tenant{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		Email:     email,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Get free plan
	freePlan, err := s.GetPlanByTier(ctx, models.TierFree)
	if err != nil {
		return nil, fmt.Errorf("failed to get free plan: %w", err)
	}

	// Create subscription
	subscription := &models.Subscription{
		ID:                 uuid.New(),
		TenantID:           tenant.ID,
		PlanID:             freePlan.ID,
		Status:             models.SubStatusActive,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Insert subscription
	_, err = tx.Exec(ctx, `
		INSERT INTO subscriptions (id, tenant_id, plan_id, status, current_period_start, current_period_end, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, subscription.ID, subscription.TenantID, subscription.PlanID, subscription.Status,
		subscription.CurrentPeriodStart, subscription.CurrentPeriodEnd, subscription.CreatedAt, subscription.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Insert tenant
	_, err = tx.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, email, subscription_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, tenant.ID, tenant.Name, tenant.Slug, tenant.Email, subscription.ID, tenant.Status, tenant.CreatedAt, tenant.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	tenant.SubscriptionID = &subscription.ID
	return tenant, nil
}

func (s *BillingService) GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	err := database.Pool.QueryRow(ctx, `
		SELECT id, name, slug, email, stripe_customer_id, subscription_id, status, created_at, updated_at
		FROM tenants WHERE id = $1
	`).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Email, &tenant.StripeCustomerID,
		&tenant.SubscriptionID, &tenant.Status, &tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Get subscription with plan
	if tenant.SubscriptionID != nil {
		subscription, err := s.GetSubscriptionByID(ctx, *tenant.SubscriptionID)
		if err == nil {
			tenant.SubscriptionID = &subscription.ID
		}
	}

	return &tenant, nil
}

func (s *BillingService) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := database.Pool.QueryRow(ctx, `
		SELECT id, name, slug, email, stripe_customer_id, subscription_id, status, created_at, updated_at
		FROM tenants WHERE slug = $1
	`).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Email, &tenant.StripeCustomerID,
		&tenant.SubscriptionID, &tenant.Status, &tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *BillingService) UpdateTenantStatus(ctx context.Context, tenantID uuid.UUID, status string) error {
	_, err := database.Pool.Exec(ctx, `
		UPDATE tenants SET status = $1, updated_at = $2 WHERE id = $3
	`, status, time.Now(), tenantID)
	return err
}

// Subscription operations

func (s *BillingService) GetSubscriptionByID(ctx context.Context, subID uuid.UUID) (*models.Subscription, error) {
	var sub models.Subscription
	err := database.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, plan_id, stripe_subscription_id, status, 
			current_period_start, current_period_end, trial_start, trial_end,
			cancel_at_period_end, canceled_at, created_at, updated_at
		FROM subscriptions WHERE id = $1
	`).Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &sub.StripeSubscriptionID, &sub.Status,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialStart, &sub.TrialEnd,
		&sub.CancelAtPeriodEnd, &sub.CanceledAt, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	plan, err := s.GetPlanByID(ctx, sub.PlanID)
	if err == nil {
		sub.Plan = plan
	}

	return &sub, nil
}

func (s *BillingService) GetSubscriptionByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.Subscription, error) {
	var sub models.Subscription
	err := database.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, plan_id, stripe_subscription_id, status, 
			current_period_start, current_period_end, trial_start, trial_end,
			cancel_at_period_end, canceled_at, created_at, updated_at
		FROM subscriptions WHERE tenant_id = $1
	`, tenantID).Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &sub.StripeSubscriptionID, &sub.Status,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialStart, &sub.TrialEnd,
		&sub.CancelAtPeriodEnd, &sub.CanceledAt, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	plan, err := s.GetPlanByID(ctx, sub.PlanID)
	if err == nil {
		sub.Plan = plan
	}

	return &sub, nil
}

func (s *BillingService) UpdateSubscriptionStatus(ctx context.Context, subID uuid.UUID, status models.SubscriptionStatus) error {
	_, err := database.Pool.Exec(ctx, `
		UPDATE subscriptions SET status = $1, updated_at = $2 WHERE id = $3
	`, status, time.Now(), subID)
	return err
}

func (s *BillingService) UpdateSubscriptionPeriod(ctx context.Context, subID uuid.UUID, start, end time.Time) error {
	_, err := database.Pool.Exec(ctx, `
		UPDATE subscriptions SET current_period_start = $1, current_period_end = $2, updated_at = $3 WHERE id = $4
	`, start, end, time.Now(), subID)
	return err
}

// Usage tracking

func (s *BillingService) RecordUsage(ctx context.Context, tenantID uuid.UUID, apiMethod string, count int64) error {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0)

	// Try to update existing record
	result, err := database.Pool.Exec(ctx, `
		UPDATE usage_records 
		SET count = count + $1, updated_at = $2
		WHERE tenant_id = $3 AND api_method = $4 AND period_start = $5 AND period_end = $6
	`, count, now, tenantID, apiMethod, periodStart, periodEnd)
	if err != nil {
		return err
	}

	// If no rows updated, insert new record
	if result.RowsAffected() == 0 {
		_, err = database.Pool.Exec(ctx, `
			INSERT INTO usage_records (tenant_id, api_method, count, period_start, period_end, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, tenantID, apiMethod, count, periodStart, periodEnd, now)
	}

	return err
}

func (s *BillingService) GetUsageSummary(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*models.UsageSummary, error) {
	var summary models.UsageSummary
	summary.TenantID = tenantID
	summary.PeriodStart = periodStart
	summary.PeriodEnd = periodEnd

	// Get API calls
	err := database.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(count), 0) FROM usage_records 
		WHERE tenant_id = $1 AND period_start >= $2 AND period_end <= $3
	`, tenantID, periodStart, periodEnd).Scan(&summary.TotalAPICalls)
	if err != nil {
		return nil, err
	}

	// Get active counts
	err = database.Pool.QueryRow(ctx, `
		SELECT 
			(SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND created_at <= $2),
			(SELECT COUNT(*) FROM wallets WHERE tenant_id = $1 AND created_at <= $2),
			(SELECT COUNT(*) FROM bots WHERE tenant_id = $1 AND created_at <= $2)
	`, tenantID, periodEnd).Scan(&summary.ActiveUsers, &summary.ActiveWallets, &summary.ActiveBots)
	if err != nil {
		// Tables might not exist, set defaults
		summary.ActiveUsers = 0
		summary.ActiveWallets = 0
		summary.ActiveBots = 0
	}

	// Get subscription to calculate overages
	sub, err := s.GetSubscriptionByTenantID(ctx, tenantID)
	if err == nil && sub.Plan != nil {
		// Check API overage
		if sub.Plan.APIQuotaMonthly > 0 && summary.TotalAPICalls > sub.Plan.APIQuotaMonthly {
			summary.OverageAPICalls = summary.TotalAPICalls - sub.Plan.APIQuotaMonthly
		}
	}

	// Cache in Redis
	cacheKey := fmt.Sprintf("usage:%s:%s:%s", tenantID.String(), periodStart.Format("2006-01"), periodEnd.Format("2006-01"))
	data, _ := json.Marshal(summary)
	database.RedisClient.Set(ctx, cacheKey, data, 24*time.Hour)

	return &summary, nil
}

func (s *BillingService) CheckQuota(ctx context.Context, tenantID uuid.UUID, resourceType string) (bool, error) {
	sub, err := s.GetSubscriptionByTenantID(ctx, tenantID)
	if err != nil {
		return false, err
	}

	if sub.Plan == nil {
		return false, fmt.Errorf("no plan found")
	}

	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0)

	summary, err := s.GetUsageSummary(ctx, tenantID, periodStart, periodEnd)
	if err != nil {
		return false, err
	}

	switch resourceType {
	case "api":
		if sub.Plan.APIQuotaMonthly < 0 {
			return true, nil // unlimited
		}
		return summary.TotalAPICalls < sub.Plan.APIQuotaMonthly, nil
	case "wallets":
		if sub.Plan.MaxWallets < 0 {
			return true, nil
		}
		return int64(summary.ActiveWallets) < int64(sub.Plan.MaxWallets), nil
	case "bots":
		if sub.Plan.MaxBots < 0 {
			return true, nil
		}
		return int64(summary.ActiveBots) < int64(sub.Plan.MaxBots), nil
	case "users":
		if sub.Plan.MaxUsers < 0 {
			return true, nil
		}
		return int64(summary.ActiveUsers) < int64(sub.Plan.MaxUsers), nil
	}

	return false, fmt.Errorf("unknown resource type: %s", resourceType)
}
