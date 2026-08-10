/**
 * TigerWallet Multisig Service
 *
 * Multi-signature transaction support.
 * Built with Go for high-load distributed operations.
 */

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
)

// MultisigWallet represents a multisig wallet
type MultisigWallet struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Owners       []string `json:"owners"`
	RequiredSigs int      `json:"required_sigs"`
	Threshold    int      `json:"threshold"`
	ChainID      uint64   `json:"chain_id"`
	Address      string   `json:"address"`
	Status       string   `json:"status"`
	CreatedAt    int64    `json:"created_at"`
}

// MultisigTransaction represents a multisig transaction
type MultisigTransaction struct {
	ID         string   `json:"id"`
	WalletID   string   `json:"wallet_id"`
	To         string   `json:"to"`
	Value      string   `json:"value"`
	Data       string   `json:"data"`
	Signatures []string `json:"signatures"`
	SignedBy   []string `json:"signed_by"`
	Status     string   `json:"status"`
	ExecutedAt int64    `json:"executed_at"`
	CreatedAt  int64    `json:"created_at"`
}

// MultisigService manages multisig operations
type MultisigService struct {
	mu           sync.RWMutex
	wallets      map[string]*MultisigWallet
	transactions map[string]*MultisigTransaction
	privateKey   *ecdsa.PrivateKey
	rpcURL       string
	chainID      int64
}

var (
	multisigService     *MultisigService
	multisigServiceOnce sync.Once
)

// GetMultisigService returns the singleton multisig service. The relayer key is
// loaded from ETH_RELAYER_PRIVATE_KEY (real secp256k1); if absent a fresh key is
// generated. The RPC endpoint is taken from ETH_RPC_URL.
func GetMultisigService() *MultisigService {
	multisigServiceOnce.Do(func() {
		priv, err := loadRelayerKey()
		if err != nil {
			// loadRelayerKey only fails on an invalid hex key, which is a
			// configuration error we want surfaced.
			panic(fmt.Sprintf("invalid ETH_RELAYER_PRIVATE_KEY: %v", err))
		}
		cfg := LoadConfig()
		multisigService = &MultisigService{
			wallets:      make(map[string]*MultisigWallet),
			transactions: make(map[string]*MultisigTransaction),
			privateKey:   priv,
			rpcURL:       cfg.RpcURL,
			chainID:      cfg.ChainID,
		}
	})
	return multisigService
}

func (s *MultisigService) CreateWallet(ctx context.Context, wallet *MultisigWallet) (*MultisigWallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(wallet.Owners) < wallet.RequiredSigs {
		return nil, fmt.Errorf("owners must be >= required signatures")
	}

	wallet.ID = "multisig_" + uuid.New().String()
	wallet.Status = "active"
	wallet.CreatedAt = time.Now().Unix()

	s.wallets[wallet.ID] = wallet
	return wallet, nil
}

func (s *MultisigService) CreateTransaction(ctx context.Context, tx *MultisigTransaction) (*MultisigTransaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.wallets[tx.WalletID]; !exists {
		return nil, fmt.Errorf("wallet not found")
	}

	tx.ID = "tx_" + uuid.New().String()
	tx.Status = "pending"
	tx.CreatedAt = time.Now().Unix()

	s.transactions[tx.ID] = tx
	return tx, nil
}

func (s *MultisigService) SignTransaction(ctx context.Context, txID, signature, signer string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, exists := s.transactions[txID]
	if !exists {
		return fmt.Errorf("transaction not found")
	}

	if tx.Status == "executed" {
		return fmt.Errorf("transaction already executed")
	}

	wallet, _ := s.wallets[tx.WalletID]
	if wallet == nil {
		return fmt.Errorf("wallet not found")
	}

	// Check if signer is an owner
	isOwner := false
	for _, owner := range wallet.Owners {
		if owner == signer {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return fmt.Errorf("signer is not an owner")
	}

	// Check if already signed
	for _, signed := range tx.SignedBy {
		if signed == signer {
			return fmt.Errorf("already signed")
		}
	}

	// signature is expected to be a 0x-prefixed 65-byte secp256k1 signature
	// (r||s||v) produced by crypto.Sign over the transaction digest. We verify
	// it recovers to the signer's address before recording it.
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	if err != nil || len(sigBytes) != 65 {
		return fmt.Errorf("invalid signature: must be 65-byte hex")
	}
	v := sigBytes[64]
	if v == 27 || v == 28 {
		v -= 27
		sigBytes[64] = v
	}
	digest := s.transactionDigest(tx)
	pubKey, err := crypto.Ecrecover(digest, sigBytes)
	if err != nil {
		return fmt.Errorf("signature recovery failed: %w", err)
	}
	recovered := crypto.PubkeyToAddress(ecdsaPubKey(pubKey))
	if !strings.EqualFold(recovered.Hex(), signer) {
		return fmt.Errorf("signature does not match signer address")
	}

	tx.Signatures = append(tx.Signatures, signature)
	tx.SignedBy = append(tx.SignedBy, signer)

	// Check if threshold reached
	if len(tx.SignedBy) >= wallet.RequiredSigs {
		tx.Status = "ready"
	}

	return nil
}

// transactionDigest is the 32-byte keccak256 hash owners sign over for a
// multisig transaction (chainID || nonce || to || value || data).
func (s *MultisigService) transactionDigest(tx *MultisigTransaction) []byte {
	to := common.HexToAddress(tx.To)
	value, ok := new(big.Int).SetString(tx.Value, 10)
	if !ok {
		value = new(big.Int)
	}
	data, _ := hex.DecodeString(strings.TrimPrefix(tx.Data, "0x"))
	return crypto.Keccak256(
		common.LeftPadBytes(big.NewInt(s.chainID).Bytes(), 32),
		common.LeftPadBytes(big.NewInt(int64(tx.CreatedAt)).Bytes(), 32),
		common.LeftPadBytes(to.Bytes(), 32),
		common.LeftPadBytes(value.Bytes(), 32),
		common.LeftPadBytes(new(big.Int).SetUint64(uint64(len(data))).Bytes(), 32),
		data,
	)
}

// ecdsaPubKey converts a 65-byte uncompressed secp256k1 public key (as returned
// by crypto.Ecrecover) into an ecdsa.PublicKey value.
func ecdsaPubKey(uncompressed []byte) ecdsa.PublicKey {
	if len(uncompressed) == 0 {
		return ecdsa.PublicKey{}
	}
	pub, err := crypto.UnmarshalPubkey(uncompressed)
	if err != nil || pub == nil {
		return ecdsa.PublicKey{}
	}
	return *pub
}

// ExecuteTransaction broadcasts the approved multisig transaction to a real
// Ethereum node via JSON-RPC (eth_sendRawTransaction) using the relayer key.
// The supplied txHash is ignored on input; the real hash returned by the node
// is assigned to the transaction record.
func (s *MultisigService) ExecuteTransaction(ctx context.Context, txID, txHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, exists := s.transactions[txID]
	if !exists {
		return fmt.Errorf("transaction not found")
	}

	if tx.Status == "executed" {
		return fmt.Errorf("transaction already executed")
	}

	wallet, _ := s.wallets[tx.WalletID]
	if wallet == nil {
		return fmt.Errorf("wallet not found")
	}

	if len(tx.SignedBy) < wallet.RequiredSigs {
		return fmt.Errorf("insufficient signatures")
	}

	realHash, err := s.broadcastRawTransaction(tx)
	if err != nil {
		tx.Status = "failed"
		return fmt.Errorf("broadcast failed: %w", err)
	}
	_ = txHash // legacy placeholder input; the real hash comes from the node

	tx.Status = "executed"
	tx.ExecutedAt = time.Now().Unix()
	tx.Signatures = append(tx.Signatures, realHash) // record the on-chain tx hash
	return nil
}

// broadcastRawTransaction signs and submits the multisig transaction to the
// configured Ethereum node, returning the real transaction hash.
func (s *MultisigService) broadcastRawTransaction(tx *MultisigTransaction) (string, error) {
	if s.rpcURL == "" {
		s.rpcURL = os.Getenv("ETH_RPC_URL")
	}
	if s.rpcURL == "" {
		return "", fmt.Errorf("ETH_RPC_URL not configured")
	}

	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(callCtx, s.rpcURL)
	if err != nil {
		return "", fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

	fromAddr := crypto.PubkeyToAddress(s.privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(callCtx, fromAddr)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	valueWei, ok := new(big.Int).SetString(tx.Value, 10)
	if !ok {
		valueWei = new(big.Int)
	}
	toAddr := common.HexToAddress(tx.To)
	data, _ := hex.DecodeString(strings.TrimPrefix(tx.Data, "0x"))

	gasLimit := uint64(210000)
	if est, err := client.EstimateGas(callCtx, ethereum.CallMsg{
		From: fromAddr, To: &toAddr, GasPrice: big.NewInt(0), Value: valueWei, Data: data,
	}); err == nil {
		gasLimit = est * 120 / 100
		if gasLimit < 21000 {
			gasLimit = 21000
		}
	}

	chainID := big.NewInt(s.chainID)
	gasPrice, err := client.SuggestGasPrice(callCtx)
	if err != nil {
		return "", fmt.Errorf("gas price: %w", err)
	}
	tip, err := client.SuggestGasTipCap(callCtx)
	if err != nil {
		tip = gasPrice
	}
	feeCap := new(big.Int).Add(gasPrice, tip)

	signer := types.NewLondonSigner(chainID)
	rawTx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        &toAddr,
		Value:     valueWei,
		Gas:       gasLimit,
		GasFeeCap: feeCap,
		GasTipCap: tip,
		Data:      data,
	})

	signedTx, err := types.SignTx(rawTx, signer, s.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign tx: %w", err)
	}
	if err := client.SendTransaction(callCtx, signedTx); err != nil {
		return "", fmt.Errorf("send raw transaction: %w", err)
	}
	return signedTx.Hash().Hex(), nil
}

func (s *MultisigService) GetTransaction(ctx context.Context, txID string) (*MultisigTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, exists := s.transactions[txID]
	if !exists {
		return nil, fmt.Errorf("transaction not found")
	}
	return tx, nil
}

func (s *MultisigService) GetWalletTransactions(ctx context.Context, walletID string) ([]*MultisigTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*MultisigTransaction, 0)
	for _, tx := range s.transactions {
		if tx.WalletID == walletID {
			result = append(result, tx)
		}
	}
	return result, nil
}

func (w *MultisigWallet) ToJSON() (string, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
