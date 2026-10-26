// nft_signer.go — NFT write operations.
//
// The previous build referenced an NFTSigner interface and a NewRustNFTSigner
// constructor that were never defined anywhere in the repository, so the module
// could not compile. Signing is performed in-process with go-ethereum: the
// private key is read from configuration, used to sign an EIP-155 transaction,
// and never persisted or logged.

package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// NFTSigner performs the write half of the ChainClient surface.
type NFTSigner interface {
	SignAndSendNFTTransfer(ctx context.Context, from, to, contractAddr, tokenID string, amount *big.Int) (string, error)
	SignAndMintNFT(ctx context.Context, to, contractAddr, uri string) (string, string, error)
	Address() string
}

// EVMSigner signs NFT transactions with a local secp256k1 key.
type EVMSigner struct {
	client *EVMChainClient
}

// NewEVMSigner wraps an EVM chain client. The key comes from the client's
// configuration; when it is absent every write fails closed.
func NewEVMSigner(client *EVMChainClient) *EVMSigner {
	return &EVMSigner{client: client}
}

// Address returns the signer's checksummed address, or an empty string when no
// key is configured.
func (s *EVMSigner) Address() string {
	if s == nil || s.client == nil {
		return ""
	}
	key, err := s.client.signerKey()
	if err != nil {
		return ""
	}
	return crypto.PubkeyToAddress(key.PublicKey).Hex()
}

// SignAndSendNFTTransfer signs and broadcasts safeTransferFrom, then waits for
// the receipt. It returns the confirmed transaction hash.
func (s *EVMSigner) SignAndSendNFTTransfer(ctx context.Context, from, to, contractAddr, tokenID string, amount *big.Int) (string, error) {
	if s == nil || s.client == nil {
		return "", errors.New("signer is not configured")
	}
	if _, err := parseAddress(from); err != nil {
		return "", err
	}
	if _, err := parseAddress(to); err != nil {
		return "", err
	}
	if amount == nil || amount.Sign() <= 0 {
		return "", errors.New("transfer amount must be positive")
	}
	if _, ok := new(big.Int).SetString(strings.TrimSpace(tokenID), 10); !ok {
		return "", fmt.Errorf("invalid token id %q", tokenID)
	}
	return s.client.TransferNFT(ctx, from, to, contractAddr, tokenID, amount)
}

// SignAndMintNFT signs and broadcasts mint(address,string) and reads the minted
// token id back out of the receipt logs. When the contract does not emit a
// transfer log the token id is empty and the caller must not invent one.
func (s *EVMSigner) SignAndMintNFT(ctx context.Context, to, contractAddr, uri string) (string, string, error) {
	if s == nil || s.client == nil {
		return "", "", errors.New("signer is not configured")
	}
	recipient, err := parseAddress(to)
	if err != nil {
		return "", "", err
	}
	contract, err := parseAddress(contractAddr)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(uri) == "" {
		return "", "", errors.New("token URI is required")
	}

	hash, err := s.client.MintNFT(ctx, to, contractAddr, uri)
	if err != nil {
		return hash, "", err
	}

	tokenID, err := s.tokenIDFromMintReceipt(ctx, hash, contract, recipient)
	if err != nil {
		// The mint itself succeeded; the token id is simply unknown.
		return hash, "", nil
	}
	return hash, tokenID, nil
}

// erc721TransferTopic is keccak256("Transfer(address,address,uint256)").
var erc721TransferTopic = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

// tokenIDFromMintReceipt scans the mint receipt for an ERC-721 Transfer log
// whose recipient matches, and returns the token id from that log.
func (s *EVMSigner) tokenIDFromMintReceipt(ctx context.Context, hash string, contract, recipient common.Address) (string, error) {
	receipt, err := s.client.client.TransactionReceipt(ctx, common.HexToHash(hash))
	if err != nil {
		return "", fmt.Errorf("read mint receipt: %w", err)
	}

	for _, log := range receipt.Logs {
		if log.Address != contract {
			continue
		}
		if len(log.Topics) != 4 || log.Topics[0] != erc721TransferTopic {
			continue
		}
		if common.BytesToAddress(log.Topics[2].Bytes()) != recipient {
			continue
		}
		return new(big.Int).SetBytes(log.Topics[3].Bytes()).String(), nil
	}
	return "", errors.New("no ERC-721 Transfer log found in mint receipt")
}

// Ensure the interface is satisfied at compile time.
var _ NFTSigner = (*EVMSigner)(nil)

// EstimateNFTTransferGas reports the gas a transfer would consume. It is a real
// eth_estimateGas call, used by the API to show fees before signing.
func (c *EVMChainClient) EstimateNFTTransferGas(ctx context.Context, from, to, contractAddr, tokenID string) (uint64, error) {
	if !c.Available() {
		return 0, errRPCNotConfigured
	}
	contract, err := parseAddress(contractAddr)
	if err != nil {
		return 0, err
	}
	sender, err := parseAddress(from)
	if err != nil {
		return 0, err
	}
	recipient, err := parseAddress(to)
	if err != nil {
		return 0, err
	}
	token, ok := new(big.Int).SetString(strings.TrimSpace(tokenID), 10)
	if !ok {
		return 0, fmt.Errorf("invalid token id %q", tokenID)
	}

	data := selector("safeTransferFrom(address,address,uint256)")
	data = append(data, padAddress32(sender)...)
	data = append(data, padAddress32(recipient)...)
	data = append(data, padLeft32(token)...)

	return c.client.EstimateGas(ctx, ethereum.CallMsg{From: sender, To: &contract, Data: data})
}

// ChainID returns the chain the client is connected to, or nil when the RPC is
// unreachable.
func (c *EVMChainClient) ChainID() *big.Int {
	if !c.Available() {
		return nil
	}
	if c.chainID != nil {
		return new(big.Int).Set(c.chainID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id, err := c.client.ChainID(ctx)
	if err != nil {
		return nil
	}
	c.chainID = id
	return new(big.Int).Set(id)
}

// unusedTypesImport keeps types imported for the receipt status constants used
// by sendTransaction.
var _ = types.ReceiptStatusSuccessful
