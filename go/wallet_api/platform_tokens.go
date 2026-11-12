package main

// platform_tokens.go — the platform's own assets, as every wallet surface sees
// them.
//
// TigerSmartChain (EIP-155 chain id 7710) is the network TigerWallet operates.
// Its genesis block contains two contracts as predeploys, so their addresses
// are fixed by consensus: they are the same on every node, in every
// environment, from block zero. RUSD is the USD stablecoin; WTGR is the ERC-20
// wrapper of the native TGR coin, which is what lets TigerSwap pools quote TGR
// against RUSD.
//
// TGR itself is the native coin of the chain, exactly like ETH on Ethereum, so
// it has no contract address and is not listed as a token.
//
// The addresses below are duplicated from tiger_smart_chain/chain/predeploy.go
// because the two live in separate Go modules. TestPlatformTokensMatchChain
// reads that file and fails if the two ever disagree, so the duplication cannot
// drift silently.

import (
	"context"
	"strings"
	"time"
)

// The canonical predeploy addresses and metadata.
const (
	// rusdPredeployAddress is the RUSD stablecoin on TigerSmartChain.
	rusdPredeployAddress = "0x0000000000000000000000000000000000007710"

	// wtgrPredeployAddress is the WTGR wrapped-native token on TigerSmartChain.
	wtgrPredeployAddress = "0x0000000000000000000000000000000000007711"

	// tigerSmartChainID is the EIP-155 chain id of TigerSmartChain mainnet.
	tigerSmartChainID int64 = 7710

	// tigerSmartChainTestnetID is the EIP-155 chain id of the TigerSmartChain
	// public testnet. It shares the mainnet predeploy addresses, because both
	// networks build their genesis from the same predeploy set.
	tigerSmartChainTestnetID int64 = 7711

	// tgrNativeSymbol is the native coin symbol of TigerSmartChain.
	tgrNativeSymbol = "TGR"
)

// PlatformToken describes a first-class platform asset. It carries the metadata
// a client needs to render the asset and, for contract tokens, the address to
// call.
type PlatformToken struct {
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	ChainID    int64   `json:"chain_id"`
	Decimals   int     `json:"decimals"`
	Contract   string  `json:"contract,omitempty"` // empty for the native coin
	IsNative   bool    `json:"is_native"`
	Standard   string  `json:"standard"` // "native" | "ERC-20"
	USDPrice   float64 `json:"usd_price"`
	USDSource  string  `json:"usd_price_source"`
	PeggedUSD  bool    `json:"pegged_usd"`
	IsPlatform bool    `json:"is_platform"`
}

// platformTokens returns the platform assets with a live USD price attached.
//
// The price comes from the same aggregator the rest of the wallet uses, so a
// client never has to special-case these three assets. When a price cannot be
// obtained the field stays 0 and USDSource reads "unavailable": a client must
// then omit the fiat value rather than show a wrong number.
func platformTokens(ctx context.Context) []PlatformToken {
	out := []PlatformToken{
		{
			Symbol:     tgrNativeSymbol,
			Name:       "Tiger",
			ChainID:    tigerSmartChainID,
			Decimals:   18,
			IsNative:   true,
			Standard:   "native",
			IsPlatform: true,
		},
		{
			Symbol:     "WTGR",
			Name:       "Wrapped TGR",
			ChainID:    tigerSmartChainID,
			Decimals:   18,
			Contract:   wtgrPredeployAddress,
			Standard:   "ERC-20",
			IsPlatform: true,
		},
		{
			Symbol:     "RUSD",
			Name:       "Royal Tiger USD",
			ChainID:    tigerSmartChainID,
			Decimals:   18,
			Contract:   rusdPredeployAddress,
			Standard:   "ERC-20",
			IsPlatform: true,
			PeggedUSD:  true,
		},
	}

	for i := range out {
		quote := cachedPriceQuote(ctx, out[i].Symbol)
		out[i].USDPrice = quote.USD
		out[i].USDSource = quote.Source
	}
	return out
}

// cachedPriceQuote returns a price, reusing the Redis cache when it is warm.
// The cache is a latency optimisation only: a miss, a Redis outage or a decode
// failure all fall through to a live fetch.
func cachedPriceQuote(ctx context.Context, symbol string) PriceQuote {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if store != nil && store.PG != nil {
		var cached PriceQuote
		if store.GetCache(ctx, store.cacheKey("pricefeed", sym), &cached) == nil && cached.USD > 0 {
			return cached
		}
	}
	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	quote := FetchAggregatedPrice(fetchCtx, sym)
	if quote.Available() && store != nil && store.PG != nil {
		_ = store.SetCache(ctx, store.cacheKey("pricefeed", sym), quote, 60*time.Second)
	}
	return quote
}

// priceForSymbol is the convenience form used by the balance handlers: it
// returns the USD price of an asset symbol, or 0 when none is available.
func priceForSymbol(ctx context.Context, symbol string) float64 {
	return cachedPriceQuote(ctx, symbol).USD
}

// platformTokenBySymbol looks a platform asset up by symbol (case-insensitive).
func platformTokenBySymbol(symbol string) (PlatformToken, bool) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	for _, t := range platformTokensStatic() {
		if t.Symbol == sym {
			return t, true
		}
	}
	return PlatformToken{}, false
}

// platformTokensStatic returns the platform assets without touching the network.
// It exists for callers that need the metadata only (validation, address
// resolution) and must not block on a price fetch.
func platformTokensStatic() []PlatformToken {
	return []PlatformToken{
		{Symbol: tgrNativeSymbol, Name: "Tiger", ChainID: tigerSmartChainID, Decimals: 18, IsNative: true, Standard: "native", IsPlatform: true},
		{Symbol: "WTGR", Name: "Wrapped TGR", ChainID: tigerSmartChainID, Decimals: 18, Contract: wtgrPredeployAddress, Standard: "ERC-20", IsPlatform: true},
		{Symbol: "RUSD", Name: "Royal Tiger USD", ChainID: tigerSmartChainID, Decimals: 18, Contract: rusdPredeployAddress, Standard: "ERC-20", IsPlatform: true, PeggedUSD: true},
	}
}

// isPlatformToken reports whether a symbol belongs to the platform asset set.
func isPlatformToken(symbol string) bool {
	_, ok := platformTokenBySymbol(symbol)
	return ok
}
