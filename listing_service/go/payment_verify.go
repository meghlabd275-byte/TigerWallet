// Payment verification for token listing fees.
//
// A listing fee is only marked confirmed after the payment transaction is found
// on chain, succeeded, and transferred at least the quoted amount of the
// expected token from the listing owner to the treasury. Nothing is confirmed
// on a timer, and no payment is assumed to have arrived.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const paymentRPCTimeout = 15 * time.Second

// erc20TransferTopic is keccak256("Transfer(address,address,uint256)").
const erc20TransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

// treasuryAddress is the address listing fees must be paid to.
func treasuryAddress() string {
	return strings.ToLower(strings.TrimSpace(getEnv("LISTING_TREASURY_ADDRESS", "")))
}

// chainRPCURL resolves the RPC endpoint for a listing's chain.
func chainRPCURL(chainID int) string {
	return strings.TrimSpace(getEnv(fmt.Sprintf("CHAIN_RPC_%d", chainID), ""))
}

type rpcReceipt struct {
	Status string `json:"status"`
	From   string `json:"from"`
	To     string `json:"to"`
	Logs   []struct {
		Address string   `json:"address"`
		Topics  []string `json:"topics"`
		Data    string   `json:"data"`
	} `json:"logs"`
}

// verifyListingPayment checks a payment transaction against the chain.
//
// It returns the confirmed amount in base units and the token contract that
// emitted the transfer. A transaction that is missing, reverted, or that did not
// move the expected token to the treasury produces an error, so a listing can
// never be activated by a payment that did not happen.
func verifyListingPayment(ctx context.Context, chainID int, txHash, payer, feeToken string) (*big.Int, string, error) {
	if !isTxHash(txHash) {
		return nil, "", fmt.Errorf("invalid transaction hash %q", txHash)
	}

	rpcURL := chainRPCURL(chainID)
	if rpcURL == "" {
		return nil, "", fmt.Errorf("no RPC endpoint configured for chain %d (CHAIN_RPC_%d)", chainID, chainID)
	}
	treasury := treasuryAddress()
	if treasury == "" {
		return nil, "", fmt.Errorf("LISTING_TREASURY_ADDRESS is not configured")
	}

	receipt, err := fetchReceipt(ctx, rpcURL, txHash)
	if err != nil {
		return nil, "", err
	}
	if receipt == nil {
		return nil, "", fmt.Errorf("transaction %s is not yet mined", txHash)
	}
	if receipt.Status != "0x1" {
		return nil, "", fmt.Errorf("transaction %s reverted on chain", txHash)
	}

	payer = strings.ToLower(strings.TrimSpace(payer))
	expectedToken := strings.ToLower(strings.TrimSpace(feeToken))
	if expectedToken != "" && !strings.HasPrefix(expectedToken, "0x") {
		expectedToken = strings.ToLower(strings.TrimSpace(getEnv("LISTING_FEE_TOKEN_"+strings.ToUpper(expectedToken), "")))
	}

	var best *big.Int
	var bestToken string

	for _, entry := range receipt.Logs {
		if len(entry.Topics) < 3 || !strings.EqualFold(entry.Topics[0], erc20TransferTopic) {
			continue
		}

		from := topicToAddress(entry.Topics[1])
		to := topicToAddress(entry.Topics[2])
		if from != payer || to != treasury {
			continue
		}

		contract := strings.ToLower(entry.Address)
		if expectedToken != "" && contract != expectedToken {
			continue
		}

		amount := new(big.Int)
		if _, ok := amount.SetString(strings.TrimPrefix(entry.Data, "0x"), 16); !ok {
			continue
		}
		if best == nil || amount.Cmp(best) > 0 {
			best = amount
			bestToken = contract
		}
	}

	if best == nil {
		return nil, "", fmt.Errorf("transaction %s contains no transfer of the fee token from %s to the treasury", txHash, payer)
	}
	return best, bestToken, nil
}

// fetchReceipt performs eth_getTransactionReceipt and returns nil when the
// transaction is not yet mined.
func fetchReceipt(ctx context.Context, rpcURL, txHash string) (*rpcReceipt, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_getTransactionReceipt",
		"params":  []interface{}{txHash},
	})
	if err != nil {
		return nil, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, paymentRPCTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, rpcURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: paymentRPCTimeout}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("payment verification RPC call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}

	var decoded struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode receipt response: %w", err)
	}
	if decoded.Error != nil {
		return nil, fmt.Errorf("payment verification RPC error: %s", decoded.Error.Message)
	}
	if string(decoded.Result) == "null" {
		return nil, nil
	}

	var receipt rpcReceipt
	if err := json.Unmarshal(decoded.Result, &receipt); err != nil {
		return nil, fmt.Errorf("decode receipt: %w", err)
	}
	return &receipt, nil
}

// feeDecimalsForToken returns the base-unit scale of a fee token.
func feeDecimalsForToken(symbol string) int {
	override := strings.TrimSpace(getEnv("LISTING_FEE_DECIMALS_"+strings.ToUpper(symbol), ""))
	if override != "" {
		if n, err := strconv.Atoi(override); err == nil && n >= 0 && n <= 36 {
			return n
		}
	}
	switch strings.ToUpper(symbol) {
	case "USDT", "USDC", "DAI", "BUSD", "TUSD":
		return 6
	default:
		return 18
	}
}

// feeInBaseUnits converts a decimal fee amount into base units for the fee token.
func feeInBaseUnits(amount float64) *big.Int {
	decimals := feeDecimalsForToken(getEnv("LISTING_FEE_TOKEN", "USDT"))
	scaled, _ := new(big.Float).Mul(big.NewFloat(amount), new(big.Float).SetInt(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))).Int(nil)
	if scaled == nil {
		return big.NewInt(0)
	}
	return scaled
}

// formatBaseUnits renders a base-unit amount as a decimal string.
func formatBaseUnits(value *big.Int) string {
	if value == nil {
		return "0"
	}
	decimals := feeDecimalsForToken(getEnv("LISTING_FEE_TOKEN", "USDT"))
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole, frac := new(big.Int).QuoRem(value, divisor, new(big.Int))
	if frac.Sign() == 0 {
		return whole.String()
	}
	fracStr := fmt.Sprintf("%0*s", decimals, frac.String())
	fracStr = strings.TrimRight(fracStr, "0")
	return whole.String() + "." + fracStr
}

// topicToAddress extracts a 20-byte address from a 32-byte indexed topic.
func topicToAddress(topic string) string {
	trimmed := strings.TrimPrefix(strings.TrimSpace(topic), "0x")
	if len(trimmed) < 40 {
		return ""
	}
	return strings.ToLower(trimmed[len(trimmed)-40:])
}

// isTxHash reports whether s is a 32-byte 0x-prefixed hash.
func isTxHash(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 66 || !strings.HasPrefix(s, "0x") {
		return false
	}
	for _, r := range s[2:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
