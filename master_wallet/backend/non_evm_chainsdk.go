package main

// non_evm_chainsdk.go — custodial broadcast for the non-EVM families the
// hand-written MasterWallet signers did not cover.
//
// MasterWallet originally broadcast only EVM, Bitcoin/Litecoin (UTXO),
// Solana and Cosmos. Every other seeded non-EVM chain failed closed, which is
// the wrong outcome for a custodial product: a user with a Tron or Cardano
// balance could not be swept.
//
// The chain builders live in github.com/tigerwallet/chainsdk, the shared
// module also used by go/wallet_api. There is exactly one implementation per
// chain — MasterWallet calls it rather than keeping a second copy.
//
// Amount convention: chainsdk builders take BASE UNITS (lamports, sun,
// satoshi, yoctoNEAR, lovelace, drops, ...). transactions.amount is stored at
// wei-scale, so amounts are converted here with the chain's decimals before
// the builder is called. A chain whose decimals are unknown fails closed
// rather than guessing a scale.

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/tigerwallet/chainsdk"
)

// chainsdkBroadcastTimeout bounds a single build+sign+broadcast round trip.
const chainsdkBroadcastTimeout = 45 * time.Second

// weiScaleToBaseUnits converts an amount stored at wei scale (18 decimals)
// to the chain's base units. Returns an error when the conversion is not
// exact, so a rounding artifact can never silently change the amount that
// gets signed.
//
// Two chains in the registry (NEAR, 24 decimals; Hedera, 30) are finer than
// wei scale, so the conversion multiplies instead of dividing there.
func weiScaleToBaseUnits(amount string, decimals int) (*big.Int, error) {
	v, ok := new(big.Int).SetString(strings.TrimSpace(amount), 10)
	if !ok {
		return nil, fmt.Errorf("amount %q is not an integer", amount)
	}
	if v.Sign() < 0 {
		return nil, fmt.Errorf("amount %q is negative", amount)
	}
	if decimals < 0 || decimals > 30 {
		return nil, fmt.Errorf("unsupported decimals %d", decimals)
	}
	ten := big.NewInt(10)
	delta := 18 - decimals
	scale := new(big.Int).Exp(ten, big.NewInt(int64(absInt(delta))), nil)
	if delta >= 0 {
		out, rem := new(big.Int).QuoRem(v, scale, new(big.Int))
		if rem.Sign() != 0 {
			return nil, fmt.Errorf("amount %q is not representable in %d decimals (remainder %s)", amount, decimals, rem)
		}
		return out, nil
	}
	return new(big.Int).Mul(v, scale), nil
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// chainDecimals resolves the registry decimals for a non-EVM chain id.
func chainDecimals(chainID int64) (int, bool) {
	for _, c := range defaultNonEVMChains {
		if c.ChainID == chainID {
			return c.Decimals, true
		}
	}
	return 0, false
}

// chainsdkEndpoint resolves the RPC/API endpoint for a chain: operator env
// override first (so a private node can be pinned), then the shared registry.
func chainsdkEndpoint(chainID int64, chainType string) string {
	if ep := svcNonEVMEnvEndpoint(chainType); ep != "" {
		return ep
	}
	for _, c := range defaultNonEVMChains {
		if c.ChainID == chainID && c.RPCURL != "" {
			return c.RPCURL
		}
	}
	return ""
}

// svcNonEVMEnvEndpoint lets an operator pin a private endpoint per family.
// Empty means "use the registry", never a fabricated URL.
func svcNonEVMEnvEndpoint(chainType string) string {
	switch strings.ToLower(chainType) {
	case "tron":
		return getEnvDefault("TRON_RPC_URL", "")
	case "cardano":
		return getEnvDefault("CARDANO_RPC_URL", "")
	case "near":
		return getEnvDefault("NEAR_RPC_URL", "")
	case "aptos":
		return getEnvDefault("APTOS_RPC_URL", "")
	case "sui":
		return getEnvDefault("SUI_RPC_URL", "")
	case "ripple":
		return getEnvDefault("XRP_RPC_URL", "")
	case "stellar":
		return getEnvDefault("STELLAR_RPC_URL", "")
	case "algorand":
		return getEnvDefault("ALGO_RPC_URL", "")
	case "nano":
		return getEnvDefault("NANO_RPC_URL", "")
	case "tezos":
		return getEnvDefault("TEZOS_RPC_URL", "")
	case "multiversx", "elrond":
		return getEnvDefault("MULTIVERSX_RPC_URL", "")
	case "vechain":
		return getEnvDefault("VET_RPC_URL", "")
	}
	return ""
}

// autoSignChainSDK broadcasts a transfer on a chain whose builder lives in the
// shared chainsdk module. Every path signs a real transaction and broadcasts
// it to the real network; a failure is returned as an error, never as a
// fabricated hash.
func (svc *Service) autoSignChainSDK(seed []byte, req *AutoSignRequest, chainType string) (string, string, error) {
	if strings.TrimSpace(req.ToAddress) == "" {
		return "", "failed", fmt.Errorf("%s transfer requires to_address", chainType)
	}
	decimals, ok := chainDecimals(req.ChainID)
	if !ok {
		return "", "failed", fmt.Errorf("chain %d is not in the non-EVM registry; refusing to guess a decimals scale", req.ChainID)
	}
	baseUnits, err := weiScaleToBaseUnits(req.Value, decimals)
	if err != nil {
		return "", "failed", err
	}
	endpoint := chainsdkEndpoint(req.ChainID, chainType)
	if endpoint == "" {
		return "", "failed", fmt.Errorf("%s: no RPC endpoint configured (set the family's *_RPC_URL env or seed the chain registry)", chainType)
	}

	ctx, cancel := context.WithTimeout(context.Background(), chainsdkBroadcastTimeout)
	defer cancel()

	path := req.DerivationPath
	if path == "" {
		path = defaultDerivationForChain(req.ChainID)
	}

	var txHash string
	switch strings.ToLower(chainType) {
	case "tron":
		_, txHash, err = chainsdk.TronBuildSend(ctx, seed, path, endpoint, req.ToAddress, baseUnits.Uint64(), true)
	case "aptos":
		_, txHash, err = chainsdk.AptosBuildSend(ctx, seed, path, endpoint, req.ToAddress, baseUnits.Uint64(), 0, 0, 100, 0, 0, true)
	case "sui":
		_, txHash, err = chainsdk.SuiBuildSend(ctx, seed, path, endpoint, req.ToAddress, baseUnits.Uint64(), true)
	case "near":
		_, txHash, err = chainsdk.NearBuildSend(ctx, seed, path, endpoint, req.ToAddress, baseUnits, chainsdk.NearParams{}, true)
	case "algorand":
		_, txHash, err = chainsdk.AlgoBuildSend(ctx, seed, path, endpoint, "", req.ToAddress, chainsdk.AlgoBuildRequest{Amount: baseUnits.Uint64()}, true)
	case "stellar":
		_, txHash, err = chainsdk.StellarBuildSend(ctx, seed, path, endpoint, chainType, req.ToAddress, baseUnits.Int64(), 0, 100, true)
	case "ripple":
		_, txHash, err = chainsdk.RippleBuildSend(ctx, seed, path, endpoint, req.ToAddress, baseUnits.Uint64(), nil, true)
	case "vechain":
		_, txHash, err = chainsdk.VETBuildSend(ctx, seed, path, endpoint, req.ToAddress, baseUnits.Uint64(), 0, true)
	case "tezos":
		_, txHash, err = chainsdk.TezosBuildSend(ctx, seed, path, endpoint, req.ToAddress, baseUnits.Uint64(), 0, 0, 0, 0, true)
	case "multiversx", "elrond":
		_, txHash, err = chainsdk.MultiversXBuildSend(ctx, seed, path, endpoint, req.ToAddress, baseUnits.String(), true)
	default:
		return "", "failed", fmt.Errorf("non-EVM chain %q has no broadcast builder in the shared SDK; fail-closed rather than fabricating a hash", chainType)
	}
	if err != nil {
		return "", "broadcast_failed", err
	}
	if txHash == "" {
		return "", "failed", fmt.Errorf("%s broadcast returned no transaction hash", chainType)
	}
	return txHash, "broadcast", nil
}

// defaultDerivationForChain resolves the seeded BIP-44 path for a chain id.
func defaultDerivationForChain(chainID int64) string {
	for _, c := range defaultNonEVMChains {
		if c.ChainID == chainID && c.DerivationPath != "" {
			return c.DerivationPath
		}
	}
	return ""
}
