/**
 * TigerWallet Gasless Transaction Service - Production-Ready Go Implementation
 * Ultra-low latency, high-throughput meta-transaction relayer
 * Supports gas payment in ERC-20 tokens
 */

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	// Server
	ServerPort string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisHost string
	RedisPort string

	// Blockchain
	EthereumRPC  string
	PolygonRPC   string
	ArbitrumRPC  string
	OptimismRPC  string
	BSCRPC       string
	AvalancheRPC string

	// Relayer
	RelayerPrivateKey string
	RelayerAddress    string

	// Forwarder is the deployed GaslessMetaTx contract that verifies the user
	// EIP-712 signature on-chain. The relayer must call this contract rather
	// than transferring tokens directly, otherwise the user signature is never
	// checked and funds could be moved without consent.
	ForwarderAddress string

	// Name/version must match the EIP-712 domain used by the forwarder.
	ForwarderName    string
	ForwarderVersion string

	// Gas Settings
	GasPriceBufferPercent int   // Extra gas price to ensure inclusion
	MaxGasPriceGwei       int64 // Maximum gas price in gwei
	MinGasPriceGwei       int64 // Minimum gas price in gwei

	// Token Support
	SupportedTokens map[string]TokenConfig

	// Rate Limiting
	MaxRequestsPerMinute int
	MaxTokensPerDay      int64
}

type TokenConfig struct {
	Address          common.Address
	Symbol           string
	Decimals         uint8
	GasUsed          uint64 // Gas cost for transfer
	MinConfirmations uint64
}

func LoadConfig() *Config {
	supportedTokens := map[string]TokenConfig{
		// Ethereum
		"0xa0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48": { // USDC
			Address:  common.HexToAddress("0xa0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
			Symbol:   "USDC",
			Decimals: 6,
			GasUsed:  65000,
		},
		"0x6B175474E89094C44Da98b954EadeAC462271d19": { // DAI
			Address:  common.HexToAddress("0x6B175474E89094C44Da98b954EadeAC462271d19"),
			Symbol:   "DAI",
			Decimals: 18,
			GasUsed:  65000,
		},
		"0xdAC17F958D2ee523a2206206994597C13D831ec7": { // USDT
			Address:  common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
			Symbol:   "USDT",
			Decimals: 6,
			GasUsed:  55000,
		},
		// Polygon
		"0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174": { // USDC Polygon
			Address:  common.HexToAddress("0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"),
			Symbol:   "USDC",
			Decimals: 6,
			GasUsed:  65000,
		},
		// BSC
		"0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd540d": { // USDC BSC
			Address:  common.HexToAddress("0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd540d"),
			Symbol:   "USDC",
			Decimals: 18,
			GasUsed:  65000,
		},
	}

	return &Config{
		ServerPort:            getEnv("GASLESS_PORT", "9201"),
		DBHost:                getEnv("DB_HOST", "localhost"),
		DBPort:                getEnv("DB_PORT", "5432"),
		DBUser:                getEnv("DB_USER", "tigerwallet"),
		DBPassword:            mustEnv("DB_PASSWORD"),
		DBName:                getEnv("DB_NAME", "tigerwallet"),
		RedisHost:             getEnv("REDIS_HOST", "localhost"),
		RedisPort:             getEnv("REDIS_PORT", "6379"),
		EthereumRPC:           getEnv("ETHEREUM_RPC", "https://eth.llamarpc.com"),
		PolygonRPC:            getEnv("POLYGON_RPC", "https://polygon-rpc.com"),
		ArbitrumRPC:           getEnv("ARBITRUM_RPC", "https://arb1.arbitrum.io/rpc"),
		OptimismRPC:           getEnv("OPTIMISM_RPC", "https://mainnet.optimism.io"),
		BSCRPC:                getEnv("BSC_RPC", "https://bsc-dataseed.binance.org"),
		AvalancheRPC:          getEnv("AVALANCHE_RPC", "https://api.avax.network/ext/bc/C/rpc"),
		RelayerPrivateKey:     getEnv("RELAYER_PRIVATE_KEY", ""),
		ForwarderAddress:      getEnv("FORWARDER_ADDRESS", ""),
		ForwarderName:         getEnv("FORWARDER_NAME", "TigerWallet Gasless"),
		ForwarderVersion:      getEnv("FORWARDER_VERSION", "1.0.0"),
		GasPriceBufferPercent: 20,
		MaxGasPriceGwei:       500,
		MinGasPriceGwei:       1,
		SupportedTokens:       supportedTokens,
		MaxRequestsPerMinute:  100,
		MaxTokensPerDay:       1000000,
	}
}

// mustEnv returns the named environment variable or terminates the process.
// A secret has no safe default: a committed fallback would ship a known
// credential to every deployment.
func mustEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	log.Fatalf("%s must be set", key)
	return ""
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

type MetaTransaction struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// User info
	UserAddress  common.Address `gorm:"index" json:"user_address"`
	TokenAddress common.Address `json:"token_address"`
	ChainID      uint64         `json:"chain_id"`

	// Transaction details
	TokenAmount string         `json:"token_amount"`
	ToAddress   string         `json:"to_address"`
	FeeToken    common.Address `json:"fee_token"`
	FeeAmount   string         `json:"fee_amount"`
	GasPrice    string         `json:"gas_price"`
	GasLimit    uint64         `json:"gas_limit"`
	Nonce       uint64         `gorm:"uniqueIndex" json:"nonce"`

	// User's authorization
	UserSignature string `json:"user_signature"`

	// Relayer processing
	RelayerSignature string         `json:"relayer_signature"`
	RelayerAddress   common.Address `json:"relayer_address"`

	// Status
	Status       string `json:"status"` // pending, submitted, confirmed, failed
	TxHash       string `json:"tx_hash"`
	BlockNumber  uint64 `json:"block_number"`
	GasUsed      uint64 `json:"gas_used"`
	ErrorMessage string `json:"error_message"`
}

type UserNonce struct {
	ID          uint           `gorm:"primarykey"`
	UserAddress common.Address `gorm:"uniqueIndex"`
	ChainID     uint64         `gorm:"index"`
	Nonce       uint64         `gorm:"index"`
	LastUpdated time.Time      `json:"last_updated"`
}

type DailyQuota struct {
	ID          uint           `gorm:"primarykey"`
	UserAddress common.Address `gorm:"index"`
	Date        string         `gorm:"index"` // YYYY-MM-DD
	TokenAmount int64          `json:"token_amount"`
}

// ============================================================================
// Services
// ============================================================================

type GaslessService struct {
	config       *Config
	db           *gorm.DB
	redis        *redis.Client
	ethClients   map[uint64]*ethclient.Client
	relayerKey   *ecdsa.PrivateKey
	relayerAddr  common.Address
	nonceManager *NonceManager
	rateLimiter  *RateLimiter
	chainStates  map[uint64]*ChainState
	mu           sync.RWMutex

	// Contract ABIs
	tokenABI     abi.ABI
	forwarderABI abi.ABI

	// Stats
	stats Stats
}

type Stats struct {
	TotalTransactions     int64  `json:"total_transactions"`
	PendingTransactions   int64  `json:"pending_transactions"`
	ConfirmedTransactions int64  `json:"confirmed_transactions"`
	FailedTransactions    int64  `json:"failed_transactions"`
	TotalFeesCollected    string `json:"total_fees_collected"`
}

type ChainState struct {
	chainID      uint64
	client       *ethclient.Client
	gasPrice     *big.Int
	lastUpdated  time.Time
	suggestedFee *FeeSuggestion
}

type FeeSuggestion struct {
	BaseFee        *big.Int
	MaxPriorityFee *big.Int
	MaxFee         *big.Int
	BufferPercent  int
}

type NonceManager struct {
	mu     sync.Mutex
	nonces map[string]uint64 // "chainID:userAddress" -> nonce
}

type RateLimiter struct {
	mu           sync.RWMutex
	requests     map[string][]time.Time
	maxPerMinute int
	maxPerDay    int64
}

func NewGaslessService(config *Config) (*GaslessService, error) {
	// Initialize database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Auto migrate
	err = db.AutoMigrate(&MetaTransaction{}, &UserNonce{}, &DailyQuota{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("Warning: Redis connection failed: %v\n", err)
	}

	// Initialize Ethereum clients
	ethClients := make(map[uint64]*ethclient.Client)
	rpcURLs := map[uint64]string{
		1:     config.EthereumRPC,
		56:    config.BSCRPC,
		137:   config.PolygonRPC,
		42161: config.ArbitrumRPC,
		10:    config.OptimismRPC,
	}

	for chainID, rpcURL := range rpcURLs {
		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			fmt.Printf("Warning: Failed to connect to chain %d: %v\n", chainID, err)
			continue
		}
		ethClients[chainID] = client
	}

	// Initialize relayer key
	var relayerKey *ecdsa.PrivateKey
	var relayerAddr common.Address

	if config.RelayerPrivateKey != "" {
		key, err := crypto.HexToECDSA(config.RelayerPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid relayer private key: %v", err)
		}
		relayerKey = key
		relayerAddr = crypto.PubkeyToAddress(key.PublicKey)
	} else {
		// Generate new key for testing
		key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate relayer key: %v", err)
		}
		relayerKey = key
		relayerAddr = crypto.PubkeyToAddress(key.PublicKey)
	}

	// Load ABIs
	tokenABIJSON := `[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"}]`
	tokenABI, err := abi.JSON(strings.NewReader(tokenABIJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token ABI: %v", err)
	}

	forwarderABIJSON := `[{"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"},{"name":"gasLimit","type":"uint256"},{"name":"relayer","type":"address"},{"name":"signature","type":"bytes"}],"name":"executeMetaTransaction","outputs":[{"name":"","type":"bytes"}],"stateMutability":"payable","type":"function"}]`
	forwarderABI, err := abi.JSON(strings.NewReader(forwarderABIJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse forwarder ABI: %v", err)
	}

	service := &GaslessService{
		config:       config,
		db:           db,
		redis:        redisClient,
		ethClients:   ethClients,
		relayerKey:   relayerKey,
		relayerAddr:  relayerAddr,
		nonceManager: &NonceManager{nonces: make(map[string]uint64)},
		rateLimiter: &RateLimiter{
			requests:     make(map[string][]time.Time),
			maxPerMinute: config.MaxRequestsPerMinute,
			maxPerDay:    config.MaxTokensPerDay,
		},
		chainStates:  make(map[uint64]*ChainState),
		tokenABI:     tokenABI,
		forwarderABI: forwarderABI,
		stats:        Stats{},
	}

	// Start background tasks
	go service.updateGasPrices()
	go service.monitorPendingTransactions()
	go service.cleanupRateLimits()

	return service, nil
}

// ============================================================================
// API Handlers
// ============================================================================

type SendMetaTxRequest struct {
	From      string `json:"from" binding:"required"`
	To        string `json:"to" binding:"required"`
	Token     string `json:"token" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
	FeeToken  string `json:"feeToken" binding:"required"`
	FeeAmount string `json:"feeAmount"`
	ChainID   uint64 `json:"chainId" binding:"required"`
	GasPrice  string `json:"gasPrice"`
	GasLimit  uint64 `json:"gasLimit"`
	Nonce     uint64 `json:"nonce"`
	Signature string `json:"signature" binding:"required"`
}

type SendMetaTxResponse struct {
	TxHash string `json:"txHash"`
	Nonce  uint64 `json:"nonce"`
	Status string `json:"status"`
}

type GetFeeRequest struct {
	Token    string `json:"token" binding:"required"`
	ChainID  uint64 `json:"chainId" binding:"required"`
	GasLimit uint64 `json:"gasLimit"`
}

type GetFeeResponse struct {
	FeeToken     string `json:"feeToken"`
	FeeAmount    string `json:"feeAmount"`
	GasPrice     string `json:"gasPrice"`
	EstimatedGas uint64 `json:"estimatedGas"`
}

type GetNonceResponse struct {
	Nonce uint64 `json:"nonce"`
}

func (s *GaslessService) registerRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		// Meta-transaction endpoints
		api.POST("/meta-tx", s.sendMetaTransaction)
		api.GET("/meta-tx/:txHash", s.getTransactionStatus)
		api.GET("/nonce/:user/:chainId", s.getNonce)
		api.GET("/fee", s.getFee)

		// Stats
		api.GET("/stats", s.getStats)
		api.GET("/relayer-address", s.getRelayerAddress)

		// Health
		api.GET("/health", s.healthCheck)
	}
}

// ============================================================================
// Meta-Transaction Logic
// ============================================================================

func (s *GaslessService) sendMetaTransaction(c *gin.Context) {
	var req SendMetaTxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Validate user address
	userAddr := common.HexToAddress(req.From)
	if userAddr == (common.Address{}) {
		c.JSON(400, gin.H{"error": "invalid from address"})
		return
	}

	// Validate token
	tokenAddr := common.HexToAddress(req.Token)
	if tokenAddr == (common.Address{}) {
		c.JSON(400, gin.H{"error": "invalid token address"})
		return
	}

	// Check rate limit
	if !s.rateLimiter.allow(userAddr.Hex()) {
		c.JSON(429, gin.H{"error": "rate limit exceeded"})
		return
	}

	// Get or initialize nonce
	nonce, err := s.getOrInitNonce(userAddr, req.ChainID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get nonce"})
		return
	}

	// Parse amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		c.JSON(400, gin.H{"error": "invalid amount"})
		return
	}

	// The gas limit is part of the signed payload, so it must be final before
	// the signature is checked.
	if req.GasLimit == 0 {
		req.GasLimit = s.getEstimatedGasLimit(tokenAddr)
	}

	// Calculate fee
	feeAmount := s.calculateFee(req.ChainID, tokenAddr, req.GasLimit)

	// Verify the EIP-712 signature over the exact payload that will be relayed.
	if !s.verifyUserSignature(userAddr, tokenAddr, req.To, amount, req.GasLimit, req.ChainID, nonce, req.Signature) {
		c.JSON(400, gin.H{"error": "invalid signature"})
		return
	}

	// Get gas price
	gasPrice, err := s.getGasPrice(req.ChainID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get gas price"})
		return
	}

	// Create meta transaction record
	tx := &MetaTransaction{
		UserAddress:   userAddr,
		TokenAddress:  tokenAddr,
		ToAddress:     req.To,
		ChainID:       req.ChainID,
		TokenAmount:   req.Amount,
		FeeToken:      tokenAddr,
		FeeAmount:     feeAmount.String(),
		GasPrice:      gasPrice.String(),
		GasLimit:      req.GasLimit,
		Nonce:         nonce,
		UserSignature: req.Signature,
		Status:        "pending",
	}

	if err := s.db.Create(tx).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to create transaction"})
		return
	}

	// Execute transaction
	go s.executeMetaTransaction(tx)

	s.stats.TotalTransactions++

	c.JSON(200, SendMetaTxResponse{
		TxHash: fmt.Sprintf("pending-%d", tx.ID),
		Nonce:  nonce,
		Status: "pending",
	})
}

func (s *GaslessService) executeMetaTransaction(tx *MetaTransaction) {
	s.mu.Lock()
	chainState, ok := s.chainStates[tx.ChainID]
	s.mu.Unlock()

	if !ok {
		tx.Status = "failed"
		tx.ErrorMessage = "chain not supported"
		s.db.Save(tx)
		return
	}

	forwarder := common.HexToAddress(s.config.ForwarderAddress)
	if forwarder == (common.Address{}) {
		tx.Status = "failed"
		tx.ErrorMessage = "FORWARDER_ADDRESS not configured; refusing to relay unverified meta-transaction"
		s.db.Save(tx)
		return
	}

	// The user's signed intent is an ERC-20 transfer of the token to the
	// recipient, executed by the forwarder after it verifies the signature.
	tokenAmount, ok := new(big.Int).SetString(tx.TokenAmount, 10)
	if !ok {
		tx.Status = "failed"
		tx.ErrorMessage = "invalid token amount"
		s.db.Save(tx)
		return
	}
	recipient := common.HexToAddress(tx.ToAddress)
	innerCallData, err := s.tokenABI.Pack("transfer", recipient, tokenAmount)
	if err != nil {
		tx.Status = "failed"
		tx.ErrorMessage = err.Error()
		s.db.Save(tx)
		return
	}

	signature, err := hex.DecodeString(strings.TrimPrefix(tx.UserSignature, "0x"))
	if err != nil {
		tx.Status = "failed"
		tx.ErrorMessage = "invalid user signature encoding"
		s.db.Save(tx)
		return
	}

	callData, err := s.forwarderABI.Pack(
		"executeMetaTransaction",
		common.HexToAddress(tx.UserAddress.Hex()),
		tx.TokenAddress,
		big.NewInt(0), // value: meta-transactions here never move native currency
		innerCallData,
		new(big.Int).SetUint64(tx.GasLimit),
		s.relayerAddr,
		signature,
	)
	if err != nil {
		tx.Status = "failed"
		tx.ErrorMessage = err.Error()
		s.db.Save(tx)
		return
	}

	// Get nonce for relayer
	relayerNonce, err := s.getRelayerNonce(tx.ChainID)
	if err != nil {
		tx.Status = "failed"
		tx.ErrorMessage = "failed to get relayer nonce"
		s.db.Save(tx)
		return
	}

	// Create transaction
	chainID := big.NewInt(int64(tx.ChainID))
	txData := types.DynamicFeeTx{
		To:        &forwarder,
		Nonce:     relayerNonce,
		Gas:       tx.GasLimit,
		GasFeeCap: new(big.Int).Mul(gasPriceToBigInt(tx.GasPrice), big.NewInt(2)),
		GasTipCap: big.NewInt(1e9), // 1 gwei tip
		Value:     big.NewInt(0),
		Data:      callData,
	}

	txn := types.NewTx(&txData)
	signedTx, err := types.SignTx(txn, types.NewLondonSigner(chainID), s.relayerKey)
	if err != nil {
		tx.Status = "failed"
		tx.ErrorMessage = err.Error()
		s.db.Save(tx)
		return
	}

	// Submit transaction
	err = chainState.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		tx.Status = "failed"
		tx.ErrorMessage = err.Error()
		s.db.Save(tx)
		return
	}

	tx.TxHash = signedTx.Hash().Hex()
	tx.RelayerAddress = s.relayerAddr
	tx.Status = "submitted"
	s.db.Save(tx)

	// Update stats
	atomic.AddInt64(&s.stats.PendingTransactions, 1)

	// Wait for confirmation
	go s.waitForConfirmation(tx)
}

func (s *GaslessService) waitForConfirmation(tx *MetaTransaction) {
	s.mu.RLock()
	_, ok := s.chainStates[tx.ChainID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	// Wait for receipt
	receipt, err := s.waitForReceipt(tx.ChainID, common.HexToHash(tx.TxHash))
	if err != nil {
		tx.Status = "failed"
		tx.ErrorMessage = err.Error()
		s.db.Save(tx)
		return
	}

	tx.BlockNumber = receipt.BlockNumber.Uint64()
	tx.GasUsed = receipt.GasUsed

	if receipt.Status == 1 {
		tx.Status = "confirmed"
		atomic.AddInt64(&s.stats.PendingTransactions, -1)
		atomic.AddInt64(&s.stats.ConfirmedTransactions, 1)
	} else {
		tx.Status = "failed"
		atomic.AddInt64(&s.stats.PendingTransactions, -1)
		atomic.AddInt64(&s.stats.FailedTransactions, 1)
	}

	s.db.Save(tx)
}

func (s *GaslessService) waitForReceipt(chainID uint64, txHash common.Hash) (*types.Receipt, error) {
	ctx := context.Background()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			chainState, ok := s.chainStates[chainID]
			s.mu.RUnlock()

			if !ok {
				return nil, fmt.Errorf("chain not found")
			}

			receipt, err := chainState.client.TransactionReceipt(ctx, txHash)
			if err == nil {
				return receipt, nil
			}

			if err != ethereum.NotFound {
				return nil, err
			}

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for receipt")
		}
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *GaslessService) getOrInitNonce(userAddr common.Address, chainID uint64) (uint64, error) {
	key := fmt.Sprintf("%d:%s", chainID, userAddr.Hex())

	s.nonceManager.mu.Lock()
	defer s.nonceManager.mu.Unlock()

	if nonce, ok := s.nonceManager.nonces[key]; ok {
		s.nonceManager.nonces[key] = nonce + 1
		return nonce, nil
	}

	// Get from database
	var userNonce UserNonce
	err := s.db.Where("user_address = ? AND chain_id = ?", userAddr.Hex(), chainID).First(&userNonce).Error
	if err == gorm.ErrRecordNotFound {
		nonce := uint64(0)
		s.nonceManager.nonces[key] = nonce + 1

		// Create new record
		s.db.Create(&UserNonce{
			UserAddress: userAddr,
			ChainID:     chainID,
			Nonce:       nonce + 1,
			LastUpdated: time.Now(),
		})

		return nonce, nil
	} else if err != nil {
		return 0, err
	}

	s.nonceManager.nonces[key] = userNonce.Nonce + 1
	return userNonce.Nonce, nil
}

func (s *GaslessService) getRelayerNonce(chainID uint64) (uint64, error) {
	s.mu.RLock()
	chainState, ok := s.chainStates[chainID]
	s.mu.RUnlock()

	if !ok {
		return 0, fmt.Errorf("chain not supported")
	}

	nonce, err := chainState.client.NonceAt(context.Background(), s.relayerAddr, nil)
	return nonce, err
}

func (s *GaslessService) getGasPrice(chainID uint64) (*big.Int, error) {
	s.mu.RLock()
	chainState, ok := s.chainStates[chainID]
	s.mu.RUnlock()

	if !ok {
		return big.NewInt(0), fmt.Errorf("chain not supported")
	}

	gasPrice := chainState.gasPrice

	// Apply buffer
	buffer := big.NewInt(int64(s.config.GasPriceBufferPercent))
	gasPrice = new(big.Int).Mul(gasPrice, buffer.Add(buffer, big.NewInt(100)))
	gasPrice = new(big.Int).Div(gasPrice, big.NewInt(100))

	// Apply limits
	maxGasPrice := big.NewInt(s.config.MaxGasPriceGwei * 1e9)
	minGasPrice := big.NewInt(s.config.MinGasPriceGwei * 1e9)

	if gasPrice.Cmp(maxGasPrice) > 0 {
		gasPrice = maxGasPrice
	}
	if gasPrice.Cmp(minGasPrice) < 0 {
		gasPrice = minGasPrice
	}

	return gasPrice, nil
}

func (s *GaslessService) calculateFee(chainID uint64, tokenAddr common.Address, gasLimit uint64) *big.Int {
	if gasLimit == 0 {
		gasLimit = 65000 // Default gas limit
	}

	tokenConfig, ok := s.config.SupportedTokens[strings.ToLower(tokenAddr.Hex())]
	if !ok {
		tokenConfig = TokenConfig{GasUsed: 65000}
	}

	gasPrice := big.NewInt(1e9) // 1 gwei default

	fee := new(big.Int).Mul(big.NewInt(int64(tokenConfig.GasUsed)), gasPrice)

	return fee
}

func (s *GaslessService) getEstimatedGasLimit(tokenAddr common.Address) uint64 {
	tokenConfig, ok := s.config.SupportedTokens[strings.ToLower(tokenAddr.Hex())]
	if !ok {
		return 65000
	}
	return tokenConfig.GasUsed
}

// metaTxTypehash is keccak256 of the EIP-712 MetaTransaction struct used by
// the deployed forwarder contract. It must stay byte-identical to the string
// in GaslessMetaTx.sol or every signature will be rejected.
var metaTxTypehash = crypto.Keccak256Hash([]byte(
	"MetaTransaction(address from,address to,uint256 value,bytes data,uint256 gasLimit,address relayer,uint256 nonce)"))

var eip712DomainTypehash = crypto.Keccak256Hash([]byte(
	"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))

// domainSeparator mirrors the contract's DOMAIN_SEPARATOR so the digest the
// relayer recovers from is exactly the one the user signed.
func (s *GaslessService) domainSeparator(chainID uint64) common.Hash {
	return crypto.Keccak256Hash(
		eip712DomainTypehash.Bytes(),
		crypto.Keccak256Hash([]byte(s.config.ForwarderName)).Bytes(),
		crypto.Keccak256Hash([]byte(s.config.ForwarderVersion)).Bytes(),
		common.LeftPadBytes(new(big.Int).SetUint64(chainID).Bytes(), 32),
		common.LeftPadBytes(common.HexToAddress(s.config.ForwarderAddress).Bytes(), 32),
	)
}

// metaTxDigest builds the EIP-712 digest the user signs. It is shared by the
// verifier and by tests so both always agree on the exact signed payload.
func (s *GaslessService) metaTxDigest(
	userAddr common.Address,
	tokenAddr common.Address,
	recipient common.Address,
	amount *big.Int,
	gasLimit uint64,
	chainID uint64,
	nonce uint64,
) common.Hash {
	callData, err := s.tokenABI.Pack("transfer", recipient, amount)
	if err != nil {
		// The ABI is fixed at build time, so an encoding failure here is a
		// programming error; return an unusable digest rather than a guess.
		return common.Hash{}
	}

	structHash := crypto.Keccak256Hash(
		metaTxTypehash.Bytes(),
		common.LeftPadBytes(userAddr.Bytes(), 32),
		common.LeftPadBytes(recipient.Bytes(), 32),
		common.LeftPadBytes(big.NewInt(0).Bytes(), 32), // value
		crypto.Keccak256Hash(callData).Bytes(),
		common.LeftPadBytes(new(big.Int).SetUint64(gasLimit).Bytes(), 32),
		common.LeftPadBytes(s.relayerAddr.Bytes(), 32),
		common.LeftPadBytes(new(big.Int).SetUint64(nonce).Bytes(), 32),
	)

	return crypto.Keccak256Hash(
		[]byte{0x19, 0x01},
		s.domainSeparator(chainID).Bytes(),
		structHash.Bytes(),
	)
}

// verifyUserSignature recovers the EIP-712 signer of a meta-transaction and
// requires it to match the claimed user. Returning true for any input would
// let an attacker relay a transfer for an address they do not control, so this
// fails closed whenever the forwarder is unconfigured or recovery mismatches.
func (s *GaslessService) verifyUserSignature(
	userAddr common.Address,
	tokenAddr common.Address,
	to string,
	amount *big.Int,
	gasLimit uint64,
	chainID uint64,
	nonce uint64,
	signature string,
) bool {
	forwarder := common.HexToAddress(s.config.ForwarderAddress)
	if forwarder == (common.Address{}) {
		log.Println("gasless: FORWARDER_ADDRESS not configured; rejecting meta-transaction")
		return false
	}
	if to == "" {
		return false
	}

	// The forwarder executes a call to `to` with the encoded calldata. The
	// relayer builds an ERC-20 transfer of `amount` to the recipient, which is
	// what the user signs over.
	recipient := common.HexToAddress(to)
	digest := s.metaTxDigest(userAddr, tokenAddr, recipient, amount, gasLimit, chainID, nonce)
	if digest == (common.Hash{}) {
		return false
	}

	sig, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	if err != nil || len(sig) != 65 {
		return false
	}
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	pubKey, err := crypto.SigToPub(digest.Bytes(), sig)
	if err != nil {
		return false
	}
	return crypto.PubkeyToAddress(*pubKey) == userAddr
}

func gasPriceToBigInt(gasPrice string) *big.Int {
	price, ok := new(big.Int).SetString(gasPrice, 10)
	if !ok {
		return big.NewInt(1e9)
	}
	return price
}

// ============================================================================
// Background Tasks
// ============================================================================

func (s *GaslessService) updateGasPrices() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for chainID, client := range s.ethClients {
			gasPrice, err := client.SuggestGasPrice(context.Background())
			if err != nil {
				fmt.Printf("Failed to get gas price for chain %d: %v\n", chainID, err)
				continue
			}

			s.mu.Lock()
			if state, ok := s.chainStates[chainID]; ok {
				state.gasPrice = gasPrice
				state.lastUpdated = time.Now()
			} else {
				s.chainStates[chainID] = &ChainState{
					chainID:     chainID,
					client:      client,
					gasPrice:    gasPrice,
					lastUpdated: time.Now(),
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *GaslessService) monitorPendingTransactions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var pending []MetaTransaction
		s.db.Where("status = ?", "submitted").Find(&pending)

		for _, tx := range pending {
			go s.waitForConfirmation(&tx)
		}
	}
}

func (s *GaslessService) cleanupRateLimits() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.rateLimiter.mu.Lock()
		now := time.Now()
		oneDayAgo := now.Add(-24 * time.Hour)

		for user, times := range s.rateLimiter.requests {
			var validTimes []time.Time
			for _, t := range times {
				if t.After(oneDayAgo) {
					validTimes = append(validTimes, t)
				}
			}
			if len(validTimes) == 0 {
				delete(s.rateLimiter.requests, user)
			} else {
				s.rateLimiter.requests[user] = validTimes
			}
		}
		s.rateLimiter.mu.Unlock()
	}
}

func (s *RateLimiter) allow(user string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)

	times, ok := s.requests[user]
	if !ok {
		s.requests[user] = []time.Time{now}
		return true
	}

	var validTimes []time.Time
	minuteCount := 0
	dayCount := 0
	oneDayAgo := time.Now().Add(-24 * time.Hour)

	for _, t := range times {
		if t.After(oneMinuteAgo) {
			minuteCount++
		}
		if t.After(oneDayAgo) {
			dayCount++
			validTimes = append(validTimes, t)
		}
	}

	if minuteCount >= s.maxPerMinute {
		return false
	}

	if int64(dayCount) >= s.maxPerDay {
		return false
	}

	s.requests[user] = append(validTimes, now)
	return true
}

// ============================================================================
// Stats and Health
// ============================================================================

func (s *GaslessService) getStats(c *gin.Context) {
	s.mu.RLock()
	stats := s.stats
	s.mu.RUnlock()

	c.JSON(200, stats)
}

func (s *GaslessService) getRelayerAddress(c *gin.Context) {
	c.JSON(200, gin.H{
		"address": s.relayerAddr.Hex(),
	})
}

func (s *GaslessService) getNonce(c *gin.Context) {
	user := c.Param("user")
	chainIDStr := c.Param("chainId")

	chainID, err := strconv.ParseUint(chainIDStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid chain id"})
		return
	}

	userAddr := common.HexToAddress(user)
	nonce, err := s.getOrInitNonce(userAddr, chainID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, GetNonceResponse{Nonce: nonce})
}

func (s *GaslessService) getFee(c *gin.Context) {
	var req GetFeeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	tokenAddr := common.HexToAddress(req.Token)
	gasLimit := req.GasLimit
	if gasLimit == 0 {
		gasLimit = s.getEstimatedGasLimit(tokenAddr)
	}

	feeAmount := s.calculateFee(req.ChainID, tokenAddr, gasLimit)
	gasPrice, _ := s.getGasPrice(req.ChainID)

	c.JSON(200, GetFeeResponse{
		FeeToken:     req.Token,
		FeeAmount:    feeAmount.String(),
		GasPrice:     gasPrice.String(),
		EstimatedGas: gasLimit,
	})
}

func (s *GaslessService) getTransactionStatus(c *gin.Context) {
	txHash := c.Param("txHash")

	var tx MetaTransaction
	if err := s.db.Where("tx_hash = ?", txHash).First(&tx).Error; err != nil {
		c.JSON(404, gin.H{"error": "transaction not found"})
		return
	}

	c.JSON(200, tx)
}

func (s *GaslessService) healthCheck(c *gin.Context) {
	checks := make(map[string]string)

	for chainID, client := range s.ethClients {
		if _, err := client.ChainID(context.Background()); err == nil {
			checks[fmt.Sprintf("chain_%d", chainID)] = "ok"
		} else {
			checks[fmt.Sprintf("chain_%d", chainID)] = "error"
		}
	}

	allOK := true
	for _, status := range checks {
		if status != "ok" {
			allOK = false
			break
		}
	}

	if allOK {
		c.JSON(200, gin.H{"status": "healthy", "checks": checks})
	} else {
		c.JSON(503, gin.H{"status": "unhealthy", "checks": checks})
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewGaslessService(config)
	if err != nil {
		fmt.Printf("Failed to initialize service: %v\n", err)
		os.Exit(1)
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	service.registerRoutes(r)

	// Start server
	go func() {
		addr := fmt.Sprintf(":%s", config.ServerPort)
		fmt.Printf("Gasless Transaction Service starting on %s\n", addr)
		fmt.Printf("Relayer address: %s\n", service.relayerAddr.Hex())
		if err := r.Run(addr); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}
