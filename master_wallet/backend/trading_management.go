package main

// trading_management.go — MasterWallet operator's full trading-management
// surface.
//
// MasterWallet shares the canonical `tigerwallet` PostgreSQL database with the
// UserWallet backend (go/wallet_api), so the management writes here govern the
// exact trading_control tables the user-facing engines enforce on (trading
// contracts, liquidity pools, trading pairs, margin markets, options series,
// copy-trading traders, whole-vertical halt/resume). Status changes are also
// published to the shared Redis control namespace (same pattern as the
// kill-switch) so every TigerWallet service enforces one decision.
//
// Policy: requires role "admin" or "operator" (real JWT role check); every
// mutation is recorded in trading_control_audit with actor + role.

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// tradingControlDDL mirrors go/wallet_api/trading_control.go. Both services
// share the `tigerwallet` DB; either one may boot first, so the DDL is
// idempotently ensured lazily (CREATE TABLE IF NOT EXISTS).
const tradingControlDDL = `
CREATE TABLE IF NOT EXISTS trading_contracts (
    id UUID PRIMARY KEY,
    kind TEXT NOT NULL,
    symbol TEXT NOT NULL,
    base_asset TEXT NOT NULL,
    quote_asset TEXT NOT NULL,
    chain_id BIGINT NOT NULL DEFAULT 0,
    max_leverage INT NOT NULL DEFAULT 1,
    min_size TEXT NOT NULL DEFAULT '0',
    tick_size TEXT NOT NULL DEFAULT '0',
    status TEXT NOT NULL DEFAULT 'active',
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (kind, symbol, tenant)
);
CREATE TABLE IF NOT EXISTS liquidity_pools (
    id UUID PRIMARY KEY,
    chain_id BIGINT NOT NULL,
    dex TEXT NOT NULL,
    pool_address TEXT,
    token0 TEXT NOT NULL,
    token1 TEXT NOT NULL,
    fee_bps INT NOT NULL DEFAULT 30,
    status TEXT NOT NULL DEFAULT 'active',
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (chain_id, dex, token0, token1, tenant)
);
CREATE TABLE IF NOT EXISTS managed_trading_pairs (
    id UUID PRIMARY KEY,
    symbol TEXT NOT NULL,
    base_asset TEXT NOT NULL,
    quote_asset TEXT NOT NULL,
    chain_id BIGINT NOT NULL DEFAULT 0,
    market TEXT NOT NULL DEFAULT 'spot',
    status TEXT NOT NULL DEFAULT 'active',
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (symbol, market, tenant)
);
CREATE TABLE IF NOT EXISTS margin_markets (
    id UUID PRIMARY KEY,
    symbol TEXT NOT NULL,
    base_asset TEXT NOT NULL,
    quote_asset TEXT NOT NULL,
    max_leverage INT NOT NULL DEFAULT 3,
    borrow_cap TEXT NOT NULL DEFAULT '0',
    status TEXT NOT NULL DEFAULT 'active',
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (symbol, tenant)
);
CREATE TABLE IF NOT EXISTS options_series (
    id UUID PRIMARY KEY,
    underlying TEXT NOT NULL,
    quote_asset TEXT NOT NULL DEFAULT 'USDT',
    strike TEXT NOT NULL,
    expiry_unix BIGINT NOT NULL,
    style TEXT NOT NULL,
    iv_bps INT NOT NULL DEFAULT 8000,
    contract_size TEXT NOT NULL DEFAULT '1',
    status TEXT NOT NULL DEFAULT 'active',
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (underlying, strike, expiry_unix, style, tenant)
);
CREATE TABLE IF NOT EXISTS copy_trader_registry (
    id UUID PRIMARY KEY,
    trader TEXT NOT NULL,
    display_name TEXT,
    fee_bps INT NOT NULL DEFAULT 100,
    max_copiers INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (trader, tenant)
);
CREATE TABLE IF NOT EXISTS trading_control_audit (
    id UUID PRIMARY KEY,
    actor TEXT,
    actor_role TEXT,
    action TEXT NOT NULL,
    kind TEXT NOT NULL,
    entity TEXT NOT NULL,
    detail TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
`

var tradingTablesOnce sync.Once

func (svc *Service) ensureTradingTables(ctx context.Context) {
	tradingTablesOnce.Do(func() {
		if svc.store != nil && svc.store.DB() != nil {
			svc.store.DB().Exec(ctx, tradingControlDDL)
		}
	})
}

func (svc *Service) tradingActor(c *gin.Context) (actor, role string) {
	v, _ := c.Get(string(ctxUserID))
	actor, _ = v.(string)
	r, _ := c.Get(string(ctxRole))
	role, _ = r.(string)
	return actor, role
}

// publishTradingFlag writes the control flag to the shared Redis namespace.
// Best-effort: Redis outage never blocks the management write (DB is
// authoritative; wallet_api also enforces from DB).
func (svc *Service) publishTradingFlag(ctx context.Context, kind, key, status string) {
	if svc.store == nil || svc.store.redis == nil {
		return
	}
	svc.store.redis.Set(ctx, "trading:control:global:"+kind+":"+strings.ToUpper(key), status, 0)
}

func (svc *Service) auditTrading(c *gin.Context, action, kind, entity, detail string) {
	actor, role := svc.tradingActor(c)
	if svc.store == nil || svc.store.DB() == nil {
		return
	}
	svc.store.DB().Exec(c.Request.Context(),
		`INSERT INTO trading_control_audit (id, actor, actor_role, action, kind, entity, detail) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New(), actor, role, action, kind, entity, detail)
}

func validTradingStatus(s string) bool {
	switch s {
	case "active", "stopped", "removed", "suspended":
		return true
	}
	return false
}

var tradingVerticals = map[string]bool{
	"spot": true, "perpetual": true, "futures": true, "margin": true,
	"options": true, "copy": true, "liquidity": true,
}

// ---- Generic list helper ----

func (svc *Service) listTradingEntity(c *gin.Context, table, kind string, columns string) {
	svc.ensureTradingTables(c.Request.Context())
	rows, err := svc.store.DB().Query(c.Request.Context(),
		fmt.Sprintf(`SELECT %s FROM %s ORDER BY created_at DESC LIMIT 500`, columns, table))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	cols := strings.Split(columns, ",")
	out := []gin.H{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		for i := range vals {
			vals[i] = new(interface{})
		}
		if err := rows.Scan(vals...); err != nil {
			continue
		}
		m := gin.H{}
		for i, col := range cols {
			m[strings.TrimSpace(col)] = *(vals[i].(*interface{}))
		}
		out = append(out, m)
	}
	c.JSON(http.StatusOK, gin.H{kind: out})
}

// ---- Status transitions (stop/resume/remove) ----

func (svc *Service) setTradingStatus(c *gin.Context, table, kind, redisKeyExpr string) {
	svc.ensureTradingTables(c.Request.Context())
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !validTradingStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be active|stopped|removed|suspended"})
		return
	}
	var entity string
	err := svc.store.DB().QueryRow(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s SET status=$1, updated_at=NOW() WHERE id=$2 RETURNING %s`, table, redisKeyExpr),
		req.Status, c.Param("id")).Scan(&entity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
		return
	}
	svc.publishTradingFlag(c.Request.Context(), kind, entity, req.Status)
	svc.auditTrading(c, req.Status, kind, entity, "")
	c.JSON(http.StatusOK, gin.H{"message": kind + " " + req.Status, "entity": entity, "status": req.Status})
}

func (svc *Service) deleteTradingEntity(c *gin.Context, table, kind, redisKeyExpr string) {
	svc.ensureTradingTables(c.Request.Context())
	var entity string
	err := svc.store.DB().QueryRow(c.Request.Context(),
		fmt.Sprintf(`DELETE FROM %s WHERE id=$1 RETURNING %s`, table, redisKeyExpr), c.Param("id")).Scan(&entity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
		return
	}
	svc.publishTradingFlag(c.Request.Context(), kind, entity, "removed")
	svc.auditTrading(c, "remove", kind, entity, "")
	c.JSON(http.StatusOK, gin.H{"message": kind + " removed"})
}

// ---- Trading contracts ----

func (svc *Service) TradingContractsList(c *gin.Context) {
	svc.listTradingEntity(c, "trading_contracts", "contracts",
		"id, kind, symbol, base_asset, quote_asset, chain_id, max_leverage, min_size, tick_size, status, tenant, created_at")
}

func (svc *Service) TradingContractsCreate(c *gin.Context) {
	svc.ensureTradingTables(c.Request.Context())
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
	actor, _ := svc.tradingActor(c)
	id := uuid.New()
	symbol := strings.ToUpper(req.Symbol)
	_, err := svc.store.DB().Exec(c.Request.Context(),
		`INSERT INTO trading_contracts (id, kind, symbol, base_asset, quote_asset, chain_id, max_leverage, min_size, tick_size, status, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10)
		 ON CONFLICT (kind, symbol, tenant) DO UPDATE SET base_asset=EXCLUDED.base_asset, quote_asset=EXCLUDED.quote_asset,
		   chain_id=EXCLUDED.chain_id, max_leverage=EXCLUDED.max_leverage, status='active', updated_at=NOW()`,
		id, req.Kind, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.ChainID, req.MaxLeverage, req.MinSize, req.TickSize, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	svc.publishTradingFlag(c.Request.Context(), "contract", symbol, "active")
	svc.auditTrading(c, "create", "contract", symbol, req.Kind)
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func (svc *Service) TradingContractsStop(c *gin.Context) {
	svc.setTradingStatus(c, "trading_contracts", "contract", "symbol")
}
func (svc *Service) TradingContractsResume(c *gin.Context) {
	svc.setTradingStatus(c, "trading_contracts", "contract", "symbol")
}
func (svc *Service) TradingContractsDelete(c *gin.Context) {
	svc.deleteTradingEntity(c, "trading_contracts", "contract", "symbol")
}

// ---- Liquidity pools ----

func (svc *Service) LiquidityPoolsList(c *gin.Context) {
	svc.listTradingEntity(c, "liquidity_pools", "pools",
		"id, chain_id, dex, pool_address, token0, token1, fee_bps, status, tenant, created_at")
}

func (svc *Service) LiquidityPoolsCreate(c *gin.Context) {
	svc.ensureTradingTables(c.Request.Context())
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
	actor, _ := svc.tradingActor(c)
	id := uuid.New()
	_, err := svc.store.DB().Exec(c.Request.Context(),
		`INSERT INTO liquidity_pools (id, chain_id, dex, pool_address, token0, token1, fee_bps, status, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'active',$8)
		 ON CONFLICT (chain_id, dex, token0, token1, tenant) DO UPDATE SET pool_address=EXCLUDED.pool_address,
		   fee_bps=EXCLUDED.fee_bps, status='active', updated_at=NOW()`,
		id, req.ChainID, req.Dex, req.PoolAddress, req.Token0, req.Token1, req.FeeBps, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	svc.auditTrading(c, "create", "pool", fmt.Sprintf("%d:%s/%s", req.ChainID, req.Token0, req.Token1), req.Dex)
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active"})
}

func (svc *Service) LiquidityPoolsStop(c *gin.Context) {
	svc.setTradingStatus(c, "liquidity_pools", "pool", "token0 || '/' || token1")
}
func (svc *Service) LiquidityPoolsResume(c *gin.Context) {
	svc.setTradingStatus(c, "liquidity_pools", "pool", "token0 || '/' || token1")
}
func (svc *Service) LiquidityPoolsDelete(c *gin.Context) {
	svc.deleteTradingEntity(c, "liquidity_pools", "pool", "token0 || '/' || token1")
}

// ---- Managed trading pairs ----

func (svc *Service) TradingPairsList(c *gin.Context) {
	svc.listTradingEntity(c, "managed_trading_pairs", "pairs",
		"id, symbol, base_asset, quote_asset, chain_id, market, status, tenant, created_at")
}

func (svc *Service) TradingPairsCreate(c *gin.Context) {
	svc.ensureTradingTables(c.Request.Context())
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
	actor, _ := svc.tradingActor(c)
	id := uuid.New()
	symbol := strings.ToUpper(req.Symbol)
	_, err := svc.store.DB().Exec(c.Request.Context(),
		`INSERT INTO managed_trading_pairs (id, symbol, base_asset, quote_asset, chain_id, market, status, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,'active',$7)
		 ON CONFLICT (symbol, market, tenant) DO UPDATE SET base_asset=EXCLUDED.base_asset,
		   quote_asset=EXCLUDED.quote_asset, chain_id=EXCLUDED.chain_id, status='active', updated_at=NOW()`,
		id, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.ChainID, req.Market, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	svc.publishTradingFlag(c.Request.Context(), "pair", symbol, "active")
	svc.auditTrading(c, "create", "pair", symbol, req.Market)
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func (svc *Service) TradingPairsStop(c *gin.Context) {
	svc.setTradingStatus(c, "managed_trading_pairs", "pair", "symbol")
}
func (svc *Service) TradingPairsResume(c *gin.Context) {
	svc.setTradingStatus(c, "managed_trading_pairs", "pair", "symbol")
}
func (svc *Service) TradingPairsDelete(c *gin.Context) {
	svc.deleteTradingEntity(c, "managed_trading_pairs", "pair", "symbol")
}

// ---- Margin markets ----

func (svc *Service) MarginMarketsList(c *gin.Context) {
	svc.listTradingEntity(c, "margin_markets", "margin_markets",
		"id, symbol, base_asset, quote_asset, max_leverage, borrow_cap, status, tenant, created_at")
}

func (svc *Service) MarginMarketsCreate(c *gin.Context) {
	svc.ensureTradingTables(c.Request.Context())
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
	actor, _ := svc.tradingActor(c)
	id := uuid.New()
	symbol := strings.ToUpper(req.Symbol)
	_, err := svc.store.DB().Exec(c.Request.Context(),
		`INSERT INTO margin_markets (id, symbol, base_asset, quote_asset, max_leverage, borrow_cap, status, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,'active',$7)
		 ON CONFLICT (symbol, tenant) DO UPDATE SET max_leverage=EXCLUDED.max_leverage,
		   borrow_cap=EXCLUDED.borrow_cap, status='active', updated_at=NOW()`,
		id, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.MaxLeverage, req.BorrowCap, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	svc.publishTradingFlag(c.Request.Context(), "margin_market", symbol, "active")
	svc.auditTrading(c, "create", "margin_market", symbol, "")
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func (svc *Service) MarginMarketsStop(c *gin.Context) {
	svc.setTradingStatus(c, "margin_markets", "margin_market", "symbol")
}
func (svc *Service) MarginMarketsResume(c *gin.Context) {
	svc.setTradingStatus(c, "margin_markets", "margin_market", "symbol")
}
func (svc *Service) MarginMarketsDelete(c *gin.Context) {
	svc.deleteTradingEntity(c, "margin_markets", "margin_market", "symbol")
}

// ---- Options series ----

func (svc *Service) OptionsSeriesList(c *gin.Context) {
	svc.listTradingEntity(c, "options_series", "series",
		"id, underlying, quote_asset, strike, expiry_unix, style, iv_bps, contract_size, status, tenant, created_at")
}

func (svc *Service) OptionsSeriesCreate(c *gin.Context) {
	svc.ensureTradingTables(c.Request.Context())
	var req struct {
		Underlying   string `json:"underlying" binding:"required"`
		QuoteAsset   string `json:"quote_asset"`
		Strike       string `json:"strike" binding:"required"`
		ExpiryUnix   int64  `json:"expiry_unix" binding:"required"`
		Style        string `json:"style" binding:"required"`
		IVBps        int    `json:"iv_bps"`
		ContractSize string `json:"contract_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Style = strings.ToLower(req.Style)
	if req.Style != "call" && req.Style != "put" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "style must be call|put"})
		return
	}
	if _, err := strconv.ParseFloat(req.Strike, 64); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "strike must be numeric"})
		return
	}
	if req.ExpiryUnix <= time.Now().Unix() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expiry must be in the future"})
		return
	}
	if req.IVBps <= 0 {
		req.IVBps = 8000
	}
	if req.QuoteAsset == "" {
		req.QuoteAsset = "USDT"
	}
	if req.ContractSize == "" {
		req.ContractSize = "1"
	}
	actor, _ := svc.tradingActor(c)
	id := uuid.New()
	und := strings.ToUpper(req.Underlying)
	_, err := svc.store.DB().Exec(c.Request.Context(),
		`INSERT INTO options_series (id, underlying, quote_asset, strike, expiry_unix, style, iv_bps, contract_size, status, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'active',$9)
		 ON CONFLICT (underlying, strike, expiry_unix, style, tenant) DO UPDATE SET iv_bps=EXCLUDED.iv_bps,
		   contract_size=EXCLUDED.contract_size, status='active', updated_at=NOW()`,
		id, und, strings.ToUpper(req.QuoteAsset), req.Strike, req.ExpiryUnix, req.Style, req.IVBps, req.ContractSize, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	svc.auditTrading(c, "create", "option_series", und+" "+req.Strike+" "+req.Style, "")
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active"})
}

func (svc *Service) OptionsSeriesStop(c *gin.Context) {
	svc.setTradingStatus(c, "options_series", "option_series", "underlying")
}
func (svc *Service) OptionsSeriesResume(c *gin.Context) {
	svc.setTradingStatus(c, "options_series", "option_series", "underlying")
}
func (svc *Service) OptionsSeriesDelete(c *gin.Context) {
	svc.deleteTradingEntity(c, "options_series", "option_series", "underlying")
}

// ---- Copy-trading traders ----

func (svc *Service) CopyTradersList(c *gin.Context) {
	svc.listTradingEntity(c, "copy_trader_registry", "traders",
		"id, trader, display_name, fee_bps, max_copiers, status, tenant, created_at")
}

func (svc *Service) CopyTradersCreate(c *gin.Context) {
	svc.ensureTradingTables(c.Request.Context())
	var req struct {
		Trader      string `json:"trader" binding:"required"`
		DisplayName string `json:"display_name"`
		FeeBps      int    `json:"fee_bps"`
		MaxCopiers  int    `json:"max_copiers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.FeeBps < 0 {
		req.FeeBps = 100
	}
	actor, _ := svc.tradingActor(c)
	id := uuid.New()
	_, err := svc.store.DB().Exec(c.Request.Context(),
		`INSERT INTO copy_trader_registry (id, trader, display_name, fee_bps, max_copiers, status, created_by)
		 VALUES ($1,$2,$3,$4,$5,'active',$6)
		 ON CONFLICT (trader, tenant) DO UPDATE SET display_name=EXCLUDED.display_name,
		   fee_bps=EXCLUDED.fee_bps, max_copiers=EXCLUDED.max_copiers, status='active', updated_at=NOW()`,
		id, req.Trader, req.DisplayName, req.FeeBps, req.MaxCopiers, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	svc.publishTradingFlag(c.Request.Context(), "copy_trader", req.Trader, "active")
	svc.auditTrading(c, "create", "copy_trader", req.Trader, "")
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active"})
}

func (svc *Service) CopyTradersStop(c *gin.Context) {
	svc.setTradingStatus(c, "copy_trader_registry", "copy_trader", "trader")
}
func (svc *Service) CopyTradersResume(c *gin.Context) {
	svc.setTradingStatus(c, "copy_trader_registry", "copy_trader", "trader")
}
func (svc *Service) CopyTradersDelete(c *gin.Context) {
	svc.deleteTradingEntity(c, "copy_trader_registry", "copy_trader", "trader")
}

// ---- Vertical halt/resume + overview + audit ----

func (svc *Service) TradingHalt(c *gin.Context) {
	vertical := strings.ToLower(c.Param("vertical"))
	if !tradingVerticals[vertical] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown vertical"})
		return
	}
	svc.publishTradingFlag(c.Request.Context(), "vertical", vertical, "stopped")
	svc.auditTrading(c, "halt", "vertical", vertical, "")
	c.JSON(http.StatusOK, gin.H{"vertical": vertical, "status": "stopped"})
}

func (svc *Service) TradingResume(c *gin.Context) {
	vertical := strings.ToLower(c.Param("vertical"))
	if !tradingVerticals[vertical] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown vertical"})
		return
	}
	svc.publishTradingFlag(c.Request.Context(), "vertical", vertical, "active")
	svc.auditTrading(c, "unhalt", "vertical", vertical, "")
	c.JSON(http.StatusOK, gin.H{"vertical": vertical, "status": "active"})
}

func (svc *Service) TradingOverview(c *gin.Context) {
	svc.ensureTradingTables(c.Request.Context())
	ctx := c.Request.Context()
	count := func(table string) int64 {
		var active, stopped int64
		if svc.store == nil || svc.store.DB() == nil {
			return 0
		}
		svc.store.DB().QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FILTER (WHERE status='active'), COUNT(*) FILTER (WHERE status<>'active') FROM %s`, table)).Scan(&active, &stopped)
		return active
	}
	halts := gin.H{}
	for v := range tradingVerticals {
		halted := false
		if svc.store != nil && svc.store.redis != nil {
			val, err := svc.store.redis.Get(ctx, "trading:control:global:vertical:"+v).Result()
			halted = err == nil && val == "stopped"
		}
		halts[v] = halted
	}
	c.JSON(http.StatusOK, gin.H{
		"contracts_active":      count("trading_contracts"),
		"pools_active":          count("liquidity_pools"),
		"pairs_active":          count("managed_trading_pairs"),
		"margin_markets_active": count("margin_markets"),
		"options_series_active": count("options_series"),
		"copy_traders_active":   count("copy_trader_registry"),
		"vertical_halts":        halts,
	})
}

func (svc *Service) TradingAudit(c *gin.Context) {
	svc.ensureTradingTables(c.Request.Context())
	rows, err := svc.store.DB().Query(c.Request.Context(),
		`SELECT actor, actor_role, action, kind, entity, detail, created_at
		 FROM trading_control_audit ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var actor, role, detail *string
		var action, kind, entity string
		var createdAt time.Time
		if err := rows.Scan(&actor, &role, &action, &kind, &entity, &detail, &createdAt); err != nil {
			continue
		}
		out = append(out, gin.H{"actor": actor, "actor_role": role, "action": action, "kind": kind,
			"entity": entity, "detail": detail, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"audit": out})
}
