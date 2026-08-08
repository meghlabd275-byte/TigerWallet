// TigerWallet NFT Service - Comprehensive NFT Marketplace and Management
// Supports ERC-721, ERC-1155, Solana NFTs with marketplace, minting, and trading

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port     int    `json:"port"`
	RedisAddr string `json:"redis_addr"`
}

var cfg = Config{
	Port:     8004,
	RedisAddr: "localhost:6379",
}

// ============================================================================
// Data Models
// ============================================================================

type NFTCollection struct {
	ID             string            `json:"id" bson:"_id"`
	Name           string            `json:"name" bson:"name"`
	Symbol         string            `json:"symbol" bson:"symbol"`
	Chain          string            `json:"chain" bson:"chain"`
	ContractAddress string           `json:"contract_address" bson:"contract_address"`
	Owner          string            `json:"owner" bson:"owner"`
	Creator        string            `json:"creator" bson:"creator"`
	Description    string            `json:"description" bson:"description"`
	ImageURL       string            `json:"image_url" bson:"image_url"`
	ExternalURL    string            `json:"external_url" bson:"external_url"`
	Category       string            `json:"category" bson:"category"`
	Standard       string            `json:"standard" bson:"standard"` // erc721, erc1155, spl
	TotalSupply    string            `json:"total_supply" bson:"total_supply"`
	FloorPrice     string            `json:"floor_price" bson:"floor_price"`
	Volume24h      string            `json:"volume_24h" bson:"volume_24h"`
	Sales24h       int               `json:"sales_24h" bson:"sales_24h"`
	Owners         int               `json:"owners" bson:"owners"`
	Verified       bool              `json:"verified" bson:"verified"`
	Featured       bool              `json:"featured" bson:"featured"`
	RoyaltyFee     string            `json:"royalty_fee" bson:"royalty_fee"` // percentage
	Status         string            `json:"status" bson:"status"` // active, paused, sold_out
	CreatedAt      time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" bson:"updatedAt"`
}

type NFT struct {
	ID              string            `json:"id" bson:"_id"`
	CollectionID    string            `json:"collection_id" bson:"collection_id"`
	TokenID         string            `json:"token_id" bson:"token_id"`
	Chain           string            `json:"chain" bson:"chain"`
	ContractAddress string            `json:"contract_address" bson:"contract_address"`
	Owner           string            `json:"owner" bson:"owner"`
	Creator         string            `json:"creator" bson:"creator"`
	Name            string            `json:"name" bson:"name"`
	Description     string            `json:"description" bson:"description"`
	ImageURL        string            `json:"image_url" bson:"image_url"`
	AnimationURL    string            `json:"animation_url" bson:"animation_url"`
	ExternalURL     string            `json:"external_url" bson:"external_url"`
	Attributes      []NFTAttribute    `json:"attributes" bson:"attributes"`
	Edition         int               `json:"edition" bson:"edition"` // for 1155
	Quantity        int               `json:"quantity" bson:"quantity"` // for 1155
	TokenURI        string            `json:"token_uri" bson:"token_uri"`
	IsForSale       bool              `json:"is_for_sale" bson:"is_for_sale"`
	Price           string            `json:"price" bson:"price"`
	PriceToken      string            `json:"price_token" bson:"price_token"` // ETH, USDC, etc
	ListingFee      string            `json:"listing_fee" bson:"listing_fee"`
	LastSalePrice   string            `json:"last_sale_price" bson:"last_sale_price"`
	LastSaleTime    *time.Time       `json:"last_sale_time" bson:"last_sale_time"`
	CreatedAt       time.Time        `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" bson:"updated_at"`
}

type NFTAttribute struct {
	TraitType   string `json:"trait_type" bson:"trait_type"`
	Value       string `json:"value" bson:"value"`
	DisplayType string `json:"display_type" bson:"display_type"`
	Rarity      string `json:"rarity" bson:"rarity"`
}

type NFTListing struct {
	ID              string    `json:"id" bson:"_id"`
	NFTID          string    `json:"nft_id" bson:"nft_id"`
	Seller          string    `json:"seller" bson:"seller"`
	Price           string    `json:"price" bson:"price"`
	PriceToken      string    `json:"price_token" bson:"price_token"`
	Quantity        int       `json:"quantity" bson:"quantity"`
	Status          string    `json:"status" bson:"status"` // active, sold, cancelled, expired
	StartTime       time.Time `json:"start_time" bson:"start_time"`
	EndTime         *time.Time `json:"end_time" bson:"end_time"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
}

type NFTOffer struct {
	ID          string    `json:"id" bson:"_id"`
	NFTID       string    `json:"nft_id" bson:"nft_id"`
	Buyer       string    `json:"buyer" bson:"buyer"`
	Price       string    `json:"price" bson:"price"`
	PriceToken  string    `json:"price_token" bson:"price_token"`
	Quantity    int       `json:"quantity" bson:"quantity"`
	Status      string    `json:"status" bson:"status"` // pending, accepted, rejected, expired
	ExpiredAt   *time.Time `json:"expired_at" bson:"expired_at"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
}

type NFTTransaction struct {
	ID          string    `json:"id" bson:"_id"`
	NFTID      string    `json:"nft_id" bson:"nft_id"`
	CollectionID string `json:"collection_id" bson:"collection_id"`
	Chain       string   `json:"chain" bson:"chain"`
	Seller      string   `json:"seller" bson:"seller"`
	Buyer       string   `json:"buyer" bson:"buyer"`
	Price       string   `json:"price" bson:"price"`
	PriceToken  string   `json:"price_token" bson:"price_token"`
	Fee         string   `json:"fee" bson:"fee"`
	Royalty     string   `json:"royalty" bson:"royalty"`
	TxHash      string   `json:"tx_hash" bson:"tx_hash"`
	Timestamp   time.Time `json:"timestamp" bson:"timestamp"`
}

type NFTAuction struct {
	ID          string    `json:"id" bson:"_id"`
	NFTID       string    `json:"nft_id" bson:"nft_id"`
	Seller      string    `json:"seller" bson:"seller"`
	StartPrice  string    `json:"start_price" bson:"start_price"`
	EndPrice    string    `json:"end_price" bson:"end_price"`
	CurrentBid  string    `json:"current_bid" bson:"current_bid"`
	Bidder      string    `json:"bidder" bson:"bidder"`
	Quantity    int       `json:"quantity" bson:"quantity"`
	Status      string    `json:"status" bson:"status"` // active, ended, cancelled
	StartTime   time.Time `json:"start_time" bson:"start_time"`
	EndTime     time.Time `json:"end_time" bson:"end_time"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
}

// ============================================================================
// NFT Service
// ============================================================================

type NFTService struct {
	redis       *redis.Client
	mu          sync.RWMutex
	collections map[string]*NFTCollection
	nfts        map[string]*NFT
	listings    map[string]*NFTListing
	offers      map[string]*NFTOffer
	transactions map[string]*NFTTransaction
	auctions    map[string]*NFTAuction
}

func NewNFTService() *NFTService {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	ns := &NFTService{
		redis:       rdb,
		collections: make(map[string]*NFTCollection),
		nfts:        make(map[string]*NFT),
		listings:    make(map[string]*NFTListing),
		offers:      make(map[string]*NFTOffer),
		transactions: make(map[string]*NFTTransaction),
		auctions:    make(map[string]*NFTAuction),
	}

	ns.initializeDefaultData()

	return ns
}

func (ns *NFTService) initializeDefaultData() {
	// Initialize collections
	collections := []NFTCollection{
		{
			ID: "bored-ape", Name: "Bored Ape Yacht Club", Symbol: "BAYC", Chain: "ethereum",
			ContractAddress: "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D",
			Owner: "0x000", Creator: "0x000", Description: "The Bored Ape Yacht Club is a collection of 10,000 unique Bored Ape NFTs.",
			ImageURL: "https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfNEuhunv6FNeMFGa9bx6mQ",
			Category: "PFP", Standard: "erc721", TotalSupply: "10000", FloorPrice: "15.5",
			Volume24h: "2500", Sales24h: 50, Owners: 6500, Verified: true, Featured: true,
			RoyaltyFee: "2.5", Status: "active",
		},
		{
			ID: "crypto-punk", Name: "CryptoPunks", Symbol: "PUNK", Chain: "ethereum",
			ContractAddress: "0xb47e3cd837dDF8e4c57F05d70Ab865de6e193BBB",
			Owner: "0x000", Creator: "0x000", Description: "CryptoPunks launched as a fixed set of 10,000 unique collectible characters with proof of ownership stored on the Ethereum blockchain.",
			ImageURL: "https://cryptopunks.app/cryptopunks/cryptopunk001.png",
			Category: "PFP", Standard: "erc721", TotalSupply: "10000", FloorPrice: "45.0",
			Volume24h: "5000", Sales24h: 25, Owners: 4000, Verified: true, Featured: true,
			RoyaltyFee: "0", Status: "active",
		},
		{
			ID: "azuki", Name: "Azuki", Symbol: "AZUKI", Chain: "ethereum",
			ContractAddress: "0xED5AF388653567Af2F388E6224dC7C4b3241C544",
			Owner: "0x000", Creator: "0x000", Description: "Azuki starts with a collection of 10,000 avatars that give you membership access to The Garden.",
			ImageURL: "https://ipfs.io/ipfs/QmYDnL5T3q7k1K7J3KZJ5J5J5J5J5J5J5J5J5J5J5J",
			Category: "PFP", Standard: "erc721", TotalSupply: "10000", FloorPrice: "8.5",
			Volume24h: "1200", Sales24h: 35, Owners: 5500, Verified: true, Featured: true,
			RoyaltyFee: "5", Status: "active",
		},
		{
			ID: "deGods", Name: "DeGods", Symbol: "DGOD", Chain: "solana",
			ContractAddress: "degods1...", Owner: "0x000", Creator: "0x000", Description: "DeGods is a digital art collection of 10,000 utility-enabled NFTs.",
			ImageURL: "https://arweave.io/de gods.png", Category: "PFP", Standard: "spl",
			TotalSupply: "10000", FloorPrice: "250", Volume24h: "15000", Sales24h: 100,
			Owners: 7000, Verified: true, Featured: true, RoyaltyFee: "5", Status: "active",
		},
	}

	for _, col := range collections {
		col.CreatedAt = time.Now()
		col.UpdatedAt = time.Now()
		ns.collections[col.ID] = &col
	}

	// Initialize sample NFTs
	for _, col := range collections {
		for i := 1; i <= 5; i++ {
			nftID := fmt.Sprintf("%s-%d", col.ID, i)
			nft := &NFT{
				ID:              nftID,
				CollectionID:    col.ID,
				TokenID:         fmt.Sprintf("%d", i),
				Chain:           col.Chain,
				ContractAddress: col.ContractAddress,
				Owner:           "0xOwner" + fmt.Sprintf("%d", i),
				Creator:         col.Creator,
				Name:            fmt.Sprintf("%s #%d", col.Name, i),
				Description:     col.Description,
				ImageURL:        col.ImageURL,
				Attributes: []NFTAttribute{
					{TraitType: "Background", Value: "Blue", Rarity: "Common"},
					{TraitType: "Eyes", Value: "Laser", Rarity: "Legendary"},
				},
				IsForSale:   i%2 == 0,
				Price:       col.FloorPrice,
				PriceToken:  "ETH",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			ns.nfts[nftID] = nft
		}
	}
}

// ============================================================================
// API Handlers
// ============================================================================

// Get all collections
func (ns *NFTService) GetCollections(c *gin.Context) {
	category := c.Query("category")
	chain := c.Query("chain")
	featured := c.Query("featured")

	collections := make([]*NFTCollection, 0)
	for _, col := range ns.collections {
		if category != "" && col.Category != category {
			continue
		}
		if chain != "" && col.Chain != chain {
			continue
		}
		if featured == "true" && !col.Featured {
			continue
		}
		collections = append(collections, col)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"collections": collections,
		"total":      len(collections),
	})
}

// Get collection by ID
func (ns *NFTService) GetCollection(c *gin.Context) {
	collectionID := c.Param("id")

	collection, exists := ns.collections[collectionID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"collection": collection,
	})
}

// Get NFTs in collection
func (ns *NFTService) GetCollectionNFTs(c *gin.Context) {
	collectionID := c.Param("id")

	nfts := make([]*NFT, 0)
	for _, nft := range ns.nfts {
		if nft.CollectionID == collectionID {
			nfts = append(nfts, nft)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"nfts":   nfts,
		"total":  len(nfts),
	})
}

// Get NFT by ID
func (ns *NFTService) GetNFT(c *gin.Context) {
	nftID := c.Param("id")

	nft, exists := ns.nfts[nftID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "nft not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"nft":     nft,
	})
}

// Create collection
type CreateCollectionRequest struct {
	Name           string `json:"name" binding:"required"`
	Symbol         string `json:"symbol" binding:"required"`
	Chain          string `json:"chain" binding:"required"`
	ContractAddress string `json:"contract_address" binding:"required"`
	Description    string `json:"description"`
	Category       string `json:"category"`
	Standard       string `json:"standard" binding:"required"`
	RoyaltyFee     string `json:"royalty_fee"`
}

func (ns *NFTService) CreateCollection(c *gin.Context) {
	var req CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collectionID := uuid.New().String()
	collection := &NFTCollection{
		ID:              collectionID,
		Name:            req.Name,
		Symbol:          req.Symbol,
		Chain:           req.Chain,
		ContractAddress: req.ContractAddress,
		Owner:           c.GetString("user_id"),
		Creator:         c.GetString("user_id"),
		Description:     req.Description,
		Category:        req.Category,
		Standard:        req.Standard,
		RoyaltyFee:      req.RoyaltyFee,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	ns.collections[collectionID] = collection

	c.JSON(http.StatusCreated, gin.H{
		"success":      true,
		"collection_id": collectionID,
		"collection":    collection,
	})
}

// Mint NFT
type MintNFTRequest struct {
	CollectionID string   `json:"collection_id" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	ImageURL     string   `json:"image_url" binding:"required"`
	Attributes   []NFTAttribute `json:"attributes"`
	Quantity     int      `json:"quantity"`
	IsForSale    bool     `json:"is_for_sale"`
	Price        string   `json:"price"`
	PriceToken   string   `json:"price_token"`
}

func (ns *NFTService) MintNFT(c *gin.Context) {
	var req MintNFTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate collection
	collection, exists := ns.collections[req.CollectionID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}

	// Generate token ID
	tokenID := fmt.Sprintf("%d", len(ns.nfts)+1)

	nftID := uuid.New().String()
	nft := &NFT{
		ID:              nftID,
		CollectionID:    req.CollectionID,
		TokenID:         tokenID,
		Chain:           collection.Chain,
		ContractAddress: collection.ContractAddress,
		Owner:           c.GetString("user_id"),
		Creator:         c.GetString("user_id"),
		Name:            req.Name,
		Description:     req.Description,
		ImageURL:        req.ImageURL,
		Attributes:      req.Attributes,
		Quantity:        req.Quantity,
		IsForSale:       req.IsForSale,
		Price:           req.Price,
		PriceToken:      req.PriceToken,
		TokenURI:        fmt.Sprintf("ipfs://%s/%s.json", nftID, tokenID),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	ns.nfts[nftID] = nft

	// Update collection
	collection.TotalSupply = fmt.Sprintf("%d", len(ns.nfts))

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"nft_id":    nftID,
		"token_id":  tokenID,
		"token_uri": nft.TokenURI,
		"owner":     nft.Owner,
	})
}

// List NFT for sale
type ListNFTRequest struct {
	NFTID     string `json:"nft_id" binding:"required"`
	Price     string `json:"price" binding:"required"`
	PriceToken string `json:"price_token" binding:"required"`
	Quantity  int    `json:"quantity"`
}

func (ns *NFTService) ListNFT(c *gin.Context) {
	var req ListNFTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate NFT
	nft, exists := ns.nfts[req.NFTID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "nft not found"})
		return
	}

	if nft.Owner != c.GetString("user_id") {
		c.JSON(http.StatusForbidden, gin.H{"error": "not owner"})
		return
	}

	// Create listing
	listingID := uuid.New().String()
	quantity := req.Quantity
	if quantity == 0 {
		quantity = 1
	}

	listing := &NFTListing{
		ID:         listingID,
		NFTID:      req.NFTID,
		Seller:     c.GetString("user_id"),
		Price:      req.Price,
		PriceToken: req.PriceToken,
		Quantity:   quantity,
		Status:     "active",
		StartTime:  time.Now(),
		CreatedAt:  time.Now(),
	}

	ns.listings[listingID] = listing

	// Update NFT
	nft.IsForSale = true
	nft.Price = req.Price
	nft.PriceToken = req.PriceToken

	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"listing_id": listingID,
		"price":      req.Price,
		"price_token": req.PriceToken,
	})
}

// Buy NFT
type BuyNFTRequest struct {
	ListingID string `json:"listing_id" binding:"required"`
}

func (ns *NFTService) BuyNFT(c *gin.Context) {
	var req BuyNFTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate listing
	listing, exists := ns.listings[req.ListingID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	if listing.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "listing not active"})
		return
	}

	// Get NFT
	nft := ns.nfts[listing.NFTID]
	collection := ns.collections[nft.CollectionID]

	// Create transaction
	txID := uuid.New().String()
	tx := &NFTTransaction{
		ID:            txID,
		NFTID:         listing.NFTID,
		CollectionID:  nft.CollectionID,
		Chain:         nft.Chain,
		Seller:        listing.Seller,
		Buyer:         c.GetString("user_id"),
		Price:         listing.Price,
		PriceToken:    listing.PriceToken,
		Fee:           "2.5", // platform fee
		Royalty:       collection.RoyaltyFee,
		TxHash:        "", // not broadcast via RPC; real hash requires on-chain broadcast
		Timestamp:     time.Now(),
	}

	ns.transactions[txID] = tx

	// Update listing
	listing.Status = "sold"

	// Update NFT
	nft.Owner = c.GetString("user_id")
	nft.IsForSale = false
	nft.LastSalePrice = listing.Price
	nft.LastSaleTime = &tx.Timestamp

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"tx_id":       txID,
		"nft_id":      listing.NFTID,
		"buyer":       c.GetString("user_id"),
		"seller":      listing.Seller,
		"price":       listing.Price,
		"price_token": listing.PriceToken,
		"tx_hash":     tx.TxHash,
		"status":      "pending",
	})
}

// Make offer
type MakeOfferRequest struct {
	NFTID      string `json:"nft_id" binding:"required"`
	Price      string `json:"price" binding:"required"`
	PriceToken string `json:"price_token" binding:"required"`
	Quantity   int    `json:"quantity"`
	Duration   int    `json:"duration"` // hours
}

func (ns *NFTService) MakeOffer(c *gin.Context) {
	var req MakeOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate NFT
	if _, exists := ns.nfts[req.NFTID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "nft not found"})
		return
	}

	// Create offer
	offerID := uuid.New().String()
	quantity := req.Quantity
	if quantity == 0 {
		quantity = 1
	}

	expiredAt := time.Now().Add(time.Duration(req.Duration) * time.Hour)

	offer := &NFTOffer{
		ID:         offerID,
		NFTID:      req.NFTID,
		Buyer:      c.GetString("user_id"),
		Price:      req.Price,
		PriceToken: req.PriceToken,
		Quantity:   quantity,
		Status:     "pending",
		ExpiredAt:  &expiredAt,
		CreatedAt:  time.Now(),
	}

	ns.offers[offerID] = offer

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"offer_id": offerID,
		"price":    req.Price,
		"expires":  expiredAt.Unix(),
	})
}

// Get user NFTs
func (ns *NFTService) GetUserNFTs(c *gin.Context) {
	userID := c.Param("user_id")

	nfts := make([]*NFT, 0)
	for _, nft := range ns.nfts {
		if nft.Owner == userID {
			nfts = append(nfts, nft)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"nfts":   nfts,
		"total":  len(nfts),
	})
}

// Get NFT transactions
func (ns *NFTService) GetNFTHistory(c *gin.Context) {
	nftID := c.Param("id")

	txs := make([]*NFTTransaction, 0)
	for _, tx := range ns.transactions {
		if tx.NFTID == nftID {
			txs = append(txs, tx)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"transactions": txs,
		"total":     len(txs),
	})
}

// Get listings
func (ns *NFTService) GetListings(c *gin.Context) {
	nftID := c.Query("nft_id")
	status := c.Query("status")

	listings := make([]*NFTListing, 0)
	for _, listing := range ns.listings {
		if nftID != "" && listing.NFTID != nftID {
			continue
		}
		if status != "" && listing.Status != status {
			continue
		}
		listings = append(listings, listing)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"listings": listings,
		"total":  len(listings),
	})
}

// Search NFTs
func (ns *NFTService) SearchNFTs(c *gin.Context) {
	query := c.Query("q")
	collection := c.Query("collection")
	minPrice := c.Query("min_price")
	maxPrice := c.Query("max_price")

	nfts := make([]*NFT, 0)
	for _, nft := range ns.nfts {
		// Check query
		if query != "" {
			if !strings.Contains(strings.ToLower(nft.Name), strings.ToLower(query)) &&
				!strings.Contains(strings.ToLower(nft.Description), strings.ToLower(query)) {
				continue
			}
		}

		// Check collection
		if collection != "" && nft.CollectionID != collection {
			continue
		}

		// Check price range (simplified)
		if minPrice != "" || maxPrice != "" {
			// Would need proper big number comparison
		}

		nfts = append(nfts, nft)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"nfts":   nfts,
		"total":  len(nfts),
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// ============================================================================
// Middleware
// ============================================================================

func (ns *NFTService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// In production, validate JWT
		c.Set("user_id", "user-"+uuid.New().String()[:8])
		c.Next()
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("TigerWallet NFT Service")
	log.Println("========================")
	log.Printf("Starting on port %d", cfg.Port)

	ns := NewNFTService()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "nft-service",
			"timestamp": time.Now().Unix(),
		})
	})

	// Public routes
	r.GET("/api/v1/nft/collections", ns.GetCollections)
	r.GET("/api/v1/nft/collections/:id", ns.GetCollection)
	r.GET("/api/v1/nft/collections/:id/nfts", ns.GetCollectionNFTs)
	r.GET("/api/v1/nft/nfts/:id", ns.GetNFT)
	r.GET("/api/v1/nft/search", ns.SearchNFTs)
	r.GET("/api/v1/nft/listings", ns.GetListings)

	// Protected routes
	api := r.Group("/api/v1/nft")
	api.Use(ns.AuthMiddleware())
	{
		// Collection
		api.POST("/collections", ns.CreateCollection)

		// NFT
		api.POST("/mint", ns.MintNFT)
		api.POST("/list", ns.ListNFT)
		api.POST("/buy", ns.BuyNFT)
		api.POST("/offer", ns.MakeOffer)

		// User
		api.GET("/users/:user_id/nfts", ns.GetUserNFTs)
		api.GET("/nfts/:id/history", ns.GetNFTHistory)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
