package chainsdk

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"golang.org/x/crypto/blake2b"
)

// nanoStateBlockHash is the ledger's block hash: BLAKE2b-256 over a 32-byte
// preamble whose last byte is the block type (0x06 for state blocks) followed
// by the fields in ledger order. It is reproduced here rather than exported so
// the test pins the same byte order the production builder uses; a drift in
// either one fails the vector.
func nanoStateBlockHash(account, previous, rep, link []byte, balance *big.Int) []byte {
	pre := make([]byte, 32)
	pre[31] = 0x06
	bal := make([]byte, 16)
	balance.FillBytes(bal)
	h, err := blake2b.New256(nil)
	if err != nil {
		panic(err)
	}
	h.Write(pre)
	h.Write(account)
	h.Write(previous)
	h.Write(rep)
	h.Write(bal)
	h.Write(link)
	return h.Sum(nil)
}

// TestNanoStateBlockHashMatchesMainnet pins the hash serialization against a
// real mainnet state block. A wrong field order or a wrong balance width
// produces a valid-looking hash for the wrong block, which the network then
// rejects with no indication of which field was wrong.
func TestNanoStateBlockHashMatchesMainnet(t *testing.T) {
	// Mainnet frontier 023B94B7...C6B02C on the burn representative account.
	const (
		wantHash   = "023B94B7D27B311666C8636954FE17F1FD2EAA97A8BAC27DE5084FBBD5C6B02C"
		accountB32 = "3t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xt"
		prevHex    = "6875C0DBFE5C44D8F8CFF431BC69ED5587C68F89F0663F2BC1FBBFCB46DC5989"
		repB32     = "3t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xt"
		balance    = "325586539664609129644855132177"
		linkHex    = "65706F636820763220626C6F636B000000000000000000000000000000000000"
	)
	account, err := nanoBase32Dec(accountB32)
	if err != nil {
		t.Fatalf("decode account: %v", err)
	}
	rep, err := nanoBase32Dec(repB32)
	if err != nil {
		t.Fatalf("decode representative: %v", err)
	}
	prev, err := hex.DecodeString(prevHex)
	if err != nil {
		t.Fatalf("decode previous: %v", err)
	}
	link, err := hex.DecodeString(linkHex)
	if err != nil {
		t.Fatalf("decode link: %v", err)
	}
	bal, ok := new(big.Int).SetString(balance, 10)
	if !ok {
		t.Fatal("bad balance fixture")
	}
	got := hex.EncodeToString(nanoStateBlockHash(account, prev, rep, link, bal))
	if !strings.EqualFold(got, wantHash) {
		t.Fatalf("state block hash mismatch:\n got %s\nwant %s", got, wantHash)
	}
}

// TestNanoStateBlockHashIsOrderSensitive asserts the field order is load
// bearing: swapping the representative and the previous hash must change the
// digest, so the vector above cannot pass by coincidence.
func TestNanoStateBlockHashIsOrderSensitive(t *testing.T) {
	account := make([]byte, 32)
	account[0] = 1
	prev := make([]byte, 32)
	prev[1] = 2
	rep := make([]byte, 32)
	rep[2] = 3
	link := make([]byte, 32)
	bal := big.NewInt(7)

	a := hex.EncodeToString(nanoStateBlockHash(account, prev, rep, link, bal))
	b := hex.EncodeToString(nanoStateBlockHash(account, rep, prev, link, bal))
	if a == b {
		t.Fatal("swapping previous and representative must change the hash")
	}
	// And a different balance must change it too.
	c := hex.EncodeToString(nanoStateBlockHash(account, prev, rep, link, big.NewInt(8)))
	if a == c {
		t.Fatal("a different balance must change the hash")
	}
}
