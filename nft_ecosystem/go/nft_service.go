// NFT Management Service - Go Implementation
// Handles NFT minting, transfer, trading, and viewing

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Configuration
type Config struct {
	ServerPort    string `json:"server_port"`
	DBHost       string `json:"db_host"`
	DBPort       string `json:"db_port"`
	DBUser       string `json:"db_user"`
	DBPassword   string `json:"db_password"`
	DBName       string `json:"db_name"`
	RedisHost    string `json:"redis_host"`
	RedisPort    string `json:"redis_port"`
	ChainRPCURL  string `json:"chain_rpc_url"`
	PrivateKey  string `json:"private_key"` // Encrypted
}

// NFT Model
type NFT struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TokenID        string    `gorm:"uniqueIndex" json:"token_id"`
	ContractAddress string   `gorm:"index" json:"contract_address"`
	OwnerAddress  string    `gorm:"index" json:"owner_address"`
	Name          string    `json:"name"`
	Symbol        string    `json:"symbol"`
	Description   string    `json:"description"`
	ImageURL      string    `json:"image_url"`
	AnimationURL string    `json:"animation_url"`
	ExternalURL  string    `json:"external_url"`
	Attributes   JSON      `gorm:"type:jsonb" json:"attributes"`
	URI          string    `json:"uri"`
	ChainID       int64     `json:"chain_id"`
	MintedAt      time.Time `json:"minted_at"`
	TransferredAt time.Time `json:"transferred_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NFTTransfer Model
type NFTTransfer struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TransactionHash string   `gorm:"uniqueIndex" json:"transaction_hash"`
	TokenID        string   `json:"token_id"`
	ContractAddress string  `json:"contract_address"`
	FromAddress   string  `json:"from_address"`
	ToAddress     string   `json:"to_address"`
	Amount        string   `json:"amount"`
	ChainID       int64    `json:"chain_id"`
	BlockNumber   int64    `json:"block_number"`
	Status        string   `json:"status"` // pending, confirmed, failed
	GasUsed        string   `json:"gas_used"`
	GasPrice      string   `json:"gas_price"`
	CreatedAt     time.Time `json:"created_at"`
	ConfirmedAt   time.Time `json:"confirmed_at"`
}

// NFTOffer Model
type NFTOffer struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	OfferID       string    `gorm:"uniqueIndex" json:"offer_id"`
	TokenID       string    `json:"token_id"`
	ContractAddress string   `json:"contract_address"`
	SellerAddress string   `json:"seller_address"`
	BuyerAddress string   `json:"buyer_address"`
	Price        string    `json:"price"`
	PriceToken   string    `json:"price_token"`
	ChainID      int64     `json:"chain_id"`
	Status       string    `json:"status"` // open, cancelled, accepted, completed
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Collection Model
type Collection struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ContractAddress string    `gorm:"uniqueIndex" json:"contract_address"`
	Name          string    `json:"name"`
	Symbol        string    `json:"symbol"`
	Description  string    `json:"description"`
	ImageURL      string    `json:"image_url"`
	ExternalURL  string    `json:"external_url"`
	Creator      string    `json:"creator"`
	ChainID       int64     `json:"chain_id"`
	TotalSupply  int64     `json:"total_supply"`
	IsVerified   bool      `json:"is_verified"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// JSON wrapper for PostgreSQL JSONB
type JSON map[string]interface{}

// NFTService main struct
type NFTService struct {
	db           *gorm.DB
	redis        *redis.Client
	config       Config
	chainClient  ChainClient
	signer      NFTSigner
	mu          sync.RWMutex
}

// ChainClient interface for blockchain interaction
type ChainClient interface {
	GetNFTMetadata(ctx context.Context, contractAddr, tokenID string) (*NFTMetadata, error)
	GetOwnerOf(ctx context.Context, contractAddr, tokenID string) (string, error)
	TransferNFT(ctx context.Context, from, to, contractAddr, tokenID string, amount *big.Int) (string, error)
	MintNFT(ctx context.Context, to, contractAddr, uri string) (string, error)
	SetApprovalForAll(ctx context.Context, owner, operator string, approved bool) (string, error)
}

// NFTMetadata from chain
type NFTMetadata struct {
	TokenID        string
	Name          string
	Symbol        string
	Description  string
	Image        string
	AnimationURL string
	ExternalURL  string
	Attributes   []map[string]interface{}
}

// NewNFTService creates new NFT service
func NewNFTService(cfg Config) (*NFTService, error) {
	// Initialize database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate
	err = db.AutoMigrate(&NFT{}, &NFTTransfer{}, &NFTOffer{}, &Collection{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	// Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// Initialize signer (Rust bridge)
	signer := NewRustNFTSigner(cfg.PrivateKey)

	return &NFTService{
		db:          db,
		redis:       rdb,
		config:      cfg,
		chainClient: NewEVMChainClient(cfg.ChainRPCURL),
		signer:      signer,
	}, nil
}

// API Handlers

// GetNFTsByOwner returns all NFTs for an address
func (s *NFTService) GetNFTsByOwner(c *gin.Context) {
	owner := c.Param("address")
	chainID := c.Query("chain_id")
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	var nfts []NFT
	query := s.db.Where("owner_address = ?", owner)

	if chainID != "" {
		query = query.Where("chain_id = ?", chainID)
	}

	err := query.Limit(parseInt(limit)).Offset(parseInt(offset)).Find(&nfts).Error
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"nfts":   nfts,
		"total":  len(nfts),
		"limit":  limit,
		"offset": offset,
	})
}

// GetNFTDetails returns details of a specific NFT
func (s *NFTService) GetNFTDetails(c *gin.Context) {
	contractAddr := c.Param("contract")
	tokenID := c.Param("token_id")

	var nft NFT
	err := s.db.Where("contract_address = ? AND token_id = ?", contractAddr, tokenID).First(&nft).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Try to fetch from chain
			metadata, err := s.chainClient.GetNFTMetadata(c.Request.Context(), contractAddr, tokenID)
			if err != nil {
				c.JSON(404, gin.H{"error": "NFT not found"})
				return
			}
			c.JSON(200, metadata)
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, nft)
}

// TransferNFT transfers an NFT to another address
func (s *NFTService) TransferNFT(c *gin.Context) {
	var req struct {
		From           string `json:"from" binding:"required"`
		To             string `json:"to" binding:"required"`
		ContractAddress string `json:"contract_address" binding:"required"`
		TokenID        string `json:"token_id" binding:"required"`
		Amount        string `json:"amount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	amount := big.NewInt(1)
	if req.Amount != "" {
		amount.SetString(req.Amount, 10)
	}

	// Sign transaction via Rust signer
	txHash, err := s.signer.SignAndSendNFTTransfer(c.Request.Context(), req.From, req.To, req.ContractAddress, req.TokenID, amount)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Record transfer in database
	transfer := NFTTransfer{
		TransactionHash: txHash,
		TokenID:       req.TokenID,
		ContractAddress: req.ContractAddress,
		FromAddress:   req.From,
		ToAddress:    req.To,
		Amount:      amount.String(),
		ChainID:      1,
		Status:      "pending",
	}

	s.db.Create(&transfer)

	c.JSON(200, gin.H{
		"transaction_hash": txHash,
		"status":        "pending",
	})
}

// MintNFT mints a new NFT
func (s *NFTService) MintNFT(c *gin.Context) {
	var req struct {
		To              string `json:"to" binding:"required"`
		ContractAddress string `json:"contract_address" binding:"required"`
		URI             string `json:"uri" binding:"required"`
		Name            string `json:"name"`
		Description     string `json:"description"`
		ImageURL       string `json:"image_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Sign mint transaction via Rust signer
	txHash, tokenID, err := s.signer.SignAndMintNFT(c.Request.Context(), req.To, req.ContractAddress, req.URI)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Record in database
	nft := NFT{
		TokenID:         tokenID,
		ContractAddress: req.ContractAddress,
		OwnerAddress:   req.To,
		Name:          req.Name,
		Description:   req.Description,
		ImageURL:     req.ImageURL,
		URI:          req.URI,
		ChainID:       1,
		MintedAt:      time.Now(),
	}

	s.db.Create(&nft)

	c.JSON(200, gin.H{
		"transaction_hash": txHash,
		"token_id":      tokenID,
		"status":       "minted",
	})
}

// GetCollections returns all collections
func (s *NFTService) GetCollections(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	var collections []Collection
	err := s.db.Limit(parseInt(limit)).Offset(parseInt(offset)).Find(&collections).Error
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, collections)
}

// GetCollectionNFTs returns NFTs in a collection
func (s *NFTService) GetCollectionNFTs(c *gin.Context) {
	contractAddr := c.Param("contract")
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	var nfts []NFT
	err := s.db.Where("contract_address = ?", contractAddr).
		Limit(parseInt(limit)).Offset(parseInt(offset)).
		Find(&nfts).Error
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, nfts)
}

// CreateOffer creates an NFT offer
func (s *NFTService) CreateOffer(c *gin.Context) {
	var req struct {
		TokenID         string `json:"token_id" binding:"required"`
		ContractAddress string `json:"contract_address" binding:"required"`
		SellerAddress  string `json:"seller_address" binding:"required"`
		Price         string `json:"price" binding:"required"`
		PriceToken    string `json:"price_token" binding:"required"`
		ExpiresAt     int64  `json:"expires_at" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	offerID := generateOfferID(req.ContractAddress, req.TokenID, req.SellerAddress)

	offer := NFTOffer{
		OfferID:        offerID,
		TokenID:        req.TokenID,
		ContractAddress: req.ContractAddress,
		SellerAddress: req.SellerAddress,
		Price:        req.Price,
		PriceToken:   req.PriceToken,
		ChainID:      1,
		Status:       "open",
		ExpiresAt:    time.Unix(req.ExpiresAt, 0),
	}

	s.db.Create(&offer)

	// Cache in Redis for fast access
	s.redis.Set(c.Request.Context(), fmt.Sprintf("offer:%s", offerID), offerID, 24*time.Hour)

	c.JSON(200, offer)
}

// AcceptOffer accepts an NFT offer
func (s *NFTService) AcceptOffer(c *gin.Context) {
	offerID := c.Param("offer_id")
	buyerAddress := c.PostForm("buyer_address")

	var offer NFTOffer
	err := s.db.Where("offer_id = ?", offerID).First(&offer).Error
	if err != nil {
		c.JSON(404, gin.H{"error": "Offer not found"})
		return
	}

	if offer.Status != "open" {
		c.JSON(400, gin.H{"error": "Offer is not available"})
		return
	}

	// Execute transfer via Rust signer
	txHash, err := s.signer.SignAndExecuteNFTSale(c.Request.Context(), offer.ContractAddress, offer.TokenID, offer.SellerAddress, buyerAddress, offer.Price, offer.PriceToken)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Update offer status
	offer.Status = "completed"
	offer.BuyerAddress = buyerAddress
	s.db.Save(&offer)

	// Update NFT owner
	s.db.Model(&NFT{}).Where("contract_address = ? AND token_id = ?", offer.ContractAddress, offer.TokenID).
		Update("owner_address", buyerAddress)

	c.JSON(200, gin.H{
		"transaction_hash": txHash,
		"status":        "completed",
	})
}

// GetUserOffers returns offers for a user
func (s *NFTService) GetUserOffers(c *gin.Context) {
	address := c.Param("address")
	side := c.DefaultQuery("side", "all") // sell, buy, all

	var offers []NFTOffer
	query := s.db.Where("1 = 1")

	switch side {
	case "sell":
		query = query.Where("seller_address = ?", address)
	case "buy":
		query = query.Where("buyer_address = ?", address)
	default:
		query = query.Where("seller_address = ? OR buyer_address = ?", address, address)
	}

	err := query.Find(&offers).Error
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, offers)
}

// BatchGetNFTs returns NFTs for multiple addresses
func (s *NFTService) BatchGetNFTs(c *gin.Context) {
	var req struct {
		Addresses []string `json:"addresses" binding:"required"`
		ChainID   int64   `json:"chain_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	chainID := req.ChainID
	if chainID == 0 {
		chainID = 1
	}

	var nfts []NFT
	err := s.db.Where("owner_address IN ? AND chain_id = ?", req.Addresses, chainID).Find(&nfts).Error
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Group by owner
	result := make(map[string][]NFT)
	for _, nft := range nfts {
		result[nft.OwnerAddress] = append(result[nft.OwnerAddress], nft)
	}

	c.JSON(200, result)
}

// GetFloorPrice returns floor price for a collection
func (s *NFTService) GetFloorPrice(c *gin.Context) {
	contractAddr := c.Param("contract")

	// Try cache first
	ctx := c.Request.Context()
	cacheKey := fmt.Sprintf("floor_price:%s", contractAddr)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		c.JSON(200, gin.H{"floor_price": cached})
		return
	}

	// Calculate from recent sales
	var offers []NFTOffer
	err = s.db.Where("contract_address = ? AND status = ?", contractAddr, "completed").
		Order("price DESC").Limit(10).Find(&offers).Error
	if err != nil || len(offers) == 0 {
		c.JSON(200, gin.H{"floor_price": "0"})
		return
	}

	// Simple floor price - lowest recent sale
	floorPrice := offers[len(offers)-1].Price

	s.redis.Set(ctx, cacheKey, floorPrice, 15*time.Minute)

	c.JSON(200, gin.H{"floor_price": floorPrice})
}

// SearchNFTs searches NFTs by name/attribute
func (s *NFTService) SearchNFTs(c *gin.Context) {
	query := c.Query("q")
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	var nfts []NFT
	err := s.db.Where("name ILIKE ? OR description ILIKE ?", "%"+query+"%", "%"+query+"%").
		Limit(parseInt(limit)).Offset(parseInt(offset)).
		Find(&nfts).Error
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, nfts)
}

// GetTrendingCollections returns trending collections
func (s *NFTService) GetTrendingCollections(c *gin.Context) {
	limit := c.DefaultQuery("limit", "20")

	var collections []Collection
	err := s.db.Order("total_supply DESC").Limit(parseInt(limit)).Find(&collections).Error
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, collections)
}

// Utility functions

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func generateOfferID(contract, tokenID, seller string) string {
	data := fmt.Sprintf("%s:%s:%s:%d", contract, tokenID, seller, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

// EVMChainClient implements ChainClient for EVM chains
type EVMChainClient struct {
	rpcURL string
}

func NewEVMChainClient(rpcURL string) *EVMChainClient {
	return &EVMChainClient{rpcURL: rpcURL}
}

func (c *EVMChainClient) GetNFTMetadata(ctx context.Context, contractAddr, tokenID string) (*NFTMetadata, error) {
	// This would make actual RPC calls to the blockchain
	// For demo, return placeholder
	return &NFTMetadata{
		TokenID:       tokenID,
		Name:         "NFT #" + tokenID,
		Symbol:       "NFT",
		Description:  "A unique digital asset",
		Image:       "",
		Attributes:   []map[string]interface{}{},
	}, nil
}

func (c *EVMChainClient) GetOwnerOf(ctx context.Context, contractAddr, tokenID string) (string, error) {
	return "", nil
}

func (c *EVMChainClient) TransferNFT(ctx context.Context, from, to, contractAddr, tokenID string, amount *big.Int) (string, error) {
	return "", nil
}

func (c *EVMChainClient) MintNFT(ctx context.Context, to, contractAddr, uri string) (string, error) {
	return "", nil
}

func (c *EVMChainClient) SetApprovalForAll(ctx context.Context, owner, operator string, approved bool) (string, error) {
	return "", nil
}

// Main entry point
func main() {
	// Load configuration
	cfg := Config{
		ServerPort:   getEnv("NFT_SERVICE_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "nft_db"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
		ChainRPCURL: getEnv("CHAIN_RPC_URL", ""),
		PrivateKey: getEnv("PRIVATE_KEY", ""),
	}

	// Initialize service
	service, err := NewNFTService(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize service: %v\n", err)
		os.Exit(1)
	}

	// Setup Gin router
	r := gin.Default()

	// NFT routes
	nft := r.Group("/api/v1/nfts")
	{
		nft.GET("/owner/:address", service.GetNFTsByOwner)
		nft.GET("/:contract/:token_id", service.GetNFTsDetails)
		nft.POST("/transfer", service.TransferNFT)
		nft.POST("/mint", service.MintNFT)
		nft.GET("/search", service.SearchNFTs)
		nft.POST("/batch", service.BatchGetNFTs)
	}

	// Collection routes
	collections := r.Group("/api/v1/collections")
	{
		collections.GET("", service.GetCollections)
		collections.GET("/:contract", service.GetCollectionNFTs)
		collections.GET("/:contract/floor", service.GetFloorPrice)
		collections.GET("/trending", service.GetTrendingCollections)
	}

	// Offer routes
	offers := r.Group("/api/v1/offers")
	{
		offers.POST("", service.CreateOffer)
		offers.POST("/:offer_id/accept", service.AcceptOffer)
		offers.GET("/user/:address", service.GetUserOffers)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Start server
	go func() {
		fmt.Printf("NFT Service starting on port %s\n", cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}