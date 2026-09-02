package main

// trading_control.go — TigerWallet's builtin trading control-plane.
//
// One canonical, fully internal management surface for every trading vertical:
//   - trading contracts (perpetual / futures / options markets)
//   - liquidity pools
//   - trading pairs
//   - margin markets
//   - options series (with a builtin user-facing options engine)
//   - copy-trading trader registry
//
// Policy (owner directive): SuperAdmin, White-label clients, and RBAC admins
// can create / add / remove / stop / resume any of the above. TigerWallet never
// depends on an external broker/exchange for these services — the engines are
// builtin (this file + portfolio_features.go + amm_router.go + the internal
// copy_trading_service), and every enforcement decision is computed from
// TigerWallet's own PostgreSQL + shared Redis.
//
// Cross-service propagation follows the established kill-switch / feature-flag
// pattern: admin tiers (super_admin, admin/go, white_label_admin) WRITE control
// state to the shared Redis namespace; wallet_api (and the other wallet
// services) READ it. Redis outage degrades to DB-only enforcement (never
// self-paralyzing); an explicit stop in either store always blocks.

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// tradingControlSchemaSQL is appended to schemaSQL in store.go (auto-migrate).
const tradingControlSchemaSQL = `

CREATE TABLE IF NOT EXISTS trading_contracts (
    id UUID PRIMARY KEY,
    kind TEXT NOT NULL,                  -- perpetual | futures | options
    symbol TEXT NOT NULL,                -- e.g. BTC-USDT-PERP
    base_asset TEXT NOT NULL,
    quote_asset TEXT NOT NULL,
    chain_id BIGINT NOT NULL DEFAULT 0,
    max_leverage INT NOT NULL DEFAULT 1,
    min_size TEXT NOT NULL DEFAULT '0',
    tick_size TEXT NOT NULL DEFAULT '0',
    status TEXT NOT NULL DEFAULT 'active',  -- active | stopped | removed
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (kind, symbol, tenant)
);
CREATE INDEX IF NOT EXISTS idx_trading_contracts_kind ON trading_contracts(kind, status);

CREATE TABLE IF NOT EXISTS liquidity_pools (
    id UUID PRIMARY KEY,
    chain_id BIGINT NOT NULL,
    dex TEXT NOT NULL,
    pool_address TEXT,
    token0 TEXT NOT NULL,
    token1 TEXT NOT NULL,
    fee_bps INT NOT NULL DEFAULT 30,
    status TEXT NOT NULL DEFAULT 'active',  -- active | stopped | removed
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (chain_id, dex, token0, token1, tenant)
);
CREATE INDEX IF NOT EXISTS idx_liquidity_pools_chain ON liquidity_pools(chain_id, status);

CREATE TABLE IF NOT EXISTS managed_trading_pairs (
    id UUID PRIMARY KEY,
    symbol TEXT NOT NULL,                -- e.g. BTC/USDT
    base_asset TEXT NOT NULL,
    quote_asset TEXT NOT NULL,
    chain_id BIGINT NOT NULL DEFAULT 0,
    market TEXT NOT NULL DEFAULT 'spot', -- spot | perpetual | margin | options
    status TEXT NOT NULL DEFAULT 'active',  -- active | stopped | removed
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (symbol, market, tenant)
);
CREATE INDEX IF NOT EXISTS idx_managed_pairs_market ON managed_trading_pairs(market, status);

CREATE TABLE IF NOT EXISTS margin_markets (
    id UUID PRIMARY KEY,
    symbol TEXT NOT NULL,                -- e.g. BTC/USDT
    base_asset TEXT NOT NULL,
    quote_asset TEXT NOT NULL,
    max_leverage INT NOT NULL DEFAULT 3,
    borrow_cap TEXT NOT NULL DEFAULT '0',   -- 0 = uncapped
    status TEXT NOT NULL DEFAULT 'active',  -- active | stopped | removed
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (symbol, tenant)
);

CREATE TABLE IF NOT EXISTS options_series (
    id UUID PRIMARY KEY,
    underlying TEXT NOT NULL,            -- e.g. BTC
    quote_asset TEXT NOT NULL DEFAULT 'USDT',
    strike TEXT NOT NULL,
    expiry_unix BIGINT NOT NULL,
    style TEXT NOT NULL,                 -- call | put
    iv_bps INT NOT NULL DEFAULT 8000,    -- implied volatility, admin-set (no fabricated vol)
    contract_size TEXT NOT NULL DEFAULT '1',
    status TEXT NOT NULL DEFAULT 'active',  -- active | stopped | removed
    tenant TEXT NOT NULL DEFAULT 'global',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (underlying, strike, expiry_unix, style, tenant)
);
CREATE INDEX IF NOT EXISTS idx_options_series_underlying ON options_series(underlying, status);

CREATE TABLE IF NOT EXISTS options_positions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    series_id UUID NOT NULL REFERENCES options_series(id),
    side TEXT NOT NULL,                  -- buy | sell
    contracts TEXT NOT NULL,
    premium TEXT NOT NULL,               -- per-contract premium in quote units at open
    status TEXT NOT NULL DEFAULT 'open', -- open | closed
    pnl TEXT,
    opened_at TIMESTAMPTZ DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_options_positions_user ON options_positions(user_id, status);

CREATE TABLE IF NOT EXISTS copy_trader_registry (
    id UUID PRIMARY KEY,
    trader TEXT NOT NULL,                -- address or handle
    display_name TEXT,
    fee_bps INT NOT NULL DEFAULT 100,
    max_copiers INT NOT NULL DEFAULT 0,  -- 0 = unlimited
    status TEXT NOT NULL DEFAULT 'active',  -- active | suspended | removed
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
    action TEXT NOT NULL,                -- create | update | stop | resume | remove | halt | unhalt
    kind TEXT NOT NULL,                  -- contract | pool | pair | margin_market | option_series | copy_trader | vertical
    entity TEXT NOT NULL,
    detail TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_trading_audit_kind ON trading_control_audit(kind, created_at);
`

// ---- Trading verticals (the granularity at which whole engines halt) ----

var tradingVerticals = map[string]bool{
	"spot":      true,
	"perpetual": true,
	"futures":   true,
	"margin":    true,
	"options":   true,
	"copy":      true,
	"liquidity": true,
}

// tradingEntityKinds enumerates the managed entity kinds (Redis key segment).
var tradingEntityKinds = map[string]bool{
	"contract":      true,
	"pool":          true,
	"pair":          true,
	"margin_market": true,
	"option_series": true,
	"copy_trader":   true,
}

func tradingControlKey(parts ...string) string {
	return "trading:control:" + strings.Join(parts, ":")
}

// publishTradingControl writes control state to the shared Redis namespace so
// every TigerWallet service enforces the same decision. Best-effort: a Redis
// outage never blocks the management write itself (DB remains authoritative).
func publishTradingControl(ctx context.Context, tenant, kind, key, status string) {
	if store == nil || store.Redis == nil {
		return
	}
	store.Redis.Set(ctx, tradingControlKey(tenant, kind, strings.ToUpper(key)), status, 0)
}

// tradingVerticalHalted reports whether a whole vertical (futures, margin,
// options, copy, spot, liquidity, perpetual) is halted globally. Reads the
// shared Redis flag; a Redis outage fails OPEN (DB rows still enforce
// per-entity stops) so an infra blip never halts trading by itself.
func tradingVerticalHalted(ctx context.Context, vertical string) bool {
	if store == nil || store.Redis == nil {
		return false
	}
	v, err := store.Redis.Get(ctx, tradingControlKey("global", "vertical", vertical)).Result()
	return err == nil && v == "stopped"
}

// tradingEntityStoppedRedis reports an explicit per-entity stop flag.
func tradingEntityStoppedRedis(ctx context.Context, kind, key string) bool {
	if store == nil || store.Redis == nil {
		return false
	}
	v, err := store.Redis.Get(ctx, tradingControlKey("global", kind, strings.ToUpper(key))).Result()
	return err == nil && (v == "stopped" || v == "removed" || v == "suspended")
}

// tradingGuard is the shared enforcement gate for user-facing engines. It
// blocks when the whole vertical is halted or the specific entity was
// explicitly stopped/removed by SuperAdmin / RBAC admin / MasterWallet
// operator. Unknown/unmanaged entities trade freely (blacklist semantics), so
// enabling the control-plane never breaks existing flows.
func tradingGuard(c *gin.Context, vertical, kind, key string) bool {
	ctx := c.Request.Context()
	if tradingVerticalHalted(ctx, vertical) {
		c.JSON(http.StatusForbidden, gin.H{"error": vertical + " trading is halted by the platform operator"})
		return false
	}
	if key != "" && tradingEntityStoppedRedis(ctx, kind, key) {
		c.JSON(http.StatusForbidden, gin.H{"error": kind + " " + key + " is stopped by the platform operator"})
		return false
	}
	return true
}

// tradingVerticalGuard returns gin middleware that blocks a whole vertical
// (used to gate the copy-trading reverse proxy).
func tradingVerticalGuard(vertical string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !tradingGuard(c, vertical, "", "") {
			c.Abort()
			return
		}
		c.Next()
	}
}

// tradingContractFor loads the global contract row for (kind, symbol), or nil
// when unmanaged. Used for leverage caps + DB-persisted stops.
func tradingContractFor(ctx context.Context, kind, symbol string) (maxLeverage int, status string, found bool) {
	if store == nil || store.PG == nil {
		return 0, "", false
	}
	err := store.PG.QueryRow(ctx,
		`SELECT max_leverage, status FROM trading_contracts WHERE kind=$1 AND symbol=$2 AND tenant='global'`,
		kind, strings.ToUpper(symbol)).Scan(&maxLeverage, &status)
	if err != nil {
		return 0, "", false
	}
	return maxLeverage, status, true
}

// marginMarketFor loads the global margin market row for a pair symbol.
func marginMarketFor(ctx context.Context, symbol string) (maxLeverage int, status string, found bool) {
	if store == nil || store.PG == nil {
		return 0, "", false
	}
	err := store.PG.QueryRow(ctx,
		`SELECT max_leverage, status FROM margin_markets WHERE symbol=$1 AND tenant='global'`,
		strings.ToUpper(symbol)).Scan(&maxLeverage, &status)
	if err != nil {
		return 0, "", false
	}
	return maxLeverage, status, true
}

// auditTradingControl records every management mutation (compliance trail).
func auditTradingControl(c *gin.Context, action, kind, entity, detail string) {
	if store == nil || store.PG == nil {
		return
	}
	store.PG.Exec(c.Request.Context(),
		`INSERT INTO trading_control_audit (id, actor, actor_role, action, kind, entity, detail) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New(), getUserID(c), getUserRole(c), action, kind, entity, detail)
}

// ---- Status machine ----

// tradingPairStopped reports whether a managed trading pair was explicitly
// stopped/removed by an operator, checking both symbol orderings. Blacklist
// semantics: unmanaged pairs trade freely.
func tradingPairStopped(ctx context.Context, a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	A, B := strings.ToUpper(a), strings.ToUpper(b)
	return tradingEntityStoppedRedis(ctx, "pair", A+"/"+B) ||
		tradingEntityStoppedRedis(ctx, "pair", B+"/"+A)
}

func validTradingStatus(s string) bool {
	switch s {
	case "active", "stopped", "removed", "suspended":
		return true
	}
	return false
}

// ---- Admin handlers: trading contracts ----

func handleAdminListTradingContracts(c *gin.Context) {
	kind := c.Query("kind")
	q := `SELECT id, kind, symbol, base_asset, quote_asset, chain_id, max_leverage, min_size, tick_size, status, tenant, created_by, created_at, updated_at FROM trading_contracts`
	args := []interface{}{}
	if kind != "" {
		q += ` WHERE kind=$1`
		args = append(args, kind)
	}
	q += ` ORDER BY created_at DESC LIMIT 500`
	rows, err := store.PG.Query(c.Request.Context(), q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, k, sym, base, quote, minSize, tick, status, tenant string
		var chainID int64
		var maxLev int
		var createdBy *string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &k, &sym, &base, &quote, &chainID, &maxLev, &minSize, &tick, &status, &tenant, &createdBy, &createdAt, &updatedAt); err != nil {
			continue
		}
		out = append(out, gin.H{"id": id, "kind": k, "symbol": sym, "base_asset": base, "quote_asset": quote,
			"chain_id": chainID, "max_leverage": maxLev, "min_size": minSize, "tick_size": tick,
			"status": status, "tenant": tenant, "created_by": createdBy, "created_at": createdAt, "updated_at": updatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"contracts": out})
}

func handleAdminCreateTradingContract(c *gin.Context) {
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
	_, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO trading_contracts (id, kind, symbol, base_asset, quote_asset, chain_id, max_leverage, min_size, tick_size, status, created_by)
                 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10)
                 ON CONFLICT (kind, symbol, tenant) DO UPDATE SET base_asset=EXCLUDED.base_asset, quote_asset=EXCLUDED.quote_asset,
                   chain_id=EXCLUDED.chain_id, max_leverage=EXCLUDED.max_leverage, min_size=EXCLUDED.min_size,
                   tick_size=EXCLUDED.tick_size, status='active', updated_at=NOW()`,
		id, req.Kind, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.ChainID, req.MaxLeverage, req.MinSize, req.TickSize, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	publishTradingControl(c.Request.Context(), "global", "contract", symbol, "active")
	auditTradingControl(c, "create", "contract", symbol, req.Kind)
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func handleAdminUpdateTradingContract(c *gin.Context) {
	var req struct {
		MaxLeverage *int   `json:"max_leverage"`
		MinSize     string `json:"min_size"`
		TickSize    string `json:"tick_size"`
		ChainID     *int64 `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := store.PG.Exec(c.Request.Context(),
		`UPDATE trading_contracts SET
                   max_leverage = COALESCE($1, max_leverage),
                   min_size = COALESCE(NULLIF($2,''), min_size),
                   tick_size = COALESCE(NULLIF($3,''), tick_size),
                   chain_id = COALESCE($4, chain_id),
                   updated_at=NOW()
                 WHERE id=$5`,
		req.MaxLeverage, req.MinSize, req.TickSize, req.ChainID, c.Param("id"))
	if err != nil || res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contract not found"})
		return
	}
	auditTradingControl(c, "update", "contract", c.Param("id"), "")
	c.JSON(http.StatusOK, gin.H{"message": "contract updated"})
}

// handleAdminSetTradingContractStatus implements stop / resume / remove for
// contracts (and is reused by the other entity handlers below).
func setTradingEntityStatus(c *gin.Context, table, kind, redisKey string) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validTradingStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be active|stopped|removed|suspended"})
		return
	}
	var symbol string
	err := store.PG.QueryRow(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s SET status=$1, updated_at=NOW() WHERE id=$2 RETURNING %s`, table, redisKey),
		req.Status, c.Param("id")).Scan(&symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
		return
	}
	publishTradingControl(c.Request.Context(), "global", kind, symbol, req.Status)
	auditTradingControl(c, req.Status, kind, symbol, "")
	c.JSON(http.StatusOK, gin.H{"message": kind + " " + req.Status, "symbol": symbol, "status": req.Status})
}

func deleteTradingEntity(c *gin.Context, table, kind, redisKey string) {
	var symbol string
	err := store.PG.QueryRow(c.Request.Context(),
		fmt.Sprintf(`DELETE FROM %s WHERE id=$1 RETURNING %s`, table, redisKey), c.Param("id")).Scan(&symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
		return
	}
	publishTradingControl(c.Request.Context(), "global", kind, symbol, "removed")
	auditTradingControl(c, "remove", kind, symbol, "")
	c.JSON(http.StatusOK, gin.H{"message": kind + " removed"})
}

func handleAdminStopTradingContract(c *gin.Context) {
	setTradingEntityStatus(c, "trading_contracts", "contract", "symbol")
}
func handleAdminResumeTradingContract(c *gin.Context) {
	setTradingEntityStatus(c, "trading_contracts", "contract", "symbol")
}
func handleAdminDeleteTradingContract(c *gin.Context) {
	deleteTradingEntity(c, "trading_contracts", "contract", "symbol")
}

// ---- Admin handlers: liquidity pools ----

func handleAdminListLiquidityPools(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, chain_id, dex, pool_address, token0, token1, fee_bps, status, tenant, created_by, created_at, updated_at
                 FROM liquidity_pools ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, dex, t0, t1, status, tenant string
		var poolAddr, createdBy *string
		var chainID int64
		var feeBps int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &chainID, &dex, &poolAddr, &t0, &t1, &feeBps, &status, &tenant, &createdBy, &createdAt, &updatedAt); err != nil {
			continue
		}
		out = append(out, gin.H{"id": id, "chain_id": chainID, "dex": dex, "pool_address": poolAddr,
			"token0": t0, "token1": t1, "fee_bps": feeBps, "status": status, "tenant": tenant,
			"created_by": createdBy, "created_at": createdAt, "updated_at": updatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"pools": out})
}

func handleAdminCreateLiquidityPool(c *gin.Context) {
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
	_, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO liquidity_pools (id, chain_id, dex, pool_address, token0, token1, fee_bps, status, created_by)
                 VALUES ($1,$2,$3,$4,$5,$6,$7,'active',$8)
                 ON CONFLICT (chain_id, dex, token0, token1, tenant) DO UPDATE SET pool_address=EXCLUDED.pool_address,
                   fee_bps=EXCLUDED.fee_bps, status='active', updated_at=NOW()`,
		id, req.ChainID, req.Dex, req.PoolAddress, req.Token0, req.Token1, req.FeeBps, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	publishTradingControl(c.Request.Context(), "global", "pool", poolRedisKey(req.ChainID, req.Token0, req.Token1), "active")
	auditTradingControl(c, "create", "pool", fmt.Sprintf("%d:%s/%s", req.ChainID, req.Token0, req.Token1), req.Dex)
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active"})
}

func poolRedisKey(chainID int64, t0, t1 string) string {
	return fmt.Sprintf("%d:%s/%s", chainID, strings.ToUpper(t0), strings.ToUpper(t1))
}

func handleAdminStopLiquidityPool(c *gin.Context) {
	setTradingEntityStatus(c, "liquidity_pools", "pool", "token0 || '/' || token1")
}
func handleAdminResumeLiquidityPool(c *gin.Context) {
	setTradingEntityStatus(c, "liquidity_pools", "pool", "token0 || '/' || token1")
}
func handleAdminDeleteLiquidityPool(c *gin.Context) {
	deleteTradingEntity(c, "liquidity_pools", "pool", "token0 || '/' || token1")
}

// ---- Admin handlers: managed trading pairs ----

func handleAdminListManagedPairs(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, symbol, base_asset, quote_asset, chain_id, market, status, tenant, created_by, created_at, updated_at
                 FROM managed_trading_pairs ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, sym, base, quote, market, status, tenant string
		var chainID int64
		var createdBy *string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &sym, &base, &quote, &chainID, &market, &status, &tenant, &createdBy, &createdAt, &updatedAt); err != nil {
			continue
		}
		out = append(out, gin.H{"id": id, "symbol": sym, "base_asset": base, "quote_asset": quote,
			"chain_id": chainID, "market": market, "status": status, "tenant": tenant,
			"created_by": createdBy, "created_at": createdAt, "updated_at": updatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"pairs": out})
}

func handleAdminCreateManagedPair(c *gin.Context) {
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
	if req.Market != "spot" && req.Market != "perpetual" && req.Market != "margin" && req.Market != "options" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "market must be spot|perpetual|margin|options"})
		return
	}
	id := uuid.New()
	symbol := strings.ToUpper(req.Symbol)
	_, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO managed_trading_pairs (id, symbol, base_asset, quote_asset, chain_id, market, status, created_by)
                 VALUES ($1,$2,$3,$4,$5,$6,'active',$7)
                 ON CONFLICT (symbol, market, tenant) DO UPDATE SET base_asset=EXCLUDED.base_asset,
                   quote_asset=EXCLUDED.quote_asset, chain_id=EXCLUDED.chain_id, status='active', updated_at=NOW()`,
		id, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.ChainID, req.Market, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	publishTradingControl(c.Request.Context(), "global", "pair", symbol, "active")
	auditTradingControl(c, "create", "pair", symbol, req.Market)
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func handleAdminStopManagedPair(c *gin.Context) {
	setTradingEntityStatus(c, "managed_trading_pairs", "pair", "symbol")
}
func handleAdminResumeManagedPair(c *gin.Context) {
	setTradingEntityStatus(c, "managed_trading_pairs", "pair", "symbol")
}
func handleAdminDeleteManagedPair(c *gin.Context) {
	deleteTradingEntity(c, "managed_trading_pairs", "pair", "symbol")
}

// ---- Admin handlers: margin markets ----

func handleAdminListMarginMarkets(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, symbol, base_asset, quote_asset, max_leverage, borrow_cap, status, tenant, created_by, created_at, updated_at
                 FROM margin_markets ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, sym, base, quote, cap_, status, tenant string
		var maxLev int
		var createdBy *string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &sym, &base, &quote, &maxLev, &cap_, &status, &tenant, &createdBy, &createdAt, &updatedAt); err != nil {
			continue
		}
		out = append(out, gin.H{"id": id, "symbol": sym, "base_asset": base, "quote_asset": quote,
			"max_leverage": maxLev, "borrow_cap": cap_, "status": status, "tenant": tenant,
			"created_by": createdBy, "created_at": createdAt, "updated_at": updatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"margin_markets": out})
}

func handleAdminCreateMarginMarket(c *gin.Context) {
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
	_, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO margin_markets (id, symbol, base_asset, quote_asset, max_leverage, borrow_cap, status, created_by)
                 VALUES ($1,$2,$3,$4,$5,$6,'active',$7)
                 ON CONFLICT (symbol, tenant) DO UPDATE SET base_asset=EXCLUDED.base_asset, quote_asset=EXCLUDED.quote_asset,
                   max_leverage=EXCLUDED.max_leverage, borrow_cap=EXCLUDED.borrow_cap, status='active', updated_at=NOW()`,
		id, symbol, strings.ToUpper(req.BaseAsset), strings.ToUpper(req.QuoteAsset), req.MaxLeverage, req.BorrowCap, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	publishTradingControl(c.Request.Context(), "global", "margin_market", symbol, "active")
	auditTradingControl(c, "create", "margin_market", symbol, "")
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": symbol, "status": "active"})
}

func handleAdminStopMarginMarket(c *gin.Context) {
	setTradingEntityStatus(c, "margin_markets", "margin_market", "symbol")
}
func handleAdminResumeMarginMarket(c *gin.Context) {
	setTradingEntityStatus(c, "margin_markets", "margin_market", "symbol")
}
func handleAdminDeleteMarginMarket(c *gin.Context) {
	deleteTradingEntity(c, "margin_markets", "margin_market", "symbol")
}

// ---- Admin handlers: options series ----

func handleAdminListOptionsSeries(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, underlying, quote_asset, strike, expiry_unix, style, iv_bps, contract_size, status, tenant, created_by, created_at, updated_at
                 FROM options_series ORDER BY expiry_unix ASC LIMIT 500`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, und, quote, strike, style, size, status, tenant string
		var expiry int64
		var ivBps int
		var createdBy *string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &und, &quote, &strike, &expiry, &style, &ivBps, &size, &status, &tenant, &createdBy, &createdAt, &updatedAt); err != nil {
			continue
		}
		out = append(out, gin.H{"id": id, "underlying": und, "quote_asset": quote, "strike": strike,
			"expiry_unix": expiry, "style": style, "iv_bps": ivBps, "contract_size": size,
			"status": status, "tenant": tenant, "created_by": createdBy, "created_at": createdAt, "updated_at": updatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"series": out})
}

func handleAdminCreateOptionsSeries(c *gin.Context) {
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
	id := uuid.New()
	und := strings.ToUpper(req.Underlying)
	_, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO options_series (id, underlying, quote_asset, strike, expiry_unix, style, iv_bps, contract_size, status, created_by)
                 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'active',$9)
                 ON CONFLICT (underlying, strike, expiry_unix, style, tenant) DO UPDATE SET iv_bps=EXCLUDED.iv_bps,
                   contract_size=EXCLUDED.contract_size, status='active', updated_at=NOW()`,
		id, und, strings.ToUpper(req.QuoteAsset), req.Strike, req.ExpiryUnix, req.Style, req.IVBps, req.ContractSize, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	publishTradingControl(c.Request.Context(), "global", "option_series", optionSeriesKey(und, req.Strike, req.ExpiryUnix, req.Style), "active")
	auditTradingControl(c, "create", "option_series", und+" "+req.Strike+" "+req.Style, "")
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active"})
}

func optionSeriesKey(und, strike string, expiry int64, style string) string {
	return fmt.Sprintf("%s-%s-%d-%s", und, strike, expiry, strings.ToUpper(style))
}

func handleAdminStopOptionsSeries(c *gin.Context) {
	setTradingEntityStatus(c, "options_series", "option_series", "underlying")
}
func handleAdminResumeOptionsSeries(c *gin.Context) {
	setTradingEntityStatus(c, "options_series", "option_series", "underlying")
}
func handleAdminDeleteOptionsSeries(c *gin.Context) {
	deleteTradingEntity(c, "options_series", "option_series", "underlying")
}

// ---- Admin handlers: copy-trading trader registry ----

func handleAdminListCopyTraders(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, trader, display_name, fee_bps, max_copiers, status, tenant, created_by, created_at, updated_at
                 FROM copy_trader_registry ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, trader, status, tenant string
		var displayName, createdBy *string
		var feeBps, maxCopiers int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &trader, &displayName, &feeBps, &maxCopiers, &status, &tenant, &createdBy, &createdAt, &updatedAt); err != nil {
			continue
		}
		out = append(out, gin.H{"id": id, "trader": trader, "display_name": displayName, "fee_bps": feeBps,
			"max_copiers": maxCopiers, "status": status, "tenant": tenant, "created_by": createdBy,
			"created_at": createdAt, "updated_at": updatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"traders": out})
}

func handleAdminCreateCopyTrader(c *gin.Context) {
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
	id := uuid.New()
	_, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO copy_trader_registry (id, trader, display_name, fee_bps, max_copiers, status, created_by)
                 VALUES ($1,$2,$3,$4,$5,'active',$6)
                 ON CONFLICT (trader, tenant) DO UPDATE SET display_name=EXCLUDED.display_name,
                   fee_bps=EXCLUDED.fee_bps, max_copiers=EXCLUDED.max_copiers, status='active', updated_at=NOW()`,
		id, req.Trader, req.DisplayName, req.FeeBps, req.MaxCopiers, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	publishTradingControl(c.Request.Context(), "global", "copy_trader", req.Trader, "active")
	auditTradingControl(c, "create", "copy_trader", req.Trader, "")
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active"})
}

func handleAdminStopCopyTrader(c *gin.Context) {
	setTradingEntityStatus(c, "copy_trader_registry", "copy_trader", "trader")
}
func handleAdminResumeCopyTrader(c *gin.Context) {
	setTradingEntityStatus(c, "copy_trader_registry", "copy_trader", "trader")
}
func handleAdminDeleteCopyTrader(c *gin.Context) {
	deleteTradingEntity(c, "copy_trader_registry", "copy_trader", "trader")
}

// ---- Admin handlers: vertical halt / resume + overview + audit ----

func handleAdminHaltVertical(c *gin.Context) {
	vertical := strings.ToLower(c.Param("vertical"))
	if !tradingVerticals[vertical] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown vertical"})
		return
	}
	publishTradingControl(c.Request.Context(), "global", "vertical", vertical, "stopped")
	auditTradingControl(c, "halt", "vertical", vertical, "")
	c.JSON(http.StatusOK, gin.H{"vertical": vertical, "status": "stopped"})
}

func handleAdminResumeVertical(c *gin.Context) {
	vertical := strings.ToLower(c.Param("vertical"))
	if !tradingVerticals[vertical] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown vertical"})
		return
	}
	publishTradingControl(c.Request.Context(), "global", "vertical", vertical, "active")
	auditTradingControl(c, "unhalt", "vertical", vertical, "")
	c.JSON(http.StatusOK, gin.H{"vertical": vertical, "status": "active"})
}

func handleAdminTradingOverview(c *gin.Context) {
	ctx := c.Request.Context()
	count := func(q string) int64 {
		var n int64
		if err := store.PG.QueryRow(ctx, q).Scan(&n); err != nil {
			return 0
		}
		return n
	}
	halts := gin.H{}
	for v := range tradingVerticals {
		halts[v] = tradingVerticalHalted(ctx, v)
	}
	c.JSON(http.StatusOK, gin.H{
		"contracts_active":      count(`SELECT COUNT(*) FROM trading_contracts WHERE status='active'`),
		"contracts_stopped":     count(`SELECT COUNT(*) FROM trading_contracts WHERE status<>'active'`),
		"pools_active":          count(`SELECT COUNT(*) FROM liquidity_pools WHERE status='active'`),
		"pools_stopped":         count(`SELECT COUNT(*) FROM liquidity_pools WHERE status<>'active'`),
		"pairs_active":          count(`SELECT COUNT(*) FROM managed_trading_pairs WHERE status='active'`),
		"pairs_stopped":         count(`SELECT COUNT(*) FROM managed_trading_pairs WHERE status<>'active'`),
		"margin_markets_active": count(`SELECT COUNT(*) FROM margin_markets WHERE status='active'`),
		"options_series_active": count(`SELECT COUNT(*) FROM options_series WHERE status='active'`),
		"copy_traders_active":   count(`SELECT COUNT(*) FROM copy_trader_registry WHERE status='active'`),
		"vertical_halts":        halts,
	})
}

func handleAdminTradingAudit(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
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

// ---- Builtin options engine (user-facing) ----

// normCDF is the standard normal cumulative distribution (via math.Erf).
func normCDF(x float64) float64 { return 0.5 * (1 + math.Erf(x/math.Sqrt2)) }

// blackScholes prices a European option with risk-free rate 0 (deterministic;
// IV is operator-set per series, underlying price is live — nothing invented).
func blackScholes(style string, S, K, tYears, sigma float64) float64 {
	if S <= 0 || K <= 0 || sigma <= 0 {
		return 0
	}
	if tYears <= 0 {
		if style == "call" {
			return math.Max(S-K, 0)
		}
		return math.Max(K-S, 0)
	}
	d1 := (math.Log(S/K) + 0.5*sigma*sigma*tYears) / (sigma * math.Sqrt(tYears))
	d2 := d1 - sigma*math.Sqrt(tYears)
	if style == "call" {
		return S*normCDF(d1) - K*normCDF(d2)
	}
	return K*normCDF(-d2) - S*normCDF(-d1)
}

// optionsIntrinsic returns the immediate-exercise value per contract.
func optionsIntrinsic(style string, S, K float64) float64 {
	if style == "call" {
		return math.Max(S-K, 0)
	}
	return math.Max(K-S, 0)
}

type optionsSeriesRow struct {
	ID           string
	Underlying   string
	QuoteAsset   string
	Strike       string
	ExpiryUnix   int64
	Style        string
	IVBps        int
	ContractSize string
	Status       string
}

func loadOptionsSeries(ctx context.Context, id string) (*optionsSeriesRow, error) {
	var s optionsSeriesRow
	err := store.PG.QueryRow(ctx,
		`SELECT id, underlying, quote_asset, strike, expiry_unix, style, iv_bps, contract_size, status
                 FROM options_series WHERE id=$1`, id).
		Scan(&s.ID, &s.Underlying, &s.QuoteAsset, &s.Strike, &s.ExpiryUnix, &s.Style, &s.IVBps, &s.ContractSize, &s.Status)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// premiumForSeries computes the per-contract premium from the LIVE underlying
// price (builtin price fetcher) and the operator-set IV. Fail-closed: a price
// feed outage returns an error, never an invented premium.
func premiumForSeries(ctx context.Context, s *optionsSeriesRow) (premium, underlying float64, err error) {
	p, err := FetchTokenPrice(ctx, coinIDForSymbol(s.Underlying))
	if err != nil || p == nil || p.PriceUSD <= 0 {
		return 0, 0, fmt.Errorf("live price unavailable for %s", s.Underlying)
	}
	K, err := strconv.ParseFloat(s.Strike, 64)
	if err != nil || K <= 0 {
		return 0, 0, fmt.Errorf("invalid strike")
	}
	size, err := strconv.ParseFloat(s.ContractSize, 64)
	if err != nil || size <= 0 {
		size = 1
	}
	tYears := float64(s.ExpiryUnix-time.Now().Unix()) / (365.25 * 24 * 3600)
	sigma := float64(s.IVBps) / 10000.0
	return blackScholes(s.Style, p.PriceUSD, K, tYears, sigma) * size, p.PriceUSD, nil
}

func handleListOptionsSeries(c *gin.Context) {
	und := strings.ToUpper(c.Query("underlying"))
	q := `SELECT id, underlying, quote_asset, strike, expiry_unix, style, iv_bps, contract_size, status
              FROM options_series WHERE status='active' AND expiry_unix > $1 AND tenant='global'`
	args := []interface{}{time.Now().Unix()}
	if und != "" {
		q += ` AND underlying=$2`
		args = append(args, und)
	}
	q += ` ORDER BY expiry_unix ASC LIMIT 200`
	rows, err := store.PG.Query(c.Request.Context(), q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []optionsSeriesRow{}
	for rows.Next() {
		var s optionsSeriesRow
		if err := rows.Scan(&s.ID, &s.Underlying, &s.QuoteAsset, &s.Strike, &s.ExpiryUnix, &s.Style, &s.IVBps, &s.ContractSize, &s.Status); err != nil {
			continue
		}
		out = append(out, s)
	}
	c.JSON(http.StatusOK, gin.H{"series": out})
}

func handleOptionsQuote(c *gin.Context) {
	if !tradingGuard(c, "options", "", "") {
		return
	}
	s, err := loadOptionsSeries(c.Request.Context(), c.Query("series_id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "series not found"})
		return
	}
	premium, underlying, err := premiumForSeries(c.Request.Context(), s)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"series_id": s.ID, "underlying_price": underlying, "premium_per_contract": premium, "style": s.Style, "strike": s.Strike, "expiry_unix": s.ExpiryUnix})
}

func handleOpenOptionsPosition(c *gin.Context) {
	if !tradingGuard(c, "options", "", "") {
		return
	}
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		SeriesID  string `json:"series_id" binding:"required"`
		Side      string `json:"side" binding:"required"`
		Contracts string `json:"contracts" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Side = strings.ToLower(req.Side)
	if req.Side != "buy" && req.Side != "sell" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "side must be buy|sell"})
		return
	}
	qty, err := strconv.ParseFloat(req.Contracts, 64)
	if err != nil || qty <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contracts must be a positive number"})
		return
	}
	s, err := loadOptionsSeries(c.Request.Context(), req.SeriesID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "series not found"})
		return
	}
	if s.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "series is stopped"})
		return
	}
	if s.ExpiryUnix <= time.Now().Unix() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "series expired"})
		return
	}
	premium, _, err := premiumForSeries(c.Request.Context(), s)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	id := uuid.New()
	if _, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO options_positions (id, user_id, series_id, side, contracts, premium, status)
                 VALUES ($1,$2,$3,$4,$5,$6,'open')`,
		id, userUUID, s.ID, req.Side, req.Contracts, strconv.FormatFloat(premium, 'f', 8, 64)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "open failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "premium_per_contract": premium, "status": "open"})
}

func handleListOptionsPositions(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT p.id, p.series_id, s.underlying, s.strike, s.expiry_unix, s.style, p.side, p.contracts, p.premium, p.status, p.pnl, p.opened_at, p.closed_at
                 FROM options_positions p JOIN options_series s ON s.id=p.series_id
                 WHERE p.user_id=$1 ORDER BY p.opened_at DESC LIMIT 200`, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, seriesID, und, strike, style, side, contracts, premium, status string
		var expiry int64
		var pnl *string
		var openedAt time.Time
		var closedAt *time.Time
		if err := rows.Scan(&id, &seriesID, &und, &strike, &expiry, &style, &side, &contracts, &premium, &status, &pnl, &openedAt, &closedAt); err != nil {
			continue
		}
		out = append(out, gin.H{"id": id, "series_id": seriesID, "underlying": und, "strike": strike,
			"expiry_unix": expiry, "style": style, "side": side, "contracts": contracts,
			"premium": premium, "status": status, "pnl": pnl, "opened_at": openedAt, "closed_at": closedAt})
	}
	c.JSON(http.StatusOK, gin.H{"positions": out})
}

func handleCloseOptionsPosition(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var seriesID, side, contractsStr, premiumStr string
	err = store.PG.QueryRow(c.Request.Context(),
		`SELECT series_id, side, contracts, premium FROM options_positions WHERE id=$1 AND user_id=$2 AND status='open'`,
		c.Param("id"), userUUID).Scan(&seriesID, &side, &contractsStr, &premiumStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "open position not found"})
		return
	}
	s, err := loadOptionsSeries(c.Request.Context(), seriesID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "series lookup failed"})
		return
	}
	p, err := FetchTokenPrice(c.Request.Context(), coinIDForSymbol(s.Underlying))
	if err != nil || p == nil || p.PriceUSD <= 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "live price unavailable"})
		return
	}
	K, _ := strconv.ParseFloat(s.Strike, 64)
	qty, _ := strconv.ParseFloat(contractsStr, 64)
	premium, _ := strconv.ParseFloat(premiumStr, 64)
	size, _ := strconv.ParseFloat(s.ContractSize, 64)
	if size <= 0 {
		size = 1
	}
	intrinsic := optionsIntrinsic(s.Style, p.PriceUSD, K) * size
	var pnl float64
	if side == "buy" {
		pnl = (intrinsic - premium) * qty
	} else {
		pnl = (premium - intrinsic) * qty
	}
	if _, err := store.PG.Exec(c.Request.Context(),
		`UPDATE options_positions SET status='closed', pnl=$1, closed_at=NOW() WHERE id=$2`,
		strconv.FormatFloat(pnl, 'f', 8, 64), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "close failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id"), "status": "closed", "intrinsic_per_contract": intrinsic, "pnl": pnl})
}
