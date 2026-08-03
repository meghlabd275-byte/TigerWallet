package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MarginHandler struct {
	db *sql.DB
}

func NewMarginHandler(db *sql.DB) *MarginHandler {
	return &MarginHandler{db: db}
}

func (h *MarginHandler) GetAccount(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var account struct {
		ID                uuid.UUID `json:"id"`
		TotalAssets      float64   `json:"totalAssets"`
		TotalLiabilities float64   `json:"totalLiabilities"`
		NetAssets        float64   `json:"netAssets"`
		AvailableBalance float64   `json:"availableBalance"`
		MarginRatio      float64   `json:"marginRatio"`
		RiskLevel        string    `json:"riskLevel"`
	}

	err := h.db.QueryRow(`
		SELECT id, total_assets, total_liabilities, net_assets, 
			available_balance, margin_ratio, risk_level
		FROM margin_accounts WHERE user_id = $1
	`, userID).Scan(
		&account.ID, &account.TotalAssets, &account.TotalLiabilities,
		&account.NetAssets, &account.AvailableBalance, &account.MarginRatio,
		&account.RiskLevel,
	)

	if err == sql.ErrNoRows {
		// Create margin account if doesn't exist
		var newID uuid.UUID
		h.db.QueryRow(`
			INSERT INTO margin_accounts (user_id, total_assets, total_liabilities, 
				net_assets, available_balance, margin_ratio, risk_level)
			VALUES ($1, 0, 0, 0, 0, 0, 'SAFE')
			RETURNING id
		`, userID).Scan(&newID)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"id": newID, "totalAssets": 0, "totalLiabilities": 0,
				"netAssets": 0, "availableBalance": 0, "marginRatio": 0, "riskLevel": "SAFE",
			},
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": account})
}

func (h *MarginHandler) Borrow(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req struct {
		Token  string  `json:"token" binding:"required"`
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check collateral
	var availableBalance float64
	err := h.db.QueryRow(`
		SELECT available_balance FROM margin_accounts WHERE user_id = $1
	`, userID).Scan(&availableBalance)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Margin account not found"})
		return
	}

	// Get token price (simplified)
	usdValue := req.Amount * 43000 // BTC price

	if usdValue > availableBalance*0.5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient collateral"})
		return
	}

	// Create borrow record
	h.db.Exec(`
		INSERT INTO margin_borrows (account_id, token_id, amount, interest_rate, status)
		VALUES ((SELECT id FROM margin_accounts WHERE user_id = $1),
			(SELECT id FROM tokens WHERE symbol = $2), $3, 0.0001, 'ACTIVE')
	`, userID, req.Token, req.Amount)

	// Update account
	h.db.Exec(`
		UPDATE margin_accounts 
		SET total_liabilities = total_liabilities + $1,
			net_assets = total_assets - total_liabilities,
			available_balance = available_balance + $1
		WHERE user_id = $2
	`, usdValue, userID)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"borrowed": req.Amount}})
}

func (h *MarginHandler) Repay(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req struct {
		Token  string  `json:"token" binding:"required"`
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Repay logic
	h.db.Exec(`
		UPDATE margin_borrows 
		SET status = 'REPAID', repaid_at = NOW()
		WHERE account_id = (SELECT id FROM margin_accounts WHERE user_id = $1)
		AND token_id = (SELECT id FROM tokens WHERE symbol = $2)
		AND status = 'ACTIVE'
		LIMIT 1
	`, userID, req.Token)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"repaid": req.Amount}})
}

func (h *MarginHandler) GetPositions(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	rows, err := h.db.Query(`
		SELECT p.id, p.side, t.symbol, p.size, p.entry_price, p.leverage, 
			p.margin, p.pnl, p.status, p.opened_at
		FROM margin_positions p
		JOIN tokens t ON p.token_id = t.id
		WHERE p.account_id = (SELECT id FROM margin_accounts WHERE user_id = $1)
		AND p.status = 'OPEN'
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch positions"})
		return
	}
	defer rows.Close()

	var positions []gin.H
	for rows.Next() {
		var pos struct {
			ID          uuid.UUID `json:"id"`
			Side        string    `json:"side"`
			Token       string    `json:"token"`
			Size        float64   `json:"size"`
			EntryPrice  float64   `json:"entryPrice"`
			Leverage    int       `json:"leverage"`
			Margin      float64   `json:"margin"`
			PNL         float64   `json:"pnl"`
			Status      string    `json:"status"`
			OpenedAt    string    `json:"openedAt"`
		}
		rows.Scan(&pos.ID, &pos.Side, &pos.Token, &pos.Size, &pos.EntryPrice,
			&pos.Leverage, &pos.Margin, &pos.PNL, &pos.Status, &pos.OpenedAt)
		positions = append(positions, gin.H{
			"id":         pos.ID,
			"side":       pos.Side,
			"token":      pos.Token,
			"size":       pos.Size,
			"entryPrice": pos.EntryPrice,
			"leverage":    pos.Leverage,
			"margin":      pos.Margin,
			"pnl":        pos.PNL,
			"status":     pos.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": positions})
}

func (h *MarginHandler) OpenPosition(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req struct {
		Token     string  `json:"token" binding:"required"`
		Side      string  `json:"side" binding:"required,oneof=LONG SHORT"`
		Size      float64 `json:"size" binding:"required,gt=0"`
		Leverage  int     `json:"leverage" binding:"required,min=1,max=10"`
		Margin    float64 `json:"margin" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Leverage < 1 || req.Leverage > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Leverage must be between 1x and 10x"})
		return
	}

	// Get current price (simplified)
	entryPrice := 43000.0

	// Calculate liquidation price
	var liquidationPrice float64
	if req.Side == "LONG" {
		liquidationPrice = entryPrice * (1 - (1/float64(req.Leverage)))
	} else {
		liquidationPrice = entryPrice * (1 + (1/float64(req.Leverage)))
	}

	// Get account ID
	var accountID uuid.UUID
	h.db.QueryRow("SELECT id FROM margin_accounts WHERE user_id = $1", userID).Scan(&accountID)

	// Get token ID
	var tokenID uuid.UUID
	h.db.QueryRow("SELECT id FROM tokens WHERE symbol = $1", req.Token).Scan(&tokenID)

	// Create position
	h.db.Exec(`
		INSERT INTO margin_positions (account_id, token_id, side, size, entry_price, 
			leverage, margin, liquidation_price, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'OPEN')
	`, accountID, tokenID, req.Side, req.Size, entryPrice, req.Leverage, req.Margin, liquidationPrice)

	// Update account
	h.db.Exec(`
		UPDATE margin_accounts 
		SET available_balance = available_balance - $1,
			total_assets = total_assets + $1,
			net_assets = total_assets - total_liabilities
		WHERE user_id = $2
	`, req.Margin, userID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"status":             "OPEN",
			"entryPrice":         entryPrice,
			"liquidationPrice":   liquidationPrice,
		},
	})
}

func (h *MarginHandler) ClosePosition(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	positionID := c.Param("id")

	// Close position
	h.db.Exec(`
		UPDATE margin_positions 
		SET status = 'CLOSED', closed_at = NOW()
		WHERE id = $1 AND account_id = (SELECT id FROM margin_accounts WHERE user_id = $2)
		AND status = 'OPEN'
	`, positionID, userID)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "CLOSED"}})
}
