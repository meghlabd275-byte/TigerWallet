package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ListenAddr    string
	CheckInterval time.Duration
	Timeout       time.Duration
	MaxRetries    int
	EnableAutoFailover bool
}

var config = Config{
	ListenAddr:      getEnv("RPC_MANAGER_LISTEN_ADDR", ":8087"),
	CheckInterval:   time.Second * 30,
	Timeout:         time.Second * 10,
	MaxRetries:      3,
	EnableAutoFailover: true,
}

// ============================================================================
// Models
// ============================================================================

type BlockchainNetwork struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Symbol         string            `json:"symbol"`
	ChainID        int64             `json:"chain_id"`
	Type           string            `json:"type"` // evm, solana, cosmos, etc.
	RPCEndpoints   []RPCEndpoint     `json:"rpc_endpoints"`
	ExplorerURL    string            `json:"explorer_url"`
	IconURL        string            `json:"icon_url"`
	Status         string            `json:"status"` // active, inactive, maintenance
}

type RPCEndpoint struct {
	URL          string         `json:"url"`
	Name         string         `json:"name"`
	Provider     string         `json:"provider"`
	Region       string         `json:"region"`
	Weight       int            `json:"weight"`
	Status       string         `json:"status"` // healthy, degraded, down
	Latency      float64        `json:"latency"` // ms
	SuccessRate  float64        `json:"success_rate"`
	Requests     int64          `json:"requests"`
	Errors       int64           `json:"errors"`
	LastCheck    time.Time       `json:"last_check"`
	ResponseTimes []float64      `json:"response_times"`
	CostPerMillion float64      `json:"cost_per_million"`
	Priority     int            `json:"priority"`
}

type RPCRequest struct {
	NetworkID   string            `json:"network_id"`
	Method      string            `json:"method"`
	Params      []interface{}     `json:"params"`
	EndpointID string            `json:"endpoint_id,omitempty"`
}

type RPCResponse struct {
	Result    interface{} `json:"result,omitempty"`
	Error     *RPCError   `json:"error,omitempty"`
	Endpoint  string     `json:"endpoint"`
	Latency   float64    `json:"latency"`
	Timestamp time.Time  `json:"timestamp"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type NetworkHealth struct {
	NetworkID     string           `json:"network_id"`
	Status        string           `json:"status"` // healthy, degraded, down
	ActiveEndpoint *RPCEndpoint    `json:"active_endpoint"`
	Endpoints     []RPCEndpoint   `json:"endpoints"`
	AvgLatency    float64          `json:"avg_latency"`
	TotalRequests int64           `json:"total_requests"`
	SuccessRate   float64         `json:"success_rate"`
}

type NodeStats struct {
	TotalNetworks    int `json:"total_networks"`
	ActiveNetworks   int `json:"active_networks"`
	TotalEndpoints   int `json:"total_endpoints"`
	HealthyEndpoints int `json:"healthy_endpoints"`
	DegradedEndpoints int `json:"degraded_endpoints"`
	DownEndpoints    int `json:"down_endpoints"`
}

// ============================================================================
// RPC Node Manager
// ============================================================================

type RPCNodeManager struct {
	networks  map[string]*BlockchainNetwork
	networksMu sync.RWMutex
	health    map[string]*NetworkHealth
	healthMu  sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	stats     NodeStats
	statsMu   sync.RWMutex
}

func NewRPCNodeManager() *RPCNodeManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &RPCNodeManager{
		networks: make(map[string]*BlockchainNetwork),
		health:   make(map[string]*NetworkHealth),
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// Initialize with default networks
	manager.initializeDefaultNetworks()
	
	return manager
}

func (m *RPCNodeManager) initializeDefaultNetworks() {
	networks := []BlockchainNetwork{
		// EVM Chains
		{
			ID: "ethereum", Name: "Ethereum", Symbol: "ETH", ChainID: 1, Type: "evm",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://eth.llamarpc.com", Name: "Llama RPC", Provider: "Llama", Region: "global", Weight: 100, Priority: 1, CostPerMillion: 0.0},
				{URL: "https://mainnet.infura.io/v3/YOUR_KEY", Name: "Infura", Provider: "Infura", Region: "us-east", Weight: 90, Priority: 2, CostPerMillion: 50.0},
				{URL: "https://eth-mainnet.g.alchemy.com/v3/YOUR_KEY", Name: "Alchemy", Provider: "Alchemy", Region: "us-west", Weight: 90, Priority: 3, CostPerMillion: 25.0},
				{URL: "https://rpc.ankr.com/eth", Name: "Ankr", Provider: "Ankr", Region: "global", Weight: 80, Priority: 4, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://etherscan.io", Status: "active",
		},
		{
			ID: "bsc", Name: "BNB Smart Chain", Symbol: "BNB", ChainID: 56, Type: "evm",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://bsc-dataseed.binance.org", Name: "BSC DataSeed", Provider: "Binance", Region: "singapore", Weight: 100, Priority: 1, CostPerMillion: 0.0},
				{URL: "https://bsc-rpc.gateway.pokt.network", Name: "Pocket", Provider: "Pocket", Region: "global", Weight: 90, Priority: 2, CostPerMillion: 0.0},
				{URL: "https://rpc.ankr.com/bsc", Name: "Ankr", Provider: "Ankr", Region: "global", Weight: 80, Priority: 3, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://bscscan.com", Status: "active",
		},
		{
			ID: "polygon", Name: "Polygon", Symbol: "MATIC", ChainID: 137, Type: "evm",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://polygon-rpc.com", Name: "Polygon RPC", Provider: "Polygon", Weight: 100, Priority: 1, CostPerMillion: 0.0},
				{URL: "https://rpc.ankr.com/polygon", Name: "Ankr", Provider: "Ankr", Region: "global", Weight: 90, Priority: 2, CostPerMillion: 0.0},
				{URL: "https://polygon-mainnet.g.alchemy.com/v3/YOUR_KEY", Name: "Alchemy", Provider: "Alchemy", Region: "us-east", Weight: 85, Priority: 3, CostPerMillion: 25.0},
			},
			ExplorerURL: "https://polygonscan.com", Status: "active",
		},
		{
			ID: "arbitrum", Name: "Arbitrum One", Symbol: "ETH", ChainID: 42161, Type: "evm",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://arb1.arbitrum.io/rpc", Name: "Arbitrum Nova", Provider: "Arbitrum", Weight: 100, Priority: 1, CostPerMillion: 0.0},
				{URL: "https://rpc.ankr.com/arbitrum", Name: "Ankr", Provider: "Ankr", Region: "global", Weight: 90, Priority: 2, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://arbiscan.io", Status: "active",
		},
		{
			ID: "optimism", Name: "Optimism", Symbol: "ETH", ChainID: 10, Type: "evm",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://mainnet.optimism.io", Name: "Optimism", Provider: "Optimism", Weight: 100, Priority: 1, CostPerMillion: 0.0},
				{URL: "https://rpc.ankr.com/optimism", Name: "Ankr", Provider: "Ankr", Region: "global", Weight: 90, Priority: 2, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://optimistic.etherscan.io", Status: "active",
		},
		{
			ID: "avalanche", Name: "Avalanche C-Chain", Symbol: "AVAX", ChainID: 43114, Type: "evm",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://api.avax.network/ext/bc/C/rpc", Name: "Avalanche", Provider: "Avalanche", Weight: 100, Priority: 1, CostPerMillion: 0.0},
				{URL: "https://rpc.ankr.com/avalanche", Name: "Ankr", Provider: "Ankr", Region: "global", Weight: 90, Priority: 2, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://snowtrace.io", Status: "active",
		},
		{
			ID: "base", Name: "Base", Symbol: "ETH", ChainID: 8453, Type: "evm",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://mainnet.base.org", Name: "Base", Provider: "Base", Weight: 100, Priority: 1, CostPerMillion: 0.0},
				{URL: "https://rpc.ankr.com/base", Name: "Ankr", Provider: "Ankr", Region: "global", Weight: 90, Priority: 2, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://basescan.org", Status: "active",
		},
		{
			ID: "solana", Name: "Solana", Symbol: "SOL", ChainID: 0, Type: "solana",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://api.mainnet-beta.solana.com", Name: "Solana", Provider: "Solana", Weight: 100, Priority: 1, CostPerMillion: 0.0},
				{URL: "https://rpc.ankr.com/solana", Name: "Ankr", Provider: "Ankr", Region: "global", Weight: 90, Priority: 2, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://solscan.io", Status: "active",
		},
		{
			ID: "tron", Name: "Tron", Symbol: "TRX", ChainID: 195, Type: "tron",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://api.trongrid.io", Name: "TronGrid", Provider: "Tron", Weight: 100, Priority: 1, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://tronscan.org", Status: "active",
		},
		{
			ID: "near", Name: "Near Protocol", Symbol: "NEAR", ChainID: 0, Type: "near",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://rpc.mainnet.near.org", Name: "Near", Provider: "Near", Weight: 100, Priority: 1, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://explorer.near.org", Status: "active",
		},
		{
			ID: "aptos", Name: "Aptos", Symbol: "APT", ChainID: 0, Type: "aptos",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://fullnode.mainnet.aptoslabs.com", Name: "Aptos", Provider: "Aptos", Weight: 100, Priority: 1, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://explorer.aptoslabs.com", Status: "active",
		},
		{
			ID: "cosmos", Name: "Cosmos Hub", Symbol: "ATOM", ChainID: 0, Type: "cosmos",
			RPCEndpoints: []RPCEndpoint{
				{URL: "https://rpc.cosmoshub4.theta-testnet.xyz:443", Name: "Cosmos", Provider: "Cosmos", Weight: 100, Priority: 1, CostPerMillion: 0.0},
			},
			ExplorerURL: "https://mintscan.io/cosmos", Status: "active",
		},
	}
	
	for i := range networks {
		network := &networks[i]
		m.networks[network.ID] = network
		
		// Initialize health
		m.health[network.ID] = &NetworkHealth{
			NetworkID:   network.ID,
			Status:      "healthy",
			Endpoints:   network.RPCEndpoints,
		}
	}
	
	m.updateStats()
}

func (m *RPCNodeManager) Start() error {
	fmt.Println("Starting RPC Node Manager...")
	
	// Start health check loop
	go m.healthCheckLoop()
	
	// Start HTTP server
	go m.startHTTPServer()
	
	fmt.Println("RPC Node Manager started successfully")
	return nil
}

func (m *RPCNodeManager) Stop() {
	fmt.Println("Stopping RPC Node Manager...")
	m.cancel()
	fmt.Println("RPC Node Manager stopped")
}

func (m *RPCNodeManager) startHTTPServer() {
	router := gin.Default()
	
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})
	
	router.GET("/networks", m.getNetworksHandler)
	router.GET("/networks/:id", m.getNetworkHandler)
	router.POST("/networks", m.addNetworkHandler)
	router.PUT("/networks/:id", m.updateNetworkHandler)
	router.DELETE("/networks/:id", m.deleteNetworkHandler)
	
	router.GET("/health", m.getHealthHandler)
	router.GET("/health/:network_id", m.getNetworkHealthHandler)
	
	router.POST("/request", m.handleRPCRequest)
	
	router.GET("/stats", m.getStatsHandler)
	
	router.GET("/endpoints", m.getEndpointsHandler)
	router.GET("/endpoints/:network_id", m.getEndpointsByNetworkHandler)
	router.POST("/endpoints/:network_id", m.addEndpointHandler)
	router.DELETE("/endpoints/:network_id/:endpoint_url", m.deleteEndpointHandler)
	
	fmt.Printf("RPC Node Manager API starting on %s\n", config.ListenAddr)
	router.Run(config.ListenAddr)
}

// ============================================================================
// Health Check
// ============================================================================

func (m *RPCNodeManager) healthCheckLoop() {
	ticker := time.NewTicker(config.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAllEndpoints()
		}
	}
}

func (m *RPCNodeManager) checkAllEndpoints() {
	m.networksMu.RLock()
	networks := make([]*BlockchainNetwork, 0, len(m.networks))
	for _, n := range m.networks {
		networks = append(networks, n)
	}
	m.networksMu.RUnlock()
	
	for _, network := range networks {
		m.checkNetworkEndpoints(network)
	}
	
	m.updateStats()
}

func (m *RPCNodeManager) checkNetworkEndpoints(network *BlockchainNetwork) {
	m.healthMu.Lock()
	health := m.health[network.ID]
	if health == nil {
		health = &NetworkHealth{NetworkID: network.ID, Endpoints: network.RPCEndpoints}
		m.health[network.ID] = health
	}
	m.healthMu.Unlock()
	
	var totalLatency float64
	var healthyCount int
	var totalRequests int64
	
	for i := range network.RPCEndpoints {
		endpoint := &network.RPCEndpoints[i]
		
		// Simulate health check
	latency := m.checkEndpoint(endpoint)
	healthyCount++
		totalLatency += latency
		
		// Update endpoint status
		if latency < 1000 { // Less than 1 second
			endpoint.Status = "healthy"
		} else if latency < 5000 { // Less than 5 seconds
			endpoint.Status = "degraded"
		} else {
			endpoint.Status = "down"
		}
		
		endpoint.LastCheck = time.Now()
		endpoint.Latency = latency
		totalRequests += endpoint.Requests
	}
	
	// Update health
	avgLatency := float64(0)
	if healthyCount > 0 {
		avgLatency = totalLatency / float64(healthyCount)
	}
	
	m.healthMu.Lock()
	health.AvgLatency = avgLatency
	health.TotalRequests = totalRequests
	
	if avgLatency < 1000 {
		health.Status = "healthy"
	} else if avgLatency < 5000 {
		health.Status = "degraded"
	} else {
		health.Status = "down"
	}
	
	// Find active endpoint
	health.ActiveEndpoint = m.selectBestEndpoint(network.RPCEndpoints)
	m.healthMu.Unlock()
}

func (m *RPCNodeManager) checkEndpoint(endpoint *RPCEndpoint) float64 {
	// Simulate endpoint check with random latency
	// In production, would make actual HTTP requests
	latency := rand.Float64() * 500 // 0-500ms
	
	// Add response time to history
	endpoint.ResponseTimes = append(endpoint.ResponseTimes, latency)
	if len(endpoint.ResponseTimes) > 100 {
		endpoint.ResponseTimes = endpoint.ResponseTimes[len(endpoint.ResponseTimes)-100:]
	}
	
	return latency
}

func (m *RPCNodeManager) selectBestEndpoint(endpoints []RPCEndpoint) *RPCEndpoint {
	// Select best endpoint based on latency, success rate, and priority
	var best *RPCEndpoint
	var bestScore float64 = -1
	
	for i := range endpoints {
		endpoint := &endpoints[i]
		if endpoint.Status == "down" {
			continue
		}
		
		// Calculate score (higher is better)
		latencyScore := 1.0 / (endpoint.Latency + 1)
		successScore := endpoint.SuccessRate
		priorityScore := float64(endpoint.Priority)
		
		score := latencyScore * successScore * priorityScore * 100
		
		if score > bestScore {
			bestScore = score
			best = endpoint
		}
	}
	
	return best
}

func (m *RPCNodeManager) updateStats() {
	m.networksMu.RLock()
	totalNetworks := len(m.networks)
	m.networksMu.RUnlock()
	
	m.healthMu.RLock()
	activeNetworks := 0
	totalEndpoints := 0
	healthyEndpoints := 0
	degradedEndpoints := 0
	downEndpoints := 0
	
	for _, health := range m.health {
		if health.Status != "down" {
			activeNetworks++
		}
		for _, ep := range health.Endpoints {
			totalEndpoints++
			switch ep.Status {
			case "healthy":
				healthyEndpoints++
			case "degraded":
				degradedEndpoints++
			case "down":
				downEndpoints++
			}
		}
	}
	m.healthMu.Unlock()
	
	m.statsMu.Lock()
	m.stats = NodeStats{
		TotalNetworks:     totalNetworks,
		ActiveNetworks:   activeNetworks,
		TotalEndpoints:   totalEndpoints,
		HealthyEndpoints: healthyEndpoints,
		DegradedEndpoints: degradedEndpoints,
		DownEndpoints:    downEndpoints,
	}
	m.statsMu.Unlock()
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (m *RPCNodeManager) getNetworksHandler(c *gin.Context) {
	m.networksMu.RLock()
	networks := make([]BlockchainNetwork, 0, len(m.networks))
	for _, n := range m.networks {
		networks = append(networks, *n)
	}
	m.networksMu.RUnlock()
	
	c.JSON(200, networks)
}

func (m *RPCNodeManager) getNetworkHandler(c *gin.Context) {
	id := c.Param("id")
	
	m.networksMu.RLock()
	network, ok := m.networks[id]
	m.networksMu.RUnlock()
	
	if !ok {
		c.JSON(404, gin.H{"error": "network not found"})
		return
	}
	
	c.JSON(200, network)
}

func (m *RPCNodeManager) addNetworkHandler(c *gin.Context) {
	var network BlockchainNetwork
	if err := c.ShouldBindJSON(&network); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	m.networksMu.Lock()
	m.networks[network.ID] = &network
	m.networksMu.Unlock()
	
	m.healthMu.Lock()
	m.health[network.ID] = &NetworkHealth{
		NetworkID: network.ID,
		Status:    "healthy",
		Endpoints: network.RPCEndpoints,
	}
	m.healthMu.Unlock()
	
	m.updateStats()
	
	c.JSON(200, network)
}

func (m *RPCNodeManager) updateNetworkHandler(c *gin.Context) {
	id := c.Param("id")
	
	var network BlockchainNetwork
	if err := c.ShouldBindJSON(&network); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	m.networksMu.Lock()
	if _, ok := m.networks[id]; !ok {
		m.networksMu.Unlock()
		c.JSON(404, gin.H{"error": "network not found"})
		return
	}
	
	m.networks[id] = &network
	m.networksMu.Unlock()
	
	// Update health
	m.healthMu.Lock()
	m.health[id] = &NetworkHealth{
		NetworkID: network.ID,
		Status:    "healthy",
		Endpoints: network.RPCEndpoints,
	}
	m.healthMu.Unlock()
	
	m.updateStats()
	
	c.JSON(200, network)
}

func (m *RPCNodeManager) deleteNetworkHandler(c *gin.Context) {
	id := c.Param("id")
	
	m.networksMu.Lock()
	if _, ok := m.networks[id]; !ok {
		m.networksMu.Unlock()
		c.JSON(404, gin.H{"error": "network not found"})
		return
	}
	
	delete(m.networks, id)
	m.networksMu.Unlock()
	
	m.healthMu.Lock()
	delete(m.health, id)
	m.healthMu.Unlock()
	
	m.updateStats()
	
	c.JSON(200, gin.H{"status": "ok"})
}

func (m *RPCNodeManager) getHealthHandler(c *gin.Context) {
	m.healthMu.RLock()
	healthList := make([]NetworkHealth, 0, len(m.health))
	for _, h := range m.health {
		healthList = append(healthList, *h)
	}
	m.healthMu.RUnlock()
	
	c.JSON(200, healthList)
}

func (m *RPCNodeManager) getNetworkHealthHandler(c *gin.Context) {
	networkID := c.Param("network_id")
	
	m.healthMu.RLock()
	health, ok := m.health[networkID]
	m.healthMu.RUnlock()
	
	if !ok {
		c.JSON(404, gin.H{"error": "network health not found"})
		return
	}
	
	c.JSON(200, health)
}

func (m *RPCNodeManager) handleRPCRequest(c *gin.Context) {
	var req RPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	// Get network
	m.networksMu.RLock()
	network, ok := m.networks[req.NetworkID]
	m.networksMu.RUnlock()
	
	if !ok {
		c.JSON(404, gin.H{"error": "network not found"})
		return
	}
	
	// Select endpoint
	var endpoint *RPCEndpoint
	if req.EndpointID != "" {
		for i := range network.RPCEndpoints {
			if network.RPCEndpoints[i].URL == req.EndpointID {
				endpoint = &network.RPCEndpoints[i]
				break
			}
		}
	} else {
		m.healthMu.RLock()
		health := m.health[req.NetworkID]
		m.healthMu.RUnlock()
		
		if health != nil {
			endpoint = health.ActiveEndpoint
		}
	}
	
	if endpoint == nil {
		c.JSON(500, gin.H{"error": "no available endpoint"})
		return
	}
	
	// Make request (simulated)
	startTime := time.Now()
	latency := rand.Float64() * 100 // Simulated latency
	
	// Update endpoint stats
	atomic.AddInt64(&endpoint.Requests, 1)
	
	response := RPCResponse{
		Result:   fmt.Sprintf("Response from %s", req.Method),
		Endpoint: endpoint.URL,
		Latency:  latency,
		Timestamp: startTime,
	}
	
	c.JSON(200, response)
}

func (m *RPCNodeManager) getStatsHandler(c *gin.Context) {
	m.statsMu.RLock()
	stats := m.stats
	m.statsMu.RUnlock()
	
	c.JSON(200, stats)
}

func (m *RPCNodeManager) getEndpointsHandler(c *gin.Context) {
	m.networksMu.RLock()
	var endpoints []RPCEndpoint
	for _, n := range m.networks {
		endpoints = append(endpoints, n.RPCEndpoints...)
	}
	m.networksMu.RUnlock()
	
	c.JSON(200, endpoints)
}

func (m *RPCNodeManager) getEndpointsByNetworkHandler(c *gin.Context) {
	networkID := c.Param("network_id")
	
	m.networksMu.RLock()
	network, ok := m.networks[networkID]
	m.networksMu.RUnlock()
	
	if !ok {
		c.JSON(404, gin.H{"error": "network not found"})
		return
	}
	
	c.JSON(200, network.RPCEndpoints)
}

func (m *RPCNodeManager) addEndpointHandler(c *gin.Context) {
	networkID := c.Param("network_id")
	
	m.networksMu.RLock()
	network, ok := m.networks[networkID]
	m.networksMu.RUnlock()
	
	if !ok {
		c.JSON(404, gin.H{"error": "network not found"})
		return
	}
	
	var endpoint RPCEndpoint
	if err := c.ShouldBindJSON(&endpoint); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	m.networksMu.Lock()
	network.RPCEndpoints = append(network.RPCEndpoints, endpoint)
	m.networksMu.Unlock()
	
	m.updateStats()
	
	c.JSON(200, endpoint)
}

func (m *RPCNodeManager) deleteEndpointHandler(c *gin.Context) {
	networkID := c.Param("network_id")
	endpointURL := c.Param("endpoint_url")
	endpointURL, _ = endpointURL, nil
	
	m.networksMu.RLock()
	network, ok := m.networks[networkID]
	m.networksMu.RUnlock()
	
	if !ok {
		c.JSON(404, gin.H{"error": "network not found"})
		return
	}
	
	m.networksMu.Lock()
	for i, ep := range network.RPCEndpoints {
		if ep.URL == endpointURL {
			network.RPCEndpoints = append(network.RPCEndpoints[:i], network.RPCEndpoints[i+1:]...)
			break
		}
	}
	m.networksMu.Unlock()
	
	m.updateStats()
	
	c.JSON(200, gin.H{"status": "ok"})
}

// ============================================================================
// Helper Functions
// ============================================================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Main
// ============================================================================

func main() {
	rand.Seed(time.Now().UnixNano())
	
	fmt.Println("============================================")
	fmt.Println("TigerWallet RPC Node Manager")
	fmt.Println("============================================")
	
	manager := NewRPCNodeManager()
	
	if err := manager.Start(); err != nil {
		fmt.Printf("Failed to start RPC manager: %v\n", err)
		os.Exit(1)
	}
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	fmt.Println("\nShutting down...")
	manager.Stop()
	
	fmt.Println("RPC Node Manager stopped")
}
