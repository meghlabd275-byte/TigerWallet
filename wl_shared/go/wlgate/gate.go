// Package wlgate provides the fail-closed license gate + heartbeat phone-home
// client shared by ALL standalone white-label backends (WL-MasterWallet,
// WL-UserWallet, WL-Bots, WL-ProjectParty). This is the pure-Go mirror of the
// C++ ultra-low-latency WlGate (wl_control_plane/cpp) — used by Go backends
// that have no cgo dependency. The C++ gate remains the hot-path checker for
// C++/Rust services; Go services use this.
//
// Semantics (identical to the C++ gate):
//   - wait-free atomic `alive` flag (std::atomic in C++, sync/atomic here)
//   - shared-mutex-protected flag map: product\x1ffetcher -> enabled
//   - product\x1f* disables the whole product
//   - fail-closed: any heartbeat failure or unreachable control plane => dead
//
// License: fail-closed. No WL product serves a single request without a valid
// license validated against the TigerWallet SuperAdmin control plane.
package wlgate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Gate is the in-process license gate (mirrors wl_control_plane/cpp WlGate).
type Gate struct {
	alive  atomic.Bool
	mu     sync.RWMutex
	reason string
	flags  map[string]bool // product\x1ffetcher -> enabled
	auto   *AutoApprover   // optional: the transaction classifier (fee/revenue => Manual)
}

// New returns a Gate initialized to dead (fail-closed on boot until first
// successful heartbeat).
func New() *Gate {
	return &Gate{
		flags:  map[string]bool{},
		reason: "license not yet validated (heartbeat pending)",
	}
}

// WithAutoApprover attaches an AutoApprover so the heartbeat also pushes the
// policy snapshot (treasury addresses + auto-sign rules) into it. Returns the
// gate for chaining: gate := wlgate.New().WithAutoApprover(wlgate.NewAutoApprover()).
func (g *Gate) WithAutoApprover(a *AutoApprover) *Gate {
	g.auto = a
	return g
}

// AutoApprover returns the attached AutoApprover (nil if none).
func (g *Gate) AutoApprover() *AutoApprover { return g.auto }

// SetAlive sets the liveness flag.
func (g *Gate) SetAlive(alive bool, reason string) {
	g.alive.Store(alive)
	g.mu.Lock()
	if alive {
		g.reason = ""
	} else {
		g.reason = reason
	}
	g.mu.Unlock()
}

// Flag represents a single feature flag (product + fetcher granularity).
type Flag struct {
	Product string `json:"product"`
	Fetcher string `json:"fetcher"`
	Enabled bool   `json:"enabled"`
}

// SetFlags pushes a flag snapshot.
func (g *Gate) SetFlags(flags []Flag) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.flags = map[string]bool{}
	for _, f := range flags {
		g.flags[f.Product+"\x1f"+f.Fetcher] = f.Enabled
	}
}

// IsAlive returns the liveness flag.
func (g *Gate) IsAlive() bool { return g.alive.Load() }

// Reason returns the fail-closed reason.
func (g *Gate) Reason() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.reason
}

// FetcherEnabled checks the per-fetcher gate.
func (g *Gate) FetcherEnabled(product, fetcher string) bool {
	if !g.IsAlive() {
		return false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if en, ok := g.flags[product+"\x1f*"]; ok && !en {
		return false
	}
	if en, ok := g.flags[product+"\x1f"+fetcher]; ok && !en {
		return false
	}
	return true
}

// Claims for the WL backend JWT (tenant-scoped).
type Claims struct {
	UserID       uuid.UUID `json:"user_id"`
	Email        string    `json:"email"`
	WhiteLabelID uuid.UUID `json:"white_label_id"`
	Scopes       []string  `json:"scopes"`
	jwt.RegisteredClaims
}

// JWTAuth is the JWT auth middleware (real HS256).
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Authorization header required"})
			return
		}
		parts := strings.Split(h, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid header"})
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("white_label_id", claims.WhiteLabelID)
		c.Set("scopes", claims.Scopes)
		c.Next()
	}
}

// UserID extracts the caller's user id from context.
func UserID(c *gin.Context) uuid.UUID {
	if v, ok := c.Get("user_id"); ok {
		return v.(uuid.UUID)
	}
	return uuid.Nil
}

// HasScope checks if the caller has a specific scope.
func HasScope(c *gin.Context, scope string) bool {
	if v, ok := c.Get("scopes"); ok {
		scopes := v.([]string)
		for _, s := range scopes {
			if s == scope || s == "wl_client" {
				return true
			}
		}
	}
	return false
}

// RequireScope is middleware that enforces a specific scope.
func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasScope(c, scope) {
			c.AbortWithStatusJSON(403, gin.H{"error": "insufficient scope: " + scope})
			return
		}
		c.Next()
	}
}

// Gate is the license-gate middleware. Fail-closeds (503) when the product is
// not alive or a fetcher is disabled by SuperAdmin.
func (g *Gate) Middleware(product string, fetcherForPath func(string) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !g.IsAlive() {
			c.AbortWithStatusJSON(503, gin.H{
				"error":  "product is not authorized to serve (license suspended/revoked or heartbeat stale)",
				"reason": g.Reason(),
			})
			return
		}
		fetcher := "*"
		if fetcherForPath != nil {
			if f := fetcherForPath(c.Request.URL.Path); f != "" {
				fetcher = f
			}
		}
		if !g.FetcherEnabled(product, fetcher) {
			c.AbortWithStatusJSON(403, gin.H{
				"error":   "fetcher disabled by SuperAdmin",
				"product": product,
				"fetcher": fetcher,
			})
			return
		}
		c.Next()
	}
}

// IssueJWT mints a JWT for a user.
func IssueJWT(secret string, userID uuid.UUID, email string, whiteLabelID uuid.UUID, scopes []string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID:       userID,
		Email:        email,
		WhiteLabelID: whiteLabelID,
		Scopes:       scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// SimpleFetcher derives the fetcher name from the last path segment.
func SimpleFetcher(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return "*"
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// HeartbeatLoop phones home to the license control plane at the configured
// interval. Fail-closed: if validation fails, the gate goes dead.
func (g *Gate) HeartbeatLoop(ctx context.Context, cpURL, token, licenseKey, product, instanceID string, interval time.Duration) {
	if cpURL == "" {
		g.SetAlive(false, "license control plane not configured (TWO_PARTY_GATE_URL unset)")
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	g.beat(ctx, client, cpURL, token, licenseKey, product, instanceID)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.beat(ctx, client, cpURL, token, licenseKey, product, instanceID)
		}
	}
}

func (g *Gate) beat(ctx context.Context, client *http.Client, cpURL, token, licenseKey, product, instanceID string) {
	url := fmt.Sprintf("%s/api/v1/license/validate", cpURL)
	body := fmt.Sprintf(`{"license_key":%q,"product":%q,"instance_id":%q,"version":"1.0.0"}`, licenseKey, product, instanceID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		g.SetAlive(false, "control plane unreachable: "+err.Error())
		if g.auto != nil {
			g.auto.SetAlive(false, "control plane unreachable: "+err.Error())
		}
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		g.SetAlive(false, fmt.Sprintf("control plane rejected license (HTTP %d)", resp.StatusCode))
		if g.auto != nil {
			g.auto.SetAlive(false, fmt.Sprintf("control plane rejected license (HTTP %d)", resp.StatusCode))
		}
		return
	}
	var vr struct {
		Valid             bool           `json:"valid"`
		Alive             bool           `json:"alive"`
		Reason            string         `json:"reason"`
		Flags             []Flag         `json:"flags"`
		TreasuryAddresses []string       `json:"treasury_addresses"`
		AutoSignRules     []AutoSignRule `json:"auto_sign_rules"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		g.SetAlive(false, "control plane response parse error")
		if g.auto != nil {
			g.auto.SetAlive(false, "control plane response parse error")
		}
		return
	}
	if !vr.Valid || !vr.Alive {
		g.SetAlive(false, vr.Reason)
		if g.auto != nil {
			g.auto.SetAlive(false, vr.Reason)
		}
		return
	}
	if vr.Flags != nil {
		g.SetFlags(vr.Flags)
	}
	// Push the AutoApprover policy snapshot (the security boundary that
	// defines fee/revenue => Manual, user tx => Auto).
	if g.auto != nil {
		if vr.TreasuryAddresses != nil {
			g.auto.SetTreasuryAddresses(vr.TreasuryAddresses)
		}
		if vr.AutoSignRules != nil {
			g.auto.SetRules(vr.AutoSignRules)
		}
		g.auto.SetAlive(true, "")
	}
	g.SetAlive(true, "")
}

// TwoPartyGate checks the control plane for a two-party-approved withdrawal.
// Fail-closed: if the control plane is unreachable or unconfigured, the
// withdrawal is REFUSED (no payout without SuperAdmin co-sign).
type TwoPartyGate struct {
	cpURL     string
	token     string
	product   string
	wlClientID string
	client    *http.Client
}

func NewTwoPartyGate(cpURL, token, product, wlClientID string) *TwoPartyGate {
	return &TwoPartyGate{
		cpURL:      cpURL,
		token:      token,
		product:    product,
		wlClientID: wlClientID,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// setServiceHeaders adds the service-to-service auth headers (SERVICE_AUTH_TOKEN
// bearer + X-Service-Product + X-WL-Client-ID) that the license_service
// ServiceOrUserAuth middleware accepts for machine-to-machine gate calls.
func (t *TwoPartyGate) setServiceHeaders(req *http.Request) {
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	if t.product != "" {
		req.Header.Set("X-Service-Product", t.product)
	}
	if t.wlClientID != "" {
		req.Header.Set("X-WL-Client-ID", t.wlClientID)
	}
}

// IsWithdrawalApproved returns true ONLY when both WL client + SuperAdmin approved.
func (t *TwoPartyGate) IsWithdrawalApproved(ctx context.Context, withdrawalID uuid.UUID) bool {
	if t.cpURL == "" {
		return false
	}
	url := fmt.Sprintf("%s/api/v1/withdrawals/%s/approved", t.cpURL, withdrawalID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	t.setServiceHeaders(req)
	resp, err := t.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// RequestWithdrawal creates a two-party withdrawal request (WL-side).
// Sends the full control-plane contract: {product, resource_type, resource_id,
// amount_wei, to_address, chain_id}. resource_type="wallet", resource_id=walletID.
func (t *TwoPartyGate) RequestWithdrawal(ctx context.Context, walletID uuid.UUID, toAddress, amountWei, currency string, chainID int64) (uuid.UUID, error) {
	if t.cpURL == "" {
		return uuid.Nil, fmt.Errorf("two-party gate not configured")
	}
	body := fmt.Sprintf(`{"product":%q,"resource_type":"wallet","resource_id":%q,"amount_wei":%q,"to_address":%q,"chain_id":%d}`,
		t.product, walletID, amountWei, toAddress, chainID)
	url := t.cpURL + "/api/v1/withdrawals/request"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	t.setServiceHeaders(req)
	resp, err := t.client.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		rb, _ := io.ReadAll(resp.Body)
		return uuid.Nil, fmt.Errorf("control plane rejected withdrawal request (HTTP %d): %s", resp.StatusCode, string(rb))
	}
	var out struct {
		WithdrawalID uuid.UUID `json:"withdrawal_id"`
		ID           uuid.UUID `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return uuid.Nil, err
	}
	if out.WithdrawalID != uuid.Nil {
		return out.WithdrawalID, nil
	}
	return out.ID, nil
}

// MarkWithdrawalExecuted records the on-chain tx hash after a gated broadcast.
func (t *TwoPartyGate) MarkWithdrawalExecuted(ctx context.Context, withdrawalID uuid.UUID, txHash string) error {
	if t.cpURL == "" {
		return nil
	}
	body := fmt.Sprintf(`{"tx_hash":%q}`, txHash)
	url := fmt.Sprintf("%s/api/v1/withdrawals/%s/executed", t.cpURL, withdrawalID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	t.setServiceHeaders(req)
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
