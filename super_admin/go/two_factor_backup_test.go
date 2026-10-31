package main

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestBackupCodeAlphabetHasPowerOfTwoLength guards the modulo reduction in
// generateBackupCode: 256 % len(alphabet) must be 0, otherwise some characters
// would be produced more often than others and the effective entropy would drop.
func TestBackupCodeAlphabetHasPowerOfTwoLength(t *testing.T) {
	if len(backupCodeAlphabet) != 32 {
		t.Fatalf("backup code alphabet has %d characters, want 32 so that 256%%32==0", len(backupCodeAlphabet))
	}
	if 256%len(backupCodeAlphabet) != 0 {
		t.Fatalf("256 %% %d != 0, the modulo reduction would be biased", len(backupCodeAlphabet))
	}
}

// TestGenerateBackupCode checks the codes are the expected length, drawn only
// from the unambiguous alphabet, and distinct across a batch.
func TestGenerateBackupCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		code, err := generateBackupCode()
		if err != nil {
			t.Fatalf("generateBackupCode: %v", err)
		}
		if len(code) != 10 {
			t.Fatalf("code %q has length %d, want 10", code, len(code))
		}
		for _, r := range code {
			if !strings.ContainsRune(backupCodeAlphabet, r) {
				t.Fatalf("code %q contains %q, which is outside the alphabet", code, r)
			}
		}
		// Confusable characters must never appear.
		if strings.ContainsAny(code, "IO01") {
			t.Fatalf("code %q contains a confusable character", code)
		}
		if seen[code] {
			t.Fatalf("duplicate code %q in 200 draws", code)
		}
		seen[code] = true
	}
}

// TestBackupCodeHashRoundTrip mirrors how replaceBackupCodes stores a code and
// how consumeBackupCode verifies it: the database only ever holds a bcrypt hash.
func TestBackupCodeHashRoundTrip(t *testing.T) {
	code, err := generateBackupCode()
	if err != nil {
		t.Fatalf("generateBackupCode: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if string(hash) == code {
		t.Fatal("the stored value is the plaintext code")
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte(code)); err != nil {
		t.Fatalf("the issued code does not verify against its own hash: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte("WRONGCODE1")); err == nil {
		t.Fatal("an unrelated code verified against the hash")
	}
}
