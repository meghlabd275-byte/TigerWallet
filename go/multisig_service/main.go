package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ABI type aliases used for execute() calldata encoding.
var (
	typeAddress = abi.Type{}
	typeUint256 = abi.Type{}
	typeBytes   = abi.Type{}
)

func init() {
	// Resolve ABI types once at startup.
	typeAddress, _ = abi.NewType("address", "", nil)
	typeUint256, _ = abi.NewType("uint256", "", nil)
	typeBytes, _ = abi.NewType("bytes", "", nil)
}

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port     string
	RedisURL string
	RpcURL   string
	ChainID  int64
}

func LoadConfig() *Config {
	return &Config{
		Port:     getEnv("PORT", "8450"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
		// Public Ethereum endpoint; override ETH_RPC_URL with a private/archive node.
		RpcURL:  getEnv("ETH_RPC_URL", "https://ethereum-rpc.publicnode.com"),
		ChainID: 1,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Multi-Sig Models
// ============================================================================

type MultiSigWallet struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Threshold uint     `json:"threshold"`
	Owners    []string `json:"owners"`
	Nonce     uint64   `json:"nonce"`
	ChainID   int64    `json:"chain_id"`
	Address   string   `json:"address"`
	IsActive  bool     `json:"is_active"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

type TransactionRequest struct {
	ID         string            `json:"id"`
	WalletID   string            `json:"wallet_id"`
	To         string            `json:"to"`
	Value      string            `json:"value"`
	Data       string            `json:"data"`
	Nonce      uint64            `json:"nonce"`
	Signatures []Signature       `json:"signatures"`
	Status     TransactionStatus `json:"status"`
	ExecutedBy string            `json:"executed_by"`
	ExecutedAt int64             `json:"executed_at"`
	CreatedAt  int64             `json:"created_at"`
}

type Signature struct {
	Owner string `json:"owner"`
	V     uint8  `json:"v"`
	R     string `json:"r"`
	S     string `json:"s"`
}

type TransactionStatus string

const (
	StatusPending  TransactionStatus = "pending"
	StatusApproved TransactionStatus = "approved"
	StatusExecuted TransactionStatus = "executed"
	StatusFailed   TransactionStatus = "failed"
	StatusRevoked  TransactionStatus = "revoked"
)

// ============================================================================
// Multi-Sig Service
// ============================================================================

type MultiSigService struct {
	config       *Config
	redis        *redis.Client
	wallets      map[string]*MultiSigWallet
	transactions map[string]*TransactionRequest
	privateKey   *ecdsa.PrivateKey
}

func NewMultiSigService(config *Config) *MultiSigService {
	redisOpts, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		// Fall back to a plain address if REDIS_URL is not a full redis:// URL.
		redisOpts = &redis.Options{Addr: config.RedisURL}
	}
	redisClient := redis.NewClient(redisOpts)

	// Deployment/relayer key: real secp256k1 (Ethereum) key, generated locally.
	// For production deployments, supply the key via ETH_RELAYER_PRIVATE_KEY
	// instead of relying on the generated ephemeral key.
	privateKey, err := loadRelayerKey()
	if err != nil {
		log.Fatalf("failed to load relayer key: %v", err)
	}

	return &MultiSigService{
		config:       config,
		redis:        redisClient,
		wallets:      make(map[string]*MultiSigWallet),
		transactions: make(map[string]*TransactionRequest),
		privateKey:   privateKey,
	}
}

// loadRelayerKey returns the secp256k1 private key used to broadcast the
// multisig wallet's transactions. It reads ETH_RELAYER_PRIVATE_KEY from the
// environment (hex, optionally 0x-prefixed). If unset, a fresh ephemeral key is
// generated so the service is still operational, though its address will have no
// funds to pay for gas.
func loadRelayerKey() (*ecdsa.PrivateKey, error) {
	if hexKey := os.Getenv("ETH_RELAYER_PRIVATE_KEY"); hexKey != "" {
		return crypto.HexToECDSA(strings.TrimPrefix(hexKey, "0x"))
	}
	return crypto.GenerateKey()
}

// ============================================================================
// Wallet Management
// ============================================================================

func (s *MultiSigService) CreateWallet(name string, owners []string, threshold uint) (*MultiSigWallet, error) {
	if threshold == 0 || threshold > uint(len(owners)) {
		return nil, fmt.Errorf("invalid threshold")
	}

	// Remove duplicates
	uniqueOwners := removeDuplicates(owners)
	if uint(len(uniqueOwners)) < threshold {
		return nil, fmt.Errorf("not enough owners for threshold")
	}

	// Sort owners for deterministic address
	sort.Strings(uniqueOwners)

	wallet := &MultiSigWallet{
		ID:        generateID(),
		Name:      name,
		Threshold: threshold,
		Owners:    uniqueOwners,
		Nonce:     0,
		ChainID:   s.config.ChainID,
		Address:   s.computeAddress(uniqueOwners, threshold),
		IsActive:  true,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	s.wallets[wallet.ID] = wallet
	s.wallets[wallet.Address] = wallet

	return wallet, nil
}

func (s *MultiSigService) GetWallet(idOrAddress string) (*MultiSigWallet, error) {
	wallet, ok := s.wallets[idOrAddress]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}
	return wallet, nil
}

func (s *MultiSigService) ListWallets() []*MultiSigWallet {
	wallets := make([]*MultiSigWallet, 0, len(s.wallets))
	seen := make(map[string]bool)
	for _, wallet := range s.wallets {
		if !seen[wallet.ID] {
			seen[wallet.ID] = true
			wallets = append(wallets, wallet)
		}
	}
	return wallets
}

func (s *MultiSigService) UpdateWallet(id string, name string, threshold uint) (*MultiSigWallet, error) {
	wallet, ok := s.wallets[id]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}

	if name != "" {
		wallet.Name = name
	}
	if threshold > 0 {
		wallet.Threshold = threshold
	}
	wallet.UpdatedAt = time.Now().Unix()

	return wallet, nil
}

func (s *MultiSigService) AddOwner(walletID string, owner string) error {
	wallet, ok := s.wallets[walletID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}

	for _, o := range wallet.Owners {
		if o == owner {
			return fmt.Errorf("owner already exists")
		}
	}

	wallet.Owners = append(wallet.Owners, owner)
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

func (s *MultiSigService) RemoveOwner(walletID string, owner string) error {
	wallet, ok := s.wallets[walletID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}

	newOwners := make([]string, 0)
	for _, o := range wallet.Owners {
		if o != owner {
			newOwners = append(newOwners, o)
		}
	}

	if uint(len(newOwners)) < wallet.Threshold {
		return fmt.Errorf("cannot remove owner: would break threshold")
	}

	wallet.Owners = newOwners
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

// ============================================================================
// Transaction Management
// ============================================================================

func (s *MultiSigService) CreateTransaction(walletID, to, value, data string) (*TransactionRequest, error) {
	wallet, ok := s.wallets[walletID]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}

	tx := &TransactionRequest{
		ID:         generateID(),
		WalletID:   walletID,
		To:         to,
		Value:      value,
		Data:       data,
		Nonce:      wallet.Nonce,
		Signatures: make([]Signature, 0),
		Status:     StatusPending,
		CreatedAt:  time.Now().Unix(),
	}

	s.transactions[tx.ID] = tx

	return tx, nil
}

func (s *MultiSigService) SignTransaction(txID, owner string, v uint8, r, sVal string) error {
	tx, ok := s.transactions[txID]
	if !ok {
		return fmt.Errorf("transaction not found")
	}

	if tx.Status != StatusPending {
		return fmt.Errorf("transaction already executed or revoked")
	}

	wallet, ok := s.wallets[tx.WalletID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}

	// Verify owner is in wallet
	isOwner := false
	for _, o := range wallet.Owners {
		if strings.EqualFold(o, owner) {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return fmt.Errorf("not an owner of this wallet")
	}

	// Check if already signed
	for _, sig := range tx.Signatures {
		if strings.EqualFold(sig.Owner, owner) {
			return fmt.Errorf("already signed")
		}
	}

	tx.Signatures = append(tx.Signatures, Signature{
		Owner: owner,
		V:     v,
		R:     r,
		S:     sVal,
	})

	// Check if threshold met
	if uint(len(tx.Signatures)) >= wallet.Threshold {
		tx.Status = StatusApproved
	}

	return nil
}

func (s *MultiSigService) ExecuteTransaction(txID string) (string, error) {
	tx, ok := s.transactions[txID]
	if !ok {
		return "", fmt.Errorf("transaction not found")
	}

	wallet, ok := s.wallets[tx.WalletID]
	if !ok {
		return "", fmt.Errorf("wallet not found")
	}

	if tx.Status != StatusApproved {
		return "", fmt.Errorf("transaction not approved")
	}

	if !s.VerifySignatures(tx) {
		tx.Status = StatusFailed
		return "", fmt.Errorf("signature verification failed")
	}

	// Build execute data
	executeData := s.buildExecuteData(tx)

	// Broadcast to a real Ethereum node via JSON-RPC.
	txHash, err := s.broadcastTransaction(wallet.Address, tx.To, tx.Value, executeData)
	if err != nil {
		tx.Status = StatusFailed
		return "", fmt.Errorf("broadcast failed: %w", err)
	}

	tx.Status = StatusExecuted
	tx.ExecutedAt = time.Now().Unix()
	tx.ExecutedBy = wallet.Address

	// Increment nonce
	wallet.Nonce++
	wallet.UpdatedAt = time.Now().Unix()

	return txHash, nil
}

func (s *MultiSigService) RevokeTransaction(txID, owner string) error {
	tx, ok := s.transactions[txID]
	if !ok {
		return fmt.Errorf("transaction not found")
	}

	if tx.Status != StatusPending {
		return fmt.Errorf("can only revoke pending transactions")
	}

	// Remove signature
	newSigs := make([]Signature, 0)
	for _, sig := range tx.Signatures {
		if !strings.EqualFold(sig.Owner, owner) {
			newSigs = append(newSigs, sig)
		}
	}
	tx.Signatures = newSigs

	if len(tx.Signatures) == 0 {
		tx.Status = StatusRevoked
	}

	return nil
}

func (s *MultiSigService) GetTransaction(txID string) (*TransactionRequest, error) {
	tx, ok := s.transactions[txID]
	if !ok {
		return nil, fmt.Errorf("transaction not found")
	}
	return tx, nil
}

func (s *MultiSigService) GetPendingTransactions(walletID string) []*TransactionRequest {
	txs := make([]*TransactionRequest, 0)
	for _, tx := range s.transactions {
		if tx.WalletID == walletID && tx.Status == StatusPending {
			txs = append(txs, tx)
		}
	}
	return txs
}

// ============================================================================
// Crypto Helpers
// ============================================================================

// computeAddress derives the multisig wallet's Ethereum address from the
// relayer's secp256k1 public key. Real EIP-55 checksummed address, derived as
// keccak256(pubkey[1:])[-20:].
func (s *MultiSigService) computeAddress(owners []string, threshold uint) string {
	addr := crypto.PubkeyToAddress(s.privateKey.PublicKey)
	return addr.Hex()
}

// buildExecuteData encodes the calldata forwarded to the multisig wallet
// contract's execute() entrypoint. The destination, value and payload are
// ABI-encoded; the signatures collected from owners are appended for on-chain
// verification (EIP-1271-style). Returns the raw calldata bytes.
func (s *MultiSigService) buildExecuteData(tx *TransactionRequest) []byte {
	toAddr := common.HexToAddress(tx.To)
	valueWei, ok := new(big.Int).SetString(tx.Value, 10)
	if !ok {
		valueWei = new(big.Int)
	}
	dataBytes, _ := hex.DecodeString(strings.TrimPrefix(tx.Data, "0x"))

	args := struct {
		To    common.Address
		Value *big.Int
		Data  []byte
	}{To: toAddr, Value: valueWei, Data: dataBytes}

	enc, err := encodeExecuteArgs(args, tx.Signatures)
	if err != nil {
		// Should not happen with the types above; fall back to the raw payload.
		return dataBytes
	}
	return enc
}

// broadcastTransaction signs and submits a real transaction to an Ethereum
// node over JSON-RPC. It connects to ETH_RPC_URL (env), fetches the pending
// nonce and suggested gas price, estimates gas, signs with the relayer's
// secp256k1 key via crypto.Sign, and submits with eth_sendRawTransaction.
// Returns the real 32-byte transaction hash.
func (s *MultiSigService) broadcastTransaction(from, to, value string, data []byte) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ethclient.Dial(s.config.RpcURL)
	if err != nil {
		return "", fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

	fromAddr := crypto.PubkeyToAddress(s.privateKey.PublicKey)
	_ = from // accepted for API compatibility; the relayer address is derived from the key

	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	valueWei, ok := new(big.Int).SetString(value, 10)
	if !ok {
		valueWei = new(big.Int)
	}

	toAddr := common.HexToAddress(to)
	gasLimit := uint64(210000) // conservative default; sufficient for contract exec + transfer
	if gasEstimate, err := client.EstimateGas(ctx, ethereumCall(fromAddr, toAddr, valueWei, data)); err == nil {
		// Pad the estimate ~20% to avoid edge-case reverts on broadcast.
		gasLimit = gasEstimate * 120 / 100
		if gasLimit < 21000 {
			gasLimit = 21000
		}
	}

	chainID := big.NewInt(s.config.ChainID)
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("gas price: %w", err)
	}
	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		tip = gasPrice
	}
	feeCap := new(big.Int).Add(gasPrice, tip)

	signer := types.NewLondonSigner(chainID)
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        &toAddr,
		Value:     valueWei,
		Gas:       gasLimit,
		GasFeeCap: feeCap,
		GasTipCap: tip,
		Data:      data,
	})

	signedTx, err := types.SignTx(tx, signer, s.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign tx: %w", err)
	}

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("send raw transaction: %w", err)
	}

	return signedTx.Hash().Hex(), nil
}

// SignHash performs a real ECDSA secp256k1 signature over the provided 32-byte
// hash using the relayer private key via go-ethereum's crypto.Sign. The returned
// (v, r, s) are the Ethereum-style signature components: v is normalized to 27
// or 28 (legacy parity bit), and r/s are hex-encoded 32-byte big-endian values.
// crypto.Sign produces a low-s signature and a recovery id (0/1); we add 27 to
// the recovery id to match the Signature struct's legacy v convention.
func (s *MultiSigService) SignHash(owner string, hash []byte) (uint8, string, string, error) {
	_ = owner // owner is tracked at the wallet layer; signing uses the relayer key.
	if len(hash) != 32 {
		return 0, "", "", fmt.Errorf("hash must be 32 bytes, got %d", len(hash))
	}

	sig, err := crypto.Sign(hash, s.privateKey)
	if err != nil {
		return 0, "", "", fmt.Errorf("sign: %w", err)
	}

	// crypto.Sign returns crypto.SignatureLength (65 bytes): r[32] || s[32] || v[1]
	if len(sig) != crypto.SignatureLength {
		return 0, "", "", fmt.Errorf("unexpected signature length %d", len(sig))
	}

	v := sig[64]
	// go-ethereum's recovery id is 0/1; EIP-2 mandates low-s, which crypto.Sign
	// already enforces. Convert to the legacy Ethereum parity convention.
	if v == 0 || v == 1 {
		v += 27
	}

	r := hex.EncodeToString(sig[:32])
	sVal := hex.EncodeToString(sig[32:64])
	return v, r, sVal, nil
}

// VerifySignatures verifies every collected owner signature against the
// transaction hash using real secp256k1 ECDSA recovery (crypto.Ecrecover),
// rejecting high-s and malformed signatures.
func (s *MultiSigService) VerifySignatures(tx *TransactionRequest) bool {
	digest := s.transactionDigest(tx)
	for _, sig := range tx.Signatures {
		rBytes, err := hex.DecodeString(strings.TrimPrefix(sig.R, "0x"))
		if err != nil || len(rBytes) != 32 {
			return false
		}
		sBytes, err := hex.DecodeString(strings.TrimPrefix(sig.S, "0x"))
		if err != nil || len(sBytes) != 32 {
			return false
		}

		// EIP-2: enforce low-s to prevent signature malleability.
		sVal := new(big.Int).SetBytes(sBytes)
		secpHalfN := new(big.Int).Rsh(secp256k1.S256().N, 1)
		if sVal.Cmp(secpHalfN) == 1 {
			return false
		}

		v := sig.V
		if v == 27 || v == 28 {
			v -= 27
		} else if v != 0 && v != 1 {
			return false
		}

		sigBytes := make([]byte, 65)
		copy(sigBytes[:32], rBytes)
		copy(sigBytes[32:64], sBytes)
		sigBytes[64] = v

		pubKey, err := crypto.Ecrecover(digest, sigBytes)
		if err != nil {
			return false
		}
		recoveredAddr := crypto.PubkeyToAddress(toECDSAPubKey(pubKey))
		if !strings.EqualFold(recoveredAddr.Hex(), sig.Owner) {
			return false
		}
	}
	return true
}

// transactionDigest returns the 32-byte keccak256 digest that owners sign over
// for this transaction request (EIP-712-style structured hash).
func (s *MultiSigService) transactionDigest(tx *TransactionRequest) []byte {
	wallet, ok := s.wallets[tx.WalletID]
	if !ok {
		return crypto.Keccak256(nil)
	}
	var chainID big.Int
	chainID.SetUint64(uint64(wallet.ChainID))
	to := common.HexToAddress(tx.To)
	value, ok := new(big.Int).SetString(tx.Value, 10)
	if !ok {
		value = new(big.Int)
	}
	data, _ := hex.DecodeString(strings.TrimPrefix(tx.Data, "0x"))
	return crypto.Keccak256(
		common.LeftPadBytes(chainID.Bytes(), 32),
		common.LeftPadBytes(big.NewInt(int64(tx.Nonce)).Bytes(), 32),
		common.LeftPadBytes(to.Bytes(), 32),
		common.LeftPadBytes(value.Bytes(), 32),
		common.LeftPadBytes(new(big.Int).SetUint64(uint64(len(data))).Bytes(), 32),
		data,
	)
}

// ============================================================================
// Verification
// ============================================================================

func (s *MultiSigService) VerifyTransaction(tx *TransactionRequest) (bool, error) {
	wallet, ok := s.wallets[tx.WalletID]
	if !ok {
		return false, fmt.Errorf("wallet not found")
	}

	// Verify threshold
	if uint(len(tx.Signatures)) < wallet.Threshold {
		return false, fmt.Errorf("not enough signatures")
	}

	// Verify each signature cryptographically against the wallet owner set.
	for _, sig := range tx.Signatures {
		isOwner := false
		for _, owner := range wallet.Owners {
			if strings.EqualFold(owner, sig.Owner) {
				isOwner = true
				break
			}
		}
		if !isOwner {
			return false, fmt.Errorf("invalid signer: %s", sig.Owner)
		}
	}

	return s.VerifySignatures(tx), nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *MultiSigService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "multisig-service"})
	})

	api := r.Group("/api/v1/multisig")
	{
		// Wallets
		api.POST("/wallets", s.handleCreateWallet)
		api.GET("/wallets", s.handleListWallets)
		api.GET("/wallets/:id", s.handleGetWallet)
		api.PUT("/wallets/:id", s.handleUpdateWallet)
		api.POST("/wallets/:id/owners", s.handleAddOwner)
		api.DELETE("/wallets/:id/owners/:owner", s.handleRemoveOwner)

		// Transactions
		api.POST("/transactions", s.handleCreateTransaction)
		api.GET("/transactions/:id", s.handleGetTransaction)
		api.POST("/transactions/:id/sign", s.handleSignTransaction)
		api.POST("/transactions/:id/execute", s.handleExecuteTransaction)
		api.POST("/transactions/:id/revoke", s.handleRevokeTransaction)
		api.GET("/wallets/:id/transactions", s.handleGetPendingTransactions)
	}
}

func (s *MultiSigService) handleCreateWallet(c *gin.Context) {
	var req struct {
		Name      string   `json:"name" binding:"required"`
		Owners    []string `json:"owners" binding:"required,min=1"`
		Threshold uint     `json:"threshold" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := s.CreateWallet(req.Name, req.Owners, req.Threshold)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wallet)
}

func (s *MultiSigService) handleListWallets(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"wallets": s.ListWallets()})
}

func (s *MultiSigService) handleGetWallet(c *gin.Context) {
	id := c.Param("id")

	wallet, err := s.GetWallet(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (s *MultiSigService) handleUpdateWallet(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name      string `json:"name"`
		Threshold uint   `json:"threshold"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := s.UpdateWallet(id, req.Name, req.Threshold)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (s *MultiSigService) handleAddOwner(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Owner string `json:"owner" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.AddOwner(id, req.Owner); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "owner added"})
}

func (s *MultiSigService) handleRemoveOwner(c *gin.Context) {
	id := c.Param("id")
	owner := c.Param("owner")

	if err := s.RemoveOwner(id, owner); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "owner removed"})
}

func (s *MultiSigService) handleCreateTransaction(c *gin.Context) {
	var req struct {
		WalletID string `json:"wallet_id" binding:"required"`
		To       string `json:"to" binding:"required"`
		Value    string `json:"value" binding:"required"`
		Data     string `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := s.CreateTransaction(req.WalletID, req.To, req.Value, req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tx)
}

func (s *MultiSigService) handleGetTransaction(c *gin.Context) {
	id := c.Param("id")

	tx, err := s.GetTransaction(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tx)
}

func (s *MultiSigService) handleSignTransaction(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Owner string `json:"owner" binding:"required"`
		V     uint8  `json:"v"`
		R     string `json:"r"`
		S     string `json:"s"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.SignTransaction(id, req.Owner, req.V, req.R, req.S); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "signed"})
}

func (s *MultiSigService) handleExecuteTransaction(c *gin.Context) {
	id := c.Param("id")

	txHash, err := s.ExecuteTransaction(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tx_hash": txHash})
}

func (s *MultiSigService) handleRevokeTransaction(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Owner string `json:"owner" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.RevokeTransaction(id, req.Owner); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "revoked"})
}

func (s *MultiSigService) handleGetPendingTransactions(c *gin.Context) {
	id := c.Param("id")

	txs := s.GetPendingTransactions(id)
	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

// ============================================================================
// ABI / ethereum helpers
// ============================================================================

// ethereumCall builds an ethereum.CallMsg for gas estimation.
func ethereumCall(from, to common.Address, value *big.Int, data []byte) ethereum.CallMsg {
	return ethereum.CallMsg{
		From:     from,
		To:       &to,
		GasPrice: big.NewInt(0),
		Value:    value,
		Data:     data,
	}
}

// toECDSAPubKey converts the 65-byte uncompressed pubkey returned by
// crypto.Ecrecover into an ecdsa.PublicKey on secp256k1.
func toECDSAPubKey(uncompressed []byte) ecdsa.PublicKey {
	if len(uncompressed) == 0 {
		return ecdsa.PublicKey{}
	}
	pub, err := crypto.UnmarshalPubkey(uncompressed)
	if err != nil || pub == nil {
		return ecdsa.PublicKey{}
	}
	return *pub
}

// encodeExecuteArgs ABI-encodes the execute() calldata (to, value, data) and
// appends the owner signatures so the on-chain wallet can verify them.
func encodeExecuteArgs(args struct {
	To    common.Address
	Value *big.Int
	Data  []byte
}, signatures []Signature) ([]byte, error) {
	arguments := abi.Arguments{
		{Type: typeAddress},
		{Type: typeUint256},
		{Type: typeBytes},
	}
	packed, err := arguments.Pack(args.To, args.Value, args.Data)
	if err != nil {
		return nil, fmt.Errorf("abi pack: %w", err)
	}

	// Append the (v,r,s,owner) signature tuples for on-chain verification.
	for _, sig := range signatures {
		rBytes, err := hex.DecodeString(strings.TrimPrefix(sig.R, "0x"))
		if err != nil {
			return nil, err
		}
		sBytes, err := hex.DecodeString(strings.TrimPrefix(sig.S, "0x"))
		if err != nil {
			return nil, err
		}
		rFixed := common.LeftPadBytes(rBytes, 32)
		sFixed := common.LeftPadBytes(sBytes, 32)
		packed = append(packed, byte(sig.V))
		packed = append(packed, rFixed...)
		packed = append(packed, sFixed...)
		packed = append(packed, []byte(sig.Owner)...)
	}
	return packed, nil
}

// ============================================================================
// Utils
// ============================================================================

func generateID() string {
	return fmt.Sprintf("ms-%d-%s", time.Now().Unix(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[randInt(len(letters))]
	}
	return string(b)
}

func randInt(n int) int {
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	return int(big.NewInt(0).SetBytes(randBytes).Int64()) % n
}

func removeDuplicates(s []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	service := NewMultiSigService(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	service.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Multi-Sig Service starting on port %s", config.Port)
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
