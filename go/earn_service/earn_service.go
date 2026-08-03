/**
 * TigerWallet Earn Service
 * 
 * Complete earn products including fixed deposits, flexible savings,
 * and yield farming.
 * Built with Go for high-load distributed operations.
 */

package earn

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

// EarnProduct represents an earn product
type EarnProduct struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	ProductType    ProductType       `json:"product_type"`
	ChainID        uint64           `json:"chain_id"`
	TokenAddress   string           `json:"token_address"`
	Token          *TokenInfo       `json:"token"`
	APY            string           `json:"apy"`
	APYType        APYType          `json:"apy_type"` // fixed, flexible, tiered
	MinDeposit     string           `json:"min_deposit"`
	MaxDeposit     string           `json:"max_deposit"`
	MinTerm        int64            `json:"min_term"` // in seconds
	MaxTerm        int64            `json:"max_term"` // in seconds
	TotalDeposited string           `json:"total_deposited"`
	TotalValue     string           `json:"total_value"`
	MaxCapacity    string           `json:"max_capacity"`
	Status         ProductStatus    `json:"status"`
	Features       []string         `json:"features"` // auto-compound, early-withdrawal, etc.
	CreatedAt      int64            `json:"created_at"`
	UpdatedAt      int64            `json:"updated_at"`
}

// TokenInfo represents token information
type TokenInfo struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	LogoURL  string `json:"logo_url"`
}

// ProductType represents product type
type ProductType string

const (
	ProductTypeFixed     ProductType = "fixed"
	ProductTypeFlexible  ProductType = "flexible"
	ProductTypeTiered    ProductType = "tiered"
	ProductTypeLockdrop  ProductType = "lockdrop"
)

// APYType represents APY calculation type
type APYType string

const (
	APYTypeFixed    APYType = "fixed"
	APYTypeFlexible APYType = "flexible"
	APYTypeTiered   APYType = "tiered"
)

// ProductStatus represents product status
type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "active"
	ProductStatusPaused  ProductStatus = "paused"
	ProductStatusClosed  ProductStatus = "closed"
	ProductStatusExpired ProductStatus = "expired"
)

// UserDeposit represents a user deposit
type UserDeposit struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	ProductID      string    `json:"product_id"`
	Amount         string    `json:"amount"`
	Principal      string    `json:"principal"`
	Interest       string    `json:"interest"`
	APY            string    `json:"apy"`
	Term           int64     `json:"term"` // in seconds
	StartTime      int64     `json:"start_time"`
	MaturityTime   int64     `json:"maturity_time"`
	ClaimTime      int64     `json:"claim_time"`
	Status         string    `json:"status"` // active, matured, claimed, withdrawn
	AutoCompound   bool      `json:"auto_compound"`
}

// Tier represents APY tier
type Tier struct {
	MinAmount string `json:"min_amount"`
	MaxAmount string `json:"max_amount"`
	APY       string `json:"apy"`
}

// YieldSnapshot represents yield snapshot
type YieldSnapshot struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	APY       string    `json:"apy"`
	Timestamp int64     `json:"timestamp"`
}

// EarnService manages earn products
type EarnService struct {
	mu            sync.RWMutex
	products      map[string]*EarnProduct
	deposits      map[string]*UserDeposit
	tierAPYs      map[string][]Tier
	snapshots     map[string]*YieldSnapshot
}

// ============================================================================
// Service Methods
// ============================================================================

var (
	earnService     *EarnService
	earnServiceOnce sync.Once
)

// GetEarnService returns the singleton earn service
func GetEarnService() *EarnService {
	earnServiceOnce.Do(func() {
		earnService = &EarnService{
			products:  make(map[string]*EarnProduct),
			deposits: make(map[string]*UserDeposit),
			tierAPYs: make(map[string][]Tier),
			snapshots: make(map[string]*YieldSnapshot),
		}
	})
	return earnService
}

// ============================================================================
// Product Operations
// ============================================================================

// CreateProduct creates a new earn product
func (s *EarnService) CreateProduct(ctx context.Context, product *EarnProduct) (*EarnProduct, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product.ID = "earn_" + uuid.New().String()
	product.Status = ProductStatusActive
	product.TotalDeposited = "0"
	product.TotalValue = "0"
	product.CreatedAt = time.Now().Unix()
	product.UpdatedAt = time.Now().Unix()

	s.products[product.ID] = product
	return product, nil
}

// GetProduct returns a product by ID
func (s *EarnService) GetProduct(ctx context.Context, productID string) (*EarnProduct, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, exists := s.products[productID]
	if !exists {
		return nil, fmt.Errorf("product not found")
	}
	return product, nil
}

// GetAllProducts returns all products
func (s *EarnService) GetAllProducts(ctx context.Context, productType string, status string) ([]*EarnProduct, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*EarnProduct, 0)
	for _, product := range s.products {
		if productType != "" && string(product.ProductType) != productType {
			continue
		}
		if status != "" && string(product.Status) != status {
			continue
		}
		result = append(result, product)
	}
	return result, nil
}

// GetProductsByChain returns products for a chain
func (s *EarnService) GetProductsByChain(ctx context.Context, chainID uint64) ([]*EarnProduct, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*EarnProduct, 0)
	for _, product := range s.products {
		if product.ChainID == chainID {
			result = append(result, product)
		}
	}
	return result, nil
}

// UpdateProduct updates a product
func (s *EarnService) UpdateProduct(ctx context.Context, product *EarnProduct) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.products[product.ID]
	if !exists {
		return fmt.Errorf("product not found")
	}

	existing.Name = product.Name
	existing.Description = product.Description
	existing.APY = product.APY
	existing.MinDeposit = product.MinDeposit
	existing.MaxDeposit = product.MaxDeposit
	existing.MinTerm = product.MinTerm
	existing.MaxTerm = product.MaxTerm
	existing.Status = product.Status
	existing.Features = product.Features
	existing.UpdatedAt = time.Now().Unix()

	return nil
}

// UpdateProductStatus updates product status
func (s *EarnService) UpdateProductStatus(ctx context.Context, productID string, status ProductStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, exists := s.products[productID]
	if !exists {
		return fmt.Errorf("product not found")
	}

	product.Status = status
	product.UpdatedAt = time.Now().Unix()
	return nil
}

// DeleteProduct deletes a product
func (s *EarnService) DeleteProduct(ctx context.Context, productID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.products[productID]; !exists {
		return fmt.Errorf("product not found")
	}

	delete(s.products, productID)
	return nil
}

// ============================================================================
// Deposit Operations
// ============================================================================

// Deposit creates a deposit
func (s *EarnService) Deposit(ctx context.Context, deposit *UserDeposit) (*UserDeposit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify product exists
	product, exists := s.products[deposit.ProductID]
	if !exists {
		return nil, fmt.Errorf("product not found")
	}

	// Validate amount
	amount, err := new(big.Int).SetString(deposit.Amount, 10)
	if err != nil {
		return nil, fmt.Errorf("invalid amount")
	}

	minDeposit, _ := new(big.Int).SetString(product.MinDeposit, 10)
	maxDeposit, _ := new(big.Int).SetString(product.MaxDeposit, 10)

	if amount.Cmp(minDeposit) < 0 {
		return nil, fmt.Errorf("amount below minimum deposit")
	}

	if maxDeposit.Cmp(big.NewInt(0)) > 0 && amount.Cmp(maxDeposit) > 0 {
		return nil, fmt.Errorf("amount exceeds maximum deposit")
	}

	// Check capacity
	totalDeposited, _ := new(big.Int).SetString(product.TotalDeposited, 10)
	maxCapacity, _ := new(big.Int).SetString(product.MaxCapacity, 10)

	if maxCapacity.Cmp(big.NewInt(0)) > 0 {
		newTotal := new(big.Int).Add(totalDeposited, amount)
		if newTotal.Cmp(maxCapacity) > 0 {
			return nil, fmt.Errorf("product capacity exceeded")
		}
	}

	// Get APY based on type
	apy := product.APY
	if product.APYType == APYTypeTiered {
		apy = s.calculateTieredAPY(product.ID, amount.String())
	}

	// Create deposit
	deposit.ID = "deposit_" + uuid.New().String()
	deposit.Principal = deposit.Amount
	deposit.Interest = "0"
	deposit.APY = apy
	deposit.StartTime = time.Now().Unix()
	deposit.Status = "active"

	// Set maturity time for fixed products
	if product.ProductType == ProductTypeFixed && product.MaxTerm > 0 {
		deposit.Term = product.MaxTerm
		deposit.MaturityTime = deposit.StartTime + product.MaxTerm
	}

	s.deposits[deposit.ID] = deposit

	// Update product totals
	totalDeposited.Add(totalDeposited, amount)
	product.TotalDeposited = totalDeposited.String()
	product.TotalValue = totalDeposited.String()

	return deposit, nil
}

// calculateTieredAPY calculates APY based on tier
func (s *EarnService) calculateTieredAPY(productID string, amount string) string {
	tiers, exists := s.tierAPYs[productID]
	if !exists {
		return "0"
	}

	amountInt, err := new(big.Int).SetString(amount, 10)
	if err != nil {
		return "0"
	}

	for _, tier := range tiers {
		minAmount, _ := new(big.Int).SetString(tier.MinAmount, 10)
		maxAmount, _ := new(big.Int).SetString(tier.MaxAmount, 10)

		if amountInt.Cmp(minAmount) >= 0 {
			if maxAmount.Cmp(big.NewInt(0)) == 0 || amountInt.Cmp(maxAmount) <= 0 {
				return tier.APY
			}
		}
	}

	return "0"
}

// GetDeposit returns a deposit
func (s *EarnService) GetDeposit(ctx context.Context, depositID string) (*UserDeposit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deposit, exists := s.deposits[depositID]
	if !exists {
		return nil, fmt.Errorf("deposit not found")
	}
	return deposit, nil
}

// GetUserDeposits returns all deposits for a user
func (s *EarnService) GetUserDeposits(ctx context.Context, userID string) ([]*UserDeposit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*UserDeposit, 0)
	for _, deposit := range s.deposits {
		if deposit.UserID == userID {
			result = append(result, deposit)
		}
	}
	return result, nil
}

// GetUserActiveDeposits returns active deposits for a user
func (s *EarnService) GetUserActiveDeposits(ctx context.Context, userID string) ([]*UserDeposit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*UserDeposit, 0)
	for _, deposit := range s.deposits {
		if deposit.UserID == userID && deposit.Status == "active" {
			result = append(result, deposit)
		}
	}
	return result, nil
}

// GetUserDepositsByProduct returns deposits for a user in a product
func (s *EarnService) GetUserDepositsByProduct(ctx context.Context, userID, productID string) ([]*UserDeposit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*UserDeposit, 0)
	for _, deposit := range s.deposits {
		if deposit.UserID == userID && deposit.ProductID == productID {
			result = append(result, deposit)
		}
	}
	return result, nil
}

// ============================================================================
// Withdrawal Operations
// ============================================================================

// Withdraw withdraws from a deposit
func (s *EarnService) Withdraw(ctx context.Context, depositID string, amount string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deposit, exists := s.deposits[depositID]
	if !exists {
		return "0", fmt.Errorf("deposit not found")
	}

	if deposit.Status != "active" {
		return "0", fmt.Errorf("deposit not active")
	}

	product, exists := s.products[deposit.ProductID]
	if !exists {
		return "0", fmt.Errorf("product not found")
	}

	// Check if early withdrawal is allowed
	isEarlyWithdrawal := false
	for _, feature := range product.Features {
		if feature == "early-withdrawal" {
			isEarlyWithdrawal = true
			break
		}
	}

	// For fixed products, check maturity
	if product.ProductType == ProductTypeFixed {
		if time.Now().Unix() < deposit.MaturityTime && !isEarlyWithdrawal {
			return "0", fmt.Errorf("deposit not yet matured")
		}
	}

	// Calculate interest
	withdrawAmount, err := new(big.Int).SetString(amount, 10)
	if err != nil {
		return "0", err
	}

	principal, _ := new(big.Int).SetString(deposit.Principal, 10)
	interest, _ := new(big.Int).SetString(deposit.Interest, 10)

	if withdrawAmount.Cmp(principal) > 0 {
		withdrawAmount = principal
	}

	// Calculate proportional interest
	interestRatio := new(big.Int).Div(withdrawAmount, principal)
	claimedInterest := new(big.Int).Mul(interest, interestRatio)

	// Update deposit
	if withdrawAmount.Cmp(principal) == 0 {
		deposit.Status = "withdrawn"
	} else {
		principal.Sub(principal, withdrawAmount)
		deposit.Principal = principal.String()
		interest.Sub(interest, claimedInterest)
		deposit.Interest = interest.String()
	}

	// Update product total
	totalDeposited, _ := new(big.Int).SetString(product.TotalDeposited, 10)
	totalDeposited.Sub(totalDeposited, withdrawAmount)
	product.TotalDeposited = totalDeposited.String()
	product.TotalValue = totalDeposited.String()

	return claimedInterest.String(), nil
}

// Claim claims matured deposit
func (s *EarnService) Claim(ctx context.Context, depositID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deposit, exists := s.deposits[depositID]
	if !exists {
		return "0", fmt.Errorf("deposit not found")
	}

	if deposit.Status == "claimed" {
		return "0", fmt.Errorf("already claimed")
	}

	product, exists := s.products[deposit.ProductID]
	if !exists {
		return "0", fmt.Errorf("product not found")
	}

	// Calculate interest
	interest, err := s.calculateInterest(deposit)
	if err != nil {
		return "0", err
	}

	// Update deposit
	deposit.Interest = interest
	deposit.Status = "matured"
	deposit.ClaimTime = time.Now().Unix()

	// If auto-compound, add interest to principal
	if deposit.AutoCompound {
		principal, _ := new(big.Int).SetString(deposit.Principal, 10)
		interestInt, _ := new(big.Int).SetString(interest, 10)
		principal.Add(principal, interestInt)
		deposit.Principal = principal.String()
		deposit.Status = "active"
	}

	return interest, nil
}

// calculateInterest calculates interest for a deposit
func (s *EarnService) calculateInterest(deposit *UserDeposit) (string, error) {
	product, exists := s.products[deposit.ProductID]
	if !exists {
		return "0", fmt.Errorf("product not found")
	}

	principal, _ := new(big.Int).SetString(deposit.Principal, 10)
	apy, _ := new(big.Int).SetString(deposit.APY, 10)

	// Time in seconds
	var timeStaked int64
	if deposit.Status == "active" {
		timeStaked = time.Now().Unix() - deposit.StartTime
	} else if deposit.MaturityTime > 0 {
		timeStaked = deposit.MaturityTime - deposit.StartTime
	}

	// Interest = principal * apy * time / (100 * 365 * 86400)
	interest := new(big.Int).Mul(principal, apy)
	interest.Mul(interest, big.NewInt(timeStaked))
	denominator := big.NewInt(100 * 365 * 86400)
	interest.Div(interest, denominator)

	return interest.String(), nil
}

// ============================================================================
// Tier Operations
// ============================================================================

// SetTiers sets APY tiers for a product
func (s *EarnService) SetTiers(ctx context.Context, productID string, tiers []Tier) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.products[productID]; !exists {
		return fmt.Errorf("product not found")
	}

	s.tierAPYs[productID] = tiers
	return nil
}

// GetTiers returns APY tiers for a product
func (s *EarnService) GetTiers(ctx context.Context, productID string) ([]Tier, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tiers, exists := s.tierAPYs[productID]
	if !exists {
		return nil, fmt.Errorf("tiers not found")
	}

	return tiers, nil
}

// ============================================================================
// Utility Methods
// ============================================================================

// CalculateProjectedReturns calculates projected returns
func (s *EarnService) CalculateProjectedReturns(ctx context.Context, productID, amount, term string) (string, error) {
	s.mu.RLock()
	product, exists := s.products[productID]
	s.mu.RUnlock()

	if !exists {
		return "0", fmt.Errorf("product not found")
	}

	amountInt, err := new(big.Int).SetString(amount, 10)
	if err != nil {
		return "0", err
	}

	termInt, err := new(big.Int).SetString(term, 10)
	if err != nil {
		return "0", err
	}

	// Get APY
	apy, err := new(big.Int).SetString(product.APY, 10)
	if err != nil {
		return "0", err
	}

	// Returns = amount * apy * term / (100 * 365 * 86400)
	returns := new(big.Int).Mul(amountInt, apy)
	returns.Mul(returns, termInt)
	denominator := big.NewInt(100 * 365 * 86400)
	returns.Div(returns, denominator)

	return returns.String(), nil
}

// GetUserTotalEarnings returns total earnings for a user
func (s *EarnService) GetUserTotalEarnings(ctx context.Context, userID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalEarnings := big.NewInt(0)

	for _, deposit := range s.deposits {
		if deposit.UserID == userID {
			interest, _ := new(big.Int).SetString(deposit.Interest, 10)
			totalEarnings.Add(totalEarnings, interest)
		}
	}

	return totalEarnings.String(), nil
}

// ToJSON converts deposit to JSON
func (d *UserDeposit) ToJSON() (string, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
