// TigerWallet Card Service — virtual crypto-backed card balances and
// transaction history with PostgreSQL persistence and Redis caching. No
// mock balances or transactions: every entry is a real DB row.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	svc := &service{pg: pool, redis: rdb, jwt: cfg.JWTSecret, cgAPIKey: cfg.CoinGeckoAPIKey}
	if err := svc.migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "card"}) })

	api := r.Group("/api/v1/card", svc.auth())
	{
		api.GET("/balance", svc.getBalance)
		api.GET("/transactions", svc.listTransactions)
		api.POST("/transactions", svc.createTransaction)
		api.GET("/rates", svc.getRates)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("card service on :%s", cfg.Port)
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
	Port, DBURL, RedisAddr, JWTSecret, CoinGeckoAPIKey string
}

func loadCfg() config {
	g := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return config{
		Port:            g("PORT", "8457"),
		DBURL:           g("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr:       g("REDIS_ADDR", "localhost:6379"),
		JWTSecret:       g("JWT_SECRET", ""),
		CoinGeckoAPIKey: g("COINGECKO_API_KEY", ""),
	}
}

type service struct {
	pg       *pgxpool.Pool
	redis    *redis.Client
	jwt      string
	cgAPIKey string
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

func (s *service) migrate(ctx context.Context) error {
	_, err := s.pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS card_accounts (
    user_id        TEXT PRIMARY KEY,
    currency       TEXT NOT NULL DEFAULT 'USD',
    credit_limit   NUMERIC(18,2) NOT NULL DEFAULT 10000,
    available      NUMERIC(18,2) NOT NULL DEFAULT 10000,
    daily_limit    NUMERIC(18,2) NOT NULL DEFAULT 10000,
    monthly_limit  NUMERIC(18,2) NOT NULL DEFAULT 50000,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS card_transactions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    merchant    TEXT NOT NULL,
    amount       NUMERIC(18,2) NOT NULL,
    currency     TEXT NOT NULL DEFAULT 'USD',
    category     TEXT NOT NULL DEFAULT 'other',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_card_tx_user ON card_transactions(user_id);
`)
	return err
}

func (s *service) ensureAccount(c *gin.Context, user string) {
	_, err := s.pg.Exec(c,
		`INSERT INTO card_accounts (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, user)
	if err != nil {
		_ = c.Error(err)
	}
}

func (s *service) getBalance(c *gin.Context) {
	user := c.GetString("user_id")
	s.ensureAccount(c, user)
	var cur string
	var avail, dlimit, mlimit float64
	err := s.pg.QueryRow(c,
		`SELECT currency,available,daily_limit,monthly_limit FROM card_accounts WHERE user_id=$1`, user).
		Scan(&cur, &avail, &dlimit, &mlimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot read balance"})
		return
	}
	// used today / month from real transactions
	var usedToday, usedMonth float64
	s.pg.QueryRow(c,
		`SELECT COALESCE(sum(amount),0) FROM card_transactions WHERE user_id=$1 AND amount<0 AND created_at >= date_trunc('day',now())`, user).Scan(&usedToday)
	usedToday = -usedToday
	s.pg.QueryRow(c,
		`SELECT COALESCE(sum(amount),0) FROM card_transactions WHERE user_id=$1 AND amount<0 AND created_at >= date_trunc('month',now())`, user).Scan(&usedMonth)
	usedMonth = -usedMonth
	c.JSON(http.StatusOK, gin.H{
		"balance":          avail,
		"currency":         cur,
		"available_credit": avail,
		"daily_limit":      dlimit,
		"monthly_limit":    mlimit,
		"used_today":       usedToday,
		"used_month":       usedMonth,
	})
}

func (s *service) listTransactions(c *gin.Context) {
	user := c.GetString("user_id")
	rows, err := s.pg.Query(c,
		`SELECT id,merchant,amount::text,currency,category,extract(epoch from created_at)::bigint
FROM card_transactions WHERE user_id=$1 ORDER BY created_at DESC LIMIT 200`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var t struct {
			ID, Mer, Amt, Cur, Cat string
			Ts                     int64
		}
		if err := rows.Scan(&t.ID, &t.Mer, &t.Amt, &t.Cur, &t.Cat, &t.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": t.ID, "merchant": t.Mer, "amount": t.Amt, "currency": t.Cur, "category": t.Cat, "timestamp": t.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"transactions": out, "count": len(out)})
}

type txReq struct {
	Merchant string  `json:"merchant" binding:"required"`
	Amount   float64 `json:"amount" binding:"required"`
	Currency string  `json:"currency"`
	Category string  `json:"category"`
}

func (s *service) createTransaction(c *gin.Context) {
	if !s.enforceFeature(c, GatedFeature) {
		return
	}
	user := c.GetString("user_id")
	var req txReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.Category == "" {
		req.Category = "other"
	}

	tx, err := s.pg.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot start tx"})
		return
	}
	defer tx.Rollback(c)

	var avail float64
	err = tx.QueryRow(c, `SELECT available FROM card_accounts WHERE user_id=$1 FOR UPDATE`, user).Scan(&avail)
	if err != nil {
		// create on demand
		_, _ = tx.Exec(c, `INSERT INTO card_accounts (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, user)
		err = tx.QueryRow(c, `SELECT available FROM card_accounts WHERE user_id=$1 FOR UPDATE`, user).Scan(&avail)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no card account"})
			return
		}
	}
	newAvail := avail + req.Amount // purchases are negative
	if newAvail < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient credit"})
		return
	}
	_, err = tx.Exec(c, `UPDATE card_accounts SET available=$1 WHERE user_id=$2`, newAvail, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update balance"})
		return
	}
	id := newID()
	_, err = tx.Exec(c,
		`INSERT INTO card_transactions (id,user_id,merchant,amount,currency,category) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, user, req.Merchant, req.Amount, req.Currency, req.Category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot record transaction"})
		return
	}
	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id, "new_balance": newAvail})
}

// fundingAssets are the card program's supported top-up assets, priced in USD.
// The rates are REAL (CoinGecko simple/price), cached in Redis for 60s, and
// fail-closed: if the price oracle is unreachable the endpoint returns 502/503
// rather than ever fabricating a conversion rate.
var fundingAssets = []string{"ethereum", "bitcoin", "binancecoin", "usd-coin", "tether"}

func (s *service) getRates(c *gin.Context) {
	const cacheKey = "card:funding-rates"
	if cached, err := s.redis.Get(c, cacheKey).Result(); err == nil && cached != "" {
		c.Data(http.StatusOK, "application/json", []byte(cached))
		return
	}
	req, err := http.NewRequestWithContext(c, http.MethodGet,
		"https://api.coingecko.com/api/v3/simple/price?ids=ethereum,bitcoin,binancecoin,usd-coin,tether&vs_currencies=usd", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot build price request"})
		return
	}
	if s.cgAPIKey != "" {
		req.Header.Set("x-cg-demo-api-key", s.cgAPIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "price oracle unavailable"})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("price oracle error (HTTP %d)", resp.StatusCode)})
		return
	}
	var prices map[string]map[string]float64
	if err := json.Unmarshal(body, &prices); err != nil || len(prices) == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "price oracle returned no data"})
		return
	}
	rates := gin.H{}
	for _, id := range fundingAssets {
		if usd, ok := prices[id]["usd"]; ok && usd > 0 {
			rates[id] = gin.H{"usd": usd}
		}
	}
	if len(rates) == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "no rates available from oracle"})
		return
	}
	payload, _ := json.Marshal(gin.H{"rates": rates, "count": len(rates)})
	s.redis.Set(c, cacheKey, payload, 60*time.Second)
	c.Data(http.StatusOK, "application/json", payload)
}
