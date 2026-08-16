package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"golang.org/x/crypto/scrypt"
)

// hmacSHA512 computes HMAC-SHA512(key, data).
func hmacSHA512(key, data []byte) []byte {
	mac := hmac.New(sha512.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// beUint32 encodes a uint32 as 4 big-endian bytes.
func beUint32(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

// paddedBytes left-pads a byte slice to the given length.
func paddedBytes(b []byte, length int) []byte {
	if len(b) >= length {
		return b[:length]
	}
	out := make([]byte, length)
	copy(out[length-len(b):], b)
	return out
}

// secp256k1CurveOrder returns the curve order n.
func secp256k1CurveOrder() *big.Int {
	return secp256k1.S256().Params().N
}

// encryptSeedScryptAESGCM encrypts a seed with a passphrase-derived key using
// scrypt (N=32768, r=8, p=1) + AES-256-GCM. Output: salt(32)||nonce(12)||ciphertext,
// hex-encoded. Fail-closed on any crypto error.
func encryptSeedScryptAESGCM(seed []byte, passphrase string) (string, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := scrypt.Key([]byte(passphrase), salt, 32768, 8, 1, 32)
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
	ct := gcm.Seal(nil, nonce, seed, nil)
	out := append(salt, nonce...)
	out = append(out, ct...)
	return hex.EncodeToString(out), nil
}

// decryptSeedScryptAESGCM decrypts a hex-encoded salt||nonce||ciphertext blob.
// Fail-closed on any error (wrong passphrase, tampered ciphertext).
func decryptSeedScryptAESGCM(blobHex, passphrase string) ([]byte, error) {
	raw, err := hex.DecodeString(blobHex)
	if err != nil {
		return nil, err
	}
	if len(raw) < 44 {
		return nil, errors.New("invalid blob")
	}
	salt := raw[:32]
	nonce := raw[32:44]
	ct := raw[44:]
	key, err := scrypt.Key([]byte(passphrase), salt, 32768, 8, 1, 32)
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
	seed, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, errors.New("invalid passphrase or tampered ciphertext")
	}
	return seed, nil
}
