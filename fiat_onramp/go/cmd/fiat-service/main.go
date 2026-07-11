package main

import (
"context"
"encoding/json"
"fmt"
"log"
"net/http"
"os"
"os/signal"
"strings"
"syscall"
"time"

"github.com/gin-gonic/gin"
"github.com/google/uuid"
)

// Configuration
type Config struct {
Port             string
StripeKey        string
MoonPayAPIKey   string
RampAPIKey      string
DatabaseURL     string
}

// Fiat Provider
type Provider string

const (
ProviderStripe   Provider = "stripe"
ProviderMoonPay  Provider = "moonpay"
ProviderRamp     Provider = "ramp"
ProviderTransak Provider = "transak"
)

// Fiat Order
type FiatOrder struct {
ID            string    `json:"id"`
UserID        string    `json:"user_id"`
Provider      Provider  `json:"provider"`
FiatAmount    float64   `json:"fiat_amount"`
CryptoAmount  float64   `json:"crypto_amount"`
Status        string    `json:"status"`
ExternalID    string    `json:"external_id"`
CreatedAt     time.Time `json:"created_at"`
}

// Fiat Service
type FiatService struct {
config     *Config
httpServer *http.Server
}

func New(config *Config) (*FiatService, error) {
return &FiatService{config: config}, nil
}

func (s *FiatService) Start() error {
gin.SetMode(gin.ReleaseMode)
router := gin.New()
router.Use(gin.Recovery())

router.GET("/health", func(c *gin.Context) {
(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now().Unix()})
})

api := router.Group("/api/v1")
api.POST("/orders", s.createOrder)
api.GET("/orders/:id", s.getOrder)
api.GET("/orders", s.listOrders)
api.POST("/quote", s.getQuote)
api.GET("/rates", s.getRates)

s.httpServer = &http.Server{
        ":" + s.config.Port,
dler:      router,
 15 * time.Second,
15 * time.Second,
}

go func() {
tf("Starting fiat service on port %s", s.config.Port)
AndServe()
}()

return nil
}

func (s *FiatService) createOrder(c *gin.Context) {
var req struct {
      string  `json:"user_id" binding:"required"`
t   float64 `json:"fiat_amount" binding:"required,gt=0"`
cy string  `json:"fiat_currency" binding:"required"`
ptoAsset  string  `json:"crypto_asset" binding:"required"`
}

if err := c.ShouldBindJSON(&req); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

rate := getMockRate(req.FiatCurrency, req.CryptoAsset)
order := FiatOrder{
          uuid.New().String(),
      req.UserID,
   ProviderStripe,
t:  req.FiatAmount,
ptoAmount: req.FiatAmount / rate,
     "pending",
alID:  "ext_" + uuid.New().String(),
  time.Now(),
}

c.JSON(http.StatusCreated, order)
}

func (s *FiatService) getOrder(c *gin.Context) {
id := c.Param("id")
order := FiatOrder{
          id,
      "user123",
t:   100.0,
ptoAmount: 0.0015,
      "completed",
   time.Now(),
}
c.JSON(http.StatusOK, order)
}

func (s *FiatService) listOrders(c *gin.Context) {
orders := []FiatOrder{
"1", UserID: "user123", FiatAmount: 100.0, CryptoAmount: 0.0015, Status: "completed", CreatedAt: time.Now()},
"2", UserID: "user123", FiatAmount: 50.0, CryptoAmount: 0.014, Status: "pending", CreatedAt: time.Now()},
}
c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (s *FiatService) getQuote(c *gin.Context) {
var req struct {
t   float64 `json:"fiat_amount"`
cy string  `json:"fiat_currency"`
ptoAsset  string  `json:"crypto_asset"`
}
c.ShouldBindJSON(&req)

rate := getMockRate(req.FiatCurrency, req.CryptoAsset)
c.JSON(http.StatusOK, gin.H{
t":    req.FiatAmount,
pto_amount":  req.FiatAmount / rate,
ge_rate":  rate,
           req.FiatAmount * 0.035,
         req.FiatAmount * 1.035,
})
}

func (s *FiatService) getRates(c *gin.Context) {
rates := []map[string]interface{}{
"USD", "to": "BTC", "rate": 1 / 65000},
"USD", "to": "ETH", "rate": 1 / 3500},
"USD", "to": "USDT", "rate": 1.0},
"USD", "to": "SOL", "rate": 1 / 145},
}
c.JSON(http.StatusOK, gin.H{"rates": rates})
}

func getMockRate(from, to string) float64 {
rates := map[string]float64{
 1 / 65000,
 1 / 3500,
1,
 1 / 145,
}
if rate, ok := rates[from+"-"+to]; ok {
 rate
}
return 0.00001
}

func main() {
config := &Config{Port: "8081"}
if p := os.Getenv("PORT"); p != "" {
fig.Port = p
}

service, _ := New(config)
service.Start()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

log.Println("Shutting down fiat service...")
}
