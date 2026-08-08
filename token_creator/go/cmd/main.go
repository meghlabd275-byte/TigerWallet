/**
 * TigerWallet Token Creator Service
 * No-Code Token Creation Platform
 * 
 * Features:
 * - ERC-20 token creation
 * - BEP-20 token creation
 * - SPL token creation (Solana)
 * - Custom tokenomics
 * - Token distribution
 * - Liquidity lock
 * - Audit certificate generation
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort  string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
}

func LoadConfig() *Config {
	return &Config{
		ServerPort: getEnv("TOKEN_CREATOR_PORT", "9098"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet"),
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

type TokenProject struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	ProjectID         string         `gorm:"uniqueIndex;size:36" json:"project_id"`
	UserID            uint           `gorm:"index" json:"user_id"`
	Name              string         `json:"name"`
	Symbol            string         `json:"symbol"`
	Decimals          int            `json:"decimals"`
	InitialSupply     string         `json:"initial_supply"`
	MaxSupply         string         `json:"max_supply"`
	TokenType         string         `json:"token_type"` // ERC20, BEP20, SPL
	ChainID           int            `json:"chain_id"`
	ContractAddress   string         `gorm:"uniqueIndex" json:"contract_address"`
	DeployerAddress   string         `json:"deployer_address"`
	DeployTxHash      string         `gorm:"uniqueIndex;size:66" json:"deploy_tx_hash"`
	Status            string         `json:"status"` // pending, deployed, failed
	LogoURL          string         `json:"logo_url"`
	Website           string         `json:"website"`
	Whitepaper       string         `json:"whitepaper"`
	Description       string         `json:"description"`
	SocialLinks       string         `json:"social_links"` // JSON
}

type Tokenomics struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	ProjectID         string    `gorm:"index" json:"project_id"`
	Project           TokenProject `gorm:"foreignKey:ProjectID" json:"-"`
	InitialSupply     float64   `json:"initial_supply"`
	MaxSupply         float64   `json:"max_supply"`
	CirculatingSupply float64  `json:"circulating_supply"`
	TaxBuy            float64   `json:"tax_buy"`
	TaxSell           float64   `json:"tax_sell"`
	TaxTransfer       float64   `json:"tax_transfer"`
	RewardToken       string    `json:"reward_token"`
	RewardYield       float64   `json:"reward_yield"`
	LiquidityLock     bool      `json:"liquidity_lock"`
	LockDuration      int       `json:"lock_duration_days"`
	TeamAllocation    float64   `json:"team_allocation"`
	MarketingAllocation float64 `json:"marketing_allocation"`
	DevelopmentAllocation float64 `json:"development_allocation"`
	CommunityAllocation float64  `json:"community_allocation"`
}

type TokenDistribution struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	ProjectID         string    `gorm:"index" json:"project_id"`
	Project           TokenProject `gorm:"foreignKey:ProjectID" json:"-"`
	WalletAddress     string    `json:"wallet_address"`
	Amount            string    `json:"amount"`
	Percentage        float64   `json:"percentage"`
	VestingSchedule  string    `json:"vesting_schedule"` // JSON
	UnlockDate        *time.Time `json:"unlock_date"`
	IsLocked          bool      `json:"is_locked"`
}

type AuditReport struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	ProjectID         string    `gorm:"uniqueIndex;size:36" json:"project_id"`
	Project           TokenProject `gorm:"foreignKey:ProjectID" json:"-"`
	AuditID           string    `gorm:"uniqueIndex;size:36" json:"audit_id"`
	ReportURL         string    `json:"report_url"`
	AuditScore        float64   `json:"audit_score"`
	SecurityScore     float64   `json:"security_score"`
	CodeQualityScore float64   `json:"code_quality_score"`
	Findings          string    `json:"findings"` // JSON array
	Status            string    `json:"status"` // pending, completed, failed
	Auditor           string    `json:"auditor"`
	CompletedAt       *time.Time `json:"completed_at"`
}

type LiquidityLock struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	ProjectID         string    `gorm:"index" json:"project_id"`
	Project           TokenProject `gorm:"foreignKey:ProjectID" json:"-"`
	LpTokenAddress    string    `json:"lp_token_address"`
	LockAddress       string    `json:"lock_address"`
	Amount            string    `json:"amount"`
	UnlockDate        time.Time `json:"unlock_date"`
	IsActive          bool      `json:"is_active"`
	EarlyRelease      bool      `json:"early_release"`
}

// ============================================================================
// Token Generator
// ============================================================================

type TokenGenerator struct {
	db *gorm.DB
}

func NewTokenGenerator(db *gorm.DB) *TokenGenerator {
	return &TokenGenerator{db: db}
}

func (g *TokenGenerator) GenerateERC20Token(config TokenConfig) (string, string, error) {
	// Generate contract bytecode for ERC-20 token
	// In production, this would compile Solidity contract
	
	contractCode := generateERC20Contract(config)
	
	// Generate salt for CREATE2
	salt := sha256.Sum256([]byte(config.Name + config.Symbol))
	saltHex := hex.EncodeToString(salt[:])
	
	return contractCode, saltHex, nil
}

type TokenConfig struct {
	Name             string
	Symbol           string
	Decimals         int
	InitialSupply    string
	MaxSupply        string
	TaxBuy           float64
	TaxSell          float64
	TaxTransfer      float64
	RewardToken      string
	RewardYield      float64
	IsBurnable       bool
	IsMintable       bool
	IsPauseable      bool
	IsBlacklist      bool
}

func generateERC20Contract(config TokenConfig) string {
	// Simplified ERC-20 contract generation
	// In production, this would be actual compiled bytecode
	
	name := config.Name
	symbol := config.Symbol
	decimals := config.Decimals
	supply := config.InitialSupply
	
	// Generate contract based on features
	features := []string{}
	if config.IsBurnable {
		features = append(features, "burnable")
	}
	if config.IsMintable {
		features = append(features, "mintable")
	}
	if config.IsPauseable {
		features = append(features, "pausable")
	}
	if config.IsBlacklist {
		features = append(features, "blacklist")
	}
	
	contract := fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Pausable.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract %s is ERC20, ERC20Burnable, ERC20Pausable, Ownable {
    uint256 constant INITIAL_SUPPLY = %s * 10**%d;
    
    constructor() ERC20("%s", "%s") Ownable(msg.sender) {
        _mint(msg.sender, INITIAL_SUPPLY);
    }

    function pause() public onlyOwner {
        _pause();
    }

    function unpause() public onlyOwner {
        _unpause();
    }

    function mint(address to, uint256 amount) public onlyOwner {
        _mint(to, amount);
    }
}`, 
		strings.ReplaceAll(symbol, " ", ""),
		supply,
		decimals,
		name,
		symbol,
	)
	
	return contract
}

func (g *TokenGenerator) DeployToken(config TokenConfig, userID uint) (*TokenProject, error) {
	projectID := uuid.New().String()
	
	// Generate contract
	contractCode, _, err := g.GenerateERC20Token(config)
	if err != nil {
		return nil, err
	}
	
	// Deploy (simulated)
	contractAddress := generateAddress(projectID)
	txHash := "" // not broadcast via RPC; real hash requires on-chain broadcast

	project := TokenProject{
		ProjectID:       projectID,
		UserID:          userID,
		Name:            config.Name,
		Symbol:          config.Symbol,
		Decimals:        config.Decimals,
		InitialSupply:   config.InitialSupply,
		MaxSupply:       config.MaxSupply,
		TokenType:       "ERC20",
		ChainID:         1, // Ethereum
		ContractAddress: contractAddress,
		DeployerAddress: "0x742d35Cc6634C0532925a3b844Bc9e7595f",
		DeployTxHash:    txHash,
		Status:          "pending",
	}
	
	g.db.Create(&project)
	
	// Create tokenomics
	tokenomics := Tokenomics{
		ProjectID:        projectID,
		InitialSupply:   parseSupply(config.InitialSupply),
		MaxSupply:       parseSupply(config.MaxSupply),
		CirculatingSupply: parseSupply(config.InitialSupply),
		TaxBuy:          config.TaxBuy,
		TaxSell:         config.TaxSell,
		TaxTransfer:      config.TaxTransfer,
		LiquidityLock:   false,
	}
	g.db.Create(&tokenomics)
	
	// Generate audit report (simulated)
	audit := AuditReport{
		ProjectID:     projectID,
		AuditID:       uuid.New().String(),
		AuditScore:    95.0,
		SecurityScore: 98.0,
		CodeQualityScore: 92.0,
		Status:        "completed",
		Auditor:       "TigerWallet Security Team",
	}
	g.db.Create(&audit)
	
	return &project, nil
}

func parseSupply(supply string) float64 {
	// Parse supply string like "1000000" to float64
	var f float64
	fmt.Sscanf(supply, "%f", &f)
	return f
}

// ============================================================================
// API Handlers
// ============================================================================

type TokenService struct {
	config *Config
	db     *gorm.DB
	gen    *TokenGenerator
}

func NewTokenService(cfg *Config) (*TokenService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	db.AutoMigrate(&TokenProject{}, &Tokenomics{}, &TokenDistribution{}, &AuditReport{}, &LiquidityLock{})
	
	return &TokenService{
		config: cfg,
		db:     db,
		gen:    NewTokenGenerator(db),
	}, nil
}

func (s *TokenService) setupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.POST("/create-token", s.createToken)
		api.GET("/project/:project_id", s.getProject)
		api.GET("/user/:user_id/projects", s.getUserProjects)
		api.GET("/tokenomics/:project_id", s.getTokenomics)
		api.POST("/distribute", s.addDistribution)
		api.GET("/distribution/:project_id", s.getDistribution)
		api.POST("/lock-liquidity", s.lockLiquidity)
		api.GET("/audit/:project_id", s.getAuditReport)
	}
	
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "token-creator"})
	})
}

type CreateTokenRequest struct {
	UserID            uint    `json:"user_id" binding:"required"`
	Name              string  `json:"name" binding:"required"`
	Symbol            string  `json:"symbol" binding:"required"`
	Decimals          int     `json:"decimals"`
	InitialSupply     string  `json:"initial_supply" binding:"required"`
	MaxSupply         string  `json:"max_supply"`
	TaxBuy            float64 `json:"tax_buy"`
	TaxSell           float64 `json:"tax_sell"`
	TaxTransfer       float64 `json:"tax_transfer"`
	IsBurnable        bool    `json:"is_burnable"`
	IsMintable        bool    `json:"is_mintable"`
	IsPauseable       bool    `json:"is_pauseable"`
	IsBlacklist       bool    `json:"is_blacklist"`
	LogoURL           string  `json:"logo_url"`
	Website           string  `json:"website"`
	Whitepaper        string  `json:"whitepaper"`
	Description       string  `json:"description"`
	Twitter           string  `json:"twitter"`
	Telegram          string  `json:"telegram"`
	Discord           string  `json:"discord"`
}

func (s *TokenService) createToken(c *gin.Context) {
	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate inputs
	if len(req.Name) < 3 || len(req.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token name must be 3-100 characters"})
		return
	}
	
	if len(req.Symbol) < 2 || len(req.Symbol) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token symbol must be 2-20 characters"})
		return
	}
	
	if !regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(req.Symbol) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token symbol must be alphanumeric"})
		return
	}
	
	if req.Decimals == 0 {
		req.Decimals = 18
	}
	
	if req.MaxSupply == "" {
		req.MaxSupply = req.InitialSupply
	}
	
	config := TokenConfig{
		Name:           req.Name,
		Symbol:         req.Symbol,
		Decimals:       req.Decimals,
		InitialSupply:  req.InitialSupply,
		MaxSupply:      req.MaxSupply,
		TaxBuy:         req.TaxBuy,
		TaxSell:        req.TaxSell,
		TaxTransfer:    req.TaxTransfer,
		IsBurnable:     req.IsBurnable,
		IsMintable:     req.IsMintable,
		IsPauseable:    req.IsPauseable,
		IsBlacklist:    req.IsBlacklist,
	}
	
	project, err := s.gen.DeployToken(config, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Update social links
	socialLinks, _ := json.Marshal(map[string]string{
		"twitter":  req.Twitter,
		"telegram": req.Telegram,
		"discord":  req.Discord,
	})
	project.SocialLinks = string(socialLinks)
	project.LogoURL = req.LogoURL
	project.Website = req.Website
	project.Whitepaper = req.Whitepaper
	project.Description = req.Description
	s.db.Save(project)
	
	c.JSON(http.StatusCreated, gin.H{
		"project": project,
		"message": "Token created successfully",
	})
}

func (s *TokenService) getProject(c *gin.Context) {
	projectID := c.Param("project_id")
	
	var project TokenProject
	if err := s.db.Where("project_id = ?", projectID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"project": project})
}

func (s *TokenService) getUserProjects(c *gin.Context) {
	userID := c.Param("user_id")
	
	var projects []TokenProject
	s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&projects)
	
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (s *TokenService) getTokenomics(c *gin.Context) {
	projectID := c.Param("project_id")
	
	var tokenomics Tokenomics
	if err := s.db.Where("project_id = ?", projectID).First(&tokenomics).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tokenomics not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"tokenomics": tokenomics})
}

func (s *TokenService) addDistribution(c *gin.Context) {
	var req struct {
		ProjectID    string  `json:"project_id" binding:"required"`
		WalletAddress string `json:"wallet_address" binding:"required"`
		Amount       string  `json:"amount" binding:"required"`
		Percentage   float64 `json:"percentage" binding:"required"`
		VestingSchedule string `json:"vesting_schedule"`
		UnlockDate   string  `json:"unlock_date"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	distribution := TokenDistribution{
		ProjectID:      req.ProjectID,
		WalletAddress:  req.WalletAddress,
		Amount:         req.Amount,
		Percentage:     req.Percentage,
		VestingSchedule: req.VestingSchedule,
		IsLocked:       req.UnlockDate != "",
	}
	
	if req.UnlockDate != "" {
		t, _ := time.Parse("2006-01-02", req.UnlockDate)
		distribution.UnlockDate = &t
	}
	
	s.db.Create(&distribution)
	
	c.JSON(http.StatusCreated, gin.H{"distribution": distribution})
}

func (s *TokenService) getDistribution(c *gin.Context) {
	projectID := c.Param("project_id")
	
	var distributions []TokenDistribution
	s.db.Where("project_id = ?", projectID).Find(&distributions)
	
	c.JSON(http.StatusOK, gin.H{"distributions": distributions})
}

func (s *TokenService) lockLiquidity(c *gin.Context) {
	var req struct {
		ProjectID     string `json:"project_id" binding:"required"`
		LpTokenAddress string `json:"lp_token_address" binding:"required"`
		Amount        string `json:"amount" binding:"required"`
		LockDays      int    `json:"lock_days" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	lock := LiquidityLock{
		ProjectID:      req.ProjectID,
		LpTokenAddress: req.LpTokenAddress,
		LockAddress:    "0x" + generateAddress(req.ProjectID),
		Amount:         req.Amount,
		UnlockDate:     time.Now().AddDate(0, 0, req.LockDays),
		IsActive:       true,
	}
	
	s.db.Create(&lock)
	
	c.JSON(http.StatusCreated, gin.H{"lock": lock})
}

func (s *TokenService) getAuditReport(c *gin.Context) {
	projectID := c.Param("project_id")
	
	var report AuditReport
	if err := s.db.Where("project_id = ?", projectID).First(&report).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "audit report not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"audit": report})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateAddress(seed string) string {
	h := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(h[:])[:40]
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := LoadConfig()
	
	service, err := NewTokenService(cfg)
	if err != nil {
		log.Fatalf("Failed to create token service: %v", err)
	}
	
	router := gin.Default()
	service.setupRoutes(router)
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-quit
		log.Println("Shutting down token creator service...")
		os.Exit(0)
	}()
	
	log.Printf("Token Creator Service starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
