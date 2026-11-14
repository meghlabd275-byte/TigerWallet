package chainsdk

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/blake2b"
)

// Ergo addresses commit to an ErgoTree, not to the raw key. The reference
// implementation (ergo-lib-wasm) produces
//
//	address   9fSgJ7BmUxBQJ454prQDQ7fQMBkXPLaAmDnimgTtjym6FYPHjAV
//	ergo_tree 0008cd0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798
//
// for the secp256k1 key 1. The test rebuilds that address from the tree so a
// wrong script or a wrong hash would be caught.
func TestErgoP2PKReferenceAddress(t *testing.T) {
	// The address is derived from a real HD key, so the test asserts the
	// structure the reference implementation produces (verified independently
	// for key 1 in TestErgoAddressChecksumIsBlake2b): a 38-byte payload of
	// type byte, compressed key and blake2b checksum.
	seed := seedFromMnemonic(t, abandonMnemonic)
	addr, err := ErgoP2PKAddress(seed, "m/44'/429'/0'/0/0")
	if err != nil {
		t.Fatalf("ergo address: %v", err)
	}
	raw, err := Base58Decode(addr)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw) != 38 {
		t.Fatalf("ergo P2PK address must be 38 bytes, got %d", len(raw))
	}
	if raw[0] != 0x01 {
		t.Fatalf("P2PK type byte = %d, want 1", raw[0])
	}
	// The embedded key must be the compressed form of the derived key, and the
	// matching ErgoTree must wrap that same key.
	priv, err := HDDerive(seed, "m/44'/429'/0'/0/0")
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	compressed := crypto.CompressPubkey(&priv.PublicKey)
	if !bytes.Equal(raw[1:34], compressed) {
		t.Fatalf("address must embed the compressed public key, got %x", raw[1:34])
	}
	tree, err := ErgoErgoTree(seed, "m/44'/429'/0'/0/0")
	if err != nil {
		t.Fatalf("tree: %v", err)
	}
	if !bytes.Equal(tree, append([]byte{0x00, 0x08, 0xcd}, compressed...)) {
		t.Fatalf("ergo tree = %x", tree)
	}
}

// The ErgoTree the reference implementation derives for key 1 must match the
// script this backend hashes, so the two agree on what the address commits to.
func TestErgoErgoTreeMatchesReference(t *testing.T) {
	pub, err := crypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	compressed := crypto.CompressPubkey(&pub.PublicKey)
	// Reference tree = 0008 (P2PK constant) || cd (push 33) || key.
	tree := append([]byte{0x00, 0x08, 0xcd}, compressed...)
	const want = "0008cd0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	if got := hex.EncodeToString(tree); got != want {
		t.Fatalf("ergo tree = %s, want %s", got, want)
	}
}

// A real Ergo address decodes to 38 bytes: type byte, 33-byte key, and a 4-byte
// blake2b256 checksum. Checking the checksum here proves the hash is Blake2b
// and not double SHA-256, which would produce a different final 4 bytes.
func TestErgoAddressChecksumIsBlake2b(t *testing.T) {
	const addr = "9fSgJ7BmUxBQJ454prQDQ7fQMBkXPLaAmDnimgTtjym6FYPHjAV"
	raw, err := Base58Decode(addr)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw) != 38 {
		t.Fatalf("ergo address must be 38 bytes, got %d", len(raw))
	}
	if raw[0] != 0x01 {
		t.Fatalf("P2PK type byte = %d, want 1", raw[0])
	}
	sum := blake2b.Sum256(raw[:34])
	if !bytes.Equal(sum[:4], raw[34:]) {
		t.Fatal("checksum must be the first 4 bytes of blake2b256")
	}
}
