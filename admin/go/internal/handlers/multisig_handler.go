package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
)

// MultisigHandler handles multisig wallet requests - COMPLETE IMPLEMENTATION
type MultisigHandler struct {
	db *database.PostgresDB
}

// NewMultisigHandler creates a new multisig handler
func NewMultisigHandler(db *database.PostgresDB) *MultisigHandler {
	return &MultisigHandler{db: db}
}

// ListWallets lists all multisig wallets
func (h *MultisigHandler) ListWallets(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	chain := c.Query("chain")
	status := c.Query("status")
	search := c.Query("search")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var wallets []models.MultisigWallet
	var total int64

	query := h.db.Model(&models.MultisigWallet{})

	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		query = query.Where("name ILIKE ? OR address ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&wallets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        wallets,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetWallet gets a multisig wallet by ID
func (h *MultisigHandler) GetWallet(c *gin.Context) {
	walletID := c.Param("id")

	var wallet models.MultisigWallet
	if err := h.db.Preload("Signers").Preload("Transactions").First(&wallet, walletID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

// CreateWallet creates a new multisig wallet
func (h *MultisigHandler) CreateWallet(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		Name        string   `json:"name" binding:"required"`
		Chain       string   `json:"chain" binding:"required"`
		Address     string   `json:"address" binding:"required"`
		Threshold   int      `json:"threshold" binding:"required"`
		Signers     []string `json:"signers" binding:"required"`
		Description string   `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate threshold
	if req.Threshold < 2 || req.Threshold > len(req.Signers) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Threshold must be between 2 and number of signers"})
		return
	}

	wallet := models.MultisigWallet{
		Name:        req.Name,
		Chain:       req.Chain,
		Address:     req.Address,
		Threshold:   req.Threshold,
		Description: req.Description,
		Status:      "active",
		CreatedBy:   adminID,
	}

	if err := h.db.Create(&wallet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wallet"})
		return
	}

	// Create signers
	for _, signer := range req.Signers {
		s := models.MultisigSigner{
			WalletID: wallet.ID,
			Address:  signer,
		}
		h.db.Create(&s)
	}

	logAdminActivity(h.db, adminID, "create_multisig_wallet", "wallet",
		strconv.FormatUint(uint64(wallet.ID), 10), "Created multisig wallet: "+wallet.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, wallet)
}

// UpdateWallet updates a multisig wallet
func (h *MultisigHandler) UpdateWallet(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	walletID := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Threshold   *int   `json:"threshold"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var wallet models.MultisigWallet
	if err := h.db.First(&wallet, walletID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	updates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Threshold != nil {
		// Get current signers count
		var signerCount int64
		h.db.Model(&models.MultisigSigner{}).Where("wallet_id = ?", wallet.ID).Count(&signerCount)

		if *req.Threshold < 2 || *req.Threshold > int(signerCount) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Threshold must be between 2 and number of signers"})
			return
		}
		updates["threshold"] = *req.Threshold
	}

	if err := h.db.Model(&wallet).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wallet"})
		return
	}

	logAdminActivity(h.db, adminID, "update_multisig_wallet", "wallet", walletID,
		"Updated wallet: "+wallet.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, wallet)
}

// DeleteWallet deletes a multisig wallet
func (h *MultisigHandler) DeleteWallet(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	walletID := c.Param("id")

	var wallet models.MultisigWallet
	if err := h.db.First(&wallet, walletID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	// Delete signers first
	h.db.Where("wallet_id = ?", wallet.ID).Delete(&models.MultisigSigner{})

	if err := h.db.Delete(&wallet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete wallet"})
		return
	}

	logAdminActivity(h.db, adminID, "delete_multisig_wallet", "wallet", walletID,
		"Deleted wallet: "+wallet.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Wallet deleted successfully"})
}

// GetTransactions gets wallet transactions
func (h *MultisigHandler) GetTransactions(c *gin.Context) {
	walletID := c.Param("id")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	txStatus := c.Query("status")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var transactions []models.MultisigTransaction
	var total int64

	query := h.db.Model(&models.MultisigTransaction{}).Where("wallet_id = ?", walletID)

	if txStatus != "" {
		query = query.Where("status = ?", txStatus)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        transactions,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// ApproveTransaction approves a multisig transaction
func (h *MultisigHandler) ApproveTransaction(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	walletID := c.Param("id")
	txID := c.Param("txId")

	var tx models.MultisigTransaction
	if err := h.db.Where("id = ? AND wallet_id = ?", txID, walletID).First(&tx).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	if tx.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction is not pending"})
		return
	}

	// Get wallet to find threshold
	var wallet models.MultisigWallet
	if err := h.db.First(&wallet, walletID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	// Get current approval count
	var approvalCount int64
	h.db.Model(&models.MultisigApproval{}).Where("transaction_id = ?", tx.ID).Count(&approvalCount)

	// Add approval
	approval := models.MultisigApproval{
		TransactionID: tx.ID,
		ApprovedBy:    adminID,
		ApprovedAt:    time.Now(),
	}
	h.db.Create(&approval)

	// Check if threshold reached
	approvalCount++
	if int(approvalCount) >= wallet.Threshold {
		h.db.Model(&tx).Updates(map[string]interface{}{
			"status":      "approved",
			"approved_at": time.Now(),
		})
	} else {
		h.db.Model(&tx).Update("approval_count", approvalCount)
	}

	logAdminActivity(h.db, adminID, "approve_multisig_transaction", "transaction", txID,
		"Approved transaction", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Transaction approved"})
}

// RejectTransaction rejects a multisig transaction
func (h *MultisigHandler) RejectTransaction(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	walletID := c.Param("id")
	txID := c.Param("txId")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reason is required"})
		return
	}

	var tx models.MultisigTransaction
	if err := h.db.Where("id = ? AND wallet_id = ?", txID, walletID).First(&tx).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	if tx.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction is not pending"})
		return
	}

	if err := h.db.Model(&tx).Updates(map[string]interface{}{
		"status":        "rejected",
		"rejected_at":   time.Now(),
		"reject_reason": req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject transaction"})
		return
	}

	logAdminActivity(h.db, adminID, "reject_multisig_transaction", "transaction", txID,
		"Rejected transaction: "+req.Reason, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Transaction rejected"})
}

// GetWalletStats gets wallet statistics
func (h *MultisigHandler) GetWalletStats(c *gin.Context) {
	var stats struct {
		TotalWallets        int64 `json:"total_wallets"`
		ActiveWallets       int64 `json:"active_wallets"`
		PendingTransactions int64 `json:"pending_transactions"`
		TotalTransactions   int64 `json:"total_transactions"`
	}

	h.db.Model(&models.MultisigWallet{}).Count(&stats.TotalWallets)
	h.db.Model(&models.MultisigWallet{}).Where("status = ?", "active").Count(&stats.ActiveWallets)
	h.db.Model(&models.MultisigTransaction{}).Where("status = ?", "pending").Count(&stats.PendingTransactions)
	h.db.Model(&models.MultisigTransaction{}).Count(&stats.TotalTransactions)

	c.JSON(http.StatusOK, stats)
}
