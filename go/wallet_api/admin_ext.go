package main

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---- Admin chain configuration ----
// Backs the frontend app/api/v1/admin/chains* routes. Reads/writes the
// admin_chain_config table; GET also seeds rows from the static
// SupportedChains registry so the dashboard shows every supported chain even
// before an admin has touched a config row.

type adminChainRow struct {
	ID          uuid.UUID `json:"id"`
	ChainID     int64     `json:"chain_id"`
	Name        string    `json:"name"`
	Symbol      string    `json:"symbol"`
	RPCURL      string    `json:"rpc_url"`
	ExplorerURL string    `json:"explorer_url"`
	Status      string    `json:"status"`
	IsDefault   bool      `json:"is_default"`
}

func handleAdminListChains(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, chain_id, name, symbol, rpc_url, explorer_url, status, is_default FROM admin_chain_config ORDER BY chain_id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []adminChainRow{}
	for rows.Next() {
		var r adminChainRow
		if err := rows.Scan(&r.ID, &r.ChainID, &r.Name, &r.Symbol, &r.RPCURL, &r.ExplorerURL, &r.Status, &r.IsDefault); err != nil {
			continue
		}
		out = append(out, r)
	}
	// Seed missing supported chains so the dashboard always lists all of them.
	known := map[int64]bool{}
	for _, r := range out {
		known[r.ChainID] = true
	}
	for id, cc := range SupportedChains {
		if known[id] {
			continue
		}
		out = append(out, adminChainRow{ChainID: cc.ID, Name: cc.Name, Symbol: cc.Symbol, RPCURL: cc.RPCEndpoint, ExplorerURL: cc.ExplorerAPI, Status: "active"})
	}
	c.JSON(http.StatusOK, gin.H{"chains": out, "count": len(out)})
}

type adminChainCreate struct {
	ChainID     int64  `json:"chain_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Symbol      string `json:"symbol" binding:"required"`
	RPCURL      string `json:"rpc_url" binding:"required"`
	ExplorerURL string `json:"explorer_url" binding:"required"`
	Status      string `json:"status"`
	IsDefault   bool   `json:"is_default"`
}

func handleAdminCreateChain(c *gin.Context) {
	var req adminChainCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	id := uuid.New()
	_, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO admin_chain_config (id, chain_id, name, symbol, rpc_url, explorer_url, status, is_default)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (chain_id) DO UPDATE SET name=$3, symbol=$4, rpc_url=$5, explorer_url=$6, status=$7, is_default=$8, updated_at=NOW()`,
		id, req.ChainID, req.Name, req.Symbol, req.RPCURL, req.ExplorerURL, req.Status, req.IsDefault)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot save chain"})
		return
	}
	applyAdminChainOverrides(c.Request.Context())
	c.JSON(http.StatusCreated, gin.H{"id": id, "chain_id": req.ChainID, "name": req.Name, "status": req.Status})
}

func handleAdminUpdateChain(c *gin.Context) {
	chainID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain id"})
		return
	}
	var req adminChainCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := store.PG.Exec(c.Request.Context(),
		`UPDATE admin_chain_config SET name=$1, symbol=$2, rpc_url=$3, explorer_url=$4, status=$5, is_default=$6, updated_at=NOW()
		 WHERE chain_id=$7`,
		req.Name, req.Symbol, req.RPCURL, req.ExplorerURL, req.Status, req.IsDefault, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	applyAdminChainOverrides(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"chain_id": chainID, "updated": true})
}

func handleAdminDeleteChain(c *gin.Context) {
	chainID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain id"})
		return
	}
	_, err := store.PG.Exec(c.Request.Context(), `DELETE FROM admin_chain_config WHERE chain_id=$1`, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	applyAdminChainOverrides(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"chain_id": chainID, "deleted": true})
}

// applyAdminChainOverrides merges admin_chain_config rows into the in-memory
// SupportedChains registry so admin-added/updated chains become immediately
// visible to every client via GET /api/v1/chains. A row with status "inactive"
// or "maintenance" removes the chain from the live registry (it remains in the
// admin table for re-activation). Called at startup and after every admin CRUD.
func applyAdminChainOverrides(ctx context.Context) {
	if store == nil || store.PG == nil {
		return
	}
	initSupportedChains()
	rows, err := store.PG.Query(ctx,
		`SELECT chain_id, name, symbol, rpc_url, explorer_url, status FROM admin_chain_config`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var cid int64
		var name, symbol, rpc, explorer, status string
		if err := rows.Scan(&cid, &name, &symbol, &rpc, &explorer, &status); err != nil {
			continue
		}
		if status == "inactive" || status == "maintenance" {
			delete(SupportedChains, cid)
			continue
		}
		// merge: base on existing entry if present (preserves derivation path,
		// coin type, chain type), else create a fresh admin-added entry.
		if base, ok := SupportedChains[cid]; ok {
			if name != "" {
				base.Name = name
			}
			if symbol != "" {
				base.Symbol = symbol
			}
			if rpc != "" {
				base.RPCEndpoint = rpc
			}
			if explorer != "" {
				base.ExplorerAPI = explorer
				base.ExplorerURL = explorer
			}
			SupportedChains[cid] = base
		} else {
			SupportedChains[cid] = ChainConfig{
				ID:             cid,
				Name:           name,
				Symbol:         symbol,
				RPCEndpoint:    rpc,
				ExplorerAPI:    explorer,
				ExplorerURL:    explorer,
				ChainType:      "evm", // admin-added chains default to EVM unless extended
				DerivationPath: "m/44'/60'/0'/0/0",
				CoinType:       60,
				Decimals:       18,
			}
		}
	}
}

// ---- Bridges ----

type adminBridgeRow struct {
	ID              uuid.UUID `json:"id"`
	FromChainID     int64     `json:"from_chain_id"`
	ToChainID       int64     `json:"to_chain_id"`
	BridgeName      string    `json:"bridge_name"`
	ContractAddress string    `json:"contract_address"`
	Status          string    `json:"status"`
}

func handleAdminListBridges(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, from_chain_id, to_chain_id, bridge_name, contract_address, status FROM admin_chain_bridge ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []adminBridgeRow{}
	for rows.Next() {
		var r adminBridgeRow
		if err := rows.Scan(&r.ID, &r.FromChainID, &r.ToChainID, &r.BridgeName, &r.ContractAddress, &r.Status); err != nil {
			continue
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"bridges": out, "count": len(out)})
}

type adminBridgeCreate struct {
	FromChainID     int64  `json:"from_chain_id" binding:"required"`
	ToChainID       int64  `json:"to_chain_id" binding:"required"`
	BridgeName      string `json:"bridge_name" binding:"required"`
	ContractAddress string `json:"contract_address" binding:"required"`
	Status          string `json:"status"`
}

func handleAdminCreateBridge(c *gin.Context) {
	var req adminBridgeCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	id := uuid.New()
	_, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO admin_chain_bridge (id, from_chain_id, to_chain_id, bridge_name, contract_address, status) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, req.FromChainID, req.ToChainID, req.BridgeName, req.ContractAddress, req.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot save bridge"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "bridge": req.BridgeName})
}

// ---- Validators ----

type adminValidatorRow struct {
	ID               uuid.UUID `json:"id"`
	ChainID          int64     `json:"chain_id"`
	ValidatorAddress string    `json:"validator_address"`
	Name             string    `json:"name"`
	Status           string    `json:"status"`
}

func handleAdminListValidators(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, chain_id, validator_address, COALESCE(name,''), status FROM admin_chain_validator ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []adminValidatorRow{}
	for rows.Next() {
		var r adminValidatorRow
		if err := rows.Scan(&r.ID, &r.ChainID, &r.ValidatorAddress, &r.Name, &r.Status); err != nil {
			continue
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"validators": out, "count": len(out)})
}

type adminValidatorCreate struct {
	ChainID          int64  `json:"chain_id" binding:"required"`
	ValidatorAddress string `json:"validator_address" binding:"required"`
	Name             string `json:"name"`
	Status           string `json:"status"`
}

func handleAdminCreateValidator(c *gin.Context) {
	var req adminValidatorCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	id := uuid.New()
	_, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO admin_chain_validator (id, chain_id, validator_address, name, status) VALUES ($1,$2,$3,$4,$5)`,
		id, req.ChainID, req.ValidatorAddress, req.Name, req.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot save validator"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "validator": req.ValidatorAddress})
}

// ---- Chain metrics ----
// Aggregates: per-chain wallet counts + transaction counts from the wallet_api
// tables plus configured chain status.

type chainMetricRow struct {
	ChainID          int64  `json:"chain_id"`
	WalletCount      int64  `json:"wallet_count"`
	TransactionCount int64  `json:"transaction_count"`
	Status           string `json:"status"`
}

func handleAdminChainMetrics(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT w.chain_id, COUNT(DISTINCT w.id) AS wallets, COUNT(t.id) AS txs, COALESCE(c.status,'active')
		 FROM wallets w
		 LEFT JOIN transaction_log t ON t.wallet_id = w.id
		 LEFT JOIN admin_chain_config c ON c.chain_id = w.chain_id
		 GROUP BY w.chain_id, c.status ORDER BY w.chain_id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "metrics query failed"})
		return
	}
	defer rows.Close()
	out := []chainMetricRow{}
	for rows.Next() {
		var r chainMetricRow
		if err := rows.Scan(&r.ChainID, &r.WalletCount, &r.TransactionCount, &r.Status); err != nil {
			continue
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"metrics": out, "count": len(out)})
}

// ---- Admin role management ----

// bootstrapAdminRole promotes the user matching ADMIN_BOOTSTRAP_EMAIL to the
// "admin" role at startup. This seeds the first admin so the admin API is
// usable without a pre-existing admin. Idempotent + safe (no-op if unset).
func bootstrapAdminRole(ctx context.Context, email string) {
	if store == nil || store.PG == nil || email == "" {
		return
	}
	_, err := store.PG.Exec(ctx,
		`UPDATE users SET role='admin', updated_at=NOW() WHERE email=$1 AND role='user'`, email)
	if err != nil {
		log.Printf("admin bootstrap: failed to promote %s: %v", email, err)
		return
	}
	log.Printf("admin bootstrap: ensured admin role for %s", email)
}

// validAdminRoles is the set of roles an admin may assign to other users.
var validAdminRoles = map[string]bool{
	"user": true, "admin": true, "wl_admin": true, "master_wallet_admin": true,
}

// handleAdminSetUserRole lets an existing admin change another user's role.
// Route: PUT /api/v1/admin/users/:id/role  body: {"role":"admin"}
func handleAdminSetUserRole(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validAdminRoles[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role; must be one of user, admin, wl_admin, master_wallet_admin"})
		return
	}
	callerID, _ := uuid.Parse(getUserID(c))
	if targetID == callerID && req.Role != "admin" && req.Role != "master_wallet_admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot demote yourself"})
		return
	}
	tag, err := store.PG.Exec(c.Request.Context(),
		`UPDATE users SET role=$1, updated_at=NOW() WHERE id=$2`, req.Role, targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": targetID, "role": req.Role, "updated": true})
}

// ---- Token deployments ----
// Lists ERC-20 deployment records derived from transaction_log (contract
// creations / interactions) joined with admin_chain_config for chain context.

func handleAdminTokenDeployments(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT t.id, t.chain_id, t.to_addr, t.tx_hash, t.status, t.created_at
		 FROM transaction_log t ORDER BY t.created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	type tokenDep struct {
		ID        uuid.UUID `json:"id"`
		ChainID   int64     `json:"chain_id"`
		Address   string    `json:"address"`
		TxHash    string    `json:"tx_hash"`
		Status    string    `json:"status"`
		CreatedAt string    `json:"created_at"`
	}
	out := []tokenDep{}
	for rows.Next() {
		var d tokenDep
		var created []byte
		if err := rows.Scan(&d.ID, &d.ChainID, &d.Address, &d.TxHash, &d.Status, &created); err != nil {
			continue
		}
		d.CreatedAt = string(created)
		out = append(out, d)
	}
	c.JSON(http.StatusOK, gin.H{"deployments": out, "count": len(out)})
}

// ---- Fees ----

type feeTierRow struct {
	ID              uuid.UUID `json:"id"`
	TierName        string    `json:"tier_name"`
	FeeType         string    `json:"fee_type"`
	RateBasisPoints string    `json:"rate_basis_points"`
	MinAmount       string    `json:"min_amount"`
	MaxAmount       string    `json:"max_amount"`
	ChainID         *int64    `json:"chain_id"`
	Status          string    `json:"status"`
}

func handleAdminListFees(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, tier_name, fee_type, rate_basis_points::text, COALESCE(min_amount,'0'), COALESCE(max_amount,''), chain_id, status FROM fee_tier ORDER BY fee_type, tier_name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []feeTierRow{}
	for rows.Next() {
		var r feeTierRow
		if err := rows.Scan(&r.ID, &r.TierName, &r.FeeType, &r.RateBasisPoints, &r.MinAmount, &r.MaxAmount, &r.ChainID, &r.Status); err != nil {
			continue
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"fees": out, "count": len(out)})
}

type feeTierCreate struct {
	TierName        string `json:"tier_name" binding:"required"`
	FeeType         string `json:"fee_type" binding:"required"`
	RateBasisPoints string `json:"rate_basis_points" binding:"required"`
	MinAmount       string `json:"min_amount"`
	MaxAmount       string `json:"max_amount"`
	ChainID         *int64 `json:"chain_id"`
	Status          string `json:"status"`
}

func handleAdminCreateFee(c *gin.Context) {
	var req feeTierCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if req.MinAmount == "" {
		req.MinAmount = "0"
	}
	id := uuid.New()
	_, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO fee_tier (id, tier_name, fee_type, rate_basis_points, min_amount, max_amount, chain_id, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, req.TierName, req.FeeType, req.RateBasisPoints, req.MinAmount, req.MaxAmount, req.ChainID, req.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot save fee tier"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "tier": req.TierName, "fee_type": req.FeeType})
}

func handleAdminUpdateFee(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req feeTierCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err = store.PG.Exec(c.Request.Context(),
		`UPDATE fee_tier SET tier_name=$1, fee_type=$2, rate_basis_points=$3, min_amount=$4, max_amount=$5, chain_id=$6, status=$7, updated_at=NOW() WHERE id=$8`,
		req.TierName, req.FeeType, req.RateBasisPoints, req.MinAmount, req.MaxAmount, req.ChainID, req.Status, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "updated": true})
}

func handleAdminDeleteFee(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	_, err = store.PG.Exec(c.Request.Context(), `DELETE FROM fee_tier WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}

// ---- Fee transactions / revenue ----

type feeTxRow struct {
	ID       uuid.UUID  `json:"id"`
	UserID   *uuid.UUID `json:"user_id"`
	FeeType  string     `json:"fee_type"`
	Amount   string     `json:"amount"`
	Currency string     `json:"currency"`
	ChainID  *int64     `json:"chain_id"`
	TxHash   string     `json:"tx_hash"`
	Status   string     `json:"status"`
}

func handleAdminFeeTransactions(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, user_id, fee_type, amount, currency, chain_id, COALESCE(tx_hash,''), status FROM fee_transaction ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []feeTxRow{}
	for rows.Next() {
		var r feeTxRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.FeeType, &r.Amount, &r.Currency, &r.ChainID, &r.TxHash, &r.Status); err != nil {
			continue
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"transactions": out, "count": len(out)})
}

func handleAdminFeeRevenue(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT fee_type, currency, SUM(amount::numeric) AS total FROM fee_transaction WHERE status='settled' GROUP BY fee_type, currency ORDER BY fee_type`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "revenue query failed"})
		return
	}
	defer rows.Close()
	type revRow struct {
		FeeType  string `json:"fee_type"`
		Currency string `json:"currency"`
		Total    string `json:"total"`
	}
	out := []revRow{}
	for rows.Next() {
		var r revRow
		if err := rows.Scan(&r.FeeType, &r.Currency, &r.Total); err != nil {
			continue
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"revenue": out, "count": len(out)})
}

// handlePublicListFees is the public read-only mirror of handleAdminListFees:
// returns only active fee tiers so users can see the fee schedule before
// transacting. No admin auth required — fee transparency is a public good.
func handlePublicListFees(c *gin.Context) {
        rows, err := store.PG.Query(c.Request.Context(),
                `SELECT tier_name, fee_type, rate_basis_points::text, COALESCE(min_amount,'0'), COALESCE(max_amount,''), chain_id FROM fee_tier WHERE status='active' ORDER BY fee_type, tier_name`)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
                return
        }
        defer rows.Close()
        type publicFee struct {
                TierName        string `json:"tier_name"`
                FeeType         string `json:"fee_type"`
                RateBasisPoints string `json:"rate_basis_points"`
                MinAmount       string `json:"min_amount"`
                MaxAmount       string `json:"max_amount"`
                ChainID         *int64 `json:"chain_id"`
        }
        out := []publicFee{}
        for rows.Next() {
                var r publicFee
                if err := rows.Scan(&r.TierName, &r.FeeType, &r.RateBasisPoints, &r.MinAmount, &r.MaxAmount, &r.ChainID); err != nil {
                        continue
                }
                out = append(out, r)
        }
        c.JSON(http.StatusOK, gin.H{"fees": out, "count": len(out)})
}

// handlePublicFeeTransactions is the public read-only mirror of
// handleAdminFeeTransactions: returns settled fee transactions for
// transparency. No admin auth required.
func handlePublicFeeTransactions(c *gin.Context) {
        rows, err := store.PG.Query(c.Request.Context(),
                `SELECT fee_type, currency, amount::text, chain_id, created_at FROM fee_transaction WHERE status='settled' ORDER BY created_at DESC LIMIT 100`)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
                return
        }
        defer rows.Close()
        type publicFeeTx struct {
                FeeType   string `json:"fee_type"`
                Currency  string `json:"currency"`
                Amount    string `json:"amount"`
                ChainID   *int64 `json:"chain_id"`
                CreatedAt string `json:"created_at"`
        }
        out := []publicFeeTx{}
        for rows.Next() {
                var r publicFeeTx
                if err := rows.Scan(&r.FeeType, &r.Currency, &r.Amount, &r.ChainID, &r.CreatedAt); err != nil {
                        continue
                }
                out = append(out, r)
        }
        c.JSON(http.StatusOK, gin.H{"transactions": out, "count": len(out)})
}
