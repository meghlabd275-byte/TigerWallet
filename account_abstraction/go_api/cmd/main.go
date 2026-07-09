package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Configuration
type Config struct {
	Port            string
	RPCURL          string
	PrivateKey      string
	EntryPoint      string
	Beneficiary     string
	MaxBundleGas    uint64
	BundleInterval  time.Duration
	MaxBundleSize   int
}

// UserOperation represents an ERC-4337 UserOperation
type UserOperation struct {
	Sender                   common.Address `json:"sender"`
	Nonce                    string         `json:"nonce"`
	InitCode                 string         `json:"initCode"`
	CallData                 string         `json:"callData"`
	CallGasLimit             string         `json:"callGasLimit"`
	VerificationGasLimit     string         `json:"verificationGasLimit"`
	PreVerificationGas       string         `json:"preVerificationGas"`
	MaxFeePerGas             string         `json:"maxFeePerGas"`
	MaxPriorityFeePerGas     string         `json:"maxPriorityFeePerGas"`
	Signature                string         `json:"signature"`
}

// UserOperationReceipt represents the receipt of a user operation
type UserOperationReceipt struct {
	Success           bool      `json:"success"`
	TxHash            string    `json:"txHash"`
	Nonce             string    `json:"nonce"`
	ActualGasCost     string    `json:"actualGasCost"`
	ActualGasUsed     string    `json:"actualGasUsed"`
	Logs              []Log     `json:"logs"`
}

// Log represents an event log
type Log struct {
	Address     common.Address `json:"address"`
	Topics      []string      `json:"topics"`
	Data        string        `json:"data"`
	LogIndex    string        `json:"logIndex"`
	TransactionHash string    `json:"transactionHash"`
}

// BundlerAPI represents the bundler API
type BundlerAPI struct {
	client     *ethclient.Client
	config     *Config
	mempool    *MempoolService
}

// NewBundlerAPI creates a new bundler API
func NewBundlerAPI(config *Config) (*BundlerAPI, error) {
	client, err := ethclient.Dial(config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	return &BundlerAPI{
		client:   client,
		config:   config,
		mempool:  NewMempoolService(),
	}, nil
}

// eth_sendUserOperation handles user operation submission
func (api *BundlerAPI) eth_sendUserOperation(c *gin.Context) {
	var req struct {
		UserOperation UserOperation `json:"userOperation"`
		EntryPoint   string        `json:"entryPoint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate user operation
	if req.UserOperation.Sender == (common.Address{}) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sender"})
		return
	}

	// Add to mempool
	err := api.mempool.Add(req.UserOperation)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Return user operation hash
	hash := api.mempool.GetHash(req.UserOperation)
	c.JSON(http.StatusOK, gin.H{"userOpHash": hash})
}

// eth_estimateUserOperationGas estimates gas for user operation
func (api *BundlerAPI) eth_estimateUserOperationGas(c *gin.Context) {
	var req struct {
		UserOperation UserOperation `json:"userOperation"`
		EntryPoint    string        `json:"entryPoint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Estimate verification gas
	verificationGasLimit := api.estimateVerificationGas(req.UserOperation)

	// Estimate call gas
	callGasLimit := api.estimateCallGas(req.UserOperation)

	// Estimate pre-verification gas
	preVerificationGas := api.estimatePreVerificationGas(req.UserOperation)

	c.JSON(http.StatusOK, gin.H{
		"verificationGasLimit": verificationGasLimit,
		"callGasLimit":        callGasLimit,
		"preVerificationGas":  preVerificationGas,
	})
}

// eth_getUserOperationReceipt gets the receipt of a user operation
func (api *BundlerAPI) eth_getUserOperationReceipt(c *gin.Context) {
	var req struct {
		UserOpHash string `json:"userOpHash"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get receipt from mempool or chain
	receipt := api.mempool.GetReceipt(req.UserOpHash)
	if receipt == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "receipt not found"})
		return
	}

	c.JSON(http.StatusOK, receipt)
}

// eth_getUserOperationByHash gets user operation by hash
func (api *BundlerAPI) eth_getUserOperationByHash(c *gin.Context) {
	var req struct {
		UserOpHash string `json:"userOpHash"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userOp := api.mempool.GetByHash(req.UserOpHash)
	if userOp == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user operation not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"userOperation": userOp,
		"entryPoint":    api.config.EntryPoint,
	})
}

// SupportedEntryPoints returns supported entry points
func (api *BundlerAPI) SupportedEntryPoints(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"supportedEntryPoints": []string{api.config.EntryPoint},
	})
}

// ChainID returns the chain ID
func (api *BundlerAPI) ChainID(c *gin.Context) {
	chainID, err := api.client.ChainID(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chainId": fmt.Sprintf("0x%x", chainID),
	})
}

// estimateVerificationGas estimates verification gas
func (api *BundlerAPI) estimateVerificationGas(op UserOperation) string {
	// Simplified estimation - in production would simulate
	baseVerificationGas := uint64(21000)
	if op.InitCode != "0x" && len(op.InitCode) > 2 {
		baseVerificationGas += 20000 // Account creation
	}
	return fmt.Sprintf("0x%x", baseVerificationGas)
}

// estimateCallGas estimates call gas
func (api *BundlerAPI) estimateCallGas(op UserOperation) string {
	// Simplified estimation - in production would simulate
	return "0x5208" // 21000 gas
}

// estimatePreVerificationGas estimates pre-verification gas
func (api *BundlerAPI) estimatePreVerificationGas(op UserOperation) string {
	// Simplified estimation
	gas := uint64(21000)
	if op.InitCode != "0x" {
		gas += 30000
	}
	return fmt.Sprintf("0x%x", gas)
}

// MempoolService handles the user operation mempool
type MempoolService struct {
	operations map[string]*UserOperation
	receipts  map[string]*UserOperationReceipt
}

func NewMempoolService() *MempoolService {
	return &MempoolService{
		operations: make(map[string]*UserOperation),
		receipts:  make(map[string]*UserOperationReceipt),
	}
}

func (m *MempoolService) Add(op UserOperation) error {
	// Validate operation
	if op.Sender == (common.Address{}) {
		return fmt.Errorf("invalid sender")
	}

	hash := m.GetHash(op)
	m.operations[hash] = &op
	return nil
}

func (m *MempoolService) GetHash(op UserOperation) string {
	// Simplified hash - in production would use proper EIP-4337 hashing
	data := fmt.Sprintf("%s%s%s%s", 
		op.Sender.Hex(), 
		op.Nonce, 
		op.CallData,
		op.Signature)
	
	hash := common.BytesToHash([]byte(data))
	return hash.Hex()
}

func (m *MempoolService) GetReceipt(hash string) *UserOperationReceipt {
	return m.receipts[hash]
}

func (m *MempoolService) GetByHash(hash string) *UserOperation {
	return m.operations[hash]
}

func (m *MempoolService) GetAll() []*UserOperation {
	var ops []*UserOperation
	for _, op := range m.operations {
		ops = append(ops, op)
	}
	return ops
}

// EntryPointABI is the EntryPoint contract ABI
var EntryPointABI = `[{"inputs":[{"name":"ops","type":"tuple[]","components":[{"name":"sender","type":"address"},{"name":"nonce","type":"uint256"},{"name":"initCode","type":"bytes"},{"name":"callData","type":"bytes"},{"name":"callGasLimit","type":"uint256"},{"name":"verificationGasLimit","type":"uint256"},{"name":"preVerificationGas","type":"uint256"},{"name":"maxFeePerGas","type":"uint256"},{"name":"maxPriorityFeePerGas","type":"uint256"},{"name":"paymasterAndData","type":"bytes"},{"name":"signature","type":"bytes"}]},{"name":"beneficiary","type":"address"}],"name":"handleOps","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

func main() {
	// Load configuration
	config := &Config{
		Port:           getEnv("PORT", "8080"),
		RPCURL:         getEnv("RPC_URL", "https://mainnet.infura.io/v3/YOUR_PROJECT_ID"),
		PrivateKey:     getEnv("PRIVATE_KEY", ""),
		EntryPoint:     getEnv("ENTRY_POINT", "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"),
		Beneficiary:    getEnv("BENEFICIARY", ""),
		MaxBundleGas:  5000000,
		BundleInterval: 1 * time.Second,
		MaxBundleSize:  16,
	}

	// Initialize API
	api, err := NewBundlerAPI(config)
	if err != nil {
		log.Fatalf("Failed to initialize API: %v", err)
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ERC-4337 Bundler RPC methods
	rpc := router.Group("/rpc")
	{
		rpc.POST("/", func(c *gin.Context) {
			var rpcReq map[string]interface{}
			if err := c.ShouldBindJSON(&rpcReq); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			method, ok := rpcReq["method"].(string)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing method"})
				return
			}

			switch method {
			case "eth_sendUserOperation":
				api.eth_sendUserOperation(c)
			case "eth_estimateUserOperationGas":
				api.eth_estimateUserOperationGas(c)
			case "eth_getUserOperationReceipt":
				api.eth_getUserOperationReceipt(c)
			case "eth_getUserOperationByHash":
				api.eth_getUserOperationByHash(c)
			case "_supportedEntryPoints":
				api.SupportedEntryPoints(c)
			case "eth_chainId":
				api.ChainID(c)
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported method"})
			}
		})
	}

	// Debug endpoints
	debug := router.Group("/debug")
	{
		debug.GET("/mempool", func(c *gin.Context) {
			ops := api.mempool.GetAll()
			c.JSON(http.StatusOK, gin.H{"operations": ops, "count": len(ops)})
		})
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Starting bundler API on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function to parse hex string
func parseHex(s string) ([]byte, error) {
	s = s[2:] // Remove 0x prefix
	return hex.DecodeString(s)
}

// Helper function to convert to hex
func toHex(b []byte) string {
	return "0x" + hex.EncodeToString(b)
}
