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

	if err := ensureSchema(context.Background(), db); err != nil {
		log.Fatalf("Failed to ensure schema: %v", err)
	}
	h := &handlers{db: db}

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
		permissions.Use(h.authMiddleware())
		{
			permissions.GET("", h.getPermissionsHandler)
			permissions.GET("/:product", h.getProductPermissionsHandler)
			permissions.PUT("/:product", h.updatePermissionsHandler)
			permissions.POST("/sync", h.syncPermissionsHandler)
			permissions.GET("/cache", h.getCachedPermissionsHandler)
		}

		// Product registration
		products := api.Group("/products")
		products.Use(h.authMiddleware())
		{
			products.POST("/register", h.registerProductHandler)
			products.GET("", h.listProductsHandler)
			products.PUT("/:product_id/status", h.updateProductStatusHandler)
		}

		// Super Admin endpoints
		superAdmin := api.Group("/super-admin")
		superAdmin.Use(superAdminMiddleware(cfg))
		{
			superAdmin.GET("/tenants/:tenant_id/permissions", h.getTenantPermissionsHandler)
			superAdmin.PUT("/tenants/:tenant_id/permissions", h.updateTenantPermissionsHandler)
			superAdmin.POST("/tenants/:tenant_id/products/:product/enable", h.enableProductHandler)
			superAdmin.POST("/tenants/:tenant_id/products/:product/disable", h.disableProductHandler)
			superAdmin.POST("/tenants/:tenant_id/sync-all", h.syncAllPermissionsHandler)
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
	Port          string
	Database      DatabaseConfig
	SuperAdminURL string
	CacheTTL      time.Duration
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
		CacheTTL:      getDurationEnv("CACHE_TTL", 5*time.Minute),
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

const bridgeSchema = `
CREATE TABLE IF NOT EXISTS pb_products (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    endpoint TEXT NOT NULL DEFAULT '',
    api_key TEXT NOT NULL UNIQUE,
    api_secret TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'enabled',
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);
CREATE TABLE IF NOT EXISTS pb_permissions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    product TEXT NOT NULL,
    feature TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    perm_limit BIGINT,
    scope TEXT NOT NULL DEFAULT 'read',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product, feature)
);
CREATE INDEX IF NOT EXISTS idx_pb_permissions_tenant ON pb_permissions(tenant_id, product);
`

func ensureSchema(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, bridgeSchema)
	return err
}

// ============== Models ==============

// ============== Handlers ==============

type handlers struct {
	db *pgxpool.Pool
}

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

func (h *handlers) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID required"})
			c.Abort()
			return
		}
		tid, err := uuid.Parse(tenantID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant id"})
			c.Abort()
			return
		}
		// Real credential check: the presented API key must belong to an
		// enabled product registered by this tenant. Fail-closed.
		var ok bool
		err = h.db.QueryRow(c.Request.Context(),
			`SELECT EXISTS(
			   SELECT 1 FROM pb_products
			   WHERE tenant_id = $1 AND api_key = $2 AND status = 'enabled')`,
			tid, apiKey).Scan(&ok)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "auth check failed"})
			c.Abort()
			return
		}
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key for tenant"})
			c.Abort()
			return
		}
		c.Set("tenant_id", tenantID)
		c.Set("api_key", apiKey)
		c.Next()
	}
}

func superAdminMiddleware(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Super-admin endpoints require the platform shared secret
		// (SUPER_ADMIN_SECRET) as a Bearer token. Fail-closed: if no secret
		// is configured the route is disabled rather than left open.
		secret := os.Getenv("SUPER_ADMIN_SECRET")
		if secret == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "super admin access not configured"})
			c.Abort()
			return
		}
		auth := c.GetHeader("Authorization")
		token := ""
		if len(auth) > 7 && auth[:7] == "Bearer " {
			token = auth[7:]
		}
		if token == "" || !hmac.Equal([]byte(token), []byte(secret)) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid super admin credentials"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *handlers) getPermissionsHandler(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	product := c.Query("product")

	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	ctx := c.Request.Context()
	query := `SELECT product, feature, enabled, perm_limit, scope, updated_at FROM pb_permissions WHERE tenant_id = $1`
	args := []interface{}{tid}
	if product != "" {
		query += ` AND product = $2`
		args = append(args, product)
	}
	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load permissions"})
		return
	}
	defer rows.Close()

	perms := map[string]bool{}
	limits := map[string]interface{}{}
	var updatedAt time.Time
	for rows.Next() {
		var prod, feature, scope string
		var enabled bool
		var limit *int64
		var ts time.Time
		if err := rows.Scan(&prod, &feature, &enabled, &limit, &scope, &ts); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read permissions"})
			return
		}
		perms[prod+"."+feature] = enabled
		if limit != nil {
			limits[prod+"."+feature] = *limit
		}
		if ts.After(updatedAt) {
			updatedAt = ts
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id":   tenantID,
		"product":     product,
		"permissions": perms,
		"limits":      limits,
		"updated_at":  updatedAt.Unix(),
	})
}

func (h *handlers) getProductPermissionsHandler(c *gin.Context) {
	product := c.Param("product")
	tenantID := c.GetHeader("X-Tenant-ID")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT feature, enabled, scope FROM pb_permissions WHERE tenant_id = $1 AND product = $2 ORDER BY feature`,
		tid, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load permissions"})
		return
	}
	defer rows.Close()
	features := []map[string]interface{}{}
	for rows.Next() {
		var feature, scope string
		var enabled bool
		if err := rows.Scan(&feature, &enabled, &scope); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read permissions"})
			return
		}
		features = append(features, map[string]interface{}{"feature": feature, "enabled": enabled, "scope": scope})
	}
	c.JSON(http.StatusOK, gin.H{"product": product, "tenant_id": tenantID, "features": features})
}

func (h *handlers) updatePermissionsHandler(c *gin.Context) {
	product := c.Param("product")
	tenantID := c.GetHeader("X-Tenant-ID")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	var req struct {
		Features []struct {
			Feature string `json:"feature" binding:"required"`
			Enabled bool   `json:"enabled"`
			Scope   string `json:"scope"`
			Limit   *int64 `json:"limit"`
		} `json:"features" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	tx, err := h.db.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback(ctx)
	for _, f := range req.Features {
		scope := f.Scope
		if scope == "" {
			scope = "read"
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO pb_permissions (id, tenant_id, product, feature, enabled, perm_limit, scope, updated_at)
                         VALUES ($1, $2, $3, $4, $5, $6, $7, now())
                         ON CONFLICT (tenant_id, product, feature)
                         DO UPDATE SET enabled = EXCLUDED.enabled, perm_limit = EXCLUDED.perm_limit, scope = EXCLUDED.scope, updated_at = now()`,
			uuid.New(), tid, product, f.Feature, f.Enabled, f.Limit, scope); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update permission " + f.Feature})
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit permissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": product, "updated": len(req.Features)})
}

func (h *handlers) syncPermissionsHandler(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	res, err := h.db.Exec(c.Request.Context(),
		`UPDATE pb_products SET last_sync_at = now(), updated_at = now() WHERE tenant_id = $1`, tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync permissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tenant_id":       tenantID,
		"synced":          true,
		"products_synced": res.RowsAffected(),
		"timestamp":       time.Now().Unix(),
	})
}

func (h *handlers) getCachedPermissionsHandler(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	product := c.Query("product")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	// The DB is the source of truth; "cache" reads only rows refreshed
	// within the last 5 minutes so callers can distinguish warm data.
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT feature, enabled FROM pb_permissions
                 WHERE tenant_id = $1 AND ($2 = '' OR product = $2)
                   AND updated_at > now() - interval '5 minutes'`,
		tid, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load cached permissions"})
		return
	}
	defer rows.Close()
	perms := map[string]bool{}
	for rows.Next() {
		var feature string
		var enabled bool
		if err := rows.Scan(&feature, &enabled); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read cache"})
			return
		}
		perms[feature] = enabled
	}
	c.JSON(http.StatusOK, gin.H{
		"tenant_id":   tenantID,
		"product":     product,
		"permissions": perms,
		"expires_at":  time.Now().Add(5 * time.Minute).Unix(),
	})
}

func (h *handlers) registerProductHandler(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Endpoint string `json:"endpoint" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tenantID := c.GetHeader("X-Tenant-ID")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	id := uuid.New()
	apiKey := uuid.New().String()
	apiSecret := generateSecret()
	var createdAt time.Time
	err = h.db.QueryRow(c.Request.Context(),
		`INSERT INTO pb_products (id, tenant_id, name, endpoint, api_key, api_secret)
                 VALUES ($1, $2, $3, $4, $5, $6)
                 ON CONFLICT (tenant_id, name) DO UPDATE
                   SET endpoint = EXCLUDED.endpoint, updated_at = now()
                 RETURNING created_at`,
		id, tid, req.Name, req.Endpoint, apiKey, apiSecret).Scan(&createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register product"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"product": map[string]interface{}{
			"id": id.String(), "tenant_id": tenantID, "name": req.Name,
			"endpoint": req.Endpoint, "status": "enabled", "created_at": createdAt.Unix(),
		},
		"credentials": map[string]string{"api_key": apiKey, "api_secret": apiSecret},
	})
}

func (h *handlers) listProductsHandler(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT id, name, status, endpoint, last_sync_at FROM pb_products WHERE tenant_id = $1 ORDER BY name`, tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}
	defer rows.Close()
	products := []map[string]interface{}{}
	for rows.Next() {
		var id uuid.UUID
		var name, status, endpoint string
		var lastSync *time.Time
		if err := rows.Scan(&id, &name, &status, &endpoint, &lastSync); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read products"})
			return
		}
		p := map[string]interface{}{"id": id.String(), "name": name, "status": status, "endpoint": endpoint}
		if lastSync != nil {
			p["synced_at"] = lastSync.Unix()
		}
		products = append(products, p)
	}
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "products": products})
}

func (h *handlers) updateProductStatusHandler(c *gin.Context) {
	productID := c.Param("product_id")
	pid, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product_id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=enabled disabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.db.Exec(c.Request.Context(),
		`UPDATE pb_products SET status = $2, updated_at = now() WHERE id = $1`, pid, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product status"})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_id": productID, "status": req.Status})
}

func (h *handlers) getTenantPermissionsHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT product, feature, enabled FROM pb_permissions WHERE tenant_id = $1 ORDER BY product, feature`, tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tenant permissions"})
		return
	}
	defer rows.Close()
	byProduct := map[string][]string{}
	anyEnabled := map[string]bool{}
	for rows.Next() {
		var prod, feature string
		var en bool
		if err := rows.Scan(&prod, &feature, &en); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read tenant permissions"})
			return
		}
		byProduct[prod] = append(byProduct[prod], feature)
		anyEnabled[prod] = anyEnabled[prod] || en
	}
	products := []map[string]interface{}{}
	for prod, feats := range byProduct {
		products = append(products, map[string]interface{}{
			"name": prod, "enabled": anyEnabled[prod], "permissions": feats,
		})
	}
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "products": products})
}

func (h *handlers) updateTenantPermissionsHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	var req struct {
		Products []struct {
			Name        string   `json:"name" binding:"required"`
			Enabled     bool     `json:"enabled"`
			Permissions []string `json:"permissions"`
		} `json:"products" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	tx, err := h.db.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback(ctx)
	for _, prod := range req.Products {
		for _, feat := range prod.Permissions {
			if _, err := tx.Exec(ctx,
				`INSERT INTO pb_permissions (id, tenant_id, product, feature, enabled, updated_at)
                                 VALUES ($1, $2, $3, $4, $5, now())
                                 ON CONFLICT (tenant_id, product, feature)
                                 DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = now()`,
				uuid.New(), tid, prod.Name, feat, prod.Enabled); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update " + prod.Name})
				return
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit tenant permissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "updated": len(req.Products)})
}

func (h *handlers) enableProductHandler(c *gin.Context) {
	h.setProductStatus(c, "enabled")
}

func (h *handlers) disableProductHandler(c *gin.Context) {
	h.setProductStatus(c, "disabled")
}

func (h *handlers) syncAllPermissionsHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(),
		`UPDATE pb_products SET last_sync_at = now(), updated_at = now() WHERE tenant_id = $1 RETURNING name`, tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync products"})
		return
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err == nil {
			names = append(names, n)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID, "synced": true,
		"timestamp": time.Now().Unix(), "products": names,
	})
}

func (h *handlers) setProductStatus(c *gin.Context, status string) {
	tenantID := c.Param("tenant_id")
	product := c.Param("product")
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}
	res, err := h.db.Exec(c.Request.Context(),
		`UPDATE pb_products SET status = $3, updated_at = now() WHERE tenant_id = $1 AND name = $2`,
		tid, product, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found for tenant"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "product": product, "status": status})
}

// ============== Helpers ==============

func generateSecret() string {
	h := sha256.New()
	h.Write([]byte(time.Now().String() + uuid.New().String()))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

func validateSignature(secret, message, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
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
