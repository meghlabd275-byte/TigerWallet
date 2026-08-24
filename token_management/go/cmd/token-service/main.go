// TigerWallet Token Management Service - token registry with real on-chain
// ERC-20 verification and real market prices (CoinGecko public API).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Token represents a cryptocurrency token
type Token struct {
	ID          string    `json:"id"`
	Address     string    `json:"address"`
	Symbol      string    `json:"symbol"`
	Name        string    `json:"name"`
	Decimals    int       `json:"decimals"`
	ChainID     int       `json:"chain_id"`
	TotalSupply string    `json:"total_supply"`
	IsVerified  bool      `json:"is_verified"`
	IsSpam      bool      `json:"is_spam"`
	Price       float64   `json:"price"`
	MarketCap   float64   `json:"market_cap"`
	Volume24h   float64   `json:"volume_24h"`
	LogoURL     string    `json:"logo_url"`
	Website     string    `json:"website"`
	AddedAt     time.Time `json:"added_at"`
}

// TokenAlert is a user price alert
type TokenAlert struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	TokenID     string     `json:"token_id"`
	Condition   string     `json:"condition"` // above, below
	TargetPrice float64    `json:"target_price"`
	IsActive    bool       `json:"is_active"`
	TriggeredAt *time.Time `json:"triggered_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TokenService is the registry with real chain + price backends.
type TokenService struct {
	mu     sync.RWMutex
	tokens map[string]Token
	alerts map[string][]TokenAlert

	rpcByChain map[int]string
	httpClient *http.Client
}

func NewTokenService() *TokenService {
	rpcByChain := map[int]string{}
	for _, entry := range [][2]string{
		{"1", "ETH_RPC_URL"},
		{"56", "BSC_RPC_URL"},
		{"137", "POLYGON_RPC_URL"},
		{"42161", "ARBITRUM_RPC_URL"},
		{"10", "OPTIMISM_RPC_URL"},
	} {
		var chainID int
		fmt.Sscanf(entry[0], "%d", &chainID)
		if url := os.Getenv(entry[1]); url != "" {
			rpcByChain[chainID] = url
		}
	}
	return &TokenService{
		tokens:     map[string]Token{},
		alerts:     map[string][]TokenAlert{},
		rpcByChain: rpcByChain,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func main() {
	service := NewTokenService()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now().Unix()})
	})

	api := router.Group("/api/v1")
	{
		api.GET("/tokens", service.listTokens)
		api.GET("/tokens/:id", service.getToken)
		api.GET("/tokens/search", service.searchTokens)
		api.POST("/tokens/verify", service.verifyToken)
		api.POST("/alerts", service.createAlert)
		api.GET("/alerts/:user_id", service.listAlerts)
		api.DELETE("/alerts/:id", service.deleteAlert)
		api.POST("/spam/report", service.reportSpam)
		api.GET("/tokens/filtered", service.getFilteredTokens)
	}

	go service.priceUpdateWorker()

	srv := &http.Server{Addr: ":8083", Handler: router}
	go func() {
		log.Println("Starting token service on port 8083")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down token service...")
}

// ---------- on-chain ERC-20 metadata (real eth_call) ----------

func callStringMethod(ctx context.Context, client *ethclient.Client, addr common.Address, sig string) (string, error) {
	selector := crypto.Keccak256([]byte(sig))[:4]
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: selector}, nil)
	if err != nil {
		return "", err
	}
	if len(out) < 64 {
		return "", fmt.Errorf("short response for %s", sig)
	}
	strLen := new(big.Int).SetBytes(out[32:64]).Int64()
	if strLen < 0 || len(out) < 64+int(strLen) {
		return "", fmt.Errorf("bad string length for %s", sig)
	}
	return string(out[64 : 64+strLen]), nil
}

func callUint8Method(ctx context.Context, client *ethclient.Client, addr common.Address, sig string) (int, error) {
	selector := crypto.Keccak256([]byte(sig))[:4]
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: selector}, nil)
	if err != nil {
		return 0, err
	}
	if len(out) < 32 {
		return 0, fmt.Errorf("short response for %s", sig)
	}
	return int(new(big.Int).SetBytes(out).Int64()), nil
}

func callUint256Method(ctx context.Context, client *ethclient.Client, addr common.Address, sig string) (*big.Int, error) {
	selector := crypto.Keccak256([]byte(sig))[:4]
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: selector}, nil)
	if err != nil {
		return nil, err
	}
	if len(out) < 32 {
		return nil, fmt.Errorf("short response for %s", sig)
	}
	return new(big.Int).SetBytes(out), nil
}

// fetchOnChainToken reads real ERC-20 metadata from the contract.
func (s *TokenService) fetchOnChainToken(ctx context.Context, address string, chainID int) (*Token, error) {
	rpcURL, ok := s.rpcByChain[chainID]
	if !ok {
		return nil, fmt.Errorf("no RPC configured for chain %d", chainID)
	}
	if !common.IsHexAddress(address) {
		return nil, fmt.Errorf("invalid contract address")
	}
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial failed: %w", err)
	}
	defer client.Close()

	addr := common.HexToAddress(address)
	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("code lookup failed: %w", err)
	}
	if len(code) == 0 {
		return nil, fmt.Errorf("no contract deployed at %s on chain %d", address, chainID)
	}

	name, err := callStringMethod(ctx, client, addr, "name()")
	if err != nil {
		return nil, fmt.Errorf("name() call failed: %w", err)
	}
	symbol, err := callStringMethod(ctx, client, addr, "symbol()")
	if err != nil {
		return nil, fmt.Errorf("symbol() call failed: %w", err)
	}
	decimals, err := callUint8Method(ctx, client, addr, "decimals()")
	if err != nil {
		return nil, fmt.Errorf("decimals() call failed: %w", err)
	}
	supply, err := callUint256Method(ctx, client, addr, "totalSupply()")
	if err != nil {
		return nil, fmt.Errorf("totalSupply() call failed: %w", err)
	}

	return &Token{
		Address:     addr.Hex(),
		Name:        name,
		Symbol:      symbol,
		Decimals:    decimals,
		ChainID:     chainID,
		TotalSupply: supply.String(),
	}, nil
}

// ---------- real market data (CoinGecko public API) ----------

var coingeckoPlatforms = map[int]string{
	1:     "ethereum",
	56:    "binance-smart-chain",
	137:   "polygon-pos",
	42161: "arbitrum-one",
	10:    "optimistic-ethereum",
}

func (s *TokenService) fetchPriceUSD(chainID int, address string) (price, marketCap, volume float64, err error) {
	platform, ok := coingeckoPlatforms[chainID]
	if !ok {
		return 0, 0, 0, fmt.Errorf("no price platform for chain %d", chainID)
	}
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/token_price/%s?contract_addresses=%s&vs_currencies=usd&include_market_cap=true&include_24hr_vol=true",
		platform, strings.ToLower(address))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0, fmt.Errorf("coingecko returned %s", resp.Status)
	}
	var body map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, 0, 0, err
	}
	entry, ok := body[strings.ToLower(address)]
	if !ok {
		return 0, 0, 0, fmt.Errorf("no market data for %s", address)
	}
	return entry["usd"], entry["usd_market_cap"], entry["usd_24h_vol"], nil
}

// ---------- handlers ----------

func (s *TokenService) listTokens(c *gin.Context) {
	verified := c.Query("verified")
	chainID := c.Query("chain_id")

	s.mu.RLock()
	defer s.mu.RUnlock()
	tokens := []Token{}
	for _, t := range s.tokens {
		if verified == "true" && !t.IsVerified {
			continue
		}
		if chainID != "" {
			id, err := strconv.Atoi(chainID)
			if err == nil && t.ChainID != id {
				continue
			}
		}
		if t.IsSpam {
			continue
		}
		tokens = append(tokens, t)
	}
	c.JSON(http.StatusOK, gin.H{"tokens": tokens, "total": len(tokens)})
}

func (s *TokenService) getToken(c *gin.Context) {
	id := c.Param("id")
	s.mu.RLock()
	token, ok := s.tokens[id]
	s.mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}
	c.JSON(http.StatusOK, token)
}

func (s *TokenService) searchTokens(c *gin.Context) {
	query := strings.ToLower(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query required"})
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := []Token{}
	for _, t := range s.tokens {
		if t.IsSpam {
			continue
		}
		if strings.Contains(strings.ToLower(t.Name), query) || strings.Contains(strings.ToLower(t.Symbol), query) {
			results = append(results, t)
		}
	}
	c.JSON(http.StatusOK, gin.H{"tokens": results, "total": len(results)})
}

// verifyToken registers a token after reading its real metadata on-chain.
func (s *TokenService) verifyToken(c *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
		ChainID int    `json:"chain_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	meta, err := s.fetchOnChainToken(c.Request.Context(), req.Address, req.ChainID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("on-chain verification failed: %v", err)})
		return
	}

	token := *meta
	token.ID = uuid.New().String()
	token.IsVerified = true
	token.AddedAt = time.Now()
	if price, mcap, vol, err := s.fetchPriceUSD(req.ChainID, meta.Address); err == nil {
		token.Price = price
		token.MarketCap = mcap
		token.Volume24h = vol
	}

	s.mu.Lock()
	s.tokens[token.ID] = token
	s.mu.Unlock()
	c.JSON(http.StatusCreated, token)
}

func (s *TokenService) createAlert(c *gin.Context) {
	var alert TokenAlert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if alert.Condition != "above" && alert.Condition != "below" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "condition must be above or below"})
		return
	}
	alert.ID = uuid.New().String()
	alert.CreatedAt = time.Now()
	alert.IsActive = true

	s.mu.Lock()
	s.alerts[alert.UserID] = append(s.alerts[alert.UserID], alert)
	s.mu.Unlock()
	c.JSON(http.StatusCreated, alert)
}

func (s *TokenService) listAlerts(c *gin.Context) {
	userID := c.Param("user_id")
	s.mu.RLock()
	alerts := s.alerts[userID]
	s.mu.RUnlock()
	if alerts == nil {
		alerts = []TokenAlert{}
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (s *TokenService) deleteAlert(c *gin.Context) {
	id := c.Param("id")
	s.mu.Lock()
	defer s.mu.Unlock()
	for userID, alerts := range s.alerts {
		for i, a := range alerts {
			if a.ID == id {
				s.alerts[userID] = append(alerts[:i], alerts[i+1:]...)
				c.JSON(http.StatusOK, gin.H{"message": "Alert deleted"})
				return
			}
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
}

func (s *TokenService) reportSpam(c *gin.Context) {
	var req struct {
		TokenID string `json:"token_id" binding:"required"`
		Reason  string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if token, ok := s.tokens[req.TokenID]; ok {
		token.IsSpam = true
		s.tokens[req.TokenID] = token
		c.JSON(http.StatusOK, gin.H{"message": "Spam reported"})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
}

func (s *TokenService) getFilteredTokens(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tokens := []Token{}
	for _, t := range s.tokens {
		if !t.IsSpam && t.IsVerified {
			tokens = append(tokens, t)
		}
	}
	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// priceUpdateWorker refreshes real market data and fires triggered alerts.
func (s *TokenService) priceUpdateWorker() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		tokens := make([]Token, 0, len(s.tokens))
		for _, t := range s.tokens {
			tokens = append(tokens, t)
		}
		s.mu.RUnlock()

		for _, t := range tokens {
			price, mcap, vol, err := s.fetchPriceUSD(t.ChainID, t.Address)
			if err != nil {
				continue
			}
			s.mu.Lock()
			if current, ok := s.tokens[t.ID]; ok {
				current.Price = price
				current.MarketCap = mcap
				current.Volume24h = vol
				s.tokens[t.ID] = current
			}
			for userID, alerts := range s.alerts {
				for i, a := range alerts {
					if !a.IsActive || a.TokenID != t.ID {
						continue
					}
					triggered := (a.Condition == "above" && price >= a.TargetPrice) ||
						(a.Condition == "below" && price <= a.TargetPrice)
					if triggered {
						now := time.Now()
						alerts[i].IsActive = false
						alerts[i].TriggeredAt = &now
						log.Printf("alert %s triggered for user %s: %s price %f %s %f",
							a.ID, userID, t.Symbol, price, a.Condition, a.TargetPrice)
					}
				}
				s.alerts[userID] = alerts
			}
			s.mu.Unlock()
		}
	}
}
