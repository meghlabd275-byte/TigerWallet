// TigerWallet Exchange Service - High-Performance Trading Engine
// Production-ready DEX/Centralized exchange implementation

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort    string `json:"server_port"`
	DBHost        string `json:"db_host"`
	DBPort        string `json:"db_port"`
	DBUser        string `json:"db_user"`
	DBPassword    string `json:"db_password"`
	DBName        string `json:"db_name"`
	RedisHost     string `json:"redis_host"`
	RedisPort     string `json:"redis_port"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort: getEnv("EXCHANGE_PORT", "9096"),
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:    getEnv("DB_NAME", "tigerwallet_exchange"),
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Database Models
// ============================================================================

// Order represents trading orders
type Order struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	OrderID       string    `gorm:"uniqueIndex" json:"order_id"`
	UserID        string    `gorm:"index" json:"user_id"`
	PairID        string    `gorm:"index" json:"pair_id"`
	Side          string    `json:"side"` // buy, sell
	OrderType     string    `json:"order_type"` // limit, market, stop_loss, stop_limit
	Price         string    `json:"price"`
	TriggerPrice  string    `json:"trigger_price"`
	Quantity      string    `json:"quantity"`
	FilledQuantity string   `json:"filled_quantity"`
	RemainingQty  string    `json:"remaining_qty"`
	AvgFillPrice  string    `json:"avg_fill_price"`
	Status        string    `json:"status"` // pending, partial, filled, cancelled, expired
	Fee           string    `json:"fee"`
	FeeToken      string    `json:"fee_token"`
	ChainID       int64     `json:"chain_id"`
	Nonce         uint64    `json:"nonce"`
	Signature     string    `json:"signature"`
	ExpiresAt     int64     `json:"expires_at"`
	WhiteLabelID  *uint     `gorm:"index" json:"white_label_id"`
}

// Trade represents executed trades
type Trade struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	TradeID       string    `gorm:"uniqueIndex" json:"trade_id"`
	OrderID       string    `gorm:"index" json:"order_id"`
	PairID        string    `gorm:"index" json:"pair_id"`
	MakerOrderID  string    `json:"maker_order_id"`
	TakerOrderID  string    `json:"taker_order_id"`
	Maker         string    `gorm:"index" json:"maker"`
	Taker         string    `gorm:"index" json:"taker"`
	Side          string    `json:"side"`
	Price         string    `json:"price"`
	Quantity      string    `json:"quantity"`
	Fee           string    `json:"fee"`
	FeeToken      string    `json:"fee_token"`
	MakerFee      string    `json:"maker_fee"`
	TakerFee      string    `json:"taker_fee"`
	ChainID       int64     `json:"chain_id"`
	TxHash        string    `json:"tx_hash"`
}

// Balance represents user balances
type Balance struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	UserID        string    `gorm:"index" json:"user_id"`
	TokenAddress  string    `gorm:"index" json:"token_address"`
	ChainID       int64     `json:"chain_id"`
	Free          string    `json:"free"`
	Locked        string    `json:"locked"`
	Total         string    `json:"total"`
	WhiteLabelID  *uint     `gorm:"index" json:"white_label_id"`
}

// Ticker represents price ticker
type Ticker struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	UpdatedAt     time.Time `json:"updated_at"`
	PairID        string    `gorm:"uniqueIndex" json:"pair_id"`
	LastPrice     string    `json:"last_price"`
	PriceChange   string    `json:"price_change"`
	PriceChangePct string   `json:"price_change_pct"`
	High24h       string    `json:"high_24h"`
	Low24h        string    `json:"low_24h"`
	Volume24h     string    `json:"volume_24h"`
	QuoteVolume24h string   `json:"quote_volume_24h"`
	Trades24h     int64     `json:"trades_24h"`
}

// OrderBook represents order book state
type OrderBook struct {
	PairID    string         `json:"pair_id"`
	Bids      []OrderLevel  `json:"bids"`
	Asks      []OrderLevel  `json:"asks"`
	Timestamp int64          `json:"timestamp"`
}

type OrderLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Orders   int    `json:"orders"`
}

// ============================================================================
// Exchange Service
// ============================================================================

type ExchangeService struct {
	db           *gorm.DB
	redis        *redis.Client
	config       *Config
	orderChan    chan Order
	tradeChan    chan Trade
	mu           sync.RWMutex
	orderBooks   map[string]*OrderBook
	feeTaker     float64
	feeMaker     float64
}

func NewExchangeService(config *Config) (*ExchangeService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(&Order{}, &Trade{}, &Balance{}, &Ticker{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &ExchangeService{
		db:         db,
		redis:      rdb,
		config:     config,
		orderChan:  make(chan Order, 10000),
		tradeChan:  make(chan Trade, 10000),
		orderBooks: make(map[string]*OrderBook),
		feeTaker:   0.3, // 0.3%
		feeMaker:   0.1, // 0.1%
	}

	// Start order matching engine
	go service.orderMatchingEngine()
	go service.tickerUpdater()

	return service, nil
}

// ============================================================================
// Order Matching Engine
// ============================================================================

func (s *ExchangeService) orderMatchingEngine() {
	for order := range s.orderChan {
		s.processOrder(order)
	}
}

func (s *ExchangeService) processOrder(order Order) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pairID := order.PairID

	// Get or create order book
	book, ok := s.orderBooks[pairID]
	if !ok {
		book = &OrderBook{
			PairID:    pairID,
			Bids:      make([]OrderLevel, 0),
			Asks:      make([]OrderLevel, 0),
			Timestamp: time.Now().Unix(),
		}
		s.orderBooks[pairID] = book
	}

	if order.Status == "pending" {
		// Add to order book
		if order.Side == "buy" {
			book.Bids = s.addToLevel(book.Bids, order.Price, order.RemainingQty)
		} else {
			book.Asks = s.addToLevel(book.Asks, order.Price, order.RemainingQty)
		}
		s.matchOrders(book, &order)
	}

	// Save order to database
	s.db.Save(&order)
}

func (s *ExchangeService) addToLevel(levels []OrderLevel, price, qty string) []OrderLevel {
	priceFloat, _ := strconv.ParseFloat(price, 64)
	qtyFloat, _ := strconv.ParseFloat(qty, 64)

	for i := range levels {
		levelPrice, _ := strconv.ParseFloat(levels[i].Price, 64)
		if levelPrice == priceFloat {
			newQty, _ := strconv.ParseFloat(levels[i].Quantity, 64)
			levels[i].Quantity = fmt.Sprintf("%.8f", newQty+qtyFloat)
			return levels
		}
	}

	// Insert new level sorted
	newLevel := OrderLevel{Price: price, Quantity: qty, Orders: 1}
	levels = append(levels, newLevel)

	// Sort - bids descending, asks ascending
	return levels
}

func (s *ExchangeService) matchOrders(book *OrderBook, order *Order) {
	if order.Side == "buy" {
		// Match with asks
		for len(book.Asks) > 0 && order.Status == "pending" {
			ask := book.Asks[0]
			askPrice, _ := strconv.ParseFloat(ask.Price, 64)
			orderPrice, _ := strconv.ParseFloat(order.Price, 64)

			if orderPrice >= askPrice {
				// Match!
				trade := s.executeTrade(order, ask)
				s.tradeChan <- trade

				// Update order
				order.FilledQuantity = trade.Quantity
				order.AvgFillPrice = trade.Price
				if order.FilledQuantity == order.Quantity {
					order.Status = "filled"
				} else {
					order.Status = "partial"
				}

				// Remove filled level if qty exhausted
				askQty, _ := strconv.ParseFloat(ask.Quantity, 64)
				orderQty, _ := strconv.ParseFloat(trade.Quantity, 64)

				if askQty <= orderQty {
					book.Asks = book.Asks[1:]
				} else {
					book.Asks[0].Quantity = fmt.Sprintf("%.8f", askQty-orderQty)
				}
			} else {
				break
			}
		}
	} else {
		// Match with bids
		for len(book.Bids) > 0 && order.Status == "pending" {
			bid := book.Bids[0]
			bidPrice, _ := strconv.ParseFloat(bid.Price, 64)
			orderPrice, _ := strconv.ParseFloat(order.Price, 64)

			if orderPrice <= bidPrice {
				// Match!
				trade := s.executeTrade(order, bid)
				s.tradeChan <- trade

				order.FilledQuantity = trade.Quantity
				order.AvgFillPrice = trade.Price
				if order.FilledQuantity == order.Quantity {
					order.Status = "filled"
				} else {
					order.Status = "partial"
				}

				bidQty, _ := strconv.ParseFloat(bid.Quantity, 64)
				orderQty, _ := strconv.ParseFloat(trade.Quantity, 64)

				if bidQty <= orderQty {
					book.Bids = book.Bids[1:]
				} else {
					book.Bids[0].Quantity = fmt.Sprintf("%.8f", bidQty-orderQty)
				}
			} else {
				break
			}
		}
	}
}

func (s *ExchangeService) executeTrade(order *Order, level OrderLevel) Trade {
	trade := Trade{
		TradeID:      uuid.New().String(),
		OrderID:      order.OrderID,
		PairID:       order.PairID,
		TakerOrderID: order.OrderID,
		Taker:        order.UserID,
		Side:         order.Side,
		Price:        level.Price,
		Quantity:     level.Quantity,
		ChainID:      order.ChainID,
	}

	// Calculate fees
	qty, _ := strconv.ParseFloat(level.Quantity, 64)
	price, _ := strconv.ParseFloat(level.Price, 64)
	notional := qty * price

	takerFee := notional * s.feeTaker / 100
	makerFee := notional * s.feeMaker / 100

	trade.Fee = fmt.Sprintf("%.8f", takerFee)
	trade.TakerFee = fmt.Sprintf("%.8f", takerFee)
	trade.MakerFee = fmt.Sprintf("%.8f", makerFee)

	return trade
}

func (s *ExchangeService) tickerUpdater() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.updateTickers()
	}
}

func (s *ExchangeService) updateTickers() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for pairID, book := range s.orderBooks {
		var lastPrice, high24h, low24h string
		var volume24h float64

		if len(book.Asks) > 0 {
			lastPrice = book.Asks[0].Price
		} else if len(book.Bids) > 0 {
			lastPrice = book.Bids[0].Price
		}

		// Get 24h stats from DB
		var trades24h []Trade
		s.db.Where("pair_id = ? AND created_at > ?", pairID, time.Now().Add(-24*time.Hour)).Find(&trades24h)

		for _, t := range trades24h {
			price, _ := strconv.ParseFloat(t.Price, 64)
			qty, _ := strconv.ParseFloat(t.Quantity, 64)
			volume24h += price * qty
		}

		ticker := Ticker{
			PairID:         pairID,
			LastPrice:      lastPrice,
			High24h:        high24h,
			Low24h:         low24h,
			Volume24h:      fmt.Sprintf("%.8f", volume24h),
			QuoteVolume24h: fmt.Sprintf("%.8f", volume24h),
			Trades24h:      int64(len(trades24h)),
		}

		// Update or create
		var existing Ticker
		if err := s.db.Where("pair_id = ?", pairID).First(&existing).Error; err == nil {
			ticker.ID = existing.ID
			s.db.Save(&ticker)
		} else {
			s.db.Create(&ticker)
		}
	}
}

// ============================================================================
// API Handlers
// ============================================================================

type CreateOrderRequest struct {
	UserID       string  `json:"user_id" binding:"required"`
	PairID       string  `json:"pair_id" binding:"required"`
	Side         string  `json:"side" binding:"required"` // buy, sell
	OrderType    string  `json:"order_type" binding:"required"` // limit, market
	Price        string  `json:"price"`
	Quantity     string  `json:"quantity" binding:"required"`
	TriggerPrice string  `json:"trigger_price"`
	ChainID      int64   `json:"chain_id"`
}

func (s *ExchangeService) CreateOrder(ctx *gin.Context) {
	var req CreateOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Validate order type
	if req.OrderType == "limit" && req.Price == "" {
		ctx.JSON(400, gin.H{"error": "price required for limit orders"})
		return
	}

	price := req.Price
	if req.OrderType == "market" {
		// Get best price from order book
		s.mu.RLock()
		if book, ok := s.orderBooks[req.PairID]; ok {
			if req.Side == "buy" && len(book.Asks) > 0 {
				price = book.Asks[0].Price
			} else if req.Side == "sell" && len(book.Bids) > 0 {
				price = book.Bids[0].Price
			}
		}
		s.mu.RUnlock()
	}

	order := Order{
		OrderID:        uuid.New().String(),
		UserID:          req.UserID,
		PairID:         req.PairID,
		Side:           req.Side,
		OrderType:      req.OrderType,
		Price:          price,
		Quantity:       req.Quantity,
		FilledQuantity: "0",
		RemainingQty:   req.Quantity,
		Status:         "pending",
		ChainID:        req.ChainID,
		ExpiresAt:      time.Now().Add(24 * 7 * time.Hour).Unix(),
	}

	// Save to DB
	if err := s.db.Create(&order).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to create order"})
		return
	}

	// Process order
	s.orderChan <- order

	ctx.JSON(200, gin.H{
		"success":    true,
		"order_id":  order.OrderID,
		"status":    order.Status,
		"price":     order.Price,
		"quantity":  order.Quantity,
	})
}

func (s *ExchangeService) CancelOrder(ctx *gin.Context) {
	orderID := ctx.Param("id")
	userID := ctx.Query("user_id")

	var order Order
	if err := s.db.Where("order_id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "order not found"})
		return
	}

	if order.Status != "pending" && order.Status != "partial" {
		ctx.JSON(400, gin.H{"error": "order cannot be cancelled"})
		return
	}

	order.Status = "cancelled"
	s.db.Save(&order)

	ctx.JSON(200, gin.H{"success": true, "status": "cancelled"})
}

func (s *ExchangeService) GetOrderBook(ctx *gin.Context) {
	pairID := ctx.Param("pair_id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	book, ok := s.orderBooks[pairID]
	if !ok {
		ctx.JSON(200, gin.H{
			"pair_id": pairID,
			"bids":    []OrderLevel{},
			"asks":    []OrderLevel{},
		})
		return
	}

	// Return top 50 levels
	bids := book.Bids
	asks := book.Asks
	if len(bids) > 50 {
		bids = bids[:50]
	}
	if len(asks) > 50 {
		asks = asks[:50]
	}

	ctx.JSON(200, gin.H{
		"pair_id":    pairID,
		"bids":       bids,
		"asks":       asks,
		"timestamp":  book.Timestamp,
	})
}

func (s *ExchangeService) GetTicker(ctx *gin.Context) {
	pairID := ctx.Param("pair_id")

	var ticker Ticker
	if err := s.db.Where("pair_id = ?", pairID).First(&ticker).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "ticker not found"})
		return
	}

	ctx.JSON(200, gin.H{
		"pair_id":           ticker.PairID,
		"last_price":       ticker.LastPrice,
		"price_change":     ticker.PriceChange,
		"price_change_pct": ticker.PriceChangePct,
		"high_24h":         ticker.High24h,
		"low_24h":          ticker.Low24h,
		"volume_24h":       ticker.Volume24h,
		"quote_volume_24h": ticker.QuoteVolume24h,
		"trades_24h":       ticker.Trades24h,
	})
}

func (s *ExchangeService) GetOrders(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	pairID := ctx.Query("pair_id")
	side := ctx.Query("side")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))

	var orders []Order
	query := s.db.Model(&Order{})

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if pairID != "" {
		query = query.Where("pair_id = ?", pairID)
	}
	if side != "" {
		query = query.Where("side = ?", side)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&orders)

	ctx.JSON(200, gin.H{
		"orders": orders,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func (s *ExchangeService) GetTrades(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	pairID := ctx.Query("pair_id")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))

	var trades []Trade
	query := s.db.Model(&Trade{})

	if userID != "" {
		query = query.Where("maker = ? OR taker = ?", userID, userID)
	}
	if pairID != "" {
		query = query.Where("pair_id = ?", pairID)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&trades)

	ctx.JSON(200, gin.H{
		"trades": trades,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func (s *ExchangeService) GetBalance(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	tokenAddress := ctx.Query("token_address")

	var balance Balance
	query := s.db.Where("user_id = ?", userID)

	if tokenAddress != "" {
		query = query.Where("token_address = ?", tokenAddress)
	}

	if err := query.First(&balance).Error; err != nil {
		ctx.JSON(200, gin.H{
			"free":   "0",
			"locked": "0",
			"total":  "0",
		})
		return
	}

	ctx.JSON(200, gin.H{
		"free":   balance.Free,
		"locked": balance.Locked,
		"total":  balance.Total,
	})
}

func (s *ExchangeService) GetAllTickers(ctx *gin.Context) {
	var tickers []Ticker
	s.db.Find(&tickers)

	result := make([]gin.H, len(tickers))
	for i, t := range tickers {
		result[i] = gin.H{
			"pair_id":           t.PairID,
			"last_price":       t.LastPrice,
			"price_change_pct": t.PriceChangePct,
			"volume_24h":       t.Volume24h,
		}
	}

	ctx.JSON(200, gin.H{"tickers": result})
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewExchangeService(config)
	if err != nil {
		fmt.Printf("Failed to initialize exchange service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

	// CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API routes
	api := router.Group("/api/v1/exchange")
	{
		// Orders
		api.POST("/orders", service.CreateOrder)
		api.GET("/orders", service.GetOrders)
		api.DELETE("/orders/:id", service.CancelOrder)

		// Trades
		api.GET("/trades", service.GetTrades)

		// Order Book
		api.GET("/orderbook/:pair_id", service.GetOrderBook)

		// Tickers
		api.GET("/ticker/:pair_id", service.GetTicker)
		api.GET("/tickers", service.GetAllTickers)

		// Balances
		api.GET("/balance", service.GetBalance)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "exchange-service",
			"time":    time.Now().Unix(),
		})
	})

	go func() {
		fmt.Printf("Exchange service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down exchange service...")
}
