package main

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// newTestRedis connects to a Redis instance for the durability tests and skips
// when none is reachable, so the suite still runs in an environment without
// one. It flushes only a dedicated database.
func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	addr := "127.0.0.1:6379"
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 15})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("no Redis at %s, skipping durability test: %v", addr, err)
	}
	if err := rdb.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("cannot flush the test database: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		rdb.FlushDB(ctx)
		rdb.Close()
	})
	return rdb
}

// A deposit address must survive a restart. The chain listener attributes an
// incoming transfer by looking the address up, so an address held only in the
// process that issued it is money that can never be matched once that process
// is recycled. This is the exact defect: the address lived in a plain map.
func TestDepositAddressSurvivesRestart(t *testing.T) {
	rdb := newTestRedis(t)

	issuing := newPaymentStore(rdb)
	addr := &PaymentAddress{
		ID:        "pa1",
		Address:   "0x1111111111111111111111111111111111111111",
		Chain:     "ethereum",
		Token:     "USDT",
		OrderID:   "order-1",
		IsActive:  true,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	payment := &Payment{
		ID:        "p1",
		OrderID:   "order-1",
		ToAddress: addr.Address,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := issuing.putPaymentWithAddress(payment, addr); err != nil {
		t.Fatalf("cannot record the deposit: %v", err)
	}

	// A second replica, or the same process after a restart: a fresh store with
	// no local cache must still resolve the address.
	restarted := newPaymentStore(rdb)
	got, ok := restarted.getAddress(addr.Address)
	if !ok {
		t.Fatal("the deposit address was lost across a restart")
	}
	if got.OrderID != "order-1" {
		t.Errorf("address resolved to order %q, want order-1", got.OrderID)
	}
	if got.Chain != "ethereum" {
		t.Errorf("address resolved to chain %q, want ethereum", got.Chain)
	}

	// The payment must be findable too, so the listener can complete it.
	p, ok := restarted.getPayment("p1")
	if !ok {
		t.Fatal("the payment was lost across a restart")
	}
	if p.ToAddress != addr.Address {
		t.Errorf("payment points at %q, want %q", p.ToAddress, addr.Address)
	}
}

// A payment and its address are recorded together or not at all. A half-written
// pair would leave an address nobody watches, or a payment nobody can match.
func TestPaymentAndAddressAreRecordedTogether(t *testing.T) {
	rdb := newTestRedis(t)

	store := newPaymentStore(rdb)
	addr := &PaymentAddress{Address: "0x2222222222222222222222222222222222222222", OrderID: "o2"}
	payment := &Payment{ID: "p2", OrderID: "o2", ToAddress: addr.Address}

	if err := store.putPaymentWithAddress(payment, addr); err != nil {
		t.Fatalf("cannot record the pair: %v", err)
	}

	fresh := newPaymentStore(rdb)
	if _, ok := fresh.getAddress(addr.Address); !ok {
		t.Error("the address half of the pair is missing")
	}
	if _, ok := fresh.getPayment("p2"); !ok {
		t.Error("the payment half of the pair is missing")
	}
}

// An operator's fee edit must survive a restart, and the defaults must not
// overwrite it when the service comes back up.
func TestFeeEditSurvivesRestartAndIsNotReseeded(t *testing.T) {
	rdb := newTestRedis(t)

	store := newPaymentStore(rdb)
	if err := store.putFee(&FeeConfig{ID: "listing_tier1", FeeType: "listing", Amount: "5000", IsActive: true}); err != nil {
		t.Fatalf("cannot seed the fee: %v", err)
	}

	edited := &FeeConfig{ID: "listing_tier1", FeeType: "listing", Amount: "1234", IsActive: true, UpdatedBy: "superadmin"}
	if err := store.putFee(edited); err != nil {
		t.Fatalf("cannot store the edit: %v", err)
	}

	restarted := newPaymentStore(rdb)
	got, ok := restarted.getFee("listing_tier1")
	if !ok {
		t.Fatal("the fee was lost across a restart")
	}
	if got.Amount != "1234" {
		t.Errorf("fee amount is %q, want the edited 1234", got.Amount)
	}
	if got.UpdatedBy != "superadmin" {
		t.Errorf("fee editor is %q, want superadmin", got.UpdatedBy)
	}
}

// Writing through an unreachable store must fail, not silently succeed. The
// caller has to learn that the deposit address was not saved.
func TestWriteFailsWhenStoreIsUnreachable(t *testing.T) {
	// Nothing listens on this port.
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		MaxRetries:   -1,
	})
	defer rdb.Close()

	store := newPaymentStore(rdb)
	addr := &PaymentAddress{Address: "0x3333333333333333333333333333333333333333", OrderID: "o3"}
	payment := &Payment{ID: "p3", OrderID: "o3", ToAddress: addr.Address}

	if err := store.putPaymentWithAddress(payment, addr); err == nil {
		t.Fatal("a write to an unreachable store reported success")
	}
	// The local cache must not have been populated either, or a later read
	// would serve an address that was never stored.
	if _, ok := store.addrs[addr.Address]; ok {
		t.Error("the cache was populated despite the write failing")
	}
}
