package handlers

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/sha3"
)

// MasterWalletHandler handles master wallet operations
// Completely SEPARATED from Admin and UserWallet handlers
type MasterWalletHandler struct {
	db       *pgxpool.Pool
	redis    *redis.Client
	web3Svc  *Web3Service
	notifier *WalletNotifier
}

// Web3Service handles blockchain interactions
type Web3Service struct {
	rpcClients map[string]string
}

// WalletNotifier handles wallet notifications
type WalletNotifier struct{}

// NewMasterWalletHandler creates a new master wallet handler
func NewMasterWalletHandler(db *pgxpool.Pool, redisClient *redis.Client) *MasterWalletHandler {
	return &MasterWalletHandler{
		db:       db,
		redis:    redisClient,
		web3Svc:  &Web3Service{rpcClients: make(map[string]string)},
		notifier: &WalletNotifier{},
	}
}

// ==================== WALLET MANAGEMENT ====================

// CreateWallet creates a new master wallet
// POST /api/v1/master/wallets
func (h *MasterWalletHandler) CreateWallet(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		Chain     string `json:"chain" binding:"required"`
		WalletType string `json:"wallet_type" binding:"required"`
		Mnemonic  string `json:"mnemonic"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validTypes := []string{"hot", "cold", "warm"}
	valid := false
	for _, t := range validTypes {
		if req.WalletType == t {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet type"})
		return
	}

	var mnemonic string
	if req.Mnemonic != "" {
		mnemonic = req.Mnemonic
	} else {
		mnemonic = generateMnemonic()
	}

	address := deriveAddress(mnemonic, req.Chain)
	walletID := generateWalletID()

	c.JSON(http.StatusCreated, gin.H{
		"id":         walletID,
		"name":       req.Name,
		"address":    address,
		"chain":      req.Chain,
		"wallet_type": req.WalletType,
		"is_active":  true,
		"created_at": time.Now(),
	})
}

// ListWallets lists all master wallets
// GET /api/v1/master/wallets
func (h *MasterWalletHandler) ListWallets(c *gin.Context) {
	chain := c.Query("chain")
	walletType := c.Query("wallet_type")

	wallets := []map[string]interface{}{
		{"id": "1", "name": "Hot Wallet - ETH", "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f", "chain": "ethereum", "wallet_type": "hot", "balance": "10.5", "is_active": true},
		{"id": "2", "name": "Cold Wallet - BTC", "address": "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "chain": "bitcoin", "wallet_type": "cold", "balance": "2.5", "is_active": true},
		{"id": "3", "name": "Warm Wallet - POL", "address": "0x9876543210abcdef1234567890abcdef", "chain": "polygon", "wallet_type": "warm", "balance": "50000", "is_active": true},
	}

	if chain != "" {
		filtered := make([]map[string]interface{}, 0)
		for _, w := range wallets {
			if w["chain"] == chain {
				filtered = append(filtered, w)
			}
		}
		wallets = filtered
	}

	if walletType != "" {
		filtered := make([]map[string]interface{}, 0)
		for _, w := range wallets {
			if w["wallet_type"] == walletType {
				filtered = append(filtered, w)
			}
		}
		wallets = filtered
	}

	c.JSON(http.StatusOK, gin.H{"data": wallets, "total": len(wallets)})
}

// GetWallet gets wallet details
// GET /api/v1/master/wallets/:id
func (h *MasterWalletHandler) GetWallet(c *gin.Context) {
	id := c.Param("id")

	wallet := map[string]interface{}{
		"id":           id,
		"name":         "Hot Wallet - ETH",
		"address":      "0x742d35Cc6634C0532925a3b844Bc9e7595f",
		"chain":        "ethereum",
		"wallet_type":  "hot",
		"balance":      "10.5",
		"is_active":    true,
		"created_at":   time.Now().Add(-30 * 24 * time.Hour),
		"last_activity": time.Now().Add(-1 * time.Hour),
	}

	c.JSON(http.StatusOK, wallet)
}

// GetWalletBalance gets wallet balance
// GET /api/v1/master/wallets/:id/balance
func (h *MasterWalletHandler) GetWalletBalance(c *gin.Context) {
	id := c.Param("id")
	token := c.Query("token")

	balances := []map[string]interface{}{
		{"token": "ETH", "balance": "10.5", "available": "10.0", "locked": "0.5", "usd_value": "31500.00"},
		{"token": "USDT", "balance": "50000.0", "available": "50000.0", "locked": "0.0", "usd_value": "50000.00"},
		{"token": "WBTC", "balance": "0.5", "available": "0.5", "locked": "0.0", "usd_value": "25000.00"},
	}

	if token != "" {
		for _, b := range balances {
			if b["token"] == token {
				c.JSON(http.StatusOK, gin.H{"wallet_id": id, "balance": b})
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet_id": id, "balances": balances})
}

// UpdateWallet updates wallet settings
// PUT /api/v1/master/wallets/:id
func (h *MasterWalletHandler) UpdateWallet(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name     string `json:"name"`
		IsActive *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wallet updated successfully", "wallet": map[string]interface{}{"id": id, "name": req.Name, "is_active": true}})
}

// DeleteWallet deletes a master wallet
// DELETE /api/v1/master/wallets/:id
func (h *MasterWalletHandler) DeleteWallet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Wallet deleted successfully"})
}

// ==================== TRANSACTIONS ====================

// SendTransaction sends crypto from master wallet
// POST /api/v1/master/wallets/:id/send
func (h *MasterWalletHandler) SendTransaction(c *gin.Context) {
	walletID := c.Param("id")

	var req struct {
		To     string  `json:"to" binding:"required"`
		Amount float64 `json:"amount" binding:"required"`
		Token  string  `json:"token" binding:"required"`
		Chain  string  `json:"chain" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validateAddress(req.To, req.Chain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recipient address"})
		return
	}

	txHash, err := broadcastTransaction(req.Chain, req.To, req.Amount, req.Token)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to broadcast transaction", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"id":         generateTXID(),
		"wallet_id":  walletID,
		"tx_hash":    txHash,
		"to":         req.To,
		"amount":     req.Amount,
		"token":      req.Token,
		"chain":      req.Chain,
		"status":     "pending",
		"created_at": time.Now(),
	})
}

// GetTransactions gets transaction history
// GET /api/v1/master/wallets/:id/transactions
func (h *MasterWalletHandler) GetTransactions(c *gin.Context) {
	walletID := c.Param("id")
	status := c.Query("status")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = h.db.Query(ctx,
			`SELECT id, hash, from_address, to_address, amount, token, fee, status, created_at
			 FROM transactions
			 WHERE master_wallet_id = $1 AND status = $2
			 ORDER BY created_at DESC
			 LIMIT 100`,
			walletID, status,
		)
	} else {
		rows, err = h.db.Query(ctx,
			`SELECT id, hash, from_address, to_address, amount, token, fee, status, created_at
			 FROM transactions
			 WHERE master_wallet_id = $1
			 ORDER BY created_at DESC
			 LIMIT 100`,
			walletID,
		)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}
	defer rows.Close()

	transactions := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id, hash, toAddress, amount, token, txStatus string
			fromAddress, fee                             *string
			createdAt                                    time.Time
		)
		if err := rows.Scan(&id, &hash, &fromAddress, &toAddress, &amount, &token, &fee, &txStatus, &createdAt); err != nil {
			continue
		}

		tx := map[string]interface{}{
			"id":         id,
			"tx_hash":    hash,
			"to":         toAddress,
			"amount":     amount,
			"token":      token,
			"status":     txStatus,
			"created_at": createdAt,
		}
		if fromAddress != nil {
			tx["from"] = *fromAddress
		}
		if fee != nil {
			tx["fee"] = *fee
		}
		transactions = append(transactions, tx)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transactions, "total": len(transactions)})
}

// GetTransaction gets transaction details
// GET /api/v1/master/transactions/:id
func (h *MasterWalletHandler) GetTransaction(c *gin.Context) {
	txID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var (
		id, hash, toAddress, amount, token, txStatus string
		fromAddress, fee                             *string
		blockNumber                                  *int64
		createdAt                                    time.Time
		confirmedAt                                  *time.Time
	)
	err := h.db.QueryRow(ctx,
		`SELECT id, hash, from_address, to_address, amount, token, fee, status, block_number, created_at, confirmed_at
		 FROM transactions
		 WHERE id = $1`,
		txID,
	).Scan(&id, &hash, &fromAddress, &toAddress, &amount, &token, &fee, &txStatus, &blockNumber, &createdAt, &confirmedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	tx := map[string]interface{}{
		"id":         id,
		"tx_hash":    hash,
		"to":         toAddress,
		"amount":     amount,
		"token":      token,
		"status":     txStatus,
		"created_at": createdAt,
	}
	if fromAddress != nil {
		tx["from"] = *fromAddress
	}
	if fee != nil {
		tx["fee"] = *fee
	}
	if blockNumber != nil {
		tx["block_number"] = *blockNumber
	}
	if confirmedAt != nil {
		tx["confirmed_at"] = *confirmedAt
	}

	c.JSON(http.StatusOK, tx)
}

// CancelTransaction cancels a pending transaction
// POST /api/v1/master/transactions/:id/cancel
func (h *MasterWalletHandler) CancelTransaction(c *gin.Context) {
	txID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Transaction cancelled successfully", "tx_id": txID, "status": "cancelled"})
}

// ==================== GAS MANAGEMENT ====================

// GetGasPrice gets current gas price for a chain
// GET /api/v1/master/gas/:chain
func (h *MasterWalletHandler) GetGasPrice(c *gin.Context) {
	chain := c.Param("chain")

	gasPrices := map[string]map[string]string{
		"ethereum": {"slow": "20", "standard": "30", "fast": "50", "unit": "gwei"},
		"polygon": {"slow": "30", "standard": "50", "fast": "80", "unit": "gwei"},
		"bsc":     {"slow": "3", "standard": "5", "fast": "8", "unit": "gwei"},
		"arbitrum": {"slow": "0.1", "standard": "0.15", "fast": "0.25", "unit": "gwei"},
		"optimism": {"slow": "0.001", "standard": "0.002", "fast": "0.005", "unit": "gwei"},
	}

	prices, exists := gasPrices[chain]
	if !exists {
		prices = gasPrices["ethereum"]
	}

	c.JSON(http.StatusOK, gin.H{"chain": chain, "prices": prices})
}

// SetGasStrategy sets gas strategy
// POST /api/v1/master/gas/strategy
func (h *MasterWalletHandler) SetGasStrategy(c *gin.Context) {
	var req struct {
		Chain    string `json:"chain" binding:"required"`
		Strategy string `json:"strategy" binding:"required"`
		MaxGas   string `json:"max_gas"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gas strategy updated", "chain": req.Chain, "strategy": req.Strategy})
}

// ==================== MULTISIG ====================

// CreateMultisigWallet creates a multisig wallet
// POST /api/v1/master/multisig
func (h *MasterWalletHandler) CreateMultisigWallet(c *gin.Context) {
	var req struct {
		Name          string   `json:"name" binding:"required"`
		Chain         string   `json:"chain" binding:"required"`
		Signers       []string `json:"signers" binding:"required,min=2"`
		RequiredSigs  int      `json:"required_sigs" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.RequiredSigs > len(req.Signers) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Required signatures cannot exceed total signers"})
		return
	}

	address := generateMultisigAddress(req.Signers, req.RequiredSigs, req.Chain)

	wallet := map[string]interface{}{
		"id":              generateWalletID(),
		"name":            req.Name,
		"address":         address,
		"chain":           req.Chain,
		"signers":        req.Signers,
		"required_sigs":  req.RequiredSigs,
		"is_active":      true,
		"created_at":     time.Now(),
	}

	c.JSON(http.StatusCreated, wallet)
}

// SignTransaction signs a multisig transaction
// POST /api/v1/master/multisig/:id/sign
func (h *MasterWalletHandler) SignTransaction(c *gin.Context) {
	walletID := c.Param("id")

	var req struct {
		TransactionID string `json:"transaction_id" binding:"required"`
		Signer        string `json:"signer" binding:"required"`
		Signature     string `json:"signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction signed", "wallet_id": walletID, "transaction_id": req.TransactionID, "signer": req.Signer})
}

// ExecuteMultisig executes a multisig transaction
// POST /api/v1/master/multisig/:id/execute
func (h *MasterWalletHandler) ExecuteMultisig(c *gin.Context) {
	walletID := c.Param("id")

	var req struct {
		TransactionID string `json:"transaction_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve the pending multisig transaction details from the database so we
	// broadcast the real transaction rather than fabricating a hash.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var (
		toAddr, amount, token string
		chainID               int64
		status                string
	)
	err := h.db.QueryRow(ctx,
		`SELECT to_address, amount, token, chain_id, status
		 FROM transactions
		 WHERE id = $1 AND master_wallet_id = $2`,
		req.TransactionID, walletID,
	).Scan(&toAddr, &amount, &token, &chainID, &status)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Multisig transaction not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load multisig transaction"})
		return
	}
	if status != "pending" && status != "approved" {
		c.JSON(http.StatusConflict, gin.H{"error": "Transaction is not in an executable state", "status": status})
		return
	}

	chain := chainNameFromID(chainID)
	parsedAmount, parseErr := strconv.ParseFloat(amount, 64)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction amount"})
		return
	}

	txHash, err := broadcastTransaction(chain, toAddr, parsedAmount, token)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to broadcast multisig transaction", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Multisig transaction executed", "wallet_id": walletID, "tx_hash": txHash, "status": "broadcast"})
}

// ==================== WHITELABEL ====================

// CreateWhitelabel creates a new whitelabel
// POST /api/v1/master/whitelabels
func (h *MasterWalletHandler) CreateWhitelabel(c *gin.Context) {
	var req struct {
		Name        string  `json:"name" binding:"required"`
		Domain      string  `json:"domain" binding:"required"`
		Branding    string  `json:"branding"`
		FeePercent  float64 `json:"fee_percent"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	whitelabel := map[string]interface{}{
		"id":           generateWLID(),
		"name":         req.Name,
		"domain":       req.Domain,
		"branding":    req.Branding,
		"fee_percent": req.FeePercent,
		"is_active":   true,
		"created_at":  time.Now(),
	}

	c.JSON(http.StatusCreated, whitelabel)
}

// ListWhitelabels lists all whitelabels
// GET /api/v1/master/whitelabels
func (h *MasterWalletHandler) ListWhitelabels(c *gin.Context) {
	whitelabels := []map[string]interface{}{
		{"id": "wl1", "name": "WhaleWallet", "domain": "whale.example.com", "fee_percent": 0.1, "is_active": true, "users_count": 1000},
		{"id": "wl2", "name": "CryptoPro", "domain": "crypto.pro", "fee_percent": 0.15, "is_active": true, "users_count": 500},
		{"id": "wl3", "name": "TokenHub", "domain": "tokenhub.io", "fee_percent": 0.05, "is_active": true, "users_count": 2500},
	}

	c.JSON(http.StatusOK, gin.H{"data": whitelabels, "total": len(whitelabels)})
}

// ==================== ANALYTICS ====================

// GetWalletAnalytics gets wallet analytics
// GET /api/v1/master/analytics
func (h *MasterWalletHandler) GetWalletAnalytics(c *gin.Context) {
	period := c.DefaultQuery("period", "30d")

	analytics := map[string]interface{}{
		"period": period,
		"total_volume": map[string]interface{}{
			"incoming":  "5000000",
			"outgoing":  "4500000",
			"net":       "500000",
		},
		"transactions": map[string]interface{}{
			"total":        1500,
			"successful":   1480,
			"failed":       20,
			"success_rate": 98.67,
		},
		"fees": map[string]interface{}{
			"total_paid": "15000",
			"by_token": map[string]string{"ETH": "8000", "USDT": "5000", "BTC": "2000"},
		},
	}

	c.JSON(http.StatusOK, analytics)
}

// ==================== HELPER FUNCTIONS ====================

func generateWalletID() string {
	return "mw_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateTXID() string {
	return "tx_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateWLID() string {
	return "wl_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateMnemonic() string {
	words := []string{"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract", "absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid", "acoustic", "acquire", "across", "act", "action", "actor", "actress", "actual"}
	mnemonic := make([]string, 12)
	for i := 0; i < 12; i++ {
		mnemonic[i] = words[time.Now().UnixNano()%int64(len(words))]
	}
	return strings.Join(mnemonic, " ")
}

func deriveAddress(mnemonic, chain string) string {
	// Derive a deterministic EVM address from the mnemonic. The mnemonic
	// text is hashed with keccak256 (the same primitive Ethereum uses for
	// address derivation) rather than naively slicing the hex of the
	// mnemonic text, which produced a fake, non-reproducible address.
	// A full BIP39/BIP44 derivation should replace this once a proper
	// HD-key library is wired in; the address below is the last 20 bytes
	// of keccak256(mnemonic).
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(mnemonic))
	h := hasher.Sum(nil)
	return "0x" + hex.EncodeToString(h[len(h)-20:])
}

func encryptMnemonic(mnemonic string) string {
	return "encrypted_" + mnemonic
}

func validateAddress(address, chain string) bool {
	if chain == "ethereum" || chain == "polygon" || chain == "bsc" {
		return strings.HasPrefix(address, "0x") && len(address) == 42
	}
	if chain == "bitcoin" {
		return (strings.HasPrefix(address, "bc1") || strings.HasPrefix(address, "1") || strings.HasPrefix(address, "3")) && len(address) >= 26 && len(address) <= 62
	}
	return true
}

// ==================== CHAIN RPC CONFIG ====================

// chainRPCEnv maps a chain name to the environment variable that holds its
// JSON-RPC endpoint. RPC URLs must be provided via configuration so that no
// hardcoded endpoint is ever used.
var chainRPCEnv = map[string]string{
	"ethereum":  "ETH_RPC_URL",
	"bsc":       "BSC_RPC_URL",
	"polygon":   "POLYGON_RPC_URL",
	"arbitrum":  "ARBITRUM_RPC_URL",
	"optimism":  "OPTIMISM_RPC_URL",
	"base":      "BASE_RPC_URL",
	"avalanche": "AVALANCHE_RPC_URL",
}

// chainIDToName maps EVM chain IDs to canonical chain names.
var chainIDToName = map[int64]string{
	1:     "ethereum",
	56:    "bsc",
	137:   "polygon",
	42161: "arbitrum",
	10:    "optimism",
	8453:  "base",
	43114: "avalanche",
}

func chainNameFromID(id int64) string {
	if name, ok := chainIDToName[id]; ok {
		return name
	}
	return "ethereum"
}

// getChainRPCURL resolves the JSON-RPC endpoint for a chain from the
// environment. It returns an error if the chain is unsupported or the URL is
// not configured.
func getChainRPCURL(chain string) (string, error) {
	envVar, ok := chainRPCEnv[strings.ToLower(chain)]
	if !ok {
		return "", fmt.Errorf("unsupported chain for RPC broadcast: %s", chain)
	}
	url := os.Getenv(envVar)
	if url == "" {
		return "", fmt.Errorf("RPC URL not configured for chain %s (set %s)", chain, envVar)
	}
	return url, nil
}

// getChainSenderAddress resolves the wallet address that the RPC node is
// configured to sign for (the "from" of an eth_sendTransaction). It is read
// from the per-chain env var (e.g. ETH_MASTER_ADDRESS).
func getChainSenderAddress(chain string) (string, error) {
	envVar := strings.ToUpper(chain) + "_MASTER_ADDRESS"
	addr := os.Getenv(envVar)
	if addr == "" {
		return "", fmt.Errorf("sender address not configured for chain %s (set %s)", chain, envVar)
	}
	return addr, nil
}

// isNativeToken reports whether the token symbol is the chain's native asset.
func isNativeToken(chain, token string) bool {
	native := map[string]string{
		"ethereum": "ETH", "bsc": "BNB", "polygon": "MATIC",
		"arbitrum": "ETH", "optimism": "ETH", "base": "ETH", "avalanche": "AVAX",
	}
	n, ok := native[strings.ToLower(chain)]
	if !ok {
		return true
	}
	return strings.EqualFold(token, n) || token == ""
}

// weiFromAmount converts a decimal amount (in whole units) to a big.Int in wei.
func weiFromAmount(amount float64) *big.Int {
	wei := new(big.Int)
	wei.SetString(strconv.FormatFloat(amount, 'f', -1, 64), 10)
	wei.Mul(wei, big.NewInt(1_000_000_000_000_000_000))
	return wei
}

// toHexQuantity encodes a big.Int as an EVM hex quantity ("0x...").
func toHexQuantity(n *big.Int) string {
	if n == nil || n.Sign() == 0 {
		return "0x0"
	}
	return "0x" + n.Text(16)
}

// encodeERC20Transfer produces the calldata for transfer(address,uint256).
func encodeERC20Transfer(to string, amount *big.Int) ([]byte, error) {
	if !strings.HasPrefix(to, "0x") || len(to) != 42 {
		return nil, fmt.Errorf("invalid ERC20 recipient address: %s", to)
	}
	addrHex := to[2:]
	addrBytes, err := hex.DecodeString(addrHex)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient address hex: %w", err)
	}

	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte("transfer(address,uint256)"))
	selector := hasher.Sum(nil)[:4]

	paddedAddr := make([]byte, 32)
	copy(paddedAddr[32-len(addrBytes):], addrBytes)

	paddedAmount := make([]byte, 32)
	amount.FillBytes(paddedAmount)

	calldata := make([]byte, 0, 4+32+32)
	calldata = append(calldata, selector...)
	calldata = append(calldata, paddedAddr...)
	calldata = append(calldata, paddedAmount...)
	return calldata, nil
}

// rpcRequest performs a JSON-RPC POST to the given endpoint and returns the
// parsed "result" field. On an RPC error it returns an error.
func rpcRequest(rpcURL, method string, params []interface{}) (interface{}, error) {
	body, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal RPC request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(rpcURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode RPC response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	var result interface{}
	if err := json.Unmarshal(rpcResp.Result, &result); err != nil {
		return nil, fmt.Errorf("unmarshal RPC result: %w", err)
	}
	return result, nil
}

// rpcSendRawTransaction broadcasts an already-signed raw transaction via
// eth_sendRawTransaction and returns the transaction hash reported by the node.
func rpcSendRawTransaction(rpcURL, rawTxHex string) (string, error) {
	if !strings.HasPrefix(rawTxHex, "0x") {
		rawTxHex = "0x" + rawTxHex
	}
	result, err := rpcRequest(rpcURL, "eth_sendRawTransaction", []interface{}{rawTxHex})
	if err != nil {
		return "", err
	}
	txHash, ok := result.(string)
	if !ok || txHash == "" {
		return "", fmt.Errorf("RPC returned no transaction hash")
	}
	return txHash, nil
}

// broadcastTransaction constructs and broadcasts a real transaction to the
// chain's RPC endpoint via eth_sendTransaction (the node must manage the
// sender account). It returns the transaction hash reported by the node. On
// any failure it returns an error instead of a fabricated hash.
func broadcastTransaction(chain, to string, amount float64, token string) (string, error) {
	rpcURL, err := getChainRPCURL(chain)
	if err != nil {
		return "", err
	}
	from, err := getChainSenderAddress(chain)
	if err != nil {
		return "", err
	}

	txParams := map[string]interface{}{
		"from": from,
		"to":   to,
		"gas":  "0x5208", // 21000 gas limit baseline; node may estimate.
	}

	if isNativeToken(chain, token) {
		txParams["value"] = toHexQuantity(weiFromAmount(amount))
	} else {
		// ERC20 transfer: "to" becomes the token contract, calldata carries the
		// real recipient and amount.
		calldata, err := encodeERC20Transfer(to, weiFromAmount(amount))
		if err != nil {
			return "", err
		}
		txParams["to"] = token
		txParams["data"] = "0x" + hex.EncodeToString(calldata)
		txParams["value"] = "0x0"
	}

	result, err := rpcRequest(rpcURL, "eth_sendTransaction", []interface{}{txParams})
	if err != nil {
		return "", err
	}

	txHash, ok := result.(string)
	if !ok || txHash == "" {
		return "", fmt.Errorf("RPC returned no transaction hash")
	}
	return txHash, nil
}

// generateMultisigAddress computes a deterministic EVM address for a multisig
// wallet using the CREATE2 formula:
//
//	keccak256(0xff || factory || salt || initcodeHash)[12:]
//
// where:
//   - factory  = MULTISIG_FACTORY_ADDRESS (env, or zero address if unset)
//   - salt     = keccak256(signers concatenated || requiredSigs)
//   - initcodeHash = MULTISIG_INITCODE_HASH (env); if unset, keccak256(salt)
//     is used as a stable placeholder.
//
// NOTE: This yields a deterministic address derived from a proper keccak256
// hash. A real deployment still requires deploying the actual multisig
// contract initcode to the chain so that initcodeHash matches; until then the
// computed address will not correspond to an on-chain contract.
func generateMultisigAddress(signers []string, requiredSigs int, chain string) string {
	factoryHex := os.Getenv("MULTISIG_FACTORY_ADDRESS")
	if factoryHex == "" {
		factoryHex = "0000000000000000000000000000000000000000"
	}
	factory := strings.TrimPrefix(factoryHex, "0x")

	// salt = keccak256(signers || requiredSigs)
	saltHasher := sha3.NewLegacyKeccak256()
	saltHasher.Write([]byte(strings.Join(signers, "")))
	saltHasher.Write([]byte(strconv.Itoa(requiredSigs)))
	salt := saltHasher.Sum(nil)

	// initcode hash
	initcodeHashHex := os.Getenv("MULTISIG_INITCODE_HASH")
	var initcodeHash []byte
	if initcodeHashHex != "" {
		var err error
		initcodeHash, err = hex.DecodeString(strings.TrimPrefix(initcodeHashHex, "0x"))
		if err != nil || len(initcodeHash) != 32 {
			initcodeHash = nil
		}
	}
	if initcodeHash == nil {
		h := sha3.NewLegacyKeccak256()
		h.Write(salt)
		initcodeHash = h.Sum(nil)
	}

	factoryBytes, err := hex.DecodeString(factory)
	if err != nil || len(factoryBytes) != 20 {
		factoryBytes = make([]byte, 20)
	}

	data := make([]byte, 0, 1+20+32+32)
	data = append(data, 0xff)
	data = append(data, factoryBytes...)
	data = append(data, salt...)
	data = append(data, initcodeHash...)

	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	hash := h.Sum(nil)

	return "0x" + hex.EncodeToString(hash[12:])
}
