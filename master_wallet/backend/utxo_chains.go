package main

// utxo_chains.go — parameters for every UTXO-family chain the custodial
// MasterWallet can sweep.
//
// All chains below share one transaction format (legacy P2PKH with
// double-SHA256 SIGHASH_ALL) except the Bitcoin Cash descendants, which sign
// with BIP-143 SIGHASH_ALL|FORKID, and Zcash, which requires the Sapling v4 /
// ZIP-243 sighash. The signing itself lives in the shared chainsdk module
// (chainsdk.UTXOSign / chainsdk.ZECSign) so there is exactly one
// implementation of each sighash in the repository; this file only supplies
// the per-chain parameters:
//
//   - the address version prefix (1-2 bytes, canonical mainnet P2PKH)
//   - the indexer endpoint and protocol (see utxo_indexer.go)
//   - the BIP-44 derivation path template
//   - the fee policy (base units per byte + a floor)
//
// A chain absent from this table fails closed before any signing happens. A
// chain present here whose indexer cannot be reached also fails closed: the
// caller gets an error, never a fabricated txid.

import (
	"fmt"
	"os"
	"strings"
)

// utxoChainParams parameterizes one UTXO-family chain.
type utxoChainParams struct {
	name          string          // canonical chain_type
	indexerKind   utxoIndexerKind // explorer protocol
	indexerBase   string          // explorer API base (no trailing slash)
	p2pkhVersion  []byte          // address version prefix (1-2 bytes)
	defaultDerive string          // BIP-44 path template (%d = address index)
	feeRatePerB   int64           // fee in base units per serialized byte
	minFeeSat     int64           // fee floor in base units
	sighash       string          // "legacy" | "forkid" | "zcash"
}

// utxoChains is the supported UTXO family. Version prefixes are the canonical
// mainnet P2PKH values from each chain's own chainparams.
var utxoChains = map[string]utxoChainParams{
	"bitcoin": {
		name: "bitcoin", indexerKind: indexerEsplora,
		indexerBase: "https://blockstream.info/api", p2pkhVersion: []byte{0x00},
		defaultDerive: "m/44'/0'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1500, sighash: "legacy",
	},
	"litecoin": {
		name: "litecoin", indexerKind: indexerEsplora,
		indexerBase: "https://litecoinspace.org/api", p2pkhVersion: []byte{0x30},
		defaultDerive: "m/44'/2'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1500, sighash: "legacy",
	},
	"dogecoin": {
		name: "dogecoin", indexerKind: indexerDogechain,
		indexerBase: "https://dogechain.info", p2pkhVersion: []byte{0x1e},
		defaultDerive: "m/44'/3'/0'/0/%d", feeRatePerB: 100, minFeeSat: 100000000, sighash: "legacy",
	},
	"dash": {
		name: "dash", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.dash.org", p2pkhVersion: []byte{0x4c},
		defaultDerive: "m/44'/5'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1000, sighash: "legacy",
	},
	"bitcoincash": {
		name: "bitcoincash", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.bitcoincash.network", p2pkhVersion: []byte{0x00},
		defaultDerive: "m/44'/145'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1000, sighash: "forkid",
	},
	"bitcoinsv": {
		name: "bitcoinsv", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.bitcoinsv.io", p2pkhVersion: []byte{0x00},
		defaultDerive: "m/44'/236'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1000, sighash: "forkid",
	},
	"ecash": {
		name: "ecash", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.ecash.network", p2pkhVersion: []byte{0x00},
		defaultDerive: "m/44'/899'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1000, sighash: "forkid",
	},
	"raven": {
		name: "raven", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.ravencoin.org", p2pkhVersion: []byte{0x3c},
		defaultDerive: "m/44'/175'/0'/0/%d", feeRatePerB: 100, minFeeSat: 1000000, sighash: "legacy",
	},
	"zcash": {
		name: "zcash", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.zec.zelcore.io", p2pkhVersion: []byte{0x1c, 0xb8},
		defaultDerive: "m/44'/133'/0'/0/%d", feeRatePerB: 10, minFeeSat: 10000, sighash: "zcash",
	},
	"groestlcoin": {
		name: "groestlcoin", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.groestlcoin.org", p2pkhVersion: []byte{0x24},
		defaultDerive: "m/44'/17'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1500, sighash: "legacy",
	},
	"digibyte": {
		name: "digibyte", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.digibyte.io", p2pkhVersion: []byte{0x1e},
		defaultDerive: "m/44'/20'/0'/0/%d", feeRatePerB: 100, minFeeSat: 1000000, sighash: "legacy",
	},
	"qtum": {
		name: "qtum", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.qtum.org", p2pkhVersion: []byte{0x3a},
		defaultDerive: "m/44'/2301'/0'/0/%d", feeRatePerB: 40, minFeeSat: 100000, sighash: "legacy",
	},
	"verge": {
		name: "verge", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.vergecurrency.network", p2pkhVersion: []byte{0x1e},
		defaultDerive: "m/44'/77'/0'/0/%d", feeRatePerB: 100, minFeeSat: 1000000, sighash: "legacy",
	},
	"namecoin": {
		name: "namecoin", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.namecoin.org", p2pkhVersion: []byte{0x34},
		defaultDerive: "m/44'/7'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1500, sighash: "legacy",
	},
	"monacoin": {
		name: "monacoin", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.monacoin.jp", p2pkhVersion: []byte{0x32},
		defaultDerive: "m/44'/22'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1500, sighash: "legacy",
	},
	"blackcoin": {
		name: "blackcoin", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.blackcoin.nl", p2pkhVersion: []byte{0x19},
		defaultDerive: "m/44'/10'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1500, sighash: "legacy",
	},
	"komodo": {
		name: "komodo", indexerKind: indexerBlockbook,
		indexerBase: "https://blockbook.komodo.live", p2pkhVersion: []byte{0x3c},
		defaultDerive: "m/44'/141'/0'/0/%d", feeRatePerB: 5, minFeeSat: 1500, sighash: "legacy",
	},
}

// utxoAliases maps the shorthand chain_type spellings clients use onto the
// canonical table key.
var utxoAliases = map[string]string{
	"btc": "bitcoin", "ltc": "litecoin", "doge": "dogecoin",
	"bch": "bitcoincash", "bsv": "bitcoinsv", "xec": "ecash",
	"rvn": "raven", "zec": "zcash", "grs": "groestlcoin",
	"dgb": "digibyte", "qtum": "qtum", "xvg": "verge",
	"nmc": "namecoin", "mona": "monacoin", "blk": "blackcoin",
	"kmd": "komodo",
}

// canonicalUTXOType normalizes a chain_type (or resolves one from the seeded
// registry by chain id) to the utxoChains key. Empty means unknown.
func canonicalUTXOType(chainType string, chainID int64) string {
	ct := strings.ToLower(strings.TrimSpace(chainType))
	if ct == "" && chainID != 0 {
		for _, c := range defaultNonEVMChains {
			if c.ChainID == chainID {
				ct = strings.ToLower(c.ChainType)
				break
			}
		}
	}
	if alias, ok := utxoAliases[ct]; ok {
		ct = alias
	}
	return ct
}

// utxoParamsFor resolves UTXO parameters by chain_type name or by numeric
// chain id from the seeded non-EVM registry. ok=false means fail-closed
// (unsupported chain). The indexer endpoint is operator-overridable per chain
// via UTXO_INDEXER_URL_<NAME>, and via BTC_ESPLORA_URL / LTC_ESPLORA_URL for
// the two esplora chains.
func utxoParamsFor(chainType string, chainID int64) (utxoChainParams, bool) {
	ct := canonicalUTXOType(chainType, chainID)
	p, ok := utxoChains[ct]
	if !ok {
		return utxoChainParams{}, false
	}
	if env := strings.TrimSpace(os.Getenv(utxoIndexerEnvName(ct))); env != "" {
		p.indexerBase = strings.TrimRight(env, "/")
	}
	// Legacy per-chain overrides kept for deployments already using them.
	switch ct {
	case "bitcoin":
		if env := strings.TrimSpace(os.Getenv("BTC_ESPLORA_URL")); env != "" {
			p.indexerBase = strings.TrimRight(env, "/")
			p.indexerKind = indexerEsplora
		}
	case "litecoin":
		if env := strings.TrimSpace(os.Getenv("LTC_ESPLORA_URL")); env != "" {
			p.indexerBase = strings.TrimRight(env, "/")
			p.indexerKind = indexerEsplora
		}
	}
	return p, true
}

// utxoIndexerEnvName maps a canonical chain_type to its endpoint override var.
func utxoIndexerEnvName(chainType string) string {
	return "UTXO_INDEXER_URL_" + strings.ToUpper(chainType)
}

// utxoEsploraEnvName retains the historical helper name used by earlier
// deployments for the two esplora chains.
func utxoEsploraEnvName(chainType string) string {
	switch chainType {
	case "bitcoin":
		return "BTC_ESPLORA_URL"
	case "litecoin":
		return "LTC_ESPLORA_URL"
	default:
		return utxoIndexerEnvName(chainType)
	}
}

// indexer returns the bound indexer client for this chain.
func (p utxoChainParams) indexer() (utxoIndexer, error) {
	return indexerFor(p.indexerKind, p.indexerBase)
}

// feeForSize applies the chain's fee policy to an estimated transaction size.
// A real fee is always at least the chain's floor, so a low-rate chain cannot
// produce a transaction a node will reject for dust relay.
func (p utxoChainParams) feeForSize(estBytes int64) int64 {
	if estBytes < 0 {
		estBytes = 0
	}
	fee := p.feeRatePerB * estBytes
	if fee < p.minFeeSat {
		fee = p.minFeeSat
	}
	return fee
}

// supportedUTXOChains lists the canonical chain_types this build can sweep, in
// a stable order, for diagnostics and the fail-closed error message.
func supportedUTXOChains() []string {
	return []string{
		"bitcoin", "litecoin", "dogecoin", "dash", "bitcoincash", "bitcoinsv",
		"ecash", "raven", "zcash", "groestlcoin", "digibyte", "qtum", "verge",
		"namecoin", "monacoin", "blackcoin", "komodo",
	}
}

// errUnsupportedUTXO builds the fail-closed error naming the supported set.
func errUnsupportedUTXO(chainType string) error {
	return fmt.Errorf("UTXO chain %q has no signer in this build (supported: %s)",
		chainType, strings.Join(supportedUTXOChains(), ", "))
}
