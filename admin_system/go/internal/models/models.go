// Models - Data models for Admin Backend
package models

import (
	"time"

	"github.com/google/uuid"
)

type AdminUser struct {
	ID               uuid.UUID  `json:"id"`
	Username         string     `json:"username"`
	Email            string     `json:"email"`
	PasswordHash     string     `json:"-"`
	Role             string     `json:"role"`
	TwoFactorSecret  string     `json:"-"`
	TwoFactorEnabled bool       `json:"two_factor_enabled"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LastLogin        *time.Time `json:"last_login"`
}

type Session struct {
	ID        uuid.UUID `json:"id"`
	AdminID   uuid.UUID `json:"admin_id"`
	TokenHash string    `json:"-"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type IPWhitelist struct {
	ID          uuid.UUID `json:"id"`
	IPAddress   string    `json:"ip_address"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

type FeatureFlag struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	IsEnabled         bool      `json:"is_enabled"`
	RolloutPercentage int       `json:"rollout_percentage"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	UpdatedBy         uuid.UUID `json:"updated_by"`
}

type AuditLog struct {
	ID           uuid.UUID              `json:"id"`
	AdminID      *uuid.UUID             `json:"admin_id"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Details      map[string]interface{} `json:"details"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	CreatedAt    time.Time              `json:"created_at"`
}

type Notification struct {
	ID               uuid.UUID `json:"id"`
	AdminID          uuid.UUID `json:"admin_id"`
	Title            string    `json:"title"`
	Message          string    `json:"message"`
	NotificationType string    `json:"notification_type"`
	IsRead           bool      `json:"is_read"`
	CreatedAt        time.Time `json:"created_at"`
}

type User struct {
	ID               uuid.UUID              `json:"id"`
	Email            string                 `json:"email"`
	Username         string                 `json:"username"`
	WalletAddress    string                 `json:"wallet_address"`
	KYCStatus        string                 `json:"kyc_status"`
	Status           string                 `json:"status"`
	TwoFactorEnabled bool                   `json:"two_factor_enabled"`
	IPAddress        string                 `json:"ip_address"`
	Country          string                 `json:"country"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	LastLogin        *time.Time             `json:"last_login"`
	Balance          map[string]interface{} `json:"balance"`
}

type KYCRequest struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	DocType      string     `json:"doc_type"`
	Status       string     `json:"status"`
	DocumentURL  string     `json:"document_url"`
	SubmittedAt  time.Time  `json:"submitted_at"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	ReviewedBy   *uuid.UUID `json:"reviewed_by"`
	RejectReason string     `json:"reject_reason"`
}

type Transaction struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Type        string    `json:"type"`
	Amount      string    `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	TXHash      string    `json:"tx_hash"`
	Fee         string    `json:"fee"`
	ChainID     int       `json:"chain_id"`
	Timestamp   time.Time `json:"timestamp"`
}

type Withdrawal struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Amount      string     `json:"amount"`
	Currency    string     `json:"currency"`
	Status      string     `json:"status"`
	Address     string     `json:"address"`
	TXHash      string     `json:"tx_hash"`
	ApprovedBy  *uuid.UUID `json:"approved_by"`
	ProcessedAt *time.Time `json:"processed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Token struct {
	ID              uuid.UUID `json:"id"`
	Symbol          string    `json:"symbol"`
	Name            string    `json:"name"`
	ContractAddress string    `json:"contract_address"`
	Decimals        int       `json:"decimals"`
	IsActive        bool      `json:"is_active"`
	IsVerified      bool      `json:"is_verified"`
	TotalSupply     string    `json:"total_supply"`
	ChainID         int       `json:"chain_id"`
	CreatedAt       time.Time `json:"created_at"`
}

type TradingPair struct {
	ID           uuid.UUID `json:"id"`
	BaseTokenID  uuid.UUID `json:"base_token_id"`
	QuoteTokenID uuid.UUID `json:"quote_token_id"`
	PairName     string    `json:"pair_name"`
	Price        string    `json:"price"`
	Volume24h    string    `json:"volume_24h"`
	Liquidity    string    `json:"liquidity"`
	Status       string    `json:"status"`
	ChainID      int       `json:"chain_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type FeeStructure struct {
	ID         uuid.UUID `json:"id"`
	FeeType    string    `json:"fee_type"`
	Asset      string    `json:"asset"`
	FeePercent string    `json:"fee_percent"`
	FeeFixed   string    `json:"fee_fixed"`
	MinFee     string    `json:"min_fee"`
	MaxFee     string    `json:"max_fee"`
	Tier       string    `json:"tier"`
	IsActive   bool      `json:"is_active"`
	ChainID    int       `json:"chain_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Blockchain struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Symbol          string    `json:"symbol"`
	ChainID         int       `json:"chain_id"`
	IsEVM           bool      `json:"is_evm"`
	RPCURL          string    `json:"rpc_url"`
	ExplorerURL     string    `json:"explorer_url"`
	NativeToken     string    `json:"native_token"`
	Decimals        int       `json:"decimals"`
	IsActive        bool      `json:"is_active"`
	AvgGasPriceGwei string    `json:"avg_gas_price_gwei"`
	CreatedAt       time.Time `json:"created_at"`
}

type Webhook struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Secret    string    `json:"-"`
	Events    []string  `json:"events"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy uuid.UUID `json:"created_by"`
}

type WhiteLabel struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Domain         string    `json:"domain"`
	LogoURL        string    `json:"logo_url"`
	PrimaryColor   string    `json:"primary_color"`
	SecondaryColor string    `json:"secondary_color"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

type Ticket struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	TicketType  string     `json:"ticket_type"`
	Priority    string     `json:"priority"`
	Status      string     `json:"status"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	AssignedTo  *uuid.UUID `json:"assigned_to"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
}

type TicketMessage struct {
	ID         uuid.UUID `json:"id"`
	TicketID   uuid.UUID `json:"ticket_id"`
	Message    string    `json:"message"`
	IsInternal bool      `json:"is_internal"`
	CreatedBy  uuid.UUID `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type Backup struct {
	ID          uuid.UUID  `json:"id"`
	BackupType  string     `json:"backup_type"`
	FilePath    string     `json:"file_path"`
	FileSize    int64      `json:"file_size"`
	Status      string     `json:"status"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type ApprovalWorkflow struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	WorkflowType      string    `json:"workflow_type"`
	ThresholdAmount   string    `json:"threshold_amount"`
	RequiredApprovals int       `json:"required_approvals"`
	Approvers         []string  `json:"approvers"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         uuid.UUID `json:"created_by"`
}

type ApprovalRequest struct {
	ID           uuid.UUID              `json:"id"`
	WorkflowID   *uuid.UUID             `json:"workflow_id"`
	RequestType  string                 `json:"request_type"`
	ResourceID   string                 `json:"resource_id"`
	RequesterID  uuid.UUID              `json:"requester_id"`
	Status       string                 `json:"status"`
	Details      map[string]interface{} `json:"details"`
	ApprovedBy   *uuid.UUID             `json:"approved_by"`
	ApprovedAt   *time.Time             `json:"approved_at"`
	RejectReason string                 `json:"reject_reason"`
	CreatedAt    time.Time              `json:"created_at"`
}

type PlatformStats struct {
	TotalUsers        int64   `json:"total_users"`
	ActiveUsers       int64   `json:"active_users"`
	TotalVolume       float64 `json:"total_volume"`
	TotalTransactions int64   `json:"total_transactions"`
	TotalFees         float64 `json:"total_fees"`
	ActiveBots        int     `json:"active_bots"`
	TotalBots         int     `json:"total_bots"`
}
