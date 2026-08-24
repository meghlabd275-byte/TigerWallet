// TigerWallet Cross-Chain Aggregator
//
// Real implementation backed by the LI.FI public API (https://li.quest).
// Quotes, routes, chains and tokens come from LI.FI in real time.
// /execute returns the unsigned transaction request for the user wallet to
// sign and broadcast - this service never holds keys and never fabricates
// transaction hashes.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port        string
	LifiAPIURL  string
	DatabaseURL string
	APIKey      string // optional LI.FI API key
}

func loadConfig() *Config {
	return &Config{
		Port:        getEnv("CROSS_CHAIN_PORT", "8092"),
		LifiAPIURL:  getEnv("LIFI_API_URL", "https://li.quest/v1"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		APIKey:      getEnv("LIFI_API_KEY", ""),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ============================================================================
// Persistence (quote history)
// ============================================================================

type QuoteRecord struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	FromChain  int       `json:"from_chain"`
	ToChain    int       `json:"to_chain"`
	FromToken  string    `json:"from_token"`
	ToToken    string    `json:"to_token"`
	FromAmount string    `json:"from_amount"`
	FromAddr   string    `json:"from_address"`
	Tool       string    `json:"tool"`
	ToAmount   string    `json:"to_amount"`
	RawQuote   string    `gorm:"type:text" json:"-"`
}

// ============================================================================
// Service
// ============================================================================

type CrossChainService struct {
	cfg    *Config
	db     *gorm.DB
	client *http.Client
}

func NewCrossChainService(cfg *Config) (*CrossChainService, error) {
	s := &CrossChainService{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
	if cfg.DatabaseURL != "" {
		db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("database connection failed: %w", err)
		}
		if err := db.AutoMigrate(&QuoteRecord{}); err != nil {
			return nil, fmt.Errorf("migration failed: %w", err)
		}
		s.db = db
	} else {
		log.Println("DATABASE_URL not set: quote history persistence disabled")
	}
	return s, nil
}

// lifiGet performs a real GET against the LI.FI API.
func (s *CrossChainService) lifiGet(ctx context.Context, path string, query url.Values) (json.RawMessage, int, error) {
	u := s.cfg.LifiAPIURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	if s.cfg.APIKey != "" {
		req.Header.Set("x-lifi-api-key", s.cfg.APIKey)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("lifi request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("lifi returned %s: %s", resp.Status, string(body))
	}
	return json.RawMessage(body), resp.StatusCode, nil
}

// ============================================================================
// Handlers
// ============================================================================

// GetChains proxies the real LI.FI chain list.
func (s *CrossChainService) GetChains(c *gin.Context) {
	data, _, err := s.lifiGet(c.Request.Context(), "/chains", nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// GetTools proxies real bridge/DEX tool metadata from LI.FI.
func (s *CrossChainService) GetTools(c *gin.Context) {
	data, _, err := s.lifiGet(c.Request.Context(), "/tools", nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// GetBridges returns the bridge subset of LI.FI tools.
func (s *CrossChainService) GetBridges(c *gin.Context) {
	data, _, err := s.lifiGet(c.Request.Context(), "/tools", nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	var tools struct {
		Bridges []json.RawMessage `json:"bridges"`
	}
	if err := json.Unmarshal(data, &tools); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to parse tools response"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bridges": tools.Bridges})
}

// GetDexes returns the exchange subset of LI.FI tools.
func (s *CrossChainService) GetDexes(c *gin.Context) {
	data, _, err := s.lifiGet(c.Request.Context(), "/tools", nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	var tools struct {
		Exchanges []json.RawMessage `json:"exchanges"`
	}
	if err := json.Unmarshal(data, &tools); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to parse tools response"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dexes": tools.Exchanges})
}

// GetTokens proxies the real LI.FI token list, optionally per chain.
func (s *CrossChainService) GetTokens(c *gin.Context) {
	q := url.Values{}
	if chains := c.Query("chains"); chains != "" {
		q.Set("chains", chains)
	}
	data, _, err := s.lifiGet(c.Request.Context(), "/tokens", q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

type QuoteParams struct {
	FromChain   int    `json:"from_chain" binding:"required"`
	ToChain     int    `json:"to_chain" binding:"required"`
	FromToken   string `json:"from_token" binding:"required"`
	ToToken     string `json:"to_token" binding:"required"`
	FromAmount  string `json:"from_amount" binding:"required"`
	FromAddress string `json:"from_address" binding:"required"`
	ToAddress   string `json:"to_address"`
	Slippage    string `json:"slippage"`
}

// GetQuote fetches a real cross-chain quote from LI.FI.
func (s *CrossChainService) GetQuote(c *gin.Context) {
	var p QuoteParams
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	q := url.Values{}
	q.Set("fromChain", strconv.Itoa(p.FromChain))
	q.Set("toChain", strconv.Itoa(p.ToChain))
	q.Set("fromToken", p.FromToken)
	q.Set("toToken", p.ToToken)
	q.Set("fromAmount", p.FromAmount)
	q.Set("fromAddress", p.FromAddress)
	if p.ToAddress != "" {
		q.Set("toAddress", p.ToAddress)
	}
	if p.Slippage != "" {
		q.Set("slippage", p.Slippage)
	}

	data, _, err := s.lifiGet(c.Request.Context(), "/quote", q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	if s.db != nil {
		var parsed struct {
			Tool      string `json:"tool"`
			Estimate  struct {
				ToAmount string `json:"toAmount"`
			} `json:"estimate"`
		}
		if json.Unmarshal(data, &parsed) == nil {
			s.db.Create(&QuoteRecord{
				ID:         uuid.New().String(),
				FromChain:  p.FromChain,
				ToChain:    p.ToChain,
				FromToken:  p.FromToken,
				ToToken:    p.ToToken,
				FromAmount: p.FromAmount,
				FromAddr:   p.FromAddress,
				Tool:       parsed.Tool,
				ToAmount:   parsed.Estimate.ToAmount,
				RawQuote:   string(data),
			})
		}
	}

	c.Data(http.StatusOK, "application/json", data)
}

// ExecuteQuote fetches a fresh real quote and returns its unsigned
// transactionRequest for the user wallet to sign and broadcast.
// This service never signs, never holds keys, never fabricates hashes.
func (s *CrossChainService) ExecuteQuote(c *gin.Context) {
	var p QuoteParams
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	q := url.Values{}
	q.Set("fromChain", strconv.Itoa(p.FromChain))
	q.Set("toChain", strconv.Itoa(p.ToChain))
	q.Set("fromToken", p.FromToken)
	q.Set("toToken", p.ToToken)
	q.Set("fromAmount", p.FromAmount)
	q.Set("fromAddress", p.FromAddress)
	if p.ToAddress != "" {
		q.Set("toAddress", p.ToAddress)
	}
	if p.Slippage != "" {
		q.Set("slippage", p.Slippage)
	}

	data, _, err := s.lifiGet(c.Request.Context(), "/quote", q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var quote struct {
		ID                 string          `json:"id"`
		Tool               string          `json:"tool"`
		TransactionRequest json.RawMessage `json:"transactionRequest"`
		Estimate           json.RawMessage `json:"estimate"`
	}
	if err := json.Unmarshal(data, &quote); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to parse quote"})
		return
	}
	if len(quote.TransactionRequest) == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "quote contains no transaction request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":               "ready_to_sign",
		"quote_id":             quote.ID,
		"tool":                 quote.Tool,
		"transaction_request":  quote.TransactionRequest,
		"estimate":             quote.Estimate,
	})
}

// GetHistory returns persisted quote history for an address.
func (s *CrossChainService) GetHistory(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "history storage not configured"})
		return
	}
	addr := c.Param("address")
	var records []QuoteRecord
	if err := s.db.Where("from_addr = ?", addr).Order("created_at DESC").Limit(100).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": records})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := loadConfig()
	svc, err := NewCrossChainService(cfg)
	if err != nil {
		log.Fatalf("failed to start cross-chain aggregator: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "cross-chain-aggregator"})
	})

	api := router.Group("/api/v1")
	{
		api.GET("/chains", svc.GetChains)
		api.GET("/tools", svc.GetTools)
		api.GET("/bridges", svc.GetBridges)
		api.GET("/dexes", svc.GetDexes)
		api.GET("/tokens", svc.GetTokens)
		api.POST("/quote", svc.GetQuote)
		api.POST("/execute", svc.ExecuteQuote)
		api.GET("/history/:address", svc.GetHistory)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: router}
	go func() {
		log.Printf("Cross-chain aggregator listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down cross-chain aggregator")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

