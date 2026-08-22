// two_party.go — the Manual-mode two-party withdrawal gate for the standalone
// WL-UserWallet backend.
//
// When the AutoApprover classifier returns ModeManual (fee/revenue/treasury
// withdrawal), the fund-moving handler MUST call IsWithdrawalApproved with a
// withdrawal_id. The control plane (license_service) is the authoritative
// approver: it records the WL-side + SuperAdmin co-sign. Fail-closed: if the
// control plane is unreachable or the id is not approved => REFUSE.
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TwoPartyGate verifies a withdrawal_id against the control plane.
type TwoPartyGate struct {
	cpURL      string
	token      string
	product    string
	wlClientID string
	client     *http.Client
}

var (
	twoPartyOnce sync.Once
	twoPartyGate *TwoPartyGate
)

// InitTwoPartyGate configures the singleton two-party gate. Called once from
// main() with the control plane URL + WL service token + product + wlClientID
// (the latter two populate the X-Service-Product / X-WL-Client-ID headers the
// license_service ServiceOrUserAuth middleware requires for machine calls).
func InitTwoPartyGate(cpURL, token, product, wlClientID string) {
	twoPartyOnce.Do(func() {
		twoPartyGate = &TwoPartyGate{
			cpURL:      cpURL,
			token:      token,
			product:    product,
			wlClientID: wlClientID,
			client:     &http.Client{Timeout: 5 * time.Second},
		}
	})
}

// TwoPartyGate returns the singleton (nil if not configured).
func GetTwoPartyGate() *TwoPartyGate { return twoPartyGate }

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

// IsWithdrawalApproved returns true ONLY when the control plane confirms both
// the WL client AND SuperAdmin co-approved the withdrawal. Fail-closed: any
// error (unreachable, unconfigured, not-found, not-approved) => false.
func (t *TwoPartyGate) IsWithdrawalApproved(ctx context.Context, withdrawalID uuid.UUID) bool {
	if t.cpURL == "" || withdrawalID == uuid.Nil {
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
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var out struct {
		Approved bool `json:"approved"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false
	}
	return out.Approved
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
