// Asset Management Service - PostgreSQL Version
// Complete digital asset management for TigerWallet ecosystem

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

// Asset types
type Asset struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Symbol            string    `json:"symbol"`
	AssetType         string    `json:"asset_type"` // crypto, nft, token, fiat
	ChainID           *int      `json:"chain_id"`
	ContractAddress   *string   `json:"contract_address"`
	Decimals          int       `json:"decimals"`
	TotalSupply       string    `json:"total_supply"`
	IsActive          bool      `json:"is_active"`
	IsVerified        bool      `json:"is_verified"`
	LogoURL           string    `json:"logo_url"`
	Description       string    `json:"description"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type AssetBalance struct {
	ID             uuid.UUID `json:"id"`
	WalletAddress  string    `json:"wallet_address"`
	AssetID        uuid.UUID `json:"asset_id"`
	Balance        string    `json:"balance"`
	Available      string    `json:"available"`
	Locked         string    `json:"locked"`
	LastUpdated    time.Time `json:"last_updated"`
}

type AssetPrice struct {
	ID            uuid.UUID `json:"id"`
	AssetID       uuid.UUID `json:"asset_id"`
	Price         string    `json:"price"`
	Change24h     string    `json:"change_24h"`
	Volume24h     string    `json:"volume_24h"`
	MarketCap     string    `json:"market_cap"`
	Timestamp     time.Time `json:"timestamp"`
}

type Transaction struct {
	ID              uuid.UUID `json:"id"`
	FromAddress     string    `json:"from_address"`
	ToAddress       string    `json:"to_address"`
	AssetID         uuid.UUID `json:"asset_id"`
	Amount          string    `json:"amount"`
	Fee             string    `json:"fee"`
	Status          string    `json:"status"` // pending, confirmed, failed
	Hash            string    `json:"hash"`
	ChainID         int       `json:"chain_id"`
	BlockNumber     *int64    `json:"block_number"`
	Timestamp       time.Time `json:"timestamp"`
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
		CREATE TABLE IF NOT EXISTS assets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			symbol VARCHAR(50) NOT NULL,
			asset_type VARCHAR(50) NOT NULL,
			chain_id INTEGER,
			contract_address VARCHAR(255),
			decimals INTEGER DEFAULT 18,
			total_supply VARCHAR(255),
			is_active BOOLEAN DEFAULT true,
			is_verified BOOLEAN DEFAULT false,
			logo_url TEXT,
			description TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS asset_balances (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_address VARCHAR(255) NOT NULL,
			asset_id UUID REFERENCES assets(id),
			balance VARCHAR(255) DEFAULT '0',
			available VARCHAR(255) DEFAULT '0',
			locked VARCHAR(255) DEFAULT '0',
			last_updated TIMESTAMP DEFAULT NOW(),
			UNIQUE(wallet_address, asset_id)
		);

		CREATE TABLE IF NOT EXISTS asset_prices (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			asset_id UUID REFERENCES assets(id),
			price VARCHAR(255) NOT NULL,
			change_24h VARCHAR(255),
			volume_24h VARCHAR(255),
			market_cap VARCHAR(255),
			timestamp TIMESTAMP DEFAULT NOW(),
			UNIQUE(asset_id, timestamp)
		);

		CREATE TABLE IF NOT EXISTS asset_transactions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			from_address VARCHAR(255) NOT NULL,
			to_address VARCHAR(255) NOT NULL,
			asset_id UUID REFERENCES assets(id),
			amount VARCHAR(255) NOT NULL,
			fee VARCHAR(255),
			status VARCHAR(50) DEFAULT 'pending',
			hash VARCHAR(255) UNIQUE,
			chain_id INTEGER NOT NULL,
			block_number BIGINT,
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_assets_symbol ON assets(symbol);
		CREATE INDEX IF NOT EXISTS idx_assets_chain ON assets(chain_id);
		CREATE INDEX IF NOT EXISTS idx_balances_wallet ON asset_balances(wallet_address);
		CREATE INDEX IF NOT EXISTS idx_transactions_hash ON asset_transactions(hash);
		CREATE INDEX IF NOT EXISTS idx_transactions_address ON asset_transactions(from_address, to_address);
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

// CreateAsset - Create a new asset
func CreateAsset(c *gin.Context) {
	var asset Asset
	if err := c.ShouldBindJSON(&asset); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset.ID = uuid.New()
	asset.CreatedAt = time.Now()
	asset.UpdatedAt = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO assets (id, name, symbol, asset_type, chain_id, contract_address, decimals, total_supply, is_active, is_verified, logo_url, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, asset.ID, asset.Name, asset.Symbol, asset.AssetType, asset.ChainID, asset.ContractAddress, asset.Decimals, asset.TotalSupply, asset.IsActive, asset.IsVerified, asset.LogoURL, asset.Description, asset.CreatedAt, asset.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, asset)
}

// GetAssets - Get all assets
func GetAssets(c *gin.Context) {
	rows, err := db.Query(context.Background(), `
		SELECT id, name, symbol, asset_type, chain_id, contract_address, decimals, total_supply, is_active, is_verified, logo_url, description, created_at, updated_at
		FROM assets
		ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var asset Asset
		if err := rows.Scan(&asset.ID, &asset.Name, &asset.Symbol, &asset.AssetType, &asset.ChainID, &asset.ContractAddress, &asset.Decimals, &asset.TotalSupply, &asset.IsActive, &asset.IsVerified, &asset.LogoURL, &asset.Description, &asset.CreatedAt, &asset.UpdatedAt); err != nil {
			continue
		}
		assets = append(assets, asset)
	}

	c.JSON(http.StatusOK, gin.H{"assets": assets, "total": len(assets)})
}

// GetAsset - Get asset by ID
func GetAsset(c *gin.Context) {
	id := c.Param("id")
	assetID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
		return
	}

	var asset Asset
	err = db.QueryRow(context.Background(), `
		SELECT id, name, symbol, asset_type, chain_id, contract_address, decimals, total_supply, is_active, is_verified, logo_url, description, created_at, updated_at
		FROM assets WHERE id = $1
	`, assetID).Scan(&asset.ID, &asset.Name, &asset.Symbol, &asset.AssetType, &asset.ChainID, &asset.ContractAddress, &asset.Decimals, &asset.TotalSupply, &asset.IsActive, &asset.IsVerified, &asset.LogoURL, &asset.Description, &asset.CreatedAt, &asset.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}

	c.JSON(http.StatusOK, asset)
}

// UpdateAsset - Update asset
func UpdateAsset(c *gin.Context) {
	id := c.Param("id")
	assetID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
		return
	}

	var asset Asset
	if err := c.ShouldBindJSON(&asset); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset.UpdatedAt = time.Now()

	_, err = db.Exec(context.Background(), `
		UPDATE assets SET name = $1, symbol = $2, asset_type = $3, chain_id = $4, contract_address = $5, decimals = $6, total_supply = $7, is_active = $8, is_verified = $9, logo_url = $10, description = $11, updated_at = $12
		WHERE id = $13
	`, asset.Name, asset.Symbol, asset.AssetType, asset.ChainID, asset.ContractAddress, asset.Decimals, asset.TotalSupply, asset.IsActive, asset.IsVerified, asset.LogoURL, asset.Description, asset.UpdatedAt, assetID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "asset updated successfully"})
}

// DeleteAsset - Delete asset
func DeleteAsset(c *gin.Context) {
	id := c.Param("id")
	assetID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
		return
	}

	_, err = db.Exec(context.Background(), "DELETE FROM assets WHERE id = $1", assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "asset deleted successfully"})
}

// GetWalletBalance - Get wallet balance for an asset
func GetWalletBalance(c *gin.Context) {
	wallet := c.Param("address")
	assetID := c.Query("asset_id")

	if assetID != "" {
		assetUUID, _ := uuid.Parse(assetID)
		var balance AssetBalance
		err := db.QueryRow(context.Background(), `
			SELECT id, wallet_address, asset_id, balance, available, locked, last_updated
			FROM asset_balances WHERE wallet_address = $1 AND asset_id = $2
		`, wallet, assetUUID).Scan(&balance.ID, &balance.WalletAddress, &balance.AssetID, &balance.Balance, &balance.Available, &balance.Locked, &balance.LastUpdated)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "balance not found"})
			return
		}
		c.JSON(http.StatusOK, balance)
	} else {
		rows, err := db.Query(context.Background(), `
			SELECT id, wallet_address, asset_id, balance, available, locked, last_updated
			FROM asset_balances WHERE wallet_address = $1
		`, wallet)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var balances []AssetBalance
		for rows.Next() {
			var balance AssetBalance
			if err := rows.Scan(&balance.ID, &balance.WalletAddress, &balance.AssetID, &balance.Balance, &balance.Available, &balance.Locked, &balance.LastUpdated); err != nil {
				continue
			}
			balances = append(balances, balance)
		}
		c.JSON(http.StatusOK, gin.H{"balances": balances})
	}
}

// UpdateWalletBalance - Update wallet balance
func UpdateWalletBalance(c *gin.Context) {
	var balance AssetBalance
	if err := c.ShouldBindJSON(&balance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	balance.LastUpdated = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO asset_balances (id, wallet_address, asset_id, balance, available, locked, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (wallet_address, asset_id) DO UPDATE SET
			balance = $4, available = $5, locked = $6, last_updated = $7
	`, uuid.New(), balance.WalletAddress, balance.AssetID, balance.Balance, balance.Available, balance.Locked, balance.LastUpdated)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "balance updated successfully"})
}

// GetAssetPrice - Get current asset price
func GetAssetPrice(c *gin.Context) {
	assetID := c.Param("id")
	assetUUID, err := uuid.Parse(assetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
		return
	}

	var price AssetPrice
	err = db.QueryRow(context.Background(), `
		SELECT id, asset_id, price, change_24h, volume_24h, market_cap, timestamp
		FROM asset_prices WHERE asset_id = $1
		ORDER BY timestamp DESC LIMIT 1
	`, assetUUID).Scan(&price.ID, &price.AssetID, &price.Price, &price.Change24h, &price.Volume24h, &price.MarketCap, &price.Timestamp)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "price not found"})
		return
	}

	c.JSON(http.StatusOK, price)
}

// UpdateAssetPrice - Update asset price
func UpdateAssetPrice(c *gin.Context) {
	var price AssetPrice
	if err := c.ShouldBindJSON(&price); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	price.ID = uuid.New()
	price.Timestamp = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO asset_prices (id, asset_id, price, change_24h, volume_24h, market_cap, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, price.ID, price.AssetID, price.Price, price.Change24h, price.Volume24h, price.MarketCap, price.Timestamp)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, price)
}

// CreateTransaction - Create a new transaction
func CreateTransaction(c *gin.Context) {
	var tx Transaction
	if err := c.ShouldBindJSON(&tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx.ID = uuid.New()
	tx.Timestamp = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO asset_transactions (id, from_address, to_address, asset_id, amount, fee, status, hash, chain_id, block_number, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, tx.ID, tx.FromAddress, tx.ToAddress, tx.AssetID, tx.Amount, tx.Fee, tx.Status, tx.Hash, tx.ChainID, tx.BlockNumber, tx.Timestamp)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tx)
}

// GetTransaction - Get transaction by hash
func GetTransaction(c *gin.Context) {
	hash := c.Param("hash")

	var tx Transaction
	err := db.QueryRow(context.Background(), `
		SELECT id, from_address, to_address, asset_id, amount, fee, status, hash, chain_id, block_number, timestamp
		FROM asset_transactions WHERE hash = $1
	`, hash).Scan(&tx.ID, &tx.FromAddress, &tx.ToAddress, &tx.AssetID, &tx.Amount, &tx.Fee, &tx.Status, &tx.Hash, &tx.ChainID, &tx.BlockNumber, &tx.Timestamp)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	c.JSON(http.StatusOK, tx)
}

// GetTransactions - Get transactions for a wallet
func GetTransactions(c *gin.Context) {
	wallet := c.Query("wallet")
	limit := c.DefaultQuery("limit", "50")

	rows, err := db.Query(context.Background(), `
		SELECT id, from_address, to_address, asset_id, amount, fee, status, hash, chain_id, block_number, timestamp
		FROM asset_transactions
		WHERE from_address = $1 OR to_address = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, wallet, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.FromAddress, &tx.ToAddress, &tx.AssetID, &tx.Amount, &tx.Fee, &tx.Status, &tx.Hash, &tx.ChainID, &tx.BlockNumber, &tx.Timestamp); err != nil {
			continue
		}
		txs = append(txs, tx)
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txs, "total": len(txs)})
}

// UpdateTransactionStatus - Update transaction status
func UpdateTransactionStatus(c *gin.Context) {
	hash := c.Param("hash")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(context.Background(), `
		UPDATE asset_transactions SET status = $1 WHERE hash = $2
	`, req.Status, hash)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated successfully"})
}

// GetWalletAssets - Get all assets for a wallet
func GetWalletAssets(c *gin.Context) {
	wallet := c.Param("address")

	rows, err := db.Query(context.Background(), `
		SELECT ab.id, ab.wallet_address, ab.asset_id, ab.balance, ab.available, ab.last_updated,
			   a.name, a.symbol, a.asset_type, a.logo_url
		FROM asset_balances ab
		JOIN assets a ON ab.asset_id = a.id
		WHERE ab.wallet_address = $1 AND ab.balance > '0'
		ORDER BY ab.last_updated DESC
	`, wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type WalletAsset struct {
		AssetBalance
		AssetName   string `json:"asset_name"`
		AssetSymbol string `json:"asset_symbol"`
		AssetType   string `json:"asset_type"`
		LogoURL     string `json:"logo_url"`
	}

	var assets []WalletAsset
	for rows.Next() {
		var wa WalletAsset
		if err := rows.Scan(&wa.ID, &wa.WalletAddress, &wa.AssetID, &wa.Balance, &wa.Available, &wa.LastUpdated, &wa.AssetName, &wa.AssetSymbol, &wa.AssetType, &wa.LogoURL); err != nil {
			continue
		}
		assets = append(assets, wa)
	}

	c.JSON(http.StatusOK, gin.H{"assets": assets, "total": len(assets)})
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

// Cache price in Redis
func CachePrice(assetID uuid.UUID, price string) error {
	ctx := context.Background()
	key := fmt.Sprintf("asset:price:%s", assetID.String())
	return redis.Set(ctx, key, price, 5*time.Minute).Err()
}

// Get cached price from Redis
func GetCachedPrice(assetID uuid.UUID) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("asset:price:%s", assetID.String())
	return redis.Get(ctx, key).Result()
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	logger = log.New(os.Stdout, "Asset Management: ", log.LstdFlags)
	logger.Println("Starting Asset Management Service...")

	// Load configuration
	config.Port = getEnv("ASSET_PORT", "8085")
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

	// Asset routes
	assets := router.Group("/api/v1/assets")
	{
		assets.POST("", CreateAsset)
		assets.GET("", GetAssets)
		assets.GET("/:id", GetAsset)
		assets.PUT("/:id", UpdateAsset)
		assets.DELETE("/:id", DeleteAsset)
		assets.GET("/:id/price", GetAssetPrice)
		assets.POST("/price", UpdateAssetPrice)
	}

	// Balance routes
	balances := router.Group("/api/v1/balances")
	{
		balances.GET("/:address", GetWalletBalance)
		balances.GET("/:address/assets", GetWalletAssets)
		balances.POST("", UpdateWalletBalance)
	}

	// Transaction routes
	transactions := router.Group("/api/v1/transactions")
	{
		transactions.POST("", CreateTransaction)
		transactions.GET("", GetTransactions)
		transactions.GET("/:hash", GetTransaction)
		transactions.PUT("/:hash/status", UpdateTransactionStatus)
	}

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
