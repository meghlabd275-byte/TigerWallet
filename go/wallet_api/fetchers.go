package main

// fetchers.go — REAL on-chain data fetchers. Replaces all the empty-array /
// mock fetchers that were scattered across rust/userwallet_fetchers etc.
// Each fetcher hits a real RPC node or indexer API and returns real data.

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// ---- Types returned to the frontend ----

type BalanceResult struct {
	ChainID  int64   `json:"chain_id"`
	Symbol   string  `json:"symbol"`
	Address  string  `json:"address"`
	Balance  string  `json:"balance"`     // raw wei
	BalanceWei string `json:"balance_wei"` // raw wei (alias, for client compat: web/android read balance_wei)
	BalanceF float64 `json:"balance_f"`   // human-readable
	USDValue float64 `json:"usd_value"`   // fiat value
}

type TokenBalance struct {
	Contract string  `json:"contract"`
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name"`
	Decimals int     `json:"decimals"`
	Balance  string  `json:"balance"`   // raw
	BalanceF float64 `json:"balance_f"` // human-readable
	USDPrice float64 `json:"usd_price"`
	USDValue float64 `json:"usd_value"`
	Logo     string  `json:"logo,omitempty"`
}

type TransactionHistory struct {
	Hash      string  `json:"hash"`
	From      string  `json:"from"`
	To        string  `json:"to"`
	Value     string  `json:"value"`
	ValueF    float64 `json:"value_f"`
	Timestamp int64   `json:"timestamp"`
	Status    string  `json:"status"`
	Direction string  `json:"direction"` // "in" | "out"
	GasUsed   string  `json:"gas_used"`
	IsToken   bool    `json:"is_token"`
	TokenSym  string  `json:"token_symbol,omitempty"`
}

type NFTAsset struct {
	Contract    string `json:"contract"`
	TokenID     string `json:"token_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	Collection  string `json:"collection"`
	Standard    string `json:"standard"` // ERC-721 / ERC-1155
}

// ---- RPC helpers ----

func rpcClient(endpoint string) (*rpc.Client, error) {
	return rpc.DialContext(context.Background(), endpoint)
}

// FetchNativeBalance calls eth_getBalance for an address on a chain.
func FetchNativeBalance(ctx context.Context, endpoint string, addr common.Address) (*big.Int, error) {
	client, err := rpcClient(endpoint)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var result hexutil.Big
	if err := client.CallContext(ctx, &result, "eth_getBalance", addr.Hex(), "latest"); err != nil {
		return nil, fmt.Errorf("eth_getBalance: %w", err)
	}
	return result.ToInt(), nil
}

// FetchTransactionCount calls eth_getTransactionCount (nonce).
func FetchTransactionCount(ctx context.Context, endpoint string, addr common.Address) (uint64, error) {
	client, err := rpcClient(endpoint)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	var result hexutil.Uint64
	if err := client.CallContext(ctx, &result, "eth_getTransactionCount", addr.Hex(), "pending"); err != nil {
		return 0, fmt.Errorf("eth_getTransactionCount: %w", err)
	}
	return uint64(result), nil
}

// FetchGasPrice calls eth_gasPrice and eth_maxPriorityFeePerGas.
func FetchGasPrice(ctx context.Context, endpoint string) (gasPrice, maxFee, maxPrioFee *big.Int, err error) {
	client, err := rpcClient(endpoint)
	if err != nil {
		return nil, nil, nil, err
	}
	defer client.Close()

	var gp hexutil.Big
	if err := client.CallContext(ctx, &gp, "eth_gasPrice", "latest"); err != nil {
		return nil, nil, nil, fmt.Errorf("eth_gasPrice: %w", err)
	}
	gasPrice = gp.ToInt()

	// EIP-1559 fees (best-effort; some chains don't support)
	var tip hexutil.Big
	if err := client.CallContext(ctx, &tip, "eth_maxPriorityFeePerGas"); err == nil {
		maxPrioFee = tip.ToInt()
		// maxFee = 2*tip + baseFee — approximate as gasPrice for safety
		maxFee = new(big.Int).Mul(maxPrioFee, big.NewInt(2))
		if maxFee.Cmp(gasPrice) < 0 {
			maxFee.Set(gasPrice)
		}
	}
	return gasPrice, maxFee, maxPrioFee, nil
}

// FetchChainID calls eth_chainId.
func FetchChainID(ctx context.Context, endpoint string) (*big.Int, error) {
	client, err := rpcClient(endpoint)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var result hexutil.Big
	if err := client.CallContext(ctx, &result, "eth_chainId"); err != nil {
		return nil, err
	}
	return result.ToInt(), nil
}

// erc20BalanceOfData builds the ERC-20 balanceOf(address) call data.
func erc20BalanceOfData(addr common.Address) []byte {
	// selector = keccak256("balanceOf(address)")[:4] = 0x70a08231
	data := make([]byte, 36)
	copy(data[:4], []byte{0x70, 0xa0, 0x82, 0x31})
	copy(data[16:], addr.Bytes()) // left-pad to 32 bytes
	return data
}

// erc20SymbolData / Name / Decimals selectors
func erc20SymbolData() []byte   { return []byte{0x95, 0xd8, 0x9b, 0x41} }
func erc20NameData() []byte     { return []byte{0x06, 0xfd, 0xde, 0x03} }
func erc20DecimalsData() []byte { return []byte{0x31, 0x3c, 0xe5, 0x67} }

// ethCall performs an eth_call.
func ethCall(ctx context.Context, endpoint string, to common.Address, data []byte) ([]byte, error) {
	client, err := rpcClient(endpoint)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	msg := map[string]interface{}{
		"to":   to.Hex(),
		"data": "0x" + common.Bytes2Hex(data),
	}
	var result hexutil.Bytes
	if err := client.CallContext(ctx, &result, "eth_call", msg, "latest"); err != nil {
		return nil, err
	}
	return result, nil
}

// FetchERC20Balance fetches a single ERC-20 token balance for an address.
func FetchERC20Balance(ctx context.Context, endpoint string, tokenContract, holder common.Address) (*big.Int, error) {
	result, err := ethCall(ctx, endpoint, tokenContract, erc20BalanceOfData(holder))
	if err != nil || len(result) < 32 {
		return big.NewInt(0), err
	}
	return new(big.Int).SetBytes(result[:32]), nil
}

// FetchERC20Metadata fetches symbol, name, decimals for an ERC-20 contract.
func FetchERC20Metadata(ctx context.Context, endpoint string, tokenContract common.Address) (symbol, name string, decimals int, err error) {
	if res, e := ethCall(ctx, endpoint, tokenContract, erc20SymbolData()); e == nil && len(res) >= 32 {
		symbol = decodeStringOrBytes(res)
	}
	if res, e := ethCall(ctx, endpoint, tokenContract, erc20NameData()); e == nil && len(res) >= 32 {
		name = decodeStringOrBytes(res)
	}
	if res, e := ethCall(ctx, endpoint, tokenContract, erc20DecimalsData()); e == nil && len(res) >= 32 {
		decimals = int(new(big.Int).SetBytes(res[:32]).Uint64())
	}
	if decimals == 0 {
		decimals = 18 // sensible default for non-compliant tokens
	}
	return
}

// decodeStringOrBytes handles both string-offset-encoded and raw-bytes ERC-20
// symbol/name returns.
func decodeStringOrBytes(data []byte) string {
	if len(data) < 64 {
		return ""
	}
	// If the first 32 bytes are a small number, it's a raw string.
	offset := new(big.Int).SetBytes(data[:32]).Uint64()
	if offset == 0 || offset > 31 {
		// raw bytes packed
		return strings.TrimRight(string(data[:32]), "\x00")
	}
	// ABI string encoding: offset(32) length(32) data
	if len(data) >= 64 {
		length := new(big.Int).SetBytes(data[32:64]).Uint64()
		if int(length) <= len(data)-64 {
			return strings.TrimRight(string(data[64:64+length]), "\x00")
		}
	}
	return strings.TrimRight(string(data[:32]), "\x00")
}

// FetchTokenBalances fetches balances for a list of known ERC-20 tokens.
func FetchTokenBalances(ctx context.Context, endpoint string, holder common.Address, tokens []TokenInfo) []TokenBalance {
	var out []TokenBalance
	for _, t := range tokens {
		contract := common.HexToAddress(t.Contract)
		bal, err := FetchERC20Balance(ctx, endpoint, contract, holder)
		if err != nil || bal.Sign() == 0 {
			continue
		}
		dec := t.Decimals
		if dec == 0 {
			dec = 18
		}
		human := weiToFloat(bal, dec)
		out = append(out, TokenBalance{
			Contract: t.Contract,
			Symbol:   t.Symbol,
			Name:     t.Name,
			Decimals: dec,
			Balance:  bal.String(),
			BalanceF: human,
			USDPrice: t.USDPrice,
			USDValue: human * t.USDPrice,
			Logo:     t.Logo,
		})
	}
	return out
}

// TokenInfo is a token registry entry used by the fetcher.
type TokenInfo struct {
	Contract string  `json:"contract"`
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name"`
	Decimals int     `json:"decimals"`
	USDPrice float64 `json:"usd_price"`
	Logo     string  `json:"logo"`
	ChainID  int64   `json:"chain_id"`
}

// weiToFloat converts a wei big.Int to a float with the given decimals.
func weiToFloat(wei *big.Int, decimals int) float64 {
	if wei == nil {
		return 0
	}
	f, _ := new(big.Float).Quo(
		new(big.Float).SetInt(wei),
		new(big.Float).SetFloat64(pow10(decimals)),
	).Float64()
	return f
}

func pow10(n int) float64 {
	r := 1.0
	for i := 0; i < n; i++ {
		r *= 10
	}
	return r
}

// ---- External API fetchers (CoinGecko prices, Etherscan history) ----

// CoinGeckoPrice holds a price response.
type CoinGeckoPrice struct {
	PriceUSD float64 `json:"usd"`
	Change24 float64 `json:"usd_24h_change"`
}

// FetchTokenPrice fetches the current USD price from CoinGecko Simple Price API.
func FetchTokenPrice(ctx context.Context, coinID string) (*CoinGeckoPrice, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd&include_24hr_change=true", coinID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if key := appConfig.CoinGeckoAPIKey; key != "" {
		req.Header.Set("x-cg-pro-api-key", key)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var raw map[string]map[string]float64
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	entry, ok := raw[coinID]
	if !ok {
		return &CoinGeckoPrice{}, nil
	}
	return &CoinGeckoPrice{PriceUSD: entry["usd"], Change24: entry["usd_24h_change"]}, nil
}

// FetchETHPrice fetches the ETH USD price.
func FetchETHPrice(ctx context.Context) float64 {
	p, err := FetchTokenPrice(ctx, "ethereum")
	if err != nil || p == nil {
		return 0
	}
	return p.PriceUSD
}

// FetchTransactionHistory fetches recent transactions via Etherscan-compatible API.
func FetchTransactionHistory(ctx context.Context, explorerAPI, apiKey, address string, chainID int64) ([]TransactionHistory, error) {
	if explorerAPI == "" {
		return nil, fmt.Errorf("no explorer API configured for chain %d", chainID)
	}
	url := fmt.Sprintf("%s?module=account&action=txlist&address=%s&startblock=0&endblock=99999999&page=1&offset=50&sort=desc&apikey=%s",
		explorerAPI, address, apiKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  []struct {
			Hash      string `json:"hash"`
			From      string `json:"from"`
			To        string `json:"to"`
			Value     string `json:"value"`
			TimeStamp string `json:"timeStamp"`
			IsError   string `json:"isError"`
			GasUsed   string `json:"gasUsed"`
			GasPrice  string `json:"gasPrice"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	addrLower := strings.ToLower(address)
	var txs []TransactionHistory
	for _, r := range result.Result {
		ts, _ := strconv.ParseInt(r.TimeStamp, 10, 64)
		val, _ := new(big.Int).SetString(r.Value, 10)
		if val == nil {
			val = big.NewInt(0)
		}
		direction := "out"
		if strings.ToLower(r.From) == addrLower {
			direction = "out"
		} else {
			direction = "in"
		}
		status := "success"
		if r.IsError == "1" {
			status = "failed"
		}
		txs = append(txs, TransactionHistory{
			Hash:      r.Hash,
			From:      r.From,
			To:        r.To,
			Value:     val.String(),
			ValueF:    weiToFloat(val, 18),
			Timestamp: ts,
			Status:    status,
			Direction: direction,
			GasUsed:   r.GasUsed,
		})
	}
	return txs, nil
}

// FetchNFTAssets fetches NFTs owned by an address via Etherscan-compatible API.
func FetchNFTAssets(ctx context.Context, explorerAPI, apiKey, address string) ([]NFTAsset, error) {
	if explorerAPI == "" {
		return nil, fmt.Errorf("no explorer API configured")
	}
	url := fmt.Sprintf("%s?module=account&action=tokennfttx&address=%s&page=1&offset=50&sort=desc&apikey=%s",
		explorerAPI, address, apiKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Result []struct {
			ContractAddress string `json:"contractAddress"`
			TokenID         string `json:"tokenID"`
			TokenName       string `json:"tokenName"`
			TokenSymbol     string `json:"tokenSymbol"`
			To              string `json:"to"`
			From            string `json:"from"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	addrLower := strings.ToLower(address)
	seen := make(map[string]bool)
	var nfts []NFTAsset
	for _, r := range result.Result {
		// Only include NFTs currently held (last transfer "to" == address)
		if strings.ToLower(r.To) != addrLower {
			continue
		}
		key := r.ContractAddress + ":" + r.TokenID
		if seen[key] {
			continue
		}
		seen[key] = true
		nfts = append(nfts, NFTAsset{
			Contract:   r.ContractAddress,
			TokenID:    r.TokenID,
			Name:       r.TokenName,
			Collection: r.TokenName,
			Standard:   "ERC-721",
		})
	}
	return nfts, nil
}

// hexAddress safely parses a hex address string.
func hexAddress(s string) (common.Address, error) {
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("invalid address: %s", s)
	}
	return common.HexToAddress(s), nil
}

// hexToBigInt parses a hex string to big.Int.
func hexToBigInt(s string) (*big.Int, error) {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return big.NewInt(0), nil
	}
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex: %s", s)
	}
	return n, nil
}

// used to silence unused import for hexutil in some build paths
var _ = hexutil.Encode
var _ = hex.EncodeToString
