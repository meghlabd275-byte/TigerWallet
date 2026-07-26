package main

import (
"context"
"encoding/json"
"fmt"
"log"
"net/http"
"os"
"os/signal"
"sync"
"syscall"
"time"

"github.com/gin-gonic/gin"
"github.com/go-redis/redis/v8"
)

type Config struct {
Port     string
RedisURL string
}

func LoadConfig() *Config {
return &Config{
    getEnv("PORT", "8451"),
getEnv("REDIS_URL", "redis://localhost:6379"),
}
}

func getEnv(key, def string) string {
if v := os.Getenv(key); v != "" { return v }
return def
}

type Provider struct {
ID                    string   `json:"id"`
Name                  string   `json:"name"`
Logo                  string   `json:"logo"`
FiatCurrency          string   `json:"fiatCurrency"`
CryptoCurrency        string   `json:"cryptoCurrency"`
MinAmount             float64  `json:"minAmount"`
MaxAmount             float64  `json:"maxAmount"`
FeePercent            float64  `json:"feePercent"`
ProcessingTime        string   `json:"processingTime"`
SupportedCountries    []string `json:"supportedCountries"`
PaymentMethods        []string `json:"paymentMethods"`
}

type Order struct {
ID                string  `json:"id"`
UserID            string  `json:"userId"`
ProviderID        string  `json:"providerId"`
FiatAmount        float64 `json:"fiatAmount"`
CryptoAmount      string  `json:"cryptoAmount"`
FiatCurrency      string  `json:"fiatCurrency"`
CryptoCurrency    string  `json:"cryptoCurrency"`
ExchangeRate      float64 `json:"exchangeRate"`
FeeAmount         float64 `json:"feeAmount"`
NetworkFee        float64 `json:"networkFee"`
TotalAmount       float64 `json:"totalAmount"`
RecipientAddress  string  `json:"recipientAddress"`
Status            string  `json:"status"`
PaymentMethod     string  `json:"paymentMethod"`
KYCStatus         string  `json:"kycStatus"`
CreatedAt         int64   `json:"createdAt"`
UpdatedAt         int64   `json:"updatedAt"`
}

type FiatRampService struct {
config    *Config
redis     *redis.Client
orders    map[string]*Order
providers map[string]Provider
mu        sync.RWMutex
}

func NewFiatRampService(cfg *Config) *FiatRampService {
redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})

providers := map[string]Provider{
pay": {ID: "moonpay", Name: "MoonPay", Logo: "https://cryptologos.cc/logos/moonpay-mpay-logo.png", FiatCurrency: "USD", CryptoCurrency: "ETH", MinAmount: 30, MaxAmount: 50000, FeePercent: 4.5, ProcessingTime: "5-30 minutes", SupportedCountries: []string{"US", "UK", "EU", "AU", "CA"}, PaymentMethods: []string{"card", "bank_transfer"}},
sak": {ID: "transak", Name: "Transak", Logo: "https://cryptologos.cc/logos/transak-trak-logo.png", FiatCurrency: "USD", CryptoCurrency: "ETH", MinAmount: 20, MaxAmount: 25000, FeePercent: 3.5, ProcessingTime: "10-45 minutes", SupportedCountries: []string{"US", "UK", "EU", "IN", "SG"}, PaymentMethods: []string{"card", "bank_transfer", "apple_pay"}},
ramper": {ID: "onramper", Name: "Onramper", Logo: "https://cryptologos.cc/logos/onramper-onramp-logo.png", FiatCurrency: "USD", CryptoCurrency: "BTC", MinAmount: 50, MaxAmount: 10000, FeePercent: 2.9, ProcessingTime: "15-60 minutes", SupportedCountries: []string{"US", "UK", "EU"}, PaymentMethods: []string{"card", "bank_transfer"}},
{ID: "simplex", Name: "Simplex", Logo: "https://cryptologos.cc/logos/simplex-logo.png", FiatCurrency: "USD", CryptoCurrency: "USDT", MinAmount: 50, MaxAmount: 100000, FeePercent: 3.9, ProcessingTime: "5-20 minutes", SupportedCountries: []string{"US", "UK", "EU", "AU"}, PaymentMethods: []string{"card"}},
}

return &FiatRampService{config: cfg, redis: redisClient, orders: make(map[string]*Order), providers: providers}
}

func (s *FiatRampService) GetQuote(providerID string, amount, fiatCurr, cryptoCurr, payMethod string) (map[string]interface{}, error) {
provider, ok := s.providers[providerID]
if !ok { return nil, fmt.Errorf("provider not found") }

amt, _ := strconv.ParseFloat(amount, 64)
if amt < provider.MinAmount { return nil, fmt.Errorf("min amount is %v", provider.MinAmount) }
if amt > provider.MaxAmount { return nil, fmt.Errorf("max amount is %v", provider.MaxAmount) }

price := s.getCryptoPrice(cryptoCurr)
cryptoAmt := amt / price
feeAmt := amt * (provider.FeePercent / 100)
networkFee := 0.50
totalAmt := amt + feeAmt + networkFee

return map[string]interface{}{
provider.ID, "providerName": provider.Name, "fiatAmount": amt,
ptoAmount": fmt.Sprintf("%.8f", cryptoAmt), "cryptoCurrency": cryptoCurr,
geRate": price, "feeAmount": feeAmt, "networkFee": networkFee,
t": totalAmt, "processingTime": provider.ProcessingTime,
}, nil
}

func (s *FiatRampService) getCryptoPrice(symbol string) float64 {
prices := map[string]float64{"ETH": 2500.0, "BTC": 45000.0, "USDT": 1.0, "USDC": 1.0, "BNB": 300.0, "MATIC": 0.85, "AVAX": 35.0}
if p, ok := prices[symbol]; ok { return p }
return 1.0
}

func (s *FiatRampService) GetProviders() []Provider {
list := []Provider{}
for _, p := range s.providers { list = append(list, p) }
return list
}

func (s *FiatRampService) CreateOrder(userID, providerID, amount, fiatCurr, cryptoCurr, recipient, payMethod string) (*Order, error) {
provider, ok := s.providers[providerID]
if !ok { return nil, fmt.Errorf("provider not found") }

amt, _ := strconv.ParseFloat(amount, 64)
if amt < provider.MinAmount { return nil, fmt.Errorf("min amount is %v", provider.MinAmount) }
if amt > provider.MaxAmount { return nil, fmt.Errorf("max amount is %v", provider.MaxAmount) }

price := s.getCryptoPrice(cryptoCurr)
cryptoAmt := amt / price
feeAmt := amt * (provider.FeePercent / 100)
networkFee := 0.50
totalAmt := amt + feeAmt + networkFee

order := &Order{
fmt.Sprintf("ramp-%d-%s", time.Now().Unix(), randomStr(8)),
userID, ProviderID: providerID, FiatAmount: amt, CryptoAmount: fmt.Sprintf("%.8f", cryptoAmt),
cy: fiatCurr, CryptoCurrency: cryptoCurr, ExchangeRate: price,
t: feeAmt, NetworkFee: networkFee, TotalAmount: totalAmt,
tAddress: recipient, Status: "pending", PaymentMethod: payMethod,
CStatus: "pending", CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(),
}

s.mu.Lock()
s.orders[order.ID] = order
s.mu.Unlock()
return order, nil
}

func (s *FiatRampService) GetOrder(orderID string) *Order {
s.mu.RLock()
defer s.mu.RUnlock()
return s.orders[orderID]
}

func (s *FiatRampService) GetUserOrders(userID string) []*Order {
s.mu.RLock()
defer s.mu.RUnlock()
list := []*Order{}
for _, o := range s.orders {
o.UserID == userID { list = append(list, o) }
}
return list
}

func randomStr(length int) string {
cs := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
b := make([]byte, length)
for i := range b { b[i] = cs[i%len(cs)] }
return string(b)
}

func (s *FiatRampService) RegisterRoutes(r *gin.Engine) {
r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "fiat-ramp"}) })
api := r.Group("/api/v1/ramp")
api.GET("/providers", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"providers": s.GetProviders()}) })
api.POST("/quote", func(c *gin.Context) {
req struct {
string `json:"providerId" binding:"required"`
t string `json:"amount" binding:"required"`
cy string `json:"fiatCurrency" binding:"required"`
ptoCurrency string `json:"cryptoCurrency" binding:"required"`
mentMethod string `json:"paymentMethod" binding:"required"`
err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
err := s.GetQuote(req.ProviderID, req.Amount, req.FiatCurrency, req.CryptoCurrency, req.PaymentMethod)
err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
(http.StatusOK, quote)
})
api.POST("/order", func(c *gin.Context) {
req struct {
string `json:"userId" binding:"required"`
string `json:"providerId" binding:"required"`
t string `json:"amount" binding:"required"`
cy string `json:"fiatCurrency" binding:"required"`
ptoCurrency string `json:"cryptoCurrency" binding:"required"`
tAddress string `json:"recipientAddress" binding:"required"`
mentMethod string `json:"paymentMethod" binding:"required"`
err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
err := s.CreateOrder(req.UserID, req.ProviderID, req.Amount, req.FiatCurrency, req.CryptoCurrency, req.RecipientAddress, req.PaymentMethod)
err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
(http.StatusCreated, order)
})
api.GET("/order/:orderId", func(c *gin.Context) {
:= s.GetOrder(c.Param("orderId"))
order == nil { c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return }
(http.StatusOK, order)
})
api.GET("/orders/:userId", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"orders": s.GetUserOrders(c.Param("userId"))}) })
}

func main() {
cfg := LoadConfig()
svc := NewFiatRampService(cfg)
gin.SetMode(gin.ReleaseMode)
r := gin.New()
r.Use(gin.Recovery())
svc.RegisterRoutes(r)
srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
go func() { log.Printf("Fiat Ramp starting on %s", cfg.Port); srv.ListenAndServe() }()
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit
log.Println("Shutting down...")
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
srv.Shutdown(ctx)
}
