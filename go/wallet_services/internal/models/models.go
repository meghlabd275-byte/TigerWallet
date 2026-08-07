/**
 * Data Models
 */

package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// JSON type for PostgreSQL JSONB
type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// User represents a user account
type User struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	Username          string     `json:"username"`
	PasswordHash     string     `json:"-"`
	FirstName         string     `json:"first_name"`
	LastName          string     `json:"last_name"`
	Phone             string     `json:"phone"`
	EmailVerified     bool       `json:"email_verified"`
	KYCStatus         string     `json:"kyc_status"`
	KYCLevel          int        `json:"kyc_level"`
	TwoFactorEnabled  bool       `json:"two_factor_enabled"`
	TwoFactorSecret   string     `json:"-"`
	RiskScore         int        `json:"risk_score"`
	Status            string     `json:"status"`
	ReferralCode      string     `json:"referral_code"`
	ReferrerID        string     `json:"referrer_id"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastLoginAt       *time.Time `json:"last_login_at"`
}

// Session represents a user session
type Session struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	TokenHash        string     `json:"-"`
	RefreshTokenHash string     `json:"-"`
	IPAddress        string     `json:"ip_address"`
	UserAgent        string     `json:"user_agent"`
	DeviceID         string     `json:"device_id"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RefreshExpiresAt *time.Time `json:"refresh_expires_at"`
	CreatedAt        time.Time  `json:"created_at"`
	LastActivityAt   time.Time  `json:"last_activity_at"`
}

// Wallet represents a crypto wallet
type Wallet struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	Name            string          `json:"name"`
	Type            string          `json:"type"`
	DerivationType string          `json:"derivation_type"`
	EncryptedSeed  string          `json:"encrypted_seed,omitempty"`
	PublicKey      string          `json:"public_key"`
	ChainType      string          `json:"chain_type"`
	ChainID        int64           `json:"chain_id"`
	Address        string          `json:"address"`
	DerivationPath string          `json:"derivation_path"`
	IsImported     bool            `json:"is_imported"`
	IsWatchOnly    bool            `json:"is_watch_only"`
	Status         string          `json:"status"`
	Metadata       JSON            `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Balance represents a token balance
type Balance struct {
	ID             string    `json:"id"`
	WalletID       string    `json:"wallet_id"`
	TokenAddress   string    `json:"token_address"`
	Symbol         string    `json:"symbol"`
	Name           string    `json:"name"`
	Decimals       int       `json:"decimals"`
	Balance        string    `json:"balance"`
	PendingBalance string    `json:"pending_balance"`
	LockedBalance  string    `json:"locked_balance"`
	IsNative       bool      `json:"is_native"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	WalletID       string     `json:"wallet_id"`
	TxHash         string     `json:"tx_hash"`
	ChainType      string     `json:"chain_type"`
	ChainID        int64      `json:"chain_id"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	FromAddress    string     `json:"from_address"`
	ToAddress      string     `json:"to_address"`
	Amount         string     `json:"amount"`
	TokenAddress   string     `json:"token_address"`
	TokenSymbol    string     `json:"token_symbol"`
	TokenDecimals  int        `json:"token_decimals"`
	Fee            string     `json:"fee"`
	FeeToken       string     `json:"fee_token"`
	Nonce          int64      `json:"nonce"`
	BlockNumber    int64      `json:"block_number"`
	BlockHash      string     `json:"block_hash"`
	Timestamp      *time.Time `json:"timestamp"`
	Confirmations  int        `json:"confirmations"`
	Data           string     `json:"data"`
	Metadata       JSON       `json:"metadata"`
	ErrorMessage   string     `json:"error_message"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Token represents a cryptocurrency token
type Token struct {
	ID             string     `json:"id"`
	Address        string     `json:"address"`
	ChainType      string     `json:"chain_type"`
	ChainID        int64      `json:"chain_id"`
	Symbol         string     `json:"symbol"`
	Name           string     `json:"name"`
	Decimals       int        `json:"decimals"`
	TotalSupply    string     `json:"total_supply"`
	IsVerified     bool       `json:"is_verified"`
	IsFake         bool       `json:"is_fake"`
	LogoURL        string     `json:"logo_url"`
	CoingeckoID    string     `json:"coingecko_id"`
	PriceUSD       float64    `json:"price_usd"`
	MarketCap      float64    `json:"market_cap"`
	Volume24h      float64    `json:"volume_24h"`
	PriceChange24h float64    `json:"price_change_24h"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Price represents token price data
type Price struct {
	Symbol     string  `json:"symbol"`
	PriceUSD   float64 `json:"price_usd"`
	Change24h  float64 `json:"change_24h"`
	Volume24h  float64 `json:"volume_24h"`
	MarketCap  float64 `json:"market_cap"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Notification represents a user notification
type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Data      JSON       `json:"data"`
	IsRead    bool       `json:"is_read"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// APIKey represents an API key for developers
type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"`
	Permissions JSON       `json:"permissions"`
	RateLimit   int        `json:"rate_limit"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID            string     `json:"id"`
	UserID        *string    `json:"user_id"`
	Action        string     `json:"action"`
	ResourceType string     `json:"resource_type"`
	ResourceID    string     `json:"resource_id"`
	OldValue     JSON       `json:"old_value"`
	NewValue     JSON       `json:"new_value"`
	IPAddress    string     `json:"ip_address"`
	UserAgent    string     `json:"user_agent"`
	Success      bool       `json:"success"`
	ErrorMessage string     `json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
}

// GasPrice represents gas price data
type GasPrice struct {
	ChainType      string  `json:"chain_type"`
	ChainID        int64   `json:"chain_id"`
	Slow           string  `json:"slow"`
	Standard       string  `json:"standard"`
	Fast           string  `json:"fast"`
	BaseFee        string  `json:"base_fee"`
	PriorityFee    string  `json:"priority_fee"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ChainInfo represents blockchain information
type ChainInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	Decimals     int    `json:"decimals"`
	ChainID      int64  `json:"chain_id"`
	RPCURL       string `json:"rpc_url"`
	ExplorerURL  string `json:"explorer_url"`
	IsEVM        bool   `json:"is_evm"`
	IsActive     bool   `json:"is_active"`
	Confirmations int   `json:"confirmations"`
}

// Portfolio represents user portfolio data
type Portfolio struct {
	UserID         string           `json:"user_id"`
	TotalValueUSD  float64          `json:"total_value_usd"`
	Change24h      float64          `json:"change_24h"`
	Change24hPercent float64        `json:"change_24h_percent"`
	Tokens         []PortfolioToken `json:"tokens"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// PortfolioToken represents a token in portfolio
type PortfolioToken struct {
	TokenID       string  `json:"token_id"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Balance       string  `json:"balance"`
	ValueUSD      float64 `json:"value_usd"`
	PriceUSD      float64 `json:"price_usd"`
	Change24h     float64 `json:"change_24h"`
	Allocation    float64 `json:"allocation"`
}

// Request/Response types

type RegisterRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	Username        string `json:"username" binding:"required,min=3,max=30"`
	ReferralCode    string `json:"referral_code"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreateWalletRequest struct {
	Name           string `json:"name" binding:"required"`
	ChainType      string `json:"chain_type" binding:"required"`
	DerivationType string `json:"derivation_type"`
}

type CreateTransactionRequest struct {
	WalletID      string `json:"wallet_id" binding:"required"`
	ToAddress      string `json:"to_address" binding:"required"`
	Amount         string `json:"amount" binding:"required"`
	TokenAddress   string `json:"token_address"`
	FeeLevel       string `json:"fee_level"` // slow, standard, fast
	Data           string `json:"data"`
}

type SendTokenRequest struct {
	WalletID    string `json:"wallet_id" binding:"required"`
	ToAddress   string `json:"to_address" binding:"required"`
	Amount      string `json:"amount" binding:"required"`
	TokenAddress string `json:"token_address"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         User   `json:"user"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    int         `json:"code"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}
