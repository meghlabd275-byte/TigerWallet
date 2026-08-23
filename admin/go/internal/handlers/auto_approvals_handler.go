package handlers

// auto_approvals_handler.go — Admin approval surface for UserWallet
// transactions.
//
// Requirement: "UserWallet always gets automatic sign and automatic approval
// within a second from SuperAdmin or MasterWallet owner or Admin from admin
// panel."
//
// The MasterWallet backend (:8450) already auto-approves/auto-signs
// user-initiated transactions via its AutoSigner daemon within a second; these
// endpoints give the admin panel a real-time view of whatever is still pending
// (e.g. over the auto-sign cap, or a kind the owner disabled) plus a
// one-click approve/reject that hits the MasterWallet backend
// service-to-service:
//
//	GET  /api/v1/admin/auto-approvals/pending
//	POST /api/v1/admin/auto-approvals/:id/approve
//	POST /api/v1/admin/auto-approvals/:id/reject
//
// Config (env):
//	MASTER_WALLET_URL    base URL of the master wallet backend
//	                     (default http://localhost:8450)
//	SERVICE_AUTH_TOKEN   service JWT sent as Bearer to the master wallet backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AutoApprovalsHandler proxies the admin panel's approval actions to the
// canonical MasterWallet backend (service-to-service).
type AutoApprovalsHandler struct {
	client *http.Client
}

// NewAutoApprovalsHandler creates the handler. Upstream config is resolved
// per-request so runtime env changes take effect without a restart.
func NewAutoApprovalsHandler() *AutoApprovalsHandler {
	return &AutoApprovalsHandler{client: &http.Client{Timeout: 10 * time.Second}}
}

func masterWalletURL() string {
	if v := strings.TrimSpace(os.Getenv("MASTER_WALLET_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:8450"
}

// forward calls the MasterWallet backend with the service token and streams
// the upstream status/body back to the admin panel transparently.
func (h *AutoApprovalsHandler) forward(c *gin.Context, method, path string, body []byte) {
	url := masterWalletURL() + path
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), method, url, rdr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build upstream request"})
		return
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := strings.TrimSpace(os.Getenv("SERVICE_AUTH_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "master wallet backend unavailable", "detail": err.Error()})
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read master wallet response"})
		return
	}
	c.Data(resp.StatusCode, "application/json", respBody)
}

// masterWalletID resolves the target master wallet from the query string or
// JSON body (the admin panel may manage several master wallets).
func masterWalletID(c *gin.Context) string {
	if id := strings.TrimSpace(c.Query("master_wallet_id")); id != "" {
		return id
	}
	if id := strings.TrimSpace(os.Getenv("MASTER_WALLET_ID")); id != "" {
		return id
	}
	var body struct {
		MasterWalletID string `json:"master_wallet_id"`
	}
	if c.Request.Body != nil {
		raw, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		_ = json.Unmarshal(raw, &body)
	}
	return strings.TrimSpace(body.MasterWalletID)
}

// ListPending GET /api/v1/admin/auto-approvals/pending?master_wallet_id=<id>
// Lists pending user transactions awaiting approval on the MasterWallet.
func (h *AutoApprovalsHandler) ListPending(c *gin.Context) {
	mwID := masterWalletID(c)
	if mwID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "master_wallet_id is required (query param or MASTER_WALLET_ID env)"})
		return
	}
	h.forward(c, http.MethodGet,
		fmt.Sprintf("/api/v1/master-wallet/%s/transactions?master_wallet_id=%s&status=pending", mwID, mwID), nil)
}

// Approve POST /api/v1/admin/auto-approvals/:id/approve?master_wallet_id=<id>
// Approves a pending user transaction on the MasterWallet (counts toward its
// approval threshold and resolves its approval request).
func (h *AutoApprovalsHandler) Approve(c *gin.Context) {
	mwID := masterWalletID(c)
	if mwID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "master_wallet_id is required (query param or MASTER_WALLET_ID env)"})
		return
	}
	txID := c.Param("id")
	h.forward(c, http.MethodPost,
		fmt.Sprintf("/api/v1/master-wallet/%s/transactions/%s/approve", mwID, txID), nil)
}

// Reject POST /api/v1/admin/auto-approvals/:id/reject?master_wallet_id=<id>
// Rejects a pending user transaction; an optional {"reason": "..."} body is
// forwarded as the reject reason.
func (h *AutoApprovalsHandler) Reject(c *gin.Context) {
	mwID := masterWalletID(c)
	if mwID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "master_wallet_id is required (query param or MASTER_WALLET_ID env)"})
		return
	}
	txID := c.Param("id")
	reason := strings.TrimSpace(c.Query("reason"))
	if reason == "" && c.Request.Body != nil {
		var body struct {
			Reason string `json:"reason"`
		}
		raw, _ := io.ReadAll(c.Request.Body)
		_ = json.Unmarshal(raw, &body)
		reason = strings.TrimSpace(body.Reason)
	}
	path := fmt.Sprintf("/api/v1/master-wallet/%s/transactions/%s/reject", mwID, txID)
	if reason != "" {
		path += "?reason=" + urlQueryEscape(reason)
	}
	h.forward(c, http.MethodPost, path, nil)
}

// urlQueryEscape escapes a value for use in a query string.
func urlQueryEscape(s string) string {
	replacer := strings.NewReplacer(
		"%", "%25", " ", "%20", "&", "%26", "?", "%3F", "=", "%3D",
		"#", "%23", "+", "%2B", "/", "%2F",
	)
	return replacer.Replace(s)
}
