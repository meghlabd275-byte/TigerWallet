package main

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

// signApproval produces the EIP-191 personal-sign signature a guardian sends.
func signApproval(t *testing.T, privateKeyHex, message string) string {
	t.Helper()

	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		t.Fatalf("cannot parse the test key: %v", err)
	}

	sig, err := crypto.Sign(accounts.TextHash([]byte(message)), key)
	if err != nil {
		t.Fatalf("cannot sign the approval: %v", err)
	}
	return "0x" + hex.EncodeToString(sig)
}

// TestRecoverPersonalSignerMatchesTheGuardian checks that a signature produced
// by the guardian's key recovers to the guardian's address, and that a
// signature produced by anyone else does not.
func TestRecoverPersonalSignerMatchesTheGuardian(t *testing.T) {
	guardianKey := "4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"
	attackerKey := "3a6e1b6ac1c9c7e5b8a2f4d9e7c3b1a5f8d2e4c6a9b7d5f3e1c8a6b4d2f0e8c6"

	guardian, err := crypto.HexToECDSA(guardianKey)
	if err != nil {
		t.Fatalf("cannot parse the guardian key: %v", err)
	}
	guardianAddress := crypto.PubkeyToAddress(guardian.PublicKey).Hex()

	request := &RecoveryRequest{
		ID:              "req-1",
		WalletID:        "0x1111111111111111111111111111111111111111",
		NewOwnerAddress: "0x2222222222222222222222222222222222222222",
	}
	message := recoveryApprovalMessage(request)

	t.Run("a genuine signature recovers to the guardian", func(t *testing.T) {
		signer, err := recoverPersonalSigner(message, signApproval(t, guardianKey, message))
		if err != nil {
			t.Fatalf("a valid signature was rejected: %v", err)
		}
		if !strings.EqualFold(signer, guardianAddress) {
			t.Fatalf("recovered %s, want the guardian %s", signer, guardianAddress)
		}
	})

	t.Run("another key recovers to a different address", func(t *testing.T) {
		signer, err := recoverPersonalSigner(message, signApproval(t, attackerKey, message))
		if err != nil {
			t.Fatalf("a well-formed signature was rejected: %v", err)
		}
		if strings.EqualFold(signer, guardianAddress) {
			t.Fatal("a signature from a different key recovered to the guardian")
		}
	})

	t.Run("malformed signatures are rejected", func(t *testing.T) {
		cases := map[string]string{
			"not hex":      "0xzz",
			"too short":    "0x1234",
			"not prefixed": "",
		}
		for name, sig := range cases {
			if _, err := recoverPersonalSigner(message, sig); err == nil {
				t.Errorf("%s: expected an error, got none", name)
			}
		}
	})
}

// TestApprovalMessageBindsTheRequestAndNewOwner checks that an approval for one
// recovery does not verify against a different request or a different new
// owner. Without that binding, a signature harvested from any past recovery
// could be replayed to hand the wallet to an attacker.
func TestApprovalMessageBindsTheRequestAndNewOwner(t *testing.T) {
	key := "4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"

	original := &RecoveryRequest{
		ID:              "req-1",
		WalletID:        "0x1111111111111111111111111111111111111111",
		NewOwnerAddress: "0x2222222222222222222222222222222222222222",
	}
	replayed := &RecoveryRequest{
		ID:              "req-1",
		WalletID:        "0x1111111111111111111111111111111111111111",
		NewOwnerAddress: "0x3333333333333333333333333333333333333333",
	}

	sig := signApproval(t, key, recoveryApprovalMessage(original))

	// The signature is genuine for the original message.
	if _, err := recoverPersonalSigner(recoveryApprovalMessage(original), sig); err != nil {
		t.Fatalf("the original message did not verify: %v", err)
	}

	// It must not be accepted once the new owner differs, because the recovered
	// address will no longer be the guardian.
	guardian, err := crypto.HexToECDSA(key)
	if err != nil {
		t.Fatalf("cannot parse the key: %v", err)
	}
	guardianAddress := crypto.PubkeyToAddress(guardian.PublicKey).Hex()

	signer, err := recoverPersonalSigner(recoveryApprovalMessage(replayed), sig)
	if err != nil {
		// Recovering to a nonsense address is also acceptable here.
		return
	}
	if strings.EqualFold(signer, guardianAddress) {
		t.Fatal("a signature for one new owner was accepted for a different new owner")
	}
}

// TestSignRecoveryRejectsUnauthenticatedApprovals is the regression test for the
// account-takeover bug: the service used to append any signature it was given
// and count it towards the threshold.
func TestSignRecoveryRejectsUnauthenticatedApprovals(t *testing.T) {
	guardianKey := "4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"
	attackerKey := "3a6e1b6ac1c9c7e5b8a2f4d9e7c3b1a5f8d2e4c6a9b7d5f3e1c8a6b4d2f0e8c6"

	guardian, err := crypto.HexToECDSA(guardianKey)
	if err != nil {
		t.Fatalf("cannot parse the guardian key: %v", err)
	}
	guardianAddress := crypto.PubkeyToAddress(guardian.PublicKey).Hex()

	service := newTestService(t, guardianAddress)

	wallet, err := service.CreateWallet(context.Background(), guardianAddress, "ethereum", 1,
		[]Guardian{{ID: "g1", Address: guardianAddress, Type: GuardianTypeAddress, Confirmed: true}})
	if err != nil {
		t.Fatalf("cannot create the wallet: %v", err)
	}

	request, err := service.InitiateRecovery(context.Background(), wallet.ID, "0x2222222222222222222222222222222222222222")
	if err != nil {
		t.Fatalf("cannot start the recovery: %v", err)
	}

	message := recoveryApprovalMessage(request)

	t.Run("an attacker cannot approve on the guardian's behalf", func(t *testing.T) {
		_, err := service.SignRecovery(context.Background(), request.ID, "g1",
			signApproval(t, attackerKey, message))
		if err == nil {
			t.Fatal("a signature from the wrong key was accepted as the guardian's approval")
		}
		if !strings.Contains(err.Error(), "was produced by") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("a garbage signature is rejected", func(t *testing.T) {
		if _, err := service.SignRecovery(context.Background(), request.ID, "g1", "0xdeadbeef"); err == nil {
			t.Fatal("a malformed signature was accepted")
		}
	})

	t.Run("the guardian's own signature is accepted and meets the threshold", func(t *testing.T) {
		approved, err := service.SignRecovery(context.Background(), request.ID, "g1",
			signApproval(t, guardianKey, message))
		if err != nil {
			t.Fatalf("the guardian's own approval was rejected: %v", err)
		}
		if !approved.ThresholdMet {
			t.Fatal("the threshold was not met after the only guardian approved")
		}
	})

	t.Run("the same guardian cannot approve twice", func(t *testing.T) {
		// The threshold is one, so the first approval already moved the request
		// out of pending and a second attempt is refused for that reason.
		_, err := service.SignRecovery(context.Background(), request.ID, "g1",
			signApproval(t, guardianKey, message))
		if err == nil {
			t.Fatal("a second approval from the same guardian was counted")
		}
	})

	t.Run("a guardian cannot approve twice at a higher threshold", func(t *testing.T) {
		// Two guardians and a threshold of two, so the request stays pending
		// after the first approval and the duplicate check is what answers.
		second := newTestService(t, guardianAddress)
		secondKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("cannot generate the second guardian key: %v", err)
		}
		secondAddress := crypto.PubkeyToAddress(secondKey.PublicKey).Hex()

		wallet, err := second.CreateWallet(context.Background(), guardianAddress, "ethereum", 2,
			[]Guardian{
				{ID: "g1", Address: guardianAddress, Type: GuardianTypeAddress, Confirmed: true},
				{ID: "g2", Address: secondAddress, Type: GuardianTypeAddress, Confirmed: true},
			})
		if err != nil {
			t.Fatalf("cannot create the wallet: %v", err)
		}

		pending, err := second.InitiateRecovery(context.Background(), wallet.ID,
			"0x2222222222222222222222222222222222222222")
		if err != nil {
			t.Fatalf("cannot start the recovery: %v", err)
		}
		pendingMessage := recoveryApprovalMessage(pending)

		if _, err := second.SignRecovery(context.Background(), pending.ID, "g1",
			signApproval(t, guardianKey, pendingMessage)); err != nil {
			t.Fatalf("the first approval was rejected: %v", err)
		}

		_, err = second.SignRecovery(context.Background(), pending.ID, "g1",
			signApproval(t, guardianKey, pendingMessage))
		if err == nil {
			t.Fatal("a second approval from the same guardian was counted")
		}
		if !strings.Contains(err.Error(), "already approved") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestExecuteRecoveryDoesNotFabricateCompletion is the regression test for the
// second bug: the service marked a recovery completed, and told every client
// the wallet had moved, without ever sending a transaction.
func TestExecuteRecoveryDoesNotFabricateCompletion(t *testing.T) {
	key := "4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"
	guardian, err := crypto.HexToECDSA(key)
	if err != nil {
		t.Fatalf("cannot parse the key: %v", err)
	}
	guardianAddress := crypto.PubkeyToAddress(guardian.PublicKey).Hex()

	service := newTestService(t, guardianAddress)

	wallet, err := service.CreateWallet(context.Background(), guardianAddress, "ethereum", 1,
		[]Guardian{{ID: "g1", Address: guardianAddress, Type: GuardianTypeAddress, Confirmed: true}})
	if err != nil {
		t.Fatalf("cannot create the wallet: %v", err)
	}

	request, err := service.InitiateRecovery(context.Background(), wallet.ID, "0x2222222222222222222222222222222222222222")
	if err != nil {
		t.Fatalf("cannot start the recovery: %v", err)
	}
	if _, err := service.SignRecovery(context.Background(), request.ID, "g1",
		signApproval(t, key, recoveryApprovalMessage(request))); err != nil {
		t.Fatalf("cannot approve: %v", err)
	}

	// Move the clock past the delay rather than waiting a day or mutating the
	// stored request.
	service.now = func() time.Time { return time.Now().Add(48 * time.Hour) }

	result, err := service.ExecuteRecovery(context.Background(), request.ID)
	if err == nil {
		t.Fatal("execution succeeded with no configured contract; it must fail closed")
	}
	if !strings.Contains(err.Error(), "not configured to broadcast") {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil && result.Status == RecoveryCompleted {
		t.Fatal("the recovery was reported completed without an on-chain transaction")
	}
	if result != nil && result.ExecutedAt != nil {
		t.Fatal("an execution timestamp was recorded without an on-chain transaction")
	}
}

// newTestService builds a service with no Redis and no on-chain configuration,
// which is the state that must fail closed.
func newTestService(t *testing.T, guardianAddress string) *SocialRecoveryService {
	t.Helper()

	service, err := NewSocialRecoveryService(&Config{
		Port:          "0",
		RecoveryDelay: time.Hour,
		MinGuardians:  1,
		MaxGuardians:  10,
	})
	if err != nil {
		t.Fatalf("cannot build the service: %v", err)
	}
	return service
}
