package main

// utxo_indexer.go — UTXO indexer abstraction.
//
// Every UTXO-family chain needs two operations from a block explorer: list the
// unspent outputs of an address, and relay a signed raw transaction. Three
// protocols are in use across the seeded chains and each is implemented here
// behind one interface, so the signing path never branches on the provider:
//
//   esplora   — GET  /address/<addr>/utxo        POST /tx            (BTC, LTC)
//   blockbook — GET  /api/v2/utxo/<addr>         POST /api/v2/sendtx (RVN, ZEC, ...)
//   dogechain — GET  /api/v1/address/unspent/<a> POST /api/v1/tx/send (DOGE)
//
// Endpoints are operator-overridable per chain (UTXO_INDEXER_URL_<NAME>, plus
// the legacy BTC_ESPLORA_URL / LTC_ESPLORA_URL). A chain whose indexer cannot
// answer fails closed: the caller receives an error and never a fabricated
// txid or an empty UTXO set that would be mistaken for a zero balance.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// utxoIndexerKind identifies the explorer protocol a chain's endpoint speaks.
type utxoIndexerKind string

const (
	indexerEsplora   utxoIndexerKind = "esplora"
	indexerBlockbook utxoIndexerKind = "blockbook"
	indexerDogechain utxoIndexerKind = "dogechain"
)

// utxoIndexerClient is shared by every indexer implementation.
var utxoIndexerClient = &http.Client{Timeout: 20 * time.Second}

// utxoIndexer is the address-index + relay surface used by the UTXO signer.
type utxoIndexer interface {
	// ListUTXOs returns the confirmed unspent outputs paying the address, in
	// base units. An empty slice means the address genuinely has no confirmed
	// UTXOs; a failure to reach the indexer is an error.
	ListUTXOs(address string) ([]btcUTXO, error)
	// Broadcast relays a signed raw transaction (hex) and returns the on-chain
	// transaction id.
	Broadcast(rawTxHex string) (string, error)
}

// indexerFor returns the indexer bound to a chain's endpoint.
func indexerFor(kind utxoIndexerKind, base string) (utxoIndexer, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil, fmt.Errorf("no indexer endpoint configured")
	}
	if _, err := url.Parse(base); err != nil {
		return nil, fmt.Errorf("invalid indexer endpoint %q: %w", base, err)
	}
	switch kind {
	case indexerEsplora:
		return &esploraIndexer{base: base}, nil
	case indexerBlockbook:
		return &blockbookIndexer{base: base}, nil
	case indexerDogechain:
		return &dogechainIndexer{base: base}, nil
	default:
		return nil, fmt.Errorf("unknown UTXO indexer kind %q", kind)
	}
}

// ---------------------------------------------------------------------------
// esplora (blockstream.info, litecoinspace.org, self-hosted esplora)
// ---------------------------------------------------------------------------

type esploraIndexer struct{ base string }

func (e *esploraIndexer) ListUTXOs(address string) ([]btcUTXO, error) {
	body, err := httpGet(e.base + "/address/" + url.PathEscape(address) + "/utxo")
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, nil
	}
	var raw []struct {
		TxID   string `json:"txid"`
		Vout   int    `json:"vout"`
		Value  int64  `json:"value"`
		Status struct {
			Confirmed bool `json:"confirmed"`
		} `json:"status"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("esplora utxo decode: %w", err)
	}
	out := make([]btcUTXO, 0, len(raw))
	for _, r := range raw {
		if !r.Status.Confirmed {
			continue
		}
		out = append(out, btcUTXO{TxID: r.TxID, Vout: uint32(r.Vout), Value: big.NewInt(r.Value)})
	}
	return out, nil
}

func (e *esploraIndexer) Broadcast(rawTxHex string) (string, error) {
	return postPlainText(utxoIndexerClient, e.base+"/tx", rawTxHex, 4096)
}

// ---------------------------------------------------------------------------
// blockbook (trezor blockbook: ravencoin, zcash, and operator deployments)
// ---------------------------------------------------------------------------

type blockbookIndexer struct{ base string }

func (b *blockbookIndexer) ListUTXOs(address string) ([]btcUTXO, error) {
	body, err := httpGet(b.base + "/api/v2/utxo/" + url.PathEscape(address))
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, nil
	}
	// Blockbook reports an invalid address as {"error": "..."} with HTTP 200.
	var asErr struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(trimmed, &asErr); err == nil && asErr.Error != "" {
		return nil, fmt.Errorf("blockbook rejected address: %s", asErr.Error)
	}
	var raw []struct {
		TxID     string `json:"txid"`
		Vout     uint32 `json:"vout"`
		Value    string `json:"value"`
		Height   int64  `json:"height"`
		Coinbase bool   `json:"coinbase"`
	}
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return nil, fmt.Errorf("blockbook utxo decode: %w", err)
	}
	out := make([]btcUTXO, 0, len(raw))
	for _, r := range raw {
		// height <= 0 means the output is still in the mempool and cannot be
		// spent by a new transaction yet.
		if r.Height <= 0 {
			continue
		}
		v, err := strconv.ParseInt(r.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("blockbook value %q: %w", r.Value, err)
		}
		out = append(out, btcUTXO{TxID: r.TxID, Vout: r.Vout, Value: big.NewInt(v)})
	}
	return out, nil
}

func (b *blockbookIndexer) Broadcast(rawTxHex string) (string, error) {
	// Blockbook's relay is POST /api/v2/sendtx (with a trailing slash accepted
	// by every public deployment) returning {"result": "<txid>"}.
	body, err := postPlainText(utxoIndexerClient, b.base+"/api/v2/sendtx/", rawTxHex, 4096)
	if err != nil {
		return "", err
	}
	var ok struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &ok); err != nil {
		return "", fmt.Errorf("blockbook relay decode: %w", err)
	}
	if ok.Result == "" {
		return "", fmt.Errorf("blockbook relay returned no txid")
	}
	return ok.Result, nil
}

// ---------------------------------------------------------------------------
// dogechain.info (the only public Dogecoin address index that lists UTXOs)
// ---------------------------------------------------------------------------

type dogechainIndexer struct{ base string }

func (d *dogechainIndexer) ListUTXOs(address string) ([]btcUTXO, error) {
	body, err := httpGet(d.base + "/api/v1/address/unspent/" + url.PathEscape(address))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Success int `json:"success"`
		Unspent []struct {
			TxHash string `json:"tx_hash"`
			TxOut  int    `json:"tx_output_n"`
			Value  string `json:"value"`
		} `json:"unspent_outputs"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("dogechain utxo decode: %w", err)
	}
	if resp.Success != 1 {
		return nil, fmt.Errorf("dogechain rejected address %q", address)
	}
	out := make([]btcUTXO, 0, len(resp.Unspent))
	for _, u := range resp.Unspent {
		v, err := strconv.ParseInt(u.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("dogechain value %q: %w", u.Value, err)
		}
		out = append(out, btcUTXO{TxID: u.TxHash, Vout: uint32(u.TxOut), Value: big.NewInt(v)})
	}
	return out, nil
}

func (d *dogechainIndexer) Broadcast(rawTxHex string) (string, error) {
	form := url.Values{}
	form.Set("tx", rawTxHex)
	req, err := http.NewRequest(http.MethodPost, d.base+"/api/v1/tx/send",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("dogechain relay request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := utxoIndexerClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("dogechain relay: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("dogechain relay rejected (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		TxHash  string `json:"tx_hash"`
		Success int    `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("dogechain relay decode: %w", err)
	}
	if out.Error != "" {
		return "", fmt.Errorf("dogechain relay rejected: %s", out.Error)
	}
	if out.TxHash == "" {
		return "", fmt.Errorf("dogechain relay returned no tx_hash")
	}
	return out.TxHash, nil
}

// postPlainText issues a text/plain POST and returns the trimmed response body.
func postPlainText(client *http.Client, endpoint, payload string, limit int64) (string, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBufferString(payload))
	if err != nil {
		return "", fmt.Errorf("build relay request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("relay: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, limit))
	text := strings.TrimSpace(string(body))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("relay rejected (HTTP %d): %s", resp.StatusCode, text)
	}
	if text == "" {
		return "", fmt.Errorf("relay returned an empty response")
	}
	return text, nil
}
