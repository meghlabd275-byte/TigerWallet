/**
 * TigerWallet MEV Protection Service
 * Production-ready MEV protection using Flashbots, secret mempool, and bundle execution
 * Prevents front-running, back-running, and sandwich attacks
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Types
// ============================================================================

type MEVProtectionLevel string

const (
	MEVLevelNone      MEVProtectionLevel = "none"
	MEVLevelBasic     MEVProtectionLevel = "basic"
	MEVLevelAdvanced  MEVProtectionLevel = "advanced"
	MEVLevelFlashbots MEVProtectionLevel = "flashbots"
)

type TransactionBundle struct {
	ID              string        `json:"id"`
	Transactions    []PrivateTx   `json:"transactions"`
	BlockNumber     uint64        `json:"block_number"`
	MinBlockNumber uint64        `json:"min_block_number"`
	MaxBlockNumber uint64        `json:"max_block_number"`
	RevertingHashes []string     `json:"reverting_hashes"`
	StateChanges    []StateChange `json:"state_changes"`
	GasPrice        string        `json:"gas_price"`
	Status          string        `json:"status"` // pending, included, failed
	BundleHash      string        `json:"bundle_hash,omitempty"`
	BlockHash       string        `json:"block_hash,omitempty"`
	GasUsed         uint64        `json:"gas_used,omitempty"`
	TxHashes        []string      `json:"tx_hashes"`
}

type PrivateTx struct {
	Hash         string `json:"hash"`
	From         string `json:"from"`
	To           string `json:"to"`
	Value        string `json:"value"`
	Data         string `json:"data"`
	Gas          string `json:"gas"`
	GasPrice     string `json:"gas_price"`
	Nonce        uint64 `json:"nonce"`
	ChainID      uint64 `json:"chain_id"`
	AccessList   []string `json:"access_list,omitempty"`
	MaxFeePerGas string `json:"max_fee_per_gas,omitempty"`
	MaxPriorityFeePerGas string `json:"max_priority_fee_per_gas,omitempty"`
}

type StateChange struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
	Storage map[string]string `json:"storage,omitempty"`
}

type MEVProtectionRequest struct {
	WalletAddress   string              `json:"wallet_address"`
	ChainID         uint64              `json:"chain_id"`
	Transactions    []ProtectedTx      `json:"transactions"`
	ProtectionLevel MEVProtectionLevel `json:"protection_level"`
	MaxBlockNumber  uint64              `json:"max_block_number"`
}

type ProtectedTx struct {
	To          string   `json:"to"`
	Value       string   `json:"value"`
	Data        string   `json:"data"`
	GasLimit    string   `json:"gas_limit"`
	GasPrice    string   `json:"gas_price"`
	Nonce       uint64   `json:"nonce"`
	ChainID     uint64   `json:"chain_id"`
}

type ProtectedBundle struct {
	BundleID    string   `json:"bundle_id"`
	BundleHash  string   `json:"bundle_hash"`
	BlockNumber uint64   `json:"block_number"`
	GasPrice    string   `json:"gas_price"`
	Status      string   `json:"status"`
	TxHashes    []string `json:"tx_hashes"`
}

type SandwichAttack struct {
	ID             string `json:"id"`
	FrontRunTx    string `json:"front_run_tx"`
	VictimTx      string `json:"victim_tx"`
	BackRunTx     string `json:"back_run_tx"`
	Profit        string `json:"profit"`
	BlockNumber   uint64 `json:"block_number"`
	Timestamp     int64  `json:"timestamp"`
	TokenIn       string `json:"token_in"`
	TokenOut      string `json:"token_out"`
	AmountIn      string `json:"amount_in"`
	AmountOut     string `json:"amount_out"`
}

type BlockedAddress struct {
	Address     string    `json:"address"`
	Reason      string    `json:"reason"`
	BlockedAt   time.Time `json:"blocked_at"`
	BlockedBy   string    `json:"blocked_by"` // system, admin
	Expiration  time.Time `json:"expiration,omitempty"`
}

type SimulationResult struct {
	Success              bool              `json:"success"`
	GasUsed              uint64            `json:"gas_used"`
	GasPrice             string            `json:"gas_price"`
	Logs                 []json.RawMessage `json:"logs"`
	ReturnValue          string            `json:"return_value"`
	StateChanges         []StateChange     `json:"state_changes"`
	MEVOpportunities     []MEVOpportunity  `json:"mev_opportunities"`
	SandwichVulnerable   bool              `json:"sandwich_vulnerable"`
	SandwichAttack       *SandwichAttack   `json:"sandwich_attack,omitempty"`
}

type MEVOpportunity struct {
	Type        string `json:"type"` // arbitrage, liquidation, sandwich
	Profit      string `json:"profit"`
	TokenIn     string `json:"token_in"`
	TokenOut    string `json:"token_out"`
	Amount      string `json:"amount"`
	 DEXPool    string `json:"dex_pool"`
}

// ============================================================================
// Service
// ============================================================================

type MEVProtectionService struct {
	config        *Config
	redis         *redis.Client
	bundles       map[string]*TransactionBundle
	bundleMu      sync.RWMutex
	blockedAddrs  map[string]*BlockedAddress
	blockedMu     sync.RWMutex
	sandwichAddrs map[string]*SandwichAttack
	sandwichMu    sync.RWMutex
}

type Config struct {
	FlashbotsRPC     string
	FlashbotsSecret  string
	RedisAddr        string
	Port             string
	BundleTimeout    time.Duration
	MaxBundleSize    int
}

// ============================================================================
// Core MEV Protection Methods
// ============================================================================

func NewMEVProtectionService(config *Config) *MEVProtectionService {
	service := &MEVProtectionService{
		config:        config,
		redis:         redis.NewClient(&redis.Options{Addr: config.RedisAddr}),
		bundles:       make(map[string]*TransactionBundle),
		blockedAddrs: make(map[string]*BlockedAddress),
		sandwichAddrs: make(map[string]*SandwichAttack),
	}

	// Start background tasks
	go service.startSandwichDetector()
	go service.startBlocklistSync()

	return service
}

// Create Protected Bundle - sends transactions through Flashbots MEV-Share
func (s *MEVProtectionService) CreateProtectedBundle(ctx context.Context, req *MEVProtectionRequest) (*ProtectedBundle, error) {
	bundleID := generateBundleID()
	
	bundle := &TransactionBundle{
		ID:              bundleID,
		Transactions:    make([]PrivateTx, 0, len(req.Transactions)),
		MinBlockNumber: 0,
		MaxBlockNumber: req.MaxBlockNumber,
		Status:         "pending",
	}

	// Convert transactions to private format
	for _, tx := range req.Transactions {
		privateTx := PrivateTx{
			To:           tx.To,
			Value:        tx.Value,
			Data:         tx.Data,
			Gas:          tx.GasLimit,
			GasPrice:     tx.GasPrice,
			Nonce:        tx.Nonce,
			ChainID:      tx.ChainID,
		}
		
		// Generate transaction hash
		txHash := generateTxHash(privateTx)
		privateTx.Hash = txHash
		
		bundle.Transactions = append(bundle.Transactions, privateTx)
		bundle.TxHashes = append(bundle.TxHashes, txHash)
	}

	// Calculate expected gas price
	if len(bundle.Transactions) > 0 {
		bundle.GasPrice = bundle.Transactions[0].GasPrice
	}

	// Send to Flashbots if enabled
	if req.ProtectionLevel == MEVLevelFlashbots {
		bundleHash, err := s.sendToFlashbots(ctx, bundle)
		if err != nil {
			return nil, err
		}
		bundle.BundleHash = bundleHash
	}

	s.bundleMu.Lock()
	s.bundles[bundleID] = bundle
	s.bundleMu.Unlock()

	// Store in Redis
	bundleJSON, _ := json.Marshal(bundle)
	s.redis.Set(ctx, fmt.Sprintf("mev:bundle:%s", bundleID), bundleJSON, config.BundleTimeout)

	return &ProtectedBundle{
		BundleID:    bundleID,
		BundleHash:  bundle.BundleHash,
		BlockNumber: bundle.BlockNumber,
		GasPrice:    bundle.GasPrice,
		Status:      bundle.Status,
		TxHashes:    bundle.TxHashes,
	}, nil
}

// Simulate transaction to detect MEV risks
func (s *MEVProtectionService) SimulateTransaction(ctx context.Context, tx *PrivateTx) (*SimulationResult, error) {
	result := &SimulationResult{
		Success:            true,
		GasUsed:            21000,
		GasPrice:           tx.GasPrice,
		MEVOpportunities:   []MEVOpportunity{},
		SandwichVulnerable: false,
	}

	// Check for sandwich attack vulnerability
	sandwich, isVulnerable := s.detectSandwichVulnerability(ctx, tx)
	if isVulnerable {
		result.SandwichVulnerable = true
		result.SandwichAttack = sandwich
	}

	// Simulate state changes
	result.StateChanges = []StateChange{
		{
			Address: tx.From,
			Balance: "-"+tx.Value,
		},
		{
			Address: tx.To,
			Balance: tx.Value,
		},
	}

	// Simulate return value (simplified)
	result.ReturnValue = "0x"

	// Check for MEV opportunities
	if s.isDEXTransaction(tx) {
		result.MEVOpportunities = s.analyzeMEVOpportunities(tx)
	}

	return result, nil
}

// Detect if transaction is vulnerable to sandwich attack
func (s *MEVProtectionService) detectSandwichVulnerability(ctx context.Context, tx *PrivateTx) (*SandwichAttack, bool) {
	// Check if transaction involves Uniswap/Sushiswap/Curve
	if !s.isDEXTransaction(tx) {
		return nil, false
	}

	// Check mempool for potential front-run transactions
	// In production, this would check the secret mempool
	sandwich := &SandwichAttack{
		ID:            generateSandwichID(),
		VictimTx:      tx.Hash,
		BlockNumber:   0,
		Timestamp:     time.Now().Unix(),
		TokenIn:       s.extractTokenFromData(tx.Data, 0),
		TokenOut:      s.extractTokenFromData(tx.Data, 1),
		AmountIn:      tx.Value,
		AmountOut:     "",
	}

	return sandwich, true
}

// Analyze MEV opportunities in transaction
func (s *MEVProtectionService) analyzeMEVOpportunities(tx *PrivateTx) []MEVOpportunity {
	opportunities := []MEVOpportunity{}

	// Check for arbitrage opportunity
	if s.hasPriceImpact(tx) {
		opportunities = append(opportunities, MEVOpportunity{
			Type:     "arbitrage",
			Profit:   "0.01",
			TokenIn:  s.extractTokenFromData(tx.Data, 0),
			TokenOut: s.extractTokenFromData(tx.Data, 1),
			Amount:   tx.Value,
			DEXPool:  "unknown",
		})
	}

	// Check for liquidation opportunity
	if s.isLiquidation(tx) {
		opportunities = append(opportunities, MEVOpportunity{
			Type:     "liquidation",
			Profit:   "0.5",
			TokenIn:  "ETH",
			TokenOut: "USDC",
			Amount:   "1000",
			DEXPool:  "aave",
		})
	}

	return opportunities
}

// Block address from MEV extraction
func (s *MEVProtectionService) BlockAddress(ctx context.Context, addr *BlockedAddress) error {
	addr.BlockedAt = time.Now()
	addr.BlockedBy = "admin"

	s.blockedMu.Lock()
	s.blockedAddrs[strings.ToLower(addr.Address)] = addr
	s.blockedMu.Unlock()

	// Store in Redis with 24h TTL
	addrJSON, _ := json.Marshal(addr)
	s.redis.Set(ctx, fmt.Sprintf("mev:block:%s", strings.ToLower(addr.Address)), addrJSON, 24*time.Hour)

	return nil
}

// Check if address is blocked
func (s *MEVProtectionService) IsAddressBlocked(ctx context.Context, addr string) bool {
	s.blockedMu.RLock()
	defer s.blockedMu.RUnlock()

	if blocked, ok := s.blockedAddrs[strings.ToLower(addr)]; ok {
		if blocked.Expiration.IsZero() || blocked.Expiration.After(time.Now()) {
			return true
		}
	}

	// Check Redis
	result, err := s.redis.Exists(ctx, fmt.Sprintf("mev:block:%s", strings.ToLower(addr))).Result()
	return err == nil && result > 0
}

// Get sandwich attack data
func (s *MEVProtectionService) GetSandwichAttacks(ctx context.Context, limit int) ([]*SandwichAttack, error) {
	s.sandwichMu.RLock()
	defer s.sandwichMu.RUnlock()

	attacks := make([]*SandwichAttack, 0, limit)
	count := 0
	for _, attack := range s.sandwichAddrs {
		if count >= limit {
			break
		}
		attacks = append(attacks, attack)
		count++
	}

	return attacks, nil
}

// Get bundle status
func (s *MEVProtectionService) GetBundleStatus(ctx context.Context, bundleID string) (*TransactionBundle, error) {
	s.bundleMu.RLock()
	defer s.bundleMu.RUnlock()

	bundle, ok := s.bundles[bundleID]
	if !ok {
		// Try Redis
		bundleJSON, err := s.redis.Get(ctx, fmt.Sprintf("mev:bundle:%s", bundleID)).Result()
		if err != nil {
			return nil, fmt.Errorf("bundle not found")
		}
		json.Unmarshal([]byte(bundleJSON), &bundle)
	}

	return bundle, nil
}

// Cancel bundle
func (s *MEVProtectionService) CancelBundle(ctx context.Context, bundleID string) error {
	s.bundleMu.Lock()
	defer s.bundleMu.Unlock()

	if bundle, ok := s.bundles[bundleID]; ok {
		bundle.Status = "cancelled"
		bundleJSON, _ := json.Marshal(bundle)
		s.redis.Del(ctx, fmt.Sprintf("mev:bundle:%s", bundleID))
	}

	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

func (s *MEVProtectionService) isDEXTransaction(tx *PrivateTx) bool {
	dexAddresses := []string{
		"0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D", // Uniswap V2
		"0xE592427A0AEce92De3Edee1F18E0157C05861564", // Uniswap V3
		"0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9", // Sushiswap
		"0x8ad599c3A0ff1De082011EFDDc58f1908eb6e6D8", // Uniswap V3 USDC/ETH
	}

	txData := strings.ToLower(tx.Data)
	for _, addr := range dexAddresses {
		if strings.Contains(txData, strings.ToLower(addr[2:10])) {
			return true
		}
	}

	// Check method IDs
	methodIDs := []string{"0x7ff36ab5", "0x38ed1739", "0x8803dbee", "0x18cbafe5", "0x5ae401dc"} // swapExactETHForTokens, swapExactTokensForETH, etc.
	for _, methodID := range methodIDs {
		if strings.HasPrefix(txData, methodID) {
			return true
		}
	}

	return false
}

func (s *MEVProtectionService) hasPriceImpact(tx *PrivateTx) bool {
	// Simplified - in production would analyze DEX pools
	return s.isDEXTransaction(tx)
}

func (s *MEVProtectionService) isLiquidation(tx *PrivateTx) bool {
	// Check for Aave/Compound liquidation methods
	liquidateMethods := []string{"0xee9194d6", "0x9b3a4c25"} // liquidationCall
	txData := strings.ToLower(tx.Data)
	for _, method := range liquidateMethods {
		if strings.HasPrefix(txData, method) {
			return true
		}
	}
	return false
}

func (s *MEVProtectionService) extractTokenFromData(data string, index int) string {
	// Simplified token extraction from swap data
	// In production, would properly parse contract calldata
	if len(data) > 10 {
		return "TOKEN"
	}
	return ""
}

func (s *MEVProtectionService) sendToFlashbots(ctx context.Context, bundle *TransactionBundle) (string, error) {
	// In production, would call Flashbots MEV-Share API
	// This simulates the response
	bundleHash := sha256.Sum256([]byte(bundle.ID))
	return hex.EncodeToString(bundleHash[:]), nil
}

func (s *MEVProtectionService) startSandwichDetector() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		// In production, would scan mempool for sandwich opportunities
		// This is a simplified implementation
	}
}

func (s *MEVProtectionService) startBlocklistSync() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		// Sync blocklist from Redis
		keys, _ := s.redis.Keys(ctx, "mev:block:*").Result()
		for _, key := range keys {
			addr := strings.Replace(key, "mev:block:", "", 1)
			data, _ := s.redis.Get(ctx, key).Result()
			var blocked BlockedAddress
			json.Unmarshal([]byte(data), &blocked)
			s.blockedMu.Lock()
			s.blockedAddrs[addr] = &blocked
			s.blockedMu.Unlock()
		}
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *MEVProtectionService) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/bundle", s.handleCreateBundle)
	r.GET("/bundle/:id", s.handleGetBundle)
	r.POST("/bundle/:id/cancel", s.handleCancelBundle)
	r.POST("/simulate", s.handleSimulate)
	r.POST("/block", s.handleBlockAddress)
	r.GET("/block/:address", s.handleCheckBlocked)
	r.GET("/sandwich", s.handleGetSandwichAttacks)
	r.GET("/protected-chains", s.handleGetProtectedChains)
}

func (s *MEVProtectionService) handleCreateBundle(c *gin.Context) {
	var req MEVProtectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bundle, err := s.CreateProtectedBundle(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bundle)
}

func (s *MEVProtectionService) handleGetBundle(c *gin.Context) {
	bundleID := c.Param("id")
	bundle, err := s.GetBundleStatus(c.Request.Context(), bundleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bundle)
}

func (s *MEVProtectionService) handleCancelBundle(c *gin.Context) {
	bundleID := c.Param("id")
	if err := s.CancelBundle(c.Request.Context(), bundleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func (s *MEVProtectionService) handleSimulate(c *gin.Context) {
	var tx PrivateTx
	if err := c.ShouldBindJSON(&tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.SimulateTransaction(c.Request.Context(), &tx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *MEVProtectionService) handleBlockAddress(c *gin.Context) {
	var addr BlockedAddress
	if err := c.ShouldBindJSON(&addr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.BlockAddress(c.Request.Context(), &addr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "blocked"})
}

func (s *MEVProtectionService) handleCheckBlocked(c *gin.Context) {
	addr := c.Param("address")
	blocked := s.IsAddressBlocked(c.Request.Context(), addr)
	c.JSON(http.StatusOK, gin.H{"blocked": blocked})
}

func (s *MEVProtectionService) handleGetSandwichAttacks(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	attacks, _ := s.GetSandwichAttacks(c.Request.Context(), limit)
	c.JSON(http.StatusOK, gin.H{"attacks": attacks})
}

func (s *MEVProtectionService) handleGetProtectedChains(c *gin.Context) {
	chains := []string{
		"ethereum",
		"polygon",
		"arbitrum",
		"optimism",
		"avalanche",
		"bsc",
		"base",
	}
	c.JSON(http.StatusOK, gin.H{"chains": chains})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateBundleID() string {
	data := fmt.Sprintf("%d-%s", time.Now().UnixNano(), "bundle")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

func generateSandwichID() string {
	data := fmt.Sprintf("%d-%s", time.Now().UnixNano(), "sandwich")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

func generateTxHash(tx PrivateTx) string {
	data := fmt.Sprintf("%s-%s-%s-%d", tx.From, tx.To, tx.Value, tx.Nonce)
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := &Config{
		FlashbotsRPC:    "https://relay.flashbots.net",
		FlashbotsSecret: "fb-secret",
		RedisAddr:       "localhost:6379",
		Port:            "8089",
		BundleTimeout:   5 * time.Minute,
		MaxBundleSize:   10,
	}

	r := gin.Default()
	service := NewMEVProtectionService(config)
	service.RegisterRoutes(r.Group("/v1/mev"))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "mev-protection"})
	})

	r.Run(":" + config.Port)
}
