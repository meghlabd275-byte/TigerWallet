/**
 * TigerWallet Coupon Service
 * 
 * Promo codes and discounts management.
 * Built with Go for high-load distributed operations.
 */

package coupon

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Coupon represents a coupon
type Coupon struct {
	ID              string    `json:"id"`
	Code            string    `json:"code"`
	Type            string    `json:"type"` // percentage, fixed, fee_discount
	Value           string    `json:"value"`
	MinAmount       string    `json:"min_amount"`
	MaxUses         int       `json:"max_uses"`
	UsedCount       int       `json:"used_count"`
	ValidFrom       int64     `json:"valid_from"`
	ValidUntil      int64     `json:"valid_until"`
	ApplicableChains []string `json:"applicable_chains"`
	ApplicablePairs  []string `json:"applicable_pairs"`
	Status          string    `json:"status"`
	CreatedAt       int64     `json:"created_at"`
}

// CouponUsage represents coupon usage
type CouponUsage struct {
	ID          string    `json:"id"`
	CouponID    string    `json:"coupon_id"`
	UserID      string    `json:"user_id"`
	OrderID     string    `json:"order_id"`
	Discount    string    `json:"discount"`
	UsedAt      int64     `json:"used_at"`
}

// CouponService manages coupon operations
type CouponService struct {
	mu     sync.RWMutex
	coupons map[string]*Coupon
	usages  map[string]*CouponUsage
}

var (
	couponService     *CouponService
	couponServiceOnce sync.Once
)

func GetCouponService() *CouponService {
	couponServiceOnce.Do(func() {
		couponService = &CouponService{
			coupons: make(map[string]*Coupon),
			usages:  make(map[string]*CouponUsage),
		}
	})
	return couponService
}

func (s *CouponService) CreateCoupon(ctx context.Context, coupon *Coupon) (*Coupon, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if code exists
	for _, c := range s.coupons {
		if c.Code == coupon.Code {
			return nil, fmt.Errorf("coupon code already exists")
		}
	}

	coupon.ID = "coupon_" + uuid.New().String()
	coupon.UsedCount = 0
	coupon.Status = "active"
	coupon.CreatedAt = time.Now().Unix()

	s.coupons[coupon.ID] = coupon
	return coupon, nil
}

func (s *CouponService) GetCoupon(ctx context.Context, code string) (*Coupon, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, coupon := range s.coupons {
		if coupon.Code == code {
			// Check if valid
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
	}
	return nil, fmt.Errorf("coupon not found")
}

func (s *CouponService) ValidateCoupon(ctx context.Context, code, chainID, pair string) (*Coupon, error) {
	coupon, err := s.GetCoupon(ctx, code)
	if err != nil {
		return nil, err
	}

	// Check chain applicability
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

	// Check pair applicability
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
	s.mu.Lock()
	defer s.mu.Unlock()

	coupon, err := s.GetCoupon(ctx, code)
	if err != nil {
		return "", err
	}

	// Check min amount
	if coupon.MinAmount != "" {
		minAmt, _ := new(big.Int).SetString(coupon.MinAmount, 10)
		orderAmt, _ := new(big.Int).SetString(orderAmount, 10)
		if orderAmt.Cmp(minAmt) < 0 {
			return "", fmt.Errorf("order amount below minimum")
		}
	}

	// Calculate discount
	var discount string
	switch coupon.Type {
	case "percentage":
		orderAmt, _ := new(big.Int).SetString(orderAmount, 10)
		value, _ := new(big.Int).SetString(coupon.Value, 10)
		discount = new(big.Int).Div(new(big.Int).Mul(orderAmt, value), big.NewInt(10000))
	case "fixed":
		discount = coupon.Value
	case "fee_discount":
		discount = coupon.Value
	default:
		discount = "0"
	}

	// Record usage
	usage := &CouponUsage{
		ID:       "usage_" + uuid.New().String(),
		CouponID: coupon.ID,
		UserID:   userID,
		OrderID:  orderID,
		Discount: discount,
		UsedAt:   time.Now().Unix(),
	}
	s.usages[usage.ID] = usage

	// Update coupon
	coupon.UsedCount++

	return discount, nil
}

func (s *CouponService) GetUserUsages(ctx context.Context, userID string) ([]*CouponUsage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*CouponUsage, 0)
	for _, usage := range s.usages {
		if usage.UserID == userID {
			result = append(result, usage)
		}
	}
	return result, nil
}

func (c *Coupon) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
