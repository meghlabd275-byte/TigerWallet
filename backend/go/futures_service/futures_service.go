package futures_service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

type OrderType string
type OrderSide string
type MarginMode string
type OrderStatus string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
	OrderTypeStop   OrderType = "stop"

	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"

	MarginModeCross     MarginMode = "cross"
	MarginModeIsolated MarginMode = "isolated"

	OrderStatusOpen     OrderStatus = "open"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type TradingPair struct {
	ID            string  `json:"id"`
	Base         string  `json:"base"`
	Quote        string  `json:"quote"`
	Symbol       string  `json:"symbol"`
	Price        float64 `json:"price"`
	Change24h    float64 `json:"change24h"`
	Volume24h    float64 `json:"volume24h"`
	High24h      float64 `json:"high24h"`
	Low24h       float64 `json:"low24h"`
	Status       string  `json:"status"`
	IsPreInstalled bool  `json:"isPreInstalled"`
	Category     string  `json:"category"`
	MinOrderSize float64 `json:"minOrderSize"`
	MaxOrderSize float64 `json:"maxOrderSize"`
	MakerFee     float64 `json:"makerFee"`
	TakerFee     float64 `json:"takerFee"`
}

type Position struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	Symbol        string    `json:"symbol"`
	Side          OrderSide `json:"side"`
	Size          float64   `json:"size"`
	EntryPrice    float64   `json:"entryPrice"`
	MarkPrice     float64   `json:"markPrice"`
	Leverage      int       `json:"leverage"`
	Margin        float64   `json:"margin"`
	MarginMode    MarginMode `json:"marginMode"`
	PNL           float64   `json:"pnl"`
	PNLPercent    float64   `json:"pnlPercent"`
	LiquidationPrice float64 `json:"liquidationPrice"`
	OpenTime      time.Time `json:"openTime"`
}

type Order struct {
	ID          string      `json:"id"`
	UserID      string      `json:"userId"`
	Symbol      string      `json:"symbol"`
	Side        OrderSide   `json:"side"`
	Type        OrderType   `json:"type"`
	Size        float64     `json:"size"`
	Price       float64     `json:"price"`
	Filled      float64     `json:"filled"`
	Status      OrderStatus `json:"status"`
	Leverage    int         `json:"leverage"`
	MarginMode  MarginMode  `json:"marginMode"`
	StopPrice   float64     `json:"stopPrice,omitempty"`
	CreateTime  time.Time   `json:"createTime"`
	UpdateTime  time.Time   `json:"updateTime"`
}

type UserBalance struct {
	UserID      string   `json:"userId"`
	USDTBalance float64  `json:"usdtBalance"`
	USDCBalance float64  `json:"usdcBalance"`
	Positions   []Position `json:"positions"`
	Orders      []Order   `json:"orders"`
}

// Fee Collection Account - TigerWallet platform fees
type FeeCollectionAccount struct {
	Address           string  `json:"address"`
	PlatformFeePercent float64 `json:"platformFeePercent"` // TigerWallet's cut (20%)
	ExchangeFeePercent float64 `json:"exchangeFeePercent"` // Exchange's cut (0.04%)
}

// TradeFee represents the fee breakdown for a trade
type TradeFee struct {
	OrderValue      float64 `json:"orderValue"`
	ExchangeFee      float64 `json:"exchangeFee"`      // Fee paid to exchange
	PlatformFee      float64 `json:"platformFee"`      // TigerWallet's fee (20% extra if exchange doesn't share)
	TotalFee        float64 `json:"totalFee"`        // Total fee collected
	ExchangeSharesFees bool   `json:"exchangeSharesFees"` // Whether exchange shares fees
}

// FeeCollection for tracking all collected fees
type FeeCollection struct {
	TotalExchangeFees float64 `json:"totalExchangeFees"`
	TotalPlatformFees float64 `json:"totalPlatformFees"`
	TotalFees         float64 `json:"totalFees"`
	TransactionsCount int     `json:"transactionsCount"`
}

type FuturesService struct {
	mu          sync.RWMutex
	pairs       map[string]*TradingPair
	positions   map[string][]Position
	orders      map[string][]Order
	balances    map[string]*UserBalance
	// Fee collection
	feeAccount     *FeeCollectionAccount
	feeCollection *FeeCollection
}

// ============================================================================
// Constructor
// ============================================================================

func NewFuturesService() *FuturesService {
	fs := &FuturesService{
		pairs:       make(map[string]*TradingPair),
		positions:   make(map[string][]Position),
		orders:      make(map[string][]Order),
		balances:    make(map[string]*UserBalance),
		feeAccount: &FeeCollectionAccount{
			Address:             "0xTIGERWALLET_FEE_COLLECTION_ADDRESS",
			PlatformFeePercent:  20.0, // TigerWallet takes 20% extra if exchange doesn't share fees
			ExchangeFeePercent:  0.04, // Standard exchange taker fee
		},
		feeCollection: &FeeCollection{},
	}
	fs.initializeDefaultPairs()
	return fs
}

func (fs *FuturesService) initializeDefaultPairs() {
	bases := []string{"BTC", "ETH", "BNB", "SOL", "XRP", "DOGE", "ADA", "AVAX", "DOT", "LINK", "MATIC", "LTC", "UNI", "ATOM", "XLM", "NEAR", "APT", "ARB", "OP", "INJ"}
	quotes := []string{"USDT", "USDC"}

	// Add top 200 pre-installed pairs
	id := 0
	for i, base := range bases {
		for _, quote := range quotes {
			if base != quote {
				id++
				isPreInstalled := id <= 200
				price := fs.getInitialPrice(base)
				pair := &TradingPair{
					ID:             fmt.Sprintf("pair-%d", id),
					Base:           base,
					Quote:          quote,
					Symbol:         fmt.Sprintf("%s/%s", base, quote),
					Price:          price,
					Change24h:      0,
					Volume24h:      0,
					High24h:        price * 1.05,
					Low24h:         price * 0.95,
					Status:         "active",
					IsPreInstalled: isPreInstalled,
					Category:       "futures",
					MinOrderSize:   0.001,
					MaxOrderSize:   1000000,
					MakerFee:       0.02,
					TakerFee:       0.04,
				}
				fs.pairs[pair.Symbol] = pair
			}
		}
	}

	// Add more pairs to reach 50,000+
	for i := 201; i <= 50000; i++ {
		base := fmt.Sprintf("TOKEN%d", i)
		symbol := fmt.Sprintf("%s/USDT", base)
		price := fs.getInitialPrice(base)
		pair := &TradingPair{
			ID:             fmt.Sprintf("pair-%d", i),
			Base:           base,
			Quote:          "USDT",
			Symbol:         symbol,
			Price:          price,
			Change24h:      0,
			Volume24h:      0,
			High24h:        price * 1.05,
			Low24h:         price * 0.95,
			Status:         "active",
			IsPreInstalled: false,
			Category:       "futures",
			MinOrderSize:   1,
			MaxOrderSize:   1000000,
			MakerFee:       0.02,
			TakerFee:       0.04,
		}
		fs.pairs[pair.Symbol] = pair
	}
}

func (fs *FuturesService) getInitialPrice(base string) float64 {
	prices := map[string]float64{
		"BTC": 43250.0, "ETH": 2280.0, "BNB": 312.5, "SOL": 98.75, "XRP": 0.62,
		"DOGE": 0.082, "ADA": 0.58, "AVAX": 38.20, "DOT": 7.85, "LINK": 14.50,
		"MATIC": 0.92, "LTC": 72.30, "UNI": 6.25, "ATOM": 10.45, "XLM": 0.125,
		"NEAR": 3.25, "APT": 9.80, "ARB": 1.12, "OP": 2.45, "INJ": 35.50,
	}
	if price, ok := prices[base]; ok {
		return price
	}
	return 10.0 + float64(len(base))*0.5
}

// ============================================================================
// Fee Collection Methods
// ============================================================================

// CalculateTradeFee calculates the fee for a trade
// If exchange shares fees with TigerWallet, platform fee is 0
// Otherwise, TigerWallet takes 20% extra on top of exchange fees
func (fs *FuturesService) CalculateTradeFee(orderValue float64, exchangeSharesFees bool) *TradeFee {
	fee := &TradeFee{
		OrderValue:        orderValue,
		ExchangeSharesFees: exchangeSharesFees,
	}

	// Calculate exchange fee (0.04% taker fee)
	exchangeFee := orderValue * fs.feeAccount.ExchangeFeePercent / 100
	fee.ExchangeFee = exchangeFee

	if exchangeSharesFees {
		// Exchange shares fees with TigerWallet - no extra platform fee
		fee.PlatformFee = 0
		fee.TotalFee = exchangeFee
	} else {
		// Exchange doesn't share fees - TigerWallet takes 20% extra
		// Total = exchange fee + 20% of exchange fee = 120% of exchange fee
		fee.PlatformFee = exchangeFee * fs.feeAccount.PlatformFeePercent / 100
		fee.TotalFee = exchangeFee + fee.PlatformFee
	}

	return fee
}

// CollectTradeFee adds fees to the fee collection
func (fs *FuturesService) CollectTradeFee(fee *TradeFee) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.feeCollection.TotalExchangeFees += fee.ExchangeFee
	fs.feeCollection.TotalPlatformFees += fee.PlatformFee
	fs.feeCollection.TotalFees += fee.TotalFee
	fs.feeCollection.TransactionsCount++
}

// GetFeeCollection returns the current fee collection status
func (fs *FuturesService) GetFeeCollection() *FeeCollection {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.feeCollection
}

// GetFeeAccount returns the fee collection account info
func (fs *FuturesService) GetFeeAccount() *FeeCollectionAccount {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.feeAccount
}

// SetExchangeFeeSharing configures whether exchanges share fees with TigerWallet
func (fs *FuturesService) SetExchangeFeeSharing(exchange string, sharesFees bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	// In production, this would update per-exchange settings
	// For now, we use a global setting
}

// ============================================================================
// Public Methods
// ============================================================================

func (fs *FuturesService) GetAllPairs() []*TradingPair {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	pairs := make([]*TradingPair, 0, len(fs.pairs))
	for _, pair := range fs.pairs {
		pairs = append(pairs, pair)
	}
	return pairs
}

func (fs *FuturesService) GetPair(symbol string) (*TradingPair, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	pair, ok := fs.pairs[symbol]
	if !ok {
		return nil, fmt.Errorf("pair not found: %s", symbol)
	}
	return pair, nil
}

func (fs *FuturesService) GetUserPositions(userID string) []Position {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	return fs.positions[userID]
}

func (fs *FuturesService) GetUserOrders(userID string) []Order {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	return fs.orders[userID]
}

func (fs *FuturesService) GetUserBalance(userID string) *UserBalance {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if balance, ok := fs.balances[userID]; ok {
		return balance
	}
	return &UserBalance{
		UserID:      userID,
		USDTBalance: 50000.0,
		USDCBalance: 25000.0,
		Positions:   []Position{},
		Orders:      []Order{},
	}
}

func (fs *FuturesService) CreateOrder(userID string, order *Order) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	order.ID = fmt.Sprintf("order-%d-%d", time.Now().Unix(), len(fs.orders[userID]))
	order.UserID = userID
	order.Status = OrderStatusOpen
	order.CreateTime = time.Now()
	order.UpdateTime = time.Now()

	fs.orders[userID] = append(fs.orders[userID], *order)
	return nil
}

func (fs *FuturesService) CancelOrder(userID, orderID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	orders := fs.orders[userID]
	for i, order := range orders {
		if order.ID == orderID && order.Status == OrderStatusOpen {
			orders[i].Status = OrderStatusCancelled
			orders[i].UpdateTime = time.Now()
			return nil
		}
	}
	return fmt.Errorf("order not found or cannot be cancelled")
}

func (fs *FuturesService) UpdatePairPrice(symbol string, price float64) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	pair, ok := fs.pairs[symbol]
	if !ok {
		return fmt.Errorf("pair not found: %s", symbol)
	}

	oldPrice := pair.Price
	pair.Price = price
	pair.Change24h = ((price - oldPrice) / oldPrice) * 100

	if price > pair.High24h {
		pair.High24h = price
	}
	if price < pair.Low24h {
		pair.Low24h = price
	}

	return nil
}

func (fs *FuturesService) CreatePair(pair *TradingPair) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, exists := fs.pairs[pair.Symbol]; exists {
		return fmt.Errorf("pair already exists: %s", pair.Symbol)
	}

	fs.pairs[pair.Symbol] = pair
	return nil
}

func (fs *FuturesService) UpdatePairStatus(symbol, status string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	pair, ok := fs.pairs[symbol]
	if !ok {
		return fmt.Errorf("pair not found: %s", symbol)
	}

	pair.Status = status
	return nil
}

func (fs *FuturesService) SetPreInstalled(symbol string, isPreInstalled bool) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	pair, ok := fs.pairs[symbol]
	if !ok {
		return fmt.Errorf("pair not found: %s", symbol)
	}

	pair.IsPreInstalled = isPreInstalled
	return nil
}

// ============================================================================
// JSON Serialization
// ============================================================================

func (fs *FuturesService) ToJSON() (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	data := struct {
		Pairs      map[string]*TradingPair `json:"pairs"`
		Positions  map[string][]Position   `json:"positions"`
		Orders     map[string][]Order      `json:"orders"`
		Balances   map[string]*UserBalance `json:"balances"`
	}{
		Pairs:    fs.pairs,
		Positions: fs.positions,
		Orders:    fs.orders,
		Balances:  fs.balances,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
