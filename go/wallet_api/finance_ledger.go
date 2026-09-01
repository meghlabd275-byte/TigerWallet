package main

// finance_ledger.go — Wallet & finance plane core: double-entry ledger,
// multi-chain accounts, atomic KYC-gated P2P transfers, full transaction
// history, per-token feature switches, and the admin audit log.
//
// Invariants (enforced by PostgreSQL constraints + single-tx posting):
//   - every balance movement is a journal with >= 2 legs that nets to zero
//     per currency (double-entry);
//   - balances can never go negative (CHECK constraint -> tx rollback);
//   - locked funds (escrow / pending withdrawals) can never exceed balance;
//   - all finance admin actions are audit-logged.

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ---------------------------------------------------------------------------
// Configuration (env-driven, fail-closed)
// ---------------------------------------------------------------------------

var financeCfg struct {
	// masterSeed derives deterministic per-user deposit addresses (HKDF).
	// WALLET_MASTER_SEED — hex (preferred) or raw passphrase. Empty =>
	// deposit-address endpoints fail closed (503), never a random address.
	masterSeed []byte
	// withdrawHMAC signs every withdrawal request. WITHDRAW_HMAC_SECRET,
	// falling back to JWT_SECRET (which is mandatory at boot).
	withdrawHMAC []byte
	// autoWithdrawUSD — WITHDRAW_AUTO_THRESHOLD. Requests below this USD
	// value AND with a low risk score are auto-approved within a second.
	autoWithdrawUSD float64
	// riskAutoMax — risk score at or above this always queues for
	// superadmin sign-off regardless of amount.
	riskAutoMax int
}

func loadFinanceConfig() {
	seed := strings.TrimSpace(os.Getenv("WALLET_MASTER_SEED"))
	if s, err := hex.DecodeString(seed); err == nil && len(s) >= 16 {
		financeCfg.masterSeed = s
	} else if seed != "" {
		financeCfg.masterSeed = []byte(seed)
	}
	secret := os.Getenv("WITHDRAW_HMAC_SECRET")
	if secret == "" {
		secret = appConfig.JWTSecret // mandatory at boot (fail-closed there)
	}
	financeCfg.withdrawHMAC = []byte(secret)
	financeCfg.autoWithdrawUSD = 500
	if v := strings.TrimSpace(os.Getenv("WITHDRAW_AUTO_THRESHOLD")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			financeCfg.autoWithdrawUSD = f
		}
	}
	financeCfg.riskAutoMax = 50
	if v := strings.TrimSpace(os.Getenv("WITHDRAW_RISK_AUTO_MAX")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			financeCfg.riskAutoMax = n
		}
	}
}

// ---------------------------------------------------------------------------
// Schema (additive, idempotent)
// ---------------------------------------------------------------------------

const financeSchemaSQL = `
CREATE TABLE IF NOT EXISTS ledger_account (
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency TEXT NOT NULL,
    balance  NUMERIC(38,18) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    locked   NUMERIC(38,18) NOT NULL DEFAULT 0 CHECK (locked >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, currency),
    CHECK (locked <= balance)
);
CREATE TABLE IF NOT EXISTS ledger_journal (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind TEXT NOT NULL,          -- p2p_transfer | convert | withdraw | refund | deposit
    reference TEXT,              -- withdrawal id / escrow id / convert id
    memo TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS ledger_entry (
    id BIGSERIAL PRIMARY KEY,
    journal_id UUID NOT NULL REFERENCES ledger_journal(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency TEXT NOT NULL,
    amount NUMERIC(38,18) NOT NULL,         -- signed: credit + / debit -
    direction TEXT NOT NULL CHECK (direction IN ('debit','credit')),
    balance_after NUMERIC(38,18) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ledger_entry_user_idx ON ledger_entry(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS ledger_journal_kind_idx ON ledger_journal(kind, created_at DESC);

CREATE TABLE IF NOT EXISTS withdrawal_request (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency TEXT NOT NULL,
    amount NUMERIC(38,18) NOT NULL CHECK (amount > 0),
    to_address TEXT NOT NULL,
    usd_value NUMERIC(20,2),
    risk_score INT NOT NULL,
    risk_reasons TEXT[] NOT NULL DEFAULT '{}',
    signature TEXT NOT NULL,                -- HMAC-SHA256 over the canonical payload
    status TEXT NOT NULL CHECK (status IN ('auto_approved','queued','approved','rejected')),
    decided_by UUID, decided_at TIMESTAMPTZ, decision_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS withdrawal_status_idx ON withdrawal_request(status, created_at DESC);
CREATE INDEX IF NOT EXISTS withdrawal_user_idx ON withdrawal_request(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS convert_rate (
    from_currency TEXT NOT NULL,
    to_currency TEXT NOT NULL,
    rate NUMERIC(38,18) NOT NULL CHECK (rate > 0),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (from_currency, to_currency)
);
CREATE TABLE IF NOT EXISTS convert_order (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_currency TEXT NOT NULL,
    to_currency TEXT NOT NULL,
    from_amount NUMERIC(38,18) NOT NULL CHECK (from_amount > 0),
    to_amount NUMERIC(38,18) NOT NULL CHECK (to_amount > 0),
    rate NUMERIC(38,18) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS convert_order_user_idx ON convert_order(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS p2p_escrow_order (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    buyer_id UUID REFERENCES users(id) ON DELETE CASCADE,
    currency TEXT NOT NULL,
    amount NUMERIC(38,18) NOT NULL CHECK (amount > 0),
    fiat_currency TEXT NOT NULL,
    fiat_amount NUMERIC(20,2) NOT NULL CHECK (fiat_amount > 0),
    payment_method_code TEXT NOT NULL,
    country_code TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','escrowed','paid','released','disputed','cancelled','resolved')),
    dispute_reason TEXT,
    resolved_by UUID, resolution TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS p2p_escrow_status_idx ON p2p_escrow_order(status, created_at DESC);

CREATE TABLE IF NOT EXISTS token_switch (
    currency TEXT PRIMARY KEY,
    deposit_enabled BOOLEAN NOT NULL DEFAULT true,
    withdraw_enabled BOOLEAN NOT NULL DEFAULT true,
    p2p_enabled BOOLEAN NOT NULL DEFAULT true,
    convert_enabled BOOLEAN NOT NULL DEFAULT true,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS admin_role (
    name TEXT PRIMARY KEY,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS admin_role_grant (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_name TEXT NOT NULL REFERENCES admin_role(name) ON DELETE CASCADE,
    granted_by UUID,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_name)
);

CREATE TABLE IF NOT EXISTS finance_audit_log (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id UUID,
    actor_role TEXT,
    action TEXT NOT NULL,
    target TEXT,
    detail JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS finance_audit_action_idx ON finance_audit_log(action, created_at DESC);
`

func migrateFinance(ctx context.Context) error {
	_, err := store.PG.Exec(ctx, financeSchemaSQL)
	return err
}

// ---------------------------------------------------------------------------
// Double-entry ledger core (all callers run inside their own pgx.Tx)
// ---------------------------------------------------------------------------

type ledgerLeg struct {
	UserID   uuid.UUID
	Currency string
	Amount   string // positive decimal string
	Debit    bool
}

// validateLegs enforces the double-entry invariant: every currency nets to
// zero across the legs and every amount is a positive decimal.
func validateLegs(legs []ledgerLeg) error {
	if len(legs) < 2 {
		return errors.New("a journal requires at least two legs")
	}
	net := map[string]float64{}
	for _, l := range legs {
		if l.Currency == "" {
			return errors.New("leg currency required")
		}
		f, err := strconv.ParseFloat(l.Amount, 64)
		if err != nil || f <= 0 {
			return fmt.Errorf("invalid leg amount %q", l.Amount)
		}
		if l.Debit {
			net[l.Currency] -= f
		} else {
			net[l.Currency] += f
		}
	}
	for cur, n := range net {
		if n < -1e-9 || n > 1e-9 {
			return fmt.Errorf("journal does not balance for %s (net %v)", cur, n)
		}
	}
	return nil
}

// ledgerPost writes a journal + entries and moves balances atomically.
// Negative balances are rejected by the CHECK constraint -> caller rolls back.
func ledgerPost(ctx context.Context, tx pgx.Tx, kind, reference, memo string, legs []ledgerLeg) (string, error) {
	if err := validateLegs(legs); err != nil {
		return "", err
	}
	var journalID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO ledger_journal(kind, reference, memo) VALUES($1,$2,$3) RETURNING id`,
		kind, reference, memo).Scan(&journalID); err != nil {
		return "", err
	}
	for _, l := range legs {
		signed := l.Amount
		dir := "credit"
		if l.Debit {
			signed = "-" + l.Amount
			dir = "debit"
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO ledger_account(user_id, currency) VALUES($1,$2)
			 ON CONFLICT (user_id, currency) DO NOTHING`, l.UserID, l.Currency); err != nil {
			return "", err
		}
		var balAfter pgtype.Numeric
		if err := tx.QueryRow(ctx,
			`UPDATE ledger_account SET balance = balance + $3::numeric, updated_at = now()
			 WHERE user_id=$1 AND currency=$2 RETURNING balance`,
			l.UserID, l.Currency, signed).Scan(&balAfter); err != nil {
			return "", fmt.Errorf("insufficient funds or ledger error for %s: %w", l.Currency, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO ledger_entry(journal_id, user_id, currency, amount, direction, balance_after)
			 VALUES($1,$2,$3,$4::numeric,$5,$6)`,
			journalID, l.UserID, l.Currency, signed, dir, balAfter); err != nil {
			return "", err
		}
	}
	return journalID, nil
}

// ledgerLock moves amount from available to locked (escrow / withdrawal hold).
func ledgerLock(ctx context.Context, tx pgx.Tx, userID uuid.UUID, currency, amount string) error {
	if _, err := tx.Exec(ctx,
		`INSERT INTO ledger_account(user_id, currency) VALUES($1,$2)
		 ON CONFLICT (user_id, currency) DO NOTHING`, userID, currency); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx,
		`UPDATE ledger_account SET locked = locked + $3::numeric, updated_at = now()
		 WHERE user_id=$1 AND currency=$2 AND balance - locked >= $3::numeric`,
		userID, currency, amount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("insufficient available balance")
	}
	return nil
}

// ledgerUnlock releases a hold (rejection / cancellation refund path).
func ledgerUnlock(ctx context.Context, tx pgx.Tx, userID uuid.UUID, currency, amount string) error {
	tag, err := tx.Exec(ctx,
		`UPDATE ledger_account SET locked = locked - $3::numeric, updated_at = now()
		 WHERE user_id=$1 AND currency=$2 AND locked >= $3::numeric`,
		userID, currency, amount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("no locked funds to release")
	}
	return nil
}

// ledgerSettleLocked burns locked funds (approved withdrawal / escrow release
// pays out): balance and locked both decrease.
func ledgerSettleLocked(ctx context.Context, tx pgx.Tx, userID uuid.UUID, currency, amount string) error {
	tag, err := tx.Exec(ctx,
		`UPDATE ledger_account SET balance = balance - $3::numeric, locked = locked - $3::numeric, updated_at = now()
		 WHERE user_id=$1 AND currency=$2 AND locked >= $3::numeric`,
		userID, currency, amount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("no locked funds to settle")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Valuation (best-effort; stablecoins pinned, majors via CoinGecko w/ cache)
// ---------------------------------------------------------------------------

var financeStablecoins = map[string]bool{
	"USDT": true, "USDC": true, "DAI": true, "BUSD": true,
	"TUSD": true, "USDP": true, "FDUSD": true, "PYUSD": true,
}

// usdValueOf returns the USD value of amount of currency. ok=false when no
// price source is available (callers must fail safe, never guess).
func usdValueOf(ctx context.Context, currency string, amount float64) (float64, bool) {
	cur := strings.ToUpper(currency)
	if financeStablecoins[cur] {
		return amount, true
	}
	coinID := coinIDForSymbol(cur)
	cacheK := store.cacheKey("price", coinID)
	var p CoinGeckoPrice
	if err := store.GetCache(ctx, cacheK, &p); err == nil && p.PriceUSD > 0 {
		return amount * p.PriceUSD, true
	}
	pctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	fresh, err := FetchTokenPrice(pctx, coinID)
	if err != nil || fresh == nil || fresh.PriceUSD <= 0 {
		return 0, false
	}
	_ = store.SetCache(ctx, cacheK, *fresh, 60*time.Second)
	return amount * fresh.PriceUSD, true
}

// ---------------------------------------------------------------------------
// Per-token feature switches
// ---------------------------------------------------------------------------

type tokenSwitch struct {
	Currency        string `json:"currency"`
	DepositEnabled  bool   `json:"deposit_enabled"`
	WithdrawEnabled bool   `json:"withdraw_enabled"`
	P2PEnabled      bool   `json:"p2p_enabled"`
	ConvertEnabled  bool   `json:"convert_enabled"`
}

// switchEnabled reports whether a feature is enabled for a currency.
// Absent row = enabled (default-on); superadmin turns features off per token.
func switchEnabled(ctx context.Context, currency, feature string) bool {
	col := map[string]string{
		"deposit":  "deposit_enabled",
		"withdraw": "withdraw_enabled",
		"p2p":      "p2p_enabled",
		"convert":  "convert_enabled",
	}[feature]
	if col == "" {
		return false
	}
	var enabled bool
	err := store.PG.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM token_switch WHERE currency=$1`, col),
		strings.ToUpper(currency)).Scan(&enabled)
	if err != nil {
		return true // no row -> default enabled
	}
	return enabled
}

// ---------------------------------------------------------------------------
// Audit log
// ---------------------------------------------------------------------------

func auditFinance(ctx context.Context, actorID uuid.UUID, actorRole, action, target string, detail any) {
	d := "null"
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			d = string(b)
		}
	}
	_, _ = store.PG.Exec(ctx,
		`INSERT INTO finance_audit_log(actor_user_id, actor_role, action, target, detail) VALUES($1,$2,$3,$4,$5::jsonb)`,
		actorID, actorRole, action, target, d)
}

// ---------------------------------------------------------------------------
// User handlers: accounts, history, internal transfer, switches
// ---------------------------------------------------------------------------

var financeAssets = []string{"BTC", "ETH", "USDT", "USDC", "BNB", "SOL", "TRX", "MATIC", "LTC", "DOGE"}

// handleFinanceAccounts returns the user's multi-chain accounts with
// balance / locked / available and best-effort USD valuation.
func handleFinanceAccounts(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	ctx := c.Request.Context()
	type acct struct {
		Currency  string  `json:"currency"`
		Balance   string  `json:"balance"`
		Locked    string  `json:"locked"`
		Available string  `json:"available"`
		USDValue  float64 `json:"usd_value,omitempty"`
	}
	out := make([]acct, 0, len(financeAssets))
	for _, cur := range financeAssets {
		var bal, locked pgtype.Numeric
		err := store.PG.QueryRow(ctx,
			`SELECT balance, locked FROM ledger_account WHERE user_id=$1 AND currency=$2`,
			uid, cur).Scan(&bal, &locked)
		if err != nil { // no account yet -> zeroed entry
			out = append(out, acct{Currency: cur, Balance: "0", Locked: "0", Available: "0"})
			continue
		}
		b, _ := bal.Float64Value()
		l, _ := locked.Float64Value()
		avail := b.Float64 - l.Float64
		a := acct{
			Currency:  cur,
			Balance:   numericString(bal),
			Locked:    numericString(locked),
			Available: strconv.FormatFloat(avail, 'f', -1, 64),
		}
		if usd, ok := usdValueOf(ctx, cur, b.Float64); ok {
			a.USDValue = usd
		}
		out = append(out, a)
	}
	c.JSON(http.StatusOK, gin.H{"accounts": out})
}

// handleFinanceHistory returns the user's full ledger history (every entry
// joined with its journal: kind, reference, direction, balance_after).
func handleFinanceHistory(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	currency := strings.ToUpper(c.Query("currency"))
	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	q := `SELECT e.id, e.journal_id, j.kind, COALESCE(j.reference,''), COALESCE(j.memo,''),
	             e.currency, e.amount::text, e.direction, e.balance_after::text, e.created_at
	      FROM ledger_entry e JOIN ledger_journal j ON j.id = e.journal_id
	      WHERE e.user_id=$1`
	args := []any{uid}
	if currency != "" {
		q += ` AND e.currency=$2`
		args = append(args, currency)
	}
	q += ` ORDER BY e.id DESC LIMIT ` + strconv.Itoa(limit)
	rows, err := store.PG.Query(c.Request.Context(), q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "history query failed"})
		return
	}
	defer rows.Close()
	type entry struct {
		ID           int64     `json:"id"`
		JournalID    string    `json:"journal_id"`
		Kind         string    `json:"kind"`
		Reference    string    `json:"reference"`
		Memo         string    `json:"memo"`
		Currency     string    `json:"currency"`
		Amount       string    `json:"amount"`
		Direction    string    `json:"direction"`
		BalanceAfter string    `json:"balance_after"`
		CreatedAt    time.Time `json:"created_at"`
	}
	out := []entry{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.ID, &e.JournalID, &e.Kind, &e.Reference, &e.Memo,
			&e.Currency, &e.Amount, &e.Direction, &e.BalanceAfter, &e.CreatedAt); err != nil {
			continue
		}
		out = append(out, e)
	}
	c.JSON(http.StatusOK, gin.H{"history": out, "count": len(out)})
}

// handleFinanceTransfer performs an atomic internal P2P transfer between two
// users, gated on KYC verification of BOTH parties. One transaction, two
// legs, double-entry balanced.
func handleFinanceTransfer(c *gin.Context) {
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		ToUserID string `json:"to_user_id"`
		ToEmail  string `json:"to_email"`
		Currency string `json:"currency" binding:"required"`
		Amount   string `json:"amount" binding:"required"`
		Memo     string `json:"memo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if !switchEnabled(c.Request.Context(), req.Currency, "p2p") {
		c.JSON(http.StatusForbidden, gin.H{"error": "P2P transfers are disabled for " + req.Currency})
		return
	}
	if _, err := strconv.ParseFloat(req.Amount, 64); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	ctx := c.Request.Context()
	// KYC gate (fail-closed) — both parties must be verified.
	if !kycVerified(ctx, uid) {
		c.JSON(http.StatusForbidden, gin.H{"error": "KYC verification required for P2P transfers", "kyc_required": true})
		return
	}
	var toID uuid.UUID
	switch {
	case req.ToUserID != "":
		toID, err = uuid.Parse(req.ToUserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to_user_id"})
			return
		}
	case req.ToEmail != "":
		err = store.PG.QueryRow(ctx, `SELECT id FROM users WHERE email=$1`, strings.ToLower(strings.TrimSpace(req.ToEmail))).Scan(&toID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipient not found"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "to_user_id or to_email required"})
		return
	}
	if toID == uid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot transfer to yourself"})
		return
	}
	if !kycVerified(ctx, toID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "recipient has not completed KYC verification", "kyc_required": true})
		return
	}
	tx, err := store.PG.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ledger unavailable"})
		return
	}
	defer tx.Rollback(ctx)
	journalID, err := ledgerPost(ctx, tx, "p2p_transfer", "", req.Memo, []ledgerLeg{
		{UserID: uid, Currency: req.Currency, Amount: req.Amount, Debit: true},
		{UserID: toID, Currency: req.Currency, Amount: req.Amount, Debit: false},
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "transferred", "journal_id": journalID, "currency": req.Currency, "amount": req.Amount})
}

// handleFinanceSwitches exposes the per-token switches to users (read-only,
// so clients can hide disabled flows before submitting).
func handleFinanceSwitches(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT currency, deposit_enabled, withdraw_enabled, p2p_enabled, convert_enabled FROM token_switch`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "switch query failed"})
		return
	}
	defer rows.Close()
	m := map[string]tokenSwitch{}
	for rows.Next() {
		var s tokenSwitch
		if err := rows.Scan(&s.Currency, &s.DepositEnabled, &s.WithdrawEnabled, &s.P2PEnabled, &s.ConvertEnabled); err != nil {
			continue
		}
		m[s.Currency] = s
	}
	out := make([]tokenSwitch, 0, len(financeAssets))
	for _, cur := range financeAssets {
		if s, ok := m[cur]; ok {
			out = append(out, s)
		} else {
			out = append(out, tokenSwitch{Currency: cur, DepositEnabled: true, WithdrawEnabled: true, P2PEnabled: true, ConvertEnabled: true})
		}
	}
	c.JSON(http.StatusOK, gin.H{"switches": out})
}

// numericString renders a pgtype.Numeric as a plain decimal string.
func numericString(n pgtype.Numeric) string {
	if !n.Valid {
		return "0"
	}
	f, err := n.Float64Value()
	if err != nil {
		return "0"
	}
	return strconv.FormatFloat(f.Float64, 'f', -1, 64)
}
