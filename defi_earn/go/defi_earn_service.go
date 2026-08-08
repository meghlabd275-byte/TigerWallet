/**
 * TigerWallet DeFi & Earn Module
 * 
 * Comprehensive DeFi services including:
 * - Launchpad & Launchpool
 * - Staking & Earn Products
 * - ETF Trading
 * - Convert System
 * - Internal Transfers
 * - Coupons & Red Packets
 * 
 * Built with Go for high-load distributed operations.
 */

package defi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

// Launchpad represents an IDO launchpad
type Launchpad struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Token           string             `json:"token"`
	TokenAddress    string             `json:"token_address"`
	TokenSymbol    string             `json:"token_symbol"`
	TotalSupply    float64           `json:"total_supply"`
	SoftCap        float64           `json:"soft_cap"`
	HardCap        float64           `json:"hard_cap"`
	MinAllocation  float64           `json:"min_allocation"`
	MaxAllocation  float64           `json:"max_allocation"`
	PricePerToken  float64           `json:"price_per_token"`
	AcceptedToken  string             `json:"accepted_token"`
	StartTime      int64              `json:"start_time"`
	EndTime        int64              `json:"end_time"`
	ReleaseTime    int64              `json:"release_time"`
	Status         string             `json:"status"` // upcoming, active, completed, cancelled
	PoolDetails    []PoolDetail       `json:"pool_details"`
	Progress       float64            `json:"progress"`
	RaisedAmount   float64            `json:"raised_amount"`
	Participants   int               `json:"participants"`
	CreatedAt      int64              `json:"created_at"`
	UpdatedAt      int64              `json:"updated_at"`
}

// PoolDetail represents a pool in launchpad
type PoolDetail struct {
	PoolID      string  `json:"pool_id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"` // public, private, whitelist
	Allocation  float64 `json:"allocation"`
	Weight     float64 `json:"weight"`
	MinStake   float64 `json:"min_stake"`
	RequiredToken string `json:"required_token"`
}

// Launchpool represents a staking pool for token distribution
type Launchpool struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	StakeToken     string             `json:"stake_token"`
	RewardToken    string             `json:"reward_token"`
	RewardPerBlock float64            `json:"reward_per_block"`
	TotalStaked    float64           `json:"total_staked"`
	StartTime      int64              `json:"start_time"`
	EndTime        int64              `json:"end_time"`
	Status         string             `json:"status"`
	Participants   int               `json:"participants"`
	CreatedAt      int64              `json:"created_at"`
}

// StakingProduct represents a staking product
type StakingProduct struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Token          string             `json:"token"`
	APY            float64            `json:"apy"`
	MinStake       float64            `json:"min_stake"`
	MaxStake       float64            `json:"max_stake"`
	LockPeriod     int64              `json:"lock_period"` // seconds
	EarlyUnstakeFee float64           `json:"early_unstake_fee"`
	TotalStaked    float64           `json:"total_staked"`
	StakersCount   int               `json:"stakers_count"`
	Status         string             `json:"status"` // active, paused, ended
	CreatedAt      int64              `json:"created_at"`
	EndTime        int64              `json:"end_time,omitempty"`
}

// EarnProduct represents an earn/savings product
type EarnProduct struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Token          string  `json:"token"`
	APY            float64 `json:"apy"`
	MinDeposit     float64 `json:"min_deposit"`
	MaxDeposit     float64 `json:"max_deposit"`
	Flexible       bool    `json:"flexible"` // true = flexible, false = fixed term
	TermDays       int     `json:"term_days,omitempty"`
	TotalDeposited float64 `json:"total_deposited"`
	DepositorsCount int   `json:"depositors_count"`
	Status         string  `json:"status"`
}

// ETFAvailable represents available ETF products
type ETFAvailable struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Symbol     string  `json:"symbol"`
	Underlying []string `json:"underlying"` // tokens in the ETF
	Ratio       map[string]float64 `json:"ratio"` // token -> ratio
	TotalSupply float64 `json:"total_supply"`
	NAV         float64 `json:"nav"` // net asset value
	APY         float64 `json:"apy"`
	ManagementFee float64 `json:"management_fee"`
	Status      string  `json:"status"`
}

// Coupon represents a promotional coupon
type Coupon struct {
	ID            string             `json:"id"`
	Code          string             `json:"code"`
	Type          string             `json:"type"` // discount, cashback, reward
	Value         float64            `json:"value"`
	MinTransaction float64          `json:"min_transaction"`
	MaxUses       int               `json:"max_uses"`
	UsedCount     int               `json:"used_count"`
	ValidFrom    int64              `json:"valid_from"`
	ValidUntil   int64              `json:"valid_until"`
	EligibleUsers []string         `json:"eligible_users,omitempty"`
	Status       string             `json:"status"` // active, expired, cancelled
	CreatedAt    int64              `json:"created_at"`
}

// RedPacket represents a red packet (gift) distribution
type RedPacket struct {
	ID             string             `json:"id"`
	SenderID      string             `json:"sender_id"`
	TotalAmount   float64            `json:"total_amount"`
	Token         string             `json:"token"`
	PacketType   string             `json:"packet_type"` // random, equal
	Count        int                `json:"count"`
	RemainingAmount float64         `json:"remaining_amount"`
	RemainingCount int              `json:"remaining_count"`
	Message      string             `json:"message"`
	ExpiresAt    int64              `json:"expires_at"`
	Status       string             `json:"status"` // active, claimed, expired
	Claims       []RedPacketClaim  `json:"claims"`
	CreatedAt    int64              `json:"created_at"`
}

// RedPacketClaim represents a claim on a red packet
type RedPacketClaim struct {
	ClaimID   string  `json:"claim_id"`
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	ClaimedAt int64   `json:"claimed_at"`
}

// InternalTransfer represents an internal wallet transfer
type InternalTransfer struct {
	ID          string  `json:"id"`
	FromUserID string  `json:"from_user_id"`
	ToUserID   string  `json:"to_user_id"`
	Token      string  `json:"token"`
	Amount     float64 `json:"amount"`
	Fee        float64 `json:"fee"`
	Status     string  `json:"status"` // pending, completed, failed
	TxHash     string  `json:"tx_hash,omitempty"`
	CreatedAt  int64   `json:"created_at"`
	CompletedAt int64   `json:"completed_at,omitempty"`
}

// ============================================================================
// DeFi Earn Service
// ============================================================================

// DeFiEarnService provides DeFi and Earn functionality
type DeFiEarnService struct {
	mu           sync.RWMutex
	launchpads   map[string]*Launchpad
	launchpools map[string]*Launchpool
	staking     map[string]*StakingProduct
	earn        map[string]*EarnProduct
	etf         map[string]*ETFAvailable
	coupons     map[string]*Coupon
	redPackets map[string]*RedPacket
	transfers   map[string]*InternalTransfer
	stakes      map[string][]StakingPosition
	deposits    map[string][]EarnDeposit
}

// StakingPosition represents a user's staking position
type StakingPosition struct {
	ID           string  `json:"id"`
	UserID      string  `json:"user_id"`
	ProductID   string  `json:"product_id"`
	Amount      float64 `json:"amount"`
	StartTime   int64   `json:"start_time"`
	EndTime     int64   `json:"end_time"`
	RewardDebt  float64 `json:"reward_debt"`
	Claimed     float64 `json:"claimed"`
	Status      string  `json:"status"` // staked, withdrawn
}

// EarnDeposit represents a user's earn deposit
type EarnDeposit struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	ProductID   string  `json:"product_id"`
	Amount      float64 `json:"amount"`
	StartTime   int64   `json:"start_time"`
	EndTime     int64   `json:"end_time"`
	Interest    float64 `json:"interest"`
	Status      string  `json:"status"` // active, matured, withdrawn
}

// NewDeFiEarnService creates a new DeFi Earn service
func NewDeFiEarnService() *DeFiEarnService {
	return &DeFiEarnService{
		launchpads:   make(map[string]*Launchpad),
		launchpools: make(map[string]*Launchpool),
		staking:     make(map[string]*StakingProduct),
		earn:        make(map[string]*EarnProduct),
		etf:         make(map[string]*ETFAvailable),
		coupons:     make(map[string]*Coupon),
		redPackets: make(map[string]*RedPacket),
		transfers:   make(map[string]*InternalTransfer),
		stakes:      make(map[string][]StakingPosition),
		deposits:    make(map[string][]EarnDeposit),
	}
}

// ============================================================================
// Launchpad Functions
// ============================================================================

// CreateLaunchpad creates a new launchpad
func (s *DeFiEarnService) CreateLaunchpad(ctx context.Context, lp *Launchpad) (*Launchpad, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lp.ID = generateID()
	lp.Status = "upcoming"
	lp.Progress = 0
	lp.RaisedAmount = 0
	lp.Participants = 0
	lp.CreatedAt = time.Now().UnixMilli()
	lp.UpdatedAt = time.Now().UnixMilli()

	s.launchpads[lp.ID] = lp
	return lp, nil
}

// GetLaunchpad retrieves a launchpad
func (s *DeFiEarnService) GetLaunchpad(ctx context.Context, id string) (*Launchpad, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lp, exists := s.launchpads[id]
	if !exists {
		return nil, fmt.Errorf("launchpad not found")
	}
	return lp, nil
}

// ListLaunchpads lists all launchpads with filters
func (s *DeFiEarnService) ListLaunchpads(ctx context.Context, status string) ([]*Launchpad, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Launchpad, 0)
	for _, lp := range s.launchpads {
		if status != "" && lp.Status != status {
			continue
		}
		result = append(result, lp)
	}
	return result, nil
}

// ParticipateInLaunchpad participates in a launchpad
func (s *DeFiEarnService) ParticipateInLaunchpad(ctx context.Context, launchpadID, userID, poolID string, amount float64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lp, exists := s.launchpads[launchpadID]
	if !exists {
		return "", fmt.Errorf("launchpad not found")
	}

	now := time.Now().UnixMilli()
	if now < lp.StartTime {
		return "", fmt.Errorf("launchpad not started yet")
	}
	if now > lp.EndTime {
		return "", fmt.Errorf("launchpad ended")
	}

	if amount < lp.MinAllocation {
		return "", fmt.Errorf("amount below minimum allocation")
	}
	if amount > lp.MaxAllocation {
		return "", fmt.Errorf("amount above maximum allocation")
	}

	// Calculate tokens received
	tokensReceived := amount / lp.PricePerToken

	// Update progress
	lp.RaisedAmount += amount
	lp.Participants++
	lp.Progress = (lp.RaisedAmount / lp.HardCap) * 100

	// Check if hard cap reached
	if lp.RaisedAmount >= lp.HardCap {
		lp.Status = "completed"
	}

	return generateID(), nil
}

// ============================================================================
// Launchpool Functions
// ============================================================================

// CreateLaunchpool creates a new launchpool
func (s *DeFiEarnService) CreateLaunchpool(ctx context.Context, lp *Launchpool) (*Launchpool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lp.ID = generateID()
	lp.Status = "upcoming"
	lp.TotalStaked = 0
	lp.Participants = 0
	lp.CreatedAt = time.Now().UnixMilli()

	s.launchpools[lp.ID] = lp
	return lp, nil
}

// GetLaunchpool retrieves a launchpool
func (s *DeFiEarnService) GetLaunchpool(ctx context.Context, id string) (*Launchpool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lp, exists := s.launchpools[id]
	if !exists {
		return nil, fmt.Errorf("launchpool not found")
	}
	return lp, nil
}

// StakeInLaunchpool stakes tokens in a launchpool
func (s *DeFiEarnService) StakeInLaunchpool(ctx context.Context, poolID, userID string, amount float64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lp, exists := s.launchpools[poolID]
	if !exists {
		return "", fmt.Errorf("launchpool not found")
	}

	now := time.Now().UnixMilli()
	if now < lp.StartTime {
		return "", fmt.Errorf("launchpool not started yet")
	}
	if now > lp.EndTime {
		return "", fmt.Errorf("launchpool ended")
	}

	positionID := generateID()
	position := StakingPosition{
		ID:         positionID,
		UserID:     userID,
		ProductID:  poolID,
		Amount:     amount,
		StartTime:  now,
		RewardDebt: 0,
		Claimed:    0,
		Status:     "staked",
	}

	// Add to user's stakes
	s.stakes[userID] = append(s.stakes[userID], position)

	// Update pool
	lp.TotalStaked += amount
	lp.Participants++

	return positionID, nil
}

// ClaimLaunchpoolRewards claims rewards from a launchpool
func (s *DeFiEarnService) ClaimLaunchpoolRewards(ctx context.Context, poolID, userID string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lp, exists := s.launchpools[poolID]
	if !exists {
		return 0, fmt.Errorf("launchpool not found")
	}

	// Find user's stake
	var position *StakingPosition
	for i := range s.stakes[userID] {
		if s.stakes[userID][i].ProductID == poolID && s.stakes[userID][i].Status == "staked" {
			position = &s.stakes[userID][i]
			break
		}
	}

	if position == nil {
		return 0, fmt.Errorf("no stake found")
	}

	// Calculate rewards
	now := time.Now().UnixMilli()
	daysStaked := float64(now-position.StartTime) / (1000 * 60 * 60 * 24)
	rewards := position.Amount * (lp.RewardPerBlock / 10000) * daysStaked

	// Update position
	position.Claimed += rewards

	return rewards, nil
}

// ============================================================================
// Staking Functions
// ============================================================================

// CreateStakingProduct creates a new staking product
func (s *DeFiEarnService) CreateStakingProduct(ctx context.Context, sp *StakingProduct) (*StakingProduct, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sp.ID = generateID()
	sp.Status = "active"
	sp.TotalStaked = 0
	sp.StakersCount = 0
	sp.CreatedAt = time.Now().UnixMilli()

	s.staking[sp.ID] = sp
	return sp, nil
}

// GetStakingProduct retrieves a staking product
func (s *DeFiEarnService) GetStakingProduct(ctx context.Context, id string) (*StakingProduct, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sp, exists := s.staking[id]
	if !exists {
		return nil, fmt.Errorf("staking product not found")
	}
	return sp, nil
}

// ListStakingProducts lists all staking products
func (s *DeFiEarnService) ListStakingProducts(ctx context.Context, status string) ([]*StakingProduct, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*StakingProduct, 0)
	for _, sp := range s.staking {
		if status != "" && sp.Status != status {
			continue
		}
		result = append(result, sp)
	}
	return result, nil
}

// Stake stakes tokens in a product
func (s *DeFiEarnService) Stake(ctx context.Context, productID, userID string, amount float64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sp, exists := s.staking[productID]
	if !exists {
		return "", fmt.Errorf("staking product not found")
	}

	if sp.Status != "active" {
		return "", fmt.Errorf("staking product not active")
	}

	if amount < sp.MinStake {
		return "", fmt.Errorf("amount below minimum stake")
	}
	if sp.MaxStake > 0 && amount > sp.MaxStake {
		return "", fmt.Errorf("amount above maximum stake")
	}

	positionID := generateID()
	position := StakingPosition{
		ID:         positionID,
		UserID:     userID,
		ProductID:  productID,
		Amount:     amount,
		StartTime:  time.Now().UnixMilli(),
		EndTime:    time.Now().UnixMilli() + sp.LockPeriod,
		RewardDebt: 0,
		Claimed:    0,
		Status:     "staked",
	}

	s.stakes[userID] = append(s.stakes[userID], position)

	// Update product
	sp.TotalStaked += amount
	sp.StakersCount++

	return positionID, nil
}

// Unstake unstakes tokens from a product
func (s *DeFiEarnService) Unstake(ctx context.Context, productID, userID, positionID string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sp, exists := s.staking[productID]
	if !exists {
		return 0, fmt.Errorf("staking product not found")
	}

	// Find position
	var positionIndex int
	var position *StakingPosition
	for i := range s.stakes[userID] {
		if s.stakes[userID][i].ID == positionID {
			positionIndex = i
			position = &s.stakes[userID][i]
			break
		}
	}

	if position == nil {
		return 0, fmt.Errorf("position not found")
	}

	if position.Status != "staked" {
		return 0, fmt.Errorf("position already withdrawn")
	}

	// Check lock period
	now := time.Now().UnixMilli()
	if now < position.EndTime {
		// Early unstake - apply fee
		fee := position.Amount * sp.EarlyUnstakeFee / 100
		position.Amount -= fee
	}

	// Calculate pending rewards
	daysStaked := float64(now-position.StartTime) / (1000 * 60 * 60 * 24)
	rewards := position.Amount * (sp.APY / 100) * (daysStaked / 365)

	// Update position
	position.Status = "withdrawn"

	// Update product
	sp.TotalStaked -= position.Amount
	sp.StakersCount--

	return position.Amount + rewards, nil
}

// GetStakingPositions retrieves user's staking positions
func (s *DeFiEarnService) GetStakingPositions(ctx context.Context, userID string) ([]StakingPosition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stakes[userID], nil
}

// ============================================================================
// Earn Functions
// ============================================================================

// CreateEarnProduct creates a new earn product
func (s *DeFiEarnService) CreateEarnProduct(ctx context.Context, ep *EarnProduct) (*EarnProduct, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ep.ID = generateID()
	ep.Status = "active"
	ep.TotalDeposited = 0
	ep.DepositorsCount = 0

	s.earn[ep.ID] = ep
	return ep, nil
}

// Deposit deposits tokens in an earn product
func (s *DeFiEarnService) Deposit(ctx context.Context, productID, userID string, amount float64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ep, exists := s.earn[productID]
	if !exists {
		return "", fmt.Errorf("earn product not found")
	}

	if ep.Status != "active" {
		return "", fmt.Errorf("earn product not active")
	}

	if amount < ep.MinDeposit {
		return "", fmt.Errorf("amount below minimum deposit")
	}
	if ep.MaxDeposit > 0 && amount > ep.MaxDeposit {
		return "", fmt.Errorf("amount above maximum deposit")
	}

	depositID := generateID()
	deposit := EarnDeposit{
		ID:        depositID,
		UserID:    userID,
		ProductID: productID,
		Amount:    amount,
		StartTime: time.Now().UnixMilli(),
		Interest: 0,
		Status:    "active",
	}

	if !ep.Flexible && ep.TermDays > 0 {
		deposit.EndTime = time.Now().AddDate(0, 0, ep.TermDays).UnixMilli()
	}

	s.deposits[userID] = append(s.deposits[userID], deposit)

	// Update product
	ep.TotalDeposited += amount
	ep.DepositorsCount++

	return depositID, nil
}

// Withdraw withdraws from an earn product
func (s *DeFiEarnService) Withdraw(ctx context.Context, productID, userID, depositID string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ep, exists := s.earn[productID]
	if !exists {
		return 0, fmt.Errorf("earn product not found")
	}

	// Find deposit
	var depositIndex int
	var deposit *EarnDeposit
	for i := range s.deposits[userID] {
		if s.deposits[userID][i].ID == depositID {
			depositIndex = i
			deposit = &s.deposits[userID][i]
			break
		}
	}

	if deposit == nil {
		return 0, fmt.Errorf("deposit not found")
	}

	if deposit.Status != "active" {
		return 0, fmt.Errorf("deposit already withdrawn")
	}

	// Check if matured (for fixed term)
	if deposit.EndTime > 0 && time.Now().UnixMilli() < deposit.EndTime {
		return 0, fmt.Errorf("deposit not yet matured")
	}

	// Calculate interest
	days := float64(time.Now().UnixMilli()-deposit.StartTime) / (1000 * 60 * 60 * 24)
	interest := deposit.Amount * (ep.APY / 100) * (days / 365)

	totalWithdrawal := deposit.Amount + interest

	// Update deposit
	deposit.Interest = interest
	deposit.Status = "withdrawn"

	// Update product
	ep.TotalDeposited -= deposit.Amount
	ep.DepositorsCount--

	return totalWithdrawal, nil
}

// ============================================================================
// ETF Functions
// ============================================================================

// CreateETF creates a new ETF product
func (s *DeFiEarnService) CreateETF(ctx context.Context, etf *ETFAvailable) (*ETFAvailable, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	etf.ID = generateID()
	etf.Status = "active"

	s.etf[etf.ID] = etf
	return etf, nil
}

// GetETF retrieves an ETF
func (s *DeFiEarnService) GetETF(ctx context.Context, id string) (*ETFAvailable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	etf, exists := s.etf[id]
	if !exists {
		return nil, fmt.Errorf("ETF not found")
	}
	return etf, nil
}

// ListETFs lists all ETFs
func (s *DeFiEarnService) ListETFs(ctx context.Context) ([]*ETFAvailable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*ETFAvailable, 0)
	for _, etf := range s.etf {
		if etf.Status == "active" {
			result = append(result, etf)
		}
	}
	return result, nil
}

// ============================================================================
// Coupon Functions
// ============================================================================

// CreateCoupon creates a new coupon
func (s *DeFiEarnService) CreateCoupon(ctx context.Context, coupon *Coupon) (*Coupon, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	coupon.ID = generateID()
	coupon.Status = "active"
	coupon.UsedCount = 0
	coupon.CreatedAt = time.Now().UnixMilli()

	s.coupons[coupon.Code] = coupon
	return coupon, nil
}

// ValidateCoupon validates a coupon code
func (s *DeFiEarnService) ValidateCoupon(ctx context.Context, code, userID string, amount float64) (*Coupon, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	coupon, exists := s.coupons[code]
	if !exists {
		return nil, fmt.Errorf("coupon not found")
	}

	if coupon.Status != "active" {
		return nil, fmt.Errorf("coupon not active")
	}

	now := time.Now().UnixMilli()
	if now < coupon.ValidFrom || now > coupon.ValidUntil {
		return nil, fmt.Errorf("coupon expired")
	}

	if coupon.MaxUses > 0 && coupon.UsedCount >= coupon.MaxUses {
		return nil, fmt.Errorf("coupon usage limit reached")
	}

	if coupon.MinTransaction > 0 && amount < coupon.MinTransaction {
		return nil, fmt.Errorf("amount below minimum transaction for coupon")
	}

	if len(coupon.EligibleUsers) > 0 {
		eligible := false
		for _, u := range coupon.EligibleUsers {
			if u == userID {
				eligible = true
				break
			}
		}
		if !eligible {
			return nil, fmt.Errorf("user not eligible for this coupon")
		}
	}

	return coupon, nil
}

// UseCoupon marks a coupon as used
func (s *DeFiEarnService) UseCoupon(ctx context.Context, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	coupon, exists := s.coupons[code]
	if !exists {
		return fmt.Errorf("coupon not found")
	}

	coupon.UsedCount++

	if coupon.MaxUses > 0 && coupon.UsedCount >= coupon.MaxUses {
		coupon.Status = "expired"
	}

	return nil
}

// ============================================================================
// Red Packet Functions
// ============================================================================

// CreateRedPacket creates a new red packet
func (s *DeFiEarnService) CreateRedPacket(ctx context.Context, rp *RedPacket) (*RedPacket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rp.ID = generateID()
	rp.Status = "active"
	rp.RemainingAmount = rp.TotalAmount
	rp.RemainingCount = rp.Count
	rp.Claims = make([]RedPacketClaim, 0)
	rp.CreatedAt = time.Now().UnixMilli()

	s.redPackets[rp.ID] = rp
	return rp, nil
}

// ClaimRedPacket claims from a red packet
func (s *DeFiEarnService) ClaimRedPacket(ctx context.Context, packetID, userID string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rp, exists := s.redPackets[packetID]
	if !exists {
		return 0, fmt.Errorf("red packet not found")
	}

	if rp.Status != "active" {
		return 0, fmt.Errorf("red packet not active")
	}

	if time.Now().UnixMilli() > rp.ExpiresAt {
		rp.Status = "expired"
		return 0, fmt.Errorf("red packet expired")
	}

	if rp.RemainingCount <= 0 {
		return 0, fmt.Errorf("no more claims available")
	}

	// Check if user already claimed
	for _, claim := range rp.Claims {
		if claim.UserID == userID {
			return 0, fmt.Errorf("already claimed")
		}
	}

	// Calculate claim amount
	var claimAmount float64
	if rp.PacketType == "equal" {
		claimAmount = rp.TotalAmount / float64(rp.Count)
	} else {
		// Random distribution
		// Simplified: use remaining average
		claimAmount = rp.RemainingAmount / float64(rp.RemainingCount)
	}

	// Update red packet
	rp.RemainingAmount -= claimAmount
	rp.RemainingCount--

	claim := RedPacketClaim{
		ClaimID:   generateID(),
		UserID:    userID,
		Amount:    claimAmount,
		ClaimedAt: time.Now().UnixMilli(),
	}
	rp.Claims = append(rp.Claims, claim)

	if rp.RemainingCount <= 0 {
		rp.Status = "claimed"
	}

	return claimAmount, nil
}

// GetRedPacket retrieves a red packet
func (s *DeFiEarnService) GetRedPacket(ctx context.Context, id string) (*RedPacket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rp, exists := s.redPackets[id]
	if !exists {
		return nil, fmt.Errorf("red packet not found")
	}
	return rp, nil
}

// ============================================================================
// Internal Transfer Functions
// ============================================================================

// CreateInternalTransfer creates an internal transfer
func (s *DeFiEarnService) CreateInternalTransfer(ctx context.Context, transfer *InternalTransfer) (*InternalTransfer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	transfer.ID = generateID()
	transfer.Status = "pending"
	transfer.CreatedAt = time.Now().UnixMilli()

	// Calculate fee (0.1% for internal transfers)
	transfer.Fee = transfer.Amount * 0.001

	s.transfers[transfer.ID] = transfer
	return transfer, nil
}

// ExecuteInternalTransfer executes an internal transfer
func (s *DeFiEarnService) ExecuteInternalTransfer(ctx context.Context, transferID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	transfer, exists := s.transfers[transferID]
	if !exists {
		return fmt.Errorf("transfer not found")
	}

	if transfer.Status != "pending" {
		return fmt.Errorf("transfer already processed")
	}

	transfer.Status = "failed"
	transfer.CompletedAt = time.Now().UnixMilli()
	return fmt.Errorf("transaction broadcast not implemented - cannot generate tx hash without broadcasting")
}

// GetTransfer retrieves a transfer
func (s *DeFiEarnService) GetTransfer(ctx context.Context, id string) (*InternalTransfer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	transfer, exists := s.transfers[id]
	if !exists {
		return nil, fmt.Errorf("transfer not found")
	}
	return transfer, nil
}

// GetUserTransfers retrieves user's transfers
func (s *DeFiEarnService) GetUserTransfers(ctx context.Context, userID string) ([]*InternalTransfer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*InternalTransfer, 0)
	for _, t := range s.transfers {
		if t.FromUserID == userID || t.ToUserID == userID {
			result = append(result, t)
		}
	}
	return result, nil
}

// ============================================================================
// Convert Functions
// ============================================================================

// ConvertToken converts tokens at current rate
func (s *DeFiEarnService) ConvertToken(ctx context.Context, fromToken, toToken string, amount float64, userID string) (float64, error) {
	// In production, this would call price oracle for rates
	// Simplified: 1:1 conversion for demo
	conversionRate := 1.0 // Get from oracle
	convertedAmount := amount * conversionRate

	// Create internal transfer
	transfer := &InternalTransfer{
		FromUserID: userID,
		ToUserID:   "system", // or another user for P2P
		Token:      fromToken,
		Amount:     amount,
	}

	_, err := s.CreateInternalTransfer(ctx, transfer)
	if err != nil {
		return 0, err
	}

	return convertedAmount, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateID() string {
	return fmt.Sprintf("id_%d_%s", time.Now().UnixNano(), randomString(12))
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}

// ============================================================================
// Serialization
// ============================================================================

func (lp *Launchpad) Serialize() (string, error) {
	data, err := json.Marshal(lp)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func DeserializeLaunchpad(data string) (*Launchpad, error) {
	decoded, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	var lp Launchpad
	if err := json.Unmarshal(decoded, &lp); err != nil {
		return nil, err
	}
	return &lp, nil
}

// CalculateAPY calculates APY from APR
func CalculateAPY(apr float64, compoundingPeriods int) float64 {
	principal := big.NewFloat(1.0)
	rate := big.NewFloat(apr / 100)
	periods := big.NewFloat(float64(compoundingPeriods))
	
	result := new(big.Float).Add(principal, rate)
	result = new(big.Float).Exp(result, periods)
	result.Sub(result, principal)
	
	apy, _ := result.Float64()
	return apy * 100
}
