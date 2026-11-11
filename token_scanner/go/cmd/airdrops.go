package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AirdropStore holds the operator-curated airdrop campaigns and the per-wallet
// eligibility a claim is gated on.
//
// Airdrops are not invented here. A campaign is registered by an operator, and
// a wallet is only ever told it is eligible when the campaign's verifier has
// recorded that fact. A wallet with no eligibility row is reported as not
// eligible, never as eligible.
type AirdropStore struct {
	pool *pgxpool.Pool
}

const airdropSchema = `
CREATE TABLE IF NOT EXISTS airdrop_campaigns (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    symbol        TEXT NOT NULL,
    chain_id      TEXT NOT NULL DEFAULT '1',
    token_address TEXT NOT NULL DEFAULT '',
    claim_url     TEXT NOT NULL DEFAULT '',
    claim_amount  TEXT NOT NULL DEFAULT '',
    start_at      TIMESTAMPTZ NOT NULL,
    end_at        TIMESTAMPTZ NOT NULL,
    snapshot_at   TIMESTAMPTZ,
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS airdrop_eligibility (
    campaign_id  TEXT NOT NULL REFERENCES airdrop_campaigns(id) ON DELETE CASCADE,
    wallet       TEXT NOT NULL,
    claim_amount TEXT NOT NULL DEFAULT '',
    eligible     BOOLEAN NOT NULL DEFAULT FALSE,
    claimed      BOOLEAN NOT NULL DEFAULT FALSE,
    tx_hash      TEXT,
    checked_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (campaign_id, wallet)
);

CREATE INDEX IF NOT EXISTS airdrop_eligibility_wallet_idx
    ON airdrop_eligibility (wallet);
`

// NewAirdropStore connects to PostgreSQL and ensures the campaign schema exists.
func NewAirdropStore(ctx context.Context, databaseURL string) (*AirdropStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect airdrop store: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("airdrop store unreachable: %w", err)
	}
	if _, err := pool.Exec(ctx, airdropSchema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply airdrop schema: %w", err)
	}
	return &AirdropStore{pool: pool}, nil
}

// Close releases the connection pool.
func (a *AirdropStore) Close() {
	if a != nil && a.pool != nil {
		a.pool.Close()
	}
}

// ListForWallet returns every active campaign together with this wallet's real
// eligibility. A campaign the wallet has never been checked against is reported
// as not eligible rather than assumed eligible.
func (a *AirdropStore) ListForWallet(ctx context.Context, wallet string) ([]*Airdrop, error) {
	if a == nil || a.pool == nil {
		return nil, fmt.Errorf("airdrop store is not configured")
	}

	rows, err := a.pool.Query(ctx, `
		SELECT c.id, c.name, c.symbol, c.claim_url,
		       COALESCE(NULLIF(e.claim_amount, ''), c.claim_amount) AS claim_amount,
		       c.start_at, c.end_at, c.snapshot_at,
		       COALESCE(e.eligible, FALSE), COALESCE(e.claimed, FALSE)
		  FROM airdrop_campaigns c
		  LEFT JOIN airdrop_eligibility e
		         ON e.campaign_id = c.id AND e.wallet = $1
		 WHERE c.active = TRUE
		 ORDER BY c.start_at DESC`, wallet)
	if err != nil {
		return nil, fmt.Errorf("list airdrops: %w", err)
	}
	defer rows.Close()

	var out []*Airdrop
	for rows.Next() {
		var (
			campaign                                     Airdrop
			startAt, endAt                               time.Time
			snapshotAt                                   *time.Time
		)
		if err := rows.Scan(
			&campaign.ID, &campaign.Name, &campaign.Symbol, &campaign.ClaimURL,
			&campaign.ClaimAmount, &startAt, &endAt, &snapshotAt,
			&campaign.Eligible, &campaign.Claimed,
		); err != nil {
			return nil, fmt.Errorf("scan airdrop row: %w", err)
		}
		campaign.StartDate = startAt.Unix()
		campaign.EndDate = endAt.Unix()
		if snapshotAt != nil {
			campaign.SnapshotDate = snapshotAt.Unix()
		}
		out = append(out, &campaign)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate airdrops: %w", err)
	}
	return out, nil
}
