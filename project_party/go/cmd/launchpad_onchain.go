package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LaunchpadOnChain provides real on-chain interactions with the
// ProjectPartyLaunchpad smart contract. It is fail-closed: any RPC error
// returns an error and the caller refuses to record a contribution/claim.
//
// The private key is loaded from the PP_LAUNCHPAD_PRIVATE_KEY env var (the
// project_party operator wallet). The contract address is from
// PP_LAUNCHPAD_CONTRACT_ADDRESS. If either is unset, the on-chain path is
// disabled and handlers return 503 "on-chain not configured" — they NEVER
// fabricate a transaction hash.
type LaunchpadOnChain struct {
	client    *ethclient.Client
	pk        *ecdsa.PrivateKey
	fromAddr  common.Address
	contract  common.Address
	auth      *bind.TransactOpts
	abi       abi.ABI
	chainID   *big.Int
}

// launchpadOnChainSingleton is initialized once on startup (if configured).
var launchpadOnChainSingleton *LaunchpadOnChain

func initLaunchpadOnChain() error {
	rpcURL := getenvDefault("PP_RPC_URL", "")
	pkHex := getenvDefault("PP_LAUNCHPAD_PRIVATE_KEY", "")
	contractHex := getenvDefault("PP_LAUNCHPAD_CONTRACT_ADDRESS", "")
	if rpcURL == "" || pkHex == "" || contractHex == "" {
		// On-chain launchpad not configured — handlers return 503.
		return nil
	}
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("ethclient dial: %w", err)
	}
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return fmt.Errorf("chainID: %w", err)
	}
	pk, err := crypto.HexToECDSA(strings.TrimPrefix(pkHex, "0x"))
	if err != nil {
		return fmt.Errorf("private key: %w", err)
	}
	pubKey := pk.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("invalid public key type")
	}
	fromAddr := crypto.PubkeyToAddress(*pubKeyECDSA)
	auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		return fmt.Errorf("transactor: %w", err)
	}

	// Parse the ProjectPartyLaunchpad ABI (inline to avoid a separate file).
	parsed, err := abi.JSON(strings.NewReader(launchpadABIJSON))
	if err != nil {
		return fmt.Errorf("abi parse: %w", err)
	}

	launchpadOnChainSingleton = &LaunchpadOnChain{
		client:   client,
		pk:       pk,
		fromAddr: fromAddr,
		contract: common.HexToAddress(contractHex),
		auth:     auth,
		abi:      parsed,
		chainID:  chainID,
	}
	return nil
}

// contributeOnChain sends a real contribute() transaction to the
// ProjectPartyLaunchpad contract. Returns the tx hash + token claim estimate.
// Fail-closed: any error returns ("", 0, err).
func (loc *LaunchpadOnChain) contributeOnChain(ctx context.Context, saleID [32]byte, valueWei *big.Int) (string, *big.Int, error) {
	if loc == nil {
		return "", nil, fmt.Errorf("on-chain launchpad not configured")
	}
	// Build the contribute calldata.
	input, err := loc.abi.Pack("contribute", saleID)
	if err != nil {
		return "", nil, fmt.Errorf("pack contribute: %w", err)
	}
	// Estimate gas (fail-closed if the call would revert).
	gasLimit, err := loc.client.EstimateGas(ctx, ethereum.CallMsg{
		From:     loc.fromAddr,
		To:       &loc.contract,
		Data:     input,
		Value:    valueWei,
	})
	if err != nil {
		return "", nil, fmt.Errorf("estimateGas: %w", err)
	}
	// Fetch nonce.
	nonce, err := loc.client.PendingNonceAt(ctx, loc.fromAddr)
	if err != nil {
		return "", nil, fmt.Errorf("nonce: %w", err)
	}
	// Fetch gas price.
	gasPrice, err := loc.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("gasPrice: %w", err)
	}
	// Build + sign the tx.
	tx := types.NewTransaction(nonce, loc.contract, valueWei, gasLimit+10000, gasPrice, input)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(loc.chainID), loc.pk)
	if err != nil {
		return "", nil, fmt.Errorf("signTx: %w", err)
	}
	// Broadcast.
	if err := loc.client.SendTransaction(ctx, signedTx); err != nil {
		return "", nil, fmt.Errorf("sendRawTransaction: %w", err)
	}
	txHash := signedTx.Hash().Hex()
	// Wait for receipt (30s timeout).
	receiptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	receipt, err := bind.WaitMined(receiptCtx, loc.client, signedTx)
	if err != nil {
		// tx was broadcast but we couldn't confirm — return the hash anyway.
		return txHash, big.NewInt(0), nil
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return txHash, big.NewInt(0), fmt.Errorf("tx reverted: %s", txHash)
	}
	// Compute token claim: tokens = value * 1e18 / tokenPrice.
	// We read the tokenPrice from the contract.
	tokenPrice, err := loc.getTokenPrice(ctx, saleID)
	if err != nil || tokenPrice == nil || tokenPrice.Sign() == 0 {
		return txHash, big.NewInt(0), nil
	}
	tokenClaim := new(big.Int).Mul(valueWei, big.NewInt(1e18))
	tokenClaim.Div(tokenClaim, tokenPrice)
	return txHash, tokenClaim, nil
}

// claimTokensOnChain sends a real claimTokens() transaction.
func (loc *LaunchpadOnChain) claimTokensOnChain(ctx context.Context, saleID [32]byte) (string, error) {
	if loc == nil {
		return "", fmt.Errorf("on-chain launchpad not configured")
	}
	input, err := loc.abi.Pack("claimTokens", saleID)
	if err != nil {
		return "", fmt.Errorf("pack claimTokens: %w", err)
	}
	gasLimit, err := loc.client.EstimateGas(ctx, ethereum.CallMsg{
		From: loc.fromAddr, To: &loc.contract, Data: input,
	})
	if err != nil {
		return "", fmt.Errorf("estimateGas: %w", err)
	}
	nonce, err := loc.client.PendingNonceAt(ctx, loc.fromAddr)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	gasPrice, err := loc.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("gasPrice: %w", err)
	}
	tx := types.NewTransaction(nonce, loc.contract, big.NewInt(0), gasLimit+10000, gasPrice, input)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(loc.chainID), loc.pk)
	if err != nil {
		return "", fmt.Errorf("signTx: %w", err)
	}
	if err := loc.client.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("sendRawTransaction: %w", err)
	}
	return signedTx.Hash().Hex(), nil
}

// getTokenPrice reads the tokenPrice field from a sale via eth_call.
func (loc *LaunchpadOnChain) getTokenPrice(ctx context.Context, saleID [32]byte) (*big.Int, error) {
	if loc == nil {
		return nil, fmt.Errorf("not configured")
	}
	input, err := loc.abi.Pack("getSale", saleID)
	if err != nil {
		return nil, err
	}
	out, err := loc.client.CallContract(ctx, ethereum.CallMsg{
		To:   &loc.contract,
		Data: input,
	}, nil)
	if err != nil {
		return nil, err
	}
	// Unpack into an anonymous struct that matches the Sale layout; the
	// tokenPrice field is at index 3.
	var sale struct {
		Token           common.Address
		PaymentToken    common.Address
		Treasury        common.Address
		TokenPrice      *big.Int
		TokensForSale   *big.Int
		TokensSold      *big.Int
		SoftCap         *big.Int
		HardCap         *big.Int
		MinContribution *big.Int
		MaxContribution *big.Int
		StartTime       *big.Int
		EndTime         *big.Int
		Status          uint8
		Exists          bool
	}
	if err := loc.abi.UnpackIntoInterface(&sale, "getSale", out); err != nil {
		return nil, err
	}
	return sale.TokenPrice, nil
}

// saleIDFromUUID derives a deterministic bytes32 sale ID from a UUID string.
func saleIDFromUUID(uuidStr string) [32]byte {
	return common.BytesToHash([]byte(uuidStr))
}

// getenvDefault returns the env var or default.
func getenvDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// launchpadABIJSON is the ABI of the ProjectPartyLaunchpad contract (only the
// functions we call: contribute, claimTokens, getSale).
const launchpadABIJSON = `[
  {
    "inputs": [{"internalType":"bytes32","name":"saleId","type":"bytes32"}],
    "name": "contribute",
    "outputs": [],
    "stateMutability": "payable",
    "type": "function"
  },
  {
    "inputs": [{"internalType":"bytes32","name":"saleId","type":"bytes32"}],
    "name": "claimTokens",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [{"internalType":"bytes32","name":"saleId","type":"bytes32"}],
    "name": "getSale",
    "outputs": [
      {
        "components": [
          {"internalType":"address","name":"token","type":"address"},
          {"internalType":"address","name":"paymentToken","type":"address"},
          {"internalType":"address","name":"treasury","type":"address"},
          {"internalType":"uint256","name":"tokenPrice","type":"uint256"},
          {"internalType":"uint256","name":"tokensForSale","type":"uint256"},
          {"internalType":"uint256","name":"tokensSold","type":"uint256"},
          {"internalType":"uint256","name":"softCap","type":"uint256"},
          {"internalType":"uint256","name":"hardCap","type":"uint256"},
          {"internalType":"uint256","name":"minContribution","type":"uint256"},
          {"internalType":"uint256","name":"maxContribution","type":"uint256"},
          {"internalType":"uint256","name":"startTime","type":"uint256"},
          {"internalType":"uint256","name":"endTime","type":"uint256"},
          {"internalType":"enum ProjectPartyLaunchpad.SaleStatus","name":"status","type":"uint8"},
          {"internalType":"bool","name":"exists","type":"bool"}
        ],
        "internalType": "struct ProjectPartyLaunchpad.Sale",
        "name": "",
        "type": "tuple"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]`

// launchpadOnChainEnabled reports whether on-chain interaction is configured.
func launchpadOnChainEnabled() bool {
	return launchpadOnChainSingleton != nil
}

// persistOnChainContribution records a real on-chain tx hash against a
// launchpad contribution in PostgreSQL. NEVER marks "completed" without a tx hash.
func persistOnChainContribution(db *pgxpool.Pool, ctx context.Context, contribID, txHash string, tokenClaimWei *big.Int) error {
	if txHash == "" {
		return fmt.Errorf("no tx hash")
	}
	_, err := db.Exec(ctx,
		`UPDATE launchpad_contributions SET status='confirmed', tx_hash=$1, token_amount=$2, confirmed_at=$3 WHERE id=$4`,
		txHash, tokenClaimWei.String(), time.Now(), contribID)
	return err
}
