package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// pathSafeID and masterWalletIDPathSafe guard values that are concatenated into
// the upstream MasterWallet URL. A value containing "/" or "?" would retarget
// the request at a different endpoint, so an "approve" could be turned into a
// "send". These tests exercise the guard through a real gin router.
func newApprovalContext(t *testing.T, target string, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c
}

func TestPathSafeIDRejectsPathTraversal(t *testing.T) {
	const good = "9f1d6f0e-6a2b-4c3d-8e4f-1a2b3c4d5e6f"
	cases := []struct {
		raw   string
		valid bool
	}{
		{good, true},
		{"../admin", false},
		{"abc/def", false},
		{"abc?x=1", false},
		{"", false},
		{"approve/../send", false},
	}
	for _, tc := range cases {
		c := newApprovalContext(t, "/", "")
		got, ok := pathSafeID(c, tc.raw, "transaction id")
		if ok != tc.valid {
			t.Errorf("pathSafeID(%q) ok=%v, want %v", tc.raw, ok, tc.valid)
		}
		if tc.valid && got != tc.raw {
			t.Errorf("pathSafeID(%q) returned %q", tc.raw, got)
		}
		if !tc.valid && c.Writer.Status() != http.StatusBadRequest {
			t.Errorf("pathSafeID(%q) did not write 400 (status %d)", tc.raw, c.Writer.Status())
		}
	}
}

func TestMasterWalletIDPathSafeRejectsInjection(t *testing.T) {
	good := "9f1d6f0e-6a2b-4c3d-8e4f-1a2b3c4d5e6f"
	c := newApprovalContext(t, "/?master_wallet_id="+good, "")
	if _, ok := masterWalletIDPathSafe(c); !ok {
		t.Fatalf("valid uuid rejected: status %d", c.Writer.Status())
	}

	// "?" would end the path and start the query string; "/" would add a path
	// segment. Either could aim the upstream call at another endpoint.
	for _, evil := range []string{
		"x/transactions/1/approve",
		"x?a=b",
		"../../../etc/passwd",
	} {
		c := newApprovalContext(t, "/?master_wallet_id="+evil, "")
		if _, ok := masterWalletIDPathSafe(c); ok {
			t.Errorf("masterWalletIDPathSafe accepted %q", evil)
		}
		if c.Writer.Status() != http.StatusBadRequest {
			t.Errorf("masterWalletIDPathSafe(%q) did not write 400", evil)
		}
	}
}

// The MASTER_WALLET_ID environment variable is also interpolated into the
// upstream path, so it must pass the same check.
func TestMasterWalletIDFromEnvIsValidated(t *testing.T) {
	t.Setenv("MASTER_WALLET_ID", "not-a-uuid/../send")
	c := newApprovalContext(t, "/", "")
	if _, ok := masterWalletIDPathSafe(c); ok {
		t.Fatal("invalid MASTER_WALLET_ID from the environment was accepted")
	}
}
