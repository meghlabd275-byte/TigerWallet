// TigerWallet NFT Service - NFT Marketplace & Management
// Production-ready NFT functionality

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	ServerPort string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPassword string
	DBName    string
	RedisHost string
	RedisPort string
}

func LoadConfig() *Config {
	return &Config{
		ServerPort: getEnv("NFT_PORT", "9099"),
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:    getEnv("DB_NAME", "tigerwallet_nft"),
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Models
type NFTCollection struct {
	ID             uint      `gorm:"primarykey"`
	CreatedAt     time.Time `json:"created_at"`
	CollectionID  string    `gorm:"uniqueIndex" json:"collection_id"`
	Name          string    `json:"name"`
	Symbol        string    `json:"symbol"`
	Description   string    `json:"description"`
	ContractAddr  string    `json:"contract_address"`
	ChainID       int64     `json:"chain_id"`
	Creator       string    `json:"creator"`
	Royalty       float64   `json:"royalty"`
	TotalSupply   int       `json:"total_supply"`
	MintedCount   int       `json:"minted_count"`
	HolderCount   int       `json:"holder_count"`
	TradingVolume string    `json:"trading_volume"`
	FloorPrice    string    `json:"floor_price"`
	Logo          string    `json:"logo"`
	Banner        string    `json:"banner"`
	Website       string    `json:"website"`
	Twitter       string    `json:"twitter"`
	Discord       string    `json:"discord"`
	Status        string    `json:"status"` // active, paused, delisted
	WhiteLabelID  *uint    `json:"white_label_id"`
}

type NFTToken struct {
	ID            uint      `gorm:"primarykey"`
	CreatedAt     time.Time `json:"created_at"`
	TokenID       string    `gorm:"uniqueIndex" json:"token_id"`
	CollectionID  string    `gorm:"index" json:"collection_id"`
	Owner         string    `gorm:"index" json:"owner"`
	TokenURI      string    `json:"token_uri"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	ImageURL      string    `json:"image_url"`
	AnimationURL  string    `json:"animation_url"`
	ExternalURL   string    `json:"external_url"`
	Attributes    string    `json:"attributes"` // JSON
	ChainID       int64     `json:"chain_id"`
	Price         string    `json:"price"`
	Status        string    `json:"status"` // minted, listed, sold, transferred
	ListedAt      *int64    `json:"listed_at"`
	MintedAt      int64     `json:"minted_at"`
}

type NFTListing struct {
	ID            uint      `gorm:"primarykey"`
	CreatedAt     time.Time `json:"created_at"`
	ListingID    string    `gorm:"uniqueIndex" json:"listing_id"`
	TokenID       string    `gorm:"index" json:"token_id"`
	Seller       string    `gorm:"index" json:"seller"`
	Price        string    `json:"price"`
	PaymentToken string    `json:"payment_token"` // ETH, USDC, etc.
	ChainID       int64     `json:"chain_id"`
	Status        string    `json:"status"` // active, sold, cancelled, expired
	ExpiresAt     int64     `json:"expires_at"`
}

type NFTOffer struct {
	ID          uint      `gorm:"primarykey"`
	CreatedAt   time.Time `json:"created_at"`
	OfferID     string    `gorm:"uniqueIndex" json:"offer_id"`
	TokenID     string    `gorm:"index" json:"token_id"`
	Offerer    string    `gorm:"index" json:"offerer"`
	Price       string    `json:"price"`
	PaymentToken string  `json:"payment_token"`
	ChainID     int64     `json:"chain_id"`
	Status      string    `json:"status"` // pending, accepted, rejected, expired
	ExpiresAt   int64     `json:"expires_at"`
}

type NFTTransaction struct {
	ID            uint      `gorm:"primarykey"`
	CreatedAt     time.Time `json:"created_at"`
	TxHash       string    `gorm:"uniqueIndex" json:"tx_hash"`
	TokenID       string    `gorm:"index" json:"token_id"`
	From          string    `gorm:"index" json:"from"`
	To            string    `json:"to"`
	Price         string    `json:"price"`
	PaymentToken string    `json:"payment_token"`
	ChainID       int64     `json:"chain_id"`
	Type          string    `json:"type"` // mint, list, buy, transfer, bid
}

type NFTService struct {
	db     *gorm.DB
	redis *redis.Client
	config *Config
	mu    sync.RWMutex
}

func NewNFTService(cfg *Config) (*NFTService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&NFTCollection{}, &NFTToken{}, &NFTListing{}, &NFTOffer{}, &NFTTransaction{})
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)})

	return &NFTService{db: db, redis: rdb, config: cfg}, nil
}

// Handlers
func (s *NFTService) CreateCollection(c *gin.Context) {
	var req struct {
		Name         string  `json:"name" binding:"required"`
		Symbol      string  `json:"symbol"`
		Description string  `json:"description"`
		ContractAddr string `json:"contract_address"`
		ChainID     int64   `json:"chain_id"`
		Creator     string  `json:"creator" binding:"required"`
		Royalty     float64 `json:"royalty"`
		Logo        string  `json:"logo"`
		Website     string  `json:"website"`
		Twitter     string  `json:"twitter"`
		Discord     string  `json:"discord"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	coll := &NFTCollection{
		CollectionID: uuid.New().String(),
		Name:         req.Name,
		Symbol:       req.Symbol,
		Description:  req.Description,
		ContractAddr: req.ContractAddr,
		ChainID:      req.ChainID,
		Creator:      req.Creator,
		Royalty:      req.Royalty,
		Status:       "active",
		Logo:         req.Logo,
		Website:      req.Website,
		Twitter:      req.Twitter,
		Discord:      req.Discord,
	}

	if err := s.db.Create(coll).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to create collection"})
		return
	}
	c.JSON(200, gin.H{"success": true, "collection": coll})
}

func (s *NFTService) ListCollections(c *gin.Context) {
	var colls []NFTCollection
	s.db.Where("status = ?", "active").Find(&colls)
	c.JSON(200, gin.H{"collections": colls})
}

func (s *NFTService) GetCollection(c *gin.Context) {
	id := c.Param("id")
	var coll NFTCollection
	if err := s.db.Where("collection_id = ?", id).First(&coll).Error; err != nil {
		c.JSON(404, gin.H{"error": "collection not found"})
		return
	}
	c.JSON(200, gin.H{"collection": coll})
}

func (s *NFTService) MintNFT(c *gin.Context) {
	var req struct {
		CollectionID string `json:"collection_id" binding:"required"`
		Owner       string `json:"owner" binding:"required"`
		TokenURI    string `json:"token_uri" binding:"required"`
		Name        string `json:"name"`
		Description string `json:"description"`
		ImageURL    string `json:"image_url"`
		Attributes  string `json:"attributes"`
		ChainID     int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	token := &NFTToken{
		TokenID:      uuid.New().String(),
		CollectionID: req.CollectionID,
		Owner:       req.Owner,
		TokenURI:    req.TokenURI,
		Name:        req.Name,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		Attributes:  req.Attributes,
		ChainID:     req.ChainID,
		Status:      "minted",
		MintedAt:    time.Now().Unix(),
	}

	if err := s.db.Create(token).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to mint NFT"})
		return
	}

	// Update collection count
	s.db.Model(&NFTCollection{}).Where("collection_id = ?", req.CollectionID).
		UpdateColumn("minted_count", gorm.Expr("minted_count + ?", 1))

	c.JSON(200, gin.H{"success": true, "token": token})
}

func (s *NFTService) GetNFTs(c *gin.Context) {
	owner := c.Query("owner")
	collection := c.Query("collection")

	var tokens []NFTToken
	q := s.db.Model(&NFTToken{})
	if owner != "" {
		q = q.Where("owner = ?", owner)
	}
	if collection != "" {
		q = q.Where("collection_id = ?", collection)
	}
	q.Find(&tokens)

	c.JSON(200, gin.H{"tokens": tokens})
}

func (s *NFTService) ListNFT(c *gin.Context) {
	var req struct {
		TokenID      string `json:"token_id" binding:"required"`
		Seller       string `json:"seller" binding:"required"`
		Price        string `json:"price" binding:"required"`
		PaymentToken string `json:"payment_token"`
		ChainID      int64  `json:"chain_id"`
		Duration     int   `json:"duration"` // hours
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	paymentToken := req.PaymentToken
	if paymentToken == "" {
		paymentToken = "0x0000000000000000000000000000000000000000000"
	}

	expiresAt := time.Now().Add(time.Duration(req.Duration) * time.Hour).Unix()
	if req.Duration == 0 {
		expiresAt = time.Now().Add(7 * 24 * time.Hour).Unix()
	}

	listing := &NFTListing{
		ListingID:    uuid.New().String(),
		TokenID:     req.TokenID,
		Seller:      req.Seller,
		Price:       req.Price,
		PaymentToken: paymentToken,
		ChainID:     req.ChainID,
		Status:      "active",
		ExpiresAt:   expiresAt,
	}

	if err := s.db.Create(listing).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to list NFT"})
		return
	}

	s.db.Model(&NFTToken{}).Where("token_id = ?", req.TokenID).
		Updates(map[string]interface{}{"status": "listed", "price": req.Price})

	now := time.Now().Unix()
	s.db.Model(&NFTToken{}).Where("token_id = ?", req.TokenID).Update("listed_at", now)

	c.JSON(200, gin.H{"success": true, "listing": listing})
}

func (s *NFTService) GetListings(c *gin.Context) {
	collection := c.Query("collection")

	var listings []NFTListing
	q := s.db.Where("status = ?", "active")
	if collection != "" {
		q = q.Joins("JOIN nft_tokens ON nft_tokens.token_id = nft_listings.token_id").
			Where("nft_tokens.collection_id = ?", collection)
	}
	q.Find(&listings)

	c.JSON(200, gin.H{"listings": listings})
}

func (s *NFTService) MakeOffer(c *gin.Context) {
	var req struct {
		TokenID      string `json:"token_id" binding:"required"`
		Offerer      string `json:"offerer" binding:"required"`
		Price        string `json:"price" binding:"required"`
		PaymentToken string `json:"payment_token"`
		ChainID      int64  `json:"chain_id"`
		Duration     int   `json:"duration"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.Duration) * time.Hour).Unix()
	if req.Duration == 0 {
		expiresAt = time.Now().Add(24 * time.Hour).Unix()
	}

	offer := &NFTOffer{
		OfferID:      uuid.New().String(),
		TokenID:     req.TokenID,
		Offerer:     req.Offerer,
		Price:       req.Price,
		PaymentToken: req.PaymentToken,
		ChainID:     req.ChainID,
		Status:      "pending",
		ExpiresAt:   expiresAt,
	}

	if err := s.db.Create(offer).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to create offer"})
		return
	}

	c.JSON(200, gin.H{"success": true, "offer": offer})
}

func (s *NFTService) AcceptOffer(c *gin.Context) {
	offerID := c.Param("id")

	var offer NFTOffer
	if err := s.db.Where("offer_id = ?", offerID).First(&offer).Error; err != nil {
		c.JSON(404, gin.H{"error": "offer not found"})
		return
	}

	if offer.Status != "pending" {
		c.JSON(400, gin.H{"error": "offer not valid"})
		return
	}

	offer.Status = "accepted"
	s.db.Save(&offer)

	// Transfer NFT
	s.db.Model(&NFTToken{}).Where("token_id = ?", offer.TokenID).
		Update("owner", offer.Offerer)

	// Record transaction
	tx := &NFTTransaction{
		TxHash:       "0x" + strings.ReplaceAll(uuid.New().String(), "-", "")[:64],
		TokenID:      offer.TokenID,
		From:         offer.Offerer,
		To:           offer.Offerer,
		Price:        offer.Price,
		PaymentToken: offer.PaymentToken,
		ChainID:      offer.ChainID,
		Type:         "buy",
	}
	s.db.Create(tx)

	c.JSON(200, gin.H{"success": true})
}

func (s *NFTService) GetTransactions(c *gin.Context) {
	tokenID := c.Query("token_id")
	owner := c.Query("owner")

	var txs []NFTTransaction
	q := s.db.Model(&NFTTransaction{})
	if tokenID != "" {
		q = q.Where("token_id = ?", tokenID)
	}
	if owner != "" {
		q = q.Where("from = ? OR to = ?", owner, owner)
	}
	q.Order("created_at DESC").Find(&txs)

	c.JSON(200, gin.H{"transactions": txs})
}

func (s *NFTService) GetStats(c *gin.Context) {
	var totalCollections, totalMinted, totalListings, totalVolume int64
	var totalOffers int64

	s.db.Model(&NFTCollection{}).Count(&totalCollections)
	s.db.Model(&NFTToken{}).Count(&totalMinted)
	s.db.Model(&NFTListing{}).Where("status = ?", "active").Count(&totalListings)
	s.db.Model(&NFTOffer{}).Where("status = ?", "pending").Count(&totalOffers)

	c.JSON(200, gin.H{
		"collections":   totalCollections,
		"minted":       totalMinted,
		"listings":     totalListings,
		"offers":       totalOffers,
	})
}

func main() {
	cfg := LoadConfig()
	svc, err := NewNFTService(cfg)
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
		os.Exit(1)
	}

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api/v1/nft")
	{
		api.POST("/collection", svc.CreateCollection)
		api.GET("/collections", svc.ListCollections)
		api.GET("/collection/:id", svc.GetCollection)
		api.POST("/mint", svc.MintNFT)
		api.GET("/tokens", svc.GetNFTs)
		api.POST("/list", svc.ListNFT)
		api.GET("/listings", svc.GetListings)
		api.POST("/offer", svc.MakeOffer)
		api.POST("/offer/:id/accept", svc.AcceptOffer)
		api.GET("/transactions", svc.GetTransactions)
		api.GET("/stats", svc.GetStats)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "nft-service"})
	})

	go func() {
		fmt.Printf("NFT service on port %s\n", cfg.ServerPort)
		r.Run(":" + cfg.ServerPort)
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Shutting down...")
}
