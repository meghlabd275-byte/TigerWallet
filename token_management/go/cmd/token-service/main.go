package main

import (
"encoding/json"
"fmt"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/gin-gonic/gin"
"github.com/google/uuid"
)

// Token represents a cryptocurrency token
type Token struct {
ID            string    `json:"id"`
Address       string    `json:"address"`
Symbol        string    `json:"symbol"`
Name          string    `json:"name"`
Decimals      int       `json:"decimals"`
ChainID       int       `json:"chain_id"`
TotalSupply   string    `json:"total_supply"`
IsVerified    bool      `json:"is_verified"`
IsSpam        bool      `json:"is_spam"`
Price         float64   `json:"price"`
MarketCap     float64   `json:"market_cap"`
Volume24h     float64   `json:"volume_24h"`
LogoURL       string    `json:"logo_url"`
Website       string    `json:"website"`
AddedAt       time.Time `json:"added_at"`
}

// Token Alert
type TokenAlert struct {
ID            string    `json:"id"`
UserID        string    `json:"user_id"`
TokenID       string    `json:"token_id"`
Condition     string    `json:"condition"` // above, below
TargetPrice   float64   `json:"target_price"`
IsActive      bool      `json:"is_active"`
TriggeredAt   *time.Time `json:"triggered_at,omitempty"`
CreatedAt     time.Time `json:"created_at"`
}

// Token Management Service
type TokenService struct {
tokens map[string]Token
alerts map[string][]TokenAlert
}

func NewTokenService() *TokenService {
return &TokenService{
s: make(map[string]Token),
make(map[string][]TokenAlert),
}
}

func main() {
service := NewTokenService()

// Initialize mock tokens
service.initMockTokens()

gin.SetMode(gin.ReleaseMode)
router := gin.New()
router.Use(gin.Recovery())

router.GET("/health", func(c *gin.Context) {
(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now().Unix()})
})

api := router.Group("/api/v1")
{
s", service.listTokens)
s/:id", service.getToken)
s/search", service.searchTokens)
s/verify", service.verifyToken)
service.createAlert)
service.listAlerts)
service.deleteAlert)
service.reportSpam)
service.getFilteredTokens)
}

// Price update worker
go service.priceUpdateWorker()

go func() {
tln("Starting token service on port 8083")
AndServe(":8083", router)
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit
log.Println("Shutting down token service...")
}

func (s *TokenService) initMockTokens() {
tokens := []Token{
"1", Address: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", Symbol: "WBTC", Name: "Wrapped Bitcoin",
8, ChainID: 1, TotalSupply: "140000", IsVerified: true, Price: 64500, MarketCap: 9000000000,
500000000, LogoURL: "https://cryptologos.cc/logos/wrapped-bitcoin-wbtc-logo.png",
"2", Address: "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9", Symbol: "AAVE", Name: "Aave Token",
18, ChainID: 1, TotalSupply: "16000000", IsVerified: true, Price: 285, MarketCap: 4200000000,
180000000, LogoURL: "https://cryptologos.cc/logos/aave-aave-logo.png",
"3", Address: "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984", Symbol: "UNI", Name: "Uniswap",
18, ChainID: 1, TotalSupply: "1000000000", IsVerified: true, Price: 12.5, MarketCap: 9500000000,
420000000, LogoURL: "https://cryptologos.cc/logos/uniswap-uni-logo.png",
_, t := range tokens {
s[t.ID] = t
}
}

func (s *TokenService) listTokens(c *gin.Context) {
verified := c.Query("verified")
chainID := c.Query("chain_id")

var tokens []Token
for _, t := range s.tokens {
verified == "true" && !t.IsVerified { continue }
chainID != "" && t.ChainID != 0 {
id := 0; id != 0 && t.ChainID != id { continue }
t.IsSpam { continue }
s = append(tokens, t)
}

c.JSON(http.StatusOK, gin.H{"tokens": tokens, "total": len(tokens)})
}

func (s *TokenService) getToken(c *gin.Context) {
id := c.Param("id")
if token, ok := s.tokens[id]; ok {
(http.StatusOK, token)
} else {
(http.StatusNotFound, gin.H{"error": "Token not found"})
}
}

func (s *TokenService) searchTokens(c *gin.Context) {
query := c.Query("q")
if query == "" {
(http.StatusBadRequest, gin.H{"error": "Query required"})

}

var results []Token
for _, t := range s.tokens {
t.IsSpam { continue }
contains(t.Name, query) || contains(t.Symbol, query) {
= append(results, t)
(http.StatusOK, gin.H{"tokens": results, "total": len(results)})
}

func (s *TokenService) verifyToken(c *gin.Context) {
var req struct {
string `json:"address" binding:"required"`
ID int    `json:"chain_id" binding:"required"`
}

if err := c.ShouldBindJSON(&req); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

// Mock verification
token := Token{
uuid.New().String(),
req.Address,
mbol: "NEW",
ame: "New Token",
18,
ID: req.ChainID,
true,
time.Now(),
}

s.tokens[token.ID] = token
c.JSON(http.StatusCreated, token)
}

func (s *TokenService) createAlert(c *gin.Context) {
var alert TokenAlert
if err := c.ShouldBindJSON(&alert); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

alert.ID = uuid.New().String()
alert.CreatedAt = time.Now()
alert.IsActive = true

s.alerts[alert.UserID] = append(s.alerts[alert.UserID], alert)
c.JSON(http.StatusCreated, alert)
}

func (s *TokenService) listAlerts(c *gin.Context) {
userID := c.Param("user_id")
if alerts, ok := s.alerts[userID]; ok {
(http.StatusOK, gin.H{"alerts": alerts})
} else {
(http.StatusOK, gin.H{"alerts": []})
}
}

func (s *TokenService) deleteAlert(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"message": "Alert deleted"})
}

func (s *TokenService) reportSpam(c *gin.Context) {
var req struct {
ID string `json:"token_id" binding:"required"`
  string `json:"reason" binding:"required"`
}

if err := c.ShouldBindJSON(&req); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

if token, ok := s.tokens[req.TokenID]; ok {
.IsSpam = true
s[req.TokenID] = token
}

c.JSON(http.StatusOK, gin.H{"message": "Spam reported"})
}

func (s *TokenService) getFilteredTokens(c *gin.Context) {
var tokens []Token
for _, t := range s.tokens {
!t.IsSpam && t.IsVerified {
s = append(tokens, t)
(http.StatusOK, gin.H{"tokens": tokens})
}

func (s *TokenService) priceUpdateWorker() {
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for range ticker {
i, t := range s.tokens {
Mock price update
ge := (float64(len(t.ID)) * 0.01) - 0.005
= t.Price * (1 + change)
s[i] = t
c contains(s, substr string) bool {
return len(s) >= len(substr) && 
== substr || 
(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)))
}
