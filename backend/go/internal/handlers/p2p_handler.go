package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tigerwallet/backend/internal/models"
	"github.com/tigerwallet/backend/internal/websocket"
)

type P2PHandler struct {
	db     *sql.DB
	wsHub  *websocket.Hub
}

func NewP2PHandler(db *sql.DB, wsHub *websocket.Hub) *P2PHandler {
	return &P2PHandler{db: db, wsHub: wsHub}
}

func (h *P2PHandler) GetAdverts(c *gin.Context) {
	token := c.Query("token")
	side := c.Query("side")
	fiatCurrency := c.Query("fiatCurrency")
	paymentMethod := c.Query("paymentMethod")

	query := `
		SELECT 
			a.id, a.merchant_id, a.side, t.symbol, a.fiat_currency, a.payment_method,
			a.price, a.min_amount, a.max_amount, a.available_amount, a.is_active,
			a.created_at, a.updated_at,
			m.trader_level, m.collateral_amount, m.is_verified, m.security_score,
			m.total_trades, m.completion_rate, m.avg_release_time,
			u.username, 
			CASE WHEN m.last_active_at > NOW() - INTERVAL '5 minutes' THEN true ELSE false END as is_online
		FROM p2p_adverts a
		JOIN p2p_merchants m ON a.merchant_id = m.id
		JOIN users u ON m.user_id = u.id
		JOIN tokens t ON a.token_id = t.id
		WHERE a.is_active = true AND m.status = 'ACTIVE'
	`

	args := []interface{}{}
	argNum := 1

	if token != "" {
		query += fmt.Sprintf(" AND t.symbol = $%d", argNum)
		args = append(args, token)
		argNum++
	}
	if side != "" {
		query += fmt.Sprintf(" AND a.side = $%d", argNum)
		args = append(args, side)
		argNum++
	}
	if fiatCurrency != "" {
		query += fmt.Sprintf(" AND a.fiat_currency = $%d", argNum)
		args = append(args, fiatCurrency)
		argNum++
	}
	if paymentMethod != "" && paymentMethod != "All" {
		query += fmt.Sprintf(" AND a.payment_method = $%d", argNum)
		args = append(args, paymentMethod)
	}

	query += " ORDER BY m.trader_level DESC, m.completion_rate DESC LIMIT 50"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch adverts"})
		return
	}
	defer rows.Close()

	var adverts []models.P2PAdvert
	for rows.Next() {
		var advert models.P2PAdvert
		var tokenSymbol, merchantLevel sql.NullString
		var collateralAmount, avgReleaseTime sql.NullFloat64
		var totalTrades sql.NullInt64

		err := rows.Scan(
			&advert.ID, &advert.MerchantID, &advert.Side, &tokenSymbol,
			&advert.FiatCurrency, &advert.PaymentMethod, &advert.Price,
			&advert.MinAmount, &advert.MaxAmount, &advert.AvailableAmount,
			&advert.IsActive, &advert.CreatedAt, &advert.UpdatedAt,
			&merchantLevel, &collateralAmount, &advert.IsVerified, &advert.SecurityScore,
			&totalTrades, &advert.CompletionRate, &avgReleaseTime,
			&advert.Username, &advert.IsOnline,
		)
		if err != nil {
			continue
		}

		advert.Token = tokenSymbol.String
		advert.MerchantLevel = merchantLevel.String
		advert.CollateralLocked = collateralAmount.Float64
		advert.OrdersCompleted = int(totalTrades.Int64)
		advert.AvgReleaseTime = avgReleaseTime.Float64
		advert.IsMerchant = true
		advert.Avatar = "🧑‍💼"

		adverts = append(adverts, advert)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    adverts,
	})
}

func (h *P2PHandler) CreateOrder(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req models.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	advertID, err := uuid.Parse(req.AdvertID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid advert ID"})
		return
	}

	// Get advert details
	var advert models.P2PAdvert
	var merchantUserID uuid.UUID
	err = h.db.QueryRow(`
		SELECT a.id, a.merchant_id, a.side, t.symbol, a.fiat_currency, a.payment_method,
			a.price, a.min_amount, a.max_amount, a.available_amount, m.user_id
		FROM p2p_adverts a
		JOIN tokens t ON a.token_id = t.id
		JOIN p2p_merchants m ON a.merchant_id = m.id
		WHERE a.id = $1 AND a.is_active = true
	`, advertID).Scan(
		&advert.ID, &advert.MerchantID, &advert.Side, &advert.Token,
		&advert.FiatCurrency, &advert.PaymentMethod, &advert.Price,
		&advert.MinAmount, &advert.MaxAmount, &advert.AvailableAmount, &merchantUserID,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Advert not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if req.Amount < advert.MinAmount || req.Amount > advert.AvailableAmount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
		return
	}

	// Get token ID
	var tokenID uuid.UUID
	h.db.QueryRow("SELECT id FROM tokens WHERE symbol = $1", advert.Token).Scan(&tokenID)

	fiatAmount := req.Amount * advert.Price
	buyerDeposit := req.Amount * 0.02  // 2%
	sellerDeposit := req.Amount * 0.03 // 3%

	// Create order
	var orderID uuid.UUID
	err = h.db.QueryRow(`
		INSERT INTO p2p_orders (
			advert_id, buyer_id, seller_id, side, token_id, fiat_currency,
			payment_method, price, amount, fiat_amount, buyer_deposit, seller_deposit, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'PENDING')
		RETURNING id
	`, advertID, userID, merchantUserID, advert.Side, tokenID, advert.FiatCurrency,
		advert.PaymentMethod, advert.Price, req.Amount, fiatAmount, buyerDeposit, sellerDeposit,
	).Scan(&orderID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Update available amount
	h.db.Exec("UPDATE p2p_adverts SET available_amount = available_amount - $1 WHERE id = $2",
		req.Amount, advertID)

	// Notify seller via WebSocket
	h.wsHub.SendToUser(merchantUserID.String(), map[string]interface{}{
		"type":    "NEW_ORDER",
		"orderId": orderID.String(),
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"order": gin.H{
				"id":          orderID,
				"status":      "PENDING",
				"amount":      req.Amount,
				"fiatAmount":  fiatAmount,
				"buyerDeposit": buyerDeposit,
				"sellerDeposit": sellerDeposit,
			},
		},
	})
}

func (h *P2PHandler) GetOrders(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	status := c.Query("status")

	query := `
		SELECT o.id, o.advert_id, o.side, t.symbol, o.fiat_currency, o.payment_method,
			o.price, o.amount, o.fiat_amount, o.status, o.created_at,
			o.buyer_deposit, o.seller_deposit
		FROM p2p_orders o
		JOIN tokens t ON o.token_id = t.id
		WHERE (o.buyer_id = $1 OR o.seller_id = $1)
	`
	args := []interface{}{userID}

	if status != "" {
		query += " AND o.status = $2"
		args = append(args, status)
	}

	query += " ORDER BY o.created_at DESC LIMIT 50"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	defer rows.Close()

	var orders []gin.H
	for rows.Next() {
		var order models.P2POrder
		var token string
		err := rows.Scan(
			&order.ID, &order.AdvertID, &order.Side, &token,
			&order.FiatCurrency, &order.PaymentMethod, &order.Price,
			&order.Amount, &order.FiatAmount, &order.Status, &order.CreatedAt,
			&order.BuyerDeposit, &order.SellerDeposit,
		)
		if err != nil {
			continue
		}

		orders = append(orders, gin.H{
			"id":             order.ID,
			"advertId":       order.AdvertID,
			"side":           order.Side,
			"token":          token,
			"fiatCurrency":   order.FiatCurrency,
			"paymentMethod":   order.PaymentMethod,
			"price":          order.Price,
			"amount":         order.Amount,
			"fiatAmount":     order.FiatAmount,
			"status":         order.Status,
			"buyerDeposit":   order.BuyerDeposit,
			"sellerDeposit":  order.SellerDeposit,
			"createdAt":      order.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    orders,
	})
}

func (h *P2PHandler) GetOrder(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	orderID := c.Param("id")

	var order models.P2POrder
	var token string
	err := h.db.QueryRow(`
		SELECT o.id, o.advert_id, o.side, t.symbol, o.fiat_currency, o.payment_method,
			o.price, o.amount, o.fiat_amount, o.status, o.created_at,
			o.buyer_deposit, o.seller_deposit, o.buyer_id, o.seller_id
		FROM p2p_orders o
		JOIN tokens t ON o.token_id = t.id
		WHERE o.id = $1 AND (o.buyer_id = $2 OR o.seller_id = $2)
	`, orderID, userID).Scan(
		&order.ID, &order.AdvertID, &order.Side, &token,
		&order.FiatCurrency, &order.PaymentMethod, &order.Price,
		&order.Amount, &order.FiatAmount, &order.Status, &order.CreatedAt,
		&order.BuyerDeposit, &order.SellerDeposit, &order.BuyerID, &order.SellerID,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":            order.ID,
			"side":          order.Side,
			"token":         token,
			"fiatCurrency": order.FiatCurrency,
			"price":        order.Price,
			"amount":       order.Amount,
			"fiatAmount":   order.FiatAmount,
			"status":       order.Status,
			"createdAt":    order.CreatedAt,
		},
	})
}

func (h *P2PHandler) MarkAsPaid(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	orderID := c.Param("id")

	var paymentProof string
	if err := c.ShouldBindJSON(&paymentProof); err != nil {
		paymentProof = "Payment confirmed"
	}

	result, err := h.db.Exec(`
		UPDATE p2p_orders 
		SET status = 'PAID', buyer_confirm_time = NOW(), payment_proof = $2
		WHERE id = $1 AND buyer_id = $3 AND status = 'PENDING'
	`, orderID, paymentProof, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found or already processed"})
		return
	}

	// Notify seller
	var sellerID string
	h.db.QueryRow("SELECT seller_id FROM p2p_orders WHERE id = $1", orderID).Scan(&sellerID)
	h.wsHub.SendToUser(sellerID, map[string]interface{}{
		"type":    "ORDER_PAID",
		"orderId": orderID,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "PAID"}})
}

func (h *P2PHandler) ConfirmPayment(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	orderID := c.Param("id")

	result, err := h.db.Exec(`
		UPDATE p2p_orders 
		SET status = 'COMPLETED', release_time = NOW()
		WHERE id = $1 AND seller_id = $2 AND status = 'PAID'
	`, orderID, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found or not in correct state"})
		return
	}

	// Update merchant stats
	h.db.Exec(`
		UPDATE p2p_merchants 
		SET total_trades = total_trades + 1, completed_trades = completed_trades + 1, last_active_at = NOW()
		WHERE user_id = $1
	`, userID)

	// Notify buyer
	var buyerID string
	h.db.QueryRow("SELECT buyer_id FROM p2p_orders WHERE id = $1", orderID).Scan(&buyerID)
	h.wsHub.SendToUser(buyerID, map[string]interface{}{
		"type":    "ORDER_COMPLETED",
		"orderId": orderID,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "COMPLETED"}})
}

func (h *P2PHandler) CancelOrder(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	orderID := c.Param("id")

	var reason string
	c.ShouldBindJSON(&reason)

	result, err := h.db.Exec(`
		UPDATE p2p_orders 
		SET status = 'CANCELLED', cancel_reason = $2, cancel_time = NOW()
		WHERE id = $1 AND (buyer_id = $3 OR seller_id = $3) AND status = 'PENDING'
	`, orderID, reason, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found or cannot be cancelled"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "CANCELLED"}})
}

func (h *P2PHandler) OpenDispute(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	orderID := c.Param("id")

	var reason string
	c.ShouldBindJSON(&reason)

	result, err := h.db.Exec(`
		UPDATE p2p_orders 
		SET dispute_opened = true, dispute_reason = $2, status = 'DISPUTED'
		WHERE id = $1 AND (buyer_id = $3 OR seller_id = $3)
	`, orderID, reason, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open dispute"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "DISPUTED"}})
}

func (h *P2PHandler) GetPaymentMethods(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": []gin.H{
			{"id": "bank_transfer", "name": "Bank Transfer", "type": "bank"},
			{"id": "paypal", "name": "PayPal", "type": "ewallet"},
			{"id": "alipay", "name": "AliPay", "type": "ewallet"},
			{"id": "wechat_pay", "name": "WeChat Pay", "type": "ewallet"},
			{"id": "upi", "name": "UPI", "type": "bank"},
		},
	})
}

func (h *P2PHandler) GetFiatCurrencies(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": []gin.H{
			{"code": "USD", "name": "US Dollar", "symbol": "$"},
			{"code": "EUR", "name": "Euro", "symbol": "€"},
			{"code": "GBP", "name": "British Pound", "symbol": "£"},
			{"code": "CNY", "name": "Chinese Yuan", "symbol": "¥"},
			{"code": "INR", "name": "Indian Rupee", "symbol": "₹"},
		},
	})
}

func (h *P2PHandler) ApplyMerchant(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req models.ApplyMerchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if already a merchant
	var existingID uuid.UUID
	err := h.db.QueryRow("SELECT id FROM p2p_merchants WHERE user_id = $1", userID).Scan(&existingID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Already a merchant"})
		return
	}

	// Create merchant application
	_, err = h.db.Exec(`
		INSERT INTO p2p_merchants (user_id, status, collateral_token, collateral_amount, trader_level)
		VALUES ($1, 'PENDING', $2, $3, 'NEWBIE')
	`, userID, req.CollateralToken, req.CollateralAmount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    gin.H{"status": "PENDING", "message": "Application submitted"},
	})
}

func (h *P2PHandler) GetMerchantProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var merchant models.P2PMerchant
	err := h.db.QueryRow(`
		SELECT id, user_id, status, collateral_token, collateral_amount, trader_level,
			total_trades, total_volume, completed_trades, rating, total_reviews,
			avg_response_time, avg_release_time, security_score, is_verified, joined_at
		FROM p2p_merchants WHERE user_id = $1
	`, userID).Scan(
		&merchant.ID, &merchant.UserID, &merchant.Status, &merchant.CollateralToken,
		&merchant.CollateralAmount, &merchant.TraderLevel, &merchant.TotalTrades,
		&merchant.TotalVolume, &merchant.CompletedTrades, &merchant.Rating,
		&merchant.TotalReviews, &merchant.AvgResponseTime, &merchant.AvgReleaseTime,
		&merchant.SecurityScore, &merchant.IsVerified, &merchant.JoinedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not a merchant"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": merchant})
}

func (h *P2PHandler) AddCollateral(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
		Token  string  `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.db.Exec(`
		UPDATE p2p_merchants 
		SET collateral_amount = collateral_amount + $1
		WHERE user_id = $2
	`, req.Amount, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add collateral"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Merchant not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"collateralAdded": req.Amount}})
}

// Ignore unused import warning
var _ = json.Marshal
