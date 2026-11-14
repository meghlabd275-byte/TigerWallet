// TigerWallet Institutional Custody Service
// Enterprise-grade custody solution with MPC, multi-sig, and compliance
// For institutional clients, brokers, and enterprise users

package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort string `json:"server_port"`
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	RedisHost  string `json:"redis_host"`
	RedisPort  string `json:"redis_port"`
	// MPCBackendURL is the canonical go/mpc TSS service (default :9099). The
	// custody service delegates key creation + signing to it and never
	// fabricates an address or signature. Empty => CreateAccount/Execute fail
	// honestly with an "action_required" status.
	MPCBackendURL string `json:"mpc_backend_url"`
	// MPCServiceToken is the bearer token presented to the MPC service, which
	// requires an authenticated caller for every signing and export route.
	MPCServiceToken string `json:"mpc_service_token"`
	WalletAPIURL    string `json:"wallet_api_url"`
	// ServiceToken authenticates this service to the wallet_api custody relay
	// (/api/v1/service/tx/*). It must match the wallet_api SERVICE_BROADCAST_TOKEN.
	// Empty => executeTransaction fails closed rather than calling unauthenticated.
	ServiceToken string `json:"service_token"`
	// APIKey gates every /api/v1/custody route. This service creates and
	// approves withdrawals that move real funds, so an unauthenticated caller
	// must never reach it. Required at startup: a credential has no safe default.
	APIKey string `json:"api_key"`
}

// ============================================================================
// Data Models
// ============================================================================

// Institution represents an institutional client
type Institution struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	Name                 string    `json:"name"`
	LegalName            string    `json:"legal_name"`
	Email                string    `json:"email"`
	Phone                string    `json:"phone"`
	Address              string    `json:"address"`
	Country              string    `json:"country"`
	KYCStatus            string    `json:"kyc_status"` // PENDING, VERIFIED, REJECTED
	KYCLevel             int       `json:"kyc_level"`
	Tier                 string    `json:"tier"` // TIER_1, TIER_2, TIER_3
	RiskLevel            string    `json:"risk_level"`
	MaxWithdrawalDaily   float64   `json:"max_withdrawal_daily"`
	MaxWithdrawalMonthly float64   `json:"max_withdrawal_monthly"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// CustodyAccount represents a custody account
type CustodyAccount struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	InstitutionID    uint   `gorm:"index" json:"institution_id"`
	AccountName      string `json:"account_name"`
	AccountType      string `json:"account_type"` // HOT, WARM, COLD
	WalletAddress    string `gorm:"uniqueIndex" json:"wallet_address"`
	ChainID          int64  `json:"chain_id"`
	TotalBalance     string `json:"total_balance"`
	AvailableBalance string `json:"available_balance"`
	ReservedBalance  string `json:"reserved_balance"`
	Status           string `json:"status"` // ACTIVE, FROZEN, CLOSED
	MPCCoreID        string `json:"mpc_core_id"`
	// MPCKeyID is the real keyId returned by the canonical go/mpc TSS backend
	// (POST /api/v1/mpc/create). It identifies the distributed key material
	// used to sign transactions for this custody account. Empty until the MPC
	// backend has created the key.
	MPCKeyID     string    `json:"mpc_key_id"`
	Threshold    int       `json:"threshold"` // required signatures
	TotalSigners int       `json:"total_signers"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Signer represents a key signer for multi-sig
type Signer struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	AccountID   uint       `gorm:"index" json:"account_id"`
	SignerID    string     `gorm:"uniqueIndex" json:"signer_id"`
	SignerType  string     `json:"signer_type"` // MPC, HSM, HARDWARE, API
	PublicKey   string     `json:"public_key"`
	Address     string     `json:"address"`
	Status      string     `json:"status"`      // ACTIVE, INACTIVE, REVOKED
	Permissions string     `json:"permissions"` // JSON array
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TransactionRequest represents a custody transaction request
type TransactionRequest struct {
	ID              uint        `gorm:"primaryKey" json:"id"`
	RequestID       string      `gorm:"uniqueIndex" json:"request_id"`
	InstitutionID   uint        `gorm:"index" json:"institution_id"`
	AccountID       uint        `gorm:"index" json:"account_id"`
	ToAddress       string      `json:"to_address"`
	Token           string      `json:"token"`
	Amount          string      `json:"amount"`
	Fee             string      `json:"fee"`
	TransactionType string      `json:"transaction_type"` // WITHDRAWAL, INTERNAL, TRANSFER
	Status          string      `json:"status"`           // PENDING, APPROVED, REJECTED, EXECUTED, FAILED
	ApprovedBy      []string    `gorm:"-" json:"approved_by"`
	ApprovedAt      []time.Time `gorm:"-" json:"approved_at"`
	RejectedBy      string      `json:"rejected_by"`
	RejectedAt      *time.Time  `json:"rejected_at"`
	ExecutedTxHash  string      `json:"executed_tx_hash"`
	ExecutedAt      *time.Time  `json:"executed_at"`
	ChainID         int64       `json:"chain_id"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// Approval represents transaction approval
type Approval struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	RequestID    string    `gorm:"index" json:"request_id"`
	SignerID     string    `json:"signer_id"`
	Approved     bool      `json:"approved"`
	Signature    string    `json:"signature"`
	IPAddress    string    `json:"ip_address"`
	ApprovalTime time.Time `json:"approval_time"`
}

// ComplianceLog represents compliance audit log
type ComplianceLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	InstitutionID uint      `json:"institution_id"`
	AccountID     uint      `json:"account_id"`
	Action        string    `json:"action"`
	Details       string    `json:"details"`
	PerformedBy   string    `json:"performed_by"`
	IPAddress     string    `json:"ip_address"`
	Timestamp     time.Time `json:"timestamp"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type CustodyService struct {
	db         *gorm.DB
	redis      *redis.Client
	config     Config
	mu         sync.RWMutex
	httpClient *http.Client
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
		db:         db,
		redis:      rdb,
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	return service, nil
}

// ============================================================================
// Institution Management
// ============================================================================

type CreateInstitutionRequest struct {
	Name      string `json:"name" binding:"required"`
	LegalName string `json:"legal_name"`
	Email     string `json:"email" binding:"required"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Country   string `json:"country" binding:"required"`
	Tier      string `json:"tier"`
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
		Name:                 req.Name,
		LegalName:            req.LegalName,
		Email:                req.Email,
		Phone:                req.Phone,
		Address:              req.Address,
		Country:              req.Country,
		KYCStatus:            "PENDING",
		KYCLevel:             0,
		Tier:                 tier,
		RiskLevel:            "MEDIUM",
		MaxWithdrawalDaily:   maxDaily,
		MaxWithdrawalMonthly: maxMonthly,
		IsActive:             true,
	}

	if err := s.db.Create(&institution).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to create institution"})
		return
	}

	// Log compliance event
	s.logCompliance(institution.ID, 0, "CREATE_INSTITUTION", fmt.Sprintf("Created institution: %s", req.Name), "system", ctx.ClientIP())

	ctx.JSON(200, gin.H{
		"success":        true,
		"institution_id": institution.ID,
		"status":         "PENDING_KYC",
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
	AccountName   string `json:"account_name" binding:"required"`
	AccountType   string `json:"account_type" binding:"required"` // HOT, WARM, COLD
	ChainID       int64  `json:"chain_id" binding:"required"`
	Threshold     int    `json:"threshold" binding:"required"`
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

	// Create a real MPC key via the canonical go/mpc TSS backend. The backend
	// generates a distributed secp256k1 key and returns the wallet address +
	// a keyId we store for later signing. If no MPC backend is configured we
	// fail honestly instead of fabricating an address.
	keyID, walletAddress, err := s.createMPCWallet(req.Threshold)
	if err != nil {
		ctx.JSON(503, gin.H{
			"success":         false,
			"error":           err.Error(),
			"action_required": "configure_mpc_backend",
		})
		return
	}

	account := CustodyAccount{
		InstitutionID:    req.InstitutionID,
		AccountName:      req.AccountName,
		AccountType:      req.AccountType,
		WalletAddress:    walletAddress,
		ChainID:          req.ChainID,
		TotalBalance:     "0",
		AvailableBalance: "0",
		ReservedBalance:  "0",
		Status:           "ACTIVE",
		MPCCoreID:        "mpc_" + uuid.NewString(),
		MPCKeyID:         keyID,
		Threshold:        req.Threshold,
		TotalSigners:     req.Threshold, // For threshold, we start with equal signers
	}

	if err := s.db.Create(&account).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to create account"})
		return
	}

	// Log compliance event
	s.logCompliance(req.InstitutionID, account.ID, "CREATE_ACCOUNT",
		fmt.Sprintf("Created %s account: %s", req.AccountType, req.AccountName), "system", ctx.ClientIP())

	ctx.JSON(200, gin.H{
		"success":        true,
		"account_id":     account.ID,
		"wallet_address": account.WalletAddress,
		"mpc_core_id":    account.MPCCoreID,
		"threshold":      account.Threshold,
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
	Permissions string `json:"permissions"` // JSON array
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

	signerID := "sig_" + uuid.NewString()
	signer := Signer{
		AccountID:   req.AccountID,
		SignerID:    signerID,
		SignerType:  req.SignerType,
		PublicKey:   req.PublicKey,
		Address:     req.Address,
		Status:      "ACTIVE",
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
		"status":    "ACTIVE",
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

	requestID := "tx_" + uuid.NewString()
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

		// Sign + broadcast the transaction via the real MPC TSS backend. When
		// no MPC backend is configured the request stays APPROVED and we
		// surface an honest action_required status so an operator can sign it
		// with the custody key -- we never fabricate a tx hash.
		txHash, execErr := s.executeTransaction(&account, &txRequest)
		if execErr != nil {
			txRequest.Status = "APPROVED"
			s.db.Save(&txRequest)
			s.logCompliance(txRequest.InstitutionID, txRequest.AccountID, "TX_APPROVED_PENDING_BROADCAST",
				fmt.Sprintf("Transaction %s approved; broadcast pending: %v", req.RequestID, execErr), "system", "")
			ctx.JSON(202, gin.H{
				"success":         true,
				"status":          "APPROVED",
				"approvals":       approvalCount,
				"required":        account.Threshold,
				"action_required": "broadcast_transaction",
				"error":           execErr.Error(),
			})
			return
		}
		txRequest.Status = "EXECUTED"
		txRequest.ExecutedTxHash = txHash
		txRequest.ExecutedAt = &now

		s.logCompliance(txRequest.InstitutionID, txRequest.AccountID, "TX_EXECUTED",
			fmt.Sprintf("Transaction %s executed with hash %s", req.RequestID, txHash), "system", "")
	}

	s.db.Save(&txRequest)

	ctx.JSON(200, gin.H{
		"success":       true,
		"status":        txRequest.Status,
		"approvals":     approvalCount,
		"required":      account.Threshold,
		"executed_hash": txRequest.ExecutedTxHash,
	})
}

// executeTransaction performs a real MPC custody withdrawal in two phases
// against the canonical wallet_api:
//
//  1. POST /api/v1/service/tx/prepare  -> the backend fetches the nonce and
//     gas price from the chain and returns the unsigned transaction plus the
//     exact 32-byte hash that must be signed.
//  2. The MPC TSS backend signs that hash with the account's distributed key
//     shares (POST /api/v1/mpc/sign). The key never exists in one place.
//  3. POST /api/v1/service/tx/submit   -> the backend rebuilds the identical
//     transaction, verifies the signature recovers the custody account's own
//     address, and broadcasts it.
//
// Signing a pre-built transaction hash is the only correct design for an MPC
// signer: a TSS backend produces a secp256k1 signature over a digest, and it
// cannot produce the full signed transaction itself. It NEVER fabricates a
// hash; every failure returns an error so the caller can surface an honest
// action_required status.
func (s *CustodyService) executeTransaction(account *CustodyAccount, tx *TransactionRequest) (string, error) {
	if s.config.MPCBackendURL == "" {
		return "", fmt.Errorf("MPC backend not configured (set MPC_BACKEND_URL)")
	}
	if account.MPCKeyID == "" {
		return "", fmt.Errorf("custody account has no MPC key (keyId empty)")
	}
	if s.config.WalletAPIURL == "" {
		return "", fmt.Errorf("wallet_api not configured (set WALLET_API_URL)")
	}
	if s.config.ServiceToken == "" {
		return "", fmt.Errorf("SERVICE_BROADCAST_TOKEN not configured; the wallet_api relay refuses unauthenticated calls")
	}

	// Phase 1: ask the wallet backend to build the transaction. It owns the
	// chain registry, the nonce lookup and the fee logic, so the custody
	// service never has to duplicate any of it.
	prepared, err := s.prepareTransfer(account, tx)
	if err != nil {
		return "", fmt.Errorf("prepare failed: %w", err)
	}
	if prepared.SigningHash == "" {
		return "", fmt.Errorf("wallet_api returned no signing_hash")
	}

	// Phase 2: sign the digest with the distributed key.
	sig, err := s.mpcSign(account.MPCKeyID, prepared.SigningHash)
	if err != nil {
		return "", fmt.Errorf("MPC sign failed: %w", err)
	}

	// Phase 3: submit. The backend re-derives the hash from the fields we
	// echo back and rejects the call if it does not match what we signed.
	txHash, err := s.submitTransfer(prepared, sig)
	if err != nil {
		return "", fmt.Errorf("submit failed: %w", err)
	}
	return txHash, nil
}

// preparedTx mirrors the wallet_api /service/tx/prepare response.
type preparedTx struct {
	ChainID        int64  `json:"chain_id"`
	From           string `json:"from"`
	To             string `json:"to"`
	ValueWei       string `json:"value_wei"`
	Data           string `json:"data"`
	GasLimit       uint64 `json:"gas_limit"`
	TxType         string `json:"tx_type"`
	Nonce          uint64 `json:"nonce"`
	GasPriceWei    string `json:"gas_price_wei,omitempty"`
	MaxFeeWei      string `json:"max_fee_wei,omitempty"`
	MaxPriorityWei string `json:"max_priority_fee_wei,omitempty"`
	SigningHash    string `json:"signing_hash"`
}

// tokenToWei converts a decimal token amount into the smallest unit for the
// chain. Custody transfers must not lose precision, so the conversion is done
// on integers: the fractional part is scaled by the chain's decimals.
//
// Only a plain non-negative decimal is accepted. Anything else (letters, a
// second decimal point, a sign) is rejected rather than coerced, because a
// malformed amount must never be turned into a number the operator did not
// authorise.
func tokenToWei(amount string, decimals int) (string, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return "", fmt.Errorf("amount is empty")
	}
	if strings.HasPrefix(amount, "-") || strings.HasPrefix(amount, "+") {
		return "", fmt.Errorf("amount must not be signed")
	}
	if decimals < 0 || decimals > 36 {
		decimals = 18
	}

	intPart, fracPart := amount, ""
	if i := strings.Index(amount, "."); i >= 0 {
		intPart, fracPart = amount[:i], amount[i+1:]
		if strings.Contains(fracPart, ".") {
			return "", fmt.Errorf("amount has more than one decimal point")
		}
	}
	if intPart == "" {
		intPart = "0"
	}
	if !isDigits(intPart) || !isDigits(fracPart) {
		return "", fmt.Errorf("amount must be a decimal number")
	}

	if len(fracPart) > decimals {
		// Truncate rather than round: never move more than authorised.
		fracPart = fracPart[:decimals]
	}
	for len(fracPart) < decimals {
		fracPart += "0"
	}
	scaled := strings.TrimLeft(intPart+fracPart, "0")
	if scaled == "" {
		scaled = "0"
	}
	return scaled, nil
}

// isDigits reports whether s consists only of ASCII digits. An empty string is
// accepted so the caller can treat a missing fractional part as zero.
func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// prepareTransfer asks the canonical wallet_api to build the withdrawal.
func (s *CustodyService) prepareTransfer(account *CustodyAccount, tx *TransactionRequest) (*preparedTx, error) {
	wei, err := tokenToWei(tx.Amount, 18)
	if err != nil {
		return nil, err
	}
	req := map[string]interface{}{
		"chain_id":  account.ChainID,
		"from":      account.WalletAddress,
		"to":        tx.ToAddress,
		"value_wei": wei,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := s.postJSON(s.config.WalletAPIURL+"/api/v1/service/tx/prepare", string(body))
	if err != nil {
		return nil, err
	}
	var out preparedTx
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, fmt.Errorf("prepare response parse: %w (body: %s)", err, string(resp))
	}
	return &out, nil
}

// submitTransfer sends the signature back for verification and broadcast.
func (s *CustodyService) submitTransfer(prepared *preparedTx, signature string) (string, error) {
	payload := map[string]interface{}{
		"chain_id":             prepared.ChainID,
		"from":                 prepared.From,
		"to":                   prepared.To,
		"value_wei":            prepared.ValueWei,
		"data":                 prepared.Data,
		"gas_limit":            prepared.GasLimit,
		"tx_type":              prepared.TxType,
		"nonce":                prepared.Nonce,
		"gas_price_wei":        prepared.GasPriceWei,
		"max_fee_wei":          prepared.MaxFeeWei,
		"max_priority_fee_wei": prepared.MaxPriorityWei,
		"signing_hash":         prepared.SigningHash,
		"signature":            signature,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	resp, err := s.postJSON(s.config.WalletAPIURL+"/api/v1/service/tx/submit", string(body))
	if err != nil {
		return "", err
	}
	var out struct {
		TxHash string `json:"tx_hash"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return "", fmt.Errorf("submit response parse: %w (body: %s)", err, string(resp))
	}
	if out.Error != "" {
		return "", fmt.Errorf("wallet_api error: %s", out.Error)
	}
	if out.TxHash == "" {
		return "", fmt.Errorf("wallet_api returned no tx hash (body: %s)", string(resp))
	}
	return out.TxHash, nil
}

// createMPCWallet asks the canonical go/mpc backend to generate a real
// distributed secp256k1 key (POST /api/v1/mpc/create) and returns the keyId +
// the on-chain wallet address. On failure (backend unreachable / not
// configured) it returns an error so CreateAccount fails honestly.
func (s *CustodyService) createMPCWallet(threshold int) (keyID, address string, err error) {
	if s.config.MPCBackendURL == "" {
		return "", "", fmt.Errorf("MPC backend not configured (set MPC_BACKEND_URL)")
	}
	if threshold < 1 {
		threshold = 1
	}
	body := fmt.Sprintf(`{"threshold":%d,"totalShards":%d}`, threshold, threshold)
	resp, err := s.postJSON(s.config.MPCBackendURL+"/api/v1/mpc/create", body)
	if err != nil {
		return "", "", fmt.Errorf("mpc create request: %w", err)
	}
	var out struct {
		KeyID   string `json:"keyId"`
		Address string `json:"address"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return "", "", fmt.Errorf("mpc create response parse: %w (body: %s)", err, string(resp))
	}
	if out.Error != "" {
		return "", "", fmt.Errorf("mpc backend error: %s", out.Error)
	}
	if out.KeyID == "" || out.Address == "" {
		return "", "", fmt.Errorf("mpc backend returned empty keyId/address (body: %s)", string(resp))
	}
	return out.KeyID, out.Address, nil
}

// mpcSign asks the canonical go/mpc backend to sign a message hash with the
// stored distributed key (POST /api/v1/mpc/sign). Returns the hex signature.
func (s *CustodyService) mpcSign(keyID, messageHash string) (string, error) {
	body, _ := json.Marshal(map[string]string{"keyId": keyID, "messageHash": messageHash})
	resp, err := s.postJSON(s.config.MPCBackendURL+"/api/v1/mpc/sign", string(body))
	if err != nil {
		return "", err
	}
	var out struct {
		Signature string `json:"signature"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return "", fmt.Errorf("mpc sign response parse: %w (body: %s)", err, string(resp))
	}
	if out.Error != "" {
		return "", fmt.Errorf("mpc backend error: %s", out.Error)
	}
	if out.Signature == "" {
		return "", fmt.Errorf("mpc backend returned empty signature (body: %s)", string(resp))
	}
	return out.Signature, nil
}

// postJSON is a small helper that POSTs a JSON body and returns the response
// bytes. It is the only HTTP helper the custody service needs.
func (s *CustodyService) postJSON(url, body string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := s.config.MPCServiceToken; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("backend %s returned %d: %s", url, resp.StatusCode, string(data))
	}
	return data, nil
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

func (s *CustodyService) logCompliance(institutionID, accountID uint, action, details, performedBy, ipAddress string) {
	log := ComplianceLog{
		InstitutionID: institutionID,
		AccountID:     accountID,
		Action:        action,
		Details:       details,
		PerformedBy:   performedBy,
		IPAddress:     ipAddress,
		Timestamp:     time.Now(),
	}
	s.db.Create(&log)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort:      getEnv("PORT", "8097"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "tigerwallet"),
		DBPassword:      mustEnv("DB_PASSWORD"),
		DBName:          getEnv("DB_NAME", "tigerwallet_custody"),
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
		MPCBackendURL:   getEnv("MPC_BACKEND_URL", "http://localhost:9099"),
		MPCServiceToken: getEnv("MPC_SERVICE_TOKEN", ""),
		WalletAPIURL:    getEnv("WALLET_API_URL", "http://localhost:8443"),
		APIKey:          mustEnv("CUSTODY_API_KEY"),
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
	api.Use(requireAPIKey(config.APIKey))
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

// mustEnv returns the named environment variable or terminates the process.
// A secret has no safe default: a committed fallback would ship a known
// credential to every deployment.
func mustEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	log.Fatalf("%s must be set", key)
	return ""
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// requireAPIKey authenticates every custody route. The service approves
// withdrawals that move real funds, so a request without the operator's key is
// refused before any handler runs. The comparison is constant-time: a timing
// side channel on a credential check is not worth the risk in a money path.
func requireAPIKey(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		presented := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if presented == "" {
			presented = strings.TrimSpace(c.GetHeader("X-API-Key"))
		}
		if expected == "" || subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "invalid or missing API key",
			})
			return
		}
		c.Next()
	}
}
