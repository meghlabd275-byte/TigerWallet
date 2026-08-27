// TigerWallet Launchpad Service — token IDO/ICO launchpad with PostgreSQL
// persistence. Admins create projects (token, allocation, caps, schedule);
// users participate within caps and claim tokens after a project finalizes.
// No seed/mock data: every project & allocation is a real DB row.
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

	svc := &service{pg: pool, redis: rdb, jwt: cfg.JWTSecret, adminID: cfg.AdminID}
	if err := svc.migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "launchpad"}) })

	api := r.Group("/api/v1/launchpad")
	{
		api.GET("/projects", svc.listProjects)
		api.POST("/projects", svc.auth(), svc.createProject)
		api.GET("/projects/:id", svc.getProject)
		api.POST("/participate", svc.auth(), svc.participate)
		api.POST("/allocations/:id/claim", svc.auth(), svc.claim)
		api.GET("/allocations", svc.auth(), svc.listAllocations)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("launchpad service on :%s", cfg.Port)
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
		Port:      g("PORT", "8012"),
		DBURL:     g("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr: g("REDIS_ADDR", "localhost:6379"),
		JWTSecret: g("JWT_SECRET", ""),
		AdminID:   g("ADMIN_USER_ID", ""),
	}
}

type service struct {
	pg      *pgxpool.Pool
	redis   *redis.Client
	jwt     string
	adminID string
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
CREATE TABLE IF NOT EXISTS launchpad_projects (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    symbol        TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    chain         TEXT NOT NULL DEFAULT 'ethereum',
    token_address TEXT NOT NULL DEFAULT '',
    total_supply  NUMERIC(78,0) NOT NULL,
    ido_allocation NUMERIC(78,0) NOT NULL,
    price         NUMERIC(36,18) NOT NULL,
    soft_cap      NUMERIC(78,0) NOT NULL DEFAULT 0,
    hard_cap      NUMERIC(78,0) NOT NULL DEFAULT 0,
    max_per_user  NUMERIC(78,0) NOT NULL DEFAULT 0,
    raised_amount NUMERIC(78,0) NOT NULL DEFAULT 0,
    participants  INT NOT NULL DEFAULT 0,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    status        TEXT NOT NULL DEFAULT 'upcoming',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS launchpad_allocations (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES launchpad_projects(id),
    user_id     TEXT NOT NULL,
    amount      NUMERIC(78,0) NOT NULL,
    tokens      NUMERIC(78,0) NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    claimed_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_lp_alloc_user ON launchpad_allocations(user_id);
`)
	return err
}

func (s *service) listProjects(c *gin.Context) {
	status := c.Query("status")
	q := `SELECT id,name,symbol,description,chain,token_address,total_supply::text,ido_allocation::text,price::text,soft_cap::text,hard_cap::text,max_per_user::text,raised_amount::text,participants,extract(epoch from start_time)::bigint,extract(epoch from end_time)::bigint,status FROM launchpad_projects`
	args := []interface{}{}
	if status != "" {
		q += ` WHERE status=$1`
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
			ID, Name, Sym, Desc, Chain, Addr, Supply, Alloc, Price, Soft, Hard, Max, Raised, Status string
			Parts                                                                                   int
			Start, End                                                                              int64
		}
		if err := rows.Scan(&p.ID, &p.Name, &p.Sym, &p.Desc, &p.Chain, &p.Addr, &p.Supply, &p.Alloc, &p.Price, &p.Soft, &p.Hard, &p.Max, &p.Raised, &p.Parts, &p.Start, &p.End, &p.Status); err != nil {
			continue
		}
		out = append(out, gin.H{
			"id": p.ID, "name": p.Name, "symbol": p.Sym, "description": p.Desc, "chain": p.Chain,
			"token_address": p.Addr, "total_supply": p.Supply, "ido_allocation": p.Alloc, "price": p.Price,
			"soft_cap": p.Soft, "hard_cap": p.Hard, "max_per_user": p.Max, "raised_amount": p.Raised,
			"participants": p.Parts, "start_time": p.Start, "end_time": p.End, "status": p.Status,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "projects": out, "count": len(out)})
}

func (s *service) getProject(c *gin.Context) {
	id := c.Param("id")
	var p struct {
		ID, Name, Sym, Desc, Chain, Addr, Supply, Alloc, Price, Soft, Hard, Max, Raised, Status string
		Parts                                                                                   int
		Start, End                                                                              int64
	}
	err := s.pg.QueryRow(c, `SELECT id,name,symbol,description,chain,token_address,total_supply::text,ido_allocation::text,price::text,soft_cap::text,hard_cap::text,max_per_user::text,raised_amount::text,participants,extract(epoch from start_time)::bigint,extract(epoch from end_time)::bigint,status FROM launchpad_projects WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.Sym, &p.Desc, &p.Chain, &p.Addr, &p.Supply, &p.Alloc, &p.Price, &p.Soft, &p.Hard, &p.Max, &p.Raised, &p.Parts, &p.Start, &p.End, &p.Status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "project": gin.H{
		"id": p.ID, "name": p.Name, "symbol": p.Sym, "description": p.Desc, "chain": p.Chain,
		"token_address": p.Addr, "total_supply": p.Supply, "ido_allocation": p.Alloc, "price": p.Price,
		"soft_cap": p.Soft, "hard_cap": p.Hard, "max_per_user": p.Max, "raised_amount": p.Raised,
		"participants": p.Parts, "start_time": p.Start, "end_time": p.End, "status": p.Status,
	}})
}

type createProjectReq struct {
	Name          string `json:"name" binding:"required"`
	Symbol        string `json:"symbol" binding:"required"`
	Description   string `json:"description"`
	Chain         string `json:"chain"`
	TokenAddress  string `json:"token_address"`
	TotalSupply   string `json:"total_supply" binding:"required"`
	IDOAllocation string `json:"ido_allocation" binding:"required"`
	Price         string `json:"price" binding:"required"`
	SoftCap       string `json:"soft_cap"`
	HardCap       string `json:"hard_cap"`
	MaxPerUser    string `json:"max_per_user"`
	StartEpoch    int64  `json:"start_time" binding:"required"`
	EndEpoch      int64  `json:"end_time" binding:"required"`
}

func (s *service) createProject(c *gin.Context) {
	user := c.GetString("user_id")
	if s.adminID != "" && user != s.adminID {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}
	var req createProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Chain == "" {
		req.Chain = "ethereum"
	}
	if req.SoftCap == "" {
		req.SoftCap = "0"
	}
	if req.HardCap == "" {
		req.HardCap = "0"
	}
	if req.MaxPerUser == "" {
		req.MaxPerUser = "0"
	}
	id := newID()
	_, err := s.pg.Exec(c, `INSERT INTO launchpad_projects (id,name,symbol,description,chain,token_address,total_supply,ido_allocation,price,soft_cap,hard_cap,max_per_user,start_time,end_time,status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'upcoming')`,
		id, req.Name, req.Symbol, req.Description, req.Chain, req.TokenAddress, req.TotalSupply, req.IDOAllocation, req.Price, req.SoftCap, req.HardCap, req.MaxPerUser, time.Unix(req.StartEpoch, 0), time.Unix(req.EndEpoch, 0))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create project"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "project": gin.H{"id": id, "status": "upcoming"}})
}

type participateReq struct {
	ProjectID string `json:"project_id" binding:"required"`
	Amount    string `json:"amount" binding:"required"` // USDC/base units committed
}

func (s *service) participate(c *gin.Context) {
	var req participateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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

	var priceStr, allocStr, raisedStr, maxStr, hardStr, supplyStr string
	var startT, endT time.Time
	var status string
	err = tx.QueryRow(c, `SELECT price::text,ido_allocation::text,raised_amount::text,max_per_user::text,hard_cap::text,total_supply::text,start_time,end_time,status FROM launchpad_projects WHERE id=$1 FOR UPDATE`, req.ProjectID).
		Scan(&priceStr, &allocStr, &raisedStr, &maxStr, &hardStr, &supplyStr, &startT, &endT, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	now := time.Now()
	if status != "active" && !(status == "upcoming" && now.After(startT)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project not active"})
		return
	}
	if now.After(endT) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ended"})
		return
	}
	price, _ := new(big.Float).SetString(priceStr)
	if price == nil {
		price = big.NewFloat(0)
	}
	if price.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project price"})
		return
	}
	// tokens = amount / price (amount in base units, price per token)
	amtF := new(big.Float).SetInt(amt)
	tokensF := new(big.Float).Quo(amtF, price)
	tokens, _ := tokensF.Int(nil)
	if tokens.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount too small"})
		return
	}
	raised, _ := new(big.Int).SetString(raisedStr, 10)
	if raised == nil {
		raised = big.NewInt(0)
	}
	hard, _ := new(big.Int).SetString(hardStr, 10)
	if hard != nil && hard.Sign() > 0 && new(big.Int).Add(raised, amt).Cmp(hard) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exceeds hard cap"})
		return
	}
	max, _ := new(big.Int).SetString(maxStr, 10)

	var existingAmt string
	err = tx.QueryRow(c, `SELECT amount::text FROM launchpad_allocations WHERE project_id=$1 AND user_id=$2`, req.ProjectID, user).Scan(&existingAmt)
	if err == nil {
		existing, _ := new(big.Int).SetString(existingAmt, 10)
		if existing == nil {
			existing = big.NewInt(0)
		}
		if max != nil && max.Sign() > 0 && new(big.Int).Add(existing, amt).Cmp(max) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "exceeds user max allocation"})
			return
		}
		newAmt := new(big.Int).Add(existing, amt)
		_, err = tx.Exec(c, `UPDATE launchpad_allocations SET amount=$1,tokens=$2 WHERE project_id=$3 AND user_id=$4`, newAmt.String(), tokens.String(), req.ProjectID, user)
	} else {
		if max != nil && max.Sign() > 0 && amt.Cmp(max) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "exceeds user max allocation"})
			return
		}
		aid := newID()
		_, err = tx.Exec(c, `INSERT INTO launchpad_allocations (id,project_id,user_id,amount,tokens) VALUES ($1,$2,$3,$4,$5)`, aid, req.ProjectID, user, amt.String(), tokens.String())
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot record allocation"})
		return
	}
	newRaised := new(big.Int).Add(raised, amt)
	_, err = tx.Exec(c, `UPDATE launchpad_projects SET raised_amount=$1,participants=participants+1 WHERE id=$2`, newRaised.String(), req.ProjectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update project"})
		return
	}
	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "amount": amt.String(), "tokens": tokens.String()})
}

func (s *service) claim(c *gin.Context) {
	id := c.Param("id")
	user := c.GetString("user_id")
	var projectID, status string
	var endT time.Time
	var pStatus string
	err := s.pg.QueryRow(c, `SELECT a.project_id,a.status,p.end_time,p.status FROM launchpad_allocations a JOIN launchpad_projects p ON a.project_id=p.id WHERE a.id=$1 AND a.user_id=$2`, id, user).
		Scan(&projectID, &status, &endT, &pStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "allocation not found"})
		return
	}
	if status == "claimed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already claimed"})
		return
	}
	if time.Now().Before(endT) || pStatus != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "claim not available yet"})
		return
	}
	_, err = s.pg.Exec(c, `UPDATE launchpad_allocations SET status='claimed',claimed_at=now() WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update allocation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "allocation_id": id, "status": "claimed"})
}

func (s *service) listAllocations(c *gin.Context) {
	user := c.GetString("user_id")
	rows, err := s.pg.Query(c, `SELECT id,project_id,amount::text,tokens::text,status,extract(epoch from created_at)::bigint FROM launchpad_allocations WHERE user_id=$1 ORDER BY created_at DESC`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var a struct {
			ID, PID, Amt, Tokens, Status string
			Ts                           int64
		}
		if err := rows.Scan(&a.ID, &a.PID, &a.Amt, &a.Tokens, &a.Status, &a.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": a.ID, "project_id": a.PID, "amount": a.Amt, "tokens": a.Tokens, "status": a.Status, "created_at": a.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "allocations": out, "count": len(out)})
}
