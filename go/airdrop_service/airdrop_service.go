/**
 * TigerWallet Airdrop Service
 * 
 * Airdrop campaign management and claim tracking.
 * Built with Go for high-load distributed operations.
 */

package airdrop

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AirdropCampaign represents an airdrop campaign
type AirdropCampaign struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	TokenAddress    string    `json:"token_address"`
	ChainID         uint64    `json:"chain_id"`
	TotalAmount     string    `json:"total_amount"`
	ClaimedAmount   string    `json:"claimed_amount"`
	StartTime       int64     `json:"start_time"`
	EndTime         int64     `json:"end_time"`
	Status          string    `json:"status"`
	ClaimType       string    `json:"claim_type"` // snapshot, merkle, manual
	MerkleRoot      string    `json:"merkle_root"`
	Rules           string    `json:"rules"`
	CreatedAt       int64     `json:"created_at"`
}

// AirdropClaim represents an airdrop claim
type AirdropClaim struct {
	ID            string    `json:"id"`
	CampaignID    string    `json:"campaign_id"`
	UserID        string    `json:"user_id"`
	Address       string    `json:"address"`
	Amount        string    `json:"amount"`
	ClaimedAmount string    `json:"claimed_amount"`
	Status        string    `json:"status"`
	ClaimTxHash   string    `json:"claim_tx_hash"`
	ClaimedAt     int64     `json:"claimed_at"`
	CreatedAt     int64     `json:"created_at"`
}

// AirdropService manages airdrop operations
type AirdropService struct {
	mu        sync.RWMutex
	campaigns map[string]*AirdropCampaign
	claims    map[string]*AirdropClaim
}

var (
	airdropService     *AirdropService
	airdropServiceOnce sync.Once
)

func GetAirdropService() *AirdropService {
	airdropServiceOnce.Do(func() {
		airdropService = &AirdropService{
			campaigns: make(map[string]*AirdropCampaign),
			claims:    make(map[string]*AirdropClaim),
		}
	})
	return airdropService
}

func (s *AirdropService) CreateCampaign(ctx context.Context, campaign *AirdropCampaign) (*AirdropCampaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	campaign.ID = "airdrop_" + uuid.New().String()
	campaign.ClaimedAmount = "0"
	campaign.Status = "active"
	campaign.CreatedAt = time.Now().Unix()

	s.campaigns[campaign.ID] = campaign
	return campaign, nil
}

func (s *AirdropService) GetCampaign(ctx context.Context, campaignID string) (*AirdropCampaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	campaign, exists := s.campaigns[campaignID]
	if !exists {
		return nil, fmt.Errorf("campaign not found")
	}
	return campaign, nil
}

func (s *AirdropService) GetAllCampaigns(ctx context.Context) ([]*AirdropCampaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*AirdropCampaign, 0)
	for _, campaign := range s.campaigns {
		result = append(result, campaign)
	}
	return result, nil
}

func (s *AirdropService) CreateClaim(ctx context.Context, claim *AirdropClaim) (*AirdropClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	campaign, exists := s.campaigns[claim.CampaignID]
	if !exists {
		return nil, fmt.Errorf("campaign not found")
	}

	claim.ID = "claim_" + uuid.New().String()
	claim.ClaimedAmount = "0"
	claim.Status = "pending"
	claim.CreatedAt = time.Now().Unix()

	s.claims[claim.ID] = claim
	return claim, nil
}

func (s *AirdropService) ClaimTokens(ctx context.Context, claimID, txHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	claim, exists := s.claims[claimID]
	if !exists {
		return fmt.Errorf("claim not found")
	}

	if claim.Status == "claimed" {
		return fmt.Errorf("already claimed")
	}

	campaign, _ := s.campaigns[claim.CampaignID]
	if campaign != nil {
		claimed, _ := new(big.Int).SetString(campaign.ClaimedAmount, 10)
		amount, _ := new(big.Int).SetString(claim.Amount, 10)
		claimed.Add(claimed, amount)
		campaign.ClaimedAmount = claimed.String()
	}

	claim.Status = "claimed"
	claim.ClaimTxHash = txHash
	claim.ClaimedAt = time.Now().Unix()
	return nil
}

func (s *AirdropService) GetUserClaims(ctx context.Context, userID string) ([]*AirdropClaim, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*AirdropClaim, 0)
	for _, claim := range s.claims {
		if claim.UserID == userID {
			result = append(result, claim)
		}
	}
	return result, nil
}

func (c *AirdropCampaign) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
