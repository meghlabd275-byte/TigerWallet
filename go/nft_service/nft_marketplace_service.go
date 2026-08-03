/**
 * TigerWallet NFT Marketplace Service
 * 
 * Complete NFT marketplace backend with trading, collections, minting,
 * and auction functionality.
 * Built with Go for high-load distributed operations.
 */

package nft

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Types
// ============================================================================

// NFTCollection represents an NFT collection
type NFTCollection struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Symbol        string            `json:"symbol"`
	Description   string            `json:"description"`
	ContractAddress string          `json:"contract_address"`
	ChainID       uint64            `json:"chain_id"`
	Creator       string            `json:"creator"`
	Owner         string            `json:"owner"`
	RoyaltyFee    string            `json:"royalty_fee"`
	RoyaltyAddress string           `json:"royalty_address"`
	TotalSupply   int               `json:"total_supply"`
	ItemsCount    int               `json:"items_count"`
	OwnersCount   int               `json:"owners_count"`
	FloorPrice    string            `json:"floor_price"`
	Volume24h     string            `json:"volume_24h"`
	VolumeTotal   string            `json:"volume_total"`
	ImageURL      string            `json:"image_url"`
	BannerURL     string            `json:"banner_url"`
	Website       string            `json:"website"`
	Twitter       string            `json:"twitter"`
	Discord       string            `json:"discord"`
	Category      string            `json:"category"`
	IsVerified    bool              `json:"is_verified"`
	IsFlagged     bool              `json:"is_flagged"`
	Status        CollectionStatus  `json:"status"`
	CreatedAt     int64             `json:"created_at"`
	UpdatedAt     int64             `json:"updated_at"`
}

// CollectionStatus represents collection status
type CollectionStatus string

const (
	CollectionStatusActive   CollectionStatus = "active"
	CollectionStatusPaused   CollectionStatus = "paused"
	CollectionStatusFrozen   CollectionStatus = "frozen"
)

// NFTItem represents an NFT item
type NFTItem struct {
	ID             string    `json:"id"`
	CollectionID   string    `json:"collection_id"`
	TokenID        string    `json:"token_id"`
	ContractAddress string   `json:"contract_address"`
	ChainID        uint64    `json:"chain_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	ImageURL       string    `json:"image_url"`
	AnimationURL   string    `json:"animation_url"`
	ExternalURL    string    `json:"external_url"`
	Attributes     []Trait   `json:"attributes"`
	Owner          string    `json:"owner"`
	Creator        string    `json:"creator"`
	RoyaltyFee     string    `json:"royalty_fee"`
	Price          string    `json:"price"`
	IsListed      bool      `json:"is_listed"`
	ListingPrice   string    `json:"listing_price"`
	ListingExpiry  int64     `json:"listing_expiry"`
	LastSalePrice  string    `json:"last_sale_price"`
	LastSaleTime  int64     `json:"last_sale_time"`
	MintTxHash     string    `json:"mint_tx_hash"`
	MintTime      int64     `json:"mint_time"`
	CreatedAt      int64     `json:"created_at"`
	UpdatedAt      int64     `json:"updated_at"`
}

// Trait represents NFT attribute
type Trait struct {
	TraitType   string `json:"trait_type"`
	Value       string `json:"value"`
	DisplayType string `json:"display_type"`
	Rarity      string `json:"rarity"`
}

// NFTSale represents an NFT sale
type NFTSale struct {
	ID            string    `json:"id"`
	ItemID        string    `json:"item_id"`
	CollectionID  string    `json:"collection_id"`
	TokenID       string    `json:"token_id"`
	Seller        string    `json:"seller"`
	Buyer         string    `json:"buyer"`
	Price         string    `json:"price"`
	ProtocolFee   string    `json:"protocol_fee"`
	RoyaltyFee    string    `json:"royalty_fee"`
	PaymentToken  string    `json:"payment_token"`
	TxHash        string    `json:"tx_hash"`
	Timestamp     int64     `json:"timestamp"`
}

// NFTAuction represents an NFT auction
type NFTAuction struct {
	ID            string       `json:"id"`
	ItemID        string       `json:"item_id"`
	CollectionID  string       `json:"collection_id"`
	Seller        string       `json:"seller"`
	StartPrice    string       `json:"start_price"`
	CurrentPrice  string       `json:"current_price"`
	EndPrice      string       `json:"end_price"`
	StartTime     int64        `json:"start_time"`
	EndTime       int64        `json:"end_time"`
	Status        AuctionStatus `json:"status"`
	BidsCount     int          `json:"bids_count"`
	HighestBidder string        `json:"highest_bidder"`
	CreatedAt     int64        `json:"created_at"`
}

// AuctionStatus represents auction status
type AuctionStatus string

const (
	AuctionStatusActive    AuctionStatus = "active"
	AuctionStatusEnded    AuctionStatus = "ended"
	AuctionStatusCancelled AuctionStatus = "cancelled"
)

// NFTBid represents an auction bid
type NFTBid struct {
	ID          string    `json:"id"`
	AuctionID   string    `json:"auction_id"`
	Bidder      string    `json:"bidder"`
	Amount      string    `json:"amount"`
	TxHash      string    `json:"tx_hash"`
	Timestamp   int64     `json:"timestamp"`
}

// NFTListing represents a marketplace listing
type NFTListing struct {
	ID             string    `json:"id"`
	ItemID         string    `json:"item_id"`
	CollectionID   string    `json:"collection_id"`
	Seller         string    `json:"seller"`
	Price          string    `json:"price"`
	PaymentToken   string    `json:"payment_token"`
	Quantity       int       `json:"quantity"`
	StartTime      int64     `json:"start_time"`
	EndTime        int64     `json:"end_time"`
	Status         string    `json:"status"`
	CreatedAt      int64     `json:"created_at"`
}

// NFTMarketplaceService manages NFT marketplace operations
type NFTMarketplaceService struct {
	mu           sync.RWMutex
	collections  map[string]*NFTCollection
	items        map[string]*NFTItem
	sales        map[string]*NFTSale
	auctions     map[string]*NFTAuction
	bids         map[string]*NFTBid
	listings     map[string]*NFTListing
}

// ============================================================================
// Service Methods
// ============================================================================

var (
	nftService     *NFTMarketplaceService
	nftServiceOnce sync.Once
)

// GetNFTService returns the singleton NFT service
func GetNFTService() *NFTMarketplaceService {
	nftServiceOnce.Do(func() {
		nftService = &NFTMarketplaceService{
			collections: make(map[string]*NFTCollection),
			items:       make(map[string]*NFTItem),
			sales:       make(map[string]*NFTSale),
			auctions:    make(map[string]*NFTAuction),
			bids:        make(map[string]*NFTBid),
			listings:    make(map[string]*NFTListing),
		}
	})
	return nftService
}

// ============================================================================
// Collection Operations
// ============================================================================

// CreateCollection creates a new NFT collection
func (s *NFTMarketplaceService) CreateCollection(ctx context.Context, collection *NFTCollection) (*NFTCollection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	collection.ID = "collection_" + uuid.New().String()
	collection.Status = CollectionStatusActive
	collection.TotalSupply = 0
	collection.ItemsCount = 0
	collection.OwnersCount = 0
	collection.FloorPrice = "0"
	collection.Volume24h = "0"
	collection.VolumeTotal = "0"
	collection.CreatedAt = time.Now().Unix()
	collection.UpdatedAt = time.Now().Unix()

	s.collections[collection.ID] = collection
	return collection, nil
}

// GetCollection returns a collection by ID
func (s *NFTMarketplaceService) GetCollection(ctx context.Context, collectionID string) (*NFTCollection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collection, exists := s.collections[collectionID]
	if !exists {
		return nil, fmt.Errorf("collection not found")
	}
	return collection, nil
}

// GetCollectionByAddress returns collection by contract address
func (s *NFTMarketplaceService) GetCollectionByAddress(ctx context.Context, chainID uint64, address string) (*NFTCollection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, collection := range s.collections {
		if collection.ChainID == chainID && collection.ContractAddress == address {
			return collection, nil
		}
	}
	return nil, fmt.Errorf("collection not found")
}

// GetAllCollections returns all collections
func (s *NFTMarketplaceService) GetAllCollections(ctx context.Context, category string, status string) ([]*NFTCollection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*NFTCollection, 0)
	for _, collection := range s.collections {
		if category != "" && collection.Category != category {
			continue
		}
		if status != "" && string(collection.Status) != status {
			continue
		}
		result = append(result, collection)
	}
	return result, nil
}

// UpdateCollection updates a collection
func (s *NFTMarketplaceService) UpdateCollection(ctx context.Context, collection *NFTCollection) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.collections[collection.ID]
	if !exists {
		return fmt.Errorf("collection not found")
	}

	existing.Name = collection.Name
	existing.Description = collection.Description
	existing.ImageURL = collection.ImageURL
	existing.BannerURL = collection.BannerURL
	existing.Website = collection.Website
	existing.Twitter = collection.Twitter
	existing.Discord = collection.Discord
	existing.IsVerified = collection.IsVerified
	existing.UpdatedAt = time.Now().Unix()

	return nil
}

// DeleteCollection deletes a collection
func (s *NFTMarketplaceService) DeleteCollection(ctx context.Context, collectionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.collections[collectionID]; !exists {
		return fmt.Errorf("collection not found")
	}

	delete(s.collections, collectionID)
	return nil
}

// ============================================================================
// NFT Item Operations
// ============================================================================

// MintNFT mints a new NFT
func (s *NFTMarketplaceService) MintNFT(ctx context.Context, item *NFTItem) (*NFTItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify collection exists
	collection, exists := s.collections[item.CollectionID]
	if !exists {
		return nil, fmt.Errorf("collection not found")
	}

	item.ID = "nft_" + uuid.New().String()
	item.IsListed = false
	item.ListingPrice = "0"
	item.Price = "0"
	item.MintTime = time.Now().Unix()
	item.CreatedAt = time.Now().Unix()
	item.UpdatedAt = time.Now().Unix()

	s.items[item.ID] = item

	// Update collection
	collection.ItemsCount++
	collection.TotalSupply++

	return item, nil
}

// GetNFT returns an NFT by ID
func (s *NFTMarketplaceService) GetNFT(ctx context.Context, itemID string) (*NFTItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[itemID]
	if !exists {
		return nil, fmt.Errorf("NFT not found")
	}
	return item, nil
}

// GetNFTByTokenID returns NFT by token ID
func (s *NFTMarketplaceService) GetNFTByTokenID(ctx context.Context, collectionID, tokenID string) (*NFTItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.items {
		if item.CollectionID == collectionID && item.TokenID == tokenID {
			return item, nil
		}
	}
	return nil, fmt.Errorf("NFT not found")
}

// GetNFTsByCollection returns NFTs for a collection
func (s *NFTMarketplaceService) GetNFTsByCollection(ctx context.Context, collectionID string) ([]*NFTItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*NFTItem, 0)
	for _, item := range s.items {
		if item.CollectionID == collectionID {
			result = append(result, item)
		}
	}
	return result, nil
}

// GetNFTsByOwner returns NFTs owned by an address
func (s *NFTMarketplaceService) GetNFTsByOwner(ctx context.Context, owner string) ([]*NFTItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*NFTItem, 0)
	for _, item := range s.items {
		if item.Owner == owner {
			result = append(result, item)
		}
	}
	return result, nil
}

// UpdateNFT updates an NFT
func (s *NFTMarketplaceService) UpdateNFT(ctx context.Context, item *NFTItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.items[item.ID]
	if !exists {
		return fmt.Errorf("NFT not found")
	}

	existing.Name = item.Name
	existing.Description = item.Description
	existing.ImageURL = item.ImageURL
	existing.AnimationURL = item.AnimationURL
	existing.ExternalURL = item.ExternalURL
	existing.Attributes = item.Attributes
	existing.UpdatedAt = time.Now().Unix()

	return nil
}

// TransferNFT transfers NFT ownership
func (s *NFTMarketplaceService) TransferNFT(ctx context.Context, itemID, to string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.items[itemID]
	if !exists {
		return fmt.Errorf("NFT not found")
	}

	item.Owner = to
	item.UpdatedAt = time.Now().Unix()
	return nil
}

// ============================================================================
// Listing Operations
// ============================================================================

// CreateListing creates a new listing
func (s *NFTMarketplaceService) CreateListing(ctx context.Context, listing *NFTListing) (*NFTListing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify NFT exists
	item, exists := s.items[listing.ItemID]
	if !exists {
		return nil, fmt.Errorf("NFT not found")
	}

	if item.Owner != listing.Seller {
		return nil, fmt.Errorf("not the owner")
	}

	if item.IsListed {
		return nil, fmt.Errorf("already listed")
	}

	listing.ID = "listing_" + uuid.New().String()
	listing.Status = "active"
	listing.CreatedAt = time.Now().Unix()

	s.listings[listing.ID] = listing

	// Update item
	item.IsListed = true
	item.ListingPrice = listing.Price
	item.UpdatedAt = time.Now().Unix()

	return listing, nil
}

// CancelListing cancels a listing
func (s *NFTMarketplaceService) CancelListing(ctx context.Context, listingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	listing, exists := s.listings[listingID]
	if !exists {
		return fmt.Errorf("listing not found")
	}

	listing.Status = "cancelled"

	// Update item
	item, exists := s.items[listing.ItemID]
	if exists {
		item.IsListed = false
		item.ListingPrice = "0"
		item.UpdatedAt = time.Now().Unix()
	}

	return nil
}

// ExecuteSale executes a sale
func (s *NFTMarketplaceService) ExecuteSale(ctx context.Context, listingID, buyer, txHash string) (*NFTSale, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	listing, exists := s.listings[listingID]
	if !exists {
		return nil, fmt.Errorf("listing not found")
	}

	if listing.Status != "active" {
		return nil, fmt.Errorf("listing not active")
	}

	item, exists := s.items[listing.ItemID]
	if !exists {
		return nil, fmt.Errorf("NFT not found")
	}

	// Calculate fees
	price, _ := new(big.Int).SetString(listing.Price, 10)
	protocolFee := new(big.Int).Div(price, big.NewInt(100)) // 1%
	royaltyFee := new(big.Int).Div(price, big.NewInt(1000)) // 0.1%

	sellerAmount := new(big.Int).Sub(price, protocolFee)
	sellerAmount.Sub(sellerAmount, royaltyFee)

	// Create sale
	sale := &NFTSale{
		ID:           "sale_" + uuid.New().String(),
		ItemID:       listing.ItemID,
		CollectionID: listing.CollectionID,
		TokenID:      item.TokenID,
		Seller:       listing.Seller,
		Buyer:        buyer,
		Price:        listing.Price,
		ProtocolFee:  protocolFee.String(),
		RoyaltyFee:   royaltyFee.String(),
		PaymentToken: listing.PaymentToken,
		TxHash:       txHash,
		Timestamp:    time.Now().Unix(),
	}

	s.sales[sale.ID] = sale

	// Update listing
	listing.Status = "sold"

	// Update item
	item.Owner = buyer
	item.IsListed = false
	item.ListingPrice = "0"
	item.LastSalePrice = listing.Price
	item.LastSaleTime = time.Now().Unix()
	item.UpdatedAt = time.Now().Unix()

	return sale, nil
}

// GetListing returns a listing
func (s *NFTMarketplaceService) GetListing(ctx context.Context, listingID string) (*NFTListing, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	listing, exists := s.listings[listingID]
	if !exists {
		return nil, fmt.Errorf("listing not found")
	}
	return listing, nil
}

// GetActiveListings returns active listings
func (s *NFTMarketplaceService) GetActiveListings(ctx context.Context, collectionID string) ([]*NFTListing, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*NFTListing, 0)
	for _, listing := range s.listings {
		if listing.Status == "active" {
			if collectionID == "" || listing.CollectionID == collectionID {
				result = append(result, listing)
			}
		}
	}
	return result, nil
}

// ============================================================================
// Auction Operations
// ============================================================================

// CreateAuction creates a new auction
func (s *NFTMarketplaceService) CreateAuction(ctx context.Context, auction *NFTAuction) (*NFTAuction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify NFT exists
	item, exists := s.items[auction.ItemID]
	if !exists {
		return nil, fmt.Errorf("NFT not found")
	}

	auction.ID = "auction_" + uuid.New().String()
	auction.CurrentPrice = auction.StartPrice
	auction.Status = AuctionStatusActive
	auction.BidsCount = 0
	auction.CreatedAt = time.Now().Unix()

	s.auctions[auction.ID] = auction

	// Update item
	item.IsListed = true
	item.ListingPrice = auction.StartPrice
	item.UpdatedAt = time.Now().Unix()

	return auction, nil
}

// PlaceBid places a bid on auction
func (s *NFTMarketplaceService) PlaceBid(ctx context.Context, bid *NFTBid) (*NFTBid, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify auction exists
	auction, exists := s.auctions[bid.AuctionID]
	if !exists {
		return nil, fmt.Errorf("auction not found")
	}

	if auction.Status != AuctionStatusActive {
		return nil, fmt.Errorf("auction not active")
	}

	// Verify bid is higher than current
	currentPrice, _ := new(big.Int).SetString(auction.CurrentPrice, 10)
	bidAmount, _ := new(big.Int).SetString(bid.Amount, 10)

	if bidAmount.Cmp(currentPrice) <= 0 {
		return nil, fmt.Errorf("bid must be higher than current price")
	}

	bid.ID = "bid_" + uuid.New().String()
	bid.Timestamp = time.Now().Unix()

	s.bids[bid.ID] = bid

	// Update auction
	auction.CurrentPrice = bid.Amount
	auction.HighestBidder = bid.Bidder
	auction.BidsCount++

	return bid, nil
}

// EndAuction ends an auction
func (s *NFTMarketplaceService) EndAuction(ctx context.Context, auctionID, txHash string) (*NFTSale, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	auction, exists := s.auctions[auctionID]
	if !exists {
		return nil, fmt.Errorf("auction not found")
	}

	if auction.Status != AuctionStatusActive {
		return nil, fmt.Errorf("auction not active")
	}

	auction.Status = AuctionStatusEnded

	// If there was a winning bid, create sale
	if auction.HighestBidder != "" {
		item, _ := s.items[auction.ItemID]

		sale := &NFTSale{
			ID:           "sale_" + uuid.New().String(),
			ItemID:       auction.ItemID,
			CollectionID: auction.CollectionID,
			TokenID:      item.TokenID,
			Seller:       auction.Seller,
			Buyer:        auction.HighestBidder,
			Price:        auction.CurrentPrice,
			ProtocolFee:   "0",
			RoyaltyFee:   "0",
			PaymentToken:  "0",
			TxHash:       txHash,
			Timestamp:    time.Now().Unix(),
		}

		s.sales[sale.ID] = sale

		// Update item
		if item != nil {
			item.Owner = auction.HighestBidder
			item.IsListed = false
			item.LastSalePrice = auction.CurrentPrice
			item.LastSaleTime = time.Now().Unix()
			item.UpdatedAt = time.Now().Unix()
		}

		return sale, nil
	}

	return nil, nil
}

// GetAuction returns an auction
func (s *NFTMarketplaceService) GetAuction(ctx context.Context, auctionID string) (*NFTAuction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	auction, exists := s.auctions[auctionID]
	if !exists {
		return nil, fmt.Errorf("auction not found")
	}
	return auction, nil
}

// GetActiveAuctions returns active auctions
func (s *NFTMarketplaceService) GetActiveAuctions(ctx context.Context, collectionID string) ([]*NFTAuction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*NFTAuction, 0)
	for _, auction := range s.auctions {
		if auction.Status == AuctionStatusActive {
			if collectionID == "" || auction.CollectionID == collectionID {
				result = append(result, auction)
			}
		}
	}
	return result, nil
}

// ============================================================================
// Utility Methods
// ============================================================================

// GetCollectionStats returns collection statistics
func (s *NFTMarketplaceService) GetCollectionStats(ctx context.Context, collectionID string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collection, exists := s.collections[collectionID]
	if !exists {
		return nil, fmt.Errorf("collection not found")
	}

	// Calculate floor price
	floorPrice := "0"
	owners := make(map[string]bool)
	volumeTotal := big.NewInt(0)

	for _, item := range s.items {
		if item.CollectionID == collectionID {
			if item.ListingPrice != "0" && item.ListingPrice != "" {
				price, _ := new(big.Int).SetString(item.ListingPrice, 10)
				if floorPrice == "0" || price.Cmp(big.NewInt(0)) > 0 {
					if floorPrice == "0" || price.Cmp(new(big.Int).SetString(floorPrice, 10)) < 0 {
						floorPrice = item.ListingPrice
					}
				}
			}
			owners[item.Owner] = true

			if item.LastSalePrice != "" {
				lastSale, _ := new(big.Int).SetString(item.LastSalePrice, 10)
				volumeTotal.Add(volumeTotal, lastSale)
			}
		}
	}

	stats := map[string]string{
		"floor_price":     floorPrice,
		"total_supply":    fmt.Sprintf("%d", collection.TotalSupply),
		"items_count":     fmt.Sprintf("%d", collection.ItemsCount),
		"owners_count":    fmt.Sprintf("%d", len(owners)),
		"volume_total":    volumeTotal.String(),
		"volume_24h":      collection.Volume24h,
	}

	return stats, nil
}

// GetSalesHistory returns sales history
func (s *NFTMarketplaceService) GetSalesHistory(ctx context.Context, collectionID string, limit int) ([]*NFTSale, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*NFTSale, 0)
	for _, sale := range s.sales {
		if collectionID == "" || sale.CollectionID == collectionID {
			result = append(result, sale)
		}
	}

	// Sort by timestamp descending
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Timestamp > result[i].Timestamp {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// ToJSON converts item to JSON
func (i *NFTItem) ToJSON() (string, error) {
	data, err := json.Marshal(i)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
