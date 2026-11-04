package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// newTestService builds a SlackService with just the configuration the request
// verification reads. No database or Redis is contacted by the paths under test.
func newTestService(signingSecret string) *SlackService {
	return &SlackService{
		config: SlackConfig{SigningSecret: signingSecret},
	}
}

// slackSign produces the signature Slack would send for a body.
func slackSign(secret, timestamp, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("v0:%s:%s", timestamp, body)))
	return "v0=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// TestVerifySignatureRefusesWhenUnconfigured is the regression test for an open
// webhook. VerifySignature returned true whenever the signing secret was empty,
// so any deployment that had not set SLACK_SIGNING_SECRET accepted every
// request on /slack/events, /slack/commands and /slack/interactive as though
// Slack had sent it. The secret defaults to empty, so this was the state of a
// default deployment.
func TestVerifySignatureRefusesWhenUnconfigured(t *testing.T) {
	service := newTestService("")

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	body := `{"type":"event_callback","event":{"type":"message"}}`

	if service.VerifySignature(timestamp, "v0=anything", body) {
		t.Fatal("a request was accepted with no signing secret configured")
	}
}

// TestVerifySignatureAcceptsOnlyCorrectlySignedRequests checks the two ways a
// request can be wrong: a signature that does not match, and a body that has
// been altered after signing.
func TestVerifySignatureAcceptsOnlyCorrectlySignedRequests(t *testing.T) {
	const secret = "test-signing-secret"
	service := newTestService(secret)

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	body := `{"type":"event_callback","event":{"type":"message"}}`
	goodSignature := slackSign(secret, timestamp, body)

	if !service.VerifySignature(timestamp, goodSignature, body) {
		t.Fatal("a correctly signed request was rejected")
	}

	if service.VerifySignature(timestamp, "v0="+base64.StdEncoding.EncodeToString([]byte("wrong")), body) {
		t.Fatal("a request with a wrong signature was accepted")
	}

	if service.VerifySignature(timestamp, goodSignature, body+` `) {
		t.Fatal("a request whose body was altered after signing was accepted")
	}

	if service.VerifySignature(timestamp, slackSign("other-secret", timestamp, body), body) {
		t.Fatal("a request signed with another secret was accepted")
	}
}

// TestVerifySignatureRejectsReplayedRequests covers the timestamp window. The
// timestamp is part of the signed string, so a captured request stays valid
// forever unless the age is checked.
func TestVerifySignatureRejectsReplayedRequests(t *testing.T) {
	const secret = "test-signing-secret"
	service := newTestService(secret)

	body := `{"type":"event_callback"}`
	old := fmt.Sprintf("%d", time.Now().Add(-time.Hour).Unix())

	if service.VerifySignature(old, slackSign(secret, old, body), body) {
		t.Fatal("a request timestamped an hour ago was accepted")
	}

	// A timestamp slightly in the future is a clock skew, not a replay, and is
	// still inside the window.
	soon := fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix())
	if !service.VerifySignature(soon, slackSign(secret, soon, body), body) {
		t.Fatal("a request a minute in the future was rejected")
	}

	if service.VerifySignature("not-a-number", "v0=x", body) {
		t.Fatal("a non-numeric timestamp was accepted")
	}
}

// TestEventsHandlerRejectsUnsignedRequest drives the real HTTP handler. The
// request is well-formed and would previously have been processed.
func TestEventsHandlerRejectsUnsignedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := newTestService("")

	router := gin.New()
	router.POST("/slack/events", service.SlackEventsHandler)

	body := `{"type":"event_callback","event":{"type":"message","text":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/slack/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an unsigned request, got %d", recorder.Code)
	}
}

// TestEventsHandlerSurvivesMalformedPayload covers the type assertions that
// used to be bare. A payload with no event object, or an event with no type,
// panicked inside the handler and gin turned that into a 500.
func TestEventsHandlerSurvivesMalformedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-signing-secret"
	service := newTestService(secret)

	router := gin.New()
	router.POST("/slack/events", service.SlackEventsHandler)

	cases := map[string]string{
		"event callback without an event object": `{"type":"event_callback"}`,
		"event object is not an object":          `{"type":"event_callback","event":"not-an-object"}`,
		"event without a type":                   `{"type":"event_callback","event":{"text":"hello"}}`,
		"body is not JSON":                       `not json at all`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			timestamp := fmt.Sprintf("%d", time.Now().Unix())
			req := httptest.NewRequest(http.MethodPost, "/slack/events", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Slack-Request-Timestamp", timestamp)
			req.Header.Set("X-Slack-Signature", slackSign(secret, timestamp, body))
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code == http.StatusInternalServerError {
				t.Fatalf("the handler panicked on a malformed payload: %d", recorder.Code)
			}
		})
	}
}

// TestSlashCommandsDoNotInventWalletData checks that the commands no longer
// answer with a fabricated deposit address, a balance that was never fetched,
// a withdrawal that was never submitted, or a price that was never looked up.
func TestSlashCommandsDoNotInventWalletData(t *testing.T) {
	service := newTestService("test-signing-secret")

	fabricated := map[string]string{
		"/deposit":  "0x...",
		"/balance":  "Your balance is being fetched...",
		"/price":    "Fetching price for",
		"/withdraw": "submitted for processing",
	}

	for command, marker := range fabricated {
		_, ephemeral, blocks := service.HandleSlashCommand(command, "1 BTC", "U1", "C1")
		if strings.Contains(ephemeral, marker) {
			t.Fatalf("%s still answers with invented data: %q", command, ephemeral)
		}
		for _, block := range blocks {
			if strings.Contains(fmt.Sprintf("%v", block), marker) {
				t.Fatalf("%s still answers with invented data in a block", command)
			}
		}
	}
}
