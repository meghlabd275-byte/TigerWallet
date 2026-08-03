/**
 * TigerWallet Perpetual Trading Service
 * 
 * Complete perpetual/futures trading service with position management,
 * margin handling, and liquidation.
 * Built with Go for high-load distributed operations.
 */

package perpetual

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Types
// ============================================================================

// Market represents a perpetual market
type Market struct {
	ID              string        `json:"id"`
	Pair            string        `json:"pair"` // BTC-PERP, ETH-PERP, etc.
	BaseAsset       string        `json:"base_asset"`
	QuoteAsset      string        `json:"quote_asset"`
	ContractSize    string        `json:"contract_size"`
	MaxLeverage     string        `json:"max_leverage"`
	MinLeverage     string        `json:"min_leverage"`
	MaintenanceMargin string      `json:"maintenance_margin"`
	InitialMargin   string        `json:"initial_margin"`
	FundingRate     string        `json:"funding_rate"`
	NextFundingTime int64        `json:"next_funding_time"`
	OpenInterest    string        `json:"open_interest"`
	Volume24h       string        `json:"volume_24h"`
	MarkPrice      string        `json:"mark_price"`
	IndexPrice      string        `json:"index_price"`
	PriceChange24h  string        `json:"price_change_24h"`
	High24h        string        `json:"high_24h"`
	Low24h          string        `json:"low_24h"`
	Status          MarketStatus  `json:"status"`
	CreatedAt       int64         `json:"created_at"`
	UpdatedAt       int64         `json:"updated_at"`
}

// MarketStatus represents market status
type MarketStatus string

const (
	MarketStatusActive   MarketStatus = "active"
	MarketStatusPaused  MarketStatus = "paused"
	MarketStatusClosed  MarketStatus = "closed"
)

// Position represents a trading position
type Position struct {
	ID             string        `json:"id"`
	UserID         string        `json:"user_id"`
	MarketID       string        `json:"market_id"`
	Side           string        `json:"side"` // long, short
	Size           string        `json:"size"`
	EntryPrice     string        `json:"entry_price"`
	MarkPrice      string        `json:"mark_price"`
	Leverage       string        `json:"leverage"`
	InitialMargin  string        `json:"initial_margin"`
	MaintenanceMargin string     `json:"maintenance_margin"`
	UnrealizedPnl  string        `json:"unrealized_pnl"`
	RealizedPnl    string        `json:"realized_pnl"`
	StopLoss       string        `json:"stop_loss"`
	TakeProfit     string        `json:"take_profit"`
	Status         PositionStatus `json:"status"`
	OpenedAt       int64         `json:"opened_at"`
	UpdatedAt      int64         `json:"updated_at"`
}

// PositionStatus represents position status
type PositionStatus string

const (
	PositionStatusOpen   PositionStatus = "open"
	PositionStatusClosed PositionStatus = "closed"
	PositionStatusLiquidated PositionStatus = "liquidated"
)

// Order represents a trading order
type Order struct {
	ID            string      `json:"id"`
	UserID        string      `json:"user_id"`
	MarketID      string      `json:"market_id"`
	Type          OrderType   `json:"type"` // market, limit, stop_market, stop_limit
	Side          string      `json:"side"`
	Price         string      `json:"price"`
	TriggerPrice  string      `json:"trigger_price"`
	Size          string      `json:"size"`
	FilledSize    string      `json:"filled_size"`
	AvgFillPrice  string      `json:"avg_fill_price"`
	Fee           string      `json:"fee"`
	Status        OrderStatus `json:"status"`
	CreatedAt     int64       `json:"created_at"`
	UpdatedAt     int64       `json:"updated_at"`
}

// OrderType represents order type
type OrderType string

const (
	OrderTypeMarket       OrderType = "market"
	OrderTypeLimit       OrderType = "limit"
	OrderTypeStopMarket  OrderType = "stop_market"
	OrderTypeStopLimit   OrderType = "stop_limit"
)

// OrderStatus represents order status
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusOpen     OrderStatus = "open"
	OrderStatusPartial  OrderStatus = "partial"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Liquidation represents a liquidation event
type Liquidation struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	MarketID      string    `json:"market_id"`
	PositionID    string    `json:"position_id"`
	Side          string    `json:"side"`
	Size          string    `json:"size"`
	EntryPrice    string    `json:"entry_price"`
	LiqPrice      string    `json:"liq_price"`
	ClosePrice    string    `json:"close_price"`
	Fee           string    `json:"fee"`
	Status        string    `json:"status"`
	Timestamp     int64     `json:"timestamp"`
}

// FundingPayment represents funding payment
type FundingPayment struct {
	ID           string    `json:"id"`
	MarketID     string    `json:"market_id"`
	UserID       string    `json:"user_id"`
	PositionID   string    `json:"position_id"`
	Rate         string    `json:"rate"`
	Payment      string    `json:"payment"`
	CumulativeRate string  `json:"cumulative_rate"`
	Timestamp    int64     `json:"timestamp"`
}

// Account represents user account
type Account struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	TotalBalance  string    `json:"total_balance"`
	AvailableBalance string `json:"available_balance"`
	TotalPositionValue string `json:"total_position_value"`
	UnrealizedPnl string   `json:"unrealized_pnl"`
	RealizedPnl   string   `json:"realized_pnl"`
	TotalFee      string   `json:"total_fee"`
	MarginRatio   string   `json:"margin_ratio"`
	UpdatedAt     int64     `json:"updated_at"`
}

// PerpetualService manages perpetual trading operations
type PerpetualService struct {
	mu        sync.RWMutex
	markets   map[string]*Market
	positions map[string]*Position
	orders   map[string]*Order
	liquidations map[string]*Liquidation
	fundings map[string]*FundingPayment
	accounts map[string]*Account
}

// ============================================================================
// Service Methods
// ============================================================================

var (
	perpetualService     *PerpetualService
	perpetualServiceOnce sync.Once
)

// GetPerpetualService returns the singleton perpetual service
func GetPerpetualService() *PerpetualService {
	perpetualServiceOnce.Do(func() {
		perpetualService = &PerpetualService{
			markets:      make(map[string]*Market),
			positions:    make(map[string]*Position),
			orders:       make(map[string]*Order),
			liquidations: make(map[string]*Liquidation),
			fundings:     make(map[string]*FundingPayment),
			accounts:     make(map[string]*Account),
		}
	})
	return perpetualService
}

// ============================================================================
// Market Operations
// ============================================================================

// CreateMarket creates a new perpetual market
func (s *PerpetualService) CreateMarket(ctx context.Context, market *Market) (*Market, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	market.ID = "market_" + uuid.New().String()
	market.Status = MarketStatusActive
	market.OpenInterest = "0"
	market.Volume24h = "0"
	market.PriceChange24h = "0"
	market.MarkPrice = "0"
	market.IndexPrice = "0"
	market.CreatedAt = time.Now().Unix()
	market.UpdatedAt = time.Now().Unix()

	s.markets[market.ID] = market
	return market, nil
}

// GetMarket returns a market
func (s *PerpetualService) GetMarket(ctx context.Context, marketID string) (*Market, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	market, exists := s.markets[marketID]
	if !exists {
		return nil, fmt.Errorf("market not found")
	}
	return market, nil
}

// GetMarketByPair returns market by pair
func (s *PerpetualService) GetMarketByPair(ctx context.Context, pair string) (*Market, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, market := range s.markets {
		if market.Pair == pair {
			return market, nil
		}
	}
	return nil, fmt.Errorf("market not found")
}

// GetAllMarkets returns all markets
func (s *PerpetualService) GetAllMarkets(ctx context.Context, status string) ([]*Market, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Market, 0)
	for _, market := range s.markets {
		if status == "" || string(market.Status) == status {
			result = append(result, market)
		}
	}
	return result, nil
}

// UpdateMarket updates a market
func (s *PerpetualService) UpdateMarket(ctx context.Context, market *Market) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.markets[market.ID]
	if !exists {
		return fmt.Errorf("market not found")
	}

	existing.MarkPrice = market.MarkPrice
	existing.IndexPrice = market.IndexPrice
	existing.PriceChange24h = market.PriceChange24h
	existing.High24h = market.High24h
	existing.Low24h = market.Low24h
	existing.Volume24h = market.Volume24h
	existing.OpenInterest = market.OpenInterest
	existing.FundingRate = market.FundingRate
	existing.NextFundingTime = market.NextFundingTime
	existing.UpdatedAt = time.Now().Unix()

	return nil
}

// UpdateMarketStatus updates market status
func (s *PerpetualService) UpdateMarketStatus(ctx context.Context, marketID string, status MarketStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	market, exists := s.markets[marketID]
	if !exists {
		return fmt.Errorf("market not found")
	}

	market.Status = status
	market.UpdatedAt = time.Now().Unix()
	return nil
}

// ============================================================================
// Position Operations
// ============================================================================

// OpenPosition opens a new position
func (s *PerpetualService) OpenPosition(ctx context.Context, position *Position) (*Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify market exists
	market, exists := s.markets[position.MarketID]
	if !exists {
		return nil, fmt.Errorf("market not found")
	}

	if market.Status != MarketStatusActive {
		return nil, fmt.Errorf("market not active")
	}

	position.ID = "position_" + uuid.New().String()
	position.Status = PositionStatusOpen
	position.UnrealizedPnl = "0"
	position.RealizedPnl = "0"
	position.OpenedAt = time.Now().Unix()
	position.UpdatedAt = time.Now().Unix()

	s.positions[position.ID] = position

	// Update market open interest
	oi, _ := new(big.Int).SetString(market.OpenInterest, 10)
	size, _ := new(big.Int).SetString(position.Size, 10)
	oi.Add(oi, size)
	market.OpenInterest = oi.String()

	return position, nil
}

// UpdatePosition updates a position
func (s *PerpetualService) UpdatePosition(ctx context.Context, position *Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.positions[position.ID]
	if !exists {
		return fmt.Errorf("position not found")
	}

	existing.Size = position.Size
	existing.MarkPrice = position.MarkPrice
	existing.UnrealizedPnl = position.UnrealizedPnl
	existing.StopLoss = position.StopLoss
	existing.TakeProfit = position.TakeProfit
	existing.UpdatedAt = time.Now().Unix()

	return nil
}

// ClosePosition closes a position
func (s *PerpetualService) ClosePosition(ctx context.Context, positionID, exitPrice string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	position, exists := s.positions[positionID]
	if !exists {
		return "0", fmt.Errorf("position not found")
	}

	if position.Status != PositionStatusOpen {
		return "0", fmt.Errorf("position not open")
	}

	// Calculate realized PnL
	entryPrice, _ := new(big.Float).SetString(position.EntryPrice)
	exitP, _ := new(big.Float).SetString(exitPrice)
	size, _ := new(big.Float).SetString(position.Size)

	var pnl float64
	if position.Side == "long" {
		pnl, _ = new(big.Float).Sub(exitP, entryPrice).Float64()
	} else {
		pnl, _ = new(big.Float).Sub(entryPrice, exitP).Float64()
	}

	contractSize, _ := new(big.Float).SetString(market.ContractSize)
	pnlFloat := new(big.Float).Mul(big.NewFloat(pnl), contractSize)
	position.RealizedPnl, _ = pnlFloat.String()
	position.Status = PositionStatusClosed
	position.UpdatedAt = time.Now().Unix()

	// Update market open interest
	market, _ := s.markets[position.MarketID]
	if market != nil {
		oi, _ := new(big.Int).SetString(market.OpenInterest, 10)
		posSize, _ := new(big.Int).SetString(position.Size, 10)
		oi.Sub(oi, posSize)
		market.OpenInterest = oi.String()
	}

	return position.RealizedPnl, nil
}

// GetPosition returns a position
func (s *PerpetualService) GetPosition(ctx context.Context, positionID string) (*Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	position, exists := s.positions[positionID]
	if !exists {
		return nil, fmt.Errorf("position not found")
	}
	return position, nil
}

// GetUserPosition returns user's position in a market
func (s *PerpetualService) GetUserPosition(ctx context.Context, userID, marketID string) (*Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, position := range s.positions {
		if position.UserID == userID && position.MarketID == marketID && position.Status == PositionStatusOpen {
			return position, nil
		}
	}
	return nil, nil
}

// GetUserPositions returns all positions for a user
func (s *PerpetualService) GetUserPositions(ctx context.Context, userID string) ([]*Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Position, 0)
	for _, position := range s.positions {
		if position.UserID == userID && position.Status == PositionStatusOpen {
			result = append(result, position)
		}
	}
	return result, nil
}

// GetAllPositions returns all open positions for a market
func (s *PerpetualService) GetAllPositions(ctx context.Context, marketID string) ([]*Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Position, 0)
	for _, position := range s.positions {
		if position.MarketID == marketID && position.Status == PositionStatusOpen {
			result = append(result, position)
		}
	}
	return result, nil
}

// SetStopLoss sets stop loss for a position
func (s *PerpetualService) SetStopLoss(ctx context.Context, positionID, stopLoss string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	position, exists := s.positions[positionID]
	if !exists {
		return fmt.Errorf("position not found")
	}

	position.StopLoss = stopLoss
	position.UpdatedAt = time.Now().Unix()
	return nil
}

// SetTakeProfit sets take profit for a position
func (s *PerpetualService) SetTakeProfit(ctx context.Context, positionID, takeProfit string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	position, exists := s.positions[positionID]
	if !exists {
		return fmt.Errorf("position not found")
	}

	position.TakeProfit = takeProfit
	position.UpdatedAt = time.Now().Unix()
	return nil
}

// ============================================================================
// Order Operations
// ============================================================================

// CreateOrder creates a new order
func (s *PerpetualService) CreateOrder(ctx context.Context, order *Order) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify market exists
	market, exists := s.markets[order.MarketID]
	if !exists {
		return nil, fmt.Errorf("market not found")
	}

	if market.Status != MarketStatusActive {
		return nil, fmt.Errorf("market not active")
	}

	order.ID = "order_" + uuid.New().String()
	order.FilledSize = "0"
	order.AvgFillPrice = "0"
	order.Fee = "0"
	order.Status = OrderStatusPending
	order.CreatedAt = time.Now().Unix()
	order.UpdatedAt = time.Now().Unix()

	s.orders[order.ID] = order
	return order, nil
}

// ExecuteOrder executes an order
func (s *PerpetualService) ExecuteOrder(ctx context.Context, orderID, fillPrice string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found")
	}

	if order.Status == OrderStatusFilled || order.Status == OrderStatusCancelled {
		return fmt.Errorf("order already filled or cancelled")
	}

	// Calculate fee
	fillSize, _ := new(big.Int).SetString(order.Size, 10)
	price, _ := new(big.Int).SetString(fillPrice, 10)
	fee := new(big.Int).Mul(fillSize, price)
	feeRate := big.NewInt(10) // 0.001%
	feeDivisor := big.NewInt(100000)
	fee.Div(fee, feeDivisor)

	order.FilledSize = order.Size
	order.AvgFillPrice = fillPrice
	order.Fee = fee.String()
	order.Status = OrderStatusFilled
	order.UpdatedAt = time.Now().Unix()

	return nil
}

// CancelOrder cancels an order
func (s *PerpetualService) CancelOrder(ctx context.Context, orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found")
	}

	if order.Status == OrderStatusFilled {
		return fmt.Errorf("order already filled")
	}

	order.Status = OrderStatusCancelled
	order.UpdatedAt = time.Now().Unix()
	return nil
}

// GetOrder returns an order
func (s *PerpetualService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, exists := s.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order not found")
	}
	return order, nil
}

// GetUserOrders returns all orders for a user
func (s *PerpetualService) GetUserOrders(ctx context.Context, userID string) ([]*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Order, 0)
	for _, order := range s.orders {
		if order.UserID == userID {
			result = append(result, order)
		}
	}
	return result, nil
}

// ============================================================================
// Liquidation Operations
// ============================================================================

// LiquidatePosition liquidates a position
func (s *PerpetualService) LiquidatePosition(ctx context.Context, positionID, liqPrice, closePrice, fee string) (*Liquidation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	position, exists := s.positions[positionID]
	if !exists {
		return nil, fmt.Errorf("position not found")
	}

	if position.Status != PositionStatusOpen {
		return nil, fmt.Errorf("position not open")
	}

	liquidation := &Liquidation{
		ID:         "liquidation_" + uuid.New().String(),
		UserID:     position.UserID,
		MarketID:   position.MarketID,
		PositionID: positionID,
		Side:      position.Side,
		Size:       position.Size,
		EntryPrice: position.EntryPrice,
		LiqPrice:   liqPrice,
		ClosePrice: closePrice,
		Fee:        fee,
		Status:     "completed",
		Timestamp:  time.Now().Unix(),
	}

	s.liquidations[liquidation.ID] = liquidation

	// Update position
	position.Status = PositionStatusLiquidated
	position.UpdatedAt = time.Now().Unix()

	// Update market open interest
	market, _ := s.markets[position.MarketID]
	if market != nil {
		oi, _ := new(big.Int).SetString(market.OpenInterest, 10)
		posSize, _ := new(big.Int).SetString(position.Size, 10)
		oi.Sub(oi, posSize)
		market.OpenInterest = oi.String()
	}

	return liquidation, nil
}

// GetLiquidation returns a liquidation
func (s *PerpetualService) GetLiquidation(ctx context.Context, liqID string) (*Liquidation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	liquidation, exists := s.liquidations[liqID]
	if !exists {
		return nil, fmt.Errorf("liquidation not found")
	}
	return liquidation, nil
}

// GetUserLiquidations returns liquidations for a user
func (s *PerpetualService) GetUserLiquidations(ctx context.Context, userID string) ([]*Liquidation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Liquidation, 0)
	for _, liquidation := range s.liquidations {
		if liquidation.UserID == userID {
			result = append(result, liquidation)
		}
	}
	return result, nil
}

// ============================================================================
// Funding Operations
// ============================================================================

// ProcessFunding processes funding payments
func (s *PerpetualService) ProcessFunding(ctx context.Context, marketID, rate string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get all open positions in market
	for _, position := range s.positions {
		if position.MarketID == marketID && position.Status == PositionStatusOpen {
			// Calculate funding payment
			size, _ := new(big.Int).SetString(position.Size, 10)
			r, _ := new(big.Int).SetString(rate, 10)
			payment := new(big.Int).Div(size, r)

			funding := &FundingPayment{
				ID:              "funding_" + uuid.New().String(),
				MarketID:        marketID,
				UserID:          position.UserID,
				PositionID:      position.ID,
				Rate:            rate,
				Payment:         payment.String(),
				CumulativeRate:  "0",
				Timestamp:       time.Now().Unix(),
			}

			s.fundings[funding.ID] = funding
		}
	}

	// Update market next funding time
	market, _ := s.markets[marketID]
	if market != nil {
		market.NextFundingTime = time.Now().Unix() + 8*3600 // 8 hours
		market.FundingRate = rate
	}

	return nil
}

// GetFundingHistory returns funding history
func (s *PerpetualService) GetFundingHistory(ctx context.Context, userID string) ([]*FundingPayment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*FundingPayment, 0)
	for _, funding := range s.fundings {
		if funding.UserID == userID {
			result = append(result, funding)
		}
	}
	return result, nil
}

// ============================================================================
// Account Operations
// ============================================================================

// CreateAccount creates a trading account
func (s *PerpetualService) CreateAccount(ctx context.Context, userID string) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	account := &Account{
		ID:                  "account_" + uuid.New().String(),
		UserID:              userID,
		TotalBalance:        "0",
		AvailableBalance:    "0",
		TotalPositionValue:  "0",
		UnrealizedPnl:       "0",
		RealizedPnl:         "0",
		TotalFee:            "0",
		MarginRatio:         "0",
		UpdatedAt:           time.Now().Unix(),
	}

	s.accounts[account.ID] = account
	return account, nil
}

// GetAccount returns an account
func (s *PerpetualService) GetAccount(ctx context.Context, userID string) (*Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, account := range s.accounts {
		if account.UserID == userID {
			return account, nil
		}
	}
	return nil, fmt.Errorf("account not found")
}

// Deposit deposits funds to account
func (s *PerpetualService) Deposit(ctx context.Context, userID, amount string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	account, exists := s.accounts[userID]
	if !exists {
		account = &Account{
			ID:               "account_" + uuid.New().String(),
			UserID:           userID,
			TotalBalance:     "0",
			AvailableBalance: "0",
			UpdatedAt:        time.Now().Unix(),
		}
		s.accounts[account.ID] = account
	}

	total, _ := new(big.Int).SetString(account.TotalBalance, 10)
	available, _ := new(big.Int).SetString(account.AvailableBalance, 10)
	deposit, _ := new(big.Int).SetString(amount, 10)

	total.Add(total, deposit)
	available.Add(available, deposit)

	account.TotalBalance = total.String()
	account.AvailableBalance = available.String()
	account.UpdatedAt = time.Now().Unix()

	return nil
}

// Withdraw withdraws funds from account
func (s *PerpetualService) Withdraw(ctx context.Context, userID, amount string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var account *Account
	for _, acc := range s.accounts {
		if acc.UserID == userID {
			account = acc
			break
		}
	}

	if account == nil {
		return fmt.Errorf("account not found")
	}

	available, _ := new(big.Int).SetString(account.AvailableBalance, 10)
	withdraw, _ := new(big.Int).SetString(amount, 10)

	if available.Cmp(withdraw) < 0 {
		return fmt.Errorf("insufficient balance")
	}

	available.Sub(available, withdraw)

	account.AvailableBalance = available.String()
	account.UpdatedAt = time.Now().Unix()

	return nil
}

// CalculateMarginRatio calculates margin ratio for a position
func (s *PerpetualService) CalculateMarginRatio(ctx context.Context, positionID string) (string, error) {
	s.mu.RLock()
	position, exists := s.positions[positionID]
	s.mu.RUnlock()

	if !exists {
		return "0", fmt.Errorf("position not found")
	}

	margin, _ := new(big.Float).SetString(position.InitialMargin)
	maintenance, _ := new(big.Float).SetString(position.MaintenanceMargin)

	if maintenance.Cmp(big.NewFloat(0)) == 0 {
		return "0", nil
	}

	ratio := new(big.Float).Quo(margin, maintenance)
	result, _ := ratio.String()

	return result, nil
}

// ============================================================================
// Utility Methods
// ============================================================================

// CalculateUnrealizedPnl calculates unrealized PnL
func (s *PerpetualService) CalculateUnrealizedPnl(ctx context.Context, positionID string) (string, error) {
	s.mu.RLock()
	position, exists := s.positions[positionID]
	s.mu.RUnlock()

	if !exists {
		return "0", fmt.Errorf("position not found")
	}

	entryPrice, _ := new(big.Float).SetString(position.EntryPrice)
	markPrice, _ := new(big.Float).SetString(position.MarkPrice)
	size, _ := new(big.Float).SetString(position.Size)

	var pnl float64
	if position.Side == "long" {
		pnl, _ = new(big.Float).Sub(markPrice, entryPrice).Float64()
	} else {
		pnl, _ = new(big.Float).Sub(entryPrice, markPrice).Float64()
	}

	contractSize, _ := new(big.Float).SetString("1")
	result := new(big.Float).Mul(big.NewFloat(pnl), size)
	result.Mul(result, contractSize)
	resultStr, _ := result.String()

	return resultStr, nil
}

// GetOpenInterest returns total open interest
func (s *PerpetualService) GetOpenInterest(ctx context.Context, marketID string) (string, error) {
	s.mu.RLock()
	market, exists := s.markets[marketID]
	s.mu.RUnlock()

	if !exists {
		return "0", fmt.Errorf("market not found")
	}

	return market.OpenInterest, nil
}

// ToJSON converts position to JSON
func (p *Position) ToJSON() (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Helper for market reference
var market *Market
