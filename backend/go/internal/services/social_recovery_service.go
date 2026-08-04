package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/backend/internal/models"
)

type SocialRecoveryService struct {
	db *sql.DB
}

func NewSocialRecoveryService(db *sql.DB) *SocialRecoveryService {
	return &SocialRecoveryService{db: db}
}

// Setup social recovery for user
func (s *SocialRecoveryService) SetupRecovery(ctx context.Context, userID uuid.UUID, guardians []models.Guardian) error {
	if len(guardians) < 3 {
		return fmt.Errorf("minimum 3 guardians required")
	}
	if len(guardians) > 5 {
		return fmt.Errorf("maximum 5 guardians allowed")
	}

	// Generate recovery key
	recoveryKey := generateRecoveryKey()
	hashedKey := hashRecoveryKey(recoveryKey)

	// Create recovery setup
	setup := &models.RecoverySetup{
		ID:            uuid.New(),
		UserID:        userID,
		RecoveryKey:   hashedKey,
		Threshold:     len(guardians), // Require all guardians
		Status:        "ACTIVE",
		GuardianCount: len(guardians),
		CreatedAt:     time.Now(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO recovery_setups (id, user_id, recovery_key, threshold, status, guardian_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, setup.ID, setup.UserID, setup.RecoveryKey, setup.Threshold, setup.Status, setup.GuardianCount, setup.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create recovery setup: %w", err)
	}

	// Add guardians
	for _, guardian := range guardians {
		guardianID := uuid.New()
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO guardians (id, user_id, address, name, relationship, status, added_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, guardianID, userID, guardian.Address, guardian.Name, guardian.Relationship, "PENDING", time.Now())

		if err != nil {
			return fmt.Errorf("failed to add guardian: %w", err)
		}
	}

	return nil
}

// Initiate recovery process
func (s *SocialRecoveryService) InitiateRecovery(ctx context.Context, lostUserID uuid.UUID, guardianID uuid.UUID, guardianSignature string) error {
	// Verify guardian
	var guardianAddress string
	err := s.db.QueryRowContext(ctx, `
		SELECT address FROM guardians WHERE id = $1 AND user_id = $2 AND status = 'ACTIVE'
	`, guardianID, lostUserID).Scan(&guardianAddress)

	if err == sql.ErrNoRows {
		return fmt.Errorf("guardian not found or not authorized")
	}

	// Verify signature
	if !verifyGuardianSignature(guardianAddress, lostUserID.String(), guardianSignature) {
		return fmt.Errorf("invalid guardian signature")
	}

	// Create recovery request
	request := &models.RecoveryRequest{
		ID:          uuid.New(),
		UserID:      lostUserID,
		GuardianID:  guardianID,
		Status:      "PENDING",
		InitiatedAt: time.Now(),
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO recovery_requests (id, user_id, guardian_id, status, initiated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, request.ID, request.UserID, request.GuardianID, request.Status, request.InitiatedAt)

	return err
}

// Confirm recovery (called by additional guardians)
func (s *SocialRecoveryService) ConfirmRecovery(ctx context.Context, requestID uuid.UUID, guardianID uuid.UUID, guardianSignature string) error {
	// Get request
	var userID uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id FROM recovery_requests WHERE id = $1 AND status = 'PENDING'
	`, requestID).Scan(&userID)

	if err == sql.ErrNoRows {
		return fmt.Errorf("recovery request not found")
	}

	// Verify guardian
	var guardianAddress string
	err = s.db.QueryRowContext(ctx, `
		SELECT address FROM guardians WHERE id = $1 AND user_id = $2 AND status = 'ACTIVE'
	`, guardianID, userID).Scan(&guardianAddress)

	if err != nil {
		return fmt.Errorf("guardian not authorized")
	}

	// Verify signature
	if !verifyGuardianSignature(guardianAddress, requestID.String(), guardianSignature) {
		return fmt.Errorf("invalid signature")
	}

	// Record confirmation
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO recovery_confirmations (id, request_id, guardian_id, confirmed_at)
		VALUES ($1, $2, $3, NOW())
	`, uuid.New(), requestID, guardianID)

	if err != nil {
		return fmt.Errorf("failed to confirm: %w", err)
	}

	// Check if threshold met
	var confirmationCount int
	s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM recovery_confirmations WHERE request_id = $1
	`, requestID).Scan(&confirmationCount)

	// Get required threshold
	var threshold int
	s.db.QueryRowContext(ctx, `
		SELECT threshold FROM recovery_setups WHERE user_id = $1
	`, userID).Scan(&threshold)

	if confirmationCount >= threshold {
		// Complete recovery
		s.db.ExecContext(ctx, `
			UPDATE recovery_requests SET status = 'COMPLETED', completed_at = NOW() WHERE id = $1
		`, requestID)
	}

	return nil
}

// Execute recovery with full key
func (s *SocialRecoveryService) ExecuteRecovery(ctx context.Context, userID uuid.UUID, recoveryKey string) error {
	// Verify recovery key
	hashedKey := hashRecoveryKey(recoveryKey)
	var storedKey string
	err := s.db.QueryRowContext(ctx, `
		SELECT recovery_key FROM recovery_setups WHERE user_id = $1 AND status = 'ACTIVE'
	`, userID).Scan(&storedKey)

	if err == sql.ErrNoRows {
		return fmt.Errorf("recovery not set up")
	}

	if storedKey != hashedKey {
		return fmt.Errorf("invalid recovery key")
	}

	// Get all recovery requests
	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM recovery_requests WHERE user_id = $1 AND status = 'PENDING'
	`, userID)

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var requestID uuid.UUID
		rows.Scan(&requestID)
		s.db.ExecContext(ctx, `
			UPDATE recovery_requests SET status = 'COMPLETED', completed_at = NOW() WHERE id = $1
		`, requestID)
	}

	return nil
}

// Get guardians for user
func (s *SocialRecoveryService) GetGuardians(ctx context.Context, userID uuid.UUID) ([]models.Guardian, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, address, name, relationship, status, added_at
		FROM guardians WHERE user_id = $1
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guardians []models.Guardian
	for rows.Next() {
		var g models.Guardian
		err := rows.Scan(&g.ID, &g.Address, &g.Name, &g.Relationship, &g.Status, &g.AddedAt)
		if err != nil {
			continue
		}
		guardians = append(guardians, g)
	}

	return guardians, nil
}

// Add guardian
func (s *SocialRecoveryService) AddGuardian(ctx context.Context, userID uuid.UUID, guardian models.Guardian) error {
	// Check current guardian count
	var count int
	s.db.QueryRowContext(ctx, `
		SELECT guardian_count FROM recovery_setups WHERE user_id = $1
	`, userID).Scan(&count)

	if count >= 5 {
		return fmt.Errorf("maximum guardians reached")
	}

	guardianID := uuid.New()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO guardians (id, user_id, address, name, relationship, status, added_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, guardianID, userID, guardian.Address, guardian.Name, guardian.Relationship, "ACTIVE", time.Now())

	if err != nil {
		return fmt.Errorf("failed to add guardian: %w", err)
	}

	// Update count
	s.db.ExecContext(ctx, `
		UPDATE recovery_setups SET guardian_count = guardian_count + 1 WHERE user_id = $1
	`, userID)

	return nil
}

// Remove guardian
func (s *SocialRecoveryService) RemoveGuardian(ctx context.Context, userID uuid.UUID, guardianID uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE guardians SET status = 'REMOVED' 
		WHERE id = $1 AND user_id = $2
	`, guardianID, userID)

	if err != nil {
		return fmt.Errorf("failed to remove guardian: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("guardian not found")
	}

	// Update count
	s.db.ExecContext(ctx, `
		UPDATE recovery_setups SET guardian_count = guardian_count - 1 WHERE user_id = $1
	`, userID)

	return nil
}

// Helper functions
func generateRecoveryKey() string {
	data := fmt.Sprintf("recovery_%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func hashRecoveryKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func verifyGuardianSignature(guardianAddress, message, signature string) bool {
	// In production, verify the ECDSA signature
	// For now, just check signature format
	return len(signature) > 10
}
