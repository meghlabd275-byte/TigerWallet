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

type MPCWalletService struct {
	db *sql.DB
}

func NewMPCWalletService(db *sql.DB) *MPCWalletService {
	return &MPCWalletService{db: db}
}

// Create MPC wallet share for user
func (s *MPCWalletService) CreateWalletShare(ctx context.Context, userID uuid.UUID, deviceID, publicKey string) (*models.MPCWalletShare, error) {
	// Generate share ID and encrypted share
	shareID := uuid.New()
	shareData := generateMPCShare(userID.String(), deviceID)
	encryptedShare := encryptMPCShare(shareData)

	share := &models.MPCWalletShare{
		ID:          shareID,
		UserID:      userID,
		DeviceID:    deviceID,
		PublicKey:   publicKey,
		EncryptedShare: encryptedShare,
		Status:      "ACTIVE",
		CreatedAt:   time.Now(),
		LastUsedAt:  time.Now(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mpc_wallet_shares (id, user_id, device_id, public_key, encrypted_share, status, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, share.ID, share.UserID, share.DeviceID, share.PublicKey, share.EncryptedShare, 
		share.Status, share.CreatedAt, share.LastUsedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create wallet share: %w", err)
	}

	return share, nil
}

// Sign transaction using MPC
func (s *MPCWalletService) SignTransaction(ctx context.Context, userID uuid.UUID, txHash string) (string, error) {
	// Get all active shares for user
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, device_id, encrypted_share FROM mpc_wallet_shares 
		WHERE user_id = $1 AND status = 'ACTIVE'
	`, userID)

	if err != nil {
		return "", err
	}
	defer rows.Close()

	var shares []string
	for rows.Next() {
		var shareID, encryptedShare string
		rows.Scan(&shareID, &encryptedShare)
		shares = append(shares, encryptedShare)
	}

	if len(shares) < 2 {
		return "", fmt.Errorf("insufficient shares for signing (need at least 2)")
	}

	// In production, use threshold signature (2-of-3, 3-of-5, etc.)
	// Here we simulate the signature generation
	signature := generateMPCSignature(shares, txHash)

	// Update last used
	s.db.ExecContext(ctx, `
		UPDATE mpc_wallet_shares SET last_used_at = NOW() WHERE user_id = $1
	`, userID)

	return signature, nil
}

// Sign message using MPC
func (s *MPCWalletService) SignMessage(ctx context.Context, userID uuid.UUID, message string) (string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT encrypted_share FROM mpc_wallet_shares 
		WHERE user_id = $1 AND status = 'ACTIVE'
	`, userID)

	if err != nil {
		return "", err
	}
	defer rows.Close()

	var shares []string
	for rows.Next() {
		var share string
		rows.Scan(&share)
		shares = append(shares, share)
	}

	if len(shares) < 2 {
		return "", fmt.Errorf("insufficient shares")
	}

	return generateMPCSignature(shares, message), nil
}

// Add new device share
func (s *MPCWalletService) AddDeviceShare(ctx context.Context, userID uuid.UUID, deviceID, publicKey string) (*models.MPCWalletShare, error) {
	return s.CreateWalletShare(ctx, userID, deviceID, publicKey)
}

// Remove device share
func (s *MPCWalletService) RemoveDeviceShare(ctx context.Context, userID uuid.UUID, shareID uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE mpc_wallet_shares SET status = 'REVOKED'
		WHERE id = $1 AND user_id = $2
	`, shareID, userID)

	if err != nil {
		return fmt.Errorf("failed to remove share: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("share not found")
	}

	return nil
}

// Get user's MPC shares
func (s *MPCWalletService) GetUserShares(ctx context.Context, userID uuid.UUID) ([]models.MPCWalletShare, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, device_id, public_key, status, created_at, last_used_at
		FROM mpc_wallet_shares WHERE user_id = $1
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []models.MPCWalletShare
	for rows.Next() {
		var share models.MPCWalletShare
		err := rows.Scan(&share.ID, &share.UserID, &share.DeviceID, 
			&share.PublicKey, &share.Status, &share.CreatedAt, &share.LastUsedAt)
		if err != nil {
			continue
		}
		shares = append(shares, share)
	}

	return shares, nil
}

// Get wallet address (derived from combined public keys)
func (s *MPCWalletService) GetWalletAddress(ctx context.Context, userID uuid.UUID) (string, error) {
	var address string
	err := s.db.QueryRowContext(ctx, `
		SELECT wallet_address FROM mpc_wallets WHERE user_id = $1
	`, userID).Scan(&address)

	if err == sql.ErrNoRows {
		// Generate address from shares
		shares, err := s.GetUserShares(ctx, userID)
		if err != nil || len(shares) == 0 {
			return "", fmt.Errorf("no wallet found")
		}

		address = deriveMPCAddress(shares)
		return address, nil
	}

	return address, err
}

// Verify device ownership
func (s *MPCWalletService) VerifyDevice(ctx context.Context, userID uuid.UUID, deviceID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM mpc_wallet_shares 
		WHERE user_id = $1 AND device_id = $2 AND status = 'ACTIVE'
	`, userID, deviceID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Helper functions
func generateMPCShare(userID, deviceID string) string {
	data := userID + deviceID + fmt.Sprintf("%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func encryptMPCShare(share string) string {
	// In production, use proper encryption (AES-GCM with key from user's password)
	return fmt.Sprintf("enc_%s", share)
}

func generateMPCSignature(shares []string, data string) string {
	// In production, use proper MPC signature scheme (GG18, GG20, etc.)
	combined := ""
	for _, s := range shares {
		combined += s
	}
	hash := sha256.Sum256([]byte(combined + data))
	return hex.EncodeToString(hash[:])
}

func deriveMPCAddress(shares []models.MPCWalletShare) string {
	combined := ""
	for _, s := range shares {
		combined += s.PublicKey
	}
	hash := sha256.Sum256([]byte(combined))
	return "0x" + hex.EncodeToString(hash[:12])
}
