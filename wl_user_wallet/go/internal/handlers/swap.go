package handlers

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-user-wallet/internal/onchain"
)

// GET /swap/quote?from=&to=&amount=&chain_id= — real quote. Native ERC-20
// path routing via Uniswap-V2 router getAmountsOut, with CoinGecko cross-rate
// fallback for native-vs-native (e.g. ETH->BTC) pairs that have no on-chain
// liquidity path on the chosen chain. Fail-closed 503 if neither available.
func (s *Svc) SwapQuote(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	amountStr := c.Query("amount")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if from == "" || to == "" || amountStr == "" || chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from, to, amount, chain_id required"})
		return
	}
	amount, ok := new(big.Float).SetString(amountStr)
	if !ok || amount.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}

	// If both legs are EVM contract addresses on this chain, route on-chain via
	// the V2 router. Otherwise fall back to a CoinGecko cross-rate quote.
	if common.IsHexAddress(from) && common.IsHexAddress(to) {
		routerAddr := onchain.RouterForChain(chainID)
		rpc := rpcForChain(chainID)
		if routerAddr != "" && rpc != "" {
			quote, err := s.onChainQuote(c.Request.Context(), rpc, routerAddr, from, to, amount, chainID)
			if err == nil {
				c.JSON(http.StatusOK, quote)
				return
			}
		}
	}

	// CoinGecko cross-rate fallback for native assets (e.g. ETH->SOL).
	quote, err := s.coingeckoCrossRate(c.Request.Context(), from, to, amount)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "swap quote unavailable: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, quote)
}

type swapQuoteResult struct {
	From        string  `json:"from"`
	To          string  `json:"to"`
	AmountIn    string  `json:"amount_in"`
	AmountOut   string  `json:"amount_out"`
	AmountOutF  float64 `json:"amount_out_f"`
	ChainID     int64   `json:"chain_id"`
	Route       string  `json:"route"`
	PriceImpact string  `json:"price_impact,omitempty"`
}

func (s *Svc) onChainQuote(ctx context.Context, rpc, routerAddr, from, to string, amount *big.Float, chainID int64) (*swapQuoteResult, error) {
	fromAddr := common.HexToAddress(from)
	toAddr := common.HexToAddress(to)
	fromDec, err := onchain.FetchDecimals(ctx, rpc, fromAddr)
	if err != nil {
		return nil, fmt.Errorf("from decimals: %w", err)
	}
	amountInWei, err := onchain.HumanToWei(amount, fromDec)
	if err != nil {
		return nil, err
	}
	path := []common.Address{fromAddr, toAddr}
	data := onchain.GetAmountsOutData(amountInWei, path)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	ret, err := onchain.EthCall(ctx, rpc, common.HexToAddress(routerAddr), data)
	if err != nil {
		return nil, fmt.Errorf("router getAmountsOut: %w", err)
	}
	amounts, err := onchain.DecodeAmountsOut(ret)
	if err != nil {
		return nil, err
	}
	amountOut := amounts[len(amounts)-1]
	toDec, err := onchain.FetchDecimals(ctx, rpc, toAddr)
	if err != nil {
		toDec = 18
	}
	outStr, _ := onchain.WeiToHuman(amountOut, toDec)
	return &swapQuoteResult{
		From:       from,
		To:         to,
		AmountIn:   amount.String(),
		AmountOut:  outStr,
		AmountOutF: onchain.WeiToFloat(amountOut, toDec),
		ChainID:    chainID,
		Route:      "uniswap-v2",
	}, nil
}

func (s *Svc) coingeckoCrossRate(ctx context.Context, from, to string, amount *big.Float) (*swapQuoteResult, error) {
	fromID := resolveCoinGeckoID(from)
	toID := resolveCoinGeckoID(to)
	if fromID == "" || toID == "" {
		return nil, fmt.Errorf("cannot resolve CoinGecko ids for %s/%s", from, to)
	}
	fromPrice, err := onchain.FetchTokenPrice(ctx, fromID, s.cfg.CoinGeckoAPIKey)
	if err != nil {
		return nil, err
	}
	toPrice, err := onchain.FetchTokenPrice(ctx, toID, s.cfg.CoinGeckoAPIKey)
	if err != nil {
		return nil, err
	}
	if fromPrice.PriceUSD == 0 || toPrice.PriceUSD == 0 {
		return nil, fmt.Errorf("missing price for %s or %s", fromID, toID)
	}
	amtF, _ := amount.Float64()
	out := (amtF * fromPrice.PriceUSD) / toPrice.PriceUSD
	return &swapQuoteResult{
		From:       from,
		To:         to,
		AmountIn:   amount.String(),
		AmountOut:  fmt.Sprintf("%.8f", out),
		AmountOutF: out,
		Route:      "coingecko-cross-rate",
	}, nil
}

func resolveCoinGeckoID(symbolOrAddr string) string {
	if sym, ok := symbolToCoinGeckoID[strings.ToUpper(symbolOrAddr)]; ok {
		return sym
	}
	return strings.ToLower(symbolOrAddr)
}

// POST /swap/execute — builds the on-chain swapExactTokensForTokens calldata
// (real V2 router calldata) and returns it for the client to submit via /send.
// Fail-closed 503 if no router/RPC configured for the chain.
func (s *Svc) SwapExecute(c *gin.Context) {
	var req struct {
		From        string `json:"from" binding:"required"`
		To          string `json:"to" binding:"required"`
		Amount      string `json:"amount" binding:"required"`
		ChainID     int64  `json:"chain_id" binding:"required"`
		Recipient   string `json:"recipient" binding:"required"`
		SlippagePct string `json:"slippage_pct"` // default 0.5
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	routerAddr := onchain.RouterForChain(req.ChainID)
	rpc := rpcForChain(req.ChainID)
	if routerAddr == "" || rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no swap router/RPC configured for chain"})
		return
	}
	amount, ok := new(big.Float).SetString(req.Amount)
	if !ok || amount.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	fromAddr := common.HexToAddress(req.From)
	toAddr := common.HexToAddress(req.To)
	recipient, err := onchain.HexAddress(req.Recipient)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	fromDec, err := onchain.FetchDecimals(ctx, rpc, fromAddr)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "decimals fetch: " + err.Error()})
		return
	}
	amountIn, err := onchain.HumanToWei(amount, fromDec)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	path := []common.Address{fromAddr, toAddr}
	quoteData := onchain.GetAmountsOutData(amountIn, path)
	ret, err := onchain.EthCall(ctx, rpc, common.HexToAddress(routerAddr), quoteData)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "quote failed: " + err.Error()})
		return
	}
	amounts, err := onchain.DecodeAmountsOut(ret)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "decode quote: " + err.Error()})
		return
	}
	expectedOut := amounts[len(amounts)-1]
	slip := parseSlippage(req.SlippagePct)
	// amountOutMin = expectedOut * (1 - slip)
	slipFactor := new(big.Float).Sub(big.NewFloat(1), big.NewFloat(slip))
	minOutFloat := new(big.Float).Mul(new(big.Float).SetInt(expectedOut), slipFactor)
	amountOutMin, _ := minOutFloat.Int(nil)
	if amountOutMin.Sign() < 0 {
		amountOutMin = big.NewInt(0)
	}
	deadline := big.NewInt(time.Now().Unix() + 1200) // 20 min
	calldata := onchain.SwapExactTokensForTokensData(amountIn, amountOutMin, path, recipient, deadline)
	c.JSON(http.StatusOK, gin.H{
		"chain_id":     req.ChainID,
		"router":       routerAddr,
		"to":           routerAddr,
		"data":         onchain.HexEncode(calldata),
		"value":        "0",
		"amount_in":    amountIn.String(),
		"amount_out":   expectedOut.String(),
		"min_out":      amountOutMin.String(),
		"recipient":    recipient.Hex(),
		"action":       "send_raw_tx",
	})
}

func parseSlippage(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 || v > 100 {
		return 0.005
	}
	return v / 100
}
