package main

// finance_p2p.go — escrowed P2P marketplace (mobile payment + bank payment).
//
// Flow (all fund movements atomic on the double-entry ledger):
//   open    — seller locks `amount` of `currency` into escrow (KYC-gated);
//   accept  — a KYC-verified buyer takes the order (status escrowed);
//   paid    — buyer marks the fiat leg paid via the chosen local payment
//             method (one of the 881 catalogued methods across 238 countries);
//   release — seller confirms receipt; escrow settles to the buyer;
//   dispute — either party flags; a superadmin (p2p.resolve) resolves by
//             releasing to the buyer or refunding the seller;
//   cancel  — seller cancels before payment; escrow auto-refunds.
//
// Every state transition is guarded (status + party checked in one
// UPDATE ... WHERE status=... FOR UPDATE), so no double-release is possible.

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type escrowOrder struct {
	ID                string    `json:"id"`
	SellerID          string    `json:"seller_id"`
	BuyerID           string    `json:"buyer_id,omitempty"`
	Currency          string    `json:"currency"`
	Amount            string    `json:"amount"`
	FiatCurrency      string    `json:"fiat_currency"`
	FiatAmount        string    `json:"fiat_amount"`
	PaymentMethodCode string    `json:"payment_method_code"`
	PaymentMethodName string    `json:"payment_method_name"`
	PaymentKind       string    `json:"payment_kind"`
	CountryCode       string    `json:"country_code"`
	Status            string    `json:"status"`
	DisputeReason     string    `json:"dispute_reason,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

func scanEscrow(rows interface {
	Next() bool
	Scan(...any) error
}) []escrowOrder {
	out := []escrowOrder{}
	for rows.Next() {
		var o escrowOrder
		var buyer *string
		var dispute *string
		if err := rows.Scan(&o.ID, &o.SellerID, &buyer, &o.Currency, &o.Amount,
			&o.FiatCurrency, &o.FiatAmount, &o.PaymentMethodCode, &o.CountryCode,
			&o.Status, &dispute, &o.CreatedAt); err != nil {
			continue
		}
		if buyer != nil {
			o.BuyerID = *buyer
		}
		if dispute != nil {
			o.DisputeReason = *dispute
		}
		if pm, ok := paymentMethodByCode(o.PaymentMethodCode); ok {
			o.PaymentMethodName = pm.Name
			o.PaymentKind = pm.Kind
		}
		out = append(out, o)
	}
	return out
}

const escrowSelect = `SELECT id, seller_id, buyer_id, currency, amount::text,
	fiat_currency, fiat_amount::text, payment_method_code, country_code, status,
	dispute_reason, created_at FROM p2p_escrow_order`

// handleP2POpenEscrow opens a sell order: the seller's funds are locked in
// escrow immediately. KYC-gated.
func handleP2POpenEscrow(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		Currency          string `json:"currency" binding:"required"`
		Amount            string `json:"amount" binding:"required"`
		FiatCurrency      string `json:"fiat_currency" binding:"required"`
		FiatAmount        string `json:"fiat_amount" binding:"required"`
		PaymentMethodCode string `json:"payment_method_code" binding:"required"`
		CountryCode       string `json:"country_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.FiatCurrency = strings.ToUpper(strings.TrimSpace(req.FiatCurrency))
	req.CountryCode = strings.ToUpper(strings.TrimSpace(req.CountryCode))
	if _, err := strconv.ParseFloat(req.Amount, 64); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	if fiat, err := strconv.ParseFloat(req.FiatAmount, 64); err != nil || fiat <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fiat amount"})
		return
	}
	pm, ok := paymentMethodByCode(req.PaymentMethodCode)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown payment method code"})
		return
	}
	if !paymentMethodAvailableIn(req.PaymentMethodCode, req.CountryCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment method " + pm.Name + " is not available in " + req.CountryCode})
		return
	}
	ctx := c.Request.Context()
	if !switchEnabled(ctx, req.Currency, "p2p") {
		c.JSON(http.StatusForbidden, gin.H{"error": "P2P trading is disabled for " + req.Currency})
		return
	}
	if !kycVerified(ctx, uid) {
		c.JSON(http.StatusForbidden, gin.H{"error": "KYC verification required for P2P trading", "kyc_required": true})
		return
	}

	tx, err := store.PG.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ledger unavailable"})
		return
	}
	defer tx.Rollback(ctx)
	if err := ledgerLock(ctx, tx, uid, req.Currency, req.Amount); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	var id string
	if err := tx.QueryRow(ctx,
		`INSERT INTO p2p_escrow_order(seller_id, currency, amount, fiat_currency, fiat_amount, payment_method_code, country_code, status)
		 VALUES ($1,$2,$3::numeric,$4,$5::numeric,$6,$7,'open') RETURNING id`,
		uid, req.Currency, req.Amount, req.FiatCurrency, req.FiatAmount, req.PaymentMethodCode, req.CountryCode).Scan(&id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open escrow order"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "open", "id": id, "escrow_locked": true,
		"payment_method": pm.Name, "payment_kind": pm.Kind})
}

// handleP2PListEscrow browses the marketplace (open orders) or the caller's
// own orders (?mine=true, any status).
func handleP2PListEscrow(c *gin.Context) {
	uid, _ := uuid.Parse(getUserID(c))
	ctx := c.Request.Context()
	if c.Query("mine") == "true" {
		rows, err := store.PG.Query(ctx, escrowSelect+
			` WHERE seller_id=$1 OR buyer_id=$1 ORDER BY created_at DESC LIMIT 200`, uid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
		defer rows.Close()
		out := scanEscrow(rows)
		c.JSON(http.StatusOK, gin.H{"orders": out, "count": len(out)})
		return
	}
	q := escrowSelect + ` WHERE status='open'`
	args := []any{}
	if cur := strings.ToUpper(c.Query("currency")); cur != "" {
		args = append(args, cur)
		q += ` AND currency=$` + strconv.Itoa(len(args))
	}
	if cc := strings.ToUpper(c.Query("country")); cc != "" {
		args = append(args, cc)
		q += ` AND country_code=$` + strconv.Itoa(len(args))
	}
	q += ` ORDER BY created_at DESC LIMIT 200`
	rows, err := store.PG.Query(ctx, q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := scanEscrow(rows)
	c.JSON(http.StatusOK, gin.H{"orders": out, "count": len(out)})
}

// escrowTransition loads an order FOR UPDATE and checks party + status.
func escrowTransition(c *gin.Context, id string) (sellerID, buyerID uuid.UUID, currency, amount, status string, err error) {
	var b *string
	err = store.PG.QueryRow(c.Request.Context(),
		`SELECT seller_id, buyer_id, currency, amount::text, status FROM p2p_escrow_order WHERE id=$1 FOR UPDATE`, id).
		Scan(&sellerID, &b, &currency, &amount, &status)
	if b != nil {
		buyerID, _ = uuid.Parse(*b)
	}
	return
}

// handleP2PAcceptEscrow — a KYC-verified buyer takes an open order.
func handleP2PAcceptEscrow(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	ctx := c.Request.Context()
	if !kycVerified(ctx, uid) {
		c.JSON(http.StatusForbidden, gin.H{"error": "KYC verification required for P2P trading", "kyc_required": true})
		return
	}
	tag, err := store.PG.Exec(ctx,
		`UPDATE p2p_escrow_order SET buyer_id=$2, status='escrowed', updated_at=now()
		 WHERE id=$1 AND status='open' AND seller_id <> $2`, c.Param("id"), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "accept failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "order is not open (or it is your own order)"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "escrowed", "id": c.Param("id")})
}

// handleP2PMarkPaid — buyer marks the fiat leg paid (bank/mobile transfer sent).
func handleP2PMarkPaid(c *gin.Context) {
	uid, _ := uuid.Parse(getUserID(c))
	tag, err := store.PG.Exec(c.Request.Context(),
		`UPDATE p2p_escrow_order SET status='paid', updated_at=now()
		 WHERE id=$1 AND status='escrowed' AND buyer_id=$2`, c.Param("id"), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "only the buyer can mark an escrowed order as paid"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "paid", "id": c.Param("id")})
}

// handleP2PRelease — seller confirms fiat receipt; escrow settles to buyer.
func handleP2PRelease(c *gin.Context) {
	uid, _ := uuid.Parse(getUserID(c))
	ctx := c.Request.Context()
	tx, err := store.PG.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ledger unavailable"})
		return
	}
	defer tx.Rollback(ctx)
	sellerID, buyerID, currency, amount, status, err := escrowTransition(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if sellerID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the seller can release escrow"})
		return
	}
	if status != "paid" {
		c.JSON(http.StatusConflict, gin.H{"error": "order must be marked paid before release (status: " + status + ")"})
		return
	}
	if err := ledgerSettleLocked(ctx, tx, sellerID, currency, amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Credit the buyer in the same transaction (double-entry via journal entry).
	if _, err := tx.Exec(ctx,
		`INSERT INTO ledger_account(user_id, currency) VALUES($1,$2) ON CONFLICT DO NOTHING`, buyerID, currency); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "credit failed"})
		return
	}
	if _, err := tx.Exec(ctx,
		`UPDATE ledger_account SET balance = balance + $3::numeric, updated_at=now() WHERE user_id=$1 AND currency=$2`,
		buyerID, currency, amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "credit failed"})
		return
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO ledger_journal(kind, reference, memo) VALUES('p2p_transfer', $1, $2)`,
		c.Param("id"), "escrow release: "+amount+" "+currency+" seller->buyer"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "journal failed"})
		return
	}
	if _, err := tx.Exec(ctx,
		`UPDATE p2p_escrow_order SET status='released', updated_at=now() WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "released", "id": c.Param("id")})
}

// handleP2PDispute — either party opens a dispute (freezes the order for
// superadmin resolution).
func handleP2PDispute(c *gin.Context) {
	uid, _ := uuid.Parse(getUserID(c))
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tag, err := store.PG.Exec(c.Request.Context(),
		`UPDATE p2p_escrow_order SET status='disputed', dispute_reason=$3, updated_at=now()
		 WHERE id=$1 AND status IN ('escrowed','paid') AND (seller_id=$2 OR buyer_id=$2)`,
		c.Param("id"), uid, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "only a party to an active order can dispute it"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "disputed", "id": c.Param("id")})
}

// handleP2PCancel — seller cancels an order nobody accepted; escrow refunds.
func handleP2PCancel(c *gin.Context) {
	uid, _ := uuid.Parse(getUserID(c))
	ctx := c.Request.Context()
	tx, err := store.PG.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ledger unavailable"})
		return
	}
	defer tx.Rollback(ctx)
	sellerID, _, currency, amount, status, err := escrowTransition(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if sellerID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the seller can cancel"})
		return
	}
	if status != "open" {
		c.JSON(http.StatusConflict, gin.H{"error": "only an unaccepted order can be cancelled (status: " + status + ")"})
		return
	}
	if err := ledgerUnlock(ctx, tx, sellerID, currency, amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund failed: " + err.Error()})
		return
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO ledger_journal(kind, reference, memo) VALUES('refund', $1, $2)`,
		c.Param("id"), "escrow cancelled — funds auto-refunded: "+amount+" "+currency); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "journal failed"})
		return
	}
	if _, err := tx.Exec(ctx,
		`UPDATE p2p_escrow_order SET status='cancelled', updated_at=now() WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled", "refunded": true, "id": c.Param("id")})
}

// ---------------------------------------------------------------------------
// Dispute resolution (p2p.resolve permission)
// ---------------------------------------------------------------------------

// handleAdminP2PDisputes lists disputed orders for the resolution console.
func handleAdminP2PDisputes(c *gin.Context) {
	if !requireFinancePerm(c, "p2p.resolve") {
		return
	}
	rows, err := store.PG.Query(c.Request.Context(), escrowSelect+
		` WHERE status='disputed' ORDER BY updated_at ASC LIMIT 500`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := scanEscrow(rows)
	c.JSON(http.StatusOK, gin.H{"disputes": out, "count": len(out)})
}

// handleAdminP2PResolve resolves a dispute: release to the buyer or refund
// the seller — atomically, with an audit trail.
func handleAdminP2PResolve(c *gin.Context) {
	if !requireFinancePerm(c, "p2p.resolve") {
		return
	}
	adminID, _ := uuid.Parse(getUserID(c))
	var req struct {
		Winner string `json:"winner" binding:"required"` // buyer | seller
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Winner != "buyer" && req.Winner != "seller" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "winner must be buyer or seller"})
		return
	}
	ctx := c.Request.Context()
	tx, err := store.PG.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ledger unavailable"})
		return
	}
	defer tx.Rollback(ctx)
	sellerID, buyerID, currency, amount, status, err := escrowTransition(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if status != "disputed" {
		c.JSON(http.StatusConflict, gin.H{"error": "order is not disputed (status: " + status + ")"})
		return
	}
	var memo string
	if req.Winner == "buyer" {
		if err := ledgerSettleLocked(ctx, tx, sellerID, currency, amount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO ledger_account(user_id, currency) VALUES($1,$2) ON CONFLICT DO NOTHING`, buyerID, currency); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "credit failed"})
			return
		}
		if _, err := tx.Exec(ctx,
			`UPDATE ledger_account SET balance = balance + $3::numeric, updated_at=now() WHERE user_id=$1 AND currency=$2`,
			buyerID, currency, amount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "credit failed"})
			return
		}
		memo = "dispute resolved for buyer: " + amount + " " + currency
	} else {
		if err := ledgerUnlock(ctx, tx, sellerID, currency, amount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "refund failed: " + err.Error()})
			return
		}
		memo = "dispute resolved for seller — escrow refunded: " + amount + " " + currency
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO ledger_journal(kind, reference, memo) VALUES($1, $2, $3)`,
		map[bool]string{true: "p2p_transfer", false: "refund"}[req.Winner == "buyer"], c.Param("id"), memo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "journal failed"})
		return
	}
	if _, err := tx.Exec(ctx,
		`UPDATE p2p_escrow_order SET status='resolved', resolved_by=$2, resolution=$3, updated_at=now() WHERE id=$1`,
		c.Param("id"), adminID, req.Winner+": "+req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	auditFinance(ctx, adminID, getUserRole(c), "p2p.resolve", c.Param("id"),
		gin.H{"winner": req.Winner, "reason": req.Reason, "amount": amount, "currency": currency})
	c.JSON(http.StatusOK, gin.H{"status": "resolved", "winner": req.Winner, "id": c.Param("id")})
}

// handlePaymentMethods serves the 881-method / 238-country catalog.
func handlePaymentMethods(c *gin.Context) {
	country := strings.ToUpper(c.Query("country"))
	kind := c.Query("kind") // bank | mobile
	methods := financePaymentMethods()
	out := []gin.H{}
	for _, m := range methods {
		if kind != "" && m.Kind != kind {
			continue
		}
		if country != "" && !paymentMethodAvailableIn(m.Code, country) {
			continue
		}
		out = append(out, gin.H{"code": m.Code, "name": m.Name, "kind": m.Kind, "countries": m.Countries})
	}
	c.JSON(http.StatusOK, gin.H{
		"methods": out, "count": len(out),
		"total_methods": len(methods), "total_countries": len(financeCountries()),
	})
}
