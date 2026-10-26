package services

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tigerwallet/admin/internal/config"
	"github.com/tigerwallet/admin/internal/models"
)

func testTwoFactorService() *TwoFactorService {
	return &TwoFactorService{cfg: &config.Config{PasswordPepper: "test-pepper"}}
}

func TestGenerateBackupCodesHaveFullEntropy(t *testing.T) {
	s := testTwoFactorService()

	codes := s.generateBackupCodes(10)
	if len(codes) != 10 {
		t.Fatalf("expected 10 codes, got %d", len(codes))
	}

	seen := map[string]bool{}
	for _, code := range codes {
		// Canonical form is four groups of four base32 characters.
		parts := strings.Split(code, "-")
		if len(parts) != 4 {
			t.Fatalf("code %q does not have 4 groups", code)
		}
		for _, p := range parts {
			if len(p) != backupCodeGroupLen {
				t.Fatalf("group %q in code %q has wrong length", p, code)
			}
		}
		if normalizeBackupCode(code) == "" {
			t.Fatalf("code %q normalises to empty", code)
		}
		if seen[code] {
			t.Fatalf("duplicate backup code generated: %q", code)
		}
		seen[code] = true
	}
}

// TestBackupCodesAreNotTriviallyGuessable guards the regression where codes were
// a single 4-character group (~20 bits).
func TestBackupCodesAreNotTriviallyGuessable(t *testing.T) {
	s := testTwoFactorService()
	code := s.generateBackupCodes(1)[0]
	if len(normalizeBackupCode(code)) < 16 {
		t.Fatalf("backup code %q has too little entropy", code)
	}
}

func TestNormalizeBackupCode(t *testing.T) {
	const want = "ABCD2345EFGH6789"
	for _, input := range []string{
		"ABCD-2345-EFGH-6789",
		"abcd2345efgh6789",
		"  ABCD 2345 EFGH 6789  ",
	} {
		if got := normalizeBackupCode(input); got != want {
			t.Errorf("normalizeBackupCode(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestHashCodeIsSaltedAndFormatInsensitive(t *testing.T) {
	s := testTwoFactorService()

	withDashes := s.hashCode("ABCD-2345-EFGH-6789")
	withoutDashes := s.hashCode("abcd2345efgh6789")
	if withDashes != withoutDashes {
		t.Fatal("hash should be insensitive to case and grouping")
	}

	// The pepper must change the digest, otherwise a leaked hash column can be
	// attacked with a precomputed table.
	other := &TwoFactorService{cfg: &config.Config{PasswordPepper: "different-pepper"}}
	if other.hashCode("ABCD-2345-EFGH-6789") == withDashes {
		t.Fatal("hash must depend on the configured pepper")
	}

	if strings.Contains(withDashes, "ABCD") {
		t.Fatal("hash must not contain the plaintext code")
	}
}

func TestStoredBackupCodeHashesParsesPersistedMap(t *testing.T) {
	s := testTwoFactorService()
	code := "ABCD-2345-EFGH-6789"

	raw, err := json.Marshal(map[string]bool{s.hashCode(code): false})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	hashes := s.storedBackupCodeHashes(models.TwoFactorAuth{BackupCodesHashed: string(raw)})
	if len(hashes) != 1 {
		t.Fatalf("expected 1 stored hash, got %d", len(hashes))
	}
	if used, ok := hashes[s.hashCode(code)]; !ok || used {
		t.Fatal("stored hash should be present and unused")
	}
}

func TestStoredBackupCodeHashesToleratesCorruptData(t *testing.T) {
	s := testTwoFactorService()
	for _, raw := range []string{"", "not-json", "[]"} {
		if got := s.storedBackupCodeHashes(models.TwoFactorAuth{BackupCodesHashed: raw}); len(got) != 0 {
			t.Fatalf("corrupt input %q should yield no hashes, got %v", raw, got)
		}
	}
}
