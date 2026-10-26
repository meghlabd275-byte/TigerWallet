// nft_sale.go — atomic NFT-for-payment settlement.
//
// A marketplace sale has two legs: the buyer pays the seller, and the NFT moves
// to the buyer. Settling them in one transaction means a failure cannot leave
// the buyer paid with no NFT, or the seller stripped of the token with no
// payment.
//
// The implementation below targets an escrow/marketplace contract that exposes
//
//	buy(address collection, uint256 tokenId, address seller, uint256 price)
//
// which performs both legs internally. Because the payable amount must be sent
// with the call, a plain EIP-155 transaction cannot carry it through the shared
// sendTransaction helper; settleNFTSale builds and signs the call directly.

package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// errMarketplaceNotConfigured is returned when no marketplace contract address
// is configured, so a sale can never be settled against an unknown contract.
var errMarketplaceNotConfigured = errors.New("marketplace contract is not configured (NFT_MARKETPLACE_CONTRACT)")

// settleNFTSale pays the seller and transfers the NFT to the buyer in a single
// on-chain transaction against the configured marketplace contract.
func (s *NFTService) settleNFTSale(ctx context.Context, offer NFTOffer, buyerAddress string) (string, error) {
	marketplace := strings.TrimSpace(getEnv("NFT_MARKETPLACE_CONTRACT", ""))
	if marketplace == "" {
		return "", errMarketplaceNotConfigured
	}
	marketAddr, err := parseAddress(marketplace)
	if err != nil {
		return "", fmt.Errorf("invalid marketplace contract: %w", err)
	}

	chainClient, ok := s.chainClient.(*EVMChainClient)
	if !ok || !chainClient.Available() {
		return "", errRPCNotConfigured
	}

	buyer, err := parseAddress(buyerAddress)
	if err != nil {
		return "", fmt.Errorf("invalid buyer address: %w", err)
	}
	seller, err := parseAddress(offer.SellerAddress)
	if err != nil {
		return "", fmt.Errorf("invalid seller address: %w", err)
	}
	collection, err := parseAddress(offer.ContractAddress)
	if err != nil {
		return "", fmt.Errorf("invalid collection address: %w", err)
	}

	tokenID, ok := new(big.Int).SetString(strings.TrimSpace(offer.TokenID), 10)
	if !ok {
		return "", fmt.Errorf("invalid token id %q", offer.TokenID)
	}

	price, ok := new(big.Int).SetString(strings.TrimSpace(offer.Price), 10)
	if !ok || price.Sign() <= 0 {
		return "", fmt.Errorf("invalid offer price %q", offer.Price)
	}

	key, err := chainClient.signerKey()
	if err != nil {
		return "", err
	}
	from := crypto.PubkeyToAddress(key.PublicKey)
	if from != buyer {
		return "", fmt.Errorf("configured signer %s is not the offer buyer %s", from.Hex(), buyer.Hex())
	}

	callData := selector("buy(address,uint256,address,uint256)")
	callData = append(callData, padAddress32(collection)...)
	callData = append(callData, padLeft32(tokenID)...)
	callData = append(callData, padAddress32(seller)...)
	callData = append(callData, padLeft32(price)...)

	nonce, err := chainClient.client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}
	gasPrice, err := chainClient.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("suggest gas price: %w", err)
	}
	gasLimit, err := chainClient.client.EstimateGas(ctx, ethereum.CallMsg{
		From:  from,
		To:    &marketAddr,
		Value: price,
		Data:  callData,
	})
	if err != nil {
		return "", fmt.Errorf("estimate gas: %w", err)
	}

	chainID := chainClient.ChainID()
	if chainID == nil {
		return "", errors.New("chain id is unavailable")
	}

	tx := types.NewTransaction(nonce, marketAddr, price, gasLimit, gasPrice, callData)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return "", fmt.Errorf("sign sale transaction: %w", err)
	}
	if err := chainClient.client.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("broadcast sale transaction: %w", err)
	}

	hash := signed.Hash().Hex()
	receipt, err := chainClient.waitForReceipt(ctx, signed.Hash())
	if err != nil {
		return hash, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return hash, fmt.Errorf("sale transaction %s reverted", hash)
	}
	return hash, nil
}

// nftTransferTopic is keccak256("Transfer(address,address,uint256)") used to
// confirm the token actually moved to the buyer.
var nftTransferTopic = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

// verifyNFTSaleMoved checks the settlement receipt contains a Transfer log of
// the expected token to the buyer. It is used by callers that want to confirm
// the NFT leg landed before marking an offer complete.
func (s *NFTService) verifyNFTSaleMoved(ctx context.Context, txHash string, collection, buyer common.Address, tokenID *big.Int) (bool, error) {
	chainClient, ok := s.chainClient.(*EVMChainClient)
	if !ok || !chainClient.Available() {
		return false, errRPCNotConfigured
	}
	receipt, err := chainClient.client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return false, fmt.Errorf("read settlement receipt: %w", err)
	}
	for _, log := range receipt.Logs {
		if log.Address != collection || len(log.Topics) != 4 {
			continue
		}
		if log.Topics[0] != nftTransferTopic {
			continue
		}
		if common.BytesToAddress(log.Topics[2].Bytes()) != buyer {
			continue
		}
		if new(big.Int).SetBytes(log.Topics[3].Bytes()).Cmp(tokenID) == 0 {
			return true, nil
		}
	}
	return false, nil
}
