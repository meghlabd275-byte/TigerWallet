package main

// finance_convert.go — instant convert engine on admin-managed rates.
//
// There is NO fallback price source for conversions: a currency pair can
// only be converted when an admin (rates.manage permission) has set an
// explicit rate. A conversion is one atomic journal: debit the source
// currency, credit amount*rate of the target currency. Fail-closed when no
// rate is configured.

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// handleListConvertRates returns the admin-managed rate book (public to
// authenticated users so clients can quote before converting).
func handleListConvertRates(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT from_currency, to_currency, rate::text, updated_at FROM convert_rate ORDER BY from_currency, to_currency`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rate query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var from, to, rate string
		var updated time.Time
		if err := rows.Scan(&from, &to, &rate, &updated); err != nil {
			continue
		}
		out = append(out, gin.H{"from_currency": from, "to_currency": to, "rate": rate, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"rates": out, "count": len(out)})
}

// handleConvert executes an instant conversion at the admin-managed rate.
func handleConvert(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		FromCurrency string `json:"from_currency" binding:"required"`
		ToCurrency   string `json:"to_currency" binding:"required"`
		Amount       string `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.FromCurrency = strings.ToUpper(strings.TrimSpace(req.FromCurrency))
	req.ToCurrency = strings.ToUpper(strings.TrimSpace(req.ToCurrency))
	if req.FromCurrency == req.ToCurrency {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot convert a currency to itself"})
		return
	}
	amountF, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil || amountF <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	ctx := c.Request.Context()
	if !switchEnabled(ctx, req.FromCurrency, "convert") || !switchEnabled(ctx, req.ToCurrency, "convert") {
		c.JSON(http.StatusForbidden, gin.H{"error": "conversions are disabled for this pair"})
		return
	}

	tx, err := store.PG.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ledger unavailable"})
		return
	}
	defer tx.Rollback(ctx)

	// Admin-managed rate — fail-closed when the pair is not configured.
	var rate string
	err = tx.QueryRow(ctx,
		`SELECT rate::text FROM convert_rate WHERE from_currency=$1 AND to_currency=$2`,
		req.FromCurrency, req.ToCurrency).Scan(&rate)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no admin-managed rate configured for " + req.FromCurrency + "/" + req.ToCurrency})
		return
	}

	// to_amount = amount * rate, computed by PostgreSQL NUMERIC (no float drift).
	var toAmount string
	if err := tx.QueryRow(ctx, `SELECT ($1::numeric * $2::numeric)::text`, req.Amount, rate).Scan(&toAmount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rate computation failed"})
		return
	}

	orderID := uuid.NewString()
	journalID, err := ledgerPost(ctx, tx, "convert", orderID,
		"convert "+req.Amount+" "+req.FromCurrency+" -> "+toAmount+" "+req.ToCurrency,
		[]ledgerLeg{
			{UserID: uid, Currency: req.FromCurrency, Amount: req.Amount, Debit: true},
			{UserID: uid, Currency: req.ToCurrency, Amount: toAmount, Debit: false},
		})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO convert_order(id, user_id, from_currency, to_currency, from_amount, to_amount, rate)
		 VALUES ($1,$2,$3,$4,$5::numeric,$6::numeric,$7::numeric)`,
		orderID, uid, req.FromCurrency, req.ToCurrency, req.Amount, toAmount, rate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record conversion"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "converted", "id": orderID, "journal_id": journalID,
		"from_currency": req.FromCurrency, "to_currency": req.ToCurrency,
		"from_amount": req.Amount, "to_amount": toAmount, "rate": rate,
	})
}

// handleListConverts returns the caller's conversion history.
func handleListConverts(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, from_currency, to_currency, from_amount::text, to_amount::text, rate::text, created_at
		 FROM convert_order WHERE user_id=$1 ORDER BY created_at DESC LIMIT 200`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, from, to, fa, ta, rate string
		var created time.Time
		if err := rows.Scan(&id, &from, &to, &fa, &ta, &rate, &created); err != nil {
			continue
		}
		out = append(out, gin.H{
			"id": id, "from_currency": from, "to_currency": to,
			"from_amount": fa, "to_amount": ta, "rate": rate, "created_at": created,
		})
	}
	c.JSON(http.StatusOK, gin.H{"conversions": out, "count": len(out)})
}

// ---------------------------------------------------------------------------
// Admin rate management (rates.manage permission)
// ---------------------------------------------------------------------------

// handleAdminSetConvertRate creates or updates a rate for a pair.
func handleAdminSetConvertRate(c *gin.Context) {
	if !requireFinancePerm(c, "rates.manage") {
		return
	}
	adminID, _ := uuid.Parse(getUserID(c))
	var req struct {
		FromCurrency string `json:"from_currency" binding:"required"`
		ToCurrency   string `json:"to_currency" binding:"required"`
		Rate         string `json:"rate" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.FromCurrency = strings.ToUpper(strings.TrimSpace(req.FromCurrency))
	req.ToCurrency = strings.ToUpper(strings.TrimSpace(req.ToCurrency))
	rateF, err := strconv.ParseFloat(req.Rate, 64)
	if err != nil || rateF <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rate must be a positive number"})
		return
	}
	if req.FromCurrency == req.ToCurrency {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pair must be two different currencies"})
		return
	}
	if _, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO convert_rate(from_currency, to_currency, rate, updated_by, updated_at)
		 VALUES ($1,$2,$3::numeric,$4,now())
		 ON CONFLICT (from_currency, to_currency)
		 DO UPDATE SET rate=$3::numeric, updated_by=$4, updated_at=now()`,
		req.FromCurrency, req.ToCurrency, req.Rate, adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rate update failed"})
		return
	}
	auditFinance(c.Request.Context(), adminID, getUserRole(c), "convert_rate.set",
		req.FromCurrency+"/"+req.ToCurrency, gin.H{"rate": req.Rate})
	c.JSON(http.StatusOK, gin.H{"status": "rate_set", "from_currency": req.FromCurrency, "to_currency": req.ToCurrency, "rate": req.Rate})
}

// handleAdminDeleteConvertRate removes a pair from the rate book.
func handleAdminDeleteConvertRate(c *gin.Context) {
	if !requireFinancePerm(c, "rates.manage") {
		return
	}
	adminID, _ := uuid.Parse(getUserID(c))
	from := strings.ToUpper(c.Query("from"))
	to := strings.ToUpper(c.Query("to"))
	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to query params required"})
		return
	}
	tag, err := store.PG.Exec(c.Request.Context(),
		`DELETE FROM convert_rate WHERE from_currency=$1 AND to_currency=$2`, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "rate not found"})
		return
	}
	auditFinance(c.Request.Context(), adminID, getUserRole(c), "convert_rate.delete", from+"/"+to, nil)
	c.JSON(http.StatusOK, gin.H{"status": "rate_deleted", "from_currency": from, "to_currency": to})
}
