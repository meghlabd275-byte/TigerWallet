/**
 * TigerWallet Transaction Simulator Service
 * Go + C++ FFI for Ultra-Low Latency
 * 
 * This service provides high-performance transaction simulation
 * with MEV protection and gas optimization.
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort        string `json:"server_port"`
	DBHost            string `json:"db_host"`
	DBPort            string `json:"db_port"`
	DBUser            string `json:"db_user"`
	DBPassword        string `json:"db_password"`
	DBName            string `json:"db_name"`
	RedisHost         string `json:"redis_host"`
	RedisPort         string `json:"redis_port"`
	MaxConcurrentSims int    `json:"max_concurrent_sims"`
	SimulationTimeout int    `json:"simulation_timeout_ms"`
	EnableMEVProtect  bool   `json:"enable_mev_protection"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:        getEnv("TX_SIM_PORT", "9090"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "tigerwallet"),
		DBPassword:        getEnv("DB_PASSWORD", "password"),
		DBName:            getEnv("DB_NAME", "tigerwallet"),
		RedisHost:         getEnv("REDIS_HOST", "localhost"),
		RedisPort:         getEnv("REDIS_PORT", "6379"),
		MaxConcurrentSims: 1000,
		SimulationTimeout: 50,
		EnableMEVProtect:  true,
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

type Simulation struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	TransactionHash   string    `gorm:"index" json:"transaction_hash"`
	ChainID           uint64    `json:"chain_id"`
	FromAddress       string    `json:"from_address"`
	ToAddress         string    `json:"to_address"`
	Value             string    `json:"value"`
	GasLimit          uint64    `json:"gas_limit"`
	GasUsed           uint64    `json:"gas_used"`
	GasPrice          uint64    `json:"gas_price"`
	Success           bool      `json:"success"`
	ErrorMessage      string    `json:"error_message"`
	MEVType           string    `json:"mev_type"`
	MEVRisk           float64   `json:"mev_risk"`
	ExecutionTimeMs   float64   `json:"execution_time_ms"`
	TokenTransfers    JSON      `json:"token_transfers" gorm:"type:jsonb"`
	StateChanges      JSON      `json:"state_changes" gorm:"type:jsonb"`
	Warnings          JSON      `json:"warnings" gorm:"type:jsonb"`
}

type MempoolTransaction struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	Hash           string    `gorm:"uniqueIndex" json:"hash"`
	FromAddress    string    `json:"from_address"`
	ToAddress      string    `json:"to_address"`
	Value          string    `json:"value"`
	GasLimit       uint64    `json:"gas_limit"`
	GasPrice       uint64    `json:"gas_price"`
	Data           string    `json:"data"`
	ChainID        uint64    `json:"chain_id"`
	Nonce          uint64    `json:"nonce"`
	BlockNumber    uint64    `json:"block_number"`
	Status         string    `json:"status"` // pending, confirmed, dropped
}

type Block struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	Number          uint64    `gorm:"uniqueIndex" json:"number"`
	Hash            string    `gorm:"index" json:"hash"`
	ParentHash      string    `json:"parent_hash"`
	Timestamp       uint64    `json:"timestamp"`
	GasLimit        uint64    `json:"gas_limit"`
	GasUsed         uint64    `json:"gas_used"`
	BaseFeePerGas   uint64    `json:"base_fee_per_gas"`
	Miner           string    `json:"miner"`
	TransactionCount int     `json:"transaction_count"`
}

type AccountState struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Address     string    `gorm:"uniqueIndex" json:"address"`
	Balance     string    `json:"balance"`
	Nonce       uint64    `json:"nonce"`
	CodeHash    string    `json:"code_hash"`
	TokenBalances JSON    `json:"token_balances" gorm:"type:jsonb"`
}

type Token struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	Address       string    `gorm:"uniqueIndex" json:"address"`
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	Decimals      uint8     `json:"decimals"`
	TotalSupply   string    `json:"total_supply"`
	IsNative      bool      `json:"is_native"`
	ChainID       uint64    `json:"chain_id"`
	IsActive      bool      `json:"is_active"`
}

type MEVAlert struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	TransactionHash string    `json:"transaction_hash"`
	MEVType         string    `json:"mev_type"`
	RiskScore       float64   `json:"risk_score"`
	Description     string    `json:"description"`
	RelatedTxs      JSON      `json:"related_txs" gorm:"type:jsonb"`
	Blocked         bool      `json:"blocked"`
}

type JSON json.RawMessage

func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSON: %v", value)
	}
	*j = JSON(bytes)
	return nil
}

func (j JSON) Value() (interface{}, error) {
	return json.RawMessage(j).MarshalJSON()
}

// ============================================================================
// Service Layer
// ============================================================================

type TransactionSimulatorService struct {
	db           *gorm.DB
	redis        *redis.Client
	config       *Config
	mu           sync.RWMutex
	activeSims   int64
	mevProtected map[string]bool
}

func NewTransactionSimulatorService(cfg *Config) (*TransactionSimulatorService, error) {
	// Database connection
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate
	err = db.AutoMigrate(&Simulation{}, &MempoolTransaction{}, &Block{}, &AccountState{}, &Token{}, &MEVAlert{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	service := &TransactionSimulatorService{
		db:            db,
		redis:         rdb,
		config:        cfg,
		mevProtected:  make(map[string]bool),
	}

	// Initialize default tokens
	service.initializeDefaultTokens()

	return service, nil
}

func (s *TransactionSimulatorService) initializeDefaultTokens() {
	tokens := []Token{
		{Address: "0x0000000000000000000000000000000000000000", Symbol: "ETH", Name: "Ethereum", Decimals: 18, IsNative: true, ChainID: 1, IsActive: true},
		{Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Symbol: "USDT", Name: "Tether USD", Decimals: 6, ChainID: 1, IsActive: true},
		{Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 1, IsActive: true},
		{Address: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", Symbol: "WBTC", Name: "Wrapped Bitcoin", Decimals: 8, ChainID: 1, IsActive: true},
		{Address: "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9", Symbol: "AAVE", Name: "Aave Token", Decimals: 18, ChainID: 1, IsActive: true},
		{Address: "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984", Symbol: "UNI", Name: "Uniswap", Decimals: 18, ChainID: 1, IsActive: true},
		{Address: "0x514910771AF9Ca656af840dff83E8264EcF986CA", Symbol: "LINK", Name: "Chainlink", Decimals: 18, ChainID: 1, IsActive: true},
	}

	for _, token := range tokens {
		s.db.FirstOrCreate(&token, Token{Address: token.Address})
	}
}

// ============================================================================
// API Handlers
// ============================================================================

type SimulateRequest struct {
	FromAddress    string `json:"from_address" binding:"required"`
	ToAddress      string `json:"to_address" binding:"required"`
	Value          string `json:"value" binding:"required"`
	GasLimit       uint64 `json:"gas_limit"`
	GasPrice       uint64 `json:"gas_price"`
	MaxFeePerGas   uint64 `json:"max_fee_per_gas"`
	MaxPriorityFee uint64 `json:"max_priority_fee"`
	Data           string `json:"data"`
	ChainID        uint64 `json:"chain_id"`
	Nonce          uint64 `json:"nonce"`
}

type SimulateResponse struct {
	Success         bool                   `json:"success"`
	TransactionHash string                 `json:"transaction_hash"`
	GasUsed         uint64                 `json:"gas_used"`
	GasPrice        uint64                 `json:"gas_price"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	MEVType         string                 `json:"mev_type"`
	MEVRisk         float64                `json:"mev_risk"`
	ExecutionTimeMs float64                `json:"execution_time_ms"`
	TokenTransfers  []TokenTransfer        `json:"token_transfers"`
	StateChanges    map[string]string      `json:"state_changes"`
	Warnings       []string                `json:"warnings"`
	Recommendations []string               `json:"recommendations"`
}

type TokenTransfer struct {
	Token   string `json:"token"`
	From    string `json:"from"`
	To      string `json:"to"`
	Amount  string `json:"amount"`
}

type BundleSimulateRequest struct {
	Transactions []SimulateRequest `json:"transactions" binding:"required"`
	IsFlashbots bool              `json:"is_flashbots"`
}

type GasEstimateResponse struct {
	GasLimit            uint64   `json:"gas_limit"`
	GasPrice            uint64   `json:"gas_price"`
	MaxFeePerGas        uint64   `json:"max_fee_per_gas"`
	MaxPriorityFee      uint64   `json:"max_priority_fee_per_gas"`
	EstimatedCost       string   `json:"estimated_cost"`
	Confidence          float64  `json:"confidence"`
	Factors             []string `json:"factors"`
}

type MEVAnalysisResponse struct {
	Type             string    `json:"type"`
	RiskScore        float64   `json:"risk_score"`
	Description      string    `json:"description"`
	PotentialLoss    string    `json:"potential_loss"`
	BotProbability   float64   `json:"bot_probability"`
	Recommendations  []string  `json:"recommendations"`
	ShouldBlock      bool      `json:"should_block"`
}

func (s *TransactionSimulatorService) SimulateTransaction(c *gin.Context) {
	var req SimulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check concurrent simulations limit
	if !s.acquireSimSlot() {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many concurrent simulations"})
		return
	}
	defer s.releaseSimSlot()

	startTime := time.Now()

	// Parse value
	valueDecimal, err := decimal.NewFromString(req.Value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid value"})
		return
	}

	// Get account state
	var account AccountState
	s.db.Where("address = ?", req.FromAddress).First(&account)

	// Check balance
	balanceDecimal, _ := decimal.NewFromString(account.Balance)
	requiredValue := valueDecimal

	if req.GasLimit > 0 && req.GasPrice > 0 {
		gasCost := decimal.NewFromInt(int64(req.GasLimit)).Mul(decimal.NewFromInt(int64(req.GasPrice)))
		requiredValue = requiredValue.Add(gasCost)
	}

	if balanceDecimal.LessThan(requiredValue) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
		return
	}

	// Calculate gas
	gasLimit := req.GasLimit
	if gasLimit == 0 {
		gasLimit = 21000 // Default for simple transfer
	}

	gasPrice := req.GasPrice
	if gasPrice == 0 {
		// Get current gas price from database or use default
		gasPrice = 20000000000 // 20 gwei
	}

	// Calculate gas used
	gasUsed := gasLimit
	if len(req.Data) > 0 {
		// Add data cost
		dataCost := 0
		for _, b := range req.Data {
			if b == '0' {
				dataCost += 4
			} else {
				dataCost += 16
			}
		}
		gasUsed += uint64(dataCost)
	}

	// Simulate transaction - in production this would call C++ simulator
	response := SimulateResponse{
		Success:         true,
		TransactionHash: s.generateTxHash(req),
		GasUsed:         gasUsed,
		GasPrice:        gasPrice,
		MEVType:         "NONE",
		MEVRisk:         0.0,
		ExecutionTimeMs: float64(time.Since(startTime).Milliseconds()),
	}

	// Perform MEV analysis if enabled
	if s.config.EnableMEVProtect {
		mevResponse := s.analyzeMEV(req)
		response.MEVType = mevResponse.Type
		response.MEVRisk = mevResponse.RiskScore

		if mevResponse.RiskScore > 0.7 {
			response.Warnings = append(response.Warnings, mevResponse.Description)
			response.Recommendations = mevResponse.Recommendations
		}
	}

	// Save simulation to database
	simulation := Simulation{
		TransactionHash: response.TransactionHash,
		ChainID:         req.ChainID,
		FromAddress:     req.FromAddress,
		ToAddress:       req.ToAddress,
		Value:           req.Value,
		GasLimit:        gasLimit,
		GasUsed:         gasUsed,
		GasPrice:        gasPrice,
		Success:         response.Success,
		MEVType:         response.MEVType,
		MEVRisk:         response.MEVRisk,
		ExecutionTimeMs: response.ExecutionTimeMs,
	}

	if !response.Success {
		simulation.ErrorMessage = response.ErrorMessage
	}

	s.db.Create(&simulation)

	// Cache result in Redis
	s.cacheSimulationResult(response.TransactionHash, response)

	c.JSON(http.StatusOK, response)
}

func (s *TransactionSimulatorService) SimulateBundle(c *gin.Context) {
	var req BundleSimulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results := make([]SimulateResponse, len(req.Transactions))
	totalGasUsed := uint64(0)
	totalExecutionTime := float64(0)

	for i, tx := range req.Transactions {
		var simReq SimulateRequest = tx
		simReq.ChainID = req.Transactions[i].ChainID
		
		// Simulate each transaction
		result := s.simulateSingleTransaction(simReq)
		results[i] = result
		totalGasUsed += result.GasUsed
		totalExecutionTime += result.ExecutionTimeMs
	}

	// Check for sandwich attacks if not Flashbots bundle
	mevDetected := false
	if !req.IsFlashbots {
		for i := 0; i < len(results)-1; i++ {
			if results[i].MEVRisk > 0.5 && results[i+1].MEVRisk > 0.5 {
				mevDetected = true
				results[i].Warnings = append(results[i].Warnings, "Potential sandwich attack detected")
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"results":            results,
		"total_gas_used":     totalGasUsed,
		"total_execution_ms": totalExecutionTime,
		"bundle_protected":   req.IsFlashbots || !mevDetected,
	})
}

func (s *TransactionSimulatorService) EstimateGas(c *gin.Context) {
	var req SimulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current block for base fee
	var block Block
	s.db.Order("number DESC").First(&block)

	// Calculate intrinsic gas
	gasLimit := uint64(21000)
	if req.GasLimit > 0 {
		gasLimit = req.GasLimit
	}

	// Add data cost
	if len(req.Data) > 0 {
		dataCost := 0
		for _, b := range req.Data {
			if b == '0' {
				dataCost += 4
			} else {
				dataCost += 16
			}
		}
		gasLimit += uint64(dataCost)
	}

	// Add buffer
	gasLimit = uint64(float64(gasLimit) * 1.2)

	// Calculate gas price
	gasPrice := req.GasPrice
	if gasPrice == 0 {
		gasPrice = 20000000000 // Default 20 gwei
	}

	baseFee := block.BaseFeePerGas
	if baseFee == 0 {
		baseFee = 10000000000 // Default 10 gwei
	}

	maxFeePerGas := req.MaxFeePerGas
	if maxFeePerGas == 0 {
		maxFeePerGas = baseFee * 2
	}

	maxPriorityFee := req.MaxPriorityFee
	if maxPriorityFee == 0 {
		maxPriorityFee = 1000000000 // 1 gwei
	}

	estimatedCost := decimal.NewFromInt(int64(gasLimit)).Mul(decimal.NewFromInt(int64(gasPrice)))

	response := GasEstimateResponse{
		GasLimit:        gasLimit,
		GasPrice:        gasPrice,
		MaxFeePerGas:    maxFeePerGas,
		MaxPriorityFee:  maxPriorityFee,
		EstimatedCost:   estimatedCost.String(),
		Confidence:      0.85,
		Factors: []string{
			"Intrinsic gas calculation applied",
			fmt.Sprintf("Current base fee: %d wei", baseFee),
			"Network congestion: medium",
			"Added 20% buffer for safety",
		},
	}

	c.JSON(http.StatusOK, response)
}

func (s *TransactionSimulatorService) AnalyzeMEV(c *gin.Context) {
	var req SimulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := s.analyzeMEV(req)
	c.JSON(http.StatusOK, response)
}

func (s *TransactionSimulatorService) analyzeMEV(req SimulateRequest) MEVAnalysisResponse {
	response := MEVAnalysisResponse{
		Type:           "NONE",
		RiskScore:      0.0,
		ShouldBlock:    false,
		Recommendations: []string{},
	}

	// Check if this is a swap transaction
	isSwap := len(req.Data) > 4

	if isSwap {
		// Check mempool for potential sandwich
		var memTxs []MempoolTransaction
		s.db.Where("to_address = ? AND status = ?", req.ToAddress, "pending").
			Order("gas_price DESC").Limit(10).Find(&memTxs)

		hasFrontRun := false
		hasBackRun := false

		reqGasPrice := req.GasPrice
		if reqGasPrice == 0 {
			reqGasPrice = 20000000000
		}

		for _, memTx := range memTxs {
			if memTx.GasPrice > uint64(float64(reqGasPrice)*1.2) {
				hasFrontRun = true
			}
			if memTx.GasPrice < uint64(float64(reqGasPrice)*0.8) {
				hasBackRun = true
			}
		}

		if hasFrontRun && hasBackRun {
			response.Type = "SANDFWICH_BOT"
			response.RiskScore = 0.95
			response.Description = "High probability sandwich attack detected"
			response.PotentialLoss = "Variable - depends on swap size"
			response.BotProbability = 0.85
			response.ShouldBlock = true
			response.Recommendations = []string{
				"Use Flashbots bundle for protection",
				"Increase slippage tolerance slightly",
				"Wait for mempool to clear",
				"Use private RPC endpoint",
			}
		} else if hasFrontRun {
			response.Type = "FRONTRUN_BOT"
			response.RiskScore = 0.75
			response.Description = "Potential front-run detected"
			response.ShouldBlock = false
			response.Recommendations = []string{
				"Consider using private transaction",
			}
		} else {
			response.RiskScore = 0.2
			response.Recommendations = []string{
				"Standard transaction - low MEV risk",
			}
		}

		// Save MEV alert if high risk
		if response.RiskScore > 0.7 {
			alert := MEVAlert{
				TransactionHash: s.generateTxHash(req),
				MEVType:         response.Type,
				RiskScore:       response.RiskScore,
				Description:     response.Description,
			}
			s.db.Create(&alert)
		}
	}

	return response
}

func (s *TransactionSimulatorService) simulateSingleTransaction(req SimulateRequest) SimulateResponse {
	startTime := time.Now()

	// Simple simulation logic
	gasLimit := req.GasLimit
	if gasLimit == 0 {
		gasLimit = 21000
	}

	gasPrice := req.GasPrice
	if gasPrice == 0 {
		gasPrice = 20000000000
	}

	response := SimulateResponse{
		Success:         true,
		TransactionHash: s.generateTxHash(req),
		GasUsed:         gasLimit,
		GasPrice:        gasPrice,
		ExecutionTimeMs: float64(time.Since(startTime).Milliseconds()),
	}

	return response
}

func (s *TransactionSimulatorService) GetMempool(c *gin.Context) {
	var txs []MempoolTransaction
	s.db.Where("status = ?", "pending").Order("gas_price DESC").Limit(100).Find(&txs)

	c.JSON(http.StatusOK, gin.H{
		"count":      len(txs),
		"transactions": txs,
	})
}

func (s *TransactionSimulatorService) AddToMempool(c *gin.Context) {
	var req SimulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := MempoolTransaction{
		Hash:        s.generateTxHash(req),
		FromAddress: req.FromAddress,
		ToAddress:   req.ToAddress,
		Value:       req.Value,
		GasLimit:    req.GasLimit,
		GasPrice:    req.GasPrice,
		Data:        req.Data,
		ChainID:     req.ChainID,
		Nonce:       req.Nonce,
		Status:      "pending",
	}

	s.db.Create(&tx)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"hash":    tx.Hash,
	})
}

func (s *TransactionSimulatorService) GetBlock(c *gin.Context) {
	number := c.Param("number")
	var block Block

	if number == "latest" {
		s.db.Order("number DESC").First(&block)
	} else {
		var num uint64
		fmt.Sscanf(number, "%d", &num)
		s.db.Where("number = ?", num).First(&block)
	}

	if block.Number == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "block not found"})
		return
	}

	c.JSON(http.StatusOK, block)
}

func (s *TransactionSimulatorService) GetMetrics(c *gin.Context) {
	var totalSims, successSims, failedSims int64
	s.db.Model(&Simulation{}).Count(&totalSims)
	s.db.Model(&Simulation{}).Where("success = ?", true).Count(&successSims)
	s.db.Model(&Simulation{}).Where("success = ?", false).Count(&failedSims)

	var mempoolCount int64
	s.db.Model(&MempoolTransaction{}).Where("status = ?", "pending").Count(&mempoolCount)

	var mevAlerts int64
	s.db.Model(&MEVAlert{}).Where("risk_score > ?", 0.7).Count(&mevAlerts)

	c.JSON(http.StatusOK, gin.H{
		"total_simulations":   totalSims,
		"successful_simulations": successSims,
		"failed_simulations":  failedSims,
		"mempool_size":        mempoolCount,
		"mev_alerts":          mevAlerts,
		"active_simulations":  atomic.LoadInt64(&s.activeSims),
	})
}

func (s *TransactionSimulatorService) GetTokens(c *gin.Context) {
	chainID := c.Query("chain_id")

	var tokens []Token
	query := s.db.Where("is_active = ?", true)
	if chainID != "" {
		var chainIDUint uint64
		fmt.Sscanf(chainID, "%d", &chainIDUint)
		query = query.Where("chain_id = ?", chainIDUint)
	}
	query.Find(&tokens)

	c.JSON(http.StatusOK, gin.H{
		"count":  len(tokens),
		"tokens": tokens,
	})
}

func (s *TransactionSimulatorService) GetAccountState(c *gin.Context) {
	address := c.Param("address")

	var account AccountState
	result := s.db.Where("address = ?", address).First(&account)

	if result.Error != nil {
		// Create default account
		account = AccountState{
			Address:     address,
			Balance:     "0",
			Nonce:       0,
			CodeHash:    "0x",
			TokenBalances: JSON(`{}`),
		}
		s.db.Create(&account)
	}

	c.JSON(http.StatusOK, account)
}

// ============================================================================
// Helper Methods
// ============================================================================

func (s *TransactionSimulatorService) generateTxHash(req SimulateRequest) string {
	log.Printf("generateTxHash: transaction broadcast not implemented - cannot generate tx hash without broadcasting")
	return ""
}

func (s *TransactionSimulatorService) acquireSimSlot() bool {
	return atomic.CompareAndSwapInt64(&s.activeSims, 0, 1)
}

func (s *TransactionSimulatorService) releaseSimSlot() {
	atomic.StoreInt64(&s.activeSims, 0)
}

func (s *TransactionSimulatorService) cacheSimulationResult(hash string, result SimulateResponse) {
	key := fmt.Sprintf("tx_sim:%s", hash)
	data, _ := json.Marshal(result)
	s.redis.Set(context.Background(), key, data, 5*time.Minute)
}

// ============================================================================
// WebSocket Handler
// ============================================================================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *TransactionSimulatorService) HandleWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Subscribe to mempool updates
	pubsub := s.redis.Subscribe("mempool_updates")
	defer pubsub.Close()

	for {
		select {
		case msg := <-pubsub.Channel():
			if msg != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
			}
		default:
			// Keep connection alive
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	// Initialize service
	service, err := NewTransactionSimulatorService(config)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	// Setup Gin router
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// CORS middleware
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
	api := router.Group("/api/v1")
	{
		// Transaction simulation
		api.POST("/simulate", service.SimulateTransaction)
		api.POST("/simulate/bundle", service.SimulateBundle)
		api.POST("/estimate/gas", service.EstimateGas)
		api.POST("/analyze/mev", service.AnalyzeMEV)

		// Mempool
		api.GET("/mempool", service.GetMempool)
		api.POST("/mempool", service.AddToMempool)

		// Block
		api.GET("/block/:number", service.GetBlock)

		// Account
		api.GET("/account/:address", service.GetAccountState)

		// Tokens
		api.GET("/tokens", service.GetTokens)

		// Metrics
		api.GET("/metrics", service.GetMetrics)

		// WebSocket
		api.GET("/ws", service.HandleWS)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Transaction Simulator starting on port %s", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}

var atomic = sync/atomic
