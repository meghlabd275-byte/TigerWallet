package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	cryptorand "crypto/rand"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Port     string
	RedisURL string
}

func LoadConfig() *Config {
	return &Config{
		Port:     getEnv("PORT", "8451"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type Provider struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Logo               string   `json:"logo"`
	FiatCurrency       string   `json:"fiatCurrency"`
	CryptoCurrency     string   `json:"cryptoCurrency"`
	MinAmount          float64  `json:"minAmount"`
	MaxAmount          float64  `json:"maxAmount"`
	FeePercent         float64  `json:"feePercent"`
	ProcessingTime     string   `json:"processingTime"`
	SupportedCountries []string `json:"supportedCountries"`
	PaymentMethods     []string `json:"paymentMethods"`
}

type Order struct {
	ID               string  `json:"id"`
	UserID           string  `json:"userId"`
	ProviderID       string  `json:"providerId"`
	FiatAmount       float64 `json:"fiatAmount"`
	CryptoAmount     string  `json:"cryptoAmount"`
	FiatCurrency     string  `json:"fiatCurrency"`
	CryptoCurrency   string  `json:"cryptoCurrency"`
	ExchangeRate     float64 `json:"exchangeRate"`
	FeeAmount        float64 `json:"feeAmount"`
	NetworkFee       float64 `json:"networkFee"`
	TotalAmount      float64 `json:"totalAmount"`
	RecipientAddress string  `json:"recipientAddress"`
	Status           string  `json:"status"`
	PaymentMethod    string  `json:"paymentMethod"`
	KYCStatus        string  `json:"kycStatus"`
	CreatedAt        int64   `json:"createdAt"`
	UpdatedAt        int64   `json:"updatedAt"`
}

type FiatRampService struct {
	config    *Config
	redis     *redis.Client
	orders    map[string]*Order
	providers map[string]Provider
	// providerKeys holds admin-configured provider API keys at runtime
	// (set via POST /api/v1/ramp/admin/providers/:id/key). When a key is
	// present here it takes precedence over the env var, so the admin can
	// configure provider APIs from the panel without redeploying.
	providerKeys map[string]string
	mu           sync.RWMutex
}

func NewFiatRampService(cfg *Config) *FiatRampService {
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})

	providers := map[string]Provider{
		"moonpay":  {ID: "moonpay", Name: "MoonPay", Logo: "https://cryptologos.cc/logos/moonpay-mpay-logo.png", FiatCurrency: "USD", CryptoCurrency: "ETH", MinAmount: 30, MaxAmount: 50000, FeePercent: 4.5, ProcessingTime: "5-30 minutes", SupportedCountries: []string{"US", "UK", "EU", "AU", "CA"}, PaymentMethods: []string{"card", "bank_transfer"}},
		"transak":  {ID: "transak", Name: "Transak", Logo: "https://cryptologos.cc/logos/transak-trak-logo.png", FiatCurrency: "USD", CryptoCurrency: "ETH", MinAmount: 20, MaxAmount: 25000, FeePercent: 3.5, ProcessingTime: "10-45 minutes", SupportedCountries: []string{"US", "UK", "EU", "IN", "SG"}, PaymentMethods: []string{"card", "bank_transfer", "apple_pay"}},
		"onramper": {ID: "onramper", Name: "Onramper", Logo: "https://cryptologos.cc/logos/onramper-onramp-logo.png", FiatCurrency: "USD", CryptoCurrency: "BTC", MinAmount: 50, MaxAmount: 10000, FeePercent: 2.9, ProcessingTime: "15-60 minutes", SupportedCountries: []string{"US", "UK", "EU"}, PaymentMethods: []string{"card", "bank_transfer"}},
		"simplex":  {ID: "simplex", Name: "Simplex", Logo: "https://cryptologos.cc/logos/simplex-logo.png", FiatCurrency: "USD", CryptoCurrency: "USDT", MinAmount: 50, MaxAmount: 100000, FeePercent: 3.9, ProcessingTime: "5-20 minutes", SupportedCountries: []string{"US", "UK", "EU", "AU"}, PaymentMethods: []string{"card"}},
	}

	return &FiatRampService{config: cfg, redis: redisClient, orders: make(map[string]*Order), providers: providers, providerKeys: make(map[string]string)}
}

// GetQuote returns a real fiat<->crypto quote. The exchange rate is fetched
// live from CoinGecko (never a hardcoded price table). The provider checkout
// URL is the real Transak/MoonPay hosted widget URL when the provider's API
// key is configured via env, otherwise the quote is returned without a URL
// (fail-open on the URL, fail-closed on the price — never a fabricated rate).
func (s *FiatRampService) GetQuote(providerID string, amount, fiatCurr, cryptoCurr, payMethod string) (map[string]interface{}, error) {
	provider, ok := s.providers[providerID]
	if !ok {
		return nil, fmt.Errorf("provider not found")
	}

	amt, err := strconv.ParseFloat(amount, 64)
	if err != nil || amt <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	if amt < provider.MinAmount {
		return nil, fmt.Errorf("min amount is %v", provider.MinAmount)
	}
	if amt > provider.MaxAmount {
		return nil, fmt.Errorf("max amount is %v", provider.MaxAmount)
	}

	price, err := s.getCryptoPrice(cryptoCurr)
	if err != nil {
		return nil, fmt.Errorf("price unavailable for %s: %v", cryptoCurr, err)
	}
	cryptoAmt := amt / price
	feeAmt := amt * (provider.FeePercent / 100)
	networkFee := 0.50
	totalAmt := amt + feeAmt + networkFee

	quote := map[string]interface{}{
		"providerId": provider.ID, "providerName": provider.Name, "fiatAmount": amt,
		"cryptoAmount": fmt.Sprintf("%.8f", cryptoAmt), "cryptoCurrency": cryptoCurr,
		"exchangeRate": price, "feeAmount": feeAmt, "networkFee": networkFee,
		"totalAmount": totalAmt, "processingTime": provider.ProcessingTime,
		"priceSource": "coingecko",
	}
	if url := s.buildProviderURL(provider, fiatCurr, cryptoCurr, amount, payMethod); url != "" {
		quote["checkoutUrl"] = url
	}
	return quote, nil
}

// getCryptoPrice fetches the live USD price from CoinGecko's simple/price
// endpoint and caches it in Redis for 60s. Stablecoins with no CoinGecko id
// map to 1.0. Returns an error (never a fabricated fallback) on failure.
func (s *FiatRampService) getCryptoPrice(symbol string) (float64, error) {
	upper := strings.ToUpper(symbol)
	if upper == "USDT" || upper == "USDC" || upper == "DAI" || upper == "BUSD" {
		return 1.0, nil
	}
	cacheKey := "fiatramp:price:" + upper
	if s.redis != nil {
		if cached, err := s.redis.Get(context.Background(), cacheKey).Result(); err == nil {
			if p, err := strconv.ParseFloat(cached, 64); err == nil {
				return p, nil
			}
		}
	}
	coinID := coingeckoID(upper)
	if coinID == "" {
		return 0, fmt.Errorf("unsupported symbol %s", symbol)
	}
	url := "https://api.coingecko.com/api/v3/simple/price?ids=" + coinID + "&vs_currencies=usd"
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	var parsed map[string]map[string]float64
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	inner, ok := parsed[coinID]
	if !ok {
		return 0, fmt.Errorf("price not found for %s", symbol)
	}
	price, ok := inner["usd"]
	if !ok || price <= 0 {
		return 0, fmt.Errorf("invalid price for %s", symbol)
	}
	if s.redis != nil {
		s.redis.Set(context.Background(), cacheKey, fmt.Sprintf("%f", price), 60*time.Second)
	}
	return price, nil
}

func coingeckoID(symbol string) string {
	m := map[string]string{
		"ETH": "ethereum", "BTC": "bitcoin", "BNB": "binancecoin", "MATIC": "matic-network",
		"AVAX": "avalanche-2", "SOL": "solana", "ADA": "cardano", "DOT": "polkadot",
		"LINK": "chainlink", "UNI": "uniswap", "ATOM": "cosmos", "LTC": "litecoin",
		"XRP": "ripple", "DOGE": "dogecoin", "ARB": "arbitrum", "OP": "optimism",
	}
	return m[symbol]
}

// buildProviderURL returns the real hosted-checkout URL for Transak/MoonPay
// when the provider's API key is configured via env (TRANSAK_API_KEY /
// MOONPAY_API_KEY). Returns "" if no key is configured (the quote is still
// valid; the admin must set the key to enable hosted checkout).
func (s *FiatRampService) buildProviderURL(p Provider, fiatCurr, cryptoCurr, amount, payMethod string) string {
	switch p.ID {
	case "transak":
		key := s.getProviderKey("transak")
		if key == "" {
			return ""
		}
		env := os.Getenv("TRANSAK_ENV")
		host := "https://global.transak.com"
		if env == "staging" {
			host = "https://staging-global.transak.com"
		}
		q := url.Values{}
		q.Set("apiKey", key)
		q.Set("fiatCurrency", fiatCurr)
		q.Set("cryptoCurrency", cryptoCurr)
		q.Set("fiatAmount", amount)
		q.Set("paymentMethod", payMethod)
		return host + "/?" + q.Encode()
	case "moonpay":
		key := s.getProviderKey("moonpay")
		if key == "" {
			return ""
		}
		host := "https://buy.moonpay.com"
		q := url.Values{}
		q.Set("apiKey", key)
		q.Set("baseCurrencyCode", fiatCurr)
		q.Set("currencyCode", cryptoCurr)
		q.Set("baseCurrencyAmount", amount)
		return host + "/?" + q.Encode()
	}
	return ""
}

// getProviderKey returns a provider's API key: the admin-configured runtime
// value (set via the admin panel) takes precedence over the env var.
func (s *FiatRampService) getProviderKey(providerID string) string {
	s.mu.RLock()
	if k, ok := s.providerKeys[providerID]; ok && k != "" {
		s.mu.RUnlock()
		return k
	}
	s.mu.RUnlock()
	switch providerID {
	case "transak":
		return os.Getenv("TRANSAK_API_KEY")
	case "moonpay":
		return os.Getenv("MOONPAY_API_KEY")
	}
	return ""
}

// SetProviderKey lets the admin panel configure a provider API key at runtime.
func (s *FiatRampService) SetProviderKey(providerID, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providerKeys[providerID] = key
}

// ClearProviderKey removes an admin-configured key (falls back to env).
func (s *FiatRampService) ClearProviderKey(providerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.providerKeys, providerID)
}

func (s *FiatRampService) GetProviders() []Provider {
	list := []Provider{}
	for _, p := range s.providers {
		list = append(list, p)
	}
	return list
}

func (s *FiatRampService) CreateOrder(userID, providerID, amount, fiatCurr, cryptoCurr, recipient, payMethod string) (*Order, error) {
	provider, ok := s.providers[providerID]
	if !ok {
		return nil, fmt.Errorf("provider not found")
	}

	amt, err := strconv.ParseFloat(amount, 64)
	if err != nil || amt <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	if amt < provider.MinAmount {
		return nil, fmt.Errorf("min amount is %v", provider.MinAmount)
	}
	if amt > provider.MaxAmount {
		return nil, fmt.Errorf("max amount is %v", provider.MaxAmount)
	}

	price, err := s.getCryptoPrice(cryptoCurr)
	if err != nil {
		return nil, fmt.Errorf("price unavailable for %s: %v", cryptoCurr, err)
	}
	cryptoAmt := amt / price
	feeAmt := amt * (provider.FeePercent / 100)
	networkFee := 0.50
	totalAmt := amt + feeAmt + networkFee

	order := &Order{
		ID:     "ramp-" + randomID(),
		UserID: userID, ProviderID: providerID, FiatAmount: amt, CryptoAmount: fmt.Sprintf("%.8f", cryptoAmt),
		FiatCurrency: fiatCurr, CryptoCurrency: cryptoCurr, ExchangeRate: price,
		FeeAmount: feeAmt, NetworkFee: networkFee, TotalAmount: totalAmt,
		RecipientAddress: recipient, Status: "pending", PaymentMethod: payMethod,
		KYCStatus: "pending", CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(),
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
		if o.UserID == userID {
			list = append(list, o)
		}
	}
	return list
}

// randomID returns a 16-char hex ID from a cryptographic RNG (never the
// deterministic i%len pattern that produced identical IDs on every call).
func randomID() string {
	b := make([]byte, 8)
	if _, err := cryptorand.Read(b); err != nil {
		// Fallback to time-based entropy if the CSPRNG fails (should not happen).
		binary.BigEndian.PutUint64(b, uint64(time.Now().UnixNano()))
	}
	return hex.EncodeToString(b)
}

func (s *FiatRampService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "fiat-ramp"}) })
	api := r.Group("/api/v1/ramp")
	api.GET("/providers", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"providers": s.GetProviders()}) })
	api.POST("/quote", func(c *gin.Context) {
		var req struct {
			ProviderID     string `json:"providerId" binding:"required"`
			Amount         string `json:"amount" binding:"required"`
			FiatCurrency   string `json:"fiatCurrency" binding:"required"`
			CryptoCurrency string `json:"cryptoCurrency" binding:"required"`
			PaymentMethod  string `json:"paymentMethod" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		quote, err := s.GetQuote(req.ProviderID, req.Amount, req.FiatCurrency, req.CryptoCurrency, req.PaymentMethod)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, quote)
	})
	// Off-ramp: sell crypto -> fiat. Returns the live fiat proceeds for a given
	// crypto amount (real CoinGecko price, never a hardcoded rate).
	api.POST("/offramp-quote", func(c *gin.Context) {
		var req struct {
			ProviderID     string `json:"providerId" binding:"required"`
			Amount         string `json:"amount" binding:"required"`
			FiatCurrency   string `json:"fiatCurrency" binding:"required"`
			CryptoCurrency string `json:"cryptoCurrency" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		provider, ok := s.providers[req.ProviderID]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provider not found"})
			return
		}
		cryptoAmt, err := strconv.ParseFloat(req.Amount, 64)
		if err != nil || cryptoAmt <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
			return
		}
		price, err := s.getCryptoPrice(req.CryptoCurrency)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("price unavailable: %v", err)})
			return
		}
		fiatGross := cryptoAmt * price
		feeAmt := fiatGross * (provider.FeePercent / 100)
		networkFee := 0.50
		fiatNet := fiatGross - feeAmt - networkFee
		c.JSON(http.StatusOK, gin.H{
			"providerId": provider.ID, "providerName": provider.Name,
			"cryptoAmount": req.Amount, "cryptoCurrency": req.CryptoCurrency,
			"fiatCurrency": req.FiatCurrency, "exchangeRate": price,
			"fiatGross": fiatGross, "feeAmount": feeAmt, "networkFee": networkFee,
			"fiatNet": fiatNet, "priceSource": "coingecko",
		})
	})
	api.POST("/order", s.authMiddleware(), func(c *gin.Context) {
		userID := c.GetString("user_id")
		var req struct {
			ProviderID       string `json:"providerId" binding:"required"`
			Amount           string `json:"amount" binding:"required"`
			FiatCurrency     string `json:"fiatCurrency" binding:"required"`
			CryptoCurrency   string `json:"cryptoCurrency" binding:"required"`
			RecipientAddress string `json:"recipientAddress" binding:"required"`
			PaymentMethod    string `json:"paymentMethod" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		order, err := s.CreateOrder(userID, req.ProviderID, req.Amount, req.FiatCurrency, req.CryptoCurrency, req.RecipientAddress, req.PaymentMethod)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, order)
	})
	api.GET("/order/:orderId", s.authMiddleware(), func(c *gin.Context) {
		order := s.GetOrder(c.Param("orderId"))
		if order == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, order)
	})
	api.GET("/orders/:userId", s.authMiddleware(), func(c *gin.Context) {
		// A user may only list their own orders.
		if c.Param("userId") != c.GetString("user_id") {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"orders": s.GetUserOrders(c.Param("userId"))})
	})

	// ---- Admin: configure provider API keys at runtime (admin panel) ----
	// GET  /admin/providers/:id/key  -> whether a key is configured (never the value)
	// POST /admin/providers/:id/key  -> set the key { "apiKey": "..." }
	// DELETE /admin/providers/:id/key -> clear the key (fall back to env)
	api.GET("/admin/providers/:id/key", s.adminMiddleware(), func(c *gin.Context) {
		pid := c.Param("id")
		if _, ok := s.providers[pid]; !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
			return
		}
		configured := s.getProviderKey(pid) != ""
		c.JSON(http.StatusOK, gin.H{"providerId": pid, "configured": configured})
	})
	api.POST("/admin/providers/:id/key", s.adminMiddleware(), func(c *gin.Context) {
		pid := c.Param("id")
		if _, ok := s.providers[pid]; !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
			return
		}
		var req struct {
			APIKey string `json:"apiKey" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		s.SetProviderKey(pid, req.APIKey)
		c.JSON(http.StatusOK, gin.H{"providerId": pid, "configured": true})
	})
	api.DELETE("/admin/providers/:id/key", s.adminMiddleware(), func(c *gin.Context) {
		pid := c.Param("id")
		if _, ok := s.providers[pid]; !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
			return
		}
		s.ClearProviderKey(pid)
		c.JSON(http.StatusOK, gin.H{"providerId": pid, "configured": s.getProviderKey(pid) != ""})
	})
}

// authMiddleware validates the Bearer JWT (HS256) issued by the canonical
// wallet_api using JWT_SECRET (must match wallet_api's secret). Rejects with
// 401 if absent/invalid — never trusts the client-supplied userId.
func (s *FiatRampService) authMiddleware() gin.HandlerFunc {
	secret := os.Getenv("JWT_SECRET")
	return func(c *gin.Context) {
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "JWT_SECRET not configured"})
			return
		}
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}
		uid, _ := claims["user_id"].(string)
		if uid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no user_id in token"})
			return
		}
		c.Set("user_id", uid)
		c.Next()
	}
}

// adminMiddleware validates the JWT AND requires the role claim to be "admin"
// (wallet_api RBAC: user|admin|wl_admin|master_wallet_admin). Rejects non-admins
// with 403 so only the admin panel can configure provider API keys.
func (s *FiatRampService) adminMiddleware() gin.HandlerFunc {
	secret := os.Getenv("JWT_SECRET")
	return func(c *gin.Context) {
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "JWT_SECRET not configured"})
			return
		}
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}
		role, _ := claims["role"].(string)
		if role != "admin" && role != "master_wallet_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin role required"})
			return
		}
		c.Next()
	}
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
