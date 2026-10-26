package main

import (
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// --- money conversion -------------------------------------------------------

// TestDecimalToBaseUnits checks that payout amounts convert without floating
// point drift. The previous code passed amounts around as float64 and would have
// paid an imprecise sum.
func TestDecimalToBaseUnits(t *testing.T) {
	cases := []struct {
		amount   float64
		decimals uint8
		want     string
	}{
		{1, 6, "1000000"},
		{0.5, 6, "500000"},
		{100, 18, "100000000000000000000"},
		{0.000001, 6, "1"},
		{1234.567891, 6, "1234567891"},
	}

	for _, tc := range cases {
		got, err := decimalToBaseUnits(tc.amount, tc.decimals)
		if err != nil {
			t.Fatalf("decimalToBaseUnits(%v, %d): %v", tc.amount, tc.decimals, err)
		}
		if got.String() != tc.want {
			t.Fatalf("decimalToBaseUnits(%v, %d) = %s, want %s", tc.amount, tc.decimals, got.String(), tc.want)
		}
	}
}

// TestDecimalToBaseUnitsRejectsInvalid guards against paying a zero, negative,
// NaN or infinite amount.
func TestDecimalToBaseUnitsRejectsInvalid(t *testing.T) {
	invalid := []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)}

	for _, amount := range invalid {
		if _, err := decimalToBaseUnits(amount, 6); err == nil {
			t.Fatalf("decimalToBaseUnits(%v, 6) should have failed", amount)
		}
	}
}

// TestDecimalStringToBaseUnitsExact covers the precision cases that a float
// multiply gets wrong.
func TestDecimalStringToBaseUnitsExact(t *testing.T) {
	cases := []struct {
		raw      string
		decimals uint8
		want     string
	}{
		{"0.000001", 6, "1"},
		{"0.1", 18, "100000000000000000"},
		{"0.1", 6, "100000"},
		{"1234.567891", 6, "1234567891"},
		{"1", 6, "1000000"},
		{"0.123456789", 6, "123456"},
		{"1.0000000", 6, "1000000"},
		{"10", 0, "10"},
	}

	for _, tc := range cases {
		got, err := decimalStringToBaseUnits(tc.raw, tc.decimals)
		if err != nil {
			t.Fatalf("decimalStringToBaseUnits(%q, %d): %v", tc.raw, tc.decimals, err)
		}
		if got.String() != tc.want {
			t.Fatalf("decimalStringToBaseUnits(%q, %d) = %s, want %s", tc.raw, tc.decimals, got.String(), tc.want)
		}
	}
}

// TestDecimalStringToBaseUnitsRejectsMalformed ensures a bad amount never turns
// into a payout.
func TestDecimalStringToBaseUnitsRejectsMalformed(t *testing.T) {
	for _, raw := range []string{"", " ", "abc", "1.2.3", "-1", "+1", "1.", ".", "0", "0.0", "1e6"} {
		if _, err := decimalStringToBaseUnits(raw, 6); err == nil {
			t.Fatalf("decimalStringToBaseUnits(%q, 6) should have failed", raw)
		}
	}
}

// --- address and ABI helpers ------------------------------------------------

func TestParseAddress(t *testing.T) {
	if _, err := parseAddress("0x1111111111111111111111111111111111111111"); err != nil {
		t.Fatalf("valid address rejected: %v", err)
	}
	for _, bad := range []string{"", "0x", "1111111111111111111111111111111111111111", "0x1234", "0xZZ11111111111111111111111111111111111111"} {
		if _, err := parseAddress(bad); err == nil {
			t.Fatalf("parseAddress(%q) should have failed", bad)
		}
	}
}

func TestPadLeft32(t *testing.T) {
	word := padLeft32(big.NewInt(1))
	if len(word) != 32 {
		t.Fatalf("length = %d, want 32", len(word))
	}
	if word[31] != 1 {
		t.Fatalf("last byte = %d, want 1", word[31])
	}
}

// TestSelectorMatchesERC20Transfer pins the transfer selector, which is the
// difference between a real payout and a call to an arbitrary function.
func TestSelectorMatchesERC20Transfer(t *testing.T) {
	got := selector("transfer(address,uint256)")
	want := []byte{0xa9, 0x05, 0x9c, 0xbb}
	if len(got) != 4 {
		t.Fatalf("selector length = %d, want 4", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("transfer selector = %x, want %x", got, want)
		}
	}
}

// --- payout client configuration -------------------------------------------

// TestPayoutClientFailsClosed covers the regression where a payout would be
// recorded with a fabricated hash. Without an RPC endpoint or a key the client
// must refuse to exist.
func TestPayoutClientFailsClosed(t *testing.T) {
	t.Setenv("PROTECTION_FUND_RPC_URL", "")
	t.Setenv("PROTECTION_FUND_PRIVATE_KEY", "")

	if _, err := NewPayoutClient(nil); err != errRPCNotConfigured {
		t.Fatalf("err = %v, want errRPCNotConfigured", err)
	}

	t.Setenv("PROTECTION_FUND_RPC_URL", "http://127.0.0.1:0")
	if _, err := NewPayoutClient(nil); err != errPayoutKeyNotConfigured {
		t.Fatalf("err = %v, want errPayoutKeyNotConfigured", err)
	}
}

func TestPayoutClientRejectsMalformedKey(t *testing.T) {
	t.Setenv("PROTECTION_FUND_RPC_URL", "http://127.0.0.1:8545")
	t.Setenv("PROTECTION_FUND_PRIVATE_KEY", "not-a-key")

	if _, err := NewPayoutClient(nil); err == nil {
		t.Fatal("a malformed payout key must be rejected")
	}
}

// --- authentication ---------------------------------------------------------

func newFundRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	if secret == "" {
		t := ""
		_ = t
	}

	r := gin.New()
	r.GET("/protected", RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetString("user_id"), "role": c.GetString("role")})
	})
	r.GET("/admin", RequireAuth(), RequireRole("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestFundAuthRequiresConfiguration(t *testing.T) {
	t.Setenv("PROTECTION_FUND_JWT_SECRET", "")

	r := newFundRouter("")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer whatever")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 when no secret is configured", w.Code)
	}
}

func TestFundAuthRejectsMissingToken(t *testing.T) {
	t.Setenv("PROTECTION_FUND_JWT_SECRET", "test-secret")

	r := newFundRouter("test-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestFundAuthAcceptsValidToken(t *testing.T) {
	const secret = "test-secret"
	t.Setenv("PROTECTION_FUND_JWT_SECRET", secret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "operator-1",
		"role":    "operator",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	r := newFundRouter(secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
}

func TestFundAuthRejectsWrongSecret(t *testing.T) {
	t.Setenv("PROTECTION_FUND_JWT_SECRET", "expected")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "attacker",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte("attacker-secret"))

	r := newFundRouter("expected")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a token signed with the wrong key", w.Code)
	}
}

// TestFundAuthRejectsAlgNone guards the algorithm-confusion hole.
func TestFundAuthRejectsAlgNone(t *testing.T) {
	t.Setenv("PROTECTION_FUND_JWT_SECRET", "test-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": "attacker",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	unsigned, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build unsigned token: %v", err)
	}

	r := newFundRouter("test-secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+unsigned)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for an alg=none token", w.Code)
	}
}

func TestFundRequireRoleEnforcesRole(t *testing.T) {
	const secret = "test-secret"
	t.Setenv("PROTECTION_FUND_JWT_SECRET", secret)

	r := newFundRouter(secret)

	for _, tc := range []struct {
		role string
		want int
	}{
		{"operator", http.StatusForbidden},
		{"admin", http.StatusOK},
	} {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": "u1",
			"role":    tc.role,
			"exp":     time.Now().Add(time.Hour).Unix(),
		})
		signed, _ := token.SignedString([]byte(secret))

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		r.ServeHTTP(w, req)

		if w.Code != tc.want {
			t.Fatalf("role %q: status = %d, want %d", tc.role, w.Code, tc.want)
		}
	}
}

func TestFundAuthRejectsTokenWithoutUser(t *testing.T) {
	const secret = "test-secret"
	t.Setenv("PROTECTION_FUND_JWT_SECRET", secret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))

	r := newFundRouter(secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a token with no user", w.Code)
	}
}
