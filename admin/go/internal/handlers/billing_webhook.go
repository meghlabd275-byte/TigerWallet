package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Stipe-webhook integration for admin billing. Invoices are created in "open"
// status when a subscription starts; only a signature-verified payment
// processor callback (Stripe) transitions them to "paid". A webhook arriving
// with an unconfigured secret is rejected (fail-closed) so spoofed callbacks
// can never fabricate payment state.

// stripeWebhookEvent is a minimal representation of a Stripe event envelope.
type stripeWebhookEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// stripeWebhookObject is the `data.object` portion; only the fields needed to
// correlate the event to a local invoice are decoded.
type stripeWebhookObject struct {
	ID       string            `json:"id"`
	Number   string            `json:"number"`
	Metadata map[string]string `json:"metadata"`
}

// StripePaymentWebhook handles POST /webhooks/stripe. It verifies the
// Stripe-Signature header over the raw body and advances the referenced
// invoice from "open" to "paid" (forward-only), then records the paid time.
func (h *BillingHandler) StripePaymentWebhook(c *gin.Context) {
	secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	signature := c.GetHeader("Stripe-Signature")

	payload, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to read request body"})
		return
	}

	if err := verifyStripeWebhookSignature(payload, signature, secret); err != nil {
		// 400 (not 401) so a misconfigured secret is discovered early rather
		// than silently dropping real events.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var event stripeWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload"})
		return
	}

	switch event.Type {
	case "invoice.paid", "invoice.payment_succeeded", "checkout.session.completed", "payment_intent.succeeded":
		// Implemented below.
	case "invoice.payment_failed", "checkout.session.async_payment_failed", "payment_intent.payment_failed":
		// Payment failure is intentionally a no-op here: an unsuccessful
		// attempt must not mutate subscription state. Failures are handled by
		// dunning/retry in the processor; we only act on confirmed success.
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	default:
		// Unknown or unhandled event types are acknowledged without mutation.
		c.JSON(http.StatusOK, gin.H{"received": true, "ignored": true})
		return
	}

	var data struct {
		Object stripeWebhookObject `json:"object"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event data"})
		return
	}

	invoiceNumber := data.Object.Metadata["invoice_number"]
	if invoiceNumber == "" {
		invoiceNumber = data.Object.Number
	}
	if invoiceNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event has no invoice reference"})
		return
	}

	paid, err := h.markInvoicePaid(invoiceNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !paid {
		c.JSON(http.StatusOK, gin.H{"received": true, "already_paid": true})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true, "invoice_number": invoiceNumber, "status": "paid"})
}

// markInvoicePaid transitions an invoice from "open" to "paid". The update is
// guarded by `status = 'open'` so a duplicate or replayed webhook can never
// touch an already-terminal invoice.
func (h *BillingHandler) markInvoicePaid(invoiceNumber string) (bool, error) {
	now := time.Now()
	res := h.db.DB.Model(&Invoice{}).
		Where("invoice_number = ? AND status = ?", invoiceNumber, "open").
		Updates(map[string]interface{}{"status": "paid", "paid_at": now})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil
	}

	// Reflect the successful charge on the associated subscription so the
	// subscription record stays consistent with the invoice lifecycle.
	var inv Invoice
	if err := h.db.DB.First(&inv, "invoice_number = ?", invoiceNumber).Error; err != nil {
		return true, nil // invoice already marked paid; subscription update is best-effort
	}
	if err := h.db.DB.Model(&Subscription{}).
		Where("id = ? AND status = ?", inv.SubscriptionID, "active").
		Update("status", "active").Error; err != nil {
		return true, nil // best-effort; invoice state is authoritative
	}
	return true, nil
}

// verifyStripeWebhookSignature validates the Stripe-Signature header
// (t=...,v1=...) against the raw body using the configured webhook secret.
// Mirrors go/fiat_ramp/webhooks.go: the scheme is HMAC-SHA256 over
// `timestamp + "." + payload`, compared in constant time. Events older than 5
// minutes are rejected to bound replay.
func verifyStripeWebhookSignature(payload []byte, header, secret string) error {
	if secret == "" {
		return errors.New("stripe webhook secret not configured")
	}
	var timestamp, sig string
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "t="):
			timestamp = strings.TrimPrefix(part, "t=")
		case strings.HasPrefix(part, "v1="):
			sig = strings.TrimPrefix(part, "v1=")
		}
	}
	if timestamp == "" || sig == "" {
		return errors.New("malformed stripe signature header")
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid stripe webhook timestamp")
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return errors.New("stale stripe webhook timestamp")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return errors.New("invalid stripe webhook signature")
	}
	return nil
}
