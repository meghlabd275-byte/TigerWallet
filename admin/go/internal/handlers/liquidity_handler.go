/**
 * TigerWallet Admin - Liquidity Handler
 * Complete backend implementation for liquidity pool management
 */

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LiquidityHandler struct {
	db *gorm.DB
}

func NewLiquidityHandler(db *gorm.DB) *LiquidityHandler {
	if err := db.AutoMigrate(&LiquidityPool{}, &LiquidityPosition{}); err != nil {
		// log but do not panic; tables may already exist
		_ = err
	}
	return &LiquidityHandler{db: db}
}

// LiquidityPool model
type LiquidityPool struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Pair        string    `gorm:"uniqueIndex" json:"pair"`
	TokenA      string    `json:"token_a"`
	TokenB      string    `json:"token_b"`
	ReserveA    float64   `json:"reserve_a"`
	ReserveB    float64   `json:"reserve_b"`
	TotalSupply float64   `json:"total_supply"`
	APR         float64   `json:"apr"`
	Volume24h   float64   `json:"volume_24h"`
	Fees24h     float64   `json:"fees_24h"`
	Status      string    `gorm:"default:'active'" json:"status"` // active, inactive
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (LiquidityPool) TableName() string {
	return "liquidity_pools"
}

// LiquidityPosition model
type LiquidityPosition struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	PoolID        uint      `gorm:"index" json:"pool_id"`
	UserID        uint      `gorm:"index" json:"user_id"`
	LPTokenAmount float64   `json:"lp_token_amount"`
	ReserveA      float64   `json:"reserve_a"`
	ReserveB      float64   `json:"reserve_b"`
	CreatedAt     time.Time `json:"created_at"`
}

func (LiquidityPosition) TableName() string {
	return "liquidity_positions"
}

// GetPools handles GET /liquidity/pools
func (h *LiquidityHandler) GetPools(c *gin.Context) {
	var pools []LiquidityPool
	if err := h.db.Order("created_at DESC").Find(&pools).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pools)
}

// GetPoolByID handles GET /liquidity/pools/:id
func (h *LiquidityHandler) GetPoolByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pool id"})
		return
	}

	var pool LiquidityPool
	if err := h.db.First(&pool, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "pool not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pool)
}

// CreatePool handles POST /liquidity/pools
func (h *LiquidityHandler) CreatePool(c *gin.Context) {
	var pool LiquidityPool
	if err := c.ShouldBindJSON(&pool); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pool.Status = "active"
	if err := h.db.Create(&pool).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, pool)
}

// AddLiquidity handles POST /liquidity/pools/:id/add
func (h *LiquidityHandler) AddLiquidity(c *gin.Context) {
	poolID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pool id"})
		return
	}

	var input struct {
		UserID  uint    `json:"user_id" binding:"required"`
		AmountA float64 `json:"amount_a" binding:"required"`
		AmountB float64 `json:"amount_b" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get pool
	var pool LiquidityPool
	if err := h.db.First(&pool, poolID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pool not found"})
		return
	}

	// Calculate LP tokens to mint (simplified)
	lpTokens := (input.AmountA + input.AmountB) / 2

	// Update pool reserves
	h.db.Model(&pool).Updates(map[string]interface{}{
		"reserve_a":    pool.ReserveA + input.AmountA,
		"reserve_b":    pool.ReserveB + input.AmountB,
		"total_supply": pool.TotalSupply + lpTokens,
	})

	// Create liquidity position
	position := LiquidityPosition{
		PoolID:        uint(poolID),
		UserID:        input.UserID,
		LPTokenAmount: lpTokens,
		ReserveA:      input.AmountA,
		ReserveB:      input.AmountB,
	}
	h.db.Create(&position)

	c.JSON(http.StatusOK, gin.H{
		"message":     "liquidity added successfully",
		"lp_tokens":   lpTokens,
		"position_id": position.ID,
	})
}

// RemoveLiquidity handles POST /liquidity/pools/:id/remove
func (h *LiquidityHandler) RemoveLiquidity(c *gin.Context) {
	poolID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pool id"})
		return
	}

	var input struct {
		UserID uint    `json:"user_id" binding:"required"`
		Amount float64 `json:"amount" binding:"required"` // LP token amount
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get pool
	var pool LiquidityPool
	if err := h.db.First(&pool, poolID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pool not found"})
		return
	}

	// Get user position
	var position LiquidityPosition
	if err := h.db.Where("pool_id = ? AND user_id = ?", poolID, input.UserID).First(&position).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}

	// Calculate proportional amounts
	ratio := input.Amount / position.LPTokenAmount
	amountA := position.ReserveA * ratio
	amountB := position.ReserveB * ratio

	// Update pool
	h.db.Model(&pool).Updates(map[string]interface{}{
		"reserve_a":    pool.ReserveA - amountA,
		"reserve_b":    pool.ReserveB - amountB,
		"total_supply": pool.TotalSupply - input.Amount,
	})

	// Update position
	h.db.Model(&position).Updates(map[string]interface{}{
		"lp_token_amount": position.LPTokenAmount - input.Amount,
		"reserve_a":       position.ReserveA - amountA,
		"reserve_b":       position.ReserveB - amountB,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":  "liquidity removed successfully",
		"amount_a": amountA,
		"amount_b": amountB,
	})
}

// GetStats handles GET /liquidity/stats
func (h *LiquidityHandler) GetStats(c *gin.Context) {
	var totalPools int64
	var totalValueLocked float64
	var volume24h float64
	var fees24h float64

	h.db.Model(&LiquidityPool{}).Count(&totalPools)
	h.db.Model(&LiquidityPool{}).Select("COALESCE(SUM(reserve_a + reserve_b), 0)").Scan(&totalValueLocked)
	h.db.Model(&LiquidityPool{}).Select("COALESCE(SUM(volume_24h), 0)").Scan(&volume24h)
	h.db.Model(&LiquidityPool{}).Select("COALESCE(SUM(fees_24h), 0)").Scan(&fees24h)

	c.JSON(http.StatusOK, gin.H{
		"total_pools":        totalPools,
		"total_value_locked": totalValueLocked,
		"volume_24h":         volume24h,
		"fees_24h":           fees24h,
	})
}
