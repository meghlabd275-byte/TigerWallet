// Unified TigerWallet API Gateway
// Reverse-proxy gateway that forwards every request to the canonical
// TigerWallet backend services. It performs NO data fabrication: every
// response is the real backend's response, or an honest 502 when a backend
// is unreachable.

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// Config holds all gateway configuration.
type Config struct {
	Port            string
	WalletAPI       string // canonical wallet_api (:8443)
	StakingService  string // :8001
	LendingService  string // :8009
	BridgeService   string // :8010
	SwapService     string // :8004
	NFTService      string // :8085
	PerpetualSvc    string // :8464
	GovernanceSvc   string // :8454
	CopyTradingSvc  string // :8006
	PredictionSvc   string // :8455
	FiatRampSvc     string // :8008
	JWTSecret       string
	RedisHost       string
	RedisPort       string
}

var (
	cfg         Config
	redisClient *redis.Client
	httpClient  *http.Client
)

// =============================================================================
// Initialization
// =============================================================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("TigerWallet Unified API Gateway starting...")

	loadConfig()
	initRedis()

	httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(rateLimitMiddleware())

	registerRoutes(router)

	router.GET("/health", handleHealth)
	router.GET("/ready", handleReady)

	log.Printf("Gateway listening on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start gateway: %v", err)
	}
}

func loadConfig() {
	cfg = Config{
		Port:           getEnv("PORT", "8080"),
		WalletAPI:      getEnv("WALLET_API", "http://localhost:8443"),
		StakingService: getEnv("STAKING_SERVICE", "http://localhost:8001"),
		LendingService: getEnv("LENDING_SERVICE", "http://localhost:8009"),
		BridgeService:  getEnv("BRIDGE_SERVICE", "http://localhost:8010"),
		SwapService:    getEnv("SWAP_SERVICE", "http://localhost:8004"),
		NFTService:     getEnv("NFT_SERVICE", "http://localhost:8085"),
		PerpetualSvc:   getEnv("PERPETUAL_SERVICE", "http://localhost:8464"),
		GovernanceSvc:  getEnv("GOVERNANCE_SERVICE", "http://localhost:8454"),
		CopyTradingSvc: getEnv("COPYTRADING_SERVICE", "http://localhost:8006"),
		PredictionSvc:  getEnv("PREDICTION_SERVICE", "http://localhost:8455"),
		FiatRampSvc:    getEnv("FIAT_RAMP_SERVICE", "http://localhost:8008"),
		JWTSecret:      getEnv("JWT_SECRET", "tigerwallet-secret-key-change-in-production"),
		RedisHost:      getEnv("REDIS_HOST", "localhost"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
	}
}

func initRedis() {
	addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	redisClient = redis.NewClient(&redis.Options{Addr: addr, PoolSize: 100, MinIdleConns: 10})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Redis connection failed: %v (continuing without cache)", err)
	} else {
		log.Println("Redis connected")
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// =============================================================================
// Middleware
// =============================================================================

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func rateLimitMiddleware() gin.HandlerFunc {
	type bucket struct {
		tokens    float64
		lastCheck int64
		mu        sync.Mutex
	}
	buckets := make(map[string]*bucket)
	var bucketMu sync.Mutex
	return func(c *gin.Context) {
		ip := c.ClientIP()
		bucketMu.Lock()
		b, ok := buckets[ip]
		if !ok {
			b = &bucket{tokens: 100, lastCheck: time.Now().Unix()}
			buckets[ip] = b
		}
		bucketMu.Unlock()
		b.mu.Lock()
		defer b.mu.Unlock()
		now := time.Now().Unix()
		elapsed := now - b.lastCheck
		b.tokens += float64(elapsed) * 10
		if b.tokens > 100 {
			b.tokens = 100
		}
		b.lastCheck = now
		if b.tokens < 1 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		b.tokens--
		c.Next()
	}
}

// authMiddleware validates the JWT (still issued by the canonical wallet_api).
// It forwards the original Authorization header to the backend unchanged so
// the backend re-validates authoritatively.
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}
		c.Set("user_id", claims["sub"])
		c.Next()
	}
}

// =============================================================================
// Reverse-proxy helpers
// =============================================================================

// forwardTo forwards the inbound request to a backend service, streaming the
// real response body back. It never fabricates data; on failure it returns 502.
func forwardTo(c *gin.Context, backendBase, suffix string) {
	base, err := url.Parse(backendBase)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid backend URL"})
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(base)
	// Rewrite the path so "/api/v1/wallet/create" -> "<base>/api/v1/wallets"
	// when a suffix override is given; otherwise forward the path verbatim
	// (minus the gateway's own /api/v1 prefix is kept so backends that expect
	// the same prefix work).
	targetPath := c.Request.URL.Path
	if suffix != "" {
		// Replace the captured route path with the backend suffix; query is kept.
		// The route registration guarantees the prefix matches, so we forward
		// the full original path plus the suffix mapping handled by the backend.
		targetPath = suffix
	}
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = base.Host
		if suffix != "" {
			req.URL.Path = strings.TrimRight(base.Path, "/") + targetPath
		}
		req.URL.RawQuery = c.Request.URL.RawQuery
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		log.Printf("backend %s unreachable: %v", backendBase, e)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"backend service unavailable"}`))
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}

// forwardVerbatim forwards the inbound request path+query verbatim to the
// backend base. Used when the backend exposes the same /api/v1/* paths.
func forwardVerbatim(c *gin.Context, backendBase string) {
	forwardTo(c, backendBase, "")
}

// =============================================================================
// Routes
// =============================================================================

func registerRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	// Auth -- delegate to canonical wallet_api (no demo login).
	api.POST("/auth/register", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
	api.POST("/auth/login", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })

	// Chains -- the wallet_api is the system of record for the chain registry.
	api.GET("/chains", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
	api.GET("/chains/:chain_id", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })

	// Wallet -- canonical wallet_api (:8443).
	wallet := api.Group("/wallet")
	wallet.Use(authMiddleware())
	{
		wallet.POST("/create", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
		wallet.GET("/list", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
		wallet.GET("/:address/balance", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
		wallet.GET("/:address/tokens", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
		wallet.GET("/:address/transactions", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
		wallet.GET("/:address/nfts", func(c *gin.Context) { forwardVerbatim(c, cfg.NFTService) })
	}

	// Staking -- staking_service.
	staking := api.Group("/staking")
	staking.Use(authMiddleware())
	{
		staking.GET("/pools", func(c *gin.Context) { forwardVerbatim(c, cfg.StakingService) })
		staking.GET("/positions", func(c *gin.Context) { forwardVerbatim(c, cfg.StakingService) })
		staking.POST("/stake", func(c *gin.Context) { forwardVerbatim(c, cfg.StakingService) })
		staking.POST("/unstake", func(c *gin.Context) { forwardVerbatim(c, cfg.StakingService) })
		staking.POST("/claim", func(c *gin.Context) { forwardVerbatim(c, cfg.StakingService) })
	}

	// Lending -- lending_service.
	lending := api.Group("/lending")
	lending.Use(authMiddleware())
	{
		lending.GET("/markets", func(c *gin.Context) { forwardVerbatim(c, cfg.LendingService) })
		lending.GET("/position", func(c *gin.Context) { forwardVerbatim(c, cfg.LendingService) })
		lending.POST("/supply", func(c *gin.Context) { forwardVerbatim(c, cfg.LendingService) })
		lending.POST("/withdraw", func(c *gin.Context) { forwardVerbatim(c, cfg.LendingService) })
		lending.POST("/borrow", func(c *gin.Context) { forwardVerbatim(c, cfg.LendingService) })
		lending.POST("/repay", func(c *gin.Context) { forwardVerbatim(c, cfg.LendingService) })
	}

	// Bridge -- bridge_service.
	bridge := api.Group("/bridge")
	bridge.Use(authMiddleware())
	{
		bridge.GET("/routes", func(c *gin.Context) { forwardVerbatim(c, cfg.BridgeService) })
		bridge.GET("/quote", func(c *gin.Context) { forwardVerbatim(c, cfg.BridgeService) })
		bridge.POST("/transfer", func(c *gin.Context) { forwardVerbatim(c, cfg.BridgeService) })
		bridge.GET("/history", func(c *gin.Context) { forwardVerbatim(c, cfg.BridgeService) })
	}

	// Swap -- swap_service / wallet_api AMM.
	swap := api.Group("/swap")
	swap.Use(authMiddleware())
	{
		swap.GET("/tokens", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
		swap.GET("/quote", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
		swap.GET("/routes", func(c *gin.Context) { forwardVerbatim(c, cfg.SwapService) })
		swap.POST("/execute", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
	}

	// NFT -- nft_service.
	nft := api.Group("/nft")
	nft.Use(authMiddleware())
	{
		nft.GET("/collections", func(c *gin.Context) { forwardVerbatim(c, cfg.NFTService) })
		nft.GET("/items", func(c *gin.Context) { forwardVerbatim(c, cfg.NFTService) })
		nft.GET("/:id", func(c *gin.Context) { forwardVerbatim(c, cfg.NFTService) })
		nft.POST("/buy", func(c *gin.Context) { forwardVerbatim(c, cfg.NFTService) })
		nft.POST("/sell", func(c *gin.Context) { forwardVerbatim(c, cfg.NFTService) })
		nft.POST("/list", func(c *gin.Context) { forwardVerbatim(c, cfg.NFTService) })
	}

	// Gas & Price -- wallet_api.
	api.GET("/gas/:chain_id", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
	api.GET("/price/:symbol", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })
	api.GET("/prices", func(c *gin.Context) { forwardVerbatim(c, cfg.WalletAPI) })

	// DEX info -- real mainnet router addresses served by swap_service.
	api.GET("/dex/routers", func(c *gin.Context) { forwardVerbatim(c, cfg.SwapService) })
}

// =============================================================================
// Health & Ready
// =============================================================================

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now().UTC()})
}

func handleReady(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	ready := redisClient != nil && redisClient.Ping(ctx).Err() == nil
	c.JSON(http.StatusOK, gin.H{"ready": ready, "cache": ready})
}

// io.ReadAll is used implicitly via httputil; keep import.
var _ = io.ReadAll
