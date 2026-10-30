package main

// onchain.go — REAL on-chain execution for stake / unstake / claim.
//
// The service is an operator-side dashboard: the operator wallet calls a
// staking contract on the user's behalf using the standard Synthetix-style
// StakingRewards interface (stake(uint256), withdraw(uint256), getReward()).
// Those selectors are fixed by that interface and are not configurable.
//
// Fail-closed: when STAKING_RPC_URL / STAKING_OPERATOR_PRIVATE_KEY are not
// configured, nothing is broadcast and no transaction hash is produced. The
// row is kept in "pending_broadcast" so an operator can retry once the chain
// is wired up. A hash is only ever recorded after the node accepts the
// transaction.

import (
	"context"
	"crypto/ecdsa"
	"fmt"
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

// Selectors for the standard StakingRewards interface.
var (
	selStake   = crypto.Keccak256([]byte("stake(uint256)"))[:4]
	selWithdraw = crypto.Keccak256([]byte("withdraw(uint256)"))[:4]
	selGetReward = crypto.Keccak256([]byte("getReward()"))[:4]
)

// OnChainStaking broadcasts staking operations from the operator wallet.
type OnChainStaking struct {
	client   *ethclient.Client
	signer   *ecdsa.PrivateKey
	fromAddr common.Address
	chainID  *big.Int
	// contract is the staking contract address on this chain.
	contract common.Address
}

// NewOnChainStakingFromEnv returns (nil, nil) when the integration is not
// configured, and an error only when configuration is present but invalid.
//
// Env: STAKING_RPC_URL, STAKING_OPERATOR_PRIVATE_KEY, STAKING_CONTRACT_ADDRESS.
func NewOnChainStakingFromEnv() (*OnChainStaking, error) {
	rpcURL := strings.TrimSpace(os.Getenv("STAKING_RPC_URL"))
	pkHex := strings.TrimSpace(os.Getenv("STAKING_OPERATOR_PRIVATE_KEY"))
	contractHex := strings.TrimSpace(os.Getenv("STAKING_CONTRACT_ADDRESS"))

	if rpcURL == "" && pkHex == "" && contractHex == "" {
		return nil, nil
	}
	if rpcURL == "" || pkHex == "" || contractHex == "" {
		return nil, fmt.Errorf("STAKING_RPC_URL, STAKING_OPERATOR_PRIVATE_KEY and STAKING_CONTRACT_ADDRESS must all be set")
	}
	if !common.IsHexAddress(contractHex) {
		return nil, fmt.Errorf("STAKING_CONTRACT_ADDRESS is not a valid address")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("ethclient dial: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
	pk, err := crypto.HexToECDSA(strings.TrimPrefix(pkHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("operator private key: %w", err)
	}

	return &OnChainStaking{
		client:   client,
		signer:   pk,
		fromAddr: crypto.PubkeyToAddress(pk.PublicKey),
		chainID:  chainID,
		contract: common.HexToAddress(contractHex),
	}, nil
}

// ErrNotConfigured is returned when the on-chain integration is absent.
var ErrNotConfigured = fmt.Errorf("on-chain staking is not configured")

// Stake stakes amount (in the token's smallest unit) on behalf of the caller.
func (o *OnChainStaking) Stake(ctx context.Context, amount *big.Int) (string, error) {
	data := append(append([]byte{}, selStake...), common.LeftPadBytes(amount.Bytes(), 32)...)
	return o.send(ctx, data)
}

// Withdraw withdraws amount from the staking contract.
func (o *OnChainStaking) Withdraw(ctx context.Context, amount *big.Int) (string, error) {
	data := append(append([]byte{}, selWithdraw...), common.LeftPadBytes(amount.Bytes(), 32)...)
	return o.send(ctx, data)
}

// ClaimRewards calls getReward().
func (o *OnChainStaking) ClaimRewards(ctx context.Context) (string, error) {
	return o.send(ctx, append([]byte{}, selGetReward...))
}

// send builds, signs and broadcasts a call to the staking contract, returning
// the hash the node accepted. It never invents a hash.
func (o *OnChainStaking) send(ctx context.Context, data []byte) (string, error) {
	if o == nil || o.client == nil || o.signer == nil {
		return "", ErrNotConfigured
	}

	nonce, err := o.client.PendingNonceAt(ctx, o.fromAddr)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	gasTip, _ := o.client.SuggestGasTipCap(ctx)
	gasFeeCap, err := o.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("gas price: %w", err)
	}

	to := o.contract
	gasLimit, err := o.client.EstimateGas(ctx, ethereum.CallMsg{
		From: o.fromAddr, To: &to, Data: data,
	})
	if err != nil || gasLimit == 0 {
		header, herr := o.client.HeaderByNumber(ctx, nil)
		if herr != nil || header == nil {
			return "", fmt.Errorf("gas estimation: %w", err)
		}
		gasLimit = header.GasLimit
	}

	raw := types.NewTx(&types.DynamicFeeTx{
		ChainID:   o.chainID,
		Nonce:     nonce,
		GasTipCap: gasTip,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &to,
		Data:      data,
	})

	signed, err := types.SignTx(raw, types.LatestSignerForChainID(o.chainID), o.signer)
	if err != nil {
		return "", fmt.Errorf("signing: %w", err)
	}
	if err := o.client.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("broadcast: %w", err)
	}
	return signed.Hash().Hex(), nil
}

// Close releases the RPC connection.
func (o *OnChainStaking) Close() {
	if o != nil && o.client != nil {
		o.client.Close()
	}
}

// parseTokenAmount converts a decimal token amount string into base units.
// Returns nil when the value is not a valid non-negative decimal.
func parseTokenAmount(value string, decimals int) (*big.Int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("empty amount")
	}
	// Reject anything that is not a plain non-negative decimal. big.Int's
	// SetString would happily accept a leading '-' and yield a negative
	// amount, which must never reach the contract.
	for _, r := range value {
		if (r < '0' || r > '9') && r != '.' {
			return nil, fmt.Errorf("invalid amount %q", value)
		}
	}
	parts := strings.SplitN(value, ".", 2)
	if len(parts) == 2 && strings.Contains(parts[1], ".") {
		return nil, fmt.Errorf("invalid amount %q", value)
	}
	intPart := parts[0]
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}
	if intPart == "" {
		intPart = "0"
	}
	if len(fracPart) > decimals {
		fracPart = fracPart[:decimals]
	}
	for len(fracPart) < decimals {
		fracPart += "0"
	}
	combined := intPart + fracPart
	n, ok := new(big.Int).SetString(combined, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount %q", value)
	}
	return n, nil
}

// tokenDecimals reads STAKING_TOKEN_DECIMALS (default 18).
func tokenDecimals() int {
	if v := strings.TrimSpace(os.Getenv("STAKING_TOKEN_DECIMALS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 36 {
			return n
		}
	}
	return 18
}
