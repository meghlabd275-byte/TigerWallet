package main

// Tests for Stripe webhook signature verification. The signing scheme is
// exercised with a locally computed signature so the verification path is
// proven without contacting Stripe.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// signStripe builds a Stripe-Signature header for a payload.
func signStripe(payload []byte, secret string, at time.Time) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d", at.Unix())))
	mac.Write([]byte("."))
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", at.Unix(), hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifyStripeSignatureAcceptsValidSignature(t *testing.T) {
	payload := []byte(`{"id":"evt_1","type":"invoice.paid"}`)
	now := time.Now()
	header := signStripe(payload, "whsec_test", now)

	if err := verifyStripeSignature(payload, header, "whsec_test", now); err != nil {
		t.Fatalf("a valid signature was rejected: %v", err)
	}
}

func TestVerifyStripeSignatureRejectsTamperedPayload(t *testing.T) {
	payload := []byte(`{"id":"evt_1","type":"invoice.paid"}`)
	now := time.Now()
	header := signStripe(payload, "whsec_test", now)

	tampered := []byte(`{"id":"evt_1","type":"invoice.paid","extra":true}`)
	if err := verifyStripeSignature(tampered, header, "whsec_test", now); err == nil {
		t.Fatal("a tampered payload was accepted")
	}
}

func TestVerifyStripeSignatureRejectsWrongSecret(t *testing.T) {
	payload := []byte(`{"id":"evt_1"}`)
	now := time.Now()
	header := signStripe(payload, "whsec_attacker", now)

	if err := verifyStripeSignature(payload, header, "whsec_real", now); !errors.Is(err, errBadSignature) {
		t.Fatalf("error = %v, want errBadSignature", err)
	}
}

func TestVerifyStripeSignatureRejectsStaleTimestamp(t *testing.T) {
	payload := []byte(`{"id":"evt_1"}`)
	signedAt := time.Now().Add(-1 * time.Hour)
	header := signStripe(payload, "whsec_test", signedAt)

	err := verifyStripeSignature(payload, header, "whsec_test", time.Now())
	if !errors.Is(err, errStaleSignature) {
		t.Fatalf("error = %v, want errStaleSignature", err)
	}
}

func TestVerifyStripeSignatureAcceptsFutureWithinTolerance(t *testing.T) {
	// A small clock skew between Stripe and this host must not reject a valid
	// delivery.
	payload := []byte(`{"id":"evt_1"}`)
	signedAt := time.Now().Add(30 * time.Second)
	header := signStripe(payload, "whsec_test", signedAt)

	if err := verifyStripeSignature(payload, header, "whsec_test", time.Now()); err != nil {
		t.Fatalf("a signature 30s in the future was rejected: %v", err)
	}
}

func TestVerifyStripeSignatureRequiresSecret(t *testing.T) {
	payload := []byte(`{"id":"evt_1"}`)
	now := time.Now()
	header := signStripe(payload, "whsec_test", now)

	if err := verifyStripeSignature(payload, header, "", now); !errors.Is(err, errNoWebhookSecret) {
		t.Fatalf("error = %v, want errNoWebhookSecret", err)
	}
}

func TestVerifyStripeSignatureRejectsMalformedHeaders(t *testing.T) {
	payload := []byte(`{"id":"evt_1"}`)
	now := time.Now()
	valid := signStripe(payload, "whsec_test", now)
	_, hexPart, _ := strings.Cut(valid, ",v1=")

	for _, header := range []string{
		"",
		"garbage",
		"t=,v1=" + hexPart,
		"t=notanumber,v1=" + hexPart,
		"v1=" + hexPart,               // no timestamp
		"t=" + fmt.Sprint(now.Unix()), // no signature
		"t=" + fmt.Sprint(now.Unix()) + ",v1=nothex",
	} {
		if err := verifyStripeSignature(payload, header, "whsec_test", now); err == nil {
			t.Errorf("malformed header %q was accepted", header)
		}
	}
}

func TestVerifyStripeSignatureAcceptsRotatedSecret(t *testing.T) {
	// During a secret rotation Stripe sends signatures from both secrets. Any
	// matching v1 must be accepted.
	payload := []byte(`{"id":"evt_1"}`)
	now := time.Now()

	oldSig := signStripe(payload, "whsec_old", now)
	newSig := signStripe(payload, "whsec_new", now)
	_, oldV1, _ := strings.Cut(oldSig, ",v1=")
	_, newV1, _ := strings.Cut(newSig, ",v1=")

	header := fmt.Sprintf("t=%d,v1=%s,v1=%s", now.Unix(), oldV1, newV1)
	if err := verifyStripeSignature(payload, header, "whsec_new", now); err != nil {
		t.Fatalf("a signature from the new secret was rejected during rotation: %v", err)
	}
}

func TestProcessedEventsDetectsDuplicates(t *testing.T) {
	p := newProcessedEvents()

	if p.markHandled("evt_1") {
		t.Error("the first delivery was reported as a duplicate")
	}
	if !p.markHandled("evt_1") {
		t.Error("a redelivery was not reported as a duplicate")
	}
	if p.markHandled("evt_2") {
		t.Error("a distinct event was reported as a duplicate")
	}
}

func TestHandleStripeWebhookFailsClosedWithoutSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")

	payload := []byte(`{"id":"evt_1","type":"invoice.paid"}`)
	now := time.Now()
	// Sign with a secret the server does not know.
	header := signStripe(payload, "whsec_attacker", now)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	c.Request.Header.Set("Stripe-Signature", header)

	handleStripeWebhook(c)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 when the endpoint is unconfigured", rec.Code)
	}
}

func TestHandleStripeWebhookRejectsInvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_real")

	payload := []byte(`{"id":"evt_1","type":"invoice.paid"}`)
	header := signStripe(payload, "whsec_attacker", time.Now())

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	c.Request.Header.Set("Stripe-Signature", header)

	handleStripeWebhook(c)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for an invalid signature", rec.Code)
	}
}

func TestHandleStripeWebhookAcceptsValidEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_real")

	payload := []byte(`{"id":"evt_paid_1","type":"invoice.paid","data":{"object":{"id":"in_1","subscription":"sub_1","amount_paid":1999,"currency":"usd","status":"paid"}}}`)
	now := time.Now()
	header := signStripe(payload, "whsec_real", now)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	c.Request.Header.Set("Stripe-Signature", header)

	handleStripeWebhook(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "evt_paid_1") {
		t.Errorf("response does not identify the event: %s", rec.Body.String())
	}
}

func TestHandleStripeWebhookIsIdempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_real")

	payload := []byte(`{"id":"evt_dup_test","type":"invoice.paid","data":{"object":{"id":"in_2","amount_paid":500,"currency":"usd"}}}`)
	now := time.Now()
	header := signStripe(payload, "whsec_real", now)

	deliver := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
		c.Request.Header.Set("Stripe-Signature", header)
		handleStripeWebhook(c)
		return rec
	}

	if rec := deliver(); rec.Code != http.StatusOK {
		t.Fatalf("first delivery status = %d", rec.Code)
	}
	rec := deliver()
	if rec.Code != http.StatusOK {
		t.Fatalf("redelivery status = %d, want 200 so Stripe stops retrying", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"duplicate":true`) {
		t.Errorf("redelivery was not reported as a duplicate: %s", rec.Body.String())
	}
}

func TestHandleStripeWebhookRejectsEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_real")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(nil))
	c.Request.Header.Set("Stripe-Signature", signStripe(nil, "whsec_real", time.Now()))

	handleStripeWebhook(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for an empty body", rec.Code)
	}
}

func TestHandleStripeWebhookRejectsMissingEventID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_real")

	payload := []byte(`{"type":"invoice.paid"}`)
	header := signStripe(payload, "whsec_real", time.Now())

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	c.Request.Header.Set("Stripe-Signature", header)

	handleStripeWebhook(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 when the event has no id", rec.Code)
	}
}

func TestHandleStripeWebhookAcknowledgesUnhandledTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_real")

	payload := []byte(`{"id":"evt_unknown_1","type":"charge.refunded","data":{"object":{}}}`)
	header := signStripe(payload, "whsec_real", time.Now())

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	c.Request.Header.Set("Stripe-Signature", header)

	handleStripeWebhook(c)

	// Unhandled types must be acknowledged so Stripe stops retrying, but the
	// response must not claim any state changed.
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "processed") {
		t.Errorf("an unhandled event was reported as processed: %s", rec.Body.String())
	}
}
