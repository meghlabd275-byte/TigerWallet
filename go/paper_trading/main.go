/**
 * TigerWallet Paper Trading Engine
 *
 * A risk-free order-simulation environment: orders are matched against real
 * market prices but no funds move. Prices come from the TigerWallet price
 * oracle, which aggregates live venue data. If the oracle is unreachable the
 * engine refuses to fill rather than inventing a price, so a simulated PnL is
 * always derived from a price that actually existed.
 *
 * Order ids carry a `paper_` prefix and the account/order types are named
 * Paper*, so a simulated record can never be mistaken for a settled trade.
 */

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

type PaperTradingEngine struct {
	config    *PaperTradingConfig
	accounts  map[string]*PaperAccount
	orders    map[string]*PaperOrder
	positions map[string]*PaperPosition
	prices    PriceSource
	mu        sync.RWMutex
	startedAt time.Time
}

type PaperTradingConfig struct {
	InitialCapital  float64 `json:"initialCapital"`
	Leverage        float64 `json:"leverage"`
	MakerFee        float64 `json:"makerFee"`
	TakerFee        float64 `json:"takerFee"`
	Slippage        float64 `json:"slippage"`
	SimulationSpeed float64 `json:"simulationSpeed"`
	EnableShorting  bool    `json:"enableShorting"`
	EnableMargin    bool    `json:"enableMargin"`
}

type PaperAccount struct {
	UserID        string                    `json:"userId"`
	Balance       float64                   `json:"balance"`
	Equity        float64                   `json:"equity"`
	MarginUsed    float64                   `json:"marginUsed"`
	MarginFree    float64                   `json:"marginFree"`
	UnrealizedPNL float64                   `json:"unrealizedPnl"`
	Positions     map[string]*PaperPosition `json:"positions"`
}

type PaperPosition struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	EntryPrice       float64 `json:"entryPrice"`
	CurrentPrice     float64 `json:"currentPrice"`
	Quantity         float64 `json:"quantity"`
	Leverage         float64 `json:"leverage"`
	UnrealizedPNL    float64 `json:"unrealizedPnl"`
	ROE              float64 `json:"roe"`
	LiquidationPrice float64 `json:"liquidationPrice"`
	OpenedAt         int64   `json:"openedAt"`
	UpdatedAt        int64   `json:"updatedAt"`
}

type PaperOrder struct {
	OrderID      string  `json:"orderId"`
	UserID       string  `json:"userId"`
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	OrderType    string  `json:"orderType"`
	Price        float64 `json:"price"`
	StopPrice    float64 `json:"stopPrice"`
	Quantity     float64 `json:"quantity"`
	FilledQty    float64 `json:"filledQty"`
	AvgFillPrice float64 `json:"avgFillPrice"`
	Fee          float64 `json:"fee"`
	Status       string  `json:"status"`
	SideEffect   string  `json:"sideEffect"`
	CreatedAt    int64   `json:"createdAt"`
	UpdatedAt    int64   `json:"updatedAt"`
	FilledAt     *int64  `json:"filledAt"`
}

type PaperTradingStats struct {
	TotalTrades   int     `json:"totalTrades"`
	WinningTrades int     `json:"winningTrades"`
	LosingTrades  int     `json:"losingTrades"`
	WinRate       float64 `json:"winRate"`
	TotalVolume   float64 `json:"totalVolume"`
	TotalFees     float64 `json:"totalFees"`
	TotalPNL      float64 `json:"totalPnl"`
	BestTrade     float64 `json:"bestTrade"`
	WorstTrade    float64 `json:"worstTrade"`
	AverageWin    float64 `json:"averageWin"`
	AverageLoss   float64 `json:"averageLoss"`
	ProfitFactor  float64 `json:"profitFactor"`
}

// ErrNoPrice is returned when the price oracle cannot supply a symbol. The
// engine never substitutes a synthetic price for a missing one.
var ErrNoPrice = errors.New("no market price available for symbol")

// ============================================================================
// Price source
// ============================================================================

// PriceSource supplies the reference price used to fill orders and to mark
// open positions. Implementations must return an error rather than a guessed
// value when the price is unknown.
type PriceSource interface {
	Price(ctx context.Context, symbol string) (float64, error)
}

// OraclePriceSource reads prices from the TigerWallet price oracle over HTTP.
type OraclePriceSource struct {
	baseURL string
	client  *http.Client
}

func NewOraclePriceSource(baseURL string) *OraclePriceSource {
	return &OraclePriceSource{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// oraclePriceResponse mirrors the oracle field this client consumes.
type oraclePriceResponse struct {
	Price float64 `json:"price"`
}

// Price fetches the current USD price for symbol. A non-200 response, a
// transport failure, or a non-positive price is reported as an error so the
// caller can refuse the fill.
func (o *OraclePriceSource) Price(ctx context.Context, symbol string) (float64, error) {
	if o == nil || o.baseURL == "" {
		return 0, fmt.Errorf("%w: price oracle not configured", ErrNoPrice)
	}
	// The oracle keys feeds by base asset ("BTC" for the pair "BTC/USDT").
	base := symbol
	if i := strings.IndexAny(symbol, "/-"); i > 0 {
		base = symbol[:i]
	}
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		return 0, fmt.Errorf("%w: empty symbol", ErrNoPrice)
	}

	endpoint := o.baseURL + "/api/v1/price/" + url.PathEscape(base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrNoPrice, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: oracle returned %d", ErrNoPrice, resp.StatusCode)
	}

	var body oraclePriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrNoPrice, err)
	}
	if !(body.Price > 0) || math.IsInf(body.Price, 0) || math.IsNaN(body.Price) {
		return 0, fmt.Errorf("%w: oracle reported a non-positive price", ErrNoPrice)
	}
	return body.Price, nil
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
		config:    config,
		accounts:  make(map[string]*PaperAccount),
		orders:    make(map[string]*PaperOrder),
		positions: make(map[string]*PaperPosition),
		prices:    NewOraclePriceSource(getEnv("PRICE_ORACLE_URL", "http://localhost:8087")),
		startedAt: time.Now(),
	}
}

// SetPriceSource replaces the price source. Used by tests and by deployments
// that obtain prices from a different oracle.
func (e *PaperTradingEngine) SetPriceSource(p PriceSource) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.prices = p
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// ============================================================================
// Account Management
// ============================================================================

func (e *PaperTradingEngine) CreateAccount(userID string) *PaperAccount {
	e.mu.Lock()
	defer e.mu.Unlock()

	account := &PaperAccount{
		UserID:     userID,
		Balance:    e.config.InitialCapital,
		Equity:     e.config.InitialCapital,
		MarginUsed: 0,
		MarginFree: e.config.InitialCapital,
		Positions:  make(map[string]*PaperPosition),
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
		UserID:     userID,
		Balance:    e.config.InitialCapital,
		Equity:     e.config.InitialCapital,
		MarginUsed: 0,
		MarginFree: e.config.InitialCapital,
		Positions:  make(map[string]*PaperPosition),
	}

	for id, order := range e.orders {
		if order.UserID == userID {
			delete(e.orders, id)
		}
	}
	for key := range e.positions {
		if ownerOf(key) == userID {
			delete(e.positions, key)
		}
	}

	return nil
}

// ============================================================================
// Position keys
// ============================================================================

// positionSeparator cannot appear in a user id or a symbol, so a composite key
// always splits back into exactly two fields.
const positionSeparator = "\x1f"

// positionKey namespaces a position by its owner, so two users trading the
// same symbol never share a position record.
func positionKey(userID, symbol string) string {
	return userID + positionSeparator + symbol
}

// ownerOf recovers the owning user id from a position key.
func ownerOf(key string) string {
	if i := strings.Index(key, positionSeparator); i >= 0 {
		return key[:i]
	}
	return key
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
	if !(quantity > 0) || math.IsNaN(quantity) || math.IsInf(quantity, 0) {
		return nil, errors.New("quantity must be a positive number")
	}
	if side != "buy" && side != "sell" {
		return nil, fmt.Errorf("invalid side: %s", side)
	}
	if orderType != "market" && orderType != "limit" {
		return nil, fmt.Errorf("invalid order type: %s", orderType)
	}
	if orderType == "limit" && !(price > 0) {
		return nil, errors.New("limit orders require a positive price")
	}
	if side == "sell" && !e.config.EnableShorting {
		return nil, errors.New("shorting is disabled")
	}

	// Margin is checked against the reference price, never a caller-supplied
	// one, so a caller cannot understate the margin an order requires.
	refPrice := price
	if orderType == "market" {
		quoted, err := e.prices.Price(context.Background(), symbol)
		if err != nil {
			return nil, err
		}
		refPrice = quoted
	}
	margin := quantity * refPrice / e.config.Leverage
	if margin > account.MarginFree {
		return nil, fmt.Errorf("insufficient margin: required %.2f, available %.2f", margin, account.MarginFree)
	}

	orderID := fmt.Sprintf("paper_%d_%s", time.Now().UnixNano(), shortID(userID))

	order := &PaperOrder{
		OrderID:    orderID,
		UserID:     userID,
		Symbol:     symbol,
		Side:       side,
		OrderType:  orderType,
		Price:      price,
		StopPrice:  stopPrice,
		Quantity:   quantity,
		Status:     "pending",
		SideEffect: "margin",
		CreatedAt:  time.Now().UnixMilli(),
		UpdatedAt:  time.Now().UnixMilli(),
	}

	e.orders[orderID] = order

	if orderType == "market" {
		if err := e.executeOrderLocked(order); err != nil {
			// A market order that cannot be priced stays pending rather than
			// being recorded as filled at a fabricated price.
			return order, err
		}
	}

	return order, nil
}

// shortID returns a bounded fragment of a user id for display inside an order
// id. It never slices past the end, so short ids do not panic.
func shortID(userID string) string {
	const n = 8
	if len(userID) <= n {
		return userID
	}
	return userID[:n]
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

// executeOrderLocked fills a market order against the live reference price.
// Callers must hold e.mu.
func (e *PaperTradingEngine) executeOrderLocked(order *PaperOrder) error {
	price, err := e.prices.Price(context.Background(), order.Symbol)
	if err != nil {
		return err
	}

	var fillPrice float64
	if order.Side == "buy" {
		fillPrice = price * (1 + e.config.Slippage)
	} else {
		fillPrice = price * (1 - e.config.Slippage)
	}

	order.AvgFillPrice = fillPrice
	order.FilledQty = order.Quantity
	order.Status = "filled"
	order.FilledAt = new(int64)
	*order.FilledAt = time.Now().UnixMilli()
	order.UpdatedAt = time.Now().UnixMilli()

	account := e.accounts[order.UserID]
	fee := order.Quantity * fillPrice * e.config.TakerFee
	order.Fee = fee

	if order.Side == "buy" {
		margin := order.Quantity * fillPrice / e.config.Leverage
		account.MarginUsed += margin
		account.MarginFree -= margin
		account.Balance -= fee
	} else {
		account.Balance += order.Quantity * fillPrice
		account.Balance -= fee
	}

	e.updatePositionLocked(order, fillPrice)
	e.updateAccountEquityLocked(account)
	return nil
}

// updatePositionLocked applies a fill to the owner's position for the symbol.
// Callers must hold e.mu.
func (e *PaperTradingEngine) updatePositionLocked(order *PaperOrder, fillPrice float64) {
	key := positionKey(order.UserID, order.Symbol)
	account := e.accounts[order.UserID]
	side := map[string]string{"buy": "long", "sell": "short"}[order.Side]

	if existing, ok := e.positions[key]; ok {
		sameSide := existing.Side == side

		switch {
		case sameSide:
			total := existing.Quantity + order.Quantity
			existing.EntryPrice = (existing.EntryPrice*existing.Quantity + fillPrice*order.Quantity) / total
			existing.Quantity = total
		case order.Quantity >= existing.Quantity:
			// The opposing order closes the position and opens the remainder
			// in the other direction.
			existing.Side = side
			existing.EntryPrice = fillPrice
			existing.Quantity = order.Quantity - existing.Quantity
		default:
			existing.Quantity -= order.Quantity
		}

		existing.CurrentPrice = fillPrice
		existing.UpdatedAt = time.Now().UnixMilli()
		if existing.Quantity <= 0 {
			delete(e.positions, key)
			delete(account.Positions, key)
			return
		}
		account.Positions[key] = existing
		return
	}

	pos := &PaperPosition{
		Symbol:       order.Symbol,
		Side:         side,
		EntryPrice:   fillPrice,
		CurrentPrice: fillPrice,
		Quantity:     order.Quantity,
		Leverage:     e.config.Leverage,
		OpenedAt:     time.Now().UnixMilli(),
		UpdatedAt:    time.Now().UnixMilli(),
	}
	e.positions[key] = pos
	account.Positions[key] = pos
}

// updateAccountEquityLocked revalues every open position of the account at the
// current market price and recomputes equity. Callers must hold e.mu.
func (e *PaperTradingEngine) updateAccountEquityLocked(account *PaperAccount) {
	var totalUnrealizedPNL float64

	for key, pos := range account.Positions {
		price, err := e.prices.Price(context.Background(), pos.Symbol)
		if err != nil {
			// Keep the last known mark rather than fabricating a new one.
			totalUnrealizedPNL += pos.UnrealizedPNL
			continue
		}
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
		pos.UpdatedAt = time.Now().UnixMilli()

		totalUnrealizedPNL += pos.UnrealizedPNL
		e.positions[key] = pos
	}

	account.UnrealizedPNL = totalUnrealizedPNL
	account.Equity = account.Balance + totalUnrealizedPNL
	account.MarginFree = account.Balance - account.MarginUsed + totalUnrealizedPNL
}

// ============================================================================
// Statistics
// ============================================================================

func (e *PaperTradingEngine) GetStats(userID string) (*PaperTradingStats, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	account, ok := e.accounts[userID]
	if !ok {
		return nil, fmt.Errorf("account not found: %s", userID)
	}
	e.updateAccountEquityLocked(account)

	stats := &PaperTradingStats{}
	var wins, losses int
	var grossWin, grossLoss float64

	for _, order := range e.orders {
		if order.UserID != userID || order.Status != "filled" {
			continue
		}
		stats.TotalTrades++
		stats.TotalVolume += order.FilledQty * order.AvgFillPrice
		stats.TotalFees += order.Fee
	}

	for _, pos := range account.Positions {
		pnl := pos.UnrealizedPNL
		stats.TotalPNL += pnl
		switch {
		case pnl > 0:
			wins++
			grossWin += pnl
			if pnl > stats.BestTrade {
				stats.BestTrade = pnl
			}
		case pnl < 0:
			losses++
			grossLoss += -pnl
			if pnl < stats.WorstTrade {
				stats.WorstTrade = pnl
			}
		}
	}

	stats.WinningTrades = wins
	stats.LosingTrades = losses
	if closed := wins + losses; closed > 0 {
		stats.WinRate = float64(wins) / float64(closed) * 100
	}
	if wins > 0 {
		stats.AverageWin = grossWin / float64(wins)
	}
	if losses > 0 {
		stats.AverageLoss = grossLoss / float64(losses)
	}
	if stats.AverageLoss > 0 {
		stats.ProfitFactor = stats.AverageWin / stats.AverageLoss
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
		"positions": e.positions,
		"startedAt": e.startedAt,
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// ============================================================================
// Main
// ============================================================================

func main() {
	fmt.Println("TigerWallet - Paper Trading Engine")
	fmt.Println("==================================")

	engine := NewPaperTradingEngine(nil)

	userID := "paper_demo_account"
	account := engine.CreateAccount(userID)
	fmt.Printf("Created account with balance: $%.2f\n", account.Balance)

	// Fill one simulated order against the live reference price. When the
	// oracle is unreachable the order stays pending and is reported as such.
	order, err := engine.CreateOrder(userID, "BTC/USDT", "buy", "market", 0.1, 0, 0)
	if err != nil {
		fmt.Printf("Order not filled: %v\n", err)
		if order != nil {
			fmt.Printf("Order %s remains %s\n", order.OrderID, order.Status)
		}
	} else {
		fmt.Printf("Filled order %s at $%.2f\n", order.OrderID, order.AvgFillPrice)
	}

	account, _ = engine.GetAccount(userID)
	fmt.Printf("Account balance: $%.2f\n", account.Balance)
	fmt.Printf("Account equity:  $%.2f\n", account.Equity)

	stats, _ := engine.GetStats(userID)
	fmt.Printf("Total trades: %d\n", stats.TotalTrades)
	fmt.Printf("Win rate: %.2f%%\n", stats.WinRate)
}
