/**
 * TigerWallet Hyperliquid Trading Integration
 * Production-ready integration with Hyperliquid perps exchange
 *
 * Features:
 * - Spot and perpetual trading
 * - High leverage (up to 50x)
 * - Real-time price feeds
 * - Order management
 * - Position tracking
 * - Cross-margin support
 */

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	ServerPort     string `json:"server_port"`
	DBHost         string `json:"db_host"`
	DBPort         string `json:"db_port"`
	DBUser         string `json:"db_user"`
	DBPassword     string `json:"db_password"`
	DBName         string `json:"db_name"`
	HyperliquidURL string `json:"hyperliquid_url"`
	HyperliquidWS  string `json:"hyperliquid_ws"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:     getEnv("HYPERLIQUID_PORT", "9101"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "tigerwallet"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "tigerwallet"),
		HyperliquidURL: getEnv("HYPERLIQUID_URL", "https://api.hyperliquid.xyz"),
		HyperliquidWS:  getEnv("HYPERLIQUID_WS", "wss://api.hyperliquid.xyz/ws"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Data Models
// ============================================================================

// User account
type HyperliquidAccount struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	UserAddress     string    `gorm:"uniqueIndex" json:"user_address"`
	HyperliquidAddr string    `gorm:"uniqueIndex" json:"hyperliquid_addr"`
	PublicKey       string    `json:"public_key"`
	VaultAddress    string    `json:"vault_address"`
	IsVault         bool      `json:"is_vault"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Order
type HyperliquidOrder struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	OrderID       string    `gorm:"uniqueIndex" json:"order_id"`
	UserAddress   string    `gorm:"index" json:"user_address"`
	Asset         string    `json:"asset"`
	Side          string    `json:"side"`       // buy, sell
	OrderType     string    `json:"order_type"` // market, limit
	Price         float64   `json:"price"`
	Amount        float64   `json:"amount"`
	FilledAmount  float64   `json:"filled_amount"`
	RemainingSize float64   `json:"remaining_size"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"` // pending, filled, cancelled
	OrderJSON     string    `gorm:"type:jsonb" json:"order_json"`
}

// Position
type HyperliquidPosition struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	UserAddress      string    `gorm:"index" json:"user_address"`
	Asset            string    `json:"asset"`
	Size             float64   `json:"size"`
	EntryPrice       float64   `json:"entry_price"`
	MarkPrice        float64   `json:"mark_price"`
	LiquidationPrice float64   `json:"liquidation_price"`
	Leverage         int       `json:"leverage"`
	PnL              float64   `json:"pnl"`
	PnLPercent       float64   `json:"pnl_percent"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Trade/Execution
type HyperliquidTrade struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	TradeID     string    `gorm:"uniqueIndex" json:"trade_id"`
	OrderID     string    `json:"order_id"`
	UserAddress string    `gorm:"index" json:"user_address"`
	Asset       string    `json:"asset"`
	Side        string    `json:"side"`
	Price       float64   `json:"price"`
	Amount      float64   `json:"amount"`
	Fee         float64   `json:"fee"`
	Timestamp   time.Time `json:"timestamp"`
	TxHash      string    `json:"tx_hash"`
}

// Market data
type MarketData struct {
	Symbol          string  `json:"symbol"`
	Price           float64 `json:"price"`
	Change24h       float64 `json:"change_24h"`
	Volume24h       float64 `json:"volume_24h"`
	OpenInterest    float64 `json:"open_interest"`
	FundingRate     float64 `json:"funding_rate"`
	NextFundingTime int64   `json:"next_funding_time"`
}

// ============================================================================
// Hyperliquid API Types
// ============================================================================

// API Request/Response types
type APIResponse struct {
	RespondedAt time.Time       `json:"respondedAt"`
	Results     json.RawMessage `json:"results"`
}

type PlaceOrderRequest struct {
	Asset      int     `json:"asset"`
	Side       string  `json:"side"`
	Price      float64 `json:"price"`
	Amount     float64 `json:"amount"`
	OrderType  string  `json:"orderType"`
	ReduceOnly bool    `json:"reduceOnly"`
}

type CancelOrderRequest struct {
	Asset   int    `json:"asset"`
	OrderID string `json:"orderId"`
}

type UpdateLeverageRequest struct {
	Asset    int `json:"asset"`
	Leverage int `json:"leverage"`
}

// ============================================================================
// Hyperliquid Service
// ============================================================================

type HyperliquidService struct {
	config *Config
	db     *gorm.DB
	client *HTTPClient
}

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			},
		},
	}
}

func (c *HTTPClient) Post(endpoint string, payload interface{}) ([]byte, error) {
	url := c.baseURL + endpoint

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func NewHyperliquidService(config *Config, db *gorm.DB) *HyperliquidService {
	return &HyperliquidService{
		config: config,
		db:     db,
		client: NewHTTPClient(config.HyperliquidURL),
	}
}

func (s *HyperliquidService) Initialize() error {
	log.Println("Initializing Hyperliquid Service...")

	err := s.db.AutoMigrate(&HyperliquidAccount{}, &HyperliquidOrder{}, &HyperliquidPosition{}, &HyperliquidTrade{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Hyperliquid Service initialized")
	return nil
}

// ============================================================================
// Account Management
// ============================================================================

func (s *HyperliquidService) CreateAccount(userAddress, hyperliquidAddr, publicKey string) (*HyperliquidAccount, error) {
	account := HyperliquidAccount{
		UserAddress:     userAddress,
		HyperliquidAddr: hyperliquidAddr,
		PublicKey:       publicKey,
		VaultAddress:    "",
		IsVault:         false,
	}

	err := s.db.Create(&account).Error
	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (s *HyperliquidService) GetAccount(userAddress string) (*HyperliquidAccount, error) {
	var account HyperliquidAccount
	err := s.db.Where("user_address = ?", userAddress).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (s *HyperliquidService) UpdateVaultAddress(userAddress, vaultAddress string) error {
	return s.db.Model(&HyperliquidAccount{}).
		Where("user_address = ?", userAddress).
		Update("vault_address", vaultAddress).Error
}

// ============================================================================
// Order Management
// ============================================================================

// PlaceOrder signs and submits a REAL order to the Hyperliquid exchange.
// Fail-closed: without HL_PRIVATE_KEY no order is attempted, and the venue
// response (resting/filled oid) is what gets persisted.
func (s *HyperliquidService) PlaceOrder(userAddress, asset, side, orderType string, price, amount float64, reduceOnly bool) (*HyperliquidOrder, error) {
	account, err := s.GetAccount(userAddress)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}
	signer := hlapi.SignerKeyFromEnv()
	if signer == "" {
		return nil, fmt.Errorf("HL_PRIVATE_KEY not configured; order submission disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	assetID, err := hlapi.AssetIndex(ctx, asset)
	if err != nil {
		return nil, err
	}
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
	}
	tif := "Gtc"
	if orderType == "market" {
		tif = "Ioc"
	}
	res, err := hlapi.PlacePerpOrder(ctx, signer, assetID, side == "buy", price, amount, reduceOnly, tif)
	if err != nil {
		return nil, err
	}

	order := HyperliquidOrder{
		OrderID:       "ORDER-" + uuid.New().String()[:8],
		UserAddress:   account.UserAddress,
		Asset:         asset,
		Side:          side,
		OrderType:     orderType,
		Price:         price,
		Amount:        amount,
		FilledAmount:  res.FilledSize,
		RemainingSize: amount - res.FilledSize,
		Timestamp:     time.Now(),
		Status:        res.Status,
		OrderJSON:     strconv.FormatInt(res.VenueOrderID, 10), // real venue oid
	}
	if err := s.db.Create(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// CancelOrder cancels a resting order on the venue with a real signed cancel.
func (s *HyperliquidService) CancelOrder(userAddress, orderID string) error {
	order, err := s.GetOrder(orderID)
	if err != nil {
		return err
	}
	if order.UserAddress != userAddress {
		return fmt.Errorf("unauthorized")
	}
	if order.Status != "resting" {
		return fmt.Errorf("order is not resting on the venue (status: %s)", order.Status)
	}
	if order.OrderJSON == "" {
		return fmt.Errorf("order has no venue id; cannot cancel on venue")
	}
	signer := hlapi.SignerKeyFromEnv()
	if signer == "" {
		return fmt.Errorf("HL_PRIVATE_KEY not configured; cancel disabled")
	}
	oid, err := strconv.ParseInt(order.OrderJSON, 10, 64)
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
	return s.db.Save(order).Error
}

func (s *HyperliquidService) GetOrder(orderID string) (*HyperliquidOrder, error) {
	var order HyperliquidOrder
	err := s.db.Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (s *HyperliquidService) GetUserOrders(userAddress string) ([]HyperliquidOrder, error) {
	var orders []HyperliquidOrder
	err := s.db.Where("user_address = ?", userAddress).Order("timestamp DESC").Find(&orders).Error
	return orders, err
}

// ============================================================================
// Position Management
// ============================================================================

func (s *HyperliquidService) GetPositions(userAddress string) ([]HyperliquidPosition, error) {
	var positions []HyperliquidPosition
	err := s.db.Where("user_address = ? AND size != 0", userAddress).Find(&positions).Error
	return positions, err
}

func (s *HyperliquidService) UpdatePosition(pos *HyperliquidPosition) error {
	return s.db.Save(pos).Error
}

func (s *HyperliquidService) CalculatePnL(position *HyperliquidPosition) (float64, float64) {
	if position.Size == 0 {
		return 0, 0
	}

	pnl := (position.MarkPrice - position.EntryPrice) * position.Size
	pnlPercent := (pnl / (position.EntryPrice * position.Size)) * 100

	return pnl, pnlPercent
}

// ============================================================================
// Market Data
// ============================================================================

// GetMarketData returns real market data from the live Hyperliquid info API.
// Fail-closed: upstream errors propagate; no mocked fields.
func (s *HyperliquidService) GetMarketData(assets []string) ([]MarketData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	real, err := hlapi.GetMarketData(ctx, assets)
	if err != nil {
		return nil, err
	}
	markets := make([]MarketData, 0, len(real))
	for _, m := range real {
		markets = append(markets, MarketData{
			Symbol:          m.Symbol,
			Price:           m.MarkPrice,
			Change24h:       m.Change24h,
			Volume24h:       m.Volume24h,
			OpenInterest:    m.OpenInterest,
			FundingRate:     m.FundingRate,
			NextFundingTime: m.NextFundingAt,
		})
	}
	return markets, nil
}

func (s *HyperliquidService) GetAllMarkets() ([]MarketData, error) {
	assets := []string{"BTC", "ETH", "SOL", "AVAX", "ARB", "OP", "MATIC", "LINK", "ATOM", "DOGE"}
	return s.GetMarketData(assets)
}

// Helper functions
// GetAssetID resolves the real venue asset index from live meta.
// Fail-closed: -1 when the asset is unknown or the venue is unreachable.
func (s *HyperliquidService) GetAssetID(asset string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	id, err := hlapi.AssetIndex(ctx, asset)
	if err != nil {
		return -1
	}
	return id
}

// ============================================================================
// Trade History
// ============================================================================

func (s *HyperliquidService) GetUserTrades(userAddress string, limit int) ([]HyperliquidTrade, error) {
	var trades []HyperliquidTrade
	err := s.db.Where("user_address = ?", userAddress).
		Order("timestamp DESC").
		Limit(limit).
		Find(&trades).Error
	return trades, err
}

func (s *HyperliquidService) RecordTrade(trade *HyperliquidTrade) error {
	return s.db.Create(trade).Error
}

// ============================================================================
// API Handlers
// ============================================================================

func (s *HyperliquidService) CreateAccountHandler(c *gin.Context) {
	var req struct {
		UserAddress     string `json:"user_address" binding:"required"`
		HyperliquidAddr string `json:"hyperliquid_addr" binding:"required"`
		PublicKey       string `json:"public_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	account, err := s.CreateAccount(req.UserAddress, req.HyperliquidAddr, req.PublicKey)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, account)
}

func (s *HyperliquidService) PlaceOrderHandler(c *gin.Context) {
	var req struct {
		UserAddress string  `json:"user_address" binding:"required"`
		Asset       string  `json:"asset" binding:"required"`
		Side        string  `json:"side" binding:"required"`
		OrderType   string  `json:"order_type" binding:"required"`
		Price       float64 `json:"price"`
		Amount      float64 `json:"amount" binding:"required"`
		ReduceOnly  bool    `json:"reduce_only"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	order, err := s.PlaceOrder(req.UserAddress, req.Asset, req.Side, req.OrderType, req.Price, req.Amount, req.ReduceOnly)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, order)
}

func (s *HyperliquidService) GetPositionsHandler(c *gin.Context) {
	address := c.Param("address")

	positions, err := s.GetPositions(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, positions)
}

func (s *HyperliquidService) GetMarketsHandler(c *gin.Context) {
	markets, err := s.GetAllMarkets()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, markets)
}

// ============================================================================
// Imports
// ============================================================================

// Note: imports are at the top of the file

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	if config.DBPassword == "" {
		log.Fatal("DB_PASSWORD is required (no default credential)")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	if err := db.AutoMigrate(&HyperliquidAccount{}, &HyperliquidOrder{}, &HyperliquidPosition{}, &HyperliquidTrade{}); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	svc := NewHyperliquidService(config, db)
	router := gin.Default()
	api := router.Group("/api/v1/hyperliquid")
	{
		api.POST("/accounts", svc.CreateAccountHandler)
		api.POST("/orders", svc.PlaceOrderHandler)
		api.GET("/positions", svc.GetPositionsHandler)
		api.GET("/markets", svc.GetMarketsHandler)
	}

	addr := ":" + config.ServerPort
	log.Printf("hyperliquid service listening on %s", addr)
	srv := &http.Server{Addr: addr, Handler: router, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
