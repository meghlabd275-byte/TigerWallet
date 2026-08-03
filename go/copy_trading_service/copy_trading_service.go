/**
 * TigerWallet Copy Trading Service
 * 
 * Complete copy trading service with signal copying, strategy replication,
 * and trader management.
 * Built with Go for high-load distributed operations.
 */

package copytrading

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

// Trader represents a trader to copy
type Trader struct {
	ID            string       `json:"id"`
	Address       string       `json:"address"`
	Name          string       `json:"name"`
	AvatarURL     string       `json:"avatar_url"`
	Bio           string       `json:"bio"`
	Strategy      string       `json:"strategy"`
	WinRate       string       `json:"win_rate"`
	TotalPnl     string       `json:"total_pnl"`
	TotalTrades  int          `json:"total_trades"`
	ActiveCopiers int         `json:"active_copiers"`
	TotalCopiers  int         `json:"_total_copiers"`
	AUM           string       `json:"aum"` // assets under management
	FollowLimit   string       `json:"follow_limit"`
	Performance   *Performance `json:"performance"`
	IsVerified    bool         `json:"is_verified"`
	IsPro        bool         `json:"is_pro"`
	Status       TraderStatus `json:"status"`
	CreatedAt    int64        `json:"created_at"`
	UpdatedAt    int64        `json:"updated_at"`
}

// Performance represents trader performance
type Performance struct {
	DailyPnl    string `json:"daily_pnl"`
	WeeklyPnl   string `json:"weekly_pnl"`
	MonthlyPnl  string `json:"monthly_pnl"`
	YearlyPnl   string `json:"yearly_pnl"`
	BestTrade   string `json:"best_trade"`
	WorstTrade  string `json:"worst_trade"`
	AvgWin      string `json:"avg_win"`
	AvgLoss     string `json:"avg_loss"`
}

// TraderStatus represents trader status
type TraderStatus string

const (
	TraderStatusActive   TraderStatus = "active"
	TraderStatusPaused  TraderStatus = "paused"
	TraderStatusClosed  TraderStatus = "closed"
)

// Copier represents a copier
type Copier struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	TraderID      string    `json:"trader_id"`
	CopyRatio     string    `json:"copy_ratio"` // 0.1 - 10.0
	Allocation    string    `json:"allocation"`
	Status        string    `json:"status"`
	TotalPnl      string    `json:"total_pnl"`
	TotalTrades   int       `json:"total_trades"`
	LastTradeTime int64     `json:"last_trade_time"`
	CreatedAt     int64     `json:"created_at"`
	UpdatedAt     int64     `json:"updated_at"`
}

// Signal represents a trading signal
type Signal struct {
	ID           string     `json:"id"`
	TraderID     string     `json:"trader_id"`
	Type         SignalType `json:"type"` // open_position, close_position, update_position
	Pair         string     `json:"pair"`
	Side         string     `json:"side"` // long, short
	EntryPrice   string     `json:"entry_price"`
	StopLoss     string     `json:"stop_loss"`
	TakeProfit   string     `json:"take_profit"`
	Quantity     string     `json:"quantity"`
	OrderType    string     `json:"order_type"` // market, limit
	Leverage     string     `json:"leverage"`
	Timestamp    int64      `json:"timestamp"`
	Executed     bool       `json:"executed"`
	ExecutedAt   int64      `json:"executed_at"`
}

// SignalType represents signal type
type SignalType string

const (
	SignalTypeOpen     SignalType = "open_position"
	SignalTypeClose    SignalType = "close_position"
	SignalTypeUpdate   SignalType = "update_position"
)

// CopiedTrade represents a copied trade
type CopiedTrade struct {
	ID           string    `json:"id"`
	CopierID    string    `json:"copier_id"`
	TraderID    string    `json:"trader_id"`
	SignalID    string    `json:"signal_id"`
	Pair        string    `json:"pair"`
	Side        string    `json:"side"`
	EntryPrice  string    `json:"entry_price"`
	ExitPrice   string    `json:"exit_price"`
	Quantity    string    `json:"quantity"`
	Pnl         string    `json:"pnl"`
	Fee         string    `json:"fee"`
	Status      string    `json:"status"` // open, closed
	OpenedAt    int64     `json:"opened_at"`
	ClosedAt    int64     `json:"closed_at"`
}

// CopyTradingService manages copy trading operations
type CopyTradingService struct {
	mu         sync.RWMutex
	traders    map[string]*Trader
	copiers    map[string]*Copier
	signals    map[string]*Signal
	trades     map[string]*CopiedTrade
}

// ============================================================================
// Service Methods
// ============================================================================

var (
	copyTradingService     *CopyTradingService
	copyTradingServiceOnce sync.Once
)

// GetCopyTradingService returns the singleton copy trading service
func GetCopyTradingService() *CopyTradingService {
	copyTradingServiceOnce.Do(func() {
		copyTradingService = &CopyTradingService{
			traders: make(map[string]*Trader),
			copiers: make(map[string]*Copier),
			signals: make(map[string]*Signal),
			trades:  make(map[string]*CopiedTrade),
		}
	})
	return copyTradingService
}

// ============================================================================
// Trader Operations
// ============================================================================

// RegisterTrader registers a new trader
func (s *CopyTradingService) RegisterTrader(ctx context.Context, trader *Trader) (*Trader, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	trader.ID = "trader_" + uuid.New().String()
	trader.Status = TraderStatusActive
	trader.TotalPnl = "0"
	trader.TotalTrades = 0
	trader.ActiveCopiers = 0
	trader.TotalCopiers = 0
	trader.AUM = "0"
	trader.CreatedAt = time.Now().Unix()
	trader.UpdatedAt = time.Now().Unix()

	s.traders[trader.ID] = trader
	return trader, nil
}

// GetTrader returns a trader
func (s *CopyTradingService) GetTrader(ctx context.Context, traderID string) (*Trader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trader, exists := s.traders[traderID]
	if !exists {
		return nil, fmt.Errorf("trader not found")
	}
	return trader, nil
}

// GetTraderByAddress returns trader by address
func (s *CopyTradingService) GetTraderByAddress(ctx context.Context, address string) (*Trader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, trader := range s.traders {
		if trader.Address == address {
			return trader, nil
		}
	}
	return nil, fmt.Errorf("trader not found")
}

// GetAllTraders returns all traders
func (s *CopyTradingService) GetAllTraders(ctx context.Context, status string) ([]*Trader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Trader, 0)
	for _, trader := range s.traders {
		if status == "" || string(trader.Status) == status {
			result = append(result, trader)
		}
	}
	return result, nil
}

// GetTopTraders returns top traders by performance
func (s *CopyTradingService) GetTopTraders(ctx context.Context, limit int) ([]*Trader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Trader, 0)
	for _, trader := range s.traders {
		if trader.Status == TraderStatusActive {
			result = append(result, trader)
		}
	}

	// Sort by total PnL
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			pnlI, _ := new(big.Int).SetString(result[i].TotalPnl, 10)
			pnlJ, _ := new(big.Int).SetString(result[j].TotalPnl, 10)
			if pnlJ.Cmp(pnlI) > 0 {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// UpdateTrader updates trader info
func (s *CopyTradingService) UpdateTrader(ctx context.Context, trader *Trader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.traders[trader.ID]
	if !exists {
		return fmt.Errorf("trader not found")
	}

	existing.Name = trader.Name
	existing.AvatarURL = trader.AvatarURL
	existing.Bio = trader.Bio
	existing.Strategy = trader.Strategy
	existing.FollowLimit = trader.FollowLimit
	existing.UpdatedAt = time.Now().Unix()

	return nil
}

// UpdateTraderStatus updates trader status
func (s *CopyTradingService) UpdateTraderStatus(ctx context.Context, traderID string, status TraderStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	trader, exists := s.traders[traderID]
	if !exists {
		return fmt.Errorf("trader not found")
	}

	trader.Status = status
	trader.UpdatedAt = time.Now().Unix()
	return nil
}

// ============================================================================
// Copier Operations
// ============================================================================

// StartCopying starts copying a trader
func (s *CopyTradingService) StartCopying(ctx context.Context, copier *Copier) (*Copier, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify trader exists
	trader, exists := s.traders[copier.TraderID]
	if !exists {
		return nil, fmt.Errorf("trader not found")
	}

	if trader.Status != TraderStatusActive {
		return nil, fmt.Errorf("trader not active")
	}

	copier.ID = "copier_" + uuid.New().String()
	copier.Status = "active"
	copier.TotalPnl = "0"
	copier.TotalTrades = 0
	copier.CreatedAt = time.Now().Unix()
	copier.UpdatedAt = time.Now().Unix()

	s.copiers[copier.ID] = copier

	// Update trader copiers count
	trader.ActiveCopiers++
	trader.TotalCopiers++

	return copier, nil
}

// StopCopying stops copying a trader
func (s *CopyTradingService) StopCopying(ctx context.Context, copierID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copier, exists := s.copiers[copierID]
	if !exists {
		return fmt.Errorf("copier not found")
	}

	trader, exists := s.traders[copier.TraderID]
	if exists {
		trader.ActiveCopiers--
	}

	copier.Status = "stopped"
	copier.UpdatedAt = time.Now().Unix()

	return nil
}

// GetCopier returns a copier
func (s *CopyTradingService) GetCopier(ctx context.Context, copierID string) (*Copier, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copier, exists := s.copiers[copierID]
	if !exists {
		return nil, fmt.Errorf("copier not found")
	}
	return copier, nil
}

// GetCopiersByUser returns all copiers for a user
func (s *CopyTradingService) GetCopiersByUser(ctx context.Context, userID string) ([]*Copier, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Copier, 0)
	for _, copier := range s.copiers {
		if copier.UserID == userID {
			result = append(result, copier)
		}
	}
	return result, nil
}

// GetCopiersByTrader returns all copiers for a trader
func (s *CopyTradingService) GetCopiersByTrader(ctx context.Context, traderID string) ([]*Copier, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Copier, 0)
	for _, copier := range s.copiers {
		if copier.TraderID == traderID {
			result = append(result, copier)
		}
	}
	return result, nil
}

// UpdateCopierSettings updates copier settings
func (s *CopyTradingService) UpdateCopierSettings(ctx context.Context, copierID, copyRatio, allocation string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copier, exists := s.copiers[copierID]
	if !exists {
		return fmt.Errorf("copier not found")
	}

	copier.CopyRatio = copyRatio
	copier.Allocation = allocation
	copier.UpdatedAt = time.Now().Unix()

	return nil
}

// ============================================================================
// Signal Operations
// ============================================================================

// EmitSignal emits a trading signal
func (s *CopyTradingService) EmitSignal(ctx context.Context, signal *Signal) (*Signal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify trader exists
	trader, exists := s.traders[signal.TraderID]
	if !exists {
		return nil, fmt.Errorf("trader not found")
	}

	if trader.Status != TraderStatusActive {
		return nil, fmt.Errorf("trader not active")
	}

	signal.ID = "signal_" + uuid.New().String()
	signal.Timestamp = time.Now().Unix()
	signal.Executed = false

	s.signals[signal.ID] = signal

	// Execute signal for all active copiers
	go s.executeSignalForCopiers(signal)

	return signal, nil
}

// executeSignalForCopiers executes a signal for all copiers
func (s *CopyTradingService) executeSignalForCopiers(signal *Signal) {
	s.mu.RLock()
	var activeCopiers []*Copier
	for _, copier := range s.copiers {
		if copier.TraderID == signal.TraderID && copier.Status == "active" {
			activeCopiers = append(activeCopiers, copier)
		}
	}
	s.mu.RUnlock()

	for _, copier := range activeCopiers {
		trade := &CopiedTrade{
			ID:          "trade_" + uuid.New().String(),
			CopierID:   copier.ID,
			TraderID:   signal.TraderID,
			SignalID:   signal.ID,
			Pair:       signal.Pair,
			Side:       signal.Side,
			EntryPrice: signal.EntryPrice,
			Quantity:   signal.Quantity,
			Status:     "open",
			OpenedAt:   time.Now().Unix(),
		}

		// Calculate quantity based on copy ratio
		originalQty, _ := new(big.Int).SetString(signal.Quantity, 10)
		copyRatio, _ := new(big.Float).SetString(copier.CopyRatio)
		ratio, _ := copyRatio.Float64()
		adjustedQty := new(big.Int).Mul(originalQty, big.NewInt(int64(ratio*100)))
		adjustedQty.Div(adjustedQty, big.NewInt(100))
		trade.Quantity = adjustedQty.String()

		s.mu.Lock()
		s.trades[trade.ID] = trade
		s.mu.Unlock()
	}
}

// GetSignal returns a signal
func (s *CopyTradingService) GetSignal(ctx context.Context, signalID string) (*Signal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	signal, exists := s.signals[signalID]
	if !exists {
		return nil, fmt.Errorf("signal not found")
	}
	return signal, nil
}

// GetSignalsByTrader returns signals for a trader
func (s *CopyTradingService) GetSignalsByTrader(ctx context.Context, traderID string, limit int) ([]*Signal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Signal, 0)
	for _, signal := range s.signals {
		if signal.TraderID == traderID {
			result = append(result, signal)
		}
	}

	// Sort by timestamp descending
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Timestamp > result[i].Timestamp {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// GetLatestSignals returns latest signals
func (s *CopyTradingService) GetLatestSignals(ctx context.Context, limit int) ([]*Signal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Signal, 0)
	for _, signal := range s.signals {
		result = append(result, signal)
	}

	// Sort by timestamp descending
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Timestamp > result[i].Timestamp {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// ============================================================================
// Trade Operations
// ============================================================================

// CloseTrade closes a copied trade
func (s *CopyTradingService) CloseTrade(ctx context.Context, tradeID, exitPrice, pnl, fee string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	trade, exists := s.trades[tradeID]
	if !exists {
		return fmt.Errorf("trade not found")
	}

	if trade.Status == "closed" {
		return fmt.Errorf("trade already closed")
	}

	trade.ExitPrice = exitPrice
	trade.Pnl = pnl
	trade.Fee = fee
	trade.Status = "closed"
	trade.ClosedAt = time.Now().Unix()

	// Update copier stats
	copier, exists := s.copiers[trade.CopierID]
	if exists {
		copier.TotalTrades++
		copier.LastTradeTime = time.Now().Unix()

		// Update total PnL
		copierPnl, _ := new(big.Int).SetString(copier.TotalPnl, 10)
		tradePnl, _ := new(big.Int).SetString(pnl, 10)
		copierPnl.Add(copierPnl, tradePnl)
		copier.TotalPnl = copierPnl.String()
	}

	// Update trader stats
	trader, exists := s.traders[trade.TraderID]
	if exists {
		trader.TotalTrades++
		traderPnl, _ := new(big.Int).SetString(trader.TotalPnl, 10)
		tradePnl, _ := new(big.Int).SetString(pnl, 10)
		traderPnl.Add(traderPnl, tradePnl)
		trader.TotalPnl = traderPnl.String()
	}

	return nil
}

// GetTrade returns a trade
func (s *CopyTradingService) GetTrade(ctx context.Context, tradeID string) (*CopiedTrade, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trade, exists := s.trades[tradeID]
	if !exists {
		return nil, fmt.Errorf("trade not found")
	}
	return trade, nil
}

// GetOpenTrades returns open trades for a copier
func (s *CopyTradingService) GetOpenTrades(ctx context.Context, copierID string) ([]*CopiedTrade, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*CopiedTrade, 0)
	for _, trade := range s.trades {
		if trade.CopierID == copierID && trade.Status == "open" {
			result = append(result, trade)
		}
	}
	return result, nil
}

// GetClosedTrades returns closed trades for a copier
func (s *CopyTradingService) GetClosedTrades(ctx context.Context, copierID string) ([]*CopiedTrade, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*CopiedTrade, 0)
	for _, trade := range s.trades {
		if trade.CopierID == copierID && trade.Status == "closed" {
			result = append(result, trade)
		}
	}
	return result, nil
}

// GetTradesByTrader returns trades by trader
func (s *CopyTradingService) GetTradesByTrader(ctx context.Context, traderID string, limit int) ([]*CopiedTrade, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*CopiedTrade, 0)
	for _, trade := range s.trades {
		if trade.TraderID == traderID {
			result = append(result, trade)
		}
	}

	// Sort by timestamp descending
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].OpenedAt > result[i].OpenedAt {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// ============================================================================
// Utility Methods
// ============================================================================

// GetCopierStats returns copier statistics
func (s *CopyTradingService) GetCopierStats(ctx context.Context, copierID string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copier, exists := s.copiers[copierID]
	if !exists {
		return nil, fmt.Errorf("copier not found")
	}

	// Calculate open positions
	openTrades := 0
	closedTrades := 0
	totalPnl := big.NewInt(0)

	for _, trade := range s.trades {
		if trade.CopierID == copierID {
			if trade.Status == "open" {
				openTrades++
			} else {
				closedTrades++
				pnl, _ := new(big.Int).SetString(trade.Pnl, 10)
				totalPnl.Add(totalPnl, pnl)
			}
		}
	}

	stats := map[string]interface{}{
		"total_pnl":      totalPnl.String(),
		"total_trades":   copier.TotalTrades,
		"open_trades":    openTrades,
		"closed_trades":  closedTrades,
		"copy_ratio":     copier.CopyRatio,
		"allocation":     copier.Allocation,
		"last_trade_time": copier.LastTradeTime,
	}

	return stats, nil
}

// ToJSON converts trader to JSON
func (t *Trader) ToJSON() (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
