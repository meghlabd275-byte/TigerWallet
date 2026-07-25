// TigerWallet Institutional Custody Service
// Enterprise-grade custody solution with MPC, multi-sig, and compliance
// For institutional clients, brokers, and enterprise users

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort    string `json:"server_port"`
	DBHost       string `json:"db_host"`
	DBPort       string `json:"db_port"`
	DBUser       string `json:"db_user"`
	DBPassword   string `json:"db_password"`
	DBName       string `json:"db_name"`
	RedisHost    string `json:"redis_host"`
	RedisPort    string `json:"redis_port"`
}

// ============================================================================
// Data Models
// ============================================================================

// Institution represents an institutional client
type Institution struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Name             string    `json:"name"`
	LegalName        string    `json:"legal_name"`
	Email            string    `json:"email"`
	Phone            string    `json:"phone"`
	Address          string    `json:"address"`
	Country          string    `json:"country"`
	KYCStatus        string    `json:"kyc_status"` // PENDING, VERIFIED, REJECTED
	KYCLevel         int       `json:"kyc_level"`
	Tier             string    `json:"tier"` // TIER_1, TIER_2, TIER_3
	RiskLevel        string    `json:"risk_level"`
	MaxWithdrawalDaily float64 `json:"max_withdrawal_daily"`
	MaxWithdrawalMonthly float64 `json:"max_withdrawal_monthly"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CustodyAccount represents a custody account
type CustodyAccount struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	InstitutionID    uint      `gorm:"index" json:"institution_id"`
	AccountName      string    `json:"account_name"`
	AccountType      string    `json:"account_type"` // HOT, WARM, COLD
	WalletAddress    string    `gorm:"uniqueIndex" json:"wallet_address"`
	ChainID          int64     `json:"chain_id"`
	TotalBalance     string    `json:"total_balance"`
	AvailableBalance string    `json:"available_balance"`
	ReservedBalance  string    `json:"reserved_balance"`
	Status           string    `json:"status"` // ACTIVE, FROZEN, CLOSED
	MPCCoreID        string    `json:"mpc_core_id"`
	Threshold        int       `json:"threshold"` // required signatures
	TotalSigners     int       `json:"total_signers"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Signer represents a key signer for multi-sig
type Signer struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	AccountID       uint      `gorm:"index" json:"account_id"`
	SignerID        string    `gorm:"uniqueIndex" json:"signer_id"`
	SignerType      string    `json:"signer_type"` // MPC, HSM, HARDWARE, API
	PublicKey       string    `json:"public_key"`
	Address         string    `json:"address"`
	Status          string    `json:"status"` // ACTIVE, INACTIVE, REVOKED
	Permissions     string    `json:"permissions"` // JSON array
	LastUsedAt     *time.Time `json:"last_used_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// TransactionRequest represents a custody transaction request
type TransactionRequest struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	RequestID       string    `gorm:"uniqueIndex" json:"request_id"`
	InstitutionID   uint      `gorm:"index" json:"institution_id"`
	AccountID       uint      `gorm:"index" json:"account_id"`
	ToAddress       string    `json:"to_address"`
	Token           string    `json:"token"`
	Amount          string    `json:"amount"`
	Fee             string    `json:"fee"`
	TransactionType string    `json:"transaction_type"` // WITHDRAWAL, INTERNAL, TRANSFER
	Status          string    `json:"status"` // PENDING, APPROVED, REJECTED, EXECUTED, FAILED
	ApprovedBy      []string  `gorm:"-" json:"approved_by"`
	ApprovedAt      []time.Time `gorm:"-" json:"approved_at"`
	RejectedBy      string    `json:"rejected_by"`
	RejectedAt      *time.Time `json:"rejected_at"`
	ExecutedTxHash  string    `json:"executed_tx_hash"`
	ExecutedAt      *time.Time `json:"executed_at"`
	ChainID         int64     `json:"chain_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Approval represents transaction approval
type Approval struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	RequestID      string    `gorm:"index" json:"request_id"`
	SignerID       string    `json:"signer_id"`
	Approved       bool      `json:"approved"`
	Signature      string    `json:"signature"`
	IPAddress      string    `json:"ip_address"`
	ApprovalTime   time.Time `json:"approval_time"`
}

// ComplianceLog represents compliance audit log
type ComplianceLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	InstitutionID uint      `json:"institution_id"`
	AccountID    uint      `json:"account_id"`
	Action       string    `json:"action"`
	Details      string    `json:"details"`
	PerformedBy  string    `json:"performed_by"`
	IPAddress    string    `json:"ip_address"`
	Timestamp    time.Time `json:"timestamp"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type CustodyService struct {
	db     *gorm.DB
	redis  *redis.Client
	config Config
	mu     sync.RWMutex
}

func NewCustodyService(config Config) (*CustodyService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&Institution{},
		&CustodyAccount{},
		&Signer{},
		&TransactionRequest{},
		&Approval{},
		&ComplianceLog{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &CustodyService{
		db:     db,
		redis:  rdb,
		config: config,
	}

	return service, nil
}

// ============================================================================
// Institution Management
// ============================================================================

type CreateInstitutionRequest struct {
	Name             string  `json:"name" binding:"required"`
	LegalName        string  `json:"legal_name"`
	Email            string  `json:"email" binding:"required"`
	Phone            string  `json:"phone"`
	Address          string  `json:"address"`
	Country          string  `json:"country" binding:"required"`
	Tier             string  `json:"tier"`
}

func (s *CustodyService) CreateInstitution(ctx *gin.Context) {
	var req CreateInstitutionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	tier := req.Tier
	if tier == "" {
		tier = "TIER_1"
	}

	// Set limits based on tier
	maxDaily, maxMonthly := s.getWithdrawalLimits(tier)

	institution := Institution{
		Name:                  req.Name,
		LegalName:             req.LegalName,
		Email:                 req.Email,
		Phone:                 req.Phone,
		Address:               req.Address,
		Country:               req.Country,
		KYCStatus:             "PENDING",
		KYCLevel:              0,
		Tier:                  tier,
		RiskLevel:             "MEDIUM",
		MaxWithdrawalDaily:    maxDaily,
		MaxWithdrawalMonthly:  maxMonthly,
		IsActive:              true,
	}

	if err := s.db.Create(&institution).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to create institution"})
		return
	}

	// Log compliance event
	s.logCompliance(institution.ID, 0, "CREATE_INSTITUTION", fmt.Sprintf("Created institution: %s", req.Name), "system", ctx.ClientIP())

	ctx.JSON(200, gin.H{
		"success":       true,
		"institution_id": institution.ID,
		"status":       "PENDING_KYC",
	})
}

func (s *CustodyService) getWithdrawalLimits(tier string) (daily, monthly float64) {
	switch tier {
	case "TIER_1":
		return 10000, 100000
	case "TIER_2":
		return 100000, 1000000
	case "TIER_3":
		return 1000000, 10000000
	default:
		return 10000, 100000
	}
}

// ============================================================================
// Account Management
// ============================================================================

type CreateAccountRequest struct {
	InstitutionID uint   `json:"institution_id" binding:"required"`
	AccountName  string `json:"account_name" binding:"required"`
	AccountType  string `json:"account_type" binding:"required"` // HOT, WARM, COLD
	ChainID      int64  `json:"chain_id" binding:"required"`
	Threshold    int    `json:"threshold" binding:"required"`
}

func (s *CustodyService) CreateAccount(ctx *gin.Context) {
	var req CreateAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Verify institution exists and is active
	var institution Institution
	if err := s.db.First(&institution, req.InstitutionID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Institution not found"})
		return
	}

	if !institution.IsActive {
		ctx.JSON(400, gin.H{"success": false, "error": "Institution is not active"})
		return
	}

	// Generate MPC wallet address (simulated)
	walletAddress := s.generateMPCAddress(req.ChainID)

	account := CustodyAccount{
		InstitutionID:    req.InstitutionID,
		AccountName:      req.AccountName,
		AccountType:      req.AccountType,
		WalletAddress:   walletAddress,
		ChainID:         req.ChainID,
		TotalBalance:    "0",
		AvailableBalance: "0",
		ReservedBalance: "0",
		Status:          "ACTIVE",
		MPCCoreID:       s.generateMPCCoreID(),
		Threshold:       req.Threshold,
		TotalSigners:    req.Threshold, // For threshold, we start with equal signers
	}

	if err := s.db.Create(&account).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to create account"})
		return
	}

	// Log compliance event
	s.logCompliance(req.InstitutionID, account.ID, "CREATE_ACCOUNT", 
		fmt.Sprintf("Created %s account: %s", req.AccountType, req.AccountName), "system", ctx.ClientIP())

	ctx.JSON(200, gin.H{
		"success":      true,
		"account_id":  account.ID,
		"wallet_address": account.WalletAddress,
		"mpc_core_id": account.MPCCoreID,
		"threshold":    account.Threshold,
	})
}

// ============================================================================
// Signer Management
// ============================================================================

type AddSignerRequest struct {
	AccountID   uint   `json:"account_id" binding:"required"`
	SignerType  string `json:"signer_type" binding:"required"` // MPC, HSM, HARDWARE, API
	PublicKey   string `json:"public_key" binding:"required"`
	Address     string `json:"address"`
	Permissions string  `json:"permissions"` // JSON array
}

func (s *CustodyService) AddSigner(ctx *gin.Context) {
	var req AddSignerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Verify account exists
	var account CustodyAccount
	if err := s.db.First(&account, req.AccountID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Account not found"})
		return
	}

	signerID := s.generateSignerID()
	signer := Signer{
		AccountID:   req.AccountID,
		SignerID:   signerID,
		SignerType: req.SignerType,
		PublicKey:  req.PublicKey,
		Address:    req.Address,
		Status:     "ACTIVE",
		Permissions: req.Permissions,
	}

	if err := s.db.Create(&signer).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to add signer"})
		return
	}

	// Update account signers count
	account.TotalSigners++
	s.db.Save(&account)

	ctx.JSON(200, gin.H{
		"success":   true,
		"signer_id": signer.SignerID,
		"status":   "ACTIVE",
	})
}

// ============================================================================
// Transaction Requests
// ============================================================================

type CreateTransactionRequest struct {
	InstitutionID   uint   `json:"institution_id" binding:"required"`
	AccountID       uint   `json:"account_id" binding:"required"`
	ToAddress       string `json:"to_address" binding:"required"`
	Token           string `json:"token" binding:"required"`
	Amount          string `json:"amount" binding:"required"`
	TransactionType string `json:"transaction_type" binding:"required"`
	ChainID         int64  `json:"chain_id"`
}

func (s *CustodyService) CreateTransactionRequest(ctx *gin.Context) {
	var req CreateTransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Verify account
	var account CustodyAccount
	if err := s.db.First(&account, req.AccountID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Account not found"})
		return
	}

	// Verify institution owns account
	if account.InstitutionID != req.InstitutionID {
		ctx.JSON(403, gin.H{"success": false, "error": "Account does not belong to institution"})
		return
	}

	// Check withdrawal limits
	if req.TransactionType == "WITHDRAWAL" {
		var institution Institution
		s.db.First(&institution, req.InstitutionID)
		
		// Calculate today's withdrawals
		var todayTotal float64
		s.db.Model(&TransactionRequest{}).
			Where("institution_id = ? AND transaction_type = ? AND status = ? AND DATE(created_at) = ?",
				req.InstitutionID, "WITHDRAWAL", "EXECUTED", time.Now().Format("2006-01-02")).
			Select("COALESCE(SUM(CAST(amount AS FLOAT)), 0)").
			Scan(&todayTotal)

		amountFloat := 0.0
		fmt.Sscanf(req.Amount, "%f", &amountFloat)

		if todayTotal+amountFloat > institution.MaxWithdrawalDaily {
			ctx.JSON(400, gin.H{"success": false, "error": "Daily withdrawal limit exceeded"})
			return
		}
	}

	// Calculate fee
	fee := s.calculateTransactionFee(req.Token, req.Amount, req.TransactionType)

	requestID := s.generateRequestID()
	txRequest := TransactionRequest{
		RequestID:       requestID,
		InstitutionID:   req.InstitutionID,
		AccountID:       req.AccountID,
		ToAddress:       req.ToAddress,
		Token:           req.Token,
		Amount:          req.Amount,
		Fee:             fee,
		TransactionType: req.TransactionType,
		Status:          "PENDING",
		ChainID:         req.ChainID,
	}

	if err := s.db.Create(&txRequest).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to create transaction request"})
		return
	}

	// Log compliance event
	s.logCompliance(req.InstitutionID, req.AccountID, "CREATE_TX_REQUEST", 
		fmt.Sprintf("Created %s request: %s %s", req.TransactionType, req.Amount, req.Token), 
		"system", ctx.ClientIP())

	ctx.JSON(200, gin.H{
		"success":    true,
		"request_id": requestID,
		"fee":        fee,
		"status":     "PENDING_APPROVAL",
	})
}

func (s *CustodyService) calculateTransactionFee(token, amount, txType string) string {
	amountFloat := 0.0
	fmt.Sscanf(amount, "%f", &amountFloat)

	var fee float64
	if txType == "INTERNAL" {
		fee = 0 // Free internal transfers
	} else {
		// Base fee + percentage
		fee = 1.0 + amountFloat*0.001
	}

	return fmt.Sprintf("%.6f", fee)
}

// ============================================================================
// Approval Management
// ============================================================================

type ApproveTransactionRequest struct {
	RequestID string `json:"request_id" binding:"required"`
	SignerID  string `json:"signer_id" binding:"required"`
	Approved  bool   `json:"approved"`
	Signature string `json:"signature"`
}

func (s *CustodyService) ApproveTransaction(ctx *gin.Context) {
	var req ApproveTransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var txRequest TransactionRequest
	if err := s.db.Where("request_id = ?", req.RequestID).First(&txRequest).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Transaction request not found"})
		return
	}

	if txRequest.Status != "PENDING" {
		ctx.JSON(400, gin.H{"success": false, "error": "Transaction is not pending"})
		return
	}

	// Verify signer
	var signer Signer
	if err := s.db.Where("signer_id = ? AND account_id = ?", req.SignerID, txRequest.AccountID).First(&signer).Error; err != nil {
		ctx.JSON(403, gin.H{"success": false, "error": "Invalid signer"})
		return
	}

	// Record approval
	approval := Approval{
		RequestID:    req.RequestID,
		SignerID:     req.SignerID,
		Approved:     req.Approved,
		Signature:    req.Signature,
		IPAddress:    ctx.ClientIP(),
		ApprovalTime: time.Now(),
	}
	s.db.Create(&approval)

	if !req.Approved {
		txRequest.Status = "REJECTED"
		txRequest.RejectedBy = req.SignerID
		now := time.Now()
		txRequest.RejectedAt = &now
		s.db.Save(&txRequest)
		
		s.logCompliance(txRequest.InstitutionID, txRequest.AccountID, "TX_REJECTED",
			fmt.Sprintf("Transaction %s rejected by %s", req.RequestID, req.SignerID), req.SignerID, ctx.ClientIP())

		ctx.JSON(200, gin.H{"success": true, "status": "REJECTED"})
		return
	}

	// Update signer last used
	now := time.Now()
	signer.LastUsedAt = &now
	s.db.Save(&signer)

	// Count approvals
	var approvalCount int64
	s.db.Model(&Approval{}).Where("request_id = ? AND approved = ?", req.RequestID, true).Count(&approvalCount)

	// Get account threshold
	var account CustodyAccount
	s.db.First(&account, txRequest.AccountID)

	// Check if we have enough approvals
	if int(approvalCount) >= account.Threshold {
		txRequest.Status = "APPROVED"
		
		// Execute transaction (simulated)
		txHash := s.executeTransaction(&txRequest)
		txRequest.Status = "EXECUTED"
		txRequest.ExecutedTxHash = txHash
		txRequest.ExecutedAt = &now
		
		s.logCompliance(txRequest.InstitutionID, txRequest.AccountID, "TX_EXECUTED",
			fmt.Sprintf("Transaction %s executed with hash %s", req.RequestID, txHash), "system", "")
	}

	s.db.Save(&txRequest)

	ctx.JSON(200, gin.H{
		"success":       true,
		"status":       txRequest.Status,
		"approvals":    approvalCount,
		"required":     account.Threshold,
		"executed_hash": txRequest.ExecutedTxHash,
	})
}

func (s *CustodyService) executeTransaction(tx *TransactionRequest) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%d", tx.ToAddress, tx.Token, tx.Amount, tx.RequestID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}

// ============================================================================
// Queries
// ============================================================================

func (s *CustodyService) GetInstitutions(ctx *gin.Context) {
	var institutions []Institution
	s.db.Find(&institutions)

	ctx.JSON(200, gin.H{"institutions": institutions})
}

func (s *CustodyService) GetAccounts(ctx *gin.Context) {
	institutionID := ctx.Query("institution_id")

	query := s.db.Model(&CustodyAccount{})
	if institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}

	var accounts []CustodyAccount
	query.Find(&accounts)

	ctx.JSON(200, gin.H{"accounts": accounts})
}

func (s *CustodyService) GetTransactionRequests(ctx *gin.Context) {
	institutionID := ctx.Query("institution_id")
	status := ctx.Query("status")

	query := s.db.Model(&TransactionRequest{})
	if institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var requests []TransactionRequest
	query.Order("created_at DESC").Find(&requests)

	ctx.JSON(200, gin.H{"requests": requests})
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *CustodyService) generateMPCAddress(chainID int64) string {
	data := fmt.Sprintf("mpc:%d:%d", chainID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])[0:40]
}

func (s *CustodyService) generateMPCCoreID() string {
	data := fmt.Sprintf("core:%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "mpc_" + hex.EncodeToString(hash[:])[0:16]
}

func (s *CustodyService) generateSignerID() string {
	data := fmt.Sprintf("signer:%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "sig_" + hex.EncodeToString(hash[:])[0:16]
}

func (s *CustodyService) generateRequestID() string {
	data := fmt.Sprintf("req:%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "tx_" + hex.EncodeToString(hash[:])[0:20]
}

func (s *CustodyService) logCompliance(institutionID, accountID uint, action, details, performedBy, ipAddress string) {
	log := ComplianceLog{
		InstitutionID: institutionID,
		AccountID:    accountID,
		Action:       action,
		Details:      details,
		PerformedBy:  performedBy,
		IPAddress:    ipAddress,
		Timestamp:    time.Now(),
	}
	s.db.Create(&log)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort: "8097",
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet_custody"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewCustodyService(config)
	if err != nil {
		fmt.Printf("Failed to start custody service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := router.Group("/api/v1/custody")
	{
		api.GET("/institutions", service.GetInstitutions)
		api.POST("/institutions", service.CreateInstitution)
		api.GET("/accounts", service.GetAccounts)
		api.POST("/accounts", service.CreateAccount)
		api.POST("/signers", service.AddSigner)
		api.GET("/transactions", service.GetTransactionRequests)
		api.POST("/transactions", service.CreateTransactionRequest)
		api.POST("/approve", service.ApproveTransaction)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "custody"})
	})

	go func() {
		fmt.Printf("Institutional custody service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down custody service...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
