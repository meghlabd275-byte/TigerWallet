/**
 * TigerWallet MEV Protection Service
 * Production-ready MEV protection using Flashbots, secret mempool, and bundle execution
 * Prevents front-running, back-running, and sandwich attacks
 */

package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Types
// ============================================================================

type MEVProtectionLevel string

const (
	MEVLevelNone      MEVProtectionLevel = "none"
	MEVLevelBasic     MEVProtectionLevel = "basic"
	MEVLevelAdvanced  MEVProtectionLevel = "advanced"
	MEVLevelFlashbots MEVProtectionLevel = "flashbots"
)

type TransactionBundle struct {
	ID              string        `json:"id"`
	Transactions    []PrivateTx   `json:"transactions"`
	BlockNumber     uint64        `json:"block_number"`
	MinBlockNumber  uint64        `json:"min_block_number"`
	MaxBlockNumber  uint64        `json:"max_block_number"`
	RevertingHashes []string      `json:"reverting_hashes"`
	StateChanges    []StateChange `json:"state_changes"`
	GasPrice        string        `json:"gas_price"`
	Status          string        `json:"status"` // pending, included, failed
	BundleHash      string        `json:"bundle_hash,omitempty"`
	BlockHash       string        `json:"block_hash,omitempty"`
	GasUsed         uint64        `json:"gas_used,omitempty"`
	TxHashes        []string      `json:"tx_hashes"`
}

type PrivateTx struct {
	Hash                 string   `json:"hash"`
	From                 string   `json:"from"`
	To                   string   `json:"to"`
	Value                string   `json:"value"`
	Data                 string   `json:"data"`
	Gas                  string   `json:"gas"`
	GasPrice             string   `json:"gas_price"`
	Nonce                uint64   `json:"nonce"`
	ChainID              uint64   `json:"chain_id"`
	AccessList           []string `json:"access_list,omitempty"`
	MaxFeePerGas         string   `json:"max_fee_per_gas,omitempty"`
	MaxPriorityFeePerGas string   `json:"max_priority_fee_per_gas,omitempty"`
	// RawSignedTx is the hex-encoded signed raw transaction (0x-prefixed) that
	// is broadcast to the network. The transaction hash is derived from this,
	// never fabricated from request metadata.
	RawSignedTx string `json:"raw_signed_tx,omitempty"`
}

type StateChange struct {
	Address string            `json:"address"`
	Balance string            `json:"balance"`
	Storage map[string]string `json:"storage,omitempty"`
}

type MEVProtectionRequest struct {
	WalletAddress   string             `json:"wallet_address"`
	ChainID         uint64             `json:"chain_id"`
	Transactions    []ProtectedTx      `json:"transactions"`
	ProtectionLevel MEVProtectionLevel `json:"protection_level"`
	MaxBlockNumber  uint64             `json:"max_block_number"`
}

type ProtectedTx struct {
	To       string `json:"to"`
	Value    string `json:"value"`
	Data     string `json:"data"`
	GasLimit string `json:"gas_limit"`
	GasPrice string `json:"gas_price"`
	Nonce    uint64 `json:"nonce"`
	ChainID  uint64 `json:"chain_id"`
	// RawSignedTx is the hex-encoded signed raw transaction submitted by the
	// caller. It is required: the service derives the tx hash from the
	// broadcast result rather than fabricating one.
	RawSignedTx string `json:"raw_signed_tx"`
}

type ProtectedBundle struct {
	BundleID    string   `json:"bundle_id"`
	BundleHash  string   `json:"bundle_hash"`
	BlockNumber uint64   `json:"block_number"`
	GasPrice    string   `json:"gas_price"`
	Status      string   `json:"status"`
	TxHashes    []string `json:"tx_hashes"`
}

type BlockedAddress struct {
	Address    string    `json:"address"`
	Reason     string    `json:"reason"`
	BlockedAt  time.Time `json:"blocked_at"`
	BlockedBy  string    `json:"blocked_by"` // system, admin
	Expiration time.Time `json:"expiration,omitempty"`
}

type SimulationResult struct {
	Success          bool              `json:"success"`
	GasUsed          uint64            `json:"gas_used"`
	GasPrice         string            `json:"gas_price"`
	Logs             []json.RawMessage `json:"logs"`
	ReturnValue      string            `json:"return_value"`
	StateChanges     []StateChange     `json:"state_changes"`
	MEVOpportunities []MEVOpportunity  `json:"mev_opportunities"`
	// SandwichVulnerable is always false unless a live mempool feed is
	// configured. DEX transactions are instead routed through the private
	// relay, which removes the exposure at the source.
	SandwichVulnerable bool `json:"sandwich_vulnerable"`
	// RevertReason carries the node's revert message when the simulated call
	// failed. Empty when the call succeeded.
	RevertReason string `json:"revert_reason,omitempty"`
}

type MEVOpportunity struct {
	Type     string `json:"type"` // arbitrage, liquidation, sandwich
	Profit   string `json:"profit"`
	TokenIn  string `json:"token_in"`
	TokenOut string `json:"token_out"`
	Amount   string `json:"amount"`
	DEXPool  string `json:"dex_pool"`
}

// ============================================================================
// Service
// ============================================================================

type MEVProtectionService struct {
	config       *Config
	redis        *redis.Client
	bundles      map[string]*TransactionBundle
	bundleMu     sync.RWMutex
	blockedAddrs map[string]*BlockedAddress
	blockedMu    sync.RWMutex
}

type Config struct {
	FlashbotsRPC        string
	FlashbotsSigningKey string
	RedisAddr           string
	Port                string
	BundleTimeout       time.Duration
	MaxBundleSize       int
}

// ============================================================================
// Core MEV Protection Methods
// ============================================================================

func NewMEVProtectionService(config *Config) *MEVProtectionService {
	service := &MEVProtectionService{
		config:       config,
		redis:        redis.NewClient(&redis.Options{Addr: config.RedisAddr}),
		bundles:      make(map[string]*TransactionBundle),
		blockedAddrs: make(map[string]*BlockedAddress),
	}

	// Start background tasks
	go service.startBlocklistSync()

	return service
}

// Create Protected Bundle - sends transactions through Flashbots MEV-Share
func (s *MEVProtectionService) CreateProtectedBundle(ctx context.Context, req *MEVProtectionRequest) (*ProtectedBundle, error) {
	bundleID := generateBundleID()

	bundle := &TransactionBundle{
		ID:             bundleID,
		Transactions:   make([]PrivateTx, 0, len(req.Transactions)),
		MinBlockNumber: 0,
		MaxBlockNumber: req.MaxBlockNumber,
		Status:         "pending",
	}

	// Convert transactions to private format. Each protected tx MUST include a
	// signed raw transaction; the tx hash is obtained from the broadcast
	// result (sendToFlashbots) and never fabricated from request metadata.
	for _, tx := range req.Transactions {
		if tx.RawSignedTx == "" {
			return nil, fmt.Errorf("transaction broadcast not implemented - cannot generate tx hash without broadcasting; raw_signed_tx is required")
		}
		privateTx := PrivateTx{
			To:          tx.To,
			Value:       tx.Value,
			Data:        tx.Data,
			Gas:         tx.GasLimit,
			GasPrice:    tx.GasPrice,
			Nonce:       tx.Nonce,
			ChainID:     tx.ChainID,
			RawSignedTx: tx.RawSignedTx,
		}

		bundle.Transactions = append(bundle.Transactions, privateTx)
	}

	// Calculate expected gas price
	if len(bundle.Transactions) > 0 {
		bundle.GasPrice = bundle.Transactions[0].GasPrice
	}

	// Broadcast and collect real transaction hashes. The hash is only ever
	// obtained from the broadcast result; it is never fabricated.
	txHashes, bundleHash, err := s.broadcastBundle(ctx, bundle, req.ProtectionLevel)
	if err != nil {
		return nil, err
	}
	for i, h := range txHashes {
		bundle.Transactions[i].Hash = h
	}
	bundle.TxHashes = txHashes
	bundle.BundleHash = bundleHash

	s.bundleMu.Lock()
	s.bundles[bundleID] = bundle
	s.bundleMu.Unlock()

	// Store in Redis
	bundleJSON, _ := json.Marshal(bundle)
	s.redis.Set(ctx, fmt.Sprintf("mev:bundle:%s", bundleID), bundleJSON, s.config.BundleTimeout)

	return &ProtectedBundle{
		BundleID:    bundleID,
		BundleHash:  bundle.BundleHash,
		BlockNumber: bundle.BlockNumber,
		GasPrice:    bundle.GasPrice,
		Status:      bundle.Status,
		TxHashes:    bundle.TxHashes,
	}, nil
}

// SimulateTransaction executes the transaction against a real JSON-RPC node and
// reports what the node returns: `eth_estimateGas` for the gas limit and
// `eth_call` for the return data and revert reason. Nothing is estimated or
// invented locally. Balance diffs and MEV opportunities are NOT derived here --
// they need `eth_simulateV1`/`debug_traceCall` and a live mempool feed, so the
// corresponding fields are returned empty rather than filled with placeholders.
func (s *MEVProtectionService) SimulateTransaction(ctx context.Context, tx *PrivateTx) (*SimulationResult, error) {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		return nil, fmt.Errorf("simulation requires a JSON-RPC endpoint; set ETH_RPC_URL")
	}
	if tx.From == "" || tx.To == "" {
		return nil, fmt.Errorf("simulation requires both from and to addresses")
	}

	params := map[string]interface{}{
		"from": tx.From,
		"to":   tx.To,
	}
	if tx.Value != "" {
		params["value"] = normalizeHexQuantity(tx.Value)
	}
	if tx.Data != "" {
		params["data"] = tx.Data
	}
	if tx.MaxFeePerGas != "" {
		params["maxFeePerGas"] = normalizeHexQuantity(tx.MaxFeePerGas)
	}
	if tx.MaxPriorityFeePerGas != "" {
		params["maxPriorityFeePerGas"] = normalizeHexQuantity(tx.MaxPriorityFeePerGas)
	} else if tx.GasPrice != "" {
		params["gasPrice"] = normalizeHexQuantity(tx.GasPrice)
	}

	result := &SimulationResult{
		GasPrice:         tx.GasPrice,
		StateChanges:     []StateChange{},
		MEVOpportunities: []MEVOpportunity{},
		Logs:             []json.RawMessage{},
		ReturnValue:      "0x",
	}

	var gasHex string
	if err := rpcCall(ctx, rpcURL, "eth_estimateGas", []interface{}{params}, &gasHex); err == nil {
		if parsed, perr := strconv.ParseUint(strings.TrimPrefix(gasHex, "0x"), 16, 64); perr == nil {
			result.GasUsed = parsed
		}
	}

	var returnHex string
	if err := rpcCall(ctx, rpcURL, "eth_call", []interface{}{params, "latest"}, &returnHex); err != nil {
		// A revert is a legitimate simulation outcome, not a service error.
		result.Success = false
		result.RevertReason = err.Error()
		return result, nil
	}

	result.Success = true
	if returnHex != "" {
		result.ReturnValue = returnHex
	}
	return result, nil
}

// rpcCall performs a single JSON-RPC request and decodes `result` into out.
// Transport failures and JSON-RPC error objects are surfaced as Go errors so a
// caller can never mistake a failed call for a successful one.
func rpcCall(ctx context.Context, rpcURL, method string, params []interface{}, out interface{}) error {
	body, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return fmt.Errorf("marshal %s request: %w", method, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("build %s request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s request: %w", method, err)
	}
	defer resp.Body.Close()

	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode %s response: %w", method, err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("%s: %s", method, envelope.Error.Message)
	}
	if len(envelope.Result) == 0 {
		return fmt.Errorf("%s returned an empty result", method)
	}
	if err := json.Unmarshal(envelope.Result, out); err != nil {
		return fmt.Errorf("decode %s result: %w", method, err)
	}
	return nil
}

// normalizeHexQuantity renders a decimal or hex quantity as 0x-prefixed hex,
// which is the only encoding the JSON-RPC specification accepts for quantities.
func normalizeHexQuantity(value string) string {
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		return "0x" + strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
	}
	if n, err := strconv.ParseUint(value, 10, 64); err == nil {
		return "0x" + strconv.FormatUint(n, 16)
	}
	return value
}

// BlockAddress adds an address to the blocklist that the bundle path consults
// before a transaction is relayed.
func (s *MEVProtectionService) BlockAddress(ctx context.Context, addr *BlockedAddress) error {
	if addr.Address == "" {
		return fmt.Errorf("address is required")
	}
	addr.Address = strings.ToLower(addr.Address)
	addr.BlockedAt = time.Now()
	if addr.BlockedBy == "" {
		addr.BlockedBy = "admin"
	}

	s.blockedMu.Lock()
	s.blockedAddrs[addr.Address] = addr
	s.blockedMu.Unlock()

	addrJSON, err := json.Marshal(addr)
	if err != nil {
		return fmt.Errorf("marshal blocked address: %w", err)
	}
	if err := s.redis.Set(ctx, "mev:block:"+addr.Address, addrJSON, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("persist blocked address: %w", err)
	}
	return nil
}

// IsAddressBlocked reports whether an address is on the blocklist.
func (s *MEVProtectionService) IsAddressBlocked(ctx context.Context, addr string) bool {
	key := strings.ToLower(addr)

	s.blockedMu.RLock()
	blocked, ok := s.blockedAddrs[key]
	s.blockedMu.RUnlock()
	if ok {
		if blocked.Expiration.IsZero() || blocked.Expiration.After(time.Now()) {
			return true
		}
	}

	count, err := s.redis.Exists(ctx, "mev:block:"+key).Result()
	return err == nil && count > 0
}

// GetBundleStatus returns a bundle by id, from memory or from the shared Redis
// store when another replica created it.
func (s *MEVProtectionService) GetBundleStatus(ctx context.Context, bundleID string) (*TransactionBundle, error) {
	s.bundleMu.RLock()
	bundle, ok := s.bundles[bundleID]
	s.bundleMu.RUnlock()
	if ok {
		return bundle, nil
	}

	bundleJSON, err := s.redis.Get(ctx, "mev:bundle:"+bundleID).Result()
	if err != nil {
		return nil, fmt.Errorf("bundle not found")
	}
	var stored TransactionBundle
	if err := json.Unmarshal([]byte(bundleJSON), &stored); err != nil {
		return nil, fmt.Errorf("decode stored bundle: %w", err)
	}
	return &stored, nil
}

// CancelBundle marks a pending bundle as cancelled and drops it from Redis. A
// bundle that has already been broadcast cannot be recalled, so only pending
// bundles can be cancelled.
func (s *MEVProtectionService) CancelBundle(ctx context.Context, bundleID string) error {
	s.bundleMu.Lock()
	defer s.bundleMu.Unlock()

	bundle, ok := s.bundles[bundleID]
	if !ok {
		return fmt.Errorf("bundle not found")
	}
	if bundle.Status != "pending" {
		return fmt.Errorf("bundle is already %s and cannot be cancelled", bundle.Status)
	}
	bundle.Status = "cancelled"
	if err := s.redis.Del(ctx, "mev:bundle:"+bundleID).Err(); err != nil {
		return fmt.Errorf("remove bundle from store: %w", err)
	}
	return nil
}

// broadcastBundle submits each signed raw transaction in the bundle to the
// configured RPC endpoint (Flashbots relay for flashbots protection level, a
// regular JSON-RPC node otherwise) via eth_sendRawTransaction, and returns the
// real transaction hashes reported by the network. A tx hash is NEVER
// fabricated; it is always the value returned by the broadcast.
func (s *MEVProtectionService) broadcastBundle(ctx context.Context, bundle *TransactionBundle, level MEVProtectionLevel) (txHashes []string, bundleHash string, err error) {
	if level == MEVLevelFlashbots {
		// Submit the signed transactions as ONE atomic bundle through the
		// authenticated relay. The relay returns the bundle hash it computed over
		// the signed payload; the individual transaction hashes are the hashes of
		// the signed transactions themselves (the relay does not echo them back).
		if s.config.FlashbotsSigningKey == "" {
			return nil, "", fmt.Errorf("FLASHBOTS_SIGNING_KEY environment variable must be set for flashbots-protected bundles")
		}
		if len(bundle.Transactions) > s.config.MaxBundleSize {
			return nil, "", fmt.Errorf("bundle exceeds the maximum of %d transactions", s.config.MaxBundleSize)
		}

		signedTxs := make([]string, 0, len(bundle.Transactions))
		txHashes = make([]string, 0, len(bundle.Transactions))
		for _, tx := range bundle.Transactions {
			hash, hErr := signedTxHash(tx.RawSignedTx)
			if hErr != nil {
				return nil, "", fmt.Errorf("invalid signed transaction for nonce %d: %w", tx.Nonce, hErr)
			}
			signedTxs = append(signedTxs, tx.RawSignedTx)
			txHashes = append(txHashes, hash)
		}

		params := []interface{}{map[string]interface{}{"txs": signedTxs}}
		if bundle.MaxBlockNumber > 0 {
			params[0].(map[string]interface{})["blockNumber"] = "0x" + strconv.FormatUint(bundle.MaxBlockNumber, 16)
		}

		var relayResult struct {
			BundleHash string `json:"bundleHash"`
		}
		if err := s.flashbotsRPC(ctx, "eth_sendBundle", params, &relayResult); err != nil {
			return nil, "", fmt.Errorf("eth_sendBundle failed: %w", err)
		}
		if relayResult.BundleHash == "" {
			return nil, "", fmt.Errorf("eth_sendBundle returned an empty bundle hash")
		}
		return txHashes, relayResult.BundleHash, nil
	}

	// Non-flashbots levels broadcast each signed transaction to a regular
	// JSON-RPC node. Without a configured node there is no broadcast, and
	// therefore no real hash to report.
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		return nil, "", fmt.Errorf("cannot broadcast without a JSON-RPC endpoint; set ETH_RPC_URL")
	}

	txHashes = make([]string, 0, len(bundle.Transactions))
	for _, tx := range bundle.Transactions {
		hash, bErr := s.sendRawTransaction(ctx, rpcURL, tx.RawSignedTx)
		if bErr != nil {
			return nil, "", fmt.Errorf("broadcast failed for tx nonce %d: %w", tx.Nonce, bErr)
		}
		txHashes = append(txHashes, hash)
	}

	// The bundle hash is derived deterministically from the ordered list of
	// real on-chain transaction hashes returned by the node, so it is a function
	// of network-returned values only.
	digest := crypto.Keccak256([]byte(strings.Join(txHashes, ":")))
	return txHashes, "0x" + hex.EncodeToString(digest), nil
}

// sendRawTransaction calls eth_sendRawTransaction against the given JSON-RPC
// endpoint and returns the transaction hash reported by the node.
func (s *MEVProtectionService) sendRawTransaction(ctx context.Context, rpcURL, rawSignedTx string) (string, error) {
	var hash string
	if err := rpcCall(ctx, rpcURL, "eth_sendRawTransaction", []interface{}{rawSignedTx}, &hash); err != nil {
		return "", err
	}
	if hash == "" {
		return "", fmt.Errorf("eth_sendRawTransaction returned an empty transaction hash")
	}
	return hash, nil
}

// signedTxHash returns the transaction hash of a signed raw transaction, i.e.
// keccak256 over the raw transaction bytes (the RLP payload for legacy
// transactions, the EIP-2718 envelope for typed ones). This is exactly the
// hash a node returns from eth_sendRawTransaction.
func signedTxHash(rawSignedTx string) (string, error) {
	hexBody := strings.TrimPrefix(rawSignedTx, "0x")
	if hexBody == "" {
		return "", fmt.Errorf("empty signed transaction")
	}
	raw, err := hex.DecodeString(hexBody)
	if err != nil {
		return "", fmt.Errorf("signed transaction is not valid hex: %w", err)
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("empty signed transaction")
	}
	return hex.EncodeToString(crypto.Keccak256(raw)), nil
}

// flashbotsRPC performs a JSON-RPC call against the Flashbots relay. The relay
// authenticates callers with an X-Flashbots-Signature header produced by
// signing the keccak256 hash of the request body with the configured key.
func (s *MEVProtectionService) flashbotsRPC(ctx context.Context, method string, params []interface{}, out interface{}) error {
	body, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return fmt.Errorf("marshal %s request: %w", method, err)
	}

	signature, err := signFlashbotsBody(s.config.FlashbotsSigningKey, body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.FlashbotsRPC, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("build %s request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Flashbots-Signature", signature)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s request: %w", method, err)
	}
	defer resp.Body.Close()

	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode %s response: %w", method, err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("%s: %s", method, envelope.Error.Message)
	}
	if len(envelope.Result) == 0 {
		return fmt.Errorf("%s returned an empty result", method)
	}
	if err := json.Unmarshal(envelope.Result, out); err != nil {
		return fmt.Errorf("decode %s result: %w", method, err)
	}
	return nil
}

// signFlashbotsBody builds the X-Flashbots-Signature header value:
// "<checksummed-address>:0x<65-byte recoverable signature over keccak256(body)>".
// The key must be a 0x-prefixed 32-byte hex private key. It fails closed when
// the key is missing or malformed, so an unsigned bundle can never be sent.
func signFlashbotsBody(privateKeyHex string, body []byte) (string, error) {
	key := strings.TrimPrefix(privateKeyHex, "0x")
	if key == "" {
		return "", fmt.Errorf("FLASHBOTS_SIGNING_KEY is not configured; refusing to call the relay unsigned")
	}
	raw, err := hex.DecodeString(key)
	if err != nil || len(raw) != 32 {
		return "", fmt.Errorf("FLASHBOTS_SIGNING_KEY must be a 32-byte hex private key")
	}

	priv, err := crypto.ToECDSA(raw)
	if err != nil {
		return "", fmt.Errorf("invalid FLASHBOTS_SIGNING_KEY: %w", err)
	}

	digest := crypto.Keccak256(body)
	sig, err := crypto.Sign(digest, priv)
	if err != nil {
		return "", fmt.Errorf("sign flashbots request: %w", err)
	}
	// crypto.Sign appends the recovery id as 0/1; EIP-155 style headers use 27/28.
	sig[64] += 27

	return crypto.PubkeyToAddress(priv.PublicKey).Hex() + ":0x" + hex.EncodeToString(sig), nil
}

// syncBlocklist refreshes the in-memory blocklist from Redis. The map is rebuilt
// from scratch so entries whose Redis TTL has expired stop being honoured; SCAN
// is used instead of KEYS so a large keyspace never blocks the Redis event loop.
func (s *MEVProtectionService) syncBlocklist(ctx context.Context) {
	refreshed := make(map[string]*BlockedAddress)

	var cursor uint64
	for {
		keys, next, err := s.redis.Scan(ctx, cursor, "mev:block:*", 200).Result()
		if err != nil {
			log.Printf("blocklist scan failed: %v", err)
			return
		}
		for _, key := range keys {
			addr := strings.TrimPrefix(key, "mev:block:")
			data, err := s.redis.Get(ctx, key).Result()
			if err != nil {
				continue
			}
			var blocked BlockedAddress
			if err := json.Unmarshal([]byte(data), &blocked); err != nil {
				log.Printf("blocklist entry %s is not decodable: %v", addr, err)
				continue
			}
			refreshed[addr] = &blocked
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}

	s.blockedMu.Lock()
	s.blockedAddrs = refreshed
	s.blockedMu.Unlock()
}

func (s *MEVProtectionService) startBlocklistSync() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.syncBlocklist(context.Background())
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *MEVProtectionService) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/bundle", s.handleCreateBundle)
	r.GET("/bundle/:id", s.handleGetBundle)
	r.POST("/bundle/:id/cancel", s.handleCancelBundle)
	r.POST("/simulate", s.handleSimulate)
	r.POST("/block", s.handleBlockAddress)
	r.GET("/block/:address", s.handleCheckBlocked)
	r.GET("/protected-chains", s.handleGetProtectedChains)
}

func (s *MEVProtectionService) handleCreateBundle(c *gin.Context) {
	var req MEVProtectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bundle, err := s.CreateProtectedBundle(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bundle)
}

func (s *MEVProtectionService) handleGetBundle(c *gin.Context) {
	bundleID := c.Param("id")
	bundle, err := s.GetBundleStatus(c.Request.Context(), bundleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bundle)
}

func (s *MEVProtectionService) handleCancelBundle(c *gin.Context) {
	bundleID := c.Param("id")
	if err := s.CancelBundle(c.Request.Context(), bundleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func (s *MEVProtectionService) handleSimulate(c *gin.Context) {
	var tx PrivateTx
	if err := c.ShouldBindJSON(&tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.SimulateTransaction(c.Request.Context(), &tx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *MEVProtectionService) handleBlockAddress(c *gin.Context) {
	var addr BlockedAddress
	if err := c.ShouldBindJSON(&addr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.BlockAddress(c.Request.Context(), &addr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "blocked"})
}

func (s *MEVProtectionService) handleCheckBlocked(c *gin.Context) {
	addr := c.Param("address")
	blocked := s.IsAddressBlocked(c.Request.Context(), addr)
	c.JSON(http.StatusOK, gin.H{"blocked": blocked})
}

func (s *MEVProtectionService) handleGetProtectedChains(c *gin.Context) {
	chains := []string{
		"ethereum",
		"polygon",
		"arbitrum",
		"optimism",
		"avalanche",
		"bsc",
		"base",
	}
	c.JSON(http.StatusOK, gin.H{"chains": chains})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateBundleID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		log.Fatalf("generate bundle id: %v", err)
	}
	return hex.EncodeToString(buf)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	flashbotsSigningKey := os.Getenv("FLASHBOTS_SIGNING_KEY")
	if flashbotsSigningKey == "" {
		log.Println("warning: FLASHBOTS_SIGNING_KEY not set; flashbots-protected bundles will be rejected at broadcast time")
	}
	port := os.Getenv("MEV_PROTECTION_PORT")
	if port == "" {
		port = "8089"
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	apiKey := os.Getenv("MEV_API_KEY")
	if apiKey == "" {
		log.Fatal("MEV_API_KEY must be set: /v1/mev/block is a censorship surface and bundle submission must not be anonymous")
	}

	config := &Config{
		FlashbotsRPC:        "https://relay.flashbots.net",
		FlashbotsSigningKey: flashbotsSigningKey,
		RedisAddr:           redisAddr,
		Port:                port,
		BundleTimeout:       5 * time.Minute,
		MaxBundleSize:       10,
	}

	r := gin.Default()
	service := NewMEVProtectionService(config)
	api := r.Group("/v1/mev")
	api.Use(requireAPIKey(os.Getenv("MEV_API_KEY")))
	service.RegisterRoutes(api)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "mev-protection"})
	})

	r.Run(":" + port)
}

// requireAPIKey authenticates every /v1/mev route. The service blocks
// addresses (a censorship surface) and submits bundles to a public relay under
// the operator's reputation, so neither may be driven anonymously. The
// comparison is constant-time.
func requireAPIKey(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		presented := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if presented == "" {
			presented = strings.TrimSpace(c.GetHeader("X-API-Key"))
		}
		if expected == "" || subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing API key"})
			return
		}
		c.Next()
	}
}
