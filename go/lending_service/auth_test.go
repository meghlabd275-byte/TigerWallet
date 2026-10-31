package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "lending-test-secret"

func signToken(t *testing.T, sub, role string) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	t.Setenv("JWT_SECRET", testSecret)
	gin.SetMode(gin.TestMode)
	ls := &LendingService{} // no database: ownership checks still apply
	r := gin.New()
	v1 := r.Group("/api/v1/lending")
	v1.Use(requireCaller())
	{
		v1.GET("/markets", ls.GetMarkets)
		v1.GET("/position", ls.GetUserPosition)
		v1.GET("/positions", ls.GetUserPositions)
		v1.POST("/supply", ls.Supply)
		v1.POST("/borrow", ls.Borrow)
		v1.POST("/withdraw", ls.Withdraw)
		v1.POST("/repay", ls.Repay)
	}
	return r
}

func call(r *gin.Engine, method, path, body, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLendingRequiresToken(t *testing.T) {
	r := testRouter(t)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/lending/supply", `{"user_address":"0xabc","asset_address":"0xdef","amount":"1"}`},
		{http.MethodPost, "/api/v1/lending/withdraw", `{"user_address":"0xabc","asset_address":"0xdef","amount":"1"}`},
		{http.MethodPost, "/api/v1/lending/borrow", `{"user_address":"0xabc","asset_address":"0xdef","amount":"1"}`},
		{http.MethodPost, "/api/v1/lending/repay", `{"user_address":"0xabc","asset_address":"0xdef","amount":"1"}`},
		{http.MethodGet, "/api/v1/lending/positions?user_id=victim", ""},
		{http.MethodGet, "/api/v1/lending/position?user_address=0xabc", ""},
	} {
		if w := call(r, tc.method, tc.path, tc.body, ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: got %d, want 401", tc.method, tc.path, w.Code)
		}
	}
}

// A caller must not be able to open or move a position for someone else's
// address. Before this check the body address was used verbatim.
func TestLendingRejectsForeignAddress(t *testing.T) {
	r := testRouter(t)
	bearer := signToken(t, "attacker", "user")
	body := `{"user_address":"0xVictimWallet","asset_address":"0xdef","amount":"1"}`
	for _, p := range []string{"/api/v1/lending/supply", "/api/v1/lending/withdraw",
		"/api/v1/lending/borrow", "/api/v1/lending/repay"} {
		w := call(r, http.MethodPost, p, body, bearer)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s with foreign address: got %d, want 403 (%s)", p, w.Code, w.Body.String())
		}
	}
}

func TestLendingPositionsRejectsForeignAddress(t *testing.T) {
	r := testRouter(t)
	bearer := signToken(t, "attacker", "user")
	w := call(r, http.MethodGet, "/api/v1/lending/positions?user_id=0xVictimWallet", "", bearer)
	if w.Code != http.StatusForbidden {
		t.Fatalf("foreign positions: got %d, want 403", w.Code)
	}
}

func TestLendingNoWalletFailsClosed(t *testing.T) {
	r := testRouter(t)
	bearer := signToken(t, "user-without-wallet", "user")
	w := call(r, http.MethodGet, "/api/v1/lending/positions", "", bearer)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("no wallet: got %d, want 400", w.Code)
	}
}

func TestLendingWrongSecretRejected(t *testing.T) {
	r := testRouter(t)
	forged, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "victim",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("some-other-secret"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if w := call(r, http.MethodGet, "/api/v1/lending/positions", "", forged); w.Code != http.StatusUnauthorized {
		t.Fatalf("forged token: got %d, want 401", w.Code)
	}
}
