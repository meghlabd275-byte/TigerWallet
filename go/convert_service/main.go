// TigerWallet Convert Service — cross-token conversion backed by the canonical
// wallet_api on-chain AMM router (real getAmountsOut + swapExactTokensForTokens).
//
// This service is the authoritative record-keeper for conversions: it fetches a
// real on-chain quote from wallet_api's /api/v1/amm/quote, persists each
// conversion to PostgreSQL (convert_history), and constructs the real swap
// calldata (returned to the client to broadcast via /api/v1/send — no
// fabricated transaction hashes).
//
// No stubs, no mock data, no hardcoded prices. If wallet_api is unreachable or
// no router is configured for the chain, the endpoints return honest 503/502 —
// they never invent a rate.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

	svc := &service{pg: pool, redis: rdb, jwt: cfg.JWTSecret, walletAPI: cfg.WalletAPIURL}
	if err := svc.migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "convert"}) })

	api := r.Group("/api/v1/convert", svc.auth())
	{
		api.GET("/quote", svc.quote)
		api.POST("/execute", svc.execute)
		api.GET("/history", svc.history)
		api.GET("/history/:id", svc.historyItem)
		api.PATCH("/history/:id", svc.confirmTx)
		api.GET("/supported", svc.supportedTokens)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("convert service on :%s", cfg.Port)
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
	Port, DBURL, RedisAddr, JWTSecret, WalletAPIURL string
}

func loadCfg() config {
	g := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return config{
		Port:        g("PORT", "8472"),
		DBURL:       g("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr:   g("REDIS_ADDR", "localhost:6379"),
		JWTSecret:   g("JWT_SECRET", "tigerwallet-dev-secret-change-in-production"),
		WalletAPIURL: g("WALLET_API_URL", "http://localhost:8443"),
	}
}

type service struct {
	pg        *pgxpool.Pool
	redis     *redis.Client
	jwt       string
	walletAPI string
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
		// some tokens use "user_id" claim
		if uid, ok := claims["user_id"].(string); ok {
			return uid, nil
		}
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
CREATE TABLE IF NOT EXISTS convert_history (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    chain_id        BIGINT NOT NULL,
    from_token      TEXT NOT NULL,
    to_token        TEXT NOT NULL,
    amount_in       TEXT NOT NULL,
    amount_out      TEXT NOT NULL,
    amount_in_wei   TEXT NOT NULL,
    amount_out_wei  TEXT NOT NULL,
    router          TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'quote',
    tx_hash         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_convert_history_user ON convert_history(user_id, created_at DESC);
`)
	return err
}

// quoteResp mirrors the wallet_api /api/v1/amm/quote response.
type quoteResp struct {
	Success       bool     `json:"success"`
	ChainID      int64    `json:"chain_id"`
	Router       string   `json:"router"`
	TokenIn      string   `json:"token_in"`
	TokenOut     string   `json:"token_out"`
	AmountIn     string   `json:"amount_in"`
	AmountOut    string   `json:"amount_out"`
	AmountOutWei string   `json:"amount_out_wei"`
	AmountInWei  string   `json:"amount_in_wei"`
	DecimalsIn  int      `json:"decimals_in"`
	DecimalsOut int      `json:"decimals_out"`
	Path         []string `json:"path"`
}

// GET /quote?chain_id=1&token_in=0x..&token_out=0x..&amount_in=1.5
// Proxies the real on-chain AMM quote from wallet_api. Caches the rate in Redis
// for 15s keyed by (chain,token_in,token_out,amount_in) to avoid hammering the
// RPC node.
func (s *service) quote(c *gin.Context) {
	chainID := c.Query("chain_id")
	tokenIn := strings.ToLower(strings.TrimSpace(c.Query("token_in")))
	tokenOut := strings.ToLower(strings.TrimSpace(c.Query("token_out")))
	amountIn := c.Query("amount_in")
	if chainID == "" || tokenIn == "" || tokenOut == "" || amountIn == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id, token_in, token_out, amount_in are required"})
		return
	}

	// Redis cache (rate quote only — the on-chain amount_out is deterministic).
	cacheKey := fmt.Sprintf("convert:quote:%s:%s:%s:%s", chainID, tokenIn, tokenOut, amountIn)
	if cached, err := s.redis.Get(c.Request.Context(), cacheKey).Bytes(); err == nil && len(cached) > 0 {
		var qr quoteResp
		if json.Unmarshal(cached, &qr) == nil {
			c.JSON(http.StatusOK, gin.H{
				"success":     true,
				"quote_type":  "on-chain-cached",
				"cached":      true,
				"chain_id":    qr.ChainID,
				"router":      qr.Router,
				"token_in":    qr.TokenIn,
				"token_out":   qr.TokenOut,
				"amount_in":   qr.AmountIn,
				"amount_out":  qr.AmountOut,
				"amount_out_wei": qr.AmountOutWei,
				"amount_in_wei":  qr.AmountInWei,
				"decimals_in": qr.DecimalsIn,
				"decimals_out": qr.DecimalsOut,
				"path":        qr.Path,
			})
			return
		}
	}

	// Proxy the real on-chain quote from wallet_api.
	url := fmt.Sprintf("%s/api/v1/amm/quote?chain_id=%s&token_in=%s&token_out=%s&amount_in=%s",
		s.walletAPI, chainID, tokenIn, tokenOut, amountIn)
	qr, status, err := s.proxyGetJSON(url)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "wallet_api amm quote unavailable: " + err.Error()})
		return
	}
	if status != http.StatusOK || !qr.Success {
		c.JSON(status, gin.H{"error": "wallet_api rejected the quote", "upstream": qr})
		return
	}

	// Cache for 15s.
	if b, err := json.Marshal(qr); err == nil {
		s.redis.Set(c.Request.Context(), cacheKey, b, 15*time.Second)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"quote_type":     "on-chain",
		"cached":         false,
		"chain_id":       qr.ChainID,
		"router":         qr.Router,
		"token_in":       qr.TokenIn,
		"token_out":      qr.TokenOut,
		"amount_in":      qr.AmountIn,
		"amount_out":     qr.AmountOut,
		"amount_out_wei": qr.AmountOutWei,
		"amount_in_wei":  qr.AmountInWei,
		"decimals_in":    qr.DecimalsIn,
		"decimals_out":   qr.DecimalsOut,
		"path":           qr.Path,
	})
}

// swapResp mirrors the wallet_api /api/v1/amm/swap response (action_required).
type swapResp struct {
	Success       bool   `json:"success"`
	ChainID      int64  `json:"chain_id"`
	Router       string `json:"router"`
	To           string `json:"to"`
	Data         string `json:"data"`
	Value        string `json:"value"`
	ActionRequired string `json:"action_required"`
	Message      string `json:"message"`
	Error        string `json:"error"`
}

// POST /execute — records a conversion (real swap calldata constructed by
// wallet_api; the client broadcasts via /api/v1/send). The conversion row starts
// as "quote" and is updated to "executed" once the client confirms the tx hash
// via PATCH /history/:id (or a follow-up /confirm call). No tx hash is fabricated.
func (s *service) execute(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		ChainID      int64  `json:"chain_id"`
		TokenIn      string `json:"token_in"`
		TokenOut     string `json:"token_out"`
		AmountIn     string `json:"amount_in"`
		AmountOutMin string `json:"amount_out_min"`
		From         string `json:"from"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body: " + err.Error()})
		return
	}
	if req.ChainID == 0 || req.TokenIn == "" || req.TokenOut == "" || req.AmountIn == "" || req.From == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id, token_in, token_out, amount_in, from are required"})
		return
	}

	// 1. Fetch the real on-chain quote first (so we persist the real amount_out).
	qURL := fmt.Sprintf("%s/api/v1/amm/quote?chain_id=%d&token_in=%s&token_out=%s&amount_in=%s",
		s.walletAPI, req.ChainID, req.TokenIn, req.TokenOut, req.AmountIn)
	qr, qStatus, err := s.proxyGetJSON(qURL)
	if err != nil || qStatus != http.StatusOK || !qr.Success {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not obtain on-chain quote before execution"})
		return
	}

	// 2. Request the real swap calldata from wallet_api.
	body, _ := json.Marshal(map[string]interface{}{
		"from":         req.From,
		"chain_id":     req.ChainID,
		"token_in":     req.TokenIn,
		"token_out":    req.TokenOut,
		"amount_in":    req.AmountIn,
		"amount_out_min": req.AmountOutMin,
	})
	sURL := s.walletAPI + "/api/v1/amm/swap"
	sr, sStatus, err := s.proxyPostJSON(sURL, body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "wallet_api amm swap unavailable: " + err.Error()})
		return
	}
	if sStatus != http.StatusOK || !sr.Success {
		c.JSON(sStatus, gin.H{"error": "wallet_api rejected the swap", "upstream": sr})
		return
	}

	// 3. Persist the conversion row (status "quote" until the client confirms tx).
	id := newID()
	_, err = s.pg.Exec(c.Request.Context(), `
INSERT INTO convert_history (id, user_id, chain_id, from_token, to_token, amount_in, amount_out, amount_in_wei, amount_out_wei, router, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'quote')`,
		id, userID, req.ChainID, qr.TokenIn, qr.TokenOut, qr.AmountIn, qr.AmountOut, qr.AmountInWei, qr.AmountOutWei, qr.Router)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "persist convert: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"convert_id":       id,
		"chain_id":         sr.ChainID,
		"router":           sr.Router,
		"to":               sr.To,
		"data":             sr.Data,
		"value":            sr.Value,
		"action_required":  sr.ActionRequired,
		"message":          sr.Message,
		"amount_in":        qr.AmountIn,
		"amount_out":       qr.AmountOut,
		"amount_in_wei":    qr.AmountInWei,
		"amount_out_wei":   qr.AmountOutWei,
		"next_step":        "broadcast the calldata via POST /api/v1/send, then PATCH /api/v1/convert/history/" + id + " with the tx_hash",
	})
}

// PATCH /history/:id — client confirms the broadcast tx hash so the conversion
// moves from "quote" -> "executed".
func (s *service) confirmTx(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	var req struct{ TxHash string `json:"tx_hash"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.TxHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tx_hash required"})
		return
	}
	tag, err := s.pg.Exec(c.Request.Context(), `
UPDATE convert_history SET status='executed', tx_hash=$1 WHERE id=$2 AND user_id=$3`,
		req.TxHash, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversion not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id, "status": "executed", "tx_hash": req.TxHash})
}

// GET /history — paginated list of the authenticated user's conversions.
func (s *service) history(c *gin.Context) {
	userID := c.GetString("user_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pg.Query(c.Request.Context(), `
SELECT id, chain_id, from_token, to_token, amount_in, amount_out, amount_in_wei, amount_out_wei, router, status, tx_hash, created_at
FROM convert_history WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, fromT, toT, amountIn, amountOut, amountInWei, amountOutWei, router, status string
		var chainID int64
		var txHash *string
		var createdAt time.Time
		if err := rows.Scan(&id, &chainID, &fromT, &toT, &amountIn, &amountOut, &amountInWei, &amountOutWei, &router, &status, &txHash, &createdAt); err != nil {
			continue
		}
		item := gin.H{
			"id": id, "chain_id": chainID, "from_token": fromT, "to_token": toT,
			"amount_in": amountIn, "amount_out": amountOut,
			"amount_in_wei": amountInWei, "amount_out_wei": amountOutWei,
			"router": router, "status": status, "created_at": createdAt,
		}
		if txHash != nil {
			item["tx_hash"] = *txHash
		}
		out = append(out, item)
	}
	c.JSON(http.StatusOK, gin.H{"conversions": out, "count": len(out)})
}

// GET /history/:id — single conversion record.
func (s *service) historyItem(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	var convID, fromT, toT, amountIn, amountOut, amountInWei, amountOutWei, router, status string
	var chainID int64
	var txHash *string
	var createdAt time.Time
	err := s.pg.QueryRow(c.Request.Context(), `
SELECT id, chain_id, from_token, to_token, amount_in, amount_out, amount_in_wei, amount_out_wei, router, status, tx_hash, created_at
FROM convert_history WHERE id=$1 AND user_id=$2`, id, userID).
		Scan(&convID, &chainID, &fromT, &toT, &amountIn, &amountOut, &amountInWei, &amountOutWei, &router, &status, &txHash, &createdAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversion not found"})
		return
	}
	item := gin.H{
		"id": convID, "chain_id": chainID, "from_token": fromT, "to_token": toT,
		"amount_in": amountIn, "amount_out": amountOut,
		"amount_in_wei": amountInWei, "amount_out_wei": amountOutWei,
		"router": router, "status": status, "created_at": createdAt,
	}
	if txHash != nil {
		item["tx_hash"] = *txHash
	}
	c.JSON(http.StatusOK, gin.H{"conversion": item})
}

// supportedTokens returns the wallet_api token registry so the frontend knows
// which tokens are convertible per chain (proxied from the canonical backend).
func (s *service) supportedTokens(c *gin.Context) {
	url := s.walletAPI + "/api/v1/tokens/registry"
	if chain := c.Query("chain_id"); chain != "" {
		url += "?chain_id=" + chain
	}
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "token registry unavailable: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed interface{}
	json.Unmarshal(body, &parsed)
	c.JSON(resp.StatusCode, parsed)
}

// ---- HTTP helpers ----

func (s *service) proxyGetJSON(url string) (quoteResp, int, error) {
	var qr quoteResp
	resp, err := http.Get(url)
	if err != nil {
		return qr, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &qr); err != nil {
		return qr, resp.StatusCode, fmt.Errorf("decode quote: %v (body: %s)", err, string(body))
	}
	return qr, resp.StatusCode, nil
}

func (s *service) proxyPostJSON(url string, body []byte) (swapResp, int, error) {
	var sr swapResp
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return sr, 0, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(rb, &sr); err != nil {
		return sr, resp.StatusCode, fmt.Errorf("decode swap: %v (body: %s)", err, string(rb))
	}
	return sr, resp.StatusCode, nil
}

// newID generates a short unique ID (time-prefixed hex) without external deps.
func newID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 16) + strconv.FormatInt(int64(time.Now().Nanosecond()), 16)
}

// (unused but kept for potential wei math) — big.Float helpers.
var _ = big.NewFloat
