package repository

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPasswordProducesBcryptWithRandomSalt(t *testing.T) {
	const pw = "correct-horse-battery"

	h1, err := hashPassword(pw)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	h2, err := hashPassword(pw)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}

	if _, err := bcrypt.Cost([]byte(h1)); err != nil {
		t.Fatalf("hash is not bcrypt formatted: %v", err)
	}
	if h1 == h2 {
		t.Fatal("identical passwords produced identical hashes: salt is not random")
	}
	if strings.Contains(h1, pw) {
		t.Fatal("hash embeds the plaintext password")
	}
}

func TestHashPasswordRejectsWeakInput(t *testing.T) {
	if _, err := hashPassword("short"); err == nil {
		t.Fatal("expected an error for a password below the minimum length")
	}
	if _, err := hashPassword(strings.Repeat("a", maxPasswordBytes+1)); err == nil {
		t.Fatal("expected an error for a password beyond the bcrypt input limit")
	}
}

func TestVerifyPassword(t *testing.T) {
	const pw = "correct-horse-battery"
	h, err := hashPassword(pw)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}

	if !verifyPassword(h, pw) {
		t.Fatal("correct password was rejected")
	}
	if verifyPassword(h, "wrong-password") {
		t.Fatal("incorrect password was accepted")
	}
	if verifyPassword("", pw) {
		t.Fatal("empty hash was accepted")
	}
}

// TestVerifyPasswordRejectsLegacySHA256 locks in the fix for the unsalted
// SHA-256 digest this package used to store: a downgraded hash must never
// authenticate.
func TestVerifyPasswordRejectsLegacySHA256(t *testing.T) {
	const pw = "correct-horse-battery"
	legacy := fmt.Sprintf("%x", sha256.Sum256([]byte(pw)))

	if verifyPassword(legacy, pw) {
		t.Fatal("legacy unsalted SHA-256 digest authenticated")
	}
}
