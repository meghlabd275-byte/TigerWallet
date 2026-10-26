// onchain.go — real payouts for the Protection Fund.
//
// Approved claims are paid by signing and broadcasting a real transaction from
// the configured payout wallet: either a native-coin transfer or an ERC-20
// transfer. Nothing here invents a transaction hash. When the RPC endpoint or
// the payout key is missing, Payout returns an error and the claim stays
// approved, which is the safe state: an unpaid claim can be retried, whereas a
// claim marked paid without a transaction cannot be reconciled.

package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// errRPCNotConfigured is returned when no chain endpoint is set.
	errRPCNotConfigured = errors.New("chain RPC endpoint is not configured (PROTECTION_FUND_RPC_URL)")
	// errPayoutKeyNotConfigured is returned when no payout key is set.
	errPayoutKeyNotConfigured = errors.New("payout wallet key is not configured (PROTECTION_FUND_PRIVATE_KEY)")
	// errInvalidPayoutAddress is returned for a malformed destination address.
	errInvalidPayoutAddress = errors.New("invalid payout address")
	// errInvalidPayoutAmount is returned for a zero or negative payout.
	errInvalidPayoutAmount = errors.New("payout amount must be positive")
)

// receiptTimeout bounds how long a payout waits for confirmation.
const receiptTimeout = 3 * time.Minute

// PayoutClient signs and broadcasts payouts from the fund wallet.
type PayoutClient struct {
	client   *ethclient.Client
	key      *ecdsa.PrivateKey
	token    common.Address // zero address means pay in the native coin
	chainID  *big.Int
	decimals uint8
}

// NewPayoutClient builds a payout client from the environment. It fails when the
// RPC endpoint or the payout key is absent, so a caller cannot accidentally
// proceed without the ability to pay.
func NewPayoutClient(ctx context.Context) (*PayoutClient, error) {
	rpcURL := strings.TrimSpace(os.Getenv("PROTECTION_FUND_RPC_URL"))
	if rpcURL == "" {
		return nil, errRPCNotConfigured
	}

	keyHex := strings.TrimPrefix(strings.TrimSpace(os.Getenv("PROTECTION_FUND_PRIVATE_KEY")), "0x")
	if keyHex == "" {
		return nil, errPayoutKeyNotConfigured
	}
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("parse payout key: %w", err)
	}

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("connect to chain: %w", err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("read chain id: %w", err)
	}

	decimals := uint8(6)
	if raw := strings.TrimSpace(os.Getenv("PROTECTION_FUND_PAYOUT_DECIMALS")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 8)
		if err != nil || parsed > 36 {
			return nil, fmt.Errorf("invalid PROTECTION_FUND_PAYOUT_DECIMALS %q", raw)
		}
		decimals = uint8(parsed)
	}

	var token common.Address
	if raw := strings.TrimSpace(os.Getenv("PROTECTION_FUND_TOKEN_ADDRESS")); raw != "" {
		addr, err := parseAddress(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid PROTECTION_FUND_TOKEN_ADDRESS: %w", err)
		}
		token = addr
	}

	return &PayoutClient{client: client, key: key, token: token, chainID: chainID, decimals: decimals}, nil
}

// Address is the wallet payouts are sent from.
func (p *PayoutClient) Address() common.Address {
	return crypto.PubkeyToAddress(p.key.PublicKey)
}

// Payout sends amount (whole token units, e.g. USDC) to the destination and
// returns the confirmed transaction hash.
func (p *PayoutClient) Payout(ctx context.Context, to common.Address, amount float64) (string, error) {
	if to == (common.Address{}) {
		return "", errInvalidPayoutAddress
	}
	value, err := decimalToBaseUnits(amount, p.decimals)
	if err != nil {
		return "", err
	}

	if p.token == (common.Address{}) {
		return p.send(ctx, to, value, nil)
	}

	data := append(selector("transfer(address,uint256)"), padAddress32(to)...)
	data = append(data, padLeft32(value)...)
	return p.send(ctx, p.token, big.NewInt(0), data)
}

// send builds, signs, broadcasts and awaits a transaction.
func (p *PayoutClient) send(ctx context.Context, to common.Address, value *big.Int, data []byte) (string, error) {
	from := p.Address()

	nonce, err := p.client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}
	gasPrice, err := p.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("suggest gas price: %w", err)
	}

	call := ethereum.CallMsg{From: from, To: &to, Value: value, Data: data}
	gasLimit, err := p.client.EstimateGas(ctx, call)
	if err != nil {
		return "", fmt.Errorf("estimate gas: %w", err)
	}

	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(p.chainID), p.key)
	if err != nil {
		return "", fmt.Errorf("sign payout: %w", err)
	}
	if err := p.client.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("broadcast payout: %w", err)
	}

	hash := signed.Hash().Hex()
	receipt, err := p.waitForReceipt(ctx, signed.Hash())
	if err != nil {
		// The hash is returned so the caller can record what was broadcast and
		// reconcile later rather than losing track of an in-flight payout.
		return hash, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return hash, fmt.Errorf("payout transaction %s reverted", hash)
	}
	return hash, nil
}

// waitForReceipt polls until the transaction is mined or the timeout expires.
func (p *PayoutClient) waitForReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	deadline := time.Now().Add(receiptTimeout)
	for {
		receipt, err := p.client.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if !errors.Is(err, ethereum.NotFound) {
			return nil, fmt.Errorf("read payout receipt: %w", err)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("payout %s was not mined within %s", hash.Hex(), receiptTimeout)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// decimalToBaseUnits converts a whole-token amount into base units.
//
// The conversion goes through the amount's shortest round-tripping decimal
// form and then uses integer arithmetic. Scaling the float directly loses
// precision: 0.000001 * 1e6 evaluates to 0.9999999999999999 and truncates to
// zero, which would silently drop a legitimate payout.
func decimalToBaseUnits(amount float64, decimals uint8) (*big.Int, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		return nil, errInvalidPayoutAmount
	}
	// 'f' with precision -1 yields the shortest decimal that round-trips, and
	// never uses exponent notation.
	return decimalStringToBaseUnits(strconv.FormatFloat(amount, 'f', -1, 64), decimals)
}

// decimalStringToBaseUnits converts a plain decimal string such as "1234.56"
// into base units using exact integer arithmetic.
func decimalStringToBaseUnits(raw string, decimals uint8) (*big.Int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errInvalidPayoutAmount
	}

	// A leading sign is accepted only for zero, which is rejected below; this
	// keeps "-1" and "+1" out without a separate parse.
	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return nil, errInvalidPayoutAmount
	}

	whole, fraction, hasFraction := strings.Cut(value, ".")
	if whole == "" {
		whole = "0"
	}
	if hasFraction && fraction == "" {
		return nil, errInvalidPayoutAmount
	}
	if !isDigits(whole) || (hasFraction && !isDigits(fraction)) {
		return nil, errInvalidPayoutAmount
	}

	// Pad or truncate the fraction to the token's precision. Truncation never
	// rounds a payout up.
	if uint(len(fraction)) < uint(decimals) {
		fraction += strings.Repeat("0", int(decimals)-len(fraction))
	} else if uint(len(fraction)) > uint(decimals) {
		fraction = fraction[:decimals]
	}

	scaled := whole + fraction
	units, ok := new(big.Int).SetString(scaled, 10)
	if !ok {
		return nil, errInvalidPayoutAmount
	}
	if units.Sign() <= 0 {
		return nil, errInvalidPayoutAmount
	}
	return units, nil
}

// isDigits reports whether every rune in s is an ASCII digit and s is non-empty.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// parseAddress validates a 0x-prefixed 20-byte hex address.
func parseAddress(raw string) (common.Address, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "0x") || len(trimmed) != 42 {
		return common.Address{}, fmt.Errorf("%w: %q", errInvalidPayoutAddress, raw)
	}
	if !common.IsHexAddress(trimmed) {
		return common.Address{}, fmt.Errorf("%w: %q", errInvalidPayoutAddress, raw)
	}
	return common.HexToAddress(trimmed), nil
}

// selector returns the four-byte function selector for a signature.
func selector(signature string) []byte {
	return crypto.Keccak256([]byte(signature))[:4]
}

// padLeft32 left-pads a value to a 32-byte ABI word.
func padLeft32(value *big.Int) []byte {
	word := make([]byte, 32)
	bytes := value.Bytes()
	if len(bytes) > 32 {
		bytes = bytes[len(bytes)-32:]
	}
	copy(word[32-len(bytes):], bytes)
	return word
}

// padAddress32 left-pads an address to a 32-byte ABI word.
func padAddress32(addr common.Address) []byte {
	word := make([]byte, 32)
	copy(word[12:], addr.Bytes())
	return word
}
