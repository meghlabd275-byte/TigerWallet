package main

import (
	"context"
	"net/http"
	"testing"
)

// Two listings whose ids share a first character must not be confused. The
// previous lookup took the eight characters after "LISTING-" from the order id
// and matched them as a prefix against every listing, so a callback for one
// listing could resolve to the other and mark the wrong application paid.
func TestOrderResolvesToItsOwnListingDespiteSharedPrefix(t *testing.T) {
	const secret = "chain-listener-secret"
	t.Setenv("PAYMENT_WEBHOOK_SECRET", secret)

	svc := newTestStore(t)

	// Same first eight characters, different listings.
	first := sampleListing("abcdef01-0000-0000-0000-000000000001")
	second := sampleListing("abcdef01-0000-0000-0000-000000000002")
	for _, l := range []*TokenListing{first, second} {
		l.PaymentStatus = "pending"
		l.PaymentID = "pay-" + l.ID
		if err := svc.storeListing(context.Background(), l); err != nil {
			t.Fatalf("storeListing: %v", err)
		}
	}

	// Only the second listing has a payment order.
	if err := svc.indexOrder(context.Background(), "LISTING-abcdef01-0000-0000-0000-000000000002", second.ID); err != nil {
		t.Fatalf("indexOrder: %v", err)
	}

	body := completedPaymentBody(
		"LISTING-abcdef01-0000-0000-0000-000000000002",
		second.PaymentID,
		"0xfeedface",
	)
	code, resp := postSignedWebhook(t, svc, secret, body)
	if code != http.StatusOK {
		t.Fatalf("completion for the indexed listing returned %d (%s), want 200", code, resp)
	}

	paid, _ := svc.loadListing(context.Background(), second.ID)
	if paid.Status != "paid" {
		t.Errorf("the listing that owns the order is %q, want paid", paid.Status)
	}

	// The other listing shares the same eight-character prefix and must be
	// untouched.
	untouched, _ := svc.loadListing(context.Background(), first.ID)
	if untouched.Status == "paid" {
		t.Error("a listing that does not own the order was marked paid")
	}
	if untouched.PaymentTxHash != "" {
		t.Errorf("the unrelated listing recorded tx %q", untouched.PaymentTxHash)
	}
}

// An order id that was never issued must not resolve to anything.
func TestUnknownOrderIsNotResolved(t *testing.T) {
	const secret = "chain-listener-secret"
	t.Setenv("PAYMENT_WEBHOOK_SECRET", secret)

	svc := newTestStore(t)
	listing := sampleListing("deadbeef-0000-0000-0000-000000000001")
	listing.PaymentStatus = "pending"
	if err := svc.storeListing(context.Background(), listing); err != nil {
		t.Fatalf("storeListing: %v", err)
	}

	// No indexOrder call: this order was never handed out.
	body := completedPaymentBody("LISTING-deadbeef", "any-payment", "0xabc")
	code, _ := postSignedWebhook(t, svc, secret, body)
	if code != http.StatusNotFound {
		t.Fatalf("an unknown order returned %d, want %d", code, http.StatusNotFound)
	}

	got, _ := svc.loadListing(context.Background(), listing.ID)
	if got.Status == "paid" {
		t.Error("a listing was marked paid by an order that was never issued")
	}
}
