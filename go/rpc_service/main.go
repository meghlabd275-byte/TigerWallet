package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// RPC Request/Response Types
// ============================================================================

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError      `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type BlockNumberResult string

type BalanceResult string

type TransactionByHashResult struct {
	Hash             string `json:"hash"`
	Nonce            string `json:"nonce"`
	BlockHash        string `json:"blockHash"`
	BlockNumber      string `json:"blockNumber"`
	TransactionIndex string `json:"transactionIndex"`
	From             string `json:"from"`
	To               string `json:"to"`
	Value            string `json:"value"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	Input            string `json:"input"`
}

type TransactionReceipt struct {
	TransactionHash   string   `json:"transactionHash"`
	BlockHash         string   `json:"blockHash"`
	BlockNumber       string   `json:"blockNumber"`
	CumulativeGasUsed string   `json:"cumulativeGasUsed"`
	GasUsed           string   `json:"gasUsed"`
	Logs              []Log    `json:"logs"`
	Status            string   `json:"status"`
}

type Log struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

// ============================================================================
// Chain Configuration
// ============================================================================

type ChainConfig struct {
	ChainID      uint64            `json:"chainId"`
	Name         string            `json:"name"`
	Symbol       string            `json:"symbol"`
	Decimals     uint8             `json:"decimals"`
	RPCEndpoints []string          `json:"rpcEndpoints"`
	WSEndpoints  []string          `json:"wsEndpoints"`
	ExplorerURL string            `json:"explorerUrl"`
	Features     map[string]bool   `json:"features"`
}

var ChainConfigs = map[string]ChainConfig{
	"1": {
		ChainID:      1,
		Name:         "Ethereum",
		Symbol:       "ETH",
		Decimals:     18,
		RPCEndpoints: []string{
			"https://eth.llamarpc.com",
			"https://eth-mainnet.g.alchemy.com/v2/demo",
			"https://rpc.ankr.com/eth",
			"https://1rpc.io/eth",
		},
		ExplorerURL: "https://etherscan.io",
		Features:    map[string]bool{"eip1559": true},
	},
	"56": {
		ChainID:      56,
		Name:         "BNB Smart Chain",
		Symbol:       "BNB",
		Decimals:     18,
		RPCEndpoints: []string{
			"https://bsc-dataseed.binance.org",
			"https://bsc-rpc.gateway.pokt.network",
			"https://1rpc.io/bnb",
		},
		ExplorerURL: "https://bscscan.com",
	},
	"137": {
		ChainID:      137,
		Name:         "Polygon",
		Symbol:       "MATIC",
		Decimals:     18,
		RPCEndpoints: []string{
			"https://polygon-rpc.com",
			"https://1rpc.io/polygon",
			"https://polygon.llamarpc.com",
		},
		ExplorerURL: "https://polygonscan.com",
		Features:    map[string]bool{"eip1559": true},
	},
	"42161": {
		ChainID:      42161,
		Name:         "Arbitrum One",
		Symbol:       "ETH",
		Decimals:     18,
		RPCEndpoints: []string{
			"https://arb1.arbitrum.io/rpc",
			"https://1rpc.io/arb",
		},
		ExplorerURL: "https://arbiscan.io",
		Features:    map[string]bool{"eip1559": true},
	},
	"10": {
		ChainID:      10,
		Name:         "Optimism",
		Symbol:       "ETH",
		Decimals:     18,
		RPCEndpoints: []string{
			"https://mainnet.optimism.io",
			"https://1rpc.io/op",
		},
		ExplorerURL: "https://optimistic.etherscan.io",
		Features:    map[string]bool{"eip1559": true},
	},
	"8453": {
		ChainID:      8453,
		Name:         "Base",
		Symbol:       "ETH",
		Decimals:     18,
		RPCEndpoints: []string{
			"https://mainnet.base.org",
			"https://1rpc.io/base",
		},
		ExplorerURL: "https://basescan.org",
		Features:    map[string]bool{"eip1559": true},
	},
	"43114": {
		ChainID:      43114,
		Name:         "Avalanche C-Chain",
		Symbol:       "AVAX",
		Decimals:     18,
		RPCEndpoints: []string{
			"https://api.avax.network/ext/bc/C/rpc",
			"https://1rpc.io/avax",
		},
		ExplorerURL: "https://snowtrace.io",
		Features:    map[string]bool{"eip1559": true},
	},
	"101": {
		ChainID:      101,
		Name:         "Solana",
		Symbol:       "SOL",
		Decimals:     9,
		RPCEndpoints: []string{
			"https://api.mainnet-beta.solana.com",
			"https://1rpc.io/sol",
		},
		ExplorerURL: "https://solscan.io",
	},
	"0": {
		ChainID:      0,
		Name:         "Bitcoin",
		Symbol:       "BTC",
		Decimals:     8,
		RPCEndpoints: []string{
			"https://blockstream.info/api",
		},
		ExplorerURL: "https://mempool.space",
	},
}

// ============================================================================
// RPC Manager
// ============================================================================

type RPCManager struct {
	chains         map[string]ChainConfig
	clients        map[string]*ChainClient
	healthStatus   map[string]*HealthStatus
	mu             sync.RWMutex
	requestCount   uint64
	failureCount   uint64
	lastHealthCheck time.Time
	redisClient    *redis.Client
	httpClient     *http.Client
}

type ChainClient struct {
	endpoints []string
	current   int
	mu        sync.Mutex
	latencies []time.Duration
}

type HealthStatus struct {
	ChainID       string
	IsHealthy     bool
	AvgLatencyMs  float64
	SuccessRate   float64
	LastCheck     time.Time
	LastError     string
}

func NewRPCManager(redisAddr string) *RPCManager {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
				MinVersion:         tls.VersionTLS12,
			},
		},
	}

	var redisClient *redis.Client
	if redisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: "",
			DB:       0,
		})
	}

	manager := &RPCManager{
		chains:       ChainConfigs,
		clients:      make(map[string]*ChainClient),
		healthStatus: make(map[string]*HealthStatus),
		httpClient:   httpClient,
		redisClient:  redisClient,
	}

	// Initialize chain clients
	for chainID, config := range ChainConfigs {
		manager.clients[chainID] = &ChainClient{
			endpoints: config.RPCEndpoints,
			current:   0,
			latencies: make([]time.Duration, 0),
		}
		manager.healthStatus[chainID] = &HealthStatus{
			ChainID:   chainID,
			IsHealthy: true,
		}
	}

	return manager
}

func (m *RPCManager) SendRequest(chainID string, method string, params interface{}) (json.RawMessage, error) {
	atomic.AddUint64(&m.requestCount, 1)

	client, ok := m.clients[chainID]
	if !ok {
		atomic.AddUint64(&m.failureCount, 1)
		return nil, fmt.Errorf("chain not found: %s", chainID)
	}

	// Try each endpoint
	var lastErr error
	for i := 0; i < len(client.endpoints); i++ {
		endpoint := client.getEndpoint()
		result, err := m.doRequest(endpoint, method, params)
		if err == nil {
			return result, nil
		}
		lastErr = err
		client.nextEndpoint()
	}

	atomic.AddUint64(&m.failureCount, 1)
	return nil, fmt.Errorf("all endpoints failed: %v", lastErr)
}

func (m *RPCManager) doRequest(endpoint string, method string, params interface{}) (json.RawMessage, error) {
	// Build JSON-RPC request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// Create request
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	start := time.Now()
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	if latency > 5*time.Second {
		fmt.Printf("rpc slow request to %s: %s\n", endpoint, latency)
	}

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse response
	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func (c *ChainClient) getEndpoint() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.endpoints[c.current]
}

func (c *ChainClient) nextEndpoint() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = (c.current + 1) % len(c.endpoints)
}

// ============================================================================
// High-Level API Methods
// ============================================================================

func (m *RPCManager) GetBlockNumber(chainID string) (string, error) {
	result, err := m.SendRequest(chainID, "eth_blockNumber", []interface{}{})
	if err != nil {
		return "", err
	}

	var blockNum string
	if err := json.Unmarshal(result, &blockNum); err != nil {
		return "", err
	}

	return blockNum, nil
}

func (m *RPCManager) GetBalance(chainID, address, block string) (string, error) {
	if block == "" {
		block = "latest"
	}
	result, err := m.SendRequest(chainID, "eth_getBalance", []interface{}{address, block})
	if err != nil {
		return "", err
	}

	var balance string
	if err := json.Unmarshal(result, &balance); err != nil {
		return "", err
	}

	return balance, nil
}

func (m *RPCManager) GetTransactionCount(chainID, address, block string) (uint64, error) {
	if block == "" {
		block = "latest"
	}
	result, err := m.SendRequest(chainID, "eth_getTransactionCount", []interface{}{address, block})
	if err != nil {
		return 0, err
	}

	var countStr string
	if err := json.Unmarshal(result, &countStr); err != nil {
		return 0, err
	}

	// Parse hex
	count, ok := new(big.Int).SetString(countStr[2:], 16)
	if !ok {
		return 0, fmt.Errorf("failed to parse nonce")
	}

	return count.Uint64(), nil
}

func (m *RPCManager) GetTransactionByHash(chainID, txHash string) (*TransactionByHashResult, error) {
	result, err := m.SendRequest(chainID, "eth_getTransactionByHash", []interface{}{txHash})
	if err != nil {
		return nil, err
	}

	var tx TransactionByHashResult
	if err := json.Unmarshal(result, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

func (m *RPCManager) GetTransactionReceipt(chainID, txHash string) (*TransactionReceipt, error) {
	result, err := m.SendRequest(chainID, "eth_getTransactionReceipt", []interface{}{txHash})
	if err != nil {
		return nil, err
	}

	var receipt TransactionReceipt
	if err := json.Unmarshal(result, &receipt); err != nil {
		return nil, err
	}

	return &receipt, nil
}

func (m *RPCManager) Call(chainID string, to string, data string, block string) (string, error) {
	if block == "" {
		block = "latest"
	}

	callObj := map[string]interface{}{
		"to":   to,
		"data": data,
	}

	result, err := m.SendRequest(chainID, "eth_call", []interface{}{callObj, block})
	if err != nil {
		return "", err
	}

	var resultStr string
	if err := json.Unmarshal(result, &resultStr); err != nil {
		return "", err
	}

	return resultStr, nil
}

func (m *RPCManager) EstimateGas(chainID string, from, to, value, data string) (string, error) {
	callObj := map[string]interface{}{}
	if from != "" {
		callObj["from"] = from
	}
	if to != "" {
		callObj["to"] = to
	}
	if value != "" {
		callObj["value"] = value
	}
	if data != "" {
		callObj["data"] = data
	}

	result, err := m.SendRequest(chainID, "eth_estimateGas", []interface{}{callObj})
	if err != nil {
		return "", err
	}

	var gasStr string
	if err := json.Unmarshal(result, &gasStr); err != nil {
		return "", err
	}

	return gasStr, nil
}

func (m *RPCManager) GetGasPrice(chainID string) (string, error) {
	result, err := m.SendRequest(chainID, "eth_gasPrice", []interface{}{})
	if err != nil {
		return "", err
	}

	var gasPrice string
	if err := json.Unmarshal(result, &gasPrice); err != nil {
		return "", err
	}

	return gasPrice, nil
}

func (m *RPCManager) SendRawTransaction(chainID, signedTx string) (string, error) {
	result, err := m.SendRequest(chainID, "eth_sendRawTransaction", []interface{}{signedTx})
	if err != nil {
		return "", err
	}

	var txHash string
	if err := json.Unmarshal(result, &txHash); err != nil {
		return "", err
	}

	return txHash, nil
}

func (m *RPCManager) GetCode(chainID, address, block string) (string, error) {
	if block == "" {
		block = "latest"
	}
	result, err := m.SendRequest(chainID, "eth_getCode", []interface{}{address, block})
	if err != nil {
		return "", err
	}

	var code string
	if err := json.Unmarshal(result, &code); err != nil {
		return "", err
	}

	return code, nil
}

// ============================================================================
// EIP-1559 Support
// ============================================================================

type FeeHistoryResult struct {
	BaseFeePerGas        []string `json:"baseFeePerGas"`
	GasUsedRatio         []float64 `json:"gasUsedRatio"`
	OldestBlock          string    `json:"oldestBlock"`
	Reward               []string  `json:"reward"`
}

func (m *RPCManager) GetFeeHistory(chainID string, blockCount uint64) (*FeeHistoryResult, error) {
	result, err := m.SendRequest(chainID, "eth_feeHistory", []interface{}{
		fmt.Sprintf("0x%x", blockCount),
		"latest",
		[]float64{25.0, 50.0, 75.0},
	})
	if err != nil {
		return nil, err
	}

	var feeHistory FeeHistoryResult
	if err := json.Unmarshal(result, &feeHistory); err != nil {
		return nil, err
	}

	return &feeHistory, nil
}

func (m *RPCManager) GetMaxPriorityFeePerGas(chainID string) (string, error) {
	result, err := m.SendRequest(chainID, "eth_maxPriorityFeePerGas", []interface{}{})
	if err != nil {
		return "", err
	}

	var fee string
	if err := json.Unmarshal(result, &fee); err != nil {
		return "", err
	}

	return fee, nil
}

// ============================================================================
// WebSocket Support
// ============================================================================

type WSClient struct {
	conn      *websocket.Conn
	chainID   string
	subs      map[string]bool
	mu        sync.RWMutex
	handler   func(string, json.RawMessage)
}

func NewWSClient(url, chainID string, handler func(string, json.RawMessage)) (*WSClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	client := &WSClient{
		conn:    conn,
		chainID: chainID,
		subs:    make(map[string]bool),
		handler: handler,
	}

	go client.readLoop()

	return client, nil
}

func (c *WSClient) Subscribe(event string, params interface{}) error {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_subscribe",
		"params":  []interface{}{event, params},
		"id":      1,
	}

	return c.conn.WriteJSON(req)
}

func (c *WSClient) readLoop() {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue
		}

		if method, ok := resp["method"].(string); ok && c.handler != nil {
			if params, ok := resp["params"].(json.RawMessage); ok {
				c.handler(method, params)
			}
		}
	}
}

func (c *WSClient) Close() error {
	return c.conn.Close()
}

// ============================================================================
// HTTP Handlers
// ============================================================================

type RPCServer struct {
	manager *RPCManager
	server  *gin.Engine
}

func NewRPCServer(manager *RPCManager) *RPCServer {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	server := &RPCServer{
		manager: manager,
		server:  r,
	}

	// API routes
	r.GET("/health", server.health)
	r.GET("/chains", server.listChains)
	r.GET("/chains/:id", server.getChain)
	r.GET("/metrics", server.metrics)

	// RPC proxy
	r.POST("/rpc/:chain", server.rpcProxy)
	r.POST("/rpc/:chain/:method", server.rpcMethod)

	return server
}

func (s *RPCServer) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"timestamp": time.Now().Unix(),
	})
}

func (s *RPCServer) listChains(c *gin.Context) {
	chains := make([]map[string]interface{}, 0)
	for id, config := range s.manager.chains {
		status := s.manager.healthStatus[id]
		chains = append(chains, map[string]interface{}{
			"id":           id,
			"name":         config.Name,
			"symbol":       config.Symbol,
			"isHealthy":    status.IsHealthy,
			"avgLatency":   status.AvgLatencyMs,
			"successRate":  status.SuccessRate,
		})
	}
	c.JSON(http.StatusOK, chains)
}

func (s *RPCServer) getChain(c *gin.Context) {
	chainID := c.Param("id")
	config, ok := s.manager.chains[chainID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "chain not found"})
		return
	}

	status := s.manager.healthStatus[chainID]
	c.JSON(http.StatusOK, gin.H{
		"config":  config,
		"status":  status,
	})
}

func (s *RPCServer) metrics(c *gin.Context) {
	total := atomic.LoadUint64(&s.manager.requestCount)
	failed := atomic.LoadUint64(&s.manager.failureCount)

	c.JSON(http.StatusOK, gin.H{
		"totalRequests":    total,
		"failedRequests":   failed,
		"successRate":     float64(total-failed) / float64(total) * 100,
		"uptimeSeconds":   time.Since(time.Now()).Seconds(),
	})
}

func (s *RPCServer) rpcProxy(c *gin.Context) {
	chainID := c.Param("chain")

	var req JSONRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.manager.SendRequest(chainID, req.Method, req.Params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"jsonrpc": "2.0",
			"error":   RPCError{-32603, err.Error()},
			"id":      req.ID,
		})
		return
	}

	c.JSON(http.StatusOK, JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	})
}

func (s *RPCServer) rpcMethod(c *gin.Context) {
	chainID := c.Param("chain")
	method := c.Param("method")

	var params interface{}
	c.ShouldBindJSON(&params)

	result, err := s.manager.SendRequest(chainID, method, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": string(result)})
}

func (s *RPCServer) Run(addr string) error {
	return s.server.Run(addr)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	// Initialize RPC manager
	manager := NewRPCManager("")

	// Create and start server
	server := NewRPCServer(manager)

	fmt.Println("Starting RPC server on :8080...")
	if err := server.Run(":8080"); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
