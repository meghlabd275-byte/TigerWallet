/**
 * TigerWallet Launchpad Service
 * 
 * Complete IDO/ICO launchpad functionality for token launches.
 * Built with Go for high-load distributed operations.
 */

package launchpad

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Types
// ============================================================================

// Launchpad represents a token launch
type Launchpad struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	TokenName           string             `json:"token_name"`
	TokenSymbol         string             `json:"token_symbol"`
	TokenAddress        string             `json:"token_address"`
	TokenDecimals       int                `json:"token_decimals"`
	TokenLogo           string             `json:"token_logo"`
	Description         string             `json:"description"`
	Website             string             `json:"website"`
	Whitepaper          string             `json:"whitepaper"`
	SocialLinks         SocialLinks        `json:"social_links"`
	StartTime           int64              `json:"start_time"`
	EndTime             int64              `json:"end_time"`
	ClaimTime           int64              `json:"claim_time"`
	Status              LaunchpadStatus    `json:"status"`
	TokenPrice          string             `json:"token_price"`
	TotalSupply         string             `json:"total_supply"`
	HardCap             string             `json:"hard_cap"`
	SoftCap             string             `json:"soft_cap"`
	MinContribution     string             `json:"min_contribution"`
	MaxContribution     string             `json:"max_contribution"`
	AcceptedTokens      []AcceptedToken    `json:"accepted_tokens"`
	分配Allocation       Allocation           `json:"allocation"`
	TierSystem          []Tier              `json:"tier_system"`
	TotalRaised         string             `json:"total_raised"`
	TotalParticipants   int                `json:"total_participants"`
	CreatorAddress      string             `json:"creator_address"`
	AdminAddress        string             `json:"admin_address"`
	IsKYCRequired       bool               `json:"is_kyc_required"`
	IsAuditRequired     bool               `json:"is_audit_required"`
	AuditReport         string             `json:"audit_report"`
	CreatedAt           int64              `json:"created_at"`
	UpdatedAt           int64              `json:"updated_at"`
}

// SocialLinks holds social media links
type SocialLinks struct {
	Twitter    string `json:"twitter"`
	Telegram   string `json:"telegram"`
	Discord    string `json:"discord"`
	Medium     string `json:"medium"`
	Reddit     string `json:"reddit"`
	GitHub     string `json:"github"`
}

// AcceptedToken represents a token accepted for payment
type AcceptedToken struct {
	Address       string `json:"address"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	Decimals      int    `json:"decimals"`
	PriceFeed     string `json:"price_feed"`
	MinAmount     string `json:"min_amount"`
	MaxAmount     string `json:"max_amount"`
}

// Allocation represents token allocation
type Allocation struct {
	PublicSale      string `json:"public_sale"`
	PrivateSale    string `json:"private_sale"`
	Team            string `json:"team"`
	Marketing       string `json:"marketing"`
	Development     string `json:"development"`
	Rewards         string `json:"rewards"`
	Liquidity       string `json:"liquidity"`
	Community       string `json:"community"`
}

// Tier represents a participation tier
type Tier struct {
	Level           int    `json:"level"`
	Name            string `json:"name"`
	MinStaking      string `json:"min_staking"`
	Allocation      string `json:"allocation"`
	MaxAllocation   string `json:"max_allocation"`
	Discount        string `json:"discount"`
}

// LaunchpadStatus represents launchpad status
type LaunchpadStatus string

const (
	StatusUpcoming   LaunchpadStatus = "upcoming"
	StatusActive    LaunchpadStatus = "active"
	StatusCompleted LaunchpadStatus = "completed"
	StatusCancelled LaunchpadStatus = "cancelled"
	StatusFailed    LaunchpadStatus = "failed"
)

// Pool represents a launch pool
type Pool struct {
	ID              string          `json:"id"`
	LaunchpadID     string          `json:"launchpad_id"`
	Name            string          `json:"name"`
	TokenAddress    string          `json:"token_address"`
	StartTime       int64           `json:"start_time"`
	EndTime         int64           `json:"end_time"`
	ClaimTime       int64           `json:"claim_time"`
	Status          PoolStatus      `json:"status"`
	MinAllocation   string          `json:"min_allocation"`
	MaxAllocation   string          `json:"max_allocation"`
	TotalStaked     string          `json:"total_staked"`
	RewardPerBlock  string          `json:"reward_per_block"`
	TotalRewards    string          `json:"total_rewards"`
	Participants    int             `json:"participants"`
	CreatedAt       int64           `json:"created_at"`
}

// PoolStatus represents pool status
type PoolStatus string

const (
	PoolStatusActive    PoolStatus = "active"
	PoolStatusUpcoming PoolStatus = "upcoming"
	PoolStatusEnded    PoolStatus = "ended"
	PoolStatusClaimed  PoolStatus = "claimed"
)

// UserParticipation represents user participation in a launchpad
type UserParticipation struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	LaunchpadID       string    `json:"launchpad_id"`
	PoolID            string    `json:"pool_id"`
	TierLevel         int       `json:"tier_level"`
	Amount            string    `json:"amount"`
	TokenAmount        string    `json:"token_amount"`
	ClaimedAmount      string    `json:"claimed_amount"`
	Status            string    `json:"status"`
	TransactionHash   string    `json:"transaction_hash"`
	ClaimTransactionHash string `json:"claim_transaction_hash"`
	CreatedAt         int64     `json:"created_at"`
	UpdatedAt         int64     `json:"updated_at"`
}

// UserStake represents user staking position
type UserStake struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	PoolID        string    `json:"pool_id"`
	Amount        string    `json:"amount"`
	PendingReward string    `json:"pending_reward"`
	TotalEarned   string    `json:"total_earned"`
	StakeTime     int64     `json:"stake_time"`
	LastClaimTime int64     `json:"last_claim_time"`
}

// LaunchpadService manages launchpad operations
type LaunchpadService struct {
	mu            sync.RWMutex
	launchpads    map[string]*Launchpad
	pools         map[string]*Pool
	participations map[string]*UserParticipation
	userStakes    map[string]*UserStake
}

// ============================================================================
// Service Methods
// ============================================================================

var (
	launchpadService     *LaunchpadService
	launchpadServiceOnce sync.Once
)

// GetLaunchpadService returns the singleton launchpad service
func GetLaunchpadService() *LaunchpadService {
	launchpadServiceOnce.Do(func() {
		launchpadService = &LaunchpadService{
			launchpads:    make(map[string]*Launchpad),
			pools:         make(map[string]*Pool),
			participations: make(map[string]*UserParticipation),
			userStakes:    make(map[string]*UserStake),
		}
	})
	return launchpadService
}

// ============================================================================
// Launchpad Operations
// ============================================================================

// CreateLaunchpad creates a new launchpad
func (s *LaunchpadService) CreateLaunchpad(ctx context.Context, req *Launchpad) (*Launchpad, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req.ID = "launchpad_" + uuid.New().String()
	req.Status = StatusUpcoming
	req.TotalRaised = "0"
	req.TotalParticipants = 0
	req.CreatedAt = time.Now().Unix()
	req.UpdatedAt = time.Now().Unix()

	s.launchpads[req.ID] = req
	return req, nil
}

// GetLaunchpad returns a launchpad by ID
func (s *LaunchpadService) GetLaunchpad(ctx context.Context, id string) (*Launchpad, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	launchpad, exists := s.launchpads[id]
	if !exists {
		return nil, fmt.Errorf("launchpad not found")
	}
	return launchpad, nil
}

// GetAllLaunchpads returns all launchpads
func (s *LaunchpadService) GetAllLaunchpads(ctx context.Context, status string) ([]*Launchpad, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Launchpad, 0)
	for _, launchpad := range s.launchpads {
		if status == "" || string(launchpad.Status) == status {
			result = append(result, launchpad)
		}
	}
	return result, nil
}

// UpdateLaunchpadStatus updates launchpad status
func (s *LaunchpadService) UpdateLaunchpadStatus(ctx context.Context, id string, status LaunchpadStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	launchpad, exists := s.launchpads[id]
	if !exists {
		return fmt.Errorf("launchpad not found")
	}

	launchpad.Status = status
	launchpad.UpdatedAt = time.Now().Unix()
	return nil
}

// UpdateLaunchpad updates launchpad information
func (s *LaunchpadService) UpdateLaunchpad(ctx context.Context, req *Launchpad) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	launchpad, exists := s.launchpads[req.ID]
	if !exists {
		return fmt.Errorf("launchpad not found")
	}

	// Update fields
	launchpad.Name = req.Name
	launchpad.Description = req.Description
	launchpad.Website = req.Website
	launchpad.Whitepaper = req.Whitepaper
	launchpad.SocialLinks = req.SocialLinks
	launchpad.UpdatedAt = time.Now().Unix()

	return nil
}

// DeleteLaunchpad deletes a launchpad
func (s *LaunchpadService) DeleteLaunchpad(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.launchpads[id]; !exists {
		return fmt.Errorf("launchpad not found")
	}

	delete(s.launchpads, id)
	return nil
}

// ============================================================================
// Pool Operations
// ============================================================================

// CreatePool creates a new launch pool
func (s *LaunchpadService) CreatePool(ctx context.Context, req *Pool) (*Pool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req.ID = "pool_" + uuid.New().String()
	req.Status = PoolStatusUpcoming
	req.TotalStaked = "0"
	req.TotalRewards = "0"
	req.Participants = 0
	req.CreatedAt = time.Now().Unix()

	s.pools[req.ID] = req
	return req, nil
}

// GetPool returns a pool by ID
func (s *LaunchpadService) GetPool(ctx context.Context, id string) (*Pool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, exists := s.pools[id]
	if !exists {
		return nil, fmt.Errorf("pool not found")
	}
	return pool, nil
}

// GetPoolsByLaunchpad returns all pools for a launchpad
func (s *LaunchpadService) GetPoolsByLaunchpad(ctx context.Context, launchpadID string) ([]*Pool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Pool, 0)
	for _, pool := range s.pools {
		if pool.LaunchpadID == launchpadID {
			result = append(result, pool)
		}
	}
	return result, nil
}

// UpdatePool updates a pool
func (s *LaunchpadService) UpdatePool(ctx context.Context, req *Pool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, exists := s.pools[req.ID]
	if !exists {
		return fmt.Errorf("pool not found")
	}

	pool.Name = req.Name
	pool.StartTime = req.StartTime
	pool.EndTime = req.EndTime
	pool.MinAllocation = req.MinAllocation
	pool.MaxAllocation = req.MaxAllocation
	return nil
}

// DeletePool deletes a pool
func (s *LaunchpadService) DeletePool(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pools[id]; !exists {
		return fmt.Errorf("pool not found")
	}

	delete(s.pools, id)
	return nil
}

// ============================================================================
// Participation Operations
// ============================================================================

// Participate creates user participation
func (s *LaunchpadService) Participate(ctx context.Context, req *UserParticipation) (*UserParticipation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate launchpad exists and is active
	launchpad, exists := s.launchpads[req.LaunchpadID]
	if !exists {
		return nil, fmt.Errorf("launchpad not found")
	}

	if launchpad.Status != StatusActive {
		return nil, fmt.Errorf("launchpad is not active")
	}

	// Check contribution limits
	minContribution, _ := new(big.Int).SetString(launchpad.MinContribution, 10)
	maxContribution, _ := new(big.Int).SetString(launchpad.MaxContribution, 10)
	amount, _ := new(big.Int).SetString(req.Amount, 10)

	if amount.Cmp(minContribution) < 0 {
		return nil, fmt.Errorf("amount below minimum contribution")
	}

	if amount.Cmp(maxContribution) > 0 {
		return nil, fmt.Errorf("amount exceeds maximum contribution")
	}

	req.ID = "participation_" + uuid.New().String()
	req.Status = "pending"
	req.ClaimedAmount = "0"
	req.CreatedAt = time.Now().Unix()
	req.UpdatedAt = time.Now().Unix()

	s.participations[req.ID] = req

	// Update total raised
	totalRaised, _ := new(big.Int).SetString(launchpad.TotalRaised, 10)
	totalRaised.Add(totalRaised, amount)
	launchpad.TotalRaised = totalRaised.String()
	launchpad.TotalParticipants++

	return req, nil
}

// GetUserParticipation returns user participation
func (s *LaunchpadService) GetUserParticipation(ctx context.Context, userID, launchpadID string) (*UserParticipation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, participation := range s.participations {
		if participation.UserID == userID && participation.LaunchpadID == launchpadID {
			return participation, nil
		}
	}
	return nil, nil
}

// GetUserParticipations returns all participations for a user
func (s *LaunchpadService) GetUserParticipations(ctx context.Context, userID string) ([]*UserParticipation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*UserParticipation, 0)
	for _, participation := range s.participations {
		if participation.UserID == userID {
			result = append(result, participation)
		}
	}
	return result, nil
}

// ClaimTokens claims tokens for a participation
func (s *LaunchpadService) ClaimTokens(ctx context.Context, participationID, txHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	participation, exists := s.participations[participationID]
	if !exists {
		return fmt.Errorf("participation not found")
	}

	if participation.Status == "claimed" {
		return fmt.Errorf("tokens already claimed")
	}

	participation.Status = "claimed"
	participation.ClaimTransactionHash = txHash
	participation.UpdatedAt = time.Now().Unix()

	return nil
}

// ============================================================================
// Staking Operations
// ============================================================================

// Stake creates a staking position
func (s *LaunchpadService) Stake(ctx context.Context, req *UserStake) (*UserStake, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate pool exists
	pool, exists := s.pools[req.PoolID]
	if !exists {
		return nil, fmt.Errorf("pool not found")
	}

	if pool.Status != PoolStatusActive {
		return nil, fmt.Errorf("pool is not active")
	}

	// Create unique stake ID
	stakeID := fmt.Sprintf("%s_%s", req.UserID, req.PoolID)
	hash := sha256.Sum256([]byte(stakeID))
	req.ID = "stake_" + hex.EncodeToString(hash[:])[:16]

	req.PendingReward = "0"
	req.TotalEarned = "0"
	req.StakeTime = time.Now().Unix()
	req.LastClaimTime = req.StakeTime

	s.userStakes[req.ID] = req

	// Update pool total staked
	totalStaked, _ := new(big.Int).SetString(pool.TotalStaked, 10)
	amount, _ := new(big.Int).SetString(req.Amount, 10)
	totalStaked.Add(totalStaked, amount)
	pool.TotalStaked = totalStaked.String()
	pool.Participants++

	return req, nil
}

// Unstake unstakes tokens
func (s *LaunchpadService) Unstake(ctx context.Context, stakeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stake, exists := s.userStakes[stakeID]
	if !exists {
		return fmt.Errorf("stake not found")
	}

	pool, exists := s.pools[stake.PoolID]
	if !exists {
		return fmt.Errorf("pool not found")
	}

	// Update pool total staked
	totalStaked, _ := new(big.Int).SetString(pool.TotalStaked, 10)
	amount, _ := new(big.Int).SetString(stake.Amount, 10)
	totalStaked.Sub(totalStaked, amount)
	pool.TotalStaked = totalStaked.String()
	pool.Participants--

	// Claim pending rewards first
	if stake.PendingReward != "0" && stake.PendingReward != "" {
		totalEarned, _ := new(big.Int).SetString(stake.TotalEarned, 10)
		pending, _ := new(big.Int).SetString(stake.PendingReward, 10)
		totalEarned.Add(totalEarned, pending)
		stake.TotalEarned = totalEarned.String()
	}

	delete(s.userStakes, stakeID)
	return nil
}

// ClaimStakingRewards claims staking rewards
func (s *LaunchpadService) ClaimStakingRewards(ctx context.Context, stakeID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stake, exists := s.userStakes[stakeID]
	if !exists {
		return "", fmt.Errorf("stake not found")
	}

	reward := stake.PendingReward
	if reward == "0" || reward == "" {
		return "0", nil
	}

	// Update stake
	totalEarned, _ := new(big.Int).SetString(stake.TotalEarned, 10)
	pending, _ := new(big.Int).SetString(reward, 10)
	totalEarned.Add(totalEarned, pending)
	stake.TotalEarned = totalEarned.String()
	stake.PendingReward = "0"
	stake.LastClaimTime = time.Now().Unix()

	return reward, nil
}

// GetUserStake returns user stake for a pool
func (s *LaunchpadService) GetUserStake(ctx context.Context, userID, poolID string) (*UserStake, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stakeID := fmt.Sprintf("%s_%s", userID, poolID)
	hash := sha256.Sum256([]byte(stakeID))
	id := "stake_" + hex.EncodeToString(hash[:])[:16]

	stake, exists := s.userStakes[id]
	if !exists {
		return nil, nil
	}
	return stake, nil
}

// GetUserStakes returns all stakes for a user
func (s *LaunchpadService) GetUserStakes(ctx context.Context, userID string) ([]*UserStake, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*UserStake, 0)
	for _, stake := range s.userStakes {
		if stake.UserID == userID {
			result = append(result, stake)
		}
	}
	return result, nil
}

// ============================================================================
// Utility Methods
// ============================================================================

// CalculateTier calculates user tier based on staked amount
func (s *LaunchpadService) CalculateTier(ctx context.Context, stakedAmount string, tiers []Tier) (int, string) {
	amount, _ := new(big.Int).SetString(stakedAmount, 10)

	for i := len(tiers) - 1; i >= 0; i-- {
		minStaking, _ := new(big.Int).SetString(tiers[i].MinStaking, 10)
		if amount.Cmp(minStaking) >= 0 {
			return tiers[i].Level, tiers[i].Name
		}
	}

	return 0, "None"
}

// CalculateTokenAmount calculates token amount for payment
func (s *LaunchpadService) CalculateTokenAmount(ctx context.Context, paymentAmount, tokenPrice string) (string, error) {
	payment, err := new(big.Int).SetString(paymentAmount, 10)
	if err != nil {
		return "", err
	}

	price, err := new(big.Int).SetString(tokenPrice, 10)
	if err != nil {
		return "", err
	}

	// Token amount = payment / price * 10^decimals
	decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	result := new(big.Int).Div(payment, price)
	result.Mul(result, decimals)

	return result.String(), nil
}

// ValidateTiers validates tier configuration
func (s *LaunchpadService) ValidateTiers(ctx context.Context, tiers []Tier) error {
	if len(tiers) == 0 {
		return fmt.Errorf("at least one tier is required")
	}

	for i, tier := range tiers {
		if tier.Name == "" {
			return fmt.Errorf("tier %d: name is required", i)
		}
		if tier.MinStaking == "" {
			return fmt.Errorf("tier %d: min staking is required", i)
		}
		if tier.Allocation == "" {
			return fmt.Errorf("tier %d: allocation is required", i)
		}
	}

	return nil
}

// ToJSON converts launchpad to JSON
func (l *Launchpad) ToJSON() (string, error) {
	data, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
