package main

// onchain.go — REAL on-chain execution for matured timelock transactions.
// Fail-closed: when TIMELOCK_RPC_URL / TIMELOCK_EXECUTOR_PRIVATE_KEY are not
// configured the service keeps the honest "pending_broadcast" status and never
// fabricates a transaction hash.
//
// Execution model: the executor wallet calls tx.Target directly with the
// stored calldata + value (this service's simplified timelock — the delay,
// grace window, and cancellation are enforced here, not by an on-chain
// TimelockController).

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// OnChainExecutor broadcasts matured timelock transactions from the
// operator-configured executor wallet.
type OnChainExecutor struct {
	client   *ethclient.Client
	pk       *ecdsa.PrivateKey
	fromAddr common.Address
	chainID  *big.Int
}

// NewOnChainExecutorFromEnv returns (nil, nil) when unconfigured; an error
// only when configuration is present but invalid.
func NewOnChainExecutorFromEnv() (*OnChainExecutor, error) {
	rpcURL := os.Getenv("TIMELOCK_RPC_URL")
	pkHex := os.Getenv("TIMELOCK_EXECUTOR_PRIVATE_KEY")
	if rpcURL == "" && pkHex == "" {
		return nil, nil
	}
	if rpcURL == "" || pkHex == "" {
		return nil, fmt.Errorf("TIMELOCK_RPC_URL and TIMELOCK_EXECUTOR_PRIVATE_KEY must both be set")
	}
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("ethclient dial: %w", err)
	}
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("chainID: %w", err)
	}
	pk, err := crypto.HexToECDSA(strings.TrimPrefix(pkHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}
	pubKey, ok := pk.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid public key type")
	}
	return &OnChainExecutor{
		client:   client,
		pk:       pk,
		fromAddr: crypto.PubkeyToAddress(*pubKey),
		chainID:  chainID,
	}, nil
}

// Execute broadcasts target(calldata) with valueWei from the executor wallet
// and waits for confirmation. Fail-closed: revert/RPC failure returns an
// error and the caller keeps the transaction un-executed.
func (e *OnChainExecutor) Execute(ctx context.Context, target common.Address, valueWei *big.Int, calldata []byte) (string, error) {
	gasLimit, err := e.client.EstimateGas(ctx, ethereum.CallMsg{
		From:  e.fromAddr,
		To:    &target,
		Data:  calldata,
		Value: valueWei,
	})
	if err != nil {
		return "", fmt.Errorf("estimateGas: %w", err)
	}
	nonce, err := e.client.PendingNonceAt(ctx, e.fromAddr)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	gasPrice, err := e.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("gasPrice: %w", err)
	}
	tx := types.NewTransaction(nonce, target, valueWei, gasLimit+10000, gasPrice, calldata)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(e.chainID), e.pk)
	if err != nil {
		return "", fmt.Errorf("signTx: %w", err)
	}
	if err := e.client.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("sendRawTransaction: %w", err)
	}
	txHash := signedTx.Hash().Hex()
	receiptCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	receipt, err := bind.WaitMined(receiptCtx, e.client, signedTx)
	if err != nil {
		return txHash, fmt.Errorf("tx broadcast but unconfirmed: %s", txHash)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return txHash, fmt.Errorf("tx reverted: %s", txHash)
	}
	return txHash, nil
}
