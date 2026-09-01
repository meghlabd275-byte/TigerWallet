package main

// finance_withdraw.go — signed withdrawal pipeline.
//
// Every withdrawal request is:
//   1. risk-scored (account age, velocity, daily volume, first-seen address,
//      USD tiers) — real signals from the ledger + withdrawal tables;
//   2. HMAC-SHA256 signed over its canonical payload (tamper-evident: the
//      superadmin approval path re-verifies the signature before paying out);
//   3. auto-approved in under a second when below WITHDRAW_AUTO_THRESHOLD
//      AND below the risk ceiling; anything larger or riskier queues for
//      superadmin sign-off;
//   4. on rejection the locked funds are automatically refunded (unlocked)
//      and a refund journal is written.
//
// Funds are locked at request time (available -> locked) so a queued
// withdrawal can never be double-spent; approval settles the lock, rejection
// releases it.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// withdrawalPayload is the canonical string that is HMAC-signed.
func withdrawalPayload(id, userID, currency, amount, toAddress string, createdAt time.Time) string {
	return strings.Join([]string{id, userID, currency, amount, toAddress,
		strconv.FormatInt(createdAt.Unix(), 10)}, "|")
}

func signWithdrawal(payload string) string {
	mac := hmac.New(sha256.New, financeCfg.withdrawHMAC)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// riskScoreWithdrawal computes a 0-100 risk score from real account signals.
func riskScoreWithdrawal(c *gin.Context, uid uuid.UUID, currency string, usd float64, usdKnown bool, toAddress string) (int, []string) {
	ctx := c.Request.Context()
	score := 0
	reasons := []string{}
	add := func(points int, reason string) {
		score += points
		reasons = append(reasons, reason)
	}

	// Account age.
	var createdAt time.Time
	if err := store.PG.QueryRow(ctx, `SELECT created_at FROM users WHERE id=$1`, uid).Scan(&createdAt); err == nil {
		age := time.Since(createdAt)
		switch {
		case age < 24*time.Hour:
			add(25, "account younger than 24 hours")
		case age < 7*24*time.Hour:
			add(10, "account younger than 7 days")
		}
	}

	// Velocity: requests in the last hour / 24h.
	var hourly, daily int
	_ = store.PG.QueryRow(ctx,
		`SELECT COUNT(*) FROM withdrawal_request WHERE user_id=$1 AND created_at > now() - interval '1 hour'`, uid).Scan(&hourly)
	_ = store.PG.QueryRow(ctx,
		`SELECT COUNT(*) FROM withdrawal_request WHERE user_id=$1 AND created_at > now() - interval '24 hours'`, uid).Scan(&daily)
	if hourly >= 3 {
		add(15, "3+ withdrawal requests in the last hour")
	}
	if daily >= 10 {
		add(10, "10+ withdrawal requests in the last 24 hours")
	}

	// Volume anomaly vs the user's own 30-day average.
	var avgDaily float64
	_ = store.PG.QueryRow(ctx,
		`SELECT COALESCE(AVG(day_total),0) FROM (
		   SELECT SUM(usd_value) AS day_total FROM withdrawal_request
		   WHERE user_id=$1 AND created_at > now() - interval '30 days' AND usd_value IS NOT NULL
		   GROUP BY date_trunc('day', created_at)) t`, uid).Scan(&avgDaily)
	if usdKnown && avgDaily > 0 && usd > 5*avgDaily {
		add(20, "amount exceeds 5x the account's average daily volume")
	}

	// First-seen destination address.
	var seen int
	_ = store.PG.QueryRow(ctx,
		`SELECT COUNT(*) FROM withdrawal_request WHERE user_id=$1 AND to_address=$2 AND status IN ('auto_approved','approved')`,
		uid, toAddress).Scan(&seen)
	if seen == 0 {
		add(10, "first withdrawal to this address")
	}

	// USD tiers.
	switch {
	case !usdKnown:
		add(15, "USD valuation unavailable (fail-safe)")
	case usd > 50000:
		add(40, "amount above $50,000")
	case usd > 10000:
		add(20, "amount above $10,000")
	}

	if score > 100 {
		score = 100
	}
	return score, reasons
}

// handleCreateWithdrawal creates a risk-scored, HMAC-signed withdrawal
// request. Auto-approves sub-threshold low-risk requests within a second.
func handleCreateWithdrawal(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		Currency  string `json:"currency" binding:"required"`
		Amount    string `json:"amount" binding:"required"`
		ToAddress string `json:"to_address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	start := time.Now()
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.ToAddress = strings.TrimSpace(req.ToAddress)
	amountF, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil || amountF <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	if len(req.ToAddress) < 20 || len(req.ToAddress) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid destination address"})
		return
	}
	ctx := c.Request.Context()
	if !switchEnabled(ctx, req.Currency, "withdraw") {
		c.JSON(http.StatusForbidden, gin.H{"error": "withdrawals are disabled for " + req.Currency})
		return
	}
	if len(financeCfg.withdrawHMAC) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "withdrawal signing is not configured on this node"})
		return
	}

	usd, usdKnown := usdValueOf(ctx, req.Currency, amountF)
	score, reasons := riskScoreWithdrawal(c, uid, req.Currency, usd, usdKnown, req.ToAddress)
	auto := usdKnown && usd < financeCfg.autoWithdrawUSD && score < financeCfg.riskAutoMax

	tx, err := store.PG.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ledger unavailable"})
		return
	}
	defer tx.Rollback(ctx)

	// Lock the funds first: a queued request can never be double-spent.
	if err := ledgerLock(ctx, tx, uid, req.Currency, req.Amount); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	status := "queued"
	if auto {
		status = "auto_approved"
	}
	sig := signWithdrawal(withdrawalPayload(id, uid.String(), req.Currency, req.Amount, req.ToAddress, now))

	var usdVal any
	if usdKnown {
		usdVal = usd
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO withdrawal_request
		   (id, user_id, currency, amount, to_address, usd_value, risk_score, risk_reasons, signature, status, created_at)
		 VALUES ($1,$2,$3,$4::numeric,$5,$6,$7,$8,$9,$10,$11)`,
		id, uid, req.Currency, req.Amount, req.ToAddress, usdVal, score, reasons, sig, status, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record withdrawal"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}

	resp := gin.H{
		"id": id, "status": status, "currency": req.Currency, "amount": req.Amount,
		"to_address": req.ToAddress, "risk_score": score, "risk_reasons": reasons,
		"signature": sig,
	}
	if usdKnown {
		resp["usd_value"] = usd
	}
	if auto {
		resp["approved_in_ms"] = time.Since(start).Milliseconds()
		resp["message"] = "Withdrawal auto-approved and released for broadcast"
	} else {
		resp["message"] = "Withdrawal queued for superadmin sign-off"
	}
	c.JSON(http.StatusOK, resp)
}

// handleListWithdrawals returns the caller's own withdrawal requests.
func handleListWithdrawals(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, currency, amount::text, to_address, COALESCE(usd_value::text,''), risk_score,
		        status, COALESCE(decision_reason,''), created_at
		 FROM withdrawal_request WHERE user_id=$1 ORDER BY created_at DESC LIMIT 200`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, cur, amt, to, usd, status, reason string
		var score int
		var created time.Time
		if err := rows.Scan(&id, &cur, &amt, &to, &usd, &score, &status, &reason, &created); err != nil {
			continue
		}
		out = append(out, gin.H{
			"id": id, "currency": cur, "amount": amt, "to_address": to,
			"usd_value": usd, "risk_score": score, "status": status,
			"decision_reason": reason, "created_at": created,
		})
	}
	c.JSON(http.StatusOK, gin.H{"withdrawals": out, "count": len(out)})
}

// ---------------------------------------------------------------------------
// Superadmin sign-off (dynamic-permission gated: withdrawals.sign)
// ---------------------------------------------------------------------------

// handleAdminListWithdrawals lists requests (default: the queued sign-off queue).
func handleAdminListWithdrawals(c *gin.Context) {
	if !requireFinancePerm(c, "withdrawals.sign") {
		return
	}
	status := c.DefaultQuery("status", "queued")
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, user_id, currency, amount::text, to_address, COALESCE(usd_value::text,''),
		        risk_score, risk_reasons, status, created_at
		 FROM withdrawal_request WHERE status=$1 ORDER BY created_at ASC LIMIT 500`, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, userID, cur, amt, to, usd, status string
		var score int
		var reasons []string
		var created time.Time
		if err := rows.Scan(&id, &userID, &cur, &amt, &to, &usd, &score, &reasons, &status, &created); err != nil {
			continue
		}
		out = append(out, gin.H{
			"id": id, "user_id": userID, "currency": cur, "amount": amt, "to_address": to,
			"usd_value": usd, "risk_score": score, "risk_reasons": reasons,
			"status": status, "created_at": created,
		})
	}
	c.JSON(http.StatusOK, gin.H{"withdrawals": out, "count": len(out)})
}

// handleAdminApproveWithdrawal verifies the stored HMAC signature (tamper
// check), settles the locked funds, and marks the request approved.
func handleAdminApproveWithdrawal(c *gin.Context) {
	if !requireFinancePerm(c, "withdrawals.sign") {
		return
	}
	adminID, _ := uuid.Parse(getUserID(c))
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	ctx := c.Request.Context()
	tx, err := store.PG.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ledger unavailable"})
		return
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	var cur, amt, to, sig, status string
	var created time.Time
	err = tx.QueryRow(ctx,
		`SELECT user_id, currency, amount::text, to_address, signature, status, created_at
		 FROM withdrawal_request WHERE id=$1 FOR UPDATE`, c.Param("id")).
		Scan(&userID, &cur, &amt, &to, &sig, &status, &created)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "withdrawal not found"})
		return
	}
	if status != "queued" {
		c.JSON(http.StatusConflict, gin.H{"error": "withdrawal is not in the sign-off queue (status: " + status + ")"})
		return
	}
	// Tamper-evidence: recompute the signature over the stored payload.
	expect := signWithdrawal(withdrawalPayload(c.Param("id"), userID.String(), cur, amt, to, created))
	if !hmac.Equal([]byte(expect), []byte(sig)) {
		c.JSON(http.StatusConflict, gin.H{"error": "signature mismatch — refusing to approve a tampered request"})
		return
	}
	if err := ledgerSettleLocked(ctx, tx, userID, cur, amt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO ledger_journal(kind, reference, memo) VALUES('withdraw', $1, $2)`,
		c.Param("id"), "withdrawal approved by superadmin: "+amt+" "+cur+" -> "+to); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "journal failed"})
		return
	}
	if _, err := tx.Exec(ctx,
		`UPDATE withdrawal_request SET status='approved', decided_by=$2, decided_at=now(), decision_reason=$3 WHERE id=$1`,
		c.Param("id"), adminID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	auditFinance(ctx, adminID, getUserRole(c), "withdrawal.approve", c.Param("id"),
		gin.H{"currency": cur, "amount": amt, "to": to, "reason": req.Reason})
	c.JSON(http.StatusOK, gin.H{"status": "approved", "id": c.Param("id")})
}

// handleAdminRejectWithdrawal rejects a queued request and automatically
// refunds the locked funds back to the user's available balance.
func handleAdminRejectWithdrawal(c *gin.Context) {
	if !requireFinancePerm(c, "withdrawals.sign") {
		return
	}
	adminID, _ := uuid.Parse(getUserID(c))
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	ctx := c.Request.Context()
	tx, err := store.PG.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ledger unavailable"})
		return
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	var cur, amt, status string
	err = tx.QueryRow(ctx,
		`SELECT user_id, currency, amount::text, status FROM withdrawal_request WHERE id=$1 FOR UPDATE`,
		c.Param("id")).Scan(&userID, &cur, &amt, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "withdrawal not found"})
		return
	}
	if status != "queued" {
		c.JSON(http.StatusConflict, gin.H{"error": "withdrawal is not in the sign-off queue (status: " + status + ")"})
		return
	}
	// Auto-refund: release the lock so the funds return to available balance.
	if err := ledgerUnlock(ctx, tx, userID, cur, amt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund failed: " + err.Error()})
		return
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO ledger_journal(kind, reference, memo) VALUES('refund', $1, $2)`,
		c.Param("id"), "withdrawal rejected — funds auto-refunded: "+amt+" "+cur); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "journal failed"})
		return
	}
	if _, err := tx.Exec(ctx,
		`UPDATE withdrawal_request SET status='rejected', decided_by=$2, decided_at=now(), decision_reason=$3 WHERE id=$1`,
		c.Param("id"), adminID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	auditFinance(ctx, adminID, getUserRole(c), "withdrawal.reject", c.Param("id"),
		gin.H{"currency": cur, "amount": amt, "reason": req.Reason})
	c.JSON(http.StatusOK, gin.H{"status": "rejected", "refunded": true, "id": c.Param("id")})
}

// recomputeWithdrawalSignature is exported for tests.
func recomputeWithdrawalSignature(id, userID, currency, amount, toAddress string, createdAt time.Time) string {
	return signWithdrawal(withdrawalPayload(id, userID, currency, amount, toAddress, createdAt))
}

var _ = fmt.Sprintf // keep fmt imported for future error wrapping
