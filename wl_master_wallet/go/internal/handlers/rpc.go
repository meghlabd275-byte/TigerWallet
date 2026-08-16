package handlers

// rpcForChain returns a public RPC endpoint for common EVM chain IDs. The WL
// client can override chain 1 via ETH_RPC_URL env. Fail-closed: unknown chains
// return "" (caller 503s).
import "os"

var rpcByChain = map[int64]string{
	1:     firstNonEmpty(os.Getenv("ETH_RPC_URL"), "https://ethereum-rpc.publicnode.com"),
	56:    "https://bsc-dataseed.binance.org",
	137:   "https://polygon-rpc.publicnode.com",
	42161: "https://arbitrum-one-rpc.publicnode.com",
	10:    "https://optimism-rpc.publicnode.com",
	8453:  "https://base-rpc.publicnode.com",
}

func rpcForChain(chainID int64) string {
	if r, ok := rpcByChain[chainID]; ok {
		return r
	}
	return ""
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
