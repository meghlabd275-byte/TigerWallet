// TigerWallet Insurance Service — DeFi smart-contract insurance products,
// positions (active coverage), and claim filing with PostgreSQL persistence.
// No mock products or claims: every record is a real DB row.
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
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "insurance"}) })

	api := r.Group("/api/v1/insurance")
	{
		api.GET("/products", svc.listProducts)
		api.POST("/products", svc.authAdmin(), svc.createProduct)
		api.POST("/positions", svc.auth(), svc.buyCoverage)
		api.GET("/positions", svc.auth(), svc.listPositions)
		api.POST("/claims", svc.auth(), svc.fileClaim)
		api.GET("/claims", svc.auth(), svc.listClaims)
		api.POST("/claims/:id/process", svc.authAdmin(), svc.processClaim)
		api.GET("/stats", svc.stats)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("insurance service on :%s", cfg.Port)
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
		Port:      g("PORT", "8459"),
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

func (s *service) authAdmin() gin.HandlerFunc {
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
CREATE TABLE IF NOT EXISTS insurance_products (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    protocol     TEXT NOT NULL,
    coverage_max NUMERIC(78,0) NOT NULL,
    premium_rate NUMERIC(10,6) NOT NULL,
    duration_days INT NOT NULL DEFAULT 30,
    status       TEXT NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS insurance_positions (
    id           TEXT PRIMARY KEY,
    product_id   TEXT NOT NULL REFERENCES insurance_products(id),
    user_id      TEXT NOT NULL,
    coverage     NUMERIC(78,0) NOT NULL,
    premium      NUMERIC(78,0) NOT NULL,
    status       TEXT NOT NULL DEFAULT 'active',
    start_time   TIMESTAMPTZ NOT NULL DEFAULT now(),
    end_time     TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS insurance_claims (
    id           TEXT PRIMARY KEY,
    position_id   TEXT NOT NULL REFERENCES insurance_positions(id),
    user_id      TEXT NOT NULL,
    amount       NUMERIC(78,0) NOT NULL,
    reason       TEXT NOT NULL,
    evidence     TEXT,
    status       TEXT NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ins_positions_user ON insurance_positions(user_id);
CREATE INDEX IF NOT EXISTS idx_ins_claims_user ON insurance_claims(user_id);
`)
	return err
}

func (s *service) listProducts(c *gin.Context) {
	rows, err := s.pg.Query(c, `SELECT id,name,protocol,coverage_max::text,premium_rate::text,duration_days,status FROM insurance_products WHERE status='active' ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var p struct {
			ID, Name, Proto, Cov, Rate, Status string
			Dur                                int
		}
		if err := rows.Scan(&p.ID, &p.Name, &p.Proto, &p.Cov, &p.Rate, &p.Dur, &p.Status); err != nil {
			continue
		}
		out = append(out, gin.H{"id": p.ID, "name": p.Name, "protocol": p.Proto, "coverage_max": p.Cov, "premium_rate": p.Rate, "duration_days": p.Dur, "status": p.Status})
	}
	c.JSON(http.StatusOK, gin.H{"products": out, "count": len(out)})
}

type createProductReq struct {
	Name         string `json:"name" binding:"required"`
	Protocol     string `json:"protocol" binding:"required"`
	CoverageMax  string `json:"coverage_max" binding:"required"`
	PremiumRate  string `json:"premium_rate" binding:"required"`
	DurationDays int    `json:"duration_days"`
}

func (s *service) createProduct(c *gin.Context) {
	var req createProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dur := req.DurationDays
	if dur <= 0 {
		dur = 30
	}
	id := newID()
	_, err := s.pg.Exec(c, `INSERT INTO insurance_products (id,name,protocol,coverage_max,premium_rate,duration_days) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, req.Name, req.Protocol, req.CoverageMax, req.PremiumRate, dur)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create product"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active"})
}

type buyReq struct {
	ProductID string `json:"product_id" binding:"required"`
	Coverage  string `json:"coverage" binding:"required"`
}

func (s *service) buyCoverage(c *gin.Context) {
	var req buyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var covMax, rateStr string
	var dur int
	err := s.pg.QueryRow(c, `SELECT coverage_max::text,premium_rate::text,duration_days FROM insurance_products WHERE id=$1 AND status='active'`, req.ProductID).Scan(&covMax, &rateStr, &dur)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	// premium = coverage * rate (string big-int math via float)
	var covF, rateF float64
	fmtSscan(req.Coverage, &covF)
	fmtSscan(rateStr, &rateF)
	if covF <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coverage"})
		return
	}
	premium := covF * rateF
	if premium < 0 {
		premium = 0
	}
	end := time.Now().Add(time.Duration(dur) * 24 * time.Hour)
	pid := newID()
	_, err = s.pg.Exec(c, `INSERT INTO insurance_positions (id,product_id,user_id,coverage,premium,end_time) VALUES ($1,$2,$3,$4,$5,$6)`,
		pid, req.ProductID, c.GetString("user_id"), req.Coverage, premiumStr(premium), end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create position"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": pid, "premium": premiumStr(premium), "end_time": end.Unix()})
}

func (s *service) listPositions(c *gin.Context) {
	user := c.GetString("user_id")
	rows, err := s.pg.Query(c, `SELECT id,product_id,coverage::text,premium::text,status,extract(epoch from end_time)::bigint FROM insurance_positions WHERE user_id=$1 ORDER BY created_at DESC`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var p struct {
			ID, PID, Cov, Prem, Status string
			End                        int64
		}
		if err := rows.Scan(&p.ID, &p.PID, &p.Cov, &p.Prem, &p.Status, &p.End); err != nil {
			continue
		}
		out = append(out, gin.H{"id": p.ID, "product_id": p.PID, "coverage": p.Cov, "premium": p.Prem, "status": p.Status, "end_time": p.End})
	}
	c.JSON(http.StatusOK, gin.H{"positions": out, "count": len(out)})
}

type claimReq struct {
	PositionID string `json:"position_id" binding:"required"`
	Amount     string `json:"amount" binding:"required"`
	Reason     string `json:"reason" binding:"required"`
	Evidence   string `json:"evidence"`
}

func (s *service) fileClaim(c *gin.Context) {
	var req claimReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := c.GetString("user_id")
	var status string
	err := s.pg.QueryRow(c, `SELECT status FROM insurance_positions WHERE id=$1 AND user_id=$2`, req.PositionID, user).Scan(&status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	if status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "position not active"})
		return
	}
	id := newID()
	_, err = s.pg.Exec(c, `INSERT INTO insurance_claims (id,position_id,user_id,amount,reason,evidence) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, req.PositionID, user, req.Amount, req.Reason, req.Evidence)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot file claim"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "pending"})
}

func (s *service) listClaims(c *gin.Context) {
	user := c.GetString("user_id")
	rows, err := s.pg.Query(c, `SELECT id,position_id,amount::text,reason,status,extract(epoch from created_at)::bigint FROM insurance_claims WHERE user_id=$1 ORDER BY created_at DESC`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var cl struct {
			ID, PID, Amt, Reason, Status string
			Ts                           int64
		}
		if err := rows.Scan(&cl.ID, &cl.PID, &cl.Amt, &cl.Reason, &cl.Status, &cl.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": cl.ID, "position_id": cl.PID, "amount": cl.Amt, "reason": cl.Reason, "status": cl.Status, "created_at": cl.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"claims": out, "count": len(out)})
}

type processClaimReq struct {
	Status string `json:"status" binding:"required"` // approved | rejected
}

func (s *service) processClaim(c *gin.Context) {
	id := c.Param("id")
	var req processClaimReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status != "approved" && req.Status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be approved or rejected"})
		return
	}
	_, err := s.pg.Exec(c, `UPDATE insurance_claims SET status=$1 WHERE id=$2`, req.Status, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update claim"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id, "status": req.Status})
}

func (s *service) stats(c *gin.Context) {
	var totalProducts, totalPositions, pendingClaims int64
	s.pg.QueryRow(c, `SELECT count(*) FROM insurance_products WHERE status='active'`).Scan(&totalProducts)
	s.pg.QueryRow(c, `SELECT count(*) FROM insurance_positions WHERE status='active'`).Scan(&totalPositions)
	s.pg.QueryRow(c, `SELECT count(*) FROM insurance_claims WHERE status='pending'`).Scan(&pendingClaims)
	c.JSON(http.StatusOK, gin.H{"total_products": totalProducts, "active_positions": totalPositions, "pending_claims": pendingClaims})
}
