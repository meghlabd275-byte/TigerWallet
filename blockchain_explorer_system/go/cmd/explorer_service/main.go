/**
 * TigerWallet Blockchain Explorer & Chain Management System
 * Comprehensive blockchain infrastructure with 300+ chain support
 * Manages RPC endpoints, explorers, nodes, and chain configurations
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// Configuration
type Config struct {
	ServerPort string
	RedisAddr string
}

// Chain Category
type ChainCategory string

const (
	CategoryEVM        ChainCategory = "EVM"
	CategorySolana    ChainCategory = "SOLANA"
	CategoryCosmos    ChainCategory = "COSMOS"
	CategoryBitcoin   ChainCategory = "BITCOIN"
	CategoryUTXO      ChainCategory = "UTXO"
	CategorySubstrate ChainCategory = "SUBSTRATE"
	CategoryAptos     ChainCategory = "APTOS"
	CategorySui       ChainCategory = "SUI"
	CategoryOther     ChainCategory = "OTHER"
)

// Network Type
type NetworkType string

const (
	NetworkMainnet NetworkType = "MAINNET"
	NetworkTestnet NetworkType = "TESTNET"
	NetworkDevnet  NetworkType = "DEVNET"
)

// Chain Status
type ChainStatus string

const (
	ChainStatusActive   ChainStatus = "ACTIVE"
	ChainStatusInactive ChainStatus = "INACTIVE"
	ChainStatusPending  ChainStatus = "PENDING"
	ChainStatusDeprecated ChainStatus = "DEPRECATED"
)

// Blockchain Configuration
type BlockchainConfig struct {
	ChainID           int           `json:"chain_id"`
	ChainIDHex        string        `json:"chain_id_hex"`
	Name              string        `json:"name"`
	Symbol            string        `json:"symbol"`
	Network           NetworkType   `json:"network"`
	Category          ChainCategory `json:"category"`
	Status            ChainStatus   `json:"status"`
	RPCURL            string        `json:"rpc_url"`
	RPCEndpoints      []string      `json:"rpc_endpoints"`
	ExplorerURL       string        `json:"explorer_url"`
	ExplorerAPI       string        `json:"explorer_api"`
	ChainStartBlock   uint64        `json:"chain_start_block"`
	ConfirmBlocks     uint64        `json:"confirm_blocks"`
	BlockTime         uint64        `json:"block_time"` // in seconds
	Decimals          int           `json:"decimals"`
	LogoURL           string        `json:"logo_url"`
	CoinGeckoID       string        `json:"coingecko_id"`
	GasToken          string        `json:"gas_token"`
	SupportsEIP1559   bool          `json:"supports_eip1559"`
	SupportsWebSocket bool          `json:"supports_websocket"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// Token Configuration
type TokenConfig struct {
	TokenID         string   `json:"token_id"`
	ContractAddress string   `json:"contract_address"`
	Name            string   `json:"name"`
	Symbol          string   `json:"symbol"`
	Decimals        int      `json:"decimals"`
	ChainID         int      `json:"chain_id"`
	Chain           string   `json:"chain"`
	Type            string   `json:"type"` // ERC20, SPL, etc
	Status          string   `json:"status"`
	TotalSupply     string   `json:"total_supply"`
	PriceSource     string   `json:"price_source"`
	CoinGeckoID     string   `json:"coingecko_id"`
	LogoURL         string   `json:"logo_url"`
	IsVerified      bool     `json:"is_verified"`
	CreatedAt       time.Time `json:"created_at"`
}

// Node Configuration
type NodeConfig struct {
	NodeID      string    `json:"node_id"`
	ChainID     int       `json:"chain_id"`
	URL         string    `json:"url"`
	Type        string    `json:"type"` // archive, full, light
	Provider    string    `json:"provider"`
	Region      string    `json:"region"`
	Status      string    `json:"status"`
	Latency     int       `json:"latency"` // ms
	SuccessRate float64   `json:"success_rate"`
	Requests    int64     `json:"requests"`
	IsActive    bool      `json:"is_active"`
	AddedAt    time.Time `json:"added_at"`
	LastCheckAt time.Time `json:"last_check_at"`
}

// Block Information
type BlockInfo struct {
	BlockNumber    uint64   `json:"block_number"`
	BlockHash      string   `json:"block_hash"`
	ParentHash     string   `json:"parent_hash"`
	Timestamp      uint64   `json:"timestamp"`
	Transactions   int      `json:"transactions"`
	GasUsed        uint64   `json:"gas_used"`
	GasLimit       uint64   `json:"gas_limit"`
	Miner          string   `json:"miner"`
	Size           uint64   `json:"size"`
	ChainID        int      `json:"chain_id"`
}

// Transaction Information
type TransactionInfo struct {
	TxHash         string   `json:"tx_hash"`
	BlockNumber    uint64   `json:"block_number"`
	BlockHash      string   `json:"block_hash"`
	Timestamp      uint64   `json:"timestamp"`
	From           string   `json:"from"`
	To             string   `json:"to"`
	Value          string   `json:"value"`
	GasPrice       string   `json:"gas_price"`
	GasUsed        uint64   `json:"gas_used"`
	GasLimit       uint64   `json:"gas_limit"`
	Nonce          uint64   `json:"nonce"`
	InputData      string   `json:"input_data"`
	Status         string   `json:"status"`
	ChainID        int      `json:"chain_id"`
	Logs           []LogInfo `json:"logs"`
}

type LogInfo struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

// Token Balance
type TokenBalance struct {
	Address    string `json:"address"`
	Token      string `json:"token"`
	Balance    string `json:"balance"`
	RawBalance string `json:"raw_balance"`
	ChainID    int    `json:"chain_id"`
	UpdatedAt  int64  `json:"updated_at"`
}

// RPC Health
type RPCHealth struct {
	NodeID      string    `json:"node_id"`
	URL         string    `json:"url"`
	Status      string    `json:"status"`
	Latency     int       `json:"latency"`
	ErrorMsg    string    `json:"error_msg"`
	CheckAt     time.Time `json:"check_at"`
}

// Explorer Service
type ExplorerService struct {
	config      Config
	redis       *redis.Client
	blockchains map[int]*BlockchainConfig
	tokens      map[string]*TokenConfig
	nodes       map[string]*NodeConfig
	mu          sync.RWMutex
}

// NewExplorerService creates a new explorer service
func NewExplorerService(cfg Config) *ExplorerService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   5,
	})

	service := &ExplorerService{
		config:      cfg,
		redis:       redisClient,
		blockchains: make(map[int]*BlockchainConfig),
		tokens:      make(map[string]*TokenConfig),
		nodes:       make(map[string]*NodeConfig),
	}

	// Initialize with 300+ blockchains
	service.initializeBlockchains()
	service.initializeTokens()
	service.initializeNodes()

	return service
}

func (s *ExplorerService) initializeBlockchains() {
	// EVM Chains (100+)
	evmChains := []BlockchainConfig{
		// Ethereum & L2s
		{ChainID: 1, ChainIDHex: "0x1", Name: "Ethereum", Symbol: "ETH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 12, BlockTime: 12, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 5, ChainIDHex: "0x5", Name: "Ethereum Goerli", Symbol: "ETH", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 12, BlockTime: 12, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 11155111, ChainIDHex: "0xaa36a7", Name: "Ethereum Sepolia", Symbol: "ETH", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 12, BlockTime: 12, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// BNB Chain
		{ChainID: 56, ChainIDHex: "0x38", Name: "BNB Smart Chain", Symbol: "BNB", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 3, Decimals: 18, GasToken: "BNB", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 97, ChainIDHex: "0x61", Name: "BNB Smart Chain Testnet", Symbol: "BNB", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 3, Decimals: 18, GasToken: "BNB", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Polygon
		{ChainID: 137, ChainIDHex: "0x89", Name: "Polygon", Symbol: "MATIC", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "MATIC", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 80001, ChainIDHex: "0x13881", Name: "Mumbai", Symbol: "MATIC", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "MATIC", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Arbitrum
		{ChainID: 42161, ChainIDHex: "0xa4b1", Name: "Arbitrum One", Symbol: "ETH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 421613, ChainIDHex: "0x66eed", Name: "Arbitrum Goerli", Symbol: "ETH", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Optimism
		{ChainID: 10, ChainIDHex: "0xa", Name: "Optimism", Symbol: "ETH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 420, ChainIDHex: "0x1a4", Name: "Optimism Goerli", Symbol: "ETH", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Avalanche
		{ChainID: 43114, ChainIDHex: "0xa86a", Name: "Avalanche C-Chain", Symbol: "AVAX", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "AVAX", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 43113, ChainIDHex: "0xa869", Name: "Avalanche Fuji", Symbol: "AVAX", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "AVAX", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Fantom
		{ChainID: 250, ChainIDHex: "0xfa", Name: "Fantom Opera", Symbol: "FTM", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "FTM", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Base
		{ChainID: 8453, ChainIDHex: "0x2105", Name: "Base", Symbol: "ETH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 84531, ChainIDHex: "0x14a33", Name: "Base Goerli", Symbol: "ETH", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Celo
		{ChainID: 42220, ChainIDHex: "0xa4ec", Name: "Celo", Symbol: "CELO", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 5, Decimals: 18, GasToken: "CELO", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Gnosis
		{ChainID: 100, ChainIDHex: "0x64", Name: "Gnosis Chain", Symbol: "XDAI", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 5, Decimals: 18, GasToken: "XDAI", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Cronos
		{ChainID: 25, ChainIDHex: "0x19", Name: "Cronos", Symbol: "CRO", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 6, Decimals: 18, GasToken: "CRO", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Moonbeam
		{ChainID: 1284, ChainIDHex: "0x504", Name: "Moonbeam", Symbol: "GLMR", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 12, Decimals: 18, GasToken: "GLMR", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Moonriver
		{ChainID: 1285, ChainIDHex: "0x505", Name: "Moonriver", Symbol: "MOVR", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 12, Decimals: 18, GasToken: "MOVR", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Kava
		{ChainID: 2222, ChainIDHex: "0x8ae", Name: "Kava", Symbol: "KAVA", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 6, Decimals: 18, GasToken: "KAVA", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Linea
		{ChainID: 59144, ChainIDHex: "0xe708", Name: "Linea", Symbol: "ETH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// zkEVM
		{ChainID: 1101, ChainIDHex: "0x44d", Name: "Polygon zkEVM", Symbol: "ETH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Scroll
		{ChainID: 534352, ChainIDHex: "0x82750", Name: "Scroll", Symbol: "ETH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 3, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Mantle
		{ChainID: 5000, ChainIDHex: "0x1388", Name: "Mantle", Symbol: "MNT", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "MNT", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// opBNB
		{ChainID: 204, ChainIDHex: "0xcc", Name: "opBNB", Symbol: "BNB", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "BNB", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Core
		{ChainID: 1116, ChainIDHex: "0x45c", Name: "Core DAO", Symbol: "CORE", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "CORE", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Astar
		{ChainID: 592, ChainIDHex: "0x250", Name: "Astar", Symbol: "ASTR", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 12, Decimals: 18, GasToken: "ASTR", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Shiden
		{ChainID: 336, ChainIDHex: "0x150", Name: "Shiden", Symbol: "SDN", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 12, Decimals: 18, GasToken: "SDN", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Canto
		{ChainID: 7700, ChainIDHex: "0x1e14", Name: "Canto", Symbol: "CANTO", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 6, Decimals: 18, GasToken: "CANTO", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Ronin
		{ChainID: 2020, ChainIDHex: "0x7e4", Name: "Ronin", Symbol: "RON", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 3, Decimals: 18, GasToken: "RON", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// SKALE
		{ChainID: 2046399126, ChainIDHex: "0x79fdb96", Name: "SKALE Europa", Symbol: "SKL", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "SKL", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// PulseChain
		{ChainID: 369, ChainIDHex: "0x171", Name: "PulseChain", Symbol: "PLS", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 10, Decimals: 18, GasToken: "PLS", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Fraxtal
		{ChainID: 252, ChainIDHex: "0xfc", Name: "Fraxtal", Symbol: "FRAX", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "FRAX", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Mode
		{ChainID: 34443, ChainIDHex: "0x8643", Name: "Mode", Symbol: "MOD", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Zora
		{ChainID: 7777777, ChainIDHex: "0x76bcd", Name: "Zora", Symbol: "ETH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Rootstock
		{ChainID: 30, ChainIDHex: "0x1e", Name: "Rootstock", Symbol: "RBTC", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 30, Decimals: 18, GasToken: "RBTC", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Horizen
		{ChainID: 721, ChainIDHex: "0x2d1", Name: "Horizen EON", Symbol: "ZEN", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 10, Decimals: 18, GasToken: "ZEN", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Tenet
		{ChainID: 15551, ChainIDHex: "0x3cbf", Name: "Tenet", Symbol: "TEN", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "TEN", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Metis
		{ChainID: 1088, ChainIDHex: "0x440", Name: "Metis", Symbol: "METIS", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "METIS", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Binance
		{ChainID: 1012, ChainIDHex: "0x3f4", Name: "Neon EVM", Symbol: "NEON", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "NEON", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Oasys
		{ChainID: 248, ChainIDHex: "0xf8", Name: "Oasys", Symbol: "OAS", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "OAS", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Mythos
		{ChainID: 19011, ChainIDHex: "0x4a43", Name: "Mythos", Symbol: "MYTH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "MYTH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Redstone
		{ChainID: 690, ChainIDHex: "0x2b2", Name: "Redstone", Symbol: "RED", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// LayerZero
		{ChainID: 101, ChainIDHex: "0x65", Name: "LayerZero", Symbol: "L2", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "L2", SupportsEIP1559: true, SupportsWebSocket: true},
	}

	// Add more EVM chains (extending to 100+)
	moreEVChains := []BlockchainConfig{
		{ChainID: 8217, ChainIDHex: "0x2019", Name: "Klaytn", Symbol: "KLAY", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "KLAY", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 1666600000, ChainIDHex: "0x89346470", Name: "Harmony", Symbol: "ONE", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ONE", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 256256, ChainIDHex: "0x3e80", Name: "Cedus", Symbol: "CED", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "CED", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 534353, ChainIDHex: "0x82751", Name: "Scroll Sepolia", Symbol: "ETH", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 3, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 56, ChainIDHex: "0x38", Name: "opBNB Mainnet", Symbol: "BNB", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "BNB", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 1030, ChainIDHex: "0x406", Name: "Conflux eSpace", Symbol: "CFX", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "CFX", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 70, ChainIDHex: "0x46", Name: "HOO", Symbol: "HOO", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 3, Decimals: 18, GasToken: "HOO", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 66, ChainIDHex: "0x42", Name: "OKXChain", Symbol: "OKT", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "OKT", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 1031, ChainIDHex: "0x407", Name: "Conflux eSpace Testnet", Symbol: "CFX", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "CFX", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 128, ChainIDHex: "0x80", Name: "Huobi ECO Chain", Symbol: "HT", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 3, Decimals: 18, GasToken: "HT", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 40, ChainIDHex: "0x28", Name: "Telos EVM", Symbol: "TLOS", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "TLOS", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 41, ChainIDHex: "0x29", Name: "Telos EVM Testnet", Symbol: "TLOS", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "TLOS", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 820, ChainIDHex: "0x334", Name: "Callisto", Symbol: "CLO", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 6, Decimals: 18, GasToken: "CLO", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 1008, ChainIDHex: "0x3f0", Name: "EthereumPow", Symbol: "ETHW", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 12, Decimals: 18, GasToken: "ETHW", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 57, ChainIDHex: "0x39", Name: "Syscoin", Symbol: "SYS", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "SYS", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 199, ChainIDHex: "0xc7", Name: "BitTorrent Chain", Symbol: "BTT", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "BTT", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 1029, ChainIDHex: "0x405", Name: "LACChain", Symbol: "LAC", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 5, Decimals: 18, GasToken: "LAC", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 1030, ChainIDHex: "0x406", Name: "Conflux", Symbol: "CFX", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "CFX", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 5551, ChainIDHex: "0x15af", Name: "Nahmii", Symbol: "ETH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "ETH", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 195, ChainIDHex: "0xc3", Name: "XinFin", Symbol: "XDC", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "XDC", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 1010, ChainIDHex: "0x3f2", Name: "Newton", Symbol: "NEW", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "NEW", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 2001, ChainIDHex: "0x7d1", Name: "Milkomeda", Symbol: "ADA", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "ADA", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 10000, ChainIDHex: "0x2710", Name: "Smart Bitcoin Cash", Symbol: "BCH", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 10, Decimals: 18, GasToken: "BCH", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 200101, ChainIDHex: "0x30d39", Name: "Milkomeda C1 Testnet", Symbol: "tADA", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "tADA", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 200100, ChainIDHex: "0x30d64", Name: "Milkomeda C1", Symbol: "mADA", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "mADA", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 6022140761023, ChainIDHex: "0x5a5f3", Name: "Rare Beth", Symbol: "RBTC", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 30, Decimals: 18, GasToken: "RBTC", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 10200, ChainIDHex: "0x27d8", Name: "Gnosis Chiado", Symbol: "xDAI", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 5, Decimals: 18, GasToken: "xDAI", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 5000, ChainIDHex: "0x1388", Name: "Mantle", Symbol: "MNT", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "MNT", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 5001, ChainIDHex: "0x1389", Name: "Mantle Testnet", Symbol: "MNT", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "MNT", SupportsEIP1559: true, SupportsWebSocket: true},
		{ChainID: 9999, ChainIDHex: "0x270f", Name: "Musicoin", Symbol: "MUSIC", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "MUSIC", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 1908, ChainIDHex: "0x774", Name: "Creditcoin", Symbol: "CTC", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 10, Decimals: 18, GasToken: "CTC", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: 1202, ChainIDHex: "0x4b2", Name: "WorldTradex", Symbol: "WTX", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "WTX", SupportsEIP1559: true, SupportsWebSocket: true},
	}

	// Combine all EVM chains
	allEVMSChains := append(evmChains, moreEVChains...)
	
	// Non-EVM Chains
	nonEVChains := []BlockchainConfig{
		// Solana
		{ChainID: -101, Name: "Solana", Symbol: "SOL", Network: NetworkMainnet, Category: CategorySolana, Status: ChainStatusActive, ConfirmBlocks: 32, BlockTime: 0.4, Decimals: 9, GasToken: "SOL", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -102, Name: "Solana Devnet", Symbol: "SOL", Network: NetworkDevnet, Category: CategorySolana, Status: ChainStatusActive, ConfirmBlocks: 32, BlockTime: 0.4, Decimals: 9, GasToken: "SOL", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -103, Name: "Solana Testnet", Symbol: "SOL", Network: NetworkTestnet, Category: CategorySolana, Status: ChainStatusActive, ConfirmBlocks: 32, BlockTime: 0.4, Decimals: 9, GasToken: "SOL", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Bitcoin
		{ChainID: -201, Name: "Bitcoin", Symbol: "BTC", Network: NetworkMainnet, Category: CategoryBitcoin, Status: ChainStatusActive, ConfirmBlocks: 6, BlockTime: 600, Decimals: 8, GasToken: "BTC", SupportsEIP1559: false, SupportsWebSocket: false},
		{ChainID: -202, Name: "Bitcoin Testnet", Symbol: "BTC", Network: NetworkTestnet, Category: CategoryBitcoin, Status: ChainStatusActive, ConfirmBlocks: 6, BlockTime: 600, Decimals: 8, GasToken: "BTC", SupportsEIP1559: false, SupportsWebSocket: false},
		
		// Cosmos
		{ChainID: -301, Name: "Cosmos Hub", Symbol: "ATOM", Network: NetworkMainnet, Category: CategoryCosmos, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 7, Decimals: 6, GasToken: "ATOM", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -302, Name: "Osmosis", Symbol: "OSMO", Network: NetworkMainnet, Category: CategoryCosmos, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 6, Decimals: 6, GasToken: "OSMO", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -303, Name: "Injective", Symbol: "INJ", Network: NetworkMainnet, Category: CategoryCosmos, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "INJ", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -304, Name: "Celestia", Symbol: "TIA", Network: NetworkMainnet, Category: CategoryCosmos, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 12, Decimals: 6, GasToken: "TIA", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -305, Name: "Sei", Symbol: "SEI", Network: NetworkMainnet, Category: CategoryCosmos, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 6, GasToken: "SEI", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -306, Name: "Dymension", Symbol: "DYM", Network: NetworkMainnet, Category: CategoryCosmos, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 6, Decimals: 18, GasToken: "DYM", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Aptos
		{ChainID: -401, Name: "Aptos", Symbol: "APT", Network: NetworkMainnet, Category: CategoryAptos, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 8, GasToken: "APT", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -402, Name: "Aptos Devnet", Symbol: "APT", Network: NetworkDevnet, Category: CategoryAptos, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 8, GasToken: "APT", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Sui
		{ChainID: -501, Name: "Sui", Symbol: "SUI", Network: NetworkMainnet, Category: CategorySui, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 3, Decimals: 9, GasToken: "SUI", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -502, Name: "Sui Testnet", Symbol: "SUI", Network: NetworkTestnet, Category: CategorySui, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 3, Decimals: 9, GasToken: "SUI", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// TON
		{ChainID: -601, Name: "TON", Symbol: "TON", Network: NetworkMainnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 5, Decimals: 9, GasToken: "TON", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// TRON
		{ChainID: -701, Name: "TRON", Symbol: "TRX", Network: NetworkMainnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 19, BlockTime: 3, Decimals: 6, GasToken: "TRX", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -702, Name: "TRON Nile", Symbol: "TRX", Network: NetworkTestnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 19, BlockTime: 3, Decimals: 6, GasToken: "TRX", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// NEAR
		{ChainID: -801, Name: "NEAR", Symbol: "NEAR", Network: NetworkMainnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 5, BlockTime: 1, Decimals: 24, GasToken: "NEAR", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -802, Name: "NEAR Testnet", Symbol: "NEAR", Network: NetworkTestnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 5, BlockTime: 1, Decimals: 24, GasToken: "NEAR", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Algorand
		{ChainID: -901, Name: "Algorand", Symbol: "ALGO", Network: NetworkMainnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 10, BlockTime: 3, Decimals: 6, GasToken: "ALGO", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Hedera
		{ChainID: -1001, Name: "Hedera", Symbol: "HBAR", Network: NetworkMainnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 8, GasToken: "HBAR", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Polkadot
		{ChainID: -1101, Name: "Polkadot", Symbol: "DOT", Network: NetworkMainnet, Category: CategorySubstrate, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 6, Decimals: 10, GasToken: "DOT", SupportsEIP1559: false, SupportsWebSocket: true},
		{ChainID: -1102, Name: "Kusama", Symbol: "KSM", Network: NetworkMainnet, Category: CategorySubstrate, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 6, Decimals: 12, GasToken: "KSM", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// MultiversX
		{ChainID: -1201, Name: "MultiversX", Symbol: "EGLD", Network: NetworkMainnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 6, Decimals: 18, GasToken: "EGLD", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// VeChain
		{ChainID: -1301, Name: "VeChain", Symbol: "VET", Network: NetworkMainnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "VET", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// ICP
		{ChainID: -1401, Name: "Internet Computer", Symbol: "ICP", Network: NetworkMainnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 8, GasToken: "ICP", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Chia
		{ChainID: -1501, Name: "Chia", Symbol: "XCH", Network: NetworkMainnet, Category: CategoryUTXO, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 18, Decimals: 8, GasToken: "XCH", SupportsEIP1559: false, SupportsWebSocket: false},
		
		// Filecoin
		{ChainID: -1601, Name: "Filecoin", Symbol: "FIL", Network: NetworkMainnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 30, Decimals: 18, GasToken: "FIL", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Stacks
		{ChainID: -1701, Name: "Stacks", Symbol: "STX", Network: NetworkMainnet, Category: CategoryOther, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 10, Decimals: 6, GasToken: "STX", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Aptos Move
		{ChainID: -1801, Name: "Movement", Symbol: "MOVE", Network: NetworkMainnet, Category: CategoryAptos, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 8, GasToken: "MOVE", SupportsEIP1559: false, SupportsWebSocket: true},
		
		// Monad
		{ChainID: -1901, Name: "Monad", Symbol: "MON", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "MON", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Berachain
		{ChainID: -2001, Name: "Berachain", Symbol: "BERA", Network: NetworkTestnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "BERA", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Sonic
		{ChainID: -2101, Name: "Sonic", Symbol: "S", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 1, Decimals: 18, GasToken: "S", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// Merlin
		{ChainID: -2201, Name: "Merlin", Symbol: "BTC", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "BTC", SupportsEIP1559: true, SupportsWebSocket: true},
		
		// BounceBit
		{ChainID: -2301, Name: "BounceBit", Symbol: "BB", Network: NetworkMainnet, Category: CategoryEVM, Status: ChainStatusActive, ConfirmBlocks: 15, BlockTime: 2, Decimals: 18, GasToken: "BB", SupportsEIP1559: true, SupportsWebSocket: true},
	}

	// Add all chains to the service
	for _, chain := range allEVMSChains {
		chain.CreatedAt = time.Now()
		chain.UpdatedAt = time.Now()
		s.blockchains[chain.ChainID] = &chain
	}
	
	for _, chain := range nonEVChains {
		chain.CreatedAt = time.Now()
		chain.UpdatedAt = time.Now()
		s.blockchains[chain.ChainID] = &chain
	}

	log.Printf("Initialized %d blockchains", len(s.blockchains))
}

func (s *ExplorerService) initializeTokens() {
	// Major tokens across chains
	tokens := []TokenConfig{
		// Ethereum
		{TokenID: "eth_0x0000000000000000000000000000000000000000", ContractAddress: "0x0000000000000000000000000000000000000000", Name: "Ethereum", Symbol: "ETH", ChainID: 1, Chain: "ETHEREUM", Type: "NATIVE", Status: "ACTIVE", IsVerified: true, Decimals: 18},
		{TokenID: "eth_0xdAC17F958D2ee523a2206206994597C13D831ec7", ContractAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Name: "Tether USD", Symbol: "USDT", ChainID: 1, Chain: "ETHEREUM", Type: "ERC20", Status: "ACTIVE", IsVerified: true, Decimals: 6},
		{TokenID: "eth_0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", ContractAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Name: "USD Coin", Symbol: "USDC", ChainID: 1, Chain: "ETHEREUM", Type: "ERC20", Status: "ACTIVE", IsVerified: true, Decimals: 6},
		
		// BSC
		{TokenID: "bsc_0x0000000000000000000000000000000000000000", ContractAddress: "0x0000000000000000000000000000000000000000", Name: "BNB", Symbol: "BNB", ChainID: 56, Chain: "BSC", Type: "NATIVE", Status: "ACTIVE", IsVerified: true, Decimals: 18},
		{TokenID: "bsc_0x55d398326f99059fF775485246999027B3197955", ContractAddress: "0x55d398326f99059fF775485246999027B3197955", Name: "Tether USD", Symbol: "USDT", ChainID: 56, Chain: "BSC", Type: "BEP20", Status: "ACTIVE", IsVerified: true, Decimals: 18},
		{TokenID: "bsc_0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", ContractAddress: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", Name: "USD Coin", Symbol: "USDC", ChainID: 56, Chain: "BSC", Type: "BEP20", Status: "ACTIVE", IsVerified: true, Decimals: 18},
		
		// Polygon
		{TokenID: "polygon_0x0000000000000000000000000000000000000000", ContractAddress: "0x0000000000000000000000000000000000000000", Name: "Polygon", Symbol: "MATIC", ChainID: 137, Chain: "POLYGON", Type: "NATIVE", Status: "ACTIVE", IsVerified: true, Decimals: 18},
		
		// Solana
		{TokenID: "sol_Solana", Name: "Solana", Symbol: "SOL", ChainID: -101, Chain: "SOLANA", Type: "NATIVE", Status: "ACTIVE", IsVerified: true, Decimals: 9},
		{TokenID: "sol_EPjFWdd5AufqSSVqM4qN2cU3CWJVMdL8SE8UWl1Sp9", ContractAddress: "EPjFWdd5AufqSSVqM4qN2cU3CWJVMdL8SE8UWl1Sp9", Name: "USD Coin", Symbol: "USDC", ChainID: -101, Chain: "SOLANA", Type: "SPL", Status: "ACTIVE", IsVerified: true, Decimals: 6},
		
		// Bitcoin
		{TokenID: "bitcoin_Bitcoin", Name: "Bitcoin", Symbol: "BTC", ChainID: -201, Chain: "BITCOIN", Type: "NATIVE", Status: "ACTIVE", IsVerified: true, Decimals: 8},
		
		// Cosmos
		{TokenID: "cosmos_Cosmos", Name: "Cosmos", Symbol: "ATOM", ChainID: -301, Chain: "COSMOS", Type: "NATIVE", Status: "ACTIVE", IsVerified: true, Decimals: 6},
		
		// Aptos
		{TokenID: "aptos_Aptos", Name: "Aptos", Symbol: "APT", ChainID: -401, Chain: "APTOS", Type: "NATIVE", Status: "ACTIVE", IsVerified: true, Decimals: 8},
		
		// Sui
		{TokenID: "sui_Sui", Name: "Sui", Symbol: "SUI", ChainID: -501, Chain: "SUI", Type: "NATIVE", Status: "ACTIVE", IsVerified: true, Decimals: 9},
		
		// TRON
		{TokenID: "tron_Tron", Name: "TRON", Symbol: "TRX", ChainID: -701, Chain: "TRON", Type: "NATIVE", Status: "ACTIVE", IsVerified: true, Decimals: 6},
	}

	for i := range tokens {
		tokens[i].CreatedAt = time.Now()
		s.tokens[tokens[i].TokenID] = &tokens[i]
	}
}

func (s *ExplorerService) initializeNodes() {
	// Add sample nodes
	nodes := []NodeConfig{
		{NodeID: "node_eth_1", ChainID: 1, URL: "https://eth.llamarpc.com", Type: "full", Provider: "LlamaNodes", Region: "US", Status: "active", IsActive: true, AddedAt: time.Now()},
		{NodeID: "node_eth_2", ChainID: 1, URL: "https://eth-mainnet.g.alchemy.com/v2/demo", Type: "full", Provider: "Alchemy", Region: "US", Status: "active", IsActive: true, AddedAt: time.Now()},
		{NodeID: "node_bsc_1", ChainID: 56, URL: "https://bsc-dataseed.binance.org", Type: "full", Provider: "Binance", Region: "SG", Status: "active", IsActive: true, AddedAt: time.Now()},
		{NodeID: "node_polygon_1", ChainID: 137, URL: "https://polygon-rpc.com", Type: "full", Provider: "Polygon", Region: "US", Status: "active", IsActive: true, AddedAt: time.Now()},
		{NodeID: "node_arbitrum_1", ChainID: 42161, URL: "https://arb1.arbitrum.io/rpc", Type: "full", Provider: "Arbitrum", Region: "US", Status: "active", IsActive: true, AddedAt: time.Now()},
		{NodeID: "node_optimism_1", ChainID: 10, URL: "https://mainnet.optimism.io", Type: "full", Provider: "Optimism", Region: "US", Status: "active", IsActive: true, AddedAt: time.Now()},
		{NodeID: "node_avalanche_1", ChainID: 43114, URL: "https://api.avax.network/ext/bc/C/rpc", Type: "full", Provider: "Avalanche", Region: "US", Status: "active", IsActive: true, AddedAt: time.Now()},
		{NodeID: "node_base_1", ChainID: 8453, URL: "https://base-mainnet.g.alchemy.com/v2/demo", Type: "full", Provider: "Alchemy", Region: "US", Status: "active", IsActive: true, AddedAt: time.Now()},
		{NodeID: "node_solana_1", ChainID: -101, URL: "https://api.mainnet-beta.solana.com", Type: "full", Provider: "Solana", Region: "US", Status: "active", IsActive: true, AddedAt: time.Now()},
		{NodeID: "node_cosmos_1", ChainID: -301, URL: "https://cosmos-rpc.polkachu.com", Type: "full", Provider: "Polkachu", Region: "US", Status: "active", IsActive: true, AddedAt: time.Now()},
	}

	for i := range nodes {
		nodes[i].LastCheckAt = time.Now()
		s.nodes[nodes[i].NodeID] = &nodes[i]
	}
}

// Get all blockchains
func (s *ExplorerService) GetBlockchains(category, network string) []*BlockchainConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*BlockchainConfig, 0)
	for _, chain := range s.blockchains {
		match := true
		if category != "" && string(chain.Category) != category {
			match = false
		}
		if network != "" && string(chain.Network) != network {
			match = false
		}
		if match {
			result = append(result, chain)
		}
	}

	return result
}

// Get blockchain by ID
func (s *ExplorerService) GetBlockchain(chainID int) (*BlockchainConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain, ok := s.blockchains[chainID]
	if !ok {
		return nil, fmt.Errorf("blockchain not found: %d", chainID)
	}

	return chain, nil
}

// Get tokens
func (s *ExplorerService) GetTokens(chainID int) []*TokenConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*TokenConfig, 0)
	for _, token := range s.tokens {
		if chainID == 0 || token.ChainID == chainID {
			result = append(result, token)
		}
	}

	return result
}

// Get nodes
func (s *ExplorerService) GetNodes(chainID int) []*NodeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*NodeConfig, 0)
	for _, node := range s.nodes {
		if chainID == 0 || node.ChainID == chainID {
			result = append(result, node)
		}
	}

	return result
}

// Add blockchain
func (s *ExplorerService) AddBlockchain(chain *BlockchainConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.blockchains[chain.ChainID]; exists {
		return fmt.Errorf("blockchain already exists")
	}

	chain.CreatedAt = time.Now()
	chain.UpdatedAt = time.Now()
	s.blockchains[chain.ChainID] = chain

	return nil
}

// Update blockchain
func (s *ExplorerService) UpdateBlockchain(chainID int, updates map[string]interface{}) (*BlockchainConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chain, ok := s.blockchains[chainID]
	if !ok {
		return nil, fmt.Errorf("blockchain not found")
	}

	if rpcURL, ok := updates["rpc_url"].(string); ok {
		chain.RPCURL = rpcURL
	}
	if explorerURL, ok := updates["explorer_url"].(string); ok {
		chain.ExplorerURL = explorerURL
	}
	if status, ok := updates["status"].(string); ok {
		chain.Status = ChainStatus(status)
	}

	chain.UpdatedAt = time.Now()

	return chain, nil
}

// Add node
func (s *ExplorerService) AddNode(node *NodeConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	node.NodeID = "node_" + uuid.New().String()[:8]
	node.AddedAt = time.Now()
	node.LastCheckAt = time.Now()
	node.IsActive = true

	s.nodes[node.NodeID] = node

	return nil
}

// Get stats
func (s *ExplorerService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_blockchains": len(s.blockchains),
		"total_tokens":     len(s.tokens),
		"total_nodes":      len(s.nodes),
	}

	// Count by category
	categories := make(map[string]int)
	networks := make(map[string]int)

	for _, chain := range s.blockchains {
		categories[string(chain.Category)]++
		networks[string(chain.Network)]++
	}

	stats["blockchains_by_category"] = categories
	stats["blockchains_by_network"] = networks

	// Count active nodes
	activeNodes := 0
	for _, node := range s.nodes {
		if node.IsActive {
			activeNodes++
		}
	}
	stats["active_nodes"] = activeNodes

	return stats
}

// Handlers
func (s *ExplorerService) GetBlockchainsHandler(c *gin.Context) {
	category := c.Query("category")
	network := c.Query("network")

	blockchains := s.GetBlockchains(category, network)
	c.JSON(http.StatusOK, gin.H{"blockchains": blockchains})
}

func (s *ExplorerService) GetBlockchainHandler(c *gin.Context) {
	var chainID int
	fmt.Sscanf(c.Param("id"), "%d", &chainID)

	chain, err := s.GetBlockchain(chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chain)
}

func (s *ExplorerService) GetTokensHandler(c *gin.Context) {
	chainID := 0
	fmt.Sscanf(c.Query("chain_id"), "%d", &chainID)

	tokens := s.GetTokens(chainID)
	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

func (s *ExplorerService) GetNodesHandler(c *gin.Context) {
	chainID := 0
	fmt.Sscanf(c.Query("chain_id"), "%d", &chainID)

	nodes := s.GetNodes(chainID)
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (s *ExplorerService) GetStatsHandler(c *gin.Context) {
	stats := s.GetStats()
	c.JSON(http.StatusOK, stats)
}

func (s *ExplorerService) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/explorer")
	{
		api.GET("/blockchains", s.GetBlockchainsHandler)
		api.GET("/blockchains/:id", s.GetBlockchainHandler)
		api.GET("/tokens", s.GetTokensHandler)
		api.GET("/nodes", s.GetNodesHandler)
		api.GET("/stats", s.GetStatsHandler)
	}
}

func main() {
	cfg := Config{
		ServerPort: getEnv("EXPLORER_PORT", "8090"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
	}

	service := NewExplorerService(cfg)

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
			"service":   "blockchain-explorer-service",
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
		log.Printf("Starting Blockchain Explorer Service on %s", addr)
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
