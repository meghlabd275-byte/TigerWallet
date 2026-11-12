package main

import (
	"strings"
	"testing"
)

// TestGuestSentinelHashIsUsable pins the bug that made /auth/guest fail with a
// 500 for every client. The sentinel password was derived as
// uuid.NewString()+"|"+deviceID, which is 73 bytes for a 36-character UUID, and
// bcrypt rejects inputs longer than 72 bytes — so provisioning a guest account
// always failed. The sentinel must also never be verifiable as a password.
func TestGuestSentinelHashIsUsable(t *testing.T) {
	if guestPasswordHash == "" {
		t.Fatal("guest sentinel hash is empty")
	}
	if !strings.HasPrefix(guestPasswordHash, "$2") {
		t.Fatalf("guest sentinel is not a bcrypt hash: %q", guestPasswordHash)
	}
	// A guest account must not be loginable: no plausible password verifies.
	for _, guess := range []string{"", "guest", "password", "guest+device@tigerwallet.local"} {
		if VerifyPassword(guestPasswordHash, guess) {
			t.Errorf("guest sentinel verifies against %q — guest accounts would be loginable", guess)
		}
	}
}

// TestHashPasswordRejectsOversizedInput documents the bcrypt limit that caused
// the guest-auth failure, so the constraint is not silently reintroduced.
func TestHashPasswordRejectsOversizedInput(t *testing.T) {
	if _, err := HashPassword(strings.Repeat("a", 73)); err == nil {
		t.Error("HashPassword accepted a 73-byte password; bcrypt must reject it")
	}
	if _, err := HashPassword(strings.Repeat("a", 72)); err != nil {
		t.Errorf("HashPassword rejected a 72-byte password, which is the bcrypt maximum: %v", err)
	}
}

// TestValidDeviceID covers the identifiers the real clients send. The handler
// previously required a UUID, which rejected Android's 16-hex ANDROID_ID and
// 32-hex digests from other platforms.
func TestValidDeviceID(t *testing.T) {
	valid := []string{
		"3f2504e0-4f89-11d3-9a0c-0305e82c3301", // UUID
		"9774d56d682e549c",                     // Android ANDROID_ID (16 hex)
		"9f8e7d6c5b4a39281706f5e4d3c2b1a0",     // 32-hex digest
		"device_id-with_dashes-1234",
	}
	for _, id := range valid {
		if !validDeviceID(id) {
			t.Errorf("validDeviceID(%q) = false, want true", id)
		}
	}

	invalid := []string{
		"",                                  // empty
		"short",                             // guessable
		"9774d56d682e549",                   // 15 characters
		strings.Repeat("a", 65),             // too long
		"has spaces in the identifier",      // illegal character
		"device+id@example.com",             // illegal characters
		"'; DROP TABLE users; --aaaaaaaaaa", // injection-shaped
	}
	for _, id := range invalid {
		if validDeviceID(id) {
			t.Errorf("validDeviceID(%q) = true, want false", id)
		}
	}
}
