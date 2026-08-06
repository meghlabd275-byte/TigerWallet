package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/tigerwallet/multi-tenancy/internal/models"
)

var Pool *pgxpool.Pool
var RedisClient *redis.Client

type TenantService struct{}

func NewTenantService() *TenantService {
	return &TenantService{}
}

// Initialize creates database tables
func Initialize(pool *pgxpool.Pool, redis *redis.Client) {
	Pool = pool
	RedisClient = redis
}

// Tenant CRUD operations

func (s *TenantService) CreateTenant(ctx context.Context, name, email, slug string) (*models.Tenant, error) {
	tenant := &models.Tenant{
		ID:          uuid.New(),
		Name:        name,
		Slug:        slug,
		Email:       email,
		Status:      models.TenantStatusActive,
		Timezone:    "UTC",
		Language:    "en",
		Features:    []string{},
		Metadata:    map[string]interface{}{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		TrialEndsAt: func() *time.Time { t := time.Now().Add(14 * 24 * time.Hour); return &t }(),
	}

	// Get free plan
	var planID uuid.UUID
	err := Pool.QueryRow(ctx, `SELECT id FROM plans WHERE tier = 'free' LIMIT 1`).Scan(&planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get free plan: %w", err)
	}
	tenant.PlanID = planID

	_, err = Pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, email, status, plan_id, timezone, language, features, metadata, created_at, updated_at, trial_ends_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, tenant.ID, tenant.Name, tenant.Slug, tenant.Email, tenant.Status, tenant.PlanID,
		tenant.Timezone, tenant.Language, tenant.Features, tenant.Metadata, tenant.CreatedAt, tenant.UpdatedAt, tenant.TrialEndsAt)

	if err != nil {
		return nil, err
	}

	// Create default quotas
	if err := s.createDefaultQuotas(ctx, tenant.ID); err != nil {
		return nil, err
	}

	// Create default config
	if err := s.createDefaultConfig(ctx, tenant.ID); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *TenantService) GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	err := Pool.QueryRow(ctx, `
		SELECT id, name, slug, email, status, plan_id, custom_domain, logo_url, primary_color, 
			secondary_color, timezone, language, features, metadata, created_at, updated_at, trial_ends_at
		FROM tenants WHERE id = $1
	`).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Email, &tenant.Status, &tenant.PlanID,
		&tenant.CustomDomain, &tenant.LogoURL, &tenant.PrimaryColor, &tenant.SecondaryColor,
		&tenant.Timezone, &tenant.Language, &tenant.Features, &tenant.Metadata,
		&tenant.CreatedAt, &tenant.UpdatedAt, &tenant.TrialEndsAt,
	)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *TenantService) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := Pool.QueryRow(ctx, `
		SELECT id, name, slug, email, status, plan_id, custom_domain, logo_url, primary_color, 
			secondary_color, timezone, language, features, metadata, created_at, updated_at, trial_ends_at
		FROM tenants WHERE slug = $1
	`).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Email, &tenant.Status, &tenant.PlanID,
		&tenant.CustomDomain, &tenant.LogoURL, &tenant.PrimaryColor, &tenant.SecondaryColor,
		&tenant.Timezone, &tenant.Language, &tenant.Features, &tenant.Metadata,
		&tenant.CreatedAt, &tenant.UpdatedAt, &tenant.TrialEndsAt,
	)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *TenantService) UpdateTenant(ctx context.Context, tenantID uuid.UUID, updates map[string]interface{}) error {
	// Build dynamic update query
	query := "UPDATE tenants SET updated_at = NOW(), "
	args := []interface{}{}
	argNum := 1

	for key, value := range updates {
		query += fmt.Sprintf("%s = $%d, ", key, argNum)
		args = append(args, value)
		argNum++
	}

	query = query[:len(query)-2] // Remove last comma
	query += fmt.Sprintf(" WHERE id = $%d", argNum)
	args = append(args, tenantID)

	_, err := Pool.Exec(ctx, query, args...)
	return err
}

func (s *TenantService) UpdateTenantStatus(ctx context.Context, tenantID uuid.UUID, status models.TenantStatus) error {
	_, err := Pool.Exec(ctx, `
		UPDATE tenants SET status = $1, updated_at = NOW() WHERE id = $2
	`, status, tenantID)
	return err
}

func (s *TenantService) DeleteTenant(ctx context.Context, tenantID uuid.UUID) error {
	// Soft delete - just mark as terminated
	return s.UpdateTenantStatus(ctx, tenantID, models.TenantStatusTerminated)
}

func (s *TenantService) ListTenants(ctx context.Context, status *models.TenantStatus, limit, offset int) ([]models.Tenant, error) {
	query := `
		SELECT id, name, slug, email, status, plan_id, custom_domain, logo_url, primary_color, 
			secondary_color, timezone, language, features, metadata, created_at, updated_at, trial_ends_at
		FROM tenants WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, *status)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []models.Tenant
	for rows.Next() {
		var tenant models.Tenant
		err := rows.Scan(
			&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Email, &tenant.Status, &tenant.PlanID,
			&tenant.CustomDomain, &tenant.LogoURL, &tenant.PrimaryColor, &tenant.SecondaryColor,
			&tenant.Timezone, &tenant.Language, &tenant.Features, &tenant.Metadata,
			&tenant.CreatedAt, &tenant.UpdatedAt, &tenant.TrialEndsAt,
		)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, tenant)
	}
	return tenants, nil
}

// Quota management

func (s *TenantService) createDefaultQuotas(ctx context.Context, tenantID uuid.UUID) error {
	now := time.Now()
	periodEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())

	quotas := []struct {
		resource string
		limit    int64
	}{
		{"api_calls", 1000},
		{"storage_gb", 1},
		{"users", 1},
		{"wallets", 5},
		{"bots", 1},
	}

	for _, quota := range quotas {
		_, err := Pool.Exec(ctx, `
			INSERT INTO quotas (id, tenant_id, resource, limit, used, period_start, period_end, reset_at)
			VALUES ($1, $2, $3, $4, 0, $5, $6, $6)
		`, uuid.New(), tenantID, quota.resource, quota.limit, now, periodEnd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *TenantService) createDefaultConfig(ctx context.Context, tenantID uuid.UUID) error {
	_, err := Pool.Exec(ctx, `
		INSERT INTO tenant_configs (id, tenant_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, uuid.New(), tenantID)
	return err
}

func (s *TenantService) GetQuota(ctx context.Context, tenantID, resource string) (*models.Quota, error) {
	var quota models.Quota
	err := Pool.QueryRow(ctx, `
		SELECT id, tenant_id, resource, limit, used, period_start, period_end, reset_at
		FROM quotas WHERE tenant_id = $1 AND resource = $2
	`, tenantID, resource).Scan(
		&quota.ID, &quota.TenantID, &quota.Resource, &quota.Limit, &quota.Used,
		&quota.PeriodStart, &quota.PeriodEnd, &quota.ResetAt,
	)
	if err != nil {
		return nil, err
	}
	return &quota, nil
}

func (s *TenantService) IncrementQuota(ctx context.Context, tenantID, resource string, amount int64) error {
	_, err := Pool.Exec(ctx, `
		UPDATE quotas SET used = used + $1, updated_at = NOW()
		WHERE tenant_id = $2 AND resource = $3
	`, amount, tenantID, resource)
	return err
}

func (s *TenantService) CheckQuota(ctx context.Context, tenantID, resource string) (bool, error) {
	quota, err := s.GetQuota(ctx, tenantID, resource)
	if err != nil {
		return false, err
	}

	if quota.Limit < 0 {
		return true, nil // Unlimited
	}

	return quota.Used < quota.Limit, nil
}

// Tenant user management

func (s *TenantService) AddUserToTenant(ctx context.Context, tenantID, userID uuid.UUID, role string) (*models.TenantUser, error) {
	tenantUser := &models.TenantUser{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		Role:       role,
		Permissions: []string{},
		Status:     "active",
		JoinedAt:   func() *time.Time { t := time.Now(); return &t }(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err := Pool.Exec(ctx, `
		INSERT INTO tenant_users (id, tenant_id, user_id, role, permissions, status, joined_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, tenantUser.ID, tenantUser.TenantID, tenantUser.UserID, tenantUser.Role,
		tenantUser.Permissions, tenantUser.Status, tenantUser.JoinedAt,
		tenantUser.CreatedAt, tenantUser.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return tenantUser, nil
}

func (s *TenantService) GetTenantUsers(ctx context.Context, tenantID uuid.UUID) ([]models.TenantUser, error) {
	rows, err := Pool.Query(ctx, `
		SELECT id, tenant_id, user_id, role, permissions, status, invited_at, joined_at, created_at, updated_at
		FROM tenant_users WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.TenantUser
	for rows.Next() {
		var u models.TenantUser
		err := rows.Scan(&u.ID, &u.TenantID, &u.UserID, &u.Role, &u.Permissions,
			&u.Status, &u.InvitedAt, &u.JoinedAt, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *TenantService) UpdateUserRole(ctx context.Context, tenantID, userID uuid.UUID, role string) error {
	_, err := Pool.Exec(ctx, `
		UPDATE tenant_users SET role = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND user_id = $3
	`, role, tenantID, userID)
	return err
}

func (s *TenantService) RemoveUserFromTenant(ctx context.Context, tenantID, userID uuid.UUID) error {
	_, err := Pool.Exec(ctx, `
		DELETE FROM tenant_users WHERE tenant_id = $1 AND user_id = $2
	`, tenantID, userID)
	return err
}

// Invitation management

func (s *TenantService) CreateInvitation(ctx context.Context, tenantID, invitedBy uuid.UUID, email, role string) (*models.TenantInvitation, error) {
	token := uuid.New().String()
	invitation := &models.TenantInvitation{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Email:     email,
		Role:      role,
		Token:     token,
		Status:    "pending",
		InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}

	_, err := Pool.Exec(ctx, `
		INSERT INTO tenant_invitations (id, tenant_id, email, role, token, status, invited_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, invitation.ID, invitation.TenantID, invitation.Email, invitation.Role,
		invitation.Token, invitation.Status, invitation.InvitedBy, invitation.ExpiresAt, invitation.CreatedAt)

	if err != nil {
		return nil, err
	}

	return invitation, nil
}

func (s *TenantService) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	// Get invitation
	var invitation models.TenantInvitation
	err := Pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, role, token, status, expires_at
		FROM tenant_invitations WHERE token = $1 AND status = 'pending'
	`, token).Scan(&invitation.ID, &invitation.TenantID, &invitation.Email,
		&invitation.Role, &invitation.Token, &invitation.Status, &invitation.ExpiresAt)

	if err != nil {
		return err
	}

	if time.Now().After(invitation.ExpiresAt) {
		return fmt.Errorf("invitation expired")
	}

	// Add user to tenant
	_, err = Pool.Exec(ctx, `
		INSERT INTO tenant_users (id, tenant_id, user_id, role, permissions, status, joined_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, '{}', 'active', NOW(), NOW(), NOW())
	`, uuid.New(), invitation.TenantID, userID, invitation.Role)

	if err != nil {
		return err
	}

	// Update invitation status
	_, err = Pool.Exec(ctx, `
		UPDATE tenant_invitations SET status = 'accepted' WHERE id = $1
	`, invitation.ID)

	return err
}

// Audit logging

func (s *TenantService) CreateAuditLog(ctx context.Context, tenantID, userID uuid.UUID, action, resource string, details map[string]interface{}) error {
	_, err := Pool.Exec(ctx, `
		INSERT INTO tenant_audit_logs (id, tenant_id, user_id, action, resource, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, uuid.New(), tenantID, userID, action, resource, details)

	return err
}

func (s *TenantService) GetAuditLogs(ctx context.Context, tenantID uuid.UUID, limit int) ([]models.TenantAuditLog, error) {
	rows, err := Pool.Query(ctx, `
		SELECT id, tenant_id, user_id, action, resource, resource_id, details, ip_address, user_agent, created_at
		FROM tenant_audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.TenantAuditLog
	for rows.Next() {
		var log models.TenantAuditLog
		err := rows.Scan(&log.ID, &log.TenantID, &log.UserID, &log.Action,
			&log.Resource, &log.ResourceID, &log.Details, &log.IPAddress,
			&log.UserAgent, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// Tenant config management

func (s *TenantService) GetTenantConfig(ctx context.Context, tenantID uuid.UUID) (*models.TenantConfig, error) {
	var config models.TenantConfig
	err := Pool.QueryRow(ctx, `
		SELECT id, tenant_id, wallet_settings, bot_settings, token_settings, security_settings, notification_settings, created_at, updated_at
		FROM tenant_configs WHERE tenant_id = $1
	`, tenantID).Scan(
		&config.ID, &config.TenantID, &config.WalletSettings, &config.BotSettings,
		&config.TokenSettings, &config.SecuritySettings, &config.NotificationSettings,
		&config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *TenantService) UpdateTenantConfig(ctx context.Context, tenantID uuid.UUID, config *models.TenantConfig) error {
	_, err := Pool.Exec(ctx, `
		UPDATE tenant_configs SET 
			wallet_settings = $1,
			bot_settings = $2,
			token_settings = $3,
			security_settings = $4,
			notification_settings = $5,
			updated_at = NOW()
		WHERE tenant_id = $6
	`, config.WalletSettings, config.BotSettings, config.TokenSettings,
		config.SecuritySettings, config.NotificationSettings, tenantID)

	return err
}
