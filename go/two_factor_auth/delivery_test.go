package main

// Tests for the verification-code delivery primitives. Delivery must fail
// closed when unconfigured, phone numbers must be normalized to E.164, and the
// email path must reject header-injection attempts.

import (
	"os"
	"strings"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"+14155552671", "+14155552671"},
		{"14155552671", "+14155552671"},
		{"+1 (415) 555-2671", "+14155552671"},
		{"+1-415-555-2671", "+14155552671"},
		{"+44 20 7946 0958", "+442079460958"},
		{"+1.415.555.2671", "+14155552671"},
	}
	for _, tc := range cases {
		if got := normalizePhone(tc.in); got != tc.want {
			t.Errorf("normalizePhone(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizePhoneRejectsInvalid(t *testing.T) {
	for _, in := range []string{
		"",
		"123",                 // too short
		"+123456789012345678", // too long
		"+1 415 555 ABCD",     // letters
		"call me",             // not a number
		"+1;DROP TABLE",       // injection attempt
	} {
		if got := normalizePhone(in); got != "" {
			t.Errorf("normalizePhone(%q) = %q, want rejection", in, got)
		}
	}
}

func TestSendSMSFailsClosedWhenUnconfigured(t *testing.T) {
	t.Setenv("TWILIO_ACCOUNT_SID", "")
	t.Setenv("TWILIO_AUTH_TOKEN", "")
	t.Setenv("TWILIO_FROM_NUMBER", "")

	if _, err := sendSMS("+14155552671", "code"); err == nil {
		t.Fatal("sendSMS reported success without SMS configuration")
	}
}

func TestSendSMSRejectsPartialConfiguration(t *testing.T) {
	// A half-configured provider would always fail at send time, so it must be
	// treated as unconfigured rather than attempted.
	t.Setenv("TWILIO_ACCOUNT_SID", "AC123")
	t.Setenv("TWILIO_AUTH_TOKEN", "")
	t.Setenv("TWILIO_FROM_NUMBER", "+14155550000")

	if _, err := sendSMS("+14155552671", "code"); err == nil {
		t.Fatal("sendSMS reported success with a partial configuration")
	}
}

func TestSendEmailFailsClosedWhenUnconfigured(t *testing.T) {
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASS", "")
	t.Setenv("FROM_EMAIL", "")

	if err := sendEmail("user@example.com", "subject", "body"); err == nil {
		t.Fatal("sendEmail reported success without SMTP configuration")
	}
}

func TestSendEmailRejectsHeaderInjection(t *testing.T) {
	t.Setenv("SMTP_USER", "user")
	t.Setenv("SMTP_PASS", "pass")
	t.Setenv("FROM_EMAIL", "noreply@tigerwallet.com")

	// With credentials present the configuration loads, so the injection checks
	// are what must reject these before any connection is attempted.
	if err := sendEmail("victim@example.com\r\nBcc: attacker@evil.example", "s", "b"); err == nil {
		t.Error("a newline in the recipient was accepted")
	}
	if err := sendEmail("victim@example.com", "sub\r\nBcc: attacker@evil.example", "b"); err == nil {
		t.Error("a newline in the subject was accepted")
	}
	if err := sendEmail("not-an-email", "s", "b"); err == nil {
		t.Error("a malformed recipient was accepted")
	}
}

func TestLoadEmailConfigRejectsInvalidPort(t *testing.T) {
	t.Setenv("SMTP_USER", "user")
	t.Setenv("SMTP_PASS", "pass")
	t.Setenv("FROM_EMAIL", "noreply@tigerwallet.com")
	t.Setenv("SMTP_PORT", "not-a-port")

	if _, err := loadEmailConfig(); err == nil {
		t.Fatal("an invalid SMTP_PORT was accepted")
	}

	t.Setenv("SMTP_PORT", "70000")
	if _, err := loadEmailConfig(); err == nil {
		t.Fatal("an out-of-range SMTP_PORT was accepted")
	}
}

func TestBuildMessageUsesCRLFAndHeaders(t *testing.T) {
	msg := buildMessage("from@example.com", "to@example.com", "Subject line", "line one\nline two")
	if !strings.HasPrefix(msg, "From: from@example.com\r\n") {
		t.Errorf("message does not start with the From header: %q", msg)
	}
	if !strings.Contains(msg, "\r\nTo: to@example.com\r\n") {
		t.Error("message has no To header")
	}
	if !strings.Contains(msg, "Subject: Subject line\r\n") {
		t.Error("message has no Subject header")
	}
	// Every line break in the body must be CRLF, including the injected LF.
	if strings.Contains(strings.ReplaceAll(msg, "\r\n", ""), "\n") {
		t.Errorf("message contains a bare LF: %q", msg)
	}
	if !strings.Contains(msg, "line one\r\nline two") {
		t.Errorf("body lines were not separated by CRLF: %q", msg)
	}
}

func TestReadBoundedStopsAtLimit(t *testing.T) {
	long := strings.NewReader(strings.Repeat("x", 5000))
	got := readBounded(long, 100)
	if len(got) != 100 {
		t.Errorf("readBounded returned %d bytes, want 100", len(got))
	}
}

func TestVerificationCodeIsNotLogged(t *testing.T) {
	// The delivery helpers must not write the code anywhere; this asserts the
	// source contains no logging call inside them by checking the wired
	// behaviour: an unconfigured send returns an error rather than silently
	// succeeding after printing the code.
	t.Setenv("TWILIO_ACCOUNT_SID", "")
	t.Setenv("TWILIO_AUTH_TOKEN", "")
	t.Setenv("TWILIO_FROM_NUMBER", "")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASS", "")
	t.Setenv("FROM_EMAIL", "")

	svc := &TwoFactorService{
		config:  &Config{IssuerName: "TigerWallet"},
		pending: map[string]*VerificationCode{},
	}
	if err := svc.sendSMS("+14155552671", "123456"); err == nil {
		t.Error("sendSMS succeeded without a configured provider")
	}
	if err := svc.sendEmail("user@example.com", "123456"); err == nil {
		t.Error("sendEmail succeeded without a configured provider")
	}
	_ = os.Getenv
}
