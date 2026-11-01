package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

// rpcFailover maps a chain's primary RPC endpoint to its alternates. It is built
// once from the chain registry so every call site inherits failover without
// having to thread the whole ChainConfig through its signature.
var (
	rpcFailover     map[string][]string
	rpcFailoverOnce sync.Once
)

func initRPCFailover() {
	rpcFailoverOnce.Do(func() {
		initSupportedChains()
		rpcFailover = make(map[string][]string, len(SupportedChains))
		for _, c := range SupportedChains {
			if c.RPCEndpoint == "" || len(c.RPCAlternates) == 0 {
				continue
			}
			rpcFailover[c.RPCEndpoint] = c.RPCAlternates
		}
	})
}

// endpointsFor returns the endpoint to try first followed by its alternates,
// with duplicates removed. An endpoint with no alternates yields just itself, so
// behaviour is unchanged for chains that only have one node.
func endpointsFor(endpoint string) []string {
	if endpoint == "" {
		return nil
	}
	initRPCFailover()
	alts := rpcFailover[endpoint]
	if len(alts) == 0 {
		return []string{endpoint}
	}
	out := make([]string, 0, len(alts)+1)
	seen := map[string]bool{endpoint: true}
	out = append(out, endpoint)
	for _, a := range alts {
		if a != "" && !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	return out
}

// isRetryableRPCError reports whether a failure is worth retrying on a different
// endpoint. A JSON-RPC error means a node answered the request: a revert, a bad
// parameter or an unsupported method is identical on every node, so retrying
// would only hide the real error behind a slower failure. Rate limits and
// transport faults are node-specific and are worth a second endpoint.
func isRetryableRPCError(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr rpc.Error
	if errors.As(err, &rpcErr) {
		switch rpcErr.ErrorCode() {
		case -32005, // limit exceeded
			-32016, // request rate limited
			-32603, // internal error, frequently a overloaded node
			429:
			return true
		}
		return false
	}
	return true
}

// rpcCall runs fn against each endpoint for the chain until one succeeds. The
// first non-retryable error is returned as-is so the caller sees the node's own
// message rather than a generic "all endpoints failed".
func rpcCall(ctx context.Context, endpoint string, fn func(*rpc.Client) error) error {
	eps := endpointsFor(endpoint)
	if len(eps) == 0 {
		return errors.New("no RPC endpoint configured")
	}
	var lastErr error
	for _, ep := range eps {
		client, err := rpc.DialContext(ctx, ep)
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", ep, err)
			continue
		}
		err = fn(client)
		client.Close()
		if err == nil {
			return nil
		}
		if !isRetryableRPCError(err) {
			return err
		}
		lastErr = fmt.Errorf("%s: %w", ep, err)
	}
	return lastErr
}
