package distributed_trading

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// Types
// ============================================================================

type OrderType string
type OrderSide string
type OrderStatus string
type MarginMode string
type PositionSide string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
	OrderTypeStop   OrderType = "stop"

	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"

	OrderStatusOpen      OrderStatus = "open"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"

	MarginModeCross    MarginMode = "cross"
	MarginModeIsolated MarginMode = "isolated"

	PositionSideLong  PositionSide = "long"
	PositionSideShort PositionSide = "short"
)

// ============================================================================
// Trading Pair
// ============================================================================

type TradingPair struct {
	ID             string  `json:"id"`
	Symbol         string  `json:"symbol"`
	Base           string  `json:"base"`
	Quote          string  `json:"quote"`
	Price          float64 `json:"price"`
	High24h        float64 `json:"high24h"`
	Low24h         float64 `json:"low24h"`
	Volume24h      float64 `json:"volume24h"`
	Change24h      float64 `json:"change24h"`
	IsPreInstalled bool    `json:"isPreInstalled"`
	Status         string  `json:"status"`
	MinOrderSize   float64 `json:"minOrderSize"`
	MaxOrderSize   float64 `json:"maxOrderSize"`
	MakerFee       float64 `json:"makerFee"`
	TakerFee       float64 `json:"takerFee"`
}

// ============================================================================
// Order
// ============================================================================

type Order struct {
	ID         string      `json:"id"`
	UserID     string      `json:"userId"`
	Symbol     string      `json:"symbol"`
	Side       OrderSide   `json:"side"`
	Type       OrderType   `json:"type"`
	Size       float64     `json:"size"`
	Filled     float64     `json:"filled"`
	Price      float64     `json:"price"`
	StopPrice  float64     `json:"stopPrice,omitempty"`
	Leverage   int         `json:"leverage"`
	MarginMode MarginMode  `json:"marginMode"`
	Status     OrderStatus `json:"status"`
	CreateTime int64       `json:"createTime"`
	UpdateTime int64       `json:"updateTime"`
}

// ============================================================================
// Position
// ============================================================================

type Position struct {
	ID               string       `json:"id"`
	UserID           string       `json:"userId"`
	Symbol           string       `json:"symbol"`
	Side             PositionSide `json:"side"`
	Size             float64      `json:"size"`
	EntryPrice       float64      `json:"entryPrice"`
	MarkPrice        float64      `json:"markPrice"`
	Leverage         int          `json:"leverage"`
	Margin           float64      `json:"margin"`
	MarginMode       MarginMode   `json:"marginMode"`
	PNL              float64      `json:"pnl"`
	PNLPercent       float64      `json:"pnlPercent"`
	LiquidationPrice float64      `json:"liquidationPrice"`
	OpenTime         int64        `json:"openTime"`
}

// ============================================================================
// Distributed Trading Service
// ============================================================================

type DistributedTradingService struct {
	mu         sync.RWMutex
	pairs      map[string]*TradingPair
	orders     map[string]*Order
	positions  map[string]*Position
	orderCount int64
	pairCount  int64

	// For distributed consistency
	shards       int
	shardMutexes []sync.RWMutex

	// Metrics
	totalOrders  int64
	totalVolume  float64
	ordersPerSec int64
}

func NewDistributedTradingService(shards int) *DistributedTradingService {
	dts := &DistributedTradingService{
		pairs:        make(map[string]*TradingPair),
		orders:       make(map[string]*Order),
		positions:    make(map[string]*Position),
		shards:       shards,
		shardMutexes: make([]sync.RWMutex, shards),
	}
	dts.initializePairs()
	go dts.StartPriceUpdater()
	return dts
}

func (dts *DistributedTradingService) initializePairs() {
	bases := []string{"BTC", "ETH", "BNB", "SOL", "XRP", "DOGE", "ADA", "AVAX", "DOT", "LINK", "MATIC", "LTC", "UNI", "ATOM", "XLM", "NEAR", "APT", "ARB", "OP", "INJ"}
	quotes := []string{"USDT", "USDC"}

	// Real live prices from the CoinGecko oracle. Fail-closed: a base asset
	// without a real price is not listed (never fabricated).
	prices, err := fetchLivePricesUSD(append(append([]string{}, bases...), quotes...))
	if err != nil {
		prices = map[string]float64{}
	}

	id := 0
	for _, base := range bases {
		price := prices[base]
		if price <= 0 {
			continue // no real price: do not list the pair
		}
		for _, quote := range quotes {
			if base == quote {
				continue
			}
			quotePrice := prices[quote]
			if quotePrice <= 0 {
				continue
			}
			id++
			pairPrice := price / quotePrice
			pair := &TradingPair{
				ID:             fmt.Sprintf("pair-%d", id),
				Symbol:         fmt.Sprintf("%s/%s", base, quote),
				Base:           base,
				Quote:          quote,
				Price:          pairPrice,
				High24h:        pairPrice,
				Low24h:         pairPrice,
				Volume24h:      0,
				Change24h:      0,
				IsPreInstalled: true,
				Status:         "active",
				MinOrderSize:   0.001,
				MaxOrderSize:   1000000,
				MakerFee:       0.02,
				TakerFee:       0.04,
			}
			dts.pairs[pair.Symbol] = pair
		}
	}
}

// StartPriceUpdater refreshes every listed pair from the real CoinGecko
// oracle every 30s via UpdatePrice (which also re-marks positions).
func (dts *DistributedTradingService) StartPriceUpdater() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		dts.mu.RLock()
		type pq struct{ sym, base, quote string }
		var list []pq
		bases := map[string]bool{}
		quotes := map[string]bool{}
		for sym, pair := range dts.pairs {
			list = append(list, pq{sym, pair.Base, pair.Quote})
			bases[pair.Base] = true
			quotes[pair.Quote] = true
		}
		dts.mu.RUnlock()
		symbols := make([]string, 0, len(bases)+len(quotes))
		for s := range bases {
			symbols = append(symbols, s)
		}
		for s := range quotes {
			symbols = append(symbols, s)
		}
		prices, err := fetchLivePricesUSD(symbols)
		if err != nil || len(prices) == 0 {
			continue // keep last known real prices
		}
		for _, item := range list {
			baseP := prices[item.base]
			quoteP := prices[item.quote]
			if baseP <= 0 || quoteP <= 0 {
				continue
			}
			_ = dts.UpdatePrice(item.sym, baseP/quoteP)
		}
	}
}

// ============================================================================
// Public Methods
// ============================================================================

func (dts *DistributedTradingService) GetAllPairs() []*TradingPair {
	dts.mu.RLock()
	defer dts.mu.RUnlock()

	pairs := make([]*TradingPair, 0, len(dts.pairs))
	for _, pair := range dts.pairs {
		pairs = append(pairs, pair)
	}
	return pairs
}

func (dts *DistributedTradingService) GetPair(symbol string) (*TradingPair, error) {
	dts.mu.RLock()
	defer dts.mu.RUnlock()

	pair, ok := dts.pairs[symbol]
	if !ok {
		return nil, fmt.Errorf("pair not found: %s", symbol)
	}
	return pair, nil
}

func (dts *DistributedTradingService) GetPreInstalledPairs() []*TradingPair {
	dts.mu.RLock()
	defer dts.mu.RUnlock()

	var pairs []*TradingPair
	for _, pair := range dts.pairs {
		if pair.IsPreInstalled {
			pairs = append(pairs, pair)
		}
	}
	return pairs
}

func (dts *DistributedTradingService) GetTotalPairs() int64 {
	return atomic.LoadInt64(&dts.pairCount)
}

func (dts *DistributedTradingService) CreateOrder(userID, symbol string, side OrderSide, orderType OrderType, size, price float64, leverage int) (*Order, error) {
	dts.mu.Lock()
	defer dts.mu.Unlock()

	// Validate pair exists
	if _, ok := dts.pairs[symbol]; !ok {
		return nil, fmt.Errorf("pair not found: %s", symbol)
	}

	atomic.AddInt64(&dts.orderCount, 1)
	orderID := fmt.Sprintf("order-%d", dts.orderCount)
	now := time.Now().UnixNano()

	order := &Order{
		ID:         orderID,
		UserID:     userID,
		Symbol:     symbol,
		Side:       side,
		Type:       orderType,
		Size:       size,
		Filled:     0,
		Price:      price,
		StopPrice:  0,
		Leverage:   leverage,
		MarginMode: MarginModeCross,
		Status:     OrderStatusOpen,
		CreateTime: now,
		UpdateTime: now,
	}

	dts.orders[orderID] = order
	atomic.AddInt64(&dts.totalOrders, 1)

	return order, nil
}

func (dts *DistributedTradingService) CancelOrder(orderID string) error {
	dts.mu.Lock()
	defer dts.mu.Unlock()

	order, ok := dts.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	if order.Status != OrderStatusOpen {
		return fmt.Errorf("order cannot be cancelled")
	}

	order.Status = OrderStatusCancelled
	order.UpdateTime = time.Now().UnixNano()
	return nil
}

func (dts *DistributedTradingService) GetOrder(orderID string) (*Order, error) {
	dts.mu.RLock()
	defer dts.mu.RUnlock()

	order, ok := dts.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	return order, nil
}

func (dts *DistributedTradingService) OpenPosition(userID, symbol string, side PositionSide, size, entryPrice float64, leverage int) (*Position, error) {
	dts.mu.Lock()
	defer dts.mu.Unlock()

	margin := (size * entryPrice) / float64(leverage)
	var liquidationPrice float64

	if side == PositionSideLong {
		liquidationPrice = entryPrice * (1.0 - 1.0/float64(leverage))
	} else {
		liquidationPrice = entryPrice * (1.0 + 1.0/float64(leverage))
	}

	atomic.AddInt64(&dts.orderCount, 1)
	positionID := fmt.Sprintf("pos-%d", dts.orderCount)
	now := time.Now().UnixNano()

	position := &Position{
		ID:               positionID,
		UserID:           userID,
		Symbol:           symbol,
		Side:             side,
		Size:             size,
		EntryPrice:       entryPrice,
		MarkPrice:        entryPrice,
		Leverage:         leverage,
		Margin:           margin,
		MarginMode:       MarginModeCross,
		PNL:              0,
		PNLPercent:       0,
		LiquidationPrice: liquidationPrice,
		OpenTime:         now,
	}

	key := fmt.Sprintf("%s_%s", userID, symbol)
	dts.positions[key] = position
	dts.totalVolume += size * entryPrice

	return position, nil
}

func (dts *DistributedTradingService) GetPosition(userID, symbol string) (*Position, error) {
	dts.mu.RLock()
	defer dts.mu.RUnlock()

	key := fmt.Sprintf("%s_%s", userID, symbol)
	position, ok := dts.positions[key]
	if !ok {
		return nil, fmt.Errorf("position not found")
	}
	return position, nil
}

func (dts *DistributedTradingService) GetUserPositions(userID string) []*Position {
	dts.mu.RLock()
	defer dts.mu.RUnlock()

	var positions []*Position
	for _, position := range dts.positions {
		if position.UserID == userID {
			positions = append(positions, position)
		}
	}
	return positions
}

func (dts *DistributedTradingService) UpdatePrice(symbol string, newPrice float64) error {
	dts.mu.Lock()
	defer dts.mu.Unlock()

	pair, ok := dts.pairs[symbol]
	if !ok {
		return fmt.Errorf("pair not found: %s", symbol)
	}

	oldPrice := pair.Price
	pair.Price = newPrice

	if newPrice > pair.High24h {
		pair.High24h = newPrice
	}
	if newPrice < pair.Low24h {
		pair.Low24h = newPrice
	}

	pair.Change24h = ((newPrice - oldPrice) / oldPrice) * 100

	// Update all positions for this symbol
	for _, position := range dts.positions {
		if position.Symbol == symbol {
			position.MarkPrice = newPrice
			dts.calculatePNL(position)
		}
	}

	return nil
}

func (dts *DistributedTradingService) calculatePNL(position *Position) {
	var priceDiff float64
	if position.Side == PositionSideLong {
		priceDiff = position.MarkPrice - position.EntryPrice
	} else {
		priceDiff = position.EntryPrice - position.MarkPrice
	}

	position.PNL = priceDiff * position.Size
	position.PNLPercent = (position.PNL / position.Margin) * 100
}

// ============================================================================
// Distributed Metrics
// ============================================================================

func (dts *DistributedTradingService) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"totalPairs":   atomic.LoadInt64(&dts.pairCount),
		"totalOrders":  atomic.LoadInt64(&dts.totalOrders),
		"totalVolume":  dts.totalVolume,
		"ordersPerSec": atomic.LoadInt64(&dts.ordersPerSec),
		"activeShards": dts.shards,
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func CalculateRequiredMargin(orderValue float64, leverage int) float64 {
	return orderValue / float64(leverage)
}

func CalculatePNL(entryPrice, currentPrice float64, size float64, side PositionSide) float64 {
	if side == PositionSideLong {
		return (currentPrice - entryPrice) * size
	}
	return (entryPrice - currentPrice) * size
}

// ============================================================================
// Sharded Access for High Load
// ============================================================================

func (dts *DistributedTradingService) getShardIndex(key string) int {
	hash := 0
	for _, c := range key {
		hash = hash*31 + int(c)
	}
	return hash % dts.shards
}

func (dts *DistributedTradingService) getShardMutex(key string) *sync.RWMutex {
	index := dts.getShardIndex(key)
	return &dts.shardMutexes[index]
}
