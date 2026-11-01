package main

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// newTestService builds the minimum state needed to exercise signature checks.
func newTestService(forwarder string) *GaslessService {
	tokenABIJSON := `[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"}]`
	forwarderABIJSON := `[{"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"},{"name":"gasLimit","type":"uint256"},{"name":"relayer","type":"address"},{"name":"signature","type":"bytes"}],"name":"executeMetaTransaction","outputs":[{"name":"","type":"bytes"}],"stateMutability":"payable","type":"function"}]`
	tokenABI, err := abi.JSON(strings.NewReader(tokenABIJSON))
	if err != nil {
		panic(err)
	}
	fwABI, err := abi.JSON(strings.NewReader(forwarderABIJSON))
	if err != nil {
		panic(err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	return &GaslessService{
		config: &Config{
			ForwarderAddress: forwarder,
			ForwarderName:    "TigerWallet Gasless",
			ForwarderVersion: "1.0.0",
		},
		relayerKey:   key,
		relayerAddr:  crypto.PubkeyToAddress(key.PublicKey),
		tokenABI:     tokenABI,
		forwarderABI: fwABI,
	}
}

// TestVerifyUserSignatureRejectsForgedSignature is the regression guard for the
// original stub that accepted any signature.
func TestVerifyUserSignatureRejectsForgedSignature(t *testing.T) {
	forwarder := "0x1111111111111111111111111111111111111111"
	s := newTestService(forwarder)

	userKey, _ := crypto.GenerateKey()
	userAddr := crypto.PubkeyToAddress(userKey.PublicKey)
	token := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	amount := big.NewInt(1000)
	gasLimit := uint64(65000)
	chainID := uint64(1)
	nonce := uint64(0)

	// A signature that is not a valid 65-byte ECDSA signature must be rejected.
	if s.verifyUserSignature(userAddr, token, recipient.Hex(), amount, gasLimit, chainID, nonce, "0xdeadbeef") {
		t.Fatal("malformed signature was accepted")
	}

	// A structurally valid but unrelated signature must also be rejected.
	digest := s.metaTxDigest(userAddr, token, recipient, amount, gasLimit, chainID, nonce)
	otherKey, _ := crypto.GenerateKey()
	badSig, _ := crypto.Sign(digest.Bytes(), otherKey)
	if s.verifyUserSignature(userAddr, token, recipient.Hex(), amount, gasLimit, chainID, nonce, "0x"+hex.EncodeToString(badSig)) {
		t.Fatal("signature from the wrong key was accepted")
	}
}

// TestVerifyUserSignatureAcceptsRealSignature proves a correctly signed
// meta-transaction verifies, so the guard is not simply rejecting everything.
func TestVerifyUserSignatureAcceptsRealSignature(t *testing.T) {
	forwarder := "0x1111111111111111111111111111111111111111"
	s := newTestService(forwarder)

	userKey, _ := crypto.GenerateKey()
	userAddr := crypto.PubkeyToAddress(userKey.PublicKey)
	token := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	amount := big.NewInt(1000)
	gasLimit := uint64(65000)
	chainID := uint64(1)
	nonce := uint64(0)

	digest := s.metaTxDigest(userAddr, token, recipient, amount, gasLimit, chainID, nonce)
	sig, err := crypto.Sign(digest.Bytes(), userKey)
	if err != nil {
		t.Fatal(err)
	}
	sig[64] += 27 // wallets emit the recovery id as 27/28

	if !s.verifyUserSignature(userAddr, token, recipient.Hex(), amount, gasLimit, chainID, nonce, "0x"+hex.EncodeToString(sig)) {
		t.Fatal("a valid signature was rejected")
	}
}

// TestVerifyUserSignatureFailsClosedWithoutForwarder ensures an unconfigured
// deployment refuses meta-transactions instead of trusting them.
func TestVerifyUserSignatureFailsClosedWithoutForwarder(t *testing.T) {
	s := newTestService("")
	userKey, _ := crypto.GenerateKey()
	userAddr := crypto.PubkeyToAddress(userKey.PublicKey)
	if s.verifyUserSignature(userAddr, common.Address{}, "0x2222222222222222222222222222222222222222",
		big.NewInt(1), 65000, 1, 0, "0x00") {
		t.Fatal("verification passed with no forwarder configured")
	}
}

// TestVerifyUserSignatureBindsToNonce ensures a replayed signature for a
// different nonce is not accepted.
func TestVerifyUserSignatureBindsToNonce(t *testing.T) {
	forwarder := "0x1111111111111111111111111111111111111111"
	s := newTestService(forwarder)

	userKey, _ := crypto.GenerateKey()
	userAddr := crypto.PubkeyToAddress(userKey.PublicKey)
	token := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	amount := big.NewInt(1000)

	digest := s.metaTxDigest(userAddr, token, recipient, amount, 65000, 1, 0)
	sig, _ := crypto.Sign(digest.Bytes(), userKey)

	if s.verifyUserSignature(userAddr, token, recipient.Hex(), amount, 65000, 1, 1, "0x"+hex.EncodeToString(sig)) {
		t.Fatal("signature for nonce 0 was accepted for nonce 1")
	}
}

// TestVerifyUserSignatureBindsToRecipient ensures the signer authorises the
// exact recipient, so the relayer cannot redirect funds.
func TestVerifyUserSignatureBindsToRecipient(t *testing.T) {
	forwarder := "0x1111111111111111111111111111111111111111"
	s := newTestService(forwarder)

	userKey, _ := crypto.GenerateKey()
	userAddr := crypto.PubkeyToAddress(userKey.PublicKey)
	token := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	attacker := "0x3333333333333333333333333333333333333333"
	amount := big.NewInt(1000)

	digest := s.metaTxDigest(userAddr, token, recipient, amount, 65000, 1, 0)
	sig, _ := crypto.Sign(digest.Bytes(), userKey)

	if s.verifyUserSignature(userAddr, token, attacker, amount, 65000, 1, 0, "0x"+hex.EncodeToString(sig)) {
		t.Fatal("signature for one recipient was accepted for another")
	}
}

func TestDomainSeparatorDependsOnChain(t *testing.T) {
	s := newTestService("0x1111111111111111111111111111111111111111")
	if s.domainSeparator(1) == s.domainSeparator(137) {
		t.Fatal("domain separator must differ per chain")
	}
}

func TestTrimHexPrefix(t *testing.T) {
	if got := strings.TrimPrefix("0xabcd", "0x"); got != "abcd" {
		t.Fatalf("unexpected trim result %q", got)
	}
}
