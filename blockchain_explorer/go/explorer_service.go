// TigerWallet Blockchain Explorer API
// High-performance Go implementation for blockchain exploration
// Supports EVM and non-EVM chains

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
}

func DefaultConfig() *Config {
	return &Config{
		Port:           ":8081",
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

// ============================================================================
// Data Types
// ============================================================================

type Block struct {
	Number           uint64    `json:"number"`
	Hash             string    `json:"hash"`
	ParentHash       string    `json:"parentHash"`
	Timestamp        uint64    `json:"timestamp"`
	Transactions     []string  `json:"transactions"`
	GasUsed          uint64    `json:"gasUsed"`
	GasLimit         uint64    `json:"gasLimit"`
	Miner            string    `json:"miner"`
	Difficulty       uint64    `json:"difficulty"`
	TotalDifficulty  uint64    `json:"totalDifficulty"`
	Size             uint64    `json:"size"`
	ExtraData        string    `json:"extraData"`
	Nonce            string    `json:"nonce"`
	BaseFeePerGas    uint64    `json:"baseFeePerGas,omitempty"`
}

type Transaction struct {
	Hash             string    `json:"hash"`
	BlockNumber      uint64    `json:"blockNumber"`
	BlockHash        string    `json:"blockHash"`
	TransactionIndex uint64    `json:"transactionIndex"`
	From             string    `json:"from"`
	To               string    `json:"to"`
	Value            string    `json:"value"`
	Gas              uint64    `json:"gas"`
	GasPrice         uint64    `json:"gasPrice"`
	Nonce            uint64    `json:"nonce"`
	Input            string    `json:"input"`
	SignatureV       uint64    `json:"v"`
	SignatureR       string    `json:"r"`
	SignatureS       string    `json:"s"`
	Status           uint64    `json:"status"`
	Timestamp        uint64    `json:"timestamp"`
}

type Address struct {
	Address          string    `json:"address"`
	Balance          string    `json:"balance"`
	TransactionCount uint64    `json:"transactionCount"`
	Code             string    `json:"code,omitempty"`
	IsContract       bool      `json:"isContract"`
	TokenTransfers   []Transfer `json:"tokenTransfers,omitempty"`
}

type Token struct {
	Address          string `json:"address"`
	Name             string `json:"name"`
	Symbol           string `json:"symbol"`
	Decimals         uint8  `json:"decimals"`
	TotalSupply      string `json:"totalSupply"`
	Type             string `json:"type"`
	Holders          int    `json:"holders"`
	Transfers        int    `json:"transfers"`
}

type Transfer struct {
	Hash            string `json:"hash"`
	BlockNumber     uint64 `json:"blockNumber"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	Token           string `json:"token"`
	Timestamp       uint64 `json:"timestamp"`
}

type Chain struct {
	ID            uint64   `json:"id"`
	Name          string   `json:"name"`
	Symbol        string   `json:"symbol"`
	RPCURL        string   `json:"rpcURL"`
	ExplorerURL   string   `json:"explorerURL"`
	Icon          string   `json:"icon"`
	BlockTime     uint64   `json:"blockTime"`
}

type Stats struct {
	TotalBlocks      uint64  `json:"totalBlocks"`
	TotalTransactions uint64  `json:"totalTransactions"`
	TotalAddresses   uint64  `json:"totalAddresses"`
	TotalTokens      uint64  `json:"totalTokens"`
	GasPrice         uint64  `json:"gasPrice"`
	HashRate         string  `json:"hashRate"`
}

// ============================================================================
// Explorer API
// ============================================================================

type ExplorerAPI struct {
	router *mux.Router
	config *Config
	chains map[uint64]*Chain
}

func NewExplorerAPI(cfg *Config) *ExplorerAPI {
	api := &ExplorerAPI{
		router: mux.NewRouter(),
		config: cfg,
		chains: make(map[uint64]*Chain),
	}
	api.initializeRoutes()
	api.initializeChains()
	return api
}

func (e *ExplorerAPI) initializeRoutes() {
	e.router.HandleFunc("/health", e.handleHealth).Methods("GET")
	e.router.HandleFunc("/stats", e.handleStats).Methods("GET")

	// Block routes
	e.router.HandleFunc("/blocks", e.handleBlocks).Methods("GET")
	e.router.HandleFunc("/blocks/{number}", e.handleBlock).Methods("GET")
	e.router.HandleFunc("/blocks/{number}/transactions", e.handleBlockTransactions).Methods("GET")

	// Transaction routes
	e.router.HandleFunc("/transactions", e.handleTransactions).Methods("GET")
	e.router.HandleFunc("/transactions/{hash}", e.handleTransaction).Methods("GET")

	// Address routes
	e.router.HandleFunc("/addresses/{address}", e.handleAddress).Methods("GET")
	e.router.HandleFunc("/addresses/{address}/transactions", e.handleAddressTransactions).Methods("GET")
	e.router.HandleFunc("/addresses/{address}/tokens", e.handleAddressTokens).Methods("GET")

	// Token routes
	e.router.HandleFunc("/tokens", e.handleTokens).Methods("GET")
	e.router.HandleFunc("/tokens/{address}", e.handleToken).Methods("GET")
	e.router.HandleFunc("/tokens/{address}/holders", e.handleTokenHolders).Methods("GET")
	e.router.HandleFunc("/tokens/{address}/transfers", e.handleTokenTransfers).Methods("GET")

	// Chain routes
	e.router.HandleFunc("/chains", e.handleChains).Methods("GET")
	e.router.HandleFunc("/chains/{id}", e.handleChain).Methods("GET")

	// Search
	e.router.HandleFunc("/search", e.handleSearch).Methods("GET")
}

func (e *ExplorerAPI) initializeChains() {
	e.chains[1] = &Chain{ID: 1, Name: "Ethereum", Symbol: "ETH", RPCURL: "https://eth.llamarpc.com", ExplorerURL: "https://etherscan.io", Icon: "🔷", BlockTime: 12}
	e.chains[56] = &Chain{ID: 56, Name: "BNB Chain", Symbol: "BNB", RPCURL: "https://bsc-dataseed.binance.org", ExplorerURL: "https://bscscan.com", Icon: "🟡", BlockTime: 3}
	e.chains[137] = &Chain{ID: 137, Name: "Polygon", Symbol: "MATIC", RPCURL: "https://polygon-rpc.com", ExplorerURL: "https://polygonscan.com", Icon: "🟣", BlockTime: 2}
	e.chains[42161] = &Chain{ID: 42161, Name: "Arbitrum One", Symbol: "ETH", RPCURL: "https://arb1.arbitrum.io/rpc", ExplorerURL: "https://arbiscan.io", Icon: "🔵", BlockTime: 4}
	e.chains[10] = &Chain{ID: 10, Name: "Optimism", Symbol: "ETH", RPCURL: "https://mainnet.optimism.io", ExplorerURL: "https://optimistic.etherscan.io", Icon: "🔴", BlockTime: 2}
	e.chains[43114] = &Chain{ID: 43114, Name: "Avalanche C-Chain", Symbol: "AVAX", RPCURL: "https://api.avax.network/ext/bc/C/rpc", ExplorerURL: "https://snowtrace.io", Icon: "❄️", BlockTime: 2}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (e *ExplorerAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "tigerswap-explorer-api",
		"version":   "1.0.0",
	})
}

func (e *ExplorerAPI) handleStats(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, Stats{
		TotalBlocks:      18945632,
		TotalTransactions: 156782345,
		TotalAddresses:   2345678,
		TotalTokens:      45678,
		GasPrice:         25000000000,
		HashRate:         "156.78 TH/s",
	})
}

func (e *ExplorerAPI) handleBlocks(w http.ResponseWriter, r *http.Request) {
	limit := getQueryInt(r, "limit", 20)
	offset := getQueryInt(r, "offset", 0)

	blocks := make([]Block, limit)
	for i := 0; i < limit; i++ {
		blkNum := 18945632 - uint64(offset+i)
		blocks[i] = Block{
			Number:       blkNum,
			Hash:         generateHash(),
			Timestamp:    uint64(time.Now().Unix()) - uint64(i*12),
			Transactions: []string{generateHash(), generateHash()},
			GasUsed:      15000000,
			GasLimit:     30000000,
			Miner:        "0x742d35Cc6634C0532925a3b844Bc9e7595f12eB3",
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"blocks": blocks,
		"total":  18945632,
	})
}

func (e *ExplorerAPI) handleBlock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	number := vars["number"]

	blockNum := parseBlockNumber(number)
	respondJSON(w, http.StatusOK, Block{
		Number:           blockNum,
		Hash:             generateHash(),
		ParentHash:       generateHash(),
		Timestamp:        uint64(time.Now().Unix()),
		Transactions:     []string{generateHash(), generateHash(), generateHash()},
		GasUsed:          15000000,
		GasLimit:         30000000,
		Miner:            "0x742d35Cc6634C0532925a3b844Bc9e7595f12eB3",
		Difficulty:       5123456789,
		TotalDifficulty: 12345678901234567890,
		Size:             45678,
		ExtraData:        "0x",
		Nonce:            "0x0000000000000000",
	})
}

func (e *ExplorerAPI) handleBlockTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	number := vars["number"]

	txs := make([]Transaction, 10)
	for i := 0; i < 10; i++ {
		txs[i] = Transaction{
			Hash:             generateHash(),
			BlockNumber:      parseBlockNumber(number),
			BlockHash:        generateHash(),
			TransactionIndex: uint64(i),
			From:             "0x742d35Cc6634C0532925a3b844Bc9e7595f12eB3",
			To:               "0x8ba1f109551bD432803012645Ac136ddd64DBA72",
			Value:            "1000000000000000000",
			Gas:              21000,
			GasPrice:         25000000000,
			Nonce:            uint64(i),
			Input:            "0x",
			Status:           1,
			Timestamp:        uint64(time.Now().Unix()),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": txs,
	})
}

func (e *ExplorerAPI) handleTransactions(w http.ResponseWriter, r *http.Request) {
	limit := getQueryInt(r, "limit", 20)
	address := r.URL.Query().Get("address")

	txs := make([]Transaction, limit)
	for i := 0; i < limit; i++ {
		txs[i] = Transaction{
			Hash:             generateHash(),
			BlockNumber:      18945632 - uint64(i),
			BlockHash:        generateHash(),
			TransactionIndex: uint64(i),
			From:             address,
			To:               "0x8ba1f109551bD432803012645Ac136ddd64DBA72",
			Value:            "1000000000000000000",
			Status:           1,
			Timestamp:        uint64(time.Now().Unix()) - uint64(i*300),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": txs,
	})
}

func (e *ExplorerAPI) handleTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	respondJSON(w, http.StatusOK, Transaction{
		Hash:             hash,
		BlockNumber:      18945632,
		BlockHash:        generateHash(),
		TransactionIndex: 5,
		From:             "0x742d35Cc6634C0532925a3b844Bc9e7595f12eB3",
		To:               "0x8ba1f109551bD432803012645Ac136ddd64DBA72",
		Value:            "1000000000000000000",
		Gas:              21000,
		GasPrice:         25000000000,
		Nonce:            42,
		Input:            "0xa9059cbb0000000000000000000000008ba1f109551bd432803012645ac136ddd64dba7200000000000000000000000000000000000000000000000000000de0b6b3a7640000",
		Status:           1,
		Timestamp:        uint64(time.Now().Unix()),
	})
}

func (e *ExplorerAPI) handleAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	respondJSON(w, http.StatusOK, Address{
		Address:          address,
		Balance:          "1000000000000000000",
		TransactionCount: 156,
		IsContract:       false,
	})
}

func (e *ExplorerAPI) handleAddressTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["address"]

	txs := make([]Transaction, 10)
	for i := 0; i < 10; i++ {
		txs[i] = Transaction{
			Hash:    generateHash(),
			From:    "0x742d35Cc6634C0532925a3b844Bc9e7595f12eB3",
			To:      "0x8ba1f109551bD432803012645Ac136ddd64DBA72",
			Value:   "1000000000000000000",
			Status:  1,
			Timestamp: uint64(time.Now().Unix()) - uint64(i*3600),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": txs,
	})
}

func (e *ExplorerAPI) handleAddressTokens(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["address"]

	tokens := []map[string]interface{}{
		{"address": "0xdAC17F958D2ee523a2206206994597C13D831ec7", "symbol": "USDT", "balance": "1000000000"},
		{"address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "symbol": "USDC", "balance": "500000000"},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tokens": tokens,
	})
}

func (e *ExplorerAPI) handleTokens(w http.ResponseWriter, r *http.Request) {
	tokens := []Token{
		{Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Name: "Tether USD", Symbol: "USDT", Decimals: 6, TotalSupply: "1000000000000000", Type: "ERC20", Holders: 5000000, Transfers: 10000000},
		{Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Name: "USD Coin", Symbol: "USDC", Decimals: 6, TotalSupply: "500000000000000", Type: "ERC20", Holders: 3000000, Transfers: 8000000},
		{Address: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", Name: "Wrapped Bitcoin", Symbol: "WBTC", Decimals: 8, TotalSupply: "150000000000", Type: "ERC20", Holders: 50000, Transfers: 500000},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tokens": tokens,
	})
}

func (e *ExplorerAPI) handleToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	respondJSON(w, http.StatusOK, Token{
		Address:     address,
		Name:        "Tether USD",
		Symbol:      "USDT",
		Decimals:    6,
		TotalSupply: "1000000000000000",
		Type:        "ERC20",
		Holders:     5000000,
		Transfers:   10000000,
	})
}

func (e *ExplorerAPI) handleTokenHolders(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["address"]

	holders := []map[string]interface{}{
		{"address": "0x742d35Cc6634C0532925a3b844Bc9e7595f12eB3", "balance": "1000000000000"},
		{"address": "0x8ba1f109551bD432803012645Ac136ddd64DBA72", "balance": "500000000000"},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"holders": holders,
	})
}

func (e *ExplorerAPI) handleTokenTransfers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["address"]

	transfers := make([]Transfer, 10)
	for i := 0; i < 10; i++ {
		transfers[i] = Transfer{
			Hash:        generateHash(),
			BlockNumber: 18945632 - uint64(i),
			From:        "0x742d35Cc6634C0532925a3b844Bc9e7595f12eB3",
			To:          "0x8ba1f109551bD432803012645Ac136ddd64DBA72",
			Value:       "1000000000",
			Token:       "USDT",
			Timestamp:   uint64(time.Now().Unix()) - uint64(i*120),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"transfers": transfers,
	})
}

func (e *ExplorerAPI) handleChains(w http.ResponseWriter, r *http.Request) {
	chains := make([]*Chain, 0, len(e.chains))
	for _, chain := range e.chains {
		chains = append(chains, chain)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"chains": chains,
	})
}

func (e *ExplorerAPI) handleChain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var chainID uint64
	fmt.Sscanf(id, "%d", &chainID)

	chain, ok := e.chains[chainID]
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "Chain not found",
		})
		return
	}

	respondJSON(w, http.StatusOK, chain)
}

func (e *ExplorerAPI) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Query parameter 'q' is required",
		})
		return
	}

	query = strings.ToLower(query)

	// Check if it's a transaction hash
	if len(query) == 66 && strings.HasPrefix(query, "0x") {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"type":  "tx",
			"result": generateHash(),
		})
		return
	}

	// Check if it's a block number
	var blockNum uint64
	if _, err := fmt.Sscanf(query, "%d", &blockNum); err == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"type":  "block",
			"result": blockNum,
		})
		return
	}

	// Check if it's an address
	if len(query) == 42 && strings.HasPrefix(query, "0x") {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"type":  "address",
			"result": query,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"type":  "unknown",
		"result": nil,
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func getQueryInt(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	var result int
	fmt.Sscanf(value, "%d", &result)
	return result
}

func parseBlockNumber(s string) uint64 {
	if s == "latest" || s == "0" {
		return 18945632
	}
	var num uint64
	fmt.Sscanf(s, "%d", &num)
	return num
}

func generateHash() string {
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte((i * 17 + 13) % 256)
	}
	return "0x" + hex.EncodeToString(b)
}

func generateAddress() string {
	b := make([]byte, 20)
	for i := range b {
		b[i] = byte((i * 13 + 7) % 256)
	}
	return "0x" + hex.EncodeToString(b)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := DefaultConfig()

	server := &http.Server{
		Addr:           cfg.Port,
		Handler:        NewExplorerAPI(cfg).router,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}

	log.Printf("TigerWallet Blockchain Explorer API starting on %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
