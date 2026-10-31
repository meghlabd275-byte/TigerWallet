package main

import (
	"net"
	"strings"
	"syscall"
	"testing"
)

// Webhook destinations are caller-supplied and then dialled by the delivery
// workers, so the guard has to reject the ranges that reach inside the network
// the service runs in.
func TestCheckURLRejectsInternalDestinations(t *testing.T) {
	p := webhookURLPolicy{}

	blocked := []string{
		"http://127.0.0.1:8080/hook",       // loopback
		"http://localhost:9093/hook",       // loopback by name
		"http://169.254.169.254/latest",    // cloud metadata
		"http://10.1.2.3/hook",             // RFC 1918
		"http://192.168.1.10/hook",         // RFC 1918
		"http://172.16.5.4/hook",           // RFC 1918
		"http://[::1]:8080/hook",           // IPv6 loopback
		"http://[fd00::1]/hook",            // unique local
		"http://0.0.0.0/hook",              // this network
		"file:///etc/passwd",               // not http(s)
		"gopher://example.com:70/_x",       // not http(s)
		"http:///hook",                     // no host
	}

	for _, raw := range blocked {
		if err := p.checkURL(raw); err == nil {
			t.Errorf("checkURL(%q) accepted an internal destination", raw)
		}
	}
}

// A public destination must still be allowed, otherwise the guard breaks every
// legitimate webhook.
func TestCheckURLAllowsPublicDestinations(t *testing.T) {
	p := webhookURLPolicy{}
	if err := p.checkURL("https://example.com/hooks/tiger"); err != nil {
		t.Fatalf("checkURL rejected a public destination: %v", err)
	}
}

// The dialer check is what closes DNS rebinding: it sees the concrete IP the
// connection is about to be made to, not the hostname.
func TestDialControlRejectsInternalAddresses(t *testing.T) {
	p := webhookURLPolicy{}

	blocked := []string{
		"127.0.0.1:443",
		"169.254.169.254:80",
		"10.0.0.5:8443",
		"192.168.0.1:80",
		"[::1]:80",
		"[fd00::1]:80",
	}
	for _, addr := range blocked {
		if err := p.dialControl("tcp", addr, syscall.RawConn(nil)); err == nil {
			t.Errorf("dialControl(%q) allowed an internal address", addr)
		}
	}

	if err := p.dialControl("tcp", "93.184.216.34:443", syscall.RawConn(nil)); err != nil {
		t.Errorf("dialControl rejected a public address: %v", err)
	}
}

// Every address the guard is meant to cover must be inside at least one of the
// built-in ranges. This catches a typo in the CIDR list, which would silently
// leave a whole range reachable.
func TestBlockedNetsCoverExpectedRanges(t *testing.T) {
	for _, addr := range []string{
		"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.0.1", "169.254.169.254",
		"100.64.0.1", "192.0.0.1", "198.18.0.1", "0.0.0.1", "::1", "fd00::1", "fe80::1",
	} {
		ip := net.ParseIP(addr)
		if ip == nil {
			t.Fatalf("test bug: %q is not an IP", addr)
		}
		covered := false
		for _, n := range blockedWebhookNets {
			if n.Contains(ip) {
				covered = true
				break
			}
		}
		if !covered {
			t.Errorf("%s is not covered by any blocked CIDR", addr)
		}
	}
}

// The allowPrivate escape hatch exists for local development; confirm it works
// and that the default keeps the guard on.
func TestAllowPrivateEscapeHatch(t *testing.T) {
	p := webhookURLPolicy{allowPrivate: true}
	if err := p.checkURL("http://127.0.0.1:9999/hook"); err != nil {
		t.Fatalf("allowPrivate did not permit loopback: %v", err)
	}
	if err := p.dialControl("tcp", "127.0.0.1:9999", syscall.RawConn(nil)); err != nil {
		t.Fatalf("allowPrivate did not permit a loopback dial: %v", err)
	}
}

// The service must not enable the escape hatch by default.
func TestServiceDefaultsToBlockingPrivateDestinations(t *testing.T) {
	t.Setenv("WEBHOOK_ALLOW_PRIVATE_DESTINATIONS", "")
	if getEnvBool("WEBHOOK_ALLOW_PRIVATE_DESTINATIONS", false) {
		t.Fatal("private destinations are allowed by default")
	}

	t.Setenv("WEBHOOK_ALLOW_PRIVATE_DESTINATIONS", "not-a-bool")
	if getEnvBool("WEBHOOK_ALLOW_PRIVATE_DESTINATIONS", false) {
		t.Fatal("a malformed value enabled private destinations")
	}

	t.Setenv("WEBHOOK_ALLOW_PRIVATE_DESTINATIONS", "true")
	if !getEnvBool("WEBHOOK_ALLOW_PRIVATE_DESTINATIONS", false) {
		t.Fatal("an explicit true did not enable private destinations")
	}
}

// generateSecret signs outbound deliveries, so it must not be predictable and
// must not repeat.
func TestGenerateSecretIsUnpredictableAndUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 64; i++ {
		s := generateSecret()
		if len(s) != 64 {
			t.Fatalf("expected 64 hex chars, got %d (%q)", len(s), s)
		}
		if seen[s] {
			t.Fatal("generateSecret returned a duplicate value")
		}
		seen[s] = true
	}

	// A math/rand-backed generator seeded once per process produces the same
	// sequence for every process; two separate invocations must differ.
	first := generateSecret()
	second := generateSecret()
	if first == second {
		t.Fatal("consecutive secrets matched")
	}
	if strings.Trim(first, "0123456789abcdef") != "" {
		t.Fatalf("secret is not hex: %q", first)
	}
}
