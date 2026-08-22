// TigerWallet Governance Service — on-chain-style DAO governance with
// PostgreSQL persistence and Redis-cached vote tallies. No mocks: every
// proposal, vote and delegate is a real DB row.
package main

import (
	"context"
	"errors"
	"fmt"
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

type config struct {
	Port      string
	DBURL     string
	RedisAddr string
	JWTSecret string
}

func loadConfig() config {
	return config{
		Port:      envOr("PORT", "8454"),
		DBURL:     envOr("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr: envOr("REDIS_ADDR", "localhost:6379"),
		JWTSecret: envOr("JWT_SECRET", "tigerwallet-dev-secret-change-in-production"),
	}
}

func envOr(key, dflt string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return dflt
}

type service struct {
	pg    *pgxpool.Pool
	redis *redis.Client
	jwt   string
}

func main() {
	cfg := loadConfig()
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

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "governance"})
	})

	api := r.Group("/api/v1/governance")
	{
		api.GET("/proposals", svc.listProposals)
		api.GET("/proposals/:id", svc.getProposal)
		api.POST("/proposals", svc.auth(), svc.createProposal)
		api.POST("/proposals/:id/vote", svc.auth(), svc.castVote)
		api.GET("/proposals/:id/votes", svc.listVotes)
		api.POST("/delegate", svc.auth(), svc.delegate)
		api.GET("/delegates/:user", svc.getDelegate)
		api.GET("/voting-power/:user", svc.votingPower)
		api.POST("/proposals/:id/execute", svc.auth(), svc.executeProposal)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("governance service on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	srv.Shutdown(ctx2)
}

// auth validates an HS256 JWT issued by wallet_api and sets user_id.
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

// migrate creates the governance schema if absent.
func (s *service) migrate(ctx context.Context) error {
	_, err := s.pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS governance_proposals (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL,
    proposer      TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'active',
    for_votes     NUMERIC(78,0) NOT NULL DEFAULT 0,
    against_votes NUMERIC(78,0) NOT NULL DEFAULT 0,
    abstain_votes NUMERIC(78,0) NOT NULL DEFAULT 0,
    start_time    TIMESTAMPTZ NOT NULL DEFAULT now(),
    end_time      TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS governance_votes (
    id           TEXT PRIMARY KEY,
    proposal_id  TEXT NOT NULL REFERENCES governance_proposals(id) ON DELETE CASCADE,
    voter        TEXT NOT NULL,
    support      SMALLINT NOT NULL,
    weight       NUMERIC(78,0) NOT NULL DEFAULT 0,
    reason       TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (proposal_id, voter)
);
CREATE TABLE IF NOT EXISTS governance_delegates (
    delegator TEXT PRIMARY KEY,
    delegatee TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_votes_proposal ON governance_votes(proposal_id);
CREATE INDEX IF NOT EXISTS idx_proposals_status ON governance_proposals(status);
`)
	return err
}

// --- handlers ---

type createProposalReq struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	DurationH   int    `json:"duration_hours"`
}

func (s *service) createProposal(c *gin.Context) {
	if !s.enforceFeature(c, GatedFeature) {
		return
	}
	var req createProposalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dur := time.Duration(req.DurationH) * time.Hour
	if dur <= 0 {
		dur = 72 * time.Hour
	}
	id := newID()
	end := time.Now().Add(dur)
	_, err := s.pg.Exec(c,
		`INSERT INTO governance_proposals (id,title,description,proposer,end_time) VALUES ($1,$2,$3,$4,$5)`,
		id, req.Title, req.Description, c.GetString("user_id"), end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create proposal"})
		return
	}
	s.redis.Del(c, "gov:proposals:active")
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active", "end_time": end.Unix()})
}

func (s *service) listProposals(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	rows, err := s.pg.Query(c, `
SELECT id,title,description,proposer,status,for_votes,against_votes,abstain_votes,
       extract(epoch from end_time)::bigint, extract(epoch from created_at)::bigint
FROM governance_proposals
WHERE ($1='' OR status=$1)
ORDER BY created_at DESC LIMIT 100`, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type proposal struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		Proposer     string `json:"proposer"`
		Status       string `json:"status"`
		ForVotes     string `json:"for_votes"`
		AgainstVotes string `json:"against_votes"`
		AbstainVotes string `json:"abstain_votes"`
		EndTime      int64  `json:"end_time"`
		CreatedAt    int64  `json:"created_at"`
	}
	out := []proposal{}
	for rows.Next() {
		var p proposal
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Proposer, &p.Status,
			&p.ForVotes, &p.AgainstVotes, &p.AbstainVotes, &p.EndTime, &p.CreatedAt); err != nil {
			continue
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, gin.H{"proposals": out, "count": len(out)})
}

func (s *service) getProposal(c *gin.Context) {
	id := c.Param("id")
	var p struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		Proposer     string `json:"proposer"`
		Status       string `json:"status"`
		ForVotes     string `json:"for_votes"`
		AgainstVotes string `json:"against_votes"`
		AbstainVotes string `json:"abstain_votes"`
		EndTime      int64  `json:"end_time"`
		CreatedAt    int64  `json:"created_at"`
	}
	err := s.pg.QueryRow(c, `
SELECT id,title,description,proposer,status,for_votes,against_votes,abstain_votes,
       extract(epoch from end_time)::bigint, extract(epoch from created_at)::bigint
FROM governance_proposals WHERE id=$1`, id).
		Scan(&p.ID, &p.Title, &p.Description, &p.Proposer, &p.Status,
			&p.ForVotes, &p.AgainstVotes, &p.AbstainVotes, &p.EndTime, &p.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "proposal not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

type castVoteReq struct {
	ProposalID string `json:"proposal_id" binding:"required"`
	Support    int    `json:"support" binding:"required"` // 1 for, 0 against, 2 abstain
	Reason     string `json:"reason"`
	Weight     string `json:"weight"`
}

func (s *service) castVote(c *gin.Context) {
	if !s.enforceFeature(c, GatedFeature) {
		return
	}
	var req castVoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	voter := c.GetString("user_id")
	if req.Support < 0 || req.Support > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "support must be 0,1,2"})
		return
	}
	weight := req.Weight
	if weight == "" {
		weight = "1"
	}
	col := map[int]string{1: "for_votes", 0: "against_votes", 2: "abstain_votes"}[req.Support]

	tx, err := s.pg.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot start tx"})
		return
	}
	defer tx.Rollback(c)

	var status string
	var endTime time.Time
	err = tx.QueryRow(c, `SELECT status,end_time FROM governance_proposals WHERE id=$1 FOR UPDATE`, req.ProposalID).Scan(&status, &endTime)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "proposal not found"})
		return
	}
	if status != "active" || time.Now().After(endTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proposal not active"})
		return
	}

	vid := newID()
	_, err = tx.Exec(c,
		`INSERT INTO governance_votes (id,proposal_id,voter,support,weight,reason) VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (proposal_id,voter) DO NOTHING`,
		vid, req.ProposalID, voter, req.Support, weight, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot record vote"})
		return
	}
	_, err = tx.Exec(c, fmt.Sprintf(
		`UPDATE governance_proposals SET %s = %s + $1 WHERE id=$2`, col, col), weight, req.ProposalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot tally vote"})
		return
	}
	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	s.redis.Del(c, "gov:proposals:active")
	c.JSON(http.StatusOK, gin.H{"success": true, "vote_id": vid, "support": req.Support})
}

func (s *service) listVotes(c *gin.Context) {
	id := c.Param("id")
	rows, err := s.pg.Query(c,
		`SELECT id,voter,support,weight,reason,extract(epoch from created_at)::bigint FROM governance_votes WHERE proposal_id=$1 ORDER BY created_at DESC LIMIT 200`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	type vote struct {
		ID        string `json:"id"`
		Voter     string `json:"voter"`
		Support   int    `json:"support"`
		Weight    string `json:"weight"`
		Reason    string `json:"reason"`
		CreatedAt int64  `json:"created_at"`
	}
	out := []vote{}
	for rows.Next() {
		var v vote
		if err := rows.Scan(&v.ID, &v.Voter, &v.Support, &v.Weight, &v.Reason, &v.CreatedAt); err != nil {
			continue
		}
		out = append(out, v)
	}
	c.JSON(http.StatusOK, gin.H{"votes": out, "count": len(out)})
}

type delegateReq struct {
	Delegatee string `json:"delegatee" binding:"required"`
}

func (s *service) delegate(c *gin.Context) {
	if !s.enforceFeature(c, GatedFeature) {
		return
	}
	var req delegateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	delegator := c.GetString("user_id")
	if delegator == req.Delegatee {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delegate to self"})
		return
	}
	_, err := s.pg.Exec(c,
		`INSERT INTO governance_delegates (delegator,delegatee,updated_at) VALUES ($1,$2,now())
		 ON CONFLICT (delegator) DO UPDATE SET delegatee=$2,updated_at=now()`,
		delegator, req.Delegatee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot delegate"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "delegator": delegator, "delegatee": req.Delegatee})
}

func (s *service) getDelegate(c *gin.Context) {
	user := c.Param("user")
	var delegatee string
	err := s.pg.QueryRow(c, `SELECT delegatee FROM governance_delegates WHERE delegator=$1`, user).Scan(&delegatee)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"delegator": user, "delegatee": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"delegator": user, "delegatee": delegatee})
}

// votingPower returns the delegated voting power for a user: their own base
// unit plus the number of users who delegated to them.
func (s *service) votingPower(c *gin.Context) {
	user := c.Param("user")
	var count int
	s.pg.QueryRow(c, `SELECT count(*) FROM governance_delegates WHERE delegatee=$1`, user).Scan(&count)
	c.JSON(http.StatusOK, gin.H{"user": user, "delegated_count": count, "voting_power": count + 1})
}

// executeProposal finalizes a proposal whose voting period has ended: it sets
// the status to "succeeded" if for-votes exceed against-votes, otherwise
// "defeated". It refuses to execute still-active or already-finalized proposals.
func (s *service) executeProposal(c *gin.Context) {
	if !s.enforceFeature(c, GatedFeature) {
		return
	}
	id := c.Param("id")
	var status, forV, against string
	var endTime time.Time
	err := s.pg.QueryRow(c,
		`SELECT status,for_votes::text,against_votes::text,end_time FROM governance_proposals WHERE id=$1 FOR UPDATE`,
		id).Scan(&status, &forV, &against, &endTime)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "proposal not found"})
		return
	}
	if status == "succeeded" || status == "defeated" || status == "executed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proposal already finalized", "status": status})
		return
	}
	if time.Now().Before(endTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voting period not ended"})
		return
	}

	// Compare integer vote tallies (stored as NUMERIC).
	newStatus := "defeated"
	if cmpBigInt(forV, against) > 0 {
		newStatus = "succeeded"
	}
	_, err = s.pg.Exec(c, `UPDATE governance_proposals SET status=$1 WHERE id=$2`, newStatus, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot finalize"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id, "status": newStatus, "for_votes": forV, "against_votes": against})
}
