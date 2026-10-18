package main

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// TestTSSThresholdSignatureIsReal verifies that the 2-of-3 TSS engine produces
// a genuine ECDSA signature that recovers to the group public key, and that a
// sub-threshold quorum cannot sign.
func TestTSSThresholdSignatureIsReal(t *testing.T) {
	engine := NewTSSEngine(2, 3)
	shares := engine.GenerateKeyShares()
	if len(shares) != 3 {
		t.Fatalf("expected 3 shares, got %d", len(shares))
	}
	pub := engine.GetPublicKey()
	if len(pub) != 65 {
		t.Fatalf("expected 65-byte uncompressed group pubkey, got %d", len(pub))
	}
	addr := DeriveAddress(pub)
	if addr == "0x0000000000000000000000000000000000000000" {
		t.Fatal("group address derivation failed")
	}

	msgHash := ethcrypto.Keccak256([]byte("tigerwallet threshold signing test"))

	// Two shares satisfy the threshold.
	var sigShares []SignatureShare
	for party := uint32(1); party <= 2; party++ {
		if _, err := engine.SignWithShare(party, msgHash); err != nil {
			t.Fatalf("partial sign party %d: %v", party, err)
		}
		scalar, ok := engine.ShareFor(party)
		if !ok {
			t.Fatalf("share %d missing", party)
		}
		sigShares = append(sigShares, SignatureShare{PartyID: party, Share: scalarBytes(scalar)})
	}

	sig := engine.CombineShares(sigShares, addr, msgHash)
	if len(sig.Signature) != 65 {
		t.Fatalf("expected 65-byte signature, got %d", len(sig.Signature))
	}
	// Signature must recover to the group public key.
	recovered, err := ethcrypto.Ecrecover(msgHash, sig.Signature)
	if err != nil {
		t.Fatalf("ecrecover failed: %v", err)
	}
	if hex.EncodeToString(recovered) != hex.EncodeToString(pub) {
		t.Fatalf("recovered pubkey %s != group pubkey %s",
			hex.EncodeToString(recovered), hex.EncodeToString(pub))
	}

	// A single share (below threshold) must not produce a signature.
	one := engine.CombineShares(sigShares[:1], addr, msgHash)
	if len(one.Signature) != 0 {
		t.Fatal("sub-threshold quorum produced a signature")
	}
}

// TestShamirSplitRecoverRoundTrip checks the Shamir/Lagrange primitives
// reconstruct the exact secret from any threshold-sized share subset.
func TestShamirSplitRecoverRoundTrip(t *testing.T) {
	secret, err := randScalar()
	if err != nil {
		t.Fatal(err)
	}
	shares, err := shamirSplit(secret, 3, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 5 {
		t.Fatalf("expected 5 shares, got %d", len(shares))
	}
	// Every 3-of-5 subset must recover the secret.
	for _, subset := range [][]uint32{{1, 2, 3}, {1, 3, 5}, {2, 4, 5}, {1, 4, 5}} {
		pts := map[uint32]*big.Int{}
		for _, id := range subset {
			pts[id] = shares[id]
		}
		got, err := lagrangeRecover(pts)
		if err != nil {
			t.Fatalf("recover %v: %v", subset, err)
		}
		if got.Cmp(secret) != 0 {
			t.Fatalf("subset %v recovered wrong secret", subset)
		}
	}
}

// TestDeriveAddressMatchesEthereum ensures the derived address equals the
// canonical keccak256(pubkey[1:])[12:] derivation.
func TestDeriveAddressMatchesEthereum(t *testing.T) {
	priv, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub := ethcrypto.FromECDSAPub(&priv.PublicKey)
	want := strings.ToLower(ethcrypto.PubkeyToAddress(priv.PublicKey).Hex())
	if got := strings.ToLower(string(DeriveAddress(pub))); got != want {
		t.Fatalf("DeriveAddress=%s want %s", got, want)
	}
	// Compressed form must derive the same address.
	compressed := ethcrypto.CompressPubkey(&priv.PublicKey)
	if got := strings.ToLower(string(DeriveAddress(compressed))); got != want {
		t.Fatalf("compressed DeriveAddress=%s want %s", got, want)
	}
}
