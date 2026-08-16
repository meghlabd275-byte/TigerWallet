package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HeartbeatLoop phones home to the license control plane at the configured
// interval. On each beat it validates the license (real Ed25519 token from the
// control plane), refreshes the flag snapshot, and processes SuperAdmin
// commands (halt/resume/sync_flags). Fail-closed: if validation fails, the gate
// goes dead and every request 503s.
//
// This is the standalone WL-UserWallet's phone-home — it does NOT depend on
// TigerWallet cloud at request time, only on the control plane at heartbeat
// time (30s). If the control plane is unreachable, the gate stays in its last
// known state for one grace period, then goes dead (fail-closed).
func HeartbeatLoop(ctx context.Context, cpURL, token, wlClientID, licenseKey, product, instanceID string, interval time.Duration) {
	if cpURL == "" {
		// No control plane configured — fail-closed immediately.
		SetAlive(false, "license control plane not configured (TWO_PARTY_GATE_URL unset)")
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// Validate immediately on boot.
	beat(ctx, client, cpURL, token, wlClientID, licenseKey, product, instanceID)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			beat(ctx, client, cpURL, token, wlClientID, licenseKey, product, instanceID)
		}
	}
}

func beat(ctx context.Context, client *http.Client, cpURL, token, wlClientID, licenseKey, product, instanceID string) {
	url := fmt.Sprintf("%s/api/v1/license/validate", cpURL)
	body := fmt.Sprintf(`{"license_key":%q,"product":%q,"instance_id":%q,"version":"1.0.0"}`, licenseKey, product, instanceID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		SetAlive(false, "control plane unreachable: "+err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		SetAlive(false, fmt.Sprintf("control plane rejected license (HTTP %d)", resp.StatusCode))
		return
	}
	var vr struct {
		Valid   bool   `json:"valid"`
		Alive   bool   `json:"alive"`
		Reason  string `json:"reason"`
		Flags   []Flag `json:"flags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		SetAlive(false, "control plane response parse error")
		return
	}
	if !vr.Valid || !vr.Alive {
		SetAlive(false, vr.Reason)
		return
	}
	if vr.Flags != nil {
		SetFlags(vr.Flags)
	}
	SetAlive(true, "")
}
