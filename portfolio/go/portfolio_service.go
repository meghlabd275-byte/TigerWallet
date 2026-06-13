// Portfolio Analytics Service - Go Implementation
// Real-time portfolio tracking and analytics

package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Configuration
type PortfolioConfig struct {
	ServerPort string `json:"server_port"`
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	RedisHost  string `json:"redis_host"`
	RedisPort  string `json:"redis_port"`
}

// TokenBalance represents a token balance
type TokenBalance struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserAddress string   `gorm:"index" json:"user_address"`
	ChainID    int64     `json:"chain_id"`
	Token     string    `json:"token"`
	Balance   string    `json:"balance"`
	USDValue  string    `json:"usd_value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NFTBalance represents an NFT balance
type NFTBalance struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserAddress    string    `gorm:"index" json:"user_address"`
	ChainID       int64     `json:"chain_id"`
	ContractAddr  string    `json:"contract_address"`
	TokenID      string    `json:"token_id"`
	Name         string    `json:"name"`
	ImageURL     string    `json:"image_url"`
	USDValue     string    `json:"usd_value"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Transaction represents a transaction
type Transaction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserAddress string    `gorm:"index" json:"user_address"`
	TxHash     string    `gorm:"uniqueIndex" json:"tx_hash"`
	ChainID    int64     `json:"chain_id"`
	Type       string    `json:"type"` // send, receive, swap, stake, unstake
	Token      string    `json:"token"`
	Amount     string    `json:"amount"`
	USDValue   string    `json:"usd_value"`
	From       string    `json:"from"`
	To         string    `json:"to"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
}

// Portfolio represents a portfolio summary
type Portfolio struct {
	UserAddress     string          `json:"user_address"`
	TotalUSDValue string          `json:"total_usd_value"`
	Tokens        []TokenBalance   `json:"tokens"`
	NFTs          []NFTBalance   `json:"nfts"`
	Transactions  []Transaction  `json:"transactions"`
	History       []PortfolioSnap `json:"history"`
}

// PortfolioSnap represents a historical snapshot
type PortfolioSnap struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserAddress string    `gorm:"index" json:"user_address"`
	TotalValue string    `json:"total_value"`
	Timestamp  time.Time `json:"timestamp"`
}

// PortfolioService
type PortfolioService struct {
	db      *gorm.DB
	redis  *redis.Client
	config PortfolioConfig
}

// NewPortfolioService creates new service
func NewPortfolioService(cfg PortfolioConfig) (*PortfolioService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&TokenBalance{}, &NFTBalance{}, &Transaction{}, &PortfolioSnap{})
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &PortfolioService{
		db:      db,
		redis:  rdb,
		config: cfg,
	}, nil
}

// GetPortfolio gets full portfolio
func (s *PortfolioService) GetPortfolio(userAddress string) (*Portfolio, error) {
	var tokens []TokenBalance
	s.db.Where("user_address = ?", userAddress).Find(&tokens)

	var nfts []NFTBalance
	s.db.Where("user_address = ?", userAddress).Find(&nfts)

	var txs []Transaction
	s.db.Where("user_address = ?", userAddress).Order("timestamp DESC").Limit(50).Find(&txs)

	var history []PortfolioSnap
	s.db.Where("user_address = ?", userAddress).Order("timestamp DESC").Limit(30).Find(&history)

	totalUSD := "0"
	for _, t := range tokens {
		totalUSD = addStrings(totalUSD, t.USDValue)
	}
	for _, n := range nfts {
		totalUSD = addStrings(totalUSD, n.USDValue)
	}

	return &Portfolio{
		UserAddress:     userAddress,
		TotalUSDValue: totalUSD,
		Tokens:        tokens,
		NFTs:          nfts,
		Transactions:  txs,
		History:       history,
	}, nil
}

// GetTokenBalances gets token balances
func (s *PortfolioService) GetTokenBalances(userAddress string) ([]TokenBalance, error) {
	var balances []TokenBalance
	err := s.db.Where("user_address = ?", userAddress).Find(&balances).Error
	return balances, err
}

// GetNFTBalances gets NFT balances
func (s *PortfolioService) GetNFTBalances(userAddress string) ([]NFTBalance, error) {
	var balances []NFTBalance
	err := s.db.Where("user_address = ?", userAddress).Find(&balances).Error
	return balances, err
}

// GetTransactions gets transactions
func (s *PortfolioService) GetTransactions(userAddress string, limit int) ([]Transaction, error) {
	var txs []Transaction
	if limit == 0 {
		limit = 50
	}
	err := s.db.Where("user_address = ?", userAddress).Order("timestamp DESC").Limit(limit).Find(&txs).Error
	return txs, err
}

// GetPortfolioHistory gets portfolio history
func (s *PortfolioService) GetPortfolioHistory(userAddress string, days int) ([]PortfolioSnap, error) {
	if days == 0 {
		days = 30
	}

	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var history []PortfolioSnap
	err := s.db.Where("user_address = ? AND timestamp >= ?", userAddress, since).Order("timestamp ASC").Find(&history).Error
	return history, err
}

// UpdateBalance updates a token balance
func (s *PortfolioService) UpdateBalance(userAddress string, chainID int64, token, balance, usdValue string) error {
	result := s.db.Model(&TokenBalance{}).Where("user_address = ? AND chain_id = ? AND token = ?", userAddress, chainID, token).
		Updates(map[string]interface{}{"balance": balance, "usd_value": usdValue, "updated_at": time.Now()})

	if result.RowsAffected == 0 {
		bal := TokenBalance{
			UserAddress: userAddress,
			ChainID:    chainID,
			Token:     token,
			Balance:  balance,
			USDValue: usdValue,
			UpdatedAt: time.Now(),
		}
		s.db.Create(&bal)
	}

	return nil
}

// AddTransaction adds a transaction
func (s *PortfolioService) AddTransaction(tx Transaction) error {
	s.db.Create(&tx)
	return nil
}

// RecordSnapshot records a portfolio snapshot
func (s *PortfolioService) RecordSnapshot(userAddress, totalValue string) error {
	snap := PortfolioSnap{
		UserAddress: userAddress,
		TotalValue:  totalValue,
		Timestamp:  time.Now(),
	}
	s.db.Create(&snap)
	return nil
}

// GetAnalytics gets portfolio analytics
func (s *PortfolioService) GetAnalytics(userAddress string) (map[string]interface{}, error) {
	portfolio, err := s.GetPortfolio(userAddress)
	if err != nil {
		return nil, err
	}

	// Calculate analytics
	totalValue := parseAmount(portfolio.TotalUSDValue)
	assetCount := len(portfolio.Tokens) + len(portfolio.NFTs)

	// Calculate allocation
	allocation := make(map[string]float64)
	for _, t := range portfolio.Tokens {
		tokenValue := parseAmount(t.USDValue)
		if totalValue > 0 {
			allocation[t.Token] = (tokenValue / totalValue) * 100
		}
	}

	// Calculate changes (simplified)
	dayChange := totalValue * 0.02 // 2% simulated
	weekChange := totalValue * 0.05
	monthChange := totalValue * 0.10

	return map[string]interface{}{
		"total_value":     portfolio.TotalUSDValue,
		"asset_count":    assetCount,
		"token_count":   len(portfolio.Tokens),
		"nft_count":     len(portfolio.NFTs),
		"allocation":    allocation,
		"day_change":    formatAmount(dayChange),
		"week_change":   formatAmount(weekChange),
		"month_change":  formatAmount(monthChange),
		"day_change_pct": "2.0",
		"week_change_pct": "5.0",
		"month_change_pct": "10.0",
	}, nil
}

// Handlers

func (s *PortfolioService) GetPortfolioHandler(c *gin.Context) {
	address := c.Param("address")
	portfolio, err := s.GetPortfolio(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, portfolio)
}

func (s *PortfolioService) GetTokensHandler(c *gin.Context) {
	address := c.Param("address")
	tokens, err := s.GetTokenBalances(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, tokens)
}

func (s *PortfolioService) GetNFTsHandler(c *gin.Context) {
	address := c.Param("address")
	nfts, err := s.GetNFTBalances(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, nfts)
}

func (s *PortfolioService) GetTransactionsHandler(c *gin.Context) {
	address := c.Param("address")
	limit := parseInt(c.Query("limit"))
	txs, err := s.GetTransactions(address, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, txs)
}

func (s *PortfolioService) GetHistoryHandler(c *gin.Context) {
	address := c.Param("address")
	days := parseInt(c.Query("days"))
	history, err := s.GetPortfolioHistory(address, days)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, history)
}

func (s *PortfolioService) GetAnalyticsHandler(c *gin.Context) {
	address := c.Param("address")
	analytics, err := s.GetAnalytics(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, analytics)
}

// Utility functions

func parseAmount(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func formatAmount(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func addStrings(a, b string) string {
	af := parseAmount(a)
	bf := parseAmount(b)
	return formatAmount(af + bf)
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

// Main

func main() {
	cfg := PortfolioConfig{
		ServerPort: getEnv("PORTFOLIO_SERVER_PORT", "8085"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "portfolio_db"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewPortfolioService(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize service: %v\n", err)
		os.Exit(1)
	}

	r := gin.Default()

	r.GET("/portfolio/:address", service.GetPortfolioHandler)
	r.GET("/portfolio/:address/tokens", service.GetTokensHandler)
	r.GET("/portfolio/:address/nfts", service.GetNFTsHandler)
	r.GET("/portfolio/:address/transactions", service.GetTransactionsHandler)
	r.GET("/portfolio/:address/history", service.GetHistoryHandler)
	r.GET("/portfolio/:address/analytics", service.GetAnalyticsHandler)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	go func() {
		fmt.Printf("Portfolio Service starting on port %s\n", cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}