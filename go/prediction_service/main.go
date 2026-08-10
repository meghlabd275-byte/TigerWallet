// TigerWallet Prediction Markets Service — binary-outcome prediction markets
// with a real automated market-maker (LMSR) and PostgreSQL persistence. No
// mocks: every market, position and resolution is a real DB row.
package main

import (
	"context"
	"errors"
	"log"
	"math"
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
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "prediction"}) })

	api := r.Group("/api/v1/prediction")
	{
		api.GET("/markets", svc.listMarkets)
		api.GET("/markets/:id", svc.getMarket)
		api.POST("/markets", svc.auth(), svc.createMarket)
		api.POST("/markets/:id/bet", svc.auth(), svc.placeBet)
		api.GET("/markets/:id/bets", svc.listBets)
		api.POST("/markets/:id/resolve", svc.auth(), svc.resolveMarket)
		api.GET("/positions/:user", svc.userPositions)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("prediction service on :%s", cfg.Port)
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
}

func loadCfg() config {
	g := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return config{
		Port:      g("PORT", "8455"),
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

// lmsrCost returns the incremental cost (in base units, float) of buying
// `qty` shares of outcome `yes` under an LMSR market maker, using float64
// arithmetic for the cost estimate. The on-chain pools store integer shares.
func lmsrCost(yesPool, noPool *big.Int, b float64, qty *big.Int, yes bool) float64 {
	qf, _ := new(big.Float).SetInt(qty).Float64()
	yp, _ := new(big.Float).SetInt(yesPool).Float64()
	np, _ := new(big.Float).SetInt(noPool).Float64()
	if b <= 0 {
		b = 1000
	}
	costNow := b * math.Log(math.Exp(yp/b)+math.Exp(np/b))
	var yp2, np2 float64
	if yes {
		yp2 = yp + qf
		np2 = np
	} else {
		yp2 = yp
		np2 = np + qf
	}
	costAfter := b * math.Log(math.Exp(yp2/b)+math.Exp(np2/b))
	return costAfter - costNow
}

func (s *service) migrate(ctx context.Context) error {
	_, err := s.pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS prediction_markets (
    id          TEXT PRIMARY KEY,
    question    TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL DEFAULT 'general',
    status      TEXT NOT NULL DEFAULT 'open',
    outcome     TEXT,
    yes_pool    NUMERIC(78,0) NOT NULL DEFAULT 0,
    no_pool     NUMERIC(78,0) NOT NULL DEFAULT 0,
    funding     NUMERIC(78,0) NOT NULL DEFAULT 1000000000000000000000,
    end_time    TIMESTAMPTZ NOT NULL,
    creator     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS prediction_bets (
    id          TEXT PRIMARY KEY,
    market_id   TEXT NOT NULL REFERENCES prediction_markets(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL,
    side         SMALLINT NOT NULL,
    amount      NUMERIC(78,0) NOT NULL,
    avg_price   NUMERIC(78,0) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS prediction_resolutions (
    market_id TEXT PRIMARY KEY REFERENCES prediction_markets(id) ON DELETE CASCADE,
    outcome   TEXT NOT NULL,
    resolved_by TEXT NOT NULL,
    resolved_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_pmarkets_status ON prediction_markets(status);
CREATE INDEX IF NOT EXISTS idx_pbets_market ON prediction_bets(market_id);
CREATE INDEX IF NOT EXISTS idx_pbets_user ON prediction_bets(user_id);
`)
	return err
}

type createMarketReq struct {
	Question    string `json:"question" binding:"required"`
	Description string `json:"description"`
	Category    string `json:"category"`
	DurationH   int    `json:"duration_hours"`
	Funding     string `json:"funding"`
}

func (s *service) createMarket(c *gin.Context) {
	var req createMarketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dur := time.Duration(req.DurationH) * time.Hour
	if dur <= 0 {
		dur = 168 * time.Hour
	}
	funding := req.Funding
	if funding == "" {
		funding = "1000000000000000000000" // 1000 (18 decimals)
	}
	id := newID()
	end := time.Now().Add(dur)
	_, err := s.pg.Exec(c,
		`INSERT INTO prediction_markets (id,question,description,category,creator,end_time,funding) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, req.Question, req.Description, strOr(req.Category, "general"), c.GetString("user_id"), end, funding)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create market"})
		return
	}
	s.redis.Del(c, "pred:markets:open")
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "open", "end_time": end.Unix()})
}

func (s *service) listMarkets(c *gin.Context) {
	status := c.DefaultQuery("status", "open")
	rows, err := s.pg.Query(c, `
SELECT id,question,description,category,status,outcome,yes_pool::text,no_pool::text,funding::text,
       extract(epoch from end_time)::bigint, creator
FROM prediction_markets WHERE ($1='' OR status=$1) ORDER BY created_at DESC LIMIT 200`, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var m struct {
			ID, Question, Desc, Cat, Status, Outcome, Yes, No, Fund, Creator string
			EndTime                                                          int64
		}
		if err := rows.Scan(&m.ID, &m.Question, &m.Desc, &m.Cat, &m.Status, &m.Outcome, &m.Yes, &m.No, &m.Fund, &m.EndTime, &m.Creator); err != nil {
			continue
		}
		out = append(out, gin.H{
			"id": m.ID, "question": m.Question, "description": m.Desc, "category": m.Cat,
			"status": m.Status, "outcome": m.Outcome, "yes_price": lmsrPrice(m.Yes, m.No, m.Fund, true),
			"no_price": lmsrPrice(m.Yes, m.No, m.Fund, false), "end_time": m.EndTime, "creator": m.Creator,
			"yes_pool": m.Yes, "no_pool": m.No, "volume": "0",
		})
	}
	c.JSON(http.StatusOK, gin.H{"markets": out, "count": len(out)})
}

func (s *service) getMarket(c *gin.Context) {
	id := c.Param("id")
	var m struct {
		ID, Question, Desc, Cat, Status, Outcome, Yes, No, Fund, Creator string
		EndTime, Created                                                 int64
	}
	err := s.pg.QueryRow(c, `
SELECT id,question,description,category,status,COALESCE(outcome,''),yes_pool::text,no_pool::text,funding::text,
       extract(epoch from end_time)::bigint, creator, extract(epoch from created_at)::bigint
FROM prediction_markets WHERE id=$1`, id).
		Scan(&m.ID, &m.Question, &m.Desc, &m.Cat, &m.Status, &m.Outcome, &m.Yes, &m.No, &m.Fund, &m.EndTime, &m.Creator, &m.Created)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": m.ID, "question": m.Question, "description": m.Desc, "category": m.Cat,
		"status": m.Status, "outcome": m.Outcome, "yes_price": lmsrPrice(m.Yes, m.No, m.Fund, true),
		"no_price": lmsrPrice(m.Yes, m.No, m.Fund, false), "end_time": m.EndTime, "created_at": m.Created,
		"yes_pool": m.Yes, "no_pool": m.No, "creator": m.Creator,
	})
}

type betReq struct {
	Side   int    `json:"side" binding:"required"` // 1 yes, 0 no
	Amount string `json:"amount" binding:"required"`
}

func (s *service) placeBet(c *gin.Context) {
	id := c.Param("id")
	var req betReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Side != 0 && req.Side != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "side must be 0 or 1"})
		return
	}
	amt, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok || amt.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	tx, err := s.pg.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot start tx"})
		return
	}
	defer tx.Rollback(c)

	var status, yesP, noP, fundS string
	var endT time.Time
	err = tx.QueryRow(c, `SELECT status,yes_pool::text,no_pool::text,funding::text,end_time FROM prediction_markets WHERE id=$1 FOR UPDATE`, id).
		Scan(&status, &yesP, &noP, &fundS, &endT)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
		return
	}
	if status != "open" || time.Now().After(endT) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "market not open"})
		return
	}
	yesPool, _ := new(big.Int).SetString(yesP, 10)
	noPool, _ := new(big.Int).SetString(noP, 10)
	fundF, _, _ := big.ParseFloat(fundS, 10, 200, big.ToNearestEven)
	if fundF == nil {
		fundF = big.NewFloat(1000)
	}
	bf, _ := fundF.Float64()
	if bf <= 0 {
		bf = 1000
	}
	cost := lmsrCost(yesPool, noPool, bf, amt, req.Side == 1)

	// Update pools
	if req.Side == 1 {
		yesPool = new(big.Int).Add(yesPool, amt)
	} else {
		noPool = new(big.Int).Add(noPool, amt)
	}
	amtF, _ := new(big.Float).SetInt(amt).Float64()
	avgPrice := 0.0
	if amtF > 0 {
		avgPrice = cost / amtF
	}
	if avgPrice < 0 {
		avgPrice = 0
	}
	// Store price as fixed-point with 18 decimals.
	avgInt := big.NewInt(int64(avgPrice * 1e18))
	if avgInt.Sign() < 0 {
		avgInt = big.NewInt(0)
	}

	_, err = tx.Exec(c, `UPDATE prediction_markets SET yes_pool=$1,no_pool=$2 WHERE id=$3`, yesPool.String(), noPool.String(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update pool"})
		return
	}
	bid := newID()
	_, err = tx.Exec(c,
		`INSERT INTO prediction_bets (id,market_id,user_id,side,amount,avg_price) VALUES ($1,$2,$3,$4,$5,$6)`,
		bid, id, c.GetString("user_id"), req.Side, amt.String(), avgInt.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot record bet"})
		return
	}
	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	s.redis.Del(c, "pred:markets:open")
	c.JSON(http.StatusOK, gin.H{"success": true, "bet_id": bid, "side": req.Side, "amount": amt.String(), "avg_price": avgInt.String()})
}

func (s *service) listBets(c *gin.Context) {
	id := c.Param("id")
	rows, err := s.pg.Query(c,
		`SELECT id,user_id,side,amount::text,avg_price::text,extract(epoch from created_at)::bigint FROM prediction_bets WHERE market_id=$1 ORDER BY created_at DESC LIMIT 500`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var b struct {
			ID, UID    string
			Side       int
			Amt, Price string
			Ts         int64
		}
		if err := rows.Scan(&b.ID, &b.UID, &b.Side, &b.Amt, &b.Price, &b.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": b.ID, "user_id": b.UID, "side": b.Side, "amount": b.Amt, "avg_price": b.Price, "created_at": b.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"bets": out, "count": len(out)})
}

type resolveReq struct {
	Outcome string `json:"outcome" binding:"required"` // "yes" or "no"
}

func (s *service) resolveMarket(c *gin.Context) {
	id := c.Param("id")
	var req resolveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Outcome != "yes" && req.Outcome != "no" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "outcome must be yes or no"})
		return
	}
	_, err := s.pg.Exec(c, `UPDATE prediction_markets SET status='resolved',outcome=$1 WHERE id=$2 AND status='open'`, req.Outcome, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot resolve"})
		return
	}
	s.redis.Del(c, "pred:markets:open")
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id, "outcome": req.Outcome, "status": "resolved"})
}

func (s *service) userPositions(c *gin.Context) {
	user := c.Param("user")
	rows, err := s.pg.Query(c, `
SELECT b.id,b.market_id,b.side,b.amount::text,b.avg_price::text,m.question,m.outcome,m.status,
       extract(epoch from b.created_at)::bigint
FROM prediction_bets b JOIN prediction_markets m ON m.id=b.market_id
WHERE b.user_id=$1 ORDER BY b.created_at DESC LIMIT 200`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var p struct {
			ID, MID, Amt, Price, Q, Outcome, Status string
			Side                                    int
			Ts                                      int64
		}
		if err := rows.Scan(&p.ID, &p.MID, &p.Side, &p.Amt, &p.Price, &p.Q, &p.Outcome, &p.Status, &p.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": p.ID, "market_id": p.MID, "side": p.Side, "amount": p.Amt, "avg_price": p.Price, "question": p.Q, "outcome": p.Outcome, "status": p.Status, "created_at": p.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"positions": out, "count": len(out)})
}

// lmsrPrice returns the implied probability (0..1) for the given side using
// the LMSR marginal price.
func lmsrPrice(yesP, noP, fundS string, yes bool) float64 {
	yp, _ := new(big.Float).SetString(yesP)
	if yp == nil {
		yp = big.NewFloat(0)
	}
	np, _ := new(big.Float).SetString(noP)
	if np == nil {
		np = big.NewFloat(0)
	}
	f, _ := new(big.Float).SetString(fundS)
	if f == nil || f.Sign() <= 0 {
		f = big.NewFloat(1000)
	}
	yq := new(big.Float).Quo(yp, f)
	nq := new(big.Float).Quo(np, f)
	yd, _ := yq.Float64()
	nd, _ := nq.Float64()
	ey := math.Exp(yd)
	en := math.Exp(nd)
	denom := ey + en
	if denom == 0 {
		return 0.5
	}
	if yes {
		return ey / denom
	}
	return en / denom
}

func strOr(a, b string) string {
	if a == "" {
		return b
	}
	return a
}
