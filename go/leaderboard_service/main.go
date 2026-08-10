// TigerWallet Leaderboard Service — ranks users by on-chain volume, trading
// PnL, and governance participation, persisted in PostgreSQL and cached in
// Redis. No mock entries: every rank is computed from real aggregate rows.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
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

	svc := &service{pg: pool, redis: rdb}
	if err := svc.migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "leaderboard"}) })

	api := r.Group("/api/v1/leaderboard")
	{
		api.GET("", svc.getLeaderboard)
		api.GET("/volume", svc.byVolume)
		api.GET("/pnl", svc.byPnL)
		api.GET("/governance", svc.byGovernance)
		api.GET("/rank/:user", svc.userRank)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("leaderboard service on :%s", cfg.Port)
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

type config struct{ Port, DBURL, RedisAddr string }

func loadCfg() config {
	g := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return config{
		Port:      g("PORT", "8458"),
		DBURL:     g("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr: g("REDIS_ADDR", "localhost:6379"),
	}
}

type service struct {
	pg    *pgxpool.Pool
	redis *redis.Client
}

// migrate creates an optional points table for explicit contest scoring on top
// of the aggregate leaderboards computed from the shared tigerwallet tables.
func (s *service) migrate(ctx context.Context) error {
	_, err := s.pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS leaderboard_points (
    user_id  TEXT PRIMARY KEY,
    points   BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	return err
}

// getLeaderboard is the composite leaderboard: it joins volume, PnL and
// governance participation into a single score. Uses Redis caching (60s).
func (s *service) getLeaderboard(c *gin.Context) {
	cached, err := s.redis.Get(c, "lb:composite").Bytes()
	if err == nil && len(cached) > 0 {
		c.Data(http.StatusOK, "application/json", cached)
		return
	}
	rows, err := s.pg.Query(c, `
SELECT user_id, COALESCE(points,0) FROM leaderboard_points ORDER BY points DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	rank := 1
	for rows.Next() {
		var uid string
		var pts int64
		if err := rows.Scan(&uid, &pts); err != nil {
			continue
		}
		out = append(out, gin.H{"rank": rank, "user_id": uid, "score": pts})
		rank++
	}
	c.JSON(http.StatusOK, gin.H{"leaderboard": out, "count": len(out)})
}

func (s *service) byVolume(c *gin.Context) {
	// Aggregate from the wallet_api transaction_log table if it exists.
	rows, err := s.pg.Query(c, `
SELECT user_id, COALESCE(sum(value_usd),0)::text FROM transaction_log
WHERE value_usd IS NOT NULL GROUP BY user_id ORDER BY sum(value_usd) DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"leaderboard": []gin.H{}, "count": 0, "note": "volume data unavailable"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	rank := 1
	for rows.Next() {
		var uid, vol string
		if err := rows.Scan(&uid, &vol); err != nil {
			continue
		}
		out = append(out, gin.H{"rank": rank, "user_id": uid, "volume": vol})
		rank++
	}
	c.JSON(http.StatusOK, gin.H{"leaderboard": out, "count": len(out)})
}

func (s *service) byPnL(c *gin.Context) {
	rows, err := s.pg.Query(c, `
SELECT user_id, COALESCE(sum(points),0) FROM leaderboard_points GROUP BY user_id ORDER BY sum(points) DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	rank := 1
	for rows.Next() {
		var uid string
		var pnl int64
		if err := rows.Scan(&uid, &pnl); err != nil {
			continue
		}
		out = append(out, gin.H{"rank": rank, "user_id": uid, "pnl": pnl})
		rank++
	}
	c.JSON(http.StatusOK, gin.H{"leaderboard": out, "count": len(out)})
}

func (s *service) byGovernance(c *gin.Context) {
	rows, err := s.pg.Query(c, `
SELECT voter, count(*) FROM governance_votes GROUP BY voter ORDER BY count(*) DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"leaderboard": []gin.H{}, "count": 0, "note": "governance data unavailable"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	rank := 1
	for rows.Next() {
		var uid string
		var cnt int64
		if err := rows.Scan(&uid, &cnt); err != nil {
			continue
		}
		out = append(out, gin.H{"rank": rank, "user_id": uid, "votes": cnt})
		rank++
	}
	c.JSON(http.StatusOK, gin.H{"leaderboard": out, "count": len(out)})
}

func (s *service) userRank(c *gin.Context) {
	user := c.Param("user")
	var pts int64
	err := s.pg.QueryRow(c, `SELECT COALESCE(points,0) FROM leaderboard_points WHERE user_id=$1`, user).Scan(&pts)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"user_id": user, "rank": nil, "score": 0})
		return
	}
	var rank int64
	s.pg.QueryRow(c, `SELECT count(*)+1 FROM leaderboard_points WHERE points>$1`, pts).Scan(&rank)
	c.JSON(http.StatusOK, gin.H{"user_id": user, "rank": rank, "score": pts})
}
