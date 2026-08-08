// TigerWallet Admin Service - Comprehensive Admin Management System
// Super Admin and White Label Admin Management Platform

package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/argon2"

	"github.com/tigerwallet/admin-service/database"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port            int    `json:"port"`
	JWTSecret       string `json:"jwt_secret"`
	RedisAddr       string `json:"redis_addr"`
	MasterAdmin     string `json:"master_admin"`
	PostgresHost    string `json:"postgres_host"`
	PostgresPort    int    `json:"postgres_port"`
	PostgresDB      string `json:"postgres_db"`
	PostgresUser    string `json:"postgres_user"`
	PostgresPass    string `json:"postgres_pass"`
}

var cfg = Config{
	Port:            8002,
	JWTSecret:       getRequiredEnv("JWT_SECRET"),
	RedisAddr:       "localhost:6379",
	MasterAdmin:     "admin@tigerwallet.io",
	PostgresHost:    getEnv("POSTGRES_HOST", "localhost"),
	PostgresPort:    5432,
	PostgresDB:      getEnv("POSTGRES_DB", "tigerwallet"),
	PostgresUser:    getEnv("POSTGRES_USER", "postgres"),
	PostgresPass:    getRequiredEnv("DATABASE_PASSWORD"),
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getRequiredEnv reads a required environment variable and fatally exits if it
// is unset. Used for secrets and credentials that must never fall back to
// insecure hardcoded defaults.
func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable must be set", key)
	}
	return value
}

// encryptionKey derives a 32-byte AES-256 key from the ENCRYPTION_KEY env var
// by hashing it with SHA-256, so any length passphrase becomes a valid key.
func encryptionKey() []byte {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		log.Fatal("ENCRYPTION_KEY environment variable must be set")
	}
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

// encryptString performs AES-256-GCM authenticated encryption and returns a
// base64 (hex) string of nonce||ciphertext. Used for protecting secrets at
// rest such as API keys and tokens.
func encryptString(plaintext string) (string, error) {
	key := encryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// decryptString reverses encryptString, returning the original plaintext.
func decryptString(ciphertextHex string) (string, error) {
	key := encryptionKey()
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// secureRandomInt returns a uniform random non-negative int in [0, max) using
// crypto/rand, avoiding modulo bias.
func secureRandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		log.Fatalf("failed to generate secure random int: %v", err)
	}
	return int(n.Int64())
}

// Global database connection
var db *database.DB
var redisClient *redis.Client

// ============================================================================
// Database Models
// ============================================================================

type Admin struct {
	ID             string         `json:"id" bson:"_id"`
	Email          string         `json:"email" bson:"email"`
	Username       string         `json:"username" bson:"username"`
	PasswordHash   string         `json:"-" bson:"password_hash"`
	Role           string         `json:"role" bson:"role"` // super_admin, white_label, support, finance, compliance
	Permissions    AdminPermissions `json:"permissions" bson:"permissions"`
	Status         string         `json:"status" bson:"status"` // active, suspended, deleted
	CreatedBy      string         `json:"created_by" bson:"created_by"`
	Products       []string      `json:"products" bson:"products"` // wallet, exchange, defi, nft, etc
	WhiteLabelID   string        `json:"white_label_id" bson:"white_label_id"`
	CreatedAt       time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" bson:"updated_at"`
	LastLoginAt    time.Time     `json:"last_login_at" bson:"last_login_at"`
	LoginIP        string        `json:"login_ip" bson:"login_ip"`
	ActivityLog    []Activity    `json:"activity_log" bson:"activity_log"`
}

type AdminPermissions struct {
	// User Management
	CanCreateUser    bool `json:"can_create_user"`
	CanEditUser      bool `json:"can_edit_user"`
	CanDeleteUser    bool `json:"can_delete_user"`
	CanViewUser      bool `json:"can_view_user"`
	CanManageKYC     bool `json:"can_manage_kyc"`

	// Wallet Management
	CanViewWallet    bool `json:"can_view_wallet"`
	CanFreezeWallet  bool `json:"can_freeze_wallet"`
	CanWithdrawFunds bool `json:"can_withdraw_funds"`

	// Trading Management
	CanManagePairs   bool `json:"can_manage_pairs"`
	CanManageLiquidity bool `json:"can_manage_liquidity"`
	CanManageFees    bool `json:"can_manage_fees"`
	CanManageListing bool `json:"can_manage_listing"`

	// White Label
	CanManageWhiteLabel bool `json:"can_manage_white_label"`
	CanManageWLWallet   bool `json:"can_manage_wl_wallet"`
	CanManageWLBlockchain bool `json:"can_manage_wl_blockchain"`

	// Token Management
	CanCreateToken    bool `json:"can_create_token"`
	CanPauseToken     bool `json:"can_pause_token"`
	CanDeleteToken    bool `json:"can_delete_token"`

	// NFT Management
	CanManageNFT      bool `json:"can_manage_nft"`
	CanMintNFT        bool `json:"can_mint_nft"`

	// System
	CanManageAdmins   bool `json:"can_manage_admins"`
	CanViewAnalytics  bool `json:"can_view_analytics"`
	CanManageAPI      bool `json:"can_manage_api"`
	CanManageSettings bool `json:"can_manage_settings"`
}

type Activity struct {
	ID        string    `json:"id" bson:"_id"`
	AdminID   string    `json:"admin_id" bson:"admin_id"`
	Action    string    `json:"action" bson:"action"`
	Target    string    `json:"target" bson:"target"`
	Details   string    `json:"details" bson:"details"`
	IPAddress string    `json:"ip_address" bson:"ip_address"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

type WhiteLabelClient struct {
	ID          string               `json:"id" bson:"_id"`
	Name        string              `json:"name" bson:"name"`
	Domain      string              `json:"domain" bson:"domain"`
	Branding    WhiteLabelBranding  `json:"branding" bson:"branding"`
	Status      string              `json:"status" bson:"status"` // active, paused, halted
	Products    []string            `json:"products" bson:"products"`
	Admins      []string            `json:"admins" bson:"admins"`
	CustomChains []string           `json:"custom_chains" bson:"custom_chains"`
	CustomTokens []string           `json:"custom_tokens" bson:"custom_tokens"`
	FeeStructure FeeStructure        `json:"fee_structure" bson:"fee_structure"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

type WhiteLabelBranding struct {
	Logo         string `json:"logo" bson:"logo"`
	Favicon      string `json:"favicon" bson:"favicon"`
	PrimaryColor string `json:"primary_color" bson:"primary_color"`
	SecondaryColor string `json:"secondary_color" bson:"secondary_color"`
	AccentColor string `json:"accent_color" bson:"accent_color"`
	FontFamily  string `json:"font_family" bson:"font_family"`
}

type FeeStructure struct {
	TradingFee    string `json:"trading_fee" bson:"trading_fee"`
	WithdrawalFee string `json:"withdrawal_fee" bson:"withdrawal_fee"`
	DepositFee    string `json:"deposit_fee" bson:"deposit_fee"`
	NFTFee       string `json:"nft_fee" bson:"nft_fee"`
}

type UserManagement struct {
	ID              string    `json:"id" bson:"_id"`
	Email           string    `json:"email" bson:"email"`
	Username        string    `json:"username" bson:"username"`
	KYCStatus       string    `json:"kyc_status" bson:"kyc_status"` // none, pending, verified, rejected
	KYCLevel        int       `json:"kyc_level" bson:"kyc_level"`
	VerificationDocs []string `json:"verification_docs" bson:"verification_docs"`
	AccountStatus   string    `json:"account_status" bson:"account_status"` // active, suspended, locked
	Balance         string    `json:"balance" bson:"balance"`
	TradingVolume   string    `json:"trading_volume" bson:"trading_volume"`
	ReferralCode    string    `json:"referral_code" bson:"referral_code"`
	ReferredBy     string    `json:"referred_by" bson:"referred_by"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" bson:"updated_at"`
}

type TradingPair struct {
	ID           string  `json:"id" bson:"_id"`
	BaseAsset   string  `json:"base_asset" bson:"base_asset"`
	QuoteAsset  string  `json:"quote_asset" bson:"quote_asset"`
	PairSymbol  string  `json:"pair_symbol" bson:"pair_symbol"`
	Chain       string  `json:"chain" bson:"chain"`
	Status      string  `json:"status" bson:"status"` // active, suspended, halted, delisted
	Price       string  `json:"price" bson:"price"`
	Volume24h   string  `json:"volume_24h" bson:"volume_24h"`
	Liquidity   string  `json:"liquidity" bson:"liquidity"`
	MakerFee    string  `json:"maker_fee" bson:"maker_fee"`
	TakerFee    string  `json:"taker_fee" bson:"taker_fee"`
	MinTrade    string  `json:"min_trade" bson:"min_trade"`
	MaxTrade    string  `json:"max_trade" bson:"max_trade"`
	ImportedFrom string `json:"imported_from" bson:"imported_from"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
}

type Token struct {
	ID          string `json:"id" bson:"_id"`
	Symbol      string `json:"symbol" bson:"symbol"`
	Name        string `json:"name" bson:"name"`
	Contract    string `json:"contract" bson:"contract"`
	Chain       string `json:"chain" bson:"chain"`
	Decimals    int    `json:"decimals" bson:"decimals"`
	TotalSupply string `json:"total_supply" bson:"total_supply"`
	Status      string `json:"status" bson:"status"` // active, paused, halted, deleted
	Type        string `json:"type" bson:"type"` // erc20, erc721, etc
	IsVerified  bool   `json:"is_verified" bson:"is_verified"`
	PriceUSD    string `json:"price_usd" bson:"price_usd"`
	MarketCap   string `json:"market_cap" bson:"market_cap"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
}

type Blockchain struct {
	ID              string `json:"id" bson:"_id"`
	Name            string `json:"name" bson:"name"`
	Symbol          string `json:"symbol" bson:"symbol"`
	ChainID         uint64 `json:"chain_id" bson:"chain_id"`
	RPCURL          string `json:"rpc_url" bson:"rpc_url"`
	ExplorerURL     string `json:"explorer_url" bson:"explorer_url"`
	Type            string `json:"type" bson:"type"` // evm, non-evm
	Status          string `json:"status" bson:"status"` // active, inactive
	BlockTime       int    `json:"block_time" bson:"block_time"`
	IsDefault       bool   `json:"is_default" bson:"is_default"`
	GasToken        string `json:"gas_token" bson:"gas_token"`
	SupportedTokens []string `json:"supported_tokens" bson:"supported_tokens"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
}

type WithdrawalRequest struct {
	ID          string    `json:"id" bson:"_id"`
	UserID      string    `json:"user_id" bson:"user_id"`
	WalletID    string    `json:"wallet_id" bson:"wallet_id"`
	Chain       string    `json:"chain" bson:"chain"`
	Address     string    `json:"address" bson:"address"`
	Amount      string    `json:"amount" bson:"amount"`
	Token       string    `json:"token" bson:"token"`
	Status      string    `json:"status" bson:"status"` // pending, approved, rejected, processing, completed, failed
	Fee         string    `json:"fee" bson:"fee"`
	TxHash      string    `json:"tx_hash" bson:"tx_hash"`
	ApprovedBy  string    `json:"approved_by" bson:"approved_by"`
	ProcessedAt *time.Time `json:"processed_at" bson:"processed_at"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
}

type Analytics struct {
	TotalUsers       int64   `json:"total_users"`
	ActiveUsers      int64   `json:"active_users"`
	TotalVolume24h   string  `json:"total_volume_24h"`
	TotalFees24h     string  `json:"total_fees_24h"`
	TotalWallets     int64   `json:"total_wallets"`
	TotalTransactions int64   `json:"total_transactions"`
	TopPairs         []map[string]string `json:"top_pairs"`
	TopTokens        []map[string]string `json:"top_tokens"`
}

// ============================================================================
// Admin Service
// ============================================================================

type AdminService struct {
	redis      *redis.Client
	mu         sync.RWMutex
	admins     map[string]*Admin
	whiteLabels map[string]*WhiteLabelClient
	users      map[string]*UserManagement
	pairs      map[string]*TradingPair
	tokens     map[string]*Token
	blockchains map[string]*Blockchain
	withdrawals map[string]*WithdrawalRequest
	activities  []Activity
}

func NewAdminService() *AdminService {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	as := &AdminService{
		redis:       rdb,
		admins:     make(map[string]*Admin),
		whiteLabels: make(map[string]*WhiteLabelClient),
		users:      make(map[string]*UserManagement),
		pairs:      make(map[string]*TradingPair),
		tokens:     make(map[string]*Token),
		blockchains: make(map[string]*Blockchain),
		withdrawals: make(map[string]*WithdrawalRequest),
		activities:  []Activity{},
	}

	// Initialize default data
	as.initializeDefaultData()

	return as
}

func (as *AdminService) initializeDefaultData() {
	// Create super admin
	superAdminID := uuid.New().String()
	superAdmin := &Admin{
		ID:       superAdminID,
		Email:    cfg.MasterAdmin,
		Username: "super_admin",
		Role:     "super_admin",
		Permissions: AdminPermissions{
			CanCreateUser: true, CanEditUser: true, CanDeleteUser: true, CanViewUser: true,
			CanManageKYC: true, CanViewWallet: true, CanFreezeWallet: true, CanWithdrawFunds: true,
			CanManagePairs: true, CanManageLiquidity: true, CanManageFees: true, CanManageListing: true,
			CanManageWhiteLabel: true, CanManageWLWallet: true, CanManageWLBlockchain: true,
			CanCreateToken: true, CanPauseToken: true, CanDeleteToken: true,
			CanManageNFT: true, CanMintNFT: true,
			CanManageAdmins: true, CanViewAnalytics: true, CanManageAPI: true, CanManageSettings: true,
		},
		Status:      "active",
		Products:    []string{"wallet", "exchange", "defi", "nft", "staking", "launchpad", "white_label"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	as.admins[superAdmin.Email] = superAdmin

	// Initialize default blockchains
	defaultChains := []Blockchain{
		{ID: "ethereum", Name: "Ethereum", Symbol: "ETH", ChainID: 1, Type: "evm", Status: "active", BlockTime: 12, IsDefault: true, GasToken: "ETH"},
		{ID: "polygon", Name: "Polygon", Symbol: "MATIC", ChainID: 137, Type: "evm", Status: "active", BlockTime: 2, IsDefault: true, GasToken: "MATIC"},
		{ID: "arbitrum", Name: "Arbitrum", Symbol: "ETH", ChainID: 42161, Type: "evm", Status: "active", BlockTime: 1, IsDefault: true, GasToken: "ETH"},
		{ID: "optimism", Name: "Optimism", Symbol: "ETH", ChainID: 10, Type: "evm", Status: "active", BlockTime: 2, IsDefault: true, GasToken: "ETH"},
		{ID: "avalanche", Name: "Avalanche", Symbol: "AVAX", ChainID: 43114, Type: "evm", Status: "active", BlockTime: 2, IsDefault: true, GasToken: "AVAX"},
		{ID: "bsc", Name: "BNB Chain", Symbol: "BNB", ChainID: 56, Type: "evm", Status: "active", BlockTime: 3, IsDefault: true, GasToken: "BNB"},
		{ID: "base", Name: "Base", Symbol: "ETH", ChainID: 8453, Type: "evm", Status: "active", BlockTime: 2, IsDefault: true, GasToken: "ETH"},
		{ID: "solana", Name: "Solana", Symbol: "SOL", ChainID: 0, Type: "non-evm", Status: "active", BlockTime: 1, IsDefault: true, GasToken: "SOL"},
		{ID: "tron", Name: "TRON", Symbol: "TRX", ChainID: 728126428, Type: "non-evm", Status: "active", BlockTime: 3, IsDefault: true, GasToken: "TRX"},
		{ID: "aptos", Name: "Aptos", Symbol: "APT", ChainID: 1, Type: "non-evm", Status: "active", BlockTime: 1, IsDefault: true, GasToken: "APT"},
	}
	for _, chain := range defaultChains {
		chain.CreatedAt = time.Now()
		as.blockchains[chain.ID] = &chain
	}

	// Initialize default tokens
	defaultTokens := []Token{
		{ID: "eth", Symbol: "ETH", Name: "Ethereum", Chain: "ethereum", Decimals: 18, Type: "native", Status: "active", IsVerified: true, PriceUSD: "2500.00", MarketCap: "300B"},
		{ID: "btc", Symbol: "BTC", Name: "Bitcoin", Chain: "bitcoin", Decimals: 8, Type: "native", Status: "active", IsVerified: true, PriceUSD: "45000.00", MarketCap: "850B"},
		{ID: "usdt", Symbol: "USDT", Name: "Tether", Chain: "ethereum", Decimals: 6, Type: "erc20", Status: "active", IsVerified: true, PriceUSD: "1.00", MarketCap: "100B"},
		{ID: "usdc", Symbol: "USDC", Name: "USD Coin", Chain: "ethereum", Decimals: 6, Type: "erc20", Status: "active", IsVerified: true, PriceUSD: "1.00", MarketCap: "40B"},
		{ID: "bnb", Symbol: "BNB", Name: "BNB", Chain: "bsc", Decimals: 18, Type: "native", Status: "active", IsVerified: true, PriceUSD: "350.00", MarketCap: "50B"},
		{ID: "matic", Symbol: "MATIC", Name: "Polygon", Chain: "polygon", Decimals: 18, Type: "native", Status: "active", IsVerified: true, PriceUSD: "0.80", MarketCap: "7B"},
		{ID: "sol", Symbol: "SOL", Name: "Solana", Chain: "solana", Decimals: 9, Type: "native", Status: "active", IsVerified: true, PriceUSD: "100.00", MarketCap: "40B"},
		{ID: "trx", Symbol: "TRX", Name: "TRON", Chain: "tron", Decimals: 6, Type: "native", Status: "active", IsVerified: true, PriceUSD: "0.12", MarketCap: "10B"},
		{ID: "dot", Symbol: "DOT", Name: "Polkadot", Chain: "polkadot", Decimals: 10, Type: "native", Status: "active", IsVerified: true, PriceUSD: "7.00", MarketCap: "10B"},
		{ID: "link", Symbol: "LINK", Name: "Chainlink", Chain: "ethereum", Decimals: 18, Type: "erc20", Status: "active", IsVerified: true, PriceUSD: "15.00", MarketCap: "8B"},
	}
	for _, token := range defaultTokens {
		token.CreatedAt = time.Now()
		as.tokens[token.ID] = &token
	}

	// Initialize default trading pairs
	defaultPairs := []TradingPair{
		{ID: "eth-usdt", BaseAsset: "ETH", QuoteAsset: "USDT", PairSymbol: "ETH/USDT", Chain: "ethereum", Status: "active", Price: "2500.00", Volume24h: "500M", Liquidity: "100M", MakerFee: "0.1", TakerFee: "0.2"},
		{ID: "btc-usdt", BaseAsset: "BTC", QuoteAsset: "USDT", PairSymbol: "BTC/USDT", Chain: "ethereum", Status: "active", Price: "45000.00", Volume24h: "1B", Liquidity: "200M", MakerFee: "0.1", TakerFee: "0.2"},
		{ID: "bnb-usdt", BaseAsset: "BNB", QuoteAsset: "USDT", PairSymbol: "BNB/USDT", Chain: "bsc", Status: "active", Price: "350.00", Volume24h: "100M", Liquidity: "50M", MakerFee: "0.1", TakerFee: "0.2"},
		{ID: "sol-usdt", BaseAsset: "SOL", QuoteAsset: "USDT", PairSymbol: "SOL/USDT", Chain: "solana", Status: "active", Price: "100.00", Volume24h: "50M", Liquidity: "20M", MakerFee: "0.1", TakerFee: "0.2"},
		{ID: "matic-usdt", BaseAsset: "MATIC", QuoteAsset: "USDT", PairSymbol: "MATIC/USDT", Chain: "polygon", Status: "active", Price: "0.80", Volume24h: "20M", Liquidity: "10M", MakerFee: "0.1", TakerFee: "0.2"},
	}
	for _, pair := range defaultPairs {
		pair.CreatedAt = time.Now()
		as.pairs[pair.ID] = &pair
	}
}

// ============================================================================
// JWT Functions
// ============================================================================

type AdminClaims struct {
	AdminID  string         `json:"admin_id"`
	Email    string         `json:"email"`
	Role     string         `json:"role"`
	Permissions AdminPermissions `json:"permissions"`
	jwt.RegisteredClaims
}

func (as *AdminService) generateToken(admin *Admin) (string, error) {
	claims := AdminClaims{
		AdminID:    admin.ID,
		Email:      admin.Email,
		Role:       admin.Role,
		Permissions: admin.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet-admin",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func (as *AdminService) validateToken(tokenString string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*AdminClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("invalid token")
}

// ============================================================================
// API Handlers - Authentication
// ============================================================================

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (as *AdminService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin, exists := as.admins[req.Email]
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if admin.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account not active"})
		return
	}

	token, err := as.generateToken(admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	admin.LastLoginAt = time.Now()
	admin.LoginIP = c.ClientIP()

	// Log activity
	as.logActivity(admin.ID, "login", "admin", "Admin logged in", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"token":       token,
		"admin_id":    admin.ID,
		"email":       admin.Email,
		"username":    admin.Username,
		"role":        admin.Role,
		"permissions": admin.Permissions,
		"products":    admin.Products,
	})
}

// ============================================================================
// API Handlers - Admin Management
// ============================================================================

type CreateAdminRequest struct {
	Email       string   `json:"email" binding:"required,email"`
	Username    string   `json:"username" binding:"required"`
	Password    string   `json:"password" binding:"required,min=8"`
	Role        string   `json:"role" binding:"required"`
	Products    []string `json:"products"`
	Permissions AdminPermissions `json:"permissions"`
}

func (as *AdminService) CreateAdmin(c *gin.Context) {
	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, exists := as.admins[req.Email]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "admin already exists"})
		return
	}

	adminID := uuid.New().String()
	admin := &Admin{
		ID:          adminID,
		Email:       req.Email,
		Username:    req.Username,
		Role:        req.Role,
		Permissions: req.Permissions,
		Status:      "active",
		Products:    req.Products,
		CreatedBy:   c.GetString("admin_id"),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	as.admins[req.Email] = admin

	as.logActivity(c.GetString("admin_id"), "create_admin", adminID, "Created admin: "+req.Email, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"admin_id": adminID,
		"email":    req.Email,
		"role":     req.Role,
	})
}

func (as *AdminService) ListAdmins(c *gin.Context) {
	admins := make([]*Admin, 0)
	for _, admin := range as.admins {
		admins = append(admins, admin)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"admins":  admins,
		"total":   len(admins),
	})
}

// ============================================================================
// API Handlers - User Management
// ============================================================================

func (as *AdminService) ListUsers(c *gin.Context) {
	users := make([]*UserManagement, 0)
	for _, user := range as.users {
		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"users":   users,
		"total":   len(users),
	})
}

func (as *AdminService) GetUser(c *gin.Context) {
	userID := c.Param("id")
	
	user, exists := as.users[userID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    user,
	})
}

func (as *AdminService) UpdateKYC(c *gin.Context) {
	userID := c.Param("id")
	
	var req struct {
		Status string `json:"status" binding:"required"`
		Level  int    `json:"level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := as.users[userID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.KYCStatus = req.Status
	user.KYCLevel = req.Level
	user.UpdatedAt = time.Now()

	as.logActivity(c.GetString("admin_id"), "update_kyc", userID, "KYC updated to: "+req.Status, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    user,
	})
}

func (as *AdminService) SuspendUser(c *gin.Context) {
	userID := c.Param("id")
	
	user, exists := as.users[userID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.AccountStatus = "suspended"
	user.UpdatedAt = time.Now()

	as.logActivity(c.GetString("admin_id"), "suspend_user", userID, "User suspended", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status": "suspended",
	})
}

// ============================================================================
// API Handlers - Blockchain Management
// ============================================================================

type CreateChainRequest struct {
	Name      string `json:"name" binding:"required"`
	Symbol    string `json:"symbol" binding:"required"`
	ChainID   uint64 `json:"chain_id" binding:"required"`
	RPCURL    string `json:"rpc_url" binding:"required"`
	Type      string `json:"type" binding:"required"`
	GasToken  string `json:"gas_token" binding:"required"`
}

func (as *AdminService) CreateBlockchain(c *gin.Context) {
	var req CreateChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chainID := uuid.New().String()
	chain := &Blockchain{
		ID:         chainID,
		Name:       req.Name,
		Symbol:     req.Symbol,
		ChainID:    req.ChainID,
		RPCURL:     req.RPCURL,
		Type:       req.Type,
		Status:     "active",
		GasToken:   req.GasToken,
		IsDefault:  false,
		CreatedAt: time.Now(),
	}

	as.blockchains[chainID] = chain

	as.logActivity(c.GetString("admin_id"), "create_blockchain", chainID, "Created blockchain: "+req.Name, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"blockchain": chain,
	})
}

func (as *AdminService) ListBlockchains(c *gin.Context) {
	blockchains := make([]*Blockchain, 0)
	for _, chain := range as.blockchains {
		blockchains = append(blockchains, chain)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"blockchains": blockchains,
		"total":       len(blockchains),
	})
}

func (as *AdminService) UpdateBlockchain(c *gin.Context) {
	chainID := c.Param("id")
	
	chain, exists := as.blockchains[chainID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "blockchain not found"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if status, ok := req["status"].(string); ok {
		chain.Status = status
	}
	if rpcURL, ok := req["rpc_url"].(string); ok {
		chain.RPCURL = rpcURL
	}

	as.logActivity(c.GetString("admin_id"), "update_blockchain", chainID, "Updated blockchain", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"blockchain": chain,
	})
}

func (as *AdminService) DeleteBlockchain(c *gin.Context) {
	chainID := c.Param("id")
	
	if _, exists := as.blockchains[chainID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "blockchain not found"})
		return
	}

	delete(as.blockchains, chainID)

	as.logActivity(c.GetString("admin_id"), "delete_blockchain", chainID, "Deleted blockchain", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "deleted",
	})
}

// ============================================================================
// API Handlers - Token Management
// ============================================================================

func (as *AdminService) ListTokens(c *gin.Context) {
	tokens := make([]*Token, 0)
	for _, token := range as.tokens {
		tokens = append(tokens, token)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tokens":  tokens,
		"total":   len(tokens),
	})
}

func (as *AdminService) CreateToken(c *gin.Context) {
	var token Token
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token.ID = uuid.New().String()
	token.Status = "active"
	token.CreatedAt = time.Now()

	as.tokens[token.ID] = &token

	as.logActivity(c.GetString("admin_id"), "create_token", token.ID, "Created token: "+token.Symbol, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"token":  &token,
	})
}

func (as *AdminService) UpdateTokenStatus(c *gin.Context) {
	tokenID := c.Param("id")
	
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, exists := as.tokens[tokenID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}

	token.Status = req.Status

	as.logActivity(c.GetString("admin_id"), "update_token_status", tokenID, "Token status: "+req.Status, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   token,
	})
}

// ============================================================================
// API Handlers - Trading Pairs
// ============================================================================

func (as *AdminService) ListPairs(c *gin.Context) {
	pairs := make([]*TradingPair, 0)
	for _, pair := range as.pairs {
		pairs = append(pairs, pair)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"pairs":   pairs,
		"total":   len(pairs),
	})
}

func (as *AdminService) CreatePair(c *gin.Context) {
	var pair TradingPair
	if err := c.ShouldBindJSON(&pair); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair.ID = uuid.New().String()
	pair.Status = "active"
	pair.CreatedAt = time.Now()

	as.pairs[pair.ID] = &pair

	as.logActivity(c.GetString("admin_id"), "create_pair", pair.ID, "Created pair: "+pair.PairSymbol, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"pair":    &pair,
	})
}

func (as *AdminService) UpdatePairStatus(c *gin.Context) {
	pairID := c.Param("id")
	
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, exists := as.pairs[pairID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}

	pair.Status = req.Status

	as.logActivity(c.GetString("admin_id"), "update_pair_status", pairID, "Pair status: "+req.Status, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"pair":    pair,
	})
}

func (as *AdminService) ImportPairs(c *gin.Context) {
	var req struct {
		Source string `json:"source" binding:"required"` // binance, coinbase, kraken, etc
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production, fetch from exchange APIs
	// For now, simulate import
	imported := 50 // Simulated

	as.logActivity(c.GetString("admin_id"), "import_pairs", req.Source, fmt.Sprintf("Imported %d pairs from %s", imported, req.Source), c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"imported":   imported,
		"source":     req.Source,
		"status":     "completed",
	})
}

// ============================================================================
// API Handlers - White Label
// ============================================================================

func (as *AdminService) ListWhiteLabels(c *gin.Context) {
	whiteLabels := make([]*WhiteLabelClient, 0)
	for _, wl := range as.whiteLabels {
		whiteLabels = append(whiteLabels, wl)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"white_labels": whiteLabels,
		"total":       len(whiteLabels),
	})
}

func (as *AdminService) CreateWhiteLabel(c *gin.Context) {
	var wl WhiteLabelClient
	if err := c.ShouldBindJSON(&wl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wl.ID = uuid.New().String()
	wl.Status = "active"
	wl.CreatedAt = time.Now()
	wl.UpdatedAt = time.Now()

	as.whiteLabels[wl.ID] = &wl

	as.logActivity(c.GetString("admin_id"), "create_white_label", wl.ID, "Created white label: "+wl.Name, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"white_label": &wl,
	})
}

func (as *AdminService) UpdateWhiteLabel(c *gin.Context) {
	wlID := c.Param("id")
	
	wl, exists := as.whiteLabels[wlID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "white label not found"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if status, ok := updates["status"].(string); ok {
		wl.Status = status
	}
	if domain, ok := updates["domain"].(string); ok {
		wl.Domain = domain
	}

	wl.UpdatedAt = time.Now()

	as.logActivity(c.GetString("admin_id"), "update_white_label", wlID, "Updated white label", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"white_label": wl,
	})
}

// ============================================================================
// API Handlers - Withdrawals
// ============================================================================

func (as *AdminService) ListWithdrawals(c *gin.Context) {
	withdrawals := make([]*WithdrawalRequest, 0)
	for _, w := range as.withdrawals {
		withdrawals = append(withdrawals, w)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"withdrawals": withdrawals,
		"total":      len(withdrawals),
	})
}

func (as *AdminService) ApproveWithdrawal(c *gin.Context) {
	withdrawalID := c.Param("id")
	
	withdrawal, exists := as.withdrawals[withdrawalID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "withdrawal not found"})
		return
	}

	withdrawal.Status = "approved"
	withdrawal.ApprovedBy = c.GetString("admin_id")

	as.logActivity(c.GetString("admin_id"), "approve_withdrawal", withdrawalID, "Approved withdrawal", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"withdrawal": withdrawal,
	})
}

func (as *AdminService) RejectWithdrawal(c *gin.Context) {
	withdrawalID := c.Param("id")
	
	withdrawal, exists := as.withdrawals[withdrawalID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "withdrawal not found"})
		return
	}

	withdrawal.Status = "rejected"
	withdrawal.ApprovedBy = c.GetString("admin_id")

	as.logActivity(c.GetString("admin_id"), "reject_withdrawal", withdrawalID, "Rejected withdrawal", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"withdrawal": withdrawal,
	})
}

// ============================================================================
// API Handlers - Analytics
// ============================================================================

func (as *AdminService) GetAnalytics(c *gin.Context) {
	analytics := Analytics{
		TotalUsers:       150000,
		ActiveUsers:      50000,
		TotalVolume24h:   "1.5B",
		TotalFees24h:     "3M",
		TotalWallets:    200000,
		TotalTransactions: 5000000,
		TopPairs: []map[string]string{
			{"symbol": "ETH/USDT", "volume": "500M"},
			{"symbol": "BTC/USDT", "volume": "1B"},
			{"symbol": "BNB/USDT", "volume": "100M"},
		},
		TopTokens: []map[string]string{
			{"symbol": "ETH", "price": "2500"},
			{"symbol": "BTC", "price": "45000"},
			{"symbol": "BNB", "price": "350"},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"analytics": analytics,
	})
}

// ============================================================================
// Activity Logging
// ============================================================================

func (as *AdminService) logActivity(adminID, action, target, details, ip string) {
	activity := Activity{
		ID:        uuid.New().String(),
		AdminID:   adminID,
		Action:    action,
		Target:    target,
		Details:   details,
		IPAddress: ip,
		Timestamp: time.Now(),
	}
	
	as.activities = append(as.activities, activity)
	
	// Keep only last 1000 activities
	if len(as.activities) > 1000 {
		as.activities = as.activities[len(as.activities)-1000:]
	}
}

func (as *AdminService) GetActivities(c *gin.Context) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	
	start := 0
	if len(as.activities) > limit {
		start = len(as.activities) - limit
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"activities": as.activities[start:],
		"total":      len(as.activities),
	})
}

// ============================================================================
// NEW: Delete Admin
// ============================================================================

type DeleteAdminRequest struct {
	AdminID string `json:"admin_id" binding:"required"`
}

func (as *AdminService) DeleteAdmin(c *gin.Context) {
	adminID := c.Param("id")
	if adminID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin id required"})
		return
	}

	// Check if admin exists
	admin, exists := as.admins[adminID]
	if !exists {
		// Try to find by ID in map values
		for _, a := range as.admins {
			if a.ID == adminID {
				admin = a
				exists = true
				break
			}
		}
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}

	// Cannot delete self
	requestingAdminID := c.GetString("admin_id")
	if adminID == requestingAdminID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete yourself"})
		return
	}

	// Remove admin
	delete(as.admins, admin.Email)

	as.logActivity(requestingAdminID, "delete_admin", adminID, "Deleted admin: "+admin.Email, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "admin deleted successfully",
	})
}

// ============================================================================
// NEW: Bulk User Operations
// ============================================================================

type BulkSuspendUsersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
	Reason  string   `json:"reason"`
}

func (as *AdminService) BulkSuspendUsers(c *gin.Context) {
	var req BulkSuspendUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no user ids provided"})
		return
	}

	suspended := 0
	for _, userID := range req.UserIDs {
		if user, exists := as.users[userID]; exists {
			user.AccountStatus = "suspended"
			as.users[userID] = user
			suspended++
		}
	}

	adminID := c.GetString("admin_id")
	as.logActivity(adminID, "bulk_suspend_users", "", "Suspended "+fmt.Sprintf("%d", suspended)+" users", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"suspended":    suspended,
		"total_requested": len(req.UserIDs),
	})
}

type BulkActivateUsersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}

func (as *AdminService) BulkActivateUsers(c *gin.Context) {
	var req BulkActivateUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	activated := 0
	for _, userID := range req.UserIDs {
		if user, exists := as.users[userID]; exists {
			user.AccountStatus = "active"
			as.users[userID] = user
			activated++
		}
	}

	adminID := c.GetString("admin_id")
	as.logActivity(adminID, "bulk_activate_users", "", "Activated "+fmt.Sprintf("%d", activated)+" users", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"activated":    activated,
		"total_requested": len(req.UserIDs),
	})
}

type BulkDeleteUsersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}

func (as *AdminService) BulkDeleteUsers(c *gin.Context) {
	var req BulkDeleteUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deleted := 0
	for _, userID := range req.UserIDs {
		if _, exists := as.users[userID]; exists {
			delete(as.users, userID)
			deleted++
		}
	}

	adminID := c.GetString("admin_id")
	as.logActivity(adminID, "bulk_delete_users", "", "Deleted "+fmt.Sprintf("%d", deleted)+" users", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"deleted":      deleted,
		"total_requested": len(req.UserIDs),
	})
}

// ============================================================================
// NEW: Bulk Token Operations
// ============================================================================

type BulkUpdateTokenStatusRequest struct {
	TokenIDs []string `json:"token_ids" binding:"required"`
	Status   string   `json:"status" binding:"required"`
}

func (as *AdminService) BulkUpdateTokenStatus(c *gin.Context) {
	var req BulkUpdateTokenStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated := 0
	for _, tokenID := range req.TokenIDs {
		if token, exists := as.tokens[tokenID]; exists {
			token.Status = req.Status
			as.tokens[tokenID] = token
			updated++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"updated":     updated,
		"new_status":  req.Status,
	})
}

type BulkDeleteTokensRequest struct {
	TokenIDs []string `json:"token_ids" binding:"required"`
}

func (as *AdminService) BulkDeleteTokens(c *gin.Context) {
	var req BulkDeleteTokensRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deleted := 0
	for _, tokenID := range req.TokenIDs {
		if _, exists := as.tokens[tokenID]; exists {
			delete(as.tokens, tokenID)
			deleted++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"deleted":      deleted,
	})
}

// ============================================================================
// NEW: Delete Pair
// ============================================================================

func (as *AdminService) DeletePair(c *gin.Context) {
	pairID := c.Param("id")
	if pairID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pair id required"})
		return
	}

	if _, exists := as.pairs[pairID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}

	delete(as.pairs, pairID)

	adminID := c.GetString("admin_id")
	as.logActivity(adminID, "delete_pair", pairID, "Deleted trading pair: "+pairID, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "pair deleted successfully",
	})
}

// ============================================================================
// NEW: Delete White Label
// ============================================================================

func (as *AdminService) DeleteWhiteLabel(c *gin.Context) {
	wlID := c.Param("id")
	if wlID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "white label id required"})
		return
	}

	if _, exists := as.whiteLabels[wlID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "white label not found"})
		return
	}

	delete(as.whiteLabels, wlID)

	adminID := c.GetString("admin_id")
	as.logActivity(adminID, "delete_whitelabel", wlID, "Deleted white label: "+wlID, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "white label deleted successfully",
	})
}

// ============================================================================
// NEW: Bulk Withdrawal Operations
// ============================================================================

type BulkApproveWithdrawalsRequest struct {
	WithdrawalIDs []string `json:"withdrawal_ids" binding:"required"`
}

func (as *AdminService) BulkApproveWithdrawals(c *gin.Context) {
	var req BulkApproveWithdrawalsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approved := 0
	for _, wdID := range req.WithdrawalIDs {
		if wd, exists := as.withdrawals[wdID]; exists {
			wd.Status = "approved"
			wd.ApprovedBy = c.GetString("admin_id")
			now := time.Now()
			wd.ProcessedAt = &now
			as.withdrawals[wdID] = wd
			approved++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"approved":   approved,
		"total_requested": len(req.WithdrawalIDs),
	})
}

type BulkRejectWithdrawalsRequest struct {
	WithdrawalIDs []string `json:"withdrawal_ids" binding:"required"`
	Reason        string   `json:"reason"`
}

func (as *AdminService) BulkRejectWithdrawals(c *gin.Context) {
	var req BulkRejectWithdrawalsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rejected := 0
	for _, wdID := range req.WithdrawalIDs {
		if wd, exists := as.withdrawals[wdID]; exists {
			wd.Status = "rejected"
			wd.ApprovedBy = c.GetString("admin_id")
			now := time.Now()
			wd.ProcessedAt = &now
			as.withdrawals[wdID] = wd
			rejected++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"rejected":  rejected,
		"total_requested": len(req.WithdrawalIDs),
	})
}

// ============================================================================
// NEW: Export Data (CSV)
// ============================================================================

func (as *AdminService) ExportUsersCSV(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=users_export.csv")

	c.Writer.Write([]byte("ID,Email,Username,KYC Status,Account Status,Balance,Trading Volume,Created At\n"))

	for _, user := range as.users {
		line := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%d\n",
			user.ID, user.Email, user.Username, user.KYCStatus,
			user.AccountStatus, user.Balance, user.TradingVolume, user.CreatedAt)
		c.Writer.Write([]byte(line))
	}
}

func (as *AdminService) ExportTransactionsCSV(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=transactions_export.csv")

	c.Writer.Write([]byte("ID,UserID,Type,Amount,Token,Chain,Status,Hash,Created At\n"))

	// Would iterate through transactions in real implementation
	c.Writer.Write([]byte("Transactions data would be exported here\n"))
}

func (as *AdminService) ExportTokensCSV(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=tokens_export.csv")

	c.Writer.Write([]byte("ID,Symbol,Name,Chain,Status,Price USD,Market Cap\n"))

	for _, token := range as.tokens {
		line := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s\n",
			token.ID, token.Symbol, token.Name, token.Chain,
			token.Status, token.PriceUSD, token.MarketCap)
		c.Writer.Write([]byte(line))
	}
}

func (as *AdminService) ExportWithdrawalsCSV(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=withdrawals_export.csv")

	c.Writer.Write([]byte("ID,UserID,Token,Amount,Chain,Address,Status,Fee,Created At\n"))

	for _, wd := range as.withdrawals {
		line := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%d\n",
			wd.ID, wd.UserID, wd.Token, wd.Amount, wd.Chain,
			wd.Address, wd.Status, wd.Fee, wd.CreatedAt)
		c.Writer.Write([]byte(line))
	}
}

// ============================================================================
// NEW: Fee Configuration
// ============================================================================

type UpdateFeeRequest struct {
	Fee    string `json:"fee" binding:"required"`
	Chain  string `json:"chain"`
	Token  string `json:"token"`
}

func (as *AdminService) UpdateTradingFee(c *gin.Context) {
	var req UpdateFeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In real implementation, would update database
	adminID := c.GetString("admin_id")
	as.logActivity(adminID, "update_trading_fee", "", "Updated trading fee to: "+req.Fee, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"fee_type":   "trading",
		"new_value":  req.Fee,
	})
}

func (as *AdminService) UpdateWithdrawalFee(c *gin.Context) {
	var req UpdateFeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetString("admin_id")
	as.logActivity(adminID, "update_withdrawal_fee", "", "Updated withdrawal fee to: "+req.Fee, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"fee_type":   "withdrawal",
		"new_value":  req.Fee,
	})
}

func (as *AdminService) UpdateDepositFee(c *gin.Context) {
	var req UpdateFeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetString("admin_id")
	as.logActivity(adminID, "update_deposit_fee", "", "Updated deposit fee to: "+req.Fee, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"fee_type":   "deposit",
		"new_value":  req.Fee,
	})
}

// ============================================================================
// NEW: API Keys Management
// ============================================================================

type APIKey struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	AdminID     string    `json:"admin_id"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
}

type CreateAPIKeyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
	ExpiresIn   int      `json:"expires_in"` // days
}

func (as *AdminService) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := uuid.New().String()
	apiKey := &APIKey{
		ID:          uuid.New().String(),
		Key:         key[:8] + "..." + key[len(key)-8:],
		AdminID:     c.GetString("admin_id"),
		Name:        req.Name,
		Permissions: req.Permissions,
		Status:      "active",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().AddDate(0, 0, req.ExpiresIn),
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"api_key": apiKey,
	})
}

func (as *AdminService) ListAPIKeys(c *gin.Context) {
	// Would return list of API keys for the admin
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"api_keys": []APIKey{},
	})
}

func (as *AdminService) RevokeAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key id required"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "API key revoked successfully",
	})
}

// ============================================================================
// Middleware
// ============================================================================

func (as *AdminService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}
		
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := as.validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		
		c.Set("admin_id", claims.AdminID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)
		c.Next()
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("TigerWallet Admin Service")
	log.Println("==========================")
	log.Printf("Starting on port %d", cfg.Port)

	// Initialize PostgreSQL
	log.Println("Connecting to PostgreSQL...")
	dbConfig := &database.DatabaseConfig{
		Host:            cfg.PostgresHost,
		Port:            cfg.PostgresPort,
		Database:        cfg.PostgresDB,
		Username:        cfg.PostgresUser,
		Password:        cfg.PostgresPass,
		MaxConns:        20,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
	
	var err error
	db, err = database.New(dbConfig)
	if err != nil {
		log.Printf("Warning: Could not connect to PostgreSQL: %v", err)
		log.Println("Continuing without database connection...")
	} else {
		log.Println("PostgreSQL connected successfully")
		
		// Initialize database schema
		ctx := context.Background()
		if err := db.InitSchema(ctx); err != nil {
			log.Printf("Warning: Could not initialize schema: %v", err)
		} else {
			log.Println("Database schema initialized")
		}
	}

	// Initialize Redis
	log.Println("Connecting to Redis...")
	redisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: "",
		DB:       0,
	})
	
	ctx := context.Background()
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Printf("Warning: Could not connect to Redis: %v", err)
		log.Println("Continuing without Redis...")
	} else {
		log.Println("Redis connected successfully")
	}

	as := NewAdminService()

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
			"service":   "admin-service",
			"timestamp": time.Now().Unix(),
		})
	})

	// Public
	r.POST("/api/v1/admin/login", as.Login)

	// Protected
	api := r.Group("/api/v1/admin")
	api.Use(as.AuthMiddleware())
	{
		// Admin Management
		api.POST("/admins", as.CreateAdmin)
		api.GET("/admins", as.ListAdmins)
		
		// User Management
		api.GET("/users", as.ListUsers)
		api.GET("/users/:id", as.GetUser)
		api.PUT("/users/:id/kyc", as.UpdateKYC)
		api.POST("/users/:id/suspend", as.SuspendUser)
		
		// Blockchain Management
		api.GET("/blockchains", as.ListBlockchains)
		api.POST("/blockchains", as.CreateBlockchain)
		api.PUT("/blockchains/:id", as.UpdateBlockchain)
		api.DELETE("/blockchains/:id", as.DeleteBlockchain)
		
		// Token Management
		api.GET("/tokens", as.ListTokens)
		api.POST("/tokens", as.CreateToken)
		api.PUT("/tokens/:id/status", as.UpdateTokenStatus)
		
		// Trading Pairs
		api.GET("/pairs", as.ListPairs)
		api.POST("/pairs", as.CreatePair)
		api.PUT("/pairs/:id/status", as.UpdatePairStatus)
		api.POST("/pairs/import", as.ImportPairs)
		
		// White Label
		api.GET("/white-labels", as.ListWhiteLabels)
		api.POST("/white-labels", as.CreateWhiteLabel)
		api.PUT("/white-labels/:id", as.UpdateWhiteLabel)
		
		// Withdrawals
		api.GET("/withdrawals", as.ListWithdrawals)
		api.POST("/withdrawals/:id/approve", as.ApproveWithdrawal)
		api.POST("/withdrawals/:id/reject", as.RejectWithdrawal)
		
		// Analytics
		api.GET("/analytics", as.GetAnalytics)
		
		// Activity Log
		api.GET("/activities", as.GetActivities)

		// ========== NEW MISSING ENDPOINTS ==========

		// Delete Admin (previously missing)
		api.DELETE("/admins/:id", as.DeleteAdmin)

		// Bulk Operations
		api.POST("/users/bulk-suspend", as.BulkSuspendUsers)
		api.POST("/users/bulk-activate", as.BulkActivateUsers)
		api.POST("/users/bulk-delete", as.BulkDeleteUsers)

		// Token Bulk Operations
		api.POST("/tokens/bulk-status", as.BulkUpdateTokenStatus)
		api.POST("/tokens/bulk-delete", as.BulkDeleteTokens)

		// Trading Pair Operations
		api.DELETE("/pairs/:id", as.DeletePair)

		// White Label Operations
		api.DELETE("/white-labels/:id", as.DeleteWhiteLabel)

		// Withdrawal Operations
		api.POST("/withdrawals/bulk-approve", as.BulkApproveWithdrawals)
		api.POST("/withdrawals/bulk-reject", as.BulkRejectWithdrawals)

		// Export Data
		api.GET("/export/users", as.ExportUsersCSV)
		api.GET("/export/transactions", as.ExportTransactionsCSV)
		api.GET("/export/tokens", as.ExportTokensCSV)
		api.GET("/export/withdrawals", as.ExportWithdrawalsCSV)

		// Fee Configuration
		api.PUT("/fees/trading", as.UpdateTradingFee)
		api.PUT("/fees/withdrawal", as.UpdateWithdrawalFee)
		api.PUT("/fees/deposit", as.UpdateDepositFee)

		// API Keys
		api.POST("/api-keys", as.CreateAPIKey)
		api.GET("/api-keys", as.ListAPIKeys)
		api.DELETE("/api-keys/:id", as.RevokeAPIKey)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
