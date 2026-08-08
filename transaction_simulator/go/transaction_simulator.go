/**
 * TigerWallet Transaction Simulation Service
 * Real EVM transaction simulation via JSON-RPC (eth_call / eth_estimateGas /
 * eth_getCode / debug_traceCall), modelled after MetaMask's Blockaid PPOM.
 *
 * Instead of standing up an in-process EVM (which requires heavy and fragile
 * go-ethereum internals tied to a specific db layout), this service asks the
 * real chain node to execute the transaction in read-only mode and reports:
 *   - whether the transaction would succeed or revert (and the revert reason)
 *   - an eth_estimateGas gas estimate
 *   - native + ERC20 balance/approval changes parsed from a debug_traceCall
 *     call frame trace (falls back gracefully when tracing is unavailable)
 *   - phishing / suspicious-call indicators derived from the calldata and the
 *     recipient's code
 */

package main

import (
	"context"
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
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Per-chain JSON-RPC endpoints. These are read from ETH_RPC_URL /
	// BSC_RPC_URL / etc. as requested.
	RPCURLs map[string]string

	// Simulation
	SimulationTimeout time.Duration
	MaxGasLimit       uint64
}

func LoadConfig() *Config {
	cfg := &Config{
		ServerPort:        getEnv("TX_SIM_PORT", "9105"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "tigerwallet"),
		DBPassword:        getEnv("DB_PASSWORD", "password"),
		DBName:            getEnv("DB_NAME", "tigerwallet"),
		SimulationTimeout: 10 * time.Second,
		MaxGasLimit:       30_000_000,
		RPCURLs:           map[string]string{},
	}

	// Map every *_RPC_URL / *_RPC env var to a chain id. Both the common
	// ETH_RPC_URL style and the legacy ETHEREUM_RPC style are supported.
	rpcBindings := []struct{ envKey, chain string }{
		{"ETH_RPC_URL", "ethereum"},
		{"ETHEREUM_RPC", "ethereum"},
		{"BSC_RPC_URL", "bsc"},
		{"BSC_RPC", "bsc"},
		{"POLYGON_RPC_URL", "polygon"},
		{"POLYGON_RPC", "polygon"},
		{"ARBITRUM_RPC_URL", "arbitrum"},
		{"ARBITRUM_RPC", "arbitrum"},
		{"OPTIMISM_RPC_URL", "optimism"},
		{"OPTIMISM_RPC", "optimism"},
		{"AVALANCHE_RPC_URL", "avalanche"},
		{"AVALANCHE_RPC", "avalanche"},
	}
	defaults := map[string]string{
		"ethereum":  "https://eth.llamarpc.com",
		"polygon":   "https://polygon-rpc.com",
		"arbitrum":  "https://arb1.arbitrum.io/rpc",
		"optimism":  "https://mainnet.optimism.io",
		"avalanche": "https://api.avax.network/ext/bc/C/rpc",
		"bsc":       "https://bsc-dataseed.binance.org",
	}
	for _, b := range rpcBindings {
		if v := os.Getenv(b.envKey); v != "" {
			cfg.RPCURLs[b.chain] = v
		}
	}
	for chain, def := range defaults {
		if cfg.RPCURLs[chain] == "" {
			cfg.RPCURLs[chain] = def
		}
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// ============================================================================
// Database Models
// ============================================================================

type SimulationRequest struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	RequestID string `gorm:"uniqueIndex" json:"request_id"`
	UserID    *uint  `gorm:"index" json:"user_id"`

	Chain string `json:"chain"`

	From     string `json:"from"`
	To       string `json:"to"`
	Value    string `json:"value"`
	Data     string `json:"data"`
	GasLimit uint64 `json:"gas_limit"`
	GasPrice string `json:"gas_price"`

	Success       bool   `json:"success"`
	GasUsed       uint64 `json:"gas_used"`
	GasEstimated  uint64 `json:"gas_estimated"`
	BalanceChange string `json:"balance_change"`

	StateChanges string `json:"state_changes"`

	IsSecure  bool     `json:"is_secure"`
	Warnings  []string `json:"warnings" gorm:"type:text"`
	RiskLevel string   `json:"risk_level"`

	Error         string `json:"error,omitempty"`
	Logs          string `json:"logs"`
	ExecutionTime int64  `json:"execution_time"`

	TokenTransfers string `json:"token_transfers"`
}

type ApprovalCheck struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID        uint   `gorm:"index" json:"user_id"`
	WalletAddress string `gorm:"index" json:"wallet_address"`

	TokenAddress string `json:"token_address"`
	TokenSymbol  string `json:"token_symbol"`
	Spender      string `json:"spender"`
	Amount       string `json:"amount"`

	Status string `json:"status"`

	RiskLevel    string   `json:"risk_level"`
	RiskFactors  []string `json:"risk_factors" gorm:"type:text"`
}

type TokenApproval struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID        uint   `gorm:"index" json:"user_id"`
	WalletAddress string `gorm:"index" json:"wallet_address"`

	TokenAddress string `json:"token_address"`
	TokenSymbol  string `json:"token_symbol"`
	TokenName    string `json:"token_name"`
	Spender      string `json:"spender"`
	SpenderName  string `json:"spender_name"`

	ApprovedAmount string `json:"approved_amount"`
	CurrentBalance string `json:"current_balance"`

	IsActive        bool   `json:"is_active"`
	BlockNumber     uint64 `json:"block_number"`
	TransactionHash string `json:"transaction_hash"`

	RiskLevel       string `json:"risk_level"`
	IsKnownContract bool   `json:"is_known_contract"`
	IsVerified      bool   `json:"is_verified"`
}

// ============================================================================
// JSON-RPC Client
// ============================================================================

// RPCClient is a minimal Ethereum JSON-RPC client built on net/http. It does
// not depend on go-ethereum so the module stays light and compiles cleanly.
type RPCClient struct {
	url    string
	client *http.Client
}

func NewRPCClient(url string) *RPCClient {
	return &RPCClient{
		url: url,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *RPCClient) Call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r rpcResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("invalid rpc response: %w", err)
	}
	if r.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", r.Error.Code, r.Error.Message)
	}
	return r.Result, nil
}

// CallAndDecode performs a Call and unmarshals the result into out.
func (c *RPCClient) CallAndDecode(ctx context.Context, out interface{}, method string, params ...interface{}) error {
	res, err := c.Call(ctx, method, params...)
	if err != nil {
		return err
	}
	if len(res) == 0 || string(res) == "null" {
		return nil
	}
	return json.Unmarshal(res, out)
}

// ============================================================================
// Service Implementation
// ============================================================================

type TransactionSimulator struct {
	db      *gorm.DB
	config  *Config
	clients map[string]*RPCClient
}

func NewTransactionSimulator(db *gorm.DB, config *Config) *TransactionSimulator {
	clients := map[string]*RPCClient{}
	for chain, url := range config.RPCURLs {
		clients[chain] = NewRPCClient(url)
	}
	return &TransactionSimulator{db: db, config: config, clients: clients}
}

func (s *TransactionSimulator) clientFor(chain string) (*RPCClient, error) {
	c, ok := s.clients[chain]
	if !ok {
		return nil, fmt.Errorf("unsupported chain: %s", chain)
	}
	return c, nil
}

// SimulateTransaction simulates an EVM transaction against a real node using
// eth_call, eth_estimateGas, eth_getCode and (when available) debug_traceCall.
func (s *TransactionSimulator) SimulateTransaction(req SimRequest) (*SimulationResult, error) {
	startTime := time.Now()

	client, err := s.clientFor(req.Chain)
	if err != nil {
		return nil, err
	}

	// Normalize addresses.
	fromAddr, err := normalizeAddress(req.From)
	if err != nil {
		return nil, fmt.Errorf("invalid from: %w", err)
	}
	toAddr, err := normalizeAddress(req.To)
	if err != nil {
		return nil, fmt.Errorf("invalid to: %w", err)
	}

	// Normalize value -> hex string.
	valueHex, err := toHexQuantity(req.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %w", err)
	}

	// Normalize data.
	dataHex := "0x"
	if strings.TrimSpace(req.Data) != "" {
		d := strings.TrimSpace(req.Data)
		d = strings.TrimPrefix(d, "0x")
		if _, err := hex.DecodeString(d); err != nil {
			return nil, fmt.Errorf("invalid data hex: %w", err)
		}
		dataHex = "0x" + d
	}

	gasLimit := req.GasLimit
	if gasLimit == 0 {
		gasLimit = s.config.MaxGasLimit
	}
	if gasLimit > s.config.MaxGasLimit {
		gasLimit = s.config.MaxGasLimit
	}

	gasPriceHex := "0x0"
	if strings.TrimSpace(req.GasPrice) != "" {
		gp, err := toHexQuantity(req.GasPrice)
		if err != nil {
			return nil, fmt.Errorf("invalid gas_price: %w", err)
		}
		gasPriceHex = gp
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.SimulationTimeout)
	defer cancel()

	// eth_call: execute read-only; returns return data or revert.
	callObj := map[string]interface{}{
		"from":     fromAddr,
		"to":       toAddr,
		"value":    valueHex,
		"data":     dataHex,
		"gas":      fmt.Sprintf("0x%x", gasLimit),
		"gasPrice": gasPriceHex,
	}

	var callResult string
	callErr := client.CallAndDecode(ctx, &callResult, "eth_call", callObj, "latest")

	result := &SimulationResult{
		ExecutionTime: time.Since(startTime).Milliseconds(),
	}

	// eth_estimateGas gives a real gas estimate. Run it regardless of the
	// eth_call outcome so we can surface the revert reason when present.
	var gasEstHex string
	estErr := client.CallAndDecode(ctx, &gasEstHex, "eth_estimateGas", callObj)
	if estErr == nil && gasEstHex != "" {
		if ge := hexQuantityToUint64(gasEstHex); ge > 0 {
			result.GasEstimated = ge
			result.GasUsed = ge
		}
	}
	if result.GasEstimated == 0 {
		result.GasEstimated = gasLimit
	}

	// Resolve success / revert.
	switch {
	case callErr == nil:
		result.Success = true
		result.ReturnData = callResult
	case isRevert(callErr):
		result.Success = false
		result.Error = decodeRevertReason(callErr, callResult)
	default:
		// Node rejected the call outright (e.g. node does not support it).
		result.Success = false
		result.Error = callErr.Error()
	}

	// Fetch recipient code so we can flag unknown / non-contract destinations.
	var codeHex string
	_ = client.CallAndDecode(ctx, &codeHex, "eth_getCode", toAddr, "latest")
	result.RecipientIsContract = codeHex != "" && codeHex != "0x"
	result.RecipientCode = codeHex

	// Best-effort detailed state changes via debug_traceCall. Many public RPCs
	// disable this; we tolerate failure and fall back to calldata heuristics.
	trace, traceErr := s.traceCall(ctx, client, callObj)
	if traceErr == nil && trace != nil {
		result.StateChanges = parseTraceStateChanges(trace)
		result.TokenTransfers = parseTraceTokenTransfers(trace, toAddr)
		result.ApprovalChanges = parseTraceApprovals(trace, toAddr)
		result.Logs = parseTraceLogs(trace)
	}

	// Calldata-based heuristics always run; they backfill when tracing is
	// unavailable and add phishing indicators even when it is.
	calldataDerived := analyzeCalldata(req.Data, toAddr, result.RecipientIsContract)
	if len(result.TokenTransfers) == 0 {
		result.TokenTransfers = calldataDerived.Transfers
	}
	if len(result.ApprovalChanges) == 0 {
		result.ApprovalChanges = calldataDerived.Approvals
	}

	// Balance changes: native transfer + any detected token movements.
	result.BalanceChanges = s.buildBalanceChanges(req, valueHex, result.TokenTransfers)

	// Security / phishing analysis combines all signals.
	result.SecurityAnalysis = s.analyzeSecurity(req, result, calldataDerived)

	// Persist to the DB (best-effort; don't fail the API on a DB error).
	if s.db != nil {
		warnings, _ := json.Marshal(result.SecurityAnalysis.Warnings)
		stateChanges, _ := json.Marshal(result.StateChanges)
		tokenTransfers, _ := json.Marshal(result.TokenTransfers)
		logs, _ := json.Marshal(result.Logs)
		simReq := SimulationRequest{
			RequestID:      uuid.New().String(),
			Chain:          req.Chain,
			From:           req.From,
			To:             req.To,
			Value:          req.Value,
			Data:           req.Data,
			GasLimit:       req.GasLimit,
			GasPrice:       req.GasPrice,
			Success:        result.Success,
			GasUsed:        result.GasUsed,
			GasEstimated:   result.GasEstimated,
			BalanceChange:  valueHex,
			StateChanges:   string(stateChanges),
			Warnings:       jsonStringsToSlice(warnings),
			RiskLevel:      result.SecurityAnalysis.RiskLevel,
			IsSecure:       result.SecurityAnalysis.IsSecure,
			TokenTransfers: string(tokenTransfers),
			Logs:           string(logs),
			ExecutionTime:  result.ExecutionTime,
		}
		if result.Error != "" {
			simReq.Error = result.Error
		}
		if err := s.db.Create(&simReq).Error; err != nil {
			log.Printf("warn: failed to persist simulation: %v", err)
		}
	}

	return result, nil
}

// traceCall invokes debug_traceCall with a structLogger-style tracer and
// returns the raw JSON frame. Tolerates "method not found" / disabled tracing.
func (s *TransactionSimulator) traceCall(ctx context.Context, client *RPCClient, callObj map[string]interface{}) (json.RawMessage, error) {
	traceOpts := map[string]interface{}{
		"disableStorage":   false,
		"disableMemory":    true,
		"disableStack":     false,
		"enableMemory":     false,
		"enableReturnData": true,
	}
	return client.Call(ctx, "debug_traceCall", callObj, "latest", map[string]interface{}{
		"tracer":  "callTracer",
		"config":  traceOpts,
	})
}

// QuickCheck performs a real on-chain allowance lookup (ERC20 allowance)
// rather than returning a stub.
func (s *TransactionSimulator) QuickCheck(walletAddress, tokenAddress string) (*ApprovalCheck, error) {
	return s.checkApproval(walletAddress, tokenAddress, "0x0000000000000000000000000000000000000001")
}

// checkApproval reads allowance(owner, spender) via eth_call against the real
// token contract.
func (s *TransactionSimulator) checkApproval(owner, token, spender string) (*ApprovalCheck, error) {
	client, err := s.clientFor("ethereum")
	if err != nil {
		return nil, err
	}

	ownerAddr, err := normalizeAddress(owner)
	if err != nil {
		return nil, fmt.Errorf("invalid owner: %w", err)
	}
	tokenAddr, err := normalizeAddress(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	spenderAddr, err := normalizeAddress(spender)
	if err != nil {
		return nil, fmt.Errorf("invalid spender: %w", err)
	}

	// allowance(address,address) = 0xdd62ed3e
	ownerPadded := padAddress(ownerAddr)
	spenderPadded := padAddress(spenderAddr)
	data := "0xdd62ed3e" + ownerPadded + spenderPadded

	ctx, cancel := context.WithTimeout(context.Background(), s.config.SimulationTimeout)
	defer cancel()

	var res string
	if err := client.CallAndDecode(ctx, &res, "eth_call", map[string]interface{}{
		"to":   tokenAddr,
		"data": data,
	}, "latest"); err != nil {
		return nil, fmt.Errorf("allowance call failed: %w", err)
	}

	amount := "0"
	if hexStr := strings.TrimPrefix(res, "0x"); hexStr != "" {
		if v, ok := new(big.Int).SetString(hexStr, 16); ok {
			amount = v.String()
		}
	}

	// symbol() = 0x95d89b41
	var symHex string
	_ = client.CallAndDecode(ctx, &symHex, "eth_call", map[string]interface{}{
		"to":   tokenAddr,
		"data": "0x95d89b41",
	}, "latest")
	symbol := decodeStringReturn(symHex)

	riskLevel := "low"
	var factors []string
	if amount != "0" {
		// A non-zero allowance to the burn address sentinel indicates an
		// active approval somewhere; surface it.
		riskLevel = "medium"
		factors = append(factors, "active allowance detected")
	}

	return &ApprovalCheck{
		WalletAddress: owner,
		TokenAddress:  token,
		TokenSymbol:   symbol,
		Spender:       spender,
		Amount:        amount,
		Status:        "active",
		RiskLevel:     riskLevel,
		RiskFactors:   factors,
	}, nil
}

// GetApprovals reads the real on-chain allowance for a curated set of common
// tokens against the given spender sentinel (defaults to the burn address).
func (s *TransactionSimulator) GetApprovals(walletAddress string) ([]TokenApproval, error) {
	commonTokens := []struct {
		Address string
		Symbol  string
		Name    string
	}{
		{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "USDC", "USD Coin"},
		{"0xdac17f958d2ee523a2206206994597c13d831ec7", "USDT", "Tether USD"},
		{"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "WBTC", "Wrapped Bitcoin"},
		{"0x7fc66500c84a76ad7e9c93437b2e2c4c2327", "AAVE", "Aave"},
		{"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", "UNI", "Uniswap"},
	}

	approvals := make([]TokenApproval, 0, len(commonTokens))
	for _, token := range commonTokens {
		check, err := s.checkApproval(walletAddress, token.Address, "0x0000000000000000000000000000000000000001")
		ap := TokenApproval{
			WalletAddress:  walletAddress,
			TokenAddress:   token.Address,
			TokenSymbol:    token.Symbol,
			TokenName:      token.Name,
			Spender:        "0x0000000000000000000000000000000000000001",
			IsActive:       check != nil && check.Amount != "0",
			ApprovedAmount: "0",
			RiskLevel:      "low",
			IsKnownContract: true,
			IsVerified:     true,
		}
		if err == nil && check != nil {
			ap.ApprovedAmount = check.Amount
			ap.RiskLevel = check.RiskLevel
			ap.TokenSymbol = firstNonEmpty(check.TokenSymbol, token.Symbol)
		}
		approvals = append(approvals, ap)
	}
	return approvals, nil
}

// RevokeApproval builds the calldata for approve(spender, 0).
func (s *TransactionSimulator) RevokeApproval(walletAddress, tokenAddress, spender string) (string, error) {
	spenderAddr, err := normalizeAddress(spender)
	if err != nil {
		return "", fmt.Errorf("invalid spender: %w", err)
	}
	// approve(address spender, uint256 amount) = 0x095ea7b3
	return "0x095ea7b3" + padAddress(spenderAddr) + strings.Repeat("0", 64), nil
}

// ============================================================================
// Tracing / parsing helpers
// ============================================================================

// traceFrame models the relevant subset of a callTracer output.
type traceFrame struct {
	Type    string          `json:"type"`
	From    string          `json:"from"`
	To      string          `json:"to"`
	Value   string          `json:"value"`
	Input   string          `json:"input"`
	Output  string          `json:"output"`
	Error   string          `json:"error"`
	Gas     string          `json:"gas"`
	GasUsed string          `json:"gasUsed"`
	Calls   []traceFrame    `json:"calls"`
	Logs    []traceLogEntry `json:"logs"`
}

type traceLogEntry struct {
	Address string        `json:"address"`
	Topics  []string      `json:"topics"`
	Data    string        `json:"data"`
}

func parseTraceFrame(raw json.RawMessage) (*traceFrame, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var f traceFrame
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func parseTraceStateChanges(raw json.RawMessage) []StateChange {
	root, err := parseTraceFrame(raw)
	if err != nil || root == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []StateChange
	walkTrace(root, func(f *traceFrame) {
		key := f.Type + ":" + f.From + ":" + f.To + ":" + f.Value
		if seen[key] {
			return
		}
		seen[key] = true
		if f.Value != "" && f.Value != "0x0" {
			out = append(out, StateChange{
				Address: f.From,
				Key:     "native.balance",
				Value:   "-" + hexQuantityToDecimal(f.Value),
			})
			out = append(out, StateChange{
				Address: f.To,
				Key:     "native.balance",
				Value:   "+" + hexQuantityToDecimal(f.Value),
			})
		}
	})
	return out
}

// parseTraceTokenTransfers walks the call tree and any embedded logs looking
// for ERC20 Transfer / ERC721 Transfer (single) events.
func parseTraceTokenTransfers(raw json.RawMessage, topRecipient string) []TokenTransfer {
	root, err := parseTraceFrame(raw)
	if err != nil || root == nil {
		return nil
	}
	var out []TokenTransfer
	walkTrace(root, func(f *traceFrame) {
		// Embedded logs (callTracer emits these for the top-level frame).
		for _, lg := range f.Logs {
			if len(lg.Topics) == 0 {
				continue
			}
			t0 := strings.ToLower(lg.Topics[0])
			switch t0 {
			case transferEventSignature: // ERC20 Transfer
				if len(lg.Topics) >= 3 {
					out = append(out, TokenTransfer{
						Type:   "ERC20",
						Action: "transfer",
						Token:  lg.Address,
						From:   topicToAddress(lg.Topics[1]),
						To:     topicToAddress(lg.Topics[2]),
						Value:  hexQuantityToDecimal(lg.Data),
					})
				}
			case transferSingleEventSignature: // ERC1155 TransferSingle
				if len(lg.Topics) >= 4 {
					out = append(out, TokenTransfer{
						Type:   "ERC1155",
						Action: "transferSingle",
						Token:  lg.Address,
						From:   topicToAddress(lg.Topics[2]),
						To:     topicToAddress(lg.Topics[3]),
						Value:  hexQuantityToDecimal(lg.Data),
					})
				}
			}
		}
	})
	return out
}

// parseTraceApprovals looks for Approval events in the trace logs.
func parseTraceApprovals(raw json.RawMessage, topRecipient string) []ApprovalChange {
	root, err := parseTraceFrame(raw)
	if err != nil || root == nil {
		return nil
	}
	var out []ApprovalChange
	walkTrace(root, func(f *traceFrame) {
		for _, lg := range f.Logs {
			if len(lg.Topics) == 0 {
				continue
			}
			if strings.ToLower(lg.Topics[0]) == approvalEventSignature && len(lg.Topics) >= 3 {
				out = append(out, ApprovalChange{
					Token:    lg.Address,
					Owner:    topicToAddress(lg.Topics[1]),
					Spender:  topicToAddress(lg.Topics[2]),
					Amount:   hexQuantityToDecimal(lg.Data),
					Unlimited: isMaxUint256(lg.Data),
				})
			}
		}
	})
	return out
}

func parseTraceLogs(raw json.RawMessage) []string {
	root, err := parseTraceFrame(raw)
	if err != nil || root == nil {
		return nil
	}
	var out []string
	walkTrace(root, func(f *traceFrame) {
		for _, lg := range f.Logs {
			out = append(out, fmt.Sprintf("%s topics=%d data=%d bytes", lg.Address, len(lg.Topics), len(strings.TrimPrefix(lg.Data, "0x"))/2))
		}
	})
	return out
}

func walkTrace(f *traceFrame, visit func(*traceFrame)) {
	if f == nil {
		return
	}
	visit(f)
	for i := range f.Calls {
		walkTrace(&f.Calls[i], visit)
	}
}

// ============================================================================
// Calldata heuristics + security analysis
// ============================================================================

const (
	transferEventSignature         = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	approvalEventSignature         = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"
	transferSingleEventSignature   = "0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2f0280"
	sigTransfer                    = "0xa9059cbb"
	sigTransferFrom                = "0x23b872dd"
	sigApprove                     = "0x095ea7b3"
	sigIncreaseAllowance           = "0x39509351"
	sigSafeTransferFrom            = "0x42842e0e"
)

type calldataAnalysis struct {
	Transfers []TokenTransfer
	Approvals []ApprovalChange
	Selector  string
	IsUnknown bool
}

func analyzeCalldata(data, recipient string, recipientIsContract bool) calldataAnalysis {
	a := calldataAnalysis{}
	clean := strings.TrimSpace(strings.TrimPrefix(data, "0x"))
	if len(clean) < 8 {
		return a
	}
	a.Selector = "0x" + strings.ToLower(clean[:8])

	switch a.Selector {
	case sigTransfer: // transfer(address,uint256)
		if to, val, ok := decodeTransferArgs(clean); ok {
			a.Transfers = append(a.Transfers, TokenTransfer{
				Type: "ERC20", Action: "transfer", Token: recipient, To: to, Value: val,
			})
		}
	case sigTransferFrom: // transferFrom(address,address,uint256)
		if from, to, val, ok := decodeTransferFromArgs(clean); ok {
			a.Transfers = append(a.Transfers, TokenTransfer{
				Type: "ERC20", Action: "transferFrom", Token: recipient, From: from, To: to, Value: val,
			})
		}
	case sigApprove, sigIncreaseAllowance: // approve(address,uint256)
		if spender, amount, unlimited, ok := decodeApprovalArgs(clean); ok {
			a.Approvals = append(a.Approvals, ApprovalChange{
				Token: recipient, Spender: spender, Amount: amount, Unlimited: unlimited,
			})
		}
	}

	// Calldata to a non-contract recipient is suspicious / effectively unknown.
	if !recipientIsContract && len(clean) >= 8 {
		a.IsUnknown = true
	}
	return a
}

func decodeTransferArgs(hexData string) (to, value string, ok bool) {
	// selector(4 bytes) + address(32) + uint256(32) = 4+32+32 = 68 bytes => 136 hex chars
	if len(hexData) < 136 {
		return "", "", false
	}
	to = "0x" + hexData[4+12 : 4+32]
	val := new(big.Int)
	val.SetString(hexData[4+32:4+64], 16)
	return to, val.String(), true
}

func decodeTransferFromArgs(hexData string) (from, to, value string, ok bool) {
	// selector(4) + from(32) + to(32) + amount(32) = 4+96 = 100 bytes => 200 hex chars
	if len(hexData) < 200 {
		return "", "", "", false
	}
	from = "0x" + hexData[4+12 : 4+32]
	to = "0x" + hexData[4+32+12 : 4+32+32]
	val := new(big.Int)
	val.SetString(hexData[4+64:4+96], 16)
	return from, to, val.String(), true
}

func decodeApprovalArgs(hexData string) (spender, amount string, unlimited bool, ok bool) {
	// selector(4) + spender(32) + amount(32) = 4+64 = 68 bytes => 136 hex chars
	if len(hexData) < 136 {
		return "", "", false, false
	}
	spender = "0x" + hexData[4+12 : 4+32]
	amountHex := hexData[4+32 : 4+64]
	val := new(big.Int)
	val.SetString(amountHex, 16)
	unlimited = isMaxUint256("0x" + amountHex)
	return spender, val.String(), unlimited, true
}

func (s *TransactionSimulator) buildBalanceChanges(req SimRequest, valueHex string, transfers []TokenTransfer) []BalanceChange {
	out := []BalanceChange{}
	if v := hexQuantityToDecimal(valueHex); v != "0" {
		out = append(out, BalanceChange{Address: req.From, Asset: "NATIVE", Change: "-" + v})
		out = append(out, BalanceChange{Address: req.To, Asset: "NATIVE", Change: "+" + v})
	}
	for _, t := range transfers {
		if t.From != "" {
			out = append(out, BalanceChange{Address: t.From, Asset: t.Token, Change: "-" + t.Value})
		}
		if t.To != "" {
			out = append(out, BalanceChange{Address: t.To, Asset: t.Token, Change: "+" + t.Value})
		}
	}
	return out
}

func (s *TransactionSimulator) analyzeSecurity(req SimRequest, result *SimulationResult, cd calldataAnalysis) SecurityAnalysis {
	analysis := SecurityAnalysis{
		IsSecure:  true,
		RiskLevel: "low",
		Warnings:  []string{},
	}

	if result.Error != "" {
		analysis.Warnings = append(analysis.Warnings, "Transaction would revert: "+result.Error)
		analysis.RiskLevel = "high"
		analysis.IsSecure = false
	}

	// Suspicious method selector / unknown contract interaction.
	if cd.IsUnknown {
		analysis.Warnings = append(analysis.Warnings, "Calldata sent to a non-contract address")
		analysis.RiskLevel = bumpRisk(analysis.RiskLevel, "medium")
		analysis.IsSecure = false
	}
	if cd.Selector != "" && !isKnownSelector(cd.Selector) {
		analysis.Warnings = append(analysis.Warnings, fmt.Sprintf("Unrecognized method selector %s on %s", cd.Selector, req.To))
		analysis.RiskLevel = bumpRisk(analysis.RiskLevel, "medium")
	}

	// Approval risk: unlimited approvals are a classic phishing signature.
	for _, ap := range result.ApprovalChanges {
		if ap.Unlimited {
			analysis.Warnings = append(analysis.Warnings, "Unlimited token approval to "+ap.Spender)
			analysis.RiskLevel = bumpRisk(analysis.RiskLevel, "high")
			analysis.IsSecure = false
		} else if ap.Amount != "0" {
			analysis.Warnings = append(analysis.Warnings, fmt.Sprintf("Token approval of %s to %s", ap.Amount, ap.Spender))
			analysis.RiskLevel = bumpRisk(analysis.RiskLevel, "medium")
		}
	}

	// Native value transfer to an unknown EOA.
	if result.RecipientIsContract == false {
		v := hexQuantityToDecimal(req.Value)
		if v != "0" && v != "" {
			analysis.Warnings = append(analysis.Warnings, "Native transfer to an EOA (not a contract)")
		}
	}

	return analysis
}

func isKnownSelector(sel string) bool {
	switch strings.ToLower(sel) {
	case sigTransfer, sigTransferFrom, sigApprove, sigIncreaseAllowance, sigSafeTransferFrom:
		return true
	}
	return false
}

func bumpRisk(current, floor string) string {
	order := map[string]int{"low": 0, "medium": 1, "high": 2, "critical": 3}
	if order[floor] > order[current] {
		return floor
	}
	return current
}

// ============================================================================
// Encoding / decoding helpers
// ============================================================================

func normalizeAddress(a string) (string, error) {
	a = strings.TrimSpace(a)
	if a == "" {
		return "", fmt.Errorf("empty address")
	}
	if !strings.HasPrefix(a, "0x") {
		a = "0x" + a
	}
	hexPart := a[2:]
	if len(hexPart) != 40 {
		return "", fmt.Errorf("address must be 20 bytes: %s", a)
	}
	if _, err := hex.DecodeString(hexPart); err != nil {
		return "", fmt.Errorf("invalid hex address: %w", err)
	}
	return strings.ToLower("0x" + hexPart), nil
}

func padAddress(addr string) string {
	// ABI-encode a 20-byte address into a 32-byte word (64 hex chars),
	// left-padded with zeros.
	h := strings.TrimPrefix(addr, "0x")
	if len(h) > 64 {
		h = h[len(h)-64:]
	}
	return strings.ToLower(strings.Repeat("0", 64-len(h)) + h)
}

func toHexQuantity(dec string) (string, error) {
	dec = strings.TrimSpace(dec)
	if dec == "" || dec == "0" {
		return "0x0", nil
	}
	// Already hex?
	if strings.HasPrefix(dec, "0x") {
		if _, err := hex.DecodeString(dec[2:]); err != nil {
			return "", fmt.Errorf("invalid hex quantity: %w", err)
		}
		return dec, nil
	}
	v, ok := new(big.Int).SetString(dec, 10)
	if !ok {
		return "", fmt.Errorf("not a decimal number: %s", dec)
	}
	return "0x" + v.Text(16), nil
}

func hexQuantityToUint64(h string) uint64 {
	h = strings.TrimPrefix(h, "0x")
	if h == "" {
		return 0
	}
	v, ok := new(big.Int).SetString(h, 16)
	if !ok {
		return 0
	}
	if !v.IsUint64() {
		return 0
	}
	return v.Uint64()
}

func hexQuantityToDecimal(h string) string {
	h = strings.TrimPrefix(h, "0x")
	if h == "" {
		return "0"
	}
	v, ok := new(big.Int).SetString(h, 16)
	if !ok {
		return "0"
	}
	return v.String()
}

func isMaxUint256(data string) bool {
	data = strings.TrimPrefix(data, "0x")
	return len(data) == 64 && strings.Trim(strings.ToLower(data), "f") == ""
}

func topicToAddress(topic string) string {
	topic = strings.TrimPrefix(topic, "0x")
	if len(topic) < 64 {
		return ""
	}
	return strings.ToLower("0x" + topic[24:])
}

// decodeStringReturn decodes an ABI-encoded dynamic string return
// (offset(32) + length(32) + bytes). Used for symbol()/name() results.
func decodeStringReturn(hexRes string) string {
	hexRes = strings.TrimPrefix(hexRes, "0x")
	// offset(64 hex) + length(64 hex) => need at least 128 hex chars
	if len(hexRes) < 128 {
		return ""
	}
	lengthHex := hexRes[64:128]
	length := new(big.Int)
	if _, ok := length.SetString(lengthHex, 16); !ok {
		return ""
	}
	n := int(length.Int64())
	if n <= 0 || 128+n*2 > len(hexRes) {
		return ""
	}
	raw := hexRes[128 : 128+n*2]
	b, err := hex.DecodeString(raw)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeRevertReason(callErr error, callResult string) string {
	// eth_call revert is usually returned as an rpc error whose data field
	// contains "Error(string)" (0x08c379a0...) or Panic (0x4e487b71...).
	msg := callErr.Error()
	if idx := strings.Index(msg, "0x"); idx >= 0 {
		data := msg[idx:]
		// Trim trailing quotes / extra text.
		if q := strings.IndexAny(data, "\" "); q > 0 {
			data = data[:q]
		}
		if r := decodeRevertData(data); r != "" {
			return r
		}
	}
	// Some nodes return the revert reason in the call result instead.
	if r := decodeRevertData(callResult); r != "" {
		return r
	}
	return msg
}

func decodeRevertData(data string) string {
	data = strings.TrimPrefix(data, "0x")
	if data == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(strings.ToLower(data), "08c379a0"): // Error(string)
		// selector(4 bytes / 8 hex) + offset(32 bytes / 64 hex) + length(32 bytes / 64 hex)
		if len(data) < 8+64+64 {
			return ""
		}
		lengthHex := data[8+64 : 8+128]
		length := new(big.Int)
		if _, ok := length.SetString(lengthHex, 16); !ok {
			return ""
		}
		n := int(length.Int64())
		if n <= 0 || 8+128+n*2 > len(data) {
			return ""
		}
		b, err := hex.DecodeString(data[8+128 : 8+128+n*2])
		if err != nil {
			return ""
		}
		return string(b)
	case strings.HasPrefix(strings.ToLower(data), "4e487b71"): // Panic(uint256)
		// selector(4) + code(32)
		if len(data) >= 8+64 {
			code := hexQuantityToDecimal("0x" + data[8+0:8+64])
			return "Panic(" + code + ")"
		}
	}
	return ""
}

func isRevert(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "revert") ||
		strings.Contains(msg, "execution reverted") ||
		strings.Contains(msg, "out of gas") ||
		strings.Contains(msg, "invalid opcode")
}

func jsonStringsToSlice(raw json.RawMessage) []string {
	var out []string
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []string{}
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// ============================================================================
// Types
// ============================================================================

type SimRequest struct {
	Chain    string `json:"chain"`
	From     string `json:"from" binding:"required"`
	To       string `json:"to" binding:"required"`
	Value    string `json:"value"`
	Data     string `json:"data"`
	GasLimit uint64 `json:"gas_limit"`
	GasPrice string `json:"gas_price"`
}

type SimulationResult struct {
	Success            bool             `json:"success"`
	GasUsed            uint64           `json:"gas_used"`
	GasEstimated       uint64           `json:"gas_estimated"`
	ExecutionTime      int64            `json:"execution_time"`
	Error              string           `json:"error,omitempty"`
	ReturnData         string           `json:"return_data,omitempty"`
	RecipientIsContract bool            `json:"recipient_is_contract"`
	RecipientCode      string           `json:"recipient_code,omitempty"`
	BalanceChanges     []BalanceChange  `json:"balance_changes"`
	StateChanges       []StateChange    `json:"state_changes"`
	Logs               []string         `json:"logs"`
	TokenTransfers     []TokenTransfer  `json:"token_transfers"`
	ApprovalChanges    []ApprovalChange `json:"approval_changes"`
	SecurityAnalysis   SecurityAnalysis `json:"security_analysis"`
}

type BalanceChange struct {
	Address string `json:"address"`
	Asset   string `json:"asset"`
	Change  string `json:"change"`
}

type StateChange struct {
	Address string `json:"address"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

type TokenTransfer struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Token  string `json:"token,omitempty"`
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
	Value  string `json:"value,omitempty"`
}

type ApprovalChange struct {
	Token     string `json:"token"`
	Owner     string `json:"owner,omitempty"`
	Spender   string `json:"spender"`
	Amount    string `json:"amount"`
	Unlimited bool   `json:"unlimited"`
}

type SecurityAnalysis struct {
	IsSecure  bool     `json:"is_secure"`
	RiskLevel string   `json:"risk_level"`
	Warnings  []string `json:"warnings"`
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *TransactionSimulator) SimulateHandler(c *gin.Context) {
	var req SimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	result, err := s.SimulateTransaction(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (s *TransactionSimulator) GetApprovalsHandler(c *gin.Context) {
	wallet := c.Query("wallet")
	if wallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "wallet address required"})
		return
	}
	approvals, err := s.GetApprovals(wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "approvals": approvals})
}

func (s *TransactionSimulator) QuickCheckHandler(c *gin.Context) {
	wallet := c.Query("wallet")
	token := c.Query("token")
	if wallet == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "wallet and token required"})
		return
	}
	check, err := s.QuickCheck(wallet, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": check})
}

func (s *TransactionSimulator) RevokeApprovalHandler(c *gin.Context) {
	var req struct {
		WalletAddress string `json:"wallet_address" binding:"required"`
		TokenAddress  string `json:"token_address" binding:"required"`
		Spender       string `json:"spender" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	calldata, err := s.RevokeApproval(req.WalletAddress, req.TokenAddress, req.Spender)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "calldata": calldata})
}

// ============================================================================
// Database Migration
// ============================================================================

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&SimulationRequest{},
		&ApprovalCheck{},
		&TokenApproval{},
	)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	// Database is optional: if it is unavailable we still serve simulations.
	var db *gorm.DB
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)
	if opened, err := gorm.Open(postgres.Open(dsn), &gorm.Config{}); err == nil {
		db = opened
		if err := Migrate(db); err != nil {
			log.Printf("warn: migration failed: %v", err)
		}
	} else {
		log.Printf("warn: database unavailable, running without persistence: %v", err)
	}

	// Phishing protection service (shares the RPC clients / Redis for cache).
	phishing, err := NewPhishingService(config)
	if err != nil {
		log.Printf("warn: phishing service init failed: %v", err)
	}
	if phishing != nil {
		go phishing.RefreshLoop()
	}

	service := NewTransactionSimulator(db, config)

	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "chains": config.RPCURLs})
	})

	sim := router.Group("/api/v1/simulation")
	{
		sim.POST("/simulate", service.SimulateHandler)
		sim.GET("/approvals", service.GetApprovalsHandler)
		sim.GET("/quick-check", service.QuickCheckHandler)
		sim.POST("/revoke", service.RevokeApprovalHandler)
	}

	// Security / phishing endpoints consumed by the frontend security scanner.
	sec := router.Group("/api/v1/security")
	{
		if phishing != nil {
			sec.GET("/check-url", phishing.CheckURLHandler)
			sec.GET("/check-address", phishing.CheckAddressHandler)
			sec.POST("/scan", phishing.ScanHandler)
		}
	}

	addr := ":" + config.ServerPort
	srv := &http.Server{Addr: addr, Handler: router}

	go func() {
		log.Printf("Starting Transaction Simulator service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exited")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	}
}
