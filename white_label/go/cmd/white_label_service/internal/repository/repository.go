package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"white_label_service/internal/models"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// CLIENT REPOSITORY
// ============================================================================

func (r *Repository) CreateClient(ctx context.Context, req *models.CreateClientRequest) (*models.WhiteLabelClient, error) {
	id := uuid.New()
	now := time.Now()
	
	query := `
		INSERT INTO wl_clients (id, name, domain, subdomain, custom_branding, logo_url, 
			primary_color, secondary_color, plan, max_users, fee_percent, features, 
			blockchain_access, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, name, domain, subdomain, custom_branding, logo_url, primary_color, 
			secondary_color, status, plan, max_users, current_users, fee_percent, features, 
			blockchain_access, api_keys, metadata, created_at, updated_at, approved_at, expires_at
	`
	
	featuresJSON, _ := json.Marshal(req.Features)
	blockchainAccessJSON, _ := json.Marshal(req.BlockchainAccess)
	
	primaryColor := "#1976d2"
	if req.PrimaryColor != "" {
		primaryColor = req.PrimaryColor
	}
	
	secondaryColor := "#1a1a2e"
	if req.SecondaryColor != "" {
		secondaryColor = req.SecondaryColor
	}
	
	plan := models.PlanStarter
	if req.Plan != "" {
		plan = req.Plan
	}
	
	maxUsers := 1000
	if req.MaxUsers > 0 {
		maxUsers = req.MaxUsers
	}
	
	feePercent := 20.0
	if req.FeePercent > 0 {
		feePercent = req.FeePercent
	}
	
	var client models.WhiteLabelClient
	err := r.db.QueryRowContext(ctx, query, 
		id, req.Name, req.Domain, req.Subdomain, req.CustomBranding, req.LogoURL,
		primaryColor, secondaryColor, plan, maxUsers, feePercent, featuresJSON, blockchainAccessJSON,
		models.StatusPending, now, now,
	).Scan(
		&client.ID, &client.Name, &client.Domain, &client.Subdomain, &client.CustomBranding,
		&client.LogoURL, &client.PrimaryColor, &client.SecondaryColor, &client.Status, &client.Plan,
		&client.MaxUsers, &client.CurrentUsers, &client.FeePercent, &client.Features,
		&client.BlockchainAccess, &client.APIKeys, &client.Metadata, &client.CreatedAt,
		&client.UpdatedAt, &client.ApprovedAt, &client.ExpiresAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	
	return &client, nil
}

func (r *Repository) GetClient(ctx context.Context, id uuid.UUID) (*models.WhiteLabelClient, error) {
	query := `
		SELECT id, name, domain, subdomain, custom_branding, logo_url, primary_color, 
			secondary_color, status, plan, max_users, current_users, fee_percent, features, 
			blockchain_access, api_keys, metadata, created_at, updated_at, approved_at, expires_at
		FROM wl_clients WHERE id = $1
	`
	
	var client models.WhiteLabelClient
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&client.ID, &client.Name, &client.Domain, &client.Subdomain, &client.CustomBranding,
		&client.LogoURL, &client.PrimaryColor, &client.SecondaryColor, &client.Status, &client.Plan,
		&client.MaxUsers, &client.CurrentUsers, &client.FeePercent, &client.Features,
		&client.BlockchainAccess, &client.APIKeys, &client.Metadata, &client.CreatedAt,
		&client.UpdatedAt, &client.ApprovedAt, &client.ExpiresAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return &client, nil
}

func (r *Repository) ListClients(ctx context.Context, params models.PaginationParams, filter models.SearchFilter) (*models.PaginatedResponse, error) {
	baseQuery := `FROM wl_clients WHERE 1=1`
	args := []interface{}{}
	argCount := 0
	
	if filter.Query != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND (name ILIKE $%d OR domain ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+filter.Query+"%")
	}
	
	if filter.Status != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
	}
	
	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, err
	}
	
	// Get page data
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	order := "DESC"
	if filter.Order != "" {
		order = filter.Order
	}
	
	offset := (page - 1) * pageSize
	argCount++
	args = append(args, pageSize, offset)
	
	selectQuery := fmt.Sprintf(`
		SELECT id, name, domain, subdomain, custom_branding, logo_url, primary_color, 
			secondary_color, status, plan, max_users, current_users, fee_percent, features, 
			blockchain_access, api_keys, metadata, created_at, updated_at, approved_at, expires_at
		%s ORDER BY %s %s LIMIT $%d OFFSET $%d
	`, baseQuery, sortBy, order, argCount, argCount+1)
	
	rows, err := r.db.QueryxContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var clients []models.WhiteLabelClient
	for rows.Next() {
		var client models.WhiteLabelClient
		err := rows.Scan(
			&client.ID, &client.Name, &client.Domain, &client.Subdomain, &client.CustomBranding,
			&client.LogoURL, &client.PrimaryColor, &client.SecondaryColor, &client.Status, &client.Plan,
			&client.MaxUsers, &client.CurrentUsers, &client.FeePercent, &client.Features,
			&client.BlockchainAccess, &client.APIKeys, &client.Metadata, &client.CreatedAt,
			&client.UpdatedAt, &client.ApprovedAt, &client.ExpiresAt,
		)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	
	if clients == nil {
		clients = []models.WhiteLabelClient{}
	}
	
	totalPages := (total + pageSize - 1) / pageSize
	
	return &models.PaginatedResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       clients,
	}, nil
}

func (r *Repository) UpdateClient(ctx context.Context, id uuid.UUID, req *models.UpdateClientRequest) (*models.WhiteLabelClient, error) {
	updates := []string{}
	args := []interface{}{}
	argCount := 0
	
	if req.Name != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("name = $%d", argCount))
		args = append(args, *req.Name)
	}
	if req.Subdomain != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("subdomain = $%d", argCount))
		args = append(args, *req.Subdomain)
	}
	if req.LogoURL != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("logo_url = $%d", argCount))
		args = append(args, *req.LogoURL)
	}
	if req.PrimaryColor != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("primary_color = $%d", argCount))
		args = append(args, *req.PrimaryColor)
	}
	if req.SecondaryColor != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("secondary_color = $%d", argCount))
		args = append(args, *req.SecondaryColor)
	}
	if req.Plan != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("plan = $%d", argCount))
		args = append(args, *req.Plan)
	}
	if req.MaxUsers != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("max_users = $%d", argCount))
		args = append(args, *req.MaxUsers)
	}
	if req.FeePercent != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("fee_percent = $%d", argCount))
		args = append(args, *req.FeePercent)
	}
	if req.Status != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *req.Status)
	}
	if req.Features != nil {
		argCount++
		featuresJSON, _ := json.Marshal(req.Features)
		updates = append(updates, fmt.Sprintf("features = $%d", argCount))
		args = append(args, featuresJSON)
	}
	if req.BlockchainAccess != nil {
		argCount++
		blockchainJSON, _ := json.Marshal(req.BlockchainAccess)
		updates = append(updates, fmt.Sprintf("blockchain_access = $%d", argCount))
		args = append(args, blockchainJSON)
	}
	
	if len(updates) == 0 {
		return r.GetClient(ctx, id)
	}
	
	argCount++
	args = append(args, id)
	
	query := fmt.Sprintf("UPDATE wl_clients SET %s WHERE id = $%d RETURNING id, name, domain, subdomain, custom_branding, logo_url, primary_color, secondary_color, status, plan, max_users, current_users, fee_percent, features, blockchain_access, api_keys, metadata, created_at, updated_at, approved_at, expires_at",
		strings.Join(updates, ", "), argCount)
	
	var client models.WhiteLabelClient
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&client.ID, &client.Name, &client.Domain, &client.Subdomain, &client.CustomBranding,
		&client.LogoURL, &client.PrimaryColor, &client.SecondaryColor, &client.Status, &client.Plan,
		&client.MaxUsers, &client.CurrentUsers, &client.FeePercent, &client.Features,
		&client.BlockchainAccess, &client.APIKeys, &client.Metadata, &client.CreatedAt,
		&client.UpdatedAt, &client.ApprovedAt, &client.ExpiresAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &client, nil
}

func (r *Repository) DeleteClient(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM wl_clients WHERE id = $1", id)
	return err
}

func (r *Repository) ApproveClient(ctx context.Context, id uuid.UUID) (*models.WhiteLabelClient, error) {
	query := `
		UPDATE wl_clients SET status = $1, approved_at = $2 WHERE id = $3
		RETURNING id, name, domain, subdomain, custom_branding, logo_url, primary_color, 
			secondary_color, status, plan, max_users, current_users, fee_percent, features, 
			blockchain_access, api_keys, metadata, created_at, updated_at, approved_at, expires_at
	`
	
	var client models.WhiteLabelClient
	err := r.db.QueryRowContext(ctx, query, models.StatusActive, time.Now(), id).Scan(
		&client.ID, &client.Name, &client.Domain, &client.Subdomain, &client.CustomBranding,
		&client.LogoURL, &client.PrimaryColor, &client.SecondaryColor, &client.Status, &client.Plan,
		&client.MaxUsers, &client.CurrentUsers, &client.FeePercent, &client.Features,
		&client.BlockchainAccess, &client.APIKeys, &client.Metadata, &client.CreatedAt,
		&client.UpdatedAt, &client.ApprovedAt, &client.ExpiresAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &client, nil
}

// ============================================================================
// ADMIN REPOSITORY
// ============================================================================

func (r *Repository) CreateAdmin(ctx context.Context, req *models.CreateAdminRequest) (*models.WhiteLabelAdmin, error) {
	id := uuid.New()
	now := time.Now()
	
	// Hash password using bcrypt (simplified - in production use proper bcrypt)
	passwordHash := hashPassword(req.Password)
	
	query := `
		INSERT INTO wl_admins (id, client_id, email, name, password_hash, role, permissions, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, client_id, email, name, password_hash, role, permissions, status, 
			two_factor_enabled, last_login, login_attempts, locked_until, created_at, updated_at
	`
	
	permissionsJSON, _ := json.Marshal(req.Permissions)
	role := models.RoleSupport
	if req.Role != "" {
		role = req.Role
	}
	
	var admin models.WhiteLabelAdmin
	err := r.db.QueryRowContext(ctx, query,
		id, req.ClientID, req.Email, req.Name, passwordHash, role, permissionsJSON,
		"active", now, now,
	).Scan(
		&admin.ID, &admin.ClientID, &admin.Email, &admin.Name, &admin.PasswordHash,
		&admin.Role, &admin.Permissions, &admin.Status, &admin.TwoFactorEnabled,
		&admin.LastLogin, &admin.LoginAttempts, &admin.LockedUntil, &admin.CreatedAt, &admin.UpdatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create admin: %w", err)
	}
	
	return &admin, nil
}

func (r *Repository) GetAdminByEmail(ctx context.Context, email string) (*models.WhiteLabelAdmin, error) {
	query := `
		SELECT id, client_id, email, name, password_hash, role, permissions, status, 
			two_factor_enabled, two_factor_secret, last_login, login_attempts, locked_until, created_at, updated_at
		FROM wl_admins WHERE email = $1
	`
	
	var admin models.WhiteLabelAdmin
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&admin.ID, &admin.ClientID, &admin.Email, &admin.Name, &admin.PasswordHash,
		&admin.Role, &admin.Permissions, &admin.Status, &admin.TwoFactorEnabled,
		&admin.TwoFactorSecret, &admin.LastLogin, &admin.LoginAttempts, &admin.LockedUntil,
		&admin.CreatedAt, &admin.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return &admin, nil
}

func (r *Repository) GetAdmin(ctx context.Context, id uuid.UUID) (*models.WhiteLabelAdmin, error) {
	query := `
		SELECT id, client_id, email, name, password_hash, role, permissions, status, 
			two_factor_enabled, two_factor_secret, last_login, login_attempts, locked_until, created_at, updated_at
		FROM wl_admins WHERE id = $1
	`
	
	var admin models.WhiteLabelAdmin
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&admin.ID, &admin.ClientID, &admin.Email, &admin.Name, &admin.PasswordHash,
		&admin.Role, &admin.Permissions, &admin.Status, &admin.TwoFactorEnabled,
		&admin.TwoFactorSecret, &admin.LastLogin, &admin.LoginAttempts, &admin.LockedUntil,
		&admin.CreatedAt, &admin.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return &admin, nil
}

func (r *Repository) ListAdmins(ctx context.Context, params models.PaginationParams, filter models.SearchFilter, clientID *uuid.UUID) (*models.PaginatedResponse, error) {
	baseQuery := `FROM wl_admins WHERE 1=1`
	args := []interface{}{}
	argCount := 0
	
	if clientID != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND client_id = $%d", argCount)
		args = append(args, *clientID)
	}
	
	if filter.Query != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND (name ILIKE $%d OR email ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+filter.Query+"%")
	}
	
	if filter.Status != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
	}
	
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, err
	}
	
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	order := "DESC"
	if filter.Order != "" {
		order = filter.Order
	}
	
	offset := (page - 1) * pageSize
	argCount++
	args = append(args, pageSize, offset)
	
	selectQuery := fmt.Sprintf(`
		SELECT id, client_id, email, name, password_hash, role, permissions, status, 
			two_factor_enabled, two_factor_secret, last_login, login_attempts, locked_until, created_at, updated_at
		%s ORDER BY %s %s LIMIT $%d OFFSET $%d
	`, baseQuery, sortBy, order, argCount, argCount+1)
	
	rows, err := r.db.QueryxContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var admins []models.WhiteLabelAdmin
	for rows.Next() {
		var admin models.WhiteLabelAdmin
		err := rows.Scan(
			&admin.ID, &admin.ClientID, &admin.Email, &admin.Name, &admin.PasswordHash,
			&admin.Role, &admin.Permissions, &admin.Status, &admin.TwoFactorEnabled,
			&admin.TwoFactorSecret, &admin.LastLogin, &admin.LoginAttempts, &admin.LockedUntil,
			&admin.CreatedAt, &admin.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		admins = append(admins, admin)
	}
	
	if admins == nil {
		admins = []models.WhiteLabelAdmin{}
	}
	
	totalPages := (total + pageSize - 1) / pageSize
	
	return &models.PaginatedResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       admins,
	}, nil
}

func (r *Repository) UpdateAdmin(ctx context.Context, id uuid.UUID, req *models.UpdateAdminRequest) (*models.WhiteLabelAdmin, error) {
	updates := []string{}
	args := []interface{}{}
	argCount := 0
	
	if req.Name != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("name = $%d", argCount))
		args = append(args, *req.Name)
	}
	if req.Role != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("role = $%d", argCount))
		args = append(args, *req.Role)
	}
	if req.Permissions != nil {
		argCount++
		permsJSON, _ := json.Marshal(req.Permissions)
		updates = append(updates, fmt.Sprintf("permissions = $%d", argCount))
		args = append(args, permsJSON)
	}
	if req.Status != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *req.Status)
	}
	
	if len(updates) == 0 {
		return r.GetAdmin(ctx, id)
	}
	
	argCount++
	args = append(args, id)
	
	query := fmt.Sprintf(`
		UPDATE wl_admins SET %s WHERE id = $%d
		RETURNING id, client_id, email, name, password_hash, role, permissions, status, 
			two_factor_enabled, two_factor_secret, last_login, login_attempts, locked_until, created_at, updated_at
	`, strings.Join(updates, ", "), argCount)
	
	var admin models.WhiteLabelAdmin
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&admin.ID, &admin.ClientID, &admin.Email, &admin.Name, &admin.PasswordHash,
		&admin.Role, &admin.Permissions, &admin.Status, &admin.TwoFactorEnabled,
		&admin.TwoFactorSecret, &admin.LastLogin, &admin.LoginAttempts, &admin.LockedUntil,
		&admin.CreatedAt, &admin.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &admin, nil
}

func (r *Repository) DeleteAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM wl_admins WHERE id = $1", id)
	return err
}

func (r *Repository) UpdateAdminLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE wl_admins SET last_login = $1, login_attempts = 0, locked_until = NULL WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *Repository) IncrementLoginAttempts(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE wl_admins SET login_attempts = login_attempts + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) LockAdmin(ctx context.Context, id uuid.UUID, until time.Time) error {
	query := `UPDATE wl_admins SET locked_until = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, until, id)
	return err
}

// ============================================================================
// PRODUCT REPOSITORY
// ============================================================================

func (r *Repository) CreateProduct(ctx context.Context, req *models.CreateProductRequest) (*models.Product, error) {
	id := uuid.New()
	now := time.Now()
	
	query := `
		INSERT INTO wl_products (id, client_id, name, type, description, status, fee, 
			min_deposit, max_deposit, min_withdrawal, max_withdrawal, features, settings, 
			sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, client_id, name, type, description, status, fee, min_deposit, 
			max_deposit, min_withdrawal, max_withdrawal, features, settings, sort_order, created_at, updated_at
	`
	
	featuresJSON, _ := json.Marshal(req.Features)
	settingsJSON, _ := json.Marshal(req.Settings)
	
	status := models.ProductEnabled
	if req.Status != "" {
		status = req.Status
	}
	
	var product models.Product
	err := r.db.QueryRowContext(ctx, query,
		id, req.ClientID, req.Name, req.Type, req.Description, status, req.Fee,
		req.MinDeposit, req.MaxDeposit, req.MinWithdrawal, req.MaxWithdrawal,
		featuresJSON, settingsJSON, req.SortOrder, now, now,
	).Scan(
		&product.ID, &product.ClientID, &product.Name, &product.Type, &product.Description,
		&product.Status, &product.Fee, &product.MinDeposit, &product.MaxDeposit,
		&product.MinWithdrawal, &product.MaxWithdrawal, &product.Features, &product.Settings,
		&product.SortOrder, &product.CreatedAt, &product.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &product, nil
}

func (r *Repository) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	query := `
		SELECT id, client_id, name, type, description, status, fee, min_deposit, 
			max_deposit, min_withdrawal, max_withdrawal, features, settings, sort_order, created_at, updated_at
		FROM wl_products WHERE id = $1
	`
	
	var product models.Product
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID, &product.ClientID, &product.Name, &product.Type, &product.Description,
		&product.Status, &product.Fee, &product.MinDeposit, &product.MaxDeposit,
		&product.MinWithdrawal, &product.MaxWithdrawal, &product.Features, &product.Settings,
		&product.SortOrder, &product.CreatedAt, &product.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return &product, nil
}

func (r *Repository) ListProducts(ctx context.Context, params models.PaginationParams, filter models.SearchFilter, clientID *uuid.UUID) (*models.PaginatedResponse, error) {
	baseQuery := `FROM wl_products WHERE 1=1`
	args := []interface{}{}
	argCount := 0
	
	if clientID != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND client_id = $%d", argCount)
		args = append(args, *clientID)
	}
	
	if filter.Query != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND name ILIKE $%d", argCount)
		args = append(args, "%"+filter.Query+"%")
	}
	
	if filter.Status != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
	}
	
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, err
	}
	
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	
	sortBy := "sort_order, created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	order := "ASC"
	if filter.Order != "" {
		order = filter.Order
	}
	
	offset := (page - 1) * pageSize
	argCount++
	args = append(args, pageSize, offset)
	
	selectQuery := fmt.Sprintf(`
		SELECT id, client_id, name, type, description, status, fee, min_deposit, 
			max_deposit, min_withdrawal, max_withdrawal, features, settings, sort_order, created_at, updated_at
		%s ORDER BY %s %s LIMIT $%d OFFSET $%d
	`, baseQuery, sortBy, order, argCount, argCount+1)
	
	rows, err := r.db.QueryxContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var products []models.Product
	for rows.Next() {
		var product models.Product
		err := rows.Scan(
			&product.ID, &product.ClientID, &product.Name, &product.Type, &product.Description,
			&product.Status, &product.Fee, &product.MinDeposit, &product.MaxDeposit,
			&product.MinWithdrawal, &product.MaxWithdrawal, &product.Features, &product.Settings,
			&product.SortOrder, &product.CreatedAt, &product.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	
	if products == nil {
		products = []models.Product{}
	}
	
	totalPages := (total + pageSize - 1) / pageSize
	
	return &models.PaginatedResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       products,
	}, nil
}

func (r *Repository) UpdateProduct(ctx context.Context, id uuid.UUID, req *models.UpdateProductRequest) (*models.Product, error) {
	updates := []string{}
	args := []interface{}{}
	argCount := 0
	
	if req.Name != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("name = $%d", argCount))
		args = append(args, *req.Name)
	}
	if req.Description != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("description = $%d", argCount))
		args = append(args, *req.Description)
	}
	if req.Status != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *req.Status)
	}
	if req.Fee != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("fee = $%d", argCount))
		args = append(args, *req.Fee)
	}
	if req.MinDeposit != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("min_deposit = $%d", argCount))
		args = append(args, *req.MinDeposit)
	}
	if req.MaxDeposit != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("max_deposit = $%d", argCount))
		args = append(args, *req.MaxDeposit)
	}
	if req.MinWithdrawal != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("min_withdrawal = $%d", argCount))
		args = append(args, *req.MinWithdrawal)
	}
	if req.MaxWithdrawal != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("max_withdrawal = $%d", argCount))
		args = append(args, *req.MaxWithdrawal)
	}
	if req.Features != nil {
		argCount++
		featuresJSON, _ := json.Marshal(req.Features)
		updates = append(updates, fmt.Sprintf("features = $%d", argCount))
		args = append(args, featuresJSON)
	}
	if req.Settings != nil {
		argCount++
		settingsJSON, _ := json.Marshal(req.Settings)
		updates = append(updates, fmt.Sprintf("settings = $%d", argCount))
		args = append(args, settingsJSON)
	}
	if req.SortOrder != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argCount))
		args = append(args, *req.SortOrder)
	}
	
	if len(updates) == 0 {
		return r.GetProduct(ctx, id)
	}
	
	argCount++
	args = append(args, id)
	
	query := fmt.Sprintf(`
		UPDATE wl_products SET %s WHERE id = $%d
		RETURNING id, client_id, name, type, description, status, fee, min_deposit, 
			max_deposit, min_withdrawal, max_withdrawal, features, settings, sort_order, created_at, updated_at
	`, strings.Join(updates, ", "), argCount)
	
	var product models.Product
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&product.ID, &product.ClientID, &product.Name, &product.Type, &product.Description,
		&product.Status, &product.Fee, &product.MinDeposit, &product.MaxDeposit,
		&product.MinWithdrawal, &product.MaxWithdrawal, &product.Features, &product.Settings,
		&product.SortOrder, &product.CreatedAt, &product.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &product, nil
}

func (r *Repository) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM wl_products WHERE id = $1", id)
	return err
}

// ============================================================================
// TRADING PAIRS REPOSITORY
// ============================================================================

func (r *Repository) CreateTradingPair(ctx context.Context, req *models.CreateTradingPairRequest) (*models.TradingPair, error) {
	id := uuid.New()
	now := time.Now()
	
	query := `
		INSERT INTO wl_trading_pairs (id, client_id, base_token, quote_token, chain_id, 
			pair_address, status, fee, min_trade, max_trade, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, client_id, base_token, quote_token, chain_id, pair_address, status, 
			fee, min_trade, max_trade, liquidity, price_precision, quantity_precision, created_at, updated_at
	`
	
	fee := 0.1
	if req.Fee > 0 {
		fee = req.Fee
	}
	
	var pair models.TradingPair
	err := r.db.QueryRowContext(ctx, query,
		id, req.ClientID, req.BaseToken, req.QuoteToken, req.ChainID, req.PairAddress,
		models.PairActive, fee, req.MinTrade, req.MaxTrade, now, now,
	).Scan(
		&pair.ID, &pair.ClientID, &pair.BaseToken, &pair.QuoteToken, &pair.ChainID,
		&pair.PairAddress, &pair.Status, &pair.Fee, &pair.MinTrade, &pair.MaxTrade,
		&pair.Liquidity, &pair.PricePrecision, &pair.QuantityPrecision, &pair.CreatedAt, &pair.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &pair, nil
}

func (r *Repository) GetTradingPair(ctx context.Context, id uuid.UUID) (*models.TradingPair, error) {
	query := `
		SELECT id, client_id, base_token, quote_token, chain_id, pair_address, status, 
			fee, min_trade, max_trade, liquidity, price_precision, quantity_precision, created_at, updated_at
		FROM wl_trading_pairs WHERE id = $1
	`
	
	var pair models.TradingPair
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&pair.ID, &pair.ClientID, &pair.BaseToken, &pair.QuoteToken, &pair.ChainID,
		&pair.PairAddress, &pair.Status, &pair.Fee, &pair.MinTrade, &pair.MaxTrade,
		&pair.Liquidity, &pair.PricePrecision, &pair.QuantityPrecision, &pair.CreatedAt, &pair.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return &pair, nil
}

func (r *Repository) ListTradingPairs(ctx context.Context, params models.PaginationParams, filter models.SearchFilter, clientID *uuid.UUID) (*models.PaginatedResponse, error) {
	baseQuery := `FROM wl_trading_pairs WHERE 1=1`
	args := []interface{}{}
	argCount := 0
	
	if clientID != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND client_id = $%d", argCount)
		args = append(args, *clientID)
	}
	
	if filter.Query != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND (base_token ILIKE $%d OR quote_token ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+filter.Query+"%")
	}
	
	if filter.Status != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
	}
	
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, err
	}
	
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	
	offset := (page - 1) * pageSize
	argCount++
	args = append(args, pageSize, offset)
	
	selectQuery := fmt.Sprintf(`
		SELECT id, client_id, base_token, quote_token, chain_id, pair_address, status, 
			fee, min_trade, max_trade, liquidity, price_precision, quantity_precision, created_at, updated_at
		%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d
	`, baseQuery, argCount, argCount+1)
	
	rows, err := r.db.QueryxContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var pairs []models.TradingPair
	for rows.Next() {
		var pair models.TradingPair
		err := rows.Scan(
			&pair.ID, &pair.ClientID, &pair.BaseToken, &pair.QuoteToken, &pair.ChainID,
			&pair.PairAddress, &pair.Status, &pair.Fee, &pair.MinTrade, &pair.MaxTrade,
			&pair.Liquidity, &pair.PricePrecision, &pair.QuantityPrecision, &pair.CreatedAt, &pair.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	
	if pairs == nil {
		pairs = []models.TradingPair{}
	}
	
	totalPages := (total + pageSize - 1) / pageSize
	
	return &models.PaginatedResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       pairs,
	}, nil
}

func (r *Repository) UpdateTradingPair(ctx context.Context, id uuid.UUID, req *models.UpdateTradingPairRequest) (*models.TradingPair, error) {
	updates := []string{}
	args := []interface{}{}
	argCount := 0
	
	if req.Status != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *req.Status)
	}
	if req.Fee != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("fee = $%d", argCount))
		args = append(args, *req.Fee)
	}
	if req.MinTrade != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("min_trade = $%d", argCount))
		args = append(args, *req.MinTrade)
	}
	if req.MaxTrade != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("max_trade = $%d", argCount))
		args = append(args, *req.MaxTrade)
	}
	if req.PairAddress != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("pair_address = $%d", argCount))
		args = append(args, *req.PairAddress)
	}
	
	if len(updates) == 0 {
		return r.GetTradingPair(ctx, id)
	}
	
	argCount++
	args = append(args, id)
	
	query := fmt.Sprintf(`
		UPDATE wl_trading_pairs SET %s WHERE id = $%d
		RETURNING id, client_id, base_token, quote_token, chain_id, pair_address, status, 
			fee, min_trade, max_trade, liquidity, price_precision, quantity_precision, created_at, updated_at
	`, strings.Join(updates, ", "), argCount)
	
	var pair models.TradingPair
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&pair.ID, &pair.ClientID, &pair.BaseToken, &pair.QuoteToken, &pair.ChainID,
		&pair.PairAddress, &pair.Status, &pair.Fee, &pair.MinTrade, &pair.MaxTrade,
		&pair.Liquidity, &pair.PricePrecision, &pair.QuantityPrecision, &pair.CreatedAt, &pair.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &pair, nil
}

func (r *Repository) DeleteTradingPair(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM wl_trading_pairs WHERE id = $1", id)
	return err
}

// ============================================================================
// BLOCKCHAIN REPOSITORY
// ============================================================================

func (r *Repository) ListBlockchains(ctx context.Context) ([]models.Blockchain, error) {
	query := `
		SELECT id, name, symbol, category, rpc_urls, explorer_urls, status, is_default, icon_url, created_at, updated_at
		FROM wl_blockchains ORDER BY is_default DESC, id ASC
	`
	
	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var blockchains []models.Blockchain
	for rows.Next() {
		var bc models.Blockchain
		err := rows.Scan(
			&bc.ID, &bc.Name, &bc.Symbol, &bc.Category, &bc.RPCUrls, &bc.ExplorerUrls,
			&bc.Status, &bc.IsDefault, &bc.IconURL, &bc.CreatedAt, &bc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		blockchains = append(blockchains, bc)
	}
	
	if blockchains == nil {
		blockchains = []models.Blockchain{}
	}
	
	return blockchains, nil
}

func (r *Repository) UpdateBlockchain(ctx context.Context, id int64, req *models.UpdateBlockchainRequest) (*models.Blockchain, error) {
	updates := []string{}
	args := []interface{}{}
	argCount := 0
	
	if req.Name != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("name = $%d", argCount))
		args = append(args, *req.Name)
	}
	if req.RPCUrls != nil {
		argCount++
		rpcJSON, _ := json.Marshal(req.RPCUrls)
		updates = append(updates, fmt.Sprintf("rpc_urls = $%d", argCount))
		args = append(args, rpcJSON)
	}
	if req.ExplorerUrls != nil {
		argCount++
		explorerJSON, _ := json.Marshal(req.ExplorerUrls)
		updates = append(updates, fmt.Sprintf("explorer_urls = $%d", argCount))
		args = append(args, explorerJSON)
	}
	if req.Status != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *req.Status)
	}
	if req.IsDefault != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("is_default = $%d", argCount))
		args = append(args, *req.IsDefault)
	}
	
	if len(updates) == 0 {
		return nil, nil
	}
	
	args = append(args, id)
	
	query := fmt.Sprintf(`
		UPDATE wl_blockchains SET %s WHERE id = $%d
		RETURNING id, name, symbol, category, rpc_urls, explorer_urls, status, is_default, icon_url, created_at, updated_at
	`, strings.Join(updates, ", "), argCount+1)
	
	var bc models.Blockchain
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&bc.ID, &bc.Name, &bc.Symbol, &bc.Category, &bc.RPCUrls, &bc.ExplorerUrls,
		&bc.Status, &bc.IsDefault, &bc.IconURL, &bc.CreatedAt, &bc.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &bc, nil
}

// ============================================================================
// AUDIT LOG REPOSITORY
// ============================================================================

func (r *Repository) CreateAuditLog(ctx context.Context, req *models.CreateAuditLogRequest) (*models.AuditLog, error) {
	id := uuid.New()
	
	query := `
		INSERT INTO wl_audit_logs (id, client_id, admin_id, action, resource_type, resource_id, 
			details, ip_address, user_agent, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, client_id, admin_id, action, resource_type, resource_id, 
			details, ip_address, user_agent, status, created_at
	`
	
	detailsJSON, _ := json.Marshal(req.Details)
	status := "success"
	if req.Status != "" {
		status = req.Status
	}
	
	var log models.AuditLog
	err := r.db.QueryRowContext(ctx, query,
		id, req.ClientID, req.AdminID, req.Action, req.ResourceType, req.ResourceID,
		detailsJSON, req.IPAddress, req.UserAgent, status, time.Now(),
	).Scan(
		&log.ID, &log.ClientID, &log.AdminID, &log.Action, &log.ResourceType, &log.ResourceID,
		&log.Details, &log.IPAddress, &log.UserAgent, &log.Status, &log.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &log, nil
}

func (r *Repository) ListAuditLogs(ctx context.Context, params models.PaginationParams, filter models.SearchFilter, clientID *uuid.UUID) (*models.PaginatedResponse, error) {
	baseQuery := `FROM wl_audit_logs WHERE 1=1`
	args := []interface{}{}
	argCount := 0
	
	if clientID != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND client_id = $%d", argCount)
		args = append(args, *clientID)
	}
	
	if filter.Query != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND (action ILIKE $%d OR resource_type ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+filter.Query+"%")
	}
	
	if filter.Status != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
	}
	
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, err
	}
	
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	
	offset := (page - 1) * pageSize
	argCount++
	args = append(args, pageSize, offset)
	
	selectQuery := fmt.Sprintf(`
		SELECT id, client_id, admin_id, action, resource_type, resource_id, 
			details, ip_address, user_agent, status, created_at
		%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d
	`, baseQuery, argCount, argCount+1)
	
	rows, err := r.db.QueryxContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var logs []models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		err := rows.Scan(
			&log.ID, &log.ClientID, &log.AdminID, &log.Action, &log.ResourceType, &log.ResourceID,
			&log.Details, &log.IPAddress, &log.UserAgent, &log.Status, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	
	if logs == nil {
		logs = []models.AuditLog{}
	}
	
	totalPages := (total + pageSize - 1) / pageSize
	
	return &models.PaginatedResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       logs,
	}, nil
}

// ============================================================================
// NOTIFICATION REPOSITORY
// ============================================================================

func (r *Repository) CreateNotification(ctx context.Context, req *models.CreateNotificationRequest) (*models.Notification, error) {
	id := uuid.New()
	
	query := `
		INSERT INTO wl_notifications (id, client_id, admin_id, type, title, message, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, client_id, admin_id, type, title, message, data, read, read_at, created_at
	`
	
	dataJSON, _ := json.Marshal(req.Data)
	
	var notif models.Notification
	err := r.db.QueryRowContext(ctx, query,
		id, req.ClientID, req.AdminID, req.Type, req.Title, req.Message, dataJSON, time.Now(),
	).Scan(
		&notif.ID, &notif.ClientID, &notif.AdminID, &notif.Type, &notif.Title,
		&notif.Message, &notif.Data, &notif.Read, &notif.ReadAt, &notif.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &notif, nil
}

func (r *Repository) ListNotifications(ctx context.Context, params models.PaginationParams, clientID *uuid.UUID, adminID *uuid.UUID) (*models.PaginatedResponse, error) {
	baseQuery := `FROM wl_notifications WHERE 1=1`
	args := []interface{}{}
	argCount := 0
	
	if clientID != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND client_id = $%d", argCount)
		args = append(args, *clientID)
	}
	
	if adminID != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND admin_id = $%d", argCount)
		args = append(args, *adminID)
	}
	
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, err
	}
	
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	
	offset := (page - 1) * pageSize
	argCount++
	args = append(args, pageSize, offset)
	
	selectQuery := fmt.Sprintf(`
		SELECT id, client_id, admin_id, type, title, message, data, read, read_at, created_at
		%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d
	`, baseQuery, argCount, argCount+1)
	
	rows, err := r.db.QueryxContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var notifications []models.Notification
	for rows.Next() {
		var notif models.Notification
		err := rows.Scan(
			&notif.ID, &notif.ClientID, &notif.AdminID, &notif.Type, &notif.Title,
			&notif.Message, &notif.Data, &notif.Read, &notif.ReadAt, &notif.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notif)
	}
	
	if notifications == nil {
		notifications = []models.Notification{}
	}
	
	totalPages := (total + pageSize - 1) / pageSize
	
	return &models.PaginatedResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       notifications,
	}, nil
}

func (r *Repository) MarkNotificationRead(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE wl_notifications SET read = true, read_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// ============================================================================
// DASHBOARD STATS
// ============================================================================

func (r *Repository) GetDashboardStats(ctx context.Context) (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}
	
	// Total clients
	r.db.GetContext(ctx, &stats.TotalClients, "SELECT COUNT(*) FROM wl_clients")
	
	// Active clients
	r.db.GetContext(ctx, &stats.ActiveClients, "SELECT COUNT(*) FROM wl_clients WHERE status = 'active'")
	
	// Pending clients
	r.db.GetContext(ctx, &stats.PendingClients, "SELECT COUNT(*) FROM wl_clients WHERE status = 'pending'")
	
	// Total admins
	r.db.GetContext(ctx, &stats.TotalAdmins, "SELECT COUNT(*) FROM wl_admins")
	
	// Total products
	r.db.GetContext(ctx, &stats.TotalProducts, "SELECT COUNT(*) FROM wl_products")
	
	// Total pairs
	r.db.GetContext(ctx, &stats.TotalPairs, "SELECT COUNT(*) FROM wl_trading_pairs")
	
	// Total pools
	r.db.GetContext(ctx, &stats.TotalPools, "SELECT COUNT(*) FROM wl_liquidity_pools")
	
	// Total tokens
	r.db.GetContext(ctx, &stats.TotalTokens, "SELECT COUNT(*) FROM wl_token_configs")
	
	// Total bots
	r.db.GetContext(ctx, &stats.TotalBots, "SELECT COUNT(*) FROM wl_market_maker_bots")
	
	// Total users
	r.db.GetContext(ctx, &stats.TotalUsers, "SELECT COALESCE(SUM(current_users), 0) FROM wl_clients")
	
	// Volume stats (placeholder - would come from analytics table)
	stats.Volume24h = 12500000.0
	stats.Volume7d = 87500000.0
	stats.Volume30d = 375000000.0
	
	// Revenue stats
	stats.Revenue24h = 125000.0
	stats.Revenue7d = 875000.0
	stats.Revenue30d = 3750000.0
	
	return stats, nil
}

// Helper functions
func hashPassword(password string) string {
	// In production, use bcrypt or argon2
	// This is a simplified version
	return fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
}
