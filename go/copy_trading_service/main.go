// TigerWallet Copy Trading Service
// Social trading platform for following expert traders

package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Config struct {
	Port int
}

var cfg = Config{Port: 8006}

type Trader struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	Address     string  `json:"address"`
	TotalPnl    string  `json:"total_pnl"`
	WinRate     string  `json:"win_rate"`
	Followers   int     `json:"followers"`
	CopyVolume  string  `json:"copy_volume"`
	Verified   bool    `json:"verified"`
	Trades     int     `json:"trades"`
	Rating     float64 `json:"rating"`
	Status     string  `json:"status"`
}

type Follower struct {
	ID           string    `json:"id"`
	TraderID    string    `json:"trader_id"`
	FollowerID   string    `json:"follower_id"`
	Allocated   string    `json:"allocated"`
	CopyRatio   float64   `json:"copy_ratio"`
	StopLoss    string    `json:"stop_loss"`
	TakeProfit  string    `json:"take_profit"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type CopyTrade struct {
	ID          string    `json:"id"`
	MasterTradeID string  `json:"master_trade_id"`
	TraderID   string    `json:"trader_id"`
	FollowerID string    `json:"follower_id"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	Amount     string    `json:"amount"`
	EntryPrice string    `json:"entry_price"`
	ExitPrice  string    `json:"exit_price"`
	Pnl         string    `json:"pnl"`
	Status     string    `json:"status"`
	OpenedAt    time.Time `json:"opened_at"`
	ClosedAt   *time.Time `json:"closed_at"`
}

type CopyTradingService struct {
	traders   map[string]*Trader
	followers map[string]*Follower
	trades   map[string]*CopyTrade
	mu       sync.RWMutex
}

func NewCopyTradingService() *CopyTradingService {
	ct := &CopyTradingService{
		traders:   make(map[string]*Trader),
		followers: make(map[string]*Follower),
		trades:   make(map[string]*CopyTrade),
	}
	ct.initData()
	return ct
}

func (ct *CopyTradingService) initData() {
	traders := []*Trader{
		{ID: "t1", Username: "CryptoKing", Address: "0x123", TotalPnl: "125000", WinRate: "72%", Followers: 5200, CopyVolume: "2500000", Verified: true, Trades: 342, Rating: 4.8, Status: "active"},
		{ID: "t2", Username: "DeFiMaster", Address: "0x456", TotalPnl: "89000", WinRate: "65%", Followers: 2100, CopyVolume: "1200000", Verified: true, Trades: 189, Rating: 4.5, Status: "active"},
		{ID: "t3", Username: "YieldHunter", Address: "0x789", TotalPnl: "45000", WinRate: "58%", Followers: 890, CopyVolume: "450000", Verified: false, Trades: 78, Rating: 4.2, Status: "active"},
	}
	for _, t := range traders {
		ct.traders[t.ID] = t
	}
}

func (ct *CopyTradingService) GetTraders(c *gin.Context) {
	traders := make([]*Trader, 0)
	for _, t := range ct.traders {
		traders = append(traders, t)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "traders": traders})
}

func (ct *CopyTradingService) GetTrader(c *gin.Context) {
	id := c.Param("id")
	t, ok := ct.traders[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "trader": t})
}

func (ct *CopyTradingService) FollowTrader(c *gin.Context) {
	var req struct {
		TraderID  string  `json:"trader_id" binding:"required"`
		FollowerID string  `json:"follower_id" binding:"required"`
		Allocated string  `json:"allocated" binding:"required"`
		CopyRatio float64 `json:"copy_ratio"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, ok := ct.traders[req.TraderID]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}

	f := &Follower{
		ID:         uuid.New().String(),
		TraderID:   req.TraderID,
		FollowerID: req.FollowerID,
		Allocated:  req.Allocated,
		CopyRatio:  req.CopyRatio,
		Status:     "active",
		CreatedAt:  time.Now(),
	}
	ct.followers[f.ID] = f
	ct.traders[req.TraderID].Followers++

	c.JSON(http.StatusCreated, gin.H{"success": true, "follower": f})
}

func (ct *CopyTradingService) GetFollowerTrades(c *gin.Context) {
	followerID := c.Param("follower_id")
	trades := make([]*CopyTrade, 0)
	for _, t := range ct.trades {
		if t.FollowerID == followerID {
			trades = append(trades, t)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "trades": trades})
}

func main() {
	log.Println("TigerWallet Copy Trading Service")
	log.Printf("Starting on port %d", cfg.Port)

	ct := NewCopyTradingService()

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
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "copy-trading"})
	})

	api := r.Group("/api/v1/copytrading")
	{
		api.GET("/traders", ct.GetTraders)
		api.GET("/traders/:id", ct.GetTrader)
		api.POST("/follow", ct.FollowTrader)
		api.GET("/followers/:follower_id/trades", ct.GetFollowerTrades)
	}

	log.Printf("Server starting on :%d", cfg.Port)
	r.Run(fmt.Sprintf(":%d", cfg.Port))
}
