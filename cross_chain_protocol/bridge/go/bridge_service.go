// TigerWallet Bridge Aggregator Service
// High-Load Distributed Go Implementation
// Supports LayerZero, Multichain, and Axelar for cross-chain bridging

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort string `json:"server_port"`
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	RedisHost  string `json:"redis_host"`
	RedisPort  string `json:"redis_port"`
}

// ============================================================================
// Data Models
// ============================================================================

// BridgeTransaction represents a cross-chain bridge transaction
type BridgeTransaction struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	TransactionID string     `gorm:"uniqueIndex" json:"transaction_id"`
	UserAddress   string     `gorm:"index" json:"user_address"`
	FromChain     int64      `json:"from_chain"`
	ToChain       int64      `json:"to_chain"`
	FromToken     string     `json:"from_token"`
	ToToken       string     `json:"to_token"`
	FromAmount    string     `json:"from_amount"`
	ToAmount      string     `json:"to_amount"`
	Bridge        string     `json:"bridge"` // LAYERZERO, MULTICHAIN, AXELAR
	Status        string     `json:"status"` // PENDING, CONFIRMED, COMPLETED, FAILED
	FromTxHash    string     `json:"from_tx_hash"`
	ToTxHash      string     `json:"to_tx_hash"`
	EstimatedTime int64      `json:"estimated_time"` // seconds
	Fee           string     `json:"fee"`
	FeeToken      string     `json:"fee_token"`
	ChainID       int64      `json:"chain_id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at"`
}

// BridgeRoute represents available bridge route
type BridgeRoute struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	FromChain     int64   `json:"from_chain"`
	ToChain       int64   `json:"to_chain"`
	FromToken     string  `json:"from_token"`
	ToToken       string  `json:"to_token"`
	Bridge        string  `json:"bridge"`
	Rate          float64 `json:"rate"`           // exchange rate
	Fee           float64 `json:"fee"`            // bridge fee in USD
	FeePercentage float64 `json:"fee_percentage"` // fee as percentage
	EstimatedTime int64   `json:"estimated_time"` // seconds
	MinAmount     float64 `json:"min_amount"`
	MaxAmount     float64 `json:"max_amount"`
	IsActive      bool    `json:"is_active"`
	ChainID       int64   `json:"chain_id"`
}

// SupportedChain represents a supported blockchain
type SupportedChain struct {
	ID              int64             `gorm:"primaryKey" json:"id"`
	Name            string            `json:"name"`
	Symbol          string            `json:"symbol"`
	ChainID         int64             `json:"chain_id"`
	RPCURL          string            `json:"rpc_url"`
	ExplorerURL     string            `json:"explorer_url"`
	BridgeContracts map[string]string `json:"bridge_contracts"`
	IsActive        bool              `json:"is_active"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type BridgeService struct {
	db            *gorm.DB
	redis         *redis.Client
	config        Config
	bridgeClients map[string]BridgeClient
	mu            sync.RWMutex
}

// BridgeClient interface for different bridge protocols
type BridgeClient interface {
	GetQuote(ctx context.Context, fromChain, toChain int64, token string, amount string) (*BridgeQuote, error)
	ExecuteBridge(ctx context.Context, quote *BridgeQuote, userAddress string) (string, error)
	GetTransactionStatus(ctx context.Context, txHash string) (string, error)
}

type BridgeQuote struct {
	Bridge        string
	FromChain     int64
	ToChain       int64
	FromToken     string
	ToToken       string
	FromAmount    string
	ToAmount      string
	Fee           string
	FeeToken      string
	EstimatedTime int64
	FromTxHash    string
	ToTxHash      string
}

func NewBridgeService(config Config) (*BridgeService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&BridgeTransaction{},
		&BridgeRoute{},
		&SupportedChain{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &BridgeService{
		db:            db,
		redis:         rdb,
		config:        config,
		bridgeClients: make(map[string]BridgeClient),
	}

	// Initialize bridge clients
	service.bridgeClients["LAYERZERO"] = NewLayerZeroClient()
	service.bridgeClients["MULTICHAIN"] = NewMultichainClient()
	service.bridgeClients["AXELAR"] = NewAxelarClient()

	// Initialize supported chains
	go service.initializeChains()
	go service.initializeRoutes()

	return service, nil
}

func (s *BridgeService) initializeChains() {
	chains := []SupportedChain{
		{ID: 1, Name: "Ethereum", Symbol: "ETH", ChainID: 1, RPCURL: "https://eth.llamarpc.com", ExplorerURL: "https://etherscan.io", IsActive: true},
		{ID: 56, Name: "BNB Smart Chain", Symbol: "BNB", ChainID: 56, RPCURL: "https://bsc-dataseed.binance.org", ExplorerURL: "https://bscscan.com", IsActive: true},
		{ID: 137, Name: "Polygon", Symbol: "MATIC", ChainID: 137, RPCURL: "https://polygon-rpc.com", ExplorerURL: "https://polygonscan.com", IsActive: true},
		{ID: 42161, Name: "Arbitrum", Symbol: "ETH", ChainID: 42161, RPCURL: "https://arb1.arbitrum.io/rpc", ExplorerURL: "https://arbiscan.io", IsActive: true},
		{ID: 10, Name: "Optimism", Symbol: "ETH", ChainID: 10, RPCURL: "https://mainnet.optimism.io", ExplorerURL: "https://optimistic.etherscan.io", IsActive: true},
		{ID: 8453, Name: "Base", Symbol: "ETH", ChainID: 8453, RPCURL: "https://mainnet.base.org", ExplorerURL: "https://basescan.org", IsActive: true},
		{ID: 43114, Name: "Avalanche", Symbol: "AVAX", ChainID: 43114, RPCURL: "https://api.avax.network/ext/bc/C/rpc", ExplorerURL: "https://snowtrace.io", IsActive: true},
		{ID: 25, Name: "Cronos", Symbol: "CRO", ChainID: 25, RPCURL: "https://rpc.cronos.org", ExplorerURL: "https://cronoscan.com", IsActive: true},
		{ID: 42220, Name: "Celo", Symbol: "CELO", ChainID: 42220, RPCURL: "https://forno.celo.org", ExplorerURL: "https://explorer.celo.org", IsActive: true},
		{ID: 1284, Name: "Moonbeam", Symbol: "GLMR", ChainID: 1284, RPCURL: "https://rpc.api.moonbeam.network", ExplorerURL: "https://moonscan.io", IsActive: true},
	}

	for _, chain := range chains {
		var existing SupportedChain
		if s.db.First(&existing, chain.ID).RowsAffected == 0 {
			s.db.Create(&chain)
		}
	}
}

func (s *BridgeService) initializeRoutes() {
	// Initialize popular routes
	routes := []BridgeRoute{
		// LayerZero routes
		{FromChain: 1, ToChain: 42161, FromToken: "ETH", ToToken: "ETH", Bridge: "LAYERZERO", Rate: 1.0, Fee: 5.0, FeePercentage: 0.001, EstimatedTime: 900, MinAmount: 10, MaxAmount: 1000000, IsActive: true, ChainID: 1},
		{FromChain: 1, ToChain: 10, FromToken: "ETH", ToToken: "ETH", Bridge: "LAYERZERO", Rate: 1.0, Fee: 5.0, FeePercentage: 0.001, EstimatedTime: 600, MinAmount: 10, MaxAmount: 1000000, IsActive: true, ChainID: 1},
		{FromChain: 1, ToChain: 8453, FromToken: "ETH", ToToken: "ETH", Bridge: "LAYERZERO", Rate: 1.0, Fee: 3.0, FeePercentage: 0.001, EstimatedTime: 300, MinAmount: 10, MaxAmount: 1000000, IsActive: true, ChainID: 1},
		{FromChain: 42161, ToChain: 1, FromToken: "ETH", ToToken: "ETH", Bridge: "LAYERZERO", Rate: 1.0, Fee: 5.0, FeePercentage: 0.001, EstimatedTime: 900, MinAmount: 10, MaxAmount: 1000000, IsActive: true, ChainID: 1},

		// Multichain routes
		{FromChain: 1, ToChain: 56, FromToken: "ETH", ToToken: "ETH", Bridge: "MULTICHAIN", Rate: 0.998, Fee: 8.0, FeePercentage: 0.002, EstimatedTime: 1800, MinAmount: 20, MaxAmount: 500000, IsActive: true, ChainID: 1},
		{FromChain: 56, ToChain: 1, FromToken: "BNB", ToToken: "BNB", Bridge: "MULTICHAIN", Rate: 0.998, Fee: 0.5, FeePercentage: 0.001, EstimatedTime: 1800, MinAmount: 0.01, MaxAmount: 10000, IsActive: true, ChainID: 1},
		{FromChain: 1, ToChain: 137, FromToken: "USDT", ToToken: "USDT", Bridge: "MULTICHAIN", Rate: 1.0, Fee: 1.0, FeePercentage: 0.0001, EstimatedTime: 600, MinAmount: 100, MaxAmount: 1000000, IsActive: true, ChainID: 1},

		// Axelar routes
		{FromChain: 1, ToChain: 43114, FromToken: "ETH", ToToken: "ETH", Bridge: "AXELAR", Rate: 0.997, Fee: 10.0, FeePercentage: 0.002, EstimatedTime: 1200, MinAmount: 50, MaxAmount: 500000, IsActive: true, ChainID: 1},
		{FromChain: 1, ToChain: 25, FromToken: "USDC", ToToken: "USDC", Bridge: "AXELAR", Rate: 1.0, Fee: 2.0, FeePercentage: 0.0002, EstimatedTime: 900, MinAmount: 50, MaxAmount: 1000000, IsActive: true, ChainID: 1},
	}

	for _, route := range routes {
		var existing BridgeRoute
		query := s.db.Where("from_chain = ? AND to_chain = ? AND from_token = ? AND bridge = ?",
			route.FromChain, route.ToChain, route.FromToken, route.Bridge)
		if query.First(&existing).RowsAffected == 0 {
			s.db.Create(&route)
		}
	}
}

// ============================================================================
// Quote Operations
// ============================================================================

type QuoteRequest struct {
	FromChain int64  `json:"from_chain" binding:"required"`
	ToChain   int64  `json:"to_chain" binding:"required"`
	FromToken string `json:"from_token" binding:"required"`
	ToToken   string `json:"to_token"`
	Amount    string `json:"amount" binding:"required"`
}

type QuoteResponse struct {
	Quotes    []BridgeQuote `json:"quotes"`
	BestQuote BridgeQuote   `json:"best_quote"` // lowest fee + fastest
}

func (s *BridgeService) GetQuotes(ctx *gin.Context) {
	var req QuoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Find available routes
	var routes []BridgeRoute
	s.db.Where("from_chain = ? AND to_chain = ? AND from_token = ? AND is_active = ?",
		req.FromChain, req.ToChain, req.FromToken, true).Find(&routes)

	if len(routes) == 0 {
		ctx.JSON(404, gin.H{"error": "No bridge routes available for this pair"})
		return
	}

	// Get quotes from each bridge
	quotes := make([]BridgeQuote, 0, len(routes))
	for _, route := range routes {
		client, ok := s.bridgeClients[route.Bridge]
		if !ok {
			continue
		}

		quote, err := client.GetQuote(ctx.Request.Context(), req.FromChain, req.ToChain, req.FromToken, req.Amount)
		if err != nil {
			continue
		}

		quotes = append(quotes, *quote)
	}

	if len(quotes) == 0 {
		ctx.JSON(404, gin.H{"error": "Failed to get quotes from any bridge"})
		return
	}

	// Find best quote (lowest total cost + fastest)
	bestQuote := quotes[0]
	minScore := quotes[0].EstimatedTime + s.calculateFeeScore(&quotes[0])

	for i := range quotes {
		score := quotes[i].EstimatedTime + s.calculateFeeScore(&quotes[i])
		if score < minScore {
			minScore = score
			bestQuote = quotes[i]
		}
	}

	ctx.JSON(200, QuoteResponse{
		Quotes:    quotes,
		BestQuote: bestQuote,
	})
}

func (s *BridgeService) calculateFeeScore(quote *BridgeQuote) int64 {
	// Convert fee to time equivalent (higher fee = higher score)
	feeFloat := 0.0
	fmt.Sscanf(quote.Fee, "%f", &feeFloat)
	return int64(feeFloat * 100) // $1 fee = 100 seconds added to score
}

// ============================================================================
// Bridge Execution
// ============================================================================

type ExecuteRequest struct {
	UserAddress string `json:"user_address" binding:"required"`
	FromChain   int64  `json:"from_chain" binding:"required"`
	ToChain     int64  `json:"to_chain" binding:"required"`
	FromToken   string `json:"from_token" binding:"required"`
	ToToken     string `json:"to_token"`
	Amount      string `json:"amount" binding:"required"`
	Bridge      string `json:"bridge" binding:"required"`
}

type ExecuteResponse struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id"`
	FromTxHash    string `json:"from_tx_hash"`
	EstimatedTime int64  `json:"estimated_time"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
}

func (s *BridgeService) ExecuteBridge(ctx *gin.Context) {
	var req ExecuteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, ExecuteResponse{Success: false, Error: err.Error()})
		return
	}

	client, ok := s.bridgeClients[req.Bridge]
	if !ok {
		ctx.JSON(400, ExecuteResponse{Success: false, Error: "Invalid bridge"})
		return
	}

	quote, err := client.GetQuote(ctx.Request.Context(), req.FromChain, req.ToChain, req.FromToken, req.Amount)
	if err != nil {
		ctx.JSON(502, ExecuteResponse{Success: false, Error: "Failed to get quote: " + err.Error()})
		return
	}

	// The bridge adapter cannot move the user's funds: the deposit
	// transaction must be signed by whoever holds them. Returning success
	// here with a fabricated hash would report a transfer that never
	// happened, so the caller is told to sign and broadcast instead.
	txHash, err := client.ExecuteBridge(ctx.Request.Context(), quote, req.UserAddress)
	if err != nil {
		if errors.Is(err, ErrExecutionRequiresSignature) {
			ctx.JSON(http.StatusAccepted, ExecuteResponse{
				Success: false,
				Status:  "SIGNATURE_REQUIRED",
				Error:   "the bridge deposit must be signed and broadcast by the caller; no funds have moved",
			})
			return
		}
		ctx.JSON(502, ExecuteResponse{Success: false, Error: err.Error()})
		return
	}

	// The adapter broadcast the deposit, so the hash it returned identifies
	// the transfer. Discarding it would leave the caller unable to track the
	// deposit.
	ctx.JSON(http.StatusOK, ExecuteResponse{
		Success:       true,
		TransactionID: txHash,
		FromTxHash:    txHash,
		EstimatedTime: quote.EstimatedTime,
		Status:        "SUBMITTED",
	})
}

// ============================================================================
// Transaction Status
// ============================================================================

func (s *BridgeService) GetTransactionStatus(ctx *gin.Context) {
	txID := ctx.Param("id")

	var transaction BridgeTransaction
	if err := s.db.Where("transaction_id = ?", txID).First(&transaction).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "Transaction not found"})
		return
	}

	// If completed or failed, return directly
	if transaction.Status == "COMPLETED" || transaction.Status == "FAILED" {
		ctx.JSON(200, gin.H{
			"transaction_id": transaction.TransactionID,
			"status":         transaction.Status,
			"from_tx_hash":   transaction.FromTxHash,
			"to_tx_hash":     transaction.ToTxHash,
		})
		return
	}

	// Check status via bridge client
	client, ok := s.bridgeClients[transaction.Bridge]
	if ok {
		status, err := client.GetTransactionStatus(ctx.Request.Context(), transaction.FromTxHash)
		if err == nil && status != "" {
			transaction.Status = status
			if status == "COMPLETED" {
				now := time.Now()
				transaction.CompletedAt = &now
			}
			s.db.Save(&transaction)
		}
	}

	ctx.JSON(200, gin.H{
		"transaction_id": transaction.TransactionID,
		"status":         transaction.Status,
		"from_tx_hash":   transaction.FromTxHash,
		"to_tx_hash":     transaction.ToTxHash,
		"updated_at":     transaction.UpdatedAt,
	})
}

// ============================================================================
// Supported Chains & Routes
// ============================================================================

func (s *BridgeService) GetChains(ctx *gin.Context) {
	var chains []SupportedChain
	s.db.Where("is_active = ?", true).Find(&chains)

	ctx.JSON(200, gin.H{"chains": chains})
}

func (s *BridgeService) GetRoutes(ctx *gin.Context) {
	fromChain := ctx.Query("from_chain")
	toChain := ctx.Query("to_chain")

	query := s.db.Where("is_active = ?", true)

	if fromChain != "" {
		query = query.Where("from_chain = ?", fromChain)
	}
	if toChain != "" {
		query = query.Where("to_chain = ?", toChain)
	}

	var routes []BridgeRoute
	query.Find(&routes)

	ctx.JSON(200, gin.H{"routes": routes})
}

// ============================================================================
// Bridge Clients
//
// A bridge adapter cannot settle a transfer on its own: the deposit transaction
// must be signed by the holder of the funds. These adapters return a real quote
// derived from the protocol's on-chain state and refuse to invent a settlement.
// ExecuteBridge reports that a client-signed transaction is required, and
// GetTransactionStatus reports only the status it can observe on the node.
// ============================================================================

// ErrExecutionRequiresSignature signals that the adapter cannot move funds by
// itself and the caller must sign and broadcast the deposit transaction.
var ErrExecutionRequiresSignature = errors.New("bridge execution requires a client-signed transaction")

type LayerZeroClient struct {
	rpcURL string
}

func NewLayerZeroClient() *LayerZeroClient {
	return &LayerZeroClient{rpcURL: getEnv("LAYERZERO_RPC_URL", "https://eth.llamarpc.com")}
}

// GetQuote prices the route from the LayerZero endpoint's own messaging fee.
//
// The fee is read with a live eth_call to the endpoint's quote function, so the
// figure reflects the protocol's current price rather than a fixed multiplier.
// When the endpoint is not configured the quote fails instead of falling back
// to an invented number.
func (c *LayerZeroClient) GetQuote(ctx context.Context, fromChain, toChain int64, token, amount string) (*BridgeQuote, error) {
	endpoint := getEnv("LAYERZERO_ENDPOINT", "")
	if endpoint == "" {
		return nil, fmt.Errorf("LAYERZERO_ENDPOINT is not configured")
	}

	amountFloat, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil || amountFloat <= 0 {
		return nil, fmt.Errorf("invalid amount %q", amount)
	}

	// quoteLayerZeroFee(uint16 dstChainId, uint8 functionType, bytes payload,
	// bytes adapterParams, address zroPaymentAddress, bytes adapterParams)
	calldata := "0x" + encodeSelectorHex("quoteLayerZeroFee(uint16,uint8,bytes,bytes,address,bytes)")
	calldata += encodeUintArg(toChain)
	calldata += encodeUintArg(0)
	calldata += encodeDynamicBytesArg(2, nil)
	calldata += encodeDynamicBytesArg(3, nil)
	calldata += encodeAddressArg("0x0000000000000000000000000000000000000000")
	calldata += encodeDynamicBytesArg(5, nil)

	raw, err := ethCallRaw(ctx, c.rpcURL, endpoint, calldata)
	if err != nil {
		return nil, fmt.Errorf("layerzero fee quote: %w", err)
	}

	nativeFee, err := decodeABIUintAt(raw, 0)
	if err != nil {
		return nil, fmt.Errorf("decode layerzero fee: %w", err)
	}
	feeFloat := weiToFloat(nativeFee)
	if feeFloat <= 0 {
		return nil, fmt.Errorf("layerzero returned a zero messaging fee")
	}

	return &BridgeQuote{
		Bridge:        "LAYERZERO",
		FromChain:     fromChain,
		ToChain:       toChain,
		FromToken:     token,
		ToToken:       token,
		FromAmount:    amount,
		ToAmount:      formatAmount(amountFloat),
		Fee:           strconv.FormatFloat(feeFloat, 'f', 8, 64),
		FeeToken:      nativeSymbolForChain(fromChain),
		EstimatedTime: 600,
	}, nil
}

func (c *LayerZeroClient) ExecuteBridge(ctx context.Context, quote *BridgeQuote, userAddress string) (string, error) {
	return "", ErrExecutionRequiresSignature
}

func (c *LayerZeroClient) GetTransactionStatus(ctx context.Context, txHash string) (string, error) {
	return bridgeStatusForHash(ctx, c.rpcURL, txHash)
}

type MultichainClient struct {
	rpcURL string
}

func NewMultichainClient() *MultichainClient {
	return &MultichainClient{rpcURL: getEnv("MULTICHAIN_RPC_URL", "https://eth.llamarpc.com")}
}

// GetQuote prices the route from the Multichain router's on-chain fee query.
func (c *MultichainClient) GetQuote(ctx context.Context, fromChain, toChain int64, token, amount string) (*BridgeQuote, error) {
	router := getEnv("MULTICHAIN_ROUTER", "")
	if router == "" {
		return nil, fmt.Errorf("MULTICHAIN_ROUTER is not configured")
	}

	amountFloat, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil || amountFloat <= 0 {
		return nil, fmt.Errorf("invalid amount %q", amount)
	}

	raw, err := ethCallRaw(ctx, c.rpcURL, router, "0x"+encodeSelectorHex("getRouterFee()"))
	if err != nil {
		return nil, fmt.Errorf("multichain fee quote: %w", err)
	}
	feeWei, err := decodeABIUintAt(raw, 0)
	if err != nil {
		return nil, fmt.Errorf("decode multichain fee: %w", err)
	}

	return &BridgeQuote{
		Bridge:        "MULTICHAIN",
		FromChain:     fromChain,
		ToChain:       toChain,
		FromToken:     token,
		ToToken:       token,
		FromAmount:    amount,
		ToAmount:      formatAmount(amountFloat),
		Fee:           strconv.FormatFloat(weiToFloat(feeWei), 'f', 8, 64),
		FeeToken:      nativeSymbolForChain(fromChain),
		EstimatedTime: 1200,
	}, nil
}

func (c *MultichainClient) ExecuteBridge(ctx context.Context, quote *BridgeQuote, userAddress string) (string, error) {
	return "", ErrExecutionRequiresSignature
}

func (c *MultichainClient) GetTransactionStatus(ctx context.Context, txHash string) (string, error) {
	return bridgeStatusForHash(ctx, c.rpcURL, txHash)
}

type AxelarClient struct {
	rpcURL string
}

func NewAxelarClient() *AxelarClient {
	return &AxelarClient{rpcURL: getEnv("AXELAR_RPC_URL", "https://eth.llamarpc.com")}
}

// GetQuote prices the route from the Axelar gas service's on-chain fee record.
func (c *AxelarClient) GetQuote(ctx context.Context, fromChain, toChain int64, token, amount string) (*BridgeQuote, error) {
	gasService := getEnv("AXELAR_GAS_SERVICE", "")
	if gasService == "" {
		return nil, fmt.Errorf("AXELAR_GAS_SERVICE is not configured")
	}

	amountFloat, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil || amountFloat <= 0 {
		return nil, fmt.Errorf("invalid amount %q", amount)
	}

	calldata := "0x" + encodeSelectorHex("getGasFee(address,uint256)") + encodeUintArg(fromChain)
	raw, err := ethCallRaw(ctx, c.rpcURL, gasService, calldata)
	if err != nil {
		return nil, fmt.Errorf("axelar fee quote: %w", err)
	}
	feeWei, err := decodeABIUintAt(raw, 0)
	if err != nil {
		return nil, fmt.Errorf("decode axelar fee: %w", err)
	}

	return &BridgeQuote{
		Bridge:        "AXELAR",
		FromChain:     fromChain,
		ToChain:       toChain,
		FromToken:     token,
		ToToken:       token,
		FromAmount:    amount,
		ToAmount:      formatAmount(amountFloat),
		Fee:           strconv.FormatFloat(weiToFloat(feeWei), 'f', 8, 64),
		FeeToken:      nativeSymbolForChain(fromChain),
		EstimatedTime: 900,
	}, nil
}

func (c *AxelarClient) ExecuteBridge(ctx context.Context, quote *BridgeQuote, userAddress string) (string, error) {
	return "", ErrExecutionRequiresSignature
}

func (c *AxelarClient) GetTransactionStatus(ctx context.Context, txHash string) (string, error) {
	return bridgeStatusForHash(ctx, c.rpcURL, txHash)
}

// bridgeStatusForHash reports the real on-chain status of a deposit transaction.
//
// Without a receipt the status is unknown, so PENDING is returned. The previous
// implementation reported CONFIRMED unconditionally, which told a user their
// funds had arrived when no transfer had happened.
func bridgeStatusForHash(ctx context.Context, rpcURL, txHash string) (string, error) {
	if !isHexHash(txHash) {
		return "", fmt.Errorf("invalid transaction hash %q", txHash)
	}
	receipt, err := ethGetTransactionReceipt(ctx, rpcURL, txHash)
	if err != nil {
		return "", err
	}
	if receipt == nil {
		return "PENDING", nil
	}
	if receipt.Status == "0x1" {
		return "CONFIRMED", nil
	}
	return "FAILED", nil
}

// generateTransactionID derives the local record id for a bridge attempt. It
// identifies a row in this service's database; it is never presented as an
// on-chain transaction hash.
func generateTransactionID(user string, from, to int64, amount string) string {
	data := fmt.Sprintf("%s:%d:%d:%s:%d", user, from, to, amount, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "bridge_" + hex.EncodeToString(hash[:])[:24]
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort: "8096",
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: mustEnv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME", "tigerwallet_bridge"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewBridgeService(config)
	if err != nil {
		fmt.Printf("Failed to start bridge service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

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

	api := router.Group("/api/v1/bridge")
	{
		api.GET("/chains", service.GetChains)
		api.GET("/routes", service.GetRoutes)
		api.GET("/quote", service.GetQuotes)
		api.POST("/execute", service.ExecuteBridge)
		api.GET("/status/:id", service.GetTransactionStatus)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "bridge"})
	})

	go func() {
		fmt.Printf("Bridge aggregator service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down bridge service...")
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
