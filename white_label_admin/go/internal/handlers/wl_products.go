package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== WL product governance (bot_admin / liquidity_admin / card_admin) ====================
//
// The WL bot, liquidity and card products run as INDEPENDENT processes
// (wl_bots/go, wl_liquidity/go, wl_card/go). These handlers let a WL client
// govern them from the WL-admin panel. They persist GOVERNANCE RECORDS in the
// WL-admin's own PostgreSQL (the wl_* tables), tenant-scoped via
// middleware.TenantID. They do NOT move funds or flip product-internal state
// directly: the recorded halt/resume/freeze decisions are enforced downstream
// (e.g. the bot halt is honoured via the license_service feature flag).

// ---------------- bot_admin ----------------

func (s *Svc) ListWLBotOperators(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, name, strategy, status, config, created_at, updated_at
		 FROM wl_bot_operators WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, strategy, status string
		var cfg []byte
		var created, updated time.Time
		_ = rows.Scan(&id, &name, &strategy, &status, &cfg, &created, &updated)
		out = append(out, gin.H{"id": id, "name": name, "strategy": strategy, "status": status,
			"config": cfg, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"operators": out})
}

func (s *Svc) RegisterWLBotOperator(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Name     string         `json:"name" binding:"required"`
		Strategy string         `json:"strategy"`
		Config   map[string]any `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Strategy == "" {
		req.Strategy = "mm"
	}
	cfg := []byte("{}")
	if req.Config != nil {
		if b, err := json.Marshal(req.Config); err == nil {
			cfg = b
		}
	}
	id := uuid.New()
	ctx := c.Request.Context()
	if _, err := s.db.Exec(ctx,
		`INSERT INTO wl_bot_operators (id, name, strategy, status, config, white_label_id)
		 VALUES ($1,$2,$3,'active',$4,$5)`,
		id, req.Name, req.Strategy, cfg, tenantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "wl_bot_operator.create", "wl_bot_operator", id.String(), gin.H{"name": req.Name, "strategy": req.Strategy})
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "strategy": req.Strategy, "status": "active"})
}

// UpdateWLBotOperatorStatus records a halt/resume governance decision.
// Halt is recorded here; resume requires SuperAdmin two-party collaboration
// (the same rule as resumeClient / payouts) so it is rejected with 403. The
// actual bot halt is honoured downstream via the license_service feature flag.
func (s *Svc) UpdateWLBotOperatorStatus(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	switch req.Status {
	case "halted":
		ct, err := s.db.Exec(ctx,
			`UPDATE wl_bot_operators SET status='halted', updated_at=NOW() WHERE id=$1 AND white_label_id=$2`, id, tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if ct.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "operator not found"})
			return
		}
		s.audit(ctx, middleware.AdminID(c), "wl_bot_operator.halt", "wl_bot_operator", id.String(), nil)
		c.JSON(http.StatusOK, gin.H{"id": id, "status": "halted"})
	case "active":
		// Resume requires TigerWallet SuperAdmin collaboration — fail closed here.
		c.JSON(http.StatusForbidden, gin.H{"error": "SuperAdmin collaboration required to resume operator"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'halted' or 'active'"})
	}
}

func (s *Svc) GetWLBotConfig(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, key, value, updated_at FROM wl_bot_config WHERE white_label_id=$1 ORDER BY key`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var key, value string
		var updated time.Time
		_ = rows.Scan(&id, &key, &value, &updated)
		out = append(out, gin.H{"id": id, "key": key, "value": value, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"config": out})
}

func (s *Svc) WLBotStats(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	var total, active, halted int64
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wl_bot_operators WHERE white_label_id=$1`, tenantID).Scan(&total)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wl_bot_operators WHERE white_label_id=$1 AND status='active'`, tenantID).Scan(&active)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wl_bot_operators WHERE white_label_id=$1 AND status='halted'`, tenantID).Scan(&halted)
	c.JSON(http.StatusOK, gin.H{"total_operators": total, "active": active, "halted": halted})
}

// ---------------- liquidity_admin ----------------

func (s *Svc) ListWLLiquiditySources(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, name, chain, dex, pool_address, token_a, token_b, reserve_a, reserve_b, fee_pct, is_active, created_at
		 FROM wl_liquidity_sources WHERE white_label_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, chain, dex, pool, tokenA, tokenB string
		var reserveA, reserveB, feePct string
		var isActive bool
		var created time.Time
		_ = rows.Scan(&id, &name, &chain, &dex, &pool, &tokenA, &tokenB, &reserveA, &reserveB, &feePct, &isActive, &created)
		out = append(out, gin.H{"id": id, "name": name, "chain": chain, "dex": dex, "pool_address": pool,
			"token_a": tokenA, "token_b": tokenB, "reserve_a": reserveA, "reserve_b": reserveB,
			"fee_pct": feePct, "is_active": isActive, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"sources": out})
}

func (s *Svc) CreateWLLiquiditySource(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Name        string `json:"name" binding:"required"`
		Chain       string `json:"chain" binding:"required"`
		Dex         string `json:"dex" binding:"required"`
		PoolAddress string `json:"pool_address" binding:"required"`
		TokenA      string `json:"token_a" binding:"required"`
		TokenB      string `json:"token_b" binding:"required"`
		ReserveA    string `json:"reserve_a"`
		ReserveB    string `json:"reserve_b"`
		FeePct      string `json:"fee_pct"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ReserveA == "" {
		req.ReserveA = "0"
	}
	if req.ReserveB == "" {
		req.ReserveB = "0"
	}
	if req.FeePct == "" {
		req.FeePct = "0"
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	id := uuid.New()
	ctx := c.Request.Context()
	if _, err := s.db.Exec(ctx,
		`INSERT INTO wl_liquidity_sources (id, name, chain, dex, pool_address, token_a, token_b, reserve_a, reserve_b, fee_pct, is_active, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		id, req.Name, req.Chain, req.Dex, req.PoolAddress, req.TokenA, req.TokenB,
		req.ReserveA, req.ReserveB, req.FeePct, active, tenantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "wl_liquidity_source.create", "wl_liquidity_source", id.String(), gin.H{"name": req.Name, "chain": req.Chain})
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "chain": req.Chain, "dex": req.Dex})
}

func (s *Svc) UpdateWLLiquiditySource(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name     string `json:"name"`
		IsActive *bool  `json:"is_active"`
		FeePct   string `json:"fee_pct"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.Name != "" {
		if _, err := s.db.Exec(ctx, `UPDATE wl_liquidity_sources SET name=$1 WHERE id=$2 AND white_label_id=$3`, req.Name, id, tenantID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.FeePct != "" {
		if _, err := s.db.Exec(ctx, `UPDATE wl_liquidity_sources SET fee_pct=$1 WHERE id=$2 AND white_label_id=$3`, req.FeePct, id, tenantID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.IsActive != nil {
		if _, err := s.db.Exec(ctx, `UPDATE wl_liquidity_sources SET is_active=$1 WHERE id=$2 AND white_label_id=$3`, *req.IsActive, id, tenantID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	s.audit(ctx, middleware.AdminID(c), "wl_liquidity_source.update", "wl_liquidity_source", id.String(), gin.H{"name": req.Name, "fee_pct": req.FeePct})
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteWLLiquiditySource(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM wl_liquidity_sources WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "source not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "wl_liquidity_source.delete", "wl_liquidity_source", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Svc) ListWLLiquidityAllocations(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, name, fee_share_pct, destination, is_active, created_at, updated_at
		 FROM wl_liquidity_allocations WHERE white_label_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, dest, share string
		var isActive bool
		var created, updated time.Time
		_ = rows.Scan(&id, &name, &share, &dest, &isActive, &created, &updated)
		out = append(out, gin.H{"id": id, "name": name, "fee_share_pct": share, "destination": dest,
			"is_active": isActive, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"allocations": out})
}

func (s *Svc) SetWLLiquidityAllocation(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Name        string `json:"name" binding:"required"`
		FeeSharePct string `json:"fee_share_pct" binding:"required"`
		Destination string `json:"destination"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	id := uuid.New()
	ctx := c.Request.Context()
	if _, err := s.db.Exec(ctx,
		`INSERT INTO wl_liquidity_allocations (id, name, fee_share_pct, destination, is_active, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		id, req.Name, req.FeeSharePct, req.Destination, active, tenantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "wl_liquidity_allocation.set", "wl_liquidity_allocation", id.String(), gin.H{"name": req.Name, "fee_share_pct": req.FeeSharePct})
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "fee_share_pct": req.FeeSharePct})
}

func (s *Svc) WLLiquidityStats(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	var totalSources, activeSources int64
	var totalReserveA string
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wl_liquidity_sources WHERE white_label_id=$1`, tenantID).Scan(&totalSources)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wl_liquidity_sources WHERE white_label_id=$1 AND is_active=TRUE`, tenantID).Scan(&activeSources)
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(SUM(reserve_a),0) FROM wl_liquidity_sources WHERE white_label_id=$1`, tenantID).Scan(&totalReserveA)
	var allocCount int64
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wl_liquidity_allocations WHERE white_label_id=$1`, tenantID).Scan(&allocCount)
	c.JSON(http.StatusOK, gin.H{"total_sources": totalSources, "active_sources": activeSources, "total_reserve_a": totalReserveA, "allocations": allocCount})
}

// ---------------- card_admin ----------------

func (s *Svc) ListWLCards(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, COALESCE(user_id::text,''), holder_name, status, balance, currency, created_at, updated_at
		 FROM wl_cards WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var userID, holder, status, balance, currency string
		var created, updated time.Time
		_ = rows.Scan(&id, &userID, &holder, &status, &balance, &currency, &created, &updated)
		out = append(out, gin.H{"id": id, "user_id": userID, "holder_name": holder, "status": status,
			"balance": balance, "currency": currency, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"cards": out})
}

// IssueWLCard records the governance decision to issue a WL-branded card. The
// real card (with encrypted PAN/CVV) is minted in the wl_card backend; this
// row tracks it in the WL-admin panel for the WL client.
func (s *Svc) IssueWLCard(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		UserID     string `json:"user_id"`
		HolderName string `json:"holder_name" binding:"required"`
		Currency   string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	var uid *uuid.UUID
	if req.UserID != "" {
		if parsed, e := uuid.Parse(req.UserID); e == nil {
			uid = &parsed
		}
	}
	id := uuid.New()
	ctx := c.Request.Context()
	if _, err := s.db.Exec(ctx,
		`INSERT INTO wl_cards (id, user_id, holder_name, status, balance, currency, white_label_id)
		 VALUES ($1,$2,$3,'active',0,$4,$5)`,
		id, uid, req.HolderName, req.Currency, tenantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "wl_card.issue", "wl_card", id.String(), gin.H{"holder_name": req.HolderName})
	c.JSON(http.StatusCreated, gin.H{"id": id, "holder_name": req.HolderName, "status": "active", "balance": "0", "currency": req.Currency})
}

// UpdateWLCardStatus freezes/unfreezes a WL card (governance record only).
func (s *Svc) UpdateWLCardStatus(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status != "frozen" && req.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'frozen' or 'active'"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx,
		`UPDATE wl_cards SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "wl_card.status", "wl_card", id.String(), gin.H{"status": req.Status})
	c.JSON(http.StatusOK, gin.H{"id": id, "status": req.Status})
}

func (s *Svc) ListWLCardTransactions(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()

	query := `SELECT id, card_id, amount, merchant, category, status, created_at
		  FROM wl_card_transactions WHERE white_label_id=$1`
	args := []any{tenantID}

	if cardID := c.Query("card_id"); cardID != "" {
		cid, perr := uuid.Parse(cardID)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card_id"})
			return
		}
		query += ` AND card_id=$2 ORDER BY created_at DESC LIMIT 100`
		args = append(args, cid)
	} else {
		query += ` ORDER BY created_at DESC LIMIT 100`
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, card uuid.UUID
		var amount, merchant, category, status string
		var created time.Time
		_ = rows.Scan(&id, &card, &amount, &merchant, &category, &status, &created)
		out = append(out, gin.H{"id": id, "card_id": card, "amount": amount, "merchant": merchant,
			"category": category, "status": status, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"transactions": out})
}

func (s *Svc) WLCardStats(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	var total, active, frozen int64
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wl_cards WHERE white_label_id=$1`, tenantID).Scan(&total)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wl_cards WHERE white_label_id=$1 AND status='active'`, tenantID).Scan(&active)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wl_cards WHERE white_label_id=$1 AND status='frozen'`, tenantID).Scan(&frozen)
	var txCount int64
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wl_card_transactions WHERE white_label_id=$1`, tenantID).Scan(&txCount)
	c.JSON(http.StatusOK, gin.H{"total_cards": total, "active_cards": active, "frozen_cards": frozen, "transactions": txCount})
}
