package main

// trading_control.go — SuperAdmin's global trading control-plane.
//
// Owner policy: SuperAdmin (plus RBAC admins and white-label clients through
// their own tiers) can create / add / remove / stop / resume trading
// contracts, liquidity pools, trading pairs, margin markets, options series,
// copy-trading traders, and halt/resume whole trading verticals. Everything is
// builtin TigerWallet — enforcement state is published to the shared Redis
// control namespace ("trading:control:global:*") which the wallet engines
// (go/wallet_api, master_wallet) read, exactly like the kill-switch and
// feature-flag planes. TigerWallet never depends on an external broker or
// exchange for these services.

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var saTradingVerticals = map[string]bool{
	"spot": true, "perpetual": true, "futures": true, "margin": true,
	"options": true, "copy": true, "liquidity": true,
}

func saValidTradingStatus(s string) bool {
	switch s {
	case "active", "stopped", "removed", "suspended":
		return true
	}
	return false
}

// publishTradingControl writes a control flag to the shared Redis namespace.
// Best-effort: Redis outage never blocks the management write (DB remains
// authoritative in the admin tier).
func publishTradingControl(kind, key, status string) {
	if redisClient == nil || redisClient.Client == nil || key == "" {
		return
	}
	redisClient.Client.Set(redisClient.Ctx,
		"trading:control:global:"+kind+":"+strings.ToUpper(key), status, 0)
}

func auditTradingControl(c *gin.Context, action, kind, entity, detail string) {
	actor, _ := c.Get("admin_id")
	role, _ := c.Get("role")
	dbExec(c,
		`INSERT INTO trading_control_audit (id, actor, actor_role, action, kind, entity, detail) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New(), fmt.Sprint(actor), fmt.Sprint(role), action, kind, entity, detail)
}

// ---- generic list helper ----

func saListTrading(c *gin.Context, table, outKey, columns string) {
	rows, err := dbQuery(c, fmt.Sprintf(`SELECT %s FROM %s ORDER BY created_at DESC LIMIT 500`, columns, table))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{outKey: rowsToMaps(rows)})
}

// ---- generic status transition ----

func saSetTradingStatus(c *gin.Context, table, kind, redisKeyExpr string) {
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !saValidTradingStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be active|stopped|removed|suspended"})
		return
	}
	var entity string
	err := dbQueryRow(c,
		fmt.Sprintf(`UPDATE %s SET status=$1, updated_at=NOW() WHERE id=$2 RETURNING %s`, table, redisKeyExpr),
		req.Status, c.Param("id")).Scan(&entity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
		return
	}
	publishTradingControl(kind, entity, req.Status)
	auditTradingControl(c, req.Status, kind, entity, "")
	c.JSON(http.StatusOK, gin.H{"message": kind + " " + req.Status, "entity": entity, "status": req.Status})
}

func saDeleteTradingEntity(c *gin.Context, table, kind, redisKeyExpr string) {
	var entity string
	err := dbQueryRow(c,
		fmt.Sprintf(`DELETE FROM %s WHERE id=$1 RETURNING %s`, table, redisKeyExpr), c.Param("id")).Scan(&entity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
		return
	}
	publishTradingControl(kind, entity, "removed")
	auditTradingControl(c, "remove", kind, entity, "")
	c.JSON(http.StatusOK, gin.H{"message": kind + " removed"})
}

// ---- Trading contracts ----

func handleListTradingContracts(c *gin.Context) {
	saListTrading(c, "trading_contracts", "contracts",
		"id, kind, symbol, base_asset, quote_asset, chain_id, max_leverage, min_size, tick_size, status, created_by, created_at, updated_at")
}

func handleCreateTradingContract(c *gin.Context) {
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
	actor, _ := c.Get("admin_id")
	id := uuid.New()
	symbol := strings.ToUpper(req.Symbol)
	if _, err := dbExec(c,
		`INSERT INTO trading_contracts (id, kind, symbol, base_asset, quote_asset, chain_id, max_leverage, min_size, tick_size, status, created_by)
                 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10)
                 ON CONFLICT (kind, symbol) DO UPDATE SET base_asset=EXCLUDED.base_asset, quote_asset=EXCLUDED.quote_asset,
                   chain_id=EXCLUDED.chain_id, max_leverage=EXCLUDED.max_leverage, status='active', updated_at=NOW()`,
		id, req.Kind, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.ChainID, req.MaxLeverage, req.MinSize, req.TickSize, fmt.Sprint(actor)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	publishTradingControl("contract", symbol, "active")
	auditTradingControl(c, "create", "contract", symbol, req.Kind)
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func handleStopTradingContract(c *gin.Context) {
	saSetTradingStatus(c, "trading_contracts", "contract", "symbol")
}
func handleResumeTradingContract(c *gin.Context) {
	saSetTradingStatus(c, "trading_contracts", "contract", "symbol")
}
func handleDeleteTradingContract(c *gin.Context) {
	saDeleteTradingEntity(c, "trading_contracts", "contract", "symbol")
}

// ---- Liquidity pools ----

func handleListLiquidityPools(c *gin.Context) {
	saListTrading(c, "liquidity_pools", "pools",
		"id, chain_id, dex, pool_address, token0, token1, fee_bps, status, created_by, created_at, updated_at")
}

func handleCreateLiquidityPool(c *gin.Context) {
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
	actor, _ := c.Get("admin_id")
	id := uuid.New()
	if _, err := dbExec(c,
		`INSERT INTO liquidity_pools (id, chain_id, dex, pool_address, token0, token1, fee_bps, status, created_by)
                 VALUES ($1,$2,$3,$4,$5,$6,$7,'active',$8)
                 ON CONFLICT (chain_id, dex, token0, token1) DO UPDATE SET pool_address=EXCLUDED.pool_address,
                   fee_bps=EXCLUDED.fee_bps, status='active', updated_at=NOW()`,
		id, req.ChainID, req.Dex, req.PoolAddress, req.Token0, req.Token1, req.FeeBps, fmt.Sprint(actor)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	auditTradingControl(c, "create", "pool", fmt.Sprintf("%d:%s/%s", req.ChainID, req.Token0, req.Token1), req.Dex)
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active"})
}

func handleStopLiquidityPool(c *gin.Context) {
	saSetTradingStatus(c, "liquidity_pools", "pool", "token0 || '/' || token1")
}
func handleResumeLiquidityPool(c *gin.Context) {
	saSetTradingStatus(c, "liquidity_pools", "pool", "token0 || '/' || token1")
}
func handleDeleteLiquidityPool(c *gin.Context) {
	saDeleteTradingEntity(c, "liquidity_pools", "pool", "token0 || '/' || token1")
}

// ---- Trading pairs (lifecycle on the existing trading_pairs table) ----

func handleStopTradingPair(c *gin.Context) {
	saSetTradingStatus(c, "trading_pairs", "pair", "pair_name")
}
func handleResumeTradingPair(c *gin.Context) {
	saSetTradingStatus(c, "trading_pairs", "pair", "pair_name")
}

// ---- Margin markets ----

func handleListMarginMarkets(c *gin.Context) {
	saListTrading(c, "margin_markets", "margin_markets",
		"id, symbol, base_asset, quote_asset, max_leverage, borrow_cap, status, created_by, created_at, updated_at")
}

func handleCreateMarginMarket(c *gin.Context) {
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
	actor, _ := c.Get("admin_id")
	id := uuid.New()
	symbol := strings.ToUpper(req.Symbol)
	if _, err := dbExec(c,
		`INSERT INTO margin_markets (id, symbol, base_asset, quote_asset, max_leverage, borrow_cap, status, created_by)
                 VALUES ($1,$2,$3,$4,$5,$6,'active',$7)
                 ON CONFLICT (symbol) DO UPDATE SET max_leverage=EXCLUDED.max_leverage,
                   borrow_cap=EXCLUDED.borrow_cap, status='active', updated_at=NOW()`,
		id, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.MaxLeverage, req.BorrowCap, fmt.Sprint(actor)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	publishTradingControl("margin_market", symbol, "active")
	auditTradingControl(c, "create", "margin_market", symbol, "")
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func handleStopMarginMarket(c *gin.Context) {
	saSetTradingStatus(c, "margin_markets", "margin_market", "symbol")
}
func handleResumeMarginMarket(c *gin.Context) {
	saSetTradingStatus(c, "margin_markets", "margin_market", "symbol")
}
func handleDeleteMarginMarket(c *gin.Context) {
	saDeleteTradingEntity(c, "margin_markets", "margin_market", "symbol")
}

// ---- Options series lifecycle (existing options_contracts table) ----

func handleStopOptionsContract(c *gin.Context) {
	saSetTradingStatus(c, "options_contracts", "option_series", "underlying")
}
func handleResumeOptionsContract(c *gin.Context) {
	saSetTradingStatus(c, "options_contracts", "option_series", "underlying")
}

// ---- Copy-trading lifecycle (existing copy_trading_configs table) ----

func handleStopCopyTrading(c *gin.Context) {
	saSetTradingStatus(c, "copy_trading_configs", "copy_trader", "id::text")
}
func handleResumeCopyTrading(c *gin.Context) {
	saSetTradingStatus(c, "copy_trading_configs", "copy_trader", "id::text")
}

// ---- Vertical halt/resume + overview + audit ----

func handleHaltTradingVertical(c *gin.Context) {
	vertical := strings.ToLower(c.Param("vertical"))
	if !saTradingVerticals[vertical] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown vertical"})
		return
	}
	publishTradingControl("vertical", vertical, "stopped")
	auditTradingControl(c, "halt", "vertical", vertical, "")
	c.JSON(http.StatusOK, gin.H{"vertical": vertical, "status": "stopped"})
}

func handleResumeTradingVertical(c *gin.Context) {
	vertical := strings.ToLower(c.Param("vertical"))
	if !saTradingVerticals[vertical] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown vertical"})
		return
	}
	publishTradingControl("vertical", vertical, "active")
	auditTradingControl(c, "unhalt", "vertical", vertical, "")
	c.JSON(http.StatusOK, gin.H{"vertical": vertical, "status": "active"})
}

func handleTradingOverview(c *gin.Context) {
	count := func(table string) int64 {
		var n int64
		if err := dbQueryRow(c, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE status='active'`, table)).Scan(&n); err != nil {
			return 0
		}
		return n
	}
	halts := gin.H{}
	for v := range saTradingVerticals {
		halted := false
		if redisClient != nil && redisClient.Client != nil {
			val, err := redisClient.Client.Get(redisClient.Ctx, "trading:control:global:vertical:"+v).Result()
			halted = err == nil && val == "stopped"
		}
		halts[v] = halted
	}
	c.JSON(http.StatusOK, gin.H{
		"contracts_active":      count("trading_contracts"),
		"pools_active":          count("liquidity_pools"),
		"pairs_active":          count("trading_pairs"),
		"margin_markets_active": count("margin_markets"),
		"options_active":        count("options_contracts"),
		"copy_configs_active":   count("copy_trading_configs"),
		"vertical_halts":        halts,
	})
}

func handleTradingControlAudit(c *gin.Context) {
	rows, err := dbQuery(c,
		`SELECT actor, actor_role, action, kind, entity, detail, created_at
                 FROM trading_control_audit ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"audit": rowsToMaps(rows)})
}
