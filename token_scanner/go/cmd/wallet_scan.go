package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"time"
)

// nativeChainSymbols maps a configured chain onto the native coin symbol.
var nativeChainSymbols = map[string]string{
	"ethereum": "ETH",
	"polygon":  "MATIC",
	"bsc":      "BNB",
}

// nativeCoinGeckoIDs maps a native symbol onto its CoinGecko asset id.
var nativeCoinGeckoIDs = map[string]string{
	"ETH":   "ethereum",
	"MATIC": "matic-network",
	"BNB":   "binancecoin",
	"BTC":   "bitcoin",
	"SOL":   "solana",
	"USDT":  "tether",
	"USDC":  "usd-coin",
}

// transferTopic is keccak256("Transfer(address,address,uint256)").
var transferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

// scanLookbackBlocks bounds eth_getLogs to a range most public nodes accept.
const scanLookbackBlocks = 10000

// WalletScan is the result of a real on-chain wallet scan.
type WalletScan struct {
	Chain       string   `json:"chain"`
	Native      string   `json:"nativeBalance"`
	NativeWei   string   `json:"nativeBalanceWei"`
	TokenCount  int      `json:"tokenCount"`
	Tokens      []*Token `json:"tokens"`
	FromBlock   int64    `json:"fromBlock"`
	ToBlock     int64    `json:"toBlock"`
	ScanPartial bool     `json:"scanPartial"`
}

// ScanWalletOnChain discovers the tokens a wallet actually holds on one chain.
//
// Discovery is a real eth_getLogs query for ERC-20 Transfer events addressed to
// the wallet over the recent block window. Each candidate contract is confirmed
// with balanceOf; a token whose balance is zero is not reported. The native coin
// balance is read with eth_getBalance. Nothing is invented: if the chain has no
// configured endpoint the call fails instead of returning a placeholder list.
func (s *TokenScannerService) ScanWalletOnChain(ctx context.Context, wallet, chain string) (*WalletScan, error) {
	if !isValidAddress(wallet) {
		return nil, fmt.Errorf("invalid wallet address %q", wallet)
	}
	wallet = strings.ToLower(wallet)

	endpoint := s.rpcForChain(chain)
	if endpoint == "" {
		return nil, fmt.Errorf("no RPC endpoint configured for chain %q", chain)
	}
	chainKey := normalizeChainKey(chain)
	client := &http.Client{Timeout: rpcTimeout}

	head, err := rpcBlockNumber(ctx, client, endpoint)
	if err != nil {
		return nil, err
	}

	from := new(big.Int).Sub(head, big.NewInt(scanLookbackBlocks))
	if from.Sign() < 0 {
		from = big.NewInt(0)
	}

	scan := &WalletScan{
		Chain:     chainKey,
		Tokens:    []*Token{},
		FromBlock: from.Int64(),
		ToBlock:   head.Int64(),
	}

	wei, err := rpcBalance(ctx, client, endpoint, wallet)
	if err != nil {
		return nil, fmt.Errorf("read native balance: %w", err)
	}
	scan.NativeWei = wei.String()
	scan.Native = formatUnits(wei, 18) + " " + nativeChainSymbols[chainKey]

	contracts, err := s.transferContracts(ctx, client, endpoint, wallet, from)
	if err != nil {
		// Report honestly: the native balance is real, the token list is not
		// claimed to be complete.
		scan.ScanPartial = true
		return scan, fmt.Errorf("read transfer logs: %w", err)
	}

	for _, contract := range contracts {
		token, err := s.readToken(ctx, client, endpoint, contract, chainKey, wallet)
		if err != nil {
			continue
		}
		scan.Tokens = append(scan.Tokens, token)
	}

	sort.Slice(scan.Tokens, func(i, j int) bool {
		return strings.ToLower(scan.Tokens[i].Symbol) < strings.ToLower(scan.Tokens[j].Symbol)
	})
	scan.TokenCount = len(scan.Tokens)
	return scan, nil
}

// transferContracts returns the distinct ERC-20 contracts that emitted a
// Transfer event to the wallet in the given block window.
func (s *TokenScannerService) transferContracts(ctx context.Context, client *http.Client, endpoint, wallet string, from *big.Int) ([]string, error) {
	paddedWallet := "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(wallet, "0x")
	logsRaw, err := callRPC(ctx, client, endpoint, "eth_getLogs", map[string]interface{}{
		"fromBlock": fmt.Sprintf("0x%x", from.Int64()),
		"toBlock":   "latest",
		"topics":    []interface{}{transferTopic, nil, paddedWallet},
	})
	if err != nil {
		return nil, err
	}

	var logs []struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(logsRaw, &logs); err != nil {
		return nil, fmt.Errorf("decode transfer logs: %w", err)
	}

	seen := make(map[string]bool, len(logs))
	out := make([]string, 0, len(logs))
	for _, entry := range logs {
		contract := strings.ToLower(strings.TrimSpace(entry.Address))
		if !isValidAddress(contract) || seen[contract] {
			continue
		}
		seen[contract] = true
		out = append(out, contract)
	}
	return out, nil
}

// readToken reads the ERC-20 metadata and the holder's balance for one contract.
// A contract that is not ERC-20, or whose balance for the holder is zero, is
// rejected so it is never reported as a holding.
func (s *TokenScannerService) readToken(ctx context.Context, client *http.Client, endpoint, contract, chain, holder string) (*Token, error) {
	analysis, err := s.AnalyzeTokenOnChain(ctx, contract, chain)
	if err != nil {
		return nil, err
	}
	for _, flag := range analysis.Flags {
		if flag == "not_a_contract" || flag == "not_erc20" {
			return nil, fmt.Errorf("%s is not an ERC-20 contract", contract)
		}
	}

	token := &Token{
		Address:   contract,
		Type:      TokenTypeERC20,
		Chain:     chain,
		Status:    StatusNew,
		RiskScore: analysis.RiskScore,
		UpdatedAt: time.Now().Unix(),
	}

	if raw, err := ethCall(ctx, client, endpoint, contract, selectorCall("name()")); err == nil {
		if name, err := decodeABIString(raw); err == nil {
			token.Name = name
		}
	}
	if raw, err := ethCall(ctx, client, endpoint, contract, selectorCall("symbol()")); err == nil {
		if symbol, err := decodeABIString(raw); err == nil {
			token.Symbol = symbol
		}
	}
	if raw, err := ethCall(ctx, client, endpoint, contract, selectorCall("decimals()")); err == nil {
		if decimals, err := decodeABIUint(raw); err == nil && decimals.IsInt64() {
			token.Decimals = int(decimals.Int64())
		}
	}
	if raw, err := ethCall(ctx, client, endpoint, contract, selectorCall("totalSupply()")); err == nil {
		if supply, err := decodeABIUint(raw); err == nil {
			token.TotalSupply = supply.String()
		}
	}

	if token.Symbol == "" && token.Name == "" {
		return nil, fmt.Errorf("contract %s exposes no ERC-20 metadata", contract)
	}

	balanceRaw, err := ethCall(ctx, client, endpoint, contract, selectorCallWithAddress("balanceOf(address)", holder))
	if err != nil {
		return nil, fmt.Errorf("read balanceOf: %w", err)
	}
	balance, err := decodeABIUint(balanceRaw)
	if err != nil {
		return nil, fmt.Errorf("decode balanceOf: %w", err)
	}
	if balance.Sign() == 0 {
		return nil, fmt.Errorf("zero balance for %s", contract)
	}
	token.BalanceRaw = balance.String()
	token.BalanceFormatted = formatUnits(balance, token.Decimals)
	return token, nil
}

// CalculatePortfolioValueOnChains values a wallet across the configured chains.
//
// Every figure is derived from real on-chain balances priced through the
// configured price API. A holding whose price cannot be resolved is listed in
// unpriced and contributes nothing, so a total is never inflated by a guess.
func (s *TokenScannerService) CalculatePortfolioValueOnChains(ctx context.Context, wallet string) (float64, map[string]float64, []string, error) {
	if !isValidAddress(wallet) {
		return 0, nil, nil, fmt.Errorf("invalid wallet address %q", wallet)
	}

	var chains []string
	for _, chain := range []string{"ethereum", "polygon", "bsc"} {
		if s.rpcForChain(chain) != "" {
			chains = append(chains, chain)
		}
	}
	if len(chains) == 0 {
		return 0, nil, nil, fmt.Errorf("no chain RPC endpoints are configured")
	}

	holdings := make(map[string]float64)
	var unpriced []string
	var firstErr error
	total := 0.0

	for _, chain := range chains {
		scan, err := s.ScanWalletOnChain(ctx, wallet, chain)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", chain, err)
		}
		if scan == nil {
			continue
		}

		nativeSymbol := nativeChainSymbols[normalizeChainKey(chain)]
		if scan.NativeWei != "" && scan.NativeWei != "0" {
			price, err := s.nativePrice(ctx, nativeSymbol)
			if err != nil {
				unpriced = append(unpriced, chain+":"+nativeSymbol)
			} else {
				wei, _ := new(big.Int).SetString(scan.NativeWei, 10)
				value := weiToFloat(wei) * price
				holdings[chain] += value
				total += value
			}
		}

		for _, token := range scan.Tokens {
			price, err := s.tokenPriceUSD(ctx, token.Address, chain)
			if err != nil {
				unpriced = append(unpriced, chain+":"+token.Symbol)
				continue
			}
			balance, ok := new(big.Int).SetString(token.BalanceRaw, 10)
			if !ok {
				continue
			}
			value := formatUnitsFloat(balance, token.Decimals) * price
			holdings[chain] += value
			total += value
		}
	}

	if total == 0 && firstErr != nil {
		return 0, holdings, unpriced, firstErr
	}
	return total, holdings, unpriced, nil
}

// nativePrice returns the USD price of a native coin via the configured price API.
func (s *TokenScannerService) nativePrice(ctx context.Context, symbol string) (float64, error) {
	id, ok := nativeCoinGeckoIDs[strings.ToUpper(symbol)]
	if !ok {
		return 0, fmt.Errorf("no price mapping for %s", symbol)
	}
	return s.simplePrice(ctx, "ids", id)
}

// tokenPriceUSD returns the USD price of an ERC-20 contract via the configured
// price API, keyed by contract address.
func (s *TokenScannerService) tokenPriceUSD(ctx context.Context, contract, chain string) (float64, error) {
	if !isValidAddress(contract) {
		return 0, fmt.Errorf("invalid token address %q", contract)
	}
	return s.simplePrice(ctx, "contract_addresses", strings.ToLower(contract), coinGeckoPlatform(chain))
}

// simplePrice performs a CoinGecko simple price lookup and returns the USD value.
func (s *TokenScannerService) simplePrice(ctx context.Context, key, value string, platform ...string) (float64, error) {
	endpoint := strings.TrimSpace(s.config.CoingeckoURL)
	if endpoint == "" {
		return 0, fmt.Errorf("no price API is configured (COINGECKO_URL)")
	}

	path := fmt.Sprintf("%s/simple/price?%s=%s&vs_currencies=usd", strings.TrimRight(endpoint, "/"), key, value)
	if key == "contract_addresses" {
		if len(platform) == 0 || platform[0] == "" {
			return 0, fmt.Errorf("no CoinGecko platform for chain")
		}
		path = fmt.Sprintf("%s/simple/token_price/%s?contract_addresses=%s&vs_currencies=usd",
			strings.TrimRight(endpoint, "/"), platform[0], value)
	}

	reqCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: rpcTimeout}).Do(req)
	if err != nil {
		return 0, fmt.Errorf("price API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("price API returned HTTP %d", resp.StatusCode)
	}

	var decoded map[string]struct {
		USD float64 `json:"usd"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return 0, fmt.Errorf("decode price response: %w", err)
	}
	entry, ok := decoded[strings.ToLower(value)]
	if !ok || entry.USD <= 0 {
		return 0, fmt.Errorf("no price is available for %s", value)
	}
	return entry.USD, nil
}

// coinGeckoPlatform maps a chain onto its CoinGecko platform id.
func coinGeckoPlatform(chain string) string {
	switch normalizeChainKey(chain) {
	case "polygon":
		return "polygon-pos"
	case "bsc":
		return "binance-smart-chain"
	default:
		return "ethereum"
	}
}

// selectorCall builds calldata for a no-argument call.
func selectorCall(signature string) string {
	return "0x" + hexOfSelector(signature)
}

// selectorCallWithAddress builds calldata for a call taking one address.
func selectorCallWithAddress(signature, address string) string {
	return "0x" + hexOfSelector(signature) + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(address), "0x")
}

// hexOfSelector returns the 4-byte function selector as hex.
func hexOfSelector(signature string) string {
	selector := keccakSelector(signature)
	return hex.EncodeToString(selector[:])
}

// normalizeChainKey maps a chain id or alias onto its canonical key.
func normalizeChainKey(chain string) string {
	switch strings.ToLower(strings.TrimSpace(chain)) {
	case "137", "polygon", "matic":
		return "polygon"
	case "56", "bsc", "binance":
		return "bsc"
	default:
		return "ethereum"
	}
}

// rpcBlockNumber reads the current head block height.
func rpcBlockNumber(ctx context.Context, client *http.Client, endpoint string) (*big.Int, error) {
	raw, err := callRPC(ctx, client, endpoint, "eth_blockNumber")
	if err != nil {
		return nil, fmt.Errorf("read block height: %w", err)
	}
	var headHex string
	if err := json.Unmarshal(raw, &headHex); err != nil {
		return nil, fmt.Errorf("decode block height: %w", err)
	}
	head, err := hexQuantityToBigInt(headHex)
	if err != nil {
		return nil, fmt.Errorf("parse block height: %w", err)
	}
	return head, nil
}

// rpcBalance reads the native coin balance of an address in wei.
func rpcBalance(ctx context.Context, client *http.Client, endpoint, address string) (*big.Int, error) {
	raw, err := callRPC(ctx, client, endpoint, "eth_getBalance", address, "latest")
	if err != nil {
		return nil, err
	}
	var balHex string
	if err := json.Unmarshal(raw, &balHex); err != nil {
		return nil, fmt.Errorf("decode native balance: %w", err)
	}
	return hexQuantityToBigInt(balHex)
}

// weiToFloat converts a wei amount into ether units as a float for display math.
func weiToFloat(wei *big.Int) float64 {
	if wei == nil {
		return 0
	}
	value, _ := new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(1e18)).Float64()
	return value
}

// formatUnits renders an integer amount scaled by decimals as a decimal string.
func formatUnits(value *big.Int, decimals int) string {
	if value == nil {
		return "0"
	}
	if decimals <= 0 {
		return value.String()
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole, frac := new(big.Int).QuoRem(value, divisor, new(big.Int))
	if frac.Sign() == 0 {
		return whole.String()
	}
	fracStr := frac.String()
	for len(fracStr) < decimals {
		fracStr = "0" + fracStr
	}
	fracStr = strings.TrimRight(fracStr, "0")
	return whole.String() + "." + fracStr
}

// formatUnitsFloat converts a scaled integer amount into a float.
func formatUnitsFloat(value *big.Int, decimals int) float64 {
	if value == nil {
		return 0
	}
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	out, _ := new(big.Float).Quo(new(big.Float).SetInt(value), divisor).Float64()
	return out
}
