package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Admin represents an admin user
type Admin struct {
	ID                uint            `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	DeletedAt         sql.NullTime    `gorm:"index" json:"-"`
	Username          string          `gorm:"uniqueIndex;not null" json:"username"`
	Email             string          `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash      string          `gorm:"not null" json:"-"`
	FirstName         string          `json:"first_name"`
	LastName          string          `json:"last_name"`
	Role              string          `gorm:"not null;default:'admin'" json:"role"` // super_admin, admin, support, analyst, moderator
	Permissions       json.RawMessage `gorm:"type:jsonb" json:"permissions"`
	Status            string          `gorm:"not null;default:'active'" json:"status"` // active, suspended, inactive
	TwoFactorEnabled  bool            `gorm:"default:false" json:"two_factor_enabled"`
	TwoFactorSecret   string          `json:"-"`
	LastLoginAt       *time.Time      `json:"last_login_at"`
	FailedAttempts    int             `gorm:"default:0" json:"failed_attempts"`
	LockedUntil       *time.Time      `json:"locked_until"`
	IPWhitelist       sql.NullString  `gorm:"type:text" json:"-"`
	LastIP            string          `json:"last_ip"`
	EmailVerified     bool            `gorm:"default:false" json:"email_verified"`
	PasswordChangedAt *time.Time      `json:"password_changed_at"`
}

// AdminSession represents an admin login session
type AdminSession struct {
	ID           uint         `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	DeletedAt    sql.NullTime `gorm:"index" json:"-"`
	AdminID      uint         `gorm:"not null;index" json:"admin_id"`
	Token        string       `gorm:"uniqueIndex;not null" json:"token"`
	RefreshToken string       `gorm:"uniqueIndex" json:"refresh_token"`
	ExpiresAt    time.Time    `gorm:"not null" json:"expires_at"`
	IPAddress    string       `json:"ip_address"`
	UserAgent    string       `json:"user_agent"`
	Revoked      bool         `gorm:"default:false" json:"revoked"`
	RevokedAt    *time.Time   `json:"revoked_at"`
}

// AdminActivity represents admin action logs
type AdminActivity struct {
	ID           uint            `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time       `json:"created_at"`
	AdminID      uint            `gorm:"not null;index" json:"admin_id"`
	Admin        *Admin          `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
	Action       string          `gorm:"not null" json:"action"` // create, update, delete, login, logout, etc.
	Resource     string          `json:"resource"`               // User, Transaction, Token, etc.
	ResourceID   string          `json:"resource_id"`
	Details      json.RawMessage `gorm:"type:jsonb" json:"details"`
	IPAddress    string          `json:"ip_address"`
	UserAgent    string          `json:"user_agent"`
	Status       string          `gorm:"default:'success'" json:"status"` // success, failed
	ErrorMessage string          `json:"error_message,omitempty"`
}

// User represents a platform user
type User struct {
	ID                 uint            `gorm:"primarykey" json:"id"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	DeletedAt          sql.NullTime    `gorm:"index" json:"-"`
	Email              string          `gorm:"uniqueIndex;not null" json:"email"`
	Username           string          `gorm:"index" json:"username"`
	Phone              sql.NullString  `gorm:"uniqueIndex" json:"phone"`
	PasswordHash       string          `json:"-"`
	WalletAddress      string          `gorm:"index" json:"wallet_address"`
	Status             string          `gorm:"not null;default:'active'" json:"status"`   // active, pending, suspended, banned
	KYCStatus          string          `gorm:"not null;default:'none'" json:"kyc_status"` // none, pending, level1, level2, level3, rejected
	KYCLevel           int             `gorm:"default:0" json:"kyc_level"`
	KYCSubmittedAt     *time.Time      `json:"kyc_submitted_at"`
	KYCVerifiedAt      *time.Time      `json:"kyc_verified_at"`
	KYCRejectionReason string          `json:"kyc_rejection_reason"`
	TwoFactorEnabled   bool            `gorm:"default:false" json:"two_factor_enabled"`
	ReferralCode       string          `gorm:"uniqueIndex" json:"referral_code"`
	ReferredBy         string          `json:"referred_by"`
	WhiteLabelID       *uint           `gorm:"index" json:"white_label_id"`
	LastLoginAt        *time.Time      `json:"last_login_at"`
	RegistrationIP     string          `json:"registration_ip"`
	RiskScore          int             `gorm:"default:0" json:"risk_score"` // 0-100
	Tags               json.RawMessage `gorm:"type:jsonb" json:"tags"`
	Metadata           json.RawMessage `gorm:"type:jsonb" json:"metadata"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID          uint            `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   sql.NullTime    `gorm:"index" json:"-"`
	UserID      uint            `gorm:"not null;index" json:"user_id"`
	User        *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Hash        string          `gorm:"uniqueIndex;not null" json:"hash"`
	Type        string          `gorm:"not null" json:"type"`  // transfer, swap, stake, unstake, bridge, withdraw, deposit, trade
	Chain       string          `gorm:"not null" json:"chain"` // ethereum, bsc, polygon, etc.
	FromAddress string          `json:"from_address"`
	ToAddress   string          `json:"to_address"`
	Amount      string          `gorm:"type:decimal(36,18)" json:"amount"`
	Token       string          `json:"token"` // ETH, USDT, etc.
	TokenAmount string          `gorm:"type:decimal(36,18)" json:"token_amount"`
	Fee         string          `gorm:"type:decimal(36,18);default:'0'" json:"fee"` // Transaction fee revenue
	Status      string          `gorm:"not null;default:'pending'" json:"status"`   // pending, confirmed, failed
	BlockNumber int64           `json:"block_number"`
	BlockHash   string          `json:"block_hash"`
	GasUsed     string          `json:"gas_used"`
	GasPrice    string          `json:"gas_price"`
	Nonce       int64           `json:"nonce"`
	Timestamp   time.Time       `json:"timestamp"`
	Metadata    json.RawMessage `gorm:"type:jsonb" json:"metadata"`
	Flagged     bool            `gorm:"default:false" json:"flagged"`
	FlagReason  string          `json:"flag_reason"`
}

// Token represents a cryptocurrency token
type Token struct {
	ID              uint            `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       sql.NullTime    `gorm:"index" json:"-"`
	Name            string          `gorm:"not null" json:"name"`
	Symbol          string          `gorm:"not null;index" json:"symbol"`
	ContractAddress string          `gorm:"index" json:"contract_address"`
	Chain           string          `gorm:"not null" json:"chain"`
	Decimals        int             `gorm:"not null;default:18" json:"decimals"`
	TotalSupply     string          `gorm:"type:decimal(36,0)" json:"total_supply"`
	LogoURL         string          `json:"logo_url"`
	Website         string          `json:"website"`
	Description     string          `json:"description"`
	Price           string          `gorm:"type:decimal(36,8)" json:"price"`
	MarketCap       string          `gorm:"type:decimal(36,2)" json:"market_cap"`
	Volume24h       string          `gorm:"type:decimal(36,2)" json:"volume_24h"`
	PriceChange24h  string          `gorm:"type:decimal(10,4)" json:"price_change_24h"`
	IsActive        bool            `gorm:"default:true" json:"is_active"`
	IsVerified      bool            `gorm:"default:false" json:"is_verified"`
	ListingFee      string          `gorm:"type:decimal(36,8)" json:"listing_fee"`
	ListedBy        uint            `json:"listed_by"`
	ListedAt        *time.Time      `json:"listed_at"`
	Metadata        json.RawMessage `gorm:"type:jsonb" json:"metadata"`
}

// KYCApplication represents a KYC submission
type KYCApplication struct {
	ID              uint            `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       sql.NullTime    `gorm:"index" json:"-"`
	UserID          uint            `gorm:"not null;uniqueIndex" json:"user_id"`
	User            *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Level           int             `gorm:"not null" json:"level"`                    // 1, 2, 3
	Status          string          `gorm:"not null;default:'pending'" json:"status"` // pending, approved, rejected
	SubmittedAt     time.Time       `json:"submitted_at"`
	ReviewedAt      *time.Time      `json:"reviewed_at"`
	ReviewedBy      *uint           `json:"reviewed_by"`
	Reviewer        *Admin          `gorm:"foreignKey:ReviewedBy" json:"reviewer,omitempty"`
	RejectionReason string          `json:"rejection_reason"`
	Documents       json.RawMessage `gorm:"type:jsonb" json:"documents"`
	IPAddress       string          `json:"ip_address"`
	Notes           string          `json:"notes"`
}

// Withdrawal represents a withdrawal request
type Withdrawal struct {
	ID              uint         `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
	DeletedAt       sql.NullTime `gorm:"index" json:"-"`
	UserID          uint         `gorm:"not null;index" json:"user_id"`
	User            *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Amount          string       `gorm:"type:decimal(36,18);not null" json:"amount"`
	Token           string       `gorm:"not null" json:"token"`
	Chain           string       `gorm:"not null" json:"chain"`
	ToAddress       string       `gorm:"not null" json:"to_address"`
	Status          string       `gorm:"not null;default:'pending'" json:"status"` // pending, approved, rejected, processing, completed, failed
	ApprovedAt      *time.Time   `json:"approved_at"`
	ApprovedBy      *uint        `json:"approved_by"`
	Approver        *Admin       `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
	RejectedAt      *time.Time   `json:"rejected_at"`
	RejectedBy      *uint        `json:"rejected_by"`
	RejectionReason string       `json:"rejection_reason"`
	ProcessedAt     *time.Time   `json:"processed_at"`
	TxHash          string       `json:"tx_hash"`
	Fee             string       `gorm:"type:decimal(36,18)" json:"fee"`
	IPAddress       string       `json:"ip_address"`
	Notes           string       `json:"notes"`
}

// WalletBalance represents a user's wallet balance
type WalletBalance struct {
	ID        uint         `gorm:"primarykey" json:"id"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	DeletedAt sql.NullTime `gorm:"index" json:"-"`
	UserID    uint         `gorm:"not null;index" json:"user_id"`
	User      *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Token     string       `gorm:"not null;index" json:"token"`
	Chain     string       `gorm:"not null" json:"chain"`
	Balance   string       `gorm:"type:decimal(36,18);not null" json:"balance"`
	Available string       `gorm:"type:decimal(36,18);not null" json:"available"`
	Locked    string       `gorm:"type:decimal(36,18);default:'0'" json:"locked"`
	Reserved  string       `gorm:"type:decimal(36,18);default:'0'" json:"reserved"`
}

// TransactionLog represents a blockchain transaction log
type TransactionLog struct {
	ID        uint            `gorm:"primarykey" json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	UserID    uint            `gorm:"not null;index" json:"user_id"`
	User      *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Type      string          `gorm:"not null;index" json:"type"` // deposit, withdrawal, transfer, swap, refund
	Token     string          `gorm:"not null" json:"token"`
	Chain     string          `gorm:"not null" json:"chain"`
	Amount    string          `gorm:"type:decimal(36,18);not null" json:"amount"`
	Fee       string          `gorm:"type:decimal(36,18)" json:"fee"`
	TxHash    string          `gorm:"index" json:"tx_hash"`
	Status    string          `gorm:"not null" json:"status"` // pending, broadcast, confirmed, failed
	FromAddr  string          `json:"from_addr"`
	ToAddr    string          `json:"to_addr"`
	Metadata  json.RawMessage `gorm:"type:jsonb" json:"metadata"`
}

// WhiteLabel represents a white label client
type WhiteLabel struct {
	ID             uint            `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      sql.NullTime    `gorm:"index" json:"-"`
	Name           string          `gorm:"not null" json:"name"`
	Slug           string          `gorm:"uniqueIndex;not null" json:"slug"`
	Domain         string          `gorm:"uniqueIndex" json:"domain"`
	LogoURL        string          `json:"logo_url"`
	FaviconURL     string          `json:"favicon_url"`
	PrimaryColor   string          `json:"primary_color"`
	SecondaryColor string          `json:"secondary_color"`
	Status         string          `gorm:"not null;default:'active'" json:"status"` // active, suspended, pending
	ContactEmail   string          `json:"contact_email"`
	ContactPhone   string          `json:"contact_phone"`
	Address        string          `json:"address"`
	Description    string          `json:"description"`
	CustomCSS      sql.NullString  `gorm:"type:text" json:"custom_css"`
	CustomJS       sql.NullString  `gorm:"type:text" json:"custom_js"`
	Features       json.RawMessage `gorm:"type:jsonb" json:"features"`
	FeeStructure   json.RawMessage `gorm:"type:jsonb" json:"fee_structure"`
	ApprovedAt     *time.Time      `json:"approved_at"`
	ApprovedBy     *uint           `json:"approved_by"`
	ExpiresAt      *time.Time      `json:"expires_at"`
}

// SystemConfig represents system configuration
type SystemConfig struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Key         string    `gorm:"uniqueIndex;not null" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Type        string    `gorm:"not null;default:'string'" json:"type"` // string, int, bool, json
	Description string    `json:"description"`
	UpdatedBy   *uint     `json:"updated_by"`
	Category    string    `gorm:"index" json:"category"` // fees, limits, features, etc.
	IsPublic    bool      `gorm:"default:false" json:"is_public"`
}

// APIKey represents an API key for external access
type APIKey struct {
	ID          uint            `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   sql.NullTime    `gorm:"index" json:"-"`
	Name        string          `gorm:"not null" json:"name"`
	Key         string          `gorm:"uniqueIndex;not null" json:"key"`
	KeyHash     string          `gorm:"not null;index" json:"-"`
	AdminID     uint            `gorm:"not null;index" json:"admin_id"`
	Admin       *Admin          `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
	Permissions json.RawMessage `gorm:"type:jsonb" json:"permissions"`
	RateLimit   int             `gorm:"default:1000" json:"rate_limit"` // requests per hour
	LastUsedAt  *time.Time      `json:"last_used_at"`
	ExpiresAt   *time.Time      `json:"expires_at"`
	Status      string          `gorm:"not null;default:'active'" json:"status"` // active, suspended, expired
	Scopes      json.RawMessage `gorm:"type:jsonb" json:"scopes"`
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
	ID        uint            `gorm:"primarykey" json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	AdminID   uint            `gorm:"not null;index" json:"admin_id"`
	Admin     *Admin          `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
	Title     string          `gorm:"not null" json:"title"`
	Message   string          `gorm:"not null" json:"message"`
	Type      string          `gorm:"not null;default:'info'" json:"type"` // info, warning, error, success
	Priority  string          `gorm:"default:'normal'" json:"priority"`    // low, normal, high, urgent
	IsRead    bool            `gorm:"default:false" json:"is_read"`
	ReadAt    *time.Time      `json:"read_at"`
	ActionURL string          `json:"action_url"`
	ExpiresAt *time.Time      `json:"expires_at"`
	Metadata  json.RawMessage `gorm:"type:jsonb" json:"metadata"`
}

// TwoFactorAuth represents two-factor authentication settings
type TwoFactorAuth struct {
	ID                uint       `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	UserID            uint       `gorm:"not null;index" json:"user_id"`
	UserType          string     `gorm:"not null;index" json:"user_type"` // admin, user, super_admin
	Secret            string     `gorm:"not null" json:"-"`
	Enabled           bool       `gorm:"default:false" json:"enabled"`
	Methods           string     `gorm:"type:jsonb" json:"methods"` // ["totp", "sms", "email", "backup"]
	BackupCodesHashed string     `gorm:"type:text" json:"-"`
	UsedBackupCodes   string     `gorm:"type:jsonb" json:"-"`
	EnabledAt         *time.Time `json:"enabled_at"`
	DisabledAt        *time.Time `json:"disabled_at"`
	LastVerifiedAt    *int64     `json:"last_verified_at"`
	TrustedDevices    int        `gorm:"default:0" json:"trusted_devices"`
}

// TwoFactorAttempt represents 2FA verification attempts
type TwoFactorAttempt struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	UserType    string    `gorm:"not null;index" json:"user_type"`
	IPAddress   string    `json:"ip_address"`
	AttemptType string    `json:"attempt_type"` // verification_failed, backup_code_used, etc.
	Success     bool      `gorm:"default:false" json:"success"`
	Timestamp   time.Time `json:"timestamp"`
}

// IPWhitelist represents IP whitelist entries
type IPWhitelist struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	UserID      uint       `gorm:"not null;index" json:"user_id"`
	UserType    string     `gorm:"not null;index" json:"user_type"` // admin, user, api_key
	IPAddress   string     `gorm:"not null" json:"ip_address"`      // CIDR notation supported
	Description string     `json:"description"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedBy   uint       `json:"created_by"`
}

// RateLimitRule represents rate limiting rules
type RateLimitRule struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Endpoint    string    `gorm:"not null" json:"endpoint"`      // /api/v1/users, /api/v1/login, etc.
	Method      string    `gorm:"not null" json:"method"`        // GET, POST, PUT, DELETE, ALL
	Limit       int       `gorm:"not null" json:"limit"`         // requests per window
	Window      int       `gorm:"not null" json:"window"`        // window in seconds
	Scope       string    `gorm:"default:'global'" json:"scope"` // global, per_user, per_ip
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	Description string    `json:"description"`
}

// FraudDetection represents fraud detection rules
type FraudDetection struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Name         string    `gorm:"uniqueIndex;not null" json:"name"`
	Type         string    `gorm:"not null" json:"type"`        // velocity, amount, geographic, pattern
	Condition    string    `gorm:"type:jsonb" json:"condition"` // JSON condition
	Action       string    `gorm:"not null" json:"action"`      // block, alert, review, limit
	Threshold    float64   `json:"threshold"`
	Severity     string    `gorm:"default:'medium'" json:"severity"` // low, medium, high, critical
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	Notification bool      `gorm:"default:false" json:"notification"`
	Description  string    `json:"description"`
}

// FraudAlert represents fraud alerts
type FraudAlert struct {
	ID          uint            `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time       `json:"created_at"`
	UserID      uint            `gorm:"not null;index" json:"user_id"`
	User        *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	RuleID      uint            `gorm:"index" json:"rule_id"`
	Rule        *FraudDetection `gorm:"foreignKey:RuleID" json:"rule,omitempty"`
	Type        string          `gorm:"not null" json:"type"`
	Description string          `json:"description"`
	Status      string          `gorm:"default:'pending'" json:"status"` // pending, reviewed, resolved, false_positive
	ReviewedBy  *uint           `json:"reviewed_by"`
	ReviewedAt  *time.Time      `json:"reviewed_at"`
	Resolution  string          `json:"resolution"`
	Metadata    json.RawMessage `gorm:"type:jsonb" json:"metadata"`
}

// ComplianceReport represents compliance reports
type ComplianceReport struct {
	ID           uint            `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time       `json:"created_at"`
	GeneratedAt  time.Time       `json:"generated_at"`
	Type         string          `gorm:"not null" json:"type"` // aml, gdpr, tax, sar
	PeriodStart  time.Time       `json:"period_start"`
	PeriodEnd    time.Time       `json:"period_end"`
	Status       string          `gorm:"default:'pending'" json:"status"` // pending, generating, completed, failed
	FilePath     string          `json:"file_path"`
	FileSize     int64           `json:"file_size"`
	GeneratedBy  uint            `json:"generated_by"`
	ErrorMessage string          `json:"error_message"`
	Metadata     json.RawMessage `gorm:"type:jsonb" json:"metadata"`
}

// SupportTicket represents support tickets
type SupportTicket struct {
	ID                 uint       `gorm:"primarykey" json:"id"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	TicketID           string     `gorm:"uniqueIndex" json:"ticket_id"`
	UserID             uint       `gorm:"not null;index" json:"user_id"`
	User               *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Category           string     `gorm:"not null" json:"category"`         // technical, billing, kyc, account, etc.
	Priority           string     `gorm:"default:'medium'" json:"priority"` // low, medium, high, urgent
	Status             string     `gorm:"default:'open'" json:"status"`     // open, in_progress, waiting, resolved, closed
	Subject            string     `gorm:"not null" json:"subject"`
	Description        string     `gorm:"type:text" json:"description"`
	AssignedTo         *uint      `gorm:"index" json:"assigned_to"`
	AssignedAdmin      *Admin     `gorm:"foreignKey:AssignedTo" json:"assigned_admin,omitempty"`
	ResolvedAt         *time.Time `json:"resolved_at"`
	ClosedAt           *time.Time `json:"closed_at"`
	Rating             *int       `json:"rating"`
	Feedback           string     `json:"feedback"`
	SLAFirstResponseBy *time.Time `json:"sla_first_response_by"`
	SLAResolutionBy    *time.Time `json:"sla_resolution_by"`
	FirstResponseAt    *time.Time `json:"first_response_at"`
}

// SupportTicketMessage represents messages in support tickets
type SupportTicketMessage struct {
	ID          uint            `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time       `json:"created_at"`
	TicketID    uint            `gorm:"not null;index" json:"ticket_id"`
	Ticket      *SupportTicket  `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
	SenderID    uint            `gorm:"not null" json:"sender_id"`
	SenderType  string          `gorm:"not null" json:"sender_type"` // user, admin, system
	SenderName  string          `json:"sender_name,omitempty"`
	Sender      interface{}     `json:"sender,omitempty"` // User or Admin
	Message     string          `gorm:"type:text" json:"message"`
	IsInternal  bool            `gorm:"default:false" json:"is_internal"`
	Attachments json.RawMessage `gorm:"type:jsonb" json:"attachments"`
}

// KnowledgeBaseCategory represents knowledge base categories
type KnowledgeBaseCategory struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"not null" json:"name"`
	Slug        string    `gorm:"uniqueIndex" json:"slug"`
	Description string    `json:"description"`
	ParentID    *uint     `gorm:"index" json:"parent_id"`
	Order       int       `gorm:"default:0" json:"order"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
}

// KnowledgeBaseArticle represents knowledge base articles
type KnowledgeBaseArticle struct {
	ID          uint                   `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CategoryID  uint                   `gorm:"not null;index" json:"category_id"`
	Category    *KnowledgeBaseCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Title       string                 `gorm:"not null" json:"title"`
	Slug        string                 `gorm:"uniqueIndex" json:"slug"`
	Content     string                 `gorm:"type:text" json:"content"`
	Summary     string                 `json:"summary"`
	Tags        json.RawMessage        `gorm:"type:jsonb" json:"tags"`
	ViewCount   int                    `gorm:"default:0" json:"view_count"`
	IsPublished bool                   `gorm:"default:false" json:"is_published"`
	IsFeatured  bool                   `gorm:"default:false" json:"is_featured"`
	Order       int                    `gorm:"default:0" json:"order"`
	AuthorID    uint                   `json:"author_id"`
}

// IntegrationConfig represents external integrations
type IntegrationConfig struct {
	ID           uint            `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Type         string          `gorm:"not null" json:"type"` // slack, pagerduty, datadog, cloudflare
	Name         string          `gorm:"not null" json:"name"`
	Config       json.RawMessage `gorm:"type:jsonb" json:"config"`
	IsActive     bool            `gorm:"default:true" json:"is_active"`
	LastSyncAt   *time.Time      `json:"last_sync_at"`
	SyncStatus   string          `json:"sync_status"`
	ErrorMessage string          `json:"error_message"`
}

// ScheduledReport represents scheduled report configurations
type ScheduledReport struct {
	ID           uint            `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Name         string          `gorm:"not null" json:"name"`
	Type         string          `gorm:"not null" json:"type"`     // pdf, excel
	Schedule     string          `gorm:"not null" json:"schedule"` // cron expression
	ReportConfig json.RawMessage `gorm:"type:jsonb" json:"report_config"`
	Recipients   json.RawMessage `gorm:"type:jsonb" json:"recipients"`
	IsActive     bool            `gorm:"default:true" json:"is_active"`
	LastRunAt    *time.Time      `json:"last_run_at"`
	NextRunAt    *time.Time      `json:"next_run_at"`
	CreatedBy    uint            `json:"created_by"`
}

// ==================== ADDITIONAL MODELS ====================

// Broker represents an external broker partner
type Broker struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Name           string    `gorm:"not null" json:"name"`
	Email          string    `gorm:"uniqueIndex" json:"email"`
	Phone          string    `json:"phone"`
	Company        string    `json:"company"`
	Address        string    `json:"address"`
	Commission     float64   `json:"commission"`
	MinTradeAmount float64   `json:"min_trade_amount"`
	MaxTradeAmount float64   `json:"max_trade_amount"`
	AllowedChains  []string  `gorm:"type:text[]" json:"allowed_chains"`
	AllowedTokens  []string  `gorm:"type:text[]" json:"allowed_tokens"`
	KYCRequired    bool      `gorm:"default:false" json:"kyc_required"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	Status         string    `gorm:"not null;default:'pending'" json:"status"`
	CreatedBy      uint      `json:"created_by"`
}

// BrokerClient represents a client associated with a broker
type BrokerClient struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	BrokerID  uint      `gorm:"index" json:"broker_id"`
	ClientID  uint      `json:"client_id"`
	Status    string    `gorm:"not null;default:'active'" json:"status"`
}

// FeatureFlag represents a feature toggle
type FeatureFlag struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Name              string    `gorm:"uniqueIndex;not null" json:"name"`
	Description       string    `json:"description"`
	IsEnabled         bool      `gorm:"default:false" json:"is_enabled"`
	RolloutPercentage int       `gorm:"default:0" json:"rollout_percentage"`
	CreatedBy         uint      `json:"created_by"`
}

// FeeStructure represents a configurable fee structure
type FeeStructure struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Name            string    `gorm:"not null" json:"name"`
	FeeType         string    `gorm:"not null" json:"fee_type"`
	Chain           string    `json:"chain"`
	Token           string    `json:"token"`
	FeePercent      float64   `json:"fee_percent"`
	FeeFixed        float64   `json:"fee_fixed"`
	MinFee          float64   `json:"min_fee"`
	MaxFee          float64   `json:"max_fee"`
	Tier            string    `json:"tier"`
	VolumeThreshold float64   `json:"volume_threshold"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedBy       uint      `json:"created_by"`
}

// GDPRRequest represents a GDPR data request
type GDPRRequest struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	Email       string     `json:"email"`
	RequestType string     `gorm:"not null" json:"request_type"`
	Reason      string     `json:"reason"`
	Status      string     `gorm:"not null;default:'pending'" json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	CompletedAt *time.Time `json:"completed_at"`
	ProcessedBy *uint      `json:"processed_by"`
}

// InstitutionalClient represents an institutional client account
type InstitutionalClient struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Name           string    `gorm:"not null" json:"name"`
	Email          string    `gorm:"uniqueIndex" json:"email"`
	Phone          string    `json:"phone"`
	Company        string    `gorm:"not null" json:"company"`
	ClientType     string    `gorm:"not null" json:"client_type"`
	Address        string    `json:"address"`
	RegistrationNo string    `json:"registration_no"`
	TaxID          string    `json:"tax_id"`
	DailyLimit     float64   `json:"daily_limit"`
	MonthlyLimit   float64   `json:"monthly_limit"`
	YearlyLimit    float64   `json:"yearly_limit"`
	MinTradeAmount float64   `json:"min_trade_amount"`
	MaxTradeAmount float64   `json:"max_trade_amount"`
	AllowedChains  []string  `gorm:"type:text[]" json:"allowed_chains"`
	AllowedTokens  []string  `gorm:"type:text[]" json:"allowed_tokens"`
	RequiresKYC    bool      `gorm:"default:true" json:"requires_kyc"`
	HasAPIAccess   bool      `gorm:"default:false" json:"has_api_access"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	Status         string    `gorm:"not null;default:'pending'" json:"status"`
	CreatedBy      uint      `json:"created_by"`
}

// KYCRecord represents a KYC verification record
type KYCRecord struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Level     int       `json:"level"`
	Status    string    `gorm:"not null;default:'pending'" json:"status"`
}

// MultisigWallet represents a multi-signature wallet
type MultisigWallet struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"not null" json:"name"`
	Chain       string    `gorm:"not null" json:"chain"`
	Address     string    `gorm:"uniqueIndex" json:"address"`
	Threshold   int       `gorm:"not null" json:"threshold"`
	Balance     float64   `json:"balance"`
	Currency    string    `json:"currency"`
	Description string    `json:"description"`
	Status      string    `gorm:"not null;default:'active'" json:"status"`
	CreatedBy   uint      `json:"created_by"`
}

// MultisigSigner represents a signer of a multisig wallet
type MultisigSigner struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	WalletID  uint      `gorm:"index;not null" json:"wallet_id"`
	Address   string    `gorm:"not null" json:"address"`
}

// MultisigTransaction represents a pending multisig transaction
type MultisigTransaction struct {
	ID            uint       `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	WalletID      uint       `gorm:"index;not null" json:"wallet_id"`
	UserID        uint       `json:"user_id"`
	Type          string     `gorm:"not null" json:"type"`
	Token         string     `json:"token"`
	Chain         string     `json:"chain"`
	FromAddress   string     `json:"from_address"`
	ToAddress     string     `json:"to_address"`
	Amount        float64    `json:"amount"`
	Fee           float64    `json:"fee"`
	Status        string     `gorm:"not null;default:'pending'" json:"status"`
	ApprovalCount int        `gorm:"default:0" json:"approval_count"`
	ApprovedAt    *time.Time `json:"approved_at"`
	RejectedAt    *time.Time `json:"rejected_at"`
	RejectReason  string     `json:"reject_reason"`
}

// MultisigApproval represents an approval for a multisig transaction
type MultisigApproval struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	TransactionID uint      `gorm:"index;not null" json:"transaction_id"`
	ApprovedBy    uint      `gorm:"not null" json:"approved_by"`
	ApprovedAt    time.Time `json:"approved_at"`
}

// NFT represents a non-fungible token listing
type NFT struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Name              string    `gorm:"not null" json:"name"`
	Description       string    `json:"description"`
	CollectionAddress string    `json:"collection_address"`
	TokenID           string    `json:"token_id"`
	Chain             string    `gorm:"not null" json:"chain"`
	TokenType         string    `json:"token_type"`
	ContractType      string    `json:"contract_type"`
	MetadataURL       string    `json:"metadata_url"`
	ImageURL          string    `json:"image_url"`
	ExternalURL       string    `json:"external_url"`
	Creator           string    `json:"creator"`
	Owner             string    `json:"owner"`
	Royalty           float64   `json:"royalty"`
	Attributes        string    `json:"attributes"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	Status            string    `gorm:"not null;default:'active'" json:"status"`
	CreatedBy         uint      `json:"created_by"`
}

// TradingPair represents a tradable token pair
type TradingPair struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	PairName          string    `gorm:"uniqueIndex;not null" json:"pair_name"`
	BaseToken         string    `gorm:"not null" json:"base_token"`
	QuoteToken        string    `gorm:"not null" json:"quote_token"`
	Chain             string    `gorm:"not null" json:"chain"`
	MinTradeAmount    float64   `json:"min_trade_amount"`
	MaxTradeAmount    float64   `json:"max_trade_amount"`
	MakerFee          float64   `json:"maker_fee"`
	TakerFee          float64   `json:"taker_fee"`
	PricePrecision    int       `json:"price_precision"`
	QuantityPrecision int       `json:"quantity_precision"`
	MinPrice          float64   `json:"min_price"`
	MaxPrice          float64   `json:"max_price"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	Status            string    `gorm:"not null;default:'active'" json:"status"`
	CreatedBy         uint      `json:"created_by"`
}
