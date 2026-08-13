package main

// chains.go — MasterWallet chain configuration. A curated, mainnet-only set of
// EVM chains with real RPC env-var resolution (no fabricated endpoints). Each
// chain maps to a BIP-44 coin type + derivation path. The full 120+66 chain
// registry lives in the canonical go/wallet_api; this is the subset the
// MasterWallet treasury/operator wallet needs for its managed chains.

// ChainConfig describes one supported chain for the master wallet.
type ChainConfig struct {
	ChainID        int64  `json:"chain_id"`
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	Blockchain     string `json:"blockchain"` // canonical name in DB enum
	RPCEnv         string `json:"rpc_env"`    // env var holding RPC URL
	Decimals       int    `json:"decimals"`
	DerivationPath string `json:"derivation_path"`
	IsEVM          bool   `json:"is_evm"`
}

// supportedChains is the curated mainnet chain list. RPC endpoints are
// resolved from env vars at runtime (fail-closed when unset).
var supportedChains = []ChainConfig{
	{ChainID: 1, Name: "Ethereum", Symbol: "ETH", Blockchain: "ethereum", RPCEnv: "ETH_RPC_URL", Decimals: 18, DerivationPath: "m/44'/60'/0'/0/0", IsEVM: true},
	{ChainID: 56, Name: "BNB Smart Chain", Symbol: "BNB", Blockchain: "bsc", RPCEnv: "BSC_RPC_URL", Decimals: 18, DerivationPath: "m/44'/60'/0'/0/0", IsEVM: true},
	{ChainID: 137, Name: "Polygon", Symbol: "POL", Blockchain: "polygon", RPCEnv: "POLYGON_RPC_URL", Decimals: 18, DerivationPath: "m/44'/60'/0'/0/0", IsEVM: true},
	{ChainID: 42161, Name: "Arbitrum One", Symbol: "ETH", Blockchain: "arbitrum", RPCEnv: "ARBITRUM_RPC_URL", Decimals: 18, DerivationPath: "m/44'/60'/0'/0/0", IsEVM: true},
	{ChainID: 10, Name: "Optimism", Symbol: "ETH", Blockchain: "optimism", RPCEnv: "OPTIMISM_RPC_URL", Decimals: 18, DerivationPath: "m/44'/60'/0'/0/0", IsEVM: true},
	{ChainID: 43114, Name: "Avalanche C-Chain", Symbol: "AVAX", Blockchain: "avalanche", RPCEnv: "AVALANCHE_RPC_URL", Decimals: 18, DerivationPath: "m/44'/60'/0'/0/0", IsEVM: true},
	{ChainID: 8453, Name: "Base", Symbol: "ETH", Blockchain: "base", RPCEnv: "BASE_RPC_URL", Decimals: 18, DerivationPath: "m/44'/60'/0'/0/0", IsEVM: true},
	{ChainID: 42220, Name: "Celo", Symbol: "CELO", Blockchain: "celo", RPCEnv: "CELO_RPC_URL", Decimals: 18, DerivationPath: "m/44'/60'/0'/0/0", IsEVM: true},
	{ChainID: 250, Name: "Fantom", Symbol: "FTM", Blockchain: "fantom", RPCEnv: "FANTOM_RPC_URL", Decimals: 18, DerivationPath: "m/44'/60'/0'/0/0", IsEVM: true},
	{ChainID: 25, Name: "Cronos", Symbol: "CRO", Blockchain: "cronos", RPCEnv: "CRONOS_RPC_URL", Decimals: 18, DerivationPath: "m/44'/60'/0'/0/0", IsEVM: true},
}

// chainByID looks up a chain by id.
func chainByID(chainID int64) (ChainConfig, bool) {
	for _, c := range supportedChains {
		if c.ChainID == chainID {
			return c, true
		}
	}
	return ChainConfig{}, false
}

// chainByBlockchain looks up a chain by its canonical blockchain name.
func chainByBlockchain(name string) (ChainConfig, bool) {
	name = normalizeChain(name)
	for _, c := range supportedChains {
		if normalizeChain(c.Blockchain) == name {
			return c, true
		}
	}
	return ChainConfig{}, false
}

// rpcEndpointForChain resolves the RPC URL for a chain id from env.
func rpcEndpointForChain(chainID int64) string {
	if c, ok := chainByID(chainID); ok {
		return chainRPCEndpoint(c.ChainID)
	}
	return ""
}
