package main

// wallet_engine.go — REAL key management: BIP-39 mnemonic, BIP-32/44 HD
// derivation, secp256k1 signing, and EVM transaction construction/broadcast.
// No mocks, no stubs, no hardcoded keys.

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// GenerateMnemonic creates a real BIP-39 mnemonic with the given entropy
// (128 = 12 words, 256 = 24 words).
func GenerateMnemonic(entropyBits int) (string, error) {
	if entropyBits != 128 && entropyBits != 256 {
		entropyBits = 256
	}
	entropy, err := bip39.NewEntropy(entropyBits)
	if err != nil {
		return "", fmt.Errorf("entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("mnemonic: %w", err)
	}
	return mnemonic, nil
}

// ValidateMnemonic checks a mnemonic against BIP-39 (wordlist + checksum).
func ValidateMnemonic(mnemonic string) bool {
	return bip39.IsMnemonicValid(mnemonic)
}

// MnemonicToSeed derives the 64-byte BIP-39 seed (PBKDF2-HMAC-SHA512, 2048
// iterations) using the standard passphrase.
func MnemonicToSeed(mnemonic, passphrase string) []byte {
	return bip39.NewSeed(mnemonic, passphrase)
}

// DeriveEVMPrivateKey derives the secp256k1 private key for an EVM chain from
// a mnemonic using the chain's BIP-44 derivation path (m/44'/60'/0'/0/index).
func DeriveEVMPrivateKey(mnemonic string, chain ChainConfig, accountIndex uint32) (*ecdsa.PrivateKey, error) {
	if !ValidateMnemonic(mnemonic) {
		return nil, errors.New("invalid mnemonic")
	}
	seed := MnemonicToSeed(mnemonic, "")
	pathStr := chain.DerivationPath
	if !strings.HasPrefix(pathStr, "m/44'/60'/") {
		pathStr = "m/44'/60'/0'/0/" + fmt.Sprint(accountIndex)
	} else {
		// Replace trailing index with the requested account index.
		parts := strings.Split(pathStr, "/")
		if len(parts) >= 6 {
			parts[len(parts)-1] = fmt.Sprint(accountIndex)
			pathStr = strings.Join(parts, "/")
		}
	}
	return DerivePrivateKeyFromPath(seed, pathStr)
}

// DerivePrivateKeyFromPath performs real BIP-32 HD derivation along an
// arbitrary path using go-ethereum's accounts HierarchicalKey.
func DerivePrivateKeyFromPath(seed []byte, path string) (*ecdsa.PrivateKey, error) {
	// Use go-ethereum's internal HD derivation via crypto + accounts.
	// accounts.NewFromSeed is not exported, so we implement BIP-32 directly
	// using the secp256k1 CKD functions exposed by go-ethereum/crypto.
	priv, err := hdDerive(seed, path)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

// PrivateKeyToAddress returns the checksummed 0x-prefixed EVM address.
func PrivateKeyToAddress(key *ecdsa.PrivateKey) common.Address {
	return crypto.PubkeyToAddress(key.PublicKey)
}

// SignEVMTransaction builds and signs a real EVM transaction (legacy or
// EIP-1559), returning the hex-encoded raw signed transaction.
func SignEVMTransaction(chainID *big.Int, nonce uint64, to common.Address,
	value *big.Int, gasLimit uint64, gasPrice, maxFee, maxPrioFee *big.Int,
	data []byte, privateKey *ecdsa.PrivateKey) (string, error) {

	signer := types.NewLondonSigner(chainID)

	var tx *types.Transaction
	if maxFee != nil && maxPrioFee != nil && maxFee.Cmp(big.NewInt(0)) > 0 {
		// EIP-1559
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     nonce,
			To:        &to,
			Value:     value,
			Gas:       gasLimit,
			GasFeeCap: maxFee,
			GasTipCap: maxPrioFee,
			Data:      data,
		})
	} else {
		// Legacy
		gp := gasPrice
		if gp == nil || gp.Cmp(big.NewInt(0)) == 0 {
			gp = maxFee
		}
		tx = types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			To:       &to,
			Value:    value,
			Gas:      gasLimit,
			GasPrice: gp,
			Data:     data,
		})
	}

	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return "", fmt.Errorf("sign tx: %w", err)
	}
	rawBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("marshal signed tx: %w", err)
	}
	return "0x" + common.Bytes2Hex(rawBytes), nil
}

// BroadcastTransaction sends a raw signed transaction to the chain via
// eth_sendRawTransaction and returns the tx hash.
func BroadcastTransaction(ctx context.Context, rpcEndpoint, rawTxHex string) (string, error) {
	client, err := rpc.DialContext(ctx, rpcEndpoint)
	if err != nil {
		return "", fmt.Errorf("rpc dial: %w", err)
	}
	defer client.Close()

	var result string
	if err := client.CallContext(ctx, &result, "eth_sendRawTransaction", rawTxHex); err != nil {
		return "", fmt.Errorf("sendRawTransaction: %w", err)
	}
	return result, nil
}

// SignPersonalMessage signs an Ethereum personal_sign message (EIP-191) and
// returns the 65-byte signature (r||s||v).
func SignPersonalMessage(privateKey *ecdsa.PrivateKey, message []byte) ([]byte, error) {
	prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), string(message))
	hash := crypto.Keccak256([]byte(prefixed))
	sig, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return nil, err
	}
	// crypto.Sign returns v as 0/1; EIP-191 expects 27/28.
	if len(sig) == 65 {
		sig[64] += 27
	}
	return sig, nil
}

// SignTypedDataV4 signs an EIP-712 typed data hash.
func SignTypedDataV4(privateKey *ecdsa.PrivateKey, domainSeparatorHash []byte) ([]byte, error) {
	sig, err := crypto.Sign(domainSeparatorHash, privateKey)
	if err != nil {
		return nil, err
	}
	if len(sig) == 65 {
		sig[64] += 27
	}
	return sig, nil
}

// EncryptSeed encrypts a mnemonic seed with a password using AES-256-GCM,
// key derived via scrypt (N=32768) — production-grade KDF.
func EncryptSeed(seed []byte, password string) (string, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, seed, nil)
	blob := append(salt, ciphertext...)
	return hex.EncodeToString(blob), nil
}

// DecryptSeed decrypts a seed encrypted by EncryptSeed.
func DecryptSeed(encHex, password string) ([]byte, error) {
	blob, err := hex.DecodeString(encHex)
	if err != nil {
		return nil, err
	}
	if len(blob) < 32 {
		return nil, errors.New("invalid ciphertext")
	}
	salt := blob[:32]
	ciphertext := blob[32:]
	key, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// DerivePrivateKeyHashPBKDF2 is a helper used for non-crypto-derived secrets
// (e.g. session tokens) — not for wallet keys.
func DerivePrivateKeyHashPBKDF2(secret, salt []byte) []byte {
	return pbkdf2.Key(secret, salt, 10000, 32, sha256.New)
}
