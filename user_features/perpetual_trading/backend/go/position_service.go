package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
)

// Position represents a trading position
type Position struct {
	ID               string `json:"id"`
	UserID           string `json:"userId"`
	Symbol           string `json:"symbol"`
	Side             string `json:"side"`
	Quantity         string `json:"quantity"`
	EntryPrice       string `json:"entryPrice"`
	MarkPrice        string `json:"markPrice"`
	Leverage         string `json:"leverage"`
	Margin           string `json:"margin"`
	UnrealizedPNL    string `json:"unrealizedPnl"`
	RealizedPNL      string `json:"realizedPnl"`
	LiquidationPrice string `json:"liquidationPrice"`
	MarginType       string `json:"marginType"`
	ROE              string `json:"roe"`
	MarginRatio      string `json:"marginRatio"`
	UpdatedAt        int64  `json:"updatedAt"`
}

// PositionService handles position operations
type PositionService struct {
	mu        sync.RWMutex
	positions map[string]*Position
}

// NewPositionService creates a new position service
func NewPositionService() *PositionService {
	return &PositionService{
		positions: make(map[string]*Position),
	}
}

// GetPositions gets all positions for a user
func (s *PositionService) GetPositions(ctx context.Context, userID string) ([]*Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var positions []*Position
	for _, pos := range s.positions {
		if pos.UserID == userID {
			positions = append(positions, pos)
		}
	}

	return positions, nil
}

// GetPosition gets a position for a user and symbol
func (s *PositionService) GetPosition(ctx context.Context, userID, symbol string) (*Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", userID, symbol)
	pos, ok := s.positions[key]
	if !ok {
		return nil, fmt.Errorf("position not found")
	}

	return pos, nil
}

// ClosePosition closes a position
func (s *PositionService) ClosePosition(ctx context.Context, req ClosePositionRequest) (*ClosePositionResult, error) {
	key := fmt.Sprintf("%s:%s", req.UserID, req.Symbol)

	s.mu.Lock()
	defer s.mu.Unlock()

	pos, ok := s.positions[key]
	if !ok {
		return nil, fmt.Errorf("position not found")
	}

	result := &ClosePositionResult{
		Symbol:         req.Symbol,
		ClosedQuantity: pos.Quantity,
		RealizedPNL:    pos.RealizedPNL,
		AvgExitPrice:   pos.MarkPrice,
	}

	delete(s.positions, key)

	return result, nil
}

// GetMarginInfo reports the margin the user has locked in open positions.
//
// Every figure is summed from the positions actually held. This previously
// declared totalMargin and totalUnrealized, never assigned them, and returned
// the zero-value strings, so the endpoint reported empty margin for a user
// holding open positions. It also reported UsedMargin as the constant "0"
// while AvailableMargin was the total, which is only correct if nothing is
// locked.
func (s *PositionService) GetMarginInfo(ctx context.Context, userID string) (*MarginInfo, error) {
	if userID == "" {
		return nil, fmt.Errorf("userId required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	totalMargin := new(big.Float)
	totalUnrealized := new(big.Float)
	var count int

	for _, pos := range s.positions {
		if pos.UserID != userID {
			continue
		}
		count++

		if margin, ok := parseDecimal(pos.Margin); ok {
			totalMargin.Add(totalMargin, margin)
		}
		if pnl, ok := parseDecimal(pos.UnrealizedPNL); ok {
			totalUnrealized.Add(totalUnrealized, pnl)
		}
	}

	// Equity is collateral plus unrealized PnL. Free margin is that equity less
	// the collateral already committed, floored at zero: a position that has
	// moved against the user reduces what they can commit, and never shows a
	// negative amount available.
	equity := new(big.Float).Add(totalMargin, totalUnrealized)
	available := new(big.Float).Sub(equity, totalMargin)
	if available.Sign() < 0 {
		available = new(big.Float)
	}

	return &MarginInfo{
		UserID:          userID,
		TotalMargin:     formatDecimal(totalMargin),
		AvailableMargin: formatDecimal(available),
		UsedMargin:      formatDecimal(totalMargin),
		TotalUnrealized: formatDecimal(totalUnrealized),
		PositionsCount:  count,
	}, nil
}

// ClosePositionResult represents the result of closing a position
type ClosePositionResult struct {
	Symbol         string `json:"symbol"`
	ClosedQuantity string `json:"closedQuantity"`
	RealizedPNL    string `json:"realizedPnl"`
	AvgExitPrice   string `json:"avgExitPrice"`
}

// MarginInfo represents margin information
type MarginInfo struct {
	UserID          string `json:"userId"`
	TotalMargin     string `json:"totalMargin"`
	AvailableMargin string `json:"availableMargin"`
	UsedMargin      string `json:"usedMargin"`
	TotalUnrealized string `json:"totalUnrealized"`
	PositionsCount  int    `json:"positionsCount"`
}

// PositionToJSON converts position to JSON
func PositionToJSON(pos *Position) string {
	data, _ := json.Marshal(pos)
	return string(data)
}
