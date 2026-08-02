package copy_trading_service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

type Trader struct {
	ID               string   `json:"id"`
	Username         string   `json:"username"`
	Avatar           string   `json:"avatar"`
	WinRate          float64  `json:"winRate"`
	TotalPnL         float64  `json:"totalPnL"`
	PnLPercent       float64  `json:"pnlPercent"`
	Followers        int      `json:"followers"`
	CopyCount        int      `json:"copyCount"`
	TradingPair      string   `json:"tradingPair"`
	MonthlyPnL       float64  `json:"monthlyPnL"`
	WeeklyPnL        float64  `json:"weeklyPnL"`
	DailyPnL         float64  `json:"dailyPnL"`
	MaxDrawdown      float64  `json:"maxDrawdown"`
	AvgHoldingTime   string   `json:"avgHoldingTime"`
	RiskLevel        string   `json:"riskLevel"`
	IsFollowing      bool     `json:"isFollowing"`
	IsPreInstalled   bool     `json:"isPreInstalled"`
	TotalTrades     int      `json:"totalTrades"`
	ProfitableTrades int      `json:"profitableTrades"`
}

type CopyPosition struct {
	ID           string    `json:"id"`
	TraderID     string    `json:"traderId"`
	TraderName   string    `json:"traderName"`
	UserID       string    `json:"userId"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"`
	Size         float64   `json:"size"`
	EntryPrice   float64   `json:"entryPrice"`
	CurrentPrice float64   `json:"currentPrice"`
	PnL          float64   `json:"pnl"`
	PnLPercent   float64   `json:"pnlPercent"`
	OpenTime     time.Time `json:"openTime"`
	Status       string    `json:"status"`
}

type CopySettings struct {
	UserID          string  `json:"userId"`
	CopyAmount      float64 `json:"copyAmount"`
	CopyLeverage    int     `json:"copyLeverage"`
	StopLossPercent float64 `json:"stopLossPercent"`
	TakeProfitPercent float64 `json:"takeProfitPercent"`
	AutoCopy        bool    `json:"autoCopy"`
}

type CopyTrade struct {
	ID          string    `json:"id"`
	TraderID    string    `json:"traderId"`
	UserID      string    `json:"userId"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	Size        float64   `json:"size"`
	Price       float64   `json:"price"`
	Status      string    `json:"status"`
	CreateTime  time.Time `json:"createTime"`
}

// ============================================================================
// Service
// ============================================================================

type CopyTradingService struct {
	mu           sync.RWMutex
	traders      map[string]*Trader
	positions    map[string][]CopyPosition
	settings     map[string]*CopySettings
	trades       map[string][]CopyTrade
	followers    map[string][]string // traderID -> userIDs
}

func NewCopyTradingService() *CopyTradingService {
	cts := &CopyTradingService{
		traders:   make(map[string]*Trader),
		positions: make(map[string][]CopyPosition),
		settings:  make(map[string]*CopySettings),
		trades:    make(map[string][]CopyTrade),
		followers: make(map[string][]string),
	}
	cts.initializeDefaultTraders()
	return cts
}

func (cts *CopyTradingService) initializeDefaultTraders() {
	// Pre-installed top traders
	preInstalledTraders := []*Trader{
		{ID: "1", Username: "CryptoWhale", Avatar: "🐋", WinRate: 78.5, TotalPnL: 125000, PnLPercent: 156.2, Followers: 15234, CopyCount: 4521, TradingPair: "BTC/USDT", MonthlyPnL: 12.5, WeeklyPnL: 3.2, DailyPnL: 0.8, MaxDrawdown: -8.5, AvgHoldingTime: "2h 30m", RiskLevel: "medium", IsPreInstalled: true},
		{ID: "2", Username: "DeFiMaster", Avatar: "🎯", WinRate: 82.3, TotalPnL: 98500, PnLPercent: 142.8, Followers: 12456, CopyCount: 3890, TradingPair: "ETH/USDT", MonthlyPnL: 15.2, WeeklyPnL: 4.1, DailyPnL: 1.2, MaxDrawdown: -6.2, AvgHoldingTime: "4h 15m", RiskLevel: "low", IsPreInstalled: true},
		{ID: "3", Username: "AltSeason", Avatar: "🚀", WinRate: 71.2, TotalPnL: 87000, PnLPercent: 198.5, Followers: 8923, CopyCount: 2156, TradingPair: "SOL/USDT", MonthlyPnL: 22.5, WeeklyPnL: 8.3, DailyPnL: 2.1, MaxDrawdown: -12.8, AvgHoldingTime: "1h 45m", RiskLevel: "high", IsPreInstalled: true},
		{ID: "4", Username: "GridTrader", Avatar: "📊", WinRate: 85.1, TotalPnL: 67800, PnLPercent: 98.3, Followers: 6543, CopyCount: 1890, TradingPair: "BNB/USDT", MonthlyPnL: 8.2, WeeklyPnL: 2.1, DailyPnL: 0.5, MaxDrawdown: -4.2, AvgHoldingTime: "6h 20m", RiskLevel: "low", IsPreInstalled: true},
		{ID: "5", Username: "MomentumKing", Avatar: "👑", WinRate: 75.8, TotalPnL: 54200, PnLPercent: 125.6, Followers: 9876, CopyCount: 2567, TradingPair: "DOGE/USDT", MonthlyPnL: 18.5, WeeklyPnL: 5.2, DailyPnL: 1.5, MaxDrawdown: -15.2, AvgHoldingTime: "0h 45m", RiskLevel: "high", IsPreInstalled: true},
		{ID: "6", Username: "SwingTrader", Avatar: "🌊", WinRate: 68.5, TotalPnL: 42500, PnLPercent: 88.2, Followers: 5432, CopyCount: 1234, TradingPair: "XRP/USDT", MonthlyPnL: 10.2, WeeklyPnL: 2.8, DailyPnL: 0.3, MaxDrawdown: -9.5, AvgHoldingTime: "12h 30m", RiskLevel: "medium", IsPreInstalled: true},
		{ID: "7", Username: "BotMaster", Avatar: "🤖", WinRate: 88.2, TotalPnL: 38900, PnLPercent: 72.5, Followers: 4321, CopyCount: 987, TradingPair: "AVAX/USDT", MonthlyPnL: 6.8, WeeklyPnL: 1.5, DailyPnL: 0.2, MaxDrawdown: -3.2, AvgHoldingTime: "8h 00m", RiskLevel: "low", IsPreInstalled: true},
		{ID: "8", Username: "NanoGainer", Avatar: "💎", WinRate: 73.2, TotalPnL: 31500, PnLPercent: 145.8, Followers: 7654, CopyCount: 1876, TradingPair: "PEPE/USDT", MonthlyPnL: 25.2, WeeklyPnL: 9.5, DailyPnL: 3.2, MaxDrawdown: -18.5, AvgHoldingTime: "0h 30m", RiskLevel: "high", IsPreInstalled: true},
		{ID: "9", Username: "StableTrader", Avatar: "🛡️", WinRate: 91.2, TotalPnL: 28900, PnLPercent: 52.3, Followers: 3210, CopyCount: 654, TradingPair: "LINK/USDT", MonthlyPnL: 4.2, WeeklyPnL: 1.0, DailyPnL: 0.1, MaxDrawdown: -2.1, AvgHoldingTime: "24h 00m", RiskLevel: "low", IsPreInstalled: true},
		{ID: "10", Username: "FlashBoys", Avatar: "⚡", WinRate: 76.8, TotalPnL: 24500, PnLPercent: 168.5, Followers: 5678, CopyCount: 1432, TradingPair: "MATIC/USDT", MonthlyPnL: 14.5, WeeklyPnL: 4.8, DailyPnL: 1.8, MaxDrawdown: -11.2, AvgHoldingTime: "1h 15m", RiskLevel: "high", IsPreInstalled: true},
	}

	for _, trader := range preInstalledTraders {
		cts.traders[trader.ID] = trader
	}

	// Generate additional traders
	avatars := []string{"🐵", "🦊", "🦁", "🐯", "🐲", "🐍", "🐴", "🦄", "🐝", "🦋", "🌸", "🌺", "🌻", "🌹", "🍀"}
	pairs := []string{"BTC/USDT", "ETH/USDT", "BNB/USDT", "SOL/USDT", "XRP/USDT", "DOGE/USDT", "ADA/USDT", "AVAX/USDT", "DOT/USDT", "LINK/USDT"}
	riskLevels := []string{"low", "medium", "high"}

	for i := 0; i < 500; i++ {
		trader := &Trader{
			ID:             fmt.Sprintf("trader-%d", i+100),
			Username:       fmt.Sprintf("Trader%d", i+100),
			Avatar:         avatars[i%len(avatars)],
			WinRate:        60 + float64(i%30),
			TotalPnL:       1000 + float64(i*200),
			PnLPercent:     20 + float64(i%200),
			Followers:      100 + i*20,
			CopyCount:      50 + i*10,
			TradingPair:    pairs[i%len(pairs)],
			MonthlyPnL:     float64(i%30) - 5,
			WeeklyPnL:      float64(i%10) - 2,
			DailyPnL:       float64(i%3) - 1,
			MaxDrawdown:    -float64(2 + i%20),
			AvgHoldingTime: fmt.Sprintf("%dh %dm", i%24, i%60),
			RiskLevel:      riskLevels[i%3],
			IsFollowing:    false,
			IsPreInstalled: false,
		}
		cts.traders[trader.ID] = trader
	}
}

func (cts *CopyTradingService) GetAllTraders() []*Trader {
	cts.mu.RLock()
	defer cts.mu.RUnlock()

	traders := make([]*Trader, 0, len(cts.traders))
	for _, trader := range cts.traders {
		traders = append(traders, trader)
	}
	return traders
}

func (cts *CopyTradingService) GetTrader(traderID string) (*Trader, error) {
	cts.mu.RLock()
	defer cts.mu.RUnlock()

	trader, ok := cts.traders[traderID]
	if !ok {
		return nil, fmt.Errorf("trader not found: %s", traderID)
	}
	return trader, nil
}

func (cts *CopyTradingService) FollowTrader(userID, traderID string) error {
	cts.mu.Lock()
	defer cts.mu.Unlock()

	trader, ok := cts.traders[traderID]
	if !ok {
		return fmt.Errorf("trader not found: %s", traderID)
	}

	trader.IsFollowing = !trader.IsFollowing
	if trader.IsFollowing {
		trader.Followers++
		cts.followers[traderID] = append(cts.followers[traderID], userID)
	} else {
		trader.Followers--
		// Remove user from followers
		for i, id := range cts.followers[traderID] {
			if id == userID {
				cts.followers[traderID] = append(cts.followers[traderID][:i], cts.followers[traderID][i+1:]...)
				break
			}
		}
	}
	return nil
}

func (cts *CopyTradingService) GetUserFollowing(userID string) []*Trader {
	cts.mu.RLock()
	defer cts.mu.RUnlock()

	var following []*Trader
	for _, trader := range cts.traders {
		if trader.IsFollowing {
			following = append(following, trader)
		}
	}
	return following
}

func (cts *CopyTradingService) GetUserPositions(userID string) []CopyPosition {
	cts.mu.RLock()
	defer cts.mu.RUnlock()
	return cts.positions[userID]
}

func (cts *CopyTradingService) GetUserSettings(userID string) *CopySettings {
	cts.mu.RLock()
	defer cts.mu.RUnlock()

	if settings, ok := cts.settings[userID]; ok {
		return settings
	}
	return &CopySettings{
		UserID:          userID,
		CopyAmount:      1000,
		CopyLeverage:    1,
		StopLossPercent: 10,
		TakeProfitPercent: 20,
		AutoCopy:        false,
	}
}

func (cts *CopyTradingService) UpdateUserSettings(settings *CopySettings) error {
	cts.mu.Lock()
	defer cts.mu.Unlock()
	cts.settings[settings.UserID] = settings
	return nil
}

func (cts *CopyTradingService) CopyTrade(traderID, userID, symbol, side string, size float64) (*CopyPosition, error) {
	cts.mu.Lock()
	defer cts.mu.Unlock()

	trader, ok := cts.traders[traderID]
	if !ok {
		return nil, fmt.Errorf("trader not found: %s", traderID)
	}

	settings := cts.GetUserSettings(userID)
	price := 100.0 // Simulated price

	position := &CopyPosition{
		ID:           fmt.Sprintf("copy-pos-%d", time.Now().Unix()),
		TraderID:     traderID,
		TraderName:   trader.Username,
		UserID:       userID,
		Symbol:       symbol,
		Side:         side,
		Size:         size,
		EntryPrice:   price,
		CurrentPrice: price,
		PnL:          0,
		PnLPercent:   0,
		OpenTime:     time.Now(),
		Status:       "open",
	}

	cts.positions[userID] = append(cts.positions[userID], *position)
	return position, nil
}

func (cts *CopyTradingService) ToJSON() (string, error) {
	cts.mu.RLock()
	defer cts.mu.RUnlock()

	data := struct {
		Traders   map[string]*Trader   `json:"traders"`
		Positions map[string][]CopyPosition `json:"positions"`
		Settings  map[string]*CopySettings `json:"settings"`
	}{
		Traders:   cts.traders,
		Positions: cts.positions,
		Settings:  cts.settings,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
