package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/sha3"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port      string
	RedisURL  string
	ChainID   int64
}

func LoadConfig() *Config {
	return &Config{
		Port:     getEnv("PORT", "8450"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
		ChainID:  1,
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
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Threshold       uint     `json:"threshold"`
	Owners          []string `json:"owners"`
	Nonce           uint64   `json:"nonce"`
	ChainID         int64    `json:"chain_id"`
	Address         string   `json:"address"`
	IsActive        bool     `json:"is_active"`
	CreatedAt       int64    `json:"created_at"`
	UpdatedAt       int64    `json:"updated_at"`
}

type TransactionRequest struct {
	ID          string   `json:"id"`
	WalletID    string   `json:"wallet_id"`
	To          string   `json:"to"`
	Value       string   `json:"value"`
	Data        string   `json:"data"`
	Nonce       uint64   `json:"nonce"`
	Signatures  []Signature `json:"signatures"`
	Status      TransactionStatus `json:"status"`
	ExecutedBy  string   `json:"executed_by"`
	ExecutedAt  int64    `json:"executed_at"`
	CreatedAt   int64    `json:"created_at"`
}

type Signature struct {
	Owner   string `json:"owner"`
	V       uint8  `json:"v"`
	R       string `json:"r"`
	S       string `json:"s"`
}

type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusApproved TransactionStatus = "approved"
	StatusExecuted TransactionStatus = "executed"
	StatusFailed   TransactionStatus = "failed"
	StatusRevoked TransactionStatus = "revoked"
)

// ============================================================================
// Multi-Sig Service
// ============================================================================

type MultiSigService struct {
	config    *Config
	redis     *redis.Client
	wallets   map[string]*MultiSigWallet
	transactions map[string]*TransactionRequest
	privateKey *ecdsa.PrivateKey
}

func NewMultiSigService(config *Config) *MultiSigService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	// Generate deployment key
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	return &MultiSigService{
		config:         config,
		redis:          redisClient,
		wallets:        make(map[string]*MultiSigWallet),
		transactions:   make(map[string]*TransactionRequest),
		privateKey:     privateKey,
	}
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
		Nonce:     wallet.Nonce,
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

	// Build execute data
	executeData := s.buildExecuteData(tx)

	// In production, would broadcast to network
	txHash := s.broadcastTransaction(wallet.Address, tx.To, tx.Value, executeData)

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

func (s *MultiSigService) computeAddress(owners []string, threshold uint) string {
	// Simplified address computation
	data := fmt.Sprintf("%s:%d:%v", strings.Join(owners, ","), threshold, owners)
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:20])
}

func (s *MultiSigService) buildExecuteData(tx *TransactionRequest) []byte {
	// EIP-1271 magic value
	magicValue := "0x1626ba7e"
	
	// Encode parameters
	// In production, would properly encode the transaction data
	return []byte(magicValue)
}

func (s *MultiSigService) broadcastTransaction(from, to, value string, data []byte) string {
	// Simplified - would broadcast to network
	txHash := sha256.Sum256([]byte(from + to + value + string(data)))
	return "0x" + hex.EncodeToString(txHash[:])
}

func (s *MultiSigService) SignHash(owner string, hash []byte) (uint8, string, string, error) {
	// Simplified signature - in production would use actual private key
	r := sha256.Sum256(hash)
	sig := sha3.New256()
	sig.Write(r[:])
	sig.Write([]byte(owner))
	signature := sig.Sum(nil)

	return 27, hex.EncodeToString(signature[:32]), hex.EncodeToString(signature[32:]), nil
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

	// Verify each signature
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

	return true, nil
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
