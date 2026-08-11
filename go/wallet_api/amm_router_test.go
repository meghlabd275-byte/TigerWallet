package main

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Test that the getAmountsOut calldata selector + encoding is byte-exact
// (selector 0xd06ca61f, then amountIn, dynamic offset 0x40, length, addresses).
func TestGetAmountsOutData(t *testing.T) {
	addrA := common.HexToAddress("0x" + strings.Repeat("a", 40))
	addrB := common.HexToAddress("0x" + strings.Repeat("b", 40))
	amount := big.NewInt(1_000_000)
	data := getAmountsOutData(amount, []common.Address{addrA, addrB})

	if len(data) != 4+32+32+32+32*2 {
		t.Fatalf("unexpected length %d", len(data))
	}
	// selector
	wantSel := []byte{0xd0, 0x6c, 0xa6, 0x1f}
	for i, b := range wantSel {
		if data[i] != b {
			t.Fatalf("selector byte %d: got %x want %x", i, data[i], b)
		}
	}
	// amountIn at offset 4..36
	gotAmount := new(big.Int).SetBytes(data[4:36])
	if gotAmount.Cmp(amount) != 0 {
		t.Fatalf("amountIn: got %s want %s", gotAmount.String(), amount.String())
	}
	// dynamic offset at 36..68 must be 0x40 (64)
	if off := new(big.Int).SetBytes(data[36:68]).Int64(); off != 64 {
		t.Fatalf("dynamic offset: got %d want 64", off)
	}
	// path length at 68..100 must be 2
	if n := new(big.Int).SetBytes(data[68:100]).Int64(); n != 2 {
		t.Fatalf("path length: got %d want 2", n)
	}
	// first address at 100..132
	if got := common.BytesToAddress(data[100:132]); got != addrA {
		t.Fatalf("addr[0]: got %s want %s", got.Hex(), addrA.Hex())
	}
	// second address at 132..164
	if got := common.BytesToAddress(data[132:164]); got != addrB {
		t.Fatalf("addr[1]: got %s want %s", got.Hex(), addrB.Hex())
	}
}

func TestSwapExactTokensForTokensData(t *testing.T) {
	addrA := common.HexToAddress("0x" + strings.Repeat("a", 40))
	addrB := common.HexToAddress("0x" + strings.Repeat("b", 40))
	recipient := common.HexToAddress("0x" + strings.Repeat("c", 40))
	amountIn := big.NewInt(1_500_000)
	amountOutMin := big.NewInt(1_400_000)
	deadline := big.NewInt(1_700_000_000)
	data := swapExactTokensForTokensData(amountIn, amountOutMin, []common.Address{addrA, addrB}, recipient, deadline)

	// selector 0x18cbafe5
	wantSel := []byte{0x18, 0xcb, 0xaf, 0xe5}
	for i, b := range wantSel {
		if data[i] != b {
			t.Fatalf("selector byte %d: got %x want %x", i, data[i], b)
		}
	}
	// amountIn at 4..36
	if got := new(big.Int).SetBytes(data[4:36]); got.Cmp(amountIn) != 0 {
		t.Fatalf("amountIn: got %s", got.String())
	}
	// amountOutMin at 36..68
	if got := new(big.Int).SetBytes(data[36:68]); got.Cmp(amountOutMin) != 0 {
		t.Fatalf("amountOutMin: got %s", got.String())
	}
	// path offset at 68..100 must be 0xa0 (160)
	if off := new(big.Int).SetBytes(data[68:100]).Int64(); off != 160 {
		t.Fatalf("path offset: got %d want 160", off)
	}
	// recipient at 100..132
	if got := common.BytesToAddress(data[100:132]); got != recipient {
		t.Fatalf("recipient: got %s", got.Hex())
	}
	// deadline at 132..164
	if got := new(big.Int).SetBytes(data[132:164]); got.Cmp(deadline) != 0 {
		t.Fatalf("deadline: got %s", got.String())
	}
	// path length at 164..196 must be 2
	if n := new(big.Int).SetBytes(data[164:196]).Int64(); n != 2 {
		t.Fatalf("path length: got %d want 2", n)
	}
}

func TestDecodeAmountsOut(t *testing.T) {
	// Construct a fake (uint256[]) return: offset(32) + length(2) + a + b
	ret := make([]byte, 32+32+32+32)
	// offset (ignored by our parser, but present)
	copy(ret[0:32], make([]byte, 32))
	// length = 2
	ret[63] = 2
	// first amount = 123
	a := big.NewInt(123)
	copy(ret[64:96], common.LeftPadBytes(a.Bytes(), 32))
	// second amount = 456
	b := big.NewInt(456)
	copy(ret[96:128], common.LeftPadBytes(b.Bytes(), 32))

	amounts, err := decodeAmountsOut(ret)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(amounts) != 2 {
		t.Fatalf("got %d amounts, want 2", len(amounts))
	}
	if amounts[0].Cmp(a) != 0 || amounts[1].Cmp(b) != 0 {
		t.Fatalf("amounts: got %s,%s want %s,%s", amounts[0], amounts[1], a, b)
	}
}

func TestDecodeAmountsOutRejectsShort(t *testing.T) {
	if _, err := decodeAmountsOut(make([]byte, 10)); err == nil {
		t.Fatal("expected error for short return")
	}
}

func TestRouterForChainKnown(t *testing.T) {
	// Ethereum mainnet + BSC + Polygon must resolve to a non-zero router.
	for _, id := range []int64{1, 56, 137, 42161, 10, 8453} {
		if r := routerForChain(id); r == (common.Address{}) {
			t.Errorf("chain %d: expected a router", id)
		}
	}
}

func TestRouterForChainUnknown(t *testing.T) {
	if r := routerForChain(999999); r != (common.Address{}) {
		t.Errorf("unknown chain should have zero router, got %s", r.Hex())
	}
}

func TestHumanToWei(t *testing.T) {
	amt, _ := new(big.Float).SetString("1.5")
	wei, err := humanToWei(amt, 18)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	want.Mul(want, big.NewInt(15))
	want.Div(want, big.NewInt(10))
	if wei.Cmp(want) != 0 {
		t.Fatalf("got %s want %s", wei.String(), want.String())
	}
}

func TestHumanToWeiRejectsZeroOrNegative(t *testing.T) {
	zero, _ := new(big.Float).SetString("0")
	if _, err := humanToWei(zero, 6); err == nil {
		t.Fatal("expected error for zero")
	}
	neg, _ := new(big.Float).SetString("-1")
	if _, err := humanToWei(neg, 6); err == nil {
		t.Fatal("expected error for negative")
	}
}
