package abi

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// The operation used for the canonical vectors: owner 0x742d35Cc...,
// nonce 0, empty init code and calldata, callGasLimit 100000,
// verificationGasLimit 150000, preVerificationGas 21000,
// maxFeePerGas 100 gwei, maxPriorityFeePerGas 1 gwei.
func vectorOp() *UserOperation {
	return &UserOperation{
		Sender:               common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"),
		Nonce:                big.NewInt(0),
		InitCode:             nil,
		CallData:             nil,
		CallGasLimit:         big.NewInt(100_000),
		VerificationGasLimit: big.NewInt(150_000),
		PreVerificationGas:   big.NewInt(21_000),
		MaxFeePerGas:         big.NewInt(100_000_000_000),
		MaxPriorityFeePerGas: big.NewInt(1_000_000_000),
		PaymasterAndData:     nil,
		Signature:            nil,
	}
}

func TestUserOpHashV07MatchesTheDeployedEntryPoint(t *testing.T) {
	// Read from getUserOpHash on the canonical v0.7 EntryPoint at
	// 0x0000000071727De22E5E9d8BAf0edAc6f37da032 (chain 1).
	const want = "bcdd35b5495841f65623e710c01a50266b4083a611885774de6c567f54ee0c99"

	got := UserOpHashV07(vectorOp(), EntryPointV07, big.NewInt(1))

	if hex.EncodeToString(got.Bytes()) != want {
		t.Fatalf("v0.7 hash = %s, want %s", hex.EncodeToString(got.Bytes()), want)
	}
}

func TestUserOpHashV06MatchesTheDeployedEntryPoint(t *testing.T) {
	// Read from getUserOpHash on the canonical v0.6 EntryPoint at
	// 0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789 (chain 1).
	const want = "658bcf6d542bae91bf338b3bdb4165015cab0e824097f813d148a0ce370c110c"

	got := UserOpHashV06(vectorOp(), EntryPointV06, big.NewInt(1))

	if hex.EncodeToString(got.Bytes()) != want {
		t.Fatalf("v0.6 hash = %s, want %s", hex.EncodeToString(got.Bytes()), want)
	}
}

func TestTheHashIsBoundToTheChain(t *testing.T) {
	op := vectorOp()

	onMainnet := UserOpHashV07(op, EntryPointV07, big.NewInt(1))
	onPolygon := UserOpHashV07(op, EntryPointV07, big.NewInt(137))

	if onMainnet == onPolygon {
		t.Fatal("the hash did not change with the chain id")
	}
}

func TestTheHashIsBoundToTheEntryPoint(t *testing.T) {
	op := vectorOp()

	onV07 := UserOpHashV07(op, EntryPointV07, big.NewInt(1))
	onV08 := UserOpHashV07(op, EntryPointV08, big.NewInt(1))

	if onV07 == onV08 {
		t.Fatal("the hash did not change with the entry point")
	}
}

func TestTheVersionsProduceDifferentHashes(t *testing.T) {
	op := vectorOp()

	// The two versions pack the gas fields differently, so the same operation
	// must not hash the same way under both.
	if UserOpHashV06(op, EntryPointV06, big.NewInt(1)) ==
		UserOpHashV07(op, EntryPointV07, big.NewInt(1)) {
		t.Fatal("v0.6 and v0.7 produced the same hash")
	}
}

func TestGasLimitsArePackedHighThenLow(t *testing.T) {
	packed := packUint128Pair(big.NewInt(150_000), big.NewInt(100_000))

	high := new(big.Int).SetBytes(packed[:16])
	low := new(big.Int).SetBytes(packed[16:])

	if high.Int64() != 150_000 {
		t.Fatalf("high half = %d, want 150000", high.Int64())
	}
	if low.Int64() != 100_000 {
		t.Fatalf("low half = %d, want 100000", low.Int64())
	}
}

func TestSelectorsMatchTheCanonicalSignatures(t *testing.T) {
	cases := []struct {
		name     string
		got      []byte
		expected string
	}{
		{"getUserOpHash v0.7", GetUserOpHashV07Selector, "22cdde4c"},
		{"getUserOpHash v0.6", GetUserOpHashV06Selector, "a6193531"},
		{"handleOps v0.7", HandleOpsV07Selector, "765e827f"},
		{"handleOps v0.6", HandleOpsV06Selector, "1fad948c"},
	}

	for _, c := range cases {
		if got := hex.EncodeToString(c.got); got != c.expected {
			t.Errorf("%s selector = %s, want %s", c.name, got, c.expected)
		}
	}
}

func TestHandleOpsCalldataIsWellFormed(t *testing.T) {
	beneficiary := common.HexToAddress("0x00000000000000000000000000000000000000ff")

	encoded, err := EncodeHandleOpsV07([]*UserOperation{vectorOp()}, beneficiary)
	if err != nil {
		t.Fatalf("EncodeHandleOpsV07: %v", err)
	}

	if got := hex.EncodeToString(encoded[:4]); got != "765e827f" {
		t.Fatalf("selector = %s, want 765e827f", got)
	}

	wordAt := func(index int) *big.Int {
		start := 4 + index*32
		return new(big.Int).SetBytes(encoded[start : start+32])
	}

	// Word 0 is the offset of the operations array; word 1 is the beneficiary.
	if wordAt(0).Int64() != 64 {
		t.Fatalf("array offset = %d, want 64", wordAt(0).Int64())
	}
	if wordAt(1).Int64() != 0xff {
		t.Fatalf("beneficiary = %d, want 255", wordAt(1).Int64())
	}
	// Word 2 is the array length.
	if wordAt(2).Int64() != 1 {
		t.Fatalf("array length = %d, want 1", wordAt(2).Int64())
	}
	// Word 3 is the offset of the first element, measured from the length word.
	if wordAt(3).Int64() != 32 {
		t.Fatalf("first element offset = %d, want 32", wordAt(3).Int64())
	}
}

func TestHandleOpsRejectsAnEmptyBatch(t *testing.T) {
	beneficiary := common.HexToAddress("0x00000000000000000000000000000000000000ff")

	if _, err := EncodeHandleOpsV07(nil, beneficiary); err == nil {
		t.Fatal("an empty batch was accepted")
	}
}

func TestHandleOpsRejectsANilOperation(t *testing.T) {
	beneficiary := common.HexToAddress("0x00000000000000000000000000000000000000ff")

	if _, err := EncodeHandleOpsV06([]*UserOperation{nil}, beneficiary); err == nil {
		t.Fatal("a nil operation was accepted")
	}
}

func TestTupleOffsetsPointAtTheDynamicContents(t *testing.T) {
	// The offsets inside a tuple are measured from the start of the tuple, so
	// the first dynamic field must follow the head.
	op := vectorOp()
	op.CallData = []byte{0xaa, 0xbb}

	encoded := encodeOpTupleV07(op)

	offsetAt := func(index int) int64 {
		start := index * 32
		return new(big.Int).SetBytes(encoded[start : start+32]).Int64()
	}

	const headLen = 9 * 32
	if got := offsetAt(2); got != headLen {
		t.Fatalf("initCode offset = %d, want %d", got, headLen)
	}
	// initCode is empty, so it occupies a single length word.
	if got := offsetAt(3); got != headLen+32 {
		t.Fatalf("callData offset = %d, want %d", got, headLen+32)
	}
	// callData holds two bytes: a length word plus one padded data word.
	if got := offsetAt(7); got != headLen+32+64 {
		t.Fatalf("paymasterAndData offset = %d, want %d", got, headLen+32+64)
	}
	// paymasterAndData is empty.
	if got := offsetAt(8); got != headLen+32+64+32 {
		t.Fatalf("signature offset = %d, want %d", got, headLen+32+64+32)
	}
}

func TestDynamicBytesAreLengthPrefixedAndPadded(t *testing.T) {
	encoded := encodeDynamicBytes(make([]byte, 32))
	if len(encoded) != 64 {
		t.Fatalf("32-byte value encoded to %d bytes, want 64", len(encoded))
	}

	// A value that is not a multiple of 32 is padded to the next word.
	encoded = encodeDynamicBytes(make([]byte, 33))
	if len(encoded) != 96 {
		t.Fatalf("33-byte value encoded to %d bytes, want 96", len(encoded))
	}
}
