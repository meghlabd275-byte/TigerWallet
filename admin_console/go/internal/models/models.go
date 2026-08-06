package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID  `json:"id"`
	Email             string     `json:"email"`
	Username          string     `json:"username"`
	PasswordHash      string     `json:"-"`
	FirstName         *string    `json:"first_name,omitempty"`
	LastName          *string    `json:"last_name,omitempty"`
	Phone             *string    `json:"phone,omitempty"`
	Role              string     `json:"role"`
	Status            string     `json:"status"`
	EmailVerified     bool       `json:"email_verified"`
	TwoFactorEnabled  bool       `json:"two_factor_enabled"`
	TwoFactorSecret   *string    `json:"-"`
	LastLoginAt       *time.Time `json:"last_login_at"`
	LoginAttempts     int        `json:"login_attempts"`
	LockedUntil       *time.Time `json:"locked_until"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type UserProfile struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	AvatarURL     *string   `json:"avatar_url,omitempty"`
	Bio           *string   `json:"bio,omitempty"`
	DateOfBirth   *Date     `json:"date_of_birth,omitempty"`
	Address       *string   `json:"address,omitempty"`
	City          *string   `json:"city,omitempty"`
	Country       *string   `json:"country,omitempty"`
	PostalCode    *string   `json:"postal_code,omitempty"`
	Timezone      string    `json:"timezone"`
	Language      string    `json:"language"`
	Preferences   map[string]interface{} `json:"preferences"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Date time.Time

func (d *Date) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = s[1 : len(s)-1] // Remove quotes
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	*d = Date(t)
	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format("2006-01-02") + `"`), nil
}

type KYCReview struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	DocumentType    string     `json:"document_type"`
	DocumentNumber  *string    `json:"document_number,omitempty"`
	DocumentFront   *string    `json:"document_front,omitempty"`
	DocumentBack    *string    `json:"document_back,omitempty"`
	SelfieURL       *string    `json:"selfie_url,omitempty"`
	ProofOfAddress  *string    `json:"proof_of_address,omitempty"`
	Status          string     `json:"status"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	ReviewedBy      *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Token struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Symbol          string     `json:"symbol"`
	ContractAddress *string    `json:"contract_address,omitempty"`
	Chain           string     `json:"chain"`
	Decimals        int        `json:"decimals"`
	TotalSupply     *string    `json:"total_supply,omitempty"`
	Description     *string    `json:"description,omitempty"`
	LogoURL         *string    `json:"logo_url,omitempty"`
	WebsiteURL      *string    `json:"website_url,omitempty"`
	WhitepaperURL   *string    `json:"whitepaper_url,omitempty"`
	Status          string     `json:"status"`
	ListingFee      *string    `json:"listing_fee,omitempty"`
	ApprovedBy      *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	RejectedBy      *uuid.UUID `json:"rejected_by,omitempty"`
	RejectedAt      *time.Time `json:"rejected_at,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Transaction struct {
	ID            uuid.UUID              `json:"id"`
	TxHash        string                 `json:"tx_hash"`
	UserID        uuid.UUID              `json:"user_id"`
	TokenID       *uuid.UUID             `json:"token_id,omitempty"`
	Type          string                 `json:"type"`
	Amount        string                 `json:"amount"`
	Fee           *string                `json:"fee,omitempty"`
	Status        string                 `json:"status"`
	FlagReason    *string                `json:"flag_reason,omitempty"`
	FlaggedBy     *uuid.UUID             `json:"flagged_by,omitempty"`
	FlaggedAt     *time.Time             `json:"flagged_at,omitempty"`
	ApprovedBy    *uuid.UUID             `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time             `json:"approved_at,omitempty"`
	RejectedBy    *uuid.UUID             `json:"rejected_by,omitempty"`
	RejectedAt    *time.Time             `json:"rejected_at,omitempty"`
	FromAddress   *string                `json:"from_address,omitempty"`
	ToAddress     *string                `json:"to_address,omitempty"`
	BlockNumber   *int64                 `json:"block_number,omitempty"`
	Chain         *string                `json:"chain,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type AuditLog struct {
	ID            uuid.UUID              `json:"id"`
	UserID        *uuid.UUID            `json:"user_id,omitempty"`
	Action        string                 `json:"action"`
	ResourceType  string                 `json:"resource_type"`
	ResourceID    *uuid.UUID            `json:"resource_id,omitempty"`
	OldValues     map[string]interface{} `json:"old_values,omitempty"`
	NewValues     map[string]interface{} `json:"new_values,omitempty"`
	IPAddress     *string               `json:"ip_address,omitempty"`
	UserAgent     *string               `json:"user_agent,omitempty"`
	Location      *string               `json:"location,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

type Notification struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Type      string     `json:"type"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type Session struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"-"`
	IPAddress *string   `json:"ip_address,omitempty"`
	UserAgent *string   `json:"user_agent,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type APIKey struct {
	ID          uuid.UUID              `json:"id"`
	UserID      uuid.UUID              `json:"user_id"`
	Name        string                 `json:"name"`
	KeyHash     string                 `json:"-"`
	Permissions []string               `json:"permissions"`
	RateLimit   int                    `json:"rate_limit"`
	LastUsedAt  *time.Time             `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	RevokedAt   *time.Time             `json:"revoked_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

type ComplianceReport struct {
	ID          uuid.UUID  `json:"id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	GeneratedBy *uuid.UUID `json:"generated_by,omitempty"`
	DateFrom    *time.Time `json:"date_from,omitempty"`
	DateTo      *time.Time `json:"date_to,omitempty"`
	Status      string     `json:"status"`
	FileURL     *string    `json:"file_url,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type SystemConfig struct {
	Key         string     `json:"key"`
	Value       string     `json:"value"`
	Description *string    `json:"description,omitempty"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type UserActivity struct {
	ID        uuid.UUID              `json:"id"`
	UserID    uuid.UUID              `json:"user_id"`
	Action    string                 `json:"action"`
	Details   map[string]interface{} `json:"details"`
	IPAddress *string               `json:"ip_address,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// Request/Response DTOs
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type RegisterRequest struct {
	Email    string  `json:"email" binding:"required,email"`
	Username string  `json:"username" binding:"required,min=3,max=50"`
	Password string  `json:"password" binding:"required,min=8"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         *User  `json:"user"`
}

type DashboardStats struct {
	TotalUsers        int64   `json:"total_users"`
	ActiveUsers       int64   `json:"active_users"`
	TotalTransactions int64   `json:"total_transactions"`
	PendingKYC       int64   `json:"pending_kyc"`
	PendingTokens    int64   `json:"pending_tokens"`
	Revenue          float64 `json:"revenue"`
	Change24h        float64 `json:"change_24h"`
}

type TransactionStats struct {
	TotalVolume    float64 `json:"total_volume"`
	TodayVolume    float64 `json:"today_volume"`
	WeekVolume     float64 `json:"week_volume"`
	MonthVolume    float64 `json:"month_volume"`
	TotalCount     int64   `json:"total_count"`
	PendingCount   int64   `json:"pending_count"`
	FlaggedCount   int64   `json:"flagged_count"`
}

type PaginationParams struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type PaginatedResponse struct {
	Total       int         `json:"total"`
	Page        int         `json:"page"`
	PageSize    int         `json:"page_size"`
	TotalPages  int         `json:"total_pages"`
	Data        interface{} `json:"data"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
