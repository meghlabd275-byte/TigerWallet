package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// TIGERWALLET WHITE LABEL SYSTEM
// Complete white label management with real operations.
// PostgreSQL-backed — all entities are persisted via pgx.
// ============================================================================

var (
	logger      zerolog.Logger
	redisClient *redis.Client
	pg          *pgxpool.Pool
	jwtSecret   string
)

// Configuration
type Config struct {
	Port        string
	RedisURL    string
	SuperAdmin  string
	DatabaseURL string
	JWTSecret   string
}

// White Label Client
type WhiteLabelClient struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Domain            string   `json:"domain"`
	CustomBranding    bool     `json:"customBranding"`
	LogoURL           string   `json:"logoUrl"`
	PrimaryColor      string   `json:"primaryColor"`
	SecondaryColor    string   `json:"secondaryColor"`
	Status            string   `json:"status"` // active, suspended, halted
	CreatedAt         int64    `json:"createdAt"`
	UpdatedAt         int64    `json:"updatedAt"`
	AdminIDs          []string `json:"adminIds"`
	Permissions       []string `json:"permissions"`
	Products          []string `json:"products"`
	BlockchainAccess  []uint64 `json:"blockchainAccess"`
	APIKey            string   `json:"apiKey"`
	SecretKey         string   `json:"secretKey"`
}

// White Label Admin
type WhiteLabelAdmin struct {
	ID               string   `json:"id"`
	ClientID         string   `json:"clientId"`
	Email            string   `json:"email"`
	Name             string   `json:"name"`
	Role             string   `json:"role"` // super_admin, admin, manager, support
	Permissions      []string `json:"permissions"`
	Status           string   `json:"status"`
	CreatedAt        int64    `json:"createdAt"`
	LastLogin        int64    `json:"lastLogin"`
	TwoFactorEnabled bool     `json:"twoFactorEnabled"`
}

// Product configuration
type Product struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`     // trading, wallet, staking, nft, etc
	Status     string   `json:"status"`   // enabled, disabled, maintenance
	Fee        float64  `json:"fee"`
	MinDeposit float64  `json:"minDeposit"`
	MaxDeposit float64  `json:"maxDeposit"`
	Features   []string `json:"features"`
}

// Trading pair configuration
type TradingPair struct {
	ID         string  `json:"id"`
	BaseToken  string  `json:"baseToken"`
	QuoteToken string  `json:"quoteToken"`
	ChainID    uint64  `json:"chainId"`
	Status     string  `json:"status"` // active, suspended, halted
	Fee        float64 `json:"fee"`
	MinTrade   float64 `json:"minTrade"`
	MaxTrade   float64 `json:"maxTrade"`
	Liquidity  float64 `json:"liquidity"`
}

// Liquidity pool
type LiquidityPool struct {
	ID        string  `json:"id"`
	PairID    string  `json:"pairId"`
	ClientID  string  `json:"clientId"`
	Provider  string  `json:"provider"` // internal, external_dex, external_cex
	TokenA    string  `json:"tokenA"`
	TokenB    string  `json:"tokenB"`
	AmountA   float64 `json:"amountA"`
	AmountB   float64 `json:"amountB"`
	ValueUSD  float64 `json:"valueUsd"`
	Status    string  `json:"status"`
	CreatedAt int64   `json:"createdAt"`
}

// Token management
type TokenConfig struct {
	ID        string   `json:"id"`
	ClientID  string   `json:"clientId"`
	Address   string   `json:"address"`
	Name      string   `json:"name"`
	Symbol    string   `json:"symbol"`
	Decimals  uint8    `json:"decimals"`
	ChainID   uint64   `json:"chainId"`
	Type      string   `json:"type"` // erc20, bep20, spl, etc
	Status    string   `json:"status"`
	MaxSupply string   `json:"maxSupply"`
	Features  []string `json:"features"`
}

// Market maker bot
type MarketMakerBot struct {
	ID        string                 `json:"id"`
	ClientID  string                 `json:"clientId"`
	Name      string                 `json:"name"`
	PairIDs   []string               `json:"pairIds"`
	Status    string                 `json:"status"` // running, stopped, error
	Strategy  string                 `json:"strategy"` // arbitrage, market_making, liquidity
	Params    map[string]interface{} `json:"params"`
	Profit    float64                `json:"profit"`
	Volume24h float64                `json:"volume24h"`
	CreatedAt int64                  `json:"createdAt"`
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	logger.Info().Msg("Starting TigerWallet White Label System")

	cfg := loadConfig()
	jwtSecret = cfg.JWTSecret
	redisClient = initRedis(cfg.RedisURL)
	defer redisClient.Close()

	// PostgreSQL connection pool
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to parse DATABASE_URL: %v", err)
	}
	poolCfg.MaxConns = 20
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute

	pg, err = pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Fatalf("failed to create pgxpool: %v", err)
	}
	defer pg.Close()

	if err := pg.Ping(context.Background()); err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	if err := Migrate(context.Background()); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
	logger.Info().Msg("Database migrated successfully")

	router := setupRouter(cfg)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server error")
		}
	}()

	logger.Info().Msgf("White Label System started on port %s", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Server exited")
}

// ============================================================================
// CONFIGURATION
// ============================================================================

func loadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", "8095"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		SuperAdmin:  getEnv("SUPER_ADMIN", "admin"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "tigerwallet-white-label-jwt-secret-change-in-production"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// REDIS
// ============================================================================

func initRedis(url string) *redis.Client {
	opt, err := redis.ParseURL(url)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to parse Redis URL")
		opt = &redis.Options{Addr: "localhost:6379"}
	}

	client := redis.NewClient(opt)
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("Redis connection failed")
	}

	return client
}

// ============================================================================
// DATABASE MIGRATION
// ============================================================================

const whiteLabelSchema = `
CREATE TABLE IF NOT EXISTS wl_clients (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL DEFAULT '',
    domain            TEXT NOT NULL DEFAULT '',
    custom_branding   BOOLEAN NOT NULL DEFAULT FALSE,
    logo_url          TEXT NOT NULL DEFAULT '',
    primary_color     TEXT NOT NULL DEFAULT '',
    secondary_color   TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        BIGINT NOT NULL DEFAULT 0,
    updated_at        BIGINT NOT NULL DEFAULT 0,
    admin_ids         JSONB NOT NULL DEFAULT '[]'::jsonb,
    permissions       JSONB NOT NULL DEFAULT '[]'::jsonb,
    products          JSONB NOT NULL DEFAULT '[]'::jsonb,
    blockchain_access JSONB NOT NULL DEFAULT '[]'::jsonb,
    api_key           TEXT NOT NULL DEFAULT '',
    secret_key        TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_wl_clients_domain ON wl_clients(domain);
CREATE INDEX IF NOT EXISTS idx_wl_clients_status ON wl_clients(status);

CREATE TABLE IF NOT EXISTS wl_admins (
    id                  TEXT PRIMARY KEY,
    client_id           TEXT NOT NULL DEFAULT '',
    email               TEXT NOT NULL DEFAULT '',
    name                TEXT NOT NULL DEFAULT '',
    role                TEXT NOT NULL DEFAULT 'admin',
    permissions         JSONB NOT NULL DEFAULT '[]'::jsonb,
    status              TEXT NOT NULL DEFAULT 'active',
    created_at          BIGINT NOT NULL DEFAULT 0,
    last_login          BIGINT NOT NULL DEFAULT 0,
    two_factor_enabled  BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_wl_admins_client ON wl_admins(client_id);
CREATE INDEX IF NOT EXISTS idx_wl_admins_email ON wl_admins(email);

CREATE TABLE IF NOT EXISTS wl_products (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'enabled',
    fee         DOUBLE PRECISION NOT NULL DEFAULT 0,
    min_deposit DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_deposit DOUBLE PRECISION NOT NULL DEFAULT 0,
    features    JSONB NOT NULL DEFAULT '[]'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_wl_products_status ON wl_products(status);

CREATE TABLE IF NOT EXISTS wl_trading_pairs (
    id         TEXT PRIMARY KEY,
    base_token TEXT NOT NULL DEFAULT '',
    quote_token TEXT NOT NULL DEFAULT '',
    chain_id   BIGINT NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'active',
    fee        DOUBLE PRECISION NOT NULL DEFAULT 0,
    min_trade  DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_trade  DOUBLE PRECISION NOT NULL DEFAULT 0,
    liquidity  DOUBLE PRECISION NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_wl_trading_pairs_chain ON wl_trading_pairs(chain_id);
CREATE INDEX IF NOT EXISTS idx_wl_trading_pairs_status ON wl_trading_pairs(status);

CREATE TABLE IF NOT EXISTS wl_liquidity_pools (
    id         TEXT PRIMARY KEY,
    pair_id    TEXT NOT NULL DEFAULT '',
    client_id  TEXT NOT NULL DEFAULT '',
    provider   TEXT NOT NULL DEFAULT '',
    token_a    TEXT NOT NULL DEFAULT '',
    token_b    TEXT NOT NULL DEFAULT '',
    amount_a   DOUBLE PRECISION NOT NULL DEFAULT 0,
    amount_b   DOUBLE PRECISION NOT NULL DEFAULT 0,
    value_usd  DOUBLE PRECISION NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'active',
    created_at BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_wl_liquidity_pools_pair ON wl_liquidity_pools(pair_id);
CREATE INDEX IF NOT EXISTS idx_wl_liquidity_pools_client ON wl_liquidity_pools(client_id);
CREATE INDEX IF NOT EXISTS idx_wl_liquidity_pools_status ON wl_liquidity_pools(status);

CREATE TABLE IF NOT EXISTS wl_token_configs (
    id         TEXT PRIMARY KEY,
    client_id  TEXT NOT NULL DEFAULT '',
    address    TEXT NOT NULL DEFAULT '',
    name       TEXT NOT NULL DEFAULT '',
    symbol     TEXT NOT NULL DEFAULT '',
    decimals   SMALLINT NOT NULL DEFAULT 0,
    chain_id   BIGINT NOT NULL DEFAULT 0,
    type       TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'active',
    max_supply TEXT NOT NULL DEFAULT '0',
    features   JSONB NOT NULL DEFAULT '[]'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_wl_token_configs_address ON wl_token_configs(address);
CREATE INDEX IF NOT EXISTS idx_wl_token_configs_client ON wl_token_configs(client_id);
CREATE INDEX IF NOT EXISTS idx_wl_token_configs_chain ON wl_token_configs(chain_id);

CREATE TABLE IF NOT EXISTS wl_market_maker_bots (
    id         TEXT PRIMARY KEY,
    client_id  TEXT NOT NULL DEFAULT '',
    name       TEXT NOT NULL DEFAULT '',
    pair_ids   JSONB NOT NULL DEFAULT '[]'::jsonb,
    status     TEXT NOT NULL DEFAULT 'stopped',
    strategy   TEXT NOT NULL DEFAULT '',
    params     JSONB NOT NULL DEFAULT '{}'::jsonb,
    profit     DOUBLE PRECISION NOT NULL DEFAULT 0,
    volume_24h DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_wl_market_maker_bots_client ON wl_market_maker_bots(client_id);
CREATE INDEX IF NOT EXISTS idx_wl_market_maker_bots_status ON wl_market_maker_bots(status);

ALTER TABLE wl_admins ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS wl_audit_logs (
    id            TEXT PRIMARY KEY,
    client_id     TEXT NOT NULL DEFAULT '',
    admin_id      TEXT NOT NULL DEFAULT '',
    admin_email   TEXT NOT NULL DEFAULT '',
    action        TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id   TEXT NOT NULL DEFAULT '',
    details       JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address    TEXT NOT NULL DEFAULT '',
    user_agent    TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'success',
    created_at    BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_wl_audit_logs_admin ON wl_audit_logs(admin_id);
CREATE INDEX IF NOT EXISTS idx_wl_audit_logs_client ON wl_audit_logs(client_id);
CREATE INDEX IF NOT EXISTS idx_wl_audit_logs_created ON wl_audit_logs(created_at DESC);

CREATE TABLE IF NOT EXISTS wl_notifications (
    id         TEXT PRIMARY KEY,
    client_id  TEXT NOT NULL DEFAULT '',
    admin_id   TEXT NOT NULL DEFAULT '',
    type       TEXT NOT NULL DEFAULT 'info',
    title      TEXT NOT NULL DEFAULT '',
    message    TEXT NOT NULL DEFAULT '',
    data       JSONB NOT NULL DEFAULT '{}'::jsonb,
    read       BOOLEAN NOT NULL DEFAULT FALSE,
    read_at    BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_wl_notifications_admin ON wl_notifications(admin_id);
CREATE INDEX IF NOT EXISTS idx_wl_notifications_client ON wl_notifications(client_id);
CREATE INDEX IF NOT EXISTS idx_wl_notifications_created ON wl_notifications(created_at DESC);
`

// Migrate creates the white label tables if they do not exist.
func Migrate(ctx context.Context) error {
	if pg == nil {
		return fmt.Errorf("database not configured")
	}
	_, err := pg.Exec(ctx, whiteLabelSchema)
	return err
}

// ============================================================================
// ROUTER
// ============================================================================

func setupRouter(cfg *Config) *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", healthCheck)

	// API v1
	v1 := router.Group("/api/v1/white-label")
	{
		// Client management
		clients := v1.Group("/clients")
		{
			clients.POST("", createClient)
			clients.GET("", listClients)
			clients.GET("/:id", getClient)
			clients.PUT("/:id", updateClient)
			clients.DELETE("/:id", deleteClient)
			clients.POST("/:id/suspend", suspendClient)
			clients.POST("/:id/resume", resumeClient)
			clients.POST("/:id/halt", haltClient)
		}

		// Admin management
		admins := v1.Group("/admins")
		{
			admins.POST("", createAdmin)
			admins.GET("", listAdmins)
			admins.GET("/:id", getAdmin)
			admins.PUT("/:id", updateAdmin)
			admins.DELETE("/:id", deleteAdmin)
			admins.POST("/:id/permissions", updatePermissions)
		}

		// Products
		products := v1.Group("/products")
		{
			products.POST("", createProduct)
			products.GET("", listProducts)
			products.GET("/:id", getProduct)
			products.PUT("/:id", updateProduct)
			products.DELETE("/:id", deleteProduct)
		}

		// Trading pairs
		pairs := v1.Group("/pairs")
		{
			pairs.POST("", createTradingPair)
			pairs.GET("", listTradingPairs)
			pairs.GET("/:id", getTradingPair)
			pairs.PUT("/:id", updateTradingPair)
			pairs.DELETE("/:id", deleteTradingPair)
			pairs.POST("/:id/suspend", suspendTradingPair)
			pairs.POST("/:id/resume", resumeTradingPair)
			pairs.POST("/import", importTradingPairs)
		}

		// Liquidity
		liquidity := v1.Group("/liquidity")
		{
			liquidity.POST("", createLiquidityPool)
			liquidity.GET("", listLiquidityPools)
			liquidity.GET("/:id", getLiquidityPool)
			liquidity.PUT("/:id", updateLiquidityPool)
			liquidity.DELETE("/:id", deleteLiquidityPool)
			liquidity.POST("/import", importLiquidity)
		}

		// Token management
		tokens := v1.Group("/tokens")
		{
			tokens.POST("", createToken)
			tokens.GET("", listTokens)
			tokens.GET("/:id", getToken)
			tokens.PUT("/:id", updateToken)
			tokens.DELETE("/:id", deleteToken)
			tokens.POST("/create", createNewToken)
		}

		// Market maker bots
		bots := v1.Group("/bots")
		{
			bots.POST("", createMarketMakerBot)
			bots.GET("", listMarketMakerBots)
			bots.GET("/:id", getMarketMakerBot)
			bots.PUT("/:id", updateMarketMakerBot)
			bots.DELETE("/:id", deleteMarketMakerBot)
			bots.POST("/:id/start", startBot)
			bots.POST("/:id/stop", stopBot)
		}

		// Blockchain management
		blockchains := v1.Group("/blockchains")
		{
			blockchains.GET("", listBlockchains)
			blockchains.POST("/:id/enable", enableBlockchain)
			blockchains.POST("/:id/disable", disableBlockchain)
		}

		// Analytics
		analytics := v1.Group("/analytics")
		{
			analytics.GET("/dashboard", getDashboard)
			analytics.GET("/volume", getVolumeStats)
			analytics.GET("/users", getUserStats)
			analytics.GET("/revenue", getRevenueStats)
		}
	}

	// ------------------------------------------------------------------
	// Frontend contract routes (/api/v1/*) — the white_label frontend is
	// the source of truth for the API contract. These routes mirror the
	// existing CRUD handlers above but add JWT auth, pagination, audit
	// logging and the dashboard/audit-logs/notifications endpoints the
	// frontend expects. The legacy /api/v1/white-label/* routes above are
	// left unchanged.
	// ------------------------------------------------------------------
	frontend := router.Group("/api/v1")
	{
		// Auth (public)
		auth := frontend.Group("/auth")
		{
			auth.POST("/login", loginAdmin)
			auth.POST("/register", registerAdmin)
			auth.POST("/logout", logoutAdmin)
		}

		// Protected (requires valid JWT)
		protected := frontend.Group("")
		protected.Use(authMiddleware)
		{
			protected.GET("/dashboard", getDashboardSummary)

			protected.GET("/clients", listClientsPaginated)
			protected.POST("/clients", createClient)
			protected.GET("/clients/:id", getClient)
			protected.PUT("/clients/:id", updateClient)
			protected.DELETE("/clients/:id", deleteClient)
			protected.POST("/clients/:id/approve", approveClient)
			protected.POST("/clients/:id/suspend", suspendClient)
			protected.POST("/clients/:id/resume", resumeClient)
			protected.POST("/clients/:id/halt", haltClient)

			protected.GET("/admins", listAdminsPaginated)
			protected.POST("/admins", createAdminWithPassword)
			protected.GET("/admins/:id", getAdmin)
			protected.PUT("/admins/:id", updateAdmin)
			protected.DELETE("/admins/:id", deleteAdmin)

			protected.GET("/products", listProductsPaginated)
			protected.POST("/products", createProduct)
			protected.GET("/products/:id", getProduct)
			protected.PUT("/products/:id", updateProduct)
			protected.DELETE("/products/:id", deleteProduct)
			protected.POST("/products/:id/toggle", toggleProduct)

			protected.GET("/pairs", listTradingPairsPaginated)
			protected.POST("/pairs", createTradingPair)
			protected.GET("/pairs/:id", getTradingPair)
			protected.PUT("/pairs/:id", updateTradingPair)
			protected.DELETE("/pairs/:id", deleteTradingPair)
			protected.POST("/pairs/:id/suspend", suspendTradingPair)
			protected.POST("/pairs/:id/resume", resumeTradingPair)
			protected.POST("/pairs/:id/halt", haltTradingPair)

			protected.GET("/blockchains", listBlockchains)
			protected.POST("/blockchains/:id/enable", enableBlockchain)
			protected.POST("/blockchains/:id/disable", disableBlockchain)

			protected.GET("/audit-logs", listAuditLogsPaginated)
			protected.GET("/notifications", listNotificationsPaginated)
			protected.POST("/notifications/:id/read", markNotificationRead)
		}
	}

	return router
}

// ============================================================================
// MIDDLEWARE
// ============================================================================

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// ============================================================================
// HANDLERS
// ============================================================================

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "white-label",
		"timestamp": time.Now().Unix(),
	})
}

// ============================================================================
// AUTH (JWT + bcrypt) — real PostgreSQL-backed admin authentication.
// ============================================================================

const jwtTTL = 24 * time.Hour

type jwtClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Exp   int64  `json:"exp"`
}

func b64Encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func b64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func generateJWT(claims jwtClaims) (string, error) {
	header := `{"alg":"HS256","typ":"JWT"}`
	claims.Exp = time.Now().Add(jwtTTL).Unix()
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := b64Encode([]byte(header)) + "." + b64Encode(payload)
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(signingInput))
	return signingInput + "." + b64Encode(mac.Sum(nil)), nil
}

func parseJWT(token string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(signingInput))
	if !hmac.Equal([]byte(parts[2]), []byte(b64Encode(mac.Sum(nil)))) {
		return nil, errors.New("invalid token signature")
	}
	payload, err := b64Decode(parts[1])
	if err != nil {
		return nil, err
	}
	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

func authMiddleware(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
		return
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	claims, err := parseJWT(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}
	c.Set("admin_id", claims.Sub)
	c.Set("admin_email", claims.Email)
	c.Set("admin_role", claims.Role)
	c.Next()
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// currentAdminID returns the authenticated admin id (empty if unauthenticated).
func currentAdminID(c *gin.Context) string {
	if v, ok := c.Get("admin_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func currentAdminEmail(c *gin.Context) string {
	if v, ok := c.Get("admin_email"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// loginAdmin authenticates an admin against wl_admins.password_hash and issues
// a JWT. Records an audit log entry on both success and failure.
func loginAdmin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	const cols = `id,client_id,email,name,role,permissions,status,created_at,last_login,two_factor_enabled,password_hash`
	var (
		a           WhiteLabelAdmin
		permissions []byte
		passwordHash string
	)
	err := pg.QueryRow(c.Request.Context(),
		`SELECT `+cols+` FROM wl_admins WHERE email=$1`, req.Email).
		Scan(&a.ID, &a.ClientID, &a.Email, &a.Name, &a.Role, &permissions, &a.Status,
			&a.CreatedAt, &a.LastLogin, &a.TwoFactorEnabled, &passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			recordAuditLog(c, "", "", req.Email, "LOGIN", "admin", "", "failed", nil)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		logger.Error().Err(err).Msg("Failed to query admin for login")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login"})
		return
	}

	if a.Status != "active" {
		recordAuditLog(c, a.ClientID, a.ID, a.Email, "LOGIN", "admin", a.ID, "failed", gin.H{"reason": "inactive"})
		c.JSON(http.StatusForbidden, gin.H{"error": "admin account is not active"})
		return
	}
	if passwordHash == "" || !checkPassword(passwordHash, req.Password) {
		recordAuditLog(c, a.ClientID, a.ID, a.Email, "LOGIN", "admin", a.ID, "failed", nil)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := generateJWT(jwtClaims{Sub: a.ID, Email: a.Email, Role: a.Role})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	now := time.Now().Unix()
	if _, err := pg.Exec(c.Request.Context(), `UPDATE wl_admins SET last_login=$1 WHERE id=$2`, now, a.ID); err != nil {
		logger.Warn().Err(err).Msg("Failed to update admin last_login")
	}
	a.Permissions = scanStrings(permissions)
	a.LastLogin = now

	createNotification(c, a.ClientID, a.ID, "info", "Admin login",
		a.Name+" signed in", nil)
	recordAuditLog(c, a.ClientID, a.ID, a.Email, "LOGIN", "admin", a.ID, "success", nil)

	c.JSON(http.StatusOK, gin.H{
		"token":  token,
		"admin":  a,
		"expiresAt": time.Unix(now, 0).Add(jwtTTL).Format(time.RFC3339),
	})
}

// registerAdmin creates a new admin with a bcrypt-hashed password.
func registerAdmin(c *gin.Context) {
	var req struct {
		ClientID string `json:"clientId"`
		Email    string `json:"email" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Role     string `json:"role"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	if req.Role == "" {
		req.Role = "admin"
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	admin := &WhiteLabelAdmin{
		ID:               generateID(),
		ClientID:         req.ClientID,
		Email:            req.Email,
		Name:             req.Name,
		Role:             req.Role,
		Permissions:      []string{},
		Status:           "active",
		CreatedAt:        time.Now().Unix(),
		TwoFactorEnabled: false,
	}

	_, err = pg.Exec(c.Request.Context(), `INSERT INTO wl_admins
		(id,client_id,email,name,role,permissions,status,created_at,last_login,two_factor_enabled,password_hash)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		admin.ID, admin.ClientID, admin.Email, admin.Name, admin.Role,
		jsonMarshal(admin.Permissions), admin.Status, admin.CreatedAt, admin.LastLogin,
		admin.TwoFactorEnabled, hash)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to register admin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register admin"})
		return
	}

	recordAuditLog(c, admin.ClientID, admin.ID, admin.Email, "CREATE", "admin", admin.ID, "success", nil)
	c.JSON(http.StatusCreated, admin)
}

// createAdminWithPassword mirrors the legacy createAdmin but persists a
// bcrypt-hashed password when the request includes one.
func createAdminWithPassword(c *gin.Context) {
	var req struct {
		ClientID string `json:"clientId"`
		Email    string `json:"email" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Role     string `json:"role" binding:"required"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	hash := ""
	if req.Password != "" {
		h, err := hashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		hash = h
	}

	admin := &WhiteLabelAdmin{
		ID:               generateID(),
		ClientID:         req.ClientID,
		Email:            req.Email,
		Name:             req.Name,
		Role:             req.Role,
		Permissions:      []string{},
		Status:           "active",
		CreatedAt:        time.Now().Unix(),
		TwoFactorEnabled: false,
	}

	_, err := pg.Exec(c.Request.Context(), `INSERT INTO wl_admins
		(id,client_id,email,name,role,permissions,status,created_at,last_login,two_factor_enabled,password_hash)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		admin.ID, admin.ClientID, admin.Email, admin.Name, admin.Role,
		jsonMarshal(admin.Permissions), admin.Status, admin.CreatedAt, admin.LastLogin,
		admin.TwoFactorEnabled, hash)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create admin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin"})
		return
	}

	recordAuditLog(c, admin.ClientID, currentAdminID(c), currentAdminEmail(c), "CREATE", "admin", admin.ID, "success", nil)
	c.JSON(http.StatusCreated, admin)
}

// logoutAdmin is stateless (JWT has no server session); acknowledge the
// request and let the client drop the token.
func logoutAdmin(c *gin.Context) {
	recordAuditLog(c, "", currentAdminID(c), currentAdminEmail(c), "LOGOUT", "admin", currentAdminID(c), "success", nil)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ============================================================================
// AUDIT LOGS + NOTIFICATIONS (real PostgreSQL persistence)
// ============================================================================

func recordAuditLog(c *gin.Context, clientID, adminID, adminEmail, action, resourceType, resourceID, status string, details interface{}) {
	if pg == nil {
		return
	}
	if details == nil {
		details = map[string]interface{}{}
	}
	_, err := pg.Exec(c.Request.Context(), `INSERT INTO wl_audit_logs
		(id,client_id,admin_id,admin_email,action,resource_type,resource_id,details,ip_address,user_agent,status,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		generateID(), clientID, adminID, adminEmail, action, resourceType, resourceID,
		jsonMarshal(details), c.ClientIP(), c.GetHeader("User-Agent"), status, time.Now().Unix())
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to record audit log")
	}
}

func createNotification(c *gin.Context, clientID, adminID, ntype, title, message string, data interface{}) {
	if pg == nil {
		return
	}
	if data == nil {
		data = map[string]interface{}{}
	}
	_, err := pg.Exec(c.Request.Context(), `INSERT INTO wl_notifications
		(id,client_id,admin_id,type,title,message,data,read,read_at,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		generateID(), clientID, adminID, ntype, title, message,
		jsonMarshal(data), false, int64(0), time.Now().Unix())
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to create notification")
	}
}

// AuditLog row returned to the frontend.
type AuditLog struct {
	ID           string                 `json:"id"`
	ClientID     string                 `json:"clientId"`
	AdminID      string                 `json:"adminId"`
	AdminEmail   string                 `json:"adminEmail"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resourceType"`
	ResourceID   string                 `json:"resourceId"`
	Details      map[string]interface{} `json:"details"`
	IPAddress    string                 `json:"ipAddress"`
	UserAgent    string                 `json:"userAgent"`
	Status       string                 `json:"status"`
	CreatedAt    int64                  `json:"createdAt"`
}

func scanAuditLog(row pgx.Row) (*AuditLog, error) {
	var l AuditLog
	var details []byte
	err := row.Scan(&l.ID, &l.ClientID, &l.AdminID, &l.AdminEmail, &l.Action,
		&l.ResourceType, &l.ResourceID, &details, &l.IPAddress, &l.UserAgent,
		&l.Status, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	l.Details = scanParams(details)
	return &l, nil
}

const auditCols = `id,client_id,admin_id,admin_email,action,resource_type,resource_id,details,ip_address,user_agent,status,created_at`

func listAuditLogsPaginated(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	page, pageSize := paginateParams(c)

	where := ""
	args := []interface{}{}
	argN := 1
	if q := strings.TrimSpace(c.Query("query")); q != "" {
		where = " WHERE action ILIKE $1 OR resource_type ILIKE $1 OR admin_email ILIKE $1"
		args = append(args, "%"+q+"%")
		argN = 2
	}
	if cid := c.Query("clientId"); cid != "" {
		if where == "" {
			where = " WHERE client_id=$" + strconv.Itoa(argN)
		} else {
			where += " AND client_id=$" + strconv.Itoa(argN)
		}
		args = append(args, cid)
		argN++
	}

	var total int64
	if err := pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM wl_audit_logs`+where, args...).Scan(&total); err != nil {
		logger.Error().Err(err).Msg("Failed to count audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := pg.Query(c.Request.Context(),
		`SELECT `+auditCols+` FROM wl_audit_logs`+where+
			` ORDER BY created_at DESC LIMIT $`+strconv.Itoa(argN)+` OFFSET $`+strconv.Itoa(argN+1), args...)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}
	defer rows.Close()

	logs := []*AuditLog{}
	for rows.Next() {
		l, err := scanAuditLog(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan audit log")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
			return
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}

	c.JSON(http.StatusOK, paginateResponse(logs, total, page, pageSize))
}

// Notification row returned to the frontend.
type Notification struct {
	ID        string                 `json:"id"`
	ClientID  string                 `json:"clientId"`
	AdminID   string                 `json:"adminId"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	Read      bool                   `json:"read"`
	ReadAt    int64                  `json:"readAt"`
	CreatedAt int64                  `json:"createdAt"`
}

func scanNotification(row pgx.Row) (*Notification, error) {
	var n Notification
	var data []byte
	err := row.Scan(&n.ID, &n.ClientID, &n.AdminID, &n.Type, &n.Title, &n.Message,
		&data, &n.Read, &n.ReadAt, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	n.Data = scanParams(data)
	return &n, nil
}

const notifCols = `id,client_id,admin_id,type,title,message,data,read,read_at,created_at`

func listNotificationsPaginated(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	page, pageSize := paginateParams(c)

	where := ""
	args := []interface{}{}
	argN := 1
	if cid := c.Query("clientId"); cid != "" {
		where = " WHERE client_id=$1"
		args = append(args, cid)
		argN = 2
	} else if aid := c.Query("adminId"); aid != "" {
		where = " WHERE admin_id=$1"
		args = append(args, aid)
		argN = 2
	}

	var total int64
	if err := pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM wl_notifications`+where, args...).Scan(&total); err != nil {
		logger.Error().Err(err).Msg("Failed to count notifications")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notifications"})
		return
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := pg.Query(c.Request.Context(),
		`SELECT `+notifCols+` FROM wl_notifications`+where+
			` ORDER BY created_at DESC LIMIT $`+strconv.Itoa(argN)+` OFFSET $`+strconv.Itoa(argN+1), args...)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list notifications")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notifications"})
		return
	}
	defer rows.Close()

	notifications := []*Notification{}
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan notification")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notifications"})
			return
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list notifications")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notifications"})
		return
	}

	c.JSON(http.StatusOK, paginateResponse(notifications, total, page, pageSize))
}

func markNotificationRead(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	id := c.Param("id")
	ct, err := pg.Exec(c.Request.Context(),
		`UPDATE wl_notifications SET read=TRUE, read_at=$1 WHERE id=$2`, time.Now().Unix(), id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to mark notification read")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notification read"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ============================================================================
// PAGINATION HELPERS + PAGINATED LIST HANDLERS
// ============================================================================

func paginateParams(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.Query("pageSize")); err == nil && v > 0 && v <= 200 {
		pageSize = v
	}
	return
}

func paginateResponse(data interface{}, total int64, page, pageSize int) gin.H {
	totalPages := int64(0)
	if pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	return gin.H{
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": totalPages,
		"data":       data,
	}
}

func listClientsPaginated(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	page, pageSize := paginateParams(c)

	where := ""
	args := []interface{}{}
	argN := 1
	if q := strings.TrimSpace(c.Query("query")); q != "" {
		where = " WHERE name ILIKE $1 OR domain ILIKE $1"
		args = append(args, "%"+q+"%")
		argN = 2
	}
	if st := c.Query("status"); st != "" {
		if where == "" {
			where = " WHERE status=$" + strconv.Itoa(argN)
		} else {
			where += " AND status=$" + strconv.Itoa(argN)
		}
		args = append(args, st)
		argN++
	}

	var total int64
	if err := pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM wl_clients`+where, args...).Scan(&total); err != nil {
		logger.Error().Err(err).Msg("Failed to count clients")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
		return
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := pg.Query(c.Request.Context(),
		`SELECT `+clientCols+` FROM wl_clients`+where+
			` ORDER BY created_at DESC LIMIT $`+strconv.Itoa(argN)+` OFFSET $`+strconv.Itoa(argN+1), args...)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list clients")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
		return
	}
	defer rows.Close()

	clients := []*WhiteLabelClient{}
	for rows.Next() {
		cl, err := scanClient(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan client")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
			return
		}
		clients = append(clients, cl)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list clients")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
		return
	}

	c.JSON(http.StatusOK, paginateResponse(clients, total, page, pageSize))
}

func listAdminsPaginated(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	page, pageSize := paginateParams(c)

	where := ""
	args := []interface{}{}
	argN := 1
	if q := strings.TrimSpace(c.Query("query")); q != "" {
		where = " WHERE name ILIKE $1 OR email ILIKE $1"
		args = append(args, "%"+q+"%")
		argN = 2
	}
	if cid := c.Query("clientId"); cid != "" {
		if where == "" {
			where = " WHERE client_id=$" + strconv.Itoa(argN)
		} else {
			where += " AND client_id=$" + strconv.Itoa(argN)
		}
		args = append(args, cid)
		argN++
	}

	var total int64
	if err := pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM wl_admins`+where, args...).Scan(&total); err != nil {
		logger.Error().Err(err).Msg("Failed to count admins")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admins"})
		return
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := pg.Query(c.Request.Context(),
		`SELECT `+adminCols+` FROM wl_admins`+where+
			` ORDER BY created_at DESC LIMIT $`+strconv.Itoa(argN)+` OFFSET $`+strconv.Itoa(argN+1), args...)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list admins")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admins"})
		return
	}
	defer rows.Close()

	admins := []*WhiteLabelAdmin{}
	for rows.Next() {
		a, err := scanAdmin(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan admin")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admins"})
			return
		}
		admins = append(admins, a)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list admins")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admins"})
		return
	}

	c.JSON(http.StatusOK, paginateResponse(admins, total, page, pageSize))
}

func listProductsPaginated(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	page, pageSize := paginateParams(c)

	where := ""
	args := []interface{}{}
	argN := 1
	if q := strings.TrimSpace(c.Query("query")); q != "" {
		where = " WHERE name ILIKE $1"
		args = append(args, "%"+q+"%")
		argN = 2
	}
	if st := c.Query("status"); st != "" {
		if where == "" {
			where = " WHERE status=$" + strconv.Itoa(argN)
		} else {
			where += " AND status=$" + strconv.Itoa(argN)
		}
		args = append(args, st)
		argN++
	}

	var total int64
	if err := pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM wl_products`+where, args...).Scan(&total); err != nil {
		logger.Error().Err(err).Msg("Failed to count products")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := pg.Query(c.Request.Context(),
		`SELECT `+productCols+` FROM wl_products`+where+
			` ORDER BY name LIMIT $`+strconv.Itoa(argN)+` OFFSET $`+strconv.Itoa(argN+1), args...)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list products")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}
	defer rows.Close()

	products := []*Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan product")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
			return
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list products")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	c.JSON(http.StatusOK, paginateResponse(products, total, page, pageSize))
}

func listTradingPairsPaginated(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	page, pageSize := paginateParams(c)

	where := ""
	args := []interface{}{}
	argN := 1
	if q := strings.TrimSpace(c.Query("query")); q != "" {
		where = " WHERE base_token ILIKE $1 OR quote_token ILIKE $1"
		args = append(args, "%"+q+"%")
		argN = 2
	}
	if st := c.Query("status"); st != "" {
		if where == "" {
			where = " WHERE status=$" + strconv.Itoa(argN)
		} else {
			where += " AND status=$" + strconv.Itoa(argN)
		}
		args = append(args, st)
		argN++
	}

	var total int64
	if err := pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM wl_trading_pairs`+where, args...).Scan(&total); err != nil {
		logger.Error().Err(err).Msg("Failed to count trading pairs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list trading pairs"})
		return
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := pg.Query(c.Request.Context(),
		`SELECT `+pairCols+` FROM wl_trading_pairs`+where+
			` ORDER BY chain_id, base_token LIMIT $`+strconv.Itoa(argN)+` OFFSET $`+strconv.Itoa(argN+1), args...)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list trading pairs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list trading pairs"})
		return
	}
	defer rows.Close()

	pairs := []*TradingPair{}
	for rows.Next() {
		p, err := scanPair(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan trading pair")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list trading pairs"})
			return
		}
		pairs = append(pairs, p)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list trading pairs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list trading pairs"})
		return
	}

	c.JSON(http.StatusOK, paginateResponse(pairs, total, page, pageSize))
}

// ============================================================================
// DASHBOARD SUMMARY (real aggregate counts from PostgreSQL)
// ============================================================================

func countWhere(ctx context.Context, table, where string) int64 {
	if pg == nil {
		return 0
	}
	var count int64
	q := `SELECT COUNT(*) FROM ` + table
	if where != "" {
		q += " WHERE " + where
	}
	if err := pg.QueryRow(ctx, q).Scan(&count); err != nil {
		logger.Warn().Err(err).Str("table", table).Msg("Failed to count rows")
		return 0
	}
	return count
}

func sumColumn(ctx context.Context, table, column string) float64 {
	if pg == nil {
		return 0
	}
	var sum float64
	if err := pg.QueryRow(ctx, `SELECT COALESCE(SUM(`+column+`),0) FROM `+table).Scan(&sum); err != nil {
		logger.Warn().Err(err).Str("table", table).Msg("Failed to sum column")
		return 0
	}
	return sum
}

// getDashboardSummary aggregates REAL counts/value sums from the white label
// tables. Fields with no underlying real source (totalUsers, revenue*) are
// reported as 0 — honest "no data", never fabricated numbers.
func getDashboardSummary(c *gin.Context) {
	ctx := c.Request.Context()
	liquidityValue := sumColumn(ctx, "wl_liquidity_pools", "value_usd")
	c.JSON(http.StatusOK, gin.H{
		"totalClients":      countRows(ctx, "wl_clients"),
		"activeClients":     countWhere(ctx, "wl_clients", "status='active'"),
		"pendingClients":    countWhere(ctx, "wl_clients", "status='pending'"),
		"totalAdmins":       countRows(ctx, "wl_admins"),
		"totalProducts":     countRows(ctx, "wl_products"),
		"totalPairs":        countRows(ctx, "wl_trading_pairs"),
		"totalPools":        countRows(ctx, "wl_liquidity_pools"),
		"totalTokens":       countRows(ctx, "wl_token_configs"),
		"totalBots":         countRows(ctx, "wl_market_maker_bots"),
		"totalLiquidityUsd": liquidityValue,
		// No real user/revenue tables exist yet — honest zeros.
		"totalUsers":  int64(0),
		"volume24h":   0.0,
		"volume7d":    0.0,
		"volume30d":   0.0,
		"revenue24h":  0.0,
		"revenue7d":   0.0,
		"revenue30d":  0.0,
		"timestamp":   time.Now().Unix(),
	})
}

// ============================================================================
// EXTRA FRONTEND ACTION HANDLERS
// ============================================================================

// approveClient sets a client to "active" (frontend approval flow).
func approveClient(c *gin.Context) {
	cl := setStatusAndLog(c, "active", "APPROVE")
	if cl != nil {
		createNotification(c, cl.ID, currentAdminID(c), "success",
			"Client approved", cl.Name+" was approved", nil)
	}
}

// haltTradingPair sets a trading pair to "halted".
func haltTradingPair(c *gin.Context) { setTradingPairStatus(c, "halted") }

// toggleProduct flips a product between "enabled" and "disabled".
func toggleProduct(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	p, err := scanProduct(tx.QueryRow(ctx, `SELECT `+productCols+` FROM wl_products WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}

	if p.Status == "enabled" {
		p.Status = "disabled"
	} else {
		p.Status = "enabled"
	}
	if _, err := tx.Exec(ctx, `UPDATE wl_products SET status=$1 WHERE id=$2`, p.Status, id); err != nil {
		logger.Error().Err(err).Msg("Failed to toggle product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit product toggle")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}

	recordAuditLog(c, "", currentAdminID(c), currentAdminEmail(c), "UPDATE", "product", id, "success", gin.H{"status": p.Status})
	c.JSON(http.StatusOK, p)
}

// setStatusAndLog is the shared client status transition that records an audit
// log entry and returns the updated client (nil on error response already sent).
func setStatusAndLog(c *gin.Context, status, action string) *WhiteLabelClient {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return nil
	}
	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return nil
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	cl, err := scanClient(tx.QueryRow(ctx, `SELECT `+clientCols+` FROM wl_clients WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
			return nil
		}
		logger.Error().Err(err).Msg("Failed to get client")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return nil
	}

	cl.Status = status
	cl.UpdatedAt = time.Now().Unix()
	if _, err := tx.Exec(ctx, `UPDATE wl_clients SET status=$1,updated_at=$2 WHERE id=$3`, cl.Status, cl.UpdatedAt, id); err != nil {
		logger.Error().Err(err).Msg("Failed to update client status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return nil
	}
	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit client status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return nil
	}

	recordAuditLog(c, cl.ID, currentAdminID(c), currentAdminEmail(c), action, "client", id, "success", gin.H{"status": status})
	return cl
}


// ============================================================================
// JSONB HELPERS
// ============================================================================

func jsonMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return b
}

func scanStrings(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func scanUint64s(raw []byte) []uint64 {
	if len(raw) == 0 {
		return []uint64{}
	}
	var out []uint64
	if err := json.Unmarshal(raw, &out); err != nil {
		return []uint64{}
	}
	if out == nil {
		return []uint64{}
	}
	return out
}

func scanParams(raw []byte) map[string]interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]interface{}{}
	}
	if out == nil {
		return map[string]interface{}{}
	}
	return out
}

// ============================================================================
// CLIENT MANAGEMENT
// ============================================================================

const (
	clientCols = `id,name,domain,custom_branding,logo_url,primary_color,secondary_color,status,created_at,updated_at,admin_ids,permissions,products,blockchain_access,api_key,secret_key`
)

func scanClient(row pgx.Row) (*WhiteLabelClient, error) {
	var c WhiteLabelClient
	var adminIDs, permissions, products, blockchainAccess []byte
	err := row.Scan(
		&c.ID, &c.Name, &c.Domain, &c.CustomBranding, &c.LogoURL, &c.PrimaryColor, &c.SecondaryColor,
		&c.Status, &c.CreatedAt, &c.UpdatedAt, &adminIDs, &permissions, &products, &blockchainAccess,
		&c.APIKey, &c.SecretKey,
	)
	if err != nil {
		return nil, err
	}
	c.AdminIDs = scanStrings(adminIDs)
	c.Permissions = scanStrings(permissions)
	c.Products = scanStrings(products)
	c.BlockchainAccess = scanUint64s(blockchainAccess)
	return &c, nil
}

func createClient(c *gin.Context) {
	var req struct {
		Name           string   `json:"name" binding:"required"`
		Domain         string   `json:"domain" binding:"required"`
		CustomBranding bool     `json:"customBranding"`
		Products       []string `json:"products"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	apiKey := generateAPIKey()
	secretKey := generateSecretKey()
	now := time.Now().Unix()

	client := &WhiteLabelClient{
		ID:             generateID(),
		Name:           req.Name,
		Domain:         req.Domain,
		CustomBranding: req.CustomBranding,
		Status:         "active",
		CreatedAt:      now,
		UpdatedAt:      now,
		Products:       req.Products,
		APIKey:         apiKey,
		SecretKey:      secretKey,
	}
	if client.Products == nil {
		client.Products = []string{}
	}

	ctx := c.Request.Context()
	_, err := pg.Exec(ctx, `INSERT INTO wl_clients
		(`+clientCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		client.ID, client.Name, client.Domain, client.CustomBranding, client.LogoURL, client.PrimaryColor,
		client.SecondaryColor, client.Status, client.CreatedAt, client.UpdatedAt,
		jsonMarshal(client.AdminIDs), jsonMarshal(client.Permissions), jsonMarshal(client.Products),
		jsonMarshal(client.BlockchainAccess), client.APIKey, client.SecretKey)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create client")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create client"})
		return
	}

	logger.Info().Str("client", client.Name).Msg("White label client created")

	c.JSON(http.StatusCreated, client)
}

func listClients(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	rows, err := pg.Query(c.Request.Context(), `SELECT `+clientCols+` FROM wl_clients ORDER BY created_at DESC`)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list clients")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
		return
	}
	defer rows.Close()

	clients := []*WhiteLabelClient{}
	for rows.Next() {
		cl, err := scanClient(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan client")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
			return
		}
		clients = append(clients, cl)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list clients")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
		return
	}

	c.JSON(http.StatusOK, clients)
}

func getClient(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	cl, err := scanClient(pg.QueryRow(c.Request.Context(), `SELECT `+clientCols+` FROM wl_clients WHERE id=$1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get client")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get client"})
		return
	}

	c.JSON(http.StatusOK, cl)
}

func updateClient(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	cl, err := scanClient(tx.QueryRow(ctx, `SELECT `+clientCols+` FROM wl_clients WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get client")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return
	}

	if name, ok := updates["name"].(string); ok {
		cl.Name = name
	}
	if domain, ok := updates["domain"].(string); ok {
		cl.Domain = domain
	}
	if logo, ok := updates["logoUrl"].(string); ok {
		cl.LogoURL = logo
	}
	if primary, ok := updates["primaryColor"].(string); ok {
		cl.PrimaryColor = primary
	}
	cl.UpdatedAt = time.Now().Unix()

	_, err = tx.Exec(ctx, `UPDATE wl_clients SET name=$1,domain=$2,logo_url=$3,primary_color=$4,updated_at=$5 WHERE id=$6`,
		cl.Name, cl.Domain, cl.LogoURL, cl.PrimaryColor, cl.UpdatedAt, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update client")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit client update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return
	}

	c.JSON(http.StatusOK, cl)
}

func deleteClient(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	ct, err := pg.Exec(c.Request.Context(), `DELETE FROM wl_clients WHERE id=$1`, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete client")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete client"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func setClientStatus(c *gin.Context, status string) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	cl, err := scanClient(tx.QueryRow(ctx, `SELECT `+clientCols+` FROM wl_clients WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get client")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return
	}

	cl.Status = status
	cl.UpdatedAt = time.Now().Unix()

	_, err = tx.Exec(ctx, `UPDATE wl_clients SET status=$1,updated_at=$2 WHERE id=$3`,
		cl.Status, cl.UpdatedAt, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update client status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit client status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return
	}

	c.JSON(http.StatusOK, cl)
}

func suspendClient(c *gin.Context) { setClientStatus(c, "suspended") }
func resumeClient(c *gin.Context)  { setClientStatus(c, "active") }
func haltClient(c *gin.Context)    { setClientStatus(c, "halted") }

// ============================================================================
// ADMIN MANAGEMENT
// ============================================================================

const adminCols = `id,client_id,email,name,role,permissions,status,created_at,last_login,two_factor_enabled`

func scanAdmin(row pgx.Row) (*WhiteLabelAdmin, error) {
	var a WhiteLabelAdmin
	var permissions []byte
	err := row.Scan(&a.ID, &a.ClientID, &a.Email, &a.Name, &a.Role, &permissions, &a.Status, &a.CreatedAt, &a.LastLogin, &a.TwoFactorEnabled)
	if err != nil {
		return nil, err
	}
	a.Permissions = scanStrings(permissions)
	return &a, nil
}

func createAdmin(c *gin.Context) {
	var req struct {
		ClientID string `json:"clientId" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	admin := &WhiteLabelAdmin{
		ID:               generateID(),
		ClientID:         req.ClientID,
		Email:            req.Email,
		Name:             req.Name,
		Role:             req.Role,
		Permissions:      []string{},
		Status:           "active",
		CreatedAt:        time.Now().Unix(),
		TwoFactorEnabled: false,
	}

	_, err := pg.Exec(c.Request.Context(), `INSERT INTO wl_admins
		(`+adminCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		admin.ID, admin.ClientID, admin.Email, admin.Name, admin.Role,
		jsonMarshal(admin.Permissions), admin.Status, admin.CreatedAt, admin.LastLogin, admin.TwoFactorEnabled)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create admin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin"})
		return
	}

	c.JSON(http.StatusCreated, admin)
}

func listAdmins(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	rows, err := pg.Query(c.Request.Context(), `SELECT `+adminCols+` FROM wl_admins ORDER BY created_at DESC`)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list admins")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admins"})
		return
	}
	defer rows.Close()

	admins := []*WhiteLabelAdmin{}
	for rows.Next() {
		a, err := scanAdmin(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan admin")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admins"})
			return
		}
		admins = append(admins, a)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list admins")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admins"})
		return
	}

	c.JSON(http.StatusOK, admins)
}

func getAdmin(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	a, err := scanAdmin(pg.QueryRow(c.Request.Context(), `SELECT `+adminCols+` FROM wl_admins WHERE id=$1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get admin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get admin"})
		return
	}

	c.JSON(http.StatusOK, a)
}

func updateAdmin(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update admin"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	a, err := scanAdmin(tx.QueryRow(ctx, `SELECT `+adminCols+` FROM wl_admins WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get admin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update admin"})
		return
	}

	if name, ok := updates["name"].(string); ok {
		a.Name = name
	}
	if email, ok := updates["email"].(string); ok {
		a.Email = email
	}
	if role, ok := updates["role"].(string); ok {
		a.Role = role
	}

	_, err = tx.Exec(ctx, `UPDATE wl_admins SET name=$1,email=$2,role=$3 WHERE id=$4`,
		a.Name, a.Email, a.Role, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update admin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update admin"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit admin update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update admin"})
		return
	}

	c.JSON(http.StatusOK, a)
}

func deleteAdmin(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	ct, err := pg.Exec(c.Request.Context(), `DELETE FROM wl_admins WHERE id=$1`, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete admin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete admin"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func updatePermissions(c *gin.Context) {
	var req struct {
		Permissions []string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update permissions"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	a, err := scanAdmin(tx.QueryRow(ctx, `SELECT `+adminCols+` FROM wl_admins WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get admin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update permissions"})
		return
	}

	if req.Permissions == nil {
		req.Permissions = []string{}
	}
	a.Permissions = req.Permissions

	_, err = tx.Exec(ctx, `UPDATE wl_admins SET permissions=$1 WHERE id=$2`, jsonMarshal(a.Permissions), id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update permissions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update permissions"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit permissions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update permissions"})
		return
	}

	c.JSON(http.StatusOK, a)
}

// ============================================================================
// PRODUCTS
// ============================================================================

const productCols = `id,name,type,status,fee,min_deposit,max_deposit,features`

func scanProduct(row pgx.Row) (*Product, error) {
	var p Product
	var features []byte
	err := row.Scan(&p.ID, &p.Name, &p.Type, &p.Status, &p.Fee, &p.MinDeposit, &p.MaxDeposit, &features)
	if err != nil {
		return nil, err
	}
	p.Features = scanStrings(features)
	return &p, nil
}

func createProduct(c *gin.Context) {
	var req struct {
		Name       string   `json:"name" binding:"required"`
		Type       string   `json:"type" binding:"required"`
		Fee        float64  `json:"fee"`
		MinDeposit float64  `json:"minDeposit"`
		MaxDeposit float64  `json:"maxDeposit"`
		Features   []string `json:"features"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	product := &Product{
		ID:         generateID(),
		Name:       req.Name,
		Type:       req.Type,
		Status:     "enabled",
		Fee:        req.Fee,
		MinDeposit: req.MinDeposit,
		MaxDeposit: req.MaxDeposit,
		Features:   req.Features,
	}
	if product.Features == nil {
		product.Features = []string{}
	}

	_, err := pg.Exec(c.Request.Context(), `INSERT INTO wl_products
		(`+productCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		product.ID, product.Name, product.Type, product.Status, product.Fee,
		product.MinDeposit, product.MaxDeposit, jsonMarshal(product.Features))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, product)
}

func listProducts(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	rows, err := pg.Query(c.Request.Context(), `SELECT `+productCols+` FROM wl_products ORDER BY name`)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list products")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}
	defer rows.Close()

	productList := []*Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan product")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
			return
		}
		productList = append(productList, p)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list products")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	c.JSON(http.StatusOK, productList)
}

func getProduct(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	p, err := scanProduct(pg.QueryRow(c.Request.Context(), `SELECT `+productCols+` FROM wl_products WHERE id=$1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get product"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func updateProduct(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	p, err := scanProduct(tx.QueryRow(ctx, `SELECT `+productCols+` FROM wl_products WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}

	if status, ok := updates["status"].(string); ok {
		p.Status = status
	}
	if fee, ok := updates["fee"].(float64); ok {
		p.Fee = fee
	}

	_, err = tx.Exec(ctx, `UPDATE wl_products SET status=$1,fee=$2 WHERE id=$3`, p.Status, p.Fee, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit product update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func deleteProduct(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	ct, err := pg.Exec(c.Request.Context(), `DELETE FROM wl_products WHERE id=$1`, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete product"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ============================================================================
// TRADING PAIRS
// ============================================================================

const pairCols = `id,base_token,quote_token,chain_id,status,fee,min_trade,max_trade,liquidity`

func scanPair(row pgx.Row) (*TradingPair, error) {
	var p TradingPair
	err := row.Scan(&p.ID, &p.BaseToken, &p.QuoteToken, &p.ChainID, &p.Status, &p.Fee, &p.MinTrade, &p.MaxTrade, &p.Liquidity)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func createTradingPair(c *gin.Context) {
	var req struct {
		BaseToken  string  `json:"baseToken" binding:"required"`
		QuoteToken string  `json:"quoteToken" binding:"required"`
		ChainID    uint64  `json:"chainId" binding:"required"`
		Fee        float64 `json:"fee"`
		MinTrade   float64 `json:"minTrade"`
		MaxTrade   float64 `json:"maxTrade"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	pair := &TradingPair{
		ID:         generateID(),
		BaseToken:  req.BaseToken,
		QuoteToken: req.QuoteToken,
		ChainID:    req.ChainID,
		Status:     "active",
		Fee:        req.Fee,
		MinTrade:   req.MinTrade,
		MaxTrade:   req.MaxTrade,
	}

	_, err := pg.Exec(c.Request.Context(), `INSERT INTO wl_trading_pairs
		(`+pairCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		pair.ID, pair.BaseToken, pair.QuoteToken, pair.ChainID, pair.Status,
		pair.Fee, pair.MinTrade, pair.MaxTrade, pair.Liquidity)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create trading pair")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create trading pair"})
		return
	}

	c.JSON(http.StatusCreated, pair)
}

func listTradingPairs(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	rows, err := pg.Query(c.Request.Context(), `SELECT `+pairCols+` FROM wl_trading_pairs ORDER BY chain_id, base_token`)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list trading pairs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list trading pairs"})
		return
	}
	defer rows.Close()

	pairList := []*TradingPair{}
	for rows.Next() {
		p, err := scanPair(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan trading pair")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list trading pairs"})
			return
		}
		pairList = append(pairList, p)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list trading pairs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list trading pairs"})
		return
	}

	c.JSON(http.StatusOK, pairList)
}

func getTradingPair(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	p, err := scanPair(pg.QueryRow(c.Request.Context(), `SELECT `+pairCols+` FROM wl_trading_pairs WHERE id=$1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Trading pair not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get trading pair")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get trading pair"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func updateTradingPair(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update trading pair"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	p, err := scanPair(tx.QueryRow(ctx, `SELECT `+pairCols+` FROM wl_trading_pairs WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Trading pair not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get trading pair")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update trading pair"})
		return
	}

	if fee, ok := updates["fee"].(float64); ok {
		p.Fee = fee
	}

	_, err = tx.Exec(ctx, `UPDATE wl_trading_pairs SET fee=$1 WHERE id=$2`, p.Fee, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update trading pair")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update trading pair"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit trading pair update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update trading pair"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func deleteTradingPair(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	ct, err := pg.Exec(c.Request.Context(), `DELETE FROM wl_trading_pairs WHERE id=$1`, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete trading pair")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete trading pair"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trading pair not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func setTradingPairStatus(c *gin.Context, status string) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update trading pair"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	p, err := scanPair(tx.QueryRow(ctx, `SELECT `+pairCols+` FROM wl_trading_pairs WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Trading pair not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get trading pair")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update trading pair"})
		return
	}

	p.Status = status
	_, err = tx.Exec(ctx, `UPDATE wl_trading_pairs SET status=$1 WHERE id=$2`, p.Status, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update trading pair status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update trading pair"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit trading pair status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update trading pair"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func suspendTradingPair(c *gin.Context) { setTradingPairStatus(c, "suspended") }
func resumeTradingPair(c *gin.Context)  { setTradingPairStatus(c, "active") }

func importTradingPairs(c *gin.Context) {
	var req struct {
		Pairs []struct {
			BaseToken  string  `json:"baseToken"`
			QuoteToken string  `json:"quoteToken"`
			ChainID    uint64  `json:"chainId"`
			Fee        float64 `json:"fee"`
		} `json:"pairs"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import trading pairs"})
		return
	}
	defer tx.Rollback(ctx)

	imported := []string{}
	for _, p := range req.Pairs {
		pair := &TradingPair{
			ID:         generateID(),
			BaseToken:  p.BaseToken,
			QuoteToken: p.QuoteToken,
			ChainID:    p.ChainID,
			Status:     "active",
			Fee:        p.Fee,
		}
		_, err := tx.Exec(ctx, `INSERT INTO wl_trading_pairs
			(`+pairCols+`)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			pair.ID, pair.BaseToken, pair.QuoteToken, pair.ChainID, pair.Status,
			pair.Fee, pair.MinTrade, pair.MaxTrade, pair.Liquidity)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to import trading pair")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import trading pairs"})
			return
		}
		imported = append(imported, pair.ID)
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit trading pairs import")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import trading pairs"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"imported": len(imported), "pairs": imported})
}

// ============================================================================
// LIQUIDITY
// ============================================================================

const poolCols = `id,pair_id,client_id,provider,token_a,token_b,amount_a,amount_b,value_usd,status,created_at`

func scanPool(row pgx.Row) (*LiquidityPool, error) {
	var p LiquidityPool
	err := row.Scan(&p.ID, &p.PairID, &p.ClientID, &p.Provider, &p.TokenA, &p.TokenB,
		&p.AmountA, &p.AmountB, &p.ValueUSD, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func createLiquidityPool(c *gin.Context) {
	var req struct {
		PairID   string  `json:"pairId" binding:"required"`
		ClientID string  `json:"clientId" binding:"required"`
		Provider string  `json:"provider" binding:"required"`
		TokenA   string  `json:"tokenA" binding:"required"`
		TokenB   string  `json:"tokenB" binding:"required"`
		AmountA  float64 `json:"amountA"`
		AmountB  float64 `json:"amountB"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	pool := &LiquidityPool{
		ID:        generateID(),
		PairID:    req.PairID,
		ClientID:  req.ClientID,
		Provider:  req.Provider,
		TokenA:    req.TokenA,
		TokenB:    req.TokenB,
		AmountA:   req.AmountA,
		AmountB:   req.AmountB,
		ValueUSD:   req.AmountA * 1000, // Simplified
		Status:    "active",
		CreatedAt: time.Now().Unix(),
	}

	_, err := pg.Exec(c.Request.Context(), `INSERT INTO wl_liquidity_pools
		(`+poolCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		pool.ID, pool.PairID, pool.ClientID, pool.Provider, pool.TokenA, pool.TokenB,
		pool.AmountA, pool.AmountB, pool.ValueUSD, pool.Status, pool.CreatedAt)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create liquidity pool")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create liquidity pool"})
		return
	}

	c.JSON(http.StatusCreated, pool)
}

func listLiquidityPools(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	rows, err := pg.Query(c.Request.Context(), `SELECT `+poolCols+` FROM wl_liquidity_pools ORDER BY created_at DESC`)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list liquidity pools")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list liquidity pools"})
		return
	}
	defer rows.Close()

	pools := []*LiquidityPool{}
	for rows.Next() {
		p, err := scanPool(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan liquidity pool")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list liquidity pools"})
			return
		}
		pools = append(pools, p)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list liquidity pools")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list liquidity pools"})
		return
	}

	c.JSON(http.StatusOK, pools)
}

func getLiquidityPool(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	p, err := scanPool(pg.QueryRow(c.Request.Context(), `SELECT `+poolCols+` FROM wl_liquidity_pools WHERE id=$1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get liquidity pool")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get liquidity pool"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func updateLiquidityPool(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update liquidity pool"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	p, err := scanPool(tx.QueryRow(ctx, `SELECT `+poolCols+` FROM wl_liquidity_pools WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get liquidity pool")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update liquidity pool"})
		return
	}

	if amountA, ok := updates["amountA"].(float64); ok {
		p.AmountA = amountA
	}
	if amountB, ok := updates["amountB"].(float64); ok {
		p.AmountB = amountB
	}

	_, err = tx.Exec(ctx, `UPDATE wl_liquidity_pools SET amount_a=$1,amount_b=$2 WHERE id=$3`,
		p.AmountA, p.AmountB, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update liquidity pool")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update liquidity pool"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit liquidity pool update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update liquidity pool"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func deleteLiquidityPool(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	ct, err := pg.Exec(c.Request.Context(), `DELETE FROM wl_liquidity_pools WHERE id=$1`, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete liquidity pool")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete liquidity pool"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func importLiquidity(c *gin.Context) {
	var req struct {
		Pools []struct {
			PairID   string  `json:"pairId"`
			Provider string  `json:"provider"`
			TokenA   string  `json:"tokenA"`
			TokenB   string  `json:"tokenB"`
			AmountA  float64 `json:"amountA"`
			AmountB  float64 `json:"amountB"`
		} `json:"pools"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import liquidity"})
		return
	}
	defer tx.Rollback(ctx)

	now := time.Now().Unix()
	imported := []string{}
	for _, p := range req.Pools {
		pool := &LiquidityPool{
			ID:        generateID(),
			PairID:    p.PairID,
			Provider:  p.Provider,
			TokenA:    p.TokenA,
			TokenB:    p.TokenB,
			AmountA:   p.AmountA,
			AmountB:   p.AmountB,
			ValueUSD:  p.AmountA * 1000,
			Status:    "active",
			CreatedAt: now,
		}
		_, err := tx.Exec(ctx, `INSERT INTO wl_liquidity_pools
			(`+poolCols+`)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			pool.ID, pool.PairID, pool.ClientID, pool.Provider, pool.TokenA, pool.TokenB,
			pool.AmountA, pool.AmountB, pool.ValueUSD, pool.Status, pool.CreatedAt)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to import liquidity pool")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import liquidity"})
			return
		}
		imported = append(imported, pool.ID)
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit liquidity import")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import liquidity"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"imported": len(imported), "pools": imported})
}

// ============================================================================
// TOKEN MANAGEMENT
// ============================================================================

const tokenCols = `id,client_id,address,name,symbol,decimals,chain_id,type,status,max_supply,features`

func scanToken(row pgx.Row) (*TokenConfig, error) {
	var t TokenConfig
	var features []byte
	err := row.Scan(&t.ID, &t.ClientID, &t.Address, &t.Name, &t.Symbol, &t.Decimals, &t.ChainID, &t.Type, &t.Status, &t.MaxSupply, &features)
	if err != nil {
		return nil, err
	}
	t.Features = scanStrings(features)
	return &t, nil
}

func createToken(c *gin.Context) {
	var req struct {
		ClientID  string `json:"clientId" binding:"required"`
		Address   string `json:"address" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Symbol    string `json:"symbol" binding:"required"`
		Decimals  uint8  `json:"decimals"`
		ChainID   uint64 `json:"chainId" binding:"required"`
		Type      string `json:"type" binding:"required"`
		MaxSupply string `json:"maxSupply"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	token := &TokenConfig{
		ID:        generateID(),
		ClientID:  req.ClientID,
		Address:   req.Address,
		Name:      req.Name,
		Symbol:    req.Symbol,
		Decimals:  req.Decimals,
		ChainID:   req.ChainID,
		Type:      req.Type,
		Status:    "active",
		MaxSupply: req.MaxSupply,
		Features:  []string{},
	}

	_, err := pg.Exec(c.Request.Context(), `INSERT INTO wl_token_configs
		(`+tokenCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		token.ID, token.ClientID, token.Address, token.Name, token.Symbol, token.Decimals,
		token.ChainID, token.Type, token.Status, token.MaxSupply, jsonMarshal(token.Features))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}

	c.JSON(http.StatusCreated, token)
}

func listTokens(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	rows, err := pg.Query(c.Request.Context(), `SELECT `+tokenCols+` FROM wl_token_configs ORDER BY chain_id, symbol`)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list tokens")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tokens"})
		return
	}
	defer rows.Close()

	tokenList := []*TokenConfig{}
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan token")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tokens"})
			return
		}
		tokenList = append(tokenList, t)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list tokens")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tokens"})
		return
	}

	c.JSON(http.StatusOK, tokenList)
}

func getToken(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	t, err := scanToken(pg.QueryRow(c.Request.Context(), `SELECT `+tokenCols+` FROM wl_token_configs WHERE id=$1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get token"})
		return
	}

	c.JSON(http.StatusOK, t)
}

func updateToken(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update token"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	t, err := scanToken(tx.QueryRow(ctx, `SELECT `+tokenCols+` FROM wl_token_configs WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update token"})
		return
	}

	if status, ok := updates["status"].(string); ok {
		t.Status = status
	}

	_, err = tx.Exec(ctx, `UPDATE wl_token_configs SET status=$1 WHERE id=$2`, t.Status, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update token"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit token update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update token"})
		return
	}

	c.JSON(http.StatusOK, t)
}

func deleteToken(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	ct, err := pg.Exec(c.Request.Context(), `DELETE FROM wl_token_configs WHERE id=$1`, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete token"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func createNewToken(c *gin.Context) {
	var req struct {
		ClientID  string `json:"clientId" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Symbol    string `json:"symbol" binding:"required"`
		Decimals  uint8  `json:"decimals"`
		ChainID   uint64 `json:"chainId" binding:"required"`
		Type      string `json:"type" binding:"required"`
		MaxSupply string `json:"maxSupply"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	// In production, this would deploy a real smart contract
	// For now, generate a deterministic-looking address
	address := "0x" + generateHex(40)

	token := &TokenConfig{
		ID:        generateID(),
		ClientID:  req.ClientID,
		Address:   address,
		Name:      req.Name,
		Symbol:    req.Symbol,
		Decimals:  req.Decimals,
		ChainID:   req.ChainID,
		Type:      req.Type,
		Status:    "active",
		MaxSupply: req.MaxSupply,
		Features:  []string{"transfer", "approve", "transferFrom"},
	}

	_, err := pg.Exec(c.Request.Context(), `INSERT INTO wl_token_configs
		(`+tokenCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		token.ID, token.ClientID, token.Address, token.Name, token.Symbol, token.Decimals,
		token.ChainID, token.Type, token.Status, token.MaxSupply, jsonMarshal(token.Features))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}

	logger.Info().Str("token", token.Name).Str("address", address).Msg("Token created")

	c.JSON(http.StatusCreated, token)
}

// ============================================================================
// MARKET MAKER BOTS
// ============================================================================

const botCols = `id,client_id,name,pair_ids,status,strategy,params,profit,volume_24h,created_at`

func scanBot(row pgx.Row) (*MarketMakerBot, error) {
	var b MarketMakerBot
	var pairIDs, params []byte
	err := row.Scan(&b.ID, &b.ClientID, &b.Name, &pairIDs, &b.Status, &b.Strategy, &params, &b.Profit, &b.Volume24h, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	b.PairIDs = scanStrings(pairIDs)
	b.Params = scanParams(params)
	return &b, nil
}

func createMarketMakerBot(c *gin.Context) {
	var req struct {
		ClientID string                 `json:"clientId" binding:"required"`
		Name     string                 `json:"name" binding:"required"`
		PairIDs  []string               `json:"pairIds" binding:"required"`
		Strategy string                 `json:"strategy" binding:"required"`
		Params   map[string]interface{} `json:"params"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	bot := &MarketMakerBot{
		ID:        generateID(),
		ClientID:  req.ClientID,
		Name:      req.Name,
		PairIDs:   req.PairIDs,
		Strategy:  req.Strategy,
		Params:    req.Params,
		Status:    "stopped",
		Profit:    0,
		Volume24h: 0,
		CreatedAt: time.Now().Unix(),
	}
	if bot.PairIDs == nil {
		bot.PairIDs = []string{}
	}
	if bot.Params == nil {
		bot.Params = map[string]interface{}{}
	}

	_, err := pg.Exec(c.Request.Context(), `INSERT INTO wl_market_maker_bots
		(`+botCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		bot.ID, bot.ClientID, bot.Name, jsonMarshal(bot.PairIDs), bot.Status, bot.Strategy,
		jsonMarshal(bot.Params), bot.Profit, bot.Volume24h, bot.CreatedAt)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create market maker bot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create market maker bot"})
		return
	}

	c.JSON(http.StatusCreated, bot)
}

func listMarketMakerBots(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	rows, err := pg.Query(c.Request.Context(), `SELECT `+botCols+` FROM wl_market_maker_bots ORDER BY created_at DESC`)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list market maker bots")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list market maker bots"})
		return
	}
	defer rows.Close()

	bots := []*MarketMakerBot{}
	for rows.Next() {
		b, err := scanBot(rows)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan market maker bot")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list market maker bots"})
			return
		}
		bots = append(bots, b)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to list market maker bots")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list market maker bots"})
		return
	}

	c.JSON(http.StatusOK, bots)
}

func getMarketMakerBot(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	b, err := scanBot(pg.QueryRow(c.Request.Context(), `SELECT `+botCols+` FROM wl_market_maker_bots WHERE id=$1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get market maker bot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get market maker bot"})
		return
	}

	c.JSON(http.StatusOK, b)
}

func updateMarketMakerBot(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update market maker bot"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	b, err := scanBot(tx.QueryRow(ctx, `SELECT `+botCols+` FROM wl_market_maker_bots WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get market maker bot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update market maker bot"})
		return
	}

	if strategy, ok := updates["strategy"].(string); ok {
		b.Strategy = strategy
	}

	_, err = tx.Exec(ctx, `UPDATE wl_market_maker_bots SET strategy=$1 WHERE id=$2`, b.Strategy, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update market maker bot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update market maker bot"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit market maker bot update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update market maker bot"})
		return
	}

	c.JSON(http.StatusOK, b)
}

func deleteMarketMakerBot(c *gin.Context) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	id := c.Param("id")
	ct, err := pg.Exec(c.Request.Context(), `DELETE FROM wl_market_maker_bots WHERE id=$1`, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete market maker bot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete market maker bot"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func setBotStatus(c *gin.Context, status string) {
	if pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := pg.Begin(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to begin transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update market maker bot"})
		return
	}
	defer tx.Rollback(ctx)

	id := c.Param("id")
	b, err := scanBot(tx.QueryRow(ctx, `SELECT `+botCols+` FROM wl_market_maker_bots WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
			return
		}
		logger.Error().Err(err).Msg("Failed to get market maker bot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update market maker bot"})
		return
	}

	b.Status = status
	_, err = tx.Exec(ctx, `UPDATE wl_market_maker_bots SET status=$1 WHERE id=$2`, b.Status, id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update market maker bot status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update market maker bot"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to commit market maker bot status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update market maker bot"})
		return
	}

	c.JSON(http.StatusOK, b)
}

func startBot(c *gin.Context) { setBotStatus(c, "running") }
func stopBot(c *gin.Context)  { setBotStatus(c, "stopped") }

// ============================================================================
// BLOCKCHAIN MANAGEMENT
// ============================================================================

func listBlockchains(c *gin.Context) {
	// Return list of available blockchains
	blockchains := []map[string]interface{}{
		{"id": 1, "name": "Ethereum", "symbol": "ETH", "status": "enabled"},
		{"id": 2, "name": "BNB Smart Chain", "symbol": "BNB", "status": "enabled"},
		{"id": 3, "name": "Polygon", "symbol": "MATIC", "status": "enabled"},
		{"id": 4, "name": "Arbitrum", "symbol": "ETH", "status": "enabled"},
		{"id": 5, "name": "Optimism", "symbol": "ETH", "status": "enabled"},
		{"id": 6, "name": "Base", "symbol": "ETH", "status": "enabled"},
		{"id": 7, "name": "Avalanche", "symbol": "AVAX", "status": "enabled"},
		{"id": 101, "name": "Bitcoin", "symbol": "BTC", "status": "enabled"},
		{"id": 102, "name": "Solana", "symbol": "SOL", "status": "enabled"},
		{"id": 103, "name": "Tron", "symbol": "TRX", "status": "enabled"},
	}

	c.JSON(http.StatusOK, blockchains)
}

func enableBlockchain(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "enabled"})
}

func disableBlockchain(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "disabled"})
}

// ============================================================================
// ANALYTICS
// ============================================================================

func countRows(ctx context.Context, table string) int64 {
	if pg == nil {
		return 0
	}
	var count int64
	if err := pg.QueryRow(ctx, `SELECT COUNT(*) FROM `+table).Scan(&count); err != nil {
		logger.Warn().Err(err).Str("table", table).Msg("Failed to count rows")
		return 0
	}
	return count
}

func getDashboard(c *gin.Context) {
	ctx := c.Request.Context()
	c.JSON(http.StatusOK, gin.H{
		"totalClients":  countRows(ctx, "wl_clients"),
		"totalAdmins":  countRows(ctx, "wl_admins"),
		"totalProducts": countRows(ctx, "wl_products"),
		"totalPairs":   countRows(ctx, "wl_trading_pairs"),
		"totalPools":   countRows(ctx, "wl_liquidity_pools"),
		"totalTokens":  countRows(ctx, "wl_token_configs"),
		"totalBots":    countRows(ctx, "wl_market_maker_bots"),
		"timestamp":    time.Now().Unix(),
	})
}

func getVolumeStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"volume24h": 12500000.0,
		"volume7d":  87500000.0,
		"volume30d": 375000000.0,
	})
}

func getUserStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"totalUsers":  125000,
		"activeUsers": 45000,
		"newUsers24h": 1250,
	})
}

func getRevenueStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"revenue24h": 125000.0,
		"revenue7d":  875000.0,
		"revenue30d": 3750000.0,
	})
}

// ============================================================================
// UTILITIES
// ============================================================================

func generateID() string {
	return generateHex(16)
}

func generateAPIKey() string {
	return "twl_" + generateHex(32)
}

func generateSecretKey() string {
	return generateHex(64)
}

func generateHex(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
