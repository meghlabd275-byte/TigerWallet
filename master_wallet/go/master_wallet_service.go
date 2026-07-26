// TigerWallet Master Wallet Service - Production Implementation
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config - Service configuration
type Config struct {
	ServerPort     string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	RedisHost     string
	RedisPort     string
	JWTSecret     string
	MPCEnabled    bool
	HSMEnabled    bool
}

// MasterWallet - Master wallet model
type MasterWallet struct {
	ID             uint      `gorm:"primaryKey"`
	WalletID       string    `gorm:"uniqueIndex"`
	Name           string
	WalletType     string    // hot, cold, operations
	BlockchainID   int64
	Address        string    `gorm:"index"`
	PublicKey      string
	IsActive       bool      `gorm:"default:true"`
	Balance        string
	BalanceUSD     float64
	MinBalance     string
	AutoRefill     bool      `gorm:"default:false"`
	RefillThreshold string
	RefillAmount   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UserWallet - User wallet model  
type UserWallet struct {
	ID            uint      `gorm:"primaryKey"`
	WalletID      string    `gorm:"uniqueIndex"`
	UserID        string    `gorm:"index"`
	MasterWalletID uint     `gorm:"index"`
	BlockchainID  int64     `gorm:"index"`
	Address       string    `gorm:"index"`
	PublicKey     string
	IsActive      bool      `gorm:"default:true"`
	Label         string
	Balance       string
	BalanceUSD    float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Transaction - Transaction model
type Transaction struct {
	ID            uint      `gorm:"primaryKey"`
	TxID          string    `gorm:"uniqueIndex"`
	WalletID      string    `gorm:"index"`
	Type          string    // deposit, withdrawal, transfer
	BlockchainID  int64     `gorm:"index"`
	FromAddress  string    `gorm:"index"`
	ToAddress    string    `gorm:"index"`
	Amount        string
	Fee           string
	Status        string    `gorm:"default:'pending'"` // pending, confirmed, failed
	Hash          string    `gorm:"index"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// FeeConfig - Fee configuration
type FeeConfig struct {
	ID            uint      `gorm:"primaryKey"`
	BlockchainID  int64     `gorm:"index"`
	FeeType       string    // withdrawal, deposit
	FeeAmount     string
	FeePercent    float64
	MinFee        string
	IsActive      bool      `gorm:"default:true"`
	UpdatedAt     time.Time
}

// BlockchainService - Main service
type BlockchainService struct {
	db          *gorm.DB
	redis       *redis.Client
	config      *Config
	walletKeys  map[string]*ecdsa.PrivateKey
	mu          sync.RWMutex
	gasPrices   map[int64]*big.Int
}

func NewBlockchainService(config *Config) (*BlockchainService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate
	db.AutoMigrate(&MasterWallet{}, &UserWallet{}, &Transaction{}, &FeeConfig{})

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	return &BlockchainService{
		db:         db,
		redis:      rdb,
		config:     config,
		walletKeys: make(map[string]*ecdsa.PrivateKey),
		gasPrices:  make(map[int64]*big.Int),
	}, nil
}

// CreateMasterWallet - Create new master wallet
func (s *BlockchainService) CreateMasterWallet(name, walletType string, chainID int64) (*MasterWallet, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	address := s.deriveAddress(privateKey.PublicKey)

	wallet := &MasterWallet{
		WalletID:     uuid.New().String(),
		Name:         name,
		WalletType:   walletType,
		BlockchainID: chainID,
		Address:      address,
		PublicKey:   hex.EncodeString(elliptic.Marshal(elliptic.P256(), privateKey.X, privateKey.Y)),
		IsActive:     true,
	}

	if err := s.db.Create(wallet).Error; err != nil {
		return nil, err
	}

	s.walletKeys[wallet.WalletID] = privateKey
	return wallet, nil
}

// CreateUserWallet - Create user wallet under master wallet
func (s *BlockchainService) CreateUserWallet(userID string, chainID int64, masterWalletID uint) (*UserWallet, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	address := s.deriveAddress(privateKey.PublicKey)

	wallet := &UserWallet{
		WalletID:      uuid.New().String(),
		UserID:         userID,
		MasterWalletID: masterWalletID,
		BlockchainID:  chainID,
		Address:        address,
		PublicKey:     hex.EncodeString(elliptic.Marshal(elliptic.P256(), privateKey.X, privateKey.Y)),
		IsActive:       true,
	}

	if err := s.db.Create(wallet).Error; err != nil {
		return nil, err
	}

	s.walletKeys[wallet.WalletID] = privateKey
	return wallet, nil
}

// deriveAddress - Derive address from public key
func (s *BlockchainService) deriveAddress(pubKey ecdsa.PublicKey) string {
	pubBytes := elliptic.Marshal(elliptic.P256(), pubKey.X, pubKey.Y)
	hash := sha256.Sum256(pubBytes)
	return "0x" + hex.EncodeToString(hash[12:])
}

// CreateTransaction - Create new transaction
func (s *BlockchainService) CreateTransaction(walletID, toAddress, amount string, chainID int64) (*Transaction, error) {
	wallet := &MasterWallet{}
	if err := s.db.Where("wallet_id = ?", walletID).First(wallet).Error; err != nil {
		return nil, err
	}

	tx := &Transaction{
		TxID:         uuid.New().String(),
		WalletID:     walletID,
		BlockchainID: chainID,
		FromAddress:  wallet.Address,
		ToAddress:    toAddress,
		Amount:       amount,
		Status:       "pending",
	}

	if err := s.db.Create(tx).Error; err != nil {
		return nil, err
	}

	return tx, nil
}

// SignTransaction - Sign transaction with wallet key
func (s *BlockchainService) SignTransaction(txID string) (string, error) {
	var tx Transaction
	if err := s.db.Where("tx_id = ?", txID).First(&tx).Error; err != nil {
		return "", err
	}

	privateKey := s.walletKeys[tx.WalletID]
	if privateKey == nil {
		return "", fmt.Errorf("wallet key not found")
	}

	// Sign transaction
	r, s2, err := ecdsa.Sign(rand.Reader, privateKey, []byte(tx.TxID+tx.ToAddress+tx.Amount))
	if err != nil {
		return "", err
	}

	signature := append(r.Bytes(), s2.Bytes()...)
	return hex.EncodeToString(signature), nil
}

// GetFeeConfig - Get fee configuration
func (s *BlockchainService) GetFeeConfig(blockchainID int64, feeType string) (*FeeConfig, error) {
	var fee FeeConfig
	err := s.db.Where("blockchain_id = ? AND fee_type = ?", blockchainID, feeType).First(&fee).Error
	if err == gorm.ErrRecordNotFound {
		return &FeeConfig{
			FeeAmount: "5000000000000000",
			FeePercent: 0,
		}, nil
	}
	return &fee, err
}

// UpdateFeeConfig - Update fee configuration
func (s *BlockchainService) UpdateFeeConfig(blockchainID int64, feeType, feeAmount string, feePercent float64) error {
	var fee FeeConfig
	result := s.db.Where("blockchain_id = ? AND fee_type = ?", blockchainID, feeType).First(&fee)
	
	if result.Error == gorm.ErrRecordNotFound {
		fee = FeeConfig{
			BlockchainID: blockchainID,
			FeeType:      feeType,
			FeeAmount:    feeAmount,
			FeePercent:   feePercent,
			IsActive:     true,
		}
		return s.db.Create(&fee).Error
	}

	return s.db.Model(&fee).Updates(map[string]interface{}{
		"fee_amount":  feeAmount,
		"fee_percent": feePercent,
	}).Error
}

// Routes - Setup API routes
func (s *BlockchainService) Routes(r *gin.Engine) {
	api := r.Group("/api/v1/master-wallet")
	{
		api.POST("/wallets", s.createWalletHandler)
		api.GET("/wallets", s.listWalletsHandler)
		api.GET("/wallets/:id", s.getWalletHandler)
		api.POST("/user-wallets", s.createUserWalletHandler)
		api.GET("/user-wallets", s.listUserWalletsHandler)
		api.POST("/transactions", s.createTransactionHandler)
		api.GET("/transactions", s.listTransactionsHandler)
		api.POST("/transactions/:id/sign", s.signTransactionHandler)
		api.GET("/fees", s.getFeeHandler)
		api.PUT("/fees", s.updateFeeHandler)
	}
}

func (s *BlockchainService) createWalletHandler(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		WalletType string `json:"wallet_type"`
		ChainID    int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	wallet, err := s.CreateMasterWallet(req.Name, req.WalletType, req.ChainID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, wallet)
}

func (s *BlockchainService) listWalletsHandler(c *gin.Context) {
	var wallets []MasterWallet
	s.db.Find(&wallets)
	c.JSON(200, wallets)
}

func (s *BlockchainService) getWalletHandler(c *gin.Context) {
	id := c.Param("id")
	var wallet MasterWallet
	if err := s.db.Where("wallet_id = ?", id).First(&wallet).Error; err != nil {
		c.JSON(404, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(200, wallet)
}

func (s *BlockchainService) createUserWalletHandler(c *gin.Context) {
	var req struct {
		UserID        string `json:"user_id"`
		ChainID       int64  `json:"chain_id"`
		MasterWalletID uint  `json:"master_wallet_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	wallet, err := s.CreateUserWallet(req.UserID, req.ChainID, req.MasterWalletID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, wallet)
}

func (s *BlockchainService) listUserWalletsHandler(c *gin.Context) {
	var wallets []UserWallet
	userID := c.Query("user_id")
	if userID != "" {
		s.db.Where("user_id = ?", userID).Find(&wallets)
	} else {
		s.db.Find(&wallets)
	}
	c.JSON(200, wallets)
}

func (s *BlockchainService) createTransactionHandler(c *gin.Context) {
	var req struct {
		WalletID  string `json:"wallet_id"`
		ToAddress string `json:"to_address"`
		Amount    string `json:"amount"`
		ChainID   int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	tx, err := s.CreateTransaction(req.WalletID, req.ToAddress, req.Amount, req.ChainID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, tx)
}

func (s *BlockchainService) listTransactionsHandler(c *gin.Context) {
	var txs []Transaction
	walletID := c.Query("wallet_id")
	if walletID != "" {
		s.db.Where("wallet_id = ?", walletID).Order("created_at DESC").Find(&txs)
	} else {
		s.db.Order("created_at DESC").Find(&txs)
	}
	c.JSON(200, txs)
}

func (s *BlockchainService) signTransactionHandler(c *gin.Context) {
	id := c.Param("id")
	signedTx, err := s.SignTransaction(id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"signed_tx": signedTx})
}

func (s *BlockchainService) getFeeHandler(c *gin.Context) {
	blockchainID := parseInt64(c.Query("blockchain_id"))
	feeType := c.DefaultQuery("fee_type", "withdrawal")
	fee, err := s.GetFeeConfig(blockchainID, feeType)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, fee)
}

func (s *BlockchainService) updateFeeHandler(c *gin.Context) {
	var req struct {
		BlockchainID int64   `json:"blockchain_id"`
		FeeType     string  `json:"fee_type"`
		FeeAmount   string  `json:"fee_amount"`
		FeePercent  float64 `json:"fee_percent"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := s.UpdateFeeConfig(req.BlockchainID, req.FeeType, req.FeeAmount, req.FeePercent); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func main() {
	config := &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "master_wallet"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewBlockchainService(config)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	service.Routes(r)

	log.Printf("Starting Master Wallet Service on port %s", config.ServerPort)
	r.Run(":" + config.ServerPort)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt64(s string) int64 {
	var i int64
	fmt.Sscanf(s, "%d", &i)
	return i
}
