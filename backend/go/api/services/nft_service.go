package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"tigerwallet/backend/go/api/models"
)

type NFTService struct {
	mu          sync.RWMutex
	collections map[string]*models.NFTCollection
	nfts         map[string]*models.NFT
}

var (
	nftInstance *NFTService
	nftOnce    sync.Once
)

func NewNFTService() *NFTService {
	nftOnce.Do(func() {
		nftInstance = &NFTService{
			collections: make(map[string]*models.NFTCollection),
			nfts:         make(map[string]*models.NFT),
		}
		nftInstance.initializeDefaultCollections()
	})
	return nftInstance
}

func (s *NFTService) initializeDefaultCollections() {
	defaultCollections := []*models.NFTCollection{
		{Address: "0xbc4ca0eda7647a8ab7c2061c2e118a18a936f13d", BlockchainID: "ethereum", Name: "Bored Ape Yacht Club", Symbol: "BAYC", Description: "The premier NFT collection on Ethereum", ImageURL: "https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfNEuhunv6FNeMFGa9bx6mQ", BannerURL: "", FloorPrice: "30", FloorPriceUSD: 103500, TotalSupply: "10000", HolderCount: "6500"},
		{Address: "0x23581767a106ae36c844b052ba0fafe2d1dde1d0", BlockchainID: "ethereum", Name: "Milady Maker", Symbol: "MILADY", Description: "A collection of 10000 avatars", ImageURL: "https://ipfs.io/ipfs/QmYxL6YvBXS8YpX9vN4zV1N5xX8Y9Z", BannerURL: "", FloorPrice: "1.8", FloorPriceUSD: 6210, TotalSupply: "10000", HolderCount: "5200"},
		{Address: "0x49cf6f3d145c3325b5d4c27b7f2ae4b4b3d5c9e1", BlockchainID: "ethereum", Name: "Pudgy Penguins", Symbol: "PENGU", Description: "Cute NFT collection with utility", ImageURL: "https://ipfs.io/ipfs/QmPudgyPenguins", BannerURL: "", FloorPrice: "2.5", FloorPriceUSD: 8625, TotalSupply: "8888", HolderCount: "4800"},
		{Address: "0x8a90cab6b2abafb87c1a14acf7c3f3e8b5c8b3e", BlockchainID: "ethereum", Name: "Doodle", Symbol: "DOODLE", Description: "Community-driven NFT collection", ImageURL: "https://ipfs.io/ipfs/QmDoodles", BannerURL: "", FloorPrice: "1.9", FloorPriceUSD: 6555, TotalSupply: "10000", HolderCount: "5800"},
		{Address: "0xed5af388653567af2f388e6224dc7c4b55b5e5ac", BlockchainID: "ethereum", Name: "Azuki", Symbol: "AZUKI", Description: "The rice army is coming", ImageURL: "https://ipfs.io/ipfs/QmYyM9E25W", BannerURL: "", FloorPrice: "8.5", FloorPriceUSD: 29325, TotalSupply: "10000", HolderCount: "6200"},
		{Address: "0xb47e3cd837dDF8e4c57F05d30Ab875a6a2c05D33", BlockchainID: "ethereum", Name: "CryptoPunks", Symbol: "PUNK", Description: "The original NFT collection", ImageURL: "https://ipfs.io/ipfs/QmebX1c", BannerURL: "", FloorPrice: "35", FloorPriceUSD: 120750, TotalSupply: "10000", HolderCount: "4200"},
		{Address: "0x4e1f416c0efb14a7c5a6d04a52b8d5c6d7c4e5f6", BlockchainID: "solana", Name: "Degenerate Ape Academy", Symbol: "DEGOD", Description: "Premier Solana NFT collection", ImageURL: "https://arweave.net/degod", BannerURL: "", FloorPrice: "25", FloorPriceUSD: 3625, TotalSupply: "10000", HolderCount: "5800"},
		{Address: "0x7bd2d3c4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", BlockchainID: "solana", Name: "Okay Bears", Symbol: "OKAY", Description: "Community-driven Solana collection", ImageURL: "https://arweave.net/okaybears", BannerURL: "", FloorPrice: "40", FloorPriceUSD: 5800, TotalSupply: "10000", HolderCount: "7200"},
	}

	for _, col := range defaultCollections {
		s.collections[col.Address] = col
	}
}

func (s *NFTService) GetCollections(ctx context.Context, blockchainID string) ([]*models.NFTCollection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.NFTCollection
	for _, col := range s.collections {
		if blockchainID == "" || col.BlockchainID == blockchainID {
			result = append(result, col)
		}
	}

	return result, nil
}

func (s *NFTService) GetCollectionByAddress(ctx context.Context, address string) (*models.NFTCollection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	col, ok := s.collections[address]
	if !ok {
		return nil, errors.New("collection not found")
	}

	return col, nil
}

func (s *NFTService) GetNFTs(ctx context.Context, walletID, collectionAddress string) ([]*models.NFT, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.NFT
	for _, nft := range s.nfts {
		if walletID != "" && nft.WalletID != walletID {
			continue
		}
		if collectionAddress != "" && nft.CollectionAddr != collectionAddress {
			continue
		}
		result = append(result, nft)
	}

	return result, nil
}

func (s *NFTService) GetNFTByID(ctx context.Context, nftID string) (*models.NFT, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nft, ok := s.nfts[nftID]
	if !ok {
		return nil, errors.New("NFT not found")
	}

	return nft, nil
}

func (s *NFTService) TransferNFT(ctx context.Context, nftID, toAddress string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	nft, ok := s.nfts[nftID]
	if !ok {
		return errors.New("NFT not found")
	}

	nft.Owner = toAddress
	nft.UpdatedAt = time.Now()

	return nil
}

func (s *NFTService) MintNFT(ctx context.Context, walletID, collectionAddress, name, description, imageURL string) (*models.NFT, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nftID := fmt.Sprintf("nft_%d", time.Now().UnixNano())

	nft := &models.NFT{
		ID:             nftID,
		WalletID:       walletID,
		CollectionAddr: collectionAddress,
		TokenID:       fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:          name,
		Description:   description,
		ImageURL:      imageURL,
		Owner:         walletID,
		Standard:      "ERC721",
		IsListed:      false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	s.nfts[nftID] = nft

	return nft, nil
}

func (s *NFTService) ListNFT(ctx context.Context, nftID, price string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	nft, ok := s.nfts[nftID]
	if !ok {
		return errors.New("NFT not found")
	}

	nft.IsListed = true
	nft.ListingPrice = price
	nft.UpdatedAt = time.Now()

	return nil
}

func (s *NFTService) DelistNFT(ctx context.Context, nftID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	nft, ok := s.nfts[nftID]
	if !ok {
		return errors.New("NFT not found")
	}

	nft.IsListed = false
	nft.ListingPrice = ""
	nft.UpdatedAt = time.Now()

	return nil
}
