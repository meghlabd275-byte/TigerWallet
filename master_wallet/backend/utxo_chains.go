package main

// utxo_chains.go — UTXO-family broadcast parameters for the MasterWallet
// auto-signer. The legacy P2PKH sign path (mwBTCSignTx / buildSignBTCP2PKH,
// SIGHASH_ALL, esplora UTXO index) is identical across Bitcoin-derived chains;
// what differs per chain is exactly three things captured here:
//
//   - the esplora-compatible API base (UTXO index + raw-tx relay)
//   - the P2PKH address version byte
//   - the default BIP-44 derivation path
//
// Only chains whose consensus signing is byte-identical to legacy Bitcoin
// P2PKH are listed. Chains needing a different sighash scheme (Bitcoin Cash /
// BSV / eCash fork-id), different hashing (Groestlcoin), or different tx
// formats (Zcash overwinter/sapling, Qtum) are deliberately ABSENT — the
// router fails closed with an explicit error for them instead of producing a
// tx the network would reject. Endpoint bases are env-overridable so
// operators can point at their own esplora instances.

import (
        "os"
        "strings"
)

// utxoChainParams parameterizes one UTXO-family chain.
type utxoChainParams struct {
        name           string // canonical chain_type
        esploraBase    string // esplora API base (no trailing slash)
        p2pkhVersion   byte   // address version byte
        defaultDerive  string // default BIP-44 derivation path (%d = account)
        feeSat         int64  // flat fee in the chain's base units
}

// utxoChains is the supported UTXO family beyond which the router fails
// closed. Versions are the canonical mainnet P2PKH prefixes.
var utxoChains = map[string]utxoChainParams{
        "bitcoin": {
                name:           "bitcoin",
                esploraBase:    "https://blockstream.info/api",
                p2pkhVersion:   0x00,
                defaultDerive:  "m/44'/0'/0'/0/%d",
                feeSat:         1500,
        },
        "litecoin": {
                name:           "litecoin",
                esploraBase:    "https://litecoinspace.org/api",
                p2pkhVersion:   0x30, // 'L...' addresses
                defaultDerive:  "m/44'/2'/0'/0/%d",
                feeSat:         1500, // litoshis
        },
}

// utxoParamsFor resolves UTXO parameters by chain_type name or by numeric
// chain id from the seeded non-EVM registry. The esplora base is
// env-overridable per chain (BTC_ESPLORA_URL, LTC_ESPLORA_URL) for operators
// running their own indexers. ok=false means fail-closed (unsupported).
func utxoParamsFor(chainType string, chainID int64) (utxoChainParams, bool) {
        ct := strings.ToLower(strings.TrimSpace(chainType))
        if ct == "" && chainID != 0 {
                for _, c := range defaultNonEVMChains {
                        if c.ChainID == chainID {
                                ct = c.ChainType
                                break
                        }
                }
        }
        switch ct {
        case "btc":
                ct = "bitcoin"
        case "ltc":
                ct = "litecoin"
        }
        p, ok := utxoChains[ct]
        if !ok {
                return utxoChainParams{}, false
        }
        if env := strings.TrimSpace(os.Getenv(utxoEsploraEnvName(ct))); env != "" {
                p.esploraBase = strings.TrimRight(env, "/")
        }
        return p, true
}

// utxoEsploraEnvName maps a chain_type to its esplora endpoint env override.
func utxoEsploraEnvName(chainType string) string {
        switch chainType {
        case "bitcoin":
                return "BTC_ESPLORA_URL"
        case "litecoin":
                return "LTC_ESPLORA_URL"
        default:
                return ""
        }
}
