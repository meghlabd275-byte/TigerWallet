package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// signatureLength is the size of an `r || s || v` ECDSA signature. The recovery
// id is the final byte, so a shorter value cannot be recovered.
const signatureLength = 65

// ErrInvalidSignature covers every way a signature can fail to recover.
var ErrInvalidSignature = errors.New("the signature is invalid")

// normalizeSignatureHex validates a signature and returns its bytes.
//
// The value may be given with or without the `0x` prefix, because devices
// differ in which form they report.
func normalizeSignatureHex(signature string) ([]byte, error) {
	trimmed := strings.TrimSpace(signature)
	if trimmed == "" {
		return nil, errors.New("a signature is required")
	}

	if !strings.HasPrefix(trimmed, "0x") && !strings.HasPrefix(trimmed, "0X") {
		trimmed = "0x" + trimmed
	}

	decoded, err := hexutil.Decode(trimmed)
	if err != nil {
		return nil, fmt.Errorf("the signature is not valid hex: %w", err)
	}

	if len(decoded) != signatureLength {
		return nil, fmt.Errorf("the signature is %d bytes, expected %d", len(decoded), signatureLength)
	}

	return decoded, nil
}

// normalizeSignatureV returns a signature whose recovery id is in the 0-1 range
// the crypto library expects.
//
// Wallets report the recovery id as 0/1 or as 27/28. The EIP-155 form, which
// folds the chain id into the value, describes a signed transaction rather than
// a signature over a hash, so it is rejected here.
func normalizeSignatureV(signature []byte) ([]byte, error) {
	normalized := make([]byte, signatureLength)
	copy(normalized, signature)

	switch v := normalized[64]; {
	case v <= 1:
		// Already in the expected range.
	case v >= 27 && v <= 28:
		normalized[64] = v - 27
	default:
		return nil, fmt.Errorf("the recovery id %d is not a valid signature recovery id", v)
	}

	return normalized, nil
}

// recoverAddress returns the address that produced a signature over a 32-byte
// digest.
func recoverAddress(digest []byte, signature []byte) (common.Address, error) {
	if len(digest) != 32 {
		return common.Address{}, fmt.Errorf("the digest is %d bytes, expected 32", len(digest))
	}

	normalized, err := normalizeSignatureV(signature)
	if err != nil {
		return common.Address{}, err
	}

	r := new(big.Int).SetBytes(normalized[:32])
	s := new(big.Int).SetBytes(normalized[32:64])

	// `SigToPub` accepts some values that are not valid signatures, so the
	// range and malleability checks are applied first. The homestead flag makes
	// a high `s` scalar invalid, which is what stops a third party turning one
	// valid signature into a second valid one.
	if !crypto.ValidateSignatureValues(normalized[64], r, s, true) {
		return common.Address{}, ErrInvalidSignature
	}

	pubkey, err := crypto.SigToPub(digest, normalized)
	if err != nil {
		return common.Address{}, fmt.Errorf("%w: %v", ErrInvalidSignature, err)
	}

	return crypto.PubkeyToAddress(*pubkey), nil
}

// personalMessageDigest builds the EIP-191 digest a device signs for a message.
func personalMessageDigest(message string) []byte {
	prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	return crypto.Keccak256([]byte(prefixed))
}

// RecoverPersonalSignature recovers the address that signed a message with the
// EIP-191 personal-message prefix.
func RecoverPersonalSignature(message string, signature string) (common.Address, error) {
	sig, err := normalizeSignatureHex(signature)
	if err != nil {
		return common.Address{}, err
	}

	return recoverAddress(personalMessageDigest(message), sig)
}

// RecoverTransactionSignature recovers the address that signed a raw
// transaction's signing hash.
func RecoverTransactionSignature(transaction string, signature string) (common.Address, error) {
	raw, err := decodeTransaction(transaction)
	if err != nil {
		return common.Address{}, err
	}

	sig, err := normalizeSignatureHex(signature)
	if err != nil {
		return common.Address{}, err
	}

	return recoverAddress(crypto.Keccak256(raw), sig)
}

// decodeTransaction accepts a transaction as hex bytes or as a JSON object
// carrying a `raw` hex field, because clients send both shapes.
func decodeTransaction(transaction string) ([]byte, error) {
	trimmed := strings.TrimSpace(transaction)
	if trimmed == "" {
		return nil, errors.New("the transaction is required")
	}

	if strings.HasPrefix(trimmed, "{") {
		var payload struct {
			Raw string `json:"raw"`
		}

		if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
			return nil, fmt.Errorf("the transaction object is malformed: %w", err)
		}

		trimmed = payload.Raw
	}

	if !strings.HasPrefix(trimmed, "0x") && !strings.HasPrefix(trimmed, "0X") {
		trimmed = "0x" + trimmed
	}

	raw, err := hexutil.Decode(trimmed)
	if err != nil {
		return nil, fmt.Errorf("the transaction is not valid hex: %w", err)
	}

	if len(raw) == 0 {
		return nil, errors.New("the transaction is empty")
	}

	return raw, nil
}

// RecoverTypedDataSignature recovers the address that signed an EIP-712
// payload.
//
// The digest follows the canonical rules, including the domain separator and
// the recursive struct encoding, so a signature over a payload the device
// displayed verifies against exactly that payload.
func RecoverTypedDataSignature(typedDataJSON string, signature string) (common.Address, error) {
	sig, err := normalizeSignatureHex(signature)
	if err != nil {
		return common.Address{}, err
	}

	typedData, err := ParseTypedData(typedDataJSON)
	if err != nil {
		return common.Address{}, err
	}

	digest, _, err := apitypes.TypedDataAndHash(*typedData)
	if err != nil {
		return common.Address{}, fmt.Errorf("the typed data could not be hashed: %w", err)
	}

	return recoverAddress(digest, sig)
}

// ParseTypedData decodes an EIP-712 payload and validates the parts the hashing
// rules require.
func ParseTypedData(payload string) (*apitypes.TypedData, error) {
	if strings.TrimSpace(payload) == "" {
		return nil, errors.New("typed data is required")
	}

	typedData := new(apitypes.TypedData)
	if err := json.Unmarshal([]byte(payload), typedData); err != nil {
		return nil, fmt.Errorf("the typed data is not valid JSON: %w", err)
	}

	if typedData.PrimaryType == "" {
		return nil, errors.New("the typed data has no primary type")
	}

	if len(typedData.Types) == 0 {
		return nil, errors.New("the typed data has no types")
	}

	if _, ok := typedData.Types[typedData.PrimaryType]; !ok {
		return nil, fmt.Errorf("the primary type %q is not defined", typedData.PrimaryType)
	}

	// A domain without these fields cannot be bound to one contract on one
	// chain, which is what stops a signature being replayed elsewhere.
	if typedData.Domain.Name == "" {
		return nil, errors.New("the typed data domain must set a name")
	}
	if typedData.Domain.Version == "" {
		return nil, errors.New("the typed data domain must set a version")
	}
	if typedData.Domain.ChainId == nil {
		return nil, errors.New("the typed data domain must set a chain id")
	}

	return typedData, nil
}

// TypedDataDigest returns the EIP-712 digest and the domain separator for a
// payload, so a caller can show what the device is being asked to sign.
func TypedDataDigest(payload string) (digestHex string, domainSeparatorHex string, err error) {
	typedData, err := ParseTypedData(payload)
	if err != nil {
		return "", "", err
	}

	digest, _, err := apitypes.TypedDataAndHash(*typedData)
	if err != nil {
		return "", "", fmt.Errorf("the typed data could not be hashed: %w", err)
	}

	separator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return "", "", fmt.Errorf("the domain separator could not be hashed: %w", err)
	}

	return hexutil.Encode(digest), hexutil.Encode(separator), nil
}

// DeriveAddressFromPublicKey returns the address for a secp256k1 public key.
//
// The key may be compressed (33 bytes) or uncompressed (65 bytes), and the
// uncompressed form may omit its `0x04` prefix.
func DeriveAddressFromPublicKey(publicKey string) (string, error) {
	trimmed := strings.TrimSpace(publicKey)
	if trimmed == "" {
		return "", errors.New("a public key is required")
	}

	if !strings.HasPrefix(trimmed, "0x") && !strings.HasPrefix(trimmed, "0X") {
		trimmed = "0x" + trimmed
	}

	decoded, err := hexutil.Decode(trimmed)
	if err != nil {
		return "", fmt.Errorf("the public key is not valid hex: %w", err)
	}

	switch len(decoded) {
	case 33:
		// A compressed point needs expanding before it can be used.
		key, err := crypto.DecompressPubkey(decoded)
		if err != nil {
			return "", fmt.Errorf("the public key is not a valid secp256k1 point: %w", err)
		}

		return crypto.PubkeyToAddress(*key).Hex(), nil
	case 64:
		decoded = append([]byte{0x04}, decoded...)
	case 65:
		if decoded[0] != 0x04 {
			return "", errors.New("the uncompressed public key does not start with 0x04")
		}
	default:
		return "", fmt.Errorf("the public key is %d bytes, expected 33, 64 or 65", len(decoded))
	}

	key, err := crypto.UnmarshalPubkey(decoded)
	if err != nil {
		return "", fmt.Errorf("the public key is not a valid secp256k1 point: %w", err)
	}

	return crypto.PubkeyToAddress(*key).Hex(), nil
}

// newChallenge returns a random 32-byte challenge.
//
// The challenge is what makes pairing a proof of possession: only the device
// holding the private key can produce a signature that recovers to its address.
func newChallenge() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("a challenge could not be generated: %w", err)
	}

	return "0x" + hex.EncodeToString(buffer), nil
}

// newDeviceID returns a random device identifier.
func newDeviceID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("a device id could not be generated: %w", err)
	}

	return "dev_" + hex.EncodeToString(buffer), nil
}

// verifyChallenge checks that a signature over a challenge recovers to the
// expected address.
func verifyChallenge(challenge, signature, expectedAddress string) (common.Address, error) {
	sig, err := normalizeSignatureHex(signature)
	if err != nil {
		return common.Address{}, err
	}

	recovered, err := recoverAddress(personalMessageDigest(challenge), sig)
	if err != nil {
		return common.Address{}, err
	}

	expected, err := checksumAddress(expectedAddress)
	if err != nil {
		return common.Address{}, err
	}

	if recovered != common.HexToAddress(expected) {
		return common.Address{}, fmt.Errorf(
			"%w: the signature recovers to %s, not %s",
			ErrInvalidSignature,
			recovered.Hex(),
			expected,
		)
	}

	return recovered, nil
}

// SignDigest signs a 32-byte digest with a private key.
//
// This exists for keys the service itself holds. A hardware wallet never
// exports its key, so device signatures arrive as input and are verified rather
// than produced here.
func SignDigest(digest []byte, privateKey *ecdsa.PrivateKey) (string, error) {
	if len(digest) != 32 {
		return "", fmt.Errorf("the digest is %d bytes, expected 32", len(digest))
	}

	signature, err := crypto.Sign(digest, privateKey)
	if err != nil {
		return "", fmt.Errorf("the digest could not be signed: %w", err)
	}

	return hexutil.Encode(signature), nil
}

// checksumAddress validates an address and returns it in EIP-55 form.
func checksumAddress(address string) (string, error) {
	if !common.IsHexAddress(address) {
		return "", fmt.Errorf("%q is not a valid address", address)
	}

	return common.HexToAddress(address).Hex(), nil
}
