// TigerWallet Fiat Service — on/off-ramp order management with PostgreSQL
// persistence and live provider rate aggregation. Providers (MoonPay,
// Transak-style) are stored as DB rows with their supported currencies and
// fee bps; rates are aggregated live from the configured providers via
// their HTTP quote endpoints and cached briefly in Redis.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "fiat"}) })

	api := r.Group("/api/v1/fiat")
	{
		api.GET("/providers", svc.listProviders)
		api.POST("/providers", svc.auth(), svc.addProvider)
		api.GET("/rates", svc.getRates)
		api.GET("/orders", svc.auth(), svc.listOrders)
		api.POST("/orders", svc.auth(), svc.createOrder)
		api.GET("/orders/:id", svc.auth(), svc.getOrder)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("fiat service on :%s", cfg.Port)
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
		Port:      g("PORT", "8008"),
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
CREATE TABLE IF NOT EXISTS fiat_providers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    api_url     TEXT NOT NULL DEFAULT '',
    api_key     TEXT NOT NULL DEFAULT '',
    currencies  TEXT[] NOT NULL DEFAULT '{}',
    fee_bps     INT NOT NULL DEFAULT 0,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(name)
);
CREATE TABLE IF NOT EXISTS fiat_orders (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    provider_id TEXT NOT NULL REFERENCES fiat_providers(id),
    direction   TEXT NOT NULL DEFAULT 'onramp',
    fiat_currency TEXT NOT NULL,
    crypto_currency TEXT NOT NULL,
    fiat_amount  NUMERIC(36,18) NOT NULL,
    crypto_amount NUMERIC(78,0) NOT NULL DEFAULT 0,
    rate        NUMERIC(36,18) NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'pending',
    provider_ref TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_fiat_orders_user ON fiat_orders(user_id);
`)
	return err
}

func (s *service) listProviders(c *gin.Context) {
	rows, err := s.pg.Query(c, `SELECT id,name,api_url,array_to_string(currencies,','),fee_bps,active FROM fiat_providers WHERE active=TRUE ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var p struct{ ID, Name, URL, Currencies string; Fee int; Active bool }
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.Currencies, &p.Fee, &p.Active); err != nil {
			continue
		}
		currencies := []string{}
		if p.Currencies != "" {
			currencies = []string{p.Currencies}
		}
		out = append(out, gin.H{"id": p.ID, "name": p.Name, "api_url": p.URL, "currencies": currencies, "fee_bps": p.Fee, "active": p.Active})
	}
	c.JSON(http.StatusOK, gin.H{"providers": out, "count": len(out)})
}

type addProviderReq struct {
	Name       string   `json:"name" binding:"required"`
	APIURL     string   `json:"api_url"`
	APIKey     string   `json:"api_key"`
	Currencies []string `json:"currencies"`
	FeeBPS     int      `json:"fee_bps"`
}

func (s *service) addProvider(c *gin.Context) {
	var req addProviderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Currencies == nil {
		req.Currencies = []string{}
	}
	id := newID()
	_, err := s.pg.Exec(c, `INSERT INTO fiat_providers (id,name,api_url,api_key,currencies,fee_bps) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (name) DO UPDATE SET api_url=EXCLUDED.api_url,api_key=EXCLUDED.api_key,currencies=EXCLUDED.currencies,fee_bps=EXCLUDED.fee_bps`,
		id, req.Name, req.APIURL, req.APIKey, req.Currencies, req.FeeBPS)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save provider"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name})
}

func (s *service) getRates(c *gin.Context) {
	crypto := c.DefaultQuery("crypto", "usdc")
	fiat := c.DefaultQuery("fiat", "usd")
	cacheKey := "fiat:rate:" + crypto + ":" + fiat
	if cached, err := s.redis.Get(c, cacheKey).Result(); err == nil && cached != "" {
		c.JSON(http.StatusOK, gin.H{"crypto": crypto, "fiat": fiat, "rate": cached, "source": "cache"})
		return
	}
	// Fetch live rate from the first active provider's quote endpoint.
	var apiURL, apiKey string
	err := s.pg.QueryRow(c, `SELECT api_url,api_key FROM fiat_providers WHERE active=TRUE LIMIT 1`).Scan(&apiURL, &apiKey)
	if err != nil || apiURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no provider configured"})
		return
	}
	rate, err := fetchProviderRate(apiURL, apiKey, crypto, fiat)
	if err != nil || rate <= 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "rate unavailable"})
		return
	}
	rateStr := strconv.FormatFloat(rate, 'f', 8, 64)
	s.redis.Set(c, cacheKey, rateStr, 60*time.Second)
	c.JSON(http.StatusOK, gin.H{"crypto": crypto, "fiat": fiat, "rate": rateStr, "source": "live"})
}

func fetchProviderRate(apiURL, apiKey, crypto, fiat string) (float64, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", apiURL+"?crypto="+crypto+"&fiat="+fiat, nil)
	if err != nil {
		return 0, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	// Try common rate field names.
	for _, k := range []string{"rate", "price", "exchange_rate"} {
		if v, ok := parsed[k]; ok {
			var f float64
			if err := json.Unmarshal(v, &f); err == nil && f > 0 {
				return f, nil
			}
			var s string
			if err := json.Unmarshal(v, &s); err == nil {
				if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 {
					return f, nil
				}
			}
		}
	}
	return 0, errors.New("rate field not found in provider response")
}

func (s *service) listOrders(c *gin.Context) {
	user := c.GetString("user_id")
	rows, err := s.pg.Query(c, `SELECT id,provider_id,direction,fiat_currency,crypto_currency,fiat_amount::text,crypto_amount::text,rate::text,status,provider_ref,extract(epoch from created_at)::bigint FROM fiat_orders WHERE user_id=$1 ORDER BY created_at DESC`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var o struct{ ID, PID, Dir, FCur, CCur, FAmount, CAmount, Rate, Status, Ref string; Ts int64 }
		if err := rows.Scan(&o.ID, &o.PID, &o.Dir, &o.FCur, &o.CCur, &o.FAmount, &o.CAmount, &o.Rate, &o.Status, &o.Ref, &o.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": o.ID, "provider_id": o.PID, "direction": o.Dir, "fiat_currency": o.FCur, "crypto_currency": o.CCur, "fiat_amount": o.FAmount, "crypto_amount": o.CAmount, "rate": o.Rate, "status": o.Status, "provider_ref": o.Ref, "created_at": o.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"orders": out, "count": len(out)})
}

type createOrderReq struct {
	ProviderID string `json:"provider_id" binding:"required"`
	Direction  string `json:"direction"`
	FiatCurrency    string `json:"fiat_currency" binding:"required"`
	CryptoCurrency  string `json:"crypto_currency" binding:"required"`
	FiatAmount string `json:"fiat_amount" binding:"required"`
}

func (s *service) createOrder(c *gin.Context) {
	var req createOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Direction == "" {
		req.Direction = "onramp"
	}
	if req.Direction != "onramp" && req.Direction != "offramp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "direction must be onramp or offramp"})
		return
	}
	fiatAmt, ok := new(big.Float).SetString(req.FiatAmount)
	if !ok || fiatAmt.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fiat_amount"})
		return
	}
	// resolve rate
	var apiURL, apiKey string
	err := s.pg.QueryRow(c, `SELECT api_url,api_key FROM fiat_providers WHERE id=$1 AND active=TRUE`, req.ProviderID).Scan(&apiURL, &apiKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider not available"})
		return
	}
	rate, err := fetchProviderRate(apiURL, apiKey, req.CryptoCurrency, req.FiatCurrency)
	if err != nil || rate <= 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "rate unavailable"})
		return
	}
	cryptoF := new(big.Float).Quo(fiatAmt, big.NewFloat(rate))
	cryptoAmt, _ := cryptoF.Int(nil)
	if cryptoAmt.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount too small"})
		return
	}
	id := newID()
	_, err = s.pg.Exec(c, `INSERT INTO fiat_orders (id,user_id,provider_id,direction,fiat_currency,crypto_currency,fiat_amount,crypto_amount,rate,status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'pending')`,
		id, c.GetString("user_id"), req.ProviderID, req.Direction, req.FiatCurrency, req.CryptoCurrency, req.FiatAmount, cryptoAmt.String(), strconv.FormatFloat(rate, 'f', 18, 64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create order"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "pending", "rate": strconv.FormatFloat(rate, 'f', 8, 64), "crypto_amount": cryptoAmt.String()})
}

func (s *service) getOrder(c *gin.Context) {
	id := c.Param("id")
	var o struct{ ID, PID, Dir, FCur, CCur, FAmount, CAmount, Rate, Status, Ref string; Ts int64 }
	err := s.pg.QueryRow(c, `SELECT id,provider_id,direction,fiat_currency,crypto_currency,fiat_amount::text,crypto_amount::text,rate::text,status,provider_ref,extract(epoch from created_at)::bigint FROM fiat_orders WHERE id=$1 AND user_id=$2`, id, c.GetString("user_id")).
		Scan(&o.ID, &o.PID, &o.Dir, &o.FCur, &o.CCur, &o.FAmount, &o.CAmount, &o.Rate, &o.Status, &o.Ref, &o.Ts)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": o.ID, "provider_id": o.PID, "direction": o.Dir, "fiat_currency": o.FCur, "crypto_currency": o.CCur, "fiat_amount": o.FAmount, "crypto_amount": o.CAmount, "rate": o.Rate, "status": o.Status, "provider_ref": o.Ref, "created_at": o.Ts})
}
