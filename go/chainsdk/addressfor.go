package chainsdk

// addressfor.go — family -> address derivation router. Pure key derivation:
// it needs no database, no HTTP and no wallet state, so both wallet backends
// share this single implementation instead of each keeping a copy.

import (
	"errors"
	"fmt"
)

// AddressFor routes address derivation by family.
func AddressFor(family Family, ct string, chainID int64, seed []byte, path string, prefix byte) (string, error) {
	switch family {
	case FamilyUTXO:
		if ct == "zcash" {
			return ZECAddressFromSeed(seed, path)
		}
		return UTXOAddressFromSeed(seed, path, ct)
	case FamilyCosmos:
		addr, _, err := CosmosAddressFromSeedNew(seed, path, chainID)
		return addr, err
	case FamilySolana:
		pub, err := EdPubKey(seed, path)
		if err != nil {
			return "", err
		}
		return SolanaAddressFromKey(pub)
	case FamilyAptos:
		return AptosAddress(seed, path)
	case FamilySui:
		return SuiAddress(seed, path)
	case FamilyNear:
		return NearAddress(seed, path)
	case FamilyStellar:
		return StrKeyAddress(seed, path)
	case FamilyAlgorand:
		return AlgoAddress(seed, path)
	case FamilyNano:
		return NanoAddressFor(seed, path, ct)
	case FamilyGraphene:
		return GrapheneAddress(seed, path, ct)
	case FamilyNEM:
		return NEMAddress(seed, path, ct)
	case FamilySiacoin:
		return SiacoinAddress(seed, path)
	case FamilyMultiversX:
		return MultiversXAddress(seed, path)
	case FamilyStarknet:
		// A Starknet account address is a contract address whose class hash
		// selects the account implementation, so it is configured rather than
		// guessed; see StarknetAddressForChain.
		return StarknetAddressForChain(ct, seed, path)
	case FamilyMonero:
		return MoneroAddressFromSeed(seed, path)
	case FamilyStacks:
		return StacksAddress(seed, path)
	case FamilyMina:
		return MinaAddress(seed, path)
	case FamilyErgo:
		return ErgoP2PKAddress(seed, path)
	case FamilyNEO:
		return NEOAddress(seed, path)
	case FamilyArweave:
		// Arweave addresses come from an RSA-4096 public key, which needs a
		// generated key pair rather than an HD derivation; fail closed here
		// and expose ArweaveAddressFromModulus for callers that hold one.
		return "", errors.New("arweave: an address is Base64URL(SHA-256(512-byte RSA-4096 modulus)); derive it from the wallet key with ArweaveAddressFromModulus — no HD scheme exists to fabricate")
	case FamilyTezos:
		return TezosAddress(seed, path)
	case FamilyCardano:
		return CardanoAddress(seed, path)
	case FamilyWaves:
		return WavesAddress(seed, path)
	case FamilyTron:
		return TronAddress(seed, path)
	case FamilyVeChain:
		return VETAddress(seed, path)
	case FamilyRipple:
		return XRPAddress(seed, path)
	case FamilyICP:
		return ICPAddress(seed, path)
	case FamilyZilliqa:
		return ZilAddress(seed, path)
	case FamilyKaspa:
		return KaspaAddress(seed, path)
	case FamilyNervos:
		return NervosAddress(seed, path)
	case FamilyFilecoin:
		return FilAddress(seed, path)
	case FamilySubstrate:
		return SubstrateAddress(seed, prefix)
	case FamilyAleo, FamilyHedera, FamilyFlow:
		return "", NotFeasible(family)
	case FamilyTON:
		return "", errors.New("ton: wallet v4r2 state-init cell derivation is fail-closed (ed25519 message signing stays available)")
	}
	return "", fmt.Errorf("unknown non-EVM chain %q", ct)
}
