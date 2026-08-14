/**
 * TigerWallet Earn Service
 *
 * Complete earn products including fixed deposits, flexible savings,
 * and yield farming.
 * Built with Go for high-load distributed operations.
 * PostgreSQL-backed — all products, deposits, tiers, and snapshots persisted.
 */

package earn

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================================
// Types
// ============================================================================

// EarnProduct represents an earn product
type EarnProduct struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	ProductType    ProductType   `json:"product_type"`
	ChainID        uint64        `json:"chain_id"`
	TokenAddress   string        `json:"token_address"`
	Token          *TokenInfo    `json:"token"`
	APY            string        `json:"apy"`
	APYType        APYType       `json:"apy_type"` // fixed, flexible, tiered
	MinDeposit     string        `json:"min_deposit"`
	MaxDeposit     string        `json:"max_deposit"`
	MinTerm        int64         `json:"min_term"` // in seconds
	MaxTerm        int64         `json:"max_term"` // in seconds
	TotalDeposited string        `json:"total_deposited"`
	TotalValue     string        `json:"total_value"`
	MaxCapacity    string        `json:"max_capacity"`
	Status         ProductStatus `json:"status"`
	Features       []string      `json:"features"` // auto-compound, early-withdrawal, etc.
	CreatedAt      int64         `json:"created_at"`
	UpdatedAt      int64         `json:"updated_at"`
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
	ProductTypeFixed    ProductType = "fixed"
	ProductTypeFlexible ProductType = "flexible"
	ProductTypeTiered   ProductType = "tiered"
	ProductTypeLockdrop ProductType = "lockdrop"
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
	ProductStatusActive  ProductStatus = "active"
	ProductStatusPaused  ProductStatus = "paused"
	ProductStatusClosed  ProductStatus = "closed"
	ProductStatusExpired ProductStatus = "expired"
)

// UserDeposit represents a user deposit
type UserDeposit struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	ProductID    string `json:"product_id"`
	Amount       string `json:"amount"`
	Principal    string `json:"principal"`
	Interest     string `json:"interest"`
	APY          string `json:"apy"`
	Term         int64  `json:"term"` // in seconds
	StartTime    int64  `json:"start_time"`
	MaturityTime int64  `json:"maturity_time"`
	ClaimTime    int64  `json:"claim_time"`
	Status       string `json:"status"` // active, matured, claimed, withdrawn
	AutoCompound bool   `json:"auto_compound"`
}

// Tier represents APY tier
type Tier struct {
	MinAmount string `json:"min_amount"`
	MaxAmount string `json:"max_amount"`
	APY       string `json:"apy"`
}

// YieldSnapshot represents yield snapshot
type YieldSnapshot struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	APY       string `json:"apy"`
	Timestamp int64  `json:"timestamp"`
}

// EarnService manages earn operations backed by PostgreSQL.
type EarnService struct {
	pg *pgxpool.Pool
}

// ============================================================================
// Service Methods
// ============================================================================

var earnService *EarnService

// NewEarnService returns a service backed by the given pgxpool.
func NewEarnService(pg *pgxpool.Pool) *EarnService {
	return &EarnService{pg: pg}
}

// GetEarnService returns the package-level singleton (must be set via
// SetEarnService before first use; falls back to a nil-pool service that
// will fail-closed on every operation).
func GetEarnService() *EarnService {
	if earnService != nil {
		return earnService
	}
	return &EarnService{}
}

// SetEarnService wires the PostgreSQL-backed singleton. Called from main.
func SetEarnService(pg *pgxpool.Pool) {
	earnService = NewEarnService(pg)
}

const earnSchema = `
CREATE TABLE IF NOT EXISTS earn_products (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    product_type    TEXT NOT NULL DEFAULT 'flexible',
    chain_id        BIGINT NOT NULL DEFAULT 0,
    token_address   TEXT NOT NULL DEFAULT '',
    token           JSONB,
    apy             TEXT NOT NULL DEFAULT '0',
    apy_type        TEXT NOT NULL DEFAULT 'fixed',
    min_deposit     TEXT NOT NULL DEFAULT '0',
    max_deposit     TEXT NOT NULL DEFAULT '0',
    min_term        BIGINT NOT NULL DEFAULT 0,
    max_term        BIGINT NOT NULL DEFAULT 0,
    total_deposited TEXT NOT NULL DEFAULT '0',
    total_value     TEXT NOT NULL DEFAULT '0',
    max_capacity    TEXT NOT NULL DEFAULT '0',
    status          TEXT NOT NULL DEFAULT 'active',
    features        JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at      BIGINT NOT NULL DEFAULT 0,
    updated_at      BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_earn_products_chain ON earn_products(chain_id);
CREATE INDEX IF NOT EXISTS idx_earn_products_type ON earn_products(product_type);
CREATE INDEX IF NOT EXISTS idx_earn_products_status ON earn_products(status);

CREATE TABLE IF NOT EXISTS earn_deposits (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL DEFAULT '',
    product_id    TEXT NOT NULL REFERENCES earn_products(id) ON DELETE CASCADE,
    amount        TEXT NOT NULL DEFAULT '0',
    principal     TEXT NOT NULL DEFAULT '0',
    interest      TEXT NOT NULL DEFAULT '0',
    apy           TEXT NOT NULL DEFAULT '0',
    term          BIGINT NOT NULL DEFAULT 0,
    start_time    BIGINT NOT NULL DEFAULT 0,
    maturity_time BIGINT NOT NULL DEFAULT 0,
    claim_time    BIGINT NOT NULL DEFAULT 0,
    status        TEXT NOT NULL DEFAULT 'active',
    auto_compound BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_earn_deposits_user ON earn_deposits(user_id);
CREATE INDEX IF NOT EXISTS idx_earn_deposits_product ON earn_deposits(product_id);
CREATE INDEX IF NOT EXISTS idx_earn_deposits_status ON earn_deposits(status);

CREATE TABLE IF NOT EXISTS earn_tiers (
    id         TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES earn_products(id) ON DELETE CASCADE,
    min_amount TEXT NOT NULL DEFAULT '0',
    max_amount TEXT NOT NULL DEFAULT '0',
    apy        TEXT NOT NULL DEFAULT '0',
    rank       INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_earn_tiers_product ON earn_tiers(product_id);

CREATE TABLE IF NOT EXISTS earn_yield_snapshots (
    id         TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES earn_products(id) ON DELETE CASCADE,
    apy        TEXT NOT NULL DEFAULT '0',
    timestamp  BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_earn_yield_snapshots_product ON earn_yield_snapshots(product_id);
CREATE INDEX IF NOT EXISTS idx_earn_yield_snapshots_ts ON earn_yield_snapshots(timestamp);
`

// Migrate creates the tables if they do not exist.
func (s *EarnService) Migrate(ctx context.Context) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	_, err := s.pg.Exec(ctx, earnSchema)
	return err
}

// scanProduct scans a product row (including JSONB token + features) into an EarnProduct.
func scanProduct(row interface {
	Scan(dest ...interface{}) error
}) (*EarnProduct, error) {
	var p EarnProduct
	var tokenJSON, featuresJSON []byte
	if err := row.Scan(&p.ID, &p.Name, &p.Description, &p.ProductType, &p.ChainID, &p.TokenAddress,
		&tokenJSON, &p.APY, &p.APYType, &p.MinDeposit, &p.MaxDeposit, &p.MinTerm, &p.MaxTerm,
		&p.TotalDeposited, &p.TotalValue, &p.MaxCapacity, &p.Status, &featuresJSON,
		&p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	if len(tokenJSON) > 0 && string(tokenJSON) != "null" {
		var tok TokenInfo
		if err := json.Unmarshal(tokenJSON, &tok); err == nil {
			p.Token = &tok
		}
	}
	_ = json.Unmarshal(featuresJSON, &p.Features)
	return &p, nil
}

const productColumns = `id,name,description,product_type,chain_id,token_address,token,
	apy,apy_type,min_deposit,max_deposit,min_term,max_term,total_deposited,total_value,max_capacity,
	status,features,created_at,updated_at`

// ============================================================================
// Product Operations
// ============================================================================

// CreateProduct creates a new earn product
func (s *EarnService) CreateProduct(ctx context.Context, product *EarnProduct) (*EarnProduct, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	product.ID = "earn_" + uuid.New().String()
	product.Status = ProductStatusActive
	product.TotalDeposited = "0"
	product.TotalValue = "0"
	product.CreatedAt = time.Now().Unix()
	product.UpdatedAt = time.Now().Unix()

	featuresJSON, _ := json.Marshal(product.Features)
	var tokenJSON []byte
	if product.Token != nil {
		tokenJSON, _ = json.Marshal(product.Token)
	}

	_, err := s.pg.Exec(ctx, `INSERT INTO earn_products
		(id,name,description,product_type,chain_id,token_address,token,apy,apy_type,
		min_deposit,max_deposit,min_term,max_term,total_deposited,total_value,max_capacity,
		status,features,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		product.ID, product.Name, product.Description, product.ProductType, product.ChainID,
		product.TokenAddress, tokenJSON, product.APY, product.APYType, product.MinDeposit,
		product.MaxDeposit, product.MinTerm, product.MaxTerm, product.TotalDeposited,
		product.TotalValue, product.MaxCapacity, product.Status, featuresJSON,
		product.CreatedAt, product.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return product, nil
}

// GetProduct returns a product by ID
func (s *EarnService) GetProduct(ctx context.Context, productID string) (*EarnProduct, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	row := s.pg.QueryRow(ctx, `SELECT `+productColumns+` FROM earn_products WHERE id=$1`, productID)
	product, err := scanProduct(row)
	if err != nil {
		return nil, fmt.Errorf("product not found")
	}
	return product, nil
}

// GetAllProducts returns all products
func (s *EarnService) GetAllProducts(ctx context.Context, productType string, status string) ([]*EarnProduct, error) {
	if s.pg == nil {
		return []*EarnProduct{}, fmt.Errorf("database not configured")
	}
	q := `SELECT ` + productColumns + ` FROM earn_products`
	args := []interface{}{}
	where := []string{}
	if productType != "" {
		args = append(args, productType)
		where = append(where, fmt.Sprintf("product_type=$%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}
	if len(where) > 0 {
		q += " WHERE " + joinStrings(where, " AND ")
	}
	q += " ORDER BY created_at DESC"

	rows, err := s.pg.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*EarnProduct, 0)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// GetProductsByChain returns products for a chain
func (s *EarnService) GetProductsByChain(ctx context.Context, chainID uint64) ([]*EarnProduct, error) {
	if s.pg == nil {
		return []*EarnProduct{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT `+productColumns+` FROM earn_products WHERE chain_id=$1 ORDER BY created_at DESC`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*EarnProduct, 0)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// UpdateProduct updates a product
func (s *EarnService) UpdateProduct(ctx context.Context, product *EarnProduct) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	featuresJSON, _ := json.Marshal(product.Features)
	now := time.Now().Unix()
	tag, err := s.pg.Exec(ctx, `UPDATE earn_products SET
		name=$1,description=$2,apy=$3,min_deposit=$4,max_deposit=$5,min_term=$6,max_term=$7,
		status=$8,features=$9,updated_at=$10 WHERE id=$11`,
		product.Name, product.Description, product.APY, product.MinDeposit, product.MaxDeposit,
		product.MinTerm, product.MaxTerm, product.Status, featuresJSON, now, product.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	product.UpdatedAt = now
	return nil
}

// UpdateProductStatus updates product status
func (s *EarnService) UpdateProductStatus(ctx context.Context, productID string, status ProductStatus) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	tag, err := s.pg.Exec(ctx, `UPDATE earn_products SET status=$1, updated_at=$2 WHERE id=$3`,
		status, time.Now().Unix(), productID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

// DeleteProduct deletes a product
func (s *EarnService) DeleteProduct(ctx context.Context, productID string) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	tag, err := s.pg.Exec(ctx, `DELETE FROM earn_products WHERE id=$1`, productID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

// ============================================================================
// Deposit Operations
// ============================================================================

// scanDeposit scans a deposit row into a UserDeposit.
func scanDeposit(row interface {
	Scan(dest ...interface{}) error
}) (*UserDeposit, error) {
	var d UserDeposit
	if err := row.Scan(&d.ID, &d.UserID, &d.ProductID, &d.Amount, &d.Principal, &d.Interest,
		&d.APY, &d.Term, &d.StartTime, &d.MaturityTime, &d.ClaimTime, &d.Status, &d.AutoCompound); err != nil {
		return nil, err
	}
	return &d, nil
}

const depositColumns = `id,user_id,product_id,amount,principal,interest,apy,term,
	start_time,maturity_time,claim_time,status,auto_compound`

// Deposit creates a deposit
func (s *EarnService) Deposit(ctx context.Context, deposit *UserDeposit) (*UserDeposit, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Lock product row and load it.
	var productType, apyStr, apyType, minDeposit, maxDeposit, totalDeposited, maxCapacity string
	var featuresJSON []byte
	var maxTerm int64
	err = tx.QueryRow(ctx, `SELECT product_type,apy,apy_type,min_deposit,max_deposit,
		total_deposited,max_capacity,features,max_term FROM earn_products WHERE id=$1 FOR UPDATE`,
		deposit.ProductID).Scan(&productType, &apyStr, &apyType, &minDeposit, &maxDeposit,
		&totalDeposited, &maxCapacity, &featuresJSON, &maxTerm)
	if err != nil {
		return nil, fmt.Errorf("product not found")
	}

	// Validate amount
	amount, ok := new(big.Int).SetString(deposit.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}

	minDep, _ := new(big.Int).SetString(minDeposit, 10)
	if minDep == nil {
		minDep = new(big.Int)
	}
	maxDep, _ := new(big.Int).SetString(maxDeposit, 10)
	if maxDep == nil {
		maxDep = new(big.Int)
	}

	if amount.Cmp(minDep) < 0 {
		return nil, fmt.Errorf("amount below minimum deposit")
	}

	if maxDep.Cmp(big.NewInt(0)) > 0 && amount.Cmp(maxDep) > 0 {
		return nil, fmt.Errorf("amount exceeds maximum deposit")
	}

	// Check capacity
	totalDep, _ := new(big.Int).SetString(totalDeposited, 10)
	if totalDep == nil {
		totalDep = new(big.Int)
	}
	maxCap, _ := new(big.Int).SetString(maxCapacity, 10)
	if maxCap == nil {
		maxCap = new(big.Int)
	}

	if maxCap.Cmp(big.NewInt(0)) > 0 {
		newTotal := new(big.Int).Add(totalDep, amount)
		if newTotal.Cmp(maxCap) > 0 {
			return nil, fmt.Errorf("product capacity exceeded")
		}
	}

	// Get APY based on type
	apy := apyStr
	if apyType == string(APYTypeTiered) {
		apy = s.calculateTieredAPYTx(ctx, tx, deposit.ProductID, amount.String())
	}

	// Create deposit
	deposit.ID = "deposit_" + uuid.New().String()
	deposit.Principal = deposit.Amount
	deposit.Interest = "0"
	deposit.APY = apy
	deposit.StartTime = time.Now().Unix()
	deposit.Status = "active"

	// Set maturity time for fixed products
	if ProductType(productType) == ProductTypeFixed && maxTerm > 0 {
		deposit.Term = maxTerm
		deposit.MaturityTime = deposit.StartTime + maxTerm
	}

	_, err = tx.Exec(ctx, `INSERT INTO earn_deposits
		(id,user_id,product_id,amount,principal,interest,apy,term,start_time,maturity_time,claim_time,status,auto_compound)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		deposit.ID, deposit.UserID, deposit.ProductID, deposit.Amount, deposit.Principal, deposit.Interest,
		deposit.APY, deposit.Term, deposit.StartTime, deposit.MaturityTime, deposit.ClaimTime,
		deposit.Status, deposit.AutoCompound)
	if err != nil {
		return nil, err
	}

	// Update product totals
	totalDep.Add(totalDep, amount)
	if _, err := tx.Exec(ctx, `UPDATE earn_products SET total_deposited=$1, total_value=$2, updated_at=$3 WHERE id=$4`,
		totalDep.String(), totalDep.String(), time.Now().Unix(), deposit.ProductID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return deposit, nil
}

// calculateTieredAPY calculates APY based on tier (pool path).
func (s *EarnService) calculateTieredAPY(ctx context.Context, productID string, amount string) string {
	if s.pg == nil {
		return "0"
	}
	return s.calculateTieredAPYTx(ctx, s.pg, productID, amount)
}

// queryer is implemented by both *pgxpool.Pool and pgx.Tx.
type queryer interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

// calculateTieredAPYTx calculates APY based on tier using the given executor (pool or tx).
func (s *EarnService) calculateTieredAPYTx(ctx context.Context, q queryer, productID string, amount string) string {
	amountInt, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return "0"
	}
	rows, err := q.Query(ctx, `SELECT min_amount,max_amount,apy FROM earn_tiers WHERE product_id=$1 ORDER BY rank ASC`, productID)
	if err != nil {
		return "0"
	}
	defer rows.Close()
	for rows.Next() {
		var minA, maxA, apy string
		if err := rows.Scan(&minA, &maxA, &apy); err != nil {
			return "0"
		}
		minAmount, _ := new(big.Int).SetString(minA, 10)
		if minAmount == nil {
			minAmount = new(big.Int)
		}
		maxAmount, _ := new(big.Int).SetString(maxA, 10)
		if maxAmount == nil {
			maxAmount = new(big.Int)
		}
		if amountInt.Cmp(minAmount) >= 0 {
			if maxAmount.Cmp(big.NewInt(0)) == 0 || amountInt.Cmp(maxAmount) <= 0 {
				return apy
			}
		}
	}
	return "0"
}

// GetDeposit returns a deposit
func (s *EarnService) GetDeposit(ctx context.Context, depositID string) (*UserDeposit, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	row := s.pg.QueryRow(ctx, `SELECT `+depositColumns+` FROM earn_deposits WHERE id=$1`, depositID)
	d, err := scanDeposit(row)
	if err != nil {
		return nil, fmt.Errorf("deposit not found")
	}
	return d, nil
}

// GetUserDeposits returns all deposits for a user
func (s *EarnService) GetUserDeposits(ctx context.Context, userID string) ([]*UserDeposit, error) {
	if s.pg == nil {
		return []*UserDeposit{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT `+depositColumns+` FROM earn_deposits WHERE user_id=$1 ORDER BY start_time DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*UserDeposit, 0)
	for rows.Next() {
		d, err := scanDeposit(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// GetUserActiveDeposits returns active deposits for a user
func (s *EarnService) GetUserActiveDeposits(ctx context.Context, userID string) ([]*UserDeposit, error) {
	if s.pg == nil {
		return []*UserDeposit{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT `+depositColumns+` FROM earn_deposits WHERE user_id=$1 AND status='active' ORDER BY start_time DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*UserDeposit, 0)
	for rows.Next() {
		d, err := scanDeposit(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// GetUserDepositsByProduct returns deposits for a user in a product
func (s *EarnService) GetUserDepositsByProduct(ctx context.Context, userID, productID string) ([]*UserDeposit, error) {
	if s.pg == nil {
		return []*UserDeposit{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT `+depositColumns+` FROM earn_deposits WHERE user_id=$1 AND product_id=$2 ORDER BY start_time DESC`, userID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*UserDeposit, 0)
	for rows.Next() {
		d, err := scanDeposit(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// ============================================================================
// Withdrawal Operations
// ============================================================================

// Withdraw withdraws from a deposit
func (s *EarnService) Withdraw(ctx context.Context, depositID string, amount string) (string, error) {
	if s.pg == nil {
		return "0", fmt.Errorf("database not configured")
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "0", err
	}
	defer tx.Rollback(ctx)

	// Lock deposit row.
	var deposit UserDeposit
	err = tx.QueryRow(ctx, `SELECT `+depositColumns+` FROM earn_deposits WHERE id=$1 FOR UPDATE`, depositID).
		Scan(&deposit.ID, &deposit.UserID, &deposit.ProductID, &deposit.Amount, &deposit.Principal,
			&deposit.Interest, &deposit.APY, &deposit.Term, &deposit.StartTime, &deposit.MaturityTime,
			&deposit.ClaimTime, &deposit.Status, &deposit.AutoCompound)
	if err != nil {
		return "0", fmt.Errorf("deposit not found")
	}
	if deposit.Status != "active" {
		return "0", fmt.Errorf("deposit not active")
	}

	// Lock product row.
	var productType, totalDepositedStr string
	var featuresJSON []byte
	err = tx.QueryRow(ctx, `SELECT product_type,features,total_deposited FROM earn_products WHERE id=$1 FOR UPDATE`,
		deposit.ProductID).Scan(&productType, &featuresJSON, &totalDepositedStr)
	if err != nil {
		return "0", fmt.Errorf("product not found")
	}

	var features []string
	_ = json.Unmarshal(featuresJSON, &features)

	// Check if early withdrawal is allowed
	isEarlyWithdrawal := false
	for _, feature := range features {
		if feature == "early-withdrawal" {
			isEarlyWithdrawal = true
			break
		}
	}

	// For fixed products, check maturity
	if ProductType(productType) == ProductTypeFixed {
		if time.Now().Unix() < deposit.MaturityTime && !isEarlyWithdrawal {
			return "0", fmt.Errorf("deposit not yet matured")
		}
	}

	// Calculate interest
	withdrawAmount, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return "0", fmt.Errorf("invalid number")
	}

	principal, _ := new(big.Int).SetString(deposit.Principal, 10)
	if principal == nil {
		principal = new(big.Int)
	}
	interest, _ := new(big.Int).SetString(deposit.Interest, 10)
	if interest == nil {
		interest = new(big.Int)
	}

	if withdrawAmount.Cmp(principal) > 0 {
		withdrawAmount = principal
	}

	// Calculate proportional interest (avoid divide-by-zero).
	var claimedInterest *big.Int
	if principal.Cmp(big.NewInt(0)) > 0 {
		interestRatio := new(big.Int).Div(withdrawAmount, principal)
		claimedInterest = new(big.Int).Mul(interest, interestRatio)
	} else {
		claimedInterest = new(big.Int)
	}

	now := time.Now().Unix()
	// Update deposit
	if withdrawAmount.Cmp(principal) == 0 {
		if _, err := tx.Exec(ctx, `UPDATE earn_deposits SET status='withdrawn', claim_time=$1 WHERE id=$2`, now, depositID); err != nil {
			return "0", err
		}
	} else {
		principal.Sub(principal, withdrawAmount)
		interest.Sub(interest, claimedInterest)
		if _, err := tx.Exec(ctx, `UPDATE earn_deposits SET principal=$1, interest=$2 WHERE id=$3`,
			principal.String(), interest.String(), depositID); err != nil {
			return "0", err
		}
	}

	// Update product total
	totalDeposited, _ := new(big.Int).SetString(totalDepositedStr, 10)
	if totalDeposited == nil {
		totalDeposited = new(big.Int)
	}
	totalDeposited.Sub(totalDeposited, withdrawAmount)
	if _, err := tx.Exec(ctx, `UPDATE earn_products SET total_deposited=$1, total_value=$2, updated_at=$3 WHERE id=$4`,
		totalDeposited.String(), totalDeposited.String(), now, deposit.ProductID); err != nil {
		return "0", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "0", err
	}
	return claimedInterest.String(), nil
}

// Claim claims matured deposit
func (s *EarnService) Claim(ctx context.Context, depositID string) (string, error) {
	if s.pg == nil {
		return "0", fmt.Errorf("database not configured")
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "0", err
	}
	defer tx.Rollback(ctx)

	// Lock deposit row.
	var deposit UserDeposit
	err = tx.QueryRow(ctx, `SELECT `+depositColumns+` FROM earn_deposits WHERE id=$1 FOR UPDATE`, depositID).
		Scan(&deposit.ID, &deposit.UserID, &deposit.ProductID, &deposit.Amount, &deposit.Principal,
			&deposit.Interest, &deposit.APY, &deposit.Term, &deposit.StartTime, &deposit.MaturityTime,
			&deposit.ClaimTime, &deposit.Status, &deposit.AutoCompound)
	if err != nil {
		return "0", fmt.Errorf("deposit not found")
	}
	if deposit.Status == "claimed" {
		return "0", fmt.Errorf("already claimed")
	}

	// Verify product exists (locked).
	var productType string
	err = tx.QueryRow(ctx, `SELECT product_type FROM earn_products WHERE id=$1 FOR UPDATE`, deposit.ProductID).Scan(&productType)
	if err != nil {
		return "0", fmt.Errorf("product not found")
	}

	// Calculate interest
	interest, err := s.calculateInterestTx(&deposit)
	if err != nil {
		return "0", err
	}

	now := time.Now().Unix()
	deposit.Interest = interest

	// If auto-compound, add interest to principal and keep active; else mark matured.
	if deposit.AutoCompound {
		principal, _ := new(big.Int).SetString(deposit.Principal, 10)
		if principal == nil {
			principal = new(big.Int)
		}
		interestInt, _ := new(big.Int).SetString(interest, 10)
		if interestInt == nil {
			interestInt = new(big.Int)
		}
		principal.Add(principal, interestInt)
		if _, err := tx.Exec(ctx, `UPDATE earn_deposits SET principal=$1, interest=$2, status='active', claim_time=$3 WHERE id=$4`,
			principal.String(), interest, now, depositID); err != nil {
			return "0", err
		}
	} else {
		if _, err := tx.Exec(ctx, `UPDATE earn_deposits SET interest=$1, status='matured', claim_time=$2 WHERE id=$3`,
			interest, now, depositID); err != nil {
			return "0", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "0", err
	}
	return interest, nil
}

// calculateInterest calculates interest for a deposit.
func (s *EarnService) calculateInterest(deposit *UserDeposit) (string, error) {
	principal, _ := new(big.Int).SetString(deposit.Principal, 10)
	if principal == nil {
		principal = new(big.Int)
	}
	apy, _ := new(big.Int).SetString(deposit.APY, 10)
	if apy == nil {
		apy = new(big.Int)
	}

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

// calculateInterestTx mirrors calculateInterest; kept for clarity within tx scope.
func (s *EarnService) calculateInterestTx(deposit *UserDeposit) (string, error) {
	return s.calculateInterest(deposit)
}

// ============================================================================
// Tier Operations
// ============================================================================

// SetTiers sets APY tiers for a product
func (s *EarnService) SetTiers(ctx context.Context, productID string, tiers []Tier) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Verify product exists.
	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM earn_products WHERE id=$1)`, productID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("product not found")
	}

	// Replace tiers.
	if _, err := tx.Exec(ctx, `DELETE FROM earn_tiers WHERE product_id=$1`, productID); err != nil {
		return err
	}
	for i, tier := range tiers {
		tierID := "tier_" + uuid.New().String()
		if _, err := tx.Exec(ctx, `INSERT INTO earn_tiers (id,product_id,min_amount,max_amount,apy,rank)
			VALUES ($1,$2,$3,$4,$5,$6)`, tierID, productID, tier.MinAmount, tier.MaxAmount, tier.APY, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// GetTiers returns APY tiers for a product
func (s *EarnService) GetTiers(ctx context.Context, productID string) ([]Tier, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT min_amount,max_amount,apy FROM earn_tiers WHERE product_id=$1 ORDER BY rank ASC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Tier, 0)
	for rows.Next() {
		var t Tier
		if err := rows.Scan(&t.MinAmount, &t.MaxAmount, &t.APY); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("tiers not found")
	}
	return result, nil
}

// ============================================================================
// Utility Methods
// ============================================================================

// CalculateProjectedReturns calculates projected returns
func (s *EarnService) CalculateProjectedReturns(ctx context.Context, productID, amount, term string) (string, error) {
	if s.pg == nil {
		return "0", fmt.Errorf("database not configured")
	}
	var apyStr string
	err := s.pg.QueryRow(ctx, `SELECT apy FROM earn_products WHERE id=$1`, productID).Scan(&apyStr)
	if err != nil {
		return "0", fmt.Errorf("product not found")
	}

	amountInt, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return "0", fmt.Errorf("invalid number")
	}

	termInt, ok := new(big.Int).SetString(term, 10)
	if !ok {
		return "0", fmt.Errorf("invalid number")
	}

	// Get APY
	apy, ok := new(big.Int).SetString(apyStr, 10)
	if !ok {
		return "0", fmt.Errorf("invalid number")
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
	if s.pg == nil {
		return "0", fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT interest FROM earn_deposits WHERE user_id=$1`, userID)
	if err != nil {
		return "0", err
	}
	defer rows.Close()

	totalEarnings := big.NewInt(0)
	for rows.Next() {
		var interestStr string
		if err := rows.Scan(&interestStr); err != nil {
			return "0", err
		}
		interest, _ := new(big.Int).SetString(interestStr, 10)
		if interest == nil {
			interest = new(big.Int)
		}
		totalEarnings.Add(totalEarnings, interest)
	}
	return totalEarnings.String(), rows.Err()
}

// ToJSON converts deposit to JSON
func (d *UserDeposit) ToJSON() (string, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// joinStrings joins strings with the given separator (small helper to avoid
// importing strings solely for this).
func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}
