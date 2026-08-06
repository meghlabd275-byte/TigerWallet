// TokenService - Token and trading pair management
package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type TokenService struct{}

func NewTokenService() *TokenService {
	return &TokenService{}
}

func (s *TokenService) ListTokens(ctx context.Context) ([]models.Token, error) {
	rows, err := database.Query(ctx, `
		SELECT id, symbol, name, contract_address, decimals, is_active, is_verified, total_supply, chain_id, created_at
		FROM tokens ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []models.Token
	for rows.Next() {
		var token models.Token
		err := rows.Scan(
			&token.ID, &token.Symbol, &token.Name, &token.ContractAddress,
			&token.Decimals, &token.IsActive, &token.IsVerified, &token.TotalSupply, &token.ChainID, &token.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (s *TokenService) CreateToken(ctx context.Context, token *models.Token) (*models.Token, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO tokens (symbol, name, contract_address, decimals, is_active, is_verified, total_supply, chain_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, created_at
	`, token.Symbol, token.Name, token.ContractAddress, token.Decimals, token.IsActive, token.IsVerified, token.TotalSupply, token.ChainID).Scan(&token.ID, &token.CreatedAt)
	return token, err
}

func (s *TokenService) UpdateToken(ctx context.Context, id uuid.UUID, name string, isActive, isVerified *bool) error {
	// Build dynamic update query
	_, err := database.Exec(ctx, "UPDATE tokens SET name = $1, is_active = $2, is_verified = $3 WHERE id = $4", name, isActive, isVerified, id)
	return err
}

func (s *TokenService) DeleteToken(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "DELETE FROM tokens WHERE id = $1", id)
	return err
}

func (s *TokenService) ListTradingPairs(ctx context.Context, limit, offset int) ([]models.TradingPair, int, error) {
	var total int
	database.QueryRow(ctx, "SELECT COUNT(*) FROM trading_pairs").Scan(&total)

	rows, err := database.Query(ctx, `
		SELECT id, base_token_id, quote_token_id, pair_name, price, volume_24h, liquidity, status, chain_id, created_at, updated_at
		FROM trading_pairs ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pairs []models.TradingPair
	for rows.Next() {
		var pair models.TradingPair
		err := rows.Scan(
			&pair.ID, &pair.BaseTokenID, &pair.QuoteTokenID, &pair.PairName,
			&pair.Price, &pair.Volume24h, &pair.Liquidity, &pair.Status, &pair.ChainID, &pair.CreatedAt, &pair.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, total, nil
}

func (s *TokenService) CreateTradingPair(ctx context.Context, pair *models.TradingPair) (*models.TradingPair, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO trading_pairs (base_token_id, quote_token_id, pair_name, price, volume_24h, liquidity, status, chain_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, pair.BaseTokenID, pair.QuoteTokenID, pair.PairName, pair.Price, pair.Volume24h, pair.Liquidity, pair.Status, pair.ChainID).Scan(&pair.ID, &pair.CreatedAt, &pair.UpdatedAt)
	return pair, err
}

func (s *TokenService) UpdatePairStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := database.Exec(ctx, "UPDATE trading_pairs SET status = $1, updated_at = NOW() WHERE id = $2", status, id)
	return err
}
