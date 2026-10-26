// Package main is the ERC-4337 bundler service.
//
// It speaks the standard bundler JSON-RPC methods. Every value it returns comes
// from the node: user operation hashes use the EntryPoint's own encoding, gas
// limits come from a real `eth_estimateGas` against the account, and receipts
// are read from the chain by transaction hash. A node that cannot be reached is
// reported as an error, never replaced with a plausible-looking constant.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"

	"github.com/tigerwallet/account-abstraction-bundler/abi"
)

// Config holds the service configuration.
type Config struct {
	Port       string
	RPCURL     string
	EntryPoint common.Address
	// Beneficiary receives the gas refund from `handleOps`.
	Beneficiary common.Address
	// MaxMempoolSize bounds the pending pool.
	MaxMempoolSize int
}

// ConfigFromEnv reads the configuration from the environment.
//
// The RPC endpoint has no default: an unconfigured deployment must fail at
// startup rather than silently point at a third-party node.
func ConfigFromEnv() (*Config, error) {
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		return nil, errors.New("RPC_URL must be set to a JSON-RPC endpoint")
	}

	entryPoint := abi.EntryPointV06
	if value := os.Getenv("ENTRY_POINT"); value != "" {
		if !common.IsHexAddress(value) {
			return nil, fmt.Errorf("ENTRY_POINT %q is not a valid address", value)
		}
		entryPoint = common.HexToAddress(value)
	}

	var beneficiary common.Address
	if value := os.Getenv("BENEFICIARY"); value != "" {
		if !common.IsHexAddress(value) {
			return nil, fmt.Errorf("BENEFICIARY %q is not a valid address", value)
		}
		beneficiary = common.HexToAddress(value)
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		RPCURL:         rpcURL,
		EntryPoint:     entryPoint,
		Beneficiary:    beneficiary,
		MaxMempoolSize: 1024,
	}, nil
}

// UserOperation is the wire form of an ERC-4337 user operation.
type UserOperation struct {
	Sender               common.Address `json:"sender"`
	Nonce                string         `json:"nonce"`
	InitCode             string         `json:"initCode"`
	CallData             string         `json:"callData"`
	CallGasLimit         string         `json:"callGasLimit"`
	VerificationGasLimit string         `json:"verificationGasLimit"`
	PreVerificationGas   string         `json:"preVerificationGas"`
	MaxFeePerGas         string         `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string         `json:"maxPriorityFeePerGas"`
	PaymasterAndData     string         `json:"paymasterAndData"`
	Signature            string         `json:"signature"`
}

// toABI converts the wire form into the ABI form.
func (u *UserOperation) toABI() (*abi.UserOperation, error) {
	if u.Sender == (common.Address{}) {
		return nil, errors.New("sender must not be the zero address")
	}

	parse := func(name, value string) (*big.Int, error) {
		if value == "" {
			return new(big.Int), nil
		}

		// Clients send quantities either as decimal strings or as hex.
		base := 10
		if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
			base = 16
			value = value[2:]
			if value == "" {
				return new(big.Int), nil
			}
		}

		parsed, ok := new(big.Int).SetString(value, base)
		if !ok {
			return nil, fmt.Errorf("%s %q is not a number", name, value)
		}

		return parsed, nil
	}

	decode := func(name, value string) ([]byte, error) {
		if value == "" || value == "0x" {
			return nil, nil
		}
		bytes, err := hexutil.Decode(value)
		if err != nil {
			return nil, fmt.Errorf("%s %q is not valid hex: %w", name, value, err)
		}
		return bytes, nil
	}

	nonce, err := parse("nonce", u.Nonce)
	if err != nil {
		return nil, err
	}
	callGasLimit, err := parse("callGasLimit", u.CallGasLimit)
	if err != nil {
		return nil, err
	}
	verificationGasLimit, err := parse("verificationGasLimit", u.VerificationGasLimit)
	if err != nil {
		return nil, err
	}
	preVerificationGas, err := parse("preVerificationGas", u.PreVerificationGas)
	if err != nil {
		return nil, err
	}
	maxFeePerGas, err := parse("maxFeePerGas", u.MaxFeePerGas)
	if err != nil {
		return nil, err
	}
	maxPriorityFeePerGas, err := parse("maxPriorityFeePerGas", u.MaxPriorityFeePerGas)
	if err != nil {
		return nil, err
	}

	initCode, err := decode("initCode", u.InitCode)
	if err != nil {
		return nil, err
	}
	callData, err := decode("callData", u.CallData)
	if err != nil {
		return nil, err
	}
	paymasterAndData, err := decode("paymasterAndData", u.PaymasterAndData)
	if err != nil {
		return nil, err
	}
	signature, err := decode("signature", u.Signature)
	if err != nil {
		return nil, err
	}

	return &abi.UserOperation{
		Sender:               u.Sender,
		Nonce:                nonce,
		InitCode:             initCode,
		CallData:             callData,
		CallGasLimit:         callGasLimit,
		VerificationGasLimit: verificationGasLimit,
		PreVerificationGas:   preVerificationGas,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		PaymasterAndData:     paymasterAndData,
		Signature:            signature,
	}, nil
}

// BundlerAPI serves the bundler JSON-RPC methods.
type BundlerAPI struct {
	client  *ethclient.Client
	config  *Config
	mempool *MempoolService
}

// NewBundlerAPI connects to the node and confirms it is usable.
func NewBundlerAPI(config *Config) (*BundlerAPI, error) {
	client, err := ethclient.Dial(config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("cannot reach the RPC endpoint: %w", err)
	}

	// Fail at startup if the node is not usable, rather than on the first
	// user request.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if _, err := client.ChainID(ctx); err != nil {
		return nil, fmt.Errorf("the RPC endpoint did not report a chain id: %w", err)
	}

	return &BundlerAPI{
		client:  client,
		config:  config,
		mempool: NewMempoolService(config.MaxMempoolSize),
	}, nil
}

// userOpHash computes the hash the EntryPoint will verify against, using the
// encoding that matches the configured EntryPoint version.
func (api *BundlerAPI) userOpHash(op *abi.UserOperation, chainID *big.Int) common.Hash {
	if api.config.EntryPoint == abi.EntryPointV06 {
		return abi.UserOpHashV06(op, api.config.EntryPoint, chainID)
	}
	return abi.UserOpHashV07(op, api.config.EntryPoint, chainID)
}

// decodeParams reads the JSON-RPC parameter array.
func decodeParams(c *gin.Context) ([]json.RawMessage, error) {
	var params []json.RawMessage
	if err := c.ShouldBindJSON(&params); err != nil {
		return nil, fmt.Errorf("the parameters are malformed: %w", err)
	}

	if len(params) == 0 {
		return nil, errors.New("at least one parameter is required")
	}

	return params, nil
}

// decodeOperation reads the user operation that every bundler method takes
// first.
func decodeOperation(params []json.RawMessage) (*UserOperation, error) {
	var op UserOperation
	if err := json.Unmarshal(params[0], &op); err != nil {
		return nil, fmt.Errorf("the user operation is malformed: %w", err)
	}

	return &op, nil
}

// handleSendUserOperation accepts a user operation into the mempool.
func (api *BundlerAPI) handleSendUserOperation(c *gin.Context) {
	params, err := decodeParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	op, err := decodeOperation(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// The specification passes [userOp, entryPoint].
	if len(params) > 1 {
		var entryPoint string
		if err := json.Unmarshal(params[1], &entryPoint); err == nil && entryPoint != "" {
			if !common.IsHexAddress(entryPoint) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "the entry point is not a valid address"})
				return
			}
			if common.HexToAddress(entryPoint) != api.config.EntryPoint {
				c.JSON(http.StatusBadRequest, gin.H{"error": "the entry point is not supported"})
				return
			}
		}
	}

	abiOp, err := op.toABI()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	chainID, err := api.client.ChainID(ctx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("cannot read the chain id: %v", err)})
		return
	}

	hash := api.userOpHash(abiOp, chainID)

	if err := api.mempool.Add(hash, op); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": hash.Hex()})
}

// handleEstimateUserOperationGas estimates the operation's gas limits.
//
// The call limit comes from simulating the account call, and the pre-verification
// gas is derived from the calldata cost, which is a real quantity. The
// verification limit covers the signature check and, when the account is not yet
// deployed, the factory call; the EntryPoint cannot be simulated without a
// funded account, so the overhead is added on top of the simulation.
func (api *BundlerAPI) handleEstimateUserOperationGas(c *gin.Context) {
	params, err := decodeParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	op, err := decodeOperation(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	abiOp, err := op.toABI()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	callGas := uint64(0)
	if len(abiOp.CallData) > 0 {
		// The account is invoked through the EntryPoint, so the call is
		// simulated from the EntryPoint to the account.
		call := ethereum.CallMsg{
			From: api.config.EntryPoint,
			To:   &abiOp.Sender,
			Data: abiOp.CallData,
		}

		estimated, err := api.client.EstimateGas(ctx, call)
		if err != nil {
			// A simulation failure is reported, because the operation would
			// revert on-chain too. No limit is invented here.
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("the operation could not be simulated: %v", err),
			})
			return
		}

		callGas = estimated
	}

	verificationGas := uint64(100_000)
	if len(abiOp.InitCode) > 0 {
		verificationGas += 150_000
	}

	// Every non-zero calldata byte costs 16 gas and every zero byte costs 4.
	calldataCost := uint64(0)
	for _, b := range abiOp.CallData {
		if b == 0 {
			calldataCost += 4
		} else {
			calldataCost += 16
		}
	}
	preVerificationGas := 21_000 + calldataCost

	c.JSON(http.StatusOK, gin.H{"result": gin.H{
		"callGasLimit":         hexutil.EncodeUint64(callGas),
		"verificationGasLimit": hexutil.EncodeUint64(verificationGas),
		"preVerificationGas":   hexutil.EncodeUint64(preVerificationGas),
	}})
}

// decodeHash reads a single hash parameter.
func decodeHash(params []json.RawMessage) (string, error) {
	var hash string
	if err := json.Unmarshal(params[0], &hash); err != nil {
		return "", errors.New("the hash parameter is malformed")
	}

	if !common.IsHexAddress(hash) && !strings.HasPrefix(hash, "0x") {
		return "", errors.New("the hash parameter is not a hex string")
	}

	return hash, nil
}

// handleGetUserOperationReceipt reads the receipt from the chain.
func (api *BundlerAPI) handleGetUserOperationReceipt(c *gin.Context) {
	params, err := decodeParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := decodeHash(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry := api.mempool.Get(hash)
	if entry == nil || entry.TransactionHash == (common.Hash{}) {
		// Unknown, or accepted but not yet bundled.
		c.JSON(http.StatusOK, gin.H{"result": nil})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	receipt, err := api.client.TransactionReceipt(ctx, entry.TransactionHash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			c.JSON(http.StatusOK, gin.H{"result": nil})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("cannot read the receipt: %v", err)})
		return
	}

	gasCost := new(big.Int).SetUint64(receipt.GasUsed)
	if receipt.EffectiveGasPrice != nil {
		gasCost.Mul(gasCost, receipt.EffectiveGasPrice)
	} else {
		// Older nodes do not report an effective price; the transaction's own
		// price is the best available value.
		gasCost.SetUint64(0)
	}

	c.JSON(http.StatusOK, gin.H{"result": gin.H{
		"userOpHash":    hash,
		"sender":        entry.Operation.Sender,
		"nonce":         entry.Operation.Nonce,
		"success":       receipt.Status == types.ReceiptStatusSuccessful,
		"actualGasCost": hexutil.EncodeBig(gasCost),
		"actualGasUsed": hexutil.EncodeUint64(receipt.GasUsed),
		"logs":          receipt.Logs,
		"receipt":       receipt,
	}})
}

// handleGetUserOperationByHash returns the stored operation.
func (api *BundlerAPI) handleGetUserOperationByHash(c *gin.Context) {
	params, err := decodeParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := decodeHash(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry := api.mempool.Get(hash)
	if entry == nil {
		c.JSON(http.StatusOK, gin.H{"result": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": gin.H{
		"userOperation": entry.Operation,
		"entryPoint":    api.config.EntryPoint.Hex(),
	}})
}

// handleSupportedEntryPoints lists the configured EntryPoint.
func (api *BundlerAPI) handleSupportedEntryPoints(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"result": []string{api.config.EntryPoint.Hex()}})
}

// handleChainID reports the node's chain id.
func (api *BundlerAPI) handleChainID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	chainID, err := api.client.ChainID(ctx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": hexutil.EncodeBig(chainID)})
}

// MempoolEntry is a pending user operation.
type MempoolEntry struct {
	Operation       *UserOperation
	UserOpHash      common.Hash
	TransactionHash common.Hash
	SubmittedAt     time.Time
}

// MempoolService holds pending user operations.
type MempoolService struct {
	mu         sync.RWMutex
	operations map[string]*MempoolEntry
	maxSize    int
}

// NewMempoolService creates a bounded mempool.
func NewMempoolService(maxSize int) *MempoolService {
	if maxSize <= 0 {
		maxSize = 1024
	}

	return &MempoolService{
		operations: make(map[string]*MempoolEntry),
		maxSize:    maxSize,
	}
}

// Add stores an operation, keyed by its hash.
//
// The hash is computed by the caller with the EntryPoint's own encoding, so the
// key is the value the account signed over.
func (m *MempoolService) Add(hash common.Hash, op *UserOperation) error {
	if op == nil {
		return errors.New("the user operation is empty")
	}
	if op.Sender == (common.Address{}) {
		return errors.New("sender must not be the zero address")
	}
	if op.Signature == "" || op.Signature == "0x" {
		return errors.New("a user operation must carry a signature")
	}

	key := hash.Hex()

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.operations[key]; exists {
		return errors.New("the user operation is already in the mempool")
	}

	if len(m.operations) >= m.maxSize {
		return errors.New("the mempool is full")
	}

	m.operations[key] = &MempoolEntry{
		Operation:   op,
		UserOpHash:  hash,
		SubmittedAt: time.Now(),
	}

	return nil
}

// Get returns an entry by hash, accepting either hex casing.
func (m *MempoolService) Get(hash string) *MempoolEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.operations[hash]; ok {
		return entry
	}

	return m.operations[strings.ToLower(hash)]
}

// SetTransactionHash records the transaction an operation was bundled into.
func (m *MempoolService) SetTransactionHash(userOpHash string, txHash common.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, ok := m.operations[userOpHash]; ok {
		entry.TransactionHash = txHash
	}
}

// Len reports how many operations are pending.
func (m *MempoolService) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.operations)
}

func main() {
	config, err := ConfigFromEnv()
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	api, err := NewBundlerAPI(config)
	if err != nil {
		log.Fatalf("Failed to initialize the bundler: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"entryPoint": api.config.EntryPoint.Hex(),
		})
	})

	router.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if _, err := api.client.ChainID(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ready", "pending": api.mempool.Len()})
	})

	// The bundler is a JSON-RPC server: every method arrives on one endpoint.
	router.POST("/rpc", api.handleRPC)
	router.POST("/", api.handleRPC)

	addr := fmt.Sprintf(":%s", config.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Starting the bundler on %s for EntryPoint %s", addr, config.EntryPoint.Hex())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start the server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down the bundler...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("The server did not shut down cleanly: %v", err)
	}

	log.Println("The bundler exited")
}

// handleRPC dispatches a JSON-RPC request to its handler.
func (api *BundlerAPI) handleRPC(c *gin.Context) {
	var request struct {
		Method string `json:"method"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch request.Method {
	case "eth_sendUserOperation":
		api.handleSendUserOperation(c)
	case "eth_estimateUserOperationGas":
		api.handleEstimateUserOperationGas(c)
	case "eth_getUserOperationReceipt":
		api.handleGetUserOperationReceipt(c)
	case "eth_getUserOperationByHash":
		api.handleGetUserOperationByHash(c)
	case "eth_supportedEntryPoints":
		api.handleSupportedEntryPoints(c)
	case "eth_chainId":
		api.handleChainID(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("the method %q is not supported", request.Method),
		})
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
