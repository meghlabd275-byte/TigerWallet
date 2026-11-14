package main

// Real on-chain AMM router for Uniswap-V2-compatible DEXes.
//
// Unlike the indicative CoinGecko cross-rate in defi_handlers.go, this module
// performs REAL on-chain reads:
//   - getAmountsOut(uint256, address[]) via eth_call to the V2 Router02, which
//     walks the actual AMM reserves and returns the exact output amount for a
//     given path (no fabricated prices).
//   - Constructs swapExactTokensForTokens calldata so the client can submit the
//     swap through the real /api/v1/send (eth_sendRawTransaction).
//
// No fabricated quotes, no hardcoded reserves. If the RPC or router is
// unavailable, the endpoint returns 503 — it never invents a number.

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

// V2-compatible router per chain. Override via env CHAIN_<id>_ROUTER to point at
// a different deployment (PancakeSwap, SushiSwap, QuickSwap, Camelot and
// Uniswap V2 all share the getAmountsOut / swapExactTokensForTokens interface).
//
// Every address below was checked with eth_getCode and a live
// getAmountsOut(WETH -> USDC) eth_call on the chain it is listed for. A router
// that holds no code makes getAmountsOut return 0x, so the caller would fail
// with "could not decode router response" on every swap. Ethereum, Polygon,
// Arbitrum, Optimism and Base previously pointed at addresses with no code.
var v2Routers = map[int64]string{
	1:        "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D", // Uniswap V2 Router02 (Ethereum)
	56:       "0x10ED43C718714eb63d5aA57B78B54704E256024E", // PancakeSwap V2 (BSC)
	137:      "0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff", // QuickSwap V2 (Polygon)
	42161:    "0xc873fEcbd354f5A56E00E710B90EF4201db2448d", // Camelot V2 (Arbitrum)
	10:       "0x2ABf469074dc0b54d793850807E6eb5Faf2625b1", // SushiSwap V2 (Optimism)
	8453:     "0x4752ba5DBc23f44D87826276BF6Fd6b1C372aD24", // Uniswap V2 Router02 (Base)
	11155111: "0xC532a74256D3Db42D0Bf7a0400fEFDbad7694008", // Uniswap V2 Router02 (Sepolia)
}

// routerForChain resolves the V2 router for a chain.
//
// CHAIN_<id>_ROUTER takes precedence over the built-in table. That is what lets
// an operator point at a TigerSwap deployment on a chain that has no public
// router listed here - notably TigerSmartChain, where the router address is
// chosen by whoever deploys the factory.
//
// A chain with no router resolves to the zero address, and every caller treats
// that as "AMM unavailable" and fails closed. It never falls back to a guessed
// address.
func routerForChain(chainID int64) common.Address {
	if override := strings.TrimSpace(os.Getenv(fmt.Sprintf("CHAIN_%d_ROUTER", chainID))); override != "" {
		if common.IsHexAddress(override) {
			return common.HexToAddress(override)
		}
		return common.Address{}
	}
	if r, ok := v2Routers[chainID]; ok {
		return common.HexToAddress(r)
	}
	return common.Address{}
}

// ---- ABI calldata builders (manual, no abigen dependency) ----

// getAmountsOut(uint256 amountIn, address[] path) selector = 0xd06ca61f
func getAmountsOutData(amountIn *big.Int, path []common.Address) []byte {
	data := make([]byte, 0, 4+32+32+32*len(path))
	data = append(data, 0xd0, 0x6c, 0xa6, 0x1f)
	// amountIn (offset 4)
	data = append(data, common.LeftPadBytes(amountIn.Bytes(), 32)...)
	// path is dynamic -> offset = 0x40 (64 bytes after amountIn+length slot)
	offset := big.NewInt(64)
	data = append(data, common.LeftPadBytes(offset.Bytes(), 32)...)
	// path.length
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(path))).Bytes(), 32)...)
	for _, p := range path {
		data = append(data, common.LeftPadBytes(p.Bytes(), 32)...)
	}
	return data
}

// swapExactTokensForTokens(uint256 amountIn, uint256 amountOutMin, address[] path, address to, uint256 deadline)
// selector = 0x18cbafe5
func swapExactTokensForTokensData(amountIn, amountOutMin *big.Int, path []common.Address, to common.Address, deadline *big.Int) []byte {
	data := make([]byte, 0, 4+32*6+32*len(path))
	data = append(data, 0x18, 0xcb, 0xaf, 0xe5)
	data = append(data, common.LeftPadBytes(amountIn.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amountOutMin.Bytes(), 32)...)
	// path offset: 5 fixed args * 32 = 0xa0 (160)
	pathOffset := big.NewInt(160)
	data = append(data, common.LeftPadBytes(pathOffset.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(to.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(deadline.Bytes(), 32)...)
	// path.length
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(path))).Bytes(), 32)...)
	for _, p := range path {
		data = append(data, common.LeftPadBytes(p.Bytes(), 32)...)
	}
	return data
}

// decodeAmountsOut parses the (uint256[]) return of getAmountsOut. The dynamic
// array encoding: [offset][length][elem0][elem1]...
func decodeAmountsOut(ret []byte) ([]*big.Int, error) {
	if len(ret) < 64 {
		return nil, fmt.Errorf("router returned %d bytes (expected >= 64)", len(ret))
	}
	length := new(big.Int).SetBytes(ret[32:64]).Int64()
	if length < 2 {
		return nil, fmt.Errorf("router returned path length %d", length)
	}
	if int(length) > 64 {
		return nil, fmt.Errorf("implausible path length %d", length)
	}
	out := make([]*big.Int, 0, length)
	for i := int64(0); i < length; i++ {
		off := 64 + int(i)*32
		if off+32 > len(ret) {
			return nil, fmt.Errorf("truncated amounts at index %d", i)
		}
		out = append(out, new(big.Int).SetBytes(ret[off:off+32]))
	}
	return out, nil
}

// ---- HTTP handlers ----

// GET /api/v1/amm/quote?chain_id=1&token_in=0x..&token_out=0x..&amount_in=<human>
// Real on-chain quote via getAmountsOut. amount_in is in human units; converted
// to wei using the token's on-chain decimals (another real eth_call).
func handleAmmQuote(c *gin.Context) {
	if !enforceFeature(c, FeatureSwapTrading) {
		return
	}
	chainIDStr := c.Query("chain_id")
	tokenIn := c.Query("token_in")
	tokenOut := c.Query("token_out")
	amountIn := c.Query("amount_in")
	if chainIDStr == "" || tokenIn == "" || tokenOut == "" || amountIn == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id, token_in, token_out, amount_in are required"})
		return
	}
	var chainID int64
	if _, err := fmt.Sscan(chainIDStr, &chainID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	// Trading control-plane: liquidity vertical halt + explicit pool stop.
	if !tradingGuard(c, "liquidity", "", "") {
		return
	}
	if tradingEntityStoppedRedis(c.Request.Context(), "pool", poolRedisKey(chainID, tokenIn, tokenOut)) ||
		tradingEntityStoppedRedis(c.Request.Context(), "pool", poolRedisKey(chainID, tokenOut, tokenIn)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "liquidity pool is stopped by the platform operator"})
		return
	}
	chain := evmChainByChainID(chainID)
	if chain == nil || chain.RPCEndpoint == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "unsupported chain_id"})
		return
	}
	router := routerForChain(chainID)
	if router == (common.Address{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": "no V2 router configured for this chain"})
		return
	}

	if !common.IsHexAddress(tokenIn) || !common.IsHexAddress(tokenOut) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token_in/token_out must be 0x addresses"})
		return
	}
	addrIn := common.HexToAddress(tokenIn)
	addrOut := common.HexToAddress(tokenOut)

	amountHuman, ok := new(big.Float).SetString(amountIn)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount_in"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), httpTimeout)
	defer cancel()

	// Real on-chain decimals for each token (so we convert human->wei correctly).
	decIn, err := fetchDecimals(ctx, chain.RPCEndpoint, addrIn)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not fetch token_in decimals: " + err.Error()})
		return
	}
	decOut, err := fetchDecimals(ctx, chain.RPCEndpoint, addrOut)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not fetch token_out decimals: " + err.Error()})
		return
	}

	amountInWei, err := humanToWei(amountHuman, decIn)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	path := []common.Address{addrIn, addrOut}
	ret, err := ethCall(ctx, chain.RPCEndpoint, router, getAmountsOutData(amountInWei, path))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "router eth_call failed: " + err.Error(), "router": router.Hex()})
		return
	}
	amounts, err := decodeAmountsOut(ret)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not decode router response: " + err.Error()})
		return
	}
	amountOutWei := amounts[len(amounts)-1]

	outHuman, _ := weiToHuman(amountOutWei, decOut)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"quote_type":     "on-chain",
		"chain_id":       chainID,
		"router":         router.Hex(),
		"token_in":       addrIn.Hex(),
		"token_out":      addrOut.Hex(),
		"amount_in":      amountIn,
		"amount_out":     outHuman,
		"amount_out_wei": amountOutWei.String(),
		"amount_in_wei":  amountInWei.String(),
		"decimals_in":    decIn,
		"decimals_out":   decOut,
		"path":           []string{addrIn.Hex(), addrOut.Hex()},
		"raw_return":     "0x" + hex.EncodeToString(ret),
	})
}

// POST /api/v1/amm/swap  — constructs swapExactTokensForTokens calldata. The
// client broadcasts it via POST /api/v1/send (real eth_sendRawTransaction).
// No transaction hash is fabricated here.
func handleAmmSwap(c *gin.Context) {
	if !enforceFeature(c, FeatureSwapTrading) {
		return
	}
	var req struct {
		From         string `json:"from"`
		ChainID      int64  `json:"chain_id"`
		TokenIn      string `json:"token_in"`
		TokenOut     string `json:"token_out"`
		AmountIn     string `json:"amount_in"`
		AmountOutMin string `json:"amount_out_min"` // human units
		Slippage     string `json:"slippage"`       // percent, used when amount_out_min is absent
		Recipient    string `json:"recipient"`      // optional; defaults to From
		DeadlineSec  int64  `json:"deadline_sec"`   // optional; default 20 min
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.From == "" || req.ChainID == 0 || req.TokenIn == "" || req.TokenOut == "" || req.AmountIn == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from, chain_id, token_in, token_out, amount_in are required"})
		return
	}
	// Trading control-plane: liquidity vertical halt + explicit pool stop.
	if !tradingGuard(c, "liquidity", "", "") {
		return
	}
	if tradingEntityStoppedRedis(c.Request.Context(), "pool", poolRedisKey(req.ChainID, req.TokenIn, req.TokenOut)) ||
		tradingEntityStoppedRedis(c.Request.Context(), "pool", poolRedisKey(req.ChainID, req.TokenOut, req.TokenIn)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "liquidity pool is stopped by the platform operator"})
		return
	}
	chain := evmChainByChainID(req.ChainID)
	if chain == nil || chain.RPCEndpoint == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "unsupported chain_id"})
		return
	}
	router := routerForChain(req.ChainID)
	if router == (common.Address{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": "no V2 router configured for this chain"})
		return
	}
	if !common.IsHexAddress(req.TokenIn) || !common.IsHexAddress(req.TokenOut) || !common.IsHexAddress(req.From) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token_in/token_out/from must be 0x addresses"})
		return
	}
	addrIn := common.HexToAddress(req.TokenIn)
	addrOut := common.HexToAddress(req.TokenOut)
	recipient := common.HexToAddress(req.From)
	if req.Recipient != "" && common.IsHexAddress(req.Recipient) {
		recipient = common.HexToAddress(req.Recipient)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), httpTimeout)
	defer cancel()

	decIn, err := fetchDecimals(ctx, chain.RPCEndpoint, addrIn)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not fetch token_in decimals: " + err.Error()})
		return
	}
	decOut, err := fetchDecimals(ctx, chain.RPCEndpoint, addrOut)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not fetch token_out decimals: " + err.Error()})
		return
	}

	amountInHuman, ok := new(big.Float).SetString(req.AmountIn)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount_in"})
		return
	}
	amountInWei, err := humanToWei(amountInHuman, decIn)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// The quote is always read on-chain: it is both the default limit basis
	// and the reference the caller's explicit limit is checked against.
	path := []common.Address{addrIn, addrOut}
	ret, err := ethCall(ctx, chain.RPCEndpoint, router, getAmountsOutData(amountInWei, path))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "router eth_call failed: " + err.Error()})
		return
	}
	amounts, err := decodeAmountsOut(ret)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not decode router response: " + err.Error()})
		return
	}
	quotedOutWei := amounts[len(amounts)-1]

	// The caller's limit is binding: min_output is used verbatim, otherwise
	// slippage is applied to the live quote. Ignoring these fields would remove
	// the protection the caller asked for.
	amountOutMinWei, err := resolveMinOut(req.AmountOutMin, req.Slippage, decOut, quotedOutWei)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deadline := big.NewInt(0)
	if req.DeadlineSec > 0 {
		deadline.SetInt64(req.DeadlineSec)
	} else {
		deadline.SetInt64(timeNowUnix() + 1200) // 20 min default
	}

	calldata := swapExactTokensForTokensData(amountInWei, amountOutMinWei, path, recipient, deadline)

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"action_required": "submit_on_chain",
		"submit_endpoint": "/api/v1/send",
		"chain_id":        req.ChainID,
		"router":          router.Hex(),
		"tx": gin.H{
			"from":     req.From,
			"to":       router.Hex(),
			"data":     "0x" + hex.EncodeToString(calldata),
			"chain_id": req.ChainID,
			"value":    "0",
		},
		"amount_in_wei":      amountInWei.String(),
		"quoted_out_wei":     quotedOutWei.String(),
		"amount_out_min_wei": amountOutMinWei.String(),
		"quoted_out_human":   expectedHumanStr(quotedOutWei, decOut),
		"deadline":           deadline.String(),
		"note":               "Approve the router to spend token_in (ERC-20 approve) first, then broadcast this via POST /api/v1/send.",
	})
}

// ---- helpers ----

func fetchDecimals(ctx context.Context, endpoint string, token common.Address) (int, error) {
	// decimals() selector = 0x313ce567, returns uint8
	res, err := ethCall(ctx, endpoint, token, erc20DecimalsData())
	if err != nil {
		return 0, err
	}
	if len(res) < 32 {
		return 0, fmt.Errorf("decimals() returned %d bytes", len(res))
	}
	d := new(big.Int).SetBytes(res[24:32]).Int64() // uint8 is right-aligned
	if d < 0 || d > 36 {
		return 0, fmt.Errorf("implausible decimals %d", d)
	}
	return int(d), nil
}

func humanToWei(amount *big.Float, decimals int) (*big.Int, error) {
	// amount * 10^decimals
	scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	weiFloat := new(big.Float).Mul(amount, scale)
	wei, _ := weiFloat.Int(nil)
	if wei.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	return wei, nil
}

func weiToHuman(wei *big.Int, decimals int) (string, *big.Float) {
	if decimals <= 0 {
		return wei.String(), new(big.Float).SetInt(wei)
	}
	scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(wei), scale).Float64()
	_ = f
	human, _ := new(big.Float).SetString("0")
	human = new(big.Float).Quo(new(big.Float).SetInt(wei), scale)
	s := human.Text('f', min(decimals, 8))
	return strings.TrimRight(strings.TrimRight(s, "0"), "."), human
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}
