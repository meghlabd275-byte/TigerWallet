package main

import (
	"fmt"

	"github.com/tigerwallet/chainsdk"
	"sort"
)

// ChainConfig is an alias for the shared chain record. It is defined once in
// chainsdk so the registry data and the SDK cannot drift apart; every client
// sees it via GET /api/v1/chains.
type ChainConfig = chainsdk.ChainConfig

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

// platformChainRPCEnv returns the dedicated environment variable holding the
// JSON-RPC endpoint for a chain the platform operates itself. These networks
// have no public URL to bundle in the registry, so the operator supplies one
// under a named variable rather than the generic CHAIN_<id>_RPC.
func platformChainRPCEnv(id int64) string {
	switch id {
	case tigerSmartChainID:
		return tscRPCEnv
	case tigerSmartChainTestnetID:
		return tscRPCEnvTestnet
	}
	return ""
}

// chainByID returns a chain config or nil, applying env overrides for RPC and
// explorer. Every chain in the static registry is a verified mainnet.
//
// The copy is returned by value: callers must never be able to mutate the
// process-wide SupportedChains map, which is read concurrently by every
// request handler.
func chainByID(id int64) *ChainConfig {
	initSupportedChains()
	c, ok := SupportedChains[id]
	if !ok {
		return nil
	}
	// The dedicated platform variable wins over the generic CHAIN_<id>_RPC: it is
	// the documented contract for a network the platform operates itself, and it
	// is the only one a single-node deployment sets. An operator who prefers the
	// generic form simply leaves the dedicated one unset.
	set := false
	if env := platformChainRPCEnv(id); env != "" {
		if v := envOr(env, ""); v != "" {
			c.RPCEndpoint = v
			set = true
		}
	}
	if !set {
		if v := envOr(fmt.Sprintf("CHAIN_%d_RPC", id), ""); v != "" {
			c.RPCEndpoint = v
		}
	}
	if v := envOr(fmt.Sprintf("CHAIN_%d_EXPLORER", id), ""); v != "" {
		c.ExplorerAPI = v
	}
	return &c
}

// listSupportedChains returns all active chains sorted by chain id. It is the
// single source of truth for the public GET /api/v1/chains endpoint and is
// consumed by every frontend (web/desktop/android/ios/extension).
//
// Each entry is resolved through chainByID so the operator's environment
// overrides (CHAIN_<id>_RPC, and the dedicated platform variables such as
// TSC_RPC_URL) are reflected in what clients receive. Without this a chain the
// platform operates itself would be advertised with an empty endpoint and
// every client would treat it as unreachable.
func listSupportedChains() []ChainConfig {
	initSupportedChains()
	out := make([]ChainConfig, 0, len(SupportedChains))
	for id := range SupportedChains {
		if c := chainByID(id); c != nil {
			out = append(out, *c)
		}
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
