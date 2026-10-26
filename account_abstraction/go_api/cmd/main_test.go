package main

import (
	"encoding/hex"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/tigerwallet/account-abstraction-bundler/abi"
)

// vectorOp is the operation used for the canonical vectors: owner
// 0x742d35Cc..., nonce 0, empty init code and calldata, callGasLimit 100000,
// verificationGasLimit 150000, preVerificationGas 21000, maxFeePerGas 100 gwei,
// maxPriorityFeePerGas 1 gwei.
func vectorOp() *abi.UserOperation {
	return &abi.UserOperation{
		Sender:               common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"),
		Nonce:                big.NewInt(0),
		CallGasLimit:         big.NewInt(100_000),
		VerificationGasLimit: big.NewInt(150_000),
		PreVerificationGas:   big.NewInt(21_000),
		MaxFeePerGas:         big.NewInt(100_000_000_000),
		MaxPriorityFeePerGas: big.NewInt(1_000_000_000),
	}
}

func TestTheHashFollowsTheConfiguredEntryPointVersion(t *testing.T) {
	// Read from getUserOpHash on the deployed EntryPoints (chain 1). If the
	// bundler reported a hash the EntryPoint does not compute, every signature
	// it collected would be rejected on-chain.
	cases := []struct {
		name       string
		entryPoint common.Address
		want       string
	}{
		{
			name:       "v0.6",
			entryPoint: abi.EntryPointV06,
			want:       "658bcf6d542bae91bf338b3bdb4165015cab0e824097f813d148a0ce370c110c",
		},
		{
			name:       "v0.7",
			entryPoint: abi.EntryPointV07,
			want:       "bcdd35b5495841f65623e710c01a50266b4083a611885774de6c567f54ee0c99",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			api := &BundlerAPI{config: &Config{EntryPoint: c.entryPoint}}

			got := api.userOpHash(vectorOp(), big.NewInt(1))

			if hex.EncodeToString(got.Bytes()) != c.want {
				t.Fatalf("hash = %s, want %s", hex.EncodeToString(got.Bytes()), c.want)
			}
		})
	}
}

func TestConfigRejectsAnInvalidEntryPoint(t *testing.T) {
	t.Setenv("RPC_URL", "http://127.0.0.1:1")
	t.Setenv("ENTRY_POINT", "0x1234")

	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("an invalid ENTRY_POINT was accepted")
	}
}

func TestConfigRequiresAnRPCURL(t *testing.T) {
	os.Unsetenv("RPC_URL")

	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("a missing RPC_URL was accepted")
	}
}

func TestConfigDefaultsToTheCanonicalEntryPoint(t *testing.T) {
	t.Setenv("RPC_URL", "http://127.0.0.1:1")
	t.Setenv("ENTRY_POINT", "")

	config, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}

	if config.EntryPoint != abi.EntryPointV06 {
		t.Fatalf("entry point = %s, want the canonical v0.6 address", config.EntryPoint.Hex())
	}
}

func TestMempoolRejectsAnUnsignedOperation(t *testing.T) {
	pool := NewMempoolService(4)

	op := &UserOperation{Sender: common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")}

	if err := pool.Add(common.HexToHash("0x01"), op); err == nil {
		t.Fatal("an unsigned operation was accepted")
	}
}

func TestMempoolRejectsAZeroSender(t *testing.T) {
	pool := NewMempoolService(4)

	op := &UserOperation{Signature: "0x1234"}

	if err := pool.Add(common.HexToHash("0x01"), op); err == nil {
		t.Fatal("an operation with a zero sender was accepted")
	}
}

func TestMempoolRejectsADuplicate(t *testing.T) {
	pool := NewMempoolService(4)

	hash := common.HexToHash("0x01")
	op := &UserOperation{
		Sender:    common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"),
		Signature: "0x1234",
	}

	if err := pool.Add(hash, op); err != nil {
		t.Fatalf("the first add failed: %v", err)
	}
	if err := pool.Add(hash, op); err == nil {
		t.Fatal("a duplicate operation was accepted")
	}
}

func TestMempoolIsBounded(t *testing.T) {
	pool := NewMempoolService(1)

	op := &UserOperation{
		Sender:    common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"),
		Signature: "0x1234",
	}

	if err := pool.Add(common.HexToHash("0x01"), op); err != nil {
		t.Fatalf("the first add failed: %v", err)
	}
	if err := pool.Add(common.HexToHash("0x02"), op); err == nil {
		t.Fatal("the mempool accepted an operation past its bound")
	}
}

func TestMempoolLookupAcceptsEitherHexCasing(t *testing.T) {
	pool := NewMempoolService(4)

	hash := common.HexToHash("0xABCDEF")
	op := &UserOperation{
		Sender:    common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"),
		Signature: "0x1234",
	}

	if err := pool.Add(hash, op); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if pool.Get(hash.Hex()) == nil {
		t.Fatal("the operation was not found by its checksummed hash")
	}
}

func TestToABIRejectsMalformedNumbers(t *testing.T) {
	op := &UserOperation{
		Sender: common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"),
		Nonce:  "not-a-number",
	}

	if _, err := op.toABI(); err == nil {
		t.Fatal("a non-numeric nonce was accepted")
	}
}

func TestToABIReadsHexQuantities(t *testing.T) {
	// Bundler clients send quantities as hex as often as decimal.
	op := &UserOperation{
		Sender:       common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"),
		Nonce:        "0x1",
		CallGasLimit: "0x186a0",
		CallData:     "0x1234",
	}

	abiOp, err := op.toABI()
	if err != nil {
		t.Fatalf("toABI: %v", err)
	}

	if abiOp.Nonce.Int64() != 1 {
		t.Fatalf("nonce = %d, want 1", abiOp.Nonce.Int64())
	}
	if abiOp.CallGasLimit.Int64() != 100_000 {
		t.Fatalf("callGasLimit = %d, want 100000", abiOp.CallGasLimit.Int64())
	}
	if hex.EncodeToString(abiOp.CallData) != "1234" {
		t.Fatalf("callData = %s, want 1234", hex.EncodeToString(abiOp.CallData))
	}
}

func TestToABIRejectsMalformedHex(t *testing.T) {
	op := &UserOperation{
		Sender:   common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"),
		CallData: "0xzz",
	}

	if _, err := op.toABI(); err == nil {
		t.Fatal("malformed calldata hex was accepted")
	}
}
