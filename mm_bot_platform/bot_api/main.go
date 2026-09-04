// TigerWallet Bot Platform API — real PostgreSQL + Redis bot management.
//
// Full bot lifecycle (create/start/stop/pause/delete), 18 bot types, 4
// subscription tiers (Free/Basic/Pro/Enterprise), fee configs, admin fee
// addresses, CEX/DEX connector management, per-user API keys, and admin user
// management + platform stats. JWT auth with role-based access control.
//
// No in-memory stubs, no fake data. Every record is persisted in PostgreSQL;
// rate-limit + session caches in Redis. Bot lifecycle transitions are persisted.
// Actual strategy execution is performed by the Rust bot_core (mm_bot_platform/
// bot_core) invoked over an internal dispatch queue; this API is the control
// plane (CRUD + lifecycle + observability), never fabricating trades.
package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/tigerwallet/wl-shared/wlgate"
	"golang.org/x/crypto/bcrypt"
)

// ---- roles / bot types / tiers (domain constants) ----

type UserRole string

const (
	RoleSuperAdmin   UserRole = "super_admin"
	RoleBotOperator  UserRole = "bot_operator"
	RoleFinanceAdmin UserRole = "finance_admin"
	RoleClient       UserRole = "client"
)

func validRoles() map[UserRole]bool {
	return map[UserRole]bool{RoleSuperAdmin: true, RoleBotOperator: true, RoleFinanceAdmin: true, RoleClient: true}
}

// 18 bot types (matches the Rust bot_core BotType enum).
var botTypes = []string{
	"market_maker", "liquidity_provider", "sniper", "front_run", "mev",
	"sandwich", "flash_loan", "cross_chain", "perp_hedge", "grid",
	"dca", "momentum", "mean_reversion", "scalping", "ai_trading",
	"signal", "arbitrage", "custom",
}

func validBotType(t string) bool {
	for _, b := range botTypes {
		if b == t {
			return true
		}
	}
	return false
}

// 4 subscription tiers.
var defaultTiers = []map[string]interface{}{
	{"id": "free", "name": "Free", "max_bots": 1, "max_dex": 1, "max_cex": 0, "latency_ms": 5000, "monthly_fee": "0"},
	{"id": "basic", "name": "Basic", "max_bots": 3, "max_dex": 5, "max_cex": 3, "latency_ms": 2000, "monthly_fee": "49"},
	{"id": "pro", "name": "Pro", "max_bots": 10, "max_dex": 15, "max_cex": 10, "latency_ms": 500, "monthly_fee": "299"},
	{"id": "enterprise", "name": "Enterprise", "max_bots": 50, "max_dex": 50, "max_cex": 30, "latency_ms": 100, "monthly_fee": "1499"},
}

func main() {
	cfg := loadCfg()
	if cfg.JWTSecret == "" {
		log.Fatalf("JWT_SECRET environment variable must be set")
	}
	gin.SetMode(gin.ReleaseMode)

	// Fail-closed SuperAdmin license gate (mirrors wl_shared/wlgate). The
	// canonical bot_api starts DEAD and only serves requests once a valid
	// license has been validated against the TigerWallet SuperAdmin control
	// plane, so SuperAdmin can suspend/revoke the product at any time.
	gate := wlgate.New()
	hbCtx, hbCancel := context.WithCancel(context.Background())
	defer hbCancel()
	go gate.HeartbeatLoop(hbCtx, cfg.ControlPlaneURL, cfg.ControlPlaneToken,
		cfg.LicenseKey, cfg.Product, cfg.InstanceID, cfg.HeartbeatInterval)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Printf("warning: postgres ping failed: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer rdb.Close()

	svc := &service{pg: pool, redis: rdb, jwt: cfg.JWTSecret, httpClient: &http.Client{Timeout: 15 * time.Second}}
	if err := svc.migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := svc.seed(ctx); err != nil {
		log.Printf("warning: seed: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "bot_api", "licensed": gate.IsAlive()})
	})
	r.GET("/api/v1/public/tiers", svc.publicTiers)

	// License gate on every authenticated route (fail-closed 503 when the
	// product is not authorized; per-fetcher disable by SuperAdmin).
	auth := r.Group("/api/v1", gate.Middleware(cfg.Product, wlgate.CategoryFetcher), svc.auth())
	{
		auth.POST("/auth/login", svc.login)
		auth.POST("/auth/register", svc.register)
		auth.POST("/auth/logout", svc.logout)

		auth.GET("/bots", svc.listBots)
		auth.POST("/bots", svc.createBot)
		auth.GET("/bots/:id", svc.getBot)
		auth.POST("/bots/:id/start", svc.startBot)
		auth.POST("/bots/:id/stop", svc.stopBot)
		auth.POST("/bots/:id/pause", svc.pauseBot)
		auth.DELETE("/bots/:id", svc.deleteBot)

		// Frontend-compat aliases (the web client.ts calls these paths).
		auth.GET("/bots/instances", svc.listBots)
		auth.POST("/bots/create", svc.createBot)
		auth.GET("/bots/me", svc.currentBotUser)
		auth.GET("/bots/users", svc.listBotUsers)
		auth.POST("/bots/users", svc.createBotUser)
		auth.DELETE("/bots/users/:id", svc.deleteBotUser)
		auth.GET("/bots/transactions", svc.listBotTransactions)

		auth.GET("/subscription", svc.getSubscription)
		auth.POST("/subscription", svc.createSubscription)

		auth.GET("/fees", svc.getFeeConfigs)
		// PUT /fees moved to the admin group: platform fee percentages are
		// operator-controlled, never user-writable.

		auth.GET("/cex", svc.listCEX)
		auth.POST("/cex", svc.addCEX)
		auth.DELETE("/cex/:id", svc.removeCEX)
		auth.GET("/dex", svc.listDEX)
		auth.POST("/dex", svc.addDEX)
		auth.DELETE("/dex/:id", svc.removeDEX)

		auth.GET("/keys", svc.listAPIKeys)
		auth.POST("/keys", svc.createAPIKey)
		auth.DELETE("/keys/:id", svc.deleteAPIKey)

		// Bots↔ProjectParty linkage: fetch market-making configs linked to
		// listed tokens from the ProjectParty backend. Lets the bots platform
		// auto-create market-maker bots for newly-listed tokens.
		auth.GET("/mm-configs", svc.getMMConfigs)

		// admin-only
		admin := auth.Group("/admin", svc.requireRole(string(RoleSuperAdmin), string(RoleFinanceAdmin)))
		{
			admin.GET("/users", svc.adminListUsers)
			admin.PUT("/users/:id/status", svc.adminUserStatus)
			admin.GET("/stats", svc.adminStats)
			admin.GET("/fee-addresses", svc.adminGetFeeAddresses)
			admin.POST("/fee-addresses", svc.adminSetFeeAddress)
			admin.DELETE("/fee-addresses/:id", svc.adminDeleteFeeAddress)
			admin.POST("/bots/:id/status", svc.adminBotStatus)
			admin.PUT("/fees", svc.updateFeeConfig)
			// Admin grants a paid subscription tier after payment is verified
			// out-of-band (invoice/fee-address receipt). Users can self-serve
			// only the free tier via POST /subscription.
			admin.POST("/subscriptions/grant", svc.adminGrantSubscription)
		}
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("bot_api service on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	c2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	srv.Shutdown(c2)
}

type config struct {
	Port, DBURL, RedisAddr, JWTSecret string
	// SuperAdmin control-plane license gate (fail-closed). The canonical
	// bot_api is controllable by TigerWallet SuperAdmin via the same control
	// plane as white-label products.
	ControlPlaneURL, ControlPlaneToken, LicenseKey, Product, InstanceID string
	HeartbeatInterval                                                   time.Duration
}

func loadCfg() config {
	g := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	gd := func(k string, d time.Duration) time.Duration {
		if v := os.Getenv(k); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				return d
			}
		}
		return d
	}
	return config{
		Port:              g("PORT", "8471"),
		DBURL:             g("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr:         g("REDIS_ADDR", "localhost:6379"),
		JWTSecret:         g("JWT_SECRET", ""),
		ControlPlaneURL:   g("TWO_PARTY_GATE_URL", ""),
		ControlPlaneToken: g("TWO_PARTY_GATE_TOKEN", ""),
		LicenseKey:        g("WL_LICENSE_KEY", ""),
		Product:           g("WL_PRODUCT", "bot_api"),
		InstanceID:        g("WL_INSTANCE_ID", "default"),
		HeartbeatInterval: gd("HEARTBEAT_INTERVAL", 30*time.Second),
	}
}

type service struct {
	pg         *pgxpool.Pool
	redis      *redis.Client
	jwt        string
	httpClient *http.Client // shared HTTP client for dispatch to bot_core
}

// ---- schema ----

func (s *service) migrate(ctx context.Context) error {
	_, err := s.pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS bot_users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_address  TEXT UNIQUE,
    email           TEXT UNIQUE,
    username        TEXT UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL DEFAULT 'client',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS bot_tiers (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    max_bots        INT NOT NULL DEFAULT 1,
    max_dex         INT NOT NULL DEFAULT 1,
    max_cex         INT NOT NULL DEFAULT 0,
    latency_ms      INT NOT NULL DEFAULT 5000,
    monthly_fee     TEXT NOT NULL DEFAULT '0'
);
CREATE TABLE IF NOT EXISTS bot_subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES bot_users(id) ON DELETE CASCADE,
    tier_id         TEXT NOT NULL REFERENCES bot_tiers(id),
    status          TEXT NOT NULL DEFAULT 'active',
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ,
    UNIQUE(user_id)
);
CREATE TABLE IF NOT EXISTS bots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES bot_users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    bot_type        TEXT NOT NULL,
    config          JSONB NOT NULL DEFAULT '{}'::jsonb,
    status          TEXT NOT NULL DEFAULT 'stopped',
    exchange        TEXT,
    pair            TEXT,
    stats           JSONB NOT NULL DEFAULT '{"trades":0,"volume":"0","pnl":"0","wins":0,"losses":0}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS fee_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fee_type        TEXT NOT NULL,
    asset           TEXT,
    fee_percent     NUMERIC(10,6) NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(fee_type, asset)
);
CREATE TABLE IF NOT EXISTS admin_fee_addresses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label           TEXT NOT NULL,
    address         TEXT NOT NULL,
    chain_id        BIGINT NOT NULL DEFAULT 1,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(address, chain_id)
);
CREATE TABLE IF NOT EXISTS bot_cex_connections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES bot_users(id) ON DELETE CASCADE,
    exchange        TEXT NOT NULL,
    api_key_encrypted TEXT NOT NULL,
    api_secret_encrypted TEXT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS bot_dex_connections (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES bot_users(id) ON DELETE CASCADE,
    dex                     TEXT NOT NULL,
    chain_id                BIGINT NOT NULL,
    rpc_url                 TEXT,
    wallet_seed_encrypted   TEXT NOT NULL DEFAULT '',
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS bot_api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES bot_users(id) ON DELETE CASCADE,
    key_hash        TEXT NOT NULL,
    label           TEXT NOT NULL,
    permissions     JSONB NOT NULL DEFAULT '["read"]'::jsonb,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS bot_sessions (
    token_hash      TEXT PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES bot_users(id) ON DELETE CASCADE,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS bot_trades (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id          UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    side            INT NOT NULL,
    price           DOUBLE PRECISION NOT NULL,
    amount          DOUBLE PRECISION NOT NULL,
    pnl             DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_bot_trades_bot ON bot_trades(bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_trades_created ON bot_trades(created_at DESC);
`)
	if err != nil {
		return err
	}
	// Add wallet_seed_encrypted column to existing bot_dex_connections tables
	// (idempotent — fails silently if column already exists).
	_, _ = s.pg.Exec(context.Background(),
		`ALTER TABLE bot_dex_connections ADD COLUMN IF NOT EXISTS wallet_seed_encrypted TEXT NOT NULL DEFAULT ''`)
	// Add bot_executions table (written by bot_core Rust execution plane).
	_, _ = s.pg.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS bot_executions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id          TEXT NOT NULL,
    strategy        TEXT NOT NULL,
    action          TEXT NOT NULL,
    detail          TEXT NOT NULL DEFAULT '',
    latency_us      BIGINT NOT NULL DEFAULT 0,
    success         BOOLEAN NOT NULL DEFAULT TRUE,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	_, _ = s.pg.Exec(context.Background(),
		`CREATE INDEX IF NOT EXISTS idx_bot_executions_bot ON bot_executions(bot_id)`)
	return nil
}

// seed the 4 default tiers if the table is empty.
func (s *service) seed(ctx context.Context) error {
	var n int
	if err := s.pg.QueryRow(ctx, `SELECT COUNT(*) FROM bot_tiers`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	for _, t := range defaultTiers {
		if _, err := s.pg.Exec(ctx, `INSERT INTO bot_tiers (id,name,max_bots,max_dex,max_cex,latency_ms,monthly_fee) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			t["id"], t["name"], t["max_bots"], t["max_dex"], t["max_cex"], t["latency_ms"], t["monthly_fee"]); err != nil {
			return err
		}
	}
	return nil
}

// ---- auth ----

func (s *service) parseJWT(tokenStr string) (string, string, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwt), nil
	})
	if err != nil || !tok.Valid {
		return "", "", errors.New("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("invalid claims")
	}
	uid, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	if uid == "" {
		return "", "", errors.New("missing subject")
	}
	return uid, role, nil
}

func (s *service) auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if len(h) < 8 || h[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		uid, role, err := s.parseJWT(h[7:])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("user_id", uid)
		c.Set("role", role)
		c.Next()
	}
}

func (s *service) requireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
	}
}

func issueJWT(secret, uid, role string) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  uid,
		"role": role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})
	return tok.SignedString([]byte(secret))
}

// ---- auth handlers ----

func (s *service) login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
		return
	}
	var id, hash, role string
	var active bool
	err := s.pg.QueryRow(c.Request.Context(),
		`SELECT id, password_hash, role, is_active FROM bot_users WHERE username=$1`, req.Username).
		Scan(&id, &hash, &role, &active)
	if err != nil || !active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token, err := issueJWT(s.jwt, id, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	// persist session
	th := hashToken(token)
	s.pg.Exec(c.Request.Context(), `INSERT INTO bot_sessions (token_hash,user_id,expires_at) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
		th, id, time.Now().Add(24*time.Hour))
	c.JSON(http.StatusOK, gin.H{"token": token, "user_id": id, "role": role})
}

func (s *service) register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Wallet   string `json:"wallet_address"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
		return
	}
	role := req.Role
	if role == "" {
		role = string(RoleClient)
	}
	if !validRoles()[UserRole(role)] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	var id string
	err = s.pg.QueryRow(c.Request.Context(),
		`INSERT INTO bot_users (username,password_hash,email,wallet_address,role) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		req.Username, string(hash), strOrNull(req.Email), strOrNull(req.Wallet), role).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username/email/wallet already exists"})
		return
	}
	token, _ := issueJWT(s.jwt, id, role)
	c.JSON(http.StatusCreated, gin.H{"token": token, "user_id": id, "role": role})
}

func (s *service) logout(c *gin.Context) {
	h := c.GetHeader("Authorization")
	if len(h) >= 7 {
		s.pg.Exec(c.Request.Context(), `DELETE FROM bot_sessions WHERE token_hash=$1`, hashToken(h[7:]))
	}
	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

// ---- bot handlers ----

func (s *service) listBots(c *gin.Context) {
	uid := c.GetString("user_id")
	role := c.GetString("role")
	rows, err := s.pg.Query(c.Request.Context(), `
SELECT id,name,bot_type,config,status,exchange,pair,stats,created_at,updated_at,user_id FROM bots ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		b, owner := scanBot(rows)
		if role != string(RoleSuperAdmin) && owner != uid {
			continue
		}
		out = append(out, b)
	}
	c.JSON(http.StatusOK, gin.H{"bots": out, "count": len(out)})
}

func (s *service) createBot(c *gin.Context) {
	uid := c.GetString("user_id")
	var req struct {
		Name     string          `json:"name"`
		BotType  string          `json:"bot_type"`
		Config   json.RawMessage `json:"config"`
		Exchange string          `json:"exchange"`
		Pair     string          `json:"pair"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Name == "" || req.BotType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and bot_type required"})
		return
	}
	if !validBotType(req.BotType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bot_type", "valid_types": botTypes})
		return
	}
	// enforce subscription bot limit (client/operator)
	if c.GetString("role") != string(RoleSuperAdmin) {
		var maxBots int
		err := s.pg.QueryRow(c.Request.Context(), `
SELECT t.max_bots FROM bot_subscriptions sub JOIN bot_tiers t ON t.id=sub.tier_id WHERE sub.user_id=$1 AND sub.status='active'`, uid).Scan(&maxBots)
		if err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "no active subscription"})
			return
		}
		var count int
		s.pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM bots WHERE user_id=$1`, uid).Scan(&count)
		if count >= maxBots {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "bot limit reached for your tier", "max": maxBots})
			return
		}
	}
	cfg := req.Config
	if len(cfg) == 0 {
		cfg = json.RawMessage("{}")
	}
	var id string
	err := s.pg.QueryRow(c.Request.Context(), `
INSERT INTO bots (user_id,name,bot_type,config,status,exchange,pair) VALUES ($1,$2,$3,$4,'stopped',$5,$6) RETURNING id`,
		uid, req.Name, req.BotType, cfg, strOrNull(req.Exchange), strOrNull(req.Pair)).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "stopped", "message": "bot created; call /bots/" + id + "/start to launch"})
}

func (s *service) getBot(c *gin.Context) {
	b, ok := s.fetchOwnedBot(c, c.Param("id"))
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"bot": b})
}

func (s *service) startBot(c *gin.Context) {
	bot, ok := s.fetchOwnedBot(c, c.Param("id"))
	if !ok {
		return
	}
	// Fail-closed: validate the full dispatch payload BEFORE flipping the
	// status. A bot that cannot actually execute (missing exchange creds,
	// missing DEX wallet, unsupported type) is never shown as "running".
	userID := bot["user_id"].(string)
	if _, err := s.buildStartPayload(c.Request.Context(), c.Param("id"), userID); err != nil {
		if errors.Is(err, errSignalOnlyStrategy) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":            "bot type has no execution runner yet",
				"bot_type":         bot["bot_type"],
				"executable_types": executableBotTypes(),
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot start bot: " + err.Error()})
		return
	}
	if !s.setBotStatusOwned(c, c.Param("id"), "running") {
		return
	}
	// Dispatch real start to the Rust bot_core execution plane.
	s.dispatchBotCore(c, c.Param("id"), "start")
}

func (s *service) stopBot(c *gin.Context) {
	if !s.setBotStatusOwned(c, c.Param("id"), "stopped") {
		return
	}
	s.dispatchBotCore(c, c.Param("id"), "stop")
}

func (s *service) pauseBot(c *gin.Context) {
	if !s.setBotStatusOwned(c, c.Param("id"), "paused") {
		return
	}
	s.dispatchBotCore(c, c.Param("id"), "pause")
}

// botCoreURL returns the Rust bot_core execution plane endpoint. Defaults to
// http://localhost:8472 (overridable via BOT_CORE_URL env).
func botCoreURL() string {
	u := os.Getenv("BOT_CORE_URL")
	if u == "" {
		return "http://localhost:8472"
	}
	return u
}

// projectPartyURL returns the ProjectParty backend URL (for fetching
// market-making configs linked to listed tokens). Defaults to
// http://localhost:8106 (overridable via PROJECT_PARTY_URL env).
func projectPartyURL() string {
	u := os.Getenv("PROJECT_PARTY_URL")
	if u == "" {
		return "http://localhost:8106"
	}
	return u
}

// dispatchBotCore sends a real start/stop/pause/resume command to the Rust
// bot_core execution plane at /dispatch/<action>. For "start" it fetches the
// bot config + decrypted CEX/DEX credentials from DB and builds the proper
// StartReq tagged-enum payload that bot_core expects (MarketMaker/Arbitrage/
// Sniper). For stop/pause it sends a simple BotIdReq {bot_id}. Failures are
// logged but do NOT fail the API request (the DB status update already
// succeeded; the execution plane is best-effort and may be down for
// maintenance). This is the same async-dispatch pattern as canonical wallet
// control planes.
func (s *service) dispatchBotCore(c *gin.Context, botID, action string) {
	var body []byte
	var err error
	if action == "start" {
		body, err = s.buildStartPayload(c.Request.Context(), botID, c.GetString("user_id"))
		if err != nil {
			log.Printf("dispatchBotCore start %s: failed to build payload: %v", botID, err)
			return
		}
	} else {
		// stop / pause / resume — simple bot_id payload
		body, _ = json.Marshal(map[string]string{"bot_id": botID})
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST",
		botCoreURL()+"/dispatch/"+action, bytes.NewReader(body))
	if err != nil {
		log.Printf("dispatchBotCore: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("dispatchBotCore %s %s: %v (bot_core unreachable)", action, botID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("dispatchBotCore %s %s: bot_core returned %d", action, botID, resp.StatusCode)
	}
}

// buildStartPayload fetches the bot row + decrypted CEX/DEX credentials from
// DB and builds the StartReq tagged-enum JSON that bot_core's dispatch_start
// endpoint expects. Maps bot_type -> strategy kind:
//
// market_maker -> MarketMaker, arbitrage -> Arbitrage, sniper -> Sniper,
// grid/dca/momentum/mean_reversion/scalping -> the corresponding CEX runners
// (bot_core serde tags).
//
// Bot types without a real execution runner in bot_core (ai_trading/signal/
// cross_chain/flash_loan/sandwich/front_run/mev/custom) return the
// errSignalOnlyStrategy sentinel so startBot fails closed
// with a 400 — the DB status is never flipped to "running" for a bot that
// cannot actually execute. No fake execution, ever.
func (s *service) buildStartPayload(ctx context.Context, botID, userID string) ([]byte, error) {
	var botType, exchange, pair string
	var configJSON json.RawMessage
	err := s.pg.QueryRow(ctx,
		`SELECT bot_type, config, exchange, pair FROM bots WHERE id=$1 AND user_id=$2`,
		botID, userID).Scan(&botType, &configJSON, &exchange, &pair)
	if err != nil {
		return nil, fmt.Errorf("fetch bot: %w", err)
	}

	// Parse optional config fields (order_size, spread_bps, threshold_bps, etc.)
	var cfg map[string]any
	if len(configJSON) > 0 {
		_ = json.Unmarshal(configJSON, &cfg)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	switch botType {
	case "market_maker":
		// Fetch decrypted CEX credentials for this user + exchange
		cexCreds, err := s.fetchCEXCreds(ctx, userID, exchange)
		if err != nil {
			return nil, fmt.Errorf("cex creds: %w", err)
		}
		payload := map[string]any{
			"kind":       "marketmaker",
			"bot_id":     botID,
			"exchange":   cexCreds.exchange,
			"api_key":    cexCreds.apiKey,
			"secret_key": cexCreds.apiSecret,
			"symbol":     pair,
			"order_size": cfgFloat(cfg, "order_size", 0.01),
			"spread_bps": cfgFloat(cfg, "spread_bps", 10),
		}
		if baseURL, ok := cfg["base_url"].(string); ok && baseURL != "" {
			payload["base_url"] = baseURL
		}
		if pi, ok := cfg["poll_interval_ms"].(float64); ok {
			payload["poll_interval_ms"] = int64(pi)
		}
		if pp, ok := cfg["passphrase"].(string); ok && pp != "" {
			payload["passphrase"] = pp
		}
		return json.Marshal(payload)

	case "arbitrage":
		cexCreds, err := s.fetchCEXCreds(ctx, userID, exchange)
		if err != nil {
			return nil, fmt.Errorf("cex creds: %w", err)
		}
		dexReq, err := s.buildDexReq(ctx, userID, cfg)
		if err != nil {
			return nil, fmt.Errorf("dex config: %w", err)
		}
		payload := map[string]any{
			"kind":          "arbitrage",
			"bot_id":        botID,
			"exchange":      cexCreds.exchange,
			"api_key":       cexCreds.apiKey,
			"secret_key":    cexCreds.apiSecret,
			"symbol":        pair,
			"threshold_bps": cfgFloat(cfg, "threshold_bps", 50),
			"dex_req":       dexReq,
		}
		if baseURL, ok := cfg["base_url"].(string); ok && baseURL != "" {
			payload["base_url"] = baseURL
		}
		if pi, ok := cfg["poll_interval_ms"].(float64); ok {
			payload["poll_interval_ms"] = int64(pi)
		}
		if pp, ok := cfg["passphrase"].(string); ok && pp != "" {
			payload["passphrase"] = pp
		}
		return json.Marshal(payload)

	case "sniper":
		dexReq, err := s.buildDexReq(ctx, userID, cfg)
		if err != nil {
			return nil, fmt.Errorf("dex config: %w", err)
		}
		mempoolURL := ""
		if v, ok := cfg["mempool_url"].(string); ok {
			mempoolURL = v
		}
		if mempoolURL == "" {
			return nil, errors.New("sniper requires mempool_url in config")
		}
		payload := map[string]any{
			"kind":        "sniper",
			"bot_id":      botID,
			"dex_req":     dexReq,
			"mempool_url": mempoolURL,
		}
		if pi, ok := cfg["poll_interval_ms"].(float64); ok {
			payload["poll_interval_ms"] = int64(pi)
		}
		if mta, ok := cfg["min_target_amount"].(float64); ok {
			payload["min_target_amount"] = int64(mta)
		}
		return json.Marshal(payload)

	case "grid", "dca", "momentum", "mean_reversion", "scalping":
		cexCreds, err := s.fetchCEXCreds(ctx, userID, exchange)
		if err != nil {
			return nil, fmt.Errorf("cex creds: %w", err)
		}
		kind := botType
		if kind == "mean_reversion" {
			kind = "meanreversion" // bot_core serde tag
		}
		payload := map[string]any{
			"kind":       kind,
			"bot_id":     botID,
			"exchange":   cexCreds.exchange,
			"api_key":    cexCreds.apiKey,
			"secret_key": cexCreds.apiSecret,
			"symbol":     pair,
		}
		switch botType {
		case "grid":
			payload["grid_count"] = int64(cfgFloat(cfg, "grid_count", 10))
			payload["grid_spacing_pct"] = cfgFloat(cfg, "grid_spacing_pct", 1.0)
			payload["order_size_usd"] = cfgFloat(cfg, "order_size_usd", 100)
		case "dca":
			payload["buy_interval_hours"] = int64(cfgFloat(cfg, "buy_interval_hours", 24))
			payload["buy_amount_usd"] = cfgFloat(cfg, "buy_amount_usd", 50)
			payload["max_positions"] = int64(cfgFloat(cfg, "max_positions", 30))
		case "momentum":
			payload["order_size"] = cfgFloat(cfg, "order_size", 0.01)
			payload["lookback_period"] = int64(cfgFloat(cfg, "lookback_period", 20))
			payload["entry_threshold"] = cfgFloat(cfg, "entry_threshold", 0.02)
			payload["exit_threshold"] = cfgFloat(cfg, "exit_threshold", 0.005)
		case "mean_reversion":
			payload["order_size"] = cfgFloat(cfg, "order_size", 0.01)
			payload["lookback_period"] = int64(cfgFloat(cfg, "lookback_period", 20))
			payload["std_dev_threshold"] = cfgFloat(cfg, "std_dev_threshold", 2.0)
		case "scalping":
			payload["order_size"] = cfgFloat(cfg, "order_size", 0.01)
			payload["profit_target_pct"] = cfgFloat(cfg, "profit_target_pct", 0.3)
			payload["stop_loss_pct"] = cfgFloat(cfg, "stop_loss_pct", 0.5)
		}
		if baseURL, ok := cfg["base_url"].(string); ok && baseURL != "" {
			payload["base_url"] = baseURL
		}
		if pp, ok := cfg["passphrase"].(string); ok && pp != "" {
			payload["passphrase"] = pp
		}
		if pi, ok := cfg["poll_interval_ms"].(float64); ok {
			payload["poll_interval_ms"] = int64(pi)
		}
		return json.Marshal(payload)

	case "perp_hedge":
		cexCreds, err := s.fetchCEXCreds(ctx, userID, exchange)
		if err != nil {
			return nil, fmt.Errorf("cex creds: %w", err)
		}
		payload := map[string]any{
			"kind":                    "perp_hedge",
			"bot_id":                  botID,
			"exchange":                cexCreds.exchange,
			"api_key":                 cexCreds.apiKey,
			"secret_key":              cexCreds.apiSecret,
			"symbol":                  pair,
			"spot_notional_usd":       cfgFloat(cfg, "spot_notional_usd", 1000),
			"hedge_ratio":             cfgFloat(cfg, "hedge_ratio", 1.0),
			"rebalance_threshold_pct": cfgFloat(cfg, "rebalance_threshold_pct", 0.05),
		}
		if baseURL, ok := cfg["base_url"].(string); ok && baseURL != "" {
			payload["base_url"] = baseURL
		}
		if pp, ok := cfg["passphrase"].(string); ok && pp != "" {
			payload["passphrase"] = pp
		}
		if pi, ok := cfg["poll_interval_ms"].(float64); ok {
			payload["poll_interval_ms"] = int64(pi)
		}
		return json.Marshal(payload)

	case "liquidity_provider":
		// Reuses the per-user DEX connection (rpc_url + decrypted wallet seed);
		// the flattened payload matches bot_core DexAddLiquidityRequest exactly.
		dexReq, err := s.buildDexReq(ctx, userID, cfg)
		if err != nil {
			return nil, fmt.Errorf("dex config: %w", err)
		}
		tokenA := cfgString(cfg, "token_a", cfgString(cfg, "token_in", ""))
		tokenB := cfgString(cfg, "token_b", cfgString(cfg, "token_out", ""))
		if tokenA == "" || tokenB == "" {
			return nil, errors.New("liquidity_provider requires token_a and token_b in config")
		}
		payload := map[string]any{
			"kind":   "liquidity_provider",
			"bot_id": botID,
			"liq_req": map[string]any{
				"rpc_url":      dexReq["rpc_url"],
				"chain_id":     dexReq["chain_id"],
				"private_key":  dexReq["private_key"],
				"router":       cfgString(cfg, "router", ""),
				"token_a":      tokenA,
				"token_b":      tokenB,
				"amount_a":     cfgFloat(cfg, "amount_a", cfgFloat(cfg, "amount_in", 0)),
				"amount_b":     cfgFloat(cfg, "amount_b", 0),
				"amount_a_min": cfgFloat(cfg, "amount_a_min", 0),
				"amount_b_min": cfgFloat(cfg, "amount_b_min", 0),
			},
		}
		if ai, ok := cfg["add_interval_hours"].(float64); ok {
			payload["add_interval_hours"] = int64(ai)
		}
		if ma, ok := cfg["max_adds"].(float64); ok {
			payload["max_adds"] = int64(ma)
		}
		if pi, ok := cfg["poll_interval_ms"].(float64); ok {
			payload["poll_interval_ms"] = int64(pi)
		}
		return json.Marshal(payload)

	default:
		// Bot types without a real execution runner in bot_core
		// (ai_trading/signal/cross_chain/flash_loan/sandwich/front_run/mev/
		// custom). Return a sentinel so the caller fails closed (start
		// rejected, never faked).
		return nil, errSignalOnlyStrategy
	}
}

// executableBotTypes lists the bot types with a real execution runner in
// bot_core (Rust). All other types fail closed at start time.
func executableBotTypes() []string {
	return []string{"market_maker", "arbitrage", "sniper", "grid", "dca", "momentum", "mean_reversion", "scalping", "perp_hedge", "liquidity_provider"}
}

// errSignalOnlyStrategy is a sentinel indicating the bot type is a
// signal-generation-only strategy with no real execution runner in bot_core.
// dispatchBotCore handles this by skipping the dispatch (no fake execution).
var errSignalOnlyStrategy = errors.New("bot type is signal-only (no execution runner)")

type cexCreds struct {
	exchange  string
	apiKey    string
	apiSecret string
}

// fetchCEXCreds fetches + decrypts the user's CEX API credentials for a given
// exchange from the bot_cex_connections table.
func (s *service) fetchCEXCreds(ctx context.Context, userID, exchange string) (cexCreds, error) {
	var encKey, encSecret, ex string
	err := s.pg.QueryRow(ctx,
		`SELECT exchange, api_key_encrypted, api_secret_encrypted FROM bot_cex_connections WHERE user_id=$1 AND exchange=$2 ORDER BY created_at DESC LIMIT 1`,
		userID, exchange).Scan(&ex, &encKey, &encSecret)
	if err != nil {
		return cexCreds{}, fmt.Errorf("no cex connection for exchange %s: %w", exchange, err)
	}
	apiKey, err := decryptSecret(encKey)
	if err != nil {
		return cexCreds{}, fmt.Errorf("decrypt api_key: %w", err)
	}
	apiSecret, err := decryptSecret(encSecret)
	if err != nil {
		return cexCreds{}, fmt.Errorf("decrypt api_secret: %w", err)
	}
	return cexCreds{exchange: ex, apiKey: apiKey, apiSecret: apiSecret}, nil
}

// buildDexReq builds a DexSwapRequest from the bot config + user's DEX
// connector (decrypted wallet seed). Field names match bot_core's
// DexSwapRequest struct: rpc_url, chain_id, private_key, router, token_in,
// token_out, amount_in (f64), amount_out_min (f64).
func (s *service) buildDexReq(ctx context.Context, userID string, cfg map[string]any) (map[string]any, error) {
	chainID := int64(0)
	if v, ok := cfg["chain_id"].(float64); ok {
		chainID = int64(v)
	}
	if chainID == 0 {
		if chain, ok := cfg["chain"].(string); ok {
			chainID = chainIDForChain(chain)
		}
	}
	if chainID == 0 {
		return nil, errors.New("dex requires 'chain_id' or 'chain' in config")
	}
	var encSeed, rpcURL string
	var dbChainID int64
	err := s.pg.QueryRow(ctx,
		`SELECT chain_id, rpc_url, wallet_seed_encrypted FROM bot_dex_connections WHERE user_id=$1 AND chain_id=$2 ORDER BY created_at DESC LIMIT 1`,
		userID, chainID).Scan(&dbChainID, &rpcURL, &encSeed)
	if err != nil {
		return nil, fmt.Errorf("no dex connection for chain_id %d: %w", chainID, err)
	}
	if encSeed == "" {
		return nil, fmt.Errorf("dex connection for chain_id %d has no wallet seed (re-add with wallet_seed)", chainID)
	}
	seedHex, err := decryptSecret(encSeed)
	if err != nil {
		return nil, fmt.Errorf("decrypt wallet_seed: %w", err)
	}
	return map[string]any{
		"rpc_url":        rpcURL,
		"chain_id":       dbChainID,
		"private_key":    seedHex,
		"router":         cfgString(cfg, "router", ""),
		"token_in":       cfgString(cfg, "token_in", ""),
		"token_out":      cfgString(cfg, "token_out", ""),
		"amount_in":      cfgFloat(cfg, "amount_in", 0),
		"amount_out_min": cfgFloat(cfg, "amount_out_min", 0),
	}, nil
}

// chainIDForChain maps a chain name to its canonical chain id.
func chainIDForChain(chain string) int64 {
	switch strings.ToLower(chain) {
	case "ethereum", "eth":
		return 1
	case "bsc", "binance":
		return 56
	case "polygon":
		return 137
	case "arbitrum":
		return 42161
	case "optimism":
		return 10
	case "base":
		return 8453
	case "avalanche":
		return 43114
	default:
		return 1
	}
}

func cfgFloat(m map[string]any, key string, def float64) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return def
}

func cfgString(m map[string]any, key string, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}

func (s *service) deleteBot(c *gin.Context) {
	uid := c.GetString("user_id")
	role := c.GetString("role")
	id := c.Param("id")
	tag, err := s.pg.Exec(c.Request.Context(), `DELETE FROM bots WHERE id=$1`+ownerClause(role), id, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

// ---- frontend-compat handlers (aliases the web client.ts calls) ----

// currentBotUser returns the authenticated user's profile (GET /bots/me).
func (s *service) currentBotUser(c *gin.Context) {
	uid := c.GetString("user_id")
	var id, username, role string
	var email sql.NullString
	var active bool
	var ct time.Time
	err := s.pg.QueryRow(c.Request.Context(),
		`SELECT id,username,email,role,is_active,created_at FROM bot_users WHERE id=$1`, uid).
		Scan(&id, &username, &email, &role, &active, &ct)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": id, "username": username, "email": email.String,
		"role": role, "is_active": active, "created_at": ct,
		"address": "", "name": username,
	})
}

// listBotUsers returns users visible to the caller (admin: all; others: just self).
func (s *service) listBotUsers(c *gin.Context) {
	uid := c.GetString("user_id")
	role := c.GetString("role")
	q := `SELECT id,username,email,role,is_active,created_at FROM bot_users ORDER BY created_at DESC LIMIT 500`
	if role != string(RoleSuperAdmin) && role != string(RoleFinanceAdmin) {
		q = `SELECT id,username,email,role,is_active,created_at FROM bot_users WHERE id=$1`
	}
	var rows pgx.Rows
	var err error
	if role != string(RoleSuperAdmin) && role != string(RoleFinanceAdmin) {
		rows, err = s.pg.Query(c.Request.Context(), q, uid)
	} else {
		rows, err = s.pg.Query(c.Request.Context(), q)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, username, role string
		var email sql.NullString
		var active bool
		var ct time.Time
		rows.Scan(&id, &username, &email, &role, &active, &ct)
		out = append(out, gin.H{
			"id": id, "username": username, "name": username, "email": email.String,
			"role": role, "is_active": active, "address": "", "created_at": ct,
		})
	}
	c.JSON(http.StatusOK, gin.H{"users": out, "count": len(out)})
}

// createBotUser creates a new bot-platform user (admin/operator only).
func (s *service) createBotUser(c *gin.Context) {
	role := c.GetString("role")
	if role != string(RoleSuperAdmin) && role != string(RoleFinanceAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}
	var req struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Role    string `json:"role"`
		Address string `json:"address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	if req.Role == "" {
		req.Role = string(RoleClient)
	}
	if !validRoles()[UserRole(req.Role)] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	var id string
	err := s.pg.QueryRow(c.Request.Context(),
		`INSERT INTO bot_users (id,username,email,role,is_active) VALUES (gen_random_uuid(),$1,$2,$3,true) RETURNING id`,
		req.Name, strOrNull(req.Email), req.Role).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": id, "username": req.Name, "name": req.Name, "email": req.Email,
		"role": req.Role, "is_active": true, "address": req.Address,
	})
}

// deleteBotUser removes a bot-platform user (admin only).
func (s *service) deleteBotUser(c *gin.Context) {
	role := c.GetString("role")
	if role != string(RoleSuperAdmin) && role != string(RoleFinanceAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}
	id := c.Param("id")
	tag, err := s.pg.Exec(c.Request.Context(), `DELETE FROM bot_users WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

// listBotTransactions returns recent trades across the caller's bots.
func (s *service) listBotTransactions(c *gin.Context) {
	uid := c.GetString("user_id")
	role := c.GetString("role")
	q := `SELECT id,bot_id,side,price,amount,pnl,created_at FROM bot_trades ORDER BY created_at DESC LIMIT 200`
	var rows pgx.Rows
	var err error
	if role != string(RoleSuperAdmin) && role != string(RoleFinanceAdmin) {
		q = `SELECT t.id,t.bot_id,t.side,t.price,t.amount,t.pnl,t.created_at FROM bot_trades t JOIN bots b ON b.id=t.bot_id WHERE b.user_id=$1 ORDER BY t.created_at DESC LIMIT 200`
		rows, err = s.pg.Query(c.Request.Context(), q, uid)
	} else {
		rows, err = s.pg.Query(c.Request.Context(), q)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, botID string
		var side int
		var price, amount, pnl float64
		var ct time.Time
		rows.Scan(&id, &botID, &side, &price, &amount, &pnl, &ct)
		out = append(out, gin.H{
			"id": id, "bot_id": botID, "side": side, "price": price,
			"amount": amount, "profit": pnl, "status": "success",
			"timestamp": ct.UnixMilli(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"transactions": out, "count": len(out)})
}

// fetchOwnedBot loads a bot enforcing ownership: non-admin callers may only
// read their own bots. Writes a 404 (indistinguishable from not-found, so
// existence is not leaked) on failure.
func (s *service) fetchOwnedBot(c *gin.Context, id string) (gin.H, bool) {
	b, ok := s.fetchBot(c, id)
	if !ok {
		return nil, false
	}
	if r := c.GetString("role"); r == string(RoleSuperAdmin) || r == string(RoleFinanceAdmin) {
		return b, true
	}
	owner, _ := b["user_id"].(string)
	if owner != c.GetString("user_id") {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return nil, false
	}
	return b, true
}

// setBotStatusOwned transitions a bot's status with ownership enforcement:
// non-admin callers can only transition bots they own. Returns true when the
// transition succeeded (response already written either way).
func (s *service) setBotStatusOwned(c *gin.Context, id, status string) bool {
	uid := c.GetString("user_id")
	role := c.GetString("role")
	q := `UPDATE bots SET status=$1, updated_at=now() WHERE id=$2`
	args := []any{status, id}
	if role != string(RoleSuperAdmin) && role != string(RoleFinanceAdmin) {
		q += ` AND user_id=$3`
		args = append(args, uid)
	}
	tag, err := s.pg.Exec(c.Request.Context(), q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return false
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return false
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": status})
	return true
}

// ---- subscription handlers ----

func (s *service) getSubscription(c *gin.Context) {
	uid := c.GetString("user_id")
	var subID, tierID, status string
	var started, expires sql.NullTime
	err := s.pg.QueryRow(c.Request.Context(),
		`SELECT id, tier_id, status, started_at, expires_at FROM bot_subscriptions WHERE user_id=$1`, uid).
		Scan(&subID, &tierID, &status, &started, &expires)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"subscription": nil, "tiers": defaultTiers})
		return
	}
	var tierName string
	var maxBots, maxDex, maxCex, lat int
	var fee string
	s.pg.QueryRow(c.Request.Context(), `SELECT name,max_bots,max_dex,max_cex,latency_ms,monthly_fee FROM bot_tiers WHERE id=$1`, tierID).
		Scan(&tierName, &maxBots, &maxDex, &maxCex, &lat, &fee)
	c.JSON(http.StatusOK, gin.H{"subscription": gin.H{
		"id": subID, "tier_id": tierID, "tier_name": tierName, "status": status,
		"started_at": started.Time, "expires_at": expires.Time,
		"limits":      gin.H{"max_bots": maxBots, "max_dex": maxDex, "max_cex": maxCex, "latency_ms": lat},
		"monthly_fee": fee,
	}})
}

// createSubscription is self-serve for the FREE tier only. Paid tiers carry a
// monthly_fee and are granted exclusively by an admin via
// POST /admin/subscriptions/grant after payment verification — a user must
// never be able to upgrade themselves for free.
func (s *service) createSubscription(c *gin.Context) {
	uid := c.GetString("user_id")
	var req struct {
		TierID string `json:"tier_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TierID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tier_id required"})
		return
	}
	var fee string
	err := s.pg.QueryRow(c.Request.Context(),
		`SELECT monthly_fee FROM bot_tiers WHERE id=$1`, req.TierID).Scan(&fee)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown tier"})
		return
	}
	if fee != "0" && fee != "0.0" && fee != "" {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": "paid tiers are activated by the platform after payment verification",
			"tier":  req.TierID, "monthly_fee": fee,
		})
		return
	}
	s.upsertSubscription(c, uid, req.TierID)
}

// adminGrantSubscription activates any tier for a target user. Admin-only
// (route-level role gate); payment is verified out-of-band before granting.
func (s *service) adminGrantSubscription(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
		TierID string `json:"tier_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == "" || req.TierID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id and tier_id required"})
		return
	}
	var valid bool
	s.pg.QueryRow(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM bot_tiers WHERE id=$1)`, req.TierID).Scan(&valid)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown tier"})
		return
	}
	s.upsertSubscription(c, req.UserID, req.TierID)
}

func (s *service) upsertSubscription(c *gin.Context, userID, tierID string) {
	_, err := s.pg.Exec(c.Request.Context(), `
INSERT INTO bot_subscriptions (user_id,tier_id,status,expires_at) VALUES ($1,$2,'active',now()+interval '30 days')
ON CONFLICT (user_id) DO UPDATE SET tier_id=excluded.tier_id, status='active', started_at=now(), expires_at=now()+interval '30 days'`,
		userID, tierID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "active", "tier_id": tierID, "user_id": userID})
}

// ---- fee handlers ----

func (s *service) getFeeConfigs(c *gin.Context) {
	rows, err := s.pg.Query(c.Request.Context(), `SELECT id,fee_type,asset,fee_percent,is_active FROM fee_configs ORDER BY fee_type`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, ft string
		var asset sql.NullString
		var pct float64
		var active bool
		rows.Scan(&id, &ft, &asset, &pct, &active)
		out = append(out, gin.H{"id": id, "fee_type": ft, "asset": asset.String, "fee_percent": pct, "is_active": active})
	}
	c.JSON(http.StatusOK, gin.H{"fees": out, "count": len(out)})
}

func (s *service) updateFeeConfig(c *gin.Context) {
	var req struct {
		FeeType    string  `json:"fee_type"`
		Asset      string  `json:"asset"`
		FeePercent float64 `json:"fee_percent"`
		IsActive   *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FeeType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fee_type required"})
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	_, err := s.pg.Exec(c.Request.Context(), `
INSERT INTO fee_configs (fee_type,asset,fee_percent,is_active) VALUES ($1,$2,$3,$4)
ON CONFLICT (fee_type,asset) DO UPDATE SET fee_percent=excluded.fee_percent, is_active=excluded.is_active, updated_at=now()`,
		req.FeeType, strOrNull(req.Asset), req.FeePercent, active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated", "fee_type": req.FeeType})
}

// ---- CEX/DEX connector handlers ----

func (s *service) listCEX(c *gin.Context) {
	uid := c.GetString("user_id")
	rows, err := s.pg.Query(c.Request.Context(), `SELECT id,exchange,is_active,created_at FROM bot_cex_connections WHERE user_id=$1`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, ex string
		var active bool
		var ct time.Time
		rows.Scan(&id, &ex, &active, &ct)
		out = append(out, gin.H{"id": id, "exchange": ex, "is_active": active, "created_at": ct})
	}
	c.JSON(http.StatusOK, gin.H{"connections": out})
}

func (s *service) addCEX(c *gin.Context) {
	uid := c.GetString("user_id")
	var req struct {
		Exchange  string `json:"exchange"`
		APIKey    string `json:"api_key"`
		APISecret string `json:"api_secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Exchange == "" || req.APIKey == "" || req.APISecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exchange, api_key, api_secret required"})
		return
	}
	encKey := encryptSecret(req.APIKey)
	encSecret := encryptSecret(req.APISecret)
	var id string
	err := s.pg.QueryRow(c.Request.Context(),
		`INSERT INTO bot_cex_connections (user_id,exchange,api_key_encrypted,api_secret_encrypted) VALUES ($1,$2,$3,$4) RETURNING id`,
		uid, req.Exchange, encKey, encSecret).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "exchange": req.Exchange, "is_active": true})
}

func (s *service) removeCEX(c *gin.Context) {
	uid := c.GetString("user_id")
	tag, err := s.pg.Exec(c.Request.Context(), `DELETE FROM bot_cex_connections WHERE id=$1 AND user_id=$2`, c.Param("id"), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (s *service) listDEX(c *gin.Context) {
	uid := c.GetString("user_id")
	rows, err := s.pg.Query(c.Request.Context(), `SELECT id,dex,chain_id,rpc_url,is_active,created_at FROM bot_dex_connections WHERE user_id=$1`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, dex string
		var chainID int64
		var rpc sql.NullString
		var active bool
		var ct time.Time
		rows.Scan(&id, &dex, &chainID, &rpc, &active, &ct)
		out = append(out, gin.H{"id": id, "dex": dex, "chain_id": chainID, "rpc_url": rpc.String, "is_active": active, "created_at": ct})
	}
	c.JSON(http.StatusOK, gin.H{"connections": out})
}

func (s *service) addDEX(c *gin.Context) {
	uid := c.GetString("user_id")
	var req struct {
		DEX        string `json:"dex"`
		ChainID    int64  `json:"chain_id"`
		RPCURL     string `json:"rpc_url"`
		WalletSeed string `json:"wallet_seed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.DEX == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dex required"})
		return
	}
	if req.WalletSeed == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet_seed required (hex private key for DEX signing)"})
		return
	}
	encSeed := encryptSecret(req.WalletSeed)
	var id string
	err := s.pg.QueryRow(c.Request.Context(),
		`INSERT INTO bot_dex_connections (user_id,dex,chain_id,rpc_url,wallet_seed_encrypted) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		uid, req.DEX, req.ChainID, strOrNull(req.RPCURL), encSeed).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "dex": req.DEX, "chain_id": req.ChainID})
}

func (s *service) removeDEX(c *gin.Context) {
	uid := c.GetString("user_id")
	tag, err := s.pg.Exec(c.Request.Context(), `DELETE FROM bot_dex_connections WHERE id=$1 AND user_id=$2`, c.Param("id"), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// getMMConfigs fetches market-making configs from the ProjectParty backend.
// This is the Bots↔ProjectParty linkage: when a token is listed on ProjectParty,
// the project team can create a market-making config; the bots platform reads
// these configs to auto-create market-maker bots for listed tokens.
// Proxies GET <PP_URL>/api/v1/market-making/configs with the shared
// service-to-service token (X-Service-Token, PP_SERVICE_TOKEN env on both
// sides) — project-party fails closed when the token is unset/mismatched.
func (s *service) getMMConfigs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, projectPartyURL()+"/api/v1/market-making/configs", nil)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to build request to project-party"})
		return
	}
	if tok := os.Getenv("PP_SERVICE_TOKEN"); tok != "" {
		req.Header.Set("X-Service-Token", tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "project-party backend unreachable", "detail": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}

// ---- API key handlers ----

func (s *service) listAPIKeys(c *gin.Context) {
	uid := c.GetString("user_id")
	rows, err := s.pg.Query(c.Request.Context(), `SELECT id,label,permissions,is_active,created_at FROM bot_api_keys WHERE user_id=$1`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, label string
		var perms []byte
		var active bool
		var ct time.Time
		rows.Scan(&id, &label, &perms, &active, &ct)
		var p []string
		json.Unmarshal(perms, &p)
		out = append(out, gin.H{"id": id, "label": label, "permissions": p, "is_active": active, "created_at": ct})
	}
	c.JSON(http.StatusOK, gin.H{"keys": out})
}

func (s *service) createAPIKey(c *gin.Context) {
	uid := c.GetString("user_id")
	var req struct {
		Label       string   `json:"label"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Label == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "label required"})
		return
	}
	if len(req.Permissions) == 0 {
		req.Permissions = []string{"read"}
	}
	raw := generateAPIKey()
	h := hashToken(raw)
	perms, _ := json.Marshal(req.Permissions)
	_, err := s.pg.Exec(c.Request.Context(),
		`INSERT INTO bot_api_keys (user_id,key_hash,label,permissions) VALUES ($1,$2,$3,$4)`,
		uid, h, req.Label, perms)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"api_key": raw, "label": req.Label, "permissions": req.Permissions, "message": "store this key securely; it will not be shown again"})
}

func (s *service) deleteAPIKey(c *gin.Context) {
	uid := c.GetString("user_id")
	tag, err := s.pg.Exec(c.Request.Context(), `DELETE FROM bot_api_keys WHERE id=$1 AND user_id=$2`, c.Param("id"), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// ---- admin handlers ----

func (s *service) adminListUsers(c *gin.Context) {
	rows, err := s.pg.Query(c.Request.Context(), `SELECT id,username,email,role,is_active,created_at FROM bot_users ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, username, role string
		var email sql.NullString
		var active bool
		var ct time.Time
		rows.Scan(&id, &username, &email, &role, &active, &ct)
		out = append(out, gin.H{"id": id, "username": username, "email": email.String, "role": role, "is_active": active, "created_at": ct})
	}
	c.JSON(http.StatusOK, gin.H{"users": out, "count": len(out)})
}

func (s *service) adminUserStatus(c *gin.Context) {
	var req struct {
		IsActive bool   `json:"is_active"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Role != "" && !validRoles()[UserRole(req.Role)] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	if req.Role != "" {
		_, err := s.pg.Exec(c.Request.Context(), `UPDATE bot_users SET is_active=$1, role=$2 WHERE id=$3`, req.IsActive, req.Role, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		s.pg.Exec(c.Request.Context(), `UPDATE bot_users SET is_active=$1 WHERE id=$2`, req.IsActive, c.Param("id"))
	}
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id"), "is_active": req.IsActive, "role": req.Role})
}

func (s *service) adminStats(c *gin.Context) {
	var totalUsers, totalBots, activeBots, totalSubs int
	s.pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM bot_users`).Scan(&totalUsers)
	s.pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM bots`).Scan(&totalBots)
	s.pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM bots WHERE status='running'`).Scan(&activeBots)
	s.pg.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM bot_subscriptions WHERE status='active'`).Scan(&totalSubs)
	// bot type distribution
	rows, _ := s.pg.Query(c.Request.Context(), `SELECT bot_type, COUNT(*) FROM bots GROUP BY bot_type ORDER BY COUNT(*) DESC`)
	defer rows.Close()
	typeDist := []gin.H{}
	if rows != nil {
		for rows.Next() {
			var bt string
			var n int
			rows.Scan(&bt, &n)
			typeDist = append(typeDist, gin.H{"bot_type": bt, "count": n})
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"total_users": totalUsers, "total_bots": totalBots,
		"active_bots": activeBots, "active_subscriptions": totalSubs,
		"bot_type_distribution": typeDist,
	})
}

func (s *service) adminGetFeeAddresses(c *gin.Context) {
	rows, err := s.pg.Query(c.Request.Context(), `SELECT id,label,address,chain_id,is_active,created_at FROM admin_fee_addresses ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, label, addr string
		var chainID int64
		var active bool
		var ct time.Time
		rows.Scan(&id, &label, &addr, &chainID, &active, &ct)
		out = append(out, gin.H{"id": id, "label": label, "address": addr, "chain_id": chainID, "is_active": active, "created_at": ct})
	}
	c.JSON(http.StatusOK, gin.H{"addresses": out})
}

func (s *service) adminSetFeeAddress(c *gin.Context) {
	var req struct {
		Label   string `json:"label"`
		Address string `json:"address"`
		ChainID int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Label == "" || req.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "label and address required"})
		return
	}
	var id string
	err := s.pg.QueryRow(c.Request.Context(),
		`INSERT INTO admin_fee_addresses (label,address,chain_id) VALUES ($1,$2,$3) ON CONFLICT (address,chain_id) DO UPDATE SET label=excluded.label, is_active=TRUE RETURNING id`,
		req.Label, req.Address, req.ChainID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "label": req.Label, "address": req.Address, "chain_id": req.ChainID})
}

func (s *service) adminDeleteFeeAddress(c *gin.Context) {
	tag, err := s.pg.Exec(c.Request.Context(), `DELETE FROM admin_fee_addresses WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "address not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (s *service) adminBotStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status required"})
		return
	}
	// Admin route: role gate already applied; setBotStatusOwned passes through
	// for super_admin without an ownership clause.
	s.setBotStatusOwned(c, c.Param("id"), req.Status)
}

// ---- public ----

func (s *service) publicTiers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tiers": defaultTiers, "bot_types": botTypes})
}

// ---- helpers ----

func (s *service) fetchBot(c *gin.Context, id string) (gin.H, bool) {
	row := s.pg.QueryRow(c.Request.Context(), `
SELECT id,name,bot_type,config,status,exchange,pair,stats,created_at,updated_at,user_id FROM bots WHERE id=$1`, id)
	b, ok := scanBotRow(row)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return nil, false
	}
	return b, true
}

// scanBot works with pgx.Rows
func scanBot(rows pgx.Rows) (gin.H, string) {
	var id, name, bt, status string
	var config, stats []byte
	var exchange, pair sql.NullString
	var owner string
	var ct, ut time.Time
	rows.Scan(&id, &name, &bt, &config, &status, &exchange, &pair, &stats, &ct, &ut, &owner)
	return botJSON(id, name, bt, config, stats, status, exchange, pair, ct, ut, owner), owner
}

func scanBotRow(row pgx.Row) (gin.H, bool) {
	var id, name, bt, status string
	var config, stats []byte
	var exchange, pair sql.NullString
	var owner string
	var ct, ut time.Time
	if err := row.Scan(&id, &name, &bt, &config, &status, &exchange, &pair, &stats, &ct, &ut, &owner); err != nil {
		return nil, false
	}
	return botJSON(id, name, bt, config, stats, status, exchange, pair, ct, ut, owner), true
}

func botJSON(id, name, bt string, config, stats []byte, status string, exchange, pair sql.NullString, ct, ut time.Time, owner string) gin.H {
	var cfg, st interface{}
	if len(config) > 0 {
		json.Unmarshal(config, &cfg)
	}
	if len(stats) > 0 {
		json.Unmarshal(stats, &st)
	}
	return gin.H{
		"id": id, "name": name, "bot_type": bt, "config": cfg,
		"status": status, "exchange": exchange.String, "pair": pair.String,
		"stats": st, "created_at": ct, "updated_at": ut, "user_id": owner,
	}
}

func ownerClause(role string) string {
	if role == string(RoleSuperAdmin) {
		return ""
	}
	return " AND user_id=$2"
}

func strOrNull(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

func generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "tb_" + hex.EncodeToString(b)
}

// secretsEncryptionKey derives a 32-byte AES-256 key from the SECRETS_ENC_KEY
// env var (or JWT_SECRET fallback). The key is SHA-256 of the env value so any
// non-empty string produces a valid 256-bit key.
func secretsEncryptionKey() []byte {
	k := os.Getenv("SECRETS_ENC_KEY")
	if k == "" {
		k = os.Getenv("JWT_SECRET")
	}
	if k == "" {
		log.Fatalf("SECRETS_ENC_KEY or JWT_SECRET environment variable must be set")
	}
	h := sha256.Sum256([]byte(k))
	return h[:]
}

// encryptSecret encrypts a CEX API key/secret or DEX wallet seed at rest using
// AES-256-GCM. Output: "enc2:" || hex(nonce(12) || ciphertext+tag). The nonce is
// random per encryption. Real authenticated encryption — GCM tag detects
// tampering on decrypt (fail-closed).
func encryptSecret(s string) string {
	key := secretsEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("encryptSecret: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalf("encryptSecret: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		log.Fatalf("encryptSecret: %v", err)
	}
	ct := gcm.Seal(nonce, nonce, []byte(s), nil)
	return "enc2:" + hex.EncodeToString(ct)
}

// decryptSecret reverses encryptSecret. Returns plaintext or error on any
// tampering / wrong key (fail-closed). Legacy "enc1:" XOR blobs return an
// error (owner must re-enter to migrate).
func decryptSecret(enc string) (string, error) {
	if len(enc) < 5 || enc[:5] != "enc2:" {
		return "", errors.New("secret blob is not enc2 format (re-enter credential to migrate)")
	}
	raw, err := hex.DecodeString(enc[5:])
	if err != nil {
		return "", err
	}
	key := secretsEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// silence unused import for subtle (used for future constant-time compares)
var _ = subtle.ConstantTimeCompare
var _ = uuid.New
