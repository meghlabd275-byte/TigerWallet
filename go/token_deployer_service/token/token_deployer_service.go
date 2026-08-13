/**
 * TigerWallet Token Deployer Service
 *
 * Complete token deployment service for EVM and non-EVM chains.
 * Built with Go for high-load distributed operations.
 */

package token

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

// TokenDeployment represents a token deployment request
type TokenDeployment struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	Name            string         `json:"name"`
	Symbol          string         `json:"symbol"`
	Decimals        int            `json:"decimals"`
	TotalSupply     string         `json:"total_supply"`
	ChainID         uint64         `json:"chain_id"`
	TokenType       TokenType      `json:"token_type"`
	ContractAddress string         `json:"contract_address"`
	TransactionHash string         `json:"transaction_hash"`
	DeployerAddress string         `json:"deployer_address"`
	Features        []TokenFeature `json:"features"`
	Status          DeployStatus   `json:"status"`
	Deployer        string         `json:"deployer"`
	CreatedAt       int64          `json:"created_at"`
	DeployedAt      int64          `json:"deployed_at"`
}

// TokenType represents token type
type TokenType string

const (
	TokenTypeERC20   TokenType = "ERC20"
	TokenTypeERC721  TokenType = "ERC721"
	TokenTypeERC1155 TokenType = "ERC1155"
	TokenTypeBEP20   TokenType = "BEP20"
	TokenTypeTRC20   TokenType = "TRC20"
	TokenTypeSPL     TokenType = "SPL"
	TokenTypeCAP20   TokenType = "CAP20"
)

// TokenFeature represents token features
type TokenFeature string

const (
	FeatureMintable  TokenFeature = "mintable"
	FeatureBurnable  TokenFeature = "burnable"
	FeaturePausable  TokenFeature = "pausable"
	FeatureSnapshots TokenFeature = "snapshots"
	FeatureVotes     TokenFeature = "votes"
	FeatureFlashMint TokenFeature = "flash_mint"
)

// DeployStatus represents deployment status
type DeployStatus string

const (
	DeployStatusPending   DeployStatus = "pending"
	DeployStatusDeploying DeployStatus = "deploying"
	DeployStatusDeployed  DeployStatus = "deployed"
	DeployStatusFailed    DeployStatus = "failed"
)

// TokenConfig represents token configuration
type TokenConfig struct {
	Name           string   `json:"name"`
	Symbol         string   `json:"symbol"`
	Decimals       int      `json:"decimals"`
	TotalSupply    string   `json:"total_supply"`
	InitialHolders []Holder `json:"initial_holders"`
	MintAddress    string   `json:"mint_address"`
	MaxSupply      string   `json:"max_supply"`
	TransferFee    string   `json:"transfer_fee"`
	FeeRecipient   string   `json:"fee_recipient"`
}

// Holder represents initial holder
type Holder struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
}

// TokenDeployerService manages token deployments
type TokenDeployerService struct {
	mu          sync.RWMutex
	deployments map[string]*TokenDeployment
	presets     map[string]*TokenConfig
}

// ============================================================================
// Service Methods
// ============================================================================

var (
	tokenDeployerService     *TokenDeployerService
	tokenDeployerServiceOnce sync.Once
)

// GetTokenDeployerService returns the singleton token deployer service
func GetTokenDeployerService() *TokenDeployerService {
	tokenDeployerServiceOnce.Do(func() {
		tokenDeployerService = &TokenDeployerService{
			deployments: make(map[string]*TokenDeployment),
			presets:     make(map[string]*TokenConfig),
		}
		tokenDeployerService.initPresets()
	})
	return tokenDeployerService
}

func (s *TokenDeployerService) initPresets() {
	s.presets["standard"] = &TokenConfig{
		Name:        "",
		Symbol:      "",
		Decimals:    18,
		TotalSupply: "1000000000",
	}
	s.presets["deflationary"] = &TokenConfig{
		Name:         "",
		Symbol:       "",
		Decimals:     18,
		TotalSupply:  "1000000000",
		TransferFee:  "100",
		FeeRecipient: "",
	}
	s.presets["reward"] = &TokenConfig{
		Name:        "",
		Symbol:      "",
		Decimals:    18,
		TotalSupply: "1000000000",
		MintAddress: "",
	}
}

func (s *TokenDeployerService) CreateDeployment(ctx context.Context, deployment *TokenDeployment) (*TokenDeployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if deployment.Name == "" {
		return nil, fmt.Errorf("token name is required")
	}
	if deployment.Symbol == "" {
		return nil, fmt.Errorf("token symbol is required")
	}
	if deployment.Decimals < 0 || deployment.Decimals > 18 {
		return nil, fmt.Errorf("invalid decimals")
	}
	if deployment.TotalSupply == "" {
		return nil, fmt.Errorf("total supply is required")
	}

	deployment.ID = "deployment_" + uuid.New().String()
	deployment.Status = DeployStatusPending
	deployment.CreatedAt = time.Now().Unix()

	s.deployments[deployment.ID] = deployment
	return deployment, nil
}

func (s *TokenDeployerService) Deploy(ctx context.Context, deploymentID, txHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	deployment, exists := s.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found")
	}

	deployment.Status = DeployStatusDeploying
	deployment.TransactionHash = txHash
	return nil
}

func (s *TokenDeployerService) CompleteDeployment(ctx context.Context, deploymentID, contractAddress string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	deployment, exists := s.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found")
	}

	deployment.Status = DeployStatusDeployed
	deployment.ContractAddress = contractAddress
	deployment.DeployedAt = time.Now().Unix()
	return nil
}

func (s *TokenDeployerService) GetDeployment(ctx context.Context, deploymentID string) (*TokenDeployment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deployment, exists := s.deployments[deploymentID]
	if !exists {
		return nil, fmt.Errorf("deployment not found")
	}
	return deployment, nil
}

func (s *TokenDeployerService) GetUserDeployments(ctx context.Context, userID string) ([]*TokenDeployment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*TokenDeployment, 0)
	for _, deployment := range s.deployments {
		if deployment.UserID == userID {
			result = append(result, deployment)
		}
	}
	return result, nil
}

func (s *TokenDeployerService) GetPresets(ctx context.Context) (map[string]*TokenConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.presets, nil
}

func (s *TokenDeployerService) ValidateTokenConfig(config *TokenConfig) error {
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}
	if config.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if config.Decimals < 0 || config.Decimals > 18 {
		return fmt.Errorf("invalid decimals")
	}
	totalSupply, ok := new(big.Int).SetString(config.TotalSupply, 10)
	if !ok {
		return fmt.Errorf("invalid total supply")
	}
	if totalSupply.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("total supply must be greater than 0")
	}
	return nil
}

func (d *TokenDeployment) ToJSON() (string, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
