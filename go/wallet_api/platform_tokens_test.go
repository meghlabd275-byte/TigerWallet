package main

// platform_tokens_test.go — guards the TigerSmartChain integration.
//
// The wallet API and the chain live in separate Go modules, so the RUSD and
// WTGR addresses are declared in both. That duplication is the one thing in
// this integration that could silently drift: if the chain moved a predeploy
// and the wallet kept the old address, every client would show a token
// contract that does not exist. TestPlatformTokensMatchChain reads the chain
// source and fails on any difference.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// chainSourcePath resolves a file inside the TigerSmartChain module. The two
// modules sit at different depths depending on how the test is invoked
// (go test runs with the package directory as the working directory, while an
// IDE may use the module root), so the path is discovered rather than assumed.
func chainSourcePath(rel string) string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "tiger_smart_chain", filepath.FromSlash(rel))
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(wd, "tiger_smart_chain", filepath.FromSlash(rel))
}

func TestPlatformTokensMatchChain(t *testing.T) {
	path := chainSourcePath("chain/predeploy.go")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	src := string(raw)

	// RUSDAddress = common.HexToAddress("0x...")
	addrRe := regexp.MustCompile(`(RUSDAddress|WTGRAddress)\s*=\s*common\.HexToAddress\("(0x[0-9a-fA-F]{40})"\)`)
	found := map[string]string{}
	for _, m := range addrRe.FindAllStringSubmatch(src, -1) {
		found[m[1]] = strings.ToLower(m[2])
	}
	if len(found) != 2 {
		t.Fatalf("expected RUSDAddress and WTGRAddress in %s, found %d", path, len(found))
	}
	if got, want := strings.ToLower(rusdPredeployAddress), found["RUSDAddress"]; got != want {
		t.Fatalf("RUSD address drift: wallet_api has %s, chain has %s", got, want)
	}
	if got, want := strings.ToLower(wtgrPredeployAddress), found["WTGRAddress"]; got != want {
		t.Fatalf("WTGR address drift: wallet_api has %s, chain has %s", got, want)
	}

	// The ERC-20 metadata the chain writes into the predeploy storage must match
	// the names and symbols the wallet advertises, otherwise the token list and
	// the contract disagree about what the asset is called.
	for _, want := range []string{
		`RUSDName = "Royal Tiger USD"`,
		`RUSDSymbol = "RUSD"`,
		`WTGRName = "Wrapped TGR"`,
		`WtgrSymbol = "WTGR"`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("chain metadata %q is not present in %s", want, path)
		}
	}

	// The chain id the wallet registers for TigerSmartChain must be the one the
	// chain itself defaults to.
	chainCfg := chainSourcePath("chain/config.go")
	rawCfg, err := os.ReadFile(chainCfg)
	if err != nil {
		t.Fatalf("read %s: %v", chainCfg, err)
	}
	idRe := regexp.MustCompile(`DefaultChainID\s+int64\s*=\s*(\d+)`)
	m := idRe.FindStringSubmatch(string(rawCfg))
	if m == nil {
		t.Fatalf("DefaultChainID not found in %s", chainCfg)
	}
	if m[1] != "7710" {
		t.Fatalf("chain DefaultChainID is %s, wallet_api registers %d", m[1], tigerSmartChainID)
	}
}

func TestPlatformTokensAreRegisteredInChainAndTokenRegistry(t *testing.T) {
	// The chain must exist in the EVM registry with the TGR native symbol.
	c := chainByID(tigerSmartChainID)
	if c == nil {
		t.Fatalf("chain %d is not in the EVM registry", tigerSmartChainID)
	}
	if c.Symbol != tgrNativeSymbol {
		t.Fatalf("chain %d native symbol = %q, want %q", tigerSmartChainID, c.Symbol, tgrNativeSymbol)
	}
	if c.ChainType != "evm" {
		t.Fatalf("chain %d type = %q, want evm", tigerSmartChainID, c.ChainType)
	}
	if c.Decimals != 18 {
		t.Fatalf("chain %d decimals = %d, want 18", tigerSmartChainID, c.Decimals)
	}

	// RUSD and WTGR must be listed as tokens so the balance endpoint reads them.
	tokens := defaultTokenRegistry[tigerSmartChainID]
	if len(tokens) != 2 {
		t.Fatalf("defaultTokenRegistry[%d] has %d entries, want 2", tigerSmartChainID, len(tokens))
	}
	bySymbol := map[string]TokenInfo{}
	for _, tok := range tokens {
		bySymbol[tok.Symbol] = tok
	}
	for symbol, wantAddr := range map[string]string{
		"RUSD": rusdPredeployAddress,
		"WTGR": wtgrPredeployAddress,
	} {
		tok, ok := bySymbol[symbol]
		if !ok {
			t.Fatalf("%s is not in the token registry for chain %d", symbol, tigerSmartChainID)
		}
		if !strings.EqualFold(tok.Contract, wantAddr) {
			t.Fatalf("%s contract = %s, want %s", symbol, tok.Contract, wantAddr)
		}
		if tok.Decimals != 18 {
			t.Fatalf("%s decimals = %d, want 18", symbol, tok.Decimals)
		}
		if tok.ChainID != tigerSmartChainID {
			t.Fatalf("%s chain id = %d, want %d", symbol, tok.ChainID, tigerSmartChainID)
		}
	}

	// The platform metadata table must agree with the token registry.
	for _, pt := range platformTokensStatic() {
		if !pt.IsNative {
			tok, ok := bySymbol[pt.Symbol]
			if !ok {
				t.Fatalf("platform token %s is missing from the token registry", pt.Symbol)
			}
			if !strings.EqualFold(tok.Contract, pt.Contract) {
				t.Fatalf("platform token %s contract = %s, registry has %s", pt.Symbol, pt.Contract, tok.Contract)
			}
		}
	}
}

// The platform chain must not bundle an endpoint, and the environment override
// must be the only way one appears. Shipping a URL would make every wallet dial
// a host the operator never chose; leaving one out means an unconfigured
// deployment reports the chain as unreachable instead.
func TestPlatformChainBundlesNoEndpoint(t *testing.T) {
	c := chainByID(tigerSmartChainID)
	if c == nil {
		t.Fatalf("chain %d is not registered", tigerSmartChainID)
	}
	// Read the raw registry entry too: chainByID applies the env override, so
	// it would mask a bundled URL.
	for _, raw := range evmMainnet {
		if raw.ID != tigerSmartChainID {
			continue
		}
		if raw.RPCEndpoint != "" {
			t.Fatalf("the TigerSmartChain registry entry bundles an endpoint: %q", raw.RPCEndpoint)
		}
	}

	// With no override the resolved endpoint stays empty. Both the generic
	// and the platform-specific variable are cleared, because either can
	// supply the endpoint and an operator running the platform's own node
	// will have set the latter. Without this the assertion fails in exactly
	// the deployment it describes.
	t.Setenv("CHAIN_7710_RPC", "")
	t.Setenv(tscRPCEnv, "")
	if got := chainByID(tigerSmartChainID); got.RPCEndpoint != "" {
		t.Fatalf("resolved endpoint with no configuration = %q, want empty", got.RPCEndpoint)
	}

	// The platform-specific variable takes precedence when both are set, so
	// the documented TSC_RPC_URL contract is what an operator relies on.
	t.Setenv("CHAIN_7710_RPC", "http://generic.internal:8545")
	t.Setenv(tscRPCEnv, "http://tsc.internal:8545")
	if got := chainByID(tigerSmartChainID); got.RPCEndpoint != "http://tsc.internal:8545" {
		t.Fatalf("resolved endpoint = %q, want %s to take precedence", got.RPCEndpoint, tscRPCEnv)
	}

	// The generic override is honoured when the platform one is unset.
	t.Setenv(tscRPCEnv, "")
	t.Setenv("CHAIN_7710_RPC", "http://tsc.internal:8545")
	if got := chainByID(tigerSmartChainID); got.RPCEndpoint != "http://tsc.internal:8545" {
		t.Fatalf("resolved endpoint = %q, want the configured override", got.RPCEndpoint)
	}
}

// A public chain must still work out of the box: only the platform's own chain
// is allowed to be endpoint-less.
func TestPublicChainsKeepTheirEndpoints(t *testing.T) {
	for _, id := range []int64{1, 56, 137, 8453, 42161} {
		c := chainByID(id)
		if c == nil {
			t.Fatalf("chain %d is not registered", id)
		}
		if c.RPCEndpoint == "" {
			t.Errorf("public chain %d (%s) has no RPC endpoint", id, c.Name)
		}
	}
}

func TestPlatformTokensStaticMetadata(t *testing.T) {
	toks := platformTokensStatic()
	if len(toks) != 3 {
		t.Fatalf("platform token set has %d entries, want 3 (TGR, WTGR, RUSD)", len(toks))
	}

	seen := map[string]PlatformToken{}
	for _, tok := range toks {
		if _, dup := seen[tok.Symbol]; dup {
			t.Fatalf("duplicate platform token %s", tok.Symbol)
		}
		seen[tok.Symbol] = tok
		if tok.ChainID != tigerSmartChainID {
			t.Fatalf("%s chain id = %d, want %d", tok.Symbol, tok.ChainID, tigerSmartChainID)
		}
		if tok.Decimals != 18 {
			t.Fatalf("%s decimals = %d, want 18", tok.Symbol, tok.Decimals)
		}
		if !tok.IsPlatform {
			t.Fatalf("%s is not flagged as a platform asset", tok.Symbol)
		}
	}

	// The native coin has no contract; the ERC-20s must have one.
	if tgr, ok := seen["TGR"]; !ok {
		t.Fatal("TGR is missing")
	} else if !tgr.IsNative || tgr.Contract != "" {
		t.Fatalf("TGR must be native with no contract, got native=%v contract=%q", tgr.IsNative, tgr.Contract)
	}
	for _, symbol := range []string{"WTGR", "RUSD"} {
		tok, ok := seen[symbol]
		if !ok {
			t.Fatalf("%s is missing", symbol)
		}
		if tok.IsNative {
			t.Fatalf("%s must not be flagged native", symbol)
		}
		if !strings.HasPrefix(tok.Contract, "0x") || len(tok.Contract) != 42 {
			t.Fatalf("%s contract %q is not an address", symbol, tok.Contract)
		}
		if tok.Standard != "ERC-20" {
			t.Fatalf("%s standard = %q, want ERC-20", symbol, tok.Standard)
		}
	}

	if !seen["RUSD"].PeggedUSD {
		t.Fatal("RUSD must be flagged as a USD-pegged asset")
	}
	if seen["WTGR"].PeggedUSD {
		t.Fatal("WTGR must not be flagged as a USD-pegged asset")
	}
}

func TestIsPlatformToken(t *testing.T) {
	for _, sym := range []string{"TGR", "tgr", "WTGR", "wtgr", "RUSD", "rusd", " rusd "} {
		if !isPlatformToken(sym) {
			t.Fatalf("isPlatformToken(%q) = false, want true", sym)
		}
	}
	for _, sym := range []string{"", "ETH", "USDT", "USDC", "RYUSD"} {
		if isPlatformToken(sym) {
			t.Fatalf("isPlatformToken(%q) = true, want false", sym)
		}
	}
}

func TestPriceFeedDeclaredPegOnlyForIssuerStablecoins(t *testing.T) {
	// The declared peg exists because the issuer redeems the asset 1:1. A
	// market stablecoin must not be listed: its price has to come from the
	// exchanges, or an outage would silently report USDT at exactly one dollar
	// while it trades elsewhere.
	if _, ok := declaredPegs["USDT"]; ok {
		t.Fatal("USDT must not have a declared peg; its price must come from the market")
	}
	if _, ok := declaredPegs["USDC"]; ok {
		t.Fatal("USDC must not have a declared peg; its price must come from the market")
	}
	if peg, ok := declaredPegs["RUSD"]; !ok || peg != 1.0 {
		t.Fatalf("RUSD peg = %v (present=%v), want 1.0", peg, ok)
	}
}

func TestMedian(t *testing.T) {
	cases := []struct {
		in   []float64
		want float64
	}{
		{nil, 0},
		{[]float64{5}, 5},
		{[]float64{1, 3}, 2},
		{[]float64{1, 2, 3}, 2},
		{[]float64{1, 2, 3, 4}, 2.5},
		{[]float64{100, 101, 102, 103, 1000000}, 102},
	}
	for _, c := range cases {
		if got := median(c.in); got != c.want {
			t.Fatalf("median(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRouterForChainFailsClosedOnBadOverride(t *testing.T) {
	// An operator-supplied router address that is not an address must resolve to
	// the zero address (AMM unavailable), never to a guessed contract.
	t.Setenv("CHAIN_7710_ROUTER", "not-an-address")
	if got := routerForChain(tigerSmartChainID); got != (common.Address{}) {
		t.Fatalf("routerForChain with a malformed override = %s, want the zero address", got.Hex())
	}

	t.Setenv("CHAIN_7710_ROUTER", "0x000000000000000000000000000000000000dEaD")
	if got := routerForChain(tigerSmartChainID); got.Hex() != "0x000000000000000000000000000000000000dEaD" {
		t.Fatalf("routerForChain did not honour a valid override, got %s", got.Hex())
	}

	// With no override and no built-in entry, the chain has no router.
	t.Setenv("CHAIN_7710_ROUTER", "")
	if got := routerForChain(tigerSmartChainID); got != (common.Address{}) {
		t.Fatalf("routerForChain without a router = %s, want the zero address", got.Hex())
	}
}

// TestFetchOnChainPoolPriceDerivesFromLiveReserves drives the price path against
// a stub node that returns the exact payload a Uniswap-V2 pool produces. The
// stub stands in for the chain, not for wallet code: the handler under test
// parses real ABI output and derives the price from it.
func TestFetchOnChainPoolPriceDerivesFromLiveReserves(t *testing.T) {
	word := func(hex string) string {
		if len(hex) >= 64 {
			return hex
		}
		return strings.Repeat("0", 64-len(hex)) + hex
	}

	// 1000 TGR against 2500 RUSD -> 2.50 USD per TGR.
	tgrReserve := "3635c9adc5dea00000"  // 1000e18
	rusdReserve := "878678326eac900000" // 2500e18
	wtgrAddress := strings.ToLower(wtgrPredeployAddress)[2:]
	rusdAddress := strings.ToLower(rusdPredeployAddress)[2:]

	var sawPair string
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var call struct {
			To   string `json:"to"`
			Data string `json:"data"`
		}
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params[0], &call)
		}
		sawPair = call.To

		var result string
		switch call.Data {
		case "0x0902f1ac": // getReserves()
			result = "0x" + word(tgrReserve) + word(rusdReserve) + word("00000000")
		case "0x0dfe1681": // token0()
			result = "0x" + word(wtgrAddress)
		case "0x4f1a4a83": // token1()
			result = "0x" + word(rusdAddress)
		default:
			result = "0x"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
	}))
	defer node.Close()

	t.Setenv("TSC_RPC_URL", node.URL)
	t.Setenv("TSC_TGR_RUSD_PAIR", "0x00000000000000000000000000000000000000aa")

	for _, symbol := range []string{"TGR", "WTGR"} {
		usd, ok := fetchOnChainPoolPrice(context.Background(), symbol)
		if !ok {
			t.Fatalf("%s: no on-chain price was derived", symbol)
		}
		if usd < 2.49 || usd > 2.51 {
			t.Fatalf("%s: price = %v, want 2.50", symbol, usd)
		}
	}
	if !strings.EqualFold(sawPair, "0x00000000000000000000000000000000000000aa") {
		t.Fatalf("the configured pair was not the one queried, got %s", sawPair)
	}

	// RUSD is the dollar leg; it must never be priced from the pool.
	if _, ok := fetchOnChainPoolPrice(context.Background(), "RUSD"); ok {
		t.Fatal("RUSD must not be priced from the pool")
	}
}

// TestFetchOnChainPoolPriceRejectsUnusablePools covers the cases where a pool
// exists but its reserves do not describe a dollar price. Each one must produce
// no price rather than a wrong one.
func TestFetchOnChainPoolPriceRejectsUnusablePools(t *testing.T) {
	word := func(hex string) string {
		if len(hex) >= 64 {
			return hex
		}
		return strings.Repeat("0", 64-len(hex)) + hex
	}
	wtgrAddress := strings.ToLower(wtgrPredeployAddress)[2:]
	rusdAddress := strings.ToLower(rusdPredeployAddress)[2:]

	cases := []struct {
		name    string
		token0  string
		token1  string
		reserve string
	}{
		{"no RUSD leg", word(wtgrAddress), word("00000000000000000000000000000000000000cc"), word("3635c9adc5dea00000")},
		{"both legs RUSD", word(rusdAddress), word(rusdAddress), word("3635c9adc5dea00000")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Params []json.RawMessage `json:"params"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				var call struct {
					Data string `json:"data"`
				}
				if len(req.Params) > 0 {
					_ = json.Unmarshal(req.Params[0], &call)
				}
				var result string
				switch call.Data {
				case "0x0902f1ac":
					result = "0x" + tc.reserve + word("3635c9adc5dea00000") + word("00000000")
				case "0x0dfe1681":
					result = "0x" + tc.token0
				case "0x4f1a4a83":
					result = "0x" + tc.token1
				default:
					result = "0x"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
			}))
			defer node.Close()

			t.Setenv("TSC_RPC_URL", node.URL)
			t.Setenv("TSC_TGR_RUSD_PAIR", "0x00000000000000000000000000000000000000aa")
			if usd, ok := fetchOnChainPoolPrice(context.Background(), "TGR"); ok {
				t.Fatalf("a pool with %s produced a price of %v; want none", tc.name, usd)
			}
		})
	}

	// An empty pool has no price, which is different from a price of zero.
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Params []json.RawMessage `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var call struct {
			Data string `json:"data"`
		}
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params[0], &call)
		}
		var result string
		switch call.Data {
		case "0x0902f1ac":
			result = "0x" + word("0") + word(rusdReserveHex()) + word("00000000")
		case "0x0dfe1681":
			result = "0x" + word(wtgrAddress)
		case "0x4f1a4a83":
			result = "0x" + word(rusdAddress)
		default:
			result = "0x"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
	}))
	defer node.Close()
	t.Setenv("TSC_RPC_URL", node.URL)
	t.Setenv("TSC_TGR_RUSD_PAIR", "0x00000000000000000000000000000000000000aa")
	if usd, ok := fetchOnChainPoolPrice(context.Background(), "TGR"); ok {
		t.Fatalf("an empty pool produced a price of %v; want none", usd)
	}

	// An unconfigured pool address must not produce a price either.
	t.Setenv("TSC_TGR_RUSD_PAIR", "not-an-address")
	if usd, ok := fetchOnChainPoolPrice(context.Background(), "TGR"); ok {
		t.Fatalf("a malformed pair address produced a price of %v; want none", usd)
	}
}

func rusdReserveHex() string { return "878678326eac900000" }
