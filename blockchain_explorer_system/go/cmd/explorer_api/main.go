// TigerWallet Blockchain Explorer API
//
// Serves EVM chain data straight from JSON-RPC nodes. Every response field is
// read from the chain; nothing is synthesised. Endpoints that need an indexed
// database (address transaction history, token holder lists) answer 501 rather
// than inventing data, because a plain RPC node cannot serve them.
//
// Chain endpoints come from EXPLORER_RPC_URLS as a comma separated list of
// "chainID=url" pairs, falling back to the public endpoints below.

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
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
	RPCURLs        map[uint64]string
}

func DefaultConfig() *Config {
	return &Config{
		Port:           ":8081",
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
		RPCURLs:        defaultRPCURLs(),
	}
}

func defaultRPCURLs() map[uint64]string {
	urls := map[uint64]string{
		1:     "https://ethereum-rpc.publicnode.com",
		10:    "https://mainnet.optimism.io",
		56:    "https://bsc-dataseed.binance.org",
		137:   "https://polygon-bor-rpc.publicnode.com",
		250:   "https://rpcapi.fantom.network",
		324:   "https://mainnet.era.zksync.io",
		8453:  "https://base-rpc.publicnode.com",
		42161: "https://arb1.arbitrum.io/rpc",
		42220: "https://forno.celo.org",
		43114: "https://avalanche-c-chain-rpc.publicnode.com",
		59144: "https://rpc.linea.build",
	}
	for _, pair := range strings.Split(os.Getenv("EXPLORER_RPC_URLS"), ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		id, url, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		chainID, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
		if err != nil || strings.TrimSpace(url) == "" {
			continue
		}
		urls[chainID] = strings.TrimSpace(url)
	}
	return urls
}

// ============================================================================
// Data Types
// ============================================================================

type Block struct {
	Number           uint64   `json:"number"`
	Hash             string   `json:"hash"`
	ParentHash       string   `json:"parentHash"`
	Timestamp        uint64   `json:"timestamp"`
	Transactions     []string `json:"transactions"`
	GasUsed          uint64   `json:"gasUsed"`
	GasLimit         uint64   `json:"gasLimit"`
	Miner            string   `json:"miner"`
	BaseFeePerGas    string   `json:"baseFeePerGas,omitempty"`
	Size             uint64   `json:"size"`
	ExtraData        string   `json:"extraData"`
	Nonce            string   `json:"nonce"`
	TransactionCount int      `json:"transactionCount"`
}

type Transaction struct {
	Hash             string `json:"hash"`
	BlockNumber      uint64 `json:"blockNumber"`
	BlockHash        string `json:"blockHash"`
	TransactionIndex uint64 `json:"transactionIndex"`
	From             string `json:"from"`
	To               string `json:"to"`
	Value            string `json:"value"`
	Gas              uint64 `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	Nonce            uint64 `json:"nonce"`
	Input            string `json:"input"`
	Status           uint64 `json:"status"`
	Timestamp        uint64 `json:"timestamp"`
}

type Address struct {
	Address          string `json:"address"`
	Balance          string `json:"balance"`
	TransactionCount uint64 `json:"transactionCount"`
	IsContract       bool   `json:"isContract"`
	CodeSize         int    `json:"codeSize"`
}

type Token struct {
	Address     string `json:"address"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    uint8  `json:"decimals"`
	TotalSupply string `json:"totalSupply"`
	Type        string `json:"type"`
}

type Chain struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	RPCURL      string `json:"rpcURL"`
	ExplorerURL string `json:"explorerURL"`
	BlockTime   uint64 `json:"blockTime"`
}

// ============================================================================
// Explorer API
// ============================================================================

type ExplorerAPI struct {
	router *mux.Router
	config *Config
	chains map[uint64]*Chain
	client *http.Client
}

func NewExplorerAPI(cfg *Config) *ExplorerAPI {
	api := &ExplorerAPI{
		router: mux.NewRouter(),
		config: cfg,
		chains: make(map[uint64]*Chain),
		client: &http.Client{Timeout: 20 * time.Second},
	}
	api.initializeRoutes()
	api.initializeChains()
	return api
}

func (e *ExplorerAPI) initializeRoutes() {
	e.router.HandleFunc("/health", e.handleHealth).Methods("GET")
	e.router.HandleFunc("/stats", e.handleStats).Methods("GET")

	e.router.HandleFunc("/blocks", e.handleBlocks).Methods("GET")
	e.router.HandleFunc("/blocks/{number}", e.handleBlock).Methods("GET")
	e.router.HandleFunc("/blocks/{number}/transactions", e.handleBlockTransactions).Methods("GET")

	e.router.HandleFunc("/transactions", e.handleTransactions).Methods("GET")
	e.router.HandleFunc("/transactions/{hash}", e.handleTransaction).Methods("GET")

	e.router.HandleFunc("/addresses/{address}", e.handleAddress).Methods("GET")
	e.router.HandleFunc("/addresses/{address}/transactions", e.handleAddressTransactions).Methods("GET")
	e.router.HandleFunc("/addresses/{address}/tokens", e.handleAddressTokens).Methods("GET")

	e.router.HandleFunc("/tokens", e.handleTokens).Methods("GET")
	e.router.HandleFunc("/tokens/{address}", e.handleToken).Methods("GET")
	e.router.HandleFunc("/tokens/{address}/holders", e.handleTokenHolders).Methods("GET")
	e.router.HandleFunc("/tokens/{address}/transfers", e.handleTokenTransfers).Methods("GET")

	e.router.HandleFunc("/chains", e.handleChains).Methods("GET")
	e.router.HandleFunc("/chains/{id}", e.handleChain).Methods("GET")

	e.router.HandleFunc("/search", e.handleSearch).Methods("GET")
}

func (e *ExplorerAPI) initializeChains() {
	type chainMeta struct {
		name      string
		symbol    string
		explorer  string
		blockTime uint64
	}
	meta := map[uint64]chainMeta{
		1:     {"Ethereum", "ETH", "https://etherscan.io", 12},
		10:    {"Optimism", "ETH", "https://optimistic.etherscan.io", 2},
		56:    {"BNB Chain", "BNB", "https://bscscan.com", 3},
		137:   {"Polygon", "POL", "https://polygonscan.com", 2},
		250:   {"Fantom", "FTM", "https://ftmscan.com", 1},
		324:   {"zkSync Era", "ETH", "https://explorer.zksync.io", 1},
		8453:  {"Base", "ETH", "https://basescan.org", 2},
		42161: {"Arbitrum One", "ETH", "https://arbiscan.io", 1},
		42220: {"Celo", "CELO", "https://celoscan.io", 5},
		43114: {"Avalanche C-Chain", "AVAX", "https://snowtrace.io", 2},
		59144: {"Linea", "ETH", "https://lineascan.build", 3},
	}
	for id, rpcURL := range e.config.RPCURLs {
		m, ok := meta[id]
		if !ok {
			m = chainMeta{name: fmt.Sprintf("Chain %d", id), symbol: "NATIVE", blockTime: 12}
		}
		e.chains[id] = &Chain{
			ID:          id,
			Name:        m.name,
			Symbol:      m.symbol,
			RPCURL:      rpcURL,
			ExplorerURL: m.explorer,
			BlockTime:   m.blockTime,
		}
	}
}

// ============================================================================
// JSON-RPC
// ============================================================================

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *ExplorerAPI) rpc(chainID uint64, method string, params ...any) (json.RawMessage, error) {
	url, ok := e.config.RPCURLs[chainID]
	if !ok {
		return nil, fmt.Errorf("no RPC endpoint configured for chain %d", chainID)
	}
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TigerWallet-Explorer/1.0")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rpc %s unreachable: %w", method, err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var out rpcResponse
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("rpc %s returned a malformed response", method)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("rpc %s failed: %s", method, out.Error.Message)
	}
	return out.Result, nil
}

func (e *ExplorerAPI) chainFromRequest(r *http.Request) (uint64, error) {
	raw := r.URL.Query().Get("chain_id")
	if raw == "" {
		raw = r.URL.Query().Get("chainId")
	}
	if raw == "" {
		return 1, nil
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid chain_id %q", raw)
	}
	if _, ok := e.config.RPCURLs[id]; !ok {
		return 0, fmt.Errorf("chain %d is not configured", id)
	}
	return id, nil
}

// ============================================================================
// Decoding helpers
// ============================================================================

type rawBlock struct {
	Number     string `json:"number"`
	Hash       string `json:"hash"`
	ParentHash string `json:"parentHash"`
	Timestamp  string `json:"timestamp"`
	// eth_getBlockByNumber returns hashes when full=false and objects when
	// full=true, so the field stays raw and is decoded per call site.
	Transactions json.RawMessage `json:"transactions"`
	GasUsed      string          `json:"gasUsed"`
	GasLimit     string          `json:"gasLimit"`
	Miner        string          `json:"miner"`
	BaseFee      string          `json:"baseFeePerGas"`
	Size         string          `json:"size"`
	ExtraData    string          `json:"extraData"`
	Nonce        string          `json:"nonce"`
}

func (b rawBlock) transactionHashes() []string {
	var hashes []string
	if len(b.Transactions) == 0 {
		return hashes
	}
	if err := json.Unmarshal(b.Transactions, &hashes); err == nil {
		return hashes
	}
	var objs []struct {
		Hash string `json:"hash"`
	}
	if err := json.Unmarshal(b.Transactions, &objs); err != nil {
		return hashes
	}
	for _, o := range objs {
		hashes = append(hashes, o.Hash)
	}
	return hashes
}

type rawTx struct {
	Hash        string `json:"hash"`
	BlockNumber string `json:"blockNumber"`
	BlockHash   string `json:"blockHash"`
	TxIndex     string `json:"transactionIndex"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	Gas         string `json:"gas"`
	GasPrice    string `json:"gasPrice"`
	Nonce       string `json:"nonce"`
	Input       string `json:"input"`
}

func hexUint(s string) uint64 {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0
	}
	return n
}

func hexBigString(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "0"
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
	if !ok {
		return "0"
	}
	return n.String()
}

func decodeABIString(hexData string) string {
	raw, err := hex.DecodeString(strings.TrimPrefix(hexData, "0x"))
	if err != nil || len(raw) < 64 {
		return ""
	}
	offset := new(big.Int).SetBytes(raw[:32]).Uint64()
	if offset+32 > uint64(len(raw)) {
		return ""
	}
	length := new(big.Int).SetBytes(raw[offset : offset+32]).Uint64()
	if offset+32+length > uint64(len(raw)) {
		return ""
	}
	return strings.TrimRight(string(raw[offset+32:offset+32+length]), "\x00")
}

func blockFromRaw(b rawBlock) Block {
	hashes := b.transactionHashes()
	return Block{
		Number:           hexUint(b.Number),
		Hash:             b.Hash,
		ParentHash:       b.ParentHash,
		Timestamp:        hexUint(b.Timestamp),
		Transactions:     hashes,
		GasUsed:          hexUint(b.GasUsed),
		GasLimit:         hexUint(b.GasLimit),
		Miner:            b.Miner,
		BaseFeePerGas:    hexBigString(b.BaseFee),
		Size:             hexUint(b.Size),
		ExtraData:        b.ExtraData,
		Nonce:            b.Nonce,
		TransactionCount: len(hashes),
	}
}

func txFromRaw(t rawTx, timestamp uint64, status uint64) Transaction {
	return Transaction{
		Hash:             t.Hash,
		BlockNumber:      hexUint(t.BlockNumber),
		BlockHash:        t.BlockHash,
		TransactionIndex: hexUint(t.TxIndex),
		From:             t.From,
		To:               t.To,
		Value:            hexBigString(t.Value),
		Gas:              hexUint(t.Gas),
		GasPrice:         hexBigString(t.GasPrice),
		Nonce:            hexUint(t.Nonce),
		Input:            t.Input,
		Status:           status,
		Timestamp:        timestamp,
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (e *ExplorerAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	reachable := make([]uint64, 0, len(e.config.RPCURLs))
	unreachable := make(map[string]string)
	for id := range e.config.RPCURLs {
		if _, err := e.rpc(id, "eth_blockNumber"); err != nil {
			unreachable[strconv.FormatUint(id, 10)] = err.Error()
			continue
		}
		reachable = append(reachable, id)
	}
	status := "healthy"
	code := http.StatusOK
	if len(reachable) == 0 {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}
	respondJSON(w, code, map[string]any{
		"status":             status,
		"timestamp":          time.Now().Unix(),
		"service":            "tigerwallet-explorer-api",
		"reachable_chains":   reachable,
		"unreachable_chains": unreachable,
	})
}

func (e *ExplorerAPI) handleStats(w http.ResponseWriter, r *http.Request) {
	chainID, err := e.chainFromRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	headRaw, err := e.rpc(chainID, "eth_blockNumber")
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	var head string
	_ = json.Unmarshal(headRaw, &head)

	gasRaw, err := e.rpc(chainID, "eth_gasPrice")
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	var gasPrice string
	_ = json.Unmarshal(gasRaw, &gasPrice)

	chainIDRaw, err := e.rpc(chainID, "eth_chainId")
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	var reportedChainID string
	_ = json.Unmarshal(chainIDRaw, &reportedChainID)

	respondJSON(w, http.StatusOK, map[string]any{
		"blockHeight": hexUint(head),
		"gasPrice":    hexBigString(gasPrice),
		"chainId":     hexUint(reportedChainID),
		"chainName":   e.chainName(chainID),
		"note":        "address and token totals need an indexer; this endpoint reports chain head and gas only",
	})
}

func (e *ExplorerAPI) chainName(chainID uint64) string {
	if c, ok := e.chains[chainID]; ok {
		return c.Name
	}
	return fmt.Sprintf("Chain %d", chainID)
}

func (e *ExplorerAPI) handleBlocks(w http.ResponseWriter, r *http.Request) {
	chainID, err := e.chainFromRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	limit := getQueryInt(r, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	headRaw, err := e.rpc(chainID, "eth_blockNumber")
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	var headHex string
	_ = json.Unmarshal(headRaw, &headHex)
	head := hexUint(headHex)

	blocks := make([]Block, 0, limit)
	for i := 0; i < limit && head > uint64(i); i++ {
		number := head - uint64(i)
		raw, err := e.rpc(chainID, "eth_getBlockByNumber", fmt.Sprintf("0x%x", number), false)
		if err != nil {
			continue
		}
		var rb rawBlock
		if json.Unmarshal(raw, &rb) != nil {
			continue
		}
		blocks = append(blocks, blockFromRaw(rb))
	}
	if len(blocks) == 0 {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": "no blocks could be read from the node"})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"blocks": blocks, "head": head})
}

func (e *ExplorerAPI) blockTag(w http.ResponseWriter, r *http.Request) (string, bool) {
	number := mux.Vars(r)["number"]
	if number == "latest" {
		return "latest", true
	}
	n, err := strconv.ParseUint(number, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "block number must be decimal or 'latest'"})
		return "", false
	}
	return fmt.Sprintf("0x%x", n), true
}

func (e *ExplorerAPI) handleBlock(w http.ResponseWriter, r *http.Request) {
	chainID, err := e.chainFromRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	tag, ok := e.blockTag(w, r)
	if !ok {
		return
	}

	raw, err := e.rpc(chainID, "eth_getBlockByNumber", tag, false)
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if string(raw) == "null" {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "block not found"})
		return
	}
	var rb rawBlock
	if err := json.Unmarshal(raw, &rb); err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": "malformed block payload"})
		return
	}
	respondJSON(w, http.StatusOK, blockFromRaw(rb))
}

func (e *ExplorerAPI) handleBlockTransactions(w http.ResponseWriter, r *http.Request) {
	chainID, err := e.chainFromRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	tag, ok := e.blockTag(w, r)
	if !ok {
		return
	}

	raw, err := e.rpc(chainID, "eth_getBlockByNumber", tag, true)
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if string(raw) == "null" {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "block not found"})
		return
	}
	var rb struct {
		Timestamp    string          `json:"timestamp"`
		Transactions json.RawMessage `json:"transactions"`
	}
	if err := json.Unmarshal(raw, &rb); err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": "malformed block payload"})
		return
	}
	timestamp := hexUint(rb.Timestamp)

	var rawTxs []rawTx
	if len(rb.Transactions) > 0 {
		if err := json.Unmarshal(rb.Transactions, &rawTxs); err != nil {
			respondJSON(w, http.StatusBadGateway, map[string]any{"error": "malformed transaction list"})
			return
		}
	}

	txs := make([]Transaction, 0, len(rawTxs))
	for _, t := range rawTxs {
		status := uint64(0)
		if receipt, rerr := e.rpc(chainID, "eth_getTransactionReceipt", t.Hash); rerr == nil && string(receipt) != "null" {
			var rr struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(receipt, &rr) == nil {
				status = hexUint(rr.Status)
			}
		}
		txs = append(txs, txFromRaw(t, timestamp, status))
	}
	respondJSON(w, http.StatusOK, map[string]any{"transactions": txs})
}

func (e *ExplorerAPI) handleTransactions(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusNotImplemented, map[string]any{
		"error": "listing transactions by address requires an indexed explorer database",
		"hint":  "use /transactions/{hash} for a single transaction or /tokens/{address}/transfers for ERC-20 transfer logs",
	})
}

func (e *ExplorerAPI) handleTransaction(w http.ResponseWriter, r *http.Request) {
	chainID, err := e.chainFromRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	hash := mux.Vars(r)["hash"]
	if len(hash) != 66 || !strings.HasPrefix(hash, "0x") {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "a transaction hash must be 32 bytes of hex"})
		return
	}

	raw, err := e.rpc(chainID, "eth_getTransactionByHash", hash)
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if string(raw) == "null" {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "transaction not found"})
		return
	}
	var t rawTx
	if err := json.Unmarshal(raw, &t); err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": "malformed transaction payload"})
		return
	}

	var timestamp, status uint64
	if receipt, rerr := e.rpc(chainID, "eth_getTransactionReceipt", hash); rerr == nil && string(receipt) != "null" {
		var rr struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(receipt, &rr) == nil {
			status = hexUint(rr.Status)
		}
	}
	if t.BlockNumber != "" && t.BlockNumber != "0x0" {
		if blk, berr := e.rpc(chainID, "eth_getBlockByNumber", t.BlockNumber, false); berr == nil && string(blk) != "null" {
			var rb rawBlock
			if json.Unmarshal(blk, &rb) == nil {
				timestamp = hexUint(rb.Timestamp)
			}
		}
	}
	respondJSON(w, http.StatusOK, txFromRaw(t, timestamp, status))
}

func (e *ExplorerAPI) handleAddress(w http.ResponseWriter, r *http.Request) {
	chainID, err := e.chainFromRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	address := mux.Vars(r)["address"]
	if len(address) != 42 || !strings.HasPrefix(address, "0x") {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "an address must be 20 bytes of hex"})
		return
	}

	balanceRaw, err := e.rpc(chainID, "eth_getBalance", address, "latest")
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	nonceRaw, err := e.rpc(chainID, "eth_getTransactionCount", address, "latest")
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	codeRaw, err := e.rpc(chainID, "eth_getCode", address, "latest")
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	var balance, nonce, code string
	_ = json.Unmarshal(balanceRaw, &balance)
	_ = json.Unmarshal(nonceRaw, &nonce)
	_ = json.Unmarshal(codeRaw, &code)

	codeSize := (len(code) - 2) / 2
	respondJSON(w, http.StatusOK, Address{
		Address:          address,
		Balance:          hexBigString(balance),
		TransactionCount: hexUint(nonce),
		IsContract:       codeSize > 0,
		CodeSize:         codeSize,
	})
}

func (e *ExplorerAPI) handleAddressTransactions(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusNotImplemented, map[string]any{
		"error": "address transaction history requires an indexed explorer database",
		"hint":  "use /tokens/{address}/transfers for ERC-20 transfer logs",
	})
}

func (e *ExplorerAPI) handleAddressTokens(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusNotImplemented, map[string]any{
		"error": "token discovery per address requires an indexed explorer database",
	})
}

func (e *ExplorerAPI) handleTokens(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusNotImplemented, map[string]any{
		"error": "a global token list requires an indexed explorer database",
		"hint":  "query /tokens/{address} for metadata of a known contract",
	})
}

func (e *ExplorerAPI) handleToken(w http.ResponseWriter, r *http.Request) {
	chainID, err := e.chainFromRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	address := mux.Vars(r)["address"]
	if len(address) != 42 || !strings.HasPrefix(address, "0x") {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "a token address must be 20 bytes of hex"})
		return
	}

	codeRaw, err := e.rpc(chainID, "eth_getCode", address, "latest")
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	var code string
	_ = json.Unmarshal(codeRaw, &code)
	if len(code) <= 2 {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "no contract deployed at this address"})
		return
	}

	token := Token{Address: address, Type: "ERC20"}
	// name(), symbol() return a dynamic string; decimals() and totalSupply() return uint.
	for _, call := range []struct {
		selector string
		isString bool
		apply    func(string)
	}{
		{"0x06fdde03", true, func(v string) { token.Name = v }},
		{"0x95d89b41", true, func(v string) { token.Symbol = v }},
		{"0x313ce567", false, func(v string) { token.Decimals = uint8(hexUint(v)) }},
		{"0x18160ddd", false, func(v string) { token.TotalSupply = hexBigString(v) }},
	} {
		out, cerr := e.rpc(chainID, "eth_call", map[string]string{"to": address, "data": call.selector}, "latest")
		if cerr != nil {
			continue
		}
		var result string
		if json.Unmarshal(out, &result) != nil || len(result) < 4 {
			continue
		}
		if call.isString {
			call.apply(decodeABIString(result))
		} else {
			call.apply(result)
		}
	}
	if token.Symbol == "" && token.Name == "" {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "the contract does not implement the ERC-20 metadata interface"})
		return
	}
	respondJSON(w, http.StatusOK, token)
}

func (e *ExplorerAPI) handleTokenHolders(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusNotImplemented, map[string]any{
		"error": "token holder lists require an indexed explorer database",
	})
}

func (e *ExplorerAPI) handleTokenTransfers(w http.ResponseWriter, r *http.Request) {
	chainID, err := e.chainFromRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	address := mux.Vars(r)["address"]
	if len(address) != 42 || !strings.HasPrefix(address, "0x") {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "a token address must be 20 bytes of hex"})
		return
	}

	headRaw, err := e.rpc(chainID, "eth_blockNumber")
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	var headHex string
	_ = json.Unmarshal(headRaw, &headHex)
	head := hexUint(headHex)

	window := uint64(getQueryInt(r, "blocks", 5000))
	if window == 0 || window > 100000 {
		window = 5000
	}
	fromBlock := uint64(0)
	if head > window {
		fromBlock = head - window
	}

	logsRaw, err := e.rpc(chainID, "eth_getLogs", map[string]any{
		"fromBlock": fmt.Sprintf("0x%x", fromBlock),
		"toBlock":   "latest",
		"address":   address,
		"topics":    []string{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"},
	})
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	var logs []struct {
		TransactionHash string   `json:"transactionHash"`
		BlockNumber     string   `json:"blockNumber"`
		Topics          []string `json:"topics"`
		Data            string   `json:"data"`
	}
	if err := json.Unmarshal(logsRaw, &logs); err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]any{"error": "malformed log payload"})
		return
	}

	transfers := make([]map[string]any, 0, len(logs))
	for _, l := range logs {
		if len(l.Topics) < 3 {
			continue
		}
		transfers = append(transfers, map[string]any{
			"hash":        l.TransactionHash,
			"blockNumber": hexUint(l.BlockNumber),
			"from":        "0x" + strings.TrimPrefix(l.Topics[1], "0x000000000000000000000000"),
			"to":          "0x" + strings.TrimPrefix(l.Topics[2], "0x000000000000000000000000"),
			"value":       hexBigString(l.Data),
			"token":       address,
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"transfers": transfers,
		"fromBlock": fromBlock,
		"toBlock":   head,
	})
}

func (e *ExplorerAPI) handleChains(w http.ResponseWriter, r *http.Request) {
	chains := make([]*Chain, 0, len(e.chains))
	for _, chain := range e.chains {
		chains = append(chains, chain)
	}
	respondJSON(w, http.StatusOK, map[string]any{"chains": chains})
}

func (e *ExplorerAPI) handleChain(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "chain id must be decimal"})
		return
	}
	chain, ok := e.chains[id]
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "chain not found"})
		return
	}
	respondJSON(w, http.StatusOK, chain)
}

func (e *ExplorerAPI) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "query parameter 'q' is required"})
		return
	}
	chainID, err := e.chainFromRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if len(query) == 66 && strings.HasPrefix(query, "0x") {
		raw, err := e.rpc(chainID, "eth_getTransactionByHash", query)
		if err != nil {
			respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		if string(raw) == "null" {
			respondJSON(w, http.StatusNotFound, map[string]any{"error": "no transaction with that hash on this chain"})
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"type": "transaction", "value": query})
		return
	}

	if blockNum, perr := strconv.ParseUint(query, 10, 64); perr == nil {
		raw, err := e.rpc(chainID, "eth_getBlockByNumber", fmt.Sprintf("0x%x", blockNum), false)
		if err != nil {
			respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		if string(raw) == "null" {
			respondJSON(w, http.StatusNotFound, map[string]any{"error": "no block at that height on this chain"})
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"type": "block", "value": blockNum})
		return
	}

	if len(query) == 42 && strings.HasPrefix(query, "0x") {
		codeRaw, err := e.rpc(chainID, "eth_getCode", query, "latest")
		if err != nil {
			respondJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		var code string
		_ = json.Unmarshal(codeRaw, &code)
		kind := "address"
		if len(code) > 2 {
			kind = "contract"
		}
		respondJSON(w, http.StatusOK, map[string]any{"type": kind, "value": query})
		return
	}

	respondJSON(w, http.StatusNotFound, map[string]any{"error": "the query matches no transaction, block, or address"})
}

// ============================================================================
// Helper Functions
// ============================================================================

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func getQueryInt(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return result
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
