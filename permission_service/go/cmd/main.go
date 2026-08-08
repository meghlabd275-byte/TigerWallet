// Permission Service - Go Implementation
// High-performance, distributed permission management for TigerWallet ecosystem

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

// ============ DATA MODELS ============

// WhiteLevelProduct types
type WhiteLevelProduct string

const (
	ProductMasterWallet      WhiteLevelProduct = "master_wallet"
	ProductUserWallet       WhiteLevelProduct = "user_wallet"
	ProductBots            WhiteLevelProduct = "bots"
	ProductBotsClients     WhiteLevelProduct = "bots_clients"
	ProductProjectParty    WhiteLevelProduct = "project_party"
)

// Permission levels
type PermissionLevel string

const (
	PermNone       PermissionLevel = "none"
	PermRead       PermissionLevel = "read"
	PermWrite      PermissionLevel = "write"
	PermExecute    PermissionLevel = "execute"
	PermAdmin      PermissionLevel = "admin"
	PermSuperAdmin PermissionLevel = "super_admin"
)

// Fetcher types
type FetcherType string

const (
	FetcherPrices        FetcherType = "prices"
	FetcherBalances      FetcherType = "balances"
	FetcherTransactions  FetcherType = "transactions"
	FetcherUserData      FetcherType = "user_data"
	FetcherMarketData    FetcherType = "market_data"
	FetcherBlockchain    FetcherType = "blockchain"
	FetcherTokenInfo     FetcherType = "token_info"
	FetcherKYC           FetcherType = "kyc"
)

// Core models
type WhiteLevelClient struct {
	ID                uuid.UUID         `json:"id"`
	Name              string            `json:"name"`
	Domain            string            `json:"domain"`
	Products          []WhiteLevelProduct `json:"products"`
	Status            string            `json:"status"` // active, suspended, terminated
	APIKey            string            `json:"api_key"`
	APIKeyHash        string            `json:"-"`
	APIKeyPrefix      string            `json:"api_key_prefix"`
	RateLimit         int               `json:"rate_limit"` // requests per minute
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type WhiteLevelAdmin struct {
	ID             uuid.UUID       `json:"id"`
	ClientID       uuid.UUID       `json:"client_id"`
	Email          string          `json:"email"`
	Username       string          `json:"username"`
	PasswordHash   string          `json:"-"`
	Role           string          `json:"role"` // owner, admin, manager, viewer
	Products       []WhiteLevelProduct `json:"products"`
	Permissions    map[string]map[PermissionLevel]bool `json:"permissions"`
	IsActive       bool            `json:"is_active"`
	TwoFactorEnabled bool          `json:"two_factor_enabled"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	LastLogin      *time.Time      `json:"last_login"`
}

type ProductPermission struct {
	ID             uuid.UUID         `json:"id"`
	ClientID       uuid.UUID         `json:"client_id"`
	Product        WhiteLevelProduct `json:"product"`
	Fetcher        FetcherType       `json:"fetcher"`
	Permission     PermissionLevel   `json:"permission"`
	IsEnabled      bool              `json:"is_enabled"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type FetcherConfig struct {
	ID            uuid.UUID   `json:"id"`
	Product       WhiteLevelProduct `json:"product"`
	Fetcher       FetcherType `json:"fetcher"`
	Endpoint      string     `json:"endpoint"`
	Timeout       int        `json:"timeout"` // milliseconds
	RetryCount    int        `json:"retry_count"`
	CacheTTL      int        `json:"cache_ttl"` // seconds
	IsActive     bool       `json:"is_active"`
	ConfigData   string     `json:"config_data"` // JSON
}

type PermissionAudit struct {
	ID           uuid.UUID `json:"id"`
	AdminID      uuid.UUID `json:"admin_id"`
	ClientID     uuid.UUID `json:"client_id"`
	Action       string    `json:"action"` // grant, revoke, modify
	ResourceType string    `json:"resource_type"` // product, fetcher, admin
	ResourceID   string    `json:"resource_id"`
	Details      string    `json:"details"` // JSON
	IPAddress    string    `json:"ip_address"`
	Timestamp    time.Time `json:"timestamp"`
}

type APIConnection struct {
	ID            uuid.UUID `json:"id"`
	ClientID      uuid.UUID `json:"client_id"`
	Product       WhiteLevelProduct `json:"product"`
	ConnectionKey string    `json:"connection_key"`
	Status        string    `json:"status"` // connected, disconnected, error
	LastHeartbeat time.Time `json:"last_heartbeat"`
	IPAddress     string    `json:"ip_address"`
	CreatedAt     time.Time `json:"created_at"`
}

// Global variables
var (
	db             *pgxpool.Pool
	redis          *redis.Client
	config         Config
	logger         *log.Logger
	permCache      *sync.Map // permission cache
	connManager    *sync.Map // connection manager
	jwtSecret      []byte
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
		CREATE TABLE IF NOT EXISTS white_level_clients (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			domain VARCHAR(255),
			products JSONB NOT NULL,
			status VARCHAR(50) DEFAULT 'active',
			api_key VARCHAR(255) UNIQUE,
			api_key_hash VARCHAR(255),
			api_key_prefix VARCHAR(20),
			rate_limit INTEGER DEFAULT 100,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS white_level_admins (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			client_id UUID REFERENCES white_level_clients(id),
			email VARCHAR(255) NOT NULL,
			username VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(50) DEFAULT 'admin',
			products JSONB NOT NULL,
			permissions JSONB,
			is_active BOOLEAN DEFAULT true,
			two_factor_enabled BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			last_login TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS product_permissions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			client_id UUID REFERENCES white_level_clients(id),
			product VARCHAR(50) NOT NULL,
			fetcher VARCHAR(50) NOT NULL,
			permission VARCHAR(50) NOT NULL,
			is_enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(client_id, product, fetcher)
		);

		CREATE TABLE IF NOT EXISTS fetcher_configs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			product VARCHAR(50) NOT NULL,
			fetcher VARCHAR(50) NOT NULL,
			endpoint VARCHAR(512) NOT NULL,
			timeout INTEGER DEFAULT 5000,
			retry_count INTEGER DEFAULT 3,
			cache_ttl INTEGER DEFAULT 60,
			is_active BOOLEAN DEFAULT true,
			config_data JSONB,
			UNIQUE(product, fetcher)
		);

		CREATE TABLE IF NOT EXISTS permission_audits (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			admin_id UUID,
			client_id UUID,
			action VARCHAR(50) NOT NULL,
			resource_type VARCHAR(50) NOT NULL,
			resource_id VARCHAR(255),
			details JSONB,
			ip_address VARCHAR(45),
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS api_connections (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			client_id UUID REFERENCES white_level_clients(id),
			product VARCHAR(50) NOT NULL,
			connection_key VARCHAR(255) UNIQUE,
			status VARCHAR(50) DEFAULT 'connected',
			last_heartbeat TIMESTAMP,
			ip_address VARCHAR(45),
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_clients_api_key ON white_level_clients(api_key);
		CREATE INDEX IF NOT EXISTS idx_admins_client ON white_level_admins(client_id);
		CREATE INDEX IF NOT EXISTS idx_permissions_client ON product_permissions(client_id);
		CREATE INDEX IF NOT EXISTS idx_audits_timestamp ON permission_audits(timestamp);
		CREATE INDEX IF NOT EXISTS idx_connections_key ON api_connections(connection_key);
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

// ============ PERMISSION LOGIC ============

// CheckPermission checks if a client has specific permission
func CheckPermission(clientID uuid.UUID, product WhiteLevelProduct, fetcher FetcherType, required PermLevel) bool {
	// Check cache first
	cacheKey := fmt.Sprintf("perm:%s:%s:%s", clientID.String(), product, fetcher)
	if cached, ok := permCache.Load(cacheKey); ok {
		return cached.(bool)
	}

	// Check database
	var hasPerm bool
	err := db.QueryRow(context.Background(), `
		SELECT is_enabled FROM product_permissions 
		WHERE client_id = $1 AND product = $2 AND fetcher = $3 AND is_enabled = true
	`, clientID, product, fetcher).Scan(&hasPerm)

	if err != nil {
		hasPerm = false
	}

	// Cache result
	permCache.Store(cacheKey, hasPerm)
	return hasPerm
}

// GrantPermission grants a specific permission
func GrantPermission(clientID uuid.UUID, product WhiteLevelProduct, fetcher FetcherType, level PermLevel) error {
	_, err := db.Exec(context.Background(), `
		INSERT INTO product_permissions (id, client_id, product, fetcher, permission, is_enabled)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (client_id, product, fetcher) DO UPDATE SET permission = $5, is_enabled = true, updated_at = NOW()
	`, uuid.New(), clientID, product, fetcher, level)

	// Clear cache
	cacheKey := fmt.Sprintf("perm:%s:%s:%s", clientID.String(), product, fetcher)
	permCache.Delete(cacheKey)

	return err
}

// RevokePermission revokes a specific permission
func RevokePermission(clientID uuid.UUID, product WhiteLevelProduct, fetcher FetcherType) error {
	_, err := db.Exec(context.Background(), `
		UPDATE product_permissions SET is_enabled = false, updated_at = NOW()
		WHERE client_id = $1 AND product = $2 AND fetcher = $3
	`, clientID, product, fetcher)

	// Clear cache
	cacheKey := fmt.Sprintf("perm:%s:%s:%s", clientID.String(), product, fetcher)
	permCache.Delete(cacheKey)

	return err
}

// GetAllPermissions gets all permissions for a client
func GetAllPermissions(clientID uuid.UUID) ([]ProductPermission, error) {
	rows, err := db.Query(context.Background(), `
		SELECT id, client_id, product, fetcher, permission, is_enabled, created_at, updated_at
		FROM product_permissions WHERE client_id = $1
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []ProductPermission
	for rows.Next() {
		var p ProductPermission
		if err := rows.Scan(&p.ID, &p.ClientID, &p.Product, &p.Fetcher, &p.Permission, &p.IsEnabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		perms = append(perms, p)
	}
	return perms, nil
}

// ============ AUTHENTICATION ============

func generateAPIKey() (string, string) {
	key := uuid.New().String() + uuid.New().String()
	prefix := "tw_" + key[:8]
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])
	return key, keyHash, prefix
}

func generateJWT(adminID, clientID uuid.UUID, role string) (string, error) {
	claims := jwt.MapClaims{
		"admin_id": adminID.String(),
		"client_id": clientID.String(),
		"role": role,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func validateAPIKey(apiKey string) (*WhiteLevelClient, error) {
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	var client WhiteLevelClient
	err := db.QueryRow(context.Background(), `
		SELECT id, name, domain, products, status, rate_limit, created_at, updated_at
		FROM white_level_clients WHERE api_key_hash = $1 AND status = 'active'
	`, keyHash).Scan(&client.ID, &client.Name, &client.Domain, &client.Products, &client.Status, &client.RateLimit, &client.CreatedAt, &client.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &client, nil
}

// ============ CONNECTION MANAGEMENT ============

func registerConnection(clientID uuid.UUID, product WhiteLevelProduct, ipAddress string) (*APIConnection, error) {
	connKey := uuid.New().String()
	
	conn := &APIConnection{
		ID:            uuid.New(),
		ClientID:      clientID,
		Product:        product,
		ConnectionKey: connKey,
		Status:        "connected",
		LastHeartbeat: time.Now(),
		IPAddress:     ipAddress,
		CreatedAt:     time.Now(),
	}

	_, err := db.Exec(context.Background(), `
		INSERT INTO api_connections (id, client_id, product, connection_key, status, last_heartbeat, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, conn.ID, conn.ClientID, conn.Product, conn.ConnectionKey, conn.Status, conn.LastHeartbeat, conn.IPAddress, conn.CreatedAt)

	if err != nil {
		return nil, err
	}

	connManager.Store(connKey, conn)
	return conn, nil
}

func heartbeatConnection(connKey string) error {
	connManager.Store(connKey, time.Now())
	
	_, err := db.Exec(context.Background(), `
		UPDATE api_connections SET last_heartbeat = NOW() WHERE connection_key = $1
	`, connKey)
	return err
}

func disconnectConnection(connKey string) error {
	connManager.Delete(connKey)
	
	_, err := db.Exec(context.Background(), `
		UPDATE api_connections SET status = 'disconnected' WHERE connection_key = $1
	`, connKey)
	return err
}

// ============ AUDIT LOGGING ============

func logAudit(adminID, clientID uuid.UUID, action, resourceType, resourceID, details, ipAddress string) error {
	detailsJSON, _ := json.Marshal(details)
	
	_, err := db.Exec(context.Background(), `
		INSERT INTO permission_audits (id, admin_id, client_id, action, resource_type, resource_id, details, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.New(), adminID, clientID, action, resourceType, resourceID, detailsJSON, ipAddress)
	return err
}

// ============ HTTP HANDLERS ============

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
		"status":   "ok",
		"database": dbStatus,
		"redis":    redisStatus,
		"timestamp": time.Now(),
	})
}

// RegisterClient - Register a new white level client
func RegisterClient(c *gin.Context) {
	var req struct {
		Name     string               `json:"name" binding:"required"`
		Domain   string               `json:"domain"`
		Products []WhiteLevelProduct `json:"products" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, keyHash, prefix := generateAPIKey()
	
	client := WhiteLevelClient{
		ID:           uuid.New(),
		Name:         req.Name,
		Domain:        req.Domain,
		Products:       req.Products,
		Status:        "active",
		APIKey:        apiKey,
		APIKeyHash:    keyHash,
		APIKeyPrefix:  prefix,
		RateLimit:     100,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	productsJSON, _ := json.Marshal(client.Products)

	_, err := db.Exec(context.Background(), `
		INSERT INTO white_level_clients (id, name, domain, products, status, api_key, api_key_hash, api_key_prefix, rate_limit, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, client.ID, client.Name, client.Domain, productsJSON, client.Status, client.APIKey, client.APIKeyHash, client.APIKeyPrefix, client.RateLimit, client.CreatedAt, client.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Initialize default permissions for each product
	for _, product := range client.Products {
		for _, fetcher := range getProductFetchers(product) {
			GrantPermission(client.ID, product, fetcher, PermAdmin)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"client": gin.H{
			"id":           client.ID,
			"name":         client.Name,
			"domain":       client.Domain,
			"products":      client.Products,
			"status":       client.Status,
			"api_key":      client.APIKey,
			"rate_limit":   client.RateLimit,
		},
	})
}

// GetClient - Get client by API key
func GetClient(c *gin.Context) {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
		return
	}

	client, err := validateAPIKey(apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}

	perms, _ := GetAllPermissions(client.ID)

	c.JSON(http.StatusOK, gin.H{
		"client":      client,
		"permissions": perms,
	})
}

// CreateAdmin - Create admin for client
func CreateAdmin(c *gin.Context) {
	var req struct {
		ClientID uuid.UUID       `json:"client_id" binding:"required"`
		Email    string          `json:"email" binding:"required"`
		Username string          `json:"username" binding:"required"`
		Password string          `json:"password" binding:"required"`
		Role     string          `json:"role"`
		Products []WhiteLevelProduct `json:"products"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role == "" {
		req.Role = "admin"
	}

	// Hash password
	hash := sha256.Sum256([]byte(req.Password))
	passwordHash := hex.EncodeToString(hash[:])

	admin := WhiteLevelAdmin{
		ID:               uuid.New(),
		ClientID:         req.ClientID,
		Email:            req.Email,
		Username:         req.Username,
		PasswordHash:     passwordHash,
		Role:             req.Role,
		Products:         req.Products,
		Permissions:      make(map[string]map[PermissionLevel]bool),
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	productsJSON, _ := json.Marshal(admin.Products)
	permsJSON, _ := json.Marshal(admin.Permissions)

	_, err := db.Exec(context.Background(), `
		INSERT INTO white_level_admins (id, client_id, email, username, password_hash, role, products, permissions, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, admin.ID, admin.ClientID, admin.Email, admin.Username, admin.PasswordHash, admin.Role, productsJSON, permsJSON, admin.IsActive, admin.CreatedAt, admin.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(admin.ID, req.ClientID, "create", "admin", admin.ID.String(), "{}", c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{"admin_id": admin.ID})
}

// Login - Admin login
func Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash := sha256.Sum256([]byte(req.Password))
	passwordHash := hex.EncodeToString(hash[:])

	var admin WhiteLevelAdmin
	err := db.QueryRow(context.Background(), `
		SELECT id, client_id, email, username, role, products, permissions, is_active, two_factor_enabled
		FROM white_level_admins WHERE email = $1 AND password_hash = $2 AND is_active = true
	`, req.Email, passwordHash).Scan(&admin.ID, &admin.ClientID, &admin.Email, &admin.Username, &admin.Role, &admin.Products, &admin.Permissions, &admin.IsActive, &admin.TwoFactorEnabled)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, _ := generateJWT(admin.ID, admin.ClientID, admin.Role)

	// Update last login
	db.Exec(context.Background(), "UPDATE white_level_admins SET last_login = NOW() WHERE id = $1", admin.ID)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"admin": gin.H{
			"id":      admin.ID,
			"email":   admin.Email,
			"username": admin.Username,
			"role":    admin.Role,
		},
	})
}

// GrantPermissionHandler - Grant permission to client
func GrantPermissionHandler(c *gin.Context) {
	var req struct {
		ClientID uuid.UUID         `json:"client_id" binding:"required"`
		Product  WhiteLevelProduct `json:"product" binding:"required"`
		Fetcher  FetcherType       `json:"fetcher" binding:"required"`
		Level    PermLevel         `json:"level" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := GrantPermission(req.ClientID, req.Product, req.Fetcher, req.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(uuid.Nil, req.ClientID, "grant", "permission", fmt.Sprintf("%s:%s", req.Product, req.Fetcher), "{}", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Permission granted"})
}

// RevokePermissionHandler - Revoke permission from client
func RevokePermissionHandler(c *gin.Context) {
	var req struct {
		ClientID uuid.UUID         `json:"client_id" binding:"required"`
		Product  WhiteLevelProduct `json:"product" binding:"required"`
		Fetcher  FetcherType       `json:"fetcher" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := RevokePermission(req.ClientID, req.Product, req.Fetcher)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(uuid.Nil, req.ClientID, "revoke", "permission", fmt.Sprintf("%s:%s", req.Product, req.Fetcher), "{}", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Permission revoked"})
}

// GetPermissions - Get all permissions for client
func GetPermissions(c *gin.Context) {
	clientID := c.Param("client_id")
	id, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	perms, err := GetAllPermissions(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"permissions": perms})
}

// CheckPermissionHandler - Check if client has permission
func CheckPermissionHandler(c *gin.Context) {
	var req struct {
		ClientID uuid.UUID         `json:"client_id" binding:"required"`
		Product  WhiteLevelProduct `json:"product" binding:"required"`
		Fetcher  FetcherType       `json:"fetcher" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hasPerm := CheckPermission(req.ClientID, req.Product, req.Fetcher, PermRead)

	c.JSON(http.StatusOK, gin.H{"has_permission": hasPerm})
}

// RegisterConnection - Register product connection
func RegisterConnection(c *gin.Context) {
	var req struct {
		Product   WhiteLevelProduct `json:"product" binding:"required"`
		IPAddress string             `json:"ip_address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey := c.GetHeader("X-API-Key")
	client, err := validateAPIKey(apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}

	ipAddr := c.ClientIP()
	if req.IPAddress != "" {
		ipAddr = req.IPAddress
	}

	conn, err := registerConnection(client.ID, req.Product, ipAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"connection_key": conn.ConnectionKey,
		"status":        conn.Status,
	})
}

// Heartbeat - Connection heartbeat
func Heartbeat(c *gin.Context) {
	var req struct {
		ConnectionKey string `json:"connection_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := heartbeatConnection(req.ConnectionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "heartbeat received"})
}

// Disconnect - Disconnect product
func Disconnect(c *gin.Context) {
	var req struct {
		ConnectionKey string `json:"connection_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := disconnectConnection(req.ConnectionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "disconnected"})
}

// GetAuditLog - Get audit logs
func GetAuditLog(c *gin.Context) {
	clientID := c.Query("client_id")
	limit := c.DefaultQuery("limit", "100")

	query := `
		SELECT id, admin_id, client_id, action, resource_type, resource_id, details, ip_address, timestamp
		FROM permission_audits
		WHERE 1=1
	`
	if clientID != "" {
		query += fmt.Sprintf(" AND client_id = '%s'", clientID)
	}
	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT %s", limit)

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logs []PermissionAudit
	for rows.Next() {
		var log PermissionAudit
		if err := rows.Scan(&log.ID, &log.AdminID, &log.ClientID, &log.Action, &log.ResourceType, &log.ResourceID, &log.Details, &log.IPAddress, &log.Timestamp); err != nil {
			continue
		}
		logs = append(logs, log)
	}

	c.JSON(http.StatusOK, gin.H{"audit_logs": logs})
}

// GetClients - Get all clients
func GetClients(c *gin.Context) {
	rows, err := db.Query(context.Background(), `
		SELECT id, name, domain, products, status, rate_limit, created_at, updated_at
		FROM white_level_clients ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var clients []WhiteLevelClient
	for rows.Next() {
		var client WhiteLevelClient
		var products []byte
		if err := rows.Scan(&client.ID, &client.Name, &client.Domain, &products, &client.Status, &client.RateLimit, &client.CreatedAt, &client.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal(products, &client.Products)
		client.APIKey = "" // Don't expose API key
		clients = append(clients, client)
	}

	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

// UpdateClientStatus - Update client status (suspend/activate)
func UpdateClientStatus(c *gin.Context) {
	clientID := c.Param("client_id")
	id, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validStatuses := []string{"active", "suspended", "terminated"}
	if !contains(validStatuses, req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	_, err = db.Exec(context.Background(), `
		UPDATE white_level_clients SET status = $1, updated_at = NOW() WHERE id = $2
	`, req.Status, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(uuid.Nil, id, "update", "client_status", req.Status, "{}", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// ============ HELPER FUNCTIONS ============

func getProductFetchers(product WhiteLevelProduct) []FetcherType {
	switch product {
	case ProductMasterWallet:
		return []FetcherType{FetcherPrices, FetcherBalances, FetcherTransactions, FetcherUserData, FetcherMarketData, FetcherBlockchain, FetcherTokenInfo}
	case ProductUserWallet:
		return []FetcherType{FetcherPrices, FetcherBalances, FetcherTransactions, FetcherUserData}
	case ProductBots, ProductBotsClients:
		return []FetcherType{FetcherPrices, FetcherMarketData, FetcherBlockchain}
	case ProductProjectParty:
		return []FetcherType{FetcherTokenInfo, FetcherMarketData, FetcherBlockchain, FetcherKYC}
	default:
		return []FetcherType{}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// PermLevel alias for PermissionLevel
type PermLevel = PermissionLevel

// ============ MAIN ============

func main() {
	logger = log.New(os.Stdout, "Permission Service: ", log.LstdFlags)
	logger.Println("Starting Permission Service...")

	config.Port = getEnv("PERMISSION_PORT", "8091")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")
	config.JWTSecret = getEnv("JWT_SECRET", "")
	jwtSecret = []byte(config.JWTSecret)

	permCache = &sync.Map{}
	connManager = &sync.Map{}

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
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/health", HealthCheck)

	// Client management
	router.POST("/api/v1/clients", RegisterClient)
	router.GET("/api/v1/clients", GetClient)
	router.GET("/api/v1/admin/clients", GetClients)
	router.PUT("/api/v1/admin/clients/:client_id/status", UpdateClientStatus)

	// Admin management
	router.POST("/api/v1/admins", CreateAdmin)
	router.POST("/api/v1/auth/login", Login)

	// Permission management
	router.POST("/api/v1/permissions/grant", GrantPermissionHandler)
	router.POST("/api/v1/permissions/revoke", RevokePermissionHandler)
	router.GET("/api/v1/permissions/:client_id", GetPermissions)
	router.POST("/api/v1/permissions/check", CheckPermissionHandler)

	// Connection management
	router.POST("/api/v1/connections", RegisterConnection)
	router.POST("/api/v1/connections/heartbeat", Heartbeat)
	router.POST("/api/v1/connections/disconnect", Disconnect)

	// Audit
	router.GET("/api/v1/audit", GetAuditLog)

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
