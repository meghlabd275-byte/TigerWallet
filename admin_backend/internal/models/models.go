package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Admin represents an admin user
type Admin struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         sql.NullTime   `gorm:"index" json:"-"`
	Username          string         `gorm:"uniqueIndex;not null" json:"username"`
	Email             string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash      string         `gorm:"not null" json:"-"`
	FirstName         string         `json:"first_name"`
	LastName          string         `json:"last_name"`
	Role              string         `gorm:"not null;default:'admin'" json:"role"` // super_admin, admin, support, analyst, moderator
	Permissions       json.RawMessage `gorm:"type:jsonb" json:"permissions"`
	Status            string         `gorm:"not null;default:'active'" json:"status"` // active, suspended, inactive
	TwoFactorEnabled  bool           `gorm:"default:false" json:"two_factor_enabled"`
	TwoFactorSecret   string         `json:"-"`
	LastLoginAt       *time.Time    `json:"last_login_at"`
	FailedAttempts    int           `gorm:"default:0" json:"failed_attempts"`
	LockedUntil      *time.Time    `json:"locked_until"`
	IPWhitelist      sql.NullString `gorm:"type:text" json:"-"`
	LastIP           string         `json:"last_ip"`
	EmailVerified    bool           `gorm:"default:false" json:"email_verified"`
	PasswordChangedAt *time.Time    `json:"password_changed_at"`
}

// AdminSession represents an admin login session
type AdminSession struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    sql.NullTime   `gorm:"index" json:"-"`
	AdminID      uint           `gorm:"not null;index" json:"admin_id"`
	Token        string         `gorm:"uniqueIndex;not null" json:"token"`
	RefreshToken string         `gorm:"uniqueIndex" json:"refresh_token"`
	ExpiresAt    time.Time      `gorm:"not null" json:"expires_at"`
	IPAddress    string         `json:"ip_address"`
	UserAgent    string         `json:"user_agent"`
	Revoked     bool           `gorm:"default:false" json:"revoked"`
	RevokedAt   *time.Time    `json:"revoked_at"`
}

// AdminActivity represents admin action logs
type AdminActivity struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	AdminID     uint           `gorm:"not null;index" json:"admin_id"`
	Admin       *Admin         `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
	Action      string         `gorm:"not null" json:"action"` // create, update, delete, login, logout, etc.
	Resource    string         `json:"resource"` // User, Transaction, Token, etc.
	ResourceID  string         `json:"resource_id"`
	Details     json.RawMessage `gorm:"type:jsonb" json:"details"`
	IPAddress   string         `json:"ip_address"`
	UserAgent   string         `json:"user_agent"`
	Status      string         `gorm:"default:'success'" json:"status"` // success, failed
	ErrorMessage string        `json:"error_message,omitempty"`
}

// User represents a platform user
type User struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         sql.NullTime   `gorm:"index" json:"-"`
	Email             string         `gorm:"uniqueIndex;not null" json:"email"`
	Username          string         `gorm:"index" json:"username"`
	Phone             sql.NullString `gorm:"uniqueIndex" json:"phone"`
	WalletAddress     string         `gorm:"index" json:"wallet_address"`
	Status            string         `gorm:"not null;default:'active'" json:"status"` // active, pending, suspended, banned
	KYCStatus         string         `gorm:"not null;default:'none'" json:"kyc_status"` // none, pending, level1, level2, level3, rejected
	KYCLevel          int            `gorm:"default:0" json:"kyc_level"`
	KYCSubmittedAt    *time.Time    `json:"kyc_submitted_at"`
	KYCVerifiedAt     *time.Time    `json:"kyc_verified_at"`
	KYCRejectionReason string        `json:"kyc_rejection_reason"`
	TwoFactorEnabled  bool           `gorm:"default:false" json:"two_factor_enabled"`
	ReferralCode      string         `gorm:"uniqueIndex" json:"referral_code"`
	ReferredBy       string         `json:"referred_by"`
	WhiteLabelID     *uint          `gorm:"index" json:"white_label_id"`
	LastLoginAt      *time.Time    `json:"last_login_at"`
	RegistrationIP   string         `json:"registration_ip"`
	RiskScore        int            `gorm:"default:0" json:"risk_score"` // 0-100
	Tags             json.RawMessage `gorm:"type:jsonb" json:"tags"`
	Metadata         json.RawMessage `gorm:"type:jsonb" json:"metadata"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     sql.NullTime   `gorm:"index" json:"-"`
	UserID        uint           `gorm:"not null;index" json:"user_id"`
	User          *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Hash          string         `gorm:"uniqueIndex;not null" json:"hash"`
	Type          string         `gorm:"not null" json:"type"` // transfer, swap, stake, unstake, bridge, withdraw, deposit
	Chain         string         `gorm:"not null" json:"chain"` // ethereum, bsc, polygon, etc.
	FromAddress   string         `json:"from_address"`
	ToAddress     string         `json:"to_address"`
	Amount        string         `gorm:"type:decimal(36,18)" json:"amount"`
	Token         string         `json:"token"` // ETH, USDT, etc.
	TokenAmount   string         `gorm:"type:decimal(36,18)" json:"token_amount"`
	Status        string         `gorm:"not null;default:'pending'" json:"status"` // pending, confirmed, failed
	BlockNumber   int64          `json:"block_number"`
	BlockHash     string         `json:"block_hash"`
	GasUsed        string         `json:"gas_used"`
	GasPrice       string         `json:"gas_price"`
	Nonce          int64          `json:"nonce"`
	Timestamp      time.Time      `json:"timestamp"`
	Metadata       json.RawMessage `gorm:"type:jsonb" json:"metadata"`
	Flagged        bool           `gorm:"default:false" json:"flagged"`
	FlagReason     string         `json:"flag_reason"`
}

// Token represents a cryptocurrency token
type Token struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     sql.NullTime   `gorm:"index" json:"-"`
	Name          string         `gorm:"not null" json:"name"`
	Symbol        string         `gorm:"not null;index" json:"symbol"`
	ContractAddress string       `gorm:"index" json:"contract_address"`
	Chain         string         `gorm:"not null" json:"chain"`
	Decimals      int            `gorm:"not null;default:18" json:"decimals"`
	TotalSupply   string         `gorm:"type:decimal(36,0)" json:"total_supply"`
	LogoURL       string         `json:"logo_url"`
	Website       string         `json:"website"`
	Description   string         `json:"description"`
	Price         string         `gorm:"type:decimal(36,8)" json:"price"`
	MarketCap     string         `gorm:"type:decimal(36,2)" json:"market_cap"`
	Volume24h     string         `gorm:"type:decimal(36,2)" json:"volume_24h"`
	PriceChange24h string        `gorm:"type:decimal(10,4)" json:"price_change_24h"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	IsVerified    bool           `gorm:"default:false" json:"is_verified"`
	ListingFee    string         `gorm:"type:decimal(36,8)" json:"listing_fee"`
	ListedBy      uint           `json:"listed_by"`
	ListedAt      *time.Time    `json:"listed_at"`
	Metadata      json.RawMessage `gorm:"type:jsonb" json:"metadata"`
}

// KYCApplication represents a KYC submission
type KYCApplication struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     sql.NullTime   `gorm:"index" json:"-"`
	UserID        uint           `gorm:"not null;uniqueIndex" json:"user_id"`
	User          *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Level         int            `gorm:"not null" json:"level"` // 1, 2, 3
	Status        string         `gorm:"not null;default:'pending'" json:"status"` // pending, approved, rejected
	SubmittedAt   time.Time      `json:"submitted_at"`
	ReviewedAt    *time.Time    `json:"reviewed_at"`
	ReviewedBy    *uint         `json:"reviewed_by"`
	Reviewer      *Admin        `gorm:"foreignKey:ReviewedBy" json:"reviewer,omitempty"`
	RejectionReason string       `json:"rejection_reason"`
	Documents     json.RawMessage `gorm:"type:jsonb" json:"documents"`
	IPAddress     string         `json:"ip_address"`
	Notes         string         `json:"notes"`
}

// Withdrawal represents a withdrawal request
type Withdrawal struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     sql.NullTime   `gorm:"index" json:"-"`
	UserID        uint           `gorm:"not null;index" json:"user_id"`
	User          *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Amount        string         `gorm:"type:decimal(36,18);not null" json:"amount"`
	Token         string         `gorm:"not null" json:"token"`
	Chain         string         `gorm:"not null" json:"chain"`
	ToAddress     string         `gorm:"not null" json:"to_address"`
	Status        string         `gorm:"not null;default:'pending'" json:"status"` // pending, approved, rejected, processing, completed, failed
	ApprovedAt    *time.Time    `json:"approved_at"`
	ApprovedBy    *uint         `json:"approved_by"`
	Approver      *Admin        `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
	RejectedAt    *time.Time    `json:"rejected_at"`
	RejectedBy   *uint         `json:"rejected_by"`
	RejectionReason string       `json:"rejection_reason"`
	ProcessedAt   *time.Time    `json:"processed_at"`
	TxHash        string         `json:"tx_hash"`
	Fee           string         `gorm:"type:decimal(36,18)" json:"fee"`
	IPAddress     string         `json:"ip_address"`
	Notes         string         `json:"notes"`
}

// WhiteLabel represents a white label client
type WhiteLabel struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       sql.NullTime   `gorm:"index" json:"-"`
	Name            string         `gorm:"not null" json:"name"`
	Slug            string         `gorm:"uniqueIndex;not null" json:"slug"`
	Domain          string         `gorm:"uniqueIndex" json:"domain"`
	LogoURL         string         `json:"logo_url"`
	FaviconURL      string         `json:"favicon_url"`
	PrimaryColor    string         `json:"primary_color"`
	SecondaryColor  string         `json:"secondary_color"`
	Status          string         `gorm:"not null;default:'active'" json:"status"` // active, suspended, pending
	ContactEmail    string         `json:"contact_email"`
	ContactPhone    string         `json:"contact_phone"`
	Address         string         `json:"address"`
	Description     string         `json:"description"`
	CustomCSS       sql.NullString `gorm:"type:text" json:"custom_css"`
	CustomJS        sql.NullString `gorm:"type:text" json:"custom_js"`
	Features        json.RawMessage `gorm:"type:jsonb" json:"features"`
	FeeStructure    json.RawMessage `gorm:"type:jsonb" json:"fee_structure"`
	ApprovedAt      *time.Time    `json:"approved_at"`
	ApprovedBy      *uint         `json:"approved_by"`
	ExpiresAt       *time.Time    `json:"expires_at"`
}

// SystemConfig represents system configuration
type SystemConfig struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Key         string         `gorm:"uniqueIndex;not null" json:"key"`
	Value       string         `gorm:"type:text" json:"value"`
	Type        string         `gorm:"not null;default:'string'" json:"type"` // string, int, bool, json
	Description string         `json:"description"`
	UpdatedBy  *uint          `json:"updated_by"`
	Category    string         `gorm:"index" json:"category"` // fees, limits, features, etc.
	IsPublic    bool           `gorm:"default:false" json:"is_public"`
}

// APIKey represents an API key for external access
type APIKey struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"created_at"`
	DeletedAt     sql.NullTime   `gorm:"index" json:"-"`
	Name          string         `gorm:"not null" json:"name"`
	Key           string         `gorm:"uniqueIndex;not null" json:"key"`
	KeyHash       string         `gorm:"not null;index" json:"-"`
	AdminID       uint           `gorm:"not null;index" json:"admin_id"`
	Admin         *Admin         `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
	Permissions   json.RawMessage `gorm:"type:jsonb" json:"permissions"`
	RateLimit     int            `gorm:"default:1000" json:"rate_limit"` // requests per hour
	LastUsedAt    *time.Time    `json:"last_used_at"`
	ExpiresAt     *time.Time    `json:"expires_at"`
	Status        string         `gorm:"not null;default:'active'" json:"status"` // active, suspended, expired
	Scopes        json.RawMessage `gorm:"type:jsonb" json:"scopes"`
}

// AuditLog represents a system audit log
type AuditLog struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	AdminID     uint           `gorm:"index" json:"admin_id"`
	Admin       *Admin         `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
	Action      string         `gorm:"not null;index" json:"action"`
	Entity      string         `gorm:"not null;index" json:"entity"`
	EntityID    string         `gorm:"index" json:"entity_id"`
	OldValue    sql.NullString `gorm:"type:jsonb" json:"old_value"`
	NewValue    sql.NullString `gorm:"type:jsonb" json:"new_value"`
	IPAddress   string         `json:"ip_address"`
	UserAgent   string         `json:"user_agent"`
	Status      string         `gorm:"default:'success'" json:"status"`
	ErrorDetail string         `json:"error_detail"`
}

// Notification represents a system notification
type Notification struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	AdminID     uint           `gorm:"not null;index" json:"admin_id"`
	Admin       *Admin         `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
	Title       string         `gorm:"not null" json:"title"`
	Message     string         `gorm:"not null" json:"message"`
	Type        string         `gorm:"not null;default:'info'" json:"type"` // info, warning, error, success
	Priority    string         `gorm:"default:'normal'" json:"priority"` // low, normal, high, urgent
	IsRead      bool           `gorm:"default:false" json:"is_read"`
	ReadAt      *time.Time    `json:"read_at"`
	ActionURL   string         `json:"action_url"`
	ExpiresAt   *time.Time    `json:"expires_at"`
	Metadata    json.RawMessage `gorm:"type:jsonb" json:"metadata"`
}
