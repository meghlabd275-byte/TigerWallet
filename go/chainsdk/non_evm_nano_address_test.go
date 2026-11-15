package chainsdk

import (
	"encoding/hex"
	"strings"
	"testing"
)

// nanoAddressFromPubHexTest renders an address from a hex-encoded key so the
// vectors below can be written the way the network publishes them.
func nanoAddressFromPubHexTest(keyHex, chainType string) (string, error) {
	pub, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", err
	}
	return NanoAddressFromPub(pub, chainType)
}

// TestNanoBase32MainnetVector pins the ledger's base32 against a real account.
//
// The all-zero burn address cannot catch a rotated encoding, because every
// character of it is alphabet[0]. This key is the burn REPRESENTATIVE account,
// whose encoding differs in every character position.
func TestNanoBase32MainnetVector(t *testing.T) {
	pub, err := hex.DecodeString("e89208dd038fbb269987689621d52292ae9c35941a7484756ecced92a65093ba")
	if err != nil {
		t.Fatalf("decode pubkey: %v", err)
	}
	const want = "3t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xt"
	if got := nanoBase32Enc(pub); got != want {
		t.Fatalf("encoded key mismatch:\n got %s\nwant %s", got, want)
	}
	dec, err := nanoBase32Dec(want)
	if err != nil {
		t.Fatalf("nanoBase32Dec: %v", err)
	}
	if len(dec) < 32 || hex.EncodeToString(dec[:32]) != hex.EncodeToString(pub) {
		t.Fatalf("round trip changed the key: %s", hex.EncodeToString(dec))
	}
}

// TestNanoAddressMainnetVector pins the full address, so the checksum and the
// prefix are covered together with the encoding.
func TestNanoAddressMainnetVector(t *testing.T) {
	const want = "nano_3t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xtoncuohr3"
	got, err := nanoAddressFromPubHexTest("e89208dd038fbb269987689621d52292ae9c35941a7484756ecced92a65093ba", "nano")
	if err != nil {
		t.Fatalf("nanoAddressFromPubHex: %v", err)
	}
	if got != want {
		t.Fatalf("address mismatch:\n got %s\nwant %s", got, want)
	}
	// The address must decode back to the same key, and its checksum must be
	// self-consistent: a rotated encoding still passes the checksum, which is
	// exactly why the vector above is required as well.
	back, err := nanoPub(got)
	if err != nil {
		t.Fatalf("nanoPub: %v", err)
	}
	if hex.EncodeToString(back) != "e89208dd038fbb269987689621d52292ae9c35941a7484756ecced92a65093ba" {
		t.Fatalf("decoded key mismatch: %s", hex.EncodeToString(back))
	}
}

// TestNanoAddressFromPubBurnVector is the canonical all-zero address, kept as a second
// independent check of the padding width.
func TestNanoAddressFromPubBurnVector(t *testing.T) {
	const want = "nano_1111111111111111111111111111111111111111111111111111hifc8npp"
	got, err := nanoAddressFromPubHexTest(strings.Repeat("00", 32), "nano")
	if err != nil {
		t.Fatalf("nanoAddressFromPubHex: %v", err)
	}
	if got != want {
		t.Fatalf("burn address mismatch:\n got %s\nwant %s", got, want)
	}
}

// TestNanoAddressPrefixes asserts the prefix is part of the address and that a
// Banano key is not presented as a Nano key.
func TestNanoAddressPrefixes(t *testing.T) {
	const key = "e89208dd038fbb269987689621d52292ae9c35941a7484756ecced92a65093ba"
	nano, err := nanoAddressFromPubHexTest(key, "nano")
	if err != nil {
		t.Fatalf("nano: %v", err)
	}
	ban, err := nanoAddressFromPubHexTest(key, "banano")
	if err != nil {
		t.Fatalf("banano: %v", err)
	}
	if !strings.HasPrefix(nano, "nano_") {
		t.Fatalf("expected a nano_ prefix, got %s", nano)
	}
	if !strings.HasPrefix(ban, "ban_") {
		t.Fatalf("expected a ban_ prefix, got %s", ban)
	}
	// The body after the prefix must be identical; only the prefix differs.
	if strings.SplitN(nano, "_", 2)[1] != strings.SplitN(ban, "_", 2)[1] {
		t.Fatal("nano_ and ban_ must encode the same key body")
	}
	if _, err := nanoAddressFromPubHexTest(key, "dogecoin"); err == nil {
		t.Fatal("an unknown prefix must be rejected")
	}
}

// TestNanoAddressRejectsMalformedInput covers the decode guards.
func TestNanoAddressRejectsMalformedInput(t *testing.T) {
	cases := map[string]string{
		"empty":           "",
		"no separator":    "nano3t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xtoncuohr3",
		"short body":      "nano_3t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xt",
		"bad prefix":      "xrb_3t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xtoncuohr3",
		"bad base32 char": "nano_0t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xtoncuohr3",
	}
	for name, addr := range cases {
		if name == "bad prefix" {
			// xrb_ is a legacy but valid Nano prefix and must be accepted.
			if _, err := nanoPub(addr); err != nil {
				t.Errorf("xrb_ is a legacy Nano prefix and must decode: %v", err)
			}
			continue
		}
		if _, err := nanoPub(addr); err == nil {
			t.Errorf("%s: %q must be rejected", name, addr)
		}
	}
}

// TestNanoAddressRejectsBadChecksum flips one body character of a valid
// address; the checksum must no longer verify.
func TestNanoAddressRejectsBadChecksum(t *testing.T) {
	const good = "nano_3t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xtoncuohr3"
	body := strings.SplitN(good, "_", 2)[1]
	// Corrupt the final checksum character.
	last := body[len(body)-1]
	repl := byte('1')
	if last == repl {
		repl = '3'
	}
	bad := "nano_" + body[:len(body)-1] + string(repl)
	if _, err := nanoPub(bad); err == nil {
		t.Fatalf("a corrupted checksum must be rejected: %s", bad)
	}
}
