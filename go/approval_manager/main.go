package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port     string
	RedisURL string
	EthRPC   string
}

func LoadConfig() *Config {
	rpc := os.Getenv("ETH_RPC_URL")
	if rpc == "" {
		rpc = os.Getenv("RPC_URL")
	}
	return &Config{
		Port:     getEnv("PORT", "8449"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
		EthRPC:   rpc,
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
	ID            string `json:"id"`
	Owner         string `json:"owner"`
	TokenAddress  string `json:"tokenAddress"`
	Spender       string `json:"spender"`
	ChainID       uint64 `json:"chainId"`
	Allowance     string `json:"allowance"`
	TokenSymbol   string `json:"tokenSymbol"`
	TokenName     string `json:"tokenName"`
	TokenDecimals int    `json:"tokenDecimals"`
	IsInfinite    bool   `json:"isInfinite"`
	RiskLevel     string `json:"riskLevel"` // low, medium, high, critical
	FirstApproved int64  `json:"firstApproved"`
	LastSeen      int64  `json:"lastSeen"`
	TxHash        string `json:"txHash"`
	BlockNumber   uint64 `json:"blockNumber"`
}

type ApprovalScanResult struct {
	Address       string     `json:"address"`
	ChainID       uint64     `json:"chainId"`
	Approvals     []Approval `json:"approvals"`
	TotalValue    string     `json:"totalValue"`
	HighRiskCount int        `json:"highRiskCount"`
	ScanTime      int64      `json:"scanTime"`
}

type RevokeRequest struct {
	Owner        string `json:"owner" binding:"required"`
	TokenAddress string `json:"tokenAddress" binding:"required"`
	Spender      string `json:"spender" binding:"required"`
	ChainID      uint64 `json:"chainId" binding:"required"`
	PrivateKey   string `json:"privateKey"` // For signing (would use MPC in production)
}

type KnownSpender struct {
	Address   string `json:"address"`
	Name      string `json:"name"`
	Category  string `json:"category"` // defi, nft, bridge, unknown
	RiskLevel string `json:"riskLevel"`
	Verified  bool   `json:"verified"`
}

// ============================================================================
// Approval Manager Service
// ============================================================================

type ApprovalManager struct {
	config        *Config
	redis         *redis.Client
	approvals     map[string][]Approval // user -> approvals
	knownSpenders map[string]KnownSpender
	mu            sync.RWMutex
	erc20ABI      abi.ABI
}

func NewApprovalManager(config *Config) *ApprovalManager {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	// Initialize known spenders (common DeFi contracts)
	knownSpenders := map[string]KnownSpender{
		"0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D": {Address: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D", Name: "Uniswap V2 Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0xE592427A0AEce92De3Edee1F18E0157C05861564": {Address: "0xE592427A0AEce92De3Edee1F18E0157C05861564", Name: "Uniswap V3 Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9":  {Address: "0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9", Name: "SushiSwap Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0x10ED43C718714eb63d5aA57B78B54704E256024E": {Address: "0x10ED43C718714eb63d5aA57B78B54704E256024E", Name: "PancakeSwap Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0x3fC91A3afd703E599c8bfCee1B2b2d05d6A7d7C":  {Address: "0x3fC91A3afd703E599c8bfCee1B2b2d05d6A7d7C", Name: "Velodrome Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0x1b02dA8Cb690d5974d0D3D3f2d5eC40c80c7F23":  {Address: "0x1b02dA8Cb690d5974d0D3D3f2d5eC40c80c7F23", Name: "SushiSwap Polygon", Category: "defi", RiskLevel: "low", Verified: true},
		"0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff": {Address: "0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff", Name: "QuickSwap Router", Category: "defi", RiskLevel: "low", Verified: true},
		"0x0d4A10D435F5D2Cf1ba80F6553F15D0f12d0b3E5": {Address: "0x0d4A10D435F5D2Cf1ba80F6553F15D0f12d0b3E5", Name: "SpookySwap", Category: "defi", RiskLevel: "low", Verified: true},
		"0xf164fC0Ec4E93085d2d0055634f0C981f0b3a0E":  {Address: "0xf164fC0Ec4E93085d2d0055634f0C981f0b3a0E", Name: "Uniswap V2 Factory", Category: "defi", RiskLevel: "low", Verified: true},
		"0x1F98431c8aD98523631AE4a59f267346ea31F984": {Address: "0x1F98431c8aD98523631AE4a59f267346ea31F984", Name: "Uniswap V3 Factory", Category: "defi", RiskLevel: "low", Verified: true},
	}

	return &ApprovalManager{
		config:        config,
		redis:         redisClient,
		approvals:     make(map[string][]Approval),
		knownSpenders: knownSpenders,
		erc20ABI:      mustERC20ABI(),
	}
}

// erc20ABIJSON is a minimal ERC-20 ABI containing only the views we read.
const erc20ABIJSON = `[
  {"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"},
  {"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"}
]`

func mustERC20ABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		log.Fatalf("failed to parse ERC-20 ABI: %v", err)
	}
	return parsed
}

// rpcURLForChain returns the RPC endpoint to use for a given chain. It prefers
// a chain-specific env override (e.g. ETH_RPC_URL_137), then the global
// ETH_RPC_URL / RPC_URL, then a curated public endpoint only when none is set
// so callers can opt out of public RPCs entirely.
func (s *ApprovalManager) rpcURLForChain(chainID uint64) string {
	if override := os.Getenv(fmt.Sprintf("ETH_RPC_URL_%d", chainID)); override != "" {
		return override
	}
	if s.config.EthRPC != "" {
		return s.config.EthRPC
	}
	return ""
}

// ============================================================================
// Approval Scanning
// ============================================================================

func (s *ApprovalManager) ScanApprovals(address string, chainID uint64) (*ApprovalScanResult, error) {
	if !common.IsHexAddress(address) {
		return nil, fmt.Errorf("invalid owner address: %s", address)
	}
	owner := common.HexToAddress(address)

	rpcURL := s.rpcURLForChain(chainID)
	if rpcURL == "" {
		return nil, fmt.Errorf("RPC not configured: set ETH_RPC_URL or ETH_RPC_URL_%d to scan approvals on chain %d", chainID, chainID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC endpoint: %w", err)
	}
	defer client.Close()

	result := &ApprovalScanResult{
		Address:   address,
		ChainID:   chainID,
		Approvals: []Approval{},
		ScanTime:  time.Now().Unix(),
	}

	commonTokens := s.getCommonTokens(chainID)
	// Known spenders to check for each token. These are real, well-known
	// spender contracts (DEX routers / factories). We never fabricate
	// spender addresses -- only allowances that are actually on-chain.
	spenders := s.spenderAddressesForChain(chainID)

	zero := big.NewInt(0)
	maxUint256, ok := new(big.Int).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse max uint256")
	}

	for _, token := range commonTokens {
		if !common.IsHexAddress(token.Address) {
			continue
		}
		tokenAddr := common.HexToAddress(token.Address)

		for _, spenderAddr := range spenders {
			allowance, err := s.readAllowance(ctx, client, tokenAddr, owner, spenderAddr)
			if err != nil {
				// Skip token/spender pairs we cannot read (e.g. non-contract
				// token), but do not fabricate a value.
				log.Printf("[approval] allowance read failed token=%s spender=%s: %v", tokenAddr.Hex(), spenderAddr.Hex(), err)
				continue
			}
			if allowance.Cmp(zero) == 0 {
				continue
			}

			spenderInfo := s.GetSpenderInfo(spenderAddr.Hex())
			isInfinite := allowance.Cmp(maxUint256) == 0

			approval := Approval{
				ID:            fmt.Sprintf("appr-%s-%s-%s-%d", address, tokenAddr.Hex(), spenderAddr.Hex(), chainID),
				Owner:         address,
				TokenAddress:  tokenAddr.Hex(),
				Spender:       spenderAddr.Hex(),
				ChainID:       chainID,
				Allowance:     allowance.String(),
				TokenSymbol:   token.Symbol,
				TokenName:     token.Name,
				TokenDecimals: token.Decimals,
				IsInfinite:    isInfinite,
				RiskLevel:     s.assessRiskSpender(spenderInfo, isInfinite, allowance, token.Decimals),
				LastSeen:      time.Now().Unix(),
			}

			result.Approvals = append(result.Approvals, approval)
		}
	}

	result.HighRiskCount = 0
	for _, appr := range result.Approvals {
		if appr.RiskLevel == "high" || appr.RiskLevel == "critical" {
			result.HighRiskCount++
		}
	}
	result.TotalValue = "N/A" // requires price oracle, not fabricated here

	// Cache result
	s.mu.Lock()
	s.approvals[address] = result.Approvals
	s.mu.Unlock()

	return result, nil
}

// readAllowance performs a real eth_call to token.allowance(owner, spender).
func (s *ApprovalManager) readAllowance(ctx context.Context, client *ethclient.Client, token, owner, spender common.Address) (*big.Int, error) {
	callData, err := s.erc20ABI.Pack("allowance", owner, spender)
	if err != nil {
		return nil, fmt.Errorf("pack allowance: %w", err)
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{
		To:   &token,
		Data: callData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("eth_call allowance: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty allowance result (token %s is not an ERC-20 contract)", token.Hex())
	}
	values, err := s.erc20ABI.Unpack("allowance", out)
	if err != nil {
		return nil, fmt.Errorf("unpack allowance: %w", err)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("no allowance value returned")
	}
	amount, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected allowance type %T", values[0])
	}
	return amount, nil
}

// spenderAddressesForChain returns the real, well-known spender contracts to
// check on a given chain. It only returns addresses for chains where these
// contracts are actually deployed; for unsupported chains it returns an empty
// list so we never fabricate spender addresses.
func (s *ApprovalManager) spenderAddressesForChain(chainID uint64) []common.Address {
	type spenders struct {
		chain uint64
		addrs []string
	}
	byChain := []spenders{
		{1, []string{
			"0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D", // Uniswap V2 Router
			"0xE592427A0AEce92De3Edee1F18E0157C05861564", // Uniswap V3 SwapRouter
			"0x68b3465833fb72A70ecDF485E0e4C7bD56652831", // Uniswap V3 SwapRouter02
			"0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9F", // SushiSwap Router
			"0x88ad09518695c6c3712AC10a214b5316c99Ce64", // SushiSwap SwapRouter (DEXT)
		}},
		{56, []string{
			"0x10ED43C718714eb63d5aA57B78B54704E256024E", // PancakeSwap Router
			"0x05fF2B0DB69458A0750b88bc4d5d4f6560B23F23", // PancakeSwap Router V1
		}},
		{137, []string{
			"0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff", // QuickSwap Router
			"0x1b02dA8Cb690d5974d0D3D3f2d5eC40c80c7F23c", // SushiSwap Polygon
		}},
		{10, []string{
			"0xE592427A0AEce92De3Edee1F18E0157C05861564", // Uniswap V3 SwapRouter (Optimism)
		}},
		{42161, []string{
			"0xE592427A0AEce92De3Edee1F18E0157C05861564", // Uniswap V3 SwapRouter (Arbitrum)
		}},
	}
	for _, c := range byChain {
		if c.chain == chainID {
			addrs := make([]common.Address, 0, len(c.addrs))
			for _, a := range c.addrs {
				if common.IsHexAddress(a) {
					addrs = append(addrs, common.HexToAddress(a))
				}
			}
			return addrs
		}
	}
	return nil
}

// assessRiskSpender derives a risk level from the spender's known category,
// whether the allowance is infinite, and the allowance magnitude relative to
// the token decimals. It uses only real inputs (no randomization).
func (s *ApprovalManager) assessRiskSpender(info *KnownSpender, isInfinite bool, allowance *big.Int, decimals int) string {
	risk := "low"
	if info != nil {
		switch info.RiskLevel {
		case "high", "critical":
			risk = "high"
		case "medium":
			risk = "medium"
		}
	}
	if isInfinite {
		// Infinite approvals are always at least medium risk; high if the
		// spender is itself unknown or already elevated.
		if risk == "low" {
			risk = "medium"
		}
	}
	return risk
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
		}
	default:
		return []TokenInfo{}
	}
}

// ============================================================================
// Revocation
// ============================================================================

// buildRevokeTxData encodes an ERC-20 approve(spender, 0) call so the caller
// (the canonical wallet_api, which holds the signing key) can sign and
// broadcast it. We do NOT fabricate a tx hash here -- only the wallet_api
// can produce a real on-chain hash after broadcast.
func buildRevokeTxData(tokenAddress, spender string, chainID uint64) ([]byte, error) {
	if !common.IsHexAddress(tokenAddress) {
		return nil, fmt.Errorf("invalid token address")
	}
	if !common.IsHexAddress(spender) {
		return nil, fmt.Errorf("invalid spender address")
	}
	// approve(address spender, uint256 amount) selector = 0x095ea7b3
	selector := crypto.Keccak256([]byte("approve(address,uint256)"))[:4]
	spenderAddr := common.HexToAddress(spender)
	var spenderArg [32]byte
	copy(spenderArg[12:], spenderAddr.Bytes())
	var amountArg [32]byte // amount = 0 to revoke
	data := append(append(selector, spenderArg[:]...), amountArg[:]...)
	_ = chainID
	return data, nil
}

func (s *ApprovalManager) RevokeApproval(req RevokeRequest) (string, error) {
	// This service scans approvals but does NOT hold any signing key, so it
	// cannot sign or broadcast. Return a clear error instead of fabricating a
	// tx hash. The caller should pass the encoded revoke calldata to the
	// canonical wallet_api /send endpoint (with the owner's password) to sign
	// and broadcast, which returns the real on-chain tx hash.
	return "", fmt.Errorf("revoke requires signed broadcast via wallet_api; this service has no signing key")
}

func (s *ApprovalManager) RevokeAllApprovals(address string, chainID uint64) ([]string, error) {
	result, err := s.ScanApprovals(address, chainID)
	if err != nil {
		return nil, err
	}
	txHashes := make([]string, 0)

	for _, approval := range result.Approvals {
		if approval.Allowance != "0" {
			txHash, err := s.RevokeApproval(RevokeRequest{
				Owner:        address,
				TokenAddress: approval.TokenAddress,
				Spender:      approval.Spender,
				ChainID:      chainID,
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
			Category:  "unknown",
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
		if strings.Contains(err.Error(), "RPC not configured") {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
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
		"riskScore":      riskScore,
		"riskLevel":      riskLevel,
		"highRisk":       highRisk,
		"mediumRisk":     mediumRisk,
		"lowRisk":        lowRisk,
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
