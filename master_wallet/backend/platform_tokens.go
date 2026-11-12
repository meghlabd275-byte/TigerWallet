package main

// platform_tokens.go — the platform's own assets as MasterWallet sees them.
//
// TigerSmartChain (EIP-155 chain id 7710) is the network TigerWallet operates.
// Its genesis block carries two contracts as predeploys, so their addresses are
// fixed by consensus and identical on every node from block zero:
//
//	RUSD  Royal Tiger USD - the USD stablecoin, minted against a treasury reserve
//	WTGR  Wrapped TGR      - the ERC-20 wrapper of the native TGR coin
//
// TGR itself is the native coin, like ETH on Ethereum, so it has no contract
// address.
//
// Two things in this file matter beyond the metadata:
//
//  1. The RPC endpoint of the chain defaults to loopback and is overridden by
//     TSC_RPC_URL. The chain is operated by the platform, so unlike a public
//     chain there is no well-known endpoint to fall back on; the operator has
//     to state where the node is. When it is unset every read fails closed
//     rather than dialling a placeholder host.
//
//  2. The price of TGR is read from the on-chain TigerSwap pool. Until a
//     centralised exchange lists TGR there is no order book, and the pool
//     reserves are the only real market price. If the pool address is not
//     configured the price is reported as unavailable, never guessed.

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// The canonical predeploy addresses and chain metadata. These are duplicated
// from tiger_smart_chain/chain/predeploy.go because the two live in separate Go
// modules; TestPlatformTokensMatchChain reads that file and fails if the two
// ever disagree, so the duplication cannot drift silently.
const (
	// tigerSmartChainID is the EIP-155 chain id of TigerSmartChain.
	tigerSmartChainID int64 = 7710

	// rusdPredeployAddress is the RUSD stablecoin on TigerSmartChain.
	rusdPredeployAddress = "0x0000000000000000000000000000000000007710"

	// wtgrPredeployAddress is the WTGR wrapped-native token on TigerSmartChain.
	wtgrPredeployAddress = "0x0000000000000000000000000000000000007711"

	// tgrNativeSymbol is the native coin symbol of TigerSmartChain.
	tgrNativeSymbol = "TGR"

	// tscRPCEnv names the environment variable holding the TigerSmartChain RPC.
	tscRPCEnv = "TSC_RPC_URL"

	// tscPairEnv names the environment variable holding the TGR/RUSD pair
	// address used to derive the TGR price from live pool reserves.
	tscPairEnv = "TSC_TGR_RUSD_PAIR"
)

// platformChain is the TigerSmartChain entry for the MasterWallet chain list.
// It is appended to supportedChains at package load so every code path that
// resolves a chain (treasury, signer, balance, user wallets) can select it.
var platformChain = ChainConfig{
	ChainID:        tigerSmartChainID,
	Name:           "TigerSmartChain",
	Symbol:         tgrNativeSymbol,
	Blockchain:     "tigersmartchain",
	RPCEnv:         tscRPCEnv,
	Decimals:       18,
	DerivationPath: "m/44'/60'/0'/0/0",
	IsEVM:          true,
}

func init() {
	for _, c := range supportedChains {
		if c.ChainID == tigerSmartChainID {
			return
		}
	}
	supportedChains = append(supportedChains, platformChain)
}

// PlatformToken describes a first-class platform asset.
type PlatformToken struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	ChainID   int64   `json:"chain_id"`
	Decimals  int     `json:"decimals"`
	Contract  string  `json:"contract,omitempty"`
	IsNative  bool    `json:"is_native"`
	Standard  string  `json:"standard"`
	USDPrice  float64 `json:"usd_price"`
	USDSource string  `json:"usd_price_source"`
	PeggedUSD bool    `json:"pegged_usd"`
}

// platformTokensStatic returns the platform asset metadata without touching the
// network. Used for validation and address resolution where blocking on a price
// fetch would be wrong.
func platformTokensStatic() []PlatformToken {
	return []PlatformToken{
		{Symbol: tgrNativeSymbol, Name: "Tiger", ChainID: tigerSmartChainID, Decimals: 18, IsNative: true, Standard: "native"},
		{Symbol: "WTGR", Name: "Wrapped TGR", ChainID: tigerSmartChainID, Decimals: 18, Contract: wtgrPredeployAddress, Standard: "ERC-20"},
		{Symbol: "RUSD", Name: "Royal Tiger USD", ChainID: tigerSmartChainID, Decimals: 18, Contract: rusdPredeployAddress, Standard: "ERC-20", PeggedUSD: true},
	}
}

// isPlatformToken reports whether a symbol belongs to the platform asset set.
func isPlatformToken(symbol string) bool {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	for _, t := range platformTokensStatic() {
		if t.Symbol == sym {
			return true
		}
	}
	return false
}

// platformTokenContract returns the contract address of a platform token, or
// ("", false) when the symbol is not a platform contract token.
func platformTokenContract(symbol string) (string, bool) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	for _, t := range platformTokensStatic() {
		if t.Symbol == sym && t.Contract != "" {
			return t.Contract, true
		}
	}
	return "", false
}

// ---- TGR price from live pool reserves -----------------------------------

const (
	// platformPriceTTL is how long a pool-derived price is reused. Pool
	// reserves only change when somebody trades, so a short TTL keeps the price
	// fresh without hammering the node on every balance request.
	platformPriceTTL = 30 * time.Second
)

var (
	platformPriceMu   sync.Mutex
	platformPriceVal  float64
	platformPriceExp  time.Time
	platformPriceSour string
)

// fetchTGRPriceUSD returns the USD price of TGR read from the live TGR/RUSD
// pool on TigerSmartChain. RUSD is the USD leg, so the ratio of the two
// reserves is the price directly; no external oracle is involved and no value
// is invented when the pool is unreachable.
func fetchTGRPriceUSD(ctx context.Context) (float64, string) {
	platformPriceMu.Lock()
	if time.Now().Before(platformPriceExp) && platformPriceVal > 0 {
		v, s := platformPriceVal, platformPriceSour
		platformPriceMu.Unlock()
		return v, s
	}
	platformPriceMu.Unlock()

	rpc := strings.TrimSpace(os.Getenv(tscRPCEnv))
	pair := strings.TrimSpace(os.Getenv(tscPairEnv))
	if rpc == "" || !common.IsHexAddress(pair) {
		return 0, "unavailable"
	}

	callCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	// getReserves() -> (uint112, uint112, uint32): selector 0x0902f1ac
	raw, err := rpcEthCallRaw(callCtx, rpc, pair, "0x0902f1ac")
	if err != nil {
		return 0, "unavailable"
	}
	raw = strings.TrimPrefix(raw, "0x")
	if len(raw) < 192 {
		return 0, "unavailable"
	}
	reserve0, ok0 := new(big.Int).SetString(raw[0:64], 16)
	reserve1, ok1 := new(big.Int).SetString(raw[64:128], 16)
	if !ok0 || !ok1 || reserve0.Sign() <= 0 || reserve1.Sign() <= 0 {
		return 0, "unavailable"
	}

	// The pool's token0/token1 ordering decides which reserve is RUSD. Read it
	// rather than assume it: token0() 0x0dfe1681, token1() 0x4f1a4a83.
	t0Raw, err0 := rpcEthCallRaw(callCtx, rpc, pair, "0x0dfe1681")
	t1Raw, err1 := rpcEthCallRaw(callCtx, rpc, pair, "0x4f1a4a83")
	if err0 != nil || err1 != nil {
		return 0, "unavailable"
	}
	t0Raw = strings.TrimPrefix(t0Raw, "0x")
	t1Raw = strings.TrimPrefix(t1Raw, "0x")
	if len(t0Raw) < 64 || len(t1Raw) < 64 {
		return 0, "unavailable"
	}
	token0 := strings.ToLower(common.HexToAddress("0x" + t0Raw[24:64]).Hex())
	token1 := strings.ToLower(common.HexToAddress("0x" + t1Raw[24:64]).Hex())
	rusd := strings.ToLower(common.HexToAddress(rusdPredeployAddress).Hex())

	// Exactly one leg must be RUSD. A pool holding RUSD on both sides has no
	// asset leg, and its ratio would always read 1.0; that is a malformed
	// pool, not a one-dollar price, so it is refused.
	var tgrReserve, rusdReserve *big.Int
	switch {
	case token0 == rusd && token1 != rusd:
		rusdReserve, tgrReserve = reserve0, reserve1
	case token1 == rusd && token0 != rusd:
		rusdReserve, tgrReserve = reserve1, reserve0
	default:
		// The configured pair does not hold RUSD, so its ratio says nothing
		// about the dollar price. Fail closed rather than return a wrong number.
		return 0, "unavailable"
	}
	if tgrReserve.Sign() <= 0 {
		return 0, "unavailable"
	}

	ratio := new(big.Float).Quo(new(big.Float).SetInt(rusdReserve), new(big.Float).SetInt(tgrReserve))
	usd, _ := ratio.Float64()
	if usd <= 0 {
		return 0, "unavailable"
	}

	platformPriceMu.Lock()
	platformPriceVal, platformPriceExp, platformPriceSour = usd, time.Now().Add(platformPriceTTL), "onchain_pool"
	platformPriceMu.Unlock()
	return usd, "onchain_pool"
}

// nativePriceUSD returns the USD price of a chain's native coin together with
// the source it came from.
//
// TigerSmartChain is handled separately: TGR is the platform's own coin and
// there is no exchange listing for it, so its price is read from the live
// TigerSwap pool (see fetchTGRPriceUSD). Every other chain uses the cached
// CoinGecko lookup the wallet already relies on.
//
// A zero price with source "unavailable" means no source could answer. Callers
// must omit the fiat value in that case rather than substitute a default.
func (svc *Service) nativePriceUSD(ctx context.Context, chainID int64) (float64, string) {
	if chainID == tigerSmartChainID {
		return fetchTGRPriceUSD(ctx)
	}
	coinID := chainCoinGeckoID(chainID)
	if coinID == "" {
		return 0, "unavailable"
	}
	p, err := svc.FetchTokenPriceCached(ctx, coinID)
	if err != nil || p == nil || p.USD <= 0 {
		return 0, "unavailable"
	}
	return p.USD, "coingecko"
}

// rpcEthCallRaw performs an eth_call and returns the 0x-prefixed hex result.
func rpcEthCallRaw(ctx context.Context, endpoint, to, data string) (string, error) {
	client, err := rpcClient(ctx, endpoint)
	if err != nil {
		return "", err
	}
	defer client.Close()
	msg := map[string]interface{}{"to": to, "data": data}
	var result string
	if err := client.Client().CallContext(ctx, &result, "eth_call", msg, "latest"); err != nil {
		return "", err
	}
	if result == "" {
		return "", fmt.Errorf("empty eth_call result")
	}
	return result, nil
}
