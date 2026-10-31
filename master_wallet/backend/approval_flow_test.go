package main

import (
	"strings"
	"testing"
)

// The approve/reject routes are /:id/transactions/:tid/{approve,reject}, so :id
// is the master wallet and :tid is the transaction. Both handlers previously read
// c.Param("id") as the transaction id, which meant the INSERT into
// transaction_signatures referenced a master wallet id and violated the foreign
// key, so no approval or rejection was ever recorded.
func TestApprovalHandlersReadTheTransactionParam(t *testing.T) {
	for _, handler := range []string{"ApproveTransaction", "RejectTransaction"} {
		got := paramLiterals(t, "handlers.go", handler)
		if !got["tid"] {
			t.Errorf("%s must read c.Param(\"tid\"); it reads %v", handler, keys(got))
		}
		if got["id"] {
			t.Errorf("%s must not read c.Param(\"id\") as the transaction id - that is the master wallet id", handler)
		}
	}
}

// An approval must be counted from the signature rows. The previous code ran
// `current_approvals = current_approvals + 1` on every call and the signature
// upsert used DO UPDATE, so one signer could call approve repeatedly and drive
// the counter to the threshold on their own.
func TestApprovalIsIdempotentPerSigner(t *testing.T) {
	src := funcBody(t, "handlers.go", "ApproveTransaction")
	if !strings.Contains(src, "DO NOTHING") {
		t.Error("ApproveTransaction must upsert the signature with DO NOTHING so a repeat call from the same signer is a no-op")
	}
	if strings.Contains(src, "DO UPDATE SET signature_status='signed'") {
		t.Error("ApproveTransaction still uses DO UPDATE for the signature, which lets one signer approve repeatedly")
	}
	if strings.Contains(src, "current_approvals = current_approvals + 1") {
		t.Error("ApproveTransaction still increments the approval counter blindly instead of deriving it from the signature rows")
	}
	if !strings.Contains(src, "COUNT(*) FROM transaction_signatures") {
		t.Error("ApproveTransaction must derive current_approvals from the signature rows")
	}
	if !strings.Contains(src, "RowsAffected()") {
		t.Error("ApproveTransaction must use the insert's affected row count to detect a repeat approval")
	}
}

// Every approval and rejection must be scoped to the master wallet in the path,
// otherwise a user could act on another tenant's transaction by id.
func TestApprovalHandlersAreMasterWalletScoped(t *testing.T) {
	for _, handler := range []string{"ApproveTransaction", "RejectTransaction"} {
		src := funcBody(t, "handlers.go", handler)
		if !strings.Contains(src, "requireMasterWalletAccess") {
			t.Errorf("%s does not call requireMasterWalletAccess", handler)
		}
		if !strings.Contains(src, "master_wallet_id = $2") {
			t.Errorf("%s does not constrain the transaction by master_wallet_id", handler)
		}
	}
}

// The auto-signer resolves the same approval row. It must also derive the count
// rather than increment, so a transaction that already carries a manual approval
// is not double-counted.
func TestAutoSignerApprovalIsNotDoubleCounted(t *testing.T) {
	src := funcBody(t, "auto_signer.go", "processTx")
	if src == "" {
		t.Fatal("processTx not found in auto_signer.go")
	}
	if strings.Contains(src, "current_approvals = current_approvals + 1") {
		t.Error("the auto-signer still increments the approval counter blindly")
	}
	if !strings.Contains(src, "COUNT(*) FROM transaction_signatures") {
		t.Error("the auto-signer must derive current_approvals from the signature rows")
	}
}
