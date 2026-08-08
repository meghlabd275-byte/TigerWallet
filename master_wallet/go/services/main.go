package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/http2"
)

const (
	// Database
	DB_MAX_CONNS         = 100
	DB_MIN_CONNS         = 10
	DB_MAX_CONN_LIFETIME = 30 // minutes

	// Redis
	REDIS_POOL_SIZE = 50

	// JWT
	JWT_EXPIRY = 24 * time.Hour

	// Server
	GRACEFUL_TIMEOUT = 10 * time.Second
	MAX_HEADER_BYTES = 1 << 20 // 1MB
)

var (
	globalConnPool *pgxpool.Pool
	redisClient   *redis.Client
	upgrader      = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // In production, configure properly
		},
	}
)

// ==================== CONFIGURATION ====================

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig   `json:"redis"`
	Security SecurityConfig `json:"security"`
}

type ServerConfig struct {
	Port            int      `json:"port"`
	Mode            string   `json:"mode"`
	ReadTimeout     int      `json:"read_timeout"`
	WriteTimeout    int      `json:"write_timeout"`
	MaxHeaderBytes int      `json:"max_header_bytes"`
	EnableTLS      bool     `json:"enable_tls"`
	TLSCertFile    string   `json:"tls_cert_file"`
	TLSKeyFile     string   `json:"tls_key_file"`
}

type DatabaseConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Database        string `json:"database"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	MaxConns        int32  `json:"max_conns"`
	MinConns        int32  `json:"min_conns"`
	MaxConnLifetime int    `json:"max_conn_lifetime"`
	SSLMode         string `json:"ssl_mode"`
}

type RedisConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Password     string `json:"password"`
	DB           int    `json:"db"`
	PoolSize     int    `json:"pool_size"`
	MinIdleConns int    `json:"min_idle_conns"`
}

type SecurityConfig struct {
	JWTSecret          string `json:"jwt_secret"`
	Argon2Time        int    `json:"argon2_time"`
	Argon2Memory       int    `json:"argon2_memory"`
	Argon2Threads      int    `json:"argon2_threads"`
	Argon2KeyLen       int    `json:"argon2_key_len"`
	BcryptCost         int    `json:"bcrypt_cost"`
	MaxLoginAttempts   int    `json:"max_login_attempts"`
	LockoutDuration    int    `json:"lockout_duration"`
}

// ==================== MODELS ====================

type MasterWallet struct {
	ID              string    `json:"id"`
	Name           string    `json:"name"`
	WalletType     string    `json:"wallet_type"`
	Address        string    `json:"address"`
	PublicKey      string    `json:"public_key"`
	EncryptedSeed string    `json:"encrypted_seed"`
	ChainID        int64     `json:"chain_id"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SubWallet struct {
	ID                string    `json:"id"`
	MasterWalletID   string    `json:"master_wallet_id"`
	Name             string    `json:"name"`
	Address          string    `json:"address"`
	AddressType      string    `json:"address_type"`
	PublicKey        string    `json:"public_key"`
	EncryptedKey     string    `json:"encrypted_key"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type User struct {
	ID                string    `json:"id"`
	Email            string    `json:"email"`
	Name             string    `json:"name"`
	Role             string    `json:"role"`
	MasterWalletID   string    `json:"master_wallet_id"`
	SubWalletID      string    `json:"sub_wallet_id"`
	Permissions      []string  `json:"permissions"`
	PasswordHash     string    `json:"-"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
	TwoFactorSecret  string    `json:"-"`
	IsActive         bool      `json:"is_active"`
	FailedAttempts   int       `json:"failed_attempts"`
	LockedUntil      *time.Time `json:"locked_until,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	LastLoginAt      *time.Time `json:"last_login_at"`
}

type Transaction struct {
	ID              string    `json:"id"`
	MasterWalletID string    `json:"master_wallet_id"`
	SubWalletID    string    `json:"sub_wallet_id"`
	Hash           string    `json:"hash"`
	From           string    `json:"from"`
	To             string    `json:"to"`
	Amount         string    `json:"amount"`
	Token          string    `json:"token"`
	ChainID        int64     `json:"chain_id"`
	Status         string    `json:"status"`
	Fee            string    `json:"fee"`
	BlockNumber    int64     `json:"block_number"`
	CreatedAt      time.Time `json:"created_at"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
}

type AutoSignRule struct {
	ID            string    `json:"id"`
	MasterWalletID string   `json:"master_wallet_id"`
	Name          string    `json:"name"`
	MaxAmount     string    `json:"max_amount"`
	ChainIDs      []string  `json:"chain_ids"`
	TokenIDs      []string  `json:"token_ids"`
	Enabled       bool      `json:"enabled"`
	Conditions    []string  `json:"conditions"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Action      string    `json:"action"`
	EntityType  string    `json:"entity_type"`
	EntityID    string    `json:"entity_id"`
	Details     string    `json:"details"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`
}

type FeeConfig struct {
	ID              string    `json:"id"`
	MasterWalletID string    `json:"master_wallet_id"`
	Name            string    `json:"name"`
	FeeType         string    `json:"fee_type"` // withdrawal, swap, transaction
	Percentage      float64   `json:"percentage"`
	FlatFee         string    `json:"flat_fee"`
	MinAmount       string    `json:"min_amount"`
	MaxAmount       string    `json:"max_amount"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Policy struct {
	ID              string          `json:"id"`
	MasterWalletID  string          `json:"master_wallet_id"`
	Name            string          `json:"name"`
	PolicyType      string          `json:"policy_type"`
	Rules           json.RawMessage `json:"rules"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ==================== SERVICES ====================

type MasterWalletService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewMasterWalletService(db *pgxpool.Pool, redis *redis.Client) *MasterWalletService {
	return &MasterWalletService{db: db, redis: redis}
}

// ==================== DATABASE OPERATIONS ====================

func initDatabase(cfg DatabaseConfig) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	config.MaxConns = cfg.MaxConns
	config.MinConns = cfg.MinConns
	config.MaxConnLifetime = time.Duration(cfg.MaxConnLifetime) * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func initRedis(cfg RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}

// ==================== ROUTES ====================

func setupRoutes(r *gin.Engine, svc *MasterWalletService, cfg SecurityConfig) {
	// Middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MAX_HEADER_BYTES)
		c.Next()
	})
	r.Use(corsMiddleware())

	// Health check
	r.GET("/health", healthCheck)

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", svc.Register)
			auth.POST("/login", svc.Login)
			auth.POST("/logout", svc.Logout)
			auth.POST("/refresh", svc.RefreshToken)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(JWTAuthMiddleware(cfg.JWTSecret))
		{
			// Master Wallet routes
			mw := protected.Group("/master-wallet")
			{
				mw.GET("", svc.GetMasterWallets)
				mw.POST("", svc.CreateMasterWallet)
				mw.GET("/:id", svc.GetMasterWallet)
				mw.PUT("/:id", svc.UpdateMasterWallet)
				mw.DELETE("/:id", svc.DeleteMasterWallet)
				mw.GET("/:id/balance", svc.GetMasterWalletBalance)
				mw.POST("/:id/sign", svc.SignTransaction)
			}

			// Sub Wallet routes
			sw := protected.Group("/sub-wallet")
			{
				sw.GET("", svc.GetSubWallets)
				sw.POST("", svc.CreateSubWallet)
				sw.GET("/:id", svc.GetSubWallet)
				sw.PUT("/:id", svc.UpdateSubWallet)
				sw.DELETE("/:id", svc.DeleteSubWallet)
				sw.GET("/:id/balance", svc.GetSubWalletBalance)
				sw.POST("/:id/transfer", svc.TransferFromSubWallet)
			}

			// Transaction routes
			tx := protected.Group("/transactions")
			{
				tx.GET("", svc.GetTransactions)
				tx.GET("/:id", svc.GetTransaction)
				tx.POST("", svc.CreateTransaction)
				tx.POST("/:id/approve", svc.ApproveTransaction)
				tx.POST("/:id/reject", svc.RejectTransaction)
			}

			// Auto-sign routes
			autoSign := protected.Group("/auto-sign")
			{
				autoSign.GET("", svc.GetAutoSignRules)
				autoSign.POST("", svc.CreateAutoSignRule)
				autoSign.PUT("/:id", svc.UpdateAutoSignRule)
				autoSign.DELETE("/:id", svc.DeleteAutoSignRule)
			}

			// Fee config routes
			fees := protected.Group("/fees")
			{
				fees.GET("", svc.GetFeeConfigs)
				fees.POST("", svc.CreateFeeConfig)
				fees.PUT("/:id", svc.UpdateFeeConfig)
				fees.DELETE("/:id", svc.DeleteFeeConfig)
			}

			// Policy routes
			policies := protected.Group("/policies")
			{
				policies.GET("", svc.GetPolicies)
				policies.POST("", svc.CreatePolicy)
				policies.PUT("/:id", svc.UpdatePolicy)
				policies.DELETE("/:id", svc.DeletePolicy)
			}

			// User management routes
			users := protected.Group("/users")
			{
				users.GET("", svc.GetUsers)
				users.POST("", svc.CreateUser)
				users.PUT("/:id", svc.UpdateUser)
				users.DELETE("/:id", svc.DeleteUser)
				users.POST("/:id/reset-password", svc.ResetPassword)
			}

			// Analytics routes
			analytics := protected.Group("/analytics")
			{
				analytics.GET("/volume", svc.GetVolumeAnalytics)
				analytics.GET("/transactions", svc.GetTransactionAnalytics)
				analytics.GET("/wallets", svc.GetWalletAnalytics)
			}

			// Audit log routes
			audit := protected.Group("/audit")
			{
				audit.GET("", svc.GetAuditLogs)
			}
		}
	}

	// WebSocket
	r.GET("/ws", svc.HandleWebSocket)
}

// ==================== HANDLERS ====================

func (s *MasterWalletService) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Name     string `json:"name" binding:"required"`
		Role     string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := User{
		ID:              uuid.New().String(),
		Email:           req.Email,
		Name:            req.Name,
		Role:            req.Role,
		PasswordHash:    string(hashedPassword),
		IsActive:        true,
		FailedAttempts:  0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err = s.db.Exec(context.Background(),
		`INSERT INTO users (id, email, name, role, password_hash, is_active, failed_attempts, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		user.ID, user.Email, user.Name, user.Role, user.PasswordHash, user.IsActive, user.FailedAttempts, user.CreatedAt, user.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "user_id": user.ID})
}

func (s *MasterWalletService) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	err := s.db.QueryRow(context.Background(),
		`SELECT id, email, name, role, password_hash, failed_attempts, locked_until, is_active FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.PasswordHash, &user.FailedAttempts, &user.LockedUntil, &user.IsActive)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "Account locked. Try again later."})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Increment failed attempts
		s.db.Exec(context.Background(),
			"UPDATE users SET failed_attempts = failed_attempts + 1, locked_until = CASE WHEN failed_attempts >= 4 THEN NOW() + INTERVAL '15 minutes' ELSE NULL END WHERE id = $1",
			user.ID,
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := generateJWT(user.ID, user.Email, user.Role, "tigerwallet-secret-key")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Update last login
	s.db.Exec(context.Background(),
		"UPDATE users SET last_login_at = NOW(), failed_attempts = 0 WHERE id = $1",
		user.ID,
	)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

func (s *MasterWalletService) Logout(c *gin.Context) {
	// In production, would invalidate JWT token in Redis
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (s *MasterWalletService) RefreshToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify and refresh token
	claims, err := verifyJWT(req.Token, "tigerwallet-secret-key")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	newToken, err := generateJWT(claims["user_id"].(string), claims["email"].(string), claims["role"].(string), "tigerwallet-secret-key")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": newToken})
}

// Master Wallet Handlers
func (s *MasterWalletService) GetMasterWallets(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := s.db.Query(context.Background(),
		`SELECT id, name, wallet_type, address, public_key, chain_id, is_active, created_at, updated_at 
		 FROM master_wallets WHERE created_by = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallets"})
		return
	}
	defer rows.Close()

	var wallets []MasterWallet
	for rows.Next() {
		var w MasterWallet
		if err := rows.Scan(&w.ID, &w.Name, &w.WalletType, &w.Address, &w.PublicKey, &w.ChainID, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			continue
		}
		wallets = append(wallets, w)
	}

	c.JSON(http.StatusOK, gin.H{"wallets": wallets})
}

func (s *MasterWalletService) CreateMasterWallet(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		WalletType string `json:"wallet_type"`
		ChainID    int64  `json:"chain_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")

	// Generate wallet
	privateKey, err := generatePrivateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate key"})
		return
	}

	publicKey := derivePublicKey(privateKey)
	address := deriveAddress(publicKey)

	wallet := MasterWallet{
		ID:          uuid.New().String(),
		Name:        req.Name,
		WalletType:  req.WalletType,
		Address:     address,
		PublicKey:   hex.EncodeToString(publicKey),
		ChainID:     req.ChainID,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = s.db.Exec(context.Background(),
		`INSERT INTO master_wallets (id, name, wallet_type, address, public_key, chain_id, is_active, created_at, updated_at, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		wallet.ID, wallet.Name, wallet.WalletType, wallet.Address, wallet.PublicKey, wallet.ChainID, wallet.IsActive, wallet.CreatedAt, wallet.UpdatedAt, userID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wallet"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"wallet": wallet})
}

func (s *MasterWalletService) GetMasterWallet(c *gin.Context) {
	id := c.Param("id")

	var wallet MasterWallet
	err := s.db.QueryRow(context.Background(),
		`SELECT id, name, wallet_type, address, public_key, chain_id, is_active, created_at, updated_at 
		 FROM master_wallets WHERE id = $1`,
		id,
	).Scan(&wallet.ID, &wallet.Name, &wallet.WalletType, &wallet.Address, &wallet.PublicKey, &wallet.ChainID, &wallet.IsActive, &wallet.CreatedAt, &wallet.UpdatedAt)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet": wallet})
}

func (s *MasterWalletService) UpdateMasterWallet(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name      string `json:"name"`
		IsActive  bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := s.db.Exec(context.Background(),
		"UPDATE master_wallets SET name = COALESCE(NULLIF($1, ''), name), is_active = $2, updated_at = NOW() WHERE id = $3",
		req.Name, req.IsActive, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wallet updated"})
}

func (s *MasterWalletService) DeleteMasterWallet(c *gin.Context) {
	id := c.Param("id")

	_, err := s.db.Exec(context.Background(), "DELETE FROM master_wallets WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wallet deleted"})
}

func (s *MasterWalletService) GetMasterWalletBalance(c *gin.Context) {
	id := c.Param("id")

	// In production, would query blockchain for actual balance
	c.JSON(http.StatusOK, gin.H{
		"wallet_id": id,
		"balance":   "0",
		"tokens":    []string{},
	})
}

func (s *MasterWalletService) SignTransaction(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		To     string `json:"to" binding:"required"`
		Amount string `json:"amount" binding:"required"`
		Token  string `json:"token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve the master wallet's chain and address so we broadcast a real
	// transaction through the chain's RPC node instead of fabricating a hash.
	ctx := context.Background()
	var (
		fromAddr string
		chainID  int64
	)
	err := s.db.QueryRow(ctx,
		`SELECT address, chain_id FROM master_wallets WHERE id = $1`,
		id,
	).Scan(&fromAddr, &chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Master wallet not found"})
		return
	}

	chain := chainNameFromID(chainID)
	txHash, err := broadcastTransactionByChain(chain, fromAddr, req.To, req.Amount, req.Token)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to broadcast transaction", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_hash": txHash,
		"status":           "broadcast",
	})
}

// Sub Wallet Handlers
func (s *MasterWalletService) GetSubWallets(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")

	rows, err := s.db.Query(context.Background(),
		`SELECT id, master_wallet_id, name, address, address_type, is_active, created_at, updated_at 
		 FROM sub_wallets WHERE master_wallet_id = $1 ORDER BY created_at DESC`,
		masterWalletID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallets"})
		return
	}
	defer rows.Close()

	var wallets []SubWallet
	for rows.Next() {
		var w SubWallet
		if err := rows.Scan(&w.ID, &w.MasterWalletID, &w.Name, &w.Address, &w.AddressType, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			continue
		}
		wallets = append(wallets, w)
	}

	c.JSON(http.StatusOK, gin.H{"wallets": wallets})
}

func (s *MasterWalletService) CreateSubWallet(c *gin.Context) {
	var req struct {
		MasterWalletID string `json:"master_wallet_id" binding:"required"`
		Name           string `json:"name" binding:"required"`
		AddressType    string `json:"address_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate sub-wallet
	privateKey, _ := generatePrivateKey()
	publicKey := derivePublicKey(privateKey)
	address := deriveAddress(publicKey)

	wallet := SubWallet{
		ID:              uuid.New().String(),
		MasterWalletID:  req.MasterWalletID,
		Name:            req.Name,
		Address:         address,
		AddressType:     req.AddressType,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err := s.db.Exec(context.Background(),
		`INSERT INTO sub_wallets (id, master_wallet_id, name, address, address_type, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		wallet.ID, wallet.MasterWalletID, wallet.Name, wallet.Address, wallet.AddressType, wallet.IsActive, wallet.CreatedAt, wallet.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wallet"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"wallet": wallet})
}

func (s *MasterWalletService) GetSubWallet(c *gin.Context) {
	id := c.Param("id")

	var wallet SubWallet
	err := s.db.QueryRow(context.Background(),
		`SELECT id, master_wallet_id, name, address, address_type, is_active, created_at, updated_at 
		 FROM sub_wallets WHERE id = $1`,
		id,
	).Scan(&wallet.ID, &wallet.MasterWalletID, &wallet.Name, &wallet.Address, &wallet.AddressType, &wallet.IsActive, &wallet.CreatedAt, &wallet.UpdatedAt)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet": wallet})
}

func (s *MasterWalletService) UpdateSubWallet(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name     string `json:"name"`
		IsActive bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := s.db.Exec(context.Background(),
		"UPDATE sub_wallets SET name = COALESCE(NULLIF($1, ''), name), is_active = $2, updated_at = NOW() WHERE id = $3",
		req.Name, req.IsActive, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wallet updated"})
}

func (s *MasterWalletService) DeleteSubWallet(c *gin.Context) {
	id := c.Param("id")

	_, err := s.db.Exec(context.Background(), "DELETE FROM sub_wallets WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wallet deleted"})
}

func (s *MasterWalletService) GetSubWalletBalance(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusOK, gin.H{
		"wallet_id": id,
		"balance":   "0",
	})
}

func (s *MasterWalletService) TransferFromSubWallet(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		To     string `json:"to" binding:"required"`
		Amount string `json:"amount" binding:"required"`
		Token  string `json:"token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve the sub wallet's address and its master wallet's chain so we
	// broadcast a real transaction instead of fabricating a hash.
	ctx := context.Background()
	var (
		fromAddr     string
		masterWalletID string
	)
	err := s.db.QueryRow(ctx,
		`SELECT address, master_wallet_id FROM sub_wallets WHERE id = $1`,
		id,
	).Scan(&fromAddr, &masterWalletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sub wallet not found"})
		return
	}

	var chainID int64
	err = s.db.QueryRow(ctx,
		`SELECT chain_id FROM master_wallets WHERE id = $1`,
		masterWalletID,
	).Scan(&chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Master wallet not found"})
		return
	}

	chain := chainNameFromID(chainID)
	txHash, err := broadcastTransactionByChain(chain, fromAddr, req.To, req.Amount, req.Token)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to broadcast transaction", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_hash": txHash,
		"status":           "pending",
	})
}

// Transaction Handlers
func (s *MasterWalletService) GetTransactions(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")
	status := c.Query("status")
	ctx := context.Background()

	query := `SELECT id, master_wallet_id, sub_wallet_id, hash, from_address, to_address, amount, token, chain_id, status, fee, block_number, created_at, confirmed_at
	          FROM transactions WHERE master_wallet_id = $1`
	args := []interface{}{masterWalletID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}
	defer rows.Close()

	txs := []Transaction{} // non-nil so the JSON response is [] not null
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.MasterWalletID, &tx.SubWalletID, &tx.Hash, &tx.From, &tx.To, &tx.Amount, &tx.Token, &tx.ChainID, &tx.Status, &tx.Fee, &tx.BlockNumber, &tx.CreatedAt, &tx.ConfirmedAt); err != nil {
			continue
		}
		txs = append(txs, tx)
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

func (s *MasterWalletService) GetTransaction(c *gin.Context) {
	id := c.Param("id")

	var tx Transaction
	err := s.db.QueryRow(context.Background(),
		`SELECT id, master_wallet_id, sub_wallet_id, hash, from_address, to_address, amount, token, chain_id, status, fee, block_number, created_at, confirmed_at 
		 FROM transactions WHERE id = $1`,
		id,
	).Scan(&tx.ID, &tx.MasterWalletID, &tx.SubWalletID, &tx.Hash, &tx.From, &tx.To, &tx.Amount, &tx.Token, &tx.ChainID, &tx.Status, &tx.Fee, &tx.BlockNumber, &tx.CreatedAt, &tx.ConfirmedAt)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction": tx})
}

func (s *MasterWalletService) CreateTransaction(c *gin.Context) {
	var req struct {
		MasterWalletID string `json:"master_wallet_id" binding:"required"`
		SubWalletID     string `json:"sub_wallet_id"`
		To             string `json:"to" binding:"required"`
		Amount         string `json:"amount" binding:"required"`
		Token          string `json:"token"`
		ChainID        int64  `json:"chain_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Resolve the sending address. If a sub wallet is supplied, prefer it;
	// otherwise fall back to the master wallet's address.
	var (
		fromAddr string
		chainID  int64 = req.ChainID
	)
	if req.SubWalletID != "" {
		var masterWalletID string
		if err := s.db.QueryRow(ctx,
			`SELECT address, master_wallet_id FROM sub_wallets WHERE id = $1`,
			req.SubWalletID,
		).Scan(&fromAddr, &masterWalletID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sub wallet not found"})
			return
		}
		if req.ChainID == 0 {
			if err := s.db.QueryRow(ctx,
				`SELECT chain_id FROM master_wallets WHERE id = $1`,
				masterWalletID,
			).Scan(&chainID); err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Master wallet not found"})
				return
			}
		}
	} else {
		if err := s.db.QueryRow(ctx,
			`SELECT address, chain_id FROM master_wallets WHERE id = $1`,
			req.MasterWalletID,
		).Scan(&fromAddr, &chainID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Master wallet not found"})
			return
		}
	}

	// Broadcast the real transaction through the chain's RPC node. We do not
	// persist a fabricated hash; the stored hash comes from the RPC response.
	txHash, err := broadcastTransactionByChain(chainNameFromID(chainID), fromAddr, req.To, req.Amount, req.Token)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to broadcast transaction", "detail": err.Error()})
		return
	}

	tx := Transaction{
		ID:             uuid.New().String(),
		MasterWalletID: req.MasterWalletID,
		SubWalletID:    req.SubWalletID,
		Hash:           txHash,
		From:           fromAddr,
		To:             req.To,
		Amount:         req.Amount,
		Token:          req.Token,
		ChainID:        chainID,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO transactions (id, master_wallet_id, sub_wallet_id, hash, from_address, to_address, amount, token, chain_id, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		tx.ID, tx.MasterWalletID, tx.SubWalletID, tx.Hash, tx.From, tx.To, tx.Amount, tx.Token, tx.ChainID, tx.Status, tx.CreatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"transaction": tx})
}

func (s *MasterWalletService) ApproveTransaction(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	_, err := s.db.Exec(context.Background(),
		"UPDATE transactions SET status = 'approved', approved_by = $1, updated_at = NOW() WHERE id = $2",
		userID, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction approved"})
}

func (s *MasterWalletService) RejectTransaction(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	_, err := s.db.Exec(context.Background(),
		"UPDATE transactions SET status = 'rejected', rejected_by = $1, reject_reason = $2, updated_at = NOW() WHERE id = $3",
		userID, req.Reason, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction rejected"})
}

// Auto-Sign Rule Handlers
func (s *MasterWalletService) GetAutoSignRules(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")

	rows, err := s.db.Query(context.Background(),
		`SELECT id, master_wallet_id, name, max_amount, chain_ids, token_ids, enabled, conditions, created_at, updated_at 
		 FROM auto_sign_rules WHERE master_wallet_id = $1`,
		masterWalletID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rules"})
		return
	}
	defer rows.Close()

	var rules []AutoSignRule
	for rows.Next() {
		var r AutoSignRule
		if err := rows.Scan(&r.ID, &r.MasterWalletID, &r.Name, &r.MaxAmount, &r.ChainIDs, &r.TokenIDs, &r.Enabled, &r.Conditions, &r.CreatedAt, &r.UpdatedAt); err != nil {
			continue
		}
		rules = append(rules, r)
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (s *MasterWalletService) CreateAutoSignRule(c *gin.Context) {
	var req AutoSignRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID = uuid.New().String()
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	_, err := s.db.Exec(context.Background(),
		`INSERT INTO auto_sign_rules (id, master_wallet_id, name, max_amount, chain_ids, token_ids, enabled, conditions, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		req.ID, req.MasterWalletID, req.Name, req.MaxAmount, req.ChainIDs, req.TokenIDs, req.Enabled, req.Conditions, req.CreatedAt, req.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"rule": req})
}

func (s *MasterWalletService) UpdateAutoSignRule(c *gin.Context) {
	id := c.Param("id")

	var req AutoSignRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := s.db.Exec(context.Background(),
		`UPDATE auto_sign_rules SET name = $1, max_amount = $2, chain_ids = $3, token_ids = $4, enabled = $5, conditions = $6, updated_at = NOW() WHERE id = $7`,
		req.Name, req.MaxAmount, req.ChainIDs, req.TokenIDs, req.Enabled, req.Conditions, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule updated"})
}

func (s *MasterWalletService) DeleteAutoSignRule(c *gin.Context) {
	id := c.Param("id")

	_, err := s.db.Exec(context.Background(), "DELETE FROM auto_sign_rules WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted"})
}

// Fee Config Handlers
func (s *MasterWalletService) GetFeeConfigs(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")

	rows, err := s.db.Query(context.Background(),
		`SELECT id, master_wallet_id, name, fee_type, percentage, flat_fee, min_amount, max_amount, is_active, created_at, updated_at 
		 FROM fee_configs WHERE master_wallet_id = $1`,
		masterWalletID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch fee configs"})
		return
	}
	defer rows.Close()

	var fees []FeeConfig
	for rows.Next() {
		var f FeeConfig
		if err := rows.Scan(&f.ID, &f.MasterWalletID, &f.Name, &f.FeeType, &f.Percentage, &f.FlatFee, &f.MinAmount, &f.MaxAmount, &f.IsActive, &f.CreatedAt, &f.UpdatedAt); err != nil {
			continue
		}
		fees = append(fees, f)
	}

	c.JSON(http.StatusOK, gin.H{"fees": fees})
}

func (s *MasterWalletService) CreateFeeConfig(c *gin.Context) {
	var req FeeConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID = uuid.New().String()
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	_, err := s.db.Exec(context.Background(),
		`INSERT INTO fee_configs (id, master_wallet_id, name, fee_type, percentage, flat_fee, min_amount, max_amount, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		req.ID, req.MasterWalletID, req.Name, req.FeeType, req.Percentage, req.FlatFee, req.MinAmount, req.MaxAmount, req.IsActive, req.CreatedAt, req.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create fee config"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"fee": req})
}

func (s *MasterWalletService) UpdateFeeConfig(c *gin.Context) {
	id := c.Param("id")

	var req FeeConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := s.db.Exec(context.Background(),
		`UPDATE fee_configs SET name = $1, fee_type = $2, percentage = $3, flat_fee = $4, min_amount = $5, max_amount = $6, is_active = $7, updated_at = NOW() WHERE id = $8`,
		req.Name, req.FeeType, req.Percentage, req.FlatFee, req.MinAmount, req.MaxAmount, req.IsActive, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update fee config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fee config updated"})
}

func (s *MasterWalletService) DeleteFeeConfig(c *gin.Context) {
	id := c.Param("id")

	_, err := s.db.Exec(context.Background(), "DELETE FROM fee_configs WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete fee config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fee config deleted"})
}

// Policy Handlers
func (s *MasterWalletService) GetPolicies(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")

	rows, err := s.db.Query(context.Background(),
		`SELECT id, master_wallet_id, name, policy_type, rules, is_active, created_at, updated_at 
		 FROM policies WHERE master_wallet_id = $1`,
		masterWalletID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch policies"})
		return
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var p Policy
		if err := rows.Scan(&p.ID, &p.MasterWalletID, &p.Name, &p.PolicyType, &p.Rules, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		policies = append(policies, p)
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (s *MasterWalletService) CreatePolicy(c *gin.Context) {
	var req Policy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID = uuid.New().String()
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	_, err := s.db.Exec(context.Background(),
		`INSERT INTO policies (id, master_wallet_id, name, policy_type, rules, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.ID, req.MasterWalletID, req.Name, req.PolicyType, req.Rules, req.IsActive, req.CreatedAt, req.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create policy"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"policy": req})
}

func (s *MasterWalletService) UpdatePolicy(c *gin.Context) {
	id := c.Param("id")

	var req Policy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := s.db.Exec(context.Background(),
		`UPDATE policies SET name = $1, policy_type = $2, rules = $3, is_active = $4, updated_at = NOW() WHERE id = $5`,
		req.Name, req.PolicyType, req.Rules, req.IsActive, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update policy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Policy updated"})
}

func (s *MasterWalletService) DeletePolicy(c *gin.Context) {
	id := c.Param("id")

	_, err := s.db.Exec(context.Background(), "DELETE FROM policies WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete policy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Policy deleted"})
}

// User Management Handlers
func (s *MasterWalletService) GetUsers(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")

	rows, err := s.db.Query(context.Background(),
		`SELECT id, email, name, role, master_wallet_id, sub_wallet_id, permissions, two_factor_enabled, is_active, created_at, updated_at, last_login_at 
		 FROM users WHERE master_wallet_id = $1 ORDER BY created_at DESC`,
		masterWalletID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.MasterWalletID, &u.SubWalletID, &u.Permissions, &u.TwoFactorEnabled, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt); err != nil {
			continue
		}
		users = append(users, u)
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (s *MasterWalletService) CreateUser(c *gin.Context) {
	var req struct {
		Email         string   `json:"email" binding:"required,email"`
		Password      string   `json:"password" binding:"required,min=8"`
		Name          string   `json:"name" binding:"required"`
		Role          string   `json:"role"`
		MasterWalletID string  `json:"master_wallet_id"`
		Permissions   []string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	user := User{
		ID:              uuid.New().String(),
		Email:           req.Email,
		Name:            req.Name,
		Role:            req.Role,
		MasterWalletID:  req.MasterWalletID,
		PasswordHash:    string(hashedPassword),
		IsActive:        true,
		FailedAttempts:  0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	_, err := s.db.Exec(context.Background(),
		`INSERT INTO users (id, email, name, role, master_wallet_id, password_hash, permissions, is_active, failed_attempts, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		user.ID, user.Email, user.Name, user.Role, user.MasterWalletID, user.PasswordHash, user.Permissions, user.IsActive, user.FailedAttempts, user.CreatedAt, user.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user_id": user.ID})
}

func (s *MasterWalletService) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name      string   `json:"name"`
		Role     string   `json:"role"`
		Permissions []string `json:"permissions"`
		IsActive bool     `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := s.db.Exec(context.Background(),
		`UPDATE users SET name = COALESCE(NULLIF($1, ''), name), role = COALESCE(NULLIF($2, ''), role), permissions = $3, is_active = $4, updated_at = NOW() WHERE id = $5`,
		req.Name, req.Role, req.Permissions, req.IsActive, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated"})
}

func (s *MasterWalletService) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	_, err := s.db.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func (s *MasterWalletService) ResetPassword(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)

	_, err := s.db.Exec(context.Background(),
		"UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2",
		string(hashedPassword), id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// Analytics Handlers
func (s *MasterWalletService) GetVolumeAnalytics(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")
	ctx := context.Background()

	// Aggregate real transaction volume from the transactions table instead
	// of returning hardcoded mock numbers.
	query := `SELECT COALESCE(SUM(amount::numeric), 0),
	                 COALESCE(SUM(amount::numeric) FILTER (WHERE created_at >= date_trunc('day', NOW())), 0),
	                 COALESCE(SUM(amount::numeric) FILTER (WHERE created_at >= date_trunc('month', NOW())), 0),
	                 COUNT(*)
	          FROM transactions`
	args := []interface{}{}
	if masterWalletID != "" {
		query += " WHERE master_wallet_id = $1"
		args = append(args, masterWalletID)
	}

	var total, daily, monthly string
	var count int64
	if err := s.db.QueryRow(ctx, query, args...).Scan(&total, &daily, &monthly, &count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load volume analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_volume":      total,
		"daily_volume":      daily,
		"monthly_volume":    monthly,
		"transaction_count": count,
	})
}

func (s *MasterWalletService) GetTransactionAnalytics(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")
	ctx := context.Background()

	query := `SELECT COUNT(*) FILTER (WHERE status = 'pending'),
	                 COUNT(*) FILTER (WHERE status = 'confirmed'),
	                 COUNT(*) FILTER (WHERE status = 'failed')
	          FROM transactions`
	args := []interface{}{}
	if masterWalletID != "" {
		query += " WHERE master_wallet_id = $1"
		args = append(args, masterWalletID)
	}

	var pending, confirmed, failed int64
	if err := s.db.QueryRow(ctx, query, args...).Scan(&pending, &confirmed, &failed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load transaction analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_transactions":     pending + confirmed + failed,
		"pending_transactions":   pending,
		"confirmed_transactions": confirmed,
		"failed_transactions":    failed,
	})
}

func (s *MasterWalletService) GetWalletAnalytics(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")
	ctx := context.Background()

	// Aggregate real wallet counts from master_wallets and sub_wallets.
	var totalMW, activeMW int64
	mwQuery := `SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active) FROM master_wallets`
	mwArgs := []interface{}{}
	if masterWalletID != "" {
		mwQuery += " WHERE id = $1"
		mwArgs = append(mwArgs, masterWalletID)
	}
	if err := s.db.QueryRow(ctx, mwQuery, mwArgs...).Scan(&totalMW, &activeMW); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load wallet analytics"})
		return
	}

	var totalSW, activeSW int64
	swQuery := `SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active) FROM sub_wallets`
	swArgs := []interface{}{}
	if masterWalletID != "" {
		swQuery += " WHERE master_wallet_id = $1"
		swArgs = append(swArgs, masterWalletID)
	}
	if err := s.db.QueryRow(ctx, swQuery, swArgs...).Scan(&totalSW, &activeSW); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sub-wallet analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_wallets":      totalMW,
		"active_wallets":     activeMW,
		"sub_wallets":        totalSW,
		"active_sub_wallets": activeSW,
	})
}

// Audit Log Handlers
func (s *MasterWalletService) GetAuditLogs(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")

	rows, err := s.db.Query(context.Background(),
		`SELECT id, user_id, action, entity_type, entity_id, details, ip_address, user_agent, created_at 
		 FROM audit_logs WHERE master_wallet_id = $1 ORDER BY created_at DESC LIMIT 100`,
		masterWalletID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.EntityType, &l.EntityID, &l.Details, &l.IPAddress, &l.UserAgent, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// WebSocket Handler
func (s *MasterWalletService) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Process message and respond
		var msg map[string]interface{}
		json.Unmarshal(message, &msg)

		response := map[string]interface{}{
			"type":    "response",
			"status":  "ok",
			"message": "Processed",
		}

		conn.WriteJSON(response)
	}
}

// ==================== MIDDLEWARE ====================

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:] // Remove "Bearer "
		claims, err := verifyJWT(tokenString, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("email", claims["email"])
		c.Set("role", claims["role"])

		c.Next()
	}
}

// ==================== HELPERS ====================

func healthCheck(c *gin.Context) {
	ctx := context.Background()
	
	// Check database
	dbStatus := "healthy"
	if err := globalConnPool.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
	}

	// Check Redis
	redisStatus := "healthy"
	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisStatus = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"database":  dbStatus,
		"redis":     redisStatus,
		"timestamp": time.Now().Unix(),
	})
}

func generateJWT(userID, email, role, secret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(JWT_EXPIRY).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func verifyJWT(tokenString, secret string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

// ==================== CHAIN RPC CONFIG ====================

// chainRPCEnv maps a chain name to the environment variable that holds its
// JSON-RPC endpoint (e.g. ETH_RPC_URL, BSC_RPC_URL).
var chainRPCEnv = map[string]string{
	"ethereum": "ETH_RPC_URL",
	"bsc":      "BSC_RPC_URL",
	"polygon":  "POLYGON_RPC_URL",
	"arbitrum": "ARBITRUM_RPC_URL",
	"optimism": "OPTIMISM_RPC_URL",
	"avalanche": "AVALANCHE_RPC_URL",
}

// chainIDToName maps the canonical EVM chain ids used by master_wallets to a
// chain name understood by chainRPCEnv.
var chainIDToName = map[int64]string{
	1:       "ethereum",
	56:      "bsc",
	137:     "polygon",
	42161:   "arbitrum",
	10:      "optimism",
	43114:   "avalanche",
}

// chainNameFromID maps a chain id to its canonical name. Unknown ids default
// to ethereum.
func chainNameFromID(chainID int64) string {
	if name, ok := chainIDToName[chainID]; ok {
		return name
	}
	return "ethereum"
}

func rpcURLForChain(chain string) (string, error) {
	envVar, ok := chainRPCEnv[chain]
	if !ok {
		return "", fmt.Errorf("unsupported chain: %s", chain)
	}
	url := os.Getenv(envVar)
	if url == "" {
		return "", fmt.Errorf("RPC URL not configured for chain %s (env %s)", chain, envVar)
	}
	return url, nil
}

// jsonRPCRequest is the standard JSON-RPC 2.0 request envelope.
type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int64         `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// jsonRPCResponse holds a JSON-RPC 2.0 response. Result is decoded lazily by
// the caller.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// rpcCall performs a JSON-RPC 2.0 call against url and unmarshals Result into
// out.
func rpcCall(url, method string, params []interface{}, out interface{}) error {
	body, err := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return fmt.Errorf("marshal rpc request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("rpc call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rpc returned HTTP %d", resp.StatusCode)
	}

	var rpcResp jsonRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("decode rpc response: %w", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if out != nil {
		if err := json.Unmarshal(rpcResp.Result, out); err != nil {
			return fmt.Errorf("decode rpc result: %w", err)
		}
	}
	return nil
}

// ethAddressBytes returns the 20-byte address payload of an 0x-prefixed hex
// address, padding/truncating as needed.
func ethAddressBytes(addr string) ([]byte, error) {
	h := strings.TrimPrefix(addr, "0x")
	raw, err := hex.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", addr, err)
	}
	if len(raw) < 20 {
		padded := make([]byte, 20)
		copy(padded[20-len(raw):], raw)
		raw = padded
	}
	return raw[len(raw)-20:], nil
}

// erc20TransferData builds the calldata for an ERC-20 transfer(to, amount).
func erc20TransferData(to string, amount *big.Int) ([]byte, error) {
	toBytes, err := ethAddressBytes(to)
	if err != nil {
		return nil, err
	}
	// transfer(address,uint256) selector = 0xa9059cbb
	data := make([]byte, 4+32+32)
	data[0], data[1], data[2], data[3] = 0xa9, 0x05, 0x9c, 0xbb
	copy(data[4+12:], toBytes) // right-align address in 32-byte word
	amountBytes := amount.Bytes()
	copy(data[4+32+(32-len(amountBytes)):], amountBytes)
	return data, nil
}

// amountToWei converts a decimal amount string (e.g. "1.5") to its smallest
// unit integer value with the given decimals (e.g. 18 for ETH).
func amountToWei(amount string, decimals int) (*big.Int, error) {
	f, ok := new(big.Float).SetString(amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount %q", amount)
	}
	scaler := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	wei, _ := new(big.Float).Mul(f, scaler).Int(nil)
	return wei, nil
}

// hexToQuantity formats an *big.Int as a JSON-RPC hex quantity ("0x...").
func hexToQuantity(n *big.Int) string {
	if n == nil {
		return "0x0"
	}
	return "0x" + n.Text(16)
}

// broadcastTransactionByChain constructs and broadcasts a transaction via the
// chain's RPC node using eth_sendTransaction, returning the real tx hash from
// the RPC response. It never fabricates a hash; on failure it returns an
// error.
//
// The transaction is submitted as an unlocked "from" send via
// eth_sendTransaction, which requires the node to hold the sending account's
// key (e.g. a managed keystore/HSM-backed node). For a fully self-managed
// flow, callers should sign locally and use eth_sendRawTransaction with the
// resulting raw bytes instead.
func broadcastTransactionByChain(chain, from, to, amount, token string) (string, error) {
	rpcURL, err := rpcURLForChain(chain)
	if err != nil {
		return "", err
	}

	var value string
	var data string

	// Native asset transfer (token == "" or token == chain's native symbol).
	nativeSymbol := map[string]string{
		"ethereum": "ETH", "bsc": "BNB", "polygon": "MATIC",
		"arbitrum": "ETH", "optimism": "ETH", "avalanche": "AVAX",
	}
	if token == "" || strings.EqualFold(token, nativeSymbol[chain]) {
		wei, err := amountToWei(amount, 18)
		if err != nil {
			return "", err
		}
		value = hexToQuantity(wei)
		data = "0x"
	} else {
		// ERC-20 transfer: value 0, data = transfer(to, amount) with the
		// token contract as 'to'. 'token' is expected to be the token
		// contract address here.
		wei, err := amountToWei(amount, 18) // default 18 decimals; refine with per-token metadata as needed
		if err != nil {
			return "", err
		}
		calldata, err := erc20TransferData(to, wei)
		if err != nil {
			return "", err
		}
		value = "0x0"
		data = "0x" + hex.EncodeToString(calldata)
		to = token // destination becomes the token contract
	}

	// Fetch the sender's nonce and a suggested gas price from the node so the
	// submitted transaction is valid.
	var nonceHex, gasPriceHex string
	if err := rpcCall(rpcURL, "eth_getTransactionCount", []interface{}{from, "latest"}, &nonceHex); err != nil {
		return "", fmt.Errorf("get nonce: %w", err)
	}
	if err := rpcCall(rpcURL, "eth_gasPrice", []interface{}{}, &gasPriceHex); err != nil {
		return "", fmt.Errorf("get gas price: %w", err)
	}

	txObj := map[string]interface{}{
		"from":     from,
		"to":       to,
		"value":    value,
		"data":     data,
		"nonce":    nonceHex,
		"gasPrice": gasPriceHex,
	}
	// Default gas limit for a simple transfer; node may still estimate.
	txObj["gas"] = "0x5208"

	var txHash string
	if err := rpcCall(rpcURL, "eth_sendTransaction", []interface{}{txObj}, &txHash); err != nil {
		return "", fmt.Errorf("send transaction: %w", err)
	}
	if txHash == "" {
		return "", fmt.Errorf("rpc returned empty transaction hash")
	}
	return txHash, nil
}


func generatePrivateKey() ([]byte, error) {
	curve := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}
	return privateKey.D.Bytes(), nil
}

func derivePublicKey(privateKey []byte) []byte {
	curve := elliptic.P256()
	x, y := curve.ScalarBaseMult(privateKey)
	return elliptic.Marshal(curve, x, y)
}

func deriveAddress(publicKey []byte) string {
	hash := sha256.Sum256(publicKey)
	return fmt.Sprintf("0x%x", hash[len(hash)-20:])
}


// ==================== DATABASE MIGRATION ====================

func runMigrations(db *pgxpool.Pool) error {
	ctx := context.Background()

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS master_wallets (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			wallet_type VARCHAR(50),
			address VARCHAR(100) UNIQUE NOT NULL,
			public_key TEXT,
			encrypted_seed TEXT,
			chain_id BIGINT,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by UUID
		)`,
		`CREATE TABLE IF NOT EXISTS sub_wallets (
			id UUID PRIMARY KEY,
			master_wallet_id UUID REFERENCES master_wallets(id),
			name VARCHAR(255) NOT NULL,
			address VARCHAR(100) NOT NULL,
			address_type VARCHAR(50),
			public_key TEXT,
			encrypted_key TEXT,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255),
			role VARCHAR(50) DEFAULT 'user',
			master_wallet_id UUID REFERENCES master_wallets(id),
			sub_wallet_id UUID REFERENCES sub_wallets(id),
			password_hash TEXT NOT NULL,
			two_factor_enabled BOOLEAN DEFAULT false,
			two_factor_secret TEXT,
			permissions TEXT[],
			is_active BOOLEAN DEFAULT true,
			failed_attempts INT DEFAULT 0,
			locked_until TIMESTAMP,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_login_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id UUID PRIMARY KEY,
			master_wallet_id UUID REFERENCES master_wallets(id),
			sub_wallet_id UUID REFERENCES sub_wallets(id),
			hash VARCHAR(100) UNIQUE NOT NULL,
			from_address VARCHAR(100),
			to_address VARCHAR(100) NOT NULL,
			amount VARCHAR(100) NOT NULL,
			token VARCHAR(50),
			chain_id BIGINT,
			status VARCHAR(50) DEFAULT 'pending',
			fee VARCHAR(50),
			block_number BIGINT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP,
			confirmed_at TIMESTAMP,
			approved_by UUID,
			rejected_by UUID,
			reject_reason TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS auto_sign_rules (
			id UUID PRIMARY KEY,
			master_wallet_id UUID REFERENCES master_wallets(id),
			name VARCHAR(255) NOT NULL,
			max_amount VARCHAR(50),
			chain_ids TEXT[],
			token_ids TEXT[],
			enabled BOOLEAN DEFAULT true,
			conditions TEXT[],
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS fee_configs (
			id UUID PRIMARY KEY,
			master_wallet_id UUID REFERENCES master_wallets(id),
			name VARCHAR(255) NOT NULL,
			fee_type VARCHAR(50) NOT NULL,
			percentage DECIMAL(10,4) DEFAULT 0,
			flat_fee VARCHAR(50) DEFAULT '0',
			min_amount VARCHAR(50),
			max_amount VARCHAR(50),
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS policies (
			id UUID PRIMARY KEY,
			master_wallet_id UUID REFERENCES master_wallets(id),
			name VARCHAR(255) NOT NULL,
			policy_type VARCHAR(50) NOT NULL,
			rules JSONB,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY,
			master_wallet_id UUID,
			user_id UUID,
			action VARCHAR(100) NOT NULL,
			entity_type VARCHAR(50),
			entity_id UUID,
			details TEXT,
			ip_address VARCHAR(50),
			user_agent TEXT,
			created_at TIMESTAMP NOT NULL
		)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// ==================== MAIN ====================

func main() {
	// Configuration
	cfg := Config{
		Server: ServerConfig{
			Port:            8080,
			Mode:            "release",
			ReadTimeout:     30,
			WriteTimeout:    30,
			MaxHeaderBytes: MAX_HEADER_BYTES,
			EnableTLS:      false,
		},
		Database: DatabaseConfig{
			Host:            os.Getenv("DB_HOST"),
			Port:            5432,
			Database:        os.Getenv("DB_NAME"),
			Username:        os.Getenv("DB_USER"),
			Password:        os.Getenv("DB_PASSWORD"),
			MaxConns:        DB_MAX_CONNS,
			MinConns:        DB_MIN_CONNS,
			MaxConnLifetime: DB_MAX_CONN_LIFETIME,
			SSLMode:         "disable",
		},
		Redis: RedisConfig{
			Host:         os.Getenv("REDIS_HOST"),
			Port:         6379,
			Password:     os.Getenv("REDIS_PASSWORD"),
			DB:           0,
			PoolSize:     REDIS_POOL_SIZE,
			MinIdleConns: 10,
		},
		Security: SecurityConfig{
			JWTSecret:        "tigerwallet-secret-key-change-in-production",
			Argon2Time:       2,
			Argon2Memory:     65536,
			Argon2Threads:    4,
			Argon2KeyLen:     32,
			BcryptCost:       12,
			MaxLoginAttempts:  5,
			LockoutDuration:  900,
		},
	}

	// Override with defaults if not set
	if cfg.Database.Host == "" {
		cfg.Database.Host = "localhost"
		cfg.Database.Database = "tigerwallet"
		cfg.Database.Username = "postgres"
		cfg.Database.Password = "password"
	}
	if cfg.Redis.Host == "" {
		cfg.Redis.Host = "localhost"
	}

	// Initialize database
	var err error
	globalConnPool, err = initDatabase(cfg.Database)
	if err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer globalConnPool.Close()

	// Run migrations
	if err := runMigrations(globalConnPool); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	// Initialize Redis
	redisClient, err = initRedis(cfg.Redis)
	if err != nil {
		fmt.Printf("Failed to initialize Redis: %v\n", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Initialize service
	svc := NewMasterWalletService(globalConnPool, redisClient)

	// Setup Gin
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Setup routes
	setupRoutes(r, svc, cfg.Security)

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Configure HTTP/2
	http2.ConfigureServer(srv, &http2.Server{})

	// Enable TLS if configured
	if cfg.Server.EnableTLS {
		srv.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:        []tls.CurveID{tls.CurveP256, tls.X25519},
			PreferServerCipherSuites: true,
		}
	}

	// Start server in goroutine
	go func() {
		fmt.Printf("Starting MasterWallet API server on port %d\n", cfg.Server.Port)
		if cfg.Server.EnableTLS {
			srv.ListenAndServeTLS(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
		} else {
			srv.ListenAndServe()
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), GRACEFUL_TIMEOUT)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server exited")
}
