package main

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

// GasTier is a fee level derived from what recent blocks actually paid, rather
// than from a multiplier applied to a single sample. A wallet that shows the
// user invented slow/fast numbers is guessing on their behalf about money.
type GasTier struct {
	// PriorityFeeWei is the median tip miners accepted in the sampled blocks.
	PriorityFeeWei *big.Int
	// MaxFeeWei is the ceiling to sign: base fee with headroom plus the tip.
	MaxFeeWei *big.Int
	// BaseFeeWei is the next block's base fee.
	BaseFeeWei *big.Int
}

// FeeHistoryTiers samples recent blocks and returns low / market / aggressive
// fee levels.
//
// It calls eth_feeHistory with the 10th, 50th and 90th reward percentiles over
// the last 20 blocks. Percentiles describe what transactions in those blocks
// actually paid to be included, so "fast" means "paid more than 90% of recent
// transactions" - a fact from the chain, not a factor we chose.
//
// Chains that do not implement eth_feeHistory (pre-EIP-1559 or non-Ethereum
// RPCs) return an error so the caller can fall back to eth_gasPrice instead of
// presenting a fabricated tier.
func FeeHistoryTiers(ctx context.Context, endpoint string) (low, market, fast GasTier, err error) {
	// A short timeout: the caller has a live user waiting.
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var history feeHistory
	err = rpcCall(callCtx, endpoint, func(client *rpc.Client) error {
		return client.CallContext(callCtx, &history, "eth_feeHistory",
			feeHistoryBlocks,
			"latest",
			feeHistoryPercentiles,
		)
	})
	if err != nil {
		return GasTier{}, GasTier{}, GasTier{}, fmt.Errorf("eth_feeHistory: %w", err)
	}
	return computeTiers(history)
}

// feeHistory is the subset of the eth_feeHistory response this service uses.
// The node also returns baseFeePerBlobGas and blobGasUsedRatio; they are
// irrelevant to legacy and EIP-1559 fee selection.
type feeHistory struct {
	OldestBlock   string     `json:"oldestBlock"`
	BaseFeePerGas []string   `json:"baseFeePerGas"`
	Reward        [][]string `json:"reward"`
}

const (
	feeHistoryBlocks      = "0x14" // 20 blocks
	feeHistoryPercentiles = "[10,50,90]"
)

// computeTiers turns a fee-history sample into the three fee levels. It is
// separated from the RPC call so the arithmetic can be tested against a
// recorded node response.
func computeTiers(history feeHistory) (low, market, fast GasTier, err error) {
	if len(history.BaseFeePerGas) == 0 {
		return GasTier{}, GasTier{}, GasTier{}, fmt.Errorf("eth_feeHistory returned no base fee")
	}

	// The last entry is the base fee for the block after the sample window.
	baseFee, ok := hexToBig(history.BaseFeePerGas[len(history.BaseFeePerGas)-1])
	if !ok {
		return GasTier{}, GasTier{}, GasTier{}, fmt.Errorf("unparsable base fee")
	}

	percentile := func(index int) (*big.Int, bool) {
		var sum *big.Int
		var count int64
		for _, blockRewards := range history.Reward {
			if index >= len(blockRewards) {
				continue
			}
			tip, ok := hexToBig(blockRewards[index])
			if !ok {
				continue
			}
			if sum == nil {
				sum = new(big.Int)
			}
			sum.Add(sum, tip)
			count++
		}
		if count == 0 {
			return nil, false
		}
		return new(big.Int).Div(sum, big.NewInt(count)), true
	}

	build := func(tip *big.Int, baseMultiplier, baseDivisor int64, tipMultiplier int64) GasTier {
		maxFee := new(big.Int).Mul(baseFee, big.NewInt(baseMultiplier))
		maxFee.Div(maxFee, big.NewInt(baseDivisor))
		maxFee.Add(maxFee, new(big.Int).Mul(tip, big.NewInt(tipMultiplier)))
		return GasTier{PriorityFeeWei: tip, MaxFeeWei: maxFee, BaseFeeWei: new(big.Int).Set(baseFee)}
	}

	lowTip, ok := percentile(0)
	if !ok {
		return GasTier{}, GasTier{}, GasTier{}, fmt.Errorf("eth_feeHistory returned no low-percentile rewards")
	}
	marketTip, ok := percentile(1)
	if !ok {
		return GasTier{}, GasTier{}, GasTier{}, fmt.Errorf("eth_feeHistory returned no median rewards")
	}
	fastTip, ok := percentile(2)
	if !ok {
		return GasTier{}, GasTier{}, GasTier{}, fmt.Errorf("eth_feeHistory returned no high-percentile rewards")
	}

	// Headroom over the base fee. Every tier must cover at least the current
	// base fee: EIP-1559 rejects a transaction whose maxFeePerGas is below the
	// base fee of the block that includes it, so a tier priced under the base
	// fee is not "slow", it is unminable. The levels therefore differ in tip
	// percentile and in how much base-fee movement they tolerate.
	low = build(lowTip, 1000, 1000, 1) // pays the base fee exactly; waits if it rises
	market = build(marketTip, 1125, 1000, 1)
	fast = build(fastTip, 1250, 1000, 2)

	return low, market, fast, nil
}

// hexToBig parses a 0x-quantity into a big.Int.
func hexToBig(s string) (*big.Int, bool) {
	n := new(big.Int)
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		_, ok := n.SetString(s[2:], 16)
		return n, ok
	}
	_, ok := n.SetString(s, 10)
	return n, ok
}

// gasTiersPayload renders tiers for a JSON response, with nil-safe fields.
func gasTiersPayload(tier GasTier) map[string]string {
	out := map[string]string{}
	if tier.PriorityFeeWei != nil {
		out["priority_fee_wei"] = tier.PriorityFeeWei.String()
	}
	if tier.MaxFeeWei != nil {
		out["max_fee_wei"] = tier.MaxFeeWei.String()
	}
	if tier.BaseFeeWei != nil {
		out["base_fee_wei"] = tier.BaseFeeWei.String()
	}
	return out
}
