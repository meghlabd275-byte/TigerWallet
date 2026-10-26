package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// --- amount conversion ------------------------------------------------------

// TestToMinorUnitsExact covers the precision cases a float multiply gets wrong.
func TestToMinorUnitsExact(t *testing.T) {
	cases := []struct {
		amount float64
		want   int64
	}{
		{1, 100},
		{0.01, 1},
		{10.5, 1050},
		{99.99, 9999},
		{0.1, 10},
		{1234.56, 123456},
	}

	for _, tc := range cases {
		got, err := toMinorUnits(tc.amount)
		if err != nil {
			t.Fatalf("toMinorUnits(%v): %v", tc.amount, err)
		}
		if got != tc.want {
			t.Fatalf("toMinorUnits(%v) = %d, want %d", tc.amount, got, tc.want)
		}
	}
}

func TestToMinorUnitsRejectsInvalid(t *testing.T) {
	for _, amount := range []float64{0, -1, -0.01} {
		if _, err := toMinorUnits(amount); err == nil {
			t.Fatalf("toMinorUnits(%v) should have failed", amount)
		}
	}
}

// --- provider configuration -------------------------------------------------

// TestStripeClientFailsClosed ensures the gateway refuses to run without a key
// rather than accepting payments it cannot settle.
func TestStripeClientFailsClosed(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "")
	if _, err := NewStripeClient(); err != errProviderNotConfigured {
		t.Fatalf("err = %v, want errProviderNotConfigured", err)
	}
}

func TestStripeClientUsesConfiguredBaseURL(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_dummy")
	t.Setenv("STRIPE_API_BASE_URL", "https://example.test/")

	client, err := NewStripeClient()
	if err != nil {
		t.Fatalf("NewStripeClient: %v", err)
	}
	if client.baseURL != "https://example.test" {
		t.Fatalf("baseURL = %q, want the trailing slash removed", client.baseURL)
	}
}

// TestStripeClientSurfacesProviderError checks a declined card is reported
// rather than treated as success.
func TestStripeClientSurfacesProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"type":"card_error","code":"card_declined","message":"Your card was declined."}}`))
	}))
	defer server.Close()

	t.Setenv("STRIPE_SECRET_KEY", "sk_test_dummy")
	t.Setenv("STRIPE_API_BASE_URL", server.URL)

	client, err := NewStripeClient()
	if err != nil {
		t.Fatalf("NewStripeClient: %v", err)
	}

	if _, _, err := client.CreateCharge(context.Background(), 100, "usd", "pm_1", "order"); err == nil {
		t.Fatal("a declined charge must return an error")
	}
}

// TestStripeClientReturnsRealReference checks the reference comes from the
// provider rather than being generated locally.
func TestStripeClientReturnsRealReference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pi_3AbCdEf","status":"succeeded","paid":true}`))
	}))
	defer server.Close()

	t.Setenv("STRIPE_SECRET_KEY", "sk_test_dummy")
	t.Setenv("STRIPE_API_BASE_URL", server.URL)

	client, err := NewStripeClient()
	if err != nil {
		t.Fatalf("NewStripeClient: %v", err)
	}

	ref, _, err := client.CreateCharge(context.Background(), 100, "usd", "pm_1", "order")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if ref != "pi_3AbCdEf" {
		t.Fatalf("reference = %q, want the provider id", ref)
	}
}

// --- authentication ---------------------------------------------------------

func newPaymentRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(RequireAuth())
	{
		api.GET("/payment-methods/:user_id", requireSelf, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"user_id": currentUserID(c)})
		})
	}
	return r
}

func paymentToken(t *testing.T, secret, subject string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": subject,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestPaymentAuthRequiresConfiguration(t *testing.T) {
	t.Setenv("PAYMENT_GATEWAY_JWT_SECRET", "")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-methods/1", nil)
	req.Header.Set("Authorization", "Bearer anything")
	newPaymentRouter().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 without a secret", w.Code)
	}
}

func TestPaymentAuthRejectsMissingToken(t *testing.T) {
	t.Setenv("PAYMENT_GATEWAY_JWT_SECRET", "test-secret")

	w := httptest.NewRecorder()
	newPaymentRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/payment-methods/1", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// TestPaymentAuthBlocksCrossUserRead covers the IDOR: a valid token must not
// grant access to another user's payment methods.
func TestPaymentAuthBlocksCrossUserRead(t *testing.T) {
	const secret = "test-secret"
	t.Setenv("PAYMENT_GATEWAY_JWT_SECRET", secret)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-methods/2", nil)
	req.Header.Set("Authorization", "Bearer "+paymentToken(t, secret, "1"))
	newPaymentRouter().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 when reading another user", w.Code)
	}
}

func TestPaymentAuthAllowsOwnRead(t *testing.T) {
	const secret = "test-secret"
	t.Setenv("PAYMENT_GATEWAY_JWT_SECRET", secret)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-methods/7", nil)
	req.Header.Set("Authorization", "Bearer "+paymentToken(t, secret, "7"))
	newPaymentRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for the caller's own data (body %s)", w.Code, w.Body.String())
	}
}

// TestPaymentAuthRejectsAlgNone guards algorithm confusion.
func TestPaymentAuthRejectsAlgNone(t *testing.T) {
	t.Setenv("PAYMENT_GATEWAY_JWT_SECRET", "test-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": "1",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	unsigned, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build unsigned token: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-methods/1", nil)
	req.Header.Set("Authorization", "Bearer "+unsigned)
	newPaymentRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for an alg=none token", w.Code)
	}
}

func TestPaymentAuthRejectsWrongSecret(t *testing.T) {
	t.Setenv("PAYMENT_GATEWAY_JWT_SECRET", "expected")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-methods/1", nil)
	req.Header.Set("Authorization", "Bearer "+paymentToken(t, "attacker-secret", "1"))
	newPaymentRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a token signed with another key", w.Code)
	}
}

func TestPaymentAuthRejectsNonNumericSubject(t *testing.T) {
	const secret = "test-secret"
	t.Setenv("PAYMENT_GATEWAY_JWT_SECRET", secret)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-methods/1", nil)
	req.Header.Set("Authorization", "Bearer "+paymentToken(t, secret, "not-a-number"))
	newPaymentRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a non-numeric subject", w.Code)
	}
}
