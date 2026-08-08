package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	JWT_EXPIRATION_HOURS = 24
)

// JWT_SECRET is loaded at startup from the JWT_SECRET environment variable.
// It must never fall back to a hardcoded default.
var JWT_SECRET string

func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable must be set", key)
	}
	return value
}

// ============ Models ============

type User struct {
	ID                string    `json:"id" gorm:"primaryKey"`
	Email            string    `json:"email" gorm:"uniqueIndex;not null"`
	Username        string    `json:"username" gorm:"uniqueIndex;not null"`
	PasswordHash    string    `json:"-" gorm:"not null"`
	Role            string    `json:"role" gorm:"default:'user'"` // user, admin, super_admin, white_label_admin
	WhiteLabelID    string    `json:"whiteLabelId"`
	KYCStatus       string    `json:"kycStatus" gorm:"default:'pending'"` // pending, approved, rejected
	KYCLevel        int       `json:"kycLevel" gorm:"default:0"`
	WalletAddresses []string  `json:"walletAddresses" gorm:"type:json"`
	IsActive        bool      `json:"isActive" gorm:"default:true"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	LastLogin       time.Time `json:"lastLogin"`
}

type Wallet struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	UserID       string    `json:"userId" gorm:"index"`
	Chain        string    `json:"chain"`
	Address      string    `json:"address"`
	PrivateKey  string    `json:"-"`
	SeedEncrypted string   `json:"-"`
	IsPrimary    bool      `json:"isPrimary"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Transaction struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID     string    `json:"userId" gorm:"index"`
	WalletID   string    `json:"walletId"`
	Chain      string    `json:"chain"`
	Type      string    `json:"type"` // send, receive, swap, stake, bridge
	Token      string    `json:"token"`
	Amount     string    `json:"amount"`
	Fee        string    `json:"fee"`
	Status     string    `json:"status"` // pending, confirmed, failed
	TxHash     string    `json:"txHash"`
	FromAddr   string    `json:"fromAddr"`
	ToAddr     string    `json:"toAddr"`
	Timestamp  time.Time `json:"timestamp"`
	CreatedAt  time.Time `json:"createdAt"`
}

type WhiteLabel struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	Name            string    `json:"name" gorm:"not null"`
	Domain          string    `json:"domain" gorm:"uniqueIndex"`
	Logo            string    `json:"logo"`
	PrimaryColor    string    `json:"primaryColor"`
	SecondaryColor string    `json:"secondaryColor"`
	APIKey         string    `json:"apiKey"`
	SecretKey      string    `json:"secretKey"`
	FeePercent     float64   `json:"feePercent"` // 0-20%
	Status         string    `json:"status"` // active, paused, suspended
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Blockchain struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	Name            string    `json:"name" gorm:"not null"`
	Symbol          string    `json:"symbol" gorm:"not null"`
	ChainID         int64     `json:"chainId"`
	Type            string    `json:"type"` // evm, solana, cosmos, tron, etc
	RPCURL          string    `json:"rpcUrl"`
	ExplorerURL     string    `json:"explorerUrl"`
	ExplorerAPI    string    `json:"explorerApi"`
	IsSupported     bool      `json:"isSupported"`
	IsTestnet      bool      `json:"isTestnet"`
	Confirmations  int       `json:"confirmations"`
	MinTransfer    float64   `json:"minTransfer"`
	MaxTransfer    float64   `json:"maxTransfer"`
	GasLimit       uint64    `json:"gasLimit"`
	Explorer      string    `json:"explorer"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Token struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Chain       string    `json:"chain" gorm:"index"`
	Name        string    `json:"name" gorm:"not null"`
	Symbol      string    `json:"symbol" gorm:"not null"`
	Address     string    `json:"address"`
	Decimals    int       `json:"decimals"`
	Type        string    `json:"type"` // native, erc20, bep20, spl, trc20
	IsStableCoin bool     `json:"isStableCoin"`
	IsWrapped   bool     `json:"isWrapped"`
	LogoURL     string    `json:"logoUrl"`
	Price       float64   `json:"price"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SwapPair struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	WhiteLabelID string    `json:"whiteLabelId"`
	BaseToken    string    `json:"baseToken"`
	QuoteToken   string    `json:"quoteToken"`
	Chain        string    `json:"chain"`
	FeePercent   float64   `json:"feePercent"`
	Status      string    `json:"status"` // active, suspended, halted
	MinTrade    float64   `json:"minTrade"`
	MaxTrade    float64   `json:"maxTrade"`
	CreatedAt   time.Time `json:"createdAt"`
}

type RevenueRecord struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	WhiteLabelID string    `json:"whiteLabelId"`
	UserID       string    `json:"userId"`
	Type         string    `json:"type"` // swap, transfer, withdraw, deposit, bridge, staking
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	USDValue     float64   `json:"usdValue"`
	FeePercent   float64   `json:"feePercent"`
	FeeAmount   float64   `json:"feeAmount"`
	Chain       string    `json:"chain"`
	Token        string    `json:"token"`
	TxHash       string    `json:"txHash"`
	Timestamp    time.Time `json:"timestamp"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ============ Services ============

type Service struct {
	redis      *redis.Client
	users      map[string]*User
	wallets    map[string]*Wallet
	transactions map[string]*Transaction
	whiteLabels map[string]*WhiteLabel
	blockchains map[string]*Blockchain
	tokens     map[string]*Token
	swapPairs  map[string]*SwapPair
	revenues   map[string]*RevenueRecord
	mu        sync.RWMutex
}

func NewService() *Service {
	return &Service{
		users:        make(map[string]*User),
		wallets:      make(map[string]*Wallet),
		transactions: make(map[string]*Transaction),
		whiteLabels:  make(map[string]*WhiteLabel),
		blockchains:  make(map[string]*Blockchain),
		tokens:       make(map[string]*Token),
		swapPairs:    make(map[string]*SwapPair),
		revenues:     make(map[string]*RevenueRecord),
	}
}

func (s *Service) initData() {
	// Initialize blockchains
	blockchains := []*Blockchain{
		{ID: "ethereum", Name: "Ethereum", Symbol: "ETH", ChainID: 1, Type: "evm", RPCURL: "https://eth.llamarpc.com", ExplorerURL: "https://etherscan.io", Confirmations: 12, MinTransfer: 0.001, MaxTransfer: 1000000, GasLimit: 21000, IsSupported: true},
		{ID: "bsc", Name: "BNB Smart Chain", Symbol: "BNB", ChainID: 56, Type: "evm", RPCURL: "https://bsc-dataseed.binance.org", ExplorerURL: "https://bscscan.com", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, GasLimit: 21000, IsSupported: true},
		{ID: "polygon", Name: "Polygon", Symbol: "MATIC", ChainID: 137, Type: "evm", RPCURL: "https://polygon-rpc.com", ExplorerURL: "https://polygonscan.com", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 100000, GasLimit: 21000, IsSupported: true},
		{ID: "arbitrum", Name: "Arbitrum One", Symbol: "ETH", ChainID: 42161, Type: "evm", RPCURL: "https://arb1.arbitrum.io/rpc", ExplorerURL: "https://arbiscan.io", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, GasLimit: 21000, IsSupported: true},
		{ID: "optimism", Name: "Optimism", Symbol: "ETH", ChainID: 10, Type: "evm", RPCURL: "https://mainnet.optimism.io", ExplorerURL: "https://optimistic.etherscan.io", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, GasLimit: 21000, IsSupported: true},
		{ID: "avalanche", Name: "Avalanche C-Chain", Symbol: "AVAX", ChainID: 43114, Type: "evm", RPCURL: "https://api.avax.network/ext/bc/C/rpc", ExplorerURL: "https://snowtrace.io", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 100000, GasLimit: 21000, IsSupported: true},
		{ID: "base", Name: "Base", Symbol: "ETH", ChainID: 8453, Type: "evm", RPCURL: "https://mainnet.base.org", ExplorerURL: "https://basescan.org", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, GasLimit: 21000, IsSupported: true},
		{ID: "linea", Name: "Linea", Symbol: "ETH", ChainID: 59144, Type: "evm", RPCURL: "https://rpc.linea.build", ExplorerURL: "https://lineascan.build", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, GasLimit: 21000, IsSupported: true},
		{ID: "zksync", Name: "zkSync Era", Symbol: "ETH", ChainID: 324, Type: "evm", RPCURL: "https://zksync2-mainnet.zksync.io", ExplorerURL: "https://explorer.zksync.io", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, GasLimit: 21000, IsSupported: true},
		{ID: "solana", Name: "Solana", Symbol: "SOL", ChainID: 0, Type: "solana", RPCURL: "https://api.mainnet-beta.solana.com", ExplorerURL: "https://solscan.io", Confirmations: 32, MinTransfer: 0.001, MaxTransfer: 1000000, GasLimit: 0, IsSupported: true},
		{ID: "tron", Name: "Tron", Symbol: "TRX", ChainID: 195, Type: "tron", RPCURL: "https://api.trongrid.io", ExplorerURL: "https://tronscan.org", Confirmations: 19, MinTransfer: 1, MaxTransfer: 100000000, GasLimit: 0, IsSupported: true},
		{ID: "cosmos", Name: "Cosmos Hub", Symbol: "ATOM", ChainID: 0, Type: "cosmos", RPCURL: "https://rpc.cosmoshub4.theta-testnet.xyz:443", ExplorerURL: "https://mintscan.io/cosmos", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, GasLimit: 0, IsSupported: true},
	}
	for _, b := range blockchains { s.blockchains[b.ID] = b }

	// Initialize tokens
	tokens := []*Token{
		{ID: "eth", Chain: "ethereum", Name: "Ethereum", Symbol: "ETH", Address: "", Decimals: 18, Type: "native"},
		{ID: "usdt_eth", Chain: "ethereum", Name: "Tether USD", Symbol: "USDT", Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Decimals: 6, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_eth", Chain: "ethereum", Name: "USD Coin", Symbol: "USDC", Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Decimals: 6, Type: "erc20", IsStableCoin: true},
		{ID: "dai_eth", Chain: "ethereum", Name: "Dai Stablecoin", Symbol: "DAI", Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Decimals: 18, Type: "erc20", IsStableCoin: true},
		{ID: "bnb", Chain: "bsc", Name: "BNB", Symbol: "BNB", Address: "", Decimals: 18, Type: "native"},
		{ID: "usdt_bsc", Chain: "bsc", Name: "Tether USD", Symbol: "USDT", Address: "0x55d398326f99059ff775485246999027b3197955", Decimals: 18, Type: "bep20", IsStableCoin: true},
		{ID: "matic", Chain: "polygon", Name: "Polygon", Symbol: "MATIC", Address: "", Decimals: 18, Type: "native"},
		{ID: "sol", Chain: "solana", Name: "Solana", Symbol: "SOL", Address: "", Decimals: 9, Type: "native"},
		{ID: "trx", Chain: "tron", Name: "Tron", Symbol: "TRX", Address: "", Decimals: 6, Type: "native"},
	}
	for _, t := range tokens { s.tokens[t.ID] = t }

	// Initialize admin user
	adminPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	s.users["admin@tigerwallet.com"] = &User{
		ID:         "admin-001",
		Email:      "admin@tigerwallet.com",
		Username:   "admin",
		PasswordHash: string(adminPassword),
		Role:       "super_admin",
		KYCStatus:  "approved",
		KYCLevel:   3,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}
}

// ============ Auth ============

type AuthClaims struct {
	UserID   string `json:"userId"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func generateToken(user *User) (string, error) {
	claims := AuthClaims{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * JWT_EXPIRATION_HOURS)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWT_SECRET))
}

func verifyToken(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWT_SECRET), nil
	})
	if err != nil { return nil, err }
	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// ============ Crypto ============

func generateKeyPair() (string, string, error) {
	privateKey, err := elliptic.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { return "", "", err }
	publicKey := privateKey.PublicKey
	x := hex.EncodeToString(publicKey.X.Bytes())
	y := hex.EncodeToString(publicKey.Y.Bytes())
	privateKeyHex := hex.EncodeToString(privateKey.D.Bytes())
	return privateKeyHex, "0x" + x + y, nil
}

func generateAddress(chain string) (string, error) {
	_, pubKey, err := generateKeyPair()
	return pubKey, err
}

func encrypt(data, key string) (string, error) {
	block, err := aes.NewCipher([]byte(sha256.Sum256([]byte(key)).Bytes()))
	if err != nil { return "", err }
	cfb := cipher.NewCFBEncrypter(block, []byte(key)[:block.BlockSize()])
	encrypted := make([]byte, len(data))
	cfb.XORKeyStream(encrypted, []byte(data))
	return hex.EncodeToString(encrypted), nil
}

func decrypt(encrypted, key string) (string, error) {
	data, err := hex.DecodeString(encrypted)
	if err != nil { return "", err }
	block, err := aes.NewCipher([]byte(sha256.Sum256([]byte(key)).Bytes()))
	if err != nil { return "", err }
	cfb := cipher.NewCFBDecrypter(block, []byte(key)[:block.BlockSize()])
	decrypted := make([]byte, len(data))
	cfb.XORKeyStream(decrypted, data)
	return string(decrypted), nil
}

// ============ Handlers ============

func (s *Service) Register(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Username string `json:"username" binding:"required,min=3,max=30"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[input.Email]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user := &User{
		ID:            fmt.Sprintf("user-%d", len(s.users)+1),
		Email:         input.Email,
		Username:      input.Username,
		PasswordHash:  string(hashedPassword),
		Role:          "user",
		KYCStatus:    "pending",
		KYCLevel:      0,
		IsActive:      true,
		CreatedAt:     time.Now(),
	}
	s.users[input.Email] = user

	// Generate wallets for all supported chains
	for chainID := range s.blockchains {
		addr, _ := generateAddress(chainID)
		wallet := &Wallet{
			ID:     fmt.Sprintf("wallet-%d-%s", len(s.wallets)+1, chainID),
			UserID: user.ID,
			Chain:  chainID,
			Address: addr,
		}
		s.wallets[wallet.ID] = wallet
		user.WalletAddresses = append(user.WalletAddresses, wallet.ID)
	}

	token, _ := generateToken(user)
	c.JSON(http.StatusCreated, gin.H{"user": user, "token": token})
}

func (s *Service) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.RLock()
	user, exists := s.users[input.Email]
	s.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	user.LastLogin = time.Now()
	token, _ := generateToken(user)
	c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
}

func (s *Service) GetUser(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	s.mu.RLock()
	user, exists := s.users[auth.Email]
	s.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (s *Service) GetWallets(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	s.mu.RLock()
	var userWallets []*Wallet
	for _, w := range s.wallets {
		if w.UserID == auth.UserID {
			userWallets = append(userWallets, w)
		}
	}
	s.mu.RUnlock()

	c.JSON(http.StatusOK, userWallets)
}

func (s *Service) GetBlockchains(c *gin.Context) {
	s.mu.RLock()
	var chains []*Blockchain
	for _, b := range s.blockchains {
		chains = append(chains, b)
	}
	s.mu.RUnlock()
	c.JSON(http.StatusOK, chains)
}

func (s *Service) GetTokens(c *gin.Context) {
	chain := c.Query("chain")
	s.mu.RLock()
	var tokens []*Token
	for _, t := range s.tokens {
		if chain == "" || t.Chain == chain {
			tokens = append(tokens, t)
		}
	}
	s.mu.RUnlock()
	c.JSON(http.StatusOK, tokens)
}

func (s *Service) CreateTransaction(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input struct {
		Chain    string  `json:"chain" binding:"required"`
		Type     string  `json:"type" binding:"required"`
		Token    string  `json:"token" binding:"required"`
		Amount   string  `json:"amount" binding:"required"`
		ToAddr   string  `json:"toAddr" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := &Transaction{
		ID:         fmt.Sprintf("tx-%d", len(s.transactions)+1),
		UserID:     auth.UserID,
		Chain:     input.Chain,
		Type:      input.Type,
		Token:     input.Token,
		Amount:    input.Amount,
		Fee:       "0.001",
		Status:     "pending",
		FromAddr:  "user-wallet-address",
		ToAddr:    input.ToAddr,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.transactions[tx.ID] = tx
	s.mu.Unlock()

	// Record revenue
	s.mu.Lock()
	s.revenues[tx.ID] = &RevenueRecord{
		ID:          fmt.Sprintf("rev-%d", len(s.revenues)+1),
		UserID:      auth.UserID,
		Type:        input.Type,
		Amount:      0.001,
		FeePercent:  0.1,
		FeeAmount:   0.0001,
		Chain:       input.Chain,
		Token:       input.Token,
		Timestamp:   time.Now(),
		CreatedAt:   time.Now(),
	}
	s.mu.Unlock()

	c.JSON(http.StatusCreated, tx)
}

func (s *Service) GetTransactions(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	s.mu.RLock()
	var userTxs []*Transaction
	for _, tx := range s.transactions {
		if tx.UserID == auth.UserID {
			userTxs = append(userTxs, tx)
		}
	}
	s.mu.RUnlock()

	c.JSON(http.StatusOK, userTxs)
}

// Admin handlers
func (s *Service) AdminGetUsers(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil || (auth.Role != "admin" && auth.Role != "super_admin") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	s.mu.RLock()
	var users []*User
	for _, u := range s.users {
		users = append(users, u)
	}
	s.mu.RUnlock()

	c.JSON(http.StatusOK, users)
}

func (s *Service) AdminUpdateUser(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil || (auth.Role != "admin" && auth.Role != "super_admin") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID := c.Param("id")
	var input struct {
		KYCStatus string `json:"kycStatus"`
		Role     string `json:"role"`
		IsActive *bool  `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	if user, exists := s.users[userID]; exists {
		if input.KYCStatus != "" { user.KYCStatus = input.KYCStatus }
		if input.Role != "" && auth.Role == "super_admin" { user.Role = input.Role }
		if input.IsActive != nil { user.IsActive = *input.IsActive }
		user.UpdatedAt = time.Now()
	}
	s.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Service) AdminGetRevenue(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil || (auth.Role != "admin" && auth.Role != "super_admin") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	s.mu.RLock()
	var totalRevenue float64
	for _, r := range s.revenues {
		totalRevenue += r.FeeAmount
	}
	s.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"totalRevenue": totalRevenue,
		"transactionCount": len(s.revenues),
	})
}

func (s *Service) AdminAddBlockchain(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil || auth.Role != "super_admin" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var blockchain Blockchain
	if err := c.ShouldBindJSON(&blockchain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	blockchain.CreatedAt = time.Now()
	s.mu.Lock()
	s.blockchains[blockchain.ID] = &blockchain
	s.mu.Unlock()

	c.JSON(http.StatusCreated, &blockchain)
}

func (s *Service) AdminAddToken(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil || auth.Role != "super_admin" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var token Token
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token.CreatedAt = time.Now()
	s.mu.Lock()
	s.tokens[token.ID] = &token
	s.mu.Unlock()

	c.JSON(http.StatusCreated, &token)
}

// White Label handlers
func (s *Service) CreateWhiteLabel(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil || auth.Role != "super_admin" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input struct {
		Name            string  `json:"name" binding:"required"`
		Domain          string  `json:"domain" binding:"required"`
		FeePercent      float64 `json:"feePercent"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wl := &WhiteLabel{
		ID:              fmt.Sprintf("wl-%d", len(s.whiteLabels)+1),
		Name:            input.Name,
		Domain:          input.Domain,
		FeePercent:      input.FeePercent,
		APIKey:         "twl_" + hex.EncodeToString(randBytes(32)),
		SecretKey:       hex.EncodeToString(randBytes(64)),
		Status:          "active",
		CreatedAt:       time.Now(),
	}

	s.mu.Lock()
	s.whiteLabels[wl.ID] = wl
	s.mu.Unlock()

	c.JSON(http.StatusCreated, wl)
}

func (s *Service) GetWhiteLabels(c *gin.Context) {
	auth, err := verifyToken(c.GetHeader("Authorization"))
	if err != nil || (auth.Role != "admin" && auth.Role != "super_admin") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	s.mu.RLock()
	var wls []*WhiteLabel
	for _, wl := range s.whiteLabels {
		wls = append(wls, wl)
	}
	s.mu.RUnlock()

	c.JSON(http.StatusOK, wls)
}

// Health check
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now()})
}

// Helpers
func randBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// ============ Main ============

func main() {
	log.Println("Starting TigerWallet Backend...")

	JWT_SECRET = getRequiredEnv("JWT_SECRET")

	// Initialize service
	svc := NewService()
	svc.initData()

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// CORS middleware
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

	// Health
	router.GET("/health", healthCheck)

	// Public routes
	api := router.Group("/api/v1")
	{
		api.POST("/auth/register", svc.Register)
		api.POST("/auth/login", svc.Login)
		
		// Public data
		api.GET("/blockchains", svc.GetBlockchains)
		api.GET("/tokens", svc.GetTokens)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			c.Abort()
			return
		}
		_, err := verifyToken(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		c.Next()
	})
	{
		protected.GET("/user", svc.GetUser)
		protected.GET("/wallets", svc.GetWallets)
		protected.POST("/transactions", svc.CreateTransaction)
		protected.GET("/transactions", svc.GetTransactions)
	}

	// Admin routes
	admin := api.Group("/admin")
	admin.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			c.Abort()
			return
		}
		claims, err := verifyToken(authHeader)
		if err != nil || (claims.Role != "admin" && claims.Role != "super_admin") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}
		c.Set("claims", claims)
		c.Next()
	})
	{
		admin.GET("/users", svc.AdminGetUsers)
		admin.PUT("/users/:id", svc.AdminUpdateUser)
		admin.GET("/revenue", svc.AdminGetRevenue)
		admin.POST("/blockchains", svc.AdminAddBlockchain)
		admin.POST("/tokens", svc.AdminAddToken)
		admin.POST("/white-labels", svc.CreateWhiteLabel)
		admin.GET("/white-labels", svc.GetWhiteLabels)
	}

	// Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown")
	}
	log.Println("Server exited")
}
