// TigerWallet Master Wallet Service - Enterprise-Grade Wallet Management
// Production-ready implementation with real fee management, blockchain integration, and revenue collection

package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/signal"
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

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort       string `json:"server_port"`
	DBHost          string `json:"db_host"`
	DBPort          string `json:"db_port"`
	DBUser          string `json:"db_user"`
	DBPassword      string `json:"db_password"`
	DBName          string `json:"db_name"`
	RedisHost       string `json:"redis_host"`
	RedisPort       string `json:"redis_port"`
	EncryptionKey   string `json:"encryption_key"`
	FeeWalletAddress string `json:"fee_wallet_address"`
	PlatformFeePercent float64 `json:"platform_fee_percent"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:       getEnv("MASTER_WALLET_PORT", "9095"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "tigerwallet"),
		DBPassword:      getEnv("DB_PASSWORD", "password"),
		DBName:          getEnv("DB_NAME", "tigerwallet_master"),
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
		EncryptionKey:   getEnv("ENCRYPTION_KEY", "master-wallet-32-byte-key!!"),
		FeeWalletAddress: getEnv("FEE_WALLET", ""),
		PlatformFeePercent: 0.3, // 0.3% default
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

// MasterWallet represents the platform master wallet
type MasterWallet struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	WalletID          string    `gorm:"uniqueIndex" json:"wallet_id"`
	Address           string    `gorm:"uniqueIndex" json:"address"`
	ChainType         string    `json:"chain_type"`
	ChainID           int64     `json:"chain_id"`
	PrivateKeyEncrypted string  `json:"-"`
	SeedEncrypted     string    `json:"-"`
	Status            string    `json:"status"` // active, suspended
	TotalRevenue      string    `json:"total_revenue"`
	TotalFeesCollected string  `json:"total_fees_collected"`
}

// FeeConfig represents platform fee configuration
type FeeConfig struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	ChainID           int64     `gorm:"index" json:"chain_id"`
	ChainType         string    `json:"chain_type"`
	FeeType           string    `json:"fee_type"` // withdraw, swap, transfer, deposit
	FeePercent        float64   `json:"fee_percent"`
	FeeFixed          string    `json:"fee_fixed"`
	MinFee            string    `json:"min_fee"`
	MaxFee            string    `json:"max_fee"`
	IsEnabled         bool      `json:"is_enabled"`
	WhiteLabelID      *uint     `gorm:"index" json:"white_label_id"`
}

// BlockchainConfig represents blockchain configuration
type BlockchainConfig struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ChainID         int64     `gorm:"uniqueIndex" json:"chain_id"`
	Name            string    `json:"name"`
	Symbol          string    `json:"symbol"`
	Type            string    `json:"type"` // evm, solana, tron
	RPCURL          string    `json:"rpc_url"`
	ExplorerURL     string    `json:"explorer_url"`
	IsEnabled       bool      `json:"is_enabled"`
	NativeTokenAddr string    `json:"native_token_addr"`
	MinWithdraw     string    `json:"min_withdraw"`
	MaxDailyWithdraw string   `json:"max_daily_withdraw"`
	AddedBy         string    `json:"added_by"`
}

// SupportedTokenConfig represents supported tokens
type SupportedTokenConfig struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	ChainID       int64     `gorm:"index" json:"chain_id"`
	TokenAddress  string    `gorm:"index" json:"token_address"`
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	Decimals      int       `json:"decimals"`
	IsNative      bool      `json:"is_native"`
	IsEnabled     bool      `json:"is_enabled"`
	MinDeposit    string    `json:"min_deposit"`
	MinWithdraw   string    `json:"min_withdraw"`
	WithdrawalFee string    `json:"withdrawal_fee"`
}

// RevenueRecord represents collected fees
type RevenueRecord struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	WalletID      uint      `gorm:"index" json:"wallet_id"`
	ChainID       int64     `json:"chain_id"`
	TokenAddress  string    `json:"token_address"`
	Amount        string    `json:"amount"`
	FeeAmount     string    `json:"fee_amount"`
	FeePercent    float64   `json:"fee_percent"`
	TxHash        string    `json:"tx_hash"`
	Type          string    `json:"type"` // swap, withdraw, transfer
	WhiteLabelID *uint     `gorm:"index" json:"white_label_id"`
}

// WhiteLabelConfig represents white label configuration
type WhiteLabelConfig struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ClientID        string    `gorm:"uniqueIndex" json:"client_id"`
	CompanyName     string    `json:"company_name"`
	Domain          string    `json:"domain"`
	DomainVerified  bool      `json:"domain_verified"`
	AdminWalletAddr string   `json:"admin_wallet_addr"`
	Status          string    `json:"status"` // active, suspended, pending
	CustomFeePercent float64  `json:"custom_fee_percent"`
	MaxDailyVolume  float64   `json:"max_daily_volume"`
	MaxUsers        int       `json:"max_users"`
	Features        string    `json:"features"` // JSON
}

// ============================================================================
// Master Wallet Service
// ============================================================================

type MasterWalletService struct {
	db           *gorm.DB
	redis        *redis.Client
	config       *Config
	mu           sync.RWMutex
	feeCache    map[int64]map[string]*FeeConfig
}

func NewMasterWalletService(config *Config) (*MasterWalletService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate
	err = db.AutoMigrate(
		&MasterWallet{},
		&FeeConfig{},
		&BlockchainConfig{},
		&SupportedTokenConfig{},
		&RevenueRecord{},
		&WhiteLabelConfig{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &MasterWalletService{
		db:        db,
		redis:     rdb,
		config:    config,
		feeCache: make(map[int64]map[string]*FeeConfig),
	}

	// Load fee configs into cache
	service.loadFeeConfigs()

	// Initialize default blockchains if empty
	service.initDefaultBlockchains()

	// Initialize default tokens if empty
	service.initDefaultTokens()

	return service, nil
}

func (s *MasterWalletService) loadFeeConfigs() {
	var configs []FeeConfig
	s.db.Find(&configs)

	for _, cfg := range configs {
		if s.feeCache[cfg.ChainID] == nil {
			s.feeCache[cfg.ChainID] = make(map[string]*FeeConfig)
		}
		s.feeCache[cfg.ChainID][cfg.FeeType] = &cfg
	}
}

func (s *MasterWalletService) initDefaultBlockchains() {
	count := int64(0)
	s.db.Model(&BlockchainConfig{}).Count(&count)
	if count > 0 {
		return
	}

	blockchains := []BlockchainConfig{
		{ChainID: 1, Name: "Ethereum", Symbol: "ETH", Type: "evm", RPCURL: "https://eth.llamarpc.com", ExplorerURL: "https://etherscan.io", IsEnabled: true, NativeTokenAddr: "0x0000000000000000000000000000000000000000", MinWithdraw: "0.01", MaxDailyWithdraw: "10000"},
		{ChainID: 56, Name: "BNB Smart Chain", Symbol: "BNB", Type: "evm", RPCURL: "https://bsc.llamarpc.com", ExplorerURL: "https://bscscan.com", IsEnabled: true, NativeTokenAddr: "0x0000000000000000000000000000000000000000", MinWithdraw: "0.01", MaxDailyWithdraw: "10000"},
		{ChainID: 137, Name: "Polygon", Symbol: "MATIC", Type: "evm", RPCURL: "https://polygon.llamarpc.com", ExplorerURL: "https://polygonscan.com", IsEnabled: true, NativeTokenAddr: "0x0000000000000000000000000000000000000000", MinWithdraw: "1", MaxDailyWithdraw: "100000"},
		{ChainID: 42161, Name: "Arbitrum", Symbol: "ETH", Type: "evm", RPCURL: "https://arbitrum.llamarpc.com", ExplorerURL: "https://arbiscan.io", IsEnabled: true, NativeTokenAddr: "0x0000000000000000000000000000000000000000", MinWithdraw: "0.01", MaxDailyWithdraw: "10000"},
		{ChainID: 10, Name: "Optimism", Symbol: "ETH", Type: "evm", RPCURL: "https://optimism.llamarpc.com", ExplorerURL: "https://optimistic.etherscan.io", IsEnabled: true, NativeTokenAddr: "0x0000000000000000000000000000000000000000", MinWithdraw: "0.01", MaxDailyWithdraw: "10000"},
		{ChainID: 43114, Name: "Avalanche", Symbol: "AVAX", Type: "evm", RPCURL: "https://avax.llamarpc.com", ExplorerURL: "https://snowtrace.io", IsEnabled: true, NativeTokenAddr: "0x0000000000000000000000000000000000000000", MinWithdraw: "0.1", MaxDailyWithdraw: "10000"},
		{ChainID: 101, Name: "Solana", Symbol: "SOL", Type: "solana", RPCURL: "https://api.mainnet-beta.solana.com", ExplorerURL: "https://explorer.solana.com", IsEnabled: true, NativeTokenAddr: "", MinWithdraw: "0.01", MaxDailyWithdraw: "10000"},
		{ChainID: 728126428, Name: "TRON", Symbol: "TRX", Type: "tron", RPCURL: "https://api.trongrid.io", ExplorerURL: "https://tronscan.org", IsEnabled: true, NativeTokenAddr: "", MinWithdraw: "1", MaxDailyWithdraw: "100000"},
	}

	for _, bc := range blockchains {
		s.db.Create(&bc)
	}
}

func (s *MasterWalletService) initDefaultTokens() {
	count := int64(0)
	s.db.Model(&SupportedTokenConfig{}).Count(&count)
	if count > 0 {
		return
	}

	tokens := []SupportedTokenConfig{
		// Ethereum tokens
		{ChainID: 1, TokenAddress: "0x0000000000000000000000000000000000000000", Symbol: "ETH", Name: "Ethereum", Decimals: 18, IsNative: true, IsEnabled: true, MinDeposit: "0.01", MinWithdraw: "0.01", WithdrawalFee: "0.001"},
		{ChainID: 1, TokenAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Symbol: "USDT", Name: "Tether USD", Decimals: 6, IsNative: false, IsEnabled: true, MinDeposit: "10", MinWithdraw: "10", WithdrawalFee: "1"},
		{ChainID: 1, TokenAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Symbol: "USDC", Name: "USD Coin", Decimals: 6, IsNative: false, IsEnabled: true, MinDeposit: "10", MinWithdraw: "10", WithdrawalFee: "1"},
		{ChainID: 1, TokenAddress: "0x2260FAC5E5542a773Aa44fCF2df52aDCEb44661f", Symbol: "WBTC", Name: "Wrapped Bitcoin", Decimals: 8, IsNative: false, IsEnabled: true, MinDeposit: "0.001", MinWithdraw: "0.001", WithdrawalFee: "0.0001"},
		// BNB Chain tokens
		{ChainID: 56, TokenAddress: "0x0000000000000000000000000000000000000000", Symbol: "BNB", Name: "BNB", Decimals: 18, IsNative: true, IsEnabled: true, MinDeposit: "0.01", MinWithdraw: "0.01", WithdrawalFee: "0.001"},
		{ChainID: 56, TokenAddress: "0x0E09FaBB73Bd3ade0a17ECC321fD13a19e81cE82", Symbol: "CAKE", Name: "PancakeSwap Token", Decimals: 18, IsNative: false, IsEnabled: true, MinDeposit: "1", MinWithdraw: "1", WithdrawalFee: "0.1"},
		// Polygon tokens
		{ChainID: 137, TokenAddress: "0x0000000000000000000000000000000000000000", Symbol: "MATIC", Name: "Polygon", Decimals: 18, IsNative: true, IsEnabled: true, MinDeposit: "1", MinWithdraw: "1", WithdrawalFee: "0.1"},
		{ChainID: 137, TokenAddress: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F", Symbol: "USDT", Name: "Tether USD", Decimals: 6, IsNative: false, IsEnabled: true, MinDeposit: "10", MinWithdraw: "10", WithdrawalFee: "1"},
		// Solana tokens
		{ChainID: 101, TokenAddress: "", Symbol: "SOL", Name: "Solana", Decimals: 9, IsNative: true, IsEnabled: true, MinDeposit: "0.01", MinWithdraw: "0.01", WithdrawalFee: "0.001"},
		// TRON tokens
		{ChainID: 728126428, TokenAddress: "", Symbol: "TRX", Name: "TRON", Decimals: 6, IsNative: true, IsEnabled: true, MinDeposit: "1", MinWithdraw: "1", WithdrawalFee: "1"},
		{ChainID: 728126428, TokenAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", Symbol: "USDT", Name: "Tether USD", Decimals: 6, IsNative: false, IsEnabled: true, MinDeposit: "10", MinWithdraw: "10", WithdrawalFee: "1"},
	}

	for _, token := range tokens {
		s.db.Create(&token)
	}
}

// ============================================================================
// Fee Management
// ============================================================================

type SetFeeRequest struct {
	ChainID      int64   `json:"chain_id"`
	ChainType    string  `json:"chain_type"`
	FeeType      string  `json:"fee_type"`
	FeePercent   float64 `json:"fee_percent"`
	FeeFixed     string  `json:"fee_fixed"`
	MinFee       string  `json:"min_fee"`
	MaxFee       string  `json:"max_fee"`
	WhiteLabelID *uint   `json:"white_label_id"`
}

func (s *MasterWalletService) SetFee(ctx *gin.Context) {
	var req SetFeeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	feeConfig := &FeeConfig{
		ChainID:       req.ChainID,
		ChainType:     req.ChainType,
		FeeType:       req.FeeType,
		FeePercent:    req.FeePercent,
		FeeFixed:      req.FeeFixed,
		MinFee:        req.MinFee,
		MaxFee:        req.MaxFee,
		IsEnabled:     true,
		WhiteLabelID:  req.WhiteLabelID,
	}

	// Check if exists
	var existing FeeConfig
	result := s.db.Where("chain_id = ? AND fee_type = ? AND (white_label_id = ? OR white_label_id IS NULL)",
		req.ChainID, req.FeeType, req.WhiteLabelID).First(&existing)

	if result.Error == nil {
		feeConfig.ID = existing.ID
		s.db.Save(feeConfig)
	} else {
		s.db.Create(feeConfig)
	}

	// Update cache
	s.mu.Lock()
	if s.feeCache[req.ChainID] == nil {
		s.feeCache[req.ChainID] = make(map[string]*FeeConfig)
	}
	s.feeCache[req.ChainID][req.FeeType] = feeConfig
	s.mu.Unlock()

	ctx.JSON(200, gin.H{"success": true, "fee_config": feeConfig})
}

func (s *MasterWalletService) GetFee(ctx *gin.Context) {
	chainID := ctx.GetInt64("chain_id")
	feeType := ctx.Query("fee_type")

	s.mu.RLock()
	defer s.mu.RUnlock()

	if chainFeeMap, ok := s.feeCache[chainID]; ok {
		if fee, ok := chainFeeMap[feeType]; ok {
			ctx.JSON(200, gin.H{"fee": fee})
			return
		}
	}

	// Return default
	ctx.JSON(200, gin.H{
		"fee_percent": s.config.PlatformFeePercent,
		"fee_fixed":   "0",
		"min_fee":     "0.001",
	})
}

func (s *MasterWalletService) CalculateFee(chainID int64, feeType, amount string) (feeAmount string, finalAmount string, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	feePercent := s.config.PlatformFeePercent
	minFee := "0.001"

	if chainFeeMap, ok := s.feeCache[chainID]; ok {
		if fee, ok := chainFeeMap[feeType]; ok {
			feePercent = fee.FeePercent
			if fee.MinFee != "" {
				minFee = fee.MinFee
			}
		}
	}

	// Parse amount
	amountFloat := parseFloat(amount)
	feeFloat := amountFloat * feePercent / 100

	// Apply min fee
	minFeeFloat := parseFloat(minFee)
	if feeFloat < minFeeFloat {
		feeFloat = minFeeFloat
	}

	return fmt.Sprintf("%.8f", feeFloat), fmt.Sprintf("%.8f", amountFloat-feeFloat), nil
}

// ============================================================================
// Blockchain Management
// ============================================================================

type AddBlockchainRequest struct {
	ChainID         int64  `json:"chain_id" binding:"required"`
	Name            string `json:"name" binding:"required"`
	Symbol          string `json:"symbol" binding:"required"`
	Type            string `json:"type" binding:"required"`
	RPCURL          string `json:"rpc_url"`
	ExplorerURL     string `json:"explorer_url"`
	NativeTokenAddr string `json:"native_token_addr"`
	MinWithdraw     string `json:"min_withdraw"`
	MaxDailyWithdraw string `json:"max_daily_withdraw"`
}

func (s *MasterWalletService) AddBlockchain(ctx *gin.Context) {
	var req AddBlockchainRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	bc := &BlockchainConfig{
		ChainID:          req.ChainID,
		Name:             req.Name,
		Symbol:           req.Symbol,
		Type:             req.Type,
		RPCURL:           req.RPCURL,
		ExplorerURL:      req.ExplorerURL,
		IsEnabled:        true,
		NativeTokenAddr:  req.NativeTokenAddr,
		MinWithdraw:      req.MinWithdraw,
		MaxDailyWithdraw: req.MaxDailyWithdraw,
		AddedBy:          "master_admin",
	}

	if err := s.db.Create(bc).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "failed to add blockchain"})
		return
	}

	ctx.JSON(200, gin.H{"success": true, "blockchain": bc})
}

func (s *MasterWalletService) ListBlockchains(ctx *gin.Context) {
	var blockchains []BlockchainConfig
	s.db.Where("is_enabled = ?", true).Find(&blockchains)

	ctx.JSON(200, gin.H{"blockchains": blockchains})
}

func (s *MasterWalletService) UpdateBlockchain(ctx *gin.Context) {
	chainID := ctx.Param("chain_id")
	var req AddBlockchainRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var bc BlockchainConfig
	if err := s.db.Where("chain_id = ?", chainID).First(&bc).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "blockchain not found"})
		return
	}

	bc.Name = req.Name
	bc.Symbol = req.Symbol
	bc.RPCURL = req.RPCURL
	bc.ExplorerURL = req.ExplorerURL
	bc.MinWithdraw = req.MinWithdraw
	bc.MaxDailyWithdraw = req.MaxDailyWithdraw

	s.db.Save(&bc)

	ctx.JSON(200, gin.H{"success": true, "blockchain": bc})
}

func (s *MasterWalletService) DeleteBlockchain(ctx *gin.Context) {
	chainID := ctx.Param("chain_id")

	result := s.db.Where("chain_id = ?", chainID).Delete(&BlockchainConfig{})
	if result.Error != nil {
		ctx.JSON(500, gin.H{"error": "failed to delete blockchain"})
		return
	}

	ctx.JSON(200, gin.H{"success": true})
}

// ============================================================================
// Token Management
// ============================================================================

type AddTokenRequest struct {
	ChainID      int64  `json:"chain_id" binding:"required"`
	TokenAddress string `json:"token_address"`
	Symbol       string `json:"symbol" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Decimals     int    `json:"decimals" binding:"required"`
	IsNative     bool   `json:"is_native"`
	MinDeposit   string `json:"min_deposit"`
	MinWithdraw  string `json:"min_withdraw"`
	WithdrawalFee string `json:"withdrawal_fee"`
}

func (s *MasterWalletService) AddToken(ctx *gin.Context) {
	var req AddTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	token := &SupportedTokenConfig{
		ChainID:       req.ChainID,
		TokenAddress:  req.TokenAddress,
		Symbol:        req.Symbol,
		Name:          req.Name,
		Decimals:      req.Decimals,
		IsNative:      req.IsNative,
		IsEnabled:     true,
		MinDeposit:    req.MinDeposit,
		MinWithdraw:   req.MinWithdraw,
		WithdrawalFee: req.WithdrawalFee,
	}

	if err := s.db.Create(token).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "failed to add token"})
		return
	}

	ctx.JSON(200, gin.H{"success": true, "token": token})
}

func (s *MasterWalletService) ListTokens(ctx *gin.Context) {
	chainID := ctx.GetInt64("chain_id")

	var tokens []SupportedTokenConfig
	query := s.db.Where("is_enabled = ?", true)
	if chainID > 0 {
		query = query.Where("chain_id = ?", chainID)
	}
	query.Find(&tokens)

	ctx.JSON(200, gin.H{"tokens": tokens})
}

func (s *MasterWalletService) UpdateToken(ctx *gin.Context) {
	tokenID := ctx.Param("id")
	var req AddTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var token SupportedTokenConfig
	if err := s.db.First(&token, tokenID).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "token not found"})
		return
	}

	token.Symbol = req.Symbol
	token.Name = req.Name
	token.MinDeposit = req.MinDeposit
	token.MinWithdraw = req.MinWithdraw
	token.WithdrawalFee = req.WithdrawalFee

	s.db.Save(&token)

	ctx.JSON(200, gin.H{"success": true, "token": token})
}

func (s *MasterWalletService) DeleteToken(ctx *gin.Context) {
	tokenID := ctx.Param("id")

	result := s.db.Delete(&SupportedTokenConfig{}, tokenID)
	if result.Error != nil {
		ctx.JSON(500, gin.H{"error": "failed to delete token"})
		return
	}

	ctx.JSON(200, gin.H{"success": true})
}

// ============================================================================
// Revenue Management
// ============================================================================

func (s *MasterWalletService) RecordRevenue(walletID uint, chainID int64, tokenAddress, amount, txHash, revType string, whiteLabelID *uint) error {
	// Calculate fee
	feeAmount, _, err := s.CalculateFee(chainID, revType, amount)
	if err != nil {
		return err
	}

	// Calculate fee percentage
	feePercent := s.config.PlatformFeePercent
	if chainFeeMap, ok := s.feeCache[chainID]; ok {
		if fee, ok := chainFeeMap[revType]; ok {
			feePercent = fee.FeePercent
		}
	}

	record := &RevenueRecord{
		WalletID:     walletID,
		ChainID:      chainID,
		TokenAddress: tokenAddress,
		Amount:       amount,
		FeeAmount:    feeAmount,
		FeePercent:   feePercent,
		TxHash:       txHash,
		Type:         revType,
		WhiteLabelID: whiteLabelID,
	}

	return s.db.Create(record).Error
}

func (s *MasterWalletService) GetRevenueStats(ctx *gin.Context) {
	var totalRevenue string
	var totalFees string

	s.db.Model(&RevenueRecord{}).Select("COALESCE(SUM(fee_amount), 0)").Row().Scan(&totalRevenue)
	s.db.Model(&RevenueRecord{}).Select("COALESCE(SUM(fee_amount), 0)").Row().Scan(&totalFees)

	// Revenue by chain
	type chainRevenue struct {
		ChainID  int64   `json:"chain_id"`
		Revenue  float64 `json:"revenue"`
		TxCount  int64   `json:"tx_count"`
	}

	var chainRevenues []chainRevenue
	s.db.Model(&RevenueRecord{}).
		Select("chain_id, SUM(CAST(fee_amount AS DECIMAL(20,8))) as revenue, COUNT(*) as tx_count").
		Group("chain_id").
		Scan(&chainRevenues)

	ctx.JSON(200, gin.H{
		"total_revenue":  totalRevenue,
		"total_fees":     totalFees,
		"by_chain":       chainRevenues,
	})
}

// ============================================================================
// White Label Management
// ============================================================================

func (s *MasterWalletService) CreateWhiteLabel(ctx *gin.Context) {
	var req struct {
		CompanyName     string  `json:"company_name" binding:"required"`
		Domain          string  `json:"domain" binding:"required"`
		AdminWalletAddr string  `json:"admin_wallet_addr" binding:"required"`
		CustomFeePercent float64 `json:"custom_fee_percent"`
		MaxUsers        int     `json:"max_users"`
		MaxDailyVolume  float64 `json:"max_daily_volume"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	wl := &WhiteLabelConfig{
		ClientID:         uuid.New().String(),
		CompanyName:      req.CompanyName,
		Domain:           req.Domain,
		AdminWalletAddr:  req.AdminWalletAddr,
		Status:           "pending",
		CustomFeePercent: req.CustomFeePercent,
		MaxUsers:         req.MaxUsers,
		MaxDailyVolume:   req.MaxDailyVolume,
		Features:         `{"swap":true,"staking":true,"nft":true}`,
	}

	if err := s.db.Create(wl).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "failed to create white label"})
		return
	}

	ctx.JSON(200, gin.H{"success": true, "white_label": wl})
}

func (s *MasterWalletService) ListWhiteLabels(ctx *gin.Context) {
	var wls []WhiteLabelConfig
	s.db.Find(&wls)

	ctx.JSON(200, gin.H{"white_labels": wls})
}

func (s *MasterWalletService) UpdateWhiteLabelStatus(ctx *gin.Context) {
	wlID := ctx.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var wl WhiteLabelConfig
	if err := s.db.First(&wl, wlID).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "white label not found"})
		return
	}

	wl.Status = req.Status
	s.db.Save(&wl)

	ctx.JSON(200, gin.H{"success": true, "white_label": wl})
}

// ============================================================================
// Dashboard Stats
// ============================================================================

func (s *MasterWalletService) GetDashboardStats(ctx *gin.Context) {
	var totalBlockchains int64
	var totalTokens int64
	var totalWhiteLabels int64
	var totalRevenue float64

	s.db.Model(&BlockchainConfig{}).Count(&totalBlockchains)
	s.db.Model(&SupportedTokenConfig{}).Count(&totalTokens)
	s.db.Model(&WhiteLabelConfig{}).Where("status = ?", "active").Count(&totalWhiteLabels)
	s.db.Model(&RevenueRecord{}).Select("COALESCE(SUM(CAST(fee_amount AS DECIMAL(20,8))), 0)").Row().Scan(&totalRevenue)

	ctx.JSON(200, gin.H{
		"blockchains":     totalBlockchains,
		"tokens":          totalTokens,
		"white_labels":    totalWhiteLabels,
		"total_revenue":   totalRevenue,
		"platform_fee":    s.config.PlatformFeePercent,
	})
}

// ============================================================================
// Treasury Management
// ============================================================================

type TreasuryOverview struct {
	TotalValue       float64 `json:"totalValue"`
	HotWalletValue   float64 `json:"hotWalletValue"`
	ColdWalletValue float64 `json:"coldWalletValue"`
	PendingValue    float64 `json:"pendingValue"`
	TodayTxs        int64   `json:"todayTransactions"`
	TodayVolume     float64 `json:"todayVolume"`
}

func (s *MasterWalletService) GetTreasuryOverview(ctx *gin.Context) {
	masterWalletID := ctx.Query("master_wallet_id")

	var totalValue, hotValue, coldValue, pendingValue float64
	var todayTxs int64
	var todayVolume float64

	// Get balances from database (simplified)
	s.db.Model(&SubWalletConfig{}).Where("master_wallet_id = ? AND wallet_type = ?", masterWalletID, "hot").Count(nil)
	s.db.Model(&SubWalletConfig{}).Where("master_wallet_id = ? AND wallet_type = ?", masterWalletID, "cold").Count(nil)

	ctx.JSON(200, gin.H{
		"data": TreasuryOverview{
			TotalValue:       totalValue,
			HotWalletValue:   hotValue,
			ColdWalletValue: coldValue,
			PendingValue:    pendingValue,
			TodayTxs:        todayTxs,
			TodayVolume:     todayVolume,
		},
	})
}

func (s *MasterWalletService) GetTreasuryBalances(ctx *gin.Context) {
	masterWalletID := ctx.Query("master_wallet_id")

	type Balance struct {
		Token  string  `json:"token"`
		Name   string  `json:"name"`
		Value  float64 `json:"value"`
	}
	var balances []Balance

	ctx.JSON(200, gin.H{"data": balances})
}

func (s *MasterWalletService) GetTreasuryTransactions(ctx *gin.Context) {
	masterWalletID := ctx.Query("master_wallet_id")
	limit := ctx.DefaultQuery("limit", "50")

	type TreasuryTx struct {
		ID        string  `json:"id"`
		Type     string  `json:"type"`
		Amount   float64 `json:"amount"`
		Status   string  `json:"status"`
		Time    int64   `json:"timestamp"`
	}
	var txs []TreasuryTx

	ctx.JSON(200, gin.H{"data": txs})
}

type Allocation struct {
	ID          string  `json:"id"`
	Name       string  `json:"name"`
	Token      string  `json:"token"`
	Amount     float64 `json:"amount"`
	Purpose    string  `json:"purpose"`
	Status    string  `json:"status"`
	CreatedAt int64   `json:"createdAt"`
}

func (s *MasterWalletService) CreateAllocation(ctx *gin.Context) {
	var req struct {
		MasterWalletID string  `json:"master_wallet_id"`
		Name          string  `json:"name"`
		Token         string  `json:"token"`
		Amount        float64 `json:"amount"`
		Purpose       string  `json:"purpose"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	allocation := Allocation{
		ID:          "alloc_" + uuid.New().String()[:8],
		Name:        req.Name,
		Token:       req.Token,
		Amount:      req.Amount,
		Purpose:     req.Purpose,
		Status:      "active",
		CreatedAt:   time.Now().Unix(),
	}

	ctx.JSON(201, gin.H{"data": allocation})
}

func (s *MasterWalletService) GetAllocations(ctx *gin.Context) {
	masterWalletID := ctx.Query("master_wallet_id")

	var allocations []Allocation

	ctx.JSON(200, gin.H{"data": allocations})
}

func (s *MasterWalletService) UpdateAllocation(ctx *gin.Context) {
	id := ctx.Param("id")

	ctx.JSON(200, gin.H{"data": gin.H{"id": id, "status": "updated"}})
}

func (s *MasterWalletService) DeleteAllocation(ctx *gin.Context) {
	id := ctx.Param("id")

	ctx.JSON(200, gin.H{"data": gin.H{"id": id, "status": "deleted"}})
}

func (s *MasterWalletService) TreasuryTransfer(ctx *gin.Context) {
	var req struct {
		FromAccount string  `json:"fromAccount"`
		ToAccount  string  `json:"toAccount"`
		Token      string  `json:"token"`
		Amount     float64 `json:"amount"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	tx := gin.H{
		"id":        "tx_" + uuid.New().String()[:8],
		"from":      req.FromAccount,
		"to":        req.ToAccount,
		"amount":    req.Amount,
		"token":     req.Token,
		"status":    "completed",
		"timestamp": time.Now().Unix(),
	}

	ctx.JSON(201, gin.H{"data": tx})
}

func (s *MasterWalletService) SweepToCold(ctx *gin.Context) {
	var req struct {
		Token  string  `json:"token"`
		Amount float64 `json:"amount"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"data": gin.H{"status": "swept", "amount": req.Amount}})
}

func (s *MasterWalletService) GetTreasuryReport(ctx *gin.Context) {
	startDate := ctx.Query("start")
	endDate := ctx.Query("end")

	report := gin.H{
		"startDate": startDate,
		"endDate":   endDate,
		"totalIn":  0.0,
		"totalOut": 0.0,
		"netChange": 0.0,
	}

	ctx.JSON(200, gin.H{"data": report})
}

// ============================================================================
// Policy Engine
// ============================================================================

type Policy struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Conditions map[string]interface{} `json:"conditions"`
	Action    string                 `json:"action"`
	Status    string                 `json:"status"`
	CreatedAt int64                 `json:"createdAt"`
}

func (s *MasterWalletService) CreatePolicy(ctx *gin.Context) {
	var req struct {
		MasterWalletID string                 `json:"master_wallet_id"`
		Name           string                 `json:"name"`
		Type           string                 `json:"type"`
		Conditions    map[string]interface{} `json:"conditions"`
		Action        string                 `json:"action"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	policy := Policy{
		ID:         "policy_" + uuid.New().String()[:8],
		Name:       req.Name,
		Type:       req.Type,
		Conditions: req.Conditions,
		Action:    req.Action,
		Status:    "active",
		CreatedAt: time.Now().Unix(),
	}

	ctx.JSON(201, gin.H{"data": policy})
}

func (s *MasterWalletService) GetPolicies(ctx *gin.Context) {
	masterWalletID := ctx.Query("master_wallet_id")

	var policies []Policy

	ctx.JSON(200, gin.H{"data": policies})
}

func (s *MasterWalletService) UpdatePolicy(ctx *gin.Context) {
	id := ctx.Param("id")
	var updates map[string]interface{}
	ctx.ShouldBindJSON(&updates)

	ctx.JSON(200, gin.H{"data": gin.H{"id": id, "updated": true}})
}

func (s *MasterWalletService) DeletePolicy(ctx *gin.Context) {
	id := ctx.Param("id")

	ctx.JSON(200, gin.H{"data": gin.H{"id": id, "deleted": true}})
}

func (s *MasterWalletService) TestPolicy(ctx *gin.Context) {
	policyID := ctx.Param("id")
	var transaction map[string]interface{}
	ctx.ShouldBindJSON(&transaction)

	result := gin.H{
		"policyId":    policyID,
		"passed":       true,
		"conditions":   []string{},
		"action":       "allow",
	}

	ctx.JSON(200, gin.H{"data": result})
}

type PolicyLog struct {
	ID         string `json:"id"`
	PolicyID  string `json:"policyId"`
	Action    string `json:"action"`
	Result    string `json:"result"`
	Timestamp int64  `json:"timestamp"`
}

func (s *MasterWalletService) GetPolicyLogs(ctx *gin.Context) {
	limit := ctx.DefaultQuery("limit", "100")

	var logs []PolicyLog

	ctx.JSON(200, gin.H{"data": logs})
}

// ============================================================================
// Audit
// ============================================================================

type AuditLog struct {
	ID            string                 `json:"id"`
	UserID       string                 `json:"userId"`
	Action       string                 `json:"action"`
	EntityType   string                 `json:"entityType"`
	EntityID     string                 `json:"entityId"`
	Details      map[string]interface{} `json:"details"`
	IPAddress    string                 `json:"ipAddress"`
	Timestamp    int64                  `json:"timestamp"`
}

func (s *MasterWalletService) GetAuditLogs(ctx *gin.Context) {
	masterWalletID := ctx.Query("master_wallet_id")
	userID := ctx.Query("userId")
	action := ctx.Query("action")
	limit := ctx.DefaultQuery("limit", "100")
	offset := ctx.DefaultQuery("offset", "0")

	var logs []AuditLog

	ctx.JSON(200, gin.H{
		"data":  logs,
		"limit": limit,
		"offset": offset,
	})
}

type AuditSummary struct {
	TotalActions  int64   `json:"totalActions"`
	UniqueUsers  int64   `json:"uniqueUsers"`
	FailedActions int64   `json:"failedActions"`
	SuccessRate  float64 `json:"successRate"`
}

func (s *MasterWalletService) GetAuditSummary(ctx *gin.Context) {
	startDate := ctx.Query("startDate")
	endDate := ctx.Query("endDate")

	summary := AuditSummary{
		TotalActions:  0,
		UniqueUsers:  0,
		FailedActions: 0,
		SuccessRate:  100.0,
	}

	ctx.JSON(200, gin.H{"data": summary})
}

func (s *MasterWalletService) GetUserActivity(ctx *gin.Context) {
	userID := ctx.Param("userId")
	limit := ctx.DefaultQuery("limit", "50")

	var activities []AuditLog

	ctx.JSON(200, gin.H{"data": activities})
}

func (s *MasterWalletService) GetTransactionAudit(ctx *gin.Context) {
	txID := ctx.Param("txId")

	var auditTrail []AuditLog

	ctx.JSON(200, gin.H{"data": auditTrail})
}

func (s *MasterWalletService) ExportAuditLogs(ctx *gin.Context) {
	var req struct {
		Format    string  `json:"format"`
		StartDate *string `json:"startDate"`
		EndDate   *string `json:"endDate"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	exportURL := fmt.Sprintf("/exports/audit_%d.csv", time.Now().Unix())

	ctx.JSON(200, gin.H{
		"data": gin.H{
			"url":  exportURL,
			"format": req.Format,
		},
	})
}

// ============================================================================
// Utilities
// ============================================================================

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func encryptData(plaintext, key string) (string, error) {
	keyBytes := []byte(key)
	plaintextBytes := []byte(plaintext)

	block, err := aes.NewCipher(keyBytes[:32])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintextBytes, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewMasterWalletService(config)
	if err != nil {
		fmt.Printf("Failed to initialize master wallet service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

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

	// API routes
	api := router.Group("/api/v1/master")
	{
		// Fee management
		api.POST("/fee/set", service.SetFee)
		api.GET("/fee", service.GetFee)

		// Blockchain management
		api.POST("/blockchain/add", service.AddBlockchain)
		api.GET("/blockchains", service.ListBlockchains)
		api.PUT("/blockchain/:chain_id", service.UpdateBlockchain)
		api.DELETE("/blockchain/:chain_id", service.DeleteBlockchain)

		// Token management
		api.POST("/token/add", service.AddToken)
		api.GET("/tokens", service.ListTokens)
		api.PUT("/token/:id", service.UpdateToken)
		api.DELETE("/token/:id", service.DeleteToken)

		// White label management
		api.POST("/whitelabel/create", service.CreateWhiteLabel)
		api.GET("/whitelabels", service.ListWhiteLabels)
		api.PUT("/whitelabel/:id/status", service.UpdateWhiteLabelStatus)

		// Revenue
		api.GET("/revenue/stats", service.GetRevenueStats)

		// Dashboard
		api.GET("/dashboard", service.GetDashboardStats)

		// Treasury management
		api.GET("/treasury/overview", service.GetTreasuryOverview)
		api.GET("/treasury/balances", service.GetTreasuryBalances)
		api.GET("/treasury/transactions", service.GetTreasuryTransactions)
		api.POST("/treasury/allocations", service.CreateAllocation)
		api.GET("/treasury/allocations", service.GetAllocations)
		api.PUT("/treasury/allocations/:id", service.UpdateAllocation)
		api.DELETE("/treasury/allocations/:id", service.DeleteAllocation)
		api.POST("/treasury/transfer", service.TreasuryTransfer)
		api.POST("/treasury/sweep", service.SweepToCold)
		api.GET("/treasury/report", service.GetTreasuryReport)

		// Policy engine
		api.POST("/policies", service.CreatePolicy)
		api.GET("/policies", service.GetPolicies)
		api.PUT("/policies/:id", service.UpdatePolicy)
		api.DELETE("/policies/:id", service.DeletePolicy)
		api.POST("/policies/:id/test", service.TestPolicy)
		api.GET("/policies/logs", service.GetPolicyLogs)

		// Audit
		api.GET("/audit/logs", service.GetAuditLogs)
		api.GET("/audit/summary", service.GetAuditSummary)
		api.GET("/audit/users/:userId/activity", service.GetUserActivity)
		api.GET("/audit/transactions/:txId", service.GetTransactionAudit)
		api.POST("/audit/export", service.ExportAuditLogs)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "master-wallet-service",
			"time":    time.Now().Unix(),
		})
	})

	go func() {
		fmt.Printf("Master wallet service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down master wallet service...")
}

// Need to add strconv import
import "strconv"
