// TigerWallet Perpetual Trading Service
// Futures and perpetual contract trading

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Config struct {
	Port int
}

var cfg = Config{Port: 8009}

type PerpetualPair struct {
	ID           string  `json:"id"`
	BaseToken   string  `json:"base_token"`
	QuoteToken  string  `json:"quote_token"`
	Chain       string  `json:"chain"`
	IndexPrice  string  `json:"index_price"`
	MarkPrice   string  `json:"mark_price"`
	FundingRate string  `json:"Funding_rate"`
	OpenInterest string `json:"open_interest"`
	Volume24h   string `json:"volume_24h"`
	Longs        string `json:"longs"`
	Shorts       string `json:"shorts"`
	Status       string `json:"status"`
}

type Position struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	PairID      string    `json:"pair_id"`
	Side        string    `json:"side"` // long, short
	Size        string    `json:"size"`
	EntryPrice string    `json:"entry_price"`
	Leverage    int       `json:"leverage"`
	Maintenance string    `json:"maintenance"`
	PnL         string    `json:"pnl"`
	Status      string    `json:"status"` // open, closed, liquidated
	OpenedAt    time.Time `json:"opened_at"`
	ClosedAt   *time.Time `json:"closed_at"`
}

type Order struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	PairID    string    `json:"pair_id"`
	Side      string    `json:"side"`
	Type      string    `json:"type"` // market, limit
	Size      string    `json:"size"`
	Price     string    `json:"price"`
	Status    string    `json:"status"` // pending, filled, cancelled
	FilledSize string   `json:"filled_size"`
	OpenedAt  time.Time `json:"opened_at"`
}

type PerpService struct {
	pairs     map[string]*PerpetualPair
	positions map[string]*Position
	orders    map[string]*Order
}

func NewPerpService() *PerpService {
	ps := &PerpService{
		pairs:     make(map[string]*PerpetualPair),
		positions: make(map[string]*Position),
		orders:    make(map[string]*Order),
	}
	ps.initData()
	return ps
}

func (ps *PerpService) initData() {
	pairs := []*PerpetualPair{
		{ID: "eth-perp", BaseToken: "ETH", QuoteToken: "USDT", Chain: "ethereum", IndexPrice: "2500.00", MarkPrice: "2502.50", FundingRate: "0.01%", OpenInterest: "500M", Volume24h: "1B", Longs: "60%", Shorts: "40%", Status: "active"},
		{ID: "btc-perp", BaseToken: "BTC", QuoteToken: "USDT", Chain: "ethereum", IndexPrice: "45000.00", MarkPrice: "45050.00", FundingRate: "0.01%", OpenInterest: "2B", Volume24h: "5B", Longs: "55%", Shorts: "45%", Status: "active"},
		{ID: "sol-perp", BaseToken: "SOL", QuoteToken: "USDT", Chain: "solana", IndexPrice: "100.00", MarkPrice: "100.50", FundingRate: "0.02%", OpenInterest: "100M", Volume24h: "200M", Longs: "65%", Shorts: "35%", Status: "active"},
	}
	for _, p := range pairs {
		ps.pairs[p.ID] = p
	}
}

func (ps *PerpService) GetPairs(c *gin.Context) {
	pairs := make([]*PerpetualPair, 0)
	for _, p := range ps.pairs {
		pairs = append(pairs, p)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "pairs": pairs})
}

func (ps *PerpService) GetPair(c *gin.Context) {
	id := c.Param("id")
	p, ok := ps.pairs[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "pair": p})
}

func (ps *PerpService) OpenPosition(c *gin.Context) {
	var req struct {
		UserID    string `json:"user_id" binding:"required"`
		PairID    string `json:"pair_id" binding:"required"`
		Side      string `json:"side" binding:"required"`
		Size      string `json:"size" binding:"required"`
		Leverage  int    `json:"leverage" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, ok := ps.pairs[req.PairID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}

	pos := &Position{
		ID:          uuid.New().String(),
		UserID:      req.UserID,
		PairID:      req.PairID,
		Side:        req.Side,
		Size:        req.Size,
		EntryPrice:  pair.MarkPrice,
		Leverage:    req.Leverage,
		Maintenance: "0.5",
		PnL:         "0",
		Status:      "open",
		OpenedAt:    time.Now(),
	}

	ps.positions[pos.ID] = pos

	c.JSON(http.StatusCreated, gin.H{"success": true, "position": pos})
}

func (ps *PerpService) ClosePosition(c *gin.Context) {
	posID := c.Param("id")
	pos, ok := ps.positions[posID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}

	now := time.Now()
	pos.Status = "closed"
	pos.ClosedAt = &now

	c.JSON(http.StatusOK, gin.H{"success": true, "position": pos})
}

func (ps *PerpService) GetPositions(c *gin.Context) {
	userID := c.Param("user_id")
	positions := make([]*Position, 0)
	for _, p := range ps.positions {
		if p.UserID == userID && p.Status == "open" {
			positions = append(positions, p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "positions": positions})
}

func (ps *PerpService) PlaceOrder(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		PairID string `json:"pair_id" binding:"required"`
		Side   string `json:"side" binding:"required"`
		Type   string `json:"type" binding:"required"`
		Size   string `json:"size" binding:"required"`
		Price  string `json:"price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order := &Order{
		ID:         uuid.New().String(),
		UserID:     req.UserID,
		PairID:    req.PairID,
		Side:      req.Side,
		Type:      req.Type,
		Size:      req.Size,
		Price:     req.Price,
		Status:     "filled",
		FilledSize: req.Size,
		OpenedAt:  time.Now(),
	}

	ps.orders[order.ID] = order

	c.JSON(http.StatusCreated, gin.H{"success": true, "order": order})
}

func main() {
	log.Println("TigerWallet Perpetual Trading Service")
	log.Printf("Starting on port %d", cfg.Port)

	ps := NewPerpService()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "perpetual"})
	})

	api := r.Group("/api/v1/perpetual")
	{
		api.GET("/pairs", ps.GetPairs)
		api.GET("/pairs/:id", ps.GetPair)
		api.POST("/position", ps.OpenPosition)
		api.POST("/position/:id/close", ps.ClosePosition)
		api.GET("/users/:user_id/positions", ps.GetPositions)
		api.POST("/order", ps.PlaceOrder)
	}

	log.Printf("Server starting on :%d", cfg.Port)
	r.Run(fmt.Sprintf(":%d", cfg.Port))
}
