// TigerWallet Liquidity Pool Service — AMM-style liquidity pools with
// add/remove liquidity and LP positions tracked in PostgreSQL. The math is
// the standard constant-product (x*y=k) used by Uniswap V2; reserves are
// persisted as big integers and LP tokens are minted proportionally.
package main

import (
	"context"
	"errors"
	"log"
	"math/big"
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
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "pool"}) })

	api := r.Group("/api/v1/pool", svc.auth())
	{
		api.GET("/list", svc.listPools)
		api.POST("/create", svc.createPool)
		api.POST("/add-liquidity", svc.addLiquidity)
		api.POST("/remove-liquidity", svc.removeLiquidity)
		api.GET("/positions", svc.listPositions)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("pool service on :%s", cfg.Port)
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
		Port:      g("PORT", "8463"),
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
CREATE TABLE IF NOT EXISTS liquidity_pools (
    id          TEXT PRIMARY KEY,
    pair        TEXT NOT NULL,
    token0      TEXT NOT NULL,
    token1      TEXT NOT NULL,
    reserve0    NUMERIC(78,0) NOT NULL DEFAULT 0,
    reserve1    NUMERIC(78,0) NOT NULL DEFAULT 0,
    total_supply NUMERIC(78,0) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(pair)
);
CREATE TABLE IF NOT EXISTS liquidity_positions (
    id          TEXT PRIMARY KEY,
    pool_id      TEXT NOT NULL REFERENCES liquidity_pools(id),
    user_id     TEXT NOT NULL,
    lp_tokens   NUMERIC(78,0) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(pool_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_lp_positions_user ON liquidity_positions(user_id);
`)
	return err
}

func (s *service) listPools(c *gin.Context) {
	rows, err := s.pg.Query(c, `SELECT id,pair,token0,token1,reserve0::text,reserve1::text,total_supply::text FROM liquidity_pools ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var p struct{ ID, Pair, T0, T1, R0, R1, TS string }
		if err := rows.Scan(&p.ID, &p.Pair, &p.T0, &p.T1, &p.R0, &p.R1, &p.TS); err != nil {
			continue
		}
		out = append(out, gin.H{"id": p.ID, "pair": p.Pair, "token0": p.T0, "token1": p.T1, "reserve0": p.R0, "reserve1": p.R1, "total_supply": p.TS})
	}
	c.JSON(http.StatusOK, gin.H{"pools": out, "count": len(out)})
}

type createPoolReq struct {
	Pair   string `json:"pair" binding:"required"`
	Token0 string `json:"token0" binding:"required"`
	Token1 string `json:"token1" binding:"required"`
}

func (s *service) createPool(c *gin.Context) {
	var req createPoolReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := newID()
	_, err := s.pg.Exec(c, `INSERT INTO liquidity_pools (id,pair,token0,token1) VALUES ($1,$2,$3,$4) ON CONFLICT (pair) DO NOTHING`, id, req.Pair, req.Token0, req.Token1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create pool"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "pair": req.Pair})
}

type addLiqReq struct {
	Pair    string `json:"pair" binding:"required"`
	Amount0 string `json:"amount0" binding:"required"`
	Amount1 string `json:"amount1" binding:"required"`
}

func (s *service) addLiquidity(c *gin.Context) {
	var req addLiqReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a0, ok := new(big.Int).SetString(req.Amount0, 10)
	if !ok || a0.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount0"})
		return
	}
	a1, ok := new(big.Int).SetString(req.Amount1, 10)
	if !ok || a1.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount1"})
		return
	}
	user := c.GetString("user_id")
	tx, err := s.pg.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot start tx"})
		return
	}
	defer tx.Rollback(c)
	var pid, r0Str, r1Str, tsStr string
	err = tx.QueryRow(c, `SELECT id,reserve0::text,reserve1::text,total_supply::text FROM liquidity_pools WHERE pair=$1 FOR UPDATE`, req.Pair).Scan(&pid, &r0Str, &r1Str, &tsStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pool not found"})
		return
	}
	r0, _ := new(big.Int).SetString(r0Str, 10)
	r1, _ := new(big.Int).SetString(r1Str, 10)
	ts, _ := new(big.Int).SetString(tsStr, 10)
	if r0 == nil {
		r0 = big.NewInt(0)
	}
	if r1 == nil {
		r1 = big.NewInt(0)
	}
	if ts == nil {
		ts = big.NewInt(0)
	}
	var minted *big.Int
	if ts.Sign() == 0 {
		// first liquidity: mint sqrt(a0*a1)
		minted = sqrtBig(new(big.Int).Mul(a0, a1))
	} else {
		// min(a0*ts/r0, a1*ts/r1)
		m0 := new(big.Int).Mul(a0, ts)
		m0.Div(m0, r0)
		m1 := new(big.Int).Mul(a1, ts)
		m1.Div(m1, r1)
		if m0.Cmp(m1) < 0 {
			minted = m0
		} else {
			minted = m1
		}
	}
	if minted.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "zero liquidity minted"})
		return
	}
	newR0 := new(big.Int).Add(r0, a0)
	newR1 := new(big.Int).Add(r1, a1)
	newTS := new(big.Int).Add(ts, minted)
	_, err = tx.Exec(c, `UPDATE liquidity_pools SET reserve0=$1,reserve1=$2,total_supply=$3 WHERE id=$4`, newR0.String(), newR1.String(), newTS.String(), pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update pool"})
		return
	}
	var existingLP string
	err = tx.QueryRow(c, `SELECT lp_tokens::text FROM liquidity_positions WHERE pool_id=$1 AND user_id=$2`, pid, user).Scan(&existingLP)
	if err == nil {
		cur, _ := new(big.Int).SetString(existingLP, 10)
		if cur == nil {
			cur = big.NewInt(0)
		}
		_, err = tx.Exec(c, `UPDATE liquidity_positions SET lp_tokens=$1 WHERE pool_id=$2 AND user_id=$3`, new(big.Int).Add(cur, minted).String(), pid, user)
	} else {
		_, err = tx.Exec(c, `INSERT INTO liquidity_positions (id,pool_id,user_id,lp_tokens) VALUES ($1,$2,$3,$4)`, newID(), pid, user, minted.String())
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update position"})
		return
	}
	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "lp_tokens": minted.String(), "pool_id": pid})
}

type removeLiqReq struct {
	Pair     string `json:"pair" binding:"required"`
	LPAmount string `json:"lp_amount" binding:"required"`
}

func (s *service) removeLiquidity(c *gin.Context) {
	var req removeLiqReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lpAmt, ok := new(big.Int).SetString(req.LPAmount, 10)
	if !ok || lpAmt.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lp_amount"})
		return
	}
	user := c.GetString("user_id")
	tx, err := s.pg.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot start tx"})
		return
	}
	defer tx.Rollback(c)
	var pid, r0Str, r1Str, tsStr, userLPStr string
	err = tx.QueryRow(c, `SELECT p.id,p.reserve0::text,p.reserve1::text,p.total_supply::text,COALESCE((SELECT lp_tokens::text FROM liquidity_positions WHERE pool_id=p.id AND user_id=$2),'0') FROM liquidity_pools p WHERE p.pair=$1 FOR UPDATE`, req.Pair, user).Scan(&pid, &r0Str, &r1Str, &tsStr, &userLPStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pool not found"})
		return
	}
	r0, _ := new(big.Int).SetString(r0Str, 10)
	r1, _ := new(big.Int).SetString(r1Str, 10)
	ts, _ := new(big.Int).SetString(tsStr, 10)
	userLP, _ := new(big.Int).SetString(userLPStr, 10)
	if r0 == nil || r1 == nil || ts == nil || userLP == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid pool state"})
		return
	}
	if lpAmt.Cmp(userLP) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient LP tokens"})
		return
	}
	// amount0 = lpAmt * r0 / ts
	amount0 := new(big.Int).Mul(lpAmt, r0)
	amount0.Div(amount0, ts)
	amount1 := new(big.Int).Mul(lpAmt, r1)
	amount1.Div(amount1, ts)
	newR0 := new(big.Int).Sub(r0, amount0)
	newR1 := new(big.Int).Sub(r1, amount1)
	newTS := new(big.Int).Sub(ts, lpAmt)
	newUserLP := new(big.Int).Sub(userLP, lpAmt)
	_, err = tx.Exec(c, `UPDATE liquidity_pools SET reserve0=$1,reserve1=$2,total_supply=$3 WHERE id=$4`, newR0.String(), newR1.String(), newTS.String(), pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update pool"})
		return
	}
	if newUserLP.Sign() > 0 {
		_, err = tx.Exec(c, `UPDATE liquidity_positions SET lp_tokens=$1 WHERE pool_id=$2 AND user_id=$3`, newUserLP.String(), pid, user)
	} else {
		_, err = tx.Exec(c, `DELETE FROM liquidity_positions WHERE pool_id=$1 AND user_id=$2`, pid, user)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update position"})
		return
	}
	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "amount0": amount0.String(), "amount1": amount1.String(), "burned": lpAmt.String()})
}

func (s *service) listPositions(c *gin.Context) {
	user := c.GetString("user_id")
	rows, err := s.pg.Query(c, `SELECT p.id,p.pair,p.token0,p.token1,p.reserve0::text,p.reserve1::text,p.total_supply::text,lp.lp_tokens::text FROM liquidity_positions lp JOIN liquidity_pools p ON lp.pool_id=p.id WHERE lp.user_id=$1 ORDER BY lp.created_at DESC`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var p struct{ ID, Pair, T0, T1, R0, R1, TS, LP string }
		if err := rows.Scan(&p.ID, &p.Pair, &p.T0, &p.T1, &p.R0, &p.R1, &p.TS, &p.LP); err != nil {
			continue
		}
		out = append(out, gin.H{"pool_id": p.ID, "pair": p.Pair, "token0": p.T0, "token1": p.T1, "reserve0": p.R0, "reserve1": p.R1, "total_supply": p.TS, "lp_tokens": p.LP})
	}
	c.JSON(http.StatusOK, gin.H{"positions": out, "count": len(out)})
}

// sqrtBig computes floor(sqrt(n)) via Newton's method on big.Int.
func sqrtBig(n *big.Int) *big.Int {
	if n.Sign() <= 0 {
		return big.NewInt(0)
	}
	x := new(big.Int).Set(n)
	y := new(big.Int).Add(x, big.NewInt(1))
	y.Rsh(y, 1)
	for y.Cmp(x) < 0 {
		x.Set(y)
		y.Rsh(y, 1)
		y.Add(y, new(big.Int).Div(n, y))
		y.Rsh(y, 1)
	}
	return x
}
