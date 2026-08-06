// WhiteLabelService - White label management service
package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type WhiteLabelService struct{}

func NewWhiteLabelService() *WhiteLabelService {
	return &WhiteLabelService{}
}

func (s *WhiteLabelService) ListWhiteLabels(ctx context.Context) ([]models.WhiteLabel, error) {
	rows, err := database.Query(ctx, `
		SELECT id, name, domain, logo_url, primary_color, secondary_color, is_active, created_at
		FROM white_labels ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var whiteLabels []models.WhiteLabel
	for rows.Next() {
		var wl models.WhiteLabel
		err := rows.Scan(&wl.ID, &wl.Name, &wl.Domain, &wl.LogoURL, &wl.PrimaryColor, &wl.SecondaryColor, &wl.IsActive, &wl.CreatedAt)
		if err != nil {
			return nil, err
		}
		whiteLabels = append(whiteLabels, wl)
	}
	return whiteLabels, nil
}

func (s *WhiteLabelService) CreateWhiteLabel(ctx context.Context, wl *models.WhiteLabel) (*models.WhiteLabel, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO white_labels (name, domain, logo_url, primary_color, secondary_color, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`, wl.Name, wl.Domain, wl.LogoURL, wl.PrimaryColor, wl.SecondaryColor, wl.IsActive).Scan(&wl.ID, &wl.CreatedAt)
	return wl, err
}

func (s *WhiteLabelService) UpdateWhiteLabel(ctx context.Context, id uuid.UUID, name, domain, logoURL, primaryColor, secondaryColor string, isActive *bool) error {
	_, err := database.Exec(ctx, `
		UPDATE white_labels SET name = $1, domain = $2, logo_url = $3, primary_color = $4, secondary_color = $5, is_active = $6 WHERE id = $7
	`, name, domain, logoURL, primaryColor, secondaryColor, isActive, id)
	return err
}

func (s *WhiteLabelService) DeleteWhiteLabel(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "DELETE FROM white_labels WHERE id = $1", id)
	return err
}

func (s *WhiteLabelService) GetWhiteLabelByDomain(ctx context.Context, domain string) (*models.WhiteLabel, error) {
	var wl models.WhiteLabel
	err := database.QueryRow(ctx, `
		SELECT id, name, domain, logo_url, primary_color, secondary_color, is_active, created_at
		FROM white_labels WHERE domain = $1 AND is_active = true
	`, domain).Scan(&wl.ID, &wl.Name, &wl.Domain, &wl.LogoURL, &wl.PrimaryColor, &wl.SecondaryColor, &wl.IsActive, &wl.CreatedAt)
	return &wl, err
}
