/**
 * TigerWallet Master Wallet Service
 * Comprehensive master wallet management system with fee control, blockchain integration, and automated operations
 */

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// Configuration
type Config struct {
	ServerPort    string
	RedisAddr    string
	JWTSecret    string
	MasterKeyHex string
	HotWalletLimit int64
}

// Chain types
type ChainType string

const (
	ChainEthereum    ChainType = "ETHEREUM"
	ChainBSC         ChainType = "BSC"
	ChainPolygon     ChainType = "POLYGON"
	ChainArbitrum    ChainType = "ARBITRUM"
	ChainOptimism    ChainType = "OPTIMISM"
	ChainAvalanche   ChainType = "AVALANCHE"
	ChainSolana      ChainType = "SOLANA"
	ChainBitcoin     ChainType = "BITCOIN"
	ChainTron        ChainType = "TRON"
)

// Transaction Status
type TxStatus string

const (
	TxStatusPending   TxStatus = "PENDING"
	TxStatusConfirmed TxStatus = "CONFIRMED"
	TxStatusFailed    TxStatus = "FAILED"
)

// Token
type Token struct {
	TokenID        string  `json:"token_id"`
	Symbol         string  `json:"symbol"`
	Name           string  `json:"name"`
	ContractAddress string `json:"contract_address"`
	Chain          ChainType `json:"chain"`
	Decimals       int     `json:"decimals"`
	Status         string  `json:"status"`
	MinAmount      float64 `json:"min_amount"`
	MaxAmount      float64 `json:"max_amount"`
	FeeAmount      float64 `json:"fee_amount"`
	FeeType        string  `json:"fee_type"` // fixed or percentage
}

// Wallet
type Wallet struct {
	WalletID     string            `json:"wallet_id"`
	Address      string            `json:"address"`
	Chain        ChainType         `json:"chain"`
	Type         string            `json:"type"` // master, user, hot, cold
	Balance      map[string]string `json:"balance"`
	IsActive     bool              `json:"is_active"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// Transaction
type Transaction struct {
	TxID         string            `json:"tx_id"`
	WalletID     string            `json:"wallet_id"`
	FromAddress  string            `json:"from_address"`
	ToAddress    string            `json:"to_address"`
	Token        string            `json:"token"`
	Amount       string            `json:"amount"`
	Fee          string            `json:"fee"`
	Total        string            `json:"total"`
	Status       TxStatus          `json:"status"`
	Chain        ChainType         `json:"chain"`
	Hash         string            `json:"hash"`
	BlockNumber  uint64           `json:"block_number"`
	Nonce        uint64           `json:"nonce"`
	GasPrice     string            `json:"gas_price"`
	GasLimit     uint64           `json:"gas_limit"`
	CreatedAt    time.Time        `json:"created_at"`
	ConfirmedAt  *time.Time       `json:"confirmed_at,omitempty"`
	FailureReason string          `json:"failure_reason,omitempty"`
}

// Fee Configuration
type FeeConfig struct {
	FeeID          string  `json:"fee_id"`
	FeeType        string  `json:"fee_type"` // withdraw, swap, transfer, deposit
	Token          string  `json:"token"`
	Chain          ChainType `json:"chain"`
	FeeAmount      float64 `json:"fee_amount"`
	FeePercentage  float64 `json:"fee_percentage"`
	MinFee         float64 `json:"min_fee"`
	MaxFee         float64 `json:"max_fee"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Blockchain Config
type BlockchainConfig struct {
	ChainID       int      `json:"chain_id"`
	Chain         ChainType `json:"chain"`
	Name          string   `json:"name"`
	RPCURL        string   `json:"rpc_url"`
	ExplorerURL   string   `json:"explorer_url"`
	Symbol        string   `json:"symbol"`
	Decimals      int      `json:"decimals"`
	ConfirmBlocks uint64  `json:"confirm_blocks"`
	IsActive      bool     `json:"is_active"`
}

// Master Wallet Service
type MasterWalletService struct {
	config         Config
	redis          *redis.Client
	masterKey     *ecdsa.PrivateKey
	masterAddress common.Address
	wallets       map[string]*Wallet
	transactions  map[string]*Transaction
	tokens        map[string]*Token
	feeConfigs    map[string]*FeeConfig
	blockchains   map[ChainType]*BlockchainConfig
	mu            sync.RWMutex
}

// NewMasterWalletService creates a new master wallet service
func NewMasterWalletService(cfg Config) *MasterWalletService {
	// Generate or load master key
	var masterKey *ecdsa.PrivateKey
	var masterAddress common.Address

	if cfg.MasterKeyHex != "" {
		// Load existing key
		keyBytes, err := hex.DecodeString(cfg.MasterKeyHex)
		if err != nil {
			log.Fatalf("Invalid master key: %v", err)
		}
		masterKey, err = crypto.ToECDSA(keyBytes)
		if err != nil {
			log.Fatalf("Invalid master key: %v", err)
		}
		masterAddress = crypto.PubkeyToAddress(masterKey.PublicKey)
	} else {
		// Generate new key
		var err error
		masterKey, err = crypto.GenerateKey()
		if err != nil {
			log.Fatalf("Failed to generate master key: %v", err)
		}
		masterAddress = crypto.PubkeyToAddress(masterKey.PublicKey)
		log.Printf("Generated new master address: %s", masterAddress.Hex())
		log.Printf("IMPORTANT: Save this master key: %s", hex.EncodeToString(crypto.FromECDSA(masterKey)))
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   3,
	})

	service := &MasterWalletService{
		config:        cfg,
		redis:         redisClient,
		masterKey:     masterKey,
		masterAddress: masterAddress,
		wallets:      make(map[string]*Wallet),
		transactions: make(map[string]*Transaction),
		tokens:       make(map[string]*Token),
		feeConfigs:   make(map[string]*FeeConfig),
		blockchains:  make(map[ChainType]*BlockchainConfig),
	}

	// Initialize default blockchains
	service.initializeBlockchains()

	// Initialize default tokens
	service.initializeTokens()

	// Initialize default fees
	service.initializeFees()

	// Create master wallet
	service.createMasterWallet()

	return service
}

func (s *MasterWalletService) initializeBlockchains() {
	s.blockchains = map[ChainType]*BlockchainConfig{
		ChainEthereum: {
			ChainID: 1, Chain: ChainEthereum, Name: "Ethereum",
			RPCURL: "https://eth.llamarpc.com", ExplorerURL: "https://etherscan.io",
			Symbol: "ETH", Decimals: 18, ConfirmBlocks: 12, IsActive: true,
		},
		ChainBSC: {
			ChainID: 56, Chain: ChainBSC, Name: "BNB Smart Chain",
			RPCURL: "https://bsc-dataseed.binance.org", ExplorerURL: "https://bscscan.com",
			Symbol: "BNB", Decimals: 18, ConfirmBlocks: 15, IsActive: true,
		},
		ChainPolygon: {
			ChainID: 137, Chain: ChainPolygon, Name: "Polygon",
			RPCURL: "https://polygon-rpc.com", ExplorerURL: "https://polygonscan.com",
			Symbol: "MATIC", Decimals: 18, ConfirmBlocks: 15, IsActive: true,
		},
		ChainArbitrum: {
			ChainID: 42161, Chain: ChainArbitrum, Name: "Arbitrum One",
			RPCURL: "https://arb1.arbitrum.io/rpc", ExplorerURL: "https://arbiscan.io",
			Symbol: "ETH", Decimals: 18, ConfirmBlocks: 15, IsActive: true,
		},
		ChainOptimism: {
			ChainID: 10, Chain: ChainOptimism, Name: "Optimism",
			RPCURL: "https://mainnet.optimism.io", ExplorerURL: "https://optimistic.etherscan.io",
			Symbol: "ETH", Decimals: 18, ConfirmBlocks: 15, IsActive: true,
		},
		ChainAvalanche: {
			ChainID: 43114, Chain: ChainAvalanche, Name: "Avalanche C-Chain",
			RPCURL: "https://api.avax.network/ext/bc/C/rpc", ExplorerURL: "https://snowtrace.io",
			Symbol: "AVAX", Decimals: 18, ConfirmBlocks: 15, IsActive: true,
		},
	}
}

func (s *MasterWalletService) initializeTokens() {
	s.tokens = map[string]*Token{
		"ETH_ETHEREUM":   {TokenID: "ETH_ETHEREUM", Symbol: "ETH", Name: "Ethereum", Chain: ChainEthereum, Decimals: 18, Status: "ACTIVE", MinAmount: 0.001, MaxAmount: 10000, FeeAmount: 0.001, FeeType: "fixed"},
		"BNB_BSC":        {TokenID: "BNB_BSC", Symbol: "BNB", Name: "BNB", Chain: ChainBSC, Decimals: 18, Status: "ACTIVE", MinAmount: 0.001, MaxAmount: 10000, FeeAmount: 0.001, FeeType: "fixed"},
		"USDT_ETHEREUM":  {TokenID: "USDT_ETHEREUM", Symbol: "USDT", Name: "Tether USD", ContractAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Chain: ChainEthereum, Decimals: 6, Status: "ACTIVE", MinAmount: 10, MaxAmount: 100000, FeeAmount: 1, FeeType: "fixed"},
		"USDC_ETHEREUM":  {TokenID: "USDC_ETHEREUM", Symbol: "USDC", Name: "USD Coin", ContractAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Chain: ChainEthereum, Decimals: 6, Status: "ACTIVE", MinAmount: 10, MaxAmount: 100000, FeeAmount: 1, FeeType: "fixed"},
		"USDT_BSC":       {TokenID: "USDT_BSC", Symbol: "USDT", Name: "Tether USD", ContractAddress: "0x55d398326f99059fF775485246999027B3197955", Chain: ChainBSC, Decimals: 18, Status: "ACTIVE", MinAmount: 10, MaxAmount: 100000, FeeAmount: 1, FeeType: "fixed"},
	}
}

func (s *MasterWalletService) initializeFees() {
	s.feeConfigs = map[string]*FeeConfig{
		"withdraw_eth":    {FeeID: "withdraw_eth", FeeType: "withdraw", Token: "ETH", FeeAmount: 0.001, FeePercentage: 0, MinFee: 0.001, MaxFee: 10, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		"withdraw_usdt":   {FeeID: "withdraw_usdt", FeeType: "withdraw", Token: "USDT", FeeAmount: 1, FeePercentage: 0, MinFee: 1, MaxFee: 100, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		"swap_eth":        {FeeID: "swap_eth", FeeType: "swap", Token: "ETH", FeeAmount: 0, FeePercentage: 0.3, MinFee: 0.001, MaxFee: 100, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		"transfer_eth":    {FeeID: "transfer_eth", FeeType: "transfer", Token: "ETH", FeeAmount: 0.0005, FeePercentage: 0, MinFee: 0.0005, MaxFee: 5, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
}

func (s *MasterWalletService) createMasterWallet() {
	masterWallet := &Wallet{
		WalletID:  "master_" + uuid.New().String()[:8],
		Address:   s.masterAddress.Hex(),
		Chain:     ChainEthereum,
		Type:      "master",
		Balance:   make(map[string]string),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.wallets[masterWallet.WalletID] = masterWallet

	// Create hot wallet
	hotKey, _ := crypto.GenerateKey()
	hotAddress := crypto.PubkeyToAddress(hotKey.PublicKey)

	hotWallet := &Wallet{
		WalletID:  "hot_" + uuid.New().String()[:8],
		Address:   hotAddress.Hex(),
		Chain:     ChainEthereum,
		Type:      "hot",
		Balance:   make(map[string]string),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.wallets[hotWallet.WalletID] = hotWallet
	log.Printf("Created master wallet: %s", masterWallet.Address)
	log.Printf("Created hot wallet: %s", hotWallet.Address)
}

// Create user wallet
func (s *MasterWalletService) CreateUserWallet(userID string, chain ChainType) (*Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate new key for user
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	address := crypto.PubkeyToAddress(key.PublicKey)

	wallet := &Wallet{
		WalletID:  "user_" + userID + "_" + strings.ToLower(string(chain)),
		Address:   address.Hex(),
		Chain:     chain,
		Type:      "user",
		Balance:   make(map[string]string),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.wallets[wallet.WalletID] = wallet

	return wallet, nil
}

// Get wallet by address
func (s *MasterWalletService) GetWalletByAddress(address string) *Wallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, wallet := range s.wallets {
		if strings.ToLower(wallet.Address) == strings.ToLower(address) {
			return wallet
		}
	}
	return nil
}

// Get wallet by ID
func (s *MasterWalletService) GetWallet(walletID string) *Wallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.wallets[walletID]
}

// Get all wallets
func (s *MasterWalletService) GetWallets(walletType, chain string) []*Wallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallets := make([]*Wallet, 0)
	for _, wallet := range s.wallets {
		match := true
		if walletType != "" && wallet.Type != walletType {
			match = false
		}
		if chain != "" && string(wallet.Chain) != chain {
			match = false
		}
		if match {
			wallets = append(wallets, wallet)
		}
	}

	return wallets
}

// Process withdrawal
func (s *MasterWalletService) ProcessWithdrawal(walletID, toAddress, token, amountStr string) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}

	if !wallet.IsActive {
		return nil, fmt.Errorf("wallet is not active")
	}

	// Parse amount
	var amount *big.Int
	if token == "ETH" || token == "BNB" {
		// Convert from ether to wei
		amountFloat := 0.0
		fmt.Sscanf(amountStr, "%f", &amountFloat)
		amount = new(big.Int).Mul(big.NewInt(int64(amountFloat*1e18)), big.NewInt(1))
	} else {
		// For tokens, use simpler parsing
		amount = new(big.Int)
		amount.SetString(amountStr, 10)
	}

	// Calculate fee
	feeConfig, ok := s.feeConfigs["withdraw_"+strings.ToLower(token)]
	if !ok {
		feeConfig = s.feeConfigs["withdraw_eth"]
	}

	fee := calculateFee(amount.Float64(), feeConfig)

	// Create transaction
	tx := &Transaction{
		TxID:        "tx_" + uuid.New().String()[:12],
		WalletID:    walletID,
		FromAddress: wallet.Address,
		ToAddress:   toAddress,
		Token:       token,
		Amount:      amountStr,
		Fee:         fmt.Sprintf("%f", fee),
		Status:      TxStatusPending,
		Chain:       wallet.Chain,
		CreatedAt:   time.Now(),
	}

	s.transactions[tx.TxID] = tx

	return tx, nil
}

func calculateFee(amount float64, config *FeeConfig) float64 {
	if config.FeePercentage > 0 {
		fee := amount * config.FeePercentage / 100
		if fee < config.MinFee {
			return config.MinFee
		}
		if fee > config.MaxFee {
			return config.MaxFee
		}
		return fee
	}
	return config.FeeAmount
}

// Update fee configuration
func (s *MasterWalletService) UpdateFeeConfig(feeID string, updates map[string]interface{}) (*FeeConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fee, ok := s.feeConfigs[feeID]
	if !ok {
		return nil, fmt.Errorf("fee config not found")
	}

	if feeAmount, ok := updates["fee_amount"].(float64); ok {
		fee.FeeAmount = feeAmount
	}
	if feePercentage, ok := updates["fee_percentage"].(float64); ok {
		fee.FeePercentage = feePercentage
	}
	if minFee, ok := updates["min_fee"].(float64); ok {
		fee.MinFee = minFee
	}
	if maxFee, ok := updates["max_fee"].(float64); ok {
		fee.MaxFee = maxFee
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		fee.IsActive = isActive
	}

	fee.UpdatedAt = time.Now()

	return fee, nil
}

// Add new blockchain
func (s *MasterWalletService) AddBlockchain(config *BlockchainConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.blockchains[config.Chain]; exists {
		return fmt.Errorf("blockchain already exists")
	}

	s.blockchains[config.Chain] = config

	return nil
}

// Update blockchain
func (s *MasterWalletService) UpdateBlockchain(chain ChainType, updates map[string]interface{}) (*BlockchainConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	blockchain, ok := s.blockchains[chain]
	if !ok {
		return nil, fmt.Errorf("blockchain not found")
	}

	if rpcURL, ok := updates["rpc_url"].(string); ok {
		blockchain.RPCURL = rpcURL
	}
	if explorerURL, ok := updates["explorer_url"].(string); ok {
		blockchain.ExplorerURL = explorerURL
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		blockchain.IsActive = isActive
	}

	return blockchain, nil
}

// Add new token
func (s *MasterWalletService) AddToken(token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenID := token.Symbol + "_" + string(token.Chain)
	if _, exists := s.tokens[tokenID]; exists {
		return fmt.Errorf("token already exists")
	}

	token.TokenID = tokenID
	token.Status = "ACTIVE"
	s.tokens[tokenID] = token

	return nil
}

// Get transactions
func (s *MasterWalletService) GetTransactions(walletID, status, chain string, limit, offset int) []*Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txs := make([]*Transaction, 0)
	for _, tx := range s.transactions {
		match := true
		if walletID != "" && tx.WalletID != walletID {
			match = false
		}
		if status != "" && string(tx.Status) != status {
			match = false
		}
		if chain != "" && string(tx.Chain) != chain {
			match = false
		}
		if match {
			txs = append(txs, tx)
		}
	}

	// Apply pagination
	if offset >= len(txs) {
		return []*Transaction{}
	}
	end := offset + limit
	if end > len(txs) {
		end = len(txs)
	}

	return txs[offset:end]
}

// Get master address
func (s *MasterWalletService) GetMasterAddress() string {
	return s.masterAddress.Hex()
}

// Get blockchain configs
func (s *MasterWalletService) GetBlockchains() map[ChainType]*BlockchainConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[ChainType]*BlockchainConfig)
	for k, v := range s.blockchains {
		result[k] = v
	}
	return result
}

// Get tokens
func (s *MasterWalletService) GetTokens(chain string) []*Token {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokens := make([]*Token, 0)
	for _, token := range s.tokens {
		if chain == "" || string(token.Chain) == chain {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

// Get fee configs
func (s *MasterWalletService) GetFeeConfigs() []*FeeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fees := make([]*FeeConfig, 0)
	for _, fee := range s.feeConfigs {
		fees = append(fees, fee)
	}
	return fees
}

// Get wallet stats
func (s *MasterWalletService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_wallets":      len(s.wallets),
		"total_transactions": len(s.transactions),
		"total_tokens":       len(s.tokens),
		"total_blockchains":  len(s.blockchains),
		"total_fees":        len(s.feeConfigs),
		"master_address":    s.masterAddress.Hex(),
	}

	// Count by type
	walletTypes := make(map[string]int)
	walletChains := make(map[string]int)
	txStatuses := make(map[string]int)

	for _, wallet := range s.wallets {
		walletTypes[wallet.Type]++
		walletChains[string(wallet.Chain)]++
	}

	for _, tx := range s.transactions {
		txStatuses[string(tx.Status)]++
	}

	stats["wallets_by_type"] = walletTypes
	stats["wallets_by_chain"] = walletChains
	stats["transactions_by_status"] = txStatuses

	return stats
}

// Handlers
func (s *MasterWalletService) GetMasterAddressHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"master_address": s.GetMasterAddress(),
	})
}

func (s *MasterWalletService) GetWalletsHandler(c *gin.Context) {
	walletType := c.Query("type")
	chain := c.Query("chain")

	wallets := s.GetWallets(walletType, chain)
	c.JSON(http.StatusOK, gin.H{"wallets": wallets})
}

func (s *MasterWalletService) GetWalletHandler(c *gin.Context) {
	walletID := c.Param("id")

	wallet := s.GetWallet(walletID)
	if wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (s *MasterWalletService) CreateWalletHandler(c *gin.Context) {
	var req struct {
		UserID string   `json:"user_id" binding:"required"`
		Chain  ChainType `json:"chain"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Chain == "" {
		req.Chain = ChainEthereum
	}

	wallet, err := s.CreateUserWallet(req.UserID, req.Chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wallet)
}

func (s *MasterWalletService) ProcessWithdrawalHandler(c *gin.Context) {
	var req struct {
		WalletID string `json:"wallet_id" binding:"required"`
		ToAddress string `json:"to_address" binding:"required"`
		Token    string `json:"token" binding:"required"`
		Amount   string `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := s.ProcessWithdrawal(req.WalletID, req.ToAddress, req.Token, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tx)
}

func (s *MasterWalletService) GetTransactionsHandler(c *gin.Context) {
	walletID := c.Query("wallet_id")
	status := c.Query("status")
	chain := c.Query("chain")

	limit := 50
	offset := 0
	fmt.Sscanf(c.DefaultQuery("limit", "50"), "%d", &limit)
	fmt.Sscanf(c.DefaultQuery("offset", "0"), "%d", &offset)

	txs := s.GetTransactions(walletID, status, chain, limit, offset)
	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

func (s *MasterWalletService) GetBlockchainsHandler(c *gin.Context) {
	blockchains := s.GetBlockchains()
	c.JSON(http.StatusOK, gin.H{"blockchains": blockchains})
}

func (s *MasterWalletService) GetTokensHandler(c *gin.Context) {
	chain := c.Query("chain")

	tokens := s.GetTokens(chain)
	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

func (s *MasterWalletService) GetFeeConfigsHandler(c *gin.Context) {
	fees := s.GetFeeConfigs()
	c.JSON(http.StatusOK, gin.H{"fees": fees})
}

func (s *MasterWalletService) GetStatsHandler(c *gin.Context) {
	stats := s.GetStats()
	c.JSON(http.StatusOK, stats)
}

func (s *MasterWalletService) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/master-wallet")
	{
		api.GET("/address", s.GetMasterAddressHandler)
		api.GET("/wallets", s.GetWalletsHandler)
		api.POST("/wallets", s.CreateWalletHandler)
		api.GET("/wallets/:id", s.GetWalletHandler)
		api.POST("/withdraw", s.ProcessWithdrawalHandler)
		api.GET("/transactions", s.GetTransactionsHandler)
		api.GET("/blockchains", s.GetBlockchainsHandler)
		api.GET("/tokens", s.GetTokensHandler)
		api.GET("/fees", s.GetFeeConfigsHandler)
		api.GET("/stats", s.GetStatsHandler)
	}
}

func main() {
	cfg := Config{
		ServerPort:    getEnv("MASTER_WALLET_PORT", "8088"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:    getEnv("JWT_SECRET", "tiger-master-secret-2026"),
		MasterKeyHex: getEnv("MASTER_KEY_HEX", ""),
		HotWalletLimit: 1000000000000000000, // 1 ETH
	}

	service := NewMasterWalletService(cfg)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "master-wallet-service",
			"timestamp": time.Now().Unix(),
		})
	})

	service.SetupRoutes(r)

	addr := ":" + cfg.ServerPort
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("Starting Master Wallet Service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
