package launchpad

// launchpad_onchain.go — REAL on-chain claim / reward-payout execution for the
// launchpad + launchpool. Fail-closed: when the operator has not configured an
// RPC endpoint, distribution key, and contract address, every on-chain handler
// returns 503 and records nothing. No fabricated transaction hashes.
//
// Configuration (operator environment, never hardcoded):
//   LAUNCHPAD_RPC_URL           EVM JSON-RPC endpoint
//   LAUNCHPAD_PRIVATE_KEY       launchpad distribution wallet key (hex)
//   LAUNCHPAD_CONTRACT_ADDRESS  deployed launchpad/pool contract
//   LAUNCHPAD_CLAIM_FN          claim function signature (default "claimTokens(bytes32)")
//   LAUNCHPAD_REWARD_FN         reward function signature (default "claimRewards(bytes32)")
//
// The sale/pool id is derived from the project id exactly like the canonical
// project_party service: common.BytesToHash([]byte(projectID)).

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

// OnChainClient executes real signed transactions against the launchpad/pool
// contract from the operator-controlled distribution wallet.
type OnChainClient struct {
	client    *ethclient.Client
	pk        *ecdsa.PrivateKey
	fromAddr  common.Address
	contract  common.Address
	chainID   *big.Int
	claimFn   string
	rewardFn  string
}

func onchainEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// NewOnChainClientFromEnv builds the on-chain executor. Returns (nil, nil)
// when unconfigured so handlers can fail closed with 503; returns an error
// only when configuration is present but invalid.
func NewOnChainClientFromEnv() (*OnChainClient, error) {
	rpcURL := onchainEnv("LAUNCHPAD_RPC_URL", "")
	pkHex := onchainEnv("LAUNCHPAD_PRIVATE_KEY", "")
	contractHex := onchainEnv("LAUNCHPAD_CONTRACT_ADDRESS", "")
	if rpcURL == "" && pkHex == "" && contractHex == "" {
		return nil, nil
	}
	if rpcURL == "" || pkHex == "" || contractHex == "" {
		return nil, fmt.Errorf("LAUNCHPAD_RPC_URL, LAUNCHPAD_PRIVATE_KEY and LAUNCHPAD_CONTRACT_ADDRESS must all be set")
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
	return &OnChainClient{
		client:   client,
		pk:       pk,
		fromAddr: crypto.PubkeyToAddress(*pubKey),
		contract: common.HexToAddress(contractHex),
		chainID:  chainID,
		claimFn:  onchainEnv("LAUNCHPAD_CLAIM_FN", "claimTokens(bytes32)"),
		rewardFn: onchainEnv("LAUNCHPAD_REWARD_FN", "claimRewards(bytes32)"),
	}, nil
}

// selector returns the 4-byte function selector for a canonical signature.
func selector(sig string) []byte {
	return crypto.Keccak256([]byte(sig))[:4]
}

// saleID derives the deterministic bytes32 id for a project (canonical form).
func saleID(projectID string) [32]byte {
	return common.BytesToHash([]byte(projectID))
}

// sendTx estimates gas, signs (EIP-155), broadcasts and waits for the receipt.
// Returns the tx hash. Fail-closed: a revert or RPC error returns an error and
// the caller records nothing.
func (c *OnChainClient) sendTx(ctx context.Context, data []byte, valueWei *big.Int) (string, error) {
	gasLimit, err := c.client.EstimateGas(ctx, ethereum.CallMsg{
		From:  c.fromAddr,
		To:    &c.contract,
		Data:  data,
		Value: valueWei,
	})
	if err != nil {
		return "", fmt.Errorf("estimateGas: %w", err)
	}
	nonce, err := c.client.PendingNonceAt(ctx, c.fromAddr)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("gasPrice: %w", err)
	}
	tx := types.NewTransaction(nonce, c.contract, valueWei, gasLimit+10000, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(c.chainID), c.pk)
	if err != nil {
		return "", fmt.Errorf("signTx: %w", err)
	}
	if err := c.client.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("sendRawTransaction: %w", err)
	}
	txHash := signedTx.Hash().Hex()
	receiptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	receipt, err := bind.WaitMined(receiptCtx, c.client, signedTx)
	if err != nil {
		// Broadcast succeeded but confirmation timed out — return the real hash
		// so the caller can reconcile later; do NOT claim success.
		return txHash, fmt.Errorf("tx broadcast but unconfirmed: %s", txHash)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return txHash, fmt.Errorf("tx reverted: %s", txHash)
	}
	return txHash, nil
}

// ClaimTokens broadcasts the configured claim call for a project sale.
func (c *OnChainClient) ClaimTokens(ctx context.Context, projectID string) (string, error) {
	id := saleID(projectID)
	data := append(selector(c.claimFn), id[:]...)
	return c.sendTx(ctx, data, big.NewInt(0))
}

// ClaimRewards broadcasts the configured reward-payout call for a pool.
func (c *OnChainClient) ClaimRewards(ctx context.Context, projectID string) (string, error) {
	id := saleID(projectID)
	data := append(selector(c.rewardFn), id[:]...)
	return c.sendTx(ctx, data, big.NewInt(0))
}
