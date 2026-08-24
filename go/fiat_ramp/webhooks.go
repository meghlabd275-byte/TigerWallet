package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Provider webhook handlers. Orders are created in "pending" status and only
// transition to a terminal state when the payment provider confirms the
// outcome via a signature-verified webhook. A webhook with an unconfigured
// secret is rejected (fail-closed) so spoofed callbacks can never mark an
// order paid.

// getWebhookSecret resolves the webhook signing secret for a provider. The
// admin-configured runtime key (same store as the provider API key) takes
// precedence, then the provider-specific env var.
func (s *FiatRampService) getWebhookSecret(providerID string) string {
	switch providerID {
	case "stripe":
		if v := os.Getenv("STRIPE_WEBHOOK_SECRET"); v != "" {
			return v
		}
	case "moonpay":
		if v := os.Getenv("MOONPAY_SECRET_KEY"); v != "" {
			return v
		}
	case "transak":
		if v := os.Getenv("TRANSAK_SECRET_KEY"); v != "" {
			return v
		}
	}
	return s.getProviderKey(providerID + "_webhook")
}

// updateOrderStatusByProviderRef moves an order to a new status, matched by
// the provider's order/reference id stored in the order id or recipient
// metadata. Only forward transitions are applied (pending -> processing ->
// completed/failed/expired) so a late duplicate webhook cannot resurrect a
// finished order.
func (s *FiatRampService) updateOrderStatus(ctx context.Context, orderID, newStatus string) error {
	res, err := s.pg.Exec(ctx,
		`UPDATE fiat_ramp_orders SET status = $2, updated_at = $3
		 WHERE id = $1 AND status IN ('pending','processing')`,
		orderID, newStatus, time.Now().Unix())
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("order %s not found or already terminal", orderID)
	}
	return nil
}

// verifyStripeSignature validates the Stripe-Signature header (t=...,v1=...)
// against the raw body using the configured webhook secret.
func verifyStripeSignature(payload []byte, header, secret string) error {
	if secret == "" {
		return fmt.Errorf("stripe webhook secret not configured")
	}
	var timestamp, sig string
	for _, part := range strings.Split(header, ",") {
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			sig = strings.TrimPrefix(part, "v1=")
		}
	}
	if timestamp == "" || sig == "" {
		return fmt.Errorf("malformed stripe signature header")
	}
	// Reject events older than 5 minutes to prevent replay.
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return fmt.Errorf("stale stripe webhook timestamp")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return fmt.Errorf("invalid stripe webhook signature")
	}
	return nil
}

// verifyHMACBody validates a simple hex HMAC-SHA256 over the raw body
// (MoonPay `Moonpay-Signature`, Transak `x-transak-signature`).
func verifyHMACBody(payload []byte, signature, secret, provider string) error {
	if secret == "" {
		return fmt.Errorf("%s webhook secret not configured", provider)
	}
	if signature == "" {
		return fmt.Errorf("missing %s webhook signature", provider)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return fmt.Errorf("invalid %s webhook signature", provider)
	}
	return nil
}

// stripeEventToStatus maps Stripe event types to order status transitions.
func stripeEventToStatus(eventType string) string {
	switch eventType {
	case "checkout.session.completed", "payment_intent.succeeded":
		return "completed"
	case "payment_intent.payment_failed", "checkout.session.async_payment_failed":
		return "failed"
	case "checkout.session.expired":
		return "expired"
	case "checkout.session.async_payment_succeeded":
		return "completed"
	}
	return ""
}

func (s *FiatRampService) handleStripeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}
	if err := verifyStripeSignature(body, c.GetHeader("Stripe-Signature"), s.getWebhookSecret("stripe")); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var event struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID            string `json:"id"`
				ClientReferenceID string `json:"client_reference_id"`
				Metadata      map[string]string `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload"})
		return
	}
	status := stripeEventToStatus(event.Type)
	if status == "" {
		c.JSON(http.StatusOK, gin.H{"received": true, "action": "ignored"})
		return
	}
	orderID := event.Data.Object.ClientReferenceID
	if orderID == "" {
		orderID = event.Data.Object.Metadata["order_id"]
	}
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no order reference in event"})
		return
	}
	if err := s.updateOrderStatus(c.Request.Context(), orderID, status); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"received": true, "orderId": orderID, "status": status})
}

// moonpayEventToStatus maps MoonPay transaction statuses to order statuses.
func moonpayEventToStatus(s string) string {
	switch strings.ToLower(s) {
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	case "pending", "waitingpayment", "waitingauthorization":
		return "processing"
	}
	return ""
}

func (s *FiatRampService) handleMoonPayWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}
	sig := c.GetHeader("Moonpay-Signature")
	if sig == "" {
		sig = c.GetHeader("Moonpay-Signature-V2")
	}
	if err := verifyHMACBody(body, sig, s.getWebhookSecret("moonpay"), "moonpay"); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var event struct {
		Type string `json:"type"`
		Data struct {
			ID               string `json:"id"`
			Status           string `json:"status"`
			ExternalCustomerID string `json:"externalCustomerId"`
			Metadata         map[string]string `json:"metadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload"})
		return
	}
	status := moonpayEventToStatus(event.Data.Status)
	if status == "" {
		c.JSON(http.StatusOK, gin.H{"received": true, "action": "ignored"})
		return
	}
	orderID := event.Data.Metadata["order_id"]
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no order reference in event"})
		return
	}
	if err := s.updateOrderStatus(c.Request.Context(), orderID, status); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"received": true, "orderId": orderID, "status": status})
}

// transakEventToStatus maps Transak order statuses to order statuses.
func transakEventToStatus(s string) string {
	switch strings.ToUpper(s) {
	case "COMPLETED":
		return "completed"
	case "FAILED", "CANCELLED", "EXPIRED":
		return "failed"
	case "PENDING_DELIVERY_FROM_TRANSAK", "AWAITING_PAYMENT_FROM_USER", "PROCESSING_FROM_TRANSAK":
		return "processing"
	}
	return ""
}

func (s *FiatRampService) handleTransakWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}
	if err := verifyHMACBody(body, c.GetHeader("x-transak-signature"), s.getWebhookSecret("transak"), "transak"); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var event struct {
		EventID string `json:"eventID"`
		Data    struct {
			ID           string `json:"id"`
			Status       string `json:"status"`
			PartnerOrderID string `json:"partnerOrderId"`
		} `json:"webhookData"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		// Transak may deliver the order object at top level on some plans.
		var alt struct {
			Status         string `json:"status"`
			PartnerOrderID string `json:"partnerOrderId"`
		}
		if err2 := json.Unmarshal(body, &alt); err2 != nil || alt.PartnerOrderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload"})
			return
		}
		event.Data.Status = alt.Status
		event.Data.PartnerOrderID = alt.PartnerOrderID
	}
	status := transakEventToStatus(event.Data.Status)
	if status == "" {
		c.JSON(http.StatusOK, gin.H{"received": true, "action": "ignored"})
		return
	}
	orderID := event.Data.PartnerOrderID
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no order reference in event"})
		return
	}
	if err := s.updateOrderStatus(c.Request.Context(), orderID, status); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"received": true, "orderId": orderID, "status": status})
}
