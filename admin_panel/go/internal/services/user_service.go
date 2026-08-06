// UserService - User management service
package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]models.User, int, error) {
	var total int
	database.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total)

	rows, err := database.Query(ctx, `
		SELECT id, email, username, wallet_address, kyc_status, status, two_factor_enabled, ip_address, country, created_at, updated_at, last_login
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Email, &user.Username, &user.WalletAddress, &user.KYCStatus,
			&user.Status, &user.TwoFactorEnabled, &user.IPAddress, &user.Country,
			&user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
		)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, total, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := database.QueryRow(ctx, `
		SELECT id, email, username, wallet_address, kyc_status, status, two_factor_enabled, ip_address, country, created_at, updated_at, last_login
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Email, &user.Username, &user.WalletAddress, &user.KYCStatus,
		&user.Status, &user.TwoFactorEnabled, &user.IPAddress, &user.Country,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
	)
	if err == pgx.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return &user, err
}

func (s *UserService) SearchUsers(ctx context.Context, query string) ([]models.User, error) {
	rows, err := database.Query(ctx, `
		SELECT id, email, username, wallet_address, kyc_status, status, two_factor_enabled, ip_address, country, created_at, updated_at, last_login
		FROM users 
		WHERE email ILIKE $1 OR username ILIKE $1 OR wallet_address ILIKE $1
		LIMIT 50
	`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Email, &user.Username, &user.WalletAddress, &user.KYCStatus,
			&user.Status, &user.TwoFactorEnabled, &user.IPAddress, &user.Country,
			&user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (s *UserService) UpdateUserStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := database.Exec(ctx, "UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2", status, id)
	return err
}

func (s *UserService) BanUser(ctx context.Context, id uuid.UUID) error {
	return s.UpdateUserStatus(ctx, id, "banned")
}

func (s *UserService) UnbanUser(ctx context.Context, id uuid.UUID) error {
	return s.UpdateUserStatus(ctx, id, "active")
}

func (s *UserService) SuspendUser(ctx context.Context, id uuid.UUID) error {
	return s.UpdateUserStatus(ctx, id, "suspended")
}

func (s *UserService) GetPlatformStats(ctx context.Context) (*models.PlatformStats, error) {
	stats := &models.PlatformStats{}

	err := database.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	if err != nil {
		return nil, err
	}

	err = database.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE status = 'active'").Scan(&stats.ActiveUsers)
	if err != nil {
		return nil, err
	}

	err = database.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&stats.TotalTransactions)
	if err != nil {
		return nil, err
	}

	// Additional stats would be calculated here
	stats.TotalVolume = 0
	stats.TotalFees = 0

	return stats, nil
}
