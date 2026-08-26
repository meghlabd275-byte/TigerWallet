package main

// non_evm_broadcast.go — network broadcast of signed non-EVM transactions.
// Real HTTP, no fakes. Bitcoin raw txs are submitted to the public
// blockstream.info relay; the returned txid is the on-chain transaction id.

import (
        "bytes"
        "fmt"
        "io"
        "net/http"
        "time"
)

// broadcastHTTPClient is shared by the non-EVM broadcast helpers.
var broadcastHTTPClient = &http.Client{Timeout: 20 * time.Second}

// broadcastBitcoinTx submits a signed raw Bitcoin transaction (hex) to the
// public blockstream.info relay and returns the transaction id. The relay is a
// public esplora-compatible endpoint (no auth); the signed tx is the caller's
// own and carries no secrets beyond the public network.
func broadcastBitcoinTx(rawTxHex string) (string, error) {
        url := "https://blockstream.info/api/tx"
        req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(rawTxHex))
        if err != nil {
                return "", fmt.Errorf("build broadcast request: %w", err)
        }
        req.Header.Set("Content-Type", "text/plain")
        resp, err := broadcastHTTPClient.Do(req)
        if err != nil {
                return "", fmt.Errorf("broadcast: %w", err)
        }
        defer resp.Body.Close()
        body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
        if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
                return "", fmt.Errorf("broadcast rejected (HTTP %d): %s", resp.StatusCode, string(body))
        }
        txid := string(bytes.TrimSpace(body))
        if txid == "" {
                return "", fmt.Errorf("broadcast returned empty txid")
        }
        return txid, nil
}