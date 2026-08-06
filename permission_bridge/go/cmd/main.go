package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
)

func main() {
	cfg := loadConfig()

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Gin router
	router := gin.Default()

	// CORS
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-permission-bridge"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Permission endpoints
		permissions := api.Group("/permissions")
		permissions.Use(authMiddleware(cfg))
		{
			permissions.GET("", getPermissionsHandler)
			permissions.GET("/:product", getProductPermissionsHandler)
			permissions.PUT("/:product", updatePermissionsHandler)
			permissions.POST("/sync", syncPermissionsHandler)
			permissions.GET("/cache", getCachedPermissionsHandler)
		}

		// Product registration
		products := api.Group("/products")
		products.Use(authMiddleware(cfg))
		{
			products.POST("/register", registerProductHandler)
			products.GET("", listProductsHandler)
			products.PUT("/:product_id/status", updateProductStatusHandler)
		}

		// Super Admin endpoints
		superAdmin := api.Group("/super-admin")
		superAdmin.Use(superAdminMiddleware(cfg))
		{
			superAdmin.GET("/tenants/:tenant_id/permissions", getTenantPermissionsHandler)
			superAdmin.PUT("/tenants/:tenant_id/permissions", updateTenantPermissionsHandler)
			superAdmin.POST("/tenants/:tenant_id/products/:product/enable", enableProductHandler)
			superAdmin.POST("/tenants/:tenant_id/products/:product/disable", disableProductHandler)
			superAdmin.POST("/tenants/:tenant_id/sync-all", syncAllPermissionsHandler)
		}
	}

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Permission Bridge service starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}

type Config struct {
	Port           string
	Database       DatabaseConfig
	SuperAdminURL  string
	CacheTTL       time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

func loadConfig() *Config {
	return &Config{
		Port:          getEnv("PERMISSION_BRIDGE_PORT", "9007"),
		SuperAdminURL: getEnv("SUPER_ADMIN_URL", "https://super-admin.tigerwallet.com"),
		CacheTTL:       getDurationEnv("CACHE_TTL", 5*time.Minute),
		Database: DatabaseConfig{
			Host:     getEnv("PERMISSION_DB_HOST", "localhost"),
			Port:     getEnvInt("PERMISSION_DB_PORT", 5432),
			User:     getEnv("PERMISSION_DB_USER", "tigerwallet"),
			Password: getEnv("PERMISSION_DB_PASSWORD", "password"),
			DBName:   getEnv("PERMISSION_DB_NAME", "tigerwallet_permissions"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	var value int
	_, err := fmt.Sscan(os.Getenv(key), &value)
	if err != nil {
		return defaultValue
	}
	return value
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// ============== Models ==============

type Tenant struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string   `json:"name" db:"name"`
	Slug        string   `json:"slug" db:"slug"`
	Email       string   `json:"email" db:"email"`
	Status      string   `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Product struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID   uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name       string   `json:"name" db:"name"` // master_wallet, user_wallet, bots, project_party
	Status     string   `json:"status" db:"status"` // enabled, disabled
	APIKey    string   `json:"api_key" db:"api_key"`
	APISecret string   `json:"api_secret" db:"api_secret"`
	Endpoint   string   `json:"endpoint" db:"endpoint"`
	LastSyncAt *time.Time `json:"last_sync_at" db:"last_sync_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type Permission struct {
	ID         uuid.UUID `json:"id" db:"id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Product   string   `json:"product" db:"product"` // master_wallet, user_wallet, bots, project_party
	Feature   string   `json:"feature" db:"feature"` // fetcher.access, wallet.create, bot.create, token.list
	Enabled   bool     `json:"enabled" db:"enabled"`
	Limit     *int64   `json:"limit" db:"limit"`
	Scope     string   `json:"scope" db:"scope"` // read, write, admin
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type PermissionCache struct {
	TenantID   uuid.UUID         `json:"tenant_id"`
	Product    string            `json:"product"`
	Permissions map[string]bool  `json:"permissions"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// ============== Handlers ==============

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Tenant-ID")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func authMiddleware(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// Validate API key
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID required"})
			c.Abort()
			return
		}

		// In production, validate against database
		c.Set("tenant_id", tenantID)
		c.Set("api_key", apiKey)
		c.Next()
	}
}

func superAdminMiddleware(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// In production, validate Super Admin JWT token
		c.Next()
	}
}

func getPermissionsHandler(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	product := c.Query("product")

	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	// Return mock permissions
	permissions := map[string]interface{}{
		"tenant_id": tenantID,
		"product":   product,
		"permissions": map[string]bool{
			"wallet.create":            true,
			"wallet.send":              true,
			"wallet.receive":           true,
			"wallet.swap":             true,
			"wallet.stake":             true,
			"bot.create":              true,
			"bot.start":               true,
			"bot.stop":                true,
			"bot.configure":           true,
			"token.list":              true,
			"token.create":            true,
			"launchpad.create":        true,
			"fetcher.prices":          true,
			"fetcher.blockchain":      true,
			"fetcher.wallet":          true,
		},
		"limits": map[string]interface{}{
			"max_wallets":     100,
			"max_bots":        50,
			"max_api_calls":   100000,
			"max_transactions": 10000,
		},
		"updated_at": time.Now().Unix(),
	}

	c.JSON(http.StatusOK, permissions)
}

func getProductPermissionsHandler(c *gin.Context) {
	product := c.Param("product")
	tenantID := c.GetHeader("X-Tenant-ID")

	permissions := map[string]interface{}{
		"product": product,
		"features": []string{
			"wallet.create",
			"wallet.send",
			"wallet.receive",
			"bot.create",
			"bot.start",
			"token.list",
			"fetcher.prices",
		},
	}

	c.JSON(http.StatusOK, permissions)
}

func updatePermissionsHandler(c *gin.Context) {
	product := c.Param("product")

	var req struct {
		Features []string `json:"features"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{
		"message":  "permissions updated",
		"product":  product,
		"features": req.Features,
	})
}

func syncPermissionsHandler(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")

	// Sync with Super Admin
	permissions := map[string]interface{}{
		"tenant_id":  tenantID,
		"synced":    true,
		"timestamp": time.Now().Unix(),
	}

	c.JSON(http.StatusOK, permissions)
}

func getCachedPermissionsHandler(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	product := c.Query("product")

	// Return cached permissions
	permissions := map[string]interface{}{
		"tenant_id":  tenantID,
		"product":    product,
		"cached":    true,
		"expires_at": time.Now().Add(5 * time.Minute).Unix(),
		"permissions": map[string]bool{
			"wallet.create": true,
			"bot.create":   true,
		},
	}

	c.JSON(http.StatusOK, permissions)
}

func registerProductHandler(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Endpoint string `json:"endpoint" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	tenantID := c.GetHeader("X-Tenant-ID")

	// Generate API credentials
	apiKey := uuid.New().String()
	apiSecret := generateSecret()

	product := map[string]interface{}{
		"id":           uuid.New().String(),
		"tenant_id":    tenantID,
		"name":         req.Name,
		"endpoint":     req.Endpoint,
		"api_key":      apiKey,
		"api_secret":   apiSecret,
		"status":       "enabled",
		"created_at":   time.Now().Unix(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"product":    product,
		"message":   "product registered successfully",
		"credentials": map[string]string{
			"api_key":    apiKey,
			"api_secret": apiSecret,
		},
	})
}

func listProductsHandler(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")

	products := []map[string]interface{}{
		{
			"id":       uuid.New().String(),
			"name":     "master_wallet",
			"status":   "enabled",
			"synced_at": time.Now().Unix(),
		},
		{
			"id":       uuid.New().String(),
			"name":     "user_wallet",
			"status":   "enabled",
			"synced_at": time.Now().Unix(),
		},
		{
			"id":       uuid.New().String(),
			"name":     "bots",
			"status":   "enabled",
			"synced_at": time.Now().Unix(),
		},
		{
			"id":       uuid.New().String(),
			"name":     "project_party",
			"status":   "enabled",
			"synced_at": time.Now().Unix(),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"products":  products,
	})
}

func updateProductStatusHandler(c *gin.Context) {
	productID := c.Param("product_id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{
		"product_id": productID,
		"status":    req.Status,
		"message":   "product status updated",
	})
}

func getTenantPermissionsHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	permissions := map[string]interface{}{
		"tenant_id": tenantID,
		"products": []map[string]interface{}{
			{
				"name":        "master_wallet",
				"enabled":     true,
				"permissions": []string{"wallet.create", "wallet.send", "wallet.receive"},
			},
			{
				"name":        "user_wallet",
				"enabled":     true,
				"permissions": []string{"wallet.create", "wallet.send"},
			},
			{
				"name":        "bots",
				"enabled":     true,
				"permissions": []string{"bot.create", "bot.start"},
			},
			{
				"name":        "project_party",
				"enabled":     true,
				"permissions": []string{"token.list", "launchpad.create"},
			},
		},
	}

	c.JSON(http.StatusOK, permissions)
}

func updateTenantPermissionsHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	var req struct {
		Products []map[string]interface{} `json:"products"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"message":  "permissions updated",
	})
}

func enableProductHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	product := c.Param("product")

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"product":  product,
		"status":   "enabled",
		"message":  "product enabled",
	})
}

func disableProductHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	product := c.Param("product")

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"product":  product,
		"status":   "disabled",
		"message":  "product disabled",
	})
}

func syncAllPermissionsHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	// Sync all permissions for a tenant
	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"synced":   true,
		"timestamp": time.Now().Unix(),
		"products": []string{"master_wallet", "user_wallet", "bots", "project_party"},
	})
}

// ============== Helpers ==============

func generateSecret() string {
	h := sha256.New()
	h.Write([]byte(time.Now().String() + uuid.New().String()))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

func validateSignature(secret, message, signature string) bool {
	mac := hmac.New(sha256.New(), []byte(secret))
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ============== Database ==============

func initDatabase(cfg *Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=require",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return pool, nil
}
