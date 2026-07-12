package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"tigerwallet/backend/go/api/models"
)

type CopyTradingService struct {
	mu       sync.RWMutex
	traders  map[string]*models.CopyTrader
	trades   map[string]*models.CopyTrade
	followers map[string]*models.Follower
}

var (
	copyInstance *CopyTradingService
	copyOnce   sync.Once
)

func NewCopyTradingService() *CopyTradingService {
	copyOnce.Do(func() {
		copyInstance = &CopyTradingService{
			traders:  make(map[string]*models.CopyTrader),
			trades:   make(map[string]*models.CopyTrade),
			followers: make(map[string]*models.Follower),
		}
		copyInstance.initializeDefaultTraders()
	})
	return copyInstance
}

func (s *CopyTradingService) initializeDefaultTraders() {
	defaultTraders := []*models.CopyTrader{
		{ID: "trader_1", Address: "0x1234567890abcdef1234567890abcdef12345678", Name: "CryptoMaster", Avatar: "", TotalTrades: 1250, WinRate: 72.5, ProfitFactor: 2.8, AUM: "5000000", FollowersCount: 15420, Performance: struct {
			Daily   float64 `json:"daily"`
			Weekly  float64 `json:"weekly"`
			Monthly float64 `json:"monthly"`
			AllTime float64 `json:"all_time"`
		}{Daily: 2.5, Weekly: 8.2, Monthly: 35.5, AllTime: 245.8}, IsVerified: true},
		{ID: "trader_2", Address: "0xabcdef1234567890abcdef1234567890abcdef12", Name: "DeFiPro", Avatar: "", TotalTrades: 890, WinRate: 68.2, ProfitFactor: 2.3, AUM: "3200000", FollowersCount: 8930, Performance: struct {
			Daily   float64 `json:"daily"`
			Weekly  float64 `json:"weekly"`
			Monthly float64 `json:"monthly"`
			AllTime float64 `json:"all_time"`
		}{Daily: 1.8, Weekly: 6.5, Monthly: 28.2, AllTime: 185.3}, IsVerified: true},
		{ID: "trader_3", Address: "0x9876543210fedcba9876543210fedcba98765432", Name: "AltcoinKing", Avatar: "", TotalTrades: 2100, WinRate: 58.5, ProfitFactor: 1.9, AUM: "2800000", FollowersCount: 12350, Performance: struct {
			Daily   float64 `json:"daily"`
			Weekly  float64 `json:"weekly"`
			Monthly float64 `json:"monthly"`
			AllTime float64 `json:"all_time"`
		}{Daily: 3.2, Weekly: 12.5, Monthly: 45.8, AllTime: 320.5}, IsVerified: true},
		{ID: "trader_4", Address: "0xfedcba9876543210fedcba9876543210fedcba98", Name: "YieldHunter", Avatar: "", TotalTrades: 560, WinRate: 75.8, ProfitFactor: 3.2, AUM: "4500000", FollowersCount: 9870, Performance: struct {
			Daily   float64 `json:"daily"`
			Weekly  float64 `json:"weekly"`
			Monthly float64 `json:"monthly"`
			AllTime float64 `json:"all_time"`
		}{Daily: 1.5, Weekly: 5.8, Monthly: 22.5, AllTime: 165.2}, IsVerified: true},
		{ID: "trader_5", Address: "0x5678901234abcdef5678901234abcdef56789012", Name: "MomentumTrader", Avatar: "", TotalTrades: 1850, WinRate: 62.5, ProfitFactor: 2.1, AUM: "3800000", FollowersCount: 7650, Performance: struct {
			Daily   float64 `json:"daily"`
			Weekly  float64 `json:"weekly"`
			Monthly float64 `json:"monthly"`
			AllTime float64 `json:"all_time"`
		}{Daily: 2.8, Weekly: 9.5, Monthly: 38.2, AllTime: 275.8}, IsVerified: true},
		{ID: "trader_6", Address: "0xdef1234567890abcdef1234567890abcdef123456", Name: "StableYield", Avatar: "", TotalTrades: 420, WinRate: 82.5, ProfitFactor: 4.5, AUM: "6500000", FollowersCount: 15680, Performance: struct {
			Daily   float64 `json:"daily"`
			Weekly  float64 `json:"weekly"`
			Monthly float64 `json:"monthly"`
			AllTime float64 `json:"all_time"`
		}{Daily: 0.8, Weekly: 3.2, Monthly: 12.5, AllTime: 95.2}, IsVerified: true},
		{ID: "trader_7", Address: "0xabc9876543210fedcba9876543210fedcba98765", Name: "SwingKing", Avatar: "", TotalTrades: 980, WinRate: 65.8, ProfitFactor: 2.5, AUM: "4200000", FollowersCount: 11200, Performance: struct {
			Daily   float64 `json:"daily"`
			Weekly  float64 `json:"weekly"`
			Monthly float64 `json:"monthly"`
			AllTime float64 `json:"all_time"`
		}{Daily: 2.2, Weekly: 7.8, Monthly: 32.5, AllTime: 225.8}, IsVerified: true},
		{ID: "trader_8", Address: "0x123abc456def789ghi012jkl345mno678pqr901st", Name: "GridBot", Avatar: "", TotalTrades: 3200, WinRate: 78.5, ProfitFactor: 2.9, AUM: "7200000", FollowersCount: 18900, Performance: struct {
			Daily   float64 `json:"daily"`
			Weekly  float64 `json:"weekly"`
			Monthly float64 `json:"monthly"`
			AllTime float64 `json:"all_time"`
		}{Daily: 1.2, Weekly: 4.5, Monthly: 18.2, AllTime: 145.8}, IsVerified: true},
		{ID: "trader_9", Address: "0x789xyz012abc345def678ghi901jkl234mno567qr", Name: "ArbitragePro", Avatar: "", TotalTrades: 750, WinRate: 85.2, ProfitFactor: 5.2, AUM: "8900000", FollowersCount: 22100, Performance: struct {
			Daily   float64 `json:"daily"`
			Weekly  float64 `json:"weekly"`
			Monthly float64 `json:"monthly"`
			AllTime float64 `json:"all_time"`
		}{Daily: 0.5, Weekly: 2.2, Monthly: 8.5, AllTime: 65.2}, IsVerified: true},
		{ID: "trader_10", Address: "0x456ghi789jkl012mno345pqr678stu901vwx234yza", Name: "TrendFollower", Avatar: "", TotalTrades: 1680, WinRate: 60.2, ProfitFactor: 1.85, AUM: "2100000", FollowersCount: 6540, Performance: struct {
			Daily   float64 `json:"daily"`
			Weekly  float64 `json:"weekly"`
			Monthly float64 `json:"monthly"`
			AllTime float64 `json:"all_time"`
		}{Daily: 3.5, Weekly: 15.2, Monthly: 52.8, AllTime: 385.5}, IsVerified: false},
	}

	for _, trader := range defaultTraders {
		s.traders[trader.ID] = trader
	}
}

func (s *CopyTradingService) GetTraders(ctx context.Context, sortBy string) ([]*models.CopyTrader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.CopyTrader
	for _, trader := range s.traders {
		result = append(result, trader)
	}

	return result, nil
}

func (s *CopyTradingService) GetTraderByID(ctx context.Context, traderID string) (*models.CopyTrader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trader, ok := s.traders[traderID]
	if !ok {
		return nil, errors.New("trader not found")
	}

	return trader, nil
}

func (s *CopyTradingService) FollowTrader(ctx context.Context, followerID, traderID string, allocation string, maxSlippage float64) (*models.Follower, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify trader exists
	_, ok := s.traders[traderID]
	if !ok {
		return nil, errors.New("trader not found")
	}

	follower := &models.Follower{
		ID:          fmt.Sprintf("follower_%d", time.Now().UnixNano()),
		FollowerID:  followerID,
		TraderID:    traderID,
		Allocation:  allocation,
		MaxSlippage: maxSlippage,
		Status:      "active",
		CreatedAt:   time.Now(),
	}

	s.followers[follower.ID] = follower

	// Update follower count
	s.traders[traderID].FollowersCount++

	return follower, nil
}

func (s *CopyTradingService) UnfollowTrader(ctx context.Context, followerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	follower, ok := s.followers[followerID]
	if !ok {
		return errors.New("follow not found")
	}

	traderID := follower.TraderID
	follower.Status = "cancelled"
	follower.UpdatedAt = time.Now()

	// Update follower count
	if trader, ok := s.traders[traderID]; ok {
		trader.FollowersCount--
	}

	return nil
}

func (s *CopyTradingService) GetFollowers(ctx context.Context, traderID string) ([]*models.Follower, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Follower
	for _, follower := range s.followers {
		if follower.TraderID == traderID && follower.Status == "active" {
			result = append(result, follower)
		}
	}

	return result, nil
}

func (s *CopyTradingService) GetCopyTrades(ctx context.Context, followerID string) ([]*models.CopyTrade, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.CopyTrade
	for _, trade := range s.trades {
		if trade.FollowerID == followerID {
			result = append(result, trade)
		}
	}

	return result, nil
}

func (s *CopyTradingService) RecordCopyTrade(ctx context.Context, followerID, traderID, symbol, side, size, entryPrice string, pnl float64) (*models.CopyTrade, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	trade := &models.CopyTrade{
		ID:         fmt.Sprintf("copytrade_%d", time.Now().UnixNano()),
		FollowerID: followerID,
		TraderID:   traderID,
		Symbol:     symbol,
		Side:       side,
		Size:       size,
		EntryPrice: entryPrice,
		PnL:        fmt.Sprintf("%f", pnl),
		PnLPercent: pnl,
		Status:     "closed",
		OpenedAt:   time.Now().Add(-time.Hour),
		ClosedAt:   time.Now(),
	}

	s.trades[trade.ID] = trade

	return trade, nil
}
