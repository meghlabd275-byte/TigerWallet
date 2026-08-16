package handlers

import (
	"os"

	"github.com/tigerwallet/wl-user-wallet/internal/chains"
)

// rpcForChain returns a public RPC endpoint for an EVM chain ID. Uses the
// canonical chain registry (internal/chains) as the source of truth and
// honors per-chain env overrides (CHAIN_<id>_RPC, ETH_RPC_URL for chain 1).
// Fail-closed: unknown/non-EVM chains return "" (caller 503s).
func rpcForChain(chainID int64) string {
	if chainID == 1 {
		if v := os.Getenv("ETH_RPC_URL"); v != "" {
			return v
		}
	}
	if c := chains.ChainByID(chainID); c != nil && c.IsEVM() && c.RPCEndpoint != "" {
		return c.RPCEndpoint
	}
	return ""
}

// explorerForChain returns the Etherscan-compatible explorer API base URL for
// a chain (tx history / NFT / receipt proxies), with env override
// CHAIN_<id>_EXPLORER. Fail-closed: "" if not configured.
func explorerForChain(chainID int64) string {
	if c := chains.ChainByID(chainID); c != nil {
		return c.ExplorerAPI
	}
	return ""
}
