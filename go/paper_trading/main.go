/**
 * TigerWallet Paper Trading Engine
 * Simulated trading environment for testing strategies
 */

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

type PaperTradingEngine struct {
	config         *PaperTradingConfig
	accounts       map[string]*PaperAccount
	orders         map[string]*PaperOrder
	positions      map[string]*PaperPosition
	orderBook      *OrderBook
	mu             sync.RWMutex
	priceGenerator *PriceGenerator
	startedAt      time.Time
}

type PaperTradingConfig struct {
	InitialCapital    float64   `json:"initialCapital"`
	Leverage          float64   `json:"leverage"`
	MakerFee          float64   `json:"makerFee"`
	TakerFee          float64   `json:"takerFee"`
	Slippage          float64   `json:"slippage"`
	SimulationSpeed  float64   `json:"simulationSpeed"`
	EnableShorting    bool      `json:"enableShorting"`
	EnableMargin      bool      `json:"enableMargin"`
}

type PaperAccount struct {
	UserID       string             `json:"userId"`
	Balance      float64            `json:"balance"`
	Equity       float64            `json:"equity"`
	MarginUsed   float64            `json:"marginUsed"`
	MarginFree   float64            `json:"marginFree"`
	UnrealizedPNL float64           `json:"unrealizedPnl"`
	Positions    map[string]*PaperPosition `json:"positions"`
}

type PaperPosition struct {
	Symbol           string    `json:"symbol"`
	Side             string    `json:"side"`
	EntryPrice       float64   `json:"entryPrice"`
	CurrentPrice     float64   `json:"currentPrice"`
	Quantity         float64   `json:"quantity"`
	Leverage         float64   `json:"leverage"`
	UnrealizedPNL    float64   `json:"unrealizedPnl"`
	ROE              float64   `json:"roe"`
	LiquidationPrice float64   `json:"liquidationPrice"`
	OpenedAt         int64     `json:"openedAt"`
	UpdatedAt        int64     `json:"updatedAt"`
}

type PaperOrder struct {
	OrderID      string    `json:"orderId"`
	UserID       string    `json:"userId"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"`
	OrderType    string    `json:"orderType"`
	Price        float64   `json:"price"`
	StopPrice    float64   `json:"stopPrice"`
	Quantity     float64   `json:"quantity"`
	FilledQty    float64   `json:"filledQty"`
	AvgFillPrice float64   `json:"avgFillPrice"`
	Status       string    `json:"status"`
	SideEffect   string    `json:"sideEffect"`
	CreatedAt    int64     `json:"createdAt"`
	UpdatedAt    int64     `json:"updatedAt"`
	FilledAt     *int64    `json:"filledAt"`
}

type OrderBook struct {
	Bids []PriceLevel `json:"bids"`
	Asks []PriceLevel `json:"asks"`
}

type PriceLevel struct {
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
}

type PriceGenerator struct {
	basePrice  float64
	volatility float64
	trend      float64
	random     *rand.Rand
	mu         sync.RWMutex
}

type Trade struct {
	TradeID   string  `json:"tradeId"`
	OrderID   string  `json:"orderId"`
	UserID    string  `json:"userId"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Price     float64 `json:"price"`
	Quantity  float64 `json:"quantity"`
	Fee       float64 `json:"fee"`
	PNL       float64 `json:"pnl"`
	Timestamp int64   `json:"timestamp"`
}

type PaperTradingStats struct {
	TotalTrades    int     `json:"totalTrades"`
	WinningTrades  int     `json:"winningTrades"`
	LosingTrades   int     `json:"losingTrades"`
	WinRate        float64 `json:"winRate"`
	TotalVolume    float64 `json:"totalVolume"`
	TotalFees      float64 `json:"totalFees"`
	TotalPNL       float64 `json:"totalPnl"`
	BestTrade      float64 `json:"bestTrade"`
	WorstTrade     float64 `json:"worstTrade"`
	AverageWin     float64 `json:"averageWin"`
	AverageLoss    float64 `json:"averageLoss"`
	ProfitFactor   float64 `json:"profitFactor"`
	SharpeRatio    float64 `json:"sharpeRatio"`
	MaxDrawdown    float64 `json:"maxDrawdown"`
}

// ============================================================================
// Constructor
// ============================================================================

func NewPaperTradingEngine(config *PaperTradingConfig) *PaperTradingEngine {
	if config == nil {
		config = &PaperTradingConfig{
			InitialCapital:  10000.0,
			Leverage:        1.0,
			MakerFee:        0.001,
			TakerFee:        0.001,
			Slippage:        0.0005,
			SimulationSpeed: 1.0,
			EnableShorting:  true,
			EnableMargin:    true,
		}
	}

	return &PaperTradingEngine{
		config:         config,
		accounts:      make(map[string]*PaperAccount),
		orders:         make(map[string]*PaperOrder),
		positions:      make(map[string]*PaperPosition),
		orderBook:      NewOrderBook(),
		priceGenerator: NewPriceGenerator(config.InitialCapital),
		startedAt:      time.Now(),
	}
}

func NewPriceGenerator(basePrice float64) *PriceGenerator {
	return &PriceGenerator{
		basePrice:  basePrice,
		volatility: 0.02,
		trend:      0.0,
		random:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		Bids: make([]PriceLevel, 0),
		Asks: make([]PriceLevel, 0),
	}
}

// ============================================================================
// Account Management
// ============================================================================

func (e *PaperTradingEngine) CreateAccount(userID string) *PaperAccount {
	e.mu.Lock()
	defer e.mu.Unlock()

	account := &PaperAccount{
		UserID:      userID,
		Balance:     e.config.InitialCapital,
		Equity:      e.config.InitialCapital,
		MarginUsed:  0,
		MarginFree:  e.config.InitialCapital,
		Positions:   make(map[string]*PaperPosition),
	}

	e.accounts[userID] = account
	return account
}

func (e *PaperTradingEngine) GetAccount(userID string) (*PaperAccount, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	account, ok := e.accounts[userID]
	if !ok {
		return nil, fmt.Errorf("account not found: %s", userID)
	}

	return account, nil
}

func (e *PaperTradingEngine) ResetAccount(userID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.accounts[userID]; !ok {
		return fmt.Errorf("account not found: %s", userID)
	}

	e.accounts[userID] = &PaperAccount{
		UserID:      userID,
		Balance:     e.config.InitialCapital,
		Equity:      e.config.InitialCapital,
		MarginUsed:  0,
		MarginFree:  e.config.InitialCapital,
		Positions:   make(map[string]*PaperPosition),
	}

	for id, order := range e.orders {
		if order.UserID == userID {
			delete(e.orders, id)
		}
	}

	return nil
}

// ============================================================================
// Order Management
// ============================================================================

func (e *PaperTradingEngine) CreateOrder(userID, symbol, side, orderType string, quantity, price, stopPrice float64) (*PaperOrder, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	account, ok := e.accounts[userID]
	if !ok {
		return nil, fmt.Errorf("account not found: %s", userID)
	}

	margin := quantity * price / e.config.Leverage

	if side == "buy" && margin > account.MarginFree {
		return nil, fmt.Errorf("insufficient margin: required %f, available %f", margin, account.MarginFree)
	}

	orderID := fmt.Sprintf("paper_%d_%s", time.Now().UnixNano(), userID[:8])

	order := &PaperOrder{
		OrderID:    orderID,
		UserID:     userID,
		Symbol:     symbol,
		Side:       side,
		OrderType:  orderType,
		Price:      price,
		StopPrice:  stopPrice,
		Quantity:   quantity,
		FilledQty:  0,
		Status:     "pending",
		SideEffect: "margin",
		CreatedAt:  time.Now().UnixMilli(),
		UpdatedAt:  time.Now().UnixMilli(),
	}

	e.orders[orderID] = order

	if orderType == "market" {
		e.executeOrder(order)
	}

	return order, nil
}

func (e *PaperTradingEngine) CancelOrder(orderID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	order, ok := e.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	if order.Status != "pending" {
		return fmt.Errorf("order cannot be cancelled: %s", order.Status)
	}

	order.Status = "cancelled"
	order.UpdatedAt = time.Now().UnixMilli()

	return nil
}

func (e *PaperTradingEngine) GetOrder(orderID string) (*PaperOrder, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	order, ok := e.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	return order, nil
}

func (e *PaperTradingEngine) GetOpenOrders(userID string) []*PaperOrder {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var openOrders []*PaperOrder
	for _, order := range e.orders {
		if order.UserID == userID && order.Status == "pending" {
			openOrders = append(openOrders, order)
		}
	}

	return openOrders
}

// ============================================================================
// Order Execution
// ============================================================================

func (e *PaperTradingEngine) executeOrder(order *PaperOrder) {
	price := e.priceGenerator.GetPrice(order.Symbol)

	var fillPrice float64
	if order.Side == "buy" {
		fillPrice = price * (1 + e.config.Slippage)
	} else {
		fillPrice = price * (1 - e.config.Slippage)
	}

	if order.Price > 0 && order.OrderType == "limit" {
		fillPrice = order.Price
	}

	order.AvgFillPrice = fillPrice
	order.FilledQty = order.Quantity
	order.Status = "filled"
	order.FilledAt = new(int64)
	*order.FilledAt = time.Now().UnixMilli()
	order.UpdatedAt = time.Now().UnixMilli()

	account := e.accounts[order.UserID]
	fee := order.Quantity * fillPrice * e.config.TakerFee

	if order.Side == "buy" {
		margin := order.Quantity * fillPrice / e.config.Leverage
		account.MarginUsed += margin
		account.MarginFree -= margin
		account.Balance -= fee
	} else {
		account.Balance += order.Quantity * fillPrice
		account.Balance -= fee
	}

	e.updatePosition(order, fillPrice)
	e.updateAccountEquity(account)
}

func (e *PaperTradingEngine) updatePosition(order *PaperOrder, fillPrice float64) {
	positionKey := fmt.Sprintf("%s_%s", order.UserID, order.Symbol)

	if existingPos, ok := e.positions[positionKey]; ok {
		totalQty := existingPos.Quantity + order.Quantity
		if order.Side == existingPos.Side {
			existingPos.EntryPrice = (existingPos.EntryPrice*existingPos.Quantity + fillPrice*order.Quantity) / totalQty
			existingPos.Quantity = totalQty
		} else {
			if order.Quantity >= existingPos.Quantity {
				existingPos.Side = order.Side
				existingPos.EntryPrice = fillPrice
				existingPos.Quantity = order.Quantity - existingPos.Quantity
			} else {
				existingPos.Quantity -= order.Quantity
			}
		}
		existingPos.UpdatedAt = time.Now().UnixMilli()
	} else if order.Side == "buy" {
		e.positions[positionKey] = &PaperPosition{
			Symbol:           order.Symbol,
			Side:             "long",
			EntryPrice:       fillPrice,
			CurrentPrice:     fillPrice,
			Quantity:         order.Quantity,
			Leverage:         e.config.Leverage,
			OpenedAt:         time.Now().UnixMilli(),
			UpdatedAt:        time.Now().UnixMilli(),
		}
	} else if e.config.EnableShorting {
		e.positions[positionKey] = &PaperPosition{
			Symbol:           order.Symbol,
			Side:             "short",
			EntryPrice:       fillPrice,
			CurrentPrice:     fillPrice,
			Quantity:         order.Quantity,
			Leverage:         e.config.Leverage,
			OpenedAt:         time.Now().UnixMilli(),
			UpdatedAt:        time.Now().UnixMilli(),
		}
	}
}

func (e *PaperTradingEngine) updateAccountEquity(account *PaperAccount) {
	var totalUnrealizedPNL float64

	for _, pos := range account.Positions {
		price := e.priceGenerator.GetPrice(pos.Symbol)
		pos.CurrentPrice = price

		if pos.Side == "long" {
			pos.UnrealizedPNL = (price - pos.EntryPrice) * pos.Quantity
		} else {
			pos.UnrealizedPNL = (pos.EntryPrice - price) * pos.Quantity
		}

		margin := pos.EntryPrice * pos.Quantity / pos.Leverage
		if margin > 0 {
			pos.ROE = (pos.UnrealizedPNL / margin) * 100
		}

		if pos.Side == "long" {
			pos.LiquidationPrice = pos.EntryPrice * (1 - 1/pos.Leverage)
		} else {
			pos.LiquidationPrice = pos.EntryPrice * (1 + 1/pos.Leverage)
		}

		totalUnrealizedPNL += pos.UnrealizedPNL
	}

	account.UnrealizedPNL = totalUnrealizedPNL
	account.Equity = account.Balance + totalUnrealizedPNL
	account.MarginFree = account.Balance - account.MarginUsed + totalUnrealizedPNL
}

// ============================================================================
// Price Generation
// ============================================================================

func (pg *PriceGenerator) GetPrice(symbol string) float64 {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	change := (pg.random.Float64() - 0.5) * 2 * pg.volatility
	change += pg.trend * 0.1

	pg.basePrice *= (1 + change)

	noise := (pg.random.Float64() - 0.5) * pg.volatility * 0.5

	return pg.basePrice * (1 + noise)
}

func (pg *PriceGenerator) SetVolatility(volatility float64) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.volatility = volatility
}

func (pg *PriceGenerator) SetTrend(trend float64) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.trend = trend
}

// ============================================================================
// Statistics
// ============================================================================

func (e *PaperTradingEngine) GetStats(userID string) (*PaperTradingStats, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	positionKey := fmt.Sprintf("%s_%s", userID, "*")
	var totalPNL, bestTrade, worstTrade float64
	var winningTrades, losingTrades int

	for key, pos := range e.positions {
		if len(key) > len(userID) && key[:len(userID)] == userID {
			pnl := pos.UnrealizedPNL
			totalPNL += pnl

			if pnl > 0 {
				winningTrades++
				if pnl > bestTrade {
					bestTrade = pnl
				}
			} else if pnl < 0 {
				losingTrades++
				if pnl < worstTrade || worstTrade == 0 {
					worstTrade = pnl
				}
			}
		}
	}

	totalTrades := winningTrades + losingTrades
	winRate := float64(winningTrades) / float64(totalTrades) * 100

	var averageWin, averageLoss float64
	if winningTrades > 0 {
		averageWin = bestTrade / float64(winningTrades)
	}
	if losingTrades > 0 {
		averageLoss = math.Abs(worstTrade) / float64(losingTrades)
	}

	profitFactor := 0.0
	if averageLoss > 0 {
		profitFactor = averageWin / averageLoss
	}

	stats := &PaperTradingStats{
		TotalTrades:   totalTrades,
		WinningTrades: winningTrades,
		LosingTrades:  losingTrades,
		WinRate:       winRate,
		TotalPNL:      totalPNL,
		BestTrade:     bestTrade,
		WorstTrade:    worstTrade,
		AverageWin:    averageWin,
		AverageLoss:   averageLoss,
		ProfitFactor:  profitFactor,
	}

	return stats, nil
}

func (e *PaperTradingEngine) ToJSON() (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	data := map[string]interface{}{
		"config":    e.config,
		"accounts":  e.accounts,
		"orders":    e.orders,
		"startedAt": e.startedAt.Unix(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// ============================================================================
// Main
// ============================================================================

func main() {
	fmt.Println("TigerWallet - Paper Trading Engine")
	fmt.Println("==================================")

	config := &PaperTradingConfig{
		InitialCapital: 10000.0,
		Leverage:       3.0,
		MakerFee:       0.001,
		TakerFee:       0.001,
		Slippage:       0.0005,
		SimulationSpeed: 1.0,
		EnableShorting: true,
		EnableMargin:   true,
	}

	engine := NewPaperTradingEngine(config)

	userID := "test_user_1"
	account := engine.CreateAccount(userID)
	fmt.Printf("Created account with balance: $%.2f\n", account.Balance)

	order1, err := engine.CreateOrder(userID, "BTC/USDT", "buy", "market", 0.1, 0, 0)
	if err != nil {
		fmt.Printf("Order 1 error: %v\n", err)
	} else {
		fmt.Printf("Placed order 1: %s\n", order1.OrderID)
	}

	account, _ = engine.GetAccount(userID)
	fmt.Printf("Account balance: $%.2f\n", account.Balance)
	fmt.Printf("Account equity: $%.2f\n", account.Equity)

	stats, _ := engine.GetStats(userID)
	fmt.Printf("Total trades: %d\n", stats.TotalTrades)
	fmt.Printf("Win rate: %.2f%%\n", stats.WinRate)
}
