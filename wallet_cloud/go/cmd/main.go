// Wallet Cloud Service - PostgreSQL Version
// Cloud-hosted wallet management for TigerWallet ecosystem

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

// Wallet Cloud Models
type CloudWallet struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	WalletType    string    `json:"wallet_type"` // hot, warm, cold
	EncryptionKey  string    `json:"-"`
	PublicKey     string    `json:"public_key"`
	Address       string    `json:"address"`
	ChainID       int       `json:"chain_id"`
	Status        string    `json:"status"` // active, frozen, archived
	IsMultiSig    bool      `json:"is_multi_sig"`
	Threshold     int       `json:"threshold"` // for multi-sig
	Signers       string    `json:"signers"` // JSON array
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CloudTransaction struct {
	ID            uuid.UUID `json:"id"`
	WalletID      uuid.UUID `json:"wallet_id"`
	FromAddress   string    `json:"from_address"`
	ToAddress     string    `json:"to_address"`
	Amount        string    `json:"amount"`
	TokenAddress  string    `json:"token_address"`
	Fee           string    `json:"fee"`
	Status        string    `json:"status"` // pending, signed, broadcast, confirmed, failed
	Hash          string    `json:"hash"`
	RawTX         string    `json:"raw_tx"`
	Signatures    string    `json:"signatures"` // JSON array
	ChainID       int       `json:"chain_id"`
	Nonce         int64     `json:"nonce"`
	GasLimit      uint64    `json:"gas_limit"`
	GasPrice      string    `json:"gas_price"`
	CreatedAt     time.Time `json:"created_at"`
	ConfirmedAt   *time.Time `json:"confirmed_at"`
}

type WalletPolicy struct {
	ID             uuid.UUID `json:"id"`
	WalletID       uuid.UUID `json:"wallet_id"`
	PolicyType     string    `json:"policy_type"` // spending, withdrawal, daily_limit
	Condition      string    `json:"condition"` // JSON expression
	LimitAmount    string    `json:"limit_amount"`
	LimitPeriod    string    `json:"limit_period"` // daily, weekly, monthly
	IsEnabled      bool      `json:"is_enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Backup struct {
	ID          uuid.UUID `json:"id"`
	WalletID    uuid.UUID `json:"wallet_id"`
	BackupType  string    `json:"backup_type"` // encrypted_key, mnemonic, shard
	EncryptedData string  `json:"encrypted_data"`
	ShardIndex  int       `json:"shard_index"`
	TotalShards int       `json:"total_shards"`
	Status      string    `json:"status"` // active, revoked
	CreatedAt   time.Time `json:"created_at"`
}

type KeyShare struct {
	ID             uuid.UUID `json:"id"`
	WalletID       uuid.UUID `json:"wallet_id"`
	ShareID        string    `json:"share_id"`
	EncryptedShare string    `json:"encrypted_share"`
	HolderID       uuid.UUID `json:"holder_id"`
	HolderType     string    `json:"holder_type"` // user, device, server
	Status         string    `json:"status"` // active, revoked
	CreatedAt      time.Time `json:"created_at"`
}

type Recovery struct {
	ID             uuid.UUID `json:"id"`
	WalletID       uuid.UUID `json:"wallet_id"`
	RecoveryType   string    `json:"recovery_type"` // social, device, seed
	Guardians      string    `json:"guardians"` // JSON array of guardian info
	Threshold      int       `json:"threshold"`
	Status         string    `json:"status"` // pending, active, used, expired
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// Global variables
var (
	db     *pgxpool.Pool
	redis  *redis.Client
	config Config
	logger *log.Logger
)

// Initialize database
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
		CREATE TABLE IF NOT EXISTS cloud_wallets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			wallet_type VARCHAR(50) NOT NULL,
			encryption_key TEXT,
			public_key TEXT NOT NULL,
			address VARCHAR(255) NOT NULL,
			chain_id INTEGER NOT NULL,
			status VARCHAR(50) DEFAULT 'active',
			is_multi_sig BOOLEAN DEFAULT false,
			threshold INTEGER DEFAULT 1,
			signers JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS cloud_transactions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID REFERENCES cloud_wallets(id),
			from_address VARCHAR(255) NOT NULL,
			to_address VARCHAR(255) NOT NULL,
			amount VARCHAR(255) NOT NULL,
			token_address VARCHAR(255),
			fee VARCHAR(255),
			status VARCHAR(50) DEFAULT 'pending',
			hash VARCHAR(255),
			raw_tx TEXT,
			signatures JSONB,
			chain_id INTEGER NOT NULL,
			nonce BIGINT,
			gas_limit BIGINT,
			gas_price VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW(),
			confirmed_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS wallet_policies (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID REFERENCES cloud_wallets(id),
			policy_type VARCHAR(50) NOT NULL,
			condition TEXT,
			limit_amount VARCHAR(255),
			limit_period VARCHAR(50),
			is_enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS wallet_backups (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID REFERENCES cloud_wallets(id),
			backup_type VARCHAR(50) NOT NULL,
			encrypted_data TEXT NOT NULL,
			shard_index INTEGER DEFAULT 0,
			total_shards INTEGER DEFAULT 1,
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS key_shares (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID REFERENCES cloud_wallets(id),
			share_id VARCHAR(255) NOT NULL,
			encrypted_share TEXT NOT NULL,
			holder_id UUID NOT NULL,
			holder_type VARCHAR(50) NOT NULL,
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS wallet_recovery (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID REFERENCES cloud_wallets(id),
			recovery_type VARCHAR(50) NOT NULL,
			guardians JSONB,
			threshold INTEGER DEFAULT 1,
			status VARCHAR(50) DEFAULT 'pending',
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_cloud_wallets_user ON cloud_wallets(user_id);
		CREATE INDEX IF NOT EXISTS idx_cloud_wallets_address ON cloud_wallets(address);
		CREATE INDEX IF NOT EXISTS idx_cloud_transactions_wallet ON cloud_transactions(wallet_id);
		CREATE INDEX IF NOT EXISTS idx_cloud_transactions_hash ON cloud_transactions(hash);
		CREATE INDEX IF NOT EXISTS idx_wallet_policies_wallet ON wallet_policies(wallet_id);
		CREATE INDEX IF NOT EXISTS idx_wallet_backups_wallet ON wallet_backups(wallet_id);
		CREATE INDEX IF NOT EXISTS idx_key_shares_wallet ON key_shares(wallet_id);
		CREATE INDEX IF NOT EXISTS idx_wallet_recovery_wallet ON wallet_recovery(wallet_id);
	`)

	return err
}

// Initialize Redis
func initRedis() error {
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return err
	}

	redis = redis.NewClient(opt)
	return redis.Ping(context.Background()).Err()
}

// Handlers

// CreateCloudWallet - Create a new cloud wallet
func CreateCloudWallet(c *gin.Context) {
	var wallet CloudWallet
	if err := c.ShouldBindJSON(&wallet); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet.ID = uuid.New()
	wallet.CreatedAt = time.Now()
	wallet.UpdatedAt = time.Now()

	signersJSON, _ := json.Marshal(wallet.Signers)

	_, err := db.Exec(context.Background(), `
		INSERT INTO cloud_wallets (id, user_id, wallet_type, encryption_key, public_key, address, chain_id, status, is_multi_sig, threshold, signers, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, wallet.ID, wallet.UserID, wallet.WalletType, wallet.EncryptionKey, wallet.PublicKey, wallet.Address, wallet.ChainID, wallet.Status, wallet.IsMultiSig, wallet.Threshold, signersJSON, wallet.CreatedAt, wallet.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wallet)
}

// GetCloudWallets - Get cloud wallets
func GetCloudWallets(c *gin.Context) {
	userID := c.Query("user_id")

	query := `
		SELECT id, user_id, wallet_type, public_key, address, chain_id, status, is_multi_sig, threshold, signers, created_at, updated_at
		FROM cloud_wallets
	`
	if userID != "" {
		query += fmt.Sprintf(" WHERE user_id = '%s'", userID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var wallets []CloudWallet
	for rows.Next() {
		var wallet CloudWallet
		var signers []byte
		if err := rows.Scan(&wallet.ID, &wallet.UserID, &wallet.WalletType, &wallet.EncryptionKey, &wallet.PublicKey, &wallet.Address, &wallet.ChainID, &wallet.Status, &wallet.IsMultiSig, &wallet.Threshold, &signers, &wallet.CreatedAt, &wallet.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal(signers, &wallet.Signers)
		wallets = append(wallets, wallet)
	}

	c.JSON(http.StatusOK, gin.H{"wallets": wallets})
}

// CreateTransaction - Create cloud transaction
func CreateTransaction(c *gin.Context) {
	var tx CloudTransaction
	if err := c.ShouldBindJSON(&tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx.ID = uuid.New()
	tx.CreatedAt = time.Now()
	tx.Status = "pending"

	signaturesJSON, _ := json.Marshal(tx.Signatures)

	_, err := db.Exec(context.Background(), `
		INSERT INTO cloud_transactions (id, wallet_id, from_address, to_address, amount, token_address, fee, status, hash, raw_tx, signatures, chain_id, nonce, gas_limit, gas_price, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, tx.ID, tx.WalletID, tx.FromAddress, tx.ToAddress, tx.Amount, tx.TokenAddress, tx.Fee, tx.Status, tx.Hash, tx.RawTX, signaturesJSON, tx.ChainID, tx.Nonce, tx.GasLimit, tx.GasPrice, tx.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tx)
}

// GetTransactions - Get transactions
func GetTransactions(c *gin.Context) {
	walletID := c.Query("wallet_id")
	limit := c.DefaultQuery("limit", "50")

	query := `
		SELECT id, wallet_id, from_address, to_address, amount, token_address, fee, status, hash, raw_tx, signatures, chain_id, nonce, gas_limit, gas_price, created_at, confirmed_at
		FROM cloud_transactions
	`
	if walletID != "" {
		query += fmt.Sprintf(" WHERE wallet_id = '%s'", walletID)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %s", limit)

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var txs []CloudTransaction
	for rows.Next() {
		var tx CloudTransaction
		var signatures []byte
		if err := rows.Scan(&tx.ID, &tx.WalletID, &tx.FromAddress, &tx.ToAddress, &tx.Amount, &tx.TokenAddress, &tx.Fee, &tx.Status, &tx.Hash, &tx.RawTX, &signatures, &tx.ChainID, &tx.Nonce, &tx.GasLimit, &tx.GasPrice, &tx.CreatedAt, &tx.ConfirmedAt); err != nil {
			continue
		}
		json.Unmarshal(signatures, &tx.Signatures)
		txs = append(txs, tx)
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

// SignTransaction - Sign transaction
func SignTransaction(c *gin.Context) {
	txID := c.Param("id")
	var req struct {
		Signature string `json:"signature"`
		Signer    string `json:"signer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current signatures
	var currentSigs string
	db.QueryRow(context.Background(), "SELECT signatures FROM cloud_transactions WHERE id = $1", txID).Scan(&currentSigs)

	var sigs []string
	json.Unmarshal([]byte(currentSigs), &sigs)
	sigs = append(sigs, req.Signer+":"+req.Signature)
	sigsJSON, _ := json.Marshal(sigs)

	_, err := db.Exec(context.Background(), `
		UPDATE cloud_transactions SET signatures = $1 WHERE id = $2
	`, sigsJSON, txID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transaction signed"})
}

// BroadcastTransaction - Broadcast transaction
func BroadcastTransaction(c *gin.Context) {
	txID := c.Param("id")
	var req struct {
		Hash string `json:"hash"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(context.Background(), `
		UPDATE cloud_transactions SET status = 'broadcast', hash = $1 WHERE id = $2
	`, req.Hash, txID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transaction broadcast"})
}

// CreatePolicy - Create wallet policy
func CreatePolicy(c *gin.Context) {
	var policy WalletPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy.ID = uuid.New()
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO wallet_policies (id, wallet_id, policy_type, condition, limit_amount, limit_period, is_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, policy.ID, policy.WalletID, policy.PolicyType, policy.Condition, policy.LimitAmount, policy.LimitPeriod, policy.IsEnabled, policy.CreatedAt, policy.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// GetPolicies - Get policies
func GetPolicies(c *gin.Context) {
	walletID := c.Query("wallet_id")

	query := `
		SELECT id, wallet_id, policy_type, condition, limit_amount, limit_period, is_enabled, created_at, updated_at
		FROM wallet_policies
	`
	if walletID != "" {
		query += fmt.Sprintf(" WHERE wallet_id = '%s'", walletID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var policies []WalletPolicy
	for rows.Next() {
		var policy WalletPolicy
		if err := rows.Scan(&policy.ID, &policy.WalletID, &policy.PolicyType, &policy.Condition, &policy.LimitAmount, &policy.LimitPeriod, &policy.IsEnabled, &policy.CreatedAt, &policy.UpdatedAt); err != nil {
			continue
		}
		policies = append(policies, policy)
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// CreateBackup - Create wallet backup
func CreateBackup(c *gin.Context) {
	var backup Backup
	if err := c.ShouldBindJSON(&backup); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	backup.ID = uuid.New()
	backup.CreatedAt = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO wallet_backups (id, wallet_id, backup_type, encrypted_data, shard_index, total_shards, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, backup.ID, backup.WalletID, backup.BackupType, backup.EncryptedData, backup.ShardIndex, backup.TotalShards, backup.Status, backup.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, backup)
}

// GetBackups - Get backups
func GetBackups(c *gin.Context) {
	walletID := c.Query("wallet_id")

	query := `
		SELECT id, wallet_id, backup_type, encrypted_data, shard_index, total_shards, status, created_at
		FROM wallet_backups
	`
	if walletID != "" {
		query += fmt.Sprintf(" WHERE wallet_id = '%s'", walletID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var backups []Backup
	for rows.Next() {
		var backup Backup
		if err := rows.Scan(&backup.ID, &backup.WalletID, &backup.BackupType, &backup.EncryptedData, &backup.ShardIndex, &backup.TotalShards, &backup.Status, &backup.CreatedAt); err != nil {
			continue
		}
		backups = append(backups, backup)
	}

	c.JSON(http.StatusOK, gin.H{"backups": backups})
}

// CreateKeyShare - Create key share
func CreateKeyShare(c *gin.Context) {
	var keyShare KeyShare
	if err := c.ShouldBindJSON(&keyShare); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	keyShare.ID = uuid.New()
	keyShare.CreatedAt = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO key_shares (id, wallet_id, share_id, encrypted_share, holder_id, holder_type, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, keyShare.ID, keyShare.WalletID, keyShare.ShareID, keyShare.EncryptedShare, keyShare.HolderID, keyShare.HolderType, keyShare.Status, keyShare.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, keyShare)
}

// GetKeyShares - Get key shares
func GetKeyShares(c *gin.Context) {
	walletID := c.Query("wallet_id")

	query := `
		SELECT id, wallet_id, share_id, encrypted_share, holder_id, holder_type, status, created_at
		FROM key_shares
	`
	if walletID != "" {
		query += fmt.Sprintf(" WHERE wallet_id = '%s'", walletID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var shares []KeyShare
	for rows.Next() {
		var share KeyShare
		if err := rows.Scan(&share.ID, &share.WalletID, &share.ShareID, &share.EncryptedShare, &share.HolderID, &share.HolderType, &share.Status, &share.CreatedAt); err != nil {
			continue
		}
		shares = append(shares, share)
	}

	c.JSON(http.StatusOK, gin.H{"key_shares": shares})
}

// CreateRecovery - Create recovery setup
func CreateRecovery(c *gin.Context) {
	var recovery Recovery
	if err := c.ShouldBindJSON(&recovery); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recovery.ID = uuid.New()
	recovery.CreatedAt = time.Now()

	guardiansJSON, _ := json.Marshal(recovery.Guardians)

	_, err := db.Exec(context.Background(), `
		INSERT INTO wallet_recovery (id, wallet_id, recovery_type, guardians, threshold, status, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, recovery.ID, recovery.WalletID, recovery.RecoveryType, guardiansJSON, recovery.Threshold, recovery.Status, recovery.ExpiresAt, recovery.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, recovery)
}

// GetRecoveries - Get recoveries
func GetRecoveries(c *gin.Context) {
	walletID := c.Query("wallet_id")

	query := `
		SELECT id, wallet_id, recovery_type, guardians, threshold, status, expires_at, created_at
		FROM wallet_recovery
	`
	if walletID != "" {
		query += fmt.Sprintf(" WHERE wallet_id = '%s'", walletID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var recoveries []Recovery
	for rows.Next() {
		var recovery Recovery
		var guardians []byte
		if err := rows.Scan(&recovery.ID, &recovery.WalletID, &recovery.RecoveryType, &guardians, &recovery.Threshold, &recovery.Status, &recovery.ExpiresAt, &recovery.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(guardians, &recovery.Guardians)
		recoveries = append(recoveries, recovery)
	}

	c.JSON(http.StatusOK, gin.H{"recoveries": recoveries})
}

// Health check
func HealthCheck(c *gin.Context) {
	ctx := context.Background()
	
	dbStatus := "healthy"
	if err := db.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
	}
	
	redisStatus := "healthy"
	if err := redis.Ping(ctx).Err(); err != nil {
		redisStatus = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"database":   dbStatus,
		"redis":      redisStatus,
		"timestamp":  time.Now(),
	})
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	logger = log.New(os.Stdout, "Wallet Cloud: ", log.LstdFlags)
	logger.Println("Starting Wallet Cloud Service...")

	// Load configuration
	config.Port = getEnv("WALLET_CLOUD_PORT", "8097")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")
	config.JWTSecret = getEnv("JWT_SECRET", "tigerwallet-secret-key")

	// Initialize database
	if err := initDatabase(); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	logger.Println("Database connected successfully")

	// Initialize Redis
	if err := initRedis(); err != nil {
		logger.Fatalf("Failed to initialize Redis: %v", err)
	}
	logger.Println("Redis connected successfully")

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
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

	// Health check
	router.GET("/health", HealthCheck)

	// Wallet routes
	router.POST("/api/v1/wallets", CreateCloudWallet)
	router.GET("/api/v1/wallets", GetCloudWallets)

	// Transaction routes
	router.POST("/api/v1/transactions", CreateTransaction)
	router.GET("/api/v1/transactions", GetTransactions)
	router.POST("/api/v1/transactions/:id/sign", SignTransaction)
	router.POST("/api/v1/transactions/:id/broadcast", BroadcastTransaction)

	// Policy routes
	router.POST("/api/v1/policies", CreatePolicy)
	router.GET("/api/v1/policies", GetPolicies)

	// Backup routes
	router.POST("/api/v1/backups", CreateBackup)
	router.GET("/api/v1/backups", GetBackups)

	// Key share routes
	router.POST("/api/v1/key-shares", CreateKeyShare)
	router.GET("/api/v1/key-shares", GetKeyShares)

	// Recovery routes
	router.POST("/api/v1/recovery", CreateRecovery)
	router.GET("/api/v1/recovery", GetRecoveries)

	// Start server
	logger.Printf("Starting server on port %s", config.Port)
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Println("Server started successfully")

	// Wait for interrupt signal
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
