package main

// non_evm_accounts.go — address derivation and per-family constants for the
// account-model non-EVM families.
//
// The derivation itself lives in the shared chainsdk module; this file maps a
// chain_type onto the right entry point so the custodial path and the
// non-custodial wallet derive the same address from the same seed. There is
// exactly one implementation of each curve/encoding in the repository.

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/tigerwallet/chainsdk"
)

// Per-family transaction constants. Each is the value the chain's own
// documentation prescribes for a plain native transfer; they are named rather
// than inlined so a fee change is a one-line edit.
const (
	// vetTransferGas is the gas limit for a single empty-data VET clause.
	vetTransferGas = 9000
	// vetMainnetChainTag is the last byte of the VeChain mainnet genesis
	// block id (0x...ec1b4a), which the protocol requires in every tx body.
	vetMainnetChainTag = 0x4a

	// aptosMaxGas / aptosGasPrice bound an APT transfer in octas.
	aptosMaxGas   = 2000
	aptosGasPrice = 100

	// stellarBaseFee is 100 stroops — the network base fee per operation.
	stellarBaseFee = 100

	// tezosRevealFee covers the manager-key reveal plus the transfer. A fresh
	// implicit account must reveal before it can send, so the fee is paid on
	// the reveal operation in the same forge.
	tezosRevealFee = 3000
	// tezosGasLimit / tezosStorageLimit are the standard transfer limits.
	tezosGasLimit     = 15400
	tezosStorageLimit = 257
)

// chainsdkAddressFor derives the family's address for the given derivation
// path. An unsupported chain_type fails closed.
func chainsdkAddressFor(chainType string, seed []byte, path string) (string, error) {
	switch strings.ToLower(chainType) {
	case "tron":
		return chainsdk.TronAddress(seed, path)
	case "sui":
		return chainsdk.SuiAddress(seed, path)
	case "ripple", "xrp":
		return chainsdk.XRPAddress(seed, path)
	case "vechain":
		return chainsdk.VETAddress(seed, path)
	case "aptos":
		return chainsdk.AptosAddress(seed, path)
	case "near":
		return chainsdk.NearAddress(seed, path)
	case "algorand":
		return chainsdk.AlgoAddress(seed, path)
	case "stellar", "pi":
		return chainsdk.StrKeyAddress(seed, path)
	case "tezos":
		return chainsdk.TezosAddress(seed, path)
	case "multiversx", "elrond":
		return chainsdk.MultiversXAddress(seed, path)
	default:
		return "", fmt.Errorf("no address derivation for non-EVM chain %q", chainType)
	}
}

// nearPublicKey formats the account's ed25519 public key the way NEAR's RPC
// expects it ("ed25519:<hex>").
func nearPublicKey(seed []byte, path string) string {
	pub, err := chainsdk.EdPubKey(seed, path)
	if err != nil {
		return ""
	}
	return "ed25519:" + hex.EncodeToString(pub[:32])
}

// algoToken returns the algod API token, if the operator pinned a private
// node. Public algod providers need no token.
func algoToken() string {
	return strings.TrimSpace(os.Getenv("ALGOD_TOKEN"))
}

// tezosManagerKey returns the account's revealed manager key, or empty when
// the account has never been revealed. A transaction from an unrevealed
// account requires a reveal operation in the same batch, which this build does
// not forge, so the caller fails closed with an actionable message instead of
// broadcasting an operation the network will reject.
func tezosManagerKey(ctx context.Context, endpoint, address string) (string, error) {
	u := strings.TrimRight(endpoint, "/") + "/chains/main/blocks/head/context/contracts/" +
		url.PathEscape(address) + "/manager_key"
	raw, err := httpGetCtx(ctx, u)
	if err != nil {
		// A 404 here means "not revealed", which is a normal state, not a
		// transport failure; anything else is a real error.
		if strings.Contains(err.Error(), "HTTP 404") {
			return "", nil
		}
		return "", fmt.Errorf("tezos manager key: %w", err)
	}
	return strings.Trim(strings.TrimSpace(string(raw)), "\""), nil
}
