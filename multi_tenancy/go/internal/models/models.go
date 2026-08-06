package models

import (
	"time"

	"github.com/google/uuid"
)

// TenantStatus represents the status of a tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusTerminated TenantStatus = "terminated"
	TenantStatusTrial     TenantStatus = "trial"
)

// Tenant represents a white label client organization
type Tenant struct {
	ID                uuid.UUID    `json:"id" db:"id"`
	Name              string       `json:"name" db:"name"`
	Slug              string       `json:"slug" db:"slug"`
	Email             string       `json:"email" db:"email"`
	Status            TenantStatus `json:"status" db:"status"`
	PlanID            uuid.UUID    `json:"plan_id" db:"plan_id"`
	CustomDomain      *string      `json:"custom_domain" db:"custom_domain"`
	LogoURL           *string      `json:"logo_url" db:"logo_url"`
	PrimaryColor      *string      `json:"primary_color" db:"primary_color"`
	SecondaryColor    *string      `json:"secondary_color" db:"secondary_color"`
	Timezone          string       `json:"timezone" db:"timezone"`
	Language          string       `json:"language" db:"language"`
	Features          []string     `json:"features" db:"features"`
	Metadata          map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt         time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at" db:"updated_at"`
	TrialEndsAt       *time.Time   `json:"trial_ends_at" db:"trial_ends_at"`
}

// TenantUser represents a user within a tenant
type TenantUser struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Role      string    `json:"role" db:"role"` // admin, manager, member, viewer
	Permissions []string `json:"permissions" db:"permissions"`
	Status    string    `json:"status" db:"status"` // active, invited, suspended
	InvitedAt *time.Time `json:"invited_at" db:"invited_at"`
	JoinedAt  *time.Time `json:"joined_at" db:"joined_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// Quota represents the resource quota for a tenant
type Quota struct {
	ID             uuid.UUID `json:"id" db:"id"`
	TenantID       uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Resource       string    `json:"resource" db:"resource"` // api_calls, storage, users, wallets, bots
	Limit          int64     `json:"limit" db:"limit"`
	Used           int64     `json:"used" db:"used"`
	PeriodStart    time.Time `json:"period_start" db:"period_start"`
	PeriodEnd      time.Time `json:"period_end" db:"period_end"`
	ResetAt        time.Time `json:"reset_at" db:"reset_at"`
}

// TenantConfig represents tenant-specific configuration
type TenantConfig struct {
	ID                uuid.UUID              `json:"id" db:"id"`
	TenantID          uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	WalletSettings    *WalletSettings       `json:"wallet_settings" db:"wallet_settings"`
	BotSettings       *BotSettings          `json:"bot_settings" db:"bot_settings"`
	TokenSettings     *TokenSettings        `json:"token_settings" db:"token_settings"`
	SecuritySettings  *SecuritySettings     `json:"security_settings" db:"security_settings"`
	NotificationSettings *NotificationSettings `json:"notification_settings" db:"notification_settings"`
	CreatedAt         time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at" db:"updated_at"`
}

type WalletSettings struct {
	SupportedChains    []string `json:"supported_chains"`
	MaxWalletsPerUser  int      `json:"max_wallets_per_user"`
	EnableSwap         bool     `json:"enable_swap"`
	EnableStaking      bool     `json:"enable_staking"`
	EnableNFT          bool     `json:"enable_nft"`
	MinTransactionUSD  float64  `json:"min_transaction_usd"`
	MaxTransactionUSD  float64  `json:"max_transaction_usd"`
}

type BotSettings struct {
	MaxBots               int      `json:"max_bots"`
	EnableTradingBots     bool     `json:"enable_trading_bots"`
	EnableArbitrageBots  bool     `json:"enable_arbitrage_bots"`
	EnableSignalBots      bool     `json:"enable_signal_bots"`
	MaxBotCapitalUSD     float64  `json:"max_bot_capital_usd"`
	RequireKYCForBots    bool     `json:"require_kyc_for_bots"`
}

type TokenSettings struct {
	EnableTokenCreation  bool     `json:"enable_token_creation"`
	EnableTokenListing   bool     `json:"enable_token_listing"`
	EnableLaunchpad      bool     `json:"enable_launchpad"`
	ListingFeeUSD        float64  `json:"listing_fee_usd"`
	RequireKYCForTokens  bool     `json:"require_kyc_for_tokens"`
}

type SecuritySettings struct {
	RequireMFA            bool     `json:"require_mfa"`
	SessionTimeoutMinutes int      `json:"session_timeout_minutes"`
	MaxLoginAttempts      int      `json:"max_login_attempts"`
	IPWhitelistEnabled    bool     `json:"ip_whitelist_enabled"`
	IPWhitelist          []string `json:"ip_whitelist"`
	AllowedEmailDomains  []string `json:"allowed_email_domains"`
}

type NotificationSettings struct {
	EmailEnabled    bool     `json:"email_enabled"`
	SMSEnabled     bool     `json:"sms_enabled"`
	PushEnabled    bool     `json:"push_enabled"`
	WebhookEnabled bool     `json:"webhook_enabled"`
	WebhookURL     *string  `json:"webhook_url"`
}

// TenantFeature represents enabled features for a tenant
type TenantFeature struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Feature     string    `json:"feature" db:"feature"`
	Enabled     bool      `json:"enabled" db:"enabled"`
	Limit       *int64    `json:"limit" db:"limit"`
	Used        int64     `json:"used" db:"used"`
	ExpiresAt   *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// TenantInvitation represents an invitation to join a tenant
type TenantInvitation struct {
	ID         uuid.UUID `json:"id" db:"id"`
	TenantID   uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Email      string    `json:"email" db:"email"`
	Role       string    `json:"role" db:"role"`
	Token      string    `json:"token" db:"token"`
	Status     string    `json:"status" db:"status"` // pending, accepted, expired
	InvitedBy  uuid.UUID `json:"invited_by" db:"invited_by"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// TenantAuditLog represents audit log entries for tenant actions
type TenantAuditLog struct {
	ID         uuid.UUID `json:"id" db:"id"`
	TenantID   uuid.UUID `json:"tenant_id" db:"tenant_id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Action     string    `json:"action" db:"action"`
	Resource   string    `json:"resource" db:"resource"`
	ResourceID *uuid.UUID `json:"resource_id" db:"resource_id"`
	Details    map[string]interface{} `json:"details" db:"details"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
	UserAgent  string    `json:"user_agent" db:"user_agent"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
