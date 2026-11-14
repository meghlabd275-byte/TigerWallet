package main

// non_evm_broadcast.go — shared HTTP client for the non-EVM broadcast paths.
//
// The UTXO families relay through the pluggable indexer in utxo_indexer.go;
// the account-chain families (cosmos, solana, ...) share this client so a
// stalled upstream cannot hold a signing worker indefinitely.

import (
	"net/http"
	"time"
)

// broadcastHTTPClient is shared by the non-EVM broadcast helpers.
var broadcastHTTPClient = &http.Client{Timeout: 20 * time.Second}
