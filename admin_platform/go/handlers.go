package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// AUTH HANDLERS
// ============================================================================

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production, validate against database
	// For now, create a demo admin
	user := map[string]interface{}{
		"id":       uuid.New().String(),
		"email":    req.Email,
		"username": "admin",
		"role":     "super_admin",
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user["id"].(string),
		"email": req.Email,
		"role":  user["role"].(string),
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte("admin-platform-secret-key"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: tokenString,
		User:  user,
	})
}

func handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func handleRefreshToken(c *gin.Context) {
	// Refresh token logic
	c.JSON(http.StatusOK, gin.H{"message": "Token refreshed"})
}

func authMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:] // Remove "Bearer "
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		c.Set("user_id", claims["sub"])
		c.Set("user_email", claims["email"])
		c.Set("user_role", claims["role"])

		c.Next()
	}
}

// ============================================================================
// DASHBOARD HANDLERS
// ============================================================================

func handleGetDashboard(c *gin.Context) {
	dashboard := map[string]interface{}{
		"total_users":       125000,
		"active_users":      45000,
		"kyc_pending":       1250,
		"total_transactions": 875000,
		"volume_24h":        125000000.0,
		"revenue_24h":       125000.0,
		"timestamp":        time.Now().Unix(),
	}
	c.JSON(http.StatusOK, dashboard)
}

func handleGetStats(c *gin.Context) {
	stats := map[string]interface{}{
		"users": map[string]interface{}{
			"total":    125000,
			"active":   45000,
			"new_24h":  1250,
			"banned":   150,
		},
		"transactions": map[string]interface{}{
			"total":     875000,
			"pending":   1250,
			"completed": 870000,
			"failed":    3750,
		},
		"volume": map[string]interface{}{
			"24h":  125000000.0,
			"7d":   875000000.0,
			"30d":  3750000000.0,
		},
	}
	c.JSON(http.StatusOK, stats)
}

// ============================================================================
// USER HANDLERS
// ============================================================================

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	WalletAddr   string `json:"wallet_address"`
	KYCStatus    string `json:"kyc_status"`
	Status       string `json:"status"`
	RiskScore    int    `json:"risk_score"`
	CreatedAt    int64  `json:"created_at"`
}

func handleListUsers(c *gin.Context) {
	users := []User{
		{
			ID:        uuid.New().String(),
			Email:     "user1@example.com",
			Username:  "user1",
			KYCStatus: "approved",
			Status:    "active",
			RiskScore: 10,
		},
		{
			ID:        uuid.New().String(),
			Email:     "user2@example.com",
			Username:  "user2",
			KYCStatus: "pending",
			Status:    "active",
			RiskScore: 25,
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": len(users)})
}

func handleGetUser(c *gin.Context) {
	id := c.Param("id")
	user := User{
		ID:           id,
		Email:        "user@example.com",
		Username:     "sampleuser",
		WalletAddr:   "0x742d35Cc6634C0532925a3b844Bc9e7595f12eB3",
		KYCStatus:    "approved",
		Status:       "active",
		RiskScore:    5,
		CreatedAt:    time.Now().Unix(),
	}
	c.JSON(http.StatusOK, user)
}

func handleUpdateUser(c *gin.Context) {
	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)
	c.JSON(http.StatusOK, gin.H{"message": "User updated", "updates": updates})
}

func handleBanUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User banned"})
}

func handleUnbanUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User unbanned"})
}

func handleSuspendUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User suspended"})
}

// ============================================================================
// KYC HANDLERS
// ============================================================================

func handleListKYC(c *gin.Context) {
	requests := []map[string]interface{}{
		{
			"id":         uuid.New().String(),
			"user_id":    uuid.New().String(),
			"type":       "identity",
			"status":     "pending",
			"created_at": time.Now().Unix(),
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": requests, "total": len(requests)})
}

func handleGetKYC(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":          id,
		"type":        "identity",
		"status":       "pending",
		"documents":    []string{},
		"reviewed_by":  nil,
	})
}

func handleApproveKYC(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "KYC approved"})
}

func handleRejectKYC(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "KYC rejected"})
}

// ============================================================================
// TRANSACTION HANDLERS
// ============================================================================

func handleListTransactions(c *gin.Context) {
	txs := []map[string]interface{}{
		{
			"id":         uuid.New().String(),
			"type":       "deposit",
			"status":     "completed",
			"amount":     "1.5",
			"token":      "ETH",
			"created_at": time.Now().Unix(),
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": txs, "total": len(txs)})
}

func handleGetTransaction(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":         id,
		"type":       "deposit",
		"status":     "completed",
		"amount":     "1.5",
		"token":      "ETH",
		"tx_hash":    "0x123...",
		"created_at": time.Now().Unix(),
	})
}

// ============================================================================
// TRADING PAIR HANDLERS
// ============================================================================

func handleListPairs(c *gin.Context) {
	pairs := []map[string]interface{}{
		{
			"id":          uuid.New().String(),
			"base_token":  "ETH",
			"quote_token": "USDT",
			"chain_id":    1,
			"status":      "active",
			"maker_fee":   "0.001",
			"taker_fee":   "0.001",
		},
		{
			"id":          uuid.New().String(),
			"base_token":  "BTC",
			"quote_token": "USDT",
			"chain_id":    1,
			"status":      "active",
			"maker_fee":   "0.001",
			"taker_fee":   "0.001",
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": pairs, "total": len(pairs)})
}

func handleCreatePair(c *gin.Context) {
	var pair map[string]interface{}
	c.ShouldBindJSON(&pair)
	pair["id"] = uuid.New().String()
	c.JSON(http.StatusCreated, pair)
}

func handleUpdatePair(c *gin.Context) {
	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)
	c.JSON(http.StatusOK, gin.H{"message": "Pair updated"})
}

func handleSuspendPair(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Pair suspended"})
}

func handleResumePair(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Pair resumed"})
}

func handleHaltPair(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Pair halted"})
}

// ============================================================================
// BLOCKCHAIN HANDLERS
// ============================================================================

func handleListBlockchains(c *gin.Context) {
	blockchains := []map[string]interface{}{
		{
			"id":          1,
			"name":        "Ethereum",
			"symbol":      "ETH",
			"chain_type":  "evm",
			"chain_id":     1,
			"is_active":   true,
			"rpc_urls":    []string{"https://eth.llamarpc.com"},
		},
		{
			"id":          56,
			"name":        "BNB Smart Chain",
			"symbol":      "BNB",
			"chain_type":  "evm",
			"chain_id":    56,
			"is_active":   true,
			"rpc_urls":    []string{"https://bsc-dataseed.binance.org"},
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": blockchains})
}

func handleCreateBlockchain(c *gin.Context) {
	var chain map[string]interface{}
	c.ShouldBindJSON(&chain)
	chain["id"] = 999
	c.JSON(http.StatusCreated, chain)
}

func handleUpdateBlockchain(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Blockchain updated"})
}

func handleEnableBlockchain(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Blockchain enabled"})
}

func handleDisableBlockchain(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Blockchain disabled"})
}

// ============================================================================
// FEE HANDLERS
// ============================================================================

func handleListFees(c *gin.Context) {
	fees := []map[string]interface{}{
		{
			"id":         uuid.New().String(),
			"name":        "Trading Fee",
			"fee_type":   "trading",
			"maker_fee":  "0.001",
			"taker_fee":  "0.001",
			"is_active":  true,
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": fees})
}

func handleCreateFee(c *gin.Context) {
	var fee map[string]interface{}
	c.ShouldBindJSON(&fee)
	fee["id"] = uuid.New().String()
	c.JSON(http.StatusCreated, fee)
}

func handleUpdateFee(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Fee updated"})
}

// ============================================================================
// ADMIN HANDLERS
// ============================================================================

func handleListAdmins(c *gin.Context) {
	admins := []map[string]interface{}{
		{
			"id":       uuid.New().String(),
			"email":    "admin@tigerwallet.com",
			"username": "admin",
			"role":     "super_admin",
			"status":   "active",
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": admins})
}

func handleCreateAdmin(c *gin.Context) {
	var admin map[string]interface{}
	c.ShouldBindJSON(&admin)
	
	// Hash password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(admin["password"].(string)), bcrypt.DefaultCost)
	admin["password_hash"] = string(hashedPassword)
	admin["id"] = uuid.New().String()
	
	c.JSON(http.StatusCreated, admin)
}

func handleUpdateAdmin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Admin updated"})
}

func handleDeleteAdmin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Admin deleted"})
}

// ============================================================================
// WHITE LABEL HANDLERS
// ============================================================================

func handleListWhiteLabels(c *gin.Context) {
	clients := []map[string]interface{}{
		{
			"id":            uuid.New().String(),
			"name":          "Client A",
			"domain":        "client-a.tigerwallet.com",
			"status":        "active",
			"plan":          "professional",
			"fee_percent":   20.0,
			"current_users":  5000,
			"max_users":      10000,
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": clients})
}

func handleCreateWhiteLabel(c *gin.Context) {
	var wl map[string]interface{}
	c.ShouldBindJSON(&wl)
	wl["id"] = uuid.New().String()
	wl["status"] = "pending"
	c.JSON(http.StatusCreated, wl)
}

func handleUpdateWhiteLabel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "White label updated"})
}

func handleDeleteWhiteLabel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "White label deleted"})
}

func handleApproveWhiteLabel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "White label approved"})
}

func handleSuspendWhiteLabel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "White label suspended"})
}

// ============================================================================
// AUDIT LOG HANDLERS
// ============================================================================

func handleListAuditLogs(c *gin.Context) {
	logs := []map[string]interface{}{
		{
			"id":          uuid.New().String(),
			"admin_id":    uuid.New().String(),
			"action":      "user_ban",
			"details":     "Banned user xyz",
			"ip_address":  "192.168.1.1",
			"created_at":  time.Now().Unix(),
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": logs, "total": len(logs)})
}

// ============================================================================
// SETTINGS HANDLERS
// ============================================================================

func handleGetSettings(c *gin.Context) {
	settings := map[string]interface{}{
		"platform_name":     "TigerWallet",
		"maintenance_mode":  false,
		"registration_open": true,
		"kyc_required":     true,
	}
	c.JSON(http.StatusOK, settings)
}

func handleUpdateSettings(c *gin.Context) {
	var settings map[string]interface{}
	c.ShouldBindJSON(&settings)
	c.JSON(http.StatusOK, gin.H{"message": "Settings updated", "settings": settings})
}
