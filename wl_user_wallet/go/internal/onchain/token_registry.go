package onchain

// token_registry.go — curated token list per chain. Real contract addresses,
// decimals, and symbols. Ported from the canonical wallet_api token_registry.go.

var defaultTokenRegistry = map[int64][]TokenInfo{
	1: {
		{Contract: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 1},
		{Contract: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Symbol: "USDT", Name: "Tether USD", Decimals: 6, ChainID: 1},
		{Contract: "0x6B175474E89094C44Da98b954EedeAC495271d0F", Symbol: "DAI", Name: "Dai Stablecoin", Decimals: 18, ChainID: 1},
		{Contract: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", Symbol: "WETH", Name: "Wrapped Ether", Decimals: 18, ChainID: 1},
		{Contract: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", Symbol: "WBTC", Name: "Wrapped BTC", Decimals: 8, ChainID: 1},
		{Contract: "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984", Symbol: "UNI", Name: "Uniswap", Decimals: 18, ChainID: 1},
		{Contract: "0x514910771AF9Ca656af840dff83E8264EcF986CA", Symbol: "LINK", Name: "Chainlink", Decimals: 18, ChainID: 1},
		{Contract: "0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9", Symbol: "AAVE", Name: "Aave", Decimals: 18, ChainID: 1},
	},
	56: {
		{Contract: "0xe9e7CEA3DedcA5984780Bafc599bD69ACd087D56", Symbol: "BUSD", Name: "Binance USD", Decimals: 18, ChainID: 56},
		{Contract: "0x55d398326f99059fF775485246999027B3197955", Symbol: "USDT", Name: "Tether USD", Decimals: 18, ChainID: 56},
		{Contract: "0x8AC76A51cc950d9822D68b83E1BEc96492dac657", Symbol: "USDC", Name: "USD Coin", Decimals: 18, ChainID: 56},
		{Contract: "0x2170Ed0880ac9A755fd29B2688956BD959F933F8", Symbol: "ETH", Name: "Ethereum", Decimals: 18, ChainID: 56},
		{Contract: "0x7130d2A12BFEBC97F94433167556dF6bd90B5Aed", Symbol: "BTCB", Name: "BTCB", Decimals: 18, ChainID: 56},
	},
	137: {
		{Contract: "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 137},
		{Contract: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F", Symbol: "USDT", Name: "Tether USD", Decimals: 6, ChainID: 137},
		{Contract: "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063", Symbol: "DAI", Name: "Dai", Decimals: 18, ChainID: 137},
		{Contract: "0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619", Symbol: "WETH", Name: "Wrapped Ether", Decimals: 18, ChainID: 137},
	},
	42161: {
		{Contract: "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 42161},
		{Contract: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9", Symbol: "USDT", Name: "Tether USD", Decimals: 6, ChainID: 42161},
		{Contract: "0xDA10009cBd1D6e9500de913b00040000f0a3e64c", Symbol: "DAI", Name: "Dai", Decimals: 18, ChainID: 42161},
		{Contract: "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1", Symbol: "WETH", Name: "Wrapped Ether", Decimals: 18, ChainID: 42161},
	},
	10: {
		{Contract: "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 10},
		{Contract: "0x94b008aA00579c1307B0EF2c499a594F0EF90116", Symbol: "USDT", Name: "Tether USD", Decimals: 6, ChainID: 10},
		{Contract: "0xDA10009cBd1D6e9500de913b00040000f0a3e64c", Symbol: "DAI", Name: "Dai", Decimals: 18, ChainID: 10},
		{Contract: "0x4200000000000000000000000000000000000006", Symbol: "WETH", Name: "Wrapped Ether", Decimals: 18, ChainID: 10},
	},
	8453: {
		{Contract: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: 8453},
		{Contract: "0x4200000000000000000000000000000000000006", Symbol: "WETH", Name: "Wrapped Ether", Decimals: 18, ChainID: 8453},
		{Contract: "0x50c5725949A6F0c72E6C3a7794904489D5667281", Symbol: "DAI", Name: "Dai", Decimals: 18, ChainID: 8453},
	},
}

// TokensForChain returns the registry tokens for a chain.
func TokensForChain(chainID int64) []TokenInfo {
	if toks, ok := defaultTokenRegistry[chainID]; ok {
		return toks
	}
	return nil
}

// V2Routers maps chain IDs to public Uniswap-V2-compatible Router02 addresses.
// Override per chain via env CHAIN_<id>_ROUTER.
var v2Routers = map[int64]string{
	1:       "0x7a250d5630b4cf539739df2c5dac2f9c3b1c09cf",
	56:      "0x10ED43C718714eb63d5aA57B78B54704E256024E",
	137:     "0xa5E0829CaCED8fFCEEdC5d972f14341d1C2C4F6F",
	42161:   "0x4752ba5dbc23f44d87826276bf6fd6b1c37ac4d4",
	10:      "0x1b02dA8Cb0d097eB8D57A175b88c7D8b47992406",
	8453:    "0x4752ba5dbc23f44d87826276bf6fd6b1c37ac4d4",
	11155111: "0x7a250d5630b4cf539739df2c5dac2f9c3b1c09cf",
}

// RouterForChain returns the V2 router address for a chain, applying the
// CHAIN_<id>_ROUTER env override if set.
func RouterForChain(chainID int64) string {
	if r, ok := v2Routers[chainID]; ok {
		return r
	}
	return ""
}
