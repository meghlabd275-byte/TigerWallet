// Package crypto provides REAL BIP-39 mnemonic generation, BIP-32 HD key
// derivation, BIP-44 Ethereum key derivation, and real EVM transaction signing
// (secp256k1 + keccak256 + EIP-1559). This is the standalone WL-UserWallet's
// own key management — it does NOT delegate to TigerWallet cloud.
package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/tyler-smith/go-bip39"
)

// GenerateMnemonic generates a real 24-word BIP-39 mnemonic (256-bit entropy).
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	return bip39.NewMnemonic(entropy)
}

// MnemonicToSeed converts a mnemonic to a BIP-39 seed (PBKDF2-HMAC-SHA512,
// 2048 iterations, "mnemonic" + passphrase as salt).
func MnemonicToSeed(mnemonic, passphrase string) []byte {
	return bip39.NewSeed(mnemonic, passphrase)
}

// DeriveEVMPrivateKey derives the EVM private key at m/44'/60'/0'/0/0 from a
// seed using real BIP-32 HMAC-SHA512 CKD (the canonical path; matches the
// go-ethereum / MetaMask derivation).
func DeriveEVMPrivateKey(seed []byte, accountIndex uint32) (*ecdsa.PrivateKey, error) {
	master, err := deriveMaster(seed)
	if err != nil {
		return nil, err
	}
	// m/44'/60'/0'/0/accountIndex
	path := []uint32{
		44 + 0x80000000,
		60 + 0x80000000,
		0 + 0x80000000,
		0,
		accountIndex,
	}
	current := master
	for _, child := range path {
		next, err := deriveChild(current, child)
		if err != nil {
			return nil, err
		}
		current = next
	}
	// current is 64 bytes: [0:32] = private key, [32:64] = chain code.
	return crypto.ToECDSA(current[:32])
}

// deriveMaster computes the BIP-32 master key from a seed (HMAC-SHA512 with
// "Bitcoin seed" as the key). Returns the 64-byte master key (32 priv + 32 chain).
func deriveMaster(seed []byte) ([]byte, error) {
	if len(seed) < 16 || len(seed) > 64 {
		return nil, errors.New("invalid seed length")
	}
	mac := hmacSHA512([]byte("Bitcoin seed"), seed)
	return mac, nil
}

// deriveChild performs a single BIP-32 CKDpriv step (hardened or normal).
// parentKey is 64 bytes: [0:32] = private key, [32:64] = chain code.
func deriveChild(parentKey []byte, index uint32) ([]byte, error) {
	if len(parentKey) != 64 {
		return nil, errors.New("invalid parent key length")
	}
	parentPriv := parentKey[:32]
	parentChain := parentKey[32:64]

	// Reconstruct the ecdsa private key to get the public key.
	priv, err := crypto.ToECDSA(parentPriv)
	if err != nil {
		return nil, err
	}
	var data []byte
	if index >= 0x80000000 {
		// hardened: 0x00 || parentPriv || index(BE)
		data = append(data, 0x00)
		data = append(data, parentPriv...)
	} else {
		// normal: serP(parentPubKey) || index(BE)  (compressed 33-byte form)
		pub := crypto.CompressPubkey(&priv.PublicKey)
		data = append(data, pub...)
	}
	idxBytes := beUint32(index)
	data = append(data, idxBytes...)

	mac := hmacSHA512(parentChain, data)
	il := mac[:32]
	ir := mac[32:64]
	// child priv = (il + parentPriv) mod n
	ilInt := new(big.Int).SetBytes(il)
	parentInt := new(big.Int).SetBytes(parentPriv)
	n := secp256k1CurveOrder()
	childInt := new(big.Int).Add(ilInt, parentInt)
	childInt.Mod(childInt, n)
	if childInt.Sign() == 0 {
		return nil, errors.New("invalid child key (zero)")
	}
	childPriv := paddedBytes(childInt.Bytes(), 32)
	child := append(childPriv, ir...)
	return child, nil
}

// AddressFromPrivateKey returns the EIP-55 checksummed address.
func AddressFromPrivateKey(key *ecdsa.PrivateKey) string {
	return crypto.PubkeyToAddress(key.PublicKey).Hex()
}

// SignTransaction builds + signs a real EIP-1559 DynamicFee transaction.
func SignTransaction(key *ecdsa.PrivateKey, chainID *big.Int, to common.Address, amount *big.Int, gasLimit uint64, maxFee, maxPriorityFee *big.Int, nonce uint64, data []byte) (string, error) {
	if gasLimit == 0 {
		gasLimit = 21000
	}
	if maxFee == nil {
		maxFee = big.NewInt(params.GWei * 20)
	}
	if maxPriorityFee == nil {
		maxPriorityFee = big.NewInt(params.GWei * 2)
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        &to,
		Value:     amount,
		Gas:       gasLimit,
		GasFeeCap: maxFee,
		GasTipCap: maxPriorityFee,
		Data:      data,
	})
	signer := types.NewLondonSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, key)
	if err != nil {
		return "", err
	}
	raw, err := signedTx.MarshalBinary()
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(raw), nil
}

// SignMessage signs a personal_sign message (EIP-191 prefix + keccak256).
func SignMessage(key *ecdsa.PrivateKey, message string) (string, error) {
	prefixed := crypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)))
	sig, err := crypto.Sign(prefixed, key)
	if err != nil {
		return "", err
	}
	if len(sig) != 65 {
		return "", errors.New("invalid signature length")
	}
	// recovery byte: 0/1 -> 27/28 for personal_sign
	sig[64] += 27
	return "0x" + hex.EncodeToString(sig), nil
}

// EncryptSeedAtRest encrypts a seed with a passphrase-derived key. Used to
// store the seed in PG without exposing it. (The master_wallet backend has a
// full scrypt+AES-GCM implementation; this is a thin wrapper for the standalone
// WL backend.)
func EncryptSeedAtRest(seed []byte, passphrase string) (string, error) {
	// Delegate to the canonical scrypt+AES-GCM path via go-ethereum's crypto
	// utilities is not available; use a real scrypt + AES-GCM implementation.
	return encryptSeedScryptAESGCM(seed, passphrase)
}

func DecryptSeedAtRest(blobHex, passphrase string) ([]byte, error) {
	return decryptSeedScryptAESGCM(blobHex, passphrase)
}

// RandomBytes returns cryptographically secure random bytes.
func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
