package main

import "context"

// token_registry.go — curated token list per chain. Real contract addresses,
// decimals, and symbols. The defaults seed the DB-backed token_registry table
// on startup (store.seedTokenRegistry); runtime additions/updates persist in
// the table and are made by the MasterWallet owner / ProjectParty approval
// flow over POST /api/v1/admin/tokens.

var defaultTokenRegistry = map[int64][]TokenInfo{
	1: { // Ethereum mainnet
		{Contract: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 1},
		{Contract: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Symbol: "USDT", Name: "Tether USD", Decimals: 6, ChainID: 1},
		{Contract: "0x6B175474E89094C44Da98b954EedeAC495271d0F", Symbol: "DAI", Name: "Dai Stablecoin", Decimals: 18, ChainID: 1},
		{Contract: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", Symbol: "WETH", Name: "Wrapped Ether", Decimals: 18, ChainID: 1},
		{Contract: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", Symbol: "WBTC", Name: "Wrapped BTC", Decimals: 8, ChainID: 1},
		{Contract: "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984", Symbol: "UNI", Name: "Uniswap", Decimals: 18, ChainID: 1},
		{Contract: "0x514910771AF9Ca656af840dff83E8264EcF986CA", Symbol: "LINK", Name: "Chainlink", Decimals: 18, ChainID: 1},
		{Contract: "0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9", Symbol: "AAVE", Name: "Aave", Decimals: 18, ChainID: 1},
	},
	56: { // BSC
		{Contract: "0xe9e7CEA3DedcA5984780Bafc599bD69ACd087D56", Symbol: "BUSD", Name: "Binance USD", Decimals: 18, ChainID: 56},
		{Contract: "0x55d398326f99059fF775485246999027B3197955", Symbol: "USDT", Name: "Tether USD", Decimals: 18, ChainID: 56},
		{Contract: "0x8AC76A51cc950d9822D68b83E1BEc96492dac657", Symbol: "USDC", Name: "USD Coin", Decimals: 18, ChainID: 56},
		{Contract: "0x2170Ed0880ac9A755fd29B2688956BD959F933F8", Symbol: "ETH", Name: "Ethereum", Decimals: 18, ChainID: 56},
		{Contract: "0x7130d2A12BFEBC97F94433167556dF6bd90B5Aed", Symbol: "BTCB", Name: "BTCB", Decimals: 18, ChainID: 56},
	},
	137: { // Polygon
		{Contract: "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 137},
		{Contract: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F", Symbol: "USDT", Name: "Tether USD", Decimals: 6, ChainID: 137},
		{Contract: "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063", Symbol: "DAI", Name: "Dai", Decimals: 18, ChainID: 137},
		{Contract: "0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619", Symbol: "WETH", Name: "Wrapped Ether", Decimals: 18, ChainID: 137},
	},
	42161: { // Arbitrum
		{Contract: "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 42161},
		{Contract: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9", Symbol: "USDT", Name: "Tether USD", Decimals: 6, ChainID: 42161},
		{Contract: "0xDA10009cBd1D6e9500de913b00040000f0a3e64c", Symbol: "DAI", Name: "Dai", Decimals: 18, ChainID: 42161},
		{Contract: "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1", Symbol: "WETH", Name: "Wrapped Ether", Decimals: 18, ChainID: 42161},
	},
	10: { // Optimism
		{Contract: "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 10},
		{Contract: "0x94b008aA00579c1307B0EF2c499a594F0EF90116", Symbol: "USDT", Name: "Tether USD", Decimals: 6, ChainID: 10},
		{Contract: "0xDA10009cBd1D6e9500de913b00040000f0a3e64c", Symbol: "DAI", Name: "Dai", Decimals: 18, ChainID: 10},
		{Contract: "0x4200000000000000000000000000000000000006", Symbol: "WETH", Name: "Wrapped Ether", Decimals: 18, ChainID: 10},
	},
	8453: { // Base
		{Contract: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 8453},
		{Contract: "0x4200000000000000000000000000000000000006", Symbol: "WETH", Name: "Wrapped Ether", Decimals: 18, ChainID: 8453},
		{Contract: "0x50c5725949A6F0c72E6C3a7794904489D5667281", Symbol: "DAI", Name: "Dai", Decimals: 18, ChainID: 8453},
	},
}

// tokensForChain returns the registry tokens for a chain. Reads from the
// DB-backed token_registry (so runtime additions appear); falls back to the
// curated in-memory defaults if the DB is unavailable (e.g. store not yet
// initialized during boot).
func tokensForChain(chainID int64) []TokenInfo {
	if store != nil && store.PG != nil {
		rows, err := store.ListRegistryTokens(context.Background(), chainID)
		if err == nil && len(rows) > 0 {
			out := make([]TokenInfo, 0, len(rows))
			for i := range rows {
				r := &rows[i]
				out = append(out, TokenInfo{
					Contract: r.Contract,
					Symbol:   r.Symbol,
					Name:     r.Name,
					Decimals: r.Decimals,
					Logo:     r.LogoURI,
					ChainID:  r.ChainID,
				})
			}
			return out
		}
	}
	if toks, ok := defaultTokenRegistry[chainID]; ok {
		return toks
	}
	return nil
}

// allRegistryTokens returns the full registry grouped by chain id (string keys
// for JSON). Reads from DB with fallback to defaults.
func allRegistryTokens() map[int64][]TokenInfo {
	out := map[int64][]TokenInfo{}
	if store != nil && store.PG != nil {
		rows, err := store.ListRegistryTokens(context.Background(), 0)
		if err == nil && len(rows) > 0 {
			for i := range rows {
				r := &rows[i]
				out[r.ChainID] = append(out[r.ChainID], TokenInfo{
					Contract: r.Contract,
					Symbol:   r.Symbol,
					Name:     r.Name,
					Decimals: r.Decimals,
					Logo:     r.LogoURI,
					ChainID:  r.ChainID,
				})
			}
			return out
		}
	}
	return defaultTokenRegistry
}
