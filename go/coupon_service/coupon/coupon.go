/**
 * TigerWallet Coupon Service
 *
 * Promo codes and discounts management.
 * Built with Go for high-load distributed operations.
 * PostgreSQL-backed — all coupons and usages are persisted.
 */

package coupon

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Coupon represents a coupon
type Coupon struct {
	ID               string   `json:"id"`
	Code             string   `json:"code"`
	Type             string   `json:"type"` // percentage, fixed, fee_discount
	Value            string   `json:"value"`
	MinAmount        string   `json:"min_amount"`
	MaxUses          int      `json:"max_uses"`
	UsedCount        int      `json:"used_count"`
	ValidFrom        int64    `json:"valid_from"`
	ValidUntil       int64    `json:"valid_until"`
	ApplicableChains []string `json:"applicable_chains"`
	ApplicablePairs  []string `json:"applicable_pairs"`
	Status           string   `json:"status"`
	CreatedAt        int64    `json:"created_at"`
}

// CouponUsage represents coupon usage
type CouponUsage struct {
	ID       string `json:"id"`
	CouponID string `json:"coupon_id"`
	UserID   string `json:"user_id"`
	OrderID  string `json:"order_id"`
	Discount string `json:"discount"`
	UsedAt   int64  `json:"used_at"`
}

// CouponService manages coupon operations backed by PostgreSQL.
type CouponService struct {
	pg *pgxpool.Pool
}

var couponService *CouponService

func NewCouponService(pg *pgxpool.Pool) *CouponService {
	return &CouponService{pg: pg}
}

func GetCouponService() *CouponService {
	if couponService != nil {
		return couponService
	}
	return &CouponService{}
}

func SetCouponService(pg *pgxpool.Pool) {
	couponService = NewCouponService(pg)
}

const couponSchema = `
CREATE TABLE IF NOT EXISTS coupons (
    id                TEXT PRIMARY KEY,
    code              TEXT NOT NULL UNIQUE,
    type              TEXT NOT NULL DEFAULT 'fixed',
    value             TEXT NOT NULL DEFAULT '0',
    min_amount        TEXT NOT NULL DEFAULT '0',
    max_uses          INTEGER NOT NULL DEFAULT 0,
    used_count        INTEGER NOT NULL DEFAULT 0,
    valid_from        BIGINT NOT NULL DEFAULT 0,
    valid_until       BIGINT NOT NULL DEFAULT 0,
    applicable_chains JSONB NOT NULL DEFAULT '[]'::jsonb,
    applicable_pairs  JSONB NOT NULL DEFAULT '[]'::jsonb,
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        BIGINT NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS coupon_usages (
    id        TEXT PRIMARY KEY,
    coupon_id TEXT NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
    user_id   TEXT NOT NULL DEFAULT '',
    order_id  TEXT NOT NULL DEFAULT '',
    discount  TEXT NOT NULL DEFAULT '0',
    used_at   BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_coupon_usages_user ON coupon_usages(user_id);
`

func (s *CouponService) Migrate(ctx context.Context) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	_, err := s.pg.Exec(ctx, couponSchema)
	return err
}

func (s *CouponService) CreateCoupon(ctx context.Context, coupon *Coupon) (*Coupon, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	var exists bool
	err := s.pg.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM coupons WHERE code=$1)`, coupon.Code).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("coupon code already exists")
	}

	coupon.ID = "coupon_" + uuid.New().String()
	coupon.UsedCount = 0
	coupon.Status = "active"
	coupon.CreatedAt = time.Now().Unix()

	chainsJSON, _ := json.Marshal(coupon.ApplicableChains)
	pairsJSON, _ := json.Marshal(coupon.ApplicablePairs)
	_, err = s.pg.Exec(ctx, `INSERT INTO coupons
		(id,code,type,value,min_amount,max_uses,used_count,valid_from,valid_until,applicable_chains,applicable_pairs,status,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		coupon.ID, coupon.Code, coupon.Type, coupon.Value, coupon.MinAmount, coupon.MaxUses, coupon.UsedCount,
		coupon.ValidFrom, coupon.ValidUntil, chainsJSON, pairsJSON, coupon.Status, coupon.CreatedAt)
	if err != nil {
		return nil, err
	}
	return coupon, nil
}

func (s *CouponService) scanCoupon(row interface {
	Scan(dest ...interface{}) error
}) (*Coupon, error) {
	var c Coupon
	var chainsJSON, pairsJSON []byte
	if err := row.Scan(&c.ID, &c.Code, &c.Type, &c.Value, &c.MinAmount, &c.MaxUses, &c.UsedCount,
		&c.ValidFrom, &c.ValidUntil, &chainsJSON, &pairsJSON, &c.Status, &c.CreatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(chainsJSON, &c.ApplicableChains)
	_ = json.Unmarshal(pairsJSON, &c.ApplicablePairs)
	return &c, nil
}

func (s *CouponService) GetCoupon(ctx context.Context, code string) (*Coupon, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	row := s.pg.QueryRow(ctx, `SELECT id,code,type,value,min_amount,max_uses,used_count,valid_from,valid_until,
		applicable_chains,applicable_pairs,status,created_at FROM coupons WHERE code=$1`, code)
	coupon, err := s.scanCoupon(row)
	if err != nil {
		return nil, fmt.Errorf("coupon not found")
	}
	now := time.Now().Unix()
	if coupon.ValidFrom > 0 && now < coupon.ValidFrom {
		return nil, fmt.Errorf("coupon not yet valid")
	}
	if coupon.ValidUntil > 0 && now > coupon.ValidUntil {
		return nil, fmt.Errorf("coupon expired")
	}
	if coupon.Status != "active" {
		return nil, fmt.Errorf("coupon not active")
	}
	if coupon.MaxUses > 0 && coupon.UsedCount >= coupon.MaxUses {
		return nil, fmt.Errorf("coupon usage limit reached")
	}
	return coupon, nil
}

func (s *CouponService) ValidateCoupon(ctx context.Context, code, chainID, pair string) (*Coupon, error) {
	coupon, err := s.GetCoupon(ctx, code)
	if err != nil {
		return nil, err
	}
	if len(coupon.ApplicableChains) > 0 {
		found := false
		for _, chain := range coupon.ApplicableChains {
			if chain == chainID {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("coupon not applicable for this chain")
		}
	}
	if len(coupon.ApplicablePairs) > 0 {
		found := false
		for _, p := range coupon.ApplicablePairs {
			if p == pair {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("coupon not applicable for this pair")
		}
	}
	return coupon, nil
}

func (s *CouponService) ApplyCoupon(ctx context.Context, code, userID, orderID, orderAmount string) (string, error) {
	if s.pg == nil {
		return "", fmt.Errorf("database not configured")
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var couponID, ctype, value, minAmount string
	var maxUses, usedCount int
	row := tx.QueryRow(ctx, `SELECT id,type,value,min_amount,max_uses,used_count FROM coupons WHERE code=$1 AND status='active' FOR UPDATE`, code)
	if err := row.Scan(&couponID, &ctype, &value, &minAmount, &maxUses, &usedCount); err != nil {
		return "", fmt.Errorf("coupon not found or not active")
	}
	if maxUses > 0 && usedCount >= maxUses {
		return "", fmt.Errorf("coupon usage limit reached")
	}
	if minAmount != "" {
		minAmt, _ := new(big.Int).SetString(minAmount, 10)
		orderAmt, _ := new(big.Int).SetString(orderAmount, 10)
		if minAmt != nil && orderAmt != nil && orderAmt.Cmp(minAmt) < 0 {
			return "", fmt.Errorf("order amount below minimum")
		}
	}

	var discount string
	switch ctype {
	case "percentage":
		orderAmt, _ := new(big.Int).SetString(orderAmount, 10)
		val, _ := new(big.Int).SetString(value, 10)
		discount = new(big.Int).Div(new(big.Int).Mul(orderAmt, val), big.NewInt(10000)).String()
	case "fixed", "fee_discount":
		discount = value
	default:
		discount = "0"
	}

	usageID := "usage_" + uuid.New().String()
	if _, err := tx.Exec(ctx, `INSERT INTO coupon_usages (id,coupon_id,user_id,order_id,discount,used_at)
		VALUES ($1,$2,$3,$4,$5,$6)`, usageID, couponID, userID, orderID, discount, time.Now().Unix()); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE coupons SET used_count=used_count+1 WHERE id=$1`, couponID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return discount, nil
}

func (s *CouponService) GetUserUsages(ctx context.Context, userID string) ([]*CouponUsage, error) {
	if s.pg == nil {
		return []*CouponUsage{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT id,coupon_id,user_id,order_id,discount,used_at
		FROM coupon_usages WHERE user_id=$1 ORDER BY used_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*CouponUsage, 0)
	for rows.Next() {
		var u CouponUsage
		if err := rows.Scan(&u.ID, &u.CouponID, &u.UserID, &u.OrderID, &u.Discount, &u.UsedAt); err != nil {
			return nil, err
		}
		result = append(result, &u)
	}
	return result, rows.Err()
}

func (c *Coupon) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
