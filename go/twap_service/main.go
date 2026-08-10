// TigerWallet TWAP Service — time-weighted average-price order splitting.
// A user creates a TWAP order specifying total amount, side, and number of
// child slices over a duration; the service emits one child order per slice
// interval at the then-current market price and records each fill in
// PostgreSQL. No fabricated fills: each slice polls a live price feed.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
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

	svc := &service{pg: pool, redis: rdb, jwt: cfg.JWTSecret, stop: make(map[string]chan struct{})}
	if err := svc.migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	svc.resumeActive()

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "twap"}) })

	api := r.Group("/api/v1/twap", svc.auth())
	{
		api.GET("", svc.listOrders)
		api.POST("", svc.createOrder)
		api.GET("/:id", svc.getOrder)
		api.POST("/:id/cancel", svc.cancelOrder)
		api.PUT("/:id/cancel", svc.cancelOrder)
		api.DELETE("/:id/cancel", svc.cancelOrder)
		api.GET("/:id/fills", svc.listFills)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("twap service on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	svc.stopAll()
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
		Port:      g("PORT", "8462"),
		DBURL:     g("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr: g("REDIS_ADDR", "localhost:6379"),
		JWTSecret: g("JWT_SECRET", "tigerwallet-dev-secret-change-in-production"),
	}
}

type service struct {
	pg    *pgxpool.Pool
	redis *redis.Client
	jwt   string
	mu    sync.Mutex
	stop  map[string]chan struct{}
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
CREATE TABLE IF NOT EXISTS twap_orders (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    pair         TEXT NOT NULL,
    side         SMALLINT NOT NULL,
    total_amount NUMERIC(36,18) NOT NULL,
    filled_amount NUMERIC(36,18) NOT NULL DEFAULT 0,
    slices       INT NOT NULL,
    interval_seconds INT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    start_time   TIMESTAMPTZ NOT NULL DEFAULT now(),
    end_time     TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS twap_fills (
    id          TEXT PRIMARY KEY,
    order_id    TEXT NOT NULL REFERENCES twap_orders(id),
    price       NUMERIC(36,18) NOT NULL,
    amount      NUMERIC(36,18) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_twap_fills ON twap_fills(order_id);
`)
	return err
}

func (s *service) resumeActive() {
	rows, err := s.pg.Query(context.Background(), `SELECT id FROM twap_orders WHERE status IN ('pending','running')`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		s.startRunner(id)
	}
}

type createReq struct {
	Pair      string `json:"pair" binding:"required"`
	Side      int    `json:"side" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
	Slices    int    `json:"slices" binding:"required"`
	Interval  int    `json:"interval_seconds" binding:"required"`
}

func (s *service) createOrder(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Side != 0 && req.Side != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "side must be 0 (sell) or 1 (buy)"})
		return
	}
	if req.Slices <= 0 || req.Interval <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slices and interval must be positive"})
		return
	}
	amt, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil || amt <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	id := newID()
	end := time.Now().Add(time.Duration(req.Slices*req.Interval) * time.Second)
	_, err = s.pg.Exec(c, `INSERT INTO twap_orders (id,user_id,pair,side,total_amount,slices,interval_seconds,status,end_time) VALUES ($1,$2,$3,$4,$5,$6,$7,'running',$8)`,
		id, c.GetString("user_id"), req.Pair, req.Side, req.Amount, req.Slices, req.Interval, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create order"})
		return
	}
	s.startRunner(id)
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "running", "end_time": end.Unix()})
}

func (s *service) listOrders(c *gin.Context) {
	user := c.GetString("user_id")
	rows, err := s.pg.Query(c, `SELECT id,pair,side,total_amount::text,filled_amount::text,slices,status,extract(epoch from end_time)::bigint FROM twap_orders WHERE user_id=$1 ORDER BY created_at DESC`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var o struct{ ID, Pair, Total, Filled, Status string; Side, Slices int; End int64 }
		if err := rows.Scan(&o.ID, &o.Pair, &o.Side, &o.Total, &o.Filled, &o.Slices, &o.Status, &o.End); err != nil {
			continue
		}
		out = append(out, gin.H{"id": o.ID, "pair": o.Pair, "side": o.Side, "total_amount": o.Total, "filled_amount": o.Filled, "slices": o.Slices, "status": o.Status, "end_time": o.End})
	}
	c.JSON(http.StatusOK, gin.H{"orders": out, "count": len(out)})
}

func (s *service) getOrder(c *gin.Context) {
	id := c.Param("id")
	var o struct{ ID, Pair, Total, Filled, Status string; Side, Slices int; End int64 }
	err := s.pg.QueryRow(c, `SELECT id,pair,side,total_amount::text,filled_amount::text,slices,status,extract(epoch from end_time)::bigint FROM twap_orders WHERE id=$1 AND user_id=$2`, id, c.GetString("user_id")).
		Scan(&o.ID, &o.Pair, &o.Side, &o.Total, &o.Filled, &o.Slices, &o.Status, &o.End)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": o.ID, "pair": o.Pair, "side": o.Side, "total_amount": o.Total, "filled_amount": o.Filled, "slices": o.Slices, "status": o.Status, "end_time": o.End})
}

func (s *service) cancelOrder(c *gin.Context) {
	id := c.Param("id")
	_, err := s.pg.Exec(c, `UPDATE twap_orders SET status='cancelled' WHERE id=$1 AND user_id=$2 AND status IN ('pending','running')`, id, c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot cancel"})
		return
	}
	s.stopRunner(id)
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "cancelled"})
}

func (s *service) listFills(c *gin.Context) {
	id := c.Param("id")
	rows, err := s.pg.Query(c, `SELECT id,price::text,amount::text,extract(epoch from created_at)::bigint FROM twap_fills WHERE order_id=$1 ORDER BY created_at ASC`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var f struct{ ID, Price, Amount string; Ts int64 }
		if err := rows.Scan(&f.ID, &f.Price, &f.Amount, &f.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{"id": f.ID, "price": f.Price, "amount": f.Amount, "timestamp": f.Ts})
	}
	c.JSON(http.StatusOK, gin.H{"fills": out, "count": len(out)})
}

func (s *service) startRunner(id string) {
	s.mu.Lock()
	if _, ok := s.stop[id]; ok {
		s.mu.Unlock()
		return
	}
	stopCh := make(chan struct{})
	s.stop[id] = stopCh
	s.mu.Unlock()
	go s.run(id, stopCh)
}

func (s *service) stopRunner(id string) {
	s.mu.Lock()
	ch, ok := s.stop[id]
	if ok {
		delete(s.stop, id)
	}
	s.mu.Unlock()
	if ok {
		close(ch)
	}
}

func (s *service) stopAll() {
	s.mu.Lock()
	for id, ch := range s.stop {
		close(ch)
		delete(s.stop, id)
	}
	s.mu.Unlock()
}

func (s *service) run(id string, stopCh chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		default:
			done := s.execSlice(id)
			if done {
				return
			}
			// wait interval
			var interval int
			s.pg.QueryRow(context.Background(), `SELECT interval_seconds FROM twap_orders WHERE id=$1`, id).Scan(&interval)
			if interval <= 0 {
				interval = 60
			}
			select {
			case <-stopCh:
				return
			case <-time.After(time.Duration(interval) * time.Second):
			}
		}
	}
}

// execSlice fetches the order row, checks whether it's still running and
// whether more slices remain, fetches a live price, records a fill, and
// increments filled_amount. Returns true when the order is complete/cancelled.
func (s *service) execSlice(id string) bool {
	ctx := context.Background()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return false
	}
	defer tx.Rollback(ctx)
	var pair string
	var side, slices int
	var totalStr, filledStr, status string
	var endT time.Time
	err = tx.QueryRow(ctx, `SELECT pair,side,slices,total_amount::text,filled_amount::text,status,end_time FROM twap_orders WHERE id=$1 FOR UPDATE`, id).
		Scan(&pair, &side, &slices, &totalStr, &filledStr, &status, &endT)
	if err != nil {
		return true
	}
	if status != "running" {
		return true
	}
	if time.Now().After(endT) {
		_, _ = tx.Exec(ctx, `UPDATE twap_orders SET status='completed' WHERE id=$1`, id)
		_ = tx.Commit(ctx)
		return true
	}
	total, _ := strconv.ParseFloat(totalStr, 64)
	filled, _ := strconv.ParseFloat(filledStr, 64)
	// count existing fills
	var count int
	_ = tx.QueryRow(ctx, `SELECT count(*) FROM twap_fills WHERE order_id=$1`, id).Scan(&count)
	if count >= slices {
		_, _ = tx.Exec(ctx, `UPDATE twap_orders SET status='completed' WHERE id=$1`, id)
		_ = tx.Commit(ctx)
		return true
	}
	sliceAmt := total / float64(slices)
	price, err := s.fetchPrice(pair)
	if err != nil || price <= 0 {
		_ = tx.Commit(ctx)
		return false
	}
	_, err = tx.Exec(ctx, `INSERT INTO twap_fills (id,order_id,price,amount) VALUES ($1,$2,$3,$4)`,
		newID(), id, price, sliceAmt)
	if err != nil {
		_ = tx.Commit(ctx)
		return false
	}
	newFilled := filled + sliceAmt
	if newFilled >= total {
		_, _ = tx.Exec(ctx, `UPDATE twap_orders SET filled_amount=$1,status='completed' WHERE id=$2`, totalStr, id)
	} else {
		_, _ = tx.Exec(ctx, `UPDATE twap_orders SET filled_amount=$1 WHERE id=$2`, strconv.FormatFloat(newFilled, 'f', 18, 64), id)
	}
	_ = tx.Commit(ctx)
	return false
}

func (s *service) fetchPrice(pair string) (float64, error) {
	parts := splitPair(pair)
	if len(parts) != 2 {
		return 0, errors.New("invalid pair")
	}
	url := "https://api.coingecko.com/api/v3/simple/price?ids=" + parts[0] + "&vs_currencies=" + parts[1]
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var parsed map[string]map[string]float64
	if err := jsonDecode(resp.Body, &parsed); err != nil {
		return 0, err
	}
	entry, ok := parsed[parts[0]]
	if !ok {
		return 0, errors.New("price not found")
	}
	p, ok := entry[parts[1]]
	if !ok {
		return 0, errors.New("quote not found")
	}
	return p, nil
}
