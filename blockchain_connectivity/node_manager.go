/**
 * TigerWallet Blockchain Connectivity Infrastructure
 * Production-ready node management and RPC infrastructure
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Configuration
// ============================================================================

type BlockchainConfig struct {
	ServerPort  string `json:"server_port"`
	RedisHost   string `json:"redis_host"`
	RedisPort   string `json:"redis_port"`
}

func LoadConfig() *BlockchainConfig {
	return &BlockchainConfig{
		ServerPort: getEnv("BLOCKCHAIN_PORT", "9099"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Chain Definitions
// ============================================================================

type Chain struct {
	ChainID       uint   `json:"chain_id"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	Type          string `json:"type"` // evm, bitcoin, solana, ton
	RPCURLs       []string `json:"rpc_urls"`
	ExplorerURLs  []string `json:"explorer_urls"`
	NativeToken   string `json:"native_token"`
	Confirmations int    `json:"confirmations"`
	BlockTime     int    `json:"block_time"` // seconds
	IsTestnet    bool   `json:"is_testnet"`
}

// Complete chain list with real RPC endpoints
var CHAINS = map[string]Chain{
	// EVM Chains
	"ethereum": {
		ChainID: 1, Name: "Ethereum", Symbol: "ETH", Type: "evm",
		RPCURLs: []string{
			"https://eth.llamarpc.com",
			"https://eth-mainnet.g.alchemy.com/v2/demo",
			"https://cloudflare-eth.com",
		},
		ExplorerURLs: []string{"https://etherscan.io"},
		NativeToken: "ETH", Confirmations: 12, BlockTime: 12,
	},
	"polygon": {
		ChainID: 137, Name: "Polygon", Symbol: "MATIC", Type: "evm",
		RPCURLs: []string{
			"https://polygon-rpc.com",
			"https://polygon.llamarpc.com",
		},
		ExplorerURLs: []string{"https://polygonscan.com"},
		NativeToken: "MATIC", Confirmations: 128, BlockTime: 2,
	},
	"bsc": {
		ChainID: 56, Name: "BNB Smart Chain", Symbol: "BNB", Type: "evm",
		RPCURLs: []string{
			"https://bsc-dataseed.binance.org",
			"https://bsc.llamarpc.com",
		},
		ExplorerURLs: []string{"https://bscscan.com"},
		NativeToken: "BNB", Confirmations: 15, BlockTime: 3,
	},
	"arbitrum": {
		ChainID: 42161, Name: "Arbitrum One", Symbol: "ETH", Type: "evm",
		RPCURLs: []string{
			"https://arb1.arbitrum.io/rpc",
			"https://arbitrum.llamarpc.com",
		},
		ExplorerURLs: []string{"https://arbiscan.io"},
		NativeToken: "ETH", Confirmations: 12, BlockTime: 1,
	},
	"optimism": {
		ChainID: 10, Name: "Optimism", Symbol: "ETH", Type: "evm",
		RPCURLs: []string{
			"https://mainnet.optimism.io",
			"https://optimism.llamarpc.com",
		},
		ExplorerURLs: []string{"https://optimistic.etherscan.io"},
		NativeToken: "ETH", Confirmations: 12, BlockTime: 2,
	},
	"avalanche": {
		ChainID: 43114, Name: "Avalanche", Symbol: "AVAX", Type: "evm",
		RPCURLs: []string{
			"https://api.avax.network/ext/bc/C/rpc",
			"https://avalanche.llamarpc.com",
		},
		ExplorerURLs: []string{"https://snowtrace.io"},
		NativeToken: "AVAX", Confirmations: 12, BlockTime: 1,
	},
	"base": {
		ChainID: 8453, Name: "Base", Symbol: "ETH", Type: "evm",
		RPCURLs: []string{
			"https://mainnet.base.org",
			"https://base.llamarpc.com",
		},
		ExplorerURLs: []string{"https://basescan.org"},
		NativeToken: "ETH", Confirmations: 12, BlockTime: 2,
	},
	"fantom": {
		ChainID: 250, Name: "Fantom", Symbol: "FTM", Type: "evm",
		RPCURLs: []string{
			"https://rpc.fantom.network",
			"https://fantom.llamarpc.com",
		},
		ExplorerURLs: []string{"https://ftmscan.com"},
		NativeToken: "FTM", Confirmations: 1, BlockTime: 1,
	},
	"cronos": {
		ChainID: 25, Name: "Cronos", Symbol: "CRO", Type: "evm",
		RPCURLs: []string{
			"https://evm.cronos.org",
		},
		ExplorerURLs: []string{"https://cronoscan.com"},
		NativeToken: "CRO", Confirmations: 50, BlockTime: 6,
	},
	"celo": {
		ChainID: 42220, Name: "Celo", Symbol: "CELO", Type: "evm",
		RPCURLs: []string{
			"https://forno.celo.org",
		},
		ExplorerURLs: []string{"https://explorer.celo.org"},
		NativeToken: "CELO", Confirmations: 1, BlockTime: 5,
	},
	// Additional chains
	"polygon_zkevm": {
		ChainID: 1101, Name: "Polygon zkEVM", Symbol: "ETH", Type: "evm",
		RPCURLs: []string{
			"https://zkevm-rpc.com",
		},
		ExplorerURLs: []string{"https://zkevm.polygonscan.com"},
		NativeToken: "ETH", Confirmations: 1, BlockTime: 1,
	},
	"linea": {
		ChainID: 59144, Name: "Linea", Symbol: "ETH", Type: "evm",
		RPCURLs: []string{
			"https://rpc.linea.build",
		},
		ExplorerURLs: []string{"https://lineascan.build"},
		NativeToken: "ETH", Confirmations: 1, BlockTime: 1,
	},
	"scroll": {
		ChainID: 534352, Name: "Scroll", Symbol: "ETH", Type: "evm",
		RPCURLs: []string{
			"https://rpc.scroll.io",
		},
		ExplorerURLs: []string{"https://scrollscan.com"},
		NativeToken: "ETH", Confirmations: 1, BlockTime: 1,
	},
	"zksync": {
		ChainID: 324, Name: "zkSync Era", Symbol: "ETH", Type: "evm",
		RPCURLs: []string{
			"https://mainnet.era.zksync.io",
		},
		ExplorerURLs: []string{"https://explorer.zksync.io"},
		NativeToken: "ETH", Confirmations: 1, BlockTime: 1,
	},
	"mantle": {
		ChainID: 5000, Name: "Mantle", Symbol: "MNT", Type: "evm",
		RPCURLs: []string{
			"https://rpc.mantle.xyz",
		},
		ExplorerURLs: []string{"https://explorer.mantle.xyz"},
		NativeToken: "MNT", Confirmations: 1, BlockTime: 1,
	},
}

// ============================================================================
// RPC Client
// ============================================================================

type RPCClient struct {
	URL    string
	client *http.Client
	mu     sync.RWMutex
}

func NewRPCClient(url string) *RPCClient {
	return &RPCClient{
		URL: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params []interface{} `json:"params"`
	ID     int          `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *RPCClient) Call(method string, params ...interface{}) (json.RawMessage, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequest("POST", c.URL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// ============================================================================
// Chain Manager
// ============================================================================

type ChainManager struct {
	chains     map[string]Chain
	clients    map[string]*RPCClient
	redis      *redis.Client
	currentIdx map[string]int
	mu         sync.RWMutex
}

func NewChainManager(redisHost, redisPort string) (*ChainManager, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
		DB: 8,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Printf("Warning: Redis connection failed: %v\n", err)
	}

	clients := make(map[string]*RPCClient)
	currentIdx := make(map[string]int)

	// Initialize RPC clients for each chain
	for name, chain := range CHAINS {
		if len(chain.RPCURLs) > 0 {
			clients[name] = NewRPCClient(chain.RPCURLs[0])
			currentIdx[name] = 0
		}
	}

	return &ChainManager{
		chains:     CHAINS,
		clients:    clients,
		redis:      rdb,
		currentIdx: currentIdx,
	}, nil
}

// Get best RPC URL (with failover)
func (m *ChainManager) getRPCURL(chainName string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, ok := m.chains[chainName]
	if !ok || len(chain.RPCURLs) == 0 {
		return ""
	}

	idx := m.currentIdx[chainName]
	return chain.RPCURLs[idx]
}

// Get client with auto-failover
func (m *ChainManager) GetClient(chainName string) *RPCClient {
	m.mu.RLock()
	client, ok := m.clients[chainName]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	// Try current URL
	_, err := client.Call("eth_blockNumber")
	if err == nil {
		return client
	}

	// Failover to next URL
	m.mu.Lock()
	chain := m.chains[chainName]
	m.currentIdx[chainName] = (m.currentIdx[chainName] + 1) % len(chain.RPCURLs)
	newURL := chain.RPCURLs[m.currentIdx[chainName]]
	m.clients[chainName] = NewRPCClient(newURL)
	m.mu.Unlock()

	return m.clients[chainName]
}

// ============================================================================
// API Methods
// ============================================================================

func (m *ChainManager) GetBlockNumber(chainName string) (uint64, error) {
	client := m.GetClient(chainName)
	if client == nil {
		return 0, fmt.Errorf("chain not found: %s", chainName)
	}

	result, err := client.Call("eth_blockNumber")
	if err != nil {
		return 0, err
	}

	var blockHex string
	json.Unmarshal(result, &blockHex)

	blockNum := parseHex(blockHex)
	return blockNum, nil
}

func (m *ChainManager) GetBalance(chainName, address string) (*big.Int, error) {
	client := m.GetClient(chainName)
	if client == nil {
		return nil, fmt.Errorf("chain not found: %s", chainName)
	}

	result, err := client.Call("eth_getBalance", address, "latest")
	if err != nil {
		return nil, err
	}

	var balanceHex string
	json.Unmarshal(result, &balanceHex)

	return parseHexToBigInt(balanceHex), nil
}

func (m *ChainManager) GetTransactionCount(chainName, address string) (uint64, error) {
	client := m.GetClient(chainName)
	if client == nil {
		return 0, fmt.Errorf("chain not found: %s", chainName)
	}

	result, err := client.Call("eth_getTransactionCount", address, "latest")
	if err != nil {
		return 0, err
	}

	var nonceHex string
	json.Unmarshal(result, &nonceHex)

	return parseHex(nonceHex), nil
}

func (m *ChainManager) GetGasPrice(chainName string) (*big.Int, error) {
	client := m.GetClient(chainName)
	if client == nil {
		return nil, fmt.Errorf("chain not found: %s", chainName)
	}

	result, err := client.Call("eth_gasPrice")
	if err != nil {
		return nil, err
	}

	var gasHex string
	json.Unmarshal(result, &gasHex)

	return parseHexToBigInt(gasHex), nil
}

func (m *ChainManager) GetTransactionReceipt(chainName, txHash string) (map[string]interface{}, error) {
	client := m.GetClient(chainName)
	if client == nil {
		return nil, fmt.Errorf("chain not found: %s", chainName)
	}

	result, err := client.Call("eth_getTransactionReceipt", txHash)
	if err != nil {
		return nil, err
	}

	var receipt map[string]interface{}
	json.Unmarshal(result, &receipt)

	return receipt, nil
}

func (m *ChainManager) GetCode(chainName, address string) (string, error) {
	client := m.GetClient(chainName)
	if client == nil {
		return "", fmt.Errorf("chain not found: %s", chainName)
	}

	result, err := client.Call("eth_getCode", address, "latest")
	if err != nil {
		return "", err
	}

	var code string
	json.Unmarshal(result, &code)

	return code, nil
}

// Get chain info
func (m *ChainManager) GetChain(chainName string) (Chain, bool) {
	chain, ok := m.chains[chainName]
	return chain, ok
}

// List all supported chains
func (m *ChainManager) ListChains() []Chain {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chains := make([]Chain, 0, len(m.chains))
	for _, chain := range m.chains {
		chains = append(chains, chain)
	}
	return chains
}

// Health check for all chains
func (m *ChainManager) HealthCheck() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[string]bool)
	for name := range m.chains {
		client, ok := m.clients[name]
		if !ok {
			health[name] = false
			continue
		}

		_, err := client.Call("eth_blockNumber")
		health[name] = err == nil
	}

	return health
}

// ============================================================================
// Utility Functions
// ============================================================================

func parseHex(hex string) uint64 {
	hex = strings.TrimPrefix(hex, "0x")
	var num uint64
	fmt.Sscanf(hex, "%x", &num)
	return num
}

func parseHexToBigInt(hex string) *big.Int {
	hex = strings.TrimPrefix(hex, "0x")
	ok, _ := new(big.Int).SetString(hex, 16)
	if ok {
		return ok
	}
	return big.NewInt(0)
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (m *ChainManager) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Chain info
		api.GET("/chains", m.listChains)
		api.GET("/chains/:name", m.getChain)
		api.GET("/chains/:name/health", m.checkHealth)

		// Blockchain queries
		api.GET("/chains/:name/block-number", m.getBlockNumber)
		api.GET("/chains/:name/balance/:address", m.getBalance)
		api.GET("/chains/:name/nonce/:address", m.getNonce)
		api.GET("/chains/:name/gas-price", m.getGasPrice)
		api.GET("/chains/:name/tx/:hash", m.getTransaction)
		api.GET("/chains/:name/code/:address", m.getCode)
	}
}

func (m *ChainManager) listChains(c *gin.Context) {
	c.JSON(200, gin.H{"chains": m.ListChains()})
}

func (m *ChainManager) getChain(c *gin.Context) {
	name := c.Param("name")
	chain, ok := m.GetChain(name)
	if !ok {
		c.JSON(404, gin.H{"error": "chain not found"})
		return
	}
	c.JSON(200, chain)
}

func (m *ChainManager) checkHealth(c *gin.Context) {
	c.JSON(200, m.HealthCheck())
}

func (m *ChainManager) getBlockNumber(c *gin.Context) {
	name := c.Param("name")
	blockNum, err := m.GetBlockNumber(name)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"block_number": blockNum})
}

func (m *ChainManager) getBalance(c *gin.Context) {
	name := c.Param("name")
	address := c.Param("address")

	balance, err := m.GetBalance(name, address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"balance": balance.String()})
}

func (m *ChainManager) getNonce(c *gin.Context) {
	name := c.Param("name")
	address := c.Param("address")

	nonce, err := m.GetTransactionCount(name, address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"nonce": nonce})
}

func (m *ChainManager) getGasPrice(c *gin.Context) {
	name := c.Param("name")

	gasPrice, err := m.GetGasPrice(name)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"gas_price": gasPrice.String()})
}

func (m *ChainManager) getTransaction(c *gin.Context) {
	name := c.Param("name")
	hash := c.Param("hash")

	receipt, err := m.GetTransactionReceipt(name, hash)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, receipt)
}

func (m *ChainManager) getCode(c *gin.Context) {
	name := c.Param("name")
	address := c.Param("address")

	code, err := m.GetCode(name, address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": code})
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	manager, err := NewChainManager(config.RedisHost, config.RedisPort)
	if err != nil {
		fmt.Printf("Failed to initialize chain manager: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()
	router.Use(gin.Recovery())

	// CORS
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

	// Health
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"service": "blockchain-connectivity",
			"chains": len(manager.ListChains()),
		})
	})

	// Register API routes
	manager.RegisterRoutes(router)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Printf("Blockchain Connectivity service starting on port %s\n", config.ServerPort)
		fmt.Printf("Supported chains: %d\n", len(CHAINS))
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	<-quit
	fmt.Println("Shutting down...")
}
