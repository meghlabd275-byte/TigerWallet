package main

import (
	"bytes"
	"strings"
	"testing"
)

// Subject, recipient name and custom header values all arrive from API callers.
// buildEmail must not let a CRLF in any of them start a new header line.
func TestBuildEmailRejectsHeaderInjection(t *testing.T) {
	svc := &EmailService{config: EmailConfig{SMTPHost: "smtp.example.com"}}

	var buf bytes.Buffer
	err := svc.buildEmail(
		&buf,
		"<id@example.com>",
		"victim@example.com",
		"Victim\r\nBcc: attacker@example.com",
		"Subject line\r\nBcc: attacker@example.com",
		"<p>hi</p>",
		"hi",
		&EmailOptions{
			From:    "noreply@example.com",
			Headers: map[string]string{"X-Custom": "v\r\nBcc: attacker@example.com"},
		},
	)
	if err != nil {
		t.Fatalf("buildEmail: %v", err)
	}

	headerBlock := buf.String()
	if i := strings.Index(headerBlock, "\r\n\r\n"); i >= 0 {
		headerBlock = headerBlock[:i]
	}
	for _, line := range strings.Split(headerBlock, "\r\n") {
		if strings.HasPrefix(line, "Bcc:") {
			t.Fatalf("injected header survived sanitization: %q", line)
		}
	}
}

func TestRandomStringUsesFullAlphabet(t *testing.T) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	got := randomString(64)
	if len(got) != 64 {
		t.Fatalf("expected 64 characters, got %d", len(got))
	}
	for _, c := range got {
		if !strings.ContainsRune(letters, c) {
			t.Fatalf("unexpected character %q in %q", c, got)
		}
	}
	if randomString(32) == randomString(32) {
		t.Fatal("two consecutive tokens are identical")
	}
}
