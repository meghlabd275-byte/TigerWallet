package main

import (
	"context"
	"crypto/rand"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Every statement below is executed, not merely prepared, because several of
// these defects only surface when PostgreSQL has to infer a parameter type or
// coerce an argument into a column.
func TestStatementsExecuteAgainstMigratedSchema(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()

	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer admin.Close()

	const dbName = "mw_stmt_check"
	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS `+dbName); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+dbName); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), `DROP DATABASE IF EXISTS `+dbName)
	})

	pool, err := pgxpool.New(ctx, replaceDatabase(dsn, dbName))
	if err != nil {
		t.Fatalf("connect scoped: %v", err)
	}
	defer pool.Close()
	for _, ext := range []string{"pgcrypto", `"uuid-ossp"`, "citext"} {
		if _, err := pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS "+ext); err != nil {
			t.Fatalf("extension %s: %v", ext, err)
		}
	}
	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	walletID, userID := seedWallet(t, ctx, pool)

	t.Run("sign and sub-wallet transfers persist", func(t *testing.T) {
		// Both statements list chain_id as the eleventh column. With only nine
		// placeholders the INSERT is rejected outright, so the transaction the
		// chain already accepted is never recorded.
		if _, err := pool.Exec(ctx, sqlInsertSignedTx,
			uuidStr(), walletID, "0xdead", "ethereum", "0xfrom", "0xto", "1", "ETH", 1); err != nil {
			t.Fatalf("insert: %v", err)
		}
		subID := uuidStr()
		if _, err := pool.Exec(ctx,
			`INSERT INTO sub_wallets (id, master_wallet_id, name, derivation_path, address) VALUES ($1,$2,'sub',$3,'0xsub')`,
			subID, walletID, "m/44'/60'/0'/0/1"); err != nil {
			t.Fatalf("sub wallet: %v", err)
		}
		if _, err := pool.Exec(ctx, sqlInsertSubWalletSignedTx,
			uuidStr(), walletID, subID, "0xbeef", "ethereum", "0xfrom", "0xto", "1", "ETH", 1); err != nil {
			t.Fatalf("sub insert: %v", err)
		}
	})

	t.Run("auto-sign failure path records attempts", func(t *testing.T) {
		txID := uuidStr()
		if _, err := pool.Exec(ctx,
			`INSERT INTO transactions (id, master_wallet_id, tx_hash, tx_type, status, blockchain, from_address, to_address, amount, token_symbol, chain_id)
			 VALUES ($1,$2,'','transfer','approved','ethereum','0xfrom','0xto','1','ETH',1)`,
			txID, walletID); err != nil {
			t.Fatalf("seed tx: %v", err)
		}
		// requeueOrFail: jsonb_build_object receives a Go int. Without the cast
		// the server cannot infer the parameter type and the UPDATE is rejected,
		// leaving the transaction stuck at 'approved' with no recorded attempt.
		if _, err := pool.Exec(ctx, sqlAutoSignFail, txID, "rpc down", 3); err != nil {
			t.Fatalf("fail path: %v", err)
		}
		if _, err := pool.Exec(ctx, sqlAutoSignRequeue, txID, "rpc down", 3, "30s"); err != nil {
			t.Fatalf("requeue path: %v", err)
		}
		var attempts int
		if err := pool.QueryRow(ctx, sqlAutoSignAttempts, txID).Scan(&attempts); err != nil {
			t.Fatalf("read attempts: %v", err)
		}
		if attempts != 3 {
			t.Fatalf("attempts = %d, want 3", attempts)
		}
	})

	t.Run("fee and rule updates accept numeric input", func(t *testing.T) {
		feeID := uuidStr()
		if _, err := pool.Exec(ctx,
			`INSERT INTO fee_config (id, master_wallet_id, fee_type, fee_percentage, fee_fixed)
			 VALUES ($1,$2,'swap',0.5,0)`, feeID, walletID); err != nil {
			t.Fatalf("seed fee: %v", err)
		}
		// fee_fixed arrives as a Go string; NULLIF yields text, and text cannot be
		// COALESCEd with the numeric column.
		if _, err := pool.Exec(ctx, sqlUpdateFeeConfig,
			"swap", 0.75, "100", true, feeID, walletID); err != nil {
			t.Fatalf("fee update: %v", err)
		}
		var pct float64
		var fixed string
		if err := pool.QueryRow(ctx, `SELECT fee_percentage, fee_fixed FROM fee_config WHERE id=$1`, feeID).
			Scan(&pct, &fixed); err != nil {
			t.Fatalf("read fee: %v", err)
		}
		if pct != 0.75 || fixed != "100" {
			t.Fatalf("fee = %v/%s, want 0.75/100", pct, fixed)
		}

		ruleID := uuidStr()
		if _, err := pool.Exec(ctx,
			`INSERT INTO auto_sign_rules (id, master_wallet_id, name, rule_type, max_amount)
			 VALUES ($1,$2,'r','any',0)`, ruleID, walletID); err != nil {
			t.Fatalf("seed rule: %v", err)
		}
		if _, err := pool.Exec(ctx, sqlUpdateAutoSignRule,
			"r", "any", []byte(`{"a":1}`), "5000", false, ruleID, walletID); err != nil {
			t.Fatalf("rule update: %v", err)
		}
	})

	t.Run("approval request carries the wallet threshold", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE master_wallets SET threshold=2 WHERE id=$1`, walletID); err != nil {
			t.Fatalf("set threshold: %v", err)
		}
		var threshold int
		if err := pool.QueryRow(ctx, `SELECT threshold FROM master_wallets WHERE id=$1`, walletID).Scan(&threshold); err != nil {
			t.Fatalf("read threshold: %v", err)
		}
		txID := uuidStr()
		if _, err := pool.Exec(ctx,
			`INSERT INTO transactions (id, master_wallet_id, tx_hash, tx_type, status, blockchain, from_address, to_address, amount, token_symbol, chain_id)
			 VALUES ($1,$2,'','transfer','pending','ethereum','0xfrom','0xto','1','ETH',1)`,
			txID, walletID); err != nil {
			t.Fatalf("seed tx: %v", err)
		}
		if _, err := pool.Exec(ctx, sqlInsertApprovalRequest, txID, threshold, "{}"); err != nil {
			t.Fatalf("approval insert: %v", err)
		}
		var required int
		if err := pool.QueryRow(ctx, sqlApprovalThreshold, txID, walletID).Scan(&required); err != nil {
			t.Fatalf("threshold read: %v", err)
		}
		if required != 2 {
			t.Fatalf("required = %d, want 2", required)
		}
	})

	t.Run("auto-sign attribution requires a signer", func(t *testing.T) {
		// CreateMasterWallet registers the creator; the daemon resolves that row
		// to attribute the approval. Without it there is nothing to sign as.
		var signerID string
		if err := pool.QueryRow(ctx, sqlActiveSignerForWallet, walletID).Scan(&signerID); err != nil {
			t.Fatalf("signer lookup: %v", err)
		}
		txID := uuidStr()
		if _, err := pool.Exec(ctx,
			`INSERT INTO transactions (id, master_wallet_id, tx_hash, tx_type, status, blockchain, from_address, to_address, amount, token_symbol, chain_id)
			 VALUES ($1,$2,'','transfer','approved','ethereum','0xfrom','0xto','1','ETH',1)`,
			txID, walletID); err != nil {
			t.Fatalf("seed tx: %v", err)
		}
		if _, err := pool.Exec(ctx, sqlRecordSignature, txID, signerID); err != nil {
			t.Fatalf("signature insert: %v", err)
		}
		var count int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM transaction_signatures WHERE transaction_id = $1 AND signature_status = 'signed'`,
			txID).Scan(&count); err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 1 {
			t.Fatalf("signed rows = %d, want 1", count)
		}
		_ = userID
	})
}

func seedWallet(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (walletID, userID string) {
	t.Helper()
	walletID, userID = uuidStr(), uuidStr()
	if _, err := pool.Exec(ctx,
		`INSERT INTO master_wallets (id, name, address, encrypted_seed, created_by)
		 VALUES ($1,'w',$2,'seed',$3)`, walletID, "0x"+walletID[:16], userID); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO mw_users (id, email, password_hash) VALUES ($1,$2,'x')`,
		userID, userID[:8]+"@example.com"); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// Mirrors the owner signer CreateMasterWallet registers.
	if _, err := pool.Exec(ctx,
		`INSERT INTO signers (master_wallet_id, user_id, label, signer_type, address)
		 VALUES ($1,$2,'Owner','owner','0xowner')`, walletID, userID); err != nil {
		t.Fatalf("seed signer: %v", err)
	}
	return walletID, userID
}

func uuidStr() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	j := 0
	for i, v := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[j] = '-'
			j++
		}
		out[j] = hex[v>>4]
		out[j+1] = hex[v&0x0f]
		j += 2
	}
	return string(out)
}
