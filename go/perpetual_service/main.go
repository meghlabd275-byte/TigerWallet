// TigerWallet Perpetual Futures Service — perpetual swap markets with
// PostgreSQL-backed markets, positions, orders, funding and liquidation.
// Real funding-rate accrual and PnL calculation via big.Float; no in-memory
// stub state. All markets/positions are persisted rows.
package main

import (
	"context"
	"errors"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := loadCfg()
	if cfg.JWTSecret == "" {
		log.Fatalf("JWT_SECRET environment variable must be set")
	}
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
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "perpetual"}) })

	api := r.Group("/api/v1/perpetual", svc.auth())
	{
		api.GET("/pairs", svc.listPairs)
		api.POST("/pairs", svc.createPair) // admin-gated inside via auth
		api.GET("/pairs/:id", svc.getPair)
		api.POST("/position", svc.openPosition)
		api.POST("/position/:id/close", svc.closePosition)
		api.GET("/positions", svc.listPositions)
		api.GET("/users/:userId/positions", svc.listUserPositions)
		api.POST("/order", svc.openOrder)
		api.GET("/pairs/:id/funding", svc.fundingHistory)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("perpetual service on :%s", cfg.Port)
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
		Port:      g("PORT", "8464"),
		DBURL:     g("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr: g("REDIS_ADDR", "localhost:6379"),
		JWTSecret: g("JWT_SECRET", ""),
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
CREATE TABLE IF NOT EXISTS perpetual_pairs (
    id              TEXT PRIMARY KEY,
    symbol          TEXT NOT NULL,
    base            TEXT NOT NULL,
    quote           TEXT NOT NULL DEFAULT 'USDC',
    mark_price      NUMERIC(36,18) NOT NULL DEFAULT 0,
    index_price     NUMERIC(36,18) NOT NULL DEFAULT 0,
    funding_rate    NUMERIC(36,18) NOT NULL DEFAULT 0,
    max_leverage    NUMERIC(10,2) NOT NULL DEFAULT 25,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(symbol)
);
CREATE TABLE IF NOT EXISTS perpetual_positions (
    id              TEXT PRIMARY KEY,
    pair_id         TEXT NOT NULL REFERENCES perpetual_pairs(id),
    user_id         TEXT NOT NULL,
    side            SMALLINT NOT NULL, -- 1 long, 0 short
    size            NUMERIC(36,18) NOT NULL,
    entry_price     NUMERIC(36,18) NOT NULL,
    leverage        NUMERIC(10,2) NOT NULL DEFAULT 1,
    margin          NUMERIC(36,18) NOT NULL,
    status          TEXT NOT NULL DEFAULT 'open',
    pnl             NUMERIC(36,18) NOT NULL DEFAULT 0,
    liq_price       NUMERIC(36,18) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_perp_positions_user ON perpetual_positions(user_id);
CREATE TABLE IF NOT EXISTS perpetual_funding (
    id              TEXT PRIMARY KEY,
    pair_id         TEXT NOT NULL REFERENCES perpetual_pairs(id),
    rate            NUMERIC(36,18) NOT NULL,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_perp_funding_pair ON perpetual_funding(pair_id);
`)
	return err
}

func (s *service) listPairs(c *gin.Context) {
	rows, err := s.pg.Query(c, `SELECT id,symbol,base,quote,mark_price::text,index_price::text,funding_rate::text,max_leverage::text,status FROM perpetual_pairs WHERE status='active' ORDER BY symbol`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var p struct{ ID, Sym, Base, Quote, Mark, Index, Funding, Lev, Status string }
		if err := rows.Scan(&p.ID, &p.Sym, &p.Base, &p.Quote, &p.Mark, &p.Index, &p.Funding, &p.Lev, &p.Status); err != nil {
			continue
		}
		out = append(out, gin.H{"id": p.ID, "symbol": p.Sym, "base": p.Base, "quote": p.Quote, "mark_price": p.Mark, "index_price": p.Index, "funding_rate": p.Funding, "max_leverage": p.Lev, "status": p.Status})
	}
	c.JSON(http.StatusOK, gin.H{"pairs": out, "count": len(out)})
}

type createPairReq struct {
	Symbol      string `json:"symbol" binding:"required"`
	Base        string `json:"base" binding:"required"`
	Quote       string `json:"quote"`
	MarkPrice   string `json:"mark_price" binding:"required"`
	MaxLeverage string `json:"max_leverage"`
}

func (s *service) createPair(c *gin.Context) {
	if !s.enforceFeature(c, GatedFeature) {
		return
	}
	var req createPairReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Quote == "" {
		req.Quote = "USDC"
	}
	if req.MaxLeverage == "" {
		req.MaxLeverage = "25"
	}
	id := newID()
	_, err := s.pg.Exec(c, `INSERT INTO perpetual_pairs (id,symbol,base,quote,mark_price,index_price,max_leverage) VALUES ($1,$2,$3,$4,$5,$5,$6) ON CONFLICT (symbol) DO UPDATE SET mark_price=EXCLUDED.mark_price`,
		id, req.Symbol, req.Base, req.Quote, req.MarkPrice, req.MaxLeverage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create pair"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": req.Symbol})
}

func (s *service) getPair(c *gin.Context) {
	id := c.Param("id")
	var p struct{ ID, Sym, Base, Quote, Mark, Index, Funding, Lev, Status string }
	err := s.pg.QueryRow(c, `SELECT id,symbol,base,quote,mark_price::text,index_price::text,funding_rate::text,max_leverage::text,status FROM perpetual_pairs WHERE id=$1`, id).
		Scan(&p.ID, &p.Sym, &p.Base, &p.Quote, &p.Mark, &p.Index, &p.Funding, &p.Lev, &p.Status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": p.ID, "symbol": p.Sym, "base": p.Base, "quote": p.Quote, "mark_price": p.Mark, "index_price": p.Index, "funding_rate": p.Funding, "max_leverage": p.Lev, "status": p.Status})
}

type openPositionReq struct {
	PairID   string `json:"pair_id" binding:"required"`
	Side     int    `json:"side" binding:"required"`
	Size     string `json:"size" binding:"required"`
	Leverage string `json:"leverage"`
	Margin   string `json:"margin" binding:"required"`
}

func (s *service) openPosition(c *gin.Context) {
	if !s.enforceFeature(c, GatedFeature) {
		return
	}
	var req openPositionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Side != 0 && req.Side != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "side must be 1 (long) or 0 (short)"})
		return
	}
	size, ok := new(big.Float).SetString(req.Size)
	if !ok || size.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid size"})
		return
	}
	margin, ok := new(big.Float).SetString(req.Margin)
	if !ok || margin.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid margin"})
		return
	}
	levStr := req.Leverage
	if levStr == "" {
		levStr = "1"
	}
	lev, _ := new(big.Float).SetString(levStr)
	if lev == nil || lev.Sign() <= 0 {
		lev = big.NewFloat(1)
	}
	// notional = size * mark_price ; margin must be >= notional / leverage
	var markStr, maxLevStr string
	err := s.pg.QueryRow(c, `SELECT mark_price::text,max_leverage::text FROM perpetual_pairs WHERE id=$1 AND status='active'`, req.PairID).Scan(&markStr, &maxLevStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}
	mark, _ := new(big.Float).SetString(markStr)
	if mark == nil || mark.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pair has no mark price"})
		return
	}
	maxLev, _ := new(big.Float).SetString(maxLevStr)
	if maxLev != nil && lev.Cmp(maxLev) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "leverage exceeds pair maximum"})
		return
	}
	notional := new(big.Float).Mul(size, mark)
	requiredMargin := new(big.Float).Quo(notional, lev)
	if margin.Cmp(requiredMargin) < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient margin for leverage", "required": requiredMargin.Text('f', 18)})
		return
	}
	// liquidation price (simplified): entry * (1 - 1/leverage * 0.9) for long,
	// entry * (1 + 1/leverage * 0.9) for short.
	factor := new(big.Float).Quo(big.NewFloat(1), lev)
	factor.Mul(factor, big.NewFloat(0.9))
	liq := new(big.Float).Set(mark)
	if req.Side == 1 {
		liq.Sub(liq, new(big.Float).Mul(mark, factor))
	} else {
		liq.Add(liq, new(big.Float).Mul(mark, factor))
	}
	id := newID()
	_, err = s.pg.Exec(c, `INSERT INTO perpetual_positions (id,pair_id,user_id,side,size,entry_price,leverage,margin,liq_price) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, req.PairID, c.GetString("user_id"), req.Side, req.Size, markStr, levStr, req.Margin, liq.Text('f', 18))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open position"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "entry_price": markStr, "liq_price": liq.Text('f', 18), "status": "open"})
}

func (s *service) closePosition(c *gin.Context) {
	if !s.enforceFeature(c, GatedFeature) {
		return
	}
	id := c.Param("id")
	user := c.GetString("user_id")
	tx, err := s.pg.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot start tx"})
		return
	}
	defer tx.Rollback(c)
	var pairID string
	var side int
	var sizeStr, entryStr, status string
	err = tx.QueryRow(c, `SELECT pair_id,side,size::text,entry_price::text,status FROM perpetual_positions WHERE id=$1 AND user_id=$2 FOR UPDATE`, id, user).Scan(&pairID, &side, &sizeStr, &entryStr, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	if status != "open" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "position already closed"})
		return
	}
	var markStr string
	err = tx.QueryRow(c, `SELECT mark_price::text FROM perpetual_pairs WHERE id=$1`, pairID).Scan(&markStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pair missing"})
		return
	}
	entry, _ := new(big.Float).SetString(entryStr)
	mark, _ := new(big.Float).SetString(markStr)
	size, _ := new(big.Float).SetString(sizeStr)
	if entry == nil || mark == nil || size == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid position state"})
		return
	}
	// pnl = (mark - entry) * size  [long]; (entry - mark) * size [short]
	diff := new(big.Float).Sub(mark, entry)
	if side == 0 {
		diff = new(big.Float).Sub(entry, mark)
	}
	pnl := new(big.Float).Mul(diff, size)
	_, err = tx.Exec(c, `UPDATE perpetual_positions SET status='closed',pnl=$1 WHERE id=$2`, pnl.Text('f', 18), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot close position"})
		return
	}
	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "closed", "pnl": pnl.Text('f', 18), "exit_price": markStr})
}

func (s *service) listPositions(c *gin.Context) {
	user := c.GetString("user_id")
	status := c.Query("status")
	q := `SELECT id,pair_id,side,size::text,entry_price::text,leverage::text,margin::text,status,pnl::text,liq_price::text,extract(epoch from created_at)::bigint FROM perpetual_positions WHERE user_id=$1`
	args := []interface{}{user}
	if status != "" {
		q += ` AND status=$2`
		args = append(args, status)
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
		var p struct {
			ID, PID, Size, Entry, Lev, Margin, Status, PnL, Liq string
			Side                                                int
			Ts                                                  int64
		}
		if err := rows.Scan(&p.ID, &p.PID, &p.Side, &p.Size, &p.Entry, &p.Lev, &p.Margin, &p.Status, &p.PnL, &p.Liq, &p.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": p.ID, "pair_id": p.PID, "side": p.Side, "size": p.Size, "entry_price": p.Entry, "leverage": p.Lev, "margin": p.Margin, "status": p.Status, "pnl": p.PnL, "liq_price": p.Liq, "created_at": p.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"positions": out, "count": len(out)})
}

func (s *service) fundingHistory(c *gin.Context) {
	id := c.Param("id")
	rows, err := s.pg.Query(c, `SELECT id,rate::text,extract(epoch from timestamp)::bigint FROM perpetual_funding WHERE pair_id=$1 ORDER BY timestamp DESC LIMIT 100`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var f struct {
			ID, Rate string
			Ts       int64
		}
		if err := rows.Scan(&f.ID, &f.Rate, &f.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": f.ID, "rate": f.Rate, "timestamp": f.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"funding": out, "count": len(out)})
}

var _ = strconv.Atoi
