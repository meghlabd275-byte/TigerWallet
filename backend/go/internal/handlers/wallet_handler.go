package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WalletHandler struct {
	db *sql.DB
}

func NewWalletHandler(db *sql.DB) *WalletHandler {
	return &WalletHandler{db: db}
}

func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	type Balance struct {
		Chain    string  `json:"chain"`
		Address  string  `json:"address"`
		Balance  float64 `json:"balance"`
		Reserved float64 `json:"reserved"`
	}

	rows, err := h.db.Query(`
		SELECT chain, address, balance, reserved_balance
		FROM wallets WHERE user_id = $1
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balances"})
		return
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		var b Balance
		rows.Scan(&b.Chain, &b.Address, &b.Balance, &b.Reserved)
		balances = append(balances, b)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": balances})
}

func (h *WalletHandler) Transfer(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req struct {
		To      string  `json:"to" binding:"required"`
		Amount  float64 `json:"amount" binding:"required,gt=0"`
		Chain   string  `json:"chain" binding:"required"`
		Token   string  `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate address format (basic check)
	if len(req.To) < 20 || len(req.To) > 44 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recipient address"})
		return
	}

	// Check balance
	var balance float64
	err := h.db.QueryRow(`
		SELECT balance FROM wallets 
		WHERE user_id = $1 AND chain = $2
	`, userID, req.Chain).Scan(&balance)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	if balance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
		return
	}

	// In production: initiate blockchain transfer
	// For now, just record the transaction

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"txHash": "0x" + uuid.New().String(),
			"status": "PENDING",
		},
	})
}

func (h *WalletHandler) GetTransactions(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	// Simplified transaction history
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": []gin.H{},
	})
}
