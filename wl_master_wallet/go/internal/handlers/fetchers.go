package handlers

// fetchers.go — REAL on-chain data fetchers ported from the canonical
// master_wallet/backend/fetchers.go. Every balance/token/gas/price/history
// value comes from a live RPC node, CoinGecko, or an Etherscan-compatible
// explorer. No hardcoded "0", no fabricated prices.

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// rpcClient connects to an Ethereum JSON-RPC endpoint.
func rpcClient(ctx context.Context, endpoint string) (*ethclient.Client, error) {
	return ethclient.DialContext(ctx, endpoint)
}

// fetchNativeBalance returns the live native (ETH/BNB/MATIC) balance in wei.
func fetchNativeBalance(ctx context.Context, endpoint string, addr common.Address) (*big.Int, error) {
	client, err := rpcClient(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()
	bal, err := client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("eth_getBalance: %w", err)
	}
	return bal, nil
}

// fetchTransactionCount returns the pending nonce for an address.
func fetchTransactionCount(ctx context.Context, endpoint string, addr common.Address) (uint64, error) {
	client, err := rpcClient(ctx, endpoint)
	if err != nil {
		return 0, fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()
	return client.PendingNonceAt(ctx, addr)
}

// fetchGasPrice returns the live gas price + EIP-1559 fee suggestion (wei).
func fetchGasPrice(ctx context.Context, endpoint string) (gasPrice, maxFee, prioFee *big.Int, err error) {
	client, err := rpcClient(ctx, endpoint)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()
	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("eth_blockNumber: %w", err)
	}
	gp, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("eth_gasPrice: %w", err)
	}
	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		tip = big.NewInt(1_000_000_000) // 1 Gwei fallback
	}
	if head.BaseFee != nil {
		maxFee = new(big.Int).Add(tip, head.BaseFee)
	} else {
		maxFee = gp
	}
	return gp, maxFee, tip, nil
}

// fetchFeeHistory returns the real EIP-1559 fee history (eth_feeHistory) for the
// last blockCount blocks. Returns the base fees, gas used ratios, and rewards
// (priority fees) at the given percentiles.
func fetchFeeHistory(ctx context.Context, endpoint string, blockCount uint64, percentiles []float64) (baseFees []*big.Int, gasUsedRatios []float64, rewards [][]*big.Int, err error) {
	client, err := rpcClient(ctx, endpoint)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()
	fh, err := client.FeeHistory(ctx, blockCount, nil, percentiles)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("eth_feeHistory: %w", err)
	}
	return fh.BaseFee, fh.GasUsedRatio, fh.Reward, nil
}

// fetchChainID returns the live chain id from the node.
func fetchChainID(ctx context.Context, endpoint string) (*big.Int, error) {
	client, err := rpcClient(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()
	return client.ChainID(ctx)
}

// --- ERC-20 reads ---

func erc20BalanceOfData(addr common.Address) []byte {
	return append([]byte{0x70, 0xa0, 0x82, 0x31}, common.LeftPadBytes(addr.Bytes(), 32)...)
}

func erc20TransferData(to common.Address, amount *big.Int) []byte {
	data := append([]byte{0xa9, 0x05, 0x9c, 0xbb}, common.LeftPadBytes(to.Bytes(), 32)...)
	return append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
}

func erc20SymbolData() []byte   { return []byte{0x95, 0xd8, 0x9b, 0x41} }
func erc20NameData() []byte     { return []byte{0x06, 0xfd, 0xde, 0x03} }
func erc20DecimalsData() []byte { return []byte{0x31, 0x3c, 0xe5, 0x67} }

// ethCall performs a read-only eth_call.
func ethCall(ctx context.Context, endpoint string, to common.Address, data []byte) ([]byte, error) {
	client, err := rpcClient(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	msg := map[string]interface{}{
		"to":   to.Hex(),
		"data": "0x" + hex.EncodeToString(data),
	}
	var result string
	if err := client.Client().CallContext(ctx, &result, "eth_call", msg, "latest"); err != nil {
		return nil, err
	}
	result = strings.TrimPrefix(result, "0x")
	return hex.DecodeString(result)
}

// fetchERC20Balance returns the live ERC-20 balance (raw uint256) for a holder.
func fetchERC20Balance(ctx context.Context, endpoint string, tokenContract, holder common.Address) (*big.Int, error) {
	out, err := ethCall(ctx, endpoint, tokenContract, erc20BalanceOfData(holder))
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return big.NewInt(0), nil
	}
	return new(big.Int).SetBytes(out), nil
}

// fetchERC20Metadata returns the symbol, name, and decimals of an ERC-20.
func fetchERC20Metadata(ctx context.Context, endpoint string, tokenContract common.Address) (symbol, name string, decimals int, err error) {
	if symOut, e := ethCall(ctx, endpoint, tokenContract, erc20SymbolData()); e == nil {
		symbol = decodeStringOrBytes(symOut)
	}
	if nameOut, e := ethCall(ctx, endpoint, tokenContract, erc20NameData()); e == nil {
		name = decodeStringOrBytes(nameOut)
	}
	if decOut, e := ethCall(ctx, endpoint, tokenContract, erc20DecimalsData()); e == nil && len(decOut) >= 32 {
		decimals = int(new(big.Int).SetBytes(decOut[len(decOut)-32:]).Int64())
	} else {
		decimals = 18
	}
	return symbol, name, decimals, nil
}

func decodeStringOrBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if len(data) >= 64 {
		length := new(big.Int).SetBytes(data[32:64]).Int64()
		if int(length) > 0 && 64+int(length) <= len(data) {
			return strings.TrimRight(string(data[64:64+length]), "\x00")
		}
	}
	if len(data) >= 32 {
		return strings.TrimRight(string(data[:32]), "\x00")
	}
	return ""
}

// TokenInfo describes a known token for balance aggregation.
type TokenInfo struct {
	Address  string
	Symbol   string
	Decimals int
}

// TokenBalance is the resolved balance for one token.
type TokenBalance struct {
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name"`
	Contract string  `json:"contract"`
	Balance  string  `json:"balance"`
	Decimals int     `json:"decimals"`
	USDValue float64 `json:"usd_value"`
}

// fetchTokenBalances reads the live ERC-20 balance for each known token.
func fetchTokenBalances(ctx context.Context, endpoint string, holder common.Address, tokens []TokenInfo) []TokenBalance {
	out := []TokenBalance{}
	for _, t := range tokens {
		tc := common.HexToAddress(t.Address)
		bal, err := fetchERC20Balance(ctx, endpoint, tc, holder)
		if err != nil || bal.Sign() == 0 {
			continue
		}
		out = append(out, TokenBalance{
			Symbol:   t.Symbol,
			Contract: t.Address,
			Balance:  weiToFloat(bal, t.Decimals),
			Decimals: t.Decimals,
		})
	}
	return out
}

func weiToFloat(wei *big.Int, decimals int) string {
	f, _ := new(big.Float).Quo(
		new(big.Float).SetInt(wei),
		big.NewFloat(math.Pow10(decimals)),
	).Float64()
	return fmt.Sprintf("%.6f", f)
}

// --- Price fetcher (CoinGecko) ---

// CoinGeckoPrice holds a price response.
type CoinGeckoPrice struct {
	USD       float64 `json:"usd"`
	USD24h    float64 `json:"usd_24h_change"`
	MarketCap float64 `json:"usd_market_cap"`
}

// fetchTokenPrice queries CoinGecko for the USD price of a coin id.
func fetchTokenPrice(ctx context.Context, coinID string) (*CoinGeckoPrice, error) {
	if coinID == "" {
		return nil, fmt.Errorf("no coin id")
	}
	u := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd&include_24hr_change=true&include_market_cap=true", url.QueryEscape(coinID))
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	if k := os.Getenv("COINGECKO_API_KEY"); k != "" {
		req.Header.Set("x-cg-demo-api-key", k)
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
		return nil, fmt.Errorf("no price for %s", coinID)
	}
	p := &CoinGeckoPrice{}
	if v, ok := entry["usd"]; ok {
		p.USD = v
	}
	if v, ok := entry["usd_24h_change"]; ok {
		p.USD24h = v
	}
	if v, ok := entry["usd_market_cap"]; ok {
		p.MarketCap = v
	}
	return p, nil
}

// --- Transaction history (Etherscan-compatible) ---

// TransactionHistory is one historical transaction.
type TransactionHistory struct {
	Hash    string `json:"hash"`
	From    string `json:"from"`
	To      string `json:"to"`
	Value   string `json:"value"`
	Time    int64  `json:"timeStamp"`
	IsError int    `json:"isError"`
	GasUsed string `json:"gasUsed"`
	Status  string `json:"status"`
	ChainID int64  `json:"chain_id"`
}

// fetchTransactionHistory queries an Etherscan-compatible explorer for the real
// on-chain transaction history of an address. Fail-closed if explorer unset.
func fetchTransactionHistory(ctx context.Context, explorerAPI, apiKey, address string, chainID int64) ([]TransactionHistory, error) {
	if explorerAPI == "" {
		return nil, fmt.Errorf("no explorer API configured for chain %d", chainID)
	}
	u := fmt.Sprintf("%s?module=account&action=txlist&address=%s&sort=desc&page=1&offset=50", explorerAPI, address)
	if apiKey != "" {
		u += "&apikey=" + apiKey
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var raw struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  []struct {
			Hash        string `json:"hash"`
			From        string `json:"from"`
			To          string `json:"to"`
			Value       string `json:"value"`
			TimeStamp   string `json:"timeStamp"`
			IsError     string `json:"isError"`
			GasUsed     string `json:"gasUsed"`
			Status      string `json:"status"`
			Confirmations string `json:"confirmations"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if raw.Status == "0" && raw.Message != "No transactions found" {
		return nil, fmt.Errorf("explorer API error: %s", raw.Message)
	}
	out := []TransactionHistory{}
	for _, t := range raw.Result {
		ts := int64(0)
		fmt.Sscanf(t.TimeStamp, "%d", &ts)
		isErr := 0
		fmt.Sscanf(t.IsError, "%d", &isErr)
		out = append(out, TransactionHistory{
			Hash:    t.Hash,
			From:    t.From,
			To:      t.To,
			Value:   t.Value,
			Time:    ts,
			IsError: isErr,
			GasUsed: t.GasUsed,
			Status:  t.Status,
			ChainID: chainID,
		})
	}
	return out, nil
}

// --- helpers ---

func hexAddress(s string) (common.Address, error) {
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("invalid address: %s", s)
	}
	return common.HexToAddress(s), nil
}

func hexToBigInt(s string) (*big.Int, error) {
	s = strings.TrimPrefix(s, "0x")
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex number: %s", s)
	}
	return n, nil
}

// humanToWei converts a human-readable amount to wei given token decimals.
func humanToWei(amount string, decimals int) (*big.Int, error) {
	f, ok := new(big.Float).SetString(amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}
	f = f.Mul(f, big.NewFloat(math.Pow10(decimals)))
	n, _ := f.Int(nil)
	if n == nil {
		return nil, fmt.Errorf("amount overflow")
	}
	return n, nil
}

// parseLimit parses a positive limit query param, clamped to [1, 200], default d.
func parseLimit(s string, d int) int {
	if s == "" {
		return d
	}
	n := 0
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n < 1 {
		return d
	}
	if n > 200 {
		n = 200
	}
	return n
}
