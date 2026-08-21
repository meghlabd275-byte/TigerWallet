// policy_snapshot.go — store methods for the AutoApprover policy snapshot.
//
// Treasury addresses + auto-sign rules are SuperAdmin-configured policies that
// get pushed into the WL product's AutoApprover on each heartbeat (alongside
// the feature flags). They define the security boundary:
//   - treasury_addresses: any tx to one of these => MANUAL two-party mode
//   - auto_sign_rules: block rules can deny an auto-approve even when licensed
package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// TreasuryAddress is a fee/revenue/treasury destination address.
type TreasuryAddress struct {
	ID         uuid.UUID `json:"id"`
	WLClientID uuid.UUID `json:"wl_client_id"`
	Product    string    `json:"product"`
	Address    string    `json:"address"` // lowercase hex, no 0x prefix
	Label      string    `json:"label"`
	CreatedAt  string    `json:"created_at"`
}

// AutoSignRule is a SuperAdmin policy that can block an auto-approve.
type AutoSignRule struct {
	ID         uuid.UUID `json:"id"`
	WLClientID uuid.UUID `json:"wl_client_id"`
	Product    string    `json:"product"`
	Fetcher    string    `json:"fetcher"`
	TxType     string    `json:"tx_type"`
	Token      string    `json:"token"`
	MaxAmount  string    `json:"max_amount"`
	Block      bool      `json:"block"`
}

// AddTreasuryAddress marks an address as a fee/revenue/treasury destination.
// Normalizes to lowercase hex without the 0x prefix for storage + matching.
func (s *Store) AddTreasuryAddress(ctx context.Context, wlClientID uuid.UUID, product, address, label string, adminID *uuid.UUID) (*TreasuryAddress, error) {
	addr := strings.ToLower(strings.TrimPrefix(address, "0x"))
	var out TreasuryAddress
	err := s.db.QueryRow(ctx,
		`INSERT INTO treasury_addresses (wl_client_id, product, address, label, created_by)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (wl_client_id, product, address) DO UPDATE SET label = EXCLUDED.label
		 RETURNING id, wl_client_id, product, address, label, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		wlClientID, product, addr, label, adminID).
		Scan(&out.ID, &out.WLClientID, &out.Product, &out.Address, &out.Label, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("add treasury address: %w", err)
	}
	return &out, nil
}

// ListTreasuryAddresses returns all treasury addresses for a WL client + product.
func (s *Store) ListTreasuryAddresses(ctx context.Context, wlClientID uuid.UUID, product string) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT address FROM treasury_addresses WHERE wl_client_id=$1 AND product=$2`,
		wlClientID, product)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DeleteTreasuryAddress removes a treasury address.
func (s *Store) DeleteTreasuryAddress(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM treasury_addresses WHERE id=$1`, id)
	return err
}

// SetAutoSignRule creates or updates an auto-sign rule.
func (s *Store) SetAutoSignRule(ctx context.Context, wlClientID uuid.UUID, product, fetcher, txType, token, maxAmount string, block bool, adminID *uuid.UUID) (*AutoSignRule, error) {
	var out AutoSignRule
	err := s.db.QueryRow(ctx,
		`INSERT INTO auto_sign_rules (wl_client_id, product, fetcher, tx_type, token, max_amount, block, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (wl_client_id, product, fetcher, tx_type, token)
		 DO UPDATE SET max_amount=EXCLUDED.max_amount, block=EXCLUDED.block, updated_at=NOW()
		 RETURNING id, wl_client_id, product, fetcher, tx_type, token, max_amount, block`,
		wlClientID, product, fetcher, txType, token, maxAmount, block, adminID).
		Scan(&out.ID, &out.WLClientID, &out.Product, &out.Fetcher, &out.TxType, &out.Token, &out.MaxAmount, &out.Block)
	if err != nil {
		return nil, fmt.Errorf("set auto-sign rule: %w", err)
	}
	return &out, nil
}

// ListAutoSignRules returns all auto-sign rules for a WL client + product.
func (s *Store) ListAutoSignRules(ctx context.Context, wlClientID uuid.UUID, product string) ([]*AutoSignRule, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, wl_client_id, product, fetcher, tx_type, token, max_amount, block
		 FROM auto_sign_rules WHERE wl_client_id=$1 AND product=$2`,
		wlClientID, product)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*AutoSignRule
	for rows.Next() {
		r := &AutoSignRule{}
		if err := rows.Scan(&r.ID, &r.WLClientID, &r.Product, &r.Fetcher, &r.TxType, &r.Token, &r.MaxAmount, &r.Block); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DeleteAutoSignRule removes an auto-sign rule.
func (s *Store) DeleteAutoSignRule(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM auto_sign_rules WHERE id=$1`, id)
	return err
}
