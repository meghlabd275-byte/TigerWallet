package chainsdk

// ChainConfig is the canonical chain record shared by every backend and
// client. Kept identical to the wallet_api definition so the registry data
// and the SDK agree on every field.

type ChainConfig struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	Symbol         string   `json:"symbol"`
	RPCEndpoint    string   `json:"rpc_endpoint"`
	DerivationPath string   `json:"derivation_path"`
	ExplorerAPI    string   `json:"explorer_api"`
	ExplorerURL    string   `json:"explorer_url"`
	ChainType      string   `json:"chain_type"` // "evm" | "bitcoin" | "solana" | ...
	Decimals       int      `json:"decimals"`
	CoinType       uint32   `json:"coin_type"` // BIP-44 SLIP-44 registered coin type
	IsTestnet      bool     `json:"is_testnet"`
	RPCAlternates  []string `json:"rpc_alternates,omitempty"`
	NativeID       int64    `json:"native_id,omitempty"` // native network chain id (non-EVM)
}

// IsEVM reports whether the chain is an EVM chain.
func (c ChainConfig) IsEVM() bool { return c.ChainType == "evm" }
