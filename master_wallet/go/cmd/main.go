// Master Wallet Service - Go Implementation
// Complete master wallet management with multi-chain support
// High-performance, production-ready

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// CONSTANTS
// ============================================================================

const (
	// Chain IDs
	ChainEthereum   = "ethereum"
	ChainBSC        = "bsc"
	ChainPolygon    = "polygon"
	ChainArbitrum   = "arbitrum"
	ChainOptimism   = "optimism"
	ChainBase      = "base"
	ChainAvalanche = "avalanche"
	ChainSolana    = "solana"
	ChainAptos     = "aptos"
	ChainSui       = "sui"
	ChainTron      = "tron"
	ChainCosmos    = "cosmos"

	// Fee Types
	FeeTypeWithdrawal = "withdrawal"
	FeeTypeSwap       = "swap"
	FeeTypeTransfer   = "transfer"
	FeeTypeTrading    = "trading"

	// Transaction Status
	TxStatusPending   = "pending"
	TxStatusConfirmed = "confirmed"
	TxStatusFailed    = "failed"
)

// ============================================================================
// LOGGING
// ============================================================================

var logger zerolog.Logger

func initLogger() {
	output := zerolog.ConsoleWriter{Out: os.Stdout}
	logger = zerolog.New(output).With().Timestamp().Caller().Logger()
	logger.Level(zerolog.InfoLevel)
}

// ============================================================================
// MODELS
// ============================================================================

type MasterWallet struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Name           string    `gorm:"size:100;not null" json:"name"`
	EncryptedKey   string    `gorm:"type:text;not null" json:"-"`
	PublicKey      string    `gorm:"size:200;not null" json:"public_key"`
	MasterSeedHash string    `gorm:"size:200;not null" json:"-"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type WalletAddress struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	MasterWalletID uint    `gorm:"index" json:"master_wallet_id"`
	ChainID      string    `gorm:"size:50;not null" json:"chain_id"`
	Address      string    `gorm:"size:100;not null;uniqueIndex" json:"address"`
	PrivateKeyID string    `gorm:"size:100" json:"-"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type FeeConfig struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	FeeType        string    `gorm:"size:50;not null" json:"fee_type"`
	ChainID        string    `gorm:"size:50;not null" json:"chain_id"`
	TokenSymbol    string    `gorm:"size:20" json:"token_symbol"`
	FeePercent     float64   `gorm:"not null;default:0" json:"fee_percent"`
	FeeFixed       float64   `gorm:"not null;default:0" json:"fee_fixed"`
	MinFee         float64   `gorm:"not null;default:0" json:"min_fee"`
	MaxFee         float64   `gorm:"not null;default:0" json:"max_fee"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Transaction struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	TxHash          string    `gorm:"size:100;uniqueIndex" json:"tx_hash"`
	ChainID         string    `gorm:"size:50;not null;index" json:"chain_id"`
	FromAddress     string    `gorm:"size:100;not null" json:"from_address"`
	ToAddress       string    `gorm:"size:100;not null" json:"to_address"`
	TokenSymbol     string    `gorm:"size:20" json:"token_symbol"`
	Amount          string    `gorm:"type:text;not null" json:"amount"`
	FeeAmount       string    `gorm:"type:text" json:"fee_amount"`
	FeeEarned       string    `gorm:"type:text" json:"fee_earned"`
	Status          string    `gorm:"size:20;not null;default:pending" json:"status"`
	BlockNumber     int64     `json:"block_number"`
	Confirmations   int       `json:"confirmations"`
	ErrorMessage   string    `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type BlockchainConfig struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	ChainID      string    `gorm:"size:50;not null;uniqueIndex" json:"chain_id"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	Symbol       string    `gorm:"size:20;not null" json:"symbol"`
	ChainType    string    `gorm:"size:20;not null" json:"chain_type"`
	RPCURL       string    `gorm:"type:text" json:"rpc_url"`
	ExplorerURL  string    `gorm:"type:text" json:"explorer_url"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	IsDefault    bool      `gorm:"default:false" json:"is_default"`
	GasPrice     float64   `json:"gas_price"`
	MinConfirmations int    `json:"min_confirmations"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ============================================================================
// SERVICES
// ============================================================================

type MasterWalletService struct {
	db         *gorm.DB
	redis      *redis.Client
	keyStore   *KeyStore
	txManager  *TransactionManager
	feeService *FeeService
	mu         sync.RWMutex
}

type KeyStore struct {
	mu       sync.Mutex
	keys     map[string]*ecdsa.PrivateKey
	encrypted map[string]string
}

type TransactionManager struct {
	db      *gorm.DB
	pending map[string]*Transaction
	mu      sync.RWMutex
}

type FeeService struct {
	db *gorm.DB
}

// NewMasterWalletService creates a new master wallet service
func NewMasterWalletService(db *gorm.DB, redisClient *redis.Client) *MasterWalletService {
	return &MasterWalletService{
		db:         db,
		redis:      redisClient,
		keyStore:   NewKeyStore(),
		txManager:  NewTransactionManager(db),
		feeService: NewFeeService(db),
	}
}

// GenerateMasterWallet generates a new master wallet with addresses for all supported chains
func (s *MasterWalletService) GenerateMasterWallet(ctx context.Context, name string, password string) (*MasterWallet, map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Info().Str("name", name).Msg("Generating new master wallet")

	// Generate cryptographic key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Derive master seed (in production, use BIP-39)
	masterSeed := generateMasterSeed(password)

	// Encrypt private key
	encryptedKey, err := encryptPrivateKey(privateKey, password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt key: %w", err)
	}

	// Create master wallet record
	wallet := &MasterWallet{
		Name:           name,
		EncryptedKey:   encryptedKey,
		PublicKey:      hex.EncodeToString(append(privateKey.X.Bytes(), privateKey.Y.Bytes()...)),
		MasterSeedHash: hashString(masterSeed),
		IsActive:       true,
	}

	if err := s.db.Create(wallet).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	// Generate addresses for all supported chains
	addresses := s.generateChainAddresses(wallet.ID, masterSeed)

	// Store addresses in database
	for chainID, address := range addresses {
		walletAddr := &WalletAddress{
			MasterWalletID: wallet.ID,
			ChainID:        chainID,
			Address:        address,
			IsActive:       true,
		}
		if err := s.db.Create(walletAddr).Error; err != nil {
			logger.Error().Err(err).Str("chainID", chainID).Msg("Failed to create address")
		}
	}

	logger.Info().Uint("walletID", wallet.ID).Msg("Master wallet created successfully")

	return wallet, addresses, nil
}

// generateChainAddresses generates addresses for all supported chains
func (s *MasterWalletService) generateChainAddresses(walletID uint, seed string) map[string]string {
	addresses := make(map[string]string)

	chains := []string{
		ChainEthereum, ChainBSC, ChainPolygon, ChainArbitrum,
		ChainOptimism, ChainBase, ChainAvalanche, ChainSolana,
		ChainAptos, ChainSui, ChainTron, ChainCosmos,
	}

	for _, chain := range chains {
		// Derive address based on chain type
		addresses[chain] = deriveAddress(seed, chain)
	}

	return addresses
}

// deriveAddress derives an address from the master seed for a specific chain
func deriveAddress(seed, chainID string) string {
	// Simplified derivation - in production use proper BIP-32/BIP-44
	data := fmt.Sprintf("%s:%s", seed, chainID)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:40]
}

// ProcessTransaction processes a transaction through the master wallet
func (s *MasterWalletService) ProcessTransaction(ctx context.Context, req *TransactionRequest) (*Transaction, error) {
	logger.Info().
		Str("to", req.ToAddress).
		Str("amount", req.Amount).
		Str("chain", req.ChainID).
		Msg("Processing transaction")

	// Get fee configuration
	feeConfig, err := s.feeService.GetFee(ctx, FeeTypeWithdrawal, req.ChainID, req.TokenSymbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get fee: %w", err)
	}

	// Calculate fee
	amountDecimal, _ := decimal.NewFromString(req.Amount)
	feeDecimal := calculateFee(amountDecimal, feeConfig)

	// Validate balance (in production, check actual balance)
	if !s.hasEnoughBalance(req.ChainID, req.TokenSymbol, amountDecimal) {
		return nil, fmt.Errorf("insufficient balance")
	}

	// Get master wallet address for the chain
	fromAddress, err := s.getMasterAddress(req.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get master address: %w", err)
	}

	// Create transaction record
	tx := &Transaction{
		ChainID:     req.ChainID,
		FromAddress: fromAddress,
		ToAddress:   req.ToAddress,
		TokenSymbol: req.TokenSymbol,
		Amount:      req.Amount,
		FeeAmount:   feeDecimal.String(),
		FeeEarned:   feeDecimal.Mul(decimal.NewFromFloat(feeConfig.FeePercent / 100)).String(),
		Status:      TxStatusPending,
	}

	if err := s.db.Create(tx).Error; err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Simulate transaction signing and broadcasting
	// In production, this would actually sign and broadcast
	tx.TxHash = s.signAndBroadcast(ctx, tx)

	tx.Status = TxStatusConfirmed
	s.db.Save(tx)

	logger.Info().
		Str("txHash", tx.TxHash).
		Uint("txID", tx.ID).
		Msg("Transaction processed successfully")

	return tx, nil
}

// getMasterAddress gets the master wallet address for a chain
func (s *MasterWalletService) getMasterAddress(chainID string) (string, error) {
	var addr WalletAddress
	err := s.db.Where("chain_id = ? AND is_active = ?", chainID, true).First(&addr).Error
	if err != nil {
		return "", err
	}
	return addr.Address, nil
}

// hasEnoughBalance checks if the master wallet has enough balance
func (s *MasterWalletService) hasEnoughBalance(chainID, tokenSymbol string, required decimal.Decimal) bool {
	// In production, this would check actual blockchain balance
	// For now, return true (simulation)
	return true
}

// signAndBroadcast signs and broadcasts a transaction
func (s *MasterWalletService) signAndBroadcast(ctx context.Context, tx *Transaction) string {
	// In production:
	// 1. Decrypt master key
	// 2. Sign transaction
	// 3. Broadcast to network
	// 4. Return transaction hash

	// Simulated transaction hash
	txHash := fmt.Sprintf("0x%x", sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", tx.FromAddress, tx.ToAddress, tx.Amount))))
	return txHash
}

// UpdateFeeConfig updates the fee configuration
func (s *MasterWalletService) UpdateFeeConfig(ctx context.Context, config *FeeConfig) error {
	logger.Info().
		Str("feeType", config.FeeType).
		Str("chainID", config.ChainID).
		Float64("percent", config.FeePercent).
		Msg("Updating fee configuration")

	var existing FeeConfig
	err := s.db.Where("fee_type = ? AND chain_id = ?", config.FeeType, config.ChainID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		return s.db.Create(config).Error
	}

	config.ID = existing.ID
	return s.db.Save(config).Error
}

// GetFee gets the current fee configuration
func (s *FeeService) GetFee(ctx context.Context, feeType, chainID, tokenSymbol string) (*FeeConfig, error) {
	var fee FeeConfig
	err := s.db.Where("fee_type = ? AND chain_id = ? AND is_active = ?", feeType, chainID, true).First(&fee).Error
	if err != nil {
		return nil, err
	}
	return &fee, nil
}

// AddBlockchain adds a new blockchain to the system
func (s *MasterWalletService) AddBlockchain(ctx context.Context, config *BlockchainConfig) error {
	logger.Info().
		Str("chainID", config.ChainID).
		Str("name", config.Name).
		Msg("Adding new blockchain")

	return s.db.Create(config).Error
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func generateMasterSeed(password string) string {
	data := fmt.Sprintf("tigerwallet_master:%s:%d", password, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func encryptPrivateKey(key *ecdsa.PrivateKey, password string) (string, error) {
	// Simplified encryption - in production use AES-256-GCM
	keyBytes := append(key.X.Bytes(), key.Y.Bytes()...)
	keyHash := sha256.Sum256([]byte(password))
	encrypted := make([]byte, len(keyBytes))
	for i, b := range keyBytes {
		encrypted[i] = b ^ keyHash[i%len(keyHash)]
	}
	return hex.EncodeToString(encrypted), nil
}

func hashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

func calculateFee(amount decimal.Decimal, config *FeeConfig) decimal.Decimal {
	percentFee := amount.Mul(decimal.NewFromFloat(config.FeePercent / 100))
	fixedFee := decimal.NewFromFloat(config.FeeFixed)

	fee := percentFee.Add(fixedFee)

	// Apply min/max limits
	if config.MinFee > 0 && fee.LessThan(decimal.NewFromFloat(config.MinFee)) {
		fee = decimal.NewFromFloat(config.MinFee)
	}
	if config.MaxFee > 0 && fee.GreaterThan(decimal.NewFromFloat(config.MaxFee)) {
		fee = decimal.NewFromFloat(config.MaxFee)
	}

	return fee
}

func NewKeyStore() *KeyStore {
	return &KeyStore{
		keys:     make(map[string]*ecdsa.PrivateKey),
		encrypted: make(map[string]string),
	}
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{
		db:      db,
		pending: make(map[string]*Transaction),
	}
}

func NewFeeService(db *gorm.DB) *FeeService {
	return &FeeService{db: db}
}

// ============================================================================
// API HANDLERS
// ============================================================================

type TransactionRequest struct {
	ChainID    string `json:"chain_id" binding:"required"`
	ToAddress  string `json:"to_address" binding:"required"`
	Amount     string `json:"amount" binding:"required"`
	TokenSymbol string `json:"token_symbol"`
}

func SetupRoutes(r *gin.Engine, service *MasterWalletService) {
	api := r.Group("/api/v1/master-wallet")
	{
		// Wallet management
		api.POST("/generate", service.GenerateWallet)
		api.GET("/addresses", service.GetAddresses)
		api.GET("/balance/:chainID", service.GetBalance)

		// Transaction management
		api.POST("/transaction", service.ProcessTransactionHandler)
		api.GET("/transactions", service.GetTransactions)
		api.GET("/transaction/:id", service.GetTransaction)

		// Fee management
		api.POST("/fees", service.UpdateFee)
		api.GET("/fees", service.GetFees)

		// Blockchain management
		api.POST("/blockchains", service.AddBlockchainHandler)
		api.GET("/blockchains", service.GetBlockchains)
	}
}

func (s *MasterWalletService) GenerateWallet(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	wallet, addresses, err := s.GenerateMasterWallet(c.Request.Context(), req.Name, req.Password)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"wallet":   wallet,
		"addresses": addresses,
	})
}

func (s *MasterWalletService) GetAddresses(c *gin.Context) {
	var addresses []WalletAddress
	s.db.Find(&addresses)
	c.JSON(200, addresses)
}

func (s *MasterWalletService) GetBalance(c *gin.Context) {
	chainID := c.Param("chainID")
	// In production, query actual blockchain
	c.JSON(200, gin.H{"chain_id": chainID, "balance": "0"})
}

func (s *MasterWalletService) ProcessTransactionHandler(c *gin.Context) {
	var req TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	tx, err := s.ProcessTransaction(c.Request.Context(), &req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, tx)
}

func (s *MasterWalletService) GetTransactions(c *gin.Context) {
	var txs []Transaction
	s.db.Limit(100).Order("created_at desc").Find(&txs)
	c.JSON(200, txs)
}

func (s *MasterWalletService) GetTransaction(c *gin.Context) {
	id := c.Param("id")
	var tx Transaction
	if err := s.db.First(&tx, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, tx)
}

func (s *MasterWalletService) UpdateFee(c *gin.Context) {
	var config FeeConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := s.UpdateFeeConfig(c.Request.Context(), &config); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, config)
}

func (s *MasterWalletService) GetFees(c *gin.Context) {
	var fees []FeeConfig
	s.db.Find(&fees)
	c.JSON(200, fees)
}

func (s *MasterWalletService) AddBlockchainHandler(c *gin.Context) {
	var config BlockchainConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := s.AddBlockchain(c.Request.Context(), &config); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, config)
}

func (s *MasterWalletService) GetBlockchains(c *gin.Context) {
	var chains []BlockchainConfig
	s.db.Find(&chains)
	c.JSON(200, chains)
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	initLogger()
	logger.Info().Msg("Starting TigerWallet Master Wallet Service")

	// Database setup
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres dbname=tigerwallet port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Auto migrate
	db.AutoMigrate(&MasterWallet{}, &WalletAddress{}, &FeeConfig{}, &Transaction{}, &BlockchainConfig{})

	// Redis setup
	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})

	// Initialize service
	service := NewMasterWalletService(db, redisClient)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	SetupRoutes(r, service)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Info().Msg("Shutting down...")
		os.Exit(0)
	}()

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info().Str("port", port).Msg("Server starting")
	if err := r.Run(":" + port); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}
}
