package models

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionTier represents the subscription plan levels
type SubscriptionTier string

const (
	TierFree     SubscriptionTier = "free"
	TierBasic    SubscriptionTier = "basic"
	TierPro      SubscriptionTier = "pro"
	TierEnterprise SubscriptionTier = "enterprise"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubStatusActive    SubscriptionStatus = "active"
	SubStatusTrialing SubscriptionStatus = "trialing"
	SubStatusPastDue  SubscriptionStatus = "past_due"
	SubStatusCanceled SubscriptionStatus = "canceled"
	SubStatusUnpaid   SubscriptionStatus = "unpaid"
	SubStatusIncomplete SubscriptionStatus = "incomplete"
)

// Plan represents a subscription plan
type Plan struct {
	ID                uuid.UUID        `json:"id" db:"id"`
	Name              string           `json:"name" db:"name"`
	Tier              SubscriptionTier `json:"tier" db:"tier"`
	Description       string           `json:"description" db:"description"`
	PriceMonthly      int64            `json:"price_monthly" db:"price_monthly"` // in cents
	PriceYearly       int64            `json:"price_yearly" db:"price_yearly"`   // in cents
	FeatureFlags      []string         `json:"feature_flags" db:"feature_flags"`
	APIQuotaMonthly   int64            `json:"api_quota_monthly" db:"api_quota_monthly"`
	StorageQuotaGB    int64            `json:"storage_quota_gb" db:"storage_quota_gb"`
	MaxUsers          int              `json:"max_users" db:"max_users"`
	MaxWallets        int              `json:"max_wallets" db:"max_wallets"`
	MaxBots           int              `json:"max_bots" db:"max_bots"`
	SupportLevel      string           `json:"support_level" db:"support_level"`
	IsActive          bool             `json:"is_active" db:"is_active"`
	StripePriceID     string           `json:"stripe_price_id" db:"stripe_price_id"`
	CreatedAt         time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at" db:"updated_at"`
}

// Subscription represents a client's subscription
type Subscription struct {
	ID                   uuid.UUID           `json:"id" db:"id"`
	TenantID             uuid.UUID           `json:"tenant_id" db:"tenant_id"`
	PlanID               uuid.UUID           `json:"plan_id" db:"plan_id"`
	StripeSubscriptionID string              `json:"stripe_subscription_id" db:"stripe_subscription_id"`
	Status               SubscriptionStatus  `json:"status" db:"status"`
	CurrentPeriodStart   time.Time           `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd     time.Time           `json:"current_period_end" db:"current_period_end"`
	TrialStart           *time.Time          `json:"trial_start" db:"trial_start"`
	TrialEnd             *time.Time          `json:"trial_end" db:"trial_end"`
	CancelAtPeriodEnd    bool                `json:"cancel_at_period_end" db:"cancel_at_period_end"`
	CanceledAt           *time.Time          `json:"canceled_at" db:"canceled_at"`
	CreatedAt            time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at" db:"updated_at"`
	Plan                 *Plan               `json:"plan,omitempty"`
}

// UsageRecord tracks API usage for quota management
type UsageRecord struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	APIMethod   string    `json:"api_method" db:"api_method"`
	Count       int64     `json:"count" db:"count"`
	PeriodStart time.Time `json:"period_start" db:"period_start"`
	PeriodEnd   time.Time `json:"period_end" db:"period_end"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// UsageSummary represents usage for a billing period
type UsageSummary struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	TotalAPICalls    int64     `json:"total_api_calls"`
	StorageUsedGB    float64   `json:"storage_used_gb"`
	ActiveUsers      int       `json:"active_users"`
	ActiveWallets    int       `json:"active_wallets"`
	ActiveBots       int       `json:"active_bots"`
	OverageAPICalls int64     `json:"overage_api_calls"`
	OverageStorageGB float64   `json:"overage_storage_gb"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
}

// Invoice represents an invoice
type Invoice struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	TenantID          uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	InvoiceNumber     string     `json:"invoice_number" db:"invoice_number"`
	StripeInvoiceID   string     `json:"stripe_invoice_id" db:"stripe_invoice_id"`
	Amount            int64      `json:"amount" db:"amount"` // in cents
	AmountDue         int64      `json:"amount_due" db:"amount_due"`
	AmountPaid        int64      `json:"amount_paid" db:"amount_paid"`
	Currency          string     `json:"currency" db:"currency"`
	Status            string     `json:"status" db:"status"`
	DueDate           time.Time  `json:"due_date" db:"due_date"`
	PaidAt            *time.Time `json:"paid_at" db:"paid_at"`
	InvoiceURL        string     `json:"invoice_url" db:"invoice_url"`
	InvoicePDF        string     `json:"invoice_pdf" db:"invoice_pdf"`
	LineItems         []LineItem `json:"line_items"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// LineItem represents a line item in an invoice
type LineItem struct {
	ID          uuid.UUID `json:"id" db:"id"`
	InvoiceID   uuid.UUID `json:"invoice_id" db:"invoice_id"`
	Description string    `json:"description" db:"description"`
	Quantity    int64     `json:"quantity" db:"quantity"`
	UnitPrice   int64     `json:"unit_price" db:"unit_price"` // in cents
	Amount      int64     `json:"amount" db:"amount"`          // in cents
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Payment represents a payment record
type Payment struct {
	ID                uuid.UUID `json:"id" db:"id"`
	TenantID          uuid.UUID `json:"tenant_id" db:"tenant_id"`
	InvoiceID         uuid.UUID `json:"invoice_id" db:"invoice_id"`
	StripePaymentID   string    `json:"stripe_payment_id" db:"stripe_payment_id"`
	Amount            int64      `json:"amount" db:"amount"` // in cents
	Currency          string     `json:"currency" db:"currency"`
	Status            string     `json:"status" db:"status"`
	PaymentMethod     string     `json:"payment_method" db:"payment_method"`
	PaymentMethodID   string     `json:"payment_method_id" db:"payment_method_id"`
	ReceiptURL        string     `json:"receipt_url" db:"receipt_url"`
	FailureReason     string     `json:"failure_reason" db:"failure_reason"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// PaymentMethod represents a payment method
type PaymentMethod struct {
	ID                uuid.UUID `json:"id" db:"id"`
	TenantID          uuid.UUID `json:"tenant_id" db:"tenant_id"`
	StripePaymentID   string    `json:"stripe_payment_id" db:"stripe_payment_id"`
	Type              string    `json:"type" db:"type"` // card, bank_account
	CardBrand         *string   `json:"card_brand" db:"card_brand"`
	Last4             *string   `json:"last4" db:"last4"`
	ExpMonth          *int      `json:"exp_month" db:"exp_month"`
	ExpYear           *int      `json:"exp_year" db:"exp_year"`
	IsDefault         bool      `json:"is_default" db:"is_default"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// Tenant represents a white label client/tenant
type Tenant struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	Name                string    `json:"name" db:"name"`
	Slug                string    `json:"slug" db:"slug"`
	Email               string    `json:"email" db:"email"`
	StripeCustomerID    string    `json:"stripe_customer_id" db:"stripe_customer_id"`
	SubscriptionID      *uuid.UUID `json:"subscription_id" db:"subscription_id"`
	Status              string    `json:"status" db:"status"` // active, suspended, terminated
	APIKeys            []APIKey  `json:"api_keys"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// APIKey represents API keys for tenant access
type APIKey struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Key         string    `json:"key" db:"key"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Permissions []string  `json:"permissions" db:"permissions"`
	RateLimit   int       `json:"rate_limit" db:"rate_limit"` // requests per minute
	IsActive    bool      `json:"is_active" db:"is_active"`
	LastUsedAt  *time.Time `json:"last_used_at" db:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Webhook represents webhook configurations
type Webhook struct {
	ID            uuid.UUID `json:"id" db:"id"`
	TenantID      uuid.UUID `json:"tenant_id" db:"tenant_id"`
	URL           string    `json:"url" db:"url"`
	Events        []string `json:"events" db:"events"`
	Secret        string    `json:"secret" db:"secret"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	FailureCount  int       `json:"failure_count" db:"failure_count"`
	LastFailureAt *time.Time `json:"last_failure_at" db:"last_failure_at"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// FeatureFlag represents feature flags for plans
type FeatureFlag struct {
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// APIRequest represents an API request for logging
type APIRequest struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	APIKeyID    uuid.UUID `json:"api_key_id" db:"api_key_id"`
	Method      string    `json:"method" db:"method"`
	Path        string    `json:"path" db:"path"`
	StatusCode  int       `json:"status_code" db:"status_code"`
	LatencyMs   int64     `json:"latency_ms" db:"latency_ms"`
	IPAddress   string    `json:"ip_address" db:"ip_address"`
	UserAgent   string    `json:"user_agent" db:"user_agent"`
	RequestSize int64     `json:"request_size" db:"request_size"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
