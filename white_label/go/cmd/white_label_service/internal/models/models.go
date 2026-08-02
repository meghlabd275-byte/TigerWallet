package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Enums
type ClientStatus string
type ClientPlan string
type AdminRole string
type ProductType string
type ProductStatus string
type PairStatus string
type BotStrategy string
type BotStatus string
type ChainCategory string
type TokenType string

const (
	StatusPending   ClientStatus = "pending"
	StatusActive   ClientStatus = "active"
	StatusSuspended ClientStatus = "suspended"
	StatusHalted   ClientStatus = "halted"
	StatusExpired   ClientStatus = "expired"
	StatusRevoked  ClientStatus = "revoked"
)

const (
	PlanStarter      ClientPlan = "starter"
	PlanProfessional ClientPlan = "professional"
	PlanEnterprise   ClientPlan = "enterprise"
	PlanCustom       ClientPlan = "custom"
)

const (
	RoleSuperAdmin AdminRole = "super_admin"
	RoleAdmin      AdminRole = "admin"
	RoleManager    AdminRole = "manager"
	RoleSupport    AdminRole = "support"
)

const (
	ProductTrading   ProductType = "trading"
	ProductPerpetual ProductType = "perpetual"
	ProductStaking   ProductType = "staking"
	ProductNFT       ProductType = "nft"
	ProductWallet    ProductType = "wallet"
	ProductBridge    ProductType = "bridge"
	ProductLaunchpad ProductType = "launchpad"
)

const (
	ProductEnabled     ProductStatus = "enabled"
	ProductDisabled   ProductStatus = "disabled"
	ProductMaintenance ProductStatus = "maintenance"
)

const (
	PairActive    PairStatus = "active"
	PairSuspended PairStatus = "suspended"
	PairHalted    PairStatus = "halted"
)

const (
	StrategyArbitrage    BotStrategy = "arbitrage"
	StrategyMarketMaking BotStrategy = "market_making"
	StrategyLiquidity    BotStrategy = "liquidity"
	StrategyGrid         BotStrategy = "grid"
	StrategyDCA         BotStrategy = "dca"
)

const (
	BotRunning  BotStatus = "running"
	BotStopped  BotStatus = "stopped"
	BotError    BotStatus = "error"
	BotPaused   BotStatus = "paused"
)

const (
	ChainEVM     ChainCategory = "evm"
	ChainSolana  ChainCategory = "solana"
	ChainAptos   ChainCategory = "aptos"
	ChainSui     ChainCategory = "sui"
	ChainTON     ChainCategory = "ton"
	ChainBitcoin ChainCategory = "bitcoin"
	ChainCosmos  ChainCategory = "cosmos"
)

const (
	TokenERC20  TokenType = "erc20"
	TokenBEP20  TokenType = "bep20"
	TokenSPL    TokenType = "spl"
	TokenNative TokenType = "native"
	TokenTRC20  TokenType = "trc20"
)

// ============================================================================
// CLIENT MODEL
// ============================================================================

type WhiteLabelClient struct {
	ID               uuid.UUID        `json:"id" db:"id"`
	Name             string           `json:"name" db:"name"`
	Domain           string           `json:"domain" db:"domain"`
	Subdomain        sql.NullString  `json:"subdomain" db:"subdomain"`
	CustomBranding   bool             `json:"customBranding" db:"custom_branding"`
	LogoURL          sql.NullString  `json:"logoUrl" db:"logo_url"`
	PrimaryColor    string           `json:"primaryColor" db:"primary_color"`
	SecondaryColor  string           `json:"secondaryColor" db:"secondary_color"`
	Status          ClientStatus     `json:"status" db:"status"`
	Plan            ClientPlan       `json:"plan" db:"plan"`
	MaxUsers        int              `json:"maxUsers" db:"max_users"`
	CurrentUsers     int              `json:"currentUsers" db:"current_users"`
	FeePercent      float64          `json:"feePercent" db:"fee_percent"`
	Features        json.RawMessage  `json:"features" db:"features"`
	BlockchainAccess json.RawMessage  `json:"blockchainAccess" db:"blockchain_access"`
	APIKeys         json.RawMessage  `json:"apiKeys" db:"api_keys"`
	Metadata        json.RawMessage  `json:"metadata" db:"metadata"`
	CreatedAt       time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time        `json:"updatedAt" db:"updated_at"`
	ApprovedAt      sql.NullTime     `json:"approvedAt" db:"approved_at"`
	ExpiresAt       sql.NullTime     `json:"expiresAt" db:"expires_at"`
}

type CreateClientRequest struct {
	Name             string           `json:"name" binding:"required"`
	Domain           string           `json:"domain" binding:"required,url"`
	Subdomain        string           `json:"subdomain"`
	CustomBranding   bool             `json:"customBranding"`
	LogoURL          string           `json:"logoUrl"`
	PrimaryColor     string           `json:"primaryColor"`
	SecondaryColor   string           `json:"secondaryColor"`
	Plan             ClientPlan       `json:"plan"`
	MaxUsers         int              `json:"maxUsers"`
	FeePercent       float64          `json:"feePercent"`
	Features         map[string]bool  `json:"features"`
	BlockchainAccess []int64          `json:"blockchainAccess"`
}

type UpdateClientRequest struct {
	Name             *string          `json:"name"`
	Subdomain        *string          `json:"subdomain"`
	LogoURL          *string          `json:"logoUrl"`
	PrimaryColor     *string          `json:"primaryColor"`
	SecondaryColor   *string          `json:"secondaryColor"`
	Plan             *ClientPlan      `json:"plan"`
	MaxUsers         *int             `json:"maxUsers"`
	FeePercent       *float64         `json:"feePercent"`
	Features         *map[string]bool `json:"features"`
	BlockchainAccess *[]int64         `json:"blockchainAccess"`
	Status           *ClientStatus    `json:"status"`
}

// ============================================================================
// ADMIN MODEL
// ============================================================================

type WhiteLabelAdmin struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	ClientID         uuid.NullUUID   `json:"clientId" db:"client_id"`
	Email            string          `json:"email" db:"email"`
	Name             string          `json:"name" db:"name"`
	PasswordHash     string          `json:"-" db:"password_hash"`
	Role             AdminRole       `json:"role" db:"role"`
	Permissions      json.RawMessage `json:"permissions" db:"permissions"`
	Status           string          `json:"status" db:"status"`
	TwoFactorEnabled bool            `json:"twoFactorEnabled" db:"two_factor_enabled"`
	TwoFactorSecret  sql.NullString  `json:"-" db:"two_factor_secret"`
	LastLogin        sql.NullTime    `json:"lastLogin" db:"last_login"`
	LoginAttempts    int             `json:"loginAttempts" db:"login_attempts"`
	LockedUntil      sql.NullTime    `json:"lockedUntil" db:"locked_until"`
	CreatedAt        time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time       `json:"updatedAt" db:"updated_at"`
}

type CreateAdminRequest struct {
	ClientID   *uuid.UUID `json:"clientId"`
	Email      string     `json:"email" binding:"required,email"`
	Name       string     `json:"name" binding:"required"`
	Password   string     `json:"password" binding:"required,min=8"`
	Role       AdminRole  `json:"role"`
	Permissions []string  `json:"permissions"`
}

type UpdateAdminRequest struct {
	Name        *string    `json:"name"`
	Role        *AdminRole `json:"role"`
	Permissions *[]string  `json:"permissions"`
	Status      *string    `json:"status"`
}

type LoginRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required"`
	TwoFactorCode string `json:"twoFactorCode"`
}

type LoginResponse struct {
	Token       string          `json:"token"`
	Admin       *WhiteLabelAdmin `json:"admin"`
	ExpiresAt   time.Time       `json:"expiresAt"`
}

// ============================================================================
// PRODUCT MODEL
// ============================================================================

type Product struct {
	ID          uuid.UUID     `json:"id" db:"id"`
	ClientID    uuid.NullUUID `json:"clientId" db:"client_id"`
	Name        string        `json:"name" db:"name"`
	Type        ProductType   `json:"type" db:"type"`
	Description sql.NullString `json:"description" db:"description"`
	Status      ProductStatus `json:"status" db:"status"`
	Fee         float64       `json:"fee" db:"fee"`
	MinDeposit  float64       `json:"minDeposit" db:"min_deposit"`
	MaxDeposit  float64       `json:"maxDeposit" db:"max_deposit"`
	MinWithdrawal float64    `json:"minWithdrawal" db:"min_withdrawal"`
	MaxWithdrawal float64    `json:"maxWithdrawal" db:"max_withdrawal"`
	Features    json.RawMessage `json:"features" db:"features"`
	Settings    json.RawMessage `json:"settings" db:"settings"`
	SortOrder   int           `json:"sortOrder" db:"sort_order"`
	CreatedAt   time.Time     `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time     `json:"updatedAt" db:"updated_at"`
}

type CreateProductRequest struct {
	ClientID    *uuid.UUID   `json:"clientId"`
	Name        string       `json:"name" binding:"required"`
	Type        ProductType  `json:"type" binding:"required"`
	Description string       `json:"description"`
	Status      ProductStatus `json:"status"`
	Fee         float64      `json:"fee"`
	MinDeposit  float64      `json:"minDeposit"`
	MaxDeposit  float64      `json:"maxDeposit"`
	MinWithdrawal float64   `json:"minWithdrawal"`
	MaxWithdrawal float64   `json:"maxWithdrawal"`
	Features    []string     `json:"features"`
	Settings    map[string]interface{} `json:"settings"`
	SortOrder   int          `json:"sortOrder"`
}

type UpdateProductRequest struct {
	Name          *string                `json:"name"`
	Description   *string                `json:"description"`
	Status        *ProductStatus         `json:"status"`
	Fee           *float64               `json:"fee"`
	MinDeposit    *float64               `json:"minDeposit"`
	MaxDeposit    *float64               `json:"maxDeposit"`
	MinWithdrawal *float64               `json:"minWithdrawal"`
	MaxWithdrawal *float64               `json:"maxWithdrawal"`
	Features      *[]string              `json:"features"`
	Settings      *map[string]interface{} `json:"settings"`
	SortOrder     *int                   `json:"sortOrder"`
}

// ============================================================================
// TRADING PAIR MODEL
// ============================================================================

type TradingPair struct {
	ID             uuid.UUID   `json:"id" db:"id"`
	ClientID       uuid.NullUUID `json:"clientId" db:"client_id"`
	BaseToken      string      `json:"baseToken" db:"base_token"`
	QuoteToken     string      `json:"quoteToken" db:"quote_token"`
	ChainID        int64       `json:"chainId" db:"chain_id"`
	PairAddress    sql.NullString `json:"pairAddress" db:"pair_address"`
	Status         PairStatus  `json:"status" db:"status"`
	Fee            float64     `json:"fee" db:"fee"`
	MinTrade       float64     `json:"minTrade" db:"min_trade"`
	MaxTrade       float64     `json:"maxTrade" db:"max_trade"`
	Liquidity      float64     `json:"liquidity" db:"liquidity"`
	PricePrecision int         `json:"pricePrecision" db:"price_precision"`
	QuantityPrecision int     `json:"quantityPrecision" db:"quantity_precision"`
	CreatedAt      time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time   `json:"updatedAt" db:"updated_at"`
}

type CreateTradingPairRequest struct {
	ClientID   *uuid.UUID `json:"clientId"`
	BaseToken  string    `json:"baseToken" binding:"required"`
	QuoteToken string    `json:"quoteToken" binding:"required"`
	ChainID    int64     `json:"chainId" binding:"required"`
	PairAddress string   `json:"pairAddress"`
	Fee        float64   `json:"fee"`
	MinTrade   float64   `json:"minTrade"`
	MaxTrade   float64   `json:"maxTrade"`
}

type UpdateTradingPairRequest struct {
	Status         *PairStatus `json:"status"`
	Fee            *float64    `json:"fee"`
	MinTrade       *float64    `json:"minTrade"`
	MaxTrade       *float64    `json:"maxTrade"`
	PairAddress    *string     `json:"pairAddress"`
}

// ============================================================================
// LIQUIDITY POOL MODEL
// ============================================================================

type LiquidityPool struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	ClientID    uuid.NullUUID `json:"clientId" db:"client_id"`
	PairID      uuid.UUID   `json:"pairId" db:"pair_id"`
	Provider    string      `json:"provider" db:"provider"`
	TokenA      string      `json:"tokenA" db:"token_a"`
	TokenB      string      `json:"tokenB" db:"token_b"`
	AmountA     float64     `json:"amountA" db:"amount_a"`
	AmountB     float64     `json:"amountB" db:"amount_b"`
	ValueUSD    float64     `json:"valueUsd" db:"value_usd"`
	APR         float64     `json:"apr" db:"apr"`
	Status      string      `json:"status" db:"status"`
	CreatedAt   time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time   `json:"updatedAt" db:"updated_at"`
}

type CreateLiquidityPoolRequest struct {
	ClientID uuid.UUID `json:"clientId" binding:"required"`
	PairID   uuid.UUID `json:"pairId" binding:"required"`
	Provider string    `json:"provider"`
	TokenA   string   `json:"tokenA" binding:"required"`
	TokenB   string   `json:"tokenB" binding:"required"`
	AmountA  float64  `json:"amountA" binding:"required"`
	AmountB  float64  `json:"amountB" binding:"required"`
}

type UpdateLiquidityPoolRequest struct {
	AmountA  *float64 `json:"amountA"`
	AmountB  *float64 `json:"amountB"`
	Provider *string  `json:"provider"`
	Status   *string  `json:"status"`
}

// ============================================================================
// TOKEN CONFIG MODEL
// ============================================================================

type TokenConfig struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	ClientID    uuid.NullUUID `json:"clientId" db:"client_id"`
	Address     string      `json:"address" db:"address"`
	Name        string      `json:"name" db:"name"`
	Symbol      string      `json:"symbol" db:"symbol"`
	Decimals    int         `json:"decimals" db:"decimals"`
	ChainID     int64       `json:"chainId" db:"chain_id"`
	Type        TokenType   `json:"type" db:"type"`
	Status      string      `json:"status" db:"status"`
	MaxSupply   sql.NullString `json:"maxSupply" db:"max_supply"`
	Features    json.RawMessage `json:"features" db:"features"`
	Metadata    json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt   time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time   `json:"updatedAt" db:"updated_at"`
}

type CreateTokenConfigRequest struct {
	ClientID  uuid.UUID `json:"clientId" binding:"required"`
	Address   string   `json:"address" binding:"required"`
	Name      string   `json:"name" binding:"required"`
	Symbol    string   `json:"symbol" binding:"required"`
	Decimals  int      `json:"decimals"`
	ChainID   int64    `json:"chainId" binding:"required"`
	Type      TokenType `json:"type"`
	MaxSupply string   `json:"maxSupply"`
	Features  []string  `json:"features"`
}

// ============================================================================
// MARKET MAKER BOT MODEL
// ============================================================================

type MarketMakerBot struct {
	ID          uuid.UUID        `json:"id" db:"id"`
	ClientID    uuid.NullUUID    `json:"clientId" db:"client_id"`
	Name       string           `json:"name" db:"name"`
	PairIDs    json.RawMessage `json:"pairIds" db:"pair_ids"`
	Strategy   BotStrategy      `json:"strategy" db:"strategy"`
	Status     BotStatus       `json:"status" db:"status"`
	Params     json.RawMessage `json:"params" db:"params"`
	Profit     float64         `json:"profit" db:"profit"`
	Volume24h  float64         `json:"volume24h" db:"volume_24h"`
	ErrorMessage sql.NullString `json:"errorMessage" db:"error_message"`
	CreatedAt  time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time       `json:"updatedAt" db:"updated_at"`
	StartedAt  sql.NullTime    `json:"startedAt" db:"started_at"`
	StoppedAt  sql.NullTime    `json:"stoppedAt" db:"stopped_at"`
}

type CreateBotRequest struct {
	ClientID uuid.UUID   `json:"clientId" binding:"required"`
	Name     string     `json:"name" binding:"required"`
	PairIDs  []string   `json:"pairIds" binding:"required"`
	Strategy BotStrategy `json:"strategy" binding:"required"`
	Params   map[string]interface{} `json:"params"`
}

type UpdateBotRequest struct {
	Name     *string                 `json:"name"`
	PairIDs  *[]string               `json:"pairIds"`
	Strategy *BotStrategy             `json:"strategy"`
	Params   *map[string]interface{} `json:"params"`
}

// ============================================================================
// BLOCKCHAIN MODEL
// ============================================================================

type Blockchain struct {
	ID          int64          `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Symbol      string         `json:"symbol" db:"symbol"`
	Category    ChainCategory  `json:"category" db:"category"`
	RPCUrls     json.RawMessage `json:"rpcUrls" db:"rpc_urls"`
	ExplorerUrls json.RawMessage `json:"explorerUrls" db:"explorer_urls"`
	Status      string         `json:"status" db:"status"`
	IsDefault   bool           `json:"isDefault" db:"is_default"`
	IconURL     sql.NullString `json:"iconUrl" db:"icon_url"`
	CreatedAt   time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time      `json:"updatedAt" db:"updated_at"`
}

type UpdateBlockchainRequest struct {
	Name         *string `json:"name"`
	RPCUrls       *[]string `json:"rpcUrls"`
	ExplorerUrls *[]string `json:"explorerUrls"`
	Status       *string  `json:"status"`
	IsDefault    *bool    `json:"isDefault"`
}

// ============================================================================
// API KEY MODEL
// ============================================================================

type APIKey struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	ClientID    uuid.UUID   `json:"clientId" db:"client_id"`
	Name        string      `json:"name" db:"name"`
	KeyHash     string      `json:"-" db:"key_hash"`
	SecretHash  sql.NullString `json:"-" db:"secret_hash"`
	Permissions json.RawMessage `json:"permissions" db:"permissions"`
	RateLimit   int         `json:"rateLimit" db:"rate_limit"`
	Status      string      `json:"status" db:"status"`
	LastUsed    sql.NullTime `json:"lastUsed" db:"last_used"`
	ExpiresAt   sql.NullTime `json:"expiresAt" db:"expires_at"`
	CreatedAt   time.Time   `json:"createdAt" db:"created_at"`
	RevokedAt   sql.NullTime `json:"revokedAt" db:"revoked_at"`
}

type CreateAPIKeyRequest struct {
	ClientID   uuid.UUID `json:"clientId" binding:"required"`
	Name       string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
	RateLimit  int      `json:"rateLimit"`
	ExpiresIn  int      `json:"expiresIn"` // days
}

type APIKeyResponse struct {
	ID          uuid.UUID `json:"id"`
	ClientID    uuid.UUID `json:"clientId"`
	Name        string   `json:"name"`
	Key         string   `json:"key"` // Only returned on creation
	Secret      string   `json:"secret"` // Only returned on creation
	Permissions []string `json:"permissions"`
	RateLimit   int      `json:"rateLimit"`
	Status      string   `json:"status"`
	ExpiresAt   *time.Time `json:"expiresAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ============================================================================
// AUDIT LOG MODEL
// ============================================================================

type AuditLog struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	ClientID     uuid.NullUUID `json:"clientId" db:"client_id"`
	AdminID      uuid.NullUUID `json:"adminId" db:"admin_id"`
	Action       string        `json:"action" db:"action"`
	ResourceType string        `json:"resourceType" db:"resource_type"`
	ResourceID   uuid.NullUUID `json:"resourceId" db:"resource_id"`
	Details      json.RawMessage `json:"details" db:"details"`
	IPAddress    sql.NullString `json:"ipAddress" db:"ip_address"`
	UserAgent    sql.NullString `json:"userAgent" db:"user_agent"`
	Status      string        `json:"status" db:"status"`
	CreatedAt   time.Time     `json:"createdAt" db:"created_at"`
}

type CreateAuditLogRequest struct {
	ClientID     *uuid.UUID `json:"clientId"`
	AdminID      *uuid.UUID `json:"adminId"`
	Action       string     `json:"action" binding:"required"`
	ResourceType string     `json:"resourceType" binding:"required"`
	ResourceID   *uuid.UUID `json:"resourceId"`
	Details      map[string]interface{} `json:"details"`
	IPAddress    string     `json:"ipAddress"`
	UserAgent    string     `json:"userAgent"`
	Status       string     `json:"status"`
}

// ============================================================================
// NOTIFICATION MODEL
// ============================================================================

type Notification struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	ClientID  uuid.NullUUID `json:"clientId" db:"client_id"`
	AdminID   uuid.NullUUID `json:"adminId" db:"admin_id"`
	Type      string        `json:"type" db:"type"`
	Title     string        `json:"title" db:"title"`
	Message   sql.NullString `json:"message" db:"message"`
	Data      json.RawMessage `json:"data" db:"data"`
	Read      bool          `json:"read" db:"read"`
	ReadAt    sql.NullTime  `json:"readAt" db:"read_at"`
	CreatedAt time.Time     `json:"createdAt" db:"created_at"`
}

type CreateNotificationRequest struct {
	ClientID *uuid.UUID              `json:"clientId"`
	AdminID  *uuid.UUID              `json:"adminId"`
	Type     string                  `json:"type" binding:"required"`
	Title    string                  `json:"title" binding:"required"`
	Message  string                  `json:"message"`
	Data     map[string]interface{}  `json:"data"`
}

// ============================================================================
// ANALYTICS MODEL
// ============================================================================

type AnalyticsDaily struct {
	ID               uuid.UUID `json:"id" db:"id"`
	ClientID         uuid.NullUUID `json:"clientId" db:"client_id"`
	Date             time.Time `json:"date" db:"date"`
	ActiveUsers      int       `json:"activeUsers" db:"active_users"`
	NewUsers         int       `json:"newUsers" db:"new_users"`
	TotalVolume      float64   `json:"totalVolume" db:"total_volume"`
	TradingVolume    float64   `json:"tradingVolume" db:"trading_volume"`
	FeesCollected    float64   `json:"feesCollected" db:"fees_collected"`
	TransactionsCount int       `json:"transactionsCount" db:"transactions_count"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
}

// ============================================================================
// PAGINATION & FILTERS
// ============================================================================

type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type PaginatedResponse struct {
	Total       int         `json:"total"`
	Page        int         `json:"page"`
	PageSize    int         `json:"pageSize"`
	TotalPages  int         `json:"totalPages"`
	Data        interface{} `json:"data"`
}

type SearchFilter struct {
	Query  string                 `json:"query"`
	Status string                 `json:"status"`
	SortBy string                 `json:"sortBy"`
	Order  string                 `json:"order"`
}

// ============================================================================
// DASHBOARD MODEL
// ============================================================================

type DashboardStats struct {
	TotalClients     int     `json:"totalClients"`
	ActiveClients   int     `json:"activeClients"`
	PendingClients  int     `json:"pendingClients"`
	TotalAdmins     int     `json:"totalAdmins"`
	TotalProducts   int     `json:"totalProducts"`
	TotalPairs      int     `json:"totalPairs"`
	TotalPools      int     `json:"totalPools"`
	TotalTokens     int     `json:"totalTokens"`
	TotalBots       int     `json:"totalBots"`
	TotalUsers      int     `json:"totalUsers"`
	Volume24h       float64 `json:"volume24h"`
	Volume7d        float64 `json:"volume7d"`
	Volume30d       float64 `json:"volume30d"`
	Revenue24h      float64 `json:"revenue24h"`
	Revenue7d       float64 `json:"revenue7d"`
	Revenue30d      float64 `json:"revenue30d"`
}
