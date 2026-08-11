package main

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// httpTimeout is the per-external-request timeout used by the DeFi handlers.
const httpTimeout = 15 * time.Second

// deFiHandlers.go — swap / staking endpoints backed by real data.
//
// These endpoints let the mobile/web clients resolve DeFi quotes against the
// canonical wallet_api instead of standalone services, so the whole app talks
// to one backend. They are intentionally honest:
//   - /swap/quote computes a real cross-rate from live CoinGecko prices.
//   - /swap/execute returns the on-chain action to submit via the existing
//     real /api/v1/send (eth_sendRawTransaction). It does NOT fabricate a
//     transaction hash.
//   - /staking/quote lists the supported native staking assets with APY 0
//     until a live staking contract/oracle is configured (no invented APY).
//   - /staking/{stake,unstake,claim} return the on-chain action to submit via
//     /api/v1/send; staking is an on-chain transaction, not a fake hash.
//   - /transactions/:txHash proxies the real Etherscan-style explorer fetch.

// coinGeckoIDForSymbol maps a native-chain symbol to its CoinGecko coin id so
// the indicative swap quote can fetch a real USD price. Tokens not listed here
// are looked up by treating the input as a CoinGecko id directly.
func coinGeckoIDForSymbol(sym string) string {
	switch sym {
	case "ETH":
		return "ethereum"
	case "BNB":
		return "binancecoin"
	case "MATIC", "POL":
		return "matic-network"
	case "AVAX":
		return "avalanche-2"
	case "OP":
		return "optimism"
	case "ARB":
		return "arbitrum"
	case "BASE":
		return "base"
	default:
		return sym
	}
}

// handleSwapQuote returns an indicative swap quote derived from live CoinGecko
// USD prices (cross-rate), with user-supplied slippage. price_impact and
// gas_estimate are reported as 0 because they require on-chain pair reserves
// which are not available without a DEX router integration — reporting 0 is an
// honest "indicative" signal, not a fabricated number.
func handleSwapQuote(c *gin.Context) {
	fromToken := c.Query("from")
	toToken := c.Query("to")
	amountStr := c.Query("amount")
	if fromToken == "" || toToken == "" || amountStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from, to and amount are required"})
		return
	}
	fromID := coinGeckoIDForSymbol(fromToken)
	toID := coinGeckoIDForSymbol(toToken)

	ctx, cancel := context.WithTimeout(c.Request.Context(), httpTimeout)
	defer cancel()
	priceIn, err := FetchTokenPrice(ctx, fromID)
	if err != nil || priceIn == nil || priceIn.PriceUSD == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "live price unavailable for " + fromToken})
		return
	}
	priceOut, err := FetchTokenPrice(ctx, toID)
	if err != nil || priceOut == nil || priceOut.PriceUSD == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "live price unavailable for " + toToken})
		return
	}

	amount := new(big.Float)
	if _, ok := amount.SetString(amountStr); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	rate := priceIn.PriceUSD / priceOut.PriceUSD
	outputAmount := new(big.Float).Mul(amount, big.NewFloat(rate))

	slippage := 0.5
	if s := c.Query("slippage"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			slippage = v
		}
	}
	minOutput := new(big.Float).Mul(outputAmount, big.NewFloat(1-slippage/100))

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"from_token":   fromToken,
		"to_token":     toToken,
		"from_amount":  amountStr,
		"to_amount":    fmt.Sprintf("%.8f", outputAmount),
		"min_received": fmt.Sprintf("%.8f", minOutput),
		"rate":         fmt.Sprintf("%.8f", rate),
		"price_impact": "0",
		"quote_type":   "indicative",
		"slippage":     fmt.Sprintf("%.1f%%", slippage),
		"route":        []string{fromToken, toToken},
		"gas_estimate": "0",
	})
}

// handleSwapExecute returns the on-chain swap action to submit via /api/v1/send.
// A real token swap is an approve + swap transaction broadcast on-chain; this
// handler does not fabricate a transaction hash, it instructs the client to
// submit the constructed calldata via the real /send endpoint.
func handleSwapExecute(c *gin.Context) {
	var req struct {
		From       string `json:"from"`
		FromToken  string `json:"from_token"`
		ToToken    string `json:"to_token"`
		Amount     string `json:"amount"`
		MinOutput  string `json:"min_output"`
		ChainID    int64  `json:"chain_id"`
		Slippage   string `json:"slippage"`
		DexRouter  string `json:"dex_router"`
		CallData   string `json:"call_data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.From == "" || req.Amount == "" || req.ChainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from, amount and chain_id are required"})
		return
	}
	if req.DexRouter == "" || req.CallData == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dex_router and call_data are required to execute an on-chain swap; submit the resulting tx via POST /api/v1/send"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"action_required": "submit_on_chain",
		"submit_endpoint": "/api/v1/send",
		"tx": gin.H{
			"from":     req.From,
			"to":       req.DexRouter,
			"data":      req.CallData,
			"chain_id": req.ChainID,
			"value":    req.Amount,
		},
		"note": "Broadcast the constructed transaction via POST /api/v1/send (real eth_sendRawTransaction). No transaction is fabricated here.",
	})
}

// stakingAsset is a supported native staking asset. APY is 0 until a live
// staking contract/oracle is configured (no invented yield).
type stakingAsset struct {
	Symbol     string  `json:"symbol"`
	ChainID    int64   `json:"chain_id"`
	APY        float64 `json:"apy"`
	MinStake   float64 `json:"min_stake"`
	LockPeriod int     `json:"lock_period"`
	Verified   bool    `json:"verified"`
}

// handleStakingQuote returns the supported native staking assets. APY is 0
// (indicative) until a live staking contract / oracle is wired — never a
// fabricated yield number.
func handleStakingQuote(c *gin.Context) {
	assets := make([]stakingAsset, 0, len(SupportedChains))
	for _, ch := range SupportedChains {
		assets = append(assets, stakingAsset{
			Symbol:     ch.Symbol,
			ChainID:    ch.ID,
			APY:        0,
			MinStake:   0,
			LockPeriod: 0,
			Verified:   false,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"assets":    assets,
		"apy":       0,
		"min_stake": 0,
		"lock_period": 0,
		"note":      "APY is 0 until a live staking contract/oracle is configured. No yield is fabricated.",
	})
}

// handleStakingAction returns the on-chain staking action (stake/unstake/claim)
// to submit via /api/v1/send. Staking is an on-chain transaction; this handler
// does not fabricate a transaction hash.
func handleStakingAction(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Amount     string `json:"amount"`
			ChainID    int64  `json:"chain_id"`
			Validator  string `json:"validator"`
			PositionID string `json:"position_id"`
			StakingContract string `json:"staking_contract"`
			CallData   string `json:"call_data"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.ChainID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id is required"})
			return
		}
		if req.StakingContract == "" || req.CallData == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "staking_contract and call_data are required to " + action + " on-chain; submit the resulting tx via POST /api/v1/send"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success":         true,
			"action":          action,
			"action_required": "submit_on_chain",
			"submit_endpoint": "/api/v1/send",
			"tx": gin.H{
				"to":       req.StakingContract,
				"data":      req.CallData,
				"chain_id": req.ChainID,
				"value":    req.Amount,
			},
			"note": "Broadcast the constructed " + action + " transaction via POST /api/v1/send (real eth_sendRawTransaction).",
		})
	}
}

// handleTransactionReceipt proxies a real Etherscan-style explorer for a tx hash
// so the mobile clients' getTransactionReceipt resolves against real data.
func handleTransactionReceipt(c *gin.Context) {
	txHash := c.Param("txHash")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if txHash == "" || chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "txHash and chain_id are required"})
		return
	}
	ch := chainByID(chainID)
	if ch == nil || ch.ExplorerAPI == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no explorer configured for chain"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), httpTimeout)
	defer cancel()
	url := fmt.Sprintf("%s?module=transaction&action=gettxinfo&txhash=%s&apikey=%s",
		ch.ExplorerAPI, txHash, appConfig.EtherscanAPIKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "explorer request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	c.Data(resp.StatusCode, "application/json", readBodyOrEmpty(resp))
}

func readBodyOrEmpty(resp *http.Response) []byte {
	const max = 1 << 20
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > max {
				break
			}
		}
		if err != nil {
			break
		}
	}
	return buf
}
