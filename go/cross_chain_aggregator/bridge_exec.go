package main

// bridge_exec.go — real cross-chain bridge execution via the LI.FI
// aggregation API plus real EVM transaction signing/broadcast. Fail-closed
// everywhere: without BRIDGE_EXECUTOR_PRIVATE_KEY no transfer is attempted,
// and a transaction hash is only ever recorded after a real broadcast
// confirms. Never fabricates hashes or completion states.

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// lifiBaseURL is the LI.FI aggregation API. LI_FI_API_KEY is optional
// (keyless tier works with an integrator id).
const lifiBaseURL = "https://li.quest/v1"

// nativeTokenAddress is the conventional zero-address placeholder for native
// gas tokens in bridge/DEX aggregator APIs.
const nativeTokenAddress = "0x0000000000000000000000000000000000000000"

// erc20Tokens is the registry of supported ERC-20 tokens per chain:
// chainID -> symbol -> (address, decimals). Public contract constants.
var erc20Tokens = map[int64]map[string][2]any{
	1: {
		"USDC": {"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", 6},
		"USDT": {"0xdAC17F958D2ee523a2206206994597C13D831ec7", 6},
		"DAI":  {"0x6B175474E89094C44Da98b954EedeAC495271d0F", 18},
	},
	137: {
		"USDC": {"0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", 6},
		"USDT": {"0xc2132D05D31c914a87C6611C10748AEb04B58e8F", 6},
		"DAI":  {"0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063", 18},
	},
	42161: {
		"USDC": {"0xaf88d065e77c8cC2239327C5EDb3A432268e5831", 6},
		"USDT": {"0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9", 6},
		"DAI":  {"0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1", 18},
	},
	10: {
		"USDC": {"0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85", 6},
		"USDT": {"0x94b008aA00579c1307B0EF2c499aD98a8ce58e58", 6},
		"DAI":  {"0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1", 18},
	},
	56: {
		"USDC": {"0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", 18},
		"USDT": {"0x55d398326f99059fF775485246999027B3197955", 18},
		"DAI":  {"0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3", 18},
	},
	8453: {
		"USDC": {"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", 6},
		"DAI":  {"0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb", 18},
	},
}

// defaultRPCs are public RPC fallbacks; operators should override with
// RPC_URL_<CHAINNAME> env vars for production reliability.
var defaultRPCs = map[string]string{
	"ethereum":  "https://eth.llamarpc.com",
	"polygon":   "https://polygon-rpc.com",
	"avalanche": "https://api.avax.network/ext/bc/C/rpc",
	"arbitrum":  "https://arb1.arbitrum.io/rpc",
	"optimism":  "https://mainnet.optimism.io",
	"bsc":       "https://bsc-dataseed.binance.org",
	"base":      "https://mainnet.base.org",
	"fantom":    "https://rpc.ftm.tools",
}

// rpcForChain resolves the RPC endpoint for a chain: env override first,
// then the public default. Fail-closed (error) when neither exists.
func rpcForChain(chainName string) (string, error) {
	envKey := "RPC_URL_" + strings.ToUpper(strings.ReplaceAll(chainName, "-", "_"))
	if v := os.Getenv(envKey); v != "" {
		return v, nil
	}
	if v, ok := defaultRPCs[chainName]; ok {
		return v, nil
	}
	return "", fmt.Errorf("no RPC configured for chain %q (set %s)", chainName, envKey)
}

// resolveTokenAddress returns the on-chain token address and decimals for a
// symbol on a chain. Native tokens resolve to the zero address. Fail-closed:
// unknown ERC-20 tokens are an error, never guessed.
func resolveTokenAddress(chainName, symbol string, chainID int64) (string, int, error) {
	symbol = strings.ToUpper(symbol)
	if native, ok := nativeSymbolForChain(chainName); ok && native == symbol {
		return nativeTokenAddress, 18, nil
	}
	if perChain, ok := erc20Tokens[chainID]; ok {
		if entry, ok := perChain[symbol]; ok {
			addr, _ := entry[0].(string)
			dec, _ := entry[1].(int)
			return addr, dec, nil
		}
	}
	return "", 0, fmt.Errorf("token %s not registered on chain %s", symbol, chainName)
}

func nativeSymbolForChain(chainName string) (string, bool) {
	switch chainName {
	case "ethereum", "arbitrum", "optimism", "base":
		return "ETH", true
	case "polygon":
		return "MATIC", true
	case "avalanche":
		return "AVAX", true
	case "bsc":
		return "BNB", true
	case "fantom":
		return "FTM", true
	}
	return "", false
}

// lifiQuote is the parsed LI.FI quote carrying a real unsigned transaction.
type lifiQuote struct {
	TxTo    string
	TxData  string
	TxValue *big.Int
	Tool    string
	EstTime int // seconds
}

// fetchLiFiQuote requests a real bridge quote+transaction from LI.FI.
func fetchLiFiQuote(ctx context.Context, fromChainID, toChainID int64, fromToken, toToken, fromAddress, toAddress string, fromAmountWei *big.Int, slippage float64) (*lifiQuote, error) {
	if slippage <= 0 || slippage > 20 {
		slippage = 0.5 // percent
	}
	q := url.Values{}
	q.Set("fromChain", fmt.Sprint(fromChainID))
	q.Set("toChain", fmt.Sprint(toChainID))
	q.Set("fromToken", fromToken)
	q.Set("toToken", toToken)
	q.Set("fromAmount", fromAmountWei.String())
	q.Set("fromAddress", fromAddress)
	q.Set("toAddress", toAddress)
	q.Set("slippage", fmt.Sprintf("%.4f", slippage/100.0))
	q.Set("integrator", "tigerwallet")
	if key := os.Getenv("LI_FI_API_KEY"); key != "" {
		q.Set("apiKey", key)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lifiBaseURL+"/quote?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&e)
		return nil, fmt.Errorf("lifi HTTP %d: %s", resp.StatusCode, e.Message)
	}
	var parsed struct {
		Tool     string `json:"tool"`
		Estimate struct {
			ExecutionDuration int `json:"executionDuration"`
		} `json:"estimate"`
		TransactionRequest struct {
			To    string `json:"to"`
			Data  string `json:"data"`
			Value string `json:"value"`
		} `json:"transactionRequest"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if parsed.TransactionRequest.To == "" || parsed.TransactionRequest.Data == "" {
		return nil, fmt.Errorf("lifi returned no executable transaction")
	}
	value := new(big.Int)
	if parsed.TransactionRequest.Value != "" {
		if _, ok := value.SetString(strings.TrimPrefix(parsed.TransactionRequest.Value, "0x"), 16); !ok {
			value.SetString(parsed.TransactionRequest.Value, 10)
		}
	}
	return &lifiQuote{
		TxTo:    parsed.TransactionRequest.To,
		TxData:  parsed.TransactionRequest.Data,
		TxValue: value,
		Tool:    parsed.Tool,
		EstTime: parsed.Estimate.ExecutionDuration,
	}, nil
}

// signAndBroadcastTx signs and broadcasts a real EVM transaction; returns the
// real tx hash only after the node accepted it.
func signAndBroadcastTx(ctx context.Context, rpcURL, privKeyHex, to string, value *big.Int, dataHex string) (string, error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(privKeyHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("invalid executor key: %w", err)
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return "", err
	}
	defer client.Close()

	from := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("chain id: %w", err)
	}
	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return "", fmt.Errorf("gas tip: %w", err)
	}
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("header: %w", err)
	}
	data := common.FromHex(dataHex)
	toAddr := common.HexToAddress(to)
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &toAddr, Value: value, Data: data})
	if err != nil {
		return "", fmt.Errorf("estimate gas: %w", err)
	}
	gasLimit = gasLimit * 12 / 10 // 20% headroom
	feeCap := new(big.Int).Add(header.BaseFee.Mul(header.BaseFee, big.NewInt(2)), tip)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        ptrAddr(common.HexToAddress(to)),
		Value:     value,
		Gas:       gasLimit,
		GasTipCap: tip,
		GasFeeCap: feeCap,
		Data:      data,
	})
	signed, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("broadcast: %w", err)
	}
	return signed.Hash().Hex(), nil
}

func ptrAddr(a common.Address) *common.Address { return &a }

// lifiBridgeStatus polls LI.FI for the cross-chain status of a source tx.
func lifiBridgeStatus(ctx context.Context, txHash string, fromChainID, toChainID int64) (status string, receivingTx string, err error) {
	q := url.Values{}
	q.Set("txHash", txHash)
	q.Set("fromChain", fmt.Sprint(fromChainID))
	q.Set("toChain", fmt.Sprint(toChainID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lifiBaseURL+"/status?"+q.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("lifi status HTTP %d", resp.StatusCode)
	}
	var parsed struct {
		Status    string `json:"status"` // NOT_FOUND, PENDING, DONE, FAILED
		Substatus string `json:"substatus"`
		Receiving struct {
			TxHash string `json:"txHash"`
		} `json:"receiving"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", "", err
	}
	return parsed.Status, parsed.Receiving.TxHash, nil
}
