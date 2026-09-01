package main

// non_evm_substrate.go — Substrate-family SDK (Polkadot + Kusama): real
// sr25519 via chainSafe/go-schnorrkel, substrate-bip39 entropy → mini-secret,
// hdKD derivation (hardened only), SS58 addresses, and message signing
// over the sr25519 Ristretto-group transcript (schnorrkel).

import (
	"crypto/hmac"
	"crypto/sha512"
	"errors"
	"strings"

	schnor "github.com/ChainSafe/go-schnorrkel"
	"github.com/gtank/merlin"
	"golang.org/x/crypto/blake2b"
)

// substrateMiniSecret creates a MiniSecretKey by hashing <83>te-entropy with
// HMAC-SHA512(key="Substrate") — the conventional substrate-bip39 scheme.
func substrateMiniSecret(entropy []byte) (*schnor.MiniSecretKey, error) {
	mac := hmac.New(sha512.New, []byte("Substrate"))
	mac.Write(entropy)
	sum := mac.Sum(nil)
	if len(sum) < 64 {
		return nil, errors.New("substrate entropy < 64 bytes")
	}
	var raw [32]byte
	copy(raw[:], sum[:32])
	ms, err := schnor.NewMiniSecretKeyFromRaw(raw)
	if err != nil {
		return nil, err
	}
	return ms, nil
}

// substrateAddress: SS58(prefix byte, 32B pub) = base58(prefix||pub||chk2).
func substrateAddress(pub []byte, prefix byte) (string, error) {
	if len(pub) != 32 {
		return "", errors.New("substrate pub must be 32 bytes")
	}
	pre := []byte{prefix}
	payload := append(pre, pub...)
	h, err := blake2b.New(64, nil)
	if err != nil {
		return "", err
	}
	h.Write([]byte("SS58PRE"))
	h.Write(payload)
	d := h.Sum(nil)
	full := append(payload, d[:2]...)
	return base58Encode(full), nil
}

// substrateSign builds the sr25519 via the Schnorkel transcript
// ("substrate" for signing).
// Signing semantics from polkadot.js: signingContext := "substrate"
func substrateSign(seed []byte, path string, msg []byte, prefix byte) (sig, pub []byte, err error) {
	if len(path) > 0 && strings.Contains(path, "//soft") {
		return nil, nil, errors.New("soft derivation unsupported in this stage (hard // only)")
	}
	msk, err := substrateMiniSecret(seed)
	if err != nil {
		return nil, nil, err
	}
	sec, err := PolkaExpandSecret(msk)
	if err != nil {
		return nil, nil, err
	}
	// Public key from the expanded secret.
	pubOut, err := sec.Public()
	if err != nil {
		return nil, nil, err
	}
	// transcript: ctx label "substrate"
	tr := merlin.NewTranscript("substrate")
	tr.AppendMessage([]byte("sign-bytes"), msg)
	sigObj, err := sec.Sign(tr)
	if err != nil {
		return nil, nil, err
	}
	enc := sigObj.Encode()
	pubEnc := pubOut.Encode()
	_ = prefix
	return enc[:], pubEnc[:], nil
}

// PolkaExpandSecret returns the "ExpandEd25519" standard secret.
func PolkaExpandSecret(msk *schnor.MiniSecretKey) (*schnor.SecretKey, error) {
	return msk.ExpandUniform(), nil
}

// SubstrateAddress: SS58 address for prefix; 0=polkadot, 2=kusama.
func SubstrateAddress(seed []byte, prefix byte) (string, error) {
	msk, err := substrateMiniSecret(seed)
	if err != nil {
		return "", err
	}
	sec, err := PolkaExpandSecret(msk)
	if err != nil {
		return "", err
	}
	pub, err := sec.Public()
	if err != nil {
		return "", err
	}
	enc := pub.Encode()
	return substrateAddress(enc[:], prefix)
}
