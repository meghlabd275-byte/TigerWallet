/**
 * TigerWallet Staking Service
 * 
 * Complete staking service with liquid staking, validator management,
 * and reward distribution.
 * Built with Go for high-load distributed operations.
 */

package staking

import (
	"context"
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

// StakingPool represents a staking pool
type StakingPool struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	ChainID           uint64            `json:"chain_id"`
	TokenAddress      string            `json:"token_address"`
	RewardTokenAddress string           `json:"reward_token_address"`
	StakingToken      *TokenInfo        `json:"staking_token"`
	RewardToken       *TokenInfo        `json:"reward_token"`
	TotalStaked      string            `json:"total_staked"`
	TotalRewards     string            `json:"total_rewards"`
	RewardPerSecond  string            `json:"reward_per_second"`
	MinStakeAmount   string            `json:"min_stake_amount"`
	MaxStakeAmount   string            `json:"max_stake_amount"`
	LockPeriod       int64             `json:"lock_period"` // in seconds
	UnbondingPeriod  int64             `json:"unbonding_period"`
	Status           PoolStatus         `json:"status"`
	APY              string             `json:"apy"`
	TVL              string             `json:"tvl"`
	Creator          string             `json:"creator"`
	CreatedAt        int64              `json:"created_at"`
	UpdatedAt        int64              `json:"updated_at"`
}

// TokenInfo represents token information
type TokenInfo struct {
	Address   string `json:"address"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	Decimals  int    `json:"decimals"`
	LogoURL   string `json:"logo_url"`
}

// PoolStatus represents pool status
type PoolStatus string

const (
	PoolStatusActive    PoolStatus = "active"
	PoolStatusPaused   PoolStatus = "paused"
	PoolStatusStopped  PoolStatus = "stopped"
	PoolStatusRetired  PoolStatus = "retired"
)

// Validator represents a validator
type Validator struct {
	ID          string          `json:"id"`
	PoolID      string          `json:"pool_id"`
	Address     string          `json:"address"`
	Name        string          `json:"name"`
	Commission  string          `json:"commission"`
	StakedAmount string        `json:"staked_amount"`
	Rewards     string          `json:"rewards"`
	Status      ValidatorStatus `json:"status"`
	Delegators  int            `json:"delegators"`
	Uptime      string         `json:"uptime"`
	CreatedAt   int64          `json:"created_at"`
}

// ValidatorStatus represents validator status
type ValidatorStatus string

const (
	ValidatorStatusActive   ValidatorStatus = "active"
	ValidatorStatusInactive ValidatorStatus = "inactive"
	ValidatorStatusJailed  ValidatorStatus = "jailed"
)

// UserStake represents a user's stake
type UserStake struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	PoolID         string    `json:"pool_id"`
	ValidatorID    string    `json:"validator_id"`
	Amount         string    `json:"amount"`
	PendingRewards string    `json:"pending_rewards"`
	ClaimedRewards string    `json:"claimed_rewards"`
	StakeTime      int64     `json:"stake_time"`
	UnlockTime     int64     `json:"unlock_time"`
	Status         string    `json:"status"` // staked, unbonding, claimed
}

// UnbondingRequest represents an unbonding request
type UnbondingRequest struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	PoolID         string    `json:"pool_id"`
	Amount         string    `json:"amount"`
	CompleteTime   int64     `json:"complete_time"`
	Status         string    `json:"status"` // pending, completed
	CreatedAt      int64     `json:"created_at"`
}

// Delegation represents validator delegation
type Delegation struct {
	ID           string `json:"id"`
	ValidatorID  string `json:"validator_id"`
	UserID       string `json:"user_id"`
	Amount       string `json:"amount"`
	Reward       string `json:"reward"`
	Commission   string `json:"commission"`
	CreatedAt    int64  `json:"created_at"`
}

// StakingService manages staking operations
type StakingService struct {
	mu              sync.RWMutex
	pools           map[string]*StakingPool
	validators      map[string]*Validator
	userStakes      map[string]*UserStake
	unbondingQueue  map[string]*UnbondingRequest
	delegations     map[string]*Delegation
}

// ============================================================================
// Service Methods
// ============================================================================

var (
	stakingService     *StakingService
	stakingServiceOnce sync.Once
)

// GetStakingService returns the singleton staking service
func GetStakingService() *StakingService {
	stakingServiceOnce.Do(func() {
		stakingService = &StakingService{
			pools:          make(map[string]*StakingPool),
			validators:     make(map[string]*Validator),
			userStakes:     make(map[string]*UserStake),
			unbondingQueue: make(map[string]*UnbondingRequest),
			delegations:    make(map[string]*Delegation),
		}
	})
	return stakingService
}

// ============================================================================
// Pool Operations
// ============================================================================

// CreatePool creates a new staking pool
func (s *StakingService) CreatePool(ctx context.Context, pool *StakingPool) (*StakingPool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool.ID = "pool_" + uuid.New().String()
	pool.Status = PoolStatusActive
	pool.TotalStaked = "0"
	pool.TotalRewards = "0"
	pool.CreatedAt = time.Now().Unix()
	pool.UpdatedAt = time.Now().Unix()

	s.pools[pool.ID] = pool
	return pool, nil
}

// GetPool returns a pool by ID
func (s *StakingService) GetPool(ctx context.Context, poolID string) (*StakingPool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, exists := s.pools[poolID]
	if !exists {
		return nil, fmt.Errorf("pool not found")
	}
	return pool, nil
}

// GetAllPools returns all pools
func (s *StakingService) GetAllPools(ctx context.Context, status string) ([]*StakingPool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*StakingPool, 0)
	for _, pool := range s.pools {
		if status == "" || string(pool.Status) == status {
			result = append(result, pool)
		}
	}
	return result, nil
}

// GetPoolsByChain returns pools for a specific chain
func (s *StakingService) GetPoolsByChain(ctx context.Context, chainID uint64) ([]*StakingPool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*StakingPool, 0)
	for _, pool := range s.pools {
		if pool.ChainID == chainID {
			result = append(result, pool)
		}
	}
	return result, nil
}

// UpdatePool updates a pool
func (s *StakingService) UpdatePool(ctx context.Context, pool *StakingPool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existingPool, exists := s.pools[pool.ID]
	if !exists {
		return fmt.Errorf("pool not found")
	}

	existingPool.Name = pool.Name
	existingPool.RewardPerSecond = pool.RewardPerSecond
	existingPool.MinStakeAmount = pool.MinStakeAmount
	existingPool.MaxStakeAmount = pool.MaxStakeAmount
	existingPool.LockPeriod = pool.LockPeriod
	existingPool.UpdatedAt = time.Now().Unix()

	return nil
}

// UpdatePoolStatus updates pool status
func (s *StakingService) UpdatePoolStatus(ctx context.Context, poolID string, status PoolStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, exists := s.pools[poolID]
	if !exists {
		return fmt.Errorf("pool not found")
	}

	pool.Status = status
	pool.UpdatedAt = time.Now().Unix()
	return nil
}

// DeletePool deletes a pool
func (s *StakingService) DeletePool(ctx context.Context, poolID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pools[poolID]; !exists {
		return fmt.Errorf("pool not found")
	}

	delete(s.pools, poolID)
	return nil
}

// ============================================================================
// Validator Operations
// ============================================================================

// AddValidator adds a validator to a pool
func (s *StakingService) AddValidator(ctx context.Context, validator *Validator) (*Validator, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify pool exists
	if _, exists := s.pools[validator.PoolID]; !exists {
		return nil, fmt.Errorf("pool not found")
	}

	validator.ID = "validator_" + uuid.New().String()
	validator.Status = ValidatorStatusActive
	validator.StakedAmount = "0"
	validator.Rewards = "0"
	validator.Uptime = "100"
	validator.CreatedAt = time.Now().Unix()

	s.validators[validator.ID] = validator
	return validator, nil
}

// GetValidator returns a validator
func (s *StakingService) GetValidator(ctx context.Context, validatorID string) (*Validator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	validator, exists := s.validators[validatorID]
	if !exists {
		return nil, fmt.Errorf("validator not found")
	}
	return validator, nil
}

// GetValidatorsByPool returns validators for a pool
func (s *StakingService) GetValidatorsByPool(ctx context.Context, poolID string) ([]*Validator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Validator, 0)
	for _, validator := range s.validators {
		if validator.PoolID == poolID {
			result = append(result, validator)
		}
	}
	return result, nil
}

// UpdateValidator updates a validator
func (s *StakingService) UpdateValidator(ctx context.Context, validator *Validator) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.validators[validator.ID]
	if !exists {
		return fmt.Errorf("validator not found")
	}

	existing.Name = validator.Name
	existing.Commission = validator.Commission
	existing.Status = validator.Status

	return nil
}

// RemoveValidator removes a validator
func (s *StakingService) RemoveValidator(ctx context.Context, validatorID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.validators[validatorID]; !exists {
		return fmt.Errorf("validator not found")
	}

	delete(s.validators, validatorID)
	return nil
}

// ============================================================================
// Staking Operations
// ============================================================================

// Stake creates a stake
func (s *StakingService) Stake(ctx context.Context, stake *UserStake) (*UserStake, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify pool exists
	pool, poolExists := s.pools[stake.PoolID]
	if !poolExists {
		return nil, fmt.Errorf("pool not found")
	}

	// Verify validator exists
	if stake.ValidatorID != "" {
		if _, exists := s.validators[stake.ValidatorID]; !exists {
			return nil, fmt.Errorf("validator not found")
		}
	}

	// Validate amount
	amount, err := new(big.Int).SetString(stake.Amount, 10)
	if err != nil {
		return nil, fmt.Errorf("invalid amount")
	}

	minStake, _ := new(big.Int).SetString(pool.MinStakeAmount, 10)
	maxStake, _ := new(big.Int).SetString(pool.MaxStakeAmount, 10)

	if amount.Cmp(minStake) < 0 {
		return nil, fmt.Errorf("amount below minimum stake")
	}

	if maxStake.Cmp(big.NewInt(0)) > 0 && amount.Cmp(maxStake) > 0 {
		return nil, fmt.Errorf("amount exceeds maximum stake")
	}

	// Create stake
	stake.ID = "stake_" + uuid.New().String()
	stake.PendingRewards = "0"
	stake.ClaimedRewards = "0"
	stake.StakeTime = time.Now().Unix()
	stake.Status = "staked"

	// Calculate unlock time if lock period exists
	if pool.LockPeriod > 0 {
		stake.UnlockTime = stake.StakeTime + pool.LockPeriod
	}

	s.userStakes[stake.ID] = stake

	// Update pool total staked
	totalStaked, _ := new(big.Int).SetString(pool.TotalStaked, 10)
	totalStaked.Add(totalStaked, amount)
	pool.TotalStaked = totalStaked.String()

	// Update validator staked amount
	if stake.ValidatorID != "" {
		validator, _ := s.validators[stake.ValidatorID]
		if validator != nil {
			valStaked, _ := new(big.Int).SetString(validator.StakedAmount, 10)
			valStaked.Add(valStaked, amount)
			validator.StakedAmount = valStaked.String()
		}
	}

	return stake, nil
}

// GetStake returns a stake
func (s *StakingService) GetStake(ctx context.Context, stakeID string) (*UserStake, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stake, exists := s.userStakes[stakeID]
	if !exists {
		return nil, fmt.Errorf("stake not found")
	}
	return stake, nil
}

// GetUserStakes returns all stakes for a user
func (s *StakingService) GetUserStakes(ctx context.Context, userID string) ([]*UserStake, error) {
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

// GetUserStakesByPool returns all stakes for a user in a pool
func (s *StakingService) GetUserStakesByPool(ctx context.Context, userID, poolID string) ([]*UserStake, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*UserStake, 0)
	for _, stake := range s.userStakes {
		if stake.UserID == userID && stake.PoolID == poolID {
			result = append(result, stake)
		}
	}
	return result, nil
}

// ============================================================================
// Unbonding Operations
// ============================================================================

// RequestUnbonding requests unbonding
func (s *StakingService) RequestUnbonding(ctx context.Context, userID, poolID, amount string) (*UnbondingRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find user's stake in pool
	var stake *UserStake
	for _, s := range s.userStakes {
		if s.UserID == userID && s.PoolID == poolID && s.Status == "staked" {
			stake = s
			break
		}
	}

	if stake == nil {
		return nil, fmt.Errorf("no active stake found")
	}

	// Validate amount
	stakedAmount, _ := new(big.Int).SetString(stake.Amount, 10)
	requestAmount, err := new(big.Int).SetString(amount, 10)
	if err != nil {
		return nil, fmt.Errorf("invalid amount")
	}

	if requestAmount.Cmp(stakedAmount) > 0 {
		return nil, fmt.Errorf("amount exceeds staked amount")
	}

	// Get pool unbonding period
	pool, _ := s.pools[poolID]
	unbondingPeriod := int64(86400) // default 24 hours
	if pool != nil && pool.UnbondingPeriod > 0 {
		unbondingPeriod = pool.UnbondingPeriod
	}

	// Create unbonding request
	request := &UnbondingRequest{
		ID:           "unbonding_" + uuid.New().String(),
		UserID:       userID,
		PoolID:       poolID,
		Amount:       amount,
		CompleteTime:  time.Now().Unix() + unbondingPeriod,
		Status:       "pending",
		CreatedAt:    time.Now().Unix(),
	}

	s.unbondingQueue[request.ID] = request

	// Update stake
	if requestAmount.Cmp(stakedAmount) == 0 {
		stake.Status = "unbonding"
	} else {
		// Partial unbonding
		stakedAmount.Sub(stakedAmount, requestAmount)
		stake.Amount = stakedAmount.String()
	}

	// Update pool total
	if pool != nil {
		totalStaked, _ := new(big.Int).SetString(pool.TotalStaked, 10)
		totalStaked.Sub(totalStaked, requestAmount)
		pool.TotalStaked = totalStaked.String()
	}

	return request, nil
}

// ClaimUnbonding claims unbonded tokens
func (s *StakingService) ClaimUnbonding(ctx context.Context, requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, exists := s.unbondingQueue[requestID]
	if !exists {
		return fmt.Errorf("unbonding request not found")
	}

	if request.Status == "completed" {
		return fmt.Errorf("already claimed")
	}

	if time.Now().Unix() < request.CompleteTime {
		return fmt.Errorf("unbonding period not complete")
	}

	request.Status = "completed"
	return nil
}

// GetUserUnbondingRequests returns unbonding requests for a user
func (s *StakingService) GetUserUnbondingRequests(ctx context.Context, userID string) ([]*UnbondingRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*UnbondingRequest, 0)
	for _, request := range s.unbondingQueue {
		if request.UserID == userID {
			result = append(result, request)
		}
	}
	return result, nil
}

// ============================================================================
// Reward Operations
// ============================================================================

// CalculateRewards calculates pending rewards for a stake
func (s *StakingService) CalculateRewards(ctx context.Context, stakeID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stake, exists := s.userStakes[stakeID]
	if !exists {
		return "0", fmt.Errorf("stake not found")
	}

	pool, exists := s.pools[stake.PoolID]
	if !exists {
		return "0", fmt.Errorf("pool not found")
	}

	// Calculate rewards: amount * time * reward_rate
	staked, _ := new(big.Int).SetString(stake.Amount, 10)
	rewardPerSecond, _ := new(big.Int).SetString(pool.RewardPerSecond, 10)

	// Time staked in seconds
	timeStaked := time.Now().Unix() - stake.StakeTime

	// Rewards = staked * time * reward_per_second / 1e18
	rewards := new(big.Int).Mul(staked, big.NewInt(timeStaked))
	rewards.Mul(rewards, rewardPerSecond)
	rewards.Div(rewards, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	return rewards.String(), nil
}

// ClaimRewards claims rewards for a stake
func (s *StakingService) ClaimRewards(ctx context.Context, stakeID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stake, exists := s.userStakes[stakeID]
	if !exists {
		return "0", fmt.Errorf("stake not found")
	}

	// Calculate pending rewards
	rewards, err := s.calculateRewardsInternal(stake)
	if err != nil {
		return "0", err
	}

	// Update stake
	claimed, _ := new(big.Int).SetString(stake.ClaimedRewards, 10)
	rewardAmount, _ := new(big.Int).SetString(rewards, 10)
	claimed.Add(claimed, rewardAmount)
	stake.ClaimedRewards = claimed.String()
	stake.PendingRewards = "0"
	stake.LastClaimTime = time.Now().Unix()

	return rewards, nil
}

// calculateRewardsInternal calculates rewards (must be called with lock)
func (s *StakingService) calculateRewardsInternal(stake *UserStake) (string, error) {
	pool, exists := s.pools[stake.PoolID]
	if !exists {
		return "0", fmt.Errorf("pool not found")
	}

	staked, _ := new(big.Int).SetString(stake.Amount, 10)
	rewardPerSecond, _ := new(big.Int).SetString(pool.RewardPerSecond, 10)

	// Use last claim time if available, otherwise use stake time
	lastClaimTime := stake.LastClaimTime
	if lastClaimTime == 0 {
		lastClaimTime = stake.StakeTime
	}

	timeStaked := time.Now().Unix() - lastClaimTime
	if timeStaked <= 0 {
		return "0", nil
	}

	rewards := new(big.Int).Mul(staked, big.NewInt(timeStaked))
	rewards.Mul(rewards, rewardPerSecond)
	rewards.Div(rewards, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	return rewards.String(), nil
}

// GetUserTotalRewards returns total rewards for a user
func (s *StakingService) GetUserTotalRewards(ctx context.Context, userID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalRewards := big.NewInt(0)

	for _, stake := range s.userStakes {
		if stake.UserID == userID {
			// Get pending rewards
			pool, exists := s.pools[stake.PoolID]
			if exists {
				staked, _ := new(big.Int).SetString(stake.Amount, 10)
				rewardPerSecond, _ := new(big.Int).SetString(pool.RewardPerSecond, 10)
				lastClaimTime := stake.LastClaimTime
				if lastClaimTime == 0 {
					lastClaimTime = stake.StakeTime
				}
				timeStaked := time.Now().Unix() - lastClaimTime
				if timeStaked > 0 {
					rewards := new(big.Int).Mul(staked, big.NewInt(timeStaked))
					rewards.Mul(rewards, rewardPerSecond)
					rewards.Div(rewards, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
					totalRewards.Add(totalRewards, rewards)
				}
			}

			// Add claimed rewards
			claimed, _ := new(big.Int).SetString(stake.ClaimedRewards, 10)
			totalRewards.Add(totalRewards, claimed)
		}
	}

	return totalRewards.String(), nil
}

// ============================================================================
// Utility Methods
// ============================================================================

// CalculateAPY calculates APY for a pool
func (s *StakingService) CalculateAPY(ctx context.Context, poolID string) (string, error) {
	s.mu.RLock()
	pool, exists := s.pools[poolID]
	s.mu.RUnlock()

	if !exists {
		return "0", fmt.Errorf("pool not found")
	}

	rewardPerSecond, _ := new(big.Int).SetString(pool.RewardPerSecond, 10)
	totalStaked, _ := new(big.Int).SetString(pool.TotalStaked, 10)

	if totalStaked.Cmp(big.NewInt(0)) == 0 {
		return "0", nil
	}

	// APY = (reward_per_second * 31536000 / total_staked) * 100
	secondsPerYear := big.NewInt(31536000)
	yearlyRewards := new(big.Int).Mul(rewardPerSecond, secondsPerYear)

	apy := new(big.Int).Div(yearlyRewards, totalStaked)
	apy.Mul(apy, big.NewInt(100))

	return apy.String(), nil
}

// ToJSON converts stake to JSON
func (st *UserStake) ToJSON() (string, error) {
	data, err := json.Marshal(st)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
