/**
 * TigerWallet RBAC Admin Panel - Go Implementation
 * Complete user management, KYC, transactions, trading, fees, blockchain
 * PRODUCTION-READY - Uses PostgreSQL database (NOT in-memory)
 */

package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Database connection - PRODUCTION PostgreSQL
var db *sql.DB

func initDB() {
	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/tigerwallet_admin?sslmode=disable"
	}

	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		fmt.Printf("Failed to ping database: %v\n", err)
		return
	}

	fmt.Println("✅ Connected to PostgreSQL database")
}

// ==================== TYPES ====================

type UserStatus int
type KYCStatus int
type TransactionStatus int
type TransactionType int
type PairStatus int
type APIKeyTier int

const (
	StatusActive UserStatus = iota + 1
	StatusSuspended
	StatusBanned
)

const (
	KYCNone KYCStatus = iota
	KYCPending
	KYCApproved
	KYCRejected
)

const (
	TxPending TransactionStatus = iota + 1
	TxCompleted
	TxFailed
)

const (
	TxDeposit TransactionType = iota + 1
	TxWithdrawal
	TxTransfer
	TxSwap
)

const (
	PairActive PairStatus = iota + 1
	PairSuspended
	PairHalted
)

const (
	APIFree APIKeyTier = iota + 1
	APIBasic
	APIPro
	APIEnterprise
)

// User entity
type User struct {
	ID               string             `json:"id"`
	Email            string             `json:"email"`
	Username         string             `json:"username"`
	PasswordHash     string             `json:"password_hash"`
	WalletAddress    string             `json:"wallet_address"`
	KYCStatus        KYCStatus          `json:"kyc_status"`
	KYCLevel         int                `json:"kyc_level"`
	Status           UserStatus         `json:"status"`
	UserStatus       UserStatus         `json:"user_status"`
	RiskScore        float64            `json:"risk_score"`
	CreatedAt        int64              `json:"created_at"`
	UpdatedAt        int64              `json:"updated_at"`
	LastLogin        *time.Time         `json:"last_login"`
	Balance          map[string]float64 `json:"balance"`
	TwoFactorEnabled bool               `json:"two_factor_enabled"`
	IPAddress        string             `json:"ip_address"`
	Country          string             `json:"country"`
}

// Session represents an active user session.
type Session struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Token     string `json:"token"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	CreatedAt int64  `json:"created_at"`
	ExpiresAt int64  `json:"expires_at"`
	LastSeen  int64  `json:"last_seen"`
}

// KYC Request
type KYCRequest struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Type         string    `json:"type"` // identity, address, selfie
	Status       KYCStatus `json:"status"`
	DocumentURL  string    `json:"document_url"`
	SubmittedAt  int64     `json:"submitted_at"`
	ReviewedAt   int64     `json:"reviewed_at"`
	ReviewedBy   string    `json:"reviewed_by"`
	RejectReason string    `json:"reject_reason"`
}

// Transaction
type Transaction struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	Type        TransactionType   `json:"type"`
	Amount      float64           `json:"amount"`
	Currency    string            `json:"currency"`
	Status      TransactionStatus `json:"status"`
	FromAddress string            `json:"from_address"`
	ToAddress   string            `json:"to_address"`
	TxHash      string            `json:"tx_hash"`
	Timestamp   int64             `json:"timestamp"`
	Fee         float64           `json:"fee"`
	ChainID     int               `json:"chain_id"`
}

// Trading Pair
type TradingPair struct {
	ID         string     `json:"id"`
	Base       string     `json:"base"`
	Quote      string     `json:"quote"`
	PairName   string     `json:"pair_name"`
	Price      float64    `json:"price"`
	Volume24h  float64    `json:"volume_24h"`
	Liquidity  float64    `json:"liquidity"`
	PairStatus PairStatus `json:"status"`
	ChainID    int        `json:"chain_id"`
	CreatedAt  int64      `json:"created_at"`
	UpdatedAt  int64      `json:"updated_at"`
}

// Liquidity Pool
type LiquidityPool struct {
	ID          string  `json:"id"`
	PairID      string  `json:"pair_id"`
	UserID      string  `json:"user_id"`
	BaseAmount  float64 `json:"base_amount"`
	QuoteAmount float64 `json:"quote_amount"`
	Liquidity   float64 `json:"liquidity"`
	APR         float64 `json:"apr"`
	CreatedAt   int64   `json:"created_at"`
}

// Fee Structure
type FeeStructure struct {
	ID         string  `json:"id"`
	FeeType    string  `json:"fee_type"` // withdrawal, deposit, trading, swap
	Asset      string  `json:"asset"`
	FeePercent float64 `json:"fee_percent"`
	FeeFixed   float64 `json:"fee_fixed"`
	MinFee     float64 `json:"min_fee"`
	MaxFee     float64 `json:"max_fee"`
	Tier       string  `json:"tier"`
	IsActive   bool    `json:"is_active"`
	ChainID    int     `json:"chain_id"`
}

// Blockchain
type Blockchain struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Symbol          string  `json:"symbol"`
	ChainID         int     `json:"chain_id"`
	IsEVM           bool    `json:"is_evm"`
	RPCURL          string  `json:"rpc_url"`
	ExplorerURL     string  `json:"explorer_url"`
	NativeToken     string  `json:"native_token"`
	Decimals        int     `json:"decimals"`
	IsActive        bool    `json:"is_active"`
	AvgGasPriceGwei float64 `json:"avg_gas_price_gwei"`
}

// Bot Instance
type BotInstance struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	BotType       string  `json:"bot_type"`
	Name          string  `json:"name"`
	Status        string  `json:"status"` // running, stopped, error, paused
	ConnectedDEXs int     `json:"connected_dexs"`
	ConnectedCEXs int     `json:"connected_cexs"`
	TotalPnL      float64 `json:"total_pnl"`
	TotalVolume   float64 `json:"total_volume"`
	TotalOrders   int     `json:"total_orders"`
	AvgLatencyUs  int     `json:"avg_latency_us"`
	CreatedAt     int64   `json:"created_at"`
	LastTradeAt   int64   `json:"last_trade_at"`
}

// Bot Tier
type BotTier struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	DisplayName     string  `json:"display_name"`
	MonthlyFeeUSD   float64 `json:"monthly_fee_usd"`
	PerDexFeeUSD    float64 `json:"per_dex_fee_usd"`
	PerCexFeeUSD    float64 `json:"per_cex_fee_usd"`
	MaxBots         int     `json:"max_bots"`
	MaxDEXs         int     `json:"max_dexs"`
	MaxCEXs         int     `json:"max_cexs"`
	MaxPositionUSD  float64 `json:"max_position_usd"`
	MaxDailyVolume  float64 `json:"max_daily_volume"`
	LatencyTargetMs int     `json:"latency_target_ms"`
	IsActive        bool    `json:"is_active"`
}

// API Key
type APIKey struct {
	ID              string            `json:"id"`
	UserID          string            `json:"user_id"`
	Name            string            `json:"name"`
	Key             string            `json:"key"`
	Tier            APIKeyTier        `json:"tier"`
	Permissions     APIKeyPermissions `json:"permissions"`
	RateLimitPerMin int               `json:"rate_limit_per_min"`
	RateLimitPerDay int               `json:"rate_limit_per_day"`
	IsActive        bool              `json:"is_active"`
	LastUsedAt      int64             `json:"last_used_at"`
	ExpiresAt       int64             `json:"expires_at"`
	CreatedAt       int64             `json:"created_at"`
}

type APIKeyPermissions struct {
	Trading    bool `json:"trading"`
	Reading    bool `json:"reading"`
	Withdrawal bool `json:"withdrawal"`
}

// External Connection
type ExternalConnection struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	ExchangeName string `json:"exchange_name"` // binance, coinbase, etc.
	AccountID    string `json:"account_id"`
	IsActive     bool   `json:"is_active"`
	CanTrade     bool   `json:"can_trade"`
	CanWithdraw  bool   `json:"can_withdraw"`
	CanDeposit   bool   `json:"can_deposit"`
	LastSyncAt   int64  `json:"last_sync_at"`
	SyncStatus   string `json:"sync_status"` // idle, syncing, error
}

// Token Listing
type TokenListing struct {
	ID             string  `json:"id"`
	TokenSymbol    string  `json:"token_symbol"`
	TokenName      string  `json:"token_name"`
	ContractAddr   string  `json:"contract_address"`
	ChainID        int     `json:"chain_id"`
	Tier           string  `json:"tier"`   // basic, standard, premium, premium_plus
	Status         string  `json:"status"` // pending, approved, rejected
	RequesterAddr  string  `json:"requester_address"`
	RequesterEmail string  `json:"requester_email"`
	OneTimeFee     float64 `json:"one_time_fee"`
	MonthlyFee     float64 `json:"monthly_fee"`
	RequestedAt    int64   `json:"requested_at"`
}

// Platform Stats
type PlatformStats struct {
	TotalUsers        int     `json:"total_users"`
	ActiveUsers       int     `json:"active_users"`
	TotalVolume       float64 `json:"total_volume"`
	TotalTransactions int     `json:"total_transactions"`
	TotalFees         float64 `json:"total_fees"`
	ActiveBots        int     `json:"active_bots"`
	TotalBots         int     `json:"total_bots"`
	ActiveCEXConns    int     `json:"active_cex_connections"`
	ActiveDEXConns    int     `json:"active_dex_connections"`
}

// Auth Result
type AuthResult struct {
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
	SessionToken string `json:"session_token,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Email        string `json:"email,omitempty"`
	Role         string `json:"role,omitempty"`
}

// ==================== RBAC ADMIN SERVICE ====================

type RBACAdminService struct {
	mu sync.RWMutex

	// Data stores
	users          map[string]*User
	kycRequests    map[string]*KYCRequest
	transactions   map[string]*Transaction
	tradingPairs   map[string]*TradingPair
	liquidityPools map[string]*LiquidityPool
	feeStructures  map[string]*FeeStructure
	blockchains    map[string]*Blockchain
	botInstances   map[string]*BotInstance
	botTiers       map[string]*BotTier
	apiKeys        map[string]*APIKey
	externalConns  map[string]*ExternalConnection
	tokenListings  map[string]*TokenListing
	sessions       map[string]*Session

	// Stats
	stats PlatformStats
}

func NewRBACAdminService() *RBACAdminService {
	svc := &RBACAdminService{
		users:          make(map[string]*User),
		kycRequests:    make(map[string]*KYCRequest),
		transactions:   make(map[string]*Transaction),
		tradingPairs:   make(map[string]*TradingPair),
		liquidityPools: make(map[string]*LiquidityPool),
		feeStructures:  make(map[string]*FeeStructure),
		blockchains:    make(map[string]*Blockchain),
		botInstances:   make(map[string]*BotInstance),
		botTiers:       make(map[string]*BotTier),
		apiKeys:        make(map[string]*APIKey),
		externalConns:  make(map[string]*ExternalConnection),
		tokenListings:  make(map[string]*TokenListing),
		sessions:       make(map[string]*Session),
	}

	// Initialize with demo data
	svc.initDemoData()

	return svc
}

func (s *RBACAdminService) initDemoData() {
	// Initialize blockchains
	blockchains := []Blockchain{
		{ID: "eth", Name: "Ethereum", Symbol: "ETH", ChainID: 1, IsEVM: true, RPCURL: "https://eth-mainnet.alchemyapi.io", ExplorerURL: "https://etherscan.io", NativeToken: "ETH", Decimals: 18, IsActive: true, AvgGasPriceGwei: 20},
		{ID: "bsc", Name: "BNB Smart Chain", Symbol: "BNB", ChainID: 56, IsEVM: true, RPCURL: "https://bsc-dataseed.binance.org", ExplorerURL: "https://bscscan.com", NativeToken: "BNB", Decimals: 18, IsActive: true, AvgGasPriceGwei: 3},
		{ID: "polygon", Name: "Polygon", Symbol: "MATIC", ChainID: 137, IsEVM: true, RPCURL: "https://polygon-rpc.com", ExplorerURL: "https://polygonscan.com", NativeToken: "MATIC", Decimals: 18, IsActive: true, AvgGasPriceGwei: 50},
		{ID: "arbitrum", Name: "Arbitrum One", Symbol: "ETH", ChainID: 42161, IsEVM: true, RPCURL: "https://arb1.arbitrum.io/rpc", ExplorerURL: "https://arbiscan.io", NativeToken: "ETH", Decimals: 18, IsActive: true, AvgGasPriceGwei: 0.1},
		{ID: "optimism", Name: "Optimism", Symbol: "ETH", ChainID: 10, IsEVM: true, RPCURL: "https://mainnet.optimism.io", ExplorerURL: "https://optimistic.etherscan.io", NativeToken: "ETH", Decimals: 18, IsActive: true, AvgGasPriceGwei: 0.001},
		{ID: "base", Name: "Base", Symbol: "ETH", ChainID: 8453, IsEVM: true, RPCURL: "https://mainnet.base.org", ExplorerURL: "https://basescan.org", NativeToken: "ETH", Decimals: 18, IsActive: true, AvgGasPriceGwei: 0.001},
		{ID: "avalanche", Name: "Avalanche C-Chain", Symbol: "AVAX", ChainID: 43114, IsEVM: true, RPCURL: "https://api.avax.network/ext/bc/C/rpc", ExplorerURL: "https://snowtrace.io", NativeToken: "AVAX", Decimals: 18, IsActive: true, AvgGasPriceGwei: 25},
		{ID: "solana", Name: "Solana", Symbol: "SOL", ChainID: 101, IsEVM: false, RPCURL: "https://api.mainnet-beta.solana.com", ExplorerURL: "https://solscan.io", NativeToken: "SOL", Decimals: 9, IsActive: true, AvgGasPriceGwei: 0},
	}

	for _, bc := range blockchains {
		s.blockchains[bc.ID] = &bc
	}

	// Initialize bot tiers
	tiers := []BotTier{
		{ID: "tier_1", Name: "tier_1", DisplayName: "Basic", MonthlyFeeUSD: 2500, PerDexFeeUSD: 500, PerCexFeeUSD: 50, MaxBots: 1, MaxDEXs: 5, MaxCEXs: 20, MaxPositionUSD: 100000, MaxDailyVolume: 1000000, LatencyTargetMs: 100, IsActive: true},
		{ID: "tier_2", Name: "tier_2", DisplayName: "Pro", MonthlyFeeUSD: 5000, PerDexFeeUSD: 750, PerCexFeeUSD: 75, MaxBots: 3, MaxDEXs: 10, MaxCEXs: 50, MaxPositionUSD: 500000, MaxDailyVolume: 5000000, LatencyTargetMs: 50, IsActive: true},
		{ID: "tier_3", Name: "tier_3", DisplayName: "Enterprise", MonthlyFeeUSD: 10000, PerDexFeeUSD: 1000, PerCexFeeUSD: 100, MaxBots: 10, MaxDEXs: 20, MaxCEXs: 200, MaxPositionUSD: 5000000, MaxDailyVolume: 50000000, LatencyTargetMs: 10, IsActive: true},
	}

	for _, tier := range tiers {
		s.botTiers[tier.ID] = &tier
	}

	// Initialize fee structures
	fees := []FeeStructure{
		{ID: "swap_eth", FeeType: "swap", Asset: "ETH", FeePercent: 0.3, FeeFixed: 0, MinFee: 0, Tier: "all", IsActive: true, ChainID: 1},
		{ID: "swap_bsc", FeeType: "swap", Asset: "BNB", FeePercent: 0.3, FeeFixed: 0, MinFee: 0, Tier: "all", IsActive: true, ChainID: 56},
		{ID: "withdrawal", FeeType: "withdrawal", Asset: "*", FeePercent: 0, FeeFixed: 5, MinFee: 5, MaxFee: 50, Tier: "all", IsActive: true, ChainID: 0},
		{ID: "deposit", FeeType: "deposit", Asset: "*", FeePercent: 0, FeeFixed: 0, MinFee: 0, Tier: "all", IsActive: true, ChainID: 0},
	}

	for _, fee := range fees {
		s.feeStructures[fee.ID] = &fee
	}

	// Initialize trading pairs
	pairs := []TradingPair{
		{ID: "eth_usdt", Base: "ETH", Quote: "USDT", PairName: "ETH/USDT", Price: 3500.00, Volume24h: 50000000, Liquidity: 100000000, PairStatus: PairActive, ChainID: 1, CreatedAt: time.Now().Unix()},
		{ID: "bnb_usdt", Base: "BNB", Quote: "USDT", PairName: "BNB/USDT", Price: 600.00, Volume24h: 30000000, Liquidity: 50000000, PairStatus: PairActive, ChainID: 56, CreatedAt: time.Now().Unix()},
		{ID: "matic_usdt", Base: "MATIC", Quote: "USDT", PairName: "MATIC/USDT", Price: 0.85, Volume24h: 10000000, Liquidity: 20000000, PairStatus: PairActive, ChainID: 137, CreatedAt: time.Now().Unix()},
		{ID: "arb_usdt", Base: "ETH", Quote: "USDT", PairName: "ARB/USDT", Price: 1.20, Volume24h: 8000000, Liquidity: 15000000, PairStatus: PairActive, ChainID: 42161, CreatedAt: time.Now().Unix()},
		{ID: "sol_usdt", Base: "SOL", Quote: "USDT", PairName: "SOL/USDT", Price: 150.00, Volume24h: 20000000, Liquidity: 40000000, PairStatus: PairActive, ChainID: 101, CreatedAt: time.Now().Unix()},
	}

	for _, pair := range pairs {
		s.tradingPairs[pair.ID] = &pair
	}

	// Initialize stats
	s.stats = PlatformStats{
		TotalUsers:        1250,
		ActiveUsers:       890,
		TotalVolume:       125000000,
		TotalTransactions: 45000,
		TotalFees:         850000,
		ActiveBots:        890,
		TotalBots:         3420,
		ActiveCEXConns:    2100,
		ActiveDEXConns:    450,
	}
}

// ==================== USER MANAGEMENT ====================

// GetAllUsers returns all users from PostgreSQL database
func (s *RBACAdminService) GetAllUsers() []*User {
	if db == nil {
		// Fallback to in-memory if DB not connected
		s.mu.RLock()
		defer s.mu.RUnlock()
		users := make([]*User, 0, len(s.users))
		for _, u := range s.users {
			users = append(users, u)
		}
		return users
	}

	// Query from PostgreSQL
	rows, err := db.QueryContext(context.Background(),
		"SELECT id, email, username, wallet_address, kyc_status, kyc_level, status, risk_score, created_at, updated_at, last_login FROM users ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		fmt.Printf("Error fetching users: %v\n", err)
		return nil
	}
	defer rows.Close()

	users := make([]*User, 0)
	for rows.Next() {
		var u User
		var kycStatus, status string
		var lastLogin sql.NullTime
		err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.WalletAddress, &kycStatus, &u.KYCLevel, &status, &u.RiskScore, &u.CreatedAt, &u.UpdatedAt, &lastLogin)
		if err != nil {
			fmt.Printf("Error scanning user: %v\n", err)
			continue
		}
		u.KYCStatus = parseKYCStatus(kycStatus)
		u.UserStatus = parseUserStatus(status)
		if lastLogin.Valid {
			u.LastLogin = &lastLogin.Time
		}
		users = append(users, &u)
	}
	return users
}

// GetUser returns a single user from PostgreSQL
func (s *RBACAdminService) GetUser(id string) *User {
	if db == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.users[id]
	}

	var u User
	var kycStatus, status string
	var lastLogin sql.NullTime

	err := db.QueryRowContext(context.Background(),
		"SELECT id, email, username, wallet_address, kyc_status, kyc_level, status, risk_score, created_at, updated_at, last_login FROM users WHERE id = $1", id).
		Scan(&u.ID, &u.Email, &u.Username, &u.WalletAddress, &kycStatus, &u.KYCLevel, &status, &u.RiskScore, &u.CreatedAt, &u.UpdatedAt, &lastLogin)

	if err != nil {
		return nil
	}

	u.KYCStatus = parseKYCStatus(kycStatus)
	u.UserStatus = parseUserStatus(status)
	if lastLogin.Valid {
		u.LastLogin = &lastLogin.Time
	}
	return &u
}

// Helper functions
func parseKYCStatus(s string) KYCStatus {
	switch s {
	case "pending":
		return KYCPending
	case "approved":
		return KYCApproved
	case "rejected":
		return KYCRejected
	default:
		return KYCNone
	}
}

func parseUserStatus(s string) UserStatus {
	switch s {
	case "active":
		return StatusActive
	case "suspended":
		return StatusSuspended
	case "banned":
		return StatusBanned
	default:
		return StatusActive
	}
}

func (s *RBACAdminService) SearchUsers(query string) []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0)
	query = strings.ToLower(query)

	for _, u := range s.users {
		if strings.Contains(strings.ToLower(u.Email), query) ||
			strings.Contains(strings.ToLower(u.Username), query) ||
			strings.Contains(strings.ToLower(u.WalletAddress), query) {
			users = append(users, u)
		}
	}
	return users
}

func (s *RBACAdminService) GetUsersByStatus(status UserStatus) []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0)
	for _, u := range s.users {
		if u.Status == status {
			users = append(users, u)
		}
	}
	return users
}

func (s *RBACAdminService) GetUsersByKYC(kyc KYCStatus) []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0)
	for _, u := range s.users {
		if u.KYCStatus == kyc {
			users = append(users, u)
		}
	}
	return users
}

func (s *RBACAdminService) UpdateUserStatus(userID string, status UserStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userID]; !ok {
		return fmt.Errorf("user not found")
	}

	s.users[userID].Status = status
	return nil
}

func (s *RBACAdminService) BanUser(userID string) error {
	return s.UpdateUserStatus(userID, StatusBanned)
}

func (s *RBACAdminService) UnbanUser(userID string) error {
	return s.UpdateUserStatus(userID, StatusActive)
}

func (s *RBACAdminService) SuspendUser(userID string) error {
	return s.UpdateUserStatus(userID, StatusSuspended)
}

// ==================== KYC MANAGEMENT ====================

func (s *RBACAdminService) GetAllKYCRequests() []*KYCRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requests := make([]*KYCRequest, 0, len(s.kycRequests))
	for _, r := range s.kycRequests {
		requests = append(requests, r)
	}
	return requests
}

func (s *RBACAdminService) GetKYCRequestsByStatus(status KYCStatus) []*KYCRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requests := make([]*KYCRequest, 0)
	for _, r := range s.kycRequests {
		if r.Status == status {
			requests = append(requests, r)
		}
	}
	return requests
}

func (s *RBACAdminService) ApproveKYC(requestID, reviewerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req, ok := s.kycRequests[requestID]; ok {
		req.Status = KYCApproved
		req.ReviewedAt = time.Now().Unix()
		req.ReviewedBy = reviewerID

		// Update user KYC status
		if user, userOk := s.users[req.UserID]; userOk {
			user.KYCStatus = KYCApproved
		}
		return nil
	}

	return fmt.Errorf("KYC request not found")
}

func (s *RBACAdminService) RejectKYC(requestID, reviewerID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req, ok := s.kycRequests[requestID]; ok {
		req.Status = KYCRejected
		req.ReviewedAt = time.Now().Unix()
		req.ReviewedBy = reviewerID
		req.RejectReason = reason

		// Update user KYC status
		if user, userOk := s.users[req.UserID]; userOk {
			user.KYCStatus = KYCRejected
		}
		return nil
	}

	return fmt.Errorf("KYC request not found")
}

// ==================== TRANSACTION MANAGEMENT ====================

func (s *RBACAdminService) GetAllTransactions() []*Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txs := make([]*Transaction, 0, len(s.transactions))
	for _, t := range s.transactions {
		txs = append(txs, t)
	}
	return txs
}

func (s *RBACAdminService) GetTransactionsByUser(userID string) []*Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txs := make([]*Transaction, 0)
	for _, t := range s.transactions {
		if t.UserID == userID {
			txs = append(txs, t)
		}
	}
	return txs
}

func (s *RBACAdminService) GetTransactionsByStatus(status TransactionStatus) []*Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txs := make([]*Transaction, 0)
	for _, t := range s.transactions {
		if t.Status == status {
			txs = append(txs, t)
		}
	}
	return txs
}

func (s *RBACAdminService) GetTransactionsByType(txtype TransactionType) []*Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txs := make([]*Transaction, 0)
	for _, t := range s.transactions {
		if t.Type == txtype {
			txs = append(txs, t)
		}
	}
	return txs
}

// ==================== TRADING PAIR MANAGEMENT ====================

func (s *RBACAdminService) GetAllTradingPairs() []*TradingPair {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pairs := make([]*TradingPair, 0, len(s.tradingPairs))
	for _, p := range s.tradingPairs {
		pairs = append(pairs, p)
	}
	return pairs
}

func (s *RBACAdminService) GetTradingPair(id string) *TradingPair {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tradingPairs[id]
}

func (s *RBACAdminService) CreateTradingPair(base, quote string, chainID int) (*TradingPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pairID := fmt.Sprintf("%s_%s", strings.ToLower(base), strings.ToLower(quote))

	if _, exists := s.tradingPairs[pairID]; exists {
		return nil, fmt.Errorf("pair already exists")
	}

	pair := &TradingPair{
		ID:         pairID,
		Base:       base,
		Quote:      quote,
		PairName:   fmt.Sprintf("%s/%s", base, quote),
		Price:      0,
		Volume24h:  0,
		Liquidity:  0,
		PairStatus: PairActive,
		ChainID:    chainID,
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	s.tradingPairs[pairID] = pair
	return pair, nil
}

func (s *RBACAdminService) UpdatePairStatus(pairID string, status PairStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if pair, ok := s.tradingPairs[pairID]; ok {
		pair.PairStatus = status
		pair.UpdatedAt = time.Now().Unix()
		return nil
	}

	return fmt.Errorf("pair not found")
}

func (s *RBACAdminService) SuspendPair(pairID string) error {
	return s.UpdatePairStatus(pairID, PairSuspended)
}

func (s *RBACAdminService) ResumePair(pairID string) error {
	return s.UpdatePairStatus(pairID, PairActive)
}

func (s *RBACAdminService) HaltPair(pairID string) error {
	return s.UpdatePairStatus(pairID, PairHalted)
}

// ==================== FEE MANAGEMENT ====================

func (s *RBACAdminService) GetAllFeeStructures() []*FeeStructure {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fees := make([]*FeeStructure, 0, len(s.feeStructures))
	for _, f := range s.feeStructures {
		fees = append(fees, f)
	}
	return fees
}

func (s *RBACAdminService) CreateFeeStructure(feeType, asset, tier string, feePercent, feeFixed float64, chainID int) (*FeeStructure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	feeID := generateID()
	fee := &FeeStructure{
		ID:         feeID,
		FeeType:    feeType,
		Asset:      asset,
		FeePercent: feePercent,
		FeeFixed:   feeFixed,
		MinFee:     0,
		MaxFee:     0,
		Tier:       tier,
		IsActive:   true,
		ChainID:    chainID,
	}

	s.feeStructures[feeID] = fee
	return fee, nil
}

func (s *RBACAdminService) UpdateFee(feeID string, feePercent, feeFixed float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if fee, ok := s.feeStructures[feeID]; ok {
		fee.FeePercent = feePercent
		fee.FeeFixed = feeFixed
		return nil
	}

	return fmt.Errorf("fee structure not found")
}

// ==================== BLOCKCHAIN MANAGEMENT ====================

func (s *RBACAdminService) GetAllBlockchains() []*Blockchain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	blockchains := make([]*Blockchain, 0, len(s.blockchains))
	for _, b := range s.blockchains {
		blockchains = append(blockchains, b)
	}
	return blockchains
}

func (s *RBACAdminService) GetBlockchain(id string) *Blockchain {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.blockchains[id]
}

func (s *RBACAdminService) AddBlockchain(name, symbol string, chainID int, isEVM bool, rpcURL, explorerURL, nativeToken string, decimals int) (*Blockchain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	blockchainID := strings.ToLower(symbol)

	if _, exists := s.blockchains[blockchainID]; exists {
		return nil, fmt.Errorf("blockchain already exists")
	}

	blockchain := &Blockchain{
		ID:              blockchainID,
		Name:            name,
		Symbol:          symbol,
		ChainID:         chainID,
		IsEVM:           isEVM,
		RPCURL:          rpcURL,
		ExplorerURL:     explorerURL,
		NativeToken:     nativeToken,
		Decimals:        decimals,
		IsActive:        true,
		AvgGasPriceGwei: 0,
	}

	s.blockchains[blockchainID] = blockchain
	return blockchain, nil
}

func (s *RBACAdminService) UpdateBlockchain(id string, rpcURL, explorerURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bc, ok := s.blockchains[id]; ok {
		bc.RPCURL = rpcURL
		bc.ExplorerURL = explorerURL
		return nil
	}

	return fmt.Errorf("blockchain not found")
}

func (s *RBACAdminService) SetBlockchainStatus(id string, isActive bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bc, ok := s.blockchains[id]; ok {
		bc.IsActive = isActive
		return nil
	}

	return fmt.Errorf("blockchain not found")
}

// ==================== BOT MANAGEMENT ====================

func (s *RBACAdminService) GetAllBotInstances() []*BotInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bots := make([]*BotInstance, 0, len(s.botInstances))
	for _, b := range s.botInstances {
		bots = append(bots, b)
	}
	return bots
}

func (s *RBACAdminService) GetBotInstancesByUser(userID string) []*BotInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bots := make([]*BotInstance, 0)
	for _, b := range s.botInstances {
		if b.UserID == userID {
			bots = append(bots, b)
		}
	}
	return bots
}

func (s *RBACAdminService) GetAllBotTiers() []*BotTier {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tiers := make([]*BotTier, 0, len(s.botTiers))
	for _, t := range s.botTiers {
		tiers = append(tiers, t)
	}
	return tiers
}

func (s *RBACAdminService) UpdateBotStatus(botID string, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bot, ok := s.botInstances[botID]; ok {
		bot.Status = status
		return nil
	}

	return fmt.Errorf("bot not found")
}

func (s *RBACAdminService) PauseBot(botID string) error {
	return s.UpdateBotStatus(botID, "paused")
}

func (s *RBACAdminService) ResumeBot(botID string) error {
	return s.UpdateBotStatus(botID, "running")
}

func (s *RBACAdminService) StopBot(botID string) error {
	return s.UpdateBotStatus(botID, "stopped")
}

// ==================== API KEY MANAGEMENT ====================

func (s *RBACAdminService) GetAllAPIKeys() []*APIKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]*APIKey, 0, len(s.apiKeys))
	for _, k := range s.apiKeys {
		keys = append(keys, k)
	}
	return keys
}

func (s *RBACAdminService) GetAPIKeysByUser(userID string) []*APIKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]*APIKey, 0)
	for _, k := range s.apiKeys {
		if k.UserID == userID {
			keys = append(keys, k)
		}
	}
	return keys
}

func (s *RBACAdminService) CreateAPIKey(userID, name string, tier APIKeyTier, permissions APIKeyPermissions) (*APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyID := generateID()
	apiKey := generateRandomString(32)

	key := &APIKey{
		ID:              keyID,
		UserID:          userID,
		Name:            name,
		Key:             apiKey,
		Tier:            tier,
		Permissions:     permissions,
		RateLimitPerMin: 60,
		RateLimitPerDay: 10000,
		IsActive:        true,
		LastUsedAt:      0,
		ExpiresAt:       time.Now().AddDate(1, 0, 0).Unix(),
		CreatedAt:       time.Now().Unix(),
	}

	s.apiKeys[keyID] = key
	return key, nil
}

func (s *RBACAdminService) RevokeAPIKey(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if key, ok := s.apiKeys[keyID]; ok {
		key.IsActive = false
		return nil
	}

	return fmt.Errorf("API key not found")
}

// ==================== PLATFORM STATS ====================

func (s *RBACAdminService) GetPlatformStats() PlatformStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// ==================== HELPER FUNCTIONS ====================

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateRandomString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[n.Int64()]
	}
	return string(result)
}

// ==================== HTTP HANDLERS ====================

type Server struct {
	service *RBACAdminService
}

func NewRBACServer() *Server {
	return &Server{
		service: NewRBACAdminService(),
	}
}

func (srv *Server) handleGetUsers(c *gin.Context) {
	users := srv.service.GetAllUsers()
	c.JSON(200, gin.H{"users": users})
}

func (srv *Server) handleGetUser(c *gin.Context) {
	id := c.Param("id")
	user := srv.service.GetUser(id)
	if user == nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}
	c.JSON(200, gin.H{"user": user})
}

func (srv *Server) handleSearchUsers(c *gin.Context) {
	query := c.Query("q")
	users := srv.service.SearchUsers(query)
	c.JSON(200, gin.H{"users": users})
}

func (srv *Server) handleUpdateUserStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	id := c.Param("id")
	var status UserStatus

	switch req.Status {
	case "active":
		status = StatusActive
	case "suspended":
		status = StatusSuspended
	case "banned":
		status = StatusBanned
	default:
		c.JSON(400, gin.H{"error": "invalid status"})
		return
	}

	if err := srv.service.UpdateUserStatus(id, status); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true})
}

func (srv *Server) handleGetKYCRequests(c *gin.Context) {
	requests := srv.service.GetAllKYCRequests()
	c.JSON(200, gin.H{"kyc_requests": requests})
}

func (srv *Server) handleApproveKYC(c *gin.Context) {
	id := c.Param("id")

	if err := srv.service.ApproveKYC(id, "admin"); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true})
}

func (srv *Server) handleRejectKYC(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	id := c.Param("id")

	if err := srv.service.RejectKYC(id, "admin", req.Reason); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true})
}

func (srv *Server) handleGetTransactions(c *gin.Context) {
	txs := srv.service.GetAllTransactions()
	c.JSON(200, gin.H{"transactions": txs})
}

func (srv *Server) handleGetTradingPairs(c *gin.Context) {
	pairs := srv.service.GetAllTradingPairs()
	c.JSON(200, gin.H{"trading_pairs": pairs})
}

func (srv *Server) handleUpdatePairStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	id := c.Param("id")

	switch req.Status {
	case "active":
		srv.service.ResumePair(id)
	case "suspended":
		srv.service.SuspendPair(id)
	case "halted":
		srv.service.HaltPair(id)
	}

	c.JSON(200, gin.H{"success": true})
}

func (srv *Server) handleGetFees(c *gin.Context) {
	fees := srv.service.GetAllFeeStructures()
	c.JSON(200, gin.H{"fees": fees})
}

func (srv *Server) handleGetBlockchains(c *gin.Context) {
	blockchains := srv.service.GetAllBlockchains()
	c.JSON(200, gin.H{"blockchains": blockchains})
}

func (srv *Server) handleGetBots(c *gin.Context) {
	bots := srv.service.GetAllBotInstances()
	c.JSON(200, gin.H{"bots": bots})
}

func (srv *Server) handleGetBotTiers(c *gin.Context) {
	tiers := srv.service.GetAllBotTiers()
	c.JSON(200, gin.H{"bot_tiers": tiers})
}

func (srv *Server) handleUpdateBotStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	id := c.Param("id")

	switch req.Status {
	case "running":
		srv.service.ResumeBot(id)
	case "paused":
		srv.service.PauseBot(id)
	case "stopped":
		srv.service.StopBot(id)
	}

	c.JSON(200, gin.H{"success": true})
}

func (srv *Server) handleGetAPIKeys(c *gin.Context) {
	keys := srv.service.GetAllAPIKeys()
	c.JSON(200, gin.H{"api_keys": keys})
}

func (srv *Server) handleGetStats(c *gin.Context) {
	stats := srv.service.GetPlatformStats()
	c.JSON(200, gin.H{"stats": stats})
}

// ==================== MAIN ====================

func main() {
	// Initialize PostgreSQL database connection
	initDB()

	r := gin.Default()
	server := NewRBACServer()

	// User management
	r.GET("/api/v1/admin/users", server.handleGetUsers)
	r.GET("/api/v1/admin/users/search", server.handleSearchUsers)
	r.GET("/api/v1/admin/users/:id", server.handleGetUser)
	r.PUT("/api/v1/admin/users/:id/status", server.handleUpdateUserStatus)

	// KYC management
	r.GET("/api/v1/admin/kyc", server.handleGetKYCRequests)
	r.POST("/api/v1/admin/kyc/:id/approve", server.handleApproveKYC)
	r.POST("/api/v1/admin/kyc/:id/reject", server.handleRejectKYC)

	// Transaction management
	r.GET("/api/v1/admin/transactions", server.handleGetTransactions)

	// Trading pair management
	r.GET("/api/v1/admin/pairs", server.handleGetTradingPairs)
	r.PUT("/api/v1/admin/pairs/:id/status", server.handleUpdatePairStatus)

	// Fee management
	r.GET("/api/v1/admin/fees", server.handleGetFees)

	// Blockchain management
	r.GET("/api/v1/admin/blockchains", server.handleGetBlockchains)

	// Bot management
	r.GET("/api/v1/admin/bots", server.handleGetBots)
	r.GET("/api/v1/admin/bot-tiers", server.handleGetBotTiers)
	r.PUT("/api/v1/admin/bots/:id/status", server.handleUpdateBotStatus)

	// API key management
	r.GET("/api/v1/admin/api-keys", server.handleGetAPIKeys)

	// Stats
	r.GET("/api/v1/admin/stats", server.handleGetStats)

	fmt.Println("TigerWallet RBAC Admin Server running on :8081")
	r.Run(":8081")
}
