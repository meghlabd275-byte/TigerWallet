package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== Withdrawals ====================
// Withdrawal approval here is the WL-SIDE approval only. The actual fund
// movement is gated by the SuperAdmin two-party collaboration: the
// master-wallet backend calls the license control plane's
// /withdrawals/:id/approved endpoint before broadcasting. A WL admin can
// approve/reject the WL-side request, but CANNOT execute the payout alone.

func (s *Svc) ListWithdrawals(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT w.id, w.user_id, w.amount, w.currency, w.status, w.address, COALESCE(w.tx_hash,''),
		        COALESCE(w.approved_by::text,''), COALESCE(w.processed_at, w.created_at), w.created_at
		 FROM withdrawals w JOIN users u ON w.user_id=u.id
		 WHERE u.white_label_id=$1 ORDER BY w.created_at DESC LIMIT 200`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, uid uuid.UUID
		var amount, currency, status, address, txhash, approver string
		var processed, created time.Time
		_ = rows.Scan(&id, &uid, &amount, &currency, &status, &address, &txhash, &approver, &processed, &created)
		out = append(out, gin.H{"id": id, "user_id": uid, "amount": amount, "currency": currency, "status": status, "address": address, "tx_hash": txhash, "approved_by": approver, "processed_at": processed, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"withdrawals": out})
}

func (s *Svc) reviewWithdrawal(c *gin.Context, status, action string) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	adminID := middleware.AdminID(c)
	var ct int64
	if status == "approved" {
		tag, e := s.db.Exec(ctx,
			`UPDATE withdrawals SET status='approved', approved_by=$1
			 WHERE id=$2 AND EXISTS (SELECT 1 FROM users u, withdrawals w WHERE u.id=w.user_id AND u.white_label_id=$3 AND w.id=$2)`,
			adminID, id, tenantID)
		ct, err = tag.RowsAffected(), e
	} else {
		tag, e := s.db.Exec(ctx,
			`UPDATE withdrawals SET status='rejected', approved_by=$1
			 WHERE id=$2 AND EXISTS (SELECT 1 FROM users u, withdrawals w WHERE u.id=w.user_id AND u.white_label_id=$3 AND w.id=$2)`,
			adminID, id, tenantID)
		ct, err = tag.RowsAffected(), e
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "withdrawal not found"})
		return
	}
	s.audit(ctx, adminID, action, "withdrawal", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{action: id})
}

func (s *Svc) ApproveWithdrawal(c *gin.Context) {
	// WL-side approval only. Funds do NOT move yet — SuperAdmin co-sign required.
	s.reviewWithdrawal(c, "approved", "withdrawal.approve_wl_side")
}
func (s *Svc) RejectWithdrawal(c *gin.Context) { s.reviewWithdrawal(c, "rejected", "withdrawal.reject") }

// ProcessWithdrawal — the WL admin marks a withdrawal as processed ONLY after
// the two-party gate (SuperAdmin co-sign) has been satisfied. The actual
// broadcast + tx_hash recording happens in the master-wallet backend, which
// verifies the gate via the license control plane before signing. Here we
// record the tx_hash the master-wallet returns.
func (s *Svc) ProcessWithdrawal(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		TxHash string `json:"tx_hash" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	// Verify the two-party gate: the withdrawal must be SuperAdmin-approved.
	// The WL admin panel queries the license control plane; if not configured,
	// fail-closed (no payout without SuperAdmin collaboration).
	if !s.isTwoPartyApproved(ctx, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "two-party SuperAdmin collaboration required before payout"})
		return
	}
	ct, err := s.db.Exec(ctx,
		`UPDATE withdrawals SET status='processed', tx_hash=$1, processed_at=NOW()
		 WHERE id=$2 AND EXISTS (SELECT 1 FROM users u, withdrawals w WHERE u.id=w.user_id AND u.white_label_id=$3 AND w.id=$2)`,
		req.TxHash, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "withdrawal not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "withdrawal.processed", "withdrawal", id.String(), gin.H{"tx_hash": req.TxHash})
	c.JSON(http.StatusOK, gin.H{"processed": id, "tx_hash": req.TxHash})
}

// ==================== Tokens (listing admin scope) ====================

func (s *Svc) ListTokens(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, symbol, name, COALESCE(contract_address,''), decimals, is_active, is_verified,
		        COALESCE(total_supply,0), COALESCE(chain_id,0), created_at
		 FROM tokens WHERE white_label_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var symbol, name, addr string
		var supply string
		var decimals, chainID int
		var active, verified bool
		var created time.Time
		_ = rows.Scan(&id, &symbol, &name, &addr, &decimals, &active, &verified, &supply, &chainID, &created)
		out = append(out, gin.H{"id": id, "symbol": symbol, "name": name, "contract_address": addr, "decimals": decimals, "is_active": active, "is_verified": verified, "total_supply": supply, "chain_id": chainID, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"tokens": out})
}

func (s *Svc) CreateToken(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Symbol          string `json:"symbol" binding:"required"`
		Name            string `json:"name" binding:"required"`
		ContractAddress string `json:"contract_address"`
		Decimals        int    `json:"decimals"`
		TotalSupply     string `json:"total_supply"`
		ChainID         int    `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Decimals == 0 {
		req.Decimals = 18
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO tokens (id, symbol, name, contract_address, decimals, is_active, is_verified, total_supply, chain_id, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,true,false,$6,$7,$8)`,
		id, req.Symbol, req.Name, req.ContractAddress, req.Decimals, req.TotalSupply, req.ChainID, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "token.create", "token", id.String(), gin.H{"symbol": req.Symbol})
	c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": req.Symbol, "name": req.Name})
}

func (s *Svc) UpdateToken(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name            string `json:"name"`
		ContractAddress string `json:"contract_address"`
		IsActive        *bool  `json:"is_active"`
		IsVerified      *bool  `json:"is_verified"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.Name != "" {
		_, _ = s.db.Exec(ctx, `UPDATE tokens SET name=$1 WHERE id=$2 AND white_label_id=$3`, req.Name, id, tenantID)
	}
	if req.ContractAddress != "" {
		_, _ = s.db.Exec(ctx, `UPDATE tokens SET contract_address=$1 WHERE id=$2 AND white_label_id=$3`, req.ContractAddress, id, tenantID)
	}
	if req.IsActive != nil {
		_, _ = s.db.Exec(ctx, `UPDATE tokens SET is_active=$1 WHERE id=$2 AND white_label_id=$3`, *req.IsActive, id, tenantID)
	}
	if req.IsVerified != nil {
		_, _ = s.db.Exec(ctx, `UPDATE tokens SET is_verified=$1 WHERE id=$2 AND white_label_id=$3`, *req.IsVerified, id, tenantID)
	}
	s.audit(ctx, middleware.AdminID(c), "token.update", "token", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteToken(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM tokens WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "token.delete", "token", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// ==================== Trading pairs (listing admin scope) ====================

func (s *Svc) ListPairs(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, base_token_id, quote_token_id, pair_name, COALESCE(price,0), COALESCE(volume_24h,0),
		        COALESCE(liquidity,0), status, COALESCE(chain_id,0), created_at
		 FROM trading_pairs WHERE white_label_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, base, quote uuid.UUID
		var name, status, price, vol, liq string
		var chainID int
		var created time.Time
		_ = rows.Scan(&id, &base, &quote, &name, &price, &vol, &liq, &status, &chainID, &created)
		out = append(out, gin.H{"id": id, "base_token_id": base, "quote_token_id": quote, "pair_name": name, "price": price, "volume_24h": vol, "liquidity": liq, "status": status, "chain_id": chainID, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"pairs": out})
}

func (s *Svc) CreatePair(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		BaseTokenID  string `json:"base_token_id" binding:"required"`
		QuoteTokenID string `json:"quote_token_id" binding:"required"`
		PairName     string `json:"pair_name" binding:"required"`
		ChainID      int    `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	base, err := uuid.Parse(req.BaseTokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base_token_id"})
		return
	}
	quote, err := uuid.Parse(req.QuoteTokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quote_token_id"})
		return
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err = s.db.Exec(ctx,
		`INSERT INTO trading_pairs (id, base_token_id, quote_token_id, pair_name, status, chain_id, white_label_id)
		 VALUES ($1,$2,$3,$4,'active',$5,$6)`,
		id, base, quote, req.PairName, req.ChainID, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "pair.create", "trading_pair", id.String(), gin.H{"pair": req.PairName})
	c.JSON(http.StatusCreated, gin.H{"id": id, "pair_name": req.PairName})
}

func (s *Svc) UpdatePairStatus(c *gin.Context) {
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
	ct, err := s.db.Exec(ctx, `UPDATE trading_pairs SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "pair.status", "trading_pair", id.String(), gin.H{"status": req.Status})
	c.JSON(http.StatusOK, gin.H{"updated": id, "status": req.Status})
}

// ==================== Blockchains (listing admin scope) ====================

func (s *Svc) ListBlockchains(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, name, symbol, chain_id, is_evm, COALESCE(rpc_url,''), COALESCE(explorer_url,''),
		        COALESCE(native_token,''), COALESCE(decimals,18), is_active, COALESCE(avg_gas_price_gwei,0), created_at
		 FROM blockchains WHERE white_label_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, symbol, rpc, explorer, native string
		var gas string
		var chainID, decimals int
		var isEVM, active bool
		var created time.Time
		_ = rows.Scan(&id, &name, &symbol, &chainID, &isEVM, &rpc, &explorer, &native, &decimals, &active, &gas, &created)
		out = append(out, gin.H{"id": id, "name": name, "symbol": symbol, "chain_id": chainID, "is_evm": isEVM, "rpc_url": rpc, "explorer_url": explorer, "native_token": native, "decimals": decimals, "is_active": active, "avg_gas_price_gwei": gas, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"blockchains": out})
}

func (s *Svc) CreateBlockchain(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Name        string `json:"name" binding:"required"`
		Symbol      string `json:"symbol" binding:"required"`
		ChainID     int    `json:"chain_id" binding:"required"`
		IsEVM       bool   `json:"is_evm"`
		RPCURL      string `json:"rpc_url"`
		ExplorerURL string `json:"explorer_url"`
		NativeToken string `json:"native_token"`
		Decimals    int    `json:"decimals"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Decimals == 0 {
		req.Decimals = 18
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO blockchains (id, name, symbol, chain_id, is_evm, rpc_url, explorer_url, native_token, decimals, is_active, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,true,$10)`,
		id, req.Name, req.Symbol, req.ChainID, req.IsEVM, req.RPCURL, req.ExplorerURL, req.NativeToken, req.Decimals, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "blockchain.create", "blockchain", id.String(), gin.H{"name": req.Name})
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name})
}

func (s *Svc) UpdateBlockchain(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name        string `json:"name"`
		RPCURL      string `json:"rpc_url"`
		ExplorerURL string `json:"explorer_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.Name != "" {
		_, _ = s.db.Exec(ctx, `UPDATE blockchains SET name=$1 WHERE id=$2 AND white_label_id=$3`, req.Name, id, tenantID)
	}
	if req.RPCURL != "" {
		_, _ = s.db.Exec(ctx, `UPDATE blockchains SET rpc_url=$1 WHERE id=$2 AND white_label_id=$3`, req.RPCURL, id, tenantID)
	}
	if req.ExplorerURL != "" {
		_, _ = s.db.Exec(ctx, `UPDATE blockchains SET explorer_url=$1 WHERE id=$2 AND white_label_id=$3`, req.ExplorerURL, id, tenantID)
	}
	s.audit(ctx, middleware.AdminID(c), "blockchain.update", "blockchain", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) SetBlockchainStatus(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `UPDATE blockchains SET is_active=$1 WHERE id=$2 AND white_label_id=$3`, req.IsActive, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "blockchain not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "blockchain.status", "blockchain", id.String(), gin.H{"active": req.IsActive})
	c.JSON(http.StatusOK, gin.H{"updated": id, "is_active": req.IsActive})
}
