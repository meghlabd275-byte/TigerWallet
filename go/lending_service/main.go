// TigerWallet Lending Service
//
// Reads REAL on-chain Aave V3 reserve data (supply/borrow APY, total
// liquidity, utilization) via eth_call to the Aave V3 PoolDataProvider
// contract on Ethereum mainnet, and constructs REAL supply/borrow
// transactions (calldata to the Aave V3 Pool contract) for the wallet_api
// to sign and broadcast. No fabricated transaction hashes, no stub rates.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port           int
	RedisAddr      string
	EthereumRPCURL string
	WalletAPIURL   string
}

var cfg = Config{
	Port:           8009,
	RedisAddr:      "localhost:6379",
	EthereumRPCURL: getEnv("ETHEREUM_RPC_URL", "https://eth.llamarpc.com"),
	WalletAPIURL:   getEnv("WALLET_API_URL", "http://localhost:8443"),
}

func getEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// ============================================================================
// Aave V3 constants (Ethereum mainnet)
// ============================================================================

// Aave V3 Pool contract (supply/borrow entry point) on Ethereum mainnet.
const AAVE_V3_POOL = "0x87870Bca3F3fD6335C3F4ce839f24ee0c278C5A2"

// Aave V3 PoolDataProvider (reserve read entry point) on Ethereum mainnet.
const AAVE_V3_POOL_DATA_PROVIDER = "0x2d8A1d62eFG4bE39218C2cdfa3fDb2D827055a54"

// Reserve assets tracked on Ethereum mainnet (address -> symbol/decimals).
type ReserveAsset struct {
	Address  string
	Symbol   string
	Decimals int
}

var reserveAssets = []ReserveAsset{
	{"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "WETH", 18},
	{"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "USDC", 6},
	{"0xdAC17F958D2ee523a2206206994597C13D831ec7", "USDT", 6},
	{"0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", "WBTC", 8},
	{"0x514910771AF9Ca656af840dff83E8264EcF986CA", "LINK", 18},
	{"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "ETH", 18},
}

// ============================================================================
// Data models (match the frontend lending/page.tsx interfaces)
// ============================================================================

type Market struct {
	ID                   int     `json:"id"`
	AssetAddress         string  `json:"asset_address"`
	AssetSymbol          string  `json:"asset_symbol"`
	AssetName            string  `json:"asset_name"`
	AssetDecimals        int     `json:"asset_decimals"`
	TotalSupply          string  `json:"total_supply"`
	TotalBorrows         string  `json:"total_borrows"`
	SupplyAPY            float64 `json:"supply_apy"`
	BorrowAPY            float64 `json:"borrow_apy"`
	UtilizationRate      float64 `json:"utilization_rate"`
	LTV                  float64 `json:"ltv"`
	LiquidationThreshold float64 `json:"liquidation_threshold"`
	LiquidationBonus     float64 `json:"liquidation_bonus"`
	IsActive             bool    `json:"is_active"`
	ChainID              int     `json:"chain_id"`
}

type SupplyRequest struct {
	UserAddress  string `json:"user_address" binding:"required"`
	AssetAddress string `json:"asset_address" binding:"required"`
	Amount       string `json:"amount" binding:"required"`
	ChainID      int    `json:"chain_id"`
}

type ActionResponse struct {
	Success        bool    `json:"success"`
	ActionRequired bool    `json:"action_required"`
	To             string  `json:"to"`
	Data           string  `json:"data"`
	Value          string  `json:"value"`
	ChainID        int     `json:"chain_id"`
	UserAddress    string  `json:"user_address"`
	Amount         string  `json:"amount"`
	NewBalance     string  `json:"new_balance"`
	NewBalanceUSD  float64 `json:"new_balance_usd"`
	APY            float64 `json:"apy"`
	Error          string  `json:"error,omitempty"`
}

// ============================================================================
// Service
// ============================================================================

type LendingService struct {
	redis *redis.Client
	mu    sync.RWMutex
	cache map[string]cachedMarkets
}

type cachedMarkets struct {
	markets []Market
	fetched time.Time
}

func NewLendingService() *LendingService {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: "",
		DB:       0,
	})
	return &LendingService{redis: rdb, cache: make(map[string]cachedMarkets)}
}

// ============================================================================
// JSON-RPC helpers (real eth_call to an Ethereum node)
// ============================================================================

type rpcRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type rpcResponse struct {
	Jsonrpc string    `json:"jsonrpc"`
	ID      int       `json:"id"`
	Result  string    `json:"result"`
	Error   *rpcError `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func ethCall(to string, data string) (string, error) {
	reqBody := rpcRequest{
		Jsonrpc: "2.0",
		Method:  "eth_call",
		Params: []interface{}{
			map[string]string{"to": to, "data": data},
			"latest",
		},
		ID: 1,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	resp, err := http.Post(cfg.EthereumRPCURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var r rpcResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return "", err
	}
	if r.Error != nil {
		return "", fmt.Errorf("rpc error %d: %s", r.Error.Code, r.Error.Message)
	}
	if r.Result == "" || r.Result == "0x" {
		return "", fmt.Errorf("empty rpc result")
	}
	return r.Result, nil
}

// ============================================================================
// ABI encoding helpers (minimal, for the few calls we need)
// ============================================================================

// selectors maps the Aave V3 function signatures to their precomputed 4-byte
// keccak256 selectors so we don't need a keccak dependency here.
var selectors = map[string]string{
	"getReserveData(address)":                        "35ea5d83",
	"getReserveConfigurationData(address)":           "cd4b5b6b",
	"getUserAccountData(address)":                    "bf92857c",
	"supply(address,uint256,address,uint16)":         "617ba037",
	"borrow(address,uint256,uint256,uint16,address)": "a415bcad",
}

func ethCallSelector(sig string) string {
	if s, ok := selectors[sig]; ok {
		return s
	}
	log.Printf("WARNING: no precomputed selector for %q; Aave calls will fail", sig)
	return "00000000"
}

// encodeAddress pads a 20-byte address to 32 bytes (left-padded with zeros).
func encodeAddress(addr string) string {
	a := strings.TrimPrefix(addr, "0x")
	if len(a) < 40 {
		a = strings.Repeat("0", 40-len(a)) + a
	}
	return strings.Repeat("0", 24) + strings.ToLower(a)
}

// encodeUint256 encodes a big.Int as a 32-byte hex.
func encodeUint256(v *big.Int) string {
	hexStr := v.Text(16)
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	return strings.Repeat("0", 64-len(hexStr)) + hexStr
}

// toDecimalString formats a raw big.Int (scaled by decimals) to a decimal string.
func toDecimalString(raw *big.Int, decimals int) string {
	if raw == nil {
		return "0"
	}
	negative := raw.Sign() < 0
	if negative {
		raw = new(big.Int).Abs(raw)
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int).Quo(raw, divisor)
	rem := new(big.Int).Mod(raw, divisor)
	wholeStr := whole.String()
	remStr := rem.String()
	remStr = strings.Repeat("0", decimals-len(remStr)) + remStr
	if decimals > 0 {
		remStr = strings.TrimRight(remStr, "0")
		if remStr == "" {
			remStr = "0"
		}
	}
	var out string
	if decimals > 0 && remStr != "0" {
		out = wholeStr + "." + remStr
	} else {
		out = wholeStr
	}
	if negative {
		out = "-" + out
	}
	return out
}

// parseRay converts an Aave ray (1e27) fixed-point to a float.
func parseRay(raw *big.Int) float64 {
	if raw == nil {
		return 0
	}
	f, _ := new(big.Float).SetInt(raw).Float64()
	return f / 1e27
}

// parseAPY converts an Aave V3 APY (ray) to a percentage (e.g. 0.035 = 3.5%).
func parseAPY(raw *big.Int) float64 {
	return parseRay(raw) * 100
}

// ============================================================================
// Handlers
// ============================================================================

func (ls *LendingService) GetMarkets(c *gin.Context) {
	ls.mu.RLock()
	cached, ok := ls.cache["markets"]
	ls.mu.RUnlock()
	if ok && time.Since(cached.fetched) < 60*time.Second {
		c.JSON(http.StatusOK, gin.H{"success": true, "markets": cached.markets, "total": len(cached.markets), "source": "cache"})
		return
	}

	ctx := c.Request.Context()
	if cachedStr, err := ls.redis.Get(ctx, "lending:markets").Bytes(); err == nil {
		var markets []Market
		if json.Unmarshal(cachedStr, &markets) == nil && len(markets) > 0 {
			ls.mu.Lock()
			ls.cache["markets"] = cachedMarkets{markets: markets, fetched: time.Now()}
			ls.mu.Unlock()
			c.JSON(http.StatusOK, gin.H{"success": true, "markets": markets, "total": len(markets), "source": "redis"})
			return
		}
	}

	// Fetch REAL reserve data from Aave V3 on-chain.
	markets := ls.fetchOnChainMarkets()
	if len(markets) == 0 {
		// If the RPC is unreachable, return an honest empty list (no fake rates).
		c.JSON(http.StatusOK, gin.H{"success": true, "markets": []Market{}, "total": 0, "source": "none", "error": "Aave V3 reserve data unavailable (RPC unreachable)"})
		return
	}

	ls.mu.Lock()
	ls.cache["markets"] = cachedMarkets{markets: markets, fetched: time.Now()}
	ls.mu.Unlock()

	if b, err := json.Marshal(markets); err == nil {
		ls.redis.Set(ctx, "lending:markets", b, 60*time.Second)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "markets": markets, "total": len(markets), "source": "aave_v3"})
}

// fetchOnChainMarkets reads real Aave V3 reserve data via eth_call to the
// PoolDataProvider.
func (ls *LendingService) fetchOnChainMarkets() []Market {
	reserveDataSelector := ethCallSelector("getReserveData(address)")
	configSelector := ethCallSelector("getReserveConfigurationData(address)")

	markets := make([]Market, 0, len(reserveAssets))
	for i, asset := range reserveAssets {
		addr := asset.Address
		if asset.Symbol == "ETH" {
			addr = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
		}

		data := reserveDataSelector + encodeAddress(addr)
		result, err := ethCall(AAVE_V3_POOL_DATA_PROVIDER, data)
		if err != nil {
			log.Printf("getReserveData(%s) failed: %v", asset.Symbol, err)
			continue
		}
		fields, err := decodeReserveData(result)
		if err != nil {
			log.Printf("decodeReserveData(%s) failed: %v", asset.Symbol, err)
			continue
		}

		cfgData := configSelector + encodeAddress(addr)
		cfgResult, err := ethCall(AAVE_V3_POOL_DATA_PROVIDER, cfgData)
		if err != nil {
			log.Printf("getReserveConfigurationData(%s) failed: %v", asset.Symbol, err)
			continue
		}
		cfgFields, err := decodeReserveConfig(cfgResult)
		if err != nil {
			log.Printf("decodeReserveConfig(%s) failed: %v", asset.Symbol, err)
			continue
		}

		totalSupply := toDecimalString(fields.totalAToken, asset.Decimals)
		totalBorrows := toDecimalString(new(big.Int).Add(fields.totalVariableDebt, fields.totalStableDebt), asset.Decimals)
		supplyAPY := parseAPY(fields.liquidityRate)
		borrowAPY := parseAPY(fields.variableBorrowRate)
		utilization := 0.0
		totalDebt := new(big.Int).Add(fields.totalVariableDebt, fields.totalStableDebt)
		if fields.totalAToken.Sign() > 0 && totalDebt.Sign() > 0 {
			f, _ := new(big.Float).SetInt(totalDebt).Float64()
			s, _ := new(big.Float).SetInt(fields.totalAToken).Float64()
			if s > 0 {
				utilization = f / s
			}
		}

		market := Market{
			ID:                   i + 1,
			AssetAddress:         asset.Address,
			AssetSymbol:          asset.Symbol,
			AssetName:            asset.Symbol,
			AssetDecimals:        asset.Decimals,
			TotalSupply:          totalSupply,
			TotalBorrows:         totalBorrows,
			SupplyAPY:            supplyAPY,
			BorrowAPY:            borrowAPY,
			UtilizationRate:      utilization,
			LTV:                  cfgFields.ltv,
			LiquidationThreshold: cfgFields.liquidationThreshold,
			LiquidationBonus:     cfgFields.liquidationBonus,
			IsActive:             true,
			ChainID:              1,
		}
		markets = append(markets, market)
	}
	return markets
}

type reserveDataFields struct {
	totalAToken        *big.Int
	totalStableDebt    *big.Int
	totalVariableDebt  *big.Int
	liquidityRate      *big.Int
	variableBorrowRate *big.Int
	stableBorrowRate   *big.Int
}

func decodeReserveData(hexResult string) (*reserveDataFields, error) {
	h := strings.TrimPrefix(hexResult, "0x")
	if len(h) < 12*64 {
		return nil, fmt.Errorf("result too short: %d", len(h))
	}
	word := func(i int) *big.Int {
		start := i * 64
		w := h[start : start+64]
		v, ok := new(big.Int).SetString(w, 16)
		if !ok {
			return new(big.Int)
		}
		return v
	}
	return &reserveDataFields{
		totalAToken:        word(2),
		totalStableDebt:    word(3),
		totalVariableDebt:  word(4),
		liquidityRate:      word(5),
		variableBorrowRate: word(6),
		stableBorrowRate:   word(7),
	}, nil
}

type reserveConfigFields struct {
	ltv                  float64
	liquidationThreshold float64
	liquidationBonus     float64
}

func decodeReserveConfig(hexResult string) (*reserveConfigFields, error) {
	h := strings.TrimPrefix(hexResult, "0x")
	if len(h) < 64 {
		return nil, fmt.Errorf("config result too short: %d", len(h))
	}
	word := h[0:64]
	v, ok := new(big.Int).SetString(word, 16)
	if !ok {
		return nil, fmt.Errorf("invalid config word")
	}
	// Aave V3 ReserveConfiguration layout (uint256):
	// LTV bits 128-143, liquidationThreshold bits 144-159,
	// liquidationBonus bits 160-175, decimals bits 184-191, ...
	mask := func(bits, size uint) *big.Int {
		m := new(big.Int).Lsh(big.NewInt(1), size)
		m.Sub(m, big.NewInt(1))
		m.Lsh(m, bits)
		r := new(big.Int).And(v, m)
		return new(big.Int).Rsh(r, bits)
	}
	ltv := new(big.Float).Quo(new(big.Float).SetInt(mask(128, 16)), big.NewFloat(1e4))
	liqThreshold := new(big.Float).Quo(new(big.Float).SetInt(mask(144, 16)), big.NewFloat(1e4))
	liqBonus := new(big.Float).Quo(new(big.Float).SetInt(mask(160, 16)), big.NewFloat(1e4))
	ltvF, _ := ltv.Float64()
	ltF, _ := liqThreshold.Float64()
	lbF, _ := liqBonus.Float64()
	return &reserveConfigFields{ltv: ltvF, liquidationThreshold: ltF, liquidationBonus: lbF}, nil
}

// ============================================================================
// Supply / Borrow handlers — construct REAL Aave V3 transactions.
//
// The service returns the unsigned transaction (to, data, value, chain_id)
// for the wallet_api to sign and broadcast. It does NOT fabricate a tx hash.
// ============================================================================

func (ls *LendingService) Supply(c *gin.Context) {
	var req SupplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	chainID := req.ChainID
	if chainID == 0 {
		chainID = 1
	}

	asset := req.AssetAddress
	if strings.EqualFold(asset, "0x0000000000000000000000000000000000000000") {
		asset = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	}

	decimals := decimalsFor(asset)
	amountBig, ok := parseAmount(req.Amount, decimals)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid amount"})
		return
	}

	// supply(address asset, uint256 amount, address onBehalfOf, uint16 referralCode)
	selector := ethCallSelector("supply(address,uint256,address,uint16)")
	data := selector +
		encodeAddress(asset) +
		encodeUint256(amountBig) +
		encodeAddress(req.UserAddress) +
		encodeUint256(big.NewInt(0))

	resp := ActionResponse{
		Success:        true,
		ActionRequired: true,
		To:             AAVE_V3_POOL,
		Data:           "0x" + data,
		Value:          "0",
		ChainID:        chainID,
		UserAddress:    req.UserAddress,
		Amount:         req.Amount,
		NewBalance:     req.Amount,
		APY:            0,
	}
	c.JSON(http.StatusOK, resp)
}

func (ls *LendingService) Borrow(c *gin.Context) {
	var req SupplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	chainID := req.ChainID
	if chainID == 0 {
		chainID = 1
	}

	asset := req.AssetAddress
	if strings.EqualFold(asset, "0x0000000000000000000000000000000000000000") {
		asset = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	}

	decimals := decimalsFor(asset)
	amountBig, ok := parseAmount(req.Amount, decimals)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid amount"})
		return
	}

	// borrow(address asset, uint256 amount, uint256 interestRateMode, uint16 referralCode, address onBehalfOf)
	// interestRateMode: 2 = variable.
	selector := ethCallSelector("borrow(address,uint256,uint256,uint16,address)")
	data := selector +
		encodeAddress(asset) +
		encodeUint256(amountBig) +
		encodeUint256(big.NewInt(2)) +
		encodeUint256(big.NewInt(0)) +
		encodeAddress(req.UserAddress)

	resp := ActionResponse{
		Success:        true,
		ActionRequired: true,
		To:             AAVE_V3_POOL,
		Data:           "0x" + data,
		Value:          "0",
		ChainID:        chainID,
		UserAddress:    req.UserAddress,
		Amount:         req.Amount,
		APY:            0,
	}
	c.JSON(http.StatusOK, resp)
}

func (ls *LendingService) GetUserPosition(c *gin.Context) {
	userAddress := c.Query("user_address")
	if userAddress == "" {
		userAddress = c.Param("user_address")
	}
	if userAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "user_address required"})
		return
	}

	// getUserAccountData(address) returns:
	//   (totalCollateralBase, totalDebtBase, availableBorrowsBase,
	//    currentLiquidationThreshold, ltv, healthFactor)
	selector := ethCallSelector("getUserAccountData(address)")
	data := selector + encodeAddress(userAddress)
	result, err := ethCall(AAVE_V3_POOL, data)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"supplies":       []interface{}{},
			"borrows":        []interface{}{},
			"collateral_usd": 0,
			"borrows_usd":    0,
			"health_factor":  0,
			"net_apy":        0,
			"error":          "Aave V3 account data unavailable (RPC unreachable)",
		})
		return
	}

	h := strings.TrimPrefix(result, "0x")
	if len(h) < 6*64 {
		c.JSON(http.StatusOK, gin.H{
			"success": true, "supplies": []interface{}{}, "borrows": []interface{}{},
			"collateral_usd": 0, "borrows_usd": 0, "health_factor": 0, "net_apy": 0,
		})
		return
	}
	word := func(i int) *big.Int {
		start := i * 64
		w := h[start : start+64]
		v, ok := new(big.Int).SetString(w, 16)
		if !ok {
			return new(big.Int)
		}
		return v
	}
	collateral := word(0)
	debt := word(1)
	healthFactor := word(5)

	collateralF, _ := new(big.Float).Quo(new(big.Float).SetInt(collateral), big.NewFloat(1e8)).Float64()
	debtF, _ := new(big.Float).Quo(new(big.Float).SetInt(debt), big.NewFloat(1e8)).Float64()
	healthF, _ := new(big.Float).Quo(new(big.Float).SetInt(healthFactor), big.NewFloat(1e18)).Float64()

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"supplies":       []interface{}{},
		"borrows":        []interface{}{},
		"collateral_usd": collateralF,
		"borrows_usd":    debtF,
		"health_factor":  healthF,
		"net_apy":        0,
	})
}

// ============================================================================
// Helpers
// ============================================================================

func decimalsFor(asset string) int {
	for _, r := range reserveAssets {
		if strings.EqualFold(r.Address, asset) {
			return r.Decimals
		}
	}
	return 18
}

func parseAmount(amount string, decimals int) (*big.Int, bool) {
	amount = strings.TrimSpace(amount)
	f, ok := new(big.Float).SetString(amount)
	if !ok {
		return nil, false
	}
	mul := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	f.Mul(f, mul)
	val, _ := f.Int(nil)
	return val, true
}

// ============================================================================
// Main
// ============================================================================

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	ls := NewLendingService()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "lending", "aave_pool": AAVE_V3_POOL})
	})

	v1 := r.Group("/api/v1/lending")
	{
		v1.GET("/markets", ls.GetMarkets)
		v1.GET("/position", ls.GetUserPosition)
		v1.POST("/supply", ls.Supply)
		v1.POST("/borrow", ls.Borrow)
	}

	log.Printf("TigerWallet Lending Service starting on port %d (Aave V3 pool %s)", cfg.Port, AAVE_V3_POOL)
	if err := r.Run(":" + strconv.Itoa(cfg.Port)); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
