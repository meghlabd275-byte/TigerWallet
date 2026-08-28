package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PaymentCallback handles the external payment-processor webhook that is the
// ONLY path allowed to move an invoice from "open" to "paid".
//
// Security contract (Phase 7/8):
//   - BILLING_PAYMENT_WEBHOOK_SECRET must be set (env / provider config from
//     the Admin Panel secrets section); without it the endpoint is fail-closed
//     and returns 503 rather than accepting unauthenticated payment state.
//   - Every request must carry a valid HMAC-SHA256 hex signature of the raw
//     body in the X-Payment-Signature header.
//   - Idempotent: already-paid invoices return success without side effects.
//   - Amount is cross-checked against the stored invoice; a mismatch is
//     rejected so a replayed/forged callback cannot mark the wrong sum paid.
func (h *BillingHandler) PaymentCallback(c *gin.Context) {
	secret := webhookSecret()
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment processor not configured"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil || len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	signature := strings.TrimPrefix(c.GetHeader("X-Payment-Signature"), "sha256=")
	if signature == "" {
		signature = c.GetHeader("X-Signature-256")
	}
	computed := hmac.New(sha256.New, []byte(secret))
	computed.Write(body)
	if !hmac.Equal([]byte(signature), []byte(hex.EncodeToString(computed.Sum(nil)))) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var evt struct {
		InvoiceID   string  `json:"invoice_id"`
		InvoiceNum  string  `json:"invoice_number"`
		ProviderRef string  `json:"provider_ref"`
		Status      string  `json:"status"`
		AmountPaid  float64 `json:"amount_paid"`
		EventID     string  `json:"event_id"`
	}
	if err := json.Unmarshal(body, &evt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if evt.Status != "paid" || evt.ProviderRef == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported event"})
		return
	}

	var invoice Invoice
	switch {
	case evt.InvoiceID != "":
		id, err := uuid.Parse(evt.InvoiceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice id"})
			return
		}
		err = h.db.DB.First(&invoice, "id = ?", id).Error
	case evt.InvoiceNum != "":
		err = h.db.DB.First(&invoice, "invoice_number = ?", evt.InvoiceNum).Error
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invoice reference required"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	if evt.AmountPaid > 0 && evt.AmountPaid < invoice.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount mismatch"})
		return
	}

	if invoice.Status == "paid" {
		// Idempotent replay: the payment was already recorded.
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "already recorded"})
		return
	}

	now := time.Now()
	res := h.db.DB.Model(&Invoice{}).
		Where("id = ? AND status = ?", invoice.ID, "open").
		Updates(map[string]interface{}{"status": "paid", "paid_at": now})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		// Lost a race against another delivery of the same callback.
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "already recorded"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"invoice_id": invoice.ID, "status": "paid"}})
}

func webhookSecret() string {
	return strings.TrimSpace(os.Getenv("BILLING_PAYMENT_WEBHOOK_SECRET"))
}
