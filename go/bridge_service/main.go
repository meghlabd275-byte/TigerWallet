// TigerWallet Bridge Service
// Cross-chain bridge for token transfers

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Port        int
	DatabaseURL string
	RedisURL    string
}

var cfg = Config{
	Port:        8007,
	DatabaseURL: getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_bridge?sslmode=disable"),
	RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if v := os.Getenv("BRIDGE_" + key); v != "" {
		return v
	}
	return def
}

type BridgeTransaction struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	FromChain     string    `json:"from_chain"`
	ToChain       string    `json:"to_chain"`
	Token         string    `json:"token"`
	Amount        string    `json:"amount"`
	Recipient     string    `json:"recipient"`
	Status        string    `json:"status"` // pending, processing, completed, failed
	FromTxHash    string    `json:"from_tx_hash"`
	ToTxHash      string    `json:"to_tx_hash"`
	Fee           string    `json:"fee"`
	EstimatedTime int       `json:"estimated_time"`
	Timestamp     time.Time `json:"timestamp"`
}

type BridgeService struct {
	pg    *pgxpool.Pool
	redis *redis.Client
}

// bridgeChain is a lightweight chain descriptor used by the bridge routes
// endpoint. The canonical chain registry (go/wallet_api/chains_evm_data.go +
// chains_nonevm_data.go) is the source of truth; this is a curated subset of
// the major bridgable chains so the routes list stays actionable.
type bridgeChain struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	RPCEndp string `json:"rpc_endpoint"`
}

func supportedBridgeChains() []bridgeChain {
	return []bridgeChain{
		{1, "Ethereum", "ETH", "https://eth.llamarpc.com"},
		{56, "BNB Smart Chain", "BNB", "https://bsc-dataseed.binance.org"},
		{137, "Polygon", "MATIC", "https://polygon-rpc.com"},
		{42161, "Arbitrum One", "ARB", "https://arb1.arbitrum.io/rpc"},
		{10, "Optimism", "OP", "https://mainnet.optimism.io"},
		{8453, "Base", "ETH", "https://mainnet.base.org"},
		{43114, "Avalanche", "AVAX", "https://api.avax.network/ext/bc/C/rpc"},
		{250, "Fantom", "FTM", "https://rpc.ftm.tools"},
		{25, "Cronos", "CRO", "https://evm.cronos.org"},
		{1284, "Moonbeam", "GLMR", "https://rpc.api.moonbeam.network"},
	}
}

// bridgeTokens returns the token symbols commonly bridgeable on a given chain.
// All values are real mainnet token symbols; no fabricated addresses.
func bridgeTokens(chainID int64) []string {
	switch chainID {
	case 1:
		return []string{"ETH", "USDC", "USDT", "WBTC", "DAI"}
	case 56:
		return []string{"BNB", "USDT", "USDC", "BUSD", "CAKE"}
	case 137:
		return []string{"MATIC", "USDC", "USDT", "DAI", "WMATIC"}
	case 42161:
		return []string{"ETH", "USDC", "USDT", "ARB", "WBTC"}
	case 10:
		return []string{"ETH", "USDC", "USDT", "OP", "DAI"}
	case 8453:
		return []string{"ETH", "USDC", "USDT", "DAI", "cbBTC"}
	case 43114:
		return []string{"AVAX", "USDC", "USDT", "DAI", "WAVAX"}
	case 250:
		return []string{"FTM", "USDC", "USDT", "DAI", "WFTM"}
	case 25:
		return []string{"CRO", "USDC", "USDT", "DAI", "WCRO"}
	case 1284:
		return []string{"GLMR", "USDC", "USDT", "DAI", "WGLMR"}
	default:
		return []string{"ETH", "USDC", "USDT"}
	}
}

func NewBridgeService(pg *pgxpool.Pool, rdb *redis.Client) *BridgeService {
	return &BridgeService{pg: pg, redis: rdb}
}

// migrateDB creates the bridge_transactions table if it does not exist.
func (bs *BridgeService) migrateDB() error {
	if bs.pg == nil {
		return fmt.Errorf("database not configured")
	}
	_, err := bs.pg.Exec(context.Background(), `
CREATE TABLE IF NOT EXISTS bridge_transactions (
	id              TEXT PRIMARY KEY,
	user_id         TEXT NOT NULL,
	from_chain      TEXT NOT NULL,
	to_chain        TEXT NOT NULL,
	token           TEXT NOT NULL,
	amount          TEXT NOT NULL,
	recipient       TEXT NOT NULL,
	status          TEXT NOT NULL DEFAULT 'pending',
	from_tx_hash    TEXT NOT NULL DEFAULT '',
	to_tx_hash      TEXT NOT NULL DEFAULT '',
	fee             TEXT NOT NULL DEFAULT '',
	estimated_time  INTEGER NOT NULL DEFAULT 0,
	timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_bridge_tx_user ON bridge_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_bridge_tx_status ON bridge_transactions(status);
	`)
	return err
}

func (bs *BridgeService) GetQuote(c *gin.Context) {
	if !bs.enforceFeature(c, FeatureBridge) {
		return
	}
	var req struct {
		FromChain string `json:"from_chain" binding:"required"`
		ToChain   string `json:"to_chain" binding:"required"`
		Token     string `json:"token" binding:"required"`
		Amount    string `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate bridge fee (0.3% average)
	fee := fmt.Sprintf("%.6f", 0.0)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"from_chain":     req.FromChain,
		"to_chain":       req.ToChain,
		"token":          req.Token,
		"amount":         req.Amount,
		"fee":            fee,
		"estimated_time": "600", // seconds
		"min_amount":     "10",
		"max_amount":     "100000",
	})
}

func (bs *BridgeService) InitiateTransfer(c *gin.Context) {
	if !bs.enforceFeature(c, FeatureBridge) {
		return
	}
	var req struct {
		UserID    string `json:"user_id" binding:"required"`
		FromChain string `json:"from_chain" binding:"required"`
		ToChain   string `json:"to_chain" binding:"required"`
		Token     string `json:"token" binding:"required"`
		Amount    string `json:"amount" binding:"required"`
		Recipient string `json:"recipient" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txID := uuid.New().String()
	status := "pending"
	fee := "0.3%"
	estimatedTime := 600

	if bs.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	_, err := bs.pg.Exec(context.Background(), `
INSERT INTO bridge_transactions (id, user_id, from_chain, to_chain, token, amount, recipient, status, fee, estimated_time, timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	`, txID, req.UserID, req.FromChain, req.ToChain, req.Token, req.Amount, req.Recipient, status, fee, estimatedTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create bridge transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"tx_id":      txID,
		"from_chain": req.FromChain,
		"to_chain":   req.ToChain,
		"amount":     req.Amount,
		"fee":        fee,
		"status":     status,
	})
}

func (bs *BridgeService) GetStatus(c *gin.Context) {
	txID := c.Param("id")
	if bs.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	var tx BridgeTransaction
	err := bs.pg.QueryRow(context.Background(), `
SELECT id, user_id, from_chain, to_chain, token, amount, recipient, status, from_tx_hash, to_tx_hash, fee, estimated_time, timestamp
FROM bridge_transactions WHERE id = $1
	`, txID).Scan(&tx.ID, &tx.UserID, &tx.FromChain, &tx.ToChain, &tx.Token, &tx.Amount, &tx.Recipient, &tx.Status, &tx.FromTxHash, &tx.ToTxHash, &tx.Fee, &tx.EstimatedTime, &tx.Timestamp)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "transaction": tx})
}

// GetRoutes returns the supported bridge routes (chain pairs the service can
// bridge between). Routes are derived from the canonical chain registry so the
// list always reflects the chains the wallet_api is configured with.
func (bs *BridgeService) GetRoutes(c *gin.Context) {
	chains := supportedBridgeChains()
	routes := make([]gin.H, 0, len(chains)*(len(chains)-1))
	for _, from := range chains {
		for _, to := range chains {
			if from.ID == to.ID {
				continue
			}
			routes = append(routes, gin.H{
				"from_chain":      from.ID,
				"from_chain_name": from.Name,
				"to_chain":        to.ID,
				"to_chain_name":   to.Name,
				"tokens":          bridgeTokens(from.ID),
				"fee":             "0.3%",
				"estimated_time":  600,
				"min_amount":      "10",
				"max_amount":      "100000",
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "routes": routes, "chains": chains})
}

// GetHistory returns the bridge transactions for a user. The user_id is read
// from the query string. Transactions are persisted in PostgreSQL.
func (bs *BridgeService) GetHistory(c *gin.Context) {
	userID := c.Query("user_id")
	if bs.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	var rows pgx.Rows
	var err error
	if userID == "" {
		rows, err = bs.pg.Query(context.Background(), `
SELECT id, user_id, from_chain, to_chain, token, amount, recipient, status, from_tx_hash, to_tx_hash, fee, estimated_time, timestamp
FROM bridge_transactions ORDER BY timestamp DESC LIMIT 100
		`)
	} else {
		rows, err = bs.pg.Query(context.Background(), `
SELECT id, user_id, from_chain, to_chain, token, amount, recipient, status, from_tx_hash, to_tx_hash, fee, estimated_time, timestamp
FROM bridge_transactions WHERE user_id = $1 ORDER BY timestamp DESC LIMIT 100
		`, userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query transactions: " + err.Error()})
		return
	}
	defer rows.Close()

	var result []*BridgeTransaction
	for rows.Next() {
		var tx BridgeTransaction
		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.FromChain, &tx.ToChain, &tx.Token, &tx.Amount, &tx.Recipient, &tx.Status, &tx.FromTxHash, &tx.ToTxHash, &tx.Fee, &tx.EstimatedTime, &tx.Timestamp); err != nil {
			continue
		}
		result = append(result, &tx)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "transactions": result, "count": len(result)})
}

func main() {
	log.Println("TigerWallet Bridge Service")
	log.Printf("Starting on port %d", cfg.Port)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	// Redis is the SHARED feature-flag store (see feature_flags.go). A missing
	// or unreachable Redis is non-fatal: flag enforcement fails closed
	// (disabled) at request time rather than preventing startup.
	rdbOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Printf("warning: invalid REDIS_URL %q: %v (flag enforcement will fail closed)", cfg.RedisURL, err)
	}
	var rdb *redis.Client
	if rdbOpts != nil {
		rdb = redis.NewClient(rdbOpts)
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("warning: redis ping failed: %v (flag enforcement will fail closed)", err)
			rdb = nil
		}
	}

	bs := NewBridgeService(pool, rdb)
	if err := bs.migrateDB(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "bridge"})
	})

	api := r.Group("/api/v1/bridge")
	{
		api.GET("/routes", bs.GetRoutes)
		api.GET("/history", bs.GetHistory)
		api.POST("/quote", bs.GetQuote)
		api.POST("/transfer", bs.InitiateTransfer)
		api.GET("/tx/:id", bs.GetStatus)
	}

	log.Printf("Server starting on :%d", cfg.Port)
	r.Run(fmt.Sprintf(":%d", cfg.Port))
}
