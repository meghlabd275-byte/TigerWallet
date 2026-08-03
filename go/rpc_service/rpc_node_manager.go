/**
 * TigerWallet RPC Node Manager
 * 
 * Manages RPC connections for 300+ blockchains with automatic failover,
 * load balancing, and health monitoring.
 * Built with Go for high-load distributed operations.
 */

package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// ============================================================================
// Types
// ============================================================================

// BlockchainNetwork represents a blockchain network configuration
type BlockchainNetwork struct {
	ID              uint64    `json:"id"`
	Name            string    `json:"name"`
	Symbol          string    `json:"symbol"`
	Type            string    `json:"type"` // evm, solana, bitcoin, cosmos, etc.
	ChainID         int64     `json:"chain_id"`
	RPCURLs         []string  `json:"rpc_urls"`
	ExplorerURL     string    `json:"explorer_url"`
	WSSURL          string    `json:"wss_url"`
	NativeCurrency  string    `json:"native_currency"`
	Decimals        int       `json:"decimals"`
	Confirmations   int       `json:"confirmations"`
	IsTestnet       bool      `json:"is_testnet"`
	IconURL         string    `json:"icon_url"`
}

// RPCNode represents an RPC node connection
type RPCNode struct {
	URL            string
	Name           string
	IsActive       bool
	Latency        time.Duration
	SuccessRate    float64
	RequestsCount  int64
	ErrorCount     int64
	LastError      error
	LastUsed       time.Time
	Client         *rpc.Client
	EthClient      *ethclient.Client
	mu             sync.RWMutex
}

// NodePool manages a pool of RPC nodes for a blockchain
type NodePool struct {
	Network    *BlockchainNetwork
	Nodes      []*RPCNode
	CurrentIdx int
	mu         sync.RWMutex
}

// RPCManager manages all blockchain RPC connections
type RPCManager struct {
	mu           sync.RWMutex
	Networks     map[uint64]*BlockchainNetwork
	NodePools    map[uint64]*NodePool
	HealthCheck  *HealthChecker
	Metrics      *MetricsCollector
}

// HealthChecker monitors node health
type HealthChecker struct {
	Interval time.Duration
	Nodes    map[string]*HealthStatus
	mu       sync.RWMutex
}

// HealthStatus represents node health status
type HealthStatus struct {
	NodeURL      string
	IsHealthy   bool
	Latency     time.Duration
	LastCheck   time.Time
	ErrorCount  int
	SuccessRate float64
}

// MetricsCollector collects RPC metrics
type MetricsCollector struct {
	mu           sync.RWMutex
	RequestCount map[string]int64
	LatencySum   map[string]time.Duration
	ErrorCount   map[string]int64
}

// ============================================================================
// RPC Methods
// ============================================================================

// BlockNumber returns the current block number
func (n *RPCNode) BlockNumber(ctx context.Context) (uint64, error) {
	var result string
	err := n.Client.CallContext(ctx, &result, "eth_blockNumber")
	if err != nil {
		n.recordError(err)
		return 0, err
	}
	return parseBlockNumber(result)
}

// GetBalance returns the balance of an address
func (n *RPCNode) GetBalance(ctx context.Context, address common.Address) (*big.Int, error) {
	var result string
	err := n.Client.CallContext(ctx, &result, "eth_getBalance", address.String(), "latest")
	if err != nil {
		n.recordError(err)
		return nil, err
	}
	return parseBigInt(result)
}

// GetTransactionCount returns the nonce for an address
func (n *RPCNode) GetTransactionCount(ctx context.Context, address common.Address) (uint64, error) {
	var result string
	err := n.Client.CallContext(ctx, &result, "eth_getTransactionCount", address.String(), "latest")
	if err != nil {
		n.recordError(err)
		return 0, err
	}
	return parseNonce(result)
}

// GetTransactionByHash returns a transaction by hash
func (n *RPCNode) GetTransactionByHash(ctx context.Context, hash common.Hash) (*Transaction, error) {
	var result map[string]interface{}
	err := n.Client.CallContext(ctx, &result, "eth_getTransactionByHash", hash.String())
	if err != nil {
		n.recordError(err)
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return parseTransaction(result)
}

// GetTransactionReceipt returns the receipt for a transaction
func (n *RPCNode) GetTransactionReceipt(ctx context.Context, hash common.Hash) (*TransactionReceipt, error) {
	var result map[string]interface{}
	err := n.Client.CallContext(ctx, &result, "eth_getTransactionReceipt", hash.String())
	if err != nil {
		n.recordError(err)
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return parseReceipt(result)
}

// Call executes a smart contract call
func (n *RPCNode) Call(ctx context.Context, msg ethereum.CallMsg) (string, error) {
	var result string
	err := n.Client.CallContext(ctx, &result, "eth_call", toCallMsg(msg), "latest")
	if err != nil {
		n.recordError(err)
		return "", err
	}
	return result, nil
}

// SendRawTransaction sends a signed transaction
func (n *RPCNode) SendRawTransaction(ctx context.Context, signedTx []byte) (common.Hash, error) {
	var result string
	err := n.Client.CallContext(ctx, &result, "eth_sendRawTransaction", common.ToHex(signedTx))
	if err != nil {
		n.recordError(err)
		return common.Hash{}, err
	}
	return common.HexToHash(result), nil
}

// GetBlockByNumber returns a block by number
func (n *RPCNode) GetBlockByNumber(ctx context.Context, number uint64, fullTx bool) (*Block, error) {
	var result map[string]interface{}
	err := n.Client.CallContext(ctx, &result, "eth_getBlockByNumber", fmt.Sprintf("0x%x", number), fullTx)
	if err != nil {
		n.recordError(err)
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return parseBlock(result)
}

// GetGasPrice returns the current gas price
func (n *RPCNode) GetGasPrice(ctx context.Context) (*big.Int, error) {
	var result string
	err := n.Client.CallContext(ctx, &result, "eth_gasPrice")
	if err != nil {
		n.recordError(err)
		return nil, err
	}
	return parseBigInt(result)
}

// EstimateGas estimates gas for a transaction
func (n *RPCNode) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	var result string
	err := n.Client.CallContext(ctx, &result, "eth_estimateGas", toCallMsg(msg))
	if err != nil {
		n.recordError(err)
		return 0, err
	}
	return parseUint64(result)
}

// GetCode returns the contract code at an address
func (n *RPCNode) GetCode(ctx context.Context, address common.Address) (string, error) {
	var result string
	err := n.Client.CallContext(ctx, &result, "eth_getCode", address.String(), "latest")
	if err != nil {
		n.recordError(err)
		return "", err
	}
	return result, nil
}

// GetStorageAt returns the storage value at a given position
func (n *RPCNode) GetStorageAt(ctx context.Context, address common.Address, position common.Hash) (string, error) {
	var result string
	err := n.Client.CallContext(ctx, &result, "eth_getStorageAt", address.String(), position.String(), "latest")
	if err != nil {
		n.recordError(err)
		return "", err
	}
	return result, nil
}

// GetLogs returns logs matching the given filter
func (n *RPCNode) GetLogs(ctx context.Context, filter ethereum.FilterQuery) ([]Log, error) {
	var result []map[string]interface{}
	err := n.Client.CallContext(ctx, &result, "eth_getLogs", toFilterQuery(filter))
	if err != nil {
		n.recordError(err)
		return nil, err
	}
	return parseLogs(result)
}

// SubscribeNewHead subscribes to new block headers
func (n *RPCNode) SubscribeNewHead(ctx context.Context, ch chan *Header) (ethereum.Subscription, error) {
	return n.EthClient.SubscribeNewHead(ctx, ch)
}

// ============================================================================
// Node Pool Methods
// ============================================================================

// NewNodePool creates a new node pool
func NewNodePool(network *BlockchainNetwork) *NodePool {
	nodes := make([]*RPCNode, 0, len(network.RPCURLs))
	for _, url := range network.RPCURLs {
		nodes = append(nodes, &RPCNode{
			URL:         url,
			Name:        fmt.Sprintf("%s-%d", network.Name, len(nodes)),
			IsActive:    true,
			SuccessRate: 1.0,
		})
	}
	return &NodePool{
		Network:    network,
		Nodes:      nodes,
		CurrentIdx: 0,
	}
}

// GetNode returns the next available node
func (p *NodePool) GetNode() *RPCNode {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Try each node starting from current index
	for i := 0; i < len(p.Nodes); i++ {
		idx := (p.CurrentIdx + i) % len(p.Nodes)
		node := p.Nodes[idx]
		if node.IsActive && node.SuccessRate > 0.5 {
			p.CurrentIdx = (idx + 1) % len(p.Nodes)
			return node
		}
	}

	// If no healthy node found, return first active node
	for _, node := range p.Nodes {
		if node.IsActive {
			return node
		}
	}

	return nil
}

// MarkNodeHealthy marks a node as healthy
func (p *NodePool) MarkNodeHealthy(node *RPCNode) {
	node.mu.Lock()
	node.IsActive = true
	node.SuccessRate = min(node.SuccessRate+0.1, 1.0)
	node.mu.Unlock()
}

// MarkNodeUnhealthy marks a node as unhealthy
func (p *NodePool) MarkNodeUnhealthy(node *RPCNode, err error) {
	node.mu.Lock()
	node.IsActive = false
	node.ErrorCount++
	node.LastError = err
	if node.ErrorCount > 5 {
		node.SuccessRate = max(node.SuccessRate-0.2, 0.0)
	}
	node.mu.Unlock()
}

// ============================================================================
// RPC Manager Methods
// ============================================================================

var (
	rpcManager     *RPCManager
	rpcManagerOnce sync.Once
)

// GetRPCManager returns the singleton RPC manager
func GetRPCManager() *RPCManager {
	rpcManagerOnce.Do(func() {
		rpcManager = &RPCManager{
			Networks:  make(map[uint64]*BlockchainNetwork),
			NodePools: make(map[uint64]*NodePool),
			HealthCheck: &HealthChecker{
				Interval: 30 * time.Second,
				Nodes:    make(map[string]*HealthStatus),
			},
			Metrics: &MetricsCollector{
				RequestCount: make(map[string]int64),
				LatencySum:   make(map[string]time.Duration),
				ErrorCount:   make(map[string]int64),
			},
		}
		rpcManager.initDefaultNetworks()
	})
	return rpcManager
}

// initDefaultNetworks initializes default blockchain networks
func (m *RPCManager) initDefaultNetworks() {
	networks := []*BlockchainNetwork{
		// EVM Chains
		{ID: 1, Name: "Ethereum", Symbol: "ETH", Type: "evm", ChainID: 1, RPCURLs: []string{"https://eth.llamarpc.com", "https://eth.public-rpc.com"}, ExplorerURL: "https://etherscan.io", NativeCurrency: "ETH", Decimals: 18, Confirmations: 12},
		{ID: 56, Name: "BNB Smart Chain", Symbol: "BNB", Type: "evm", ChainID: 56, RPCURLs: []string{"https://bsc-dataseed.binance.org", "https://bsc-rpc.publicnode.com"}, ExplorerURL: "https://bscscan.com", NativeCurrency: "BNB", Decimals: 18, Confirmations: 15},
		{ID: 137, Name: "Polygon", Symbol: "MATIC", Type: "evm", ChainID: 137, RPCURLs: []string{"https://polygon-rpc.com", "https://polygon.llamarpc.com"}, ExplorerURL: "https://polygonscan.com", NativeCurrency: "MATIC", Decimals: 18, Confirmations: 15},
		{ID: 42161, Name: "Arbitrum One", Symbol: "ETH", Type: "evm", ChainID: 42161, RPCURLs: []string{"https://arb1.arbitrum.io/rpc", "https://arbitrum-one.publicnode.com"}, ExplorerURL: "https://arbiscan.io", NativeCurrency: "ETH", Decimals: 18, Confirmations: 15},
		{ID: 10, Name: "Optimism", Symbol: "ETH", Type: "evm", ChainID: 10, RPCURLs: []string{"https://mainnet.optimism.io", "https://optimism.publicnode.com"}, ExplorerURL: "https://optimistic.etherscan.io", NativeCurrency: "ETH", Decimals: 18, Confirmations: 15},
		{ID: 43114, Name: "Avalanche C-Chain", Symbol: "AVAX", Type: "evm", ChainID: 43114, RPCURLs: []string{"https://api.avax.network/ext/bc/C/rpc", "https://avalanche-c-chain.publicnode.com"}, ExplorerURL: "https://snowtrace.io", NativeCurrency: "AVAX", Decimals: 18, Confirmations: 15},
		{ID: 8453, Name: "Base", Symbol: "ETH", Type: "evm", ChainID: 8453, RPCURLs: []string{"https://mainnet.base.org", "https://base.publicnode.com"}, ExplorerURL: "https://basescan.org", NativeCurrency: "ETH", Decimals: 18, Confirmations: 15},
		{ID: 250, Name: "Fantom", Symbol: "FTM", Type: "evm", ChainID: 250, RPCURLs: []string{"https://rpc.fantom.network", "https://fantom.publicnode.com"}, ExplorerURL: "https://ftmscan.com", NativeCurrency: "FTM", Decimals: 18, Confirmations: 15},
		{ID: 100, Name: "Gnosis Chain", Symbol: "XDAI", Type: "evm", ChainID: 100, RPCURLs: []string{"https://rpc.gnosischain.com", "https://gnosis.publicnode.com"}, ExplorerURL: "https://gnosisscan.io", NativeCurrency: "XDAI", Decimals: 18, Confirmations: 15},
		{ID: 1284, Name: "Moonbeam", Symbol: "GLMR", Type: "evm", ChainID: 1284, RPCURLs: []string{"https://rpc.api.moonbeam.network", "https://moonbeam.publicnode.com"}, ExplorerURL: "https://moonscan.io", NativeCurrency: "GLMR", Decimals: 18, Confirmations: 15},
		{ID: 1285, Name: "Moonriver", Symbol: "MOVR", Type: "evm", ChainID: 1285, RPCURLs: []string{"https://rpc.api.moonriver.network"}, ExplorerURL: "https://moonriver.moonscan.io", NativeCurrency: "MOVR", Decimals: 18, Confirmations: 15},
		{ID: 42220, Name: "Celo", Symbol: "CELO", Type: "evm", ChainID: 42220, RPCURLs: []string{"https://forno.celo.org", "https://celo.publicnode.com"}, ExplorerURL: "https://explorer.celo.org", NativeCurrency: "CELO", Decimals: 18, Confirmations: 15},
		{ID: 1666600000, Name: "Harmony", Symbol: "ONE", Type: "evm", ChainID: 1666600000, RPCURLs: []string{"https://api.harmony.one", "https://harmony-0-rpc.gateway.pokt.network"}, ExplorerURL: "https://explorer.harmony.one", NativeCurrency: "ONE", Decimals: 18, Confirmations: 15},
		{ID: 1313161554, Name: "Aurora", Symbol: "ETH", Type: "evm", ChainID: 1313161554, RPCURLs: []string{"https://mainnet.aurora.dev", "https://aurora.publicnode.com"}, ExplorerURL: "https://explorer.aurora.dev", NativeCurrency: "ETH", Decimals: 18, Confirmations: 15},
		{ID: 1088, Name: "Metis", Symbol: "METIS", Type: "evm", ChainID: 1088, RPCURLs: []string{"https://andromeda.metis.io/?owner=1088"}, ExplorerURL: "https://andromeda-explorer.metis.io", NativeCurrency: "METIS", Decimals: 18, Confirmations: 15},
		// Testnets
		{ID: 5, Name: "Goerli Testnet", Symbol: "ETH", Type: "evm", ChainID: 5, RPCURLs: []string{"https://goerli.infura.io/v3/"}, ExplorerURL: "https://goerli.etherscan.io", NativeCurrency: "ETH", Decimals: 18, Confirmations: 5, IsTestnet: true},
		{ID: 11155111, Name: "Sepolia Testnet", Symbol: "ETH", Type: "evm", ChainID: 11155111, RPCURLs: []string{"https://sepolia.infura.io/v3/"}, ExplorerURL: "https://sepolia.etherscan.io", NativeCurrency: "ETH", Decimals: 18, Confirmations: 5, IsTestnet: true},
		{ID: 80001, Name: "Mumbai Testnet", Symbol: "MATIC", Type: "evm", ChainID: 80001, RPCURLs: []string{"https://rpc-mumbai.maticvigil.com"}, ExplorerURL: "https://mumbai.polygonscan.com", NativeCurrency: "MATIC", Decimals: 18, Confirmations: 5, IsTestnet: true},
		{ID: 421613, Name: "Arbitrum Goerli", Symbol: "ETH", Type: "evm", ChainID: 421613, RPCURLs: []string{"https://goerli-rollup.arbitrum.io/rpc"}, ExplorerURL: "https://goerli.arbiscan.io", NativeCurrency: "ETH", Decimals: 18, Confirmations: 5, IsTestnet: true},
		{ID: 420, Name: "Optimism Goerli", Symbol: "ETH", Type: "evm", ChainID: 420, RPCURLs: []string{"https://goerli.optimism.io"}, ExplorerURL: "https://goerli-optimism.etherscan.io", NativeCurrency: "ETH", Decimals: 18, Confirmations: 5, IsTestnet: true},
		{ID: 43113, Name: "Avalanche Fuji", Symbol: "AVAX", Type: "evm", ChainID: 43113, RPCURLs: []string{"https://api.avax-test.network/ext/bc/C/rpc"}, ExplorerURL: "https://testnet.snowtrace.io", NativeCurrency: "AVAX", Decimals: 18, Confirmations: 5, IsTestnet: true},
		{ID: 84531, Name: "Base Goerli", Symbol: "ETH", Type: "evm", ChainID: 84531, RPCURLs: []string{"https://goerli.base.org"}, ExplorerURL: "https://goerli.basescan.org", NativeCurrency: "ETH", Decimals: 18, Confirmations: 5, IsTestnet: true},
		// Additional popular chains
		{ID: 100, Name: "Gnosis", Symbol: "XDAI", Type: "evm", ChainID: 100, RPCURLs: []string{"https://rpc.gnosischain.com"}, ExplorerURL: "https://gnosisscan.io", NativeCurrency: "XDAI", Decimals: 18, Confirmations: 12},
		{ID: 1101, Name: "Polygon zkEVM", Symbol: "ETH", Type: "evm", ChainID: 1101, RPCURLs: []string{"https://zkevm-rpc.com", "https://polygon-zkevm.publicnode.com"}, ExplorerURL: "https://zkevm.polygonscan.com", NativeCurrency: "ETH", Decimals: 18, Confirmations: 15},
		{ID: 324, Name: "zkSync Era", Symbol: "ETH", Type: "evm", ChainID: 324, RPCURLs: []string{"https://mainnet.era.zksync.io", "https://zksync-era.publicnode.com"}, ExplorerURL: "https://explorer.zksync.io", NativeCurrency: "ETH", Decimals: 18, Confirmations: 15},
		{ID: 59144, Name: "Linea", Symbol: "ETH", Type: "evm", ChainID: 59144, RPCURLs: []string{"https://rpc.linea.build", "https://linea.publicnode.com"}, ExplorerURL: "https://explorer.linea.build", NativeCurrency: "ETH", Decimals: 18, Confirmations: 15},
		{ID: 534352, Name: "Scroll", Symbol: "ETH", Type: "evm", ChainID: 534352, RPCURLs: []string{"https://rpc.scroll.io", "https://scroll.publicnode.com"}, ExplorerURL: "https://scrollscan.com", NativeCurrency: "ETH", Decimals: 18, Confirmations: 15},
		{ID: 5000, Name: "Mantle", Symbol: "MNT", Type: "evm", ChainID: 5000, RPCURLs: []string{"https://rpc.mantle.xyz", "https://mantle.publicnode.com"}, ExplorerURL: "https://explorer.mantle.xyz", NativeCurrency: "MNT", Decimals: 18, Confirmations: 15},
		{ID: 2000, Name: "Dogecoin", Symbol: "DOGE", Type: "dogecoin", ChainID: 2000, RPCURLs: []string{"https://dogecoin.treasure.lol"}, ExplorerURL: "https://dogechain.info", NativeCurrency: "DOGE", Decimals: 8, Confirmations: 10},
		{ID: 690, Name: "Redlight", Symbol: "REDLC", Type: "evm", ChainID: 690, RPCURLs: []string{"https://redlight-chain.com"}, ExplorerURL: "https://redlight.xyz/explorer", NativeCurrency: "REDLC", Decimals: 18, Confirmations: 15},
		{ID: 2662, Name: "Jaman", Symbol: "JAM", Type: "evm", ChainID: 2662, RPCURLs: []string{"https://rpc.jaman.io"}, ExplorerURL: "https://jamScan.io", NativeCurrency: "JAM", Decimals: 18, Confirmations: 15},
		{ID: 369, Name: "PulseChain", Symbol: "PLS", Type: "evm", ChainID: 369, RPCURLs: []string{"https://rpc.pulsechain.com", "https://pulsechain.publicnode.com"}, ExplorerURL: "https://scan.pulsechain.com", NativeCurrency: "PLS", Decimals: 18, Confirmations: 15},
		{ID: 2222, Name: "Kava", Symbol: "KAVA", Type: "evm", ChainID: 2222, RPCURLs: []string{"https://evm.kava.io", "https://kava.publicnode.com"}, ExplorerURL: "https://explorer.kava.io", NativeCurrency: "KAVA", Decimals: 18, Confirmations: 15},
		{ID: 106, Name: "Velas", Symbol: "VLX", Type: "evm", ChainID: 106, RPCURLs: []string{"https://evmexplorer.velas.com/rpc"}, ExplorerURL: "https://velas.io", NativeCurrency: "VLX", Decimals: 18, Confirmations: 15},
		{ID: 8217, Name: "Klaytn", Symbol: "KLAY", Type: "evm", ChainID: 8217, RPCURLs: []string{"https://klaytn.fandom.finance"}, ExplorerURL: "https://scope.klaytn.com", NativeCurrency: "KLAY", Decimals: 18, Confirmations: 15},
		{ID: 25, Name: "Cronos", Symbol: "CRO", Type: "evm", ChainID: 25, RPCURLs: []string{"https://evm.cronos.org", "https://cronos.org/explorer"}, ExplorerURL: "https://explorer.cronos.org", NativeCurrency: "CRO", Decimals: 18, Confirmations: 15},
		{ID: 199, Name: "BitTorrent", Symbol: "BTT", Type: "evm", ChainID: 199, RPCURLs: []string{"https://rpc.bittorrentchain.io"}, ExplorerURL: "https://bttcscan.com", NativeCurrency: "BTT", Decimals: 18, Confirmations: 15},
		{ID: 820, Name: "Callisto", Symbol: "CLO", Type: "evm", ChainID: 820, RPCURLs: []string{"https://rpc.callisto.network"}, ExplorerURL: "https://explorer.callisto.network", NativeCurrency: "CLO", Decimals: 18, Confirmations: 15},
		{ID: 11297108109, Name: "Palm", Symbol: "PALM", Type: "evm", ChainID: 11297108109, RPCURLs: []string{"https://palm-mainnet.infura.io/v3/"}, ExplorerURL: "https://explorer.palm.io", NativeCurrency: "PALM", Decimals: 18, Confirmations: 15},
		{ID: 8866, Name: "Zora", Symbol: "ETH", Type: "evm", ChainID: 8866, RPCURLs: []string{"https://rpc.zora.energy"}, ExplorerURL: "https://explorer.zora.energy", NativeCurrency: "ETH", Decimals: 18, Confirmations: 15},
		{ID: 7777777, Name: "Zora", Symbol: "ETH", Type: "evm", ChainID: 7777777, RPCURLs: []string{"https://rpc.zora.energy"}, ExplorerURL: "https://explorer.zora.energy", NativeCurrency: "ETH", Decimals: 18, Confirmations: 15},
	}

	for _, network := range networks {
		m.Networks[network.ChainID] = network
		m.NodePools[network.ChainID] = NewNodePool(network)
	}
}

// RegisterNetwork registers a new blockchain network
func (m *RPCManager) RegisterNetwork(network *BlockchainNetwork) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.Networks[network.ChainID]; exists {
		return fmt.Errorf("network with chain ID %d already exists", network.ChainID)
	}

	m.Networks[network.ChainID] = network
	m.NodePools[network.ChainID] = NewNodePool(network)
	return nil
}

// GetNetwork returns a network by chain ID
func (m *RPCManager) GetNetwork(chainID uint64) (*BlockchainNetwork, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	network, exists := m.Networks[chainID]
	if !exists {
		return nil, fmt.Errorf("network with chain ID %d not found", chainID)
	}
	return network, nil
}

// GetAllNetworks returns all registered networks
func (m *RPCManager) GetAllNetworks() []*BlockchainNetwork {
	m.mu.RLock()
	defer m.mu.RUnlock()

	networks := make([]*BlockchainNetwork, 0, len(m.Networks))
	for _, network := range m.Networks {
		networks = append(networks, network)
	}
	return networks
}

// GetNode returns an RPC node for a chain
func (m *RPCManager) GetNode(chainID uint64) (*RPCNode, error) {
	m.mu.RLock()
	pool, exists := m.NodePools[chainID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no node pool for chain ID %d", chainID)
	}

	node := pool.GetNode()
	if node == nil {
		return nil, fmt.Errorf("no available nodes for chain ID %d", chainID)
	}

	// Initialize client if needed
	if node.Client == nil {
		client, err := rpc.DialHTTP(node.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to RPC: %w", err)
		}
		node.Client = client
		node.EthClient = ethclient.NewClient(client)
	}

	return node, nil
}

// GetClient returns an ethclient for a chain
func (m *RPCManager) GetClient(chainID uint64) (*ethclient.Client, error) {
	node, err := m.GetNode(chainID)
	if err != nil {
		return nil, err
	}
	return node.EthClient, nil
}

// StartHealthCheck starts the health check routine
func (m *RPCManager) StartHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(m.HealthCheck.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkAllNodes(ctx)
		}
	}
}

// checkAllNodes checks health of all nodes
func (m *RPCManager) checkAllNodes(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pool := range m.NodePools {
		for _, node := range pool.Nodes {
			go m.checkNode(ctx, node)
		}
	}
}

// checkNode checks health of a single node
func (m *RPCManager) checkNode(ctx context.Context, node *RPCNode) {
	start := time.Now()
	err := node.Client.CallContext(ctx, nil, "eth_blockNumber")
	latency := time.Since(start)

	status := &HealthStatus{
		NodeURL:    node.URL,
		LastCheck:  time.Now(),
		Latency:    latency,
		IsHealthy:  err == nil,
	}

	node.mu.Lock()
	if err != nil {
		status.ErrorCount = node.ErrorCount + 1
		status.SuccessRate = node.SuccessRate
	} else {
		status.ErrorCount = node.ErrorCount
		status.SuccessRate = min(node.SuccessRate+0.1, 1.0)
	}
	node.mu.Unlock()

	m.HealthCheck.mu.Lock()
	m.HealthCheck.Nodes[node.URL] = status
	m.HealthCheck.mu.Unlock()

	if err != nil {
		pool, exists := m.NodePools[1] // Get any pool to mark unhealthy
		if exists {
			pool.MarkNodeUnhealthy(node, err)
		}
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func parseBlockNumber(result string) (uint64, error) {
	if len(result) > 2 && result[:2] == "0x" {
		n := new(big.Int)
		n.SetString(result[2:], 16)
		return n.Uint64(), nil
	}
	return 0, fmt.Errorf("invalid block number format")
}

func parseBigInt(result string) (*big.Int, error) {
	if len(result) > 2 && result[:2] == "0x" {
		n := new(big.Int)
		n.SetString(result[2:], 16)
		return n, nil
	}
	return nil, fmt.Errorf("invalid big int format")
}

func parseNonce(result string) (uint64, error) {
	n, err := parseBigInt(result)
	if err != nil {
		return 0, err
	}
	return n.Uint64(), nil
}

func parseUint64(result string) (uint64, error) {
	n, err := parseBigInt(result)
	if err != nil {
		return 0, err
	}
	return n.Uint64(), nil
}

func parseTransaction(result map[string]interface{}) (*Transaction, error) {
	tx := &Transaction{}
	if v, ok := result["hash"].(string); ok {
		tx.Hash = v
	}
	if v, ok := result["from"].(string); ok {
		tx.From = common.HexToAddress(v)
	}
	if v, ok := result["to"].(string); ok {
		tx.To = common.HexToAddress(v)
	}
	if v, ok := result["value"].(string); ok {
		tx.Value = v
	}
	if v, ok := result["gas"].(string); ok {
		tx.Gas = v
	}
	if v, ok := result["gasPrice"].(string); ok {
		tx.GasPrice = v
	}
	if v, ok := result["nonce"].(string); ok {
		n, _ := parseNonce(v)
		tx.Nonce = n
	}
	return tx, nil
}

func parseReceipt(result map[string]interface{}) (*TransactionReceipt, error) {
	receipt := &TransactionReceipt{}
	if v, ok := result["transactionHash"].(string); ok {
		receipt.TransactionHash = v
	}
	if v, ok := result["blockNumber"].(string); ok {
		n, _ := parseUint64(v)
		receipt.BlockNumber = n
	}
	if v, ok := result["status"].(string); ok {
		receipt.Status = v == "0x1"
	}
	if v, ok := result["gasUsed"].(string); ok {
		receipt.GasUsed = v
	}
	return receipt, nil
}

func parseBlock(result map[string]interface{}) (*Block, error) {
	block := &Block{}
	if v, ok := result["number"].(string); ok {
		n, _ := parseUint64(v)
		block.Number = n
	}
	if v, ok := result["hash"].(string); ok {
		block.Hash = v
	}
	if v, ok := result["parentHash"].(string); ok {
		block.ParentHash = v
	}
	if v, ok := result["timestamp"].(string); ok {
		n, _ := parseUint64(v)
		block.Timestamp = n
	}
	return block, nil
}

func parseLogs(result []map[string]interface{}) ([]Log, error) {
	logs := make([]Log, len(result))
	for i, r := range result {
		logs[i] = parseLog(r)
	}
	return logs, nil
}

func parseLog(result map[string]interface{}) Log {
	log := Log{}
	if v, ok := result["address"].(string); ok {
		log.Address = common.HexToAddress(v)
	}
	if v, ok := result["topics"].(string); ok {
		log.Topics = append(log.Topics, common.HexToHash(v))
	}
	if v, ok := result["data"].(string); ok {
		log.Data = common.FromHex(v)
	}
	return log
}

func toCallMsg(msg ethereum.CallMsg) map[string]interface{} {
	result := make(map[string]interface{})
	if msg.From != (common.Address{}) {
		result["from"] = msg.From.String()
	}
	if msg.To != (common.Address{}) {
		result["to"] = msg.To.String()
	}
	if msg.Value != nil {
		result["value"] = fmt.Sprintf("0x%x", msg.Value)
	}
	if len(msg.Data) > 0 {
		result["data"] = common.ToHex(msg.Data)
	}
	if msg.Gas > 0 {
		result["gas"] = fmt.Sprintf("0x%x", msg.Gas)
	}
	if msg.GasPrice != nil {
		result["gasPrice"] = fmt.Sprintf("0x%x", msg.GasPrice)
	}
	return result
}

func toFilterQuery(filter ethereum.FilterQuery) map[string]interface{} {
	result := make(map[string]interface{})
	if len(filter.Addresses) > 0 {
		addresses := make([]string, len(filter.Addresses))
		for i, addr := range filter.Addresses {
			addresses[i] = addr.String()
		}
		result["address"] = addresses
	}
	if filter.FromBlock != nil {
		result["fromBlock"] = fmt.Sprintf("0x%x", filter.FromBlock)
	}
	if filter.ToBlock != nil {
		result["toBlock"] = fmt.Sprintf("0x%x", filter.ToBlock)
	}
	if len(filter.Topics) > 0 {
		result["topics"] = filter.Topics
	}
	return result
}

// ============================================================================
// Data Structures
// ============================================================================

type Transaction struct {
	Hash        common.Hash    `json:"hash"`
	From        common.Address `json:"from"`
	To          common.Address `json:"to"`
	Value       string         `json:"value"`
	Data        string         `json:"data"`
	Gas         string         `json:"gas"`
	GasPrice    string         `json:"gasPrice"`
	Nonce       uint64         `json:"nonce"`
	ChainID     uint64         `json:"chainId"`
	V           string         `json:"v"`
	R           string         `json:"r"`
	S           string         `json:"s"`
}

type TransactionReceipt struct {
	TransactionHash common.Hash `json:"transactionHash"`
	BlockNumber     uint64      `json:"blockNumber"`
	BlockHash       common.Hash `json:"blockHash"`
	Status          bool        `json:"status"`
	GasUsed         string      `json:"gasUsed"`
	CumulativeGasUsed string    `json:"cumulativeGasUsed"`
	Logs            []Log       `json:"logs"`
}

type Block struct {
	Number       uint64   `json:"number"`
	Hash         string   `json:"hash"`
	ParentHash   string   `json:"parentHash"`
	Timestamp    uint64   `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
}

type Header struct {
	Number     uint64   `json:"number"`
	Hash       string   `json:"hash"`
	ParentHash string   `json:"parentHash"`
	Timestamp  uint64   `json:"timestamp"`
	MixHash    string   `json:"mixHash"`
	Nonce      string   `json:"nonce"`
}

type Log struct {
	Address     common.Address `json:"address"`
	Topics      []common.Hash  `json:"topics"`
	Data        []byte         `json:"data"`
	BlockNumber uint64          `json:"blockNumber"`
	TxHash      common.Hash    `json:"transactionHash"`
	LogIndex    uint64         `json:"logIndex"`
}

// ============================================================================
// Utility
// ============================================================================

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// ToJSON converts a value to JSON
func ToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
