package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

func newTestPaymentService() *PaymentService {
	return &PaymentService{
		payments:   make(map[string]*Payment),
		ethClients: make(map[string]*ethclient.Client),
	}
}

func postWebhook(t *testing.T, svc *PaymentService, secret string, body []byte) (int, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment/webhook/payment", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-Payment-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	c.Request = req

	svc.HandlePaymentWebhook(c)
	return rec.Code, rec.Body.String()
}

const depositCallback = `{"tx_hash":"0x` +
	`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","address":"0x1111111111111111111111111111111111111111"}`

// A pending deposit must not be completed by a caller who cannot sign. This is
// the forgery case: before the signature check the callback trusted the
// "confirmations" field in the body, so anyone could mark a deposit paid.
func TestDepositCallbackRejectsUnsignedCaller(t *testing.T) {
	t.Setenv("PAYMENT_WEBHOOK_SECRET", "chain-listener-secret")

	svc := newTestPaymentService()
	svc.payments["p1"] = &Payment{
		ID:        "p1",
		ToAddress: "0x1111111111111111111111111111111111111111",
		Amount:    "1000000",
		Token:     "USDT",
		Chain:     "ethereum",
		Status:    "pending",
	}

	code, _ := postWebhook(t, svc, "", []byte(depositCallback))
	if code != http.StatusUnauthorized {
		t.Fatalf("unsigned callback returned %d, want %d", code, http.StatusUnauthorized)
	}
	if svc.payments["p1"].Status != "pending" {
		t.Errorf("the payment moved to %q on an unsigned callback", svc.payments["p1"].Status)
	}
}

func TestDepositCallbackRejectsForgedSignature(t *testing.T) {
	t.Setenv("PAYMENT_WEBHOOK_SECRET", "chain-listener-secret")

	svc := newTestPaymentService()
	svc.payments["p1"] = &Payment{
		ID:        "p1",
		ToAddress: "0x1111111111111111111111111111111111111111",
		Amount:    "1000000",
		Token:     "USDT",
		Chain:     "ethereum",
		Status:    "pending",
	}

	code, _ := postWebhook(t, svc, "the-wrong-secret", []byte(depositCallback))
	if code != http.StatusUnauthorized {
		t.Fatalf("forged callback returned %d, want %d", code, http.StatusUnauthorized)
	}
	if svc.payments["p1"].Status != "pending" {
		t.Errorf("the payment moved to %q on a forged callback", svc.payments["p1"].Status)
	}
}

func TestDepositCallbackFailsClosedWithoutSecret(t *testing.T) {
	t.Setenv("PAYMENT_WEBHOOK_SECRET", "")

	svc := newTestPaymentService()
	code, _ := postWebhook(t, svc, "", []byte(depositCallback))
	if code != http.StatusServiceUnavailable {
		t.Fatalf("callback without a configured secret returned %d, want %d", code, http.StatusServiceUnavailable)
	}
}

// Even a correctly signed callback must not complete a payment that cannot be
// confirmed on chain. Here no RPC client is configured, so the transfer cannot
// be read and the payment must stay pending rather than being trusted.
func TestSignedCallbackWithoutChainEvidenceDoesNotComplete(t *testing.T) {
	const secret = "chain-listener-secret"
	t.Setenv("PAYMENT_WEBHOOK_SECRET", secret)

	svc := newTestPaymentService()
	svc.payments["p1"] = &Payment{
		ID:        "p1",
		ToAddress: "0x1111111111111111111111111111111111111111",
		Amount:    "1000000",
		Token:     "USDT",
		Chain:     "ethereum",
		Status:    "pending",
	}

	code, _ := postWebhook(t, svc, secret, []byte(depositCallback))
	if code == http.StatusOK {
		t.Fatal("a signed callback completed a payment with no on-chain evidence")
	}
	if svc.payments["p1"].Status != "pending" {
		t.Errorf("the payment moved to %q without on-chain evidence", svc.payments["p1"].Status)
	}
}

// A malformed hash must be rejected before any chain call is attempted.
func TestVerifyDepositRejectsMalformedHash(t *testing.T) {
	svc := newTestPaymentService()
	payment := &Payment{Chain: "ethereum", Token: "USDT", Amount: "1000000"}

	for _, hash := range []string{"", "0x1234", "not-a-hash", "aa" + "bb"} {
		if _, err := svc.verifyDepositOnChain(context.Background(), payment, hash); err == nil {
			t.Errorf("hash %q was accepted", hash)
		}
	}
}

// With no RPC client the verification must fail rather than assume success.
func TestVerifyDepositFailsClosedWithoutRPCClient(t *testing.T) {
	svc := newTestPaymentService()
	payment := &Payment{Chain: "ethereum", Token: "USDT", Amount: "1000000"}
	validHash := "0x" + "aa" + "0000000000000000000000000000000000000000000000000000000000"

	if _, err := svc.verifyDepositOnChain(context.Background(), payment, validHash); err == nil {
		t.Fatal("verification passed with no RPC client configured")
	}
}

// The service must take its chain endpoints, port, and confirmation depth from
// the environment. Without this it keeps its defaults, holds no RPC client, and
// can never verify a deposit.
func TestLoadConfigReadsTheEnvironment(t *testing.T) {
	original := cfg
	t.Cleanup(func() { cfg = original })

	t.Setenv("PORT", "9999")
	t.Setenv("REDIS_ADDR", "redis.example:6380")
	t.Setenv("ETH_RPC_URL", "https://eth.example")
	t.Setenv("BSC_RPC_URL", "https://bsc.example")
	t.Setenv("CONFIRMATION_BLOCKS", "6")
	t.Setenv("PAYMENT_PRIVATE_KEY", "0xabc")

	loadConfig()

	if cfg.Port != 9999 {
		t.Errorf("port = %d, want 9999", cfg.Port)
	}
	if cfg.RedisAddr != "redis.example:6380" {
		t.Errorf("redis addr = %q, want %q", cfg.RedisAddr, "redis.example:6380")
	}
	if cfg.EthRPCURL != "https://eth.example" {
		t.Errorf("eth rpc = %q, want %q", cfg.EthRPCURL, "https://eth.example")
	}
	if cfg.ConfirmationBlocks != 6 {
		t.Errorf("confirmation blocks = %d, want 6", cfg.ConfirmationBlocks)
	}
	if cfg.PrivateKey != "0xabc" {
		t.Errorf("private key was not read")
	}
	if got := countConfiguredRPCs(); got != 2 {
		t.Errorf("configured RPC count = %d, want 2", got)
	}
}

// Nonsense values must not corrupt the configuration: the previous value is
// kept rather than the service starting on a broken port or depth.
func TestLoadConfigIgnoresInvalidValues(t *testing.T) {
	original := cfg
	t.Cleanup(func() { cfg = original })

	t.Setenv("PORT", "not-a-number")
	t.Setenv("CONFIRMATION_BLOCKS", "-5")

	loadConfig()

	if cfg.Port != original.Port {
		t.Errorf("port = %d, want the previous %d", cfg.Port, original.Port)
	}
	if cfg.ConfirmationBlocks != original.ConfirmationBlocks {
		t.Errorf("confirmation blocks = %d, want the previous %d", cfg.ConfirmationBlocks, original.ConfirmationBlocks)
	}
}
