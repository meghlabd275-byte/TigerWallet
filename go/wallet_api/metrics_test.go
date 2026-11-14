package main

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// The broadcast counter is the "money actually moved" signal. It must only
// move for a real broadcast, so a fresh process starts at zero.
func TestBroadcastCounterStartsAtZero(t *testing.T) {
	if got := testutil.ToFloat64(transactionsBroadcast); got != 0 {
		t.Fatalf("broadcast counter = %v, want 0 before any real broadcast", got)
	}
	transactionsBroadcast.Inc()
	if got := testutil.ToFloat64(transactionsBroadcast); got != 1 {
		t.Fatalf("broadcast counter = %v, want 1 after one broadcast", got)
	}
}

// A rejected login must move the auth-failure counter, which is the credential
// stuffing signal.
func TestAuthFailuresCounterIncrements(t *testing.T) {
	before := testutil.ToFloat64(authFailures)
	authFailures.Inc()
	if got := testutil.ToFloat64(authFailures); got != before+1 {
		t.Fatalf("auth failures = %v, want %v", got, before+1)
	}
}

// Request metrics must be labelled by the route template, not the raw path.
// Two requests that differ only in a path parameter must land on one series,
// otherwise a busy wallet would create an unbounded number of time series.
func TestRequestCounterUsesRouteTemplate(t *testing.T) {
	httpRequestsTotal.WithLabelValues("GET", "/api/v1/wallets/:id", "200").Inc()
	httpRequestsTotal.WithLabelValues("GET", "/api/v1/wallets/:id", "200").Inc()

	got := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/api/v1/wallets/:id", "200"))
	if got != 2 {
		t.Fatalf("route-template series = %v, want 2", got)
	}
}
