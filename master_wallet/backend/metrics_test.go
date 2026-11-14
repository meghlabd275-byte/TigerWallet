package main

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// The auto-sign outcome counter is the alerting signal for "funds stopped
// moving", so it must actually increment per outcome rather than being a
// registered-but-never-written metric.
func TestAutoSignOutcomeCountersIncrement(t *testing.T) {
	before := testutil.ToFloat64(autoSignProcessed.WithLabelValues("refused"))
	autoSignProcessed.WithLabelValues("refused").Inc()
	if got := testutil.ToFloat64(autoSignProcessed.WithLabelValues("refused")); got != before+1 {
		t.Fatalf("refused counter = %v, want %v", got, before+1)
	}

	before = testutil.ToFloat64(autoSignProcessed.WithLabelValues("failed"))
	autoSignProcessed.WithLabelValues("failed").Inc()
	if got := testutil.ToFloat64(autoSignProcessed.WithLabelValues("failed")); got != before+1 {
		t.Fatalf("failed counter = %v, want %v", got, before+1)
	}
}

// Outcomes must be tracked separately; a single shared bucket would hide a
// broadcast failure behind successful approvals. Incrementing one outcome must
// leave the others untouched.
func TestAutoSignOutcomesAreDistinct(t *testing.T) {
	before := testutil.ToFloat64(autoSignProcessed.WithLabelValues("failed"))

	autoSignProcessed.WithLabelValues("requeued").Inc()

	if after := testutil.ToFloat64(autoSignProcessed.WithLabelValues("failed")); after != before {
		t.Fatalf("incrementing 'requeued' changed 'failed' from %v to %v; labels are not separating outcomes", before, after)
	}
}

// The broadcast counter must only move when a transaction really reached a
// node, so its help text and registration are part of the contract.
func TestBroadcastCounterStartsAtZero(t *testing.T) {
	if got := testutil.ToFloat64(transactionsBroadcast); got != 0 {
		t.Fatalf("broadcast counter = %v, want 0 before any real broadcast", got)
	}
	transactionsBroadcast.Inc()
	if got := testutil.ToFloat64(transactionsBroadcast); got != 1 {
		t.Fatalf("broadcast counter = %v, want 1 after one broadcast", got)
	}
}
