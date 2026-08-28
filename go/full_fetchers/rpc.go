// TigerWallet Full Fetchers - real data access layer.
//
// Stdlib-only JSON-RPC 2.0 client for EVM nodes plus helpers for the public
// market-data APIs used by the fetchers (CoinGecko, DefiLlama, Lido, OpenSea,
// LI.FI). No mocks: every function either returns live data or an error.
//
// Configuration (all optional, sane public defaults):
//   FULL_FETCHERS_RPC_<chainID>   RPC endpoint override for a chain
//   EVM_RPC_URL                   default RPC endpoint for Ethereum mainnet
//   COINGECKO_API_URL / COINGECKO_API_KEY   price data
//   OPENSEA_API_KEY               NFT floor prices (required for NFT fetcher)
//   ETHERSCAN_API_KEY             contract source verification (optional)
//   LIFI_API_URL                  cross-chain quotes (default https://li.quest/v1)

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// rpcTimeout bounds every upstream call so fetchers fail fast instead of
// hanging under load.
const rpcTimeout = 10 * time.Second

var httpClient = &http.Client{Timeout: rpcTimeout}

// -----------------------------------------------------------------------------
// JSON-RPC 2.0 client
// -----------------------------------------------------------------------------

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	} `json:"error"`
}

func rpcCall(ctx context.Context, endpoint, method string, params []interface{}, out interface{}) error {
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	var r rpcResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("invalid JSON-RPC response: %w", err)
	}
	if r.Error != nil {
		return &rpcError{Code: r.Error.Code, Message: r.Error.Message, Data: r.Error.Data}
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(r.Result, out)
}

// rpcError carries the node's revert payload so callers can decode revert
// reasons from real simulations.
type rpcError struct {
	Code    int
	Message string
	Data    string
}

func (e *rpcError) Error() string { return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message) }

// -----------------------------------------------------------------------------
// Endpoint resolution
// -----------------------------------------------------------------------------

// chainRPCEndpoint resolves the RPC URL for a chain: per-chain env override,
// EVM_RPC_URL for mainnet, otherwise well-known public endpoints.
func chainRPCEndpoint(chainID ChainID) string {
	if v := os.Getenv(fmt.Sprintf("FULL_FETCHERS_RPC_%d", uint64(chainID))); v != "" {
		return v
	}
	switch uint64(chainID) {
	case 1:
		if v := os.Getenv("EVM_RPC_URL"); v != "" {
			return v
		}
		return "https://eth.llamarpc.com"
	case 56:
		return "https://bsc-dataseed.binance.org"
	case 137:
		return "https://polygon-rpc.com"
	case 42161:
		return "https://arb1.arbitrum.io/rpc"
	case 10:
		return "https://mainnet.optimism.io"
	case 8453:
		return "https://mainnet.base.org"
	case 43114:
		return "https://api.avax.network/ext/bc/C/rpc"
	}
	return ""
}

// -----------------------------------------------------------------------------
// ABI helpers (minimal, allocation-light)
// -----------------------------------------------------------------------------

func pad32(b []byte) []byte {
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func abiAddress(addr string) []byte {
	a := strings.TrimPrefix(strings.ToLower(addr), "0x")
	raw, _ := hexDecode(a)
	return pad32(raw)
}

func abiUint(n *big.Int) []byte { return pad32(n.Bytes()) }

func hexDecode(s string) ([]byte, error) {
	if len(s)%2 == 1 {
		s = "0" + s
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		hi, ok1 := hexVal(s[2*i])
		lo, ok2 := hexVal(s[2*i+1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("invalid hex")
		}
		out[i] = hi<<4 | lo
	}
	return out, nil
}

func hexVal(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// hexQuantity encodes a uint64 as an EVM hex quantity.
func hexQuantity(n uint64) string { return "0x" + strconv.FormatUint(n, 16) }

// parseHexBig decodes a 0x-prefixed hex quantity into *big.Int.
func parseHexBig(s string) (*big.Int, error) {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return big.NewInt(0), nil
	}
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex quantity")
	}
	return n, nil
}

// -----------------------------------------------------------------------------
// Common eth_* calls
// -----------------------------------------------------------------------------

func ethBlockNumber(ctx context.Context, endpoint string) (uint64, error) {
	var res string
	if err := rpcCall(ctx, endpoint, "eth_blockNumber", []interface{}{}, &res); err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimPrefix(res, "0x"), 16, 64)
}

func ethChainID(ctx context.Context, endpoint string) (uint64, error) {
	var res string
	if err := rpcCall(ctx, endpoint, "eth_chainId", []interface{}{}, &res); err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimPrefix(res, "0x"), 16, 64)
}

func ethGasPrice(ctx context.Context, endpoint string) (*big.Int, error) {
	var res string
	if err := rpcCall(ctx, endpoint, "eth_gasPrice", []interface{}{}, &res); err != nil {
		return nil, err
	}
	return parseHexBig(res)
}

func ethMaxPriorityFee(ctx context.Context, endpoint string) (*big.Int, error) {
	var res string
	if err := rpcCall(ctx, endpoint, "eth_maxPriorityFeePerGas", []interface{}{}, &res); err != nil {
		return nil, err
	}
	return parseHexBig(res)
}

func ethGetCode(ctx context.Context, endpoint, addr string) (string, error) {
	var res string
	err := rpcCall(ctx, endpoint, "eth_getCode", []interface{}{addr, "latest"}, &res)
	return res, err
}

func ethGetStorageAt(ctx context.Context, endpoint, addr, slot string) (string, error) {
	var res string
	err := rpcCall(ctx, endpoint, "eth_getStorageAt", []interface{}{addr, slot, "latest"}, &res)
	return res, err
}

func ethTxCount(ctx context.Context, endpoint, addr string) (uint64, error) {
	var res string
	if err := rpcCall(ctx, endpoint, "eth_getTransactionCount", []interface{}{addr, "latest"}, &res); err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimPrefix(res, "0x"), 16, 64)
}

type callMsg struct {
	From  string `json:"from,omitempty"`
	To    string `json:"to"`
	Data  string `json:"data,omitempty"`
	Value string `json:"value,omitempty"`
}

func ethCall(ctx context.Context, endpoint string, msg callMsg, block string) (string, error) {
	var res string
	err := rpcCall(ctx, endpoint, "eth_call", []interface{}{msg, block}, &res)
	return res, err
}

func ethEstimateGas(ctx context.Context, endpoint string, msg callMsg) (uint64, error) {
	var res string
	if err := rpcCall(ctx, endpoint, "eth_estimateGas", []interface{}{msg}, &res); err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimPrefix(res, "0x"), 16, 64)
}

// ethFeeHistory returns base fees per block for the last n blocks.
func ethFeeHistory(ctx context.Context, endpoint string, n int) ([]*big.Int, error) {
	var res struct {
		BaseFeePerGas []string `json:"baseFeePerGas"`
	}
	if err := rpcCall(ctx, endpoint, "eth_feeHistory", []interface{}{hexQuantity(uint64(n)), "latest", []int{25, 50, 75}}, &res); err != nil {
		return nil, err
	}
	fees := make([]*big.Int, 0, len(res.BaseFeePerGas))
	for _, f := range res.BaseFeePerGas {
		bf, err := parseHexBig(f)
		if err != nil {
			return nil, err
		}
		fees = append(fees, bf)
	}
	return fees, nil
}

type rpcLog struct {
	Address  string   `json:"address"`
	Topics   []string `json:"topics"`
	Data     string   `json:"data"`
	TxHash   string   `json:"transactionHash"`
	BlockNum string   `json:"blockNumber"`
	LogIndex string   `json:"logIndex"`
}

type logFilter struct {
	FromBlock string   `json:"fromBlock"`
	ToBlock   string   `json:"toBlock"`
	Address   string   `json:"address,omitempty"`
	Topics    []string `json:"topics,omitempty"`
}

func ethGetLogs(ctx context.Context, endpoint string, filter logFilter) ([]rpcLog, error) {
	var logs []rpcLog
	if err := rpcCall(ctx, endpoint, "eth_getLogs", []interface{}{filter}, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// rpcTx is a decoded full transaction object (used when fullTx=true).
type rpcTx struct {
	Hash  string `json:"hash"`
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
	Input string `json:"input"`
}

// rpcBlock keeps transactions raw so both hash-only (fullTx=false) and full
// transaction (fullTx=true) responses decode correctly.
type rpcBlock struct {
	Number        string            `json:"number"`
	GasUsed       string            `json:"gasUsed"`
	GasLimit      string            `json:"gasLimit"`
	BaseFeePerGas string            `json:"baseFeePerGas"`
	Timestamp     string            `json:"timestamp"`
	Transactions  []json.RawMessage `json:"transactions"`
}

// FullTransactions decodes the raw transaction entries into rpcTx objects.
// Call only on blocks fetched with fullTx=true.
func (b *rpcBlock) FullTransactions() []rpcTx {
	txs := make([]rpcTx, 0, len(b.Transactions))
	for _, raw := range b.Transactions {
		var tx rpcTx
		if err := json.Unmarshal(raw, &tx); err == nil && tx.Hash != "" {
			txs = append(txs, tx)
		}
	}
	return txs
}

func ethGetBlock(ctx context.Context, endpoint, tag string, fullTx bool) (*rpcBlock, error) {
	var b rpcBlock
	if err := rpcCall(ctx, endpoint, "eth_getBlockByNumber", []interface{}{tag, fullTx}, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// -----------------------------------------------------------------------------
// ERC-20 / DEX specific calls
// -----------------------------------------------------------------------------

// 4-byte selectors used by the fetchers.
var (
	selERC20Name        = []byte{0x06, 0xfd, 0xde, 0x03} // name()
	selERC20Symbol      = []byte{0x95, 0xd8, 0x9b, 0x41} // symbol()
	selERC20Decimals    = []byte{0x31, 0x3c, 0xe5, 0x67} // decimals()
	selERC20TotalSupply = []byte{0x18, 0x16, 0x0d, 0xdd} // totalSupply()
	selV2GetReserves    = []byte{0x09, 0x02, 0xf1, 0xac} // getReserves()
	selV2FactoryGetPair = []byte{0xe6, 0xa4, 0x39, 0x05} // getPair(address,address)
	selV2Token0         = []byte{0x0d, 0xfe, 0x16, 0x81} // token0()
	selV3Slot0          = []byte{0x38, 0x50, 0xc7, 0xbd} // slot0()
)

func dataHex(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 2+len(b)*2)
	copy(out, "0x")
	for i, v := range b {
		out[2+2*i] = digits[v>>4]
		out[3+2*i] = digits[v&0x0f]
	}
	return string(out)
}

// decodeABIString handles both dynamic-string and bytes32 returns.
func decodeABIString(data string) string {
	data = strings.TrimPrefix(data, "0x")
	raw, err := hexDecode(data)
	if err != nil || len(raw) == 0 {
		return ""
	}
	if len(raw) == 32 {
		// bytes32
		return strings.TrimRight(string(raw), "\x00")
	}
	if len(raw) >= 64 {
		length := new(big.Int).SetBytes(raw[32:64]).Int64()
		if length > 0 && int64(len(raw)) >= 64+length {
			return string(raw[64 : 64+length])
		}
	}
	return ""
}

func fetchERC20Metadata(ctx context.Context, endpoint, token string) (name, symbol string, decimals uint8, totalSupply *big.Int, err error) {
	call := func(sel []byte) (string, error) {
		return ethCall(ctx, endpoint, callMsg{To: token, Data: dataHex(sel)}, "latest")
	}
	if name, err = call(selERC20Name); err != nil {
		return
	}
	name = decodeABIString(name)
	if symbol, err = call(selERC20Symbol); err != nil {
		return
	}
	symbol = decodeABIString(symbol)
	var decHex string
	if decHex, err = call(selERC20Decimals); err != nil {
		return
	}
	dec, err2 := parseHexBig(decHex)
	if err2 != nil {
		err = err2
		return
	}
	decimals = uint8(dec.Uint64())
	var tsHex string
	if tsHex, err = call(selERC20TotalSupply); err != nil {
		return
	}
	totalSupply, err = parseHexBig(tsHex)
	return
}

// fetchV2Reserves returns (reserve0, reserve1) for a Uniswap V2-style pair.
func fetchV2Reserves(ctx context.Context, endpoint, pair string) (*big.Int, *big.Int, error) {
	res, err := ethCall(ctx, endpoint, callMsg{To: pair, Data: dataHex(selV2GetReserves)}, "latest")
	if err != nil {
		return nil, nil, err
	}
	raw, err := hexDecode(strings.TrimPrefix(res, "0x"))
	if err != nil || len(raw) < 64 {
		return nil, nil, fmt.Errorf("bad getReserves response")
	}
	return new(big.Int).SetBytes(raw[0:32]), new(big.Int).SetBytes(raw[32:64]), nil
}

// resolveV2Pair asks a UniV2 factory for the pair address of (a,b).
func resolveV2Pair(ctx context.Context, endpoint, factory, a, b string) (string, error) {
	data := append(append(selV2FactoryGetPair, abiAddress(a)...), abiAddress(b)...)
	res, err := ethCall(ctx, endpoint, callMsg{To: factory, Data: dataHex(data)}, "latest")
	if err != nil {
		return "", err
	}
	raw, err := hexDecode(strings.TrimPrefix(res, "0x"))
	if err != nil || len(raw) < 32 {
		return "", fmt.Errorf("bad getPair response")
	}
	if new(big.Int).SetBytes(raw).Sign() == 0 {
		return "", fmt.Errorf("pair does not exist")
	}
	return "0x" + strings.ToLower(fmt.Sprintf("%x", raw[12:32])), nil
}

// fetchPairToken0 returns the token0 address of a UniV2-style pair.
func fetchPairToken0(ctx context.Context, endpoint, pair string) (string, error) {
	res, err := ethCall(ctx, endpoint, callMsg{To: pair, Data: dataHex(selV2Token0)}, "latest")
	if err != nil {
		return "", err
	}
	raw, err := hexDecode(strings.TrimPrefix(res, "0x"))
	if err != nil || len(raw) < 32 {
		return "", fmt.Errorf("bad token0 response")
	}
	return "0x" + fmt.Sprintf("%x", raw[12:32]), nil
}

// fetchV3Price reads a UniV3 pool slot0 and returns the sqrtPriceX96.
func fetchV3SqrtPriceX96(ctx context.Context, endpoint, pool string) (*big.Int, error) {
	res, err := ethCall(ctx, endpoint, callMsg{To: pool, Data: dataHex(selV3Slot0)}, "latest")
	if err != nil {
		return nil, err
	}
	raw, err := hexDecode(strings.TrimPrefix(res, "0x"))
	if err != nil || len(raw) < 32 {
		return nil, fmt.Errorf("bad slot0 response")
	}
	return new(big.Int).SetBytes(raw[0:32]), nil
}

// quoteExactInputSingle calls the Uniswap V3 Quoter on mainnet.
// Quoter: 0xb27308f9F90D607463bb33eA1BeBb41C27CE5AB6
// selector quoteExactInputSingle(address,address,uint24,uint256,uint160) = 0xf7729d43
func quoteExactInputSingle(ctx context.Context, endpoint, tokenIn, tokenOut string, fee uint64, amountIn *big.Int) (*big.Int, error) {
	data := []byte{0xf7, 0x72, 0x9d, 0x43}
	data = append(data, abiAddress(tokenIn)...)
	data = append(data, abiAddress(tokenOut)...)
	data = append(data, abiUint(big.NewInt(int64(fee)))...)
	data = append(data, abiUint(amountIn)...)
	data = append(data, abiUint(big.NewInt(0))...) // sqrtPriceLimitX96 = 0
	res, err := ethCall(ctx, endpoint, callMsg{To: "0xb27308f9F90D607463bb33eA1BeBb41C27CE5AB6", Data: dataHex(data)}, "latest")
	if err != nil {
		return nil, err
	}
	return parseHexBig(res)
}

// -----------------------------------------------------------------------------
// HTTP JSON helper for public market APIs
// -----------------------------------------------------------------------------

func httpGetJSON(ctx context.Context, url string, headers map[string]string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(out)
}

func coinGeckoBase() string {
	if v := os.Getenv("COINGECKO_API_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://api.coingecko.com/api/v3"
}

func coinGeckoHeaders() map[string]string {
	h := map[string]string{}
	if key := os.Getenv("COINGECKO_API_KEY"); key != "" {
		// Pro keys use x-cg-pro-api-key; demo keys use x-cg-demo-api-key.
		if strings.Contains(coinGeckoBase(), "pro-api") {
			h["x-cg-pro-api-key"] = key
		} else {
			h["x-cg-demo-api-key"] = key
		}
	}
	return h
}

func weiToGwei(wei *big.Int) float64 {
	f := new(big.Float).SetInt(wei)
	g := new(big.Float).Quo(f, big.NewFloat(1e9))
	v, _ := g.Float64()
	return v
}
