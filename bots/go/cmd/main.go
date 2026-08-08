// Bots Service - Trading Bot Platform
package main

import (
"context"
"encoding/json"
"fmt"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/gin-gonic/gin"
"github.com/golang-jwt/jwt/v5"
"github.com/google/uuid"
"github.com/jackc/pgx/v5/pgxpool"
"github.com/redis/go-redis/v9"
)

type Config struct {
Port        string
DatabaseURL string
RedisURL    string
JWTSecret   string
}

type Bot struct {
ID            uuid.UUID  `json:"id"`
UserID        uuid.UUID  `json:"user_id"`
Name          string     `json:"name"`
Strategy      string     `json:"strategy"`
Status        string     `json:"status"`
Network       string     `json:"network"`
TradingPairs  []string   `json:"trading_pairs"`
BaseToken     string     `json:"base_token"`
QuoteToken    string     `json:"quote_token"`
TotalProfit   string     `json:"total_profit"`
TotalTrades   int        `json:"total_trades"`
WinRate       float64    `json:"win_rate"`
IsActive      bool       `json:"is_active"`
StartedAt     *time.Time `json:"started_at"`
CreatedAt     time.Time  `json:"created_at"`
UpdatedAt     time.Time  `json:"updated_at"`
}

type Strategy struct {
ID          uuid.UUID `json:"id"`
Name        string   `json:"name"`
Type        string   `json:"type"`
Description string   `json:"description"`
RiskLevel   string   `json:"risk_level"`
IsActive    bool     `json:"is_active"`
CreatedAt   time.Time `json:"created_at"`
}

type BotTrade struct {
ID         uuid.UUID   `json:"id"`
BotID      uuid.UUID   `json:"bot_id"`
UserID     uuid.UUID   `json:"user_id"`
OrderType  string      `json:"order_type"`
Pair       string      `json:"pair"`
Amount     string      `json:"amount"`
Price      string      `json:"price"`
TotalValue string      `json:"total_value"`
Profit     string      `json:"profit"`
Status     string      `json:"status"`
ExecutedAt *time.Time `json:"executed_at"`
CreatedAt   time.Time  `json:"created_at"`
}

var db *pgxpool.Pool
var redis *redis.Client
var config Config
var logger *log.Logger
var jwtSecret []byte

func main() {
logger = log.New(os.Stdout, "Bots: ", log.LstdFlags)
logger.Println("Starting Bots Service...")

config.Port = getEnv("BOTS_PORT", "8107")
config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")
config.JWTSecret = getEnv("JWT_SECRET", "")
jwtSecret = []byte(config.JWTSecret)

var err error
db, err = pgxpool.New(context.Background(), config.DatabaseURL)
if err != nil {
to connect to database: %v", err)
}
logger.Println("Database connected")

opt, _ := redis.ParseURL(config.RedisURL)
redis = redis.NewClient(opt)
redis.Ping(context.Background())
logger.Println("Redis connected")

initDatabase()

gin.SetMode(gin.ReleaseMode)
router := gin.New()
router.Use(gin.Logger())
router.Use(gin.Recovery())

router.Use(func(c *gin.Context) {
trol-Allow-Origin", "*")
trol-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
trol-Allow-Headers", "Content-Type, Authorization")
c.Request.Method == "OPTIONS" {
oContent)

ext()
})

router.GET("/health", func(c *gin.Context) {
(http.StatusOK, gin.H{"status": "ok", "service": "bots"})
})

router.GET("/api/v1/strategies", getStrategies)
router.GET("/api/v1/market/prices", getMarketPrices)

api := router.Group("/api/v1")
api.Use(authMiddleware())
{
createBot)
getBots)
getBot)
startBot)
stopBot)
deleteBot)
getBotTrades)
ce", getBotPerformance)
getUserTrades)
}

logger.Printf("Starting server on port %s", config.Port)
srv := &http.Server{Addr: ":" + config.Port, Handler: router}

go func() {
err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
to start server: %v", err)
tln("Server started")
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

logger.Println("Shutting down server...")
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
srv.Shutdown(ctx)
db.Close()
redis.Close()
logger.Println("Server exited")
}

func initDatabase() {
db.Exec(context.Background(), `
TABLE IF NOT EXISTS bots (
UUID PRIMARY KEY DEFAULT gen_random_uuid(),
UUID NOT NULL,
ame VARCHAR(255) NOT NULL,
 VARCHAR(100) NOT NULL,
VARCHAR(50) DEFAULT 'stopped',
etwork VARCHAR(50),
g_pairs JSONB,
 VARCHAR(50),
 VARCHAR(50),
VARCHAR(100) DEFAULT '0',
INT DEFAULT 0,
_rate DECIMAL(5,2) DEFAULT 0,
BOOLEAN DEFAULT true,
TIMESTAMP,
TIMESTAMP DEFAULT NOW(),
TIMESTAMP DEFAULT NOW()
TABLE IF NOT EXISTS strategies (
UUID PRIMARY KEY DEFAULT gen_random_uuid(),
ame VARCHAR(255) NOT NULL,
pe VARCHAR(100) NOT NULL,
 TEXT,
VARCHAR(50) DEFAULT 'medium',
BOOLEAN DEFAULT true,
TIMESTAMP DEFAULT NOW()
TABLE IF NOT EXISTS bot_trades (
UUID PRIMARY KEY DEFAULT gen_random_uuid(),
UUID REFERENCES bots(id),
UUID NOT NULL,
pe VARCHAR(50) NOT NULL,
VARCHAR(50) NOT NULL,
t VARCHAR(100) NOT NULL,
VARCHAR(100) NOT NULL,
VARCHAR(100) NOT NULL,
VARCHAR(100),
VARCHAR(50) DEFAULT 'pending',
TIMESTAMP,
TIMESTAMP DEFAULT NOW()
INDEX IF NOT EXISTS idx_bots_user ON bots(user_id);
INDEX IF NOT EXISTS idx_trades_bot ON bot_trades(bot_id);
`)

db.Exec(context.Background(), `
SERT INTO strategies (name, type, description, risk_level, is_active)

Trading', 'grid', 'Buy low sell high in grid', 'low', true),
Bot', 'dca', 'Dollar Cost Averaging', 'low', true),
'arbitrage', 'Price difference profits', 'high', true),
tum', 'momentum', 'Trend following', 'medium', true),
Trading', 'ai', 'AI-powered signals', 'high', true)
 CONFLICT DO NOTHING
`)
}

func getStrategies(c *gin.Context) {
rows, _ := db.Query(context.Background(), `SELECT id, name, type, description, risk_level, is_active, created_at FROM strategies WHERE is_active = true`)
defer rows.Close()

var strategies []Strategy
for rows.Next() {
s Strategy
(&s.ID, &s.Name, &s.Type, &s.Description, &s.RiskLevel, &s.IsActive, &s.CreatedAt)
= append(strategies, s)
}
c.JSON(http.StatusOK, gin.H{"strategies": strategies})
}

func getMarketPrices(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"prices": map[string]map[string]string{
{"price": "67500.00", "change": "3.2"},
{"price": "3250.50", "change": "2.5"},
}})
}

func createBot(c *gin.Context) {
userID := c.GetString("user_id")
var req struct {
ame          string   `json:"name"`
      string   `json:"strategy"`
etwork       string   `json:"network"`
gPairs  []string `json:"trading_pairs"`
}
c.ShouldBindJSON(&req)

uid, _ := uuid.Parse(userID)
bot := Bot{
           uuid.New(),
       uid,
ame:          req.Name,
:      req.Strategy,
       "stopped",
etwork:       req.Network,
gPairs:  req.TradingPairs,
  "0",
  0,
Rate:       0,
     true,
    time.Now(),
    time.Now(),
}

db.Exec(context.Background(), `INSERT INTO bots (id, user_id, name, strategy, status, network, trading_pairs, total_profit, total_trades, win_rate, is_active, created_at, updated_at)
($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
bot.UserID, bot.Name, bot.Strategy, bot.Status, bot.Network, bot.TradingPairs, bot.TotalProfit, bot.TotalTrades, bot.WinRate, bot.IsActive, bot.CreatedAt, bot.UpdatedAt)

c.JSON(http.StatusCreated, bot)
}

func getBots(c *gin.Context) {
userID := c.GetString("user_id")
uid, _ := uuid.Parse(userID)

rows, _ := db.Query(context.Background(), `SELECT id, user_id, name, strategy, status, network, total_profit, total_trades, win_rate, is_active, created_at FROM bots WHERE user_id = $1 AND is_active = true`, uid)
defer rows.Close()

var bots []Bot
for rows.Next() {
b Bot
(&b.ID, &b.UserID, &b.Name, &b.Strategy, &b.Status, &b.Network, &b.TotalProfit, &b.TotalTrades, &b.WinRate, &b.IsActive, &b.CreatedAt)
= append(bots, b)
}
c.JSON(http.StatusOK, gin.H{"bots": bots})
}

func getBot(c *gin.Context) {
botID := c.Param("id")
uid, _ := uuid.Parse(botID)

var bot Bot
err := db.QueryRow(context.Background(), `SELECT id, user_id, name, strategy, status, network, trading_pairs, total_profit, total_trades, win_rate, is_active, started_at, created_at FROM bots WHERE id = $1`, uid).Scan(&bot.ID, &bot.UserID, &bot.Name, &bot.Strategy, &bot.Status, &bot.Network, &bot.TradingPairs, &bot.TotalProfit, &bot.TotalTrades, &bot.WinRate, &bot.IsActive, &bot.StartedAt, &bot.CreatedAt)

if err != nil {
(http.StatusNotFound, gin.H{"error": "bot not found"})

}
c.JSON(http.StatusOK, bot)
}

func startBot(c *gin.Context) {
botID := c.Param("id")
uid, _ := uuid.Parse(botID)
now := time.Now()
db.Exec(context.Background(), `UPDATE bots SET status = 'running', started_at = $1, updated_at = $1 WHERE id = $2`, now, uid)
c.JSON(http.StatusOK, gin.H{"message": "bot started"})
}

func stopBot(c *gin.Context) {
botID := c.Param("id")
uid, _ := uuid.Parse(botID)
db.Exec(context.Background(), `UPDATE bots SET status = 'stopped', updated_at = NOW() WHERE id = $1`, uid)
c.JSON(http.StatusOK, gin.H{"message": "bot stopped"})
}

func deleteBot(c *gin.Context) {
botID := c.Param("id")
uid, _ := uuid.Parse(botID)
db.Exec(context.Background(), `UPDATE bots SET is_active = false, status = 'stopped', updated_at = NOW() WHERE id = $1`, uid)
c.JSON(http.StatusOK, gin.H{"message": "bot deleted"})
}

func getBotTrades(c *gin.Context) {
botID := c.Param("id")
uid, _ := uuid.Parse(botID)

rows, _ := db.Query(context.Background(), `SELECT id, bot_id, order_type, pair, amount, price, total_value, profit, status, executed_at, created_at FROM bot_trades WHERE bot_id = $1 ORDER BY created_at DESC LIMIT 50`, uid)
defer rows.Close()

var trades []BotTrade
for rows.Next() {
t BotTrade
(&t.ID, &t.BotID, &t.OrderType, &t.Pair, &t.Amount, &t.Price, &t.TotalValue, &t.Profit, &t.Status, &t.ExecutedAt, &t.CreatedAt)
= append(trades, t)
}
c.JSON(http.StatusOK, gin.H{"trades": trades})
}

func getBotPerformance(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"performance": []}})
}

func getUserTrades(c *gin.Context) {
userID := c.GetString("user_id")
uid, _ := uuid.Parse(userID)

rows, _ := db.Query(context.Background(), `SELECT id, bot_id, order_type, pair, amount, price, profit, status, created_at FROM bot_trades WHERE user_id = $1 ORDER BY created_at DESC LIMIT 100`, uid)
defer rows.Close()

var trades []BotTrade
for rows.Next() {
t BotTrade
(&t.ID, &t.BotID, &t.OrderType, &t.Pair, &t.Amount, &t.Price, &t.Profit, &t.Status, &t.CreatedAt)
= append(trades, t)
}
c.JSON(http.StatusOK, gin.H{"trades": trades})
}

func authMiddleware() gin.HandlerFunc {
return func(c *gin.Context) {
:= c.GetHeader("Authorization")
authHeader == "" {
(http.StatusUnauthorized, gin.H{"error": "no authorization"})

String := authHeader[7:]
, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
 jwtSecret, nil
err != nil || !token.Valid {
(http.StatusUnauthorized, gin.H{"error": "invalid token"})

:= token.Claims.(jwt.MapClaims)
claims["user_id"].(string))
ext()
}
}

func getEnv(key, defaultValue string) string {
if value, exists := os.LookupEnv(key); exists {
 value
}
return defaultValue
}
