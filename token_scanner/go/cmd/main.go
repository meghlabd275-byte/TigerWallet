package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ============ Configuration ============

type Config struct {
	Port          string
	RedisURL      string
	EthereumRPC   string
	PolygonRPC    string
	BSCRPC        string
	CoingeckoURL  string
	InfuraProject string
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============ Models ============

type TokenType string

const (
	TokenTypeERC20   TokenType = "ERC20"
	TokenTypeERC721  TokenType = "ERC721"
	TokenTypeERC1155 TokenType = "ERC1155"
	TokenTypeSPL     TokenType = "SPL"
)

type TokenStatus string

const (
	StatusVerified   TokenStatus = "verified"
	StatusSuspicious TokenStatus = "suspicious"
	StatusScam       TokenStatus = "scam"
	StatusNew        TokenStatus = "new"
)

type Token struct {
	Address     string      `json:"address"`
	Name        string      `json:"name"`
	Symbol      string      `json:"symbol"`
	Decimals    int         `json:"decimals"`
	Type        TokenType   `json:"type"`
	Chain       string      `json:"chain"`
	TotalSupply string      `json:"totalSupply"`
	Price       float64     `json:"price"`
	MarketCap   float64     `json:"marketCap"`
	Volume24h   float64     `json:"volume24h"`
	Holders     int         `json:"holders"`
	Transfers   int64       `json:"transfers"`
	Status      TokenStatus `json:"status"`
	RiskScore   int         `json:"riskScore"`
	Verified    bool        `json:"verified"`
	Tags        []string    `json:"tags"`
	Socials     Socials     `json:"socials"`
	Contract    Contract    `json:"contract"`
	CreatedAt   int64       `json:"createdAt"`
	UpdatedAt   int64       `json:"updatedAt"`
}

type Socials struct {
	Website  string `json:"website"`
	Twitter  string `json:"twitter"`
	Telegram string `json:"telegram"`
	Discord  string `json:"discord"`
	GitHub   string `json:"github"`
}

type Contract struct {
	Verified        bool     `json:"verified"`
	Proxy           bool     `json:"proxy"`
	Implementations []string `json:"implementations"`
	Compiler        string   `json:"compiler"`
	License         string   `json:"license"`
}

type TokenAnalysis struct {
	TokenAddress      string   `json:"tokenAddress"`
	RiskScore         int      `json:"riskScore"`
	Flags             []string `json:"flags"`
	Warnings          []string `json:"warnings"`
	Honeypot          bool     `json:"honeypot"`
	MintFunction      bool     `json:"mintFunction"`
	PauseFunction     bool     `json:"pauseFunction"`
	Blacklist         bool     `json:"blacklist"`
	AntiWhale         bool     `json:"antiWhale"`
	TradingCooldown   bool     `json:"tradingCooldown"`
	HiddenOwner       bool     `json:"hiddenOwner"`
	CentralizedMinter bool     `json:"centralizedMinter"`
	ExternalCallRisk  bool     `json:"externalCallRisk"`
	RiskLevel         string   `json:"riskLevel"`
	AnalyzedAt        int64    `json:"analyzedAt"`
}

type Airdrop struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	ClaimAmount  string `json:"claimAmount"`
	ClaimURL     string `json:"claimUrl"`
	StartDate    int64  `json:"startDate"`
	EndDate      int64  `json:"endDate"`
	SnapshotDate int64  `json:"snapshotDate"`
	Eligible     bool   `json:"eligible"`
	Claimed      bool   `json:"claimed"`
}

// ============ Token Scanner Service ============

type TokenScannerService struct {
	config *Config
	redis  *redis.Client
	tokens map[string]*Token
}

func NewTokenScannerService(config *Config) (*TokenScannerService, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available: %v", err)
	}

	return &TokenScannerService{
		config: config,
		redis:  redisClient,
		tokens: make(map[string]*Token),
	}, nil
}

func (s *TokenScannerService) Initialize() {
	// Add popular tokens
	s.AddToken(&Token{
		Address:   "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
		Name:      "Wrapped Bitcoin",
		Symbol:    "WBTC",
		Decimals:  8,
		Type:      TokenTypeERC20,
		Chain:     "ethereum",
		Price:     67500.00,
		MarketCap: 4500000000,
		Status:    StatusVerified,
		RiskScore: 0,
		Verified:  true,
		Tags:      []string{"wrapped", "bridge", "asset-backed"},
	})

	s.AddToken(&Token{
		Address:   "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		Name:      "USD Coin",
		Symbol:    "USDC",
		Decimals:  6,
		Type:      TokenTypeERC20,
		Chain:     "ethereum",
		Price:     1.00,
		MarketCap: 42000000000,
		Status:    StatusVerified,
		RiskScore: 0,
		Verified:  true,
		Tags:      []string{"stablecoin", "fiat-backed"},
	})

	s.AddToken(&Token{
		Address:   "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		Name:      "Tether USD",
		Symbol:    "USDT",
		Decimals:  6,
		Type:      TokenTypeERC20,
		Chain:     "ethereum",
		Price:     1.00,
		MarketCap: 95000000000,
		Status:    StatusVerified,
		RiskScore: 0,
		Verified:  true,
		Tags:      []string{"stablecoin", "fiat-backed"},
	})

	s.AddToken(&Token{
		Address:   "0x7Fc66500c84A76Ad7e9e934DC149D2C5bD80b7d1",
		Name:      "Aave Token",
		Symbol:    "AAVE",
		Decimals:  18,
		Type:      TokenTypeERC20,
		Chain:     "ethereum",
		Price:     250.00,
		MarketCap: 3500000000,
		Status:    StatusVerified,
		RiskScore: 0,
		Verified:  true,
		Tags:      []string{"defi", "lending"},
	})
}

func (s *TokenScannerService) AddToken(token *Token) {
	token.CreatedAt = time.Now().Unix()
	token.UpdatedAt = time.Now().Unix()
	s.tokens[strings.ToLower(token.Address)] = token
}

func (s *TokenScannerService) GetToken(address string) (*Token, bool) {
	token, ok := s.tokens[strings.ToLower(address)]
	return token, ok
}

func (s *TokenScannerService) GetAllTokens() []*Token {
	var tokens []*Token
	for _, token := range s.tokens {
		tokens = append(tokens, token)
	}
	return tokens
}

func (s *TokenScannerService) SearchTokens(query string) []*Token {
	query = strings.ToLower(query)
	var results []*Token
	for _, token := range s.tokens {
		if strings.Contains(strings.ToLower(token.Name), query) ||
			strings.Contains(strings.ToLower(token.Symbol), query) {
			results = append(results, token)
		}
	}
	return results
}

func (s *TokenScannerService) GetTokensByChain(chain string) []*Token {
	var tokens []*Token
	for _, token := range s.tokens {
		if token.Chain == chain {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

func (s *TokenScannerService) GetVerifiedTokens() []*Token {
	var tokens []*Token
	for _, token := range s.tokens {
		if token.Verified {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

// AnalyzeToken inspects the deployed contract for a token and reports what the
// bytecode actually shows.
//
// The result is derived from the contract: an address holding no code is
// reported as such, and a contract that cannot be read is reported as
// unanalyzable rather than as low risk.
func (s *TokenScannerService) AnalyzeToken(address string) (*TokenAnalysis, error) {
	return s.AnalyzeTokenOnChain(context.Background(), address, s.chainFor(address))
}

// AnalyzeTokenOnChain performs the analysis against a specific chain.
func (s *TokenScannerService) AnalyzeTokenOnChain(ctx context.Context, address, chain string) (*TokenAnalysis, error) {
	if !isValidAddress(address) {
		return nil, fmt.Errorf("invalid token address %q", address)
	}
	address = strings.ToLower(address)

	endpoint := s.rpcForChain(chain)
	if endpoint == "" {
		return nil, fmt.Errorf("no RPC endpoint configured for chain %q", chain)
	}

	client := &http.Client{Timeout: rpcTimeout}

	code, err := fetchContractCode(ctx, client, endpoint, address)
	if err != nil {
		return nil, fmt.Errorf("read contract code: %w", err)
	}

	analysis := &TokenAnalysis{
		TokenAddress: address,
		Flags:        []string{},
		Warnings:     []string{},
		AnalyzedAt:   time.Now().Unix(),
	}

	codeInfo := analyzeCode(code)
	if !codeInfo.IsContract {
		// No code at this address: it is an account, not a token. Saying
		// otherwise would let a mistyped address look like a valid token.
		analysis.RiskLevel = "unknown"
		analysis.Flags = append(analysis.Flags, "not_a_contract")
		analysis.Warnings = append(analysis.Warnings, "the address holds no contract code; it is not a token contract")
		return analysis, nil
	}

	caps := codeInfo.Capabilities

	// A token must implement the ERC-20 read surface. Without balanceOf and
	// transfer it is not a fungible token, whatever else it implements.
	if !caps["balance_of"] || !caps["transfer"] {
		analysis.Warnings = append(analysis.Warnings, "the contract does not implement the ERC-20 transfer/balanceOf interface")
		analysis.Flags = append(analysis.Flags, "not_erc20")
	}

	analysis.MintFunction = caps["mint"]
	analysis.PauseFunction = caps["pause"]
	analysis.Blacklist = caps["blacklist"]
	analysis.AntiWhale = caps["anti_whale"]
	analysis.TradingCooldown = caps["cooldown"]
	analysis.HiddenOwner = caps["ownership"] && codeInfo.IsProxy
	analysis.CentralizedMinter = caps["mint"] && !caps["ownership"]
	analysis.ExternalCallRisk = caps["external_call_risk"] || codeInfo.Facts.DelegateCall > 0

	// A proxy can be repointed at new logic, so the behaviour a user relies on
	// can change after they approve it.
	proxy := codeInfo.IsProxy
	analysis.Honeypot = false // static analysis cannot prove a honeypot; see CheckHoneypot

	score := 0
	if analysis.MintFunction {
		score += 15
		analysis.Flags = append(analysis.Flags, "mint_function")
		analysis.Warnings = append(analysis.Warnings, "supply is not fixed: the contract can mint more tokens")
	}
	if analysis.PauseFunction {
		score += 20
		analysis.Flags = append(analysis.Flags, "pause_function")
		analysis.Warnings = append(analysis.Warnings, "transfers can be paused, which can block selling")
	}
	if analysis.Blacklist {
		score += 25
		analysis.Flags = append(analysis.Flags, "blacklist")
		analysis.Warnings = append(analysis.Warnings, "the operator can blacklist individual holders")
	}
	if analysis.AntiWhale {
		score += 10
		analysis.Flags = append(analysis.Flags, "anti_whale")
		analysis.Warnings = append(analysis.Warnings, "maximum transaction or wallet limits are enforced")
	}
	if analysis.TradingCooldown {
		score += 10
		analysis.Flags = append(analysis.Flags, "trading_cooldown")
		analysis.Warnings = append(analysis.Warnings, "a transfer cooldown is enforced")
	}
	if proxy {
		score += 15
		analysis.Flags = append(analysis.Flags, "proxy_contract")
		analysis.Warnings = append(analysis.Warnings, "the contract is upgradeable, so its logic can change after you approve it")
	}
	if analysis.ExternalCallRisk {
		score += 15
		analysis.Flags = append(analysis.Flags, "external_call")
		analysis.Warnings = append(analysis.Warnings, "the contract can execute external calls or arbitrary data")
	}
	if codeInfo.Facts.SelfDestruct > 0 {
		score += 30
		analysis.Flags = append(analysis.Flags, "selfdestruct")
		analysis.Warnings = append(analysis.Warnings, "the contract contains SELFDESTRUCT")
	}
	if caps["trading_switch"] {
		score += 20
		analysis.Flags = append(analysis.Flags, "trading_switch")
		analysis.Warnings = append(analysis.Warnings, "trading can be enabled or disabled by the operator")
	}
	if !caps["balance_of"] || !caps["transfer"] {
		score += 40
	}

	if score > 100 {
		score = 100
	}
	analysis.RiskScore = score
	analysis.RiskLevel = riskLevelForScore(score)

	return analysis, nil
}

// riskLevelForScore maps a 0..100 risk score onto a level.
func riskLevelForScore(score int) string {
	switch {
	case score >= 70:
		return "critical"
	case score >= 45:
		return "high"
	case score >= 20:
		return "medium"
	default:
		return "low"
	}
}

// rpcForChain returns the configured endpoint for a chain.
func (s *TokenScannerService) rpcForChain(chain string) string {
	switch strings.ToLower(strings.TrimSpace(chain)) {
	case "1", "ethereum", "eth":
		return s.config.EthereumRPC
	case "137", "polygon", "matic":
		return s.config.PolygonRPC
	case "56", "bsc", "binance":
		return s.config.BSCRPC
	default:
		return ""
	}
}

// chainFor returns the chain a known token belongs to, defaulting to Ethereum.
func (s *TokenScannerService) chainFor(address string) string {
	if token, ok := s.tokens[strings.ToLower(address)]; ok && token.Chain != "" {
		return token.Chain
	}
	return "1"
}

// CheckHoneypot reports whether the contract can be sold.
//
// A static analysis cannot prove a honeypot: the determination requires
// observing a real buy and sell. This performs the checks that are decidable
// from the contract and reports what remains unverified, rather than asserting
// that no honeypot exists.
func (s *TokenScannerService) CheckHoneypot(address string) (bool, string) {
	analysis, err := s.AnalyzeToken(address)
	if err != nil {
		return false, fmt.Sprintf("could not analyze the contract: %v", err)
	}

	var reasons []string
	if analysis.Flags == nil {
		return false, "no contract code at this address; nothing to check"
	}
	for _, flag := range analysis.Flags {
		switch flag {
		case "blacklist":
			reasons = append(reasons, "holders can be blacklisted")
		case "pause_function":
			reasons = append(reasons, "transfers can be paused")
		case "trading_switch":
			reasons = append(reasons, "trading can be switched off")
		case "not_erc20":
			reasons = append(reasons, "the contract does not implement the ERC-20 transfer interface")
		}
	}

	if len(reasons) > 0 {
		return true, "restrictions that can prevent selling were found: " + strings.Join(reasons, ", ")
	}
	return false, "no sell-restricting capability was found in the contract; a definitive check requires simulating an actual buy and sell"
}

// GetTokenSecurity returns the full security assessment for a token.
func (s *TokenScannerService) GetTokenSecurity(address string) (*TokenAnalysis, error) {
	return s.AnalyzeToken(address)
}

// GetTokenPrice returns the market price for a token.
//
// The price is read from the configured price API. A token whose price is not
// listed returns an error rather than a placeholder value, so a caller cannot
// mistake an unknown price for a real one.
func (s *TokenScannerService) GetTokenPrice(address string) (float64, error) {
	if !isValidAddress(address) {
		return 0, fmt.Errorf("invalid token address %q", address)
	}
	address = strings.ToLower(address)

	if token, ok := s.tokens[address]; ok && token.Price > 0 {
		return token.Price, nil
	}

	endpoint := strings.TrimSpace(s.config.CoingeckoURL)
	if endpoint == "" {
		return 0, fmt.Errorf("no price API is configured (COINGECKO_URL)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	url := fmt.Sprintf("%s/simple/token_price/ethereum?contract_addresses=%s&vs_currencies=usd", strings.TrimRight(endpoint, "/"), address)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: rpcTimeout}).Do(req)
	if err != nil {
		return 0, fmt.Errorf("price API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("price API returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, err
	}

	var decoded map[string]struct {
		USD float64 `json:"usd"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return 0, fmt.Errorf("decode price response: %w", err)
	}
	entry, ok := decoded[address]
	if !ok || entry.USD <= 0 {
		return 0, fmt.Errorf("no price is available for %s", address)
	}
	return entry.USD, nil
}

func (s *TokenScannerService) GetAirdrops(walletAddress string) []*Airdrop {
	// Simulate airdrop detection
	airdrops := []*Airdrop{
		{
			ID:           "arb-drop-1",
			Name:         "Arbitrum Foundation",
			Symbol:       "ARB",
			ClaimAmount:  "500",
			ClaimURL:     "https://arbitrum.foundation",
			StartDate:    time.Now().Add(-7 * 24 * time.Hour).Unix(),
			EndDate:      time.Now().Add(30 * 24 * time.Hour).Unix(),
			SnapshotDate: time.Now().Add(-30 * 24 * time.Hour).Unix(),
			Eligible:     true,
			Claimed:      false,
		},
	}

	return airdrops
}

func (s *TokenScannerService) ScanWallet(walletAddress string) (map[string][]*Token, error) {
	result := make(map[string][]*Token)

	// In production, would scan multiple chains
	// For demo, return some sample tokens
	result["ethereum"] = []*Token{
		{
			Address:  "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			Name:     "USD Coin",
			Symbol:   "USDC",
			Decimals: 6,
			Chain:    "ethereum",
		},
	}

	return result, nil
}

func (s *TokenScannerService) GetTrendingTokens(limit int) []*Token {
	var tokens []*Token
	for _, token := range s.tokens {
		if token.Volume24h > 0 {
			tokens = append(tokens, token)
		}
	}

	// Sort by volume
	for i := 0; i < len(tokens)-1; i++ {
		for j := i + 1; j < len(tokens); j++ {
			if tokens[i].Volume24h < tokens[j].Volume24h {
				tokens[i], tokens[j] = tokens[j], tokens[i]
			}
		}
	}

	if limit > 0 && limit < len(tokens) {
		tokens = tokens[:limit]
	}

	return tokens
}

func (s *TokenScannerService) CalculatePortfolioValue(walletAddress string) (float64, map[string]float64, error) {
	holdings := make(map[string]float64)

	// In production, would query all chains
	holdings["ethereum"] = 15000.50
	holdings["polygon"] = 2500.00
	holdings["bsc"] = 1000.00

	total := 0.0
	for _, value := range holdings {
		total += value
	}

	return total, holdings, nil
}

// ============ Handlers ============

type Handler struct {
	service *TokenScannerService
}

func NewHandler(service *TokenScannerService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetToken(c *gin.Context) {
	address := c.Param("address")

	token, ok := h.service.GetToken(address)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	c.JSON(http.StatusOK, token)
}

func (h *Handler) GetAllTokens(c *gin.Context) {
	tokens := h.service.GetAllTokens()
	c.JSON(http.StatusOK, gin.H{"tokens": tokens, "total": len(tokens)})
}

func (h *Handler) SearchTokens(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	tokens := h.service.SearchTokens(query)
	c.JSON(http.StatusOK, gin.H{"tokens": tokens, "total": len(tokens)})
}

func (h *Handler) GetTokensByChain(c *gin.Context) {
	chain := c.Param("chain")
	tokens := h.service.GetTokensByChain(chain)
	c.JSON(http.StatusOK, gin.H{"tokens": tokens, "total": len(tokens)})
}

func (h *Handler) AnalyzeToken(c *gin.Context) {
	address := c.Param("address")

	analysis, err := h.service.AnalyzeToken(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

func (h *Handler) CheckHoneypot(c *gin.Context) {
	address := c.Param("address")

	isHoneypot, reason := h.service.CheckHoneypot(address)
	c.JSON(http.StatusOK, gin.H{
		"address":    address,
		"isHoneypot": isHoneypot,
		"reason":     reason,
	})
}

func (h *Handler) GetSecurity(c *gin.Context) {
	address := c.Param("address")

	analysis, err := h.service.GetTokenSecurity(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

func (h *Handler) GetAirdrops(c *gin.Context) {
	wallet := c.Param("wallet")

	airdrops := h.service.GetAirdrops(wallet)
	c.JSON(http.StatusOK, gin.H{"airdrops": airdrops})
}

func (h *Handler) ScanWallet(c *gin.Context) {
	wallet := c.Param("wallet")

	tokens, err := h.service.ScanWallet(wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"holdings": tokens})
}

func (h *Handler) GetPortfolio(c *gin.Context) {
	wallet := c.Param("wallet")

	total, holdings, err := h.service.CalculatePortfolioValue(wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wallet":     wallet,
		"totalValue": total,
		"holdings":   holdings,
	})
}

func (h *Handler) GetTrending(c *gin.Context) {
	tokens := h.service.GetTrendingTokens(10)
	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// ============ Main ============

func main() {
	config := &Config{
		Port:         getEnv("PORT", "8080"),
		RedisURL:     getEnv("REDIS_URL", "localhost:6379"),
		EthereumRPC:  getEnv("ETHEREUM_RPC", "https://eth.llamarpc.com"),
		CoingeckoURL: getEnv("COINGECKO_URL", "https://api.coingecko.com/api/v3"),
	}

	service, err := NewTokenScannerService(config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	service.Initialize()
	handler := NewHandler(service)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "tokens": len(service.tokens)})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Token info
		api.GET("/tokens", handler.GetAllTokens)
		api.GET("/tokens/:address", handler.GetToken)
		api.GET("/tokens/search", handler.SearchTokens)
		api.GET("/chains/:chain/tokens", handler.GetTokensByChain)
		api.GET("/trending", handler.GetTrending)

		// Security
		api.GET("/tokens/:address/analyze", handler.AnalyzeToken)
		api.GET("/tokens/:address/honeypot", handler.CheckHoneypot)
		api.GET("/tokens/:address/security", handler.GetSecurity)

		// Wallet
		api.GET("/wallet/:wallet/airdrops", handler.GetAirdrops)
		api.GET("/wallet/:wallet/scan", handler.ScanWallet)
		api.GET("/wallet/:wallet/portfolio", handler.GetPortfolio)
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Token Scanner on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Helper functions
func formatLargeNumber(n *big.Int) string {
	str := n.String()
	if len(str) <= 3 {
		return str
	}

	result := ""
	counter := 0
	for i := len(str) - 1; i >= 0; i-- {
		if counter == 3 {
			result = "," + result
			counter = 0
		}
		result = string(str[i]) + result
		counter++
	}
	return result
}
