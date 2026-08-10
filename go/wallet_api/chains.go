package main

import (
	"fmt"
)

// ChainConfig describes a supported EVM chain and its derivation path.
type ChainConfig struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	RPCEndpoint    string `json:"rpc_endpoint"`
	DerivationPath string `json:"derivation_path"`
	ExplorerAPI    string `json:"explorer_api"`
}

// SupportedChains is the canonical chain registry. RPC endpoints are public
// community endpoints; override via env CHAIN_<ID>_RPC for production.
var SupportedChains = map[int64]ChainConfig{
	1: {
		ID: 1, Name: "Ethereum", Symbol: "ETH",
		RPCEndpoint:    "https://eth.llamarpc.com",
		DerivationPath: "m/44'/60'/0'/0/0",
		ExplorerAPI:    "https://api.etherscan.io/api",
	},
	11155111: {
		ID: 11155111, Name: "Sepolia", Symbol: "ETH",
		RPCEndpoint:    "https://rpc.sepolia.org",
		DerivationPath: "m/44'/60'/0'/0/0",
		ExplorerAPI:    "https://api-sepolia.etherscan.io/api",
	},
	56: {
		ID: 56, Name: "BNB Smart Chain", Symbol: "BNB",
		RPCEndpoint:    "https://bsc-dataseed.binance.org",
		DerivationPath: "m/44'/60'/0'/0/0",
		ExplorerAPI:    "https://api.bscscan.com/api",
	},
	137: {
		ID: 137, Name: "Polygon", Symbol: "MATIC",
		RPCEndpoint:    "https://polygon-rpc.com",
		DerivationPath: "m/44'/60'/0'/0/0",
		ExplorerAPI:    "https://api.polygonscan.com/api",
	},
	42161: {
		ID: 42161, Name: "Arbitrum One", Symbol: "ETH",
		RPCEndpoint:    "https://arb1.arbitrum.io/rpc",
		DerivationPath: "m/44'/60'/0'/0/0",
		ExplorerAPI:    "https://api.arbiscan.io/api",
	},
	10: {
		ID: 10, Name: "Optimism", Symbol: "ETH",
		RPCEndpoint:    "https://mainnet.optimism.io",
		DerivationPath: "m/44'/60'/0'/0/0",
		ExplorerAPI:    "https://api-optimistic.etherscan.io/api",
	},
	8453: {
		ID: 8453, Name: "Base", Symbol: "ETH",
		RPCEndpoint:    "https://mainnet.base.org",
		DerivationPath: "m/44'/60'/0'/0/0",
		ExplorerAPI:    "https://api.basescan.org/api",
	},
}

// chainByID returns a chain config or nil.
func chainByID(id int64) *ChainConfig {
	if c, ok := SupportedChains[id]; ok {
		// allow env override of RPC
		if v := envOr(fmt.Sprintf("CHAIN_%d_RPC", id), ""); v != "" {
			c.RPCEndpoint = v
		}
		if v := envOr(fmt.Sprintf("CHAIN_%d_EXPLORER", id), ""); v != "" {
			c.ExplorerAPI = v
		}
		return &c
	}
	return nil
}
