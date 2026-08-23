package main

// auto_signer.go — MasterWallet AUTO-APPROVE / AUTO-SIGN daemon.
//
// Requirement (product owner):
//   1. "UserWallet always gets automatic sign and automatic approval within a
//      second from SuperAdmin or MasterWallet owner or Admin from admin panel."
//   2. "MasterWallet CANNOT withdraw users' funds of any UserWallet, but
//      provides automatic approval and automatic sign within a second."
//   3. "MasterWallet owner cannot withdraw any fees earned from users'
//      transactions and cannot withdraw any revenue without TigerWallet
//      SuperAdmin permission."
//
// Design:
//   - A background goroutine polls the real `transactions` table every
//     MASTER_AUTO_SIGN_POLL_MS (default 100ms) for pending, user-initiated
//     approval requests and resolves them end-to-end: approve (real rows in
//     transaction_signatures + approval_requests), sign (EIP-1559 via
//     SignEVMTransaction; non-EVM via non_evm_crypto.go signers), broadcast via
//     eth_sendRawTransaction (BroadcastTransaction), then push a websocket
//     event through the existing hub so UIs update within a second.
//   - The transaction classifier mirrors wl_control_plane/rust/src/classifier.rs:
//     UserTransfer/Swap/Stake/NftTransfer/PersonalSign/TypedDataSign are
//     AUTO-APPROVABLE; RevenuePayout/TreasuryTransfer/TreasurySweep/
//     FeeWithdrawal are NEVER auto-approved — they stay pending for the
//     existing two-party SuperAdmin co-sign path (license_gate.go).
//   - guardUserFunds is the critical security invariant: the daemon refuses to
//     sign anything that moves funds OUT of a user sub-wallet to a destination
//     not belonging to that same user (i.e. the MasterWallet can never pull
//     user funds). Fail-closed: on any doubt the tx stays pending for manual
//     review.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

// ----------------------------------------------------------------------------
// Transaction classifier (Go mirror of wl_control_plane/rust/src/classifier.rs)
// ----------------------------------------------------------------------------

// TxKind is the coarse transaction classification (mirrors the Rust TxKind).
type TxKind int

const (
	TxKindUnknown TxKind = iota
	TxKindUserTransfer
	TxKindSwap
	TxKindStake
	TxKindNftTransfer
	TxKindPersonalSign
	TxKindTypedDataSign
	TxKindRevenuePayout
	TxKindTreasuryTransfer
	TxKindTreasurySweep
	TxKindFeeWithdrawal
)

// txKindFromString maps a tx_type string to its TxKind (mirrors
// TxKind::from_str in the Rust classifier).
func txKindFromString(s string) TxKind {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "transfer", "send":
		return TxKindUserTransfer
	case "swap", "trade":
		return TxKindSwap
	case "stake", "unstake", "claim":
		return TxKindStake
	case "nft_transfer":
		return TxKindNftTransfer
	case "personal_sign":
		return TxKindPersonalSign
	case "typed_data_sign":
		return TxKindTypedDataSign
	case "revenue_payout":
		return TxKindRevenuePayout
	case "treasury_transfer":
		return TxKindTreasuryTransfer
	case "treasury_sweep":
		return TxKindTreasurySweep
	case "fee_withdrawal":
		return TxKindFeeWithdrawal
	default:
		return TxKindUnknown
	}
}

func (k TxKind) String() string {
	switch k {
	case TxKindUserTransfer:
		return "transfer"
	case TxKindSwap:
		return "swap"
	case TxKindStake:
		return "stake"
	case TxKindNftTransfer:
		return "nft_transfer"
	case TxKindPersonalSign:
		return "personal_sign"
	case TxKindTypedDataSign:
		return "typed_data_sign"
	case TxKindRevenuePayout:
		return "revenue_payout"
	case TxKindTreasuryTransfer:
		return "treasury_transfer"
	case TxKindTreasurySweep:
		return "treasury_sweep"
	case TxKindFeeWithdrawal:
		return "fee_withdrawal"
	default:
		return "unknown"
	}
}

// autoApprovable reports whether the kind may be auto-approved/auto-signed by
// the daemon. Fee/revenue/treasury withdrawals are NEVER auto-approved: they
// require the existing two-party SuperAdmin co-sign path. Unknown kinds are
// fail-closed (manual review).
func (k TxKind) autoApprovable() bool {
	switch k {
	case TxKindUserTransfer, TxKindSwap, TxKindStake,
		TxKindNftTransfer, TxKindPersonalSign, TxKindTypedDataSign:
		return true
	default:
		return false
	}
}

// approvalDecision mirrors the Rust ApprovalDecision.
type approvalDecision struct {
	Mode     string // "auto" | "manual"
	Kind     TxKind
	Approved bool
	Reason   string
}

// classifyTransaction classifies an outgoing transaction. Manual mode for
// fee/revenue/treasury withdrawals and for unknown types; Auto otherwise.
// Pure function (mirrors classify_transaction in the Rust classifier, minus
// the license/rules inputs which are enforced by the daemon's policy gate).
func classifyTransaction(txType string) approvalDecision {
	kind := txKindFromString(txType)
	switch kind {
	case TxKindRevenuePayout, TxKindTreasuryTransfer, TxKindTreasurySweep, TxKindFeeWithdrawal:
		return approvalDecision{
			Mode:   "manual",
			Kind:   kind,
			Reason: "fee/revenue/treasury withdrawal requires SuperAdmin two-party co-sign",
		}
	case TxKindUnknown:
		return approvalDecision{
			Mode:   "manual",
			Kind:   kind,
			Reason: "unrecognized tx_type; left pending for manual review (fail-closed)",
		}
	default:
		return approvalDecision{Mode: "auto", Kind: kind, Approved: true}
	}
}

// ----------------------------------------------------------------------------
// User-funds guard (critical security invariant)
// ----------------------------------------------------------------------------

// pendingAutoTx is one pending transaction row the daemon considers.
type pendingAutoTx struct {
	ID             string
	MasterWalletID string
	SubWalletID    string
	TxType         string
	Blockchain     string
	FromAddress    string
	ToAddress      string
	Amount         string // wei-scale integer (transactions.amount is NUMERIC(78,0))
	TokenSymbol    string
	TokenAddress   string
	ChainID        int64
	UserInitiated  bool // recorded at creation in metadata (provenance)
	CreatedAt      time.Time
}

// guardContext carries the facts the daemon resolved from the store before
// invoking the guard. Kept separate from the tx so guardUserFunds is a pure
// function (unit-testable without a database).
type guardContext struct {
	// MasterAddress is the MasterWallet's own address. User funds must never
	// flow TO it via auto-sign (that would be the MasterWallet pulling user
	// funds), and auto-sign never signs FROM it (that is the treasury domain).
	MasterAddress string
	// TreasuryAddresses are extra revenue/fee/treasury destinations that must
	// never receive auto-signed user funds (normalized, case-insensitive).
	TreasuryAddresses map[string]bool
	// SubWalletAddress is the on-chain address of the tx's sub-wallet.
	SubWalletAddress string
	// SubWalletFound is false when the sub-wallet lookup failed or the
	// sub-wallet does not belong to this master wallet.
	SubWalletFound bool
}

func normalizeAddr(a string) string { return strings.ToLower(strings.TrimSpace(a)) }

// guardUserFunds enforces: the auto-signer may ONLY sign a transaction that
// (a) is an auto-approvable kind, (b) was created through the user-initiated
// approval flow from a real user sub-wallet of this MasterWallet, and (c) does
// not move user funds to the MasterWallet/treasury/revenue. Fail-closed: any
// doubt returns an error and the tx stays pending for manual review.
func guardUserFunds(tx *pendingAutoTx, g guardContext) error {
	dec := classifyTransaction(tx.TxType)
	if !dec.Approved {
		return fmt.Errorf("kind %q is never auto-signed: %s", dec.Kind, dec.Reason)
	}
	// (b1) The tx must originate from a user sub-wallet of this MasterWallet,
	// not from the MasterWallet address itself (treasury/revenue domain).
	if tx.SubWalletID == "" || !g.SubWalletFound {
		return errors.New("not created through the user-initiated sub-wallet flow")
	}
	if g.SubWalletAddress == "" || normalizeAddr(g.SubWalletAddress) != normalizeAddr(tx.FromAddress) {
		return errors.New("from_address does not match the user sub-wallet address")
	}
	if g.MasterAddress != "" && normalizeAddr(tx.FromAddress) == normalizeAddr(g.MasterAddress) {
		return errors.New("source is the master wallet (treasury domain); two-party co-sign required")
	}
	// (b2) Provenance: the destination must have been chosen by the user and
	// recorded at creation time (CreateTransaction marks metadata).
	if !tx.UserInitiated {
		return errors.New("destination was not recorded as user-chosen at creation")
	}
	if strings.TrimSpace(tx.ToAddress) == "" {
		return errors.New("empty destination")
	}
	// (c) The destination must not be the MasterWallet itself or any known
	// treasury/revenue/fee address — the MasterWallet can never pull user funds.
	if g.MasterAddress != "" && normalizeAddr(tx.ToAddress) == normalizeAddr(g.MasterAddress) {
		return errors.New("destination is the master wallet; auto-signing would move user funds to the MasterWallet")
	}
	if g.TreasuryAddresses[normalizeAddr(tx.ToAddress)] {
		return errors.New("destination is a treasury/revenue/fee address; two-party co-sign required")
	}
	// (d) The amount must be a well-formed non-negative integer (wei scale).
	if amt, ok := new(big.Int).SetString(strings.TrimSpace(tx.Amount), 10); !ok || amt.Sign() < 0 {
		return errors.New("malformed amount")
	}
	return nil
}

// exceedsCap reports whether amt exceeds cap; a nil or non-positive cap means
// unlimited (0 = unlimited per config contract).
func exceedsCap(amt *big.Int, cap *big.Int) bool {
	if cap == nil || cap.Sign() <= 0 {
		return false
	}
	return amt.Cmp(cap) > 0
}

// ----------------------------------------------------------------------------
// Auto-sign policy (per master wallet, owner/admin configurable)
// ----------------------------------------------------------------------------

// autoSignPolicy is the per-master-wallet auto-sign policy persisted in the
// auto_sign_policies table.
type autoSignPolicy struct {
	MasterWalletID     string    `json:"master_wallet_id"`
	Enabled            bool      `json:"enabled"`
	AllowTransfer      bool      `json:"allow_transfer"`
	AllowSwap          bool      `json:"allow_swap"`
	AllowStake         bool      `json:"allow_stake"`
	AllowNftTransfer   bool      `json:"allow_nft_transfer"`
	AllowPersonalSign  bool      `json:"allow_personal_sign"`
	AllowTypedDataSign bool      `json:"allow_typed_data_sign"`
	MaxAutoValueWei    string    `json:"max_auto_value_wei"` // "0" = unlimited
	CreatedAt          time.Time `json:"created_at,omitempty"`
	UpdatedAt          time.Time `json:"updated_at,omitempty"`
}

// defaultAutoSignPolicy returns the fail-open-for-users default: auto-sign
// enabled for all auto-approvable kinds, no value cap (env cap may still
// apply). Fee/revenue/treasury kinds are not represented here because they
// are NEVER auto-signed regardless of policy.
func defaultAutoSignPolicy(masterID string) *autoSignPolicy {
	return &autoSignPolicy{
		MasterWalletID:     masterID,
		Enabled:            true,
		AllowTransfer:      true,
		AllowSwap:          true,
		AllowStake:         true,
		AllowNftTransfer:   true,
		AllowPersonalSign:  true,
		AllowTypedDataSign: true,
		MaxAutoValueWei:    "0",
	}
}

// allowsKind reports whether the policy permits auto-signing this kind.
func (p *autoSignPolicy) allowsKind(k TxKind) bool {
	if p == nil || !p.Enabled {
		return false
	}
	switch k {
	case TxKindUserTransfer:
		return p.AllowTransfer
	case TxKindSwap:
		return p.AllowSwap
	case TxKindStake:
		return p.AllowStake
	case TxKindNftTransfer:
		return p.AllowNftTransfer
	case TxKindPersonalSign:
		return p.AllowPersonalSign
	case TxKindTypedDataSign:
		return p.AllowTypedDataSign
	default:
		return false // revenue/treasury/fee/unknown: never
	}
}

// maxValueWei parses the policy cap; nil/0 means unlimited.
func (p *autoSignPolicy) maxValueWei() *big.Int {
	if p == nil {
		return nil
	}
	v, ok := new(big.Int).SetString(strings.TrimSpace(p.MaxAutoValueWei), 10)
	if !ok {
		return nil
	}
	return v
}

// getAutoSignPolicy loads the policy for a master wallet, falling back to the
// defaults when no row exists. Nil-store guarded (returns defaults).
func (svc *Service) getAutoSignPolicy(ctx context.Context, masterID string) *autoSignPolicy {
	if svc == nil || svc.store == nil || svc.store.db == nil {
		return defaultAutoSignPolicy(masterID)
	}
	p := defaultAutoSignPolicy(masterID)
	err := svc.store.db.QueryRow(ctx,
		`SELECT enabled, allow_transfer, allow_swap, allow_stake, allow_nft_transfer,
			allow_personal_sign, allow_typed_data_sign, max_auto_value_wei::text, created_at, updated_at
		 FROM auto_sign_policies WHERE master_wallet_id = $1`, masterID).
		Scan(&p.Enabled, &p.AllowTransfer, &p.AllowSwap, &p.AllowStake, &p.AllowNftTransfer,
			&p.AllowPersonalSign, &p.AllowTypedDataSign, &p.MaxAutoValueWei, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return defaultAutoSignPolicy(masterID)
	}
	return p
}

// GetAutoSignPolicy GET /api/v1/master-wallet/:id/auto-sign-policy
func (svc *Service) GetAutoSignPolicy(c *gin.Context) {
	masterID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"policy": svc.getAutoSignPolicy(c.Request.Context(), masterID)})
}

// UpdateAutoSignPolicy PUT /api/v1/master-wallet/:id/auto-sign-policy
// Owner (created_by) or admin/super_admin only.
func (svc *Service) UpdateAutoSignPolicy(c *gin.Context) {
	masterID := c.Param("id")
	ctx := c.Request.Context()

	role := currentRole(c)
	if role != "admin" && role != "super_admin" {
		var createdBy string
		err := svc.store.db.QueryRow(ctx,
			`SELECT created_by::text FROM master_wallets WHERE id = $1`, masterID).Scan(&createdBy)
		if err != nil || createdBy != currentUserID(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the master wallet owner or an admin can change the auto-sign policy"})
			return
		}
	}

	var req struct {
		Enabled            *bool  `json:"enabled"`
		AllowTransfer      *bool  `json:"allow_transfer"`
		AllowSwap          *bool  `json:"allow_swap"`
		AllowStake         *bool  `json:"allow_stake"`
		AllowNftTransfer   *bool  `json:"allow_nft_transfer"`
		AllowPersonalSign  *bool  `json:"allow_personal_sign"`
		AllowTypedDataSign *bool  `json:"allow_typed_data_sign"`
		MaxAutoValueWei    string `json:"max_auto_value_wei"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p := svc.getAutoSignPolicy(ctx, masterID)
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}
	if req.AllowTransfer != nil {
		p.AllowTransfer = *req.AllowTransfer
	}
	if req.AllowSwap != nil {
		p.AllowSwap = *req.AllowSwap
	}
	if req.AllowStake != nil {
		p.AllowStake = *req.AllowStake
	}
	if req.AllowNftTransfer != nil {
		p.AllowNftTransfer = *req.AllowNftTransfer
	}
	if req.AllowPersonalSign != nil {
		p.AllowPersonalSign = *req.AllowPersonalSign
	}
	if req.AllowTypedDataSign != nil {
		p.AllowTypedDataSign = *req.AllowTypedDataSign
	}
	if strings.TrimSpace(req.MaxAutoValueWei) != "" {
		if _, ok := new(big.Int).SetString(strings.TrimSpace(req.MaxAutoValueWei), 10); !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "max_auto_value_wei must be a base-10 integer (0 = unlimited)"})
			return
		}
		p.MaxAutoValueWei = strings.TrimSpace(req.MaxAutoValueWei)
	}

	_, err := svc.store.db.Exec(ctx,
		`INSERT INTO auto_sign_policies (master_wallet_id, enabled, allow_transfer, allow_swap,
			allow_stake, allow_nft_transfer, allow_personal_sign, allow_typed_data_sign,
			max_auto_value_wei, updated_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 ON CONFLICT (master_wallet_id) DO UPDATE SET
			enabled=$2, allow_transfer=$3, allow_swap=$4, allow_stake=$5,
			allow_nft_transfer=$6, allow_personal_sign=$7, allow_typed_data_sign=$8,
			max_auto_value_wei=$9, updated_by=$10, updated_at=NOW()`,
		masterID, p.Enabled, p.AllowTransfer, p.AllowSwap, p.AllowStake, p.AllowNftTransfer,
		p.AllowPersonalSign, p.AllowTypedDataSign, p.MaxAutoValueWei, nilIfEmpty(currentUserID(c)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist policy", "detail": err.Error()})
		return
	}
	svc.store.audit(ctx, masterID, "auto_sign_policy.update", "policy", "user", currentUserID(c), "master_wallet", masterID, "normal",
		gin.H{"enabled": p.Enabled, "max_auto_value_wei": p.MaxAutoValueWei})
	c.JSON(http.StatusOK, gin.H{"policy": svc.getAutoSignPolicy(ctx, masterID)})
}

// ----------------------------------------------------------------------------
// AutoSigner daemon
// ----------------------------------------------------------------------------

// AutoSigner is the background auto-approve/auto-sign daemon. Config via env:
//
//	MASTER_AUTO_SIGN_ENABLED        (default true)
//	MASTER_AUTO_SIGN_MAX_VALUE_WEI  (default 0 = unlimited; global cap on top of
//	                                 the per-wallet policy cap)
//	MASTER_AUTO_SIGN_POLL_MS        (default 100)
//	MASTER_AUTO_SIGN_PASSWORD       password decrypting stored master-wallet
//	                                 seeds so the daemon can derive the
//	                                 sub-wallet signing keys; when empty the
//	                                 daemon still approves within a second but
//	                                 cannot broadcast (fail-closed, logged).
type AutoSigner struct {
	svc          *Service
	enabled      bool
	pollInterval time.Duration
	maxValueWei  *big.Int
	password     string
}

// NewAutoSigner builds the daemon from the environment with sane defaults.
func NewAutoSigner(svc *Service) *AutoSigner {
	enabled := true
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("MASTER_AUTO_SIGN_ENABLED"))); v == "false" || v == "0" || v == "no" {
		enabled = false
	}
	pollMS, err := strconv.Atoi(strings.TrimSpace(os.Getenv("MASTER_AUTO_SIGN_POLL_MS")))
	if err != nil || pollMS < 20 {
		pollMS = 100
	}
	maxWei, ok := new(big.Int).SetString(strings.TrimSpace(os.Getenv("MASTER_AUTO_SIGN_MAX_VALUE_WEI")), 10)
	if !ok {
		maxWei = big.NewInt(0)
	}
	return &AutoSigner{
		svc:          svc,
		enabled:      enabled,
		pollInterval: time.Duration(pollMS) * time.Millisecond,
		maxValueWei:  maxWei,
		password:     os.Getenv("MASTER_AUTO_SIGN_PASSWORD"),
	}
}

// Start runs the polling loop until ctx is cancelled (graceful shutdown).
func (a *AutoSigner) Start(ctx context.Context) {
	if !a.enabled {
		log.Println("auto-signer: disabled via MASTER_AUTO_SIGN_ENABLED")
		return
	}
	if a.svc == nil || a.svc.store == nil || a.svc.store.db == nil {
		log.Println("auto-signer: no database (degraded mode) — daemon not started")
		return
	}
	if a.password == "" {
		log.Println("auto-signer: MASTER_AUTO_SIGN_PASSWORD unset — approvals still recorded within a second, broadcast disabled (fail-closed)")
	}
	log.Printf("auto-signer: started (poll=%s, max_value_wei=%s)", a.pollInterval, a.maxValueWei.String())
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("auto-signer: stopped")
			return
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("auto-signer: poll panic recovered: %v", r)
					}
				}()
				a.pollOnce(ctx)
			}()
		}
	}
}

// pollOnce picks up a batch of pending user transactions and processes them.
func (a *AutoSigner) pollOnce(ctx context.Context) {
	rows, err := a.svc.store.db.Query(ctx,
		`SELECT id::text, master_wallet_id::text, COALESCE(sub_wallet_id::text,''), tx_type, blockchain,
			from_address, to_address, amount::text, COALESCE(token_symbol,''), COALESCE(token_address,''),
			COALESCE(chain_id,0), COALESCE(metadata,'{}'::jsonb), created_at
		 FROM transactions
		 WHERE status = 'pending' AND (tx_hash IS NULL OR tx_hash = '')
		   AND created_at > NOW() - INTERVAL '24 hours'
		 ORDER BY created_at ASC LIMIT 50`)
	if err != nil {
		log.Printf("auto-signer: poll query failed: %v", err)
		return
	}
	defer rows.Close()
	var batch []pendingAutoTx
	for rows.Next() {
		var tx pendingAutoTx
		var metadata []byte
		if err := rows.Scan(&tx.ID, &tx.MasterWalletID, &tx.SubWalletID, &tx.TxType, &tx.Blockchain,
			&tx.FromAddress, &tx.ToAddress, &tx.Amount, &tx.TokenSymbol, &tx.TokenAddress,
			&tx.ChainID, &metadata, &tx.CreatedAt); err != nil {
			log.Printf("auto-signer: scan failed: %v", err)
			continue
		}
		var meta map[string]interface{}
		if err := json.Unmarshal(metadata, &meta); err == nil {
			tx.UserInitiated, _ = meta["user_initiated"].(bool)
		}
		batch = append(batch, tx)
	}
	// Cache policies per master wallet within this batch.
	policies := map[string]*autoSignPolicy{}
	for i := range batch {
		tx := &batch[i]
		pol, ok := policies[tx.MasterWalletID]
		if !ok {
			pol = a.svc.getAutoSignPolicy(ctx, tx.MasterWalletID)
			policies[tx.MasterWalletID] = pol
		}
		a.processTx(ctx, tx, pol)
	}
}

// processTx classifies, guards, approves, signs and broadcasts one pending tx.
// The end-to-end latency target is <1s from pickup; it is measured with
// time.Since and logged, as is the age since the request was created.
func (a *AutoSigner) processTx(ctx context.Context, tx *pendingAutoTx, pol *autoSignPolicy) {
	start := time.Now()
	dec := classifyTransaction(tx.TxType)
	if dec.Mode != "auto" {
		// Fee/revenue/treasury (and unknown) kinds are NEVER auto-approved; they
		// stay pending for the two-party SuperAdmin co-sign path. Not an error.
		return
	}
	if !pol.allowsKind(dec.Kind) {
		return // owner/admin disabled this kind; leave for manual review
	}
	amount, ok := new(big.Int).SetString(strings.TrimSpace(tx.Amount), 10)
	if !ok {
		return // malformed amount; guard would also refuse — leave pending
	}
	if exceedsCap(amount, a.maxValueWei) || exceedsCap(amount, pol.maxValueWei()) {
		return // exceeds auto-sign cap; leave pending for manual approval
	}

	// Resolve the guard context from the store (fail-closed on any error).
	g, gerr := a.resolveGuardContext(ctx, tx)
	if gerr != nil {
		log.Printf("auto-signer: guard context for tx %s: %v — left pending", tx.ID, gerr)
		return
	}
	if err := guardUserFunds(tx, g); err != nil {
		log.Printf("auto-signer: guard refused tx %s (%s -> %s, kind %s): %v — left pending for manual review",
			tx.ID, tx.FromAddress, tx.ToAddress, dec.Kind, err)
		return
	}

	// Atomically claim the tx so a concurrent poll/admin action can't double-sign.
	tag, err := a.svc.store.db.Exec(ctx,
		`UPDATE transactions SET status='approved', updated_at=NOW() WHERE id=$1 AND status='pending'`, tx.ID)
	if err != nil || tag.RowsAffected() == 0 {
		return
	}

	// Record the automatic approval (real rows in the existing schema). The
	// daemon signs on behalf of the master wallet's owner signer when one is
	// registered; approval_requests are always resolved.
	var signerID string
	_ = a.svc.store.db.QueryRow(ctx,
		`SELECT id::text FROM signers WHERE master_wallet_id = $1 ORDER BY
			CASE signer_type WHEN 'owner' THEN 0 WHEN 'cosigner' THEN 1 ELSE 2 END, created_at
		 LIMIT 1`, tx.MasterWalletID).Scan(&signerID)
	if signerID != "" {
		_, _ = a.svc.store.db.Exec(ctx,
			`INSERT INTO transaction_signatures (transaction_id, signer_id, signature_status, approved_at)
			 VALUES ($1, $2, 'signed', NOW())
			 ON CONFLICT (transaction_id, signer_id) DO UPDATE SET signature_status='signed', approved_at=NOW()`,
			tx.ID, signerID)
	}
	_, _ = a.svc.store.db.Exec(ctx,
		`UPDATE approval_requests SET current_approvals = current_approvals + 1, is_approved = true, resolved_at = NOW()
		 WHERE transaction_id = $1`, tx.ID)

	status := "approved"
	txHash := ""
	var signErr error
	if a.password != "" {
		txHash, status, signErr = a.signAndBroadcast(ctx, tx)
		if signErr != nil {
			log.Printf("auto-signer: sign/broadcast failed for tx %s: %v — status stays '%s'", tx.ID, signErr, status)
			_, _ = a.svc.store.db.Exec(ctx,
				`UPDATE transactions SET status=$2, error_message=$3, updated_at=NOW() WHERE id=$1`,
				tx.ID, status, signErr.Error())
		} else {
			_, _ = a.svc.store.db.Exec(ctx,
				`UPDATE transactions SET tx_hash=$2, status=$3, updated_at=NOW() WHERE id=$1`,
				tx.ID, txHash, status)
		}
	}

	elapsed := time.Since(start)
	age := time.Since(tx.CreatedAt)
	if elapsed > time.Second {
		log.Printf("auto-signer: SLOW tx %s kind=%s status=%s hash=%s took=%s age_since_create=%s (target <1s)",
			tx.ID, dec.Kind, status, txHash, elapsed, age)
	} else {
		log.Printf("auto-signer: tx %s kind=%s status=%s hash=%s took=%s age_since_create=%s",
			tx.ID, dec.Kind, status, txHash, elapsed, age)
	}
	a.svc.store.audit(ctx, tx.MasterWalletID, "transaction.auto_sign", "transaction", "service", "auto-signer",
		"transaction", tx.ID, "normal", gin.H{"kind": dec.Kind.String(), "status": status, "tx_hash": txHash, "elapsed_ms": elapsed.Milliseconds()})
	if a.svc.hub != nil {
		a.svc.notifyEvent(tx.MasterWalletID, "transaction.auto_approved", gin.H{
			"transaction_id": tx.ID, "kind": dec.Kind.String(), "status": status,
			"tx_hash": txHash, "from": tx.FromAddress, "to": tx.ToAddress,
			"amount": tx.Amount, "chain_id": tx.ChainID, "latency_ms": elapsed.Milliseconds(),
		})
	}
}

// resolveGuardContext loads the facts guardUserFunds needs from the store.
func (a *AutoSigner) resolveGuardContext(ctx context.Context, tx *pendingAutoTx) (guardContext, error) {
	var g guardContext
	if tx.SubWalletID == "" {
		return g, errors.New("missing sub_wallet_id")
	}
	err := a.svc.store.db.QueryRow(ctx,
		`SELECT address FROM sub_wallets WHERE id = $1 AND master_wallet_id = $2 AND is_active = true`,
		tx.SubWalletID, tx.MasterWalletID).Scan(&g.SubWalletAddress)
	if err != nil {
		return g, fmt.Errorf("sub-wallet lookup failed: %w", err)
	}
	g.SubWalletFound = true
	_ = a.svc.store.db.QueryRow(ctx,
		`SELECT address FROM master_wallets WHERE id = $1`, tx.MasterWalletID).Scan(&g.MasterAddress)
	// Treasury/revenue destinations: the whitelist table can pin treasury-type
	// addresses per master wallet; treat them as forbidden auto-sign targets.
	g.TreasuryAddresses = map[string]bool{}
	rows, err := a.svc.store.db.Query(ctx,
		`SELECT address FROM whitelist WHERE master_wallet_id = $1 AND whitelist_type IN ('treasury','revenue','fee') AND is_enabled = true`,
		tx.MasterWalletID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var addr string
			if err := rows.Scan(&addr); err == nil {
				g.TreasuryAddresses[normalizeAddr(addr)] = true
			}
		}
	}
	return g, nil
}

// signAndBroadcast signs and broadcasts the tx with the user's sub-wallet key
// (derived from the master wallet seed at the sub-wallet's derivation path).
// EVM: real EIP-1559 sign + eth_sendRawTransaction. Non-EVM: the existing
// non_evm_crypto.go signers (Solana Ed25519, Bitcoin secp256k1, Cosmos
// secp256k1) which return the signature/tx-id with status "signed".
func (a *AutoSigner) signAndBroadcast(ctx context.Context, tx *pendingAutoTx) (string, string, error) {
	var encSeed, derivationPath string
	var masterChainID int64
	err := a.svc.store.db.QueryRow(ctx,
		`SELECT mw.encrypted_seed, sw.derivation_path, mw.chain_id
		 FROM sub_wallets sw JOIN master_wallets mw ON mw.id = sw.master_wallet_id
		 WHERE sw.id = $1`, tx.SubWalletID).Scan(&encSeed, &derivationPath, &masterChainID)
	if err != nil {
		return "", "approved", fmt.Errorf("load signing material: %w", err)
	}
	seed, err := DecryptSeed(encSeed, a.password)
	if err != nil {
		return "", "approved", errors.New("seed decryption failed (check MASTER_AUTO_SIGN_PASSWORD)")
	}
	chainID := tx.ChainID
	if chainID == 0 {
		chainID = masterChainID
	}

	switch strings.ToLower(tx.Blockchain) {
	case "solana", "bitcoin", "btc", "cosmos", "osmosis", "atom":
		return a.signNonEVM(tx, seed, chainID, derivationPath)
	}

	// EVM path (default): real nonce + EIP-1559 fees + local secp256k1 sign +
	// eth_sendRawTransaction broadcast.
	rpc := rpcEndpointForChain(chainID)
	if rpc == "" {
		rpc = a.svc.getUserChainRPC(chainID, "evm")
	}
	if rpc == "" {
		return "", "approved", fmt.Errorf("no RPC endpoint for chain_id %d", chainID)
	}
	privKey, err := DerivePrivateKeyFromPath(seed, derivationPath)
	if err != nil {
		return "", "approved", fmt.Errorf("key derivation: %w", err)
	}
	from := PrivateKeyToAddress(privKey)
	if normalizeAddr(from.Hex()) != normalizeAddr(tx.FromAddress) {
		// Never sign from an address that isn't the recorded user sub-wallet.
		return "", "approved", errors.New("derived key does not match the sub-wallet address; refusing to sign")
	}
	rpcCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	nonce, err := FetchTransactionCount(rpcCtx, rpc, from)
	if err != nil {
		return "", "approved", fmt.Errorf("nonce: %w", err)
	}
	_, maxFee, prioFee, err := FetchGasPrice(rpcCtx, rpc)
	if err != nil {
		return "", "approved", fmt.Errorf("gas price: %w", err)
	}
	toAddr := common.HexToAddress(tx.ToAddress)
	value, _ := new(big.Int).SetString(strings.TrimSpace(tx.Amount), 10)
	if value == nil {
		value = big.NewInt(0)
	}
	gasLimit := uint64(21000)
	data := []byte(nil)
	if tx.TokenAddress != "" && tx.TokenAddress != "0x0000000000000000000000000000000000000000" {
		// ERC-20 transfer: send 0 native value to the token contract with
		// transfer(to, amount) calldata.
		data = erc20TransferCalldata(toAddr, value)
		toAddr = common.HexToAddress(tx.TokenAddress)
		value = big.NewInt(0)
		gasLimit = 65000
	}
	rawTx, err := SignEVMTransaction(big.NewInt(chainID), nonce, toAddr, value, gasLimit, maxFee, prioFee, data, privKey)
	if err != nil {
		return "", "approved", fmt.Errorf("sign: %w", err)
	}
	txHash, err := BroadcastTransaction(rpcCtx, rpc, rawTx)
	if err != nil {
		return "", "approved", fmt.Errorf("broadcast: %w", err)
	}
	return txHash, "submitted", nil
}

// signNonEVM reuses the existing non-EVM signers (non_evm_crypto.go) through
// the same AutoSignRequest shape the HTTP endpoints use.
func (a *AutoSigner) signNonEVM(tx *pendingAutoTx, seed []byte, chainID int64, derivationPath string) (string, string, error) {
	req := &AutoSignRequest{
		ChainID:        chainID,
		ChainType:      strings.ToLower(tx.Blockchain),
		DerivationPath: derivationPath,
		TxType:         tx.TxType,
		ToAddress:      tx.ToAddress,
		Value:          tx.Amount,
		TokenAddress:   tx.TokenAddress,
	}
	switch strings.ToLower(tx.Blockchain) {
	case "solana":
		return a.svc.autoSignSolana(seed, req)
	case "bitcoin", "btc":
		return a.svc.autoSignBitcoin(seed, req)
	case "cosmos", "osmosis", "atom":
		return a.svc.autoSignCosmos(seed, req)
	default:
		return "", "approved", fmt.Errorf("unsupported non-EVM blockchain %q", tx.Blockchain)
	}
}
