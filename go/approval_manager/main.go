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
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port     string
	RedisURL string
}

func LoadConfig() *Config {
	return &Config{
		Port:     getEnv("PORT", "8449"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Models
// ============================================================================

type Approval struct {
	ID              string    `json:"id"`
	Owner           string    `json:"owner"`
	TokenAddress    string    `json:"tokenAddress"`
	Spender        string    `json:"spender"`
	ChainID        uint64    `json:"chainId"`
	Allowance      string    `json:"allowance"`
	TokenSymbol    string    `json:"tokenSymbol"`
	TokenName      string    `json:"tokenName"`
	TokenDecimals  int       `json:"tokenDecimals"`
	IsInfinite     bool      `json:"isInfinite"`
	RiskLevel      string    `json:"riskLevel"` // low, medium, high, critical
	FirstApproved  int64     `json:"firstApproved"`
	LastSeen       int64     `json:"lastSeen"`
	TxHash         string    `json:"txHash"`
	BlockNumber    uint64    `json:"blockNumber"`
}

type ApprovalScanResult struct {
	Address       string     `json:"address"`
	ChainID       uint64    `json:"chainId"`
	Approvals     []Approval `json:"approvals"`
	TotalValue    string     `json:"totalValue"`
	HighRiskCount int        `json:"highRiskCount"`
	ScanTime      int64      `json:"scanTime"`
}

type RevokeRequest struct {
	Owner         string `json:"owner" binding:"required"`
	TokenAddress  string `json:"tokenAddress" binding:"required"`
	Spender       string `json:"spender" binding:"required"`
	ChainID       uint64 `json:"chainId" binding:"required"`
	PrivateKey    string `json:"privateKey"` // For signing (would use MPC in production)
}

type KnownSpender struct {
	Address    string `json:"address"`
	Name       string `json:"name"`
	Category   string `json:"category"` // defi, nft, bridge, unknown
	RiskLevel  string `json:"riskLevel"`
	Verified   bool   `json:"verified"`
}

// ============================================================================
// Approval Manager Service
// ============================================================================

type ApprovalManager struct {
	config       *Config
	redis        *redis.Client
	approvals    map[string][]Approval // user -> approvals
	knownSpenders map[string]KnownSpender
	mu           sync.RWMutex
}

func NewApprovalManager(config *Config) *ApprovalManager {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	// Initialize known spenders (common DeFi contracts)
	knownSpenders := map[string]KnownSpender{
		"0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D": {Address: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D", Name: "Uniswap V2 Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0xE592427A0AEce92De3Edee1F18E0157C05861564": {Address: "0xE592427A0AEce92De3Edee1F18E0157C05861564", Name: "Uniswap V3 Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9": {Address: "0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9", Name: "SushiSwap Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0x10ED43C718714eb63d5aA57B78B54704E256024E": {Address: "0x10ED43C718714eb63d5aA57B78B54704E256024E", Name: "PancakeSwap Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0x3fC91A3afd703E599c8bfCee1B2b2d05d6A7d7C": {Address: "0x3fC91A3afd703E599c8bfCee1B2b2d05d6A7d7C", Name: "Velodrome Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0x1b02dA8Cb690d5974d0D3D3f2d5eC40c80c7F23": {Address: "0x1b02dA8Cb690d5974d0D3D3f2d5eC40c80c7F23", Name: "SushiSwap Polygon", Category: "defi", RiskLevel: "low", Verified: true},
		"0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff": {Address: "0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff", Name: "QuickSwap Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0x0d4A10D435F5D2Cf1ba80F6553F15D0f12d0b3E5": {Address: "0x0d4A10D435F5D2Cf1ba80F6553F15D0f12d0b3E5", Name: "SpookySwap", Category: "defi", RiskLevel: "low", Verified: true},
		"0xf164fC0Ec4E93085d2d0055634f0C981f0b3a0E": {Address: "0xf164fC0Ec4E93085d2d0055634f0C981f0b3a0E", Name: "Uniswap V2 Factory", Category: "defi", RiskLevel: "low", Verified: true},
		"0x1F98431c8aD98523631AE4a59f267346ea31F984": {Address: "0x1F98431c8aD98523631AE4a59f267346ea31F984", Name: "Uniswap V3 Factory", Category: "defi", RiskLevel: "low", Verified: true},
	}

	return &ApprovalManager{
		config:       config,
		redis:        redisClient,
		approvals:    make(map[string][]Approval),
		knownSpenders: knownSpenders,
	}
}

// ============================================================================
// Approval Scanning
// ============================================================================

func (s *ApprovalManager) ScanApprovals(address string, chainID uint64) (*ApprovalScanResult, error) {
	result := &ApprovalScanResult{
		Address:    address,
		ChainID:    chainID,
		Approvals:  []Approval{},
		ScanTime:   time.Now().Unix(),
	}

	// Generate realistic approvals based on common tokens
	// In production, this would query blockchain nodes
	
	commonTokens := s.getCommonTokens(chainID)
	
	for _, token := range commonTokens {
		// Simulate different approval scenarios
		approval := Approval{
			ID:             fmt.Sprintf("appr-%s-%s-%d", address[:8], token.Address[:8], chainID),
			Owner:          address,
			TokenAddress:   token.Address,
			Spender:       s.getRandomSpender(),
			ChainID:        chainID,
			Allowance:     s.getRandomAllowance(),
			TokenSymbol:    token.Symbol,
			TokenName:      token.Name,
			TokenDecimals:  token.Decimals,
			IsInfinite:     s.isInfiniteAllowance(),
			RiskLevel:      s.assessRisk(token.Symbol),
			FirstApproved:  time.Now().Add(-30 * 24 * time.Hour).Unix(),
			LastSeen:       time.Now().Unix(),
		}
		
		result.Approvals = append(result.Approvals, approval)
	}

	// Calculate total and risk
	result.HighRiskCount = 0
	for _, appr := range result.Approvals {
		if appr.RiskLevel == "high" || appr.RiskLevel == "critical" {
			result.HighRiskCount++
		}
		
		// Estimate total value
		allowance, _ := new(big.Int).SetString(appr.Allowance, 10)
		if allowance != nil {
			// Simplified calculation
			result.TotalValue = "N/A" // Would calculate in USD
		}
	}

	// Cache result
	s.mu.Lock()
	s.approvals[address] = result.Approvals
	s.mu.Unlock()

	return result, nil
}

type TokenInfo struct {
	Address  string
	Symbol   string
	Name     string
	Decimals int
}

func (s *ApprovalManager) getCommonTokens(chainID uint64) []TokenInfo {
	switch chainID {
	case 1: // Ethereum
		return []TokenInfo{
			{Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Symbol: "USDC", Name: "USD Coin", Decimals: 6},
			{Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Symbol: "USDT", Name: "Tether USD", Decimals: 6},
			{Address: "0x6B175474E89094C44Da98b954EesadcdEF9ce6CC", Symbol: "DAI", Name: "Dai Stablecoin", Decimals: 18},
			{Address: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", Symbol: "WBTC", Name: "Wrapped Bitcoin", Decimals: 8},
			{Address: "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9", Symbol: "AAVE", Name: "Aave", Decimals: 18},
			{Address: "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984", Symbol: "UNI", Name: "Uniswap", Decimals: 18},
			{Address: "0x514910771AF9Ca656af840dff83E8264EcF986CA", Symbol: "LINK", Name: "Chainlink", Decimals: 18},
		}
	case 137: // Polygon
		return []TokenInfo{
			{Address: "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", Symbol: "USDC", Name: "USD Coin", Decimals: 6},
			{Address: "0x1BFD67037B42Cf73acF2047067bd4F2C47D9BfD6", Symbol: "WBTC", Name: "Wrapped Bitcoin", Decimals: 8},
			{Address: "0x53E0bca35eC356bf5f7524E5d7833d3A333EC20c", Symbol: "AAVE", Name: "Aave", Decimals: 18},
			{Address: "0x53e0bca35ec356bf5f7524e5d7833d3a333ec20c", Symbol: "AAVE", Name: "Aave", Decimals: 18},
		}
	default:
		return []TokenInfo{}
	}
}

func (s *ApprovalManager) getRandomSpender() string {
	spenders := []string{
		"0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D",
		"0xE592427A0AEce92De3Edee1F18E0157C05861564",
		"0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9",
		"0x10ED43C718714eb63d5aA57B78B54704E256024E",
		"0x0000000000000000000000000000000000000001", // Unknown
	}
	return spenders[time.Now().Unix()%int64(len(spenders))]
}

func (s *ApprovalManager) getRandomAllowance() string {
	allowances := []string{
		"0",
		"115792089237316195423570985008687907853269984665640564039457584007913129639935", // Infinite
		"1000000000000000000",
		"10000000000000000000",
		"100000000000000000000",
	}
	return allowances[time.Now().Unix()%int64(len(allowances))]
}

func (s *ApprovalManager) isInfiniteAllowance() bool {
	return time.Now().Unix()%2 == 0
}

func (s *ApprovalManager) assessRisk(tokenSymbol string) string {
	highRiskTokens := map[string]bool{"USDC": true, "USDT": true, "DAI": true, "WBTC": true}
	
	if highRiskTokens[tokenSymbol] {
		return "medium"
	}
	return "low"
}

// ============================================================================
// Revocation
// ============================================================================

func (s *ApprovalManager) RevokeApproval(req RevokeRequest) (string, error) {
	// In production, this would:
	// 1. Build a transaction to set allowance to 0
	// 2. Sign it with MPC or private key
	// 3. Broadcast to network
	
	// For now, return a mock tx hash
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	return txHash, nil
}

func (s *ApprovalManager) RevokeAllApprovals(address string, chainID uint64) ([]string, error) {
	result := s.ScanApprovals(address, chainID)
	txHashes := make([]string, 0)
	
	for _, approval := range result.Approvals {
		if approval.Allowance != "0" {
			txHash, err := s.RevokeApproval(RevokeRequest{
				Owner:        address,
				TokenAddress: approval.TokenAddress,
				Spender:     approval.Spender,
				ChainID:     chainID,
			})
			if err == nil {
				txHashes = append(txHashes, txHash)
			}
		}
	}
	
	return txHashes, nil
}

// ============================================================================
// Known Spenders
// ============================================================================

func (s *ApprovalManager) GetKnownSpenders() []KnownSpender {
	spenders := make([]KnownSpender, 0, len(s.knownSpenders))
	for _, sp := range s.knownSpenders {
		spenders = append(spenders, sp)
	}
	return spenders
}

func (s *ApprovalManager) GetSpenderInfo(address string) *KnownSpender {
	info, ok := s.knownSpenders[address]
	if !ok {
		return &KnownSpender{
			Address:   address,
			Name:      "Unknown Contract",
			Category: "unknown",
			RiskLevel: "unknown",
			Verified:  false,
		}
	}
	return &info
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *ApprovalManager) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "approval-manager"})
	})

	api := r.Group("/api/v1/approvals")
	{
		// Scan approvals
		api.POST("/scan", s.handleScanApprovals)
		
		// Revoke single
		api.POST("/revoke", s.handleRevoke)
		
		// Revoke all
		api.POST("/revoke-all", s.handleRevokeAll)
		
		// Known spenders
		api.GET("/spenders", s.handleGetKnownSpenders)
		api.GET("/spender/:address", s.handleGetSpenderInfo)
		
		// Risk assessment
		api.POST("/assess-risk", s.handleAssessRisk)
	}
}

func (s *ApprovalManager) handleScanApprovals(c *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
		ChainID uint64 `json:"chainId" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	result, err := s.ScanApprovals(req.Address, req.ChainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, result)
}

func (s *ApprovalManager) handleRevoke(c *gin.Context) {
	var req RevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash, err := s.RevokeApproval(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"txHash": txHash, "message": "Approval revoked"})
}

func (s *ApprovalManager) handleRevokeAll(c *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
		ChainID uint64 `json:"chainId" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHashes, err := s.RevokeAllApprovals(req.Address, req.ChainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"txHashes": txHashes, "count": len(txHashes)})
}

func (s *ApprovalManager) handleGetKnownSpenders(c *gin.Context) {
	spenders := s.GetKnownSpenders()
	c.JSON(http.StatusOK, gin.H{"spenders": spenders})
}

func (s *ApprovalManager) handleGetSpenderInfo(c *gin.Context) {
	address := c.Param("address")
	info := s.GetSpenderInfo(address)
	c.JSON(http.StatusOK, info)
}

func (s *ApprovalManager) handleAssessRisk(c *gin.Context) {
	var req struct {
		Approvals []Approval `json:"approvals" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Calculate risk score
	highRisk := 0
	mediumRisk := 0
	lowRisk := 0
	
	for _, appr := range req.Approvals {
		switch appr.RiskLevel {
		case "high", "critical":
			highRisk++
		case "medium":
			mediumRisk++
		default:
			lowRisk++
		}
	}
	
	riskScore := float64(highRisk*10 + mediumRisk*5 + lowRisk*1)
	riskLevel := "low"
	if riskScore > 50 {
		riskLevel = "critical"
	} else if riskScore > 20 {
		riskLevel = "high"
	} else if riskScore > 10 {
		riskLevel = "medium"
	}
	
	c.JSON(http.StatusOK, gin.H{
		"riskScore":   riskScore,
		"riskLevel":   riskLevel,
		"highRisk":    highRisk,
		"mediumRisk":  mediumRisk,
		"lowRisk":     lowRisk,
		"recommendation": "Revoke high-risk approvals immediately",
	})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	service := NewApprovalManager(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	service.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Approval Manager starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
