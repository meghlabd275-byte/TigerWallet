// FeeService - Fee management service
package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type FeeService struct{}

func NewFeeService() *FeeService {
	return &FeeService{}
}

func (s *FeeService) ListFeeStructures(ctx context.Context) ([]models.FeeStructure, error) {
	rows, err := database.Query(ctx, `
		SELECT id, fee_type, asset, fee_percent, fee_fixed, min_fee, max_fee, tier, is_active, chain_id, created_at, updated_at
		FROM fee_structures ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fees []models.FeeStructure
	for rows.Next() {
		var fee models.FeeStructure
		err := rows.Scan(
			&fee.ID, &fee.FeeType, &fee.Asset, &fee.FeePercent, &fee.FeeFixed,
			&fee.MinFee, &fee.MaxFee, &fee.Tier, &fee.IsActive, &fee.ChainID, &fee.CreatedAt, &fee.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		fees = append(fees, fee)
	}
	return fees, nil
}

func (s *FeeService) CreateFeeStructure(ctx context.Context, fee *models.FeeStructure) (*models.FeeStructure, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO fee_structures (fee_type, asset, fee_percent, fee_fixed, min_fee, max_fee, tier, is_active, chain_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, fee.FeeType, fee.Asset, fee.FeePercent, fee.FeeFixed, fee.MinFee, fee.MaxFee, fee.Tier, fee.IsActive, fee.ChainID).Scan(&fee.ID, &fee.CreatedAt, &fee.UpdatedAt)
	return fee, err
}

func (s *FeeService) UpdateFeeStructure(ctx context.Context, id uuid.UUID, feePercent, feeFixed, minFee, maxFee string, isActive *bool) error {
	_, err := database.Exec(ctx, `
		UPDATE fee_structures SET fee_percent = $1, fee_fixed = $2, min_fee = $3, max_fee = $4, is_active = $5, updated_at = NOW() WHERE id = $6
	`, feePercent, feeFixed, minFee, maxFee, isActive, id)
	return err
}
