package handlers

import (
	"strings"
	"testing"
)

// A notification title is operator-supplied and is copied into the Subject
// header. An embedded newline would let the sender append headers of their
// choosing (a Bcc, for instance) to the outgoing message.
func TestBuildEmailMessageStripsHeaderNewlines(t *testing.T) {
	msg := buildEmailMessage(
		"ops@example.com",
		"user@example.com",
		"TigerWallet Notification: alert\r\nBcc: attacker@example.com",
		"body",
	)

	sep := strings.Index(msg, "\r\n\r\n")
	if sep < 0 {
		t.Fatalf("message has no header/body separator: %q", msg)
	}
	headerBlock := msg[:sep]

	// The sanitizer folds CR/LF into spaces, so the injected text may still
	// appear as ordinary characters inside the Subject value. What must not
	// happen is a second header line being produced.
	lines := strings.Split(headerBlock, "\r\n")
	if len(lines) != 6 {
		t.Fatalf("expected 6 header lines, got %d:\n%q", len(lines), headerBlock)
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "Bcc:") {
			t.Fatalf("injected header survived sanitization: %q", line)
		}
	}
}

func TestBuildEmailMessagePreservesOrdinaryHeaders(t *testing.T) {
	msg := buildEmailMessage("ops@example.com", "user@example.com", "hello", "body")
	for _, want := range []string{
		"From: ops@example.com\r\n",
		"To: user@example.com\r\n",
		"Subject: hello\r\n",
		"MIME-Version: 1.0\r\n",
		"\r\nbody",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in message:\n%q", want, msg)
		}
	}
}
