package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== Users (tenant-scoped) ====================

func (s *Svc) ListUsers(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, email, username, COALESCE(wallet_address,''), kyc_status, status, two_factor_enabled,
		        COALESCE(ip_address,''), COALESCE(country,''), created_at, COALESCE(last_login, created_at)
		 FROM users WHERE white_label_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var email, username, wallet, kyc, status, ip, country string
		var tfa bool
		var created, last time.Time
		_ = rows.Scan(&id, &email, &username, &wallet, &kyc, &status, &tfa, &ip, &country, &created, &last)
		out = append(out, gin.H{"id": id, "email": email, "username": username, "wallet_address": wallet, "kyc_status": kyc, "status": status, "two_factor_enabled": tfa, "ip_address": ip, "country": country, "created_at": created, "last_login": last})
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
}

func (s *Svc) GetUser(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var email, username, wallet, kyc, status string
	var tfa bool
	var created time.Time
	err = s.db.QueryRow(ctx,
		`SELECT email, username, COALESCE(wallet_address,''), kyc_status, status, two_factor_enabled, created_at
		 FROM users WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&email, &username, &wallet, &kyc, &status, &tfa, &created)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "email": email, "username": username, "wallet_address": wallet, "kyc_status": kyc, "status": status, "two_factor_enabled": tfa, "created_at": created})
}

func (s *Svc) UpdateUserStatus(c *gin.Context) {
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
	ct, err := s.db.Exec(ctx, `UPDATE users SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "user.status", "user", id.String(), gin.H{"status": req.Status})
	c.JSON(http.StatusOK, gin.H{"updated": id, "status": req.Status})
}

func (s *Svc) setUserStatus(c *gin.Context, status, action string) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `UPDATE users SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), action, "user", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{action: id})
}

func (s *Svc) BanUser(c *gin.Context)     { s.setUserStatus(c, "banned", "user.ban") }
func (s *Svc) UnbanUser(c *gin.Context)   { s.setUserStatus(c, "active", "user.unban") }
func (s *Svc) SuspendUser(c *gin.Context) { s.setUserStatus(c, "suspended", "user.suspend") }

// ==================== KYC (tenant-scoped) ====================

func (s *Svc) ListKYC(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT k.id, k.user_id, k.doc_type, k.status, COALESCE(k.document_url,''), k.submitted_at,
		        COALESCE(k.reviewed_at, k.submitted_at), COALESCE(k.reject_reason,'')
		 FROM kyc_requests k JOIN users u ON k.user_id=u.id
		 WHERE u.white_label_id=$1 ORDER BY k.submitted_at DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, uid uuid.UUID
		var docType, status, docURL, reason string
		var submitted, reviewed time.Time
		_ = rows.Scan(&id, &uid, &docType, &status, &docURL, &submitted, &reviewed, &reason)
		out = append(out, gin.H{"id": id, "user_id": uid, "doc_type": docType, "status": status, "document_url": docURL, "submitted_at": submitted, "reviewed_at": reviewed, "reject_reason": reason})
	}
	c.JSON(http.StatusOK, gin.H{"kyc_requests": out})
}

func (s *Svc) reviewKYC(c *gin.Context, status, action string) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	ctx := c.Request.Context()
	adminID := middleware.AdminID(c)
	var ct int64
	if status == "approved" {
		tag, e := s.db.Exec(ctx,
			`UPDATE kyc_requests SET status='approved', reviewed_at=NOW(), reviewed_by=$1, reject_reason=''
			 WHERE id=$2 AND EXISTS (SELECT 1 FROM users u WHERE u.id=kyc_requests.user_id AND u.white_label_id=$3)`,
			adminID, id, tenantID)
		ct, err = tag.RowsAffected(), e
	} else {
		tag, e := s.db.Exec(ctx,
			`UPDATE kyc_requests SET status='rejected', reviewed_at=NOW(), reviewed_by=$1, reject_reason=$2
			 WHERE id=$3 AND EXISTS (SELECT 1 FROM users u WHERE u.id=kyc_requests.user_id AND u.white_label_id=$4)`,
			adminID, req.Reason, id, tenantID)
		ct, err = tag.RowsAffected(), e
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "kyc request not found"})
		return
	}
	s.audit(ctx, adminID, action, "kyc", id.String(), gin.H{"reason": req.Reason})
	c.JSON(http.StatusOK, gin.H{action: id})
}

func (s *Svc) ApproveKYC(c *gin.Context) { s.reviewKYC(c, "approved", "kyc.approve") }
func (s *Svc) RejectKYC(c *gin.Context)  { s.reviewKYC(c, "rejected", "kyc.reject") }

// ==================== Transactions (tenant-scoped, read-only) ====================

func (s *Svc) ListTransactions(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT t.id, t.user_id, t.type, t.amount, t.currency, t.status, COALESCE(t.from_address,''),
		        COALESCE(t.to_address,''), COALESCE(t.tx_hash,''), COALESCE(t.fee,0), t.chain_id, t.timestamp
		 FROM transactions t JOIN users u ON t.user_id=u.id
		 WHERE u.white_label_id=$1 ORDER BY t.timestamp DESC LIMIT 200`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, uid uuid.UUID
		var ttype, amount, currency, status, from, to, txhash string
		var fee string
		var chainID int
		var ts time.Time
		_ = rows.Scan(&id, &uid, &ttype, &amount, &currency, &status, &from, &to, &txhash, &fee, &chainID, &ts)
		out = append(out, gin.H{"id": id, "user_id": uid, "type": ttype, "amount": amount, "currency": currency, "status": status, "from_address": from, "to_address": to, "tx_hash": txhash, "fee": fee, "chain_id": chainID, "timestamp": ts})
	}
	c.JSON(http.StatusOK, gin.H{"transactions": out})
}

func (s *Svc) GetTransaction(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var uid uuid.UUID
	var ttype, amount, currency, status, from, to, txhash string
	var fee string
	var chainID int
	var ts time.Time
	err = s.db.QueryRow(ctx,
		`SELECT t.user_id, t.type, t.amount, t.currency, t.status, COALESCE(t.from_address,''),
		        COALESCE(t.to_address,''), COALESCE(t.tx_hash,''), COALESCE(t.fee,0), t.chain_id, t.timestamp
		 FROM transactions t JOIN users u ON t.user_id=u.id
		 WHERE t.id=$1 AND u.white_label_id=$2`, id, tenantID).
		Scan(&uid, &ttype, &amount, &currency, &status, &from, &to, &txhash, &fee, &chainID, &ts)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "user_id": uid, "type": ttype, "amount": amount, "currency": currency, "status": status, "from_address": from, "to_address": to, "tx_hash": txhash, "fee": fee, "chain_id": chainID, "timestamp": ts})
}

func (s *Svc) setTxFlag(c *gin.Context, flagged bool, action string) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	// flag = mark status as 'flagged'; unflag = restore to 'confirmed'
	status := "flagged"
	if !flagged {
		status = "confirmed"
	}
	ct, err := s.db.Exec(ctx,
		`UPDATE transactions SET status=$1 WHERE id=$2 AND EXISTS (SELECT 1 FROM users u, transactions t WHERE u.id=t.user_id AND u.white_label_id=$3 AND t.id=$2)`,
		status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), action, "transaction", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{action: id})
}

func (s *Svc) FlagTransaction(c *gin.Context)   { s.setTxFlag(c, true, "transaction.flag") }
func (s *Svc) UnflagTransaction(c *gin.Context) { s.setTxFlag(c, false, "transaction.unflag") }
