package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Order status
const (
	OrderStatusPending         = "PENDING"
	OrderStatusOpen            = "OPEN"
	OrderStatusPartiallyFilled = "PARTIALLY_FILLED"
	OrderStatusFilled          = "FILLED"
	OrderStatusCancelled       = "CANCELLED"
	OrderStatusRejected        = "REJECTED"
	OrderStatusExpired         = "EXPIRED"
)

// Order type
const (
	OrderTypeLimit      = "LIMIT"
	OrderTypeMarket     = "MARKET"
	OrderTypeStopMarket = "STOP_MARKET"
	OrderTypeStopLimit  = "STOP_LIMIT"
	OrderTypeTakeProfit = "TAKE_PROFIT"
	OrderTypeTrailing   = "TRAILING_STOP"
)

// Order side
const (
	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"
)

// Time in force
const (
	TimeInForceGTC = "GTC" // Good Till Cancel
	TimeInForceIOC = "IOC" // Immediate or Cancel
	TimeInForceFOK = "FOK" // Fill or Kill
)

// Order represents a trading order
type Order struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	Symbol         string `json:"symbol"`
	Side           string `json:"side"`
	OrderType      string `json:"orderType"`
	Price          string `json:"price"`
	Quantity       string `json:"quantity"`
	FilledQuantity string `json:"filledQuantity"`
	RemainingQty   string `json:"remainingQty"`
	AvgFillPrice   string `json:"avgFillPrice"`
	Status         string `json:"status"`
	ReduceOnly     bool   `json:"reduceOnly"`
	PostOnly       bool   `json:"postOnly"`
	TimeInForce    string `json:"timeInForce"`
	StopPrice      string `json:"stopPrice"`
	Leverage       string `json:"leverage"`
	MarginType     string `json:"marginType"`
	PositionSide   string `json:"positionSide"`
	ClientOrderID  string `json:"clientOrderId"`
	CreatedAt      int64  `json:"createdAt"`
	UpdatedAt      int64  `json:"updatedAt"`
	ExpiresAt      int64  `json:"expiresAt"`
}

// OrderService handles order operations
type OrderService struct {
	mu      sync.RWMutex
	orders  map[string]*Order
	redis   *redis.Client
	orderCh chan *Order

	// positionLimit caps the quantity of open interest a single user may hold.
	// It is operator configuration (PERP_POSITION_LIMIT), not a per-user value.
	positionLimit string
}

// NewOrderService creates a new order service. rdb may be nil, in which case
// collateral reads report zero rather than a default amount.
func NewOrderService(rdb *redis.Client, positionLimit string) *OrderService {
	s := &OrderService{
		orders:        make(map[string]*Order),
		orderCh:       make(chan *Order, 1000),
		redis:         rdb,
		positionLimit: positionLimit,
	}

	go s.processOrders()

	return s
}

// parseDecimal parses a decimal string into a big.Float. It returns false for
// an empty or malformed value so a caller can distinguish "zero" from "absent"
// instead of silently treating a missing value as a usable number.
func parseDecimal(s string) (*big.Float, bool) {
	if s == "" {
		return nil, false
	}
	f, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
	if err != nil {
		return nil, false
	}
	return f, true
}

// formatDecimal renders a big.Float without an exponent and without trailing
// zeroes, so a balance round-trips through parseDecimal unchanged.
func formatDecimal(f *big.Float) string {
	return f.Text('f', -1)
}

func (s *OrderService) processOrders() {
	for range s.orderCh {
		// Process order matching
		// This would connect to the Rust matching engine
	}
}

// CreateOrder creates a new order
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
	order := &Order{
		ID:          uuid.New().String(),
		UserID:      req.UserID,
		Symbol:      req.Symbol,
		Side:        req.Side,
		OrderType:   req.OrderType,
		Price:       req.Price,
		Quantity:    req.Quantity,
		Status:      OrderStatusOpen,
		ReduceOnly:  req.ReduceOnly,
		PostOnly:    req.PostOnly,
		TimeInForce: req.TimeInForce,
		StopPrice:   req.StopPrice,
		Leverage:    req.Leverage,
		MarginType:  req.MarginType,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	// Validate order
	if err := s.validateOrder(order); err != nil {
		order.Status = OrderStatusRejected
		return nil, err
	}

	s.mu.Lock()
	s.orders[order.ID] = order
	s.mu.Unlock()

	// Submit to matching engine
	s.orderCh <- order

	return order, nil
}

// validateOrder validates order parameters
func (s *OrderService) validateOrder(order *Order) error {
	if order.Symbol == "" {
		return fmt.Errorf("symbol required")
	}
	if order.Quantity == "" {
		return fmt.Errorf("quantity required")
	}
	if order.OrderType == OrderTypeLimit && order.Price == "" {
		return fmt.Errorf("price required for limit orders")
	}
	return nil
}

// GetOrder gets an order by ID
func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, ok := s.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order not found")
	}

	return order, nil
}

// CancelOrder cancels an order
func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found")
	}

	if order.Status != OrderStatusOpen && order.Status != OrderStatusPartiallyFilled {
		return fmt.Errorf("order cannot be cancelled")
	}

	order.Status = OrderStatusCancelled
	order.UpdatedAt = time.Now().Unix()

	return nil
}

// GetOrders gets orders for a user
func (s *OrderService) GetOrders(ctx context.Context, userID, symbol string) ([]*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var orders []*Order
	for _, order := range s.orders {
		if order.UserID == userID {
			if symbol == "" || order.Symbol == symbol {
				orders = append(orders, order)
			}
		}
	}

	return orders, nil
}

// GetBalance reports the account balance for a user.
//
// Available funds are the user's free collateral, read from the real store, so
// an account that has deposited nothing reports zero.
//
// This previously returned a constant equity of "10000" and available of
// "10000" for every caller, so a brand-new account with no deposit was shown
// $10,000 of tradable funds.
//
// Margin locked in open positions is not subtracted here; it is reported by
// GetMarginInfo, which reads the positions themselves. Callers that need
// spendable funds net of margin must consult both.
func (s *OrderService) GetBalance(ctx context.Context, userID string) (*AccountBalance, error) {
	if userID == "" {
		return nil, fmt.Errorf("userId required")
	}

	free, err := s.FreeCollateral(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &AccountBalance{
		UserID:        userID,
		TotalEquity:   free,
		Available:     free,
		UsedMargin:    "0",
		UnrealizedPNL: "0",
	}, nil
}

// collateralKey is the Redis hash field holding a user's free collateral.
func collateralKey(userID string) string {
	return "perp:collateral:" + userID
}

// FreeCollateral returns the user's unencumbered collateral as a decimal
// string. The balance lives in Redis, which is the same store the settlement
// path debits, so a deposit made by one replica is visible to the others. With
// no Redis connection or no entry, the user has no collateral and "0" is
// returned rather than a default amount.
func (s *OrderService) FreeCollateral(ctx context.Context, userID string) (string, error) {
	if s.redis == nil {
		return "0", nil
	}

	value, err := s.redis.Get(ctx, collateralKey(userID)).Result()
	if err == redis.Nil {
		return "0", nil
	}
	if err != nil {
		return "", fmt.Errorf("cannot read collateral for %s: %w", userID, err)
	}

	parsed, ok := parseDecimal(value)
	if !ok {
		// A corrupt entry must not be treated as a usable balance.
		return "", fmt.Errorf("stored collateral for %s is not a valid decimal: %q", userID, value)
	}
	if parsed.Sign() < 0 {
		return "", fmt.Errorf("stored collateral for %s is negative: %q", userID, value)
	}

	return formatDecimal(parsed), nil
}

// GetRiskInfo reports the user's risk limits.
//
// The limits are operator configuration, read from the environment, and the
// allowance is what remains of the position limit after the user's open
// interest is subtracted. It previously reported a hardcoded allowance of
// "1000000" and position limit of "1000000" regardless of the user, the
// configuration or the positions actually held.
func (s *OrderService) GetRiskInfo(ctx context.Context, userID string) (*RiskInfo, error) {
	if userID == "" {
		return nil, fmt.Errorf("userId required")
	}

	limit, ok := parseDecimal(s.positionLimit)
	if !ok || limit.Sign() < 0 {
		// Fail closed: an unset or malformed limit allows nothing rather than
		// everything.
		limit = new(big.Float)
	}

	openInterest := new(big.Float)
	for _, order := range s.openOrdersFor(userID) {
		if qty, ok := parseDecimal(order.RemainingQty); ok && qty.Sign() > 0 {
			openInterest.Add(openInterest, qty)
		} else if qty, ok := parseDecimal(order.Quantity); ok && qty.Sign() > 0 {
			openInterest.Add(openInterest, qty)
		}
	}

	remaining := new(big.Float).Sub(limit, openInterest)
	if remaining.Sign() < 0 {
		remaining = new(big.Float)
	}

	return &RiskInfo{
		RiskLevel:          riskLevelFor(limit, remaining),
		AllowanceRemaining: formatDecimal(remaining),
		PositionLimit:      formatDecimal(limit),
		OpenInterest:       formatDecimal(openInterest),
	}, nil
}

// openOrdersFor returns a snapshot of the user's open orders.
func (s *OrderService) openOrdersFor(userID string) []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var open []*Order
	for _, order := range s.orders {
		if order.UserID != userID {
			continue
		}
		if order.Status == OrderStatusOpen || order.Status == OrderStatusPartiallyFilled {
			open = append(open, order)
		}
	}
	return open
}

// riskLevelFor grades a user by how much of their position limit is left.
func riskLevelFor(limit, remaining *big.Float) string {
	if limit.Sign() <= 0 {
		return "BLOCKED"
	}
	used := new(big.Float).Sub(limit, remaining)
	ratio := new(big.Float).Quo(used, limit)

	switch {
	case ratio.Cmp(big.NewFloat(0.9)) >= 0:
		return "CRITICAL"
	case ratio.Cmp(big.NewFloat(0.7)) >= 0:
		return "ELEVATED"
	default:
		return "STANDARD"
	}
}

// OrderToJSON converts order to JSON
func OrderToJSON(order *Order) string {
	data, _ := json.Marshal(order)
	return string(data)
}

// AccountBalance represents account balance
type AccountBalance struct {
	UserID        string `json:"userId"`
	TotalEquity   string `json:"totalEquity"`
	Available     string `json:"available"`
	UsedMargin    string `json:"usedMargin"`
	UnrealizedPNL string `json:"unrealizedPnl"`
}

// RiskInfo represents risk information
type RiskInfo struct {
	RiskLevel          string `json:"riskLevel"`
	AllowanceRemaining string `json:"allowanceRemaining"`
	PositionLimit      string `json:"positionLimit"`
	OpenInterest       string `json:"openInterest"`
}
