package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/backend/internal/models"
)

type LendingService struct {
	db *sql.DB
}

func NewLendingService(db *sql.DB) *LendingService {
	return &LendingService{db: db}
}

// Supply assets to lending pool
func (s *LendingService) Supply(ctx context.Context, userID uuid.UUID, token string, amount float64) (*models.LendingPosition, error) {
	// Get current APY
	apy, err := s.getLendingAPY(token)
	if err != nil {
		return nil, err
	}

	// Create lending position
	position := &models.LendingPosition{
		ID:           uuid.New(),
		UserID:       userID,
		Token:        token,
		Supplied:     amount,
		Borrowed:     0,
		APY:          apy,
		Accumulated:  0,
		Status:       "ACTIVE",
		SuppliedAt:   time.Now(),
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO lending_positions (id, user_id, token, supplied, borrowed, apy, accumulated, status, supplied_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, position.ID, position.UserID, position.Token, position.Supplied, position.Borrowed, position.APY, position.Accumulated, position.Status, position.SuppliedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create lending position: %w", err)
	}

	// Update pool liquidity
	s.updatePoolLiquidity(token, amount, "supply")

	return position, nil
}

// Borrow assets from lending pool
func (s *LendingService) Borrow(ctx context.Context, userID uuid.UUID, token string, amount float64) (*models.LendingPosition, error) {
	// Check collateral value
	collateralValue, err := s.getUserCollateralValue(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if borrow is within limits (75% LTV)
	maxBorrow := collateralValue * 0.75
	if amount > maxBorrow {
		return nil, fmt.Errorf("borrow amount exceeds maximum: max=%v, requested=%v", maxBorrow, amount)
	}

	// Get borrow APY
	borrowAPY, err := s.getBorrowAPY(token)
	if err != nil {
		return nil, err
	}

	// Update or create position
	var position models.LendingPosition
	err = s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token, supplied, borrowed, apy, accumulated, status, supplied_at
		FROM lending_positions 
		WHERE user_id = $1 AND token = $2 AND status = 'ACTIVE'
	`, userID, token).Scan(&position.ID, &position.UserID, &position.Token, &position.Supplied, &position.Borrowed, &position.APY, &position.Accumulated, &position.Status, &position.SuppliedAt)

	if err == sql.ErrNoRows {
		// Create new position
		position = models.LendingPosition{
			ID:         uuid.New(),
			UserID:     userID,
			Token:      token,
			Supplied:   0,
			Borrowed:   amount,
			APY:        borrowAPY,
			Status:     "ACTIVE",
			SuppliedAt: time.Now(),
		}

		_, err = s.db.ExecContext(ctx, `
			INSERT INTO lending_positions (id, user_id, token, supplied, borrowed, apy, accumulated, status, supplied_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, position.ID, position.UserID, position.Token, position.Supplied, position.Borrowed, position.APY, position.Accumulated, position.Status, position.SuppliedAt)
	} else {
		// Update existing position
		_, err = s.db.ExecContext(ctx, `
			UPDATE lending_positions 
			SET borrowed = borrowed + $1, apy = $2
			WHERE user_id = $3 AND token = $4 AND status = 'ACTIVE'
		`, amount, borrowAPY, userID, token)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to borrow: %w", err)
	}

	// Update pool liquidity
	s.updatePoolLiquidity(token, amount, "borrow")

	return &position, nil
}

// Repay borrowed assets
func (s *LendingService) Repay(ctx context.Context, userID uuid.UUID, token string, amount float64) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE lending_positions 
		SET borrowed = GREATEST(borrowed - $1, 0)
		WHERE user_id = $2 AND token = $3 AND status = 'ACTIVE'
	`, amount, userID, token)

	if err != nil {
		return fmt.Errorf("failed to repay: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no active position found")
	}

	// Update pool liquidity
	s.updatePoolLiquidity(token, amount, "repay")

	return nil
}

// Withdraw supplied assets
func (s *LendingService) Withdraw(ctx context.Context, userID uuid.UUID, token string, amount float64) error {
	// Check borrowed amount
	var borrowed float64
	err := s.db.QueryRowContext(ctx, `
		SELECT borrowed FROM lending_positions 
		WHERE user_id = $1 AND token = $2 AND status = 'ACTIVE'
	`, userID, token).Scan(&borrowed)

	if err == sql.ErrNoRows {
		return fmt.Errorf("no position found")
	}

	// Check if withdraw would violate health factor
	if borrowed > 0 {
		var supplied float64
		s.db.QueryRowContext(ctx, `
			SELECT supplied FROM lending_positions 
			WHERE user_id = $1 AND token = $2 AND status = 'ACTIVE'
		`, userID, token).Scan(&supplied)

		remainingSupplied := supplied - amount
		if remainingSupplied < borrowed*1.25 {
			return fmt.Errorf("withdraw would liquidate position")
		}
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE lending_positions 
		SET supplied = GREATEST(supplied - $1, 0)
		WHERE user_id = $2 AND token = $3 AND status = 'ACTIVE'
	`, amount, userID, token)

	if err != nil {
		return fmt.Errorf("failed to withdraw: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no active position found")
	}

	// Update pool liquidity
	s.updatePoolLiquidity(token, amount, "withdraw")

	return nil
}

// Get user positions
func (s *LendingService) GetUserPositions(ctx context.Context, userID uuid.UUID) ([]models.LendingPosition, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, token, supplied, borrowed, apy, accumulated, status, supplied_at
		FROM lending_positions 
		WHERE user_id = $1 AND status = 'ACTIVE'
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []models.LendingPosition
	for rows.Next() {
		var p models.LendingPosition
		err := rows.Scan(&p.ID, &p.UserID, &p.Token, &p.Supplied, &p.Borrowed, &p.APY, &p.Accumulated, &p.Status, &p.SuppliedAt)
		if err != nil {
			continue
		}
		positions = append(positions, p)
	}

	return positions, nil
}

// Get pool data
func (s *LendingService) GetPoolData(ctx context.Context, token string) (*models.LendingPool, error) {
	var pool models.LendingPool
	err := s.db.QueryRowContext(ctx, `
		SELECT token, total_supplied, total_borrowed, supply_apy, borrow_apy, liquidity, updated_at
		FROM lending_pools WHERE token = $1
	`, token).Scan(&pool.Token, &pool.TotalSupplied, &pool.TotalBorrowed, &pool.SupplyAPY, &pool.BorrowAPY, &pool.Liquidity, &pool.UpdatedAt)

	if err == sql.ErrNoRows {
		return &models.LendingPool{
			Token:       token,
			SupplyAPY:   0.05,
			BorrowAPY:   0.08,
			Liquidity:   0,
			TotalSupplied: 0,
			TotalBorrowed: 0,
		}, nil
	}

	return &pool, err
}

// Get all pools
func (s *LendingService) GetAllPools(ctx context.Context) ([]models.LendingPool, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT token, total_supplied, total_borrowed, supply_apy, borrow_apy, liquidity, updated_at
		FROM lending_pools
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []models.LendingPool
	for rows.Next() {
		var p models.LendingPool
		err := rows.Scan(&p.Token, &p.TotalSupplied, &p.TotalBorrowed, &p.SupplyAPY, &p.BorrowAPY, &p.Liquidity, &p.UpdatedAt)
		if err != nil {
			continue
		}
		pools = append(pools, p)
	}

	return pools, nil
}

// Helper functions
func (s *LendingService) getLendingAPY(token string) (float64, error) {
	var apy float64
	err := s.db.QueryRow(`
		SELECT supply_apy FROM lending_pools WHERE token = $1
	`, token).Scan(&apy)

	if err == sql.ErrNoRows {
		return 0.05, nil // Default 5%
	}
	return apy, err
}

func (s *LendingService) getBorrowAPY(token string) (float64, error) {
	var apy float64
	err := s.db.QueryRow(`
		SELECT borrow_apy FROM lending_pools WHERE token = $1
	`, token).Scan(&apy)

	if err == sql.ErrNoRows {
		return 0.08, nil // Default 8%
	}
	return apy, err
}

func (s *LendingService) getUserCollateralValue(ctx context.Context, userID uuid.UUID) (float64, error) {
	var value float64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(supplied * token_price), 0) 
		FROM lending_positions lp
		JOIN tokens t ON lp.token = t.symbol
		WHERE user_id = $1 AND status = 'ACTIVE'
	`, userID).Scan(&value)

	return value, err
}

func (s *LendingService) updatePoolLiquidity(token string, amount float64, operation string) {
	switch operation {
	case "supply":
		s.db.Exec(`UPDATE lending_pools SET total_supplied = total_supplied + $1, liquidity = liquidity + $1 WHERE token = $2`, amount, token)
	case "withdraw":
		s.db.Exec(`UPDATE lending_pools SET total_supplied = total_supplied - $1, liquidity = liquidity - $1 WHERE token = $2`, amount, token)
	case "borrow":
		s.db.Exec(`UPDATE lending_pools SET total_borrowed = total_borrowed + $1 WHERE token = $2`, amount, token)
	case "repay":
		s.db.Exec(`UPDATE lending_pools SET total_borrowed = total_borrowed - $1 WHERE token = $2`, amount, token)
	}
}

// Get user health factor
func (s *LendingService) GetHealthFactor(ctx context.Context, userID uuid.UUID) (float64, error) {
	var supplied, borrowed float64

	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(supplied * token_price), 0), COALESCE(SUM(borrowed * token_price), 0)
		FROM lending_positions lp
		JOIN tokens t ON lp.token = t.symbol
		WHERE user_id = $1 AND status = 'ACTIVE'
	`, userID)

	err := row.Scan(&supplied, &borrowed)
	if err != nil {
		return 0, err
	}

	if borrowed == 0 {
		return 999.0, nil // No debt = very healthy
	}

	return supplied / borrowed, nil
}

// Liquidate undercollateralized position
func (s *LendingService) Liquidate(ctx context.Context, liquidatorID uuid.UUID, userID uuid.UUID, token string) error {
	healthFactor, err := s.GetHealthFactor(ctx, userID)
	if err != nil {
		return err
	}

	if healthFactor > 1.0 {
		return fmt.Errorf("position not eligible for liquidation")
	}

	// Execute liquidation
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Transfer collateral to liquidator
	_, err = tx.ExecContext(ctx, `
		UPDATE lending_positions 
		SET status = 'LIQUIDATED', liquidated_by = $1, liquidated_at = NOW()
		WHERE user_id = $2 AND token = $3
	`, liquidatorID, userID, token)

	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
