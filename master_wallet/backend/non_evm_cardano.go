package main

// non_evm_cardano.go — custodial Cardano (ADA) transfers.
//
// Cardano needs three things this backend must read from the network rather
// than invent: the set of unspent outputs held by the sender, the protocol
// minimum fee (which is a function of the serialized transaction size), and a
// submit endpoint. UTXOs come from a Blockfrost-compatible indexer, the
// transaction is built and signed by the shared chainsdk, and the fee is
// computed from the real serialized size by building once and re-building only
// when the estimate was too low. Nothing here is guessed: an endpoint that is
// not configured, or an indexer that cannot answer, fails the transfer closed.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/tigerwallet/chainsdk"
)

// Cardano ledger fee constants (mainnet protocol parameters).
const (
	cardanoFeeCoefficient = 155381 // a: fixed component, lovelace
	cardanoFeePerByte     = 44     // b: per-byte component, lovelace
	// cardanoMinUTXO is the ledger's minimum lovelace for a change output; a
	// smaller remainder is not worth an output and is left to the fee.
	cardanoMinUTXO = 1000000
)

// cardanoIndexerURL resolves the Blockfrost-compatible indexer. The seeded
// registry value is the public default; an operator can point at a private
// instance.
func cardanoIndexerURL() string {
	if v := strings.TrimSpace(os.Getenv("CARDANO_BLOCKFROST_URL")); v != "" {
		return v
	}
	for _, c := range defaultNonEVMChains {
		if c.ChainType == "cardano" && c.RPCURL != "" {
			return c.RPCURL
		}
	}
	return ""
}

// cardanoSubmitURL resolves the cardano-submit-api base. It is a different
// service from the indexer, so it has its own setting and no default: a
// deployment that has not configured it fails closed instead of posting a
// signed transaction at an address that does not exist.
func cardanoSubmitURL() string {
	return strings.TrimSpace(os.Getenv("CARDANO_SUBMIT_URL"))
}

// cardanoProjectID returns the indexer API key, when the provider requires one.
func cardanoProjectID() string {
	return strings.TrimSpace(os.Getenv("CARDANO_BLOCKFROST_PROJECT_ID"))
}

// cardanoUTXO is one unspent output as the indexer reports it.
type cardanoUTXO struct {
	TxHash      string          `json:"tx_hash"`
	OutputIndex uint32          `json:"output_index"`
	Amount      []cardanoAmount `json:"amount"`
}

// cardanoAmount is one entry of a UTXO's value: a unit (policy id + asset name,
// or the literal "lovelace" for ADA) and its integer quantity.
type cardanoAmount struct {
	Unit     string `json:"unit"`
	Quantity string `json:"quantity"`
}

// cardanoUTXOIsPureLovelace reports whether an output carries ADA only. A
// UTXO holding a native asset must never be spent by a plain transfer: the
// asset would not be carried to an output and would be destroyed.
func cardanoUTXOIsPureLovelace(r cardanoUTXO) bool {
	return len(r.Amount) == 1 && r.Amount[0].Unit == "lovelace"
}

// cardanoUTXOsFor reads the unspent outputs of an address. Only the lovelace
// entry is used: a UTXO carrying native assets is skipped, because spending it
// without carrying the assets forward would destroy them.
func cardanoUTXOsFor(ctx context.Context, indexerURL, projectID, address string) ([]chainsdk.CardanoUTXO, error) {
	req := strings.TrimRight(indexerURL, "/") + "/addresses/" + url.PathEscape(address) + "/utxos"
	raw, err := httpGetWithHeaders(ctx, req, map[string]string{"project_id": projectID})
	if err != nil {
		return nil, fmt.Errorf("cardano utxo lookup: %w", err)
	}
	var rows []cardanoUTXO
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("cardano utxo decode: %w", err)
	}
	out := make([]chainsdk.CardanoUTXO, 0, len(rows))
	for _, r := range rows {
		if !cardanoUTXOIsPureLovelace(r) {
			continue // holds native assets: never spend it in a plain ADA transfer
		}
		lovelace, err := strconv.ParseUint(strings.TrimSpace(r.Amount[0].Quantity), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cardano utxo %s#%d amount %q: %w", r.TxHash, r.OutputIndex, r.Amount[0].Quantity, err)
		}
		if r.TxHash == "" {
			return nil, fmt.Errorf("cardano indexer returned a utxo without a tx hash")
		}
		out = append(out, chainsdk.CardanoUTXO{
			TxHash:     r.TxHash,
			Index:      r.OutputIndex,
			Amount:     lovelace,
			AddressB32: address,
		})
	}
	return out, nil
}

// cardanoFeeForSize is the ledger's minimum fee for a transaction of the given
// serialized size.
func cardanoFeeForSize(sizeBytes int) uint64 {
	return cardanoFeeCoefficient + cardanoFeePerByte*uint64(sizeBytes)
}

// autoSignCardano signs and broadcasts an ADA transfer. Input selection, the
// change output and the fee are all derived from the indexer's UTXO set and
// the real serialized size; any missing input is an error, never a fabricated
// hash.
func (svc *Service) autoSignCardano(seed []byte, req *AutoSignRequest, chainType string) (string, string, error) {
	if strings.TrimSpace(req.ToAddress) == "" {
		return "", "failed", fmt.Errorf("cardano transfer requires to_address")
	}
	decimals, ok := chainDecimals(req.ChainID)
	if !ok {
		return "", "failed", fmt.Errorf("chain %d is not in the non-EVM registry; refusing to guess a decimals scale", req.ChainID)
	}
	baseUnits, err := weiScaleToBaseUnits(req.Value, decimals)
	if err != nil {
		return "", "failed", err
	}
	if !baseUnits.IsUint64() {
		return "", "failed", fmt.Errorf("cardano amount exceeds the ledger's 64-bit lovelace range")
	}
	amount := baseUnits.Uint64()
	if amount < cardanoMinUTXO {
		return "", "failed", fmt.Errorf("cardano amount %d lovelace is below the %d lovelace minimum output", amount, cardanoMinUTXO)
	}

	indexer := cardanoIndexerURL()
	if indexer == "" {
		return "", "failed", fmt.Errorf("cardano: no indexer configured (set CARDANO_BLOCKFROST_URL)")
	}
	submitURL := cardanoSubmitURL()
	if submitURL == "" {
		return "", "failed", fmt.Errorf("cardano: no submit endpoint configured (set CARDANO_SUBMIT_URL to a cardano-submit-api base)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), chainsdkBroadcastTimeout)
	defer cancel()

	path := req.DerivationPath
	if path == "" {
		path = defaultDerivationForChain(req.ChainID)
	}
	fromAddr, err := chainsdk.CardanoAddress(seed, path)
	if err != nil {
		return "", "failed", fmt.Errorf("cardano address derivation: %w", err)
	}

	utxos, err := cardanoUTXOsFor(ctx, indexer, cardanoProjectID(), fromAddr)
	if err != nil {
		return "", "failed", err
	}
	if len(utxos) == 0 {
		return "", "failed", fmt.Errorf("cardano: no spendable UTXO at %s", fromAddr)
	}

	// Deterministic largest-first selection: the smallest set of inputs that
	// covers the amount plus a conservative fee. The fee is recomputed from the
	// real serialized size below and the transaction is rebuilt if it grew.
	inputs, total := selectCardanoInputs(utxos, amount+cardanoMinUTXO)
	if total < amount {
		return "", "failed", fmt.Errorf("cardano: spendable %d lovelace cannot cover %d lovelace", total, amount)
	}

	// build assembles the output set for a given fee. A remainder below the
	// ledger's minimum output size is not worth an output and is absorbed by the
	// fee instead of being emitted and rejected by the node.
	build := func(fee uint64) ([]chainsdk.CardanoOutput, error) {
		if total < amount+fee {
			return nil, fmt.Errorf("cardano: inputs %d do not cover %d + fee %d", total, amount, fee)
		}
		outputs := []chainsdk.CardanoOutput{{Address: req.ToAddress, Amount: amount}}
		if change := total - amount - fee; change >= cardanoMinUTXO {
			outputs = append(outputs, chainsdk.CardanoOutput{Address: fromAddr, Amount: change})
		}
		return outputs, nil
	}

	// The ledger's minimum fee depends on the serialized size, which in turn
	// depends on the fee. Iterate until the fee covers the size of the
	// transaction actually built with it; the loop converges in one or two
	// passes because only a CBOR integer-width boundary can change the size.
	fee := cardanoFeeForSize(0)
	var signed string
	for attempt := 0; attempt < 4; attempt++ {
		outputs, err := build(fee)
		if err != nil {
			return "", "failed", err
		}
		signed, _, err = chainsdk.CardanoBuildSend(ctx, seed, path, submitURL, inputs, outputs, fee, false)
		if err != nil {
			return "", "failed", fmt.Errorf("cardano build: %w", err)
		}
		required := cardanoFeeForSize(len(signed) / 2)
		if required <= fee {
			break
		}
		fee = required
	}

	outputs, err := build(fee)
	if err != nil {
		return "", "failed", err
	}
	_, txHash, err := chainsdk.CardanoBuildSend(ctx, seed, path, submitURL, inputs, outputs, fee, true)
	if err != nil {
		return "", "broadcast_failed", fmt.Errorf("cardano broadcast: %w", err)
	}
	if txHash == "" {
		return "", "failed", fmt.Errorf("cardano broadcast returned no transaction hash")
	}
	return txHash, "broadcast", nil
}

// selectCardanoInputs picks inputs largest-first until the target is covered,
// returning the chosen set and its total. The order is deterministic so a
// retry selects the same inputs.
func selectCardanoInputs(utxos []chainsdk.CardanoUTXO, target uint64) ([]chainsdk.CardanoUTXO, uint64) {
	sorted := make([]chainsdk.CardanoUTXO, len(utxos))
	copy(sorted, utxos)
	// Insertion sort by descending amount: the UTXO set per address is small and
	// this avoids pulling in a sort dependency for a stable, obvious order.
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].Amount > sorted[j-1].Amount; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	var chosen []chainsdk.CardanoUTXO
	var total uint64
	for _, u := range sorted {
		if total >= target {
			break
		}
		chosen = append(chosen, u)
		total += u.Amount
	}
	return chosen, total
}
