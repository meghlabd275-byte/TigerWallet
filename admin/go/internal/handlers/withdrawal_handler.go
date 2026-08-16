package handlers

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// BlockchainService handles blockchain transactions
type BlockchainService struct {
	redis *redis.Client
}

// NewBlockchainService creates a new blockchain service
func NewBlockchainService(redisClient *redis.Client) *BlockchainService {
	return &BlockchainService{redis: redisClient}
}

// TransactionRequest represents a blockchain transaction request
type TransactionRequest struct {
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	Amount      string `json:"amount"`
	Token       string `json:"token"`
	Chain       string `json:"chain"`
	GasPrice    string `json:"gas_price"`
	GasLimit    string `json:"gas_limit"`
}

// BroadcastWithdrawal is intentionally DISABLED. The admin service must never
// broadcast blockchain transactions or move crypto assets — that is the wallet
// owner's action via the canonical wallet backend (go/wallet_api). This method
// always returns an error so any caller is fail-closed rather than silently
// fabricating a transaction hash.
func (s *BlockchainService) BroadcastWithdrawal(ctx context.Context, tx *TransactionRequest) (string, error) {
	return "", fmt.Errorf("admin service must not broadcast transactions; crypto asset movement is performed only by the canonical wallet backend")
}

// GetTransactionStatus queries the locally-tracked withdrawal record status.
// It does NOT fabricate a "confirmed" status; unknown hashes return an error.
func (s *BlockchainService) GetTransactionStatus(ctx context.Context, txHash string) (string, error) {
	key := fmt.Sprintf("tx:%s", txHash)
	status, err := s.redis.HGet(ctx, key, "status").Result()
	if err == redis.Nil {
		return "", fmt.Errorf("transaction %s not found", txHash)
	}
	return status, err
}

// WalletBalanceService handles user wallet balance operations
type WalletBalanceService struct {
	db *database.PostgresDB
}

// NewWalletBalanceService creates a new wallet balance service
func NewWalletBalanceService(db *database.PostgresDB) *WalletBalanceService {
	return &WalletBalanceService{db: db}
}

// CreditBalance credits balance to a user's wallet
func (s *WalletBalanceService) CreditBalance(ctx context.Context, userID uint, token string, amount string) error {
	// Parse amount
	_, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	// Find or create wallet balance
	var balance models.WalletBalance
	result := s.db.Where("user_id = ? AND token = ?", userID, token).First(&balance)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new balance
		balance = models.WalletBalance{
			UserID:    userID,
			Token:     token,
			Balance:   amount,
			Available: amount,
		}
		return s.db.Create(&balance).Error
	}

	// Update existing balance
	currentBalance, _ := new(big.Float).SetString(balance.Balance)
	creditAmount, _ := new(big.Float).SetString(amount)
	newBalance := new(big.Float).Add(currentBalance, creditAmount)

	return s.db.Model(&balance).Updates(map[string]interface{}{
		"balance":    newBalance.String(),
		"available":  newBalance.String(),
		"updated_at": time.Now(),
	}).Error
}

// DebitBalance debits balance from a user's wallet
func (s *WalletBalanceService) DebitBalance(ctx context.Context, userID uint, token string, amount string) error {
	_, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	var balance models.WalletBalance
	if err := s.db.Where("user_id = ? AND token = ?", userID, token).First(&balance).Error; err != nil {
		return fmt.Errorf("balance not found: %w", err)
	}

	currentBalance, _ := new(big.Float).SetString(balance.Available)
	debitAmount, _ := new(big.Float).SetString(amount)
	newBalance := new(big.Float).Sub(currentBalance, debitAmount)

	if newBalance.Sign() < 0 {
		return fmt.Errorf("insufficient balance")
	}

	return s.db.Model(&balance).Updates(map[string]interface{}{
		"balance":    newBalance.String(),
		"available":  newBalance.String(),
		"updated_at": time.Now(),
	}).Error
}

// TransactionLog logs blockchain transactions
func (s *WalletBalanceService) LogTransaction(ctx context.Context, userID uint, txType, token, amount, txHash, status string) error {
	txLog := models.TransactionLog{
		UserID:    userID,
		Type:      txType,
		Token:     token,
		Amount:    amount,
		TxHash:    txHash,
		Status:    status,
		CreatedAt: time.Now(),
	}
	return s.db.Create(&txLog).Error
}

// WithdrawHandler handles withdrawal-related requests
type WithdrawHandler struct {
	db               *database.PostgresDB
	blockchainSvc    *BlockchainService
	walletBalanceSvc *WalletBalanceService
}

// NewWithdrawHandler creates a new withdrawal handler
func NewWithdrawHandler(db *database.PostgresDB, redisClient *redis.Client) *WithdrawHandler {
	return &WithdrawHandler{
		db:               db,
		blockchainSvc:    NewBlockchainService(redisClient),
		walletBalanceSvc: NewWalletBalanceService(db),
	}
}

// WithdrawalHandler handles withdrawal-related requests
type WithdrawalHandler struct {
	db               *database.PostgresDB
	blockchainSvc    *BlockchainService
	walletBalanceSvc *WalletBalanceService
}

// NewWithdrawalHandler creates a new withdrawal handler
func NewWithdrawalHandler(db *database.PostgresDB, redisClient *redis.Client) *WithdrawalHandler {
	return &WithdrawalHandler{
		db:               db,
		blockchainSvc:    NewBlockchainService(redisClient),
		walletBalanceSvc: NewWalletBalanceService(db),
	}
}

// ListWithdrawals lists all withdrawals
func (h *WithdrawalHandler) ListWithdrawals(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	chain := c.Query("chain")
	token := c.Query("token")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var withdrawals []models.Withdrawal
	var total int64

	query := h.db.Model(&models.Withdrawal{}).Preload("User")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if token != "" {
		query = query.Where("token = ?", token)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&withdrawals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch withdrawals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        withdrawals,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetWithdrawal gets a withdrawal by ID
func (h *WithdrawalHandler) GetWithdrawal(c *gin.Context) {
	withdrawalID := c.Param("id")

	var withdrawal models.Withdrawal
	if err := h.db.Preload("User").First(&withdrawal, withdrawalID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	c.JSON(http.StatusOK, withdrawal)
}

// ApproveWithdrawal approves a withdrawal and initiates blockchain transaction
func (h *WithdrawalHandler) ApproveWithdrawal(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	withdrawalID := c.Param("id")

	var req struct {
		Notes string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Notes = ""
	}

	var withdrawal models.Withdrawal
	if err := h.db.Preload("User").First(&withdrawal, withdrawalID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	if withdrawal.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Withdrawal is not pending"})
		return
	}

	now := time.Now()

	// Admin approval is a GOVERNANCE action only: it records that an admin has
	// authorized the withdrawal. It must NOT debit the user's balance and must NOT
	// broadcast any blockchain transaction. Crypto asset movement is performed
	// exclusively by the wallet owner via the canonical wallet backend
	// (go/wallet_api), which debits the balance and broadcasts the signed tx after
	// it observes the admin "approved" status. Keeping fund movement out of the
	// admin service enforces the invariant that no admin can withdraw crypto.

	if err := h.db.Model(&withdrawal).Updates(map[string]interface{}{
		"status":      "approved",
		"approved_at": now,
		"approved_by": adminID,
		"notes":       req.Notes,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve withdrawal"})
		return
	}

	// Log activity (governance record only — no tx_hash, no broadcast)
	logAdminActivity(h.db, adminID, "approve_withdrawal", "withdrawal", withdrawalID, "Withdrawal approved (governance record; fund movement handled by canonical wallet backend)", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"message": "Withdrawal approved successfully. The wallet owner's canonical wallet backend will perform the on-chain fund movement.",
		"status":  "approved",
	})
}

// RejectWithdrawal rejects a withdrawal and refunds the user's balance
func (h *WithdrawalHandler) RejectWithdrawal(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	withdrawalID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rejection reason is required"})
		return
	}

	var withdrawal models.Withdrawal
	if err := h.db.Preload("User").First(&withdrawal, withdrawalID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	if withdrawal.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Withdrawal is not pending"})
		return
	}

	now := time.Now()

	// Governance record only: do NOT credit the user's balance here. Balance
	// adjustments are the canonical wallet backend's responsibility; it observes
	// the admin "rejected" status and reconciles any held/locked funds. The admin
	// service must never touch balances or broadcast transactions.

	// Update withdrawal
	if err := h.db.Model(&withdrawal).Updates(map[string]interface{}{
		"status":           "rejected",
		"rejected_at":      now,
		"rejected_by":      adminID,
		"rejection_reason": req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject withdrawal"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "reject_withdrawal", "withdrawal", withdrawalID, "Withdrawal rejected and balance refunded: "+req.Reason, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"message":  "Withdrawal rejected and balance refunded",
		"refunded": true,
		"amount":   withdrawal.Amount,
		"token":    withdrawal.Token,
	})
}

// ProcessWithdrawal processes a withdrawal (marks as completed)
func (h *WithdrawalHandler) ProcessWithdrawal(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	withdrawalID := c.Param("id")

	var req struct {
		TxHash string `json:"tx_hash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction hash is required"})
		return
	}

	var withdrawal models.Withdrawal
	if err := h.db.Preload("User").First(&withdrawal, withdrawalID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	if withdrawal.Status != "approved" && withdrawal.Status != "processing" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Withdrawal must be approved first"})
		return
	}

	now := time.Now()

	// Update withdrawal
	if err := h.db.Model(&withdrawal).Updates(map[string]interface{}{
		"status":       "completed",
		"processed_at": now,
		"tx_hash":      req.TxHash,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process withdrawal"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "process_withdrawal", "withdrawal", withdrawalID, "Withdrawal processed: "+req.TxHash, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Withdrawal processed successfully"})
}

// GetWithdrawalStats gets withdrawal statistics
func (h *WithdrawalHandler) GetWithdrawalStats(c *gin.Context) {
	var stats struct {
		Pending   int64   `json:"pending"`
		Approved  int64   `json:"approved"`
		Completed int64   `json:"completed"`
		Rejected  int64   `json:"rejected"`
		Total     float64 `json:"total"`
	}

	h.db.Model(&models.Withdrawal{}).Where("status = ?", "pending").Count(&stats.Pending)
	h.db.Model(&models.Withdrawal{}).Where("status = ?", "approved").Count(&stats.Approved)
	h.db.Model(&models.Withdrawal{}).Where("status = ?", "completed").Count(&stats.Completed)
	h.db.Model(&models.Withdrawal{}).Where("status = ?", "rejected").Count(&stats.Rejected)

	// Get total amount
	var result []struct {
		Total string
	}
	h.db.Model(&models.Withdrawal{}).Where("status IN ?", []string{"completed", "processing"}).Pluck("COALESCE(SUM(amount), 0)", &result)
	if len(result) > 0 {
		stats.Total, _ = strconv.ParseFloat(result[0].Total, 64)
	}

	c.JSON(http.StatusOK, stats)
}

// BulkApproveWithdrawals approves multiple withdrawals at once
func (h *WithdrawalHandler) BulkApproveWithdrawals(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		WithdrawalIDs []uint `json:"withdrawal_ids" binding:"required"`
		Notes         string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Withdrawal IDs are required"})
		return
	}

	if len(req.WithdrawalIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No withdrawal IDs provided"})
		return
	}

	if len(req.WithdrawalIDs) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot approve more than 50 withdrawals at once"})
		return
	}

	now := time.Now()

	// Update withdrawals
	result := h.db.Model(&models.Withdrawal{}).
		Where("id IN ? AND status = ?", req.WithdrawalIDs, "pending").
		Updates(map[string]interface{}{
			"status":      "approved",
			"approved_at": now,
			"approved_by": adminID,
			"notes":       req.Notes,
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve withdrawals"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "bulk_approve_withdrawals", "withdrawal", "",
		"Approved "+strconv.Itoa(int(result.RowsAffected))+" withdrawals", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"message":         "Withdrawals approved successfully",
		"approved_count":  result.RowsAffected,
		"requested_count": len(req.WithdrawalIDs),
	})
}
