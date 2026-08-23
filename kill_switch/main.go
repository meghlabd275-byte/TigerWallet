// kill_switch — the TigerWallet emergency kill-switch control plane.
//
// Language placement: Go (high-load, world-wide distributed control plane).
// Persistence: PostgreSQL (durable audit + state) + Redis (sub-second halt
// propagation to every white-label product heartbeat).
//
// What it does:
//   - SuperAdmin can HALT (or RESUME) the entire platform, one WL client,
//     one product of one client, or a single fetcher of one product.
//   - A halt is written to PostgreSQL (durable, audited), pushed into Redis
//     keys, and broadcast on the Redis pub/sub channel "kill:events" so that
//     every listener learns about it in well under a second.
//   - license_service consults the same Redis keys on every WL heartbeat, so
//     a halted product fails closed ("alive": false, command "halt") on its
//     next heartbeat — the same path the Governance UI documents.
//   - A self-healing loop republishes active halts from PostgreSQL into Redis
//     every few seconds, so a Redis restart/flush cannot silently clear a
//     halt. Halts are a positive signal; they are never inferred from the
//     absence of data.
//
// Auth: the same SuperAdmin HS256 JWT minted by license_service /auth/login
// (shared JWT_SECRET). Only role "superadmin" may halt/resume. There is no
// other credential and no anonymous path: fail-closed 401/403 everywhere.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const eventsChannel = "kill:events"

var validScopes = map[string]bool{"global": true, "client": true, "product": true, "fetcher": true}

type config struct {
	Port          string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	JWTSecret     string
}

func loadConfig() config {
	cfg := config{
		Port:          getenv("PORT", "8469"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisAddr:     getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required (PostgreSQL is the durable state store)")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required (shared with license_service for SuperAdmin auth)")
	}
	return cfg
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// Claims mirrors license_service's control-plane JWT (same secret, same
// claims) so a SuperAdmin token works against both services.
type Claims struct {
	AdminID    uuid.UUID  `json:"admin_id"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	WLClientID *uuid.UUID `json:"wl_client_id,omitempty"`
	jwt.RegisteredClaims
}

func superAdminAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		parts := strings.Split(h, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bearer token required"})
			return
		}
		claims := &Claims{}
		tok, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !tok.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		if claims.Role != "superadmin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "superadmin role required"})
			return
		}
		c.Set("admin_id", claims.AdminID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// redisKeysForScope returns every Redis key that represents a scope. A scope
// is halted when its key exists (value is the reason). All keys are exact —
// no pattern scans on the hot path.
func redisKeysForScope(scopeType string, wlClientID *uuid.UUID, product, fetcher string) ([]string, error) {
	switch scopeType {
	case "global":
		return []string{"kill:global"}, nil
	case "client":
		if wlClientID == nil {
			return nil, errors.New("wl_client_id required for client scope")
		}
		return []string{"kill:client:" + wlClientID.String()}, nil
	case "product":
		if wlClientID == nil || product == "" {
			return nil, errors.New("wl_client_id and product required for product scope")
		}
		return []string{"kill:product:" + wlClientID.String() + ":" + product}, nil
	case "fetcher":
		if wlClientID == nil || product == "" || fetcher == "" {
			return nil, errors.New("wl_client_id, product and fetcher required for fetcher scope")
		}
		return []string{"kill:fetcher:" + wlClientID.String() + ":" + product + ":" + fetcher}, nil
	}
	return nil, fmt.Errorf("unknown scope_type %q", scopeType)
}

type server struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

type scopeRequest struct {
	ScopeType  string     `json:"scope_type" binding:"required"` // global|client|product|fetcher
	WLClientID *uuid.UUID `json:"wl_client_id,omitempty"`
	Product    string     `json:"product,omitempty"`
	Fetcher    string     `json:"fetcher,omitempty"`
	Reason     string     `json:"reason,omitempty"`
}

func (r scopeRequest) validate() error {
	if !validScopes[r.ScopeType] {
		return fmt.Errorf("scope_type must be one of global|client|product|fetcher")
	}
	_, err := redisKeysForScope(r.ScopeType, r.WLClientID, r.Product, r.Fetcher)
	return err
}

func (s *server) migrate(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
CREATE TABLE IF NOT EXISTS kill_state (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  scope_type    text NOT NULL CHECK (scope_type IN ('global','client','product','fetcher')),
  wl_client_id  uuid,
  product       text NOT NULL DEFAULT '',
  fetcher       text NOT NULL DEFAULT '',
  reason        text NOT NULL DEFAULT '',
  active        boolean NOT NULL DEFAULT true,
  updated_by    uuid,
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS kill_state_scope_key
  ON kill_state (scope_type, COALESCE(wl_client_id, '00000000-0000-0000-0000-000000000000'::uuid), product, fetcher);
CREATE TABLE IF NOT EXISTS kill_events (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  action        text NOT NULL CHECK (action IN ('halt','resume')),
  scope_type    text NOT NULL,
  wl_client_id  uuid,
  product       text NOT NULL DEFAULT '',
  fetcher       text NOT NULL DEFAULT '',
  reason        text NOT NULL DEFAULT '',
  issued_by     uuid,
  created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS kill_events_created_at ON kill_events (created_at DESC);
`)
	return err
}

func (s *server) setHalt(ctx context.Context, req scopeRequest, active bool, adminID uuid.UUID) error {
	keys, err := redisKeysForScope(req.ScopeType, req.WLClientID, req.Product, req.Fetcher)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var clientID any
	if req.WLClientID != nil {
		clientID = *req.WLClientID
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO kill_state (scope_type, wl_client_id, product, fetcher, reason, active, updated_by, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7, now())
ON CONFLICT (scope_type, COALESCE(wl_client_id, '00000000-0000-0000-0000-000000000000'::uuid), product, fetcher)
DO UPDATE SET active=EXCLUDED.active, reason=EXCLUDED.reason, updated_by=EXCLUDED.updated_by, updated_at=now()`,
		req.ScopeType, clientID, req.Product, req.Fetcher, req.Reason, active, adminID); err != nil {
		return err
	}
	action := "halt"
	if !active {
		action = "resume"
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO kill_events (action, scope_type, wl_client_id, product, fetcher, reason, issued_by)
VALUES ($1,$2,$3,$4,$5,$6,$7)`, action, req.ScopeType, clientID, req.Product, req.Fetcher, req.Reason, adminID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	// Durable state committed — now propagate to Redis (state keys + pub/sub).
	for _, k := range keys {
		if active {
			if err := s.rdb.Set(ctx, k, req.Reason, 0).Err(); err != nil {
				return fmt.Errorf("state committed but redis publish failed (self-heal will retry): %w", err)
			}
		} else {
			if err := s.rdb.Del(ctx, k).Err(); err != nil {
				return fmt.Errorf("state committed but redis clear failed (self-heal will retry): %w", err)
			}
		}
	}
	evt, _ := json.Marshal(gin.H{
		"action": action, "scope_type": req.ScopeType, "wl_client_id": req.WLClientID,
		"product": req.Product, "fetcher": req.Fetcher, "reason": req.Reason,
		"at": time.Now().UTC().Format(time.RFC3339Nano),
	})
	s.rdb.Publish(ctx, eventsChannel, evt)
	return nil
}

func (s *server) handleHalt(c *gin.Context) {
	var req scopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := req.validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminID, _ := c.Get("admin_id")
	if err := s.setHalt(c.Request.Context(), req, true, adminID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"halted": true, "scope_type": req.ScopeType, "product": req.Product, "fetcher": req.Fetcher})
}

func (s *server) handleResume(c *gin.Context) {
	var req scopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := req.validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminID, _ := c.Get("admin_id")
	if err := s.setHalt(c.Request.Context(), req, false, adminID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"halted": false, "scope_type": req.ScopeType, "product": req.Product, "fetcher": req.Fetcher})
}

func (s *server) handleState(c *gin.Context) {
	rows, err := s.db.Query(c.Request.Context(), `
SELECT scope_type, wl_client_id, product, fetcher, reason, updated_at
FROM kill_state WHERE active ORDER BY updated_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	halts := []gin.H{}
	for rows.Next() {
		var st string
		var cid *uuid.UUID
		var product, fetcher, reason string
		var at time.Time
		if err := rows.Scan(&st, &cid, &product, &fetcher, &reason, &at); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		halts = append(halts, gin.H{"scope_type": st, "wl_client_id": cid, "product": product, "fetcher": fetcher, "reason": reason, "since": at})
	}
	c.JSON(http.StatusOK, gin.H{"halts": halts})
}

func (s *server) handleAudit(c *gin.Context) {
	rows, err := s.db.Query(c.Request.Context(), `
SELECT action, scope_type, wl_client_id, product, fetcher, reason, issued_by, created_at
FROM kill_events ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	events := []gin.H{}
	for rows.Next() {
		var action, st, product, fetcher, reason string
		var cid, by *uuid.UUID
		var at time.Time
		if err := rows.Scan(&action, &st, &cid, &product, &fetcher, &reason, &by, &at); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		events = append(events, gin.H{"action": action, "scope_type": st, "wl_client_id": cid, "product": product, "fetcher": fetcher, "reason": reason, "issued_by": by, "at": at})
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

// selfHeal republishes every active halt from PostgreSQL into Redis and
// clears Redis keys whose PG row is no longer active. This makes the halt
// state survive Redis restarts/flushes without any operator action.
func (s *server) selfHeal(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.reconcileRedis(ctx)
		}
	}
}

func (s *server) reconcileRedis(ctx context.Context) {
	rows, err := s.db.Query(ctx, `SELECT scope_type, wl_client_id, product, fetcher, reason, active FROM kill_state`)
	if err != nil {
		log.Printf("self-heal: state query failed: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var st, product, fetcher, reason string
		var cid *uuid.UUID
		var active bool
		if err := rows.Scan(&st, &cid, &product, &fetcher, &reason, &active); err != nil {
			continue
		}
		keys, err := redisKeysForScope(st, cid, product, fetcher)
		if err != nil {
			continue
		}
		for _, k := range keys {
			if active {
				if err := s.rdb.Set(ctx, k, reason, 0).Err(); err != nil {
					log.Printf("self-heal: republish %s failed: %v", k, err)
				}
			} else {
				s.rdb.Del(ctx, k)
			}
		}
	}
}

func main() {
	cfg := loadConfig()
	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres connect: %v", err)
	}
	defer db.Close()
	bootCtx, bootCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := db.Ping(bootCtx); err != nil {
		bootCancel()
		log.Fatalf("postgres ping: %v", err)
	}
	s := &server{db: db, rdb: redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})}
	if err := s.migrate(bootCtx); err != nil {
		bootCancel()
		log.Fatalf("migrate: %v", err)
	}
	if err := s.rdb.Ping(bootCtx).Err(); err != nil {
		bootCancel()
		log.Fatalf("redis ping: %v", err)
	}
	bootCancel()

	// Rehydrate Redis from durable state immediately at boot.
	s.reconcileRedis(ctx)
	healCtx, healCancel := context.WithCancel(ctx)
	defer healCancel()
	go s.selfHeal(healCtx)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", func(c *gin.Context) {
		hctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		pgErr := s.db.Ping(hctx)
		rErr := s.rdb.Ping(hctx).Err()
		if pgErr != nil || rErr != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "postgres": errStr(pgErr), "redis": errStr(rErr)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "kill-switch"})
	})

	api := r.Group("/api/v1/kill", superAdminAuth(cfg.JWTSecret))
	{
		api.POST("/halt", s.handleHalt)
		api.POST("/resume", s.handleResume)
		api.GET("/state", s.handleState)
		api.GET("/audit", s.handleAudit)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		log.Printf("kill-switch listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutCtx, shutCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
	log.Println("kill-switch stopped")
}

func errStr(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}
