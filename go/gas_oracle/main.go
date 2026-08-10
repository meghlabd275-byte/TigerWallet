package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
	NativePrice float64
}

// ============================================================================
// Gas Oracle Service
// ============================================================================

type GasOracleService struct {
	config    *Config
	redis     *redis.Client
	chainInfo map[uint64]ChainConfig
	mu        sync.RWMutex
	prices    map[uint64]*GasPrice
	history   map[uint64][]GasPrice
}

func NewGasOracleService(config *Config) (*GasOracleService, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	redisClient.Ping(ctx)

	chainInfo := map[uint64]ChainConfig{
		1:     {ID: 1, Name: "Ethereum", Symbol: "ETH", NativePrice: 2500.0},
		137:   {ID: 137, Name: "Polygon", Symbol: "MATIC", NativePrice: 0.85},
		42161: {ID: 42161, Name: "Arbitrum One", Symbol: "ETH", NativePrice: 2500.0},
		10:    {ID: 10, Name: "Optimism", Symbol: "ETH", NativePrice: 2500.0},
		43114: {ID: 43114, Name: "Avalanche", Symbol: "AVAX", NativePrice: 35.0},
		56:    {ID: 56, Name: "BNB Chain", Symbol: "BNB", NativePrice: 300.0},
		8453:  {ID: 8453, Name: "Base", Symbol: "ETH", NativePrice: 2500.0},
	}

	return &GasOracleService{
		config:    config,
		redis:     redisClient,
		chainInfo: chainInfo,
		prices:    make(map[uint64]*GasPrice),
		history:   make(map[uint64][]GasPrice),
	}, nil
}

func (s *GasOracleService) FetchGasPrice(chainID uint64) (*GasPrice, error) {
	chain, ok := s.chainInfo[chainID]
	if !ok {
		return nil, fmt.Errorf("unsupported chain: %d", chainID)
	}

	hardcodedPrices := map[uint64]map[string]string{
		1:     {"slow": "20000000000", "standard": "25000000000", "fast": "40000000000"},
		137:   {"slow": "30000000000", "standard": "40000000000", "fast": "60000000000"},
		42161: {"slow": "100000", "standard": "150000", "fast": "200000"},
		10:    {"slow": "1000000", "standard": "2000000", "fast": "5000000"},
		43114: {"slow": "25000000000", "standard": "30000000000", "fast": "50000000000"},
		56:    {"slow": "3000000000", "standard": "4000000000", "fast": "6000000000"},
		8453:  {"slow": "100000", "standard": "150000", "fast": "200000"},
	}

	prices, ok := hardcodedPrices[chainID]
	if !ok {
		prices = hardcodedPrices[1]
	}

	gasPrice := &GasPrice{
		ChainID:     chainID,
		ChainName:   chain.Name,
		Slow:        prices["slow"],
		Standard:    prices["standard"],
		Fast:        prices["fast"],
		SlowUSD:     s.convertToUSD(prices["slow"], chain.NativePrice),
		StandardUSD: s.convertToUSD(prices["standard"], chain.NativePrice),
		FastUSD:     s.convertToUSD(prices["fast"], chain.NativePrice),
		LastUpdated: time.Now().Unix(),
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

	return map[string]GasPrice{
		"urgent": *price,
		"normal": *price,
		"slow":   *price,
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

	gasPriceWei, _ := strconv.ParseInt(price.Standard, 10, 64)
	totalWei := gasPriceWei * int64(gasLimit)

	return &CostEstimate{
		ChainID:      chainID,
		Operation:    operation,
		GasLimit:     gasLimit,
		GasPrice:     price.Standard,
		TotalCost:    strconv.FormatInt(totalWei, 10),
		TotalCostUSD: math.Round(float64(totalWei)/1e18*2500*100) / 100,
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
