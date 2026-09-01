package handlers

// trading_control.go — white-label client's tenant-scoped trading
// control-plane.
//
// Owner policy: a white-label client (and their RBAC sub-admins holding the
// trading_admin / listing_admin / liquidity_admin scopes) can create / add /
// remove / stop / resume trading contracts, liquidity pools, trading pairs,
// margin markets, options series, and copy-trading configs INSIDE THEIR OWN
// TENANCY. Every row carries white_label_id (tenant isolation); control flags
// are published to the shared Redis control namespace under the tenant's own
// key ("trading:control:<tenant>:*") — the same propagation pattern as the
// kill-switch / feature-flag planes. Everything is builtin TigerWallet; no
// external broker or exchange is involved.

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

var wlTradingVerticals = map[string]bool{
	"spot": true, "perpetual": true, "futures": true, "margin": true,
	"options": true, "copy": true, "liquidity": true,
}

func wlValidTradingStatus(s string) bool {
	switch s {
	case "active", "stopped", "removed", "suspended":
		return true
	}
	return false
}

// publishTradingControl writes a tenant-scoped control flag to shared Redis.
// Best-effort: Redis outage never blocks the management write (DB is
// authoritative).
func (s *Svc) publishTradingControl(c *gin.Context, kind, key, status string) {
	if s.rdb == nil || s.rdb.Client == nil || key == "" {
		return
	}
	tenant := middleware.TenantID(c).String()
	s.rdb.Client.Set(s.rdb.Ctx,
		"trading:control:"+tenant+":"+kind+":"+strings.ToUpper(key), status, 0)
}

func (s *Svc) auditTradingControl(c *gin.Context, action, kind, entity, detail string) {
	tenant := middleware.TenantID(c)
	adminID := middleware.AdminID(c)
	s.db.Exec(c.Request.Context(),
		`INSERT INTO wl_trading_control_audit (id, actor, action, kind, entity, detail, white_label_id) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New(), adminID, action, kind, entity, detail, tenant)
}

// ---- generic list (tenant-scoped) ----

func (s *Svc) wlListTrading(c *gin.Context, table, outKey, columns string) {
	tenant := middleware.TenantID(c)
	rows, err := s.db.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT %s FROM %s WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 500`, columns, table), tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	fields := rows.FieldDescriptions()
	out := []map[string]interface{}{}
	for rows.Next() {
		vals := make([]interface{}, len(fields))
		for i := range vals {
			vals[i] = new(interface{})
		}
		if err := rows.Scan(vals...); err != nil {
			continue
		}
		m := map[string]interface{}{}
		for i, f := range fields {
			m[string(f.Name)] = *(vals[i].(*interface{}))
		}
		out = append(out, m)
	}
	c.JSON(http.StatusOK, gin.H{outKey: out})
}

// ---- generic status transition (tenant-scoped) ----

func (s *Svc) wlSetTradingStatus(c *gin.Context, table, kind, redisKeyExpr string) {
	tenant := middleware.TenantID(c)
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !wlValidTradingStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be active|stopped|removed|suspended"})
		return
	}
	var entity string
	err := s.db.QueryRow(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3 RETURNING %s`, table, redisKeyExpr),
		req.Status, c.Param("id"), tenant).Scan(&entity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
		return
	}
	s.publishTradingControl(c, kind, entity, req.Status)
	s.auditTradingControl(c, req.Status, kind, entity, "")
	c.JSON(http.StatusOK, gin.H{"message": kind + " " + req.Status, "entity": entity, "status": req.Status})
}

func (s *Svc) wlDeleteTradingEntity(c *gin.Context, table, kind, redisKeyExpr string) {
	tenant := middleware.TenantID(c)
	var entity string
	err := s.db.QueryRow(c.Request.Context(),
		fmt.Sprintf(`DELETE FROM %s WHERE id=$1 AND white_label_id=$2 RETURNING %s`, table, redisKeyExpr),
		c.Param("id"), tenant).Scan(&entity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
		return
	}
	s.publishTradingControl(c, kind, entity, "removed")
	s.auditTradingControl(c, "remove", kind, entity, "")
	c.JSON(http.StatusOK, gin.H{"message": kind + " removed"})
}

// ---- Trading contracts ----

func (s *Svc) ListTradingContracts(c *gin.Context) {
	s.wlListTrading(c, "wl_trading_contracts", "contracts",
		"id, kind, symbol, base_asset, quote_asset, chain_id, max_leverage, min_size, tick_size, status, created_at, updated_at")
}

func (s *Svc) CreateTradingContract(c *gin.Context) {
	tenant := middleware.TenantID(c)
	var req struct {
		Kind        string `json:"kind" binding:"required"`
		Symbol      string `json:"symbol" binding:"required"`
		BaseAsset   string `json:"base_asset" binding:"required"`
		QuoteAsset  string `json:"quote_asset" binding:"required"`
		ChainID     int64  `json:"chain_id"`
		MaxLeverage int    `json:"max_leverage"`
		MinSize     string `json:"min_size"`
		TickSize    string `json:"tick_size"`
	}
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
	if req.MinSize == "" {
		req.MinSize = "0"
	}
	if req.TickSize == "" {
		req.TickSize = "0"
	}
	id := uuid.New()
	symbol := strings.ToUpper(req.Symbol)
	_, err := s.db.Exec(c.Request.Context(),
		`INSERT INTO wl_trading_contracts (id, kind, symbol, base_asset, quote_asset, chain_id, max_leverage, min_size, tick_size, status, white_label_id, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10,$11)
		 ON CONFLICT (kind, symbol, white_label_id) DO UPDATE SET base_asset=EXCLUDED.base_asset,
		   quote_asset=EXCLUDED.quote_asset, chain_id=EXCLUDED.chain_id, max_leverage=EXCLUDED.max_leverage,
		   status='active', updated_at=NOW()`,
		id, req.Kind, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.ChainID, req.MaxLeverage, req.MinSize, req.TickSize, tenant, middleware.AdminID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	s.publishTradingControl(c, "contract", symbol, "active")
	s.auditTradingControl(c, "create", "contract", symbol, req.Kind)
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func (s *Svc) StopTradingContract(c *gin.Context) {
	s.wlSetTradingStatus(c, "wl_trading_contracts", "contract", "symbol")
}
func (s *Svc) ResumeTradingContract(c *gin.Context) {
	s.wlSetTradingStatus(c, "wl_trading_contracts", "contract", "symbol")
}
func (s *Svc) DeleteTradingContract(c *gin.Context) {
	s.wlDeleteTradingEntity(c, "wl_trading_contracts", "contract", "symbol")
}

// ---- Liquidity pools ----

func (s *Svc) ListTradingPools(c *gin.Context) {
	s.wlListTrading(c, "wl_liquidity_pools", "pools",
		"id, chain_id, dex, pool_address, token0, token1, fee_bps, status, created_at, updated_at")
}

func (s *Svc) CreateTradingPool(c *gin.Context) {
	tenant := middleware.TenantID(c)
	var req struct {
		ChainID     int64  `json:"chain_id" binding:"required"`
		Dex         string `json:"dex" binding:"required"`
		PoolAddress string `json:"pool_address"`
		Token0      string `json:"token0" binding:"required"`
		Token1      string `json:"token1" binding:"required"`
		FeeBps      int    `json:"fee_bps"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.FeeBps <= 0 {
		req.FeeBps = 30
	}
	id := uuid.New()
	_, err := s.db.Exec(c.Request.Context(),
		`INSERT INTO wl_liquidity_pools (id, chain_id, dex, pool_address, token0, token1, fee_bps, status, white_label_id, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'active',$8,$9)
		 ON CONFLICT (chain_id, dex, token0, token1, white_label_id) DO UPDATE SET pool_address=EXCLUDED.pool_address,
		   fee_bps=EXCLUDED.fee_bps, status='active', updated_at=NOW()`,
		id, req.ChainID, req.Dex, req.PoolAddress, req.Token0, req.Token1, req.FeeBps, tenant, middleware.AdminID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	s.auditTradingControl(c, "create", "pool", fmt.Sprintf("%d:%s/%s", req.ChainID, req.Token0, req.Token1), req.Dex)
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active"})
}

func (s *Svc) StopTradingPool(c *gin.Context) {
	s.wlSetTradingStatus(c, "wl_liquidity_pools", "pool", "token0 || '/' || token1")
}
func (s *Svc) ResumeTradingPool(c *gin.Context) {
	s.wlSetTradingStatus(c, "wl_liquidity_pools", "pool", "token0 || '/' || token1")
}
func (s *Svc) DeleteTradingPool(c *gin.Context) {
	s.wlDeleteTradingEntity(c, "wl_liquidity_pools", "pool", "token0 || '/' || token1")
}

// ---- Trading pairs ----

func (s *Svc) ListTradingPairs(c *gin.Context) {
	s.wlListTrading(c, "wl_trading_pairs", "pairs",
		"id, symbol, base_asset, quote_asset, chain_id, market, status, created_at, updated_at")
}

func (s *Svc) CreateTradingPair(c *gin.Context) {
	tenant := middleware.TenantID(c)
	var req struct {
		Symbol     string `json:"symbol" binding:"required"`
		BaseAsset  string `json:"base_asset" binding:"required"`
		QuoteAsset string `json:"quote_asset" binding:"required"`
		ChainID    int64  `json:"chain_id"`
		Market     string `json:"market"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Market == "" {
		req.Market = "spot"
	}
	req.Market = strings.ToLower(req.Market)
	id := uuid.New()
	symbol := strings.ToUpper(req.Symbol)
	_, err := s.db.Exec(c.Request.Context(),
		`INSERT INTO wl_trading_pairs (id, symbol, base_asset, quote_asset, chain_id, market, status, white_label_id, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,'active',$7,$8)
		 ON CONFLICT (symbol, market, white_label_id) DO UPDATE SET base_asset=EXCLUDED.base_asset,
		   quote_asset=EXCLUDED.quote_asset, chain_id=EXCLUDED.chain_id, status='active', updated_at=NOW()`,
		id, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.ChainID, req.Market, tenant, middleware.AdminID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	s.publishTradingControl(c, "pair", symbol, "active")
	s.auditTradingControl(c, "create", "pair", symbol, req.Market)
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func (s *Svc) StopTradingPair(c *gin.Context) {
	s.wlSetTradingStatus(c, "wl_trading_pairs", "pair", "symbol")
}
func (s *Svc) ResumeTradingPair(c *gin.Context) {
	s.wlSetTradingStatus(c, "wl_trading_pairs", "pair", "symbol")
}
func (s *Svc) DeleteTradingPair(c *gin.Context) {
	s.wlDeleteTradingEntity(c, "wl_trading_pairs", "pair", "symbol")
}

// ---- Margin markets ----

func (s *Svc) ListMarginMarkets(c *gin.Context) {
	s.wlListTrading(c, "wl_margin_markets", "margin_markets",
		"id, symbol, base_asset, quote_asset, max_leverage, borrow_cap, status, created_at, updated_at")
}

func (s *Svc) CreateMarginMarket(c *gin.Context) {
	tenant := middleware.TenantID(c)
	var req struct {
		Symbol      string `json:"symbol" binding:"required"`
		BaseAsset   string `json:"base_asset" binding:"required"`
		QuoteAsset  string `json:"quote_asset" binding:"required"`
		MaxLeverage int    `json:"max_leverage"`
		BorrowCap   string `json:"borrow_cap"`
	}
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
	id := uuid.New()
	symbol := strings.ToUpper(req.Symbol)
	_, err := s.db.Exec(c.Request.Context(),
		`INSERT INTO wl_margin_markets (id, symbol, base_asset, quote_asset, max_leverage, borrow_cap, status, white_label_id, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,'active',$7,$8)
		 ON CONFLICT (symbol, white_label_id) DO UPDATE SET max_leverage=EXCLUDED.max_leverage,
		   borrow_cap=EXCLUDED.borrow_cap, status='active', updated_at=NOW()`,
		id, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.MaxLeverage, req.BorrowCap, tenant, middleware.AdminID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	s.publishTradingControl(c, "margin_market", symbol, "active")
	s.auditTradingControl(c, "create", "margin_market", symbol, "")
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func (s *Svc) StopMarginMarket(c *gin.Context) {
	s.wlSetTradingStatus(c, "wl_margin_markets", "margin_market", "symbol")
}
func (s *Svc) ResumeMarginMarket(c *gin.Context) {
	s.wlSetTradingStatus(c, "wl_margin_markets", "margin_market", "symbol")
}
func (s *Svc) DeleteMarginMarket(c *gin.Context) {
	s.wlDeleteTradingEntity(c, "wl_margin_markets", "margin_market", "symbol")
}

// ---- Vertical halt/resume (tenant-scoped) + overview + audit ----

func (s *Svc) HaltTradingVertical(c *gin.Context) {
	vertical := strings.ToLower(c.Param("vertical"))
	if !wlTradingVerticals[vertical] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown vertical"})
		return
	}
	s.publishTradingControl(c, "vertical", vertical, "stopped")
	s.auditTradingControl(c, "halt", "vertical", vertical, "")
	c.JSON(http.StatusOK, gin.H{"vertical": vertical, "status": "stopped"})
}

func (s *Svc) ResumeTradingVertical(c *gin.Context) {
	vertical := strings.ToLower(c.Param("vertical"))
	if !wlTradingVerticals[vertical] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown vertical"})
		return
	}
	s.publishTradingControl(c, "vertical", vertical, "active")
	s.auditTradingControl(c, "unhalt", "vertical", vertical, "")
	c.JSON(http.StatusOK, gin.H{"vertical": vertical, "status": "active"})
}

func (s *Svc) TradingOverview(c *gin.Context) {
	tenant := middleware.TenantID(c)
	count := func(table string) int64 {
		var n int64
		if err := s.db.QueryRow(c.Request.Context(),
			fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE white_label_id=$1 AND status='active'`, table), tenant).Scan(&n); err != nil {
			return 0
		}
		return n
	}
	halts := gin.H{}
	for v := range wlTradingVerticals {
		halted := false
		if s.rdb != nil && s.rdb.Client != nil {
			val, err := s.rdb.Client.Get(s.rdb.Ctx,
				"trading:control:"+tenant.String()+":vertical:"+v).Result()
			halted = err == nil && val == "stopped"
		}
		halts[v] = halted
	}
	c.JSON(http.StatusOK, gin.H{
		"contracts_active":      count("wl_trading_contracts"),
		"pools_active":          count("wl_liquidity_pools"),
		"pairs_active":          count("wl_trading_pairs"),
		"margin_markets_active": count("wl_margin_markets"),
		"vertical_halts":        halts,
	})
}

func (s *Svc) TradingControlAudit(c *gin.Context) {
	tenant := middleware.TenantID(c)
	rows, err := s.db.Query(c.Request.Context(),
		`SELECT actor, action, kind, entity, detail, created_at
		 FROM wl_trading_control_audit WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 200`, tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	fields := rows.FieldDescriptions()
	out := []map[string]interface{}{}
	for rows.Next() {
		vals := make([]interface{}, len(fields))
		for i := range vals {
			vals[i] = new(interface{})
		}
		if err := rows.Scan(vals...); err != nil {
			continue
		}
		m := map[string]interface{}{}
		for i, f := range fields {
			m[string(f.Name)] = *(vals[i].(*interface{}))
		}
		out = append(out, m)
	}
	c.JSON(http.StatusOK, gin.H{"audit": out})
}
