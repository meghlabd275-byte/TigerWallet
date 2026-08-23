/**
 * TigerWallet NFT Marketplace Service
 * Production-ready NFT marketplace with trading, minting, and collection management
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort  string `json:"server_port"`
	DBHost      string `json:"db_host"`
	DBPort      string `json:"db_port"`
	DBUser      string `json:"db_user"`
	DBPassword  string `json:"db_password"`
	DBName      string `json:"db_name"`
	RedisHost   string `json:"redis_host"`
	RedisPort   string `json:"redis_port"`
	JWTSecret   string `json:"jwt_secret"`
	
	// NFT Settings
	PlatformFeePercent float64 `json:"platform_fee_percent"`
	MaxRoyaltyPercent float64 `json:"max_royalty_percent"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort: getEnv("NFT_PORT", "9097"),
		DBHost: getEnv("DB_HOST", "localhost"),
		DBPort: getEnv("DB_PORT", "5432"),
		DBUser: getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName: getEnv("DB_NAME", "tigerwallet"),
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),
		JWTSecret: getEnv("JWT_SECRET", ""),
		PlatformFeePercent: 2.5,
		MaxRoyaltyPercent: 15.0,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Database Models
// ============================================================================

// Collection represents an NFT collection
type NFTCollection struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	
	CollectionID string    `gorm:"uniqueIndex" json:"collection_id"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	Description  string    `json:"description"`
	
	// Contract
	ContractAddress string    `json:"contract_address"`
	ContractType   string    `json:"contract_type"` // ERC721, ERC1155
	ChainID        uint      `json:"chain_id"`
	
	// Creator
	CreatorAddress string    `json:"creator_address"`
	CreatorFee     float64   `json:"creator_fee"` // Royalty percentage
	
	// Media
	LogoURL       string    `json:"logo_url"`
	BannerURL     string    `json:"banner_url"`
	FeaturedURL   string    `json:"featured_url"`
	
	// Stats
	TotalItems    uint      `json:"total_items"`
	TotalOwners   uint      `json:"total_owners"`
	FloorPrice    string    `json:"floor_price"` // In wei
	        VolumeTraded  string    `json:"volume_traded"` // In wei
	
	// Status
	IsVerified    bool      `json:"is_verified"`
	IsActive      bool      `json:"is_active"`
	Category      string    `json:"category"`
	Tags          string    `json:"tags"` // JSON array
}

// NFT Item represents a single NFT
type NFTItem struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	
	ItemID       string    `gorm:"uniqueIndex" json:"item_id"`
	CollectionID string    `gorm:"index" json:"collection_id"`
	TokenID      string    `json:"token_id"`
	
	// Metadata
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ExternalURL  string    `json:"external_url"`
	ImageURL     string    `json:"image_url"`
	AnimationURL string    `json:"animation_url"`
	
	// Attributes
	Attributes   string    `json:"attributes"` // JSON array of traits
	
	// Ownership
	OwnerAddress string    `json:"owner_address"`
	CreatorAddress string  `json:"creator_address"`
	
	// Supply (for ERC1155)
	Supply       uint      `json:"supply"`
	MaxSupply    uint      `json:"max_supply"`
	
	// Pricing
	IsListed     bool      `json:"is_listed"`
	ListingPrice string    `json:"listing_price"` // In wei
	ListingExpiry *time.Time `json:"listing_expiry"`
	
	// Status
	IsActive     bool      `json:"is_active"`
	MintedAt     *time.Time `json:"minted_at"`
	BurnedAt     *time.Time `json:"burned_at"`
}

// Sale represents an NFT sale
type NFTSale struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	
	SaleID       string    `gorm:"uniqueIndex" json:"sale_id"`
	ItemID       string    `json:"item_id"`
	CollectionID string    `json:"collection_id"`
	
	// Parties
	SellerAddress string   `json:"seller_address"`
	BuyerAddress  string   `json:"buyer_address"`
	
	// Price
	Price        string    `json:"price"` // In wei
	PlatformFee   string    `json:"platform_fee"`
	CreatorRoyalty string   `json:"creator_royalty"`
	NetProceeds   string   `json:"net_proceeds"`
	
	// Transaction
	TransactionHash string  `json:"transaction_hash"`
	BlockNumber    uint64  `json:"block_number"`
	
	// Status
	Status        string   `json:"status"` // pending, completed, cancelled
}

// Offer represents a made offer
type NFTOffer struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	
	OfferID      string    `gorm:"uniqueIndex" json:"offer_id"`
	ItemID       string    `json:"item_id"`
	CollectionID string    `json:"collection_id"`
	
	// Offerer
	OffererAddress string  `json:"offerer_address"`
	
	// Terms
	Price         string   `json:"price"` // In wei
	ExpirationTime time.Time `json:"expiration_time"`
	
	// Status
	Status        string   `json:"status"` // active, accepted, expired, cancelled
	AcceptedAt    *time.Time `json:"accepted_at"`
}

// User Collection (user's collected NFTs)
type UserCollection struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	
	UserAddress  string    `gorm:"index" json:"user_address"`
	ItemID       string    `json:"item_id"`
	Quantity     uint      `json:"quantity"` // For ERC1155
	
	AcquiredAt   time.Time `json:"acquired_at"`
}

// Activity tracks all NFT activities
type NFTActivity struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	
	ItemID       string    `json:"item_id"`
	CollectionID string    `json:"collection_id"`
	
	ActivityType string    `json:"activity_type"` // mint, transfer, sale, list, bid, offer
	FromAddress  string    `json:"from_address"`
	ToAddress    string    `json:"to_address"`
	Price        string    `json:"price"` // In wei
	TransactionHash string `json:"transaction_hash"`
}

// ============================================================================
// Service Layer
// ============================================================================

type NFTMarketplaceService struct {
	db      *gorm.DB
	redis   *redis.Client
	config  *Config
	jwtSecret []byte
}

func NewNFTMarketplaceService(cfg *Config) (*NFTMarketplaceService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&NFTCollection{}, &NFTItem{}, &NFTSale{}, &NFTOffer{}, &UserCollection{}, &NFTActivity{})

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		DB: 7,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rdb.Ping(ctx)

	return &NFTMarketplaceService{
		db: db,
		redis: rdb,
		config: cfg,
		jwtSecret: []byte(cfg.JWTSecret),
	}, nil
}

// ============================================================================
// Collection Management
// ============================================================================

type CreateCollectionRequest struct {
	Name          string  `json:"name" binding:"required"`
	Symbol        string  `json:"symbol"`
	Description   string  `json:"description"`
	ContractType  string  `json:"contract_type"`
	ChainID       uint    `json:"chain_id"`
	LogoURL       string  `json:"logo_url"`
	BannerURL     string  `json:"banner_url"`
	Category      string  `json:"category"`
	Tags          string  `json:"tags"`
	CreatorFee    float64 `json:"creator_fee"` // Royalty percentage
}

func (s *NFTMarketplaceService) CreateCollection(c *gin.Context) {
	creatorAddress := c.GetString("address")

	var req CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate royalty
	if req.CreatorFee > s.config.MaxRoyaltyPercent {
		c.JSON(http.StatusBadRequest, gin.H{"error": "royalty too high"})
		return
	}

	collection := NFTCollection{
		CollectionID: "COL-" + uuid.New().String()[:10],
		Name: req.Name,
		Symbol: req.Symbol,
		Description: req.Description,
		ContractType: req.ContractType,
		ChainID: req.ChainID,
		ContractAddress: s.generateContractAddress(),
		CreatorAddress: creatorAddress,
		CreatorFee: req.CreatorFee,
		LogoURL: req.LogoURL,
		BannerURL: req.BannerURL,
		Category: req.Category,
		Tags: req.Tags,
		IsActive: true,
	}

	s.db.Create(&collection)

	c.JSON(http.StatusCreated, collection)
}

func (s *NFTMarketplaceService) generateContractAddress() string {
	hash := sha256.Sum256([]byte(uuid.New().String()))
	return "0x" + hex.EncodeToString(hash[:])[:40]
}

func (s *NFTMarketplaceService) GetCollections(c *gin.Context) {
	category := c.Query("category")
	verified := c.Query("verified")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")

	var collections []NFTCollection
	query := s.db.Model(&NFTCollection{}).Where("is_active = ?", true)

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if verified != "" {
		query = query.Where("is_verified = ?", verified == "true")
	}

	pageNum, _ := strconv.Atoi(page)
	limitNum, _ := strconv.Atoi(limit)
	offset := (pageNum - 1) * limitNum

	query.Offset(offset).Limit(limitNum).Find(&collections)

	c.JSON(http.StatusOK, gin.H{"collections": collections})
}

func (s *NFTMarketplaceService) GetCollection(c *gin.Context) {
	collectionID := c.Param("id")

	var collection NFTCollection
	result := s.db.Where("collection_id = ?", collectionID).First(&collection)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}

	// Get items
	var items []NFTItem
	s.db.Where("collection_id = ? AND is_active = ?", collectionID, true).Find(&items)

	c.JSON(http.StatusOK, gin.H{
		"collection": collection,
		"items": items,
	})
}

// ============================================================================
// NFT Item Management
// ============================================================================

type MintNFTRequest struct {
	CollectionID  string  `json:"collection_id" binding:"required"`
	Name          string  `json:"name" binding:"required"`
	Description   string  `json:"description"`
	ExternalURL   string  `json:"external_url"`
	ImageURL      string  `json:"image_url" binding:"required"`
	AnimationURL  string  `json:"animation_url"`
	Attributes    string  `json:"attributes"`
	Supply        uint    `json:"supply"` // For ERC1155
	MaxSupply     uint    `json:"max_supply"`
}

func (s *NFTMarketplaceService) MintNFT(c *gin.Context) {
	creatorAddress := c.GetString("address")

	var req MintNFTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify collection exists
	var collection NFTCollection
	result := s.db.Where("collection_id = ?", req.CollectionID).First(&collection)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}

	// Generate token ID
	tokenID := uuid.New().String()

	item := NFTItem{
		ItemID: "NFT-" + tokenID[:10],
		CollectionID: req.CollectionID,
		TokenID: tokenID[:8],
		Name: req.Name,
		Description: req.Description,
		ExternalURL: req.ExternalURL,
		ImageURL: req.ImageURL,
		AnimationURL: req.AnimationURL,
		Attributes: req.Attributes,
		OwnerAddress: creatorAddress,
		CreatorAddress: creatorAddress,
		Supply: req.Supply,
		MaxSupply: req.MaxSupply,
		IsActive: true,
		MintedAt: func() *time.Time { t := time.Now(); return &t }(),
	}

	s.db.Create(&item)

	// Update collection stats
	collection.TotalItems++
	s.db.Save(&collection)

	// Record activity
	s.recordActivity(item.ItemID, item.CollectionID, "mint", creatorAddress, creatorAddress, "0")

	c.JSON(http.StatusCreated, item)
}

func (s *NFTMarketplaceService) GetNFT(c *gin.Context) {
	itemID := c.Param("id")

	var item NFTItem
	result := s.db.Where("item_id = ?", itemID).First(&item)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found"})
		return
	}

	// Get collection
	var collection NFTCollection
	s.db.Where("collection_id = ?", item.CollectionID).First(&collection)

	// Get owner
	var ownerCount int64
	s.db.Model(&UserCollection{}).Where("item_id = ?", itemID).Count(&ownerCount)

	c.JSON(http.StatusOK, gin.H{
		"item": item,
		"collection": collection,
		"owner_count": ownerCount,
	})
}

func (s *NFTMarketplaceService) GetUserNFTs(c *gin.Context) {
	address := c.Param("address")

	var userItems []UserCollection
	s.db.Where("user_address = ?", address).Find(&userItems)

	var items []NFTItem
	for _, ui := range userItems {
		var item NFTItem
		s.db.Where("item_id = ?", ui.ItemID).First(&item)
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ============================================================================
// Listing & Sale
// ============================================================================

type ListNFTRequest struct {
	ItemID   string  `json:"item_id" binding:"required"`
	Price    string  `json:"price" binding:"required"` // In wei
	Duration uint    `json:"duration"` // Hours, default 168 (7 days)
}

func (s *NFTMarketplaceService) ListNFT(c *gin.Context) {
	sellerAddress := c.GetString("address")

	var req ListNFTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify ownership
	var item NFTItem
	result := s.db.Where("item_id = ? AND owner_address = ?", req.ItemID, sellerAddress).First(&item)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found or not owned"})
		return
	}

	// Check if already listed
	if item.IsListed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NFT already listed"})
		return
	}

	// Calculate listing expiry
	duration := req.Duration
	if duration == 0 {
		duration = 168 // 7 days
	}
	expiry := time.Now().Add(time.Hour * time.Duration(duration))

	// Update listing
	item.IsListed = true
	item.ListingPrice = req.Price
	item.ListingExpiry = &expiry
	s.db.Save(&item)

	// Record activity
	s.recordActivity(item.ItemID, item.CollectionID, "list", sellerAddress, "", req.Price)

	c.JSON(http.StatusOK, gin.H{
		"message": "NFT listed successfully",
		"item": item,
	})
}

type BuyNFTRequest struct {
	ItemID string `json:"item_id" binding:"required"`
}

func (s *NFTMarketplaceService) BuyNFT(c *gin.Context) {
	buyerAddress := c.GetString("address")

	var req BuyNFTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get item
	var item NFTItem
	result := s.db.Where("item_id = ? AND is_listed = ?", req.ItemID, true).First(&item)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found or not listed"})
		return
	}

	// Check listing expiry
	if item.ListingExpiry != nil && time.Now().After(*item.ListingExpiry) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "listing expired"})
		return
	}

	// Get collection for royalty
	var collection NFTCollection
	s.db.Where("collection_id = ?", item.CollectionID).First(&collection)

	// Calculate fees
	price := new(big.Int)
	price.SetString(item.ListingPrice, 10)

	platformFee := new(big.Float).Mul(
		new(big.Float).SetInt(price),
		big.NewFloat(s.config.PlatformFeePercent/100),
	)
	platformFeeInt, _ := platformFee.Int(nil)

	creatorRoyalty := new(big.Float).Mul(
		new(big.Float).SetInt(price),
		big.NewFloat(collection.CreatorFee/100),
	)
	creatorRoyaltyInt, _ := creatorRoyalty.Int(nil)

	netProceeds := new(big.Int).Sub(price, platformFeeInt)
	netProceeds = new(big.Int).Sub(netProceeds, creatorRoyaltyInt)

	// Create sale record
	sale := NFTSale{
		SaleID: "SALE-" + uuid.New().String()[:10],
		ItemID: item.ItemID,
		CollectionID: item.CollectionID,
		SellerAddress: item.OwnerAddress,
		BuyerAddress: buyerAddress,
		Price: item.ListingPrice,
		PlatformFee: platformFeeInt.String(),
		CreatorRoyalty: creatorRoyaltyInt.String(),
		NetProceeds: netProceeds.String(),
		Status: "completed",
		TransactionHash: "0x" + uuid.New().String(),
	}
	s.db.Create(&sale)

	// Update ownership
	oldOwner := item.OwnerAddress
	item.OwnerAddress = buyerAddress
	item.IsListed = false
	item.ListingPrice = "0"
	s.db.Save(&item)

	// Update user collection
	var userCol UserCollection
	result = s.db.Where("user_address = ? AND item_id = ?", buyerAddress, item.ItemID).First(&userCol)
	if result.Error != nil {
		userCol = UserCollection{
			UserAddress: buyerAddress,
			ItemID: item.ItemID,
			Quantity: 1,
			AcquiredAt: time.Now(),
		}
		s.db.Create(&userCol)
	} else {
		userCol.Quantity++
		s.db.Save(&userCol)
	}

	// Record activity
	s.recordActivity(item.ItemID, item.CollectionID, "sale", oldOwner, buyerAddress, item.ListingPrice)

	// Update collection volume
	volume := new(big.Int)
	volume.SetString(collection.VolumeTraded, 10)
	volume.Add(volume, price)
	collection.VolumeTraded = volume.String()
	s.db.Save(&collection)

	c.JSON(http.StatusOK, gin.H{
		"message": "Purchase successful",
		"sale": sale,
	})
}

func (s *NFTMarketplaceService) CancelListing(c *gin.Context) {
	ownerAddress := c.GetString("address")
	itemID := c.Param("id")

	var item NFTItem
	result := s.db.Where("item_id = ? AND owner_address = ?", itemID, ownerAddress).First(&item)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found"})
		return
	}

	if !item.IsListed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NFT not listed"})
		return
	}

	item.IsListed = false
	item.ListingPrice = "0"
	item.ListingExpiry = nil
	s.db.Save(&item)

	s.recordActivity(item.ItemID, item.CollectionID, "list_cancel", ownerAddress, "", "0")

	c.JSON(http.StatusOK, gin.H{"message": "Listing cancelled"})
}

// ============================================================================
// Offers
// ============================================================================

type MakeOfferRequest struct {
	ItemID    string    `json:"item_id" binding:"required"`
	Price     string    `json:"price" binding:"required"`
	ExpiresIn uint      `json:"expires_in"` // Hours
}

func (s *NFTMarketplaceService) MakeOffer(c *gin.Context) {
	offererAddress := c.GetString("address")

	var req MakeOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify item exists
	var item NFTItem
	result := s.db.Where("item_id = ?", req.ItemID).First(&item)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found"})
		return
	}

	// Calculate expiry
	expiresIn := req.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 168 // 7 days
	}
	expiry := time.Now().Add(time.Hour * time.Duration(expiresIn))

	offer := NFTOffer{
		OfferID: "OFFER-" + uuid.New().String()[:10],
		ItemID: req.ItemID,
		CollectionID: item.CollectionID,
		OffererAddress: offererAddress,
		Price: req.Price,
		ExpirationTime: expiry,
		Status: "active",
	}

	s.db.Create(&offer)

	s.recordActivity(item.ItemID, item.CollectionID, "offer", offererAddress, "", req.Price)

	c.JSON(http.StatusCreated, offer)
}

func (s *NFTMarketplaceService) AcceptOffer(c *gin.Context) {
	ownerAddress := c.GetString("address")
	offerID := c.Param("id")

	var offer NFTOffer
	result := s.db.Where("offer_id = ? AND status = ?", offerID, "active").First(&offer)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "offer not found"})
		return
	}

	// Verify ownership
	var item NFTItem
	result = s.db.Where("item_id = ? AND owner_address = ?", offer.ItemID, ownerAddress).First(&item)
	if result.Error != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not the owner"})
		return
	}

	// Check expiry
	if time.Now().After(offer.ExpirationTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offer expired"})
		return
	}

	// Execute sale (simplified - would do full transaction in production)
	now := time.Now()
	offer.Status = "accepted"
	offer.AcceptedAt = &now
	s.db.Save(&offer)

	// Update item
	item.OwnerAddress = offer.OffererAddress
	item.IsListed = false
	s.db.Save(&item)

	s.recordActivity(item.ItemID, item.CollectionID, "offer_accept", ownerAddress, offer.OffererAddress, offer.Price)

	c.JSON(http.StatusOK, gin.H{"message": "Offer accepted"})
}

// ============================================================================
// Activity & Analytics
// ============================================================================

func (s *NFTMarketplaceService) GetActivity(c *gin.Context) {
	collectionID := c.Query("collection_id")
	activityType := c.Query("type")
	limit := c.DefaultQuery("limit", "50")

	var activities []NFTActivity
	query := s.db.Model(&NFTActivity{})

	if collectionID != "" {
		query = query.Where("collection_id = ?", collectionID)
	}
	if activityType != "" {
		query = query.Where("activity_type = ?", activityType)
	}

	limitNum, _ := strconv.Atoi(limit)
	query.Order("created_at DESC").Limit(limitNum).Find(&activities)

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

func (s *NFTMarketplaceService) GetTrending(c *gin.Context) {
	// Get collections by volume
	var collections []NFTCollection
	s.db.Where("is_active = ?", true).Order("volume_traded DESC").Limit(10).Find(&collections)

	// Get recently sold
	var recentSales []NFTSale
	s.db.Order("created_at DESC").Limit(10).Find(&recentSales)

	c.JSON(http.StatusOK, gin.H{
		"trending_collections": collections,
		"recent_sales": recentSales,
	})
}

func (s *NFTMarketplaceService) recordActivity(itemID, collectionID, activityType, from, to, price string) {
	activity := NFTActivity{
		ItemID: itemID,
		CollectionID: collectionID,
		ActivityType: activityType,
		FromAddress: from,
		ToAddress: to,
		Price: price,
		TransactionHash: "0x" + uuid.New().String(),
	}
	s.db.Create(&activity)
}

// ============================================================================
// Search & Filter
// ============================================================================

func (s *NFTMarketplaceService) Search(c *gin.Context) {
	query := c.Query("q")
	collectionID := c.Query("collection")
	attributes := c.Query("attributes")
	minPrice := c.Query("min_price")
	maxPrice := c.Query("max_price")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")

	var items []NFTItem
	db := s.db.Model(&NFTItem{}).Where("is_active = ?", true)

	if query != "" {
		db = db.Where("name ILIKE ? OR description ILIKE ?", "%"+query+"%", "%"+query+"%")
	}
	if collectionID != "" {
		db = db.Where("collection_id = ?", collectionID)
	}
	if minPrice != "" {
		db = db.Where("listing_price >= ?", minPrice)
	}
	if maxPrice != "" {
		db = db.Where("listing_price <= ?", maxPrice)
	}

	pageNum, _ := strconv.Atoi(page)
	limitNum, _ := strconv.Atoi(limit)
	offset := (pageNum - 1) * limitNum

	db.Offset(offset).Limit(limitNum).Find(&items)

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ============================================================================
// Middleware
// ============================================================================

func (s *NFTMarketplaceService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, _ := token.Claims.(jwt.MapClaims)
		// In production, extract address from JWT
		c.Set("address", "0x"+hex.EncodeToString([]byte(claims["sub"].(string)))[:40])
		c.Next()
	}
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewNFTMarketplaceService(config)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	router := gin.Default()
	router.Use(gin.Recovery())

	// CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "nft-marketplace"})
	})

	// Public routes
	api := router.Group("/api/v1")
	{
		api.GET("/collections", service.GetCollections)
		api.GET("/collections/:id", service.GetCollection)
		api.GET("/items/:id", service.GetNFT)
		api.GET("/users/:address/nfts", service.GetUserNFTs)
		api.GET("/activity", service.GetActivity)
		api.GET("/trending", service.GetTrending)
		api.GET("/search", service.Search)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(service.AuthMiddleware())
	{
		protected.POST("/collections", service.CreateCollection)
		protected.POST("/mint", service.MintNFT)
		protected.POST("/list", service.ListNFT)
		protected.POST("/buy", service.BuyNFT)
		protected.DELETE("/items/:id/list", service.CancelListing)
		protected.POST("/offers", service.MakeOffer)
		protected.POST("/offers/:id/accept", service.AcceptOffer)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("NFT Marketplace starting on port %s", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down...")
}
