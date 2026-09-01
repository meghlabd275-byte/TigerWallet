package main

// finance_addresses.go — deterministic per-user deposit addresses.
//
// Derivation: HKDF-SHA256(ikm = WALLET_MASTER_SEED, salt = versioned app
// salt, info = network|user_id) -> 32-byte account seed -> BIP-44 account 0
// per chain. The same user always gets the same address per network; no
// per-address storage or private-key persistence is required (the backend
// can re-derive at any time; the master seed never leaves the server).
//
// Encodings per family (all real, reused from the non-EVM SDK layer):
//   - EVM (ETH/BNB/MATIC): keccak256 of the uncompressed secp256k1 pubkey
//   - BTC: bech32 P2WPKH (BIP-173/141)
//   - LTC/DOGE: base58check P2PKH with the chain version byte
//   - TRX: base58check with the 0x41 Tron prefix over the keccak address
//   - SOL: base58 of the ed25519 public key
//
// Fail-closed: WALLET_MASTER_SEED unset => 503, never a random address.

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/hkdf"
)

type depositChainSpec struct {
	Asset   string   `json:"asset"`
	Network string   `json:"network"`
	Kind    string   `json:"-"` // evm | bech32 | base58 | tron | solana
	Path    string   `json:"-"` // BIP-44 path template (%d = account index)
	Version []byte   `json:"-"` // base58 P2PKH version
	HRP     string   `json:"-"` // bech32 human-readable part
	Assets  []string `json:"assets"`
}

var financeDepositChains = []depositChainSpec{
	{Asset: "BTC", Network: "Bitcoin", Kind: "bech32", Path: "m/44'/0'/%d'/0/0", HRP: "bc", Assets: []string{"BTC"}},
	{Asset: "ETH", Network: "Ethereum (ERC-20)", Kind: "evm", Path: "m/44'/60'/%d'/0/0", Assets: []string{"ETH", "USDT", "USDC"}},
	{Asset: "BNB", Network: "BNB Smart Chain (BEP-20)", Kind: "evm", Path: "m/44'/714'/%d'/0/0", Assets: []string{"BNB"}},
	{Asset: "MATIC", Network: "Polygon", Kind: "evm", Path: "m/44'/966'/%d'/0/0", Assets: []string{"MATIC"}},
	{Asset: "SOL", Network: "Solana", Kind: "solana", Path: "m/44'/501'/%d'/0/0", Assets: []string{"SOL"}},
	{Asset: "TRX", Network: "TRON (TRC-20)", Kind: "tron", Path: "m/44'/195'/%d'/0/0", Assets: []string{"TRX", "USDT"}},
	{Asset: "LTC", Network: "Litecoin", Kind: "base58", Path: "m/44'/2'/%d'/0/0", Version: []byte{0x30}, Assets: []string{"LTC"}},
	{Asset: "DOGE", Network: "Dogecoin", Kind: "base58", Path: "m/44'/3'/%d'/0/0", Version: []byte{0x1e}, Assets: []string{"DOGE"}},
}

// deriveDepositAddress deterministically derives the user's deposit address
// for one network. Pure function of (master seed, network, user id).
func deriveDepositAddress(userID uuid.UUID, spec depositChainSpec) (string, error) {
	if len(financeCfg.masterSeed) == 0 {
		return "", errors.New("WALLET_MASTER_SEED is not configured")
	}
	info := spec.Network + "|" + userID.String()
	hk := hkdf.New(sha256.New, financeCfg.masterSeed,
		[]byte("tigerwallet-finance-deposit-v1"), []byte(info))
	seed := make([]byte, 32)
	if _, err := io.ReadFull(hk, seed); err != nil {
		return "", err
	}
	switch spec.Kind {
	case "solana":
		priv := ed25519.NewKeyFromSeed(seed)
		pub, ok := priv.Public().(ed25519.PublicKey)
		if !ok || len(pub) != 32 {
			return "", errors.New("solana key derivation failed")
		}
		return base58Encode(pub), nil
	case "evm", "bech32", "base58", "tron":
		priv, err := hdDerive(seed, fmt.Sprintf(spec.Path, 0))
		if err != nil {
			return "", err
		}
		switch spec.Kind {
		case "evm":
			return crypto.PubkeyToAddress(priv.PublicKey).Hex(), nil
		case "tron":
			pub := crypto.FromECDSAPub(&priv.PublicKey) // 65 bytes, 0x04 prefix
			h := crypto.Keccak256(pub[1:])
			payload := append([]byte{0x41}, h[len(h)-20:]...)
			return base58checkEncode(payload), nil
		default: // bech32 / base58 — hash160 of the compressed pubkey
			_, btPub := btcec.PrivKeyFromBytes(crypto.FromECDSA(priv))
			pkh := hash160(btPub.SerializeCompressed())
			if spec.Kind == "bech32" {
				conv, err := convertBits(pkh, 8, 5, true)
				if err != nil {
					return "", err
				}
				return bech32Encode(spec.HRP, append([]byte{0x00}, conv...))
			}
			return base58checkEncode(append(append([]byte{}, spec.Version...), pkh...)), nil
		}
	}
	return "", fmt.Errorf("unsupported address kind %q", spec.Kind)
}

// depositURI builds the standard payment URI embedded in the QR code.
func depositURI(spec depositChainSpec, address string) string {
	switch spec.Asset {
	case "BTC":
		return "bitcoin:" + address
	case "ETH", "BNB", "MATIC":
		return "ethereum:" + address
	case "LTC":
		return "litecoin:" + address
	case "DOGE":
		return "dogecoin:" + address
	default:
		return address
	}
}

func findDepositSpec(asset string) *depositChainSpec {
	asset = strings.ToUpper(asset)
	for i := range financeDepositChains {
		if financeDepositChains[i].Asset == asset {
			return &financeDepositChains[i]
		}
	}
	return nil
}

// handleDepositAddresses returns every network's deterministic deposit
// address for the authenticated user (QR available per network).
func handleDepositAddresses(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if len(financeCfg.masterSeed) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "deposit addresses are not configured on this node"})
		return
	}
	type row struct {
		Asset          string   `json:"asset"`
		Network        string   `json:"network"`
		Address        string   `json:"address"`
		URI            string   `json:"uri"`
		Assets         []string `json:"assets"`
		DepositEnabled bool     `json:"deposit_enabled"`
	}
	out := make([]row, 0, len(financeDepositChains))
	for _, spec := range financeDepositChains {
		addr, err := deriveDepositAddress(uid, spec)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "address derivation failed"})
			return
		}
		out = append(out, row{
			Asset:          spec.Asset,
			Network:        spec.Network,
			Address:        addr,
			URI:            depositURI(spec, addr),
			Assets:         spec.Assets,
			DepositEnabled: switchEnabled(c.Request.Context(), spec.Asset, "deposit"),
		})
	}
	c.JSON(http.StatusOK, gin.H{"addresses": out})
}

// handleDepositAddress returns one network's address.
func handleDepositAddress(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	spec := findDepositSpec(c.Param("asset"))
	if spec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "unsupported asset"})
		return
	}
	addr, err := deriveDepositAddress(uid, *spec)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"asset": spec.Asset, "network": spec.Network, "address": addr,
		"uri": depositURI(*spec, addr), "assets": spec.Assets,
		"deposit_enabled": switchEnabled(c.Request.Context(), spec.Asset, "deposit"),
	})
}

// handleDepositAddressQR renders the payment URI as a PNG QR code so every
// client (web/desktop/extension/android/ios/flutter) can show a scannable
// code without shipping its own QR encoder.
func handleDepositAddressQR(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	spec := findDepositSpec(c.Param("asset"))
	if spec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "unsupported asset"})
		return
	}
	addr, err := deriveDepositAddress(uid, *spec)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	size := 256
	if s := c.Query("size"); s != "" {
		var n int
		if _, err := fmt.Sscanf(s, "%d", &n); err == nil && n >= 64 && n <= 1024 {
			size = n
		}
	}
	png, err := qrcode.Encode(depositURI(*spec, addr), qrcode.Medium, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "QR generation failed"})
		return
	}
	c.Data(http.StatusOK, "image/png", png)
}
