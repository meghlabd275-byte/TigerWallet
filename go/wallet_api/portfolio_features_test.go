package main

// portfolio_features_test.go — executes the portfolio-feature statements against
// the real migrated schema. Each statement is PREPAREd, which makes PostgreSQL
// resolve every target column, table and operator; a column/placeholder
// mismatch (42601), an unknown relation (42P01) or an undefined operator (42883)
// therefore fails here instead of at request time.

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
)

func portfolioTestStore(t *testing.T) *Store {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	s, err := NewStore(ctx, dbURL, "127.0.0.1:1")
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(func() { s.PG.Close() })
	return s
}

func TestPortfolioStatementsPrepare(t *testing.T) {
	s := portfolioTestStore(t)
	ctx := context.Background()

	statements := map[string]string{
		"insert margin position":  sqlInsertMarginPosition,
		"upsert launchpool stake": sqlUpsertLaunchpoolStake,
		"insert dao vote": `INSERT INTO dao_votes (id, proposal_id, voter_id, choice, voting_power)
                 VALUES ($1,$2,$3,$4,$5) ON CONFLICT (proposal_id, voter_id) DO UPDATE SET choice=$4, voting_power=$5`,
		"unstake launchpool": `UPDATE launchpool_stakes SET unstaked=TRUE WHERE project_id=$1 AND user_id=$2 AND unstaked=FALSE`,
		"list launchpool stakes": `SELECT id, project_id, amount, rewards, start_time, unstaked
             FROM launchpool_stakes WHERE user_id=$1 AND unstaked=FALSE ORDER BY start_time DESC`,
	}
	for name, stmt := range statements {
		t.Run(name, func(t *testing.T) {
			if _, err := s.PG.Exec(ctx, "PREPARE stmt_probe AS "+stmt); err != nil {
				t.Fatalf("statement does not prepare: %v", err)
			}
			if _, err := s.PG.Exec(ctx, "DEALLOCATE stmt_probe"); err != nil {
				t.Fatalf("deallocate: %v", err)
			}
		})
	}
}

// TestMarginPositionInsertRoundTrip checks the placeholder count matches the
// target column list: the previous form listed 12 columns with 13 placeholders.
func TestMarginPositionInsertRoundTrip(t *testing.T) {
	s := portfolioTestStore(t)
	ctx := context.Background()

	var userID uuid.UUID
	if err := s.PG.QueryRow(ctx,
		`INSERT INTO users (id, email, username, password_hash)
		 VALUES ($1,$2,$3,$4) RETURNING id`,
		uuid.New(), "margin-"+uuid.NewString()+"@test.local", "marginprobe", "x").Scan(&userID); err != nil {
		t.Skipf("cannot create fixture user: %v", err)
	}
	t.Cleanup(func() { _, _ = s.PG.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID) })

	posID := uuid.New()
	if _, err := s.PG.Exec(ctx, sqlInsertMarginPosition,
		posID, userID, "PAIR-1", "ETH/USDT", "long", "100", "1000", "10", "3000", "2700", 1, time.Now().Unix()); err != nil {
		t.Fatalf("margin insert failed: %v", err)
	}

	var stored string
	if err := s.PG.QueryRow(ctx,
		`SELECT pair_symbol FROM margin_positions WHERE id=$1`, posID).Scan(&stored); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored != "ETH/USDT" {
		t.Fatalf("pair_symbol round-trip mismatch: got %q", stored)
	}
}

// TestLaunchpoolStakeAccumulatesAndResets covers the numeric cast (without it
// PostgreSQL raises 42883 numeric+text) and the re-stake reset.
func TestLaunchpoolStakeAccumulatesAndResets(t *testing.T) {
	s := portfolioTestStore(t)
	ctx := context.Background()

	var userID uuid.UUID
	if err := s.PG.QueryRow(ctx,
		`INSERT INTO users (id, email, username, password_hash)
		 VALUES ($1,$2,$3,$4) RETURNING id`,
		uuid.New(), "lp-"+uuid.NewString()+"@test.local", "lpprobe", "x").Scan(&userID); err != nil {
		t.Skipf("cannot create fixture user: %v", err)
	}
	t.Cleanup(func() { _, _ = s.PG.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID) })

	projectID := uuid.New()
	if _, err := s.PG.Exec(ctx,
		`INSERT INTO launchpool_projects (id, token_name, token_symbol, contract_address, chain_id, total_rewards, distributed_rewards, apy, start_time, end_time, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		projectID, "Probe Token", "PRB", "0x0000000000000000000000000000000000000000", 1, "1000", "0", "10",
		time.Now().Unix(), time.Now().Add(time.Hour).Unix(), "active"); err != nil {
		t.Skipf("cannot create fixture project: %v", err)
	}
	t.Cleanup(func() { _, _ = s.PG.Exec(ctx, `DELETE FROM launchpool_projects WHERE id=$1`, projectID) })

	stake := func(amount string) {
		t.Helper()
		if _, err := s.PG.Exec(ctx, sqlUpsertLaunchpoolStake,
			uuid.New(), projectID, userID, amount, time.Now().Unix()); err != nil {
			t.Fatalf("stake %s failed: %v", amount, err)
		}
	}
	amount := func() float64 {
		t.Helper()
		var got string
		if err := s.PG.QueryRow(ctx,
			`SELECT amount FROM launchpool_stakes WHERE project_id=$1 AND user_id=$2`,
			projectID, userID).Scan(&got); err != nil {
			t.Fatalf("read stake: %v", err)
		}
		f, err := strconv.ParseFloat(got, 64)
		if err != nil {
			t.Fatalf("stake amount %q is not numeric: %v", got, err)
		}
		return f
	}

	stake("100")
	if got := amount(); got != 100 {
		t.Fatalf("after first stake want 100, got %v", got)
	}
	stake("50")
	if got := amount(); got != 150 {
		t.Fatalf("second stake must accumulate to 150, got %v", got)
	}
	if _, err := s.PG.Exec(ctx,
		`UPDATE launchpool_stakes SET unstaked=TRUE WHERE project_id=$1 AND user_id=$2`,
		projectID, userID); err != nil {
		t.Fatalf("unstake: %v", err)
	}
	stake("25")
	if got := amount(); got != 25 {
		t.Fatalf("re-stake after unstake must reset to 25, got %v", got)
	}
}
