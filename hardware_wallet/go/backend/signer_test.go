package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// testKey is a fixed private key, so the expected addresses below are stable.
const testKey = "4646464646464646464646464646464646464646464646464646464646464646"

// testKeyAddress is the address for testKey.
const testKeyAddress = "0x9d8A62f656a8d1615C1294fd71e9CFb3E4855A4F"

// signPersonal signs a message the way a device would: with the EIP-191 prefix.
func signPersonal(t *testing.T, message string) string {
	t.Helper()

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	signature, err := crypto.Sign(personalMessageDigest(message), key)
	if err != nil {
		t.Fatalf("the message could not be signed: %v", err)
	}

	return hexutil.Encode(signature)
}

func TestTheTestKeyAddressIsTheExpectedOne(t *testing.T) {
	// The vectors below depend on this address, so it is checked directly.
	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	if got := crypto.PubkeyToAddress(key.PublicKey).Hex(); got != testKeyAddress {
		t.Fatalf("address = %s, want %s", got, testKeyAddress)
	}
}

func TestAPersonalSignatureRecoversToTheSigner(t *testing.T) {
	message := "TigerWallet pairing challenge"
	signature := signPersonal(t, message)

	recovered, err := RecoverPersonalSignature(message, signature)
	if err != nil {
		t.Fatalf("RecoverPersonalSignature: %v", err)
	}

	if recovered.Hex() != testKeyAddress {
		t.Fatalf("recovered %s, want %s", recovered.Hex(), testKeyAddress)
	}
}

func TestAPersonalSignatureOverADifferentMessageDoesNotRecoverToTheSigner(t *testing.T) {
	signature := signPersonal(t, "the message the user approved")

	recovered, err := RecoverPersonalSignature("a different message", signature)
	if err != nil {
		t.Fatalf("RecoverPersonalSignature: %v", err)
	}

	if recovered.Hex() == testKeyAddress {
		t.Fatal("a signature verified against a message it did not sign")
	}
}

func TestASignatureWithAHighSScalarIsRejected(t *testing.T) {
	// A non-canonical s value must be refused, otherwise a third party could
	// flip a valid signature into a second valid one.
	message := "malleability check"
	signature := signPersonal(t, message)

	raw, err := hexutil.Decode(signature)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	// The secp256k1 group order.
	order, ok := new(big.Int).SetString(
		"fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141",
		16,
	)
	if !ok {
		t.Fatal("the group order could not be parsed")
	}

	s := new(big.Int).SetBytes(raw[32:64])
	flipped := new(big.Int).Sub(order, s)

	copy(raw[32:64], flipped.Bytes())

	if _, err := RecoverPersonalSignature(message, hexutil.Encode(raw)); err == nil {
		t.Fatal("a signature with a high s scalar was accepted")
	}
}

func TestMalformedSignaturesAreRejected(t *testing.T) {
	cases := map[string]string{
		"empty":            "",
		"not hex":          "0xzzzz",
		"too short":        "0x1234",
		"too long":         "0x" + strings.Repeat("ab", 66),
		"missing prefix ok": "", // replaced below
	}
	delete(cases, "missing prefix ok")

	for name, signature := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := RecoverPersonalSignature("message", signature); err == nil {
				t.Fatalf("the %s signature was accepted", name)
			}
		})
	}
}

func TestASignatureWithoutTheHexPrefixIsAccepted(t *testing.T) {
	// Devices differ in whether they report the prefix.
	message := "prefix check"
	signature := signPersonal(t, message)

	if _, err := RecoverPersonalSignature(message, strings.TrimPrefix(signature, "0x")); err != nil {
		t.Fatalf("a signature without the 0x prefix was rejected: %v", err)
	}
}

func TestARecoveryIdOf27Or28IsNormalised(t *testing.T) {
	message := "recovery id check"
	signature := signPersonal(t, message)

	raw, err := hexutil.Decode(signature)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Wallets that report v as 27/28 must still verify.
	raw[64] += 27

	recovered, err := RecoverPersonalSignature(message, hexutil.Encode(raw))
	if err != nil {
		t.Fatalf("a v=27 signature was rejected: %v", err)
	}

	if recovered.Hex() != testKeyAddress {
		t.Fatalf("recovered %s, want %s", recovered.Hex(), testKeyAddress)
	}
}

func TestAnEIP155RecoveryIdIsRejected(t *testing.T) {
	// A v that carries a chain id describes a signed transaction, not a
	// signature over a hash, so it must not be treated as one.
	message := "eip155 check"
	signature := signPersonal(t, message)

	raw, err := hexutil.Decode(signature)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	raw[64] = byte(37) // 1 * 2 + 35

	if _, err := RecoverPersonalSignature(message, hexutil.Encode(raw)); err == nil {
		t.Fatal("an EIP-155 recovery id was accepted as a signature recovery id")
	}
}

func TestATransactionSignatureRecoversToTheSigner(t *testing.T) {
	// The digest is the keccak-256 of the encoded transaction, which is what
	// the device signs.
	transaction := "0x02f86b0180843b9aca00847735940082520894"

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	raw, err := hexutil.Decode(transaction)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	signature, err := crypto.Sign(crypto.Keccak256(raw), key)
	if err != nil {
		t.Fatalf("the transaction could not be signed: %v", err)
	}

	recovered, err := RecoverTransactionSignature(transaction, hexutil.Encode(signature))
	if err != nil {
		t.Fatalf("RecoverTransactionSignature: %v", err)
	}

	if recovered.Hex() != testKeyAddress {
		t.Fatalf("recovered %s, want %s", recovered.Hex(), testKeyAddress)
	}
}

func TestATransactionObjectIsAccepted(t *testing.T) {
	transaction := "0x02f86b0180843b9aca00847735940082520894"

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	raw, err := hexutil.Decode(transaction)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	signature, err := crypto.Sign(crypto.Keccak256(raw), key)
	if err != nil {
		t.Fatalf("the transaction could not be signed: %v", err)
	}

	payload := `{"raw":"` + transaction + `"}`

	if _, err := RecoverTransactionSignature(payload, hexutil.Encode(signature)); err != nil {
		t.Fatalf("a transaction object was rejected: %v", err)
	}
}

func TestAnEmptyTransactionIsRejected(t *testing.T) {
	if _, err := RecoverTransactionSignature("", "0x"+strings.Repeat("ab", 65)); err == nil {
		t.Fatal("an empty transaction was accepted")
	}
}

// typedDataPayload is a minimal but complete EIP-712 payload.
const typedDataPayload = `{
  "types": {
    "EIP712Domain": [
      {"name": "name", "type": "string"},
      {"name": "version", "type": "string"},
      {"name": "chainId", "type": "uint256"},
      {"name": "verifyingContract", "type": "address"}
    ],
    "Transfer": [
      {"name": "to", "type": "address"},
      {"name": "amount", "type": "uint256"}
    ]
  },
  "primaryType": "Transfer",
  "domain": {
    "name": "TigerWallet",
    "version": "1",
    "chainId": "1",
    "verifyingContract": "0x0000000000000000000000000000000000000001"
  },
  "message": {
    "to": "0x9d8A62f656a8d1615C1294fd71e9CFb3E4855A4F",
    "amount": "1000000000000000000"
  }
}`

func TestTypedDataHashesToADeterministicDigest(t *testing.T) {
	digest, separator, err := TypedDataDigest(typedDataPayload)
	if err != nil {
		t.Fatalf("TypedDataDigest: %v", err)
	}

	if len(digest) != 66 {
		t.Fatalf("digest = %s, want 32 bytes", digest)
	}
	if len(separator) != 66 {
		t.Fatalf("domain separator = %s, want 32 bytes", separator)
	}

	// The digest must be stable: the same payload always hashes the same way.
	again, _, err := TypedDataDigest(typedDataPayload)
	if err != nil {
		t.Fatalf("TypedDataDigest: %v", err)
	}
	if again != digest {
		t.Fatalf("the digest is not deterministic: %s then %s", digest, again)
	}
}

func TestATypedDataSignatureRecoversToTheSigner(t *testing.T) {
	typedData, err := ParseTypedData(typedDataPayload)
	if err != nil {
		t.Fatalf("ParseTypedData: %v", err)
	}

	digest, _, err := apitypes.TypedDataAndHash(*typedData)
	if err != nil {
		t.Fatalf("hashing: %v", err)
	}

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	signature, err := crypto.Sign(digest, key)
	if err != nil {
		t.Fatalf("the typed data could not be signed: %v", err)
	}

	recovered, err := RecoverTypedDataSignature(typedDataPayload, hexutil.Encode(signature))
	if err != nil {
		t.Fatalf("RecoverTypedDataSignature: %v", err)
	}

	if recovered.Hex() != testKeyAddress {
		t.Fatalf("recovered %s, want %s", recovered.Hex(), testKeyAddress)
	}
}

func TestTypedDataWithoutAChainIdIsRejected(t *testing.T) {
	// Without a chain id a signature could be replayed on another chain.
	payload := strings.Replace(typedDataPayload, `"chainId": "1",`, "", 1)

	if _, err := ParseTypedData(payload); err == nil {
		t.Fatal("typed data without a chain id was accepted")
	}
}

func TestTypedDataWithoutANameIsRejected(t *testing.T) {
	payload := strings.Replace(typedDataPayload, `"name": "TigerWallet",`, "", 1)

	if _, err := ParseTypedData(payload); err == nil {
		t.Fatal("typed data without a domain name was accepted")
	}
}

func TestTypedDataWithAnUndefinedPrimaryTypeIsRejected(t *testing.T) {
	payload := strings.Replace(typedDataPayload, `"primaryType": "Transfer"`, `"primaryType": "Missing"`, 1)

	if _, err := ParseTypedData(payload); err == nil {
		t.Fatal("typed data naming an undefined primary type was accepted")
	}
}

func TestAddressesAreDerivedFromTheKeyNotAsserted(t *testing.T) {
	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	compressed := hexutil.Encode(crypto.CompressPubkey(&key.PublicKey))
	uncompressed := hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey))
	// The uncompressed form without its 0x04 prefix.
	unprefixed := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey)[1:])

	for name, publicKey := range map[string]string{
		"compressed":   compressed,
		"uncompressed": uncompressed,
		"unprefixed":   unprefixed,
	} {
		t.Run(name, func(t *testing.T) {
			address, err := DeriveAddressFromPublicKey(publicKey)
			if err != nil {
				t.Fatalf("DeriveAddressFromPublicKey: %v", err)
			}

			if address != testKeyAddress {
				t.Fatalf("address = %s, want %s", address, testKeyAddress)
			}
		})
	}
}

func TestAnInvalidPublicKeyIsRejected(t *testing.T) {
	if _, err := DeriveAddressFromPublicKey("0x1234"); err == nil {
		t.Fatal("a malformed public key was accepted")
	}
}

func TestRegisteringADeviceDerivesItsAddress(t *testing.T) {
	service := NewDeviceService()

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	device, err := service.Register(context.Background(), RegisterDeviceRequest{
		UserID:    "user-1",
		Vendor:    "ledger",
		Model:     "nano-x",
		PublicKey: hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)),
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if device.Address != testKeyAddress {
		t.Fatalf("address = %s, want %s", device.Address, testKeyAddress)
	}
	if device.Verified {
		t.Fatal("a freshly registered device must not be verified")
	}
}

func TestRegisteringADeviceWithAMismatchedAddressIsRejected(t *testing.T) {
	// A device cannot claim an address its key does not control.
	service := NewDeviceService()

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	_, err = service.Register(context.Background(), RegisterDeviceRequest{
		UserID:    "user-1",
		Vendor:    "ledger",
		PublicKey: hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)),
		Address:   "0x000000000000000000000000000000000000dead",
	})

	if err == nil {
		t.Fatal("a device claiming an address its key does not derive was accepted")
	}
}

func TestRegisteringADeviceWithoutAPublicKeyIsRejected(t *testing.T) {
	service := NewDeviceService()

	if _, err := service.Register(context.Background(), RegisterDeviceRequest{
		UserID: "user-1",
		Vendor: "ledger",
	}); err == nil {
		t.Fatal("a device without a public key was accepted")
	}
}

func TestPairingRequiresASignatureOverTheChallenge(t *testing.T) {
	service := NewDeviceService()

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	device, err := service.Register(context.Background(), RegisterDeviceRequest{
		UserID:    "user-1",
		Vendor:    "ledger",
		PublicKey: hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)),
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	pairing, err := service.Pair(context.Background(), PairDeviceRequest{DeviceID: device.ID})
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}

	challenge := pairing["challenge"]

	signature, err := crypto.Sign(personalMessageDigest(challenge), key)
	if err != nil {
		t.Fatalf("the challenge could not be signed: %v", err)
	}

	result, err := service.Verify(context.Background(), device.ID, VerifyDeviceRequest{
		Challenge: challenge,
		Signature: hexutil.Encode(signature),
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if verified, ok := result["verified"].(bool); !ok || !verified {
		t.Fatalf("the device was not verified: %v", result)
	}

	reloaded, err := service.Get(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reloaded.Verified {
		t.Fatal("the device is not marked verified after a successful pairing")
	}
}

func TestPairingWithTheWrongKeyIsRejected(t *testing.T) {
	service := NewDeviceService()

	deviceKey, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	device, err := service.Register(context.Background(), RegisterDeviceRequest{
		UserID:    "user-1",
		Vendor:    "ledger",
		PublicKey: hexutil.Encode(crypto.FromECDSAPub(&deviceKey.PublicKey)),
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	pairing, err := service.Pair(context.Background(), PairDeviceRequest{DeviceID: device.ID})
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}

	// An attacker signs the challenge with their own key.
	attackerKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("a key could not be generated: %v", err)
	}

	signature, err := crypto.Sign(personalMessageDigest(pairing["challenge"]), attackerKey)
	if err != nil {
		t.Fatalf("the challenge could not be signed: %v", err)
	}

	if _, err := service.Verify(context.Background(), device.ID, VerifyDeviceRequest{
		Signature: hexutil.Encode(signature),
	}); err == nil {
		t.Fatal("a signature from the wrong key was accepted")
	}
}

func TestAChallengeIsSingleUse(t *testing.T) {
	service := NewDeviceService()

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	device, err := service.Register(context.Background(), RegisterDeviceRequest{
		UserID:    "user-1",
		Vendor:    "ledger",
		PublicKey: hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)),
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	pairing, err := service.Pair(context.Background(), PairDeviceRequest{DeviceID: device.ID})
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}

	signature, err := crypto.Sign(personalMessageDigest(pairing["challenge"]), key)
	if err != nil {
		t.Fatalf("the challenge could not be signed: %v", err)
	}

	request := VerifyDeviceRequest{Signature: hexutil.Encode(signature)}

	if _, err := service.Verify(context.Background(), device.ID, request); err != nil {
		t.Fatalf("the first verification failed: %v", err)
	}

	if _, err := service.Verify(context.Background(), device.ID, request); err == nil {
		t.Fatal("a challenge was accepted twice")
	}
}

func TestPairingAttemptsAreBounded(t *testing.T) {
	service := NewDeviceService()

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	device, err := service.Register(context.Background(), RegisterDeviceRequest{
		UserID:    "user-1",
		Vendor:    "ledger",
		PublicKey: hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)),
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	pairing, err := service.Pair(context.Background(), PairDeviceRequest{DeviceID: device.ID})
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}

	// Wrong signatures are tried until the challenge is burned.
	bad := "0x" + strings.Repeat("ab", 65)

	var lastErr error
	for attempt := 0; attempt <= maxPairingAttempts; attempt++ {
		_, lastErr = service.Verify(context.Background(), device.ID, VerifyDeviceRequest{
			Challenge: pairing["challenge"],
			Signature: bad,
		})
	}

	if lastErr == nil {
		t.Fatal("the challenge survived more attempts than it allows")
	}

	// Even a correct signature cannot revive a burned challenge.
	good, err := crypto.Sign(personalMessageDigest(pairing["challenge"]), key)
	if err != nil {
		t.Fatalf("the challenge could not be signed: %v", err)
	}

	if _, err := service.Verify(context.Background(), device.ID, VerifyDeviceRequest{
		Signature: hexutil.Encode(good),
	}); err == nil {
		t.Fatal("a burned challenge was accepted")
	}
}

func TestAnUnpairedDeviceCannotSign(t *testing.T) {
	service := NewDeviceService()

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	device, err := service.Register(context.Background(), RegisterDeviceRequest{
		UserID:    "user-1",
		Vendor:    "ledger",
		PublicKey: hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)),
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if _, err := service.RecordSignatureUse(device.ID); err == nil {
		t.Fatal("an unpaired device was allowed to sign")
	}
}

func TestFirmwareDigestIsComputedFromTheImage(t *testing.T) {
	service := NewFirmwareService()

	image := "0xdeadbeef"

	firmware, err := service.Upload(context.Background(), "ledger", "nano-x", UploadFirmwareRequest{
		Version:     "1.0.0",
		FirmwareBin: image,
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if firmware.Size != 4 {
		t.Fatalf("size = %d, want 4", firmware.Size)
	}

	// The digest must be the keccak-256 of the image bytes.
	raw, err := hexutil.Decode(image)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	expected := hex.EncodeToString(crypto.Keccak256(raw))
	if firmware.SHA256 != expected {
		t.Fatalf("sha256 = %s, want %s", firmware.SHA256, expected)
	}
}

func TestFirmwareVerificationComparesTheImage(t *testing.T) {
	service := NewFirmwareService()

	ctx := context.Background()

	if _, err := service.Upload(ctx, "ledger", "nano-x", UploadFirmwareRequest{
		Version:     "1.0.0",
		FirmwareBin: "0xdeadbeef",
	}); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	good, err := service.Verify(ctx, "ledger", "nano-x", VerifyFirmwareRequest{FirmwareBin: "0xdeadbeef"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if good["valid"] != true {
		t.Fatalf("a matching image was rejected: %v", good)
	}

	bad, err := service.Verify(ctx, "ledger", "nano-x", VerifyFirmwareRequest{FirmwareBin: "0xdeadbeee"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if bad["valid"] != false {
		t.Fatalf("a tampered image was accepted: %v", bad)
	}
}

func TestFirmwareWithoutAnImageIsRejected(t *testing.T) {
	service := NewFirmwareService()

	if _, err := service.Upload(context.Background(), "ledger", "nano-x", UploadFirmwareRequest{
		Version: "1.0.0",
	}); err == nil {
		t.Fatal("a firmware upload without an image was accepted")
	}
}

func TestADeviceCannotPairWithAnotherUsersID(t *testing.T) {
	service := NewDeviceService()

	key, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatalf("the test key is invalid: %v", err)
	}

	device, err := service.Register(context.Background(), RegisterDeviceRequest{
		UserID:    "user-1",
		Vendor:    "ledger",
		PublicKey: hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)),
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if _, err := service.Pair(context.Background(), PairDeviceRequest{
		UserID:   "user-2",
		DeviceID: device.ID,
	}); err == nil {
		t.Fatal("a device was paired under a different user")
	}
}
