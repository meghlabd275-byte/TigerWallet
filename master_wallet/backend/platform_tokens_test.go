package main

// platform_tokens_test.go — guards the TigerSmartChain integration in the
// MasterWallet backend.
//
// The wallet API and the chain live in separate Go modules, so the RUSD and
// WTGR addresses are declared in both. That duplication is the one thing that
// could silently drift: if the chain moved a predeploy and MasterWallet kept
// the old address, every treasury and user-wallet view would show a token
// contract that does not exist. TestPlatformTokensMatchChain reads the chain
// source and fails on any difference.

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// chainSourcePath resolves a file inside the TigerSmartChain module. The two
// modules sit at different depths depending on how the test is invoked, so the
// path is discovered rather than assumed.
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

	addrRe := regexp.MustCompile(`(RUSDAddress|WTGRAddress)\s*=\s*common\.HexToAddress\("(0x[0-9a-fA-F]{40})"\)`)
	found := map[string]string{}
	for _, m := range addrRe.FindAllStringSubmatch(src, -1) {
		found[m[1]] = strings.ToLower(m[2])
	}
	if len(found) != 2 {
		t.Fatalf("expected RUSDAddress and WTGRAddress in %s, found %d", path, len(found))
	}
	if got, want := strings.ToLower(rusdPredeployAddress), found["RUSDAddress"]; got != want {
		t.Fatalf("RUSD address drift: master_wallet has %s, chain has %s", got, want)
	}
	if got, want := strings.ToLower(wtgrPredeployAddress), found["WTGRAddress"]; got != want {
		t.Fatalf("WTGR address drift: master_wallet has %s, chain has %s", got, want)
	}

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

	cfgPath := chainSourcePath("chain/config.go")
	rawCfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read %s: %v", cfgPath, err)
	}
	m := regexp.MustCompile(`DefaultChainID\s+int64\s*=\s*(\d+)`).FindStringSubmatch(string(rawCfg))
	if m == nil {
		t.Fatalf("DefaultChainID not found in %s", cfgPath)
	}
	if m[1] != "7710" {
		t.Fatalf("chain DefaultChainID is %s, master_wallet registers %d", m[1], tigerSmartChainID)
	}
}

func TestPlatformChainIsRegistered(t *testing.T) {
	c, ok := chainByID(tigerSmartChainID)
	if !ok {
		t.Fatalf("chain %d is not in supportedChains", tigerSmartChainID)
	}
	if c.Symbol != tgrNativeSymbol {
		t.Fatalf("native symbol = %q, want %q", c.Symbol, tgrNativeSymbol)
	}
	if !c.IsEVM {
		t.Fatal("TigerSmartChain must be registered as an EVM chain")
	}
	if c.RPCEnv != tscRPCEnv {
		t.Fatalf("RPC env = %q, want %q", c.RPCEnv, tscRPCEnv)
	}
	if c.Decimals != 18 {
		t.Fatalf("decimals = %d, want 18", c.Decimals)
	}
	if c.DerivationPath != "m/44'/60'/0'/0/0" {
		t.Fatalf("derivation path = %q, want the Ethereum path", c.DerivationPath)
	}

	// The chain must also be present in the canonical registry so transaction
	// history and explorer resolution work.
	found := false
	for _, rc := range defaultEVMChains {
		if rc.ChainID == tigerSmartChainID {
			found = true
			if rc.Symbol != tgrNativeSymbol {
				t.Fatalf("registry symbol = %q, want %q", rc.Symbol, tgrNativeSymbol)
			}
		}
	}
	if !found {
		t.Fatalf("chain %d is missing from defaultEVMChains", tigerSmartChainID)
	}

	// chainByBlockchain is how user-wallet creation resolves a chain name.
	if got, ok := chainByBlockchain("TigerSmartChain"); !ok || got.ChainID != tigerSmartChainID {
		t.Fatalf("chainByBlockchain(\"TigerSmartChain\") = %+v (ok=%v)", got, ok)
	}
}

func TestChainRPCEndpointResolution(t *testing.T) {
	// An explicit env var wins over the registry default.
	t.Setenv(tscRPCEnv, "http://tsc-node.internal:8545")
	if got := chainRPCEndpoint(tigerSmartChainID); got != "http://tsc-node.internal:8545" {
		t.Fatalf("chainRPCEndpoint with %s set = %q", tscRPCEnv, got)
	}

	// With no env var the platform chain resolves to empty. It is the one
	// chain the wallet cannot discover for itself: the platform runs the
	// node, so there is no public URL to bundle and nothing may be dialled
	// that the operator did not configure.
	t.Setenv(tscRPCEnv, "")
	t.Setenv("CHAIN_7710_RPC", "")
	if got := chainRPCEndpoint(tigerSmartChainID); got != "" {
		t.Fatalf("chainRPCEndpoint with no configuration = %q, want empty", got)
	}
	// Its registry entry carries no endpoint either, so the fallback cannot
	// reintroduce one.
	for _, rc := range defaultEVMChains {
		if rc.ChainID == tigerSmartChainID && rc.RPCURL != "" {
			t.Fatalf("the TigerSmartChain registry entry bundles an endpoint: %q", rc.RPCURL)
		}
	}

	// A public chain, by contrast, works out of the box: its registry entry
	// publishes a real endpoint.
	t.Setenv("ETH_RPC_URL", "")
	if got := chainRPCEndpoint(1); got == "" {
		t.Fatal("chainRPCEndpoint(1) fell through to empty despite a registry entry")
	}

	// A chain that is not registered at all resolves to empty.
	if got := chainRPCEndpoint(999999999); got != "" {
		t.Fatalf("chainRPCEndpoint for an unknown chain = %q, want empty", got)
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
	}
	if tgr := seen["TGR"]; !tgr.IsNative || tgr.Contract != "" || tgr.Standard != "native" {
		t.Fatalf("TGR must be native with no contract, got %+v", tgr)
	}
	for _, sym := range []string{"WTGR", "RUSD"} {
		tok, ok := seen[sym]
		if !ok {
			t.Fatalf("%s missing", sym)
		}
		if tok.IsNative {
			t.Fatalf("%s must not be native", sym)
		}
		if !strings.EqualFold(tok.Contract, map[string]string{
			"WTGR": wtgrPredeployAddress,
			"RUSD": rusdPredeployAddress,
		}[sym]) {
			t.Fatalf("%s contract = %s, want the genesis predeploy", sym, tok.Contract)
		}
	}
	if !seen["RUSD"].PeggedUSD {
		t.Fatal("RUSD must be flagged as USD-pegged")
	}
	if seen["WTGR"].PeggedUSD {
		t.Fatal("WTGR must not be flagged as USD-pegged")
	}
}

func TestIsPlatformToken(t *testing.T) {
	for _, sym := range []string{"TGR", "tgr", "WTGR", "RUSD", " rusd "} {
		if !isPlatformToken(sym) {
			t.Fatalf("isPlatformToken(%q) = false, want true", sym)
		}
	}
	for _, sym := range []string{"", "ETH", "USDT", "USDC", "RYUSD"} {
		if isPlatformToken(sym) {
			t.Fatalf("isPlatformToken(%q) = true, want false", sym)
		}
	}
	if _, ok := platformTokenContract("TGR"); ok {
		t.Fatal("TGR is native and must not resolve to a contract")
	}
	if c, ok := platformTokenContract("rusd"); !ok || !strings.EqualFold(c, rusdPredeployAddress) {
		t.Fatalf("platformTokenContract(\"rusd\") = %q (ok=%v)", c, ok)
	}
}

func TestFetchTGRPriceFailsClosedWithoutConfiguration(t *testing.T) {
	t.Setenv(tscRPCEnv, "")
	t.Setenv(tscPairEnv, "")

	// Reset the memoised price so the call is actually attempted.
	platformPriceMu.Lock()
	platformPriceVal, platformPriceExp, platformPriceSour = 0, time.Time{}, ""
	platformPriceMu.Unlock()

	usd, source := fetchTGRPriceUSD(context.Background())
	if usd != 0 {
		t.Fatalf("fetchTGRPriceUSD returned %v with no configuration, want 0", usd)
	}
	if source != "unavailable" {
		t.Fatalf("source = %q, want \"unavailable\"", source)
	}
}

func TestNativePriceUSDFailsClosedForUnlistedChain(t *testing.T) {
	svc := &Service{}
	// Chain 7710 with no TSC_RPC_URL: the pool cannot be read, so the price is
	// reported unavailable rather than defaulting to a number.
	t.Setenv(tscRPCEnv, "")
	t.Setenv(tscPairEnv, "")
	platformPriceMu.Lock()
	platformPriceVal, platformPriceExp, platformPriceSour = 0, time.Time{}, ""
	platformPriceMu.Unlock()

	price, source := svc.nativePriceUSD(context.Background(), tigerSmartChainID)
	if price != 0 || source != "unavailable" {
		t.Fatalf("nativePriceUSD(TigerSmartChain) = %v/%q, want 0/unavailable", price, source)
	}

	// A chain with no CoinGecko mapping must also fail closed.
	if price, source := svc.nativePriceUSD(context.Background(), 999999999); price != 0 || source != "unavailable" {
		t.Fatalf("nativePriceUSD(unknown) = %v/%q, want 0/unavailable", price, source)
	}
}
