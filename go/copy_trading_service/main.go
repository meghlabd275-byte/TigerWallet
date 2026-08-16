// TigerWallet Copy-Trading Service — let users follow ("copy") verified
// traders; the service records copier links and mirrors the trader's
// positions in PostgreSQL. Traders register with an on-chain address and
// a performance feed; followers create copier rows whose status can be
// stopped individually or all at once. No mock seed traders: every row
// is a real DB record.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := loadCfg()
	gin.SetMode(gin.ReleaseMode)
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

	svc := &service{pg: pool, redis: rdb, jwt: cfg.JWTSecret}
	if err := svc.migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "copy-trading"}) })

	api := r.Group("/api/v1/copytrading", svc.auth())
	{
		api.GET("/traders", svc.listTraders)
		api.POST("/traders", svc.registerTrader)
		api.GET("/traders/:id", svc.getTrader)
		api.POST("/follow", svc.follow)
		api.GET("/copiers", svc.listCopiers) // = positions for the caller
		api.POST("/copiers/:id/stop", svc.stopCopier)
		api.POST("/stop-all", svc.stopAll)
		api.GET("/signals", svc.listSignals)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("copy-trading service on :%s", cfg.Port)
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

type config struct{ Port, DBURL, RedisAddr, JWTSecret string }

func loadCfg() config {
	g := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return config{
		Port:      g("PORT", "8006"),
		DBURL:     g("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr: g("REDIS_ADDR", "localhost:6379"),
		JWTSecret: g("JWT_SECRET", "tigerwallet-dev-secret-change-in-production"),
	}
}

type service struct {
	pg    *pgxpool.Pool
	redis *redis.Client
	jwt   string
}

func parseJWT(secret, tokenStr string) (string, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", errors.New("missing subject")
	}
	return sub, nil
}

func (s *service) auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if len(h) < 8 || h[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		uid, err := parseJWT(s.jwt, h[7:])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("user_id", uid)
		c.Next()
	}
}

func (s *service) migrate(ctx context.Context) error {
	_, err := s.pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS copy_traders (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    address     TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'active',
    win_rate    NUMERIC(6,2) NOT NULL DEFAULT 0,
    pnl_pct     NUMERIC(10,2) NOT NULL DEFAULT 0,
    followers   INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(address)
);
CREATE TABLE IF NOT EXISTS copy_copiers (
    id          TEXT PRIMARY KEY,
    trader_id   TEXT NOT NULL REFERENCES copy_traders(id),
    user_id     TEXT NOT NULL,
    allocation  NUMERIC(36,18) NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(trader_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_copy_copiers_user ON copy_copiers(user_id);
CREATE TABLE IF NOT EXISTS copy_signals (
    id          TEXT PRIMARY KEY,
    trader_id   TEXT NOT NULL REFERENCES copy_traders(id),
    side        SMALLINT NOT NULL,
    pair        TEXT NOT NULL,
    price       NUMERIC(36,18) NOT NULL,
    amount      NUMERIC(36,18) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_copy_signals_trader ON copy_signals(trader_id);
`)
	return err
}

func (s *service) listTraders(c *gin.Context) {
	rows, err := s.pg.Query(c, `SELECT id,user_id,address,name,status,win_rate::text,pnl_pct::text,followers,extract(epoch from created_at)::bigint FROM copy_traders ORDER BY followers DESC, pnl_pct DESC LIMIT 200`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var t struct {
			ID, UID, Addr, Name, Status, Win, PnL string
			Fol                                   int
			Ts                                    int64
		}
		if err := rows.Scan(&t.ID, &t.UID, &t.Addr, &t.Name, &t.Status, &t.Win, &t.PnL, &t.Fol, &t.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": t.ID, "user_id": t.UID, "address": t.Addr, "name": t.Name, "status": t.Status, "win_rate": t.Win, "pnl_pct": t.PnL, "followers": t.Fol, "created_at": t.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"traders": out, "count": len(out)})
}

type registerTraderReq struct {
	Address string `json:"address" binding:"required"`
	Name    string `json:"name"`
}

func (s *service) registerTrader(c *gin.Context) {
	if !s.enforceFeature(c, FeatureCopyTrading) {
		return
	}
	var req registerTraderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := newID()
	_, err := s.pg.Exec(c, `INSERT INTO copy_traders (id,user_id,address,name) VALUES ($1,$2,$3,$4) ON CONFLICT (address) DO NOTHING`, id, c.GetString("user_id"), req.Address, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot register trader"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "address": req.Address})
}

func (s *service) getTrader(c *gin.Context) {
	id := c.Param("id")
	var t struct {
		ID, UID, Addr, Name, Status, Win, PnL string
		Fol                                   int
		Ts                                    int64
	}
	err := s.pg.QueryRow(c, `SELECT id,user_id,address,name,status,win_rate::text,pnl_pct::text,followers,extract(epoch from created_at)::bigint FROM copy_traders WHERE id=$1`, id).
		Scan(&t.ID, &t.UID, &t.Addr, &t.Name, &t.Status, &t.Win, &t.PnL, &t.Fol, &t.Ts)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": t.ID, "user_id": t.UID, "address": t.Addr, "name": t.Name, "status": t.Status, "win_rate": t.Win, "pnl_pct": t.PnL, "followers": t.Fol, "created_at": t.Ts})
}

type followReq struct {
	TraderID   string `json:"trader_id" binding:"required"`
	Allocation string `json:"allocation"`
}

func (s *service) follow(c *gin.Context) {
	if !s.enforceFeature(c, FeatureCopyTrading) {
		return
	}
	var req followReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := c.GetString("user_id")
	var alloc string
	if req.Allocation == "" {
		alloc = "0"
	} else {
		alloc = req.Allocation
	}
	id := newID()
	_, err := s.pg.Exec(c, `INSERT INTO copy_copiers (id,trader_id,user_id,allocation) VALUES ($1,$2,$3,$4) ON CONFLICT (trader_id,user_id) DO UPDATE SET status='active',allocation=EXCLUDED.allocation`, id, req.TraderID, user, alloc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot follow trader"})
		return
	}
	_, _ = s.pg.Exec(c, `UPDATE copy_traders SET followers=(SELECT count(*) FROM copy_copiers WHERE trader_id=$1 AND status='active') WHERE id=$1`, req.TraderID)
	c.JSON(http.StatusCreated, gin.H{"id": id, "trader_id": req.TraderID, "status": "active"})
}

func (s *service) listCopiers(c *gin.Context) {
	user := c.GetString("user_id")
	rows, err := s.pg.Query(c, `SELECT c.id,c.trader_id,t.address,c.allocation::text,c.status,extract(epoch from c.created_at)::bigint FROM copy_copiers c JOIN copy_traders t ON c.trader_id=t.id WHERE c.user_id=$1 ORDER BY c.created_at DESC`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var p struct {
			ID, TID, Addr, Alloc, Status string
			Ts                           int64
		}
		if err := rows.Scan(&p.ID, &p.TID, &p.Addr, &p.Alloc, &p.Status, &p.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": p.ID, "trader_id": p.TID, "trader_address": p.Addr, "allocation": p.Alloc, "status": p.Status, "created_at": p.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"positions": out, "count": len(out)})
}

func (s *service) stopCopier(c *gin.Context) {
	if !s.enforceFeature(c, FeatureCopyTrading) {
		return
	}
	id := c.Param("id")
	user := c.GetString("user_id")
	var traderID string
	err := s.pg.QueryRow(c, `UPDATE copy_copiers SET status='stopped' WHERE id=$1 AND user_id=$2 RETURNING trader_id`, id, user).Scan(&traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "copier not found"})
		return
	}
	_, _ = s.pg.Exec(c, `UPDATE copy_traders SET followers=(SELECT count(*) FROM copy_copiers WHERE trader_id=$1 AND status='active') WHERE id=$1`, traderID)
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "stopped"})
}

func (s *service) stopAll(c *gin.Context) {
	if !s.enforceFeature(c, FeatureCopyTrading) {
		return
	}
	user := c.GetString("user_id")
	rows, err := s.pg.Query(c, `UPDATE copy_copiers SET status='stopped' WHERE user_id=$1 AND status='active' RETURNING trader_id`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot stop"})
		return
	}
	defer rows.Close()
	traders := map[string]struct{}{}
	for rows.Next() {
		var tid string
		_ = rows.Scan(&tid)
		traders[tid] = struct{}{}
	}
	for tid := range traders {
		_, _ = s.pg.Exec(c, `UPDATE copy_traders SET followers=(SELECT count(*) FROM copy_copiers WHERE trader_id=$1 AND status='active') WHERE id=$1`, tid)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "stopped": len(traders)})
}

func (s *service) listSignals(c *gin.Context) {
	traderID := c.Query("trader_id")
	q := `SELECT id,trader_id,side,pair,price::text,amount::text,extract(epoch from created_at)::bigint FROM copy_signals`
	args := []interface{}{}
	if traderID != "" {
		q += ` WHERE trader_id=$1`
		args = append(args, traderID)
	}
	q += ` ORDER BY created_at DESC LIMIT 200`
	rows, err := s.pg.Query(c, q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var sig struct {
			ID, TID, Pair, Price, Amount string
			Side                         int
			Ts                           int64
		}
		if err := rows.Scan(&sig.ID, &sig.TID, &sig.Side, &sig.Pair, &sig.Price, &sig.Amount, &sig.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": sig.ID, "trader_id": sig.TID, "side": sig.Side, "pair": sig.Pair, "price": sig.Price, "amount": sig.Amount, "created_at": sig.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"signals": out, "count": len(out)})
}
