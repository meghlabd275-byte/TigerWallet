package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// verifyPlatformChain checks that the configured TigerSmartChain endpoint really
// is the network the wallet is built for.
//
// The chain id is compiled into the registry: token addresses, the wrapped-TGR
// predeploy and every signed transaction assume 7710. The node, however, takes
// its chain id from TSC_CHAIN_ID, so an operator can start one on a different
// network and point TSC_RPC_URL at it. Nothing about that is visible until a
// user tries to send, and then it surfaces as a signing or broadcast error far
// from the cause. Reading eth_chainId once at startup turns a confusing
// first-send failure into a clear line in the log.
//
// This only reports. A mismatch is a configuration error worth shouting about,
// but refusing to start would take a working wallet offline over a chain that
// may not even be in use yet, and BroadcastTransaction already refuses to move
// funds onto the wrong network.
func verifyPlatformChain(ctx context.Context, endpoint string) {
	if strings.TrimSpace(endpoint) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	reported, err := FetchChainID(ctx, endpoint)
	if err != nil {
		log.Printf("WARNING: TigerSmartChain at %s did not answer eth_chainId: %v", endpoint, err)
		return
	}
	if reported.Int64() != tigerSmartChainID {
		log.Printf(
			"WARNING: TigerSmartChain endpoint %s reports chain %d but this wallet is built for chain %d; "+
				"transfers will be refused. Set TSC_CHAIN_ID=%d on the node, or TSC_RPC_URL to the network on chain %d.",
			endpoint, reported.Int64(), tigerSmartChainID, tigerSmartChainID, tigerSmartChainID)
		return
	}
	log.Printf("TigerSmartChain endpoint %s confirmed on chain %d", endpoint, tigerSmartChainID)
}

// platformChainEndpoint returns the TigerSmartChain RPC endpoint from the
// environment, preferring the dedicated variable over the generic per-chain one.
func platformChainEndpoint() string {
	if v := os.Getenv(tscRPCEnv); v != "" {
		return v
	}
	return os.Getenv(fmt.Sprintf("CHAIN_%d_RPC", tigerSmartChainID))
}
