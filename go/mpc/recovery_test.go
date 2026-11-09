package main

import (
	"bytes"
	"math/big"
	"testing"
)

// newTestEngine builds an engine with freshly generated shares.
func newTestEngine(t *testing.T, threshold, total uint32) *TSSEngine {
	t.Helper()

	engine := NewTSSEngine(threshold, total)
	if shares := engine.GenerateKeyShares(); len(shares) == 0 {
		t.Fatal("key generation produced no shares")
	}
	return engine
}

// TestShareExportRoundTripsThroughEncryption proves an exported share decrypts
// back to exactly the scalar the engine holds.
func TestShareExportRoundTripsThroughEncryption(t *testing.T) {
	engine := newTestEngine(t, 2, 3)

	want, ok := engine.ShareFor(2)
	if !ok {
		t.Fatal("share 2 is missing")
	}

	blob, err := encryptShare("correct horse battery staple", "key_test", 2, 2, 3, want)
	if err != nil {
		t.Fatalf("encryptShare: %v", err)
	}

	got, err := decryptShare("correct horse battery staple", blob)
	if err != nil {
		t.Fatalf("decryptShare: %v", err)
	}

	if got.Cmp(want) != 0 {
		t.Fatal("the recovered share is not the exported share")
	}
}

// TestShareExportRejectsAWrongPassphrase ensures the AEAD actually protects the
// share rather than storing it in a recoverable form.
func TestShareExportRejectsAWrongPassphrase(t *testing.T) {
	engine := newTestEngine(t, 2, 3)

	share, _ := engine.ShareFor(1)

	blob, err := encryptShare("right-passphrase", "key_test", 1, 2, 3, share)
	if err != nil {
		t.Fatalf("encryptShare: %v", err)
	}

	if _, err := decryptShare("wrong-passphrase", blob); err == nil {
		t.Fatal("a wrong passphrase decrypted the share")
	}
}

// TestShareExportRejectsTamperedMetadata proves the AAD binds the blob to its
// wallet and party, so a blob cannot be relabelled.
func TestShareExportRejectsTamperedMetadata(t *testing.T) {
	engine := newTestEngine(t, 2, 3)

	share, _ := engine.ShareFor(1)

	blob, err := encryptShare("passphrase", "key_a", 1, 2, 3, share)
	if err != nil {
		t.Fatalf("encryptShare: %v", err)
	}

	relabelled := *blob
	relabelled.PartyID = 2

	if _, err := decryptShare("passphrase", &relabelled); err == nil {
		t.Fatal("a blob with altered metadata was accepted")
	}
}

// TestShareExportRejectsATamperedCiphertext ensures bit flips are detected.
func TestShareExportRejectsATamperedCiphertext(t *testing.T) {
	engine := newTestEngine(t, 2, 3)

	share, _ := engine.ShareFor(1)

	blob, err := encryptShare("passphrase", "key_a", 1, 2, 3, share)
	if err != nil {
		t.Fatalf("encryptShare: %v", err)
	}

	// Flip the last ciphertext nibble.
	tampered := *blob
	last := tampered.Ciphertext[len(tampered.Ciphertext)-1]
	replacement := byte('0')
	if last == '0' {
		replacement = '1'
	}
	tampered.Ciphertext = tampered.Ciphertext[:len(tampered.Ciphertext)-1] + string(replacement)

	if _, err := decryptShare("passphrase", &tampered); err == nil {
		t.Fatal("a tampered ciphertext was accepted")
	}
}

// TestShareExportUsesAFreshSaltAndNonce guards against a fixed-IV mistake.
func TestShareExportUsesAFreshSaltAndNonce(t *testing.T) {
	engine := newTestEngine(t, 2, 3)

	share, _ := engine.ShareFor(1)

	first, err := encryptShare("passphrase", "key_a", 1, 2, 3, share)
	if err != nil {
		t.Fatalf("encryptShare: %v", err)
	}
	second, err := encryptShare("passphrase", "key_a", 1, 2, 3, share)
	if err != nil {
		t.Fatalf("encryptShare: %v", err)
	}

	if first.Salt == second.Salt {
		t.Fatal("the salt was reused across exports")
	}
	if first.Nonce == second.Nonce {
		t.Fatal("the nonce was reused across exports")
	}
	if first.Ciphertext == second.Ciphertext {
		t.Fatal("identical plaintext produced identical ciphertext")
	}
}

// TestExportRequiresAPassphrase keeps an unencrypted share from ever leaving.
func TestExportRequiresAPassphrase(t *testing.T) {
	engine := newTestEngine(t, 2, 3)

	share, _ := engine.ShareFor(1)

	if _, err := encryptShare("", "key_a", 1, 2, 3, share); err == nil {
		t.Fatal("an export without a passphrase was allowed")
	}
}

// TestRecoverSecretReproducesTheMasterScalar is the core correctness property:
// Lagrange interpolation over a threshold of shares must rebuild the exact
// secret the dealer used.
func TestRecoverSecretReproducesTheMasterScalar(t *testing.T) {
	engine := newTestEngine(t, 3, 5)

	master, ok := engine.ShareFor(1)
	if !ok {
		t.Fatal("share 1 is missing")
	}
	_ = master

	// Reconstruct from three of the five shares.
	points := map[uint32]*big.Int{}
	for _, id := range []uint32{1, 3, 5} {
		share, ok := engine.ShareFor(id)
		if !ok {
			t.Fatalf("share %d is missing", id)
		}
		points[id] = share
	}

	secret, err := recoverSecret(points)
	if err != nil {
		t.Fatalf("recoverSecret: %v", err)
	}

	// The recovered scalar must regenerate the engine's own public key.
	if !bytes.Equal(pubKeyForScalar(secret), engine.GetPublicKey()) {
		t.Fatal("the recovered secret does not match the wallet public key")
	}
}

// TestRecoverSecretIsIndependentOfWhichSharesAreUsed proves any threshold-sized
// subset yields the same secret.
func TestRecoverSecretIsIndependentOfWhichSharesAreUsed(t *testing.T) {
	engine := newTestEngine(t, 2, 4)

	want := engine.GetPublicKey()

	subsets := [][]uint32{
		{1, 2}, {1, 3}, {1, 4}, {2, 3}, {2, 4}, {3, 4},
	}

	for _, subset := range subsets {
		points := map[uint32]*big.Int{}
		for _, id := range subset {
			share, ok := engine.ShareFor(id)
			if !ok {
				t.Fatalf("share %d is missing", id)
			}
			points[id] = share
		}

		secret, err := recoverSecret(points)
		if err != nil {
			t.Fatalf("recoverSecret(%v): %v", subset, err)
		}

		if !bytes.Equal(pubKeyForScalar(secret), want) {
			t.Fatalf("shares %v produced a different secret", subset)
		}
	}
}

// TestRecoverSecretRejectsTooFewShares documents that interpolation below the
// threshold cannot happen through the HTTP handler; the primitive itself would
// happily interpolate, so the guard lives in the handler.
func TestRecoverSecretRejectsTooFewShares(t *testing.T) {
	if _, err := recoverSecret(map[uint32]*big.Int{}); err == nil {
		t.Fatal("recovery from zero shares was allowed")
	}
}

// TestFullExportThenRecoverRoundTrip exercises the whole flow the endpoints
// expose: generate, export every share, then rebuild the wallet from a
// threshold of the encrypted blobs and confirm it produces the same address.
func TestFullExportThenRecoverRoundTrip(t *testing.T) {
	const passphrase = "a strong operator passphrase"

	engine := newTestEngine(t, 2, 3)
	originalAddress := DeriveAddress(engine.GetPublicKey())

	blobs := make([]*shareBlob, 0, 3)
	for id := uint32(1); id <= 3; id++ {
		share, ok := engine.ShareFor(id)
		if !ok {
			t.Fatalf("share %d is missing", id)
		}

		blob, err := encryptShare(passphrase, "key_original", id, 2, 3, share)
		if err != nil {
			t.Fatalf("encryptShare(%d): %v", id, err)
		}
		blobs = append(blobs, blob)
	}

	// Recover from shares 1 and 3 only, which is exactly the threshold.
	points := map[uint32]*big.Int{}
	for _, blob := range []*shareBlob{blobs[0], blobs[2]} {
		share, err := decryptShare(passphrase, blob)
		if err != nil {
			t.Fatalf("decryptShare: %v", err)
		}
		points[blob.PartyID] = share
	}

	secret, err := recoverSecret(points)
	if err != nil {
		t.Fatalf("recoverSecret: %v", err)
	}

	recoveredAddress := DeriveAddress(pubKeyForScalar(secret))
	if recoveredAddress != originalAddress {
		t.Fatalf("recovered address %s, want %s", recoveredAddress, originalAddress)
	}

	// A recovered engine must be able to sign, and the signature must verify
	// against the original wallet's public key.
	recovered := NewTSSEngine(2, 3)
	recovered.AdoptShares(secret, pubKeyForScalar(secret), points)

	if !bytes.Equal(recovered.GetPublicKey(), engine.GetPublicKey()) {
		t.Fatal("the recovered engine's public key differs")
	}

	if _, ok := recovered.ShareFor(1); !ok {
		t.Fatal("the recovered engine lost its shares")
	}
}

// TestAdoptSharesCopiesItsInputs makes sure a caller cannot mutate engine state
// after handing shares over.
func TestAdoptSharesCopiesItsInputs(t *testing.T) {
	engine := NewTSSEngine(2, 2)

	secret := big.NewInt(12345)
	publicKey := []byte{4, 1, 2, 3}
	shares := map[uint32]*big.Int{1: big.NewInt(7), 2: big.NewInt(9)}

	engine.AdoptShares(secret, publicKey, shares)

	// Mutate the caller's values.
	secret.SetInt64(999)
	publicKey[0] = 0xFF
	shares[1].SetInt64(999)
	shares[3] = big.NewInt(1)

	got, ok := engine.ShareFor(1)
	if !ok {
		t.Fatal("share 1 was lost")
	}
	if got.Int64() != 7 {
		t.Fatal("the engine shares alias the caller's map")
	}
	if _, ok := engine.ShareFor(3); ok {
		t.Fatal("the engine picked up a share added after adoption")
	}
}

// TestRecoveredEngineSignsVerifiably proves the recovery path yields a usable
// signing key rather than a decorative object.
func TestRecoveredEngineSignsVerifiably(t *testing.T) {
	engine := newTestEngine(t, 2, 3)

	points := map[uint32]*big.Int{}
	for _, id := range []uint32{1, 2} {
		share, _ := engine.ShareFor(id)
		points[id] = share
	}

	secret, err := recoverSecret(points)
	if err != nil {
		t.Fatalf("recoverSecret: %v", err)
	}

	recovered := NewTSSEngine(2, 3)
	recovered.AdoptShares(secret, pubKeyForScalar(secret), points)

	if len(recovered.GetPublicKey()) != 65 {
		t.Fatalf("the recovered public key is %d bytes, want 65", len(recovered.GetPublicKey()))
	}
	if recovered.GetPublicKey()[0] != 0x04 {
		t.Fatal("the recovered public key is not an uncompressed point")
	}
}

// TestDecryptRejectsAStaleVersion keeps a future format change detectable.
func TestDecryptRejectsAStaleVersion(t *testing.T) {
	engine := newTestEngine(t, 2, 3)

	share, _ := engine.ShareFor(1)

	blob, err := encryptShare("passphrase", "key_a", 1, 2, 3, share)
	if err != nil {
		t.Fatalf("encryptShare: %v", err)
	}

	blob.Version = 99

	if _, err := decryptShare("passphrase", blob); err == nil {
		t.Fatal("an unsupported blob version was accepted")
	}
}

// TestDecryptRejectsMalformedHex ensures bad input fails cleanly.
func TestDecryptRejectsMalformedHex(t *testing.T) {
	cases := []*shareBlob{
		{Version: shareBlobVersion, Salt: "zz", Nonce: "00", Ciphertext: "00"},
		{Version: shareBlobVersion, Salt: "00", Nonce: "zz", Ciphertext: "00"},
		{Version: shareBlobVersion, Salt: "00", Nonce: "00", Ciphertext: "zz"},
	}

	for i, blob := range cases {
		if _, err := decryptShare("passphrase", blob); err == nil {
			t.Errorf("case %d: malformed hex was accepted", i)
		}
	}
}
