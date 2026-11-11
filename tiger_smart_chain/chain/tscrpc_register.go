package chain

import (
	"github.com/ethereum/go-ethereum/rpc"
)

// TSCNamespace is the JSON-RPC namespace of the TigerSmartChain native API.
const TSCNamespace = "tsc"

// registerTSCRPC exposes the TigerSmartChain native namespace on the node's
// JSON-RPC and WebSocket endpoints.
func (n *Node) registerTSCRPC() error {
	n.stack.RegisterAPIs([]rpc.API{{
		Namespace: TSCNamespace,
		Version:   "1.0",
		Service:   &TSCAPI{node: n},
		Public:    true,
	}})
	return nil
}
