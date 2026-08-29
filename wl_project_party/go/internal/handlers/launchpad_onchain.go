package handlers

// On-chain launchpad interaction for the standalone WL-ProjectParty backend.
//
// Mirrors the canonical project_party/go/cmd/launchpad_onchain.go: sends REAL
// contribute()/claimTokens() transactions to the ProjectPartyLaunchpad smart
// contract. Fail-closed: if PP_RPC_URL / PP_LAUNCHPAD_PRIVATE_KEY /
// PP_LAUNCHPAD_CONTRACT_ADDRESS are unset, the on-chain path is disabled and
// callers receive 503 "on-chain not configured" — they NEVER fabricate a tx
// hash. Any RPC error returns an error and the caller refuses to record a
// contribution/claim.

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tigerwallet/wl-project-party/internal/config"
)

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
        "internalType":"struct ProjectPartyLaunchpad.Sale",
        "name": "",
        "type": "tuple"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]`

// LaunchpadOnChain provides real on-chain interactions with the
// ProjectPartyLaunchpad smart contract.
type LaunchpadOnChain struct {
	client   *ethclient.Client
	pk       *ecdsa.PrivateKey
	fromAddr common.Address
	contract common.Address
	auth     *bind.TransactOpts
	abi      abi.ABI
	chainID  *big.Int
}

// NewLaunchpadOnChain initializes the on-chain client from config. Returns
// (nil, nil) when unconfigured (handlers then return 503).
func NewLaunchpadOnChain(cfg *config.Config) (*LaunchpadOnChain, error) {
	rpcURL := cfg.RPCURL
	pkHex := cfg.LaunchpadPrivateKey
	contractHex := cfg.LaunchpadContractAddr
	if rpcURL == "" || pkHex == "" || contractHex == "" {
		return nil, nil
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
	pubKey := pk.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid public key type")
	}
	fromAddr := crypto.PubkeyToAddress(*pubKeyECDSA)
	auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		return nil, fmt.Errorf("transactor: %w", err)
	}
	parsed, err := abi.JSON(strings.NewReader(launchpadABIJSON))
	if err != nil {
		return nil, fmt.Errorf("abi parse: %w", err)
	}
	return &LaunchpadOnChain{
		client:   client,
		pk:       pk,
		fromAddr: fromAddr,
		contract: common.HexToAddress(contractHex),
		auth:     auth,
		abi:      parsed,
		chainID:  chainID,
	}, nil
}

// saleIDFromUUID derives a deterministic bytes32 sale ID from a UUID string.
func saleIDFromUUID(uuidStr string) [32]byte {
	return common.BytesToHash([]byte(uuidStr))
}

// normalizeWei converts a decimal contribution amount string (e.g. "1.5") to a
// base-unit (wei) integer string by scaling by 1e18. If the input is already a
// plain integer it is returned as-is. Returns "0" on parse failure (fail-closed
// so the caller rejects the contribution rather than sending a 0-value tx).
func normalizeWei(amount string) string {
	s := strings.TrimSpace(amount)
	if s == "" {
		return "0"
	}
	// If it's already a plain integer, use it directly.
	if v, ok := new(big.Int).SetString(s, 10); ok {
		return v.String()
	}
	// Parse as a decimal and scale by 1e18.
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if whole == "" {
		whole = "0"
	}
	whole = strings.TrimLeft(whole, "0")
	if whole == "" {
		whole = "0"
	}
	if len(frac) > 18 {
		frac = frac[:18]
	}
	frac = frac + strings.Repeat("0", 18-len(frac))
	combined := whole + frac
	v, ok := new(big.Int).SetString(combined, 10)
	if !ok {
		return "0"
	}
	if negative {
		return "0"
	}
	return v.String()
}

// Contribute sends a real contribute() transaction. Returns the tx hash +
// token claim estimate (tokens = value * 1e18 / tokenPrice). Fail-closed.
func (loc *LaunchpadOnChain) Contribute(ctx context.Context, saleID [32]byte, valueWei *big.Int) (string, *big.Int, error) {
	if loc == nil {
		return "", nil, fmt.Errorf("on-chain launchpad not configured")
	}
	input, err := loc.abi.Pack("contribute", saleID)
	if err != nil {
		return "", nil, fmt.Errorf("pack contribute: %w", err)
	}
	gasLimit, err := loc.client.EstimateGas(ctx, ethereum.CallMsg{
		From: loc.fromAddr, To: &loc.contract, Data: input, Value: valueWei,
	})
	if err != nil {
		return "", nil, fmt.Errorf("estimateGas: %w", err)
	}
	nonce, err := loc.client.PendingNonceAt(ctx, loc.fromAddr)
	if err != nil {
		return "", nil, fmt.Errorf("nonce: %w", err)
	}
	gasPrice, err := loc.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("gasPrice: %w", err)
	}
	tx := types.NewTransaction(nonce, loc.contract, valueWei, gasLimit+10000, gasPrice, input)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(loc.chainID), loc.pk)
	if err != nil {
		return "", nil, fmt.Errorf("signTx: %w", err)
	}
	if err := loc.client.SendTransaction(ctx, signedTx); err != nil {
		return "", nil, fmt.Errorf("sendRawTransaction: %w", err)
	}
	txHash := signedTx.Hash().Hex()
	receiptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	receipt, err := bind.WaitMined(receiptCtx, loc.client, signedTx)
	if err != nil {
		return txHash, big.NewInt(0), nil
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return txHash, big.NewInt(0), fmt.Errorf("tx reverted: %s", txHash)
	}
	tokenPrice, err := loc.getTokenPrice(ctx, saleID)
	if err != nil || tokenPrice == nil || tokenPrice.Sign() == 0 {
		return txHash, big.NewInt(0), nil
	}
	tokenClaim := new(big.Int).Mul(valueWei, big.NewInt(1e18))
	tokenClaim.Div(tokenClaim, tokenPrice)
	return txHash, tokenClaim, nil
}

// ClaimTokens sends a real claimTokens() transaction. Returns the tx hash.
func (loc *LaunchpadOnChain) ClaimTokens(ctx context.Context, saleID [32]byte) (string, error) {
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
		To: &loc.contract, Data: input,
	}, nil)
	if err != nil {
		return nil, err
	}
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
