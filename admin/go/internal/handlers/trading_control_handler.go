/**
 * TigerWallet Admin - Trading Control-Plane Handler
 *
 * Owner policy: SuperAdmin, white-label clients, and RBAC admins can
 * create / add / remove / stop / resume trading contracts, liquidity pools,
 * trading pairs, margin markets, options series, and copy-trading configs,
 * plus halt/resume whole trading verticals. Everything is builtin
 * TigerWallet — control state is published to the shared Redis namespace
 * ("trading:control:global:*") that the wallet engines (go/wallet_api,
 * master_wallet) enforce on, exactly like the feature-flag plane. No
 * external broker or exchange is involved.
 *
 * Write access is gated by DomainScopeMiddleware("trading_control"):
 * super_admin + admin roles manage; support/analyst/moderator are read-only.
 */

package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/tigerwallet/admin/internal/models"
	"gorm.io/gorm"
)

type TradingControlHandler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewTradingControlHandler(db *gorm.DB, rdb *redis.Client) *TradingControlHandler {
	if err := db.AutoMigrate(&TradingContract{}, &MarginMarket{}, &TradingControlAudit{}); err != nil {
		_ = err // tables may already exist (created by super_admin/wallet_api)
	}
	return &TradingControlHandler{db: db, rdb: rdb}
}

// ---- Models ----

type TradingContract struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Kind        string    `gorm:"not null;uniqueIndex:ux_tc_kind_symbol" json:"kind"` // perpetual|futures|options
	Symbol      string    `gorm:"not null;uniqueIndex:ux_tc_kind_symbol" json:"symbol"`
	BaseAsset   string    `gorm:"not null" json:"base_asset"`
	QuoteAsset  string    `gorm:"not null" json:"quote_asset"`
	ChainID     int64     `gorm:"default:0" json:"chain_id"`
	MaxLeverage int       `gorm:"default:1" json:"max_leverage"`
	MinSize     string    `gorm:"default:'0'" json:"min_size"`
	TickSize    string    `gorm:"default:'0'" json:"tick_size"`
	Status      string    `gorm:"default:'active'" json:"status"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (TradingContract) TableName() string { return "trading_contracts" }

type MarginMarket struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Symbol      string    `gorm:"not null;uniqueIndex" json:"symbol"`
	BaseAsset   string    `gorm:"not null" json:"base_asset"`
	QuoteAsset  string    `gorm:"not null" json:"quote_asset"`
	MaxLeverage int       `gorm:"default:3" json:"max_leverage"`
	BorrowCap   string    `gorm:"default:'0'" json:"borrow_cap"`
	Status      string    `gorm:"default:'active'" json:"status"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (MarginMarket) TableName() string { return "margin_markets" }

type TradingControlAudit struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Actor     string    `json:"actor"`
	ActorRole string    `json:"actor_role"`
	Action    string    `gorm:"not null" json:"action"`
	Kind      string    `gorm:"not null" json:"kind"`
	Entity    string    `gorm:"not null" json:"entity"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

func (TradingControlAudit) TableName() string { return "trading_control_audit" }

// ---- helpers ----

var tcVerticals = map[string]bool{
	"spot": true, "perpetual": true, "futures": true, "margin": true,
	"options": true, "copy": true, "liquidity": true,
}

func tcValidStatus(s string) bool {
	switch s {
	case "active", "stopped", "removed", "suspended":
		return true
	}
	return false
}

func (h *TradingControlHandler) publish(kind, key, status string) {
	if h.rdb == nil || key == "" {
		return
	}
	ctx := context.Background()
	h.rdb.Set(ctx, "trading:control:global:"+kind+":"+strings.ToUpper(key), status, 0)
}

func (h *TradingControlHandler) audit(c *gin.Context, action, kind, entity, detail string) {
	actor := fmt.Sprint(c.Value("admin_id"))
	role := fmt.Sprint(c.Value("admin_role"))
	h.db.Create(&TradingControlAudit{Actor: actor, ActorRole: role, Action: action, Kind: kind, Entity: entity, Detail: detail})
}

// genericStatus transitions any governed entity row and publishes the flag.
func (h *TradingControlHandler) genericStatus(c *gin.Context, model interface{}, kind, keyColumn string) {
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !tcValidStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be active|stopped|removed|suspended"})
		return
	}
	id := c.Param("id")
	res := h.db.Model(model).Where("id = ?", id).Update("status", req.Status)
	if res.Error != nil || res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
		return
	}
	var entity string
	h.db.Model(model).Where("id = ?", id).Select(keyColumn).Scan(&entity)
	h.publish(kind, entity, req.Status)
	h.audit(c, req.Status, kind, entity, "")
	c.JSON(http.StatusOK, gin.H{"message": kind + " " + req.Status, "entity": entity, "status": req.Status})
}

func (h *TradingControlHandler) genericDelete(c *gin.Context, model interface{}, kind, keyColumn string) {
	id := c.Param("id")
	var entity string
	h.db.Model(model).Where("id = ?", id).Select(keyColumn).Scan(&entity)
	res := h.db.Where("id = ?", id).Delete(model)
	if res.Error != nil || res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
		return
	}
	h.publish(kind, entity, "removed")
	h.audit(c, "remove", kind, entity, "")
	c.JSON(http.StatusOK, gin.H{"message": kind + " removed"})
}

// ---- Trading contracts ----

func (h *TradingControlHandler) ListContracts(c *gin.Context) {
	var items []TradingContract
	if err := h.db.Order("created_at DESC").Limit(500).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"contracts": items})
}

func (h *TradingControlHandler) CreateContract(c *gin.Context) {
	var req TradingContract
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Kind = strings.ToLower(req.Kind)
	if req.Kind != "perpetual" && req.Kind != "futures" && req.Kind != "options" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind must be perpetual|futures|options"})
		return
	}
	if req.MaxLeverage < 1 {
		req.MaxLeverage = 1
	}
	req.Symbol = strings.ToUpper(req.Symbol)
	req.Status = "active"
	req.CreatedBy = fmt.Sprint(c.Value("admin_id"))
	req.ID = 0
	// Upsert on (kind, symbol).
	var existing TradingContract
	if err := h.db.Where("kind = ? AND symbol = ?", req.Kind, req.Symbol).First(&existing).Error; err == nil {
		existing.MaxLeverage = req.MaxLeverage
		existing.Status = "active"
		h.db.Save(&existing)
		h.publish("contract", req.Symbol, "active")
		h.audit(c, "create", "contract", req.Symbol, req.Kind)
		c.JSON(http.StatusOK, gin.H{"id": existing.ID, "symbol": req.Symbol, "status": "active"})
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	h.publish("contract", req.Symbol, "active")
	h.audit(c, "create", "contract", req.Symbol, req.Kind)
	c.JSON(http.StatusCreated, gin.H{"id": req.ID, "symbol": req.Symbol, "status": "active"})
}

func (h *TradingControlHandler) StopContract(c *gin.Context) {
	h.genericStatus(c, &TradingContract{}, "contract", "symbol")
}
func (h *TradingControlHandler) ResumeContract(c *gin.Context) {
	h.genericStatus(c, &TradingContract{}, "contract", "symbol")
}
func (h *TradingControlHandler) DeleteContract(c *gin.Context) {
	h.genericDelete(c, &TradingContract{}, "contract", "symbol")
}

// ---- Liquidity pools (lifecycle on the existing LiquidityPool model) ----

func (h *TradingControlHandler) StopPool(c *gin.Context) {
	h.genericStatus(c, &LiquidityPool{}, "pool", "pair")
}
func (h *TradingControlHandler) ResumePool(c *gin.Context) {
	h.genericStatus(c, &LiquidityPool{}, "pool", "pair")
}
func (h *TradingControlHandler) DeletePool(c *gin.Context) {
	h.genericDelete(c, &LiquidityPool{}, "pool", "pair")
}

// ---- Trading pairs (lifecycle on the existing models.TradingPair model) ----

func (h *TradingControlHandler) StopPair(c *gin.Context) {
	h.genericStatus(c, &models.TradingPair{}, "pair", "pair_name")
}
func (h *TradingControlHandler) ResumePair(c *gin.Context) {
	h.genericStatus(c, &models.TradingPair{}, "pair", "pair_name")
}
func (h *TradingControlHandler) DeletePair(c *gin.Context) {
	h.genericDelete(c, &models.TradingPair{}, "pair", "pair_name")
}

// ---- Margin markets ----

func (h *TradingControlHandler) ListMarginMarkets(c *gin.Context) {
	var items []MarginMarket
	if err := h.db.Order("created_at DESC").Limit(500).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"margin_markets": items})
}

func (h *TradingControlHandler) CreateMarginMarket(c *gin.Context) {
	var req MarginMarket
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.MaxLeverage < 1 {
		req.MaxLeverage = 3
	}
	if req.BorrowCap == "" {
		req.BorrowCap = "0"
	}
	req.Symbol = strings.ToUpper(req.Symbol)
	req.Status = "active"
	req.CreatedBy = fmt.Sprint(c.Value("admin_id"))
	req.ID = 0
	var existing MarginMarket
	if err := h.db.Where("symbol = ?", req.Symbol).First(&existing).Error; err == nil {
		existing.MaxLeverage = req.MaxLeverage
		existing.BorrowCap = req.BorrowCap
		existing.Status = "active"
		h.db.Save(&existing)
		h.publish("margin_market", req.Symbol, "active")
		h.audit(c, "create", "margin_market", req.Symbol, "")
		c.JSON(http.StatusOK, gin.H{"id": existing.ID, "symbol": req.Symbol, "status": "active"})
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	h.publish("margin_market", req.Symbol, "active")
	h.audit(c, "create", "margin_market", req.Symbol, "")
	c.JSON(http.StatusCreated, gin.H{"id": req.ID, "symbol": req.Symbol, "status": "active"})
}

func (h *TradingControlHandler) StopMarginMarket(c *gin.Context) {
	h.genericStatus(c, &MarginMarket{}, "margin_market", "symbol")
}
func (h *TradingControlHandler) ResumeMarginMarket(c *gin.Context) {
	h.genericStatus(c, &MarginMarket{}, "margin_market", "symbol")
}
func (h *TradingControlHandler) DeleteMarginMarket(c *gin.Context) {
	h.genericDelete(c, &MarginMarket{}, "margin_market", "symbol")
}

// ---- Options lifecycle (existing OptionsContract model) ----

func (h *TradingControlHandler) StopOption(c *gin.Context) {
	h.genericStatus(c, &OptionsContract{}, "option_series", "underlying")
}
func (h *TradingControlHandler) ResumeOption(c *gin.Context) {
	h.genericStatus(c, &OptionsContract{}, "option_series", "underlying")
}

// ---- Copy-trading lifecycle (existing CopyTradingConfig model) ----

func (h *TradingControlHandler) StopCopyConfig(c *gin.Context) {
	h.setCopyConfigStatus(c)
}
func (h *TradingControlHandler) ResumeCopyConfig(c *gin.Context) {
	h.setCopyConfigStatus(c)
}

// setCopyConfigStatus transitions a copy-trading config; keyed by id since the
// model has no name column.
func (h *TradingControlHandler) setCopyConfigStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !tcValidStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be active|stopped|removed|suspended"})
		return
	}
	id := c.Param("id")
	res := h.db.Model(&CopyTradingConfig{}).Where("id = ?", id).Update("status", req.Status)
	if res.Error != nil || res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "copy_trader not found"})
		return
	}
	entity := "copy-config-" + id
	h.publish("copy_trader", entity, req.Status)
	h.audit(c, req.Status, "copy_trader", entity, "")
	c.JSON(http.StatusOK, gin.H{"message": "copy_trader " + req.Status, "entity": entity, "status": req.Status})
}

// ---- Vertical halt/resume + overview + audit ----

func (h *TradingControlHandler) HaltVertical(c *gin.Context) {
	vertical := strings.ToLower(c.Param("vertical"))
	if !tcVerticals[vertical] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown vertical"})
		return
	}
	h.publish("vertical", vertical, "stopped")
	h.audit(c, "halt", "vertical", vertical, "")
	c.JSON(http.StatusOK, gin.H{"vertical": vertical, "status": "stopped"})
}

func (h *TradingControlHandler) ResumeVertical(c *gin.Context) {
	vertical := strings.ToLower(c.Param("vertical"))
	if !tcVerticals[vertical] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown vertical"})
		return
	}
	h.publish("vertical", vertical, "active")
	h.audit(c, "unhalt", "vertical", vertical, "")
	c.JSON(http.StatusOK, gin.H{"vertical": vertical, "status": "active"})
}

func (h *TradingControlHandler) Overview(c *gin.Context) {
	count := func(model interface{}) int64 {
		var n int64
		h.db.Model(model).Where("status = ?", "active").Count(&n)
		return n
	}
	halts := gin.H{}
	for v := range tcVerticals {
		halted := false
		if h.rdb != nil {
			val, err := h.rdb.Get(context.Background(), "trading:control:global:vertical:"+v).Result()
			halted = err == nil && val == "stopped"
		}
		halts[v] = halted
	}
	c.JSON(http.StatusOK, gin.H{
		"contracts_active":      count(&TradingContract{}),
		"pools_active":          count(&LiquidityPool{}),
		"pairs_active":          count(&models.TradingPair{}),
		"margin_markets_active": count(&MarginMarket{}),
		"options_active":        count(&OptionsContract{}),
		"copy_configs_active":   count(&CopyTradingConfig{}),
		"vertical_halts":        halts,
	})
}

func (h *TradingControlHandler) Audit(c *gin.Context) {
	var items []TradingControlAudit
	if err := h.db.Order("created_at DESC").Limit(200).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"audit": items})
}
