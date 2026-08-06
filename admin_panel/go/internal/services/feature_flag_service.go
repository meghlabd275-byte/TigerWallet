// FeatureFlagService - Feature flag management service
package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type FeatureFlagService struct{}

func NewFeatureFlagService() *FeatureFlagService {
	return &FeatureFlagService{}
}

func (s *FeatureFlagService) ListFeatureFlags(ctx context.Context) ([]models.FeatureFlag, error) {
	rows, err := database.Query(ctx, `
		SELECT id, name, description, is_enabled, rollout_percentage, created_at, updated_at, updated_by
		FROM feature_flags ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []models.FeatureFlag
	for rows.Next() {
		var flag models.FeatureFlag
		err := rows.Scan(&flag.ID, &flag.Name, &flag.Description, &flag.IsEnabled, &flag.RolloutPercentage, &flag.CreatedAt, &flag.UpdatedAt, &flag.UpdatedBy)
		if err != nil {
			return nil, err
		}
		flags = append(flags, flag)
	}
	return flags, nil
}

func (s *FeatureFlagService) CreateFeatureFlag(ctx context.Context, flag *models.FeatureFlag, adminID uuid.UUID) (*models.FeatureFlag, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO feature_flags (name, description, is_enabled, rollout_percentage, created_at, updated_at, updated_by)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), $5)
		RETURNING id, created_at, updated_at
	`, flag.Name, flag.Description, flag.IsEnabled, flag.RolloutPercentage, adminID).Scan(&flag.ID, &flag.CreatedAt, &flag.UpdatedAt)
	return flag, err
}

func (s *FeatureFlagService) UpdateFeatureFlag(ctx context.Context, id uuid.UUID, adminID uuid.UUID, isEnabled *bool, rolloutPercentage *int) error {
	_, err := database.Exec(ctx, `
		UPDATE feature_flags SET is_enabled = COALESCE($1, is_enabled), rollout_percentage = COALESCE($2, rollout_percentage), updated_at = NOW(), updated_by = $3 WHERE id = $4
	`, isEnabled, rolloutPercentage, adminID, id)
	return err
}

func (s *FeatureFlagService) DeleteFeatureFlag(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "DELETE FROM feature_flags WHERE id = $1", id)
	return err
}

func (s *FeatureFlagService) GetFeatureFlag(ctx context.Context, name string) (*models.FeatureFlag, error) {
	var flag models.FeatureFlag
	err := database.QueryRow(ctx, `
		SELECT id, name, description, is_enabled, rollout_percentage, created_at, updated_at, updated_by
		FROM feature_flags WHERE name = $1
	`, name).Scan(&flag.ID, &flag.Name, &flag.Description, &flag.IsEnabled, &flag.RolloutPercentage, &flag.CreatedAt, &flag.UpdatedAt, &flag.UpdatedBy)
	if err != nil {
		return nil, err
	}
	return &flag, nil
}
