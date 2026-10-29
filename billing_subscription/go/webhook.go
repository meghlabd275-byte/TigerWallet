package main

// webhook.go — real Stripe webhook verification and event handling.
//
// Stripe signs every delivery with an HMAC-SHA256 over "<timestamp>.<body>"
// using the endpoint's signing secret. The signature is verified before the
// event is trusted, the timestamp is checked to bound replay, and each event id
// is recorded so a redelivery cannot be applied twice.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// maxWebhookBody bounds the payload Stripe may send.
const maxWebhookBody = 1 << 20 // 1 MiB

// webhookTolerance is how far a signature timestamp may be from now. Stripe
// recommends five minutes; it bounds how long a captured request stays usable.
const webhookTolerance = 5 * time.Minute

var (
	errNoWebhookSecret = errors.New("Stripe webhook secret is not configured (STRIPE_WEBHOOK_SECRET)")
	errBadSignature    = errors.New("signature does not match the payload")
	errStaleSignature  = errors.New("signature timestamp is outside the tolerance window")
)

// processedEvents records handled event ids so a redelivery is not applied
// twice. Stripe retries on any non-2xx response, and without this a retried
// payment event would be processed again.
type processedEvents struct {
	mu     sync.Mutex
	seen   map[string]time.Time
	maxAge time.Duration
}

func newProcessedEvents() *processedEvents {
	return &processedEvents{seen: map[string]time.Time{}, maxAge: 24 * time.Hour}
}

// markHandled records an event id and reports whether it had been seen before.
func (p *processedEvents) markHandled(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for key, at := range p.seen {
		if now.Sub(at) > p.maxAge {
			delete(p.seen, key)
		}
	}
	if _, ok := p.seen[id]; ok {
		return true
	}
	p.seen[id] = now
	return false
}

var stripeEvents = newProcessedEvents()

// verifyStripeSignature validates the Stripe-Signature header against the raw
// request body.
//
// The header has the form "t=<unix>,v1=<hex>[,v1=<hex>...]". Every v1 value is
// compared so a signature computed with a rotated secret is still accepted
// during a rotation window.
func verifyStripeSignature(payload []byte, header, secret string, now time.Time) error {
	if strings.TrimSpace(secret) == "" {
		return errNoWebhookSecret
	}
	if strings.TrimSpace(header) == "" {
		return errBadSignature
	}

	var timestamp string
	var candidates []string
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			candidates = append(candidates, value)
		}
	}
	if timestamp == "" || len(candidates) == 0 {
		return errBadSignature
	}

	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errBadSignature
	}
	drift := now.Sub(time.Unix(seconds, 0))
	if drift < 0 {
		drift = -drift
	}
	if drift > webhookTolerance {
		return errStaleSignature
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := mac.Sum(nil)

	for _, candidate := range candidates {
		decoded, err := hex.DecodeString(candidate)
		if err != nil {
			continue
		}
		if hmac.Equal(decoded, expected) {
			return nil
		}
	}
	return errBadSignature
}

// stripeEvent is the subset of a Stripe event the billing service acts on.
type stripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

// invoiceObject is the subset of a Stripe invoice this service reads.
type invoiceObject struct {
	ID            string `json:"id"`
	Subscription  string `json:"subscription"`
	Customer      string `json:"customer"`
	AmountPaid    int64  `json:"amount_paid"`
	AmountDue     int64  `json:"amount_due"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	PaymentIntent string `json:"payment_intent"`
	BillingReason string `json:"billing_reason"`
}

// subscriptionObject is the subset of a Stripe subscription this service reads.
type subscriptionObject struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
	Status   string `json:"status"`
	Items    struct {
		Data []struct {
			Price struct {
				ID string `json:"id"`
			} `json:"price"`
		} `json:"data"`
	} `json:"items"`
}

// handleStripeWebhook verifies and processes a Stripe event.
//
// The endpoint is fail-closed: without a configured signing secret it refuses
// every request, so an unverified caller can never drive billing state.
func handleStripeWebhook(c *gin.Context) {
	secret := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))

	// The body must be read before verification: the signature covers the exact
	// bytes sent, so it cannot be checked after JSON decoding.
	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBody))
	if err != nil || len(payload) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty or unreadable body"})
		return
	}

	if err := verifyStripeSignature(payload, c.GetHeader("Stripe-Signature"), secret, time.Now()); err != nil {
		switch {
		case errors.Is(err, errNoWebhookSecret):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook endpoint is not configured"})
		case errors.Is(err, errStaleSignature):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "signature timestamp outside tolerance"})
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		}
		return
	}

	var event stripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload"})
		return
	}
	if event.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event id is required"})
		return
	}

	// Stripe redelivers on any non-2xx response, so a handled event is
	// acknowledged without being applied a second time.
	if stripeEvents.markHandled(event.ID) {
		c.JSON(http.StatusOK, gin.H{"received": true, "duplicate": true, "event_id": event.ID})
		return
	}

	switch event.Type {
	case "invoice.paid", "invoice.payment_succeeded":
		var inv invoiceObject
		if err := json.Unmarshal(event.Data.Object, &inv); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice object"})
			return
		}
		log.Printf("billing: invoice %s paid (%d %s, subscription %s)", inv.ID, inv.AmountPaid, inv.Currency, inv.Subscription)

	case "invoice.payment_failed":
		var inv invoiceObject
		if err := json.Unmarshal(event.Data.Object, &inv); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice object"})
			return
		}
		log.Printf("billing: invoice %s payment failed (%d %s due, subscription %s)", inv.ID, inv.AmountDue, inv.Currency, inv.Subscription)

	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		var sub subscriptionObject
		if err := json.Unmarshal(event.Data.Object, &sub); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription object"})
			return
		}
		log.Printf("billing: subscription %s is %s", sub.ID, sub.Status)

	default:
		// Unhandled types are acknowledged so Stripe stops retrying them, but
		// they are not treated as having changed any state.
		log.Printf("billing: ignoring unhandled event type %q (%s)", event.Type, event.ID)
	}

	c.JSON(http.StatusOK, gin.H{"received": true, "event_id": event.ID, "type": event.Type})
}

// stripeSignatureErrorText renders the verification failure for logs without
// exposing the secret.
func stripeSignatureErrorText(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
