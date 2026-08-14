/**
 * TigerWallet Airdrop Service
 *
 * Airdrop campaign management and claim tracking.
 * Built with Go for high-load distributed operations.
 * PostgreSQL-backed — all campaigns and claims are persisted.
 */

package airdrop

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AirdropCampaign represents an airdrop campaign
type AirdropCampaign struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	TokenAddress  string `json:"token_address"`
	ChainID       uint64 `json:"chain_id"`
	TotalAmount   string `json:"total_amount"`
	ClaimedAmount  string `json:"claimed_amount"`
	StartTime     int64  `json:"start_time"`
	EndTime       int64  `json:"end_time"`
	Status        string `json:"status"`
	ClaimType     string `json:"claim_type"` // snapshot, merkle, manual
	MerkleRoot    string `json:"merkle_root"`
	Rules         string `json:"rules"`
	CreatedAt     int64  `json:"created_at"`
}

// AirdropClaim represents an airdrop claim
type AirdropClaim struct {
	ID             string `json:"id"`
	CampaignID     string `json:"campaign_id"`
	UserID         string `json:"user_id"`
	Address        string `json:"address"`
	Amount         string `json:"amount"`
	ClaimedAmount  string `json:"claimed_amount"`
	Status         string `json:"status"`
	ClaimTxHash    string `json:"claim_tx_hash"`
	ClaimedAt      int64  `json:"claimed_at"`
	CreatedAt      int64  `json:"created_at"`
}

// AirdropService manages airdrop operations backed by PostgreSQL.
type AirdropService struct {
	pg *pgxpool.Pool
}

var (
	airdropService     *AirdropService
	airdropServiceOnce interface{} // unused; kept for API stability
)

// NewAirdropService returns a service backed by the given pgxpool.
func NewAirdropService(pg *pgxpool.Pool) *AirdropService {
	return &AirdropService{pg: pg}
}

// GetAirdropService returns the package-level singleton (must be set via
// SetAirdropService before first use; falls back to a nil-pool service that
// will fail-closed on every operation).
func GetAirdropService() *AirdropService {
	if airdropService != nil {
		return airdropService
	}
	return &AirdropService{}
}

// SetAirdropService wires the PostgreSQL-backed singleton. Called from main.
func SetAirdropService(pg *pgxpool.Pool) {
	airdropService = NewAirdropService(pg)
}

const airdropSchema = `
CREATE TABLE IF NOT EXISTS airdrop_campaigns (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    token_address   TEXT NOT NULL DEFAULT '',
    chain_id        BIGINT NOT NULL DEFAULT 0,
    total_amount    TEXT NOT NULL DEFAULT '0',
    claimed_amount  TEXT NOT NULL DEFAULT '0',
    start_time      BIGINT NOT NULL DEFAULT 0,
    end_time        BIGINT NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'active',
    claim_type      TEXT NOT NULL DEFAULT 'manual',
    merkle_root     TEXT NOT NULL DEFAULT '',
    rules           TEXT NOT NULL DEFAULT '',
    created_at      BIGINT NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS airdrop_claims (
    id              TEXT PRIMARY KEY,
    campaign_id     TEXT NOT NULL REFERENCES airdrop_campaigns(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL DEFAULT '',
    address         TEXT NOT NULL DEFAULT '',
    amount          TEXT NOT NULL DEFAULT '0',
    claimed_amount  TEXT NOT NULL DEFAULT '0',
    status          TEXT NOT NULL DEFAULT 'pending',
    claim_tx_hash   TEXT NOT NULL DEFAULT '',
    claimed_at      BIGINT NOT NULL DEFAULT 0,
    created_at      BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_airdrop_claims_user ON airdrop_claims(user_id);
CREATE INDEX IF NOT EXISTS idx_airdrop_claims_campaign ON airdrop_claims(campaign_id);
`

// Migrate creates the tables if they do not exist.
func (s *AirdropService) Migrate(ctx context.Context) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	_, err := s.pg.Exec(ctx, airdropSchema)
	return err
}

func (s *AirdropService) CreateCampaign(ctx context.Context, campaign *AirdropCampaign) (*AirdropCampaign, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	campaign.ID = "airdrop_" + uuid.New().String()
	if campaign.ClaimedAmount == "" {
		campaign.ClaimedAmount = "0"
	}
	if campaign.Status == "" {
		campaign.Status = "active"
	}
	if campaign.ClaimType == "" {
		campaign.ClaimType = "manual"
	}
	campaign.CreatedAt = time.Now().Unix()

	_, err := s.pg.Exec(ctx, `INSERT INTO airdrop_campaigns
		(id,name,description,token_address,chain_id,total_amount,claimed_amount,start_time,end_time,status,claim_type,merkle_root,rules,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		campaign.ID, campaign.Name, campaign.Description, campaign.TokenAddress, campaign.ChainID,
		campaign.TotalAmount, campaign.ClaimedAmount, campaign.StartTime, campaign.EndTime,
		campaign.Status, campaign.ClaimType, campaign.MerkleRoot, campaign.Rules, campaign.CreatedAt)
	if err != nil {
		return nil, err
	}
	return campaign, nil
}

func (s *AirdropService) GetCampaign(ctx context.Context, campaignID string) (*AirdropCampaign, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	row := s.pg.QueryRow(ctx, `SELECT id,name,description,token_address,chain_id,total_amount,claimed_amount,
		start_time,end_time,status,claim_type,merkle_root,rules,created_at FROM airdrop_campaigns WHERE id=$1`, campaignID)
	var c AirdropCampaign
	if err := row.Scan(&c.ID, &c.Name, &c.Description, &c.TokenAddress, &c.ChainID, &c.TotalAmount,
		&c.ClaimedAmount, &c.StartTime, &c.EndTime, &c.Status, &c.ClaimType, &c.MerkleRoot, &c.Rules, &c.CreatedAt); err != nil {
		return nil, fmt.Errorf("campaign not found")
	}
	return &c, nil
}

func (s *AirdropService) GetAllCampaigns(ctx context.Context) ([]*AirdropCampaign, error) {
	if s.pg == nil {
		return []*AirdropCampaign{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT id,name,description,token_address,chain_id,total_amount,claimed_amount,
		start_time,end_time,status,claim_type,merkle_root,rules,created_at FROM airdrop_campaigns ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*AirdropCampaign, 0)
	for rows.Next() {
		var c AirdropCampaign
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.TokenAddress, &c.ChainID, &c.TotalAmount,
			&c.ClaimedAmount, &c.StartTime, &c.EndTime, &c.Status, &c.ClaimType, &c.MerkleRoot, &c.Rules, &c.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *AirdropService) CreateClaim(ctx context.Context, claim *AirdropClaim) (*AirdropClaim, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	campaign, err := s.GetCampaign(ctx, claim.CampaignID)
	if err != nil {
		return nil, err
	}
	if campaign.Status != "active" {
		return nil, fmt.Errorf("campaign is not active")
	}
	claim.ID = "claim_" + uuid.New().String()
	if claim.Amount == "" {
		claim.Amount = campaign.TotalAmount
	}
	claim.ClaimedAmount = "0"
	claim.Status = "pending"
	claim.CreatedAt = time.Now().Unix()

	_, err = s.pg.Exec(ctx, `INSERT INTO airdrop_claims
		(id,campaign_id,user_id,address,amount,claimed_amount,status,claim_tx_hash,claimed_at,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		claim.ID, claim.CampaignID, claim.UserID, claim.Address, claim.Amount,
		claim.ClaimedAmount, claim.Status, claim.ClaimTxHash, claim.ClaimedAt, claim.CreatedAt)
	if err != nil {
		return nil, err
	}
	return claim, nil
}

func (s *AirdropService) ClaimTokens(ctx context.Context, claimID, txHash string) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status, campaignID, amount string
	err = tx.QueryRow(ctx, `SELECT status,campaign_id,amount FROM airdrop_claims WHERE id=$1 FOR UPDATE`, claimID).
		Scan(&status, &campaignID, &amount)
	if err != nil {
		return fmt.Errorf("claim not found")
	}
	if status == "claimed" {
		return fmt.Errorf("already claimed")
	}

	var claimedStr string
	err = tx.QueryRow(ctx, `SELECT claimed_amount FROM airdrop_campaigns WHERE id=$1 FOR UPDATE`, campaignID).Scan(&claimedStr)
	if err != nil {
		return fmt.Errorf("campaign not found")
	}
	claimed, _ := new(big.Int).SetString(claimedStr, 10)
	amt, _ := new(big.Int).SetString(amount, 10)
	if claimed == nil {
		claimed = new(big.Int)
	}
	if amt == nil {
		amt = new(big.Int)
	}
	claimed.Add(claimed, amt)

	if _, err := tx.Exec(ctx, `UPDATE airdrop_campaigns SET claimed_amount=$1 WHERE id=$2`, claimed.String(), campaignID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE airdrop_claims SET status='claimed', claim_tx_hash=$1, claimed_at=$2 WHERE id=$3`,
		txHash, time.Now().Unix(), claimID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *AirdropService) GetUserClaims(ctx context.Context, userID string) ([]*AirdropClaim, error) {
	if s.pg == nil {
		return []*AirdropClaim{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT id,campaign_id,user_id,address,amount,claimed_amount,status,claim_tx_hash,claimed_at,created_at
		FROM airdrop_claims WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*AirdropClaim, 0)
	for rows.Next() {
		var c AirdropClaim
		if err := rows.Scan(&c.ID, &c.CampaignID, &c.UserID, &c.Address, &c.Amount, &c.ClaimedAmount,
			&c.Status, &c.ClaimTxHash, &c.ClaimedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (c *AirdropCampaign) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
