package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// LicenseGate is the client the master-wallet backend uses to verify the
// two-party SuperAdmin co-sign before broadcasting any fund/revenue withdrawal.
//
// Requirement: "No one can withdraw any fund or revenue without TigerWallet
// SuperAdmin collaboration." This gate enforces it at the broadcast boundary —
// the LAST point before funds move — so even a compromised WL admin key cannot
// move funds alone.
//
// The gate is fail-closed: if the control plane URL is unset or unreachable,
// the withdrawal is REFUSED (never permitted without the gate).
type LicenseGate struct {
	cpURL  string
	token  string
	client *http.Client
}

func NewLicenseGate() *LicenseGate {
	return &LicenseGate{
		cpURL:  os.Getenv("TWO_PARTY_GATE_URL"),
		token:  os.Getenv("TWO_PARTY_GATE_TOKEN"),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// IsWithdrawalApproved checks the control plane for a two-party-approved
// withdrawal. Returns true ONLY when both the WL client AND SuperAdmin have
// approved. Fail-closed on any error.
func (g *LicenseGate) IsWithdrawalApproved(ctx context.Context, withdrawalID uuid.UUID) bool {
	if g.cpURL == "" {
		return false // fail-closed: no gate configured => no payout
	}
	url := fmt.Sprintf("%s/api/v1/super-admin/withdrawals/%s/approved", g.cpURL, withdrawalID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// RequestWithdrawal asks the control plane to create a two-party withdrawal
// request (WL-side). The WL client approves via the WL admin panel; SuperAdmin
// approves via the control plane. Only then does IsWithdrawalApproved return true.
func (g *LicenseGate) RequestWithdrawal(ctx context.Context, walletID uuid.UUID, toAddress, amountWei, currency string, chainID int64) (uuid.UUID, error) {
	if g.cpURL == "" {
		return uuid.Nil, fmt.Errorf("two-party gate not configured; cannot request withdrawal")
	}
	body := map[string]any{
		"wallet_id":  walletID,
		"to_address": toAddress,
		"amount_wei": amountWei,
		"currency":   currency,
		"chain_id":   chainID,
	}
	b, _ := json.Marshal(body)
	url := g.cpURL + "/api/v1/wl/withdrawals/request"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return uuid.Nil, fmt.Errorf("control plane rejected withdrawal request (HTTP %d)", resp.StatusCode)
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
func (g *LicenseGate) MarkWithdrawalExecuted(ctx context.Context, withdrawalID uuid.UUID, txHash string) error {
	if g.cpURL == "" {
		return nil // best-effort; the gate check already passed
	}
	body := map[string]any{"tx_hash": txHash}
	b, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/api/v1/super-admin/withdrawals/%s/executed", g.cpURL, withdrawalID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
