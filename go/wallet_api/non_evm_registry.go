package main

// non_evm_registry.go — resolves every one of the 66 seeded non-EVM chains
// into exactly one SDK family for address derivation, message signing, and
// transaction building/broadcast. Any chain not routable to a real family
// fails closed with a descriptive error — we never fabricate an address,
// signature, or tx hash for an unknown chain.
//
// Families:
//   utxo_standard — legacy double-SHA256 SIGHASH_ALL P2PKH (BTC/LTC/DOGE/
//                   DASH/RVN/ZEC-legacy-disabled/GRS/DGB/QTUM/XVG/NMC/MONA/
//                   BLK/KMD). ZEC additionally supports ZIP-243 (Sapling v4).
//   utxo_forkid   — BIP143 + SIGHASH_FORKID (BCH/BSV/XEC).
//   cosmos        — bech32 prefix table + SIGN_MODE_DIRECT protobuf TxRaw.
//   ed25519_*     — account chains over Ed25519 slips/seeds (solana, aptos,
//                   sui, near, stellar/pi, algorand, nano, multiversx, tezos,
//                   waves, ton, cardano).
//   secp_*        — account chains over secp256k1 (tron, vechain, ripple,
//                   icp, zilliqa, kaspa, nervos, filecoin, hedera, flow).
//   substrate     — sr25519 (polkadot/kusama).

import (
	"fmt"
)

// nonEvmFamily enumerates the SDK families.
type nonEvmFamily string

// familyUnknown is the sentinel for unresolved chains.
const familyUnknown nonEvmFamily = "unknown"

const (
	familyUTXO           nonEvmFamily = "utxo"
	familyCosmos         nonEvmFamily = "cosmos"
	familySolana         nonEvmFamily = "solana"
	familyAptos          nonEvmFamily = "aptos"
	familySui            nonEvmFamily = "sui"
	familyNear           nonEvmFamily = "near"
	familyStellar        nonEvmFamily = "stellar" // stellar + pi
	familyAlgorand       nonEvmFamily = "algorand"
	familyNano           nonEvmFamily = "nano"
	familyMultiversX     nonEvmFamily = "multiversx"
	familyTezos          nonEvmFamily = "tezos"
	familyTON            nonEvmFamily = "ton"
	familyCardano        nonEvmFamily = "cardano"
	familyWaves          nonEvmFamily = "waves"
	familyTron           nonEvmFamily = "tron"
	familyVeChain        nonEvmFamily = "vechain"
	familyRipple         nonEvmFamily = "ripple"
	familyICP            nonEvmFamily = "internetcomputer"
	familyZilliqa        nonEvmFamily = "zilliqa"
	familyKaspa          nonEvmFamily = "kaspa"
	familyNervos         nonEvmFamily = "nervos"
	familyFilecoin       nonEvmFamily = "filecoin"
	familyAleo           nonEvmFamily = "aleo"
	familyHedera         nonEvmFamily = "hedera"
	familyFlow           nonEvmFamily = "flow"
	familySubstrate      nonEvmFamily = "substrate" // polkadot/kusama
	familyGraphene       nonEvmFamily = "graphene"  // steem/hive/bitshares
	familyNEM            nonEvmFamily = "nem"       // nem/symbol
	familySiacoin        nonEvmFamily = "siacoin"
	familyStarknet       nonEvmFamily = "starknet"
	familyMonero         nonEvmFamily = "monero"
	familyArweave        nonEvmFamily = "arweave"
	familyStacks         nonEvmFamily = "stacks"
	familyMina           nonEvmFamily = "mina"
	familyErgo           nonEvmFamily = "ergo"
	familyNEO            nonEvmFamily = "neo"
	// familyUnsupported marks registry chains whose address scheme needs
	// data this backend cannot obtain offline. They resolve to a named
	// family so the failure message names the chain, and every operation on
	// them fails closed instead of emitting a plausible-looking value.
	familyUnsupported    nonEvmFamily = "unsupported"
)

// utxoParams parameterizes one UTXO-family chain.
type utxoParams struct {
	p2pkhVersion []byte // address version (1-2 bytes)
	forkID       bool   // BCH-descendant BIP143 sighash
}

// utxoTable covers the 17 seeded UTXO chains. Versions are the canonical
// mainnet P2PKH prefixes (Groestlcoin reuses Bitcoin's double-SHA256
// sighash — Groestl is only its PoW hash, per the groestlcoin consensus
// libraries; it is listed in the standard family).
var utxoTable = map[string]utxoParams{
	"bitcoin":     {[]byte{0x00}, false},
	"litecoin":    {[]byte{0x30}, false},
	"dogecoin":    {[]byte{0x1e}, false},
	"dash":        {[]byte{0x4c}, false},
	"bitcoincash": {[]byte{0x00}, true},
	"bitcoinsv":   {[]byte{0x00}, true},
	"ecash":       {[]byte{0x00}, true},
	"raven":       {[]byte{0x3c}, false},
	"zcash":       {[]byte{0x1c, 0xb8}, false},
	"groestlcoin": {[]byte{0x24}, false},
	"digibyte":    {[]byte{0x1e}, false},
	"qtum":        {[]byte{0x3a}, false},
	"verge":       {[]byte{0x1e}, false},
	"namecoin":    {[]byte{0x34}, false},
	"monacoin":    {[]byte{0x32}, false},
	"blackcoin":   {[]byte{0x19}, false},
	"komodo":      {[]byte{0x3c}, false},
	// Added with the registry expansion. Every version byte below is taken from
	// the chain's own chainparams, not from a third-party list:
	//   peercoin/peercoin, feathercoin/feathercoin,
	//   gridcoin-community/Gridcoin-Research, ZencashOfficial/zen,
	//   syscoin/syscoin, lbryio/lbrycrd, BTCGPU/BTCGPU.
	"peercoin":     {[]byte{0x37}, false},
	"feathercoin":  {[]byte{0x0e}, false},
	"gridcoin":     {[]byte{0x3e}, false},
	"horizen":      {[]byte{0x20, 0x89}, false},
	"syscoin":      {[]byte{0x3f}, false},
	"lbry":         {[]byte{0x55}, false},
	"bitcoingold":  {[]byte{0x26}, false},
	"stratis":      {[]byte{0x3f}, false},
	"omni":         {[]byte{0x00}, false},
	"counterparty": {[]byte{0x00}, false},
	// Decred mainnet PubKeyHashAddrID is the two-byte prefix 0x073f
	// (dcrd chaincfg/mainnetparams.go). Primecoin uses the single byte 23
	// (primecoin src/base58.h, "starts with A").
	"decred":    {[]byte{0x07, 0x3f}, false},
	"primecoin": {[]byte{0x17}, false},
}

// cosmosIDPrefix resolves the bech32 HRP for the 23 Cosmos-family chains by
// registry chain id (their ChainType is identical, "cosmos", in the seeded
// registry, so id-based lookup is required).
var cosmosIDPrefix = map[int64]string{
	9000000118: "cosmos",
	9000000529: "secret",
	9000000330: "terra",
	9000073068: "inj",
	9000014648: "celestia",
	9000026317: "osmo",
	9000049823: "dydx",
	9000073741: "sei",
	9000009016: "akash",
	9000041857: "kujira",
	9000012099: "stride",
	9000090063: "neutron",
	9000005267: "juno",
	9000007183: "akash",
	9000018759: "persistence",
	9000034677: "evmos",
	9000054841: "canto",
	9000003318: "kava",
	9000062954: "cro",
	9000016892: "stars",
	9000021252: "saga",
	9000086660: "noble",
	9000040572: "axelar",
	9000007153: "umee",
	// Added with the registry expansion; every HRP is taken from the chain's own
	// entry in cosmos/chain-registry.
	9000006000: "thor",
	9000006900: "fetch",
	9000007200: "band",
	9000007500: "chihuahua",
	9000007600: "agoric",
	9000007700: "regen",
	9000007800: "somm",
	9000007900: "mantra",
}

// cosmosPrefixByID returns the bech32 HRP for a cosmos chain id; falls back to
// "cosmos" for a passed-by-name request.
func cosmosPrefixByID(id int64) string {
	if id == 0 {
		return "cosmos"
	}
	if p, ok := cosmosIDPrefix[id]; ok && p != "" {
		return p
	}
	return "cosmos"
}

// cosmosFamilyIDs lists the registry ids claimed by the cosmos family.
var cosmosFamilyIDs = map[int64]bool{}
func init() {
	for id := range cosmosIDPrefix {
		cosmosFamilyIDs[id] = true
	}
}

// nonEvmResolve maps a chain_type (case-insensitive) to its SDK family. The
// caller may also pass a registry chain id; when provided it wins.
func nonEvmResolve(chainType string, chainID int64) (nonEvmFamily, string, error) {
	ct := normChainType(chainType)
	if chainID != 0 {
		if cosmosFamilyIDs[chainID] {
			return familyCosmos, "cosmos", nil
		}
		for _, c := range nonEVMMainnet {
			if c.ID == chainID {
				ct = normChainType(c.ChainType)
				break
			}
		}
	}
	if _, ok := utxoTable[ct]; ok {
		return familyUTXO, ct, nil
	}
	switch ct {
	case "cosmos":
		return familyCosmos, ct, nil

	case "solana":
		return familySolana, ct, nil
	case "aptos":
		return familyAptos, ct, nil
	case "sui":
		return familySui, ct, nil
	case "near":
		return familyNear, ct, nil
	case "stellar", "pi":
		return familyStellar, ct, nil
	case "algorand":
		return familyAlgorand, ct, nil
	case "nano", "banano":
		return familyNano, ct, nil
	case "elrond", "multiversx":
		return familyMultiversX, "multiversx", nil
	case "tezos":
		return familyTezos, ct, nil
	case "ton":
		return familyTON, ct, nil
	case "cardano":
		return familyCardano, ct, nil
	case "waves":
		return familyWaves, ct, nil
	case "tron":
		return familyTron, ct, nil
	case "vechain":
		return familyVeChain, ct, nil
	case "ripple":
		return familyRipple, ct, nil
	case "internetcomputer", "icp":
		return familyICP, "internetcomputer", nil
	case "zilliqa":
		return familyZilliqa, ct, nil
	case "kaspa":
		return familyKaspa, ct, nil
	case "nervos":
		return familyNervos, ct, nil
	case "filecoin":
		return familyFilecoin, ct, nil
	case "aleo":
		return familyAleo, ct, nil
	case "hedera":
		return familyHedera, ct, nil
	case "flow":
		return familyFlow, ct, nil
	case "polkadot", "kusama", "substrate":
		return familySubstrate, ct, nil
	case "steem", "hive", "bitshares":
		return familyGraphene, ct, nil
	case "nem", "symbol":
		return familyNEM, ct, nil
	case "siacoin":
		return familySiacoin, ct, nil

	case "starknet":
		return familyStarknet, ct, nil
	case "monero":
		return familyMonero, ct, nil
	case "arweave":
		return familyArweave, ct, nil
	case "stacks":
		return familyStacks, ct, nil
	case "mina":
		return familyMina, ct, nil
	case "ergo":
		return familyErgo, ct, nil
	case "neo":
		return familyNEO, ct, nil
	case "wakt":
		return familyNano, ct, nil
	case "akash":
		return familyCosmos, "akash", nil

	// Registry chains whose address scheme is not derivable offline. They
	// resolve to familyUnsupported so the error names the chain; no caller
	// ever receives a fabricated address, signature, or transaction.
	case "hyperliquid", "supra", "iota", "kadena", "chia", "handshake",
		"lisk", "nexus", "ark", "aion", "nuls", "pascalcoin", "signum":
		return familyUnsupported, ct, nil
	}
	return familyUnknown, ct, fmt.Errorf("unsupported non-EVM chain %q — route fails closed rather than fabricating a result", ct)
}

func normChainType(s string) string {
	lower := ""
	for _, b := range []byte(s) {
		if b == ' ' || b == '-' {
			continue
		}
		if b >= 'A' && b <= 'Z' {
			b += 32
		}
		lower += string(b)
	}
	// normalize registry-held aliases
	switch lower {
	case "injective":
		return "inj"
	case "terra classic", "terraclassic":
		return "terra"
	case "pi", "pinetwork":
		return "pi"
	case "neo n3", "neo3", "neo legacy":
		return "neo"
	case "signum (burstcoin)", "burstcoin":
		return "signum"
	}
	return lower
}
