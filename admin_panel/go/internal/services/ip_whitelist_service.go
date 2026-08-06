// IPWhitelistService - IP whitelist management service
package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type IPWhitelistService struct{}

func NewIPWhitelistService() *IPWhitelistService {
	return &IPWhitelistService{}
}

func (s *IPWhitelistService) ListIPWhitelist(ctx context.Context) ([]models.IPWhitelist, error) {
	rows, err := database.Query(ctx, `
		SELECT id, ip_address, description, is_active, created_at, created_by
		FROM ip_whitelist ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.IPWhitelist
	for rows.Next() {
		var entry models.IPWhitelist
		err := rows.Scan(&entry.ID, &entry.IPAddress, &entry.Description, &entry.IsActive, &entry.CreatedAt, &entry.CreatedBy)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *IPWhitelistService) AddIP(ctx context.Context, entry *models.IPWhitelist, adminID uuid.UUID) (*models.IPWhitelist, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO ip_whitelist (ip_address, description, is_active, created_at, created_by)
		VALUES ($1, $2, $3, NOW(), $4)
		RETURNING id, created_at
	`, entry.IPAddress, entry.Description, entry.IsActive, adminID).Scan(&entry.ID, &entry.CreatedAt)
	return entry, err
}

func (s *IPWhitelistService) RemoveIP(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "DELETE FROM ip_whitelist WHERE id = $1", id)
	return err
}

func (s *IPWhitelistService) IsIPAllowed(ctx context.Context, ip string) (bool, error) {
	var count int
	err := database.QueryRow(ctx, `
		SELECT COUNT(*) FROM ip_whitelist WHERE ip_address = $1 AND is_active = true
	`, ip).Scan(&count)
	return count > 0, err
}
