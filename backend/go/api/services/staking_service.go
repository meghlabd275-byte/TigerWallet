package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"tigerwallet/backend/go/api/models"
)

type StakingService struct {
	mu      sync.RWMutex
	pools   map[string]*models.StakingPool
	positions map[string]*models.StakingPosition
}

var (
	stakingInstance *StakingService
	stakingOnce    sync.Once
)

func NewStakingService() *StakingService {
	stakingOnce.Do(func() {
		stakingInstance = &StakingService{
			pools:     make(map[string]*models.StakingPool),
			positions: make(map[string]*models.StakingPosition),
		}
		stakingInstance.initializeDefaultPools()
	})
	return stakingInstance
}

func (s *StakingService) initializeDefaultPools() {
	defaultPools := []*models.StakingPool{
		{ID: "eth-eth2", BlockchainID: "ethereum", TokenSymbol: "ETH", Name: "Ethereum 2.0 Staking", Description: "Stake ETH and earn rewards", MinStake: "0.01", MaxStake: "10000", APY: 4.5, LockPeriod: 86400 * 365, IsActive: true},
		{ID: "sol-solana", BlockchainID: "solana", TokenSymbol: "SOL", Name: "Solana Stake", Description: "Stake SOL and earn ~7% APY", MinStake: "1", MaxStake: "100000", APY: 7.2, LockPeriod: 0, IsActive: true},
		{ID: "dot-polkadot", BlockchainID: "polkadot", TokenSymbol: "DOT", Name: "Polkadot Nominator", Description: "Stake DOT and earn ~12% APY", MinStake: "10", MaxStake: "10000", APY: 12.5, LockPeriod: 86400 * 28, IsActive: true},
		{ID: "atom-cosmos", BlockchainID: "cosmos", TokenSymbol: "ATOM", Name: "Cosmos Delegation", Description: "Stake ATOM and earn ~20% APY", MinStake: "1", MaxStake: "10000", APY: 20.0, LockPeriod: 0, IsActive: true},
		{ID: "near-near", BlockchainID: "near", TokenSymbol: "NEAR", Name: "NEAR Staking", Description: "Stake NEAR and earn ~10% APY", MinStake: "1", MaxStake: "100000", APY: 10.5, LockPeriod: 0, IsActive: true},
		{ID: "apt-aptos", BlockchainID: "aptos", TokenSymbol: "APT", Name: "Aptos Stake", Description: "Stake APT and earn ~8% APY", MinStake: "1", MaxStake: "10000", APY: 8.0, LockPeriod: 0, IsActive: true},
		{ID: "bnb-bsc", BlockchainID: "bsc", TokenSymbol: "BNB", Name: "BNB Staking", Description: "Stake BNB and earn ~15% APY", MinStake: "0.1", MaxStake: "1000", APY: 15.0, LockPeriod: 86400 * 7, IsActive: true},
		{ID: "matic-polygon", BlockchainID: "polygon", TokenSymbol: "MATIC", Name: "Polygon Staking", Description: "Stake MATIC and earn ~5% APY", MinStake: "10", MaxStake: "100000", APY: 5.2, LockPeriod: 0, IsActive: true},
		{ID: "avax-avalanche", BlockchainID: "avalanche", TokenSymbol: "AVAX", Name: "Avalanche Stake", Description: "Stake AVAX and earn ~9% APY", MinStake: "25", MaxStake: "10000", APY: 9.5, LockPeriod: 86400 * 14, IsActive: true},
		{ID: "trx-tron", BlockchainID: "tron", TokenSymbol: "TRX", Name: "Tron Energy", Description: "Stake TRX and earn ~4% APY", MinStake: "100", MaxStake: "10000000", APY: 4.5, LockPeriod: 0, IsActive: true},
	}

	for _, pool := range defaultPools {
		s.pools[pool.ID] = pool
	}
}

func (s *StakingService) GetPools(ctx context.Context, blockchainID string) ([]*models.StakingPool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.StakingPool
	for _, pool := range s.pools {
		if pool.IsActive {
			if blockchainID == "" || pool.BlockchainID == blockchainID {
				result = append(result, pool)
			}
		}
	}

	return result, nil
}

func (s *StakingService) GetPoolByID(ctx context.Context, poolID string) (*models.StakingPool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok || !pool.IsActive {
		return nil, errors.New("pool not found")
	}

	return pool, nil
}

func (s *StakingService) Stake(ctx context.Context, walletID, poolID, amount string) (*models.StakingPosition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok || !pool.IsActive {
		return nil, errors.New("pool not found")
	}

	positionID := fmt.Sprintf("position_%d", time.Now().UnixNano())

	position := &models.StakingPosition{
		ID:              positionID,
		WalletID:        walletID,
		PoolID:          poolID,
		BlockchainID:    pool.BlockchainID,
		TokenSymbol:     pool.TokenSymbol,
		Amount:          amount,
		RewardAmount:    "0",
		RewardClaimed:   "0",
		APY:            pool.APY,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	s.positions[positionID] = position

	return position, nil
}

func (s *StakingService) Unstake(ctx context.Context, positionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	position, ok := s.positions[positionID]
	if !ok {
		return errors.New("position not found")
	}

	position.Status = "unbonding"
	position.UpdatedAt = time.Now()

	return nil
}

func (s *StakingService) ClaimRewards(ctx context.Context, positionID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	position, ok := s.positions[positionID]
	if !ok {
		return "", errors.New("position not found")
	}

	// Calculate pending rewards (simplified)
	rewards := position.Amount // Would calculate based on APY and time

	position.RewardClaimed = rewards
	position.UpdatedAt = time.Now()

	return rewards, nil
}

func (s *StakingService) GetPositions(ctx context.Context, walletID string) ([]*models.StakingPosition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.StakingPosition
	for _, position := range s.positions {
		if position.WalletID == walletID {
			result = append(result, position)
		}
	}

	return result, nil
}
