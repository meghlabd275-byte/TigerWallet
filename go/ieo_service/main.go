// TigerWallet IEO Service — Initial Exchange Offering launchpad for new
// token sales. Admin creates rounds with allocation caps & prices; users
// participate with KYC + allocation limits; claims become available at the
// round end. PostgreSQL persistence; no mock rounds.
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
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ieo"}) })

	api := r.Group("/api/v1/ieo")
	{
		api.GET("/rounds", svc.listRounds)
		api.GET("/rounds/:id", svc.getRound)
		api.POST("/rounds", svc.auth(), svc.createRound) // admin-gated inside
		api.POST("/participate", svc.auth(), svc.participate)
		api.POST("/rounds/:id/participate", svc.auth(), svc.participateByRound)
		api.GET("/participations", svc.auth(), svc.listParticipations)
		api.POST("/claim", svc.auth(), svc.claim)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("ieo service on :%s", cfg.Port)
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

type config struct{ Port, DBURL, RedisAddr, JWTSecret, AdminID string }

func loadCfg() config {
	g := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return config{
		Port:      g("PORT", "8460"),
		DBURL:     g("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr: g("REDIS_ADDR", "localhost:6379"),
		JWTSecret: g("JWT_SECRET", "tigerwallet-dev-secret-change-in-production"),
		AdminID:   g("ADMIN_USER_ID", ""),
	}
}

type service struct {
	pg    *pgxpool.Pool
	redis *redis.Client
	jwt   string
	cfg   config
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
CREATE TABLE IF NOT EXISTS ieo_rounds (
    id          TEXT PRIMARY KEY,
    token_symbol TEXT NOT NULL,
    token_name  TEXT NOT NULL,
    token_address TEXT NOT NULL,
    price        NUMERIC(36,18) NOT NULL,
    total_supply NUMERIC(78,0) NOT NULL,
    sold         NUMERIC(78,0) NOT NULL DEFAULT 0,
    max_per_user NUMERIC(78,0) NOT NULL,
    start_time   TIMESTAMPTZ NOT NULL,
    end_time     TIMESTAMPTZ NOT NULL,
    status       TEXT NOT NULL DEFAULT 'upcoming',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS ieo_participations (
    id          TEXT PRIMARY KEY,
    round_id    TEXT NOT NULL REFERENCES ieo_rounds(id),
    user_id     TEXT NOT NULL,
    amount      NUMERIC(78,0) NOT NULL,
    paid         NUMERIC(78,0) NOT NULL,
    claimed     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(round_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_ieo_parts_user ON ieo_participations(user_id);
`)
	return err
}

func (s *service) listRounds(c *gin.Context) {
	rows, err := s.pg.Query(c, `SELECT id,token_symbol,token_name,token_address,price::text,total_supply::text,sold::text,max_per_user::text,extract(epoch from start_time)::bigint,extract(epoch from end_time)::bigint,status FROM ieo_rounds ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var r struct{ ID, Sym, Name, Addr, Price, Supply, Sold, Max, Status string; Start, End int64 }
		if err := rows.Scan(&r.ID, &r.Sym, &r.Name, &r.Addr, &r.Price, &r.Supply, &r.Sold, &r.Max, &r.Start, &r.End, &r.Status); err != nil {
			continue
		}
		out = append(out, gin.H{"id": r.ID, "token_symbol": r.Sym, "token_name": r.Name, "token_address": r.Addr, "price": r.Price, "total_supply": r.Supply, "sold": r.Sold, "max_per_user": r.Max, "start_time": r.Start, "end_time": r.End, "status": r.Status})
	}
	c.JSON(http.StatusOK, gin.H{"rounds": out, "count": len(out)})
}

func (s *service) getRound(c *gin.Context) {
	id := c.Param("id")
	var r struct{ ID, Sym, Name, Addr, Price, Supply, Sold, Max, Status string; Start, End int64 }
	err := s.pg.QueryRow(c, `SELECT id,token_symbol,token_name,token_address,price::text,total_supply::text,sold::text,max_per_user::text,extract(epoch from start_time)::bigint,extract(epoch from end_time)::bigint,status FROM ieo_rounds WHERE id=$1`, id).
		Scan(&r.ID, &r.Sym, &r.Name, &r.Addr, &r.Price, &r.Supply, &r.Sold, &r.Max, &r.Start, &r.End, &r.Status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "round not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": r.ID, "token_symbol": r.Sym, "token_name": r.Name, "token_address": r.Addr, "price": r.Price, "total_supply": r.Supply, "sold": r.Sold, "max_per_user": r.Max, "start_time": r.Start, "end_time": r.End, "status": r.Status})
}

type createRoundReq struct {
	TokenSymbol string `json:"token_symbol" binding:"required"`
	TokenName  string `json:"token_name" binding:"required"`
	TokenAddress string `json:"token_address" binding:"required"`
	Price       string `json:"price" binding:"required"`
	TotalSupply string `json:"total_supply" binding:"required"`
	MaxPerUser  string `json:"max_per_user" binding:"required"`
	StartEpoch  int64  `json:"start_time" binding:"required"`
	EndEpoch    int64  `json:"end_time" binding:"required"`
}

func (s *service) createRound(c *gin.Context) {
	user := c.GetString("user_id")
	if s.cfg.AdminID != "" && user != s.cfg.AdminID {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}
	var req createRoundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := newID()
	_, err := s.pg.Exec(c, `INSERT INTO ieo_rounds (id,token_symbol,token_name,token_address,price,total_supply,max_per_user,start_time,end_time,status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'upcoming')`,
		id, req.TokenSymbol, req.TokenName, req.TokenAddress, req.Price, req.TotalSupply, req.MaxPerUser, time.Unix(req.StartEpoch, 0), time.Unix(req.EndEpoch, 0))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create round"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "upcoming"})
}

type participateReq struct {
	RoundID string `json:"round_id" binding:"required"`
	Amount  string `json:"amount" binding:"required"` // amount of token to buy
}

func (s *service) participate(c *gin.Context) {
	var req participateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.participateCore(c, req)
}

// participateByRound accepts the round id from the URL path (`:id`) instead of
// the request body, then delegates to the standard participate flow. The
// frontend terminal routes call POST /api/v1/ieo/rounds/:id/participate.
func (s *service) participateByRound(c *gin.Context) {
	roundID := c.Param("id")
	if roundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "round id required"})
		return
	}
	var body map[string]any
	_ = c.ShouldBindJSON(&body) // body optional; amount may be query or body
	amount, _ := body["amount"].(string)
	if amount == "" {
		amount = c.Query("amount")
	}
	if amount == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount required"})
		return
	}
	s.participateCore(c, participateReq{RoundID: roundID, Amount: amount})
}

func (s *service) participateCore(c *gin.Context, req participateReq) {
	amt, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok || amt.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	user := c.GetString("user_id")

	tx, err := s.pg.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot start tx"})
		return
	}
	defer tx.Rollback(c)

	var soldStr, supplyStr, maxStr, priceStr string
	var startT, endT time.Time
	var status string
	err = tx.QueryRow(c, `SELECT sold::text,total_supply::text,max_per_user::text,price::text,start_time,end_time,status FROM ieo_rounds WHERE id=$1 FOR UPDATE`, req.RoundID).
		Scan(&soldStr, &supplyStr, &maxStr, &priceStr, &startT, &endT, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "round not found"})
		return
	}
	now := time.Now()
	if status != "active" && !(status == "upcoming" && now.After(startT)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "round not active"})
		return
	}
	if now.After(endT) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "round ended"})
		return
	}
	sold, _ := new(big.Int).SetString(soldStr, 10)
	supply, _ := new(big.Int).SetString(supplyStr, 10)
	max, _ := new(big.Int).SetString(maxStr, 0)
	if max == nil {
		max = big.NewInt(0)
	}
	price, _ := new(big.Float).SetString(priceStr)
	if price == nil {
		price = big.NewFloat(0)
	}
	if new(big.Int).Add(sold, amt).Cmp(supply) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exceeds remaining supply"})
		return
	}
	// check user existing participation
	var existingStr string
	err = tx.QueryRow(c, `SELECT amount::text FROM ieo_participations WHERE round_id=$1 AND user_id=$2`, req.RoundID, user).Scan(&existingStr)
	if err == nil {
		existing, _ := new(big.Int).SetString(existingStr, 10)
		if existing == nil {
			existing = big.NewInt(0)
		}
		if new(big.Int).Add(existing, amt).Cmp(max) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "exceeds user max allocation"})
			return
		}
		newAmt := new(big.Int).Add(existing, amt)
		paid := new(big.Float).Mul(new(big.Float).SetInt(newAmt), price)
		_, err = tx.Exec(c, `UPDATE ieo_participations SET amount=$1,paid=$2 WHERE round_id=$3 AND user_id=$4`, newAmt.String(), paid.Text('f', 18), req.RoundID, user)
	} else {
		if amt.Cmp(max) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "exceeds user max allocation"})
			return
		}
		paid := new(big.Float).Mul(new(big.Float).SetInt(amt), price)
		pid := newID()
		_, err = tx.Exec(c, `INSERT INTO ieo_participations (id,round_id,user_id,amount,paid) VALUES ($1,$2,$3,$4,$5)`, pid, req.RoundID, user, amt.String(), paid.Text('f', 18))
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot record participation"})
		return
	}
	newSold := new(big.Int).Add(sold, amt)
	_, err = tx.Exec(c, `UPDATE ieo_rounds SET sold=$1 WHERE id=$2`, newSold.String(), req.RoundID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update round"})
		return
	}
	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "amount": amt.String()})
}

func (s *service) listParticipations(c *gin.Context) {
	user := c.GetString("user_id")
	rows, err := s.pg.Query(c, `SELECT id,round_id,amount::text,paid::text,claimed,extract(epoch from created_at)::bigint FROM ieo_participations WHERE user_id=$1 ORDER BY created_at DESC`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var p struct{ ID, RID, Amt, Paid string; Claimed bool; Ts int64 }
		if err := rows.Scan(&p.ID, &p.RID, &p.Amt, &p.Paid, &p.Claimed, &p.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": p.ID, "round_id": p.RID, "amount": p.Amt, "paid": p.Paid, "claimed": p.Claimed, "created_at": p.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"participations": out, "count": len(out)})
}

type claimReq struct {
	RoundID string `json:"round_id" binding:"required"`
}

func (s *service) claim(c *gin.Context) {
	var req claimReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := c.GetString("user_id")
	var endT time.Time
	var rStatus string
	err := s.pg.QueryRow(c, `SELECT end_time,status FROM ieo_rounds WHERE id=$1`, req.RoundID).Scan(&endT, &rStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "round not found"})
		return
	}
	if time.Now().Before(endT) || rStatus != "ended" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "claim not available yet"})
		return
	}
	var claimed bool
	err = s.pg.QueryRow(c, `SELECT claimed FROM ieo_participations WHERE round_id=$1 AND user_id=$2`, req.RoundID, user).Scan(&claimed)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no participation"})
		return
	}
	if claimed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already claimed"})
		return
	}
	_, err = s.pg.Exec(c, `UPDATE ieo_participations SET claimed=TRUE WHERE round_id=$1 AND user_id=$2`, req.RoundID, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update claim"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "claimed": true})
}
