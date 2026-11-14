package chainsdk

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

func Hex0x(b []byte) string { return "0x" + hex.EncodeToString(b) }

// btcSignMessage signs a message with the secp256k1 key (r||s, no recovery).
func BTCMessageSign(seed []byte, derivationPath string, message []byte) (sig, pub []byte, err error) {
	priv, err := HDDerive(seed, derivationPath)
	if err != nil {
		return nil, nil, err
	}
	full, err := crypto.Sign(message, priv)
	if err != nil {
		return nil, nil, err
	}
	return full[:64], crypto.CompressPubkey(&priv.PublicKey), nil
}

// SecpMessageSign signs a message with the secp256k1 key using the EIP-191
// personal-sign prefix (65-byte r||s||v). Tron, VeChain and other secp chains
// verify this exact construction, so it lives here rather than being rebuilt
// per caller.
func SecpMessageSign(seed []byte, derivationPath string, message []byte) (sig, pub []byte, err error) {
	priv, err := HDDerive(seed, derivationPath)
	if err != nil {
		return nil, nil, err
	}
	h := crypto.Keccak256Hash([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)))
	full, err := crypto.Sign(h.Bytes(), priv)
	if err != nil {
		return nil, nil, err
	}
	return full, crypto.CompressPubkey(&priv.PublicKey), nil
}

func ParseUintOr(s string) uint64 {
	if s == "" {
		return 0
	}
	v, _ := new(big.Int).SetString(s, 10)
	if v == nil {
		return 0
	}
	return v.Uint64()
}

func ParseInt64Or(s string) int64 {
	v := ParseUintOr(s)
	return int64(v)
}

func DefaultDenomOf(chainID int64, ct string) string {
	switch chainID {
	case 9000000118: // Cosmos Hub
		return "uatom"
	case 9000026317: // Osmosis
		return "uosmo"
	case 9000073068: // Injective
		return "inj"
	case 9000012099: // Stride
		return "ustrd"
	case 9000000529: // Secret
		return "uscrt"
	case 9000007183: // Akash
		return "uakt"
	case 9000090063: // Neutron
		return "untrn"
	case 9000073741: // Sei
		return "usei"
	case 9000014648: // Celestia
		return "utia"
	case 9000049823: // dYdX
		return "adydx"
	case 9000005267: // Juno
		return "ujuno"
	case 9000041857: // Kujira
		return "ukuji"
	case 9000000330: // Terra Classic
		return "uluna"
	case 9000018759: // Persistence
		return "uxprt"
	case 9000000160: // Evmos
		return "aevmos"
	case 9000000900: // Kava
		return "ukava"
	case 9000001839: // Cronos
		return "basecro"
	case 9000007017: // Axelar
		return "uaxl"
	case 9000008102: // Band
		return "uband"
	case 9000000438: // IRIS
		return "uiris"
	case 9000008899: // Gravity Bridge
		return "ugraviton"
	case 9000001183: // Stargaze
		return "ustars"
	case 9000002236: // Sommelier
		return "usomm"
	}
	// Fallback: lower-cased chain-type (correct for most SDK chains, where
	// the base denom starts with 'u' — call sites can always pass {denom}).
	return "u" + ct
}

func LookupEndpoint(chainID int64, ct string) string {
	for _, c := range NonEVMMainnet {
		if c.ID == chainID {
			return c.RPCEndpoint
		}
	}
	if ct == "pi" {
		return "" // pi endpoint operator-configured (env PI_RPC)
	}
	switch ct {
	case "solana":
		return "https://api.mainnet-beta.solana.com"
	case "aptos":
		return "https://fullnode.mainnet.aptoslabs.com"
	case "sui":
		return "https://fullnode.mainnet.sui.io:443"
	case "near":
		return "https://rpc.mainnet.near.org"
	case "algorand":
		return "https://mainnet-api.algonode.cloud"
	case "nano":
		return "https://proxy.nanos.cc"
	case "tron":
		return "https://api.trongrid.io"
	case "vechain":
		return "https://sync-mainnet.vechain.org"
	case "ripple":
		return "https://s1.ripple.com:51234"
	case "stellar", "pi":
		return "https://api.mainnet.stellar.org"
	case "tezos":
		return "https://mainnet.tezos.ecadinfra.com"
	case "elrond":
		return "https://api.multiversx.com"
	}
	return ""
}
