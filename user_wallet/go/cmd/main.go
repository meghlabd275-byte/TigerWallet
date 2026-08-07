// UserWallet Service - Complete Implementation
// Full-featured wallet service for end users

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// Configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

// ============ DATA MODELS ============

type Wallet struct {
	ID                  uuid.UUID `json:"id"`
	UserID              uuid.UUID `json:"user_id"`
	WalletType          string    `json:"wallet_type"` // tiger, eth, bsc, etc
	Name                string    `json:"name"`
	Address             string    `json:"address"`
	PublicKey           string    `json:"public_key"`
	PrivateKeyEncrypted string    `json:"private_key_encrypted"`
	Networks            []string  `json:"networks"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type User struct {
	ID               uuid.UUID `json:"id"`
	Email            string    `json:"email"`
	Username         string    `json:"username"`
	PasswordHash     string    `json:"-"`
	Phone            string    `json:"phone"`
	KYCStatus        string    `json:"kyc_status"` // none, pending, verified, rejected
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
	TwoFactorSecret  string    `json:"-"`
	Status           string    `json:"status"` // active, suspended, deleted
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Transaction struct {
	ID            uuid.UUID `json:"id"`
	WalletID      uuid.UUID `json:"wallet_id"`
	UserID        uuid.UUID `json:"user_id"`
	Type          string    `json:"type"` // send, receive, swap, stake
	FromAddress   string    `json:"from_address"`
	ToAddress     string    `json:"to_address"`
	Amount        string    `json:"amount"`
	Token         string    `json:"token"`
	Network       string    `json:"network"`
	TxHash        string    `json:"tx_hash"`
	Status        string    `json:"status"` // pending, confirmed, failed
	Confirmations int       `json:"confirmations"`
	Fee           string    `json:"fee"`
	BlockNumber   int64     `json:"block_number"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Balance struct {
	ID        uuid.UUID `json:"id"`
	WalletID  uuid.UUID `json:"wallet_id"`
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"token"`
	Network   string    `json:"network"`
	Balance   string    `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Token struct {
	ID        uuid.UUID `json:"id"`
	Address   string    `json:"address"`
	Name      string    `json:"name"`
	Symbol    string    `json:"symbol"`
	Decimals  int       `json:"decimals"`
	Network   string    `json:"network"`
	LogoURL   string    `json:"logo_url"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type Network struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Symbol    string    `json:"symbol"`
	ChainID   int64     `json:"chain_id"`
	RPCURL    string    `json:"rpc_url"`
	Explorer  string    `json:"explorer"`
	IsTestnet bool      `json:"is_testnet"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Global variables
var (
	db        *pgxpool.Pool
	redis     *redis.Client
	config    Config
	logger    *log.Logger
	jwtSecret []byte
)

// ============ INITIALIZATION ============

func initDatabase() error {
	var err error
	dbURL := getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")

	db, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = db.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			phone VARCHAR(50),
			kyc_status VARCHAR(50) DEFAULT 'none',
			two_factor_enabled BOOLEAN DEFAULT false,
			two_factor_secret VARCHAR(255),
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS wallets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id),
			wallet_type VARCHAR(50) NOT NULL,
			name VARCHAR(255) NOT NULL,
			address VARCHAR(255) NOT NULL,
			public_key TEXT,
			private_key_encrypted TEXT,
			networks JSONB,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS transactions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID REFERENCES wallets(id),
			user_id UUID REFERENCES users(id),
			type VARCHAR(50) NOT NULL,
			from_address VARCHAR(255),
			to_address VARCHAR(255),
			amount VARCHAR(100) NOT NULL,
			token VARCHAR(50) NOT NULL,
			network VARCHAR(50) NOT NULL,
			tx_hash VARCHAR(255),
			status VARCHAR(50) DEFAULT 'pending',
			confirmations INT DEFAULT 0,
			fee VARCHAR(100),
			block_number BIGINT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS balances (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID REFERENCES wallets(id),
			user_id UUID REFERENCES users(id),
			token VARCHAR(50) NOT NULL,
			network VARCHAR(50) NOT NULL,
			balance VARCHAR(100) NOT NULL,
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(wallet_id, token, network)
		);

		CREATE TABLE IF NOT EXISTS tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			address VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			symbol VARCHAR(50) NOT NULL,
			decimals INT NOT NULL,
			network VARCHAR(50) NOT NULL,
			logo_url TEXT,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(address, network)
		);

		CREATE TABLE IF NOT EXISTS networks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			symbol VARCHAR(20) NOT NULL,
			chain_id BIGINT NOT NULL,
			rpc_url TEXT NOT NULL,
			explorer TEXT,
			is_testnet BOOLEAN DEFAULT false,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_wallets_user ON wallets(user_id);
		CREATE INDEX IF NOT EXISTS idx_transactions_user ON transactions(user_id);
		CREATE INDEX IF NOT EXISTS idx_transactions_wallet ON transactions(wallet_id);
		CREATE INDEX IF NOT EXISTS idx_balances_user ON balances(user_id);
	`)

	// Insert default networks
	_, err = db.Exec(context.Background(), `
		INSERT INTO networks (name, symbol, chain_id, rpc_url, explorer, is_testnet)
		VALUES 
			('Ethereum', 'ETH', 1, 'https://eth.llamarpc.com', 'https://etherscan.io', false),
			('Ethereum Testnet', 'ETH', 11155111, 'https://sepolia.infura.io/v3/', 'https://sepolia.etherscan.io', true),
			('BNB Smart Chain', 'BNB', 56, 'https://bsc-dataseed.binance.org', 'https://bscscan.com', false),
			('BNB Testnet', 'BNB', 97, 'https://data-seed-prebsc-1-s1.binance.org:8545', 'https://testnet.bscscan.com', true),
			('Polygon', 'MATIC', 137, 'https://polygon-rpc.com', 'https://polygonscan.com', false),
			('Arbitrum', 'ARB', 42161, 'https://arb1.arbitrum.io/rpc', 'https://arbiscan.io', false),
			('Optimism', 'OP', 10, 'https://mainnet.optimism.io', 'https://optimistic.etherscan.io', false),
			('Avalanche', 'AVAX', 43114, 'https://api.avax.network/ext/bc/C/rpc', 'https://snowtrace.io', false),
			('Solana', 'SOL', 101, 'https://api.mainnet-beta.solana.com', 'https://solscan.io', false)
		ON CONFLICT DO NOTHING
	`)

	// Insert default tokens
	_, err = db.Exec(context.Background(), `
		INSERT INTO tokens (address, name, symbol, decimals, network, logo_url)
		VALUES 
			('0x0000000000000000000000000000000000000000', 'Ethereum', 'ETH', 18, 'ethereum', 'https://assets.coingecko.com/coins/images/279/small/ethereum.png'),
			('', 'BNB', 'BNB', 18, 'bsc', 'https://assets.coingecko.com/coins/images/825/small/bnb-icon2_2x.png'),
			('0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0', 'Polygon', 'MATIC', 18, 'polygon', 'https://assets.coingecko.com/coins/images/4713/small/polygon.png'),
			('0x0d8775f648430679a709e98d2b0cb6250d2887ef', 'Bitcoin', 'BTC', 8, 'ethereum', 'https://assets.coingecko.com/coins/images/1/small/bitcoin.png'),
			('0x2260fac5e5542a773aa44fbcfedf7c193bc2c599', 'Wrapped Bitcoin', 'WBTC', 8, 'ethereum', 'https://assets.coingecko.com/coins/images/7598/small/wrapped_bitcoin_wbtc.png'),
			('0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48', 'USD Coin', 'USDC', 6, 'ethereum', 'https://assets.coingecko.com/coins/images/6319/small/USD_Coin_icon.png'),
			('0xdac17f958d2ee523a2206206994597c13d831ec7', 'Tether', 'USDT', 6, 'ethereum', 'https://assets.coingecko.com/coins/images/325/small/Tether.png')
		ON CONFLICT DO NOTHING
	`)

	return err
}

func initRedis() error {
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return err
	}
	redis = redis.NewClient(opt)
	return redis.Ping(context.Background()).Err()
}

// ============ Fetcher Functions ============

// Fetcher: Get Balance
func fetchBalance(walletID, token, network string) (string, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("balance:%s:%s:%s", walletID, token, network)
	if cached, err := redis.Get(context.Background(), cacheKey).Result(); err == nil {
		return cached, nil
	}

	// Fetch from database
	var balance string
	err := db.QueryRow(context.Background(), `
		SELECT balance FROM balances WHERE wallet_id = $1 AND token = $2 AND network = $3
	`, walletID, token, network).Scan(&balance)

	if err != nil {
		return "0", nil
	}

	// Cache for 30 seconds
	redis.Set(context.Background(), cacheKey, balance, 30*time.Second)

	return balance, nil
}

// Fetcher: Get All Balances
func fetchAllBalances(userID string) ([]Balance, error) {
	rows, err := db.Query(context.Background(), `
		SELECT id, wallet_id, user_id, token, network, balance, updated_at
		FROM balances WHERE user_id = $1 ORDER BY network, token
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		var b Balance
		if err := rows.Scan(&b.ID, &b.WalletID, &b.UserID, &b.Token, &b.Network, &b.Balance, &b.UpdatedAt); err != nil {
			continue
		}
		balances = append(balances, b)
	}

	return balances, nil
}

// Fetcher: Get Transaction History
func fetchTransactions(userID, network, token string, limit, offset int) ([]Transaction, error) {
	query := `SELECT id, wallet_id, user_id, type, from_address, to_address, amount, token, network, tx_hash, status, confirmations, fee, block_number, created_at, updated_at
		FROM transactions WHERE user_id = $1`
	args := []interface{}{userID}

	if network != "" {
		query += " AND network = $2"
		args = append(args, network)
	}
	if token != "" {
		query += " AND token = $3"
		args = append(args, token)
	}

	query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", limit+2)
	args = append(args, limit)
	args = append(args, offset)

	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.WalletID, &t.UserID, &t.Type, &t.FromAddress, &t.ToAddress, &t.Amount, &t.Token, &t.Network, &t.TxHash, &t.Status, &t.Confirmations, &t.Fee, &t.BlockNumber, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		txs = append(txs, t)
	}

	return txs, nil
}

// Fetcher: Get Token Price
func fetchTokenPrice(token, network string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("price:%s:%s", token, network)
	if cached, err := redis.Get(context.Background(), cacheKey).Result(); err == nil {
		var result map[string]interface{}
		json.Unmarshal([]byte(cached), &result)
		return result, nil
	}

	return nil, fmt.Errorf("live token price provider is not configured")
}

// Fetcher: Get Network Status
func fetchNetworkStatus(network string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("network_status:%s", network)
	if cached, err := redis.Get(context.Background(), cacheKey).Result(); err == nil {
		var result map[string]interface{}
		json.Unmarshal([]byte(cached), &result)
		return result, nil
	}

	return nil, fmt.Errorf("live network status provider is not configured")
}

// Fetcher: Get Gas Price
func fetchGasPrice(network string) (string, error) {
	cacheKey := fmt.Sprintf("gas_price:%s", network)
	if cached, err := redis.Get(context.Background(), cacheKey).Result(); err == nil {
		return cached, nil
	}

	return "", fmt.Errorf("live gas-price provider is not configured")
}

// Fetcher: Get User KYC Status
func fetchKYCStatus(userID string) (string, error) {
	var status string
	err := db.QueryRow(context.Background(), `
		SELECT kyc_status FROM users WHERE id = $1
	`, userID).Scan(&status)

	if err != nil {
		return "none", nil
	}

	return status, nil
}

// Fetcher: Get All Networks
func fetchNetworks() ([]Network, error) {
	rows, err := db.Query(context.Background(), `
		SELECT id, name, symbol, chain_id, rpc_url, explorer, is_testnet, is_active, created_at
		FROM networks WHERE is_active = true ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var networks []Network
	for rows.Next() {
		var n Network
		if err := rows.Scan(&n.ID, &n.Name, &n.Symbol, &n.ChainID, &n.RPCURL, &n.Explorer, &n.IsTestnet, &n.IsActive, &n.CreatedAt); err != nil {
			continue
		}
		networks = append(networks, n)
	}

	return networks, nil
}

// Fetcher: Get All Tokens
func fetchTokens(network string) ([]Token, error) {
	query := `SELECT id, address, name, symbol, decimals, network, logo_url, is_active, created_at
		FROM tokens WHERE is_active = true`

	var rows *pgxpool.Rows
	var err error

	if network != "" {
		query += " AND network = $1"
		rows, err = db.Query(context.Background(), query, network)
	} else {
		rows, err = db.Query(context.Background(), query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []Token
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.ID, &t.Address, &t.Name, &t.Symbol, &t.Decimals, &t.Network, &t.LogoURL, &t.IsActive, &t.CreatedAt); err != nil {
			continue
		}
		tokens = append(tokens, t)
	}

	return tokens, nil
}

// ============ HTTP HANDLERS ============

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "user-wallet"})
}

// Authentication
func register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Username string `json:"username" binding:"required,min=3,max=50"`
		Password string `json:"password" binding:"required,min=8"`
		Phone    string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email/username exists
	var exists bool
	err := db.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 OR username = $2)
	`, req.Email, req.Username).Scan(&exists)

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "email or username already exists"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure credentials"})
		return
	}

	user := User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(passwordHash),
		Phone:        req.Phone,
		KYCStatus:    "none",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = db.Exec(context.Background(), `
		INSERT INTO users (id, email, username, password_hash, phone, kyc_status, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.ID, user.Email, user.Username, user.PasswordHash, user.Phone, user.KYCStatus, user.Status, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(jwtSecret)

	c.JSON(http.StatusCreated, gin.H{"user": user, "token": tokenString})
}

func login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	err := db.QueryRow(context.Background(), `
		SELECT id, email, username, password_hash, status FROM users WHERE email = $1
	`, req.Email).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Status)

	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account not active"})
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(jwtSecret)

	c.JSON(http.StatusOK, gin.H{"token": tokenString, "user_id": user.ID})
}

// ============ FETCHER ENDPOINTS ============

// Get Balance
func getBalance(c *gin.Context) {
	walletID := c.Param("wallet_id")
	token := c.DefaultQuery("token", "ETH")
	network := c.DefaultQuery("network", "ethereum")

	balance, err := fetchBalance(walletID, token, network)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": balance, "token": token, "network": network})
}

// Get All Balances
func getAllBalances(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	balances, err := fetchAllBalances(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balances": balances})
}

// Get Transaction History
func getTransactions(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	network := c.Query("network")
	token := c.Query("token")
	limit := 50
	offset := 0

	txs, err := fetchTransactions(userID, network, token, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

// Get Token Price
func getTokenPrice(c *gin.Context) {
	token := c.Param("token")
	network := c.DefaultQuery("network", "ethereum")

	price, err := fetchTokenPrice(token, network)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, price)
}

// Get Network Status
func getNetworkStatus(c *gin.Context) {
	network := c.Param("network")

	status, err := fetchNetworkStatus(network)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// Get Gas Price
func getGasPrice(c *gin.Context) {
	network := c.Param("network")

	gasPrice, err := fetchGasPrice(network)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"network": network, "gas_price": gasPrice})
}

// Get KYC Status
func getKYCStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	status, err := fetchKYCStatus(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"kyc_status": status})
}

// Get Networks
func getNetworks(c *gin.Context) {
	networks, err := fetchNetworks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"networks": networks})
}

// Get Tokens
func getTokens(c *gin.Context) {
	network := c.Query("network")

	tokens, err := fetchTokens(network)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// Create Transaction
func createTransaction(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		WalletID  uuid.UUID `json:"wallet_id" binding:"required"`
		Type      string    `json:"type" binding:"required"`
		ToAddress string    `json:"to_address" binding:"required"`
		Amount    string    `json:"amount" binding:"required"`
		Token     string    `json:"token" binding:"required"`
		Network   string    `json:"network" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, _ := uuid.Parse(userID)

	tx := Transaction{
		ID:            uuid.New(),
		WalletID:      req.WalletID,
		UserID:        uid,
		Type:          req.Type,
		ToAddress:     req.ToAddress,
		Amount:        req.Amount,
		Token:         req.Token,
		Network:       req.Network,
		Status:        "pending",
		Confirmations: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err := db.Exec(context.Background(), `
		INSERT INTO transactions (id, wallet_id, user_id, type, to_address, amount, token, network, status, confirmations, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, tx.ID, tx.WalletID, tx.UserID, tx.Type, tx.ToAddress, tx.Amount, tx.Token, tx.Network, tx.Status, tx.Confirmations, tx.CreatedAt, tx.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tx)
}

// Create Wallet
func createWallet(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Name       string   `json:"name" binding:"required"`
		WalletType string   `json:"wallet_type" binding:"required"`
		Networks   []string `json:"networks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, _ := uuid.Parse(userID)

	// Generate mock address
	address := generateAddress(req.WalletType)

	wallet := Wallet{
		ID:         uuid.New(),
		UserID:     uid,
		WalletType: req.WalletType,
		Name:       req.Name,
		Address:    address,
		Networks:   req.Networks,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err := db.Exec(context.Background(), `
		INSERT INTO wallets (id, user_id, wallet_type, name, address, networks, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, wallet.ID, wallet.UserID, wallet.WalletType, wallet.Name, wallet.Address, wallet.Networks, wallet.IsActive, wallet.CreatedAt, wallet.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wallet)
}

// Get User Wallets
func getWallets(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	uid, _ := uuid.Parse(userID)

	rows, err := db.Query(context.Background(), `
		SELECT id, user_id, wallet_type, name, address, networks, is_active, created_at, updated_at
		FROM wallets WHERE user_id = $1 AND is_active = true
	`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var wallets []Wallet
	for rows.Next() {
		var w Wallet
		if err := rows.Scan(&w.ID, &w.UserID, &w.WalletType, &w.Name, &w.Address, &w.Networks, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			continue
		}
		wallets = append(wallets, w)
	}

	c.JSON(http.StatusOK, gin.H{"wallets": wallets})
}

func generateAddress(walletType string) string {
	// In production, generate actual crypto addresses
	types := map[string]string{
		"ethereum": "0x",
		"bsc":      "0x",
		"solana":   "",
	}

	prefix := types[walletType]
	if prefix == "" {
		prefix = "0x"
	}

	return prefix + strings.ToUpper(hex.EncodeToString([]byte(uuid.New().String()[:20])))[:40]
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Middleware: JWT Auth
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"].(string))
		c.Next()
	}
}

// ============ MAIN ============

func main() {
	logger = log.New(os.Stdout, "UserWallet: ", log.LstdFlags)
	logger.Println("Starting UserWallet Service...")

	config.Port = getEnv("USER_WALLET_PORT", "8105")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")
	config.JWTSecret = getEnv("JWT_SECRET", "tigerwallet-userwallet-secret")
	jwtSecret = []byte(config.JWTSecret)

	if err := initDatabase(); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	logger.Println("Database connected")

	if err := initRedis(); err != nil {
		logger.Fatalf("Failed to initialize Redis: %v", err)
	}
	logger.Println("Redis connected")

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/health", healthCheck)

	// Auth
	router.POST("/api/v1/auth/register", register)
	router.POST("/api/v1/auth/login", login)

	// Protected routes
	api := router.Group("/api/v1")
	api.Use(authMiddleware())
	{
		// Wallets
		api.POST("/wallets", createWallet)
		api.GET("/wallets", getWallets)

		// Transactions
		api.POST("/transactions", createTransaction)
		api.GET("/transactions", getTransactions)

		// Fetchers
		api.GET("/balances", getAllBalances)
		api.GET("/balances/:wallet_id", getBalance)
		api.GET("/prices/:token", getTokenPrice)
		api.GET("/networks", getNetworks)
		api.GET("/network/:network/status", getNetworkStatus)
		api.GET("/network/:network/gas", getGasPrice)
		api.GET("/tokens", getTokens)
		api.GET("/kyc/status", getKYCStatus)
	}

	logger.Printf("Starting server on port %s", config.Port)
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Println("Server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	db.Close()
	redis.Close()
	logger.Println("Server exited")
}
