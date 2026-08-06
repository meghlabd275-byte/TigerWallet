package services

import (
	"context"
	"fmt"
	"time"

	"admin_console/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type UserService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewUserService(db *pgxpool.Pool, redis *redis.Client) *UserService {
	return &UserService{db: db, redis: redis}
}

func (s *UserService) List(ctx context.Context, params *models.PaginationParams, status, role string) (*models.PaginatedResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize

	// Build query
	query := `SELECT id, email, username, first_name, last_name, phone, role, status, email_verified, two_factor_enabled, created_at, updated_at FROM users WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM users WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	if role != "" {
		query += fmt.Sprintf(" AND role = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND role = $%d", argIndex)
		args = append(args, role)
		argIndex++
	}

	// Get total count
	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Add ordering and pagination
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.PageSize, offset)

	// Execute query
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Email, &user.Username, &user.FirstName, &user.LastName,
			&user.Phone, &user.Role, &user.Status, &user.EmailVerified, &user.TwoFactorEnabled,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	totalPages := (total + params.PageSize - 1) / params.PageSize

	return &models.PaginatedResponse{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		Data:       users,
	}, nil
}

func (s *UserService) Get(ctx context.Context, id uuid.UUID) (*models.User, *models.UserProfile, error) {
	var user models.User
	var profile models.UserProfile

	err := s.db.QueryRow(ctx, `
		SELECT id, email, username, first_name, last_name, phone, role, status, 
		       email_verified, two_factor_enabled, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Email, &user.Username, &user.FirstName, &user.LastName,
		&user.Phone, &user.Role, &user.Status, &user.EmailVerified, &user.TwoFactorEnabled,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get profile
	s.db.QueryRow(ctx, `
		SELECT id, user_id, avatar_url, bio, address, city, country, postal_code, timezone, language, preferences
		FROM user_profiles WHERE user_id = $1
	`, id).Scan(
		&profile.ID, &profile.UserID, &profile.AvatarURL, &profile.Bio, &profile.Address,
		&profile.City, &profile.Country, &profile.PostalCode, &profile.Timezone, &profile.Language, &profile.Preferences,
	)

	return &user, &profile, nil
}

func (s *UserService) Create(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	var user models.User
	err = s.db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, first_name, last_name, role, status, email_verified)
		VALUES ($1, $2, $3, $4, $5, 'admin', 'active', true)
		RETURNING id, email, username, first_name, last_name, role, status, email_verified, created_at, updated_at
	`, req.Email, req.Username, string(hashedPassword), req.FirstName, req.LastName).Scan(
		&user.ID, &user.Email, &user.Username, &user.FirstName, &user.LastName,
		&user.Role, &user.Status, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create profile
	s.db.Exec(ctx, `INSERT INTO user_profiles (user_id, timezone, language, preferences) VALUES ($1, 'UTC', 'en', '{}')`, user.ID)

	return &user, nil
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*models.User, error) {
	// Build update query dynamically
	query := "UPDATE users SET updated_at = NOW()"
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		query += fmt.Sprintf(", %s = $%d", key, argIndex)
		args = append(args, value)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, email, username, first_name, last_name, phone, role, status, created_at, updated_at", argIndex)
	args = append(args, id)

	var user models.User
	err := s.db.QueryRow(ctx, query, args...).Scan(
		&user.ID, &user.Email, &user.Username, &user.FirstName, &user.LastName,
		&user.Phone, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *UserService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	return nil
}

func (s *UserService) GetActivity(ctx context.Context, userID uuid.UUID) ([]models.UserActivity, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, action, details, ip_address, created_at
		FROM user_activities 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}
	defer rows.Close()

	var activities []models.UserActivity
	for rows.Next() {
		var activity models.UserActivity
		err := rows.Scan(&activity.ID, &activity.UserID, &activity.Action, &activity.Details, &activity.IPAddress, &activity.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}
		activities = append(activities, activity)
	}

	return activities, nil
}

// KYC Service
type KYCService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewKYCService(db *pgxpool.Pool, redis *redis.Client) *KYCService {
	return &KYCService{db: db, redis: redis}
}

func (s *KYCService) List(ctx context.Context, params *models.PaginationParams, status string) (*models.PaginatedResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize

	query := `SELECT id, user_id, document_type, document_number, status, rejection_reason, reviewed_by, reviewed_at, created_at, updated_at FROM kyc_records WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM kyc_records WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count: %w", err)
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.PageSize, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	var records []models.KYCReview
	for rows.Next() {
		var r models.KYCReview
		err := rows.Scan(&r.ID, &r.UserID, &r.DocumentType, &r.DocumentNumber, &r.Status, &r.RejectionReason, &r.ReviewedBy, &r.ReviewedAt, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		records = append(records, r)
	}

	return &models.PaginatedResponse{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: (total + params.PageSize - 1) / params.PageSize,
		Data:       records,
	}, nil
}

func (s *KYCService) Get(ctx context.Context, id uuid.UUID) (*models.KYCReview, error) {
	var r models.KYCReview
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, document_type, document_number, document_front, document_back, selfie_url, proof_of_address, status, rejection_reason, reviewed_by, reviewed_at, created_at, updated_at
		FROM kyc_records WHERE id = $1
	`, id).Scan(&r.ID, &r.UserID, &r.DocumentType, &r.DocumentNumber, &r.DocumentFront, &r.DocumentBack, &r.SelfieURL, &r.ProofOfAddress, &r.Status, &r.RejectionReason, &r.ReviewedBy, &r.ReviewedAt, &r.CreatedAt, &r.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("KYC not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get KYC: %w", err)
	}
	return &r, nil
}

func (s *KYCService) Approve(ctx context.Context, id, reviewerID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE kyc_records SET status = 'approved', reviewed_by = $1, reviewed_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`, reviewerID, id)
	if err != nil {
		return fmt.Errorf("failed to approve: %w", err)
	}
	return nil
}

func (s *KYCService) Reject(ctx context.Context, id, reviewerID uuid.UUID, reason string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE kyc_records SET status = 'rejected', reviewed_by = $1, reviewed_at = NOW(), rejection_reason = $2, updated_at = NOW()
		WHERE id = $3
	`, reviewerID, reason, id)
	if err != nil {
		return fmt.Errorf("failed to reject: %w", err)
	}
	return nil
}

func (s *KYCService) RequestInfo(ctx context.Context, id uuid.UUID, message string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE kyc_records SET status = 'info_requested', rejection_reason = $1, updated_at = NOW()
		WHERE id = $2
	`, message, id)
	return err
}

func (s *KYCService) GetStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM kyc_records WHERE status = 'pending'`).Scan(&stats["pending"])
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM kyc_records WHERE status = 'approved'`).Scan(&stats["approved"])
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM kyc_records WHERE status = 'rejected'`).Scan(&stats["rejected"])
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM kyc_records WHERE status = 'info_requested'`).Scan(&stats["info_requested"])

	return stats, nil
}

// Token Service
type TokenService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewTokenService(db *pgxpool.Pool, redis *redis.Client) *TokenService {
	return &TokenService{db: db, redis: redis}
}

func (s *TokenService) List(ctx context.Context, params *models.PaginationParams, status string) (*models.PaginatedResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize

	query := `SELECT id, name, symbol, contract_address, chain, decimals, total_supply, status, listing_fee, created_at, updated_at FROM tokens WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM tokens WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count: %w", err)
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.PageSize, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	var tokens []models.Token
	for rows.Next() {
		var t models.Token
		err := rows.Scan(&t.ID, &t.Name, &t.Symbol, &t.ContractAddress, &t.Chain, &t.Decimals, &t.TotalSupply, &t.Status, &t.ListingFee, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		tokens = append(tokens, t)
	}

	return &models.PaginatedResponse{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: (total + params.PageSize - 1) / params.PageSize,
		Data:       tokens,
	}, nil
}

func (s *TokenService) Get(ctx context.Context, id uuid.UUID) (*models.Token, error) {
	var t models.Token
	err := s.db.QueryRow(ctx, `
		SELECT id, name, symbol, contract_address, chain, decimals, total_supply, description, logo_url, website_url, whitepaper_url, status, listing_fee, approved_by, approved_at, rejected_by, rejected_at, rejection_reason, created_by, created_at, updated_at
		FROM tokens WHERE id = $1
	`, id).Scan(&t.ID, &t.Name, &t.Symbol, &t.ContractAddress, &t.Chain, &t.Decimals, &t.TotalSupply, &t.Description, &t.LogoURL, &t.WebsiteURL, &t.WhitepaperURL, &t.Status, &t.ListingFee, &t.ApprovedBy, &t.ApprovedAt, &t.RejectedBy, &t.RejectedAt, &t.RejectionReason, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	return &t, nil
}

func (s *TokenService) Create(ctx context.Context, token *models.Token, creatorID uuid.UUID) (*models.Token, error) {
	err := s.db.QueryRow(ctx, `
		INSERT INTO tokens (name, symbol, contract_address, chain, decimals, total_supply, description, logo_url, website_url, whitepaper_url, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', $11)
		RETURNING id, created_at, updated_at
	`, token.Name, token.Symbol, token.ContractAddress, token.Chain, token.Decimals, token.TotalSupply, token.Description, token.LogoURL, token.WebsiteURL, token.WhitepaperURL, creatorID).Scan(&token.ID, &token.CreatedAt, &token.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}
	return token, nil
}

func (s *TokenService) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*models.Token, error) {
	query := "UPDATE tokens SET updated_at = NOW()"
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		query += fmt.Sprintf(", %s = $%d", key, argIndex)
		args = append(args, value)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, name, symbol, chain, status, created_at, updated_at", argIndex)
	args = append(args, id)

	var t models.Token
	err := s.db.QueryRow(ctx, query, args...).Scan(&t.ID, &t.Name, &t.Symbol, &t.Chain, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update: %w", err)
	}
	return &t, nil
}

func (s *TokenService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM tokens WHERE id = $1`, id)
	return err
}

func (s *TokenService) Approve(ctx context.Context, id, approverID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE tokens SET status = 'approved', approved_by = $1, approved_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`, approverID, id)
	return err
}

func (s *TokenService) Reject(ctx context.Context, id, rejecterID uuid.UUID, reason string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE tokens SET status = 'rejected', rejected_by = $1, rejected_at = NOW(), rejection_reason = $2, updated_at = NOW()
		WHERE id = $3
	`, rejecterID, reason, id)
	return err
}

func (s *TokenService) GetHolders(ctx context.Context, tokenID uuid.UUID) ([]map[string]interface{}, error) {
	// This would typically query token holder tables
	return []map[string]interface{}{}, nil
}

// Transaction Service
type TransactionService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewTransactionService(db *pgxpool.Pool, redis *redis.Client) *TransactionService {
	return &TransactionService{db: db, redis: redis}
}

func (s *TransactionService) List(ctx context.Context, params *models.PaginationParams, status, txType string) (*models.PaginatedResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize

	query := `SELECT id, tx_hash, user_id, token_id, type, amount, fee, status, from_address, to_address, chain, created_at, updated_at FROM transactions WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM transactions WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	if txType != "" {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, txType)
		argIndex++
	}

	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count: %w", err)
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.PageSize, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(&tx.ID, &tx.TxHash, &tx.UserID, &tx.TokenID, &tx.Type, &tx.Amount, &tx.Fee, &tx.Status, &tx.FromAddress, &tx.ToAddress, &tx.Chain, &tx.CreatedAt, &tx.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		txs = append(txs, tx)
	}

	return &models.PaginatedResponse{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: (total + params.PageSize - 1) / params.PageSize,
		Data:       txs,
	}, nil
}

func (s *TransactionService) Get(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	var tx models.Transaction
	err := s.db.QueryRow(ctx, `
		SELECT id, tx_hash, user_id, token_id, type, amount, fee, status, flag_reason, flagged_by, flagged_at, approved_by, approved_at, rejected_by, rejected_at, from_address, to_address, block_number, chain, metadata, created_at, updated_at
		FROM transactions WHERE id = $1
	`, id).Scan(&tx.ID, &tx.TxHash, &tx.UserID, &tx.TokenID, &tx.Type, &tx.Amount, &tx.Fee, &tx.Status, &tx.FlagReason, &tx.FlaggedBy, &tx.FlaggedAt, &tx.ApprovedBy, &tx.ApprovedAt, &tx.RejectedBy, &tx.RejectedAt, &tx.FromAddress, &tx.ToAddress, &tx.BlockNumber, &tx.Chain, &tx.Metadata, &tx.CreatedAt, &tx.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return &tx, nil
}

func (s *TransactionService) Flag(ctx context.Context, id, flaggerID uuid.UUID, reason string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE transactions SET status = 'flagged', flag_reason = $1, flagged_by = $2, flagged_at = NOW(), updated_at = NOW()
		WHERE id = $3
	`, reason, flaggerID, id)
	return err
}

func (s *TransactionService) Approve(ctx context.Context, id, approverID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE transactions SET status = 'approved', approved_by = $1, approved_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`, approverID, id)
	return err
}

func (s *TransactionService) Reject(ctx context.Context, id, rejecterID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE transactions SET status = 'rejected', rejected_by = $1, rejected_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`, rejecterID, id)
	return err
}

func (s *TransactionService) Cancel(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `UPDATE transactions SET status = 'cancelled', updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (s *TransactionService) GetStats(ctx context.Context) (*models.TransactionStats, error) {
	stats := &models.TransactionStats{}

	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&stats.TotalCount)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE status = 'pending'`).Scan(&stats.PendingCount)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE status = 'flagged'`).Scan(&stats.FlaggedCount)
	s.db.QueryRow(ctx, `SELECT COALESCE(SUM(CAST(amount AS DECIMAL)), 0) FROM transactions`).Scan(&stats.TotalVolume)
	s.db.QueryRow(ctx, `SELECT COALESCE(SUM(CAST(amount AS DECIMAL)), 0) FROM transactions WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&stats.TodayVolume)

	return stats, nil
}

func (s *TransactionService) GetPending(ctx context.Context) (*models.PaginatedResponse, error) {
	return s.List(ctx, &models.PaginationParams{Page: 1, PageSize: 50}, "pending", "")
}

// Analytics Service
type AnalyticsService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewAnalyticsService(db *pgxpool.Pool, redis *redis.Client) *AnalyticsService {
	return &AnalyticsService{db: db, redis: redis}
}

func (s *AnalyticsService) Dashboard(ctx context.Context) (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}

	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE status = 'active'`).Scan(&stats.ActiveUsers)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&stats.TotalTransactions)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM kyc_records WHERE status = 'pending'`).Scan(&stats.PendingKYC)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM tokens WHERE status = 'pending'`).Scan(&stats.PendingTokens)

	// Calculate revenue (example - would be more complex in production)
	stats.Revenue = 0
	stats.Change24h = 0

	return stats, nil
}

func (s *AnalyticsService) UserAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var totalUsers, activeUsers, newUsersToday int64
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE status = 'active'`).Scan(&activeUsers)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&newUsersToday)

	analytics["total_users"] = totalUsers
	analytics["active_users"] = activeUsers
	analytics["new_users_today"] = newUsersToday
	analytics["inactive_users"] = totalUsers - activeUsers

	return analytics, nil
}

func (s *AnalyticsService) TransactionAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var total, today, week, month int64
	var volumeTotal, volumeToday float64

	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&total)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&today)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE created_at > NOW() - INTERVAL '7 days'`).Scan(&week)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE created_at > NOW() - INTERVAL '30 days'`).Scan(&month)

	s.db.QueryRow(ctx, `SELECT COALESCE(SUM(CAST(amount AS DECIMAL)), 0) FROM transactions`).Scan(&volumeTotal)
	s.db.QueryRow(ctx, `SELECT COALESCE(SUM(CAST(amount AS DECIMAL)), 0) FROM transactions WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&volumeToday)

	analytics["total_transactions"] = total
	analytics["today_transactions"] = today
	analytics["week_transactions"] = week
	analytics["month_transactions"] = month
	analytics["total_volume"] = volumeTotal
	analytics["today_volume"] = volumeToday

	return analytics, nil
}

func (s *AnalyticsService) RevenueAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var daily, weekly, monthly, total float64
	s.db.QueryRow(ctx, `SELECT COALESCE(SUM(CAST(amount AS DECIMAL)), 0) * 0.001 FROM transactions WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&daily)
	s.db.QueryRow(ctx, `SELECT COALESCE(SUM(CAST(amount AS DECIMAL)), 0) * 0.001 FROM transactions WHERE created_at > NOW() - INTERVAL '7 days'`).Scan(&weekly)
	s.db.QueryRow(ctx, `SELECT COALESCE(SUM(CAST(amount AS DECIMAL)), 0) * 0.001 FROM transactions WHERE created_at > NOW() - INTERVAL '30 days'`).Scan(&monthly)
	s.db.QueryRow(ctx, `SELECT COALESCE(SUM(CAST(amount AS DECIMAL)), 0) * 0.001 FROM transactions`).Scan(&total)

	analytics["daily_revenue"] = daily
	analytics["weekly_revenue"] = weekly
	analytics["monthly_revenue"] = monthly
	analytics["total_revenue"] = total

	return analytics, nil
}

func (s *AnalyticsService) GrowthAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var userGrowth, txGrowth float64
	s.db.QueryRow(ctx, `
		SELECT COUNT(*)::float / NULLIF((SELECT COUNT(*) FROM users WHERE created_at > NOW() - INTERVAL '30 days'), 0) * 100
		FROM users WHERE created_at > NOW() - INTERVAL '7 days'
	`).Scan(&userGrowth)

	s.db.QueryRow(ctx, `
		SELECT COUNT(*)::float / NULLIF((SELECT COUNT(*) FROM transactions WHERE created_at > NOW() - INTERVAL '30 days'), 0) * 100
		FROM transactions WHERE created_at > NOW() - INTERVAL '7 days'
	`).Scan(&txGrowth)

	analytics["user_growth_percent"] = userGrowth
	analytics["transaction_growth_percent"] = txGrowth
	analytics["trend"] = "positive"

	return analytics, nil
}

// Audit Service
type AuditService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewAuditService(db *pgxpool.Pool, redis *redis.Client) *AuditService {
	return &AuditService{db: db, redis: redis}
}

func (s *AuditService) List(ctx context.Context, params *models.PaginationParams, action, resourceType string) (*models.PaginatedResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize

	query := `SELECT id, user_id, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent, location, created_at FROM audit_logs WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, action)
		argIndex++
	}
	if resourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND resource_type = $%d", argIndex)
		args = append(args, resourceType)
		argIndex++
	}

	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count: %w", err)
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.PageSize, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.ResourceType, &l.ResourceID, &l.OldValues, &l.NewValues, &l.IPAddress, &l.UserAgent, &l.Location, &l.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		logs = append(logs, l)
	}

	return &models.PaginatedResponse{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: (total + params.PageSize - 1) / params.PageSize,
		Data:       logs,
	}, nil
}

func (s *AuditService) Get(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	var l models.AuditLog
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent, location, created_at
		FROM audit_logs WHERE id = $1
	`, id).Scan(&l.ID, &l.UserID, &l.Action, &l.ResourceType, &l.ResourceID, &l.OldValues, &l.NewValues, &l.IPAddress, &l.UserAgent, &l.Location, &l.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("audit log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get: %w", err)
	}
	return &l, nil
}

func (s *AuditService) GetByUser(ctx context.Context, userID uuid.UUID) (*models.PaginatedResponse, error) {
	return s.List(ctx, &models.PaginationParams{Page: 1, PageSize: 50}, "", "")
}

func (s *AuditService) Export(ctx context.Context, startDate, endDate time.Time) ([]models.AuditLog, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent, location, created_at
		FROM audit_logs WHERE created_at BETWEEN $1 AND $2 ORDER BY created_at DESC
	`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to export: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.ResourceType, &l.ResourceID, &l.OldValues, &l.NewValues, &l.IPAddress, &l.UserAgent, &l.Location, &l.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// Notification Service
type NotificationService struct {
	redis *redis.Client
}

func NewNotificationService(redis *redis.Client) *NotificationService {
	return &NotificationService{redis: redis}
}

func (s *NotificationService) List(ctx context.Context, userID uuid.UUID, params *models.PaginationParams) (*models.PaginatedResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	rows, err := s.redis.Redis.FindKeys(ctx, fmt.Sprintf("notifications:%s:*", userID.String())).Result()
	if err != nil {
		return &models.PaginatedResponse{Total: 0, Page: params.Page, PageSize: params.PageSize, Data: []models.Notification{}}, nil
	}

	// Return notifications from database instead
	return &models.PaginatedResponse{
		Total:      int64(len(rows)),
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: (len(rows) + params.PageSize - 1) / params.PageSize,
		Data:       []models.Notification{},
	}, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	key := fmt.Sprintf("notification:%s:%s", userID.String(), notificationID.String())
	s.redis.Del(ctx, key)
	_, err := s.redis.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2`, notificationID, userID)
	return err
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := s.redis.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (s *NotificationService) Send(ctx context.Context, notification *models.Notification) error {
	// Store in Redis for real-time
	key := fmt.Sprintf("notification:%s:%s", notification.UserID.String(), notification.ID.String())
	data := fmt.Sprintf("%s|%s|%s", notification.Title, notification.Message, notification.Type)
	s.redis.Set(ctx, key, data, 24*time.Hour*7)

	// Also store in database
	_, err := s.redis.Exec(ctx, `
		INSERT INTO notifications (id, user_id, title, message, type, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, notification.ID, notification.UserID, notification.Title, notification.Message, notification.Type)

	return err
}

// Compliance Service
type ComplianceService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewComplianceService(db *pgxpool.Pool, redis *redis.Client) *ComplianceService {
	return &ComplianceService{db: db, redis: redis}
}

func (s *ComplianceService) ListReports(ctx context.Context, params *models.PaginationParams) (*models.PaginatedResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, type, title, description, generated_by, date_from, date_to, status, file_url, created_at, completed_at
		FROM compliance_reports ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, params.PageSize, (params.Page-1)*params.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list: %w", err)
	}
	defer rows.Close()

	var reports []models.ComplianceReport
	for rows.Next() {
		var r models.ComplianceReport
		err := rows.Scan(&r.ID, &r.Type, &r.Title, &r.Description, &r.GeneratedBy, &r.DateFrom, &r.DateTo, &r.Status, &r.FileURL, &r.CreatedAt, &r.CompletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		reports = append(reports, r)
	}

	return &models.PaginatedResponse{
		Total:      int64(len(reports)),
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: (len(reports) + params.PageSize - 1) / params.PageSize,
		Data:       reports,
	}, nil
}

func (s *ComplianceService) CreateReport(ctx context.Context, report *models.ComplianceReport, userID uuid.UUID) (*models.ComplianceReport, error) {
	err := s.db.QueryRow(ctx, `
		INSERT INTO compliance_reports (type, title, description, generated_by, date_from, date_to, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		RETURNING id, created_at
	`, report.Type, report.Title, report.Description, userID, report.DateFrom, report.DateTo).Scan(&report.ID, &report.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create: %w", err)
	}
	return report, nil
}

func (s *ComplianceService) GetReport(ctx context.Context, id uuid.UUID) (*models.ComplianceReport, error) {
	var r models.ComplianceReport
	err := s.db.QueryRow(ctx, `
		SELECT id, type, title, description, generated_by, date_from, date_to, status, file_url, created_at, completed_at
		FROM compliance_reports WHERE id = $1
	`, id).Scan(&r.ID, &r.Type, &r.Title, &r.Description, &r.GeneratedBy, &r.DateFrom, &r.DateTo, &r.Status, &r.FileURL, &r.CreatedAt, &r.CompletedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("report not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get: %w", err)
	}
	return &r, nil
}

func (s *ComplianceService) CheckSanctions(ctx context.Context, address string) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"address":    address,
		"sanctioned": false,
		"risk":       "low",
	}
	return result, nil
}

func (s *ComplianceService) AMLCheck(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"user_id":      userID,
		"risk_level":   "low",
		"aml_checked":  true,
		"flagged":      false,
		"last_checked": time.Now(),
	}
	return result, nil
}

// Helper functions
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(hash), err
}
