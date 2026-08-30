package main

import (
	"context"
	"encoding/json"
	"fmt"
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

// GetMarginInfo gets margin information for a user
func (s *PositionService) GetMarginInfo(ctx context.Context, userID string) (*MarginInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalMargin, totalUnrealized string
	var count int

	for _, pos := range s.positions {
		if pos.UserID == userID {
			count++
		}
	}

	return &MarginInfo{
		UserID:          userID,
		TotalMargin:     totalMargin,
		AvailableMargin: totalMargin,
		UsedMargin:      "0",
		TotalUnrealized: totalUnrealized,
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
