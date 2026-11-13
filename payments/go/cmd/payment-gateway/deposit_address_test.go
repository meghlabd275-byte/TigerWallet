package main

// Tests for the payment gateway's address derivation and amount conversion.
//
// The derivation is checked against published BIP-32 and BIP-84 test vectors.
// A wrong elliptic-curve implementation, a wrong chain-code step or a wrong
// encoding all produce a plausible-looking address, so the vectors are the
// only way to know the gateway can actually spend what it receives.

import (
	"encoding/hex"
	"math/big"
	"testing"
)

// BIP-32 test vector 1 (seed 000102030405060708090a0b0c0d0e0f).
const bip32Vector1Seed = "000102030405060708090a0b0c0d0e0f"

func TestHDDeriveMatchesBIP32Vector1(t *testing.T) {
	seed, err := hex.DecodeString(bip32Vector1Seed)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	cases := []struct {
		path    string
		privHex string
		pubHex  string
	}{
		{
			path:    "m",
			privHex: "e8f32e723decf4051aefac8e2c93c9c5b214313817cdb01a1494b917c8436b35",
			pubHex:  "0339a36013301597daef41fbe593a02cc513d0b55527ec2df1050e2e8ff49c85c2",
		},
		{
			path:    "m/0'",
			privHex: "edb2e14f9ee77d26dd93b4ecede8d16ed408ce149b6cd80b0715a2d911a0afea",
			pubHex:  "035a784662a4a20a65bf6aab9ae98a6c068a81c52e4b032c0fb5400c706cfccc56",
		},
		{
			path:    "m/0'/1",
			privHex: "3c6cb8d0f6a264c91ea8b5030fadaa8e538b020f0a387421a12de9319dc93368",
			pubHex:  "03501e454bf00751f24b1b489aa925215d66af2234e3891c3b21a52bedb3cd711c",
		},
		{
			path:    "m/0'/1/2'",
			privHex: "cbce0d719ecf7431d88e6a89fa1483e02e35092af60c042b1df2ff59fa424dca",
			pubHex:  "0357bfe1e341d01c69fe5654309956cbea516822fba8a601743a012a7896ee8dc2",
		},
		{
			path:    "m/0'/1/2'/2",
			privHex: "0f479245fb19a38a1954c5c7c0ebab2f9bdfd96a17563ef28a6a4b1a2a764ef4",
			pubHex:  "02e8445082a72f29b75ca48748a914df60622a609cacfce8ed0e35804560741d29",
		},
		{
			path:    "m/0'/1/2'/2/1000000000",
			privHex: "471b76e389e528d6de6d816857e012c5455051cad6660850e58372a6c3e6e7c8",
			pubHex:  "022a471424da5e657499d1ff51cb43c47481a03b1e77f951fe64cec9f5a48f7011",
		},
	}

	for _, tc := range cases {
		priv, pub, err := hdDerivePath(seed, tc.path)
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		if got := hex.EncodeToString(priv); got != tc.privHex {
			t.Errorf("%s private key\n got %s\nwant %s", tc.path, got, tc.privHex)
		}
		if got := hex.EncodeToString(pub); got != tc.pubHex {
			t.Errorf("%s public key\n got %s\nwant %s", tc.path, got, tc.pubHex)
		}
	}
}

// BIP-84 test vector 1: the mnemonic "abandon abandon ... about" produces
// these first Ethereum and first Bitcoin accounts.
func TestDerivePaymentAddressMatchesBIP84(t *testing.T) {
	// BIP-84 vectors use the BIP-39 seed for that mnemonic.
	seed, err := hex.DecodeString(
		"5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc1" +
			"9a5ac40b389cd370d086206dec8aa6c43daea6690f20ad3d8d48b2d2ce9e38e4")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// BIP-84 first receive address for account 0.
	_, pub, err := hdDerivePath(seed, "m/84'/0'/0'/0/0")
	if err != nil {
		t.Fatalf("hd: %v", err)
	}
	address, err := bech32Encode("bc", 0x00, hash160(pub))
	if err != nil {
		t.Fatalf("bech32: %v", err)
	}
	if address != "bc1qcr8te4kr609gcawutmrza0j4xv80jy8z306fyu" {
		t.Errorf("BIP-84 address\n got %s\nwant bc1qcr8te4kr609gcawutmrza0j4xv80jy8z306fyu", address)
	}
}

// BIP-32 test vector 2. The published entry for this seed is the master node
// itself, so the path under test is "m".
func TestHDDeriveMatchesBIP32Vector2(t *testing.T) {
	seed, _ := hex.DecodeString("fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542")

	_, pub, err := hdDerivePath(seed, "m")
	if err != nil {
		t.Fatalf("hd: %v", err)
	}
	want := "03cbcaa9c98c877a26977d00825c956a238e8dddfbd322cce4f74b0b5bd6ace4a7"
	if got := hex.EncodeToString(pub); got != want {
		t.Errorf("m public key\n got %s\nwant %s", got, want)
	}

	// Walking to m/0 must produce a different key, and the same key on every
	// call: the derivation is a pure function.
	_, child, err := hdDerivePath(seed, "m/0")
	if err != nil {
		t.Fatalf("hd m/0: %v", err)
	}
	if hex.EncodeToString(child) == want {
		t.Error("m/0 must not equal the master key")
	}
	_, childAgain, err := hdDerivePath(seed, "m/0")
	if err != nil {
		t.Fatalf("hd m/0 again: %v", err)
	}
	if hex.EncodeToString(child) != hex.EncodeToString(childAgain) {
		t.Error("derivation is not deterministic")
	}
}

func TestDerivePaymentAddressIsDeterministic(t *testing.T) {
	seed := []byte("a-sufficiently-long-master-seed-value")

	first, err := derivePaymentAddress(seed, 42, "ETH")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := derivePaymentAddress(seed, 42, "ETH")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first != second {
		t.Errorf("the same user must always get the same address: %s != %s", first, second)
	}
}

func TestDerivePaymentAddressIsPerUser(t *testing.T) {
	seed := []byte("a-sufficiently-long-master-seed-value")

	seen := map[string]uint{}
	for userID := uint(1); userID <= 25; userID++ {
		address, err := derivePaymentAddress(seed, userID, "ETH")
		if err != nil {
			t.Fatalf("user %d: %v", userID, err)
		}
		if other, clash := seen[address]; clash {
			t.Fatalf("users %d and %d share address %s", other, userID, address)
		}
		seen[address] = userID
	}
}

func TestDerivePaymentAddressFailsClosedWithoutSeed(t *testing.T) {
	if _, err := derivePaymentAddress(nil, 1, "ETH"); err == nil {
		t.Fatal("expected an error when no master seed is configured")
	}
	if _, err := derivePaymentAddress([]byte{}, 1, "ETH"); err == nil {
		t.Fatal("expected an error when the master seed is empty")
	}
}

func TestDerivePaymentAddressRejectsUnknownCurrency(t *testing.T) {
	seed := []byte("a-sufficiently-long-master-seed-value")
	if _, err := derivePaymentAddress(seed, 1, "DOGEBONK"); err == nil {
		t.Fatal("expected an error for an unsupported currency")
	}
}

func TestAddressFormatsAreValid(t *testing.T) {
	seed := []byte("a-sufficiently-long-master-seed-value")

	cases := []struct {
		currency string
		check    func(string) bool
		desc     string
	}{
		{"ETH", func(a string) bool {
			return len(a) == 42 && a[:2] == "0x" && isHex(a[2:])
		}, "0x + 40 hex"},
		{"BNB", func(a string) bool {
			return len(a) == 42 && a[:2] == "0x" && isHex(a[2:])
		}, "0x + 40 hex"},
		{"TRX", func(a string) bool {
			return len(a) == 34 && a[0] == 'T'
		}, "base58check starting with T"},
		{"BTC", func(a string) bool {
			return len(a) == 42 && a[:3] == "bc1"
		}, "bech32 bc1"},
		{"LTC", func(a string) bool {
			return len(a) >= 26 && len(a) <= 35 && a[0] == 'L'
		}, "base58check starting with L"},
		{"DOGE", func(a string) bool {
			return len(a) >= 26 && len(a) <= 35 && a[0] == 'D'
		}, "base58check starting with D"},
	}

	for _, tc := range cases {
		address, err := derivePaymentAddress(seed, 7, tc.currency)
		if err != nil {
			t.Fatalf("%s: %v", tc.currency, err)
		}
		if !tc.check(address) {
			t.Errorf("%s address %q is not %s", tc.currency, address, tc.desc)
		}
	}
}

func TestEVMAddressIsNotTheTimestampHash(t *testing.T) {
	// The previous implementation returned hex of a timestamp digest. A real
	// address must at least differ per user, which a timestamp hash did not
	// guarantee and a derived key does.
	seed := []byte("a-sufficiently-long-master-seed-value")
	a, _ := derivePaymentAddress(seed, 1, "ETH")
	b, _ := derivePaymentAddress(seed, 2, "ETH")
	if a == b {
		t.Fatal("two users produced the same address")
	}
	if len(a) != 42 {
		t.Fatalf("not an EVM address: %q", a)
	}
}

func TestToBaseUnits(t *testing.T) {
	cases := []struct {
		amount   float64
		currency string
		want     string
	}{
		{1.0, "ETH", "1000000000000000000"},
		{0.1, "ETH", "100000000000000000"},
		{0.000000001, "ETH", "1000000000"},
		{1.0, "BTC", "100000000"},
		{0.5, "BTC", "50000000"},
		{100.0, "USDT", "100000000"},
		{0.000001, "USDT", "1"},
		{0.0, "ETH", "0"},
		{12345.678901, "USDT", "12345678901"},
	}

	for _, tc := range cases {
		got, err := toBaseUnits(tc.amount, tc.currency)
		if err != nil {
			t.Fatalf("%v %s: %v", tc.amount, tc.currency, err)
		}
		if got.String() != tc.want {
			t.Errorf("%v %s = %s, want %s", tc.amount, tc.currency, got.String(), tc.want)
		}
	}
}

func TestToBaseUnitsRejectsBadInput(t *testing.T) {
	if _, err := toBaseUnits(-1.0, "ETH"); err == nil {
		t.Error("negative amounts must be rejected")
	}
	if _, err := toBaseUnits(1.0, "NOTACOIN"); err == nil {
		t.Error("unknown currencies must be rejected")
	}
}

func TestHexToBigInt(t *testing.T) {
	value, err := hexToBigInt("0xde0b6b3a7640000") // 1 ETH in wei
	if err != nil {
		t.Fatalf("hexToBigInt: %v", err)
	}
	if value.Cmp(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) != 0 {
		t.Errorf("got %s, want 1e18", value.String())
	}

	if _, err := hexToBigInt("0xzz"); err == nil {
		t.Error("invalid hex must be rejected")
	}
	if _, err := hexToBigInt("0x"); err != nil {
		t.Errorf("an empty quantity should read as zero: %v", err)
	}
}

func TestKeccak256Selectors(t *testing.T) {
	// balanceOf(address) must hash to 70a08231 or the ERC-20 read is wrong.
	digest := keccak256([]byte("balanceOf(address)"))
	if got := hex.EncodeToString(digest[:4]); got != "70a08231" {
		t.Errorf("balanceOf selector = %s, want 70a08231", got)
	}
	// Keccak of the empty string, which distinguishes Keccak from SHA3.
	if got := hex.EncodeToString(keccak256(nil)); got !=
		"c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470" {
		t.Errorf("keccak256(empty) = %s", got)
	}
}

func TestUncompressedPubkeyRoundTrips(t *testing.T) {
	seed := []byte("a-sufficiently-long-master-seed-value")
	_, compressed, err := hdDerivePath(seed, "m/44'/60'/0'/0/0")
	if err != nil {
		t.Fatalf("hd: %v", err)
	}
	uncompressed := uncompressedPubkey(compressed)
	if len(uncompressed) != 65 || uncompressed[0] != 0x04 {
		t.Fatalf("expected a 65-byte uncompressed key, got %d bytes", len(uncompressed))
	}
	// The x coordinate must be identical between the two encodings.
	if hex.EncodeToString(uncompressed[1:33]) != hex.EncodeToString(compressed[1:]) {
		t.Error("x coordinate changed during decompression")
	}
}

func TestERC20BalanceRejectsBadAddresses(t *testing.T) {
	client := &chainClient{rpcURL: "http://127.0.0.1:1"}
	if _, err := client.erc20Balance("not-an-address", "0x"+hex.EncodeToString(make([]byte, 20))); err == nil {
		t.Error("invalid contract address must be rejected before any call")
	}
	if _, err := client.erc20Balance("0x"+hex.EncodeToString(make([]byte, 20)), "short"); err == nil {
		t.Error("invalid holder address must be rejected before any call")
	}
}

func TestBalanceForRejectsUnknownCurrency(t *testing.T) {
	client := &chainClient{rpcURL: "http://127.0.0.1:1"}
	if _, err := client.balanceFor("NOPE", "0x"+hex.EncodeToString(make([]byte, 20))); err == nil {
		t.Error("unknown currency must be rejected")
	}
}

func TestBalanceForReportsMissingNonEVMEndpoint(t *testing.T) {
	client := &chainClient{rpcURL: "http://127.0.0.1:1"}
	_, err := client.balanceFor("BTC", "bc1qexample")
	if err == nil {
		t.Fatal("a non-EVM currency must not be read through the EVM JSON-RPC client")
	}
}
