package claim_service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

type ClaimType string
type ClaimStatus string

const (
	ClaimTypeAirdrop  ClaimType = "airdrop"
	ClaimTypeBonus    ClaimType = "bonus"
	ClaimTypeReward   ClaimType = "reward"
	ClaimTypeRebate   ClaimType = "rebate"
	ClaimTypeCashback ClaimType = "cashback"

	ClaimStatusPending  ClaimStatus = "pending"
	ClaimStatusApproved ClaimStatus = "approved"
	ClaimStatusRejected ClaimStatus = "rejected"
	ClaimStatusClaimed ClaimStatus = "claimed"
)

type ClaimableReward struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Type        ClaimType  `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Amount      float64    `json:"amount"`
	Token       string     `json:"token"`
	Status      ClaimStatus `json:"status"`
	Source      string     `json:"source"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	CreateTime  time.Time  `json:"createTime"`
	ClaimTime   *time.Time `json:"claimTime,omitempty"`
	TxHash      string     `json:"txHash,omitempty"`
}

type ClaimHistory struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	Type       ClaimType `json:"type"`
	Title      string    `json:"title"`
	Amount     float64   `json:"amount"`
	Token      string    `json:"token"`
	ClaimedAt  time.Time `json:"claimedAt"`
	TxHash     string    `json:"txHash"`
}

type ClaimSettings struct {
	Enabled       bool    `json:"enabled"`
	AutoApprove   bool    `json:"autoApprove"`
	MinAmount     float64 `json:"minAmount"`
	MaxAmount     float64 `json:"maxAmount"`
}

// ============================================================================
// Service
// ============================================================================

type ClaimService struct {
	mu        sync.RWMutex
	rewards   map[string]*ClaimableReward
	history   map[string][]ClaimHistory
	settings  ClaimSettings
}

func NewClaimService() *ClaimService {
	cs := &ClaimService{
		rewards:  make(map[string]*ClaimableReward),
		history:  make(map[string][]ClaimHistory),
		settings: ClaimSettings{
			Enabled:     true,
			AutoApprove: false,
			MinAmount:   1,
			MaxAmount:   1000000,
		},
	}
	return cs
}

func (cs *ClaimService) GetAvailableRewards(userID string) []*ClaimableReward {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var available []*ClaimableReward
	now := time.Now()
	for _, reward := range cs.rewards {
		if reward.UserID == userID && 
		   (reward.Status == ClaimStatusApproved || reward.Status == ClaimStatusPending) &&
		   now.Before(reward.ExpiresAt) {
			available = append(available, reward)
		}
	}
	return available
}

func (cs *ClaimService) GetUserHistory(userID string) []ClaimHistory {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.history[userID]
}

func (cs *ClaimService) CreateReward(reward *ClaimableReward) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.settings.Enabled {
		reward.Status = ClaimStatusPending
		if cs.settings.AutoApprove {
			reward.Status = ClaimStatusApproved
		}
	} else {
		return fmt.Errorf("claims are currently disabled")
	}

	reward.ID = fmt.Sprintf("claim-%d", time.Now().Unix())
	reward.CreateTime = time.Now()
	cs.rewards[reward.ID] = reward

	return nil
}

func (cs *ClaimService) ApproveReward(rewardID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	reward, ok := cs.rewards[rewardID]
	if !ok {
		return fmt.Errorf("reward not found: %s", rewardID)
	}

	reward.Status = ClaimStatusApproved
	return nil
}

func (cs *ClaimService) RejectReward(rewardID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	reward, ok := cs.rewards[rewardID]
	if !ok {
		return fmt.Errorf("reward not found: %s", rewardID)
	}

	reward.Status = ClaimStatusRejected
	return nil
}

func (cs *ClaimService) ClaimReward(rewardID, txHash string) (*ClaimHistory, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	reward, ok := cs.rewards[rewardID]
	if !ok {
		return nil, fmt.Errorf("reward not found: %s", rewardID)
	}

	if reward.Status != ClaimStatusApproved {
		return nil, fmt.Errorf("reward not approved for claiming")
	}

	if time.Now().After(reward.ExpiresAt) {
		return nil, fmt.Errorf("reward has expired")
	}

	now := time.Now()
	reward.Status = ClaimStatusClaimed
	reward.ClaimTime = &now
	reward.TxHash = txHash

	history := ClaimHistory{
		ID:        fmt.Sprintf("history-%d", time.Now().Unix()),
		UserID:    reward.UserID,
		Type:      reward.Type,
		Title:     reward.Title,
		Amount:    reward.Amount,
		Token:     reward.Token,
		ClaimedAt: now,
		TxHash:    txHash,
	}

	cs.history[reward.UserID] = append(cs.history[reward.UserID], history)
	return &history, nil
}

func (cs *ClaimService) GetSettings() ClaimSettings {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.settings
}

func (cs *ClaimService) UpdateSettings(settings ClaimSettings) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.settings = settings
}

func (cs *ClaimService) GetAllRewards() []*ClaimableReward {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	rewards := make([]*ClaimableReward, 0, len(cs.rewards))
	for _, reward := range cs.rewards {
		rewards = append(rewards, reward)
	}
	return rewards
}

func (cs *ClaimService) ToJSON() (string, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	data := struct {
		Rewards  map[string]*ClaimableReward `json:"rewards"`
		History map[string][]ClaimHistory   `json:"history"`
		Settings ClaimSettings               `json:"settings"`
	}{
		Rewards:  cs.rewards,
		History:  cs.history,
		Settings: cs.settings,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
