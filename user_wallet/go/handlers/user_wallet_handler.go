package handlers

import (
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// UserWalletHandler handles user wallet operations
// Completely SEPARATED from Admin and MasterWallet handlers
type UserWalletHandler struct {
	db       *gorm.DB
	redis    *redis.Client
	web3Svc  *UserWeb3Service
	notifier *UserWalletNotifier
}

// UserWeb3Service handles blockchain interactions for users
type UserWeb3Service struct {
	rpcClients map[string]string
}

// UserWalletNotifier handles user wallet notifications
type UserWalletNotifier struct{}

// NewUserWalletHandler creates a new user wallet handler
func NewUserWalletHandler(db *gorm.DB, redisClient *redis.Client) *UserWalletHandler {
	return &UserWalletHandler{
		db:       db,
		redis:    redisClient,
		web3Svc:  &UserWeb3Service{rpcClients: make(map[string]string)},
		notifier: &UserWalletNotifier{},
	}
}

// ==================== WALLET MANAGEMENT ====================

// CreateUserWallet creates a new user wallet
// POST /api/v1/wallet
func (h *UserWalletHandler) CreateUserWallet(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		Chain      string `json:"chain" binding:"required"`
		WalletType string `json:"wallet_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	walletType := req.WalletType
	if walletType == "" {
		walletType = "internal"
	}
	address := generateUserAddress(userID, req.Chain)
	wallet := map[string]interface{}{
		"id": generateUserWalletID(), "user_id": userID, "address": address, "chain": req.Chain, "wallet_type": walletType, "is_active": true, "created_at": time.Now(),
	}
	c.JSON(http.StatusCreated, wallet)
}

// GetUserWallets gets all wallets for current user
// GET /api/v1/wallet
func (h *UserWalletHandler) GetUserWallets(c *gin.Context) {
	userID := c.GetUint("user_id")
	wallets := []map[string]interface{}{
		{"id": "uw1", "user_id": userID, "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f", "chain": "ethereum", "wallet_type": "internal", "balance": "2.5", "is_active": true},
		{"id": "uw2", "user_id": userID, "address": "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "chain": "bitcoin", "wallet_type": "internal", "balance": "0.1", "is_active": true},
	}
	c.JSON(http.StatusOK, gin.H{"data": wallets, "total": len(wallets)})
}

// GetWallet gets specific wallet details
// GET /api/v1/wallet/:id
func (h *UserWalletHandler) GetWallet(c *gin.Context) {
	walletID := c.Param("id")
	userID := c.GetUint("user_id")
	wallet := map[string]interface{}{
		"id": walletID, "user_id": userID, "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f", "chain": "ethereum", "wallet_type": "internal", "balance": "2.5", "is_active": true, "created_at": time.Now().Add(-30 * 24 * time.Hour),
	}
	c.JSON(http.StatusOK, wallet)
}

// GetWalletBalance gets wallet balance
// GET /api/v1/wallet/:id/balance
func (h *UserWalletHandler) GetWalletBalance(c *gin.Context) {
	walletID := c.Param("id")
	balances := []map[string]interface{}{
		{"token": "ETH", "balance": "2.5", "available": "2.5", "locked": "0.0", "usd_value": "7500.00"},
		{"token": "USDT", "balance": "1000.0", "available": "1000.0", "locked": "0.0", "usd_value": "1000.00"},
		{"token": "WBTC", "balance": "0.05", "available": "0.05", "locked": "0.0", "usd_value": "2500.00"},
	}
	_ = walletID
	c.JSON(http.StatusOK, gin.H{"balances": balances})
}

// ==================== TRANSACTIONS ====================

// SendTransaction sends crypto from user wallet
// POST /api/v1/wallet/:id/send
func (h *UserWalletHandler) SendTransaction(c *gin.Context) {
	walletID := c.Param("id")
	userID := c.GetUint("user_id")
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
	if !validateUserAddress(req.To, req.Chain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recipient address"})
		return
	}
	txHash := simulateUserTransaction(req.Chain, req.To, req.Amount, req.Token)
	c.JSON(http.StatusAccepted, gin.H{
		"id": generateUserTXID(), "wallet_id": walletID, "user_id": userID, "tx_hash": txHash, "to": req.To, "amount": req.Amount, "token": req.Token, "chain": req.Chain, "status": "pending", "created_at": time.Now(),
	})
}

// GetTransactions gets transaction history
// GET /api/v1/wallet/:id/transactions
func (h *UserWalletHandler) GetTransactions(c *gin.Context) {
	walletID := c.Param("id")
	userID := c.GetUint("user_id")
	transactions := []map[string]interface{}{
		{"id": "utx1", "tx_hash": "0x1234567890abcdef", "type": "outgoing", "to": "0xabcd1234567890", "amount": "0.5", "token": "ETH", "fee": "0.001", "status": "confirmed", "created_at": time.Now().Add(-1 * time.Hour)},
		{"id": "utx2", "tx_hash": "0xabcdef1234567890", "type": "incoming", "from": "0x9876543210fedcba", "amount": "1.0", "token": "USDT", "fee": "0.5", "status": "confirmed", "created_at": time.Now().Add(-2 * time.Hour)},
	}
	_ = walletID
	_ = userID
	c.JSON(http.StatusOK, gin.H{"data": transactions, "total": len(transactions)})
}

// GetTransaction gets transaction details
// GET /api/v1/transactions/:id
func (h *UserWalletHandler) GetTransaction(c *gin.Context) {
	txID := c.Param("id")
	userID := c.GetUint("user_id")
	tx := map[string]interface{}{
		"id": txID, "user_id": userID, "tx_hash": "0x1234567890abcdef", "type": "outgoing", "to": "0xabcd1234567890", "from": "0x742d35Cc6634C0532925a3b844Bc9e7595f", "amount": "0.5", "token": "ETH", "fee": "0.001", "status": "confirmed", "created_at": time.Now().Add(-1 * time.Hour),
	}
	c.JSON(http.StatusOK, tx)
}

// ==================== SWAPS ====================

// SwapTokens swaps tokens
// POST /api/v1/wallet/swap
func (h *UserWalletHandler) SwapTokens(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		FromToken string  `json:"from_token" binding:"required"`
		ToToken   string  `json:"to_token" binding:"required"`
		Amount    float64 `json:"amount" binding:"required"`
		Slippage  float64 `json:"slippage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	slippage := req.Slippage
	if slippage == 0 {
		slippage = 0.5
	}
	fromAmount := big.NewFloat(req.Amount)
	toAmount := fromAmount.Mul(fromAmount, big.NewFloat(0.95))
	txHash := simulateUserTransaction("ethereum", "0xswap", req.Amount*0.95, req.ToToken)
	swap := map[string]interface{}{
		"id": generateUserTXID(), "user_id": userID, "from_token": req.FromToken, "to_token": req.ToToken, "from_amount": req.Amount, "to_amount": toAmount.String(), "tx_hash": txHash, "status": "pending", "created_at": time.Now(),
	}
	c.JSON(http.StatusAccepted, swap)
}

// GetSwapQuote gets swap quote
// GET /api/v1/wallet/swap/quote
func (h *UserWalletHandler) GetSwapQuote(c *gin.Context) {
	fromToken := c.Query("from_token")
	toToken := c.Query("to_token")
	amount := c.Query("amount")
	quote := map[string]interface{}{
		"from_token": fromToken, "to_token": toToken, "from_amount": amount, "to_amount": "0.95", "price_impact": "0.5", "slippage": "0.5", "expires_at": time.Now().Add(30 * time.Second),
	}
	c.JSON(http.StatusOK, quote)
}

// ==================== STAKING ====================

// Stake tokens
// POST /api/v1/wallet/stake
func (h *UserWalletHandler) Stake(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		Token  string  `json:"token" binding:"required"`
		Amount float64 `json:"amount" binding:"required"`
		Chain  string  `json:"chain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	stake := map[string]interface{}{
		"id": generateStakeID(), "user_id": userID, "token": req.Token, "amount": req.Amount, "chain": req.Chain, "reward": req.Amount * 0.05, "status": "active", "staked_at": time.Now(),
	}
	c.JSON(http.StatusCreated, stake)
}

// Unstake tokens
// POST /api/v1/wallet/unstake
func (h *UserWalletHandler) Unstake(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		StakeID string `json:"stake_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Unstake initiated", "stake_id": req.StakeID, "user_id": userID, "status": "processing"})
}

// GetStakes gets user stakes
// GET /api/v1/wallet/stakes
func (h *UserWalletHandler) GetStakes(c *gin.Context) {
	userID := c.GetUint("user_id")
	stakes := []map[string]interface{}{
		{"id": "stk1", "user_id": userID, "token": "ETH", "amount": "10.0", "reward": "0.5", "status": "active", "staked_at": time.Now().Add(-30 * 24 * time.Hour)},
	}
	c.JSON(http.StatusOK, gin.H{"data": stakes, "total": len(stakes)})
}

// ==================== NFTs ====================

// GetNFTs gets user NFTs
// GET /api/v1/wallet/nfts
func (h *UserWalletHandler) GetNFTs(c *gin.Context) {
	userID := c.GetUint("user_id")
	nfts := []map[string]interface{}{
		{"id": "nft1", "user_id": userID, "token_id": "1", "contract": "0x1234", "name": "Tiger #1", "image_url": "https://example.com/nft1.png", "chain": "ethereum"},
	}
	c.JSON(http.StatusOK, gin.H{"data": nfts, "total": len(nfts)})
}

// TransferNFT transfers NFT
// POST /api/v1/wallet/nft/transfer
func (h *UserWalletHandler) TransferNFT(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		NFTID    string `json:"nft_id" binding:"required"`
		ToAddress string `json:"to_address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	txHash := simulateUserTransaction("ethereum", req.ToAddress, 1, "NFT")
	c.JSON(http.StatusAccepted, gin.H{"id": generateUserTXID(), "user_id": userID, "nft_id": req.NFTID, "to_address": req.ToAddress, "tx_hash": txHash, "status": "pending"})
}

// ==================== PORTFOLIO ====================

// GetPortfolio gets user portfolio
// GET /api/v1/wallet/portfolio
func (h *UserWalletHandler) GetPortfolio(c *gin.Context) {
	userID := c.GetUint("user_id")
	portfolio := map[string]interface{}{
		"user_id": userID, "total_value_usd": "11000.00", "change_24h": 2.5, "change_7d": 5.2,
		"assets": []map[string]interface{}{
			{"token": "ETH", "balance": "2.5", "value_usd": "7500.00", "percentage": 68.18},
			{"token": "USDT", "balance": "1000.0", "value_usd": "1000.00", "percentage": 9.09},
			{"token": "WBTC", "balance": "0.05", "value_usd": "2500.00", "percentage": 22.73},
		},
	}
	c.JSON(http.StatusOK, portfolio)
}

// GetHistory gets transaction and activity history
// GET /api/v1/wallet/history
func (h *UserWalletHandler) GetHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	history := []map[string]interface{}{
		{"id": "1", "type": "transfer", "subtype": "send", "token": "ETH", "amount": "0.5", "status": "confirmed", "created_at": time.Now().Add(-1 * time.Hour)},
		{"id": "2", "type": "transfer", "subtype": "receive", "token": "USDT", "amount": "1000", "status": "confirmed", "created_at": time.Now().Add(-2 * time.Hour)},
		{"id": "3", "type": "swap", "from_token": "ETH", "to_token": "USDT", "from_amount": "1.0", "to_amount": "3000", "status": "confirmed", "created_at": time.Now().Add(-3 * time.Hour)},
	}
	_ = userID
	c.JSON(http.StatusOK, gin.H{"data": history, "total": len(history)})
}

// ==================== HELPER FUNCTIONS ====================

func generateUserWalletID() string {
	return "uw_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateUserTXID() string {
	return "utx_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateStakeID() string {
	return "stk_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateUserAddress(userID uint, chain string) string {
	return "0x" + fmt.Sprintf("%040d", userID)
}

func validateUserAddress(address, chain string) bool {
	if chain == "ethereum" || chain == "polygon" || chain == "bsc" || chain == "arbitrum" || chain == "optimism" {
		return strings.HasPrefix(address, "0x") && len(address) == 42
	}
	if chain == "bitcoin" {
		return (strings.HasPrefix(address, "bc1") || strings.HasPrefix(address, "1") || strings.HasPrefix(address, "3")) && len(address) >= 26 && len(address) <= 62
	}
	return true
}

func simulateUserTransaction(chain, to string, amount float64, token string) string {
	return "0x" + strconv.FormatInt(time.Now().UnixNano(), 16)
}
