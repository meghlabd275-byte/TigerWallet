package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port     string
	RedisURL string
}

func LoadConfig() *Config {
	return &Config{
		Port:     getEnv("PORT", "8445"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Models
// ============================================================================

type GasPrice struct {
	ChainID     uint64  `json:"chainId"`
	ChainName   string  `json:"chainName"`
	Slow        string  `json:"slow"`
	Standard    string  `json:"standard"`
	Fast        string  `json:"fast"`
	SlowUSD     float64 `json:"slowUsd"`
	StandardUSD float64 `json:"standardUsd"`
	FastUSD     float64 `json:"fastUsd"`
	LastUpdated int64   `json:"lastUpdated"`
	BaseFee     string  `json:"baseFee,omitempty"`
	PriorityFee string  `json:"priorityFee,omitempty"`
}

type ChainConfig struct {
	ID          uint64
	Name        string
	Symbol      string
	RPCURLs     []string
	ExplorerURL string
	CoinGeckoID string
}

// ============================================================================
// Gas Oracle Service
// ============================================================================

type GasOracleService struct {
	config     *Config
	redis      *redis.Client
	chainInfo  map[uint64]ChainConfig
	httpClient *http.Client
	mu         sync.RWMutex
	prices     map[uint64]*GasPrice
	history    map[uint64][]GasPrice
}

func NewGasOracleService(config *Config) (*GasOracleService, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	redisClient.Ping(ctx)

	// Chain metadata only; gas prices come from each chain's live RPC and
	// native prices from CoinGecko. RPC URLs are env-overridable
	// (RPC_URL_<chainID>, comma-separated) and default to well-known public
	// endpoints. Nothing here is hardcoded market data.
	chainInfo := map[uint64]ChainConfig{
		1:     {ID: 1, Name: "Ethereum", Symbol: "ETH", CoinGeckoID: "ethereum", RPCURLs: rpcURLsFor(1, "https://cloudflare-eth.com")},
		137:   {ID: 137, Name: "Polygon", Symbol: "MATIC", CoinGeckoID: "matic-network", RPCURLs: rpcURLsFor(137, "https://polygon-rpc.com")},
		42161: {ID: 42161, Name: "Arbitrum One", Symbol: "ETH", CoinGeckoID: "ethereum", RPCURLs: rpcURLsFor(42161, "https://arb1.arbitrum.io/rpc")},
		10:    {ID: 10, Name: "Optimism", Symbol: "ETH", CoinGeckoID: "ethereum", RPCURLs: rpcURLsFor(10, "https://mainnet.optimism.io")},
		43114: {ID: 43114, Name: "Avalanche", Symbol: "AVAX", CoinGeckoID: "avalanche-2", RPCURLs: rpcURLsFor(43114, "https://api.avax.network/ext/bc/C/rpc")},
		56:    {ID: 56, Name: "BNB Chain", Symbol: "BNB", CoinGeckoID: "binancecoin", RPCURLs: rpcURLsFor(56, "https://bsc-dataseed.binance.org")},
		8453:  {ID: 8453, Name: "Base", Symbol: "ETH", CoinGeckoID: "ethereum", RPCURLs: rpcURLsFor(8453, "https://mainnet.base.org")},
	}

	return &GasOracleService{
		config:     config,
		redis:      redisClient,
		chainInfo:  chainInfo,
		httpClient: &http.Client{Timeout: 8 * time.Second},
		prices:     make(map[uint64]*GasPrice),
		history:    make(map[uint64][]GasPrice),
	}, nil
}

// rpcURLsFor returns the RPC endpoints for a chain: the RPC_URL_<chainID> env
// var (comma-separated) if set, otherwise the given public default.
func rpcURLsFor(chainID uint64, publicDefault string) []string {
	if v := os.Getenv(fmt.Sprintf("RPC_URL_%d", chainID)); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				out = append(out, t)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []string{publicDefault}
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// rpcCall performs a real JSON-RPC call against the chain's configured
// endpoints, trying each in order. Fail-closed: returns an error when no
// endpoint answers; the oracle never fabricates a result.
func (s *GasOracleService) rpcCall(chain ChainConfig, method string, params []interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", Method: method, Params: params, ID: 1})
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, url := range chain.RPCURLs {
		resp, err := s.httpClient.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		var rr rpcResponse
		derr := json.NewDecoder(resp.Body).Decode(&rr)
		resp.Body.Close()
		if derr != nil {
			lastErr = derr
			continue
		}
		if rr.Error != nil {
			lastErr = fmt.Errorf("rpc error %d: %s", rr.Error.Code, rr.Error.Message)
			continue
		}
		return rr.Result, nil
	}
	return nil, fmt.Errorf("all %d rpc endpoints failed for chain %d (%s): %v", len(chain.RPCURLs), chain.ID, chain.Name, lastErr)
}

// hexQuantityToBig decodes a 0x-prefixed JSON-RPC quantity.
func hexQuantityToBig(h string) (*big.Int, error) {
	h = strings.TrimPrefix(h, "0x")
	if h == "" {
		h = "0"
	}
	v, ok := new(big.Int).SetString(h, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex quantity")
	}
	return v, nil
}

// nativePriceUSD returns the chain's real native-token USD price from
// CoinGecko, cached in Redis for 60s (shared across replicas). Fail-closed:
// returns 0 when the upstream is unavailable; USD fields then reflect 0
// rather than a fabricated price.
func (s *GasOracleService) nativePriceUSD(coinID string) float64 {
	if coinID == "" {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cacheKey := "gasoracle:nativeprice:" + coinID
	if v, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		if f, perr := strconv.ParseFloat(v, 64); perr == nil {
			return f
		}
	}
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", coinID)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	var payload map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0
	}
	price := payload[coinID]["usd"]
	if price > 0 {
		s.redis.Set(ctx, cacheKey, strconv.FormatFloat(price, 'f', -1, 64), 60*time.Second)
	}
	return price
}


func (s *GasOracleService) FetchGasPrice(chainID uint64) (*GasPrice, error) {
	chain, ok := s.chainInfo[chainID]
	if !ok {
		return nil, fmt.Errorf("unsupported chain: %d", chainID)
	}

	// Short-lived Redis cache (15s) so a fleet of callers shares one upstream
	// fetch; on cache miss we query the chain's real RPC.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cacheKey := fmt.Sprintf("gasoracle:price:%d", chainID)
	if v, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		var cached GasPrice
		if json.Unmarshal([]byte(v), &cached) == nil {
			return &cached, nil
		}
	}

	// Real gas price from the chain node. Fail-closed: if the chain's RPC is
	// unreachable we return an error instead of a fabricated number.
	raw, err := s.rpcCall(chain, "eth_gasPrice", []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("gas price unavailable for chain %d: %w", chainID, err)
	}
	var gpHex string
	if err := json.Unmarshal(raw, &gpHex); err != nil {
		return nil, fmt.Errorf("invalid eth_gasPrice response for chain %d", chainID)
	}
	standard, err := hexQuantityToBig(gpHex)
	if err != nil {
		return nil, err
	}

	// EIP-1559 data when available (best-effort; legacy chains just use
	// eth_gasPrice for all tiers).
	var baseFee, priorityFee *big.Int
	if rawBF, err := s.rpcCall(chain, "eth_getBlockByNumber", []interface{}{"latest", false}); err == nil {
		var blk struct {
			BaseFeePerGas string `json:"baseFeePerGas"`
		}
		if json.Unmarshal(rawBF, &blk) == nil && blk.BaseFeePerGas != "" {
			baseFee, _ = hexQuantityToBig(blk.BaseFeePerGas)
		}
	}
	if rawPF, err := s.rpcCall(chain, "eth_maxPriorityFeePerGas", []interface{}{}); err == nil {
		var pfHex string
		if json.Unmarshal(rawPF, &pfHex) == nil {
			priorityFee, _ = hexQuantityToBig(pfHex)
		}
	}

	// Tiers derived from the real current price: slow = 90% of standard,
	// fast = base+tip or 125% of standard on legacy chains.
	slow := new(big.Int).Mul(standard, big.NewInt(90))
	slow.Div(slow, big.NewInt(100))
	fast := new(big.Int)
	if priorityFee != nil && priorityFee.Sign() > 0 {
		base := standard
		if baseFee != nil && baseFee.Sign() > 0 {
			base = new(big.Int).Add(baseFee, priorityFee)
		}
		fast.Add(base, priorityFee)
	} else {
		fast.Mul(standard, big.NewInt(125))
		fast.Div(fast, big.NewInt(100))
	}
	if slow.Sign() == 0 {
		slow.Set(standard)
	}

	nativePrice := s.nativePriceUSD(chain.CoinGeckoID)

	gasPrice := &GasPrice{
		ChainID:     chainID,
		ChainName:   chain.Name,
		Slow:        slow.String(),
		Standard:    standard.String(),
		Fast:        fast.String(),
		SlowUSD:     s.convertToUSD(slow.String(), nativePrice),
		StandardUSD: s.convertToUSD(standard.String(), nativePrice),
		FastUSD:     s.convertToUSD(fast.String(), nativePrice),
		LastUpdated: time.Now().Unix(),
	}
	if baseFee != nil {
		gasPrice.BaseFee = baseFee.String()
	}
	if priorityFee != nil {
		gasPrice.PriorityFee = priorityFee.String()
	}

	if b, err := json.Marshal(gasPrice); err == nil {
		s.redis.Set(ctx, cacheKey, b, 15*time.Second)
	}

	s.mu.Lock()
	s.prices[chainID] = gasPrice
	if len(s.history[chainID]) >= 100 {
		s.history[chainID] = s.history[chainID][1:]
	}
	s.history[chainID] = append(s.history[chainID], *gasPrice)
	s.mu.Unlock()

	return gasPrice, nil
}


func (s *GasOracleService) convertToUSD(gasPriceWei string, nativePrice float64) float64 {
	gasPrice, err := strconv.ParseFloat(gasPriceWei, 64)
	if err != nil {
		return 0
	}
	gasPriceGwei := gasPrice / 1e9
	usdCost := (gasPriceGwei * 21000 / 1e9) * nativePrice
	return math.Round(usdCost*100) / 100
}

func (s *GasOracleService) GetAllGasPrices() map[string]GasPrice {
	result := make(map[string]GasPrice)
	for chainID := range s.chainInfo {
		price, err := s.FetchGasPrice(chainID)
		if err != nil {
			continue
		}
		result[price.ChainName] = *price
	}
	return result
}

func (s *GasOracleService) GetRecommendation(chainID uint64) (map[string]GasPrice, error) {
	price, err := s.FetchGasPrice(chainID)
	if err != nil {
		return nil, err
	}

	slow := *price
	slow.Standard = price.Slow
	slow.StandardUSD = price.SlowUSD
	fast := *price
	fast.Standard = price.Fast
	fast.StandardUSD = price.FastUSD

	return map[string]GasPrice{
		"urgent": fast,
		"normal": *price,
		"slow":   slow,
	}, nil
}


type CostEstimate struct {
	ChainID      uint64  `json:"chainId"`
	Operation    string  `json:"operation"`
	GasLimit     uint64  `json:"gasLimit"`
	GasPrice     string  `json:"gasPrice"`
	TotalCost    string  `json:"totalCost"`
	TotalCostUSD float64 `json:"totalCostUsd"`
}

func (s *GasOracleService) EstimateCost(chainID uint64, operation string, gasLimit uint64) (*CostEstimate, error) {
	price, err := s.FetchGasPrice(chainID)
	if err != nil {
		return nil, err
	}

	gasLimits := map[string]uint64{
		"transfer": 21000, "swap": 200000, "approve": 46000,
		"nft_transfer": 85000, "stake": 100000, "bridge": 200000,
	}

	if gasLimit == 0 {
		gasLimit = gasLimits[operation]
		if gasLimit == 0 {
			gasLimit = 21000
		}
	}

	gasPriceWei, ok := new(big.Int).SetString(price.Standard, 10)
	if !ok {
		return nil, fmt.Errorf("invalid gas price for chain %d", chainID)
	}
	totalWei := new(big.Int).Mul(gasPriceWei, new(big.Int).SetUint64(gasLimit))

	chain := s.chainInfo[chainID]
	nativePrice := s.nativePriceUSD(chain.CoinGeckoID)
	totalNative, _ := new(big.Float).Quo(
		new(big.Float).SetInt(totalWei),
		big.NewFloat(1e18),
	).Float64()

	return &CostEstimate{
		ChainID:      chainID,
		Operation:    operation,
		GasLimit:     gasLimit,
		GasPrice:     price.Standard,
		TotalCost:    totalWei.String(),
		TotalCostUSD: math.Round(totalNative*nativePrice*100) / 100,
	}, nil
}


func (s *GasOracleService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "gas-oracle"})
	})

	api := r.Group("/api/v1")
	api.GET("/gas", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"prices": s.GetAllGasPrices()})
	})
	api.GET("/gas/:chainId", func(c *gin.Context) {
		chainID, _ := strconv.ParseUint(c.Param("chainId"), 10, 64)
		price, err := s.FetchGasPrice(chainID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, price)
	})
	api.GET("/gas/:chainId/recommend", func(c *gin.Context) {
		chainID, _ := strconv.ParseUint(c.Param("chainId"), 10, 64)
		rec, err := s.GetRecommendation(chainID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, rec)
	})
	api.POST("/gas/estimate", func(c *gin.Context) {
		var req struct {
			ChainID   uint64 `json:"chainId"`
			Operation string `json:"operation"`
			GasLimit  uint64 `json:"gasLimit"`
		}
		c.ShouldBindJSON(&req)
		est, err := s.EstimateCost(req.ChainID, req.Operation, req.GasLimit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, est)
	})
	api.GET("/gas/chains", func(c *gin.Context) {
		chains := make([]map[string]interface{}, 0)
		for id, chain := range s.chainInfo {
			chains = append(chains, map[string]interface{}{
				"id": id, "name": chain.Name, "symbol": chain.Symbol,
			})
		}
		c.JSON(http.StatusOK, gin.H{"chains": chains})
	})
}

func main() {
	config := LoadConfig()
	service, _ := NewGasOracleService(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	service.RegisterRoutes(r)

	srv := &http.Server{Addr: ":" + config.Port, Handler: r}
	go func() {
		log.Printf("Gas Oracle starting on port %s", config.Port)
		srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}
