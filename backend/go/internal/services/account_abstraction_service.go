package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/backend/internal/models"
)

type AccountAbstractionService struct {
	db *sql.DB
}

func NewAccountAbstractionService(db *sql.DB) *AccountAbstractionService {
	return &AccountAbstractionService{db: db}
}

// Create smart contract wallet (account abstraction)
func (s *AccountAbstractionService) CreateAccount(ctx context.Context, userID uuid.UUID, ownerAddress string, salt string) (*models.SmartAccount, error) {
	// Derive account address from owner and salt
	accountAddress := deriveSmartAccountAddress(ownerAddress, salt)

	account := &models.SmartAccount{
		ID:             uuid.New(),
		UserID:         userID,
		AccountAddress: accountAddress,
		OwnerAddress:   ownerAddress,
		Nonce:          0,
		Threshold:      1,
		Status:         "ACTIVE",
		Deployed:       false,
		CreatedAt:      time.Now(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO smart_accounts (id, user_id, account_address, owner_address, nonce, threshold, status, deployed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, account.ID, account.UserID, account.AccountAddress, account.OwnerAddress, 
		account.Nonce, account.Threshold, account.Status, account.Deployed, account.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return account, nil
}

// Execute user operation (meta-transaction)
func (s *AccountAbstractionService) ExecuteUserOp(ctx context.Context, userOp *models.UserOperation) (*models.UserOperation, error) {
	// Verify signature
	if !s.verifySignature(userOp) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Get account
	var account models.SmartAccount
	err := s.db.QueryRowContext(ctx, `
		SELECT id, nonce, deployed FROM smart_accounts 
		WHERE account_address = $1 AND user_id = $2
	`, userOp.Sender, userOp.UserID).Scan(&account.ID, &account.Nonce, &account.Deployed)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found")
	}
	if err != nil {
		return nil, err
	}

	// Check nonce
	if userOp.Nonce != account.Nonce {
		return nil, fmt.Errorf("invalid nonce")
	}

	// In production:
	// 1. Simulate the transaction
	// 2. Check paymaster sponsorship
	// 3. Execute via EntryPoint contract
	// 4. Update nonce

	// Update nonce
	_, err = s.db.ExecContext(ctx, `
		UPDATE smart_accounts SET nonce = nonce + 1 WHERE account_address = $1
	`, userOp.Sender)

	if err != nil {
		return nil, fmt.Errorf("failed to update nonce: %w", err)
	}

	// Mark userOp as confirmed
	userOp.Status = "CONFIRMED"
	userOp.ConfirmedAt = time.Now()

	return userOp, nil
}

// Add signer (multisig support)
func (s *AccountAbstractionService) AddSigner(ctx context.Context, userID uuid.UUID, accountAddress, signerAddress string, weight int) error {
	signerID := uuid.New()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO account_signers (id, user_id, account_address, signer_address, weight, status, added_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, signerID, userID, accountAddress, signerAddress, weight, "ACTIVE", time.Now())

	return err
}

// Remove signer
func (s *AccountAbstractionService) RemoveSigner(ctx context.Context, userID uuid.UUID, accountAddress, signerAddress string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE account_signers SET status = 'REMOVED'
		WHERE account_address = $1 AND signer_address = $2 AND user_id = $3
	`, accountAddress, signerAddress, userID)

	return err
}

// Get account signers
func (s *AccountAbstractionService) GetSigners(ctx context.Context, accountAddress string) ([]models.AccountSigner, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, signer_address, weight, status FROM account_signers 
		WHERE account_address = $1 AND status = 'ACTIVE'
	`, accountAddress)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signers []models.AccountSigner
	for rows.Next() {
		var signer models.AccountSigner
		err := rows.Scan(&signer.ID, &signer.SignerAddress, &signer.Weight, &signer.Status)
		if err != nil {
			continue
		}
		signers = append(signers, signer)
	}

	return signers, nil
}

// Set up session key (for recurring transactions)
func (s *AccountAbstractionService) AddSessionKey(ctx context.Context, userID uuid.UUID, accountAddress, sessionKey string, permissions string, expiresAt time.Time) error {
	sessionID := uuid.New()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO session_keys (id, user_id, account_address, session_key, permissions, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, sessionID, userID, accountAddress, sessionKey, permissions, expiresAt)

	return err
}

// Verify session key
func (s *AccountAbstractionService) VerifySessionKey(ctx context.Context, accountAddress, sessionKey string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM session_keys 
		WHERE account_address = $1 AND session_key = $2 AND expires_at > NOW()
	`, accountAddress, sessionKey).Scan(&count)

	return count > 0, err
}

// Get user's smart accounts
func (s *AccountAbstractionService) GetUserAccounts(ctx context.Context, userID uuid.UUID) ([]models.SmartAccount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_address, owner_address, nonce, threshold, status, deployed, created_at
		FROM smart_accounts WHERE user_id = $1
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.SmartAccount
	for rows.Next() {
		var a models.SmartAccount
		err := rows.Scan(&a.ID, &a.AccountAddress, &a.OwnerAddress, 
			&a.Nonce, &a.Threshold, &a.Status, &a.Deployed, &a.CreatedAt)
		if err != nil {
			continue
		}
		accounts = append(accounts, a)
	}

	return accounts, nil
}

// Get user operations history
func (s *AccountAbstractionService) GetUserOps(ctx context.Context, accountAddress string) ([]models.UserOperation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_op_hash, sender, nonce, init_code, call_data, call_gas_limit, verification_gas_limit, pre_verification_gas, max_fee_per_gas, max_priority_fee_per_gas, signature, status, created_at
		FROM user_operations WHERE sender = $1 ORDER BY created_at DESC LIMIT 50
	`, accountAddress)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []models.UserOperation
	for rows.Next() {
		var op models.UserOperation
		err := rows.Scan(&op.ID, &op.UserOpHash, &op.Sender, &op.Nonce, 
			&op.InitCode, &op.CallData, &op.CallGasLimit, &op.VerificationGasLimit,
			&op.PreVerificationGas, &op.MaxFeePerGas, &op.MaxPriorityFeePerGas,
			&op.Signature, &op.Status, &op.CreatedAt)
		if err != nil {
			continue
		}
		ops = append(ops, op)
	}

	return ops, nil
}

// Deploy account (bundle with first transaction)
func (s *AccountAbstractionService) DeployAccount(ctx context.Context, userID uuid.UUID, accountAddress string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE smart_accounts SET deployed = true WHERE account_address = $1 AND user_id = $2
	`, accountAddress, userID)

	return err
}

// Enable paymaster sponsorship
func (s *AccountAbstractionService) EnablePaymaster(ctx context.Context, userID uuid.UUID, accountAddress, paymasterAddress string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO paymaster_sponsors (id, user_id, account_address, paymaster_address, enabled_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, uuid.New(), userID, accountAddress, paymasterAddress)

	return err
}

// Helper functions
func deriveSmartAccountAddress(owner, salt string) string {
	data := owner + salt
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:20])
}

func (s *AccountAbstractionService) verifySignature(userOp *models.UserOperation) bool {
	// In production, verify EIP-4337 signature
	// Hash the user operation and verify against owner's signature
	data := fmt.Sprintf("%s%d%s%s", userOp.Sender, userOp.Nonce, userOp.CallData, userOp.Signature)
	hash := sha256.Sum256([]byte(data))
	
	// Simplified - in production use proper signature verification
	return len(userOp.Signature) > 0
}

func parseBigInt(s string) *big.Int {
	i, _ := new(big.Int).SetString(s, 0)
	return i
}
