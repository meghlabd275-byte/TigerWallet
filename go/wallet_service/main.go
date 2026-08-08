// TigerWallet Backend Service - Comprehensive Wallet Management
// High-performance, distributed wallet service for multi-chain operations

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/pbkdf2"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port                int           `json:"port"`
	JWTSecret           string        `json:"jwt_secret"`
	RedisAddr           string        `json:"redis_addr"`
	MongoURI            string        `json:"mongo_uri"`
	EncryptionKey       string        `json:"encryption_key"`
	MasterWalletAddress string        `json:"master_wallet_address"`
	SessionTimeout      time.Duration `json:"session_timeout"`
}

var cfg = Config{
	Port:                8001,
	JWTSecret:           getRequiredEnv("JWT_SECRET"),
	RedisAddr:           "localhost:6379",
	SessionTimeout:      24 * time.Hour,
	MasterWalletAddress: "0x0000000000000000000000000000000000000001",
}

// getRequiredEnv reads a required environment variable and fatally exits if it
// is unset. Used for secrets that must never fall back to insecure defaults.
func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable must be set", key)
	}
	return value
}

// ============================================================================
// Database Models
// ============================================================================

type User struct {
	ID                string             `json:"id" bson:"_id"`
	Email             string             `json:"email" bson:"email"`
	Username          string             `json:"username" bson:"username"`
	PasswordHash      string             `json:"-" bson:"password_hash"`
	EncryptedSeed     string             `json:"-" bson:"encrypted_seed"`
	Wallets           []UserWallet       `json:"wallets" bson:"wallets"`
	MasterWalletID    string             `json:"master_wallet_id" bson:"master_wallet_id"`
	KYCStatus         string             `json:"kyc_status" bson:"kyc_status"`
	KYCLevel          int                `json:"kyc_level" bson:"kyc_level"`
	TwoFactorEnabled  bool               `json:"two_factor_enabled" bson:"two_factor_enabled"`
	TwoFactorSecret   string             `json:"-" bson:"two_factor_secret"`
	Permissions       UserPermissions    `json:"permissions" bson:"permissions"`
	CreatedAt         time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at" bson:"updated_at"`
	LastLoginAt       time.Time          `json:"last_login_at" bson:"last_login_at"`
	FailedLoginAttempts int              `json:"failed_login_attempts" bson:"failed_login_attempts"`
	LockedUntil      *time.Time         `json:"locked_until" bson:"locked_until"`
	IPWhitelist       []string           `json:"ip_whitelist" bson:"ip_whitelist"`
	DeviceFingerprints []string         `json:"device_fingerprints" bson:"device_fingerprints"`
	PreferredLanguage string             `json:"preferred_language" bson:"preferred_language"`
	Timezone         string             `json:"timezone" bson:"timezone"`
	ReferralCode     string             `json:"referral_code" bson:"referral_code"`
	ReferredBy       string             `json:"referred_by" bson:"referred_by"`
	VIPLevel         int                `json:"vip_level" bson:"vip_level"`
}

type UserWallet struct {
	ID            string            `json:"id" bson:"id"`
	Name          string            `json:"name" bson:"name"`
	Chain         string            `json:"chain" bson:"chain"`
	Address       string            `json:"address" bson:"address"`
	PublicKey    string            `json:"public_key" bson:"public_key"`
	IsImported   bool              `json:"is_imported" bson:"is_imported"`
	CreatedAt    time.Time         `json:"created_at" bson:"created_at"`
	Balance      map[string]string `json:"balance" bson:"balance"`
	Tokens       []TokenBalance    `json:"tokens" bson:"tokens"`
	NFTs         []NFTBalance      `json:"nfts" bson:"nfts"`
}

type TokenBalance struct {
	Symbol     string `json:"symbol" bson:"symbol"`
	Contract   string `json:"contract" bson:"contract"`
	Balance    string `json:"balance" bson:"balance"`
	Decimals   int    `json:"decimals" bson:"decimals"`
	PriceUSD   string `json:"price_usd" bson:"price_usd"`
	ValueUSD   string `json:"value_usd" bson:"value_usd"`
}

type NFTBalance struct {
	Contract string `json:"contract" bson:"contract"`
	TokenID  string `json:"token_id" bson:"token_id"`
	Name     string `json:"name" bson:"name"`
	ImageURL string `json:"image_url" bson:"image_url"`
	Quantity int    `json:"quantity" bson:"quantity"`
}

type UserPermissions struct {
	CanTrade       bool `json:"can_trade" bson:"can_trade"`
	CanWithdraw    bool `json:"can_withdraw" bson:"can_withdraw"`
	CanDeposit     bool `json:"can_deposit" bson:"can_deposit"`
	CanTransfer    bool `json:"can_transfer" bson:"can_transfer"`
	CanStake       bool `json:"can_stake" bson:"can_stake"`
	CanMintNFT     bool `json:"can_mint_nft" bson:"can_mint_nft"`
	CanCreateToken bool `json:"can_create_token" bson:"can_create_token"`
	CanAccessAPI   bool `json:"can_access_api" bson:"can_access_api"`
	CanWhiteLabel bool `json:"can_white_label" bson:"can_white_label"`
}

type Transaction struct {
	ID          string    `json:"id" bson:"_id"`
	WalletID   string    `json:"wallet_id" bson:"wallet_id"`
	UserID     string    `json:"user_id" bson:"user_id"`
	Chain      string    `json:"chain" bson:"chain"`
	Hash       string    `json:"hash" bson:"hash"`
	From       string    `json:"from" bson:"from"`
	To         string    `json:"to" bson:"to"`
	Value      string    `json:"value" bson:"value"`
	Token      string    `json:"token" bson:"token"`
	TokenValue string    `json:"token_value" bson:"token_value"`
	GasUsed    string    `json:"gas_used" bson:"gas_used"`
	GasPrice   string    `json:"gas_price" bson:"gas_price"`
	Status     string    `json:"status" bson:"status"`
	Type       string    `json:"type" bson:"type"`
	Timestamp  time.Time `json:"timestamp" bson:"timestamp"`
	BlockNumber uint64   `json:"block_number" bson:"block_number"`
	Nonce      uint64    `json:"nonce" bson:"nonce"`
	Data       string    `json:"data" bson:"data"`
}

type Wallet struct {
	ID           string         `json:"id" bson:"_id"`
	MasterID    string         `json:"master_id" bson:"master_id"`
	Chain       string         `json:"chain" bson:"chain"`
	Address     string         `json:"address" bson:"address"`
	PrivateKey  string         `json:"-" bson:"private_key_encrypted"`
	PublicKey  string         `json:"public_key" bson:"public_key"`
	Type       string         `json:"type" bson:"type"` // user, master, hot, cold
	Status     string         `json:"status" bson:"status"` // active, paused, halted, deleted
	Balance    map[string]string `json:"balance" bson:"balance"`
	Nonce      uint64         `json:"nonce" bson:"nonce"`
	CreatedAt  time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at" bson:"updated_at"`
}

// ============================================================================
// Services
// ============================================================================

type WalletService struct {
	redis     *redis.Client
	mu        sync.RWMutex
	sessions  map[string]*Session
	wallets   map[string]*Wallet
	users     map[string]*User
	txCache   map[string][]*Transaction
}

type Session struct {
	UserID     string
	WalletID   string
	ExpiresAt  time.Time
	IPAddress  string
	DeviceID   string
	Permissions UserPermissions
}

func NewWalletService() *WalletService {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	return &WalletService{
		redis:    rdb,
		sessions: make(map[string]*Session),
		wallets:  make(map[string]*Wallet),
		users:    make(map[string]*User),
		txCache:  make(map[string][]*Transaction),
	}
}

// ============================================================================
// Cryptographic Functions
// ============================================================================

func generateMnemonic() (string, error) {
	entropy := make([]byte, 32)
	if _, err := rand.Read(entropy); err != nil {
		return "", err
	}
	
	// In production, use proper BIP-39 wordlist
	words := []string{
		"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
		"absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid",
		"acoustic", "acquire", "across", "act", "action", "actor", "actress", "actual",
	}
	
	// Generate 24 words
	mnemonic := []string{}
	for i := 0; i < 24; i++ {
		mnemonic = append(mnemonic, words[i%len(words)])
	}
	
	return strings.Join(mnemonic, " "), nil
}

func mnemonicToSeed(mnemonic, passphrase string) ([]byte, error) {
	salt := []byte("mnemonic" + passphrase)
	return pbkdf2.Key([]byte(mnemonic), salt, 2048, 64, sha512.New), nil
}

func deriveKeyFromSeed(seed []byte, path string) (*ecdsa.PrivateKey, error) {
	// Simplified HD key derivation
	// In production, use proper BIP-32/BIP-44
	
	hash := sha512.Sum512(seed)
	privateKey := new(ecdsa.PrivateKey)
	privateKey.D = new(big.Int).SetBytes(hash[:32])
	privateKey.PublicKey.Curve = elliptic.P256()
	privateKey.PublicKey.X, privateKey.Y = elliptic.P256().ScalarBaseMult(hash[:32])
	
	return privateKey, nil
}

func encryptData(data []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return hex.EncodeToString(ciphertext), nil
}

func decryptData(encrypted string, key []byte) ([]byte, error) {
	data, err := hex.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("data too short")
	}
	
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func hashPassword(password, salt string) string {
	hash := sha512.Sum512([]byte(password + salt))
	return hex.EncodeToString(hash[:])
}

func deriveEncryptionKey(password, salt string) []byte {
	key := pbkdf2.Key([]byte(password), []byte(salt), 100000, 32, sha256.New)
	return key
}

// ============================================================================
// JWT Functions
// ============================================================================

type Claims struct {
	UserID      string         `json:"user_id"`
	WalletID   string         `json:"wallet_id"`
	Permissions UserPermissions `json:"permissions"`
	jwt.RegisteredClaims
}

func (ws *WalletService) generateToken(user *User, walletID string) (string, error) {
	claims := Claims{
		UserID:      user.ID,
		WalletID:   walletID,
		Permissions: user.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.SessionTimeout)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func (ws *WalletService) validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("invalid token")
}

// ============================================================================
// API Handlers
// ============================================================================

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required,min=8"`
	Referral string `json:"referral"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	DeviceID string `json:"device_id"`
}

type CreateWalletRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Chain  string `json:"chain" binding:"required"`
	Name   string `json:"name"`
}

type ImportWalletRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Mnemonic string `json:"mnemonic" binding:"required"`
	Chain    string `json:"chain" binding:"required"`
	Name     string `json:"name"`
	Password string `json:"password" binding:"required"`
}

type SendTransactionRequest struct {
	From     string `json:"from" binding:"required"`
	To       string `json:"to" binding:"required"`
	Value    string `json:"value" binding:"required"`
	Chain    string `json:"chain" binding:"required"`
	Data     string `json:"data"`
	GasLimit string `json:"gas_limit"`
	GasPrice string `json:"gas_price"`
}

type SwapRequest struct {
	FromToken   string `json:"from_token" binding:"required"`
	ToToken     string `json:"to_token" binding:"required"`
	FromAmount  string `json:"from_amount" binding:"required"`
	MinOutAmount string `json:"min_out_amount"`
	Chain       string `json:"chain" binding:"required"`
}

func (ws *WalletService) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	if _, exists := ws.users[req.Email]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
		return
	}

	// Generate user ID
	userID := uuid.New().String()
	
	// Hash password
	salt := uuid.New().String()
	passwordHash := hashPassword(req.Password, salt)
	
	// Create user
	user := &User{
		ID:               userID,
		Email:            req.Email,
		Username:         req.Username,
		PasswordHash:     passwordHash,
		KYCStatus:        "none",
		KYCLevel:         0,
		Permissions:      UserPermissions{
			CanTrade: true, CanWithdraw: true, CanDeposit: true,
			CanTransfer: true, CanStake: true, CanMintNFT: false,
			CanCreateToken: false, CanAccessAPI: false, CanWhiteLabel: false,
		},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		ReferralCode: uuid.New().String()[:8],
	}
	
	if req.Referral != "" {
		user.ReferredBy = req.Referral
	}
	
	ws.users[req.Email] = user
	
	// Create default wallets for all supported chains
	ws.createDefaultWallets(userID)
	
	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"user_id":  userID,
		"username": user.Username,
		"referral_code": user.ReferralCode,
	})
}

func (ws *WalletService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user
	user, exists := ws.users[req.Email]
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	
	// Check if locked
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		c.JSON(http.StatusLocked, gin.H{"error": "account locked", "until": user.LockedUntil})
		return
	}
	
	// Verify password
	salt := uuid.New().String() // In production, store salt
	passwordHash := hashPassword(req.Password, salt)
	if passwordHash != user.PasswordHash {
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= 5 {
			locked := time.Now().Add(30 * time.Minute)
			user.LockedUntil = &locked
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	
	// Generate token
	token, err := ws.generateToken(user, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	
	// Update last login
	user.LastLoginAt = time.Now()
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	
	// Create session
	sessionID := uuid.New().String()
	ws.sessions[sessionID] = &Session{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(cfg.SessionTimeout),
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"token":       token,
		"session_id":  sessionID,
		"user_id":     user.ID,
		"username":    user.Username,
		"permissions": user.Permissions,
	})
}

func (ws *WalletService) CreateWallet(c *gin.Context) {
	var req CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Generate wallet
	mnemonic, err := generateMnemonic()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate wallet"})
		return
	}
	
	seed, err := mnemonicToSeed(mnemonic, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to derive key"})
		return
	}
	
	privateKey, err := deriveKeyFromSeed(seed, "m/44'/60'/0'/0/0")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to derive key"})
		return
	}
	
	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	
	walletID := uuid.New().String()
	wallet := &Wallet{
		ID:          walletID,
		MasterID:    req.UserID,
		Chain:       req.Chain,
		Address:     address,
		PublicKey:  hex.EncodeToString(elliptic.Marshal(elliptic.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y)),
		Type:        "user",
		Status:      "active",
		Balance:    make(map[string]string),
		Nonce:       0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	ws.wallets[walletID] = wallet
	
	c.JSON(http.StatusCreated, gin.H{
		"success":      true,
		"wallet_id":    walletID,
		"address":      address,
		"chain":        req.Chain,
		"name":         req.Name,
		"mnemonic":     mnemonic,
		"warning":      "Store this mnemonic securely - it cannot be recovered!",
	})
}

func (ws *WalletService) ImportWallet(c *gin.Context) {
	var req ImportWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate mnemonic
	mnemonicWords := strings.Split(req.Mnemonic, " ")
	if len(mnemonicWords) != 12 && len(mnemonicWords) != 24 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mnemonic length"})
		return
	}
	
	// Derive key
	seed, err := mnemonicToSeed(req.Mnemonic, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to derive key"})
		return
	}
	
	privateKey, err := deriveKeyFromSeed(seed, "m/44'/60'/0'/0/0")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to derive key"})
		return
	}
	
	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	
	walletID := uuid.New().String()
	wallet := &Wallet{
		ID:          walletID,
		MasterID:    req.UserID,
		Chain:       req.Chain,
		Address:     address,
		PublicKey:  hex.EncodeToString(elliptic.Marshal(elliptic.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y)),
		Type:        "user",
		Status:      "active",
		Balance:    make(map[string]string),
		Nonce:       0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	ws.wallets[walletID] = wallet
	
	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"wallet_id": walletID,
		"address":   address,
		"chain":     req.Chain,
		"name":      req.Name,
		"imported":  true,
	})
}

func (ws *WalletService) GetBalance(c *gin.Context) {
	walletID := c.Param("id")
	
	wallet, exists := ws.wallets[walletID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"wallet_id":  walletID,
		"address":    wallet.Address,
		"chain":      wallet.Chain,
		"balance":    wallet.Balance,
	})
}

func (ws *WalletService) GetTransactions(c *gin.Context) {
	walletID := c.Param("id")
	
	txs, exists := ws.txCache[walletID]
	if !exists {
		txs = []*Transaction{}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"transactions": txs,
	})
}

func (ws *WalletService) SendTransaction(c *gin.Context) {
	var req SendTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate address
	if !common.IsHexAddress(req.To) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient address"})
		return
	}
	
	// Find wallet
	var wallet *Wallet
	for _, w := range ws.wallets {
		if strings.ToLower(w.Address) == strings.ToLower(req.From) {
			wallet = w
			break
		}
	}
	
	if wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	
	// Transaction broadcast is not implemented: a real tx hash can only be
	// obtained by signing and broadcasting the transaction via RPC
	// (eth_sendRawTransaction). Fabricating a hash from a UUID would be
	// misleading and insecure, so we fail honestly instead.
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "transaction broadcast not implemented - cannot generate tx hash without broadcasting",
	})
}

func (ws *WalletService) GetUserWallets(c *gin.Context) {
	userID := c.Param("user_id")
	
	wallets := []*Wallet{}
	for _, w := range ws.wallets {
		if w.MasterID == userID {
			wallets = append(wallets, w)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"wallets": wallets,
	})
}

func (ws *WalletService) GetSupportedChains(c *gin.Context) {
	chains := []map[string]interface{}{
		{"id": "ethereum", "name": "Ethereum", "symbol": "ETH", "decimals": 18},
		{"id": "polygon", "name": "Polygon", "symbol": "MATIC", "decimals": 18},
		{"id": "arbitrum", "name": "Arbitrum", "symbol": "ETH", "decimals": 18},
		{"id": "optimism", "name": "Optimism", "symbol": "ETH", "decimals": 18},
		{"id": "avalanche", "name": "Avalanche", "symbol": "AVAX", "decimals": 18},
		{"id": "bsc", "name": "BNB Chain", "symbol": "BNB", "decimals": 18},
		{"id": "base", "name": "Base", "symbol": "ETH", "decimals": 18},
		{"id": "solana", "name": "Solana", "symbol": "SOL", "decimals": 9},
		{"id": "tron", "name": "TRON", "symbol": "TRX", "decimals": 6},
		{"id": "aptos", "name": "Aptos", "symbol": "APT", "decimals": 8},
	}
	
	c.JSON(http.StatusOK, chains)
}

func (ws *WalletService) Swap(c *gin.Context) {
	var req SwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// In production, call DEX or aggregator
	// For now, return simulated swap
	swapID := uuid.New().String()
	
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"swap_id":       swapID,
		"from_token":    req.FromToken,
		"to_token":      req.ToToken,
		"from_amount":   req.FromAmount,
		"to_amount":     fmt.Sprintf("%.6f", float64(1.5)), // Simulated
		"price_impact":  "0.1%",
		"chain":         req.Chain,
		"status":        "pending",
	})
}

func (ws *WalletService) Stake(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	stakeID := uuid.New().String()
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"stake_id":    stakeID,
		"amount":      req["amount"],
		"validator":   req["validator"],
		"chain":       req["chain"],
		"reward_rate": "5.2%",
		"status":      "pending",
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func (ws *WalletService) createDefaultWallets(userID string) {
	chains := []string{"ethereum", "polygon", "arbitrum", "optimism", "avalanche", "bsc", "base", "solana", "tron", "aptos"}
	
	for _, chain := range chains {
		mnemonic, _ := generateMnemonic()
		seed, _ := mnemonicToSeed(mnemonic, "")
		privateKey, _ := deriveKeyFromSeed(seed, "m/44'/60'/0'/0/0")
		
		address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
		
		wallet := &Wallet{
			ID:         uuid.New().String(),
			MasterID:   userID,
			Chain:      chain,
			Address:    address,
			PublicKey:  hex.EncodeToString(elliptic.Marshal(elliptic.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y)),
			Type:       "user",
			Status:     "active",
			Balance:    make(map[string]string),
			Nonce:      0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		
		ws.wallets[wallet.ID] = wallet
	}
}

// ============================================================================
// Middleware
// ============================================================================

func (ws *WalletService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}
		
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ws.validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		
		c.Set("user_id", claims.UserID)
		c.Set("wallet_id", claims.WalletID)
		c.Set("permissions", claims.Permissions)
		c.Next()
	}
}

func (ws *WalletService) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simplified rate limiting
		c.Next()
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("TigerWallet Backend Service - Wallet Management")
	log.Println("==============================================")
	log.Printf("Starting on port %d", cfg.Port)

	ws := NewWalletService()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "wallet-service",
			"timestamp": time.Now().Unix(),
		})
	})

	// Public routes
	r.POST("/api/v1/register", ws.Register)
	r.POST("/api/v1/login", ws.Login)

	// Protected routes
	api := r.Group("/api/v1")
	api.Use(ws.AuthMiddleware())
	{
		// Wallet management
		api.POST("/wallets", ws.CreateWallet)
		api.POST("/wallets/import", ws.ImportWallet)
		api.GET("/users/:user_id/wallets", ws.GetUserWallets)
		api.GET("/wallets/:id", ws.GetBalance)
		api.GET("/wallets/:id/transactions", ws.GetTransactions)
		
		// Transactions
		api.POST("/send", ws.SendTransaction)
		
		// Swap
		api.POST("/swap", ws.Swap)
		
		// Staking
		api.POST("/stake", ws.Stake)
		
		// Chains
		api.GET("/chains", ws.GetSupportedChains)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
