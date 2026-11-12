package main

import (
	"math/big"
	"os"
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

// The router map is hand-maintained, and the class of bug it suffered from was
// a transcribed address (one word was rewritten, leaving no contract behind).
// An EIP-55 checksum mismatch catches that offline, without an RPC call.
func TestV2RoutersHaveValidChecksums(t *testing.T) {
	if len(v2Routers) == 0 {
		t.Fatal("no routers configured")
	}
	for chainID, addr := range v2Routers {
		if !common.IsHexAddress(addr) {
			t.Errorf("chain %d: %q is not a valid hex address", chainID, addr)
			continue
		}
		if got := common.HexToAddress(addr); got == (common.Address{}) {
			t.Errorf("chain %d: %q decodes to the zero address", chainID, addr)
			continue
		}
		if canonical := common.HexToAddress(addr).Hex(); canonical != addr {
			t.Errorf("chain %d: %q fails the EIP-55 checksum, canonical form is %s",
				chainID, addr, canonical)
		}
	}
}

// Verifying that each router actually holds code needs a live node, so this is
// opt-in: set MW_VERIFY_ROUTERS=1 to run it. It exists because "address is well
// formed" is not the same as "address is a router" - five of the seven entries
// were well formed and pointed at empty accounts.
func TestV2RoutersHaveCodeOnChain(t *testing.T) {
	if os.Getenv("MW_VERIFY_ROUTERS") == "" {
		t.Skip("set MW_VERIFY_ROUTERS=1 to check router addresses against live chains")
	}
	for chainID, addr := range v2Routers {
		chain := evmChainByChainID(chainID)
		if chain == nil {
			// Testnets are not part of the mainnet registry, so there is no
			// endpoint to check against here.
			t.Logf("chain %d is not a registered mainnet; skipping live check", chainID)
			continue
		}
		if chain.RPCEndpoint == "" {
			t.Errorf("chain %d has no RPC endpoint to verify against", chainID)
			continue
		}
		code, err := ethGetCode(chain.RPCEndpoint, addr)
		if err != nil {
			t.Errorf("chain %d: eth_getCode %s: %v", chainID, addr, err)
			continue
		}
		if len(code) < 4 {
			t.Errorf("chain %d: %s holds no contract; every swap on this chain would fail",
				chainID, addr)
		}
	}
}

// The gas and swap pages let a user pick these networks, and the router map has
// an entry for each. If the chain registry loses one, every call on it fails
// with "unsupported chain_id" while the UI keeps offering it.
func TestSwapUINetworksAreRegistered(t *testing.T) {
	required := map[int64]string{
		1: "Ethereum", 10: "Optimism", 56: "BNB Chain", 137: "Polygon",
		250: "Fantom", 8453: "Base", 42161: "Arbitrum One",
	}
	for chainID, name := range required {
		chain := evmChainByChainID(chainID)
		if chain == nil {
			t.Errorf("%s (chain %d) is offered in the UI but is not registered", name, chainID)
			continue
		}
		if chain.RPCEndpoint == "" {
			t.Errorf("%s (chain %d) has no RPC endpoint", name, chainID)
		}
	}
}

// The merged registry must contain every entry in both static lists; a mismatch
// means one was dropped when the map was built.
func TestSupportedChainsCoversStaticRegistry(t *testing.T) {
	initSupportedChains()
	if got, want := len(SupportedChains), evmMainnetCount+nonEVMMainnetCount; got != want {
		t.Errorf("SupportedChains has %d entries, want %d (evm %d + non-evm %d)",
			got, want, evmMainnetCount, nonEVMMainnetCount)
	}
	for _, c := range evmMainnet {
		if _, ok := SupportedChains[c.ID]; !ok {
			t.Errorf("EVM chain %d (%s) is missing from SupportedChains", c.ID, c.Name)
		}
	}
	for _, c := range nonEVMMainnet {
		if _, ok := SupportedChains[c.ID]; !ok {
			t.Errorf("non-EVM chain %d (%s) is missing from SupportedChains", c.ID, c.Name)
		}
	}
}

// Duplicate ids would silently overwrite each other during the merge.
func TestStaticChainRegistryHasNoDuplicateIDs(t *testing.T) {
	seen := map[int64]string{}
	for _, c := range append(append([]ChainConfig{}, evmMainnet...), nonEVMMainnet...) {
		if prev, dup := seen[c.ID]; dup {
			t.Errorf("chain id %d is defined twice: %q and %q", c.ID, prev, c.Name)
		}
		seen[c.ID] = c.Name
	}
}

// A chain with no RPC endpoint cannot serve a request, so every entry must have
// one. Pi is the single deliberate exception: it has no public RPC, the operator
// sets PI_RPC, and lookupEndpoint returns "" for it on purpose. Pinning the
// exception here means a newly added chain that forgets its endpoint fails.
func TestStaticChainRegistryEntriesHaveRPC(t *testing.T) {
	// Chains whose public endpoint is operator-supplied through an environment
	// variable, so an empty registry value is expected. Pi Network publishes no
	// node at all; Regen, Sommelier, BitShares, NEM, LBRY and Bitcoin Gold have
	// no public REST endpoint that answered a probe, so the client fails closed
	// rather than being pointed at a dead host.
	operatorConfigured := map[int64]bool{
		9000004242: true, // Pi Network    (PI_RPC)
		9000007700: true, // Regen Network (REGEN_API)
		9000007800: true, // Sommelier     (SOMMELIER_API)
		9000008300: true, // BitShares     (BITSHARES_API)
		9000008400: true, // NEM           (NEM_API)
		9000009200: true, // LBRY Credits  (LBRY_API)
		9000009300: true, // Bitcoin Gold  (BTG_API)
		7710:       true, // TigerSmartChain (TSC_RPC_URL) - the platform runs this node
	}
	for _, c := range append(append([]ChainConfig{}, evmMainnet...), nonEVMMainnet...) {
		if c.RPCEndpoint != "" {
			continue
		}
		if !operatorConfigured[c.ID] {
			t.Errorf("chain %d (%s) has no RPC endpoint", c.ID, c.Name)
		}
	}
}
