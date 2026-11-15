package chainsdk

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"time"
)

// The ledger hashes the nonce FIRST, little-endian, then the 32-byte root:
//
//	value = little_endian_u64( BLAKE2b-8( nonce_le_8 || root ) )
//
// This is mainnet block 494dbb4e8bd688aa, whose work the node's work_validate
// reports as difficulty fffffe16a782ca82 against its own previous hash. It pins
// the digest size (8), the absence of a key, the input order and both byte
// orders at once, and it was taken from the live network rather than assumed.
func TestNanoWorkValueMatchesMainnetBlock(t *testing.T) {
	root, err := hex.DecodeString("6875c0dbfe5c44d8f8cff431bc69ed5587c68f89f0663f2bc1fbbfcb46dc5989")
	if err != nil {
		t.Fatalf("decode root: %v", err)
	}
	nonce, err := nanoWorkParse("494dbb4e8bd688aa")
	if err != nil {
		t.Fatalf("parse work: %v", err)
	}
	got, err := NanoWorkValue(root, nonce)
	if err != nil {
		t.Fatalf("NanoWorkValue: %v", err)
	}
	const want uint64 = 0xfffffe16a782ca82
	if got != want {
		t.Fatalf("work value mismatch: got %016x want %016x", got, want)
	}
}

// A second independent vector: the zero root with the zero nonce.
func TestNanoWorkValueZeroRoot(t *testing.T) {
	got, err := NanoWorkValue(make([]byte, 32), 0)
	if err != nil {
		t.Fatalf("NanoWorkValue: %v", err)
	}
	const want uint64 = 0xd60fba25f7d14965
	if got != want {
		t.Fatalf("work value mismatch: got %016x want %016x", got, want)
	}
}

// A nonce must be rejected when it does not reach the threshold and accepted
// when it does. Using the nonce's own value as the threshold makes the boundary
// exact rather than probabilistic.
func TestNanoWorkValidBoundary(t *testing.T) {
	root := make([]byte, 32)
	root[0] = 7
	value, err := NanoWorkValue(root, 12345)
	if err != nil {
		t.Fatalf("NanoWorkValue: %v", err)
	}
	if !NanoWorkValid(root, 12345, value) {
		t.Fatal("a nonce equal to its own work value must be valid")
	}
	if NanoWorkValid(root, 12345, value+1) {
		t.Fatal("a threshold above the nonce work value must be invalid")
	}
}

// A root that is not 32 bytes must be rejected rather than hashed.
func TestNanoWorkRejectsBadRootLength(t *testing.T) {
	if NanoWorkValid(make([]byte, 31), 0, 1) {
		t.Fatal("a 31-byte root must be rejected")
	}
	if _, err := NanoWorkValue(make([]byte, 33), 0); err == nil {
		t.Fatal("a 33-byte root must be an error")
	}
	if _, err := NanoWorkGenerate(context.Background(), make([]byte, 16), 0, 2); err == nil {
		t.Fatal("NanoWorkGenerate must reject a bad root length")
	}
}

// Generation must produce a nonce that genuinely meets the ledger threshold.
func TestNanoWorkGenerateMeetsThreshold(t *testing.T) {
	root := make([]byte, 32)
	for i := range root {
		root[i] = byte(i * 3)
	}
	start := time.Now()
	nonce, err := NanoWorkGenerate(context.Background(), root, nanoWorkThresholdSend, 4)
	if err != nil {
		t.Fatalf("NanoWorkGenerate: %v", err)
	}
	if !NanoWorkValid(root, nonce, nanoWorkThresholdSend) {
		t.Fatalf("generated nonce %016x does not meet the send threshold", nonce)
	}
	value, _ := NanoWorkValue(root, nonce)
	t.Logf("nonce=%016x value=%016x in %s", nonce, value, time.Since(start))
}

// An impossible threshold must end on cancellation rather than search forever.
func TestNanoWorkGenerateHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := NanoWorkGenerate(ctx, make([]byte, 32), ^uint64(0), 4); err == nil {
		t.Fatal("an unsatisfiable threshold must fail once the context expires")
	}
}

// Generation must be safe to run concurrently.
func TestNanoWorkGenerateConcurrent(t *testing.T) {
	root := make([]byte, 32)
	root[1] = 9
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nonce, err := NanoWorkGenerate(context.Background(), root, nanoWorkThresholdReceive, 2)
			if err != nil {
				t.Errorf("NanoWorkGenerate: %v", err)
				return
			}
			if !NanoWorkValid(root, nonce, nanoWorkThresholdReceive) {
				t.Errorf("nonce %016x does not meet the receive threshold", nonce)
			}
		}()
	}
	wg.Wait()
}

// The node reports work as big-endian hex, so rendering must round-trip through
// nanoWorkParse back to the same nonce.
func TestNanoWorkHexRoundTrip(t *testing.T) {
	const nonce = 0x0123456789abcdef
	h := nanoWorkHex(nonce)
	if len(h) != 16 {
		t.Fatalf("expected 16 hex characters, got %d (%q)", len(h), h)
	}
	if !strings.EqualFold(h, "0123456789abcdef") {
		t.Fatalf("expected big-endian byte order, got %q", h)
	}
	back, err := nanoWorkParse(h)
	if err != nil {
		t.Fatalf("nanoWorkParse: %v", err)
	}
	if back != nonce {
		t.Fatalf("round trip changed the nonce: %016x", back)
	}
	if _, err := nanoWorkParse("0011"); err == nil {
		t.Fatal("a short work string must be rejected")
	}
	if _, err := nanoWorkParse("zzzzzzzzzzzzzzzz"); err == nil {
		t.Fatal("a non-hex work string must be rejected")
	}
}

// The receive threshold must be strictly easier than the send threshold.
func TestNanoWorkThresholdOrdering(t *testing.T) {
	if nanoWorkThresholdReceive >= nanoWorkThresholdSend {
		t.Fatalf("receive threshold %016x must be below send threshold %016x",
			nanoWorkThresholdReceive, nanoWorkThresholdSend)
	}
}

// The big-endian hex form must hash identically to the value the nonce
// represents: this is the property that made the byte order a real bug.
func TestNanoWorkHexAndValueAgree(t *testing.T) {
	root := make([]byte, 32)
	root[3] = 0x2a
	nonce, err := NanoWorkGenerate(context.Background(), root, nanoWorkThresholdReceive, 2)
	if err != nil {
		t.Fatalf("NanoWorkGenerate: %v", err)
	}
	rendered := nanoWorkHex(nonce)
	raw, err := hex.DecodeString(rendered)
	if err != nil {
		t.Fatalf("decode rendered work: %v", err)
	}
	if binary.BigEndian.Uint64(raw) != nonce {
		t.Fatalf("rendered work %q does not represent nonce %016x", rendered, nonce)
	}
	if !NanoWorkValid(root, nonce, nanoWorkThresholdReceive) {
		t.Fatal("the rendered nonce must validate against its root")
	}
}
