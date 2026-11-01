package main

// Slippage handling shared by every swap path.
//
// A slippage tolerance is a spending limit, not a display preference. When a
// client sends "slippage" or "min_output" and the server ignores it, the user's
// stated protection is gone and the swap executes at whatever the pool offers.
// These helpers make the client's limit binding, and refuse values that cannot
// be a real limit.

import (
	"fmt"
	"math/big"
	"strings"
)

const (
	// defaultSlippagePercent is used only when the client sends neither
	// slippage nor min_output.
	defaultSlippagePercent = 0.5

	// maxSlippagePercent bounds the tolerance. Above this a "swap" can be a
	// near-total loss to a sandwich bot, which is the attack slippage protects
	// against, so an operator cannot widen it per-request.
	maxSlippagePercent = 50.0
)

// parseSlippagePercent reads a client slippage value. It accepts a plain number
// ("0.5") or a percent-suffixed string ("0.5%") because clients differ. It
// rejects anything outside (0, maxSlippagePercent].
func parseSlippagePercent(raw string) (float64, error) {
	raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), "%"))
	if raw == "" {
		return 0, fmt.Errorf("empty slippage")
	}
	var value float64
	if _, err := fmt.Sscanf(raw, "%f", &value); err != nil {
		return 0, fmt.Errorf("invalid slippage %q", raw)
	}
	if !(value > 0) {
		return 0, fmt.Errorf("slippage must be greater than zero")
	}
	if value > maxSlippagePercent {
		return 0, fmt.Errorf("slippage %.4f%% exceeds the maximum of %.1f%%", value, maxSlippagePercent)
	}
	return value, nil
}

// minOutFromSlippage derives the on-chain minimum received from a quoted output
// and a tolerance: minOut = quote * (1 - slippage/100), truncated to an integer
// number of wei. Truncating (rather than rounding up) keeps the limit on the
// protective side.
func minOutFromSlippage(quotedOutWei *big.Int, slippagePercent float64) *big.Int {
	scale := big.NewFloat(10000)
	// 10000 - slippage*100, in ten-thousandths, avoids float drift on values
	// like 0.1 that cannot be represented exactly in binary.
	keep := new(big.Float).Sub(scale, new(big.Float).Mul(big.NewFloat(slippagePercent), big.NewFloat(100)))
	if keep.Sign() <= 0 {
		return big.NewInt(0)
	}
	minOut := new(big.Float).Quo(new(big.Float).Mul(new(big.Float).SetInt(quotedOutWei), keep), scale)
	result, _ := minOut.Int(nil)
	if result.Sign() < 0 {
		return big.NewInt(0)
	}
	return result
}

// resolveMinOut determines the minimum output for a swap, in wei.
//
// Precedence: an explicit min_output is the user's own limit and is used as-is.
// Otherwise slippage is applied to the live on-chain quote. When neither is
// supplied the default tolerance is applied to the quote, so every swap carries
// a real limit rather than none.
func resolveMinOut(
	minOutputHuman string,
	slippageRaw string,
	decimalsOut int,
	quotedOutWei *big.Int,
) (*big.Int, error) {
	if strings.TrimSpace(minOutputHuman) != "" {
		minHuman, ok := new(big.Float).SetString(strings.TrimSpace(minOutputHuman))
		if !ok {
			return nil, fmt.Errorf("invalid min_output")
		}
		minWei, err := humanToWei(minHuman, decimalsOut)
		if err != nil {
			return nil, fmt.Errorf("invalid min_output: %w", err)
		}
		if minWei.Cmp(quotedOutWei) > 0 {
			// A limit above the current quote can never fill; failing here is
			// clearer than emitting a transaction that is guaranteed to revert.
			return nil, fmt.Errorf("min_output exceeds the current on-chain quote")
		}
		return minWei, nil
	}

	slippage := defaultSlippagePercent
	if strings.TrimSpace(slippageRaw) != "" {
		parsed, err := parseSlippagePercent(slippageRaw)
		if err != nil {
			return nil, err
		}
		slippage = parsed
	}
	return minOutFromSlippage(quotedOutWei, slippage), nil
}
