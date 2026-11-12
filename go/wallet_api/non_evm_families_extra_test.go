package main

// Tests for the non-EVM registry-expansion address schemes. Every vector here
// is published by the chain's own reference implementation; none is generated
// from our own code, so a regression cannot pass by being self-consistent.

import (
	"math/big"
	"strings"
	"testing"

	"golang.org/x/crypto/blake2b"
)

// TestStarkPedersenVectors checks the Pedersen hash against the two vectors
// published in juno core/crypto/pedersen_test.go.
func TestStarkPedersenVectors(t *testing.T) {
	cases := []struct{ a, b, want string }{
		{
			"0x03d937c035c878245caf64531a5756109c53068da139362728feb561405371cb",
			"0x0208a0a10250e382e1e4bbe2880906c2791bf6275695e02fbbc6aeff9cd8b31a",
			"0x030e480bed5fe53fa909cc0f8c4d99b8f9f2c016be4c41e13a4848797979c662",
		},
		{
			"0x58f580910a6ca59b28927c08fe6c43e2e303ca384badc365795fc645d479d45",
			"0x78734f65a067be9bdb39de18434d71e79f7b6466a4b66bbd979ab9e7515fe0b",
			"0x68cc0b76cddd1dd4ed2301ada9b7c872b23875d5ff837b3a87993e0d9996b87",
		},
	}
	for i, c := range cases {
		a, ok := new(big.Int).SetString(c.a[2:], 16)
		if !ok {
			t.Fatalf("case %d: bad input a", i)
		}
		b, ok := new(big.Int).SetString(c.b[2:], 16)
		if !ok {
			t.Fatalf("case %d: bad input b", i)
		}
		got := starknetPedersen(a, b)
		want, _ := new(big.Int).SetString(c.want[2:], 16)
		if got.Cmp(want) != 0 {
			t.Fatalf("case %d: pedersen mismatch\n got 0x%s\nwant %s", i, got.Text(16), c.want)
		}
	}
}

// TestStarkPedersenConstantsOnCurve verifies every published Pedersen constant
// really satisfies y^2 = x^3 + x + b over the STARK field. A mistyped constant
// would otherwise silently produce wrong addresses.
func TestStarkPedersenConstantsOnCurve(t *testing.T) {
	check := func(name string, p *starkPoint) {
		lhs := new(big.Int).Mul(p.y, p.y)
		rhs := new(big.Int).Mul(p.x, new(big.Int).Mul(p.x, p.x))
		rhs.Add(rhs, p.x)
		rhs.Add(rhs, mustParseBig("3141592653589793238462643383279502884197169399375105820974944592307816406665"))
		lhs.Mod(lhs, starkPrime)
		rhs.Mod(rhs, starkPrime)
		if lhs.Cmp(rhs) != 0 {
			t.Fatalf("%s is not on the STARK curve", name)
		}
	}
	check("shift", starkPedersenShift)
	for i, p := range starkPedersenBase {
		check("base"+string(rune('0'+i)), p)
	}
	check("generator", starkGenerator)
}

// TestStarknetPrecomputeAddressVector checks the contract-address hash against
// the vector published in starknet.go contracts/contracts_test.go
// (sepolia tx 0x3789fe05...).
func TestStarknetPrecomputeAddressVector(t *testing.T) {
	hexToInt := func(s string) *big.Int {
		v, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
		if !ok {
			t.Fatalf("bad hex %q", s)
		}
		return v
	}
	addr := StarknetContractAddress(
		big.NewInt(0),
		hexToInt("0x0702e82f1ec15656ad4502268dad530197141f3b59f5529835af9318ef399da5"),
		hexToInt("0x064728e0c0713811c751930f8d3292d683c23f107c89b0a101425d9e80adb1c0"),
		[]*big.Int{hexToInt("0x022f3e55b61d86c2ac5239fa3b3b8761f26b9a5c0b5f61ddbd5d756ced498b46")},
	)
	want := "31463b5263a6631be4d1fe92d64d13e3a8498c440bf789e69ccb951eb8ad5da"
	if addr.Text(16) != want {
		t.Fatalf("starknet contract address mismatch\n got %s\nwant %s", addr.Text(16), want)
	}
}

// TestStarknetAccountAddressFormat checks the account address is the padded
// 0x form and that a different class hash yields a different account.
func TestStarknetAccountAddressFormat(t *testing.T) {
	pub := new(big.Int).SetBytes([]byte{0x01, 0x02, 0x03})
	a1 := StarknetAccountAddress(pub, big.NewInt(0x111))
	a2 := StarknetAccountAddress(pub, big.NewInt(0x222))
	if !strings.HasPrefix(a1, "0x") || len(a1) != 66 {
		t.Fatalf("account address must be 0x + 64 hex chars, got %q", a1)
	}
	if a1 == a2 {
		t.Fatal("different account class hashes must produce different addresses")
	}
	if StarknetAccountAddress(pub, big.NewInt(0x111)) != a1 {
		t.Fatal("starknet account derivation must be deterministic")
	}
}

// TestStacksBurnAddressVector checks c32check against the canonical Stacks
// burn address, whose hash160 is twenty zero bytes with version 22.
func TestStacksBurnAddressVector(t *testing.T) {
	hash160 := make([]byte, 20)
	versioned := append([]byte{22}, hash160...)
	chk := doubleSHA256(versioned)[:4]
	got := "S" + string(stacksC32Alphabet[22]) + stacksC32Encode(append(append([]byte{}, hash160...), chk...))
	const want = "SP000000000000000000002Q6VF78"
	if got != want {
		t.Fatalf("stacks burn address mismatch\n got %s\nwant %s", got, want)
	}
}

// TestStacksAddressFormat checks a real derivation produces a mainnet
// single-signature address.
func TestStacksAddressFormat(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	addr, err := StacksAddress(seed, "m/44'/5757'/0'/0/0")
	if err != nil {
		t.Fatalf("stacks address: %v", err)
	}
	if !strings.HasPrefix(addr, "SP") {
		t.Fatalf("stacks mainnet p2pkh address must start with SP, got %q", addr)
	}
	if addr != mustStacks(t, seed) {
		t.Fatal("stacks derivation must be deterministic")
	}
}

func mustStacks(t *testing.T, seed []byte) string {
	t.Helper()
	a, err := StacksAddress(seed, "m/44'/5757'/0'/0/0")
	if err != nil {
		t.Fatalf("stacks address: %v", err)
	}
	return a
}

// TestMoneroBase58Blocks pins the block widths from Monero's own
// base58 block-size table.
func TestMoneroBase58Blocks(t *testing.T) {
	if got := moneroBase58Encode(make([]byte, 8)); got != "11111111111" {
		t.Fatalf("8 zero bytes must encode to 11 ones, got %q", got)
	}
	if got := moneroBase58Encode([]byte{0x00}); got != "11" {
		t.Fatalf("1 zero byte must encode to 2 ones, got %q", got)
	}
	// A Monero address payload is 69 bytes and encodes to exactly 95 chars.
	if got := len(moneroBase58Encode(make([]byte, 69))); got != 95 {
		t.Fatalf("69-byte payload must encode to 95 characters, got %d", got)
	}
}

// TestMoneroAddressLengthAndChecksum checks the address is the canonical
// 95-character form and that the Keccak-256 checksum is present.
func TestMoneroAddressLengthAndChecksum(t *testing.T) {
	spend := make([]byte, 32)
	view := make([]byte, 32)
	for i := range spend {
		spend[i] = byte(i + 1)
		view[i] = byte(0xff - i)
	}
	addr, err := MoneroAddressFromKeys(spend, view, 18)
	if err != nil {
		t.Fatalf("monero address: %v", err)
	}
	if len(addr) != 95 {
		t.Fatalf("monero address must be 95 characters, got %d (%q)", len(addr), addr)
	}
	if !strings.HasPrefix(addr, "4") {
		t.Fatalf("monero mainnet standard address must start with 4, got %q", addr)
	}
	if _, err := MoneroAddressFromKeys(spend[:31], view, 18); err == nil {
		t.Fatal("a short spend key must fail closed")
	}
}

// TestBase58CheckBitcoinVectors pins base58checkEncode to the well-known
// Bitcoin burn address so the NEO wrapper rests on a verified primitive.
func TestBase58CheckBitcoinVectors(t *testing.T) {
	burn := base58checkEncode(append([]byte{0x00}, make([]byte, 20)...))
	if burn != "1111111111111111111114oLvT2" {
		t.Fatalf("bitcoin burn address mismatch: %s", burn)
	}
}

// TestNEOAddressFormat checks the NEO N3 address structure: Base58Check with
// version byte 0x35, so it always starts with 'N'.
func TestNEOAddressFormat(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	addr, err := NEOAddress(seed, "m/44'/888'/0'/0/0")
	if err != nil {
		t.Fatalf("neo address: %v", err)
	}
	if !strings.HasPrefix(addr, "N") {
		t.Fatalf("neo n3 address must start with N, got %q", addr)
	}
	raw, err := base58Decode(addr)
	if err != nil {
		t.Fatalf("neo address base58: %v", err)
	}
	if len(raw) != 25 || raw[0] != 0x35 {
		t.Fatalf("neo address must be 25 bytes with version 0x35, got len=%d version=0x%02x", len(raw), raw[0])
	}
	again, _ := NEOAddress(seed, "m/44'/888'/0'/0/0")
	if again != addr {
		t.Fatal("neo derivation must be deterministic")
	}
}

// TestMinaAndErgoAndWaktFormats checks the remaining schemes produce their
// canonical shapes.
func TestMinaAndErgoAndWaktFormats(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)

	// Mina addresses are Base58Check version 203 over a 35-byte Pallas public
	// key. The previous expectation here (0x16 0x01) was the test asserting the
	// same wrong constant as the implementation, so it could never catch the
	// defect; the layout is now pinned by TestMinaAddressFormatAndVersionByte.
	mina, err := MinaAddress(seed, "m/44'/12586'/0'/0'")
	if err != nil {
		t.Fatalf("mina address: %v", err)
	}
	if !strings.HasPrefix(mina, "B62q") {
		t.Fatalf("mina mainnet address must start with B62q, got %q", mina)
	}
	raw, err := base58Decode(mina)
	if err != nil {
		t.Fatalf("mina base58: %v", err)
	}
	if len(raw) != 1+35+4 || raw[0] != 203 {
		t.Fatalf("mina address must be version 203 + 35-byte key + checksum, got len=%d version=%d", len(raw), raw[0])
	}

	ergo, err := ErgoP2PKAddress(seed, "m/44'/429'/0'/0/0")
	if err != nil {
		t.Fatalf("ergo address: %v", err)
	}
	ergoRaw, err := base58Decode(ergo)
	if err != nil {
		t.Fatalf("ergo base58: %v", err)
	}
	if len(ergoRaw) != 1+32+4 || ergoRaw[0] != 0x01 {
		t.Fatalf("ergo P2PK address must be type 0x01 + 32-byte hash + checksum, got len=%d type=0x%02x", len(ergoRaw), ergoRaw[0])
	}

	wakt, err := blake2bNanoAddress(seed, "m/44'/1017'/0'", "wakt_")
	if err != nil {
		t.Fatalf("wakt address: %v", err)
	}
	if !strings.HasPrefix(wakt, "wakt_") || len(wakt) != len("wakt_")+52+8 {
		t.Fatalf("wakt address must be prefix + 60 base32 chars, got %q (%d chars)", wakt, len(wakt))
	}
}

// TestArweaveAddressFromModulus checks the Arweave address is Base64URL of the
// SHA-256 of a 512-byte modulus, and that a short modulus fails closed.
func TestArweaveAddressFromModulus(t *testing.T) {
	addr, err := ArweaveAddressFromModulus(make([]byte, 512))
	if err != nil {
		t.Fatalf("arweave address: %v", err)
	}
	if len(addr) != 43 {
		t.Fatalf("arweave address must be 43 unpadded base64url chars, got %d", len(addr))
	}
	if strings.ContainsAny(addr, "+/=") {
		t.Fatalf("arweave address must be unpadded base64url, got %q", addr)
	}
	if _, err := ArweaveAddressFromModulus(make([]byte, 256)); err == nil {
		t.Fatal("a 256-byte modulus must fail closed (RSA-4096 only)")
	}
}

// TestNanoBurnAddressVector pins the Nano ledger base32 and checksum against
// the canonical burn address, whose account is the 32-byte zero key. This
// catches both a dropped final base32 group (which shortens every address by
// four characters) and a reversed checksum.
func TestNanoBurnAddressVector(t *testing.T) {
	zero := make([]byte, 32)
	h, err := blake2b.New(5, nil)
	if err != nil {
		t.Fatalf("blake2b: %v", err)
	}
	h.Write(zero)
	d := h.Sum(nil)
	rev := make([]byte, 5)
	for i := 0; i < 5; i++ {
		rev[i] = d[4-i]
	}
	got := "nano_" + nanoBase32Enc(zero) + nanoBase32Enc(rev)
	const want = "nano_1111111111111111111111111111111111111111111111111111hifc8npp"
	if got != want {
		t.Fatalf("nano burn address mismatch\n got %s\nwant %s", got, want)
	}
	if enc := nanoBase32Enc(zero); len(enc) != 52 {
		t.Fatalf("32 bytes must encode to 52 base32 characters, got %d", len(enc))
	}
}

// TestNonEvmExtraFailClosed ensures the chains without a faithful scheme
// cannot silently return something plausible.
func TestNonEvmExtraFailClosed(t *testing.T) {
	for _, ct := range []string{"iota", "kadena"} {
		if err := nonEvmExtraNotFeasible("", ct); err == nil {
			t.Fatalf("%s must fail closed", ct)
		}
	}
}
