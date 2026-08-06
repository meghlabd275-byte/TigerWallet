// BlockchainService - Blockchain management service
package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type BlockchainService struct{}

func NewBlockchainService() *BlockchainService {
	return &BlockchainService{}
}

func (s *BlockchainService) ListBlockchains(ctx context.Context) ([]models.Blockchain, error) {
	rows, err := database.Query(ctx, `
		SELECT id, name, symbol, chain_id, is_evm, rpc_url, explorer_url, native_token, decimals, is_active, avg_gas_price_gwei, created_at
		FROM blockchains ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blockchains []models.Blockchain
	for rows.Next() {
		var bc models.Blockchain
		err := rows.Scan(
			&bc.ID, &bc.Name, &bc.Symbol, &bc.ChainID, &bc.IsEVM, &bc.RPCURL,
			&bc.ExplorerURL, &bc.NativeToken, &bc.Decimals, &bc.IsActive, &bc.AvgGasPriceGwei, &bc.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		blockchains = append(blockchains, bc)
	}
	return blockchains, nil
}

func (s *BlockchainService) CreateBlockchain(ctx context.Context, bc *models.Blockchain) (*models.Blockchain, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO blockchains (name, symbol, chain_id, is_evm, rpc_url, explorer_url, native_token, decimals, is_active, avg_gas_price_gwei, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		RETURNING id, created_at
	`, bc.Name, bc.Symbol, bc.ChainID, bc.IsEVM, bc.RPCURL, bc.ExplorerURL, bc.NativeToken, bc.Decimals, bc.IsActive, bc.AvgGasPriceGwei).Scan(&bc.ID, &bc.CreatedAt)
	return bc, err
}

func (s *BlockchainService) UpdateBlockchain(ctx context.Context, id uuid.UUID, name, rpcURL, explorerURL, avgGasPriceGwei string, isActive *bool) error {
	_, err := database.Exec(ctx, `
		UPDATE blockchains SET name = $1, rpc_url = $2, explorer_url = $3, avg_gas_price_gwei = $4, is_active = $5 WHERE id = $6
	`, name, rpcURL, explorerURL, avgGasPriceGwei, isActive, id)
	return err
}

func (s *BlockchainService) SetStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	_, err := database.Exec(ctx, "UPDATE blockchains SET is_active = $1 WHERE id = $2", isActive, id)
	return err
}
