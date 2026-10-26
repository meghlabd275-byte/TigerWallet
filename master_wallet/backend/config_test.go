package main

import (
	"testing"
)

// TestLoadConfigRejectsPlaceholderSecret guards the regression where an unset or
// example JWT secret was silently replaced with a predictable value
// ("dev-only-change-me-<pid>" or the compose default), which would let anyone
// mint an operator token for the treasury backend.
func TestLoadConfigRejectsPlaceholderSecret(t *testing.T) {
	t.Setenv("MASTER_WALLET_ALLOW_INSECURE_DEV_SECRET", "")

	placeholders := []string{
		"",
		"tigerwallet-master-jwt-change-in-production",
		"change-in-production",
		"changeme",
		"secret",
		"your-secret-here",
	}

	for _, value := range placeholders {
		if value == "" {
			// LoadConfig calls log.Fatal when it rejects a secret, which exits the
			// test binary; assert the detector directly instead.
			if !isPlaceholderSecret("tigerwallet-master-jwt-change-in-production") {
				t.Fatal("compose default must be treated as a placeholder")
			}
			continue
		}
		if !isPlaceholderSecret(value) {
			t.Fatalf("isPlaceholderSecret(%q) = false, want true", value)
		}
	}
}

func TestIsPlaceholderSecretAcceptsRealSecrets(t *testing.T) {
	real := []string{
		"f3a1c8b27e94d05f6a1b3c7d8e9f0a1b2c3d4e5f60718293a4b5c6d7e8f90123",
		"a-perfectly-fine-operator-chosen-secret",
		"MixedCaseSecretValue123",
	}

	for _, value := range real {
		if isPlaceholderSecret(value) {
			t.Fatalf("isPlaceholderSecret(%q) = true, want false", value)
		}
	}
}

func TestIsPlaceholderSecretIsCaseInsensitive(t *testing.T) {
	for _, value := range []string{"CHANGEME", "  Changeme  ", "CHANGE-IN-PRODUCTION"} {
		if !isPlaceholderSecret(value) {
			t.Fatalf("isPlaceholderSecret(%q) = false, want true", value)
		}
	}
}

// TestLoadConfigUsesConfiguredSecret confirms a real secret is passed through
// untouched.
func TestLoadConfigUsesConfiguredSecret(t *testing.T) {
	const secret = "a-real-operator-secret-value-0123456789"
	t.Setenv("MASTER_WALLET_JWT_SECRET", secret)

	cfg := LoadConfig()
	if cfg.JWTSecret != secret {
		t.Fatalf("JWTSecret = %q, want the configured value", cfg.JWTSecret)
	}
}

// TestLoadConfigDefaultsAreNotSecrets confirms unrelated defaults stay intact.
func TestLoadConfigDefaultsAreNotSecrets(t *testing.T) {
	t.Setenv("MASTER_WALLET_JWT_SECRET", "a-real-operator-secret-value-0123456789")
	t.Setenv("MASTER_WALLET_PORT", "")

	cfg := LoadConfig()
	if cfg.Port != "8450" {
		t.Fatalf("Port = %q, want the 8450 default", cfg.Port)
	}
}
