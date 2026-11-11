package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// newTestStore returns a service pointed at the local Redis and a cleanup that
// removes the keys the test created. The test is skipped when Redis is not
// reachable so the suite stays runnable on a machine without one.
func newTestStore(t *testing.T) *ListingService {
	t.Helper()

	addr := "127.0.0.1:6379"
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis is not reachable at %s: %v", addr, err)
	}
	_ = client.Close()

	svc := NewListingService()
	svc.redis = redis.NewClient(&redis.Options{Addr: addr})

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		keys, _ := svc.redis.Keys(ctx, "listing:*").Result()
		if len(keys) > 0 {
			_ = svc.redis.Del(ctx, keys...).Err()
		}
		_ = svc.redis.Close()
	})

	return svc
}

func sampleListing(id string) *TokenListing {
	now := time.Now().UTC().Truncate(time.Second)
	return &TokenListing{
		ID:             id,
		ApplicantEmail: "applicant@example.com",
		TokenSymbol:    "TGR",
		TokenName:      "Tiger",
		ContractAddr:   "0x0000000000000000000000000000000000000001",
		ChainID:        1,
		Chain:          "ethereum",
		Tier:           "standard",
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// A listing written by one replica must be readable by a process that never
// held it in memory. This is the restart case: without a durable store the
// applicant's record is gone and a paid application cannot be reviewed.
func TestListingSurvivesProcessRestart(t *testing.T) {
	svc := newTestStore(t)
	ctx := context.Background()

	listing := sampleListing("restart-case")
	if err := svc.storeListing(ctx, listing); err != nil {
		t.Fatalf("storeListing: %v", err)
	}

	// A fresh service instance, as a new process would be: empty cache.
	fresh := NewListingService()
	fresh.redis = svc.redis

	got, ok := fresh.loadListing(ctx, listing.ID)
	if !ok {
		t.Fatal("the listing was not found after a restart")
	}
	if got.TokenSymbol != listing.TokenSymbol {
		t.Errorf("token symbol = %q, want %q", got.TokenSymbol, listing.TokenSymbol)
	}
	if got.Status != "pending" {
		t.Errorf("status = %q, want %q", got.Status, "pending")
	}
	if !got.CreatedAt.Equal(listing.CreatedAt) {
		t.Errorf("created_at = %v, want %v", got.CreatedAt, listing.CreatedAt)
	}
}

// A status change made on one replica must be visible to the others, otherwise
// two admins looking at the same application see different states.
func TestStatusChangeIsVisibleToOtherReplicas(t *testing.T) {
	svc := newTestStore(t)
	ctx := context.Background()

	listing := sampleListing("shared-state")
	if err := svc.storeListing(ctx, listing); err != nil {
		t.Fatalf("storeListing: %v", err)
	}

	updated := *listing
	updated.Status = "approved"
	if err := svc.storeListing(ctx, &updated); err != nil {
		t.Fatalf("storeListing(update): %v", err)
	}

	other := NewListingService()
	other.redis = svc.redis

	got, ok := other.loadListing(ctx, listing.ID)
	if !ok {
		t.Fatal("the listing was not found on the other replica")
	}
	if got.Status != "approved" {
		t.Errorf("status on the other replica = %q, want %q", got.Status, "approved")
	}
}

// The list view must reflect the durable store, not just the local cache, so a
// replica that has handled no writes still returns the full set.
func TestListReadsFromTheSharedStore(t *testing.T) {
	svc := newTestStore(t)
	ctx := context.Background()

	for _, id := range []string{"list-a", "list-b", "list-c"} {
		l := sampleListing(id)
		if id == "list-b" {
			l.Status = "approved"
		}
		if err := svc.storeListing(ctx, l); err != nil {
			t.Fatalf("storeListing(%s): %v", id, err)
		}
	}

	other := NewListingService()
	other.redis = svc.redis

	all, err := other.listListings(ctx, "", "")
	if err != nil {
		t.Fatalf("listListings: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("listed %d listings, want 3", len(all))
	}

	approved, err := other.listListings(ctx, "approved", "")
	if err != nil {
		t.Fatalf("listListings(approved): %v", err)
	}
	if len(approved) != 1 {
		t.Fatalf("listed %d approved listings, want 1", len(approved))
	}
	if approved[0].ID != "list-b" {
		t.Errorf("approved listing = %q, want %q", approved[0].ID, "list-b")
	}
}

// A deleted listing must stay deleted for every replica.
func TestDeleteRemovesTheListingForEveryone(t *testing.T) {
	svc := newTestStore(t)
	ctx := context.Background()

	listing := sampleListing("delete-case")
	if err := svc.storeListing(ctx, listing); err != nil {
		t.Fatalf("storeListing: %v", err)
	}
	if err := svc.deleteListing(ctx, listing.ID); err != nil {
		t.Fatalf("deleteListing: %v", err)
	}

	other := NewListingService()
	other.redis = svc.redis

	if _, ok := other.loadListing(ctx, listing.ID); ok {
		t.Error("the listing is still readable after being deleted")
	}
}

// loadListing must hand back a copy. If it returned the cached pointer, an
// in-flight update would be visible to readers before it was persisted, and the
// map would be written without holding the lock.
func TestLoadReturnsACopy(t *testing.T) {
	svc := newTestStore(t)
	ctx := context.Background()

	listing := sampleListing("copy-case")
	if err := svc.storeListing(ctx, listing); err != nil {
		t.Fatalf("storeListing: %v", err)
	}

	first, ok := svc.loadListing(ctx, listing.ID)
	if !ok {
		t.Fatal("the listing was not found")
	}
	first.Status = "mutated-in-place"

	second, ok := svc.loadListing(ctx, listing.ID)
	if !ok {
		t.Fatal("the listing was not found on the second read")
	}
	if second.Status != "pending" {
		t.Errorf("a caller mutated the stored listing: status = %q, want %q", second.Status, "pending")
	}
}

// signedWebhookRequest builds a request whose body carries a valid signature for
// the given secret.
func signedWebhookRequest(t *testing.T, secret string, body []byte) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/listing/payment/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-Payment-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	return req
}

func webhookBody(orderID string) []byte {
	return []byte(`{"event":"payment.completed","order_id":"` + orderID + `","status":"completed","tx_hash":"0xabc"}`)
}

// runWebhook posts the body to the handler and returns the recorded status.
func runWebhook(t *testing.T, secret string, body []byte) int {
	t.Helper()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = signedWebhookRequest(t, secret, body)

	svc := NewListingService()
	svc.HandlePaymentWebhook(c)
	return rec.Code
}

// An unsigned caller must not be able to mark a listing paid. Before the
// signature check existed this returned 200 and flipped the listing, letting an
// applicant skip the listing fee.
func TestWebhookRejectsUnsignedCaller(t *testing.T) {
	t.Setenv("PAYMENT_WEBHOOK_SECRET", "a-real-secret")

	body := webhookBody("LISTING-deadbeef")
	if code := runWebhook(t, "", body); code != http.StatusUnauthorized {
		t.Errorf("unsigned webhook returned %d, want %d", code, http.StatusUnauthorized)
	}
}

// A body signed with the wrong secret must be rejected.
func TestWebhookRejectsForgedSignature(t *testing.T) {
	t.Setenv("PAYMENT_WEBHOOK_SECRET", "a-real-secret")

	body := webhookBody("LISTING-deadbeef")
	if code := runWebhook(t, "the-wrong-secret", body); code != http.StatusUnauthorized {
		t.Errorf("forged webhook returned %d, want %d", code, http.StatusUnauthorized)
	}
}

// With no secret configured the endpoint must refuse every caller rather than
// fall open.
func TestWebhookFailsClosedWithoutSecret(t *testing.T) {
	t.Setenv("PAYMENT_WEBHOOK_SECRET", "")

	body := webhookBody("LISTING-deadbeef")
	if code := runWebhook(t, "", body); code != http.StatusServiceUnavailable {
		t.Errorf("webhook without a configured secret returned %d, want %d", code, http.StatusServiceUnavailable)
	}
}

// A correctly signed body is accepted, so the check rejects forgery without
// breaking the real payment service.
func TestWebhookAcceptsCorrectlySignedBody(t *testing.T) {
	const secret = "a-real-secret"
	t.Setenv("PAYMENT_WEBHOOK_SECRET", secret)

	body := webhookBody("LISTING-deadbeef")
	code := runWebhook(t, secret, body)

	// The listing does not exist, so the handler reports it as missing. That is
	// past the signature check, which is what this test is about.
	if code == http.StatusUnauthorized || code == http.StatusServiceUnavailable {
		t.Errorf("a correctly signed webhook was rejected with %d", code)
	}
}

// postSignedWebhook runs the handler against a service backed by the real store.
func postSignedWebhook(t *testing.T, svc *ListingService, secret string, body []byte) (int, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = signedWebhookRequest(t, secret, body)
	svc.HandlePaymentWebhook(c)
	return rec.Code, rec.Body.String()
}

func completedPaymentBody(orderID, paymentID, txHash string) []byte {
	return []byte(`{"event":"payment.completed","order_id":"` + orderID +
		`","payment_id":"` + paymentID + `","status":"completed","tx_hash":"` + txHash + `"}`)
}

// A signed completion callback must actually mark the listing paid. The order id
// carries the first eight characters of the listing id; the previous
// implementation matched on a "pay_"-prefixed payment id that the payment
// service never issues, so a paid listing stayed pending forever.
func TestCompletedPaymentMarksTheListingPaid(t *testing.T) {
	const secret = "chain-listener-secret"
	t.Setenv("PAYMENT_WEBHOOK_SECRET", secret)

	svc := newTestStore(t)
	listing := sampleListing("abcdef01-2345-6789-abcd-ef0123456789")
	listing.PaymentID = "2f1c9b7e-0000-4000-8000-000000000001"
	listing.PaymentStatus = "pending"
	if err := svc.storeListing(context.Background(), listing); err != nil {
		t.Fatalf("storeListing: %v", err)
	}

	const txHash = "0xabc123"
	if err := svc.indexOrder(context.Background(), "LISTING-abcdef01", listing.ID); err != nil {
		t.Fatalf("indexOrder: %v", err)
	}
	body := completedPaymentBody("LISTING-abcdef01", listing.PaymentID, txHash)
	code, resp := postSignedWebhook(t, svc, secret, body)
	if code != http.StatusOK {
		t.Fatalf("signed completion returned %d (%s), want 200", code, resp)
	}

	got, ok := svc.loadListing(context.Background(), listing.ID)
	if !ok {
		t.Fatal("the listing disappeared")
	}
	if got.Status != "paid" {
		t.Errorf("listing status = %q, want %q", got.Status, "paid")
	}
	if got.PaymentStatus != "paid" {
		t.Errorf("payment status = %q, want %q", got.PaymentStatus, "paid")
	}
	if got.PaymentTxHash != txHash {
		t.Errorf("payment tx hash = %q, want %q", got.PaymentTxHash, txHash)
	}
}

// The applicant must be handed the address the payment service issued, because
// that is the address the chain listener watches. Returning the admin wallet
// instead means the payment is never matched and the listing never gets paid.
func TestApplicationReturnsTheIssuedDepositAddress(t *testing.T) {
	const issued = "0x9999999999999999999999999999999999999999"

	var gotPath string
	paySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"payment":{"id":"pay-1","payment_address":"` + issued + `","expires_at":"2026-01-01T00:00:00Z"}}`))
	}))
	defer paySrv.Close()

	svc := newTestStore(t)
	svc.paymentURL = paySrv.URL

	intent, err := svc.createPaymentIntent(map[string]interface{}{"order_id": "LISTING-abcdef01"})
	if err != nil {
		t.Fatalf("createPaymentIntent: %v", err)
	}
	if gotPath != "/api/v1/payment/deposit/create" {
		t.Errorf("payment service path = %q, want %q", gotPath, "/api/v1/payment/deposit/create")
	}
	if intent.PaymentID != "pay-1" {
		t.Errorf("payment id = %q, want %q", intent.PaymentID, "pay-1")
	}
	if intent.PaymentAddress != issued {
		t.Fatalf("payment address = %q, want %q", intent.PaymentAddress, issued)
	}

	listing := &TokenListing{ID: "abcdef01-2345", PaymentAddress: issued}
	if got := svc.getPaymentAddress(listing); got != issued {
		t.Errorf("the applicant was shown %q, want the issued address %q", got, issued)
	}
}

// A deposit response without an address is a failure: the applicant would have
// no way to pay, so the listing must not be shown a payment step.
func TestPaymentIntentWithoutAddressIsAnError(t *testing.T) {
	paySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"payment":{"id":"pay-1"}}`))
	}))
	defer paySrv.Close()

	svc := newTestStore(t)
	svc.paymentURL = paySrv.URL

	if _, err := svc.createPaymentIntent(map[string]interface{}{}); err == nil {
		t.Fatal("a deposit response without an address was accepted")
	}
}

// A deposit response carrying a non-2xx status must surface as an error rather
// than being parsed as success.
func TestPaymentIntentSurfacesUpstreamErrors(t *testing.T) {
	paySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"hot wallet unavailable"}`))
	}))
	defer paySrv.Close()

	svc := newTestStore(t)
	svc.paymentURL = paySrv.URL

	_, err := svc.createPaymentIntent(map[string]interface{}{})
	if err == nil {
		t.Fatal("a failing payment service was treated as success")
	}
	if !strings.Contains(err.Error(), "hot wallet unavailable") {
		t.Errorf("the upstream reason was dropped: %v", err)
	}
}

// A completion for a payment that belongs to another listing must be refused.
func TestCompletionForAnotherListingsPaymentIsRefused(t *testing.T) {
	const secret = "chain-listener-secret"
	t.Setenv("PAYMENT_WEBHOOK_SECRET", secret)

	svc := newTestStore(t)
	listing := sampleListing("abcdef02-2345-6789-abcd-ef0123456789")
	listing.PaymentID = "2f1c9b7e-0000-4000-8000-000000000002"
	listing.PaymentStatus = "pending"
	if err := svc.storeListing(context.Background(), listing); err != nil {
		t.Fatalf("storeListing: %v", err)
	}

	if err := svc.indexOrder(context.Background(), "LISTING-abcdef02", listing.ID); err != nil {
		t.Fatalf("indexOrder: %v", err)
	}
	body := completedPaymentBody("LISTING-abcdef02", "some-other-payment", "0xabc123")
	code, _ := postSignedWebhook(t, svc, secret, body)
	if code != http.StatusConflict {
		t.Fatalf("a foreign payment returned %d, want %d", code, http.StatusConflict)
	}

	got, _ := svc.loadListing(context.Background(), listing.ID)
	if got.Status == "paid" {
		t.Error("the listing was marked paid by a payment that is not its own")
	}
}

// A completion without a transaction hash cannot be reconciled, so it must be
// refused and must not move the listing.
func TestCompletionWithoutTxHashIsRefused(t *testing.T) {
	const secret = "chain-listener-secret"
	t.Setenv("PAYMENT_WEBHOOK_SECRET", secret)

	svc := newTestStore(t)
	listing := sampleListing("abcdef03-2345-6789-abcd-ef0123456789")
	listing.PaymentID = "2f1c9b7e-0000-4000-8000-000000000003"
	listing.PaymentStatus = "pending"
	if err := svc.storeListing(context.Background(), listing); err != nil {
		t.Fatalf("storeListing: %v", err)
	}

	if err := svc.indexOrder(context.Background(), "LISTING-abcdef03", listing.ID); err != nil {
		t.Fatalf("indexOrder: %v", err)
	}
	body := completedPaymentBody("LISTING-abcdef03", listing.PaymentID, "")
	code, _ := postSignedWebhook(t, svc, secret, body)
	if code != http.StatusBadRequest {
		t.Fatalf("a completion without a tx hash returned %d, want %d", code, http.StatusBadRequest)
	}

	got, _ := svc.loadListing(context.Background(), listing.ID)
	if got.Status == "paid" {
		t.Error("the listing was marked paid without a transaction hash")
	}
}

// A non-completion event must leave the listing untouched.
func TestNonCompletionEventLeavesTheListingAlone(t *testing.T) {
	const secret = "chain-listener-secret"
	t.Setenv("PAYMENT_WEBHOOK_SECRET", secret)

	svc := newTestStore(t)
	listing := sampleListing("abcdef04-2345-6789-abcd-ef0123456789")
	listing.PaymentID = "2f1c9b7e-0000-4000-8000-000000000004"
	listing.PaymentStatus = "pending"
	if err := svc.storeListing(context.Background(), listing); err != nil {
		t.Fatalf("storeListing: %v", err)
	}

	if err := svc.indexOrder(context.Background(), "LISTING-abcdef04", listing.ID); err != nil {
		t.Fatalf("indexOrder: %v", err)
	}
	body := []byte(`{"event":"payment.pending","order_id":"LISTING-abcdef04","status":"pending"}`)
	if code, _ := postSignedWebhook(t, svc, secret, body); code != http.StatusOK {
		t.Fatalf("a pending event returned %d, want 200", code)
	}

	got, _ := svc.loadListing(context.Background(), listing.ID)
	if got.Status != "pending" {
		t.Errorf("a pending event moved the listing to %q", got.Status)
	}
}

// A redelivered completion must not rewrite the record.
func TestRedeliveredCompletionIsIdempotent(t *testing.T) {
	const secret = "chain-listener-secret"
	t.Setenv("PAYMENT_WEBHOOK_SECRET", secret)

	svc := newTestStore(t)
	listing := sampleListing("abcdef05-2345-6789-abcd-ef0123456789")
	listing.PaymentID = "2f1c9b7e-0000-4000-8000-000000000005"
	listing.PaymentStatus = "pending"
	if err := svc.storeListing(context.Background(), listing); err != nil {
		t.Fatalf("storeListing: %v", err)
	}

	if err := svc.indexOrder(context.Background(), "LISTING-abcdef05", listing.ID); err != nil {
		t.Fatalf("indexOrder: %v", err)
	}
	body := completedPaymentBody("LISTING-abcdef05", listing.PaymentID, "0xdeadbeef")
	if code, _ := postSignedWebhook(t, svc, secret, body); code != http.StatusOK {
		t.Fatalf("first delivery returned %d, want 200", code)
	}
	first, _ := svc.loadListing(context.Background(), listing.ID)

	if code, _ := postSignedWebhook(t, svc, secret, body); code != http.StatusOK {
		t.Fatalf("second delivery returned %d, want 200", code)
	}
	second, _ := svc.loadListing(context.Background(), listing.ID)

	if !second.UpdatedAt.Equal(first.UpdatedAt) {
		t.Errorf("a redelivered completion rewrote the record: %v -> %v", first.UpdatedAt, second.UpdatedAt)
	}
	if second.PaymentTxHash != "0xdeadbeef" {
		t.Errorf("payment tx hash = %q, want %q", second.PaymentTxHash, "0xdeadbeef")
	}
}
