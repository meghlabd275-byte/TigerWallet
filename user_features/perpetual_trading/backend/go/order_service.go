package main

import (
	"context"
	"encoding/json"
	"fmt"
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
}

// NewOrderService creates a new order service
func NewOrderService() *OrderService {
	s := &OrderService{
		orders:  make(map[string]*Order),
		orderCh: make(chan *Order, 1000),
	}

	go s.processOrders()

	return s
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

// GetBalance gets user balance
func (s *OrderService) GetBalance(ctx context.Context, userID string) (*AccountBalance, error) {
	// In production, fetch from database
	return &AccountBalance{
		UserID:        userID,
		TotalEquity:   "10000",
		Available:     "10000",
		UsedMargin:    "0",
		UnrealizedPNL: "0",
	}, nil
}

// GetRiskInfo gets user risk information
func (s *OrderService) GetRiskInfo(ctx context.Context, userID string) (*RiskInfo, error) {
	return &RiskInfo{
		RiskLevel:          "STANDARD",
		AllowanceRemaining: "1000000",
		PositionLimit:      "1000000",
		OpenInterest:       "0",
	}, nil
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
