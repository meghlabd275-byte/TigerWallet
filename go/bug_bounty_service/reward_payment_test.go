package main

// reward_payment_test.go — Exercises the reward payout path.
//
// The previous implementation wrote a transaction hash derived from the report
// id and decremented the reward pool without any funds moving. These tests pin
// the replacement: a report only reaches "rewarded" once a real 32-byte hash is
// recorded, the pool only moves at that point, and a placeholder is rejected.

import (
	"math/big"
	"testing"
)

const (
	validHash = "0x1111111111111111111111111111111111111111111111111111111111111111"
	otherHash = "0x2222222222222222222222222222222222222222222222222222222222222222"
)

func newTestBountyService(pool string) *BugBountyService {
	s := &BugBountyService{
		programs:   map[string]*BugBountyProgram{},
		reports:    map[string]*BugReport{},
		hackers:    map[string]*Hacker{},
		rewardPool: new(big.Int),
	}
	s.rewardPool.SetString(pool, 10)
	return s
}

func seedReport(s *BugBountyService, id, reward string) *BugReport {
	r := &BugReport{
		ID:           id,
		ProgramID:    "p1",
		Severity:     "high",
		Status:       "accepted",
		RewardAmount: reward,
	}
	s.reports[id] = r
	return r
}

// TestRequestRewardPaymentDoesNotFabricateHash is the core regression: the old
// code set report.TxHash to "0x" + hex(report.ID) and marked the report
// rewarded. Neither may happen now.
func TestRequestRewardPaymentDoesNotFabricateHash(t *testing.T) {
	s := newTestBountyService("1000000000000000000000")
	r := seedReport(s, "report-abc", "1000000000000000000")

	s.requestRewardPayment(r)

	if r.TxHash != "" {
		t.Errorf("requestRewardPayment set tx_hash to %q; no payout was broadcast", r.TxHash)
	}
	if r.Status != "payment_pending" {
		t.Errorf("status = %q, want payment_pending", r.Status)
	}
	// The pool must be untouched: no funds moved.
	if s.rewardPool.String() != "1000000000000000000000" {
		t.Errorf("reward pool moved to %s without a broadcast", s.rewardPool)
	}
	// The fabricated hash was derived from the id, so it is predictable. Make
	// sure nothing resembling it is present.
	if r.TxHash == "0x"+r.ID {
		t.Error("tx_hash is still derived from the report id")
	}
}

// TestRecordRewardTxHashRejectsInvalidHashes: a placeholder must never be
// accepted as proof of payment.
func TestRecordRewardTxHashRejectsInvalidHashes(t *testing.T) {
	bad := []struct {
		name string
		hash string
	}{
		{"empty", ""},
		{"short", "0x1234"},
		{"not hex", "0xzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"},
		{"no prefix", "1111111111111111111111111111111111111111111111111111111111111111"},
		{"too long", "0x111111111111111111111111111111111111111111111111111111111111111111"},
		{"report id", "report-abc"},
	}

	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestBountyService("1000000000000000000000")
			r := seedReport(s, "report-abc", "1000000000000000000")
			s.requestRewardPayment(r)

			if err := s.RecordRewardTxHash("report-abc", tc.hash); err == nil {
				t.Errorf("RecordRewardTxHash accepted %q", tc.hash)
			}
			if r.Status == "rewarded" {
				t.Error("report became rewarded despite an invalid hash")
			}
			if s.rewardPool.String() != "1000000000000000000000" {
				t.Errorf("reward pool moved despite an invalid hash: %s", s.rewardPool)
			}
		})
	}
}

// TestRecordRewardTxHashPaysExactlyOnce: recording the real hash moves the pool
// and the program total once, and a second call is refused so the same reward
// cannot be paid twice.
func TestRecordRewardTxHashPaysExactlyOnce(t *testing.T) {
	s := newTestBountyService("1000000000000000000000")
	s.programs["p1"] = &BugBountyProgram{ID: "p1", PaidOut: "0"}
	r := seedReport(s, "report-abc", "1000000000000000000")

	s.requestRewardPayment(r)

	if err := s.RecordRewardTxHash("report-abc", validHash); err != nil {
		t.Fatalf("RecordRewardTxHash returned %v, want nil", err)
	}

	if r.Status != "rewarded" {
		t.Errorf("status = %q, want rewarded", r.Status)
	}
	if r.TxHash != validHash {
		t.Errorf("tx_hash = %q, want %q", r.TxHash, validHash)
	}
	// 1000 - 1 = 999 ETH in wei.
	if s.rewardPool.String() != "999000000000000000000" {
		t.Errorf("reward pool = %s, want 999000000000000000000", s.rewardPool)
	}
	if s.programs["p1"].PaidOut != "1000000000000000000" {
		t.Errorf("program paid_out = %s, want 1000000000000000000", s.programs["p1"].PaidOut)
	}

	// A second attempt with a different hash must not pay again.
	if err := s.RecordRewardTxHash("report-abc", otherHash); err == nil {
		t.Error("RecordRewardTxHash accepted a second payout for the same report")
	}
	if s.rewardPool.String() != "999000000000000000000" {
		t.Errorf("reward pool moved twice: %s", s.rewardPool)
	}
}

// TestRecordRewardTxHashRefusesUnpayableState: a report that was never moved to
// payment_pending cannot be paid, so an attacker who can set a status cannot
// drain the pool by skipping the approval step.
func TestRecordRewardTxHashRefusesUnpayableState(t *testing.T) {
	for _, status := range []string{"submitted", "triaged", "rejected", "fixed"} {
		t.Run(status, func(t *testing.T) {
			s := newTestBountyService("1000000000000000000000")
			r := seedReport(s, "report-abc", "1000000000000000000")
			r.Status = status

			if err := s.RecordRewardTxHash("report-abc", validHash); err == nil {
				t.Errorf("RecordRewardTxHash paid a %q report", status)
			}
			if s.rewardPool.String() != "1000000000000000000000" {
				t.Errorf("reward pool moved for a %q report: %s", status, s.rewardPool)
			}
		})
	}
}

// TestRecordRewardTxHashRefusesUnknownReport.
func TestRecordRewardTxHashRefusesUnknownReport(t *testing.T) {
	s := newTestBountyService("1000000000000000000000")
	if err := s.RecordRewardTxHash("does-not-exist", validHash); err == nil {
		t.Error("RecordRewardTxHash accepted an unknown report id")
	}
}

// TestRecordRewardTxHashRefusesOverdraw: a reward larger than the pool must not
// leave the pool negative.
func TestRecordRewardTxHashRefusesOverdraw(t *testing.T) {
	s := newTestBountyService("1000")
	r := seedReport(s, "report-big", "5000")
	s.requestRewardPayment(r)

	if err := s.RecordRewardTxHash("report-big", validHash); err == nil {
		t.Error("RecordRewardTxHash paid a reward larger than the pool")
	}
	if s.rewardPool.Sign() < 0 {
		t.Errorf("reward pool went negative: %s", s.rewardPool)
	}
}

// TestRecordRewardTxHashRefusesUnparseableAmount: a malformed reward amount
// must not be treated as zero, which would silently pay nothing while marking
// the report rewarded.
func TestRecordRewardTxHashRefusesUnparseableAmount(t *testing.T) {
	s := newTestBountyService("1000000000000000000000")
	r := seedReport(s, "report-bad", "not-a-number")
	s.requestRewardPayment(r)

	if err := s.RecordRewardTxHash("report-bad", validHash); err == nil {
		t.Error("RecordRewardTxHash accepted an unparseable reward amount")
	}
	if r.Status == "rewarded" {
		t.Error("report became rewarded with an unparseable amount")
	}
}

// TestIsTxHash checks the validator directly, including the boundary lengths.
func TestIsTxHash(t *testing.T) {
	if !isTxHash(validHash) {
		t.Error("a 32-byte 0x hash was rejected")
	}
	if !isTxHash("0X" + validHash[2:]) {
		t.Error("an uppercase 0X prefix was rejected")
	}
	for _, bad := range []string{"", "0x", "0x12", validHash[:65], validHash + "1"} {
		if isTxHash(bad) {
			t.Errorf("isTxHash accepted %q", bad)
		}
	}
}

// TestStatusTransitionMachine pins the report lifecycle. Before this existed any
// status could be written, so a caller could jump straight to "rewarded".
func TestStatusTransitionMachine(t *testing.T) {
	allowed := []struct{ from, to string }{
		{"submitted", "triaged"},
		{"submitted", "accepted"},
		{"submitted", "rejected"},
		{"triaged", "accepted"},
		{"triaged", "rejected"},
		{"accepted", "payment_pending"},
		{"accepted", "rejected"},
		{"accepted", "fixed"},
		{"payment_pending", "accepted"},
		{"payment_pending", "rejected"},
		// Re-applying the current status is a no-op, not an error.
		{"submitted", "submitted"},
		{"accepted", "accepted"},
	}
	for _, tc := range allowed {
		if err := validateStatusTransition(tc.from, tc.to); err != nil {
			t.Errorf("transition %s -> %s rejected: %v", tc.from, tc.to, err)
		}
	}

	denied := []struct{ from, to string }{
		// The payout state is unreachable through a status write.
		{"submitted", "rewarded"},
		{"triaged", "rewarded"},
		{"accepted", "rewarded"},
		{"payment_pending", "rewarded"},
		{"rewarded", "rewarded"},
		// Terminal states cannot be left.
		{"rejected", "accepted"},
		{"fixed", "accepted"},
		{"rewarded", "accepted"},
		// Cannot skip triage.
		{"submitted", "payment_pending"},
		{"triaged", "payment_pending"},
		// Unknown source state.
		{"bogus", "accepted"},
	}
	for _, tc := range denied {
		if err := validateStatusTransition(tc.from, tc.to); err == nil {
			t.Errorf("transition %s -> %s was allowed; it must be refused", tc.from, tc.to)
		}
	}
}
