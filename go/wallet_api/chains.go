package main

import (
	"fmt"
	"sort"
)

// ChainConfig describes a supported blockchain and its derivation path. It is
// the single canonical record used by every client and frontend (web, desktop,
// android, ios, extension) via GET /api/v1/chains. EVM chains (ChainType ==
// "evm") are usable for full wallet operations (RPC, balance, signing,
// broadcast); non-EVM chains are registered for discovery and address
// derivation — on-chain operations for them require their dedicated SDK.
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

// IsEVM reports whether the chain is an EVM chain (convenience accessor).
func (c ChainConfig) IsEVM() bool { return c.ChainType == "evm" }

// SupportedChains is the merged, mainnet-only EVM + non-EVM registry exposed to
// all clients and frontends. Built by initSupportedChains at package load.
// Admin overrides from PostgreSQL (admin_chain_config) are layered on top by
// applyAdminChainOverrides at startup/handler time (see admin_ext.go).
var SupportedChains map[int64]ChainConfig

func initSupportedChains() {
	if SupportedChains != nil {
		return
	}
	SupportedChains = make(map[int64]ChainConfig, evmMainnetCount+nonEVMMainnetCount)
	for _, c := range evmMainnet {
		c.IsTestnet = false
		SupportedChains[c.ID] = c
	}
	for _, c := range nonEVMMainnet {
		c.IsTestnet = false
		SupportedChains[c.ID] = c
	}
}

// chainByID returns a chain config or nil, applying env overrides for RPC and
// explorer. Every chain in the static registry is a verified mainnet.
func chainByID(id int64) *ChainConfig {
	initSupportedChains()
	c, ok := SupportedChains[id]
	if !ok {
		return nil
	}
	if v := envOr(fmt.Sprintf("CHAIN_%d_RPC", id), ""); v != "" {
		c.RPCEndpoint = v
	}
	if v := envOr(fmt.Sprintf("CHAIN_%d_EXPLORER", id), ""); v != "" {
		c.ExplorerAPI = v
	}
	return &c
}

// listSupportedChains returns all active chains sorted by chain id. It is the
// single source of truth for the public GET /api/v1/chains endpoint and is
// consumed by every frontend (web/desktop/android/ios/extension).
func listSupportedChains() []ChainConfig {
	initSupportedChains()
	out := make([]ChainConfig, 0, len(SupportedChains))
	for _, c := range SupportedChains {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// listChainsByType returns chains matching a chain_type ("evm","bitcoin",...).
func listChainsByType(ct string) []ChainConfig {
	initSupportedChains()
	out := make([]ChainConfig, 0)
	for _, c := range SupportedChains {
		if c.ChainType == ct {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// evmChainCount / nonEvmChainCount are used by admin dashboards + tests to
// verify the preinstalled chain minimums (>=100 EVM, >=50 non-EVM).
func evmChainCount() int {
	initSupportedChains()
	n := 0
	for _, c := range SupportedChains {
		if c.IsEVM() {
			n++
		}
	}
	return n
}

func nonEvmChainCount() int {
	initSupportedChains()
	n := 0
	for _, c := range SupportedChains {
		if !c.IsEVM() {
			n++
		}
	}
	return n
}

// evmChainByChainID returns an EVM chain config by its real EVM chain ID,
// or nil if the chain is not EVM-supported. Used by EVM-only operations
// (balance, signing, broadcast) which must not run against non-EVM chains.
func evmChainByChainID(id int64) *ChainConfig {
	c := chainByID(id)
	if c == nil || !c.IsEVM() {
		return nil
	}
	return c
}
