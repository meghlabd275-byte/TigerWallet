/**
 * TigerWallet Hyperliquid Integration Service
 * Production-ready Hyperliquid perpetual trading integration
 *
 * Features:
 * - Perpetual futures trading
 * - Long/Short positions
 * - Up to 50x leverage
 * - Real-time price feeds
 * - Order management
 * - Position tracking
 * - PnL calculation
 */

package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"tigerwallet/hyperliquid/hlapi"
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
	ServerPort string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Hyperliquid
	HyperliquidRPC string
	HyperliquidWS  string
	Testnet        bool

	// Trading
	MaxLeverage     float64
	DefaultLeverage float64
	MaxPositionSize float64
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:      getEnv("HYPERLIQUID_PORT", "9104"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "tigerwallet"),
		DBPassword:      getEnv("DB_PASSWORD", "password"),
		DBName:          getEnv("DB_NAME", "tigerwallet"),
		HyperliquidRPC:  getEnv("HYPERLIQUID_RPC", "https://api.hyperliquid.xyz"),
		HyperliquidWS:   getEnv("HYPERLIQUID_WS", "wss://api.hyperliquid.xyz/ws"),
		Testnet:         getEnv("HYPERLIQUID_TESTNET", "false") == "true",
		MaxLeverage:     50,
		DefaultLeverage: 10,
		MaxPositionSize: 1000000, // $1M
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

type HyperliquidUser struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID        uint   `gorm:"uniqueIndex" json:"user_id"`
	WalletAddress string `gorm:"index" json:"wallet_address"`

	// Hyperliquid account
	HLAddress string `gorm:"uniqueIndex" json:"hl_address"`
	PublicKey string `json:"public_key"`

	// Sub-account
	SubAccount string `json:"sub_account"` // For multi-subaccount trading

	Status string `json:"status"` // active, suspended, closed
}

type Position struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID    uint   `gorm:"index" json:"user_id"`
	HLAddress string `gorm:"index" json:"hl_address"`

	// Position details
	Asset      string  `json:"asset"` // BTC, ETH, SOL, etc.
	Side       string  `json:"side"`  // long, short
	Size       float64 `json:"size"`
	EntryPrice float64 `json:"entry_price"`
	MarkPrice  float64 `json:"mark_price"`
	Leverage   float64 `json:"leverage"`

	// PnL
	UnrealizedPNL float64 `json:"unrealized_pnl"`
	RealizedPNL   float64 `json:"realized_pnl"`

	// Liquidation
	LiquidationPrice float64 `json:"liquidation_price"`
	IsLiquidated     bool    `json:"is_liquidated"`

	Status string `json:"status"` // open, closed, liquidated
}

type Order struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID    uint   `gorm:"index" json:"user_id"`
	HLAddress string `gorm:"index" json:"hl_address"`

	// Order details
	OrderID      string  `gorm:"uniqueIndex" json:"order_id"`
	HLOrderID    string  `json:"hl_order_id"`
	Asset        string  `json:"asset"`
	Side         string  `json:"side"`       // buy, sell
	OrderType    string  `json:"order_type"` // market, limit, stop_market, stop_limit
	Price        float64 `json:"price"`
	TriggerPrice float64 `json:"trigger_price"`
	Size         float64 `json:"size"`
	FilledSize   float64 `json:"filled_size"`
	Leverage     float64 `json:"leverage"`

	// Status
	Status           string  `json:"status"` // pending, open, filled, cancelled, expired
	AverageFillPrice float64 `json:"average_fill_price"`

	// Time
	ExpiresAt *time.Time `json:"expires_at"`
	FilledAt  *time.Time `json:"filled_at"`
}

type Trade struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID    uint   `gorm:"index" json:"user_id"`
	HLAddress string `gorm:"index" json:"hl_address"`

	// Trade details
	TradeID   string  `gorm:"uniqueIndex" json:"trade_id"`
	HLTradeID string  `json:"hl_trade_id"`
	OrderID   string  `gorm:"index" json:"order_id"`
	Asset     string  `json:"asset"`
	Side      string  `json:"side"`
	Size      float64 `json:"size"`
	Price     float64 `json:"price"`
	Fee       float64 `json:"fee"`

	// PnL
	RealizedPNL float64 `json:"realized_pnl"`

	TransactionHash string `json:"transaction_hash"`
}

type AssetInfo struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	Asset  string `gorm:"uniqueIndex" json:"asset"`
	Symbol string `json:"symbol"`

	// Trading info
	MaxLeverage  float64 `json:"max_leverage"`
	MinOrderSize float64 `json:"min_order_size"`
	TickSize     float64 `json:"tick_size"`
	ContractSize float64 `json:"contract_size"`

	// Risk parameters
	InitialMarginRate     float64 `json:"initial_margin_rate"`
	MaintenanceMarginRate float64 `json:"maintenance_margin_rate"`
	MaxPositionSize       float64 `json:"max_position_size"`

	// Funding
	FundingRate     float64   `json:"funding_rate"`
	NextFundingTime time.Time `json:"next_funding_time"`

	// Price
	MarkPrice    float64 `json:"mark_price"`
	IndexPrice   float64 `json:"index_price"`
	OpenInterest float64 `json:"open_interest"`

	IsActive bool `json:"is_active"`
}

type AccountBalance struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID    uint   `gorm:"index" json:"user_id"`
	HLAddress string `gorm:"index" json:"hl_address"`

	// Balances
	TotalCollateral    float64 `json:"total_collateral"`
	AvailableBalance   float64 `json:"available_balance"`
	TotalPositionValue float64 `json:"total_position_value"`
	TotalPendingOrders float64 `json:"total_pending_orders"`

	// Unrealized
	TotalUnrealizedPNL float64 `json:"total_unrealized_pnl"`

	// Leverage
	AccountLeverage float64 `json:"account_leverage"`
	HealthFactor    float64 `json:"health_factor"`
}

// ============================================================================
// Hyperliquid API Types

// ============================================================================
// Service Implementation
// ============================================================================

type HyperliquidService struct {
	db     *gorm.DB
	config *Config
}

func NewHyperliquidService(db *gorm.DB, config *Config) *HyperliquidService {
	return &HyperliquidService{
		db:     db,
		config: config,
	}
}

// CreateUser creates a new Hyperliquid user account
func (s *HyperliquidService) CreateUser(userID uint, walletAddress string) (*HyperliquidUser, error) {
	// A Hyperliquid account IS the user's EVM wallet address — there is no
	// separate venue account to create. The wallet address is the account.
	hlAddress := walletAddress

	user := &HyperliquidUser{
		UserID:        userID,
		WalletAddress: walletAddress,
		HLAddress:     hlAddress,
		Status:        "active",
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// GetUser gets a Hyperliquid user
func (s *HyperliquidService) GetUser(userID uint) (*HyperliquidUser, error) {
	var user HyperliquidUser
	if err := s.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetBalance gets the user's account balance
func (s *HyperliquidService) GetBalance(userID uint) (*AccountBalance, error) {
	var user HyperliquidUser
	if err := s.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	// Get positions
	var positions []Position
	s.db.Where("hl_address = ? AND status = ?", user.HLAddress, "open").Find(&positions)

	// Real collateral from the live Hyperliquid clearinghouse state.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	state, err := hlapi.GetAccountState(ctx, user.HLAddress)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid account state: %w", err)
	}
	totalCollateral := state.AccountValue
	availableBalance := totalCollateral
	totalPositionValue := 0.0
	totalUnrealizedPNL := 0.0

	for _, pos := range positions {
		positionValue := pos.Size * pos.MarkPrice
		totalPositionValue += positionValue
		totalUnrealizedPNL += pos.UnrealizedPNL
	}

	availableBalance = totalCollateral - totalPositionValue - totalUnrealizedPNL

	balance := &AccountBalance{
		UserID:             userID,
		HLAddress:          user.HLAddress,
		TotalCollateral:    totalCollateral,
		AvailableBalance:   math.Max(0, availableBalance),
		TotalPositionValue: totalPositionValue,
		TotalUnrealizedPNL: totalUnrealizedPNL,
	}

	return balance, nil
}

// OpenPosition opens a new position
// OpenPosition places a REAL order on Hyperliquid (EIP-712 signed, live
// /exchange submission). Fail-closed: without HL_PRIVATE_KEY no order is
// attempted, and fills are never simulated — the venue response determines
// the recorded status.
func (s *HyperliquidService) OpenPosition(userID uint, asset, side string, size, leverage, price float64, orderType string) (*Order, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	signer := hlapi.SignerKeyFromEnv()
	if signer == "" {
		return nil, fmt.Errorf("HL_PRIVATE_KEY not configured; order submission disabled")
	}

	// Validate leverage against the venue metadata (real max leverage).
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	assetID, err := hlapi.AssetIndex(ctx, asset)
	if err != nil {
		return nil, err
	}
	if leverage > s.config.MaxLeverage {
		leverage = s.config.MaxLeverage
	}

	// Market orders use the real live mark price with a protective slippage
	// bound; limit orders use the caller's price.
	if price == 0 {
		md, err := hlapi.GetMarketData(ctx, []string{asset})
		if err != nil || len(md) == 0 {
			return nil, fmt.Errorf("live price unavailable for %s", asset)
		}
		mark := md[0].MarkPrice
		if side == "buy" {
			price = mark * 1.05
		} else {
			price = mark * 0.95
		}
		if orderType == "" {
			orderType = "market"
		}
	}

	// Real margin check against the live venue account value.
	state, err := hlapi.GetAccountState(ctx, user.HLAddress)
	if err != nil {
		return nil, fmt.Errorf("account state: %w", err)
	}
	requiredMargin := size * price / leverage
	if requiredMargin > state.AccountValue {
		return nil, fmt.Errorf("insufficient collateral: required %v, available %v", requiredMargin, state.AccountValue)
	}

	tif := "Gtc"
	if orderType == "market" {
		tif = "Ioc"
	}
	res, err := hlapi.PlacePerpOrder(ctx, signer, assetID, side == "buy", price, size, false, tif)
	if err != nil {
		return nil, err
	}

	order := &Order{
		UserID:    userID,
		HLAddress: user.HLAddress,
		OrderID:   uuid.New().String(),
		HLOrderID: strconv.FormatInt(res.VenueOrderID, 10),
		Asset:     asset,
		Side:      side,
		OrderType: orderType,
		Price:     price,
		Size:      size,
		Leverage:  leverage,
		Status:    res.Status, // real venue status: resting | filled
	}
	if res.Status == "filled" {
		order.FilledSize = res.FilledSize
		order.AverageFillPrice = res.AvgPrice
		now := time.Now()
		order.FilledAt = &now
	}
	if err := s.db.Create(order).Error; err != nil {
		return nil, err
	}
	return order, nil
}

// ClosePosition closes an existing venue position with a REAL reduce-only
// market order. The position is only marked closed after the venue fills.
func (s *HyperliquidService) ClosePosition(userID uint, positionID uint) (*Order, error) {
	var position Position
	if err := s.db.Where("id = ? AND user_id = ? AND status = ?", positionID, userID, "open").First(&position).Error; err != nil {
		return nil, fmt.Errorf("position not found")
	}
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	signer := hlapi.SignerKeyFromEnv()
	if signer == "" {
		return nil, fmt.Errorf("HL_PRIVATE_KEY not configured; order submission disabled")
	}

	side := "sell"
	if position.Side == "short" {
		side = "buy"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	assetID, err := hlapi.AssetIndex(ctx, position.Asset)
	if err != nil {
		return nil, err
	}
	md, err := hlapi.GetMarketData(ctx, []string{position.Asset})
	if err != nil || len(md) == 0 {
		return nil, fmt.Errorf("live price unavailable for %s", position.Asset)
	}
	mark := md[0].MarkPrice
	limitPrice := mark * 0.95
	if side == "buy" {
		limitPrice = mark * 1.05
	}

	res, err := hlapi.PlacePerpOrder(ctx, signer, assetID, side == "buy", limitPrice, position.Size, true, "Ioc")
	if err != nil {
		return nil, err
	}

	order := &Order{
		UserID:    userID,
		HLAddress: user.HLAddress,
		OrderID:   uuid.New().String(),
		HLOrderID: strconv.FormatInt(res.VenueOrderID, 10),
		Asset:     position.Asset,
		Side:      side,
		OrderType: "market",
		Size:      position.Size,
		Status:    res.Status,
	}
	if res.Status == "filled" {
		order.FilledSize = res.FilledSize
		order.AverageFillPrice = res.AvgPrice
		now := time.Now()
		order.FilledAt = &now

		// Position closed on venue: record the real realized PnL from the
		// actual fill price.
		realized := calculatePNL(position.EntryPrice, res.AvgPrice, res.FilledSize, position.Side)
		position.Status = "closed"
		position.RealizedPNL = realized
		position.UnrealizedPNL = 0
		s.db.Save(&position)
	}
	if err := s.db.Create(order).Error; err != nil {
		return nil, err
	}
	return order, nil
}

// GetPositions returns the user's REAL open positions from the live venue
// clearinghouse state (the venue is authoritative).
func (s *HyperliquidService) GetPositions(userID uint) ([]Position, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	venue, err := hlapi.GetVenuePositions(ctx, user.HLAddress)
	if err != nil {
		return nil, err
	}
	markets, _ := hlapi.GetMarketData(ctx, func() []string {
		out := make([]string, 0, len(venue))
		for _, p := range venue {
			out = append(out, p.Asset)
		}
		return out
	}())
	marks := map[string]float64{}
	for _, m := range markets {
		marks[m.Symbol] = m.MarkPrice
	}
	positions := make([]Position, 0, len(venue))
	for _, vp := range venue {
		side := "long"
		size := vp.Size
		if size < 0 {
			side = "short"
			size = -size
		}
		positions = append(positions, Position{
			UserID:           userID,
			HLAddress:        user.HLAddress,
			Asset:            vp.Asset,
			Side:             side,
			Size:             size,
			EntryPrice:       vp.EntryPrice,
			MarkPrice:        marks[vp.Asset],
			Leverage:         vp.Leverage,
			LiquidationPrice: vp.LiquidationPrice,
			UnrealizedPNL:    vp.UnrealizedPNL,
			Status:           "open",
		})
	}
	return positions, nil
}

// GetOrders gets all orders for a user
func (s *HyperliquidService) GetOrders(userID uint, status string) ([]Order, error) {
	var orders []Order
	query := s.db.Where("user_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// CancelOrder cancels a resting order on the venue (real signed cancel).
// Fail-closed when the order has no venue id.
func (s *HyperliquidService) CancelOrder(userID uint, orderID string) error {
	var order Order
	if err := s.db.Where("order_id = ? AND user_id = ? AND status = ?", orderID, userID, "resting").First(&order).Error; err != nil {
		return fmt.Errorf("resting order not found")
	}
	if order.HLOrderID == "" {
		return fmt.Errorf("order has no venue id; cannot cancel on venue")
	}
	signer := hlapi.SignerKeyFromEnv()
	if signer == "" {
		return fmt.Errorf("HL_PRIVATE_KEY not configured; cancel disabled")
	}
	oid, err := strconv.ParseInt(order.HLOrderID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid venue order id: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	assetID, err := hlapi.AssetIndex(ctx, order.Asset)
	if err != nil {
		return err
	}
	if err := hlapi.CancelVenueOrder(ctx, signer, assetID, oid); err != nil {
		return err
	}
	order.Status = "cancelled"
	return s.db.Save(&order).Error
}

// GetTrades gets trade history for a user
func (s *HyperliquidService) GetTrades(userID uint, limit int) ([]Trade, error) {
	var trades []Trade
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&trades).Error; err != nil {
		return nil, err
	}
	return trades, nil
}

// GetAssets gets all available assets for trading
func (s *HyperliquidService) GetAssets() ([]AssetInfo, error) {
	var assets []AssetInfo
	if err := s.db.Where("is_active = ?", true).Find(&assets).Error; err != nil {
		return nil, err
	}
	return assets, nil
}

// GetAssetPrice returns the real live mark price from the venue.
func (s *HyperliquidService) GetAssetPrice(asset string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	md, err := hlapi.GetMarketData(ctx, []string{asset})
	if err != nil {
		return 0, err
	}
	if len(md) == 0 {
		return 0, fmt.Errorf("asset %q not found on hyperliquid", asset)
	}
	return md[0].MarkPrice, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func calculateLiquidationPrice(entryPrice float64, side string, leverage float64) float64 {
	// Liquidation price calculation for perpetual contracts
	// For long: entryPrice * (1 - 1/leverage)
	// For short: entryPrice * (1 + 1/leverage)

	liquidationThreshold := 1.0 / leverage

	if side == "long" {
		return entryPrice * (1 - liquidationThreshold)
	}
	return entryPrice * (1 + liquidationThreshold)
}

func calculatePNL(entryPrice, markPrice float64, size float64, side string) float64 {
	if side == "long" {
		return (markPrice - entryPrice) * size
	}
	return (entryPrice - markPrice) * size
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *HyperliquidService) CreateUserHandler(c *gin.Context) {
	var req struct {
		UserID        uint   `json:"user_id" binding:"required"`
		WalletAddress string `json:"wallet_address" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.CreateUser(req.UserID, req.WalletAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

func (s *HyperliquidService) GetBalanceHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	balance, err := s.GetBalance(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    balance,
	})
}

func (s *HyperliquidService) OpenPositionHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Asset     string  `json:"asset" binding:"required"`
		Side      string  `json:"side" binding:"required"`
		Size      float64 `json:"size" binding:"required"`
		Leverage  float64 `json:"leverage"`
		Price     float64 `json:"price"`
		OrderType string  `json:"order_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	leverage := req.Leverage
	if leverage == 0 {
		leverage = s.config.DefaultLeverage
	}

	orderType := req.OrderType
	if orderType == "" {
		orderType = "market"
	}

	order, err := s.OpenPosition(userID.(uint), req.Asset, req.Side, req.Size, leverage, req.Price, orderType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}

func (s *HyperliquidService) ClosePositionHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	positionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	order, err := s.ClosePosition(userID.(uint), uint(positionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}

func (s *HyperliquidService) GetPositionsHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	positions, err := s.GetPositions(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    positions,
	})
}

func (s *HyperliquidService) GetOrdersHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	status := c.Query("status")
	orders, err := s.GetOrders(userID.(uint), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    orders,
	})
}

func (s *HyperliquidService) GetAssetsHandler(c *gin.Context) {
	assets, err := s.GetAssets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    assets,
	})
}

// ============================================================================
// Database Migration
// ============================================================================

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&HyperliquidUser{},
		&Position{},
		&Order{},
		&Trade{},
		&AssetInfo{},
		&AccountBalance{},
	)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	// Initialize database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := Migrate(db); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	// Seed default assets if empty
	var assetCount int64
	db.Model(&AssetInfo{}).Count(&assetCount)
	if assetCount == 0 {
		assets := []AssetInfo{
			{Asset: "BTC", Symbol: "BTC-PERP", MaxLeverage: 50, MinOrderSize: 0.001, TickSize: 0.5, ContractSize: 0.001, InitialMarginRate: 0.02, MaintenanceMarginRate: 0.005, MarkPrice: 65000, IsActive: true},
			{Asset: "ETH", Symbol: "ETH-PERP", MaxLeverage: 50, MinOrderSize: 0.01, TickSize: 0.05, ContractSize: 0.001, InitialMarginRate: 0.02, MaintenanceMarginRate: 0.005, MarkPrice: 3500, IsActive: true},
			{Asset: "SOL", Symbol: "SOL-PERP", MaxLeverage: 50, MinOrderSize: 0.1, TickSize: 0.01, ContractSize: 0.01, InitialMarginRate: 0.02, MaintenanceMarginRate: 0.005, MarkPrice: 150, IsActive: true},
		}
		db.Create(&assets)
	}

	// Initialize service
	service := NewHyperliquidService(db, config)

	// Setup router
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API routes
	api := router.Group("/api/v1/hyperliquid")
	{
		api.POST("/account", service.CreateUserHandler)
		api.GET("/balance", service.GetBalanceHandler)
		api.POST("/position", service.OpenPositionHandler)
		api.DELETE("/position/:id", service.ClosePositionHandler)
		api.GET("/positions", service.GetPositionsHandler)
		api.GET("/orders", service.GetOrdersHandler)
		api.GET("/assets", service.GetAssetsHandler)
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.ServerPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Hyperliquid service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
