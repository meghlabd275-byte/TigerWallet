// Package chains provides the canonical EVM + non-EVM chain registry for the
// standalone WL-UserWallet backend. Ported verbatim from the canonical
// TigerWallet wallet_api (chains_evm_data.go + chains_nonevm_data.go) so the
// rebranded mobile/desktop clients see the same 186-chain registry.
package chains

import (
	"fmt"
	"os"
	"sort"
)

// ChainConfig describes a supported blockchain and its derivation path. It is
// the single canonical record used by every client and frontend via
// GET /api/v1/chains. EVM chains (ChainType == "evm") are usable for full
// wallet operations (RPC, balance, signing, broadcast); non-EVM chains are
// registered for discovery and address derivation.
type ChainConfig struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Symbol        string   `json:"symbol"`
	RPCEndpoint   string   `json:"rpc_endpoint"`
	DerivationPath string  `json:"derivation_path"`
	ExplorerAPI   string   `json:"explorer_api"`
	ExplorerURL   string   `json:"explorer_url"`
	ChainType     string   `json:"chain_type"`
	Decimals      int      `json:"decimals"`
	CoinType      uint32   `json:"coin_type"`
	IsTestnet     bool     `json:"is_testnet"`
	RPCAlternates []string `json:"rpc_alternates,omitempty"`
	NativeID      int64    `json:"native_id,omitempty"`
}

// IsEVM reports whether the chain is an EVM chain.
func (c ChainConfig) IsEVM() bool { return c.ChainType == "evm" }

// SupportedChains is the merged mainnet-only registry. Built by
// initSupportedChains at package load.
var SupportedChains map[int64]ChainConfig

func initSupportedChains() {
	if SupportedChains != nil {
		return
	}
	SupportedChains = make(map[int64]ChainConfig, len(evmMainnet)+len(nonEVMMainnet))
	for _, c := range evmMainnet {
		c.IsTestnet = false
		SupportedChains[c.ID] = c
	}
	for _, c := range nonEVMMainnet {
		c.IsTestnet = false
		SupportedChains[c.ID] = c
	}
}

// ChainByID returns a chain config or nil, applying env overrides for RPC and
// explorer endpoints (CHAIN_<id>_RPC / CHAIN_<id>_EXPLORER).
func ChainByID(id int64) *ChainConfig {
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

// EVMChainByChainID returns an EVM chain config by its real EVM chain ID, or
// nil if the chain is not EVM-supported.
func EVMChainByChainID(id int64) *ChainConfig {
	c := ChainByID(id)
	if c == nil || !c.IsEVM() {
		return nil
	}
	return c
}

// ListSupportedChains returns all chains sorted by chain id.
func ListSupportedChains() []ChainConfig {
	initSupportedChains()
	out := make([]ChainConfig, 0, len(SupportedChains))
	for _, c := range SupportedChains {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ListChainsByType returns chains matching a chain_type ("evm","bitcoin",...).
func ListChainsByType(ct string) []ChainConfig {
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

// EVMChainCount / NonEVMChainCount are used by dashboards to verify the
// preinstalled chain minimums.
func EVMChainCount() int {
	initSupportedChains()
	n := 0
	for _, c := range SupportedChains {
		if c.IsEVM() {
			n++
		}
	}
	return n
}

func NonEVMChainCount() int {
	initSupportedChains()
	n := 0
	for _, c := range SupportedChains {
		if !c.IsEVM() {
			n++
		}
	}
	return n
}

func envOr(k, d string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return d
}
