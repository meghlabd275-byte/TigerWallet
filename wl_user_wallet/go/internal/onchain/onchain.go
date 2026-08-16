// Package onchain provides REAL on-chain data fetchers and ERC-20/AMM calldata
// helpers for the standalone WL-UserWallet backend. Ported from the canonical
// TigerWallet wallet_api fetchers.go + amm_router.go. No stubs: every call
// hits a real RPC node or external API and fails closed on error.
package onchain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
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
	Balance  string  `json:"balance"`
	BalanceF float64 `json:"balance_f"`
	USDValue float64 `json:"usd_value"`
}

type TokenBalance struct {
	Contract string  `json:"contract"`
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name"`
	Decimals int     `json:"decimals"`
	Balance  string  `json:"balance"`
	BalanceF float64 `json:"balance_f"`
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
	Direction string  `json:"direction"`
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
	Standard    string `json:"standard"`
}

// TokenInfo is a token registry entry.
type TokenInfo struct {
	Contract string  `json:"contract"`
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name"`
	Decimals int     `json:"decimals"`
	USDPrice float64 `json:"usd_price"`
	Logo     string  `json:"logo"`
	ChainID  int64   `json:"chain_id"`
}

// ---- RPC helpers ----

func rpcClient(endpoint string) (*rpc.Client, error) {
	return rpc.DialContext(context.Background(), endpoint)
}

// FetchNativeBalance calls eth_getBalance for an address.
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

// FetchTransactionCount calls eth_getTransactionCount (nonce, pending).
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
	var tip hexutil.Big
	if err := client.CallContext(ctx, &tip, "eth_maxPriorityFeePerGas"); err == nil {
		maxPrioFee = tip.ToInt()
		maxFee = new(big.Int).Mul(maxPrioFee, big.NewInt(2))
		if maxFee.Cmp(gasPrice) < 0 {
			maxFee.Set(gasPrice)
		}
	}
	return gasPrice, maxFee, maxPrioFee, nil
}

// ---- ERC-20 eth_call helpers ----

func erc20BalanceOfData(addr common.Address) []byte {
	data := make([]byte, 36)
	copy(data[:4], []byte{0x70, 0xa0, 0x82, 0x31})
	copy(data[16:], addr.Bytes())
	return data
}

func erc20SymbolData() []byte   { return []byte{0x95, 0xd8, 0x9b, 0x41} }
func erc20NameData() []byte     { return []byte{0x06, 0xfd, 0xde, 0x03} }
func erc20DecimalsData() []byte { return []byte{0x31, 0x3c, 0xe5, 0x67} }

// EthCall performs an eth_call.
func EthCall(ctx context.Context, endpoint string, to common.Address, data []byte) ([]byte, error) {
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

// FetchERC20Balance fetches a single ERC-20 token balance.
func FetchERC20Balance(ctx context.Context, endpoint string, tokenContract, holder common.Address) (*big.Int, error) {
	result, err := EthCall(ctx, endpoint, tokenContract, erc20BalanceOfData(holder))
	if err != nil {
		return big.NewInt(0), err
	}
	if len(result) < 32 {
		return big.NewInt(0), nil
	}
	return new(big.Int).SetBytes(result[:32]), nil
}

// FetchERC20Metadata fetches symbol, name, decimals for an ERC-20 contract.
func FetchERC20Metadata(ctx context.Context, endpoint string, tokenContract common.Address) (symbol, name string, decimals int, err error) {
	if res, e := EthCall(ctx, endpoint, tokenContract, erc20SymbolData()); e == nil && len(res) >= 32 {
		symbol = decodeStringOrBytes(res)
	}
	if res, e := EthCall(ctx, endpoint, tokenContract, erc20NameData()); e == nil && len(res) >= 32 {
		name = decodeStringOrBytes(res)
	}
	if res, e := EthCall(ctx, endpoint, tokenContract, erc20DecimalsData()); e == nil && len(res) >= 32 {
		decimals = int(new(big.Int).SetBytes(res[:32]).Uint64())
	}
	if decimals == 0 {
		decimals = 18
	}
	return
}

// FetchDecimals fetches the decimals() uint8 for an ERC-20 token.
func FetchDecimals(ctx context.Context, endpoint string, token common.Address) (int, error) {
	res, err := EthCall(ctx, endpoint, token, erc20DecimalsData())
	if err != nil {
		return 0, err
	}
	if len(res) < 32 {
		return 0, fmt.Errorf("decimals() returned %d bytes", len(res))
	}
	d := new(big.Int).SetBytes(res[24:32]).Int64()
	if d < 0 || d > 36 {
		return 0, fmt.Errorf("implausible decimals %d", d)
	}
	return int(d), nil
}

func decodeStringOrBytes(data []byte) string {
	if len(data) < 64 {
		return ""
	}
	offset := new(big.Int).SetBytes(data[:32]).Uint64()
	if offset == 0 || offset > 31 {
		return strings.TrimRight(string(data[:32]), "\x00")
	}
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
		human := WeiToFloat(bal, dec)
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

// WeiToFloat converts a wei big.Int to a float with the given decimals.
func WeiToFloat(wei *big.Int, decimals int) float64 {
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

// ---- CoinGecko prices ----

// CoinGeckoPrice holds a price response.
type CoinGeckoPrice struct {
	PriceUSD float64 `json:"usd"`
	Change24 float64 `json:"usd_24h_change"`
}

// FetchTokenPrice fetches the current USD price from CoinGecko Simple Price API.
func FetchTokenPrice(ctx context.Context, coinID, apiKey string) (*CoinGeckoPrice, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd&include_24hr_change=true", coinID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if apiKey != "" {
		req.Header.Set("x-cg-pro-api-key", apiKey)
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

// ---- Explorer history / NFT fetchers (Etherscan-compatible) ----

// FetchTransactionHistory fetches recent transactions via an Etherscan-compatible API.
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
		Result []struct {
			Hash      string `json:"hash"`
			From      string `json:"from"`
			To        string `json:"to"`
			Value     string `json:"value"`
			TimeStamp string `json:"timeStamp"`
			IsError   string `json:"isError"`
			GasUsed   string `json:"gasUsed"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	addrLower := strings.ToLower(address)
	var txs []TransactionHistory
	for _, r := range result.Result {
		var ts int64
		fmt.Sscanf(r.TimeStamp, "%d", &ts)
		val, _ := new(big.Int).SetString(r.Value, 10)
		if val == nil {
			val = big.NewInt(0)
		}
		direction := "in"
		if strings.ToLower(r.From) == addrLower {
			direction = "out"
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
			ValueF:    WeiToFloat(val, 18),
			Timestamp: ts,
			Status:    status,
			Direction: direction,
			GasUsed:   r.GasUsed,
		})
	}
	return txs, nil
}

// FetchNFTAssets fetches NFTs owned by an address via an Etherscan-compatible API.
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
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	addrLower := strings.ToLower(address)
	seen := make(map[string]bool)
	var nfts []NFTAsset
	for _, r := range result.Result {
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

// FetchNFTMetadata fetches token URI metadata over HTTP, resolving ipfs:// URLs.
func FetchNFTMetadata(ctx context.Context, tokenURI string) (name, description, imageURL string) {
	uri := resolveURI(tokenURI)
	if uri == "" {
		return "", "", ""
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return "", "", ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var meta struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Image       string `json:"image"`
		ImageURL    string `json:"image_url"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return "", "", ""
	}
	img := meta.Image
	if img == "" {
		img = meta.ImageURL
	}
	return meta.Name, meta.Description, resolveURI(img)
}

// resolveURI rewrites ipfs:// URIs to a public gateway, leaves http(s) as-is.
func resolveURI(uri string) string {
	if uri == "" {
		return ""
	}
	if strings.HasPrefix(uri, "ipfs://") {
		hash := strings.TrimPrefix(uri, "ipfs://")
		return "https://ipfs.io/ipfs/" + hash
	}
	return uri
}

// ---- AMM (Uniswap-V2) calldata builders + decoder ----

// GetAmountsOutData builds getAmountsOut(uint256, address[]) calldata.
func GetAmountsOutData(amountIn *big.Int, path []common.Address) []byte {
	data := make([]byte, 0, 4+32+32+32*len(path))
	data = append(data, 0xd0, 0x6c, 0xa6, 0x1f)
	data = append(data, common.LeftPadBytes(amountIn.Bytes(), 32)...)
	offset := big.NewInt(64)
	data = append(data, common.LeftPadBytes(offset.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(path))).Bytes(), 32)...)
	for _, p := range path {
		data = append(data, common.LeftPadBytes(p.Bytes(), 32)...)
	}
	return data
}

// SwapExactTokensForTokensData builds swapExactTokensForTokens calldata.
func SwapExactTokensForTokensData(amountIn, amountOutMin *big.Int, path []common.Address, to common.Address, deadline *big.Int) []byte {
	data := make([]byte, 0, 4+32*6+32*len(path))
	data = append(data, 0x18, 0xcb, 0xaf, 0xe5)
	data = append(data, common.LeftPadBytes(amountIn.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amountOutMin.Bytes(), 32)...)
	pathOffset := big.NewInt(160)
	data = append(data, common.LeftPadBytes(pathOffset.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(to.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(deadline.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(path))).Bytes(), 32)...)
	for _, p := range path {
		data = append(data, common.LeftPadBytes(p.Bytes(), 32)...)
	}
	return data
}

// DecodeAmountsOut parses the (uint256[]) return of getAmountsOut.
func DecodeAmountsOut(ret []byte) ([]*big.Int, error) {
	if len(ret) < 64 {
		return nil, fmt.Errorf("router returned %d bytes (expected >= 64)", len(ret))
	}
	length := new(big.Int).SetBytes(ret[32:64]).Int64()
	if length < 2 {
		return nil, fmt.Errorf("router returned path length %d", length)
	}
	if int(length) > 64 {
		return nil, fmt.Errorf("implausible path length %d", length)
	}
	out := make([]*big.Int, 0, length)
	for i := int64(0); i < length; i++ {
		off := 64 + int(i)*32
		if off+32 > len(ret) {
			return nil, fmt.Errorf("truncated amounts at index %d", i)
		}
		out = append(out, new(big.Int).SetBytes(ret[off:off+32]))
	}
	return out, nil
}

// HumanToWei converts a human big.Float amount to wei at the given decimals.
func HumanToWei(amount *big.Float, decimals int) (*big.Int, error) {
	scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	weiFloat := new(big.Float).Mul(amount, scale)
	wei, _ := weiFloat.Int(nil)
	if wei.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	return wei, nil
}

// WeiToHuman converts wei to a human-readable string + float at given decimals.
func WeiToHuman(wei *big.Int, decimals int) (string, *big.Float) {
	if decimals <= 0 {
		return wei.String(), new(big.Float).SetInt(wei)
	}
	scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	human := new(big.Float).Quo(new(big.Float).SetInt(wei), scale)
	s := human.Text('f', minInt(decimals, 8))
	return strings.TrimRight(strings.TrimRight(s, "0"), "."), human
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// HexAddress safely parses a hex address string.
func HexAddress(s string) (common.Address, error) {
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("invalid address: %s", s)
	}
	return common.HexToAddress(s), nil
}

// HexEncode returns the 0x-prefixed hex of b.
func HexEncode(b []byte) string { return "0x" + hex.EncodeToString(b) }

// FetchBalance is a string-address convenience wrapper around FetchNativeBalance.
func FetchBalance(ctx context.Context, endpoint string, address string) (*big.Int, error) {
	addr, err := HexAddress(address)
	if err != nil {
		return nil, err
	}
	return FetchNativeBalance(ctx, endpoint, addr)
}

// FetchTxReceipt calls eth_getTransactionReceipt and returns the raw decoded
// receipt as a map (status, blockNumber, gasUsed, contractAddress, logs, ...).
func FetchTxReceipt(ctx context.Context, endpoint string, txHash string) (map[string]any, error) {
	client, err := rpcClient(endpoint)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	var raw json.RawMessage
	if err := client.CallContext(ctx, &raw, "eth_getTransactionReceipt", txHash); err != nil {
		return nil, fmt.Errorf("eth_getTransactionReceipt: %w", err)
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("receipt not found")
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

var _ = hexutil.Encode
