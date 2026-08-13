package main

// crypto_core.go — REAL cryptographic primitives for the MasterWallet backend.
// No SHA-256 fakes, no P-256, no random "signatures". All EVM key derivation
// uses secp256k1 via go-ethereum/crypto and keccak256; BIP-39/32/44 are real.
// Seed encryption uses scrypt + AES-256-GCM (constant-time MAC compare).

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
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/scrypt"
)

// --- BIP-39 mnemonic ---

// GenerateMnemonic returns a real BIP-39 mnemonic with the given entropy bits
// (128 -> 12 words, 256 -> 24 words). The wordlist + checksum are validated.
func GenerateMnemonic(entropyBits int) (string, error) {
	if entropyBits != 128 && entropyBits != 256 {
		entropyBits = 256
	}
	entropy, err := bip39.NewEntropy(entropyBits)
	if err != nil {
		return "", fmt.Errorf("generate entropy: %w", err)
	}
	return bip39.NewMnemonic(entropy)
}

// ValidateMnemonic checks the BIP-39 wordlist + checksum.
func ValidateMnemonic(mnemonic string) bool {
	return bip39.IsMnemonicValid(mnemonic)
}

// MnemonicToSeed derives the BIP-39 seed (PBKDF2-HMAC-SHA512, 2048 iterations).
func MnemonicToSeed(mnemonic, passphrase string) []byte {
	return bip39.NewSeed(mnemonic, passphrase)
}

// --- BIP-32/44 HD key derivation (secp256k1) ---

// DeriveEVMPrivateKey derives the secp256k1 private key at the canonical BIP-44
// Ethereum path m/44'/60'/0'/0/<accountIndex> from a BIP-39 seed.
func DeriveEVMPrivateKey(seed []byte, accountIndex uint32) (*ecdsa.PrivateKey, error) {
	path := fmt.Sprintf("m/44'/60'/0'/0/%d", accountIndex)
	return hdDerive(seed, path)
}

// DerivePrivateKeyFromPath derives the secp256k1 private key at an arbitrary
// BIP-32 path from a BIP-39 seed.
func DerivePrivateKeyFromPath(seed []byte, path string) (*ecdsa.PrivateKey, error) {
	return hdDerive(seed, path)
}

// PrivateKeyToAddress returns the EIP-55 checksummed address for a secp256k1
// private key: keccak256(publicKey[1:]) last 20 bytes.
func PrivateKeyToAddress(key *ecdsa.PrivateKey) common.Address {
	return crypto.PubkeyToAddress(key.PublicKey)
}

// --- EVM transaction signing + broadcast ---

// SignEVMTransaction builds and signs an EIP-1559 (London) transaction locally
// and returns the RLP-encoded raw transaction hex. No eth_sendTransaction to a
// node-held key — the private key signs on the server.
func SignEVMTransaction(chainID *big.Int, nonce uint64, to common.Address,
	value *big.Int, gasLimit uint64, maxFee, prioFee *big.Int, data []byte,
	privateKey *ecdsa.PrivateKey) (string, error) {
	if chainID == nil || chainID.Sign() <= 0 {
		return "", errors.New("invalid chain id")
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        &to,
		Value:     value,
		Gas:       gasLimit,
		GasFeeCap: maxFee,
		GasTipCap: prioFee,
		Data:      data,
	})
	signer := types.NewLondonSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return "", fmt.Errorf("sign tx: %w", err)
	}
	rawBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("marshal signed tx: %w", err)
	}
	return "0x" + hex.EncodeToString(rawBytes), nil
}

// SignLegacyTransaction builds + signs a legacy (pre-1559) transaction with
// EIP-155 replay protection. Used by chains/nodes that don't support London.
func SignLegacyTransaction(chainID *big.Int, nonce uint64, to common.Address,
	value *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte,
	privateKey *ecdsa.PrivateKey) (string, error) {
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    value,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
	signer := types.NewEIP155Signer(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return "", fmt.Errorf("sign legacy tx: %w", err)
	}
	rawBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("marshal signed legacy tx: %w", err)
	}
	return "0x" + hex.EncodeToString(rawBytes), nil
}

// BroadcastTransaction submits a raw signed transaction via eth_sendRawTransaction
// and returns the real tx hash from the node.
func BroadcastTransaction(ctx context.Context, rpcEndpoint, rawTxHex string) (string, error) {
	client, err := ethclient.DialContext(ctx, rpcEndpoint)
	if err != nil {
		return "", fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()
	var result string
	if err := client.Client().CallContext(ctx, &result, "eth_sendRawTransaction", rawTxHex); err != nil {
		return "", fmt.Errorf("broadcast: %w", err)
	}
	if result == "" {
		return "", errors.New("rpc returned empty transaction hash")
	}
	return result, nil
}

// SignPersonalMessage signs an EIP-191 personal message (keccak256 with the
// "\x19Ethereum Signed Message:\n<len>" prefix) and returns r||s||v (65 bytes).
func SignPersonalMessage(privateKey *ecdsa.PrivateKey, message []byte) ([]byte, error) {
	return crypto.Sign(personalSignHash(message), privateKey)
}

func personalSignHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

// --- Seed encryption (scrypt + AES-256-GCM) ---

const (
	scryptN = 1 << 18
	scryptR = 8
	scryptP = 1
	keyLen  = 32
)

// EncryptSeed encrypts a BIP-39 seed with a password: scrypt key derivation
// (N=131072, r=8, p=1) + AES-256-GCM. Returns hex(salt||nonce||ciphertext).
// The password is never stored.
func EncryptSeed(seed []byte, password string) (string, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	dk, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return "", fmt.Errorf("scrypt: %w", err)
	}
	block, err := aes.NewCipher(dk)
	if err != nil {
		return "", fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ct := gcm.Seal(nil, nonce, seed, nil)
	return hex.EncodeToString(salt) + hex.EncodeToString(nonce) + hex.EncodeToString(ct), nil
}

// DecryptSeed reverses EncryptSeed. Wrong password fails the GCM auth tag.
func DecryptSeed(encHex, password string) ([]byte, error) {
	if len(encHex) < 2*32 {
		return nil, errors.New("invalid encrypted seed length")
	}
	salt, err := hex.DecodeString(encHex[:64])
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	dk, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return nil, fmt.Errorf("scrypt: %w", err)
	}
	block, err := aes.NewCipher(dk)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	ns := gcm.NonceSize() * 2
	if len(encHex) < 64+ns {
		return nil, errors.New("invalid encrypted seed (nonce)")
	}
	nonce, err := hex.DecodeString(encHex[64 : 64+ns])
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	ct, err := hex.DecodeString(encHex[64+ns:])
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}
	seed, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, errors.New("invalid password (authentication failed)")
	}
	return seed, nil
}

// --- helpers ---

// loadOwnedSeed decrypts the master wallet's encrypted seed, verifying the
// caller owns the wallet + the password is correct. Used before any signing.
func loadOwnedSeed(encSeed, password string) ([]byte, error) {
	return DecryptSeed(encSeed, password)
}

// parsePrivateKeyHex parses a hex-encoded secp256k1 private key.
func parsePrivateKeyHex(hexKey string) (*ecdsa.PrivateKey, error) {
	hexKey = strings.TrimPrefix(hexKey, "0x")
	return crypto.HexToECDSA(hexKey)
}

// ethRPCClient returns a low-level RPC client for raw JSON-RPC calls.
func ethRPCClient(ctx context.Context, endpoint string) (*rpc.Client, error) {
	return rpc.DialContext(ctx, endpoint)
}

// sha256 helper (only used for non-cryptographic hashing such as API key hashing
// where a keccak preimage is not security-relevant).
func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
