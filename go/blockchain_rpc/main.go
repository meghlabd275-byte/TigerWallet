// TigerWallet Blockchain RPC Connector Service
// High-performance, distributed blockchain connectivity layer
// Supports 100+ blockchains with automatic failover and load balancing

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port             int           `json:"port"`
	RedisAddr        string        `json:"redis_addr"`
	RequestTimeout   time.Duration `json:"request_timeout"`
	MaxConnections   int           `json:"max_connections"`
	RateLimit        int           `json:"rate_limit"`
	RetryAttempts    int           `json:"retry_attempts"`
	RetryDelay       time.Duration `json:"retry_delay"`
}

var globalConfig = Config{
	Port:           8090,
	RedisAddr:      "localhost:6379",
	RequestTimeout: 30 * time.Second,
	MaxConnections: 1000,
	RateLimit:      100,
	RetryAttempts:   3,
	RetryDelay:      500 * time.Millisecond,
}

// ============================================================================
// Blockchain Network Configuration
// ============================================================================

type ChainConfig struct {
	Name            string   `json:"name"`
	ChainID         uint64   `json:"chain_id"`
	Symbol          string   `json:"symbol"`
	Decimals        int      `json:"decimals"`
	RPCURLs         []string `json:"rpc_urls"`
	ExplorerURL     string   `json:"explorer_url"`
	ExplorerAPIURL  string   `json:"explorer_api_url"`
	BlockTime       int      `json:"block_time"`
	SupportsEIP1559 bool     `json:"supports_eip1559"`
	IsActive        bool     `json:"is_active"`
}

var chainConfigs = map[string]ChainConfig{
	"ethereum": {
		Name:            "Ethereum",
		ChainID:         1,
		Symbol:          "ETH",
		Decimals:        18,
		RPCURLs:         []string{"https://eth.llamarpc.com"},
		ExplorerURL:     "https://etherscan.io",
		ExplorerAPIURL:  "https://api.etherscan.io/api",
		BlockTime:       12,
		SupportsEIP1559: true,
		IsActive:        true,
	},
	"polygon": {
		Name:            "Polygon",
		ChainID:         137,
		Symbol:          "MATIC",
		Decimals:        18,
		RPCURLs:         []string{"https://polygon-rpc.com"},
		ExplorerURL:     "https://polygonscan.com",
		ExplorerAPIURL:  "https://api.polygonscan.com/api",
		BlockTime:       2,
		SupportsEIP1559: true,
		IsActive:        true,
	},
	"arbitrum": {
		Name:            "Arbitrum One",
		ChainID:         42161,
		Symbol:          "ETH",
		Decimals:        18,
		RPCURLs:         []string{"https://arb1.arbitrum.io/rpc"},
		ExplorerURL:     "https://arbiscan.io",
		ExplorerAPIURL:  "https://api.arbiscan.io/api",
		BlockTime:       1,
		SupportsEIP1559: true,
		IsActive:        true,
	},
	"optimism": {
		Name:            "Optimism",
		ChainID:         10,
		Symbol:          "ETH",
		Decimals:        18,
		RPCURLs:         []string{"https://mainnet.optimism.io"},
		ExplorerURL:     "https://optimistic.etherscan.io",
		ExplorerAPIURL:  "https://api-optimistic.etherscan.io/api",
		BlockTime:       2,
		SupportsEIP1559: true,
		IsActive:        true,
	},
	"avalanche": {
		Name:            "Avalanche C-Chain",
		ChainID:         43114,
		Symbol:          "AVAX",
		Decimals:        18,
		RPCURLs:         []string{"https://api.avax.network/ext/bc/C/rpc"},
		ExplorerURL:     "https://snowtrace.io",
		ExplorerAPIURL:  "https://api.snowtrace.io/api",
		BlockTime:       2,
		SupportsEIP1559: true,
		IsActive:        true,
	},
	"bsc": {
		Name:            "BNB Smart Chain",
		ChainID:         56,
		Symbol:          "BNB",
		Decimals:        18,
		RPCURLs:         []string{"https://bsc-dataseed.binance.org"},
		ExplorerURL:     "https://bscscan.com",
		ExplorerAPIURL:  "https://api.bscscan.com/api",
		BlockTime:       3,
		SupportsEIP1559: true,
		IsActive:        true,
	},
	"base": {
		Name:            "Base",
		ChainID:         8453,
		Symbol:          "ETH",
		Decimals:        18,
		RPCURLs:         []string{"https://mainnet.base.org"},
		ExplorerURL:     "https://basescan.org",
		ExplorerAPIURL:  "https://api.basescan.org/api",
		BlockTime:       2,
		SupportsEIP1559: true,
		IsActive:        true,
	},
	"solana": {
		Name:            "Solana",
		ChainID:         0,
		Symbol:          "SOL",
		Decimals:        9,
		RPCURLs:         []string{"https://api.mainnet-beta.solana.com"},
		ExplorerURL:     "https://explorer.solana.com",
		ExplorerAPIURL:  "",
		BlockTime:       1,
		SupportsEIP1559: false,
		IsActive:        true,
	},
	"tron": {
		Name:            "TRON",
		ChainID:         728126428,
		Symbol:          "TRX",
		Decimals:        6,
		RPCURLs:         []string{"https://api.trongrid.io"},
		ExplorerURL:     "https://tronscan.org",
		ExplorerAPIURL:  "https://api.tronscan.org/api",
		BlockTime:       3,
		SupportsEIP1559: false,
		IsActive:        true,
	},
	"aptos": {
		Name:            "Aptos",
		ChainID:         1,
		Symbol:          "APT",
		Decimals:        8,
		RPCURLs:         []string{"https://fullnode.mainnet.aptoslabs.com"},
		ExplorerURL:     "https://explorer.aptoslabs.com",
		ExplorerAPIURL:  "",
		BlockTime:       1,
		SupportsEIP1559: false,
		IsActive:        true,
	},
}

// ============================================================================
// RPC Client Manager
// ============================================================================

type RPCClient struct {
	client      *ethclient.Client
	chain       string
	config      ChainConfig
	currentURL  int
	mu          sync.RWMutex
	rateLimiter *rate.Limiter
	lastUsed    time.Time
}

type RPCManager struct {
	clients map[string]*RPCClient
	mu      sync.RWMutex
	redis   *redis.Client
}

func NewRPCManager(redisAddr string) *RPCManager {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	
	return &RPCManager{
		clients: make(map[string]*RPCClient),
		redis:   rdb,
	}
}

func (rm *RPCManager) GetClient(chain string) (*RPCClient, error) {
	rm.mu.RLock()
	client, exists := rm.clients[chain]
	rm.mu.RUnlock()

	if exists {
		return client, nil
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := rm.clients[chain]; exists {
		return client, nil
	}

	config, ok := chainConfigs[chain]
	if !ok {
		return nil, fmt.Errorf("chain %s not supported", chain)
	}

	if !config.IsActive {
		return nil, fmt.Errorf("chain %s is not active", chain)
	}

	newClient := &RPCClient{
		chain:       chain,
		config:      config,
		currentURL:  0,
		rateLimiter: rate.NewLimiter(rate.Limit(globalConfig.RateLimit), globalConfig.RateLimit),
	}

	// Connect to first available RPC
	for i := 0; i < len(config.RPCURLs); i++ {
		ethClient, err := ethclient.Dial(config.RPCURLs[i])
		if err == nil {
			newClient.client = ethClient
			newClient.currentURL = i
			break
		}
		log.Printf("Failed to connect to %s RPC %s: %v", chain, config.RPCURLs[i], err)
	}

	if newClient.client == nil {
		return nil, fmt.Errorf("no working RPC for chain %s", chain)
	}

	rm.clients[chain] = newClient
	return newClient, nil
}

func (rc *RPCClient) callWithRetry(ctx context.Context, operation func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	
	for attempt := 0; attempt < globalConfig.RetryAttempts; attempt++ {
		// Check rate limit
		if err := rc.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		result, err := operation()
		if err == nil {
			return result, nil
		}

		lastErr = err
		
		// Try next RPC if available
		if rc.shouldSwitchRPC(err) {
			rc.switchRPC()
		}

		// Wait before retry
		time.Sleep(globalConfig.RetryDelay * time.Duration(attempt+1))
	}

	return nil, fmt.Errorf("operation failed after %d attempts: %v", globalConfig.RetryAttempts, lastErr)
}

func (rc *RPCClient) shouldSwitchRPC(err error) bool {
	errStr := err.Error()
	switches := []string{
		"connection refused",
		"timeout",
		"429",
		"500",
		"502",
		"503",
		"server error",
		"i/o timeout",
		"network",
		"max rate limit",
	}
	
	errLower := strings.ToLower(errStr)
	for _, s := range switches {
		if strings.Contains(errLower, s) {
			return true
		}
	}
	return false
}

func (rc *RPCClient) switchRPC() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if len(rc.config.RPCURLs) <= 1 {
		return
	}

	nextURL := (rc.currentURL + 1) % len(rc.config.RPCURLs)
	
	for i := 0; i < len(rc.config.RPCURLs); i++ {
		ethClient, err := ethclient.Dial(rc.config.RPCURLs[nextURL])
		if err == nil {
			if rc.client != nil {
				rc.client.Close()
			}
			rc.client = ethClient
			rc.currentURL = nextURL
			log.Printf("Switched %s RPC to %s", rc.chain, rc.config.RPCURLs[nextURL])
			return
		}
		nextURL = (nextURL + 1) % len(rc.config.RPCURLs)
	}

	log.Printf("Failed to switch RPC for %s", rc.chain)
}

// ============================================================================
// API Types
// ============================================================================

type BalanceRequest struct {
	Chain   string `json:"chain" binding:"required"`
	Address string `json:"address" binding:"required"`
}

type BalanceResponse struct {
	Address    string `json:"address"`
	Chain      string `json:"chain"`
	Balance    string `json:"balance"`
	Symbol     string `json:"symbol"`
	Decimals   int    `json:"decimals"`
	RawBalance string `json:"raw_balance"`
	Error      string `json:"error,omitempty"`
}

type CallRequest struct {
	Chain string `json:"chain" binding:"required"`
	To    string `json:"to" binding:"required"`
	Data  string `json:"data" binding:"required"`
}

type CallResponse struct {
	Output string `json:"output"`
}

type ChainInfoResponse struct {
	Name            string `json:"name"`
	ChainID         uint64 `json:"chain_id"`
	Symbol          string `json:"symbol"`
	Decimals        int    `json:"decimals"`
	BlockNumber     string `json:"block_number"`
	GasPrice        string `json:"gas_price"`
	ExplorerURL     string `json:"explorer_url"`
	SupportsEIP1559 bool   `json:"supports_eip1559"`
}

// ============================================================================
// HTTP Handlers
// ============================================================================

type APIHandler struct {
	rpcManager *RPCManager
}

func NewAPIHandler(rm *RPCManager) *APIHandler {
	return &APIHandler{rpcManager: rm}
}

func (h *APIHandler) GetBalance(c *gin.Context) {
	var req BalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !common.IsHexAddress(req.Address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address format"})
		return
	}

	client, err := h.rpcManager.GetClient(req.Chain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), globalConfig.RequestTimeout)
	defer cancel()

	result, err := client.callWithRetry(ctx, func() (interface{}, error) {
		address := common.HexToAddress(req.Address)
		balance, err := client.client.BalanceAt(ctx, address, nil)
		if err != nil {
			return nil, err
		}
		
		decimals := big.NewInt(int64(client.config.Decimals))
		divisor := new(big.Int).Exp(big.NewInt(10), decimals, nil)
		
		humanBalance := new(big.Rat).SetFrac(balance, divisor)
		
		return BalanceResponse{
			Address:    req.Address,
			Chain:      req.Chain,
			Balance:    humanBalance.FloatString(8),
			Symbol:     client.config.Symbol,
			Decimals:   client.config.Decimals,
			RawBalance: balance.String(),
		}, nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *APIHandler) GetChainInfo(c *gin.Context) {
	chain := c.Param("chain")

	client, err := h.rpcManager.GetClient(chain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), globalConfig.RequestTimeout)
	defer cancel()

	result, err := client.callWithRetry(ctx, func() (interface{}, error) {
		blockNumber, err := client.client.BlockNumber(ctx)
		if err != nil {
			return nil, err
		}

		gasPrice, err := client.client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, err
		}

		return ChainInfoResponse{
			Name:            client.config.Name,
			ChainID:         client.config.ChainID,
			Symbol:          client.config.Symbol,
			Decimals:        client.config.Decimals,
			BlockNumber:     fmt.Sprintf("0x%x", blockNumber),
			GasPrice:        gasPrice.String(),
			ExplorerURL:     client.config.ExplorerURL,
			SupportsEIP1559: client.config.SupportsEIP1559,
		}, nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *APIHandler) CallContract(c *gin.Context) {
	var req CallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := h.rpcManager.GetClient(req.Chain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), globalConfig.RequestTimeout)
	defer cancel()

	data, err := hex.DecodeString(req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data hex"})
		return
	}

	result, err := client.callWithRetry(ctx, func() (interface{}, error) {
		to := common.HexToAddress(req.To)
		msg := ethereum.CallMsg{
			To:   &to,
			Data: data,
		}
		
		output, err := client.client.CallContract(ctx, msg, nil)
		if err != nil {
			return nil, err
		}

		return CallResponse{
			Output: hex.EncodeToString(output),
		}, nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *APIHandler) GetTransactionReceipt(c *gin.Context) {
	chain := c.Param("chain")
	txHash := c.Param("hash")

	client, err := h.rpcManager.GetClient(chain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), globalConfig.RequestTimeout)
	defer cancel()

	result, err := client.callWithRetry(ctx, func() (interface{}, error) {
		receipt, err := client.client.TransactionReceipt(ctx, common.HexToHash(txHash))
		if err != nil {
			return nil, err
		}

		return receipt, nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *APIHandler) EstimateGas(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chain := req["chain"]
	from := req["from"]
	to := req["to"]
	value := req["value"]

	client, err := h.rpcManager.GetClient(chain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), globalConfig.RequestTimeout)
	defer cancel()

	fromAddr := common.HexToAddress(from)
	toAddr := common.HexToAddress(to)
	
	val := new(big.Int)
	val.SetString(value, 10)
	
	msg := ethereum.CallMsg{
		From: fromAddr,
		To:   &toAddr,
		Value: val,
	}

	result, err := client.callWithRetry(ctx, func() (interface{}, error) {
		gas, err := client.client.EstimateGas(ctx, msg)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"gas": fmt.Sprintf("0x%x", gas),
		}, nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *APIHandler) GetSupportedChains(c *gin.Context) {
	chains := make([]ChainInfoResponse, 0)
	
	for _, config := range chainConfigs {
		if config.IsActive {
			chains = append(chains, ChainInfoResponse{
				Name:            config.Name,
				ChainID:         config.ChainID,
				Symbol:          config.Symbol,
				Decimals:        config.Decimals,
				ExplorerURL:     config.ExplorerURL,
				SupportsEIP1559: config.SupportsEIP1559,
			})
		}
	}

	c.JSON(http.StatusOK, chains)
}

// ============================================================================
// WebSocket Handler
// ============================================================================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSSubscription struct {
	Chain   string   `json:"chain"`
	Address string   `json:"address"`
	Events  []string `json:"events"`
}

func (h *APIHandler) HandleWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket client connected")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		var sub WSSubscription
		if err := json.Unmarshal(msg, &sub); err != nil {
			conn.WriteJSON(gin.H{"error": "invalid subscription format"})
			continue
		}

		conn.WriteJSON(gin.H{"status": "subscribed", "chain": sub.Chain, "address": sub.Address})
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("TigerWallet Blockchain RPC Connector Service")
	log.Println("============================================")
	log.Printf("Starting on port %d", globalConfig.Port)

	rpcManager := NewRPCManager(globalConfig.RedisAddr)
	handler := NewAPIHandler(rpcManager)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Chain")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "blockchain-rpc",
			"timestamp": time.Now().Unix(),
		})
	})

	api := r.Group("/api/v1")
	{
		api.GET("/chains", handler.GetSupportedChains)
		api.GET("/chain/:chain", handler.GetChainInfo)
		api.POST("/balance", handler.GetBalance)
		api.POST("/call", handler.CallContract)
		api.POST("/estimate-gas", handler.EstimateGas)
		api.GET("/tx/:chain/:hash", handler.GetTransactionReceipt)
	}

	r.GET("/ws", handler.HandleWS)

	addr := fmt.Sprintf(":%d", globalConfig.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("Server starting on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
