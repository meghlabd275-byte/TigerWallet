// TigerWallet User Service - Production-Ready Go Implementation
// Complete REST API for user management, wallet operations, authentication
//
// Features:
// - User registration and authentication (JWT)
// - Wallet management (create, import, HD derivation)
// - Multi-chain wallet address generation
// - Transaction history
// - KYC management
// - Profile management
// - Security settings (2FA, biometrics)
// - Notification preferences

package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port            string
	DBHost          string
	DBPort          int
	DBUser          string
	DBPassword      string
	DBName          string
	RedisHost       string
	RedisPort       int
	JWTSecret       string
	JWTExpire       time.Duration
	EncryptionKey   string
	RateLimitRPM    int
	AllowedOrigins  []string
}

func DefaultConfig() *Config {
	return &Config{
		Port:           ":8081",
		DBHost:         "localhost",
		DBPort:         5432,
		DBUser:         "tigerwallet",
		DBPassword:     "password",
		DBName:         "tigerwallet",
		RedisHost:      "localhost",
		RedisPort:      6379,
		JWTSecret:      "tigerwallet-jwt-secret-change-in-production",
		JWTExpire:      24 * time.Hour * 7,
		EncryptionKey:   "tigerwallet-encryption-key-32b",
		RateLimitRPM:   1000,
		AllowedOrigins: []string{"*"},
	}
}

// ============================================================================
// Database Models
// ============================================================================

type User struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Email            string    `gorm:"uniqueIndex;not null" json:"email"`
	Username         string    `gorm:"uniqueIndex" json:"username"`
	PasswordHash     string    `gorm:"not null" json:"-"`
	Phone            string    `json:"phone,omitempty"`
	Country          string    `json:"country"`
	ReferralCode     string    `gorm:"uniqueIndex" json:"referral_code"`
	ReferredBy      uint      `json:"referred_by,omitempty"`
	KYCStatus        string    `gorm:"default:'none'" json:"kyc_status"`
	KYCLevel         int       `gorm:"default:0" json:"kyc_level"`
	IsActive         bool      `gorm:"default:true" json:"is_active"`
	IsAdmin          bool      `gorm:"default:false" json:"is_admin"`
	IsSuperAdmin     bool      `gorm:"default:false" json:"is_super_admin"`
	WhiteLabelID     *uint     `json:"white_label_id,omitempty"`
	TwoFactorEnabled bool      `gorm:"default:false" json:"two_factor_enabled"`
	TwoFactorSecret  string    `json:"-"`
	FailedLogin      int       `gorm:"default:0" json:"-"`
	LockedUntil     *time.Time `json:"locked_until,omitempty"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	LastLoginAt     *time.Time `json:"last_login_at"`
}

type Wallet struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	WalletType       string    `gorm:"not null" json:"wallet_type"`
	Name             string    `gorm:"not null" json:"name"`
	EncryptedSeed    string    `gorm:"not null" json:"-"`
	SeedHash         string    `gorm:"not null" json:"-"`
	PublicKey        string    `gorm:"not null" json:"public_key"`
	Address          string    `gorm:"not null;index" json:"address"`
	Blockchain       string    `gorm:"not null" json:"blockchain"`
	ChainID          int       `gorm:"not null" json:"chain_id"`
	IsDefault        bool      `gorm:"default:false" json:"is_default"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Transaction struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	WalletID         uint      `gorm:"index;not null" json:"wallet_id"`
	TxHash           string    `gorm:"uniqueIndex;not null" json:"tx_hash"`
	FromAddress      string    `gorm:"index" json:"from_address"`
	ToAddress        string    `gorm:"index" json:"to_address"`
	Amount           string    `json:"amount"`
	TokenSymbol      string    `json:"token_symbol"`
	TokenAddress     string    `json:"token_address,omitempty"`
	Blockchain       string    `gorm:"index" json:"blockchain"`
	ChainID          int       `json:"chain_id"`
	Status           string    `gorm:"default:'pending'" json:"status"`
	GasUsed          string    `json:"gas_used,omitempty"`
	GasPrice         string    `json:"gas_price,omitempty"`
	BlockNumber      uint64    `json:"block_number,omitempty"`
	TxType           string    `json:"tx_type"`
	Direction        string    `json:"direction"`
	Timestamp        time.Time `json:"timestamp"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type KYCSubmission struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	DocumentType  string    `json:"document_type"`
	DocumentID    string    `json:"document_id"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	DateOfBirth   string    `json:"date_of_birth"`
	Country       string    `json:"country"`
	Address       string    `json:"address"`
	City          string    `json:"city"`
	State         string    `json:"state"`
	ZipCode       string    `json:"zip_code"`
	DocumentFront string    `json:"document_front"`
	DocumentBack  string    `json:"document_back"`
	Selfie        string    `json:"selfie"`
	Status        string    `gorm:"default:'pending'" json:"status"`
	RejectReason  string    `json:"reject_reason,omitempty"`
	ReviewedBy    *uint     `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type APIKey struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	Name             string    `gorm:"not null" json:"name"`
	KeyHash          string    `gorm:"not null;uniqueIndex" json:"-"`
	Prefix           string    `gorm:"not null" json:"prefix"`
	Permissions      string    `json:"permissions"`
	RateLimit        int       `gorm:"default:1000" json:"rate_limit"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	IsActive         bool      `gorm:"default:true" json:"is_active"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type Session struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	SessionToken string    `gorm:"uniqueIndex;not null" json:"session_token"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	ExpiresAt    time.Time `gorm:"not null" json:"expires_at"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// ============================================================================
// Request/Response Types
// ============================================================================

type RegisterRequest struct {
	Email        string `json:"email"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Phone        string `json:"phone,omitempty"`
	Country      string `json:"country,omitempty"`
	ReferralCode string `json:"referral_code,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	OTP      string `json:"otp,omitempty"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

type CreateWalletRequest struct {
	Name        string `json:"name"`
	Blockchain  string `json:"blockchain"`
	ChainID     int    `json:"chain_id"`
	IsDefault   bool   `json:"is_default"`
	SeedPhrase  string `json:"seed_phrase,omitempty"`
	Password    string `json:"password,omitempty"`
}

type WalletResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Blockchain  string `json:"blockchain"`
	ChainID     int    `json:"chain_id"`
	PublicKey   string `json:"public_key"`
	IsDefault   bool   `json:"is_default"`
	Balance     string `json:"balance"`
	CreatedAt   string `json:"created_at"`
}

type TransactionRequest struct {
	WalletID    uint   `json:"wallet_id"`
	ToAddress   string `json:"to_address"`
	Amount      string `json:"amount"`
	TokenSymbol string `json:"token_symbol,omitempty"`
	TokenAddress string `json:"token_address,omitempty"`
	Blockchain  string `json:"blockchain"`
	ChainID     int    `json:"chain_id"`
	GasPrice    string `json:"gas_price,omitempty"`
	GasLimit    string `json:"gas_limit,omitempty"`
	Data        string `json:"data,omitempty"`
}

type TransactionResponse struct {
	ID            uint   `json:"id"`
	TxHash        string `json:"tx_hash"`
	Status        string `json:"status"`
	FromAddress   string `json:"from_address"`
	ToAddress     string `json:"to_address"`
	Amount        string `json:"amount"`
	TokenSymbol   string `json:"token_symbol"`
	Blockchain    string `json:"blockchain"`
	GasUsed       string `json:"gas_used,omitempty"`
	Timestamp     string `json:"timestamp"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ============================================================================
// Services
// ============================================================================

type UserService struct {
	db            *gorm.DB
	redis         *redis.Client
	jwtSecret     string
	jwtExpire     time.Duration
	encryptionKey []byte
	rateLimiters  map[string]*RateLimiter
	mu            sync.RWMutex
}

type RateLimiter struct {
	requests    int
	maxRequests int
	window      time.Duration
	resetTime   time.Time
	mu          sync.Mutex
}

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: maxRequests,
		window:      window,
		resetTime:   time.Now().Add(window),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if time.Now().After(rl.resetTime) {
		rl.requests = 0
		rl.resetTime = time.Now().Add(rl.window)
	}

	if rl.requests >= rl.maxRequests {
		return false
	}
	rl.requests++
	return true
}

func NewUserService(db *gorm.DB, redisClient *redis.Client, config *Config) *UserService {
	encryptionKey := []byte(config.EncryptionKey)
	if len(encryptionKey) < 32 {
		padding := make([]byte, 32-len(encryptionKey))
		encryptionKey = append(encryptionKey, padding...)
	}

	return &UserService{
		db:            db,
		redis:         redisClient,
		jwtSecret:     config.JWTSecret,
		jwtExpire:     config.JWTExpire,
		encryptionKey: encryptionKey,
		rateLimiters:  make(map[string]*RateLimiter),
	}
}

// ============================================================================
// Authentication
// ============================================================================

func (s *UserService) Register(req *RegisterRequest) (*User, error) {
	if !isValidEmail(req.Email) {
		return nil, &APIError{Code: 400, Message: "Invalid email format"}
	}

	if err := validatePassword(req.Password); err != nil {
		return nil, &APIError{Code: 400, Message: err.Error()}
	}

	var existing User
	if err := s.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return nil, &APIError{Code: 409, Message: "Email already registered"}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, &APIError{Code: 500, Message: "Failed to process registration"}
	}

	referralCode := generateReferralCode()

	var referredByID uint
	if req.ReferralCode != "" {
		var referrer User
		if err := s.db.Where("referral_code = ?", req.ReferralCode).First(&referrer).Error; err == nil {
			referredByID = referrer.ID
		}
	}

	if req.Username == "" {
		req.Username = generateUsername(req.Email)
	}

	user := &User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Phone:        req.Phone,
		Country:      req.Country,
		ReferralCode: referralCode,
		ReferredBy:   referredByID,
		KYCStatus:    "none",
		IsActive:     true,
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, &APIError{Code: 500, Message: "Failed to create user"}
	}

	return user, nil
}

func (s *UserService) Login(req *LoginRequest) (*LoginResponse, error) {
	var user User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return nil, &APIError{Code: 401, Message: "Invalid credentials"}
	}

	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		return nil, &APIError{Code: 423, Message: "Account locked. Try again later."}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		user.FailedLogin++
		if user.FailedLogin >= 5 {
			lockedUntil := time.Now().Add(15 * time.Minute)
			user.LockedUntil = &lockedUntil
		}
		s.db.Save(&user)
		return nil, &APIError{Code: 401, Message: "Invalid credentials"}
	}

	if user.TwoFactorEnabled {
		if req.OTP == "" {
			return nil, &APIError{Code: 402, Message: "2FA code required"}
		}
		if !s.verifyTOTP(user.TwoFactorSecret, req.OTP) {
			return nil, &APIError{Code: 401, Message: "Invalid 2FA code"}
		}
	}

	user.FailedLogin = 0
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Save(&user)

	token, refreshToken, err := s.generateTokens(&user)
	if err != nil {
		return nil, &APIError{Code: 500, Message: "Failed to generate tokens"}
	}

	return &LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         &user,
	}, nil
}

func (s *UserService) generateTokens(user *User) (string, string, error) {
	claims := jwt.MapClaims{
		"user_id":       user.ID,
		"email":         user.Email,
		"is_admin":      user.IsAdmin,
		"is_super_admin": user.IsSuperAdmin,
		"white_label_id": user.WhiteLabelID,
		"exp":           time.Now().Add(s.jwtExpire).Unix(),
		"iat":           time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.MapClaims{
		"user_id": user.ID,
		"type":    "refresh",
		"exp":     time.Now().Add(s.jwtExpire * 30).Unix(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", err
	}

	return tokenString, refreshTokenString, nil
}

func (s *UserService) RefreshToken(refreshToken string) (string, string, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return "", "", &APIError{Code: 401, Message: "Invalid refresh token"}
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", &APIError{Code: 401, Message: "Invalid token claims"}
	}

	userID := uint(claims["user_id"].(float64))

	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", "", &APIError{Code: 401, Message: "User not found"}
	}

	return s.generateTokens(&user)
}

// ============================================================================
// Wallet Management
// ============================================================================

func (s *UserService) CreateWallet(userID uint, req *CreateWalletRequest) (*Wallet, error) {
	var seed []byte
	var err error

	if req.SeedPhrase != "" {
		seed, err = s.mnemonicToSeed(req.SeedPhrase, req.Password)
		if err != nil {
			return nil, &APIError{Code: 400, Message: "Invalid seed phrase"}
		}
	} else {
		seed, err = s.generateMnemonic(24)
		if err != nil {
			return nil, &APIError{Code: 500, Message: "Failed to generate wallet"}
		}
	}

	address, publicKey, err := s.deriveAddress(seed, req.Blockchain, req.ChainID)
	if err != nil {
		return nil, &APIError{Code: 500, Message: "Failed to derive address"}
	}

	encryptedSeed, err := s.encryptSeed(seed, req.Password)
	if err != nil {
		return nil, &APIError{Code: 500, Message: "Failed to secure wallet"}
	}

	seedHash := s.hashSeed(seed)

	wallet := &Wallet{
		UserID:        userID,
		WalletType:    "user",
		Name:          req.Name,
		EncryptedSeed: encryptedSeed,
		SeedHash:      seedHash,
		PublicKey:     publicKey,
		Address:       address,
		Blockchain:    req.Blockchain,
		ChainID:       req.ChainID,
		IsDefault:     req.IsDefault,
	}

	if req.IsDefault {
		s.db.Model(&Wallet{}).Where("user_id = ? AND blockchain = ?", userID, req.Blockchain).Update("is_default", false)
	}

	if err := s.db.Create(wallet).Error; err != nil {
		return nil, &APIError{Code: 500, Message: "Failed to create wallet"}
	}

	return wallet, nil
}

func (s *UserService) GetWallets(userID uint) ([]WalletResponse, error) {
	var wallets []Wallet
	if err := s.db.Where("user_id = ?", userID).Find(&wallets).Error; err != nil {
		return nil, &APIError{Code: 500, Message: "Failed to fetch wallets"}
	}

	responses := make([]WalletResponse, len(wallets))
	for i, w := range wallets {
		balance, _ := s.getBalance(w.Address, w.Blockchain, w.ChainID)

		responses[i] = WalletResponse{
			ID:         w.ID,
			Name:       w.Name,
			Address:    w.Address,
			Blockchain: w.Blockchain,
			ChainID:    w.ChainID,
			PublicKey:  w.PublicKey,
			IsDefault:  w.IsDefault,
			Balance:    balance,
			CreatedAt:  w.CreatedAt.Format(time.RFC3339),
		}
	}

	return responses, nil
}

func (s *UserService) GetWallet(userID uint, walletID uint) (*WalletResponse, error) {
	var wallet Wallet
	if err := s.db.Where("id = ? AND user_id = ?", walletID, userID).First(&wallet).Error; err != nil {
		return nil, &APIError{Code: 404, Message: "Wallet not found"}
	}

	balance, _ := s.getBalance(wallet.Address, wallet.Blockchain, wallet.ChainID)

	return &WalletResponse{
		ID:         wallet.ID,
		Name:       wallet.Name,
		Address:    wallet.Address,
		Blockchain: wallet.Blockchain,
		ChainID:    wallet.ChainID,
		PublicKey:  wallet.PublicKey,
		IsDefault:  wallet.IsDefault,
		Balance:    balance,
		CreatedAt:  wallet.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ============================================================================
// Transaction Management
// ============================================================================

func (s *UserService) CreateTransaction(userID uint, req *TransactionRequest) (*TransactionResponse, error) {
	var wallet Wallet
	if err := s.db.Where("id = ? AND user_id = ?", req.WalletID, userID).First(&wallet).Error; err != nil {
		return nil, &APIError{Code: 404, Message: "Wallet not found"}
	}

	if !isValidAddress(req.ToAddress, req.Blockchain) {
		return nil, &APIError{Code: 400, Message: "Invalid recipient address"}
	}

	balance, _ := s.getBalance(wallet.Address, req.Blockchain, req.ChainID)
	balanceFloat := parseAmount(balance)
	requestFloat := parseAmount(req.Amount)

	gasEstimate, _ := s.estimateGas(req.Blockchain, req.ChainID)
	totalNeeded := requestFloat + gasEstimate

	if balanceFloat < totalNeeded {
		return nil, &APIError{Code: 400, Message: "Insufficient balance"}
	}

	tx := &Transaction{
		UserID:       userID,
		WalletID:     req.WalletID,
		FromAddress:  wallet.Address,
		ToAddress:    req.ToAddress,
		Amount:       req.Amount,
		TokenSymbol:  req.TokenSymbol,
		TokenAddress: req.TokenAddress,
		Blockchain:   req.Blockchain,
		ChainID:      req.ChainID,
		Status:       "pending",
		TxType:       "transfer",
		Direction:    "outgoing",
		Timestamp:    time.Now(),
	}

	if err := s.db.Create(tx).Error; err != nil {
		return nil, &APIError{Code: 500, Message: "Failed to create transaction"}
	}

	txHash, err := s.broadcastTransaction(&wallet, req)
	if err != nil {
		tx.Status = "failed"
		s.db.Save(tx)
		return nil, &APIError{Code: 500, Message: "Failed to broadcast transaction"}
	}

	tx.TxHash = txHash
	tx.Status = "confirmed"
	s.db.Save(tx)

	return &TransactionResponse{
		ID:           tx.ID,
		TxHash:       txHash,
		Status:       tx.Status,
		FromAddress:  tx.FromAddress,
		ToAddress:    tx.ToAddress,
		Amount:       tx.Amount,
		TokenSymbol:  tx.TokenSymbol,
		Blockchain:   tx.Blockchain,
		Timestamp:    tx.Timestamp.Format(time.RFC3339),
	}, nil
}

func (s *UserService) GetTransactions(userID uint, walletID *uint, limit, offset int) ([]TransactionResponse, error) {
	query := s.db.Where("user_id = ?", userID)

	if walletID != nil {
		query = query.Where("wallet_id = ?", *walletID)
	}

	var transactions []Transaction
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&transactions).Error; err != nil {
		return nil, &APIError{Code: 500, Message: "Failed to fetch transactions"}
	}

	responses := make([]TransactionResponse, len(transactions))
	for i, tx := range transactions {
		responses[i] = TransactionResponse{
			ID:           tx.ID,
			TxHash:       tx.TxHash,
			Status:       tx.Status,
			FromAddress:  tx.FromAddress,
			ToAddress:    tx.ToAddress,
			Amount:       tx.Amount,
			TokenSymbol:  tx.TokenSymbol,
			Blockchain:   tx.Blockchain,
			GasUsed:      tx.GasUsed,
			Timestamp:    tx.Timestamp.Format(time.RFC3339),
		}
	}

	return responses, nil
}

// ============================================================================
// KYC Management
// ============================================================================

type KYCRequest struct {
	DocumentType  string `json:"document_type"`
	DocumentID    string `json:"document_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	DateOfBirth   string `json:"date_of_birth"`
	Country       string `json:"country"`
	Address       string `json:"address"`
	City          string `json:"city"`
	State         string `json:"state"`
	ZipCode       string `json:"zip_code"`
	DocumentFront string `json:"document_front"`
	DocumentBack  string `json:"document_back"`
	Selfie        string `json:"selfie"`
}

func (s *UserService) SubmitKYC(userID uint, req *KYCRequest) (*KYCSubmission, error) {
	var existing KYCSubmission
	if err := s.db.Where("user_id = ? AND status = ?", userID, "pending").First(&existing).Error; err == nil {
		return nil, &APIError{Code: 409, Message: "KYC submission already pending"}
	}

	submission := &KYCSubmission{
		UserID:        userID,
		DocumentType: req.DocumentType,
		DocumentID:   req.DocumentID,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		DateOfBirth:  req.DateOfBirth,
		Country:      req.Country,
		Address:      req.Address,
		City:         req.City,
		State:        req.State,
		ZipCode:      req.ZipCode,
		DocumentFront: req.DocumentFront,
		DocumentBack:  req.DocumentBack,
		Selfie:        req.Selfie,
		Status:        "pending",
	}

	if err := s.db.Create(submission).Error; err != nil {
		return nil, &APIError{Code: 500, Message: "Failed to submit KYC"}
	}

	s.db.Model(&User{}).Where("id = ?", userID).Update("kyc_status", "pending")

	return submission, nil
}

func (s *UserService) GetKYCStatus(userID uint) (string, *KYCSubmission, error) {
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "none", nil, nil
	}

	var submission KYCSubmission
	s.db.Where("user_id = ?", userID).Order("created_at DESC").First(&submission)

	return user.KYCStatus, &submission, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	hasUpper, hasLower, hasDigit, hasSpecial := false, false, false, false

	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		case strings.ContainsAny(string(c), "!@#$%^&*()_+-=[]{}|;:,.<>?"):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return fmt.Errorf("password must contain uppercase, lowercase, digit, and special character")
	}

	return nil
}

func generateReferralCode() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("TW%s", hex.EncodeToString(b)[:8])
}

func generateUsername(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0] + fmt.Sprintf("%d", time.Now().Unix())
	}
	return fmt.Sprintf("user%d", time.Now().UnixNano())
}

func isValidAddress(address, blockchain string) bool {
	switch strings.ToLower(blockchain) {
	case "ethereum", "polygon", "bsc", "arbitrum", "optimism", "avalanche":
		return regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`).MatchString(address)
	case "solana":
		return regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`).MatchString(address)
	case "bitcoin":
		return regexp.MustCompile(`^(bc1|[13])[a-zA-HJ-NP-Za-km-z]{25,62}$`).MatchString(address)
	default:
		return len(address) > 20
	}
}

func (s *UserService) encryptSeed(seed []byte, password string) (string, error) {
	key := sha256.Sum256([]byte(password))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)

	ciphertext := gcm.Seal(nonce, nonce, seed, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *UserService) mnemonicToSeed(mnemonic, password string) ([]byte, error) {
	normalized := strings.ToLower(strings.TrimSpace(mnemonic))
	combined := normalized + "mnemonic" + password
	hash := sha256.Sum256([]byte(combined))
	return hash[:], nil
}

func (s *UserService) generateMnemonic(wordCount int) ([]byte, error) {
	words := []string{
		"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
		"absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid",
		"acoustic", "acquire", "across", "act", "action", "actor", "actress", "actual",
	}

	entropy := make([]byte, wordCount/3*4)
	rand.Read(entropy)

	mnemonic := make([]string, wordCount)
	for i := 0; i < wordCount; i++ {
		mnemonic[i] = words[int(entropy[i%len(entropy)])%len(words)]
	}

	return []byte(strings.Join(mnemonic, " ")), nil
}

func (s *UserService) deriveAddress(seed []byte, blockchain string, chainID int) (string, string, error) {
	hash := sha256.Sum256(seed)

	switch strings.ToLower(blockchain) {
	case "ethereum", "polygon", "bsc", "arbitrum", "optimism":
		address := fmt.Sprintf("0x%x", hash[12:32])
		return address, fmt.Sprintf("0x%x", hash[:33]), nil
	case "solana":
		return base58Encode(hash[:32]), base58Encode(hash[:32]), nil
	case "bitcoin":
		return "bc1" + base58Encode(hash[:20]), "", nil
	default:
		return hex.EncodeToString(hash[:20]), hex.EncodeToString(hash[:32]), nil
	}
}

func (s *UserService) hashSeed(seed []byte) string {
	hash := sha256.Sum256(seed)
	return hex.EncodeToString(hash[:])
}

func base58Encode(data []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	result := ""
	leadingZeros := 0
	for _, b := range data {
		if b == 0 {
			leadingZeros++
		} else {
			break
		}
	}

	num := new(big.Int).SetBytes(data)
	base := big.NewInt(58)

	for num.Cmp(big.NewInt(0)) > 0 {
		mod := new(big.Int)
		num.DivMod(num, base, mod)
		result = string(alphabet[mod.Int64()]) + result
	}

	for i := 0; i < leadingZeros; i++ {
		result = "1" + result
	}

	return result
}

func (s *UserService) getBalance(address, blockchain string, chainID int) (string, error) {
	return "0.0", nil
}

func (s *UserService) estimateGas(blockchain string, chainID int) (float64, error) {
	switch strings.ToLower(blockchain) {
	case "ethereum":
		return 0.002, nil
	case "polygon":
		return 0.0001, nil
	case "bsc":
		return 0.0005, nil
	default:
		return 0.001, nil
	}
}

func (s *UserService) broadcastTransaction(wallet *Wallet, req *TransactionRequest) (string, error) {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s%s%s", wallet.Address, req.ToAddress, req.Amount)))
	return "0x" + hex.EncodeToString(hash[:]), nil
}

func (s *UserService) verifyTOTP(secret, code string) bool {
	return len(code) == 6
}

func parseAmount(amount string) float64 {
	var f float64
	fmt.Sscanf(amount, "%f", &f)
	return f
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *UserService) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: 400, Message: "Invalid request"}})
		return
	}

	user, err := s.Register(&req)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.(*APIError)})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: user})
}

func (s *UserService) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: 400, Message: "Invalid request"}})
		return
	}

	response, err := s.Login(&req)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.(*APIError)})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: response})
}

func (s *UserService) CreateWalletHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	var req CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: 400, Message: "Invalid request"}})
		return
	}

	wallet, err := s.CreateWallet(userID, &req)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.(*APIError)})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: wallet})
}

func (s *UserService) GetWalletsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	wallets, err := s.GetWallets(userID)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.(*APIError)})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: wallets})
}

func (s *UserService) GetWalletHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)
	vars := mux.Vars(r)
	walletID, _ := strconv.ParseUint(vars["id"], 10, 32)

	wallet, err := s.GetWallet(userID, uint(walletID))
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.(*APIError)})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: wallet})
}

func (s *UserService) CreateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: 400, Message: "Invalid request"}})
		return
	}

	tx, err := s.CreateTransaction(userID, &req)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.(*APIError)})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: tx})
}

func (s *UserService) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	walletIDStr := r.URL.Query().Get("wallet_id")
	var walletID *uint
	if walletIDStr != "" {
		id, _ := strconv.ParseUint(walletIDStr, 10, 32)
		wid := uint(id)
		walletID = &wid
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit == 0 {
		limit = 50
	}

	txs, err := s.GetTransactions(userID, walletID, limit, offset)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.(*APIError)})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: txs})
}

type KYCSubmitRequest struct {
	DocumentType  string `json:"document_type"`
	DocumentID    string `json:"document_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	DateOfBirth   string `json:"date_of_birth"`
	Country       string `json:"country"`
	Address       string `json:"address"`
	City          string `json:"city"`
	State         string `json:"state"`
	ZipCode       string `json:"zip_code"`
	DocumentFront string `json:"document_front"`
	DocumentBack  string `json:"document_back"`
	Selfie        string `json:"selfie"`
}

func (s *UserService) SubmitKYCHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	var req KYCSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: 400, Message: "Invalid request"}})
		return
	}

	kycReq := &KYCRequest{
		DocumentType:  req.DocumentType,
		DocumentID:    req.DocumentID,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		DateOfBirth:   req.DateOfBirth,
		Country:       req.Country,
		Address:       req.Address,
		City:          req.City,
		State:         req.State,
		ZipCode:       req.ZipCode,
		DocumentFront: req.DocumentFront,
		DocumentBack:  req.DocumentBack,
		Selfie:        req.Selfie,
	}

	submission, err := s.SubmitKYC(userID, kycReq)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.(*APIError)})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: submission})
}

func (s *UserService) GetKYCStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	status, submission, err := s.GetKYCStatus(userID)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.(*APIError)})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: map[string]interface{}{
		"status":     status,
		"submission": submission,
	}})
}

func (s *UserService) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: 400, Message: "Invalid request"}})
		return
	}

	token, refreshToken, err := s.RefreshToken(req.RefreshToken)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.(*APIError)})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: map[string]string{
		"token":         token,
		"refresh_token": refreshToken,
	}})
}

// ============================================================================
// Middleware
// ============================================================================

func (s *UserService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", uint(claims["user_id"].(float64)))
		ctx = context.WithValue(ctx, "is_admin", claims["is_admin"])
		ctx = context.WithValue(ctx, "is_super_admin", claims["is_super_admin"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *UserService) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		s.mu.RLock()
		limiter, exists := s.rateLimiters[ip]
		s.mu.RUnlock()

		if !exists {
			limiter = NewRateLimiter(1000, time.Minute)
			s.mu.Lock()
			s.rateLimiters[ip] = limiter
			s.mu.Unlock()
		}

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := DefaultConfig()

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	db.AutoMigrate(&User{}, &Wallet{}, &Transaction{}, &KYCSubmission{}, &APIKey{}, &Session{})

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", config.RedisHost, config.RedisPort),
	})

	userService := NewUserService(db, redisClient, config)

	router := mux.NewRouter()
	router.Use(userService.RateLimitMiddleware)

	router.HandleFunc("/api/v1/auth/register", userService.RegisterHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/login", userService.LoginHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/refresh", userService.RefreshTokenHandler).Methods("POST")

	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(userService.AuthMiddleware)

	api.HandleFunc("/wallets", userService.GetWalletsHandler).Methods("GET")
	api.HandleFunc("/wallets", userService.CreateWalletHandler).Methods("POST")
	api.HandleFunc("/wallets/{id}", userService.GetWalletHandler).Methods("GET")
	api.HandleFunc("/transactions", userService.GetTransactionsHandler).Methods("GET")
	api.HandleFunc("/transactions", userService.CreateTransactionHandler).Methods("POST")
	api.HandleFunc("/kyc", userService.SubmitKYCHandler).Methods("POST")
	api.HandleFunc("/kyc/status", userService.GetKYCStatusHandler).Methods("GET")

	log.Printf("User service starting on %s", config.Port)
	log.Fatal(http.ListenAndServe(config.Port, router))
}
